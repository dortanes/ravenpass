package vault

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

// saveCredentials commits one credential per label, each in its own revision.
func saveCredentials(t *testing.T, session *Session, labels ...string) {
	t.Helper()
	for _, label := range labels {
		commitCredential(t, session, CredentialInput{Label: label, Password: "secret"}, nil)
	}
}

// numberedLabels is count labels that name the revision they are saved in.
func numberedLabels(prefix string, count int) []string {
	labels := make([]string, count)
	for i := range labels {
		labels[i] = fmt.Sprintf("%s %d", prefix, i)
	}
	return labels
}

func containerOf(t *testing.T, session *Session) ([]byte, Head) {
	t.Helper()
	container, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	return container, head
}

// recordingWitness persists nothing and keeps each witness it was asked to record.
type recordingWitness struct {
	recorded []Witness
	fail     error
}

func (r *recordingWitness) persist(witness Witness) error {
	r.recorded = append(r.recorded, witness)
	return r.fail
}

func TestEveryCommitRecordsTheRevisionsBeforeIt(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	defer session.Lock()
	_, first := containerOf(t, session)
	if len(session.ancestry) != 0 {
		t.Fatalf("the first revision names %d ancestors", len(session.ancestry))
	}
	hashes := [][32]byte{first.Hash}
	for i, label := range numberedLabels("Saved", maxAncestry+5) {
		saveCredentials(t, session, label)
		_, head := containerOf(t, session)
		if head.Revision != uint64(i+2) {
			t.Fatalf("revision %d after %d saves", head.Revision, i+1)
		}
		want := make(ancestry, 0, maxAncestry)
		for j := len(hashes) - 1; j >= 0 && len(want) < maxAncestry; j-- {
			want = append(want, hashes[j])
		}
		if !reflect.DeepEqual(session.ancestry, want) || head.PreviousHash != hashes[len(hashes)-1] {
			t.Fatalf("revision %d names %d ancestors, want the %d before it", head.Revision, len(session.ancestry), len(want))
		}
		hashes = append(hashes, head.Hash)
	}
	container, head := containerOf(t, session)
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if !reflect.DeepEqual(reopened.ancestry, session.ancestry) || mustHead(t, reopened) != head {
		t.Fatal("the ancestry did not survive a reopen")
	}
}

// earlierIndex is the index of an empty vault at revision with previous and the encoded [earlier…] field.
func earlierIndex(revision uint64, previous [32]byte, earlier ...[]byte) []byte {
	return encodeArray(encodeUint(revision), encodeBytes(previous[:]), encodeArray(), encodeArray(), encodeArray(earlier...))
}

func hashesOf(count int) [][]byte {
	hashes := make([][]byte, count)
	for i := range hashes {
		hashes[i] = encodeBytes(bytes.Repeat([]byte{byte(i + 1)}, 32))
	}
	return hashes
}

