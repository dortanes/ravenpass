package vaultservice

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

// standardSecret is the shared secret RFC 6238 publishes its test vectors for.
const standardSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func createTestCredential(t *testing.T, service *Service, input vault.CredentialInput) vault.ID {
	t.Helper()
	if input.Password == "" {
		input.Password = "secret"
	}
	id, err := service.CreateCredential(input, nil)
	if err != nil {
		t.Fatalf("create %q: %v", input.Label, err)
	}
	return id
}

func suggestionIDs(suggestions []Suggestion) []vault.ID {
	ids := make([]vault.ID, len(suggestions))
	for i, suggestion := range suggestions {
		ids[i] = suggestion.ID
	}
	return ids
}

func suggestionsFor(t *testing.T, service *Service, requester Requester, purpose Purpose) []Suggestion {
	t.Helper()
	suggestions, err := service.Suggestions(requester, purpose)
	if err != nil {
		t.Fatal(err)
	}
	return suggestions
}

func TestSuggestionsListMatchesExactFirstThenByRecentUseThenLabel(t *testing.T) {
	service, _ := readyVault(t)
	gamma := createTestCredential(t, service, vault.CredentialInput{Label: "Gamma", Websites: []string{"https://login.example.com", "https://www.example.com"}, Login: "carol"})
	zeta := createTestCredential(t, service, vault.CredentialInput{Label: "Zeta", Websites: []string{"https://example.com"}, Login: "zed"})
	beta := createTestCredential(t, service, vault.CredentialInput{Label: "Beta", Websites: []string{"https://example.com"}, Email: "beta@example.test"})
	alpha := createTestCredential(t, service, vault.CredentialInput{Label: "alpha", Websites: []string{"https://example.com:8443/login"}, Login: "al", Email: "al@example.test"})
	epsilon := createTestCredential(t, service, vault.CredentialInput{Label: "Epsilon", Websites: []string{"https://shop.example.com"}, Login: "eve"})
	delta := createTestCredential(t, service, vault.CredentialInput{Label: "Delta", Websites: []string{"https://example.org", "https://login.example.com"}, Login: "dee"})
	createTestCredential(t, service, vault.CredentialInput{Label: "Other", Websites: []string{"https://example.org"}, Login: "otto"})
	createTestCredential(t, service, vault.CredentialInput{Label: "Bare", Login: "bare"})
	if _, err := service.CreateIdentity(vault.IdentityInput{Label: "Me", Emails: []string{"me@example.com"}}, nil); err != nil {
		t.Fatal(err)
	}
	card := testCard()
	card.BankSite = "https://example.com"
	if _, err := service.CreateCard(card, nil); err != nil {
		t.Fatal(err)
	}
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: gamma, lastUsedAt: 2000}, {id: zeta, lastUsedAt: 1000}, {id: epsilon, lastUsedAt: 500}}})

	want := []Suggestion{
		{ID: gamma, Label: "Gamma", Account: "carol", Site: "example.com", Exact: true},
		{ID: zeta, Label: "Zeta", Account: "zed", Site: "example.com", Exact: true},
		{ID: alpha, Label: "alpha", Account: "al", Site: "example.com", Exact: true},
		{ID: beta, Label: "Beta", Account: "beta@example.test", Site: "example.com", Exact: true},
		{ID: epsilon, Label: "Epsilon", Account: "eve", Site: "shop.example.com"},
		{ID: delta, Label: "Delta", Account: "dee", Site: "login.example.com"},
	}
	if got := suggestionsFor(t, service, OriginRequester("https://example.com"), PurposeSignIn); !reflect.DeepEqual(got, want) {
		t.Fatalf("suggestions = %+v, want %+v", got, want)
	}
	if got := suggestionIDs(suggestionsFor(t, service, OriginRequester("https://login.example.com"), PurposeSignIn)); !slices.Equal(got, []vault.ID{gamma, delta, zeta, epsilon, alpha, beta}) {
		t.Fatalf("suggestions for a subdomain = %v", got)
	}
	for _, origin := range []string{"https://example.net", "ftp://example.com", "example.com", "", "https://"} {
		if got := suggestionsFor(t, service, OriginRequester(origin), PurposeSignIn); len(got) != 0 {
			t.Fatalf("suggestions for %q = %+v", origin, got)
		}
	}
}

