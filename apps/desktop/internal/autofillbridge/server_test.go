package autofillbridge

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/app/api"
	"github.com/dortanes/ravenpass/packages/app/autofill"
	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/verification"
)

// fakeVault matches each credential to one origin, and holds passkeys by relying party.
type fakeVault struct {
	mu       sync.Mutex
	open     bool
	offers   map[string][]autofill.Suggestion
	found    map[string][]autofill.Suggestion
	logins   map[string]autofill.Login
	codes    map[string]string
	origins  map[string]string
	asked    []autofill.Requester
	coded    []bool
	added    []string
	passkeys map[string][]autofill.PasskeyChoice
	signed   []autofill.PasskeySignIn
	created  []autofill.PasskeyCreation
}

func (v *fakeVault) Open() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.open
}

func (v *fakeVault) setOpen(open bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.open = open
}

func (v *fakeVault) Suggest(r autofill.Requester, code bool) ([]autofill.Suggestion, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.open {
		return nil, autofill.ErrLocked
	}
	v.asked = append(v.asked, r)
	v.coded = append(v.coded, code)
	return v.offers[r.Origin], nil
}

func (v *fakeVault) Fill(id string, r autofill.Requester) (autofill.Login, error) {
	if err := v.check(id, r); err != nil {
		return autofill.Login{}, err
	}
	return v.logins[id], nil
}

func (v *fakeVault) OneTimeCode(id string, r autofill.Requester) (autofill.Code, error) {
	if err := v.check(id, r); err != nil {
		return autofill.Code{}, err
	}
	code, ok := v.codes[id]
	if !ok {
		return autofill.Code{}, autofill.ErrNoCode
	}
	return autofill.Code{Code: code, ExpiresAt: 1}, nil
}

func (v *fakeVault) check(id string, r autofill.Requester) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.asked = append(v.asked, r)
	origin, ok := v.origins[id]
	switch {
	case !v.open:
		return autofill.ErrLocked
	case !ok:
		return autofill.ErrNotFound
	case origin != r.Origin:
		return autofill.ErrNoMatch
	default:
		return nil
	}
}

func (v *fakeVault) Search(query string, _ bool) ([]autofill.Suggestion, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.open {
		return nil, autofill.ErrLocked
	}
	return v.found[query], nil
}

// AddSite makes origin the one the credential matches.
func (v *fakeVault) AddSite(id string, r autofill.Requester) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	switch _, ok := v.origins[id]; {
	case r.Origin == "":
		return autofill.ErrInvalidRequest
	case !v.open:
		return autofill.ErrLocked
	case !ok:
		return autofill.ErrNotFound
	}
	v.origins[id] = r.Origin
	v.added = append(v.added, r.Origin)
	return nil
}

func (v *fakeVault) additions() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.added)
}

func (v *fakeVault) requesters() []autofill.Requester {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.asked)
}

// fakePage holds website icons by site, the interface language and the appearance.
type fakePage struct {
	icons      map[string]api.SiteIcon
	language   api.LanguageSettings
	appearance api.Appearance
}

func (p fakePage) SiteIcon(site string) (api.SiteIcon, error) {
	if site == "" {
		return api.SiteIcon{}, errors.New("no site")
	}
	return p.icons[site], nil
}

func (p fakePage) GetLanguage() (api.LanguageSettings, error) {
	return p.language, nil
}

func (p fakePage) GetAppearance() (api.Appearance, error) {
	return p.appearance, nil
}

type fakeSystem struct {
	signed    bool
	admits    bool
	container string
	announced *atomic.Int32
}

func (s fakeSystem) Signed() bool { return s.signed }

func (s fakeSystem) Admits(net.Conn) bool { return s.admits }

func (s fakeSystem) Container() (string, error) {
	if s.container == "" {
		return "", errors.New("no container")
	}
	return s.container, nil
}

func (s fakeSystem) AnnounceListening() {
	if s.announced != nil {
		s.announced.Add(1)
	}
}

type noPIN struct{}

func (noPIN) VerifyPIN(string) error { return errors.New("no PIN") }

func newQueue(t *testing.T) *confirmation.Queue {
	t.Helper()
	queue, err := confirmation.New(noPIN{})
	if err != nil {
		t.Fatal(err)
	}
	return queue
}

