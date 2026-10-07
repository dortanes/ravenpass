package linkserver

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkstore"
)

var (
	errNoKey        = errors.New("no connection key is waiting")
	errNotLinked    = errors.New("extension is not linked")
	errDisconnected = errors.New("connection ended")
)

// connection is one WebSocket carrying whole Noise messages; deadline closes it with CloseIdle. Only its exchange's
// goroutine moves or leaves its slot.
type connection struct {
	socket   *websocket.Conn
	ctx      context.Context
	deadline *time.Timer
	// slot is the pool whose place the connection holds.
	slot chan struct{}
}

// open takes over the place the handler took in slot.
func (s *Server) open(socket *websocket.Conn, slot chan struct{}) *connection {
	// read bounds messages and closes with CloseMalformed, not the library's StatusMessageTooBig.
	socket.SetReadLimit(-1)
	c := &connection{socket: socket, ctx: s.connections, slot: slot}
	c.deadline = time.AfterFunc(s.handshakeTimeout, func() { c.stop(linkproto.CloseIdle) })
	return c
}

// moveTo gives the connection's place back for one in pool, while pool has room; otherwise it keeps its place.
func (c *connection) moveTo(pool chan struct{}) {
	if c.slot == pool {
		return
	}
	select {
	case pool <- struct{}{}:
		<-c.slot
		c.slot = pool
	default:
	}
}

func (c *connection) leave() {
	<-c.slot
}

// stop closes the connection with code from outside its exchange; cancelling c.ctx sends no close frame.
func (c *connection) stop(code linkproto.CloseCode) {
	_ = c.socket.Close(websocket.StatusCode(code), "")
}

// read returns the next message, which must be one binary message of at most MaxMessageBytes.
func (c *connection) read() ([]byte, error) {
	kind, reader, err := c.socket.Reader(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDisconnected, err)
	}
	message, err := io.ReadAll(io.LimitReader(reader, linkproto.MaxMessageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDisconnected, err)
	}
	if kind != websocket.MessageBinary || len(message) > linkproto.MaxMessageBytes {
		return nil, linkproto.ErrMalformed
	}
	return message, nil
}

func (c *connection) write(message []byte) error {
	if err := c.socket.Write(c.ctx, websocket.MessageBinary, message); err != nil {
		return fmt.Errorf("%w: %w", errDisconnected, err)
	}
	return nil
}

// send seals each plaintext with transport and writes it, in order.
func (c *connection) send(transport *linkproto.Transport, plaintexts [][]byte) error {
	for _, plaintext := range plaintexts {
		sealed, err := transport.Seal(plaintext)
		if err != nil {
			return err
		}
		if err := c.write(sealed); err != nil {
			return err
		}
	}
	return nil
}

// reply sends response in as many messages as it needs.
func (c *connection) reply(transport *linkproto.Transport, response linkproto.Response) error {
	frames, err := linkproto.Frames(response)
	if err != nil {
		return err
	}
	return c.send(transport, frames)
}

// receiveEmpty reads the peer's next handshake message, which carries no payload.
func (c *connection) receiveEmpty(handshake *linkproto.Handshake) error {
	payload, err := c.receive(handshake)
	if err != nil {
		return err
	}
	if len(payload) != 0 {
		return linkproto.ErrMalformed
	}
	return nil
}

func (c *connection) receive(handshake *linkproto.Handshake) ([]byte, error) {
	message, err := c.read()
	if err != nil {
		return nil, err
	}
	return handshake.Read(message)
}

// respond writes this side's next handshake message carrying payload.
func (c *connection) respond(handshake *linkproto.Handshake, payload []byte) error {
	message, err := handshake.Write(payload)
	if err != nil {
		return err
	}
	return c.write(message)
}

// end closes the connection with the close code for err.
func (c *connection) end(err error) {
	c.deadline.Stop()
	if errors.Is(err, errDisconnected) {
		_ = c.socket.CloseNow()
		return
	}
	_ = c.socket.Close(closeCode(err), "")
}

func closeCode(err error) websocket.StatusCode {
	switch {
	case err == nil:
		return websocket.StatusCode(linkproto.CloseFinished)
	case errors.Is(err, linkproto.ErrMalformed):
		return websocket.StatusCode(linkproto.CloseMalformed)
	case errors.Is(err, errNotLinked):
		return websocket.StatusCode(linkproto.CloseNotLinked)
	case errors.Is(err, linkproto.ErrAuthentication):
		return websocket.StatusCode(linkproto.CloseUnauthenticated)
	case errors.Is(err, errNoKey):
		return websocket.StatusCode(linkproto.CloseNoKey)
	default:
		return websocket.StatusInternalError
	}
}

