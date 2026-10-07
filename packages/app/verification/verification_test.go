package verification

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/ownerauth"
	"github.com/dortanes/ravenpass/packages/app/unlock"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
)

const (
	rightPIN = "135790"
	testWait = 5 * time.Second
)

var passport = confirmation.Sharing("passport.pdf", "Alex", "example.com")

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

func (wording) FillCardReason(site, card string) string {
	return "fill the card " + card + " on " + site
}

func (wording) ChangeUnlockReason() string { return "change how the vault opens" }

func (wording) CreateVaultReason() string { return "create a vault" }

func (wording) OpenVaultReason() string { return "open a vault file" }

func (wording) DeleteVaultReason(vault string) string { return "delete " + vault }
func (wording) RevealReason(item string) string       { return "show " + item }

type fakeMethods struct {
	methods vaultservice.Methods
	err     error
}

func (f fakeMethods) UnlockMethods() (vaultservice.Methods, error) { return f.methods, f.err }

// fakeOwner answers the system prompt with answer, or waits for its context when wait is set.
type fakeOwner struct {
	answer error
	wait   bool

	mu      sync.Mutex
	reasons []string
}

func (f *fakeOwner) AuthenticateOwner(ctx context.Context, reason string) error {
	f.mu.Lock()
	f.reasons = append(f.reasons, reason)
	f.mu.Unlock()
	if f.wait {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.answer
}

// fakePINs takes rightPIN and refuses every other.
type fakePINs struct{}

func (fakePINs) VerifyPIN(pin string) error {
	if pin == rightPIN {
		return nil
	}
	return unlock.ErrWrongPIN
}

type harness struct {
	verifier *Verifier
	queue    *confirmation.Queue
	owner    *fakeOwner
}

func newHarness(t *testing.T, methods vaultservice.Methods) *harness {
	t.Helper()
	h := &harness{owner: &fakeOwner{}}
	queue, err := confirmation.New(fakePINs{})
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := New(fakeMethods{methods: methods}, h.owner, func(reason confirmation.Reason) string { return reason.Words(wording{}) }, queue)
	if err != nil {
		t.Fatal(err)
	}
	h.verifier, h.queue = verifier, queue
	return h
}

var (
	macAndPIN = vaultservice.Methods{BiometryAvailable: true, BiometryEnabled: true, PINSet: true}
	pinOnly   = vaultservice.Methods{BiometryAvailable: true, PINSet: true}
)

type outcome struct {
	methods []Method
	err     error
}

// verify runs a verification of passport and reports how it was asked and how it ended.
func (h *harness) verify(ctx context.Context) <-chan outcome {
	return h.verifyFor(ctx, passport)
}

// verifyFor runs a verification of reason and reports how it was asked and how it ended.
func (h *harness) verifyFor(ctx context.Context, reason confirmation.Reason) <-chan outcome {
	ended := make(chan outcome, 1)
	go func() {
		var asked []Method
		err := h.verifier.Verify(ctx, reason, func(method Method) { asked = append(asked, method) })
		ended <- outcome{asked, err}
	}()
	return ended
}

func (h *harness) nextRequest(t *testing.T) confirmation.Request {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()
	next, err := h.queue.Next(ctx, "")
	if err != nil {
		t.Fatalf("no PIN request was queued: %v", err)
	}
	return next
}

// assertNothingQueued fails when a request waits in the queue.
func (h *harness) assertNothingQueued(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if waiting, err := h.queue.Next(ctx, ""); err == nil {
		t.Fatalf("a request still waits: %+v", waiting)
	}
}

func wait(t *testing.T, ended <-chan outcome) outcome {
	t.Helper()
	select {
	case result := <-ended:
		return result
	case <-time.After(testWait):
		t.Fatal("the verification did not end")
		return outcome{}
	}
}

func TestNewRequiresEveryPart(t *testing.T) {
	queue, err := confirmation.New(fakePINs{})
	if err != nil {
		t.Fatal(err)
	}
	reason := func(confirmation.Reason) string { return "" }
	for name, build := range map[string]func() (*Verifier, error){
		"no methods":    func() (*Verifier, error) { return New(nil, &fakeOwner{}, reason, queue) },
		"no owner":      func() (*Verifier, error) { return New(fakeMethods{}, nil, reason, queue) },
		"no reason":     func() (*Verifier, error) { return New(fakeMethods{}, &fakeOwner{}, nil, queue) },
		"no PIN prompt": func() (*Verifier, error) { return New(fakeMethods{}, &fakeOwner{}, reason, nil) },
	} {
		if _, err := build(); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

func TestTheMacsAuthenticationVerifiesWhereTheVaultOpensWithIt(t *testing.T) {
	h := newHarness(t, macAndPIN)
	result := wait(t, h.verify(context.Background()))
	if result.err != nil || !slices.Equal(result.methods, []Method{MethodDevice}) {
		t.Fatalf("verification = %+v", result)
	}
	if !slices.Equal(h.owner.reasons, []string{"share passport.pdf from Alex with example.com"}) {
		t.Fatalf("the system prompt said %q", h.owner.reasons)
	}
	h.assertNothingQueued(t)
}

func TestEachReasonReachesItsPrompt(t *testing.T) {
	for reason, want := range map[confirmation.Reason]string{
		passport: "share passport.pdf from Alex with example.com",
		confirmation.SavingPasskey("example.com"):     "save a passkey for example.com",
		confirmation.SigningIn("example.com", "alex"): "sign in to example.com as alex",
		confirmation.Filling("example.com", "alex"):   "fill alex on example.com",
		confirmation.ChangingUnlock():                 "change how the vault opens",
	} {
		h := newHarness(t, macAndPIN)
		if result := wait(t, h.verifyFor(context.Background(), reason)); result.err != nil || !slices.Equal(result.methods, []Method{MethodDevice}) {
			t.Fatalf("verifying %+v = %+v", reason, result)
		}
		if !slices.Equal(h.owner.reasons, []string{want}) {
			t.Fatalf("the system prompt for %+v said %q, want %q", reason, h.owner.reasons, want)
		}
		h = newHarness(t, pinOnly)
		ended := h.verifyFor(context.Background(), reason)
		if request := h.nextRequest(t); request.Kind != confirmation.KindVerify || request.Reason != reason {
			t.Fatalf("the PIN request for %+v is %+v", reason, request)
		} else if err := h.queue.Decline(request.ID); err != nil {
			t.Fatal(err)
		}
		if result := wait(t, ended); !errors.Is(result.err, ErrDeclined) {
			t.Fatalf("a declined %+v: got %v, want ErrDeclined", reason, result.err)
		}
	}
}

func TestTheSystemPromptsRefusalsEndTheVerification(t *testing.T) {
	for cause, want := range map[error]error{
		ownerauth.ErrCanceled:    ErrDeclined,
		ownerauth.ErrFailed:      ErrDeclined,
		ownerauth.ErrUnavailable: ErrUnverifiable,
	} {
		h := newHarness(t, macAndPIN)
		h.owner.answer = cause
		if result := wait(t, h.verify(context.Background())); !errors.Is(result.err, want) {
			t.Fatalf("a prompt answering %v: got %v, want %v", cause, result.err, want)
		}
	}
}

func TestThePINVerifiesWhereTheMacsAuthenticationIsNotOffered(t *testing.T) {
	for name, methods := range map[string]vaultservice.Methods{
		"turned off for the vault":  pinOnly,
		"unavailable on the device": {BiometryEnabled: true, PINSet: true},
		"neither on nor available":  {PINSet: true},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, methods)
			ended := h.verify(context.Background())
			request := h.nextRequest(t)
			if request.Reason != passport || request.ID == "" {
				t.Fatalf("PIN request = %+v", request)
			}
			if err := h.queue.ConfirmPIN(request.ID, rightPIN); err != nil {
				t.Fatal(err)
			}
			result := wait(t, ended)
			if result.err != nil || !slices.Equal(result.methods, []Method{MethodPIN}) {
				t.Fatalf("verification = %+v", result)
			}
			if len(h.owner.reasons) != 0 {
				t.Fatal("a PIN verification showed the system prompt")
			}
		})
	}
}

