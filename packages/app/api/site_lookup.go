package api

import (
	"context"

	"github.com/dortanes/ravenpass/packages/vault"
)

// SiteLookup is what a credential's typed or pasted website suggests.
type SiteLookup struct {
	// Name is the site's declared name or its readable domain.
	Name string `json:"name"`
	// Website is the address cut to what the vault matches it by, as imports keep it.
	Website string `json:"website"`
}

// LookupSite names a credential's website: the name the site declares where website icons load and it answers, else
// its readable domain; an address naming no web site is empty, never an error.
func (s *Service) LookupSite(website string) (SiteLookup, error) {
	host := vault.SiteOf(website)
	if host == "" {
		return SiteLookup{}, nil
	}
	lookup := SiteLookup{Name: vault.SiteName(host), Website: vault.WebsiteOf(website)}
	if !s.preferences.SiteIcons() {
		return lookup, nil
	}
	if brand, err := s.brands.Brand(context.Background(), host); err == nil && brand.Name != "" {
		lookup.Name = brand.Name
	}
	return lookup, nil
}
