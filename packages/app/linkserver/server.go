// Package linkserver links and serves the browser extension over WebSocket on the loopback interface.
package linkserver

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	mathrand "math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/dortanes/ravenpass/packages/app/captures"
	"github.com/dortanes/ravenpass/packages/app/linkkey"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkstore"
)

const (
	loopback = "127.0.0.1"

	keyLifetime      = 5 * time.Minute
	handshakeTimeout = 10 * time.Second
	idleTimeout      = 30 * time.Second
	maxConnections   = 4
	// maxVerifying bounds the connections waiting for the person, which leave maxConnections to other requests.
	maxVerifying = 4
	// progressInterval stays under the 30 s Chrome lets an extension service worker idle.
	progressInterval = 10 * time.Second

	// firstPort and lastPort bound the IANA dynamic port range, RFC 6335.
	firstPort = 49152
	lastPort  = 65535
	// portAttempts bounds the ports tried while no linked extension depends on the kept one.
	portAttempts = 8
)

var (
	// ErrUnavailable reports a server that cannot listen.
	ErrUnavailable = errors.New("link server cannot listen")
	// ErrExpired reports a connection key that expired.
	ErrExpired = errors.New("connection key expired")
	// ErrCanceled reports a connection key canceled or replaced.
	ErrCanceled = errors.New("connection key was canceled")
)

// Refusals a Vault answers with; a session reports each to the extension by its error code.
var (
	ErrLocked              = errors.New("vault is locked")
	ErrNotFound            = errors.New("item was not found")
	ErrNoMatch             = errors.New("credential does not match the origin")
	ErrNoCode              = errors.New("credential has no one-time code")
	ErrDeclined            = errors.New("the person declined the verification")
	ErrUnverifiable        = errors.New("the vault offers no way to verify the person")
	ErrInvalidAccount      = errors.New("the vault does not accept the account")
	ErrInvalidName         = errors.New("the vault does not accept the name")
	ErrInvalidOrigin       = errors.New("the origin cannot use passkeys")
	ErrInvalidRelyingParty = errors.New("the relying party ID does not fit the origin")
	ErrExcluded            = errors.New("the vault holds a passkey the site excluded")
	ErrPasskeysFull        = errors.New("the credential holds as many passkeys as it can")
)

// Vault is the vault as a session serves it; every call but Unlocked and ShowUnlock fails with ErrLocked while locked.
type Vault interface {
	// Unlocked reports whether the vault is open.
	Unlocked() bool
	// Suggest lists the credentials to offer on origin for purpose.
	Suggest(origin string, purpose linkproto.Purpose) ([]linkproto.Suggestion, error)
	// Fill reads a credential for origin to the linked extension, once the person verifies as Share does where the
	// owner asks for it; ErrNotFound for an unknown one, ErrNoMatch for another origin's.
	Fill(ctx context.Context, extension, credential, origin string, asked func(linkproto.Progress)) (linkproto.Fill, error)
	// OneTimeCode is Fill for the credential's current code; ErrNoCode where it has no code setup.
	OneTimeCode(ctx context.Context, credential, origin string, asked func(linkproto.Progress)) (linkproto.OneTimeCode, error)
	// AddWebsite adds origin to the websites of a credential that already matches it by site; ErrNotFound for an unknown
	// one, ErrNoMatch for another site's.
	AddWebsite(credential, origin string) error
	// SiteIcon returns the icon kept for site.
	SiteIcon(site string) (linkproto.Icon, error)
	// Identities lists the identities and their files.
	Identities() ([]linkproto.Identity, error)
	// Share releases an identity file once the person verifies; ErrNotFound, ErrDeclined or ErrUnverifiable otherwise.
	Share(ctx context.Context, identity, file, origin string, asked func(linkproto.Progress)) (linkproto.SharedFile, []byte, error)
	// ShowUnlock asks the person to unlock the vault on the device.
	ShowUnlock()
	// Cards lists the cards a checkout can take.
	Cards() ([]linkproto.CardOption, error)
	// FillCard reads a card for origin to the linked extension once the person verifies it, every time; ErrNotFound for
	// an unknown one.
	FillCard(ctx context.Context, card, origin string, asked func(linkproto.Progress)) (linkproto.CardFill, error)
	// CaptureOffer answers where capture can be saved; while locked it reports CaptureLocked with no target.
	CaptureOffer(capture Capture) (linkproto.CaptureOffer, error)
	// SaveCapture saves capture as chosen; ErrNotFound, ErrInvalidAccount or ErrInvalidName when refused.
	SaveCapture(capture Capture, choice SaveChoice) (linkproto.Saved, error)
	// Passkeys lists the passkeys a sign-in can use.
	Passkeys(query linkproto.PasskeyQuery) (linkproto.Passkeys, error)
	// PasskeyTargets lists the credentials a new passkey can join.
	PasskeyTargets(query linkproto.PasskeyQuery) (linkproto.PasskeyTargets, error)
	// CreatePasskey checks origin and relying party, verifies as Share does, and adds a passkey.
	CreatePasskey(ctx context.Context, creation linkproto.PasskeyCreation, asked func(linkproto.Progress)) (linkproto.CreatedPasskey, error)
	// SignPasskey checks origin and relying party, verifies as Share does, and signs.
	SignPasskey(ctx context.Context, signIn linkproto.PasskeySignIn, asked func(linkproto.Progress)) (linkproto.PasskeyAssertion, error)
}

