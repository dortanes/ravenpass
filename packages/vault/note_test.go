package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func fullNote() NoteInput {
	return NoteInput{Label: "  Diary  ", Body: "\n  \t\n  Dear diary, Ünïcødé  \r\nsecond line\n", Hidden: true}
}

func commitNote(t *testing.T, session *Session, input NoteInput, groups []ID) ID {
	t.Helper()
	pending, id, err := session.PrepareCreateNote(input, groups)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	return id
}

func selectNote(t *testing.T, session *Session, id ID) Note {
	t.Helper()
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	value, err := session.ReadSelectedNote(ticket)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestNoteRoundTripsEveryValue(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
	input := fullNote()
	id := commitNote(t, created.Session, input, nil)
	container, _, err := created.Session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	created.Session.Lock()
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if note := selectNote(t, reopened, id); note.ID != id || !reflect.DeepEqual(note.NoteInput, input) {
		t.Fatalf("note changed across a reopen: %#v", note.NoteInput)
	}
	want := Entry{ID: id, Kind: KindNote, Label: input.Label, Note: NoteFace{Hidden: true}}
	if entry := listedEntry(t, reopened, id); !reflect.DeepEqual(entry, want) {
		t.Fatalf("note entry = %+v", entry)
	}
	if credential := listedEntry(t, reopened, ids[0]); credential.Note != (NoteFace{}) || credential.Seed != (SeedFace{}) {
		t.Fatalf("credential entry carries a summary: %+v", credential)
	}
}

func TestNoteWithoutOptionalValuesHasABareEntry(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	input := NoteInput{Label: "Empty"}
	id := commitNote(t, session, input, nil)
	if note := selectNote(t, session, id); !reflect.DeepEqual(note.NoteInput, input) {
		t.Fatalf("bare note = %#v", note.NoteInput)
	}
	want := Entry{ID: id, Kind: KindNote, Label: "Empty"}
	if entry := listedEntry(t, session, id); !reflect.DeepEqual(entry, want) {
		t.Fatalf("bare note entry = %+v", entry)
	}
}

func TestNotePreviewIsTheFirstLineWithText(t *testing.T) {
	long := strings.Repeat("ж", NotePreviewLength)
	tests := []struct {
		name  string
		input NoteInput
		want  string
	}{
		{"single line", NoteInput{Body: "Milk"}, "Milk"},
		{"leading blank lines", NoteInput{Body: "\n \t\n\r\n  Milk and eggs  \nbread"}, "Milk and eggs"},
		{"carriage returns", NoteInput{Body: "first\r\nsecond"}, "first"},
		{"line at the limit", NoteInput{Body: long}, long},
		{"long line", NoteInput{Body: "  " + long + "tail\nnext"}, long},
		{"blank body", NoteInput{Body: " \n\t\n"}, ""},
		{"empty body", NoteInput{}, ""},
		{"hidden note", NoteInput{Body: "secret", Hidden: true}, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := notePreview(test.input); got != test.want {
				t.Fatalf("preview = %q, want %q", got, test.want)
			}
		})
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	id := commitNote(t, session, NoteInput{Label: "Shopping", Body: "\n\n" + long + "tail"}, nil)
	if entry := listedEntry(t, session, id); entry.Detail != long || entry.Note.Hidden {
		t.Fatalf("shown note entry = %+v", entry)
	}
}

func TestNoteAtItsLimitsRoundTrips(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	input := NoteInput{Label: strings.Repeat("ü", MaxLabelLength), Body: strings.Repeat("ü", MaxNoteBodyLength)}
	id := commitNote(t, session, input, nil)
	if note := selectNote(t, session, id); !reflect.DeepEqual(note.NoteInput, input) {
		t.Fatal("a note at its limits changed on its way through the vault")
	}
}

func TestNoteRefusesEachBoundWithoutWriting(t *testing.T) {
	over := func(limit int) string { return strings.Repeat("ü", limit+1) }
	tests := []struct {
		name   string
		mutate func(*NoteInput)
	}{
		{"empty label", func(input *NoteInput) { input.Label = "" }},
		{"blank label", func(input *NoteInput) { input.Label = " \t\n" }},
		{"label over its limit", func(input *NoteInput) { input.Label = over(MaxLabelLength) }},
		{"label that is not text", func(input *NoteInput) { input.Label = "\xff" }},
		{"body over its limit", func(input *NoteInput) { input.Body = over(MaxNoteBodyLength) }},
		{"body that is not text", func(input *NoteInput) { input.Body = "\xff" }},
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	existing := commitNote(t, session, fullNote(), nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := fullNote()
			test.mutate(&input)
			if _, _, err := session.PrepareCreateNote(input, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("create: got %v, want ErrInvalidInput", err)
			}
			if _, err := session.PrepareEditNote(existing, input, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("edit: got %v, want ErrInvalidInput", err)
			}
		})
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused note changed the vault")
	}
}

