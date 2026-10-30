package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:     "texxer",
		Width:     1440,
		Height:    900,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Middleware: app.middleware,
		},
		BackgroundColour: &options.RGBA{R: 20, G: 20, B: 19, A: 255},
		StartHidden:      true,
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		Bind:             []any{app},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: false,
			About:                &mac.AboutInfo{Title: "texxer", Message: "Retro model painter"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
