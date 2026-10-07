package api

import (
	"context"

	"github.com/dortanes/ravenpass/packages/vault"
)

// SiteLookup is what a credential's typed or pasted website suggests.
type SiteLookup struct {
	// Name is the site's declared name or its readable domain, empty when no name was asked for.
	Name string `json:"name"`
	// Website is the address cut to what the vault matches it by, as imports keep it.
	Website string `json:"website"`
}

// LookupSite cuts a credential's website to what the vault matches it by and, when named is set, names it: the name the
// site declares where website icons load and it answers, else its readable domain. Without named no site is contacted.
// An address naming no web site is empty, never an error.
func (s *Service) LookupSite(website string, named bool) (SiteLookup, error) {
	host := vault.SiteOf(website)
	if host == "" {
		return SiteLookup{}, nil
	}
	lookup := SiteLookup{Website: vault.WebsiteOf(website)}
	if !named {
		return lookup, nil
	}
	lookup.Name = vault.SiteName(host)
	if !s.preferences.SiteIcons() {
		return lookup, nil
	}
	if brand, err := s.brands.Brand(context.Background(), host); err == nil && brand.Name != "" {
		lookup.Name = brand.Name
	}
	return lookup, nil
}