// Settings is what the desktop app reports on linking and at every session start.
type Settings interface {
	// Language is the interface language the extension speaks.
	Language() string
	// SignInStyle is how the extension offers sign-in on websites.
	SignInStyle() string
}

// Offer is a connection key waiting for an extension; Done closes once the key stops working.
type Offer struct {
	Key       string
	ExpiresAt time.Time
	Done      <-chan struct{}
}

// offer is a key until it ends; its outcome is set once, under the server's mu, before done closes.
type offer struct {
	secret    [linkkey.SecretSize]byte
	key       string
	expiresAt time.Time
	expiry    *time.Timer
	done      chan struct{}
	extension linkstore.Extension
	err       error
}

// offered is o as Begin and Waiting hand it out.
func (o *offer) offered() Offer {
	return Offer{Key: o.key, ExpiresAt: o.expiresAt, Done: o.done}
}

func (o *offer) waiting() bool {
	if o == nil {
		return false
	}
	select {
	case <-o.done:
		return false
	default:
		return true
	}
}

// settle ends o with its extension or err, and reports whether o was still waiting.
func (o *offer) settle(extension linkstore.Extension, err error) bool {
	if !o.waiting() {
		return false
	}
	o.expiry.Stop()
	o.extension, o.err = extension, err
	close(o.done)
	return true
}

// Server is the link server; the store keeps its port for linked extensions across restarts.
type Server struct {
	store    *linkstore.Store
	vault    Vault
	settings Settings

	now              func() time.Time
	choosePort       func() int
	keyLifetime      time.Duration
	handshakeTimeout time.Duration
	idleTimeout      time.Duration
	progressInterval time.Duration

	slots     chan struct{}
	verifying chan struct{}
	// connections ends every WebSocket on Close; http.Server does not close hijacked connections.
	connections context.Context
	disconnect  context.CancelFunc

	// held keeps the captures each linked extension sent, by its identifier.
	held *captures.Held[Capture]

	mu          sync.Mutex
	listening   *listening
	unreachable bool
	offer       *offer
	sessions    openSessions
	closed      bool
}

type listening struct {
	port     int
	listener net.Listener
	server   *http.Server
}

func (l *listening) stop() error {
	err := l.server.Close()
	// http.Server.Close misses a listener that Serve has not started tracking yet.
	_ = l.listener.Close()
	return err
}

