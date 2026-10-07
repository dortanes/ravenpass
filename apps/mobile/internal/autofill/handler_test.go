package autofill

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/dortanes/ravenpass/packages/app/api"
	"github.com/dortanes/ravenpass/packages/app/autofill"
	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/ownerauth"
	"github.com/dortanes/ravenpass/packages/app/unlock"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/vault"
)

// fakeService is the vault's autofill service with fixed answers, recording what it was asked.
type fakeService struct {
	open        bool
	sites       []string
	suggestions []autofill.Suggestion
	found       []autofill.Suggestion
	login       autofill.Login
	code        autofill.Code
	offer       autofill.Offer
	created     bool
	err         error

	// codeSuggestions answers Suggest for a code field when set.
	codeSuggestions []autofill.Suggestion
	// passkeyChoices belong to the relying party passkeyRPID.
	passkeyRPID    string
	passkeyChoices []autofill.PasskeyChoice

	suggested []autofill.Requester
	codes     []bool
	searched  []string
	filled    []autofill.Requester
	linked    []autofill.App
	added     []autofill.Requester
	held      []autofill.Capture
	saved     []autofill.Choice
	listed    []string
	signIns   []autofill.PasskeySignIn
	creations []autofill.PasskeyCreation
}

func (s *fakeService) Open() bool { return s.open }

func (s *fakeService) Suggest(r autofill.Requester, code bool) ([]autofill.Suggestion, error) {
	s.suggested = append(s.suggested, r)
	s.codes = append(s.codes, code)
	if code && s.codeSuggestions != nil {
		return s.codeSuggestions, s.err
	}
	return s.suggestions, s.err
}

func (s *fakeService) Fill(_ string, r autofill.Requester) (autofill.Login, error) {
	s.filled = append(s.filled, r)
	return s.login, s.err
}

func (s *fakeService) OneTimeCode(_ string, r autofill.Requester) (autofill.Code, error) {
	s.filled = append(s.filled, r)
	return s.code, s.err
}

func (s *fakeService) Search(query string, _ bool) ([]autofill.Suggestion, error) {
	s.searched = append(s.searched, query)
	return s.found, s.err
}

func (s *fakeService) Sites() ([]string, error) { return s.sites, s.err }

func (s *fakeService) Link(_ string, app autofill.App) error {
	s.linked = append(s.linked, app)
	return s.err
}

func (s *fakeService) AddSite(_ string, r autofill.Requester) error {
	s.added = append(s.added, r)
	return s.err
}

func (s *fakeService) Hold(c autofill.Capture) (autofill.Offer, error) {
	s.held = append(s.held, c)
	return s.offer, s.err
}

func (s *fakeService) Save(_ string, choice autofill.Choice) (bool, error) {
	s.saved = append(s.saved, choice)
	return s.created, s.err
}

// Passkeys lists the passkeyChoices of passkeyRPID that allowed lists, or all when it lists none.
func (s *fakeService) Passkeys(rpID string, allowed [][]byte) ([]autofill.PasskeyChoice, error) {
	s.listed = append(s.listed, rpID)
	if s.err != nil || rpID != s.passkeyRPID {
		return nil, s.err
	}
	var found []autofill.PasskeyChoice
	for _, choice := range s.passkeyChoices {
		if len(allowed) == 0 || slices.ContainsFunc(allowed, func(id []byte) bool { return bytes.Equal(id, choice.CredentialID) }) {
			found = append(found, choice)
		}
	}
	return found, nil
}

func (s *fakeService) HeldPasskey(rpID, id string, credentialID []byte) (autofill.PasskeyChoice, error) {
	found, err := s.Passkeys(rpID, [][]byte{credentialID})
	if err != nil {
		return autofill.PasskeyChoice{}, err
	}
	index := slices.IndexFunc(found, func(choice autofill.PasskeyChoice) bool { return choice.ID == id })
	if index < 0 {
		return autofill.PasskeyChoice{}, autofill.ErrNotFound
	}
	return found[index], nil
}

// CheckExclusions lists the excluded passkeys, as a creation that names any does.
func (s *fakeService) CheckExclusions(rpID string, exclude [][]byte) error {
	if len(exclude) == 0 {
		return nil
	}
	found, err := s.Passkeys(rpID, exclude)
	if err != nil {
		return err
	}
	if len(found) > 0 {
		return autofill.ErrPasskeyExcluded
	}
	return nil
}

func (s *fakeService) SignPasskey(signIn autofill.PasskeySignIn) (autofill.PasskeyAssertion, error) {
	s.signIns = append(s.signIns, signIn)
	return autofill.PasskeyAssertion{CredentialID: signIn.CredentialID, AuthenticatorData: []byte{1}, Signature: []byte{2}}, s.err
}

func (s *fakeService) CreatePasskey(creation autofill.PasskeyCreation) (autofill.CreatedPasskey, error) {
	s.creations = append(s.creations, creation)
	return autofill.CreatedPasskey{ID: "new", CredentialID: []byte{9}, AttestationObject: []byte{1}, Algorithm: -7}, s.err
}

// fakeVault's device unlock answers deviceErr, its PIN unlock pinErr, and its PIN check verifyErr.
type fakeVault struct {
	methods   vaultservice.Methods
	deviceErr error
	pinErr    error
	verifyErr error
	reasons   []string
	pins      []string
	verified  []string
}

func (v *fakeVault) UnlockMethods() (vaultservice.Methods, error) { return v.methods, nil }

func (v *fakeVault) Unlock(reason string) (vault.Head, error) {
	v.reasons = append(v.reasons, reason)
	return vault.Head{}, v.deviceErr
}

func (v *fakeVault) UnlockWithPIN(pin string) (vault.Head, error) {
	v.pins = append(v.pins, pin)
	return vault.Head{}, v.pinErr
}

func (v *fakeVault) VerifyPIN(pin string) error {
	v.verified = append(v.verified, pin)
	return v.verifyErr
}

// fakeOwner answers the device's prompt with err, recording its reasons.
type fakeOwner struct {
	err     error
	reasons []string
}

func (o *fakeOwner) AuthenticateOwner(_ context.Context, reason string) error {
	o.reasons = append(o.reasons, reason)
	return o.err
}

func words(reason confirmation.Reason) string {
	return fmt.Sprintf("%d %s %s", reason.Kind, reason.Site, reason.Account)
}

// fakeIcons is an icon cache holding icons by site, recording the sites it was asked for.
type fakeIcons struct {
	icons map[string]string
	err   error
	asked []string
}

