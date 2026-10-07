package vaultservice

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/app/devicerecords"
	"github.com/dortanes/ravenpass/packages/app/localfile"
	"github.com/dortanes/ravenpass/packages/app/storage"
	"github.com/dortanes/ravenpass/packages/app/unlock"
	"github.com/dortanes/ravenpass/packages/app/unlock/unlocktest"
	"github.com/dortanes/ravenpass/packages/vault"
)

// memoryFiles is an in-memory Storage; data is the current vault and others the unbound known ones.
type memoryFiles struct {
	mu               sync.Mutex
	data             []byte
	failAfterReplace bool
	// maxBytes refuses a larger file before writing it, as the file store does. Zero is no bound.
	maxBytes      int
	target        storage.Target
	vaults        []storage.Target
	others        map[string][]byte
	relocateError error
	// discarded lists the targets DiscardEmpty was asked to clear.
	discarded []storage.Target
	// reading runs within each read of the current vault, after its content is taken, as a slow provider delivers
	// what the file held when the read began.
	reading func()
	// unreadable fails every read of the current vault while set.
	unreadable error
}

func (m *memoryFiles) LoadCiphertext() ([]byte, error) {
	m.mu.Lock()
	data, unreadable, reading := bytes.Clone(m.data), m.unreadable, m.reading
	m.mu.Unlock()
	if reading != nil {
		reading()
	}
	if unreadable != nil {
		return nil, unreadable
	}
	if data == nil {
		return nil, storage.ErrNotFound
	}
	return data, nil
}

func (m *memoryFiles) CommitCiphertext(expected *[32]byte, candidate []byte, finalize func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if expected == nil && m.data != nil || expected != nil && (m.data == nil || sha256.Sum256(m.data) != *expected) {
		return storage.ErrStaleHead
	}
	if m.maxBytes > 0 && len(candidate) > m.maxBytes {
		return storage.ErrTooLarge
	}
	m.data = bytes.Clone(candidate)
	if m.failAfterReplace {
		m.failAfterReplace = false
		return storage.ErrDurabilityUncertain
	}
	return finalize()
}

func (m *memoryFiles) ReconcileCiphertext(expected [32]byte, finalize func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil || sha256.Sum256(m.data) != expected {
		return storage.ErrStaleHead
	}
	return finalize()
}

// seed lists the vault this fake started with, the way a selection record would.
func (m *memoryFiles) seed() {
	if m.others == nil {
		m.others = make(map[string][]byte)
	}
	if len(m.vaults) == 0 {
		m.vaults = []storage.Target{m.target}
	}
}

func (m *memoryFiles) Status() storage.Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seed()
	vaults := m.vaults
	return storage.Status{
		Kinds:      []storage.Kind{storage.LocalFile},
		Current:    m.target,
		Vaults:     vaults,
		Available:  true,
		Restricted: true,
	}
}

func (m *memoryFiles) Open() error { return nil }

// Identify records nothing.
func (m *memoryFiles) Identify(string) error { return nil }

func (m *memoryFiles) Bind(target storage.Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bind(target)
	return nil
}

func (m *memoryFiles) bind(target storage.Target) {
	m.seed()
	if m.target != target {
		if m.data != nil {
			m.others[m.target.Path] = m.data
		}
		m.data = m.others[target.Path]
		delete(m.others, target.Path)
		m.target = target
	}
	if !slices.Contains(m.vaults, target) {
		m.vaults = append(m.vaults, target)
	}
}

func (m *memoryFiles) Forget(target storage.Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seed()
	at := slices.Index(m.vaults, target)
	if at < 0 {
		return storage.ErrUnknownVault
	}
	m.vaults = slices.Delete(m.vaults, at, at+1)
	if target == m.target {
		m.data = nil
		m.target = storage.Target{}
		if len(m.vaults) > 0 {
			m.bind(m.vaults[0])
		}
	}
	return nil
}

