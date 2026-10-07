package bitwarden

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dortanes/ravenpass/packages/importers"
	"github.com/dortanes/ravenpass/packages/vault"
)

func readItems(t *testing.T, items ...string) importers.Export {
	t.Helper()
	return readContent(t, jsonExportOf(items...))
}

func onlyItem(t *testing.T, items ...string) importers.Item {
	t.Helper()
	export := readItems(t, items...)
	if len(export.Items) != 1 || len(export.Skipped) != 0 {
		t.Fatalf("read %+v", export)
	}
	return export.Items[0]
}

func wantItem(t *testing.T, got importers.Item, want importers.Item) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("read\n%s\nwant\n%s", describe(got), describe(want))
	}
}

func describe(item importers.Item) string {
	content := item.Content
	var value any
	switch {
	case content.Credential != nil:
		value = *content.Credential
	case content.Card != nil:
		value = *content.Card
	case content.Identity != nil:
		value = *content.Identity
	case content.Note != nil:
		value = *content.Note
	}
	return fmt.Sprintf("%#v pinned=%v origin=%q folders=%q", value, content.Pinned, item.Origin, item.Folders)
}

func TestLoginKeepsWhatRavenpassHasNoFieldForInItsNotes(t *testing.T) {
	item := onlyItem(t, `{
		"type": 1, "name": "Example", "notes": "Own notes", "favorite": true, "reprompt": 1,
		"fields": [
			{"name": "Recovery", "value": "ABCD-EFGH", "type": 0},
			{"name": "Linked", "value": null, "type": 3, "linkedId": 100}
		],
		"login": {
			"uris": [{"uri": "https://example.test/login", "match": null}, {"uri": " ", "match": 0}, {"uri": "https://second.example.test", "match": 3}],
			"username": "alex@example.test", "password": "  secret  ", "totp": "steam://ABCDEF",
			"fido2Credentials": [{"credentialId": "one"}, {"credentialId": "two"}]
		},
		"passwordHistory": [{"password": "old", "lastUsedDate": "2024-01-01T00:00:00Z"}]
	}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Pinned: true, Credential: &vault.CredentialInput{
			Label:    "Example",
			Websites: []string{"example.test", "second.example.test"},
			Email:    "alex@example.test",
			Password: "  secret  ",
			Notes:    "Own notes\n\nOne-time code setup: steam://ABCDEF\nRecovery: ABCD-EFGH",
		}},
		Origin: importers.OriginLogin,
	})
}

func TestLoginKeepsItsWebsitesAndLogin(t *testing.T) {
	setup := "otpauth://totp/Example:alex?secret=JBSWY3DPEHPK3PXP&issuer=Example&digits=8"
	normalized, err := vault.NormalizeTOTP(setup)
	if err != nil {
		t.Fatal(err)
	}
	item := onlyItem(t, `{"type": 1, "name": "App", "login": {
		"uris": [{"uri": "androidapp://com.example"}, {"uri": "https://example.test"}],
		"username": "Alex <alex@example.test>", "password": "pw", "totp": "`+setup+`"
	}}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Credential: &vault.CredentialInput{
			Label:    "App",
			Websites: []string{"androidapp://com.example", "example.test"},
			Login:    "Alex <alex@example.test>",
			Password: "pw",
			TOTP:     normalized,
		}},
		Origin: importers.OriginLogin,
	})
	if site := item.Content.Credential.Site(); site != "example.test" {
		t.Fatalf("site = %q", site)
	}

	appOnly := onlyItem(t, `{"type": 1, "name": "App", "login": {"uris": [{"uri": "androidapp://com.example"}], "username": "alex", "totp": "JBSWY3DPEHPK3PXP"}}`)
	if credential := appOnly.Content.Credential; !reflect.DeepEqual(credential.Websites, []string{"androidapp://com.example"}) || credential.Login != "alex" || credential.TOTP != "JBSWY3DPEHPK3PXP" || credential.Notes != "" {
		t.Fatalf("read %+v", credential)
	}

	long := "https://login.example.test/sign-in?return=" + strings.Repeat("a", vault.MaxOriginLength)
	longSite := onlyItem(t, `{"type": 1, "name": "Long", "login": {"uris": [{"uri": "`+long+`"}]}}`)
	if credential := longSite.Content.Credential; !reflect.DeepEqual(credential.Websites, []string{"login.example.test"}) || credential.Notes != "" {
		t.Fatalf("read %+v", credential)
	}

	uris := make([]string, vault.MaxCredentialWebsites+2)
	for i := range uris {
		uris[i] = fmt.Sprintf(`{"uri": "https://site%d.example"}`, i)
	}
	many := onlyItem(t, `{"type": 1, "name": "Many", "login": {"uris": [`+strings.Join(uris, ",")+`]}}`).Content.Credential
	extra := fmt.Sprintf("Website: https://site%d.example\nWebsite: https://site%d.example", vault.MaxCredentialWebsites, vault.MaxCredentialWebsites+1)
	if len(many.Websites) != vault.MaxCredentialWebsites || many.Notes != extra {
		t.Fatalf("read %d websites and notes %q", len(many.Websites), many.Notes)
	}
}

