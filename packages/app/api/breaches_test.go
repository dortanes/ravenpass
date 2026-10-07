package api

import (
	"context"
	"crypto/sha1"
	"slices"
	"sync"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/breaches"
)

// fakeBreaches counts the digests it was given and fails with err when set.
type fakeBreaches struct {
	mu        sync.Mutex
	counts    map[[sha1.Size]byte]int
	asked     int
	forgotten int
	err       error
}

func (f *fakeBreaches) Count(_ context.Context, digest [sha1.Size]byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked++
	return f.counts[digest], f.err
}

func (f *fakeBreaches) Forget() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forgotten++
}

func TestBreachChecksStayOffUntilTurnedOn(t *testing.T) {
	service := newReadyService(t)
	fake := &fakeBreaches{}
	service.breaches = fake
	if checks, err := service.GetBreachChecks(); err != nil || checks.Enabled {
		t.Fatalf("checks = %+v, %v", checks, err)
	}
	_, err := service.CheckBreaches(context.Background())
	assertFailure(t, err, failureBreachChecksOff)
	_, err = service.CheckPassword(context.Background(), "password")
	assertFailure(t, err, failureBreachChecksOff)
	if fake.asked != 0 {
		t.Fatalf("the service was asked %d times while off", fake.asked)
	}
}

func TestCheckBreachesNamesBreachedPasswordsOnly(t *testing.T) {
	service := newReadyService(t)
	fake := &fakeBreaches{counts: map[[sha1.Size]byte]int{breaches.Digest("password"): 120}}
	service.breaches = fake
	if err := service.SetBreachChecks(true); err != nil {
		t.Fatal(err)
	}
	weak, err := service.CreateCredential(CredentialInput{Label: "Forum", Password: "password"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateCredential(CredentialInput{Label: "Mail", Password: "Orbit-Velvet7-Canyon"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateCredential(CredentialInput{Label: "Empty"}, nil); err != nil {
		t.Fatal(err)
	}
	trashed, err := service.CreateCredential(CredentialInput{Label: "Old", Password: "password"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.TrashItem(trashed); err != nil {
		t.Fatal(err)
	}
	found, err := service.CheckBreaches(context.Background())
	if err != nil || found.Checked != 2 || !slices.Equal(found.Breaches, []Breach{{ID: weak, Count: 120}}) {
		t.Fatalf("breaches = %+v, %v", found, err)
	}
	if fake.asked != 2 {
		t.Fatalf("asked for %d passwords, want the two non-empty ones outside the trash", fake.asked)
	}
	if count, err := service.CheckPassword(context.Background(), "password"); err != nil || count != 120 {
		t.Fatalf("typed password = %d, %v", count, err)
	}
	if count, err := service.CheckPassword(context.Background(), ""); err != nil || count != 0 {
		t.Fatalf("empty password = %d, %v", count, err)
	}
}

func TestBreachAnswersAreForgottenOnLockAndWhenTurnedOff(t *testing.T) {
	service := newReadyService(t)
	fake := &fakeBreaches{}
	service.breaches = fake
	if err := service.SetBreachChecks(true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetBreachChecks(false); err != nil {
		t.Fatal(err)
	}
	if err := service.Lock(); err != nil {
		t.Fatal(err)
	}
	if fake.forgotten != 2 {
		t.Fatalf("forgotten %d times, want on turning off and on lock", fake.forgotten)
	}
}

func TestAnUnreachableServiceIsItsOwnFailure(t *testing.T) {
	service := newReadyService(t)
	service.breaches = &fakeBreaches{err: breaches.ErrUnanswered}
	if err := service.SetBreachChecks(true); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateCredential(CredentialInput{Label: "Forum", Password: "password"}, nil); err != nil {
		t.Fatal(err)
	}
	_, err := service.CheckBreaches(context.Background())
	assertFailure(t, err, failureBreachCheckUnreachable)
	_, err = service.CheckPassword(context.Background(), "password")
	assertFailure(t, err, failureBreachCheckUnreachable)
}
