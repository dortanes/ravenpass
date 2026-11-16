package vault

import (
	"bytes"
	"errors"
	"testing"
)

func pinInputs(t *testing.T) ([]byte, []byte) {
	t.Helper()
	salt, err := NewPINSalt()
	if err != nil {
		t.Fatal(err)
	}
	secret := make([]byte, PINSecretSize)
	if err := randomBytes(secret); err != nil {
		t.Fatal(err)
	}
	return salt, secret
}

func TestAVaultKeyWrappedForAPINOpensWithThatPIN(t *testing.T) {
	session, _ := populatedSession(t)
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	salt, secret := pinInputs(t)
	key, err := DerivePINKey("123456", salt, secret)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := session.WrapDeviceKey(key[:])
	if err != nil {
		t.Fatal(err)
	}
	witness := WitnessFor(mustHead(t, session))

	again, err := DerivePINKey("123456", salt, secret)
	if err != nil {
		t.Fatal(err)
	}
	opened, _, err := OpenWithDevice(container, again[:], envelope, &witness, nil)
	if err != nil {
		t.Fatalf("the same PIN did not open the vault: %v", err)
	}
	opened.Lock()

	wrong, err := DerivePINKey("123457", salt, secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := OpenWithDevice(container, wrong[:], envelope, &witness, nil); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("a wrong PIN: got %v, want ErrAuthentication", err)
	}

	_, otherSecret := pinInputs(t)
	elsewhere, err := DerivePINKey("123456", salt, otherSecret)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(elsewhere[:], key[:]) {
		t.Fatal("the device secret made no difference to the derived key")
	}
	if _, _, err := OpenWithDevice(container, elsewhere[:], envelope, &witness, nil); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("the same PIN on another device: got %v, want ErrAuthentication", err)
	}
}

func TestTheSaltMakesEveryPINKeyDifferent(t *testing.T) {
	_, secret := pinInputs(t)
	first, _ := NewPINSalt()
	second, _ := NewPINSalt()
	one, err := DerivePINKey("654321", first, secret)
	if err != nil {
		t.Fatal(err)
	}
	other, err := DerivePINKey("654321", second, secret)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(one[:], other[:]) {
		t.Fatal("two salts derived the same key")
	}
}

func TestOnlyALetterOrDigitPINOfTheRightLengthIsAccepted(t *testing.T) {
	salt, secret := pinInputs(t)
	for name, pin := range map[string]string{
		"empty":             "",
		"too short digits":  "12345",
		"too short letters": "abcde",
		"too long digits":   "1234567890123",
		"too long letters":  "abcdefghijklm",
		"with spaces":       "123 456",
		"with a sign":       "+12345",
		"with symbols":      "12345!",
		"not ASCII digits":  "１２３４５６",
		"not ASCII letters": "密码1234",
	} {
		if ValidPIN(pin) {
			t.Errorf("%s (%q) was accepted", name, pin)
		}
		if _, err := DerivePINKey(pin, salt, secret); !errors.Is(err, ErrInvalidPIN) {
			t.Errorf("%s (%q): got %v, want ErrInvalidPIN", name, pin, err)
		}
	}
	for _, pin := range []string{
		"000000",
		"123456",
		"123456789012",
		"abcdef",
		"ABCDEF",
		"Pass123",
		"PinCode12",
		"12345a",
	} {
		if !ValidPIN(pin) {
			t.Errorf("%q was refused", pin)
		}
	}
}

func TestPINIsCaseSensitive(t *testing.T) {
	salt, secret := pinInputs(t)
	upper, err := DerivePINKey("Pass123", salt, secret)
	if err != nil {
		t.Fatal(err)
	}
	lower, err := DerivePINKey("pass123", salt, secret)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(upper[:], lower[:]) {
		t.Fatal("PIN derivation should be case-sensitive")
	}
}

func TestDerivationRefusesInputsOfTheWrongSize(t *testing.T) {
	salt, secret := pinInputs(t)
	if _, err := DerivePINKey("123456", salt[:PINSaltSize-1], secret); !errors.Is(err, ErrInvalidPIN) {
		t.Errorf("a short salt: got %v, want ErrInvalidPIN", err)
	}
	if _, err := DerivePINKey("123456", salt, secret[:PINSecretSize-1]); !errors.Is(err, ErrInvalidPIN) {
		t.Errorf("a short device secret: got %v, want ErrInvalidPIN", err)
	}
}

func mustHead(t *testing.T, session *Session) Head {
	t.Helper()
	head, err := session.Head()
	if err != nil {
		t.Fatal(err)
	}
	return head
}
