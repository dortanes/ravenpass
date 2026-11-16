package api

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dortanes/ravenpass/packages/app/preferences"
	"github.com/dortanes/ravenpass/packages/app/savedfile"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const vaultExtension = ".rpv"

// VaultFiles is how the owner picks vault files on a host; Create and Existing report false on cancel.
type VaultFiles interface {
	// Kinds lists the storage kinds whose file the owner picks; others go where the host keeps them.
	Kinds() []storage.Kind
	// Create asks where a new vault file of kind goes, proposing name. For a local file the owner picks a folder,
	// and name there may be taken.
	Create(kind storage.Kind, name string) (storage.Target, bool, error)
	// Existing asks for a vault file that already exists.
	Existing() (storage.Target, bool, error)
}

// Sandbox keeps what the owner picks reachable on a host whose system grants access to each picked place alone.
type Sandbox interface {
	// Keep keeps access to a folder the owner just picked across launches.
	Keep(folder string) error
	// Replace writes data over a file the owner picked in a save panel, as savedfile.Replace does.
	Replace(path string, data []byte, limit int64) error
}

// unconfined reaches every file by its path.
type unconfined struct{}

func (unconfined) Keep(string) error { return nil }

func (unconfined) Replace(path string, data []byte, limit int64) error {
	return savedfile.Replace(path, data, limit)
}

// dialogFiles picks local vault files with the desktop's file dialogs. A vault goes in a folder the owner picks, since
// its lock and interrupted-write files sit beside it.
type dialogFiles struct {
	currentApp func() *application.App
	dialogs    func() preferences.Dialogs
	sandbox    Sandbox
}

func (dialogFiles) Kinds() []storage.Kind { return []storage.Kind{storage.LocalFile} }

func (files dialogFiles) Create(kind storage.Kind, name string) (storage.Target, bool, error) {
	folder, chosen, err := chooseFolder(files.currentApp, files.sandbox, files.dialogs().ChooseVaultLocation)
	if err != nil || !chosen {
		return storage.Target{}, false, err
	}
	return storage.Target{Kind: kind, Path: filepath.Join(folder, withVaultExtension(name))}, true, nil
}

// chooseFolder asks for a folder with the desktop's open dialog and keeps access to it.
func chooseFolder(currentApp func() *application.App, sandbox Sandbox, prompt string) (string, bool, error) {
	app := currentApp()
	if app == nil {
		return "", false, fail(failureWindowUnavailable)
	}
	folder, err := app.Dialog.OpenFile().
		SetTitle(prompt).
		SetMessage(prompt).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		PromptForSingleSelection()
	if err != nil {
		return "", false, fail(failureLocationNotSelected)
	}
	if folder == "" {
		return "", false, nil
	}
	if err := sandbox.Keep(folder); err != nil {
		return "", false, fail(failurePermissionDenied)
	}
	return folder, true, nil
}

// Existing opens the one vault in the folder the owner picks, else asks which file there.
func (files dialogFiles) Existing() (storage.Target, bool, error) {
	dialogs := files.dialogs()
	folder, chosen, err := chooseFolder(files.currentApp, files.sandbox, dialogs.ChooseVaultFolder)
	if err != nil || !chosen {
		return storage.Target{}, false, err
	}
	if vaults := vaultFilesIn(folder); len(vaults) == 1 {
		return storage.Target{Kind: storage.LocalFile, Path: vaults[0]}, true, nil
	}
	app := files.currentApp()
	if app == nil {
		return storage.Target{}, false, fail(failureWindowUnavailable)
	}
	path, err := app.Dialog.OpenFile().
		SetTitle(dialogs.SelectVaultFile).
		SetMessage(dialogs.SelectVaultFile).
		SetDirectory(folder).
		AddFilter(dialogs.VaultFilter, "*"+vaultExtension).
		PromptForSingleSelection()
	if err != nil {
		return storage.Target{}, false, fail(failureFileNotSelected)
	}
	if path == "" {
		return storage.Target{}, false, nil
	}
	// A file outside the picked folder needs its own folder kept, which a sandbox refuses.
	if filepath.Dir(path) != folder {
		if err := files.sandbox.Keep(filepath.Dir(path)); err != nil {
			return storage.Target{}, false, fail(failurePermissionDenied)
		}
	}
	return storage.Target{Kind: storage.LocalFile, Path: path}, true, nil
}

// vaultFilesIn lists the vault files directly in folder; the lock and interrupted-write files start with a dot.
func vaultFilesIn(folder string) []string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil
	}
	var vaults []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.Type().IsRegular() && !strings.HasPrefix(name, ".") && strings.EqualFold(filepath.Ext(name), vaultExtension) {
			vaults = append(vaults, filepath.Join(folder, name))
		}
	}
	return vaults
}

// withVaultExtension appends the vault extension when the save dialog returns a name without it.
func withVaultExtension(path string) string {
	if strings.EqualFold(filepath.Ext(path), vaultExtension) {
		return path
	}
	return path + vaultExtension
}
