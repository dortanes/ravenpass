package importers

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dortanes/ravenpass/packages/vault"
)

func TestWebsitesPlacesAddressesOnACredential(t *testing.T) {
	long := "https://login.example.test/sign-in?return=" + strings.Repeat("a", vault.MaxOriginLength)
	many := make([]string, vault.MaxCredentialWebsites+2)
	domains := make([]string, len(many))
	for i := range many {
		domains[i] = fmt.Sprintf("site%d.example", i)
		many[i] = fmt.Sprintf("https://site%d.example/login", i)
	}
	tests := []struct {
		name      string
		addresses []string
		websites  []string
		notes     string
	}{
		{"trimmed in their order", []string{" https://a.example ", "", "  ", "androidapp://com.example"}, []string{"a.example", "androidapp://com.example"}, ""},
		{"cut to the domain", []string{"https://app.example.org/", "https://www.example.test/sign-in?return=home#top"}, []string{"app.example.org", "example.test"}, ""},
		{"http keeps its scheme and port", []string{"http://localhost:8080/", "http://router.example/admin"}, []string{"http://localhost:8080", "http://router.example"}, ""},
		{"one domain once", []string{"https://a.example/one", "a.example/two", "https://www.a.example"}, []string{"a.example"}, ""},
		{"none", nil, nil, ""},
		{"blank", []string{" "}, nil, ""},
		{"over-long keeps its domain", []string{long}, []string{"login.example.test"}, ""},
		{"over-long without a site root", []string{strings.Repeat("a", vault.MaxOriginLength+1)}, nil, "Website: " + strings.Repeat("a", vault.MaxOriginLength+1)},
		{
			"beyond what a credential holds", many, domains[:vault.MaxCredentialWebsites],
			fmt.Sprintf("Website: https://site%d.example/login\nWebsite: https://site%d.example/login", vault.MaxCredentialWebsites, vault.MaxCredentialWebsites+1),
		},
		{
			"over-long beyond what a credential holds", append(many[:vault.MaxCredentialWebsites:vault.MaxCredentialWebsites], long), domains[:vault.MaxCredentialWebsites],
			"Website: " + long,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var extras Extras
			websites := Websites(test.addresses, "Website", &extras)
			if !reflect.DeepEqual(websites, test.websites) || extras.Notes("") != test.notes {
				t.Fatalf("Websites = %q with notes %q", websites, extras.Notes(""))
			}
		})
	}
}

func TestLoginLabelFallsBackToTheSiteThenTheFallbacks(t *testing.T) {
	tests := []struct {
		name      string
		label     string
		websites  []string
		fallbacks []string
		want      string
	}{
		{"the name", " Example ", []string{"https://www.site.example"}, []string{"alex"}, "Example"},
		{"the site", " ", []string{"androidapp://com.example", "https://www.site.example/login"}, []string{"alex"}, "site.example"},
		{"the first fallback", "", []string{"androidapp://com.example"}, []string{" ", " alex ", "alex@example.test"}, "alex"},
		{"nothing", "", nil, []string{"", " "}, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := LoginLabel(test.label, test.websites, test.fallbacks...); got != test.want {
				t.Fatalf("LoginLabel = %q, want %q", got, test.want)
			}
		})
	}
}
