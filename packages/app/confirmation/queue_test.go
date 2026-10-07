package confirmation

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/dortanes/ravenpass/packages/app/unlock"
)

const (
	rightPIN = "135790"
	testWait = 5 * time.Second
)

var passport = Sharing("passport.pdf", "Alex", "example.com")

// fakePINs takes rightPIN and removes the PIN after attemptsLeft wrong ones.
type fakePINs struct {
	mu           sync.Mutex
	attemptsLeft int
	removed      bool
}

func (f *fakePINs) VerifyPIN(pin string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case f.removed:
		return unlock.ErrNoPIN
	case pin == rightPIN:
		return nil
	}
	f.attemptsLeft--
	if f.attemptsLeft == 0 {
		f.removed = true
		return unlock.ErrPINRemoved
	}
	return unlock.ErrWrongPIN
}

// watcher records every state the queue announces.
type watcher struct {
	mu    sync.Mutex
	heard []bool
}

func (w *watcher) follow(waiting bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.heard = append(w.heard, waiting)
}

func (w *watcher) states() []bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.heard)
}

func newQueue(t *testing.T) (*Queue, *fakePINs) {
	t.Helper()
	pins := &fakePINs{attemptsLeft: unlock.MaxPINFailures}
	queue, err := New(pins)
	if err != nil {
		t.Fatal(err)
	}
	return queue, pins
}

// ask asks queue to verify reason and reports how the request ended.
func ask(ctx context.Context, queue *Queue, reason Reason) <-chan error {
	ended := make(chan error, 1)
	go func() { ended <- queue.Ask(ctx, reason) }()
	return ended
}

func next(t *testing.T, queue *Queue) Request {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()
	request, err := queue.Next(ctx, "")
	if err != nil {
		t.Fatalf("no request was queued: %v", err)
	}
	return request
}

func ended(t *testing.T, outcome <-chan error) error {
	t.Helper()
	select {
	case err := <-outcome:
		return err
	case <-time.After(testWait):
		t.Fatal("the request kept waiting")
		return nil
	}
}

func assertEmpty(t *testing.T, queue *Queue) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if waiting, err := queue.Next(ctx, ""); err == nil {
		t.Fatalf("a request still waits: %+v", waiting)
	}
}

func TestNewRequiresPINChecks(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("a queue without PIN checks was accepted")
	}
}

func TestTheRightPINVerifies(t *testing.T) {
	queue, _ := newQueue(t)
	outcome := ask(context.Background(), queue, passport)
	request := next(t, queue)
	if request.Kind != KindVerify || request.Reason != passport || request.ID == "" {
		t.Fatalf("request = %+v", request)
	}
	if err := queue.ConfirmPIN(request.ID, rightPIN); err != nil {
		t.Fatal(err)
	}
	if err := ended(t, outcome); err != nil {
		t.Fatalf("a verified request: %v", err)
	}
	if err := queue.ConfirmPIN(request.ID, rightPIN); !errors.Is(err, ErrEnded) {
		t.Fatalf("confirming an answered request: got %v, want ErrEnded", err)
	}
	if _, err := queue.Waiting(request.ID); !errors.Is(err, ErrEnded) {
		t.Fatalf("an answered request still waits: %v", err)
	}
	assertEmpty(t, queue)
}

func TestAWrongPINLeavesTheRequestWaiting(t *testing.T) {
	queue, _ := newQueue(t)
	outcome := ask(context.Background(), queue, passport)
	request := next(t, queue)
	if err := queue.ConfirmPIN(request.ID, "999999"); !errors.Is(err, unlock.ErrWrongPIN) {
		t.Fatalf("a wrong PIN: got %v, want ErrWrongPIN", err)
	}
	if again, err := queue.Waiting(request.ID); err != nil || again != request {
		t.Fatalf("after a wrong PIN the request is %+v, %v", again, err)
	}
	if err := queue.ConfirmPIN(request.ID, rightPIN); err != nil {
		t.Fatal(err)
	}
	if err := ended(t, outcome); err != nil {
		t.Fatalf("the right PIN after a wrong one: %v", err)
	}
}

