package vault

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

var passkeyCreatedAt = time.Date(2026, 9, 23, 10, 30, 0, 0, time.UTC)

func ecdsaKey(t *testing.T, curve elliptic.Curve) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// testPasskey is a discoverable passkey for example.com whose credential ID is 16 bytes of mark.
func testPasskey(t *testing.T, mark byte) Passkey {
	t.Helper()
	return Passkey{
		CredentialID:    bytes.Repeat([]byte{mark}, minCredentialIDBytes),
		RPID:            "example.com",
		UserHandle:      []byte{mark, 0x01},
		UserName:        "alex@example.com",
		UserDisplayName: "Alex",
		PrivateKey:      ecdsaKey(t, elliptic.P256()),
		Discoverable:    true,
		CreatedAt:       passkeyCreatedAt,
	}
}

// withoutKey is a passkey as a read that leaves private keys behind returns it.
func withoutKey(passkey Passkey) Passkey {
	passkey.PrivateKey = nil
	return passkey
}

func faceOfPasskey(passkey Passkey) PasskeyFace {
	return PasskeyFace{
		CredentialID: passkey.CredentialID, RPID: passkey.RPID, UserHandle: passkey.UserHandle,
		UserName: passkey.UserName, UserDisplayName: passkey.UserDisplayName, Discoverable: passkey.Discoverable,
	}
}

// passkeyFieldsOf are the fields of a passkey as a record holds it.
func passkeyFieldsOf(passkey Passkey) [][]byte {
	return [][]byte{encodeBytes(passkey.CredentialID), encodeBytes([]byte(passkey.RPID)), encodeBytes(passkey.UserHandle), encodeBytes([]byte(passkey.UserName)), encodeBytes([]byte(passkey.UserDisplayName)), encodeBytes(passkey.PrivateKey), encodeUint(uint64(passkey.Counter)), encodeUint(boolFlag(passkey.Discoverable)), encodeUint(uint64(passkey.CreatedAt.Unix()))}
}

// passkeyRecord is a credential record with no website, no text, no app and the given passkey list.
func passkeyRecord(passkeys []byte) []byte {
	return credentialRecordOf(encodeArray(), encodeBytes(nil), passkeys, encodeArray())
}

// sealedRecords are the sealed records of the session's container by item.
func sealedRecords(t *testing.T, session *Session) map[ID][]byte {
	t.Helper()
	raw, _ := currentRaw(t, session)
	records := make(map[ID][]byte, len(raw.records))
	for i, record := range raw.records {
		records[session.entries[i].id] = encodeBox(record)
	}
	return records
}

