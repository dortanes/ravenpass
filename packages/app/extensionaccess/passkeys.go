package extensionaccess

import (
	"context"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/vault"
)

// Passkeys lists the passkeys a sign-in at the query's origin can use.
func (a *Access) Passkeys(query linkproto.PasskeyQuery) (linkproto.Passkeys, error) {
	rpID, choices, err := a.credentials.SignInPasskeys(query.Origin, query.RPID, query.Allow)
	if err != nil {
		return linkproto.Passkeys{}, refusal(err)
	}
	answer := linkproto.Passkeys{RPID: rpID, Passkeys: make([]linkproto.PasskeyOption, len(choices))}
	for i, choice := range choices {
		answer.Passkeys[i] = linkproto.PasskeyOption{
			Credential: choice.Credential.String(), CredentialID: choice.CredentialID, Account: choice.Account, Label: choice.Label,
		}
	}
	return answer, nil
}

// PasskeyTargets lists the credentials a new passkey for the query's account can join.
func (a *Access) PasskeyTargets(query linkproto.PasskeyQuery) (linkproto.PasskeyTargets, error) {
	targets, err := a.credentials.PasskeyTargets(query.Origin, query.RPID, query.Account, query.Exclude)
	if err != nil {
		return linkproto.PasskeyTargets{}, refusal(err)
	}
	answer := linkproto.PasskeyTargets{RPID: targets.RPID, Targets: make([]linkproto.PasskeyTarget, len(targets.Targets)), Excluded: targets.Excluded}
	for i, target := range targets.Targets {
		answer.Targets[i] = linkproto.PasskeyTarget{Credential: target.ID.String(), Label: target.Label, Account: target.Account, Tags: target.Tags}
	}
	return answer, nil
}

// CreatePasskey saves a new passkey where the creation names; exclusions are checked before verification.
func (a *Access) CreatePasskey(ctx context.Context, creation linkproto.PasskeyCreation, asked func(linkproto.Progress)) (linkproto.CreatedPasskey, error) {
	var target vault.ID
	if creation.Target != linkproto.TargetNew {
		id, err := credentialID(creation.Target)
		if err != nil {
			return linkproto.CreatedPasskey{}, err
		}
		target = id
	}
	targets, err := a.credentials.PasskeyTargets(creation.Origin, creation.RPID, creation.User.Name, creation.Exclude)
	if err != nil {
		return linkproto.CreatedPasskey{}, refusal(err)
	}
	if targets.Excluded {
		return linkproto.CreatedPasskey{}, linkserver.ErrExcluded
	}
	if creation.Verify {
		if err := a.verify(ctx, confirmation.SavingPasskey(vault.SiteName(targets.RPID)), asked); err != nil {
			return linkproto.CreatedPasskey{}, refusal(err)
		}
	}
	created, err := a.credentials.CreatePasskey(vaultservice.PasskeyCreation{
		Origin: creation.Origin, RPID: creation.RPID, RPName: creation.RPName,
		User:      vaultservice.PasskeyUser{Handle: creation.User.ID, Name: creation.User.Name, DisplayName: creation.User.DisplayName},
		Challenge: creation.Challenge, Exclude: creation.Exclude, Verified: creation.Verify, Target: target,
	}, a.choices.DefaultGroup())
	if err != nil {
		return linkproto.CreatedPasskey{}, refusal(err)
	}
	return linkproto.CreatedPasskey{
		Credential:         created.Credential.String(),
		CredentialID:       created.CredentialID,
		ClientDataJSON:     created.ClientData,
		AttestationObject:  created.Attestation.AttestationObject,
		AuthenticatorData:  created.Attestation.AuthenticatorData,
		PublicKey:          created.Attestation.PublicKey,
		PublicKeyAlgorithm: created.Attestation.Algorithm,
	}, nil
}

// SignPasskey signs a sign-in with the passkey it names; the passkey is checked before verification.
func (a *Access) SignPasskey(ctx context.Context, signIn linkproto.PasskeySignIn, asked func(linkproto.Progress)) (linkproto.PasskeyAssertion, error) {
	id, err := credentialID(signIn.Credential)
	if err != nil {
		return linkproto.PasskeyAssertion{}, err
	}
	rpID, choices, err := a.credentials.SignInPasskeys(signIn.Origin, signIn.RPID, [][]byte{signIn.CredentialID})
	if err != nil {
		return linkproto.PasskeyAssertion{}, refusal(err)
	}
	held, found := vaultservice.FindPasskey(choices, id, signIn.CredentialID)
	if !found {
		return linkproto.PasskeyAssertion{}, linkserver.ErrNotFound
	}
	if signIn.Verify {
		if err := a.verify(ctx, confirmation.SigningIn(vault.SiteName(rpID), held.Account), asked); err != nil {
			return linkproto.PasskeyAssertion{}, refusal(err)
		}
	}
	signed, err := a.credentials.SignPasskey(vaultservice.PasskeySignIn{
		Origin: signIn.Origin, RPID: signIn.RPID, Challenge: signIn.Challenge, Verified: signIn.Verify,
		Credential: id, CredentialID: signIn.CredentialID,
	})
	if err != nil {
		return linkproto.PasskeyAssertion{}, refusal(err)
	}
	return linkproto.PasskeyAssertion{
		CredentialID:      signed.CredentialID,
		ClientDataJSON:    signed.ClientData,
		AuthenticatorData: signed.AuthenticatorData,
		Signature:         signed.Signature,
		UserHandle:        signed.UserHandle,
	}, nil
}
