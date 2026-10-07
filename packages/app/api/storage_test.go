package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/localfile"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
)

func newServiceWithStorage(t *testing.T, selectionPath string, vaultPath string) (*Service, *storage.Manager) {
	t.Helper()
	files, err := storage.NewManager(
		selectionPath,
		storage.Target{Kind: storage.LocalFile, Path: vaultPath},
		MaxVaultBytes,
		localfile.Backend{Home: filepath.Dir(vaultPath)},
	)
	if err != nil {
		t.Fatal(err)
	}
	core, err := vaultservice.New(files, newStubKeys(), newStubDevice().bindings())
	if err != nil {
		t.Fatal(err)
	}
	return newTestService(t, core, silentSites{}, filepath.Join(t.TempDir(), "icons")), files
}

func TestStorageStatusReportsTheBoundLocation(t *testing.T) {
	home := t.TempDir()
	vaultPath := filepath.Join(home, localfile.DefaultVaultName)
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), vaultPath)

	status, err := service.GetStorage()
	if err != nil {
		t.Fatal(err)
	}
	if status.Available || status.Path != vaultPath {
		t.Fatalf("status before opening = %+v", status)
	}
	state, err := service.GetState()
	if err != nil || state.Phase != "storage" {
		t.Fatalf("state before opening = %+v, error = %v", state, err)
	}

	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	status, err = service.GetStorage()
	if err != nil {
		t.Fatal(err)
	}
	if !status.Available || status.Kind != string(storage.LocalFile) || status.Path != vaultPath {
		t.Fatalf("status after opening = %+v", status)
	}
	if !status.Restricted || status.Reason != "" || len(status.Kinds) != 1 {
		t.Fatalf("status details = %+v", status)
	}
	state, err = service.GetState()
	if err != nil || state.Phase != "setup" {
		t.Fatalf("state after opening = %+v, error = %v", state, err)
	}
}

func TestAVaultFileThatCannotBeReadShowsTheStorageScreen(t *testing.T) {
	home := t.TempDir()
	vaultPath := filepath.Join(home, localfile.DefaultVaultName)
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), vaultPath)
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmCreation(phrase, UnlockChoice{Biometry: true}); err != nil {
		t.Fatal(err)
	}
	if err := service.Lock(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(vaultPath); err != nil {
		t.Fatal(err)
	}
	// A directory in the file's place opens and fails on read, as a document its provider will not deliver does.
	if err := os.Mkdir(vaultPath, 0o700); err != nil {
		t.Fatal(err)
	}

	if state, err := service.GetState(); err != nil || state.Phase != "storage" {
		t.Fatalf("state with an unreadable file = %+v, error = %v", state, err)
	}
	if err := os.Remove(vaultPath); err != nil {
		t.Fatal(err)
	}
	if state, err := service.GetState(); err != nil || state.Phase != "locked" {
		t.Fatalf("state once the file reads as gone = %+v, error = %v", state, err)
	}
}

func TestAVaultFileThatWentKeepsTheLockedScreen(t *testing.T) {
	home := t.TempDir()
	vaultPath := filepath.Join(home, localfile.DefaultVaultName)
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), vaultPath)
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmCreation(phrase, UnlockChoice{Biometry: true}); err != nil {
		t.Fatal(err)
	}
	if err := service.Lock(); err != nil {
		t.Fatal(err)
	}
	if status, err := service.GetStorage(); err != nil || status.Missing {
		t.Fatalf("storage with the file in place = %+v, error = %v", status, err)
	}
	if err := os.Remove(vaultPath); err != nil {
		t.Fatal(err)
	}

	if state, err := service.GetState(); err != nil || state.Phase != "locked" {
		t.Fatalf("state without the file = %+v, error = %v", state, err)
	}
	status, err := service.GetStorage()
	if err != nil || !status.Missing || status.Path != vaultPath {
		t.Fatalf("storage without the file = %+v, error = %v", status, err)
	}
}

func TestOnlyTheHostsOwnFolderIsPrivate(t *testing.T) {
	home := t.TempDir()
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), filepath.Join(home, localfile.DefaultVaultName))
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	for _, place := range []struct {
		target storage.Target
		shared bool
	}{
		{storage.Target{Kind: storage.LocalFile, Path: filepath.Join(home, "vault 2.rpv")}, false},
		{storage.Target{Kind: storage.LocalFile, Path: filepath.Join(t.TempDir(), "Drive.rpv")}, true},
	} {
		if err := files.Bind(place.target); err != nil {
			t.Fatal(err)
		}
		status, err := service.GetStorage()
		if err != nil || status.Shared != place.shared {
			t.Fatalf("storage at %s = %+v, error = %v", place.target.Path, status, err)
		}
	}
	if !shared(storage.Target{Kind: storage.Document, Path: "content://drive/document/1"}, storage.Target{Kind: storage.LocalFile, Path: filepath.Join(home, "vault.rpv")}) {
		t.Fatal("a document was reported private")
	}
}