// container is a folder whose path is short enough for a socket inside it.
func container(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "bridge")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// serve starts a server whose owner cannot be verified in a container of its own and returns its socket path.
func serve(t *testing.T, vault Vault, unlocks Unlocks, system fakeSystem) string {
	t.Helper()
	return serveVerifying(t, vault, unlocks, &fakeVerifier{err: verification.ErrUnverifiable}, system)
}

// serveVerifying starts a server with an empty page in a container of its own and returns its socket path.
func serveVerifying(t *testing.T, vault Vault, unlocks Unlocks, verifier Verifier, system fakeSystem) string {
	t.Helper()
	return servePage(t, vault, fakePage{}, unlocks, verifier, system)
}

// servePage starts a server in a container of its own, and returns the path of its socket.
func servePage(t *testing.T, vault Vault, page Page, unlocks Unlocks, verifier Verifier, system fakeSystem) string {
	t.Helper()
	system.container = container(t)
	server, err := New(vault, page, unlocks, verifier, system)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return filepath.Join(system.container, socketName)
}

func ask(t *testing.T, path string, asked request) answer {
	t.Helper()
	reply, _, err := exchangeNoticed(path, asked)
	if err != nil {
		t.Fatal(err)
	}
	return reply
}

func exchange(path string, asked request) (answer, error) {
	reply, _, err := exchangeNoticed(path, asked)
	return reply, err
}

// exchangeNoticed returns the answer and the wait notice that preceded it, nil for none.
func exchangeNoticed(path string, asked request) (answer, *waitNotice, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return answer{}, nil, err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return answer{}, nil, err
	}
	if err := writeMessage(conn, maxRequestBytes, asked); err != nil {
		return answer{}, nil, err
	}
	var first json.RawMessage
	if err := readMessage(conn, maxAnswerBytes, &first); err != nil {
		return answer{}, nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(first, &fields); err != nil {
		return answer{}, nil, err
	}
	var notice *waitNotice
	if _, ok := fields["wait"]; ok {
		notice = &waitNotice{}
		if err := json.Unmarshal(first, notice); err != nil {
			return answer{}, nil, err
		}
		if err := readMessage(conn, maxAnswerBytes, &first); err != nil {
			return answer{}, notice, err
		}
	}
	var reply answer
	err = json.Unmarshal(first, &reply)
	return reply, notice, err
}

var admitted = fakeSystem{signed: true, admits: true}

func TestAPeerThatIsNotTheExtensionIsClosedWithoutAnAnswer(t *testing.T) {
	vault := &fakeVault{open: true}
	path := serve(t, vault, newQueue(t), fakeSystem{signed: true})
	// The server may close before the request is written, which then fails instead.
	if reply, err := exchange(path, request{Op: opStatus}); err == nil {
		t.Fatalf("got %+v, want the connection closed", reply)
	}
}

// lookingSystem notes whether the server looked for the container.
type lookingSystem struct {
	fakeSystem
	looked *bool
}

func (s lookingSystem) Container() (string, error) {
	*s.looked = true
	return s.fakeSystem.Container()
}

func TestABuildTheExtensionDoesNotAdmitTouchesNothing(t *testing.T) {
	looked := false
	server, err := New(&fakeVault{}, fakePage{}, newQueue(t), &fakeVerifier{}, lookingSystem{fakeSystem{admits: true, container: container(t)}, &looked})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); !errors.Is(err, ErrUnsigned) {
		t.Fatalf("got %v, want ErrUnsigned", err)
	}
	if looked {
		t.Fatal("an unsigned build looked for the App Group container")
	}
}

func TestWithoutAContainerTheServerDoesNotListen(t *testing.T) {
	server, err := New(&fakeVault{}, fakePage{}, newQueue(t), &fakeVerifier{}, admitted)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err == nil {
		_ = server.Close()
		t.Fatal("a server without a container listens")
	}
}

func TestAStaleSocketIsReplaced(t *testing.T) {
	system := admitted
	system.container = container(t)
	path := filepath.Join(system.container, socketName)
	stale, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	stale.(*net.UnixListener).SetUnlinkOnClose(false)
	if err := stale.Close(); err != nil {
		t.Fatal(err)
	}
	server, err := New(&fakeVault{open: true}, fakePage{}, newQueue(t), &fakeVerifier{}, system)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	if reply := ask(t, path, request{Op: opStatus}); !reply.Open {
		t.Fatalf("status = %+v, want open", reply)
	}
}