func TestLoginWithoutANameTakesItsSiteThenItsUsername(t *testing.T) {
	export := readItems(t,
		`{"type": 1, "name": "", "login": {"uris": [{"uri": "https://www.fallback.example/sign-in"}], "username": "alex"}}`,
		`{"type": 1, "name": "  ", "login": {"uris": [{"uri": "androidapp://com.example"}], "username": " alex "}}`,
		`{"type": 1, "name": null, "login": {"uris": null, "username": null, "password": "orphan"}}`,
		`{"type": 2, "name": " ", "notes": "no name", "secureNote": {"type": 0}}`,
		`{"type": 3, "name": "", "card": {"number": "4111111111111111"}}`,
	)
	labels := []string{export.Items[0].Content.Credential.Label, export.Items[1].Content.Credential.Label}
	if want := []string{"fallback.example", "alex"}; !reflect.DeepEqual(labels, want) {
		t.Fatalf("labels = %q, want %q", labels, want)
	}
	want := []importers.Skip{
		{Origin: importers.OriginLogin, Reason: importers.ReasonUnnamed},
		{Origin: importers.OriginNote, Reason: importers.ReasonUnnamed},
		{Origin: importers.OriginCard, Reason: importers.ReasonUnnamed},
	}
	if len(export.Items) != 2 || !reflect.DeepEqual(export.Skipped, want) {
		t.Fatalf("read %d items, skipped %+v", len(export.Items), export.Skipped)
	}
}

func TestSecureNoteWritesItsFieldsAfterItsBody(t *testing.T) {
	item := onlyItem(t, `{"type": 2, "name": " Router ", "notes": "admin panel\n", "reprompt": 1, "secureNote": {"type": 0}, "fields": [
		{"name": "Enabled", "value": "true", "type": 2},
		{"name": "Disabled", "value": null, "type": 2},
		{"name": "", "value": "a value with no name", "type": 0},
		{"name": "Empty", "value": "", "type": 1},
		{"name": "Hidden", "value": "one\ntwo", "type": 1}
	]}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Note: &vault.NoteInput{
			Label:  "Router",
			Body:   "admin panel\n\nEnabled: true\nDisabled: false\na value with no name\nHidden:\none\ntwo",
			Hidden: true,
		}},
		Origin: importers.OriginNote,
	})
	if plain := onlyItem(t, `{"type": 2, "name": "Plain", "notes": null}`); !reflect.DeepEqual(*plain.Content.Note, vault.NoteInput{Label: "Plain"}) {
		t.Fatalf("read %+v", *plain.Content.Note)
	}
}

func TestSecureNoteHoldingASeedPhraseBecomesASeed(t *testing.T) {
	phrase := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	item := onlyItem(t, `{"type": 2, "name": "Anytype", "notes": " `+phrase+`\n", "reprompt": 1, "fields": [
		{"name": "Wallet", "value": "Anytype account", "type": 0}
	]}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Seed: &vault.SeedInput{
			Label:  "Anytype",
			Format: vault.SeedPhrase,
			Words:  strings.Fields(phrase),
			Notes:  "Wallet: Anytype account",
		}},
		Origin: importers.OriginNote,
	})
	prose := onlyItem(t, `{"type": 2, "name": "Anytype", "notes": "My phrase: `+phrase+`"}`)
	if prose.Content.Note == nil {
		t.Fatalf("a note that holds more than a phrase read %+v", prose.Content)
	}
}

