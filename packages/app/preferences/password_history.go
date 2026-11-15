package preferences

import (
	"errors"
	"slices"
)

// PasswordHistory keeps the passwords the generator handed out for Days, when Enabled.
type PasswordHistory struct {
	Enabled bool
	Days    int
}

const defaultHistoryDays = 30

var historyDays = []int{7, 30, 90, 365}

// ErrUnsupportedHistoryDays reports a history period this build does not offer.
var ErrUnsupportedHistoryDays = errors.New("unsupported password history period")

// HistoryDays lists the periods the user may choose, shortest first.
func HistoryDays() []int {
	return slices.Clone(historyDays)
}

func offeredHistoryDays(days int) bool {
	return slices.Contains(historyDays, days)
}

// PasswordHistory reports the recorded choice: on for 30 days where none was recorded.
func (s *Store) PasswordHistory() PasswordHistory {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	history := PasswordHistory{Enabled: !s.current.HistoryOff, Days: defaultHistoryDays}
	if s.current.HistoryDays != 0 {
		history.Days = s.current.HistoryDays
	}
	return history
}

// SetPasswordHistory records whether the history is kept and its period, which is kept while off.
func (s *Store) SetPasswordHistory(history PasswordHistory) error {
	if !offeredHistoryDays(history.Days) {
		return ErrUnsupportedHistoryDays
	}
	return s.update(func(next *record) {
		next.HistoryOff = !history.Enabled
		next.HistoryDays = history.Days
	})
}
