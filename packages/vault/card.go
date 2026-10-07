package vault

import (
	"slices"
	"strings"
	"time"
)

// CardNetwork is a card's payment network, numbered as vault files store it; zero is none.
type CardNetwork uint8

const (
	NetworkVisa CardNetwork = iota + 1
	NetworkMastercard
	NetworkAmericanExpress
	NetworkDiscover
	NetworkDinersClub
	NetworkJCB
	NetworkUnionPay
	NetworkMaestro
	NetworkMir
	NetworkElo
	NetworkHiper
	NetworkHipercard
	NetworkTroy
	NetworkVerve
	NetworkNaranja
)

func (n CardNetwork) known() bool {
	return n <= NetworkNaranja
}

// networkNames are the credit-card-type names clients exchange networks by.
var networkNames = [...]string{
	NetworkVisa:            "visa",
	NetworkMastercard:      "mastercard",
	NetworkAmericanExpress: "american-express",
	NetworkDiscover:        "discover",
	NetworkDinersClub:      "diners-club",
	NetworkJCB:             "jcb",
	NetworkUnionPay:        "unionpay",
	NetworkMaestro:         "maestro",
	NetworkMir:             "mir",
	NetworkElo:             "elo",
	NetworkHiper:           "hiper",
	NetworkHipercard:       "hipercard",
	NetworkTroy:            "troy",
	NetworkVerve:           "verve",
	NetworkNaranja:         "naranja",
}

// Name is the network's credit-card-type name, such as "american-express"; empty for none or an unknown network.
func (n CardNetwork) Name() string {
	if !n.known() {
		return ""
	}
	return networkNames[n]
}

// ParseCardNetwork reads a Name; the empty name is no network.
func ParseCardNetwork(name string) (CardNetwork, bool) {
	if name == "" {
		return 0, true
	}
	for network, known := range networkNames {
		if network > 0 && known == name {
			return CardNetwork(network), true
		}
	}
	return 0, false
}

// Card limits count characters, except the ASCII digits of the number, security code and PIN.
const (
	MinCardNumberLength   = 12
	MaxCardNumberLength   = 19
	MaxCardHolderLength   = 256
	MinSecurityCodeLength = 3
	MaxSecurityCodeLength = 4
	MinCardPINLength      = 4
	MaxCardPINLength      = 12
	MaxBankNameLength     = 128
)

// time.Parse accepts year 0000 with expiryLayout, so the year is checked on its own.
const expiryLayout = "2006-01"

// A colour is # and six lowercase hexadecimal digits.
const colorLength = 7

// CardFace is what the list shows of a card without decrypting it; Color is empty when the card has none.
type CardFace struct {
	Network  CardNetwork
	LastFour string
	Color    string
}

// AddressLink names one address of one identity.
type AddressLink struct {
	Identity ID
	Address  ID
}

// CardInput is one payment card; Expiry is YYYY-MM or empty, Color #rrggbb or empty, and at most one of Billing and BillingLink is set.
type CardInput struct {
	Label        string
	Holder       string
	Number       string
	Expiry       string
	SecurityCode string
	PIN          string
	Network      CardNetwork
	BankName     string
	BankSite     string
	Color        string
	Billing      *Address
	BillingLink  *AddressLink
	Notes        string
	Tags         []string
}

// Card is a card as read; Linked is the address BillingLink names, and a link whose identity or address is gone reads as none.
type Card struct {
	ID     ID
	Linked *Address
	CardInput
}

// IdentityAddresses is what a card can link to in one identity.
type IdentityAddresses struct {
	Identity  ID
	Label     string
	Addresses []Address
}

