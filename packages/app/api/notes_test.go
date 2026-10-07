package api

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

func TestNotesListFromTheIndexAndReadAsWritten(t *testing.T) {
	service := newReadyService(t)
	family := createTestGroup(t, service, "Family")
	input := NoteInput{Label: "Wi-Fi", Body: "\n  network ravenpass-home  \npassword on the router"}
	id, err := service.CreateNote(input, []string{family})
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := service.CreateNote(NoteInput{Label: "Diary", Body: "private", Hidden: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateCard(testCardInput(), nil); err != nil {
		t.Fatal(err)
	}
	notes, err := service.ListNotes()
	if err != nil {
		t.Fatal(err)
	}
	want := []NoteSummary{
		{ID: id, Label: "Wi-Fi", Preview: "network ravenpass-home", Groups: []string{family}, Tags: []string{}},
		{ID: hidden, Label: "Diary", Hidden: true, Groups: []string{}, Tags: []string{}},
	}
	if !reflect.DeepEqual(notes, want) {
		t.Fatalf("notes = %+v", notes)
	}
	note, err := service.ReadNote(id)
	if err != nil {
		t.Fatal(err)
	}
	if note.ID != id || !reflect.DeepEqual(note.Groups, []string{family}) || !reflect.DeepEqual(note.NoteInput, withTags(input)) {
		t.Fatalf("note = %+v", note)
	}
	if notes, err := service.ListNotes(); err != nil || notes[0].LastUsedAt <= 0 || notes[1].LastUsedAt != 0 {
		t.Fatalf("notes after a read = %+v, error = %v", notes, err)
	}
	read, err := service.ReadNote(hidden)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(read)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"groups":[]`) || !strings.Contains(string(encoded), `"hidden":true`) {
		t.Fatalf("encoded note = %s", encoded)
	}
	if cards, err := service.ListCards(); err != nil || len(cards) != 1 {
		t.Fatalf("cards beside notes = %+v, error = %v", cards, err)
	}
}

func TestUpdateNoteReplacesItAsAWhole(t *testing.T) {
	service := newReadyService(t)
	id, err := service.CreateNote(NoteInput{Label: "Wi-Fi", Body: "ravenpass-home"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetPinned(id, true); err != nil {
		t.Fatal(err)
	}
	replacement := NoteInput{Label: "Router", Body: "admin panel", Hidden: true}
	if err := service.UpdateNote(id, replacement, nil); err != nil {
		t.Fatal(err)
	}
	if note, err := service.ReadNote(id); err != nil || !reflect.DeepEqual(note.NoteInput, withTags(replacement)) {
		t.Fatalf("note after the update = %+v, error = %v", note, err)
	}
	if notes, err := service.ListNotes(); err != nil || !notes[0].Pinned || notes[0].Preview != "" || !notes[0].Hidden {
		t.Fatalf("notes after the update = %+v, error = %v", notes, err)
	}
	for _, refused := range []NoteInput{{Label: " "}, {Label: "Long", Body: strings.Repeat("x", vault.MaxNoteBodyLength+1)}} {
		assertFailure(t, service.UpdateNote(id, refused, nil), failureInvalidItem)
	}
}

func TestCopyNoteCopiesTheBody(t *testing.T) {
	service := newReadyService(t)
	id, err := service.CreateNote(NoteInput{Label: "Diary", Body: "private", Hidden: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	clipboard := attachPasteboard(service)
	var discard func()
	schedule := func(_ time.Duration, clear func()) { discard = clear }
	if err := service.copyNote(id, schedule); err != nil || clipboard.text != "private" {
		t.Fatalf("copy of a note = %q, %v", clipboard.text, err)
	}
	discard()
	if clipboard.text != "" {
		t.Fatal("the copied note stayed on the clipboard after its lifetime")
	}
	empty, err := service.CreateNote(NoteInput{Label: "Empty"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.copyNote(empty, schedule), failureFieldEmpty)
	notes, err := service.ListNotes()
	if err != nil {
		t.Fatal(err)
	}
	for _, note := range notes {
		if note.ID == empty && note.LastUsedAt != 0 {
			t.Fatal("copying an empty note recorded a use")
		}
	}
	card, err := service.CreateCard(testCardInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.copyNote(card, schedule), failureItemUnreadable)
	assertFailure(t, service.copyCardField(id, "number", schedule), failureItemUnreadable)
	_, err = service.ReadNote(card)
	assertFailure(t, err, failureItemUnreadable)
	assertFailure(t, service.UpdateNote(card, NoteInput{Label: "Note"}, nil), failureItemUnreadable)
	if clipboard.text != "" {
		t.Fatalf("a refused copy reached the clipboard: %q", clipboard.text)
	}
}

func TestNoteLimitsMatchTheVault(t *testing.T) {
	limits, err := (&Service{}).GetNoteLimits()
	if err != nil || limits != (NoteLimits{Label: vault.MaxLabelLength, Body: vault.MaxNoteBodyLength}) {
		t.Fatalf("limits = %+v, error = %v", limits, err)
	}
}

// withTags is a note as read back: its tags never null.
func withTags(note NoteInput) NoteInput {
	note.Tags = append([]string{}, note.Tags...)
	return note
}
