package vault

import (
	"crypto/sha256"
	"crypto/subtle"
	"sync"
)

// vaultKey is a vault key and the recovery box that seals it for its recovery phrase.
type vaultKey struct {
	data     [32]byte
	recovery sealedBox
}

// newVaultKey draws a vault key for vaultID and seals it for a new recovery phrase; the caller clears data.
func newVaultKey(vaultID ID) (vaultKey, string, error) {
	var key vaultKey
	if err := randomBytes(key.data[:]); err != nil {
		return vaultKey{}, "", err
	}
	entropy := make([]byte, 32)
	defer clear(entropy)
	if err := randomBytes(entropy); err != nil {
		clear(key.data[:])
		return vaultKey{}, "", err
	}
	phrase, err := mnemonicFromEntropy(entropy)
	if err != nil {
		clear(key.data[:])
		return vaultKey{}, "", err
	}
	recoveryKey, err := deriveKey(entropy, vaultID, "recovery-wrap")
	if err != nil {
		clear(key.data[:])
		return vaultKey{}, "", err
	}
	key.recovery, err = seal(recoveryKey, key.data[:], recoveryAAD(vaultID))
	clear(recoveryKey[:])
	if err != nil {
		clear(key.data[:])
		return vaultKey{}, "", err
	}
	return key, phrase, nil
}

// openRecovery unwraps the vault key recovery seals for phrase.
func openRecovery(vaultID ID, recovery sealedBox, phrase string) ([32]byte, error) {
	entropy, err := entropyFromMnemonic(phrase)
	if err != nil {
		return [32]byte{}, err
	}
	defer clear(entropy)
	key, err := deriveKey(entropy, vaultID, "recovery-wrap")
	if err != nil {
		return [32]byte{}, err
	}
	plaintext, err := openBox(key, recovery, recoveryAAD(vaultID))
	clear(key[:])
	if err != nil {
		return [32]byte{}, err
	}
	defer clear(plaintext)
	if len(plaintext) != 32 {
		return [32]byte{}, ErrAuthentication
	}
	var dataKey [32]byte
	copy(dataKey[:], plaintext)
	return dataKey, nil
}

// VerifyRecoveryPhrase fails with ErrInvalidPhrase or ErrAuthentication unless phrase opens the vault key the session holds.
func (s *Session) VerifyRecoveryPhrase(phrase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return ErrLocked
	}
	dataKey, err := openRecovery(s.vaultID, s.recovery, phrase)
	defer clear(dataKey[:])
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(dataKey[:], s.dataKey[:]) != 1 {
		return ErrAuthentication
	}
	return nil
}

// contentKeys are the keys a vault key yields for the index and the records, with the key and its recovery box.
type contentKeys struct {
	data     [32]byte
	index    [32]byte
	record   [32]byte
	recovery sealedBox
}

func (k *contentKeys) clear() {
	clear(k.data[:])
	clear(k.index[:])
	clear(k.record[:])
}

// deriveContentKeys yields the index and record keys of a vault key.
func deriveContentKeys(dataKey [32]byte, vaultID ID) (index, record [32]byte, err error) {
	index, err = deriveKey(dataKey[:], vaultID, "index")
	if err != nil {
		return [32]byte{}, [32]byte{}, err
	}
	record, err = deriveKey(dataKey[:], vaultID, "record")
	if err != nil {
		clear(index[:])
		return [32]byte{}, [32]byte{}, err
	}
	return index, record, nil
}

// Rekey is a new vault key and recovery phrase for an open vault; the vault changes only once PrepareRekey's save commits.
// Every method is safe for concurrent use.
type Rekey struct {
	mu        sync.Mutex
	vaultID   ID
	key       vaultKey
	phrase    string
	discarded bool
}

// BeginRekey draws a new vault key and recovery phrase for the session's vault.
func (s *Session) BeginRekey() (*Rekey, error) {
	s.mu.Lock()
	vaultID, locked := s.vaultID, s.locked
	s.mu.Unlock()
	if locked {
		return nil, ErrLocked
	}
	key, phrase, err := newVaultKey(vaultID)
	if err != nil {
		return nil, err
	}
	return &Rekey{vaultID: vaultID, key: key, phrase: phrase}, nil
}

// RecoveryPhrase is the phrase that opens the vault once the rekey commits; empty once discarded.
func (r *Rekey) RecoveryPhrase() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.discarded {
		return ""
	}
	return r.phrase
}

// VerifyPhrase fails with ErrInvalidPhrase or ErrAuthentication unless phrase opens the new vault key.
func (r *Rekey) VerifyPhrase(phrase string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.discarded {
		return ErrLocked
	}
	dataKey, err := openRecovery(r.vaultID, r.key.recovery, phrase)
	clear(dataKey[:])
	return err
}

// WrapDeviceKey seals the new vault key for deviceKey into an envelope OpenWithDevice opens once the rekey commits.
func (r *Rekey) WrapDeviceKey(deviceKey []byte) ([]byte, error) {
	if len(deviceKey) != 32 {
		return nil, ErrInvalidInput
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.discarded {
		return nil, ErrLocked
	}
	return sealDeviceEnvelope(deviceKey, r.vaultID, &r.key.data, r.key.recovery)
}

// KeyIdentity names the new vault key as Session.KeyIdentity names it once the rekey commits.
func (r *Rekey) KeyIdentity() [32]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return keyIdentity(r.key.recovery)
}

// Discard clears the new vault key; a discarded rekey neither wraps nor prepares.
func (r *Rekey) Discard() {
	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.key.data[:])
	r.phrase = ""
	r.discarded = true
}

// PrepareRekey prepares the vault sealed under rekey's key: a new index and every record sealed again under its own ID
// and revision. Commit switches the session to the new key; device envelopes for the old one then fail with ErrKeyReplaced.
func (s *Session) PrepareRekey(rekey *Rekey) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rekey.mu.Lock()
	defer rekey.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	if rekey.discarded {
		return nil, ErrLocked
	}
	if rekey.vaultID != s.vaultID {
		return nil, ErrInvalidInput
	}
	keys := &contentKeys{data: rekey.key.data, recovery: rekey.key.recovery}
	var err error
	if keys.index, keys.record, err = deriveContentKeys(keys.data, s.vaultID); err != nil {
		keys.clear()
		return nil, err
	}
	entries := append([]entryMeta(nil), s.entries...)
	records := make([]sealedBox, len(s.records))
	for i := range entries {
		if records[i], err = s.resealRecord(i, keys.record); err != nil {
			keys.clear()
			return nil, err
		}
		entries[i].digest = sha256.Sum256(encodeBox(records[i]))
	}
	pending, err := s.prepareSealed(entries, records, append([]Group(nil), s.groups...), s.retention, keys.index, keys.recovery)
	if err != nil {
		keys.clear()
		return nil, err
	}
	pending.rekeyed = keys
	return pending, nil
}

// resealRecord seals the record at index under recordKey for the same ID and revision.
func (s *Session) resealRecord(index int, recordKey [32]byte) (sealedBox, error) {
	plaintext, err := s.openRecord(index)
	if err != nil {
		return sealedBox{}, err
	}
	defer clear(plaintext)
	entry := s.entries[index]
	return seal(recordKey, plaintext, recordAAD(s.vaultID, entry.id, entry.revision))
}
