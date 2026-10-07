// Package api is the service the interface calls through its Wails bindings.
package api

import (
	"crypto/sha256"
	"errors"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dortanes/ravenpass/packages/app/appbundle"
	"github.com/dortanes/ravenpass/packages/app/backups"
	"github.com/dortanes/ravenpass/packages/app/breaches"
	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/genhistory"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/linkstore"
	"github.com/dortanes/ravenpass/packages/app/ownerauth"
	"github.com/dortanes/ravenpass/packages/app/pasteboard"
	"github.com/dortanes/ravenpass/packages/app/photos"
	"github.com/dortanes/ravenpass/packages/app/preferences"
	"github.com/dortanes/ravenpass/packages/app/savedfile"
	"github.com/dortanes/ravenpass/packages/app/siteicons"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/unlock"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/app/verification"
	"github.com/dortanes/ravenpass/packages/vault"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// MaxVaultBytes is the largest vault file the service reads.
const MaxVaultBytes int64 = vault.MaxContainerBytes

// ErrFileUnverified reports a saved file that does not read back as it was written.
var ErrFileUnverified = savedfile.ErrUnverified

// State is the phase of the vault the interface shows.
type State struct {
	Phase string `json:"phase"`
}

// RecoveryPreview is what confirming a staged recovery would do.
type RecoveryPreview struct {
	MayLoseNewerCredentials bool `json:"mayLoseNewerCredentials"`
	// KeyReplaced is true when the file is sealed under a recovery key this device saw replaced.
	KeyReplaced bool `json:"keyReplaced"`
	// NeedsWayIn is true when this device holds no way in that still opens the vault.
	NeedsWayIn bool `json:"needsWayIn"`
}

// CredentialInput is the editable content of a credential.
type CredentialInput struct {
	Label    string   `json:"label"`
	Websites []string `json:"websites"`
	Login    string   `json:"login"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Notes    string   `json:"notes"`
	TOTP     string   `json:"totp"`
	// Apps are the apps the credential signs in to. A new credential has none.
	Apps []LinkedApp `json:"apps"`
	// Tags tell the item apart from others like it, such as two accounts on one site.
	Tags []string `json:"tags"`
}

// CredentialSummary is a credential as the list shows it.
type CredentialSummary struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Login string `json:"login"`
	// Site is the host the website icon is keyed by, empty when no website names a site.
	Site string `json:"site"`
	// Sites are the distinct hosts of the credential's websites in order, Site first.
	Sites      []string `json:"sites"`
	Email      string   `json:"email"`
	Pinned     bool     `json:"pinned"`
	LastUsedAt int64    `json:"lastUsedAt"`
	Groups     []string `json:"groups"`
	Tags       []string `json:"tags"`
	// Sites, OneTimeCode and Passkeys come from the index, without decrypting the record.
	OneTimeCode bool `json:"oneTimeCode"`
	Passkeys    int  `json:"passkeys"`
}

// Credential is an opened credential with its groups and passkeys.
type Credential struct {
	ID       string        `json:"id"`
	Groups   []string      `json:"groups"`
	Site     string        `json:"site"`
	Passkeys []PasskeyView `json:"passkeys"`
	CredentialInput
}

// ExportStatus reports whether the last encrypted copy matches the vault.
type ExportStatus struct {
	State string `json:"state"`
}

// Service is the vault application service bound to the interface.
type Service struct {
	vault       *vaultservice.Service
	preferences *preferences.Store
	icons       *siteicons.Service
	brands      brandSource
	breaches    breachCounter
	currentApp  func() *application.App
	clipboard   clipboardState
	pasteboard  Pasteboard
	files       VaultFiles
	moveTo      chosenMove
	places      VaultPlaces
	openURL     func(address string) error
	hold        func() (release func())
	dialog      fileDialog
	photoPicker PhotoPicker
	saver       FileSaver
	printer     Printer
	qrCodes     QRCodeReader
	photo       photoSlot
	scans       scanStaging
	imports     importStaging
	codeSetup   heldCodeSetup
	links       ExtensionLinker
	identities  IdentityList
	offers      Capabilities
	shows       *UnlockOnShow
	screens     ScreenCapture
	// systemAutofill is the device's choice of autofill service and passkey providers.
	systemAutofill SystemAutofill
	// confirmations are the requests the confirmation panel shows.
	confirmations *confirmation.Queue
	panel         ConfirmationPanel
	showMain      func()
	reloadMain    func()
	owner         OwnerVerifier
	// backups is nil on a host without automatic backups.
	backups *backups.Keeper
	folders BackupFolders
	about   About
	apps    AppNames
	// generator is nil on a host that keeps no generator history.
	generator *genhistory.Store
}

// Confirmations is how the service reaches the confirmation panel.
type Confirmations struct {
	Queue *confirmation.Queue
	// Panel is nil on a host without a panel window.
	Panel ConfirmationPanel
	// ShowMain brings the main window forward.
	ShowMain func()
	// ReloadMain reloads an open main window after the panel changed what it shows.
	ReloadMain func()
	// Owner verifies the owner with device authentication or the PIN in the panel.
	Owner OwnerVerifier
}

// Host is what the host app wires beyond the vault; a nil field selects the desktop default or no feature.
type Host struct {
	// Links is nil on a host browser extensions cannot reach.
	Links ExtensionLinker
	// IdentityList is nil on a host whose system keeps no list of accounts to suggest.
	IdentityList  IdentityList
	Confirmations Confirmations
	CurrentApp    func() *application.App
	// Pasteboard is nil where the system clipboard package reaches the clipboard.
	Pasteboard Pasteboard
	// Files is nil where the owner picks vault files with the desktop file dialogs.
	Files VaultFiles
	// Places is nil on a host that names no folder a local vault file is in.
	Places VaultPlaces
	// Photos is nil where the owner picks photos with the desktop file dialog.
	Photos PhotoPicker
	// Saver is nil where the owner saves files with the desktop save dialog.
	Saver FileSaver
	// Printer is nil on a host without a system print dialog.
	Printer Printer
	// QRCodes is nil on a host that reads no QR code pictures.
	QRCodes QRCodeReader
	// OpenURL is nil where the Wails browser manager opens web addresses.
	OpenURL func(address string) error
	// Hold is held while a picker shows, so a host that locks when hidden keeps the vault open.
	Hold func() (release func())
	// UnlockOnShow is set on a host whose locked screen asks for device unlock as the app comes into view.
	UnlockOnShow *UnlockOnShow
	// Screenshots is set on a host that keeps its windows out of screenshots.
	Screenshots ScreenCapture
	// SystemAutofill is set on a host whose system chooses autofill and passkey providers on a system screen.
	SystemAutofill SystemAutofill
	// Backups is set on a host that backs up the open vault on its own.
	Backups *backups.Keeper
	// Generator is set on a host that keeps the passwords the generator handed out.
	Generator *genhistory.Store
	// BackupFolders is nil where the owner picks the backup folder with the desktop folder dialog.
	BackupFolders BackupFolders
	// About is nil where the process runs from a macOS app bundle.
	About About
	// Apps is nil on a host that runs no Android apps.
	Apps AppNames
	// TemporaryPicks reports a file dialog that hands over a plaintext copy the service deletes after reading.
	TemporaryPicks bool
	Offers         Capabilities
}

// New builds the service over the vault, preferences, site icons and the host's wiring.
func New(vault *vaultservice.Service, settings *preferences.Store, icons *siteicons.Service, host Host) (*Service, error) {
	if vault == nil || settings == nil || icons == nil || host.CurrentApp == nil {
		return nil, errors.New("vault service, preferences, site icons and app provider are required")
	}
	confirmations := host.Confirmations
	if confirmations.Queue == nil || confirmations.ShowMain == nil || confirmations.ReloadMain == nil || confirmations.Owner == nil {
		return nil, errors.New("the confirmation queue, the main window and the owner verifier are required")
	}
	offers := host.Offers
	var links ExtensionLinker = noExtensions{}
	if host.Links != nil {
		links = host.Links
		offers.Extensions = true
	}
	var identities IdentityList = noIdentityList{}
	if host.IdentityList != nil {
		identities = host.IdentityList
		offers.IdentityList = true
	}
	var panel ConfirmationPanel = noPanel{}
	if confirmations.Panel != nil {
		panel = confirmations.Panel
	}
	var clipboard Pasteboard = pasteboard.System{}
	if host.Pasteboard != nil {
		clipboard = host.Pasteboard
	}
	var files VaultFiles = dialogFiles{currentApp: host.CurrentApp, dialogs: settings.Dialogs}
	if host.Files != nil {
		files = host.Files
	}
	var places VaultPlaces = noPlaces{}
	if host.Places != nil {
		places = host.Places
	}
	hold := host.Hold
	if hold == nil {
		hold = noHold
	}
	dialog := fileDialog{currentApp: host.CurrentApp, hold: hold, temporary: host.TemporaryPicks}
	unheldDialog := dialog
	unheldDialog.hold = noHold
	var photoPicker PhotoPicker = dialogPhotos{dialog: unheldDialog, dialogs: settings.Dialogs}
	if host.Photos != nil {
		photoPicker = host.Photos
		offers.PhotoPicker = true
	}
	var saver FileSaver = dialogSaver{currentApp: host.CurrentApp}
	if host.Saver != nil {
		saver = host.Saver
		offers.SaveFiles = true
	}
	var printer Printer = noPrinter{}
	if host.Printer != nil {
		printer = host.Printer
		offers.Print = true
	}
	offers.QRCodes = host.QRCodes != nil
	openURL := host.OpenURL
	if openURL == nil {
		openURL = func(address string) error {
			app := host.CurrentApp()
			if app == nil {
				return fail(failureWindowUnavailable)
			}
			if err := app.Browser.OpenURL(address); err != nil {
				return fail(failureWebsiteOpenFailed)
			}
			return nil
		}
	}
	shows := host.UnlockOnShow
	if shows != nil {
		offers.UnlockOnShow = true
	} else {
		shows = &UnlockOnShow{}
	}
	var screens ScreenCapture = noScreenCapture{}
	if host.Screenshots != nil {
		screens = host.Screenshots
		offers.Screenshots = true
		screens.AllowScreenshots(settings.ScreenshotsAllowed())
	}
	var systemAutofill SystemAutofill = noSystemAutofill{}
	if host.SystemAutofill != nil {
		systemAutofill = host.SystemAutofill
		offers.SystemAutofill = true
	}
	offers.AutoBackups = host.Backups != nil
	var folders BackupFolders = dialogFolders{currentApp: host.CurrentApp}
	if host.BackupFolders != nil {
		folders = host.BackupFolders
	}
	var about About = appbundle.Running{}
	if host.About != nil {
		about = host.About
	}
	var apps AppNames = noAppNames{}
	if host.Apps != nil {
		apps = host.Apps
	}
	offers.LockWhenHidden = settings.LocksWhenHidden()
	s := &Service{
		vault: vault, preferences: settings, icons: icons, brands: siteicons.NewFetcher(), breaches: breaches.New(), currentApp: host.CurrentApp,
		pasteboard: clipboard, files: heldFiles{VaultFiles: files, hold: hold}, places: places, openURL: openURL, hold: hold, dialog: dialog,
		photoPicker: heldPhotos{PhotoPicker: photoPicker, hold: hold}, saver: saver, printer: printer, qrCodes: host.QRCodes, links: links, identities: identities, offers: offers,
		shows: shows, screens: screens, systemAutofill: systemAutofill,
		confirmations: confirmations.Queue, panel: panel,
		showMain: confirmations.ShowMain, reloadMain: confirmations.ReloadMain, owner: confirmations.Owner,
		backups: host.Backups, generator: host.Generator, folders: folders, about: about, apps: apps,
	}
	s.confirmations.UnlockOnDevice(s.unlockOnDevice)
	vault.OnLock(s.lockedInside)
	return s, nil
}

// GetState reports the vault's phase, abandoning an unfinished creation or recovery; a location that cannot be reached,
// or whose vault file cannot be read, is the storage phase.
func (s *Service) GetState() (State, error) {
	if location := s.vault.Storage(); !location.Available {
		return State{Phase: "storage"}, nil
	}
	status, err := s.vault.State()
	if errors.Is(err, storage.ErrUnavailable) {
		return State{Phase: "storage"}, nil
	}
	if err != nil {
		return State{}, present(err)
	}
	if status.Phase == vaultservice.PhaseCreating || status.Phase == vaultservice.PhaseRecovering || status.Phase == vaultservice.PhaseDiverged {
		s.vault.Lock()
		status, err = s.vault.State()
		if err != nil {
			return State{}, present(err)
		}
	}
	if status.Phase == vaultservice.PhaseReady {
		return State{Phase: "ready"}, nil
	}
	if status.VaultExists || status.VaultMissing {
		return State{Phase: "locked"}, nil
	}
	return State{Phase: "setup"}, nil
}

// BeginCreation stages a new vault and returns its recovery phrase.
func (s *Service) BeginCreation() (string, error) {
	phrase, err := s.vault.BeginCreation()
	return phrase, present(err)
}

// UnlockChoice is how a vault being created or recovered will open on this device.
type UnlockChoice struct {
	Biometry bool `json:"biometry"`
	// PIN is empty when no PIN is chosen.
	PIN string `json:"pin"`
}

func (c UnlockChoice) methods() vaultservice.MethodChoice {
	return vaultservice.MethodChoice{Biometry: c.Biometry, PIN: c.PIN}
}

// ConfirmCreation creates the staged vault once phrase matches its recovery key.
func (s *Service) ConfirmCreation(phrase string, choice UnlockChoice) error {
	_, err := s.vault.ConfirmCreation(phrase, choice.methods())
	return s.opened(err)
}

// Unlock opens the vault with device authentication, prompting in the recorded language.
func (s *Service) Unlock() error {
	_, err := s.vault.Unlock(s.preferences.Dialogs().UnlockVault)
	return s.opened(err)
}

// opened ends the waiting unlock request, removes the items kept in the trash past its period and requests any due
// backup once err reports the vault open. A purge that fails is retried at the next unlock or listing of the trash.
func (s *Service) opened(err error) error {
	if err != nil {
		return present(err)
	}
	s.confirmations.VaultOpened()
	_, _ = s.vault.PurgeTrash()
	if s.backups != nil {
		s.backups.Poke()
	}
	return nil
}

// BeginRecovery stages recovery of the vault with phrase and previews its effect.
func (s *Service) BeginRecovery(phrase string) (RecoveryPreview, error) {
	preview, err := s.vault.BeginRecovery(phrase)
	if err != nil {
		return RecoveryPreview{}, present(err)
	}
	return RecoveryPreview{
		MayLoseNewerCredentials: preview.MayLoseNewerCredentials,
		KeyReplaced:             preview.KeyReplaced,
		NeedsWayIn:              preview.NeedsWayIn,
	}, nil
}

// ConfirmRecovery opens the staged vault once accepted confirms every warning its preview gave; a non-empty choice
// replaces this device's ways in.
func (s *Service) ConfirmRecovery(accepted bool, choice UnlockChoice) error {
	_, err := s.vault.ConfirmRecovery(accepted, choice.methods())
	return s.opened(err)
}

// ListCredentials reports every credential from the index.
func (s *Service) ListCredentials() ([]CredentialSummary, error) {
	entries, usage, err := s.listKind(vault.KindCredential)
	if err != nil {
		return nil, err
	}
	result := make([]CredentialSummary, len(entries))
	for i, entry := range entries {
		result[i] = CredentialSummary{
			ID:          entry.ID.String(),
			Label:       entry.Label,
			Login:       entry.Detail,
			Site:        entry.Site,
			Sites:       append([]string{}, entry.Sites...),
			Email:       entry.Email,
			Pinned:      entry.Pinned,
			LastUsedAt:  usage[entry.ID],
			Groups:      idStrings(entry.Groups),
			Tags:        append([]string{}, entry.Tags...),
			OneTimeCode: entry.Code != vault.CodeFace{},
			Passkeys:    len(entry.Passkeys),
		}
	}
	return result, nil
}

// listKind reports the items of one kind from the index alone, with when each was last used.
func (s *Service) listKind(kind vault.Kind) ([]vault.Entry, map[vault.ID]int64, error) {
	entries, err := s.vault.List()
	if err != nil {
		return nil, nil, present(err)
	}
	usage, err := s.vault.Usage()
	if err != nil {
		return nil, nil, present(err)
	}
	kept := make([]vault.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind == kind {
			kept = append(kept, entry)
		}
	}
	return kept, usage, nil
}

// readSelected reads one item through a fresh selection.
func readSelected[T any](s *Service, id vault.ID, read func(vault.Selection) (T, error)) (T, error) {
	var none T
	s.vault.ClearSelection()
	ticket, err := s.vault.Select(id)
	if err != nil {
		return none, present(err)
	}
	item, err := read(ticket)
	if err != nil {
		return none, present(err)
	}
	return item, nil
}

// openItem reads an opened item, records the use, and reports its groups from the index.
func openItem[T any](s *Service, id vault.ID, read func(vault.Selection) (T, error)) (T, []string, error) {
	var none T
	item, err := readSelected(s, id, read)
	if err != nil {
		return none, nil, err
	}
	if err := s.vault.MarkUsed(id); err != nil {
		return none, nil, present(err)
	}
	entries, err := s.vault.List()
	if err != nil {
		return none, nil, present(err)
	}
	for _, entry := range entries {
		if entry.ID == id {
			return item, idStrings(entry.Groups), nil
		}
	}
	return item, idStrings(nil), nil
}

// ReadCredential opens the credential id and records the use.
func (s *Service) ReadCredential(id string) (Credential, error) {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return Credential{}, fail(failureItemUnreadable)
	}
	item, groups, err := openItem(s, parsed, s.vault.ReadSelected)
	if err != nil {
		return Credential{}, err
	}
	return Credential{ID: item.ID.String(), Groups: groups, Site: item.Site(), Passkeys: passkeyViews(item.Passkeys), CredentialInput: fromVaultInput(item.CredentialInput, s.apps)}, nil
}

// copyableFields reads each copyable field; "totp" yields the current code, never the shared secret.
var copyableFields = map[string]func(vault.CredentialInput) (string, error){
	"login":    func(input vault.CredentialInput) (string, error) { return input.Login, nil },
	"email":    func(input vault.CredentialInput) (string, error) { return input.Email, nil },
	"password": func(input vault.CredentialInput) (string, error) { return input.Password, nil },
	"notes":    func(input vault.CredentialInput) (string, error) { return input.Notes, nil },
	"totp": func(input vault.CredentialInput) (string, error) {
		code, err := vault.GenerateOneTimeCode(input.TOTP, time.Now())
		return code.Code, err
	},
}

// credentialField resolves a copyableFields name or "website:<index>" to its reader; a website the credential lacks is
// not copyable, as another item's missing entry is not.
func credentialField(reference string) (func(vault.CredentialInput) (string, error), bool) {
	if read, known := copyableFields[reference]; known {
		return read, true
	}
	number, found := strings.CutPrefix(reference, "website:")
	if !found {
		return nil, false
	}
	index, err := strconv.Atoi(number)
	if err != nil || index < 0 || strconv.Itoa(index) != number {
		return nil, false
	}
	return func(input vault.CredentialInput) (string, error) {
		website, found := at(input.Websites, index)
		if !found {
			return "", fail(failureFieldNotCopyable)
		}
		return website, nil
	}, true
}

// CopyCredentialField copies one field of the credential id to the clipboard and records the use.
func (s *Service) CopyCredentialField(id string, field string) error {
	return s.copyCredentialField(id, field, clearAfter)
}

func clearAfter(after time.Duration, discard func()) {
	time.AfterFunc(after, discard)
}

func (s *Service) copyCredentialField(id string, field string, schedule func(time.Duration, func())) error {
	read, supported := credentialField(field)
	if !supported {
		return fail(failureFieldNotCopyable)
	}
	return s.copyItemValue(id, schedule, func(parsed vault.ID) (string, error) {
		item, err := readSelected(s, parsed, s.vault.ReadSelected)
		if err != nil {
			return "", err
		}
		value, err := read(item.CredentialInput)
		return value, present(err)
	})
}

// copyItemValue copies one value of an item and records the use; read returns interface-ready failures. An empty value
// copies nothing and records no use.
func (s *Service) copyItemValue(id string, schedule func(time.Duration, func()), read func(vault.ID) (string, error)) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	value, err := read(parsed)
	s.vault.ClearSelection()
	if err != nil {
		return err
	}
	if value == "" {
		return fail(failureFieldEmpty)
	}
	if err := s.vault.MarkUsed(parsed); err != nil {
		return present(err)
	}
	if !s.copyToClipboard(value, schedule) {
		return fail(failureCopyFailed)
	}
	return nil
}

// ClearSelection forgets the selected item.
func (s *Service) ClearSelection() error {
	s.vault.ClearSelection()
	return nil
}

// CreateCredential creates a credential in groups. Autofill links apps, so input names none.
func (s *Service) CreateCredential(input CredentialInput, groups []string) (string, error) {
	if len(input.Apps) > 0 {
		return "", fail(failureInvalidItem)
	}
	membership, err := s.knownGroups(groups)
	if err != nil {
		return "", err
	}
	id, err := s.vault.CreateCredential(input.toVault(), membership)
	if err != nil {
		return "", present(err)
	}
	return id.String(), nil
}

// UpdateCredential saves input and groups over the credential id and drops removedPasskeys in one save.
func (s *Service) UpdateCredential(id string, input CredentialInput, groups []string, removedPasskeys []string) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	removed, ok := passkeyCredentialIDs(removedPasskeys)
	if !ok {
		return fail(failureItemUnreadable)
	}
	held, err := s.heldApps(parsed)
	if err != nil {
		return err
	}
	patch, err := s.credentialPatch(input, groups, held)
	if err != nil {
		return err
	}
	patch.RemovePasskeys = removed
	return present(s.vault.EditCredential(parsed, patch))
}

// credentialPatch is the patch saving input and groups over a credential, whose linked apps must be among held.
func (s *Service) credentialPatch(input CredentialInput, groups []string, held []vault.App) (vault.CredentialPatch, error) {
	membership, err := s.knownGroups(groups)
	if err != nil {
		return vault.CredentialPatch{}, err
	}
	apps, err := keptApps(held, input.Apps)
	if err != nil {
		return vault.CredentialPatch{}, err
	}
	return vault.CredentialPatch{
		Label: &input.Label, Websites: &input.Websites, Login: &input.Login,
		Email: &input.Email, Password: &input.Password, Notes: &input.Notes,
		TOTP: &input.TOTP, Apps: &apps, Groups: &membership, Tags: &input.Tags,
	}, nil
}

// SetPinned pins or unpins an item of either kind.
func (s *Service) SetPinned(id string, pinned bool) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	return present(s.vault.SetPinned(parsed, pinned))
}

// DeleteItem deletes an item of any kind permanently, in the trash or not.
func (s *Service) DeleteItem(id string) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	return present(s.vault.DeleteItem(parsed))
}

// DuplicateItem saves a copy of an item of any kind, named after it, and returns the copy's id.
func (s *Service) DuplicateItem(id string) (string, error) {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return "", fail(failureItemUnreadable)
	}
	entries, err := s.vault.List()
	if err != nil {
		return "", present(err)
	}
	at := slices.IndexFunc(entries, func(entry vault.Entry) bool { return entry.ID == parsed })
	if at < 0 {
		return "", fail(failureItemUnreadable)
	}
	copied, err := s.vault.Duplicate(parsed, copyLabel(s.preferences.Catalog().Text("workspace.duplicate.name"), entries[at].Label))
	if err != nil {
		return "", present(err)
	}
	return copied.String(), nil
}

// copyLabel names a copy by pattern, its name cut so the label stays within vault.MaxLabelLength.
func copyLabel(pattern, name string) string {
	room := vault.MaxLabelLength - utf8.RuneCountInString(strings.ReplaceAll(pattern, "{name}", ""))
	if runes := []rune(name); len(runes) > room {
		name = strings.TrimSpace(string(runes[:max(room, 0)]))
	}
	return strings.ReplaceAll(pattern, "{name}", name)
}

// ExportEncryptedCopy saves an encrypted copy where the owner chooses and records it if the vault is unchanged.
func (s *Service) ExportEncryptedCopy() error {
	checked, _, err := s.vault.Export()
	clear(checked)
	if err != nil {
		return present(err)
	}
	dialogs := s.preferences.Dialogs()
	file := SavedFile{
		Prompt: dialogs.SaveExport, Name: dialogs.ExportFileName, MediaType: mediaVault,
		Filter: dialogs.VaultFilter, Pattern: "*" + vaultExtension,
	}
	var head vault.Head
	var digest [sha256.Size]byte
	saved, err := s.saveFile(file, func(target SaveTarget) error {
		data, exported, err := s.vault.Export()
		if err != nil {
			return err
		}
		defer clear(data)
		if err := target.Write(data); err != nil {
			return err
		}
		head, digest = exported, sha256.Sum256(data)
		return nil
	})
	if err != nil {
		return err
	}
	if !saved {
		return fail(failureExportCanceled)
	}
	verified, currentHead, err := s.vault.Export()
	clear(verified)
	if err != nil || currentHead != head {
		return fail(failureExportUnconfirmed)
	}
	if err := s.vault.RecordExport(head, digest); err != nil {
		return fail(failureExportUnrecorded)
	}
	return nil
}

// ExportStatus reports whether the last recorded encrypted copy is current, stale or unknown.
func (s *Service) ExportStatus() (ExportStatus, error) {
	state, err := s.vault.ExportState()
	if err != nil {
		return ExportStatus{}, present(err)
	}
	switch state {
	case vaultservice.ExportCurrent:
		return ExportStatus{State: "current"}, nil
	case vaultservice.ExportStale:
		return ExportStatus{State: "stale"}, nil
	default:
		return ExportStatus{State: "unknown"}, nil
	}
}

// OpenWebsite opens a credential's site in the browser; only http and https addresses are opened.
func (s *Service) OpenWebsite(address string) error {
	target, err := webAddress(address)
	if err != nil {
		return err
	}
	return s.openURL(target)
}

func webAddress(address string) (string, error) {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return "", fail(failureWebsiteMissing)
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return "", fail(failureWebsiteUnsupported)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fail(failureWebsiteUnsupported)
	}
	return parsed.String(), nil
}

// Lock closes the vault and clears what this app put on the clipboard.
func (s *Service) Lock() error {
	// A lock made in view skips the device unlock prompt until the app has left view.
	s.shows.locked()
	s.lock()
	s.clearClipboard()
	return nil
}

// lock closes the vault; the icon cache is sealed by the vault and must be released first.
func (s *Service) lock() {
	s.icons.Release()
	s.vault.Lock()
	s.dropOpenVault()
}

// lockedInside clears what the open vault left behind, its clipboard copy included, after the vault service locked it
// on its own; the icon cache it can no longer seal is dropped.
func (s *Service) lockedInside() {
	s.icons.Release()
	s.dropOpenVault()
	s.clearClipboard()
}

// dropOpenVault ends pending confirmations and drops the staged photo, scans and import, any link share, the location
// a move was to go to, and the breach answers kept for the vault's passwords.
func (s *Service) dropOpenVault() {
	s.confirmations.EndAll(vaultservice.ErrNotReady)
	s.photo.clear()
	s.scans.clear()
	s.imports.clear()
	s.links.Cancel()
	s.moveTo.drop()
	s.breaches.Forget()
}

func (input CredentialInput) toVault() vault.CredentialInput {
	return vault.CredentialInput{
		Label: input.Label, Websites: input.Websites, Login: input.Login,
		Email: input.Email, Password: input.Password, Notes: input.Notes,
		TOTP: input.TOTP, Tags: input.Tags,
	}
}

// fromVaultInput reports websites and apps as arrays, never null, with each app named as apps name it.
func fromVaultInput(input vault.CredentialInput, apps AppNames) CredentialInput {
	return CredentialInput{
		Label: input.Label, Websites: append([]string{}, input.Websites...), Login: input.Login,
		Email: input.Email, Password: input.Password, Notes: input.Notes,
		TOTP: input.TOTP, Apps: linkedApps(input.Apps, apps), Tags: append([]string{}, input.Tags...),
	}
}

func present(err error) error {
	if err == nil {
		return nil
	}
	var worded failure
	switch {
	case errors.As(err, &worded):
		return err
	case errors.Is(err, ownerauth.ErrCanceled):
		return fail(failureAuthenticationCanceled)
	case errors.Is(err, ownerauth.ErrFailed):
		return fail(failureAuthenticationFailed)
	case errors.Is(err, ownerauth.ErrUnavailable):
		return fail(failureUnlockUnavailable)
	case errors.Is(err, unlock.ErrUnbound):
		return fail(failureUnlockKeyMissing)
	case errors.Is(err, vaultservice.ErrWayInNotSet):
		return fail(failureWayInNotSet)
	case errors.Is(err, vaultservice.ErrKeyChangeUnfinished):
		return fail(failureKeyChangeUnfinished)
	case errors.Is(err, vaultservice.ErrKeyChangeUncertain):
		return fail(failureKeyChangeUncertain)
	case errors.Is(err, vaultservice.ErrConfirmationNeeded):
		return fail(failureRecoveryNeedsConfirm)
	case errors.Is(err, vaultservice.ErrReplacedKeyNeedsConfirmation):
		return fail(failureRecoveryReplacedKey)
	case errors.Is(err, vaultservice.ErrRecoveryChanged):
		return fail(failureRecoveryChanged)
	case errors.Is(err, vault.ErrInvalidPhrase):
		return fail(failureRecoveryPhraseInvalid)
	case errors.Is(err, vaultservice.ErrAlreadyInitialized):
		return fail(failureVaultExists)
	case errors.Is(err, vaultservice.ErrSetupInProgress):
		return fail(failureSetupInProgress)
	case errors.Is(err, vaultservice.ErrNoPendingSetup):
		return fail(failureSetupNotActive)
	case errors.Is(err, storage.ErrUnavailable):
		return fail(failureStorageUnavailable)
	case errors.Is(err, storage.ErrSelectionInvalid):
		return fail(failureStorageSelectionInvalid)
	case errors.Is(err, storage.ErrTargetOccupied):
		return fail(failureStorageOccupied)
	case errors.Is(err, storage.ErrSameLocation):
		return fail(failureStorageSameLocation)
	case errors.Is(err, storage.ErrUnsupportedKind):
		return fail(failureStorageKindUnsupported)
	case errors.Is(err, storage.ErrUnknownVault):
		return fail(failureVaultUnknown)
	case errors.Is(err, storage.ErrInvalidPath):
		return fail(failureStoragePathInvalid)
	case errors.Is(err, vaultservice.ErrMoveVerification):
		return fail(failureMoveUnverified)
	case errors.Is(err, vaultservice.ErrSecondCopy):
		return fail(failureSecondCopy)
	case errors.Is(err, vaultservice.ErrOlderCopy):
		return fail(failureOlderCopy)
	case errors.Is(err, vaultservice.ErrDiverged):
		return fail(failureVaultDiverged)
	case errors.Is(err, vaultservice.ErrNoOtherVault):
		return fail(failureNoOtherVault)
	case errors.Is(err, vault.ErrWitnessMissing):
		return fail(failureWitnessMissing)
	case errors.Is(err, vault.ErrKeyReplaced):
		return fail(failureVaultKeyReplaced)
	case errors.Is(err, storage.ErrVaultMissing):
		return fail(failureVaultMissing)
	case errors.Is(err, vaultservice.ErrStaleExport):
		return fail(failureExportStale)
	case errors.Is(err, storage.ErrStaleHead), errors.Is(err, vault.ErrWitnessMismatch):
		return fail(failureVaultChanged)
	case errors.Is(err, storage.ErrDurabilityUncertain), errors.Is(err, storage.ErrFinalizerFailed):
		return fail(failureSaveUnconfirmed)
	case errors.Is(err, vault.ErrAuthentication):
		return fail(failureAuthenticationInvalid)
	case errors.Is(err, vaultservice.ErrNotReady):
		return fail(failureVaultLocked)
	case errors.Is(err, storage.ErrNotFound):
		return fail(failureNoLocalVault)
	case errors.Is(err, savedfile.ErrInvalidSize):
		return fail(failureFileSizeInvalid)
	case errors.Is(err, savedfile.ErrUnverified):
		return fail(failureFileUnverified)
	case errors.Is(err, preferences.ErrUnsupportedBackupChoice):
		return fail(failureBackupChoiceUnsupported)
	case errors.Is(err, preferences.ErrBackupFolderMissing):
		return fail(failureBackupFolderMissing)
	case errors.Is(err, photos.ErrUnsupported):
		return fail(failurePhotoUnsupported)
	case errors.Is(err, photos.ErrTooLarge):
		return fail(failurePhotoTooLarge)
	case errors.Is(err, photos.ErrOutside):
		return fail(failureInvalidItem)
	case errors.Is(err, vault.ErrUnsupported):
		return fail(failureUnsupportedFormat)
	case errors.Is(err, vault.ErrMalformed):
		return fail(failureMalformedVault)
	case errors.Is(err, unlock.ErrWrongPIN):
		return fail(failurePINWrong)
	case errors.Is(err, unlock.ErrTooSoon):
		return fail(failurePINTooSoon)
	case errors.Is(err, unlock.ErrPINRemoved):
		return fail(failurePINRemoved)
	case errors.Is(err, unlock.ErrNoPIN):
		return fail(failurePINMissing)
	case errors.Is(err, unlock.ErrNotAvailable):
		return fail(failureBiometryUnavailable)
	case errors.Is(err, unlock.ErrDisabled):
		return fail(failureBiometryDisabled)
	case errors.Is(err, unlock.ErrNoMethodLeft):
		return fail(failureUnlockMethodRequired)
	case errors.Is(err, unlock.ErrMalformed), errors.Is(err, unlock.ErrUnsupported):
		return fail(failureUnlockRecordUnreadable)
	case errors.Is(err, vault.ErrInvalidPIN):
		return fail(failurePINInvalid)
	case errors.Is(err, vault.ErrInvalidTOTP):
		return fail(failureInvalidCode)
	case errors.Is(err, vault.ErrNoTOTP):
		return fail(failureNoCode)
	case errors.Is(err, vault.ErrScanLimit):
		return fail(failureScanLimit)
	case errors.Is(err, storage.ErrTooLarge):
		return fail(failureResourceLimit)
	case errors.Is(err, vault.ErrInvalidInput):
		return fail(failureInvalidItem)
	case errors.Is(err, vault.ErrNotFound):
		return fail(failureItemUnreadable)
	case errors.Is(err, vault.ErrResourceLimit):
		return fail(failureResourceLimit)
	case errors.Is(err, linkserver.ErrExpired):
		return fail(failureLinkExpired)
	case errors.Is(err, linkserver.ErrCanceled):
		return fail(failureLinkCanceled)
	case errors.Is(err, linkserver.ErrUnavailable):
		return fail(failureLinkUnavailable)
	case errors.Is(err, linkstore.ErrNotFound):
		return fail(failureExtensionNotFound)
	case errors.Is(err, linkstore.ErrInvalidName):
		return fail(failureExtensionNameInvalid)
	case errors.Is(err, confirmation.ErrEnded):
		return fail(failureConfirmationEnded)
	case errors.Is(err, verification.ErrDeclined):
		return fail(failureOwnerUnverified)
	case errors.Is(err, os.ErrPermission):
		return fail(failurePermissionDenied)
	default:
		return fail(failureGeneral)
	}
}
