package linkserver

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/dortanes/ravenpass/packages/app/linkkey"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkstore"
)

const (
	extensionOrigin = "chrome-extension://cceiadaelnccfbakmhcleifjfilkakag"
	extensionName   = "Chrome · macOS"
	testWait        = 5 * time.Second
)

const (
	exampleOrigin = "https://example.com"
	exampleID     = "0102030405060708090a0b0c0d0e0f10"
)

// fakeVault holds one credential for exampleOrigin, records calls, and answers refuse when set.
type fakeVault struct {
	unlocked atomic.Bool
	unlocks  atomic.Int32

	mu         sync.Mutex
	refuse     error
	calls      []string
	fillers    []string
	identities []linkproto.Identity
	sharing    func(ctx context.Context, asked func(linkproto.Progress)) (linkproto.SharedFile, []byte, error)
	verifying  func(ctx context.Context, asked func(linkproto.Progress)) error
}

func (v *fakeVault) Unlocked() bool { return v.unlocked.Load() }

func (v *fakeVault) Suggest(origin string, purpose linkproto.Purpose) ([]linkproto.Suggestion, error) {
	if err := v.call("suggest " + string(purpose) + " " + origin); err != nil {
		return nil, err
	}
	if origin != exampleOrigin {
		return nil, nil
	}
	suggestion := linkproto.Suggestion{ID: exampleID, Label: "Example", Account: "alex", Site: "example.com", Exact: true}
	if purpose == linkproto.PurposeCode {
		suggestion.Digits, suggestion.Period = 6, 30
	}
	return []linkproto.Suggestion{suggestion}, nil
}

func (v *fakeVault) Fill(ctx context.Context, extension, credential, origin string, asked func(linkproto.Progress)) (linkproto.Fill, error) {
	if err := v.call("fill " + credential + " " + origin); err != nil {
		return linkproto.Fill{}, err
	}
	v.mu.Lock()
	v.fillers = append(v.fillers, extension)
	v.mu.Unlock()
	if err := v.verify(ctx, asked); err != nil {
		return linkproto.Fill{}, err
	}
	return linkproto.Fill{Login: "alex", Email: "alex@example.com", Password: "secret"}, nil
}

func (v *fakeVault) OneTimeCode(ctx context.Context, credential, origin string, asked func(linkproto.Progress)) (linkproto.OneTimeCode, error) {
	if err := v.call("code " + credential + " " + origin); err != nil {
		return linkproto.OneTimeCode{}, err
	}
	if err := v.verify(ctx, asked); err != nil {
		return linkproto.OneTimeCode{}, err
	}
	return linkproto.OneTimeCode{Code: "287082", Digits: 6, Period: 30, ExpiresAt: 1790000010000}, nil
}

func (v *fakeVault) AddWebsite(credential, origin string) error {
	return v.call("add " + credential + " " + origin)
}

func (v *fakeVault) SiteIcon(site string) (linkproto.Icon, error) {
	if err := v.call("icon " + site); err != nil {
		return linkproto.Icon{}, err
	}
	return linkproto.Icon{Image: "iVBORw0KGgo=", Tint: "#1a2b3c"}, nil
}

func (v *fakeVault) Identities() ([]linkproto.Identity, error) {
	if err := v.call("identities"); err != nil {
		return nil, err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.identities, nil
}

func (v *fakeVault) Share(ctx context.Context, identity, file, origin string, asked func(linkproto.Progress)) (linkproto.SharedFile, []byte, error) {
	if err := v.call("share " + identity + " " + file + " " + origin); err != nil {
		return linkproto.SharedFile{}, nil, err
	}
	v.mu.Lock()
	sharing := v.sharing
	v.mu.Unlock()
	if sharing == nil {
		return exampleShared, bytes.Clone(exampleFile), nil
	}
	return sharing(ctx, asked)
}

func (v *fakeVault) shareWith(sharing func(ctx context.Context, asked func(linkproto.Progress)) (linkproto.SharedFile, []byte, error)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.sharing = sharing
}

func (v *fakeVault) ShowUnlock() { v.unlocks.Add(1) }

func (v *fakeVault) call(call string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.calls = append(v.calls, call)
	return v.refuse
}

func (v *fakeVault) refuseWith(err error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.refuse = err
}

// takeCalls returns the calls recorded so far and forgets them.
func (v *fakeVault) takeCalls() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	calls := v.calls
	v.calls = nil
	return calls
}

// fakeSettings reports the language and sign-in style it holds.
type fakeSettings struct {
	mu       sync.Mutex
	language string
	signIn   string
}

func (s *fakeSettings) Language() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.language
}

func (s *fakeSettings) SignInStyle() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.signIn
}

func (s *fakeSettings) change(language, signIn string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.language, s.signIn = language, signIn
}

type keptInBackups struct{}

func (keptInBackups) Exclude(string) error { return nil }

// newServer opens a server over the records at path, in English; adjust runs before first use.
func newServer(t *testing.T, path string, adjust ...func(*Server)) (*Server, *fakeVault) {
	t.Helper()
	store, err := linkstore.New(path, keptInBackups{})
	if err != nil {
		t.Fatal(err)
	}
	vault := &fakeVault{}
	server, err := New(store, vault, &fakeSettings{language: "en", signIn: "card"})
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range adjust {
		change(server)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server, vault
}

func recordsPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "extensions.json")
}

// listeningPort reports the port the server listens on, zero while it does not listen.
func (s *Server) listeningPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listening == nil {
		return 0
	}
	return s.listening.port
}

// offeredKey is the port and secret the server waits on, which offer must carry as its text.
func (s *Server) offeredKey(t *testing.T, offer Offer) linkkey.Key {
	t.Helper()
	s.mu.Lock()
	key := linkkey.Key{Port: uint16(s.listening.port), Secret: s.offer.secret}
	s.mu.Unlock()
	if text, err := key.Encode(); err != nil || text != offer.Key {
		t.Fatalf("the offered key %q does not carry the listening port and waiting secret: %v", offer.Key, err)
	}
	return key
}

func begin(t *testing.T, server *Server) linkkey.Key {
	t.Helper()
	offer, err := server.Begin()
	if err != nil {
		t.Fatal(err)
	}
	return server.offeredKey(t, offer)
}