func (i *fakeIcons) Cached(site string) (string, error) {
	i.asked = append(i.asked, site)
	return i.icons[site], i.err
}

type fakePage struct {
	icons      map[string]api.SiteIcon
	language   api.LanguageSettings
	size       api.InterfaceSize
	appearance api.Appearance
}

func (p *fakePage) SiteIcon(site string) (api.SiteIcon, error) {
	icon, ok := p.icons[site]
	if !ok {
		return api.SiteIcon{}, errors.New("no such site")
	}
	return icon, nil
}

func (p *fakePage) GetLanguage() (api.LanguageSettings, error) { return p.language, nil }

func (p *fakePage) GetInterfaceSize() (api.InterfaceSize, error) { return p.size, nil }

func (p *fakePage) GetAppearance() (api.Appearance, error) { return p.appearance, nil }

type harness struct {
	handler   *Handler
	service   *fakeService
	vault     *fakeVault
	owner     *fakeOwner
	icons     *fakeIcons
	page      *fakePage
	sites     *fakeSites
	opened    int
	follows   int
	requested int
	holds     int
}

func newHarness(service *fakeService, sites map[string]siteResponse) *harness {
	h := &harness{service: service, vault: &fakeVault{}, owner: &fakeOwner{}, icons: &fakeIcons{}, page: &fakePage{}, sites: newFakeSites(sites)}
	h.handler = NewHandler(Options{
		Service:   service,
		Vault:     h.vault,
		Owner:     h.owner,
		Icons:     h.icons,
		Page:      h.page,
		Links:     newLinks(h.sites),
		Reason:    func() string { return "Unlock Ravenpass to fill in" },
		Words:     words,
		Opened:    func() { h.opened++ },
		Follow:    func() { h.follows++ },
		Requested: func() { h.requested++ },
		Hold: func() func() {
			h.holds++
			return func() { h.holds-- }
		},
	})
	return h
}

func (h *harness) call(t *testing.T, request map[string]any, answer any) {
	t.Helper()
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(h.handler.Call(encoded), answer); err != nil {
		t.Fatal(err)
	}
}

func wireOf(app autofill.App) map[string]any {
	var signers []string
	for _, signer := range app.Signers {
		signers = append(signers, base64.StdEncoding.EncodeToString(signer[:]))
	}
	return map[string]any{"package": app.Package, "signers": signers}
}

func chrome(t *testing.T) autofill.App {
	t.Helper()
	digest, ok := fingerprint(browserCertificates["com.android.chrome"][0])
	if !ok {
		t.Fatal("Chrome's certificate digest is malformed")
	}
	return autofill.App{Package: "com.android.chrome", Signers: [][32]byte{digest}}
}

func asking(app autofill.App, fields ...field) map[string]any {
	return map[string]any{"op": "suggest", "app": wireOf(app), "fields": fields}
}

var signIn = []field{text(0, "login"), focus(password(1, "password"))}

func pageSignIn(domain string) []field {
	return []field{onPage(text(0, "login"), domain), focus(onPage(password(1, "password"), domain))}
}

func TestALockedVaultOffersOnlyTheUnlockEntry(t *testing.T) {
	h := newHarness(&fakeService{open: false}, nil)
	var answer suggestAnswer
	h.call(t, asking(signedApp("com.example.app", "key"), signIn...), &answer)
	if answer.Status != statusLocked || answer.Form == nil || answer.Form.Username != 0 || answer.Form.Password != 1 {
		t.Fatalf("answer %+v", answer)
	}
	if answer.Form.Save != nil || answer.Requester != nil || answer.Suggestions != nil || answer.Icons != nil || answer.Search {
		t.Fatalf("a locked vault answered more than an unlock entry: %+v", answer)
	}
	if len(h.service.suggested) != 0 || len(h.icons.asked) != 0 {
		t.Fatalf("a locked vault was asked for suggestions %+v and icons %q", h.service.suggested, h.icons.asked)
	}
}

func TestALockedVaultAnswersNothingForAFormThatOnlySaves(t *testing.T) {
	h := newHarness(&fakeService{open: false}, nil)
	var answer suggestAnswer
	h.call(t, asking(signedApp("com.example.app", "key"), text(0, "login"), focus(password(1, "new_password"))), &answer)
	if answer.Status != statusNone {
		t.Fatalf("answer %+v", answer)
	}
}

func TestOnlyAFormToFillOrASubmittedSignInFollowTheVaultFile(t *testing.T) {
	h := newHarness(&fakeService{open: true}, nil)
	app := map[string]any{"app": wireOf(signedApp("com.example.app", "key"))}
	var answer outcome
	h.call(t, asking(signedApp("com.example.app", "key"), signIn...), &answer)
	h.call(t, map[string]any{"op": "capture", "requester": app, "account": "alex", "password": "pw"}, &answer)
	h.call(t, map[string]any{"op": "search", "query": "example"}, &answer)
	h.call(t, map[string]any{"op": "methods"}, &answer)
	if h.follows != 2 {
		t.Fatalf("the vault file was followed %d times, want 2", h.follows)
	}
}

func TestASlowVaultFileLeavesTheRequestItsDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(&fakeService{open: true}, nil)
		release := make(chan struct{})
		h.handler.follow = func() { <-release }
		start := time.Now()
		var answer suggestAnswer
		h.call(t, asking(chrome(t), pageSignIn("example.com")...), &answer)
		if waited := time.Since(start); waited != followWait {
			t.Fatalf("the request waited %v for the file, want %v", waited, followWait)
		}
		if answer.Status != statusOK {
			t.Fatalf("answer %+v", answer)
		}
		close(release)
	})
}

func TestRequestsDuringASlowReadWaitOnIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(&fakeService{open: true}, nil)
		release := make(chan struct{})
		var reads atomic.Int32
		h.handler.follow = func() {
			reads.Add(1)
			<-release
		}
		for range 3 {
			var answer suggestAnswer
			h.call(t, asking(chrome(t), pageSignIn("example.com")...), &answer)
		}
		if reads.Load() != 1 {
			t.Fatalf("a slow read started %d reads", reads.Load())
		}
		close(release)
		synctest.Wait()
		var answer suggestAnswer
		h.call(t, asking(chrome(t), pageSignIn("example.com")...), &answer)
		if reads.Load() != 2 {
			t.Fatalf("a request after the read ended started %d reads in all", reads.Load())
		}
	})
}