func TestSuggestionsStopAtFiftyAfterTheExactMatches(t *testing.T) {
	service, _ := readyVault(t)
	for i := range maxSuggestions + 1 {
		createTestCredential(t, service, vault.CredentialInput{Label: fmt.Sprintf("Account %02d", i), Websites: []string{"https://login.example.com"}})
	}
	exact := createTestCredential(t, service, vault.CredentialInput{Label: "Zulu", Websites: []string{"https://example.com"}})
	suggestions := suggestionsFor(t, service, OriginRequester("https://example.com"), PurposeSignIn)
	if len(suggestions) != maxSuggestions {
		t.Fatalf("%d suggestions, want %d", len(suggestions), maxSuggestions)
	}
	if suggestions[0].ID != exact || !suggestions[0].Exact {
		t.Fatalf("first suggestion = %+v, want the exact match", suggestions[0])
	}
	if last := suggestions[maxSuggestions-1]; last.Label != fmt.Sprintf("Account %02d", maxSuggestions-2) {
		t.Fatalf("last suggestion = %+v", last)
	}
}

func TestFillCredentialReleasesAMatchingCredentialAndRecordsTheUse(t *testing.T) {
	service, _ := readyVault(t)
	mail := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://example.com"}, Login: "alex", Email: "alex@example.test", Password: "mail-secret"})
	recent := createTestCredential(t, service, vault.CredentialInput{Label: "Recent", Websites: []string{"https://example.com"}, Password: "recent-secret"})
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: recent, lastUsedAt: 1000}}})
	if got := suggestionIDs(suggestionsFor(t, service, OriginRequester("https://example.com"), PurposeSignIn)); !slices.Equal(got, []vault.ID{recent, mail}) {
		t.Fatalf("suggestions before the fill = %v", got)
	}

	fill, err := service.FillCredential(mail, OriginRequester("https://login.example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if fill != (Fill{Login: "alex", Email: "alex@example.test", Password: "mail-secret"}) {
		t.Fatalf("fill = %+v", fill)
	}
	usage, err := service.Usage()
	if err != nil || usage[mail] <= usage[recent] {
		t.Fatalf("usage after the fill = %v, error = %v", usage, err)
	}
	if got := suggestionIDs(suggestionsFor(t, service, OriginRequester("https://example.com"), PurposeSignIn)); !slices.Equal(got, []vault.ID{mail, recent}) {
		t.Fatalf("suggestions after the fill = %v", got)
	}
}

func TestFillCredentialChecksTheMatchAgain(t *testing.T) {
	service, _ := readyVault(t)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://example.com"}, Password: "mail-secret"})
	identity, err := service.CreateIdentity(testIdentity(), nil)
	if err != nil {
		t.Fatal(err)
	}
	card := testCard()
	card.BankSite = "https://example.com"
	cardID, err := service.CreateCard(card, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"https://example.org", "https://example.com.attacker.test", "ftp://example.com", "example.com", ""} {
		if _, err := service.FillCredential(id, OriginRequester(origin)); !errors.Is(err, ErrNoMatch) {
			t.Fatalf("fill for %q: got %v, want ErrNoMatch", origin, err)
		}
	}
	if len(suggestionsFor(t, service, OriginRequester("https://example.com"), PurposeSignIn)) != 1 {
		t.Fatal("the credential is not suggested")
	}
	websites := []string{"https://example.org"}
	if err := service.EditCredential(id, vault.CredentialPatch{Websites: &websites}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.FillCredential(id, OriginRequester("https://example.com")); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("fill after the site changed: got %v, want ErrNoMatch", err)
	}
	usage, err := service.Usage()
	if err != nil || len(usage) != 0 {
		t.Fatalf("refused fills recorded usage %v, error = %v", usage, err)
	}
	for _, missing := range []vault.ID{{0xff}, identity, cardID} {
		if _, err := service.FillCredential(missing, OriginRequester("https://example.com")); !errors.Is(err, vault.ErrNotFound) {
			t.Fatalf("fill of %s: got %v, want ErrNotFound", missing, err)
		}
	}
	if err := service.DeleteItem(id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.FillCredential(id, OriginRequester("https://example.org")); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("fill of a deleted credential: got %v, want ErrNotFound", err)
	}
}

