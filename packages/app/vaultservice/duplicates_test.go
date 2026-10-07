package vaultservice

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/dortanes/ravenpass/packages/vault"
)

// read reads the item id of service through a fresh selection.
func read[T any](t *testing.T, service *Service, id vault.ID, readSelected func(vault.Selection) (T, error)) T {
	t.Helper()
	ticket, err := service.Select(id)
	if err != nil {
		t.Fatal(err)
	}
	item, err := readSelected(ticket)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

// groupsOf is the groups the list shows for id.
func groupsOf(t *testing.T, service *Service, id vault.ID) []vault.ID {
	t.Helper()
	entries, err := service.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == id {
			return entry.Groups
		}
	}
	t.Fatalf("the list lacks %v", id)
	return nil
}

func TestACredentialCopyKeepsItsFieldsAndGroupsButNotItsPasskeys(t *testing.T) {
	service, _ := readyVault(t)
	group, err := service.CreateGroup("Work")
	if err != nil {
		t.Fatal(err)
	}
	original := vault.CredentialInput{
		Label: "Example", Websites: []string{"https://example.com"}, Login: "alex", Email: "alex@example.com",
		Password: "example-1-DO-NOT-USE", Notes: "notes", TOTP: standardSecret, Apps: []vault.App{mailLink},
		Passkeys: []vault.Passkey{testPasskey(t, "example.com", "alex", true, 0)},
	}
	id, err := service.CreateCredential(original, []vault.ID{group})
	if err != nil {
		t.Fatal(err)
	}
	copied, err := service.Duplicate(id, "Example (copy)")
	if err != nil {
		t.Fatal(err)
	}
	got := read(t, service, copied, service.ReadSelected)
	want := read(t, service, id, service.ReadSelected).CredentialInput
	if len(want.Passkeys) != 1 {
		t.Fatal("the original lost its passkey")
	}
	want.Label, want.Passkeys = "Example (copy)", nil
	if copied == id || !reflect.DeepEqual(got.CredentialInput, want) {
		t.Fatalf("the copy holds %+v, want %+v", got.CredentialInput, want)
	}
	if !slices.Equal(groupsOf(t, service, copied), []vault.ID{group}) {
		t.Fatal("the copy left its groups")
	}
}

func TestEveryKindOfItemIsCopied(t *testing.T) {
	service, _ := readyVault(t)
	card, err := service.CreateCard(testCard(), nil)
	if err != nil {
		t.Fatal(err)
	}
	note, err := service.CreateNote(vault.NoteInput{Label: "Door", Body: "code 0000", Hidden: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := service.CreateSeed(testPhraseSeed(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]func() bool{
		"card": func() bool {
			copied, err := service.Duplicate(card, "Card copy")
			if err != nil {
				t.Fatal(err)
			}
			want := read(t, service, card, service.ReadSelectedCard).CardInput
			want.Label = "Card copy"
			return reflect.DeepEqual(read(t, service, copied, service.ReadSelectedCard).CardInput, want)
		},
		"note": func() bool {
			copied, err := service.Duplicate(note, "Note copy")
			if err != nil {
				t.Fatal(err)
			}
			got := read(t, service, copied, service.ReadSelectedNote).NoteInput
			return reflect.DeepEqual(got, vault.NoteInput{Label: "Note copy", Body: "code 0000", Hidden: true})
		},
		"seed": func() bool {
			copied, err := service.Duplicate(seed, "Seed copy")
			if err != nil {
				t.Fatal(err)
			}
			want := testPhraseSeed()
			want.Label = "Seed copy"
			return reflect.DeepEqual(read(t, service, copied, service.ReadSelectedSeed).SeedInput, want)
		},
	} {
		if !check() {
			t.Errorf("the %s copy differs from its original", name)
		}
	}
}

func TestAnIdentityCopyHoldsScansOfItsOwn(t *testing.T) {
	service, _ := readyVault(t)
	id, err := service.CreateIdentity(identityWithScan(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := service.Duplicate(id, "Identity copy")
	if err != nil {
		t.Fatal(err)
	}
	originalScans, err := service.ScansOf(id)
	if err != nil {
		t.Fatal(err)
	}
	copiedScans, err := service.ScansOf(copied)
	if err != nil || len(copiedScans) != 1 || copiedScans[0].ID == originalScans[0].ID {
		t.Fatalf("the copy's scans = %+v, error = %v", copiedScans, err)
	}
	scan, err := service.ReadScan(copiedScans[0].ID)
	if err != nil || !bytes.Equal(scan.Content, testPDF) || scan.Name != "passport.pdf" {
		t.Fatalf("the copied scan = %+v, error = %v", scan.ScanSummary, err)
	}
	if err := service.DeleteItem(id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReadScan(copiedScans[0].ID); err != nil {
		t.Fatalf("deleting the original took the copy's scan: %v", err)
	}
}

func TestDuplicatingNeedsAnOpenVaultAndAListedItem(t *testing.T) {
	service, _ := readyVault(t)
	if _, err := service.Duplicate(vault.ID{0x01}, "Missing"); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("an unknown item: got %v, want ErrNotFound", err)
	}
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Example"})
	service.Lock()
	if _, err := service.Duplicate(id, "Example (copy)"); !errors.Is(err, ErrNotReady) {
		t.Fatalf("a locked vault: got %v, want ErrNotReady", err)
	}
}
