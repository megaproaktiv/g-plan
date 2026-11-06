package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

// Build information, set via -ldflags at release time (goreleaser).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	log.Printf("pt-plan %s (commit %s, built %s)", version, commit, date)
	appState := NewApp()

	app := application.New(application.Options{
		Name:        "pt-plan",
		Description: "Local graphical project-plan editor (" + version + ")",
		Services: []application.Service{
			application.NewService(appState),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	appState.app = app

	// Autosave on exit: when the app shuts down, save the project if it is dirty
	// and the project's autosave-on-exit flag is on (default). SaveOnExit is a
	// pure Go-side write (known path or Downloads fallback), so it is safe to
	// run during shutdown without touching the webview.
	app.OnShutdown(func() { appState.SaveOnExit() })

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "pt-plan",
		Width:            1280,
		Height:           820,
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
	})
	// Also save when the window is closing (covers closing the last window even
	// if that path differs from app shutdown on some platforms).
	win.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		appState.SaveOnExit()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