func decodeAsNote(record []byte) error {
	_, err := decodeNoteRecord(record)
	return err
}

func TestNoteRecordSchemaMustMatchItsKind(t *testing.T) {
	note := mustEncode(encodeNoteRecord(fullNote()))
	card := mustEncode(encodeCardRecord(fullCard()))
	credential, err := encodeCredentialRecord(CredentialInput{Label: "Mail", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeNoteRecord(note); err != nil {
		t.Fatalf("a valid note record was refused: %v", err)
	}
	body := encodeBytes([]byte("body"))
	tests := []struct {
		name   string
		decode func([]byte) error
		record []byte
		want   error
	}{
		{"card read as a note", decodeAsNote, card, ErrMalformed},
		{"credential read as a note", decodeAsNote, credential, ErrMalformed},
		{"note read as a card", decodeAsCard, note, ErrMalformed},
		{"note read as a credential", decodeAsCredential, note, ErrMalformed},
		{"note read as an identity", decodeAsIdentity, note, ErrMalformed},
		{"unknown schema read as a note", decodeAsNote, encodeArray(encodeUint(15), body, encodeUint(0)), ErrUnsupported},
		{"hidden flag of two", decodeAsNote, encodeArray(encodeUint(recordSchemaNote), body, encodeUint(2)), ErrMalformed},
		{"body that is not text", decodeAsNote, encodeArray(encodeUint(recordSchemaNote), encodeBytes([]byte{0xff}), encodeUint(0)), ErrMalformed},
		{"one field short", decodeAsNote, encodeArray(encodeUint(recordSchemaNote), body), ErrMalformed},
		{"trailing bytes", decodeAsNote, append(append([]byte(nil), note...), 0), ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode(test.record); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestIndexHoldsANoteSummaryOnlyForANote(t *testing.T) {
	records := stubRecords(1)
	digest := sha256.Sum256(encodeBox(records[0]))
	id := ID{5}
	note := uint64(KindNote)
	hidden := encodeArray(encodeUint(1))
	_, _, index, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), currentElement(id, digest, note, hidden)), records)
	entries := index.entries
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].kind != KindNote || !entries[0].note.Hidden {
		t.Fatalf("note entry = %+v", entries[0])
	}
	tests := []struct {
		name    string
		element []byte
	}{
		{"note without a summary", currentElement(id, digest, note, encodeArray())},
		{"summary of two fields", currentElement(id, digest, note, encodeArray(encodeUint(1), encodeUint(0)))},
		{"hidden flag of two", currentElement(id, digest, note, encodeArray(encodeUint(2)))},
		{"seed summary on a note", currentElement(id, digest, note, encodeArray(encodeUint(uint64(SeedPrivateKey)), encodeUint(0), encodeUint(0)))},
		{"note summary on a credential", currentElement(id, digest, uint64(KindCredential), hidden)},
		{"note summary on a card", withField(id, digest, uint64(KindCard), hidden, fieldCard, faceOf(1, "6789", ""))},
		{"note with an expiry", withField(id, digest, note, hidden, fieldExpiresOn, encodeBytes([]byte("2030-01-01")))},
		{"note with a thumbnail", withField(id, digest, note, hidden, fieldThumbnail, encodeBytes([]byte{0xff, 0xd8}))},
		{"note with a site", withField(id, digest, note, hidden, fieldSite, encodeBytes([]byte("example.com")))},
		{"note with an owner", withField(id, digest, note, hidden, fieldOwner, encodeBytes(id[:]))},
		{"note with an email", withField(id, digest, note, hidden, fieldEmail, encodeBytes([]byte("a@example.test")))},
		{"note with a card face", withField(id, digest, note, hidden, fieldCard, faceOf(1, "6789", ""))},
		{"note with a code", codeElement(id, digest, note, hidden, 6, 30)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), test.element), records); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestNoteEntryThatDoesNotMatchItsRecordIsMalformed(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
	session := created.Session
	defer session.Lock()
	commitNote(t, session, NoteInput{Label: "Shown", Body: "Milk\nEggs"}, nil)
	commitNote(t, session, fullNote(), nil)
	raw, head := currentRaw(t, session)
	tests := []struct {
		name   string
		mutate func([]entryMeta)
	}{
		{"other preview", func(entries []entryMeta) { entries[1].detail = "Eggs" }},
		{"preview of a hidden note", func(entries []entryMeta) { entries[2].detail = "Dear diary, Ünïcødé" }},
		{"shown note listed as hidden", func(entries []entryMeta) { entries[1].note.Hidden, entries[1].detail = true, "" }},
		{"hidden note listed as shown", func(entries []entryMeta) { entries[2].note.Hidden = false }},
		{"note record listed as a credential", func(entries []entryMeta) { entries[1].kind, entries[1].note = KindCredential, NoteFace{} }},
		{"credential record listed as a note", func(entries []entryMeta) { entries[0].kind, entries[0].detail = KindNote, "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := append([]entryMeta(nil), session.entries...)
			test.mutate(entries)
			plaintext, err := encodeIndex(head.Revision, session.ancestry, entries, nil, DefaultTrashRetention)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := OpenWithRecovery(withIndex(t, session, raw, plaintext), created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestPrepareEditNoteReplacesContentAndMembershipAndKeepsPin(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family", "Travel")
	session := created.Session
	defer session.Lock()
	credential := commitCredential(t, session, CredentialInput{Label: "Mail", Password: "secret"}, nil)
	id := commitNote(t, session, fullNote(), []ID{groupIDs[0]})
	pending, err := session.PrepareSetPinned(id, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	raw, _ := currentRaw(t, session)
	replacement := NoteInput{Label: "Packing", Body: "Passport\nCharger"}
	pending, err = session.PrepareEditNote(id, replacement, []ID{groupIDs[1]})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	want := Entry{ID: id, Kind: KindNote, Label: "Packing", Detail: "Passport", Pinned: true, Groups: []ID{groupIDs[1]}}
	if entry := listedEntry(t, session, id); !reflect.DeepEqual(entry, want) {
		t.Fatalf("note entry after the edit = %+v", entry)
	}
	if session.entries[1].revision != 2 {
		t.Fatalf("note revision after one edit = %d", session.entries[1].revision)
	}
	if value := selectNote(t, session, id); !reflect.DeepEqual(value.NoteInput, replacement) {
		t.Fatalf("note after the edit = %#v", value.NoteInput)
	}
	next, _ := currentRaw(t, session)
	if !bytes.Equal(encodeBox(next.records[0]), encodeBox(raw.records[0])) {
		t.Fatal("editing a note sealed another record again")
	}
	if value := selectCredential(t, session, credential); value.Password != "secret" {
		t.Fatalf("credential after a note edit = %+v", value.CredentialInput)
	}
	if err := session.VerifyAll(); err != nil {
		t.Fatal(err)
	}
}

func TestItemOperationsActOnNotes(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family", "Travel")
	session := created.Session
	defer session.Lock()
	id := commitNote(t, session, fullNote(), []ID{groupIDs[0]})
	raw, _ := currentRaw(t, session)
	sealed := encodeBox(raw.records[0])
	pending, err := session.PrepareSetPinned(id, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	pending, err = session.PrepareSetGroups(id, []ID{groupIDs[1], groupIDs[0]})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if entry := listedEntry(t, session, id); !entry.Pinned || !reflect.DeepEqual(entry.Groups, sortedIDs(groupIDs...)) || entry.Kind != KindNote {
		t.Fatalf("note entry after pin and groups = %+v", entry)
	}
	if next, _ := currentRaw(t, session); !bytes.Equal(encodeBox(next.records[0]), sealed) {
		t.Fatal("pinning or grouping sealed the note again")
	}
	pending, err = session.PrepareDelete(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if entries, err := session.List(); err != nil || len(entries) != 0 {
		t.Fatalf("entries after deleting the note = %+v, error = %v", entries, err)
	}
}

func TestListingNotesDecryptsNoRecord(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	id := commitNote(t, session, NoteInput{Label: "Shopping", Body: "Milk"}, nil)
	for i := range session.records {
		tampered := append([]byte(nil), session.records[i].ciphertext...)
		tampered[0] ^= 1
		session.records[i].ciphertext = tampered
	}
	if entry := listedEntry(t, session, id); entry.Kind != KindNote || entry.Detail != "Milk" {
		t.Fatalf("note entry over unreadable records = %+v", entry)
	}
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadSelectedNote(ticket); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("reading an unreadable record: %v", err)
	}
}

func TestLockedSessionRefusesNoteOperations(t *testing.T) {
	session, _ := populatedSession(t)
	id := commitNote(t, session, fullNote(), nil)
	session.Lock()
	if _, _, err := session.PrepareCreateNote(fullNote(), nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareCreateNote: %v", err)
	}
	if _, err := session.PrepareEditNote(id, fullNote(), nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareEditNote: %v", err)
	}
	if _, err := session.ReadSelectedNote(Selection{ID: id}); !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("ReadSelectedNote: %v", err)
	}
}
