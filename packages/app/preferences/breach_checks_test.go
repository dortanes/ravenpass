package preferences

import (
	"path/filepath"
	"testing"
)

func TestBreachChecksDefaultToOffAndSurviveAReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if store.BreachChecks() {
		t.Fatal("breach checks without a record are on")
	}
	if err := store.SetBreachChecks(true); err != nil {
		t.Fatal(err)
	}
	if err := store.SetSiteIcons(false); err != nil {
		t.Fatal(err)
	}
	if !newStore(t, path).BreachChecks() {
		t.Fatal("breach checks turned on came back off after a restart")
	}
}