// link runs Noise_XXpsk3 with the waiting key's secret and records the extension; a failed handshake keeps the key.
func (s *Server) link(c *connection) error {
	awaited := s.waitingOffer()
	if awaited == nil {
		return errNoKey
	}
	desktop, err := s.store.DesktopKey()
	if err != nil {
		return err
	}
	handshake, err := linkproto.LinkResponder(desktop, awaited.secret[:], rand.Reader)
	if err != nil {
		return err
	}
	if err := c.receiveEmpty(handshake); err != nil {
		return err
	}
	if err := c.respond(handshake, nil); err != nil {
		return err
	}
	payload, err := c.receive(handshake)
	if err != nil {
		return err
	}
	request, err := linkproto.ParseLinkRequest(payload)
	if err != nil {
		return err
	}
	if err := s.spend(awaited, request.Name, handshake.PeerStatic()); err != nil {
		return err
	}
	message, err := linkproto.LinkedMessage(s.greeting())
	if err != nil {
		return err
	}
	linked, err := handshake.Transport().Seal(message)
	if err != nil {
		return err
	}
	return c.write(linked)
}

// greeting is what the desktop app reports at link and session start.
func (s *Server) greeting() linkproto.Greeting {
	return linkproto.Greeting{Language: s.settings.Language(), SignIn: s.settings.SignInStyle()}
}

// session runs Noise_IK for a linked static key and answers requests; a request after unlinking ends it unanswered.
func (s *Server) session(c *connection) error {
	desktop, err := s.store.DesktopKey()
	if err != nil {
		return err
	}
	handshake, err := linkproto.SessionResponder(desktop, rand.Reader)
	if err != nil {
		return err
	}
	if err := c.receiveEmpty(handshake); err != nil {
		return err
	}
	extension, err := s.admit(c, handshake.PeerStatic())
	if err != nil {
		return err
	}
	defer s.release(extension.ID, c)
	greeting, err := linkproto.SessionGreeting(s.greeting())
	if err != nil {
		return err
	}
	if err := c.respond(handshake, greeting); err != nil {
		return err
	}
	transport := handshake.Transport()
	for {
		c.deadline.Reset(s.idleTimeout)
		message, err := c.read()
		if err != nil {
			return err
		}
		plaintext, err := transport.Open(message)
		if err != nil {
			return linkproto.ErrMalformed
		}
		request, err := linkproto.ParseRequest(plaintext)
		// A capture request carries a typed password, which only its parsed request keeps.
		clear(plaintext)
		if err != nil {
			return err
		}
		if err := s.stillLinked(extension); err != nil {
			return err
		}
		finished, err := s.serve(c, transport, extension, request)
		if err != nil || finished {
			return err
		}
	}
}

// serve answers one request and reports whether it ends the session.
func (s *Server) serve(c *connection, transport *linkproto.Transport, extension linkstore.Extension, request linkproto.Request) (bool, error) {
	switch request.Type {
	case linkproto.RequestShare:
		return false, s.share(c, transport, request)
	case linkproto.RequestFill:
		return false, s.fill(c, transport, extension, request)
	case linkproto.RequestCardFill:
		return false, s.fillCard(c, transport, request)
	case linkproto.RequestCode:
		return false, s.code(c, transport, request)
	case linkproto.RequestPasskeyCreate:
		return false, s.createPasskey(c, transport, request)
	case linkproto.RequestPasskeySign:
		return false, s.signPasskey(c, transport, request)
	}
	response, finished, err := s.answer(c, extension, request)
	if err != nil {
		return false, err
	}
	return finished, c.reply(transport, response)
}

// whileVerifying runs serve with the idle limit paused, sending progress each interval; a failed write cancels serve.
// The connection moves to the verifying pool for the rest of its life, so a wait of minutes holds no connection slot.
func whileVerifying[T any](s *Server, c *connection, transport *linkproto.Transport, request linkproto.Request, serve func(ctx context.Context, asked func(linkproto.Progress)) T) (T, error) {
	c.moveTo(s.verifying)
	c.deadline.Stop()
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()
	asked := make(chan linkproto.Progress, 1)
	done := make(chan T, 1)
	go func() {
		done <- serve(ctx, func(progress linkproto.Progress) {
			select {
			case asked <- progress:
			case <-ctx.Done():
			}
		})
	}()
	ticker := time.NewTicker(s.progressInterval)
	ticker.Stop()
	defer ticker.Stop()
	var progress linkproto.Progress
	for {
		select {
		case progress = <-asked:
			ticker.Reset(s.progressInterval)
		case <-ticker.C:
		case result := <-done:
			return result, nil
		}
		if err := c.reply(transport, linkproto.Response{ID: request.ID, Progress: progress}); err != nil {
			cancel()
			return <-done, err
		}
	}
}

