package extensionaccess

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/confirmation"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
	"github.com/dortanes/ravenpass/packages/app/linkserver"
	"github.com/dortanes/ravenpass/packages/app/vaultservice"
	"github.com/dortanes/ravenpass/packages/app/verification"
	"github.com/dortanes/ravenpass/packages/authenticator"
	"github.com/dortanes/ravenpass/packages/vault"
)

// The passkey calls answer err alone, for tests that never reach a passkey.
func (f *fakeCredentials) SignInPasskeys(string, string, [][]byte) (string, []vaultservice.PasskeyChoice, error) {
	return "", nil, f.err
}

func (f *fakeCredentials) PasskeyTargets(string, string, string, [][]byte) (vaultservice.PasskeyTargets, error) {
	return vaultservice.PasskeyTargets{}, f.err
}

func (f *fakeCredentials) CreatePasskey(vaultservice.PasskeyCreation, string) (vaultservice.CreatedPasskey, error) {
	return vaultservice.CreatedPasskey{}, f.err
}

func (f *fakeCredentials) SignPasskey(vaultservice.PasskeySignIn) (vaultservice.PasskeyAssertion, error) {
	return vaultservice.PasskeyAssertion{}, f.err
}

var (
	passkeyID    = bytes.Repeat([]byte{0x5a}, 16)
	passkeyOwner = vault.ID{0x0c}
	// idnRPID is пример.рф as a relying party ID.
	idnRPID = "xn--e1afmkfd.xn--p1ai"
)

// passkeyCredentials answers the passkey calls with its fields and records what it is asked.
type passkeyCredentials struct {
	*fakeCredentials
	listErr   error
	choices   []vaultservice.PasskeyChoice
	targets   vaultservice.PasskeyTargets
	created   vaultservice.CreatedPasskey
	signed    vaultservice.PasskeyAssertion
	lists     []string
	creations []vaultservice.PasskeyCreation
	groups    []string
	signIns   []vaultservice.PasskeySignIn
}

func (c *passkeyCredentials) SignInPasskeys(origin, rpID string, allow [][]byte) (string, []vaultservice.PasskeyChoice, error) {
	c.lists = append(c.lists, "passkeys "+origin+" "+rpID)
	return idnRPID, c.choices, c.listErr
}

func (c *passkeyCredentials) PasskeyTargets(origin, rpID, account string, exclude [][]byte) (vaultservice.PasskeyTargets, error) {
	c.lists = append(c.lists, "targets "+origin+" "+rpID+" "+account)
	return c.targets, c.listErr
}

func (c *passkeyCredentials) CreatePasskey(creation vaultservice.PasskeyCreation, group string) (vaultservice.CreatedPasskey, error) {
	c.creations = append(c.creations, creation)
	c.groups = append(c.groups, group)
	return c.created, c.err
}

func (c *passkeyCredentials) SignPasskey(signIn vaultservice.PasskeySignIn) (vaultservice.PasskeyAssertion, error) {
	c.signIns = append(c.signIns, signIn)
	return c.signed, c.err
}

func newPasskeyCredentials() *passkeyCredentials {
	return &passkeyCredentials{
		fakeCredentials: &fakeCredentials{},
		choices:         []vaultservice.PasskeyChoice{{Credential: passkeyOwner, CredentialID: passkeyID, Account: "alex", Label: "Example"}},
		targets:         vaultservice.PasskeyTargets{RPID: idnRPID, Targets: []vaultservice.PasskeyTarget{{ID: passkeyOwner, Label: "Example", Account: "alex", Tags: []string{"Work"}}}},
		created: vaultservice.CreatedPasskey{
			Credential: passkeyOwner, CredentialID: passkeyID, ClientData: []byte("{}"),
			Attestation: authenticator.Attestation{AttestationObject: []byte{0xa3}, AuthenticatorData: []byte{0x49}, PublicKey: []byte{0x30}, Algorithm: authenticator.AlgorithmES256},
		},
		signed: vaultservice.PasskeyAssertion{CredentialID: passkeyID, ClientData: []byte("{}"), AuthenticatorData: []byte{0x1d}, Signature: []byte{0x30}, UserHandle: []byte{1}},
	}
}

