package vault

import (
	"encoding/hex"
	"errors"
)

// Field limits are counted in characters, not bytes.
const (
	MaxLabelLength     = 128
	MaxOriginLength    = 255
	MaxLoginLength     = 255
	MaxEmailLength     = 320
	MaxPasswordLength  = 1024
	MaxNotesLength     = 8192
	MaxTOTPLength      = 512
	MaxGroupNameLength = 64
	MaxPackageLength   = 255
)

// MaxCredentialWebsites bounds the websites one credential holds.
const MaxCredentialWebsites = 16

// MaxCredentialApps bounds the app links one credential holds.
const MaxCredentialApps = 16

// MaxCredentialPasskeys bounds the passkeys one credential holds.
const MaxCredentialPasskeys = 8

// MaxContainerBytes is the largest vault file the core reads or writes, and the bound for storage adapters.
const MaxContainerBytes = 256 << 20

const (
	maxIndexBytes  = 16 << 20
	maxRecordBytes = 1 << 20
	maxEntries     = 100_000
	// MaxGroups bounds the groups a vault holds.
	MaxGroups = 100
	// MaxCredentialGroups bounds the groups one item belongs to.
	MaxCredentialGroups = 32
)

// Errors the vault reports; compare with errors.Is.
var (
	ErrAuthentication         = errors.New("vault authentication failed")
	ErrMalformed              = errors.New("vault data is malformed")
	ErrUnsupported            = errors.New("vault format is unsupported")
	ErrResourceLimit          = errors.New("vault exceeds supported limits")
	ErrInvalidInput           = errors.New("vault item input is invalid")
	ErrInvalidTOTP            = errors.New("one-time code setup cannot produce a code")
	ErrScanLimit              = errors.New("document holds as many scans as it can")
	ErrNoTOTP                 = errors.New("credential has no one-time code setup")
	ErrPasskeysFull           = errors.New("credential holds as many passkeys as it can")
	ErrInvalidPhrase          = errors.New("recovery phrase is invalid")
	ErrInvalidPIN             = errors.New("PIN must be 6 to 12 digits")
	ErrLocked                 = errors.New("vault is locked")
	ErrNotFound               = errors.New("vault item was not found")
	ErrStaleSelection         = errors.New("vault item selection is no longer current")
	ErrPendingCommit          = errors.New("vault save is awaiting confirmation")
	ErrStaleCommit            = errors.New("vault save no longer matches the session")
	ErrWitnessMissing         = errors.New("local vault witness is missing")
	ErrWitnessMismatch        = errors.New("local vault copy does not match its witness")
	ErrWitnessOlder           = errors.New("vault copy is older than the version this device acknowledged")
	ErrWitnessDiverged        = errors.New("vault copy does not descend from the version this device acknowledged")
	ErrWitnessAdvanceRequired = errors.New("local vault witness update is required")
	// ErrKeyReplaced reports a device envelope or open session holding a vault key the vault no longer uses; its recovery phrase opens it.
	ErrKeyReplaced = errors.New("vault key was replaced")
)

// ID identifies a vault, an item, a group or an address.
type ID [16]byte

// String is the ID in lower-case hexadecimal.
func (id ID) String() string { return hex.EncodeToString(id[:]) }

// ParseID reads an ID written as String writes it.
func ParseID(value string) (ID, error) {
	var id ID
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != len(id) {
		return ID{}, ErrInvalidInput
	}
	copy(id[:], decoded)
	return id, nil
}

// CredentialInput is one credential; each list keeps its order and is nil when empty.
type CredentialInput struct {
	Label    string
	Websites []string
	Login    string
	Email    string
	Password string
	Notes    string
	// TOTP is the one-time code setup as NormalizeTOTP returns it, empty for none.
	TOTP     string
	Passkeys []Passkey
	Apps     []App
	// Tags tell the item apart from others like it; they live in the index beside the label.
	Tags []string
}

