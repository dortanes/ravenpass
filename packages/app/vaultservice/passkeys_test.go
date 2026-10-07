package vaultservice

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/authenticator"
	"github.com/dortanes/ravenpass/packages/vault"
)

const (
	passkeyOrigin = "https://login.example.com"
	passkeyRPID   = "example.com"

	flagUserPresent    = 0x01
	flagUserVerified   = 0x04
	flagBackupEligible = 0x08
	flagBackedUp       = 0x10
	flagAttestedData   = 0x40

	appOrigin = "android:apk-key-hash:x2xzU8dM3zY6Gq-nN0AjL0wYfF5sT0c3Z1d9Kq7pQ1E"
)

var (
	challenge = []byte("a challenge of at least sixteen bytes")
	// platformHash stands for the SHA-256 hash of client data a platform built.
	platformHash = sha256.Sum256([]byte("client data a platform built"))
)

// testPasskey is a passkey for rpID with a fresh key, credential ID and the given counter.
func testPasskey(t *testing.T, rpID, userName string, discoverable bool, counter uint32) vault.Passkey {
	t.Helper()
	key, err := authenticator.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	return vault.Passkey{
		CredentialID: key.CredentialID, RPID: rpID, UserHandle: []byte("user-" + userName), UserName: userName,
		PrivateKey: key.PrivateKey, Counter: counter, Discoverable: discoverable, CreatedAt: time.Unix(1_790_000_000, 0),
	}
}

func choiceIDs(choices []PasskeyChoice) [][]byte {
	ids := make([][]byte, len(choices))
	for i, choice := range choices {
		ids[i] = choice.CredentialID
	}
	return ids
}

func passkeyTargetIDs(targets PasskeyTargets) []vault.ID {
	ids := make([]vault.ID, len(targets.Targets))
	for i, target := range targets.Targets {
		ids[i] = target.ID
	}
	return ids
}

func equalIDs(a, b [][]byte) bool {
	return slices.EqualFunc(a, b, bytes.Equal)
}

// newPasskeyCreation asks for a passkey for alex@example.com on passkeyOrigin.
func newPasskeyCreation(target vault.ID) PasskeyCreation {
	return PasskeyCreation{
		Origin: passkeyOrigin, RPID: passkeyRPID, RPName: "Example",
		User:      PasskeyUser{Handle: []byte{1, 2, 3, 4}, Name: "alex@example.com", DisplayName: "Alex"},
		Challenge: challenge, Target: target,
	}
}

// authenticatorData checks the rpIdHash and returns the flags and sign counter (WebAuthn §6.1).
func authenticatorData(t *testing.T, data []byte, rpID string) (byte, uint32) {
	t.Helper()
	hash := sha256.Sum256([]byte(rpID))
	if len(data) < 37 || !bytes.Equal(data[:32], hash[:]) {
		t.Fatalf("authenticator data %x is not for %s", data, rpID)
	}
	return data[32], binary.BigEndian.Uint32(data[33:37])
}

// verifyAssertion checks the signature over authenticator data || clientDataHash with an SPKI key.
func verifyAssertion(t *testing.T, publicKey []byte, assertion PasskeyAssertion, clientDataHash [32]byte) {
	t.Helper()
	parsed, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("the public key is a %T", parsed)
	}
	signed := sha256.Sum256(append(bytes.Clone(assertion.AuthenticatorData), clientDataHash[:]...))
	if !ecdsa.VerifyASN1(key, signed[:], assertion.Signature) {
		t.Fatal("the signature does not verify with the passkey's public key")
	}
}

func clientDataOf(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

func TestSignInListsPasskeysForExactlyTheRelyingParty(t *testing.T) {
	service, _, _ := captureVault(t)
	shown := testPasskey(t, passkeyRPID, "alex", true, 0)
	hidden := testPasskey(t, passkeyRPID, "", false, 0)
	sub := testPasskey(t, "login.example.com", "sub", true, 0)
	other := testPasskey(t, "example.org", "other", true, 0)
	credential := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Login: "alex.l", Passkeys: []vault.Passkey{shown, hidden, sub, other}})

	rpID, choices, err := service.SignInPasskeys(passkeyOrigin, passkeyRPID, nil)
	if err != nil || rpID != passkeyRPID {
		t.Fatalf("relying party %q, error = %v", rpID, err)
	}
	if !equalIDs(choiceIDs(choices), [][]byte{shown.CredentialID}) {
		t.Fatalf("without an allow list the sign-in offers %+v", choices)
	}
	if choices[0].Credential != credential || choices[0].Account != "alex" || choices[0].Label != "Example" {
		t.Fatalf("choice = %+v", choices[0])
	}

	_, choices, err = service.SignInPasskeys(passkeyOrigin, passkeyRPID, [][]byte{hidden.CredentialID, other.CredentialID, sub.CredentialID})
	if err != nil || !equalIDs(choiceIDs(choices), [][]byte{hidden.CredentialID}) {
		t.Fatalf("an allow list offers %+v, error = %v", choices, err)
	}
	if choices[0].Account != "alex.l" {
		t.Fatalf("a passkey without a user name signs in as %q, want the login", choices[0].Account)
	}

	rpID, choices, err = service.SignInPasskeys(passkeyOrigin, "", nil)
	if err != nil || rpID != "login.example.com" || !equalIDs(choiceIDs(choices), [][]byte{sub.CredentialID}) {
		t.Fatalf("the origin's own host as relying party: %q offers %+v, error = %v", rpID, choices, err)
	}
}