func (m *memoryFiles) Erase(target storage.Target) error {
	m.mu.Lock()
	m.seed()
	if target == m.target {
		m.data = nil
	} else {
		delete(m.others, target.Path)
	}
	m.mu.Unlock()
	return m.Forget(target)
}

func (m *memoryFiles) LoadFrom(target storage.Target) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seed()
	data := m.others[target.Path]
	if target == m.target {
		data = m.data
	}
	if data == nil {
		return nil, storage.ErrNotFound
	}
	return bytes.Clone(data), nil
}

func (m *memoryFiles) Relocate(target storage.Target, write func(storage.Ciphertext) error) (storage.Relocation, error) {
	if m.relocateError != nil {
		return storage.Relocation{}, m.relocateError
	}
	destination := &memoryFiles{target: target}
	if err := write(destination); err != nil {
		return storage.Relocation{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	previous := m.target
	m.data = destination.data
	m.target = target
	return storage.Relocation{Target: target, Previous: previous, PreviousRemoved: true}, nil
}

func (m *memoryFiles) DiscardEmpty(target storage.Target) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.discarded = append(m.discarded, target)
}

// openingOf names the open session of service, or no session while it is locked.
func openingOf(service *Service) Opening {
	opening, _ := service.CurrentOpening()
	return opening
}

type memoryKeys struct {
	witness     map[string][]byte
	usage       map[string][]byte
	export      map[string][]byte
	policy      map[string][]byte
	keys        map[string][]byte
	usageLoads  int
	exportLoads int
	policyLoads int
	// policySaveFailures answers the next unlock record saves in order; a nil entry stores the record.
	policySaveFailures []error
}

func newMemoryKeys() *memoryKeys {
	return &memoryKeys{
		witness: make(map[string][]byte),
		usage:   make(map[string][]byte),
		export:  make(map[string][]byte),
		policy:  make(map[string][]byte),
		keys:    make(map[string][]byte),
	}
}

func (m *memoryKeys) SaveKeyRecord(id string, keys []byte) error {
	m.keys[id] = bytes.Clone(keys)
	return nil
}

func (m *memoryKeys) LoadKeyRecord(id string) ([]byte, error) {
	value, exists := m.keys[id]
	if !exists {
		return nil, devicerecords.ErrNotFound
	}
	return bytes.Clone(value), nil
}

func (m *memoryKeys) DeleteKeyRecord(id string) error {
	delete(m.keys, id)
	return nil
}

func (m *memoryKeys) SaveUnlockPolicy(id string, policy []byte) error {
	if len(m.policySaveFailures) > 0 {
		failure := m.policySaveFailures[0]
		m.policySaveFailures = m.policySaveFailures[1:]
		if failure != nil {
			return failure
		}
	}
	m.policy[id] = bytes.Clone(policy)
	return nil
}

func (m *memoryKeys) LoadUnlockPolicy(id string) ([]byte, error) {
	m.policyLoads++
	value, exists := m.policy[id]
	if !exists {
		return nil, devicerecords.ErrNotFound
	}
	return bytes.Clone(value), nil
}

func (m *memoryKeys) DeleteUnlockPolicy(id string) error {
	delete(m.policy, id)
	return nil
}

func (m *memoryKeys) SaveHeadWitness(id string, witness []byte) error {
	m.witness[id] = bytes.Clone(witness)
	return nil
}

func (m *memoryKeys) LoadHeadWitness(id string) ([]byte, error) {
	value, exists := m.witness[id]
	if !exists {
		return nil, devicerecords.ErrNotFound
	}
	return bytes.Clone(value), nil
}

func (m *memoryKeys) SaveUsageRecord(id string, usage []byte) error {
	m.usage[id] = bytes.Clone(usage)
	return nil
}

func (m *memoryKeys) LoadUsageRecord(id string) ([]byte, error) {
	m.usageLoads++
	value, exists := m.usage[id]
	if !exists {
		return nil, devicerecords.ErrNotFound
	}
	return bytes.Clone(value), nil
}

func (m *memoryKeys) DeleteUsageRecord(id string) error {
	delete(m.usage, id)
	return nil
}

func (m *memoryKeys) SaveExportRecord(id string, record []byte) error {
	m.export[id] = bytes.Clone(record)
	return nil
}

func (m *memoryKeys) LoadExportRecord(id string) ([]byte, error) {
	m.exportLoads++
	value, exists := m.export[id]
	if !exists {
		return nil, devicerecords.ErrNotFound
	}
	return bytes.Clone(value), nil
}

func (m *memoryKeys) DeleteExportRecord(id string) error {
	delete(m.export, id)
	return nil
}

func (m *memoryKeys) DeleteHeadWitness(id string) error {
	delete(m.witness, id)
	return nil
}

// testReason is what the prompt of an unlock in these tests says it is for.
const testReason = "unlock your vault"

// testOwner stands for whether this device can authenticate its owner.
type testOwner struct {
	unavailable bool
}

func (o *testOwner) DeviceOwnerAvailable() bool { return !o.unavailable }

// testDevice is the hardware and owner a test service opens vaults with.
type testDevice struct {
	owner    *testOwner
	pin      *unlocktest.Binding
	platform *unlocktest.PresenceBinding
}

func newTestDevice() *testDevice {
	return &testDevice{owner: &testOwner{}, pin: unlocktest.NewBinding(), platform: unlocktest.NewPresenceBinding()}
}

func (d *testDevice) bindings() Device {
	return Device{Owner: d.owner, PIN: d.pin, Platform: d.platform, PINKey: unlocktest.PINKey}
}

func newTestService(t *testing.T, files Storage, keys *memoryKeys) *Service {
	t.Helper()
	return newTestServiceOn(t, files, keys, newTestDevice())
}

func newTestServiceOn(t *testing.T, files Storage, keys *memoryKeys, device *testDevice) *Service {
	t.Helper()
	service, err := New(files, keys, device.bindings())
	if err != nil {
		t.Fatal(err)
	}
	service.pinThrottle = unlock.NewThrottle(passingClock())
	return service
}

// passingClock moves an hour forward on every read, past any PIN attempt delay.
func passingClock() func() time.Time {
	now := time.Unix(0, 0)
	return func() time.Time {
		now = now.Add(time.Hour)
		return now
	}
}

// reopenTestVault locks and unlocks so device records written behind the service are read again.
func reopenTestVault(t *testing.T, service *Service) {
	t.Helper()
	service.Lock()
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatal(err)
	}
}