func passkeyAccess(t *testing.T, credentials *passkeyCredentials, verifier *fakeVerifier) *Access {
	t.Helper()
	access, err := New(credentials, &fakeIdentities{}, &fakeIcons{}, verifier, &fakeChoices{group: "work"}, &fakeUnlocks{})
	if err != nil {
		t.Fatal(err)
	}
	return access
}

func creationRequest(verify bool, target string) linkproto.PasskeyCreation {
	return linkproto.PasskeyCreation{
		Origin: "https://login.example.com", RPID: idnRPID, RPName: "Example",
		User:      linkproto.PasskeyUser{ID: []byte{1, 2}, Name: "alex", DisplayName: "Alex"},
		Challenge: []byte("challenge"), Verify: verify, Target: target, Exclude: [][]byte{{9}},
	}
}

func signInRequest(verify bool) linkproto.PasskeySignIn {
	return linkproto.PasskeySignIn{
		Origin: "https://login.example.com", RPID: idnRPID, Challenge: []byte("challenge"), Verify: verify,
		Credential: passkeyOwner.String(), CredentialID: passkeyID,
	}
}

func TestPasskeyListsCarryTheCheckedRelyingParty(t *testing.T) {
	credentials := newPasskeyCredentials()
	access := passkeyAccess(t, credentials, &fakeVerifier{})
	passkeys, err := access.Passkeys(linkproto.PasskeyQuery{Origin: "https://login.example.com", Mode: linkproto.PasskeyGet})
	want := linkproto.Passkeys{RPID: idnRPID, Passkeys: []linkproto.PasskeyOption{{Credential: passkeyOwner.String(), CredentialID: passkeyID, Account: "alex", Label: "Example"}}}
	if err != nil || !reflect.DeepEqual(passkeys, want) {
		t.Fatalf("passkeys = %+v, error = %v", passkeys, err)
	}
	credentials.targets.Excluded = true
	targets, err := access.PasskeyTargets(linkproto.PasskeyQuery{Origin: "https://login.example.com", Mode: linkproto.PasskeyCreate, Account: "alex"})
	wantTargets := linkproto.PasskeyTargets{RPID: idnRPID, Targets: []linkproto.PasskeyTarget{{Credential: passkeyOwner.String(), Label: "Example", Account: "alex", Tags: []string{"Work"}}}, Excluded: true}
	if err != nil || !reflect.DeepEqual(targets, wantTargets) {
		t.Fatalf("targets = %+v, error = %v", targets, err)
	}
	for cause, want := range map[error]error{
		vaultservice.ErrNotReady:             linkserver.ErrLocked,
		authenticator.ErrInvalidOrigin:       linkserver.ErrInvalidOrigin,
		authenticator.ErrInvalidRelyingParty: linkserver.ErrInvalidRelyingParty,
	} {
		credentials.listErr = cause
		if _, err := access.Passkeys(linkproto.PasskeyQuery{}); !errors.Is(err, want) {
			t.Fatalf("passkeys failing with %v: got %v, want %v", cause, err, want)
		}
		if _, err := access.PasskeyTargets(linkproto.PasskeyQuery{}); !errors.Is(err, want) {
			t.Fatalf("targets failing with %v: got %v, want %v", cause, err, want)
		}
	}
}

func TestANewPasskeyIsSavedOnceThePersonVerifiesSavingIt(t *testing.T) {
	credentials, verifier := newPasskeyCredentials(), &fakeVerifier{method: verification.MethodPIN}
	access := passkeyAccess(t, credentials, verifier)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var asked []linkproto.Progress
	created, err := access.CreatePasskey(ctx, creationRequest(true, linkproto.TargetNew), func(p linkproto.Progress) { asked = append(asked, p) })
	if err != nil {
		t.Fatal(err)
	}
	want := linkproto.CreatedPasskey{
		Credential: passkeyOwner.String(), CredentialID: passkeyID, ClientDataJSON: []byte("{}"),
		AttestationObject: []byte{0xa3}, AuthenticatorData: []byte{0x49}, PublicKey: []byte{0x30}, PublicKeyAlgorithm: -7,
	}
	if !reflect.DeepEqual(created, want) {
		t.Fatalf("created = %+v", created)
	}
	if !slices.Equal(verifier.reasons, []confirmation.Reason{confirmation.SavingPasskey("пример.рф")}) || verifier.context != ctx {
		t.Fatalf("the prompt named %+v", verifier.reasons)
	}
	if !slices.Equal(asked, []linkproto.Progress{linkproto.ProgressConfirmInRavenpass}) {
		t.Fatalf("progress = %q", asked)
	}
	wantCreation := vaultservice.PasskeyCreation{
		Origin: "https://login.example.com", RPID: idnRPID, RPName: "Example",
		User:      vaultservice.PasskeyUser{Handle: []byte{1, 2}, Name: "alex", DisplayName: "Alex"},
		Challenge: []byte("challenge"), Exclude: [][]byte{{9}}, Verified: true,
	}
	if len(credentials.creations) != 1 || !reflect.DeepEqual(credentials.creations[0], wantCreation) || !slices.Equal(credentials.groups, []string{"work"}) {
		t.Fatalf("the vault was asked to create %+v in %q", credentials.creations, credentials.groups)
	}
	if !slices.Equal(credentials.lists, []string{"targets https://login.example.com " + idnRPID + " alex"}) {
		t.Fatalf("the vault was asked %q", credentials.lists)
	}
}

