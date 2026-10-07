// Package extensionaccess serves the link server's requests from the open vault and its companions.
package extensionaccess

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/dortanes/ravenpass/packages/app/autofill"
	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/siteicons"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/app/verification"
	"github.com/dortanes/ravenpass/packages/authenticator"
	"github.com/dortanes/ravenpass/packages/vault"
)

// Credentials is the open vault as an extension reaches it; a page asks as its origin's requester.
type Credentials interface {
	Unlocked() bool
	Suggestions(requester vaultservice.Requester, purpose vaultservice.Purpose) ([]vaultservice.Suggestion, error)
	MatchingCredential(id vault.ID, requester vaultservice.Requester) (vaultservice.Suggestion, error)
	FillCredential(id vault.ID, requester vaultservice.Requester) (vaultservice.Fill, error)
	OneTimeCode(id vault.ID, requester vaultservice.Requester) (vault.OneTimeCode, error)
	AddWebsite(id vault.ID, origin string) error
	NamesSite(site string) (bool, error)
	CaptureOffer(capture vaultservice.Capture) (vaultservice.CaptureOffer, error)
	SaveCapture(capture vaultservice.Capture, choice vaultservice.CaptureChoice, group string) (bool, error)
	FillableCards() ([]vault.Entry, error)
	FillCard(id vault.ID) (vault.Card, error)
	CardCaptureOffer(capture vaultservice.CardCapture) (vaultservice.CaptureOffer, error)
	SaveCardCapture(capture vaultservice.CardCapture, choice vaultservice.CaptureChoice, group string) (bool, error)
	SignInPasskeys(origin, rpID string, allow [][]byte) (string, []vaultservice.PasskeyChoice, error)
	PasskeyTargets(origin, rpID, account string, exclude [][]byte) (vaultservice.PasskeyTargets, error)
	CreatePasskey(creation vaultservice.PasskeyCreation, group string) (vaultservice.CreatedPasskey, error)
	SignPasskey(signIn vaultservice.PasskeySignIn) (vaultservice.PasskeyAssertion, error)
}

// Identities is the open vault's identities as an extension reaches their files.
type Identities interface {
	IdentityFiles() ([]vaultservice.IdentityFiles, error)
	ReadIdentityFile(identity vault.ID, file string) (vaultservice.FileContent, error)
}

// Icons answers a site's icon as a base64 PNG, empty while site icons are off.
type Icons interface {
	Icon(site string) (string, error)
}

// Verifier asks the person to verify a share or a passkey's creation or use.
type Verifier interface {
	Verify(ctx context.Context, reason confirmation.Reason, asked func(verification.Method)) error
}

// Unlocks asks the person to unlock the vault in the confirmation panel.
type Unlocks interface {
	PostUnlock(requester confirmation.Requester) string
}

// Choices is what the owner recorded for linked extensions: the group new passwords join and whether each fill
// waits for the owner to confirm it.
type Choices interface {
	autofill.GroupChoice
	ConfirmExtensionFills() bool
}

// Access is the link server's Vault.
type Access struct {
	credentials Credentials
	identities  Identities
	icons       Icons
	verifier    Verifier
	choices     Choices
	unlocks     Unlocks
	confirmed   *confirmedFills
}

// New fails when any dependency is nil.
func New(credentials Credentials, identities Identities, icons Icons, verifier Verifier, choices Choices, unlocks Unlocks) (*Access, error) {
	if credentials == nil || identities == nil || icons == nil || verifier == nil || choices == nil || unlocks == nil {
		return nil, errors.New("vault, identities, site icons, verifier, recorded choices and unlock requests are required")
	}
	return &Access{
		credentials: credentials, identities: identities, icons: icons, verifier: verifier, choices: choices, unlocks: unlocks,
		confirmed: newConfirmedFills(time.Now),
	}, nil
}

// Unlocked reports whether the vault is open.
func (a *Access) Unlocked() bool {
	return a.credentials.Unlocked()
}

// Suggest lists the credentials the vault suggests for a page at origin.
func (a *Access) Suggest(origin string, purpose linkproto.Purpose) ([]linkproto.Suggestion, error) {
	wanted, err := servicePurpose(purpose)
	if err != nil {
		return nil, err
	}
	suggestions, err := a.credentials.Suggestions(vaultservice.OriginRequester(origin), wanted)
	if err != nil {
		return nil, refusal(err)
	}
	result := make([]linkproto.Suggestion, len(suggestions))
	for i, suggestion := range suggestions {
		result[i] = linkproto.Suggestion{
			ID: suggestion.ID.String(), Label: suggestion.Label, Account: suggestion.Account, Site: suggestion.Site, Exact: suggestion.Exact,
			Digits: suggestion.Code.Digits, Period: suggestion.Code.Period, Tags: suggestion.Tags,
		}
	}
	return result, nil
}

