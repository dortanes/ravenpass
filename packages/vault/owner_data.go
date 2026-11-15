package vault

// Owner data is what the owner keeps on one device beside the vault, such as passwords the generator handed out. It is
// sealed like a device cache but under its own purpose and additional data tag, so neither opens as the other, and a
// rekey seals it again for the new key.

const ownerDataTag = 7

// SealOwnerData seals plaintext as the owner data name of this vault.
func (s *Session) SealOwnerData(name string, plaintext []byte) ([]byte, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	return sealForDevice(&s.dataKey, s.vaultID, ownerDataPurpose(name), ownerDataAAD(s.vaultID, name), plaintext)
}

// OpenOwnerData opens what SealOwnerData, or a committed rekey's SealOwnerData, sealed for name in this vault.
func (s *Session) OpenOwnerData(name string, sealed []byte) ([]byte, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	return openForDevice(&s.dataKey, s.vaultID, ownerDataPurpose(name), ownerDataAAD(s.vaultID, name), sealed)
}

// SealOwnerData seals plaintext as the owner data name under the new vault key, which opens it once the rekey commits.
func (r *Rekey) SealOwnerData(name string, plaintext []byte) ([]byte, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.discarded {
		return nil, ErrLocked
	}
	return sealForDevice(&r.key.data, r.vaultID, ownerDataPurpose(name), ownerDataAAD(r.vaultID, name), plaintext)
}

func ownerDataPurpose(name string) string {
	return "owner-data/" + name
}

func ownerDataAAD(vaultID ID, name string) []byte {
	hash := headerHash(vaultID)
	aad := append([]byte{ownerDataTag}, hash[:]...)
	return append(aad, name...)
}
