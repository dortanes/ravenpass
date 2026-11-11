// Package linkstore keeps the link server's port, desktop key pair and linked extensions in an owner-only file.
package linkstore

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dortanes/ravenpass/packages/app/linkkey"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/privatefile"
)

const (
	recordVersion = 1
	// maxFileBytes is far above use: one linked extension takes under 1 KiB.
	maxFileBytes = 1 << 20
	idBytes      = 16
	maxPort      = 65535
)

var (
	// ErrMalformed reports a records file that cannot be read; it is never overwritten.
	ErrMalformed = errors.New("extension records are malformed")
	// ErrNotFound reports an extension that is not linked.
	ErrNotFound = errors.New("extension is not linked")
	// ErrInvalidName reports a name that is empty once trimmed or longer than linkproto.MaxNameLength characters.
	ErrInvalidName = errors.New("extension name is invalid")

	errInvalidExtension = errors.New("extension record is invalid")
	errInvalidPort      = errors.New("port is out of range")
)

// Extension is one linked extension. PublicKey is the static key it opens sessions with.
type Extension struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PublicKey []byte    `json:"publicKey"`
	LinkedAt  time.Time `json:"linkedAt"`
}

func (e Extension) clone() Extension {
	e.PublicKey = bytes.Clone(e.PublicKey)
	return e
}

func (e Extension) valid() bool {
	id, err := base64.RawURLEncoding.DecodeString(e.ID)
	return err == nil && len(id) == idBytes && linkproto.ValidName(e.Name) &&
		linkproto.ValidPublicKey(e.PublicKey) && !e.LinkedAt.IsZero()
}

type keyRecord struct {
	Private []byte `json:"private"`
	Public  []byte `json:"public"`
}

type record struct {
	Version    int         `json:"version"`
	Port       int         `json:"port,omitempty"`
	DesktopKey *keyRecord  `json:"desktopKey,omitempty"`
	P256Key    *keyRecord  `json:"p256Key,omitempty"`
	Extensions []Extension `json:"extensions"`
}

func (r record) valid() bool {
	if r.Version != recordVersion || (r.Port != 0 && !validPort(r.Port)) {
		return false
	}
	for _, keyed := range []struct {
		key  *keyRecord
		size int
	}{{r.DesktopKey, linkproto.KeySize}, {r.P256Key, linkproto.P256PublicKeySize}} {
		if keyed.key != nil && (len(keyed.key.Public) != keyed.size || !(linkproto.KeyPair{Private: keyed.key.Private, Public: keyed.key.Public}).Valid()) {
			return false
		}
	}
	ids := make(map[string]bool, len(r.Extensions))
	keys := make(map[string]bool, len(r.Extensions))
	for _, extension := range r.Extensions {
		if !extension.valid() || ids[extension.ID] || keys[string(extension.PublicKey)] {
			return false
		}
		ids[extension.ID], keys[string(extension.PublicKey)] = true, true
	}
	return true
}

func validPort(port int) bool {
	return port >= linkkey.MinPort && port <= maxPort
}

// Store replaces extensions.json whole on every change; the file holds the desktop link private key and is kept out of backups.
type Store struct {
	mu      sync.Mutex
	path    string
	backups privatefile.BackupExclusion
	now     func() time.Time
	current record
	loaded  bool
}

// New returns a store for the records file at path.
func New(path string, backups privatefile.BackupExclusion) (*Store, error) {
	if path == "" {
		return nil, errors.New("extension records path is required")
	}
	if backups == nil {
		return nil, errors.New("a backup exclusion is required")
	}
	return &Store{path: path, backups: backups, now: time.Now}, nil
}

// Port reports the port the link server keeps, zero while none has been chosen.
func (s *Store) Port() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return 0, err
	}
	return s.current.Port, nil
}

// SetPort keeps port for the link server.
func (s *Store) SetPort(port int) error {
	if !validPort(port) {
		return errInvalidPort
	}
	return s.apply(func(next *record) error {
		next.Port = port
		return nil
	})
}

// DesktopKey returns the desktop app's static key pair, generating and keeping it on first use.
func (s *Store) DesktopKey() (linkproto.KeyPair, error) {
	return s.desktopKey(false)
}

// P256DesktopKey returns the separate v2 static key, preserving every v1 link.
func (s *Store) P256DesktopKey() (linkproto.KeyPair, error) {
	return s.desktopKey(true)
}