// createTestVault creates a vault that opens with device authentication.
func createTestVault(t *testing.T, service *Service) (string, vault.Head) {
	t.Helper()
	return createTestVaultWith(t, service, MethodChoice{Biometry: true})
}

func createTestVaultWith(t *testing.T, service *Service, methods MethodChoice) (string, vault.Head) {
	t.Helper()
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	head, err := service.ConfirmCreation(phrase, methods)
	if err != nil {
		t.Fatal(err)
	}
	return phrase, head
}

func TestWizardRequiresThePhraseAndBindsTheMacsAuthentication(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	device := newTestDevice()
	service := newTestServiceOn(t, files, keys, device)
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if files.data != nil {
		t.Fatal("setup wrote vault data before phrase confirmation")
	}
	state, err := service.State()
	if err != nil || state.Phase != PhaseCreating {
		t.Fatalf("unexpected setup state: %+v, %v", state, err)
	}
	if _, err := service.ConfirmCreation("wrong words", MethodChoice{Biometry: true}); !errors.Is(err, vault.ErrInvalidPhrase) {
		t.Fatalf("wrong phrase error = %v", err)
	}
	if files.data != nil {
		t.Fatal("failed phrase confirmation wrote vault data")
	}
	if device.platform.Created() != 0 || len(keys.policy) != 0 {
		t.Fatal("a failed phrase confirmation bound device authentication")
	}
	head, err := service.ConfirmCreation(phrase, MethodChoice{Biometry: true})
	if err != nil {
		t.Fatal(err)
	}
	if device.platform.Created() != 1 || len(device.platform.Prompts()) != 0 {
		t.Fatalf("creation made %d hardware keys and showed %d prompts", device.platform.Created(), len(device.platform.Prompts()))
	}
	policy, err := unlock.Decode(keys.policy[head.VaultID.String()])
	if err != nil || !policy.HasPlatform() || policy.HasPIN() {
		t.Fatalf("the new vault's record = %+v, error = %v", policy, err)
	}
	if !bytes.Equal(files.data, mustExport(t, service)) {
		t.Fatal("committed vault differs from session export")
	}
	if sha256.Sum256(files.data) != head.Hash {
		t.Fatal("saved vault hash differs from returned head")
	}
	state, err = service.State()
	if err != nil || state.Phase != PhaseReady {
		t.Fatalf("unexpected ready state: %+v, %v", state, err)
	}
}