func TestAncestryIsAuthenticatedAndBounded(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	defer session.Lock()
	raw, _ := currentRaw(t, session)
	previous := [32]byte{0xaa}

	valid := withIndex(t, session, raw, earlierIndex(3, previous, hashesOf(1)...))
	opened, err := OpenWithRecovery(valid, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	if want := (ancestry{previous, [32]byte(bytes.Repeat([]byte{1}, 32))}); !reflect.DeepEqual(opened.ancestry, want) {
		t.Fatalf("ancestry = %x, want %x", opened.ancestry, want)
	}
	opened.Lock()

	sealed, err := parseContainer(valid)
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Clone(valid)
	tampered[bytes.Index(tampered, sealed.index.ciphertext)+len(sealed.index.ciphertext)-1] ^= 1
	if _, err := OpenWithRecovery(tampered, created.RecoveryPhrase); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("a changed index: got %v, want ErrAuthentication", err)
	}

	tests := []struct {
		name  string
		index []byte
		want  error
	}{
		{"the first revision naming an ancestor", earlierIndex(1, [32]byte{}, hashesOf(1)...), ErrMalformed},
		{"more ancestors than revisions before it", earlierIndex(3, previous, hashesOf(2)...), ErrMalformed},
		{"a short hash", earlierIndex(4, previous, encodeBytes(make([]byte, 31))), ErrMalformed},
		{"more ancestors than the bound", earlierIndex(maxAncestry+10, previous, hashesOf(maxAncestry)...), ErrResourceLimit},
		{"four fields", encodeArray(encodeUint(3), encodeBytes(previous[:]), encodeArray(), encodeArray()), ErrMalformed},
		{"six fields", encodeArray(encodeUint(3), encodeBytes(previous[:]), encodeArray(), encodeArray(), encodeArray(hashesOf(1)...), encodeArray()), ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			container := withIndex(t, session, raw, test.index)
			if _, err := OpenWithRecovery(container, created.RecoveryPhrase); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}

	if _, err := encodeIndex(2, ancestry{{1}, {2}}, nil, nil, DefaultTrashRetention); !errors.Is(err, ErrMalformed) {
		t.Fatalf("writing more ancestors than revisions before it: got %v", err)
	}
	if _, err := encodeIndex(maxAncestry+10, make(ancestry, maxAncestry+1), nil, nil, DefaultTrashRetention); !errors.Is(err, ErrMalformed) {
		t.Fatalf("writing more ancestors than the bound: got %v", err)
	}
}

// twoDevices is one vault open on a device holding a device key and on another opened with the recovery phrase.
type twoDevices struct {
	created   Created
	this      *Session
	other     *Session
	deviceKey []byte
	envelope  []byte
}

func newTwoDevices(t *testing.T) twoDevices {
	t.Helper()
	created, _ := vaultWith(t, CredentialInput{Label: "Shared", Password: "secret"})
	deviceKey := bytes.Repeat([]byte{0x5c}, 32)
	envelope := wrapFor(t, created.Session, deviceKey)
	container, _ := containerOf(t, created.Session)
	other, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		created.Session.Lock()
		other.Lock()
	})
	return twoDevices{created: created, this: created.Session, other: other, deviceKey: deviceKey, envelope: envelope}
}

func (d twoDevices) open(t *testing.T, container []byte, witness Witness, record *recordingWitness) (*Session, *Divergence, error) {
	t.Helper()
	return OpenWithDevice(container, d.deviceKey, d.envelope, &witness, record.persist)
}

func TestDeviceOpensSuccessorsHoweverFarAhead(t *testing.T) {
	devices := newTwoDevices(t)
	_, acknowledged := containerOf(t, devices.this)
	saveCredentials(t, devices.other, "one", "two", "three")
	container, head := containerOf(t, devices.other)
	record := &recordingWitness{}
	opened, divergence, err := devices.open(t, container, WitnessFor(acknowledged), record)
	if err != nil || divergence != nil {
		t.Fatalf("three saves ahead: divergence %v, error %v", divergence, err)
	}
	defer opened.Lock()
	if !slices.Equal(record.recorded, []Witness{WitnessFor(head)}) {
		t.Fatalf("recorded %v, want the new head", record.recorded)
	}
	if entries, err := opened.List(); err != nil || len(entries) != 4 {
		t.Fatalf("the other device's changes: %d entries, error %v", len(entries), err)
	}
}

func TestDeviceRefusesAnOlderCopy(t *testing.T) {
	devices := newTwoDevices(t)
	older, _ := containerOf(t, devices.this)
	saveCredentials(t, devices.this, "newer")
	_, acknowledged := containerOf(t, devices.this)
	record := &recordingWitness{}
	opened, divergence, err := devices.open(t, older, WitnessFor(acknowledged), record)
	if opened != nil || divergence != nil || !errors.Is(err, ErrWitnessOlder) || len(record.recorded) != 0 {
		t.Fatalf("an older copy: session %v, divergence %v, error %v, recorded %v", opened, divergence, err, record.recorded)
	}
}

