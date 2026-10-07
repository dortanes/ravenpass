package extensionaccess

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/app/verification"
	"github.com/dortanes/ravenpass/packages/vault"
)

var travelID = vault.ID{0xc4}

// FillableCards and FillCard answer err alone, for the tests that never reach a card.
func (f *fakeCredentials) FillableCards() ([]vault.Entry, error) { return nil, f.err }

func (f *fakeCredentials) FillCard(vault.ID) (vault.Card, error) { return vault.Card{}, f.err }

// cardCredentials holds the Travel card, with a linked billing address, and records the cards it fills and captures.
type cardCredentials struct {
	*fakeCredentials
	filled   []vault.ID
	offer    vaultservice.CaptureOffer
	created  bool
	captured []vaultservice.CardCapture
	choices  []vaultservice.CaptureChoice
	groups   []string
}

func (c *cardCredentials) FillableCards() ([]vault.Entry, error) {
	return []vault.Entry{{
		ID: travelID, Kind: vault.KindCard, Label: "Travel", Detail: "Example Bank", Site: "bank.example", ExpiresOn: "2029-08-31",
		Card: vault.CardFace{Network: vault.NetworkVisa, LastFour: "1111", Color: "#00a0e1"},
	}}, c.err
}

func (c *cardCredentials) FillCard(id vault.ID) (vault.Card, error) {
	c.filled = append(c.filled, id)
	return vault.Card{
		ID: id,
		CardInput: vault.CardInput{
			Label: "Travel", Holder: "Alex Example", Number: "4111111111111111", Expiry: "2029-08", SecurityCode: "739", PIN: "4829",
			Billing: &vault.Address{Street: "Own street"}, BillingLink: &vault.AddressLink{Identity: alexID, Address: passportID},
		},
		Linked: &vault.Address{Street: "1 Example Street", City: "Springfield", PostalCode: "12345", Country: "US"},
	}, c.err
}

func (c *cardCredentials) CardCaptureOffer(capture vaultservice.CardCapture) (vaultservice.CaptureOffer, error) {
	c.captured = append(c.captured, capture)
	return c.offer, c.err
}

func (c *cardCredentials) SaveCardCapture(capture vaultservice.CardCapture, choice vaultservice.CaptureChoice, group string) (bool, error) {
	c.captured = append(c.captured, capture)
	c.choices = append(c.choices, choice)
	c.groups = append(c.groups, group)
	return c.created, c.err
}

func cardAccess(t *testing.T, credentials *cardCredentials, verifier *fakeVerifier, choices *fakeChoices) *Access {
	t.Helper()
	access, err := New(credentials, &fakeIdentities{}, &fakeIcons{}, verifier, choices, &fakeUnlocks{})
	if err != nil {
		t.Fatal(err)
	}
	return access
}

const checkout = "https://shop.example.com"

func TestCardsListTheIndexFaces(t *testing.T) {
	access := cardAccess(t, &cardCredentials{fakeCredentials: &fakeCredentials{}}, &fakeVerifier{}, &fakeChoices{})
	cards, err := access.Cards()
	if err != nil {
		t.Fatal(err)
	}
	want := []linkproto.CardOption{{
		ID: travelID.String(), Label: "Travel", BankName: "Example Bank", Site: "bank.example", Network: "visa", LastFour: "1111",
		Color: "#00a0e1", ExpiresOn: "2029-08-31",
	}}
	if !reflect.DeepEqual(cards, want) {
		t.Fatalf("cards = %+v", cards)
	}
	locked := cardAccess(t, &cardCredentials{fakeCredentials: &fakeCredentials{err: vaultservice.ErrNotReady}}, &fakeVerifier{}, &fakeChoices{})
	if _, err := locked.Cards(); !errors.Is(err, linkserver.ErrLocked) {
		t.Fatalf("cards while locked: %v", err)
	}
}

func TestEveryCardFillWaitsForThePersonWhateverTheOwnerChose(t *testing.T) {
	credentials := &cardCredentials{fakeCredentials: &fakeCredentials{}}
	verifier := &fakeVerifier{method: verification.MethodDevice}
	access := cardAccess(t, credentials, verifier, &fakeChoices{confirmFills: false})
	var asked []linkproto.Progress
	for range 2 {
		fill, err := access.FillCard(context.Background(), travelID.String(), checkout, func(progress linkproto.Progress) { asked = append(asked, progress) })
		if err != nil {
			t.Fatal(err)
		}
		want := linkproto.CardFill{
			Holder: "Alex Example", Number: "4111111111111111", Expiry: "2029-08", SecurityCode: "739",
			Billing: &linkproto.BillingAddress{Street: "1 Example Street", City: "Springfield", PostalCode: "12345", Country: "US"},
		}
		if !reflect.DeepEqual(fill, want) {
			t.Fatalf("fill = %+v", fill)
		}
	}
	wantReasons := []confirmation.Reason{confirmation.FillingCard("shop.example.com", "Travel"), confirmation.FillingCard("shop.example.com", "Travel")}
	if !slices.Equal(verifier.reasons, wantReasons) || !slices.Equal(asked, []linkproto.Progress{linkproto.ProgressConfirmOnDevice, linkproto.ProgressConfirmOnDevice}) {
		t.Fatalf("reasons = %+v, asked = %q", verifier.reasons, asked)
	}
	if !slices.Equal(credentials.filled, []vault.ID{travelID, travelID}) {
		t.Fatalf("filled = %v", credentials.filled)
	}
}