// acceptCard returns a card as the vault stores it, refusing any value out of bounds.
func acceptCard(input CardInput) (CardInput, error) {
	if !fits(input.Label, MaxLabelLength) || strings.TrimSpace(input.Label) == "" {
		return CardInput{}, ErrInvalidInput
	}
	tags, err := AcceptTags(input.Tags)
	if err != nil {
		return CardInput{}, err
	}
	input.Tags = tags
	if !fits(input.Holder, MaxCardHolderLength) || !fits(input.BankName, MaxBankNameLength) || !fits(input.BankSite, MaxOriginLength) || !fits(input.Notes, MaxNotesLength) {
		return CardInput{}, ErrInvalidInput
	}
	if !Digits(input.Number, MinCardNumberLength, MaxCardNumberLength) || !ValidExpiry(input.Expiry) {
		return CardInput{}, ErrInvalidInput
	}
	if input.SecurityCode != "" && !Digits(input.SecurityCode, MinSecurityCodeLength, MaxSecurityCodeLength) {
		return CardInput{}, ErrInvalidInput
	}
	if input.PIN != "" && !Digits(input.PIN, MinCardPINLength, MaxCardPINLength) {
		return CardInput{}, ErrInvalidInput
	}
	if !input.Network.known() || !validColor(input.Color) {
		return CardInput{}, ErrInvalidInput
	}
	if input.Billing != nil && input.BillingLink != nil {
		return CardInput{}, ErrInvalidInput
	}
	if input.Billing != nil {
		if input.Billing.ID != (ID{}) || input.Billing.Label != "" || !acceptAddress(*input.Billing) {
			return CardInput{}, ErrInvalidInput
		}
		billing := *input.Billing
		input.Billing = &billing
	}
	if input.BillingLink != nil {
		link := *input.BillingLink
		input.BillingLink = &link
	}
	return input, nil
}

