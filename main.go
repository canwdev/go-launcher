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
	// Single-instance guard: refuse to start a second copy of the launcher.
	// Must run before wails.Run initialises the UI/backend.
	if ok, err := acquireSingleton(); err != nil {
		log.Fatalf("single-instance check failed: %v", err)
	} else if !ok {
		activateExistingInstance()
		return
	}
	defer releaseSingleton()

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Go Launcher",
		Width:  640,
		Height: 400,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
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
