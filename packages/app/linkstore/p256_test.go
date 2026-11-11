package linkstore

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/linkproto"
)

func TestP256KeysAndExtensionsSurviveReopeningWithoutReplacingV1(t *testing.T) {
	store, path := newTestStore(t)
	legacy, err := store.DesktopKey()
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.P256DesktopKey()
	if err != nil || !first.Valid() || len(first.Public) != linkproto.P256PublicKeySize {
		t.Fatalf("invalid v2 key: %v", err)
	}
	peer, err := linkproto.GenerateP256KeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	added, err := store.Add("WebKit", peer.Public)
	if err != nil {
		t.Fatal(err)
	}
	opened := reopen(t, path)
	kept, err := opened.P256DesktopKey()
	if err != nil || !bytes.Equal(first.Private, kept.Private) || !bytes.Equal(first.Public, kept.Public) {
		t.Fatalf("v2 key changed: %v", err)
	}
	v1, err := opened.DesktopKey()
	if err != nil || !bytes.Equal(legacy.Private, v1.Private) || !bytes.Equal(legacy.Public, v1.Public) {
		t.Fatalf("v1 key changed: %v", err)
	}
	record, found, err := opened.Find(peer.Public)
	if err != nil || !found || record.ID != added.ID {
		t.Fatalf("extension changed: %v", err)
	}
	malformed := bytes.Clone(peer.Public)
	malformed[0] = 0
	if _, err := opened.Add("Invalid", malformed); err == nil {
		t.Fatal("invalid P-256 point persisted")
	}
}