// Digits reports whether value is between least and most ASCII digits.
func Digits(value string, least, most int) bool {
	if len(value) < least || len(value) > most {
		return false
	}
	for i := range len(value) {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

// ValidExpiry reports whether value is empty or a card expiry month written YYYY-MM.
func ValidExpiry(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := time.Parse(expiryLayout, value)
	return err == nil && parsed.Year() >= earliestDateYear
}

func validColor(value string) bool {
	if value == "" {
		return true
	}
	if len(value) != colorLength || value[0] != '#' {
		return false
	}
	for i := 1; i < len(value); i++ {
		if (value[i] < '0' || value[i] > '9') && (value[i] < 'a' || value[i] > 'f') {
			return false
		}
	}
	return true
}

// expiryEnd is the last day of an accepted expiry month as YYYY-MM-DD, or empty.
func expiryEnd(expiry string) string {
	parsed, err := time.Parse(expiryLayout, expiry)
	if err != nil {
		return ""
	}
	return parsed.AddDate(0, 1, -1).Format(dateLayout)
}

func cardFace(input CardInput) CardFace {
	return CardFace{Network: input.Network, LastFour: LastFour(input.Number), Color: input.Color}
}

// LastFour is the end of a card number, empty for a number shorter than four characters.
func LastFour(number string) string {
	if len(number) < 4 {
		return ""
	}
	return number[len(number)-4:]
}

// BillingAddress is the address the card bills to, its link's or its own; nil for none.
func (c Card) BillingAddress() *Address {
	if c.Linked != nil {
		return c.Linked
	}
	return c.Billing
}

// validCardFace holds the index rules for a card's face.
func validCardFace(face CardFace) bool {
	return face.Network.known() && Digits(face.LastFour, 4, 4) && validColor(face.Color)
}

// cardEntry is what the index shows of an accepted card.
func cardEntry(input CardInput, groups []ID) entryMeta {
	return entryMeta{kind: KindCard, label: input.Label, detail: input.BankName, expiresOn: expiryEnd(input.Expiry), site: SiteOf(input.BankSite), card: cardFace(input), groups: groups, tags: input.Tags}
}

// A card record holds everything but the label, which lives in the index; at most one of billing
// and link is non-empty:
//
//	[8, holder, number, expiry, securityCode, pin, network, bankName, bankSite, color, [billing…],
//	 [link…], notes]
//	billing   none, or one address without its id and with an empty label
//	link      none, or identityID, addressID
const recordSchemaCard = 8

type cardRecord struct {
	_            struct{} `cbor:",toarray"`
	Schema       uint64
	Holder       string
	Number       string
	Expiry       string
	SecurityCode string
	PIN          string
	Network      uint64
	BankName     string
	BankSite     string
	Color        string
	Billing      []wireAddress
	Link         []ID
	Notes        string
}

func encodeCardRecord(input CardInput) ([]byte, error) {
	record := cardRecord{
		Schema: recordSchemaCard, Holder: input.Holder, Number: input.Number, Expiry: input.Expiry, SecurityCode: input.SecurityCode,
		PIN: input.PIN, Network: uint64(input.Network), BankName: input.BankName, BankSite: input.BankSite, Color: input.Color, Notes: input.Notes,
	}
	if input.Billing != nil {
		record.Billing = []wireAddress{wireAddressOf(*input.Billing)}
	}
	if input.BillingLink != nil {
		record.Link = []ID{input.BillingLink.Identity, input.BillingLink.Address}
	}
	return sealableRecord(marshal(record, 0))
}

func decodeCardRecord(plaintext []byte) (CardInput, error) {
	if err := readSchema(plaintext, recordSchemaCard); err != nil {
		return CardInput{}, err
	}
	var record cardRecord
	if err := unmarshal(plaintext, &record); err != nil {
		return CardInput{}, err
	}
	if record.Network > uint64(NetworkNaranja) {
		return CardInput{}, ErrUnsupported
	}
	if len(record.Billing) > 1 || len(record.Link) != 0 && len(record.Link) != 2 || len(record.Billing) != 0 && len(record.Link) != 0 ||
		!validText(record.Holder, record.Number, record.Expiry, record.SecurityCode, record.PIN, record.BankName, record.BankSite, record.Color, record.Notes) {
		return CardInput{}, ErrMalformed
	}
	input := CardInput{
		Holder: record.Holder, Number: record.Number, Expiry: record.Expiry, SecurityCode: record.SecurityCode, PIN: record.PIN,
		Network: CardNetwork(record.Network), BankName: record.BankName, BankSite: record.BankSite, Color: record.Color, Notes: record.Notes,
	}
	if len(record.Billing) == 1 {
		billing, err := record.Billing[0].address(ID{})
		if err != nil {
			return CardInput{}, err
		}
		input.Billing = &billing
	}
	if len(record.Link) == 2 {
		input.BillingLink = &AddressLink{Identity: record.Link[0], Address: record.Link[1]}
	}
	return input, nil
}

// verifyCardEntry checks that a card entry shows what its record holds.
func verifyCardEntry(entry entryMeta, plaintext []byte) error {
	input, err := decodeCardRecord(plaintext)
	if err != nil {
		return err
	}
	input.Label = entry.label
	accepted, err := acceptCard(input)
	if err != nil {
		return ErrMalformed
	}
	want := cardEntry(accepted, entry.groups)
	if entry.detail != want.detail || entry.expiresOn != want.expiresOn || entry.site != want.site || entry.card != want.card {
		return ErrMalformed
	}
	return nil
}

// ReadSelectedCard decrypts the selected card and the address its link names.
func (s *Session) ReadSelectedCard(ticket Selection) (Card, error) {
	var input CardInput
	entry, err := s.readSelected(ticket, KindCard, func(plaintext []byte) (err error) {
		input, err = decodeCardRecord(plaintext)
		return err
	})
	if err != nil {
		return Card{}, err
	}
	input.Label = entry.label
	input.Tags = slices.Clone(entry.tags)
	if input.BillingLink == nil {
		return Card{ID: entry.id, CardInput: input}, nil
	}
	// Resolving the link under the lock keeps the ticket current until the card is returned.
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ticketCurrent(ticket) {
		return Card{}, ErrStaleSelection
	}
	return s.resolvedCard(entry.id, input)
}

// ReadCard decrypts one card and the address its link names without taking the selection.
func (s *Session) ReadCard(id ID) (Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return Card{}, ErrLocked
	}
	index := s.findKind(id, KindCard)
	if index < 0 {
		return Card{}, ErrNotFound
	}
	plaintext, err := s.openRecord(index)
	if err != nil {
		return Card{}, err
	}
	input, err := decodeCardRecord(plaintext)
	clear(plaintext)
	if err != nil {
		return Card{}, err
	}
	input.Label = s.entries[index].label
	input.Tags = slices.Clone(s.entries[index].tags)
	return s.resolvedCard(id, input)
}

// resolvedCard is input as read, with its link resolved or dropped when its address is gone. The caller holds s.mu.
func (s *Session) resolvedCard(id ID, input CardInput) (Card, error) {
	card := Card{ID: id, CardInput: input}
	if input.BillingLink == nil {
		return card, nil
	}
	linked, err := s.linkedAddress(*input.BillingLink)
	if err != nil {
		return Card{}, err
	}
	card.Linked = linked
	if linked == nil {
		card.BillingLink = nil
	}
	return card, nil
}

// linkedAddress returns the address a link names, or nil when its identity is gone or in the trash, or its address is
// gone.
func (s *Session) linkedAddress(link AddressLink) (*Address, error) {
	index := s.findKind(link.Identity, KindIdentity)
	if index < 0 {
		return nil, nil
	}
	identity, err := s.decryptIdentity(index)
	if err != nil {
		return nil, err
	}
	for _, address := range identity.Addresses {
		if address.ID == link.Address {
			return &address, nil
		}
	}
	return nil, nil
}

// acceptCardLink refuses a link to an address the vault does not hold.
func (s *Session) acceptCardLink(input CardInput) error {
	if input.BillingLink == nil {
		return nil
	}
	linked, err := s.linkedAddress(*input.BillingLink)
	if err != nil {
		return err
	}
	if linked == nil {
		return ErrInvalidInput
	}
	return nil
}

// PrepareCreateCard prepares a new card in groups.
func (s *Session) PrepareCreateCard(input CardInput, groups []ID) (*Pending, ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, ID{}, err
	}
	input, err := acceptCard(input)
	if err != nil {
		return nil, ID{}, err
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, ID{}, err
	}
	if err := s.acceptCardLink(input); err != nil {
		return nil, ID{}, err
	}
	plaintext, err := encodeCardRecord(input)
	if err != nil {
		return nil, ID{}, err
	}
	defer clear(plaintext)
	return s.prepareAdd(cardEntry(input, membership), plaintext)
}

