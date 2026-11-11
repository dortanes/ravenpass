package linkproto

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/flynn/noise"
)

// Errors a handshake or transport returns for a message it rejects.
var (
	ErrMalformed      = errors.New("link message is malformed")
	ErrAuthentication = errors.New("link message failed to authenticate")
	ErrTooLarge       = errors.New("link message is too large")
)

// prologue binds every handshake to this protocol version.
var prologue = []byte("ravenpass-link/1")

var suite = noise.NewCipherSuite(noise.DH25519, noise.CipherAESGCM, noise.HashSHA256)

const (
	// pskPlacement mixes the secret in with the third message of XX, after both static keys.
	pskPlacement = 3
	// tagSize is the AES-GCM authentication tag every transport message carries.
	tagSize = 16
)

// Handshake is one side of a handshake in progress.
type Handshake struct {
	state     *noise.HandshakeState
	initiator bool
	transport *Transport
}

// LinkResponder starts the desktop app's side of Noise_XXpsk3 with the waiting key's secret.
func LinkResponder(static KeyPair, secret []byte, random io.Reader) (*Handshake, error) {
	return newHandshake(noise.Config{
		Pattern: noise.HandshakeXX, StaticKeypair: static.noise(),
		PresharedKey: secret, PresharedKeyPlacement: pskPlacement, Random: random,
	})
}

// SessionResponder starts the desktop app's side of Noise_IK.
func SessionResponder(static KeyPair, random io.Reader) (*Handshake, error) {
	return newHandshake(noise.Config{Pattern: noise.HandshakeIK, StaticKeypair: static.noise(), Random: random})
}

func newHandshake(config noise.Config) (*Handshake, error) {
	config.CipherSuite = suite
	config.Prologue = prologue
	if len(config.StaticKeypair.Public) == P256PublicKeySize {
		config.CipherSuite = noise.NewCipherSuite(p256{}, noise.CipherAESGCM, noise.HashSHA256)
		config.Prologue = []byte("ravenpass-link/2")
	}
	state, err := noise.NewHandshakeState(config)
	if err != nil {
		return nil, err
	}
	return &Handshake{state: state, initiator: config.Initiator}, nil
}

// Write returns this side's next handshake message carrying payload.
func (h *Handshake) Write(payload []byte) ([]byte, error) {
	message, first, second, err := h.state.WriteMessage(nil, payload)
	if err != nil {
		return nil, err
	}
	if len(message) > MaxMessageBytes {
		return nil, ErrTooLarge
	}
	h.finish(first, second)
	return message, nil
}

// Read processes the peer's next handshake message and returns its payload.
func (h *Handshake) Read(message []byte) ([]byte, error) {
	if len(message) > MaxMessageBytes {
		return nil, ErrMalformed
	}
	payload, first, second, err := h.state.ReadMessage(nil, message)
	if errors.Is(err, noise.ErrShortMessage) {
		return nil, ErrMalformed
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuthentication, err)
	}
	h.finish(first, second)
	return payload, nil
}

// finish keeps the final cipher states: first protects initiator-to-responder, second the reverse.
func (h *Handshake) finish(first, second *noise.CipherState) {
	if first == nil {
		return
	}
	if h.initiator {
		h.transport = &Transport{send: first, receive: second}
		return
	}
	h.transport = &Transport{send: second, receive: first}
}

// PeerStatic returns the peer's static public key once a message carrying it has been read.
func (h *Handshake) PeerStatic() []byte {
	return bytes.Clone(h.state.PeerStatic())
}

// Transport returns the transport once the handshake is complete, and nil before.
func (h *Handshake) Transport() *Transport {
	return h.transport
}

// Transport protects the messages that follow a handshake; they must be opened in sealing order.
type Transport struct {
	send    *noise.CipherState
	receive *noise.CipherState
}

// Seal encrypts and authenticates one message for the peer.
func (t *Transport) Seal(plaintext []byte) ([]byte, error) {
	if len(plaintext) > MaxPlaintextBytes {
		return nil, ErrTooLarge
	}
	return t.send.Encrypt(nil, nil, plaintext)
}

// Open authenticates and decrypts the peer's next message.
func (t *Transport) Open(message []byte) ([]byte, error) {
	if len(message) > MaxMessageBytes {
		return nil, ErrMalformed
	}
	plaintext, err := t.receive.Decrypt(nil, nil, message)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuthentication, err)
	}
	return plaintext, nil
}
