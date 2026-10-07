package vaultservice

import (
	"crypto/sha1"

	"github.com/dortanes/ravenpass/packages/app/breaches"
	"github.com/dortanes/ravenpass/packages/vault"
)

// PasswordDigests reports the breaches.Digest of every credential's non-empty password outside the trash, so the
// passwords never leave the service.
func (s *Service) PasswordDigests() (map[vault.ID][sha1.Size]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return nil, ErrNotReady
	}
	entries, err := s.session.List()
	if err != nil {
		return nil, err
	}
	digests := make(map[vault.ID][sha1.Size]byte)
	for _, entry := range entries {
		if entry.Kind != vault.KindCredential {
			continue
		}
		credential, err := s.session.ReadCredential(entry.ID)
		if err != nil {
			return nil, err
		}
		if credential.Password != "" {
			digests[entry.ID] = breaches.Digest(credential.Password)
		}
	}
	return digests, nil
}
