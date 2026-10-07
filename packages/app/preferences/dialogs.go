package preferences

import (
	"strings"

	"github.com/dortanes/ravenpass/packages/app/messages"
)

// TitledPrompts words each authentication prompt as a title and a full sentence joined by a line feed, for
// a system prompt that shows both; without it a reason completes "Ravenpass is trying to".
func TitledPrompts() Option {
	return func(s *Store) { s.titledPrompts = true }
}

// Dialogs holds the wording of system dialogs and authentication prompts in the recorded language; a store made
// with TitledPrompts words each reason as a titled prompt.
type Dialogs struct {
	ChooseVaultLocation string
	SelectVaultFile     string
	SaveExport          string
	SaveRecoveryKey     string
	ChoosePhoto         string
	ChooseScan          string
	ChooseQRCode        string
	SaveScan            string
	SelectImport        string
	ChooseBackupFolder  string
	VaultFilter         string
	TextFilter          string
	PhotoFilter         string
	ScanFilter          string
	ImportFilter        string
	VaultFileName       string
	ExportFileName      string
	RecoveryFileName    string
	// ShareFile is a reason with {file}, {identity} and {site} placeholders.
	ShareFile string
	// SavePasskey is a reason with a {site} placeholder.
	SavePasskey string
	// SignInWithPasskey is a reason with {site} and {account} placeholders.
	SignInWithPasskey string
	// FillSignIn is a reason with {site} and {account} placeholders.
	FillSignIn   string
	ChangeUnlock string
	CreateVault  string
	OpenVault    string
	// DeleteVault is a reason with a {vault} placeholder.
	DeleteVault string
	// RevealItem is a reason with an {item} placeholder.
	RevealItem   string
	UnlockVault  string
	OnThisDevice string
}

func dialogsIn(catalog messages.Catalog, titled bool) Dialogs {
	dialogs := Dialogs{
		ChooseVaultLocation: catalog.Text("system.dialog.choose-vault-location"),
		SelectVaultFile:     catalog.Text("system.dialog.select-vault-file"),
		SaveExport:          catalog.Text("system.dialog.save-export"),
		SaveRecoveryKey:     catalog.Text("system.dialog.save-recovery-key"),
		ChoosePhoto:         catalog.Text("system.dialog.choose-photo"),
		ChooseScan:          catalog.Text("system.dialog.choose-scan"),
		ChooseQRCode:        catalog.Text("system.dialog.choose-qr-code"),
		SaveScan:            catalog.Text("system.dialog.save-scan"),
		SelectImport:        catalog.Text("system.dialog.select-import"),
		ChooseBackupFolder:  catalog.Text("system.dialog.choose-backup-folder"),
		VaultFilter:         catalog.Text("system.filter.vault"),
		TextFilter:          catalog.Text("system.filter.text"),
		PhotoFilter:         catalog.Text("system.filter.photo"),
		ScanFilter:          catalog.Text("system.filter.scan"),
		ImportFilter:        catalog.Text("system.filter.import"),
		VaultFileName:       catalog.Text("system.file.vault"),
		ExportFileName:      catalog.Text("system.file.export"),
		RecoveryFileName:    catalog.Text("system.file.recovery-key"),
		ShareFile:           catalog.Text("system.reason.share-file"),
		SavePasskey:         catalog.Text("system.reason.save-passkey"),
		SignInWithPasskey:   catalog.Text("system.reason.sign-in-passkey"),
		FillSignIn:          catalog.Text("system.reason.fill-sign-in"),
		ChangeUnlock:        catalog.Text("system.reason.change-unlock"),
		CreateVault:         catalog.Text("system.reason.create-vault"),
		OpenVault:           catalog.Text("system.reason.open-vault"),
		DeleteVault:         catalog.Text("system.reason.delete-vault"),
		RevealItem:          catalog.Text("system.reason.reveal-item"),
		UnlockVault:         catalog.Text("system.reason.unlock-vault"),
		OnThisDevice:        catalog.Text("system.place.on-this-device"),
	}
	if titled {
		prompt := func(title, sentence string) string { return catalog.Text(title) + "\n" + catalog.Text(sentence) }
		dialogs.ShareFile = prompt("system.prompt.share-file.title", "system.prompt.share-file")
		dialogs.SavePasskey = prompt("system.prompt.save-passkey.title", "system.prompt.save-passkey")
		dialogs.SignInWithPasskey = prompt("system.prompt.sign-in-passkey.title", "system.prompt.sign-in-passkey")
		dialogs.FillSignIn = prompt("system.prompt.fill-sign-in.title", "system.prompt.fill-sign-in")
		dialogs.ChangeUnlock = prompt("system.prompt.change-unlock.title", "system.prompt.change-unlock")
		dialogs.CreateVault = prompt("system.prompt.create-vault.title", "system.prompt.create-vault")
		dialogs.OpenVault = prompt("system.prompt.open-vault.title", "system.prompt.open-vault")
		dialogs.DeleteVault = prompt("system.prompt.delete-vault.title", "system.prompt.delete-vault")
		dialogs.RevealItem = prompt("system.prompt.reveal-item.title", "system.prompt.reveal-item")
		dialogs.UnlockVault = prompt("system.prompt.unlock-vault.title", "system.prompt.unlock-vault")
	}
	return dialogs
}