func TestCreationFailsClosedIfTheHardwareRefusesTheBinding(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	device := newTestDevice()
	failure := errors.New("the Secure Enclave refused")
	device.platform.FailCreate(failure)
	service := newTestServiceOn(t, files, keys, device)
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmCreation(phrase, MethodChoice{Biometry: true}); !errors.Is(err, failure) || !errors.Is(err, ErrWayInNotSet) {
		t.Fatalf("confirmation error = %v", err)
	}
	if files.data != nil || len(keys.policy) != 0 {
		t.Fatal("a vault was created without a way to open it on this device")
	}
	if _, err := service.List(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("list after failed setup = %v", err)
	}
	device.platform.FailCreate(nil)
	if _, err := service.ConfirmCreation(phrase, MethodChoice{Biometry: true}); err != nil {
		t.Fatalf("the recovery key already shown no longer creates the vault: %v", err)
	}
}

func TestCredentialWritesRoundTripAcrossLock(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	service := newTestService(t, files, keys)
	createTestVault(t, service)
	input := vault.CredentialInput{Label: "Personal", Websites: []string{"https://example.com"}, Login: "alice", Email: "alice@example.com", Password: "a secret\nwith spaces"}
	id, err := service.CreateCredential(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(files.data, []byte(input.Password)) {
		t.Fatal("plaintext password appeared in the vault file")
	}
	selection, err := service.Select(id)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := service.ReadSelected(selection)
	if err != nil || !reflect.DeepEqual(credential.CredentialInput, input) {
		t.Fatalf("credential = %+v, error = %v", credential, err)
	}
	newPassword := "replacement 🔑"
	if err := service.EditCredential(id, vault.CredentialPatch{Password: &newPassword}); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatal(err)
	}
	selection, err = service.Select(id)
	if err != nil {
		t.Fatal(err)
	}
	credential, err = service.ReadSelected(selection)
	if err != nil || credential.Password != newPassword {
		t.Fatalf("restored password = %q, error = %v", credential.Password, err)
	}
	if err := service.DeleteItem(id); err != nil {
		t.Fatal(err)
	}
	entries, err := service.List()
	if err != nil || len(entries) != 0 {
		t.Fatalf("entries after delete = %+v, error = %v", entries, err)
	}
}

func TestUncertainWriteLocksUntilWitnessReconciliation(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	service := newTestService(t, files, keys)
	createTestVault(t, service)
	files.failAfterReplace = true
	if _, err := service.CreateCredential(vault.CredentialInput{Label: "saved after ambiguous write", Password: "secret"}, nil); !errors.Is(err, storage.ErrDurabilityUncertain) {
		t.Fatalf("uncertain write error = %v", err)
	}
	if _, err := service.List(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("session stayed ready after uncertain write: %v", err)
	}
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatalf("linked successor could not be reconciled: %v", err)
	}
	entries, err := service.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("reconciled entries = %+v, error = %v", entries, err)
	}
}