func TestAVaultWithNoWayToVerifyFillsTheCardWithoutAsking(t *testing.T) {
	credentials := &cardCredentials{fakeCredentials: &fakeCredentials{}}
	access := cardAccess(t, credentials, &fakeVerifier{err: verification.ErrUnverifiable}, &fakeChoices{})
	fill, err := access.FillCard(context.Background(), travelID.String(), checkout, func(linkproto.Progress) {})
	if err != nil || fill.Number != "4111111111111111" {
		t.Fatalf("fill = %+v, error = %v", fill, err)
	}
}

func TestADeclinedCardFillReleasesNothing(t *testing.T) {
	for name, verifier := range map[string]*fakeVerifier{
		"declined":  {err: verification.ErrDeclined},
		"cancelled": {err: context.Canceled},
	} {
		credentials := &cardCredentials{fakeCredentials: &fakeCredentials{}}
		access := cardAccess(t, credentials, verifier, &fakeChoices{})
		if _, err := access.FillCard(context.Background(), travelID.String(), checkout, func(linkproto.Progress) {}); err == nil {
			t.Fatalf("a %s fill succeeded", name)
		}
		if len(credentials.filled) != 0 {
			t.Fatalf("a %s fill read the card", name)
		}
	}
}

func TestACardTheListingDoesNotHoldIsNotFoundBeforeAnyoneIsAsked(t *testing.T) {
	credentials := &cardCredentials{fakeCredentials: &fakeCredentials{}}
	verifier := &fakeVerifier{}
	access := cardAccess(t, credentials, verifier, &fakeChoices{})
	for _, card := range []string{"travel", vault.ID{0xee}.String()} {
		if _, err := access.FillCard(context.Background(), card, checkout, func(linkproto.Progress) {}); !errors.Is(err, linkserver.ErrNotFound) {
			t.Fatalf("filling %q: %v", card, err)
		}
	}
	if len(verifier.reasons) != 0 || len(credentials.filled) != 0 {
		t.Fatalf("an unknown card asked %+v and read %v", verifier.reasons, credentials.filled)
	}
}

var typedCard = linkserver.Capture{Origin: checkout, Card: &linkserver.CardCapture{
	Holder: "Alex Example", Number: "4111111111111111", Expiry: "2032-01", SecurityCode: "739", Network: "visa",
}}

func TestACardCaptureOfferShowsTheTypedCardsFace(t *testing.T) {
	credentials := &cardCredentials{fakeCredentials: &fakeCredentials{}, offer: vaultservice.CaptureOffer{
		Targets: []vaultservice.Target{{ID: travelID, Label: "Travel", Action: vaultservice.TargetUpdate}}, Suggested: travelID,
	}}
	access := cardAccess(t, credentials, &fakeVerifier{}, &fakeChoices{})
	offer, err := access.CaptureOffer(typedCard)
	if err != nil {
		t.Fatal(err)
	}
	want := linkproto.CaptureOffer{
		Kind: linkproto.CaptureCard, State: linkproto.CaptureReady, Site: "shop.example.com", Card: &linkproto.CardFace{Network: "visa", LastFour: "1111"},
		Targets: []linkproto.SaveTarget{{Credential: travelID.String(), Label: "Travel", Action: linkproto.SaveUpdate}}, Suggested: travelID.String(),
	}
	if !reflect.DeepEqual(offer, want) {
		t.Fatalf("offer = %+v", offer)
	}
	wantCapture := vaultservice.CardCapture{Holder: "Alex Example", Number: "4111111111111111", Expiry: "2032-01", SecurityCode: "739", Network: vault.NetworkVisa}
	if !slices.Equal(credentials.captured, []vaultservice.CardCapture{wantCapture}) {
		t.Fatalf("captured = %+v", credentials.captured)
	}

	unknown := linkserver.Capture{Origin: checkout, Card: &linkserver.CardCapture{Number: "4111111111111111", Network: "amex"}}
	if offer, err := access.CaptureOffer(unknown); err != nil || offer.State != linkproto.CaptureNone || len(credentials.captured) != 1 {
		t.Fatalf("offer for an unknown network = %+v, error = %v", offer, err)
	}
	credentials.err = vaultservice.ErrNotReady
	if offer, err := access.CaptureOffer(typedCard); err != nil || offer.State != linkproto.CaptureLocked || offer.Kind != linkproto.CaptureCard {
		t.Fatalf("offer while locked = %+v, error = %v", offer, err)
	}
}

func TestACardCaptureSavesWhereThePersonChose(t *testing.T) {
	credentials := &cardCredentials{fakeCredentials: &fakeCredentials{}, created: true}
	access := cardAccess(t, credentials, &fakeVerifier{}, &fakeChoices{group: "wallet"})
	saved, err := access.SaveCapture(typedCard, linkserver.SaveChoice{Name: "Visa 1111"})
	if err != nil || saved.Saved != linkproto.SavedCreated {
		t.Fatalf("saved = %+v, error = %v", saved, err)
	}
	credentials.created = false
	saved, err = access.SaveCapture(typedCard, linkserver.SaveChoice{Target: travelID.String()})
	if err != nil || saved.Saved != linkproto.SavedUpdated {
		t.Fatalf("updated = %+v, error = %v", saved, err)
	}
	if !slices.Equal(credentials.choices, []vaultservice.CaptureChoice{{Name: "Visa 1111"}, {Target: travelID}}) || !slices.Equal(credentials.groups, []string{"wallet", "wallet"}) {
		t.Fatalf("choices = %+v, groups = %q", credentials.choices, credentials.groups)
	}
	credentials.err = vaultservice.ErrNameRefused
	if _, err := access.SaveCapture(typedCard, linkserver.SaveChoice{Name: " "}); !errors.Is(err, linkserver.ErrInvalidName) {
		t.Fatalf("a refused name: %v", err)
	}
}
