// Package passwordhistory keeps, for each vault on this device, the passwords the generator handed out.
package passwordhistory

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/dortanes/ravenpass/packages/app/preferences"
	"github.com/dortanes/ravenpass/packages/app/privatefile"
	"github.com/dortanes/ravenpass/packages/vault"
)

const (
	// ownerDataName names the history among a vault's owner data.
	ownerDataName = "password-history"
	fileVersion   = 1
	maxPasswords  = 500
	maxFileBytes  = 1 << 20
	setAsideExt   = ".unreadable"
)

// Mode is how the generator made a password.
type Mode string

const (
	Words      Mode = "words"
	Characters Mode = "characters"
)

// ErrInvalidPassword reports a password that is empty, too long, or of an unknown mode.
var ErrInvalidPassword = errors.New("invalid generated password")

// Password is one password the generator handed out, and when.
type Password struct {
	Value string    `json:"value"`
	Mode  Mode      `json:"mode"`
	At    time.Time `json:"at"`
}

// Vault seals the history with the open vault.
type Vault interface {
	OpenVaultID() (vault.ID, error)
	SealOwnerData(id vault.ID, name string, plaintext []byte) ([]byte, error)
	OpenOwnerData(id vault.ID, name string, sealed []byte) ([]byte, error)
}

// Store keeps one sealed file per vault in its directory. Every method is safe for concurrent use.
type Store struct {
	mu        sync.Mutex
	directory string
	vault     Vault
	setting   func() preferences.PasswordHistory
	now       func() time.Time
}

// New returns a Store over directory that keeps passwords as setting says.
func New(directory string, vault Vault, setting func() preferences.PasswordHistory) *Store {
	return &Store{directory: directory, vault: vault, setting: setting, now: time.Now}
}

type file struct {
	Version   int        `json:"version"`
	Passwords []Password `json:"passwords"`
}

// List returns the open vault's history, newest first, without passwords past the period.
func (s *Store) List() ([]Password, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.vault.OpenVaultID()
	if err != nil {
		return nil, err
	}
	passwords, err := s.read(id)
	if err != nil {
		return nil, err
	}
	return s.within(passwords, s.setting().Days), nil
}

// Record adds a password the generator handed out to the open vault's history; nothing while the history is off.
func (s *Store) Record(value string, mode Mode) error {
	if value == "" || len(value) > 1024 || (mode != Words && mode != Characters) {
		return ErrInvalidPassword
	}
	setting := s.setting()
	if !setting.Enabled {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.vault.OpenVaultID()
	if err != nil {
		return err
	}
	passwords, err := s.read(id)
	if err != nil {
		return err
	}
	passwords = append([]Password{{Value: value, Mode: mode, At: s.now().UTC()}}, s.within(passwords, setting.Days)...)
	if len(passwords) > maxPasswords {
		passwords = passwords[:maxPasswords]
	}
	return s.write(id, passwords)
}

// Clear deletes the open vault's history.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.vault.OpenVaultID()
	if err != nil {
		return err
	}
	return removeIfPresent(s.path(id))
}

// CountPast reports how many passwords in the open vault's history are older than days.
func (s *Store) CountPast(days int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.vault.OpenVaultID()
	if err != nil {
		return 0, err
	}
	passwords, err := s.read(id)
	if err != nil {
		return 0, err
	}
	return len(passwords) - len(s.within(passwords, days)), nil
}

// Apply removes what setting no longer keeps from the open vault's history: all of it when off, else what is older
// than its period.
func (s *Store) Apply(setting preferences.PasswordHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.vault.OpenVaultID()
	if err != nil {
		return err
	}
	if !setting.Enabled {
		return removeIfPresent(s.path(id))
	}
	passwords, err := s.read(id)
	if err != nil {
		return err
	}
	kept := s.within(passwords, setting.Days)
	if len(kept) == len(passwords) {
		return nil
	}
	return s.write(id, kept)
}

// Reseal passes the history of id through reseal for a vault key change; it implements vaultservice.OwnerDataHolder.
// The key change calls it holding the vault service's lock, which every other method takes after s.mu, so it takes
// no lock of its own: a write racing it seals through that service and lands after it, under the new key.
func (s *Store) Reseal(id vault.ID, reseal func(name string, sealed []byte) ([]byte, error)) (func(), error) {
	path := s.path(id)
	sealed, err := privatefile.Read(path, maxFileBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return func() {}, nil
	}
	if err != nil {
		return nil, err
	}
	next, err := reseal(ownerDataName, sealed)
	if err != nil {
		return nil, err
	}
	return func() {
		if err := privatefile.Write(path, next); err != nil {
			slog.Warn("write a password history sealed for a new vault key", "err", err)
		}
	}, nil
}

// within returns passwords from the last days, newest first.
func (s *Store) within(passwords []Password, days int) []Password {
	since := s.now().Add(-time.Duration(days) * 24 * time.Hour)
	kept := make([]Password, 0, len(passwords))
	for _, password := range passwords {
		if password.At.After(since) {
			kept = append(kept, password)
		}
	}
	slices.SortStableFunc(kept, func(a, b Password) int { return b.At.Compare(a.At) })
	return kept
}

// read opens the history of id; a file the vault key no longer opens is set aside and the history starts empty.
func (s *Store) read(id vault.ID) ([]Password, error) {
	path := s.path(id)
	sealed, err := privatefile.Read(path, maxFileBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, s.setAside(path)
	}
	plaintext, err := s.vault.OpenOwnerData(id, ownerDataName, sealed)
	if err != nil {
		if errors.Is(err, vault.ErrAuthentication) || errors.Is(err, vault.ErrMalformed) ||
			errors.Is(err, vault.ErrUnsupported) || errors.Is(err, vault.ErrResourceLimit) {
			return nil, s.setAside(path)
		}
		return nil, err
	}
	defer clear(plaintext)
	var stored file
	if err := json.Unmarshal(plaintext, &stored); err != nil || stored.Version != fileVersion {
		return nil, s.setAside(path)
	}
	return stored.Passwords, nil
}

func (s *Store) write(id vault.ID, passwords []Password) error {
	if len(passwords) == 0 {
		return removeIfPresent(s.path(id))
	}
	plaintext, err := json.Marshal(file{Version: fileVersion, Passwords: passwords})
	if err != nil {
		return err
	}
	sealed, err := s.vault.SealOwnerData(id, ownerDataName, plaintext)
	clear(plaintext)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.directory, 0o700); err != nil {
		return err
	}
	return privatefile.Write(s.path(id), sealed)
}

// setAside moves an unreadable history out of the way, replacing an earlier one set aside, so a later key may still
// open it while the history starts again.
func (s *Store) setAside(path string) error {
	if err := os.Rename(path, path+setAsideExt); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("set aside an unreadable password history: %w", err)
	}
	return nil
}

func (s *Store) path(id vault.ID) string {
	return filepath.Join(s.directory, hex.EncodeToString(id[:])+".sealed")
}

func removeIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