func extensionDial() *websocket.DialOptions {
	return &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {extensionOrigin}}}
}

func dial(port int, path string, options *websocket.DialOptions) (*websocket.Conn, *http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()
	return websocket.Dial(ctx, fmt.Sprintf("ws://127.0.0.1:%d%s", port, path), options)
}

func connect(t *testing.T, port int, path string) *websocket.Conn {
	t.Helper()
	conn, _, err := dial(port, path, extensionDial())
	if err != nil {
		t.Fatal(err)
	}
	conn.SetReadLimit(linkproto.MaxMessageBytes)
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

func send(conn *websocket.Conn, message []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()
	return conn.Write(ctx, websocket.MessageBinary, message)
}

func receive(conn *websocket.Conn) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()
	_, message, err := conn.Read(ctx)
	return message, err
}

// closeStatus reads until the desktop app closes the connection and returns its close code.
func closeStatus(t *testing.T, conn *websocket.Conn) websocket.StatusCode {
	t.Helper()
	for {
		if _, err := receive(conn); err != nil {
			return websocket.CloseStatus(err)
		}
	}
}

func wantStatus(t *testing.T, got websocket.StatusCode, want linkproto.CloseCode) {
	t.Helper()
	if got != websocket.StatusCode(want) {
		t.Fatalf("closed with %d, want %d", got, want)
	}
}

// extension is a flynn/noise initiator over a WebSocket; greeting is the last session greeting.
type extension struct {
	key      linkproto.KeyPair
	desktop  []byte
	greeting string
}

func newExtension(t *testing.T) *extension {
	t.Helper()
	key, err := linkproto.GenerateKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &extension{key: key}
}

// link runs the link handshake with secret and returns the transport message that follows it.
func (e *extension) link(conn *websocket.Conn, secret []byte, name string) (string, error) {
	handshake, err := linkPeer(e.key, secret)
	if err != nil {
		return "", err
	}
	first, err := handshake.Write(nil)
	if err != nil {
		return "", err
	}
	if err := send(conn, first); err != nil {
		return "", err
	}
	second, err := receive(conn)
	if err != nil {
		return "", err
	}
	if _, err := handshake.Read(second); err != nil {
		return "", err
	}
	payload, err := json.Marshal(linkproto.LinkRequest{Name: name})
	if err != nil {
		return "", err
	}
	third, err := handshake.Write(payload)
	if err != nil {
		return "", err
	}
	if err := send(conn, third); err != nil {
		return "", err
	}
	sealed, err := receive(conn)
	if err != nil {
		return "", err
	}
	opened, err := handshake.Transport().Open(sealed)
	if err != nil {
		return "", err
	}
	e.desktop = handshake.PeerStatic()
	return string(opened), nil
}

// linkWith links the extension with key and expects the desktop app to confirm and close.
func (e *extension) linkWith(t *testing.T, key linkkey.Key) {
	t.Helper()
	path := linkproto.LinkPath
	if len(e.key.Public) == linkproto.P256PublicKeySize {
		path = linkproto.P256LinkPath
	}
	conn := connect(t, int(key.Port), path)
	message, err := e.link(conn, key.Secret[:], extensionName)
	if err != nil {
		t.Fatal(err)
	}
	if message != `{"type":"linked","language":"en","signIn":"card"}` {
		t.Fatalf("linked message = %q", message)
	}
	wantStatus(t, closeStatus(t, conn), linkproto.CloseFinished)
}

// session runs the session handshake towards the desktop key learned while linking.
func (e *extension) session(conn *websocket.Conn) (*peerTransport, error) {
	handshake, err := sessionPeer(e.key, e.desktop)
	if err != nil {
		return nil, err
	}
	first, err := handshake.Write(nil)
	if err != nil {
		return nil, err
	}
	if err := send(conn, first); err != nil {
		return nil, err
	}
	second, err := receive(conn)
	if err != nil {
		return nil, err
	}
	greeting, err := handshake.Read(second)
	if err != nil {
		return nil, err
	}
	e.greeting = string(greeting)
	return handshake.Transport(), nil
}

func request(t *testing.T, conn *websocket.Conn, transport *peerTransport, plaintext string) string {
	t.Helper()
	sendRequest(t, conn, transport, plaintext)
	return string(nextMessage(t, conn, transport))
}

func sendRequest(t *testing.T, conn *websocket.Conn, transport *peerTransport, plaintext string) {
	t.Helper()
	sealed, err := transport.Seal([]byte(plaintext))
	if err != nil {
		t.Fatal(err)
	}
	if err := send(conn, sealed); err != nil {
		t.Fatal(err)
	}
}

// nextMessage reads and opens the desktop app's next message.
func nextMessage(t *testing.T, conn *websocket.Conn, transport *peerTransport) []byte {
	t.Helper()
	reply, err := receive(conn)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := transport.Open(reply)
	if err != nil {
		t.Fatal(err)
	}
	return opened
}

// linked links an extension to a new server and returns it with the server's vault and port.
func linked(t *testing.T) (*extension, *fakeVault, int) {
	t.Helper()
	server, vault := newServer(t, recordsPath(t))
	key := begin(t, server)
	extension := newExtension(t)
	extension.linkWith(t, key)
	return extension, vault, int(key.Port)
}

// openSession connects to port and runs the session handshake.
func (e *extension) openSession(t *testing.T, port int) (*websocket.Conn, *peerTransport) {
	t.Helper()
	path := linkproto.SessionPath
	if len(e.key.Public) == linkproto.P256PublicKeySize {
		path = linkproto.P256SessionPath
	}
	conn := connect(t, port, path)
	transport, err := e.session(conn)
	if err != nil {
		t.Fatal(err)
	}
	return conn, transport
}

// closingRequest sends one request and returns the code the desktop app closes with.
func closingRequest(t *testing.T, conn *websocket.Conn, transport *peerTransport, plaintext string) websocket.StatusCode {
	t.Helper()
	sealed, err := transport.Seal([]byte(plaintext))
	if err != nil {
		t.Fatal(err)
	}
	if err := send(conn, sealed); err != nil {
		t.Fatal(err)
	}
	return closeStatus(t, conn)
}

