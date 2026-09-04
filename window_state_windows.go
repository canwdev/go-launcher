//go:build windows

package main

import (
	"context"
	"syscall"
	"time"
	"unsafe"
)

const swShowMaximized = 3 // SW_SHOWMAXIMIZED

// SetWindowPos flags used when restoring the window position.
const (
	swpNoZOrder   = 0x0004 // SWP_NOZORDER
	swpNoActivate = 0x0010 // SWP_NOACTIVATE
)

// placementPoint / placementRect mirror the Win32 POINT / RECT used inside
// WINDOWPLACEMENT (all fields are 32-bit LONGs, so the Go structs match the C
// layout field-for-field).
type placementPoint struct{ X, Y int32 }

type placementRect struct{ Left, Top, Right, Bottom int32 }

// windowPlacement mirrors tagWINDOWPLACEMENT (winuser.h). rcNormalPosition
// holds the bounds the window takes when restored, which Windows keeps valid
// even while the window is maximised or minimised — exactly the stable size we
// want to persist regardless of the window's current visual state.
type windowPlacement struct {
	Length      uint32
	Flags       uint32
	ShowCmd     uint32
	MinPosition placementPoint
	MaxPosition placementPoint
	NormalRect  placementRect
}

var (
	user32WindowState      = syscall.NewLazyDLL("user32.dll")
	procGetWindowPlacement = user32WindowState.NewProc("GetWindowPlacement")
	procSetWindowPlacement = user32WindowState.NewProc("SetWindowPlacement")
	procIsWindowVisible    = user32WindowState.NewProc("IsWindowVisible")
	procMonitorFromPoint   = user32WindowState.NewProc("MonitorFromPoint")
	procGetDpiForWindow    = user32WindowState.NewProc("GetDpiForWindow")
)

// currentWindowState returns the geometry to persist for the launcher window.
// The window is located by its unique install-directory title (see
// findMainWindow); GetWindowPlacement yields the restored-state bounds even
// when the window is currently maximised, which the Wails runtime cannot.
func currentWindowState(ctx context.Context) windowState {
	hwnd := findMainWindow()
	if hwnd != 0 {
		if ws, ok := placementState(hwnd); ok {
			return ws
		}
	}
	return runtimeWindowState(ctx)
}

func placementState(hwnd uintptr) (windowState, bool) {
	var wp windowPlacement
	wp.Length = uint32(unsafe.Sizeof(wp))
	ret, _, _ := procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&wp)))
	if ret == 0 {
		return windowState{}, false
	}
	w := int(wp.NormalRect.Right - wp.NormalRect.Left)
	h := int(wp.NormalRect.Bottom - wp.NormalRect.Top)
	if w <= 0 || h <= 0 {
		return windowState{}, false
	}
	// rcNormalPosition is in physical pixels; Wails sizes windows in 96-DPI
	// logical units, so convert back the same way winc.Size() does. X/Y stays
	// in physical pixels and is restored with SetWindowPos as-is.
	dpi := windowDPI(hwnd)
	return windowState{
		Width:     toDefaultDPI(w, dpi),
		Height:    toDefaultDPI(h, dpi),
		X:         int(wp.NormalRect.Left),
		Y:         int(wp.NormalRect.Top),
		Maximised: wp.ShowCmd == swShowMaximized,
	}, true
}

// applyRestoredWindowState places the window back at the persisted position
// once it exists (OnStartup). Size and maximised state are applied through the
// Wails startup options; this only fixes the position, because Wails centres
// every window at creation and offers no start-position option.
func applyRestoredWindowState(ctx context.Context) {
	ws := loadWindowState()
	if ctx == nil || ws.Width <= 0 || ws.Height <= 0 {
		return
	}
	hwnd := findMainWindow()
	if hwnd == 0 {
		return
	}
	if ws.Maximised {
		// Keep the maximised start, but pin the restored-state bounds so that
		// unmaximising later returns to the persisted position/size instead of
		// the centre Wails chose. SetWindowPlacement would show the window, so
		// defer it until Wails has shown it (avoids a blank flash before
		// WebView2 is ready).
		if isWindowVisible(hwnd) {
			pinRestoreRect(hwnd, ws)
		} else {
			go pinRestoreWhenVisible(hwnd, ws)
		}
		return
	}
	// Move before the window is shown so it appears directly at the saved spot,
	// then repeat once shown: the pre-show call can be a no-op while the window
	// thread is not pumping messages yet.
	moveWindowTo(hwnd, ws)
	if !isWindowVisible(hwnd) {
		go moveWhenVisible(hwnd, ws)
	}
}

func moveWindowTo(hwnd uintptr, ws windowState) {
	if !pointOnScreen(ws.X, ws.Y) {
		return
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(ws.X), uintptr(ws.Y), 0, 0, uintptr(swpNoSize|swpNoZOrder|swpNoActivate))
}

// moveWhenVisible retries moveWindowTo once the window has been shown, giving
// up after a few seconds if the window never appears.
func moveWhenVisible(hwnd uintptr, ws windowState) {
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if isWindowVisible(hwnd) {
			moveWindowTo(hwnd, ws)
			return
		}
	}
}

// pinRestoreWhenVisible retries pinRestoreRect once the window has been shown,
// giving up after a few seconds if the window never appears.
func pinRestoreWhenVisible(hwnd uintptr, ws windowState) {
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if isWindowVisible(hwnd) {
			pinRestoreRect(hwnd, ws)
			return
		}
	}
}

func isWindowVisible(hwnd uintptr) bool {
	ret, _, _ := procIsWindowVisible.Call(hwnd)
	return ret != 0
}

// pinRestoreRect writes the window's restored-state bounds (kept by Windows
// while the window is maximised) without changing the current maximised state.
func pinRestoreRect(hwnd uintptr, ws windowState) {
	dpi := windowDPI(hwnd)
	w := ws.Width * int(dpi) / 96
	h := ws.Height * int(dpi) / 96
	if w <= 0 || h <= 0 {
		return
	}
	var wp windowPlacement
	wp.Length = uint32(unsafe.Sizeof(wp))
	wp.ShowCmd = swShowMaximized
	wp.NormalRect = placementRect{Left: int32(ws.X), Top: int32(ws.Y), Right: int32(ws.X + w), Bottom: int32(ws.Y + h)}
	procSetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&wp)))
}

// pointOnScreen reports whether the given absolute screen coordinate lies on a
// connected monitor, guarding against restoring to a display that is no longer
// present (which would leave the window unreachable).
func pointOnScreen(x, y int) bool {
	// POINT is an 8-byte value passed by value (not a pointer): pack the two
	// int32 fields into one 64-bit register-sized argument.
	pt := uint64(uint32(int32(y)))<<32 | uint64(uint32(int32(x)))
	h, _, _ := procMonitorFromPoint.Call(uintptr(pt), 0)
	return h != 0
}

func windowDPI(hwnd uintptr) uint {
	if err := procGetDpiForWindow.Find(); err != nil {
		return 96
	}
	dpi, _, _ := procGetDpiForWindow.Call(hwnd)
	if dpi == 0 {
		return 96
	}
	return uint(dpi)
}

func toDefaultDPI(px int, dpi uint) int {
	if dpi == 0 {
		return px
	}
	return px * 96 / int(dpi)
}
