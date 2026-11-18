package api

import (
	"errors"
	"testing"
)

// recordedLoginItem keeps what it is set to, failing each change with err, and counts the screens it shows.
type recordedLoginItem struct {
	status LaunchAtLogin
	shown  int
	err    error
}

func (l *recordedLoginItem) Status() LaunchAtLogin { return l.status }

func (l *recordedLoginItem) Set(enabled bool) error {
	if l.err != nil {
		return l.err
	}
	l.status.Enabled = enabled
	return nil
}

func (l *recordedLoginItem) OpenSettings() { l.shown++ }

func TestAHostWithALoginItemOffersOpeningAtLogin(t *testing.T) {
	if newServiceOnHost(t, Host{}).Capabilities().LaunchAtLogin {
		t.Fatal("a host without the port offers opening at login")
	}
	if !newServiceOnHost(t, Host{LoginItem: &recordedLoginItem{}}).Capabilities().LaunchAtLogin {
		t.Fatal("a host with the port does not offer opening at login")
	}
}

func TestOpeningAtLoginIsWhatTheSystemHolds(t *testing.T) {
	item := &recordedLoginItem{status: LaunchAtLogin{Enabled: true, NeedsApproval: true}}
	service := newServiceOnHost(t, Host{LoginItem: item})
	if status, err := service.GetLaunchAtLogin(); err != nil || status != item.status {
		t.Fatalf("status = %+v, %v; want %+v", status, err, item.status)
	}
	if err := service.SetLaunchAtLogin(false); err != nil {
		t.Fatal(err)
	}
	if status, _ := service.GetLaunchAtLogin(); status.Enabled {
		t.Fatal("the login item stays enabled after turning it off")
	}
	service.OpenLoginItems()
	if item.shown != 1 {
		t.Fatalf("login items shown %d times, want 1", item.shown)
	}
}

func TestARefusedLoginItemChangeFailsInWords(t *testing.T) {
	service := newServiceOnHost(t, Host{LoginItem: &recordedLoginItem{err: errors.New("refused")}})
	if err := service.SetLaunchAtLogin(true); err == nil || err.Error() != fail(failureGeneral).Error() {
		t.Fatalf("setting = %v, want %s", err, failureGeneral)
	}
	if status, _ := service.GetLaunchAtLogin(); status.Enabled {
		t.Fatal("a refused change shows as enabled")
	}
}

func TestHiddenAtLoginSetting(t *testing.T) {
	service := &Service{preferences: newTestPreferences(t)}
	if current, err := service.GetHiddenAtLogin(); err != nil || !current.Hidden {
		t.Fatalf("default = %+v, %v; want hidden", current, err)
	}
	if err := service.SetHiddenAtLogin(false); err != nil {
		t.Fatal(err)
	}
	if current, _ := service.GetHiddenAtLogin(); current.Hidden {
		t.Fatal("a launch at login stays hidden after choosing the window")
	}
}

func TestAHostWithoutALoginItemRefusesTheChange(t *testing.T) {
	service := newServiceOnHost(t, Host{})
	if status, err := service.GetLaunchAtLogin(); err != nil || status != (LaunchAtLogin{}) {
		t.Fatalf("status = %+v, %v", status, err)
	}
	if err := service.SetLaunchAtLogin(true); err == nil || err.Error() != fail(failureGeneral).Error() {
		t.Fatalf("setting = %v, want %s", err, failureGeneral)
	}
	service.OpenLoginItems()
}