func TestMatchingCredentialNamesAMatchWithoutRecordingAUse(t *testing.T) {
	service, _ := readyVault(t)
	mail := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://example.com"}, Email: "alex@example.test", Password: "mail-secret"})
	matched, err := service.MatchingCredential(mail, OriginRequester("https://login.example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(matched, Suggestion{ID: mail, Label: "Mail", Account: "alex@example.test", Site: "example.com"}) {
		t.Fatalf("matched = %+v", matched)
	}
	if _, err := service.MatchingCredential(mail, OriginRequester("https://example.org")); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("another site: got %v, want ErrNoMatch", err)
	}
	if _, err := service.MatchingCredential(vault.ID{0xff}, OriginRequester("https://example.com")); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("a missing credential: got %v, want ErrNotFound", err)
	}
	usage, err := service.Usage()
	if err != nil || len(usage) != 0 {
		t.Fatalf("naming a match recorded usage %v, error = %v", usage, err)
	}
}

func TestFillCredentialLeavesTheSelectionAlone(t *testing.T) {
	service, _ := readyVault(t)
	open := createTestCredential(t, service, vault.CredentialInput{Label: "Open", Websites: []string{"https://example.org"}, Password: "open-secret"})
	filled := createTestCredential(t, service, vault.CredentialInput{Label: "Filled", Websites: []string{"https://example.com"}, Password: "filled-secret"})
	ticket, err := service.Select(open)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.FillCredential(filled, OriginRequester("https://example.com")); err != nil {
		t.Fatal(err)
	}
	credential, err := service.ReadSelected(ticket)
	if err != nil || credential.ID != open || credential.Password != "open-secret" {
		t.Fatalf("the selected credential after a fill = %+v, error = %v", credential, err)
	}
}

func TestCodeSuggestionsListOnlyMatchingCredentialsWithASetup(t *testing.T) {
	service, _ := readyVault(t)
	coded := createTestCredential(t, service, vault.CredentialInput{Label: "Coded", Websites: []string{"https://example.com"}, Login: "alex", TOTP: standardSecret})
	recent := createTestCredential(t, service, vault.CredentialInput{Label: "Recent", Websites: []string{"https://login.example.com"}, Email: "sam@example.test", TOTP: "otpauth://totp/Example?digits=8&period=60&secret=" + standardSecret})
	bare := createTestCredential(t, service, vault.CredentialInput{Label: "Bare", Websites: []string{"https://example.com"}, Login: "bare"})
	createTestCredential(t, service, vault.CredentialInput{Label: "Elsewhere", Websites: []string{"https://example.org"}, TOTP: standardSecret})
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: recent, lastUsedAt: 1000}, {id: bare, lastUsedAt: 2000}}})

	ordinary := vault.CodeFace{Digits: 6, Period: 30}
	want := []Suggestion{
		{ID: coded, Label: "Coded", Account: "alex", Site: "example.com", Exact: true, Code: ordinary},
		{ID: recent, Label: "Recent", Account: "sam@example.test", Site: "login.example.com", Code: vault.CodeFace{Digits: 8, Period: 60}},
	}
	if got := suggestionsFor(t, service, OriginRequester("https://example.com"), PurposeCode); !reflect.DeepEqual(got, want) {
		t.Fatalf("code suggestions = %+v, want %+v", got, want)
	}
	signIn := suggestionsFor(t, service, OriginRequester("https://example.com"), PurposeSignIn)
	if got := suggestionIDs(signIn); !slices.Equal(got, []vault.ID{bare, coded, recent}) {
		t.Fatalf("sign-in suggestions = %v", got)
	}
	for _, suggestion := range signIn {
		if suggestion.Code != (vault.CodeFace{}) {
			t.Fatalf("a sign-in suggestion carries a code face: %+v", suggestion)
		}
	}

	cleared, added := "", standardSecret
	if err := service.EditCredential(coded, vault.CredentialPatch{TOTP: &cleared}); err != nil {
		t.Fatal(err)
	}
	if err := service.EditCredential(bare, vault.CredentialPatch{TOTP: &added}); err != nil {
		t.Fatal(err)
	}
	after := suggestionsFor(t, service, OriginRequester("https://example.com"), PurposeCode)
	if got := suggestionIDs(after); !slices.Equal(got, []vault.ID{bare, recent}) || after[0].Code != ordinary {
		t.Fatalf("code suggestions after the edits = %+v", after)
	}
}

