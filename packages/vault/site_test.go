package vault

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// sitesElement is an entry in the current form with the given site and sites.
func sitesElement(id ID, digest [32]byte, kind uint64, site string, sites ...string) []byte {
	fields := currentFields(id, digest, kind, encodeArray())
	fields[fieldSite] = encodeBytes([]byte(site))
	fields[fieldSites] = encodeArray(encodeTexts(sites)...)
	return encodeArray(fields...)
}

func numberedSites(count int) []string {
	return CredentialInput{Websites: numberedWebsites(count)}.Sites()
}

func TestSiteOf(t *testing.T) {
	tests := []struct {
		origin string
		want   string
	}{
		{"https://www.Google.com/mail", "google.com"},
		{"google.com", "google.com"},
		{"http://google.com:8080", "google.com"},
		{"  https://google.com  ", "google.com"},
		{"google.com:443/login?next=1", "google.com"},
		{"https://google.com./", "google.com"},
		{"https://WWW.GOOGLE.COM", "google.com"},
		{"mail.google.com", "mail.google.com"},
		{"https://www.www.example.com", "www.example.com"},
		{"https://user:secret@example.com/", "example.com"},
		{"https://Bücher.example/", "xn--bcher-kva.example"},
		{"https://xn--bcher-kva.example/", "xn--bcher-kva.example"},
		{"192.0.2.1", "192.0.2.1"},
		{"", ""},
		{"   ", ""},
		{"mailto:alex@example.com", ""},
		{"alex@example.com", ""},
		{"ftp://example.com", ""},
		{"javascript:alert(1)", ""},
		{"file:///etc/passwd", ""},
		{"https://", ""},
		{"https:///path", ""},
		{"https://exa mple.com", ""},
		{"https://under_score.example", ""},
		{"https://[::1]:8080/", ""},
		{"https://" + strings.Repeat("a.", 129), ""},
		{strings.Repeat("ü", MaxOriginLength), ""},
	}
	for _, test := range tests {
		t.Run(test.origin, func(t *testing.T) {
			if got := SiteOf(test.origin); got != test.want {
				t.Fatalf("SiteOf(%q) = %q, want %q", test.origin, got, test.want)
			}
		})
	}
}

func TestOriginOfAnAddressIsItsSchemeAndHost(t *testing.T) {
	for address, want := range map[string]string{
		"https://Mail.Example.com:8443/inbox?q=1#top": "https://Mail.Example.com:8443",
		"HTTP://example.com/":                         "http://example.com",
		"https://example.com":                         "https://example.com",
		"https://alex@example.com/":                   "",
		"ftp://example.com/":                          "",
		"example.com":                                 "",
		"https://":                                    "",
		"https:example.com":                           "",
	} {
		got, ok := OriginOf(address)
		if got != want || ok != (want != "") {
			t.Errorf("OriginOf(%q) = %q, %t; want %q", address, got, ok, want)
		}
	}
}

func TestParseOriginDropsOneTrailingSlashAndRefusesAnyOtherPath(t *testing.T) {
	for raw, want := range map[string]string{
		"https://example.com":        "https://example.com",
		"https://example.com/":       "https://example.com",
		"http://login.example:8443/": "http://login.example:8443",
		"https://example.com//":      "",
		"https://example.com/login":  "",
		"https://example.com/?q=1":   "",
		"https://example.com?":       "",
		"https://example.com/#f":     "",
		"https://user@example.com":   "",
		"android:apk-key-hash:abc":   "",
		"example.com":                "",
		"https://":                   "",
	} {
		got, ok := ParseOrigin(raw)
		if got != want || ok != (want != "") {
			t.Errorf("ParseOrigin(%q) = %q, %t; want %q", raw, got, ok, want)
		}
	}
}

func TestSiteName(t *testing.T) {
	tests := []struct {
		site string
		want string
	}{
		{"xn--bcher-kva.example", "bücher.example"},
		{"example.com", "example.com"},
		{"192.0.2.1", "192.0.2.1"},
		{"localhost", "localhost"},
		{"xn--a", "xn--a"},
		{"under_score.example", "under_score.example"},
		{"https://exa mple.com", "https://exa mple.com"},
		{"", ""},
	}
	for _, test := range tests {
		if got := SiteName(test.site); got != test.want {
			t.Fatalf("SiteName(%q) = %q, want %q", test.site, got, test.want)
		}
	}
}

