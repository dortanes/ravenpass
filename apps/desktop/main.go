package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/dortanes/ravenpass/apps/desktop/internal/appearance"
	"github.com/dortanes/ravenpass/apps/desktop/internal/autofillbridge"
	"github.com/dortanes/ravenpass/apps/desktop/internal/autolock"
	"github.com/dortanes/ravenpass/apps/desktop/internal/clouddrive"
	"github.com/dortanes/ravenpass/apps/desktop/internal/confirmationpanel"
	"github.com/dortanes/ravenpass/apps/desktop/internal/dock"
	"github.com/dortanes/ravenpass/apps/desktop/internal/filewatch"
	"github.com/dortanes/ravenpass/apps/desktop/internal/identitystore"
	"github.com/dortanes/ravenpass/apps/desktop/internal/mainwindow"
	"github.com/dortanes/ravenpass/apps/desktop/internal/menubar"
	"github.com/dortanes/ravenpass/apps/desktop/internal/pagezoom"
	"github.com/dortanes/ravenpass/apps/desktop/internal/printing"
	"github.com/dortanes/ravenpass/apps/desktop/internal/qrcodes"
	"github.com/dortanes/ravenpass/apps/desktop/internal/secureenclave"
	"github.com/dortanes/ravenpass/apps/desktop/internal/systemlanguages"
	"github.com/dortanes/ravenpass/packages/app/api"
	"github.com/dortanes/ravenpass/packages/app/appcore"
	"github.com/dortanes/ravenpass/packages/app/backupexclusion"
	"github.com/dortanes/ravenpass/packages/app/backups"
	"github.com/dortanes/ravenpass/packages/app/extensionaccess"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/linkstore"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/trayicon.png
var trayIcon []byte

