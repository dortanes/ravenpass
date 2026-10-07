package vault

import (
	"bytes"
	"crypto/sha256"
	"math"
	"slices"

	"golang.org/x/crypto/chacha20poly1305"
)

// entryMeta is what the index holds of one item; each field past groups is zero outside the kinds its comment names.
type entryMeta struct {
	id        ID
	revision  uint64
	label     string
	detail    string
	pinned    bool
	digest    [32]byte
	groups    []ID
	kind      Kind
	expiresOn string
	thumbnail []byte
	// site belongs to a credential and a card.
	site string
	// sites, email, code, passkeys and apps belong to a credential; sites[0] is its site.
	sites    []string
	email    string
	code     CodeFace
	passkeys []PasskeyFace
	apps     []App
	// owner belongs to an attachment.
	owner ID
	card  CardFace
	note  NoteFace
	seed  SeedFace
	tags  []string
}

// An index is [revision, previous, [entry…], [group…], [earlier…]]:
//
//	previous  hash of the revision before, zero for the first revision
//	earlier   hashes of the revisions before previous, latest first
//	group     [id, name]
//	entry     [id, revision, label, detail, pinned, digest, [groupID…], kind, expiresOn, thumbnail,
//	           site, owner, email, card, summary, [site…], digits, period, [face…], [app…], [tag…]?]
//	tag       written only for an entry with tags, so a vault without any reads as before tags
//	card      [network, lastFour, color] for a card, [] otherwise
//	summary   [hidden] for a note, [format, total, used] for a seed, [] otherwise
//	face      [credentialID, rpID, userName, discoverable, userHandle, userDisplayName]
//	app       [package, signer]
type wireIndex struct {
	_        struct{} `cbor:",toarray"`
	Revision uint64
	Previous [32]byte
	Entries  []wireEntry
	Groups   []wireGroup
	Earlier  [][32]byte
}

// wireEntry is an entry with its tags, written as a twenty-first element only when there are any.
type wireEntry struct {
	wireFields
	Tags []string
}

type untaggedEntry struct {
	_ struct{} `cbor:",toarray"`
	wireFields
}

type taggedEntry struct {
	_ struct{} `cbor:",toarray"`
	wireFields
	Tags []string
}

// untaggedHead is the CBOR head of an entry without tags: an array of twenty elements.
const untaggedHead = 0x80 | 20

// MarshalCBOR writes the entry without its tags element when it has none.
func (w wireEntry) MarshalCBOR() ([]byte, error) {
	if len(w.Tags) == 0 {
		return encoding.Marshal(untaggedEntry{wireFields: w.wireFields})
	}
	return encoding.Marshal(taggedEntry{wireFields: w.wireFields, Tags: w.Tags})
}

// UnmarshalCBOR reads an entry of either length; an empty tags element is not canonical and fails the re-encoding check.
func (w *wireEntry) UnmarshalCBOR(data []byte) error {
	if len(data) > 0 && data[0] == untaggedHead {
		var entry untaggedEntry
		if err := decoding.Unmarshal(data, &entry); err != nil {
			return err
		}
		*w = wireEntry{wireFields: entry.wireFields}
		return nil
	}
	var entry taggedEntry
	if err := decoding.Unmarshal(data, &entry); err != nil {
		return err
	}
	*w = wireEntry{wireFields: entry.wireFields, Tags: entry.Tags}
	return nil
}

type wireFields struct {
	ID        ID
	Revision  uint64
	Label     string
	Detail    string
	Pinned    flag
	Digest    [32]byte
	Groups    []ID
	Kind      uint64
	ExpiresOn string
	Thumbnail []byte
	Site      string
	Owner     []byte
	Email     string
	Card      cardField
	Summary   []uint64
	Sites     []string
	Digits    uint64
	Period    uint64
	Passkeys  []wirePasskeyFace
	Apps      []wireApp
}

type wireGroup struct {
	_    struct{} `cbor:",toarray"`
	ID   ID
	Name string
}

type wireCardFace struct {
	_        struct{} `cbor:",toarray"`
	Network  uint64
	LastFour string
	Color    string
}

// cardField is an entry's card field: a face for a card and an empty array for any other kind.
type cardField struct{ face *wireCardFace }