func canDial(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func TestTheServerListensOnlyWhileNeeded(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if server.listeningPort() != 0 {
		t.Fatal("the server listens with nothing linked and no key waiting")
	}
	key := begin(t, server)
	if !canDial(int(key.Port)) {
		t.Fatal("the server does not listen while a key waits")
	}
	server.Cancel()
	if server.listeningPort() != 0 || canDial(int(key.Port)) {
		t.Fatal("the server kept listening after the key was canceled")
	}

	extension := newExtension(t)
	extension.linkWith(t, begin(t, server))
	if server.listeningPort() == 0 {
		t.Fatal("the server stopped listening with an extension linked")
	}
	extensions, err := server.Extensions()
	if err != nil || len(extensions) != 1 {
		t.Fatalf("extensions = %+v, error = %v", extensions, err)
	}
	if err := server.Unlink(extensions[0].ID); err != nil {
		t.Fatal(err)
	}
	if server.listeningPort() != 0 {
		t.Fatal("the server kept listening after the last extension was unlinked")
	}
	if err := server.Unlink(extensions[0].ID); !errors.Is(err, linkstore.ErrNotFound) {
		t.Fatalf("unlinking twice: got %v, want ErrNotFound", err)
	}
}

func TestRequestsFromOtherOriginsOrHostsAreRefusedBeforeTheUpgrade(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	port := int(key.Port)
	refusals := map[string]*websocket.DialOptions{
		"no origin":         {},
		"a web page":        {HTTPHeader: http.Header{"Origin": {"https://example.com"}}},
		"another extension": {HTTPHeader: http.Header{"Origin": {"chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}},
		"a rebound name":    {HTTPHeader: http.Header{"Origin": {extensionOrigin}}, Host: "ravenpass.example:" + strconv.Itoa(port)},
		"localhost":         {HTTPHeader: http.Header{"Origin": {extensionOrigin}}, Host: "localhost:" + strconv.Itoa(port)},
		"another port":      {HTTPHeader: http.Header{"Origin": {extensionOrigin}}, Host: "127.0.0.1:1"},
		"null origin":       {HTTPHeader: http.Header{"Origin": {"null"}}},
		"two origins":       {HTTPHeader: http.Header{"Origin": {extensionOrigin, "https://example.com"}}},
		"WebKit page":       {HTTPHeader: http.Header{"Origin": {"https://webkit-extension.example.com"}}},
		"WebKit path":       {HTTPHeader: http.Header{"Origin": {"webkit-extension://12345678-1234-1234-1234-123456789abc/page"}}},
		"WebKit port":       {HTTPHeader: http.Header{"Origin": {"webkit-extension://12345678-1234-1234-1234-123456789abc:80"}}},
	}
	for name, options := range refusals {
		for _, path := range []string{linkproto.LinkPath, linkproto.SessionPath, linkproto.P256LinkPath, linkproto.P256SessionPath} {
			conn, response, err := dial(port, path, options)
			if err == nil {
				conn.CloseNow()
				t.Fatalf("%s at %s was upgraded", name, path)
			}
			if response == nil || response.StatusCode != http.StatusForbidden {
				t.Fatalf("%s at %s: response %v, error %v", name, path, response, err)
			}
		}
	}
	_, response, err := dial(port, "/v1/other", extensionDial())
	if err == nil || response == nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("another path: response %v, error %v", response, err)
	}
	if _, waiting := server.Waiting(); !waiting {
		t.Fatal("a refused request spent the key")
	}
}

func TestAnExtensionLinksWithTheWaitingKey(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	server, _ := newServer(t, recordsPath(t), func(s *Server) { s.now = func() time.Time { return now } })
	offer, err := server.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if !offer.ExpiresAt.Equal(now.Add(5 * time.Minute)) {
		t.Fatalf("the key expires at %v", offer.ExpiresAt)
	}
	waiting, ok := server.Waiting()
	if !ok || waiting.Key != offer.Key || !waiting.ExpiresAt.Equal(offer.ExpiresAt) || waiting.Done != offer.Done {
		t.Fatalf("waiting = %+v, %v, want %+v", waiting, ok, offer)
	}
	key := server.offeredKey(t, offer)
	type outcome struct {
		extension linkstore.Extension
		err       error
	}
	awaiting := newWatchedContext(context.Background())
	awaited := make(chan outcome, 1)
	go func() {
		extension, err := server.Await(awaiting)
		awaited <- outcome{extension, err}
	}()
	<-awaiting.waiting

	extension := newExtension(t)
	extension.linkWith(t, key)
	desktop, err := server.store.DesktopKey()
	if err != nil || !bytes.Equal(extension.desktop, desktop.Public) {
		t.Fatalf("the extension learned another desktop key, error = %v", err)
	}
	result := <-awaited
	if result.err != nil || result.extension.Name != extensionName || !bytes.Equal(result.extension.PublicKey, extension.key.Public) {
		t.Fatalf("await = %+v, error = %v", result.extension, result.err)
	}
	recorded, found, err := server.store.Find(extension.key.Public)
	if err != nil || !found || recorded.ID != result.extension.ID {
		t.Fatalf("recorded = %+v, found = %v, error = %v", recorded, found, err)
	}
	if _, waiting := server.Waiting(); waiting {
		t.Fatal("the key still waits after it was spent")
	}
}

func ended(offer Offer) bool {
	select {
	case <-offer.Done:
		return true
	default:
		return false
	}
}

func currentOffer(t *testing.T, server *Server) Offer {
	t.Helper()
	offer, waiting := server.Waiting()
	if !waiting {
		t.Fatal("no key waits")
	}
	return offer
}

func TestAnOfferIsDoneOnceItsKeyStopsWorking(t *testing.T) {
	t.Run("linked", func(t *testing.T) {
		server, _ := newServer(t, recordsPath(t))
		key := begin(t, server)
		offer := currentOffer(t, server)
		newExtension(t).linkWith(t, key)
		if !ended(offer) {
			t.Fatal("a spent key is not done")
		}
	})
	t.Run("expired", func(t *testing.T) {
		server, _ := newServer(t, recordsPath(t), func(s *Server) { s.keyLifetime = 50 * time.Millisecond })
		offer, err := server.Begin()
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-offer.Done:
		case <-time.After(testWait):
			t.Fatal("an expired key is not done")
		}
		if _, waiting := server.Waiting(); waiting {
			t.Fatal("an ended key still waits")
		}
	})
	t.Run("canceled", func(t *testing.T) {
		server, _ := newServer(t, recordsPath(t))
		offer, err := server.Begin()
		if err != nil {
			t.Fatal(err)
		}
		server.Cancel()
		if !ended(offer) {
			t.Fatal("a canceled key is not done")
		}
	})
	t.Run("replaced", func(t *testing.T) {
		server, _ := newServer(t, recordsPath(t))
		first, err := server.Begin()
		if err != nil {
			t.Fatal(err)
		}
		second, err := server.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if !ended(first) {
			t.Fatal("a replaced key is not done")
		}
		if ended(second) {
			t.Fatal("the replacing key is done")
		}
	})
	t.Run("await canceled", func(t *testing.T) {
		server, _ := newServer(t, recordsPath(t))
		offer, err := server.Begin()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := server.Await(ctx); !errors.Is(err, ErrCanceled) {
			t.Fatalf("await with an ended context: got %v, want ErrCanceled", err)
		}
		if !ended(offer) {
			t.Fatal("the key of a canceled wait is not done")
		}
	})
	t.Run("server closed", func(t *testing.T) {
		server, _ := newServer(t, recordsPath(t))
		offer, err := server.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if err := server.Close(); err != nil {
			t.Fatal(err)
		}
		if !ended(offer) {
			t.Fatal("the key of a closed server is not done")
		}
	})
}

func TestAWrongSecretFailsWithoutSpendingTheKey(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	offer := currentOffer(t, server)
	impostor := newExtension(t)
	conn := connect(t, int(key.Port), linkproto.LinkPath)
	wrong := make([]byte, linkkey.SecretSize)
	if _, err := rand.Read(wrong); err != nil {
		t.Fatal(err)
	}
	_, err := impostor.link(conn, wrong, extensionName)
	wantStatus(t, websocket.CloseStatus(err), linkproto.CloseUnauthenticated)
	if _, waiting := server.Waiting(); !waiting || ended(offer) {
		t.Fatal("a failed handshake spent the key")
	}
	if extensions, _ := server.Extensions(); len(extensions) != 0 {
		t.Fatalf("a failed handshake recorded %+v", extensions)
	}
	newExtension(t).linkWith(t, key)
}

func TestAMalformedLinkRequestFailsWithoutSpendingTheKey(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	offer := currentOffer(t, server)
	conn := connect(t, int(key.Port), linkproto.LinkPath)
	_, err := newExtension(t).link(conn, key.Secret[:], "")
	wantStatus(t, websocket.CloseStatus(err), linkproto.CloseMalformed)
	if _, waiting := server.Waiting(); !waiting || ended(offer) {
		t.Fatal("a malformed request spent the key")
	}
}

func TestSpentAndExpiredKeysAreRefused(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	spent := begin(t, server)
	newExtension(t).linkWith(t, spent)

	conn := connect(t, int(spent.Port), linkproto.LinkPath)
	_, err := newExtension(t).link(conn, spent.Secret[:], extensionName)
	wantStatus(t, websocket.CloseStatus(err), linkproto.CloseNoKey)

	server.keyLifetime = 200 * time.Millisecond
	expired := begin(t, server)
	if _, err := server.Await(context.Background()); !errors.Is(err, ErrExpired) {
		t.Fatalf("await: got %v, want ErrExpired", err)
	}
	if _, waiting := server.Waiting(); waiting {
		t.Fatal("an expired key still waits")
	}
	conn = connect(t, int(expired.Port), linkproto.LinkPath)
	_, err = newExtension(t).link(conn, expired.Secret[:], extensionName)
	wantStatus(t, websocket.CloseStatus(err), linkproto.CloseNoKey)
	if extensions, _ := server.Extensions(); len(extensions) != 1 {
		t.Fatalf("refused keys recorded extensions: %+v", extensions)
	}
}

func TestALinkedExtensionAsksForTheVaultState(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	key := begin(t, server)
	extension := newExtension(t)
	extension.linkWith(t, key)

	conn := connect(t, int(key.Port), linkproto.SessionPath)
	transport, err := extension.session(conn)
	if err != nil {
		t.Fatal(err)
	}
	if reply := request(t, conn, transport, `{"id":1,"type":"status"}`); reply != `{"id":1,"result":{"vault":"locked"}}` {
		t.Fatalf("status while locked = %s", reply)
	}
	vault.unlocked.Store(true)
	if reply := request(t, conn, transport, `{"id":2,"type":"status"}`); reply != `{"id":2,"result":{"vault":"unlocked"}}` {
		t.Fatalf("status while unlocked = %s", reply)
	}
	if reply := request(t, conn, transport, `{"id":3,"type":"rename"}`); reply != `{"id":3,"error":"unknown-request"}` {
		t.Fatalf("unknown request = %s", reply)
	}
	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatal(err)
	}
}

func TestALinkAndEachSessionNameTheDesktopLanguageAndSignInStyle(t *testing.T) {
	settings := &fakeSettings{language: "ru", signIn: "card"}
	server, _ := newServer(t, recordsPath(t), func(s *Server) { s.settings = settings })
	key := begin(t, server)
	extension := newExtension(t)
	message, err := extension.link(connect(t, int(key.Port), linkproto.LinkPath), key.Secret[:], extensionName)
	if err != nil {
		t.Fatal(err)
	}
	if message != `{"type":"linked","language":"ru","signIn":"card"}` {
		t.Fatalf("linked message = %q", message)
	}
	extension.openSession(t, int(key.Port))
	if extension.greeting != `{"language":"ru","signIn":"card"}` {
		t.Fatalf("session greeting = %q", extension.greeting)
	}
	settings.change("en", "field")
	extension.openSession(t, int(key.Port))
	if extension.greeting != `{"language":"en","signIn":"field"}` {
		t.Fatalf("session greeting after the settings changed = %q", extension.greeting)
	}
}

func TestAnUnknownKeyIsNotLinked(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	linked := newExtension(t)
	linked.linkWith(t, key)

	stranger := newExtension(t)
	stranger.desktop = linked.desktop
	conn := connect(t, int(key.Port), linkproto.SessionPath)
	_, err := stranger.session(conn)
	wantStatus(t, websocket.CloseStatus(err), linkproto.CloseNotLinked)
}

func TestASessionTowardsAnotherDesktopKeyFails(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	extension := newExtension(t)
	extension.linkWith(t, key)

	other, err := linkproto.GenerateKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	extension.desktop = other.Public
	conn := connect(t, int(key.Port), linkproto.SessionPath)
	_, err = extension.session(conn)
	wantStatus(t, websocket.CloseStatus(err), linkproto.CloseUnauthenticated)
}

func TestAnExtensionUnlinksItselfOverASession(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	extension := newExtension(t)
	extension.linkWith(t, key)

	conn := connect(t, int(key.Port), linkproto.SessionPath)
	transport, err := extension.session(conn)
	if err != nil {
		t.Fatal(err)
	}
	if reply := request(t, conn, transport, `{"id":1,"type":"unlink"}`); reply != `{"id":1,"result":{}}` {
		t.Fatalf("unlink = %s", reply)
	}
	wantStatus(t, closeStatus(t, conn), linkproto.CloseFinished)
	if extensions, err := server.Extensions(); err != nil || len(extensions) != 0 {
		t.Fatalf("extensions after unlinking = %+v, error = %v", extensions, err)
	}
	if server.listeningPort() != 0 || canDial(int(key.Port)) {
		t.Fatal("the server kept listening with nothing linked")
	}
}

const (
	exampleFill      = `{"id":1,"type":"fill","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://example.com"}`
	exampleFillReply = `{"id":1,"result":{"login":"alex","email":"alex@example.com","password":"secret"}}`
)

// linkedID returns the id under which e is linked to server.
func linkedID(t *testing.T, server *Server, e *extension) string {
	t.Helper()
	recorded, found, err := server.store.Find(e.key.Public)
	if err != nil || !found {
		t.Fatalf("the extension is not linked: found = %v, error = %v", found, err)
	}
	return recorded.ID
}

func TestUnlinkingAnExtensionClosesOnlyItsSessions(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	key := begin(t, server)
	port := int(key.Port)
	unlinked := newExtension(t)
	unlinked.linkWith(t, key)
	kept := newExtension(t)
	kept.linkWith(t, begin(t, server))
	idle, _ := unlinked.openSession(t, port)
	asking, askingTransport := unlinked.openSession(t, port)
	other, otherTransport := kept.openSession(t, port)

	if err := server.Unlink(linkedID(t, server, unlinked)); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, closeStatus(t, idle), linkproto.CloseNotLinked)
	sealed, err := askingTransport.Seal([]byte(exampleFill))
	if err != nil {
		t.Fatal(err)
	}
	// The desktop app may have closed the session before the request arrives.
	_ = send(asking, sealed)
	wantStatus(t, closeStatus(t, asking), linkproto.CloseNotLinked)
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("a request after unlinking reached the vault: %q", calls)
	}
	if reply := request(t, other, otherTransport, exampleFill); reply != exampleFillReply {
		t.Fatalf("another extension's fill after unlinking = %s", reply)
	}
}

