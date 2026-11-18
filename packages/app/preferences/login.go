package preferences

// HiddenAtLogin reports whether a launch at login stays in the menu bar; on by default.
func (s *Store) HiddenAtLogin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	return !s.current.ShowWindowAtLogin
}

// SetHiddenAtLogin records whether a launch at login stays in the menu bar.
func (s *Store) SetHiddenAtLogin(hidden bool) error {
	return s.update(func(next *record) { next.ShowWindowAtLogin = !hidden })
}
