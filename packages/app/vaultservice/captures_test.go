package vaultservice

import (
	"bytes"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/dortanes/ravenpass/packages/vault"
)

const captureOrigin = "https://example.com"

// captureVault opens a new vault over files and device records the test can inspect.
func captureVault(t *testing.T) (*Service, *memoryFiles, *memoryKeys) {
	t.Helper()
	files, keys := &memoryFiles{}, newMemoryKeys()
	service := newTestService(t, files, keys)
	createTestVault(t, service)
	return service, files, keys
}

func offerFor(t *testing.T, service *Service, capture Capture) CaptureOffer {
	t.Helper()
	offer, err := service.CaptureOffer(capture)
	if err != nil {
		t.Fatal(err)
	}
	return offer
}

func targetIDs(offer CaptureOffer) []vault.ID {
	ids := make([]vault.ID, len(offer.Targets))
	for i, target := range offer.Targets {
		ids[i] = target.ID
	}
	return ids
}

func listedEntry(t *testing.T, service *Service, id vault.ID) vault.Entry {
	t.Helper()
	entries, err := service.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == id {
			return entry
		}
	}
	t.Fatalf("credential %s is not listed", id)
	return vault.Entry{}
}

func TestANewAccountSuggestsANewCredentialBesideTheSitesCredentials(t *testing.T) {
	service, _, _ := captureVault(t)
	alex := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{captureOrigin}, Login: "alex", Password: "alex-secret", Tags: []string{"Personal"}})
	work := createTestCredential(t, service, vault.CredentialInput{Label: "Work", Websites: []string{"https://login.example.com"}, Email: "alex@work.test", Password: "work-secret"})
	createTestCredential(t, service, vault.CredentialInput{Label: "Other", Websites: []string{"https://example.org"}, Login: "sam", Password: "typed"})
	offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "sam@example.test", Password: "typed"})
	want := CaptureOffer{Targets: []Target{
		{ID: alex, Label: "Example", Account: "alex", Action: TargetUpdate, Tags: []string{"Personal"}},
		{ID: work, Label: "Work", Account: "alex@work.test", Action: TargetUpdate},
	}}
	if !reflect.DeepEqual(offer, want) {
		t.Fatalf("offer = %+v, want %+v", offer, want)
	}
}

func TestAChangedPasswordSuggestsUpdatingTheAccountsCredential(t *testing.T) {
	service, _, _ := captureVault(t)
	other := createTestCredential(t, service, vault.CredentialInput{Label: "Another", Websites: []string{captureOrigin}, Login: "sam", Password: "sam-secret"})
	alex := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{"https://www.example.com/login"}, Login: "alex", Password: "old"})
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: other, lastUsedAt: 2000}}})
	offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "new"})
	if offer.Suggested != alex || !slices.Equal(targetIDs(offer), []vault.ID{alex, other}) {
		t.Fatalf("offer = %+v, want alex's credential first and suggested", offer)
	}
}

func TestAnEmailIsComparedWithoutCase(t *testing.T) {
	service, _, _ := captureVault(t)
	mail := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{captureOrigin}, Email: "Alex@Example.test", Password: "old"})
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex@example.TEST", Password: "new"}); offer.Suggested != mail {
		t.Fatalf("offer = %+v, want the credential with the email suggested", offer)
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex@example.test", Password: "old"}); !offer.Nothing {
		t.Fatalf("a known email and password offered %+v", offer)
	}
	login := createTestCredential(t, service, vault.CredentialInput{Label: "Login", Websites: []string{"https://example.net"}, Login: "Sam", Password: "old"})
	if offer := offerFor(t, service, Capture{Requester: OriginRequester("https://example.net"), Account: "sam", Password: "new"}); offer.Suggested != (vault.ID{}) || !slices.Equal(targetIDs(offer), []vault.ID{login}) {
		t.Fatalf("a login in another case = %+v, want no suggestion", offer)
	}
}

