package api

import (
	"path/filepath"
	"slices"
	"sync"

	"github.com/dortanes/ravenpass/packages/app/storage"
)

// StorageStatus describes where the current vault is kept and which storage kinds are offered.
type StorageStatus struct {
	Kinds []string `json:"kinds"`
	// Chosen lists the kinds whose file the owner picks; the others live where the host keeps them.
	Chosen     []string `json:"chosen"`
	Kind       string   `json:"kind"`
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	Place      string   `json:"place"`
	Vaults     []Vault  `json:"vaults"`
	Available  bool     `json:"available"`
	Restricted bool     `json:"restricted"`
	Reason     string   `json:"reason"`
	// Shared reports a file other devices may open: anywhere but the folder the host keeps its own vaults in.
	Shared bool `json:"shared"`
	// Missing reports that the file this device opened the vault from is gone from the current location.
	Missing bool `json:"missing"`
}

// Vault is one of the vaults this device knows.
type Vault struct {
	Kind string `json:"kind"`
	// Path identifies the vault.
	Path string `json:"path"`
	// Name is the file's name.
	Name string `json:"name"`
	// Place is where the file is kept, in words.
	Place   string `json:"place"`
	Current bool   `json:"current"`
}

// StorageChange reports what a location action did; Changed is false after a cancel or declined authentication.
type StorageChange struct {
	Changed         bool   `json:"changed"`
	Path            string `json:"path"`
	Previous        Vault  `json:"previous"`
	PreviousRemoved bool   `json:"previousRemoved"`
}

// GetStorage reports where the current vault is kept and the vaults this device knows.
func (s *Service) GetStorage() (StorageStatus, error) {
	status := s.vault.Storage()
	kinds := make([]string, len(status.Kinds))
	for i, kind := range status.Kinds {
		kinds[i] = string(kind)
	}
	chosen := []string{}
	for _, kind := range s.files.Kinds() {
		if slices.Contains(status.Kinds, kind) {
			chosen = append(chosen, string(kind))
		}
	}
	vaults := make([]Vault, len(status.Vaults))
	for i, target := range status.Vaults {
		vaults[i] = s.vaultAt(target, target.Same(status.Current))
	}
	current := s.label(status.Current)
	missing, err := s.vault.VaultMissing()
	if err != nil {
		return StorageStatus{}, present(err)
	}
	return StorageStatus{
		Kinds:      kinds,
		Chosen:     chosen,
		Kind:       string(status.Current.Kind),
		Path:       status.Current.Path,
		Name:       current.Name,
		Place:      current.Place,
		Vaults:     vaults,
		Available:  status.Available,
		Restricted: status.Restricted,
		Reason:     string(status.Reason),
		Shared:     shared(status.Current, status.Default),
		Missing:    missing,
	}, nil
}

// shared reports whether target lies outside the folder of home, where the host keeps vaults no other device reads.
func shared(target, home storage.Target) bool {
	return target.Kind != home.Kind || filepath.Dir(target.Path) != filepath.Dir(home.Path)
}

// SelectStorageLocation chooses where a vault not yet created will be kept, in a storage of kind.
func (s *Service) SelectStorageLocation(kind string) (StorageChange, error) {
	target, chosen, err := s.newVaultLocation(storage.Kind(kind))
	if err != nil || !chosen {
		return StorageChange{}, present(err)
	}
	if err := s.vault.BindStorage(target); err != nil {
		return StorageChange{}, present(err)
	}
	return StorageChange{Changed: true, Path: target.Path}, nil
}

// chosenMove holds the location ChooseStorageMove took, until MoveStorageLocation moves the vault there.
type chosenMove struct {
	mu     sync.Mutex
	target *storage.Target
}

func (m *chosenMove) hold(target storage.Target) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.target = &target
}

func (m *chosenMove) drop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.target = nil
}

func (m *chosenMove) take() (storage.Target, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.target
	m.target = nil
	if target == nil {
		return storage.Target{}, false
	}
	return *target, true
}

// ChooseStorageMove asks where an open vault goes, in a storage of kind, and holds the answer for
// MoveStorageLocation; false where the owner chose nothing, which holds no location.
func (s *Service) ChooseStorageMove(kind string) (bool, error) {
	s.moveTo.drop()
	target, chosen, err := s.newVaultLocation(storage.Kind(kind))
	if err != nil || !chosen {
		return false, present(err)
	}
	s.moveTo.hold(target)
	return true, nil
}

// MoveStorageLocation moves an open vault to the location ChooseStorageMove holds.
func (s *Service) MoveStorageLocation() (StorageChange, error) {
	target, chosen := s.moveTo.take()
	if !chosen {
		return StorageChange{}, fail(failureLocationNotSelected)
	}
	relocation, err := s.vault.MoveStorage(target)
	if err != nil {
		return StorageChange{}, present(err)
	}
	return StorageChange{
		Changed:         true,
		Path:            relocation.Target.Path,
		Previous:        s.vaultAt(relocation.Previous, false),
		PreviousRemoved: relocation.PreviousRemoved,
	}, nil
}

// RetryStorage opens the configured location again after it could not be reached.
func (s *Service) RetryStorage() error {
	return present(s.vault.OpenStorage())
}

// newVaultLocation asks the owner where a new vault file goes, or proposes a free file for a host-kept kind.
func (s *Service) newVaultLocation(kind storage.Kind) (storage.Target, bool, error) {
	if slices.Contains(s.files.Kinds(), kind) {
		target, chosen, err := s.files.Create(kind, s.preferences.Dialogs().VaultFileName)
		if err != nil || !chosen || kind != storage.LocalFile {
			return target, chosen, err
		}
		// A local vault goes in the folder the owner picked, under the first name free there.
		free, err := s.freeVaultFile(target, filepath.Base(target.Path))
		if err != nil {
			return storage.Target{}, false, err
		}
		if free.Path == target.Path {
			free.Label = target.Label
		}
		return free, true, nil
	}
	home := s.vault.Storage().Default
	if kind != home.Kind {
		return storage.Target{}, false, storage.ErrUnsupportedKind
	}
	target, err := s.freeVaultFile(home, s.preferences.Dialogs().VaultFileName)
	return target, err == nil, err
}

func (s *Service) vaultAt(target storage.Target, current bool) Vault {
	shown := s.label(target)
	return Vault{
		Kind:    string(target.Kind),
		Path:    target.Path,
		Name:    shown.Name,
		Place:   shown.Place,
		Current: current,
	}
}