func TestAScreenNoticeIsKnownWithoutTheVault(t *testing.T) {
	for request, want := range map[string][2]bool{
		`{"op":"shown"}`: {true, true}, `{"op":"hidden"}`: {false, true}, `{"op":"suggest"}`: {}, `not json`: {},
	} {
		if shown, notice := ScreenNotice([]byte(request)); [2]bool{shown, notice} != want {
			t.Errorf("%s: shown = %t, notice = %t", request, shown, notice)
		}
	}
	if string(Noted()) != `{"status":"ok"}` {
		t.Fatalf("a notice is answered %s", Noted())
	}
}

func TestScreensShownBeforeTheHandlerHoldTheLockUntilHidden(t *testing.T) {
	h := newHarness(&fakeService{}, nil)
	h.handler.ShowScreens(2)
	var answer outcome
	h.call(t, map[string]any{"op": "hidden"}, &answer)
	if h.holds != 1 {
		t.Fatalf("one of two early screens hidden leaves %d holds, want 1", h.holds)
	}
	h.call(t, map[string]any{"op": "hidden"}, &answer)
	if h.holds != 0 {
		t.Fatalf("both early screens hidden leave %d holds", h.holds)
	}
}

func TestATrustedBrowserAsksForThePagesOrigin(t *testing.T) {
	service := &fakeService{open: true, suggestions: []autofill.Suggestion{{ID: "a", Label: "Example", Account: "alex", Site: "example.com", Exact: true, Tags: []string{"Work"}}}}
	h := newHarness(service, nil)
	var answer suggestAnswer
	h.call(t, asking(chrome(t), pageSignIn("example.com")...), &answer)
	want := []suggestionWire{{ID: "a", Label: "Example", Account: "alex", Site: "example.com", Tags: []string{"Work"}}}
	if answer.Status != statusOK || !answer.Search || !reflect.DeepEqual(answer.Suggestions, want) {
		t.Fatalf("answer %+v", answer)
	}
	if answer.Requester == nil || answer.Requester.Origin != "https://example.com" {
		t.Fatalf("requester %+v", answer.Requester)
	}
	if len(service.suggested) != 1 || service.suggested[0].Origin != "https://example.com" || service.codes[0] {
		t.Fatalf("suggested for %+v", service.suggested)
	}
}

func TestAnAppClaimingABanksDomainAsksAsItself(t *testing.T) {
	bank := signedApp("com.bank.app", "bank key")
	for name, app := range map[string]autofill.App{
		"an app the bank's site does not name": signedApp("com.lookalike", "lookalike key"),
		"a copy of Chrome signed by another":   {Package: "com.android.chrome", Signers: signedApp("", "other key").Signers},
	} {
		t.Run(name, func(t *testing.T) {
			service := &fakeService{open: true, sites: []string{"bank.com"}}
			h := newHarness(service, map[string]siteResponse{
				"bank.com": served(statementFor(bank, "delegate_permission/common.get_login_creds")),
			})
			var answer suggestAnswer
			h.call(t, asking(app, pageSignIn("bank.com")...), &answer)
			if answer.Status != statusOK || len(service.suggested) != 1 {
				t.Fatalf("answer %+v", answer)
			}
			asked := service.suggested[0]
			if asked.Origin != "" || asked.App.Package != app.Package || len(asked.Sites) != 0 {
				t.Fatalf("the app asked as %+v", asked)
			}
		})
	}
}

func TestAWebViewPageOfAnAppItsSiteNamesAsksForThePagesOrigin(t *testing.T) {
	app := signedApp("com.example.app", "example key")
	service := &fakeService{open: true, sites: []string{"example.com"}}
	h := newHarness(service, map[string]siteResponse{
		"example.com": served(statementFor(app, "delegate_permission/common.get_login_creds")),
	})
	fields := []field{text(0, "login"), focus(onPage(password(1, "password"), "www.example.com"))}
	var answer suggestAnswer
	h.call(t, asking(app, fields...), &answer)
	if len(service.suggested) != 1 || service.suggested[0].Origin != "https://www.example.com" {
		t.Fatalf("suggested for %+v", service.suggested)
	}
	if answer.Form.Username != -1 || answer.Form.Password != 1 {
		t.Fatalf("a page's credential would fill the app's own field: %+v", answer.Form)
	}
}

func TestAnAppAsksWithTheSitesThatVerifyIt(t *testing.T) {
	app := signedApp("com.example.app", "example key")
	service := &fakeService{open: true, sites: []string{"example.com", "other.com", "offline.com"}}
	h := newHarness(service, map[string]siteResponse{
		"example.com": served(statementFor(app, "delegate_permission/common.handle_all_urls")),
		"other.com":   served(statementFor(signedApp("com.other", "other key"), "delegate_permission/common.get_login_creds")),
	})
	var answer suggestAnswer
	h.call(t, asking(app, signIn...), &answer)
	if len(service.suggested) != 1 || !slices.Equal(service.suggested[0].Sites, []string{"example.com"}) ||
		!reflect.DeepEqual(service.suggested[0].App, app) {
		t.Fatalf("suggested for %+v", service.suggested)
	}
	requester, ok := answer.Requester.requester()
	if !ok || !reflect.DeepEqual(requester, service.suggested[0]) {
		t.Fatalf("the answer's requester %+v does not read back", answer.Requester)
	}
}

func TestAnAppWithoutItsCertificatesGetsNoAnswer(t *testing.T) {
	h := newHarness(&fakeService{open: true}, nil)
	var answer suggestAnswer
	h.call(t, map[string]any{"op": "suggest", "app": map[string]any{"package": "com.example.app"}, "fields": signIn}, &answer)
	if answer.Status != statusNone {
		t.Fatalf("answer %+v", answer)
	}
}

func TestACodeFieldIsOfferedTheCredentialsWithCodesAndSearch(t *testing.T) {
	var boxes []field
	for i := range 6 {
		boxes = append(boxes, onPage(pageInput(i, map[string]string{"type": "text", "maxlength": "1", "name": "digit"}), "example.com"))
	}
	boxes[2] = focus(boxes[2])
	for name, test := range map[string]struct {
		fields []field
		code   []int
	}{
		"a declared code field": {
			fields: []field{focus(onPage(field{Index: 3, Hints: []string{"one-time-code"}, InputType: inputClassText, Visible: true}, "example.com"))},
			code:   []int{3},
		},
		"the only field of a page, named for a code": {
			fields: []field{focus(onPage(pageInput(0, map[string]string{"type": "text", "id": "auth-mfa-otpcode"}), "example.com"))},
			code:   []int{0},
		},
		"a split code field": {fields: boxes, code: []int{0, 1, 2, 3, 4, 5}},
	} {
		t.Run(name, func(t *testing.T) {
			service := &fakeService{
				open:            true,
				suggestions:     []autofill.Suggestion{{ID: "a", Label: "Plain"}, {ID: "b", Label: "Example"}},
				codeSuggestions: []autofill.Suggestion{{ID: "b", Label: "Example", Account: "alex", Site: "example.com", Exact: true}},
			}
			h := newHarness(service, nil)
			var answer suggestAnswer
			h.call(t, asking(chrome(t), test.fields...), &answer)
			if answer.Status != statusOK || answer.Form == nil || answer.Form.Kind != formCode || !slices.Equal(answer.Form.Code, test.code) {
				t.Fatalf("answer %+v", answer)
			}
			want := []suggestionWire{{ID: "b", Label: "Example", Account: "alex", Site: "example.com"}}
			if !answer.Search || !reflect.DeepEqual(answer.Suggestions, want) || !slices.Equal(service.codes, []bool{true}) {
				t.Fatalf("suggestions %+v with search %v, asked for codes %v", answer.Suggestions, answer.Search, service.codes)
			}
		})
	}
}

