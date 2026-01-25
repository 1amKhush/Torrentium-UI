package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create system tray menu
	trayMenu := menu.NewMenu()
	trayMenu.Append(menu.Text("Show Torrentium", keys.CmdOrCtrl("s"), func(cd *menu.CallbackData) {
		runtime.WindowShow(app.ctx)
	}))
	trayMenu.Append(menu.Separator())
	trayMenu.Append(menu.Text("Downloads", nil, func(cd *menu.CallbackData) {
		runtime.WindowShow(app.ctx)
		runtime.EventsEmit(app.ctx, "navigate", "downloads")
	}))
	trayMenu.Append(menu.Text("Files", nil, func(cd *menu.CallbackData) {
		runtime.WindowShow(app.ctx)
		runtime.EventsEmit(app.ctx, "navigate", "files")
	}))
	trayMenu.Append(menu.Text("Stats", nil, func(cd *menu.CallbackData) {
		runtime.WindowShow(app.ctx)
		runtime.EventsEmit(app.ctx, "navigate", "stats")
	}))
	trayMenu.Append(menu.Separator())
	trayMenu.Append(menu.Text("Quit", keys.CmdOrCtrl("q"), func(cd *menu.CallbackData) {
		runtime.Quit(app.ctx)
	}))

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Torrentium",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 18, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// Minimize to tray instead of closing
			runtime.WindowMinimise(ctx)
			return true // Prevent the window from closing
		},
		Bind: []interface{}{
			app,
		},
		// Enable drag and drop for file sharing
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			Theme:                windows.Dark,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