func TestAPasskeyCreationThatDoesNotAskForVerificationAsksNoOne(t *testing.T) {
	credentials, verifier := newPasskeyCredentials(), &fakeVerifier{}
	access := passkeyAccess(t, credentials, verifier)
	if _, err := access.CreatePasskey(context.Background(), creationRequest(false, passkeyOwner.String()), func(linkproto.Progress) {
		t.Fatal("a creation without verification reported progress")
	}); err != nil {
		t.Fatal(err)
	}
	if len(verifier.reasons) != 0 {
		t.Fatalf("the person was asked %+v", verifier.reasons)
	}
	if creation := credentials.creations[0]; creation.Verified || creation.Target != passkeyOwner {
		t.Fatalf("the vault was asked to create %+v", creation)
	}
}

func TestAPasskeyCreationIsRefusedBeforeTheVaultWhenItShouldBe(t *testing.T) {
	for name, test := range map[string]struct {
		target   string
		listErr  error
		excluded bool
		verifier *fakeVerifier
		want     error
		asked    bool
	}{
		"a target that is not an ID": {target: "mine", verifier: &fakeVerifier{}, want: linkserver.ErrNotFound},
		"a locked vault":             {target: linkproto.TargetNew, listErr: vaultservice.ErrNotReady, verifier: &fakeVerifier{}, want: linkserver.ErrLocked},
		"a relying party refused":    {target: linkproto.TargetNew, listErr: authenticator.ErrInvalidRelyingParty, verifier: &fakeVerifier{}, want: linkserver.ErrInvalidRelyingParty},
		"an excluded passkey":        {target: linkproto.TargetNew, excluded: true, verifier: &fakeVerifier{}, want: linkserver.ErrExcluded},
		"a dismissed verification":   {target: linkproto.TargetNew, verifier: &fakeVerifier{err: verification.ErrDeclined}, want: linkserver.ErrDeclined, asked: true},
		"nothing to verify with":     {target: linkproto.TargetNew, verifier: &fakeVerifier{err: verification.ErrUnverifiable}, want: linkserver.ErrUnverifiable, asked: true},
	} {
		credentials := newPasskeyCredentials()
		credentials.listErr, credentials.targets.Excluded = test.listErr, test.excluded
		access := passkeyAccess(t, credentials, test.verifier)
		created, err := access.CreatePasskey(context.Background(), creationRequest(true, test.target), func(linkproto.Progress) {})
		if !errors.Is(err, test.want) || created.CredentialID != nil {
			t.Fatalf("%s: got %+v, %v, want %v", name, created, err, test.want)
		}
		if len(credentials.creations) != 0 || (len(test.verifier.reasons) != 0) != test.asked {
			t.Fatalf("%s: created %+v after asking %+v", name, credentials.creations, test.verifier.reasons)
		}
	}
	for cause, want := range map[error]error{
		vault.ErrPasskeysFull:           linkserver.ErrPasskeysFull,
		vaultservice.ErrPasskeyExcluded: linkserver.ErrExcluded,
		vaultservice.ErrNoMatch:         linkserver.ErrNoMatch,
		vault.ErrNotFound:               linkserver.ErrNotFound,
	} {
		credentials := newPasskeyCredentials()
		credentials.err = cause
		if _, err := passkeyAccess(t, credentials, &fakeVerifier{}).CreatePasskey(context.Background(), creationRequest(false, passkeyOwner.String()), func(linkproto.Progress) {}); !errors.Is(err, want) {
			t.Fatalf("a creation failing with %v: got %v, want %v", cause, err, want)
		}
	}
}

