package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"zenith/internal/config"
)

//go:embed all:frontend/dist
var assets embed.FS

// version is the release tag, set at build time:
//
//	wails build -ldflags "-X main.version=v1.2.3"
//
// "dev" marks a development build (no update is offered).
var version = "dev"

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Restore the window size of the last session on this computer.
	win := config.DefaultWindow()
	if dir, err := config.Dir(); err == nil {
		win = config.LoadWindow(dir)
	}
	startState := options.Normal
	if win.Maximised {
		startState = options.Maximised
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "Zenith",
		Width:            win.Width,
		Height:           win.Height,
		MinWidth:         config.MinWindowWidth,
		MinHeight:        config.MinWindowHeight,
		WindowStartState: startState,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 246, G: 247, B: 249, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
