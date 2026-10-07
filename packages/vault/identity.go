package vault

import (
	"slices"
	"strings"
	"time"
)

// DocumentType is the kind of an identity document, numbered as the vault stores it.
type DocumentType uint8

const (
	DocumentPassport       DocumentType = 1
	DocumentDriversLicense DocumentType = 2
	DocumentIDCard         DocumentType = 3
	DocumentTaxNumber      DocumentType = 4
	DocumentOther          DocumentType = 5
)

func (t DocumentType) known() bool {
	return t >= DocumentPassport && t <= DocumentOther
}

var documentTypeNames = map[DocumentType]string{
	DocumentPassport:       "passport",
	DocumentDriversLicense: "drivers-license",
	DocumentIDCard:         "id-card",
	DocumentTaxNumber:      "tax-number",
	DocumentOther:          "other",
}

// Name is the name clients exchange the type by, empty for a type this version does not know.
func (t DocumentType) Name() string {
	return documentTypeNames[t]
}

// DocumentTypeNamed is the type a name stands for, and false for a name no type has.
func DocumentTypeNamed(name string) (DocumentType, bool) {
	for documentType, known := range documentTypeNames {
		if known == name {
			return documentType, true
		}
	}
	return 0, false
}

// Identity limits count characters; label, notes and emails use the credential limits of the same name.
const (
	MaxFullNameLength       = 256
	MaxPhoneLength          = 64
	MaxPartLabelLength      = 64
	MaxStreetLength         = 512
	MaxCityLength           = 128
	MaxRegionLength         = 128
	MaxPostalCodeLength     = 32
	MaxCountryLength        = 128
	MaxDocumentNumberLength = 128
	MaxIssuerLength         = 256
	MaxIdentityEmails       = 8
	MaxIdentityPhones       = 8
	MaxIdentityAddresses    = 8
	MaxIdentityDocuments    = 16
)

// time.Parse accepts year 0000 with dateLayout, so the year is checked on its own.
const (
	dateLayout       = "2006-01-02"
	earliestDateYear = 1900
)

// ValidDate reports whether value is empty or a calendar date written YYYY-MM-DD.
func ValidDate(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := time.Parse(dateLayout, value)
	return err == nil && parsed.Year() >= earliestDateYear
}

// Address is one postal address; ID stays stable across saves and is zero for a new identity address or a card's own.
type Address struct {
	ID         ID
	Label      string
	Street     string
	City       string
	Region     string
	PostalCode string
	Country    string
}

func (a Address) parts() []string {
	return []string{a.Street, a.City, a.Region, a.PostalCode, a.Country}
}

// Document is one identity document; an edit removes stored scans left out of Scans, and Attach is empty on a read.
type Document struct {
	Type      DocumentType
	Label     string
	Number    string
	Issuer    string
	IssuedOn  string
	ExpiresOn string
	Scans     []ID
	Attach    []PreparedScan
}

// IdentityInput is one identity; Photo reads as the stored JPEG and takes those bytes unchanged or a new square PNG or JPEG.
type IdentityInput struct {
	Label     string
	FullName  string
	Birthday  string
	Emails    []string
	Phones    []string
	Addresses []Address
	Documents []Document
	Notes     string
	Photo     []byte
	Tags      []string
}

// Identity is an identity as read.
type Identity struct {
	ID ID
	IdentityInput
}

// acceptIdentity returns an identity as stored, never shortened, with empty lists nil for a deterministic encoding.
func acceptIdentity(input IdentityInput) (IdentityInput, error) {
	if !fits(input.Label, MaxLabelLength) || strings.TrimSpace(input.Label) == "" {
		return IdentityInput{}, ErrInvalidInput
	}
	tags, err := AcceptTags(input.Tags)
	if err != nil {
		return IdentityInput{}, err
	}
	input.Tags = tags
	if !fits(input.FullName, MaxFullNameLength) || !fits(input.Notes, MaxNotesLength) || !ValidDate(input.Birthday) {
		return IdentityInput{}, ErrInvalidInput
	}
	if !acceptValues(input.Emails, MaxIdentityEmails, MaxEmailLength) || !acceptValues(input.Phones, MaxIdentityPhones, MaxPhoneLength) {
		return IdentityInput{}, ErrInvalidInput
	}
	if len(input.Addresses) > MaxIdentityAddresses || len(input.Documents) > MaxIdentityDocuments {
		return IdentityInput{}, ErrInvalidInput
	}
	for _, address := range input.Addresses {
		if !acceptAddress(address) {
			return IdentityInput{}, ErrInvalidInput
		}
	}
	input.Documents = slices.Clone(input.Documents)
	for i, document := range input.Documents {
		if !acceptDocument(document) {
			return IdentityInput{}, ErrInvalidInput
		}
		if len(document.Scans)+len(document.Attach) > MaxDocumentScans {
			return IdentityInput{}, ErrScanLimit
		}
		input.Documents[i].Scans = nilIfEmpty(slices.Clone(document.Scans))
		input.Documents[i].Attach = nilIfEmpty(slices.Clone(document.Attach))
	}
	input.Emails = nilIfEmpty(input.Emails)
	input.Phones = nilIfEmpty(input.Phones)
	input.Addresses = nilIfEmpty(slices.Clone(input.Addresses))
	input.Documents = nilIfEmpty(input.Documents)
	input.Photo = nilIfEmpty(input.Photo)
	return input, nil
}

