//go:build windows

package main

import (
	"errors"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// notifyBufferSize 是 ReadDirectoryChangesW 的输出缓冲：一次修改风暴可能把它塞满，
// 系统此时返回 ERROR_NOTIFY_ENUM_DIR，我们保守地当作“有变化”再读一次文件。
const notifyBufferSize = 16 * 1024

// notifyMask 只关心文件名/大小/写入时间的变动（目录内其余文件靠解析阶段按名字过滤）。
const notifyMask = windows.FILE_NOTIFY_CHANGE_FILE_NAME |
	windows.FILE_NOTIFY_CHANGE_SIZE |
	windows.FILE_NOTIFY_CHANGE_LAST_WRITE |
	windows.FILE_NOTIFY_CHANGE_CREATION

// x/sys/windows 没有导出 ReadDirectoryChangesW，需要自己取 kernel32 的符号。
var procReadDirectoryChangesW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReadDirectoryChangesW")

// watchStoreFile 用 ReadDirectoryChangesW 监听 dir 目录里针对 name 的改动：
// 每次（合并静默期后的）变化调用 onChange；stop 关闭后返回 nil。
// 返回非 nil 表示监听无法建立或已中断，调用方决定是否重试。
func watchStoreFile(dir, name string, stop <-chan struct{}, onChange func()) error {
	dirPtr, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	// 目录句柄必须带 FILE_FLAG_BACKUP_SEMANTICS；用 OVERLAPPED 才能被 stop 打断。
	h, err := windows.CreateFile(dirPtr, windows.FILE_LIST_DIRECTORY,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	// 自动重置事件：ReadDirectoryChangesW 完成时置位，等待返回后自动复位。
	ioEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(ioEvent)

	// 手动重置事件：stop 关闭时置位，用来打断 WaitForMultipleObjects。
	stopEvent, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(stopEvent)

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-stop:
			_ = windows.SetEvent(stopEvent)
		case <-done:
		}
	}()

	var overlapped windows.Overlapped
	overlapped.HEvent = ioEvent
	buf := make([]byte, notifyBufferSize)
	handles := []windows.Handle{ioEvent, stopEvent}

	var (
		pending     bool // 已检测到改动，正在等写入停下来
		outstanding bool // 是否已有未完成的 ReadDirectoryChangesW
	)
	for {
		if !outstanding {
			overlapped.Internal = 0
			overlapped.InternalHigh = 0
			overlapped.Offset = 0
			overlapped.OffsetHigh = 0
			var retLen uint32
			err := readDirectoryChanges(h, buf, notifyMask, &overlapped, &retLen)
			if err != nil && !errors.Is(err, windows.ERROR_IO_PENDING) {
				return err
			}
			outstanding = true
		}

		// pending 时用静默窗口超时：期间还有事件就继续等，安静下来才读文件。
		timeout := uint32(windows.INFINITE)
		if pending {
			timeout = uint32(storeWatchQuiet / time.Millisecond)
		}
		ev, err := windows.WaitForMultipleObjects(handles, false, timeout)
		if err != nil {
			return err
		}
		switch {
		case ev == uint32(windows.WAIT_TIMEOUT):
			pending = false
			onChange()
		case ev == 1: // stopEvent：取消挂起的 I/O 后退出
			// 等被取消的 I/O 真正结束再关句柄，避免事件对象被关闭后才收到信号
			// （句柄值那时可能已被系统复用）。
			_ = windows.CancelIoEx(h, &overlapped)
			_, _ = windows.WaitForSingleObject(ioEvent, 1000)
			return nil
		default: // ioEvent：本次读取完成
			var doneLen uint32
			gerr := windows.GetOverlappedResult(h, &overlapped, &doneLen, false)
			outstanding = false
			switch {
			case gerr == nil:
				if storeFileTouched(buf[:doneLen], name) {
					pending = true
				}
			case errors.Is(gerr, windows.ERROR_OPERATION_ABORTED):
				return nil // 已停止
			case errors.Is(gerr, windows.ERROR_NOTIFY_ENUM_DIR):
				pending = true // 缓冲溢出：丢了一批事件，保守地当作有变化
			default:
				return gerr // 交给调用方重试，避免在这里空转
			}
		}
	}
}

// readDirectoryChanges 发起一次异步 ReadDirectoryChangesW（完成时 overlapped 的
// 事件对象置位）。异步发起时返回 ERROR_IO_PENDING。
func readDirectoryChanges(h windows.Handle, buf []byte, mask uint32, overlapped *windows.Overlapped, retLen *uint32) error {
	r1, _, err := procReadDirectoryChangesW.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0, // watchSubTree=false：store 文件就在数据目录里
		uintptr(mask),
		uintptr(unsafe.Pointer(retLen)),
		uintptr(unsafe.Pointer(overlapped)),
		0,
	)
	if r1 == 0 {
		return err
	}
	return nil
}

// storeFileTouched 报告一批 FILE_NOTIFY_INFORMATION 记录里是否提到了目标文件名。
func storeFileTouched(buf []byte, name string) bool {
	const headerSize = 12 // NextEntryOffset + Action + FileNameLength
	for offset := 0; offset+headerSize <= len(buf); {
		fni := (*windows.FileNotifyInformation)(unsafe.Pointer(&buf[offset]))
		n := int(fni.FileNameLength) / 2
		if n > 0 && offset+headerSize+int(fni.FileNameLength) <= len(buf) &&
			strings.EqualFold(windows.UTF16ToString(unsafe.Slice(&fni.FileName, n)), name) {
			return true
		}
		next := int(fni.NextEntryOffset)
		if next == 0 {
			return false
		}
		offset += next
	}
	return false
}
