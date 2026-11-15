package preferences

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPasswordHistoryDefaultsToThirtyDaysAndSurvivesAReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if got := store.PasswordHistory(); got != (PasswordHistory{Enabled: true, Days: 30}) {
		t.Fatalf("history without a record: %+v", got)
	}
	if err := store.SetPasswordHistory(PasswordHistory{Enabled: false, Days: 90}); err != nil {
		t.Fatal(err)
	}
	if got := newStore(t, path).PasswordHistory(); got != (PasswordHistory{Enabled: false, Days: 90}) {
		t.Fatalf("history after a restart: %+v", got)
	}
	if err := store.SetPasswordHistory(PasswordHistory{Enabled: true, Days: 14}); !errors.Is(err, ErrUnsupportedHistoryDays) {
		t.Fatalf("a period not offered: %v", err)
	}
}

func TestAHistoryTurnedOffByAnEarlierVersionStaysOff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	record := `{"version":1,"generatorHistoryOff":true,"generatorHistoryDays":13}`
	if err := os.WriteFile(path, []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := newStore(t, path).PasswordHistory(); got != (PasswordHistory{Enabled: false, Days: 30}) {
		t.Fatalf("history from an earlier record: %+v", got)
	}
}
