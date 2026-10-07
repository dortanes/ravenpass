package vaultservice

import (
	"bytes"
	"cmp"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/dortanes/ravenpass/packages/authenticator"
	"github.com/dortanes/ravenpass/packages/vault"
)

// ErrPasskeyExcluded refuses a new passkey when the vault holds one the site excluded.
var ErrPasskeyExcluded = errors.New("the vault already holds a passkey the site excluded")

// PasskeyChoice is a passkey a sign-in can use; Account is its user name, else the login, else the email.
type PasskeyChoice struct {
	Credential   vault.ID
	CredentialID []byte
	UserHandle   []byte
	Account      string
	DisplayName  string
	Label        string
}

// PasskeyTarget is a credential a new passkey can join. Account is its login, else its email.
type PasskeyTarget struct {
	ID      vault.ID
	Label   string
	Account string
	Tags    []string
}

// PasskeyTargets lists credentials a new passkey for RPID can join and whether the site excluded one held.
type PasskeyTargets struct {
	RPID     string
	Targets  []PasskeyTarget
	Excluded bool
}

// PasskeyUser is the account a site creates a passkey for.
type PasskeyUser struct {
	Handle      []byte
	Name        string
	DisplayName string
}

// PasskeyCreation is a page's request to create a passkey; an empty RPID means the origin's host, a zero Target a new credential.
type PasskeyCreation struct {
	Origin    string
	RPID      string
	RPName    string
	User      PasskeyUser
	Challenge []byte
	Exclude   [][]byte
	Verified  bool
	Target    vault.ID
}

// CreatedPasskey is the response to a saved passkey; ClientData is nil when the platform built it.
type CreatedPasskey struct {
	Credential   vault.ID
	CredentialID []byte
	ClientData   []byte
	Attestation  authenticator.Attestation
}

// PasskeySignIn is a page's request to sign in with a passkey; an empty RPID means the origin's host.
type PasskeySignIn struct {
	Origin       string
	RPID         string
	Challenge    []byte
	Verified     bool
	Credential   vault.ID
	CredentialID []byte
}

// PasskeyAssertion is the response to a passkey sign-in; ClientData is nil when the platform built it.
type PasskeyAssertion struct {
	CredentialID      []byte
	ClientData        []byte
	AuthenticatorData []byte
	Signature         []byte
	UserHandle        []byte
}

// PlatformClient is the SHA-256 client data hash a platform built, or when nil, the Origin and Challenge to build it from.
type PlatformClient struct {
	Hash      *[32]byte
	Origin    string
	Challenge []byte
}

// PlatformPasskeySignIn is a platform's sign-in request; the platform has already checked RPID against the requester.
type PlatformPasskeySignIn struct {
	RPID         string
	Client       PlatformClient
	Verified     bool
	Credential   vault.ID
	CredentialID []byte
}

// PlatformPasskeyCreation is a platform's creation request; the platform has already checked RPID against the requester.
type PlatformPasskeyCreation struct {
	RPID     string
	RPName   string
	User     PasskeyUser
	Client   PlatformClient
	Exclude  [][]byte
	Verified bool
}

// clientData is the client data JSON built here, or when json is nil, the SHA-256 hash a platform built.
type clientData struct {
	json []byte
	hash [32]byte
}

// data fails with vault.ErrInvalidInput when c has neither a hash nor an origin and challenge.
func (c PlatformClient) data(ceremony authenticator.Ceremony) (clientData, error) {
	switch {
	case c.Hash != nil:
		return clientData{hash: *c.Hash}, nil
	case c.Origin == "" || len(c.Challenge) == 0:
		return clientData{}, vault.ErrInvalidInput
	default:
		return clientData{json: authenticator.ClientData(ceremony, c.Challenge, c.Origin)}, nil
	}
}

