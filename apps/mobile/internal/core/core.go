//go:build android

// Package core holds the vault services the process builds once, shared by the window and autofill.
package core

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/dortanes/ravenpass/apps/mobile/internal/appfiles"
	"github.com/dortanes/ravenpass/apps/mobile/internal/autofill"
	"github.com/dortanes/ravenpass/apps/mobile/internal/background"
	"github.com/dortanes/ravenpass/apps/mobile/internal/bridge"
	"github.com/dortanes/ravenpass/apps/mobile/internal/documents"
	"github.com/dortanes/ravenpass/apps/mobile/internal/keystore"
	"github.com/dortanes/ravenpass/packages/app/api"
	"github.com/dortanes/ravenpass/packages/app/appcore"
	"github.com/dortanes/ravenpass/packages/app/preferences"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Core is the process's vault and what serves it.
type Core struct {
	// Service is what the interface binds to.
	Service *api.Service

	controls api.HostControls
	vault    *vaultservice.Service
	away     *background.Lock
	shows    *api.UnlockOnShow
	autofill *autofill.Handler
	app      atomic.Pointer[application.App]
	window   atomic.Pointer[application.WebviewWindow]
}

var (
	built = sync.OnceValues(build)
	// early counts the screens shown while no core exists; the core takes their holds once built.
	early struct {
		sync.Mutex
		ready bool
		shown int
	}
)

// Get returns the process's core, built on first use.
func Get() (*Core, error) { return built() }

// Autofill answers a request of the autofill service or its screens, or nothing when the core cannot be built. A
// screen's shown or hidden notice comes from the main thread, so it never builds the core, which reads files and the
// keystore; its screen's first request, made on a worker, does.
func Autofill(request []byte) []byte {
	if shown, notice := autofill.ScreenNotice(request); notice && noteEarly(shown) {
		return autofill.Noted()
	}
	c, err := Get()
	if err != nil {
		slog.Warn("the app could not start for an autofill request", "err", err)
		return nil
	}
	return c.autofill.Call(request)
}

// ReceiveCodeSetup holds an otpauth link Android opened in Ravenpass until the owner adds it to a credential.
func ReceiveCodeSetup(link string) {
	c, err := Get()
	if err != nil {
		slog.Warn("the app could not start for a code setup link", "err", err)
		return
	}
	c.controls.ReceiveCodeSetup(link)
}

// noteEarly counts a screen notice while no core exists, reporting whether it did.
func noteEarly(shown bool) bool {
	early.Lock()
	defer early.Unlock()
	switch {
	case early.ready:
		return false
	case shown:
		early.shown++
	case early.shown > 0:
		early.shown--
	}
	return true
}

// takeEarly hands the screens shown before c existed to its handler, after which c answers every notice.
func takeEarly(c *Core) {
	early.Lock()
	defer early.Unlock()
	c.autofill.ShowScreens(early.shown)
	early.shown = 0
	early.ready = true
}

// Attach makes window, in app, the interface's window as its activity starts.
func (c *Core) Attach(app *application.App, window *application.WebviewWindow) {
	c.app.Store(app)
	c.window.Store(window)
	c.away.Shown()
	c.shows.Shown()
}

// Detach drops the window once its activity is destroyed: Wails waits forever on JavaScript sent to a window whose activity is gone.
func (c *Core) Detach() { c.window.Store(nil) }

// Shown stops the automatic lock's countdown and brings an open vault up to its file.
func (c *Core) Shown() {
	c.away.Shown()
	c.shows.Shown()
	go c.follow()
}

// Hidden starts the automatic lock's countdown unless something holds it.
func (c *Core) Hidden() {
	c.shows.Hidden()
	c.away.Hidden()
}

// Lock locks an open vault, clears the app's clipboard copy, and reloads the interface's window.
func (c *Core) Lock() { c.lockWith(c.Service.Lock) }

// lockAway leaves a clipboard copy until its clearing delay, for the app the owner switched to.
func (c *Core) lockAway() { c.lockWith(c.controls.LockAway) }

func (c *Core) lockWith(lock func() error) {
	if c.vault.Unlocked() {
		if err := lock(); err != nil {
			slog.Warn("locking the vault failed", "err", err)
		}
		c.reload()
	}
}

