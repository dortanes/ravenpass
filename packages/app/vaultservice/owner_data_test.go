package vaultservice

import (
	"errors"
	"testing"

	"github.com/dortanes/ravenpass/packages/vault"
)

// heldPiece is one piece of owner data a test holder keeps, failing its reseal on request.
type heldPiece struct {
	id         vault.ID
	sealed     []byte
	failReseal bool
}

func (h *heldPiece) Reseal(id vault.ID, reseal func(name string, sealed []byte) ([]byte, error)) (func(), error) {
	if h.failReseal {
		return nil, errors.New("reseal refused")
	}
	if id != h.id {
		return func() {}, nil
	}
	next, err := reseal("history", h.sealed)
	if err != nil {
		return nil, err
	}
	return func() { h.sealed = next }, nil
}

func TestOwnerDataServesOnlyTheOpenVaultAndFollowsAKeyChange(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	createTestVaultWith(t, service, MethodChoice{PIN: testPIN})
	id, err := service.OpenVaultID()
	if err != nil {
		t.Fatal(err)
	}
	var other vault.ID
	other[0] = id[0] ^ 1
	if _, err := service.SealOwnerData(other, "history", []byte("kept")); !errors.Is(err, ErrNotReady) {
		t.Fatalf("sealing for another vault: %v", err)
	}
	sealed, err := service.SealOwnerData(id, "history", []byte("kept"))
	if err != nil {
		t.Fatal(err)
	}
	followed := &heldPiece{id: id, sealed: sealed}
	refused := &heldPiece{id: id, sealed: sealed, failReseal: true}
	service.HoldOwnerData(followed)
	service.HoldOwnerData(refused)

	phrase, err := service.BeginRekey(openingOf(service), testPIN)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmRekey(phrase); err != nil {
		t.Fatal(err)
	}
	if opened, err := service.OpenOwnerData(id, "history", followed.sealed); err != nil || string(opened) != "kept" {
		t.Fatalf("held data after the key change: %q, %v", opened, err)
	}
	if _, err := service.OpenOwnerData(id, "history", refused.sealed); err == nil {
		t.Fatal("data whose reseal failed still opened under the new key")
	}
	service.Lock()
	if _, err := service.OpenOwnerData(id, "history", followed.sealed); !errors.Is(err, ErrNotReady) {
		t.Fatalf("opening while locked: %v", err)
	}
}
