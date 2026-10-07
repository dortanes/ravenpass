package linkserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/dortanes/ravenpass/packages/app/captures"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
)

// CaptureOffer answers locked, nothing for alex's known "secret", else the example credential as target.
func (v *fakeVault) CaptureOffer(capture Capture) (linkproto.CaptureOffer, error) {
	if capture.Card != nil {
		return v.cardCaptureOffer(capture)
	}
	if err := v.call(fmt.Sprintf("capture %s %s %s %s", capture.Origin, capture.Account, capture.Password, capture.Current)); err != nil {
		return linkproto.CaptureOffer{}, err
	}
	offer := linkproto.CaptureOffer{Kind: linkproto.CapturePassword, State: linkproto.CaptureLocked, Site: "example.com", Account: capture.Account, Name: "example.com"}
	switch {
	case !v.Unlocked():
	case capture.Account == "alex" && capture.Password == "secret":
		offer.State = linkproto.CaptureNone
	default:
		offer.State = linkproto.CaptureReady
		offer.Targets = []linkproto.SaveTarget{{Credential: exampleID, Label: "Example", Account: "alex", Action: linkproto.SaveUpdate}}
		if capture.Account == "alex" {
			offer.Suggested = exampleID
		}
	}
	return offer, nil
}

// SaveCapture creates a credential for an empty target and updates the target otherwise.
func (v *fakeVault) SaveCapture(capture Capture, choice SaveChoice) (linkproto.Saved, error) {
	saved := capture.Password
	if capture.Card != nil {
		saved = capture.Card.Number
	}
	if err := v.call(fmt.Sprintf("save %s to %q as %s %s", saved, choice.Target, choice.Account, choice.Name)); err != nil {
		return linkproto.Saved{}, err
	}
	if !v.Unlocked() {
		return linkproto.Saved{}, ErrLocked
	}
	if choice.Target == "" {
		return linkproto.Saved{Saved: linkproto.SavedCreated}, nil
	}
	return linkproto.Saved{Saved: linkproto.SavedUpdated}, nil
}

var pendingID = regexp.MustCompile(`^[A-Za-z0-9_-]{22}$`)

// captureSession is one linked extension's open session with the server.
type captureSession struct {
	t         *testing.T
	conn      *websocket.Conn
	transport *peerTransport
	id        string
}

// openCaptureSession links a new extension to server and opens a session for it.
func openCaptureSession(t *testing.T, server *Server) *captureSession {
	t.Helper()
	key := begin(t, server)
	extension := newExtension(t)
	extension.linkWith(t, key)
	recorded, found, err := server.store.Find(extension.key.Public)
	if err != nil || !found {
		t.Fatalf("the linked extension is not recorded: %v", err)
	}
	conn, transport := extension.openSession(t, int(key.Port))
	return &captureSession{t: t, conn: conn, transport: transport, id: recorded.ID}
}

// ask sends one request built from fields and returns the reply.
func (c *captureSession) ask(fields map[string]any) string {
	c.t.Helper()
	fields["id"] = 1
	encoded, err := json.Marshal(fields)
	if err != nil {
		c.t.Fatal(err)
	}
	return request(c.t, c.conn, c.transport, string(encoded))
}

// capture sends a capture and returns the pending id of the offer, which must hold one.
func (c *captureSession) capture(account, password string) string {
	c.t.Helper()
	reply := c.ask(map[string]any{"type": "capture", "origin": exampleOrigin, "account": account, "password": password})
	var answer struct {
		Result linkproto.CaptureOffer `json:"result"`
	}
	if err := json.Unmarshal([]byte(reply), &answer); err != nil || !pendingID.MatchString(answer.Result.Pending) {
		c.t.Fatalf("capture answered %s", reply)
	}
	return answer.Result.Pending
}

func (c *captureSession) review(pending string) string {
	c.t.Helper()
	return c.ask(map[string]any{"type": "review", "pending": pending})
}

func (c *captureSession) save(pending, target, account, name string) string {
	c.t.Helper()
	return c.ask(map[string]any{"type": "save", "pending": pending, "target": target, "account": account, "name": name})
}

func (c *captureSession) discard(pending string) string {
	c.t.Helper()
	return c.ask(map[string]any{"type": "discard", "pending": pending})
}

