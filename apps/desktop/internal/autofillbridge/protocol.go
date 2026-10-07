package autofillbridge

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/dortanes/ravenpass/packages/app/autofill"
)

// One connection carries one exchange of framed JSON messages:
//
//	frame:     | length: uint32, big-endian | JSON: length bytes |
//	extension: request
//	app:       [waitNotice] answer
//
// A waitNotice precedes the answer of a request that waits for the owner.

// The operations a request names.
const (
	opStatus        = "status"
	opSearch        = "search"
	opPassword      = "password"
	opCode          = "code"
	opUnlock        = "unlock"
	opAddSite       = "add-site"
	opPasskeys      = "passkeys"
	opPasskeySign   = "passkey-sign"
	opPasskeyCreate = "passkey-create"
	opIcon          = "icon"
	opLanguage      = "language"
	opAppearance    = "appearance"
)

// The refusals an answer names.
const (
	refusedLocked       = "locked"
	refusedNotFound     = "not-found"
	refusedNoMatch      = "no-match"
	refusedNoCode       = "no-code"
	refusedInvalid      = "invalid"
	refusedFailed       = "failed"
	refusedDeclined     = "declined"
	refusedUnverifiable = "unverifiable"
	refusedExcluded     = "excluded"
	refusedUnsupported  = "unsupported"
)

// The kinds of service identifier the system passes the extension.
const (
	serviceURL    = "url"
	serviceDomain = "domain"
)

const (
	maxRequestBytes = 64 << 10
	// maxAnswerBytes fits fifty suggestions or one siteicons.IconSize PNG.
	maxAnswerBytes = 1 << 20
	maxServices    = 8
)

var errFrame = errors.New("message is empty or larger than allowed")

// request is what the extension asks. Services are most specific first.
type request struct {
	Op             string                    `json:"op"`
	Services       []service                 `json:"services,omitempty"`
	Codes          bool                      `json:"codes,omitempty"`
	ID             string                    `json:"id,omitempty"`
	Query          string                    `json:"query,omitempty"`
	Site           string                    `json:"site,omitempty"`
	RPID           string                    `json:"rpID,omitempty"`
	Allowed        [][]byte                  `json:"allowed,omitempty"`
	ClientDataHash []byte                    `json:"clientDataHash,omitempty"`
	CredentialID   []byte                    `json:"credentialID,omitempty"`
	User           passkeyUser               `json:"user,omitzero"`
	Algorithms     []int                     `json:"algorithms,omitempty"`
	Excluded       [][]byte                  `json:"excluded,omitempty"`
	Verification   autofill.UserVerification `json:"verification,omitempty"`
}

type passkeyUser struct {
	Handle []byte `json:"handle"`
	Name   string `json:"name"`
}

type service struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// waitNotice names the seconds the app may take to send the answer that follows.
type waitNotice struct {
	Wait int64 `json:"wait"`
}

func noticeOf(wait time.Duration) waitNotice {
	return waitNotice{Wait: int64(wait / time.Second)}
}

// answer is the app's reply; an Error answer sets nothing else, and Site is ASCII so a Unicode look-alike cannot pose as another host.
type answer struct {
	Error             string          `json:"error,omitempty"`
	Site              string          `json:"site,omitempty"`
	Open              bool            `json:"open,omitempty"`
	Scope             autofill.Scope  `json:"scope,omitempty"`
	Suggestions       []suggestion    `json:"suggestions,omitempty"`
	User              string          `json:"user,omitempty"`
	Password          string          `json:"password,omitempty"`
	Code              string          `json:"code,omitempty"`
	Passkeys          []passkeyChoice `json:"passkeys,omitempty"`
	CredentialID      []byte          `json:"credentialID,omitempty"`
	AuthenticatorData []byte          `json:"authenticatorData,omitempty"`
	Signature         []byte          `json:"signature,omitempty"`
	UserHandle        []byte          `json:"userHandle,omitempty"`
	AttestationObject []byte          `json:"attestationObject,omitempty"`
	Image             string          `json:"image,omitempty"`
	Tint              string          `json:"tint,omitempty"`
	Languages         []string        `json:"languages,omitempty"`
	Language          string          `json:"language,omitempty"`
	Chosen            bool            `json:"chosen,omitempty"`
	Appearance        string          `json:"appearance,omitempty"`
}

// suggestion is a listed credential; Matches is false until the owner adds the most specific service to it.
type suggestion struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Account string   `json:"account"`
	Site    string   `json:"site"`
	Matches bool     `json:"matches"`
	Tags    []string `json:"tags,omitempty"`
}

type passkeyChoice struct {
	ID           string `json:"id"`
	CredentialID []byte `json:"credentialID"`
	Account      string `json:"account"`
	Label        string `json:"label"`
}

// readMessage decodes one frame of at most limit bytes of JSON into v.
func readMessage(r io.Reader, limit int, v any) error {
	var length [4]byte
	if _, err := io.ReadFull(r, length[:]); err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(length[:])
	if size == 0 || size > uint32(limit) {
		return errFrame
	}
	body := make([]byte, size)
	defer clear(body)
	if _, err := io.ReadFull(r, body); err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// writeMessage sends v as one frame, or writes nothing and returns errFrame when v's JSON exceeds limit.
func writeMessage(w io.Writer, limit int, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	defer clear(body)
	if len(body) > limit {
		return errFrame
	}
	frame := binary.BigEndian.AppendUint32(make([]byte, 0, 4+len(body)), uint32(len(body)))
	frame = append(frame, body...)
	defer clear(frame)
	_, err = w.Write(frame)
	return err
}
