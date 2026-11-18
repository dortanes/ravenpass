package api

import "errors"

// LoginItem is the system's record of whether Ravenpass opens when the owner logs in.
type LoginItem interface {
	// Status reports what the system holds.
	Status() LaunchAtLogin
	// Set asks the system to open Ravenpass at login or to stop; a blocked item is reported by Status, not as a failure.
	Set(enabled bool) error
	// OpenSettings shows the system screen listing login items.
	OpenSettings()
}

// LaunchAtLogin reports whether Ravenpass is set to open at login.
type LaunchAtLogin struct {
	Enabled bool `json:"enabled"`
	// NeedsApproval reports that the owner blocked the enabled item in the system's settings, so it does not open yet.
	NeedsApproval bool `json:"needsApproval"`
}

// HiddenAtLogin reports whether a launch at login stays in the menu bar.
type HiddenAtLogin struct {
	Hidden bool `json:"hidden"`
}

var errNoLoginItem = errors.New("the host has no login item")

// noLoginItem is the login item of a host the system does not open at login.
type noLoginItem struct{}

func (noLoginItem) Status() LaunchAtLogin { return LaunchAtLogin{} }

func (noLoginItem) Set(bool) error { return errNoLoginItem }

func (noLoginItem) OpenSettings() {}

// GetLaunchAtLogin reports whether the system opens Ravenpass at login.
func (s *Service) GetLaunchAtLogin() (LaunchAtLogin, error) {
	return s.loginItem.Status(), nil
}

// SetLaunchAtLogin asks the system to open Ravenpass at login or to stop.
func (s *Service) SetLaunchAtLogin(enabled bool) error {
	return present(s.loginItem.Set(enabled))
}

// GetHiddenAtLogin reports whether a launch at login stays in the menu bar.
func (s *Service) GetHiddenAtLogin() (HiddenAtLogin, error) {
	return HiddenAtLogin{Hidden: s.preferences.HiddenAtLogin()}, nil
}

// SetHiddenAtLogin records whether a launch at login stays in the menu bar, from the next login on.
func (s *Service) SetHiddenAtLogin(hidden bool) error {
	return present(s.preferences.SetHiddenAtLogin(hidden))
}

// OpenLoginItems shows the system screen where the owner allows or blocks login items.
func (s *Service) OpenLoginItems() {
	s.loginItem.OpenSettings()
}