func TestASocketAnotherProcessAnswersAtIsLeftAlone(t *testing.T) {
	path := serve(t, &fakeVault{}, newQueue(t), admitted)
	system := admitted
	system.container = filepath.Dir(path)
	second, err := New(&fakeVault{}, fakePage{}, newQueue(t), &fakeVerifier{}, system)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Start(); !errors.Is(err, ErrInUse) {
		t.Fatalf("got %v, want ErrInUse", err)
	}
}

func TestCloseRemovesTheSocket(t *testing.T) {
	system := admitted
	system.container = container(t)
	server, err := New(&fakeVault{}, fakePage{}, newQueue(t), &fakeVerifier{}, system)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(system.container, socketName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket after close: %v", err)
	}
}

func TestStatusReportsWhetherTheVaultIsOpen(t *testing.T) {
	vault := &fakeVault{}
	path := serve(t, vault, newQueue(t), admitted)
	if reply := ask(t, path, request{Op: opStatus}); reply.Open || reply.Error != "" {
		t.Fatalf("locked status = %+v", reply)
	}
	vault.setOpen(true)
	if reply := ask(t, path, request{Op: opStatus}); !reply.Open {
		t.Fatalf("open status = %+v", reply)
	}
}

func TestAnEmptySearchListsWhatMatchesEachServiceIdentifierMostSpecificFirst(t *testing.T) {
	login := autofill.Suggestion{ID: "a", Label: "Mail", Account: "alex", Site: "mail.example.com", Exact: true, Tags: []string{"Work"}}
	other := autofill.Suggestion{ID: "b", Label: "Example", Account: "sam", Site: "example.com"}
	vault := &fakeVault{open: true, offers: map[string][]autofill.Suggestion{
		"https://mail.example.com": {login, other},
		"https://example.com":      {other},
	}}
	path := serve(t, vault, newQueue(t), admitted)
	reply := ask(t, path, request{Op: opSearch, Codes: true, Services: []service{
		{Kind: serviceURL, Value: "https://mail.example.com/inbox?id=1"},
		{Kind: serviceDomain, Value: "example.com"},
		{Kind: serviceDomain, Value: "mail.example.com"},
		{Kind: "app", Value: "com.example.mail"},
	}})
	want := answer{Site: "mail.example.com", Scope: autofill.ScopeMatches, Suggestions: []suggestion{
		{ID: "a", Label: "Mail", Account: "alex", Site: "mail.example.com", Matches: true, Tags: []string{"Work"}},
		{ID: "b", Label: "Example", Account: "sam", Site: "example.com", Matches: true},
	}}
	if !reflect.DeepEqual(reply, want) {
		t.Fatalf("search = %+v, want %+v", reply, want)
	}
	origins := []autofill.Requester{{Origin: "https://mail.example.com"}, {Origin: "https://example.com"}}
	if asked := vault.requesters(); !slices.EqualFunc(asked, slices.Concat(origins, origins), sameOrigin) {
		t.Fatalf("asked %+v, want %+v for sign-in and then for codes", asked, origins)
	}
	if !slices.Equal(vault.coded, []bool{false, false, true, true}) {
		t.Fatalf("code suggestions asked %v", vault.coded)
	}
}

func sameOrigin(a, b autofill.Requester) bool { return a.Origin == b.Origin }

func TestASearchNamesThePagesSiteInItsASCIIForm(t *testing.T) {
	vault := &fakeVault{open: true}
	path := serve(t, vault, newQueue(t), admitted)
	reply := ask(t, path, request{Op: opSearch, Services: []service{
		{Kind: serviceURL, Value: "ftp://files.example.org/"},
		{Kind: serviceURL, Value: "https://www.xn--mller-kva.example/login"},
		{Kind: serviceDomain, Value: "xn--mller-kva.example"},
	}})
	if reply.Error != "" || reply.Site != "xn--mller-kva.example" || reply.Scope != autofill.ScopeVault {
		t.Fatalf("search = %+v, want the vault listed for xn--mller-kva.example", reply)
	}
	if reply := ask(t, path, request{Op: opSearch, Query: "mail"}); reply.Error != "" || reply.Site != "" {
		t.Fatalf("search without a service = %+v, want no site", reply)
	}
}

