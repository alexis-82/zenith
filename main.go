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
	theme := config.Defaults().Theme
	if dir, err := config.Dir(); err == nil {
		win = config.LoadWindow(dir)
		theme = config.Load(dir).Theme
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
			Assets:     assets,
			Middleware: app.themedIndex,
		},
		BackgroundColour: windowBackground(theme),
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
