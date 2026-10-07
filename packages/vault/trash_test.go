package vault

import (
	"crypto/sha256"
	"errors"
	"slices"
	"testing"
)

const trashedAt = 1_800_000_000_000

func commitTrash(t *testing.T, session *Session, id ID, at uint64) {
	t.Helper()
	pending, err := session.PrepareTrash(id, at)
	commitPending(t, session, pending, err)
}

func trashIDs(t *testing.T, session *Session) []ID {
	t.Helper()
	trashed, err := session.Trash()
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]ID, len(trashed))
	for i, entry := range trashed {
		ids[i] = entry.ID
	}
	return ids
}

func listedIDs(t *testing.T, session *Session) []ID {
	t.Helper()
	entries, err := session.List()
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]ID, len(entries))
	for i, entry := range entries {
		ids[i] = entry.ID
	}
	return ids
}

func reopenedSession(t *testing.T, created Created) *Session {
	t.Helper()
	container, _, err := created.Session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Lock)
	return reopened
}

func TestTrashKeepsAnItemOutOfEveryReadUntilRestored(t *testing.T) {
	created, groups := groupedVault(t, "Work")
	session := created.Session
	defer session.Lock()
	kept := commitCredential(t, session, CredentialInput{Label: "Mail"}, nil)
	trashed := commitCredential(t, session, CredentialInput{Label: "Forum", Login: "alex", Password: "example", Tags: []string{"Old"}}, groups)
	pinned, err := session.PrepareSetPinned(trashed, true)
	commitPending(t, session, pinned, err)
	commitTrash(t, session, trashed, trashedAt)

	if listed := listedIDs(t, session); !slices.Equal(listed, []ID{kept}) {
		t.Fatalf("listed = %x", listed)
	}
	if ids := trashIDs(t, session); !slices.Equal(ids, []ID{trashed}) {
		t.Fatalf("trash = %x", ids)
	}
	if entries, _ := session.Trash(); entries[0].DeletedAt != trashedAt || entries[0].Label != "Forum" {
		t.Fatalf("trashed entry = %+v", entries[0])
	}
	if _, err := session.BeginSelection(trashed); !errors.Is(err, ErrNotFound) {
		t.Fatalf("select: %v", err)
	}
	if _, err := session.ReadCredential(trashed); !errors.Is(err, ErrNotFound) {
		t.Fatalf("read: %v", err)
	}
	label := "Edited"
	if _, err := session.PrepareEdit(trashed, CredentialPatch{Label: &label}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("edit: %v", err)
	}
	if _, err := session.PrepareSetPinned(trashed, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("pin: %v", err)
	}
	if _, err := session.PrepareSetGroups(trashed, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("regroup: %v", err)
	}
	if _, err := session.PrepareTrash(trashed, trashedAt); !errors.Is(err, ErrNotFound) {
		t.Fatalf("trash twice: %v", err)
	}
	if _, err := session.PrepareRestore(kept); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restore a live item: %v", err)
	}

	reopened := reopenedSession(t, created)
	if ids := trashIDs(t, reopened); !slices.Equal(ids, []ID{trashed}) {
		t.Fatalf("trash across a reopen = %x", ids)
	}
	restore, err := reopened.PrepareRestore(trashed)
	commitPending(t, reopened, restore, err)
	entry := listedEntry(t, reopened, trashed)
	if !entry.Pinned || !slices.Equal(entry.Groups, groups) || !slices.Equal(entry.Tags, []string{"Old"}) || entry.DeletedAt != 0 {
		t.Fatalf("restored entry = %+v", entry)
	}
	if credential, err := reopened.ReadCredential(trashed); err != nil || credential.Password != "example" {
		t.Fatalf("restored credential = %+v, %v", credential, err)
	}
}

func TestTrashRefusesAZeroTimeAndAnUnknownItem(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail"})
	defer created.Session.Lock()
	if _, err := created.Session.PrepareTrash(ids[0], 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("a zero time: %v", err)
	}
	if _, err := created.Session.PrepareTrash(ID{0xee}, trashedAt); !errors.Is(err, ErrNotFound) {
		t.Fatalf("an unknown item: %v", err)
	}
	if _, err := created.Session.PrepareRestore(ID{0xee}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restore an unknown item: %v", err)
	}
}