func TestSignInPasskeysComeMostRecentlyUsedFirstThenByLabel(t *testing.T) {
	service, _, _ := captureVault(t)
	var ids []vault.ID
	var passkeys []vault.Passkey
	for _, label := range []string{"Charlie", "alpha", "Bravo"} {
		passkey := testPasskey(t, passkeyRPID, label, true, 0)
		passkeys = append(passkeys, passkey)
		ids = append(ids, createTestCredential(t, service, vault.CredentialInput{Label: label, Passkeys: []vault.Passkey{passkey}}))
	}
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: ids[2], lastUsedAt: 2000}}})
	_, choices, err := service.SignInPasskeys(passkeyOrigin, passkeyRPID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]byte{passkeys[2].CredentialID, passkeys[1].CredentialID, passkeys[0].CredentialID}; !equalIDs(choiceIDs(choices), want) {
		t.Fatalf("choices = %+v", choices)
	}
}

func TestPasskeyTargetsMatchTheOriginHoldersFirstAndReportExclusions(t *testing.T) {
	service, _, _ := captureVault(t)
	older := createTestCredential(t, service, vault.CredentialInput{Label: "Older", Websites: []string{passkeyOrigin}, Login: "sam"})
	recent := createTestCredential(t, service, vault.CredentialInput{Label: "Recent", Websites: []string{"https://example.com"}, Login: "kim"})
	holder := createTestCredential(t, service, vault.CredentialInput{Label: "Zulu", Websites: []string{passkeyOrigin}, Email: "Alex@Example.com", Tags: []string{"Work"}})
	createTestCredential(t, service, vault.CredentialInput{Label: "Elsewhere", Websites: []string{"https://example.org"}, Login: "alex@example.com"})
	full := make([]vault.Passkey, vault.MaxCredentialPasskeys)
	for i := range full {
		full[i] = testPasskey(t, "example.org", "full", true, 0)
	}
	createTestCredential(t, service, vault.CredentialInput{Label: "Full", Websites: []string{passkeyOrigin}, Login: "alex@example.com", Passkeys: full})
	service.device.rememberUsage(usageLog{entries: []usageEntry{{id: recent, lastUsedAt: 5000}}})

	targets, err := service.PasskeyTargets(passkeyOrigin, passkeyRPID, "alex@example.com", [][]byte{full[0].CredentialID})
	if err != nil {
		t.Fatal(err)
	}
	if targets.RPID != passkeyRPID || targets.Excluded || !slices.Equal(passkeyTargetIDs(targets), []vault.ID{holder, recent, older}) {
		t.Fatalf("targets = %+v", targets)
	}
	if !reflect.DeepEqual(targets.Targets[0], PasskeyTarget{ID: holder, Label: "Zulu", Account: "Alex@Example.com", Tags: []string{"Work"}}) {
		t.Fatalf("holder target = %+v", targets.Targets[0])
	}

	held := testPasskey(t, passkeyRPID, "alex@example.com", true, 0)
	createTestCredential(t, service, vault.CredentialInput{Label: "Held", Passkeys: []vault.Passkey{held}})
	targets, err = service.PasskeyTargets(passkeyOrigin, passkeyRPID, "alex@example.com", [][]byte{[]byte("an unrelated credential"), held.CredentialID})
	if err != nil || !targets.Excluded {
		t.Fatalf("a held excluded passkey: %+v, error = %v", targets, err)
	}
	targets, err = service.PasskeyTargets("https://example.org", "", "alex", nil)
	if err != nil || len(targets.Targets) != 1 || targets.Targets == nil || targets.Excluded {
		t.Fatalf("targets for another site = %+v, error = %v", targets, err)
	}
}

func TestPasskeyRequestsCheckTheRelyingPartyFirst(t *testing.T) {
	service := newTestService(t, &memoryFiles{}, newMemoryKeys())
	for origin, want := range map[string]error{
		"http://example.com":  authenticator.ErrInvalidOrigin,
		"https://example.com": authenticator.ErrInvalidRelyingParty,
	} {
		if _, _, err := service.SignInPasskeys(origin, "other.com", nil); !errors.Is(err, want) {
			t.Fatalf("sign-in options from %s: got %v, want %v", origin, err, want)
		}
		if _, err := service.PasskeyTargets(origin, "other.com", "", nil); !errors.Is(err, want) {
			t.Fatalf("targets from %s: got %v, want %v", origin, err, want)
		}
		if _, err := service.CreatePasskey(PasskeyCreation{Origin: origin, RPID: "other.com"}, ""); !errors.Is(err, want) {
			t.Fatalf("creation from %s: got %v, want %v", origin, err, want)
		}
		if _, err := service.SignPasskey(PasskeySignIn{Origin: origin, RPID: "other.com"}); !errors.Is(err, want) {
			t.Fatalf("signing from %s: got %v, want %v", origin, err, want)
		}
	}
	if _, _, err := service.SignInPasskeys(passkeyOrigin, passkeyRPID, nil); !errors.Is(err, ErrNotReady) {
		t.Fatalf("a locked vault: got %v, want ErrNotReady", err)
	}
}

