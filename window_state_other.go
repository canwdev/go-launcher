//go:build !windows

package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// currentWindowState reads the window geometry through the Wails runtime. The
// Wails runtime reports the maximised bounds while the window is maximised, so
// on non-Windows a session that ends maximised persists those bounds as the
// restore size; Windows gets an exact result from GetWindowPlacement instead.
func currentWindowState(ctx context.Context) windowState {
	return runtimeWindowState(ctx)
}

// applyRestoredWindowState repositions the window after startup on platforms
// where Wails does not centre the window by default. Size comes from the
// startup options and the maximised state from WindowStartState.
func applyRestoredWindowState(ctx context.Context) {
	ws := loadWindowState()
	if ctx == nil || ws.Width <= 0 || ws.Height <= 0 {
		return
	}
	if ws.Maximised {
		runtime.WindowMaximise(ctx)
	}
	runtime.WindowSetPosition(ctx, ws.X, ws.Y)
}
