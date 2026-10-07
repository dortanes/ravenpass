package api

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/dortanes/ravenpass/packages/app/messages"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/trash"
	"github.com/dortanes/ravenpass/packages/importers"
	"github.com/dortanes/ravenpass/packages/importers/aliasvault"
	"github.com/dortanes/ravenpass/packages/importers/bitwarden"
	"github.com/dortanes/ravenpass/packages/vault"
)

// importSource is a manager's reader, which decides by content, and the extensions the picker shows.
type importSource struct {
	open       func(io.ReaderAt, int64) (importers.File, error)
	extensions string
}

// importSources are the managers Ravenpass imports from, by the name the interface sends.
var importSources = map[string]importSource{
	"bitwarden":  {open: bitwarden.Open, extensions: "*.json;*.csv;*.zip"},
	"aliasvault": {open: aliasvault.Open, extensions: "*.avex;*.avux;*.csv"},
}

// kindNames names the vault's item kinds as the interface knows them.
var kindNames = map[vault.Kind]string{
	vault.KindCredential: "credential",
	vault.KindCard:       "card",
	vault.KindIdentity:   "identity",
	vault.KindNote:       "note",
	vault.KindSeed:       "seed",
}

// ImportChoice is the export the person chose; Chosen is false for a canceled picker.
type ImportChoice struct {
	Chosen bool   `json:"chosen"`
	Name   string `json:"name"`
	// Locked is true for a file that needs its password before it has a preview.
	Locked  bool          `json:"locked"`
	Preview ImportPreview `json:"preview"`
}

// ImportPreview is what an import adds and leaves behind, known before anything is written.
type ImportPreview struct {
	Format string       `json:"format"`
	Items  int          `json:"items"`
	Kinds  []ImportKind `json:"kinds"`
	// Groups counts folders with duplicates kept, GroupsWithoutDuplicates with them skipped.
	Groups                  ImportGroups `json:"groups"`
	GroupsWithoutDuplicates ImportGroups `json:"groupsWithoutDuplicates"`
	// CardIssuers are the issuer identification numbers of cards that name no network.
	CardIssuers []string     `json:"cardIssuers"`
	Skipped     []ImportSkip `json:"skipped"`
	Attachments int          `json:"attachments"`
	// Passkeys counts passkeys left behind.
	Passkeys int `json:"passkeys"`
	// ImportedPasskeys counts passkeys brought over with duplicates kept.
	ImportedPasskeys int `json:"importedPasskeys"`
}

// ImportKind is what an import adds of one kind, with duplicates kept.
type ImportKind struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
	// Duplicates counts items the vault already holds.
	Duplicates   int `json:"duplicates"`
	OneTimeCodes int `json:"oneTimeCodes"`
	// Converted counts items arriving from another kind of the source.
	Converted []ImportConversion `json:"converted"`
}

// ImportConversion counts items of source kind From that become another kind.
type ImportConversion struct {
	From  string `json:"from"`
	Count int    `json:"count"`
}

// ImportGroups is what the folders of the imported items come to.
type ImportGroups struct {
	New      int `json:"new"`
	Existing int `json:"existing"`
	// Dropped counts folders that cannot become groups.
	Dropped int `json:"dropped"`
}

// ImportSkip is an item the import cannot add.
type ImportSkip struct {
	Label string `json:"label"`
	// Origin is empty for a type the source reader does not know.
	Origin string `json:"origin"`
	Reason string `json:"reason"`
}

// ImportOptions are the person's import choices.
type ImportOptions struct {
	Groups         bool `json:"groups"`
	SkipDuplicates bool `json:"skipDuplicates"`
	// CardNetworks maps each previewed card issuer to a network name cards use.
	CardNetworks map[string]string `json:"cardNetworks"`
}

// ImportResult is what an import added.
type ImportResult struct {
	Added  int               `json:"added"`
	Kinds  []ImportKindTotal `json:"kinds"`
	Groups int               `json:"groups"`
	// Format tells whether the source file still holds the passwords unencrypted.
	Format string `json:"format"`
	// Located is false for a file picked as a temporary copy, which Ravenpass cannot trash or reveal.
	Located bool `json:"located"`
}

