//go:build cgo

package secureenclave

/*
#cgo LDFLAGS: -L/usr/lib/swift -framework LocalAuthentication -framework Security
#include <stdint.h>

int32_t ravenpass_enclave_create(int32_t keychain, int32_t policy, const uint8_t *salt, intptr_t saltLength,
    uint8_t *boundKey, intptr_t boundKeyCapacity, intptr_t *boundKeyLength,
    uint8_t *peerKey, intptr_t peerKeyCapacity, intptr_t *peerKeyLength,
    uint8_t *secret);
int32_t ravenpass_enclave_derive(int32_t keychain, int32_t policy, const uint8_t *boundKey, intptr_t boundKeyLength,
    const uint8_t *peerKey, intptr_t peerKeyLength,
    const uint8_t *salt, intptr_t saltLength,
    const uint8_t *reason, intptr_t reasonLength,
    uint8_t *secret);
*/
import "C"

import (
	"unsafe"

	"github.com/dortanes/ravenpass/packages/app/ownerauth"
	"github.com/dortanes/ravenpass/packages/app/unlock"
)

const (
	// Every key is kept as a keychain label of labelSize in secureenclave.swift.
	boundKeyBytes = 16
	// An uncompressed P-256 public key in X9.63 form.
	peerKeyBytes = 65
)

// Result codes shared with secureenclave.swift.
const (
	resultSucceeded           = 0
	resultCanceled            = 2
	resultNotAuthenticated    = 3
	resultOwnerUnavailable    = 4
	resultInteractionRequired = 5
)

func create(which policy, salt []byte) ([]byte, []byte, [unlock.SecretSize]byte, error) {
	var secret [unlock.SecretSize]byte
	boundKey := make([]byte, boundKeyBytes)
	peerKey := make([]byte, peerKeyBytes)
	var boundKeyLength, peerKeyLength C.intptr_t
	result := C.ravenpass_enclave_create(C.int32_t(keys), C.int32_t(which),
		bytePointer(salt), C.intptr_t(len(salt)),
		(*C.uint8_t)(unsafe.Pointer(&boundKey[0])), C.intptr_t(len(boundKey)), &boundKeyLength,
		(*C.uint8_t)(unsafe.Pointer(&peerKey[0])), C.intptr_t(len(peerKey)), &peerKeyLength,
		(*C.uint8_t)(unsafe.Pointer(&secret[0])))
	if result != resultSucceeded {
		clear(secret[:])
		return nil, nil, secret, ErrCreate
	}
	return boundKey[:boundKeyLength], peerKey[:peerKeyLength], secret, nil
}

func derive(which policy, boundKey, peerKey, salt []byte, reason string) ([unlock.SecretSize]byte, error) {
	var secret [unlock.SecretSize]byte
	var reasonBytes *C.uint8_t
	if reason != "" {
		reasonBytes = (*C.uint8_t)(unsafe.Pointer(unsafe.StringData(reason)))
	}
	result := C.ravenpass_enclave_derive(C.int32_t(keys), C.int32_t(which),
		bytePointer(boundKey), C.intptr_t(len(boundKey)),
		bytePointer(peerKey), C.intptr_t(len(peerKey)),
		bytePointer(salt), C.intptr_t(len(salt)),
		reasonBytes, C.intptr_t(len(reason)),
		(*C.uint8_t)(unsafe.Pointer(&secret[0])))
	if result == resultSucceeded {
		return secret, nil
	}
	clear(secret[:])
	switch result {
	case resultCanceled:
		return secret, ownerauth.ErrCanceled
	case resultNotAuthenticated:
		return secret, ownerauth.ErrFailed
	case resultOwnerUnavailable:
		return secret, ownerauth.ErrUnavailable
	case resultInteractionRequired:
		return secret, ErrInteractionRequired
	default:
		return secret, ErrRejected
	}
}

func bytePointer(value []byte) *C.uint8_t { return (*C.uint8_t)(unsafe.Pointer(&value[0])) }