func TestCardMapsOntoACard(t *testing.T) {
	item := onlyItem(t, `{"type": 3, "name": "Visa", "notes": "Own notes", "favorite": true, "card": {
		"cardholderName": "Alex Doe", "brand": "Visa", "number": "4111 1111-1111 1111", "expMonth": "1", "expYear": "27", "code": " 123 "
	}, "fields": [{"name": "Bank", "value": "Example Bank", "type": 0}]}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Pinned: true, Card: &vault.CardInput{
			Label:        "Visa",
			Holder:       "Alex Doe",
			Number:       "4111111111111111",
			Expiry:       "2027-01",
			SecurityCode: "123",
			Network:      vault.NetworkVisa,
			Notes:        "Own notes\n\nBank: Example Bank",
		}},
		Origin: importers.OriginCard,
	})
}

func TestCardTakesItsAddressFromAField(t *testing.T) {
	item := onlyItem(t, `{"type": 3, "name": "Visa", "notes": "Own notes", "card": {"number": "4111111111111111"}, "fields": [
		{"name": "Address", "value": "true", "type": 2},
		{"name": " address ", "value": " Auezovsky District, microdistrict 6, 28 ", "type": 0},
		{"name": "Billing address", "value": "Second", "type": 1},
		{"name": "Bank", "value": "Example Bank", "type": 0}
	]}`)
	card := item.Content.Card
	if card.Billing == nil || *card.Billing != (vault.Address{Street: "Auezovsky District, microdistrict 6, 28"}) {
		t.Fatalf("billing = %+v", card.Billing)
	}
	if want := "Own notes\n\nAddress: true\nBilling address: Second\nBank: Example Bank"; card.Notes != want {
		t.Fatalf("notes = %q, want %q", card.Notes, want)
	}
	for _, name := range []string{"Адрес", "Платёжный адрес", "BILLING ADDRESS"} {
		named := onlyItem(t, `{"type": 3, "name": "Card", "card": {"number": "4111111111111111"}, "fields": [{"name": "`+name+`", "value": "Main St 1", "type": 0}]}`).Content.Card
		if named.Billing == nil || named.Billing.Street != "Main St 1" || named.Notes != "" {
			t.Fatalf("%q read %+v", name, named)
		}
	}
	long := strings.Repeat("s", vault.MaxStreetLength+1)
	tooLong := onlyItem(t, `{"type": 3, "name": "Card", "card": {"number": "4111111111111111"}, "fields": [{"name": "Address", "value": "`+long+`", "type": 0}]}`).Content.Card
	if tooLong.Billing != nil || tooLong.Notes != "Address: "+long {
		t.Fatalf("a street over the bound read %+v", tooLong.Billing)
	}
}

func TestCardExpiryAndSecurityCode(t *testing.T) {
	tests := []struct {
		month, year, code string
		expiry            string
		securityCode      string
		notes             string
	}{
		{"1", "27", "123", "2027-01", "123", ""},
		{"01", "2027", "1234", "2027-01", "1234", ""},
		{"12", "2030", "", "2030-12", "", ""},
		{"", "", "", "", "", ""},
		{"13", "2027", "123", "", "123", "Expiration date: 13/2027"},
		{"0", "2027", "123", "", "123", "Expiration date: 0/2027"},
		{"1", "7", "123", "", "123", "Expiration date: 1/7"},
		{"1", "", "123", "", "123", "Expiration date: 1"},
		{"1", "1850", "123", "", "123", "Expiration date: 1/1850"},
		{"Jan", "2027", "123", "", "123", "Expiration date: Jan/2027"},
		{"1", "2027", "12", "2027-01", "", "Security code: 12"},
		{"1", "2027", "12345", "2027-01", "", "Security code: 12345"},
		{"1", "2027", "12a", "2027-01", "", "Security code: 12a"},
	}
	for _, test := range tests {
		t.Run(test.month+"/"+test.year+" "+test.code, func(t *testing.T) {
			card := onlyItem(t, `{"type": 3, "name": "Card", "card": {"number": "4111111111111111", "expMonth": "`+test.month+`", "expYear": "`+test.year+`", "code": "`+test.code+`", "brand": "Visa"}}`).Content.Card
			if card == nil || card.Expiry != test.expiry || card.SecurityCode != test.securityCode || card.Notes != test.notes {
				t.Fatalf("read %+v", card)
			}
		})
	}
}

func TestCardBrandMapsOntoANetwork(t *testing.T) {
	tests := []struct {
		brand   string
		network vault.CardNetwork
		notes   string
	}{
		{"Visa", vault.NetworkVisa, ""},
		{"Mastercard", vault.NetworkMastercard, ""},
		{"Amex", vault.NetworkAmericanExpress, ""},
		{"Discover", vault.NetworkDiscover, ""},
		{"Diners Club", vault.NetworkDinersClub, ""},
		{"JCB", vault.NetworkJCB, ""},
		{"Maestro", vault.NetworkMaestro, ""},
		{"UnionPay", vault.NetworkUnionPay, ""},
		{"Mir", vault.NetworkMir, ""},
		{"Elo", vault.NetworkElo, ""},
		{"Hipercard", vault.NetworkHipercard, ""},
		{"RuPay", 0, "Brand: RuPay"},
		{"Other", 0, "Brand: Other"},
		{"", 0, ""},
	}
	for _, test := range tests {
		t.Run(test.brand, func(t *testing.T) {
			card := onlyItem(t, `{"type": 3, "name": "Card", "card": {"number": "4111111111111111", "brand": "`+test.brand+`"}}`).Content.Card
			if card == nil || card.Network != test.network || card.Notes != test.notes {
				t.Fatalf("read %+v", card)
			}
		})
	}
}

func TestCardWithoutACardNumberBecomesANote(t *testing.T) {
	item := onlyItem(t, `{"type": 3, "name": "Gift card", "notes": "Own notes", "card": {
		"cardholderName": "Alex Doe", "brand": "Visa", "number": "1234-5678", "expMonth": "13", "expYear": "2027", "code": "12"
	}, "fields": [{"name": "Balance", "value": "50", "type": 0}]}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Note: &vault.NoteInput{
			Label:  "Gift card",
			Body:   "Own notes\n\nCardholder: Alex Doe\nCard number: 1234-5678\nExpiration date: 13/2027\nSecurity code: 12\nBrand: Visa\nBalance: 50",
			Hidden: true,
		}},
		Origin: importers.OriginCard,
	})
	if empty := onlyItem(t, `{"type": 3, "name": "No number", "card": null}`); !reflect.DeepEqual(*empty.Content.Note, vault.NoteInput{Label: "No number", Hidden: true}) {
		t.Fatalf("read %+v", empty.Content)
	}
}