func TestTheAttemptThatRemovesThePINDeclines(t *testing.T) {
	queue, pins := newQueue(t)
	pins.attemptsLeft = 2
	outcome := ask(context.Background(), queue, passport)
	request := next(t, queue)
	if err := queue.ConfirmPIN(request.ID, "999999"); !errors.Is(err, unlock.ErrWrongPIN) {
		t.Fatalf("the first wrong PIN: got %v, want ErrWrongPIN", err)
	}
	if err := queue.ConfirmPIN(request.ID, "999998"); !errors.Is(err, unlock.ErrPINRemoved) {
		t.Fatalf("the last wrong PIN: got %v, want ErrPINRemoved", err)
	}
	if err := ended(t, outcome); !errors.Is(err, ErrDeclined) {
		t.Fatalf("a request whose PIN was removed: got %v, want ErrDeclined", err)
	}
	if err := queue.ConfirmPIN(request.ID, rightPIN); !errors.Is(err, ErrEnded) {
		t.Fatalf("confirming after the PIN was removed: got %v, want ErrEnded", err)
	}
}

func TestADeclinedRequestEnds(t *testing.T) {
	queue, _ := newQueue(t)
	outcome := ask(context.Background(), queue, passport)
	request := next(t, queue)
	if err := queue.Decline(request.ID); err != nil {
		t.Fatal(err)
	}
	if err := ended(t, outcome); !errors.Is(err, ErrDeclined) {
		t.Fatalf("a declined request: got %v, want ErrDeclined", err)
	}
	if err := queue.Decline(request.ID); !errors.Is(err, ErrEnded) {
		t.Fatalf("declining twice: got %v, want ErrEnded", err)
	}
	if err := queue.Decline("missing"); !errors.Is(err, ErrEnded) {
		t.Fatalf("declining an unknown request: got %v, want ErrEnded", err)
	}
}

func TestACallersContextWithdrawsItsRequest(t *testing.T) {
	queue, _ := newQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	outcome := ask(ctx, queue, passport)
	request := next(t, queue)
	cancel()
	if err := ended(t, outcome); !errors.Is(err, context.Canceled) {
		t.Fatalf("a withdrawn request: got %v, want context.Canceled", err)
	}
	if err := queue.ConfirmPIN(request.ID, rightPIN); !errors.Is(err, ErrEnded) {
		t.Fatalf("confirming a withdrawn request: got %v, want ErrEnded", err)
	}
	assertEmpty(t, queue)
}

func TestEndAllEndsEveryRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue, _ := newQueue(t)
		first := ask(context.Background(), queue, passport)
		next(t, queue)
		second := ask(context.Background(), queue, SavingPasskey("example.com"))
		synctest.Wait()
		if waiting := kinds(queue); !slices.Equal(waiting, []Kind{KindVerify, KindVerify}) {
			t.Fatalf("waiting requests = %v, want two verify requests", waiting)
		}
		unlocking := queue.PostUnlock(RequesterExtension)
		locked := errors.New("the vault locked")
		queue.EndAll(locked)
		for _, outcome := range []<-chan error{first, second} {
			if err := ended(t, outcome); !errors.Is(err, locked) {
				t.Fatalf("an ended request: got %v, want %v", err, locked)
			}
		}
		if _, err := queue.Waiting(unlocking); !errors.Is(err, ErrEnded) {
			t.Fatalf("the unlock request outlived EndAll: %v", err)
		}
	})
}

// kinds lists the kinds of the waiting requests, oldest first.
func kinds(queue *Queue) []Kind {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	listed := make([]Kind, len(queue.waiting))
	for i, waiting := range queue.waiting {
		listed[i] = waiting.Kind
	}
	return listed
}