func TestCredentialSiteIsTheFirstWebsiteThatNamesASite(t *testing.T) {
	tests := []struct {
		websites []string
		want     string
	}{
		{nil, ""},
		{[]string{"https://www.Example.com/login", "https://admin.example.org"}, "example.com"},
		{[]string{"androidapp://com.example", "https://example.com"}, "example.com"},
		{[]string{"not a site", "mailto:alex@example.com"}, ""},
	}
	for _, test := range tests {
		if got := (CredentialInput{Websites: test.websites}).Site(); got != test.want {
			t.Errorf("Site of %q = %q, want %q", test.websites, got, test.want)
		}
	}
}

func TestIndexSiteSkipsAWebsiteThatNamesNoSite(t *testing.T) {
	created, ids := vaultWith(t,
		CredentialInput{Label: "Example", Websites: []string{"androidapp://com.example", "https://example.com"}, Password: "one"},
	)
	session := created.Session
	defer session.Lock()
	if entry := listedEntry(t, session, ids[0]); entry.Site != "example.com" {
		t.Fatalf("created site = %q", entry.Site)
	}
	websites := []string{"androidapp://com.example", "https://www.Other.example/", "https://example.com"}
	pending, err := session.PrepareEdit(ids[0], CredentialPatch{Websites: &websites})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if entry := listedEntry(t, session, ids[0]); entry.Site != "other.example" {
		t.Fatalf("edited site = %q", entry.Site)
	}
}

func TestSiteIsDerivedOnCreateAndEditAndEmptyForIdentities(t *testing.T) {
	created, ids := vaultWith(t,
		CredentialInput{Label: "Example", Websites: []string{"https://www.Example.com/login"}, Password: "one"},
		CredentialInput{Label: "Local", Websites: []string{"not a site"}, Password: "two"},
	)
	session := created.Session
	defer session.Lock()
	identity := commitIdentity(t, session, IdentityInput{Label: "Me"}, nil)
	if entry := listedEntry(t, session, ids[0]); entry.Site != "example.com" {
		t.Fatalf("created site = %q", entry.Site)
	}
	if entry := listedEntry(t, session, ids[1]); entry.Site != "" {
		t.Fatalf("site of an origin that names no site = %q", entry.Site)
	}
	if entry := listedEntry(t, session, identity); entry.Site != "" {
		t.Fatalf("identity site = %q", entry.Site)
	}

	websites := []string{"https://mail.example.org:8443"}
	pending, err := session.PrepareEdit(ids[0], CredentialPatch{Websites: &websites})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	password := "rotated"
	pending, err = session.PrepareEdit(ids[1], CredentialPatch{Password: &password})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if entry := listedEntry(t, session, ids[0]); entry.Site != "mail.example.org" {
		t.Fatalf("edited site = %q", entry.Site)
	}

	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if entry := listedEntry(t, reopened, ids[0]); entry.Site != "mail.example.org" {
		t.Fatalf("site across a reopen = %q", entry.Site)
	}
}

func TestCredentialSitesAreEveryWebsitesSiteOnce(t *testing.T) {
	tests := []struct {
		websites []string
		want     []string
	}{
		{nil, nil},
		{[]string{"not a site", "mailto:alex@example.com"}, nil},
		{[]string{"https://www.Example.com/login", "androidapp://com.example", "example.com:8443", "https://admin.example.org"}, []string{"example.com", "admin.example.org"}},
		{[]string{"androidapp://com.example", "https://example.org", "http://EXAMPLE.org./"}, []string{"example.org"}},
	}
	for _, test := range tests {
		input := CredentialInput{Websites: test.websites}
		if got := input.Sites(); !slices.Equal(got, test.want) || (got == nil) != (test.want == nil) {
			t.Errorf("Sites of %q = %q, want %q", test.websites, got, test.want)
		}
		if first := input.Site(); len(test.want) > 0 && first != test.want[0] || len(test.want) == 0 && first != "" {
			t.Errorf("Site of %q = %q, want the first of %q", test.websites, first, test.want)
		}
	}
}

