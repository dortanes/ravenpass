package confirmation

// ReasonKind is what a verification allows.
type ReasonKind uint8

const (
	// ReasonShare releases one identity file to a site.
	ReasonShare ReasonKind = iota + 1
	// ReasonSavePasskey saves a new passkey for a site.
	ReasonSavePasskey
	// ReasonSignIn signs in to a site with a passkey.
	ReasonSignIn
	// ReasonChangeUnlock sets, changes or removes the vault's PIN, or turns off device authentication.
	ReasonChangeUnlock
	// ReasonCreateVault starts setting up another vault on a device that holds one.
	ReasonCreateVault
	// ReasonOpenVault makes a vault file current on a device that holds another vault.
	ReasonOpenVault
	// ReasonDeleteVault erases a vault's file and this device's records for it.
	ReasonDeleteVault
	// ReasonFill releases a credential's password or one-time code to a site through a linked extension.
	ReasonFill
	// ReasonReveal shows or copies a hidden note or a seed's secret in the app.
	ReasonReveal
	// ReasonFillCard releases a card to a site through a linked extension.
	ReasonFillCard
)

// Reason names what one verification allows; fields its Kind does not use are empty.
type Reason struct {
	Kind     ReasonKind
	File     string
	Identity string
	Site     string
	Account  string
}

// Sharing is the reason for releasing file, of identity, to site.
func Sharing(file, identity, site string) Reason {
	return Reason{Kind: ReasonShare, File: file, Identity: identity, Site: site}
}

// SavingPasskey is the reason for saving a new passkey for site.
func SavingPasskey(site string) Reason {
	return Reason{Kind: ReasonSavePasskey, Site: site}
}

// SigningIn is the reason for signing in to site as account with a passkey.
func SigningIn(site, account string) Reason {
	return Reason{Kind: ReasonSignIn, Site: site, Account: account}
}

// Filling is the reason for filling the sign-in of account, a credential's account or else its label, on site.
func Filling(site, account string) Reason {
	return Reason{Kind: ReasonFill, Site: site, Account: account}
}

// FillingCard is the reason for filling the card labelled card on site.
func FillingCard(site, card string) Reason {
	return Reason{Kind: ReasonFillCard, File: card, Site: site}
}

// Revealing is the reason for showing the secret of the item labelled item.
func Revealing(item string) Reason {
	return Reason{Kind: ReasonReveal, File: item}
}

// ChangingUnlock is the reason for changing how the vault opens on the device.
func ChangingUnlock() Reason {
	return Reason{Kind: ReasonChangeUnlock}
}

// CreatingVault is the reason for starting to set up another vault.
func CreatingVault() Reason {
	return Reason{Kind: ReasonCreateVault}
}

// OpeningVault is the reason for opening a vault file beside the vaults the device holds.
func OpeningVault() Reason {
	return Reason{Kind: ReasonOpenVault}
}

// DeletingVault is the reason for deleting the vault named vault.
func DeletingVault(vault string) Reason {
	return Reason{Kind: ReasonDeleteVault, File: vault}
}

// Wording is the text the system prompt gives for each kind of reason, in the recorded language.
type Wording interface {
	ShareReason(file, identity, site string) string
	SavePasskeyReason(site string) string
	SignInReason(site, account string) string
	FillReason(site, account string) string
	FillCardReason(site, card string) string
	ChangeUnlockReason() string
	CreateVaultReason() string
	OpenVaultReason() string
	DeleteVaultReason(vault string) string
	RevealReason(item string) string
}

// Words is the text wording gives the system prompt for r.
func (r Reason) Words(wording Wording) string {
	switch r.Kind {
	case ReasonSavePasskey:
		return wording.SavePasskeyReason(r.Site)
	case ReasonSignIn:
		return wording.SignInReason(r.Site, r.Account)
	case ReasonFill:
		return wording.FillReason(r.Site, r.Account)
	case ReasonFillCard:
		return wording.FillCardReason(r.Site, r.File)
	case ReasonChangeUnlock:
		return wording.ChangeUnlockReason()
	case ReasonCreateVault:
		return wording.CreateVaultReason()
	case ReasonOpenVault:
		return wording.OpenVaultReason()
	case ReasonDeleteVault:
		return wording.DeleteVaultReason(r.File)
	case ReasonReveal:
		return wording.RevealReason(r.File)
	default:
		return wording.ShareReason(r.File, r.Identity, r.Site)
	}
}
