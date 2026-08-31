package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	app := NewApp()

	assetsFS, err := fs.Sub(assets, "frontend")
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:  "Go Launcher",
		Width:  640,
		Height: 400,
		AssetServer: &assetserver.Options{
			Assets: assetsFS,
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