func TestAPasswordUsedOnAnotherSiteOffersAddingTheSite(t *testing.T) {
	service, _, _ := captureVault(t)
	elsewhere := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.org"}, Login: "alex", Password: "typed"})
	createTestCredential(t, service, vault.CredentialInput{Label: "Other password", Websites: []string{"https://example.org"}, Login: "alex", Password: "different"})
	createTestCredential(t, service, vault.CredentialInput{Label: "Other account", Websites: []string{"https://example.org"}, Login: "sam", Password: "typed"})
	full := make([]string, vault.MaxCredentialWebsites)
	for i := range full {
		full[i] = "https://site" + strings.Repeat("x", i) + ".example.net"
	}
	createTestCredential(t, service, vault.CredentialInput{Label: "Full", Websites: full, Login: "alex", Password: "typed"})
	offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "typed"})
	want := CaptureOffer{Targets: []Target{{ID: elsewhere, Label: "Mail", Account: "alex", Action: TargetAddSite}}, Suggested: elsewhere}
	if !reflect.DeepEqual(offer, want) {
		t.Fatalf("offer = %+v, want %+v", offer, want)
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Password: "typed"}); len(offer.Targets) != 0 || offer.Nothing {
		t.Fatalf("a capture without an account offered %+v", offer)
	}
}

func TestAPlatformsPlainHTTPPageSavesToAnHTTPSCredentialOnlyByAddingThePage(t *testing.T) {
	service, _, _ := captureVault(t)
	secure := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{captureOrigin}, Login: "alex", Password: "typed"})
	page := PlatformOriginRequester("http://example.com")

	changed := Capture{Requester: page, Account: "alex", Password: "changed"}
	if offer := offerFor(t, service, changed); offer.Nothing || len(offer.Targets) != 0 {
		t.Fatalf("a new password on the http page offered %+v, want only a new credential", offer)
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester("http://example.com"), Account: "alex", Password: "changed"}); offer.Suggested != secure || offer.Targets[0].Action != TargetUpdate {
		t.Fatalf("a new password on a browser's http page offered %+v, want the credential updated", offer)
	}
	if _, err := service.SaveCapture(changed, CaptureChoice{Target: secure}, ""); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("updating the https credential from the http page: got %v, want ErrNotFound", err)
	}
	if password := readTestCredential(t, service, secure).Password; password != "typed" {
		t.Fatalf("the https credential's password = %q after a refused update", password)
	}

	same := Capture{Requester: page, Account: "alex", Password: "typed"}
	want := CaptureOffer{Targets: []Target{{ID: secure, Label: "Example", Account: "alex", Action: TargetAddSite}}, Suggested: secure}
	if offer := offerFor(t, service, same); !reflect.DeepEqual(offer, want) {
		t.Fatalf("the saved password on the http page offered %+v, want %+v", offer, want)
	}
	if created, err := service.SaveCapture(same, CaptureChoice{Target: secure}, ""); err != nil || created {
		t.Fatalf("adding the http page: created = %v, error = %v", created, err)
	}
	if websites := readTestCredential(t, service, secure).Websites; !slices.Equal(websites, []string{captureOrigin, "http://example.com"}) {
		t.Fatalf("websites after adding the http page = %q", websites)
	}
	if _, err := service.FillCredential(secure, page); err != nil {
		t.Fatalf("filling the http page once it was added: %v", err)
	}
	if offer := offerFor(t, service, same); !offer.Nothing {
		t.Fatalf("the saved password on the added http page offered %+v", offer)
	}
}