// ImportKindTotal counts the items an import added of one kind.
type ImportKindTotal struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// importStaging holds the chosen export in memory only, under mu, until it is imported or dropped.
type importStaging struct {
	mu     sync.Mutex
	path   string
	file   importers.File
	export *importers.Export
}

// read takes the export out of the staged file and closes it, so its text is held once.
func (s *importStaging) read(labels importers.Labels) (importers.Export, error) {
	export, err := s.file.Read(labels)
	if err != nil {
		return importers.Export{}, err
	}
	s.file.Close()
	s.file = nil
	s.export = &export
	return export, nil
}

// finish forgets the export once it is in the vault, keeping the path.
func (s *importStaging) finish() {
	s.export = nil
}

func (s *importStaging) drop() {
	if s.file != nil {
		s.file.Close()
	}
	s.path, s.file, s.export = "", nil, nil
}

func (s *importStaging) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drop()
}

// ChooseImportFile picks and previews an export of source; a protected file stays locked until UnlockImportFile.
func (s *Service) ChooseImportFile(source string) (ImportChoice, error) {
	chosen, known := importSources[source]
	if !known {
		return ImportChoice{}, fail(failureImportSourceUnknown)
	}
	dialogs := s.preferences.Dialogs()
	picked, err := s.dialog.pick(dialogs.SelectImport, dialogs.ImportFilter, chosen.extensions)
	if err != nil {
		return ImportChoice{}, err
	}
	if picked.path == "" {
		return ImportChoice{Preview: emptyPreview()}, nil
	}
	return s.stageImport(picked, chosen.open)
}

// stageImport reads the picked export into staging, keeping its path unless it was a temporary copy.
func (s *Service) stageImport(picked pickedFile, open func(io.ReaderAt, int64) (importers.File, error)) (ImportChoice, error) {
	defer picked.discard()
	s.imports.mu.Lock()
	defer s.imports.mu.Unlock()
	s.imports.drop()
	file, err := openExport(picked.path, open)
	if err != nil {
		return ImportChoice{}, presentImport(err)
	}
	s.imports.file = file
	if !picked.temporary {
		s.imports.path = picked.path
	}
	choice := ImportChoice{Chosen: true, Name: filepath.Base(picked.path), Locked: file.Locked(), Preview: emptyPreview()}
	if choice.Locked {
		return choice, nil
	}
	if choice.Preview, err = s.previewStaged(); err != nil {
		s.imports.drop()
		return ImportChoice{}, err
	}
	return choice, nil
}

// openExport opens a regular file read-only for the source's reader; the Stat before Open avoids blocking on a named pipe.
func openExport(path string, open func(io.ReaderAt, int64) (importers.File, error)) (importers.File, error) {
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return nil, importers.ErrUnrecognized
	}
	source, err := os.Open(path)
	if err != nil {
		return nil, importers.ErrUnrecognized
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, importers.ErrUnrecognized
	}
	return open(source, info.Size())
}

// UnlockImportFile opens the staged protected export and previews it; a wrong password keeps it staged.
func (s *Service) UnlockImportFile(password string) (ImportPreview, error) {
	s.imports.mu.Lock()
	defer s.imports.mu.Unlock()
	if s.imports.file == nil {
		return ImportPreview{}, fail(failureImportNotActive)
	}
	if err := s.imports.file.Unlock(password); err != nil {
		if !errors.Is(err, importers.ErrWrongPassword) {
			s.imports.drop()
		}
		return ImportPreview{}, presentImport(err)
	}
	preview, err := s.previewStaged()
	if err != nil {
		s.imports.drop()
		return ImportPreview{}, err
	}
	return preview, nil
}

// previewStaged reads the staged file and plans it. The caller holds the staging mutex.
func (s *Service) previewStaged() (ImportPreview, error) {
	export, err := s.imports.read(s.importLabels())
	if err != nil {
		return ImportPreview{}, presentImport(err)
	}
	plan, err := s.planImport(export)
	if err != nil {
		return ImportPreview{}, err
	}
	return importPreview(plan.Preview()), nil
}

