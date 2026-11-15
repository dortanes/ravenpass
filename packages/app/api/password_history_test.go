package api

import (
	"testing"

	"github.com/dortanes/ravenpass/packages/app/passwordhistory"
)

func newServiceWithHistory(t *testing.T) *Service {
	t.Helper()
	service := newReadyService(t)
	service.history = passwordhistory.New(t.TempDir(), service.vault, service.preferences.PasswordHistory)
	service.vault.HoldOwnerData(service.history)
	return service
}

func TestTheHistoryKeepsWhatTheGeneratorHandedOutUntilTurnedOff(t *testing.T) {
	service := newServiceWithHistory(t)
	if setting, err := service.GetPasswordHistory(); err != nil || !setting.Enabled || setting.Days != 30 ||
		len(setting.Offered) != 4 {
		t.Fatalf("setting = %+v, %v", setting, err)
	}
	if err := service.RecordGeneratedPassword("Orbit-Velvet7-Canyon", "words"); err != nil {
		t.Fatal(err)
	}
	listed, err := service.PasswordHistory()
	if err != nil || len(listed) != 1 || listed[0].Value != "Orbit-Velvet7-Canyon" || listed[0].Mode != "words" {
		t.Fatalf("history = %+v, %v", listed, err)
	}
	if count, err := service.CountPasswordHistoryPast(7); err != nil || count != 0 {
		t.Fatalf("passwords past 7 days: %d, %v", count, err)
	}
	if err := service.SetPasswordHistory(false, 30); err != nil {
		t.Fatal(err)
	}
	if err := service.RecordGeneratedPassword("Ladder-Mosquito3-Fern", "words"); err != nil {
		t.Fatal(err)
	}
	if listed, err := service.PasswordHistory(); err != nil || len(listed) != 0 {
		t.Fatalf("history after turning it off = %+v, %v", listed, err)
	}
	assertFailure(t, service.SetPasswordHistory(true, 14), failureInvalidItem)
	assertFailure(t, service.RecordGeneratedPassword("secret", "pin"), failureInvalidItem)
}

func TestAHistoryIsAlwaysAListAndAGeneratedPasswordCopiesLikeASecret(t *testing.T) {
	service := newReadyService(t)
	if listed, err := service.PasswordHistory(); err != nil || listed == nil {
		t.Fatalf("history on a host without one = %v, %v", listed, err)
	}
	clipboard := attachPasteboard(service)
	if err := service.CopyGeneratedPassword("x7#Lq2!vR9"); err != nil {
		t.Fatal(err)
	}
	if clipboard.text != "x7#Lq2!vR9" {
		t.Fatalf("clipboard holds %q", clipboard.text)
	}
	if err := service.Lock(); err != nil {
		t.Fatal(err)
	}
	if clipboard.text != "" {
		t.Fatal("locking left the generated password on the clipboard")
	}
}