func acceptValues(values []string, count, length int) bool {
	if len(values) > count {
		return false
	}
	for _, value := range values {
		if !fits(value, length) || strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func acceptAddress(address Address) bool {
	limits := []int{MaxStreetLength, MaxCityLength, MaxRegionLength, MaxPostalCodeLength, MaxCountryLength}
	filled := false
	for i, part := range address.parts() {
		if !fits(part, limits[i]) {
			return false
		}
		filled = filled || strings.TrimSpace(part) != ""
	}
	return filled && fits(address.Label, MaxPartLabelLength)
}

// assignAddressIDs gives each new address a fresh id and refuses an id outside held, the stored set, or one given twice.
func assignAddressIDs(addresses []Address, held map[ID]struct{}) error {
	given := make(map[ID]struct{}, len(addresses))
	for _, address := range addresses {
		if address.ID == (ID{}) {
			continue
		}
		if _, ok := held[address.ID]; !ok {
			return ErrInvalidInput
		}
		if _, repeated := given[address.ID]; repeated {
			return ErrInvalidInput
		}
		given[address.ID] = struct{}{}
	}
	for i := range addresses {
		if addresses[i].ID != (ID{}) {
			continue
		}
		id, err := uniqueID(func(candidate ID) bool {
			_, stored := held[candidate]
			_, taken := given[candidate]
			return stored || taken || candidate == ID{}
		})
		if err != nil {
			return err
		}
		given[id] = struct{}{}
		addresses[i].ID = id
	}
	return nil
}

func addressIDs(addresses []Address) map[ID]struct{} {
	ids := make(map[ID]struct{}, len(addresses))
	for _, address := range addresses {
		ids[address.ID] = struct{}{}
	}
	return ids
}

func acceptDocument(document Document) bool {
	if !document.Type.known() {
		return false
	}
	if !fits(document.Number, MaxDocumentNumberLength) || strings.TrimSpace(document.Number) == "" {
		return false
	}
	if !fits(document.Label, MaxPartLabelLength) || !fits(document.Issuer, MaxIssuerLength) {
		return false
	}
	// Only an other document is named by its label; a typed one is named by its type.
	if document.Type == DocumentOther && strings.TrimSpace(document.Label) == "" || document.Type != DocumentOther && document.Label != "" {
		return false
	}
	if !ValidDate(document.IssuedOn) || !ValidDate(document.ExpiresOn) {
		return false
	}
	// YYYY-MM-DD dates sort as text in calendar order.
	return document.IssuedOn == "" || document.ExpiresOn == "" || document.IssuedOn <= document.ExpiresOn
}

func nilIfEmpty[T any](values []T) []T {
	if len(values) == 0 {
		return nil
	}
	return values
}

// identityEntry is what the index shows of an identity, with the earliest expiry among its documents.
func identityEntry(input IdentityInput, groups []ID, thumbnail []byte) entryMeta {
	entry := entryMeta{kind: KindIdentity, label: input.Label, groups: groups, thumbnail: thumbnail, tags: input.Tags}
	if len(input.Emails) > 0 {
		entry.detail = input.Emails[0]
	}
	for _, document := range input.Documents {
		if document.ExpiresOn != "" && (entry.expiresOn == "" || document.ExpiresOn < entry.expiresOn) {
			entry.expiresOn = document.ExpiresOn
		}
	}
	return entry
}

// An identity record holds everything but the label, which lives in the index:
//
//	[9, fullName, birthday, [email…], [phone…], [address…], [document…], notes, photo]
//	address   [addressID, label, street, city, region, postalCode, country]
//	document  [type, label, number, issuer, issuedOn, expiresOn, [attachmentID…]]
const recordSchemaIdentity = 9

type identityRecord struct {
	_         struct{} `cbor:",toarray"`
	Schema    uint64
	FullName  string
	Birthday  string
	Emails    []string
	Phones    []string
	Addresses []wireIdentityAddress
	Documents []wireDocument
	Notes     string
	Photo     []byte
}

// A card's own address has no id: [label, street, city, region, postalCode, country].
type wireAddress struct {
	_          struct{} `cbor:",toarray"`
	Label      string
	Street     string
	City       string
	Region     string
	PostalCode string
	Country    string
}

type wireIdentityAddress struct {
	_  struct{} `cbor:",toarray"`
	ID ID
	wireAddress
}

type wireDocument struct {
	_         struct{} `cbor:",toarray"`
	Type      uint64
	Label     string
	Number    string
	Issuer    string
	IssuedOn  string
	ExpiresOn string
	Scans     []ID
}

func wireAddressOf(address Address) wireAddress {
	return wireAddress{Label: address.Label, Street: address.Street, City: address.City, Region: address.Region, PostalCode: address.PostalCode, Country: address.Country}
}

// address is the address this wire address holds, refused when its text is not UTF-8.
func (w wireAddress) address(id ID) (Address, error) {
	address := Address{ID: id, Label: w.Label, Street: w.Street, City: w.City, Region: w.Region, PostalCode: w.PostalCode, Country: w.Country}
	if !validText(address.Label, address.Street, address.City, address.Region, address.PostalCode, address.Country) {
		return Address{}, ErrMalformed
	}
	return address, nil
}

func encodeIdentityRecord(input IdentityInput) ([]byte, error) {
	record := identityRecord{
		Schema: recordSchemaIdentity, FullName: input.FullName, Birthday: input.Birthday, Emails: input.Emails, Phones: input.Phones,
		Addresses: make([]wireIdentityAddress, len(input.Addresses)), Documents: make([]wireDocument, len(input.Documents)),
		Notes: input.Notes, Photo: input.Photo,
	}
	for i, address := range input.Addresses {
		record.Addresses[i] = wireIdentityAddress{ID: address.ID, wireAddress: wireAddressOf(address)}
	}
	for i, document := range input.Documents {
		record.Documents[i] = wireDocument{
			Type: uint64(document.Type), Label: document.Label, Number: document.Number, Issuer: document.Issuer,
			IssuedOn: document.IssuedOn, ExpiresOn: document.ExpiresOn, Scans: document.Scans,
		}
	}
	return sealableRecord(marshal(record, 0))
}

func decodeIdentityRecord(plaintext []byte) (IdentityInput, error) {
	if err := readSchema(plaintext, recordSchemaIdentity); err != nil {
		return IdentityInput{}, err
	}
	var record identityRecord
	if err := unmarshal(plaintext, &record); err != nil {
		return IdentityInput{}, err
	}
	if len(record.Emails) > MaxIdentityEmails || len(record.Phones) > MaxIdentityPhones || len(record.Photo) > maxPhotoBytes ||
		!validText(slices.Concat([]string{record.FullName, record.Birthday, record.Notes}, record.Emails, record.Phones)...) {
		return IdentityInput{}, ErrMalformed
	}
	input := IdentityInput{FullName: record.FullName, Birthday: record.Birthday, Emails: nilIfEmpty(record.Emails), Phones: nilIfEmpty(record.Phones), Notes: record.Notes, Photo: nilIfEmpty(record.Photo)}
	var err error
	if input.Addresses, err = parseIdentityAddresses(record.Addresses); err != nil {
		return IdentityInput{}, err
	}
	if input.Documents, err = parseDocuments(record.Documents); err != nil {
		return IdentityInput{}, err
	}
	return input, nil
}

// parseIdentityAddresses reads an identity's addresses, refusing a zero or repeated id.
func parseIdentityAddresses(wires []wireIdentityAddress) ([]Address, error) {
	if len(wires) > MaxIdentityAddresses {
		return nil, ErrMalformed
	}
	addresses := make([]Address, len(wires))
	seen := make(map[ID]struct{}, len(wires))
	for i, wire := range wires {
		if _, repeated := seen[wire.ID]; repeated || wire.ID == (ID{}) {
			return nil, ErrMalformed
		}
		seen[wire.ID] = struct{}{}
		address, err := wire.address(wire.ID)
		if err != nil {
			return nil, err
		}
		addresses[i] = address
	}
	return nilIfEmpty(addresses), nil
}

func parseDocuments(wires []wireDocument) ([]Document, error) {
	if len(wires) > MaxIdentityDocuments {
		return nil, ErrMalformed
	}
	documents := make([]Document, len(wires))
	for i, wire := range wires {
		documentType := DocumentType(wire.Type)
		if wire.Type > uint64(DocumentOther) || !documentType.known() {
			return nil, ErrUnsupported
		}
		if len(wire.Scans) > MaxDocumentScans || !validText(wire.Label, wire.Number, wire.Issuer, wire.IssuedOn, wire.ExpiresOn) {
			return nil, ErrMalformed
		}
		documents[i] = Document{Type: documentType, Label: wire.Label, Number: wire.Number, Issuer: wire.Issuer, IssuedOn: wire.IssuedOn, ExpiresOn: wire.ExpiresOn, Scans: nilIfEmpty(wire.Scans)}
	}
	return nilIfEmpty(documents), nil
}

// ReadSelectedIdentity decrypts the selected identity.
func (s *Session) ReadSelectedIdentity(ticket Selection) (Identity, error) {
	var input IdentityInput
	entry, err := s.readSelected(ticket, KindIdentity, func(plaintext []byte) (err error) {
		input, err = decodeIdentityRecord(plaintext)
		return err
	})
	if err != nil {
		return Identity{}, err
	}
	input.Label = entry.label
	input.Tags = slices.Clone(entry.tags)
	return Identity{ID: entry.id, IdentityInput: input}, nil
}

// PrepareCreateIdentity prepares a new identity in groups, with the scans its documents attach.
func (s *Session) PrepareCreateIdentity(input IdentityInput, groups []ID) (*Pending, ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, ID{}, err
	}
	input, err := acceptIdentity(input)
	if err != nil {
		return nil, ID{}, err
	}
	// A stored scan belongs to an identity that exists, so a new one names none.
	if len(scansOf(input.Documents)) > 0 {
		return nil, ID{}, ErrInvalidInput
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, ID{}, err
	}
	if err := assignAddressIDs(input.Addresses, nil); err != nil {
		return nil, ID{}, err
	}
	var thumbnail []byte
	if input.Photo, thumbnail, err = acceptPhoto(input.Photo, nil, nil); err != nil {
		return nil, ID{}, err
	}
	reserved := make(map[ID]struct{})
	id, err := s.freshID(reserved)
	if err != nil {
		return nil, ID{}, err
	}
	reserved[id] = struct{}{}
	scanEntries, scanRecords, err := s.attachPrepared(id, input.Documents, reserved)
	if err != nil {
		return nil, ID{}, err
	}
	plaintext, err := encodeIdentityRecord(input)
	if err != nil {
		return nil, ID{}, err
	}
	defer clear(plaintext)
	entry, box, err := s.sealFirst(id, identityEntry(input, membership, thumbnail), plaintext)
	if err != nil {
		return nil, ID{}, err
	}
	entries := append(append(append([]entryMeta(nil), s.entries...), entry), scanEntries...)
	records := append(append(append([]sealedBox(nil), s.records...), box), scanRecords...)
	pending, err := s.prepare(entries, records)
	return pending, id, err
}

func (s *Session) decryptIdentity(index int) (IdentityInput, error) {
	plaintext, err := s.openRecord(index)
	if err != nil {
		return IdentityInput{}, err
	}
	defer clear(plaintext)
	return decodeIdentityRecord(plaintext)
}

// PrepareEditIdentity replaces an identity's content and membership, keeping its pin, and drops orphaned scans and card links in the same save.
func (s *Session) PrepareEditIdentity(id ID, input IdentityInput, groups []ID) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	index := s.findKind(id, KindIdentity)
	if index < 0 {
		return nil, ErrNotFound
	}
	input, err := acceptIdentity(input)
	if err != nil {
		return nil, err
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, err
	}
	named, err := s.acceptScanReferences(id, input.Documents)
	if err != nil {
		return nil, err
	}
	stored, err := s.decryptIdentity(index)
	if err != nil {
		return nil, err
	}
	held := addressIDs(stored.Addresses)
	if err := assignAddressIDs(input.Addresses, held); err != nil {
		return nil, err
	}
	var thumbnail []byte
	if input.Photo, thumbnail, err = acceptPhoto(input.Photo, stored.Photo, s.entries[index].thumbnail); err != nil {
		return nil, err
	}
	scanEntries, scanRecords, err := s.attachPrepared(id, input.Documents, make(map[ID]struct{}))
	if err != nil {
		return nil, err
	}
	plaintext, err := encodeIdentityRecord(input)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	entry, box, err := s.sealNext(index, identityEntry(input, membership, thumbnail), plaintext)
	if err != nil {
		return nil, err
	}
	entries := append([]entryMeta(nil), s.entries...)
	records := append([]sealedBox(nil), s.records...)
	kept := addressIDs(input.Addresses)
	removed := slices.ContainsFunc(stored.Addresses, func(address Address) bool {
		_, still := kept[address.ID]
		return !still
	})
	if removed {
		err := s.unlinkCards(entries, records, func(link AddressLink) bool {
			_, still := kept[link.Address]
			return link.Identity == id && !still
		})
		if err != nil {
			return nil, err
		}
	}
	entries[index], records[index] = entry, box
	entries, records = withoutItems(entries, records, func(candidate entryMeta) bool {
		_, kept := named[candidate.id]
		return candidate.kind == KindAttachment && candidate.owner == id && !kept
	})
	entries = append(entries, scanEntries...)
	records = append(records, scanRecords...)
	return s.prepare(entries, records)
}