func TestSearchWhileLockedIsRefused(t *testing.T) {
	path := serve(t, &fakeVault{}, newQueue(t), admitted)
	reply := ask(t, path, request{Op: opSearch, Services: []service{{Kind: serviceDomain, Value: "example.com"}}})
	if !reflect.DeepEqual(reply, answer{Error: refusedLocked}) {
		t.Fatalf("locked search = %+v", reply)
	}
}

func TestTooManyServiceIdentifiersAreRefused(t *testing.T) {
	path := serve(t, &fakeVault{open: true}, newQueue(t), admitted)
	services := make([]service, maxServices+1)
	for i := range services {
		services[i] = service{Kind: serviceDomain, Value: "example.com"}
	}
	for _, op := range []string{opSearch, opPassword, opCode, opAddSite} {
		if reply := ask(t, path, request{Op: op, ID: "a", Services: services}); reply.Error != refusedInvalid {
			t.Fatalf("%s: got %+v, want invalid", op, reply)
		}
	}
}

func TestPasswordReleasesTheLoginOrElseTheEmailAndThePasswordOnly(t *testing.T) {
	vault := &fakeVault{
		open: true,
		logins: map[string]autofill.Login{
			"a": {Login: "alex", Email: "alex@example.com", Password: "first"},
			"b": {Email: "sam@example.com", Password: "second"},
		},
		origins: map[string]string{"a": "https://example.com", "b": "https://example.com"},
	}
	path := serve(t, vault, newQueue(t), admitted)
	services := []service{{Kind: serviceDomain, Value: "example.com"}}
	if reply := ask(t, path, request{Op: opPassword, ID: "a", Services: services}); !reflect.DeepEqual(reply, answer{User: "alex", Password: "first"}) {
		t.Fatalf("login = %+v", reply)
	}
	if reply := ask(t, path, request{Op: opPassword, ID: "b", Services: services}); !reflect.DeepEqual(reply, answer{User: "sam@example.com", Password: "second"}) {
		t.Fatalf("email login = %+v", reply)
	}
}

func TestPasswordFallsBackToALessSpecificServiceIdentifier(t *testing.T) {
	vault := &fakeVault{
		open:    true,
		logins:  map[string]autofill.Login{"a": {Login: "alex", Password: "secret"}},
		origins: map[string]string{"a": "https://example.com"},
	}
	path := serve(t, vault, newQueue(t), admitted)
	reply := ask(t, path, request{Op: opPassword, ID: "a", Services: []service{
		{Kind: serviceURL, Value: "https://login.example.com/"},
		{Kind: serviceDomain, Value: "example.com"},
	}})
	if !reflect.DeepEqual(reply, answer{User: "alex", Password: "secret"}) {
		t.Fatalf("password = %+v", reply)
	}
}

func TestPasswordRefusals(t *testing.T) {
	vault := &fakeVault{
		logins:  map[string]autofill.Login{"a": {Login: "alex", Password: "secret"}},
		origins: map[string]string{"a": "https://example.com"},
	}
	path := serve(t, vault, newQueue(t), admitted)
	site := []service{{Kind: serviceDomain, Value: "example.com"}}
	if reply := ask(t, path, request{Op: opPassword, ID: "a", Services: site}); !reflect.DeepEqual(reply, answer{Error: refusedLocked}) {
		t.Fatalf("locked = %+v", reply)
	}
	vault.setOpen(true)
	cases := []struct {
		name     string
		id       string
		services []service
		want     answer
	}{
		{"unknown credential", "z", site, answer{Error: refusedNotFound}},
		{"another site", "a", []service{{Kind: serviceDomain, Value: "example.org"}}, answer{Error: refusedNoMatch}},
		{"no web origin", "a", []service{{Kind: serviceURL, Value: "ftp://example.com"}}, answer{Error: refusedNoMatch}},
		{"no service identifier", "a", nil, answer{Error: refusedNoMatch}},
	}
	for _, c := range cases {
		if reply := ask(t, path, request{Op: opPassword, ID: c.id, Services: c.services}); !reflect.DeepEqual(reply, c.want) {
			t.Errorf("%s: got %+v, want %+v", c.name, reply, c.want)
		}
	}
}

