package autofill

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/dortanes/ravenpass/packages/app/captures"
	"github.com/dortanes/ravenpass/packages/app/devicerecords"
	"github.com/dortanes/ravenpass/packages/app/localfile"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/unlock/unlocktest"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/authenticator"
	"github.com/dortanes/ravenpass/packages/vault"
)

const (
	mailPackage = "com.example.mail"
	passkeyRPID = "example.com"
	appOrigin   = "android:apk-key-hash:x2xzU8dM3zY6Gq-nN0AjL0wYfF5sT0c3Z1d9Kq7pQ1E"
)

var (
	mailSigner    = [32]byte{0x11}
	strangeSigner = [32]byte{0x33}
	mailApp       = App{Package: mailPackage, Signers: [][32]byte{mailSigner}}
	lookAlike     = App{Package: mailPackage, Signers: [][32]byte{strangeSigner}}
	platformHash  = sha256.Sum256([]byte("client data a platform built"))
)

func newPasskeyCreation() PasskeyCreation {
	return PasskeyCreation{
		RPID: passkeyRPID, RPName: "Example",
		User:           PasskeyUser{Handle: []byte{7, 7}, Name: "alex@example.com", DisplayName: "Alex"},
		ClientDataHash: &platformHash,
	}
}

// verifySignature checks a WebAuthn signature over authData || clientDataHash with an SPKI public key.
func verifySignature(t *testing.T, publicKey, authData, signature []byte, clientDataHash [32]byte) {
	t.Helper()
	parsed, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("the public key is a %T", parsed)
	}
	signed := sha256.Sum256(append(bytes.Clone(authData), clientDataHash[:]...))
	if !ecdsa.VerifyASN1(key, signed[:], signature) {
		t.Fatal("the signature does not verify with the passkey's public key")
	}
}

// ownedDevice is a device whose owner can always authenticate.
type ownedDevice struct{}

func (ownedDevice) DeviceOwnerAvailable() bool { return true }

// backedUp keeps every file in the system's backups.
type backedUp struct{}

func (backedUp) Exclude(string) error { return nil }

type recordedGroup string

func (g recordedGroup) DefaultGroup() string { return string(g) }

// openVault creates a vault unlocked by the device's authentication, and a service adding to group.
func openVault(t *testing.T, group string) (*vaultservice.Service, *Service) {
	t.Helper()
	directory := t.TempDir()
	files, err := storage.NewManager(filepath.Join(directory, "storage.json"), storage.Target{Kind: storage.LocalFile, Path: filepath.Join(directory, localfile.DefaultVaultName)}, vault.MaxContainerBytes, localfile.Backend{})
	if err != nil {
		t.Fatal(err)
	}
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	records, err := devicerecords.New(filepath.Join(directory, "device.json"), backedUp{})
	if err != nil {
		t.Fatal(err)
	}
	core, err := vaultservice.New(files, records, vaultservice.Device{Owner: ownedDevice{}, PIN: unlocktest.NewBinding(), Platform: unlocktest.NewPresenceBinding()})
	if err != nil {
		t.Fatal(err)
	}
	phrase, err := core.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.ConfirmCreation(phrase, vaultservice.MethodChoice{Biometry: true}); err != nil {
		t.Fatal(err)
	}
	service, err := New(core, recordedGroup(group))
	if err != nil {
		t.Fatal(err)
	}
	return core, service
}

func create(t *testing.T, core *vaultservice.Service, input vault.CredentialInput) string {
	t.Helper()
	if input.Password == "" {
		input.Password = "secret"
	}
	id, err := core.CreateCredential(input, nil)
	if err != nil {
		t.Fatalf("create %q: %v", input.Label, err)
	}
	return id.String()
}

func read(t *testing.T, core *vaultservice.Service, id string) vault.Credential {
	t.Helper()
	parsed, err := vault.ParseID(id)
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := core.Select(parsed)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := core.ReadSelected(ticket)
	if err != nil {
		t.Fatal(err)
	}
	return credential
}

func suggestionIDs(suggestions []Suggestion) []string {
	ids := make([]string, len(suggestions))
	for i, suggestion := range suggestions {
		ids[i] = suggestion.ID
	}
	return ids
}