func TestAFormNothingMatchesStillOffersSearch(t *testing.T) {
	app := signedApp("com.example.app", "example key")
	for name, test := range map[string]struct {
		app    func(*testing.T) autofill.App
		fields []field
		code   bool
	}{
		"a page's sign-in": {app: chrome, fields: pageSignIn("example.com")},
		"an app's sign-in": {app: func(*testing.T) autofill.App { return app }, fields: signIn},
		"a code field": {app: chrome, code: true, fields: []field{
			focus(onPage(field{Index: 0, Hints: []string{"one-time-code"}, InputType: inputClassText, Visible: true}, "example.com")),
		}},
	} {
		t.Run(name, func(t *testing.T) {
			service := &fakeService{open: true}
			h := newHarness(service, nil)
			var answer suggestAnswer
			h.call(t, asking(test.app(t), test.fields...), &answer)
			if answer.Status != statusOK || !answer.Search || answer.Form == nil || len(answer.Suggestions) != 0 {
				t.Fatalf("answer %+v", answer)
			}
			requester, ok := answer.Requester.requester()
			if !ok || !reflect.DeepEqual(requester, service.suggested[0]) || service.codes[0] != test.code {
				t.Fatalf("search would open for %+v, suggested for %+v", answer.Requester, service.suggested)
			}
		})
	}
}

func TestASaveOnlyFormAnswersWithItsRequesterAndNoSuggestions(t *testing.T) {
	service := &fakeService{open: true}
	h := newHarness(service, nil)
	var answer suggestAnswer
	h.call(t, asking(chrome(t), onPage(text(0, "login"), "example.com"), focus(onPage(password(1, "new_password"), "example.com"))), &answer)
	if answer.Status != statusOK || answer.Search || answer.Requester == nil || answer.Form.Save == nil || len(service.suggested) != 0 {
		t.Fatalf("answer %+v", answer)
	}
}

// cachedIcon's decoded length is exact only when size is a multiple of three.
func cachedIcon(size int, fill byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{fill}, size))
}

func TestSuggestionsCarryTheCachedIconsOfTheirSites(t *testing.T) {
	service := &fakeService{open: true, suggestions: []autofill.Suggestion{
		{ID: "a", Label: "Example", Site: "example.com", Exact: true},
		{ID: "b", Label: "Linked", Exact: true},
		{ID: "c", Label: "Example work", Site: "example.com"},
		{ID: "d", Label: "Plain", Site: "plain.org"},
		{ID: "e", Label: "Other", Site: "other.com"},
	}}
	h := newHarness(service, nil)
	h.icons.icons = map[string]string{"example.com": cachedIcon(300, 1), "other.com": cachedIcon(600, 2)}
	var answer suggestAnswer
	h.call(t, asking(chrome(t), pageSignIn("example.com")...), &answer)
	want := map[string]string{"example.com": cachedIcon(300, 1), "other.com": cachedIcon(600, 2)}
	if answer.Status != statusOK || len(answer.Suggestions) != 5 || !reflect.DeepEqual(answer.Icons, want) {
		t.Fatalf("answer %+v", answer)
	}
	if !slices.Equal(h.icons.asked, []string{"example.com", "plain.org", "other.com"}) {
		t.Fatalf("the cache was asked for %q", h.icons.asked)
	}
}

func TestSuggestionsWithoutCachedIconsCarryNone(t *testing.T) {
	for name, icons := range map[string]*fakeIcons{
		"icons off or none cached": {},
		"an unreadable cache":      {icons: map[string]string{"example.com": cachedIcon(300, 1)}, err: errors.New("unreadable")},
	} {
		t.Run(name, func(t *testing.T) {
			service := &fakeService{open: true, suggestions: []autofill.Suggestion{{ID: "a", Label: "Example", Site: "example.com", Exact: true}}}
			h := newHarness(service, nil)
			h.icons.icons, h.icons.err = icons.icons, icons.err
			request, err := json.Marshal(asking(chrome(t), pageSignIn("example.com")...))
			if err != nil {
				t.Fatal(err)
			}
			var answer map[string]json.RawMessage
			if err := json.Unmarshal(h.handler.Call(request), &answer); err != nil {
				t.Fatal(err)
			}
			if _, carried := answer["icons"]; carried || string(answer["status"]) != `"ok"` || answer["suggestions"] == nil {
				t.Fatalf("answer %s", answer)
			}
		})
	}
}

func TestSuggestionIconsStayWithinTheirCaps(t *testing.T) {
	large := maxIconBytes / 3 * 3
	fitting := maxAnswerIconBytes / large
	// A multiple of three within what the large icons leave of the answer's budget.
	small := (maxAnswerIconBytes - fitting*large) / 3 * 3
	if small == 0 {
		t.Fatal("the large icons leave no room for a small one")
	}
	service := &fakeService{open: true}
	icons := map[string]string{"huge.com": cachedIcon(large+3, 1)}
	service.suggestions = append(service.suggestions, autofill.Suggestion{ID: "huge", Site: "huge.com"})
	var sites []string
	for i := range fitting + 4 {
		site := fmt.Sprintf("site%d.com", i)
		sites = append(sites, site)
		icons[site] = cachedIcon(large, byte(i))
		service.suggestions = append(service.suggestions, autofill.Suggestion{ID: site, Site: site})
	}
	icons["small.com"] = cachedIcon(small, 1)
	service.suggestions = append(service.suggestions, autofill.Suggestion{ID: "small", Site: "small.com"})
	h := newHarness(service, nil)
	h.icons.icons = icons
	var answer suggestAnswer
	h.call(t, asking(chrome(t), pageSignIn("example.com")...), &answer)

	want := map[string]string{"small.com": icons["small.com"]}
	for _, site := range sites[:fitting] {
		want[site] = icons[site]
	}
	if !reflect.DeepEqual(answer.Icons, want) {
		t.Fatalf("carried icons of %d sites, want the first %d large ones and the small one", len(answer.Icons), fitting)
	}
	total := 0
	for _, icon := range answer.Icons {
		total += base64.StdEncoding.DecodedLen(len(icon))
	}
	if total > maxAnswerIconBytes {
		t.Fatalf("the answer carries %d bytes of icons", total)
	}
}

