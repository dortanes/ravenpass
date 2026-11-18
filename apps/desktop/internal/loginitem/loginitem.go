// Package loginitem keeps the running app bundle among the macOS login items.
package loginitem

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dortanes/ravenpass/packages/app/api"
)

// Mac is the login item macOS holds for the app bundle the process runs from.
type Mac struct{}

// Available reports whether the process runs from an app bundle, the only thing macOS opens as this login item.
func Available() bool {
	executable, err := os.Executable()
	return err == nil && supported && bundled(executable)
}

// WatchLaunch must run before the app does, for LaunchedAtLogin to see how macOS started it.
func WatchLaunch() { watchLaunch() }

// LaunchedAtLogin reports whether macOS started the app as a login item; it is known once launching has finished.
func LaunchedAtLogin() bool { return launchedAtLogin() }

func (Mac) Status() api.LaunchAtLogin { return status() }

func (Mac) Set(enabled bool) error { return set(enabled) }

func (Mac) OpenSettings() { openSettings() }

func bundled(executable string) bool {
	return strings.HasSuffix(filepath.Dir(executable), ".app/Contents/MacOS")
}
