package vaultservice

import "github.com/dortanes/ravenpass/packages/vault"

// TrashItem moves an item of any kind to the trash.
func (s *Service) TrashItem(id vault.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return ErrNotReady
	}
	pending, err := s.session.PrepareTrash(id, uint64(s.now().UnixMilli()))
	if err != nil {
		return err
	}
	return s.commit(pending)
}

// RestoreItem brings an item back from the trash.
func (s *Service) RestoreItem(id vault.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return ErrNotReady
	}
	pending, err := s.session.PrepareRestore(id)
	if err != nil {
		return err
	}
	return s.commit(pending)
}

// Trash lists the items in the trash and the days it keeps them, after removing those kept longer.
func (s *Service) Trash() ([]vault.Entry, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return nil, 0, ErrNotReady
	}
	if _, err := s.purge(); err != nil {
		return nil, 0, err
	}
	retention, err := s.session.TrashRetention()
	if err != nil {
		return nil, 0, err
	}
	entries, err := s.session.Trash()
	if err != nil {
		return nil, 0, err
	}
	return entries, retention, nil
}

// PurgeTrash removes the items kept in the trash longer than its period and reports how many there were.
func (s *Service) PurgeTrash() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return 0, ErrNotReady
	}
	return s.purge()
}

// EmptyTrash removes every item in the trash.
func (s *Service) EmptyTrash() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return ErrNotReady
	}
	pending, _, err := s.session.PreparePurge(vault.EmptyTrash)
	if err != nil || pending == nil {
		return err
	}
	return s.commit(pending)
}

// SetTrashRetention sets how many days the trash keeps an item and removes those kept longer. A removal that fails
// once the period is saved is not reported: the next listing of the trash or unlock retries it.
func (s *Service) SetTrashRetention(days int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return ErrNotReady
	}
	pending, err := s.session.PrepareSetTrashRetention(days)
	if err != nil {
		return err
	}
	if err := s.commit(pending); err != nil {
		return err
	}
	_, _ = s.purge()
	return nil
}

// purge removes the items kept in the trash longer than its period. The caller holds s.mu.
func (s *Service) purge() (int, error) {
	retention, err := s.session.TrashRetention()
	if err != nil {
		return 0, err
	}
	pending, removed, err := s.session.PreparePurge(vault.TrashCutoff(uint64(s.now().UnixMilli()), retention))
	if err != nil || pending == nil {
		return 0, err
	}
	if err := s.commit(pending); err != nil {
		return 0, err
	}
	return removed, nil
}
