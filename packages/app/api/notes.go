package api

import (
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

// NoteInput is one note. A hidden note shows no preview in the list.
type NoteInput struct {
	Label  string `json:"label"`
	Body   string `json:"body"`
	Hidden bool   `json:"hidden"`
	// Tags tell the item apart from others like it.
	Tags []string `json:"tags"`
}

// NoteSummary is a note as the list shows it.
type NoteSummary struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Preview is the first line of the body holding text, empty for a hidden note.
	Preview    string   `json:"preview"`
	Hidden     bool     `json:"hidden"`
	Pinned     bool     `json:"pinned"`
	LastUsedAt int64    `json:"lastUsedAt"`
	Groups     []string `json:"groups"`
	Tags       []string `json:"tags"`
}

// Note is an opened note with its groups.
type Note struct {
	ID     string   `json:"id"`
	Groups []string `json:"groups"`
	NoteInput
}

// ListNotes reports every note from the index.
func (s *Service) ListNotes() ([]NoteSummary, error) {
	entries, usage, err := s.listKind(vault.KindNote)
	if err != nil {
		return nil, err
	}
	result := make([]NoteSummary, len(entries))
	for i, entry := range entries {
		result[i] = NoteSummary{
			ID:         entry.ID.String(),
			Label:      entry.Label,
			Preview:    entry.Detail,
			Hidden:     entry.Note.Hidden,
			Pinned:     entry.Pinned,
			LastUsedAt: usage[entry.ID],
			Groups:     idStrings(entry.Groups),
			Tags:       append([]string{}, entry.Tags...),
		}
	}
	return result, nil
}

// ReadNote opens the note id and records the use.
func (s *Service) ReadNote(id string) (Note, error) {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return Note{}, fail(failureItemUnreadable)
	}
	item, groups, err := openItem(s, parsed, s.vault.ReadSelectedNote)
	if err != nil {
		return Note{}, err
	}
	return Note{ID: item.ID.String(), Groups: groups, NoteInput: fromVaultNote(item.NoteInput)}, nil
}

// CreateNote creates a note in groups and returns its id.
func (s *Service) CreateNote(input NoteInput, groups []string) (string, error) {
	membership, err := s.knownGroups(groups)
	if err != nil {
		return "", err
	}
	id, err := s.vault.CreateNote(input.toVault(), membership)
	if err != nil {
		return "", present(err)
	}
	return id.String(), nil
}

// UpdateNote replaces a note's content and membership as a whole.
func (s *Service) UpdateNote(id string, input NoteInput, groups []string) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	membership, err := s.knownGroups(groups)
	if err != nil {
		return err
	}
	return present(s.vault.EditNote(parsed, input.toVault(), membership))
}

// CopyNote copies a note's body.
func (s *Service) CopyNote(id string) error {
	return s.copyNote(id, clearAfter)
}

func (s *Service) copyNote(id string, schedule func(time.Duration, func())) error {
	return s.copyItemValue(id, schedule, func(parsed vault.ID) (string, error) {
		item, err := readSelected(s, parsed, s.vault.ReadSelectedNote)
		if err != nil {
			return "", err
		}
		return item.Body, nil
	})
}

func (input NoteInput) toVault() vault.NoteInput {
	return vault.NoteInput{Label: input.Label, Body: input.Body, Hidden: input.Hidden, Tags: input.Tags}
}

func fromVaultNote(note vault.NoteInput) NoteInput {
	return NoteInput{Label: note.Label, Body: note.Body, Hidden: note.Hidden, Tags: append([]string{}, note.Tags...)}
}