func TestATrashedIdentityLeavesCardLinksAddressesAndFilesUntilRestored(t *testing.T) {
	session, withFiles, bare := filesSession(t)
	owner := commitIdentity(t, session, homeAndWork(), nil)
	card := commitCard(t, session, linkedCard(t, session, owner, 0, "Linked"), nil)
	held, err := session.ScansOf(withFiles)
	if err != nil {
		t.Fatal(err)
	}
	commitTrash(t, session, owner, trashedAt)
	commitTrash(t, session, withFiles, trashedAt)
	if _, err := session.ReadScan(held[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a scan of a trashed identity: %v", err)
	}

	if linked := selectCard(t, session, card).Linked; linked != nil {
		t.Fatalf("a link to a trashed identity resolved: %+v", linked)
	}
	addresses, err := session.IdentityAddresses()
	if err != nil || len(addresses) != 0 {
		t.Fatalf("identity addresses = %+v, %v", addresses, err)
	}
	files, err := session.IdentityFiles()
	if err != nil || len(files) != 1 || files[0].Identity != bare {
		t.Fatalf("identity files = %+v, %v", files, err)
	}
	if _, err := session.ScansOf(withFiles); !errors.Is(err, ErrNotFound) {
		t.Fatalf("scans of a trashed identity: %v", err)
	}

	pending, err := session.PrepareRestore(owner)
	commitPending(t, session, pending, err)
	pending, err = session.PrepareRestore(withFiles)
	commitPending(t, session, pending, err)
	if linked := selectCard(t, session, card).Linked; linked == nil {
		t.Fatal("the restored identity's address does not resolve")
	}
	if scans, err := session.ScansOf(withFiles); err != nil || len(scans) != 3 {
		t.Fatalf("restored scans = %+v, %v", scans, err)
	}
}

func TestPurgeRemovesOnlyItemsTrashedByTheCutoff(t *testing.T) {
	session, withFiles, _ := filesSession(t)
	owner := commitIdentity(t, session, homeAndWork(), nil)
	card := commitCard(t, session, linkedCard(t, session, owner, 0, "Linked"), nil)
	recent := commitNote(t, session, NoteInput{Label: "Recent", Body: "kept"}, nil)
	commitTrash(t, session, owner, trashedAt)
	commitTrash(t, session, withFiles, trashedAt+1)
	commitTrash(t, session, recent, trashedAt+2)
	before := len(session.entries)

	pending, removed, err := session.PreparePurge(trashedAt + 1)
	commitPending(t, session, pending, err)
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	// The identity with files took its three scans along.
	if len(session.entries) != before-2-3 {
		t.Fatalf("entries left = %d, had %d", len(session.entries), before)
	}
	if ids := trashIDs(t, session); !slices.Equal(ids, []ID{recent}) {
		t.Fatalf("trash = %x", ids)
	}
	value := selectCard(t, session, card)
	if value.BillingLink != nil || value.Linked != nil {
		t.Fatalf("the card kept a link to a purged identity: %+v", value.BillingLink)
	}

	pending, removed, err = session.PreparePurge(trashedAt + 1)
	if pending != nil || removed != 0 || err != nil {
		t.Fatalf("a purge with nothing due = %v, %d, %v", pending, removed, err)
	}
	pending, removed, err = session.PreparePurge(EmptyTrash)
	commitPending(t, session, pending, err)
	if removed != 1 || len(trashIDs(t, session)) != 0 {
		t.Fatalf("emptying removed %d, trash = %x", removed, trashIDs(t, session))
	}
}

func TestDeleteRemovesATrashedItemPermanently(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail"})
	session := created.Session
	defer session.Lock()
	commitTrash(t, session, ids[0], trashedAt)
	pending, err := session.PrepareDelete(ids[0])
	commitPending(t, session, pending, err)
	if len(session.entries) != 0 {
		t.Fatalf("entries = %+v", session.entries)
	}
}

func TestTrashCutoffKeepsAnItemForTheWholePeriod(t *testing.T) {
	if cutoff := TrashCutoff(trashedAt, 30); cutoff != trashedAt-30*dayMillis {
		t.Fatalf("cutoff = %d", cutoff)
	}
	if cutoff := TrashCutoff(dayMillis, 30); cutoff != 0 {
		t.Fatalf("a cutoff before the epoch = %d", cutoff)
	}
}

func TestRetentionIsStoredInTheVaultWithinItsBounds(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	defer session.Lock()
	if days, err := session.TrashRetention(); err != nil || days != DefaultTrashRetention {
		t.Fatalf("default retention = %d, %v", days, err)
	}
	for _, days := range []int{0, MaxTrashRetention + 1, -7} {
		if _, err := session.PrepareSetTrashRetention(days); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%d days: %v", days, err)
		}
	}
	pending, err := session.PrepareSetTrashRetention(7)
	commitPending(t, session, pending, err)
	reopened := reopenedSession(t, created)
	if days, err := reopened.TrashRetention(); err != nil || days != 7 {
		t.Fatalf("retention across a reopen = %d, %v", days, err)
	}
}