func TestAKnownAccountAndPasswordOfferNothing(t *testing.T) {
	service, _, _ := captureVault(t)
	createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{"https://login.example.com"}, Login: "alex", Password: "typed"})
	for _, capture := range []Capture{
		{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "typed"},
		{Requester: OriginRequester(captureOrigin), Password: "typed"},
		{Requester: OriginRequester(captureOrigin), Account: "alex", Password: ""},
		{Requester: OriginRequester(captureOrigin), Account: "alex", Password: strings.Repeat("p", vault.MaxPasswordLength+1)},
	} {
		if offer := offerFor(t, service, capture); !offer.Nothing || len(offer.Targets) != 0 {
			t.Fatalf("capture %q/%d offered %+v", capture.Account, len(capture.Password), offer)
		}
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "sam", Password: "typed"}); offer.Nothing {
		t.Fatal("another account with the same password offered nothing")
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: strings.Repeat("p", vault.MaxPasswordLength)}); offer.Nothing {
		t.Fatal("the longest password the vault holds offered nothing")
	}
}

func TestTheCurrentPasswordPicksTheUpdateTarget(t *testing.T) {
	service, _, _ := captureVault(t)
	recent := createTestCredential(t, service, vault.CredentialInput{Label: "Recent", Websites: []string{captureOrigin}, Login: "alex", Password: "other"})
	changed := createTestCredential(t, service, vault.CredentialInput{Label: "Changed", Websites: []string{captureOrigin}, Login: "alex", Password: "current"})
	sam := createTestCredential(t, service, vault.CredentialInput{Label: "Sam", Websites: []string{captureOrigin}, Login: "sam", Password: "sams-current"})
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: recent, lastUsedAt: 2000}}})
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "new", Current: "current"}); offer.Suggested != changed {
		t.Fatalf("offer = %+v, want the credential holding the current password suggested", offer)
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "new", Current: "sams-current"}); offer.Suggested != recent {
		t.Fatalf("offer = %+v, want another account's current password ignored", offer)
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Password: "new", Current: "sams-current"}); offer.Suggested != sam {
		t.Fatalf("offer without an account = %+v, want the credential holding the current password", offer)
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Password: "new"}); offer.Suggested != (vault.ID{}) {
		t.Fatalf("offer without an account or a current password = %+v, want a new credential", offer)
	}
}

func TestTargetsListUpdatesFirstHoldersFirstThenByRecentUseAndStopAtFive(t *testing.T) {
	service, _, _ := captureVault(t)
	var updates []vault.ID
	for _, label := range []string{"Delta", "Charlie", "Bravo", "Alpha"} {
		updates = append(updates, createTestCredential(t, service, vault.CredentialInput{Label: label, Websites: []string{captureOrigin}, Login: "sam", Password: "old"}))
	}
	holder := createTestCredential(t, service, vault.CredentialInput{Label: "Zulu", Websites: []string{captureOrigin}, Login: "alex", Password: "old"})
	addition := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://example.org"}, Login: "alex", Password: "typed"})
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: updates[2], lastUsedAt: 3000}, {id: updates[0], lastUsedAt: 1000}}})
	offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "typed"})
	want := []vault.ID{holder, updates[2], updates[0], updates[3], updates[1]}
	if got := targetIDs(offer); !slices.Equal(got, want) || offer.Suggested != holder {
		t.Fatalf("targets = %v suggesting %v, want %v suggesting the holder", got, offer.Suggested, want)
	}
	if err := service.DeleteItem(updates[1]); err != nil {
		t.Fatal(err)
	}
	offer = offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "typed"})
	if got := targetIDs(offer); len(got) != maxCaptureTargets || got[maxCaptureTargets-1] != addition || offer.Targets[maxCaptureTargets-1].Action != TargetAddSite {
		t.Fatalf("targets = %+v, want the add-site target last", offer.Targets)
	}
}

