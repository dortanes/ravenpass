package api

import (
	"strings"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

// AddressLink names one address of one identity by their hex ids.
type AddressLink struct {
	IdentityID string `json:"identityId"`
	AddressID  string `json:"addressId"`
}

// CardInput is the editable content of a card; at most one of Billing and BillingLink is set.
type CardInput struct {
	Label        string `json:"label"`
	Holder       string `json:"holder"`
	Number       string `json:"number"`
	Expiry       string `json:"expiry"`
	SecurityCode string `json:"securityCode"`
	PIN          string `json:"pin"`
	// Network is a credit-card-type name such as "visa" or "american-express", empty for none.
	Network  string `json:"network"`
	BankName string `json:"bankName"`
	BankSite string `json:"bankSite"`
	Color    string `json:"color"`
	// Billing is the card's own address.
	Billing *Address `json:"billing"`
	// BillingLink names an identity's address.
	BillingLink *AddressLink `json:"billingLink"`
	Notes       string       `json:"notes"`
	// Tags tell the item apart from others like it.
	Tags []string `json:"tags"`
}

// CardSummary is a card as the list shows it; empty strings mean none.
type CardSummary struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	BankName string `json:"bankName"`
	LastFour string `json:"lastFour"`
	Network  string `json:"network"`
	Color    string `json:"color"`
	// Site is the host of the bank's site.
	Site       string   `json:"site"`
	Pinned     bool     `json:"pinned"`
	LastUsedAt int64    `json:"lastUsedAt"`
	Groups     []string `json:"groups"`
	Tags       []string `json:"tags"`
	// ExpiresOn is the last day of the expiry month as YYYY-MM-DD.
	ExpiresOn string `json:"expiresOn"`
}

// LinkedAddress is the identity address a card's billing link names.
type LinkedAddress struct {
	IdentityLabel string  `json:"identityLabel"`
	Address       Address `json:"address"`
}

// Card is a card as read.
type Card struct {
	ID     string   `json:"id"`
	Groups []string `json:"groups"`
	Site   string   `json:"site"`
	// Linked is the address the billing link names, null without a link or once that address is gone.
	Linked *LinkedAddress `json:"linked"`
	CardInput
}

// IdentityAddresses is what a card's billing address can link to in one identity.
type IdentityAddresses struct {
	IdentityID string    `json:"identityId"`
	Label      string    `json:"label"`
	Addresses  []Address `json:"addresses"`
}

var cardNetworkNames = map[vault.CardNetwork]string{
	vault.NetworkVisa:            "visa",
	vault.NetworkMastercard:      "mastercard",
	vault.NetworkAmericanExpress: "american-express",
	vault.NetworkDiscover:        "discover",
	vault.NetworkDinersClub:      "diners-club",
	vault.NetworkJCB:             "jcb",
	vault.NetworkUnionPay:        "unionpay",
	vault.NetworkMaestro:         "maestro",
	vault.NetworkMir:             "mir",
	vault.NetworkElo:             "elo",
	vault.NetworkHiper:           "hiper",
	vault.NetworkHipercard:       "hipercard",
	vault.NetworkTroy:            "troy",
	vault.NetworkVerve:           "verve",
	vault.NetworkNaranja:         "naranja",
}

func cardNetworkNamed(name string) (vault.CardNetwork, bool) {
	if name == "" {
		return 0, true
	}
	for network, known := range cardNetworkNames {
		if known == name {
			return network, true
		}
	}
	return 0, false
}

// ListCards reports every card from the index.
func (s *Service) ListCards() ([]CardSummary, error) {
	entries, usage, err := s.listKind(vault.KindCard)
	if err != nil {
		return nil, err
	}
	result := make([]CardSummary, len(entries))
	for i, entry := range entries {
		result[i] = CardSummary{
			ID:         entry.ID.String(),
			Label:      entry.Label,
			BankName:   entry.Detail,
			LastFour:   entry.Card.LastFour,
			Network:    cardNetworkNames[entry.Card.Network],
			Color:      entry.Card.Color,
			Site:       entry.Site,
			Pinned:     entry.Pinned,
			LastUsedAt: usage[entry.ID],
			Groups:     idStrings(entry.Groups),
			Tags:       append([]string{}, entry.Tags...),
			ExpiresOn:  entry.ExpiresOn,
		}
	}
	return result, nil
}

// ReadCard opens the card id with its linked address and records the use.
func (s *Service) ReadCard(id string) (Card, error) {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return Card{}, fail(failureItemUnreadable)
	}
	item, groups, err := openItem(s, parsed, s.vault.ReadSelectedCard)
	if err != nil {
		return Card{}, err
	}
	card := Card{ID: item.ID.String(), Groups: groups, Site: vault.SiteOf(item.BankSite), CardInput: fromVaultCard(item.CardInput)}
	if item.Linked != nil {
		label, err := s.itemLabel(item.BillingLink.Identity)
		if err != nil {
			return Card{}, err
		}
		card.Linked = &LinkedAddress{IdentityLabel: label, Address: fromVaultAddress(*item.Linked)}
	}
	return card, nil
}

// itemLabel is the label the index holds for an item.
func (s *Service) itemLabel(id vault.ID) (string, error) {
	entries, err := s.vault.List()
	if err != nil {
		return "", present(err)
	}
	for _, entry := range entries {
		if entry.ID == id {
			return entry.Label, nil
		}
	}
	return "", nil
}

// CreateCard creates a card in groups and returns its id.
func (s *Service) CreateCard(input CardInput, groups []string) (string, error) {
	membership, err := s.knownGroups(groups)
	if err != nil {
		return "", err
	}
	card, err := input.toVault()
	if err != nil {
		return "", err
	}
	id, err := s.vault.CreateCard(card, membership)
	if err != nil {
		return "", present(err)
	}
	return id.String(), nil
}

