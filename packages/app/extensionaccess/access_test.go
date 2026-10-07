package extensionaccess

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/vault"
)

var mailID = vault.ID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

// fakeCredentials answers every call with its fields and records the calls and their requesters.
type fakeCredentials struct {
	unlocked    bool
	suggestions []vaultservice.Suggestion
	matched     vaultservice.Suggestion
	fill        vaultservice.Fill
	code        vault.OneTimeCode
	named       map[string]bool
	err         error
	calls       []string
	requesters  []vaultservice.Requester
	purposes    []vaultservice.Purpose
}

func (f *fakeCredentials) Unlocked() bool { return f.unlocked }

func (f *fakeCredentials) Suggestions(requester vaultservice.Requester, purpose vaultservice.Purpose) ([]vaultservice.Suggestion, error) {
	f.calls = append(f.calls, "suggest")
	f.requesters = append(f.requesters, requester)
	f.purposes = append(f.purposes, purpose)
	return f.suggestions, f.err
}

func (f *fakeCredentials) MatchingCredential(id vault.ID, requester vaultservice.Requester) (vaultservice.Suggestion, error) {
	f.calls = append(f.calls, "match "+id.String())
	f.requesters = append(f.requesters, requester)
	return f.matched, f.err
}

func (f *fakeCredentials) FillCredential(id vault.ID, requester vaultservice.Requester) (vaultservice.Fill, error) {
	f.calls = append(f.calls, "fill "+id.String())
	f.requesters = append(f.requesters, requester)
	return f.fill, f.err
}

func (f *fakeCredentials) OneTimeCode(id vault.ID, requester vaultservice.Requester) (vault.OneTimeCode, error) {
	f.calls = append(f.calls, "code "+id.String())
	f.requesters = append(f.requesters, requester)
	return f.code, f.err
}

func (f *fakeCredentials) AddWebsite(id vault.ID, origin string) error {
	f.calls = append(f.calls, "add "+id.String()+" "+origin)
	return f.err
}

// wantPages fails unless the calls credentials got named the pages at origins, in order.
func wantPages(t *testing.T, credentials *fakeCredentials, origins ...string) {
	t.Helper()
	want := make([]vaultservice.Requester, len(origins))
	for i, origin := range origins {
		want[i] = vaultservice.OriginRequester(origin)
	}
	if !reflect.DeepEqual(credentials.requesters, want) {
		t.Fatalf("requesters = %+v, want the pages at %q", credentials.requesters, origins)
	}
}

func (f *fakeCredentials) NamesSite(site string) (bool, error) {
	f.calls = append(f.calls, "names "+site)
	return f.named[site], f.err
}

// fakeIcons holds the icon of each site, as a base64 PNG.
type fakeIcons struct {
	icons map[string]string
	err   error
	asked []string
}

func (f *fakeIcons) Icon(site string) (string, error) {
	f.asked = append(f.asked, site)
	return f.icons[site], f.err
}

// fakeUnlocks records the requester of each unlock request posted to it.
type fakeUnlocks struct {
	posted []confirmation.Requester
}

func (f *fakeUnlocks) PostUnlock(requester confirmation.Requester) string {
	f.posted = append(f.posted, requester)
	return "unlock"
}

func newAccess(t *testing.T, credentials *fakeCredentials, icons *fakeIcons) (*Access, *fakeUnlocks) {
	t.Helper()
	unlocks := &fakeUnlocks{}
	access, err := New(credentials, &fakeIdentities{}, icons, &fakeVerifier{}, &fakeChoices{}, unlocks)
	if err != nil {
		t.Fatal(err)
	}
	return access, unlocks
}

// unasked fails the test when a fill reports that it asks the person anything.
func unasked(t *testing.T) func(linkproto.Progress) {
	return func(progress linkproto.Progress) { t.Errorf("a fill that needs no confirmation reported %q", progress) }
}