func TestAnExtensionUnlinkingItselfClosesItsOtherSessions(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	extension := newExtension(t)
	extension.linkWith(t, key)
	other, _ := extension.openSession(t, int(key.Port))
	conn, transport := extension.openSession(t, int(key.Port))

	if reply := request(t, conn, transport, `{"id":1,"type":"unlink"}`); reply != `{"id":1,"result":{}}` {
		t.Fatalf("unlink = %s", reply)
	}
	wantStatus(t, closeStatus(t, conn), linkproto.CloseFinished)
	wantStatus(t, closeStatus(t, other), linkproto.CloseNotLinked)
}

func TestASessionWhoseExtensionIsNoLongerLinkedIsRefused(t *testing.T) {
	t.Run("removed", func(t *testing.T) {
		server, vault := newServer(t, recordsPath(t))
		key := begin(t, server)
		extension := newExtension(t)
		extension.linkWith(t, key)
		conn, transport := extension.openSession(t, int(key.Port))
		if err := server.store.Remove(linkedID(t, server, extension)); err != nil {
			t.Fatal(err)
		}
		wantStatus(t, closingRequest(t, conn, transport, exampleFill), linkproto.CloseNotLinked)
		if calls := vault.takeCalls(); len(calls) != 0 {
			t.Fatalf("a request from an unlinked extension reached the vault: %q", calls)
		}
	})
	t.Run("linked again", func(t *testing.T) {
		server, vault := newServer(t, recordsPath(t))
		key := begin(t, server)
		extension := newExtension(t)
		extension.linkWith(t, key)
		earlier, earlierTransport := extension.openSession(t, int(key.Port))
		extension.linkWith(t, begin(t, server))
		wantStatus(t, closingRequest(t, earlier, earlierTransport, exampleFill), linkproto.CloseNotLinked)
		if calls := vault.takeCalls(); len(calls) != 0 {
			t.Fatalf("a request from an earlier link reached the vault: %q", calls)
		}
		conn, transport := extension.openSession(t, int(key.Port))
		if reply := request(t, conn, transport, exampleFill); reply != exampleFillReply {
			t.Fatalf("fill after linking again = %s", reply)
		}
	})
}

