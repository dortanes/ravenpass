package linkproto

import (
	"net/url"
	"strings"
)

// CardFields are the fields of card requests: the card a fill names, and the values a capture carries. Expiry is
// YYYY-MM and Network a credit-card-type name.
type CardFields struct {
	Card         string `json:"card"`
	Holder       string `json:"holder"`
	Number       string `json:"number"`
	Expiry       string `json:"expiry"`
	SecurityCode string `json:"securityCode"`
	Network      string `json:"network"`
}

// Cards is the result of a cards request.
type Cards struct {
	Cards []CardOption `json:"cards"`
}

// CardOption is a card as the index shows it. Site is the bank's host and ExpiresOn the last day of the expiry month as
// YYYY-MM-DD; empty strings mean none.
type CardOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	BankName  string `json:"bankName"`
	Site      string `json:"site"`
	Network   string `json:"network"`
	LastFour  string `json:"lastFour"`
	Color     string `json:"color"`
	ExpiresOn string `json:"expiresOn"`
}

// CardFill is the result of a card fill request; Billing is null for a card without a billing address.
type CardFill struct {
	Holder       string          `json:"holder"`
	Number       string          `json:"number"`
	Expiry       string          `json:"expiry"`
	SecurityCode string          `json:"securityCode"`
	Billing      *BillingAddress `json:"billing"`
}

// BillingAddress is the address a card bills to.
type BillingAddress struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	Region     string `json:"region"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// CaptureKind tells a captured password from a captured card.
type CaptureKind string

// Kinds of capture.
const (
	CapturePassword CaptureKind = "password"
	CaptureCard     CaptureKind = "card"
)

// CardFace is what a card capture offer shows of the typed card.
type CardFace struct {
	Network  string `json:"network"`
	LastFour string `json:"lastFour"`
}

// SecureOrigin reports an https origin, or an http one on localhost or a .localhost host, which a network attacker
// cannot serve.
func SecureOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return parsed.Scheme == "https" || parsed.Scheme == "http" && (host == "localhost" || strings.HasSuffix(host, ".localhost"))
}