func TestRecoveryAfterTheMacsKeyWasLostBindsTheChosenWayAnew(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	device := newTestDevice()
	service := newTestServiceOn(t, files, keys, device)
	phrase, head := createTestVault(t, service)
	service.Lock()
	device.platform.Reset()
	if _, err := service.Unlock(testReason); !errors.Is(err, unlock.ErrUnbound) {
		t.Fatalf("unlock with a lost hardware key = %v, want ErrUnbound", err)
	}
	if methods, err := service.UnlockMethods(); err != nil || methods.BiometryEnabled {
		t.Fatalf("a lost hardware key is still offered: %+v, error = %v", methods, err)
	}
	preview, err := service.BeginRecovery(phrase)
	if err != nil || preview.MayLoseNewerCredentials || !preview.NeedsWayIn {
		t.Fatalf("recovery preview = %+v, error = %v", preview, err)
	}
	if _, err := service.ConfirmRecovery(false, MethodChoice{}); !errors.Is(err, unlock.ErrNoMethodLeft) {
		t.Fatalf("keeping a lost way in: got %v, want ErrNoMethodLeft", err)
	}
	recovered, err := service.ConfirmRecovery(false, MethodChoice{Biometry: true})
	if err != nil || recovered != head {
		t.Fatalf("recovery head = %+v, error = %v", recovered, err)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatalf("the recovered vault does not open with device authentication: %v", err)
	}
}

func TestRecoveryOnAMacWithoutAWayInTakesTheChosenWaysOnly(t *testing.T) {
	source := newTestService(t, &memoryFiles{}, newMemoryKeys())
	phrase, head := createTestVault(t, source)
	copied := mustExport(t, source)

	keys := newMemoryKeys()
	device := newTestDevice()
	files := &memoryFiles{data: copied}
	service := newTestServiceOn(t, files, keys, device)
	preview, err := service.BeginRecovery(phrase)
	if err != nil || !preview.NeedsWayIn {
		t.Fatalf("recovery preview on a Mac without a way in = %+v, error = %v", preview, err)
	}
	if _, err := service.ConfirmRecovery(true, MethodChoice{}); !errors.Is(err, unlock.ErrNoMethodLeft) {
		t.Fatalf("recovering without a choice: got %v, want ErrNoMethodLeft", err)
	}
	if device.platform.Created() != 0 || len(keys.policy) != 0 {
		t.Fatal("a refused recovery bound a way in")
	}
	recovered, err := service.ConfirmRecovery(true, MethodChoice{PIN: testPIN})
	if err != nil || recovered != head {
		t.Fatalf("recovery with a PIN = %+v, error = %v", recovered, err)
	}
	if policy := storedPolicy(t, service, keys); !policy.HasPIN() || policy.HasPlatform() || device.platform.Created() != 0 {
		t.Fatalf("a PIN alone was chosen, the record holds %+v", policy)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); !errors.Is(err, unlock.ErrDisabled) {
		t.Fatalf("device authentication was not chosen: got %v, want ErrDisabled", err)
	}
	if _, err := service.UnlockWithPIN(testPIN); err != nil {
		t.Fatalf("the chosen PIN did not open the recovered vault: %v", err)
	}
}