func TestOneTimeCodeAnswersTheCurrentCodeWithoutRecordingAUse(t *testing.T) {
	service, _ := readyVault(t)
	open := createTestCredential(t, service, vault.CredentialInput{Label: "Open", Websites: []string{"https://example.org"}, Password: "open-secret"})
	coded := createTestCredential(t, service, vault.CredentialInput{Label: "Coded", Websites: []string{"https://example.com"}, Password: "coded-secret", TOTP: standardSecret})
	ticket, err := service.Select(open)
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now()
	code, err := service.OneTimeCode(coded, OriginRequester("https://login.example.com"))
	after := time.Now()
	if err != nil {
		t.Fatal(err)
	}
	earlier, err := vault.GenerateOneTimeCode(standardSecret, before)
	if err != nil {
		t.Fatal(err)
	}
	later, err := vault.GenerateOneTimeCode(standardSecret, after)
	if err != nil {
		t.Fatal(err)
	}
	if code != earlier && code != later {
		t.Fatalf("code = %+v, want %+v or %+v", code, earlier, later)
	}
	if code.Digits != 6 || code.Period != 30 || !code.ExpiresAt.After(before) {
		t.Fatalf("code = %+v", code)
	}

	usage, err := service.Usage()
	if err != nil || len(usage) != 0 {
		t.Fatalf("a code recorded usage %v, error = %v", usage, err)
	}
	credential, err := service.ReadSelected(ticket)
	if err != nil || credential.ID != open || credential.Password != "open-secret" {
		t.Fatalf("the selected credential after a code = %+v, error = %v", credential, err)
	}
}

func TestOneTimeCodeChecksTheMatchAndTheSetup(t *testing.T) {
	service, _ := readyVault(t)
	coded := createTestCredential(t, service, vault.CredentialInput{Label: "Coded", Websites: []string{"https://example.com"}, TOTP: standardSecret})
	bare := createTestCredential(t, service, vault.CredentialInput{Label: "Bare", Websites: []string{"https://example.com"}})
	identity, err := service.CreateIdentity(testIdentity(), nil)
	if err != nil {
		t.Fatal(err)
	}
	card := testCard()
	card.BankSite = "https://example.com"
	cardID, err := service.CreateCard(card, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"https://example.org", "https://example.com.attacker.test", "ftp://example.com", "example.com", ""} {
		if _, err := service.OneTimeCode(coded, OriginRequester(origin)); !errors.Is(err, ErrNoMatch) {
			t.Fatalf("code for %q: got %v, want ErrNoMatch", origin, err)
		}
	}
	for _, missing := range []vault.ID{{0xff}, identity, cardID} {
		if _, err := service.OneTimeCode(missing, OriginRequester("https://example.com")); !errors.Is(err, vault.ErrNotFound) {
			t.Fatalf("code of %s: got %v, want ErrNotFound", missing, err)
		}
	}
	if _, err := service.OneTimeCode(bare, OriginRequester("https://example.com")); !errors.Is(err, vault.ErrNoTOTP) {
		t.Fatalf("code of a credential without a setup: got %v, want ErrNoTOTP", err)
	}
	cleared := ""
	if err := service.EditCredential(coded, vault.CredentialPatch{TOTP: &cleared}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.OneTimeCode(coded, OriginRequester("https://example.com")); !errors.Is(err, vault.ErrNoTOTP) {
		t.Fatalf("code after the setup was removed: got %v, want ErrNoTOTP", err)
	}
	if err := service.DeleteItem(coded); err != nil {
		t.Fatal(err)
	}
	if _, err := service.OneTimeCode(coded, OriginRequester("https://example.com")); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("code of a deleted credential: got %v, want ErrNotFound", err)
	}
}

func TestNamesSiteCoversEveryCredentialSiteAndCardSite(t *testing.T) {
	service, _ := readyVault(t)
	createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://example.com", "https://login.example.org"}})
	if _, err := service.CreateCard(testCard(), nil); err != nil {
		t.Fatal(err)
	}
	for site, want := range map[string]bool{"example.com": true, "login.example.org": true, "jyskebank.dk": true, "example.org": false, "": false} {
		if named, err := service.NamesSite(site); err != nil || named != want {
			t.Fatalf("NamesSite(%q) = %v, %v, want %v", site, named, err, want)
		}
	}
}