func TestOversizedAndTextMessagesAreMalformed(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	extension := newExtension(t)
	extension.linkWith(t, key)

	begin(t, server)
	for _, path := range []string{linkproto.SessionPath, linkproto.LinkPath} {
		conn := connect(t, int(key.Port), path)
		if err := send(conn, make([]byte, linkproto.MaxMessageBytes+1)); err != nil {
			t.Fatal(err)
		}
		wantStatus(t, closeStatus(t, conn), linkproto.CloseMalformed)
	}

	conn := connect(t, int(key.Port), linkproto.SessionPath)
	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"id":1,"type":"status"}`)); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, closeStatus(t, conn), linkproto.CloseMalformed)

	conn = connect(t, int(key.Port), linkproto.SessionPath)
	transport, err := extension.session(conn)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := transport.Seal([]byte(`{"type":"status"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := send(conn, sealed); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, closeStatus(t, conn), linkproto.CloseMalformed)
}

func TestSilentConnectionsAreClosedAsIdle(t *testing.T) {
	const idle = 600 * time.Millisecond
	server, _ := newServer(t, recordsPath(t), func(s *Server) {
		s.handshakeTimeout = 500 * time.Millisecond
		s.idleTimeout = idle
	})
	key := begin(t, server)
	silent := connect(t, int(key.Port), linkproto.LinkPath)
	wantStatus(t, closeStatus(t, silent), linkproto.CloseIdle)
	if _, waiting := server.Waiting(); !waiting {
		t.Fatal("an idle handshake spent the key")
	}

	extension := newExtension(t)
	extension.linkWith(t, key)
	conn := connect(t, int(key.Port), linkproto.SessionPath)
	transport, err := extension.session(conn)
	if err != nil {
		t.Fatal(err)
	}
	// Four requests idle/3 apart outlast one idle timeout only if each request restarts it.
	for id := range 4 {
		time.Sleep(idle / 3)
		reply := request(t, conn, transport, fmt.Sprintf(`{"id":%d,"type":"status"}`, id))
		if reply != fmt.Sprintf(`{"id":%d,"result":{"vault":"locked"}}`, id) {
			t.Fatalf("status = %s", reply)
		}
	}
	wantStatus(t, closeStatus(t, conn), linkproto.CloseIdle)
}