var origin = map[string]any{"origin": "https://example.com"}

func TestFillGivesTheEmailWhereTheFieldAsksForIt(t *testing.T) {
	for _, test := range []struct {
		login        autofill.Login
		email        bool
		wantUsername string
	}{
		{autofill.Login{Login: "alex", Email: "alex@example.com", Password: "pw"}, true, "alex@example.com"},
		{autofill.Login{Login: "alex", Email: "alex@example.com", Password: "pw"}, false, "alex"},
		{autofill.Login{Login: "alex", Password: "pw"}, true, "alex"},
		{autofill.Login{Email: "alex@example.com", Password: "pw"}, false, "alex@example.com"},
	} {
		service := &fakeService{open: true, login: test.login}
		h := newHarness(service, nil)
		var answer fillAnswer
		h.call(t, map[string]any{"op": "fill", "requester": origin, "id": "a", "email": test.email}, &answer)
		if answer.Status != statusOK || answer.Username != test.wantUsername || answer.Password != "pw" {
			t.Errorf("%+v (email %v): answer %+v", test.login, test.email, answer)
		}
		if len(service.filled) != 1 || service.filled[0].Origin != "https://example.com" {
			t.Errorf("filled for %+v", service.filled)
		}
	}
}

func TestReleasesFromALockedVaultReportItLocked(t *testing.T) {
	h := newHarness(&fakeService{err: fmt.Errorf("read: %w", autofill.ErrLocked)}, nil)
	app := map[string]any{"app": wireOf(signedApp("com.example.app", "example key"))}
	for _, request := range []map[string]any{
		{"op": "fill", "requester": origin, "id": "a"},
		{"op": "code", "requester": origin, "id": "a"},
		{"op": "search", "requester": origin, "query": "ex"},
		{"op": "link", "requester": app, "id": "a"},
	} {
		var answer outcome
		h.call(t, request, &answer)
		if answer.Status != statusLocked {
			t.Errorf("%s: status %s", request["op"], answer.Status)
		}
	}
	var failed outcome
	h.service.err = errors.New("the credential no longer matches")
	h.call(t, map[string]any{"op": "fill", "requester": origin, "id": "a"}, &failed)
	if failed.Status != statusFailed {
		t.Fatalf("status %s", failed.Status)
	}
}

func TestAFillThatFoundTheVaultLockedSucceedsOnceTheUnlockScreenOpensIt(t *testing.T) {
	service := &fakeService{err: autofill.ErrLocked, login: autofill.Login{Login: "alex", Password: "pw"}}
	h := newHarness(service, nil)
	fill := map[string]any{"op": "fill", "requester": origin, "id": "a"}
	var locked fillAnswer
	h.call(t, fill, &locked)
	if locked.Status != statusLocked || locked.Username != "" || locked.Password != "" {
		t.Fatalf("a locked vault answered %+v", locked)
	}
	var unlocked unlockAnswer
	h.call(t, map[string]any{"op": "unlock"}, &unlocked)
	if unlocked.Status != statusOK || h.opened != 1 {
		t.Fatalf("unlock answered %+v", unlocked)
	}
	service.open, service.err = true, nil
	var filled fillAnswer
	h.call(t, fill, &filled)
	if filled.Status != statusOK || filled.Username != "alex" || filled.Password != "pw" {
		t.Fatalf("the fill asked again answered %+v", filled)
	}
	if len(service.filled) != 2 || !reflect.DeepEqual(service.filled[0], service.filled[1]) {
		t.Fatalf("filled for %+v", service.filled)
	}
}

func TestCodeGivesTheCurrentCodeAcrossTheFormsFields(t *testing.T) {
	h := newHarness(&fakeService{open: true, code: autofill.Code{Code: "123456", ExpiresAt: 1}}, nil)
	for fields, want := range map[int][]string{
		1: {"123456"},
		6: {"1", "2", "3", "4", "5", "6"},
		4: {"123456"},
	} {
		var answer codeAnswer
		h.call(t, map[string]any{"op": "code", "requester": origin, "id": "a", "fields": fields}, &answer)
		if answer.Status != statusOK || !slices.Equal(answer.Entries, want) {
			t.Errorf("%d fields: answer %+v, want %q", fields, answer, want)
		}
	}
}

func TestSearchOpensOnTheRequestersMatches(t *testing.T) {
	service := &fakeService{
		open:        true,
		suggestions: []autofill.Suggestion{{ID: "b", Label: "Example", Account: "alex", Site: "example.com", Exact: true}},
	}
	h := newHarness(service, nil)
	var answer searchAnswer
	h.call(t, map[string]any{"op": "search", "requester": map[string]any{"origin": "https://www.xn--80ak6aa92e.com"}, "query": " "}, &answer)
	want := []searchResult{{suggestionWire: suggestionWire{ID: "b", Label: "Example", Account: "alex", Site: "example.com"}, Matches: true}}
	if answer.Status != statusOK || answer.Scope != autofill.ScopeMatches || !reflect.DeepEqual(answer.Results, want) {
		t.Fatalf("answer %+v", answer)
	}
	if answer.Site != "xn--80ak6aa92e.com" {
		t.Fatalf("the page's site reads %q", answer.Site)
	}
	if len(service.searched) != 0 {
		t.Fatalf("the vault was searched for %q while the requester has matches", service.searched)
	}
}

func TestSearchWithNoMatchesOpensOnTheWholeVault(t *testing.T) {
	app := signedApp("com.example.app", "example key")
	service := &fakeService{open: true, found: []autofill.Suggestion{{ID: "a", Label: "Pinned"}, {ID: "b", Label: "Recent"}}}
	h := newHarness(service, nil)
	var answer searchAnswer
	h.call(t, map[string]any{"op": "search", "requester": map[string]any{"app": wireOf(app)}, "query": ""}, &answer)
	if answer.Status != statusOK || answer.Scope != autofill.ScopeVault || answer.Site != "" || len(answer.Results) != 2 || answer.Results[0].Matches {
		t.Fatalf("answer %+v", answer)
	}
	if !slices.Equal(service.searched, []string{""}) {
		t.Fatalf("searched for %q", service.searched)
	}
}

