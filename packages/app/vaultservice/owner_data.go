package vaultservice

import (
	"log/slog"

	"github.com/dortanes/ravenpass/packages/vault"
)

// OwnerDataHolder keeps owner data sealed for vaults on this device.
type OwnerDataHolder interface {
	// Reseal passes each piece of owner data it keeps for id through reseal and returns the write that stores the
	// results; a piece reseal refuses is left as it was, so the new key no longer opens it.
	Reseal(id vault.ID, reseal func(name string, sealed []byte) ([]byte, error)) (write func(), err error)
}

// HoldOwnerData has a recovery key change seal holder's owner data again for the new key.
func (s *Service) HoldOwnerData(holder OwnerDataHolder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ownerData = append(s.ownerData, holder)
}

// SealOwnerData seals owner data with the open vault, or fails with ErrNotReady unless it is id.
func (s *Service) SealOwnerData(id vault.ID, name string, plaintext []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireOpen(id); err != nil {
		return nil, err
	}
	return s.session.SealOwnerData(name, plaintext)
}

// OpenOwnerData opens owner data with the open vault, or fails with ErrNotReady unless it is id.
func (s *Service) OpenOwnerData(id vault.ID, name string, sealed []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireOpen(id); err != nil {
		return nil, err
	}
	return s.session.OpenOwnerData(name, sealed)
}

// resealOwnerData seals every holder's owner data of id for rekey's key and returns the writes that store it. A holder
// that fails keeps its data under the old key. The caller holds s.mu.
func (s *Service) resealOwnerData(id vault.ID, rekey *vault.Rekey) []func() {
	reseal := func(name string, sealed []byte) ([]byte, error) {
		plaintext, err := s.session.OpenOwnerData(name, sealed)
		if err != nil {
			return nil, err
		}
		defer clear(plaintext)
		return rekey.SealOwnerData(name, plaintext)
	}
	var writes []func()
	for _, holder := range s.ownerData {
		write, err := holder.Reseal(id, reseal)
		if err != nil {
			slog.Warn("seal owner data for a new vault key", "err", err)
			continue
		}
		writes = append(writes, write)
	}
	return writes
}

func writeAll(writes []func()) {
	for _, write := range writes {
		write()
	}
}
