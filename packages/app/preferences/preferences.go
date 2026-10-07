// Package preferences keeps the device-local choices that are not vault data.
package preferences

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/dortanes/ravenpass/packages/app/changecount"
	"github.com/dortanes/ravenpass/packages/app/privatefile"
)

// Language is a tag Ravenpass offers. English is the source catalog and the fallback.
type Language string

// Languages Ravenpass offers.
const (
	English Language = "en"
	Russian Language = "ru"
)

const (
	maxRecordBytes = 4 << 10
	recordVersion  = 1
)

// ErrUnsupportedLanguage reports a language this build does not offer.
var ErrUnsupportedLanguage = errors.New("unsupported language")

// Languages lists the tags this build offers, in the order the interface shows them.
func Languages() []Language { return []Language{English, Russian} }

// supported reports whether this build offers language.
func supported(language Language) bool {
	for _, offered := range Languages() {
		if offered == language {
			return true
		}
	}
	return false
}

type record struct {
	Version int `json:"version"`
	// Language is empty until the user chooses one, meaning the device's language.
	Language Language `json:"language,omitempty"`
	// DefaultGroup is a vault group identifier, empty for none; a later vault may not hold it.
	DefaultGroup  string `json:"defaultGroup,omitempty"`
	ClipboardKept bool   `json:"clipboardKept,omitempty"`
	// ClipboardClearSeconds is zero for the default delay.
	ClipboardClearSeconds int  `json:"clipboardClearSeconds,omitempty"`
	AutoLockOff           bool `json:"autoLockOff,omitempty"`
	// AutoLockSeconds is nil for the host's default; zero locks as the app is hidden.
	AutoLockSeconds    *int `json:"autoLockSeconds,omitempty"`
	SiteIconsOff       bool `json:"siteIconsOff,omitempty"`
	HideDockWithWindow bool `json:"hideDockWithWindow,omitempty"`
	BankDetailsOff     bool `json:"bankDetailsOff,omitempty"`
	// SignInStyle is empty for SignInCard.
	SignInStyle           SignInStyle `json:"signInStyle,omitempty"`
	ConfirmExtensionFills bool        `json:"confirmExtensionFills,omitempty"`
	// Shortcuts maps an interface action to its chosen hotkey; an absent action keeps its default.
	Shortcuts map[string]string `json:"shortcuts,omitempty"`
	// InterfaceSize is zero for defaultInterfaceSize.
	InterfaceSize      int  `json:"interfaceSize,omitempty"`
	IdentityListOn     bool `json:"identityListOn,omitempty"`
	ScreenshotsAllowed bool `json:"screenshotsAllowed,omitempty"`
	BreachChecksOn     bool `json:"breachChecksOn,omitempty"`
	// Appearance is empty for AppearanceSystem.
	Appearance Appearance `json:"appearance,omitempty"`
	// AutoBackupOn requires AutoBackupFolder.
	AutoBackupOn bool `json:"autoBackupOn,omitempty"`
	// AutoBackupInterval is empty for BackupDaily.
	AutoBackupInterval BackupInterval `json:"autoBackupInterval,omitempty"`
	// AutoBackupKeep is zero for defaultBackupKeep.
	AutoBackupKeep int `json:"autoBackupKeep,omitempty"`
	// AutoBackupFolder is absent until the user chooses a folder.
	AutoBackupFolder *BackupFolder `json:"autoBackupFolder,omitempty"`
}

// Store reads and writes the preferences record; a missing or unreadable record means the defaults.
type Store struct {
	mu            sync.Mutex
	path          string
	device        DeviceLanguages
	autoLock      autoLockChoices
	titledPrompts bool
	current       record
	loaded        bool

	listenersMu         sync.Mutex
	languageListeners   []func()
	appearanceListeners []func()
	languageChanges     changecount.Counter
}

// New reads the record at path; without options the automatic lock counts from the device's last input.
func New(path string, device DeviceLanguages, options ...Option) (*Store, error) {
	if path == "" {
		return nil, errors.New("preferences path is required")
	}
	if device == nil {
		return nil, errors.New("device languages are required")
	}
	store := &Store{path: path, device: device, autoLock: idleAutoLock, current: record{Version: recordVersion}}
	for _, option := range options {
		option(store)
	}
	return store, nil
}

// Language reports the language in use and whether the user chose it; else the device's offered language or English.
func (s *Store) Language() (Language, bool) {
	s.mu.Lock()
	s.load()
	chosen := s.current.Language
	s.mu.Unlock()
	if chosen != "" {
		return chosen, true
	}
	return deviceLanguage(s.device()), false
}

// SetLanguage records an offered language.
func (s *Store) SetLanguage(language Language) error {
	if !supported(language) {
		return ErrUnsupportedLanguage
	}
	if err := s.update(func(next *record) { next.Language = language }); err != nil {
		return err
	}
	s.languageChanges.Record()
	s.notify(&s.languageListeners)
	return nil
}

// OnLanguageChange calls listener after each language the user records.
func (s *Store) OnLanguageChange(listener func()) {
	s.listen(&s.languageListeners, listener)
}

func (s *Store) listen(listeners *[]func(), listener func()) {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()
	*listeners = append(*listeners, listener)
}

// notify calls listeners outside the lock, so a listener may read the store.
func (s *Store) notify(listeners *[]func()) {
	s.listenersMu.Lock()
	called := slices.Clone(*listeners)
	s.listenersMu.Unlock()
	for _, listener := range called {
		listener()
	}
}

// AwaitLanguageChange returns the count of recorded languages since start once it passes seen.
func (s *Store) AwaitLanguageChange(ctx context.Context, seen uint64) (uint64, error) {
	return s.languageChanges.Await(ctx, seen)
}

// DefaultGroup reports the group new passwords join, empty when they join none.
func (s *Store) DefaultGroup() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	return s.current.DefaultGroup
}

// SetDefaultGroup records the group new passwords join. An empty identifier means none.
func (s *Store) SetDefaultGroup(group string) error {
	return s.update(func(next *record) { next.DefaultGroup = group })
}

// update writes the record with one change applied and keeps it only once it is on disk.
func (s *Store) update(change func(*record)) error {
	return s.apply(func(next *record) error {
		change(next)
		return nil
	})
}

// apply is update for a change that can refuse the record it finds.
func (s *Store) apply(change func(*record) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	next := s.current
	if err := change(&next); err != nil {
		return err
	}
	if err := s.save(next); err != nil {
		return err
	}
	s.current = next
	return nil
}

func (s *Store) load() {
	if s.loaded {
		return
	}
	s.loaded = true
	data, err := privatefile.Read(s.path, maxRecordBytes)
	if err != nil {
		return
	}
	var stored record
	if err := json.Unmarshal(data, &stored); err != nil {
		return
	}
	if stored.Version != recordVersion {
		return
	}
	if !supported(stored.Language) {
		stored.Language = ""
	}
	if !offeredClearDelay(stored.ClipboardClearSeconds) {
		stored.ClipboardClearSeconds = 0
	}
	s.keepOfferedAutoLock(&stored)
	if !offeredSignInStyle(stored.SignInStyle) {
		stored.SignInStyle = ""
	}
	if !validShortcuts(stored.Shortcuts) {
		stored.Shortcuts = nil
	}
	keepOfferedAutoBackup(&stored)
	s.current = stored
}

func (s *Store) save(stored record) error {
	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	if err := privatefile.Write(s.path, data); err != nil {
		return fmt.Errorf("save preferences: %w", err)
	}
	return nil
}
