package vault

import (
	"bytes"
	"errors"
	"slices"
	"testing"
)

func TestMergeKeepsTheChosenValuesAndEveryPasskeyAndTrashesTheOther(t *testing.T) {
	created, groups := groupedVault(t, "Work", "Home")
	session := created.Session
	defer session.Lock()
	shared, onlyOther := testPasskey(t, 1), testPasskey(t, 2)
	into := commitCredential(t, session, CredentialInput{Label: "Mail", Login: "alex", Password: "current", Passkeys: []Passkey{shared}}, groups[:1])
	from := commitCredential(t, session, CredentialInput{Label: "Mail (old)", Password: "old", TOTP: "JBSWY3DPEHPK3PXP", Passkeys: []Passkey{shared, onlyOther}}, groups[1:])

	totp, membership := "JBSWY3DPEHPK3PXP", sortedIDs(groups...)
	pending, err := session.PrepareMerge(into, from, CredentialPatch{TOTP: &totp, Groups: &membership}, trashedAt)
	commitPending(t, session, pending, err)

	merged, err := session.ReadCredential(into)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Password != "current" || merged.Login != "alex" || merged.TOTP == "" {
		t.Fatalf("merged = %+v", merged.CredentialInput)
	}
	if len(merged.Passkeys) != 2 || !bytes.Equal(merged.Passkeys[1].CredentialID, onlyOther.CredentialID) {
		t.Fatalf("merged passkeys = %+v", merged.Passkeys)
	}
	if key, err := session.ReadPasskey(into, onlyOther.CredentialID); err != nil || !bytes.Equal(key.PrivateKey, onlyOther.PrivateKey) {
		t.Fatalf("the moved passkey's key = %v", err)
	}
	if entry := listedEntry(t, session, into); !slices.Equal(entry.Groups, membership) {
		t.Fatalf("merged groups = %x", entry.Groups)
	}
	if trashed := trashIDs(t, session); !slices.Equal(trashed, []ID{from}) {
		t.Fatalf("trash = %x", trashed)
	}

	restore, err := session.PrepareRestore(from)
	commitPending(t, session, restore, err)
	restored, err := session.ReadCredential(from)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Passkeys) != 0 || restored.TOTP == "" || restored.Password != "old" {
		t.Fatalf("restored = %+v", restored.CredentialInput)
	}
	if entry := listedEntry(t, session, from); len(entry.Passkeys) != 0 || entry.Label != "Mail (old)" {
		t.Fatalf("restored entry = %+v", entry)
	}
}

func TestARefusedMergeChangesNothing(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	defer session.Lock()
	full := make([]Passkey, MaxCredentialPasskeys)
	for i := range full {
		full[i] = testPasskey(t, byte(10+i))
	}
	into := commitCredential(t, session, CredentialInput{Label: "Full", Passkeys: full}, nil)
	from := commitCredential(t, session, CredentialInput{Label: "One more", Passkeys: []Passkey{testPasskey(t, 1)}}, nil)
	trashed := commitCredential(t, session, CredentialInput{Label: "Trashed"}, nil)
	commitTrash(t, session, trashed, trashedAt)
	note := commitNote(t, session, NoteInput{Label: "Note", Body: "text"}, nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	blank := ""
	for name, test := range map[string]struct {
		into, from ID
		patch      CredentialPatch
		at         uint64
		want       error
	}{
		"too many passkeys":   {into, from, CredentialPatch{}, trashedAt, ErrPasskeysFull},
		"itself":              {from, from, CredentialPatch{}, trashedAt, ErrInvalidInput},
		"a zero time":         {from, into, CredentialPatch{}, 0, ErrInvalidInput},
		"a trashed password":  {from, trashed, CredentialPatch{}, trashedAt, ErrNotFound},
		"into a trashed one":  {trashed, from, CredentialPatch{}, trashedAt, ErrNotFound},
		"a note":              {from, note, CredentialPatch{}, trashedAt, ErrNotFound},
		"an invalid patch":    {from, into, CredentialPatch{Label: &blank}, trashedAt, ErrInvalidInput},
		"an unknown password": {from, ID{0xee}, CredentialPatch{}, trashedAt, ErrNotFound},
	} {
		if _, err := session.PrepareMerge(test.into, test.from, test.patch, test.at); !errors.Is(err, test.want) {
			t.Fatalf("%s: got %v, want %v", name, err, test.want)
		}
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused merge changed the vault")
	}
}
