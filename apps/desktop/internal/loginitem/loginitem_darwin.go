//go:build darwin && cgo

package loginitem

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework ServiceManagement
#include "loginitem_darwin.h"
*/
import "C"

import (
	"fmt"

	"github.com/dortanes/ravenpass/packages/app/api"
)

const supported = true

func status() api.LaunchAtLogin {
	switch C.ravenpass_login_item_status() {
	case C.RAVENPASS_LOGIN_ITEM_ENABLED:
		return api.LaunchAtLogin{Enabled: true}
	case C.RAVENPASS_LOGIN_ITEM_BLOCKED:
		return api.LaunchAtLogin{Enabled: true, NeedsApproval: true}
	default:
		return api.LaunchAtLogin{}
	}
}

func set(enabled bool) error {
	if code := C.ravenpass_login_item_set(C.bool(enabled)); code != 0 {
		return fmt.Errorf("macOS refused the login item change: error %d", int64(code))
	}
	return nil
}

func openSettings() { C.ravenpass_login_item_open_settings() }

func watchLaunch() { C.ravenpass_login_item_watch_launch() }

func launchedAtLogin() bool { return bool(C.ravenpass_login_item_launched_at_login()) }