// MarshalCBOR writes the face, or an empty array for none.
func (f cardField) MarshalCBOR() ([]byte, error) {
	if f.face == nil {
		return emptyArray, nil
	}
	return encoding.Marshal(f.face)
}

// UnmarshalCBOR reads a face, or none from an empty array.
func (f *cardField) UnmarshalCBOR(data []byte) error {
	if bytes.Equal(data, emptyArray) {
		f.face = nil
		return nil
	}
	var face wireCardFace
	if err := decoding.Unmarshal(data, &face); err != nil {
		return err
	}
	f.face = &face
	return nil
}

const (
	noteSummaryFields = 1
	seedSummaryFields = 3
)

func boolFlag(set bool) uint64 {
	if set {
		return 1
	}
	return 0
}

func parseIndex(plaintext []byte, records []sealedBox) (uint64, ancestry, []entryMeta, []Group, error) {
	var index wireIndex
	if err := unmarshal(plaintext, &index); err != nil {
		return 0, nil, nil, nil, err
	}
	if index.Revision == 0 || index.Revision == 1 && index.Previous != ([32]byte{}) || len(index.Entries) != len(records) {
		return 0, nil, nil, nil, ErrMalformed
	}
	if len(index.Groups) > MaxGroups || len(index.Earlier) > maxAncestry-1 {
		return 0, nil, nil, nil, ErrResourceLimit
	}
	entries := make([]entryMeta, len(index.Entries))
	kinds := make(map[ID]Kind, len(entries))
	for i, wire := range index.Entries {
		entry, err := wire.meta(records[i])
		if err != nil {
			return 0, nil, nil, nil, err
		}
		if _, duplicate := kinds[entry.id]; duplicate {
			return 0, nil, nil, nil, ErrMalformed
		}
		kinds[entry.id] = entry.kind
		entries[i] = entry
	}
	for _, entry := range entries {
		if entry.kind == KindAttachment && kinds[entry.owner] != KindIdentity {
			return 0, nil, nil, nil, ErrMalformed
		}
	}
	groups, err := parseGroups(index.Groups)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	ancestors := append(ancestry{index.Previous}, index.Earlier...)
	if index.Revision == 1 {
		ancestors = ancestors[1:]
	}
	// No container names more revisions than came before it.
	if uint64(len(ancestors)) >= index.Revision {
		return 0, nil, nil, nil, ErrMalformed
	}
	ancestors = nilIfEmpty(ancestors)
	for _, entry := range entries {
		for _, group := range entry.groups {
			if !slices.ContainsFunc(groups, func(held Group) bool { return held.ID == group }) {
				return 0, nil, nil, nil, ErrMalformed
			}
		}
	}
	return index.Revision, ancestors, entries, groups, nil
}