func TestDeviceHoldsADivergedFileUntilItsOwnerAdoptsIt(t *testing.T) {
	devices := newTwoDevices(t)
	saveCredentials(t, devices.this, "saved here")
	_, acknowledged := containerOf(t, devices.this)
	saveCredentials(t, devices.other, "saved there", "saved there again")
	container, head := containerOf(t, devices.other)
	record := &recordingWitness{}
	opened, divergence, err := devices.open(t, container, WitnessFor(acknowledged), record)
	if opened != nil || divergence == nil || err != nil || len(record.recorded) != 0 {
		t.Fatalf("a fork: session %v, divergence %v, error %v, recorded %v", opened, divergence, err, record.recorded)
	}
	adopted, err := divergence.Adopt(record.persist)
	if err != nil {
		t.Fatal(err)
	}
	defer adopted.Lock()
	if !slices.Equal(record.recorded, []Witness{WitnessFor(head)}) || mustHead(t, adopted) != head {
		t.Fatalf("adopting recorded %v", record.recorded)
	}
	if entries, err := adopted.List(); err != nil || len(entries) != 3 {
		t.Fatalf("the adopted version: %d entries, error %v", len(entries), err)
	}
	if again, err := divergence.Adopt(record.persist); again != nil || !errors.Is(err, ErrLocked) {
		t.Fatalf("a second adoption: session %v, error %v", again, err)
	}
}

func TestADivergedFileStaysClosedWhenItsAdoptionFails(t *testing.T) {
	devices := newTwoDevices(t)
	saveCredentials(t, devices.this, "saved here")
	_, acknowledged := containerOf(t, devices.this)
	saveCredentials(t, devices.other, "saved there", "saved there again")
	container, _ := containerOf(t, devices.other)

	_, discarded, err := devices.open(t, container, WitnessFor(acknowledged), &recordingWitness{})
	if err != nil {
		t.Fatal(err)
	}
	discarded.Discard()
	if opened, err := discarded.Adopt((&recordingWitness{}).persist); opened != nil || !errors.Is(err, ErrLocked) {
		t.Fatalf("adopting a discarded divergence: session %v, error %v", opened, err)
	}

	_, refused, err := devices.open(t, container, WitnessFor(acknowledged), &recordingWitness{})
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("the witness could not be recorded")
	if opened, err := refused.Adopt((&recordingWitness{fail: failure}).persist); opened != nil || !errors.Is(err, failure) {
		t.Fatalf("a failed record: session %v, error %v", opened, err)
	}
}

func TestDeviceTreatsAFileBeyondTheAncestryAsDiverged(t *testing.T) {
	devices := newTwoDevices(t)
	_, acknowledged := containerOf(t, devices.this)
	saveCredentials(t, devices.other, numberedLabels("There", maxAncestry)...)
	reached, _ := containerOf(t, devices.other)
	opened, divergence, err := devices.open(t, reached, WitnessFor(acknowledged), &recordingWitness{})
	if err != nil || divergence != nil {
		t.Fatalf("as far ahead as the ancestry reaches: divergence %v, error %v", divergence, err)
	}
	opened.Lock()
	saveCredentials(t, devices.other, "one more")
	beyond, _ := containerOf(t, devices.other)
	opened, divergence, err = devices.open(t, beyond, WitnessFor(acknowledged), &recordingWitness{})
	if opened != nil || divergence == nil || err != nil {
		t.Fatalf("beyond the ancestry: session %v, divergence %v, error %v", opened, divergence, err)
	}
	divergence.Discard()
}

func TestAContainerWithoutAncestryNamesOnlyItsPreviousRevision(t *testing.T) {
	devices := newTwoDevices(t)
	_, twoBack := containerOf(t, devices.other)
	saveCredentials(t, devices.other, "one")
	_, oneBack := containerOf(t, devices.other)
	saveCredentials(t, devices.other, "two")
	raw, head := currentRaw(t, devices.other)
	plaintext, err := encodeIndex(head.Revision, ancestry{head.PreviousHash}, devices.other.entries, devices.other.groups, devices.other.retention)
	if err != nil {
		t.Fatal(err)
	}
	written := withIndex(t, devices.other, raw, plaintext)
	opened, divergence, err := devices.open(t, written, WitnessFor(oneBack), &recordingWitness{})
	if err != nil || divergence != nil {
		t.Fatalf("one revision ahead: divergence %v, error %v", divergence, err)
	}
	opened.Lock()
	opened, divergence, err = devices.open(t, written, WitnessFor(twoBack), &recordingWitness{})
	if opened != nil || divergence == nil || err != nil {
		t.Fatalf("two revisions ahead: session %v, divergence %v, error %v", opened, divergence, err)
	}
	divergence.Discard()
}

