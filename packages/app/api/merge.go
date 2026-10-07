package api

import (
	"errors"

	"github.com/dortanes/ravenpass/packages/vault"
)

// MergeCredentials saves input and groups over credential into as UpdateCredential does, with linked apps either
// credential holds, moves every passkey of from into it, and moves from to the trash, in one save.
func (s *Service) MergeCredentials(into, from string, input CredentialInput, groups []string) error {
	kept, err := vault.ParseID(into)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	merged, err := vault.ParseID(from)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	held, err := s.heldApps(kept)
	if err != nil {
		return err
	}
	others, err := s.heldApps(merged)
	if err != nil {
		return err
	}
	patch, err := s.credentialPatch(input, groups, append(held, others...))
	if err != nil {
		return err
	}
	err = s.vault.MergeCredentials(kept, merged, patch)
	if errors.Is(err, vault.ErrPasskeysFull) {
		return fail(failurePasskeysFull)
	}
	return present(err)
}