// PrepareEditCard replaces a card's content and membership as a whole. Its pin stays.
func (s *Session) PrepareEditCard(id ID, input CardInput, groups []ID) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	index := s.findKind(id, KindCard)
	if index < 0 {
		return nil, ErrNotFound
	}
	input, err := acceptCard(input)
	if err != nil {
		return nil, err
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, err
	}
	if err := s.acceptCardLink(input); err != nil {
		return nil, err
	}
	plaintext, err := encodeCardRecord(input)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	return s.prepareReplace(index, cardEntry(input, membership), plaintext)
}

// unlinkCards reseals into entries and records, aligned with the session's, every card whose link gone selects, without that link.
func (s *Session) unlinkCards(entries []entryMeta, records []sealedBox, gone func(AddressLink) bool) error {
	for i, entry := range s.entries {
		if entry.kind != KindCard {
			continue
		}
		plaintext, err := s.openRecord(i)
		if err != nil {
			return err
		}
		input, err := decodeCardRecord(plaintext)
		clear(plaintext)
		if err != nil {
			return err
		}
		if input.BillingLink == nil || !gone(*input.BillingLink) {
			continue
		}
		input.BillingLink = nil
		rewritten, err := encodeCardRecord(input)
		if err != nil {
			return err
		}
		entries[i], records[i], err = s.sealNext(i, entry, rewritten)
		clear(rewritten)
		if err != nil {
			return err
		}
	}
	return nil
}

// IdentityAddresses lists what a card can link to in each identity outside the trash with an address, without marking
// use or taking a selection.
func (s *Session) IdentityAddresses() ([]IdentityAddresses, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	var result []IdentityAddresses
	for i, entry := range s.entries {
		if entry.kind != KindIdentity || entry.trashed() {
			continue
		}
		identity, err := s.decryptIdentity(i)
		if err != nil {
			return nil, err
		}
		if len(identity.Addresses) > 0 {
			result = append(result, IdentityAddresses{Identity: entry.id, Label: entry.label, Addresses: identity.Addresses})
		}
	}
	return result, nil
}
