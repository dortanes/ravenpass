//go:build !darwin || !cgo

package loginitem

import (
	"errors"

	"github.com/dortanes/ravenpass/packages/app/api"
)

const supported = false

func status() api.LaunchAtLogin { return api.LaunchAtLogin{} }

func set(bool) error { return errors.New("login items exist on macOS only") }

func openSettings() {}

func watchLaunch() {}

func launchedAtLogin() bool { return false }
