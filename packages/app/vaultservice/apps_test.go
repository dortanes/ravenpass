package vaultservice

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dortanes/ravenpass/packages/vault"
)

const mailPackage = "com.example.mail"

var (
	mailSigner    = [32]byte{0x11}
	rotatedSigner = [32]byte{0x22}
	strangeSigner = [32]byte{0x33}
	mailLink      = vault.App{Package: mailPackage, Signer: mailSigner}
)

// mailApp is the mail app signed with its own certificate, with the sites verified to name it.
func mailApp(sites ...string) Requester {
	return AppRequester(mailPackage, [][32]byte{mailSigner}, sites)
}

// lookAlike is an app of the mail app's package name signed with another certificate.
func lookAlike() Requester {
	return AppRequester(mailPackage, [][32]byte{strangeSigner}, nil)
}

func TestAnAppMatchesItsLinksExactlyAndItsVerifiedSitesAsAnOriginWould(t *testing.T) {
	service, _ := readyVault(t)
	linked := createTestCredential(t, service, vault.CredentialInput{Label: "Mail app", Login: "alex", Apps: []vault.App{mailLink}})
	site := createTestCredential(t, service, vault.CredentialInput{Label: "Mail site", Websites: []string{"https://mail.example.com"}, Login: "sam"})
	domain := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{"https://example.org", "https://www.example.com"}, Email: "kim@example.test"})
	createTestCredential(t, service, vault.CredentialInput{Label: "Other", Websites: []string{"https://example.net"}, Login: "otto"})

	if got, want := suggestionsFor(t, service, mailApp(), PurposeSignIn), []Suggestion{{ID: linked, Label: "Mail app", Account: "alex", Exact: true}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("suggestions without verified sites = %+v, want %+v", got, want)
	}
	want := []Suggestion{
		{ID: linked, Label: "Mail app", Account: "alex", Exact: true},
		{ID: site, Label: "Mail site", Account: "sam", Site: "mail.example.com", Exact: true},
		{ID: domain, Label: "Example", Account: "kim@example.test", Site: "example.com"},
	}
	if got := suggestionsFor(t, service, mailApp("mail.example.com"), PurposeSignIn); !reflect.DeepEqual(got, want) {
		t.Fatalf("suggestions with a verified site = %+v, want %+v", got, want)
	}
	if got := suggestionsFor(t, service, AppRequester(mailPackage, [][32]byte{strangeSigner, mailSigner}, nil), PurposeSignIn); len(got) != 1 || got[0].ID != linked {
		t.Fatalf("suggestions for an app signed with its linked certificate among others = %+v", got)
	}
	for name, requester := range map[string]Requester{
		"a look-alike":               lookAlike(),
		"another package":            AppRequester("com.example.chat", [][32]byte{mailSigner}, nil),
		"an app without a package":   AppRequester("", [][32]byte{mailSigner}, nil),
		"the zero requester":         {},
		"a site no credential names": AppRequester("com.example.chat", [][32]byte{mailSigner}, []string{"example.net.attacker.test"}),
	} {
		if got := suggestionsFor(t, service, requester, PurposeSignIn); len(got) != 0 {
			t.Fatalf("suggestions for %s = %+v", name, got)
		}
	}
}

func TestAnAppIsFilledOnlyWhileItStillMatches(t *testing.T) {
	service, _ := readyVault(t)
	linked := createTestCredential(t, service, vault.CredentialInput{Label: "Mail app", Login: "alex", Password: "app-secret", TOTP: standardSecret, Apps: []vault.App{mailLink}})
	site := createTestCredential(t, service, vault.CredentialInput{Label: "Mail site", Websites: []string{"https://mail.example.com"}, Password: "site-secret"})

	fill, err := service.FillCredential(linked, mailApp())
	if err != nil || fill != (Fill{Login: "alex", Password: "app-secret"}) {
		t.Fatalf("fill for the linked app = %+v, error = %v", fill, err)
	}
	if fill, err := service.FillCredential(site, mailApp("mail.example.com")); err != nil || fill.Password != "site-secret" {
		t.Fatalf("fill for a verified site = %+v, error = %v", fill, err)
	}
	if _, err := service.OneTimeCode(linked, mailApp()); err != nil {
		t.Fatalf("code for the linked app: %v", err)
	}
	usage, err := service.Usage()
	if err != nil || usage[linked] == 0 || usage[site] == 0 {
		t.Fatalf("usage after the fills = %v, error = %v", usage, err)
	}
	for name, attempt := range map[string]func() error{
		"fill for a look-alike": func() error { _, err := service.FillCredential(linked, lookAlike()); return err },
		"code for a look-alike": func() error { _, err := service.OneTimeCode(linked, lookAlike()); return err },
		"fill for a site the app was not verified for": func() error {
			_, err := service.FillCredential(site, mailApp())
			return err
		},
	} {
		if err := attempt(); !errors.Is(err, ErrNoMatch) {
			t.Fatalf("%s: got %v, want ErrNoMatch", name, err)
		}
	}
	unlinked := []vault.App{}
	if err := service.EditCredential(linked, vault.CredentialPatch{Apps: &unlinked}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.FillCredential(linked, mailApp()); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("fill after the link was removed: got %v, want ErrNoMatch", err)
	}
}

