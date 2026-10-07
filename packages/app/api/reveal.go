package api

import (
	"context"
	"errors"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/ownerauth"
	"github.com/dortanes/ravenpass/packages/app/verification"
)

// ConfirmReveal confirms the owner before the interface shows or copies the secret of the item labelled item: by
// device authentication where the vault opens with it, else by pin, the vault's PIN, which counts a wrong PIN as the
// locked screen does. A vault that opens with neither has nothing to confirm with and passes.
func (s *Service) ConfirmReveal(ctx context.Context, item, pin string) error {
	methods, err := s.vault.UnlockMethods()
	if err != nil {
		return present(err)
	}
	switch {
	case methods.BiometryEnabled && methods.BiometryAvailable:
		verified := s.owner.VerifyDevice(ctx, confirmation.Revealing(item))
		if errors.Is(verified, verification.ErrUnverifiable) || errors.Is(verified, ownerauth.ErrFailed) {
			return fail(failureOwnerUnverified)
		}
		return present(verified)
	case methods.PINSet:
		return present(s.vault.VerifyPIN(pin))
	default:
		return nil
	}
}
