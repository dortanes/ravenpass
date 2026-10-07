package api

import (
	"cmp"
	"slices"

	"github.com/dortanes/ravenpass/packages/vault"
)

// TrashedItem is an item in the trash as the trash lists it.
type TrashedItem struct {
	ID string `json:"id"`
	// Kind is the item's kind as kindNames names it.
	Kind  string `json:"kind"`
	Label string `json:"label"`
	// Detail is what the item's list row shows under its name: a login, an email, a bank, a note preview or a wallet.
	Detail string `json:"detail"`
	// DeletedAt is when the item moved to the trash, in Unix milliseconds.
	DeletedAt int64 `json:"deletedAt"`
}

// Trash is the items in the trash, latest deleted first, and how many days the trash keeps one.
type Trash struct {
	Items         []TrashedItem `json:"items"`
	RetentionDays int           `json:"retentionDays"`
}

// ListTrash lists the trash after removing the items kept longer than its period.
func (s *Service) ListTrash() (Trash, error) {
	entries, retention, err := s.vault.Trash()
	if err != nil {
		return Trash{}, present(err)
	}
	items := make([]TrashedItem, len(entries))
	for i, entry := range entries {
		items[i] = TrashedItem{
			ID: entry.ID.String(), Kind: kindNames[entry.Kind], Label: entry.Label,
			Detail: cmp.Or(entry.Detail, entry.Email), DeletedAt: int64(entry.DeletedAt),
		}
	}
	slices.SortStableFunc(items, func(a, b TrashedItem) int { return cmp.Compare(b.DeletedAt, a.DeletedAt) })
	return Trash{Items: items, RetentionDays: retention}, nil
}

// TrashItem moves an item of any kind to the trash.
func (s *Service) TrashItem(id string) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	return present(s.vault.TrashItem(parsed))
}

// RestoreItem brings an item back from the trash.
func (s *Service) RestoreItem(id string) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	return present(s.vault.RestoreItem(parsed))
}

// EmptyTrash deletes every item in the trash permanently.
func (s *Service) EmptyTrash() error {
	return present(s.vault.EmptyTrash())
}

// SetTrashRetention sets how many days the trash keeps an item, from vault.MinTrashRetention to
// vault.MaxTrashRetention, and deletes those kept longer.
func (s *Service) SetTrashRetention(days int) error {
	return present(s.vault.SetTrashRetention(days))
}
