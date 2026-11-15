package passwordhistory

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/app/preferences"
	"github.com/dortanes/ravenpass/packages/vault"
)

// openVault seals with a real session for one vault.
type openVault struct {
	session *vault.Session
	id      vault.ID
}

func newOpenVault(t *testing.T) *openVault {
	t.Helper()
	created, err := vault.Create()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(created.Session.Lock)
	head, err := created.Session.Head()
	if err != nil {
		t.Fatal(err)
	}
	return &openVault{session: created.Session, id: head.VaultID}
}

func (v *openVault) OpenVaultID() (vault.ID, error) { return v.id, nil }

func (v *openVault) SealOwnerData(id vault.ID, name string, plaintext []byte) ([]byte, error) {
	if id != v.id {
		return nil, vault.ErrLocked
	}
	return v.session.SealOwnerData(name, plaintext)
}

func (v *openVault) OpenOwnerData(id vault.ID, name string, sealed []byte) ([]byte, error) {
	if id != v.id {
		return nil, vault.ErrLocked
	}
	return v.session.OpenOwnerData(name, sealed)
}

// testStore is a Store whose setting and clock the test moves.
type testStore struct {
	*Store
	setting preferences.PasswordHistory
	clock   time.Time
}

func newTestStore(t *testing.T, v Vault) *testStore {
	t.Helper()
	store := &testStore{
		setting: preferences.PasswordHistory{Enabled: true, Days: 30},
		clock:   time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC),
	}
	store.Store = New(t.TempDir(), v, func() preferences.PasswordHistory { return store.setting })
	store.now = func() time.Time { return store.clock }
	return store
}

func values(t *testing.T, store *testStore) []string {
	t.Helper()
	passwords, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	for _, password := range passwords {
		listed = append(listed, password.Value)
	}
	return listed
}

func TestTheHistoryListsNewestFirstAndKeepsNoPlaintextOnDisk(t *testing.T) {
	v := newOpenVault(t)
	store := newTestStore(t, v)
	for _, value := range []string{"Orbit-Velvet7-Canyon", "x7#Lq2!vR9", "Ladder-Mosquito3-Fern"} {
		if err := store.Record(value, Words); err != nil {
			t.Fatal(err)
		}
		store.clock = store.clock.Add(time.Minute)
	}
	if got := values(t, store); len(got) != 3 || got[0] != "Ladder-Mosquito3-Fern" || got[2] != "Orbit-Velvet7-Canyon" {
		t.Fatalf("history order: %q", got)
	}
	sealed, err := os.ReadFile(store.path(v.id))
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Orbit", "Velvet7", "x7#Lq2!vR9", "words"} {
		if bytes.Contains(sealed, []byte(value)) {
			t.Fatalf("the history file holds %q in plaintext", value)
		}
	}
}

func TestNothingIsKeptWhileTheHistoryIsOffOrForAnInvalidPassword(t *testing.T) {
	store := newTestStore(t, newOpenVault(t))
	store.setting.Enabled = false
	if err := store.Record("Orbit-Velvet7-Canyon", Words); err != nil {
		t.Fatal(err)
	}
	if got := values(t, store); len(got) != 0 {
		t.Fatalf("history kept while off: %q", got)
	}
	store.setting.Enabled = true
	for _, invalid := range []struct {
		value string
		mode  Mode
	}{{"", Words}, {"secret", "pin"}, {string(make([]byte, 1025)), Characters}} {
		if err := store.Record(invalid.value, invalid.mode); !errors.Is(err, ErrInvalidPassword) {
			t.Fatalf("recording %d bytes of mode %q: %v", len(invalid.value), invalid.mode, err)
		}
	}
}

