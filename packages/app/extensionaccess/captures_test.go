package extensionaccess

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/vault"
)

// fakeChoices holds the recorded group new passwords join and whether fills wait for confirmation.
type fakeChoices struct {
	group        string
	confirmFills bool
}

func (f *fakeChoices) DefaultGroup() string { return f.group }

func (f *fakeChoices) ConfirmExtensionFills() bool { return f.confirmFills }

// CaptureOffer and SaveCapture answer err alone, for the tests that never reach a capture.
func (f *fakeCredentials) CaptureOffer(vaultservice.Capture) (vaultservice.CaptureOffer, error) {
	return vaultservice.CaptureOffer{}, f.err
}

func (f *fakeCredentials) SaveCapture(vaultservice.Capture, vaultservice.CaptureChoice, string) (bool, error) {
	return false, f.err
}

// CardCaptureOffer and SaveCardCapture answer err alone, for the tests that never reach a card capture.
func (f *fakeCredentials) CardCaptureOffer(vaultservice.CardCapture) (vaultservice.CaptureOffer, error) {
	return vaultservice.CaptureOffer{}, f.err
}

func (f *fakeCredentials) SaveCardCapture(vaultservice.CardCapture, vaultservice.CaptureChoice, string) (bool, error) {
	return false, f.err
}

// capturingCredentials answers captures with offer and saves with created, recording what it is asked.
type capturingCredentials struct {
	*fakeCredentials
	offer    vaultservice.CaptureOffer
	created  bool
	captured []vaultservice.Capture
	choices  []vaultservice.CaptureChoice
	groups   []string
}

func (c *capturingCredentials) CaptureOffer(capture vaultservice.Capture) (vaultservice.CaptureOffer, error) {
	c.captured = append(c.captured, capture)
	return c.offer, c.err
}

func (c *capturingCredentials) SaveCapture(capture vaultservice.Capture, choice vaultservice.CaptureChoice, group string) (bool, error) {
	c.captured = append(c.captured, capture)
	c.choices = append(c.choices, choice)
	c.groups = append(c.groups, group)
	return c.created, c.err
}

func capturingAccess(t *testing.T, credentials *capturingCredentials, group string) *Access {
	t.Helper()
	access, err := New(credentials, &fakeIdentities{}, &fakeIcons{}, &fakeVerifier{}, &fakeChoices{group: group}, &fakeUnlocks{})
	if err != nil {
		t.Fatal(err)
	}
	return access
}

var (
	updateID  = vault.ID{0x0a}
	addSiteID = vault.ID{0x0b}
	typed     = linkserver.Capture{Origin: "https://login.example.com", Account: "alex", Password: "typed", Current: "old"}
)

func TestACaptureOfferNamesItsTargetsAndSuggestion(t *testing.T) {
	credentials := &capturingCredentials{fakeCredentials: &fakeCredentials{}, offer: vaultservice.CaptureOffer{
		Targets: []vaultservice.Target{
			{ID: updateID, Label: "Example", Account: "alex", Action: vaultservice.TargetUpdate, Tags: []string{"Personal"}},
			{ID: addSiteID, Label: "Mail", Account: "alex@example.test", Action: vaultservice.TargetAddSite},
		},
		Suggested: updateID,
	}}
	access := capturingAccess(t, credentials, "")
	offer, err := access.CaptureOffer(typed)
	if err != nil {
		t.Fatal(err)
	}
	want := linkproto.CaptureOffer{
		Kind: linkproto.CapturePassword, State: linkproto.CaptureReady, Site: "login.example.com", Account: "alex", Name: "login.example.com",
		Targets: []linkproto.SaveTarget{
			{Credential: updateID.String(), Label: "Example", Account: "alex", Action: linkproto.SaveUpdate, Tags: []string{"Personal"}},
			{Credential: addSiteID.String(), Label: "Mail", Account: "alex@example.test", Action: linkproto.SaveAddSite},
		},
		Suggested: updateID.String(),
	}
	if !reflect.DeepEqual(offer, want) {
		t.Fatalf("offer = %+v, want %+v", offer, want)
	}
	if want := []vaultservice.Capture{{Requester: vaultservice.OriginRequester(typed.Origin), Account: "alex", Password: "typed", Current: "old"}}; !reflect.DeepEqual(credentials.captured, want) {
		t.Fatalf("the vault compared %+v", credentials.captured)
	}

	credentials.offer = vaultservice.CaptureOffer{}
	offer, err = access.CaptureOffer(linkserver.Capture{Origin: "https://xn--bcher-kva.example", Password: "typed"})
	if err != nil {
		t.Fatal(err)
	}
	want = linkproto.CaptureOffer{Kind: linkproto.CapturePassword, State: linkproto.CaptureReady, Site: "xn--bcher-kva.example", Name: "bücher.example", Targets: []linkproto.SaveTarget{}}
	if !reflect.DeepEqual(offer, want) {
		t.Fatalf("offer for a new credential = %+v, want %+v", offer, want)
	}
}