func TestRecoveryOfOlderAuthenticatedCopyRequiresDataLossConfirmation(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	service := newTestService(t, files, keys)
	phrase, first := createTestVault(t, service)
	olderCopy := bytes.Clone(files.data)
	if _, err := service.CreateCredential(vault.CredentialInput{Label: "newer", Password: "password"}, nil); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	files.data = olderCopy
	if _, err := service.Unlock(testReason); !errors.Is(err, ErrOlderCopy) || !errors.Is(err, vault.ErrWitnessOlder) {
		t.Fatalf("ordinary unlock accepted rollback: %v", err)
	}
	preview, err := service.BeginRecovery(phrase)
	if err != nil || !preview.MayLoseNewerCredentials {
		t.Fatalf("rollback preview = %+v, error = %v", preview, err)
	}
	if _, err := service.ConfirmRecovery(false, MethodChoice{}); !errors.Is(err, ErrConfirmationNeeded) {
		t.Fatalf("unconfirmed rollback error = %v", err)
	}
	if _, err := service.List(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("unconfirmed rollback exposed vault: %v", err)
	}
	recovered, err := service.ConfirmRecovery(true, MethodChoice{})
	if err != nil || recovered != first {
		t.Fatalf("confirmed rollback head = %+v, error = %v", recovered, err)
	}
}