// assert signs over c as profile for rpID with a passkey's private key and counter.
func (c clientData) assert(profile authenticator.Profile, rpID string, privateKey []byte, counter uint32, verified bool) (authenticator.Assertion, error) {
	if c.json == nil {
		return profile.AssertHash(rpID, privateKey, counter, verified, c.hash)
	}
	return profile.Assert(rpID, privateKey, counter, verified, c.json)
}

// SignInPasskeys checks rpID against origin and returns it with the passkeys a sign-in can use.
func (s *Service) SignInPasskeys(origin, rpID string, allow [][]byte) (string, []PasskeyChoice, error) {
	checked, err := authenticator.CheckRelyingParty(origin, rpID)
	if err != nil {
		return "", nil, err
	}
	choices, err := s.passkeyChoices(checked, allow)
	if err != nil {
		return "", nil, err
	}
	return checked, choices, nil
}

// PasskeysFor lists the passkeys a sign-in can use for an rpID the platform already checked.
func (s *Service) PasskeysFor(rpID string, allow [][]byte) ([]PasskeyChoice, error) {
	checked, err := authenticator.RelyingPartyID(rpID)
	if err != nil {
		return nil, err
	}
	return s.passkeyChoices(checked, allow)
}

// HeldPasskey returns the passkey credentialID that credential holds for an rpID the platform already checked, vault.ErrNotFound for none.
func (s *Service) HeldPasskey(rpID string, credential vault.ID, credentialID []byte) (PasskeyChoice, error) {
	choices, err := s.PasskeysFor(rpID, [][]byte{credentialID})
	if err != nil {
		return PasskeyChoice{}, err
	}
	choice, ok := FindPasskey(choices, credential, credentialID)
	if !ok {
		return PasskeyChoice{}, vault.ErrNotFound
	}
	return choice, nil
}

// FindPasskey returns the choice of the passkey credentialID that credential holds, and whether choices has one.
func FindPasskey(choices []PasskeyChoice, credential vault.ID, credentialID []byte) (PasskeyChoice, bool) {
	index := slices.IndexFunc(choices, func(choice PasskeyChoice) bool {
		return choice.Credential == credential && bytes.Equal(choice.CredentialID, credentialID)
	})
	if index < 0 {
		return PasskeyChoice{}, false
	}
	return choices[index], true
}

// CheckExclusions fails with ErrPasskeyExcluded when the vault holds a passkey for an rpID the platform already checked that exclude lists.
func (s *Service) CheckExclusions(rpID string, exclude [][]byte) error {
	checked, err := authenticator.RelyingPartyID(rpID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return ErrNotReady
	}
	entries, err := s.session.List()
	if err != nil {
		return err
	}
	return excluding(entries, checked, exclude)
}

// passkeyChoices lists allowed passkeys for rpID, or every discoverable one when allow is empty.
func (s *Service) passkeyChoices(rpID string, allow [][]byte) ([]PasskeyChoice, error) {
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
	var choices []PasskeyChoice
	for _, entry := range entries {
		for _, face := range entry.Passkeys {
			if face.RPID != rpID || len(allow) > 0 && !containsID(allow, face.CredentialID) || len(allow) == 0 && !face.Discoverable {
				continue
			}
			account := face.UserName
			if account == "" {
				account = entryAccount(entry)
			}
			choices = append(choices, PasskeyChoice{
				Credential: entry.ID, CredentialID: face.CredentialID, UserHandle: face.UserHandle,
				Account: account, DisplayName: face.UserDisplayName, Label: entry.Label,
			})
		}
	}
	slices.SortStableFunc(choices, func(a, b PasskeyChoice) int {
		return byUseThenLabel(lastUsed[a.Credential], lastUsed[b.Credential], a.Label, b.Label)
	})
	return choices[:min(len(choices), maxSuggestions)], nil
}

// passkeyCandidate is a credential a new passkey may join.
type passkeyCandidate struct {
	target   PasskeyTarget
	holder   bool
	lastUsed int64
}

