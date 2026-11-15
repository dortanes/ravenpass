package api

import (
	"errors"
	"time"

	"github.com/dortanes/ravenpass/packages/app/passwordhistory"
	"github.com/dortanes/ravenpass/packages/app/preferences"
)

// PasswordHistorySetting is whether the generator's history is kept, for how many days, and the periods offered.
type PasswordHistorySetting struct {
	Enabled bool  `json:"enabled"`
	Days    int   `json:"days"`
	Offered []int `json:"offered"`
}

// GeneratedPassword is a password the generator handed out; At is RFC 3339 in UTC.
type GeneratedPassword struct {
	Value string `json:"value"`
	Mode  string `json:"mode"`
	At    string `json:"at"`
}

// GetPasswordHistory reports the history setting.
func (s *Service) GetPasswordHistory() (PasswordHistorySetting, error) {
	setting := s.preferences.PasswordHistory()
	return PasswordHistorySetting{Enabled: setting.Enabled, Days: setting.Days, Offered: preferences.HistoryDays()}, nil
}

// SetPasswordHistory records the setting and deletes what it no longer keeps from the open vault's history: all of
// it when turned off, else what is older than days.
func (s *Service) SetPasswordHistory(enabled bool, days int) error {
	setting := preferences.PasswordHistory{Enabled: enabled, Days: days}
	if err := s.preferences.SetPasswordHistory(setting); err != nil {
		if errors.Is(err, preferences.ErrUnsupportedHistoryDays) {
			return fail(failureInvalidItem)
		}
		return present(err)
	}
	if s.history == nil {
		return nil
	}
	return present(s.history.Apply(setting))
}

// CountPasswordHistoryPast reports how many passwords in the open vault's history a period of days would delete.
func (s *Service) CountPasswordHistoryPast(days int) (int, error) {
	if s.history == nil {
		return 0, nil
	}
	count, err := s.history.CountPast(days)
	return count, present(err)
}

// PasswordHistory lists the open vault's history, newest first.
func (s *Service) PasswordHistory() ([]GeneratedPassword, error) {
	listed := []GeneratedPassword{}
	if s.history == nil {
		return listed, nil
	}
	passwords, err := s.history.List()
	if err != nil {
		return nil, present(err)
	}
	for _, password := range passwords {
		listed = append(listed, GeneratedPassword{
			Value: password.Value, Mode: string(password.Mode), At: password.At.UTC().Format(time.RFC3339),
		})
	}
	return listed, nil
}

// RecordGeneratedPassword adds a password the generator handed out to the open vault's history; nothing while the
// history is off.
func (s *Service) RecordGeneratedPassword(value string, mode string) error {
	if s.history == nil {
		return nil
	}
	err := s.history.Record(value, passwordhistory.Mode(mode))
	if errors.Is(err, passwordhistory.ErrInvalidPassword) {
		return fail(failureInvalidItem)
	}
	return present(err)
}

// ClearPasswordHistory deletes the open vault's history.
func (s *Service) ClearPasswordHistory() error {
	if s.history == nil {
		return nil
	}
	return present(s.history.Clear())
}

// CopyGeneratedPassword copies a generated password; the clipboard clears it like any other copied secret.
func (s *Service) CopyGeneratedPassword(value string) error {
	if value == "" {
		return fail(failureInvalidItem)
	}
	if !s.copyToClipboard(value, clearAfter) {
		return fail(failureCopyFailed)
	}
	return nil
}