// verifiedOutcome is how a request that may wait for the person ended.
type verifiedOutcome[T any] struct {
	result T
	err    error
}

// answerVerified runs serve while the person may verify request, then sends its result or refusal.
func answerVerified[T any](s *Server, c *connection, transport *linkproto.Transport, request linkproto.Request, serve func(ctx context.Context, asked func(linkproto.Progress)) (T, error)) error {
	outcome, err := whileVerifying(s, c, transport, request, func(ctx context.Context, asked func(linkproto.Progress)) verifiedOutcome[T] {
		result, err := serve(ctx, asked)
		return verifiedOutcome[T]{result: result, err: err}
	})
	if err != nil {
		return err
	}
	if outcome.err != nil {
		response, _, err := refusal(request, outcome.err)
		if err != nil {
			return err
		}
		return c.reply(transport, response)
	}
	return c.reply(transport, linkproto.Response{ID: request.ID, Result: outcome.result})
}

// fill answers a fill request once the vault releases the credential.
func (s *Server) fill(c *connection, transport *linkproto.Transport, extension linkstore.Extension, request linkproto.Request) error {
	if !linkproto.ValidOrigin(request.Origin) {
		return c.reply(transport, linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin})
	}
	return answerVerified(s, c, transport, request, func(ctx context.Context, asked func(linkproto.Progress)) (linkproto.Fill, error) {
		return s.vault.Fill(ctx, extension.ID, request.Credential, request.Origin, asked)
	})
}

// fillCard answers a card fill request from a secure page once the vault releases the card.
func (s *Server) fillCard(c *connection, transport *linkproto.Transport, request linkproto.Request) error {
	if !linkproto.ValidOrigin(request.Origin) || !linkproto.SecureOrigin(request.Origin) {
		return c.reply(transport, linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin})
	}
	return answerVerified(s, c, transport, request, func(ctx context.Context, asked func(linkproto.Progress)) (linkproto.CardFill, error) {
		return s.vault.FillCard(ctx, request.Card.Card, request.Origin, asked)
	})
}

// code answers a code request once the vault releases the credential's current code.
func (s *Server) code(c *connection, transport *linkproto.Transport, request linkproto.Request) error {
	if !linkproto.ValidOrigin(request.Origin) {
		return c.reply(transport, linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin})
	}
	return answerVerified(s, c, transport, request, func(ctx context.Context, asked func(linkproto.Progress)) (linkproto.OneTimeCode, error) {
		return s.vault.OneTimeCode(ctx, request.Credential, request.Origin, asked)
	})
}

// shared is how a share ended: the file and its content, or the reason it was not released.
type shared struct {
	file    linkproto.SharedFile
	content []byte
	err     error
}

// share answers a share request once the person verifies it.
func (s *Server) share(c *connection, transport *linkproto.Transport, request linkproto.Request) error {
	if !linkproto.ValidOrigin(request.Origin) {
		return c.reply(transport, linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin})
	}
	result, err := whileVerifying(s, c, transport, request, func(ctx context.Context, asked func(linkproto.Progress)) shared {
		file, content, err := s.vault.Share(ctx, request.Identity, request.File, request.Origin, asked)
		return shared{file: file, content: content, err: err}
	})
	if err != nil {
		clear(result.content)
		return err
	}
	return release(c, transport, request, result)
}

// release sends a shared file after its head, or the refusal that ended the share.
func release(c *connection, transport *linkproto.Transport, request linkproto.Request, result shared) error {
	if result.err != nil {
		response, _, err := refusal(request, result.err)
		if err != nil {
			return err
		}
		return c.reply(transport, response)
	}
	defer clear(result.content)
	frames, err := linkproto.FileFrames(request.ID, result.file, result.content)
	if err != nil {
		return err
	}
	return c.send(transport, frames)
}