func readyOffer(pending, account, suggested string) string {
	return `{"id":1,"result":{"kind":"password","state":"ready","pending":"` + pending + `","site":"example.com","account":"` + account +
		`","name":"example.com","targets":[{"credential":"` + exampleID + `","label":"Example","account":"alex","action":"update"}],"suggested":"` + suggested + `"}}`
}

const notFound = `{"id":1,"error":"not-found"}`

func TestACaptureIsOfferedHeldAndForgottenOnceSaved(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)

	pending := session.capture("alex", "typed")
	if calls := vault.takeCalls(); !slices.Equal(calls, []string{"capture https://example.com alex typed "}) {
		t.Fatalf("the capture asked the vault %q", calls)
	}
	if reply := session.review(pending); reply != readyOffer(pending, "alex", exampleID) {
		t.Fatalf("review = %s", reply)
	}
	if reply := session.save(pending, exampleID, "", ""); reply != `{"id":1,"result":{"saved":"updated"}}` {
		t.Fatalf("save = %s", reply)
	}
	for _, reply := range []string{session.review(pending), session.save(pending, exampleID, "", "")} {
		if reply != notFound {
			t.Fatalf("a saved capture answered %s", reply)
		}
	}
	want := []string{"capture https://example.com alex typed ", `save typed to "` + exampleID + `" as  `}
	if calls := vault.takeCalls(); !slices.Equal(calls, want) {
		t.Fatalf("the vault was asked %q", calls)
	}

	created := session.capture("sam", "typed")
	if reply := session.save(created, "", "sam@example.com", "Example"); reply != `{"id":1,"result":{"saved":"created"}}` {
		t.Fatalf("save as new = %s", reply)
	}
	if calls := vault.takeCalls(); !slices.Equal(calls, []string{"capture https://example.com sam typed ", `save typed to "" as sam@example.com Example`}) {
		t.Fatalf("the vault was asked %q", calls)
	}
}

func TestACaptureRequestCarriesTheCurrentPassword(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	session.ask(map[string]any{"type": "capture", "origin": exampleOrigin, "account": "alex", "password": "new", "current": "old"})
	if calls := vault.takeCalls(); !slices.Equal(calls, []string{"capture https://example.com alex new old"}) {
		t.Fatalf("the vault was asked %q", calls)
	}
}

func TestAKnownPasswordIsOfferedNothingAndNotHeld(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	reply := session.ask(map[string]any{"type": "capture", "origin": exampleOrigin, "account": "alex", "password": "secret"})
	if reply != `{"id":1,"result":{"kind":"password","state":"none","site":"example.com","account":"alex","name":"example.com","targets":[],"suggested":""}}` {
		t.Fatalf("a known password answered %s", reply)
	}
	if held := server.held.Count(session.id); held != 0 {
		t.Fatalf("a known password is held %d times", held)
	}
}

func TestALockedVaultKeepsTheCaptureUntilItIsReviewed(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	session := openCaptureSession(t, server)
	reply := session.ask(map[string]any{"type": "capture", "origin": exampleOrigin, "account": "alex", "password": "typed"})
	var answer struct {
		Result linkproto.CaptureOffer `json:"result"`
	}
	if err := json.Unmarshal([]byte(reply), &answer); err != nil {
		t.Fatal(err)
	}
	pending := answer.Result.Pending
	locked := `{"id":1,"result":{"kind":"password","state":"locked","pending":"` + pending + `","site":"example.com","account":"alex","name":"example.com","targets":[],"suggested":""}}`
	if reply != locked || !pendingID.MatchString(pending) {
		t.Fatalf("a capture while locked answered %s", reply)
	}
	if reply := session.review(pending); reply != locked {
		t.Fatalf("a review while locked = %s", reply)
	}
	if reply := session.save(pending, exampleID, "", ""); reply != notFound {
		t.Fatalf("a save to a target not offered while locked = %s", reply)
	}
	if reply := session.save(pending, "", "alex", "Example"); reply != `{"id":1,"error":"locked"}` {
		t.Fatalf("a save while locked = %s", reply)
	}

	vault.unlocked.Store(true)
	if reply := session.review(pending); reply != readyOffer(pending, "alex", exampleID) {
		t.Fatalf("a review after unlocking = %s", reply)
	}
	if reply := session.save(pending, exampleID, "", ""); reply != `{"id":1,"result":{"saved":"updated"}}` {
		t.Fatalf("a save after unlocking = %s", reply)
	}
}