// New composes a Server that does not listen until Start or Begin.
func New(store *linkstore.Store, vault Vault, settings Settings) (*Server, error) {
	if store == nil || vault == nil || settings == nil {
		return nil, errors.New("extension records, vault and settings are required")
	}
	connections, disconnect := context.WithCancel(context.Background())
	return &Server{
		store:            store,
		vault:            vault,
		settings:         settings,
		now:              time.Now,
		choosePort:       randomPort,
		keyLifetime:      keyLifetime,
		handshakeTimeout: handshakeTimeout,
		idleTimeout:      idleTimeout,
		progressInterval: progressInterval,
		slots:            make(chan struct{}, maxConnections),
		verifying:        make(chan struct{}, maxVerifying),
		connections:      connections,
		disconnect:       disconnect,
		held:             captures.New[Capture](captures.Lifetime),
		sessions:         openSessions{},
	}, nil
}

func randomPort() int {
	return firstPort + mathrand.IntN(lastPort-firstPort+1)
}

// Start listens when an extension is linked; an unbindable kept port leaves it unreachable until the next key.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrUnavailable
	}
	extensions, err := s.store.Extensions()
	if err != nil {
		return err
	}
	if len(extensions) == 0 {
		return nil
	}
	return s.listen()
}

// Close cancels the key, forgets captures, stops listening and ends every connection, for good.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.offer.settle(linkstore.Extension{}, ErrCanceled)
	s.held.Close()
	s.disconnect()
	if s.listening == nil {
		return nil
	}
	err := s.listening.stop()
	s.listening = nil
	return err
}

// Reachable is false while linked extensions depend on the server and it cannot listen.
func (s *Server) Reachable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.unreachable
}

// Extensions lists the linked extensions in link order, even after the server closes.
func (s *Server) Extensions() ([]linkstore.Extension, error) {
	return s.store.Extensions()
}

// Begin creates a connection key, replacing the key still waiting, and listens for it.
func (s *Server) Begin() (Offer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Offer{}, ErrUnavailable
	}
	next := &offer{done: make(chan struct{})}
	if _, err := rand.Read(next.secret[:]); err != nil {
		return Offer{}, err
	}
	if _, err := s.store.DesktopKey(); err != nil {
		return Offer{}, err
	}
	if err := s.listen(); err != nil {
		return Offer{}, err
	}
	key, err := linkkey.Key{Port: uint16(s.listening.port), Secret: next.secret}.Encode()
	if err != nil {
		s.stopIfIdle()
		return Offer{}, err
	}
	next.key = key
	s.offer.settle(linkstore.Extension{}, ErrCanceled)
	next.expiresAt = s.now().Add(s.keyLifetime)
	next.expiry = time.AfterFunc(s.keyLifetime, func() { s.expire(next) })
	s.offer = next
	return next.offered(), nil
}

// Await returns the extension that links with the key waiting now; ctx ending cancels that key, never a newer one.
func (s *Server) Await(ctx context.Context) (linkstore.Extension, error) {
	awaited := s.waitingOffer()
	if awaited == nil {
		return linkstore.Extension{}, ErrCanceled
	}
	select {
	case <-awaited.done:
	case <-ctx.Done():
		s.mu.Lock()
		s.cancel(awaited)
		s.mu.Unlock()
	}
	return awaited.extension, awaited.err
}

// Cancel ends the waiting key, if one is waiting.
func (s *Server) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancel(s.offer)
}

// Waiting returns the key waiting for an extension.
func (s *Server) Waiting() (Offer, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.offer.waiting() {
		return Offer{}, false
	}
	return s.offer.offered(), true
}

// Unlink forgets a linked extension and its captures, and closes its sessions with CloseNotLinked.
func (s *Server) Unlink(id string) error {
	return s.unlink(id, nil)
}

// Rename names the linked extension id anew; its sessions stay open.
func (s *Server) Rename(id, name string) error {
	return s.store.Rename(id, name)
}

// unlink is Unlink, except that it leaves kept, a session of id, open.
func (s *Server) unlink(id string, kept *connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Remove(id); err != nil {
		return err
	}
	s.held.ForgetSender(id)
	s.sessions.end(id, kept)
	s.stopIfIdle()
	return nil
}

