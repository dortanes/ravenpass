package vault

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type vectorHead struct {
	CompleteFileHex string `json:"completeFileHex"`
}

type vectors struct {
	Phrase       string                `json:"phrase"`
	HeaderHex    string                `json:"headerHex"`
	IndexKeyHex  string                `json:"indexKeyHex"`
	RecordKeyHex string                `json:"recordKeyHex"`
	Heads        map[string]vectorHead `json:"heads"`
}

func loadVectors(t *testing.T) vectors {
	t.Helper()
	data, err := os.ReadFile("testdata/v1_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture vectors
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func decodeHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func selectCredential(t *testing.T, session *Session, id ID) Credential {
	t.Helper()
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	value, err := session.ReadSelected(ticket)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

// wrapFor wraps the vault key of session for deviceKey.
func wrapFor(t *testing.T, session *Session, deviceKey []byte) []byte {
	t.Helper()
	envelope, err := session.WrapDeviceKey(deviceKey)
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

// vaultWith commits one revision per credential, so the returned head sits at 1+len(inputs).
func vaultWith(t *testing.T, inputs ...CredentialInput) (Created, []ID) {
	t.Helper()
	created, err := Create()
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]ID, 0, len(inputs))
	for _, input := range inputs {
		pending, id, err := created.Session.PrepareCreate(input, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := created.Session.Commit(pending); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return created, ids
}

// The fixture comes from an independent implementation of container version 0x0002; never regenerate it from this package.
func TestIndependentVectorsAreRefusedAsUnsupported(t *testing.T) {
	fixture := loadVectors(t)
	for _, name := range []string{"genesis", "successor", "fork"} {
		t.Run(name, func(t *testing.T) {
			container := decodeHex(t, fixture.Heads[name].CompleteFileHex)
			if _, err := parseContainer(container); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("parse got %v", err)
			}
			session, err := OpenWithRecovery(container, fixture.Phrase)
			if session != nil || !errors.Is(err, ErrUnsupported) {
				t.Fatalf("open got session %v, error %v", session, err)
			}
		})
	}
	entropy, err := entropyFromMnemonic(fixture.Phrase)
	if err != nil || len(entropy) != 32 {
		t.Fatalf("the vectors are refused for their container version, not their phrase: %v", err)
	}
	// The generator sealed every vector for vault 000102…0f with this data key; the file carries only keys derived from it.
	vaultID := ID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	dataKey := decodeHex(t, "101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f")
	if !bytes.Equal(headerBytes(vaultID), decodeHex(t, fixture.HeaderHex)) {
		t.Fatal("header encoding differs from the independent vector")
	}
	indexKey, err := deriveKey(dataKey, vaultID, "index")
	if err != nil {
		t.Fatal(err)
	}
	recordKey, err := deriveKey(dataKey, vaultID, "record")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(indexKey[:], decodeHex(t, fixture.IndexKeyHex)) || !bytes.Equal(recordKey[:], decodeHex(t, fixture.RecordKeyHex)) {
		t.Fatal("key derivation differs from the independent vector")
	}
}

// grow extends heads and their ancestries by count revisions whose hashes start with branch.
func grow(vaultID ID, heads []Head, lines []ancestry, count int, branch byte) ([]Head, []ancestry) {
	heads, lines = slices.Clone(heads), slices.Clone(lines)
	for range count {
		head := Head{VaultID: vaultID, Revision: uint64(len(heads) + 1), Hash: [32]byte{branch, byte(len(heads)), byte(len(heads) >> 8)}}
		var line ancestry
		if len(heads) > 0 {
			last := heads[len(heads)-1]
			head.PreviousHash = last.Hash
			line = lines[len(lines)-1].after(last.Hash)
		}
		heads, lines = append(heads, head), append(lines, line)
	}
	return heads, lines
}

func TestWitnessComparison(t *testing.T) {
	vaultID := ID{0xa7}
	heads, lines := grow(vaultID, nil, nil, maxAncestry+3, 1)
	// The fork shares revisions 1 to 4 and branches at revision 5.
	fork, forkLines := grow(vaultID, heads[:4], lines[:4], 2, 2)
	previousOnly := func(i int) ancestry { return ancestry{heads[i].PreviousHash} }
	tests := []struct {
		name    string
		head    Head
		line    ancestry
		witness *Witness
		want    WitnessDecision
		err     error
	}{
		{"the acknowledged version", heads[4], lines[4], ptrWitness(WitnessFor(heads[4])), WitnessEqual, nil},
		{"the next revision", heads[5], lines[5], ptrWitness(WitnessFor(heads[4])), WitnessAdvance, nil},
		{"several revisions ahead", heads[9], lines[9], ptrWitness(WitnessFor(heads[4])), WitnessAdvance, nil},
		{"as far ahead as the ancestry reaches", heads[maxAncestry+1], lines[maxAncestry+1], ptrWitness(WitnessFor(heads[1])), WitnessAdvance, nil},
		{"further ahead than the ancestry reaches", heads[maxAncestry+2], lines[maxAncestry+2], ptrWitness(WitnessFor(heads[1])), 0, ErrWitnessDiverged},
		{"an earlier version", heads[3], lines[3], ptrWitness(WitnessFor(heads[4])), 0, ErrWitnessOlder},
		{"another version at the same revision", fork[4], forkLines[4], ptrWitness(WitnessFor(heads[4])), 0, ErrWitnessDiverged},
		{"a branch ahead of the acknowledged version", fork[5], forkLines[5], ptrWitness(WitnessFor(heads[4])), 0, ErrWitnessDiverged},
		{"a branch that grew from the acknowledged version", fork[5], forkLines[5], ptrWitness(WitnessFor(heads[2])), WitnessAdvance, nil},
		{"a container without ancestry one revision ahead", heads[5], previousOnly(5), ptrWitness(WitnessFor(heads[4])), WitnessAdvance, nil},
		{"a container without ancestry two revisions ahead", heads[6], previousOnly(6), ptrWitness(WitnessFor(heads[4])), 0, ErrWitnessDiverged},
		{"no witness", heads[4], lines[4], nil, 0, ErrWitnessMissing},
		{"another vault's witness", heads[4], lines[4], &Witness{VaultID: ID{0xb8}, Revision: 4, Hash: heads[3].Hash}, 0, ErrWitnessMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision, err := compareWitness(test.head, test.line, test.witness)
			if decision != test.want || !errors.Is(err, test.err) {
				t.Fatalf("got %v, %v; want %v, %v", decision, err, test.want, test.err)
			}
		})
	}
}

func TestWitnessAdvanceAuthenticatesEveryRecord(t *testing.T) {
	deviceKey := bytes.Repeat([]byte{0x11}, 32)
	created, ids := vaultWith(t,
		CredentialInput{Label: "GitHub", Login: "alex", Password: "example-1-DO-NOT-USE"},
		CredentialInput{Label: "Mail", Email: "alex@example.test"},
	)
	session := created.Session
	defer session.Lock()
	envelope := wrapFor(t, session, deviceKey)
	_, genesisHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	password := "rotated"
	edit, err := session.PrepareEdit(ids[0], CredentialPatch{Password: &password})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(edit); err != nil {
		t.Fatal(err)
	}
	successorBytes, successorHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := parseContainer(successorBytes)
	if err != nil {
		t.Fatal(err)
	}
	modifiedRecords := append([]sealedBox(nil), raw.records...)
	modifiedRecords[0].ciphertext = append([]byte(nil), raw.records[0].ciphertext...)
	modifiedRecords[0].ciphertext[len(modifiedRecords[0].ciphertext)-1] ^= 1
	modifiedEntries := append([]entryMeta(nil), session.entries...)
	modifiedEntries[0].digest = sha256.Sum256(encodeBox(modifiedRecords[0]))
	plaintext, err := encodeIndex(successorHead.Revision, session.ancestry, modifiedEntries, nil, DefaultTrashRetention)
	if err != nil {
		t.Fatal(err)
	}
	indexBox, err := seal(session.indexKey, plaintext, indexAAD(raw.vaultID, raw.recovery))
	if err != nil {
		t.Fatal(err)
	}
	modifiedFile, err := encodeContainer(raw.vaultID, raw.recovery, indexBox, modifiedRecords)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenWithRecovery(modifiedFile, created.RecoveryPhrase); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("recovery accepted invalid record: %v", err)
	}
	callbackCalled := false
	deviceSession, divergence, err := OpenWithDevice(modifiedFile, deviceKey, envelope, ptrWitness(WitnessFor(genesisHead)), func(Witness) error {
		callbackCalled = true
		return nil
	})
	if deviceSession != nil || divergence != nil || !errors.Is(err, ErrAuthentication) || callbackCalled {
		t.Fatalf("invalid successor became visible or advanced witness: session %v, divergence %v, error %v, callback %v", deviceSession, divergence, err, callbackCalled)
	}
}

// withRecords is the current container holding records, under an index resealed with their digests so only a record's own authentication can refuse it.
func withRecords(t *testing.T, session *Session, records []sealedBox) []byte {
	t.Helper()
	raw, head := currentRaw(t, session)
	entries := withDigests(slices.Clone(session.entries), records)
	plaintext, err := encodeIndex(head.Revision, session.ancestry, entries, session.groups, session.retention)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := parseIndex(plaintext, records); err != nil {
		t.Fatalf("the resealed index refuses the records: %v", err)
	}
	raw.records = records
	return withIndex(t, session, raw, plaintext)
}

func TestSwappedRecordsAreRefused(t *testing.T) {
	created, _ := vaultWith(t,
		CredentialInput{Label: "GitHub", Login: "alex", Password: "example-1-DO-NOT-USE"},
		CredentialInput{Label: "Mail", Email: "alex@example.test"},
	)
	session := created.Session
	defer session.Lock()
	raw, _ := currentRaw(t, session)
	swapped := []sealedBox{raw.records[1], raw.records[0]}
	stale, err := encodeContainer(raw.vaultID, raw.recovery, raw.index, swapped)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenWithRecovery(stale, created.RecoveryPhrase); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("swapped records under the original index: %v", err)
	}
	if _, err := OpenWithRecovery(withRecords(t, session, swapped), created.RecoveryPhrase); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("swapped records under matching digests: %v", err)
	}
}