// servicePurpose is purpose as the vault service names it.
func servicePurpose(purpose linkproto.Purpose) (vaultservice.Purpose, error) {
	switch purpose {
	case linkproto.PurposeSignIn:
		return vaultservice.PurposeSignIn, nil
	case linkproto.PurposeCode:
		return vaultservice.PurposeCode, nil
	default:
		return 0, fmt.Errorf("unknown suggestion purpose %q", purpose)
	}
}

// Fill answers ErrNotFound for a credential that is not a vault identifier, and confirms the fill as confirmFill does;
// extension filling the same credential on the same page again within confirmedFillLifetime is not asked a second time.
func (a *Access) Fill(ctx context.Context, extension, credential, origin string, asked func(linkproto.Progress)) (linkproto.Fill, error) {
	id, err := credentialID(credential)
	if err != nil {
		return linkproto.Fill{}, err
	}
	confirmed := confirmedFill{extension: extension, credential: id, origin: origin}
	if !a.confirmed.take(confirmed) {
		verified, err := a.confirmFill(ctx, id, origin, asked)
		if err != nil {
			return linkproto.Fill{}, err
		}
		if verified {
			a.confirmed.keep(confirmed)
		}
	}
	fill, err := a.credentials.FillCredential(id, vaultservice.OriginRequester(origin))
	if err != nil {
		return linkproto.Fill{}, refusal(err)
	}
	return linkproto.Fill{Login: fill.Login, Email: fill.Email, Password: fill.Password}, nil
}

// OneTimeCode answers ErrNotFound for a credential that is not a vault identifier, and confirms the fill as confirmFill does.
func (a *Access) OneTimeCode(ctx context.Context, credential, origin string, asked func(linkproto.Progress)) (linkproto.OneTimeCode, error) {
	id, err := credentialID(credential)
	if err != nil {
		return linkproto.OneTimeCode{}, err
	}
	if _, err := a.confirmFill(ctx, id, origin, asked); err != nil {
		return linkproto.OneTimeCode{}, err
	}
	code, err := a.credentials.OneTimeCode(id, vaultservice.OriginRequester(origin))
	if err != nil {
		return linkproto.OneTimeCode{}, refusal(err)
	}
	return linkproto.OneTimeCode{Code: code.Code, Digits: code.Digits, Period: code.Period, ExpiresAt: code.ExpiresAt.UnixMilli()}, nil
}

// AddWebsite adds origin to credential only while the credential matches the page at origin, so a page never joins
// a credential of another site.
func (a *Access) AddWebsite(credential, origin string) error {
	id, err := credentialID(credential)
	if err != nil {
		return err
	}
	if _, err := a.credentials.MatchingCredential(id, vaultservice.OriginRequester(origin)); err != nil {
		return refusal(err)
	}
	if err := a.credentials.AddWebsite(id, origin); err != nil {
		return refusal(err)
	}
	return nil
}

// confirmFill asks the person to verify filling credential id on the page at origin while the owner's choice asks
// for it, and reports whether it asked; the match is checked first, so a refused credential asks no one.
func (a *Access) confirmFill(ctx context.Context, id vault.ID, origin string, asked func(linkproto.Progress)) (bool, error) {
	if !a.choices.ConfirmExtensionFills() {
		return false, nil
	}
	matched, err := a.credentials.MatchingCredential(id, vaultservice.OriginRequester(origin))
	if err != nil {
		return false, refusal(err)
	}
	if err := a.verify(ctx, confirmation.Filling(vaultservice.PageSite(origin), cmp.Or(matched.Account, matched.Label)), asked); err != nil {
		return false, refusal(err)
	}
	return true, nil
}

// credentialID is the vault identifier a request names, ErrNotFound for text that is not one.
func credentialID(credential string) (vault.ID, error) {
	id, err := vault.ParseID(credential)
	if err != nil {
		return vault.ID{}, linkserver.ErrNotFound
	}
	return id, nil
}

// SiteIcon answers an empty icon for a site the vault does not name, keeping fetches off arbitrary addresses.
func (a *Access) SiteIcon(site string) (linkproto.Icon, error) {
	named, err := a.credentials.NamesSite(site)
	if err != nil {
		return linkproto.Icon{}, refusal(err)
	}
	if !named {
		return linkproto.Icon{}, nil
	}
	icon, err := a.icons.Icon(site)
	if err != nil {
		return linkproto.Icon{}, refusal(err)
	}
	return linkproto.Icon{Image: icon, Tint: siteicons.Tint(icon)}, nil
}

// ShowUnlock asks the person to unlock the vault in the confirmation panel, unless it is open.
func (a *Access) ShowUnlock() {
	if !a.credentials.Unlocked() {
		a.unlocks.PostUnlock(confirmation.RequesterExtension)
	}
}

