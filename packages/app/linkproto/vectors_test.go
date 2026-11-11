package linkproto

import (
	"bytes"
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/linkkey"
	"github.com/flynn/noise"
)

var update = flag.Bool("update", false, "rewrite testdata/vectors.json")

// The extension's key parser and Noise tests replay this file and must agree with it.
var vectorsPath = filepath.Join("testdata", "vectors.json")

type vectorFile struct {
	Keys       keyVectors        `json:"keys"`
	Handshakes []handshakeVector `json:"handshakes"`
}

type keyVectors struct {
	Valid   []validKeyVector   `json:"valid"`
	Invalid []invalidKeyVector `json:"invalid"`
}

type validKeyVector struct {
	Key    string `json:"key"`
	Port   int    `json:"port"`
	Secret string `json:"secret"`
}

type invalidKeyVector struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

type keyPairVector struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}

type handshakeVector struct {
	Protocol           string          `json:"protocol"`
	Prologue           string          `json:"prologue"`
	PSK                *string         `json:"psk"`
	InitiatorStatic    keyPairVector   `json:"initiatorStatic"`
	InitiatorEphemeral keyPairVector   `json:"initiatorEphemeral"`
	ResponderStatic    keyPairVector   `json:"responderStatic"`
	ResponderEphemeral keyPairVector   `json:"responderEphemeral"`
	HandshakeMessages  int             `json:"handshakeMessages"`
	HandshakeHash      string          `json:"handshakeHash"`
	Messages           []messageVector `json:"messages"`
}

type messageVector struct {
	Sender     string `json:"sender"`
	Payload    string `json:"payload"`
	Ciphertext string `json:"ciphertext"`
}

const (
	initiator = "initiator"
	responder = "responder"
)

// step is one message of a transcript: who sends it and what it carries.
type step struct {
	sender  string
	payload string
}

// fixedBytes derives 32 bytes from a label, so every vector is reproducible.
func fixedBytes(label string) []byte {
	sum := sha256.Sum256([]byte("ravenpass-link/1 vectors: " + label))
	return sum[:]
}

