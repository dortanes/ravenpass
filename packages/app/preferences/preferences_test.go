package preferences

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/app/messages"
)

// devicePrefers is a device whose owner prefers tags, most preferred first.
func devicePrefers(tags ...string) DeviceLanguages {
	return func() []string { return tags }
}

func newStore(t *testing.T, path string) *Store {
	t.Helper()
	return newStoreOn(t, path, devicePrefers())
}

func newStoreOn(t *testing.T, path string, device DeviceLanguages) *Store {
	t.Helper()
	store, err := New(path, device)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestNewNeedsThePathAndTheDevice(t *testing.T) {
	if _, err := New("", devicePrefers()); err == nil {
		t.Fatal("a store without a path was made")
	}
	if _, err := New(filepath.Join(t.TempDir(), "preferences.json"), nil); err == nil {
		t.Fatal("a store without device languages was made")
	}
}

func TestTheFirstOfferedDeviceLanguageIsFollowedUntilOneIsChosen(t *testing.T) {
	for _, tc := range []struct {
		name      string
		preferred []string
		want      Language
	}{
		{"none", nil, English},
		{"region", []string{"ru-RU"}, Russian},
		{"first offered", []string{"de-DE", "ru-RU", "en-US"}, Russian},
		{"order kept", []string{"en-GB", "ru"}, English},
		{"script and region", []string{"zh-Hans-CN"}, English},
		{"case and underscore", []string{"RU_ru"}, Russian},
		{"unreadable skipped", []string{"", "not a tag", "ru-RU"}, Russian},
		{"region without a base", []string{"und-RU"}, English},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newStoreOn(t, filepath.Join(t.TempDir(), "preferences.json"), devicePrefers(tc.preferred...))
			if language, chosen := store.Language(); language != tc.want || chosen {
				t.Fatalf("language for %q = %q, chosen = %t; want %q, not chosen", tc.preferred, language, chosen, tc.want)
			}
		})
	}
}

func TestTheDeviceLanguageReachesTheDialogsAndGivesWayToAChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	preferred := []string{"ru-RU"}
	store := newStoreOn(t, path, func() []string { return preferred })
	if store.Dialogs().UnlockVault != "разблокировать хранилище" {
		t.Fatal("dialogs did not follow the device's language")
	}
	if store.RecoveryFile().Intro != "Ключ восстановления Ravenpass\n\n" {
		t.Fatal("the recovery key file did not follow the device's language")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("following the device wrote a record: %v", err)
	}

	preferred = []string{"en-US"}
	if language, chosen := store.Language(); language != English || chosen {
		t.Fatalf("language after the device changed = %q, chosen = %t", language, chosen)
	}

	preferred = []string{"ru-RU"}
	if err := store.SetLanguage(English); err != nil {
		t.Fatal(err)
	}
	if language, chosen := store.Language(); language != English || !chosen {
		t.Fatalf("language after choosing = %q, chosen = %t", language, chosen)
	}
	reopened := newStoreOn(t, path, devicePrefers("ru-RU"))
	if language, chosen := reopened.Language(); language != English || !chosen {
		t.Fatalf("language after a restart = %q, chosen = %t", language, chosen)
	}
}

func TestLanguageDefaultsToEnglishUntilChosen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	language, chosen := store.Language()
	if language != English || chosen {
		t.Fatalf("language without a record = %q, chosen = %t", language, chosen)
	}
	if store.Dialogs().SelectVaultFile != "Select a vault file" {
		t.Fatal("dialogs without a record are not English")
	}
}

func TestLanguageRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if err := store.SetLanguage(Russian); err != nil {
		t.Fatal(err)
	}
	language, chosen := store.Language()
	if language != Russian || !chosen {
		t.Fatalf("language after choosing = %q, chosen = %t", language, chosen)
	}
	if store.Dialogs().SelectVaultFile != "Выберите файл хранилища" {
		t.Fatal("dialogs did not follow the chosen language")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("preferences mode: got %04o, want 0600", got)
	}

	reopened := newStore(t, path)
	if language, chosen = reopened.Language(); language != Russian || !chosen {
		t.Fatalf("language after a restart = %q, chosen = %t", language, chosen)
	}
}

func TestUnsupportedAndMalformedRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if err := store.SetLanguage("de"); !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("unsupported language: got %v, want ErrUnsupportedLanguage", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused language wrote a record: %v", err)
	}

	for _, content := range []string{"", "{", `{"version":9,"language":"ru"}`, `{"version":1,"language":"de"}`} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		malformed := newStore(t, path)
		if language, chosen := malformed.Language(); language != English || chosen {
			t.Fatalf("record %q = %q, chosen = %t", content, language, chosen)
		}
	}
}

