package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Default window geometry used on the very first launch, before any state has
// been persisted.
const (
	defaultWindowWidth  = 640
	defaultWindowHeight = 400
)

const windowStateFile = dataDir + "/window-state.json"

// windowState is the window geometry persisted across restarts: the bounds the
// window takes in its normal (restored) state plus whether it should start
// maximised.
//
// Width/Height are stored in the same 96-DPI logical units that Wails' options
// and WindowGetSize use, so they round-trip through the startup options without
// DPI drift. X/Y is the top-left of the normal-state window: on Windows it is
// stored as absolute physical screen pixels (as returned by
// GetWindowPlacement) and restored with SetWindowPos; other platforms keep the
// raw values of the Wails WindowGetPosition/WindowSetPosition pair.
type windowState struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Maximised bool `json:"maximised"`
}

func loadWindowState() windowState {
	data, err := os.ReadFile(windowStateFile)
	if err != nil {
		return windowState{}
	}
	var ws windowState
	if err := json.Unmarshal(data, &ws); err != nil {
		return windowState{}
	}
	return ws
}

func saveWindowState(ws windowState) {
	if ws.Width <= 0 || ws.Height <= 0 {
		return
	}
	_ = os.MkdirAll(dataDir, 0755)
	data, err := json.Marshal(ws)
	if err != nil {
		return
	}
	tmp := windowStateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	_ = os.Rename(tmp, windowStateFile)
}

// persistWindowState snapshots the live window geometry and writes it to disk.
// It runs just before the window closes (OnBeforeClose), while the window still
// exists, so the next launch can restore the same size/state.
func persistWindowState(ctx context.Context) {
	saveWindowState(currentWindowState(ctx))
}

// runtimeWindowState queries the geometry through the Wails runtime. It reports
// the maximised bounds while the window is maximised, so it is only used as a
// fallback on Windows (the GetWindowPlacement-based reader in
// window_state_windows.go is authoritative there) and as the primary reader on
// other platforms.
func runtimeWindowState(ctx context.Context) windowState {
	if ctx == nil {
		return windowState{}
	}
	w, h := runtime.WindowGetSize(ctx)
	if w <= 0 || h <= 0 {
		return windowState{}
	}
	x, y := runtime.WindowGetPosition(ctx)
	return windowState{
		Width:     w,
		Height:    h,
		X:         x,
		Y:         y,
		Maximised: runtime.WindowIsMaximised(ctx),
	}
}
