package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func fullIdentity() IdentityInput {
	return IdentityInput{
		Label:    "  Alex at home  ",
		FullName: "Алекс Ünïcødé Example",
		Birthday: "1990-04-17",
		Emails:   []string{"alex@example.test", " Second@Example.test "},
		Phones:   []string{"+1 555 0100", "+7 900 000-00-00"},
		Addresses: []Address{
			{Label: "Home", Street: "1 Example Street\nFlat 2", City: "Springfield", Region: "Oregon", PostalCode: "97403", Country: "United States"},
			{Street: "Unter den Linden 5", City: "Berlin"},
		},
		Documents: []Document{
			{Type: DocumentPassport, Number: "X1234567", Issuer: "Example Passport Office", IssuedOn: "2020-01-15", ExpiresOn: "2030-01-14"},
			{Type: DocumentDriversLicense, Number: "DL-424242", ExpiresOn: "2027-06-30"},
			{Type: DocumentIDCard, Number: "ID 990099"},
			{Type: DocumentTaxNumber, Number: "123-45-678"},
			{Type: DocumentOther, Label: "Library card", Number: "LIB-7777", IssuedOn: "2024-02-29", ExpiresOn: "2024-02-29"},
		},
		Notes: " line 1\nline 2 ",
	}
}

// withoutAddressIDs is an identity as given, before the vault draws its address ids.
func withoutAddressIDs(input IdentityInput) IdentityInput {
	input.Addresses = slices.Clone(input.Addresses)
	for i := range input.Addresses {
		input.Addresses[i].ID = ID{}
	}
	return input
}

// withAddressIDs is an identity as a record holds it, with ids {1}, {2}, … for its addresses.
func withAddressIDs(input IdentityInput) IdentityInput {
	input.Addresses = slices.Clone(input.Addresses)
	for i := range input.Addresses {
		input.Addresses[i].ID = ID{byte(i + 1)}
	}
	return input
}

func commitIdentity(t *testing.T, session *Session, input IdentityInput, groups []ID) ID {
	t.Helper()
	pending, id, err := session.PrepareCreateIdentity(input, groups)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	return id
}

func selectIdentity(t *testing.T, session *Session, id ID) Identity {
	t.Helper()
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	value, err := session.ReadSelectedIdentity(ticket)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func listedEntry(t *testing.T, session *Session, id ID) Entry {
	t.Helper()
	entries, err := session.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == id {
			return entry
		}
	}
	t.Fatalf("item %s is not listed", id)
	return Entry{}
}

// withIndex seals an index plaintext under the session's keys in place of the container's own.
func withIndex(t *testing.T, session *Session, raw rawContainer, plaintext []byte) []byte {
	t.Helper()
	box, err := seal(session.indexKey, plaintext, indexAAD(raw.vaultID, raw.recovery))
	if err != nil {
		t.Fatal(err)
	}
	container, err := encodeContainer(raw.vaultID, raw.recovery, box, raw.records)
	if err != nil {
		t.Fatal(err)
	}
	return container
}

func currentRaw(t *testing.T, session *Session) (rawContainer, Head) {
	t.Helper()
	container, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := parseContainer(container)
	if err != nil {
		t.Fatal(err)
	}
	return raw, head
}

func TestIdentityRoundTripsEveryPart(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail", Login: "alex", Password: "secret"})
	input := fullIdentity()
	id := commitIdentity(t, created.Session, input, nil)
	container, _, err := created.Session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	created.Session.Lock()
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	identity := selectIdentity(t, reopened, id)
	if identity.ID != id || !reflect.DeepEqual(withoutAddressIDs(identity.IdentityInput), input) {
		t.Fatalf("identity changed across a reopen: %#v", identity.IdentityInput)
	}
	if first, second := identity.Addresses[0].ID, identity.Addresses[1].ID; first == (ID{}) || second == (ID{}) || first == second {
		t.Fatalf("address ids = %s, %s", first, second)
	}
	entry := listedEntry(t, reopened, id)
	if entry.Kind != KindIdentity || entry.Label != input.Label || entry.Detail != "alex@example.test" || entry.ExpiresOn != "2024-02-29" {
		t.Fatalf("identity entry = %+v", entry)
	}
	credential := listedEntry(t, reopened, ids[0])
	if credential.Kind != KindCredential || credential.Detail != "alex" || credential.ExpiresOn != "" {
		t.Fatalf("credential entry = %+v", credential)
	}
}

