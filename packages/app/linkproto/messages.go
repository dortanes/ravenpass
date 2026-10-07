package linkproto

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/dortanes/ravenpass/packages/vault"
)

// Field limits, request types, error codes and vault states of session messages.
const (
	// MaxNameLength bounds, in characters, the name an extension links under.
	MaxNameLength = 64
	// MaxOriginLength bounds, in characters, the origin of a request: the longest origin a vault holds.
	MaxOriginLength = vault.MaxOriginLength
	// MaxSiteLength bounds, in characters, the site of an icon request: the longest host vault.SiteOf reports.
	MaxSiteLength = vault.MaxOriginLength

	RequestStatus        = "status"
	RequestUnlink        = "unlink"
	RequestSuggest       = "suggest"
	RequestFill          = "fill"
	RequestCode          = "code"
	RequestAddWebsite    = "add-website"
	RequestIcon          = "icon"
	RequestUnlock        = "unlock"
	RequestIdentities    = "identities"
	RequestShare         = "share"
	RequestCapture       = "capture"
	RequestReview        = "review"
	RequestSave          = "save"
	RequestDiscard       = "discard"
	RequestPasskeys      = "passkeys"
	RequestPasskeyCreate = "passkey-create"
	RequestPasskeySign   = "passkey-sign"
	RequestCards         = "cards"
	RequestCardFill      = "card-fill"
	RequestCardCapture   = "card-capture"

	ErrorUnknownRequest      = "unknown-request"
	ErrorLocked              = "locked"
	ErrorNoMatch             = "no-match"
	ErrorNotFound            = "not-found"
	ErrorNoCode              = "no-code"
	ErrorInvalidOrigin       = "invalid-origin"
	ErrorInvalidPurpose      = "invalid-purpose"
	ErrorDeclined            = "declined"
	ErrorUnverifiable        = "unverifiable"
	ErrorInvalidAccount      = "invalid-account"
	ErrorInvalidName         = "invalid-name"
	ErrorInvalidRequest      = "invalid-request"
	ErrorInvalidRelyingParty = "invalid-rp"
	ErrorExcluded            = "excluded"
	ErrorFull                = "full"

	VaultUnlocked = "unlocked"
	VaultLocked   = "locked"
)

// Purpose is what a suggest request asks of a credential.
type Purpose string

// Purposes a suggest request can name.
const (
	PurposeSignIn Purpose = "sign-in"
	PurposeCode   Purpose = "code"
)

// LinkRequest is the payload of the extension's third link message.
type LinkRequest struct {
	Name string `json:"name"`
}

// Greeting is the interface language and sign-in style the desktop app sends at link and session start.
type Greeting struct {
	Language string `json:"language"`
	SignIn   string `json:"signIn"`
}

type linked struct {
	Type string `json:"type"`
	Greeting
}

// LinkedMessage is the transport message sent once a link is recorded: {"type":"linked","language":"en","signIn":"card"}.
func LinkedMessage(greeting Greeting) ([]byte, error) {
	return json.Marshal(linked{Type: "linked", Greeting: greeting})
}

// SessionGreeting is the payload of the desktop app's second session message: {"language":"en","signIn":"card"}.
func SessionGreeting(greeting Greeting) ([]byte, error) {
	return json.Marshal(greeting)
}

// ParseLinkRequest reads a link request, which must carry a valid name.
func ParseLinkRequest(payload []byte) (LinkRequest, error) {
	var request LinkRequest
	if err := json.Unmarshal(payload, &request); err != nil || !ValidName(request.Name) {
		return LinkRequest{}, ErrMalformed
	}
	return request, nil
}

// ValidName reports whether name is UTF-8 of 1 to MaxNameLength characters.
func ValidName(name string) bool {
	return validText(name, MaxNameLength)
}

// ValidOrigin reports whether origin is a vault.ParseOrigin origin of at most MaxOriginLength characters, as that parse writes it.
func ValidOrigin(origin string) bool {
	parsed, ok := vault.ParseOrigin(origin)
	return ok && parsed == origin && validText(origin, MaxOriginLength)
}

// ValidSite reports whether site is UTF-8 of 1 to MaxSiteLength characters.
func ValidSite(site string) bool {
	return validText(site, MaxSiteLength)
}