func TestRetryStorageOpensAReachableLocation(t *testing.T) {
	home := t.TempDir()
	blocked := filepath.Join(home, "disk")
	if err := os.WriteFile(blocked, []byte("not a folder"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(blocked, localfile.DefaultVaultName)
	service, _ := newServiceWithStorage(t, filepath.Join(home, "storage.json"), missing)
	if err := service.RetryStorage(); err == nil {
		t.Fatal("an unreachable location opened")
	}
	status, err := service.GetStorage()
	if err != nil {
		t.Fatal(err)
	}
	if status.Available || status.Reason != string(storage.ReasonUnreachable) {
		t.Fatalf("status after a failed retry = %+v", status)
	}
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	if err := service.RetryStorage(); err != nil {
		t.Fatal(err)
	}
	if status, err = service.GetStorage(); err != nil || !status.Available {
		t.Fatalf("status after a successful retry = %+v, error = %v", status, err)
	}
}

func TestStorageActionsNeedAWindow(t *testing.T) {
	home := t.TempDir()
	service, _ := newServiceWithStorage(t, filepath.Join(home, "storage.json"), filepath.Join(home, localfile.DefaultVaultName))
	if _, err := service.SelectStorageLocation("local-file"); err == nil {
		t.Error("a location was selected without a window")
	}
	if _, err := service.ChooseStorageMove("local-file"); err == nil {
		t.Error("a move was chosen without a window")
	}
	if _, err := service.OpenVault(context.Background()); err == nil {
		t.Error("a vault file was opened without a window")
	}
}

func TestAnotherVaultStartsBesideTheCurrentOneAndCanBeCancelled(t *testing.T) {
	home := t.TempDir()
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), filepath.Join(home, localfile.DefaultVaultName))
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	if err := service.CancelVaultCreation(); err == nil || err.Error() != failurePrefix+string(failureNoOtherVault) {
		t.Fatalf("cancelling the first vault's setup reported %v", err)
	}
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmCreation(phrase, UnlockChoice{Biometry: true}); err != nil {
		t.Fatal(err)
	}

	change, err := service.CreateVault(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "vault 2.rpv"); !change.Changed || change.Path != want {
		t.Fatalf("another vault starts at %+v, want %s", change, want)
	}
	if state, err := service.GetState(); err != nil || state.Phase != "setup" {
		t.Fatalf("state while setting up another vault = %+v, error = %v", state, err)
	}

	if err := service.CancelVaultCreation(); err != nil {
		t.Fatal(err)
	}
	if state, err := service.GetState(); err != nil || state.Phase != "locked" {
		t.Fatalf("state after cancelling = %+v, error = %v", state, err)
	}
	status, err := service.GetStorage()
	if err != nil || status.Path != filepath.Join(home, localfile.DefaultVaultName) || len(status.Vaults) != 1 {
		t.Fatalf("storage after cancelling = %+v, error = %v", status, err)
	}
}

// pickedFiles answers with the files it holds or cancels without one, counting existing-file picks.
type pickedFiles struct {
	kinds    []storage.Kind
	created  storage.Target
	existing storage.Target
	names    []string
	shown    int
}

func (files *pickedFiles) Kinds() []storage.Kind { return files.kinds }

func (files *pickedFiles) Create(kind storage.Kind, name string) (storage.Target, bool, error) {
	files.names = append(files.names, name)
	return files.created, files.created.Kind == kind, nil
}

func (files *pickedFiles) Existing() (storage.Target, bool, error) {
	files.shown++
	return files.existing, files.existing != storage.Target{}, nil
}

func TestPickedLocationsKeepTheirProvidersName(t *testing.T) {
	home := t.TempDir()
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), filepath.Join(home, localfile.DefaultVaultName))
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	picked := storage.Target{
		Kind:  storage.LocalFile,
		Path:  filepath.Join(t.TempDir(), "Personal.rpv"),
		Label: storage.Label{Name: "Personal.rpv", Place: "Drive"},
	}
	picker := &pickedFiles{kinds: []storage.Kind{storage.LocalFile}, created: picked}
	service.files = picker

	change, err := service.SelectStorageLocation(string(storage.LocalFile))
	if err != nil || !change.Changed || change.Path != picked.Path {
		t.Fatalf("selecting a picked location = %+v, %v", change, err)
	}
	if len(picker.names) != 1 || picker.names[0] != localfile.DefaultVaultName {
		t.Fatalf("names proposed to the picker = %v", picker.names)
	}
	status, err := service.GetStorage()
	if err != nil || status.Name != "Personal.rpv" || status.Place != "Drive" || len(status.Chosen) != 1 {
		t.Fatalf("storage at a picked location = %+v, %v", status, err)
	}
	if len(status.Vaults) != 1 || status.Vaults[0] != (Vault{Kind: "local-file", Path: picked.Path, Name: "Personal.rpv", Place: "Drive", Current: true}) {
		t.Fatalf("vaults at a picked location = %+v", status.Vaults)
	}

	picker.created = storage.Target{}
	if change, err := service.SelectStorageLocation(string(storage.LocalFile)); err != nil || change.Changed {
		t.Fatalf("a cancelled picker = %+v, %v", change, err)
	}
	if change, err := service.OpenVault(context.Background()); err != nil || change.Changed {
		t.Fatalf("a cancelled open = %+v, %v", change, err)
	}
	if _, err := service.SelectStorageLocation(string(storage.Document)); err == nil {
		t.Fatal("a kind this device does not offer was selected")
	}
}