func TestANewCredentialSavedFromAnAppLinksIt(t *testing.T) {
	service, _, _ := captureVault(t)
	capture := Capture{Requester: AppRequester(mailPackage, [][32]byte{mailSigner, rotatedSigner}, nil), Account: "alex@example.test", Password: "typed"}
	if offer := offerFor(t, service, capture); offer.Nothing || len(offer.Targets) != 0 {
		t.Fatalf("offer for a new sign-in = %+v", offer)
	}
	created, err := service.SaveCapture(capture, CaptureChoice{Name: capture.Requester.Name(), Account: capture.Account}, "")
	if err != nil || !created {
		t.Fatalf("save = %v, error = %v", created, err)
	}
	entries, err := service.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %+v, error = %v", entries, err)
	}
	want := vault.CredentialInput{Label: mailPackage, Email: "alex@example.test", Password: "typed", Apps: []vault.App{mailLink, {Package: mailPackage, Signer: rotatedSigner}}}
	if credential := readTestCredential(t, service, entries[0].ID); !reflect.DeepEqual(credential.CredentialInput, want) {
		t.Fatalf("created %+v, want %+v", credential.CredentialInput, want)
	}
	if offer := offerFor(t, service, Capture{Requester: mailApp(), Account: "alex@example.test", Password: "typed"}); !offer.Nothing {
		t.Fatalf("the saved sign-in still offers %+v", offer)
	}
}

func TestSavingFromAnAppOverACredentialLinksTheApp(t *testing.T) {
	service, _, _ := captureVault(t)
	site := createTestCredential(t, service, vault.CredentialInput{Label: "Mail site", Websites: []string{"https://mail.example.com"}, Login: "alex", Password: "old"})
	elsewhere := createTestCredential(t, service, vault.CredentialInput{Label: "Chat", Websites: []string{"https://chat.example.org"}, Login: "alex", Password: "typed"})
	full := createTestCredential(t, service, vault.CredentialInput{Label: "Full", Login: "alex", Password: "typed", Apps: numberedLinks(vault.MaxCredentialApps)})

	offer := offerFor(t, service, Capture{Requester: mailApp("mail.example.com"), Account: "alex", Password: "typed"})
	want := []Target{
		{ID: site, Label: "Mail site", Account: "alex", Action: TargetUpdate},
		{ID: elsewhere, Label: "Chat", Account: "alex", Action: TargetAddSite},
	}
	if !reflect.DeepEqual(offer.Targets, want) || offer.Suggested != site {
		t.Fatalf("offer = %+v, want %+v without the full credential %s", offer, want, full)
	}

	if _, err := service.SaveCapture(Capture{Requester: mailApp("mail.example.com"), Account: "alex", Password: "new"}, CaptureChoice{Target: site}, ""); err != nil {
		t.Fatal(err)
	}
	updated := readTestCredential(t, service, site)
	if updated.Password != "new" || !slices.Equal(updated.Apps, []vault.App{mailLink}) || !slices.Equal(updated.Websites, []string{"https://mail.example.com"}) {
		t.Fatalf("after the update = %+v", updated.CredentialInput)
	}

	before := readTestCredential(t, service, elsewhere)
	if _, err := service.SaveCapture(Capture{Requester: AppRequester("com.example.chat", [][32]byte{rotatedSigner}, nil), Account: "alex", Password: "typed"}, CaptureChoice{Target: elsewhere}, ""); err != nil {
		t.Fatal(err)
	}
	added := before
	added.Apps = []vault.App{{Package: "com.example.chat", Signer: rotatedSigner}}
	if after := readTestCredential(t, service, elsewhere); !reflect.DeepEqual(after, added) {
		t.Fatalf("after adding the app = %+v, want %+v", after.CredentialInput, added.CredentialInput)
	}
}

