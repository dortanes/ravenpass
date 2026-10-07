package vault

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// testApp is a link to package signed with a certificate whose digest is 32 bytes of mark.
func testApp(pkg string, mark byte) App {
	app := App{Package: pkg}
	copy(app.Signer[:], bytes.Repeat([]byte{mark}, len(app.Signer)))
	return app
}

// appFieldsOf are the fields of an app link as a record and the index hold it.
func appFieldsOf(app App) [][]byte {
	return [][]byte{encodeBytes([]byte(app.Package)), encodeBytes(app.Signer[:])}
}

// appRecord is a credential record with no website, no text, no passkey and the given app list.
func appRecord(apps []byte) []byte {
	return credentialRecordOf(encodeArray(), encodeBytes(nil), encodeArray(), apps)
}

func numberedApps(count int) []App {
	apps := make([]App, count)
	for i := range apps {
		apps[i] = testApp(fmt.Sprintf("com.example.app%d", i), byte(i+1))
	}
	return apps
}

func TestAppLinksRoundTrip(t *testing.T) {
	mail := testApp("com.example.mail", 1)
	rotated := testApp("com.example.mail", 2)
	tests := []struct {
		name  string
		input CredentialInput
	}{
		{"only an app", CredentialInput{Password: "secret", Apps: []App{mail}}},
		{"one website", CredentialInput{Websites: []string{"https://example.com"}, Login: "alex", Password: "secret", Apps: []App{mail}}},
		{"everything", CredentialInput{Websites: []string{"https://example.com", "https://example.org"}, Login: "alex", Email: "alex@example.com", Password: "secret", Notes: "line", TOTP: standardSecret, Passkeys: []Passkey{testPasskey(t, 1)}, Apps: []App{mail, rotated}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plaintext, err := encodeCredentialRecord(test.input)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := decodeCredentialRecord(plaintext)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded, test.input) {
				t.Fatalf("read back as %+v", decoded)
			}
		})
	}
}

func TestNoAppsAreAcceptedAsNil(t *testing.T) {
	for _, apps := range [][]App{nil, {}} {
		accepted, err := acceptInput(CredentialInput{Label: "Example", Password: "secret", Apps: apps})
		if err != nil {
			t.Fatal(err)
		}
		if accepted.Apps != nil {
			t.Fatalf("no apps accepted as %#v", accepted.Apps)
		}
	}
}

func TestCredentialWithoutAppLinksHasAnEmptyAppField(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail", Websites: []string{"https://mail.example"}, Login: "alex", Password: "one"})
	session := created.Session
	defer session.Lock()
	raw, head := currentRaw(t, session)
	entry := session.entries[0]

	record, err := session.openRecord(0)
	if err != nil {
		t.Fatal(err)
	}
	empty, none := encodeBytes(nil), encodeArray()
	if want := encodeArray(encodeUint(recordSchemaCredential), encodeArray(encodeBytes([]byte("https://mail.example"))), encodeBytes([]byte("alex")), empty, encodeBytes([]byte("one")), empty, empty, none, none); !bytes.Equal(record, want) {
		t.Fatalf("record written as % x, want % x", record, want)
	}
	fields := currentFields(entry.id, entry.digest, uint64(KindCredential), encodeArray())
	fields[fieldLabel] = encodeBytes([]byte("Mail"))
	fields[fieldDetail] = encodeBytes([]byte("alex"))
	fields[fieldSite] = encodeBytes([]byte("mail.example"))
	fields[fieldSites] = encodeArray(encodeBytes([]byte("mail.example")))
	expected := indexPlaintext(head.Revision, head.PreviousHash, encodeArray(), encodeArray(fields...))
	written, err := encodeIndex(head.Revision, session.ancestry, session.entries, nil, DefaultTrashRetention)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, expected) {
		t.Fatal("an entry without app links is not written with an empty app field")
	}

	reopened, err := OpenWithRecovery(withIndex(t, session, raw, expected), created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if listed := listedEntry(t, reopened, ids[0]); listed.Apps != nil {
		t.Fatalf("listed apps = %+v", listed.Apps)
	}
	credential, err := reopened.ReadCredential(ids[0])
	if err != nil || credential.Apps != nil || credential.Password != "one" {
		t.Fatalf("credential = %+v, error = %v", credential, err)
	}
}