func TestIdentityMapsItsDocumentsAndAddress(t *testing.T) {
	item := onlyItem(t, `{"type": 4, "name": "Me", "notes": "Own notes", "identity": {
		"title": "Dr", "firstName": "Alex", "middleName": " ", "lastName": "Doe",
		"address1": "1 Main St", "address2": "Apt 2", "address3": null,
		"city": "Springfield", "state": "IL", "postalCode": "62701", "country": "US",
		"company": "Acme", "email": "alex@example.test", "phone": "+1 555 0100",
		"ssn": "123-45-6789", "username": "alexd", "passportNumber": "P123", "licenseNumber": "L456"
	}}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Identity: &vault.IdentityInput{
			Label:     "Me",
			FullName:  "Alex Doe",
			Emails:    []string{"alex@example.test"},
			Phones:    []string{"+1 555 0100"},
			Addresses: []vault.Address{{Street: "1 Main St\nApt 2", City: "Springfield", Region: "IL", PostalCode: "62701", Country: "US"}},
			Documents: []vault.Document{
				{Type: vault.DocumentPassport, Number: "P123"},
				{Type: vault.DocumentDriversLicense, Number: "L456"},
				{Type: vault.DocumentTaxNumber, Number: "123-45-6789"},
			},
			Notes: "Own notes\n\nTitle: Dr\nCompany: Acme\nUsername: alexd",
		}},
		Origin: importers.OriginIdentity,
	})
	bare := onlyItem(t, `{"type": 4, "name": "Bare", "identity": {"city": "Springfield", "email": " "}}`).Content.Identity
	if want := (vault.IdentityInput{Label: "Bare", Addresses: []vault.Address{{City: "Springfield"}}}); !reflect.DeepEqual(*bare, want) {
		t.Fatalf("read %+v", *bare)
	}
}

func TestDriversLicenseBecomesAnIdentityWithItsDocument(t *testing.T) {
	item := onlyItem(t, `{"type": 7, "name": "Licence", "driversLicense": {
		"firstName": "Alex", "middleName": "J", "lastName": "Doe", "dateOfBirth": "1990-05-17",
		"licenseNumber": "D1234", "issuingCountry": "US", "issuingState": "CA",
		"issueDate": "2020-01-15T23:30:00.000+05:00", "expirationDate": "2028-01-15",
		"issuingAuthority": null, "licenseClass": "C"
	}}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Identity: &vault.IdentityInput{
			Label:     "Licence",
			FullName:  "Alex J Doe",
			Birthday:  "1990-05-17",
			Documents: []vault.Document{{Type: vault.DocumentDriversLicense, Number: "D1234", Issuer: "CA, US", IssuedOn: "2020-01-15", ExpiresOn: "2028-01-15"}},
			Notes:     "License class: C",
		}},
		Origin: importers.OriginDriversLicense,
	})
	withAuthority := onlyItem(t, `{"type": 7, "name": "Licence", "driversLicense": {
		"licenseNumber": "D1234", "issuingAuthority": "DMV", "issuingState": "CA", "issuingCountry": "US"
	}}`)
	if issuer := withAuthority.Content.Identity.Documents[0].Issuer; issuer != "DMV, CA, US" {
		t.Fatalf("issuer = %q", issuer)
	}
}

