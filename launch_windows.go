//go:build windows

package main

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	seeMaskNoCloseProcess = 0x00000040
	swShowNormal          = 1
)

type shellExecuteInfo struct {
	cbSize         uint32
	fMask          uint32
	hwnd           uintptr
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       uintptr
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      uintptr
	dwHotKey       uint32
	hIconOrMonitor uintptr
	hProcess       uintptr
}

var shellExecuteExProc = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteExW")

func shellExecuteEx(verb, path string) (windows.Handle, error) {
	verbPtr, err := windows.UTF16PtrFromString(verb)
	if err != nil {
		return 0, err
	}
	filePtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	info := &shellExecuteInfo{
		fMask:  seeMaskNoCloseProcess,
		lpVerb: verbPtr,
		lpFile: filePtr,
		nShow:  swShowNormal,
	}
	info.cbSize = uint32(unsafe.Sizeof(*info))
	r1, _, callErr := shellExecuteExProc.Call(uintptr(unsafe.Pointer(info)))
	if r1 == 0 {
		return 0, callErr
	}
	return windows.Handle(info.hProcess), nil
}

func openWithDefaultHandler(path string) error {
	h, err := shellExecuteEx("open", path)
	if h != 0 {
		_ = windows.CloseHandle(h)
	}
	return err
}

func revealFile(path string) error {
	return exec.Command("explorer", "/select,"+path).Start()
}

func startTracked(path string, proc *runningProc) error {
	h, err := shellExecuteEx("open", path)
	if err != nil {
		if errors.Is(err, syscall.Errno(1223)) || errors.Is(err, syscall.Errno(5)) {
			return err
		}
		h, err = shellExecuteEx("runas", path)
		if err != nil {
			return err
		}
	}
	proc.start = time.Now()
	proc.wait = func() error {
		_, _ = windows.WaitForSingleObject(h, windows.INFINITE)
		return nil
	}
	proc.stop = func() error { return windows.TerminateProcess(h, 1) }
	proc.cleanup = func() { _ = windows.CloseHandle(h) }
	return nil
}