func TestAddWebsiteAddsThePagesOriginOnceAndThePageThenMatches(t *testing.T) {
	service, files, _ := captureVault(t)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.org"}, Login: "alex"})
	page := OriginRequester("https://example.com")
	if got := suggestionsFor(t, service, page, PurposeSignIn); len(got) != 0 {
		t.Fatalf("suggestions before the site was added = %+v", got)
	}
	if _, err := service.FillCredential(id, page); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("fill before the site was added: got %v, want ErrNoMatch", err)
	}
	if err := service.AddWebsite(id, "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if got := suggestionsFor(t, service, page, PurposeSignIn); len(got) != 1 || got[0].ID != id || !got[0].Exact {
		t.Fatalf("suggestions after the site was added = %+v", got)
	}
	if websites := readTestCredential(t, service, id).Websites; !slices.Equal(websites, []string{"https://mail.example.org", "example.com"}) {
		t.Fatalf("websites after the site was added = %q", websites)
	}
	if count, err := service.AwaitVaultChange(t.Context(), 0); err != nil || count != 1 {
		t.Fatalf("change count after adding a site = %d, error = %v", count, err)
	}

	container := bytes.Clone(files.data)
	for _, again := range []string{"https://example.com", "https://www.example.com/login"} {
		if err := service.AddWebsite(id, again); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(files.data, container) || !quietFor(t, service, 1) {
		t.Fatal("adding a site the credential names already wrote the vault")
	}

	identity, err := service.CreateIdentity(testIdentity(), nil)
	if err != nil {
		t.Fatal(err)
	}
	full := make([]string, vault.MaxCredentialWebsites)
	for i := range full {
		full[i] = fmt.Sprintf("https://site%02d.example", i)
	}
	crowded := createTestCredential(t, service, vault.CredentialInput{Label: "Crowded", Websites: full})
	container = bytes.Clone(files.data)
	for name, test := range map[string]struct {
		id     vault.ID
		origin string
		want   error
	}{
		"an identity":             {identity, "https://example.net", vault.ErrNotFound},
		"a missing credential":    {vault.ID{0xff}, "https://example.net", vault.ErrNotFound},
		"no origin":               {id, "", vault.ErrInvalidInput},
		"an origin of no site":    {id, "android:apk-key-hash:abc", vault.ErrInvalidInput},
		"websites past the limit": {crowded, "https://example.net", vault.ErrInvalidInput},
	} {
		if err := service.AddWebsite(test.id, test.origin); !errors.Is(err, test.want) {
			t.Fatalf("%s: got %v, want %v", name, err, test.want)
		}
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("a refused addition wrote the vault")
	}
	service.Lock()
	if err := service.AddWebsite(id, "https://example.net"); !errors.Is(err, ErrNotReady) {
		t.Fatalf("an addition while locked: got %v, want ErrNotReady", err)
	}
}