func TestAddSiteAddsTheMostSpecificWebsiteSoTheCredentialFills(t *testing.T) {
	vault := &fakeVault{
		open:    true,
		logins:  map[string]autofill.Login{"a": {Login: "alex", Password: "secret"}},
		origins: map[string]string{"a": "https://example.com"},
	}
	path := serve(t, vault, newQueue(t), admitted)
	services := []service{
		{Kind: serviceURL, Value: "https://login.example.org/session"},
		{Kind: serviceDomain, Value: "example.org"},
	}
	if reply := ask(t, path, request{Op: opAddSite, ID: "a", Services: services}); !reflect.DeepEqual(reply, answer{}) {
		t.Fatalf("add-site = %+v", reply)
	}
	if added := vault.additions(); !slices.Equal(added, []string{"https://login.example.org"}) {
		t.Fatalf("added %q", added)
	}
	if reply := ask(t, path, request{Op: opPassword, ID: "a", Services: services}); !reflect.DeepEqual(reply, answer{User: "alex", Password: "secret"}) {
		t.Fatalf("password after adding the site = %+v", reply)
	}
}

func TestAddSiteRefusals(t *testing.T) {
	vault := &fakeVault{origins: map[string]string{"a": "https://example.com"}}
	path := serve(t, vault, newQueue(t), admitted)
	site := []service{{Kind: serviceDomain, Value: "example.org"}}
	if reply := ask(t, path, request{Op: opAddSite, ID: "a", Services: site}); !reflect.DeepEqual(reply, answer{Error: refusedLocked}) {
		t.Fatalf("locked = %+v", reply)
	}
	vault.setOpen(true)
	cases := []struct {
		name     string
		id       string
		services []service
		want     string
	}{
		{"unknown credential", "z", site, refusedNotFound},
		{"no web origin", "a", []service{{Kind: serviceURL, Value: "ftp://example.org"}}, refusedInvalid},
		{"no service identifier", "a", nil, refusedInvalid},
	}
	for _, c := range cases {
		if reply := ask(t, path, request{Op: opAddSite, ID: c.id, Services: c.services}); !reflect.DeepEqual(reply, answer{Error: c.want}) {
			t.Errorf("%s: got %+v, want %s", c.name, reply, c.want)
		}
	}
	if added := vault.additions(); len(added) != 0 {
		t.Fatalf("added %q", added)
	}
}

func TestCodeReleasesTheCodeOnly(t *testing.T) {
	vault := &fakeVault{
		open:    true,
		logins:  map[string]autofill.Login{"a": {Login: "alex", Password: "secret"}, "b": {Login: "sam"}},
		codes:   map[string]string{"a": "123456"},
		origins: map[string]string{"a": "https://example.com", "b": "https://example.com"},
	}
	path := serve(t, vault, newQueue(t), admitted)
	site := []service{{Kind: serviceURL, Value: "https://example.com/verify"}}
	if reply := ask(t, path, request{Op: opCode, ID: "a", Services: site}); !reflect.DeepEqual(reply, answer{Code: "123456"}) {
		t.Fatalf("code = %+v", reply)
	}
	if reply := ask(t, path, request{Op: opCode, ID: "b", Services: site}); !reflect.DeepEqual(reply, answer{Error: refusedNoCode}) {
		t.Fatalf("no code = %+v", reply)
	}
}

func TestASearchListsWhatTheVaultFindsMarkingWhatAlreadyMatches(t *testing.T) {
	mail := autofill.Suggestion{ID: "a", Label: "Mail", Account: "alex", Site: "mail.example.com"}
	bank := autofill.Suggestion{ID: "b", Label: "Bank", Account: "alex", Site: "bank.example"}
	vault := &fakeVault{
		open:   true,
		offers: map[string][]autofill.Suggestion{"https://mail.example.com": {mail}},
		found:  map[string][]autofill.Suggestion{"a": {bank, mail}},
	}
	path := serve(t, vault, newQueue(t), admitted)
	reply := ask(t, path, request{Op: opSearch, Query: "a", Services: []service{{Kind: serviceDomain, Value: "mail.example.com"}}})
	want := answer{Site: "mail.example.com", Scope: autofill.ScopeVault, Suggestions: []suggestion{
		{ID: "b", Label: "Bank", Account: "alex", Site: "bank.example"},
		{ID: "a", Label: "Mail", Account: "alex", Site: "mail.example.com", Matches: true},
	}}
	if !reflect.DeepEqual(reply, want) {
		t.Fatalf("search = %+v, want %+v", reply, want)
	}
}

