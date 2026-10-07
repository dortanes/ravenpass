package autofill

import (
	"encoding/base64"

	"github.com/dortanes/ravenpass/packages/app/autofill"
	"github.com/dortanes/ravenpass/packages/vault"
)

type status string

const (
	statusOK         status = "ok"
	statusNone       status = "none"
	statusLocked     status = "locked"
	statusCanceled   status = "canceled"
	statusWrongPIN   status = "wrong-pin"
	statusPINRemoved status = "pin-removed"
	// statusTooSoon refuses a PIN attempt made before the delay earned by wrong ones has passed.
	statusTooSoon status = "too-soon"
	// statusNoCode reports a credential without a one-time code setup for a code field.
	statusNoCode status = "no-code"
	// statusNameRefused and statusAccountRefused report a new credential's name or account the vault rejects.
	statusNameRefused    status = "name-refused"
	statusAccountRefused status = "account-refused"
	// statusPIN asks for the vault's PIN to confirm a passkey request that did not carry it.
	statusPIN status = "pin"
	// statusExcluded reports a passkey creation meeting a passkey the vault holds that it excluded.
	statusExcluded status = "excluded"
	statusFailed   status = "failed"
)

// outcome heads every answer.
type outcome struct {
	Status status `json:"status"`
}

// appWire's Signers are its certificates' SHA-256 digests in standard base64.
type appWire struct {
	Package string   `json:"package"`
	Signers []string `json:"signers"`
}

// requesterWire comes back from Java with each request its screens make for that requester.
type requesterWire struct {
	Origin string   `json:"origin,omitempty"`
	App    *appWire `json:"app,omitempty"`
	Sites  []string `json:"sites,omitempty"`
}

type suggestionWire struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Account string   `json:"account"`
	Site    string   `json:"site"`
	Tags    []string `json:"tags,omitempty"`
}

type targetWire struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Account string   `json:"account"`
	Action  string   `json:"action"`
	Tags    []string `json:"tags,omitempty"`
}

// offerWire's Site is empty for an app that no site verified.
type offerWire struct {
	Token     string       `json:"token"`
	Name      string       `json:"name"`
	Site      string       `json:"site"`
	Account   string       `json:"account"`
	Targets   []targetWire `json:"targets"`
	Suggested string       `json:"suggested"`
}

type choiceWire struct {
	Target  string `json:"target"`
	Name    string `json:"name"`
	Account string `json:"account"`
}

func (w appWire) app() (autofill.App, bool) {
	if w.Package == "" || len(w.Signers) == 0 {
		return autofill.App{}, false
	}
	app := autofill.App{Package: w.Package}
	for _, text := range w.Signers {
		var digest [32]byte
		decoded, err := base64.StdEncoding.DecodeString(text)
		if err != nil || len(decoded) != len(digest) {
			return autofill.App{}, false
		}
		copy(digest[:], decoded)
		app.Signers = append(app.Signers, digest)
	}
	return app, true
}

func appOf(app autofill.App) *appWire {
	wire := &appWire{Package: app.Package}
	for _, signer := range app.Signers {
		wire.Signers = append(wire.Signers, base64.StdEncoding.EncodeToString(signer[:]))
	}
	return wire
}

// requester accepts exactly one of an origin or an app.
func (w requesterWire) requester() (autofill.Requester, bool) {
	if (w.Origin == "") == (w.App == nil) {
		return autofill.Requester{}, false
	}
	if w.Origin != "" {
		return autofill.Requester{Origin: w.Origin}, true
	}
	app, ok := w.App.app()
	return autofill.Requester{App: app, Sites: w.Sites}, ok
}

func requesterOf(r autofill.Requester) *requesterWire {
	if r.Origin != "" {
		return &requesterWire{Origin: r.Origin}
	}
	return &requesterWire{App: appOf(r.App), Sites: r.Sites}
}

func suggestionsOf(found []autofill.Suggestion) []suggestionWire {
	wires := make([]suggestionWire, len(found))
	for i, s := range found {
		wires[i] = suggestionOf(s)
	}
	return wires
}

func suggestionOf(s autofill.Suggestion) suggestionWire {
	return suggestionWire{ID: s.ID, Label: s.Label, Account: s.Account, Site: s.Site, Tags: s.Tags}
}

func offerOf(offer autofill.Offer, captured autofill.Capture) offerWire {
	wire := offerWire{Token: offer.Token, Name: offer.Name, Site: siteOf(captured.Requester), Account: captured.Account,
		Suggested: offer.Suggested, Targets: make([]targetWire, len(offer.Targets))}
	for i, t := range offer.Targets {
		wire.Targets[i] = targetWire{ID: t.ID, Label: t.Label, Account: t.Account, Action: string(t.Action), Tags: t.Tags}
	}
	return wire
}

// siteOf is the site r's icon is kept under.
func siteOf(r autofill.Requester) string {
	switch {
	case r.Origin != "":
		return vault.SiteOf(r.Origin)
	case len(r.Sites) > 0:
		return r.Sites[0]
	default:
		return ""
	}
}