func TestRequestsAreServedOldestFirst(t *testing.T) {
	queue, _ := newQueue(t)
	arrived := make(chan Request, 1)
	go func() {
		request, err := queue.Next(context.Background(), "")
		if err == nil {
			arrived <- request
		}
	}()
	first := ask(context.Background(), queue, passport)
	var oldest Request
	select {
	case oldest = <-arrived:
	case <-time.After(testWait):
		t.Fatal("a waiting Next did not see the queued request")
	}
	unlocking := queue.PostUnlock(RequesterExtension)
	if again := next(t, queue); again != oldest {
		t.Fatalf("the oldest request is %+v, want %+v", again, oldest)
	}
	if err := queue.Decline(oldest.ID); err != nil {
		t.Fatal(err)
	}
	if newer := next(t, queue); newer.ID != unlocking || newer.Kind != KindUnlock {
		t.Fatalf("after the oldest the queue serves %+v", newer)
	}
	if err := ended(t, first); !errors.Is(err, ErrDeclined) {
		t.Fatalf("the first request: %v", err)
	}
	if err := queue.Decline(unlocking); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := queue.Next(ctx, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("Next with an ended context: got %v, want context.Canceled", err)
	}
}

func TestAWindowLearnsWhenTheRequestItShowsLeaves(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue, _ := newQueue(t)
		first := queue.PostUnlock(RequesterExtension)
		shown := make(chan Request, 1)
		go func() {
			request, err := queue.Next(context.Background(), first)
			if err == nil {
				shown <- request
			}
		}()
		outcome := ask(context.Background(), queue, passport)
		synctest.Wait()
		if waiting := kinds(queue); len(waiting) != 2 {
			t.Fatalf("waiting requests = %v, want two", waiting)
		}
		select {
		case request := <-shown:
			t.Fatalf("a newer request replaced the oldest: %+v", request)
		default:
		}
		if err := queue.Decline(first); err != nil {
			t.Fatal(err)
		}
		var verifying Request
		select {
		case verifying = <-shown:
		case <-time.After(testWait):
			t.Fatal("the window did not learn the request it showed had left")
		}
		if verifying.Kind != KindVerify || verifying.Reason != passport {
			t.Fatalf("after the unlock request the window shows %+v", verifying)
		}
		if err := queue.Decline(verifying.ID); err != nil {
			t.Fatal(err)
		}
		if err := ended(t, outcome); !errors.Is(err, ErrDeclined) {
			t.Fatalf("the verify request: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), testWait)
		defer cancel()
		if none, err := queue.Next(ctx, verifying.ID); err != nil || none != (Request{}) {
			t.Fatalf("once nothing waits the window is shown %+v, %v", none, err)
		}
	})
}

func TestAtMostOneUnlockRequestWaits(t *testing.T) {
	queue, _ := newQueue(t)
	first := queue.PostUnlock(RequesterAutofill)
	if again := queue.PostUnlock(RequesterExtension); again != first {
		t.Fatalf("a second unlock request got %q, want the waiting %q", again, first)
	}
	request := next(t, queue)
	if request.ID != first || request.Kind != KindUnlock || request.Reason != (Reason{}) || request.Requester != RequesterAutofill {
		t.Fatalf("unlock request = %+v", request)
	}
	if err := queue.ConfirmPIN(first, rightPIN); !errors.Is(err, ErrEnded) {
		t.Fatalf("a PIN for an unlock request: got %v, want ErrEnded", err)
	}
	if err := queue.Decline(first); err != nil {
		t.Fatal(err)
	}
	if later := queue.PostUnlock(RequesterExtension); later == first {
		t.Fatal("an unlock request after a declined one reused its ID")
	}
}

func TestOpeningTheVaultEndsOnlyTheUnlockRequest(t *testing.T) {
	queue, _ := newQueue(t)
	outcome := ask(context.Background(), queue, passport)
	verifying := next(t, queue)
	unlocking := queue.PostUnlock(RequesterExtension)
	queue.VaultOpened()
	if _, err := queue.Waiting(unlocking); !errors.Is(err, ErrEnded) {
		t.Fatalf("the unlock request outlived the vault opening: %v", err)
	}
	if still, err := queue.Waiting(verifying.ID); err != nil || still != verifying {
		t.Fatalf("the verify request after the vault opened: %+v, %v", still, err)
	}
	if err := queue.Decline(verifying.ID); err != nil {
		t.Fatal(err)
	}
	if err := ended(t, outcome); !errors.Is(err, ErrDeclined) {
		t.Fatalf("the verify request: %v", err)
	}
}

func TestWatchersHearEachChangeOfWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue, _ := newQueue(t)
		early := &watcher{}
		queue.Watch(early.follow)
		unlocking := queue.PostUnlock(RequesterExtension)
		queue.PostUnlock(RequesterExtension)
		late := &watcher{}
		queue.Watch(late.follow)
		outcome := ask(context.Background(), queue, passport)
		synctest.Wait()
		if waiting := kinds(queue); len(waiting) != 2 {
			t.Fatalf("waiting requests = %v, want two", waiting)
		}
		if err := queue.Decline(unlocking); err != nil {
			t.Fatal(err)
		}
		if err := queue.Decline(next(t, queue).ID); err != nil {
			t.Fatal(err)
		}
		if err := ended(t, outcome); !errors.Is(err, ErrDeclined) {
			t.Fatalf("the verify request: %v", err)
		}
		if heard := early.states(); !slices.Equal(heard, []bool{false, true, false}) {
			t.Fatalf("a watcher from the start heard %v", heard)
		}
		if heard := late.states(); !slices.Equal(heard, []bool{true, false}) {
			t.Fatalf("a watcher added while a request waited heard %v", heard)
		}
	})
}

// deviceTries holds each unlock the queue tries on the device until the test answers it.
type deviceTries struct {
	started chan struct{}
	answers chan bool
}

func tryOnDevice(queue *Queue) *deviceTries {
	tries := &deviceTries{started: make(chan struct{}, 4), answers: make(chan bool)}
	queue.UnlockOnDevice(func() bool {
		tries.started <- struct{}{}
		return <-tries.answers
	})
	return tries
}

func TestAnUnlockRequestShowsOnlyOnceTheDeviceFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue, _ := newQueue(t)
		tries := tryOnDevice(queue)
		heard := &watcher{}
		queue.Watch(heard.follow)
		unlocking := queue.PostUnlock(RequesterExtension)
		<-tries.started
		if joined := queue.PostUnlock(RequesterAutofill); joined != unlocking {
			t.Fatalf("a second unlock request got %q, want the held %q", joined, unlocking)
		}
		synctest.Wait()
		if len(tries.started) != 0 {
			t.Fatal("a second unlock request tried the device again")
		}
		if _, err := queue.Waiting(unlocking); err != nil {
			t.Fatalf("the held request does not wait: %v", err)
		}
		assertEmpty(t, queue)
		tries.answers <- false
		if shown := next(t, queue); shown.ID != unlocking || shown.Kind != KindUnlock {
			t.Fatalf("after the device failed the panel shows %+v", shown)
		}
		synctest.Wait()
		if states := heard.states(); !slices.Equal(states, []bool{false, true}) {
			t.Fatalf("the watcher heard %v", states)
		}
	})
}

func TestAHeldRequestKeepsItsPlaceOnceShown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue, _ := newQueue(t)
		tries := tryOnDevice(queue)
		unlocking := queue.PostUnlock(RequesterExtension)
		<-tries.started
		outcome := ask(context.Background(), queue, passport)
		synctest.Wait()
		if shown := next(t, queue); shown.Kind != KindVerify {
			t.Fatalf("while the unlock request is held the panel shows %+v", shown)
		}
		tries.answers <- false
		synctest.Wait()
		if shown := next(t, queue); shown.ID != unlocking {
			t.Fatalf("once shown the older unlock request is not first: %+v", shown)
		}
		queue.EndAll(ErrDeclined)
		if err := ended(t, outcome); !errors.Is(err, ErrDeclined) {
			t.Fatalf("the verify request: %v", err)
		}
	})
}

