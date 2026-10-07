package extensionaccess

import (
	"context"
	"errors"
	"slices"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/app/verification"
	"github.com/dortanes/ravenpass/packages/vault"
)

// Cards lists the cards a checkout can take, from the index alone.
func (a *Access) Cards() ([]linkproto.CardOption, error) {
	entries, err := a.credentials.FillableCards()
	if err != nil {
		return nil, refusal(err)
	}
	cards := make([]linkproto.CardOption, len(entries))
	for i, entry := range entries {
		cards[i] = linkproto.CardOption{
			ID: entry.ID.String(), Label: entry.Label, BankName: entry.Detail, Site: entry.Site,
			Network: entry.Card.Network.Name(), LastFour: entry.Card.LastFour, Color: entry.Card.Color, ExpiresOn: entry.ExpiresOn,
		}
	}
	return cards, nil
}

// FillCard releases a card the listing holds to the page at origin once the person verifies this one fill, whatever
// the owner chose for other fills; a vault that offers no way to verify releases it without asking.
func (a *Access) FillCard(ctx context.Context, card, origin string, asked func(linkproto.Progress)) (linkproto.CardFill, error) {
	id, err := vault.ParseID(card)
	if err != nil {
		return linkproto.CardFill{}, linkserver.ErrNotFound
	}
	listed, err := a.credentials.FillableCards()
	if err != nil {
		return linkproto.CardFill{}, refusal(err)
	}
	index := slices.IndexFunc(listed, func(entry vault.Entry) bool { return entry.ID == id })
	if index < 0 {
		return linkproto.CardFill{}, linkserver.ErrNotFound
	}
	reason := confirmation.FillingCard(vaultservice.PageSite(origin), listed[index].Label)
	if err := a.verify(ctx, reason, asked); err != nil && !errors.Is(err, verification.ErrUnverifiable) {
		return linkproto.CardFill{}, refusal(err)
	}
	filled, err := a.credentials.FillCard(id)
	if err != nil {
		return linkproto.CardFill{}, refusal(err)
	}
	fill := linkproto.CardFill{Holder: filled.Holder, Number: filled.Number, Expiry: filled.Expiry, SecurityCode: filled.SecurityCode}
	if billing := filled.BillingAddress(); billing != nil {
		fill.Billing = &linkproto.BillingAddress{
			Street: billing.Street, City: billing.City, Region: billing.Region, PostalCode: billing.PostalCode, Country: billing.Country,
		}
	}
	return fill, nil
}