func TestAMoveGoesOnceToTheLocationChosenBeforeIt(t *testing.T) {
	home := t.TempDir()
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), filepath.Join(home, localfile.DefaultVaultName))
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmCreation(phrase, UnlockChoice{Biometry: true}); err != nil {
		t.Fatal(err)
	}
	notChosen := failurePrefix + string(failureLocationNotSelected)
	if _, err := service.MoveStorageLocation(); err == nil || err.Error() != notChosen {
		t.Fatalf("a move to no chosen location reported %v", err)
	}

	picked := storage.Target{Kind: storage.LocalFile, Path: filepath.Join(t.TempDir(), "Moved.rpv")}
	picker := &pickedFiles{kinds: []storage.Kind{storage.LocalFile}}
	service.files = picker
	if chosen, err := service.ChooseStorageMove(string(storage.LocalFile)); err != nil || chosen {
		t.Fatalf("a cancelled picker chose %v, %v", chosen, err)
	}
	if _, err := service.MoveStorageLocation(); err == nil || err.Error() != notChosen {
		t.Fatalf("a move after a cancelled picker reported %v", err)
	}
	picker.created = picked
	if chosen, err := service.ChooseStorageMove(string(storage.LocalFile)); err != nil || !chosen {
		t.Fatalf("choosing a move = %v, %v", chosen, err)
	}
	change, err := service.MoveStorageLocation()
	if err != nil || !change.Changed || change.Path != picked.Path {
		t.Fatalf("moving to the chosen location = %+v, %v", change, err)
	}
	if _, err := service.MoveStorageLocation(); err == nil || err.Error() != notChosen {
		t.Fatalf("a second move to the same choice reported %v", err)
	}

	if chosen, err := service.ChooseStorageMove(string(storage.LocalFile)); err != nil || !chosen {
		t.Fatalf("choosing another move = %v, %v", chosen, err)
	}
	picker.created = storage.Target{}
	if chosen, err := service.ChooseStorageMove(string(storage.LocalFile)); err != nil || chosen {
		t.Fatalf("a cancelled picker chose %v, %v", chosen, err)
	}
	if _, err := service.MoveStorageLocation(); err == nil || err.Error() != notChosen {
		t.Fatalf("a move after a cancelled second pick reported %v", err)
	}

	picker.created = storage.Target{Kind: storage.LocalFile, Path: filepath.Join(t.TempDir(), "Later.rpv")}
	if chosen, err := service.ChooseStorageMove(string(storage.LocalFile)); err != nil || !chosen {
		t.Fatalf("choosing a move before locking = %v, %v", chosen, err)
	}
	if err := service.Lock(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.MoveStorageLocation(); err == nil || err.Error() != notChosen {
		t.Fatalf("a move chosen before the lock reported %v", err)
	}
}

func TestAKindTheOwnerDoesNotPickGoesWhereTheHostKeepsVaults(t *testing.T) {
	home := t.TempDir()
	defaultPath := filepath.Join(home, localfile.DefaultVaultName)
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), defaultPath)
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	service.files = &pickedFiles{}
	if err := files.Bind(storage.Target{Kind: storage.LocalFile, Path: filepath.Join(t.TempDir(), "elsewhere.rpv")}); err != nil {
		t.Fatal(err)
	}

	change, err := service.SelectStorageLocation(string(storage.LocalFile))
	if err != nil || !change.Changed || change.Path != defaultPath {
		t.Fatalf("selecting the host's own location = %+v, %v", change, err)
	}
	status, err := service.GetStorage()
	if err != nil || len(status.Chosen) != 0 || status.Name != localfile.DefaultVaultName || status.Place != filepath.Base(home) {
		t.Fatalf("storage at the host's own location = %+v, %v", status, err)
	}
}

func TestVaultExtensionIsKept(t *testing.T) {
	for given, want := range map[string]string{
		"/vaults/passwords":     "/vaults/passwords.rpv",
		"/vaults/passwords.rpv": "/vaults/passwords.rpv",
		"/vaults/passwords.RPV": "/vaults/passwords.RPV",
		"/vaults/passwords.txt": "/vaults/passwords.txt.rpv",
	} {
		if got := withVaultExtension(given); got != want {
			t.Errorf("withVaultExtension(%q) = %q, want %q", given, got, want)
		}
	}
}