func TestAReviewThatFindsNothingToOfferForgetsTheCapture(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	session := openCaptureSession(t, server)
	reply := session.ask(map[string]any{"type": "capture", "origin": exampleOrigin, "account": "alex", "password": "secret"})
	var answer struct {
		Result linkproto.CaptureOffer `json:"result"`
	}
	if err := json.Unmarshal([]byte(reply), &answer); err != nil || answer.Result.State != linkproto.CaptureLocked {
		t.Fatalf("a capture while locked answered %s", reply)
	}
	vault.unlocked.Store(true)
	if reply := session.review(answer.Result.Pending); reply != `{"id":1,"result":{"kind":"password","state":"none","site":"example.com","account":"alex","name":"example.com","targets":[],"suggested":""}}` {
		t.Fatalf("a review of a known password = %s", reply)
	}
	if reply := session.review(answer.Result.Pending); reply != notFound {
		t.Fatalf("a second review = %s", reply)
	}
}

func TestATargetThatWasNotOfferedIsRefusedBeforeTheVault(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	pending := session.capture("sam", "typed")
	vault.takeCalls()
	for _, target := range []string{"ff000000000000000000000000000000", "Example", exampleID + " "} {
		if reply := session.save(pending, target, "", ""); reply != notFound {
			t.Fatalf("a save to %q = %s", target, reply)
		}
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("targets not offered reached the vault: %q", calls)
	}
	if reply := session.review(pending); reply != readyOffer(pending, "sam", "") {
		t.Fatalf("the refused saves forgot the capture: %s", reply)
	}
}

func TestARefusedSaveKeepsTheCaptureForAnotherTry(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	pending := session.capture("sam", "typed")
	for _, test := range []struct {
		refusal error
		code    string
	}{
		{ErrInvalidAccount, linkproto.ErrorInvalidAccount},
		{fmt.Errorf("save: %w", ErrInvalidName), linkproto.ErrorInvalidName},
		{ErrNotFound, linkproto.ErrorNotFound},
		{ErrLocked, linkproto.ErrorLocked},
	} {
		vault.refuseWith(test.refusal)
		if reply := session.save(pending, "", strings.Repeat("a", 300), ""); reply != `{"id":1,"error":"`+test.code+`"}` {
			t.Fatalf("a save refused with %v = %s", test.refusal, reply)
		}
	}
	vault.refuseWith(nil)
	if reply := session.save(pending, "", "sam", "Example"); reply != `{"id":1,"result":{"saved":"created"}}` {
		t.Fatalf("a save after the refusals = %s", reply)
	}
}

func TestADiscardForgetsTheCapture(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	pending := session.capture("sam", "typed")
	if reply := session.discard(pending); reply != `{"id":1,"result":{}}` {
		t.Fatalf("discard = %s", reply)
	}
	if reply := session.review(pending); reply != notFound {
		t.Fatalf("a review after the discard = %s", reply)
	}
	if reply := session.discard(pending); reply != `{"id":1,"result":{}}` {
		t.Fatalf("a discard of a capture that is gone = %s", reply)
	}
}

func TestAnExpiredCaptureIsNotSaved(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	pending := session.capture("sam", "typed")
	server.held.Forget(session.id, pending)
	vault.takeCalls()
	if reply := session.save(pending, "", "sam", "Example"); reply != notFound {
		t.Fatalf("a save after the capture expired = %s", reply)
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("an expired capture reached the vault: %q", calls)
	}
}

func TestAFifthCapturePushesOutTheOldest(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	var pending []string
	for i := range 5 {
		pending = append(pending, session.capture("sam", fmt.Sprintf("typed %d", i)))
	}
	if reply := session.review(pending[0]); reply != notFound {
		t.Fatalf("the oldest capture after a fifth = %s", reply)
	}
	for _, kept := range pending[1:] {
		if reply := session.review(kept); reply != readyOffer(kept, "sam", "") {
			t.Fatalf("a newer capture = %s", reply)
		}
	}
}

