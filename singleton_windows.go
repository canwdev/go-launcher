//go:build windows

package main

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// singleInstanceMutexName is a fixed, globally known name used to detect
// whether another copy of this application is already running. It must stay
// stable across builds so different copies agree on the same object.
const singleInstanceMutexName = "go-launcher-singleton"

// wailsWindowClass is the Wails v2 main-window window class on Windows. It is
// used to locate the already-running copy's window so a second launch can focus
// it instead of popping a dialog.
const wailsWindowClass = "wailsWindow"

// wailsWindowTitle matches options.App.Title in main.go; used as a fallback
// when locating the running window.
const wailsWindowTitle = "Go Launcher"

// Win32 constants for window management (user32.h / winuser.h).
const (
	swRestore     = 9 // SW_RESTORE
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpShowWindow = 0x0040
)

// HWND_TOPMOST (-1) and HWND_NOTOPMOST (-2) as uintptr values (two's
// complement), since negative untyped constants cannot convert to uintptr.
var (
	hwndTopmost    = ^uintptr(0)     // HWND_TOPMOST
	hwndNotTopmost = ^uintptr(0) - 1 // HWND_NOTOPMOST
)

// instanceMutex is the handle to the named mutex held for the whole lifetime
// of the process. Keeping the handle open is what marks this instance as the
// running one; the OS releases ownership automatically when the process exits
// (even after a crash), so a dead instance never leaves a stale lock behind.
var instanceMutex windows.Handle

var (
	user32Singleton   = syscall.NewLazyDLL("user32.dll")
	procFindWindowW   = user32Singleton.NewProc("FindWindowW")
	procShowWindow    = user32Singleton.NewProc("ShowWindow")
	procSetForeground = user32Singleton.NewProc("SetForegroundWindow")
	procSetWindowPos  = user32Singleton.NewProc("SetWindowPos")
)

// acquireSingleton claims the single-instance named mutex. It returns true
// when this process is the only running copy, and false when another copy
// already owns the mutex (which must therefore be refused).
func acquireSingleton() (bool, error) {
	name, err := windows.UTF16PtrFromString(singleInstanceMutexName)
	if err != nil {
		return false, err
	}
	h, err := windows.CreateMutex(nil, true, name)
	if err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			// Another instance owns the mutex; close our handle to it.
			if h != 0 {
				_ = windows.CloseHandle(h)
			}
			return false, nil
		}
		return false, err
	}
	instanceMutex = h
	return true, nil
}

// releaseSingleton closes the local handle at shutdown. The named mutex
// object itself is destroyed by the OS once the last handle is closed.
func releaseSingleton() {
	if instanceMutex != 0 {
		_ = windows.CloseHandle(instanceMutex)
		instanceMutex = 0
	}
}

// activateExistingInstance locates the main window of the already running copy
// and brings it to the foreground, so a second launch focuses the existing app
// instead of popping a dialog. The process then exits without starting a new
// UI.
func activateExistingInstance() {
	hwnd := findMainWindow()
	if hwnd == 0 {
		return
	}
	// Restore the window if it was minimized, then ask Windows to focus it.
	procShowWindow.Call(hwnd, swRestore)
	procSetForeground.Call(hwnd)
	// SetForegroundWindow can be denied to a process that is not the current
	// foreground process; the classic topmost flicker forces the window above
	// the z-order regardless of that restriction.
	procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, uintptr(swpNoMove|swpNoSize|swpShowWindow))
	procSetWindowPos.Call(hwnd, hwndNotTopmost, 0, 0, 0, 0, uintptr(swpNoMove|swpNoSize))
}

// findMainWindow returns the HWND of the running copy's main window. It first
// looks it up by the Wails window class, then falls back to the window title.
func findMainWindow() uintptr {
	cls, err := windows.UTF16PtrFromString(wailsWindowClass)
	if err == nil {
		h, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(cls)), 0)
		if h != 0 {
			return h
		}
	}
	title, err := windows.UTF16PtrFromString(wailsWindowTitle)
	if err == nil {
		h, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
		if h != 0 {
			return h
		}
	}
	return 0
}