func TestAVaultWithoutEitherMethodCannotVerify(t *testing.T) {
	for name, methods := range map[string]vaultservice.Methods{
		"nothing":              {},
		"Mac only, not usable": {BiometryEnabled: true},
		"available, not on":    {BiometryAvailable: true},
	} {
		h := newHarness(t, methods)
		result := wait(t, h.verify(context.Background()))
		if !errors.Is(result.err, ErrUnverifiable) || len(result.methods) != 0 {
			t.Fatalf("%s: verification = %+v", name, result)
		}
		if len(h.owner.reasons) != 0 {
			t.Fatalf("%s: an unverifiable verification showed the system prompt", name)
		}
		h.assertNothingQueued(t)
	}
	h := newHarness(t, macAndPIN)
	failure := errors.New("unlock record unreadable")
	h.verifier.methods = fakeMethods{err: failure}
	if result := wait(t, h.verify(context.Background())); !errors.Is(result.err, failure) {
		t.Fatalf("unreadable methods: got %v", result.err)
	}
}

func TestAnUnansweredVerificationEndsAsDeclined(t *testing.T) {
	t.Run("system prompt", func(t *testing.T) {
		h := newHarness(t, macAndPIN)
		h.verifier.timeout = 50 * time.Millisecond
		h.owner.wait = true
		if result := wait(t, h.verify(context.Background())); !errors.Is(result.err, ErrDeclined) {
			t.Fatalf("an unanswered system prompt: got %v, want ErrDeclined", result.err)
		}
	})
	t.Run("PIN request", func(t *testing.T) {
		h := newHarness(t, pinOnly)
		h.verifier.timeout = 50 * time.Millisecond
		if result := wait(t, h.verify(context.Background())); !errors.Is(result.err, ErrDeclined) {
			t.Fatalf("an unanswered PIN request: got %v, want ErrDeclined", result.err)
		}
		h.assertNothingQueued(t)
	})
	if Timeout != 2*time.Minute {
		t.Fatalf("a verification waits %v", Timeout)
	}
}