func TestPasskeyRecordRoundTrips(t *testing.T) {
	first, second := testPasskey(t, 1), testPasskey(t, 2)
	second.RPID, second.UserName, second.UserDisplayName, second.Counter, second.Discoverable = "login.example.org", "", "", 41, false
	tests := []struct {
		name  string
		input CredentialInput
	}{
		{"only a passkey", CredentialInput{Passkeys: []Passkey{first}}},
		{"one website", CredentialInput{Websites: []string{"https://example.com"}, Login: "alex", Password: "secret", Passkeys: []Passkey{first}}},
		{"everything", CredentialInput{Websites: []string{"https://example.com", "https://example.org"}, Login: "alex", Email: "alex@example.com", Password: "secret", Notes: "line", TOTP: standardSecret, Passkeys: []Passkey{first, second}}},
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

func TestCredentialWithoutPasskeysHasAnEmptyPasskeyField(t *testing.T) {
	empty, none := encodeBytes(nil), encodeArray()
	website := encodeArray(encodeBytes([]byte("https://example.com")))
	password := encodeBytes([]byte("secret"))
	tests := []struct {
		name  string
		input CredentialInput
		want  []byte
	}{
		{"one website", CredentialInput{Label: "Example", Websites: []string{"https://example.com"}, Password: "secret"}, encodeArray(encodeUint(recordSchemaCredential), website, empty, empty, password, empty, empty, none, none)},
		{"a code", CredentialInput{Label: "Example", Websites: []string{"https://example.com"}, Password: "secret", TOTP: standardSecret}, encodeArray(encodeUint(recordSchemaCredential), website, empty, empty, password, empty, encodeBytes([]byte(standardSecret)), none, none)},
		{"two websites", CredentialInput{Label: "Example", Websites: []string{"https://example.com", "https://example.org"}, Password: "secret"}, encodeArray(encodeUint(recordSchemaCredential), encodeArray(encodeTexts([]string{"https://example.com", "https://example.org"})...), empty, empty, password, empty, empty, none, none)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, passkeys := range [][]Passkey{nil, {}} {
				input := test.input
				input.Passkeys = passkeys
				accepted, err := acceptInput(input)
				if err != nil {
					t.Fatal(err)
				}
				if accepted.Passkeys != nil {
					t.Fatalf("no passkeys accepted as %#v", accepted.Passkeys)
				}
				plaintext, err := encodeCredentialRecord(accepted)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(plaintext, test.want) {
					t.Fatalf("written as % x, want % x", plaintext, test.want)
				}
			}
		})
	}
}

func TestPasskeyRecordOutsideItsShapeIsMalformed(t *testing.T) {
	valid := testPasskey(t, 1)
	element := func(position int, value []byte) []byte {
		fields := passkeyFieldsOf(valid)
		fields[position] = value
		return encodeArray(fields...)
	}
	crowded := make([][]byte, MaxCredentialPasskeys+1)
	for i := range crowded {
		crowded[i] = encodeArray(passkeyFieldsOf(testPasskey(t, byte(i+1)))...)
	}
	empty := encodeBytes(nil)
	tests := []struct {
		name   string
		record []byte
	}{
		{"passkeys that are not a list", passkeyRecord(empty)},
		{"more passkeys than a credential holds", passkeyRecord(encodeArray(crowded...))},
		{"one credential ID twice", passkeyRecord(encodeArray(encodeArray(passkeyFieldsOf(valid)...), encodeArray(passkeyFieldsOf(valid)...)))},
		{"a passkey with eight fields", passkeyRecord(encodeArray(encodeArray(passkeyFieldsOf(valid)[:8]...)))},
		{"a credential ID under its limit", passkeyRecord(encodeArray(element(0, encodeBytes(make([]byte, minCredentialIDBytes-1)))))},
		{"a credential ID over its limit", passkeyRecord(encodeArray(element(0, encodeBytes(make([]byte, maxCredentialIDBytes+1)))))},
		{"an upper-case relying party", passkeyRecord(encodeArray(element(1, encodeBytes([]byte("Example.com")))))},
		{"an empty user handle", passkeyRecord(encodeArray(element(2, empty)))},
		{"a user handle over its limit", passkeyRecord(encodeArray(element(2, encodeBytes(make([]byte, maxUserHandleBytes+1)))))},
		{"a user name that is not text", passkeyRecord(encodeArray(element(3, encodeBytes([]byte("\xff")))))},
		{"a display name over its limit", passkeyRecord(encodeArray(element(4, encodeBytes([]byte(strings.Repeat("a", MaxLoginLength+1))))))},
		{"no private key", passkeyRecord(encodeArray(element(5, empty)))},
		{"a private key over its limit", passkeyRecord(encodeArray(element(5, encodeBytes(make([]byte, maxPasskeyKeyBytes+1)))))},
		{"a counter over 32 bits", passkeyRecord(encodeArray(element(6, encodeUint(math.MaxUint32+1))))},
		{"discoverable two", passkeyRecord(encodeArray(element(7, encodeUint(2))))},
		{"discoverable as bytes", passkeyRecord(encodeArray(element(7, encodeBytes([]byte{1}))))},
		{"a creation time past any instant", passkeyRecord(encodeArray(element(8, encodeUint(math.MaxInt64+1))))},
		{"a record without its passkeys and apps", encodeArray(encodeUint(recordSchemaCredential), encodeArray(), empty, empty, empty, empty, empty)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeCredentialRecord(test.record); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
	decoded, err := decodeCredentialRecord(passkeyRecord(encodeArray(element(5, encodeBytes(make([]byte, maxPasskeyKeyBytes))))))
	if err != nil || len(decoded.Passkeys[0].PrivateKey) != maxPasskeyKeyBytes {
		t.Fatalf("a key at its limit: %v", err)
	}
}

func TestPasskeyAcceptsEachLimitAndRefusesPast(t *testing.T) {
	label := strings.Repeat("a", 63)
	longest := strings.Join([]string{label, label, label, label}, ".")
	if len(longest) != MaxOriginLength {
		t.Fatalf("the test relying party is %d characters, not the limit", len(longest))
	}
	tests := []struct {
		name   string
		change func(*Passkey)
		want   error
	}{
		{"shortest credential ID", func(p *Passkey) { p.CredentialID = make([]byte, minCredentialIDBytes) }, nil},
		{"credential ID under its limit", func(p *Passkey) { p.CredentialID = make([]byte, minCredentialIDBytes-1) }, ErrInvalidInput},
		{"longest credential ID", func(p *Passkey) { p.CredentialID = make([]byte, maxCredentialIDBytes) }, nil},
		{"credential ID over its limit", func(p *Passkey) { p.CredentialID = make([]byte, maxCredentialIDBytes+1) }, ErrInvalidInput},
		{"longest relying party", func(p *Passkey) { p.RPID = longest }, nil},
		{"relying party over its limit", func(p *Passkey) { p.RPID = longest + "a" }, ErrInvalidInput},
		{"single-label relying party", func(p *Passkey) { p.RPID = "localhost" }, nil},
		{"relying party in punycode", func(p *Passkey) { p.RPID = "xn--bcher-kva.example" }, nil},
		{"empty relying party", func(p *Passkey) { p.RPID = "" }, ErrInvalidInput},
		{"upper-case relying party", func(p *Passkey) { p.RPID = "Example.com" }, ErrInvalidInput},
		{"relying party with a scheme", func(p *Passkey) { p.RPID = "https://example.com" }, ErrInvalidInput},
		{"relying party with a port", func(p *Passkey) { p.RPID = "example.com:443" }, ErrInvalidInput},
		{"relying party with a path", func(p *Passkey) { p.RPID = "example.com/login" }, ErrInvalidInput},
		{"relying party with a space", func(p *Passkey) { p.RPID = "exa mple.com" }, ErrInvalidInput},
		{"relying party with a trailing dot", func(p *Passkey) { p.RPID = "example.com." }, ErrInvalidInput},
		{"relying party with a leading dot", func(p *Passkey) { p.RPID = ".example.com" }, ErrInvalidInput},
		{"relying party with an empty label", func(p *Passkey) { p.RPID = "example..com" }, ErrInvalidInput},
		{"relying party in Unicode", func(p *Passkey) { p.RPID = "bücher.example" }, ErrInvalidInput},
		{"shortest user handle", func(p *Passkey) { p.UserHandle = []byte{1} }, nil},
		{"empty user handle", func(p *Passkey) { p.UserHandle = nil }, ErrInvalidInput},
		{"longest user handle", func(p *Passkey) { p.UserHandle = make([]byte, maxUserHandleBytes) }, nil},
		{"user handle over its limit", func(p *Passkey) { p.UserHandle = make([]byte, maxUserHandleBytes+1) }, ErrInvalidInput},
		{"empty names", func(p *Passkey) { p.UserName, p.UserDisplayName = "", "" }, nil},
		{"longest user name in characters", func(p *Passkey) { p.UserName = strings.Repeat("п", MaxLoginLength) }, nil},
		{"user name over its limit", func(p *Passkey) { p.UserName = strings.Repeat("a", MaxLoginLength+1) }, ErrInvalidInput},
		{"user name that is not text", func(p *Passkey) { p.UserName = "\xff" }, ErrInvalidInput},
		{"longest display name in characters", func(p *Passkey) { p.UserDisplayName = strings.Repeat("п", MaxLoginLength) }, nil},
		{"display name over its limit", func(p *Passkey) { p.UserDisplayName = strings.Repeat("a", MaxLoginLength+1) }, ErrInvalidInput},
		{"no private key", func(p *Passkey) { p.PrivateKey = nil }, ErrInvalidInput},
		{"private key over its limit", func(p *Passkey) { p.PrivateKey = append(p.PrivateKey, make([]byte, maxPasskeyKeyBytes)...) }, ErrInvalidInput},
		{"largest counter", func(p *Passkey) { p.Counter = math.MaxUint32 }, nil},
		{"created at the epoch", func(p *Passkey) { p.CreatedAt = time.Unix(0, 0) }, nil},
		{"created before the epoch", func(p *Passkey) { p.CreatedAt = time.Unix(-1, 0) }, ErrInvalidInput},
		{"no creation time", func(p *Passkey) { p.CreatedAt = time.Time{} }, ErrInvalidInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			passkey := testPasskey(t, 1)
			test.change(&passkey)
			_, err := acceptInput(CredentialInput{Label: "Example", Passkeys: []Passkey{passkey}})
			if !errors.Is(err, test.want) || test.want == nil && err != nil {
				t.Fatalf("got %v, want %v", err, test.want)
			}
			if valid := ValidPasskey(passkey); valid != (test.want == nil) {
				t.Fatalf("ValidPasskey = %v, want %v", valid, test.want == nil)
			}
		})
	}
}

func TestPasskeyCreationTimeIsKeptInWholeSecondsInUTC(t *testing.T) {
	passkey := testPasskey(t, 1)
	passkey.CreatedAt = time.Date(2026, 9, 23, 12, 30, 15, 999_999_999, time.FixedZone("UTC+2", 2*60*60))
	session, _ := populatedSession(t)
	defer session.Lock()
	id := commitCredential(t, session, CredentialInput{Label: "Example", Passkeys: []Passkey{passkey}}, nil)
	read, err := session.ReadPasskey(id, passkey.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 9, 23, 10, 30, 15, 0, time.UTC); read.CreatedAt != want {
		t.Fatalf("created at %v, want %v", read.CreatedAt, want)
	}
}

func TestPasskeyKeyMustBeP256InPKCS8(t *testing.T) {
	_, edwards, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	edwardsDER, err := x509.MarshalPKCS8PrivateKey(edwards)
	if err != nil {
		t.Fatal(err)
	}
	p256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sec1, err := x509.MarshalECPrivateKey(p256)
	if err != nil {
		t.Fatal(err)
	}
	valid := ecdsaKey(t, elliptic.P256())
	for name, key := range map[string][]byte{
		"P-384":                 ecdsaKey(t, elliptic.P384()),
		"P-521":                 ecdsaKey(t, elliptic.P521()),
		"Ed25519":               edwardsDER,
		"P-256 outside PKCS #8": sec1,
		"trailing bytes":        append(slices.Clone(valid), 0),
		"garbage":               bytes.Repeat([]byte{0x30}, 64),
	} {
		passkey := testPasskey(t, 1)
		passkey.PrivateKey = key
		if _, err := acceptInput(CredentialInput{Label: "Example", Passkeys: []Passkey{passkey}}); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: got %v, want ErrInvalidInput", name, err)
		}
		if ValidPasskey(passkey) {
			t.Errorf("%s: ValidPasskey accepts it", name)
		}
	}
}

func TestCredentialIDIsUniqueWithinACredentialOnly(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	first, again := testPasskey(t, 1), testPasskey(t, 1)
	if _, _, err := session.PrepareCreate(CredentialInput{Label: "Twice", Passkeys: []Passkey{first, again}}, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("one credential ID twice: got %v, want ErrInvalidInput", err)
	}
	held := commitCredential(t, session, CredentialInput{Label: "Held", Passkeys: []Passkey{first}}, nil)
	if _, err := session.PrepareAddPasskey(held, again); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("adding a held credential ID: got %v, want ErrInvalidInput", err)
	}
	other := commitCredential(t, session, CredentialInput{Label: "Other", Passkeys: []Passkey{again}}, nil)
	for _, id := range []ID{held, other} {
		if entry := listedEntry(t, session, id); len(entry.Passkeys) != 1 || !bytes.Equal(entry.Passkeys[0].CredentialID, first.CredentialID) {
			t.Fatalf("passkeys of %s = %+v", id, entry.Passkeys)
		}
	}
}

