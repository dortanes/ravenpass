package vault

// PrepareMerge prepares, in one save, credential into holding every passkey of credential from that it does not hold
// and then changed by patch, and from moved to the trash at the Unix millisecond at without its passkeys, so a restore
// never leaves one key in two credentials. Passkeys keep their private keys, which no caller sees; more than a
// credential holds fail with ErrPasskeysFull.
func (s *Session) PrepareMerge(into, from ID, patch CredentialPatch, at uint64) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if into == from || at == 0 {
		return nil, ErrInvalidInput
	}
	index, input, err := s.credentialToChange(into)
	if err != nil {
		return nil, err
	}
	defer forgetPasskeyKeys(input.Passkeys)
	other := s.findKind(from, KindCredential)
	if other < 0 {
		return nil, ErrNotFound
	}
	merged, err := s.decryptCredential(other)
	if err != nil {
		return nil, err
	}
	defer forgetPasskeyKeys(merged.Passkeys)
	combined := input
	for _, passkey := range merged.Passkeys {
		if passkeyIndex(combined.Passkeys, passkey.CredentialID) < 0 {
			combined.Passkeys = append(combined.Passkeys, passkey)
		}
	}
	combined, membership, err := s.patched(index, combined, patch)
	if err != nil {
		return nil, err
	}
	plaintext, err := encodeCredentialRecord(combined)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	entry, box, err := s.sealNext(index, credentialEntry(combined, membership), plaintext)
	if err != nil {
		return nil, err
	}
	entries := append([]entryMeta(nil), s.entries...)
	records := append([]sealedBox(nil), s.records...)
	entries[index], records[index] = entry, box
	if len(merged.Passkeys) > 0 {
		left := merged
		left.Passkeys = nil
		rest, err := encodeCredentialRecord(left)
		if err != nil {
			return nil, err
		}
		defer clear(rest)
		trashed := s.entries[other]
		trashed.passkeys = nil
		if entries[other], records[other], err = s.sealNext(other, trashed, rest); err != nil {
			return nil, err
		}
	}
	entries[other].deletedAt = at
	return s.prepare(entries, records)
}
