package api

import (
	"errors"
	"slices"

	"github.com/dortanes/ravenpass/packages/app/photos"
	"github.com/dortanes/ravenpass/packages/vault"
)

// ErrNoPicture reports a clipboard that holds no picture.
var ErrNoPicture = errors.New("the clipboard holds no picture")

// QRCodeReader finds QR codes in pictures on a host.
type QRCodeReader interface {
	// Picture returns the text of each QR code in content, an image file's bytes.
	Picture(content []byte) ([]string, error)
	// Clipboard returns the text of each QR code in the clipboard's picture, or in the image file copied there, or
	// ErrNoPicture.
	Clipboard() ([]string, error)
}

// The places ReadCodeSetup looks for a QR code.
const (
	qrCodeFromFile      = "file"
	qrCodeFromClipboard = "clipboard"
)

// ReadCodeSetup returns the one-time code setup a QR code holds in a picture the owner chooses, for source "file", or
// in the clipboard's picture, for "clipboard"; empty when the owner cancels the file choice.
func (s *Service) ReadCodeSetup(source string) (string, error) {
	if s.qrCodes == nil {
		return "", fail(failureGeneral)
	}
	if err := s.requireOpenVault(); err != nil {
		return "", err
	}
	var texts []string
	switch source {
	case qrCodeFromFile:
		dialogs := s.preferences.Dialogs()
		picked, err := s.dialog.pick(dialogs.ChooseQRCode, dialogs.PhotoFilter, photoExtensions)
		if err != nil || picked.path == "" {
			return "", err
		}
		content, err := readChosenFile(picked.path, photos.MaxFileBytes)
		picked.discard()
		if err != nil {
			return "", present(err)
		}
		defer clear(content)
		if texts, err = s.qrCodes.Picture(content); err != nil {
			return "", present(err)
		}
	case qrCodeFromClipboard:
		var err error
		texts, err = s.qrCodes.Clipboard()
		if errors.Is(err, ErrNoPicture) {
			return "", fail(failureQRCodeMissing)
		}
		if err != nil {
			return "", present(err)
		}
	default:
		return "", fail(failureGeneral)
	}
	return codeSetupIn(texts)
}

// codeSetupIn returns the one setup link among texts, refusing none or two different ones.
func codeSetupIn(texts []string) (string, error) {
	if len(texts) == 0 {
		return "", fail(failureQRCodeMissing)
	}
	var setups []string
	for _, text := range texts {
		if _, err := vault.ReadSetupLink(text); err == nil && !slices.Contains(setups, text) {
			setups = append(setups, text)
		}
	}
	switch len(setups) {
	case 0:
		return "", fail(failureQRCodeNotSetup)
	case 1:
		return setups[0], nil
	default:
		return "", fail(failureQRCodeAmbiguous)
	}
}
