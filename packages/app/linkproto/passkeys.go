package linkproto

import (
	"encoding/base64"
	"encoding/json"
	"unicode/utf8"

	"github.com/dortanes/ravenpass/packages/authenticator"
	"github.com/dortanes/ravenpass/packages/vault"
)

const (
	// MaxChallengeBytes bounds the challenge a passkey request carries.
	MaxChallengeBytes = 1024
	// MaxUserHandleBytes bounds the user handle a site gives a new passkey, WebAuthn §5.4.3.
	MaxUserHandleBytes = 64
	// MaxCredentialIDBytes bounds a credential ID, WebAuthn §4.
	MaxCredentialIDBytes = 1023
	// MaxCredentialIDs bounds the credential IDs an allow or exclude list carries.
	MaxCredentialIDs = 64
	// MaxPasskeyNameLength bounds, in characters, passkey names and accounts: the longest login a vault holds.
	MaxPasskeyNameLength = vault.MaxLoginLength
)

// Bytes is binary data, which JSON carries as unpadded base64url.
type Bytes []byte

// MarshalJSON writes unpadded base64url.
func (b Bytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(base64.RawURLEncoding.EncodeToString(b))
}

// UnmarshalJSON reads unpadded base64url. A JSON null leaves b as it is.
func (b *Bytes) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	decoded, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return err
	}
	*b = decoded
	return nil
}

// PasskeyMode is what a passkeys request lists.
type PasskeyMode string

// Passkey modes: the passkeys a sign-in can use, or the credentials a new passkey can join.
const (
	PasskeyGet    PasskeyMode = "get"
	PasskeyCreate PasskeyMode = "create"
)

