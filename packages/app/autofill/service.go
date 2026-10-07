package autofill

import (
	"bytes"
	"crypto/sha256"
	"errors"

	"github.com/dortanes/ravenpass/packages/app/captures"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/authenticator"
	"github.com/dortanes/ravenpass/packages/vault"
)

var (
	// ErrNotFound reports a credential the open vault does not hold, or a held capture or target that is gone.
	ErrNotFound = vault.ErrNotFound
	// ErrNoMatch reports a credential that no longer matches the requester.
	ErrNoMatch = vaultservice.ErrNoMatch
	// ErrNoCode reports a credential without a one-time code setup.
	ErrNoCode = vault.ErrNoTOTP
	// ErrNameRefused and ErrAccountRefused report a new credential's name or account the vault refuses.
	ErrNameRefused    = vaultservice.ErrNameRefused
	ErrAccountRefused = vaultservice.ErrAccountRefused
	// ErrNothingToSave reports a capture a matching credential already holds or the vault cannot hold.
	ErrNothingToSave = errors.New("the vault already holds the sign-in or cannot hold it")
	// ErrInvalidRelyingParty reports a relying party ID no requester could use.
	ErrInvalidRelyingParty = authenticator.ErrInvalidRelyingParty
	// ErrInvalidRequest reports a malformed passkey request, or a site a requester or credential cannot add.
	ErrInvalidRequest = vault.ErrInvalidInput
	// ErrPasskeyExcluded reports a creation meeting a passkey the vault holds that it excluded.
	ErrPasskeyExcluded = vaultservice.ErrPasskeyExcluded
	// ErrUnsupportedAlgorithm reports a relying party that accepts no algorithm the vault's passkeys sign with.
	ErrUnsupportedAlgorithm = errors.New("the relying party accepts no algorithm the vault's passkeys sign with")
)

// system is the one sender whose captures the service holds: the platform's autofill.
const system = "system"

// GroupChoice is the vault identifier of the group new credentials join, empty for none.
type GroupChoice interface {
	DefaultGroup() string
}

// Service answers a system autofill from the open vault; only Fill and OneTimeCode release credential values, after rechecking the match.
type Service struct {
	vault  *vaultservice.Service
	groups GroupChoice
	held   *captures.Held[vaultservice.Capture]
}

// New composes the service over the vault; groups names the group new credentials join.
func New(vault *vaultservice.Service, groups GroupChoice) (*Service, error) {
	if vault == nil || groups == nil {
		return nil, errors.New("vault service and group choice are required")
	}
	return &Service{vault: vault, groups: groups, held: captures.New[vaultservice.Capture](captures.Lifetime)}, nil
}

// Open reports whether the vault is unlocked.
func (s *Service) Open() bool {
	return s.vault.Unlocked()
}

// Suggest lists the credentials matching r, exact matches first, then the most recently used on the device.
func (s *Service) Suggest(r Requester, code bool) ([]Suggestion, error) {
	suggestions, err := s.vault.Suggestions(requester(r), purposeOf(code))
	if err != nil {
		return nil, err
	}
	return suggestionsOf(suggestions), nil
}

// Fill releases the credential's login, email and password once it still matches r, and records the use.
func (s *Service) Fill(id string, r Requester) (Login, error) {
	credential, err := vault.ParseID(id)
	if err != nil {
		return Login{}, ErrNotFound
	}
	fill, err := s.vault.FillCredential(credential, requester(r))
	if err != nil {
		return Login{}, err
	}
	return Login{Login: fill.Login, Email: fill.Email, Password: fill.Password}, nil
}

// OneTimeCode generates the credential's current code once it still matches r.
func (s *Service) OneTimeCode(id string, r Requester) (Code, error) {
	credential, err := vault.ParseID(id)
	if err != nil {
		return Code{}, ErrNotFound
	}
	code, err := s.vault.OneTimeCode(credential, requester(r))
	if err != nil {
		return Code{}, err
	}
	return Code{Code: code.Code, ExpiresAt: code.ExpiresAt.UnixMilli()}, nil
}

// Search lists credentials whose label, login, email or a site contains query, ignoring case, pinned first.
func (s *Service) Search(query string, code bool) ([]Suggestion, error) {
	found, err := s.vault.SearchCredentials(query, purposeOf(code))
	if err != nil {
		return nil, err
	}
	return suggestionsOf(found), nil
}