// PasskeyTargets reports where a new passkey for account from origin can be saved.
func (s *Service) PasskeyTargets(origin, rpID, account string, exclude [][]byte) (PasskeyTargets, error) {
	checked, err := authenticator.CheckRelyingParty(origin, rpID)
	if err != nil {
		return PasskeyTargets{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return PasskeyTargets{}, ErrNotReady
	}
	lastUsed, err := s.lastUses()
	if err != nil {
		return PasskeyTargets{}, err
	}
	entries, err := s.session.List()
	if err != nil {
		return PasskeyTargets{}, err
	}
	answer := PasskeyTargets{RPID: checked, Targets: []PasskeyTarget{}}
	var found []passkeyCandidate
	for _, entry := range entries {
		if entry.Kind != vault.KindCredential {
			continue
		}
		answer.Excluded = answer.Excluded || holdsExcluded(entry, checked, exclude)
		if len(entry.Passkeys) >= vault.MaxCredentialPasskeys || vault.MatchSite(entry.Sites, origin) == vault.MatchNone {
			continue
		}
		found = append(found, passkeyCandidate{
			target:   PasskeyTarget{ID: entry.ID, Label: entry.Label, Account: entryAccount(entry), Tags: entry.Tags},
			holder:   holdsAccount(entry, account),
			lastUsed: lastUsed[entry.ID],
		})
	}
	slices.SortStableFunc(found, func(a, b passkeyCandidate) int {
		return cmp.Or(setFirst(a.holder, b.holder), byUseThenLabel(a.lastUsed, b.lastUsed, a.target.Label, b.target.Label))
	})
	for _, candidate := range found[:min(len(found), maxSuggestions)] {
		answer.Targets = append(answer.Targets, candidate.target)
	}
	return answer, nil
}

// CreatePasskey creates and saves a passkey for a page; the response is returned only after the save.
func (s *Service) CreatePasskey(creation PasskeyCreation, group string) (CreatedPasskey, error) {
	rpID, err := authenticator.CheckRelyingParty(creation.Origin, creation.RPID)
	if err != nil {
		return CreatedPasskey{}, err
	}
	return s.createPasskey(newPasskey{
		profile: authenticator.BrowserExtension,
		rpID:    rpID, rpName: creation.RPName, website: creation.Origin, user: creation.User,
		exclude: creation.Exclude, verified: creation.Verified, target: creation.Target,
		clientData: authenticator.ClientData(authenticator.Create, creation.Challenge, creation.Origin),
	}, group)
}

// CreatePlatformPasskey saves a platform's passkey beside the same account's passkey, else in a new credential.
func (s *Service) CreatePlatformPasskey(creation PlatformPasskeyCreation, group string) (CreatedPasskey, error) {
	rpID, err := authenticator.RelyingPartyID(creation.RPID)
	if err != nil {
		return CreatedPasskey{}, err
	}
	client, err := creation.Client.data(authenticator.Create)
	if err != nil {
		return CreatedPasskey{}, err
	}
	return s.createPasskey(newPasskey{
		profile: authenticator.PlatformProvider,
		rpID:    rpID, rpName: creation.RPName, website: "https://" + rpID, user: creation.User,
		exclude: creation.Exclude, verified: creation.Verified, joinHolder: true, clientData: client.json,
	}, group)
}

// newPasskey is a creation with a checked rpID; website is also the origin an existing target must match.
type newPasskey struct {
	profile    authenticator.Profile
	rpID       string
	rpName     string
	website    string
	user       PasskeyUser
	exclude    [][]byte
	verified   bool
	target     vault.ID
	joinHolder bool
	clientData []byte
}

// createPasskey records the change after savePasskey releases s.mu.
func (s *Service) createPasskey(request newPasskey, group string) (CreatedPasskey, error) {
	created, err := s.savePasskey(request, group)
	if err != nil {
		return CreatedPasskey{}, err
	}
	s.changes.Record()
	return created, nil
}

func (s *Service) savePasskey(request newPasskey, group string) (CreatedPasskey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return CreatedPasskey{}, ErrNotReady
	}
	entries, err := s.session.List()
	if err != nil {
		return CreatedPasskey{}, err
	}
	if err := excluding(entries, request.rpID, request.exclude); err != nil {
		return CreatedPasskey{}, err
	}
	if request.target != (vault.ID{}) {
		index := slices.IndexFunc(entries, func(entry vault.Entry) bool {
			return entry.ID == request.target && entry.Kind == vault.KindCredential
		})
		if index < 0 {
			return CreatedPasskey{}, vault.ErrNotFound
		}
		if vault.MatchSite(entries[index].Sites, request.website) == vault.MatchNone {
			return CreatedPasskey{}, ErrNoMatch
		}
	} else if request.joinHolder {
		request.target = holderOf(entries, request.rpID, request.user)
	}
	key, err := authenticator.NewKey()
	if err != nil {
		return CreatedPasskey{}, err
	}
	defer clear(key.PrivateKey)
	attestation, err := request.profile.Attest(request.rpID, key.CredentialID, key.PrivateKey, request.verified)
	if err != nil {
		return CreatedPasskey{}, err
	}
	passkey := vault.Passkey{
		CredentialID:    key.CredentialID,
		RPID:            request.rpID,
		UserHandle:      request.user.Handle,
		UserName:        request.user.Name,
		UserDisplayName: request.user.DisplayName,
		PrivateKey:      key.PrivateKey,
		Discoverable:    true,
		CreatedAt:       time.Now(),
	}
	id, pending, err := s.preparePasskey(request, passkey, group)
	if err != nil {
		return CreatedPasskey{}, err
	}
	if err := s.commit(pending); err != nil {
		return CreatedPasskey{}, err
	}
	return CreatedPasskey{Credential: id, CredentialID: key.CredentialID, ClientData: request.clientData, Attestation: attestation}, nil
}

