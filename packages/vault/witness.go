package vault

import "sync"

// compareWitness places head against the acknowledged version; a head that neither equals, descends from nor predates it diverged.
func compareWitness(head Head, ancestors ancestry, witness *Witness) (WitnessDecision, error) {
	if witness == nil {
		return 0, ErrWitnessMissing
	}
	if head.VaultID != witness.VaultID || head.Revision == 0 || witness.Revision == 0 {
		return 0, ErrWitnessMismatch
	}
	switch {
	case head.Revision == witness.Revision && head.Hash == witness.Hash:
		return WitnessEqual, nil
	case head.Revision < witness.Revision:
		return 0, ErrWitnessOlder
	case ancestors.names(head.Revision, *witness):
		return WitnessAdvance, nil
	default:
		return 0, ErrWitnessDiverged
	}
}

// ReconcileWitness places the session's version against witness, verifying every record of a successor.
func (s *Session) ReconcileWitness(witness *Witness) (WitnessDecision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return 0, ErrLocked
	}
	decision, err := compareWitness(s.head, s.ancestry, witness)
	if err != nil {
		return 0, err
	}
	if decision == WitnessAdvance {
		if err := s.verifyEntries(); err != nil {
			return 0, err
		}
	}
	return decision, nil
}

// WitnessFor is the witness that acknowledges head.
func WitnessFor(head Head) Witness {
	return Witness{VaultID: head.VaultID, Revision: head.Revision, Hash: head.Hash}
}

// Follow adopts another device's container once it descends from the session's head and persistAdvancedWitness records it;
// one sealed under another vault key fails with ErrKeyReplaced.
func (s *Session) Follow(container []byte, persistAdvancedWitness func(Witness) error) (WitnessDecision, error) {
	raw, err := parseContainer(container)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return 0, ErrLocked
	}
	if s.pending != nil {
		return 0, ErrPendingCommit
	}
	if raw.vaultID != s.vaultID {
		return 0, ErrWitnessMismatch
	}
	if keyIdentity(raw.recovery) != keyIdentity(s.recovery) {
		return 0, ErrKeyReplaced
	}
	next, err := openSession(raw, s.dataKey)
	if err != nil {
		return 0, err
	}
	defer next.Lock()
	acknowledged := WitnessFor(s.head)
	decision, err := compareWitness(next.head, next.ancestry, &acknowledged)
	if err != nil || decision == WitnessEqual {
		return decision, err
	}
	if persistAdvancedWitness == nil {
		return 0, ErrWitnessAdvanceRequired
	}
	if err := next.verifyEntries(); err != nil {
		return 0, err
	}
	if err := persistAdvancedWitness(WitnessFor(next.head)); err != nil {
		return 0, err
	}
	s.recovery = next.recovery
	s.entries = next.entries
	s.groups = next.groups
	s.retention = next.retention
	s.records = next.records
	s.container = next.container
	s.head = next.head
	s.ancestry = next.ancestry
	s.selected = false
	s.selectionToken++
	return WitnessAdvance, nil
}

// Divergence is a verified vault not descending from the acknowledged version, unreadable until adopted.
type Divergence struct {
	mu      sync.Mutex
	session *Session
}

// Adopt records the diverged head through persistAdoptedWitness and returns its session; after a failed record, Adopt or Discard it opens nothing.
func (d *Divergence) Adopt(persistAdoptedWitness func(Witness) error) (*Session, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	session := d.session
	d.session = nil
	if session == nil {
		return nil, ErrLocked
	}
	if persistAdoptedWitness == nil {
		session.Lock()
		return nil, ErrWitnessAdvanceRequired
	}
	head, err := session.Head()
	if err != nil {
		return nil, err
	}
	if err := persistAdoptedWitness(WitnessFor(head)); err != nil {
		session.Lock()
		return nil, err
	}
	return session, nil
}

// Discard locks the session the divergence holds.
func (d *Divergence) Discard() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.session != nil {
		d.session.Lock()
		d.session = nil
	}
}