func TestAtMostFourConnectionsAreServedAtOnce(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	held := make([]*websocket.Conn, 0, maxConnections)
	for range maxConnections {
		held = append(held, connect(t, int(key.Port), linkproto.SessionPath))
	}
	_, response, err := dial(int(key.Port), linkproto.SessionPath, extensionDial())
	if err == nil || response == nil || response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("a fifth connection: response %v, error %v", response, err)
	}
	held[0].CloseNow()
	waitUntil(t, "a connection is admitted after one closed", func() error {
		conn, _, err := dial(int(key.Port), linkproto.SessionPath, extensionDial())
		if err == nil {
			conn.CloseNow()
		}
		return err
	})
}

// waitUntil retries attempt every 10 ms and fails with its last error once testWait has passed.
func waitUntil(t *testing.T, what string, attempt func() error) {
	t.Helper()
	deadline := time.Now().Add(testWait)
	for {
		err := attempt()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: still failing after %v: %v", what, testWait, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestThePortIsKeptAcrossRestarts(t *testing.T) {
	path := recordsPath(t)
	first, _ := newServer(t, path)
	key := begin(t, first)
	extension := newExtension(t)
	extension.linkWith(t, key)
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, vault := newServer(t, path)
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	if second.listeningPort() != int(key.Port) {
		t.Fatalf("the restarted server listens on %d, want %d", second.listeningPort(), key.Port)
	}
	vault.unlocked.Store(true)
	conn := connect(t, int(key.Port), linkproto.SessionPath)
	transport, err := extension.session(conn)
	if err != nil {
		t.Fatal(err)
	}
	if reply := request(t, conn, transport, `{"id":1,"type":"status"}`); reply != `{"id":1,"result":{"vault":"unlocked"}}` {
		t.Fatalf("status after a restart = %s", reply)
	}
}

func TestATakenPortLeavesLinkedExtensionsUnreachable(t *testing.T) {
	path := recordsPath(t)
	first, _ := newServer(t, path)
	key := begin(t, first)
	newExtension(t).linkWith(t, key)
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	occupant, err := bind(int(key.Port))
	if err != nil {
		t.Fatal(err)
	}
	second, _ := newServer(t, path)
	if err := second.Start(); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("start on a taken port: got %v, want ErrUnavailable", err)
	}
	if second.Reachable() {
		t.Fatal("the server reports itself reachable on a taken port")
	}
	if _, err := second.Begin(); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("a key on a taken port: got %v, want ErrUnavailable", err)
	}
	occupant.Close()
	if again := begin(t, second); again.Port != key.Port {
		t.Fatalf("the server moved to port %d with an extension linked", again.Port)
	}
	if !second.Reachable() {
		t.Fatal("the server stayed unreachable after it could listen")
	}
}

func TestATakenPortIsReplacedWhileNothingIsLinked(t *testing.T) {
	path := recordsPath(t)
	server, _ := newServer(t, path)
	first := begin(t, server)
	server.Cancel()
	occupant, err := bind(int(first.Port))
	if err != nil {
		t.Fatal(err)
	}
	defer occupant.Close()
	second := begin(t, server)
	if second.Port == first.Port {
		t.Fatal("the server reused a taken port")
	}
	kept, err := server.store.Port()
	if err != nil || kept != int(second.Port) {
		t.Fatalf("kept port = %d, error = %v, want %d", kept, err, second.Port)
	}
	if !server.Reachable() {
		t.Fatal("the server reports itself unreachable after it moved")
	}
}