func TestATypedSearchCoversTheVaultAndMarksWhatAlreadyMatches(t *testing.T) {
	service := &fakeService{
		open:        true,
		found:       []autofill.Suggestion{{ID: "a", Label: "Example"}, {ID: "b", Label: "Other"}},
		suggestions: []autofill.Suggestion{{ID: "b"}},
	}
	h := newHarness(service, nil)
	var answer searchAnswer
	h.call(t, map[string]any{"op": "search", "requester": origin, "query": "ex"}, &answer)
	if answer.Status != statusOK || answer.Scope != autofill.ScopeVault || len(answer.Results) != 2 || answer.Results[0].Matches || !answer.Results[1].Matches {
		t.Fatalf("answer %+v", answer)
	}
	if answer.Site != "example.com" || !slices.Equal(service.searched, []string{"ex"}) {
		t.Fatalf("site %q, searched for %q", answer.Site, service.searched)
	}
}

func TestACodeFieldsSearchListsTheMatchesWithCodesAndMarksEveryMatch(t *testing.T) {
	service := &fakeService{
		open:            true,
		suggestions:     []autofill.Suggestion{{ID: "a"}, {ID: "b"}},
		codeSuggestions: []autofill.Suggestion{{ID: "b"}},
		found:           []autofill.Suggestion{{ID: "a"}, {ID: "c"}},
	}
	h := newHarness(service, nil)
	var opening searchAnswer
	h.call(t, map[string]any{"op": "search", "requester": origin, "code": true}, &opening)
	if opening.Scope != autofill.ScopeMatches || len(opening.Results) != 1 || opening.Results[0].ID != "b" || !opening.Results[0].Matches {
		t.Fatalf("opening %+v", opening)
	}
	var typed searchAnswer
	h.call(t, map[string]any{"op": "search", "requester": origin, "code": true, "query": "a"}, &typed)
	if typed.Scope != autofill.ScopeVault || len(typed.Results) != 2 || !typed.Results[0].Matches || typed.Results[1].Matches {
		t.Fatalf("typed %+v", typed)
	}
	service.codeSuggestions = []autofill.Suggestion{}
	var none searchAnswer
	h.call(t, map[string]any{"op": "search", "requester": origin, "code": true}, &none)
	if none.Scope != autofill.ScopeVault || len(none.Results) != 2 {
		t.Fatalf("a code field no match has a code for %+v", none)
	}
}

func TestLinkingAddsAPagesSiteAndLinksAnApp(t *testing.T) {
	app := signedApp("com.example.app", "example key")
	service := &fakeService{open: true}
	h := newHarness(service, nil)
	var added, linked outcome
	h.call(t, map[string]any{"op": "link", "requester": origin, "id": "a"}, &added)
	h.call(t, map[string]any{"op": "link", "requester": map[string]any{"app": wireOf(app)}, "id": "a"}, &linked)
	if added.Status != statusOK || linked.Status != statusOK {
		t.Fatalf("a page: %s, an app: %s", added.Status, linked.Status)
	}
	if len(service.added) != 1 || service.added[0].Origin != "https://example.com" {
		t.Fatalf("added %+v", service.added)
	}
	if len(service.linked) != 1 || !reflect.DeepEqual(service.linked[0], app) {
		t.Fatalf("linked %+v", service.linked)
	}
	var refused outcome
	service.err = autofill.ErrInvalidRequest
	h.call(t, map[string]any{"op": "link", "requester": origin, "id": "a"}, &refused)
	if refused.Status != statusFailed {
		t.Fatalf("a credential without room answered %s", refused.Status)
	}
}

func TestACodeOfACredentialWithoutASetupSaysSo(t *testing.T) {
	h := newHarness(&fakeService{open: true, err: autofill.ErrNoCode}, nil)
	var answer codeAnswer
	h.call(t, map[string]any{"op": "code", "requester": origin, "id": "a", "fields": 1}, &answer)
	if answer.Status != statusNoCode || answer.Entries != nil {
		t.Fatalf("answer %+v", answer)
	}
}

func TestThePageReadsIconsAndTheLanguageAsTheAppsWindowDoes(t *testing.T) {
	h := newHarness(&fakeService{open: true}, nil)
	h.page.icons = map[string]api.SiteIcon{"example.com": {Image: "iVBOR", Tint: "#336699"}}
	h.page.language = api.LanguageSettings{Languages: []string{"en", "ru"}, Language: "ru"}
	var icon struct {
		outcome
		Image, Tint string
	}
	h.call(t, map[string]any{"op": "icon", "site": "example.com"}, &icon)
	if icon.Status != statusOK || icon.Image != "iVBOR" || icon.Tint != "#336699" {
		t.Fatalf("icon %+v", icon)
	}
	h.call(t, map[string]any{"op": "icon", "site": "example.org"}, &icon)
	if icon.Status != statusFailed {
		t.Fatalf("an icon the app refused answered %+v", icon)
	}
	var language struct {
		outcome
		Languages []string
		Language  string
		Chosen    bool
	}
	h.call(t, map[string]any{"op": "language"}, &language)
	if language.Status != statusOK || language.Language != "ru" || language.Chosen || !slices.Equal(language.Languages, []string{"en", "ru"}) {
		t.Fatalf("language %+v", language)
	}
}

func TestThePageReadsTheInterfaceSizeAsTheAppsWindowDoes(t *testing.T) {
	h := newHarness(&fakeService{open: true}, nil)
	h.page.size = api.InterfaceSize{Percent: 115, Offered: []int{85, 100, 115}}
	var size struct {
		outcome
		Percent int
		Offered []int
	}
	h.call(t, map[string]any{"op": "interface-size"}, &size)
	if size.Status != statusOK || size.Percent != 115 || !slices.Equal(size.Offered, []int{85, 100, 115}) {
		t.Fatalf("interface size %+v", size)
	}
}

func TestThePageReadsTheAppearanceAsTheAppsWindowDoes(t *testing.T) {
	h := newHarness(&fakeService{open: true}, nil)
	h.page.appearance = api.Appearance{Appearance: "light", Offered: []string{"system", "light", "dark"}}
	var appearance struct {
		outcome
		Appearance string
	}
	h.call(t, map[string]any{"op": "appearance"}, &appearance)
	if appearance.Status != statusOK || appearance.Appearance != "light" {
		t.Fatalf("appearance %+v", appearance)
	}
}

