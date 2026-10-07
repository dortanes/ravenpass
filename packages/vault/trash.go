package vault

import "math"

const (
	// DefaultTrashRetention is how many days the trash keeps an item unless the vault records another period.
	DefaultTrashRetention = 30
	// MinTrashRetention and MaxTrashRetention bound the period, in days.
	MinTrashRetention = 1
	MaxTrashRetention = 3650
	// EmptyTrash is the cutoff with which PreparePurge empties the whole trash.
	EmptyTrash = math.MaxUint64
	// dayMillis is one day in Unix milliseconds.
	dayMillis = 24 * 60 * 60 * 1000
)

// TrashCutoff is the latest deletion time, in Unix milliseconds, that an item kept retention days is due for removal by
// at now; zero when nothing can be due yet.
func TrashCutoff(now uint64, retention int) uint64 {
	kept := uint64(retention) * dayMillis
	if now <= kept {
		return 0
	}
	return now - kept
}

// Trash lists the items in the trash without decrypting a record.
func (s *Session) Trash() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	var result []Entry
	for _, entry := range s.entries {
		if entry.trashed() {
			result = append(result, entry.listed())
		}
	}
	return result, nil
}

// TrashRetention reports how many days the trash keeps an item.
func (s *Session) TrashRetention() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return 0, ErrLocked
	}
	return s.retention, nil
}

// PrepareSetTrashRetention prepares the trash keeping items for days, from MinTrashRetention to MaxTrashRetention.
func (s *Session) PrepareSetTrashRetention(days int) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	if days < MinTrashRetention || days > MaxTrashRetention {
		return nil, ErrInvalidInput
	}
	return s.prepareSealed(append([]entryMeta(nil), s.entries...), append([]sealedBox(nil), s.records...), append([]Group(nil), s.groups...), days, s.indexKey, s.recovery)
}

// PrepareTrash prepares moving an item outside the trash into it at the Unix millisecond at, which is not zero. Its
// record, scans and card links stay, sealing no record again.
func (s *Session) PrepareTrash(id ID, at uint64) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	if at == 0 {
		return nil, ErrInvalidInput
	}
	index := s.findItem(id)
	if index < 0 {
		return nil, ErrNotFound
	}
	entries := append([]entryMeta(nil), s.entries...)
	entries[index].deletedAt = at
	return s.prepare(entries, append([]sealedBox(nil), s.records...))
}

// PrepareRestore prepares bringing an item back from the trash with everything it held there.
func (s *Session) PrepareRestore(id ID) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	index := s.find(id)
	if index < 0 || !s.entries[index].trashed() {
		return nil, ErrNotFound
	}
	entries := append([]entryMeta(nil), s.entries...)
	entries[index].deletedAt = 0
	return s.prepare(entries, append([]sealedBox(nil), s.records...))
}

// PreparePurge prepares the permanent removal of every item moved to the trash at or before cutoff, as PrepareDelete
// removes one, and returns how many there are; with none it prepares nothing. math.MaxUint64 empties the trash.
func (s *Session) PreparePurge(cutoff uint64) (*Pending, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, 0, err
	}
	removed := make(map[ID]struct{})
	for _, entry := range s.entries {
		if entry.trashed() && entry.deletedAt <= cutoff {
			removed[entry.id] = struct{}{}
		}
	}
	if len(removed) == 0 {
		return nil, 0, nil
	}
	pending, err := s.prepareRemoval(removed)
	if err != nil {
		return nil, 0, err
	}
	return pending, len(removed), nil
}
