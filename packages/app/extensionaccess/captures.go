package extensionaccess

import (
	"errors"

	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/vault"
)

var saveActions = map[vaultservice.TargetAction]linkproto.SaveAction{
	vaultservice.TargetUpdate:  linkproto.SaveUpdate,
	vaultservice.TargetAddSite: linkproto.SaveAddSite,
}

// CaptureOffer answers where a password submitted on a page could be saved; CaptureLocked while locked.
func (a *Access) CaptureOffer(capture linkserver.Capture) (linkproto.CaptureOffer, error) {
	captured := serviceCapture(capture)
	answer := linkproto.CaptureOffer{
		State: linkproto.CaptureReady, Site: vaultservice.PageSite(capture.Origin), Account: capture.Account, Name: captured.Requester.Name(),
		Targets: []linkproto.SaveTarget{},
	}
	offer, err := a.credentials.CaptureOffer(captured)
	switch {
	case errors.Is(err, vaultservice.ErrNotReady):
		answer.State = linkproto.CaptureLocked
		return answer, nil
	case err != nil:
		return linkproto.CaptureOffer{}, refusal(err)
	case offer.Nothing:
		answer.State = linkproto.CaptureNone
		return answer, nil
	}
	for _, target := range offer.Targets {
		answer.Targets = append(answer.Targets, linkproto.SaveTarget{
			Credential: target.ID.String(), Label: target.Label, Account: target.Account, Action: saveActions[target.Action], Tags: target.Tags,
		})
	}
	if offer.Suggested != (vault.ID{}) {
		answer.Suggested = offer.Suggested.String()
	}
	return answer, nil
}

// SaveCapture saves a capture where the person chose; a new credential joins the default group.
func (a *Access) SaveCapture(capture linkserver.Capture, choice linkserver.SaveChoice) (linkproto.Saved, error) {
	chosen := vaultservice.CaptureChoice{Name: choice.Name, Account: choice.Account}
	if choice.Target != "" {
		target, err := credentialID(choice.Target)
		if err != nil {
			return linkproto.Saved{}, err
		}
		chosen.Target = target
	}
	created, err := a.credentials.SaveCapture(serviceCapture(capture), chosen, a.choices.DefaultGroup())
	if err != nil {
		return linkproto.Saved{}, refusal(err)
	}
	if created {
		return linkproto.Saved{Saved: linkproto.SavedCreated}, nil
	}
	return linkproto.Saved{Saved: linkproto.SavedUpdated}, nil
}

func serviceCapture(capture linkserver.Capture) vaultservice.Capture {
	return vaultservice.Capture{Requester: vaultservice.OriginRequester(capture.Origin), Account: capture.Account, Password: capture.Password, Current: capture.Current}
}
