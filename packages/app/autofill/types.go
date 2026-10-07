// Package autofill serves a system autofill's requests to the vault for a requester the platform identified.
package autofill

import "github.com/dortanes/ravenpass/packages/app/vaultservice"

// ErrLocked reports a request that needs the vault open.
var ErrLocked = vaultservice.ErrNotReady

// App is an Android app: its package and the SHA-256 digests of its current signing certificates.
type App struct {
	Package string
	Signers [][32]byte
}

// Requester is who asks for a fill: a vouched-for Origin, else an App with the Sites whose Digital Asset Links the caller verified to name it.
type Requester struct {
	Origin string
	App    App
	Sites  []string
}

// Suggestion is a credential offered to a requester; Exact marks a match by exact site or linked app.
type Suggestion struct {
	ID      string
	Label   string
	Account string
	// Site is empty for a match by linked app.
	Site  string
	Exact bool
	Tags  []string
}

// Login is what a credential gives a sign-in form.
type Login struct {
	Login    string
	Email    string
	Password string
}

// Code is a one-time code and the moment, in Unix milliseconds, it stops being accepted.
type Code struct {
	Code      string
	ExpiresAt int64
}

// Capture is a sign-in the owner asked the system to save.
type Capture struct {
	Requester Requester
	Account   string
	Password  string
}

// SaveAction is what saving a capture to an existing credential changes in it.
type SaveAction string

const (
	// SaveUpdate replaces the password of a credential that matches the requester.
	SaveUpdate SaveAction = "update"
	// SaveAddSite adds the requester, as a site or a linked app, to a credential of the same account.
	SaveAddSite SaveAction = "add-site"
)

// Target is a credential a held capture can be saved to.
type Target struct {
	ID      string
	Label   string
	Account string
	Action  SaveAction
	Tags    []string
}

// Offer is where a held capture can be saved; Suggested is empty for a new credential.
type Offer struct {
	Token     string
	Name      string
	Targets   []Target
	Suggested string
}

// Choice is where the owner saves a held capture; an empty Target means a new credential named Name.
type Choice struct {
	Target  string
	Name    string
	Account string
}

// PasskeyChoice is a passkey a sign-in can use; ID names the credential holding it.
type PasskeyChoice struct {
	ID           string
	CredentialID []byte
	// UserHandle is empty for a passkey the index shows without it.
	UserHandle  []byte
	Account     string
	DisplayName string
	Label       string
}

// PasskeyUser is the account a relying party creates a passkey for.
type PasskeyUser struct {
	Handle      []byte
	Name        string
	DisplayName string
}

// PasskeySignIn is a sign-in to RPID, which the platform or the caller must have checked against the requester.
type PasskeySignIn struct {
	RPID string
	// ClientDataHash is the SHA-256 of the platform's client data; nil builds it from Origin and Challenge.
	ClientDataHash *[32]byte
	Origin         string
	Challenge      []byte
	Verified       bool
	ID             string
	CredentialID   []byte
}

// PasskeyAssertion is what the relying party receives; ClientData is nil over the platform's hash.
type PasskeyAssertion struct {
	CredentialID      []byte
	ClientData        []byte
	AuthenticatorData []byte
	Signature         []byte
	UserHandle        []byte
}

// PasskeyCreation asks for a passkey for RPID, checked against the requester as in PasskeySignIn.
type PasskeyCreation struct {
	RPID           string
	RPName         string
	User           PasskeyUser
	ClientDataHash *[32]byte
	Origin         string
	Challenge      []byte
	// Algorithms lists the accepted COSE algorithms; empty means the WebAuthn default, which includes ES256.
	Algorithms []int
	Exclude    [][]byte
	Verified   bool
}

// CreatedPasskey is what the relying party receives for a new passkey; ClientData is as in PasskeyAssertion.
type CreatedPasskey struct {
	ID           string
	CredentialID []byte
	ClientData   []byte
	// AttestationObject is {"fmt": "none", "attStmt": {}, "authData": AuthenticatorData} in CTAP2 canonical CBOR.
	AttestationObject []byte
	AuthenticatorData []byte
	// PublicKey is SubjectPublicKeyInfo DER for the COSE algorithm Algorithm.
	PublicKey []byte
	Algorithm int
}
