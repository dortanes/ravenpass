package vaultservice

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

func testCard() vault.CardInput {
	return vault.CardInput{
		Label:        "Everyday",
		Holder:       "Alex Example",
		Number:       "4111111111111111",
		Expiry:       "2029-08",
		SecurityCode: "7391",
		PIN:          "482913",
		Network:      vault.NetworkVisa,
		BankName:     "Jyske Bank",
		BankSite:     "jyskebank.dk",
		Color:        "#00a0e1",
		Billing:      &vault.Address{Street: "Vestergade 8-16", City: "Silkeborg"},
		Notes:        "kept in the wallet",
	}
}

func readTestCard(t *testing.T, service *Service, id vault.ID) vault.Card {
	t.Helper()
	selection, err := service.Select(id)
	if err != nil {
		t.Fatal(err)
	}
	card, err := service.ReadSelectedCard(selection)
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func TestCardWritesRoundTripAcrossLock(t *testing.T) {
	files := &memoryFiles{}
	service := newTestService(t, files, newMemoryKeys())
	createTestVault(t, service)
	input := testCard()
	id, err := service.CreateCard(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{input.Number, input.Holder, input.PIN, input.Billing.Street, input.Notes} {
		if bytes.Contains(files.data, []byte(value)) {
			t.Fatalf("%q appeared in the vault file", value)
		}
	}
	if card := readTestCard(t, service, id); !reflect.DeepEqual(card.CardInput, input) {
		t.Fatalf("card = %+v", card.CardInput)
	}
	owner, err := service.CreateIdentity(vault.IdentityInput{Label: "Alex", Addresses: []vault.Address{{City: "Aarhus"}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	addresses, err := service.IdentityAddresses()
	if err != nil || len(addresses) != 1 || addresses[0].Identity != owner {
		t.Fatalf("identity addresses = %+v, error = %v", addresses, err)
	}
	replacement := vault.CardInput{Label: "Travel", Number: "2200123412341234", Network: vault.NetworkMir,
		BillingLink: &vault.AddressLink{Identity: owner, Address: addresses[0].Addresses[0].ID}}
	if err := service.EditCard(id, replacement, nil); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatal(err)
	}
	card := readTestCard(t, service, id)
	if !reflect.DeepEqual(card.CardInput, replacement) || card.Linked == nil || card.Linked.City != "Aarhus" {
		t.Fatalf("card after a lock = %+v, linked = %+v", card.CardInput, card.Linked)
	}
	if err := service.DeleteItem(owner); err != nil {
		t.Fatal(err)
	}
	if card := readTestCard(t, service, id); card.BillingLink != nil || card.Linked != nil {
		t.Fatalf("card after its identity was deleted = %+v", card.CardInput)
	}
}

func TestCardWritesRefuseAnotherKind(t *testing.T) {
	files := &memoryFiles{}
	service := newTestService(t, files, newMemoryKeys())
	createTestVault(t, service)
	identity, err := service.CreateIdentity(testIdentity(), nil)
	if err != nil {
		t.Fatal(err)
	}
	saved := bytes.Clone(files.data)
	if err := service.EditCard(identity, testCard(), nil); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("card edit of an identity = %v", err)
	}
	selection, err := service.Select(identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReadSelectedCard(selection); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("card read of an identity = %v", err)
	}
	if !bytes.Equal(files.data, saved) {
		t.Fatal("a refused card write reached the vault file")
	}
}

func TestFillableCardsListUnexpiredCardsFirstByUse(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	createTestVault(t, service)
	service.now = func() time.Time { return time.Date(2027, time.March, 1, 12, 0, 0, 0, time.Local) }
	card := func(label, expiry string) vault.ID {
		input := testCard()
		input.Label, input.Expiry = label, expiry
		id, err := service.CreateCard(input, nil)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	lapsed := card("Lapsed", "2027-02")
	current := card("Current", "2027-03")
	undated := card("Undated", "")
	used := card("Used", "2030-01")
	if _, err := service.CreateCredential(vault.CredentialInput{Label: "Mail", Password: "secret"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := service.MarkUsed(used); err != nil {
		t.Fatal(err)
	}
	cards, err := service.FillableCards()
	if err != nil {
		t.Fatal(err)
	}
	var order []vault.ID
	for _, entry := range cards {
		order = append(order, entry.ID)
	}
	if want := []vault.ID{used, current, undated, lapsed}; !reflect.DeepEqual(order, want) {
		t.Fatalf("fillable cards = %v, want %v", order, want)
	}
}

func TestFillCardResolvesTheBillingAddressAndRecordsTheUse(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	createTestVault(t, service)
	owner, err := service.CreateIdentity(vault.IdentityInput{Label: "Alex", Addresses: []vault.Address{{City: "Aarhus"}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	addresses, err := service.IdentityAddresses()
	if err != nil {
		t.Fatal(err)
	}
	linked := testCard()
	linked.Billing, linked.BillingLink = nil, &vault.AddressLink{Identity: owner, Address: addresses[0].Addresses[0].ID}
	id, err := service.CreateCard(linked, nil)
	if err != nil {
		t.Fatal(err)
	}
	card, err := service.FillCard(id)
	if err != nil {
		t.Fatal(err)
	}
	if billing := card.BillingAddress(); card.Number != linked.Number || card.SecurityCode != linked.SecurityCode || billing == nil || billing.City != "Aarhus" {
		t.Fatalf("filled card = %+v, billing = %+v", card.CardInput, billing)
	}
	usage, err := service.Usage()
	if err != nil || usage[id] <= 0 {
		t.Fatalf("usage after a fill = %v, error = %v", usage, err)
	}
	own, err := service.CreateCard(testCard(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if card, err := service.FillCard(own); err != nil || card.BillingAddress().Street != "Vestergade 8-16" {
		t.Fatalf("own billing address = %+v, error = %v", card.BillingAddress(), err)
	}
	if _, err := service.FillCard(owner); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("filling an identity as a card: %v", err)
	}
}

func TestCardMethodsNeedAnOpenVault(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	cases := []struct {
		name string
		call func() error
	}{
		{name: "ReadSelectedCard", call: func() error { _, err := service.ReadSelectedCard(vault.Selection{}); return err }},
		{name: "CreateCard", call: func() error { _, err := service.CreateCard(testCard(), nil); return err }},
		{name: "EditCard", call: func() error { return service.EditCard(vault.ID{1}, testCard(), nil) }},
		{name: "IdentityAddresses", call: func() error { _, err := service.IdentityAddresses(); return err }},
		{name: "FillableCards", call: func() error { _, err := service.FillableCards(); return err }},
		{name: "FillCard", call: func() error { _, err := service.FillCard(vault.ID{1}); return err }},
		{name: "CardCaptureOffer", call: func() error { _, err := service.CardCaptureOffer(typedCard()); return err }},
		{name: "SaveCardCapture", call: func() error {
			_, err := service.SaveCardCapture(typedCard(), CaptureChoice{Name: "Card"}, "")
			return err
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.call(); !errors.Is(err, ErrNotReady) {
				t.Fatalf("error before a vault is open = %v, want ErrNotReady", err)
			}
		})
	}
}