// admit records c as a session of the extension with publicKey; s.mu orders the lookup against Unlink.
func (s *Server) admit(c *connection, publicKey []byte) (linkstore.Extension, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	extension, linked, err := s.store.Find(publicKey)
	if err != nil {
		return linkstore.Extension{}, err
	}
	if !linked {
		return linkstore.Extension{}, errNotLinked
	}
	s.sessions.add(extension.ID, c)
	return extension, nil
}

// release forgets c as a session of the extension with id once it ends.
func (s *Server) release(id string, c *connection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions.remove(id, c)
}

// stillLinked fails with errNotLinked once extension is unlinked or its key is relinked under another id.
func (s *Server) stillLinked(extension linkstore.Extension) error {
	current, linked, err := s.store.Find(extension.PublicKey)
	if err != nil {
		return err
	}
	if !linked || current.ID != extension.ID {
		return errNotLinked
	}
	return nil
}

func (s *Server) waitingOffer() *offer {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.offer.waiting() {
		return nil
	}
	return s.offer
}

// cancel ends o if it is still waiting. The caller holds s.mu.
func (s *Server) cancel(o *offer) {
	if o.settle(linkstore.Extension{}, ErrCanceled) {
		s.stopIfIdle()
	}
}

func (s *Server) expire(o *offer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.settle(linkstore.Extension{}, ErrExpired) {
		s.stopIfIdle()
	}
}

// spend records the extension that proved it holds o's secret, while o is still waiting.
func (s *Server) spend(o *offer, name string, publicKey []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !o.waiting() {
		return errNoKey
	}
	extension, err := s.store.Add(name, publicKey)
	if err != nil {
		return err
	}
	o.settle(extension, nil)
	return nil
}

// listen binds the kept port, replacing it only while no extension is linked. The caller holds s.mu.
func (s *Server) listen() error {
	if s.listening != nil {
		return nil
	}
	port, err := s.store.Port()
	if err != nil {
		return err
	}
	extensions, err := s.store.Extensions()
	if err != nil {
		return err
	}
	var listener net.Listener
	if port != 0 {
		listener, _ = bind(port)
	}
	if listener == nil && (port == 0 || len(extensions) == 0) {
		for range portAttempts {
			candidate := s.choosePort()
			if listener, err = bind(candidate); err != nil {
				continue
			}
			if err := s.store.SetPort(candidate); err != nil {
				listener.Close()
				return err
			}
			port = candidate
			break
		}
	}
	if listener == nil {
		s.unreachable = true
		return ErrUnavailable
	}
	server := &http.Server{Handler: s.handler(port), ReadHeaderTimeout: s.handshakeTimeout}
	go func() {
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			slog.Warn("serve link server", "err", err)
		}
	}()
	s.listening = &listening{port: port, listener: listener, server: server}
	s.unreachable = false
	return nil
}

func bind(port int) (net.Listener, error) {
	return net.Listen("tcp4", net.JoinHostPort(loopback, strconv.Itoa(port)))
}

// stopIfIdle stops listening once no key waits and no extension is linked. The caller holds s.mu.
func (s *Server) stopIfIdle() {
	if s.offer.waiting() {
		return
	}
	extensions, err := s.store.Extensions()
	if err != nil || len(extensions) > 0 {
		return
	}
	s.unreachable = false
	if s.listening == nil {
		return
	}
	if err := s.listening.stop(); err != nil {
		slog.Warn("stop link server", "err", err)
	}
	s.listening = nil
}

// handler refuses before the upgrade any origin but the extension's and any Host but loopback:port, blocking web pages and DNS rebinding.
func (s *Server) handler(port int) http.Handler {
	host := net.JoinHostPort(loopback, strconv.Itoa(port))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host || !linkproto.AllowedOrigin(r.Header.Get("Origin")) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		var exchange func(*connection) error
		switch r.URL.Path {
		case linkproto.LinkPath:
			exchange = s.link
		case linkproto.SessionPath:
			exchange = s.session
		default:
			http.NotFound(w, r)
			return
		}
		select {
		case s.slots <- struct{}{}:
		default:
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: linkproto.AllowedOrigins()})
		if err != nil {
			<-s.slots
			return
		}
		c := s.open(socket, s.slots)
		defer c.leave()
		c.end(exchange(c))
	})
}