func TestAnEndedSessionWithdrawsItsVerification(t *testing.T) {
	t.Run("system prompt", func(t *testing.T) {
		h := newHarness(t, macAndPIN)
		h.owner.wait = true
		ctx, cancel := context.WithCancel(context.Background())
		ended := h.verify(ctx)
		cancel()
		if result := wait(t, ended); !errors.Is(result.err, context.Canceled) {
			t.Fatalf("an ended session: got %v, want context.Canceled", result.err)
		}
	})
	t.Run("PIN request", func(t *testing.T) {
		h := newHarness(t, pinOnly)
		ctx, cancel := context.WithCancel(context.Background())
		ended := h.verify(ctx)
		request := h.nextRequest(t)
		cancel()
		if result := wait(t, ended); !errors.Is(result.err, context.Canceled) {
			t.Fatalf("an ended session: got %v, want context.Canceled", result.err)
		}
		if err := h.queue.ConfirmPIN(request.ID, rightPIN); !errors.Is(err, confirmation.ErrEnded) {
			t.Fatalf("confirming a withdrawn request: got %v, want ErrEnded", err)
		}
	})
}

func TestTheDeviceVerifiesAVaultChangeWhateverTheVaultOpensWith(t *testing.T) {
	for name, methods := range map[string]vaultservice.Methods{
		"device and PIN": macAndPIN,
		"PIN only":       pinOnly,
		"nothing":        {},
	} {
		h := newHarness(t, methods)
		for reason, want := range map[confirmation.Reason]string{
			confirmation.CreatingVault():           "create a vault",
			confirmation.OpeningVault():            "open a vault file",
			confirmation.DeletingVault("Personal"): "delete Personal",
		} {
			h.owner.reasons = nil
			if err := h.verifier.VerifyDevice(context.Background(), reason); err != nil {
				t.Fatalf("%s: verifying %+v: %v", name, reason, err)
			}
			if !slices.Equal(h.owner.reasons, []string{want}) {
				t.Fatalf("%s: the system prompt for %+v said %q, want %q", name, reason, h.owner.reasons, want)
			}
		}
		h.assertNothingQueued(t)
	}
}

func TestTheDevicesAnswerEndsAVaultChangeVerification(t *testing.T) {
	for cause, want := range map[error]error{
		ownerauth.ErrCanceled:    ErrDeclined,
		ownerauth.ErrFailed:      ownerauth.ErrFailed,
		ownerauth.ErrUnavailable: ErrUnverifiable,
	} {
		h := newHarness(t, macAndPIN)
		h.owner.answer = cause
		if err := h.verifier.VerifyDevice(context.Background(), confirmation.CreatingVault()); !errors.Is(err, want) {
			t.Fatalf("a prompt answering %v: got %v, want %v", cause, err, want)
		}
	}
	h := newHarness(t, macAndPIN)
	h.verifier.timeout = 50 * time.Millisecond
	h.owner.wait = true
	if err := h.verifier.VerifyDevice(context.Background(), confirmation.OpeningVault()); !errors.Is(err, ErrDeclined) {
		t.Fatalf("an unanswered prompt: got %v, want ErrDeclined", err)
	}
	h.verifier.timeout = Timeout
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.verifier.VerifyDevice(ctx, confirmation.OpeningVault()); !errors.Is(err, context.Canceled) {
		t.Fatalf("an ended session: got %v, want context.Canceled", err)
	}
}

func TestLockingEndsAVerificationWithItsCause(t *testing.T) {
	h := newHarness(t, pinOnly)
	ended := h.verify(context.Background())
	h.nextRequest(t)
	h.queue.EndAll(vaultservice.ErrNotReady)
	if result := wait(t, ended); !errors.Is(result.err, vaultservice.ErrNotReady) {
		t.Fatalf("a verification when the vault locked: got %v, want ErrNotReady", result.err)
	}
}
