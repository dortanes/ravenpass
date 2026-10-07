package api

import (
	"context"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/ownerauth"
)

func TestConfirmRevealAsksDeviceAuthenticationWhereTheVaultOpensWithIt(t *testing.T) {
	service := newReadyService(t)
	declined := answerOwner(t, service, ownerauth.ErrCanceled)
	assertFailure(t, service.ConfirmReveal(context.Background(), "Wallet", ""), failureOwnerUnverified)
	approved := answerOwner(t, service, nil)
	if err := service.ConfirmReveal(context.Background(), "Wallet", "999999"); err != nil {
		t.Fatalf("an approved owner: %v", err)
	}
	if declined.times() != 1 || approved.times() != 1 {
		t.Fatalf("the owner was asked %d and %d times", declined.times(), approved.times())
	}
}

func TestConfirmRevealTakesThePINWhereDeviceAuthenticationIsOff(t *testing.T) {
	service, _ := pinVerifications(t)
	owner := answerOwner(t, service, nil)
	before := unlockMethodsOf(t, service).PINAttemptsLeft
	assertFailure(t, service.ConfirmReveal(context.Background(), "Wallet", "999999"), failurePINWrong)
	if left := unlockMethodsOf(t, service).PINAttemptsLeft; left != before-1 {
		t.Fatalf("a wrong PIN left %d attempts of %d", left, before)
	}
	if err := service.ConfirmReveal(context.Background(), "Wallet", confirmationPIN); err != nil {
		t.Fatalf("the right PIN: %v", err)
	}
	if owner.times() != 0 {
		t.Fatal("a vault without device authentication showed the system prompt")
	}
}

func TestConfirmRevealPassesWhereTheVaultOffersNoWayToAsk(t *testing.T) {
	service, device := newReadyServiceOnDevice(t)
	owner := answerOwner(t, service, nil)
	device.noDeviceOwner = true
	if methods := unlockMethodsOf(t, service); methods.PINSet || methods.BiometryAvailable {
		t.Fatalf("the vault still offers a way to ask: %+v", methods)
	}
	asked := owner.times()
	if err := service.ConfirmReveal(context.Background(), "Wallet", ""); err != nil {
		t.Fatalf("a vault without a way to ask: %v", err)
	}
	if owner.times() != asked {
		t.Fatal("a vault without device authentication showed the system prompt")
	}
}