// preparePasskey adds passkey to the request's target or a new credential. The caller holds s.mu.
func (s *Service) preparePasskey(request newPasskey, passkey vault.Passkey, group string) (vault.ID, *vault.Pending, error) {
	if request.target != (vault.ID{}) {
		pending, err := s.session.PrepareAddPasskey(request.target, passkey)
		return request.target, pending, err
	}
	label := cutLabel(request.rpName)
	if strings.TrimSpace(label) == "" {
		label = siteLabel(passkey.RPID)
	}
	input := vault.CredentialInput{Label: label, Websites: []string{vault.WebsiteOf(request.website)}, Passkeys: []vault.Passkey{passkey}}
	signInAs(&input, request.user.Name)
	groups, err := s.newItemGroups(group)
	if err != nil {
		return vault.ID{}, nil, err
	}
	pending, id, err := s.session.PrepareCreate(input, groups)
	return id, pending, err
}

// SignPasskey signs a page's challenge with the named passkey and records the credential's use.
func (s *Service) SignPasskey(signIn PasskeySignIn) (PasskeyAssertion, error) {
	rpID, err := authenticator.CheckRelyingParty(signIn.Origin, signIn.RPID)
	if err != nil {
		return PasskeyAssertion{}, err
	}
	client := clientData{json: authenticator.ClientData(authenticator.Get, signIn.Challenge, signIn.Origin)}
	return s.signPasskey(authenticator.BrowserExtension, rpID, signIn.Credential, signIn.CredentialID, signIn.Verified, client)
}

// SignPlatformPasskey signs a platform's sign-in with the named passkey and records the credential's use.
func (s *Service) SignPlatformPasskey(signIn PlatformPasskeySignIn) (PasskeyAssertion, error) {
	rpID, err := authenticator.RelyingPartyID(signIn.RPID)
	if err != nil {
		return PasskeyAssertion{}, err
	}
	client, err := signIn.Client.data(authenticator.Get)
	if err != nil {
		return PasskeyAssertion{}, err
	}
	return s.signPasskey(authenticator.PlatformProvider, rpID, signIn.Credential, signIn.CredentialID, signIn.Verified, client)
}