func TestMatchSite(t *testing.T) {
	tests := []struct {
		name   string
		sites  []string
		origin string
		want   Match
	}{
		{"exact", []string{"example.com"}, "https://example.com", MatchExact},
		{"exact over HTTP on another port", []string{"example.com"}, "http://example.com:8080", MatchExact},
		{"www", []string{"example.com"}, "https://www.example.com", MatchExact},
		{"case and a trailing dot", []string{"example.com"}, "https://EXAMPLE.com.", MatchExact},
		{"subdomain", []string{"example.com"}, "https://login.example.com", MatchDomain},
		{"parent of the site", []string{"login.example.com"}, "https://example.com", MatchDomain},
		{"sibling subdomain", []string{"mail.example.com"}, "https://login.example.com", MatchDomain},
		{"other domain", []string{"example.com"}, "https://example.org", MatchNone},
		{"name that ends like the site", []string{"example.com"}, "https://notexample.com", MatchNone},
		{"site inside another name", []string{"example.com"}, "https://example.com.attacker.test", MatchNone},
		{"subdomain under a two-label suffix", []string{"shop.example.co.uk"}, "https://example.co.uk", MatchDomain},
		{"neighbours under a two-label suffix", []string{"example.co.uk"}, "https://other.co.uk", MatchNone},
		{"shared public suffix", []string{"alex.github.io"}, "https://sam.github.io", MatchNone},
		{"exact under a shared public suffix", []string{"alex.github.io"}, "https://alex.github.io", MatchExact},
		{"subdomain under a shared public suffix", []string{"alex.github.io"}, "https://docs.alex.github.io", MatchDomain},
		{"later site", []string{"example.com", "example.org"}, "https://example.org", MatchExact},
		{"exact after a domain match", []string{"example.com", "login.example.com"}, "https://login.example.com", MatchExact},
		{"no sites", nil, "https://example.com", MatchNone},
		{"IP address", []string{"192.0.2.1"}, "https://192.0.2.1:8443", MatchExact},
		{"other IP address", []string{"192.0.2.1"}, "https://192.0.2.2", MatchNone},
		{"IP addresses ending alike", []string{"198.51.100.1"}, "https://10.0.100.1", MatchNone},
		{"localhost", []string{"localhost"}, "http://localhost:3000", MatchExact},
		{"subdomain of localhost", []string{"localhost"}, "http://app.localhost", MatchNone},
		{"localhost for a subdomain of it", []string{"app.localhost"}, "http://localhost", MatchNone},
		{"single label", []string{"intranet"}, "http://intranet", MatchExact},
		{"subdomain of a single label", []string{"intranet"}, "http://wiki.intranet", MatchNone},
		{"single label for a subdomain of it", []string{"wiki.intranet"}, "http://intranet", MatchNone},
		{"FTP", []string{"example.com"}, "ftp://example.com", MatchNone},
		{"extension page", []string{"example.com"}, "chrome-extension://example.com", MatchNone},
		{"file", []string{"example.com"}, "file:///example.com", MatchNone},
		{"no scheme", []string{"example.com"}, "example.com", MatchNone},
		{"empty", []string{"example.com"}, "", MatchNone},
		{"scheme only", []string{"example.com"}, "https://", MatchNone},
		{"space in the host", []string{"example.com"}, "https://exa mple.com", MatchNone},
		{"unclosed IPv6 address", []string{"example.com"}, "https://[::1", MatchNone},
		{"space before the scheme", []string{"example.com"}, " https://example.com", MatchNone},
		{"IDN in ASCII form", []string{"xn--bcher-kva.example"}, "https://xn--bcher-kva.example", MatchExact},
		{"IDN in Unicode form", []string{"xn--bcher-kva.example"}, "https://Bücher.example", MatchExact},
		{"subdomain of an IDN", []string{"xn--bcher-kva.example"}, "https://shop.xn--bcher-kva.example", MatchDomain},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MatchSite(test.sites, test.origin); got != test.want {
				t.Fatalf("MatchSite(%q, %q) = %d, want %d", test.sites, test.origin, got, test.want)
			}
		})
	}
}