var fileKinds = map[vaultservice.FileKind]linkproto.FileKind{
	vaultservice.FilePhoto: linkproto.FilePhoto,
	vaultservice.FileScan:  linkproto.FileScan,
}

// Identities lists every identity with the files it holds, thumbnails in base64.
func (a *Access) Identities() ([]linkproto.Identity, error) {
	held, err := a.identities.IdentityFiles()
	if err != nil {
		return nil, refusal(err)
	}
	identities := make([]linkproto.Identity, len(held))
	for i, identity := range held {
		files := make([]linkproto.IdentityFile, len(identity.Files))
		for j, file := range identity.Files {
			files[j] = linkproto.IdentityFile{
				ID: file.ID, Kind: fileKinds[file.Kind], Name: file.Name, MediaType: file.MediaType,
				Thumbnail: base64.StdEncoding.EncodeToString(file.Thumbnail),
			}
			if file.Document != nil {
				files[j].Document = &linkproto.Document{Type: file.Document.Type.Name(), Label: file.Document.Label}
			}
		}
		identities[i] = linkproto.Identity{ID: identity.ID.String(), Label: identity.Label, Thumbnail: base64.StdEncoding.EncodeToString(identity.Thumbnail), Files: files}
	}
	return identities, nil
}

var progressOf = map[verification.Method]linkproto.Progress{
	verification.MethodDevice: linkproto.ProgressConfirmOnDevice,
	verification.MethodPIN:    linkproto.ProgressConfirmInRavenpass,
}

// Share releases one of an identity's files to a page at origin once the person verifies the share.
func (a *Access) Share(ctx context.Context, identity, file, origin string, asked func(linkproto.Progress)) (linkproto.SharedFile, []byte, error) {
	owner, err := vault.ParseID(identity)
	if err != nil {
		return linkproto.SharedFile{}, nil, linkserver.ErrNotFound
	}
	held, err := a.identities.IdentityFiles()
	if err != nil {
		return linkproto.SharedFile{}, nil, refusal(err)
	}
	name, label, found := describe(held, owner, file)
	if !found {
		return linkproto.SharedFile{}, nil, linkserver.ErrNotFound
	}
	if err := a.verify(ctx, confirmation.Sharing(name, label, vaultservice.PageSite(origin)), asked); err != nil {
		return linkproto.SharedFile{}, nil, refusal(err)
	}
	content, err := a.identities.ReadIdentityFile(owner, file)
	if err != nil {
		return linkproto.SharedFile{}, nil, refusal(err)
	}
	return linkproto.SharedFile{Name: content.Name, MediaType: content.MediaType, Size: len(content.Content)}, content.Content, nil
}

// describe reports the name of file and the label of its identity owner.
func describe(held []vaultservice.IdentityFiles, owner vault.ID, file string) (string, string, bool) {
	for _, identity := range held {
		if identity.ID != owner {
			continue
		}
		for _, candidate := range identity.Files {
			if candidate.ID == file {
				return candidate.Name, identity.Label, true
			}
		}
	}
	return "", "", false
}

// verify asks the person to verify reason, reporting through asked what they are asked to do.
func (a *Access) verify(ctx context.Context, reason confirmation.Reason, asked func(linkproto.Progress)) error {
	return a.verifier.Verify(ctx, reason, func(method verification.Method) { asked(progressOf[method]) })
}

// refusal is err as the link server's refusal, when it is one.
func refusal(err error) error {
	switch {
	case errors.Is(err, vaultservice.ErrNotReady):
		return linkserver.ErrLocked
	case errors.Is(err, vault.ErrNotFound):
		return linkserver.ErrNotFound
	case errors.Is(err, vaultservice.ErrNoMatch):
		return linkserver.ErrNoMatch
	case errors.Is(err, vault.ErrNoTOTP):
		return linkserver.ErrNoCode
	case errors.Is(err, verification.ErrDeclined):
		return linkserver.ErrDeclined
	case errors.Is(err, verification.ErrUnverifiable):
		return linkserver.ErrUnverifiable
	case errors.Is(err, vaultservice.ErrAccountRefused):
		return linkserver.ErrInvalidAccount
	case errors.Is(err, vaultservice.ErrNameRefused):
		return linkserver.ErrInvalidName
	case errors.Is(err, authenticator.ErrInvalidOrigin):
		return linkserver.ErrInvalidOrigin
	case errors.Is(err, authenticator.ErrInvalidRelyingParty):
		return linkserver.ErrInvalidRelyingParty
	case errors.Is(err, vaultservice.ErrPasskeyExcluded):
		return linkserver.ErrExcluded
	case errors.Is(err, vault.ErrPasskeysFull):
		return linkserver.ErrPasskeysFull
	default:
		return err
	}
}
