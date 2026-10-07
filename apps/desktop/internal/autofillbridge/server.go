// Package autofillbridge answers Ravenpass's AutoFill extension over a Unix socket in the App Group container.
package autofillbridge

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dortanes/ravenpass/packages/app/api"
	"github.com/dortanes/ravenpass/packages/app/autofill"
	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/app/verification"
	"github.com/dortanes/ravenpass/packages/vault"
)

const (
	// socketName is short: a socket path fits in 104 bytes and the container takes 66 for a five-letter user name.
	socketName = "fill"

	maxConnections  = 4
	exchangeTimeout = 5 * time.Second
	unlockWait      = 5 * time.Minute
	acceptBackoff   = 100 * time.Millisecond
)

var (
	// ErrUnsigned reports a build whose code signature the extension does not admit.
	ErrUnsigned = errors.New("this build is not signed for the AutoFill extension")
	// ErrInUse reports another process already answering at the socket.
	ErrInUse = errors.New("another process answers the AutoFill extension")
)

// Vault is the autofill service the extension is answered from.
type Vault interface {
	autofill.Finder
	Open() bool
	Fill(id string, r autofill.Requester) (autofill.Login, error)
	OneTimeCode(id string, r autofill.Requester) (autofill.Code, error)
	AddSite(id string, r autofill.Requester) error
	Passkeys(rpID string, allowed [][]byte) ([]autofill.PasskeyChoice, error)
	HeldPasskey(rpID, id string, credentialID []byte) (autofill.PasskeyChoice, error)
	CheckExclusions(rpID string, exclude [][]byte) error
	SignPasskey(signIn autofill.PasskeySignIn) (autofill.PasskeyAssertion, error)
	CreatePasskey(creation autofill.PasskeyCreation) (autofill.CreatedPasskey, error)
}

// Page is what the extension's page reads besides the vault.
type Page interface {
	SiteIcon(site string) (api.SiteIcon, error)
	GetLanguage() (api.LanguageSettings, error)
	GetAppearance() (api.Appearance, error)
}

// Unlocks posts unlock requests to the confirmation queue and waits for them to end.
type Unlocks interface {
	PostUnlock(requester confirmation.Requester) string
	AwaitEnd(ctx context.Context, id string) error
}

// Verifier asks the owner to verify a passkey's creation or use.
type Verifier interface {
	Verify(ctx context.Context, reason confirmation.Reason, asked func(verification.Method)) error
}

// System is what the bridge asks of macOS.
type System interface {
	// Signed reports whether this app carries a code signature the extension admits.
	Signed() bool
	// Admits reports whether the process at the other end of conn is the extension.
	Admits(conn net.Conn) bool
	// Container is the folder of the App Group the app and the extension share.
	Container() (string, error)
	// AnnounceListening tells a waiting extension that the socket accepts connections.
	AnnounceListening()
}

// Server answers the extension at a socket in the App Group container.
type Server struct {
	vault    Vault
	page     Page
	unlocks  Unlocks
	verifier Verifier
	system   System
	slots    chan struct{}

	mu        sync.Mutex
	listening *listening
}

// listening is one run of the server, from Start to Close.
type listening struct {
	listener net.Listener
	stopped  chan struct{}
	served   sync.WaitGroup
}

// New fails when any dependency is nil.
func New(vault Vault, page Page, unlocks Unlocks, verifier Verifier, system System) (*Server, error) {
	if vault == nil || page == nil || unlocks == nil || verifier == nil || system == nil {
		return nil, errors.New("autofill service, page reads, unlock requests, owner verification and the system are required")
	}
	return &Server{
		vault: vault, page: page, unlocks: unlocks, verifier: verifier, system: system,
		slots: make(chan struct{}, maxConnections),
	}, nil
}

// Start listens in the App Group container, replacing a socket nobody answers at; ErrUnsigned touches nothing.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listening != nil {
		return nil
	}
	if !s.system.Signed() {
		return ErrUnsigned
	}
	container, err := s.system.Container()
	if err != nil {
		return err
	}
	listener, err := listen(filepath.Join(container, socketName))
	if err != nil {
		return err
	}
	run := &listening{listener: listener, stopped: make(chan struct{})}
	s.listening = run
	run.served.Add(1)
	go s.accept(run)
	s.system.AnnounceListening()
	return nil
}

// Close stops listening, ends every wait for the owner, and returns once every connection is closed.
func (s *Server) Close() error {
	s.mu.Lock()
	run := s.listening
	s.listening = nil
	s.mu.Unlock()
	if run == nil {
		return nil
	}
	close(run.stopped)
	err := run.listener.Close()
	run.served.Wait()
	return err
}

