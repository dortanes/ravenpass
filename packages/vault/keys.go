package vault

import (
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"strings"

	bip39 "github.com/kslamph/bip39-hdwallet/bip39"
	"golang.org/x/crypto/chacha20poly1305"
)

const formatPrefix = "RAVENVLT\x00\x03"

// wrappedKeyBytes is the ciphertext of a sealed 32-byte vault key.
const wrappedKeyBytes = 32 + chacha20poly1305.Overhead

type sealedBox struct {
	nonce      [24]byte
	ciphertext []byte
}

func randomBytes(value []byte) error {
	_, err := rand.Read(value)
	return err
}

func deriveKey(secret []byte, vaultID ID, purpose string) ([32]byte, error) {
	var key [32]byte
	derived, err := hkdf.Key(sha256.New, secret, vaultID[:], "ravenpass/vault/v1/"+purpose, 32)
	if err == nil {
		copy(key[:], derived)
		clear(derived)
	}
	return key, err
}

func seal(key [32]byte, plaintext, aad []byte) (sealedBox, error) {
	var box sealedBox
	if err := randomBytes(box.nonce[:]); err != nil {
		return sealedBox{}, err
	}
	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return sealedBox{}, err
	}
	box.ciphertext = aead.Seal(nil, box.nonce[:], plaintext, aad)
	return box, nil
}

func openBox(key [32]byte, box sealedBox, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, box.nonce[:], box.ciphertext, aad)
	if err != nil {
		return nil, ErrAuthentication
	}
	return plaintext, nil
}

// A box is [nonce, ciphertext].
type wireBox struct {
	_          struct{} `cbor:",toarray"`
	Nonce      [24]byte
	Ciphertext []byte
}

func (b sealedBox) wire() wireBox {
	return wireBox{Nonce: b.nonce, Ciphertext: b.ciphertext}
}

// sealed is the box, refused when its ciphertext exceeds maxCiphertext or is shorter than its tag.
func (w wireBox) sealed(maxCiphertext int) (sealedBox, error) {
	if len(w.Ciphertext) > maxCiphertext {
		return sealedBox{}, ErrResourceLimit
	}
	if len(w.Ciphertext) < chacha20poly1305.Overhead {
		return sealedBox{}, ErrMalformed
	}
	return sealedBox{nonce: w.Nonce, ciphertext: w.Ciphertext}, nil
}

func encodeBox(box sealedBox) []byte {
	return mustMarshal(box.wire())
}

// A header is [vaultID, suite]; suite 1 is HKDF-SHA-256 with XChaCha20-Poly1305.
const cipherSuite = 1

type wireHeader struct {
	_       struct{} `cbor:",toarray"`
	VaultID ID
	Suite   uint64
}

func headerBytes(vaultID ID) []byte {
	return mustMarshal(wireHeader{VaultID: vaultID, Suite: cipherSuite})
}

func headerHash(vaultID ID) [32]byte {
	return sha256.Sum256(append([]byte(formatPrefix), headerBytes(vaultID)...))
}

func recoveryAAD(vaultID ID) []byte {
	hash := headerHash(vaultID)
	return append([]byte{1}, hash[:]...)
}

// keyIdentity names a vault key by the recovery box sealing it, which changes only when the key is replaced.
func keyIdentity(recovery sealedBox) [32]byte {
	return sha256.Sum256(encodeBox(recovery))
}

func indexAAD(vaultID ID, recovery sealedBox) []byte {
	hash := headerHash(vaultID)
	key := keyIdentity(recovery)
	result := make([]byte, 0, 65)
	result = append(result, 2)
	result = append(result, hash[:]...)
	return append(result, key[:]...)
}

func recordAAD(vaultID ID, id ID, revision uint64) []byte {
	hash := headerHash(vaultID)
	result := make([]byte, 0, 57)
	result = append(result, 3)
	result = append(result, hash[:]...)
	result = append(result, id[:]...)
	for shift := uint(56); ; shift -= 8 {
		result = append(result, byte(revision>>shift))
		if shift == 0 {
			break
		}
	}
	return result
}

func deviceAAD(vaultID ID, key [32]byte) []byte {
	hash := headerHash(vaultID)
	result := make([]byte, 0, 65)
	result = append(result, 4)
	result = append(result, hash[:]...)
	return append(result, key[:]...)
}

func mnemonicFromEntropy(entropy []byte) (string, error) {
	if len(entropy) != 32 {
		return "", ErrInvalidPhrase
	}
	return bip39.NewMnemonic(entropy)
}

func entropyFromMnemonic(phrase string) ([]byte, error) {
	words := strings.Fields(phrase)
	if len(words) != 24 {
		return nil, ErrInvalidPhrase
	}
	entropy, err := bip39.EntropyFromMnemonic(strings.Join(words, " "))
	if err != nil || len(entropy) != 32 {
		return nil, ErrInvalidPhrase
	}
	return entropy, nil
}

func deviceCacheAAD(vaultID ID, name string) []byte {
	hash := headerHash(vaultID)
	result := make([]byte, 0, 33+len(name))
	result = append(result, 5)
	result = append(result, hash[:]...)
	return append(result, name...)
}

// An envelope is [vaultID, suite, box]: a box sealed outside the container, bound to its vault.
type wireEnvelope struct {
	_       struct{} `cbor:",toarray"`
	VaultID ID
	Suite   uint64
	Box     wireBox
}

func encodeEnvelope(vaultID ID, box sealedBox) []byte {
	return mustMarshal(wireEnvelope{VaultID: vaultID, Suite: cipherSuite, Box: box.wire()})
}

// A device envelope is [vaultID, suite, key, box]: a vault key sealed for a device, naming the key by keyIdentity.
type wireDeviceEnvelope struct {
	_       struct{} `cbor:",toarray"`
	VaultID ID
	Suite   uint64
	Key     [32]byte
	Box     wireBox
}

func encodeDeviceEnvelope(vaultID ID, key [32]byte, box sealedBox) []byte {
	return mustMarshal(wireDeviceEnvelope{VaultID: vaultID, Suite: cipherSuite, Key: key, Box: box.wire()})
}

// decodeDeviceEnvelope returns the identity of the key the envelope holds and its box.
func decodeDeviceEnvelope(encoded []byte, vaultID ID) ([32]byte, sealedBox, error) {
	var envelope wireDeviceEnvelope
	if err := unmarshal(encoded, &envelope); err != nil {
		return [32]byte{}, sealedBox{}, err
	}
	if envelope.VaultID != vaultID {
		return [32]byte{}, sealedBox{}, ErrAuthentication
	}
	if envelope.Suite != cipherSuite {
		return [32]byte{}, sealedBox{}, ErrUnsupported
	}
	box, err := envelope.Box.sealed(wrappedKeyBytes)
	if err != nil {
		return [32]byte{}, sealedBox{}, err
	}
	if len(box.ciphertext) != wrappedKeyBytes {
		return [32]byte{}, sealedBox{}, ErrMalformed
	}
	return envelope.Key, box, nil
}

func decodeEnvelope(encoded []byte, vaultID ID, maxCiphertext int) (sealedBox, error) {
	var envelope wireEnvelope
	if err := unmarshal(encoded, &envelope); err != nil {
		return sealedBox{}, err
	}
	if envelope.VaultID != vaultID {
		return sealedBox{}, ErrAuthentication
	}
	if envelope.Suite != cipherSuite {
		return sealedBox{}, ErrUnsupported
	}
	return envelope.Box.sealed(maxCiphertext)
}
