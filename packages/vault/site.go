package vault

import (
	"net"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

// SiteOf returns a web origin's HostASCII host without one leading "www.", scheme defaulting to HTTPS, or empty for no web site.
func SiteOf(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	implied := !strings.Contains(origin, "://")
	if implied {
		origin = "https://" + origin
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	// Without a scheme, text before "@" is an email address, not user info of a web address.
	if implied && parsed.User != nil {
		return ""
	}
	return siteOfURL(parsed)
}

// WebsiteOf returns a website cut to what the vault matches it by: the readable SiteOf of an https or scheme-less
// address, and the scheme, host and any port but 80 of an http one, which a platform autofill over plain http checks.
// Paths, queries, fragments and user info go; an address SiteOf cannot read comes back trimmed and otherwise unchanged.
func WebsiteOf(address string) string {
	address = strings.TrimSpace(address)
	implied := !strings.Contains(address, "://")
	target := address
	if implied {
		target = "https://" + address
	}
	parsed, err := url.Parse(target)
	if err != nil || implied && parsed.User != nil {
		return address
	}
	site := siteOfURL(parsed)
	if site == "" {
		return address
	}
	if parsed.Scheme == "https" {
		return SiteName(site)
	}
	host, _ := HostASCII(parsed.Hostname())
	if port := parsed.Port(); port != "" && port != "80" {
		host = net.JoinHostPort(host, port)
	}
	return "http://" + host
}

// OriginOf returns an http or https address's origin as scheme and host, or false for any other address or one with user info.
func OriginOf(address string) (string, bool) {
	parsed, err := url.Parse(address)
	if err != nil {
		return "", false
	}
	return originOfURL(parsed)
}

// ParseOrigin returns a bare http or https origin as scheme and host; one trailing "/", which Chrome on Android adds, is dropped, and any other path, query or fragment fails.
func ParseOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", false
	}
	return originOfURL(parsed)
}

func originOfURL(parsed *url.URL) (string, bool) {
	if parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", false
	}
	return parsed.Scheme + "://" + parsed.Host, true
}

// siteOfURL is SiteOf of a parsed address.
func siteOfURL(parsed *url.URL) string {
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	ascii, ok := HostASCII(parsed.Hostname())
	if !ok {
		return ""
	}
	ascii = strings.TrimPrefix(ascii, "www.")
	if ascii == "" || len(ascii) > MaxOriginLength {
		return ""
	}
	return ascii
}

// HostASCII returns a host name in lower-case ASCII without one trailing dot, or false when IDNA fails or nothing remains.
func HostASCII(host string) (string, bool) {
	ascii, err := idna.Lookup.ToASCII(strings.TrimSuffix(host, "."))
	if err != nil || ascii == "" {
		return "", false
	}
	return strings.ToLower(ascii), true
}

// SiteName is a site in readable Unicode, or unchanged when IDNA fails, which can leave a partial conversion.
func SiteName(site string) string {
	name, err := idna.Lookup.ToUnicode(site)
	if err != nil {
		return site
	}
	return name
}

// Site is the first of the credential's Sites, or empty.
func (input CredentialInput) Site() string {
	if sites := input.Sites(); len(sites) > 0 {
		return sites[0]
	}
	return ""
}

// Sites are SiteOf the credential's websites in order, without empties or repeats; nil for none.
func (input CredentialInput) Sites() []string {
	var sites []string
	for _, website := range input.Websites {
		if site := SiteOf(website); site != "" && !slices.Contains(sites, site) {
			sites = append(sites, site)
		}
	}
	return sites
}

// Match is how closely a credential's sites match a web origin; a greater Match is closer.
type Match uint8

const (
	MatchNone Match = iota
	MatchDomain
	MatchExact
)

// MatchSite reports how closely sites match an http or https origin; IP addresses, localhost and single labels have no registrable domain.
func MatchSite(sites []string, origin string) Match {
	parsed, err := url.Parse(origin)
	if err != nil {
		return MatchNone
	}
	site := siteOfURL(parsed)
	if site == "" {
		return MatchNone
	}
	domain := registrableDomain(site)
	match := MatchNone
	for _, candidate := range sites {
		if candidate == site {
			return MatchExact
		}
		if domain != "" && registrableDomain(candidate) == domain {
			match = MatchDomain
		}
	}
	return match
}

// registrableDomain is a site's public suffix with the label before it, or empty.
func registrableDomain(site string) string {
	if net.ParseIP(site) != nil {
		return ""
	}
	domain, err := publicsuffix.EffectiveTLDPlusOne(site)
	if err != nil {
		return ""
	}
	return domain
}