func validText(value string, limit int) bool {
	count := utf8.RuneCountInString(value)
	return utf8.ValidString(value) && count >= 1 && count <= limit
}

// Request is one session request; fields its Type does not use are empty.
type Request struct {
	ID         int64
	Type       string
	Origin     string
	Purpose    Purpose
	Credential string
	Site       string
	Identity   string
	File       string
	Account    string
	Password   string
	Current    string
	Pending    string
	Target     string
	Name       string
	Passkey    PasskeyFields
	Card       CardFields
}

// ParseRequest reads a session request, which must carry an integer id and a type.
func ParseRequest(plaintext []byte) (Request, error) {
	var fields struct {
		ID         *int64  `json:"id"`
		Type       string  `json:"type"`
		Origin     string  `json:"origin"`
		Purpose    Purpose `json:"purpose"`
		Credential string  `json:"credential"`
		Site       string  `json:"site"`
		Identity   string  `json:"identity"`
		File       string  `json:"file"`
		Account    string  `json:"account"`
		Password   string  `json:"password"`
		Current    string  `json:"current"`
		Pending    string  `json:"pending"`
		Target     string  `json:"target"`
		Name       string  `json:"name"`
		PasskeyFields
		CardFields
	}
	if err := json.Unmarshal(plaintext, &fields); err != nil || fields.ID == nil || fields.Type == "" {
		return Request{}, ErrMalformed
	}
	return Request{
		ID: *fields.ID, Type: fields.Type, Origin: fields.Origin, Purpose: fields.Purpose,
		Credential: fields.Credential, Site: fields.Site, Identity: fields.Identity, File: fields.File,
		Account: fields.Account, Password: fields.Password, Current: fields.Current,
		Pending: fields.Pending, Target: fields.Target, Name: fields.Name, Passkey: fields.PasskeyFields,
		Card: fields.CardFields,
	}, nil
}

// SuggestPurpose is the purpose of a suggest request, false for none or an unknown one.
func (r Request) SuggestPurpose() (Purpose, bool) {
	switch r.Purpose {
	case PurposeSignIn, PurposeCode:
		return r.Purpose, true
	default:
		return "", false
	}
}

// Status is the result of a status request.
type Status struct {
	Vault string `json:"vault"`
}

// Suggestions is the result of a suggest request.
type Suggestions struct {
	Credentials []Suggestion `json:"credentials"`
}

// Suggestion is a credential matching the requested origin; only a code suggestion carries Digits and Period.
type Suggestion struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Account string `json:"account"`
	Site    string `json:"site"`
	Exact   bool   `json:"exact"`
	Digits  int    `json:"digits,omitempty"`
	Period  int    `json:"period,omitempty"`
	// Tags tell two accounts on one site apart; an extension that knows none ignores them.
	Tags []string `json:"tags,omitempty"`
}

// Fill is the result of a fill request.
type Fill struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// OneTimeCode is the result of a code request; Period is in seconds and ExpiresAt in Unix milliseconds.
type OneTimeCode struct {
	Code      string `json:"code"`
	Digits    int    `json:"digits"`
	Period    int    `json:"period"`
	ExpiresAt int64  `json:"expiresAt"`
}

// Icon is the result of an icon request: a base64 PNG and its "#rrggbb" tint, both empty for none.
type Icon struct {
	Image string `json:"image"`
	Tint  string `json:"tint"`
}

// CaptureState is what the desktop app makes of a captured password.
type CaptureState string

// Capture states: an offer ready to save, nothing to offer, or an offer waiting for unlock.
const (
	CaptureReady  CaptureState = "ready"
	CaptureNone   CaptureState = "none"
	CaptureLocked CaptureState = "locked"
)

// SaveAction is what saving a capture to an existing credential changes in it.
type SaveAction string

// Save actions: replace the password, or add the capture's origin to the websites.
const (
	SaveUpdate  SaveAction = "update"
	SaveAddSite SaveAction = "add-site"
)