func (c *Core) follow() {
	if c.controls.FollowVaultFile() {
		c.reload()
	}
}

// reload must not wait: an autofill unlock answers while the activity is being destroyed.
func (c *Core) reload() {
	if window := c.window.Load(); window != nil {
		go window.ExecJS("window.location.reload()")
	}
}

func build() (*Core, error) {
	directory := bridge.FilesDirectory()
	if directory == "" {
		return nil, errors.New("the app's files directory is unavailable")
	}
	provider := bridge.Documents{}
	hardware := bridge.Keystore{}
	folders := documents.BackupFolders{Provider: provider}
	services, err := appcore.Build(appcore.Platform{
		Directory:         directory,
		Languages:         bridge.PreferredLanguages,
		Preferences:       []preferences.Option{preferences.LockWhenHidden(), preferences.TitledPrompts()},
		Backends:          []storage.Backend{documents.NewBackend(provider)},
		Owner:             bridge.Owner{},
		PIN:               keystore.Binding{Hardware: hardware},
		Presence:          keystore.PresenceBinding{Hardware: hardware},
		BackupDestination: folders,
	})
	if err != nil {
		return nil, err
	}
	settings := services.Settings
	settings.OnLanguageChange(func() { followLanguage(settings) })
	followLanguage(settings)
	settings.OnAppearanceChange(func() { followAppearance(settings) })
	followAppearance(settings)
	c := &Core{vault: services.Vault, shows: &api.UnlockOnShow{}}
	c.away = background.New(settings.AutoLock, c.lockAway)
	c.Service, err = services.Serve(api.Host{
		Confirmations: api.Confirmations{
			ShowMain:   func() {},
			ReloadMain: c.reload,
		},
		CurrentApp:    c.app.Load,
		Pasteboard:    bridge.Clipboard{},
		Files:         documents.Files{Provider: provider},
		Places:        appfiles.Places{Directory: directory, Dialogs: settings.Dialogs},
		Photos:        bridge.Photos{},
		Saver:         documents.Saves{Provider: provider},
		Printer:       bridge.Printer{},
		BackupFolders: folders,
		About:         bridge.About{},
		Apps:          bridge.Apps{},
		// Wails' browser manager runs desktop tools; its Android bridge sends an ACTION_VIEW intent.
		OpenURL: func(address string) error {
			application.Android.OpenURL(address)
			return nil
		},
		Hold:           c.away.Hold,
		UnlockOnShow:   c.shows,
		Screenshots:    bridge.Screens{},
		SystemAutofill: bridge.SystemAutofill{},
		TemporaryPicks: true,
		Offers:         api.Capabilities{StorageLocations: true, InterfaceSize: true, CopyScans: true, Appearance: true},
	})
	if err != nil {
		return nil, err
	}
	c.controls = api.ControlsOf(c.Service)
	// The core lives as long as the process.
	go services.Backups.Run(context.Background())
	c.autofill = autofill.NewHandler(autofill.Options{
		Service: services.Autofill,
		Vault:   services.Vault,
		Owner:   services.Owner,
		Icons:   services.Icons,
		Page:    c.Service,
		Links:   autofill.NewAssetLinks(&http.Client{}),
		Reason:  func() string { return settings.Dialogs().UnlockVault },
		Words:   services.Words,
		Opened: func() {
			c.away.AutofillUnlocked()
			services.Confirmations.VaultOpened()
			c.reload()
		},
		Follow:    c.follow,
		Requested: c.away.AutofillRequested,
		Hold:      c.away.Hold,
	})
	takeEarly(c)
	return c, nil
}

// followLanguage gives Android's own text the language the owner chose in Ravenpass, else the device's.
func followLanguage(settings *preferences.Store) {
	tag := ""
	if language, chosen := settings.Language(); chosen {
		tag = string(language)
	}
	bridge.SetLanguage(tag)
}

// followAppearance gives Android's own screens and the app's windows the appearance the owner chose.
func followAppearance(settings *preferences.Store) {
	bridge.SetAppearance(string(settings.Appearance()))
}