func TestALookAlikeAppIsNotOfferedTheLinkedCredentialToUpdate(t *testing.T) {
	service, _, _ := captureVault(t)
	linked := createTestCredential(t, service, vault.CredentialInput{Label: "Mail app", Login: "alex", Password: "old", Apps: []vault.App{mailLink}})
	offer := offerFor(t, service, Capture{Requester: lookAlike(), Account: "alex", Password: "new"})
	if len(offer.Targets) != 0 {
		t.Fatalf("a look-alike was offered %+v", offer)
	}
	if _, err := service.SaveCapture(Capture{Requester: lookAlike(), Account: "alex", Password: "new"}, CaptureChoice{Target: linked}, ""); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("a look-alike's update: got %v, want ErrNotFound", err)
	}
	if credential := readTestCredential(t, service, linked); credential.Password != "old" {
		t.Fatalf("a look-alike changed the password to %q", credential.Password)
	}
}

func TestARequesterNamesTheCredentialItWouldCreate(t *testing.T) {
	tests := []struct {
		requester Requester
		want      string
	}{
		{OriginRequester("https://www.xn--bcher-kva.example/login"), "bücher.example"},
		{mailApp("mail.example.com", "example.org"), "mail.example.com"},
		{mailApp(), mailPackage},
		{AppRequester("a."+strings.Repeat("b", 200), nil, nil), "a." + strings.Repeat("b", vault.MaxLabelLength-2)},
	}
	for _, test := range tests {
		if got := test.requester.Name(); got != test.want {
			t.Fatalf("name = %q, want %q", got, test.want)
		}
	}
}

func numberedLinks(count int) []vault.App {
	links := make([]vault.App, count)
	for i := range links {
		links[i] = vault.App{Package: fmt.Sprintf("com.example.app%d", i), Signer: [32]byte{byte(i + 1)}}
	}
	return links
}

func TestLinkAppAddsEachSignerOnceAndTheAppThenMatches(t *testing.T) {
	service, files, _ := captureVault(t)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.com"}, Login: "alex"})
	if got := suggestionsFor(t, service, mailApp(), PurposeSignIn); len(got) != 0 {
		t.Fatalf("suggestions before the link = %+v", got)
	}
	if err := service.LinkApp(id, mailPackage, [][32]byte{mailSigner}); err != nil {
		t.Fatal(err)
	}
	if got := suggestionsFor(t, service, mailApp(), PurposeSignIn); len(got) != 1 || got[0].ID != id || !got[0].Exact {
		t.Fatalf("suggestions after the link = %+v", got)
	}
	if count, err := service.AwaitVaultChange(t.Context(), 0); err != nil || count != 1 {
		t.Fatalf("change count after a link = %d, error = %v", count, err)
	}

	container := bytes.Clone(files.data)
	if err := service.LinkApp(id, mailPackage, [][32]byte{mailSigner}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(files.data, container) || !quietFor(t, service, 1) {
		t.Fatal("linking an app the credential links already wrote the vault")
	}
	if err := service.LinkApp(id, mailPackage, [][32]byte{mailSigner, rotatedSigner}); err != nil {
		t.Fatal(err)
	}
	if apps := readTestCredential(t, service, id).Apps; !slices.Equal(apps, []vault.App{mailLink, {Package: mailPackage, Signer: rotatedSigner}}) {
		t.Fatalf("links after a rotated key = %+v", apps)
	}

	identity, err := service.CreateIdentity(testIdentity(), nil)
	if err != nil {
		t.Fatal(err)
	}
	full := createTestCredential(t, service, vault.CredentialInput{Label: "Full", Apps: numberedLinks(vault.MaxCredentialApps)})
	container = bytes.Clone(files.data)
	for name, test := range map[string]struct {
		id      vault.ID
		pkg     string
		signers [][32]byte
		want    error
	}{
		"an identity":               {identity, mailPackage, [][32]byte{mailSigner}, vault.ErrNotFound},
		"a missing credential":      {vault.ID{0xff}, mailPackage, [][32]byte{mailSigner}, vault.ErrNotFound},
		"no signer":                 {id, mailPackage, nil, vault.ErrInvalidInput},
		"no package":                {id, "", [][32]byte{mailSigner}, vault.ErrInvalidInput},
		"a name that is no package": {id, "mail", [][32]byte{mailSigner}, vault.ErrInvalidInput},
		"links past the limit":      {full, mailPackage, [][32]byte{mailSigner}, vault.ErrInvalidInput},
	} {
		if err := service.LinkApp(test.id, test.pkg, test.signers); !errors.Is(err, test.want) {
			t.Fatalf("%s: got %v, want %v", name, err, test.want)
		}
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("a refused link wrote the vault")
	}
	service.Lock()
	if err := service.LinkApp(id, mailPackage, [][32]byte{mailSigner}); !errors.Is(err, ErrNotReady) {
		t.Fatalf("a link while locked: got %v, want ErrNotReady", err)
	}
}

func TestSearchFindsCredentialsByLabelAccountAndSiteWithoutCase(t *testing.T) {
	service, _ := readyVault(t)
	mail := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.com", "https://webmail.example.org"}, Login: "alex"})
	shop := createTestCredential(t, service, vault.CredentialInput{Label: "Shop", Email: "Kim@Example.test", Notes: "mail"})
	bank := createTestCredential(t, service, vault.CredentialInput{Label: "Bank", Login: "MAILROOM"})
	if _, err := service.CreateIdentity(vault.IdentityInput{Label: "Mail identity", Emails: []string{"mail@example.test"}}, nil); err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string][]vault.ID{
		"mail":        {bank, mail},
		"  WEBMAIL  ": {mail},
		"kim@example": {shop},
		"example":     {mail, shop},
		"nothing":     {},
	} {
		found, err := service.SearchCredentials(query, PurposeSignIn)
		if err != nil {
			t.Fatal(err)
		}
		if got := suggestionIDs(found); !slices.Equal(got, want) {
			t.Fatalf("search %q = %v, want %v", query, got, want)
		}
	}
	found, err := service.SearchCredentials("webmail", PurposeSignIn)
	if err != nil || !reflect.DeepEqual(found, []Suggestion{{ID: mail, Label: "Mail", Account: "alex", Site: "mail.example.com"}}) {
		t.Fatalf("search result = %+v, error = %v", found, err)
	}
}

