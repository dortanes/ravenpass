package vaultservice

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/dortanes/ravenpass/packages/vault"
)

// typedCard is testCard's number with values a checkout asked for.
func typedCard() CardCapture {
	return CardCapture{Holder: "Alex Example", Number: "4111111111111111", Expiry: "2029-08", SecurityCode: "7391", Network: vault.NetworkVisa}
}

func TestATypedCardTheVaultHoldsUnchangedOffersNothing(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	createTestVault(t, service)
	if _, err := service.CreateCard(testCard(), nil); err != nil {
		t.Fatal(err)
	}
	for name, capture := range map[string]CardCapture{
		"every value":  typedCard(),
		"number alone": {Number: "4111111111111111"},
		"a bad number": {Number: "4111 1111 1111 1111"},
		"a bad expiry": {Number: "5500005555555559", Expiry: "29-08"},
		"a bad code":   {Number: "5500005555555559", SecurityCode: "12"},
	} {
		offer, err := service.CardCaptureOffer(capture)
		if err != nil || !offer.Nothing {
			t.Fatalf("offer for %s = %+v, error = %v", name, offer, err)
		}
	}
}

func TestATypedCardWithNewValuesOffersToUpdateItsCard(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	createTestVault(t, service)
	held, err := service.CreateCard(testCard(), nil)
	if err != nil {
		t.Fatal(err)
	}
	other := testCard()
	other.Label, other.Number = "Same last four", "5500000000001111"
	if _, err := service.CreateCard(other, nil); err != nil {
		t.Fatal(err)
	}
	renewed := typedCard()
	renewed.Expiry, renewed.SecurityCode, renewed.Holder = "2032-01", "", ""
	offer, err := service.CardCaptureOffer(renewed)
	if err != nil {
		t.Fatal(err)
	}
	want := CaptureOffer{Targets: []Target{{ID: held, Label: "Everyday", Action: TargetUpdate}}, Suggested: held}
	if !reflect.DeepEqual(offer, want) {
		t.Fatalf("offer = %+v", offer)
	}
	created, err := service.SaveCardCapture(renewed, CaptureChoice{Target: held}, "")
	if err != nil || created {
		t.Fatalf("update created = %v, error = %v", created, err)
	}
	card := readTestCard(t, service, held)
	expected := testCard()
	expected.Expiry = "2032-01"
	if !reflect.DeepEqual(card.CardInput, expected) {
		t.Fatalf("updated card = %+v", card.CardInput)
	}
	if _, err := service.SaveCardCapture(renewed, CaptureChoice{Target: held}, ""); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("saving a capture the vault now holds: %v", err)
	}
}

func TestANewTypedCardSavesInTheDefaultGroup(t *testing.T) {
	files := &memoryFiles{}
	service := newTestService(t, files, newMemoryKeys())
	createTestVault(t, service)
	group, err := service.CreateGroup("Wallet")
	if err != nil {
		t.Fatal(err)
	}
	capture := typedCard()
	offer, err := service.CardCaptureOffer(capture)
	if err != nil || offer.Nothing || len(offer.Targets) != 0 || offer.Suggested != (vault.ID{}) {
		t.Fatalf("offer for a new card = %+v, error = %v", offer, err)
	}
	if _, err := service.SaveCardCapture(capture, CaptureChoice{Name: "   "}, group.String()); !errors.Is(err, ErrNameRefused) {
		t.Fatalf("saving under a blank name: %v", err)
	}
	created, err := service.SaveCardCapture(capture, CaptureChoice{Name: "Visa 1111", Account: "ignored"}, group.String())
	if err != nil || !created {
		t.Fatalf("created = %v, error = %v", created, err)
	}
	entries, err := service.List()
	if err != nil {
		t.Fatal(err)
	}
	var saved vault.Entry
	for _, entry := range entries {
		if entry.Kind == vault.KindCard {
			saved = entry
		}
	}
	if !reflect.DeepEqual(saved.Groups, []vault.ID{group}) || saved.Card.LastFour != "1111" || saved.Card.Network != vault.NetworkVisa {
		t.Fatalf("saved entry = %+v", saved)
	}
	card := readTestCard(t, service, saved.ID)
	want := vault.CardInput{Label: "Visa 1111", Holder: "Alex Example", Number: capture.Number, Expiry: "2029-08", SecurityCode: "7391", Network: vault.NetworkVisa}
	if !reflect.DeepEqual(card.CardInput, want) {
		t.Fatalf("saved card = %+v", card.CardInput)
	}
	for _, value := range []string{capture.Number, capture.SecurityCode, capture.Holder} {
		if bytes.Contains(files.data, []byte(value)) {
			t.Fatalf("%q appeared in the vault file", value)
		}
	}
}
