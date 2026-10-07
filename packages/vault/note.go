package vault

import (
	"slices"
	"strings"
)

// Note limits are counted in characters.
const (
	MaxNoteBodyLength = 65_536
	NotePreviewLength = 80
)

// NoteFace is what the list shows of a note beside its preview without decrypting it.
type NoteFace struct{ Hidden bool }

// NoteInput is one note. A hidden note shows no preview in the list.
type NoteInput struct {
	Label  string
	Body   string
	Hidden bool
	Tags   []string
}

// Note is a note as read.
type Note struct {
	ID ID
	NoteInput
}

// acceptNote returns a note as the vault stores it, refusing any value out of bounds.
func acceptNote(input NoteInput) (NoteInput, error) {
	if !fits(input.Label, MaxLabelLength) || strings.TrimSpace(input.Label) == "" || !fits(input.Body, MaxNoteBodyLength) {
		return NoteInput{}, ErrInvalidInput
	}
	tags, err := AcceptTags(input.Tags)
	if err != nil {
		return NoteInput{}, err
	}
	input.Tags = tags
	return input, nil
}

// notePreview is the body's first non-blank line, trimmed and cut to NotePreviewLength characters, or empty for a hidden note.
func notePreview(input NoteInput) string {
	if input.Hidden {
		return ""
	}
	for line := range strings.Lines(input.Body) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if runes := []rune(trimmed); len(runes) > NotePreviewLength {
			return string(runes[:NotePreviewLength])
		}
		return trimmed
	}
	return ""
}

// noteEntry is what the index shows of an accepted note.
func noteEntry(input NoteInput, groups []ID) entryMeta {
	return entryMeta{kind: KindNote, label: input.Label, detail: notePreview(input), note: NoteFace{Hidden: input.Hidden}, groups: groups, tags: input.Tags}
}

// A note record is [10, body, hidden]; the label lives in the index.
const recordSchemaNote = 10

type noteRecord struct {
	_      struct{} `cbor:",toarray"`
	Schema uint64
	Body   string
	Hidden flag
}

func encodeNoteRecord(input NoteInput) ([]byte, error) {
	return sealableRecord(marshal(noteRecord{Schema: recordSchemaNote, Body: input.Body, Hidden: flag(input.Hidden)}, 0))
}

func decodeNoteRecord(plaintext []byte) (NoteInput, error) {
	if err := readSchema(plaintext, recordSchemaNote); err != nil {
		return NoteInput{}, err
	}
	var record noteRecord
	if err := unmarshal(plaintext, &record); err != nil {
		return NoteInput{}, err
	}
	if !validText(record.Body) {
		return NoteInput{}, ErrMalformed
	}
	return NoteInput{Body: record.Body, Hidden: bool(record.Hidden)}, nil
}

// verifyNoteEntry checks that a note entry shows what its record holds.
func verifyNoteEntry(entry entryMeta, plaintext []byte) error {
	input, err := decodeNoteRecord(plaintext)
	if err != nil {
		return err
	}
	input.Label = entry.label
	accepted, err := acceptNote(input)
	if err != nil {
		return ErrMalformed
	}
	want := noteEntry(accepted, entry.groups)
	if entry.detail != want.detail || entry.note != want.note {
		return ErrMalformed
	}
	return nil
}

// ReadSelectedNote decrypts the selected note.
func (s *Session) ReadSelectedNote(ticket Selection) (Note, error) {
	var input NoteInput
	entry, err := s.readSelected(ticket, KindNote, func(plaintext []byte) (err error) {
		input, err = decodeNoteRecord(plaintext)
		return err
	})
	if err != nil {
		return Note{}, err
	}
	input.Label = entry.label
	input.Tags = slices.Clone(entry.tags)
	return Note{ID: entry.id, NoteInput: input}, nil
}

// PrepareCreateNote prepares a new note in groups.
func (s *Session) PrepareCreateNote(input NoteInput, groups []ID) (*Pending, ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, ID{}, err
	}
	input, err := acceptNote(input)
	if err != nil {
		return nil, ID{}, err
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, ID{}, err
	}
	plaintext, err := encodeNoteRecord(input)
	if err != nil {
		return nil, ID{}, err
	}
	defer clear(plaintext)
	return s.prepareAdd(noteEntry(input, membership), plaintext)
}

// PrepareEditNote replaces a note's content and membership, keeping its pin.
func (s *Session) PrepareEditNote(id ID, input NoteInput, groups []ID) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	index := s.findKind(id, KindNote)
	if index < 0 {
		return nil, ErrNotFound
	}
	input, err := acceptNote(input)
	if err != nil {
		return nil, err
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, err
	}
	plaintext, err := encodeNoteRecord(input)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	return s.prepareReplace(index, noteEntry(input, membership), plaintext)
}
