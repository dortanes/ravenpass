package vault

import (
	"runtime/debug"

	"golang.org/x/crypto/argon2"
)

// PIN bounds and sizes; whoever holds the device secret and a PIN-wrapped key can try every PIN offline.
const (
	MinPINLength = 6
	MaxPINLength = 12
	PINSaltSize  = 16
	// PINSecretSize is the device secret mixed into the derivation; never derive it from what the person knows.
	PINSecretSize = 32
)

// Argon2id cost of a PIN derivation: 4 passes over 256 MiB in 4 lanes.
const (
	pinPasses  = 4
	pinMemory  = 256 * 1024
	pinThreads = 4
)

// DerivePINKey turns a PIN into the key wrapping a copy of the vault key; the Argon2id salt is salt followed by deviceSecret.
func DerivePINKey(pin string, salt, deviceSecret []byte) ([32]byte, error) {
	if !ValidPIN(pin) || len(salt) != PINSaltSize || len(deviceSecret) != PINSecretSize {
		return [32]byte{}, ErrInvalidPIN
	}
	combined := make([]byte, 0, len(salt)+len(deviceSecret))
	combined = append(combined, salt...)
	combined = append(combined, deviceSecret...)
	derived := argon2.IDKey([]byte(pin), combined, pinPasses, pinMemory, pinThreads, 32)
	// Two derivations in a row kept 513 MiB resident until the heap was returned to the OS.
	debug.FreeOSMemory()
	clear(combined)
	var key [32]byte
	copy(key[:], derived)
	clear(derived)
	return key, nil
}

// ValidPIN reports a PIN of MinPINLength to MaxPINLength ASCII letters or digits.
func ValidPIN(pin string) bool {
	if len(pin) < MinPINLength || len(pin) > MaxPINLength {
		return false
	}
	for i := range len(pin) {
		b := pin[i]
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')) {
			return false
		}
	}
	return true
}

// NewPINSalt gives the random salt a device keeps beside a PIN.
func NewPINSalt() ([]byte, error) {
	salt := make([]byte, PINSaltSize)
	if err := randomBytes(salt); err != nil {
		return nil, err
	}
	return salt, nil
}