func main() {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	applicationDirectory := filepath.Join(configDirectory, "Ravenpass")
	core, err := appcore.Build(appcore.Platform{
		Directory:         applicationDirectory,
		Languages:         systemlanguages.Preferred,
		PIN:               secureenclave.Enclave{},
		Presence:          secureenclave.PresenceEnclave{},
		BackupDestination: backups.Directory{},
	})
	if err != nil {
		log.Fatal(err)
	}
	vault, settings, confirmations := core.Vault, core.Settings, core.Confirmations
	extensions, err := linkstore.New(filepath.Join(applicationDirectory, "extensions.json"), backupexclusion.System{})
	if err != nil {
		log.Fatal(err)
	}
	access, err := extensionaccess.New(vault, vault, core.Icons, core.Verifier, settings, confirmations)
	if err != nil {
		log.Fatal(err)
	}
	linkSettings, err := extensionaccess.NewSettings(settings)
	if err != nil {
		log.Fatal(err)
	}
	links, err := linkserver.New(extensions, access, linkSettings)
	if err != nil {
		log.Fatal(err)
	}
	identities, err := identitystore.New(vault, core.Autofill, settings.IdentityList, identitystore.Mac{})
	if err != nil {
		log.Fatal(err)
	}
	var app *application.App
	currentApp := func() *application.App { return app }
	panel, err := confirmationpanel.New(confirmationpanel.NewDisplay(currentApp))
	if err != nil {
		log.Fatal(err)
	}
	var showWindow, reloadWindow func()
	service, err := core.Serve(api.Host{
		Links:        links,
		IdentityList: identities,
		Confirmations: api.Confirmations{
			Panel:      panel,
			ShowMain:   func() { showWindow() },
			ReloadMain: func() { reloadWindow() },
		},
		CurrentApp: currentApp,
		Places:     clouddrive.Places{},
		Printer:    printing.Mac{},
		QRCodes:    qrcodes.Mac{},
		Offers:     api.Capabilities{Shortcuts: true, DockIcon: true, StorageLocations: true, SaveFiles: true, CopyScans: true, Appearance: true},
	})
	if err != nil {
		log.Fatal(err)
	}
	controls := api.ControlsOf(service)
	provider, err := autofillbridge.New(core.Autofill, service, confirmations, core.Verifier, autofillbridge.Mac{})
	if err != nil {
		log.Fatal(err)
	}
	watching, stopWatching := context.WithCancel(context.Background())
	app = application.New(application.Options{
		Name:        "Ravenpass",
		Description: "Local encrypted password vault",
		Icon:        appIcon,
		Services:    []application.Service{application.NewService(service)},
		OnShutdown: func() {
			stopWatching()
			warnIf("locking the vault at shutdown failed", service.Lock())
			warnIf("closing the browser extension server failed", links.Close())
			warnIf("closing the AutoFill bridge failed", provider.Close())
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})
	appearance.Follow(settings)
	menu := application.DefaultApplicationMenu()
	pagezoom.Bind(menu, app.Window.Current)
	app.Menu.Set(menu)

	// Closing the window drops the editor's pictures, the scans and the import in progress.
	window := mainwindow.New(app, application.WebviewWindowOptions{
		Title:     "Ravenpass",
		Width:     1200,
		Height:    800,
		MinWidth:  760,
		MinHeight: 540,
		URL:       "/",
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
	}, func() {
		warnIf("discarding the scans failed", service.DiscardScans())
		warnIf("discarding the identity photo failed", service.DiscardIdentityPhoto())
		warnIf("canceling the import failed", service.CancelImport())
		if settings.DockHiddenWithWindow() {
			dock.Hide()
		}
	})
	lock := func() {
		warnIf("locking the vault failed", service.Lock())
		window.Reload()
	}
	lockOpenVault := func() {
		if vault.Unlocked() {
			lock()
		}
	}
	followVaultFile := func() {
		if controls.FollowVaultFile() {
			window.Reload()
		}
	}
	showWindow = func() {
		dock.Show()
		app.SetIcon(appIcon)
		window.Show()
	}
	reloadWindow = window.Reload
	if err := links.Start(); err != nil {
		slog.Warn("the extension link server could not start", "err", err)
	}
	if err := provider.Start(); err != nil && !errors.Is(err, autofillbridge.ErrUnsigned) {
		slog.Warn("the AutoFill bridge could not start", "err", err)
	}
	// AppKit resets an unbundled executable's Dock icon and hides any window shown before launching finishes.
	var launched atomic.Bool
	app.Event.OnApplicationEvent(events.Mac.ApplicationDidFinishLaunching, func(_ *application.ApplicationEvent) {
		launched.Store(true)
		showWindow()
		confirmations.Watch(panel.Follow)
	})
	// An otpauth link can arrive before launching finishes; launching then shows the window.
	app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(event *application.ApplicationEvent) {
		if controls.ReceiveCodeSetup(event.Context().URL()) && launched.Load() {
			showWindow()
		}
	})
	app.Event.OnApplicationEvent(events.Common.SystemWillSleep, func(_ *application.ApplicationEvent) { lock() })
	app.Event.OnApplicationEvent(events.Common.SystemDidWake, func(_ *application.ApplicationEvent) { lock() })
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(_ *application.ApplicationEvent) {
		showWindow()
	})
	tray := menubar.New(app, trayIcon, settings.MenuBar(), menubar.Actions{
		Toggle: func() {
			if window.InFront() {
				window.Hide()
				return
			}
			showWindow()
		},
		Open:     showWindow,
		Lock:     lockOpenVault,
		Quit:     app.Quit,
		Unlocked: vault.Unlocked,
	})
	settings.OnLanguageChange(func() { tray.Relabel(settings.MenuBar()) })
	go autolock.New(settings.AutoLock, lockOpenVault).Run(watching)
	go core.Backups.Run(watching)
	go filewatch.New(vault, followVaultFile).Run(watching)
	go identities.Run(watching)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func warnIf(failure string, err error) {
	if err != nil {
		slog.Warn(failure, "err", err)
	}
}