func TestACaptureIsHeldAndSavedWhereTheOwnerChooses(t *testing.T) {
	app := signedApp("com.example.app", "example key")
	service := &fakeService{open: true, created: true, offer: autofill.Offer{
		Token: "t", Name: "Example",
		Targets:   []autofill.Target{{ID: "a", Label: "Example", Account: "alex", Action: autofill.SaveUpdate, Tags: []string{"Work"}}},
		Suggested: "a",
	}}
	h := newHarness(service, nil)
	var held captureAnswer
	h.call(t, map[string]any{"op": "capture", "requester": map[string]any{"app": wireOf(app)}, "account": "alex", "password": "pw"}, &held)
	if held.Status != statusOK || held.Capture == "" || held.Offer != nil {
		t.Fatalf("the capture answered more than its token: %+v", held)
	}
	if len(service.held) != 1 || service.held[0].Password != "pw" || !reflect.DeepEqual(service.held[0].Requester.App, app) {
		t.Fatalf("held %+v", service.held)
	}
	var offered captureAnswer
	h.call(t, map[string]any{"op": "offer", "capture": held.Capture}, &offered)
	want := &offerWire{Token: "t", Name: "Example", Account: "alex", Suggested: "a",
		Targets: []targetWire{{ID: "a", Label: "Example", Account: "alex", Action: "update", Tags: []string{"Work"}}}}
	if offered.Status != statusOK || !reflect.DeepEqual(offered.Offer, want) || len(service.held) != 1 {
		t.Fatalf("the save screen was offered %+v, held %d", offered, len(service.held))
	}
	var again captureAnswer
	h.call(t, map[string]any{"op": "offer", "capture": held.Capture}, &again)
	if again.Status != statusFailed {
		t.Fatalf("an offer was given twice: %+v", again)
	}
	var saved saveAnswer
	h.call(t, map[string]any{"op": "save", "token": "t", "choice": map[string]any{"name": "Example", "account": "alex"}}, &saved)
	if saved.Status != statusOK || !saved.Created || service.saved[0] != (autofill.Choice{Name: "Example", Account: "alex"}) {
		t.Fatalf("answer %+v, saved %+v", saved, service.saved)
	}
	var empty captureAnswer
	h.call(t, map[string]any{"op": "capture", "requester": origin, "account": "alex"}, &empty)
	if empty.Status != statusFailed {
		t.Fatalf("a capture without a password was held: %+v", empty)
	}
}

func TestAnOfferMadeWhileOpenWaitsWhileTheVaultIsLocked(t *testing.T) {
	service := &fakeService{open: true, offer: autofill.Offer{Token: "t", Name: "Example",
		Targets: []autofill.Target{{ID: "a", Label: "Example", Account: "alex", Action: autofill.SaveUpdate}}}}
	h := newHarness(service, nil)
	var held captureAnswer
	h.call(t, map[string]any{"op": "capture", "requester": origin, "account": "alex", "password": "pw"}, &held)
	service.open, service.err = false, autofill.ErrLocked
	var locked captureAnswer
	h.call(t, map[string]any{"op": "offer", "capture": held.Capture}, &locked)
	if locked.Status != statusLocked || locked.Offer != nil {
		t.Fatalf("a locked vault's items were offered: %+v", locked)
	}
	service.open, service.err = true, nil
	var offered captureAnswer
	h.call(t, map[string]any{"op": "offer", "capture": held.Capture}, &offered)
	if offered.Status != statusOK || offered.Offer == nil || offered.Offer.Targets[0].Label != "Example" {
		t.Fatalf("the sign-in once the vault opened: %+v", offered)
	}
}

func TestASignInCapturedWhileLockedWaitsForTheVaultToOpen(t *testing.T) {
	service := &fakeService{err: autofill.ErrLocked, offer: autofill.Offer{Token: "t", Name: "Example"}}
	h := newHarness(service, nil)
	var waiting captureAnswer
	h.call(t, map[string]any{"op": "capture", "requester": origin, "account": "alex", "password": "pw"}, &waiting)
	if waiting.Status != statusLocked || waiting.Capture == "" || waiting.Offer != nil || len(service.held) != 0 {
		t.Fatalf("answer %+v, held %+v", waiting, service.held)
	}
	service.open, service.err = true, nil
	var offered captureAnswer
	h.call(t, map[string]any{"op": "offer", "capture": waiting.Capture}, &offered)
	if offered.Status != statusOK || offered.Offer == nil || offered.Offer.Token != "t" || offered.Offer.Account != "alex" || offered.Offer.Site != "example.com" {
		t.Fatalf("answer %+v", offered)
	}
	if len(service.held) != 1 || service.held[0].Password != "pw" || service.held[0].Requester.Origin != "https://example.com" {
		t.Fatalf("held %+v", service.held)
	}
	var again captureAnswer
	h.call(t, map[string]any{"op": "offer", "capture": waiting.Capture}, &again)
	if again.Status != statusFailed {
		t.Fatalf("a sign-in offered once was offered again: %+v", again)
	}
}

func TestASignInWaitingForTheVaultKeepsWaitingWhileItStaysLocked(t *testing.T) {
	service := &fakeService{err: autofill.ErrLocked, offer: autofill.Offer{Token: "t", Name: "Example"}}
	h := newHarness(service, nil)
	var waiting captureAnswer
	h.call(t, map[string]any{"op": "capture", "requester": origin, "account": "alex", "password": "pw"}, &waiting)
	var locked captureAnswer
	h.call(t, map[string]any{"op": "offer", "capture": waiting.Capture}, &locked)
	if locked.Status != statusLocked || locked.Offer != nil {
		t.Fatalf("a locked vault answered %+v", locked)
	}
	service.open, service.err = true, nil
	var offered captureAnswer
	h.call(t, map[string]any{"op": "offer", "capture": waiting.Capture}, &offered)
	if offered.Status != statusOK || offered.Offer == nil || offered.Offer.Account != "alex" {
		t.Fatalf("the sign-in was not offered once the vault opened: %+v", offered)
	}
}

func TestASignInTheVaultAlreadyHoldsIsNotOffered(t *testing.T) {
	h := newHarness(&fakeService{open: true, err: autofill.ErrNothingToSave}, nil)
	var answer captureAnswer
	h.call(t, map[string]any{"op": "capture", "requester": origin, "account": "alex", "password": "pw"}, &answer)
	if answer.Status != statusNone || answer.Offer != nil || answer.Capture != "" {
		t.Fatalf("answer %+v", answer)
	}
}