func TestReplayedRecordRevisionIsRefused(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "GitHub", Login: "alex", Password: "example-1-DO-NOT-USE"})
	session := created.Session
	defer session.Lock()
	earlier, _ := currentRaw(t, session)
	earlierRevision := session.entries[0].revision
	password := "rotated"
	edit, err := session.PrepareEdit(ids[0], CredentialPatch{Password: &password})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(edit); err != nil {
		t.Fatal(err)
	}
	if session.entries[0].revision == earlierRevision {
		t.Fatal("the edit left the record revision unchanged")
	}
	current, _ := currentRaw(t, session)
	replayed := []sealedBox{earlier.records[0]}
	stale, err := encodeContainer(current.vaultID, current.recovery, current.index, replayed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenWithRecovery(stale, created.RecoveryPhrase); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("replayed record under the current index: %v", err)
	}
	if _, err := OpenWithRecovery(withRecords(t, session, replayed), created.RecoveryPhrase); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("replayed record under a matching digest: %v", err)
	}
}

func TestDeviceOpenRequiresWitnessBeforeExposingSession(t *testing.T) {
	deviceKey := bytes.Repeat([]byte{0x22}, 32)
	created, ids := vaultWith(t, CredentialInput{Label: "Example", Password: "secret"})
	session := created.Session
	defer session.Lock()
	envelope := wrapFor(t, session, deviceKey)
	genesisBytes, genesisHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{"genesis": genesisBytes}
	heads := map[string]Head{"genesis": genesisHead}
	// Two uncommitted edits of one parent give two containers at one revision for a witness to tell apart.
	for _, branch := range []struct{ name, password string }{{"successor", "one"}, {"fork", "two"}} {
		password := branch.password
		pending, err := session.PrepareEdit(ids[0], CredentialPatch{Password: &password})
		if err != nil {
			t.Fatal(err)
		}
		files[branch.name] = pending.Container()
		heads[branch.name] = pending.Head()
		if err := session.Abort(pending); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name    string
		file    string
		witness *Witness
		want    error
	}{
		{"missing witness", "genesis", nil, ErrWitnessMissing},
		{"older file", "genesis", ptrWitness(WitnessFor(heads["successor"])), ErrWitnessOlder},
		{"advance without persistence", "successor", ptrWitness(WitnessFor(heads["genesis"])), ErrWitnessAdvanceRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opened, divergence, err := OpenWithDevice(files[test.file], deviceKey, envelope, test.witness, nil)
			if opened != nil || divergence != nil || !errors.Is(err, test.want) {
				t.Fatalf("device open exposed a session: %v, divergence %v, error %v", opened, divergence, err)
			}
		})
	}
	callbackError := errors.New("witness persistence failed")
	opened, _, err := OpenWithDevice(files["successor"], deviceKey, envelope, ptrWitness(WitnessFor(heads["genesis"])), func(witness Witness) error {
		if witness != WitnessFor(heads["successor"]) {
			t.Fatalf("unexpected witness: %v", witness)
		}
		return callbackError
	})
	if opened != nil || !errors.Is(err, callbackError) {
		t.Fatalf("failed witness persistence exposed a session: %v, error %v", opened, err)
	}
}

