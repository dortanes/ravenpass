package preferences

import (
	"os"
	"path/filepath"
	"testing"
)

func TestALaunchAtLoginStaysHiddenUntilTheUserChoosesTheWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if !newStore(t, path).HiddenAtLogin() {
		t.Fatal("a launch at login shows the window without a record")
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"language":"ru"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if !newStore(t, path).HiddenAtLogin() {
		t.Fatal("a launch at login shows the window for a record that does not name the choice")
	}
}

func TestTheLoginChoiceSurvivesOtherChangesAndAReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if err := store.SetHiddenAtLogin(false); err != nil {
		t.Fatal(err)
	}
	if err := store.SetSiteIcons(false); err != nil {
		t.Fatal(err)
	}
	reopened := newStore(t, path)
	if reopened.HiddenAtLogin() {
		t.Fatal("the login choice was lost after a restart")
	}
	if err := reopened.SetHiddenAtLogin(true); err != nil {
		t.Fatal(err)
	}
	if !newStore(t, path).HiddenAtLogin() {
		t.Fatal("hiding at login again did not survive a restart")
	}
}
