package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	service := &App{}
	app := application.New(application.Options{Name: "CLI Bridger", Services: []application.Service{application.NewService(service)},
		Windows: application.WindowsOptions{WebviewUserDataPath: os.Getenv("CLI_BRIDGER_WEBVIEW_DATA")},
		Assets:  application.AssetOptions{Handler: application.BundledAssetFileServer(assets)}, OnShutdown: func() { _ = service.Stop() }})
	service.desktop = app
	app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "CLI Bridger", Width: 1200, Height: 900, MinWidth: 800, MinHeight: 600, URL: "/"})
	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