// meta is the entry this wire entry holds, whose record is record.
func (w wireEntry) meta(record sealedBox) (entryMeta, error) {
	if w.Revision == 0 || !validText(w.Label, w.Detail) {
		return entryMeta{}, ErrMalformed
	}
	entry := entryMeta{id: w.ID, revision: w.Revision, label: w.Label, detail: w.Detail, pinned: bool(w.Pinned), digest: w.Digest}
	if sha256.Sum256(encodeBox(record)) != entry.digest {
		return entryMeta{}, ErrAuthentication
	}
	var err error
	if entry.groups, err = parseMembership(w.Groups); err != nil {
		return entryMeta{}, err
	}
	if w.Kind > math.MaxUint8 || Kind(w.Kind) < KindCredential || Kind(w.Kind) > KindSeed {
		return entryMeta{}, ErrUnsupported
	}
	entry.kind = Kind(w.Kind)
	credential := entry.kind == KindCredential
	if !ValidDate(w.ExpiresOn) || entry.kind != KindIdentity && entry.kind != KindCard && w.ExpiresOn != "" {
		return entryMeta{}, ErrMalformed
	}
	entry.expiresOn = w.ExpiresOn
	if entry.kind != KindIdentity && entry.kind != KindAttachment && len(w.Thumbnail) != 0 || entry.kind == KindIdentity && len(w.Thumbnail) > maxThumbnailBytes {
		return entryMeta{}, ErrMalformed
	}
	entry.thumbnail = nilIfEmpty(w.Thumbnail)
	if len(w.Site) > MaxOriginLength || !validText(w.Site) || !credential && entry.kind != KindCard && w.Site != "" {
		return entryMeta{}, ErrMalformed
	}
	entry.site = w.Site
	if entry.kind == KindAttachment && len(w.Owner) != len(entry.owner) || entry.kind != KindAttachment && len(w.Owner) != 0 {
		return entryMeta{}, ErrMalformed
	}
	copy(entry.owner[:], w.Owner)
	if len(w.Email) > MaxEmailLength || !validText(w.Email) || !credential && w.Email != "" {
		return entryMeta{}, ErrMalformed
	}
	entry.email = w.Email
	if entry.card, err = parseCardFace(w.Card, entry.kind); err != nil {
		return entryMeta{}, err
	}
	if entry.note, entry.seed, err = parseSummary(w.Summary, entry.kind); err != nil {
		return entryMeta{}, err
	}
	if entry.sites, err = parseSites(w.Sites, entry); err != nil {
		return entryMeta{}, err
	}
	if entry.code, err = parseCodeFace(w.Digits, w.Period, entry.kind); err != nil {
		return entryMeta{}, err
	}
	if entry.passkeys, err = parsePasskeyFaces(w.Passkeys, entry.kind); err != nil {
		return entryMeta{}, err
	}
	if !credential && len(w.Apps) != 0 {
		return entryMeta{}, ErrMalformed
	}
	if entry.apps, err = parseApps(w.Apps); err != nil {
		return entryMeta{}, err
	}
	if entry.tags, err = parseTags(w.Tags, entry.kind); err != nil {
		return entryMeta{}, err
	}
	if entry.kind == KindAttachment && !validScanEntry(entry) {
		return entryMeta{}, ErrMalformed
	}
	return entry, nil
}

// parseMembership reads a membership whose ids are strictly sorted, the one encoding of a set.
func parseMembership(ids []ID) ([]ID, error) {
	if len(ids) > MaxCredentialGroups {
		return nil, ErrResourceLimit
	}
	for i := 1; i < len(ids); i++ {
		if bytes.Compare(ids[i-1][:], ids[i][:]) >= 0 {
			return nil, ErrMalformed
		}
	}
	return nilIfEmpty(ids), nil
}

func parseCardFace(field cardField, kind Kind) (CardFace, error) {
	if kind != KindCard || field.face == nil {
		if kind == KindCard || field.face != nil {
			return CardFace{}, ErrMalformed
		}
		return CardFace{}, nil
	}
	if field.face.Network > uint64(NetworkNaranja) {
		return CardFace{}, ErrUnsupported
	}
	face := CardFace{Network: CardNetwork(field.face.Network), LastFour: field.face.LastFour, Color: field.face.Color}
	if !validCardFace(face) {
		return CardFace{}, ErrMalformed
	}
	return face, nil
}

func parseSummary(summary []uint64, kind Kind) (NoteFace, SeedFace, error) {
	switch {
	case kind == KindNote && len(summary) == noteSummaryFields:
		if summary[0] > 1 {
			return NoteFace{}, SeedFace{}, ErrMalformed
		}
		return NoteFace{Hidden: summary[0] == 1}, SeedFace{}, nil
	case kind == KindSeed && len(summary) == seedSummaryFields:
		face, err := parseSeedFace(summary)
		return NoteFace{}, face, err
	case kind != KindNote && kind != KindSeed && len(summary) == 0:
		return NoteFace{}, SeedFace{}, nil
	default:
		return NoteFace{}, SeedFace{}, ErrMalformed
	}
}

// parseSeedFace reads a seed summary: [format, total, used].
func parseSeedFace(summary []uint64) (SeedFace, error) {
	format, total, used := summary[0], summary[1], summary[2]
	if format > uint64(SeedBackupCodes) {
		return SeedFace{}, ErrUnsupported
	}
	if total > max(MaxSeedWords, MaxBackupCodes) || used > total {
		return SeedFace{}, ErrMalformed
	}
	face := SeedFace{Format: SeedFormat(format), Total: int(total), Used: int(used)}
	if !validSeedFace(face) {
		return SeedFace{}, ErrMalformed
	}
	return face, nil
}

