package preferences

// BreachChecks reports whether passwords are checked against known breaches; off by default.
func (s *Store) BreachChecks() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	return s.current.BreachChecksOn
}

// SetBreachChecks records whether passwords are checked against known breaches.
func (s *Store) SetBreachChecks(enabled bool) error {
	return s.update(func(next *record) { next.BreachChecksOn = enabled })
}
