package vaultservice

import (
	"cmp"
	"slices"
	"strings"

	"github.com/dortanes/ravenpass/packages/vault"
)

// SearchCredentials lists credentials for purpose whose text contains query, case-insensitive, pinned first.
func (s *Service) SearchCredentials(query string, purpose Purpose) ([]Suggestion, error) {
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
	query = strings.ToLower(strings.TrimSpace(query))
	var found []vault.Entry
	for _, entry := range entries {
		if entry.Kind == vault.KindCredential && (purpose != PurposeCode || entry.Code.Digits > 0) && mentions(entry, query) {
			found = append(found, entry)
		}
	}
	slices.SortStableFunc(found, func(a, b vault.Entry) int {
		return cmp.Or(setFirst(a.Pinned, b.Pinned), byUseThenLabel(lastUsed[a.ID], lastUsed[b.ID], a.Label, b.Label))
	})
	found = found[:min(len(found), maxSuggestions)]
	results := make([]Suggestion, len(found))
	for i, entry := range found {
		results[i] = Suggestion{ID: entry.ID, Label: entry.Label, Account: entryAccount(entry), Site: entry.Site, Tags: entry.Tags}
		if purpose == PurposeCode {
			results[i].Code = entry.Code
		}
	}
	return results, nil
}

// mentions matches a lower-case query against a credential's label, login, email and sites.
func mentions(entry vault.Entry, query string) bool {
	return slices.ContainsFunc(append([]string{entry.Label, entry.Detail, entry.Email}, entry.Sites...), func(text string) bool {
		return strings.Contains(strings.ToLower(text), query)
	})
}
