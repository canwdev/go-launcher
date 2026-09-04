package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Single-instance guard, scoped per install directory: a second copy from
	// the same directory is refused (and focuses the existing window), while
	// copies in different install directories may run simultaneously. Must run
	// before wails.Run initialises the UI/backend.
	key := installKey()
	if ok, err := acquireSingleton(key); err != nil {
		log.Fatalf("single-instance check failed: %v", err)
	} else if !ok {
		activateExistingInstance()
		return
	}
	defer releaseSingleton()

	// Reopen the window at the size (and maximised state) it had when last
	// closed (persisted by beforeClose); the first run falls back to the
	// defaults. The saved position is applied separately in startup, because
	// Wails always centres the window on creation.
	ws := loadWindowState()
	width, height := defaultWindowWidth, defaultWindowHeight
	if ws.Width > 0 && ws.Height > 0 {
		width, height = ws.Width, ws.Height
	}
	startState := options.Normal
	if ws.Maximised {
		startState = options.Maximised
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:            instanceTitle(installDir()),
		Width:            width,
		Height:           height,
		WindowStartState: startState,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		Bind: []interface{}{
			app,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