func orangeIcon(t *testing.T) string {
	t.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for x := range 16 {
		for y := range 16 {
			picture.SetNRGBA(x, y, color.NRGBA{R: 230, G: 120, B: 20, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(encoded.Bytes())
}

func TestNewRequiresEveryPart(t *testing.T) {
	credentials, identities, icons, verifier, choices, unlocks := &fakeCredentials{}, &fakeIdentities{}, &fakeIcons{}, &fakeVerifier{}, &fakeChoices{}, &fakeUnlocks{}
	for name, build := range map[string]func() (*Access, error){
		"no vault":      func() (*Access, error) { return New(nil, identities, icons, verifier, choices, unlocks) },
		"no identities": func() (*Access, error) { return New(credentials, nil, icons, verifier, choices, unlocks) },
		"no icons":      func() (*Access, error) { return New(credentials, identities, nil, verifier, choices, unlocks) },
		"no verifier":   func() (*Access, error) { return New(credentials, identities, icons, nil, choices, unlocks) },
		"no choices":    func() (*Access, error) { return New(credentials, identities, icons, verifier, nil, unlocks) },
		"no unlocks":    func() (*Access, error) { return New(credentials, identities, icons, verifier, choices, nil) },
	} {
		if _, err := build(); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

func TestUnlockedReadsTheVault(t *testing.T) {
	credentials := &fakeCredentials{}
	access, _ := newAccess(t, credentials, &fakeIcons{})
	if access.Unlocked() {
		t.Fatal("a locked vault reads as unlocked")
	}
	credentials.unlocked = true
	if !access.Unlocked() {
		t.Fatal("an open vault reads as locked")
	}
}

func TestShowUnlockAsksInThePanelOnlyWhileLocked(t *testing.T) {
	credentials := &fakeCredentials{}
	access, unlocks := newAccess(t, credentials, &fakeIcons{})
	access.ShowUnlock()
	if !slices.Equal(unlocks.posted, []confirmation.Requester{confirmation.RequesterExtension}) {
		t.Fatalf("a locked vault posted unlock requests for %v", unlocks.posted)
	}
	credentials.unlocked = true
	access.ShowUnlock()
	if len(unlocks.posted) != 1 {
		t.Fatal("an open vault posted an unlock request")
	}
}

func TestSuggestAnswersTheVaultsSuggestions(t *testing.T) {
	credentials := &fakeCredentials{suggestions: []vaultservice.Suggestion{
		{ID: mailID, Label: "Mail", Account: "alex", Site: "example.com", Exact: true},
		{ID: vault.ID{0xff}, Label: "Shop", Account: "alex@example.test", Site: "shop.example.com"},
	}}
	access, _ := newAccess(t, credentials, &fakeIcons{})
	got, err := access.Suggest("https://example.com", linkproto.PurposeSignIn)
	if err != nil {
		t.Fatal(err)
	}
	want := []linkproto.Suggestion{
		{ID: "0102030405060708090a0b0c0d0e0f10", Label: "Mail", Account: "alex", Site: "example.com", Exact: true},
		{ID: "ff000000000000000000000000000000", Label: "Shop", Account: "alex@example.test", Site: "shop.example.com"},
	}
	if !reflect.DeepEqual(got, want) || !slices.Equal(credentials.calls, []string{"suggest"}) {
		t.Fatalf("suggestions = %+v after %q", got, credentials.calls)
	}
	wantPages(t, credentials, "https://example.com")
	credentials.suggestions = []vaultservice.Suggestion{{ID: mailID, Label: "Mail", Account: "alex", Site: "example.com", Exact: true, Code: vault.CodeFace{Digits: 8, Period: 60}}}
	coded, err := access.Suggest("https://example.com", linkproto.PurposeCode)
	if err != nil {
		t.Fatal(err)
	}
	if want := []linkproto.Suggestion{{ID: "0102030405060708090a0b0c0d0e0f10", Label: "Mail", Account: "alex", Site: "example.com", Exact: true, Digits: 8, Period: 60}}; !reflect.DeepEqual(coded, want) {
		t.Fatalf("code suggestions = %+v", coded)
	}
	credentials.suggestions = nil
	if got, err := access.Suggest("https://example.org", linkproto.PurposeCode); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no suggestion = %#v, error = %v", got, err)
	}
	if !slices.Equal(credentials.purposes, []vaultservice.Purpose{vaultservice.PurposeSignIn, vaultservice.PurposeCode, vaultservice.PurposeCode}) {
		t.Fatalf("purposes = %v", credentials.purposes)
	}
}

func TestSuggestRefusesAPurposeItDoesNotKnow(t *testing.T) {
	credentials := &fakeCredentials{}
	access, _ := newAccess(t, credentials, &fakeIcons{})
	for _, purpose := range []linkproto.Purpose{"", "Code", "password"} {
		if _, err := access.Suggest("https://example.com", purpose); err == nil {
			t.Fatalf("purpose %q was accepted", purpose)
		}
	}
	if len(credentials.calls) != 0 {
		t.Fatalf("unknown purposes reached the vault: %q", credentials.calls)
	}
}

func TestOneTimeCodeAnswersTheCredentialsCurrentCode(t *testing.T) {
	credentials := &fakeCredentials{code: vault.OneTimeCode{Code: "287082", Digits: 6, Period: 30, ExpiresAt: time.Unix(1_790_000_010, 0).UTC()}}
	access, _ := newAccess(t, credentials, &fakeIcons{})
	code, err := access.OneTimeCode(context.Background(), mailID.String(), "https://example.com", unasked(t))
	if err != nil {
		t.Fatal(err)
	}
	if code != (linkproto.OneTimeCode{Code: "287082", Digits: 6, Period: 30, ExpiresAt: 1_790_000_010_000}) {
		t.Fatalf("code = %+v", code)
	}
	if !slices.Equal(credentials.calls, []string{"code 0102030405060708090a0b0c0d0e0f10"}) {
		t.Fatalf("calls = %q", credentials.calls)
	}
	wantPages(t, credentials, "https://example.com")
	credentials.calls = nil
	for _, credential := range []string{"", "mail", mailID.String() + "00"} {
		if _, err := access.OneTimeCode(context.Background(), credential, "https://example.com", unasked(t)); !errors.Is(err, linkserver.ErrNotFound) {
			t.Fatalf("code of %q: got %v, want ErrNotFound", credential, err)
		}
	}
	if len(credentials.calls) != 0 {
		t.Fatalf("unreadable identifiers reached the vault: %q", credentials.calls)
	}
}

func TestFillAnswersTheCredentialsValues(t *testing.T) {
	credentials := &fakeCredentials{fill: vaultservice.Fill{Login: "alex", Email: "alex@example.test", Password: "secret"}}
	access, _ := newAccess(t, credentials, &fakeIcons{})
	fill, err := access.Fill(context.Background(), browserExtension, mailID.String(), "https://example.com", unasked(t))
	if err != nil {
		t.Fatal(err)
	}
	if fill != (linkproto.Fill{Login: "alex", Email: "alex@example.test", Password: "secret"}) {
		t.Fatalf("fill = %+v", fill)
	}
	if !slices.Equal(credentials.calls, []string{"fill 0102030405060708090a0b0c0d0e0f10"}) {
		t.Fatalf("calls = %q", credentials.calls)
	}
	wantPages(t, credentials, "https://example.com")
	credentials.calls = nil
	for _, credential := range []string{"", "mail", "0102", mailID.String() + "00", "0102030405060708090a0b0c0d0e0fzz"} {
		if _, err := access.Fill(context.Background(), browserExtension, credential, "https://example.com", unasked(t)); !errors.Is(err, linkserver.ErrNotFound) {
			t.Fatalf("fill of %q: got %v, want ErrNotFound", credential, err)
		}
	}
	if len(credentials.calls) != 0 {
		t.Fatalf("unreadable identifiers reached the vault: %q", credentials.calls)
	}
}

func TestVaultRefusalsBecomeTheServersRefusals(t *testing.T) {
	failure := errors.New("usage record could not be written")
	tests := []struct {
		err  error
		want error
	}{
		{vaultservice.ErrNotReady, linkserver.ErrLocked},
		{vault.ErrNotFound, linkserver.ErrNotFound},
		{vaultservice.ErrNoMatch, linkserver.ErrNoMatch},
		{vault.ErrNoTOTP, linkserver.ErrNoCode},
		{failure, failure},
	}
	for _, test := range tests {
		credentials := &fakeCredentials{err: test.err}
		access, _ := newAccess(t, credentials, &fakeIcons{})
		if _, err := access.Suggest("https://example.com", linkproto.PurposeSignIn); !errors.Is(err, test.want) {
			t.Fatalf("suggest refused with %v: got %v, want %v", test.err, err, test.want)
		}
		if _, err := access.Fill(context.Background(), browserExtension, mailID.String(), "https://example.com", unasked(t)); !errors.Is(err, test.want) {
			t.Fatalf("fill refused with %v: got %v, want %v", test.err, err, test.want)
		}
		if _, err := access.OneTimeCode(context.Background(), mailID.String(), "https://example.com", unasked(t)); !errors.Is(err, test.want) {
			t.Fatalf("code refused with %v: got %v, want %v", test.err, err, test.want)
		}
		if _, err := access.SiteIcon("example.com"); !errors.Is(err, test.want) {
			t.Fatalf("icon refused with %v: got %v, want %v", test.err, err, test.want)
		}
	}
}

func TestSiteIconAnswersOnlyANamedSite(t *testing.T) {
	orange := orangeIcon(t)
	credentials := &fakeCredentials{named: map[string]bool{"example.com": true, "quiet.example": true}}
	icons := &fakeIcons{icons: map[string]string{"example.com": orange, "unnamed.example": orange}}
	access, _ := newAccess(t, credentials, icons)
	icon, err := access.SiteIcon("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if icon != (linkproto.Icon{Image: orange, Tint: "#e67814"}) {
		t.Fatalf("icon = %+v", icon)
	}
	for _, site := range []string{"unnamed.example", "quiet.example"} {
		if icon, err := access.SiteIcon(site); err != nil || icon != (linkproto.Icon{}) {
			t.Fatalf("icon of %s = %+v, error = %v", site, icon, err)
		}
	}
	if !slices.Equal(icons.asked, []string{"example.com", "quiet.example"}) {
		t.Fatalf("icons asked for %q", icons.asked)
	}
	icons.err = vaultservice.ErrNotReady
	if _, err := access.SiteIcon("example.com"); !errors.Is(err, linkserver.ErrLocked) {
		t.Fatalf("icon of a vault that locked meanwhile: got %v, want ErrLocked", err)
	}
}