func TestOneExtensionCannotReachAnothersCapture(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	owner := openCaptureSession(t, server)
	other := openCaptureSession(t, server)
	pending := owner.capture("sam", "typed")
	vault.takeCalls()
	if reply := other.review(pending); reply != notFound {
		t.Fatalf("another extension's review = %s", reply)
	}
	if reply := other.save(pending, "", "sam", "Example"); reply != notFound {
		t.Fatalf("another extension's save = %s", reply)
	}
	if reply := other.discard(pending); reply != `{"id":1,"result":{}}` {
		t.Fatalf("another extension's discard = %s", reply)
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("another extension's requests reached the vault: %q", calls)
	}
	if reply := owner.review(pending); reply != readyOffer(pending, "sam", "") {
		t.Fatalf("the owner's review after another extension's requests = %s", reply)
	}
}

func TestUnlinkingForgetsTheExtensionsCaptures(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	unlinked := openCaptureSession(t, server)
	kept := openCaptureSession(t, server)
	unlinked.capture("sam", "typed")
	keptPending := kept.capture("sam", "typed")
	if err := server.Unlink(unlinked.id); err != nil {
		t.Fatal(err)
	}
	if held := server.held.Count(unlinked.id); held != 0 {
		t.Fatalf("an unlinked extension holds %d captures", held)
	}
	if _, held := server.held.Find(kept.id, keptPending); !held {
		t.Fatal("unlinking one extension forgot another's capture")
	}

	kept.capture("alex", "typed")
	if reply := kept.ask(map[string]any{"type": "unlink"}); reply != `{"id":1,"result":{}}` {
		t.Fatalf("unlink = %s", reply)
	}
	if held := server.held.Count(kept.id); held != 0 {
		t.Fatalf("an extension that unlinked itself left %d captures held", held)
	}
}

func TestClosingTheServerForgetsEveryCapture(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	session.capture("sam", "typed")
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if held := server.held.Count(session.id); held != 0 {
		t.Fatalf("a closed server holds %d captures", held)
	}
	if _, err := server.held.Keep(session.id, Capture{Origin: exampleOrigin, Password: "late"}, nil); !errors.Is(err, captures.ErrClosed) {
		t.Fatalf("a capture kept after closing: got %v, want ErrClosed", err)
	}
}

func TestNoCaptureReachesDisk(t *testing.T) {
	path := recordsPath(t)
	server, vault := newServer(t, path)
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	password := "never-on-disk-" + strings.Repeat("7", 16)
	pending := session.capture("sam", password)
	session.review(pending)
	session.capture("sam", password+" again")
	err := filepath.WalkDir(filepath.Dir(path), func(file string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if bytes.Contains(content, []byte(password)) || bytes.Contains(content, []byte(pending)) {
			return fmt.Errorf("%s holds the capture", file)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAnInvalidCaptureOriginIsRefusedBeforeTheVault(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	for _, origin := range []string{"", "example.com", "https://example.com/login", "chrome-extension://cceiadaelnccfbakmhcleifjfilkakag"} {
		if reply := session.ask(map[string]any{"type": "capture", "origin": origin, "account": "sam", "password": "typed"}); reply != `{"id":1,"error":"invalid-origin"}` {
			t.Fatalf("a capture for %q answered %s", origin, reply)
		}
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("invalid origins reached the vault: %q", calls)
	}
}

func TestMalformedCaptureFieldsCloseTheSession(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	key := begin(t, server)
	extension := newExtension(t)
	extension.linkWith(t, key)
	for _, plaintext := range []string{
		`{"id":1,"type":"capture","origin":"https://example.com","account":"sam","password":7}`,
		`{"id":1,"type":"capture","origin":"https://example.com","account":["sam"],"password":"typed"}`,
		`{"id":1,"type":"capture","origin":"https://example.com","password":"typed","current":{}}`,
		`{"id":1,"type":"review","pending":1}`,
		`{"id":1,"type":"save","pending":"p","target":7,"account":"sam","name":"Example"}`,
		`{"id":1,"type":"save","pending":"p","target":"","account":"sam","name":5}`,
		`{"id":1,"type":"discard","pending":["p"]}`,
	} {
		conn, transport := extension.openSession(t, int(key.Port))
		wantStatus(t, closingRequest(t, conn, transport, plaintext), linkproto.CloseMalformed)
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("malformed requests reached the vault: %q", calls)
	}
}
