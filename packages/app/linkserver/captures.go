package linkserver

import (
	"errors"

	"github.com/dortanes/ravenpass/packages/app/captures"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkstore"
)

// Capture is a password submitted on a page at Origin, or with Card a card; Current is the old password a change form
// asked for.
type Capture struct {
	Origin   string
	Account  string
	Password string
	Current  string
	Card     *CardCapture
}

// CardCapture is a card typed on a page; Expiry is YYYY-MM and Network a credit-card-type name.
type CardCapture struct {
	Holder       string
	Number       string
	Expiry       string
	SecurityCode string
	Network      string
}

// SaveChoice is where a person saves a capture; an empty Target means a new credential.
type SaveChoice struct {
	Target  string
	Account string
	Name    string
}

// targetCredentials names the credentials of targets, in order.
func targetCredentials(targets []linkproto.SaveTarget) []string {
	credentials := make([]string, len(targets))
	for i, target := range targets {
		credentials[i] = target.Credential
	}
	return credentials
}

// offerCapture answers a capture request, holding the capture unless there is nothing to offer.
func (s *Server) offerCapture(extension linkstore.Extension, request linkproto.Request) (linkproto.Response, bool, error) {
	if !linkproto.ValidOrigin(request.Origin) {
		return linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin}, false, nil
	}
	return s.holdCapture(extension, request, Capture{Origin: request.Origin, Account: request.Account, Password: request.Password, Current: request.Current})
}

// offerCardCapture answers a card capture request from a secure page as offerCapture does.
func (s *Server) offerCardCapture(extension linkstore.Extension, request linkproto.Request) (linkproto.Response, bool, error) {
	if !linkproto.ValidOrigin(request.Origin) || !linkproto.SecureOrigin(request.Origin) {
		return linkproto.Response{ID: request.ID, Error: linkproto.ErrorInvalidOrigin}, false, nil
	}
	card := request.Card
	return s.holdCapture(extension, request, Capture{Origin: request.Origin, Card: &CardCapture{
		Holder: card.Holder, Number: card.Number, Expiry: card.Expiry, SecurityCode: card.SecurityCode, Network: card.Network,
	}})
}

// holdCapture answers request with where capture can be saved, holding it unless there is nothing to offer.
func (s *Server) holdCapture(extension linkstore.Extension, request linkproto.Request, capture Capture) (linkproto.Response, bool, error) {
	offer, err := s.vault.CaptureOffer(capture)
	if err != nil {
		return refusal(request, err)
	}
	if offer.State != linkproto.CaptureNone {
		offer.Pending, err = s.held.Keep(extension.ID, capture, targetCredentials(offer.Targets))
		if errors.Is(err, captures.ErrClosed) {
			return linkproto.Response{}, false, ErrUnavailable
		}
		if err != nil {
			return linkproto.Response{}, false, err
		}
	}
	return captureAnswer(request, offer), false, nil
}

// reviewCapture answers a review request, forgetting a capture with nothing to offer.
func (s *Server) reviewCapture(extension linkstore.Extension, request linkproto.Request) (linkproto.Response, bool, error) {
	capture, found := s.held.Find(extension.ID, request.Pending)
	if !found {
		return linkproto.Response{ID: request.ID, Error: linkproto.ErrorNotFound}, false, nil
	}
	offer, err := s.vault.CaptureOffer(capture)
	if err != nil {
		return refusal(request, err)
	}
	if offer.State == linkproto.CaptureNone {
		s.held.Forget(extension.ID, request.Pending)
	} else {
		s.held.Offer(extension.ID, request.Pending, targetCredentials(offer.Targets))
		offer.Pending = request.Pending
	}
	return captureAnswer(request, offer), false, nil
}

// captureAnswer answers request with offer. No target is sent as [], never null.
func captureAnswer(request linkproto.Request, offer linkproto.CaptureOffer) linkproto.Response {
	offer.Targets = append([]linkproto.SaveTarget{}, offer.Targets...)
	return linkproto.Response{ID: request.ID, Result: offer}
}

// saveCapture answers a save request, forgetting the capture once it is saved.
func (s *Server) saveCapture(extension linkstore.Extension, request linkproto.Request) (linkproto.Response, bool, error) {
	capture, err := s.held.Claim(extension.ID, request.Pending, request.Target)
	if err != nil {
		return refusal(request, ErrNotFound)
	}
	saved, err := s.vault.SaveCapture(capture, SaveChoice{Target: request.Target, Account: request.Account, Name: request.Name})
	s.held.Release(extension.ID, request.Pending, err == nil)
	if err != nil {
		return refusal(request, err)
	}
	return linkproto.Response{ID: request.ID, Result: saved}, false, nil
}