func TestDefaultGroupSurvivesALanguageChangeAndAReload(t *testing.T) {
	const group = "0123456789abcdef0123456789abcdef"
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if got := store.DefaultGroup(); got != "" {
		t.Fatalf("default group without a record = %q", got)
	}
	if err := store.SetDefaultGroup(group); err != nil {
		t.Fatal(err)
	}
	if err := store.SetLanguage(Russian); err != nil {
		t.Fatal(err)
	}
	if got := store.DefaultGroup(); got != group {
		t.Fatalf("default group after a language change = %q", got)
	}

	reopened := newStore(t, path)
	if got := reopened.DefaultGroup(); got != group {
		t.Fatalf("default group after a restart = %q", got)
	}
	if language, chosen := reopened.Language(); language != Russian || !chosen {
		t.Fatalf("language beside a default group = %q, chosen = %t", language, chosen)
	}
}

func TestEmptyDefaultGroupClearsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if err := store.SetLanguage(Russian); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDefaultGroup("0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDefaultGroup(""); err != nil {
		t.Fatal(err)
	}
	if got := store.DefaultGroup(); got != "" {
		t.Fatalf("default group after clearing it = %q", got)
	}

	reopened := newStore(t, path)
	if got := reopened.DefaultGroup(); got != "" {
		t.Fatalf("default group after a restart = %q", got)
	}
	if language, chosen := reopened.Language(); language != Russian || !chosen {
		t.Fatalf("clearing the default group changed the language to %q, chosen = %t", language, chosen)
	}
}

// A recorded default group leaves the language unchosen.
func TestRecordingADefaultGroupAloneLeavesTheLanguageUnchosen(t *testing.T) {
	const group = "0123456789abcdef0123456789abcdef"
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if err := store.SetDefaultGroup(group); err != nil {
		t.Fatal(err)
	}
	if language, chosen := store.Language(); language != English || chosen {
		t.Fatalf("language after recording a default group = %q, chosen = %t", language, chosen)
	}

	reopened := newStore(t, path)
	if language, chosen := reopened.Language(); language != English || chosen {
		t.Fatalf("language after a restart = %q, chosen = %t", language, chosen)
	}
	if got := reopened.DefaultGroup(); got != group {
		t.Fatalf("default group after a restart = %q", got)
	}

	if err := reopened.SetLanguage(Russian); err != nil {
		t.Fatal(err)
	}
	chosenStore := newStore(t, path)
	if language, chosen := chosenStore.Language(); language != Russian || !chosen {
		t.Fatalf("language after the user chose one = %q, chosen = %t", language, chosen)
	}
	if got := chosenStore.DefaultGroup(); got != group {
		t.Fatalf("choosing a language dropped the default group: %q", got)
	}
}

func TestRecordWithoutADefaultGroupKeepsItsLanguage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"language":"ru"}`), 0600); err != nil {
		t.Fatal(err)
	}
	store := newStore(t, path)
	if language, chosen := store.Language(); language != Russian || !chosen {
		t.Fatalf("language of a record without a default group = %q, chosen = %t", language, chosen)
	}
	if got := store.DefaultGroup(); got != "" {
		t.Fatalf("default group of a record without one = %q", got)
	}
}

func TestClipboardClearingDefaultsToOnAfterAMinute(t *testing.T) {
	store := newStore(t, filepath.Join(t.TempDir(), "preferences.json"))
	if got := store.ClipboardClearing(); got != DefaultClipboardClearing() || !got.Enabled || got.After != time.Minute {
		t.Fatalf("clearing without a record = %+v", got)
	}
}

func TestClipboardClearingSurvivesOtherChangesAndAReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	chosen := ClipboardClearing{Enabled: false, After: 30 * time.Second}
	if err := store.SetClipboardClearing(chosen); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDefaultGroup("0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetLanguage(Russian); err != nil {
		t.Fatal(err)
	}
	reopened := newStore(t, path)
	if got := reopened.ClipboardClearing(); got != chosen {
		t.Fatalf("clearing after a restart = %+v, want %+v", got, chosen)
	}
	if language, _ := reopened.Language(); language != Russian {
		t.Fatalf("recording clearing lost the language: %q", language)
	}
}

func TestUnofferedClearDelaysAreRefusedAndIgnored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	for _, after := range []time.Duration{0, 45 * time.Second, 1500 * time.Millisecond, time.Hour} {
		if err := store.SetClipboardClearing(ClipboardClearing{Enabled: true, After: after}); !errors.Is(err, ErrUnsupportedClearDelay) {
			t.Fatalf("delay %v: got %v, want ErrUnsupportedClearDelay", after, err)
		}
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused delay wrote a record: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"clipboardKept":true,"clipboardClearSeconds":7}`), 0600); err != nil {
		t.Fatal(err)
	}
	got := newStore(t, path).ClipboardClearing()
	if got.Enabled || got.After != DefaultClipboardClearing().After {
		t.Fatalf("record with an unoffered delay = %+v", got)
	}
}