// ShareReason is the reason the prompt gives for sharing file, of identity, with site.
func (d Dialogs) ShareReason(file, identity, site string) string {
	return strings.NewReplacer("{file}", file, "{identity}", identity, "{site}", site).Replace(d.ShareFile)
}

// SavePasskeyReason is the reason the prompt gives for saving a passkey for site.
func (d Dialogs) SavePasskeyReason(site string) string {
	return strings.NewReplacer("{site}", site).Replace(d.SavePasskey)
}

// SignInReason is the reason the prompt gives for signing in to site as account with a passkey.
func (d Dialogs) SignInReason(site, account string) string {
	return strings.NewReplacer("{site}", site, "{account}", account).Replace(d.SignInWithPasskey)
}

// FillReason is the reason the prompt gives for filling the sign-in of account on site from a linked extension.
func (d Dialogs) FillReason(site, account string) string {
	return strings.NewReplacer("{site}", site, "{account}", account).Replace(d.FillSignIn)
}

// ChangeUnlockReason is the reason the prompt gives for changing how the vault opens.
func (d Dialogs) ChangeUnlockReason() string { return d.ChangeUnlock }

// CreateVaultReason is the reason the prompt gives for starting another vault.
func (d Dialogs) CreateVaultReason() string { return d.CreateVault }

// OpenVaultReason is the reason the prompt gives for opening a vault file.
func (d Dialogs) OpenVaultReason() string { return d.OpenVault }

// DeleteVaultReason is the reason the prompt gives for deleting the vault named vault.
func (d Dialogs) DeleteVaultReason(vault string) string {
	return strings.NewReplacer("{vault}", vault).Replace(d.DeleteVault)
}

// RevealReason is the reason the prompt gives for showing the secret of the item labelled item.
func (d Dialogs) RevealReason(item string) string {
	return strings.NewReplacer("{item}", item).Replace(d.RevealItem)
}

// RecoveryFileText wraps the phrase in the plaintext file the user asked Ravenpass to write.
type RecoveryFileText struct {
	Intro  string
	Notice string
}

// RecoveryFile reports the recovery key file's wording for the recorded language.
func (s *Store) RecoveryFile() RecoveryFileText {
	catalog := s.catalog()
	return RecoveryFileText{
		Intro:  catalog.Text("system.recovery-file.title") + "\n\n",
		Notice: "\n\n" + catalog.Text("system.recovery-file.notice") + "\n",
	}
}

// RecoverySheetText words the printed recovery key page.
type RecoverySheetText struct {
	// Language is the BCP 47 tag of the wording.
	Language string
	Title    string
	// Printed is a line with a {date} placeholder.
	Printed string
	Keep    string
	Warning string
}

// RecoverySheet reports the printed recovery key page's wording for the recorded language.
func (s *Store) RecoverySheet() RecoverySheetText {
	catalog := s.catalog()
	return RecoverySheetText{
		Language: catalog.Language(),
		Title:    catalog.Text("system.recovery-file.title"),
		Printed:  catalog.Text("system.recovery-sheet.printed"),
		Keep:     catalog.Text("system.recovery-sheet.keep"),
		Warning:  catalog.Text("system.recovery-sheet.warning"),
	}
}

// Dialogs reports the dialog text for the recorded language.
func (s *Store) Dialogs() Dialogs {
	return dialogsIn(s.catalog(), s.titledPrompts)
}

// Catalog returns the message catalog of the recorded language.
func (s *Store) Catalog() messages.Catalog {
	return s.catalog()
}

func (s *Store) catalog() messages.Catalog {
	language, _ := s.Language()
	return messages.For(string(language))
}
