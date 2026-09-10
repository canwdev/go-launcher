//go:build windows

package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
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

func shellExecuteEx(verb, path, params, dir string) (windows.Handle, error) {
	verbPtr, err := windows.UTF16PtrFromString(verb)
	if err != nil {
		return 0, err
	}
	filePtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var paramsPtr, dirPtr *uint16
	if params != "" {
		paramsPtr, err = windows.UTF16PtrFromString(params)
		if err != nil {
			return 0, err
		}
	}
	if dir != "" {
		dirPtr, err = windows.UTF16PtrFromString(dir)
		if err != nil {
			return 0, err
		}
	}
	info := &shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess,
		lpVerb:       verbPtr,
		lpFile:       filePtr,
		lpParameters: paramsPtr,
		lpDirectory:  dirPtr,
		nShow:        swShowNormal,
	}
	info.cbSize = uint32(unsafe.Sizeof(*info))
	r1, _, callErr := shellExecuteExProc.Call(uintptr(unsafe.Pointer(info)))
	if r1 == 0 {
		return 0, callErr
	}
	return windows.Handle(info.hProcess), nil
}

func openWithDefaultHandler(path string) error {
	// 通过 explorer.exe（Windows Shell）中转打开：打开请求会被转发给已运行的 shell
	// 实例，由它在正常交互用户上下文里拉起目标程序（如 ImageGlass），彻底脱钩本进程
	// 的 token/提权/环境/WebView2 上下文，避免 WebView2 初始化报 E_ACCESSDENIED。
	// explorer.exe 仅接一个文件路径参数；普通路径不受其开关语义影响。
	return exec.Command("explorer.exe", path).Start()
}

func revealFile(path string) error {
	return openWithDefaultHandler(filepath.Dir(path))
}

func joinArgsForShell(args []string) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if strings.ContainsAny(a, " \t") {
			parts = append(parts, `"`+a+`"`)
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}

// launchVerb 返回 ShellExecuteEx 使用的动词：item 勾选“Run as administrator”时
// 固定 "runas"（系统弹 UAC 提权），否则用默认的 "open"（仅在程序自身要求提权时
// 才回退到 runas，见 startProcess）。
func launchVerb(elevated bool) string {
	if elevated {
		return "runas"
	}
	return "open"
}

// startProcess launches path with args in workDir via ShellExecuteEx and
// returns the process handle; the caller decides how to wait/cleanup.
//
// elevated=true（item 勾选了“Run as administrator”）时固定用 "runas" 动词，由系统
// 弹 UAC 提权；否则先用 "open"，只有在程序自身要求提权等情况下才回退到 "runas"。
func startProcess(path string, args []string, workDir string, elevated bool) (windows.Handle, error) {
	if elevated {
		h, err := shellExecuteEx(launchVerb(true), path, joinArgsForShell(args), workDir)
		if err != nil {
			// 1223 = ERROR_CANCELLED：用户在 UAC 弹窗点了“否”
			if errors.Is(err, syscall.Errno(1223)) {
				return 0, errors.New("elevation was cancelled")
			}
			return 0, err
		}
		return h, nil
	}
	h, err := shellExecuteEx(launchVerb(false), path, joinArgsForShell(args), workDir)
	if err != nil {
		if errors.Is(err, syscall.Errno(1223)) || errors.Is(err, syscall.Errno(5)) {
			return 0, err
		}
		h, err = shellExecuteEx("runas", path, joinArgsForShell(args), workDir)
		if err != nil {
			return 0, err
		}
	}
	return h, nil
}

// startUntracked launches path without returning a process handle, so nothing
// can Stop or time it. The shell-executed process keeps running after the
// handle is released.
func startUntracked(path string, args []string, workDir string, elevated bool) error {
	h, err := startProcess(path, args, workDir, elevated)
	if err != nil {
		return err
	}
	_ = windows.CloseHandle(h)
	return nil
}

func startTracked(path string, args []string, workDir string, elevated bool, proc *runningProc) error {
	h, err := startProcess(path, args, workDir, elevated)
	if err != nil {
		return err
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