func TestNewRequiresTheVaultAndTheGroupChoice(t *testing.T) {
	core, _ := openVault(t, "")
	if _, err := New(nil, recordedGroup("")); err == nil {
		t.Fatal("a service without a vault was composed")
	}
	if _, err := New(core, nil); err == nil {
		t.Fatal("a service without a group choice was composed")
	}
}

func TestALockedVaultAnswersNothingButErrLocked(t *testing.T) {
	core, service := openVault(t, "")
	id := create(t, core, vault.CredentialInput{Label: "Mail", Websites: []string{"https://example.com"}, TOTP: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", Apps: []vault.App{{Package: mailPackage, Signer: mailSigner}}})
	offer, err := service.Hold(Capture{Requester: Requester{App: mailApp}, Account: "alex", Password: "typed"})
	if err != nil {
		t.Fatal(err)
	}
	core.Lock()
	if service.Open() {
		t.Fatal("a locked vault reads as open")
	}
	app := Requester{App: mailApp}
	for name, attempt := range map[string]func() error{
		"suggest": func() error { _, err := service.Suggest(app, false); return err },
		"codes":   func() error { _, err := service.Suggest(app, true); return err },
		"fill":    func() error { _, err := service.Fill(id, app); return err },
		"code":    func() error { _, err := service.OneTimeCode(id, app); return err },
		"search":  func() error { _, err := service.Search("", false); return err },
		"sites":   func() error { _, err := service.Sites(); return err },
		"link":    func() error { return service.Link(id, mailApp) },
		"add a site": func() error {
			return service.AddSite(id, Requester{Origin: "https://example.org"})
		},
		"hold": func() error {
			_, err := service.Hold(Capture{Requester: app, Account: "sam", Password: "typed"})
			return err
		},
		"save":     func() error { _, err := service.Save(offer.Token, Choice{Name: "Mail", Account: "alex"}); return err },
		"passkeys": func() error { _, err := service.Passkeys(passkeyRPID, nil); return err },
		"sign in with a passkey": func() error {
			_, err := service.SignPasskey(PasskeySignIn{RPID: passkeyRPID, ClientDataHash: &platformHash, ID: id, CredentialID: make([]byte, 16)})
			return err
		},
		"create a passkey": func() error { _, err := service.CreatePasskey(newPasskeyCreation()); return err },
	} {
		if err := attempt(); !errors.Is(err, ErrLocked) {
			t.Fatalf("%s while locked: got %v, want ErrLocked", name, err)
		}
	}
	if _, err := core.Unlock("test"); err != nil {
		t.Fatal(err)
	}
	if !service.Open() {
		t.Fatal("an unlocked vault reads as locked")
	}
	if created, err := service.Save(offer.Token, Choice{Name: "Mail", Account: "alex"}); err != nil || !created {
		t.Fatalf("a save refused while locked, after unlocking = %v, error = %v", created, err)
	}
}

func TestSuggestionsCarryNoValueAndFollowTheExtensionsOrder(t *testing.T) {
	core, service := openVault(t, "")
	exact := create(t, core, vault.CredentialInput{Label: "Zulu", Websites: []string{"https://example.com"}, Login: "zed"})
	domain := create(t, core, vault.CredentialInput{Label: "Alpha", Websites: []string{"https://login.example.com"}, Email: "al@example.test"})
	create(t, core, vault.CredentialInput{Label: "Other", Websites: []string{"https://example.org"}})
	got, err := service.Suggest(Requester{Origin: "https://example.com"}, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []Suggestion{
		{ID: exact, Label: "Zulu", Account: "zed", Site: "example.com", Exact: true},
		{ID: domain, Label: "Alpha", Account: "al@example.test", Site: "login.example.com"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("suggestions = %+v, want %+v", got, want)
	}
	extension, err := core.Suggestions(vaultservice.OriginRequester("https://example.com"), vaultservice.PurposeSignIn)
	if err != nil {
		t.Fatal(err)
	}
	for i, suggestion := range extension {
		if suggestion.ID.String() != got[i].ID {
			t.Fatalf("the extension is offered %+v, system autofill %+v", extension, got)
		}
	}
}

func TestFillReleasesALinkedAppsCredentialAfterCheckingTheMatchAndRecordsTheUse(t *testing.T) {
	core, service := openVault(t, "")
	id := create(t, core, vault.CredentialInput{Label: "Mail", Login: "alex", Email: "alex@example.test", Password: "app-secret", Apps: []vault.App{{Package: mailPackage, Signer: mailSigner}}})
	suggestions, err := service.Suggest(Requester{App: mailApp}, false)
	if err != nil || !reflect.DeepEqual(suggestions, []Suggestion{{ID: id, Label: "Mail", Account: "alex", Exact: true}}) {
		t.Fatalf("suggestions = %+v, error = %v", suggestions, err)
	}
	if suggestions, err := service.Suggest(Requester{App: lookAlike}, false); err != nil || len(suggestions) != 0 {
		t.Fatalf("a look-alike was suggested %+v, error = %v", suggestions, err)
	}
	if _, err := service.Fill(id, Requester{App: lookAlike}); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("a look-alike's fill: got %v, want ErrNoMatch", err)
	}
	if _, err := service.Fill(id, Requester{App: App{Package: mailPackage}}); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("a fill for an app without a signer: got %v, want ErrNoMatch", err)
	}
	if usage, err := core.Usage(); err != nil || len(usage) != 0 {
		t.Fatalf("refused fills recorded usage %v, error = %v", usage, err)
	}
	login, err := service.Fill(id, Requester{App: mailApp})
	if err != nil || login != (Login{Login: "alex", Email: "alex@example.test", Password: "app-secret"}) {
		t.Fatalf("fill = %+v, error = %v", login, err)
	}
	parsed, _ := vault.ParseID(id)
	if usage, err := core.Usage(); err != nil || usage[parsed] == 0 {
		t.Fatalf("usage after the fill = %v, error = %v", usage, err)
	}
	for _, missing := range []string{"", "mail", vault.ID{0xff}.String()} {
		if _, err := service.Fill(missing, Requester{App: mailApp}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("fill of %q: got %v, want ErrNotFound", missing, err)
		}
	}
}

func TestOneTimeCodesAreGeneratedForMatchingCredentialsWithASetup(t *testing.T) {
	core, service := openVault(t, "")
	coded := create(t, core, vault.CredentialInput{Label: "Coded", Websites: []string{"https://example.com"}, TOTP: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"})
	bare := create(t, core, vault.CredentialInput{Label: "Bare", Websites: []string{"https://example.com"}})
	page := Requester{Origin: "https://example.com"}
	if suggestions, err := service.Suggest(page, true); err != nil || !slices.Equal(suggestionIDs(suggestions), []string{coded}) {
		t.Fatalf("code suggestions = %+v, error = %v", suggestions, err)
	}
	before := time.Now()
	code, err := service.OneTimeCode(coded, page)
	if err != nil || len(code.Code) != 6 || code.ExpiresAt <= before.UnixMilli() {
		t.Fatalf("code = %+v, error = %v", code, err)
	}
	if _, err := service.OneTimeCode(bare, page); !errors.Is(err, ErrNoCode) {
		t.Fatalf("code of a credential without a setup: got %v, want ErrNoCode", err)
	}
	if _, err := service.OneTimeCode(coded, Requester{Origin: "https://example.org"}); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("code for another site: got %v, want ErrNoMatch", err)
	}
	if usage, err := core.Usage(); err != nil || len(usage) != 0 {
		t.Fatalf("codes recorded usage %v, error = %v", usage, err)
	}
}

func TestACredentialFilledOnASignInPageComesFirstOnItsCodePage(t *testing.T) {
	core, service := openVault(t, "")
	const setup = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	other := create(t, core, vault.CredentialInput{Label: "Alpha", Websites: []string{"https://example.com"}, Login: "al", TOTP: setup})
	filled := create(t, core, vault.CredentialInput{Label: "Zulu", Websites: []string{"https://example.com"}, Login: "zed", TOTP: setup})
	page := Requester{Origin: "https://example.com"}
	if suggestions, err := service.Suggest(page, true); err != nil || !slices.Equal(suggestionIDs(suggestions), []string{other, filled}) {
		t.Fatalf("code suggestions before any fill = %+v, error = %v", suggestions, err)
	}
	if _, err := service.Fill(filled, page); err != nil {
		t.Fatal(err)
	}
	if suggestions, err := service.Suggest(page, true); err != nil || !slices.Equal(suggestionIDs(suggestions), []string{filled, other}) {
		t.Fatalf("code suggestions after the sign-in was filled = %+v, error = %v", suggestions, err)
	}
}

func TestSearchFindsAnyCredentialAndStopsAtFifty(t *testing.T) {
	core, service := openVault(t, "")
	for i := range 51 {
		create(t, core, vault.CredentialInput{Label: fmt.Sprintf("Account %02d", i), Websites: []string{fmt.Sprintf("https://site%02d.example", i)}})
	}
	found, err := service.Search("", false)
	if err != nil || len(found) != 50 {
		t.Fatalf("%d results, error = %v", len(found), err)
	}
	found, err = service.Search("SITE07", false)
	if err != nil || len(found) != 1 || found[0].Label != "Account 07" || found[0].Site != "site07.example" || found[0].Exact {
		t.Fatalf("search = %+v, error = %v", found, err)
	}
}

func TestSitesListEachSiteOnce(t *testing.T) {
	core, service := openVault(t, "")
	create(t, core, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.com", "https://example.org"}})
	create(t, core, vault.CredentialInput{Label: "Webmail", Websites: []string{"https://www.example.org/login"}})
	sites, err := service.Sites()
	if err != nil || !slices.Equal(sites, []string{"example.org", "mail.example.com"}) {
		t.Fatalf("sites = %q, error = %v", sites, err)
	}
}

func TestAnAppLinkedFromSearchIsSuggestedNextTime(t *testing.T) {
	core, service := openVault(t, "")
	id := create(t, core, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.com"}, Login: "alex"})
	app := Requester{App: mailApp}
	if suggestions, err := service.Suggest(app, false); err != nil || len(suggestions) != 0 {
		t.Fatalf("suggestions before the link = %+v, error = %v", suggestions, err)
	}
	found, err := service.Search("mail", false)
	if err != nil || len(found) != 1 || found[0].ID != id {
		t.Fatalf("search = %+v, error = %v", found, err)
	}
	if err := service.Link(found[0].ID, mailApp); err != nil {
		t.Fatal(err)
	}
	if suggestions, err := service.Suggest(app, false); err != nil || !slices.Equal(suggestionIDs(suggestions), []string{id}) || !suggestions[0].Exact {
		t.Fatalf("suggestions after the link = %+v, error = %v", suggestions, err)
	}
	if suggestions, err := service.Suggest(Requester{App: lookAlike}, false); err != nil || len(suggestions) != 0 {
		t.Fatalf("a look-alike after the link was suggested %+v, error = %v", suggestions, err)
	}
	if err := service.Link("mail", mailApp); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a link to an unreadable identifier: got %v, want ErrNotFound", err)
	}
}

func TestASiteAddedFromSearchIsSuggestedAndFilledNextTime(t *testing.T) {
	core, service := openVault(t, "")
	id := create(t, core, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.org"}, Login: "alex", Password: "secret"})
	page := Requester{Origin: "https://example.com"}
	if _, err := service.Fill(id, page); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("a fill before the site was added: got %v, want ErrNoMatch", err)
	}
	if err := service.AddSite(id, page); err != nil {
		t.Fatal(err)
	}
	if suggestions, err := service.Suggest(page, false); err != nil || !slices.Equal(suggestionIDs(suggestions), []string{id}) || !suggestions[0].Exact {
		t.Fatalf("suggestions after the site was added = %+v, error = %v", suggestions, err)
	}
	if login, err := service.Fill(id, page); err != nil || login.Password != "secret" {
		t.Fatalf("fill after the site was added = %+v, error = %v", login, err)
	}
	if websites := read(t, core, id).Websites; !slices.Equal(websites, []string{"https://mail.example.org", "example.com"}) {
		t.Fatalf("websites = %q", websites)
	}
	for name, test := range map[string]struct {
		id        string
		requester Requester
		want      error
	}{
		"an app":                   {id, Requester{App: mailApp}, ErrInvalidRequest},
		"an unreadable identifier": {"mail", page, ErrNotFound},
		"a missing credential":     {vault.ID{0xff}.String(), page, ErrNotFound},
	} {
		if err := service.AddSite(test.id, test.requester); !errors.Is(err, test.want) {
			t.Fatalf("%s: got %v, want %v", name, err, test.want)
		}
	}
	if apps := read(t, core, id).Apps; apps != nil {
		t.Fatalf("adding a site for an app linked it: %+v", apps)
	}
}

func TestASignInHeldFromAnAppIsSavedWithTheAppLinkedInTheDefaultGroup(t *testing.T) {
	core, _ := openVault(t, "")
	work, err := core.CreateGroup("Work")
	if err != nil {
		t.Fatal(err)
	}
	service, err := New(core, recordedGroup(work.String()))
	if err != nil {
		t.Fatal(err)
	}
	offer, err := service.Hold(Capture{Requester: Requester{App: mailApp}, Account: "alex@example.test", Password: "typed"})
	if err != nil {
		t.Fatal(err)
	}
	if offer.Token == "" || offer.Name != mailPackage || len(offer.Targets) != 0 || offer.Suggested != "" {
		t.Fatalf("offer = %+v", offer)
	}
	created, err := service.Save(offer.Token, Choice{Name: "Mail", Account: "alex@example.test"})
	if err != nil || !created {
		t.Fatalf("save = %v, error = %v", created, err)
	}
	entries, err := core.List()
	if err != nil || len(entries) != 1 || !slices.Equal(entries[0].Groups, []vault.ID{work}) {
		t.Fatalf("entries = %+v, error = %v", entries, err)
	}
	want := vault.CredentialInput{Label: "Mail", Email: "alex@example.test", Password: "typed", Apps: []vault.App{{Package: mailPackage, Signer: mailSigner}}}
	if credential := read(t, core, entries[0].ID.String()); !reflect.DeepEqual(credential.CredentialInput, want) {
		t.Fatalf("saved %+v, want %+v", credential.CredentialInput, want)
	}
	if _, err := service.Save(offer.Token, Choice{Name: "Mail", Account: "alex@example.test"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a second save of the capture: got %v, want ErrNotFound", err)
	}
	if _, err := service.Hold(Capture{Requester: Requester{App: mailApp}, Account: "alex@example.test", Password: "typed"}); !errors.Is(err, ErrNothingToSave) {
		t.Fatalf("holding the saved sign-in again: got %v, want ErrNothingToSave", err)
	}
}

func TestSavingOverACredentialNeedsATargetTheOfferNamedAndLinksTheApp(t *testing.T) {
	core, service := openVault(t, "")
	site := create(t, core, vault.CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.com"}, Login: "alex", Password: "old", Tags: []string{"Work"}})
	unrelated := create(t, core, vault.CredentialInput{Label: "Other", Websites: []string{"https://example.org"}, Login: "sam", Password: "other"})
	capture := Capture{Requester: Requester{App: mailApp, Sites: []string{"mail.example.com"}}, Account: "alex", Password: "new"}
	offer, err := service.Hold(capture)
	if err != nil {
		t.Fatal(err)
	}
	if want := []Target{{ID: site, Label: "Mail", Account: "alex", Action: SaveUpdate, Tags: []string{"Work"}}}; !reflect.DeepEqual(offer.Targets, want) || offer.Suggested != site || offer.Name != "mail.example.com" {
		t.Fatalf("offer = %+v", offer)
	}
	for _, target := range []string{unrelated, "mail", vault.ID{0xff}.String()} {
		if _, err := service.Save(offer.Token, Choice{Target: target}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("a save to %q: got %v, want ErrNotFound", target, err)
		}
	}
	if created, err := service.Save(offer.Token, Choice{Target: site}); err != nil || created {
		t.Fatalf("a save to the offered target = %v, error = %v", created, err)
	}
	updated := read(t, core, site)
	if updated.Password != "new" || !slices.Equal(updated.Apps, []vault.App{{Package: mailPackage, Signer: mailSigner}}) {
		t.Fatalf("after the update = %+v", updated.CredentialInput)
	}
	if other := read(t, core, unrelated); other.Password != "other" || other.Apps != nil {
		t.Fatalf("the credential not chosen = %+v", other.CredentialInput)
	}
}

func TestARefusedSaveKeepsTheCaptureForAnotherTry(t *testing.T) {
	_, service := openVault(t, "")
	offer, err := service.Hold(Capture{Requester: Requester{Origin: "https://example.com"}, Account: "alex", Password: "typed"})
	if err != nil {
		t.Fatal(err)
	}
	if offer.Name != "example.com" {
		t.Fatalf("offer = %+v", offer)
	}
	if _, err := service.Save(offer.Token, Choice{Name: " ", Account: "alex"}); !errors.Is(err, ErrNameRefused) {
		t.Fatalf("a blank name: got %v, want ErrNameRefused", err)
	}
	if created, err := service.Save(offer.Token, Choice{Name: "Example", Account: "alex"}); err != nil || !created {
		t.Fatalf("a save after the refusal = %v, error = %v", created, err)
	}
}

func TestAtMostFourCapturesAreHeldAndEachExpires(t *testing.T) {
	core, service := openVault(t, "")
	var tokens []string
	for i := range 5 {
		offer, err := service.Hold(Capture{Requester: Requester{Origin: "https://example.com"}, Account: fmt.Sprintf("user%d", i), Password: "typed"})
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, offer.Token)
	}
	if _, err := service.Save(tokens[0], Choice{Name: "Example", Account: "user0"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("the oldest capture after a fifth: got %v, want ErrNotFound", err)
	}
	if created, err := service.Save(tokens[4], Choice{Name: "Example", Account: "user4"}); err != nil || !created {
		t.Fatalf("the newest capture = %v, error = %v", created, err)
	}

	service.held.Close()
	synctest.Test(t, func(t *testing.T) {
		service.held = captures.New[vaultservice.Capture](captures.Lifetime)
		offer, err := service.Hold(Capture{Requester: Requester{Origin: "https://example.org"}, Account: "sam", Password: "typed"})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(captures.Lifetime)
		synctest.Wait()
		if held := service.held.Count(system); held != 0 {
			t.Fatalf("%d captures outlived their lifetime", held)
		}
		if _, err := service.Save(offer.Token, Choice{Name: "Example", Account: "sam"}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("a save after the capture expired: got %v, want ErrNotFound", err)
		}
	})
	if entries, err := core.List(); err != nil || len(entries) != 1 {
		t.Fatalf("entries = %d, error = %v", len(entries), err)
	}
}

func TestAPasskeyCreatedOverThePlatformsHashIsOfferedAndSignsOverIt(t *testing.T) {
	core, _ := openVault(t, "")
	work, err := core.CreateGroup("Work")
	if err != nil {
		t.Fatal(err)
	}
	service, err := New(core, recordedGroup(work.String()))
	if err != nil {
		t.Fatal(err)
	}
	creation := newPasskeyCreation()
	creation.Algorithms = []int{-8, authenticator.AlgorithmES256, -257}
	created, err := service.CreatePasskey(creation)
	if err != nil {
		t.Fatal(err)
	}
	if created.ClientData != nil || created.Algorithm != authenticator.AlgorithmES256 || !bytes.HasSuffix(created.AttestationObject, created.AuthenticatorData) {
		t.Fatalf("created = %+v", created)
	}
	if flags := created.AuthenticatorData[32]; flags&0x40 == 0 || flags&0x04 != 0 {
		t.Fatalf("an unverified creation carries flags %08b", flags)
	}
	entries, err := core.List()
	if err != nil || len(entries) != 1 || entries[0].ID.String() != created.ID || !slices.Equal(entries[0].Groups, []vault.ID{work}) {
		t.Fatalf("entries = %+v, error = %v", entries, err)
	}

	choices, err := service.Passkeys(passkeyRPID, nil)
	want := []PasskeyChoice{{ID: created.ID, CredentialID: created.CredentialID, UserHandle: []byte{7, 7}, Account: "alex@example.com", DisplayName: "Alex", Label: "Example"}}
	if err != nil || !reflect.DeepEqual(choices, want) {
		t.Fatalf("passkeys = %+v, error = %v", choices, err)
	}
	if choices, err := service.Passkeys(passkeyRPID, [][]byte{make([]byte, 16)}); err != nil || len(choices) != 0 {
		t.Fatalf("an allow list without the passkey offers %+v, error = %v", choices, err)
	}

	signIn := PasskeySignIn{RPID: passkeyRPID, ClientDataHash: &platformHash, Verified: true, ID: created.ID, CredentialID: created.CredentialID}
	assertion, err := service.SignPasskey(signIn)
	if err != nil {
		t.Fatal(err)
	}
	verifySignature(t, created.PublicKey, assertion.AuthenticatorData, assertion.Signature, platformHash)
	if flags := assertion.AuthenticatorData[32]; flags&0x04 == 0 || assertion.ClientData != nil || !bytes.Equal(assertion.UserHandle, []byte{7, 7}) {
		t.Fatalf("a verified sign-in = %+v", assertion)
	}
	if usage, err := core.Usage(); err != nil || usage[entries[0].ID] == 0 {
		t.Fatalf("usage after the sign-in = %v, error = %v", usage, err)
	}

	elsewhere := signIn
	elsewhere.RPID = "example.org"
	unreadable := signIn
	unreadable.ID = "passkey"
	for name, refused := range map[string]PasskeySignIn{"another relying party": elsewhere, "an unreadable identifier": unreadable} {
		if assertion, err := service.SignPasskey(refused); !errors.Is(err, ErrNotFound) || assertion.Signature != nil {
			t.Fatalf("%s: signed %+v, error = %v", name, assertion, err)
		}
	}
}

func TestAnAppsPasskeyIsSignedOverClientDataTheServiceBuilds(t *testing.T) {
	_, service := openVault(t, "")
	creation := newPasskeyCreation()
	creation.ClientDataHash, creation.Origin, creation.Challenge = nil, appOrigin, []byte("a creation challenge")
	created, err := service.CreatePasskey(creation)
	if err != nil {
		t.Fatal(err)
	}
	signIn := PasskeySignIn{RPID: passkeyRPID, Origin: appOrigin, Challenge: []byte("a sign-in challenge"), ID: created.ID, CredentialID: created.CredentialID}
	assertion, err := service.SignPasskey(signIn)
	if err != nil {
		t.Fatal(err)
	}
	for ceremony, answer := range map[string]struct {
		clientData []byte
		challenge  []byte
	}{
		"webauthn.create": {created.ClientData, creation.Challenge},
		"webauthn.get":    {assertion.ClientData, signIn.Challenge},
	} {
		var fields struct {
			Type, Challenge, Origin string
		}
		if err := json.Unmarshal(answer.clientData, &fields); err != nil {
			t.Fatal(err)
		}
		if fields.Type != ceremony || fields.Origin != appOrigin || fields.Challenge != base64.RawURLEncoding.EncodeToString(answer.challenge) {
			t.Fatalf("client data = %s", answer.clientData)
		}
	}
	verifySignature(t, created.PublicKey, assertion.AuthenticatorData, assertion.Signature, sha256.Sum256(assertion.ClientData))
}

func TestAPasskeyCreationTheVaultCannotServeIsRefusedAndSavesNothing(t *testing.T) {
	core, service := openVault(t, "")
	held, err := service.CreatePasskey(newPasskeyCreation())
	if err != nil {
		t.Fatal(err)
	}
	change := func(edit func(*PasskeyCreation)) PasskeyCreation {
		creation := newPasskeyCreation()
		edit(&creation)
		return creation
	}
	for name, test := range map[string]struct {
		creation PasskeyCreation
		want     error
	}{
		"only other algorithms":          {change(func(c *PasskeyCreation) { c.Algorithms = []int{-8, -257} }), ErrUnsupportedAlgorithm},
		"a relying party no one can use": {change(func(c *PasskeyCreation) { c.RPID = "com" }), ErrInvalidRelyingParty},
		"no client data":                 {change(func(c *PasskeyCreation) { c.ClientDataHash = nil }), ErrInvalidRequest},
		"no user handle":                 {change(func(c *PasskeyCreation) { c.User.Handle = nil }), ErrInvalidRequest},
		"an excluded passkey":            {change(func(c *PasskeyCreation) { c.Exclude = [][]byte{held.CredentialID} }), ErrPasskeyExcluded},
	} {
		if created, err := service.CreatePasskey(test.creation); !errors.Is(err, test.want) || created.CredentialID != nil {
			t.Fatalf("%s: got %+v, %v, want %v", name, created, err, test.want)
		}
	}
	if entries, err := core.List(); err != nil || len(entries) != 1 {
		t.Fatalf("refused creations left %d entries, error = %v", len(entries), err)
	}
}