func ptrWitness(value Witness) *Witness { return &value }

func TestTamperingAndFormatRejection(t *testing.T) {
	created, _ := vaultWith(t,
		CredentialInput{Label: "GitHub", Login: "alex", Password: "example-1-DO-NOT-USE"},
		CredentialInput{Label: "Mail", Email: "alex@example.test"},
	)
	defer created.Session.Lock()
	base, _, err := created.Session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := parseContainer(base)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		ciphertext []byte
		want       error
	}{
		{"recovery tag", raw.recovery.ciphertext, ErrAuthentication},
		{"index tag", raw.index.ciphertext, ErrAuthentication},
		{"first record tag", raw.records[0].ciphertext, ErrAuthentication},
		{"second record tag", raw.records[1].ciphertext, ErrAuthentication},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modified := append([]byte(nil), base...)
			start := bytes.Index(modified, test.ciphertext)
			if start < 0 {
				t.Fatal("ciphertext not found in container")
			}
			modified[start+len(test.ciphertext)-1] ^= 1
			if _, err := OpenWithRecovery(modified, created.RecoveryPhrase); !errors.Is(err, test.want) {
				t.Fatalf("got %v", err)
			}
		})
	}
	wrongPhrase := strings.Join(append(repeatWord("zoo", 23), "vote"), " ")
	if _, err := OpenWithRecovery(base, wrongPhrase); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("wrong phrase: %v", err)
	}
	unsupported := append([]byte(nil), base...)
	unsupported[9] = 4
	if _, err := OpenWithRecovery(unsupported, created.RecoveryPhrase); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("version: %v", err)
	}
	noncanonical := append(append([]byte(nil), base[:10]...), 0x98, 0x04)
	noncanonical = append(noncanonical, base[11:]...)
	if _, err := OpenWithRecovery(noncanonical, created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
		t.Fatalf("noncanonical: %v", err)
	}
	if _, err := OpenWithRecovery(append(base, 0), created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
		t.Fatalf("trailing bytes: %v", err)
	}
	if _, err := OpenWithRecovery(base[:len(base)-1], created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
		t.Fatalf("truncated: %v", err)
	}
}