func TestNinthPasskeyIsRefusedAndTheCredentialIsUnchanged(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	passkeys := make([]Passkey, MaxCredentialPasskeys+1)
	for i := range passkeys {
		passkeys[i] = testPasskey(t, byte(i+1))
	}
	if _, _, err := session.PrepareCreate(CredentialInput{Label: "Crowded", Passkeys: passkeys}, nil); !errors.Is(err, ErrPasskeysFull) {
		t.Fatalf("creating with nine: got %v, want ErrPasskeysFull", err)
	}
	id := commitCredential(t, session, CredentialInput{Label: "Full", Passkeys: passkeys[:MaxCredentialPasskeys]}, nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.PrepareAddPasskey(id, passkeys[MaxCredentialPasskeys]); !errors.Is(err, ErrPasskeysFull) {
		t.Fatalf("adding a ninth: got %v, want ErrPasskeysFull", err)
	}
	after, afterHead, err := session.CurrentContainer()
	if err != nil || !bytes.Equal(before, after) || afterHead != head {
		t.Fatalf("a refused passkey changed the vault: %v", err)
	}
	if entry := listedEntry(t, session, id); len(entry.Passkeys) != MaxCredentialPasskeys {
		t.Fatalf("the full credential lists %d passkeys", len(entry.Passkeys))
	}
	pending, err := session.PrepareEdit(id, CredentialPatch{RemovePasskeys: [][]byte{passkeys[0].CredentialID}})
	commitPending(t, session, pending, err)
	pending, err = session.PrepareAddPasskey(id, passkeys[MaxCredentialPasskeys])
	commitPending(t, session, pending, err)
}

func TestReadingACredentialLeavesItsPrivateKeysBehind(t *testing.T) {
	created, groups := groupedVault(t, "Work")
	session := created.Session
	defer session.Lock()
	first, second := testPasskey(t, 1), testPasskey(t, 2)
	id := commitCredential(t, session, CredentialInput{Label: "Example", Login: "alex", Password: "secret", Passkeys: []Passkey{first, second}}, groups)
	want := []Passkey{withoutKey(first), withoutKey(second)}
	credential, err := session.ReadCredential(id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(credential.Passkeys, want) || credential.Password != "secret" {
		t.Fatalf("read credential holds %+v", credential.Passkeys)
	}
	if selected := selectCredential(t, session, id); !reflect.DeepEqual(selected.Passkeys, want) {
		t.Fatalf("selected credential holds %+v", selected.Passkeys)
	}
	read, err := session.ReadPasskey(id, second.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(read, second) {
		t.Fatalf("read passkey = %+v", read)
	}
	clear(read.PrivateKey)
	again, err := session.ReadPasskey(id, second.CredentialID)
	if err != nil || !bytes.Equal(again.PrivateKey, second.PrivateKey) {
		t.Fatalf("clearing a returned key reached the vault: %v", err)
	}
}

func TestIndexShowsEachCredentialsPasskeyFaces(t *testing.T) {
	created, ids := vaultWith(t,
		CredentialInput{Label: "Bare", Password: "one"},
	)
	session := created.Session
	defer session.Lock()
	first, second := testPasskey(t, 1), testPasskey(t, 2)
	second.RPID, second.UserName, second.Discoverable = "example.org", "user", false
	held := commitCredential(t, session, CredentialInput{Label: "Held", Passkeys: []Passkey{first, second}}, nil)
	identity := commitIdentity(t, session, IdentityInput{Label: "Me"}, nil)
	card := commitCard(t, session, fullCard(), nil)
	want := map[ID][]PasskeyFace{ids[0]: nil, held: clonedFaces([]PasskeyFace{faceOfPasskey(first), faceOfPasskey(second)}), identity: nil, card: nil}
	wantFaces := func(session *Session) {
		t.Helper()
		for id, faces := range want {
			if entry := listedEntry(t, session, id); !reflect.DeepEqual(entry.Passkeys, faces) {
				t.Fatalf("faces of %s = %+v, want %+v", id, entry.Passkeys, faces)
			}
		}
	}
	wantFaces(session)

	listed := listedEntry(t, session, held)
	listed.Passkeys[0].CredentialID[0] ^= 0xff
	listed.Passkeys[0].UserHandle[0] ^= 0xff
	second.CredentialID[0] ^= 0xff
	second.UserHandle[0] ^= 0xff
	wantFaces(session)
	second.CredentialID[0] ^= 0xff
	second.UserHandle[0] ^= 0xff

	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	wantFaces(reopened)

	preview, err := PreviewNewItem(NewItem{Credential: &CredentialInput{Label: "Imported", Passkeys: []Passkey{first}}})
	if err != nil || !reflect.DeepEqual(preview.Passkeys, []PasskeyFace{faceOfPasskey(first)}) {
		t.Fatalf("preview faces = %+v, error = %v", preview.Passkeys, err)
	}
}

func TestIndexRoundTripsPasskeyFaces(t *testing.T) {
	faces := make([]PasskeyFace, MaxCredentialPasskeys)
	for i := range faces {
		faces[i] = PasskeyFace{
			CredentialID: bytes.Repeat([]byte{byte(i + 1)}, minCredentialIDBytes+i), RPID: "example.com", UserName: strings.Repeat("п", i),
			UserHandle: bytes.Repeat([]byte{byte(i)}, i+1), UserDisplayName: strings.Repeat("д", i), Discoverable: i%2 == 0,
		}
	}
	faces[0].CredentialID = make([]byte, maxCredentialIDBytes)
	faces[1].UserHandle, faces[1].UserDisplayName = make([]byte, maxUserHandleBytes), strings.Repeat("д", MaxLoginLength)
	records := stubRecords(2)
	entries := withDigests([]entryMeta{
		{id: ID{1}, revision: 1, label: "Held", kind: KindCredential, passkeys: faces},
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
	entries[0].passkeys = append(faces, faces[0])
	if _, err := encodeIndex(3, nil, entries, nil, DefaultTrashRetention); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("nine faces: got %v, want ErrResourceLimit", err)
	}
}

func TestIndexRejectsMalformedPasskeyFaces(t *testing.T) {
	records := stubRecords(1)
	digest := withDigests(make([]entryMeta, 1), records)[0].digest
	id := ID{5}
	face := func(credentialID []byte, rpID, userName string, discoverable []byte) []byte {
		return encodeArray(encodeBytes(credentialID), encodeBytes([]byte(rpID)), encodeBytes([]byte(userName)), discoverable, encodeBytes([]byte{1}), encodeBytes([]byte("Alex")))
	}
	valid := face(make([]byte, minCredentialIDBytes), "example.com", "alex", encodeUint(1))
	userFace := func(user ...[]byte) []byte {
		return encodeArray(append([][]byte{encodeBytes(make([]byte, minCredentialIDBytes)), encodeBytes([]byte("example.com")), encodeBytes([]byte("alex")), encodeUint(1)}, user...)...)
	}
	crowded := make([][]byte, MaxCredentialPasskeys+1)
	for i := range crowded {
		crowded[i] = face(bytes.Repeat([]byte{byte(i)}, minCredentialIDBytes), "example.com", "", encodeUint(0))
	}
	element := func(kind Kind, faces ...[]byte) []byte {
		return withField(id, digest, uint64(kind), encodeArray(), fieldPasskeys, encodeArray(faces...))
	}
	if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), element(KindCredential, valid)), records); err != nil {
		t.Fatalf("a valid face: %v", err)
	}
	tests := []struct {
		name    string
		element []byte
	}{
		{"identity with a passkey", element(KindIdentity, valid)},
		{"face of four fields, without the user", element(KindCredential, userFace())},
		{"face of three fields", element(KindCredential, encodeArray(encodeBytes(make([]byte, minCredentialIDBytes)), encodeBytes([]byte("example.com")), encodeBytes(nil)))},
		{"face that is not an array", element(KindCredential, encodeBytes(nil))},
		{"discoverable two", element(KindCredential, face(make([]byte, minCredentialIDBytes), "example.com", "", encodeUint(2)))},
		{"credential ID under its limit", element(KindCredential, face(make([]byte, minCredentialIDBytes-1), "example.com", "", encodeUint(0)))},
		{"credential ID over its limit", element(KindCredential, face(make([]byte, maxCredentialIDBytes+1), "example.com", "", encodeUint(0)))},
		{"upper-case relying party", element(KindCredential, face(make([]byte, minCredentialIDBytes), "Example.com", "", encodeUint(0)))},
		{"user name over its limit", element(KindCredential, face(make([]byte, minCredentialIDBytes), "example.com", strings.Repeat("a", MaxLoginLength+1), encodeUint(0)))},
		{"user name that is not text", element(KindCredential, face(make([]byte, minCredentialIDBytes), "example.com", "\xff", encodeUint(0)))},
		{"one credential ID twice", element(KindCredential, valid, valid)},
		{"more faces than a credential holds", element(KindCredential, crowded...)},
		{"face of five fields", element(KindCredential, userFace(encodeBytes([]byte{1})))},
		{"face of seven fields", element(KindCredential, userFace(encodeBytes([]byte{1}), encodeBytes(nil), encodeBytes(nil)))},
		{"user fields with an empty handle", element(KindCredential, userFace(encodeBytes(nil), encodeBytes([]byte("Alex"))))},
		{"user handle over its limit", element(KindCredential, userFace(encodeBytes(make([]byte, maxUserHandleBytes+1)), encodeBytes(nil)))},
		{"user handle that is not bytes", element(KindCredential, userFace(encodeUint(1), encodeBytes(nil)))},
		{"display name over its limit", element(KindCredential, userFace(encodeBytes([]byte{1}), encodeBytes([]byte(strings.Repeat("a", MaxLoginLength+1)))))},
		{"display name that is not text", element(KindCredential, userFace(encodeBytes([]byte{1}), encodeBytes([]byte("\xff"))))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), test.element), records); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestAVaultWithAFaceWithoutTheUserIsMalformed(t *testing.T) {
	passkey := testPasskey(t, 1)
	created, _ := vaultWith(t, CredentialInput{Label: "Example", Password: "one", Passkeys: []Passkey{passkey}})
	session := created.Session
	defer session.Lock()
	raw, head := currentRaw(t, session)
	entry := session.entries[0]
	fields := currentFields(entry.id, entry.digest, uint64(KindCredential), encodeArray())
	fields[fieldLabel] = encodeBytes([]byte("Example"))
	fields[fieldPasskeys] = encodeArray(encodeArray(encodeBytes(passkey.CredentialID), encodeBytes([]byte(passkey.RPID)), encodeBytes([]byte(passkey.UserName)), encodeUint(1)))
	faceless := withIndex(t, session, raw, indexPlaintext(head.Revision, head.PreviousHash, encodeArray(), encodeArray(fields...)))
	if _, err := OpenWithRecovery(faceless, created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
}

func TestPasskeyChangesResealOnlyTheirRecord(t *testing.T) {
	counted := testPasskey(t, 2)
	counted.Counter = 41
	tests := []struct {
		name   string
		change func(*Session, ID) (*Pending, error)
	}{
		{"add", func(session *Session, id ID) (*Pending, error) {
			return session.PrepareAddPasskey(id, testPasskey(t, 3))
		}},
		{"remove", func(session *Session, id ID) (*Pending, error) {
			return session.PrepareEdit(id, CredentialPatch{RemovePasskeys: [][]byte{bytes.Repeat([]byte{1}, minCredentialIDBytes)}})
		}},
		{"count", func(session *Session, id ID) (*Pending, error) {
			_, pending, err := session.PrepareCountPasskeyUse(id, counted.CredentialID)
			return pending, err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			created, ids := vaultWith(t,
				CredentialInput{Label: "Before", Password: "one"},
				CredentialInput{Label: "Held", Password: "two", Passkeys: []Passkey{testPasskey(t, 1), counted}},
				CredentialInput{Label: "After", Password: "three", Passkeys: []Passkey{testPasskey(t, 1)}},
			)
			session := created.Session
			defer session.Lock()
			note := commitNote(t, session, fullNote(), nil)
			before := sealedRecords(t, session)
			pending, err := test.change(session, ids[1])
			commitPending(t, session, pending, err)
			after := sealedRecords(t, session)
			for _, id := range []ID{ids[0], ids[2], note} {
				if !bytes.Equal(before[id], after[id]) {
					t.Fatalf("the record of %s was sealed again", id)
				}
			}
			if bytes.Equal(before[ids[1]], after[ids[1]]) {
				t.Fatal("the changed record was not sealed again")
			}
			if value := selectCredential(t, session, ids[1]); value.Password != "two" {
				t.Fatalf("the changed credential = %+v", value.CredentialInput)
			}
		})
	}
}

func TestEditRemovingAPasskeyKeepsTheOthers(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	first, second := testPasskey(t, 1), testPasskey(t, 2)
	id := commitCredential(t, session, CredentialInput{Label: "Example", Websites: []string{"https://example.com"}, Password: "secret", Passkeys: []Passkey{first, second}}, nil)
	pending, err := session.PrepareEdit(id, CredentialPatch{RemovePasskeys: [][]byte{first.CredentialID}})
	commitPending(t, session, pending, err)
	if faces := listedEntry(t, session, id).Passkeys; !reflect.DeepEqual(faces, []PasskeyFace{faceOfPasskey(second)}) {
		t.Fatalf("faces after a removal = %+v", faces)
	}
	if _, err := session.ReadPasskey(id, first.CredentialID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reading a removed passkey: got %v, want ErrNotFound", err)
	}
	if read, err := session.ReadPasskey(id, second.CredentialID); err != nil || !reflect.DeepEqual(read, second) {
		t.Fatalf("the remaining passkey = %+v, error = %v", read, err)
	}

	pending, err = session.PrepareEdit(id, CredentialPatch{RemovePasskeys: [][]byte{second.CredentialID}})
	commitPending(t, session, pending, err)
	credential := selectCredential(t, session, id)
	if credential.Passkeys != nil || credential.Password != "secret" || listedEntry(t, session, id).Passkeys != nil {
		t.Fatalf("the credential without passkeys = %+v", credential.CredentialInput)
	}
}

func TestEditRemovesPasskeysWithItsFieldsInOneSave(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	first, second, third := testPasskey(t, 1), testPasskey(t, 2), testPasskey(t, 3)
	id := commitCredential(t, session, CredentialInput{Label: "Example", Password: "secret", Passkeys: []Passkey{first, second, third}}, nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	password := "rotated"
	missing := bytes.Repeat([]byte{0xee}, minCredentialIDBytes)
	for name, removed := range map[string][][]byte{
		"an unknown passkey":      {first.CredentialID, missing},
		"one passkey named twice": {first.CredentialID, first.CredentialID},
	} {
		if _, err := session.PrepareEdit(id, CredentialPatch{Password: &password, RemovePasskeys: removed}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("%s: got %v, want ErrNotFound", name, err)
		}
	}
	after, afterHead, err := session.CurrentContainer()
	if err != nil || !bytes.Equal(before, after) || afterHead != head {
		t.Fatalf("a refused edit changed the vault: %v", err)
	}

	pending, err := session.PrepareEdit(id, CredentialPatch{Password: &password, RemovePasskeys: [][]byte{third.CredentialID, first.CredentialID}})
	commitPending(t, session, pending, err)
	if _, afterHead, err := session.CurrentContainer(); err != nil || afterHead.Revision != head.Revision+1 {
		t.Fatalf("the edit took revision %d after %d, error = %v", afterHead.Revision, head.Revision, err)
	}
	credential := selectCredential(t, session, id)
	if credential.Password != password || !reflect.DeepEqual(credential.Passkeys, []Passkey{withoutKey(second)}) {
		t.Fatalf("the edited credential = %+v", credential.CredentialInput)
	}
	if faces := listedEntry(t, session, id).Passkeys; !reflect.DeepEqual(faces, []PasskeyFace{faceOfPasskey(second)}) {
		t.Fatalf("faces after the edit = %+v", faces)
	}
}

func TestEditKeepsPasskeys(t *testing.T) {
	created, ids := vaultWith(t,
		CredentialInput{Label: "Example", Password: "secret", Passkeys: []Passkey{testPasskey(t, 1)}},
	)
	session := created.Session
	defer session.Lock()
	passkey := testPasskey(t, 2)
	pending, err := session.PrepareAddPasskey(ids[0], passkey)
	commitPending(t, session, pending, err)
	faces := listedEntry(t, session, ids[0]).Passkeys
	password := "rotated"
	pending, err = session.PrepareEdit(ids[0], CredentialPatch{Password: &password})
	commitPending(t, session, pending, err)
	if got := listedEntry(t, session, ids[0]).Passkeys; !reflect.DeepEqual(got, faces) {
		t.Fatalf("faces after an edit = %+v, want %+v", got, faces)
	}
	read, err := session.ReadPasskey(ids[0], passkey.CredentialID)
	if err != nil || !reflect.DeepEqual(read, passkey) {
		t.Fatalf("passkey after an edit = %+v, error = %v", read, err)
	}
	if value := selectCredential(t, session, ids[0]); value.Password != password || len(value.Passkeys) != 2 {
		t.Fatalf("edited credential = %+v", value.CredentialInput)
	}
}

func TestPasskeyCounterAdvancesUnlessItIsZero(t *testing.T) {
	zero, counted, largest := testPasskey(t, 1), testPasskey(t, 2), testPasskey(t, 3)
	counted.Counter, largest.Counter = 41, math.MaxUint32
	created, ids := vaultWith(t,
		CredentialInput{Label: "Example", Password: "secret", Passkeys: []Passkey{zero, counted, largest}},
	)
	session := created.Session
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	counter, pending, err := session.PrepareCountPasskeyUse(ids[0], zero.CredentialID)
	if err != nil || counter != 0 || pending != nil {
		t.Fatalf("a zero counter = %d, pending %v, error %v", counter, pending, err)
	}
	if _, _, err := session.PrepareCountPasskeyUse(ids[0], largest.CredentialID); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("the largest counter: got %v, want ErrResourceLimit", err)
	}
	after, afterHead, err := session.CurrentContainer()
	if err != nil || !bytes.Equal(before, after) || afterHead != head {
		t.Fatalf("counting a zero or the largest counter changed the vault: %v", err)
	}
	counter, pending, err = session.PrepareCountPasskeyUse(ids[0], counted.CredentialID)
	if counter != 42 {
		t.Fatalf("counter 41 advanced to %d", counter)
	}
	commitPending(t, session, pending, err)
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	session.Lock()
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	for credentialID, want := range map[string]uint32{string(zero.CredentialID): 0, string(counted.CredentialID): 42, string(largest.CredentialID): math.MaxUint32} {
		read, err := reopened.ReadPasskey(ids[0], []byte(credentialID))
		if err != nil || read.Counter != want {
			t.Fatalf("counter = %d, want %d, error = %v", read.Counter, want, err)
		}
	}
}

func TestMissingPasskeyOrCredentialIsNotFound(t *testing.T) {
	session, bare := populatedSession(t)
	defer session.Lock()
	passkey := testPasskey(t, 1)
	held := commitCredential(t, session, CredentialInput{Label: "Held", Passkeys: []Passkey{passkey}}, nil)
	identity := commitIdentity(t, session, IdentityInput{Label: "Me"}, nil)
	missing := bytes.Repeat([]byte{0xee}, minCredentialIDBytes)
	tests := []struct {
		name         string
		id           ID
		credentialID []byte
	}{
		{"a passkey the credential does not hold", held, missing},
		{"a credential without passkeys", bare, passkey.CredentialID},
		{"an identity", identity, passkey.CredentialID},
		{"an item the vault does not hold", ID{0xff}, passkey.CredentialID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := session.ReadPasskey(test.id, test.credentialID); !errors.Is(err, ErrNotFound) {
				t.Errorf("read: got %v, want ErrNotFound", err)
			}
			if _, err := session.PrepareEdit(test.id, CredentialPatch{RemovePasskeys: [][]byte{test.credentialID}}); !errors.Is(err, ErrNotFound) {
				t.Errorf("remove: got %v, want ErrNotFound", err)
			}
			if _, _, err := session.PrepareCountPasskeyUse(test.id, test.credentialID); !errors.Is(err, ErrNotFound) {
				t.Errorf("count: got %v, want ErrNotFound", err)
			}
		})
	}
	for _, id := range []ID{identity, {0xff}} {
		if _, err := session.PrepareAddPasskey(id, testPasskey(t, 2)); !errors.Is(err, ErrNotFound) {
			t.Errorf("adding to %s: got %v, want ErrNotFound", id, err)
		}
	}
}

func TestPrivateKeyIsOnlyInItsCredentialsRecord(t *testing.T) {
	created, ids := vaultWith(t,
		CredentialInput{Label: "Other", Password: "one"},
	)
	session := created.Session
	defer session.Lock()
	first, second := testPasskey(t, 1), testPasskey(t, 2)
	held := commitCredential(t, session, CredentialInput{Label: "Held", Password: "two", Passkeys: []Passkey{first}}, nil)
	pending, err := session.PrepareAddPasskey(held, second)
	commitPending(t, session, pending, err)
	commitNote(t, session, fullNote(), nil)
	keys := [][]byte{first.PrivateKey, second.PrivateKey}

	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := currentRaw(t, session)
	index, err := openBox(session.indexKey, raw.index, indexAAD(raw.vaultID, raw.recovery))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if bytes.Contains(index, key) {
			t.Fatal("the index plaintext holds a private key")
		}
		if bytes.Contains(container, key) {
			t.Fatal("the container holds a private key in plaintext")
		}
	}
	for i, entry := range session.entries {
		plaintext, err := session.openRecord(i)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			if holds := bytes.Contains(plaintext, key); holds != (entry.id == held) {
				t.Fatalf("record of %s holds a private key: %v", entry.id, holds)
			}
		}
	}
	if entry := listedEntry(t, session, ids[0]); entry.Passkeys != nil {
		t.Fatalf("a credential without passkeys lists %+v", entry.Passkeys)
	}
}