func TestSiteIconsDefaultToOn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if !newStore(t, path).SiteIcons() {
		t.Fatal("website icons without a record are off")
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"language":"ru","clipboardKept":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if !newStore(t, path).SiteIcons() {
		t.Fatal("website icons of a record that does not name the choice are off")
	}
}

func TestSiteIconsSurviveOtherChangesAndAReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := newStore(t, path)
	if err := store.SetSiteIcons(false); err != nil {
		t.Fatal(err)
	}
	if err := store.SetClipboardClearing(ClipboardClearing{Enabled: false, After: 30 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetLanguage(Russian); err != nil {
		t.Fatal(err)
	}
	reopened := newStore(t, path)
	if reopened.SiteIcons() {
		t.Fatal("website icons turned off came back on after a restart")
	}
	if got := reopened.ClipboardClearing(); got.Enabled || got.After != 30*time.Second {
		t.Fatalf("recording website icons lost clipboard clearing: %+v", got)
	}
	if err := reopened.SetSiteIcons(true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "siteIconsOff") {
		t.Fatalf("a record with icons on names the choice: %s", data)
	}
	if !newStore(t, path).SiteIcons() {
		t.Fatal("website icons turned back on stayed off after a restart")
	}
}

func TestEveryOfferedLanguageHasDialogs(t *testing.T) {
	for _, language := range Languages() {
		if got := messages.For(string(language)).Language(); got != string(language) {
			t.Fatalf("%q has no message catalog, it reads %q", language, got)
		}
		for _, titled := range []bool{false, true} {
			catalog := dialogsIn(messages.For(string(language)), titled)
			fields := reflect.ValueOf(catalog)
			for i := range fields.NumField() {
				if fields.Field(i).String() == "" {
					t.Fatalf("%q has no %s dialog text", language, fields.Type().Field(i).Name)
				}
			}
			for reason, facts := range map[string][]string{
				catalog.ShareReason("passport.pdf", "Alex", "example.com"): {"passport.pdf", "Alex", "example.com"},
				catalog.SavePasskeyReason("example.com"):                   {"example.com"},
				catalog.SignInReason("example.com", "alex@example.com"):    {"example.com", "alex@example.com"},
				catalog.FillReason("example.com", "alex@example.com"):      {"example.com", "alex@example.com"},
				catalog.FillCardReason("example.com", "Travel Visa"):       {"example.com", "Travel Visa"},
				catalog.DeleteVaultReason("Personal"):                      {"Personal"},
				catalog.RevealReason("Wallet"):                             {"Wallet"},
			} {
				for _, fact := range facts {
					if !strings.Contains(reason, fact) || strings.Contains(reason, "{") {
						t.Fatalf("%q reason %q does not name %s", language, reason, fact)
					}
				}
			}
		}
	}
}

func TestTitledPromptsGiveATitleAndASentence(t *testing.T) {
	for _, language := range Languages() {
		catalog := dialogsIn(messages.For(string(language)), true)
		for _, prompt := range []string{
			catalog.ShareReason("passport.pdf", "Alex", "example.com"), catalog.SavePasskeyReason("example.com"),
			catalog.SignInReason("example.com", "alex"), catalog.FillReason("example.com", "alex"),
			catalog.FillCardReason("example.com", "Travel Visa"), catalog.ChangeUnlockReason(), catalog.CreateVaultReason(),
			catalog.OpenVaultReason(), catalog.DeleteVaultReason("Personal"), catalog.UnlockVault,
			catalog.RevealReason("Wallet"),
		} {
			title, sentence, found := strings.Cut(prompt, "\n")
			if !found || title == "" || strings.ContainsRune(sentence, '\n') || !strings.HasSuffix(sentence, ".") {
				t.Fatalf("%q prompt %q is not a title and a sentence", language, prompt)
			}
		}
	}
	store, err := New(filepath.Join(t.TempDir(), "preferences.json"), devicePrefers(), TitledPrompts())
	if err != nil {
		t.Fatal(err)
	}
	if store.Dialogs().UnlockVault != "Unlock Ravenpass\nConfirm it's you to unlock your vault." {
		t.Fatalf("titled unlock prompt = %q", store.Dialogs().UnlockVault)
	}
}

func TestThePasskeyReasonsNameTheSiteAndAccount(t *testing.T) {
	english := dialogsIn(messages.For("en"), false)
	if reason := english.SavePasskeyReason("example.com"); reason != "save a passkey for example.com" {
		t.Fatalf("save reason = %q", reason)
	}
	if reason := english.SignInReason("example.com", "{site}"); reason != "sign in to example.com as {site} with a passkey" {
		t.Fatalf("sign-in reason = %q", reason)
	}
}

func TestTheShareReasonKeepsNamesAsGiven(t *testing.T) {
	reason := dialogsIn(messages.For("en"), false).ShareReason("{site}.pdf", "{file}", "example.com")
	if reason != "share “{site}.pdf” from {file} with example.com" {
		t.Fatalf("share reason = %q", reason)
	}
}
