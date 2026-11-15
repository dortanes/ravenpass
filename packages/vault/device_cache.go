package vault

import "golang.org/x/crypto/chacha20poly1305"

const maxDeviceCacheBytes = 32 << 20

// SealDeviceCache seals device-side data that opens only in this vault under the same name, a fixed purpose and never user data.
func (s *Session) SealDeviceCache(name string, plaintext []byte) ([]byte, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	return sealForDevice(&s.dataKey, s.vaultID, deviceCachePurpose(name), deviceCacheAAD(s.vaultID, name), plaintext)
}

// OpenDeviceCache opens what SealDeviceCache sealed for the same name in this vault.
func (s *Session) OpenDeviceCache(name string, sealed []byte) ([]byte, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	return openForDevice(&s.dataKey, s.vaultID, deviceCachePurpose(name), deviceCacheAAD(s.vaultID, name), sealed)
}

func deviceCachePurpose(name string) string {
	return "device-cache/" + name
}

// sealForDevice seals plaintext outside the container under a key derived from dataKey for purpose.
func sealForDevice(dataKey *[32]byte, vaultID ID, purpose string, aad, plaintext []byte) ([]byte, error) {
	if len(plaintext) > maxDeviceCacheBytes-chacha20poly1305.Overhead {
		return nil, ErrResourceLimit
	}
	key, err := deriveKey(dataKey[:], vaultID, purpose)
	if err != nil {
		return nil, err
	}
	box, err := seal(key, plaintext, aad)
	clear(key[:])
	if err != nil {
		return nil, err
	}
	return encodeEnvelope(vaultID, box), nil
}

// openForDevice opens what sealForDevice sealed under the same key, purpose and additional data.
func openForDevice(dataKey *[32]byte, vaultID ID, purpose string, aad, sealed []byte) ([]byte, error) {
	box, err := decodeEnvelope(sealed, vaultID, maxDeviceCacheBytes)
	if err != nil {
		return nil, err
	}
	key, err := deriveKey(dataKey[:], vaultID, purpose)
	if err != nil {
		return nil, err
	}
	plaintext, err := openBox(key, box, aad)
	clear(key[:])
	return plaintext, err
}
