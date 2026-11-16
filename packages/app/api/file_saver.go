package api

import "github.com/wailsapp/wails/v3/pkg/application"

// Media types of the files the owner saves besides scans, which carry their own.
const (
	// mediaVault is the type of a vault file, which has no registered type of its own.
	mediaVault = "application/octet-stream"
	mediaText  = "text/plain"
)

// SavedFile is a file the owner saves where they choose.
type SavedFile struct {
	Prompt string
	// Name is the file name the save dialog proposes.
	Name      string
	MediaType string
	// Filter names the kind of file the dialog lists.
	Filter string
	// Pattern, such as "*.rpv", matches Filter; empty lists every file.
	Pattern string
}

// FileSaver is how the owner saves a file where they choose on a host.
type FileSaver interface {
	// Choose asks where file goes and reports false when the owner cancels.
	Choose(file SavedFile) (SaveTarget, bool, error)
}

// SaveTarget is where the owner chose to save one file.
type SaveTarget interface {
	// Write puts data there. A file that does not read back as data fails with ErrFileUnverified.
	Write(data []byte) error
	// Discard removes what choosing the target created there and what a failed Write left.
	Discard()
}

// dialogSaver saves files with the desktop's save dialog.
type dialogSaver struct {
	currentApp func() *application.App
	sandbox    Sandbox
}

func (saver dialogSaver) Choose(file SavedFile) (SaveTarget, bool, error) {
	app := saver.currentApp()
	if app == nil {
		return nil, false, fail(failureWindowUnavailable)
	}
	dialog := app.Dialog.SaveFile().SetMessage(file.Prompt).SetFilename(file.Name)
	if file.Pattern != "" {
		dialog.AddFilter(file.Filter, file.Pattern)
	}
	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return nil, false, fail(failureLocationNotSelected)
	}
	if path == "" {
		return nil, false, nil
	}
	return savedPath{path: path, sandbox: saver.sandbox}, true, nil
}

// savedPath is a file the desktop save dialog named; a failed write leaves the prior content, so Discard is a no-op.
type savedPath struct {
	path    string
	sandbox Sandbox
}

func (saved savedPath) Write(data []byte) error {
	return saved.sandbox.Replace(saved.path, data, MaxVaultBytes)
}

func (savedPath) Discard() {}

// saveFile writes to the target the owner picks while holding the host; false means canceled.
func (s *Service) saveFile(file SavedFile, write func(SaveTarget) error) (bool, error) {
	defer s.hold()()
	target, chosen, err := s.saver.Choose(file)
	if err != nil {
		return false, present(err)
	}
	if !chosen {
		return false, nil
	}
	if err := write(target); err != nil {
		target.Discard()
		return false, present(err)
	}
	return true, nil
}
