package linkserver

import (
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"

	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/flynn/noise"
)

// peerHandshake is the extension's side of a handshake, written against flynn/noise directly.
type peerHandshake struct {
	state     *noise.HandshakeState
	transport *peerTransport
}

func newPeerHandshake(config noise.Config) (*peerHandshake, error) {
	config.CipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherAESGCM, noise.HashSHA256)
	config.Prologue = []byte("ravenpass-link/1")
	if len(config.StaticKeypair.Public) == 65 {
		config.CipherSuite = noise.NewCipherSuite(peerP256{}, noise.CipherAESGCM, noise.HashSHA256)
		config.Prologue = []byte("ravenpass-link/2")
	}
	config.Initiator = true
	config.Random = rand.Reader
	state, err := noise.NewHandshakeState(config)
	if err != nil {
		return nil, err
	}
	return &peerHandshake{state: state}, nil
}

// linkPeer starts Noise_XXpsk3 with the connection key's secret.
func linkPeer(static linkproto.KeyPair, secret []byte) (*peerHandshake, error) {
	return newPeerHandshake(noise.Config{
		Pattern: noise.HandshakeXX, StaticKeypair: noise.DHKey{Private: static.Private, Public: static.Public},
		PresharedKey: secret, PresharedKeyPlacement: 3,
	})
}

// sessionPeer starts Noise_IK towards the desktop app's static key.
func sessionPeer(static linkproto.KeyPair, desktop []byte) (*peerHandshake, error) {
	if len(desktop) != len(static.Public) {
		return nil, errors.New("the desktop key has the wrong size")
	}
	return newPeerHandshake(noise.Config{
		Pattern: noise.HandshakeIK, StaticKeypair: noise.DHKey{Private: static.Private, Public: static.Public},
		PeerStatic: desktop,
	})
}

// peerP256 supplies the independently composed test initiator with native ECDH.
type peerP256 struct{}

func (peerP256) GenerateKeypair(random io.Reader) (noise.DHKey, error) {
	key, err := ecdh.P256().GenerateKey(random)
	if err != nil {
		return noise.DHKey{}, err
	}
	return noise.DHKey{Private: key.Bytes(), Public: key.PublicKey().Bytes()}, nil
}

func (peerP256) DH(private, public []byte) ([]byte, error) {
	key, err := ecdh.P256().NewPrivateKey(private)
	if err != nil {
		return nil, err
	}
	peer, err := ecdh.P256().NewPublicKey(public)
	if err != nil {
		return nil, err
	}
	return key.ECDH(peer)
}

func (peerP256) DHLen() int     { return 65 }
func (peerP256) DHName() string { return "P256" }

func (h *peerHandshake) Write(payload []byte) ([]byte, error) {
	message, send, receive, err := h.state.WriteMessage(nil, payload)
	if err != nil {
		return nil, err
	}
	h.finish(send, receive)
	return message, nil
}

func (h *peerHandshake) Read(message []byte) ([]byte, error) {
	payload, send, receive, err := h.state.ReadMessage(nil, message)
	if err != nil {
		return nil, err
	}
	h.finish(send, receive)
	return payload, nil
}

func (h *peerHandshake) finish(send, receive *noise.CipherState) {
	if send != nil {
		h.transport = &peerTransport{send: send, receive: receive}
	}
}

func (h *peerHandshake) PeerStatic() []byte {
	return h.state.PeerStatic()
}

func (h *peerHandshake) Transport() *peerTransport {
	return h.transport
}

// peerTransport seals and opens the extension's transport messages.
type peerTransport struct {
	send    *noise.CipherState
	receive *noise.CipherState
}

func (t *peerTransport) Seal(plaintext []byte) ([]byte, error) {
	return t.send.Encrypt(nil, nil, plaintext)
}

func (t *peerTransport) Open(message []byte) ([]byte, error) {
	return t.receive.Decrypt(nil, nil, message)
}