func TestANewPasskeyIsSavedInANewCredentialBeforeTheSiteReceivesIt(t *testing.T) {
	service, _, _ := captureVault(t)
	work := createTestGroup(t, service, "Work")
	creation := newPasskeyCreation(vault.ID{})
	creation.Verified = true
	created, err := service.CreatePasskey(creation, work.String())
	if err != nil {
		t.Fatal(err)
	}
	entry := listedEntry(t, service, created.Credential)
	if entry.Label != "Example" || !slices.Equal(entry.Sites, []string{"login.example.com"}) || !slices.Equal(entry.Groups, []vault.ID{work}) {
		t.Fatalf("new credential = %+v", entry)
	}
	if len(entry.Passkeys) != 1 || !bytes.Equal(entry.Passkeys[0].CredentialID, created.CredentialID) || entry.Passkeys[0].RPID != passkeyRPID || !entry.Passkeys[0].Discoverable {
		t.Fatalf("its passkeys = %+v", entry.Passkeys)
	}
	credential := readTestCredential(t, service, created.Credential)
	if credential.Email != "alex@example.com" || credential.Login != "" || !slices.Equal(credential.Websites, []string{"login.example.com"}) {
		t.Fatalf("credential = %+v", credential.CredentialInput)
	}
	passkey := credential.Passkeys[0]
	if !bytes.Equal(passkey.UserHandle, []byte{1, 2, 3, 4}) || passkey.UserName != "alex@example.com" || passkey.UserDisplayName != "Alex" || passkey.Counter != 0 || passkey.PrivateKey != nil {
		t.Fatalf("stored passkey = %+v", passkey)
	}
	if age := time.Since(passkey.CreatedAt); age < 0 || age > time.Minute {
		t.Fatalf("created at %v", passkey.CreatedAt)
	}
	fields := clientDataOf(t, created.ClientData)
	if fields["type"] != "webauthn.create" || fields["origin"] != passkeyOrigin {
		t.Fatalf("client data = %s", created.ClientData)
	}
	authData := created.Attestation.AuthenticatorData
	flags, counter := authenticatorData(t, authData, passkeyRPID)
	if flags != flagUserPresent|flagUserVerified|flagBackupEligible|flagAttestedData || counter != 0 {
		t.Fatalf("flags %08b, counter %d", flags, counter)
	}
	// rpIdHash (32), flags, counter (4), AAGUID (16).
	if !bytes.Equal(authData[37:53], authenticator.AAGUID[:]) {
		t.Fatalf("AAGUID % x", authData[37:53])
	}
	if created.Attestation.Algorithm != authenticator.AlgorithmES256 || len(created.Attestation.PublicKey) == 0 || len(created.Attestation.AttestationObject) == 0 {
		t.Fatalf("attestation = %+v", created.Attestation)
	}

	creation = newPasskeyCreation(vault.ID{})
	creation.RPName, creation.User.Name = " ", "alex"
	created, err = service.CreatePasskey(creation, "")
	if err != nil {
		t.Fatal(err)
	}
	if entry := listedEntry(t, service, created.Credential); entry.Label != passkeyRPID || entry.Detail != "alex" || len(entry.Groups) != 0 {
		t.Fatalf("a credential for a site without a name = %+v", entry)
	}
}

