package vaultservice

import (
	"crypto/sha256"
	"errors"
	"log/slog"

	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/unlock"
	"github.com/dortanes/ravenpass/packages/vault"
)

// Failures of a key change that did not finish.
var (
	// ErrKeyChangeUnfinished reports a new file that is written while the save around it failed: the new recovery key
	// opens the vault, and the ways in follow it where this device could record them.
	ErrKeyChangeUnfinished = errors.New("the vault key changed but this device did not finish recording it")
	// ErrKeyChangeUncertain reports a failed save whose file cannot be read back: either recovery key may open the vault.
	ErrKeyChangeUncertain = errors.New("whether the vault key changed is not known")
)

// stagedRekey is a new vault key and recovery phrase awaiting confirmation of the phrase.
type stagedRekey struct {
	rekey *vault.Rekey
	// pinEnvelope is the PIN's copy of the new key, nil where the vault has no PIN on this device.
	pinEnvelope []byte
}

// BeginRekey stages a new vault key and recovery phrase for the vault opening names and returns the phrase; nothing
// is written. Where the vault has a PIN on this device, pin must be it, counted and locking as VerifyPIN does, so the
// PIN keeps opening the vault; elsewhere pin is empty.
func (s *Service) BeginRekey(opening Opening, pin string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discardRekey()
	if err := s.openedAs(opening); err != nil {
		return "", err
	}
	id, policy, err := s.openVaultPolicy()
	if err != nil {
		return "", err
	}
	if !policy.HasPIN() && pin != "" {
		return "", unlock.ErrNoPIN
	}
	if policy.HasPIN() && !vault.ValidPIN(pin) {
		return "", vault.ErrInvalidPIN
	}
	rekey, err := s.session.BeginRekey()
	if err != nil {
		return "", err
	}
	stage := &stagedRekey{rekey: rekey}
	if policy.HasPIN() {
		err := s.attemptPIN(id, policy, pin, func(key []byte) error {
			if err := s.session.VerifyDeviceKey(key, policy.PIN.Envelope); err != nil {
				return err
			}
			var err error
			stage.pinEnvelope, err = rekey.WrapDeviceKey(key)
			return err
		})
		if err != nil {
			rekey.Discard()
			s.lockOnPINRemoved(err)
			return "", err
		}
	}
	s.rekeying = stage
	return rekey.RecoveryPhrase(), nil
}

// VerifyRecoveryPhrase fails with vault.ErrInvalidPhrase or vault.ErrAuthentication unless phrase opens the open vault.
func (s *Service) VerifyRecoveryPhrase(phrase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return ErrNotReady
	}
	return s.session.VerifyRecoveryPhrase(phrase)
}

// ConfirmRekey seals the open vault under the staged key once phrase matches the staged recovery phrase, and moves this
// device's ways in to the new key. The new file replaces the old one whole; other devices then need the new phrase.
// A failure that leaves the vault open keeps the staged phrase; one after the file is written fails with
// ErrKeyChangeUnfinished, and one whose file cannot be read back with ErrKeyChangeUncertain.
func (s *Service) ConfirmRekey(phrase string) (vault.Head, error) {
	stage, id, platform, err := s.rekeyToConfirm(phrase)
	if err != nil {
		return vault.Head{}, err
	}
	// Device authentication is bound to a new hardware key, which takes seconds on some devices; the vault stays
	// lockable meanwhile.
	var rebound *unlock.PlatformCredential
	if platform {
		if rebound, err = s.platforms.Set(id, stage.rekey.WrapDeviceKey); err != nil {
			return vault.Head{}, err
		}
	}
	return s.commitRekey(stage, rebound)
}

// rekeyToConfirm checks phrase against the staged rekey and reports whether device authentication must be bound again.
func (s *Service) rekeyToConfirm(phrase string) (*stagedRekey, vault.ID, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stage := s.rekeying
	if stage == nil {
		return nil, vault.ID{}, false, ErrNoPendingSetup
	}
	if err := stage.rekey.VerifyPhrase(phrase); err != nil {
		return nil, vault.ID{}, false, err
	}
	id, policy, err := s.openVaultPolicy()
	if err != nil {
		return nil, vault.ID{}, false, err
	}
	return stage, id, policy.HasPlatform(), nil
}

// commitRekey writes the vault sealed under the staged key, rebound holding device authentication's copy of it, and
// then moves this device's ways in and key record to the new key, as a save records its witness. Where the file holds
// the new key although the save failed, they move to it all the same.
func (s *Service) commitRekey(stage *stagedRekey, rebound *unlock.PlatformCredential) (vault.Head, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rekeying != stage {
		return vault.Head{}, ErrNoPendingSetup
	}
	id, policy, err := s.openVaultPolicy()
	if err != nil {
		return vault.Head{}, err
	}
	if policy.HasPlatform() != (rebound != nil) {
		return vault.Head{}, ErrSetupInProgress
	}
	history, err := s.loadKeyHistory(id)
	if err != nil {
		return vault.Head{}, err
	}
	previous, err := s.session.Head()
	if err != nil {
		return vault.Head{}, err
	}
	_, replaced, err := sessionKey(s.session)
	if err != nil {
		return vault.Head{}, err
	}
	// A device whose record never started still learns which key it replaced.
	moved := history.movedTo(replaced).movedTo(stage.rekey.KeyIdentity())
	next := rewrapped(policy, stage, rebound)
	record := func() error { return errors.Join(s.savePolicy(id, next), s.saveKeyHistory(id, moved)) }
	// Owner data is sealed for the new key ahead of the save, and stored only once the file holds that key.
	ownerWrites := s.resealOwnerData(id, stage.rekey)
	pending, err := s.session.PrepareRekey(stage.rekey)
	if err != nil {
		return vault.Head{}, err
	}
	written := pending.Head()
	err = s.commitThen(pending, record)
	switch {
	case err == nil:
		writeAll(ownerWrites)
		s.discardRekey()
		return written, nil
	case s.fileHolds(written):
		if repairErr := record(); repairErr != nil {
			slog.Warn("move the ways in to a new vault key the file holds", "err", repairErr)
		}
		writeAll(ownerWrites)
		return vault.Head{}, errors.Join(ErrKeyChangeUnfinished, err)
	case errors.Is(err, storage.ErrTooLarge), errors.Is(err, storage.ErrStaleHead), s.fileHolds(previous):
		return vault.Head{}, err
	default:
		return vault.Head{}, errors.Join(ErrKeyChangeUncertain, err)
	}
}

// fileHolds reports whether the bound location holds the version head names. The caller holds s.mu.
func (s *Service) fileHolds(head vault.Head) bool {
	container, err := s.files.LoadCiphertext()
	defer clear(container)
	return err == nil && sha256.Sum256(container) == head.Hash
}

// DiscardRekey drops the staged vault key and recovery phrase, if any.
func (s *Service) DiscardRekey() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discardRekey()
}

func (s *Service) discardRekey() {
	if s.rekeying != nil {
		s.rekeying.rekey.Discard()
		s.rekeying = nil
	}
}

// rewrapped is policy with each way in holding the staged key: the PIN's envelope wrapped when the rekey began, device
// authentication rebound.
func rewrapped(policy unlock.Policy, stage *stagedRekey, rebound *unlock.PlatformCredential) unlock.Policy {
	var next unlock.Policy
	if policy.HasPIN() && stage.pinEnvelope != nil {
		pin := *policy.PIN
		pin.Envelope = stage.pinEnvelope
		next.PIN = &pin
	}
	next.Platform = rebound
	return next
}