func TestBatchAddsCredentialsWithPasskeys(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	first, second := testPasskey(t, 1), testPasskey(t, 2)
	result := commitBatch(t, session, []NewItem{
		{Credential: &CredentialInput{Label: "Imported", Login: "alex", Password: "secret", Passkeys: []Passkey{first, second}}, Groups: []string{"Imported"}},
		{Credential: &CredentialInput{Label: "Same passkey elsewhere", Passkeys: []Passkey{first}}},
	})
	if faces := listedEntry(t, session, result.Items[0]).Passkeys; !reflect.DeepEqual(faces, []PasskeyFace{faceOfPasskey(first), faceOfPasskey(second)}) {
		t.Fatalf("imported faces = %+v", faces)
	}
	for _, id := range result.Items {
		if read, err := session.ReadPasskey(id, first.CredentialID); err != nil || !reflect.DeepEqual(read, first) {
			t.Fatalf("imported passkey of %s = %+v, error = %v", id, read, err)
		}
	}
	invalid := testPasskey(t, 3)
	invalid.RPID = "Example.com"
	crowded := make([]Passkey, MaxCredentialPasskeys+1)
	for i := range crowded {
		crowded[i] = testPasskey(t, byte(i+1))
	}
	for name, test := range map[string]struct {
		passkeys []Passkey
		want     error
	}{
		"an invalid passkey": {[]Passkey{invalid}, ErrInvalidInput},
		"nine passkeys":      {crowded, ErrPasskeysFull},
	} {
		items := []NewItem{{Note: &NoteInput{Label: "Kept out", Body: "body"}}, {Credential: &CredentialInput{Label: "Refused", Passkeys: test.passkeys}}}
		if _, _, err := session.PrepareAddItems(items); !errors.Is(err, test.want) {
			t.Errorf("%s: got %v, want %v", name, err, test.want)
		}
	}
}