func TestANewPasskeyJoinsATargetThatStillMatches(t *testing.T) {
	service, files, _ := captureVault(t)
	target := createTestCredential(t, service, vault.CredentialInput{Label: "Mine", Websites: []string{"https://example.com"}, Login: "alex"})
	created, err := service.CreatePasskey(newPasskeyCreation(target), "")
	if err != nil {
		t.Fatal(err)
	}
	if created.Credential != target {
		t.Fatalf("the passkey joined %v, want %v", created.Credential, target)
	}
	credential := readTestCredential(t, service, target)
	if credential.Login != "alex" || credential.Email != "" || len(credential.Passkeys) != 1 || !bytes.Equal(credential.Passkeys[0].CredentialID, created.CredentialID) {
		t.Fatalf("target after the passkey joined = %+v", credential.CredentialInput)
	}

	elsewhere := createTestCredential(t, service, vault.CredentialInput{Label: "Elsewhere", Websites: []string{"https://example.org"}})
	group := createTestGroup(t, service, "Work")
	note, err := service.CreateNote(vault.NoteInput{Label: "Note", Body: "text"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	container := bytes.Clone(files.data)
	for target, want := range map[vault.ID]error{
		elsewhere: ErrNoMatch,
		{0xee}:    vault.ErrNotFound,
		group:     vault.ErrNotFound,
		note:      vault.ErrNotFound,
	} {
		if created, err := service.CreatePasskey(newPasskeyCreation(target), ""); !errors.Is(err, want) || created.CredentialID != nil {
			t.Fatalf("target %v: got %+v, %v, want %v", target, created, err, want)
		}
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("a refused target wrote to the vault")
	}
}

func TestAPasskeyCreationIsRefusedForExclusionsAndFullTargets(t *testing.T) {
	service, files, _ := captureVault(t)
	held := testPasskey(t, passkeyRPID, "alex", true, 0)
	full := make([]vault.Passkey, vault.MaxCredentialPasskeys)
	full[0] = held
	for i := 1; i < len(full); i++ {
		full[i] = testPasskey(t, "example.org", "other", true, 0)
	}
	target := createTestCredential(t, service, vault.CredentialInput{Label: "Full", Websites: []string{passkeyOrigin}, Passkeys: full})

	creation := newPasskeyCreation(vault.ID{})
	creation.Exclude = [][]byte{held.CredentialID}
	if _, err := service.CreatePasskey(creation, ""); !errors.Is(err, ErrPasskeyExcluded) {
		t.Fatalf("an excluded passkey: got %v, want ErrPasskeyExcluded", err)
	}
	creation.RPID = "login.example.com"
	if _, err := service.CreatePasskey(creation, ""); err != nil {
		t.Fatalf("an excluded ID for another relying party: %v", err)
	}
	container := bytes.Clone(files.data)
	if _, err := service.CreatePasskey(newPasskeyCreation(target), ""); !errors.Is(err, vault.ErrPasskeysFull) {
		t.Fatalf("a full target: got %v, want ErrPasskeysFull", err)
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("a refused creation wrote to the vault")
	}
}

func TestASiteReceivesNoPasskeyWhenSavingFails(t *testing.T) {
	service, files, _ := captureVault(t)
	target := createTestCredential(t, service, vault.CredentialInput{Label: "Mine", Websites: []string{passkeyOrigin}})
	before := len(listedEntry(t, service, target).Passkeys)
	files.maxBytes = len(files.data)
	for _, creation := range []PasskeyCreation{newPasskeyCreation(vault.ID{}), newPasskeyCreation(target)} {
		created, err := service.CreatePasskey(creation, "")
		if err == nil || created.CredentialID != nil || created.ClientData != nil || created.Attestation.AttestationObject != nil {
			t.Fatalf("a failed save answered %+v, error = %v", created, err)
		}
	}
	entries, err := service.List()
	if err != nil || len(entries) != 1 || len(entries[0].Passkeys) != before {
		t.Fatalf("after failed saves the vault lists %+v, error = %v", entries, err)
	}
}

func TestSigningWithANewPasskeyVerifiesWithItsPublicKey(t *testing.T) {
	service, files, _ := captureVault(t)
	created, err := service.CreatePasskey(newPasskeyCreation(vault.ID{}), "")
	if err != nil {
		t.Fatal(err)
	}
	container := bytes.Clone(files.data)
	signIn := PasskeySignIn{Origin: passkeyOrigin, RPID: passkeyRPID, Challenge: challenge, Credential: created.Credential, CredentialID: created.CredentialID}
	for _, verified := range []bool{false, true} {
		signIn.Verified = verified
		assertion, err := service.SignPasskey(signIn)
		if err != nil {
			t.Fatal(err)
		}
		verifyAssertion(t, created.Attestation.PublicKey, assertion, sha256.Sum256(assertion.ClientData))
		flags, counter := authenticatorData(t, assertion.AuthenticatorData, passkeyRPID)
		if counter != 0 || flags&flagUserPresent == 0 || flags&flagBackupEligible == 0 || flags&flagUserVerified != 0 != verified || flags&(flagBackedUp|flagAttestedData) != 0 {
			t.Fatalf("verified %t: flags %08b, counter %d", verified, flags, counter)
		}
		if !bytes.Equal(assertion.CredentialID, created.CredentialID) || !bytes.Equal(assertion.UserHandle, []byte{1, 2, 3, 4}) {
			t.Fatalf("assertion = %+v", assertion)
		}
		if fields := clientDataOf(t, assertion.ClientData); fields["type"] != "webauthn.get" || fields["origin"] != passkeyOrigin {
			t.Fatalf("client data = %s", assertion.ClientData)
		}
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("signing with a zero counter wrote to the vault")
	}
	if used, err := service.Usage(); err != nil || used[created.Credential] == 0 {
		t.Fatalf("the sign-in was not recorded as a use: %v, error = %v", used, err)
	}
}

func TestNoPrivateKeyReachesTheSite(t *testing.T) {
	service, _, _ := captureVault(t)
	created, err := service.CreatePasskey(newPasskeyCreation(vault.ID{}), "")
	if err != nil {
		t.Fatal(err)
	}
	assertion, err := service.SignPasskey(PasskeySignIn{Origin: passkeyOrigin, RPID: passkeyRPID, Challenge: challenge, Credential: created.Credential, CredentialID: created.CredentialID})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := service.session.ReadPasskey(created.Credential, created.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(stored.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := parsed.(*ecdsa.PrivateKey).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	for _, sent := range [][]byte{
		created.CredentialID, created.ClientData, created.Attestation.AttestationObject, created.Attestation.AuthenticatorData, created.Attestation.PublicKey,
		assertion.CredentialID, assertion.ClientData, assertion.AuthenticatorData, assertion.Signature, assertion.UserHandle,
	} {
		if bytes.Contains(sent, stored.PrivateKey) || bytes.Contains(sent, scalar) {
			t.Fatalf("%x carries the private key", sent)
		}
	}
}

func TestSigningAdvancesANonzeroCounterBeforeItSigns(t *testing.T) {
	service, files, _ := captureVault(t)
	passkey := testPasskey(t, passkeyRPID, "alex", true, 41)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Imported", Passkeys: []vault.Passkey{passkey}})
	signIn := PasskeySignIn{Origin: passkeyOrigin, RPID: passkeyRPID, Challenge: challenge, Credential: id, CredentialID: passkey.CredentialID}
	for _, want := range []uint32{42, 43} {
		container := bytes.Clone(files.data)
		assertion, err := service.SignPasskey(signIn)
		if err != nil {
			t.Fatal(err)
		}
		if _, counter := authenticatorData(t, assertion.AuthenticatorData, passkeyRPID); counter != want {
			t.Fatalf("assertion counter %d, want %d", counter, want)
		}
		if bytes.Equal(files.data, container) {
			t.Fatal("the advanced counter was not saved")
		}
	}

	files.maxBytes = 1
	container := bytes.Clone(files.data)
	assertion, err := service.SignPasskey(signIn)
	if err == nil || assertion.Signature != nil || assertion.AuthenticatorData != nil {
		t.Fatalf("a failed save signed %+v, error = %v", assertion, err)
	}
	files.maxBytes = 0
	if !bytes.Equal(files.data, container) {
		t.Fatal("a failed save changed the vault")
	}
	assertion, err = service.SignPasskey(signIn)
	if err != nil {
		t.Fatal(err)
	}
	if _, counter := authenticatorData(t, assertion.AuthenticatorData, passkeyRPID); counter != 44 {
		t.Fatalf("after a failed save the counter is %d, want 44", counter)
	}
}

func TestSigningRefusesAPasskeyForAnotherRelyingParty(t *testing.T) {
	service, _, keys := captureVault(t)
	passkey := testPasskey(t, passkeyRPID, "alex", true, 0)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Passkeys: []vault.Passkey{passkey}})
	for name, signIn := range map[string]PasskeySignIn{
		"the origin's host as relying party": {Origin: passkeyOrigin, RPID: "login.example.com", Credential: id, CredentialID: passkey.CredentialID},
		"another credential":                 {Origin: passkeyOrigin, RPID: passkeyRPID, Credential: vault.ID{0xee}, CredentialID: passkey.CredentialID},
		"another credential ID":              {Origin: passkeyOrigin, RPID: passkeyRPID, Credential: id, CredentialID: []byte("not the passkey's credential ID")},
	} {
		if assertion, err := service.SignPasskey(signIn); !errors.Is(err, vault.ErrNotFound) || assertion.Signature != nil {
			t.Fatalf("%s: signed %+v, error = %v", name, assertion, err)
		}
	}
	if _, err := service.SignPasskey(PasskeySignIn{Origin: "https://example.org", RPID: passkeyRPID, Credential: id, CredentialID: passkey.CredentialID}); !errors.Is(err, authenticator.ErrInvalidRelyingParty) {
		t.Fatalf("a sign-in from another site: got %v, want ErrInvalidRelyingParty", err)
	}
	if len(keys.usage) != 0 {
		t.Fatal("a refused sign-in was recorded as a use")
	}
}

func TestAnEditRemovingAPasskeyLeavesTheCredential(t *testing.T) {
	service, _, _ := captureVault(t)
	kept, removed := testPasskey(t, passkeyRPID, "kept", true, 0), testPasskey(t, passkeyRPID, "removed", true, 0)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Login: "alex", Passkeys: []vault.Passkey{kept, removed}})
	label := "Renamed"
	if err := service.EditCredential(id, vault.CredentialPatch{Label: &label, RemovePasskeys: [][]byte{removed.CredentialID}}); err != nil {
		t.Fatal(err)
	}
	credential := readTestCredential(t, service, id)
	if credential.Label != label || credential.Login != "alex" || len(credential.Passkeys) != 1 || !bytes.Equal(credential.Passkeys[0].CredentialID, kept.CredentialID) {
		t.Fatalf("after removal = %+v", credential.CredentialInput)
	}
	if err := service.EditCredential(id, vault.CredentialPatch{RemovePasskeys: [][]byte{removed.CredentialID}}); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("removing it again: got %v, want ErrNotFound", err)
	}
	if _, err := service.SignPasskey(PasskeySignIn{Origin: passkeyOrigin, RPID: passkeyRPID, Challenge: challenge, Credential: id, CredentialID: removed.CredentialID}); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("signing with a removed passkey: got %v, want ErrNotFound", err)
	}
}