func TestIconAnswersTheSitesIconAsTheAppsWindowReadsIt(t *testing.T) {
	page := fakePage{icons: map[string]api.SiteIcon{"example.com": {Image: "iVBOR", Tint: "#336699"}}}
	path := servePage(t, &fakeVault{}, page, newQueue(t), &fakeVerifier{}, admitted)
	if reply := ask(t, path, request{Op: opIcon, Site: "example.com"}); !reflect.DeepEqual(reply, answer{Image: "iVBOR", Tint: "#336699"}) {
		t.Fatalf("icon = %+v", reply)
	}
	if reply := ask(t, path, request{Op: opIcon, Site: "example.org"}); !reflect.DeepEqual(reply, answer{}) {
		t.Fatalf("no icon = %+v", reply)
	}
	if reply := ask(t, path, request{Op: opIcon}); !reflect.DeepEqual(reply, answer{Error: refusedFailed}) {
		t.Fatalf("no site = %+v", reply)
	}
}

func TestLanguageAnswersTheInterfaceLanguage(t *testing.T) {
	page := fakePage{language: api.LanguageSettings{Languages: []string{"en", "ru"}, Language: "ru", Chosen: true}}
	path := servePage(t, &fakeVault{}, page, newQueue(t), &fakeVerifier{}, admitted)
	want := answer{Languages: []string{"en", "ru"}, Language: "ru", Chosen: true}
	if reply := ask(t, path, request{Op: opLanguage}); !reflect.DeepEqual(reply, want) {
		t.Fatalf("language = %+v, want %+v", reply, want)
	}
}

func TestAppearanceAnswersTheChosenAppearance(t *testing.T) {
	page := fakePage{appearance: api.Appearance{Appearance: "dark", Offered: []string{"system", "light", "dark"}}}
	path := servePage(t, &fakeVault{}, page, newQueue(t), &fakeVerifier{}, admitted)
	if reply := ask(t, path, request{Op: opAppearance}); !reflect.DeepEqual(reply, answer{Appearance: "dark"}) {
		t.Fatalf("appearance = %+v", reply)
	}
}

func TestAnUnknownOperationIsRefused(t *testing.T) {
	path := serve(t, &fakeVault{open: true}, newQueue(t), admitted)
	if reply := ask(t, path, request{Op: "export"}); !reflect.DeepEqual(reply, answer{Error: refusedInvalid}) {
		t.Fatalf("got %+v, want invalid", reply)
	}
}

func TestUnlockAnswersAtOnceWhenTheVaultIsOpen(t *testing.T) {
	queue := newQueue(t)
	path := serve(t, &fakeVault{open: true}, queue, admitted)
	if reply := ask(t, path, request{Op: opUnlock}); !reply.Open {
		t.Fatalf("unlock = %+v", reply)
	}
	if request, _ := queue.Next(context.Background(), "none"); request.ID != "" {
		t.Fatalf("an open vault posted %+v", request)
	}
}

func TestUnlockWaitsUntilTheOwnerOpensTheVault(t *testing.T) {
	queue := newQueue(t)
	vault := &fakeVault{}
	path := serve(t, vault, queue, admitted)
	go func() {
		posted, err := queue.Next(context.Background(), "")
		if err != nil || posted.Kind != confirmation.KindUnlock {
			return
		}
		vault.setOpen(true)
		queue.VaultOpened()
	}()
	if reply := ask(t, path, request{Op: opUnlock}); !reply.Open {
		t.Fatalf("unlock = %+v, want open", reply)
	}
}

