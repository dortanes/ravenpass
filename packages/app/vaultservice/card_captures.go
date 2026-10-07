package vaultservice

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/dortanes/ravenpass/packages/vault"
)

// CardCapture is a card typed on a page; Expiry is YYYY-MM, and every value but Number may be empty.
type CardCapture struct {
	Holder       string
	Number       string
	Expiry       string
	SecurityCode string
	Network      vault.CardNetwork
}

// changes reports whether capture carries a value card holds otherwise.
func (capture CardCapture) changes(card vault.CardInput) bool {
	differs := func(captured, held string) bool { return captured != "" && captured != held }
	return differs(capture.Holder, card.Holder) || differs(capture.Expiry, card.Expiry) || differs(capture.SecurityCode, card.SecurityCode)
}

// update is card with the values capture carries.
func (capture CardCapture) update(card vault.CardInput) vault.CardInput {
	card.Holder = cmp.Or(capture.Holder, card.Holder)
	card.Expiry = cmp.Or(capture.Expiry, card.Expiry)
	card.SecurityCode = cmp.Or(capture.SecurityCode, card.SecurityCode)
	return card
}

// card is capture as a new card labelled label.
func (capture CardCapture) card(label string) vault.CardInput {
	return vault.CardInput{
		Label: label, Holder: capture.Holder, Number: capture.Number, Expiry: capture.Expiry,
		SecurityCode: capture.SecurityCode, Network: capture.Network,
	}
}

// storableCard reports whether capture makes a card the vault accepts under a valid label.
func storableCard(capture CardCapture) bool {
	input := capture.card("card")
	_, err := vault.PreviewNewItem(vault.NewItem{Card: &input})
	return err == nil
}

// CardCaptureOffer compares a typed card with the open vault's cards of the same last four digits, decrypting only
// those, without recording a use: nothing for a card held unchanged, an update of each card holding the number with
// other values, else a new card.
func (s *Service) CardCaptureOffer(capture CardCapture) (CaptureOffer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	offer, _, err := s.cardCaptureOffer(capture)
	return offer, err
}

// cardCaptureOffer is CardCaptureOffer with the groups of each target, for a caller that holds s.mu.
func (s *Service) cardCaptureOffer(capture CardCapture) (CaptureOffer, map[vault.ID][]vault.ID, error) {
	if s.session == nil {
		return CaptureOffer{}, nil, ErrNotReady
	}
	if !storableCard(capture) {
		return CaptureOffer{Nothing: true}, nil, nil
	}
	lastUsed, err := s.lastUses()
	if err != nil {
		return CaptureOffer{}, nil, err
	}
	entries, err := s.session.List()
	if err != nil {
		return CaptureOffer{}, nil, err
	}
	lastFour := vault.LastFour(capture.Number)
	var found []candidate
	groups := map[vault.ID][]vault.ID{}
	for _, entry := range entries {
		if entry.Kind != vault.KindCard || entry.Card.LastFour != lastFour {
			continue
		}
		card, err := s.session.ReadCard(entry.ID)
		if err != nil {
			return CaptureOffer{}, nil, err
		}
		if card.Number != capture.Number {
			continue
		}
		if !capture.changes(card.CardInput) {
			return CaptureOffer{Nothing: true}, nil, nil
		}
		found = append(found, candidate{
			target:   Target{ID: entry.ID, Label: entry.Label, Action: TargetUpdate, Tags: entry.Tags},
			lastUsed: lastUsed[entry.ID],
		})
		groups[entry.ID] = entry.Groups
	}
	slices.SortStableFunc(found, byHolderThenUse)
	found = found[:min(len(found), maxCaptureTargets)]
	offer := CaptureOffer{Targets: make([]Target, len(found))}
	for i, target := range found {
		offer.Targets[i] = target.target
	}
	if len(found) > 0 {
		offer.Suggested = found[0].target.ID
	}
	return offer, groups, nil
}

// SaveCardCapture rechecks the offer and saves the typed card as choice names under one hold of s.mu; a new card
// joins group. It reports whether it created a card.
func (s *Service) SaveCardCapture(capture CardCapture, choice CaptureChoice, group string) (bool, error) {
	created, err := s.saveCardCapture(capture, choice, group)
	if err != nil {
		return false, err
	}
	s.changes.Record()
	return created, nil
}

func (s *Service) saveCardCapture(capture CardCapture, choice CaptureChoice, group string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	offer, groups, err := s.cardCaptureOffer(capture)
	if err != nil {
		return false, err
	}
	if offer.Nothing {
		return false, vault.ErrNotFound
	}
	if choice.Target == (vault.ID{}) {
		input := capture.card(choice.Name)
		// The capture is storable, so a refusal is the name's.
		if _, err := vault.PreviewNewItem(vault.NewItem{Card: &input}); err != nil {
			return false, fmt.Errorf("%w: %w", ErrNameRefused, err)
		}
		joined, err := s.newItemGroups(group)
		if err != nil {
			return false, err
		}
		_, err = s.createCard(input, joined)
		return err == nil, err
	}
	if !slices.ContainsFunc(offer.Targets, func(target Target) bool { return target.ID == choice.Target }) {
		return false, vault.ErrNotFound
	}
	card, err := s.session.ReadCard(choice.Target)
	if err != nil {
		return false, err
	}
	return false, s.editCard(choice.Target, capture.update(card.CardInput), groups[choice.Target])
}