// newPlatformCreation is a platform request, over its own client data hash, for a passkey on passkeyRPID.
func newPlatformCreation() PlatformPasskeyCreation {
	return PlatformPasskeyCreation{
		RPID: passkeyRPID, RPName: "Example",
		User:   PasskeyUser{Handle: []byte{1, 2, 3, 4}, Name: "alex@example.com", DisplayName: "Alex"},
		Client: PlatformClient{Hash: &platformHash},
	}
}

func TestAPlatformIsOfferedThePasskeysOfExactlyItsRelyingPartyWithTheirUsers(t *testing.T) {
	service, _, _ := captureVault(t)
	shown := testPasskey(t, passkeyRPID, "alex", true, 0)
	shown.UserDisplayName = "Alex"
	hidden := testPasskey(t, passkeyRPID, "", false, 0)
	sub := testPasskey(t, "login.example.com", "sub", true, 0)
	credential := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Login: "alex.l", Passkeys: []vault.Passkey{shown, hidden, sub}})

	choices, err := service.PasskeysFor("Example.COM", nil)
	if err != nil || !equalIDs(choiceIDs(choices), [][]byte{shown.CredentialID}) {
		t.Fatalf("without an allow list the platform is offered %+v, error = %v", choices, err)
	}
	if choice := choices[0]; choice.Credential != credential || !bytes.Equal(choice.UserHandle, shown.UserHandle) || choice.Account != "alex" || choice.DisplayName != "Alex" || choice.Label != "Example" {
		t.Fatalf("choice = %+v", choice)
	}
	choices, err = service.PasskeysFor(passkeyRPID, [][]byte{hidden.CredentialID, sub.CredentialID})
	if err != nil || !equalIDs(choiceIDs(choices), [][]byte{hidden.CredentialID}) || choices[0].Account != "alex.l" {
		t.Fatalf("an allow list offers %+v, error = %v", choices, err)
	}
	for _, rpID := range []string{"", "com", "example.com:443", "https://example.com"} {
		if _, err := service.PasskeysFor(rpID, nil); !errors.Is(err, authenticator.ErrInvalidRelyingParty) {
			t.Fatalf("relying party %q: got %v, want ErrInvalidRelyingParty", rpID, err)
		}
	}
	service.Lock()
	if _, err := service.PasskeysFor(passkeyRPID, nil); !errors.Is(err, ErrNotReady) {
		t.Fatalf("a locked vault: got %v, want ErrNotReady", err)
	}
}

