package vaultservice

import "github.com/dortanes/ravenpass/packages/vault"

// MergeCredentials saves patch over credential into with every passkey of from, and moves from to the trash, in one
// save.
func (s *Service) MergeCredentials(into, from vault.ID, patch vault.CredentialPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return ErrNotReady
	}
	pending, err := s.session.PrepareMerge(into, from, patch, uint64(s.now().UnixMilli()))
	if err != nil {
		return err
	}
	return s.commit(pending)
}