func TestASaveTheVaultRefusesSaysWhy(t *testing.T) {
	for err, want := range map[error]status{
		autofill.ErrNameRefused:    statusNameRefused,
		autofill.ErrAccountRefused: statusAccountRefused,
		autofill.ErrNotFound:       statusFailed,
		autofill.ErrLocked:         statusLocked,
	} {
		h := newHarness(&fakeService{open: true, err: err}, nil)
		var answer saveAnswer
		h.call(t, map[string]any{"op": "save", "token": "t", "choice": map[string]any{"name": ""}}, &answer)
		if answer.Status != want {
			t.Errorf("%v: status %s, want %s", err, answer.Status, want)
		}
	}
}

func TestTheUnlockScreenOpensTheVault(t *testing.T) {
	t.Run("with the device's unlock", func(t *testing.T) {
		h := newHarness(&fakeService{}, nil)
		var answer unlockAnswer
		h.call(t, map[string]any{"op": "unlock"}, &answer)
		if answer.Status != statusOK || !slices.Equal(h.vault.reasons, []string{"Unlock Ravenpass to fill in"}) || h.opened != 1 {
			t.Fatalf("answer %+v, reasons %v, opened %d", answer, h.vault.reasons, h.opened)
		}
	})
	t.Run("with a PIN", func(t *testing.T) {
		h := newHarness(&fakeService{}, nil)
		var answer unlockAnswer
		h.call(t, map[string]any{"op": "unlock", "pin": "123456"}, &answer)
		if answer.Status != statusOK || !slices.Equal(h.vault.pins, []string{"123456"}) || len(h.vault.reasons) != 0 {
			t.Fatalf("answer %+v", answer)
		}
	})
	t.Run("already open", func(t *testing.T) {
		h := newHarness(&fakeService{open: true}, nil)
		var answer unlockAnswer
		h.call(t, map[string]any{"op": "unlock"}, &answer)
		if answer.Status != statusOK || len(h.vault.reasons) != 0 || h.opened != 0 {
			t.Fatalf("answer %+v", answer)
		}
	})
}

func TestEachRequestIsReportedOnceAnswered(t *testing.T) {
	h := newHarness(&fakeService{}, nil)
	var openedBefore []int
	h.handler.requested = func() {
		openedBefore = append(openedBefore, h.opened)
		h.requested++
	}
	var answer outcome
	h.call(t, map[string]any{"op": "unlock"}, &answer)
	h.call(t, map[string]any{"op": "hidden"}, &answer)
	h.call(t, asking(signedApp("com.example.app", "key"), signIn...), &answer)
	if h.requested != 3 || !slices.Equal(openedBefore, []int{1, 1, 1}) {
		t.Fatalf("%d requests reported, the vault opened before each: %v", h.requested, openedBefore)
	}
	if err := json.Unmarshal(h.handler.Call([]byte("not json")), &answer); err != nil || h.requested != 3 {
		t.Fatalf("a malformed request was reported: %d, %v", h.requested, err)
	}
}

func TestTheUnlockScreenLearnsWhyTheVaultStayedLocked(t *testing.T) {
	for _, test := range []struct {
		deviceErr, pinErr error
		pin               string
		want              status
		wantAttempts      int
	}{
		{deviceErr: fmt.Errorf("agree: %w", ownerauth.ErrCanceled), want: statusCanceled},
		{deviceErr: ownerauth.ErrFailed, want: statusFailed},
		{pinErr: unlock.ErrWrongPIN, pin: "000000", want: statusWrongPIN, wantAttempts: 4},
		{pinErr: vault.ErrInvalidPIN, pin: "1", want: statusWrongPIN, wantAttempts: 4},
		{pinErr: unlock.ErrPINRemoved, pin: "000000", want: statusPINRemoved},
		{pinErr: fmt.Errorf("unlock: %w", unlock.ErrTooSoon), pin: "000000", want: statusTooSoon},
	} {
		h := newHarness(&fakeService{}, nil)
		h.vault.deviceErr, h.vault.pinErr = test.deviceErr, test.pinErr
		h.vault.methods = vaultservice.Methods{PINSet: true, PINAttemptsLeft: 4}
		var answer unlockAnswer
		h.call(t, map[string]any{"op": "unlock", "pin": test.pin}, &answer)
		if answer.Status != test.want || answer.AttemptsLeft != test.wantAttempts || h.opened != 0 {
			t.Errorf("%v%v: answer %+v", test.deviceErr, test.pinErr, answer)
		}
	}
}

func TestTheUnlockScreenOffersTheWaysInThisDeviceHas(t *testing.T) {
	h := newHarness(&fakeService{}, nil)
	h.vault.methods = vaultservice.Methods{BiometryAvailable: true, BiometryEnabled: true, PINSet: true, PINAttemptsLeft: 9}
	var answer methodsAnswer
	h.call(t, map[string]any{"op": "methods"}, &answer)
	if answer.Status != statusOK || !answer.Biometry || !answer.PIN || answer.AttemptsLeft != 9 ||
		answer.PINMin != vault.MinPINLength || answer.PINMax != vault.MaxPINLength {
		t.Fatalf("answer %+v", answer)
	}
	h.vault.methods.BiometryAvailable = false
	h.call(t, map[string]any{"op": "methods"}, &answer)
	if answer.Biometry {
		t.Fatal("biometrics the device cannot use were offered")
	}
}

func TestTheScreensHoldTheAutomaticLockWhileAnyIsShown(t *testing.T) {
	h := newHarness(&fakeService{}, nil)
	var answer outcome
	for _, op := range []string{"shown", "shown", "hidden"} {
		h.call(t, map[string]any{"op": op}, &answer)
	}
	if h.holds != 1 {
		t.Fatalf("%d holds with one screen shown", h.holds)
	}
	h.call(t, map[string]any{"op": "hidden"}, &answer)
	h.call(t, map[string]any{"op": "hidden"}, &answer)
	if h.holds != 0 || answer.Status != statusOK {
		t.Fatalf("%d holds with no screen shown", h.holds)
	}
}

func TestUnknownAndMalformedRequestsFail(t *testing.T) {
	h := newHarness(&fakeService{open: true}, nil)
	for _, request := range []string{`{"op":"delete"}`, `not json`, `{"op":"fill","requester":{}}`,
		`{"op":"fill","requester":{"origin":"https://example.com","app":{"package":"a","signers":["00"]}}}`} {
		var answer outcome
		if err := json.Unmarshal(h.handler.Call([]byte(request)), &answer); err != nil || answer.Status != statusFailed {
			t.Errorf("%s: answer %+v, %v", request, answer, err)
		}
	}
}
