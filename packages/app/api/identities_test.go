package api

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

func testIdentityInput() IdentityInput {
	return IdentityInput{
		Label:    "Alex",
		Tags:     []string{},
		FullName: "Alex Example",
		Birthday: "1990-04-17",
		Emails:   []string{"alex@example.com", "alex@work.example"},
		Phones:   []string{"+1 555 0100"},
		Addresses: []Address{
			{Label: "Home", Street: "1 Example Street", City: "Springfield", Region: "Oregon", PostalCode: "97403", Country: "United States"},
			{Street: "  ", City: " Berlin ", Country: "Germany"},
		},
		Documents: []IdentityDocument{
			{Type: "passport", Number: "X1234567", Issuer: "Example Office", IssuedOn: "2020-01-15", ExpiresOn: "2030-01-14", Scans: []string{}},
			{Type: "drivers-license", Number: "DL-42", ExpiresOn: "2027-06-30", Scans: []string{}},
			{Type: "id-card", Number: "ID-99", Scans: []string{}},
			{Type: "tax-number", Number: "123-45-678", Scans: []string{}},
			{Type: "other", Label: "Library card", Number: "LIB-7", Scans: []string{}},
		},
		Notes: "kept at home",
	}
}

func assertFailure(t *testing.T, err error, code failure) {
	t.Helper()
	if err == nil || err.Error() != failurePrefix+string(code) {
		t.Fatalf("error = %v, want %s", err, code)
	}
}