func TestComparingACaptureWritesNothingAndLeavesTheSelectionAndUseAlone(t *testing.T) {
	service, files, keys := captureVault(t)
	open := createTestCredential(t, service, vault.CredentialInput{Label: "Open", Websites: []string{"https://example.org"}, Login: "alex", Password: "typed"})
	createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{captureOrigin}, Login: "alex", Password: "old"})
	ticket, err := service.Select(open)
	if err != nil {
		t.Fatal(err)
	}
	container, usage := bytes.Clone(files.data), maps.Clone(keys.usage)
	offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "typed"})
	if !bytes.Equal(files.data, container) || !maps.EqualFunc(keys.usage, usage, bytes.Equal) {
		t.Fatal("comparing a capture wrote to the vault or the usage record")
	}
	if used, err := service.Usage(); err != nil || len(used) != 0 {
		t.Fatalf("comparing a capture recorded usage %v, error = %v", used, err)
	}
	if credential, err := service.ReadSelected(ticket); err != nil || credential.ID != open {
		t.Fatalf("the selected credential after a comparison = %+v, error = %v", credential, err)
	}
}

func TestSavingANewCredentialUsesTheEditedNameAndAccount(t *testing.T) {
	service, _, _ := captureVault(t)
	work := createTestGroup(t, service, "Work")
	created, err := service.SaveCapture(Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "typed"}, CaptureChoice{Name: "My example", Account: "alex@example.test"}, work.String())
	if err != nil || !created {
		t.Fatalf("save = %v, error = %v", created, err)
	}
	entries, err := service.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %+v, error = %v", entries, err)
	}
	credential := readTestCredential(t, service, entries[0].ID)
	want := vault.CredentialInput{Label: "My example", Websites: []string{"example.com"}, Email: "alex@example.test", Password: "typed"}
	if !reflect.DeepEqual(credential.CredentialInput, want) || !slices.Equal(entries[0].Groups, []vault.ID{work}) {
		t.Fatalf("created %+v in %v, want %+v in Work", credential.CredentialInput, entries[0].Groups, want)
	}

	if _, err := service.SaveCapture(Capture{Requester: OriginRequester("https://example.org"), Account: "alex", Password: "typed"}, CaptureChoice{Name: "Login", Account: "alex"}, vault.ID{0xee}.String()); err != nil {
		t.Fatal(err)
	}
	entries, err = service.List()
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries = %+v, error = %v", entries, err)
	}
	if credential := readTestCredential(t, service, entries[1].ID); credential.Login != "alex" || credential.Email != "" || len(entries[1].Groups) != 0 {
		t.Fatalf("a login account in a group that is gone = %+v in %v", credential.CredentialInput, entries[1].Groups)
	}
}

func TestSavingToAnUpdateTargetChangesOnlyItsPassword(t *testing.T) {
	service, _, _ := captureVault(t)
	work := createTestGroup(t, service, "Work")
	id, err := service.CreateCredential(vault.CredentialInput{
		Label: "Example", Websites: []string{captureOrigin, "https://example.org"}, Login: "alex", Email: "alex@example.test",
		Password: "old", Notes: "kept", TOTP: standardSecret,
	}, []vault.ID{work})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetPinned(id, true); err != nil {
		t.Fatal(err)
	}
	before, listed := readTestCredential(t, service, id), listedEntry(t, service, id)
	created, err := service.SaveCapture(Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "new"}, CaptureChoice{Target: id, Name: "Ignored", Account: "ignored"}, "")
	if err != nil || created {
		t.Fatalf("save = %v, error = %v", created, err)
	}
	after := readTestCredential(t, service, id)
	want := before
	want.Password = "new"
	if !reflect.DeepEqual(after, want) || !reflect.DeepEqual(listedEntry(t, service, id), listed) {
		t.Fatalf("after the update = %+v, want %+v", after, want)
	}
}

func TestSavingToAnAddSiteTargetOnlyAddsTheOrigin(t *testing.T) {
	service, _, _ := captureVault(t)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.org"}, Login: "alex", Password: "typed", Notes: "kept"})
	before := readTestCredential(t, service, id)
	created, err := service.SaveCapture(Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "typed"}, CaptureChoice{Target: id, Name: "Ignored", Account: "ignored"}, "")
	if err != nil || created {
		t.Fatalf("save = %v, error = %v", created, err)
	}
	want := before
	want.Websites = []string{"https://mail.example.org", "example.com"}
	if after := readTestCredential(t, service, id); !reflect.DeepEqual(after, want) {
		t.Fatalf("after adding the site = %+v, want %+v", after, want)
	}
	if offer := offerFor(t, service, Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "typed"}); !offer.Nothing {
		t.Fatalf("the added site still offers %+v", offer)
	}
}