func (s *Service) importLabels() importers.Labels {
	return importLabelsIn(s.preferences.Catalog())
}

func importLabelsIn(catalog messages.Catalog) importers.Labels {
	label := func(name string) string { return catalog.Text("settings.import.note." + name) }
	return importers.Labels{
		Website: label("website"), OneTimeCode: label("one-time-code"), Title: label("title"),
		Company: label("company"), Username: label("username"), Expiry: label("expiry"),
		SecurityCode: label("security-code"), CardNumber: label("card-number"), CardholderName: label("cardholder"),
		Brand: label("brand"), PrivateKey: label("private-key"), PublicKey: label("public-key"),
		Fingerprint: label("fingerprint"), BankName: label("bank"), AccountHolder: label("account-holder"),
		AccountType: label("account-type"), AccountNumber: label("account-number"), RoutingNumber: label("routing-number"),
		BranchNumber: label("branch-number"), PIN: label("pin"), SWIFT: label("swift"), IBAN: label("iban"),
		BankPhone: label("bank-phone"), LicenseClass: label("license-class"), Sex: label("sex"),
		BirthPlace: label("birth-place"), Nationality: label("nationality"), PassportType: label("passport-type"),
		NationalID: label("national-id"), IssuedOn: label("issued-on"), ExpiresOn: label("expires-on"),
		Birthday: label("birthday"), Issuer: label("issuer"), Name: label("name"), Gender: label("gender"),
		Nickname: label("nickname"),
	}
}

// planImport lays an export against the vault as it is now.
func (s *Service) planImport(export importers.Export) (*importers.Plan, error) {
	entries, err := s.vault.List()
	if err != nil {
		return nil, present(err)
	}
	groups, err := s.vault.Groups()
	if err != nil {
		return nil, present(err)
	}
	return importers.NewPlan(export, entries, groups), nil
}

// ImportItems replans the staged export against the current vault and adds what options keep in one all-or-nothing save.
func (s *Service) ImportItems(options ImportOptions) (ImportResult, error) {
	s.imports.mu.Lock()
	defer s.imports.mu.Unlock()
	if s.imports.export == nil {
		return ImportResult{}, fail(failureImportNotActive)
	}
	networks, err := importNetworks(options.CardNetworks)
	if err != nil {
		return ImportResult{}, err
	}
	export := *s.imports.export
	plan, err := s.planImport(export)
	if err != nil {
		return ImportResult{}, err
	}
	items := plan.Items(importers.Options{Groups: options.Groups, SkipDuplicates: options.SkipDuplicates, CardNetworks: networks})
	if len(items) == 0 {
		return ImportResult{}, fail(failureImportEmpty)
	}
	added, err := s.vault.AddItems(items)
	if err != nil {
		// A save that fails past its refusals locks the vault, and the export goes with it.
		if !s.vault.Unlocked() {
			s.imports.drop()
		}
		return ImportResult{}, presentImportSave(err)
	}
	s.imports.finish()
	return ImportResult{
		Added: len(added.Items), Kinds: kindTotals(items), Groups: added.Groups,
		Format: string(export.Format), Located: s.imports.path != "",
	}, nil
}

// importNetworks parses network names by issuer; an unknown name refuses the import.
func importNetworks(names map[string]string) (map[string]vault.CardNetwork, error) {
	networks := make(map[string]vault.CardNetwork, len(names))
	for issuer, name := range names {
		network, known := vault.ParseCardNetwork(name)
		if !known {
			return nil, fail(failureInvalidItem)
		}
		networks[issuer] = network
	}
	return networks, nil
}

// CancelImport drops the staged export and forgets the file.
func (s *Service) CancelImport() error {
	s.imports.clear()
	return nil
}

