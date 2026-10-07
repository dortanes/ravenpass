package vaultservice

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/dortanes/ravenpass/packages/importers"
	"github.com/dortanes/ravenpass/packages/vault"
)

// maxCaptureTargets bounds the credentials one capture is offered to go to.
const maxCaptureTargets = 5

// Errors returned by SaveCapture for a new credential the vault refuses.
var (
	ErrNameRefused    = errors.New("the vault does not accept the name of the new credential")
	ErrAccountRefused = errors.New("the vault does not accept the account of the new credential")
)

// Capture is a submitted password for Requester; Current is the old password a change form asked for.
type Capture struct {
	Requester Requester
	Account   string
	Password  string
	Current   string
}

// TargetAction is what saving a capture to an existing credential changes in it.
type TargetAction uint8

const (
	// TargetUpdate replaces the password of a credential matching the requester, linking a captured app.
	TargetUpdate TargetAction = iota + 1
	// TargetAddSite adds the requester to another credential holding the same account and password.
	TargetAddSite
)

// Target is a credential a capture can be saved to. Account is its login, else its email.
type Target struct {
	ID      vault.ID
	Label   string
	Account string
	Action  TargetAction
	Tags    []string
}

// CaptureOffer is where a capture can be saved; a zero Suggested means a new credential.
type CaptureOffer struct {
	Nothing   bool
	Targets   []Target
	Suggested vault.ID
}

// CaptureChoice is an offered Target, or zero for a new credential named Name that signs in with Account.
type CaptureChoice struct {
	Target  vault.ID
	Name    string
	Account string
}

// PageSite is the site of origin per vault.SiteOf, or origin itself for a host IDNA refuses.
func PageSite(origin string) string {
	if site := vault.SiteOf(origin); site != "" {
		return site
	}
	return origin
}

// siteLabel is a new credential's default label: the site in Unicode, cut to the label limit.
func siteLabel(site string) string {
	return cutLabel(vault.SiteName(site))
}

// cutLabel is name cut to the vault's label limit at a character boundary.
func cutLabel(name string) string {
	if utf8.RuneCountInString(name) <= vault.MaxLabelLength {
		return name
	}
	return string([]rune(name)[:vault.MaxLabelLength])
}

// candidate is a credential a capture may go to.
type candidate struct {
	target   Target
	password string
	holder   bool
	lastUsed int64
}

// CaptureOffer compares a capture with the open vault's credentials without recording a use.
func (s *Service) CaptureOffer(capture Capture) (CaptureOffer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.captureOffer(capture)
}

// captureOffer is CaptureOffer for a caller that holds s.mu.
func (s *Service) captureOffer(capture Capture) (CaptureOffer, error) {
	if s.session == nil {
		return CaptureOffer{}, ErrNotReady
	}
	if !storable(capture) {
		return CaptureOffer{Nothing: true}, nil
	}
	lastUsed, err := s.lastUses()
	if err != nil {
		return CaptureOffer{}, err
	}
	entries, err := s.session.List()
	if err != nil {
		return CaptureOffer{}, err
	}
	var updates, additions []candidate
	for _, entry := range entries {
		if entry.Kind != vault.KindCredential {
			continue
		}
		_, match := capture.Requester.match(entry)
		matches := match != vault.MatchNone
		holder := holdsAccount(entry, capture.Account)
		if !matches && !holder {
			continue
		}
		credential, err := s.session.ReadCredential(entry.ID)
		if err != nil {
			return CaptureOffer{}, err
		}
		// A credential the requester may not fill takes the capture as a credential of another site does.
		matches = matches && capture.Requester.admits(credential.CredentialInput)
		if !matches && !holder {
			continue
		}
		found := candidate{
			target:   Target{ID: entry.ID, Label: entry.Label, Account: entryAccount(entry), Action: TargetUpdate, Tags: entry.Tags},
			password: credential.Password,
			holder:   holder,
			lastUsed: lastUsed[entry.ID],
		}
		_, room := capture.Requester.addition(credential.CredentialInput)
		switch {
		case matches && found.password == capture.Password && (holder || capture.Account == ""):
			return CaptureOffer{Nothing: true}, nil
		case matches:
			updates = append(updates, found)
		case found.password == capture.Password && room:
			found.target.Action = TargetAddSite
			additions = append(additions, found)
		}
	}
	slices.SortStableFunc(updates, byHolderThenUse)
	slices.SortStableFunc(additions, byHolderThenUse)
	offered := append(updates, additions...)
	offered = offered[:min(len(offered), maxCaptureTargets)]
	offer := CaptureOffer{Targets: make([]Target, len(offered)), Suggested: suggestedTarget(capture, offered)}
	for i, found := range offered {
		offer.Targets[i] = found.target
	}
	return offer, nil
}