func TestAPlatformsPlainHTTPPageMatchesOnlyCredentialsSavedOverHTTPForItsSiteAndPort(t *testing.T) {
	service, _ := readyVault(t)
	secure := createTestCredential(t, service, vault.CredentialInput{Label: "Secure", Websites: []string{"https://example.com", "example.com/login"}, TOTP: standardSecret})
	sibling := createTestCredential(t, service, vault.CredentialInput{Label: "Sibling", Websites: []string{"https://bank.example.com", "http://intranet.example.com"}, TOTP: standardSecret})
	plain := createTestCredential(t, service, vault.CredentialInput{Label: "Plain", Websites: []string{"https://example.org", "HTTP://www.example.com:80/login"}, TOTP: standardSecret})
	page := PlatformOriginRequester("http://example.com")

	for _, purpose := range []Purpose{PurposeSignIn, PurposeCode} {
		if got := suggestionIDs(suggestionsFor(t, service, page, purpose)); !slices.Equal(got, []vault.ID{plain}) {
			t.Fatalf("a platform's http page is offered %v for purpose %d, want only the credential saved over http for its site", got, purpose)
		}
	}
	if _, err := service.FillCredential(plain, page); err != nil {
		t.Fatal(err)
	}
	if _, err := service.OneTimeCode(plain, page); err != nil {
		t.Fatal(err)
	}
	for name, test := range map[string]struct {
		id   vault.ID
		page Requester
	}{
		"an https credential":            {secure, page},
		"another port":                   {plain, PlatformOriginRequester("http://example.com:8080")},
		"a sibling subdomain":            {sibling, PlatformOriginRequester("http://evil.example.com")},
		"the parent of an http website":  {sibling, page},
		"the http page of an https site": {sibling, PlatformOriginRequester("http://bank.example.com")},
		"a subdomain of an http website": {sibling, PlatformOriginRequester("http://login.intranet.example.com")},
	} {
		if got := suggestionsFor(t, service, test.page, PurposeSignIn); slices.Contains(suggestionIDs(got), test.id) {
			t.Fatalf("%s is offered on %+v: %+v", name, test.page, got)
		}
		if _, err := service.FillCredential(test.id, test.page); !errors.Is(err, ErrNoMatch) {
			t.Fatalf("filling %s: got %v, want ErrNoMatch", name, err)
		}
		if _, err := service.OneTimeCode(test.id, test.page); !errors.Is(err, ErrNoMatch) {
			t.Fatalf("a code of %s: got %v, want ErrNoMatch", name, err)
		}
	}
	if _, err := service.FillCredential(sibling, PlatformOriginRequester("http://intranet.example.com")); err != nil {
		t.Fatal(err)
	}
	for _, requester := range []Requester{OriginRequester("http://example.com"), PlatformOriginRequester("https://example.com")} {
		if got := suggestionIDs(suggestionsFor(t, service, requester, PurposeSignIn)); len(got) != 3 {
			t.Fatalf("%+v is offered %v, want every credential", requester, got)
		}
	}

	if err := service.AddWebsite(secure, "http://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := service.AddWebsite(plain, "http://example.com:8080"); err != nil {
		t.Fatal(err)
	}
	if websites := readTestCredential(t, service, secure).Websites; !slices.Equal(websites, []string{"https://example.com", "example.com/login", "http://example.com"}) {
		t.Fatalf("websites after adding the http page = %q", websites)
	}
	if websites := readTestCredential(t, service, plain).Websites; !slices.Equal(websites, []string{"https://example.org", "HTTP://www.example.com:80/login", "http://example.com:8080"}) {
		t.Fatalf("websites after adding the http page on another port = %q", websites)
	}
	if _, err := service.FillCredential(secure, page); err != nil {
		t.Fatalf("filling once the owner added the http page: %v", err)
	}
	if _, err := service.FillCredential(plain, PlatformOriginRequester("http://example.com:8080")); err != nil {
		t.Fatalf("filling once the owner added the http page on another port: %v", err)
	}
}

func TestSiteReadsNeedAnOpenVault(t *testing.T) {
	service, _ := readyVault(t)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://example.com"}})
	service.Lock()
	for _, purpose := range []Purpose{PurposeSignIn, PurposeCode} {
		if _, err := service.Suggestions(OriginRequester("https://example.com"), purpose); !errors.Is(err, ErrNotReady) {
			t.Fatalf("suggestions for %d while locked: got %v, want ErrNotReady", purpose, err)
		}
	}
	if _, err := service.FillCredential(id, OriginRequester("https://example.com")); !errors.Is(err, ErrNotReady) {
		t.Fatalf("fill while locked: got %v, want ErrNotReady", err)
	}
	if _, err := service.OneTimeCode(id, OriginRequester("https://example.com")); !errors.Is(err, ErrNotReady) {
		t.Fatalf("code while locked: got %v, want ErrNotReady", err)
	}
	if _, err := service.NamesSite("example.com"); !errors.Is(err, ErrNotReady) {
		t.Fatalf("named site while locked: got %v, want ErrNotReady", err)
	}
}

func TestSuggestionsCarryTheTagsThatTellAccountsApart(t *testing.T) {
	service, _ := readyVault(t)
	personal := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{"example.com"}, Login: "alex", Password: "a", Tags: []string{"Personal"}})
	work := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{"example.com"}, Login: "sam", Password: "b", Tags: []string{"Work"}})
	tags := map[vault.ID][]string{}
	for _, suggestion := range suggestionsFor(t, service, OriginRequester("https://login.example.com"), PurposeSignIn) {
		tags[suggestion.ID] = suggestion.Tags
	}
	if !slices.Equal(tags[personal], []string{"Personal"}) || !slices.Equal(tags[work], []string{"Work"}) {
		t.Fatalf("suggested tags = %v", tags)
	}
}