func TestPassportBecomesAnIdentityWithItsDocument(t *testing.T) {
	item := onlyItem(t, `{"type": 8, "name": "Passport", "notes": "Own notes", "passport": {
		"surname": "Doe", "givenName": "Alex", "dateOfBirth": "17/05/1990", "sex": "M",
		"birthPlace": "Springfield", "nationality": "US", "issuingCountry": "US",
		"passportNumber": "X123", "passportType": "P", "nationalIdentificationNumber": "N-1",
		"issuingAuthority": "State Department", "issueDate": "2030-01-01", "expirationDate": "2020-01-01"
	}}`)
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Identity: &vault.IdentityInput{
			Label:     "Passport",
			FullName:  "Alex Doe",
			Documents: []vault.Document{{Type: vault.DocumentPassport, Number: "X123", Issuer: "State Department, US"}},
			Notes:     "Own notes\n\nDate of birth: 17/05/1990\nIssued on: 2030-01-01\nExpires on: 2020-01-01\nSex: M\nPlace of birth: Springfield\nNationality: US\nPassport type: P\nNational ID number: N-1",
		}},
		Origin: importers.OriginPassport,
	})
	unnumbered := onlyItem(t, `{"type": 8, "name": "Draft", "passport": {"givenName": "Alex", "issuingCountry": "US", "issueDate": "2020-01-01", "expirationDate": "2030-01-01"}}`)
	if want := (vault.IdentityInput{Label: "Draft", FullName: "Alex", Notes: "Issued by: US\nIssued on: 2020-01-01\nExpires on: 2030-01-01"}); !reflect.DeepEqual(*unnumbered.Content.Identity, want) {
		t.Fatalf("read %+v", *unnumbered.Content.Identity)
	}
}