func purposeOf(code bool) vaultservice.Purpose {
	if code {
		return vaultservice.PurposeCode
	}
	return vaultservice.PurposeSignIn
}

// Sites lists each site the open vault's credentials name, for the caller's Digital Asset Links check.
func (s *Service) Sites() ([]string, error) {
	return s.vault.CredentialSites()
}

// Link links the credential id to app, one link for each of its signers.
func (s *Service) Link(id string, app App) error {
	credential, err := vault.ParseID(id)
	if err != nil {
		return ErrNotFound
	}
	return s.vault.LinkApp(credential, app.Package, app.Signers)
}

// AddSite adds the origin of page r to the credential's websites; an app joins a credential through Link.
func (s *Service) AddSite(id string, r Requester) error {
	if r.Origin == "" {
		return ErrInvalidRequest
	}
	credential, err := vault.ParseID(id)
	if err != nil {
		return ErrNotFound
	}
	return s.vault.AddWebsite(credential, r.Origin)
}

// Hold keeps c in memory for captures.Lifetime and answers where it can be saved; a failed Hold keeps nothing.
func (s *Service) Hold(c Capture) (Offer, error) {
	capture := vaultservice.Capture{Requester: requester(c.Requester), Account: c.Account, Password: c.Password}
	offer, err := s.vault.CaptureOffer(capture)
	if err != nil {
		return Offer{}, err
	}
	if offer.Nothing {
		return Offer{}, ErrNothingToSave
	}
	targets := make([]Target, len(offer.Targets))
	offered := make([]string, len(offer.Targets))
	for i, target := range offer.Targets {
		targets[i] = Target{ID: target.ID.String(), Label: target.Label, Account: target.Account, Action: saveActions[target.Action], Tags: target.Tags}
		offered[i] = targets[i].ID
	}
	token, err := s.held.Keep(system, capture, offered)
	if err != nil {
		return Offer{}, err
	}
	answer := Offer{Token: token, Name: capture.Requester.Name(), Targets: targets}
	if offer.Suggested != (vault.ID{}) {
		answer.Suggested = offer.Suggested.String()
	}
	return answer, nil
}

// Passkeys lists the passkeys for exactly rpID, which the platform or caller must have checked against the requester.
func (s *Service) Passkeys(rpID string, allowed [][]byte) ([]PasskeyChoice, error) {
	choices, err := s.vault.PasskeysFor(rpID, allowed)
	if err != nil {
		return nil, err
	}
	result := make([]PasskeyChoice, len(choices))
	for i, choice := range choices {
		result[i] = passkeyChoiceOf(choice)
	}
	return result, nil
}

// HeldPasskey returns the passkey credentialID of credential id for exactly rpID, checked as in Passkeys; ErrNotFound for none.
func (s *Service) HeldPasskey(rpID, id string, credentialID []byte) (PasskeyChoice, error) {
	credential, err := vault.ParseID(id)
	if err != nil {
		return PasskeyChoice{}, ErrNotFound
	}
	choice, err := s.vault.HeldPasskey(rpID, credential, credentialID)
	if err != nil {
		return PasskeyChoice{}, err
	}
	return passkeyChoiceOf(choice), nil
}

// CheckExclusions fails with ErrPasskeyExcluded when the vault holds a passkey for rpID, checked as in Passkeys, that exclude lists.
func (s *Service) CheckExclusions(rpID string, exclude [][]byte) error {
	return s.vault.CheckExclusions(rpID, exclude)
}

func passkeyChoiceOf(choice vaultservice.PasskeyChoice) PasskeyChoice {
	return PasskeyChoice{
		ID: choice.Credential.String(), CredentialID: choice.CredentialID, UserHandle: choice.UserHandle,
		Account: choice.Account, DisplayName: choice.DisplayName, Label: choice.Label,
	}
}

// ClientDataHash is a platform's SHA-256 client data hash; any other length fails with ErrInvalidRequest.
func ClientDataHash(hash []byte) (*[sha256.Size]byte, error) {
	if len(hash) != sha256.Size {
		return nil, ErrInvalidRequest
	}
	return (*[sha256.Size]byte)(bytes.Clone(hash)), nil
}

