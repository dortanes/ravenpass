package vaultservice

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/dortanes/ravenpass/packages/vault"
)

func readTestNote(t *testing.T, service *Service, id vault.ID) vault.Note {
	t.Helper()
	selection, err := service.Select(id)
	if err != nil {
		t.Fatal(err)
	}
	note, err := service.ReadSelectedNote(selection)
	if err != nil {
		t.Fatal(err)
	}
	return note
}

func TestNoteWritesRoundTripAcrossLock(t *testing.T) {
	files := &memoryFiles{}
	service := newTestService(t, files, newMemoryKeys())
	createTestVault(t, service)
	input := vault.NoteInput{Label: "Door codes", Body: "front 4821\nback 9930", Hidden: true}
	id, err := service.CreateNote(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(files.data, []byte("front 4821")) {
		t.Fatal("the note body appeared in the vault file")
	}
	if note := readTestNote(t, service, id); !reflect.DeepEqual(note.NoteInput, input) {
		t.Fatalf("note = %+v", note.NoteInput)
	}
	replacement := vault.NoteInput{Label: "Door", Body: "front 1111"}
	if err := service.EditNote(id, replacement, nil); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatal(err)
	}
	if note := readTestNote(t, service, id); !reflect.DeepEqual(note.NoteInput, replacement) {
		t.Fatalf("note after a lock = %+v", note.NoteInput)
	}
}

func TestNoteWritesRefuseAnotherKind(t *testing.T) {
	files := &memoryFiles{}
	service := newTestService(t, files, newMemoryKeys())
	createTestVault(t, service)
	card, err := service.CreateCard(testCard(), nil)
	if err != nil {
		t.Fatal(err)
	}
	saved := bytes.Clone(files.data)
	if err := service.EditNote(card, vault.NoteInput{Label: "Note"}, nil); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("note edit of a card = %v", err)
	}
	selection, err := service.Select(card)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReadSelectedNote(selection); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("note read of a card = %v", err)
	}
	if !bytes.Equal(files.data, saved) {
		t.Fatal("a refused note write reached the vault file")
	}
}

func TestNoteMethodsNeedAnOpenVault(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	cases := []struct {
		name string
		call func() error
	}{
		{name: "ReadSelectedNote", call: func() error { _, err := service.ReadSelectedNote(vault.Selection{}); return err }},
		{name: "CreateNote", call: func() error { _, err := service.CreateNote(vault.NoteInput{Label: "Note"}, nil); return err }},
		{name: "EditNote", call: func() error { return service.EditNote(vault.ID{1}, vault.NoteInput{Label: "Note"}, nil) }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.call(); !errors.Is(err, ErrNotReady) {
				t.Fatalf("error before a vault is open = %v, want ErrNotReady", err)
			}
		})
	}
}