func TestAPlatformChecksAHeldPasskeyAndExclusionsBeforeAskingTheOwner(t *testing.T) {
	service, _, _ := captureVault(t)
	held := testPasskey(t, passkeyRPID, "alex", true, 0)
	credential := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Passkeys: []vault.Passkey{held}})
	other := createTestCredential(t, service, vault.CredentialInput{Label: "Other"})

	choice, err := service.HeldPasskey("Example.COM", credential, held.CredentialID)
	if err != nil || choice.Credential != credential || choice.Account != "alex" {
		t.Fatalf("held passkey = %+v, error = %v", choice, err)
	}
	for name, test := range map[string]struct {
		rpID         string
		credential   vault.ID
		credentialID []byte
	}{
		"another credential":    {passkeyRPID, other, held.CredentialID},
		"another passkey":       {passkeyRPID, credential, []byte("unknown")},
		"another relying party": {"login.example.com", credential, held.CredentialID},
	} {
		if _, err := service.HeldPasskey(test.rpID, test.credential, test.credentialID); !errors.Is(err, vault.ErrNotFound) {
			t.Fatalf("%s: got %v, want ErrNotFound", name, err)
		}
	}

	for exclude, want := range map[string]error{
		"":                        nil,
		"unknown":                 nil,
		string(held.CredentialID): ErrPasskeyExcluded,
	} {
		var listed [][]byte
		if exclude != "" {
			listed = [][]byte{[]byte(exclude)}
		}
		if err := service.CheckExclusions(passkeyRPID, listed); !errors.Is(err, want) {
			t.Fatalf("excluding %x: got %v, want %v", exclude, err, want)
		}
	}
	if err := service.CheckExclusions("login.example.com", [][]byte{held.CredentialID}); err != nil {
		t.Fatalf("an excluded ID for another relying party: %v", err)
	}
	if err := service.CheckExclusions("com", nil); !errors.Is(err, authenticator.ErrInvalidRelyingParty) {
		t.Fatalf("a relying party no one can use: got %v", err)
	}
	service.Lock()
	if err := service.CheckExclusions(passkeyRPID, nil); !errors.Is(err, ErrNotReady) {
		t.Fatalf("a locked vault: got %v, want ErrNotReady", err)
	}
}

func TestAPlatformCreationIsSavedInANewCredentialForTheRelyingParty(t *testing.T) {
	service, _, _ := captureVault(t)
	work := createTestGroup(t, service, "Work")
	creation := newPlatformCreation()
	creation.Verified = true
	created, err := service.CreatePlatformPasskey(creation, work.String())
	if err != nil {
		t.Fatal(err)
	}
	if created.ClientData != nil {
		t.Fatalf("a creation over the platform's hash answered client data %s", created.ClientData)
	}
	entry := listedEntry(t, service, created.Credential)
	if entry.Label != "Example" || !slices.Equal(entry.Sites, []string{passkeyRPID}) || !slices.Equal(entry.Groups, []vault.ID{work}) {
		t.Fatalf("new credential = %+v", entry)
	}
	if len(entry.Passkeys) != 1 {
		t.Fatalf("its passkeys = %+v", entry.Passkeys)
	}
	if face := entry.Passkeys[0]; !bytes.Equal(face.CredentialID, created.CredentialID) || face.RPID != passkeyRPID || !bytes.Equal(face.UserHandle, []byte{1, 2, 3, 4}) || face.UserDisplayName != "Alex" {
		t.Fatalf("its passkey = %+v", face)
	}
	if credential := readTestCredential(t, service, created.Credential); credential.Email != "alex@example.com" || !slices.Equal(credential.Websites, []string{passkeyRPID}) {
		t.Fatalf("credential = %+v", credential.CredentialInput)
	}

	authData := created.Attestation.AuthenticatorData
	flags, counter := authenticatorData(t, authData, passkeyRPID)
	if flags != flagUserPresent|flagUserVerified|flagBackupEligible|flagBackedUp|flagAttestedData || counter != 0 {
		t.Fatalf("flags %08b, counter %d", flags, counter)
	}
	// rpIdHash (32), flags, counter (4), AAGUID (16), credential ID length (2), credential ID.
	attested := authData[37:]
	if !bytes.Equal(attested[:16], authenticator.AAGUID[:]) || binary.BigEndian.Uint16(attested[16:18]) != uint16(len(created.CredentialID)) || !bytes.Equal(attested[18:18+len(created.CredentialID)], created.CredentialID) {
		t.Fatalf("attested credential data % x", attested)
	}
	if !bytes.HasSuffix(created.Attestation.AttestationObject, authData) {
		t.Fatal("the attestation object does not end with the authenticator data")
	}

	creation = newPlatformCreation()
	creation.RPName = " "
	creation.User = PasskeyUser{Handle: []byte{5, 6, 7, 8}, Name: "sam@example.com"}
	created, err = service.CreatePlatformPasskey(creation, "")
	if err != nil {
		t.Fatal(err)
	}
	if entry := listedEntry(t, service, created.Credential); entry.Label != passkeyRPID || len(entry.Groups) != 0 {
		t.Fatalf("a credential for a relying party without a name = %+v", entry)
	}
}