// storable is false for an empty password, which an update would use to erase the stored one.
func storable(capture Capture) bool {
	if capture.Password == "" {
		return false
	}
	input := capture.Requester.credential(capture.Requester.Name(), capture.Password)
	_, err := vault.PreviewNewItem(vault.NewItem{Credential: &input})
	return err == nil
}

// holdsAccount matches a non-empty account against the login, or the email without case.
func holdsAccount(entry vault.Entry, account string) bool {
	return account != "" && (entry.Detail == account || strings.EqualFold(entry.Email, account))
}

func byHolderThenUse(a, b candidate) int {
	return cmp.Or(setFirst(a.holder, b.holder), byUseThenLabel(a.lastUsed, b.lastUsed, a.target.Label, b.target.Label))
}

// suggestedTarget prefers the update holding capture.Current, then an update holder, then an add-site holder.
func suggestedTarget(capture Capture, offered []candidate) vault.ID {
	if capture.Current != "" {
		for _, found := range offered {
			if found.target.Action == TargetUpdate && found.password == capture.Current && (found.holder || capture.Account == "") {
				return found.target.ID
			}
		}
	}
	for _, action := range []TargetAction{TargetUpdate, TargetAddSite} {
		for _, found := range offered {
			if found.target.Action == action && found.holder {
				return found.target.ID
			}
		}
	}
	return vault.ID{}
}

// SaveCapture rechecks the offer and saves the capture as choice names under one hold of s.mu.
func (s *Service) SaveCapture(capture Capture, choice CaptureChoice, group string) (bool, error) {
	created, err := s.saveCapture(capture, choice, group)
	if err != nil {
		return false, err
	}
	s.changes.Record()
	return created, nil
}

func (s *Service) saveCapture(capture Capture, choice CaptureChoice, group string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	offer, err := s.captureOffer(capture)
	if err != nil {
		return false, err
	}
	if offer.Nothing {
		return false, vault.ErrNotFound
	}
	if choice.Target == (vault.ID{}) {
		if err := s.createCaptured(capture, choice, group); err != nil {
			return false, err
		}
		return true, nil
	}
	index := slices.IndexFunc(offer.Targets, func(target Target) bool { return target.ID == choice.Target })
	if index < 0 {
		return false, vault.ErrNotFound
	}
	credential, err := s.session.ReadCredential(choice.Target)
	if err != nil {
		return false, err
	}
	if offer.Targets[index].Action == TargetUpdate {
		return false, s.editCredential(choice.Target, capture.Requester.update(credential.CredentialInput, capture.Password))
	}
	added, _ := capture.Requester.addition(credential.CredentialInput)
	return false, s.editCredential(choice.Target, added)
}

// createCaptured creates a credential from a storable capture. The caller holds s.mu.
func (s *Service) createCaptured(capture Capture, choice CaptureChoice, group string) error {
	if _, err := vault.PreviewNewItem(vault.NewItem{Credential: &vault.CredentialInput{Label: choice.Name}}); err != nil {
		return fmt.Errorf("%w: %w", ErrNameRefused, err)
	}
	input := capture.Requester.credential(choice.Name, capture.Password)
	signInAs(&input, choice.Account)
	// The name, the requester and the password are accepted by now, so a refusal is the account's.
	if _, err := vault.PreviewNewItem(vault.NewItem{Credential: &input}); err != nil {
		return fmt.Errorf("%w: %w", ErrAccountRefused, err)
	}
	groups, err := s.newItemGroups(group)
	if err != nil {
		return err
	}
	_, err = s.createCredential(input, groups)
	return err
}

// signInAs stores account as the email when it is an email address, else as the login.
func signInAs(input *vault.CredentialInput, account string) {
	if importers.IsEmail(account) {
		input.Email = account
	} else {
		input.Login = account
	}
}

// newItemGroups is group while the open vault holds it, else none. The caller holds s.mu.
func (s *Service) newItemGroups(group string) ([]vault.ID, error) {
	joined, held, err := s.defaultGroup(group)
	if err != nil || !held {
		return nil, err
	}
	return []vault.ID{joined}, nil
}