// answer serves one request on c and reports whether it ends the session.
func (s *Server) answer(c *connection, extension linkstore.Extension, request linkproto.Request) (linkproto.Response, bool, error) {
	switch request.Type {
	case linkproto.RequestStatus:
		status := linkproto.VaultLocked
		if s.vault.Unlocked() {
			status = linkproto.VaultUnlocked
		}
		return linkproto.Response{ID: request.ID, Result: linkproto.Status{Vault: status}}, false, nil
	case linkproto.RequestUnlink:
		// c stays open to confirm; the desktop app may have unlinked the extension already.
		if err := s.unlink(extension.ID, c); err != nil && !errors.Is(err, linkstore.ErrNotFound) {
			return linkproto.Response{}, false, err
		}
		return linkproto.Response{ID: request.ID, Result: struct{}{}}, true, nil
	case linkproto.RequestSuggest:
		if !linkproto.ValidOrigin(request.Origin) {
			return linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin}, false, nil
		}
		purpose, known := request.SuggestPurpose()
		if !known {
			return linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidPurpose}, false, nil
		}
		credentials, err := s.vault.Suggest(request.Origin, purpose)
		if err != nil {
			return refusal(request, err)
		}
		// No suggestion is sent as [], never null.
		return linkproto.Response{ID: request.ID, Result: linkproto.Suggestions{Credentials: append([]linkproto.Suggestion{}, credentials...)}}, false, nil
	case linkproto.RequestAddWebsite:
		// Any network attacker can serve a plain http origin.
		if !linkproto.ValidOrigin(request.Origin) || !strings.HasPrefix(request.Origin, "https://") {
			return linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin}, false, nil
		}
		if err := s.vault.AddWebsite(request.Credential, request.Origin); err != nil {
			return refusal(request, err)
		}
		return linkproto.Response{ID: request.ID, Result: struct{}{}}, false, nil
	case linkproto.RequestIcon:
		if !linkproto.ValidSite(request.Site) {
			return linkproto.Response{}, false, linkproto.ErrMalformed
		}
		icon, err := s.vault.SiteIcon(request.Site)
		if err != nil {
			return refusal(request, err)
		}
		return linkproto.Response{ID: request.ID, Result: icon}, false, nil
	case linkproto.RequestUnlock:
		s.vault.ShowUnlock()
		return linkproto.Response{ID: request.ID, Result: struct{}{}}, false, nil
	case linkproto.RequestIdentities:
		if !linkproto.ValidOrigin(request.Origin) {
			return linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin}, false, nil
		}
		identities, err := s.vault.Identities()
		if err != nil {
			return refusal(request, err)
		}
		// No identity is sent as [], never null.
		return linkproto.Response{ID: request.ID, Result: linkproto.Identities{Identities: append([]linkproto.Identity{}, identities...)}}, false, nil
	case linkproto.RequestCards:
		if !linkproto.ValidOrigin(request.Origin) || !linkproto.SecureOrigin(request.Origin) {
			return linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin}, false, nil
		}
		cards, err := s.vault.Cards()
		if err != nil {
			return refusal(request, err)
		}
		// No card is sent as [], never null.
		return linkproto.Response{ID: request.ID, Result: linkproto.Cards{Cards: append([]linkproto.CardOption{}, cards...)}}, false, nil
	case linkproto.RequestCapture:
		return s.offerCapture(extension, request)
	case linkproto.RequestCardCapture:
		return s.offerCardCapture(extension, request)
	case linkproto.RequestReview:
		return s.reviewCapture(extension, request)
	case linkproto.RequestSave:
		return s.saveCapture(extension, request)
	case linkproto.RequestDiscard:
		s.held.Forget(extension.ID, request.Pending)
		return linkproto.Response{ID: request.ID, Result: struct{}{}}, false, nil
	case linkproto.RequestPasskeys:
		return s.passkeyOptions(request)
	default:
		return linkproto.Response{ID: request.ID, Error: linkproto.ErrorUnknownRequest}, false, nil
	}
}

// refusal answers request with the error code of a Vault refusal; any other error ends the session.
func refusal(request linkproto.Request, err error) (linkproto.Response, bool, error) {
	var code string
	switch {
	case errors.Is(err, ErrLocked):
		code = linkproto.ErrorLocked
	case errors.Is(err, ErrNotFound):
		code = linkproto.ErrorNotFound
	case errors.Is(err, ErrNoMatch):
		code = linkproto.ErrorNoMatch
	case errors.Is(err, ErrNoCode):
		code = linkproto.ErrorNoCode
	case errors.Is(err, ErrDeclined):
		code = linkproto.ErrorDeclined
	case errors.Is(err, ErrUnverifiable):
		code = linkproto.ErrorUnverifiable
	case errors.Is(err, ErrInvalidAccount):
		code = linkproto.ErrorInvalidAccount
	case errors.Is(err, ErrInvalidName):
		code = linkproto.ErrorInvalidName
	case errors.Is(err, ErrInvalidOrigin):
		code = linkproto.ErrorInvalidOrigin
	case errors.Is(err, ErrInvalidRelyingParty):
		code = linkproto.ErrorInvalidRelyingParty
	case errors.Is(err, ErrExcluded):
		code = linkproto.ErrorExcluded
	case errors.Is(err, ErrPasskeysFull):
		code = linkproto.ErrorFull
	default:
		return linkproto.Response{}, false, err
	}
	return linkproto.Response{ID: request.ID, Error: code}, false, nil
}
