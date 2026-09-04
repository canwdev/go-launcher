//go:build windows

package main

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// singleInstanceMutexPrefix is the name prefix (session-local namespace) for the
// per-install-directory named mutex. The full name is this prefix plus the
// install-directory hash, so different install directories get different
// mutexes and can run concurrently, while a second launch from the same
// directory is refused.
//
// The "Local\" prefix explicitly scopes the mutex to the current terminal
// session: it needs no SeCreateGlobalPrivilege (unlike "Global\") and lets two
// different user sessions run their own copies of the same installation.
const singleInstanceMutexPrefix = `Local\go-launcher-singleton-`

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

// acquireSingleton claims the per-install-directory named mutex. It returns true
// when this process is the only running copy for its install directory, and
// false when another copy for the same directory already owns the mutex (which
// must therefore be refused). Mutexes for different install directories do not
// conflict, so those instances may run simultaneously.
func acquireSingleton(key string) (bool, error) {
	name, err := windows.UTF16PtrFromString(singleInstanceMutexPrefix + key)
	if err != nil {
		return false, err
	}
	h, err := windows.CreateMutex(nil, true, name)
	if err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			// Another instance for the same directory owns the mutex; close our
			// handle to it.
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
// for the same install directory and brings it to the foreground, so a second
// launch focuses the existing app instead of popping a dialog. The process then
// exits without starting a new UI.
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

// findMainWindow returns the HWND of the running copy's main window for this
// install directory.
//
// The window title is the full install directory (see instanceTitle), so with
// several instances running concurrently the lookup targets exactly this
// directory's window: full paths differ between install directories, and
// FindWindowW's title comparison is case-insensitive, covering launches with
// different path casing. The old class-based lookup is intentionally dropped:
// all instances share the Wails window class, so it cannot distinguish install
// directories.
func findMainWindow() uintptr {
	title, err := windows.UTF16PtrFromString(instanceTitle(installDir()))
	if err != nil {
		return 0
	}
	h, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	return h
}
