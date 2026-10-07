package vaultservice

import (
	"cmp"
	"errors"
	"slices"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

// maxSuggestions bounds the credentials one requester is offered and one search lists.
const maxSuggestions = 50

// ErrNoMatch reports a credential that no longer matches the requester.
var ErrNoMatch = errors.New("credential does not match the requester")

// Purpose is what the field credentials are suggested for asks of them.
type Purpose uint8

const (
	// PurposeSignIn suggests every matching credential, to sign in with.
	PurposeSignIn Purpose = iota
	// PurposeCode suggests only the matching credentials that hold a one-time code setup.
	PurposeCode
)

// Suggestion is a matching credential; Site is empty for an app match, Code zero for PurposeSignIn.
type Suggestion struct {
	ID      vault.ID
	Label   string
	Account string
	Site    string
	Exact   bool
	Code    vault.CodeFace
	// Tags tell two accounts on one site apart.
	Tags []string
}

// Fill is what a credential gives a sign-in form.
type Fill struct {
	Login    string
	Email    string
	Password string
}

// Suggestions lists credentials matching requester for purpose, exact matches first.
func (s *Service) Suggestions(requester Requester, purpose Purpose) ([]Suggestion, error) {
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
	var suggestions []Suggestion
	for _, entry := range entries {
		var code vault.CodeFace
		if purpose == PurposeCode {
			code = entry.Code
		}
		if entry.Kind != vault.KindCredential || purpose == PurposeCode && code.Digits == 0 {
			continue
		}
		site, match := requester.match(entry)
		if match == vault.MatchNone {
			continue
		}
		if requester.checksScheme() {
			// A credential that cannot be read is left out rather than emptying every suggestion.
			credential, err := s.session.ReadCredential(entry.ID)
			if err != nil || !requester.admits(credential.CredentialInput) {
				continue
			}
		}
		suggestions = append(suggestions, Suggestion{ID: entry.ID, Label: entry.Label, Account: entryAccount(entry), Site: site, Exact: match == vault.MatchExact, Code: code, Tags: entry.Tags})
	}
	slices.SortStableFunc(suggestions, func(a, b Suggestion) int {
		return cmp.Or(setFirst(a.Exact, b.Exact), byUseThenLabel(lastUsed[a.ID], lastUsed[b.ID], a.Label, b.Label))
	})
	return suggestions[:min(len(suggestions), maxSuggestions)], nil
}

// entryAccount is a credential's login, else its email.
func entryAccount(entry vault.Entry) string {
	if entry.Detail != "" {
		return entry.Detail
	}
	return entry.Email
}

// matchedSite is the first of sites that matches origin as closely as any of them does.
func matchedSite(sites []string, origin string) (string, vault.Match) {
	site, best := "", vault.MatchNone
	for _, candidate := range sites {
		if match := vault.MatchSite([]string{candidate}, origin); match > best {
			site, best = candidate, match
		}
	}
	return site, best
}

// FillCredential rechecks the match, returns the sign-in fields and records the use.
func (s *Service) FillCredential(id vault.ID, requester Requester) (Fill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, credential, err := s.readMatching(id, requester)
	if err != nil {
		return Fill{}, err
	}
	if err := s.recordUse(id); err != nil {
		return Fill{}, err
	}
	return Fill{Login: credential.Login, Email: credential.Email, Password: credential.Password}, nil
}

// OneTimeCode rechecks the match and derives the current code without recording a use.
func (s *Service) OneTimeCode(id vault.ID, requester Requester) (vault.OneTimeCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, credential, err := s.readMatching(id, requester)
	if err != nil {
		return vault.OneTimeCode{}, err
	}
	return vault.GenerateOneTimeCode(credential.TOTP, time.Now())
}

// MatchingCredential names credential id while it still matches requester, releasing no secret and recording no use.
func (s *Service) MatchingCredential(id vault.ID, requester Requester) (Suggestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, _, err := s.readMatching(id, requester)
	if err != nil {
		return Suggestion{}, err
	}
	site, match := requester.match(entry)
	return Suggestion{ID: entry.ID, Label: entry.Label, Account: entryAccount(entry), Site: site, Exact: match == vault.MatchExact, Tags: entry.Tags}, nil
}

// readMatching reads credential id, and its entry, while it still matches requester. The caller holds s.mu.
func (s *Service) readMatching(id vault.ID, requester Requester) (vault.Entry, vault.Credential, error) {
	if s.session == nil {
		return vault.Entry{}, vault.Credential{}, ErrNotReady
	}
	entries, err := s.session.List()
	if err != nil {
		return vault.Entry{}, vault.Credential{}, err
	}
	index := slices.IndexFunc(entries, func(entry vault.Entry) bool {
		return entry.ID == id && entry.Kind == vault.KindCredential
	})
	if index < 0 {
		return vault.Entry{}, vault.Credential{}, vault.ErrNotFound
	}
	entry := entries[index]
	if _, match := requester.match(entry); match == vault.MatchNone {
		return vault.Entry{}, vault.Credential{}, ErrNoMatch
	}
	credential, err := s.session.ReadCredential(id)
	if err != nil {
		return vault.Entry{}, vault.Credential{}, err
	}
	if !requester.admits(credential.CredentialInput) {
		return vault.Entry{}, vault.Credential{}, ErrNoMatch
	}
	return entry, credential, nil
}

// AddWebsite adds origin to credential id unless one of its websites already matches it exactly, over plain http and on the same port for an http origin.
func (s *Service) AddWebsite(id vault.ID, origin string) error {
	if vault.SiteOf(origin) == "" {
		return vault.ErrInvalidInput
	}
	changed, err := s.addWebsite(id, origin)
	if err != nil || !changed {
		return err
	}
	s.changes.Record()
	return nil
}

// addWebsite is AddWebsite under one hold of the lock, reporting whether the credential changed.
func (s *Service) addWebsite(id vault.ID, origin string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return false, ErrNotReady
	}
	credential, err := s.session.ReadCredential(id)
	if err != nil {
		return false, err
	}
	known := vault.MatchSite(credential.Sites(), origin) == vault.MatchExact
	if overHTTP(origin) {
		known = savesHTTPOrigin(credential.CredentialInput, origin)
	}
	if known {
		return false, nil
	}
	added, room := OriginRequester(origin).addition(credential.CredentialInput)
	if !room {
		return false, vault.ErrInvalidInput
	}
	return true, s.editCredential(id, added)
}

// CredentialSites lists the distinct sites the open vault's credentials name, sorted.
func (s *Service) CredentialSites() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return nil, ErrNotReady
	}
	entries, err := s.session.List()
	if err != nil {
		return nil, err
	}
	var sites []string
	for _, entry := range entries {
		if entry.Kind == vault.KindCredential {
			sites = append(sites, entry.Sites...)
		}
	}
	slices.Sort(sites)
	return slices.Compact(sites), nil
}

// NamesSite reports whether a credential or a card's bank in the open vault names site.
func (s *Service) NamesSite(site string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return false, ErrNotReady
	}
	if site == "" {
		return false, nil
	}
	entries, err := s.session.List()
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Kind == vault.KindCredential && slices.Contains(entry.Sites, site) || entry.Kind == vault.KindCard && entry.Site == site {
			return true, nil
		}
	}
	return false, nil
}