// PasskeyUser is the account a site creates a passkey for; ID is its user handle.
type PasskeyUser struct {
	ID          Bytes  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// PasskeyFields are the passkey request fields as sent; an empty RPID means the origin's host.
type PasskeyFields struct {
	RPID         string      `json:"rpId"`
	RPName       string      `json:"rpName"`
	Mode         PasskeyMode `json:"mode"`
	User         PasskeyUser `json:"user"`
	Challenge    Bytes       `json:"challenge"`
	Verify       bool        `json:"verify"`
	Allow        []Bytes     `json:"allow"`
	Exclude      []Bytes     `json:"exclude"`
	CredentialID Bytes       `json:"credentialId"`
	// Algorithms are the COSE algorithms a creation's relying party accepts; none means the WebAuthn default.
	Algorithms []int `json:"algorithms"`
}

// PasskeyQuery is a passkeys request; Allow applies to PasskeyGet, Account and Exclude to PasskeyCreate.
type PasskeyQuery struct {
	Origin  string
	RPID    string
	Mode    PasskeyMode
	Allow   [][]byte
	Exclude [][]byte
	Account string
}

// PasskeyCreation is a passkey-create request; Target is TargetNew or the credential the passkey joins.
type PasskeyCreation struct {
	Origin    string
	RPID      string
	RPName    string
	User      PasskeyUser
	Challenge []byte
	Verify    bool
	Target    string
	Exclude   [][]byte
}

// TargetNew is the passkey-create target that saves the passkey in a new credential.
const TargetNew = "new"

// PasskeySignIn is a passkey-sign request for the passkey CredentialID that Credential holds.
type PasskeySignIn struct {
	Origin       string
	RPID         string
	Challenge    []byte
	Verify       bool
	Credential   string
	CredentialID []byte
}

// PasskeyQuery reads a passkeys request, false for an unknown mode or a field beyond its bound.
func (r Request) PasskeyQuery() (PasskeyQuery, bool) {
	fields := r.Passkey
	allow, allowed := credentialIDs(fields.Allow)
	exclude, excluded := credentialIDs(fields.Exclude)
	known := fields.Mode == PasskeyGet || fields.Mode == PasskeyCreate
	if !known || !allowed || !excluded || !fitsText(fields.RPID, MaxOriginLength) || !fitsText(r.Account, MaxPasskeyNameLength) {
		return PasskeyQuery{}, false
	}
	return PasskeyQuery{Origin: r.Origin, RPID: fields.RPID, Mode: fields.Mode, Allow: allow, Exclude: exclude, Account: r.Account}, true
}

// PasskeyCreation reads a passkey-create request, false for a field beyond its bound, no target, or algorithms without ES256.
func (r Request) PasskeyCreation() (PasskeyCreation, bool) {
	fields := r.Passkey
	exclude, excluded := credentialIDs(fields.Exclude)
	user := fields.User
	valid := excluded && r.Target != "" && validChallenge(fields.Challenge) && authenticator.Accepts(fields.Algorithms) &&
		len(user.ID) >= 1 && len(user.ID) <= MaxUserHandleBytes &&
		fitsText(user.Name, MaxPasskeyNameLength) && fitsText(user.DisplayName, MaxPasskeyNameLength) &&
		fitsText(fields.RPID, MaxOriginLength) && fitsText(fields.RPName, MaxPasskeyNameLength)
	if !valid {
		return PasskeyCreation{}, false
	}
	return PasskeyCreation{
		Origin: r.Origin, RPID: fields.RPID, RPName: fields.RPName, User: user, Challenge: fields.Challenge,
		Verify: fields.Verify, Target: r.Target, Exclude: exclude,
	}, true
}

// PasskeySignIn reads a passkey-sign request, false for a field beyond its bound.
func (r Request) PasskeySignIn() (PasskeySignIn, bool) {
	fields := r.Passkey
	valid := validChallenge(fields.Challenge) && validCredentialID(fields.CredentialID) && fitsText(fields.RPID, MaxOriginLength)
	if !valid {
		return PasskeySignIn{}, false
	}
	return PasskeySignIn{
		Origin: r.Origin, RPID: fields.RPID, Challenge: fields.Challenge, Verify: fields.Verify,
		Credential: r.Credential, CredentialID: fields.CredentialID,
	}, true
}

// credentialIDs returns listed as bytes, false beyond MaxCredentialIDs or for an ID beyond its bound.
func credentialIDs(listed []Bytes) ([][]byte, bool) {
	if len(listed) > MaxCredentialIDs {
		return nil, false
	}
	ids := make([][]byte, len(listed))
	for i, id := range listed {
		if !validCredentialID(id) {
			return nil, false
		}
		ids[i] = id
	}
	return ids, true
}

func validCredentialID(id []byte) bool {
	return len(id) >= 1 && len(id) <= MaxCredentialIDBytes
}

func validChallenge(challenge []byte) bool {
	return len(challenge) >= 1 && len(challenge) <= MaxChallengeBytes
}

// fitsText reports UTF-8 of at most limit characters, empty included.
func fitsText(value string, limit int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= limit
}

// Passkeys is the PasskeyGet result: the checked relying party ID and the passkeys for exactly it.
type Passkeys struct {
	RPID     string          `json:"rpId"`
	Passkeys []PasskeyOption `json:"passkeys"`
}

// PasskeyOption is a passkey a sign-in can use; Credential is the credential holding it.
type PasskeyOption struct {
	Credential   string `json:"credential"`
	CredentialID Bytes  `json:"credentialId"`
	Account      string `json:"account"`
	Label        string `json:"label"`
}

// PasskeyTargets is the PasskeyCreate result; Excluded reports that the vault holds an excluded passkey.
type PasskeyTargets struct {
	RPID     string          `json:"rpId"`
	Targets  []PasskeyTarget `json:"targets"`
	Excluded bool            `json:"excluded"`
}

// PasskeyTarget is a credential a new passkey can join. Account is its login, else its email.
type PasskeyTarget struct {
	Credential string   `json:"credential"`
	Label      string   `json:"label"`
	Account    string   `json:"account"`
	Tags       []string `json:"tags,omitempty"`
}

// CreatedPasskey is the passkey-create result; PublicKey is SubjectPublicKeyInfo DER, its algorithm COSE.
type CreatedPasskey struct {
	Credential         string `json:"credential"`
	CredentialID       Bytes  `json:"credentialId"`
	ClientDataJSON     Bytes  `json:"clientDataJSON"`
	AttestationObject  Bytes  `json:"attestationObject"`
	AuthenticatorData  Bytes  `json:"authenticatorData"`
	PublicKey          Bytes  `json:"publicKey"`
	PublicKeyAlgorithm int    `json:"publicKeyAlgorithm"`
}

// PasskeyAssertion is the passkey-sign result the page's PublicKeyCredential carries.
type PasskeyAssertion struct {
	CredentialID      Bytes `json:"credentialId"`
	ClientDataJSON    Bytes `json:"clientDataJSON"`
	AuthenticatorData Bytes `json:"authenticatorData"`
	Signature         Bytes `json:"signature"`
	UserHandle        Bytes `json:"userHandle"`
}