func TestAnOpenSessionFollowsItsFile(t *testing.T) {
	devices := newTwoDevices(t)
	ticket, err := devices.this.BeginSelection(mustList(t, devices.this)[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	saveCredentials(t, devices.other, "one", "two")
	container, head := containerOf(t, devices.other)
	record := &recordingWitness{}
	if decision, err := devices.this.Follow(container, record.persist); err != nil || decision != WitnessAdvance {
		t.Fatalf("a successor: %v, %v", decision, err)
	}
	if !slices.Equal(record.recorded, []Witness{WitnessFor(head)}) || mustHead(t, devices.this) != head {
		t.Fatalf("following recorded %v", record.recorded)
	}
	if len(mustList(t, devices.this)) != 3 {
		t.Fatal("the other device's changes are not shown")
	}
	if _, err := devices.this.ReadSelected(ticket); !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("a selection from before: got %v, want ErrStaleSelection", err)
	}
	if decision, err := devices.this.Follow(container, record.persist); err != nil || decision != WitnessEqual || len(record.recorded) != 1 {
		t.Fatalf("the same file again: %v, %v", decision, err)
	}
	saveCredentials(t, devices.this, "saved here after following")
	mine, _ := containerOf(t, devices.this)
	if decision, err := devices.other.Follow(mine, record.persist); err != nil || decision != WitnessAdvance {
		t.Fatalf("the other device following back: %v, %v", decision, err)
	}
}

func mustList(t *testing.T, session *Session) []Entry {
	t.Helper()
	entries, err := session.List()
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestAnOpenSessionKeepsItsVersionAgainstFilesItDoesNotFollow(t *testing.T) {
	devices := newTwoDevices(t)
	older, _ := containerOf(t, devices.this)
	saveCredentials(t, devices.this, "saved here")
	current, head := containerOf(t, devices.this)
	saveCredentials(t, devices.other, "saved there", "saved there again")
	diverged, _ := containerOf(t, devices.other)
	third, err := OpenWithRecovery(current, devices.created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer third.Lock()
	saveCredentials(t, third, "saved on a third device")
	successor, _ := containerOf(t, third)
	tampered := bytes.Clone(successor)
	tampered[len(tampered)-1] ^= 1
	foreign, _ := vaultWith(t)
	defer foreign.Session.Lock()
	failure := errors.New("the witness could not be recorded")

	unchanged := func(t *testing.T) {
		t.Helper()
		if mustHead(t, devices.this) != head || len(mustList(t, devices.this)) != 2 {
			t.Fatal("the session changed")
		}
	}
	tests := []struct {
		name      string
		container []byte
		record    *recordingWitness
		want      error
	}{
		{"an older copy", older, &recordingWitness{}, ErrWitnessOlder},
		{"a diverged file", diverged, &recordingWitness{}, ErrWitnessDiverged},
		{"another vault", foreign.Container, &recordingWitness{}, ErrWitnessMismatch},
		{"a changed file", tampered, &recordingWitness{}, ErrAuthentication},
		{"a successor whose witness is not recorded", successor, &recordingWitness{fail: failure}, failure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := devices.this.Follow(test.container, test.record.persist); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
			unchanged(t)
		})
	}

	pending, _, err := devices.this.PrepareCreate(CredentialInput{Label: "being saved"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := devices.this.Follow(successor, (&recordingWitness{}).persist); !errors.Is(err, ErrPendingCommit) {
		t.Fatalf("during a save: got %v, want ErrPendingCommit", err)
	}
	if err := devices.this.Abort(pending); err != nil {
		t.Fatal(err)
	}
	unchanged(t)
	devices.this.Lock()
	if _, err := devices.this.Follow(successor, (&recordingWitness{}).persist); !errors.Is(err, ErrLocked) {
		t.Fatalf("a locked session: got %v, want ErrLocked", err)
	}
}