// SignPasskey signs as a platform provider, saving a nonzero counter first; the caller must verify the owner, as Verified is only recorded.
func (s *Service) SignPasskey(signIn PasskeySignIn) (PasskeyAssertion, error) {
	credential, err := vault.ParseID(signIn.ID)
	if err != nil {
		return PasskeyAssertion{}, ErrNotFound
	}
	signed, err := s.vault.SignPlatformPasskey(vaultservice.PlatformPasskeySignIn{
		RPID:       signIn.RPID,
		Client:     vaultservice.PlatformClient{Hash: signIn.ClientDataHash, Origin: signIn.Origin, Challenge: signIn.Challenge},
		Verified:   signIn.Verified,
		Credential: credential, CredentialID: signIn.CredentialID,
	})
	if err != nil {
		return PasskeyAssertion{}, err
	}
	return PasskeyAssertion{
		CredentialID: signed.CredentialID, ClientData: signed.ClientData, AuthenticatorData: signed.AuthenticatorData,
		Signature: signed.Signature, UserHandle: signed.UserHandle,
	}, nil
}

// CreatePasskey creates a passkey as a platform provider, returned once saved; the caller must verify the owner, as Verified is only recorded.
func (s *Service) CreatePasskey(creation PasskeyCreation) (CreatedPasskey, error) {
	if !authenticator.Accepts(creation.Algorithms) {
		return CreatedPasskey{}, ErrUnsupportedAlgorithm
	}
	created, err := s.vault.CreatePlatformPasskey(vaultservice.PlatformPasskeyCreation{
		RPID:     creation.RPID,
		RPName:   creation.RPName,
		User:     vaultservice.PasskeyUser{Handle: creation.User.Handle, Name: creation.User.Name, DisplayName: creation.User.DisplayName},
		Client:   vaultservice.PlatformClient{Hash: creation.ClientDataHash, Origin: creation.Origin, Challenge: creation.Challenge},
		Exclude:  creation.Exclude,
		Verified: creation.Verified,
	}, s.groups.DefaultGroup())
	if err != nil {
		return CreatedPasskey{}, err
	}
	return CreatedPasskey{
		ID: created.Credential.String(), CredentialID: created.CredentialID, ClientData: created.ClientData,
		AttestationObject: created.Attestation.AttestationObject, AuthenticatorData: created.Attestation.AuthenticatorData,
		PublicKey: created.Attestation.PublicKey, Algorithm: created.Attestation.Algorithm,
	}, nil
}

var saveActions = map[vaultservice.TargetAction]SaveAction{
	vaultservice.TargetUpdate:  SaveUpdate,
	vaultservice.TargetAddSite: SaveAddSite,
}

// Save saves the capture held under token where choice names; a refused capture stays held for another try.
func (s *Service) Save(token string, choice Choice) (bool, error) {
	chosen := vaultservice.CaptureChoice{Name: choice.Name, Account: choice.Account}
	if choice.Target != "" {
		target, err := vault.ParseID(choice.Target)
		if err != nil {
			return false, ErrNotFound
		}
		chosen.Target = target
	}
	capture, err := s.held.Claim(system, token, choice.Target)
	if err != nil {
		return false, ErrNotFound
	}
	created, err := s.vault.SaveCapture(capture, chosen, s.groups.DefaultGroup())
	s.held.Release(system, token, err == nil)
	return created, err
}

func requester(r Requester) vaultservice.Requester {
	switch {
	case r.Origin != "":
		return vaultservice.PlatformOriginRequester(r.Origin)
	case r.App.Package == "" || len(r.App.Signers) == 0:
		return vaultservice.Requester{}
	default:
		return vaultservice.AppRequester(r.App.Package, r.App.Signers, r.Sites)
	}
}

func suggestionsOf(suggestions []vaultservice.Suggestion) []Suggestion {
	result := make([]Suggestion, len(suggestions))
	for i, suggestion := range suggestions {
		result[i] = Suggestion{ID: suggestion.ID.String(), Label: suggestion.Label, Account: suggestion.Account, Site: suggestion.Site, Exact: suggestion.Exact, Tags: suggestion.Tags}
	}
	return result
}