func TestASignInIsSignedOnceThePersonVerifiesIt(t *testing.T) {
	credentials, verifier := newPasskeyCredentials(), &fakeVerifier{method: verification.MethodDevice}
	access := passkeyAccess(t, credentials, verifier)
	var asked []linkproto.Progress
	signed, err := access.SignPasskey(context.Background(), signInRequest(true), func(p linkproto.Progress) { asked = append(asked, p) })
	if err != nil {
		t.Fatal(err)
	}
	want := linkproto.PasskeyAssertion{CredentialID: passkeyID, ClientDataJSON: []byte("{}"), AuthenticatorData: []byte{0x1d}, Signature: []byte{0x30}, UserHandle: []byte{1}}
	if !reflect.DeepEqual(signed, want) {
		t.Fatalf("signed = %+v", signed)
	}
	if !slices.Equal(verifier.reasons, []confirmation.Reason{confirmation.SigningIn("пример.рф", "alex")}) || !slices.Equal(asked, []linkproto.Progress{linkproto.ProgressConfirmOnDevice}) {
		t.Fatalf("the prompt named %+v, progress %q", verifier.reasons, asked)
	}
	wantSignIn := vaultservice.PasskeySignIn{Origin: "https://login.example.com", RPID: idnRPID, Challenge: []byte("challenge"), Verified: true, Credential: passkeyOwner, CredentialID: passkeyID}
	if len(credentials.signIns) != 1 || !reflect.DeepEqual(credentials.signIns[0], wantSignIn) {
		t.Fatalf("the vault was asked to sign %+v", credentials.signIns)
	}

	verifier.reasons = nil
	if _, err := access.SignPasskey(context.Background(), signInRequest(false), func(linkproto.Progress) {
		t.Fatal("a sign-in without verification reported progress")
	}); err != nil {
		t.Fatal(err)
	}
	if len(verifier.reasons) != 0 || credentials.signIns[1].Verified {
		t.Fatalf("a sign-in without verification asked %+v and signed %+v", verifier.reasons, credentials.signIns[1])
	}
}

func TestASignInIsRefusedBeforeTheVaultSignsWhenItShouldBe(t *testing.T) {
	for name, test := range map[string]struct {
		credential string
		choices    []vaultservice.PasskeyChoice
		listErr    error
		verifier   *fakeVerifier
		want       error
		asked      bool
	}{
		"a credential that is not an ID": {credential: "mine", verifier: &fakeVerifier{}, want: linkserver.ErrNotFound},
		"a passkey the vault lacks":      {choices: []vaultservice.PasskeyChoice{}, verifier: &fakeVerifier{}, want: linkserver.ErrNotFound},
		"a passkey of another credential": {
			choices: []vaultservice.PasskeyChoice{{Credential: vault.ID{0xee}, CredentialID: passkeyID}}, verifier: &fakeVerifier{}, want: linkserver.ErrNotFound,
		},
		"a relying party refused":  {listErr: authenticator.ErrInvalidRelyingParty, verifier: &fakeVerifier{}, want: linkserver.ErrInvalidRelyingParty},
		"a dismissed verification": {verifier: &fakeVerifier{err: verification.ErrDeclined}, want: linkserver.ErrDeclined, asked: true},
		"an ended session":         {verifier: &fakeVerifier{err: context.Canceled}, want: context.Canceled, asked: true},
	} {
		credentials := newPasskeyCredentials()
		credentials.listErr = test.listErr
		if test.choices != nil {
			credentials.choices = test.choices
		}
		request := signInRequest(true)
		if test.credential != "" {
			request.Credential = test.credential
		}
		signed, err := passkeyAccess(t, credentials, test.verifier).SignPasskey(context.Background(), request, func(linkproto.Progress) {})
		if !errors.Is(err, test.want) || signed.Signature != nil {
			t.Fatalf("%s: got %+v, %v, want %v", name, signed, err, test.want)
		}
		if len(credentials.signIns) != 0 || (len(test.verifier.reasons) != 0) != test.asked {
			t.Fatalf("%s: signed %+v after asking %+v", name, credentials.signIns, test.verifier.reasons)
		}
	}
}