func TestUnlockAnswersOnceTheDeviceOpensTheVaultUnseen(t *testing.T) {
	queue := newQueue(t)
	vault := &fakeVault{}
	queue.UnlockOnDevice(func() bool {
		vault.setOpen(true)
		queue.VaultOpened()
		return true
	})
	path := serve(t, vault, queue, admitted)
	answered := make(chan answer, 1)
	go func() { answered <- ask(t, path, request{Op: opUnlock}) }()
	select {
	case reply := <-answered:
		if !reply.Open {
			t.Fatalf("unlock = %+v, want open", reply)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the unlock kept waiting after the device opened the vault")
	}
}

func TestUnlockReportsARequestTheOwnerDeclined(t *testing.T) {
	queue := newQueue(t)
	path := serve(t, &fakeVault{}, queue, admitted)
	go func() {
		posted, err := queue.Next(context.Background(), "")
		if err == nil {
			_ = queue.Decline(posted.ID)
		}
	}()
	if reply := ask(t, path, request{Op: opUnlock}); reply.Open || reply.Error != "" {
		t.Fatalf("declined unlock = %+v", reply)
	}
}

func TestAnUnlockEndsWhenTheExtensionHangsUp(t *testing.T) {
	queue := newQueue(t)
	system := admitted
	system.container = container(t)
	server, err := New(&fakeVault{}, fakePage{}, queue, &fakeVerifier{}, system)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("unix", filepath.Join(system.container, socketName))
	if err != nil {
		t.Fatal(err)
	}
	if err := writeMessage(conn, maxRequestBytes, request{Op: opUnlock}); err != nil {
		t.Fatal(err)
	}
	if posted, err := queue.Next(context.Background(), ""); err != nil || posted.Kind != confirmation.KindUnlock || posted.Requester != confirmation.RequesterAutofill {
		t.Fatalf("posted %+v, %v", posted, err)
	}
	t.Cleanup(func() { _ = server.Close() })
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(server.slots) > 0 {
		if time.Now().After(deadline) {
			t.Fatal("the unlock still holds its connection slot 5 s after the extension hung up")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAnAnswerThatWaitsForTheOwnerIsPrecededByHowLongItMayTake(t *testing.T) {
	queue := newQueue(t)
	vault := passkeyVault()
	path := serveVerifying(t, vault, queue, &fakeVerifier{}, admitted)
	cases := []struct {
		name  string
		asked request
		want  *waitNotice
	}{
		{"status", request{Op: opStatus}, nil},
		{"unlock", request{Op: opUnlock}, &waitNotice{Wait: int64(unlockWait / time.Second)}},
		{"a sign-in that asks the owner", signIn(autofill.VerificationRequired), &waitNotice{Wait: int64(verification.Timeout / time.Second)}},
		{"a sign-in that does not", signIn(autofill.VerificationDiscouraged), nil},
		{"a creation that asks the owner", creation(autofill.VerificationPreferred), &waitNotice{Wait: int64(verification.Timeout / time.Second)}},
	}
	for _, c := range cases {
		reply, notice, err := exchangeNoticed(path, c.asked)
		if err != nil || reply.Error != "" {
			t.Fatalf("%s: %+v, %v", c.name, reply, err)
		}
		if !reflect.DeepEqual(notice, c.want) {
			t.Errorf("%s: notice %+v, want %+v", c.name, notice, c.want)
		}
	}
}

func TestStartTellsTheExtensionTheSocketListens(t *testing.T) {
	var announced atomic.Int32
	system := admitted
	system.announced = &announced
	serve(t, &fakeVault{}, newQueue(t), system)
	if announced.Load() != 1 {
		t.Fatalf("announced %d times, want once", announced.Load())
	}
}

func TestOriginOfAServiceIdentifier(t *testing.T) {
	cases := []struct {
		named service
		want  string
	}{
		{service{serviceURL, "https://Mail.Example.com:8443/inbox?q=1#top"}, "https://Mail.Example.com:8443"},
		{service{serviceURL, "http://alex@example.com/"}, ""},
		{service{serviceURL, "http://example.com/"}, "http://example.com"},
		{service{serviceURL, "ftp://example.com/"}, ""},
		{service{serviceURL, "example.com"}, ""},
		{service{serviceDomain, "example.com"}, "https://example.com"},
		{service{serviceDomain, "example.com:443"}, ""},
		{service{serviceDomain, "alex@example.com"}, ""},
		{service{serviceDomain, "example.com/login"}, ""},
		{service{serviceDomain, ""}, ""},
		{service{"app", "com.example.mail"}, ""},
	}
	for _, c := range cases {
		if got := originOf(c.named); got != c.want {
			t.Errorf("originOf(%+v) = %q, want %q", c.named, got, c.want)
		}
	}
}