func TestASearchForACodeListsOnlyCredentialsWithACodeSetup(t *testing.T) {
	service, _ := readyVault(t)
	coded := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Login: "alex", TOTP: standardSecret})
	createTestCredential(t, service, vault.CredentialInput{Label: "Mail archive", Login: "alex"})
	for _, query := range []string{"", "mail"} {
		found, err := service.SearchCredentials(query, PurposeCode)
		if err != nil || len(found) != 1 || found[0].ID != coded || found[0].Code.Digits != 6 {
			t.Fatalf("search %q for a code = %+v, error = %v", query, found, err)
		}
	}
	if found, err := service.SearchCredentials("mail", PurposeSignIn); err != nil || len(found) != 2 || found[0].Code.Digits != 0 {
		t.Fatalf("search for a sign-in = %+v, error = %v", found, err)
	}
}

func TestAnEmptySearchListsPinnedThenRecentThenByLabelAndStopsAtFifty(t *testing.T) {
	service, _ := readyVault(t)
	for i := range maxSuggestions {
		createTestCredential(t, service, vault.CredentialInput{Label: fmt.Sprintf("Account %02d", i)})
	}
	recent := createTestCredential(t, service, vault.CredentialInput{Label: "Zulu recent"})
	pinned := createTestCredential(t, service, vault.CredentialInput{Label: "Zulu pinned"})
	if err := service.SetPinned(pinned, true); err != nil {
		t.Fatal(err)
	}
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: recent, lastUsedAt: 1000}}})
	found, err := service.SearchCredentials("", PurposeSignIn)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != maxSuggestions || found[0].ID != pinned || found[1].ID != recent || found[2].Label != "Account 00" {
		t.Fatalf("%d results starting %+v", len(found), found[:3])
	}
	if last := found[maxSuggestions-1]; last.Label != fmt.Sprintf("Account %02d", maxSuggestions-3) {
		t.Fatalf("last result = %+v", last)
	}
}

func TestCredentialSitesListsEachSiteOnce(t *testing.T) {
	service, _ := readyVault(t)
	createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.com", "https://www.example.org"}})
	createTestCredential(t, service, vault.CredentialInput{Label: "Webmail", Websites: []string{"https://example.org/login", "https://chat.example.net"}})
	createTestCredential(t, service, vault.CredentialInput{Label: "Bare", Apps: []vault.App{mailLink}})
	card := testCard()
	card.BankSite = "https://bank.example"
	if _, err := service.CreateCard(card, nil); err != nil {
		t.Fatal(err)
	}
	sites, err := service.CredentialSites()
	if err != nil || !slices.Equal(sites, []string{"chat.example.net", "example.org", "mail.example.com"}) {
		t.Fatalf("sites = %q, error = %v", sites, err)
	}
	service.Lock()
	if _, err := service.CredentialSites(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("sites while locked: got %v, want ErrNotReady", err)
	}
	if _, err := service.SearchCredentials("", PurposeSignIn); !errors.Is(err, ErrNotReady) {
		t.Fatalf("search while locked: got %v, want ErrNotReady", err)
	}
}
