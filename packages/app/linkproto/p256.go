package linkproto

import (
	"crypto/ecdh"
	"io"

	"github.com/flynn/noise"
)

const (
	// P256LinkPath and P256SessionPath domain-separate the P-256 channel from v1.
	P256LinkPath    = "/v2/link"
	P256SessionPath = "/v2/session"
	// P256PublicKeySize is an uncompressed SEC1 point, including its 0x04 prefix.
	P256PublicKeySize = 65
)

// p256 supplies native, validated P-256 ECDH to the existing Noise state machine.
type p256 struct{}

func (p256) GenerateKeypair(random io.Reader) (noise.DHKey, error) {
	private, err := ecdh.P256().GenerateKey(random)
	if err != nil {
		return noise.DHKey{}, err
	}
	return noise.DHKey{Private: private.Bytes(), Public: private.PublicKey().Bytes()}, nil
}

func (p256) DH(privateBytes, publicBytes []byte) ([]byte, error) {
	private, err := ecdh.P256().NewPrivateKey(privateBytes)
	if err != nil {
		return nil, err
	}
	public, err := ecdh.P256().NewPublicKey(publicBytes)
	if err != nil {
		return nil, err
	}
	return private.ECDH(public)
}

// flynn/noise uses DHLen to frame public keys; the ECDH result remains 32 bytes.
func (p256) DHLen() int     { return P256PublicKeySize }
func (p256) DHName() string { return "P256" }

// GenerateP256KeyPair generates a native P-256 pair for the v2 channel.
func GenerateP256KeyPair(random io.Reader) (KeyPair, error) {
	key, err := (p256{}).GenerateKeypair(random)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{Private: key.Private, Public: key.Public}, nil
}

// ValidPublicKey reports whether a key can identify an extension in either channel.
func ValidPublicKey(public []byte) bool {
	if len(public) == KeySize {
		return true // v1 rejects low-order X25519 points during the handshake.
	}
	_, err := ecdh.P256().NewPublicKey(public)
	return err == nil
}