// CredentialPatch changes the fields it sets and keeps the others.
type CredentialPatch struct {
	Label    *string
	Websites *[]string
	Login    *string
	Email    *string
	Password *string
	Notes    *string
	TOTP     *string
	Apps     *[]App
	Groups   *[]ID
	Tags     *[]string
	// RemovePasskeys names passkeys to drop; one the credential does not hold is ErrNotFound.
	RemovePasskeys [][]byte
}

// Credential is a credential as read.
type Credential struct {
	ID ID
	CredentialInput
}

// Kind names what a vault item holds; an attachment is an unlisted document scan changed only through its identity.
type Kind uint8

const (
	KindCredential Kind = 1
	KindIdentity   Kind = 2
	KindAttachment Kind = 3
	KindCard       Kind = 4
	KindNote       Kind = 5
	KindSeed       Kind = 6
)

// Entry is what the list shows of an item without decrypting it; each field past Detail is zero outside the kinds its comment names.
type Entry struct {
	ID    ID
	Kind  Kind
	Label string
	// Detail is a login, an identity's first email, a bank name, a note preview or a wallet name.
	Detail string
	// ExpiresOn is an identity's earliest document expiry or the last day of a card's expiry month.
	ExpiresOn string
	// Thumbnail is an identity's photo as a small JPEG.
	Thumbnail []byte
	// Site is a credential's CredentialInput.Site or SiteOf a card's bank site.
	Site string
	// Sites, Email, Code, Passkeys and Apps belong to a credential; Sites[0] is its Site.
	Sites    []string
	Email    string
	Code     CodeFace
	Passkeys []PasskeyFace
	Apps     []App
	Card     CardFace
	Note     NoteFace
	Seed     SeedFace
	Pinned   bool
	Groups   []ID
	Tags     []string
	// DeletedAt is when the item moved to the trash, in Unix milliseconds; zero outside the trash.
	DeletedAt uint64
}

// Group is a named set of items; it lives in the index.
type Group struct {
	ID   ID
	Name string
}

// Head names one version of a vault file.
type Head struct {
	VaultID      ID
	Revision     uint64
	Hash         [32]byte
	PreviousHash [32]byte
}

// Witness is the version of a vault a device acknowledged.
type Witness struct {
	VaultID  ID
	Revision uint64
	Hash     [32]byte
}

// WitnessDecision is how a vault file stands against a device's witness.
type WitnessDecision uint8

const (
	WitnessEqual WitnessDecision = iota + 1
	WitnessAdvance
)

// Selection is a ticket to read the opened item; a lock, a save or another selection makes it stale.
type Selection struct {
	Generation uint64
	Token      uint64
	ID         ID
}

// Created is a new vault: its file, its recovery phrase and an open session.
type Created struct {
	Container      []byte
	RecoveryPhrase string
	Session        *Session
}

// Pending is a prepared save; Commit makes it the session's version once its container is stored.
type Pending struct {
	session    *Session
	parentHash [32]byte
	entries    []entryMeta
	groups     []Group
	retention  int
	records    []sealedBox
	container  []byte
	head       Head
	ancestry   ancestry
	// rekeyed holds the keys the save seals the vault under, nil when it keeps the session's.
	rekeyed *contentKeys
}

// release drops what the save holds and clears any keys it carries.
func (p *Pending) release() {
	p.session = nil
	p.container = nil
	p.records = nil
	p.entries = nil
	p.groups = nil
	p.ancestry = nil
	if p.rekeyed != nil {
		p.rekeyed.clear()
		p.rekeyed = nil
	}
}

// Container is a copy of the file the save writes.
func (p *Pending) Container() []byte {
	if p == nil {
		return nil
	}
	return append([]byte(nil), p.container...)
}

// Head is the version the save writes.
func (p *Pending) Head() Head {
	if p == nil {
		return Head{}
	}
	return p.head
}
