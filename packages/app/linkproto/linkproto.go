// Package linkproto is the Noise-over-WebSocket protocol between the desktop app and its extension.
package linkproto

import (
	"bytes"
	"crypto/ecdh"
	"io"
	"regexp"
	"slices"

	"github.com/flynn/noise"
)

const (
	// LinkPath serves the Noise_XXpsk3 link handshake; the extension always initiates.
	LinkPath = "/v1/link"
	// SessionPath serves the Noise_IK session handshake with the linked static keys.
	SessionPath = "/v1/session"

	// MaxMessageBytes bounds every WebSocket message, each exactly one Noise message.
	MaxMessageBytes = noise.MaxMsgLen
	// MaxPlaintextBytes bounds the plaintext one transport message carries.
	MaxPlaintextBytes = MaxMessageBytes - tagSize

	// KeySize is the length of an X25519 private or public key.
	KeySize = 32
)

// A CloseCode ends a WebSocket connection and tells the extension why.
type CloseCode int

// Close codes the desktop app sends.
const (
	CloseFinished        CloseCode = 1000
	CloseMalformed       CloseCode = 4400
	CloseNotLinked       CloseCode = 4401
	CloseUnauthenticated CloseCode = 4403
	CloseNoKey           CloseCode = 4404
	CloseIdle            CloseCode = 4408
)

// The extension ID derives from the public key in the extension's manifest.
var allowedOrigins = []string{
	"chrome-extension://cceiadaelnccfbakmhcleifjfilkakag",
	"webkit-extension://cceiadaelnccfbakmhcleifjfilkakag",
}

// WebKit assigns extension origins per installation; they cannot identify the publisher.
// The waiting secret and linked keys authenticate peers after this web-page exclusion gate.
var webkitOrigin = regexp.MustCompile(`^(webkit-extension|safari-web-extension)://[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

// AllowedOrigins returns fixed publisher origins; AllowedOrigin also handles WebKit's assigned origins.
func AllowedOrigins() []string {
	return slices.Clone(allowedOrigins)
}

// AllowedOrigin reports whether origin is one the desktop app answers.
func AllowedOrigin(origin string) bool {
	return slices.Contains(allowedOrigins, origin) || webkitOrigin.MatchString(origin)
}

// KeyPair is a v1 X25519 or v2 P-256 key pair, distinguished by its public encoding.
type KeyPair struct {
	Private []byte
	Public  []byte
}

// GenerateKeyPair reads a private key from random and derives its public key.
func GenerateKeyPair(random io.Reader) (KeyPair, error) {
	key, err := noise.DH25519.GenerateKeypair(random)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{Private: key.Private, Public: key.Public}, nil
}

// Valid reports whether the public key derives from a valid private key.
func (k KeyPair) Valid() bool {
	if len(k.Public) == P256PublicKeySize {
		private, err := ecdh.P256().NewPrivateKey(k.Private)
		return err == nil && bytes.Equal(private.PublicKey().Bytes(), k.Public)
	}
	if len(k.Private) != KeySize || len(k.Public) != KeySize {
		return false
	}
	derived, err := GenerateKeyPair(bytes.NewReader(k.Private))
	return err == nil && bytes.Equal(derived.Public, k.Public)
}

func (k KeyPair) noise() noise.DHKey {
	return noise.DHKey{Private: k.Private, Public: k.Public}
}