func TestAwaitEndsWhenTheKeyIsCanceledOrReplaced(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	if _, err := server.Await(context.Background()); !errors.Is(err, ErrCanceled) {
		t.Fatalf("await with no key: got %v, want ErrCanceled", err)
	}

	begin(t, server)
	awaited := make(chan error, 1)
	go func() {
		_, err := server.Await(context.Background())
		awaited <- err
	}()
	time.AfterFunc(20*time.Millisecond, server.Cancel)
	if err := <-awaited; !errors.Is(err, ErrCanceled) {
		t.Fatalf("await after cancel: got %v, want ErrCanceled", err)
	}

	begin(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := server.Await(ctx); !errors.Is(err, ErrCanceled) {
		t.Fatalf("await with an ended context: got %v, want ErrCanceled", err)
	}
	if _, waiting := server.Waiting(); waiting {
		t.Fatal("an ended await left its key waiting")
	}
	if server.listeningPort() != 0 {
		t.Fatal("an ended await left the server listening")
	}
}

// watchedContext reports when Await first waits on it, which is after it took the waiting key.
type watchedContext struct {
	context.Context
	once    sync.Once
	waiting chan struct{}
}

func newWatchedContext(parent context.Context) *watchedContext {
	return &watchedContext{Context: parent, waiting: make(chan struct{})}
}

func (c *watchedContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func TestALateCancellationNeverEndsANewerKey(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	begin(t, server)
	parent, cancelFirst := context.WithCancel(context.Background())
	first := newWatchedContext(parent)
	awaited := make(chan error, 1)
	go func() {
		_, err := server.Await(first)
		awaited <- err
	}()
	<-first.waiting

	second := begin(t, server)
	if err := <-awaited; !errors.Is(err, ErrCanceled) {
		t.Fatalf("await on a replaced key: got %v, want ErrCanceled", err)
	}
	cancelFirst()
	if waiting, ok := server.Waiting(); !ok {
		t.Fatal("cancelling the first await ended the second key")
	} else if text, err := second.Encode(); err != nil || text != waiting.Key {
		t.Fatalf("the waiting key is not the second one: %v", err)
	}
	newExtension(t).linkWith(t, second)
}

func TestAClosedServerOffersNoKey(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if canDial(int(key.Port)) {
		t.Fatal("a closed server still listens")
	}
	if _, waiting := server.Waiting(); waiting {
		t.Fatal("a closed server kept its key")
	}
	if _, err := server.Begin(); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("a key after closing: got %v, want ErrUnavailable", err)
	}
}

func TestASessionSuggestsFillsGivesCodesShowsIconsAndUnlocks(t *testing.T) {
	extension, vault, port := linked(t)
	conn, transport := extension.openSession(t, port)
	tests := []struct {
		request string
		reply   string
		call    string
	}{
		{
			`{"id":1,"type":"suggest","origin":"https://example.com","purpose":"sign-in"}`,
			`{"id":1,"result":{"credentials":[{"id":"0102030405060708090a0b0c0d0e0f10","label":"Example","account":"alex","site":"example.com","exact":true}]}}`,
			"suggest sign-in https://example.com",
		},
		{
			`{"id":2,"type":"suggest","origin":"http://other.example:8080","purpose":"sign-in"}`,
			`{"id":2,"result":{"credentials":[]}}`,
			"suggest sign-in http://other.example:8080",
		},
		{
			`{"id":3,"type":"fill","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://example.com"}`,
			`{"id":3,"result":{"login":"alex","email":"alex@example.com","password":"secret"}}`,
			"fill 0102030405060708090a0b0c0d0e0f10 https://example.com",
		},
		{
			`{"id":4,"type":"icon","site":"example.com"}`,
			`{"id":4,"result":{"image":"iVBORw0KGgo=","tint":"#1a2b3c"}}`,
			"icon example.com",
		},
		{
			`{"id":5,"type":"suggest","origin":"https://` + strings.Repeat("a", linkproto.MaxOriginLength-len("https://")) + `","purpose":"sign-in"}`,
			`{"id":5,"result":{"credentials":[]}}`,
			"suggest sign-in https://" + strings.Repeat("a", linkproto.MaxOriginLength-len("https://")),
		},
		{
			`{"id":6,"type":"icon","site":"` + strings.Repeat("a", linkproto.MaxSiteLength) + `"}`,
			`{"id":6,"result":{"image":"iVBORw0KGgo=","tint":"#1a2b3c"}}`,
			"icon " + strings.Repeat("a", linkproto.MaxSiteLength),
		},
		{
			`{"id":7,"type":"suggest","origin":"https://example.com","purpose":"sign-in"}`,
			`{"id":7,"result":{"credentials":[{"id":"0102030405060708090a0b0c0d0e0f10","label":"Example","account":"alex","site":"example.com","exact":true}]}}`,
			"suggest sign-in https://example.com",
		},
		{
			`{"id":8,"type":"suggest","origin":"https://example.com","purpose":"code"}`,
			`{"id":8,"result":{"credentials":[{"id":"0102030405060708090a0b0c0d0e0f10","label":"Example","account":"alex","site":"example.com","exact":true,"digits":6,"period":30}]}}`,
			"suggest code https://example.com",
		},
		{
			`{"id":9,"type":"code","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://example.com"}`,
			`{"id":9,"result":{"code":"287082","digits":6,"period":30,"expiresAt":1790000010000}}`,
			"code 0102030405060708090a0b0c0d0e0f10 https://example.com",
		},
		{
			`{"id":10,"type":"add-website","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://secure.example.com"}`,
			`{"id":10,"result":{}}`,
			"add 0102030405060708090a0b0c0d0e0f10 https://secure.example.com",
		},
	}
	for _, test := range tests {
		if reply := request(t, conn, transport, test.request); reply != test.reply {
			t.Fatalf("%s answered %s, want %s", test.request, reply, test.reply)
		}
		if calls := vault.takeCalls(); !slices.Equal(calls, []string{test.call}) {
			t.Fatalf("%s asked the vault %q", test.request, calls)
		}
	}
	if reply := request(t, conn, transport, `{"id":11,"type":"unlock"}`); reply != `{"id":11,"result":{}}` {
		t.Fatalf("unlock = %s", reply)
	}
	if vault.unlocks.Load() != 1 {
		t.Fatalf("unlock showed the window %d times", vault.unlocks.Load())
	}
	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatal(err)
	}
}

func TestVaultRefusalsAnswerTheirErrorCodes(t *testing.T) {
	extension, vault, port := linked(t)
	conn, transport := extension.openSession(t, port)
	suggest := `{"id":1,"type":"suggest","origin":"https://example.com","purpose":"sign-in"}`
	suggestCodes := `{"id":1,"type":"suggest","origin":"https://example.com","purpose":"code"}`
	fill := `{"id":1,"type":"fill","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://example.com"}`
	code := `{"id":1,"type":"code","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://example.com"}`
	addWebsite := `{"id":1,"type":"add-website","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://secure.example.com"}`
	icon := `{"id":1,"type":"icon","site":"example.com"}`
	identities := `{"id":1,"type":"identities","origin":"https://example.com"}`
	share := `{"id":1,"type":"share","identity":"` + exampleID + `","file":"photo","origin":"https://example.com"}`
	tests := []struct {
		refusal error
		request string
		code    string
	}{
		{ErrLocked, suggest, linkproto.ErrorLocked},
		{ErrLocked, suggestCodes, linkproto.ErrorLocked},
		{ErrLocked, fill, linkproto.ErrorLocked},
		{ErrLocked, code, linkproto.ErrorLocked},
		{ErrLocked, icon, linkproto.ErrorLocked},
		{ErrLocked, identities, linkproto.ErrorLocked},
		{ErrLocked, share, linkproto.ErrorLocked},
		{ErrNotFound, share, linkproto.ErrorNotFound},
		{ErrDeclined, share, linkproto.ErrorDeclined},
		{fmt.Errorf("share: %w", ErrDeclined), share, linkproto.ErrorDeclined},
		{ErrUnverifiable, share, linkproto.ErrorUnverifiable},
		{ErrNotFound, fill, linkproto.ErrorNotFound},
		{ErrNotFound, code, linkproto.ErrorNotFound},
		{ErrNoMatch, fill, linkproto.ErrorNoMatch},
		{ErrNoMatch, code, linkproto.ErrorNoMatch},
		{fmt.Errorf("fill: %w", ErrNoMatch), fill, linkproto.ErrorNoMatch},
		{ErrNoCode, code, linkproto.ErrorNoCode},
		{fmt.Errorf("code: %w", ErrNoCode), code, linkproto.ErrorNoCode},
		{ErrLocked, addWebsite, linkproto.ErrorLocked},
		{ErrNotFound, addWebsite, linkproto.ErrorNotFound},
		{ErrNoMatch, addWebsite, linkproto.ErrorNoMatch},
	}
	for _, test := range tests {
		vault.refuseWith(test.refusal)
		want := `{"id":1,"error":"` + test.code + `"}`
		if reply := request(t, conn, transport, test.request); reply != want {
			t.Fatalf("%s refused with %v answered %s, want %s", test.request, test.refusal, reply, want)
		}
	}
	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatal(err)
	}
}