// TrashImportFile moves the imported export to the Trash.
func (s *Service) TrashImportFile() error {
	s.imports.mu.Lock()
	defer s.imports.mu.Unlock()
	if s.imports.path == "" || s.imports.export != nil || s.imports.file != nil {
		return fail(failureImportNotActive)
	}
	if err := trash.MoveToTrash(s.imports.path); err != nil {
		return fail(failureImportTrashFailed)
	}
	s.imports.drop()
	return nil
}

// RevealImportFile shows the imported export in the file manager.
func (s *Service) RevealImportFile() error {
	s.imports.mu.Lock()
	path := s.imports.path
	s.imports.mu.Unlock()
	if path == "" {
		return fail(failureImportNotActive)
	}
	app := s.currentApp()
	if app == nil {
		return fail(failureWindowUnavailable)
	}
	if err := app.Env.OpenFileManager(path, true); err != nil {
		return fail(failureImportRevealFailed)
	}
	return nil
}

// presentImport names failures of the export itself and presents the rest as present does.
func presentImport(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, importers.ErrUnrecognized), errors.Is(err, importers.ErrLocked):
		return fail(failureImportUnrecognized)
	case errors.Is(err, importers.ErrTooLarge):
		return fail(failureImportTooLarge)
	case errors.Is(err, importers.ErrAccountBound):
		return fail(failureImportAccountBound)
	case errors.Is(err, importers.ErrWrongPassword):
		return fail(failureImportPasswordWrong)
	case errors.Is(err, importers.ErrUnsupportedEncryption):
		return fail(failureImportEncryption)
	default:
		return present(err)
	}
}

// presentImportSave reports an item, group or file-size limit as the one import-limit failure.
func presentImportSave(err error) error {
	if errors.Is(err, vault.ErrResourceLimit) || errors.Is(err, storage.ErrTooLarge) {
		return fail(failureImportLimit)
	}
	return present(err)
}

func emptyPreview() ImportPreview {
	return ImportPreview{Kinds: []ImportKind{}, CardIssuers: []string{}, Skipped: []ImportSkip{}}
}

func importPreview(preview importers.Preview) ImportPreview {
	result := emptyPreview()
	result.Format = string(preview.Format)
	result.Items = preview.Items
	result.Attachments = preview.Attachments
	result.Passkeys = preview.Passkeys
	result.ImportedPasskeys = preview.ImportedPasskeys
	result.Groups = importGroups(preview.Groups)
	result.GroupsWithoutDuplicates = importGroups(preview.GroupsWithoutDuplicates)
	result.CardIssuers = append(result.CardIssuers, preview.CardIssuers...)
	for _, kind := range preview.Kinds {
		converted := make([]ImportConversion, len(kind.Converted))
		for i, conversion := range kind.Converted {
			converted[i] = ImportConversion{From: string(conversion.From), Count: conversion.Count}
		}
		result.Kinds = append(result.Kinds, ImportKind{
			Kind: kindNames[kind.Kind], Count: kind.Count, Duplicates: kind.Duplicates,
			OneTimeCodes: kind.OneTimeCodes, Converted: converted,
		})
	}
	for _, skip := range preview.Skipped {
		result.Skipped = append(result.Skipped, ImportSkip{Label: skip.Label, Origin: string(skip.Origin), Reason: string(skip.Reason)})
	}
	return result
}

func importGroups(groups importers.GroupPreview) ImportGroups {
	return ImportGroups{New: groups.New, Existing: groups.Existing, Dropped: groups.Dropped}
}

// kindTotals counts the items added of each kind, in the order the preview lists kinds.
func kindTotals(items []vault.NewItem) []ImportKindTotal {
	counts := make(map[vault.Kind]int, len(kindNames))
	for _, item := range items {
		counts[item.Kind()]++
	}
	totals := []ImportKindTotal{}
	for _, kind := range []vault.Kind{vault.KindCredential, vault.KindCard, vault.KindIdentity, vault.KindNote, vault.KindSeed} {
		if counts[kind] > 0 {
			totals = append(totals, ImportKindTotal{Kind: kindNames[kind], Count: counts[kind]})
		}
	}
	return totals
}