func listen(path string) (net.Listener, error) {
	if conn, err := net.DialTimeout("unix", path, exchangeTimeout); err == nil {
		_ = conn.Close()
		return nil, ErrInUse
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return net.Listen("unix", path)
}

// accept serves each connection while fewer than maxConnections are served, and closes the rest.
func (s *Server) accept(run *listening) {
	defer run.served.Done()
	for {
		conn, err := run.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			// Accept fails for a while when the process runs out of file descriptors.
			select {
			case <-run.stopped:
				return
			case <-time.After(acceptBackoff):
			}
			continue
		}
		select {
		case s.slots <- struct{}{}:
			run.served.Add(1)
			go func() {
				defer run.served.Done()
				defer func() { <-s.slots }()
				s.serve(conn, run.stopped)
			}()
		default:
			_ = conn.Close()
		}
	}
}

// serve closes conn without an answer for a peer that is not the extension or a malformed request.
func (s *Server) serve(conn net.Conn, stopped <-chan struct{}) {
	defer conn.Close()
	if !s.system.Admits(conn) {
		return
	}
	if conn.SetReadDeadline(time.Now().Add(exchangeTimeout)) != nil {
		return
	}
	var asked request
	if readMessage(conn, maxRequestBytes, &asked) != nil {
		return
	}
	if wait := waitFor(asked); wait > 0 {
		if conn.SetWriteDeadline(time.Now().Add(exchangeTimeout)) != nil ||
			writeMessage(conn, maxAnswerBytes, noticeOf(wait)) != nil {
			return
		}
	}
	ctx, stop := watch(conn, stopped)
	reply := s.answer(ctx, asked)
	stop()
	if conn.SetWriteDeadline(time.Now().Add(exchangeTimeout)) != nil {
		return
	}
	if errors.Is(writeMessage(conn, maxAnswerBytes, reply), errFrame) {
		_ = writeMessage(conn, maxAnswerBytes, answer{Error: refusedFailed})
	}
}

// waitFor is the longest the answer to asked may wait for the owner, zero for one that does not.
func waitFor(asked request) time.Duration {
	switch asked.Op {
	case opUnlock:
		return unlockWait
	case opPasskeySign, opPasskeyCreate:
		if asked.Verification.AsksOwner() {
			return verification.Timeout
		}
	}
	return 0
}

