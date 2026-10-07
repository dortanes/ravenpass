// Package secureenclave derives secrets by HKDF-SHA256 from a Secure Enclave key agreed with a discarded software key.
package secureenclave

import (
	"errors"

	"github.com/dortanes/ravenpass/packages/app/unlock"
)

var (
	ErrUnavailable = errors.New("this device has no Secure Enclave")
	ErrCreate      = errors.New("the Secure Enclave did not create a key")
	ErrRejected    = errors.New("the Secure Enclave does not accept this key")
	ErrInvalid     = errors.New("a Secure Enclave binding needs a salt, a bound key and a peer key")
	ErrNoReason    = errors.New("a key that needs its owner is used only with a reason")
	// ErrInteractionRequired reports a key that needs its owner, used where no prompt may be shown.
	ErrInteractionRequired = errors.New("the key needs its owner, who may not be asked")
)

// keychainKind values match the Keychain raw values in secureenclave.swift; the build picks one as keys.
type keychainKind int32

const (
	keychainDataProtection keychainKind = 0
	keychainLogin          keychainKind = 1
)

// policy values match the Policy raw values in secureenclave.swift.
type policy int32

const (
	policyNone         policy = 0
	policyUserPresence policy = 1
)

// Enclave binds a PIN's device secret to a Secure Enclave key whose use never prompts. Every key's representation is a
// Data Protection keychain item only code signed as Ravenpass reads, so a copy of the device records cannot yield the
// secret; the item needs the application identifier entitlement, which an ad-hoc signature lacks, so a build with the
// adhoc tag keeps it in the login keychain instead.
type Enclave struct{}

// Create makes a Secure Enclave key for salt and returns its keychain label, the peer public key and the salt's secret.
// The keys made for salt before stay until this one is first used.
func (Enclave) Create(salt []byte) (boundKey, peerKey []byte, secret [unlock.SecretSize]byte, err error) {
	return createBinding(policyNone, salt)
}

// Derive yields the secret Create returned and removes salt's other keys, or fails with ErrRejected for a key removed
// that way, on another Mac or after a Secure Enclave reset.
func (Enclave) Derive(boundKey, peerKey, salt []byte) ([unlock.SecretSize]byte, error) {
	return deriveBinding(policyNone, boundKey, peerKey, salt, "")
}

// PresenceEnclave binds a secret to a Secure Enclave key usable only after Touch ID or the Mac's password, kept in the
// keychain apart from PIN keys.
type PresenceEnclave struct{}

// Create makes a key as Enclave.Create does, except its use needs the owner; creating it does not ask them.
func (PresenceEnclave) Create(salt []byte) (boundKey, peerKey []byte, secret [unlock.SecretSize]byte, err error) {
	return createBinding(policyUserPresence, salt)
}

// Derive asks the owner for reason and yields Create's secret, removing salt's other keys as Enclave.Derive does, or
// fails with an ownerauth error or ErrRejected.
func (PresenceEnclave) Derive(reason string, boundKey, peerKey, salt []byte) ([unlock.SecretSize]byte, error) {
	if reason == "" {
		return [unlock.SecretSize]byte{}, ErrNoReason
	}
	return deriveBinding(policyUserPresence, boundKey, peerKey, salt, reason)
}

func createBinding(which policy, salt []byte) ([]byte, []byte, [unlock.SecretSize]byte, error) {
	if len(salt) == 0 {
		return nil, nil, [unlock.SecretSize]byte{}, ErrInvalid
	}
	return create(which, salt)
}

// deriveBinding with no reason shows no prompt.
func deriveBinding(which policy, boundKey, peerKey, salt []byte, reason string) ([unlock.SecretSize]byte, error) {
	if len(boundKey) == 0 || len(peerKey) == 0 || len(salt) == 0 {
		return [unlock.SecretSize]byte{}, ErrInvalid
	}
	return derive(which, boundKey, peerKey, salt, reason)
}
