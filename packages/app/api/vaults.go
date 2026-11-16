package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/verification"
)

// proposedVaultNames bounds the numbered names tried for a new vault beside the current one.
const proposedVaultNames = 99

// VaultDeletion reports what a permanent deletion left on this device.
type VaultDeletion struct {
	// Deleted is false when the owner declined device authentication, which changes nothing.
	Deleted              bool   `json:"deleted"`
	Path                 string `json:"path"`
	DeviceRecordsRemoved bool   `json:"deviceRecordsRemoved"`
	SharedIdentity       bool   `json:"sharedIdentity"`
}

// SwitchVault closes the open vault and makes another known vault current.
func (s *Service) SwitchVault(path string) error {
	target, err := s.knownVault(path)
	if err != nil {
		return err
	}
	s.icons.Release()
	return present(s.vault.SwitchVault(target))
}

// KnowsVault reports whether any location this device knows holds a vault.
func (s *Service) KnowsVault() bool {
	return s.vault.HoldsAnyVault()
}

// CreateVault stages another vault at a free file beside the current one once the owner allows it; nothing is written yet.
func (s *Service) CreateVault(ctx context.Context) (StorageChange, error) {
	if allowed, err := s.ownerAllows(ctx, confirmation.CreatingVault()); !allowed {
		return StorageChange{}, err
	}
	status := s.vault.Storage()
	beside := status.Current
	if beside.Kind != status.Default.Kind {
		beside = status.Default
	}
	target, err := s.freeVaultFile(beside, s.preferences.Dialogs().VaultFileName)
	if err != nil {
		return StorageChange{}, err
	}
	s.icons.Release()
	if err := s.vault.PrepareVaultCreation(target); err != nil {
		return StorageChange{}, present(err)
	}
	return StorageChange{Changed: true, Path: target.Path}, nil
}

// CancelVaultCreation abandons setting up another vault and returns to the one before it.
func (s *Service) CancelVaultCreation() error {
	s.icons.Release()
	return present(s.vault.CancelVaultCreation())
}

// freeVaultFile is the first free vault file beside beside named as proposed: "vault.rpv", then "vault 2.rpv" and on.
func (s *Service) freeVaultFile(beside storage.Target, proposed string) (storage.Target, error) {
	folder := filepath.Dir(beside.Path)
	name := strings.TrimSuffix(proposed, vaultExtension)
	for number := 1; number <= proposedVaultNames; number++ {
		file := name + vaultExtension
		if number > 1 {
			file = fmt.Sprintf("%s %d%s", name, number, vaultExtension)
		}
		target := storage.Target{Kind: beside.Kind, Path: filepath.Join(folder, file)}
		free, err := s.vault.FreeLocation(target)
		if err != nil {
			return storage.Target{}, present(err)
		}
		if free {
			return target, nil
		}
	}
	return storage.Target{}, fail(failureVaultExists)
}

// OpenVault makes an existing vault file the owner picks current, once the owner allows it.
func (s *Service) OpenVault(ctx context.Context) (StorageChange, error) {
	if allowed, err := s.ownerAllows(ctx, confirmation.OpeningVault()); !allowed {
		return StorageChange{}, err
	}
	target, chosen, err := s.files.Existing()
	if err != nil || !chosen {
		return StorageChange{}, present(err)
	}
	s.icons.Release()
	if err := s.vault.SwitchVault(target); err != nil {
		return StorageChange{}, present(err)
	}
	return StorageChange{Changed: true, Path: target.Path}, nil
}

// ForgetVault drops a vault from the list and keeps its file.
func (s *Service) ForgetVault(path string) error {
	target, err := s.knownVault(path)
	if err != nil {
		return err
	}
	s.icons.Release()
	return present(s.vault.ForgetVault(target))
}

// DeleteVault irreversibly erases a vault's file and this device's records for it once the owner allows it.
func (s *Service) DeleteVault(ctx context.Context, path string) (VaultDeletion, error) {
	target, err := s.knownVault(path)
	if err != nil {
		return VaultDeletion{}, err
	}
	name := strings.TrimSuffix(s.label(target).Name, vaultExtension)
	if allowed, err := s.ownerAllows(ctx, confirmation.DeletingVault(name)); !allowed {
		return VaultDeletion{}, err
	}
	s.icons.Release()
	deletion, err := s.vault.DeleteVault(target)
	if err != nil {
		return VaultDeletion{}, present(err)
	}
	removed := deletion.DeviceRecordsRemoved
	if !deletion.SharedIdentity && s.icons.Forget(deletion.VaultID) != nil {
		removed = false
	}
	return VaultDeletion{
		Deleted:              true,
		Path:                 deletion.Target.Path,
		DeviceRecordsRemoved: removed,
		SharedIdentity:       deletion.SharedIdentity,
	}, nil
}

// ownerAllows asks device authentication for reason on a device that holds a vault; a decline is false with no error.
func (s *Service) ownerAllows(ctx context.Context, reason confirmation.Reason) (bool, error) {
	if !s.vault.HoldsAnyVault() {
		return true, nil
	}
	err := s.owner.VerifyDevice(ctx, reason)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, verification.ErrDeclined):
		return false, nil
	case errors.Is(err, verification.ErrUnverifiable):
		return false, fail(failureUnlockUnavailable)
	default:
		return false, present(err)
	}
}

func (s *Service) knownVault(path string) (storage.Target, error) {
	vaults, _ := s.vault.Vaults()
	for _, target := range vaults {
		if target.Path == path {
			return target, nil
		}
	}
	return storage.Target{}, fail(failureVaultUnknown)
}
