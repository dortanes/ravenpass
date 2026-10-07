package api

import (
	"errors"
	"testing"
)

// fakeQRCodes answers every read with texts or err.
type fakeQRCodes struct {
	texts []string
	err   error
}

func (f fakeQRCodes) Picture([]byte) ([]string, error) { return f.texts, f.err }

func (f fakeQRCodes) Clipboard() ([]string, error) { return f.texts, f.err }

func TestAHostQRCodeReaderIsOffered(t *testing.T) {
	if newServiceOnHost(t, Host{}).Capabilities().QRCodes {
		t.Fatal("a host without a QR code reader offers one")
	}
	if !newServiceOnHost(t, Host{QRCodes: fakeQRCodes{}}).Capabilities().QRCodes {
		t.Fatal("a host with a QR code reader does not offer it")
	}
}

func TestTheClipboardGivesTheOneSetupItsQRCodesHold(t *testing.T) {
	service := newReadyService(t)
	service.qrCodes = fakeQRCodes{texts: []string{"https://example.com", siteSetup, siteSetup}}
	if link, err := service.ReadCodeSetup(qrCodeFromClipboard); err != nil || link != siteSetup {
		t.Fatalf("read %q, error %v", link, err)
	}
}

func TestAPictureThatGivesNoSingleSetupIsRefused(t *testing.T) {
	other := "otpauth://totp/Example:sam@example.com?issuer=Example&secret=" + standardSecret
	for name, test := range map[string]struct {
		reader fakeQRCodes
		want   failure
	}{
		"no picture":           {fakeQRCodes{err: ErrNoPicture}, failureQRCodeMissing},
		"no QR code":           {fakeQRCodes{}, failureQRCodeMissing},
		"a web address":        {fakeQRCodes{texts: []string{"https://example.com"}}, failureQRCodeNotSetup},
		"a bare setup key":     {fakeQRCodes{texts: []string{standardSecret}}, failureQRCodeNotSetup},
		"two different setups": {fakeQRCodes{texts: []string{siteSetup, other}}, failureQRCodeAmbiguous},
		"a failed read":        {fakeQRCodes{err: errors.New("the reader failed")}, failureGeneral},
	} {
		t.Run(name, func(t *testing.T) {
			service := newReadyService(t)
			service.qrCodes = test.reader
			link, err := service.ReadCodeSetup(qrCodeFromClipboard)
			assertFailure(t, err, test.want)
			if link != "" {
				t.Fatalf("a refused read gave %q", link)
			}
		})
	}
}

func TestReadingAQRCodeNeedsAnOpenVaultAReaderAndAKnownSource(t *testing.T) {
	service := newReadyService(t)
	_, err := service.ReadCodeSetup(qrCodeFromClipboard)
	assertFailure(t, err, failureGeneral)

	service.qrCodes = fakeQRCodes{texts: []string{siteSetup}}
	_, err = service.ReadCodeSetup("screen")
	assertFailure(t, err, failureGeneral)
	_, err = service.ReadCodeSetup(qrCodeFromFile)
	assertFailure(t, err, failureWindowUnavailable)

	if err := service.Lock(); err != nil {
		t.Fatal(err)
	}
	_, err = service.ReadCodeSetup(qrCodeFromClipboard)
	assertFailure(t, err, failureVaultLocked)
}