func TestInspectUntrustedVaultID(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "Example", Password: "secret"})
	defer created.Session.Lock()
	container, head, err := created.Session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	id, err := InspectUntrustedVaultID(container)
	if err != nil || id != head.VaultID {
		t.Fatalf("header ID: %v", err)
	}
	tampered := append([]byte(nil), container...)
	tampered[len(tampered)-1] ^= 1
	if id, err := InspectUntrustedVaultID(tampered); err != nil || id != head.VaultID {
		t.Fatalf("header ID must remain inspectable without authentication: %v", err)
	}
	if _, err := InspectUntrustedVaultID(container[:len(container)-1]); !errors.Is(err, ErrMalformed) {
		t.Fatalf("truncated container: %v", err)
	}
	noncanonical := append(append([]byte(nil), container[:10]...), 0x98, 0x04)
	noncanonical = append(noncanonical, container[11:]...)
	if _, err := InspectUntrustedVaultID(noncanonical); !errors.Is(err, ErrMalformed) {
		t.Fatalf("noncanonical container: %v", err)
	}
	if _, err := InspectUntrustedVaultID(make([]byte, MaxContainerBytes+1)); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("oversized container: %v", err)
	}
}

func repeatWord(word string, count int) []string {
	words := make([]string, count)
	for i := range words {
		words[i] = word
	}
	return words
}