func TestADeviceUnlockEndsTheRequestUnseen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue, _ := newQueue(t)
		queue.UnlockOnDevice(func() bool {
			queue.VaultOpened()
			return true
		})
		heard := &watcher{}
		queue.Watch(heard.follow)
		unlocking := queue.PostUnlock(RequesterExtension)
		ctx, cancel := context.WithTimeout(context.Background(), testWait)
		defer cancel()
		if err := queue.AwaitEnd(ctx, unlocking); err != nil {
			t.Fatalf("the unlock request outlived the device unlock: %v", err)
		}
		synctest.Wait()
		if states := heard.states(); !slices.Equal(states, []bool{false}) {
			t.Fatalf("the watcher heard %v, want the panel never shown", states)
		}
	})
}

func TestAHeldRequestThatEndedIsNotShown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue, _ := newQueue(t)
		tries := tryOnDevice(queue)
		unlocking := queue.PostUnlock(RequesterExtension)
		<-tries.started
		queue.EndAll(errors.New("the vault locked"))
		tries.answers <- false
		synctest.Wait()
		if _, err := queue.Waiting(unlocking); !errors.Is(err, ErrEnded) {
			t.Fatalf("an ended request came back: %v", err)
		}
		assertEmpty(t, queue)
	})
}

func TestAwaitEndFollowsOneRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue, _ := newQueue(t)
		unlocking := queue.PostUnlock(RequesterExtension)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := queue.AwaitEnd(ctx, unlocking); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("a waiting request: got %v, want DeadlineExceeded", err)
		}
		awaited := make(chan error, 1)
		go func() { awaited <- queue.AwaitEnd(context.Background(), unlocking) }()
		outcome := ask(context.Background(), queue, passport)
		synctest.Wait()
		select {
		case err := <-awaited:
			t.Fatalf("another request's arrival ended the wait: %v", err)
		default:
		}
		if err := queue.Decline(unlocking); err != nil {
			t.Fatal(err)
		}
		if err := ended(t, awaited); err != nil {
			t.Fatalf("a declined request: %v", err)
		}
		queue.EndAll(ErrDeclined)
		if err := ended(t, outcome); !errors.Is(err, ErrDeclined) {
			t.Fatalf("the verify request: %v", err)
		}
	})
}

func TestEachReasonIsWordedForItsKind(t *testing.T) {
	for reason, want := range map[Reason]string{
		passport:                         "share passport.pdf from Alex with example.com",
		SavingPasskey("example.com"):     "save a passkey for example.com",
		SigningIn("example.com", "alex"): "sign in to example.com as alex",
		Filling("example.com", "alex"):   "fill alex on example.com",
		ChangingUnlock():                 "change how the vault opens",
		CreatingVault():                  "create a vault",
		OpeningVault():                   "open a vault file",
		DeletingVault("Personal"):        "delete Personal",
	} {
		if got := reason.Words(wording{}); got != want {
			t.Fatalf("%+v reads %q, want %q", reason, got, want)
		}
	}
	if passport != (Reason{Kind: ReasonShare, File: "passport.pdf", Identity: "Alex", Site: "example.com"}) {
		t.Fatalf("a share names %+v", passport)
	}
}

// wording words each reason as a test can tell them apart.
type wording struct{}

func (wording) ShareReason(file, identity, site string) string {
	return "share " + file + " from " + identity + " with " + site
}

func (wording) SavePasskeyReason(site string) string { return "save a passkey for " + site }

func (wording) SignInReason(site, account string) string {
	return "sign in to " + site + " as " + account
}

func (wording) FillReason(site, account string) string { return "fill " + account + " on " + site }

func (wording) ChangeUnlockReason() string { return "change how the vault opens" }

func (wording) CreateVaultReason() string { return "create a vault" }

func (wording) OpenVaultReason() string { return "open a vault file" }

func (wording) DeleteVaultReason(vault string) string { return "delete " + vault }
func (wording) RevealReason(item string) string       { return "show " + item }