func TestASaveThatIsRefusedWritesNothing(t *testing.T) {
	service, files, _ := captureVault(t)
	updated := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Websites: []string{captureOrigin}, Login: "alex", Password: "old"})
	unrelated := createTestCredential(t, service, vault.CredentialInput{Label: "Other", Websites: []string{"https://example.org"}, Login: "sam", Password: "other"})
	capture := Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "new"}
	tests := []struct {
		choice CaptureChoice
		want   error
	}{
		{CaptureChoice{Target: unrelated}, vault.ErrNotFound},
		{CaptureChoice{Target: vault.ID{0xff}}, vault.ErrNotFound},
		{CaptureChoice{Name: ""}, ErrNameRefused},
		{CaptureChoice{Name: "   "}, ErrNameRefused},
		{CaptureChoice{Name: strings.Repeat("n", vault.MaxLabelLength+1)}, ErrNameRefused},
		{CaptureChoice{Name: "Example", Account: strings.Repeat("a", vault.MaxLoginLength+1)}, ErrAccountRefused},
		{CaptureChoice{Name: "Example", Account: strings.Repeat("a", vault.MaxEmailLength-len("@example.test")+1) + "@example.test"}, ErrAccountRefused},
	}
	container := bytes.Clone(files.data)
	for _, test := range tests {
		if _, err := service.SaveCapture(capture, test.choice, ""); !errors.Is(err, test.want) {
			t.Fatalf("save %+v: got %v, want %v", test.choice, err, test.want)
		}
	}
	if _, err := service.SaveCapture(Capture{Requester: OriginRequester(captureOrigin), Account: "alex", Password: "old"}, CaptureChoice{Target: updated}, ""); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("a save of a known password: got %v, want ErrNotFound", err)
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("a refused save wrote the vault")
	}
	longest := strings.Repeat("a", vault.MaxEmailLength-len("@example.test")) + "@example.test"
	if _, err := service.SaveCapture(capture, CaptureChoice{Name: "Example", Account: longest}, ""); err != nil {
		t.Fatalf("the longest email: %v", err)
	}
	service.Lock()
	if _, err := service.SaveCapture(capture, CaptureChoice{Target: updated}, ""); !errors.Is(err, ErrNotReady) {
		t.Fatalf("a save while locked: got %v, want ErrNotReady", err)
	}
	if _, err := service.CaptureOffer(capture); !errors.Is(err, ErrNotReady) {
		t.Fatalf("an offer while locked: got %v, want ErrNotReady", err)
	}
}

func TestPageSiteAndSiteLabel(t *testing.T) {
	for origin, want := range map[string]string{
		"https://www.example.com":       "example.com",
		"https://xn--bcher-kva.example": "xn--bcher-kva.example",
		"https://exa_mple.com":          "https://exa_mple.com",
		"http://login.example.com:8080": "login.example.com",
	} {
		if got := PageSite(origin); got != want {
			t.Fatalf("PageSite(%q) = %q, want %q", origin, got, want)
		}
	}
	if got := siteLabel("xn--bcher-kva.example"); got != "bücher.example" {
		t.Fatalf("label of an IDN = %q", got)
	}
	long := strings.Repeat("xn--bcher-kva.", 20) + "example"
	label := siteLabel(long)
	want := string([]rune(strings.Repeat("bücher.", 20) + "example")[:vault.MaxLabelLength])
	if label != want || utf8.RuneCountInString(label) != vault.MaxLabelLength || !utf8.ValidString(label) {
		t.Fatalf("label of a long site = %q (%d characters)", label, utf8.RuneCountInString(label))
	}
}