// signPasskey records the change after assertPasskey releases s.mu when a counter was saved.
func (s *Service) signPasskey(profile authenticator.Profile, rpID string, credential vault.ID, credentialID []byte, verified bool, client clientData) (PasskeyAssertion, error) {
	assertion, saved, err := s.assertPasskey(profile, rpID, credential, credentialID, verified, client)
	if saved {
		s.changes.Record()
	}
	return assertion, err
}

// assertPasskey saves the advanced counter before signing, so a failed save signs nothing.
func (s *Service) assertPasskey(profile authenticator.Profile, rpID string, credential vault.ID, credentialID []byte, verified bool, client clientData) (PasskeyAssertion, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return PasskeyAssertion{}, false, ErrNotReady
	}
	entries, err := s.session.List()
	if err != nil {
		return PasskeyAssertion{}, false, err
	}
	held := slices.ContainsFunc(entries, func(entry vault.Entry) bool {
		return entry.ID == credential && entry.Kind == vault.KindCredential &&
			slices.ContainsFunc(entry.Passkeys, func(face vault.PasskeyFace) bool {
				return face.RPID == rpID && bytes.Equal(face.CredentialID, credentialID)
			})
	})
	if !held {
		return PasskeyAssertion{}, false, vault.ErrNotFound
	}
	passkey, err := s.session.ReadPasskey(credential, credentialID)
	if err != nil {
		return PasskeyAssertion{}, false, err
	}
	defer clear(passkey.PrivateKey)
	counter, pending, err := s.session.PrepareCountPasskeyUse(credential, credentialID)
	if err != nil {
		return PasskeyAssertion{}, false, err
	}
	saved := pending != nil
	if saved {
		if err := s.commit(pending); err != nil {
			return PasskeyAssertion{}, false, err
		}
	}
	if err := s.recordUse(credential); err != nil {
		return PasskeyAssertion{}, saved, err
	}
	assertion, err := client.assert(profile, rpID, passkey.PrivateKey, counter, verified)
	if err != nil {
		return PasskeyAssertion{}, saved, err
	}
	return PasskeyAssertion{
		CredentialID:      passkey.CredentialID,
		ClientData:        client.json,
		AuthenticatorData: assertion.AuthenticatorData,
		Signature:         assertion.Signature,
		UserHandle:        passkey.UserHandle,
	}, saved, nil
}

// holderOf returns the first credential with room that holds user's passkey for rpID, else zero.
func holderOf(entries []vault.Entry, rpID string, user PasskeyUser) vault.ID {
	for _, entry := range entries {
		if entry.Kind != vault.KindCredential || len(entry.Passkeys) >= vault.MaxCredentialPasskeys {
			continue
		}
		if slices.ContainsFunc(entry.Passkeys, func(face vault.PasskeyFace) bool {
			return face.RPID == rpID && (len(user.Handle) > 0 && bytes.Equal(face.UserHandle, user.Handle) ||
				user.Name != "" && face.UserName == user.Name)
		}) {
			return entry.ID
		}
	}
	return vault.ID{}
}

// excluding fails with ErrPasskeyExcluded when one of entries holds a passkey for rpID that exclude lists.
func excluding(entries []vault.Entry, rpID string, exclude [][]byte) error {
	if slices.ContainsFunc(entries, func(entry vault.Entry) bool { return holdsExcluded(entry, rpID, exclude) }) {
		return ErrPasskeyExcluded
	}
	return nil
}

// holdsExcluded reports whether entry holds a passkey for rpID whose credential ID exclude lists.
func holdsExcluded(entry vault.Entry, rpID string, exclude [][]byte) bool {
	return slices.ContainsFunc(entry.Passkeys, func(face vault.PasskeyFace) bool {
		return face.RPID == rpID && containsID(exclude, face.CredentialID)
	})
}

func containsID(ids [][]byte, id []byte) bool {
	return slices.ContainsFunc(ids, func(listed []byte) bool { return bytes.Equal(listed, id) })
}
