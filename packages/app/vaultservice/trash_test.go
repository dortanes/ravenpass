package vaultservice

import (
	"errors"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

// settableClock reads the time it was last set to.
type settableClock struct{ at time.Time }

func (c *settableClock) now() time.Time { return c.at }

func TestTrashKeepsAnItemForTheRetentionPeriod(t *testing.T) {
	service, _ := readyVault(t)
	clock := &settableClock{at: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}
	service.now = clock.now
	id, err := service.CreateNote(vault.NoteInput{Label: "Door", Body: "4821"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.TrashItem(id); err != nil {
		t.Fatal(err)
	}
	entries, retention, err := service.Trash()
	if err != nil || len(entries) != 1 || retention != vault.DefaultTrashRetention {
		t.Fatalf("trash = %+v, %d days, error = %v", entries, retention, err)
	}
	if entries[0].DeletedAt != uint64(clock.at.UnixMilli()) {
		t.Fatalf("deleted at %d", entries[0].DeletedAt)
	}

	clock.at = clock.at.Add(30 * 24 * time.Hour)
	if removed, err := service.PurgeTrash(); err != nil || removed != 1 {
		t.Fatalf("purge on the last day = %d, %v", removed, err)
	}
	if entries, _, err := service.Trash(); err != nil || len(entries) != 0 {
		t.Fatalf("trash after the period = %+v, %v", entries, err)
	}
}

func TestShorteningTheRetentionPurgesAtOnce(t *testing.T) {
	service, _ := readyVault(t)
	clock := &settableClock{at: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}
	service.now = clock.now
	id, err := service.CreateNote(vault.NoteInput{Label: "Door"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.TrashItem(id); err != nil {
		t.Fatal(err)
	}
	clock.at = clock.at.Add(10 * 24 * time.Hour)
	if err := service.SetTrashRetention(90); err != nil {
		t.Fatal(err)
	}
	if entries, days, err := service.Trash(); err != nil || len(entries) != 1 || days != 90 {
		t.Fatalf("trash kept 90 days = %+v, %d, %v", entries, days, err)
	}
	if err := service.SetTrashRetention(7); err != nil {
		t.Fatal(err)
	}
	if entries, days, err := service.Trash(); err != nil || len(entries) != 0 || days != 7 {
		t.Fatalf("trash kept 7 days = %+v, %d, %v", entries, days, err)
	}
	if err := service.SetTrashRetention(0); !errors.Is(err, vault.ErrInvalidInput) {
		t.Fatalf("a zero period = %v", err)
	}
}

func TestRestoreAndEmptyTheTrash(t *testing.T) {
	service, _ := readyVault(t)
	kept, err := service.CreateNote(vault.NoteInput{Label: "Kept"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	gone, err := service.CreateCard(testCard(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []vault.ID{kept, gone} {
		if err := service.TrashItem(id); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.RestoreItem(kept); err != nil {
		t.Fatal(err)
	}
	if err := service.EmptyTrash(); err != nil {
		t.Fatal(err)
	}
	if err := service.EmptyTrash(); err != nil {
		t.Fatalf("emptying an empty trash = %v", err)
	}
	listed, err := service.List()
	if err != nil || len(listed) != 1 || listed[0].ID != kept {
		t.Fatalf("listed = %+v, %v", listed, err)
	}
	if err := service.RestoreItem(gone); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("restore an emptied item = %v", err)
	}
}

func TestTrashNeedsAnOpenVault(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	for name, call := range map[string]func() error{
		"trash":     func() error { return service.TrashItem(vault.ID{1}) },
		"restore":   func() error { return service.RestoreItem(vault.ID{1}) },
		"empty":     service.EmptyTrash,
		"retention": func() error { return service.SetTrashRetention(7) },
		"list":      func() error { _, _, err := service.Trash(); return err },
		"purge":     func() error { _, err := service.PurgeTrash(); return err },
		"merge":     func() error { return service.MergeCredentials(vault.ID{1}, vault.ID{2}, vault.CredentialPatch{}) },
	} {
		if err := call(); !errors.Is(err, ErrNotReady) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestMergeCredentialsTrashesTheOther(t *testing.T) {
	service, _ := readyVault(t)
	into, err := service.CreateCredential(vault.CredentialInput{Label: "Mail", Password: "current"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	from, err := service.CreateCredential(vault.CredentialInput{Label: "Mail old", Login: "alex"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	login := "alex"
	if err := service.MergeCredentials(into, from, vault.CredentialPatch{Login: &login}); err != nil {
		t.Fatal(err)
	}
	entries, _, err := service.Trash()
	if err != nil || len(entries) != 1 || entries[0].ID != from {
		t.Fatalf("trash = %+v, %v", entries, err)
	}
	listed, err := service.List()
	if err != nil || len(listed) != 1 || listed[0].Detail != "alex" {
		t.Fatalf("listed = %+v, %v", listed, err)
	}
}