func TestSSHKeyAndBankAccountBecomeNotes(t *testing.T) {
	export := readItems(t,
		`{"type": 5, "name": "Server key", "sshKey": {
			"privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXk=\n-----END OPENSSH PRIVATE KEY-----",
			"publicKey": "ssh-ed25519 AAAAC3Nza alex@example", "keyFingerprint": "SHA256:abc"
		}}`,
		`{"type": 6, "name": "Checking", "notes": "Own notes", "bankAccount": {
			"bankName": "Example Bank", "nameOnAccount": "Alex Doe", "accountType": "Checking",
			"accountNumber": "000123", "routingNumber": "110000000", "branchNumber": "42", "pin": "1234",
			"swiftCode": "EXAMPLEX", "iban": "DE89370400440532013000", "bankContactPhone": "+1 555 0199"
		}, "fields": [{"name": "Online ID", "value": "alexd", "type": 0}]}`,
	)
	want := []importers.Item{
		{Content: vault.NewItem{Note: &vault.NoteInput{
			Label:  "Server key",
			Body:   "Private key:\n-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXk=\n-----END OPENSSH PRIVATE KEY-----\nPublic key: ssh-ed25519 AAAAC3Nza alex@example\nFingerprint: SHA256:abc",
			Hidden: true,
		}}, Origin: importers.OriginSSHKey},
		{Content: vault.NewItem{Note: &vault.NoteInput{
			Label:  "Checking",
			Body:   "Own notes\n\nBank: Example Bank\nAccount holder: Alex Doe\nAccount type: Checking\nAccount number: 000123\nRouting number: 110000000\nBranch number: 42\nPIN: 1234\nSWIFT: EXAMPLEX\nIBAN: DE89370400440532013000\nBank phone: +1 555 0199\nOnline ID: alexd",
			Hidden: true,
		}}, Origin: importers.OriginBankAccount},
	}
	for i := range want {
		wantItem(t, export.Items[i], want[i])
	}
}

func TestReadFilesItemsUnderFoldersAndCollections(t *testing.T) {
	export := readContent(t, []byte(`{"encrypted": false,
		"folders": [{"id": "f1", "name": "Work"}, {"id": "f2", "name": "  "}, {"id": "", "name": "Nameless"}],
		"collections": [{"id": "c1", "name": "Shared"}, {"id": "c2", "name": "Team"}],
		"items": [
			{"type": 2, "name": "Filed", "folderId": "f1", "collectionIds": ["c2", "c1", "missing"]},
			{"type": 2, "name": "Loose", "folderId": null, "collectionIds": null},
			{"type": 2, "name": "Blank folder", "folderId": "f2"}
		]}`))
	folders := make([][]string, len(export.Items))
	for i, item := range export.Items {
		folders[i] = item.Folders
	}
	if want := [][]string{{"Work", "Team", "Shared"}, nil, nil}; !reflect.DeepEqual(folders, want) {
		t.Fatalf("folders = %q, want %q", folders, want)
	}
}

func TestReadSkipsAnUnknownTypeAndCountsPasskeys(t *testing.T) {
	export := readItems(t,
		`{"type": 99, "name": " Future ", "future": {"value": 1}}`,
		`{"type": null, "name": "Typeless"}`,
		`{"type": 1, "name": "", "login": {"fido2Credentials": [{"credentialId": "one"}]}}`,
		`{"type": 1, "name": "Two passkeys", "login": {"fido2Credentials": [{"credentialId": "one"}, {"credentialId": "two"}]}}`,
	)
	want := []importers.Skip{
		{Label: "Future", Reason: importers.ReasonUnsupported},
		{Label: "Typeless", Reason: importers.ReasonUnsupported},
		{Origin: importers.OriginLogin, Reason: importers.ReasonUnnamed},
	}
	if !reflect.DeepEqual(export.Skipped, want) || export.Passkeys != 3 || len(export.Items) != 1 {
		t.Fatalf("read %+v", export)
	}
}

func TestReadWritesLabelsInThePersonsLanguage(t *testing.T) {
	opened := openContent(t, jsonExportOf(`{"type": 1, "name": "Mail", "login": {"totp": "steam://ABCDEF"}}`))
	russian := english
	russian.OneTimeCode = "Настройка одноразовых кодов"
	export, err := opened.Read(russian)
	if err != nil {
		t.Fatal(err)
	}
	if notes := export.Items[0].Content.Credential.Notes; notes != "Настройка одноразовых кодов: steam://ABCDEF" {
		t.Fatalf("notes = %q", notes)
	}
}

func TestReadRefusesAValueOfTheWrongType(t *testing.T) {
	for _, items := range []string{`{"type": "1", "name": "Quoted type"}`, `{"type": 3, "name": "Card", "card": {"expMonth": 1}}`, `{"type": 1, "name": "Login", "login": {"fido2Credentials": "none"}}`} {
		if _, err := openContent(t, jsonExportOf(items)).Read(english); !errors.Is(err, importers.ErrUnrecognized) {
			t.Fatalf("Read of %s = %v", items, err)
		}
	}
}
