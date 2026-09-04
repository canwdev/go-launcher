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

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  instanceTitle(installDir()),
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
