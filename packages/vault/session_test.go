package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func populatedSession(t *testing.T) (*Session, ID) {
	t.Helper()
	created, err := Create()
	if err != nil {
		t.Fatal(err)
	}
	pending, id, err := created.Session.PrepareCreate(CredentialInput{Label: "Example", Password: "secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := created.Session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	return created.Session, id
}

func TestLateSelectionResultAfterLock(t *testing.T) {
	session, id := populatedSession(t)
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	session.afterDecrypt = func() { close(started); <-release }
	result := make(chan error, 1)
	go func() {
		_, err := session.ReadSelected(ticket)
		result <- err
	}()
	<-started
	session.Lock()
	close(release)
	if err := <-result; !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("late read returned %v", err)
	}
	if _, err := session.List(); !errors.Is(err, ErrLocked) {
		t.Fatalf("locked list: %v", err)
	}
}

func TestLateSelectionResultAfterNavigation(t *testing.T) {
	session, id := populatedSession(t)
	defer session.Lock()
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	session.afterDecrypt = func() { close(started); <-release }
	result := make(chan error, 1)
	go func() {
		_, err := session.ReadSelected(ticket)
		result <- err
	}()
	<-started
	session.ClearSelection()
	close(release)
	if err := <-result; !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("navigated read returned %v", err)
	}
}

func TestConcurrentSelectionOnlyLatestReturns(t *testing.T) {
	session, id := populatedSession(t)
	defer session.Lock()
	first, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	second, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadSelected(first); !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("old ticket: %v", err)
	}
	if value, err := session.ReadSelected(second); err != nil || value.Password != "secret" {
		t.Fatalf("latest ticket: %v", err)
	}
}

func TestNonceUniquenessAndUnchangedCiphertext(t *testing.T) {
	created, err := Create()
	if err != nil {
		t.Fatal(err)
	}
	session := created.Session
	defer session.Lock()
	first, firstID, err := session.PrepareCreate(CredentialInput{Label: "First", Password: "one"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(first); err != nil {
		t.Fatal(err)
	}
	second, _, err := session.PrepareCreate(CredentialInput{Label: "Second", Password: "two"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(second); err != nil {
		t.Fatal(err)
	}
	before, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	base, err := parseContainer(before)
	if err != nil {
		t.Fatal(err)
	}
	seenRecord := map[[24]byte]struct{}{base.records[0].nonce: {}}
	seenIndex := map[[24]byte]struct{}{base.index.nonce: {}}
	unchanged := encodeBox(base.records[1])
	for i := range 64 {
		password := fmt.Sprintf("password %d", i)
		pending, err := session.PrepareEdit(firstID, CredentialPatch{Password: &password})
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := parseContainer(pending.Container())
		if err != nil {
			t.Fatal(err)
		}
		if _, reused := seenRecord[candidate.records[0].nonce]; reused {
			t.Fatal("record nonce reused")
		}
		if _, reused := seenIndex[candidate.index.nonce]; reused {
			t.Fatal("index nonce reused")
		}
		seenRecord[candidate.records[0].nonce] = struct{}{}
		seenIndex[candidate.index.nonce] = struct{}{}
		if !bytes.Equal(encodeBox(candidate.records[1]), unchanged) {
			t.Fatal("unchanged record was re-encrypted")
		}
		if err := session.Commit(pending); err != nil {
			t.Fatal(err)
		}
	}
	if value := selectCredential(t, session, firstID); value.Password != "password 63" {
		t.Fatal("last edit missing")
	}
}

func TestReadsRacingALockSeeTheWholeCredentialOrARefusal(t *testing.T) {
	session, id := populatedSession(t)
	failures := make(chan error, 32)
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			ticket, err := session.BeginSelection(id)
			if err != nil {
				if !errors.Is(err, ErrLocked) {
					failures <- fmt.Errorf("selection: %w", err)
				}
				return
			}
			credential, err := session.ReadSelected(ticket)
			switch {
			case errors.Is(err, ErrLocked), errors.Is(err, ErrStaleSelection):
			case err != nil:
				failures <- fmt.Errorf("read: %w", err)
			case credential.Label != "Example" || credential.Password != "secret":
				failures <- fmt.Errorf("read a partial credential %q", credential.Label)
			}
		})
	}
	session.Lock()
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if _, err := session.BeginSelection(id); !errors.Is(err, ErrLocked) {
		t.Fatalf("a selection after the lock: got %v, want ErrLocked", err)
	}
}

func TestConcurrentSnapshotsKeepContainerAndHeadTogether(t *testing.T) {
	session, id := populatedSession(t)
	defer session.Lock()
	done := make(chan struct{})
	failures := make(chan error, 4)
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				container, head, err := session.CurrentContainer()
				if err != nil || sha256.Sum256(container) != head.Hash {
					failures <- fmt.Errorf("current container and head diverged: %v", err)
					return
				}
				container, head, err = session.Export()
				if errors.Is(err, ErrPendingCommit) {
					continue
				}
				if err != nil || sha256.Sum256(container) != head.Hash {
					failures <- fmt.Errorf("export and head diverged: %v", err)
					return
				}
			}
		}()
	}
	for i := range 64 {
		password := fmt.Sprintf("revision %d", i)
		pending, err := session.PrepareEdit(id, CredentialPatch{Password: &password})
		if err != nil {
			close(done)
			readers.Wait()
			t.Fatal(err)
		}
		if err := session.Commit(pending); err != nil {
			close(done)
			readers.Wait()
			t.Fatal(err)
		}
	}
	close(done)
	readers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	container, head, err := session.Export()
	if err != nil || sha256.Sum256(container) != head.Hash || head.Revision != 66 {
		t.Fatalf("final export and head diverged: revision %d, error %v", head.Revision, err)
	}
}

func TestEncodingRejectsOversizedInput(t *testing.T) {
	large := string(bytes.Repeat([]byte{'x'}, 4<<20))
	if _, err := encodeCredentialRecord(CredentialInput{Password: large}); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("oversized record: %v", err)
	}
	if _, err := encodeIdentityRecord(IdentityInput{Notes: large}); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("oversized identity record: %v", err)
	}
	entries := make([]entryMeta, 5)
	for i := range entries {
		entries[i].revision = 1
		entries[i].label = large
	}
	if _, err := encodeIndex(1, nil, entries, nil, DefaultTrashRetention); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("oversized index: %v", err)
	}
	sharedCiphertext := bytes.Repeat([]byte{'x'}, maxRecordBytes)
	records := make([]sealedBox, 300)
	for i := range records {
		records[i].ciphertext = sharedCiphertext
	}
	if _, err := encodeContainer(ID{}, sealedBox{}, sealedBox{}, records); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("oversized container: %v", err)
	}
}
