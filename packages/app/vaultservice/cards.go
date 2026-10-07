package vaultservice

import (
	"cmp"
	"slices"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

// ReadSelectedCard returns the card ticket selects.
func (s *Service) ReadSelectedCard(ticket vault.Selection) (vault.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return vault.Card{}, ErrNotReady
	}
	return s.session.ReadSelectedCard(ticket)
}

// CreateCard saves a new card in groups and returns its ID.
func (s *Service) CreateCard(input vault.CardInput, groups []vault.ID) (vault.ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createCard(input, groups)
}

// createCard is CreateCard for a caller that holds s.mu.
func (s *Service) createCard(input vault.CardInput, groups []vault.ID) (vault.ID, error) {
	if s.session == nil {
		return vault.ID{}, ErrNotReady
	}
	pending, id, err := s.session.PrepareCreateCard(input, groups)
	if err != nil {
		return vault.ID{}, err
	}
	if err := s.commit(pending); err != nil {
		return vault.ID{}, err
	}
	return id, nil
}

// EditCard replaces a card's content and membership as a whole.
func (s *Service) EditCard(id vault.ID, input vault.CardInput, groups []vault.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.editCard(id, input, groups)
}

// editCard is EditCard for a caller that holds s.mu.
func (s *Service) editCard(id vault.ID, input vault.CardInput, groups []vault.ID) error {
	if s.session == nil {
		return ErrNotReady
	}
	pending, err := s.session.PrepareEditCard(id, input, groups)
	if err != nil {
		return err
	}
	return s.commit(pending)
}

// IdentityAddresses lists what a card's billing address can link to.
func (s *Service) IdentityAddresses() ([]vault.IdentityAddresses, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return nil, ErrNotReady
	}
	return s.session.IdentityAddresses()
}

// FillableCards lists the cards a checkout can take from the index alone: unexpired ones first, each part by most
// recent use, then by label.
func (s *Service) FillableCards() ([]vault.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return nil, ErrNotReady
	}
	lastUsed, err := s.lastUses()
	if err != nil {
		return nil, err
	}
	entries, err := s.session.List()
	if err != nil {
		return nil, err
	}
	today := s.now().Format(time.DateOnly)
	cards := slices.DeleteFunc(entries, func(entry vault.Entry) bool { return entry.Kind != vault.KindCard })
	slices.SortStableFunc(cards, func(a, b vault.Entry) int {
		return cmp.Or(setFirst(!expired(a, today), !expired(b, today)), byUseThenLabel(lastUsed[a.ID], lastUsed[b.ID], a.Label, b.Label))
	})
	return cards[:min(len(cards), maxSuggestions)], nil
}

// expired reports a card whose expiry month ended before today, both written YYYY-MM-DD.
func expired(entry vault.Entry, today string) bool {
	return entry.ExpiresOn != "" && entry.ExpiresOn < today
}

// FillCard reads card id for a checkout, its link resolved, and records the use.
func (s *Service) FillCard(id vault.ID) (vault.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return vault.Card{}, ErrNotReady
	}
	card, err := s.session.ReadCard(id)
	if err != nil {
		return vault.Card{}, err
	}
	if err := s.recordUse(id); err != nil {
		return vault.Card{}, err
	}
	return card, nil
}