func parseSites(sites []string, entry entryMeta) ([]string, error) {
	if len(sites) > MaxCredentialWebsites || !validText(sites...) {
		return nil, ErrMalformed
	}
	if entry.kind != KindCredential {
		if len(sites) != 0 {
			return nil, ErrMalformed
		}
		return nil, nil
	}
	if len(sites) == 0 && entry.site != "" || len(sites) != 0 && sites[0] != entry.site {
		return nil, ErrMalformed
	}
	for i, site := range sites {
		if site == "" || len(site) > MaxOriginLength || slices.Contains(sites[:i], site) {
			return nil, ErrMalformed
		}
	}
	return nilIfEmpty(sites), nil
}

func parseCodeFace(digits, period uint64, kind Kind) (CodeFace, error) {
	if digits > maxCodeDigits || period > maxCodePeriod {
		return CodeFace{}, ErrMalformed
	}
	face := CodeFace{Digits: int(digits), Period: int(period)}
	if !validCodeFace(face) || kind != KindCredential && face != (CodeFace{}) {
		return CodeFace{}, ErrMalformed
	}
	return face, nil
}

func parseGroups(wires []wireGroup) ([]Group, error) {
	groups := make([]Group, len(wires))
	names := make(map[string]struct{}, len(wires))
	for i, wire := range wires {
		if slices.ContainsFunc(groups[:i], func(held Group) bool { return held.ID == wire.ID }) {
			return nil, ErrMalformed
		}
		accepted, err := AcceptGroupName(wire.Name)
		if err != nil || accepted != wire.Name {
			return nil, ErrMalformed
		}
		folded := GroupKey(wire.Name)
		if _, duplicate := names[folded]; duplicate {
			return nil, ErrMalformed
		}
		names[folded] = struct{}{}
		groups[i] = Group{ID: wire.ID, Name: wire.Name}
	}
	return nilIfEmpty(groups), nil
}

// wire is the entry as the index writes it.
func (e entryMeta) wire() wireEntry {
	wire := wireEntry{Tags: e.tags, wireFields: wireFields{
		ID: e.id, Revision: e.revision, Label: e.label, Detail: e.detail, Pinned: flag(e.pinned), Digest: e.digest,
		Groups: e.groups, Kind: uint64(e.kind), ExpiresOn: e.expiresOn, Thumbnail: e.thumbnail, Site: e.site,
		Email: e.email, Sites: e.sites, Digits: uint64(e.code.Digits), Period: uint64(e.code.Period),
		Passkeys: wirePasskeyFaces(e.passkeys), Apps: wireApps(e.apps),
	}}
	switch e.kind {
	case KindAttachment:
		wire.Owner = e.owner[:]
	case KindCard:
		wire.Card.face = &wireCardFace{Network: uint64(e.card.Network), LastFour: e.card.LastFour, Color: e.card.Color}
	case KindNote:
		wire.Summary = []uint64{boolFlag(e.note.Hidden)}
	case KindSeed:
		wire.Summary = []uint64{uint64(e.seed.Format), uint64(e.seed.Total), uint64(e.seed.Used)}
	}
	return wire
}

func encodeIndex(revision uint64, ancestors ancestry, entries []entryMeta, groups []Group) ([]byte, error) {
	if revision == 0 || len(entries) > maxEntries || len(groups) > MaxGroups {
		return nil, ErrResourceLimit
	}
	if len(ancestors) > maxAncestry || uint64(len(ancestors)) >= revision {
		return nil, ErrMalformed
	}
	index := wireIndex{Revision: revision, Previous: ancestors.previous(), Earlier: ancestors.earlier(), Entries: make([]wireEntry, len(entries)), Groups: make([]wireGroup, len(groups))}
	for i, entry := range entries {
		if len(entry.groups) > MaxCredentialGroups || len(entry.sites) > MaxCredentialWebsites || len(entry.passkeys) > MaxCredentialPasskeys || len(entry.apps) > MaxCredentialApps || len(entry.tags) > MaxItemTags {
			return nil, ErrResourceLimit
		}
		index.Entries[i] = entry.wire()
	}
	for i, group := range groups {
		index.Groups[i] = wireGroup{ID: group.ID, Name: group.Name}
	}
	encoded, err := marshal(index, 0)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxIndexBytes-chacha20poly1305.Overhead {
		clear(encoded)
		return nil, ErrResourceLimit
	}
	return encoded, nil
}
