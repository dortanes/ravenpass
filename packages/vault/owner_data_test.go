package vault

import (
	"bytes"
	"errors"
	"testing"
)

func TestOwnerDataOpensOnlyUnderItsOwnNameInItsOwnVault(t *testing.T) {
	first, _ := vaultWith(t)
	defer first.Session.Lock()
	second, _ := vaultWith(t)
	defer second.Session.Lock()
	plaintext := []byte("Orbit-Velvet7-Canyon")
	sealed, err := first.Session.SealOwnerData("generator", plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, plaintext) {
		t.Fatal("the sealed record carries its plaintext")
	}
	if opened, err := first.Session.OpenOwnerData("generator", sealed); err != nil || !bytes.Equal(opened, plaintext) {
		t.Fatalf("opened %q, %v", opened, err)
	}
	if _, err := first.Session.OpenOwnerData("other", sealed); err == nil {
		t.Fatal("a record opened under another name")
	}
	if _, err := second.Session.OpenOwnerData("generator", sealed); err == nil {
		t.Fatal("a record opened in another vault")
	}
	tampered := append([]byte(nil), sealed...)
	tampered[len(tampered)-1] ^= 1
	if _, err := first.Session.OpenOwnerData("generator", tampered); err == nil {
		t.Fatal("an altered record opened")
	}
}

func TestOwnerDataAndDeviceCachesDoNotOpenAsEachOther(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	defer session.Lock()
	record, err := session.SealOwnerData("generator", []byte("record"))
	if err != nil {
		t.Fatal(err)
	}
	cache, err := session.SealDeviceCache("generator", []byte("cache"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.OpenDeviceCache("generator", record); err == nil {
		t.Fatal("owner data opened as a device cache")
	}
	if _, err := session.OpenOwnerData("generator", cache); err == nil {
		t.Fatal("a device cache opened as owner data")
	}
}

func TestARekeySealsOwnerDataThatOpensOnceItCommits(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "GitHub"})
	session := created.Session
	defer session.Lock()
	old, err := session.SealOwnerData("generator", []byte("before"))
	if err != nil {
		t.Fatal(err)
	}
	rekey, err := session.BeginRekey()
	if err != nil {
		t.Fatal(err)
	}
	resealed, err := rekey.SealOwnerData("generator", []byte("after"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.OpenOwnerData("generator", resealed); err == nil {
		t.Fatal("a record sealed for the new key opened before the rekey committed")
	}
	pending, err := session.PrepareRekey(rekey)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if opened, err := session.OpenOwnerData("generator", resealed); err != nil || string(opened) != "after" {
		t.Fatalf("opened after the rekey %q, %v", opened, err)
	}
	if _, err := session.OpenOwnerData("generator", old); err == nil {
		t.Fatal("a record sealed for the replaced key still opened")
	}
}

func TestLockedSessionsAndDiscardedRekeysSealNoOwnerData(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	sealed, err := session.SealOwnerData("generator", []byte("record"))
	if err != nil {
		t.Fatal(err)
	}
	rekey, err := session.BeginRekey()
	if err != nil {
		t.Fatal(err)
	}
	rekey.Discard()
	if _, err := rekey.SealOwnerData("generator", []byte("record")); !errors.Is(err, ErrLocked) {
		t.Fatalf("a discarded rekey sealed: %v", err)
	}
	session.Lock()
	if _, err := session.SealOwnerData("generator", []byte("record")); !errors.Is(err, ErrLocked) {
		t.Fatalf("a locked session sealed: %v", err)
	}
	if _, err := session.OpenOwnerData("generator", sealed); !errors.Is(err, ErrLocked) {
		t.Fatalf("a locked session opened: %v", err)
	}
	if _, err := session.SealOwnerData("", nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("an unnamed record: %v", err)
	}
}
