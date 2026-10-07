package vaultservice

import (
	"cmp"
	"net"
	"net/url"
	"slices"
	"strings"

	"github.com/dortanes/ravenpass/packages/vault"
)

// Requester is a web origin or an Android app asking for a credential; the zero Requester matches nothing.
type Requester struct {
	origin string
	// unwarned is a page whose autofill fills it without warning that plain http is insecure.
	unwarned bool
	app      string
	signers  [][32]byte
	sites    []string
}

// OriginRequester is a page at an http or https origin the browser vouched for and that warns before filling one over plain http.
func OriginRequester(origin string) Requester {
	return Requester{origin: origin}
}

// PlatformOriginRequester is a page at an http or https origin a platform autofill vouched for and fills without an insecure-page warning;
// over plain http it matches only a credential that saved an http website with the page's site and port.
func PlatformOriginRequester(origin string) Requester {
	return Requester{origin: origin, unwarned: true}
}

// AppRequester is Android app pkg with SHA-256 signer digests and its Digital Asset Links-verified sites.
func AppRequester(pkg string, signers [][32]byte, sites []string) Requester {
	return Requester{app: pkg, signers: slices.Clone(signers), sites: slices.Clone(sites)}
}

// match is exact with no site for a linked app, else the best site match over r's origins.
func (r Requester) match(entry vault.Entry) (string, vault.Match) {
	if r.app != "" && vault.LinksApp(entry.Apps, r.app, r.signers) {
		return "", vault.MatchExact
	}
	site, best := "", vault.MatchNone
	for _, origin := range r.origins() {
		if candidate, match := matchedSite(entry.Sites, origin); match > best {
			site, best = candidate, match
		}
	}
	return site, best
}

// checksScheme reports whether r matches only credentials that pass admits, which needs their websites.
func (r Requester) checksScheme() bool {
	return r.unwarned && overHTTP(r.origin)
}

// admits reports whether credential, which r matches by site, may be offered to r.
func (r Requester) admits(credential vault.CredentialInput) bool {
	return !r.checksScheme() || savesHTTPOrigin(credential, r.origin)
}

// overHTTP reports an origin over plain http.
func overHTTP(origin string) bool {
	parsed, ok := vault.OriginOf(origin)
	return ok && strings.HasPrefix(parsed, "http:")
}

// savesHTTPOrigin reports whether one of credential's websites is an http address with the site and port of http origin;
// a website without a scheme is https.
func savesHTTPOrigin(credential vault.CredentialInput, origin string) bool {
	page, ok := httpEndpoint(origin)
	return ok && slices.ContainsFunc(credential.Websites, func(website string) bool {
		saved, ok := httpEndpoint(strings.TrimSpace(website))
		return ok && saved == page
	})
}

// httpEndpoint is an http address's vault.SiteOf and port, port 80 when the address names none.
func httpEndpoint(address string) (string, bool) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil {
		return "", false
	}
	site := vault.SiteOf(address)
	if site == "" {
		return "", false
	}
	return net.JoinHostPort(site, cmp.Or(parsed.Port(), "80")), true
}

// origins is r's origin, or an https origin for each verified site of an app.
func (r Requester) origins() []string {
	if r.app == "" {
		return []string{r.origin}
	}
	origins := make([]string, len(r.sites))
	for i, site := range r.sites {
		origins[i] = "https://" + site
	}
	return origins
}

// Name is the default label of a new credential saved for r.
func (r Requester) Name() string {
	switch {
	case r.app == "":
		return siteLabel(PageSite(r.origin))
	case len(r.sites) > 0:
		return siteLabel(r.sites[0])
	default:
		return cutLabel(r.app)
	}
}

// linked appends a link to r's app for each signer apps lacks.
func (r Requester) linked(apps []vault.App) []vault.App {
	linked := slices.Clone(apps)
	for _, signer := range r.signers {
		if link := (vault.App{Package: r.app, Signer: signer}); !slices.Contains(linked, link) {
			linked = append(linked, link)
		}
	}
	return linked
}

// credential is a new credential for r with its origin as website, cut by vault.WebsiteOf, or its app linked.
func (r Requester) credential(label, password string) vault.CredentialInput {
	input := vault.CredentialInput{Label: label, Password: password, Apps: r.linked(nil)}
	if r.app == "" {
		input.Websites = []string{vault.WebsiteOf(r.origin)}
	}
	return input
}

// addition is the patch adding r to credential and whether credential has room for it.
func (r Requester) addition(credential vault.CredentialInput) (vault.CredentialPatch, bool) {
	if r.app == "" {
		websites := append(slices.Clone(credential.Websites), vault.WebsiteOf(r.origin))
		return vault.CredentialPatch{Websites: &websites}, len(websites) <= vault.MaxCredentialWebsites
	}
	apps := r.linked(credential.Apps)
	return vault.CredentialPatch{Apps: &apps}, len(apps) <= vault.MaxCredentialApps
}

// update is the patch saving password, also linking r's app while credential has room.
func (r Requester) update(credential vault.CredentialInput, password string) vault.CredentialPatch {
	patch := vault.CredentialPatch{Password: &password}
	if r.app == "" {
		return patch
	}
	if added, room := r.addition(credential); room {
		patch.Apps = added.Apps
	}
	return patch
}