// watch ends ctx once stopped closes or the peer, which sends nothing after its request, hangs up.
func watch(conn net.Conn, stopped <-chan struct{}) (ctx context.Context, stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	// A connection that refuses a deadline is closed, and the read below then ends the context.
	_ = conn.SetReadDeadline(time.Time{})
	read := make(chan struct{})
	go func() {
		defer close(read)
		defer cancel()
		var probe [1]byte
		_, _ = conn.Read(probe[:])
	}()
	go func() {
		select {
		case <-stopped:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() {
		cancel()
		_ = conn.SetReadDeadline(time.Now())
		<-read
	}
}

func (s *Server) answer(ctx context.Context, asked request) answer {
	switch asked.Op {
	case opStatus:
		return answer{Open: s.vault.Open()}
	case opUnlock:
		return answer{Open: s.unlock(ctx)}
	case opPasskeys:
		return s.passkeys(asked)
	case opPasskeySign:
		return s.signPasskey(ctx, asked)
	case opPasskeyCreate:
		return s.createPasskey(ctx, asked)
	case opIcon:
		icon, err := s.page.SiteIcon(asked.Site)
		if err != nil {
			return refusal(err)
		}
		return answer{Image: icon.Image, Tint: icon.Tint}
	case opLanguage:
		settings, err := s.page.GetLanguage()
		if err != nil {
			return refusal(err)
		}
		return answer{Languages: settings.Languages, Language: settings.Language, Chosen: settings.Chosen}
	case opAppearance:
		appearance, err := s.page.GetAppearance()
		if err != nil {
			return refusal(err)
		}
		return answer{Appearance: appearance.Appearance}
	}
	requesters, ok := requestersOf(asked.Services)
	if !ok {
		return answer{Error: refusedInvalid}
	}
	switch asked.Op {
	case opSearch:
		return s.search(requesters, asked.Query, asked.Codes)
	case opPassword:
		login, err := release(requesters, func(r autofill.Requester) (autofill.Login, error) { return s.vault.Fill(asked.ID, r) })
		if err != nil {
			return refusal(err)
		}
		user := login.Login
		if user == "" {
			user = login.Email
		}
		return answer{User: user, Password: login.Password}
	case opCode:
		code, err := release(requesters, func(r autofill.Requester) (autofill.Code, error) { return s.vault.OneTimeCode(asked.ID, r) })
		if err != nil {
			return refusal(err)
		}
		return answer{Code: code.Code}
	case opAddSite:
		if len(requesters) == 0 {
			return answer{Error: refusedInvalid}
		}
		if err := s.vault.AddSite(asked.ID, requesters[0]); err != nil {
			return refusal(err)
		}
		return answer{}
	default:
		return answer{Error: refusedInvalid}
	}
}

func (s *Server) search(requesters []autofill.Requester, query string, codes bool) answer {
	listing, err := autofill.List(s.vault, requesters, query, codes)
	if err != nil {
		return refusal(err)
	}
	listed := make([]suggestion, len(listing.Results))
	for i, result := range listing.Results {
		listed[i] = suggestion{
			ID: result.ID, Label: result.Label, Account: result.Account, Site: result.Site, Matches: result.Matches, Tags: result.Tags,
		}
	}
	reply := answer{Scope: listing.Scope, Suggestions: listed}
	if len(requesters) > 0 {
		reply.Site = vaultservice.PageSite(requesters[0].Origin)
	}
	return reply
}

// release asks as each requester in turn until one is not refused with autofill.ErrNoMatch.
func release[T any](requesters []autofill.Requester, ask func(autofill.Requester) (T, error)) (T, error) {
	for _, requester := range requesters {
		value, err := ask(requester)
		if !errors.Is(err, autofill.ErrNoMatch) {
			return value, err
		}
	}
	var none T
	return none, autofill.ErrNoMatch
}

// unlock reports whether the vault is open when its request, unlockWait or ctx ends; a locked vault queues only this request.
func (s *Server) unlock(ctx context.Context) bool {
	if s.vault.Open() {
		return true
	}
	id := s.unlocks.PostUnlock(confirmation.RequesterAutofill)
	waiting, cancel := context.WithTimeout(ctx, unlockWait)
	defer cancel()
	if err := s.unlocks.AwaitEnd(waiting, id); err != nil && ctx.Err() != nil {
		return false
	}
	return s.vault.Open()
}

// requestersOf are the web origins the services name, in order and each once.
func requestersOf(services []service) ([]autofill.Requester, bool) {
	if len(services) > maxServices {
		return nil, false
	}
	var requesters []autofill.Requester
	for _, named := range services {
		origin := originOf(named)
		if origin != "" && !slices.ContainsFunc(requesters, func(held autofill.Requester) bool { return held.Origin == origin }) {
			requesters = append(requesters, autofill.Requester{Origin: origin})
		}
	}
	return requesters, true
}

// originOf is an http(s) URL's origin, or the https origin of a domain name without a port; empty for anything else.
func originOf(named service) string {
	switch named.Kind {
	case serviceURL:
		origin, _ := vault.OriginOf(named.Value)
		return origin
	case serviceDomain:
		origin, ok := vault.ParseOrigin("https://" + named.Value)
		if !ok || origin != "https://"+named.Value || strings.Contains(named.Value, ":") {
			return ""
		}
		return origin
	default:
		return ""
	}
}

func refusal(err error) answer {
	switch {
	case errors.Is(err, autofill.ErrLocked):
		return answer{Error: refusedLocked}
	case errors.Is(err, autofill.ErrNotFound):
		return answer{Error: refusedNotFound}
	case errors.Is(err, autofill.ErrNoMatch):
		return answer{Error: refusedNoMatch}
	case errors.Is(err, autofill.ErrNoCode):
		return answer{Error: refusedNoCode}
	case errors.Is(err, autofill.ErrInvalidRelyingParty), errors.Is(err, autofill.ErrInvalidRequest):
		return answer{Error: refusedInvalid}
	case errors.Is(err, autofill.ErrPasskeyExcluded):
		return answer{Error: refusedExcluded}
	case errors.Is(err, autofill.ErrUnsupportedAlgorithm):
		return answer{Error: refusedUnsupported}
	case errors.Is(err, verification.ErrDeclined):
		return answer{Error: refusedDeclined}
	case errors.Is(err, verification.ErrUnverifiable):
		return answer{Error: refusedUnverifiable}
	default:
		return answer{Error: refusedFailed}
	}
}