func TestAVaultWithoutTrashEncodesItsIndexAsBefore(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail"})
	session := created.Session
	defer session.Lock()
	plain, err := encodeIndex(5, nil, session.entries, nil, DefaultTrashRetention)
	if err != nil {
		t.Fatal(err)
	}
	if plain[0] != 0x80|5 || session.entries[0].wire().DeletedAt != 0 {
		t.Fatalf("index head = %x", plain[0])
	}
	if entry, err := encoding.Marshal(session.entries[0].wire()); err != nil || entry[0] != untaggedHead {
		t.Fatalf("a live entry's head = %x, %v", entry[0], err)
	}
	commitTrash(t, session, ids[0], trashedAt)
	entry, err := encoding.Marshal(session.entries[0].wire())
	if err != nil || entry[0] != 0x80|22 {
		t.Fatalf("a trashed entry's head = %x, %v", entry[0], err)
	}
	kept, err := encodeIndex(5, nil, nil, nil, 90)
	if err != nil || kept[0] != 0x80|6 {
		t.Fatalf("an index keeping 90 days = %x, %v", kept, err)
	}
}

func TestIndexReadsTrashOnlyAsTheVaultWritesIt(t *testing.T) {
	records := stubRecords(1)
	digest := sha256.Sum256(encodeBox(records[0]))
	id := ID{5}
	entry := func(kind Kind, tags []byte, deletedAt uint64) []byte {
		return encodeArray(append(currentFields(id, digest, uint64(kind), encodeArray()), tags, encodeUint(deletedAt))...)
	}
	_, _, index, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), entry(KindCredential, encodeArray(), trashedAt)), records)
	if err != nil || index.entries[0].deletedAt != trashedAt {
		t.Fatalf("a trashed entry = %+v, %v", index.entries, err)
	}
	withRetention := func(days uint64) []byte {
		return encodeArray(encodeUint(1), encodeBytes(make([]byte, 32)), encodeArray(), encodeArray(), encodeArray(), encodeUint(days))
	}
	if _, _, index, err := parseIndex(withRetention(90), nil); err != nil || index.retention != 90 {
		t.Fatalf("an index keeping 90 days = %+v, %v", index, err)
	}
	if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), entry(KindCredential, encodeArray(), 0)), records); !errors.Is(err, ErrMalformed) {
		t.Fatalf("a zero deletion time: got %v, want ErrMalformed", err)
	}
	for name, days := range map[string]uint64{
		"a written default":       DefaultTrashRetention,
		"a zero period":           0,
		"a period past the bound": MaxTrashRetention + 1,
	} {
		if _, _, _, err := parseIndex(withRetention(days), nil); !errors.Is(err, ErrMalformed) {
			t.Fatalf("%s: got %v, want ErrMalformed", name, err)
		}
	}
}

func TestRekeyCarriesTheTrashAndItsPeriod(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail"}, CredentialInput{Label: "Forum"})
	session := created.Session
	defer session.Lock()
	commitTrash(t, session, ids[1], trashedAt)
	pending, err := session.PrepareSetTrashRetention(90)
	commitPending(t, session, pending, err)
	phrase, _, after, _ := rekeyed(t, session)
	reopened, err := OpenWithRecovery(after, phrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if trashed := trashIDs(t, reopened); !slices.Equal(trashed, ids[1:]) {
		t.Fatalf("trash after a rekey = %x", trashed)
	}
	if days, err := reopened.TrashRetention(); err != nil || days != 90 {
		t.Fatalf("retention after a rekey = %d, %v", days, err)
	}
}