func TestAnInvalidOriginIsRefusedBeforeTheVault(t *testing.T) {
	extension, vault, port := linked(t)
	conn, transport := extension.openSession(t, port)
	origins := []string{
		"",
		"example.com",
		"ftp://example.com",
		"chrome-extension://cceiadaelnccfbakmhcleifjfilkakag",
		"https://",
		"https://example.com/",
		"https://example.com/login",
		"https://alex@example.com",
		"https://example.com?next=1",
		"https://example.com?",
		"https://example.com#top",
		"https://exa mple.com",
		"https://" + strings.Repeat("a", linkproto.MaxOriginLength-len("https://")+1),
	}
	for _, origin := range origins {
		for _, kind := range []string{"suggest", "fill", "code", "add-website", "identities", "share"} {
			encoded, err := json.Marshal(map[string]any{"id": 1, "type": kind, "credential": exampleID, "identity": exampleID, "file": "photo", "origin": origin})
			if err != nil {
				t.Fatal(err)
			}
			if reply := request(t, conn, transport, string(encoded)); reply != `{"id":1,"error":"invalid-origin"}` {
				t.Fatalf("%s for %q answered %s", kind, origin, reply)
			}
		}
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("invalid origins reached the vault: %q", calls)
	}
	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatal(err)
	}
}

func TestAPlainHTTPPageIsNeverAddedToACredential(t *testing.T) {
	extension, vault, port := linked(t)
	conn, transport := extension.openSession(t, port)
	for _, origin := range []string{"http://example.com", "http://localhost:8080"} {
		encoded, err := json.Marshal(map[string]any{"id": 1, "type": "add-website", "credential": exampleID, "origin": origin})
		if err != nil {
			t.Fatal(err)
		}
		if reply := request(t, conn, transport, string(encoded)); reply != `{"id":1,"error":"invalid-origin"}` {
			t.Fatalf("add-website for %q answered %s", origin, reply)
		}
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("plain http pages reached the vault: %q", calls)
	}
	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatal(err)
	}
}

func TestAnUnknownPurposeIsRefusedBeforeTheVault(t *testing.T) {
	extension, vault, port := linked(t)
	conn, transport := extension.openSession(t, port)
	for _, purpose := range []string{"", "Code", "sign_in", "password", " code", "code ", "one-time-code"} {
		encoded, err := json.Marshal(map[string]any{"id": 1, "type": "suggest", "origin": exampleOrigin, "purpose": purpose})
		if err != nil {
			t.Fatal(err)
		}
		if reply := request(t, conn, transport, string(encoded)); reply != `{"id":1,"error":"invalid-purpose"}` {
			t.Fatalf("suggest for purpose %q answered %s", purpose, reply)
		}
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("unknown purposes reached the vault: %q", calls)
	}
	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedFieldsCloseTheSession(t *testing.T) {
	extension, vault, port := linked(t)
	requests := []string{
		`{"id":1,"type":"suggest","origin":5}`,
		`{"id":1,"type":"suggest","origin":"https://example.com","purpose":7}`,
		`{"id":1,"type":"fill","credential":16,"origin":"https://example.com"}`,
		`{"id":1,"type":"code","credential":16,"origin":"https://example.com"}`,
		`{"id":1,"type":"icon","site":["example.com"]}`,
		`{"id":1,"type":"icon"}`,
		`{"id":1,"type":"icon","site":"` + strings.Repeat("a", linkproto.MaxSiteLength+1) + `"}`,
		`{"id":1,"type":"identities","origin":["https://example.com"]}`,
		`{"id":1,"type":"share","identity":16,"file":"photo","origin":"https://example.com"}`,
		`{"id":1,"type":"share","identity":"` + exampleID + `","file":{},"origin":"https://example.com"}`,
	}
	for _, plaintext := range requests {
		conn, transport := extension.openSession(t, port)
		wantStatus(t, closingRequest(t, conn, transport, plaintext), linkproto.CloseMalformed)
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("malformed requests reached the vault: %q", calls)
	}
}

func TestAVaultFailureEndsTheSessionAsInternal(t *testing.T) {
	extension, vault, port := linked(t)
	vault.refuseWith(errors.New("usage record could not be written"))
	for _, plaintext := range []string{
		`{"id":1,"type":"fill","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://example.com"}`,
		`{"id":1,"type":"code","credential":"0102030405060708090a0b0c0d0e0f10","origin":"https://example.com"}`,
		`{"id":1,"type":"identities","origin":"https://example.com"}`,
		`{"id":1,"type":"share","identity":"0102030405060708090a0b0c0d0e0f10","file":"photo","origin":"https://example.com"}`,
	} {
		conn, transport := extension.openSession(t, port)
		if code := closingRequest(t, conn, transport, plaintext); code != websocket.StatusInternalError {
			t.Fatalf("%s closed with %d, want %d", plaintext, code, websocket.StatusInternalError)
		}
	}
}
