package api

import (
	"context"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
)

// reasonKinds names each kind of verification as the interface tells them apart.
var reasonKinds = map[confirmation.ReasonKind]string{
	confirmation.ReasonShare:        "share",
	confirmation.ReasonSavePasskey:  "save-passkey",
	confirmation.ReasonSignIn:       "sign-in",
	confirmation.ReasonFill:         "fill",
	confirmation.ReasonFillCard:     "fill-card",
	confirmation.ReasonChangeUnlock: "change-unlock",
}

// ConfirmationPanel is the window the confirmation page sits in.
type ConfirmationPanel interface {
	// Fit sets the panel's height, in points, to the one its page needs.
	Fit(height int)
}

// Confirmation is a request waiting in the confirmation panel; fields a request does not name are empty.
type Confirmation struct {
	ID string `json:"id"`
	// Kind is "verify" or "unlock".
	Kind string `json:"kind"`
	// Reason is "share", "save-passkey", "sign-in", "fill", "fill-card" or "change-unlock" for a verify request.
	Reason   string `json:"reason"`
	File     string `json:"file"`
	Identity string `json:"identity"`
	Site     string `json:"site"`
	Account  string `json:"account"`
	// Requester is "extension" or "autofill" for an unlock request.
	Requester string `json:"requester"`
}

// AwaitConfirmation returns the oldest waiting request other than shown, or empty once none waits unless shown is empty.
func (s *Service) AwaitConfirmation(ctx context.Context, shown string) (Confirmation, error) {
	request, err := s.confirmations.Next(ctx, shown)
	if err != nil {
		return Confirmation{}, present(err)
	}
	next := Confirmation{ID: request.ID, Kind: string(request.Kind)}
	switch request.Kind {
	case confirmation.KindVerify:
		reason := request.Reason
		next.Reason, next.File, next.Identity, next.Site, next.Account = reasonKinds[reason.Kind], reason.File, reason.Identity, reason.Site, reason.Account
	case confirmation.KindUnlock:
		next.Requester = string(request.Requester)
	}
	return next, nil
}

// ConfirmWithPIN answers the verify or unlock request id with pin; a wrong PIN leaves the request waiting.
func (s *Service) ConfirmWithPIN(id, pin string) error {
	request, err := s.confirmations.Waiting(id)
	if err != nil {
		return present(err)
	}
	if request.Kind == confirmation.KindUnlock {
		return s.unlockedFromPanel(s.UnlockWithPIN(pin))
	}
	return present(s.confirmations.ConfirmPIN(id, pin))
}

// UnlockFromConfirmation unlocks the vault for the unlock request id with device authentication.
func (s *Service) UnlockFromConfirmation(id string) error {
	if err := s.awaitingUnlock(id); err != nil {
		return err
	}
	return s.unlockedFromPanel(s.Unlock())
}

// unlockOnDevice opens the vault with device authentication for an unlock request before the panel shows it, and
// reports whether it did; false at once unless the vault opens with device authentication that is available.
func (s *Service) unlockOnDevice() bool {
	methods, err := s.vault.UnlockMethods()
	if err != nil || !methods.BiometryEnabled || !methods.BiometryAvailable {
		return false
	}
	return s.unlockedFromPanel(s.Unlock()) == nil
}

// unlockedFromPanel reloads an open main window when err reports the vault open, and returns err.
func (s *Service) unlockedFromPanel(err error) error {
	if err == nil {
		s.reloadMain()
	}
	return err
}

// RecoverFromConfirmation ends the unlock request id and brings the main window forward for recovery.
func (s *Service) RecoverFromConfirmation(id string) error {
	if err := s.awaitingUnlock(id); err != nil {
		return err
	}
	if err := s.confirmations.Decline(id); err != nil {
		return present(err)
	}
	s.showMain()
	return nil
}

// DeclineConfirmation ends the request id, of any kind, as declined; one that no longer waits fails.
func (s *Service) DeclineConfirmation(id string) error {
	return present(s.confirmations.Decline(id))
}

// noPanel is the panel of a host that shows confirmations inside its main window.
type noPanel struct{}

func (noPanel) Fit(int) {}

// FitConfirmation sets the confirmation panel's height, in points, to the one its page needs.
func (s *Service) FitConfirmation(height int) {
	s.panel.Fit(height)
}

// awaitingUnlock fails with confirmation-ended unless id is a waiting unlock request.
func (s *Service) awaitingUnlock(id string) error {
	request, err := s.confirmations.Waiting(id)
	if err != nil {
		return present(err)
	}
	if request.Kind != confirmation.KindUnlock {
		return present(confirmation.ErrEnded)
	}
	return nil
}