func newFileStorage(t *testing.T, vaultPath string) *storage.Manager {
	t.Helper()
	manager, err := storage.NewManager(
		filepath.Join(t.TempDir(), "storage.json"),
		storage.Target{Kind: storage.LocalFile, Path: vaultPath},
		256<<20,
		localfile.Backend{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Open(); err != nil {
		t.Fatal(err)
	}
	return manager
}

func TestRealLocalFileRoundTrip(t *testing.T) {
	files := newFileStorage(t, filepath.Join(t.TempDir(), localfile.DefaultVaultName))
	keys := newMemoryKeys()
	device := newTestDevice()
	first := newTestServiceOn(t, files, keys, device)
	createTestVault(t, first)
	if _, err := first.CreateCredential(vault.CredentialInput{Label: "local", Password: "encrypted on disk"}, nil); err != nil {
		t.Fatal(err)
	}
	first.Lock()
	second := newTestServiceOn(t, files, keys, device)
	if _, err := second.Unlock(testReason); err != nil {
		t.Fatal(err)
	}
	entries, err := second.List()
	if err != nil || len(entries) != 1 || entries[0].Label != "local" {
		t.Fatalf("reopened local entries = %+v, error = %v", entries, err)
	}
}

func TestExportRejectsAnotherInstanceNewerCommit(t *testing.T) {
	vaultPath := filepath.Join(t.TempDir(), localfile.DefaultVaultName)
	filesA := newFileStorage(t, vaultPath)
	filesB := newFileStorage(t, vaultPath)
	keys := newMemoryKeys()
	device := newTestDevice()
	first := newTestServiceOn(t, filesA, keys, device)
	createTestVault(t, first)
	second := newTestServiceOn(t, filesB, keys, device)
	if _, err := second.Unlock(testReason); err != nil {
		t.Fatal(err)
	}
	if _, err := second.CreateCredential(vault.CredentialInput{Label: "newer", Password: "different"}, nil); err != nil {
		t.Fatal(err)
	}
	if data, _, err := first.Export(); !errors.Is(err, ErrStaleExport) || len(data) != 0 {
		t.Fatalf("stale export returned %d bytes, error = %v", len(data), err)
	}
	if _, err := first.List(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("stale instance remained unlocked: %v", err)
	}
	if _, err := first.Unlock(testReason); err != nil {
		t.Fatalf("stale instance could not reopen current vault: %v", err)
	}
	if _, _, err := first.Export(); err != nil {
		t.Fatalf("current export failed: %v", err)
	}
}

func TestExportRejectsWitnessDrift(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	service := newTestService(t, files, keys)
	_, head := createTestVault(t, service)
	witness := keys.witness[head.VaultID.String()]
	witness[len(witness)-1] ^= 1
	if data, _, err := service.Export(); !errors.Is(err, ErrStaleExport) || len(data) != 0 {
		t.Fatalf("export with changed witness returned %d bytes, error = %v", len(data), err)
	}
	if _, err := service.List(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("session remained unlocked after witness drift: %v", err)
	}
}

func TestFailedKeyReplacementLeavesPhraseRecoveryAvailable(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	device := newTestDevice()
	service := newTestServiceOn(t, files, keys, device)
	phrase, _ := createTestVault(t, service)
	service.Lock()
	before := bytes.Clone(files.data)
	if _, err := service.BeginRecovery(phrase); err != nil {
		t.Fatal(err)
	}
	device.platform.FailCreate(errors.New("injected binding failure"))
	if _, err := service.ConfirmRecovery(false, MethodChoice{}); err == nil {
		t.Fatal("key replacement failure was ignored")
	}
	if _, err := service.List(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("failed recovery exposed session: %v", err)
	}
	if !bytes.Equal(files.data, before) {
		t.Fatal("failed recovery changed ciphertext")
	}
	device.platform.FailCreate(nil)
	if _, err := service.BeginRecovery(phrase); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmRecovery(false, MethodChoice{}); err != nil {
		t.Fatalf("phrase could not recover after partial key replacement: %v", err)
	}
}

func TestPinnedStateRoundTripsAcrossLock(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	service := newTestService(t, files, keys)
	createTestVault(t, service)
	id, err := service.CreateCredential(vault.CredentialInput{Label: "Mail", Login: "alice", Password: "secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := service.List()
	if err != nil || len(entries) != 1 || entries[0].Pinned {
		t.Fatalf("new credential entries = %+v, error = %v", entries, err)
	}
	if err := service.SetPinned(id, true); err != nil {
		t.Fatal(err)
	}
	service.Lock()
	if _, err := service.Unlock(testReason); err != nil {
		t.Fatal(err)
	}
	entries, err = service.List()
	if err != nil || len(entries) != 1 || !entries[0].Pinned || entries[0].Detail != "alice" {
		t.Fatalf("reopened entries = %+v, error = %v", entries, err)
	}
	if err := service.SetPinned(id, false); err != nil {
		t.Fatal(err)
	}
	entries, err = service.List()
	if err != nil || len(entries) != 1 || entries[0].Pinned {
		t.Fatalf("unpinned entries = %+v, error = %v", entries, err)
	}
	if err := service.SetPinned(vault.ID{}, true); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("unknown credential pin error = %v", err)
	}
	service.Lock()
	if err := service.SetPinned(id, true); !errors.Is(err, ErrNotReady) {
		t.Fatalf("locked pin error = %v", err)
	}
}

func TestRecoveryClearsDeviceUsageAndExportRecords(t *testing.T) {
	files := &memoryFiles{}
	keys := newMemoryKeys()
	service := newTestService(t, files, keys)
	phrase, head := createTestVault(t, service)
	id, err := service.CreateCredential(vault.CredentialInput{Label: "Mail", Password: "secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.MarkUsed(id); err != nil {
		t.Fatal(err)
	}
	data, exported, err := service.Export()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RecordExport(exported, sha256.Sum256(data)); err != nil {
		t.Fatal(err)
	}
	name := head.VaultID.String()
	if len(keys.usage[name]) == 0 || len(keys.export[name]) == 0 {
		t.Fatal("device usage or export record was not stored")
	}
	service.Lock()
	if _, err := service.BeginRecovery(phrase); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmRecovery(false, MethodChoice{}); err != nil {
		t.Fatal(err)
	}
	if _, exists := keys.usage[name]; exists {
		t.Fatal("device usage record survived recovery")
	}
	if _, exists := keys.export[name]; exists {
		t.Fatal("export record survived recovery")
	}
	state, err := service.ExportState()
	if err != nil || state != ExportUnknown {
		t.Fatalf("export state after recovery = %v, error = %v", state, err)
	}
}

func mustExport(t *testing.T, service *Service) []byte {
	t.Helper()
	data, _, err := service.Export()
	if err != nil {
		t.Fatal(err)
	}
	return data
}