func fixedKeyPair(t *testing.T, label string) KeyPair {
	t.Helper()
	key, err := GenerateKeyPair(bytes.NewReader(fixedBytes(label)))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func fixedP256KeyPair(t *testing.T, label string) KeyPair {
	t.Helper()
	private, err := ecdh.P256().NewPrivateKey(fixedBytes(label))
	if err != nil {
		t.Fatal(err)
	}
	return KeyPair{Private: private.Bytes(), Public: private.PublicKey().Bytes()}
}

type fixedAgreement struct {
	noise.DHFunc
	key noise.DHKey
}

func (f fixedAgreement) GenerateKeypair(io.Reader) (noise.DHKey, error) { return f.key, nil }

func vectorHandshake(static, ephemeral KeyPair, peer, secret []byte, initiator bool) (*Handshake, error) {
	var agreement noise.DHFunc = noise.DH25519
	prefix := prologue
	if len(static.Public) == P256PublicKeySize {
		agreement = p256{}
		prefix = []byte("ravenpass-link/2")
	}
	config := noise.Config{
		StaticKeypair: static.noise(), Initiator: initiator, Prologue: prefix,
		CipherSuite: noise.NewCipherSuite(fixedAgreement{agreement, ephemeral.noise()}, noise.CipherAESGCM, noise.HashSHA256),
	}
	if secret != nil {
		config.Pattern = noise.HandshakeXX
		config.PresharedKey = secret
		config.PresharedKeyPlacement = pskPlacement
	} else {
		config.Pattern = noise.HandshakeIK
		config.PeerStatic = peer
	}
	state, err := noise.NewHandshakeState(config)
	if err != nil {
		return nil, err
	}
	return &Handshake{state: state, initiator: initiator}, nil
}

func keyPairHex(key KeyPair) keyPairVector {
	return keyPairVector{Private: hex.EncodeToString(key.Private), Public: hex.EncodeToString(key.Public)}
}

func encodeKey(t *testing.T, port uint16, secret []byte) string {
	t.Helper()
	key := linkkey.Key{Port: port}
	copy(key.Secret[:], secret)
	text, err := key.Encode()
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// resealed returns a valid key altered by change, with a checksum over the altered bytes.
func resealed(t *testing.T, text string, change func([]byte)) string {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		t.Fatal(err)
	}
	change(raw)
	sum := sha256.Sum256(raw[:38])
	copy(raw[38:], sum[:4])
	return base64.RawURLEncoding.EncodeToString(raw)
}

func buildKeyVectors(t *testing.T) keyVectors {
	t.Helper()
	var vectors keyVectors
	for _, valid := range []struct {
		port  uint16
		label string
	}{{53117, "key 1"}, {1024, "key 2"}, {65535, "key 3"}} {
		secret := fixedBytes(valid.label)
		vectors.Valid = append(vectors.Valid, validKeyVector{
			Key: encodeKey(t, valid.port, secret), Port: int(valid.port), Secret: hex.EncodeToString(secret),
		})
	}
	base := vectors.Valid[0].Key
	raw, err := base64.RawURLEncoding.DecodeString(base)
	if err != nil {
		t.Fatal(err)
	}
	alteredChecksum := bytes.Clone(raw)
	alteredChecksum[41] ^= 0x01
	alteredSecret := bytes.Clone(raw)
	alteredSecret[6] ^= 0x01
	vectors.Invalid = []invalidKeyVector{
		{Key: "", Reason: "length"},
		{Key: base[:55], Reason: "length"},
		{Key: base + "A", Reason: "length"},
		{Key: base64.RawURLEncoding.EncodeToString(raw[:41]) + "=", Reason: "encoding"},
		{Key: "+" + base[1:], Reason: "encoding"},
		{Key: "/" + base[1:], Reason: "encoding"},
		{Key: "!" + base[1:], Reason: "encoding"},
		{Key: resealed(t, base, func(raw []byte) { raw[0] = 0x9b }), Reason: "header"},
		{Key: resealed(t, base, func(raw []byte) { raw[2] = 0x14 }), Reason: "header"},
		{Key: resealed(t, base, func(raw []byte) { raw[3] = 0 }), Reason: "version"},
		{Key: resealed(t, base, func(raw []byte) { raw[3] = 2 }), Reason: "version"},
		{Key: base64.RawURLEncoding.EncodeToString(alteredChecksum), Reason: "checksum"},
		{Key: base64.RawURLEncoding.EncodeToString(alteredSecret), Reason: "checksum"},
		{Key: resealed(t, base, func(raw []byte) { raw[4], raw[5] = 0x03, 0xff }), Reason: "port"},
		{Key: resealed(t, base, func(raw []byte) { raw[4], raw[5] = 0, 0 }), Reason: "port"},
	}
	return vectors
}

// transcript runs a handshake and its transport messages, checking each arrives intact.
func transcript(t *testing.T, sides map[string]*Handshake, handshakeMessages int, steps []step) ([]messageVector, []byte) {
	t.Helper()
	receiverOf := map[string]string{initiator: responder, responder: initiator}
	messages := make([]messageVector, 0, len(steps))
	for i, next := range steps {
		sender, receiver := sides[next.sender], sides[receiverOf[next.sender]]
		var message, received []byte
		var err error
		if i < handshakeMessages {
			if message, err = sender.Write([]byte(next.payload)); err != nil {
				t.Fatal(err)
			}
			if received, err = receiver.Read(message); err != nil {
				t.Fatal(err)
			}
		} else {
			if message, err = sender.Transport().Seal([]byte(next.payload)); err != nil {
				t.Fatal(err)
			}
			if received, err = receiver.Transport().Open(message); err != nil {
				t.Fatal(err)
			}
		}
		if string(received) != next.payload {
			t.Fatalf("message %d arrived as %q, want %q", i+1, received, next.payload)
		}
		messages = append(messages, messageVector{
			Sender: next.sender, Payload: hex.EncodeToString([]byte(next.payload)), Ciphertext: hex.EncodeToString(message),
		})
	}
	hash := sides[initiator].state.ChannelBinding()
	if !bytes.Equal(hash, sides[responder].state.ChannelBinding()) {
		t.Fatal("the sides ended with different handshake hashes")
	}
	return messages, bytes.Clone(hash)
}

func buildLinkVector(t *testing.T, p256 bool) handshakeVector {
	t.Helper()
	psk := fixedBytes("XXpsk3 psk")
	fixed := fixedKeyPair
	protocol := "Noise_XXpsk3_25519_AESGCM_SHA256"
	prefix := prologue
	if p256 {
		fixed = fixedP256KeyPair
		protocol = "Noise_XXpsk3_P256_AESGCM_SHA256"
		prefix = []byte("ravenpass-link/2")
	}
	initiatorStatic, initiatorEphemeral := fixed(t, "XXpsk3 initiator static"), fixed(t, "XXpsk3 initiator ephemeral")
	responderStatic, responderEphemeral := fixed(t, "XXpsk3 responder static"), fixed(t, "XXpsk3 responder ephemeral")
	extension, err := vectorHandshake(initiatorStatic, initiatorEphemeral, nil, psk, true)
	if err != nil {
		t.Fatal(err)
	}
	desktop, err := vectorHandshake(responderStatic, responderEphemeral, nil, psk, false)
	if err != nil {
		t.Fatal(err)
	}
	messages, hash := transcript(t, map[string]*Handshake{initiator: extension, responder: desktop}, 3, []step{
		{initiator, ""},
		{responder, ""},
		{initiator, `{"name":"Chrome · macOS"}`},
		{responder, `{"type":"linked","language":"ru","signIn":"field"}`},
		{initiator, ""},
		{responder, "second responder transport message"},
		{initiator, "second initiator transport message"},
	})
	pskHex := hex.EncodeToString(psk)
	return handshakeVector{
		Protocol:           protocol,
		Prologue:           hex.EncodeToString(prefix),
		PSK:                &pskHex,
		InitiatorStatic:    keyPairHex(initiatorStatic),
		InitiatorEphemeral: keyPairHex(initiatorEphemeral),
		ResponderStatic:    keyPairHex(responderStatic),
		ResponderEphemeral: keyPairHex(responderEphemeral),
		HandshakeMessages:  3,
		HandshakeHash:      hex.EncodeToString(hash),
		Messages:           messages,
	}
}

func buildSessionVector(t *testing.T, p256 bool) handshakeVector {
	t.Helper()
	fixed := fixedKeyPair
	protocol := "Noise_IK_25519_AESGCM_SHA256"
	prefix := prologue
	if p256 {
		fixed = fixedP256KeyPair
		protocol = "Noise_IK_P256_AESGCM_SHA256"
		prefix = []byte("ravenpass-link/2")
	}
	initiatorStatic, initiatorEphemeral := fixed(t, "IK initiator static"), fixed(t, "IK initiator ephemeral")
	responderStatic, responderEphemeral := fixed(t, "IK responder static"), fixed(t, "IK responder ephemeral")
	extension, err := vectorHandshake(initiatorStatic, initiatorEphemeral, responderStatic.Public, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	desktop, err := vectorHandshake(responderStatic, responderEphemeral, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	messages, hash := transcript(t, map[string]*Handshake{initiator: extension, responder: desktop}, 2, []step{
		{initiator, ""},
		{responder, `{"language":"ru","signIn":"card"}`},
		{initiator, `{"id":1,"type":"status"}`},
		{responder, `{"id":1,"result":{"vault":"unlocked"}}`},
		{initiator, `{"id":2,"type":"unlink"}`},
		{responder, `{"id":2,"result":{}}`},
	})
	return handshakeVector{
		Protocol:           protocol,
		Prologue:           hex.EncodeToString(prefix),
		InitiatorStatic:    keyPairHex(initiatorStatic),
		InitiatorEphemeral: keyPairHex(initiatorEphemeral),
		ResponderStatic:    keyPairHex(responderStatic),
		ResponderEphemeral: keyPairHex(responderEphemeral),
		HandshakeMessages:  2,
		HandshakeHash:      hex.EncodeToString(hash),
		Messages:           messages,
	}
}

func encodeVectors(t *testing.T, vectors vectorFile) []byte {
	t.Helper()
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vectors); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func TestVectorsAreCurrent(t *testing.T) {
	vectors := vectorFile{
		Keys:       buildKeyVectors(t),
		Handshakes: []handshakeVector{buildLinkVector(t, false), buildSessionVector(t, false), buildLinkVector(t, true), buildSessionVector(t, true)},
	}
	encoded := encodeVectors(t, vectors)
	if *update {
		if err := os.MkdirAll(filepath.Dir(vectorsPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(vectorsPath, encoded, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	stored, err := os.ReadFile(vectorsPath)
	if err != nil {
		t.Fatalf("read %s: %v; run go test ./internal/linkproto -update", vectorsPath, err)
	}
	if !bytes.Equal(stored, encoded) {
		t.Fatalf("%s is stale; run go test ./internal/linkproto -update", vectorsPath)
	}
}