func TestTransactionsPreserveExactValuesAndCiphertext(t *testing.T) {
	deviceKey := bytes.Repeat([]byte{0x91}, 32)
	created, err := Create()
	if err != nil {
		t.Fatal(err)
	}
	defer created.Session.Lock()
	envelope := wrapFor(t, created.Session, deviceKey)
	if len(strings.Fields(created.RecoveryPhrase)) != 24 {
		t.Fatal("recovery phrase length")
	}
	if recovered, err := OpenWithRecovery(created.Container, created.RecoveryPhrase); err != nil {
		t.Fatal(err)
	} else {
		recovered.Lock()
	}
	createdHead, err := created.Session.Head()
	if err != nil {
		t.Fatal(err)
	}
	if device, _, err := OpenWithDevice(created.Container, deviceKey, envelope, ptrWitness(WitnessFor(createdHead)), nil); err != nil {
		t.Fatal(err)
	} else {
		device.Lock()
	}
	input := CredentialInput{Label: "  École  ", Websites: []string{"https://example.test/путь", " app.example.test "}, Login: "é  ", Email: "CASE@EXAMPLE.TEST", Password: " 🔒paß\n ", Notes: " line 1\nline 2 "}
	pending, id, err := created.Session.PrepareCreate(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if entries, _ := created.Session.List(); len(entries) != 0 {
		t.Fatal("uncommitted item became visible")
	}
	if _, _, err := created.Session.PrepareCreate(input, nil); !errors.Is(err, ErrPendingCommit) {
		t.Fatalf("parallel write: %v", err)
	}
	if err := created.Session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	firstHead, _ := created.Session.Head()
	if firstHead.PreviousHash != sha256.Sum256(created.Container) {
		t.Fatal("broken predecessor")
	}
	got := selectCredential(t, created.Session, id)
	if !reflect.DeepEqual(got.CredentialInput, input) {
		t.Fatalf("values changed: %#v", got.CredentialInput)
	}
	firstBytes, _, _ := created.Session.CurrentContainer()
	firstRaw, err := parseContainer(firstBytes)
	if err != nil {
		t.Fatal(err)
	}
	updatedPassword := "  another 🔑  "
	changed, err := created.Session.PrepareEdit(id, CredentialPatch{Password: &updatedPassword})
	if err != nil {
		t.Fatal(err)
	}
	if err := created.Session.Commit(changed); err != nil {
		t.Fatal(err)
	}
	updated := selectCredential(t, created.Session, id)
	if updated.Password != updatedPassword || !slices.Equal(updated.Websites, input.Websites) || updated.Login != input.Login || updated.Email != input.Email || updated.Notes != input.Notes || updated.Label != input.Label {
		t.Fatal("patch lost unchanged fields")
	}
	secondBytes, _, _ := created.Session.CurrentContainer()
	secondRaw, err := parseContainer(secondBytes)
	if err != nil {
		t.Fatal(err)
	}
	if firstRaw.records[0].nonce == secondRaw.records[0].nonce || firstRaw.index.nonce == secondRaw.index.nonce {
		t.Fatal("nonce reused")
	}
	if firstHead.Revision != 2 {
		t.Fatal("first revision")
	}
	secondHead, _ := created.Session.Head()
	if secondHead.Revision != 3 || secondHead.PreviousHash != firstHead.Hash {
		t.Fatal("second revision")
	}
	deletion, err := created.Session.PrepareDelete(id)
	if err != nil {
		t.Fatal(err)
	}
	if entries, _ := created.Session.List(); len(entries) != 1 {
		t.Fatal("uncommitted deletion became visible")
	}
	if err := created.Session.Abort(deletion); err != nil {
		t.Fatal(err)
	}
	if entries, _ := created.Session.List(); len(entries) != 1 {
		t.Fatal("aborted deletion removed item")
	}
	deletion, err = created.Session.PrepareDelete(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := created.Session.Commit(deletion); err != nil {
		t.Fatal(err)
	}
	if entries, _ := created.Session.List(); len(entries) != 0 {
		t.Fatal("deletion not committed")
	}
}

func TestRevisionOverflowAndDuplicateManifest(t *testing.T) {
	overflow, ids := vaultWith(t, CredentialInput{Label: "Example", Password: "secret"})
	overflow.Session.entries[0].revision = math.MaxUint64
	password := "changed"
	if _, err := overflow.Session.PrepareEdit(ids[0], CredentialPatch{Password: &password}); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("record overflow: %v", err)
	}
	overflow.Session.head.Revision = math.MaxUint64
	if _, _, err := overflow.Session.PrepareCreate(CredentialInput{Label: "next"}, nil); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("vault overflow: %v", err)
	}
	overflow.Session.Lock()

	created, _ := vaultWith(t,
		CredentialInput{Label: "One", Password: "one"},
		CredentialInput{Label: "Two", Password: "two"},
	)
	defer created.Session.Lock()
	base, _, err := created.Session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := parseContainer(base)
	if err != nil {
		t.Fatal(err)
	}
	malformed := []entryMeta{
		{id: ID{1}, revision: 1, label: "one", digest: sha256.Sum256(encodeBox(raw.records[0])), kind: KindCredential},
		{id: ID{1}, revision: 1, label: "two", digest: sha256.Sum256(encodeBox(raw.records[1])), kind: KindCredential},
	}
	plaintext, err := encodeIndex(1, nil, malformed, nil, DefaultTrashRetention)
	if err != nil {
		t.Fatal(err)
	}
	indexBox, err := seal(created.Session.indexKey, plaintext, indexAAD(raw.vaultID, raw.recovery))
	if err != nil {
		t.Fatal(err)
	}
	duplicateFile, err := encodeContainer(raw.vaultID, raw.recovery, indexBox, raw.records)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenWithRecovery(duplicateFile, created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
		t.Fatalf("duplicate IDs: %v", err)
	}
}
