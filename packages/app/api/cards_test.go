package api

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

func testCardInput() CardInput {
	return CardInput{
		Label:        "Everyday",
		Tags:         []string{"Personal"},
		Holder:       "Alex Example",
		Number:       "4111111111111111",
		Expiry:       "2029-08",
		SecurityCode: "7391",
		PIN:          "482913",
		Network:      "visa",
		BankName:     "Jyske Bank",
		BankSite:     "https://www.JyskeBank.dk/privat",
		Color:        "#00a0e1",
		Billing:      &Address{Street: " Vestergade 8-16 ", City: "Silkeborg", PostalCode: "8600", Country: "Danmark"},
		Notes:        "kept in the wallet",
	}
}

func TestCardsListFromTheIndexAndReadWithTheirSite(t *testing.T) {
	service := newReadyService(t)
	family := createTestGroup(t, service, "Family")
	input := testCardInput()
	id, err := service.CreateCard(input, []string{family})
	if err != nil {
		t.Fatal(err)
	}
	bare, err := service.CreateCard(CardInput{Label: "Spare", Number: "000000000000"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateIdentity(testIdentityInput(), nil); err != nil {
		t.Fatal(err)
	}
	cards, err := service.ListCards()
	if err != nil {
		t.Fatal(err)
	}
	want := []CardSummary{
		{ID: id, Label: "Everyday", BankName: "Jyske Bank", LastFour: "1111", Network: "visa", Color: "#00a0e1", Site: "jyskebank.dk", Groups: []string{family}, Tags: []string{"Personal"}, ExpiresOn: "2029-08-31"},
		{ID: bare, Label: "Spare", LastFour: "0000", Groups: []string{}, Tags: []string{}},
	}
	if !reflect.DeepEqual(cards, want) {
		t.Fatalf("cards = %+v", cards)
	}
	card, err := service.ReadCard(id)
	if err != nil {
		t.Fatal(err)
	}
	if card.ID != id || card.Site != "jyskebank.dk" || card.Linked != nil || !reflect.DeepEqual(card.Groups, []string{family}) || !reflect.DeepEqual(card.CardInput, input) {
		t.Fatalf("card = %+v", card)
	}
	if cards, err := service.ListCards(); err != nil || cards[0].LastUsedAt <= 0 || cards[1].LastUsedAt != 0 {
		t.Fatalf("cards after a read = %+v, error = %v", cards, err)
	}
	encoded, err := json.Marshal(mustReadCard(t, service, bare))
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`"billing":null`, `"billingLink":null`, `"linked":null`, `"network":""`, `"groups":[]`} {
		if !strings.Contains(string(encoded), value) {
			t.Fatalf("%s missing from %s", value, encoded)
		}
	}
	if identities, err := service.ListIdentities(); err != nil || len(identities) != 1 {
		t.Fatalf("identities beside cards = %+v, error = %v", identities, err)
	}
	if credentials, err := service.ListCredentials(); err != nil || len(credentials) != 0 {
		t.Fatalf("credentials beside cards = %+v, error = %v", credentials, err)
	}
}

func mustReadCard(t *testing.T, service *Service, id string) Card {
	t.Helper()
	card, err := service.ReadCard(id)
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func TestCardNetworksCrossTheBridgeByName(t *testing.T) {
	service := newReadyService(t)
	for network, name := range cardNetworkNames {
		if parsed, known := cardNetworkNamed(name); !known || parsed != network {
			t.Fatalf("%q reads as %d", name, parsed)
		}
		input := testCardInput()
		input.Network = name
		id, err := service.CreateCard(input, nil)
		if err != nil {
			t.Fatal(err)
		}
		if card := mustReadCard(t, service, id); card.Network != name {
			t.Fatalf("network %q read as %q", name, card.Network)
		}
	}
	if len(cardNetworkNames) != int(vault.NetworkNaranja) {
		t.Fatalf("%d network names", len(cardNetworkNames))
	}
	for _, name := range []string{"Visa", "amex", "none", " visa"} {
		input := testCardInput()
		input.Network = name
		_, err := service.CreateCard(input, nil)
		assertFailure(t, err, failureInvalidItem)
	}
}

func TestCardInputIsRefusedWithoutWriting(t *testing.T) {
	service := newReadyService(t)
	owner, err := service.CreateIdentity(testIdentityInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*CardInput){
		"number with spaces":           func(input *CardInput) { input.Number = "4111 1111 1111 1111" },
		"expiry as typed":              func(input *CardInput) { input.Expiry = "08/29" },
		"uppercase colour":             func(input *CardInput) { input.Color = "#00A0E1" },
		"billing address with a label": func(input *CardInput) { input.Billing.Label = "Home" },
		"billing address with an id":   func(input *CardInput) { input.Billing.ID = strings.Repeat("1", 32) },
		"billing address and a link": func(input *CardInput) {
			input.BillingLink = &AddressLink{IdentityID: owner, AddressID: strings.Repeat("1", 32)}
		},
		"link that is not hex": func(input *CardInput) {
			input.Billing, input.BillingLink = nil, &AddressLink{IdentityID: owner, AddressID: "home"}
		},
		"link to an address the identity does not hold": func(input *CardInput) {
			input.Billing, input.BillingLink = nil, &AddressLink{IdentityID: owner, AddressID: strings.Repeat("e", 32)}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := testCardInput()
			billing := *input.Billing
			input.Billing = &billing
			mutate(&input)
			_, err := service.CreateCard(input, nil)
			assertFailure(t, err, failureInvalidItem)
		})
	}
	if cards, err := service.ListCards(); err != nil || len(cards) != 0 {
		t.Fatalf("cards after refused input = %+v, error = %v", cards, err)
	}
}

func TestCardLinksToAnIdentityAddress(t *testing.T) {
	service := newReadyService(t)
	owner, err := service.CreateIdentity(testIdentityInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateIdentity(IdentityInput{Label: "Bare"}, nil); err != nil {
		t.Fatal(err)
	}
	listed, err := service.ListIdentityAddresses()
	if err != nil || len(listed) != 1 || listed[0].IdentityID != owner || listed[0].Label != "Alex" || len(listed[0].Addresses) != 2 {
		t.Fatalf("identity addresses = %+v, error = %v", listed, err)
	}
	home := listed[0].Addresses[0]
	input := testCardInput()
	input.Billing, input.BillingLink = nil, &AddressLink{IdentityID: owner, AddressID: home.ID}
	id, err := service.CreateCard(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	card := mustReadCard(t, service, id)
	if !reflect.DeepEqual(card.CardInput, input) || card.Linked == nil || card.Linked.IdentityLabel != "Alex" || card.Linked.Address != home {
		t.Fatalf("linked card = %+v, linked = %+v", card.CardInput, card.Linked)
	}
	clipboard := attachPasteboard(service)
	schedule := func(time.Duration, func()) {}
	for reference, want := range map[string]string{"billing": "1 Example Street\n97403 Springfield\nOregon\nUnited States", "billing:city": "Springfield"} {
		if err := service.copyCardField(id, reference, schedule); err != nil || clipboard.text != want {
			t.Fatalf("copy of linked %s = %q, %v", reference, clipboard.text, err)
		}
	}
	if err := service.DeleteItem(owner); err != nil {
		t.Fatal(err)
	}
	if card := mustReadCard(t, service, id); card.BillingLink != nil || card.Linked != nil {
		t.Fatalf("card after its identity was deleted = %+v", card)
	}
	assertFailure(t, service.copyCardField(id, "billing", schedule), failureFieldEmpty)
}

func TestCopyCardFieldCopiesEachField(t *testing.T) {
	service := newReadyService(t)
	id, err := service.CreateCard(testCardInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	clipboard := attachPasteboard(service)
	var discard func()
	schedule := func(_ time.Duration, clear func()) { discard = clear }
	fields := map[string]string{
		"number":             "4111111111111111",
		"holder":             "Alex Example",
		"expiry":             "08/29",
		"securityCode":       "7391",
		"pin":                "482913",
		"bankName":           "Jyske Bank",
		"bankSite":           "https://www.JyskeBank.dk/privat",
		"notes":              "kept in the wallet",
		"billing":            "Vestergade 8-16\n8600 Silkeborg\nDanmark",
		"billing:street":     "Vestergade 8-16",
		"billing:city":       "Silkeborg",
		"billing:postalCode": "8600",
		"billing:country":    "Danmark",
	}
	for field, want := range fields {
		if err := service.copyCardField(id, field, schedule); err != nil {
			t.Fatalf("copy of %s failed: %v", field, err)
		}
		if clipboard.text != want {
			t.Fatalf("clipboard after copying %s = %q", field, clipboard.text)
		}
	}
	if err := service.copyCardField(id, "billing:country", schedule); err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.copyCardField(id, "billing:region", schedule), failureFieldEmpty)
	for _, field := range []string{"", "label", "network", "color", "bankname", "Number", "billing:", "billing:label", "billing:Street", "billing:city:x", "address:0", "billing:0:city"} {
		assertFailure(t, service.copyCardField(id, field, schedule), failureFieldNotCopyable)
	}
	if clipboard.text != "Danmark" {
		t.Fatalf("a refused field changed the clipboard to %q", clipboard.text)
	}
	discard()
	if clipboard.text != "" {
		t.Fatal("the copied value stayed on the clipboard after its lifetime")
	}
	bare, err := service.CreateCard(CardInput{Label: "Spare", Number: "000000000000"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"holder", "expiry", "securityCode", "pin", "bankName", "bankSite", "notes", "billing", "billing:street"} {
		assertFailure(t, service.copyCardField(bare, field, schedule), failureFieldEmpty)
	}
	identity, err := service.CreateIdentity(testIdentityInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.copyCardField(identity, "number", schedule), failureItemUnreadable)
	assertFailure(t, service.copyIdentityField(id, IdentityField{Kind: "fullName"}, schedule), failureItemUnreadable)
	_, err = service.ReadCard(identity)
	assertFailure(t, err, failureItemUnreadable)
	_, err = service.ReadIdentity(id)
	assertFailure(t, err, failureItemUnreadable)
	assertFailure(t, service.UpdateCard(identity, testCardInput(), nil), failureItemUnreadable)
}

func TestUpdateCardReplacesItAsAWhole(t *testing.T) {
	service := newReadyService(t)
	id, err := service.CreateCard(testCardInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetPinned(id, true); err != nil {
		t.Fatal(err)
	}
	replacement := CardInput{Label: "Travel", Number: "2200123412341234", Network: "mir", Tags: []string{}}
	if err := service.UpdateCard(id, replacement, nil); err != nil {
		t.Fatal(err)
	}
	if card := mustReadCard(t, service, id); !reflect.DeepEqual(card.CardInput, replacement) {
		t.Fatalf("card after the update = %+v", card.CardInput)
	}
	if cards, err := service.ListCards(); err != nil || !cards[0].Pinned || cards[0].Network != "mir" || cards[0].LastFour != "1234" {
		t.Fatalf("cards after the update = %+v, error = %v", cards, err)
	}
}

func TestSiteIconIsServedForACardsBankHost(t *testing.T) {
	sites := newRecordingSites(t)
	service := newReadyServiceWithIcons(t, sites, filepath.Join(t.TempDir(), "icons"))
	if _, err := service.CreateCard(testCardInput(), nil); err != nil {
		t.Fatal(err)
	}
	if icon, err := service.SiteIcon("jyskebank.dk"); err != nil || icon.Image == "" {
		t.Fatalf("icon of a card's bank = %+v, %v", icon, err)
	}
	if asked := sites.sites(); !slices.Equal(asked, []string{"jyskebank.dk"}) {
		t.Fatalf("sites contacted = %q", asked)
	}
}

func TestCardLimitsMatchTheVault(t *testing.T) {
	limits, err := (&Service{}).GetCardLimits()
	if err != nil {
		t.Fatal(err)
	}
	want := CardLimits{
		Label: vault.MaxLabelLength, Holder: vault.MaxCardHolderLength,
		NumberMin: vault.MinCardNumberLength, NumberMax: vault.MaxCardNumberLength,
		SecurityCodeMin: vault.MinSecurityCodeLength, SecurityCodeMax: vault.MaxSecurityCodeLength,
		PINMin: vault.MinCardPINLength, PINMax: vault.MaxCardPINLength,
		BankName: vault.MaxBankNameLength, BankSite: vault.MaxOriginLength,
		Street: vault.MaxStreetLength, City: vault.MaxCityLength, Region: vault.MaxRegionLength,
		PostalCode: vault.MaxPostalCodeLength, Country: vault.MaxCountryLength, Notes: vault.MaxNotesLength,
	}
	if limits != want {
		t.Fatalf("limits = %+v", limits)
	}
}
