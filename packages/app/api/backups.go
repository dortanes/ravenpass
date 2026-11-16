package api

import (
	"github.com/dortanes/ravenpass/packages/app/preferences"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// BackupFolders is how the owner picks the folder backups go to.
type BackupFolders interface {
	// Choose may return a zero label, which is then worded from the address.
	Choose(prompt string) (address string, label storage.Label, chosen bool, err error)
}

// AutoBackup is the automatic backup setting of the open vault and the choices offered.
type AutoBackup struct {
	Enabled   bool          `json:"enabled"`
	Interval  string        `json:"interval"`
	Intervals []string      `json:"intervals"`
	Keep      int           `json:"keep"`
	Keeps     []int         `json:"keeps"`
	Folder    *BackupFolder `json:"folder"`
	// LastBackupAt is in Unix milliseconds, zero for never.
	LastBackupAt int64 `json:"lastBackupAt"`
	// Failed reports that the last attempt for the open vault failed.
	Failed bool `json:"failed"`
}

// BackupFolder is how the interface shows the folder backups go to.
type BackupFolder struct {
	Name  string `json:"name"`
	Place string `json:"place"`
}

// dialogFolders picks the backup folder with the desktop's open dialog.
type dialogFolders struct {
	currentApp func() *application.App
	sandbox    Sandbox
}

func (folders dialogFolders) Choose(prompt string) (string, storage.Label, bool, error) {
	path, chosen, err := chooseFolder(folders.currentApp, folders.sandbox, prompt)
	return path, storage.Label{}, chosen, err
}

// GetAutoBackup reports the automatic backup setting and status.
func (s *Service) GetAutoBackup() (AutoBackup, error) {
	if s.backups == nil {
		return AutoBackup{}, fail(failureGeneral)
	}
	settings := s.preferences.AutoBackup()
	offered := preferences.BackupIntervals()
	intervals := make([]string, len(offered))
	for i, interval := range offered {
		intervals[i] = string(interval)
	}
	backup := AutoBackup{
		Enabled: settings.Enabled, Interval: string(settings.Interval), Intervals: intervals,
		Keep: settings.Keep, Keeps: preferences.BackupKeeps(),
	}
	if settings.Folder.Address != "" {
		backup.Folder = &BackupFolder{Name: settings.Folder.Name, Place: settings.Folder.Place}
	}
	status := s.backups.Status()
	if !status.Last.IsZero() {
		backup.LastBackupAt = status.Last.UnixMilli()
	}
	backup.Failed = status.Failed
	return backup, nil
}

// SetAutoBackup records the owner's choices and returns once any backup they make due is written.
func (s *Service) SetAutoBackup(enabled bool, interval string, keep int) error {
	if s.backups == nil {
		return fail(failureGeneral)
	}
	if err := s.preferences.SetAutoBackup(enabled, preferences.BackupInterval(interval), keep); err != nil {
		return present(err)
	}
	s.backups.Check()
	return nil
}

// ChooseBackupFolder records the folder the owner picks and returns after any due backup; false means canceled.
func (s *Service) ChooseBackupFolder() (bool, error) {
	if s.backups == nil {
		return false, fail(failureGeneral)
	}
	address, label, chosen, err := s.chooseBackupFolder()
	if err != nil {
		return false, present(err)
	}
	if !chosen {
		return false, nil
	}
	if label == (storage.Label{}) {
		label = s.label(storage.Target{Path: address})
	}
	if err := s.preferences.SetBackupFolder(preferences.BackupFolder{Address: address, Name: label.Name, Place: label.Place}); err != nil {
		return false, present(err)
	}
	s.backups.Check()
	return true, nil
}

// chooseBackupFolder shows the folder picker while the host is held.
func (s *Service) chooseBackupFolder() (string, storage.Label, bool, error) {
	defer s.hold()()
	return s.folders.Choose(s.preferences.Dialogs().ChooseBackupFolder)
}
