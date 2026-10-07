package api

import (
	"slices"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/siteicons"
)

func TestLookupSiteNamesTheSiteOrItsDomain(t *testing.T) {
	service := newReadyService(t)
	brands := &scriptedBrands{brand: siteicons.Brand{Name: "Example Mail"}}
	service.brands = brands
	if lookup, err := service.LookupSite("https://www.mail.example.com/login", true); err != nil || lookup != (SiteLookup{Name: "Example Mail", Website: "mail.example.com"}) {
		t.Fatalf("a declared name = %+v, %v", lookup, err)
	}
	if lookup, err := service.LookupSite("https://www.forum.example.com/login", false); err != nil || lookup != (SiteLookup{Website: "forum.example.com"}) {
		t.Fatalf("a lookup for a named credential = %+v, %v", lookup, err)
	}
	service.brands = &scriptedBrands{err: siteicons.ErrUnanswered}
	if lookup, err := service.LookupSite("xn--bcher-kva.example", true); err != nil || lookup != (SiteLookup{Name: "bücher.example", Website: "bücher.example"}) {
		t.Fatalf("a silent site = %+v, %v", lookup, err)
	}
	if err := service.SetSiteIcons(false); err != nil {
		t.Fatal(err)
	}
	quiet := &scriptedBrands{brand: siteicons.Brand{Name: "Harbour"}}
	service.brands = quiet
	if lookup, err := service.LookupSite("http://localhost:8080/", true); err != nil || lookup != (SiteLookup{Name: "localhost", Website: "http://localhost:8080"}) {
		t.Fatalf("a lookup with icons off = %+v, %v", lookup, err)
	}
	if lookup, err := service.LookupSite("mailto:alex@example.com", true); err != nil || lookup != (SiteLookup{}) {
		t.Fatalf("an address naming no site = %+v, %v", lookup, err)
	}
	if asked := append(brands.sites(), quiet.sites()...); !slices.Equal(asked, []string{"mail.example.com"}) {
		t.Fatalf("sites contacted = %q", asked)
	}
}