// CaptureOffer is the result of a capture, card capture or review request; an empty Suggested means a new item. Only
// a card's offer carries Card, and its Account is empty.
type CaptureOffer struct {
	Kind      CaptureKind  `json:"kind"`
	State     CaptureState `json:"state"`
	Pending   string       `json:"pending,omitempty"`
	Site      string       `json:"site"`
	Account   string       `json:"account"`
	Name      string       `json:"name"`
	Card      *CardFace    `json:"card,omitempty"`
	Targets   []SaveTarget `json:"targets"`
	Suggested string       `json:"suggested"`
}

// SaveTarget is a credential or a card a capture can go to. Account is a credential's login, else its email; a card's
// is empty.
type SaveTarget struct {
	Credential string     `json:"credential"`
	Label      string     `json:"label"`
	Account    string     `json:"account"`
	Action     SaveAction `json:"action"`
	Tags       []string   `json:"tags,omitempty"`
}

// Saved is the result of a save request: whether it created a credential or updated one.
type Saved struct {
	Saved SavedAs `json:"saved"`
}

// SavedAs is how a save stored a capture.
type SavedAs string

// Outcomes of a save.
const (
	SavedCreated SavedAs = "created"
	SavedUpdated SavedAs = "updated"
)

// Identities is the result of an identities request.
type Identities struct {
	Identities []Identity `json:"identities"`
}

// Identity is an identity with its files; Thumbnail is a base64 JPEG, empty for none.
type Identity struct {
	ID        string         `json:"id"`
	Label     string         `json:"label"`
	Thumbnail string         `json:"thumbnail"`
	Files     []IdentityFile `json:"files"`
}

// FileKind tells an identity's photo from a document's scan.
type FileKind string

// Kinds of identity file.
const (
	FilePhoto FileKind = "photo"
	FileScan  FileKind = "scan"
)

// IdentityFile is one file an identity holds; Thumbnail is a base64 JPEG, empty for a PDF.
type IdentityFile struct {
	ID        string    `json:"id"`
	Kind      FileKind  `json:"kind"`
	Name      string    `json:"name"`
	MediaType string    `json:"mediaType"`
	Document  *Document `json:"document,omitempty"`
	Thumbnail string    `json:"thumbnail"`
}

// Document is the document a scan belongs to; only type other carries a Label.
type Document struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

// SharedFile is the result of a share request; the parts after the head carry its Size bytes.
type SharedFile struct {
	Name      string `json:"name"`
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
}

// Progress tells the extension what the person is asked to do while a request waits for them.
type Progress string

const (
	// ProgressConfirmOnDevice asks for device authentication; the extension matches the wire value.
	ProgressConfirmOnDevice Progress = "confirm-on-device"
	// ProgressConfirmInRavenpass asks for the vault's PIN in Ravenpass.
	ProgressConfirmInRavenpass Progress = "confirm-in-ravenpass"
)

// Response answers one request; a head carries Parts, a progress message carries Progress alone.
type Response struct {
	ID       int64    `json:"id"`
	Result   any      `json:"result,omitempty"`
	Error    string   `json:"error,omitempty"`
	Parts    int      `json:"parts,omitempty"`
	Progress Progress `json:"progress,omitempty"`
}

// Frames returns response whole, or as a head {"id":N,"parts":k} and its JSON result in k parts.
func Frames(response Response) ([][]byte, error) {
	whole, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	if len(whole) <= MaxPlaintextBytes {
		return [][]byte{whole}, nil
	}
	body, err := json.Marshal(response.Result)
	if err != nil {
		return nil, err
	}
	return headed(Response{ID: response.ID}, body)
}

// FileFrames returns a head {"id":N,"result":{...},"parts":k} and the content in k >= 1 parts.
func FileFrames(id int64, file SharedFile, content []byte) ([][]byte, error) {
	return headed(Response{ID: id, Result: file}, content)
}

// headed returns head and body in parts of at most MaxPlaintextBytes that share body's memory.
func headed(head Response, body []byte) ([][]byte, error) {
	var parts [][]byte
	for len(body) > MaxPlaintextBytes {
		parts = append(parts, body[:MaxPlaintextBytes])
		body = body[MaxPlaintextBytes:]
	}
	parts = append(parts, body)
	head.Parts = len(parts)
	encoded, err := json.Marshal(head)
	if err != nil {
		return nil, err
	}
	return append([][]byte{encoded}, parts...), nil
}
