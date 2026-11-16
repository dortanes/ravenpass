package vaultservice

import (
	"bytes"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/dortanes/ravenpass/packages/app/ownerauth"
	"github.com/dortanes/ravenpass/packages/app/unlock"
	"github.com/dortanes/ravenpass/packages/app/unlock/unlocktest"
	"github.com/dortanes/ravenpass/packages/vault"
)

const testPIN = "135790"

func readyVault(t *testing.T) (*Service, *memoryKeys) {
	t.Helper()
	service, keys, _ := readyVaultOnDevice(t)
	return service, keys
}

func readyVaultOnDevice(t *testing.T) (*Service, *memoryKeys, *testDevice) {
	t.Helper()
	keys := newMemoryKeys()
	device := newTestDevice()
	service := newTestServiceOn(t, &memoryFiles{}, keys, device)
	createTestVault(t, service)
	return service, keys, device
}

// storedPolicy reads the unlock record of the vault service last opened, as it is stored.
func storedPolicy(t *testing.T, service *Service, keys *memoryKeys) unlock.Policy {
	t.Helper()
	policy, err := unlock.Decode(keys.policy[service.device.policyVault])
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func storedPIN(t *testing.T, service *Service, keys *memoryKeys) *unlock.PIN {
	t.Helper()
	policy := storedPolicy(t, service, keys)
	if policy.PIN == nil {
		t.Fatal("no PIN is stored")
	}
	return policy.PIN
}

func TestEverySetPINBindsANewHardwareKey(t *testing.T) {
	service, keys, device := readyVaultOnDevice(t)
	binding := device.pin
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	first := storedPIN(t, service, keys)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	second := storedPIN(t, service, keys)
	if binding.Created() != 2 {
		t.Fatalf("two PINs made %d hardware keys", binding.Created())
	}
	if bytes.Equal(first.BoundKey, second.BoundKey) || bytes.Equal(first.PeerKey, second.PeerKey) {
		t.Fatal("setting the PIN again kept the earlier binding")
	}
	service.Lock()
	if _, err := service.UnlockWithPIN(testPIN); err != nil {
		t.Fatalf("the PIN did not open the vault through its new binding: %v", err)
	}
	if binding.Created() != 2 {
		t.Fatal("opening the vault with the PIN made a hardware key")
	}
}

func TestAPINTheHardwareRejectsIsNotSet(t *testing.T) {
	service, _, device := readyVaultOnDevice(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	device.pin.Reset()
	methods, err := service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if methods.PINSet || methods.PINAttemptsLeft != 0 {
		t.Fatalf("a PIN the hardware rejects was offered: %+v", methods)
	}
	if _, err := service.UnlockWithPIN(testPIN); !errors.Is(err, unlock.ErrNoPIN) {
		t.Fatalf("a PIN the hardware rejects: got %v, want ErrNoPIN", err)
	}
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatalf("device authentication did not open the vault: %v", err)
	}
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if _, err := service.UnlockWithPIN(testPIN); err != nil {
		t.Fatalf("a PIN set again did not open the vault: %v", err)
	}
}

func TestAPINRejectedAfterItWasReadIsNotSet(t *testing.T) {
	service, _, device := readyVaultOnDevice(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if methods, err := service.UnlockMethods(); err != nil || !methods.PINSet {
		t.Fatalf("the PIN was not offered: %+v, error = %v", methods, err)
	}
	device.pin.Reset()
	if _, err := service.UnlockWithPIN(testPIN); !errors.Is(err, unlock.ErrNoPIN) {
		t.Fatalf("a PIN the hardware rejects: got %v, want ErrNoPIN", err)
	}
	methods, err := service.UnlockMethods()
	if err != nil || methods.PINSet {
		t.Fatalf("a rejected PIN is still offered: %+v, error = %v", methods, err)
	}
}

func TestAHardwareFailureSetsNoPIN(t *testing.T) {
	service, keys, device := readyVaultOnDevice(t)
	before := storedPolicy(t, service, keys)
	failure := errors.New("the Secure Enclave refused")
	device.pin.FailCreate(failure)
	if err := service.SetPIN(openingOf(service), testPIN); !errors.Is(err, failure) {
		t.Fatalf("a failed binding: got %v, want the failure", err)
	}
	if after := storedPolicy(t, service, keys); after.HasPIN() || !bytes.Equal(after.Platform.Envelope, before.Platform.Envelope) {
		t.Fatal("a failed binding changed the unlock record")
	}
	methods, err := service.UnlockMethods()
	if err != nil || methods.PINSet {
		t.Fatalf("a failed binding set a PIN: %+v, error = %v", methods, err)
	}
}

func TestARecordOfAnotherVersionIsRefused(t *testing.T) {
	service, keys := readyVault(t)
	head, err := service.session.Head()
	if err != nil {
		t.Fatal(err)
	}
	service.Lock()
	keys.policy[head.VaultID.String()] = []byte(`{"version":1,"biometry":true}`)
	if _, err := service.UnlockMethods(); !errors.Is(err, unlock.ErrUnsupported) {
		t.Fatalf("unlock methods: got %v, want ErrUnsupported", err)
	}
	if _, err := service.Unlock(testReason); !errors.Is(err, unlock.ErrUnsupported) {
		t.Fatalf("device authentication: got %v, want ErrUnsupported", err)
	}
	if _, err := service.UnlockWithPIN(testPIN); !errors.Is(err, unlock.ErrUnsupported) {
		t.Fatalf("a PIN: got %v, want ErrUnsupported", err)
	}
}

func TestANewVaultOpensWithTheMacsAuthentication(t *testing.T) {
	service, keys, device := readyVaultOnDevice(t)
	if !storedPolicy(t, service, keys).HasPlatform() {
		t.Fatal("creating a vault bound no way in")
	}
	methods, err := service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if !methods.BiometryEnabled || methods.PINSet || !methods.BiometryAvailable {
		t.Fatalf("methods = %+v", methods)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatalf("device authentication did not open the vault: %v", err)
	}
	if !slices.Equal(device.platform.Prompts(), []string{testReason}) {
		t.Fatalf("the owner was asked %q, want one prompt with the reason", device.platform.Prompts())
	}
	if device.platform.Created() != 1 {
		t.Fatal("opening the vault made a hardware key")
	}
}

func TestAVaultWithoutARecordOpensOnlyWithItsRecoveryKey(t *testing.T) {
	service, keys := readyVault(t)
	clear(keys.policy)
	service.Lock()
	methods, err := service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if methods.BiometryEnabled || methods.PINSet {
		t.Fatalf("a vault without a record offered a way in: %+v", methods)
	}
	if _, err := service.Unlock(testReason); !errors.Is(err, unlock.ErrDisabled) {
		t.Fatalf("unlocking without a record: got %v, want ErrDisabled", err)
	}
}

func TestAnOwnerWhoIsNotVerifiedLeavesTheVaultLockedAndCountsNothing(t *testing.T) {
	service, keys, device := readyVaultOnDevice(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if _, err := service.UnlockWithPIN("999999"); !errors.Is(err, unlock.ErrWrongPIN) {
		t.Fatalf("a wrong PIN: got %v, want ErrWrongPIN", err)
	}
	before := bytes.Clone(keys.policy[service.device.policyVault])
	for _, answer := range []error{ownerauth.ErrCanceled, ownerauth.ErrFailed, ownerauth.ErrUnavailable} {
		device.platform.Answer(answer)
		if _, err := service.Unlock(testReason); !errors.Is(err, answer) {
			t.Fatalf("an owner answering %v: got %v", answer, err)
		}
		if service.Unlocked() {
			t.Fatalf("an owner answering %v opened the vault", answer)
		}
		methods, err := service.UnlockMethods()
		if err != nil {
			t.Fatal(err)
		}
		if !methods.BiometryEnabled || methods.PINAttemptsLeft != unlock.MaxPINFailures-1 {
			t.Fatalf("an owner answering %v changed the ways in: %+v", answer, methods)
		}
	}
	if !bytes.Equal(keys.policy[service.device.policyVault], before) {
		t.Fatal("an owner who was not verified changed the unlock record")
	}
	device.platform.Answer(nil)
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatalf("the verified owner did not open the vault: %v", err)
	}
}

func TestCreationBindsExactlyTheChosenWaysIn(t *testing.T) {
	keys := newMemoryKeys()
	device := newTestDevice()
	files := &memoryFiles{}
	service := newTestServiceOn(t, files, keys, device)
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	for name, refused := range map[string]struct {
		methods MethodChoice
		want    error
	}{
		"nothing chosen":         {MethodChoice{}, unlock.ErrNoMethodLeft},
		"a PIN outside the rule": {MethodChoice{PIN: "12345"}, vault.ErrInvalidPIN},
	} {
		if _, err := service.ConfirmCreation(phrase, refused.methods); !errors.Is(err, refused.want) {
			t.Fatalf("%s: got %v, want %v", name, err, refused.want)
		}
	}
	if files.data != nil || len(keys.policy) != 0 || device.platform.Created() != 0 || device.pin.Created() != 0 {
		t.Fatal("a refused choice wrote the vault or made a hardware key")
	}

	if _, err := service.ConfirmCreation(phrase, MethodChoice{PIN: testPIN}); err != nil {
		t.Fatalf("the staged vault did not survive a refused choice: %v", err)
	}
	policy := storedPolicy(t, service, keys)
	if policy.HasPlatform() || !policy.HasPIN() || device.platform.Created() != 0 {
		t.Fatalf("a PIN alone was chosen, the record holds %+v", policy)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); !errors.Is(err, unlock.ErrDisabled) {
		t.Fatalf("device authentication was not chosen: got %v, want ErrDisabled", err)
	}
	if _, err := service.UnlockWithPIN(testPIN); err != nil {
		t.Fatalf("the chosen PIN did not open the vault: %v", err)
	}
}

func TestCreationWithBothWaysInBindsBoth(t *testing.T) {
	keys := newMemoryKeys()
	service := newTestService(t, &memoryFiles{}, keys)
	createTestVaultWith(t, service, MethodChoice{Biometry: true, PIN: testPIN})
	if policy := storedPolicy(t, service, keys); !policy.HasPlatform() || !policy.HasPIN() {
		t.Fatalf("both ways in were chosen, the record holds %+v", policy)
	}
}

func TestANewLocationOffersNoWayInYet(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	methods, err := service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if !methods.BiometryAvailable || methods.BiometryEnabled || methods.PINSet {
		t.Fatalf("a location without a vault reported %+v", methods)
	}
}

func TestAPINOpensTheVaultAndClearsItsWrongAttempts(t *testing.T) {
	service, _ := readyVault(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	methods, err := service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if !methods.PINSet || methods.PINAttemptsLeft != unlock.MaxPINFailures {
		t.Fatalf("methods after setting a PIN = %+v", methods)
	}

	service.Lock()
	if _, err := service.UnlockWithPIN("999999"); !errors.Is(err, unlock.ErrWrongPIN) {
		t.Fatalf("a wrong PIN: got %v, want ErrWrongPIN", err)
	}
	methods, err = service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if methods.PINAttemptsLeft != unlock.MaxPINFailures-1 {
		t.Fatalf("a wrong PIN was not counted: %+v", methods)
	}
	if _, err := service.UnlockWithPIN(testPIN); err != nil {
		t.Fatalf("the recorded PIN did not open the vault: %v", err)
	}
	methods, err = service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if methods.PINAttemptsLeft != unlock.MaxPINFailures {
		t.Fatalf("a correct PIN did not clear the count: %+v", methods)
	}
}

func TestTenWrongPINsRemoveThePIN(t *testing.T) {
	service, _ := readyVault(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	for attempt := 1; attempt < unlock.MaxPINFailures; attempt++ {
		if _, err := service.UnlockWithPIN("999999"); !errors.Is(err, unlock.ErrWrongPIN) {
			t.Fatalf("attempt %d: got %v, want ErrWrongPIN", attempt, err)
		}
	}
	if _, err := service.UnlockWithPIN("999999"); !errors.Is(err, unlock.ErrPINRemoved) {
		t.Fatalf("the last attempt: got %v, want ErrPINRemoved", err)
	}
	methods, err := service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if methods.PINSet {
		t.Fatal("the PIN survived ten wrong attempts")
	}
	if _, err := service.UnlockWithPIN(testPIN); !errors.Is(err, unlock.ErrNoPIN) {
		t.Fatalf("the removed PIN: got %v, want ErrNoPIN", err)
	}
	// Device authentication still opens the vault.
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatalf("the vault did not open after the PIN was removed: %v", err)
	}
}

func TestVerifyPINCountsAttemptsAsUnlockingDoes(t *testing.T) {
	service, _ := readyVault(t)
	if err := service.VerifyPIN(testPIN); !errors.Is(err, unlock.ErrNoPIN) {
		t.Fatalf("a vault without a PIN: got %v, want ErrNoPIN", err)
	}
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyPIN("12345"); !errors.Is(err, vault.ErrInvalidPIN) {
		t.Fatalf("a PIN outside the rule: got %v, want ErrInvalidPIN", err)
	}
	if err := service.VerifyPIN("999999"); !errors.Is(err, unlock.ErrWrongPIN) {
		t.Fatalf("a wrong PIN: got %v, want ErrWrongPIN", err)
	}
	methods, err := service.UnlockMethods()
	if err != nil || methods.PINAttemptsLeft != unlock.MaxPINFailures-1 {
		t.Fatalf("after a wrong PIN: %+v, error = %v", methods, err)
	}
	if err := service.VerifyPIN(testPIN); err != nil {
		t.Fatalf("the right PIN: %v", err)
	}
	methods, err = service.UnlockMethods()
	if err != nil || methods.PINAttemptsLeft != unlock.MaxPINFailures {
		t.Fatalf("a right PIN did not clear the count: %+v, error = %v", methods, err)
	}
	if !service.Unlocked() {
		t.Fatal("checking the PIN closed the vault")
	}
}

func TestVerifyPINRemovesThePINAndLocksAfterTooManyWrongOnes(t *testing.T) {
	service, _ := readyVault(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(0, 0)
	service.pinThrottle = unlock.NewThrottle(func() time.Time { return now })
	for attempt := 1; attempt < unlock.MaxPINFailures; attempt++ {
		if err := service.VerifyPIN("999999"); !errors.Is(err, unlock.ErrWrongPIN) {
			t.Fatalf("attempt %d: got %v, want ErrWrongPIN", attempt, err)
		}
		delay := unlock.AttemptDelay(attempt)
		if delay > 0 {
			now = now.Add(delay - time.Millisecond)
			if err := service.VerifyPIN(testPIN); !errors.Is(err, unlock.ErrTooSoon) {
				t.Fatalf("an attempt %v early after %d wrong PINs: got %v, want ErrTooSoon", time.Millisecond, attempt, err)
			}
		}
		now = now.Add(time.Millisecond)
	}
	if err := service.VerifyPIN("999999"); !errors.Is(err, unlock.ErrPINRemoved) {
		t.Fatalf("the last attempt: got %v, want ErrPINRemoved", err)
	}
	methods, err := service.UnlockMethods()
	if err != nil || methods.PINSet {
		t.Fatalf("the PIN survived its last attempt: %+v, error = %v", methods, err)
	}
	if service.Unlocked() {
		t.Fatal("the vault stayed open after its PIN was removed")
	}
	if err := service.VerifyPIN(testPIN); !errors.Is(err, ErrNotReady) {
		t.Fatalf("the removed PIN: got %v, want ErrNotReady", err)
	}
}

func TestVerifyPINNeedsAnOpenVault(t *testing.T) {
	service, _ := readyVault(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if err := service.VerifyPIN(testPIN); !errors.Is(err, ErrNotReady) {
		t.Fatalf("checking a PIN while locked: got %v, want ErrNotReady", err)
	}
}

func TestOnlyAPINWithinTheRuleIsAccepted(t *testing.T) {
	service, keys := readyVault(t)
	before := bytes.Clone(keys.policy[service.device.policyVault])
	for _, pin := range []string{"", "12345", "1234567890123", "12345!", "123 456"} {
		if err := service.SetPIN(openingOf(service), pin); !errors.Is(err, vault.ErrInvalidPIN) {
			t.Errorf("PIN %q: got %v, want ErrInvalidPIN", pin, err)
		}
	}
	if !bytes.Equal(keys.policy[service.device.policyVault], before) {
		t.Fatal("a refused PIN changed the unlock record")
	}

	for _, pin := range []string{"12345a", "Pass123"} {
		if err := service.SetPIN(openingOf(service), pin); err != nil {
			t.Errorf("PIN %q should be accepted: %v", pin, err)
		}
	}
}

func TestTurningOffTheMacAuthenticationNeedsAPINAndRemovesItsKeys(t *testing.T) {
	service, keys, device := readyVaultOnDevice(t)
	first := storedPolicy(t, service, keys).Platform
	if err := service.SetBiometryUnlock(openingOf(service), false); !errors.Is(err, unlock.ErrNoMethodLeft) {
		t.Fatalf("turning off the only way in: got %v, want ErrNoMethodLeft", err)
	}
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	if err := service.SetBiometryUnlock(openingOf(service), false); err != nil {
		t.Fatal(err)
	}
	if storedPolicy(t, service, keys).HasPlatform() {
		t.Fatal("turning the method off left its bound key and envelope behind")
	}
	service.Lock()
	if _, err := service.Unlock(testReason); !errors.Is(err, unlock.ErrDisabled) {
		t.Fatalf("unlocking with a turned-off method: got %v, want ErrDisabled", err)
	}
	if _, err := service.UnlockWithPIN(testPIN); err != nil {
		t.Fatalf("the PIN did not open the vault: %v", err)
	}
	if err := service.RemovePIN(openingOf(service)); !errors.Is(err, unlock.ErrNoMethodLeft) {
		t.Fatalf("removing the only way in: got %v, want ErrNoMethodLeft", err)
	}
	if err := service.SetBiometryUnlock(openingOf(service), true); err != nil {
		t.Fatal(err)
	}
	second := storedPolicy(t, service, keys).Platform
	if second == nil || device.platform.Created() != 2 || bytes.Equal(first.BoundKey, second.BoundKey) {
		t.Fatal("turning the method back on did not bind a new hardware key")
	}
	if len(device.platform.Prompts()) != 0 {
		t.Fatal("turning the method on asked the owner")
	}
	if err := service.RemovePIN(openingOf(service)); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatalf("device authentication did not open the vault: %v", err)
	}
}

func TestAMacThatCannotAuthenticateIsNotOffered(t *testing.T) {
	device := newTestDevice()
	device.owner.unavailable = true
	service := newTestServiceOn(t, &memoryFiles{}, newMemoryKeys(), device)
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmCreation(phrase, MethodChoice{Biometry: true}); !errors.Is(err, unlock.ErrNotAvailable) {
		t.Fatalf("creating with an unavailable method: got %v, want ErrNotAvailable", err)
	}
	if _, err := service.ConfirmCreation(phrase, MethodChoice{PIN: testPIN}); err != nil {
		t.Fatal(err)
	}
	methods, err := service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if methods.BiometryAvailable {
		t.Fatal("a Mac that cannot authenticate its owner offered the method")
	}
	if err := service.SetBiometryUnlock(openingOf(service), true); !errors.Is(err, unlock.ErrNotAvailable) {
		t.Fatalf("turning on an unavailable method: got %v, want ErrNotAvailable", err)
	}
}

func TestUnlockChangesNeedAnOpenVault(t *testing.T) {
	service, _ := readyVault(t)
	service.Lock()
	if err := service.SetPIN(openingOf(service), testPIN); !errors.Is(err, ErrNotReady) {
		t.Errorf("setting a PIN while locked: got %v, want ErrNotReady", err)
	}
	if err := service.RemovePIN(openingOf(service)); !errors.Is(err, ErrNotReady) {
		t.Errorf("removing a PIN while locked: got %v, want ErrNotReady", err)
	}
	if err := service.SetBiometryUnlock(openingOf(service), false); !errors.Is(err, ErrNotReady) {
		t.Errorf("changing the method while locked: got %v, want ErrNotReady", err)
	}
}

// A PIN unlock and the screens around it read the unlock record once, not once per question.
func TestTheUnlockRecordIsReadOncePerLockedVault(t *testing.T) {
	service, keys := readyVault(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	keys.policyLoads = 0

	if _, err := service.UnlockMethods(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UnlockWithPIN(testPIN); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UnlockMethods(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UnlockMethods(); err != nil {
		t.Fatal(err)
	}
	if keys.policyLoads != 1 {
		t.Fatalf("the unlock record was read %d times, want once", keys.policyLoads)
	}
}

// Using the device authentication key prompts the owner, so a PIN unlock must never touch it.
func TestAPINUnlockNeverAsksTheOwner(t *testing.T) {
	service, _, device := readyVaultOnDevice(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	service.Lock()

	if _, err := service.UnlockMethods(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UnlockWithPIN(testPIN); err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Usage(); err != nil {
		t.Fatal(err)
	}
	if prompts := device.platform.Prompts(); len(prompts) != 0 {
		t.Fatalf("opening with a PIN asked the owner %q", prompts)
	}
}

func TestSwitchingVaultsDoesNotServeTheOtherVaultsRecord(t *testing.T) {
	personal := target("/vaults/personal.rpv")
	work := target("/vaults/work.rpv")
	keys := newMemoryKeys()
	service := newTestService(t, &memoryFiles{target: personal}, keys)
	createTestVault(t, service)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	if err := service.SwitchVault(work); err != nil {
		t.Fatal(err)
	}
	createTestVault(t, service)
	methods, err := service.UnlockMethods()
	if err != nil {
		t.Fatal(err)
	}
	if methods.PINSet {
		t.Fatal("a new vault reported the PIN of the vault before it")
	}
}

func TestTheAttemptDelayGrowsAndIsCapped(t *testing.T) {
	previous := unlock.AttemptDelay(0)
	if previous != 0 {
		t.Fatalf("the first attempt waits %v", previous)
	}
	for failures := 3; failures < unlock.MaxPINFailures; failures++ {
		delay := unlock.AttemptDelay(failures)
		if delay < previous {
			t.Fatalf("the wait shrank at %d wrong attempts: %v after %v", failures, delay, previous)
		}
		previous = delay
	}
	if previous == 0 {
		t.Fatal("wrong attempts never earned a wait")
	}
}

// slowPresence holds each Create until release closes, as slow key hardware does.
type slowPresence struct {
	*unlocktest.PresenceBinding
	started chan struct{}
	release chan struct{}
}

func (p slowPresence) Create(salt []byte) ([]byte, []byte, [unlock.SecretSize]byte, error) {
	close(p.started)
	<-p.release
	return p.PresenceBinding.Create(salt)
}

func TestTheVaultLocksWhileDeviceAuthenticationIsBeingBound(t *testing.T) {
	service, _ := readyVault(t)
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatal(err)
	}
	if err := service.SetBiometryUnlock(openingOf(service), false); err != nil {
		t.Fatal(err)
	}
	slow := slowPresence{unlocktest.NewPresenceBinding(), make(chan struct{}), make(chan struct{})}
	platforms, err := unlock.NewPlatformCredentials(slow)
	if err != nil {
		t.Fatal(err)
	}
	service.platforms = platforms
	ended := make(chan error, 1)
	go func() { ended <- service.SetBiometryUnlock(openingOf(service), true) }()
	<-slow.started
	locked := make(chan struct{})
	go func() {
		service.Lock()
		close(locked)
	}()
	select {
	case <-locked:
	case <-time.After(time.Minute):
		t.Fatal("locking waited for the device authentication key")
	}
	close(slow.release)
	if err := <-ended; !errors.Is(err, ErrNotReady) && !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("binding into a vault that locked meanwhile: got %v, want ErrNotReady or vault.ErrLocked", err)
	}
	methods, err := service.UnlockMethods()
	if err != nil || methods.BiometryEnabled {
		t.Fatalf("a vault locked during binding kept the new way in: %+v, error = %v", methods, err)
	}
}

func TestAChangeBoundToASessionThatClosedChangesNothing(t *testing.T) {
	service, _ := readyVault(t)
	opening := openingOf(service)
	reopenTestVault(t, service)
	if err := service.SetPIN(opening, testPIN); !errors.Is(err, ErrNotReady) {
		t.Fatalf("a PIN for a closed session: got %v, want ErrNotReady", err)
	}
	if err := service.SetBiometryUnlock(opening, false); !errors.Is(err, ErrNotReady) {
		t.Fatalf("device authentication for a closed session: got %v, want ErrNotReady", err)
	}
	if _, err := service.BeginRekey(opening, ""); !errors.Is(err, ErrNotReady) {
		t.Fatalf("a new phrase for a closed session: got %v, want ErrNotReady", err)
	}
	if methods, err := service.UnlockMethods(); err != nil || methods.PINSet {
		t.Fatalf("after a stale change: %+v, error = %v", methods, err)
	}
	if err := service.SetPIN(openingOf(service), testPIN); err != nil {
		t.Fatalf("a PIN for the open session: %v", err)
	}
}

func TestOnlyALockTheServiceRaisesTellsItsObserver(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, _ := readyVault(t)
		var locks atomic.Int32
		service.OnLock(func() { locks.Add(1) })
		service.Lock()
		synctest.Wait()
		if locks.Load() != 0 {
			t.Fatal("a lock the caller asked for told the observer")
		}
		if _, err := service.Unlock(testReason); err != nil {
			t.Fatal(err)
		}
		if err := service.SetPIN(openingOf(service), testPIN); err != nil {
			t.Fatal(err)
		}
		for range unlock.MaxPINFailures {
			if err := service.VerifyPIN("999999"); errors.Is(err, unlock.ErrPINRemoved) {
				break
			}
		}
		synctest.Wait()
		if service.Unlocked() || locks.Load() != 1 {
			t.Fatalf("the PIN removed by wrong attempts: open = %t, observer told %d times", service.Unlocked(), locks.Load())
		}
	})
}