func TestIdentityWithoutPartsHasEmptySummaryAndCanonicalRecord(t *testing.T) {
	withNil, err := acceptIdentity(IdentityInput{Label: "Bare"})
	if err != nil {
		t.Fatal(err)
	}
	withEmpty, err := acceptIdentity(IdentityInput{Label: "Bare", Emails: []string{}, Phones: []string{}, Addresses: []Address{}, Documents: []Document{}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(withNil, withEmpty) {
		t.Fatalf("empty lists were kept apart from missing ones: %#v", withEmpty)
	}
	first, err := encodeIdentityRecord(withNil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := encodeIdentityRecord(withEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("the same identity encoded to different bytes")
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	id := commitIdentity(t, session, IdentityInput{Label: "Bare", Documents: []Document{{Type: DocumentIDCard, Number: "1"}}}, nil)
	if entry := listedEntry(t, session, id); entry.Detail != "" || entry.ExpiresOn != "" {
		t.Fatalf("an identity without emails or expiries reported %+v", entry)
	}
	if identity := selectIdentity(t, session, id); identity.Emails != nil || identity.Addresses != nil {
		t.Fatalf("empty lists came back as %#v", identity.IdentityInput)
	}
}

func TestIdentityRefusesEachBoundWithoutWriting(t *testing.T) {
	over := func(limit int) string { return strings.Repeat("ü", limit+1) }
	tests := []struct {
		name   string
		mutate func(*IdentityInput)
	}{
		{"empty label", func(input *IdentityInput) { input.Label = "" }},
		{"blank label", func(input *IdentityInput) { input.Label = " \t\n" }},
		{"label over its limit", func(input *IdentityInput) { input.Label = over(MaxLabelLength) }},
		{"label that is not text", func(input *IdentityInput) { input.Label = "\xff" }},
		{"full name over its limit", func(input *IdentityInput) { input.FullName = over(MaxFullNameLength) }},
		{"notes over their limit", func(input *IdentityInput) { input.Notes = over(MaxNotesLength) }},
		{"impossible birthday", func(input *IdentityInput) { input.Birthday = "1990-02-30" }},
		{"birthday in another form", func(input *IdentityInput) { input.Birthday = "17.04.1990" }},
		{"birthday with space around it", func(input *IdentityInput) { input.Birthday = " 1990-04-17" }},
		{"birthday before 1900", func(input *IdentityInput) { input.Birthday = "1899-12-31" }},
		{"birthday in year zero", func(input *IdentityInput) { input.Birthday = "0000-01-01" }},
		{"too many emails", func(input *IdentityInput) { input.Emails = repeatWord("alex@example.test", MaxIdentityEmails+1) }},
		{"empty email", func(input *IdentityInput) { input.Emails = []string{"alex@example.test", ""} }},
		{"blank email", func(input *IdentityInput) { input.Emails = []string{" "} }},
		{"email over its limit", func(input *IdentityInput) { input.Emails = []string{over(MaxEmailLength)} }},
		{"too many phones", func(input *IdentityInput) { input.Phones = repeatWord("+1 555 0100", MaxIdentityPhones+1) }},
		{"empty phone", func(input *IdentityInput) { input.Phones = []string{""} }},
		{"phone over its limit", func(input *IdentityInput) { input.Phones = []string{over(MaxPhoneLength)} }},
		{"too many addresses", func(input *IdentityInput) {
			input.Addresses = make([]Address, MaxIdentityAddresses+1)
			for i := range input.Addresses {
				input.Addresses[i].City = "Berlin"
			}
		}},
		{"address with only a label", func(input *IdentityInput) { input.Addresses = []Address{{Label: "Home"}} }},
		{"address of blank parts", func(input *IdentityInput) { input.Addresses = []Address{{Street: " ", City: "\t"}} }},
		{"address label over its limit", func(input *IdentityInput) { input.Addresses[0].Label = over(MaxPartLabelLength) }},
		{"street over its limit", func(input *IdentityInput) { input.Addresses[0].Street = over(MaxStreetLength) }},
		{"city over its limit", func(input *IdentityInput) { input.Addresses[0].City = over(MaxCityLength) }},
		{"region over its limit", func(input *IdentityInput) { input.Addresses[0].Region = over(MaxRegionLength) }},
		{"postal code over its limit", func(input *IdentityInput) { input.Addresses[0].PostalCode = over(MaxPostalCodeLength) }},
		{"country over its limit", func(input *IdentityInput) { input.Addresses[0].Country = over(MaxCountryLength) }},
		{"too many documents", func(input *IdentityInput) {
			input.Documents = make([]Document, MaxIdentityDocuments+1)
			for i := range input.Documents {
				input.Documents[i] = Document{Type: DocumentIDCard, Number: "1"}
			}
		}},
		{"document without a number", func(input *IdentityInput) { input.Documents[0].Number = "" }},
		{"document with a blank number", func(input *IdentityInput) { input.Documents[0].Number = "  " }},
		{"document number over its limit", func(input *IdentityInput) { input.Documents[0].Number = over(MaxDocumentNumberLength) }},
		{"issuer over its limit", func(input *IdentityInput) { input.Documents[0].Issuer = over(MaxIssuerLength) }},
		{"document label over its limit", func(input *IdentityInput) { input.Documents[4].Label = over(MaxPartLabelLength) }},
		{"other document without a label", func(input *IdentityInput) { input.Documents[4].Label = "" }},
		{"other document with a blank label", func(input *IdentityInput) { input.Documents[4].Label = " " }},
		{"passport with a label", func(input *IdentityInput) { input.Documents[0].Label = "Mine" }},
		{"driver's license with a label", func(input *IdentityInput) { input.Documents[1].Label = "Car" }},
		{"ID card with a label", func(input *IdentityInput) { input.Documents[2].Label = "Card" }},
		{"tax number with a label", func(input *IdentityInput) { input.Documents[3].Label = "Tax" }},
		{"typed document with a blank label", func(input *IdentityInput) { input.Documents[0].Label = " " }},
		{"document type zero", func(input *IdentityInput) { input.Documents[0].Type = 0 }},
		{"document type past the known ones", func(input *IdentityInput) { input.Documents[0].Type = DocumentOther + 1 }},
		{"impossible expiry", func(input *IdentityInput) { input.Documents[0].ExpiresOn = "2026-02-30" }},
		{"impossible issue date", func(input *IdentityInput) { input.Documents[0].IssuedOn = "2021-13-01" }},
		{"issued after it expires", func(input *IdentityInput) {
			input.Documents[0].IssuedOn, input.Documents[0].ExpiresOn = "2030-01-02", "2030-01-01"
		}},
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	existing := commitIdentity(t, session, fullIdentity(), nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := fullIdentity()
			test.mutate(&input)
			if _, _, err := session.PrepareCreateIdentity(input, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("create: got %v, want ErrInvalidInput", err)
			}
			if _, err := session.PrepareEditIdentity(existing, input, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("edit: got %v, want ErrInvalidInput", err)
			}
		})
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused identity changed the vault")
	}
	if identity := selectIdentity(t, session, existing); !reflect.DeepEqual(withoutAddressIDs(identity.IdentityInput), fullIdentity()) {
		t.Fatalf("a refused edit changed the identity: %#v", identity.IdentityInput)
	}
}

func TestIdentityAtEveryLimitRoundTrips(t *testing.T) {
	at := func(limit int) string { return strings.Repeat("ü", limit) }
	input := IdentityInput{
		Label:    at(MaxLabelLength),
		FullName: at(MaxFullNameLength),
		Birthday: "1900-01-01",
		Notes:    at(MaxNotesLength),
	}
	for range MaxIdentityEmails {
		input.Emails = append(input.Emails, at(MaxEmailLength))
	}
	for range MaxIdentityPhones {
		input.Phones = append(input.Phones, at(MaxPhoneLength))
	}
	for range MaxIdentityAddresses {
		input.Addresses = append(input.Addresses, Address{
			Label: at(MaxPartLabelLength), Street: at(MaxStreetLength), City: at(MaxCityLength),
			Region: at(MaxRegionLength), PostalCode: at(MaxPostalCodeLength), Country: at(MaxCountryLength),
		})
	}
	for range MaxIdentityDocuments {
		input.Documents = append(input.Documents, Document{
			Type: DocumentOther, Label: at(MaxPartLabelLength), Number: at(MaxDocumentNumberLength),
			Issuer: at(MaxIssuerLength), IssuedOn: "2000-01-01", ExpiresOn: "2000-01-01",
		})
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	id := commitIdentity(t, session, input, nil)
	if identity := selectIdentity(t, session, id); !reflect.DeepEqual(withoutAddressIDs(identity.IdentityInput), input) {
		t.Fatal("an identity at every limit changed on its way through the vault")
	}
}

func TestIdentityRecordSchemaMustMatchItsKind(t *testing.T) {
	credential, err := encodeCredentialRecord(CredentialInput{Label: "Mail", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	credentialWithCode, err := encodeCredentialRecord(CredentialInput{Label: "Mail", Password: "secret", TOTP: standardSecret})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := encodeIdentityRecord(withAddressIDs(fullIdentity()))
	if err != nil {
		t.Fatal(err)
	}
	unknownType, err := encodeIdentityRecord(IdentityInput{Documents: []Document{{Type: DocumentOther + 1, Number: "1"}}})
	if err != nil {
		t.Fatal(err)
	}
	zeroType, err := encodeIdentityRecord(IdentityInput{Documents: []Document{{Number: "1"}}})
	if err != nil {
		t.Fatal(err)
	}
	empty := encodeBytes(nil)
	unknownSchema := encodeArray(encodeUint(15), empty, empty, empty, empty, empty)
	addressID := ID{1}
	shortAddress := encodeArray(encodeUint(recordSchemaIdentity), empty, empty, encodeArray(), encodeArray(),
		encodeArray(encodeArray(encodeBytes(addressID[:]), empty, empty, empty, empty, empty)), encodeArray(), empty, empty)
	if _, err := decodeIdentityRecord(identity); err != nil {
		t.Fatalf("a valid identity record was refused: %v", err)
	}
	tests := []struct {
		name   string
		decode func([]byte) error
		record []byte
		want   error
	}{
		{"credential read as an identity", decodeAsIdentity, credential, ErrMalformed},
		{"credential with a code read as an identity", decodeAsIdentity, credentialWithCode, ErrMalformed},
		{"identity read as a credential", decodeAsCredential, identity, ErrMalformed},
		{"unknown schema read as an identity", decodeAsIdentity, unknownSchema, ErrUnsupported},
		{"unknown schema read as a credential", decodeAsCredential, unknownSchema, ErrUnsupported},
		{"document type past the known ones", decodeAsIdentity, unknownType, ErrUnsupported},
		{"document type zero", decodeAsIdentity, zeroType, ErrUnsupported},
		{"address of five parts", decodeAsIdentity, shortAddress, ErrMalformed},
		{"trailing bytes", decodeAsIdentity, append(append([]byte(nil), identity...), 0), ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode(test.record); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func decodeAsIdentity(record []byte) error {
	_, err := decodeIdentityRecord(record)
	return err
}

func decodeAsCredential(record []byte) error {
	_, err := decodeCredentialRecord(record)
	return err
}

func TestEntryKindThatDoesNotMatchItsRecordIsMalformed(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
	session := created.Session
	defer session.Lock()
	commitIdentity(t, session, IdentityInput{Label: "Me", FullName: "Alex"}, nil)
	raw, head := currentRaw(t, session)
	tests := []struct {
		name  string
		index int
		kind  Kind
	}{
		{"credential record listed as an identity", 0, KindIdentity},
		{"identity record listed as a credential", 1, KindCredential},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := append([]entryMeta(nil), session.entries...)
			entries[test.index].kind = test.kind
			plaintext, err := encodeIndex(head.Revision, session.ancestry, entries, nil, DefaultTrashRetention)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := OpenWithRecovery(withIndex(t, session, raw, plaintext), created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestCrossingKindsIsRefusedAsNotFound(t *testing.T) {
	session, credential := populatedSession(t)
	defer session.Lock()
	identity := commitIdentity(t, session, fullIdentity(), nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	identityTicket, err := session.BeginSelection(identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadSelected(identityTicket); !errors.Is(err, ErrNotFound) {
		t.Fatalf("credential read of an identity: %v", err)
	}
	credentialTicket, err := session.BeginSelection(credential)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadSelectedIdentity(credentialTicket); !errors.Is(err, ErrNotFound) {
		t.Fatalf("identity read of a credential: %v", err)
	}
	password := "changed"
	if _, err := session.PrepareEdit(identity, CredentialPatch{Password: &password}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("credential edit of an identity: %v", err)
	}
	if _, err := session.PrepareEditIdentity(credential, fullIdentity(), nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("identity edit of a credential: %v", err)
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused crossing changed the vault")
	}
	if value := selectCredential(t, session, credential); value.Password != "secret" {
		t.Fatalf("credential after refused crossings = %+v", value.CredentialInput)
	}
	if value := selectIdentity(t, session, identity); !reflect.DeepEqual(withoutAddressIDs(value.IdentityInput), fullIdentity()) {
		t.Fatalf("identity after refused crossings = %#v", value.IdentityInput)
	}
}

func TestItemOperationsActOnIdentities(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family", "Travel")
	session := created.Session
	defer session.Lock()
	id := commitIdentity(t, session, fullIdentity(), []ID{groupIDs[0]})
	raw, _ := currentRaw(t, session)
	sealed := encodeBox(raw.records[0])

	pending, err := session.PrepareSetPinned(id, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	pending, err = session.PrepareSetGroups(id, []ID{groupIDs[1], groupIDs[0]})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	entry := listedEntry(t, session, id)
	if !entry.Pinned || !reflect.DeepEqual(entry.Groups, sortedIDs(groupIDs...)) || entry.Kind != KindIdentity {
		t.Fatalf("identity entry after pin and groups = %+v", entry)
	}
	next, _ := currentRaw(t, session)
	if !bytes.Equal(encodeBox(next.records[0]), sealed) {
		t.Fatal("pinning or grouping sealed the identity again")
	}
	pending, err = session.PrepareDelete(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if entries, err := session.List(); err != nil || len(entries) != 0 {
		t.Fatalf("entries after deleting the identity = %+v, error = %v", entries, err)
	}
}

func TestPrepareEditIdentityReplacesContentAndMembershipAndKeepsPin(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family", "Travel")
	session := created.Session
	defer session.Lock()
	credential := commitCredential(t, session, CredentialInput{Label: "Mail", Password: "secret"}, nil)
	id := commitIdentity(t, session, fullIdentity(), []ID{groupIDs[0]})
	pending, err := session.PrepareSetPinned(id, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	raw, _ := currentRaw(t, session)
	replacement := IdentityInput{Label: "Alex abroad", Phones: []string{"+44 20 7946 0000"}}
	pending, err = session.PrepareEditIdentity(id, replacement, []ID{groupIDs[1]})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	entry := listedEntry(t, session, id)
	if !entry.Pinned || !reflect.DeepEqual(entry.Groups, []ID{groupIDs[1]}) || entry.Label != "Alex abroad" || entry.Detail != "" || entry.ExpiresOn != "" {
		t.Fatalf("identity entry after the edit = %+v", entry)
	}
	if session.entries[1].revision != 2 {
		t.Fatalf("identity revision after one edit = %d", session.entries[1].revision)
	}
	if value := selectIdentity(t, session, id); !reflect.DeepEqual(value.IdentityInput, replacement) {
		t.Fatalf("identity after the edit = %#v", value.IdentityInput)
	}
	next, _ := currentRaw(t, session)
	if !bytes.Equal(encodeBox(next.records[0]), encodeBox(raw.records[0])) {
		t.Fatal("editing an identity sealed another record again")
	}
	if value := selectCredential(t, session, credential); value.Password != "secret" {
		t.Fatalf("credential after an identity edit = %+v", value.CredentialInput)
	}
}

func TestIdentityValuesNeverReachTheContainerInPlaintext(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	input := fullIdentity()
	commitIdentity(t, session, input, nil)
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	values := []string{input.Label, input.FullName, input.Birthday, input.Notes}
	values = append(values, input.Emails...)
	values = append(values, input.Phones...)
	for _, address := range input.Addresses {
		values = append(values, address.Street, address.City)
	}
	for _, document := range input.Documents {
		values = append(values, document.Number)
	}
	values = append(values, input.Documents[0].Issuer, input.Documents[0].IssuedOn, input.Documents[1].ExpiresOn)
	for _, value := range values {
		if bytes.Contains(container, []byte(value)) {
			t.Fatalf("%q appears in the vault file", value)
		}
	}
}

func TestListingDecryptsNoRecord(t *testing.T) {
	session, credential := populatedSession(t)
	defer session.Lock()
	identity := commitIdentity(t, session, fullIdentity(), nil)
	for i := range session.records {
		tampered := append([]byte(nil), session.records[i].ciphertext...)
		tampered[0] ^= 1
		session.records[i].ciphertext = tampered
	}
	if entry := listedEntry(t, session, identity); entry.Kind != KindIdentity || entry.Detail != "alex@example.test" {
		t.Fatalf("identity entry over unreadable records = %+v", entry)
	}
	if entry := listedEntry(t, session, credential); entry.Kind != KindCredential {
		t.Fatalf("credential entry over unreadable records = %+v", entry)
	}
	ticket, err := session.BeginSelection(identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadSelectedIdentity(ticket); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("reading an unreadable record: %v", err)
	}
}

func TestLateIdentityReadAfterLock(t *testing.T) {
	session, _ := populatedSession(t)
	id := commitIdentity(t, session, fullIdentity(), nil)
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	session.afterDecrypt = func() { close(started); <-release }
	result := make(chan error, 1)
	go func() {
		_, err := session.ReadSelectedIdentity(ticket)
		result <- err
	}()
	<-started
	session.Lock()
	close(release)
	if err := <-result; !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("late identity read returned %v", err)
	}
}

func TestLockedSessionRefusesIdentityOperations(t *testing.T) {
	session, _ := populatedSession(t)
	id := commitIdentity(t, session, fullIdentity(), nil)
	session.Lock()
	if _, _, err := session.PrepareCreateIdentity(fullIdentity(), nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareCreateIdentity: %v", err)
	}
	if _, err := session.PrepareEditIdentity(id, fullIdentity(), nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareEditIdentity: %v", err)
	}
	if _, err := session.ReadSelectedIdentity(Selection{ID: id}); !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("ReadSelectedIdentity: %v", err)
	}
}

func TestAddressIDsAreDrawnKeptAndChecked(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	given := homeAndWork()
	given.Addresses[0].ID = ID{1}
	if _, _, err := session.PrepareCreateIdentity(given, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("a new identity with an address id: %v", err)
	}
	owner := commitIdentity(t, session, homeAndWork(), nil)
	other := commitIdentity(t, session, homeAndWork(), nil)
	stored := selectIdentity(t, session, owner).IdentityInput
	home, work := stored.Addresses[0].ID, stored.Addresses[1].ID
	if home == (ID{}) || work == (ID{}) || home == work {
		t.Fatalf("drawn address ids = %s, %s", home, work)
	}
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for name, ids := range map[string][]ID{
		"an id the identity does not hold": {home, {0xee}},
		"another identity's id":            {home, selectIdentity(t, session, other).Addresses[0].ID},
		"an id given twice":                {home, home},
	} {
		input := withoutAddressIDs(stored)
		input.Addresses[0].ID, input.Addresses[1].ID = ids[0], ids[1]
		if _, err := session.PrepareEditIdentity(owner, input, nil); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("an edit with %s: %v", name, err)
		}
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused address id changed the vault")
	}
	edited := stored
	edited.Addresses = []Address{stored.Addresses[1], {Label: "New", City: "Capital City"}, stored.Addresses[0]}
	pending, err := session.PrepareEditIdentity(owner, edited, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	addresses := selectIdentity(t, session, owner).Addresses
	fresh := addresses[1].ID
	if addresses[0].ID != work || addresses[2].ID != home || fresh == (ID{}) || fresh == home || fresh == work {
		t.Fatalf("address ids after the edit = %s, %s, %s", addresses[0].ID, fresh, addresses[2].ID)
	}
}

// identityOfSchema is input's identity record led by schema, which must encode in one byte.
func identityOfSchema(t *testing.T, schema uint64, input IdentityInput) []byte {
	t.Helper()
	record := mustEncode(encodeIdentityRecord(input))
	if schema > 23 || record[1] != recordSchemaIdentity {
		t.Fatalf("schema %d cannot replace the record's one-byte schema", schema)
	}
	record[1] = byte(schema)
	return record
}

func TestIdentityRecordOfAnUnknownSchemaIsUnsupported(t *testing.T) {
	for _, schema := range []uint64{3, 6, 15} {
		t.Run(fmt.Sprint("schema ", schema), func(t *testing.T) {
			record := identityOfSchema(t, schema, homeAndWork())
			if err := decodeAsIdentity(record); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("decoded: got %v, want ErrUnsupported", err)
			}
			created, _ := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
			session := created.Session
			owner := commitIdentity(t, session, homeAndWork(), nil)
			deviceKey := bytes.Repeat([]byte{7}, 32)
			envelope := wrapFor(t, session, deviceKey)
			_, head := currentRaw(t, session)
			container := resealed(t, session, owner, record)
			session.Lock()
			if _, err := OpenWithRecovery(container, created.RecoveryPhrase); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("opened with the recovery phrase: got %v, want ErrUnsupported", err)
			}
			current := Witness{VaultID: head.VaultID, Revision: head.Revision, Hash: sha256.Sum256(container)}
			opened, _, err := OpenWithDevice(container, deviceKey, envelope, &current, nil)
			if err != nil {
				t.Fatalf("opened with the device at its witness: %v", err)
			}
			defer opened.Lock()
			if listed, err := opened.List(); err != nil || len(listed) != 2 {
				t.Fatalf("list = %d entries, error = %v", len(listed), err)
			}
			ticket, err := opened.BeginSelection(owner)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := opened.ReadSelectedIdentity(ticket); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("read the identity: got %v, want ErrUnsupported", err)
			}
			if err := opened.VerifyAll(); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("verified the vault: got %v, want ErrUnsupported", err)
			}
			if _, _, err := opened.Export(); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("exported the vault: got %v, want ErrUnsupported", err)
			}
			previous := Witness{VaultID: head.VaultID, Revision: head.Revision - 1, Hash: head.PreviousHash}
			if _, _, err := OpenWithDevice(container, deviceKey, envelope, &previous, func(Witness) error { return nil }); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("opened with the device one revision past its witness: got %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestIdentityRecordRefusesMissingOrRepeatedAddressIDs(t *testing.T) {
	encoded := func(ids ...ID) []byte {
		input := homeAndWork()
		for i := range ids {
			input.Addresses[i].ID = ids[i]
		}
		record, err := encodeIdentityRecord(input)
		if err != nil {
			t.Fatal(err)
		}
		return record
	}
	if _, err := decodeIdentityRecord(encoded(ID{1}, ID{2})); err != nil {
		t.Fatalf("a valid record was refused: %v", err)
	}
	for name, record := range map[string][]byte{"a zero id": encoded(ID{1}, ID{}), "a repeated id": encoded(ID{1}, ID{1})} {
		if _, err := decodeIdentityRecord(record); !errors.Is(err, ErrMalformed) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
