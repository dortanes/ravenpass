package importers

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/dortanes/ravenpass/packages/vault"
)

// Websites returns the addresses as a credential's websites, each cut by vault.WebsiteOf to what the vault matches it
// by and kept once; one still over-long, or past what a credential holds, goes to extras as given.
func Websites(addresses []string, label string, extras *Extras) []string {
	var websites []string
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		website := vault.WebsiteOf(address)
		switch {
		case website == "" || slices.Contains(websites, website):
		case utf8.RuneCountInString(website) > vault.MaxOriginLength || len(websites) >= vault.MaxCredentialWebsites:
			extras.Add(label, address)
		default:
			websites = append(websites, website)
		}
	}
	return websites
}

// LoginLabel is the label of a login: its name, else the site of its first website that names one, else the first non-blank fallback.
func LoginLabel(name string, websites []string, fallbacks ...string) string {
	if label := strings.TrimSpace(name); label != "" {
		return label
	}
	if site := (vault.CredentialInput{Websites: websites}).Site(); site != "" {
		return site
	}
	for _, fallback := range fallbacks {
		if label := strings.TrimSpace(fallback); label != "" {
			return label
		}
	}
	return ""
}
