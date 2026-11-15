// Package appcore assembles the vault services every host builds once at start-up.
package appcore

import (
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/dortanes/ravenpass/packages/app/api"
	"github.com/dortanes/ravenpass/packages/app/autofill"
	"github.com/dortanes/ravenpass/packages/app/backupexclusion"
	"github.com/dortanes/ravenpass/packages/app/backups"
	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/devicerecords"
	"github.com/dortanes/ravenpass/packages/app/localfile"
	"github.com/dortanes/ravenpass/packages/app/ownerauth"
	"github.com/dortanes/ravenpass/packages/app/passwordhistory"
	"github.com/dortanes/ravenpass/packages/app/preferences"
	"github.com/dortanes/ravenpass/packages/app/siteicons"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/unlock"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/app/verification"
)

// The files the services keep in the host's directory.
const (
	storageFile     = "storage.json"
	preferencesFile = "preferences.json"
	deviceFile      = "device.json"
	iconsDirectory  = "icons"
	// historyDirectory replaces 0.3's "generator", which is left in place and no longer read.
	historyDirectory = "password-history"
	backupsFile      = "backups.json"
)

// Platform is what a host supplies to the start-up chain.
type Platform struct {
	// Directory holds the files the services keep.
	Directory string
	Languages preferences.DeviceLanguages
	// Preferences are the host's preference options.
	Preferences []preferences.Option
	// Backends are the storage kinds offered after the local file.
	Backends []storage.Backend
	// Owner is the system's owner authentication; nil uses the one ownerauth reaches itself.
	Owner             ownerauth.Platform
	PIN               unlock.Binding
	Presence          unlock.PresenceBinding
	BackupDestination backups.Destination
}

// Services are what Build assembles.
type Services struct {
	Files         *storage.Manager
	Settings      *preferences.Store
	Owner         *ownerauth.Authenticator
	Vault         *vaultservice.Service
	Icons         *siteicons.Service
	History       *passwordhistory.Store
	Confirmations *confirmation.Queue
	Verifier      *verification.Verifier
	Autofill      *autofill.Service
	Backups       *backups.Keeper
}

// Build assembles the services over platform's files; a vault location that cannot be opened is left for the interface to report.
func Build(platform Platform) (*Services, error) {
	if platform.Directory == "" || platform.Languages == nil || platform.BackupDestination == nil {
		return nil, errors.New("the directory, the device's languages and the backup destination are required")
	}
	directory := platform.Directory
	files, err := storage.NewManager(
		filepath.Join(directory, storageFile),
		storage.Target{Kind: storage.LocalFile, Path: filepath.Join(directory, localfile.DefaultVaultName)},
		api.MaxVaultBytes,
		append([]storage.Backend{localfile.Backend{Home: directory}}, platform.Backends...)...,
	)
	if err != nil {
		return nil, err
	}
	if err := files.Open(); err != nil {
		slog.Warn("the vault location could not be opened", "err", err)
	}
	settings, err := preferences.New(filepath.Join(directory, preferencesFile), platform.Languages, platform.Preferences...)
	if err != nil {
		return nil, err
	}
	owner := ownerauth.New()
	if platform.Owner != nil {
		owner = ownerauth.NewFor(platform.Owner)
	}
	keys, err := devicerecords.New(filepath.Join(directory, deviceFile), backupexclusion.System{})
	if err != nil {
		return nil, err
	}
	vault, err := vaultservice.New(files, keys, vaultservice.Device{Owner: owner, PIN: platform.PIN, Platform: platform.Presence})
	if err != nil {
		return nil, err
	}
	icons, err := siteicons.New(filepath.Join(directory, iconsDirectory), vault, siteicons.NewFetcher(), settings.SiteIcons())
	if err != nil {
		return nil, err
	}
	history := passwordhistory.New(filepath.Join(directory, historyDirectory), vault, settings.PasswordHistory)
	vault.HoldOwnerData(history)
	confirmations, err := confirmation.New(vault)
	if err != nil {
		return nil, err
	}
	s := &Services{
		Files: files, Settings: settings, Owner: owner, Vault: vault, Icons: icons, History: history, Confirmations: confirmations,
	}
	if s.Verifier, err = verification.New(vault, owner, s.Words, confirmations); err != nil {
		return nil, err
	}
	if s.Autofill, err = autofill.New(vault, settings); err != nil {
		return nil, err
	}
	if s.Backups, err = backups.New(filepath.Join(directory, backupsFile), vault, settings.AutoBackup, platform.BackupDestination); err != nil {
		return nil, err
	}
	return s, nil
}

// Words words reason for the system prompt in the interface language.
func (s *Services) Words(reason confirmation.Reason) string {
	return reason.Words(s.Settings.Dialogs())
}

// Serve composes the service the interface binds to, setting the confirmation queue, owner verifier and backups in host.
func (s *Services) Serve(host api.Host) (*api.Service, error) {
	host.Confirmations.Queue = s.Confirmations
	host.Confirmations.Owner = s.Verifier
	host.Backups = s.Backups
	host.History = s.History
	return api.New(s.Vault, s.Settings, s.Icons, host)
}