func TestAPlatformCreationIsRefusedForExclusionsAndWithoutClientData(t *testing.T) {
	service, files, _ := captureVault(t)
	held := testPasskey(t, passkeyRPID, "alex", true, 0)
	createTestCredential(t, service, vault.CredentialInput{Label: "Held", Passkeys: []vault.Passkey{held}})
	container := bytes.Clone(files.data)

	excluded := newPlatformCreation()
	excluded.Exclude = [][]byte{held.CredentialID}
	missing := newPlatformCreation()
	missing.Client = PlatformClient{}
	unchallenged := newPlatformCreation()
	unchallenged.Client = PlatformClient{Origin: appOrigin}
	public := newPlatformCreation()
	public.RPID = "com"
	for name, test := range map[string]struct {
		creation PlatformPasskeyCreation
		want     error
	}{
		"an excluded passkey":            {excluded, ErrPasskeyExcluded},
		"no client data":                 {missing, vault.ErrInvalidInput},
		"an origin without a challenge":  {unchallenged, vault.ErrInvalidInput},
		"a relying party no one can use": {public, authenticator.ErrInvalidRelyingParty},
	} {
		if created, err := service.CreatePlatformPasskey(test.creation, ""); !errors.Is(err, test.want) || created.CredentialID != nil {
			t.Fatalf("%s: got %+v, %v, want %v", name, created, err, test.want)
		}
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("a refused creation wrote to the vault")
	}
	excluded.RPID = "login.example.com"
	if _, err := service.CreatePlatformPasskey(excluded, ""); err != nil {
		t.Fatalf("an excluded ID for another relying party: %v", err)
	}
}

func TestARetriedPlatformCreationJoinsTheCredentialHoldingTheAccount(t *testing.T) {
	service, files, _ := captureVault(t)
	first, err := service.CreatePlatformPasskey(newPlatformCreation(), "")
	if err != nil {
		t.Fatal(err)
	}
	sameHandle := newPlatformCreation()
	sameHandle.User.Name = "alex.renamed@example.com"
	sameName := newPlatformCreation()
	sameName.User.Handle = []byte{9, 9, 9, 9}
	for name, creation := range map[string]PlatformPasskeyCreation{"the same user handle": sameHandle, "the same user name": sameName} {
		created, err := service.CreatePlatformPasskey(creation, "")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if created.Credential != first.Credential {
			t.Fatalf("%s: the passkey was saved in %v, want the holder %v", name, created.Credential, first.Credential)
		}
	}
	holder := readTestCredential(t, service, first.Credential)
	if len(holder.Passkeys) != 3 || holder.Label != "Example" || !slices.Equal(holder.Websites, []string{passkeyRPID}) {
		t.Fatalf("the holder after two retries = %+v", holder.CredentialInput)
	}

	retried := newPlatformCreation()
	retried.Exclude = [][]byte{first.CredentialID}
	container := bytes.Clone(files.data)
	if _, err := service.CreatePlatformPasskey(retried, ""); !errors.Is(err, ErrPasskeyExcluded) {
		t.Fatalf("a retry excluding the held passkey: got %v, want ErrPasskeyExcluded", err)
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("a refused retry wrote to the vault")
	}

	otherUser := newPlatformCreation()
	otherUser.User = PasskeyUser{Handle: []byte{5, 6, 7, 8}, Name: "sam@example.com"}
	otherParty := newPlatformCreation()
	otherParty.RPID = "login.example.com"
	for name, creation := range map[string]PlatformPasskeyCreation{"another user": otherUser, "another relying party": otherParty} {
		created, err := service.CreatePlatformPasskey(creation, "")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if created.Credential == first.Credential {
			t.Fatalf("%s: the passkey joined the credential of another account", name)
		}
	}
}

func TestAPlatformCreationForAFullHolderIsSavedInANewCredential(t *testing.T) {
	service, _, _ := captureVault(t)
	full := make([]vault.Passkey, vault.MaxCredentialPasskeys)
	for i := range full {
		full[i] = testPasskey(t, passkeyRPID, "alex@example.com", true, 0)
	}
	holder := createTestCredential(t, service, vault.CredentialInput{Label: "Full", Passkeys: full})
	created, err := service.CreatePlatformPasskey(newPlatformCreation(), "")
	if err != nil {
		t.Fatal(err)
	}
	if created.Credential == holder || len(listedEntry(t, service, created.Credential).Passkeys) != 1 {
		t.Fatalf("a creation for the account of a full credential was saved in %v", created.Credential)
	}
}

func TestAPlatformSignInOverItsHashVerifiesAgainstTheAuthenticatorDataAndTheHash(t *testing.T) {
	service, files, _ := captureVault(t)
	created, err := service.CreatePlatformPasskey(newPlatformCreation(), "")
	if err != nil {
		t.Fatal(err)
	}
	container := bytes.Clone(files.data)
	signIn := PlatformPasskeySignIn{RPID: passkeyRPID, Client: PlatformClient{Hash: &platformHash}, Credential: created.Credential, CredentialID: created.CredentialID}
	for _, verified := range []bool{false, true} {
		signIn.Verified = verified
		assertion, err := service.SignPlatformPasskey(signIn)
		if err != nil {
			t.Fatal(err)
		}
		verifyAssertion(t, created.Attestation.PublicKey, assertion, platformHash)
		flags, counter := authenticatorData(t, assertion.AuthenticatorData, passkeyRPID)
		if counter != 0 || flags&(flagUserPresent|flagBackupEligible|flagBackedUp) != flagUserPresent|flagBackupEligible|flagBackedUp || flags&flagUserVerified != 0 != verified || flags&flagAttestedData != 0 {
			t.Fatalf("verified %t: flags %08b, counter %d", verified, flags, counter)
		}
		if assertion.ClientData != nil || !bytes.Equal(assertion.CredentialID, created.CredentialID) || !bytes.Equal(assertion.UserHandle, []byte{1, 2, 3, 4}) {
			t.Fatalf("assertion = %+v", assertion)
		}
	}
	if !bytes.Equal(files.data, container) {
		t.Fatal("signing with a zero counter wrote to the vault")
	}
	if used, err := service.Usage(); err != nil || used[created.Credential] == 0 {
		t.Fatalf("the sign-in was not recorded as a use: %v, error = %v", used, err)
	}
}

