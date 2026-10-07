package api

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/linkstore"
	"github.com/dortanes/ravenpass/packages/app/ownerauth"
	"github.com/dortanes/ravenpass/packages/app/photos"
	"github.com/dortanes/ravenpass/packages/app/preferences"
	"github.com/dortanes/ravenpass/packages/app/savedfile"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/unlock"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/app/verification"
	"github.com/dortanes/ravenpass/packages/vault"
)

func TestEveryKnownCauseReportsItsOwnCode(t *testing.T) {
	causes := map[error]failure{
		ownerauth.ErrCanceled:                  failureAuthenticationCanceled,
		ownerauth.ErrFailed:                    failureAuthenticationFailed,
		ownerauth.ErrUnavailable:               failureUnlockUnavailable,
		unlock.ErrUnbound:                      failureUnlockKeyMissing,
		vaultservice.ErrWayInNotSet:            failureWayInNotSet,
		vaultservice.ErrConfirmationNeeded:     failureRecoveryNeedsConfirm,
		vaultservice.ErrRecoveryChanged:        failureRecoveryChanged,
		vault.ErrInvalidPhrase:                 failureRecoveryPhraseInvalid,
		vaultservice.ErrAlreadyInitialized:     failureVaultExists,
		vaultservice.ErrSetupInProgress:        failureSetupInProgress,
		vaultservice.ErrNoPendingSetup:         failureSetupNotActive,
		storage.ErrUnavailable:                 failureStorageUnavailable,
		storage.ErrSelectionInvalid:            failureStorageSelectionInvalid,
		storage.ErrTargetOccupied:              failureStorageOccupied,
		storage.ErrSameLocation:                failureStorageSameLocation,
		storage.ErrUnsupportedKind:             failureStorageKindUnsupported,
		storage.ErrUnknownVault:                failureVaultUnknown,
		storage.ErrInvalidPath:                 failureStoragePathInvalid,
		vaultservice.ErrMoveVerification:       failureMoveUnverified,
		vaultservice.ErrSecondCopy:             failureSecondCopy,
		vaultservice.ErrOlderCopy:              failureOlderCopy,
		vaultservice.ErrDiverged:               failureVaultDiverged,
		vaultservice.ErrNoOtherVault:           failureNoOtherVault,
		vault.ErrWitnessMissing:                failureWitnessMissing,
		vault.ErrKeyReplaced:                   failureVaultKeyReplaced,
		vaultservice.ErrStaleExport:            failureExportStale,
		storage.ErrVaultMissing:                failureVaultMissing,
		storage.ErrStaleHead:                   failureVaultChanged,
		vault.ErrWitnessMismatch:               failureVaultChanged,
		storage.ErrDurabilityUncertain:         failureSaveUnconfirmed,
		storage.ErrFinalizerFailed:             failureSaveUnconfirmed,
		vault.ErrAuthentication:                failureAuthenticationInvalid,
		vaultservice.ErrNotReady:               failureVaultLocked,
		storage.ErrNotFound:                    failureNoLocalVault,
		savedfile.ErrInvalidSize:               failureFileSizeInvalid,
		savedfile.ErrUnverified:                failureFileUnverified,
		preferences.ErrUnsupportedBackupChoice: failureBackupChoiceUnsupported,
		preferences.ErrBackupFolderMissing:     failureBackupFolderMissing,
		photos.ErrUnsupported:                  failurePhotoUnsupported,
		photos.ErrTooLarge:                     failurePhotoTooLarge,
		photos.ErrOutside:                      failureInvalidItem,
		vault.ErrScanLimit:                     failureScanLimit,
		storage.ErrTooLarge:                    failureResourceLimit,
		vault.ErrUnsupported:                   failureUnsupportedFormat,
		vault.ErrMalformed:                     failureMalformedVault,
		vault.ErrInvalidInput:                  failureInvalidItem,
		vault.ErrNotFound:                      failureItemUnreadable,
		vault.ErrInvalidTOTP:                   failureInvalidCode,
		vault.ErrNoTOTP:                        failureNoCode,
		vault.ErrInvalidPIN:                    failurePINInvalid,
		unlock.ErrWrongPIN:                     failurePINWrong,
		unlock.ErrTooSoon:                      failurePINTooSoon,
		unlock.ErrPINRemoved:                   failurePINRemoved,
		unlock.ErrNoPIN:                        failurePINMissing,
		unlock.ErrNotAvailable:                 failureBiometryUnavailable,
		unlock.ErrDisabled:                     failureBiometryDisabled,
		unlock.ErrNoMethodLeft:                 failureUnlockMethodRequired,
		unlock.ErrMalformed:                    failureUnlockRecordUnreadable,
		unlock.ErrUnsupported:                  failureUnlockRecordUnreadable,
		vault.ErrResourceLimit:                 failureResourceLimit,
		linkserver.ErrExpired:                  failureLinkExpired,
		linkserver.ErrCanceled:                 failureLinkCanceled,
		linkserver.ErrUnavailable:              failureLinkUnavailable,
		linkstore.ErrNotFound:                  failureExtensionNotFound,
		confirmation.ErrEnded:                  failureConfirmationEnded,
		verification.ErrDeclined:               failureOwnerUnverified,
		os.ErrPermission:                       failurePermissionDenied,
	}
	for cause, code := range causes {
		want := fail(code).Error()
		if got := present(cause).Error(); got != want {
			t.Errorf("present(%v) = %q, want %q", cause, got, want)
		}
		// A cause reaches the service wrapped in the context of the action that failed.
		wrapped := fmt.Errorf("saving the vault: %w", cause)
		if got := present(wrapped).Error(); got != want {
			t.Errorf("present(wrapped %v) = %q, want %q", cause, got, want)
		}
	}
	if present(nil) != nil {
		t.Error("a successful action was presented as a failure")
	}
	if got := present(errors.New("something else")).Error(); got != fail(failureGeneral).Error() {
		t.Errorf("an unmapped cause = %q", got)
	}
}

func TestAVaultFileThatIsGoneIsNotReportedAsChanged(t *testing.T) {
	gone := fmt.Errorf("%w: %w", storage.ErrStaleHead, storage.ErrVaultMissing)
	for _, cause := range []error{gone, fmt.Errorf("%w: %w", vaultservice.ErrStaleExport, gone)} {
		if got, want := present(cause).Error(), fail(failureVaultMissing).Error(); got != want {
			t.Errorf("present(%v) = %q, want %q", cause, got, want)
		}
	}
}

func TestFailureCodesTravelAsMarkedMessages(t *testing.T) {
	message := fail(failureVaultLocked).Error()
	if !strings.HasPrefix(message, failurePrefix) {
		t.Fatalf("failure message %q carries no prefix", message)
	}
	if strings.TrimPrefix(message, failurePrefix) != string(failureVaultLocked) {
		t.Fatalf("failure message %q does not carry its code", message)
	}
	if strings.ContainsAny(message, " .") {
		t.Fatalf("failure message %q reads like a sentence", message)
	}
}