// UpdateCard replaces a card's content and membership as a whole.
func (s *Service) UpdateCard(id string, input CardInput, groups []string) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	membership, err := s.knownGroups(groups)
	if err != nil {
		return err
	}
	card, err := input.toVault()
	if err != nil {
		return err
	}
	return present(s.vault.EditCard(parsed, card, membership))
}

// ListIdentityAddresses lists the identity addresses a card's billing address can link to.
func (s *Service) ListIdentityAddresses() ([]IdentityAddresses, error) {
	listed, err := s.vault.IdentityAddresses()
	if err != nil {
		return nil, present(err)
	}
	result := make([]IdentityAddresses, len(listed))
	for i, identity := range listed {
		result[i] = IdentityAddresses{IdentityID: identity.Identity.String(), Label: identity.Label, Addresses: make([]Address, len(identity.Addresses))}
		for j, address := range identity.Addresses {
			result[i].Addresses[j] = fromVaultAddress(address)
		}
	}
	return result, nil
}

// CopyCardField copies "number", "holder", "expiry", "securityCode", "pin", "bankName", "bankSite", "notes", "billing" or "billing:<part>" of a card.
func (s *Service) CopyCardField(id string, field string) error {
	return s.copyCardField(id, field, clearAfter)
}

func (s *Service) copyCardField(id string, field string, schedule func(time.Duration, func())) error {
	read, supported := cardField(field)
	if !supported {
		return fail(failureFieldNotCopyable)
	}
	return s.copyItemValue(id, schedule, func(parsed vault.ID) (string, error) {
		item, err := readSelected(s, parsed, s.vault.ReadSelectedCard)
		if err != nil {
			return "", err
		}
		return read(item), nil
	})
}

var cardFields = map[string]func(vault.Card) string{
	"number":       func(card vault.Card) string { return card.Number },
	"holder":       func(card vault.Card) string { return card.Holder },
	"expiry":       func(card vault.Card) string { return expiryDisplay(card.Expiry) },
	"securityCode": func(card vault.Card) string { return card.SecurityCode },
	"pin":          func(card vault.Card) string { return card.PIN },
	"bankName":     func(card vault.Card) string { return card.BankName },
	"bankSite":     func(card vault.Card) string { return card.BankSite },
	"notes":        func(card vault.Card) string { return card.Notes },
	"billing": func(card vault.Card) string {
		address, found := billingAddress(card)
		if !found {
			return ""
		}
		return formatAddress(address)
	},
}

// cardField resolves a cardFields name or "billing:<part>" to its reader; a missing value reads as empty.
func cardField(reference string) (func(vault.Card) string, bool) {
	if read, known := cardFields[reference]; known {
		return read, true
	}
	name, found := strings.CutPrefix(reference, "billing:")
	if !found {
		return nil, false
	}
	part, known := addressPart(name)
	if !known {
		return nil, false
	}
	return func(card vault.Card) string {
		address, found := billingAddress(card)
		if !found {
			return ""
		}
		return part(address)
	}, true
}

// billingAddress is a card's own billing address or the one its link names.
func billingAddress(card vault.Card) (vault.Address, bool) {
	switch {
	case card.Billing != nil:
		return *card.Billing, true
	case card.Linked != nil:
		return *card.Linked, true
	default:
		return vault.Address{}, false
	}
}

// expiryDisplay writes a stored YYYY-MM expiry as MM/YY.
func expiryDisplay(expiry string) string {
	parsed, err := time.Parse("2006-01", expiry)
	if err != nil {
		return ""
	}
	return parsed.Format("01/06")
}

func (input CardInput) toVault() (vault.CardInput, error) {
	network, known := cardNetworkNamed(input.Network)
	if !known {
		return vault.CardInput{}, fail(failureInvalidItem)
	}
	card := vault.CardInput{
		Label: input.Label, Holder: input.Holder, Number: input.Number, Expiry: input.Expiry,
		SecurityCode: input.SecurityCode, PIN: input.PIN, Network: network,
		BankName: input.BankName, BankSite: input.BankSite, Color: input.Color, Notes: input.Notes, Tags: input.Tags,
	}
	if input.Billing != nil {
		billing, err := input.Billing.toVault()
		if err != nil {
			return vault.CardInput{}, err
		}
		card.Billing = &billing
	}
	if input.BillingLink != nil {
		identity, identityErr := vault.ParseID(input.BillingLink.IdentityID)
		address, addressErr := vault.ParseID(input.BillingLink.AddressID)
		if identityErr != nil || addressErr != nil {
			return vault.CardInput{}, fail(failureInvalidItem)
		}
		card.BillingLink = &vault.AddressLink{Identity: identity, Address: address}
	}
	return card, nil
}

func fromVaultCard(card vault.CardInput) CardInput {
	input := CardInput{
		Label: card.Label, Holder: card.Holder, Number: card.Number, Expiry: card.Expiry,
		SecurityCode: card.SecurityCode, PIN: card.PIN, Network: cardNetworkNames[card.Network],
		BankName: card.BankName, BankSite: card.BankSite, Color: card.Color, Notes: card.Notes,
		Tags: append([]string{}, card.Tags...),
	}
	if card.Billing != nil {
		billing := fromVaultAddress(*card.Billing)
		input.Billing = &billing
	}
	if card.BillingLink != nil {
		input.BillingLink = &AddressLink{IdentityID: card.BillingLink.Identity.String(), AddressID: card.BillingLink.Address.String()}
	}
	return input
}