func TestIndexCarriesEveryWebsitesSite(t *testing.T) {
	created, ids := vaultWith(t,
		CredentialInput{Label: "Example", Websites: []string{"androidapp://com.example", "https://www.example.com/login", "https://example.org", "example.com:8443"}, Password: "one"},
		CredentialInput{Label: "Bare", Password: "two"},
	)
	session := created.Session
	defer session.Lock()
	identity := commitIdentity(t, session, IdentityInput{Label: "Me"}, nil)
	card := commitCard(t, session, fullCard(), nil)
	if entry := listedEntry(t, session, ids[0]); entry.Site != "example.com" || !slices.Equal(entry.Sites, []string{"example.com", "example.org"}) {
		t.Fatalf("created sites = %q, %q", entry.Site, entry.Sites)
	}
	for _, id := range []ID{ids[1], identity, card} {
		if entry := listedEntry(t, session, id); entry.Sites != nil {
			t.Fatalf("sites of %s = %q", id, entry.Sites)
		}
	}

	websites := []string{"https://example.net", "https://login.example.org"}
	pending, err := session.PrepareEdit(ids[0], CredentialPatch{Websites: &websites})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	want := []string{"example.net", "login.example.org"}
	if entry := listedEntry(t, session, ids[0]); entry.Site != "example.net" || !slices.Equal(entry.Sites, want) {
		t.Fatalf("edited sites = %q, %q", entry.Site, entry.Sites)
	}
	listed := listedEntry(t, session, ids[0])
	listed.Sites[0] = "changed.example"
	if entry := listedEntry(t, session, ids[0]); !slices.Equal(entry.Sites, want) {
		t.Fatal("a listed entry shares its sites with the session")
	}

	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if entry := listedEntry(t, reopened, ids[0]); !slices.Equal(entry.Sites, want) {
		t.Fatalf("sites across a reopen = %q", entry.Sites)
	}
}

func TestReadCredentialLeavesTheSelectionAlone(t *testing.T) {
	created, ids := vaultWith(t,
		CredentialInput{Label: "Open", Password: "open-secret"},
		CredentialInput{Label: "Other", Websites: []string{"https://example.com"}, Login: "alex", Password: "other-secret"},
	)
	session := created.Session
	defer session.Lock()
	identity := commitIdentity(t, session, IdentityInput{Label: "Me"}, nil)
	ticket, err := session.BeginSelection(ids[0])
	if err != nil {
		t.Fatal(err)
	}
	other, err := session.ReadCredential(ids[1])
	if err != nil {
		t.Fatal(err)
	}
	if other.ID != ids[1] || other.Label != "Other" || other.Login != "alex" || other.Password != "other-secret" || !slices.Equal(other.Websites, []string{"https://example.com"}) {
		t.Fatalf("read credential = %+v", other)
	}
	if open, err := session.ReadSelected(ticket); err != nil || open.Password != "open-secret" {
		t.Fatalf("the selected credential after another read = %+v, error = %v", open, err)
	}
	for _, id := range []ID{identity, {0xff}} {
		if _, err := session.ReadCredential(id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("read of %s: got %v, want ErrNotFound", id, err)
		}
	}
	session.Lock()
	if _, err := session.ReadCredential(ids[1]); !errors.Is(err, ErrLocked) {
		t.Fatalf("read while locked: got %v, want ErrLocked", err)
	}
}

func TestWebsiteOf(t *testing.T) {
	tests := []struct {
		address string
		want    string
	}{
		{"https://app.example.org", "app.example.org"},
		{"https://www.Example.com/mail?hl=en#inbox", "example.com"},
		{"  example.com/login  ", "example.com"},
		{"https://user:secret@example.com/", "example.com"},
		{"https://example.com:8443/", "example.com"},
		{"https://xn--bcher-kva.example/", "bücher.example"},
		{"http://localhost:8080/", "http://localhost:8080"},
		{"http://Router.Local/admin", "http://router.local"},
		{"http://www.example.com:80/path", "http://www.example.com"},
		{"androidapp://com.example.app", "androidapp://com.example.app"},
		{"ssh host", "ssh host"},
		{"name@example.com", "name@example.com"},
		{"", ""},
	}
	for _, test := range tests {
		if got := WebsiteOf(test.address); got != test.want {
			t.Errorf("WebsiteOf(%q) = %q, want %q", test.address, got, test.want)
		}
	}
}