func TestACaptureOfferSaysWhenThereIsNothingToOfferOrTheVaultIsLocked(t *testing.T) {
	credentials := &capturingCredentials{fakeCredentials: &fakeCredentials{}, offer: vaultservice.CaptureOffer{Nothing: true}}
	access := capturingAccess(t, credentials, "")
	offer, err := access.CaptureOffer(typed)
	if err != nil {
		t.Fatal(err)
	}
	want := linkproto.CaptureOffer{Kind: linkproto.CapturePassword, State: linkproto.CaptureNone, Site: "login.example.com", Account: "alex", Name: "login.example.com", Targets: []linkproto.SaveTarget{}}
	if !reflect.DeepEqual(offer, want) {
		t.Fatalf("offer with nothing to offer = %+v", offer)
	}

	credentials.offer, credentials.err = vaultservice.CaptureOffer{}, vaultservice.ErrNotReady
	offer, err = access.CaptureOffer(linkserver.Capture{Origin: "https://exa_mple.com", Account: "alex", Password: "typed"})
	if err != nil {
		t.Fatal(err)
	}
	want = linkproto.CaptureOffer{Kind: linkproto.CapturePassword, State: linkproto.CaptureLocked, Site: "https://exa_mple.com", Account: "alex", Name: "https://exa_mple.com", Targets: []linkproto.SaveTarget{}}
	if !reflect.DeepEqual(offer, want) {
		t.Fatalf("offer while locked = %+v", offer)
	}

	failure := errors.New("usage record unreadable")
	credentials.err = failure
	if _, err := access.CaptureOffer(typed); !errors.Is(err, failure) {
		t.Fatalf("a vault failure: got %v", err)
	}
}

func TestSaveCaptureSavesWhereThePersonChoseWithTheRecordedGroup(t *testing.T) {
	credentials := &capturingCredentials{fakeCredentials: &fakeCredentials{}, created: true}
	access := capturingAccess(t, credentials, "0f0e0d0c0b0a09080706050403020100")
	saved, err := access.SaveCapture(typed, linkserver.SaveChoice{Account: "alex@example.test", Name: "Example"})
	if err != nil || saved != (linkproto.Saved{Saved: linkproto.SavedCreated}) {
		t.Fatalf("save as new = %+v, error = %v", saved, err)
	}
	credentials.created = false
	saved, err = access.SaveCapture(typed, linkserver.SaveChoice{Target: updateID.String(), Account: "ignored", Name: "ignored"})
	if err != nil || saved != (linkproto.Saved{Saved: linkproto.SavedUpdated}) {
		t.Fatalf("save to a target = %+v, error = %v", saved, err)
	}
	wantChoices := []vaultservice.CaptureChoice{
		{Account: "alex@example.test", Name: "Example"},
		{Target: updateID, Account: "ignored", Name: "ignored"},
	}
	if !slices.Equal(credentials.choices, wantChoices) {
		t.Fatalf("choices = %+v", credentials.choices)
	}
	if !slices.Equal(credentials.groups, []string{"0f0e0d0c0b0a09080706050403020100", "0f0e0d0c0b0a09080706050403020100"}) {
		t.Fatalf("groups = %q", credentials.groups)
	}
	if !reflect.DeepEqual(credentials.captured[0], serviceCapture(typed)) {
		t.Fatalf("saved capture = %+v", credentials.captured[0])
	}

	credentials.choices = nil
	for _, target := range []string{"mail", updateID.String() + "00"} {
		if _, err := access.SaveCapture(typed, linkserver.SaveChoice{Target: target}); !errors.Is(err, linkserver.ErrNotFound) {
			t.Fatalf("save to %q: got %v, want ErrNotFound", target, err)
		}
	}
	if len(credentials.choices) != 0 {
		t.Fatalf("unreadable targets reached the vault: %+v", credentials.choices)
	}
}

func TestSaveCaptureRefusalsBecomeTheServersRefusals(t *testing.T) {
	failure := errors.New("vault file unwritable")
	tests := []struct {
		err  error
		want error
	}{
		{vaultservice.ErrNotReady, linkserver.ErrLocked},
		{vault.ErrNotFound, linkserver.ErrNotFound},
		{fmt.Errorf("%w: %w", vaultservice.ErrAccountRefused, vault.ErrInvalidInput), linkserver.ErrInvalidAccount},
		{fmt.Errorf("%w: %w", vaultservice.ErrNameRefused, vault.ErrInvalidInput), linkserver.ErrInvalidName},
		{failure, failure},
	}
	for _, test := range tests {
		credentials := &capturingCredentials{fakeCredentials: &fakeCredentials{err: test.err}}
		access := capturingAccess(t, credentials, "")
		if _, err := access.SaveCapture(typed, linkserver.SaveChoice{Name: "Example"}); !errors.Is(err, test.want) {
			t.Fatalf("save refused with %v: got %v, want %v", test.err, err, test.want)
		}
	}
}