func TestAPlatformSignInAdvancesANonzeroCounterBeforeItSigns(t *testing.T) {
	service, files, _ := captureVault(t)
	passkey := testPasskey(t, passkeyRPID, "alex", true, 41)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Imported", Passkeys: []vault.Passkey{passkey}})
	container := bytes.Clone(files.data)
	assertion, err := service.SignPlatformPasskey(PlatformPasskeySignIn{RPID: passkeyRPID, Client: PlatformClient{Hash: &platformHash}, Credential: id, CredentialID: passkey.CredentialID})
	if err != nil {
		t.Fatal(err)
	}
	if _, counter := authenticatorData(t, assertion.AuthenticatorData, passkeyRPID); counter != 42 {
		t.Fatalf("assertion counter %d, want 42", counter)
	}
	if bytes.Equal(files.data, container) {
		t.Fatal("the advanced counter was not saved")
	}
}

func TestAPlatformIsNeverSignedForWithAPasskeyOfAnotherRelyingParty(t *testing.T) {
	service, _, keys := captureVault(t)
	passkey := testPasskey(t, passkeyRPID, "alex", true, 0)
	id := createTestCredential(t, service, vault.CredentialInput{Label: "Example", Passkeys: []vault.Passkey{passkey}})
	client := PlatformClient{Hash: &platformHash}
	for name, signIn := range map[string]PlatformPasskeySignIn{
		"another relying party":            {RPID: "example.org", Client: client, Credential: id, CredentialID: passkey.CredentialID},
		"a subdomain of the relying party": {RPID: "login.example.com", Client: client, Credential: id, CredentialID: passkey.CredentialID},
		"another credential":               {RPID: passkeyRPID, Client: client, Credential: vault.ID{0xee}, CredentialID: passkey.CredentialID},
		"another credential ID":            {RPID: passkeyRPID, Client: client, Credential: id, CredentialID: []byte("not the passkey's credential ID")},
	} {
		if assertion, err := service.SignPlatformPasskey(signIn); !errors.Is(err, vault.ErrNotFound) || assertion.Signature != nil {
			t.Fatalf("%s: signed %+v, error = %v", name, assertion, err)
		}
	}
	for name, test := range map[string]struct {
		signIn PlatformPasskeySignIn
		want   error
	}{
		"a relying party no one can use": {PlatformPasskeySignIn{RPID: "com", Client: client, Credential: id, CredentialID: passkey.CredentialID}, authenticator.ErrInvalidRelyingParty},
		"no client data":                 {PlatformPasskeySignIn{RPID: passkeyRPID, Credential: id, CredentialID: passkey.CredentialID}, vault.ErrInvalidInput},
		"a challenge without an origin":  {PlatformPasskeySignIn{RPID: passkeyRPID, Client: PlatformClient{Challenge: challenge}, Credential: id, CredentialID: passkey.CredentialID}, vault.ErrInvalidInput},
	} {
		if assertion, err := service.SignPlatformPasskey(test.signIn); !errors.Is(err, test.want) || assertion.Signature != nil {
			t.Fatalf("%s: signed %+v, error = %v, want %v", name, assertion, err, test.want)
		}
	}
	if len(keys.usage) != 0 {
		t.Fatal("a refused sign-in was recorded as a use")
	}
}

func TestClientDataRavenpassBuildsForAPlatformCarriesTheGivenOriginAndChallenge(t *testing.T) {
	service, _, _ := captureVault(t)
	creation := newPlatformCreation()
	creation.Client = PlatformClient{Origin: appOrigin, Challenge: []byte("a creation challenge")}
	created, err := service.CreatePlatformPasskey(creation, "")
	if err != nil {
		t.Fatal(err)
	}
	wantClientData := func(data []byte, ceremony string, sent []byte) {
		t.Helper()
		fields := clientDataOf(t, data)
		if fields["type"] != ceremony || fields["origin"] != appOrigin || fields["challenge"] != base64.RawURLEncoding.EncodeToString(sent) || fields["crossOrigin"] != false {
			t.Fatalf("client data = %s", data)
		}
	}
	wantClientData(created.ClientData, "webauthn.create", creation.Client.Challenge)
	if credential := readTestCredential(t, service, created.Credential); !slices.Equal(credential.Websites, []string{passkeyRPID}) {
		t.Fatalf("a passkey an app created saved the websites %q", credential.Websites)
	}

	assertion, err := service.SignPlatformPasskey(PlatformPasskeySignIn{
		RPID: passkeyRPID, Client: PlatformClient{Origin: appOrigin, Challenge: challenge}, Credential: created.Credential, CredentialID: created.CredentialID,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantClientData(assertion.ClientData, "webauthn.get", challenge)
	verifyAssertion(t, created.Attestation.PublicKey, assertion, sha256.Sum256(assertion.ClientData))
}