func (s *Store) desktopKey(p256 bool) (linkproto.KeyPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return linkproto.KeyPair{}, err
	}
	kept := s.current.DesktopKey
	generate := linkproto.GenerateKeyPair
	if p256 {
		kept = s.current.P256Key
		generate = linkproto.GenerateP256KeyPair
	}
	if kept == nil {
		generated, err := generate(rand.Reader)
		if err != nil {
			return linkproto.KeyPair{}, err
		}
		next := s.current
		kept = &keyRecord{Private: generated.Private, Public: generated.Public}
		if p256 {
			next.P256Key = kept
		} else {
			next.DesktopKey = kept
		}
		if err := s.save(next); err != nil {
			return linkproto.KeyPair{}, err
		}
		s.current = next
	}
	return linkproto.KeyPair{
		Private: bytes.Clone(kept.Private),
		Public:  bytes.Clone(kept.Public),
	}, nil
}

// Extensions lists the linked extensions in the order they were linked.
func (s *Store) Extensions() ([]Extension, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return nil, err
	}
	extensions := make([]Extension, len(s.current.Extensions))
	for i, extension := range s.current.Extensions {
		extensions[i] = extension.clone()
	}
	return extensions, nil
}

// Find returns the linked extension with publicKey, and whether there is one.
func (s *Store) Find(publicKey []byte) (Extension, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return Extension{}, false, err
	}
	for _, extension := range s.current.Extensions {
		if bytes.Equal(extension.PublicKey, publicKey) {
			return extension.clone(), true, nil
		}
	}
	return Extension{}, false, nil
}

// Add records a newly linked extension under a new identifier, replacing one with the same public key.
func (s *Store) Add(name string, publicKey []byte) (Extension, error) {
	id := make([]byte, idBytes)
	if _, err := rand.Read(id); err != nil {
		return Extension{}, err
	}
	extension := Extension{
		ID:        base64.RawURLEncoding.EncodeToString(id),
		Name:      name,
		PublicKey: bytes.Clone(publicKey),
		LinkedAt:  s.now().UTC().Truncate(time.Second),
	}
	if !extension.valid() {
		return Extension{}, errInvalidExtension
	}
	err := s.apply(func(next *record) error {
		next.Extensions = slices.DeleteFunc(next.Extensions, func(existing Extension) bool {
			return bytes.Equal(existing.PublicKey, publicKey)
		})
		next.Extensions = append(next.Extensions, extension)
		return nil
	})
	if err != nil {
		return Extension{}, err
	}
	return extension.clone(), nil
}

// Remove forgets the extension with id.
func (s *Store) Remove(id string) error {
	return s.apply(func(next *record) error {
		index := slices.IndexFunc(next.Extensions, func(existing Extension) bool { return existing.ID == id })
		if index < 0 {
			return ErrNotFound
		}
		next.Extensions = slices.Delete(next.Extensions, index, index+1)
		return nil
	})
}

// Rename names the extension with id anew, trimmed of surrounding white space.
func (s *Store) Rename(id, name string) error {
	name = strings.TrimSpace(name)
	if !linkproto.ValidName(name) {
		return ErrInvalidName
	}
	return s.apply(func(next *record) error {
		index := slices.IndexFunc(next.Extensions, func(existing Extension) bool { return existing.ID == id })
		if index < 0 {
			return ErrNotFound
		}
		next.Extensions[index].Name = name
		return nil
	})
}

// apply writes the record with one change applied to a copy, and keeps it only once it is on disk.
func (s *Store) apply(change func(*record) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return err
	}
	next := s.current
	next.Extensions = slices.Clone(s.current.Extensions)
	if err := change(&next); err != nil {
		return err
	}
	if err := s.save(next); err != nil {
		return err
	}
	s.current = next
	return nil
}

func (s *Store) load() error {
	if s.loaded {
		return nil
	}
	data, err := privatefile.Read(s.path, maxFileBytes)
	if errors.Is(err, fs.ErrNotExist) {
		s.current = record{Version: recordVersion}
		s.loaded = true
		return nil
	}
	if errors.Is(err, fs.ErrInvalid) {
		return ErrMalformed
	}
	if err != nil {
		return fmt.Errorf("read extension records: %w", err)
	}
	var stored record
	if err := json.Unmarshal(data, &stored); err != nil || !stored.valid() {
		return ErrMalformed
	}
	s.current = stored
	s.loaded = true
	return nil
}

func (s *Store) save(next record) error {
	if next.Extensions == nil {
		next.Extensions = []Extension{}
	}
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := privatefile.Write(s.path, data, privatefile.ExcludedFromBackups(s.backups)); err != nil {
		return fmt.Errorf("save extension records: %w", err)
	}
	return nil
}