func TestAcceptAppsTakesAndroidPackageNamesWithinTheLimits(t *testing.T) {
	longest := "a." + strings.Repeat("b", MaxPackageLength-2)
	valid := [][]App{
		{testApp("com.example", 1)},
		{testApp("a.b", 1)},
		{testApp("Com.Example_1.app_", 1)},
		{testApp(longest, 1)},
		{testApp("com.example.mail", 1), testApp("com.example.mail", 2)},
		{testApp("com.example.mail", 1), testApp("com.example.chat", 1)},
		numberedApps(MaxCredentialApps),
	}
	for _, apps := range valid {
		accepted, err := acceptApps(apps)
		if err != nil || !slices.Equal(accepted, apps) {
			t.Fatalf("apps %+v accepted as %+v, error = %v", apps, accepted, err)
		}
	}
	invalid := map[string][]App{
		"one segment":              {testApp("example", 1)},
		"an empty segment":         {testApp("com..example", 1)},
		"a leading dot":            {testApp(".com.example", 1)},
		"a trailing dot":           {testApp("com.example.", 1)},
		"a segment led by a digit": {testApp("com.1example", 1)},
		"a segment led by a line":  {testApp("com._example", 1)},
		"a hyphen":                 {testApp("com.exa-mple", 1)},
		"a space":                  {testApp("com.example ", 1)},
		"a letter outside ASCII":   {testApp("com.exämple", 1)},
		"a name over its limit":    {testApp(longest+"c", 1)},
		"no signer":                {{Package: "com.example"}},
		"one link twice":           {testApp("com.example", 1), testApp("com.example", 1)},
		"more links than it holds": numberedApps(MaxCredentialApps + 1),
	}
	for name, apps := range invalid {
		if _, err := acceptApps(apps); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s: got %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestAppRecordOutsideItsShapeIsMalformed(t *testing.T) {
	valid := testApp("com.example.mail", 1)
	crowded := make([][]byte, MaxCredentialApps+1)
	for i, app := range numberedApps(MaxCredentialApps + 1) {
		crowded[i] = encodeArray(appFieldsOf(app)...)
	}
	empty := encodeBytes(nil)
	link := encodeArray(appFieldsOf(valid)...)
	tests := []struct {
		name   string
		record []byte
	}{
		{"apps that are not a list", appRecord(empty)},
		{"more apps than a credential holds", appRecord(encodeArray(crowded...))},
		{"one link twice", appRecord(encodeArray(link, link))},
		{"a link of one field", appRecord(encodeArray(encodeArray(appFieldsOf(valid)[:1]...)))},
		{"a signer of 31 bytes", appRecord(encodeArray(encodeArray(encodeBytes([]byte(valid.Package)), encodeBytes(make([]byte, 31)))))},
		{"no signer", appRecord(encodeArray(encodeArray(encodeBytes([]byte(valid.Package)), encodeBytes(make([]byte, 32)))))},
		{"a package that is not one", appRecord(encodeArray(encodeArray(encodeBytes([]byte("example")), encodeBytes(valid.Signer[:]))))},
		{"a field after the apps", encodeArray(encodeUint(recordSchemaCredential), encodeArray(), empty, empty, empty, empty, empty, encodeArray(), encodeArray(link), encodeArray())},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeCredentialRecord(test.record); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestIndexCarriesEachCredentialsAppLinks(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Bare", Password: "one"})
	session := created.Session
	defer session.Lock()
	links := []App{testApp("com.example.mail", 1), testApp("com.example.mail", 2)}
	linked := commitCredential(t, session, CredentialInput{Label: "Linked", Password: "two", Apps: links}, nil)
	identity := commitIdentity(t, session, IdentityInput{Label: "Me"}, nil)
	want := map[ID][]App{ids[0]: nil, linked: links, identity: nil}
	wantApps := func(session *Session) {
		t.Helper()
		for id, apps := range want {
			if entry := listedEntry(t, session, id); !reflect.DeepEqual(entry.Apps, apps) {
				t.Fatalf("apps of %s = %+v, want %+v", id, entry.Apps, apps)
			}
		}
	}
	wantApps(session)
	listedEntry(t, session, linked).Apps[0].Package = "com.attacker.mail"
	wantApps(session)

	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	wantApps(reopened)
	if credential := selectCredential(t, reopened, linked); !reflect.DeepEqual(credential.Apps, links) {
		t.Fatalf("record apps = %+v", credential.Apps)
	}
}

func TestIndexRoundTripsAppLinks(t *testing.T) {
	records := stubRecords(2)
	entries := withDigests([]entryMeta{
		{id: ID{1}, revision: 1, label: "Linked", kind: KindCredential, apps: numberedApps(MaxCredentialApps)},
		{id: ID{2}, revision: 1, label: "Bare", kind: KindCredential},
	}, records)
	plaintext, err := encodeIndex(3, nil, entries, nil, DefaultTrashRetention)
	if err != nil {
		t.Fatal(err)
	}
	_, _, index, err := parseIndex(plaintext, records)
	if err != nil {
		t.Fatal(err)
	}
	if parsed := index.entries; !reflect.DeepEqual(parsed, entries) {
		t.Fatalf("entries changed: %+v", parsed)
	}
	entries[0].apps = numberedApps(MaxCredentialApps + 1)
	if _, err := encodeIndex(3, nil, entries, nil, DefaultTrashRetention); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("seventeen links: got %v, want ErrResourceLimit", err)
	}
}

func TestIndexRejectsMalformedAppLinks(t *testing.T) {
	records := stubRecords(1)
	digest := withDigests(make([]entryMeta, 1), records)[0].digest
	id := ID{5}
	link := encodeArray(appFieldsOf(testApp("com.example.mail", 1))...)
	element := func(kind Kind, apps ...[]byte) []byte {
		fields := currentFields(id, digest, uint64(kind), encodeArray())
		return encodeArray(append(fields[:fieldApps], apps...)...)
	}
	for _, apps := range [][]byte{encodeArray(), encodeArray(link)} {
		if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), element(KindCredential, apps)), records); err != nil {
			t.Fatalf("a valid app field % x: %v", apps, err)
		}
	}
	crowded := make([][]byte, MaxCredentialApps+1)
	for i, app := range numberedApps(MaxCredentialApps + 1) {
		crowded[i] = encodeArray(appFieldsOf(app)...)
	}
	tests := []struct {
		name    string
		element []byte
	}{
		{"no app field", element(KindCredential)},
		{"an app field that is not a list", element(KindCredential, encodeBytes(nil))},
		{"an identity with an app field", element(KindIdentity, encodeArray(link))},
		{"one link twice", element(KindCredential, encodeArray(link, link))},
		{"more links than a credential holds", element(KindCredential, encodeArray(crowded...))},
		{"a package that is not one", element(KindCredential, encodeArray(encodeArray(encodeBytes([]byte("mail")), encodeBytes(bytes.Repeat([]byte{1}, 32)))))},
		{"twenty-one fields", element(KindCredential, encodeArray(link), encodeArray())},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), test.element), records); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestEditReplacesAppLinksOnlyWhenPatched(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	defer session.Lock()
	kept, removed := testApp("com.example.mail", 1), testApp("com.example.mail", 2)
	id := commitCredential(t, session, CredentialInput{Label: "Mail", Password: "one", Apps: []App{kept, removed}}, nil)
	password := "two"
	commitEdit := func(patch CredentialPatch) {
		t.Helper()
		pending, err := session.PrepareEdit(id, patch)
		if err != nil {
			t.Fatal(err)
		}
		if err := session.Commit(pending); err != nil {
			t.Fatal(err)
		}
	}
	commitEdit(CredentialPatch{Password: &password})
	if apps := listedEntry(t, session, id).Apps; !slices.Equal(apps, []App{kept, removed}) {
		t.Fatalf("apps after an edit without them = %+v", apps)
	}
	commitEdit(CredentialPatch{Apps: &[]App{kept}})
	if apps := selectCredential(t, session, id).Apps; !slices.Equal(apps, []App{kept}) {
		t.Fatalf("apps after removing one = %+v", apps)
	}
	if _, err := session.PrepareEdit(id, CredentialPatch{Apps: &[]App{{Package: "mail", Signer: kept.Signer}}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("an invalid link: got %v, want ErrInvalidInput", err)
	}
	commitEdit(CredentialPatch{Apps: &[]App{}})
	if apps := listedEntry(t, session, id).Apps; apps != nil {
		t.Fatalf("apps after removing every link = %+v", apps)
	}
}

func TestLinksAppMatchesOnlyTheSamePackageAndSigner(t *testing.T) {
	mail := testApp("com.example.mail", 1)
	apps := []App{mail, testApp("com.example.chat", 2)}
	tests := []struct {
		name    string
		pkg     string
		signers [][32]byte
		want    bool
	}{
		{"the linked app", "com.example.mail", [][32]byte{mail.Signer}, true},
		{"the linked app among its signers", "com.example.mail", [][32]byte{testApp("", 9).Signer, mail.Signer}, true},
		{"a look-alike signed with another certificate", "com.example.mail", [][32]byte{testApp("", 9).Signer}, false},
		{"another package signed with a linked certificate", "com.example.chat", [][32]byte{mail.Signer}, false},
		{"another package", "com.attacker.mail", [][32]byte{mail.Signer}, false},
		{"no signer", "com.example.mail", nil, false},
		{"no package", "", [][32]byte{mail.Signer}, false},
	}
	for _, test := range tests {
		if got := LinksApp(apps, test.pkg, test.signers); got != test.want {
			t.Fatalf("%s: LinksApp = %v, want %v", test.name, got, test.want)
		}
	}
	if LinksApp(nil, mail.Package, [][32]byte{mail.Signer}) {
		t.Fatal("a credential without links matched")
	}
}