func TestThePeriodDropsOlderPasswordsAndApplyCountsThenDeletesThem(t *testing.T) {
	store := newTestStore(t, newOpenVault(t))
	if err := store.Record("ten days old", Characters); err != nil {
		t.Fatal(err)
	}
	store.clock = store.clock.Add(10 * 24 * time.Hour)
	if err := store.Record("new", Characters); err != nil {
		t.Fatal(err)
	}
	if count, err := store.CountPast(7); err != nil || count != 1 {
		t.Fatalf("passwords past 7 days: %d, %v", count, err)
	}
	if err := store.Apply(preferences.PasswordHistory{Enabled: true, Days: 7}); err != nil {
		t.Fatal(err)
	}
	store.setting.Days = 90
	if got := values(t, store); len(got) != 1 || got[0] != "new" {
		t.Fatalf("history after a shorter period: %q", got)
	}
	store.clock = store.clock.Add(91 * 24 * time.Hour)
	if got := values(t, store); len(got) != 0 {
		t.Fatalf("history past its period: %q", got)
	}
}

func TestTurningTheHistoryOffOrClearingItDeletesTheFile(t *testing.T) {
	v := newOpenVault(t)
	store := newTestStore(t, v)
	if err := store.Record("Orbit-Velvet7-Canyon", Words); err != nil {
		t.Fatal(err)
	}
	if err := store.Apply(preferences.PasswordHistory{Enabled: false, Days: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.path(v.id)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("history file after turning it off: %v", err)
	}
	if err := store.Record("Ladder-Mosquito3-Fern", Words); err != nil {
		t.Fatal(err)
	}
	if err := store.Clear(); err != nil {
		t.Fatal(err)
	}
	if got := values(t, store); len(got) != 0 {
		t.Fatalf("history after clearing: %q", got)
	}
}

func TestTheHistoryHoldsAtMostItsLimit(t *testing.T) {
	store := newTestStore(t, newOpenVault(t))
	for i := range maxPasswords + 3 {
		if err := store.Record(string(rune('a'+i%26))+time.Duration(i).String(), Characters); err != nil {
			t.Fatal(err)
		}
		store.clock = store.clock.Add(time.Second)
	}
	if got := values(t, store); len(got) != maxPasswords {
		t.Fatalf("history holds %d passwords", len(got))
	}
}

func TestAHistoryFromAnotherVaultIsSetAsideAndTheHistoryStartsAgain(t *testing.T) {
	first := newOpenVault(t)
	store := newTestStore(t, first)
	if err := store.Record("Orbit-Velvet7-Canyon", Words); err != nil {
		t.Fatal(err)
	}
	second := newOpenVault(t)
	foreign, err := second.session.SealOwnerData(ownerDataName, []byte(`{"version":1,"passwords":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	path := store.path(first.id)
	if err := os.WriteFile(path, foreign, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := values(t, store); len(got) != 0 {
		t.Fatalf("history over an unreadable file: %q", got)
	}
	if aside, err := os.ReadFile(path + setAsideExt); err != nil || !bytes.Equal(aside, foreign) {
		t.Fatalf("the unreadable file was not set aside: %v", err)
	}
	if err := store.Record("Ladder-Mosquito3-Fern", Words); err != nil {
		t.Fatal(err)
	}
	if got := values(t, store); len(got) != 1 {
		t.Fatalf("history after starting again: %q", got)
	}
}

func TestAKeyChangeSealsTheHistoryAgain(t *testing.T) {
	v := newOpenVault(t)
	store := newTestStore(t, v)
	if err := store.Record("Orbit-Velvet7-Canyon", Words); err != nil {
		t.Fatal(err)
	}
	rekey, err := v.session.BeginRekey()
	if err != nil {
		t.Fatal(err)
	}
	write, err := store.Reseal(v.id, func(name string, sealed []byte) ([]byte, error) {
		plaintext, err := v.session.OpenOwnerData(name, sealed)
		if err != nil {
			return nil, err
		}
		return rekey.SealOwnerData(name, plaintext)
	})
	if err != nil {
		t.Fatal(err)
	}
	pending, err := v.session.PrepareRekey(rekey)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	write()
	if got := values(t, store); len(got) != 1 || got[0] != "Orbit-Velvet7-Canyon" {
		t.Fatalf("history after the key change: %q", got)
	}
	other := newOpenVault(t)
	if write, err := store.Reseal(other.id, nil); err != nil || write == nil {
		t.Fatalf("resealing a vault without history: %v", err)
	}
}
