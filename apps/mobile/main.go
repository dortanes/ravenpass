//go:build android

package main

import (
	"embed"
	"log"

	"github.com/dortanes/ravenpass/apps/mobile/internal/bridge"
	"github.com/dortanes/ravenpass/apps/mobile/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

// A C shared library never runs main: the Wails Java host calls it on each activity creation.
func init() {
	application.RegisterAndroidMain(main)
	bridge.ServeAutofill(core.Autofill)
	bridge.ReceiveCodeSetups(core.ReceiveCodeSetup)
}

func main() {
	c, err := core.Get()
	if err != nil {
		log.Fatal(err)
	}
	app := application.New(application.Options{
		Name:        "Ravenpass",
		Description: "Local encrypted password vault",
		Services:    []application.Service{application.NewService(c.Service)},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})
	// The Android runtime routes every runtime call to the first window.
	c.Attach(app, app.Window.New())
	// Wails v3.0.0-beta.23 declares events.Android.ActivityDestroyed but never emits it.
	bridge.OnInterfaceDestroyed(c.Detach)
	app.Event.OnApplicationEvent(events.Android.ActivityStopped, func(*application.ApplicationEvent) { c.Hidden() })
	app.Event.OnApplicationEvent(events.Android.ActivityStarted, func(*application.ApplicationEvent) { c.Shown() })
	app.Event.OnApplicationEvent(events.Common.ScreenLocked, func(*application.ApplicationEvent) { c.Lock() })
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