func TestListingSplitsByKind(t *testing.T) {
	service := newReadyService(t)
	family := createTestGroup(t, service, "Family")
	credential, err := service.CreateCredential(CredentialInput{Label: "Mail", Login: "alice", Password: "secret"}, []string{family})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := service.CreateIdentity(testIdentityInput(), []string{family})
	if err != nil {
		t.Fatal(err)
	}
	bare, err := service.CreateIdentity(IdentityInput{Label: "Bare"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	credentials, err := service.ListCredentials()
	if err != nil || len(credentials) != 1 || credentials[0].ID != credential || credentials[0].Login != "alice" {
		t.Fatalf("credentials = %+v, error = %v", credentials, err)
	}
	identities, err := service.ListIdentities()
	if err != nil {
		t.Fatal(err)
	}
	want := []IdentitySummary{
		{ID: identity, Label: "Alex", Email: "alex@example.com", Groups: []string{family}, Tags: []string{}, ExpiresOn: "2027-06-30"},
		{ID: bare, Label: "Bare", Groups: []string{}, Tags: []string{}},
	}
	if !reflect.DeepEqual(identities, want) {
		t.Fatalf("identities = %+v", identities)
	}
}

func TestReadIdentityRoundTripsAndMarksItUsed(t *testing.T) {
	service := newReadyService(t)
	input := testIdentityInput()
	id, err := service.CreateIdentity(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := service.ReadIdentity(id)
	if err != nil {
		t.Fatal(err)
	}
	read := identity.IdentityInput
	read.Addresses = append([]Address(nil), read.Addresses...)
	for i := range read.Addresses {
		if len(read.Addresses[i].ID) != 32 {
			t.Fatalf("address %d read with id %q", i, read.Addresses[i].ID)
		}
		read.Addresses[i].ID = ""
	}
	if identity.ID != id || !reflect.DeepEqual(read, input) || identity.Groups == nil || len(identity.Groups) != 0 {
		t.Fatalf("identity = %+v", identity)
	}
	summaries, err := service.ListIdentities()
	if err != nil || len(summaries) != 1 || summaries[0].LastUsedAt <= 0 {
		t.Fatalf("summaries after a read = %+v, error = %v", summaries, err)
	}
	replacement := IdentityInput{Label: "Alex abroad", Phones: []string{"+44 20 7946 0000"}}
	if err := service.UpdateIdentity(id, replacement, nil); err != nil {
		t.Fatal(err)
	}
	identity, err = service.ReadIdentity(id)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	for _, list := range []string{`"emails":[]`, `"addresses":[]`, `"documents":[]`, `"groups":[]`} {
		if !strings.Contains(string(encoded), list) {
			t.Fatalf("%s missing from %s", list, encoded)
		}
	}
}

func TestIdentityAndCredentialCallsDoNotCross(t *testing.T) {
	service := newReadyService(t)
	credential, err := service.CreateCredential(CredentialInput{Label: "Mail", Password: "secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := service.CreateIdentity(testIdentityInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ReadCredential(identity)
	assertFailure(t, err, failureItemUnreadable)
	_, err = service.ReadIdentity(credential)
	assertFailure(t, err, failureItemUnreadable)
	assertFailure(t, service.UpdateCredential(identity, CredentialInput{Label: "Mail", Password: "secret"}, nil, nil), failureItemUnreadable)
	assertFailure(t, service.UpdateIdentity(credential, testIdentityInput(), nil), failureItemUnreadable)
	clipboard := attachPasteboard(service)
	schedule := func(time.Duration, func()) {}
	assertFailure(t, service.copyCredentialField(identity, "password", schedule), failureItemUnreadable)
	assertFailure(t, service.copyIdentityField(credential, IdentityField{Kind: "fullName"}, schedule), failureItemUnreadable)
	if clipboard.text != "" {
		t.Fatalf("a refused copy reached the clipboard: %q", clipboard.text)
	}
}

func TestIdentityAddressIDsTravelBothWays(t *testing.T) {
	service := newReadyService(t)
	id, err := service.CreateIdentity(testIdentityInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := service.ReadIdentity(id)
	if err != nil {
		t.Fatal(err)
	}
	kept := identity.Addresses[1].ID
	edited := identity.IdentityInput
	edited.Addresses = []Address{identity.Addresses[1], {City: "Oslo"}}
	if err := service.UpdateIdentity(id, edited, nil); err != nil {
		t.Fatal(err)
	}
	identity, err = service.ReadIdentity(id)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Addresses[0].ID != kept || identity.Addresses[1].ID == "" || identity.Addresses[1].ID == kept {
		t.Fatalf("addresses after the edit = %+v", identity.Addresses)
	}
	for name, address := range map[string]string{"not hex": "home", "an id the identity does not hold": strings.Repeat("e", 32)} {
		broken := identity.IdentityInput
		broken.Addresses = []Address{{ID: address, City: "Oslo"}}
		if err := service.UpdateIdentity(id, broken, nil); err == nil || err.Error() != failurePrefix+string(failureInvalidItem) {
			t.Fatalf("an edit with %s: %v", name, err)
		}
	}
	created := testIdentityInput()
	created.Addresses[0].ID = kept
	_, err = service.CreateIdentity(created, nil)
	assertFailure(t, err, failureInvalidItem)
}

func TestIdentityInputIsRefusedWithoutWriting(t *testing.T) {
	service := newReadyService(t)
	cases := map[string]IdentityInput{
		"unknown document type":             {Label: "Alex", Documents: []IdentityDocument{{Type: "visa", Number: "V-1"}}},
		"document type spelled differently": {Label: "Alex", Documents: []IdentityDocument{{Type: "Passport", Number: "P-1"}}},
		"impossible date":                   {Label: "Alex", Documents: []IdentityDocument{{Type: "passport", Number: "P-1", ExpiresOn: "2026-02-30"}}},
		"no label":                          {FullName: "Alex Example"},
		"typed document with a label":       {Label: "Alex", Documents: []IdentityDocument{{Type: "passport", Label: "Mine", Number: "P-1"}}},
		"other document without a label":    {Label: "Alex", Documents: []IdentityDocument{{Type: "other", Number: "O-1"}}},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := service.CreateIdentity(input, nil)
			assertFailure(t, err, failureInvalidItem)
		})
	}
	if identities, err := service.ListIdentities(); err != nil || len(identities) != 0 {
		t.Fatalf("identities after refused input = %+v, error = %v", identities, err)
	}
}

func TestCopyIdentityFieldCopiesEachReference(t *testing.T) {
	service := newReadyService(t)
	id, err := service.CreateIdentity(testIdentityInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	clipboard := attachPasteboard(service)
	var discard func()
	schedule := func(_ time.Duration, clear func()) { discard = clear }
	fullName := IdentityField{Kind: "fullName"}
	references := map[IdentityField]string{
		fullName:                                  "Alex Example",
		{Kind: "birthday"}:                        "1990-04-17",
		{Kind: "notes"}:                           "kept at home",
		{Kind: "email"}:                           "alex@example.com",
		{Kind: "email", Index: 1}:                 "alex@work.example",
		{Kind: "phone"}:                           "+1 555 0100",
		{Kind: "address"}:                         "1 Example Street\n97403 Springfield\nOregon\nUnited States",
		{Kind: "address", Index: 1}:               "Berlin\nGermany",
		{Kind: "document"}:                        "X1234567",
		{Kind: "document", Index: 4}:              "LIB-7",
		{Kind: "address", Part: "street"}:         "1 Example Street",
		{Kind: "address", Part: "city"}:           "Springfield",
		{Kind: "address", Part: "region"}:         "Oregon",
		{Kind: "address", Part: "postalCode"}:     "97403",
		{Kind: "address", Part: "country"}:        "United States",
		{Kind: "address", Index: 1, Part: "city"}: "Berlin",
	}
	for reference, want := range references {
		if err := service.copyIdentityField(id, reference, schedule); err != nil {
			t.Fatalf("copy of %+v failed: %v", reference, err)
		}
		if clipboard.text != want {
			t.Fatalf("clipboard after copying %+v = %q", reference, clipboard.text)
		}
	}
	if err := service.copyIdentityField(id, fullName, schedule); err != nil {
		t.Fatal(err)
	}
	refused := []IdentityField{
		{}, {Kind: "fullname"}, {Kind: "label"}, {Kind: "Birthday"}, {Kind: "photo"}, {Kind: "Email"},
		{Kind: "fullName", Index: 1}, {Kind: "fullName", Part: "city"}, {Kind: "birthday", Index: 1}, {Kind: "notes", Part: "city"},
		{Kind: "email", Index: -1}, {Kind: "email", Index: 2}, {Kind: "phone", Index: 1},
		{Kind: "address", Index: 2}, {Kind: "document", Index: 5},
		{Kind: "address", Part: "label"}, {Kind: "address", Part: "Street"}, {Kind: "address", Part: "postalcode"},
		{Kind: "address", Index: -1, Part: "city"}, {Kind: "address", Index: 2, Part: "city"},
		{Kind: "phone", Part: "city"}, {Kind: "document", Part: "number"},
	}
	for _, reference := range refused {
		assertFailure(t, service.copyIdentityField(id, reference, schedule), failureFieldNotCopyable)
	}
	if clipboard.text != "Alex Example" {
		t.Fatalf("a refused reference changed the clipboard to %q", clipboard.text)
	}
	for _, part := range []string{"street", "region", "postalCode"} {
		assertFailure(t, service.copyIdentityField(id, IdentityField{Kind: "address", Index: 1, Part: part}, schedule), failureFieldEmpty)
	}
	discard()
	if clipboard.text != "" {
		t.Fatal("the copied value stayed on the clipboard after its lifetime")
	}
	bare, err := service.CreateIdentity(IdentityInput{Label: "Bare"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"fullName", "birthday", "notes"} {
		assertFailure(t, service.copyIdentityField(bare, IdentityField{Kind: kind}, schedule), failureFieldEmpty)
	}
	assertFailure(t, service.copyIdentityField("not an identity", fullName, schedule), failureItemUnreadable)
	if err := service.Lock(); err != nil {
		t.Fatal(err)
	}
	if err := service.copyIdentityField(id, fullName, schedule); err == nil {
		t.Fatal("a locked vault served an identity field")
	}
}

func TestFormatAddressSkipsEmptyParts(t *testing.T) {
	cases := []struct {
		name    string
		address vault.Address
		want    string
	}{
		{"every part", vault.Address{Label: "Home", Street: "1 Example Street", City: "Springfield", Region: "Oregon", PostalCode: "97403", Country: "United States"}, "1 Example Street\n97403 Springfield\nOregon\nUnited States"},
		{"city without a postal code", vault.Address{Street: "Main 5", City: "Riga"}, "Main 5\nRiga"},
		{"postal code without a city", vault.Address{PostalCode: "10115", Country: "Germany"}, "10115\nGermany"},
		{"country only", vault.Address{Country: "Japan"}, "Japan"},
		{"space around parts", vault.Address{Street: "  Main 5 ", PostalCode: " 10115", City: "Berlin  ", Region: " "}, "Main 5\n10115 Berlin"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := formatAddress(testCase.address); got != testCase.want {
				t.Fatalf("formatAddress = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestItemCallsActOnBothKinds(t *testing.T) {
	service := newReadyService(t)
	family := createTestGroup(t, service, "Family")
	credential, err := service.CreateCredential(CredentialInput{Label: "Mail", Password: "secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := service.CreateIdentity(testIdentityInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{credential, identity} {
		if err := service.SetItemGroups(id, []string{family}); err != nil {
			t.Fatal(err)
		}
		if err := service.SetPinned(id, true); err != nil {
			t.Fatal(err)
		}
	}
	credentials, err := service.ListCredentials()
	if err != nil || len(credentials) != 1 || !credentials[0].Pinned || !reflect.DeepEqual(credentials[0].Groups, []string{family}) {
		t.Fatalf("credentials = %+v, error = %v", credentials, err)
	}
	identities, err := service.ListIdentities()
	if err != nil || len(identities) != 1 || !identities[0].Pinned || !reflect.DeepEqual(identities[0].Groups, []string{family}) {
		t.Fatalf("identities = %+v, error = %v", identities, err)
	}
	for _, id := range []string{credential, identity} {
		if err := service.DeleteItem(id); err != nil {
			t.Fatal(err)
		}
	}
	credentials, _ = service.ListCredentials()
	identities, _ = service.ListIdentities()
	if len(credentials) != 0 || len(identities) != 0 {
		t.Fatalf("items after deleting both: %+v, %+v", credentials, identities)
	}
	assertFailure(t, service.DeleteItem("not an item"), failureItemUnreadable)
	assertFailure(t, service.DeleteItem(identity), failureItemUnreadable)
}

func TestIdentityLimitsMatchTheVault(t *testing.T) {
	limits, err := (&Service{}).GetIdentityLimits()
	if err != nil {
		t.Fatal(err)
	}
	if limits.Label != vault.MaxLabelLength || limits.Documents != vault.MaxIdentityDocuments || limits.AddressLabel != vault.MaxPartLabelLength || limits.DocumentLabel != vault.MaxPartLabelLength {
		t.Fatalf("limits = %+v", limits)
	}
}
