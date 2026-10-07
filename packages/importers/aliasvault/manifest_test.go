package aliasvault

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dortanes/ravenpass/packages/importers"
	"github.com/dortanes/ravenpass/packages/vault"
)

const loginItem = `{"id": "l1", "name": "Example", "itemType": "Login", "folderId": null, "logoId": null,
	"fieldValues": [
		{"id": "v1", "fieldKey": "login.url", "fieldDefinitionId": null, "value": "https://second.example.test", "weight": 1},
		{"id": "v2", "fieldKey": "login.username", "fieldDefinitionId": null, "value": "alex", "weight": 0},
		{"id": "v3", "fieldKey": "login.password", "fieldDefinitionId": null, "value": "  secret  ", "weight": 0},
		{"id": "v4", "fieldKey": "login.email", "fieldDefinitionId": null, "value": "alex@example.test", "weight": 0},
		{"id": "v5", "fieldKey": "login.url", "fieldDefinitionId": null, "value": " https://example.test/login ", "weight": 0},
		{"id": "v6", "fieldKey": "notes.content", "fieldDefinitionId": null, "value": "Own notes", "weight": 0}
	],
	"attachments": [], "totpCodes": [], "passkeys": []}`

func readItems(t *testing.T, items ...string) importers.Export {
	t.Helper()
	return readContent(t, avuxOf(t, manifestOf(items...)))
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
	case content.Note != nil:
		value = *content.Note
	case content.Seed != nil:
		value = *content.Seed
	}
	return fmt.Sprintf("%#v origin=%q folders=%q", value, item.Origin, item.Folders)
}

// itemOf writes an item of kind named name with values under their field keys, each of weight 0.
func itemOf(kind, name string, values ...string) string {
	written := make([]string, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		written = append(written, fmt.Sprintf(`{"fieldKey": %q, "value": %q, "weight": 0}`, values[i], values[i+1]))
	}
	return fmt.Sprintf(`{"id": "i-%s", "name": %q, "itemType": %q, "fieldValues": [%s], "totpCodes": [], "passkeys": []}`, name, name, kind, strings.Join(written, ","))
}

func normalized(t *testing.T, setup string) string {
	t.Helper()
	canonical, err := vault.NormalizeTOTP(setup)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func TestLoginMapsOntoACredential(t *testing.T) {
	wantItem(t, onlyItem(t, loginItem), importers.Item{
		Content: vault.NewItem{Credential: &vault.CredentialInput{
			Label:    "Example",
			Websites: []string{"example.test", "second.example.test"},
			Login:    "alex",
			Email:    "alex@example.test",
			Password: "  secret  ",
			Notes:    "Own notes",
		}},
		Origin: importers.OriginLogin,
	})
}

func TestLoginKeepsWhatRavenpassHasNoFieldForInItsNotes(t *testing.T) {
	long := "https://login.example.test/sign-in?return=" + strings.Repeat("a", vault.MaxOriginLength)
	manifest := `{"version": "1.0.0",
		"fieldDefinitions": [
			{"id": "d1", "fieldType": "Text", "label": "Recovery", "weight": 2},
			{"id": "d2", "fieldType": "Hidden", "label": "PIN hint", "weight": 1}
		],
		"items": [{"id": "l1", "name": "Example", "itemType": "Login", "fieldValues": [
			{"fieldKey": null, "fieldDefinitionId": "d1", "value": "second recovery", "weight": 1},
			{"fieldKey": "login.url", "fieldDefinitionId": null, "value": "` + long + `", "weight": 0},
			{"fieldKey": "login.otp_hint", "fieldDefinitionId": null, "value": "legacy", "weight": 0},
			{"fieldKey": null, "fieldDefinitionId": "d9", "value": "orphan", "weight": 0},
			{"fieldKey": null, "fieldDefinitionId": "d1", "value": "first recovery", "weight": 0},
			{"fieldKey": null, "fieldDefinitionId": "d2", "value": "hint", "weight": 3},
			{"fieldKey": null, "fieldDefinitionId": "d1", "value": " ", "weight": 2}
		], "totpCodes": [
			{"name": "Steam", "secretKey": "steam://ABCDEF"},
			{"name": "Blank", "secretKey": " "},
			{"name": " Work ", "secretKey": "JBSWY3DPEHPK3PXP"},
			{"name": null, "secretKey": "otpauth://totp/Example:alex?secret=GEZDGNBVGY3TQOJQ&issuer=Example"}
		], "passkeys": []}]}`
	credential := readContent(t, avuxOf(t, manifest)).Items[0].Content.Credential
	want := "One-time code setup (Steam): steam://ABCDEF\n" +
		"One-time code setup: otpauth://totp/Example:alex?secret=GEZDGNBVGY3TQOJQ&issuer=Example\n" +
		"PIN hint: hint\nRecovery: first recovery\nRecovery: second recovery\norphan\nlogin.otp_hint: legacy"
	if credential.Notes != want || credential.TOTP != normalized(t, "JBSWY3DPEHPK3PXP") || !reflect.DeepEqual(credential.Websites, []string{"login.example.test"}) {
		t.Fatalf("read %+v\nnotes %q", credential, credential.Notes)
	}
}

func TestTwoOneTimeCodesKeepTheFirstAndWriteTheSecond(t *testing.T) {
	second := "otpauth://totp/Example:alex?secret=GEZDGNBVGY3TQOJQ&issuer=Example"
	item := onlyItem(t, `{"id": "l1", "name": "Mail", "itemType": "Login", "fieldValues": [], "totpCodes": [
		{"name": "Phone", "secretKey": "JBSWY3DPEHPK3PXP"},
		{"name": "Backup", "secretKey": "`+second+`"}
	], "passkeys": []}`)
	credential := item.Content.Credential
	if credential.TOTP != normalized(t, "JBSWY3DPEHPK3PXP") || credential.Notes != "One-time code setup (Backup): "+second {
		t.Fatalf("read %+v", credential)
	}
}

func TestAliasWritesWhoItIsInItsNotes(t *testing.T) {
	export := readItems(t, itemOf("Alias", "Forum",
		"login.username", "alexd",
		"login.email", "alex@example.test",
		"alias.first_name", "Alex",
		"alias.last_name", " Doe ",
		"alias.gender", "Female",
		"alias.birthdate", "1990-05-17T00:00:00",
		"notes.content", "Own notes",
	))
	wantItem(t, export.Items[0], importers.Item{
		Content: vault.NewItem{Credential: &vault.CredentialInput{
			Label: "Forum",
			Login: "alexd",
			Email: "alex@example.test",
			Notes: "Own notes\n\nName: Alex Doe\nGender: Female\nDate of birth: 1990-05-17",
		}},
		Origin: importers.OriginAlias,
	})
	converted := importers.NewPlan(export, nil, nil).Preview().Kinds[0].Converted
	if want := []importers.Conversion{{From: importers.OriginAlias, Count: 1}}; !reflect.DeepEqual(converted, want) {
		t.Fatalf("converted %+v", converted)
	}
	unborn := onlyItem(t, itemOf("Alias", "Unborn", "alias.first_name", "Alex", "alias.birthdate", "0001-01-01T00:00:00"))
	if notes := unborn.Content.Credential.Notes; notes != "Name: Alex" {
		t.Fatalf("notes = %q", notes)
	}
}

func TestBirthdateReadsTheFormsAliasVaultWrites(t *testing.T) {
	tests := map[string]string{
		"1990-05-17":                "1990-05-17",
		" 1990-05-17 ":              "1990-05-17",
		"1990-05-17T00:00:00":       "1990-05-17",
		"1990-05-17T00:00:00.000Z":  "1990-05-17",
		"1990-05-17T23:30:00+05:00": "1990-05-17",
		"05/17/1990 00:00:00":       "1990-05-17",
		"05/17/1990":                "1990-05-17",
		"1850-01-01":                "1850-01-01",
		"0001-01-01":                "",
		"0001-01-01T00:00:00":       "",
		"01/01/0001 00:00:00":       "",
		"17.05.1990":                "17.05.1990",
		"":                          "",
	}
	for written, want := range tests {
		if got := birthdate(written); got != want {
			t.Errorf("birthdate(%q) = %q, want %q", written, got, want)
		}
	}
}

func TestUsernameThatIsAnEmailFillsAnEmptyEmail(t *testing.T) {
	tests := []struct {
		name         string
		values       []string
		login, email string
	}{
		{"alone", []string{"login.username", "alex@example.test"}, "", "alex@example.test"},
		{"beside an email", []string{"login.username", "alex@example.test", "login.email", "other@example.test"}, "alex@example.test", "other@example.test"},
		{"with a display name", []string{"login.username", "Alex <alex@example.test>"}, "Alex <alex@example.test>", ""},
		{"not an email", []string{"login.username", "alex"}, "alex", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			credential := onlyItem(t, itemOf("Login", "Mail", test.values...)).Content.Credential
			if credential.Login != test.login || credential.Email != test.email {
				t.Fatalf("login %q, email %q", credential.Login, credential.Email)
			}
		})
	}
}

func TestItemWithoutANameTakesAFallbackLabel(t *testing.T) {
	export := readItems(t,
		itemOf("Login", "", "login.url", "androidapp://com.example", "login.url", "https://www.fallback.example/sign-in", "login.username", "alex"),
		itemOf("Alias", " ", "login.username", " alexd ", "login.email", "alex@example.test"),
		itemOf("Login", "", "login.email", "alex@example.test", "login.password", "pw"),
		itemOf("Login", "", "login.password", "orphan"),
		itemOf("Alias", "", "alias.first_name", "Alex"),
		itemOf("Note", " ", "notes.content", "no name"),
		itemOf("CreditCard", "", "card.number", "4111111111111111"),
	)
	labels := make([]string, len(export.Items))
	for i, item := range export.Items {
		labels[i] = item.Content.Credential.Label
	}
	if want := []string{"fallback.example", "alexd", "alex@example.test"}; !reflect.DeepEqual(labels, want) {
		t.Fatalf("labels = %q, want %q", labels, want)
	}
	want := []importers.Skip{
		{Origin: importers.OriginLogin, Reason: importers.ReasonUnnamed},
		{Origin: importers.OriginAlias, Reason: importers.ReasonUnnamed},
		{Origin: importers.OriginNote, Reason: importers.ReasonUnnamed},
		{Origin: importers.OriginCard, Reason: importers.ReasonUnnamed},
	}
	if !reflect.DeepEqual(export.Skipped, want) {
		t.Fatalf("skipped %+v", export.Skipped)
	}
}

func TestCreditCardMapsOntoACard(t *testing.T) {
	manifest := `{"version": "1.0.0",
		"fieldDefinitions": [{"id": "d1", "label": "Bank", "weight": 0}],
		"items": [{"id": "c1", "name": "Visa", "itemType": "CreditCard", "fieldValues": [
			{"fieldKey": "card.cardholder_name", "value": "Alex Doe", "weight": 0},
			{"fieldKey": "card.number", "value": "4111 1111-1111 1111", "weight": 0},
			{"fieldKey": "card.expiry_month", "value": "1", "weight": 0},
			{"fieldKey": "card.expiry_year", "value": "27", "weight": 0},
			{"fieldKey": "card.cvv", "value": " 123 ", "weight": 0},
			{"fieldKey": "card.pin", "value": "1234", "weight": 0},
			{"fieldKey": "notes.content", "value": "Own notes", "weight": 0},
			{"fieldDefinitionId": "d1", "value": "Example Bank", "weight": 0}
		]}]}`
	wantItem(t, readContent(t, avuxOf(t, manifest)).Items[0], importers.Item{
		Content: vault.NewItem{Card: &vault.CardInput{
			Label:        "Visa",
			Holder:       "Alex Doe",
			Number:       "4111111111111111",
			Expiry:       "2027-01",
			SecurityCode: "123",
			PIN:          "1234",
			Notes:        "Own notes\n\nBank: Example Bank",
		}},
		Origin: importers.OriginCard,
	})
}

func TestCreditCardTakesItsAddressFieldAsItsBillingAddress(t *testing.T) {
	manifest := `{"version": "1.0.0",
		"fieldDefinitions": [{"id": "d1", "label": " Address ", "weight": 0}, {"id": "d2", "label": "Адрес", "weight": 1}],
		"items": [{"id": "c1", "name": "Visa", "itemType": "CreditCard", "fieldValues": [
			{"fieldKey": "card.number", "value": "4111111111111111", "weight": 0},
			{"fieldDefinitionId": "d1", "value": "Auezovsky District, microdistrict 6, 28", "weight": 0},
			{"fieldDefinitionId": "d2", "value": "Second address", "weight": 0}
		]}]}`
	wantItem(t, readContent(t, avuxOf(t, manifest)).Items[0], importers.Item{
		Content: vault.NewItem{Card: &vault.CardInput{
			Label:   "Visa",
			Number:  "4111111111111111",
			Billing: &vault.Address{Street: "Auezovsky District, microdistrict 6, 28"},
			Notes:   "Адрес: Second address",
		}},
		Origin: importers.OriginCard,
	})
}

func TestCreditCardWritesWhatItCannotHold(t *testing.T) {
	tests := []struct {
		month, year, code, pin string
		expiry                 string
		notes                  string
	}{
		{"12", "2030", "1234", "", "2030-12", ""},
		{"", "", "", "", "", ""},
		{"13", "2027", "123", "1234", "", "Expiration date: 13/2027"},
		{"1", "", "123", "1234", "", "Expiration date: 1"},
		{"1", "2027", "12", "1234", "2027-01", "Security code: 12"},
		{"1", "2027", "123", "12", "2027-01", "PIN: 12"},
		{"Jan", "2027", "12a", "12-34", "", "Expiration date: Jan/2027\nSecurity code: 12a\nPIN: 12-34"},
	}
	for _, test := range tests {
		t.Run(test.month+"/"+test.year+" "+test.code+" "+test.pin, func(t *testing.T) {
			card := onlyItem(t, itemOf("CreditCard", "Card",
				"card.number", "4111111111111111",
				"card.expiry_month", test.month,
				"card.expiry_year", test.year,
				"card.cvv", test.code,
				"card.pin", test.pin,
			)).Content.Card
			if card == nil || card.Expiry != test.expiry || card.Notes != test.notes || card.Network != 0 {
				t.Fatalf("read %+v", card)
			}
		})
	}
}

func TestCreditCardWithoutACardNumberBecomesAHiddenNote(t *testing.T) {
	item := onlyItem(t, itemOf("CreditCard", "Gift card",
		"card.cardholder_name", "Alex Doe",
		"card.number", "1234-5678",
		"card.expiry_month", "13",
		"card.expiry_year", "2027",
		"card.cvv", "12",
		"card.pin", "1234",
		"notes.content", "Own notes",
	))
	wantItem(t, item, importers.Item{
		Content: vault.NewItem{Note: &vault.NoteInput{
			Label:  "Gift card",
			Body:   "Own notes\n\nCardholder: Alex Doe\nCard number: 1234-5678\nExpiration date: 13/2027\nSecurity code: 12\nPIN: 1234",
			Hidden: true,
		}},
		Origin: importers.OriginCard,
	})
	empty := onlyItem(t, itemOf("CreditCard", "No number"))
	if !reflect.DeepEqual(*empty.Content.Note, vault.NoteInput{Label: "No number", Hidden: true}) {
		t.Fatalf("read %+v", empty.Content)
	}
}

func TestNoteMapsOntoANoteOrASeed(t *testing.T) {
	phrase := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	export := readContent(t, avuxOf(t, `{"version": "1.0.0",
		"fieldDefinitions": [{"id": "d1", "label": "Wallet", "weight": 0}],
		"items": [
			{"id": "n1", "name": "Router", "itemType": "Note", "fieldValues": [
				{"fieldKey": "notes.content", "value": "admin panel\n", "weight": 0},
				{"fieldDefinitionId": "d1", "value": "none", "weight": 0}
			]},
			{"id": "n2", "name": "Anytype", "itemType": "Note", "fieldValues": [
				{"fieldKey": "notes.content", "value": " `+phrase+`\n", "weight": 0},
				{"fieldDefinitionId": "d1", "value": "Anytype account", "weight": 0}
			]},
			{"id": "n3", "name": "Prose", "itemType": "Note", "fieldValues": [
				{"fieldKey": "notes.content", "value": "My phrase: `+phrase+`", "weight": 0}
			]}
		]}`))
	want := []importers.Item{
		{Content: vault.NewItem{Note: &vault.NoteInput{Label: "Router", Body: "admin panel\n\nWallet: none"}}, Origin: importers.OriginNote},
		{Content: vault.NewItem{Seed: &vault.SeedInput{Label: "Anytype", Format: vault.SeedPhrase, Words: strings.Fields(phrase), Notes: "Wallet: Anytype account"}}, Origin: importers.OriginNote},
		{Content: vault.NewItem{Note: &vault.NoteInput{Label: "Prose", Body: "My phrase: " + phrase}}, Origin: importers.OriginNote},
	}
	if len(export.Items) != len(want) {
		t.Fatalf("read %+v", export)
	}
	for i := range want {
		wantItem(t, export.Items[i], want[i])
	}
}

func TestItemsFileUnderTheirFolderPathThenTags(t *testing.T) {
	item := func(id, folder string) string {
		return `{"id": "` + id + `", "name": "` + id + `", "itemType": "Note", "folderId": ` + folder + `}`
	}
	export := readContent(t, avuxOf(t, `{"version": "1.0.0",
		"folders": [
			{"id": "f1", "name": "Work", "parentFolderId": null},
			{"id": "f2", "name": "Projects", "parentFolderId": "f1"},
			{"id": "f3", "name": "Active", "parentFolderId": "f2"},
			{"id": "c1", "name": "Loop A", "parentFolderId": "c2"},
			{"id": "c2", "name": "Loop B", "parentFolderId": "c1"},
			{"id": "s1", "name": "Self", "parentFolderId": "s1"},
			{"id": "o1", "name": "Orphan", "parentFolderId": "gone"},
			{"id": "b1", "name": "  ", "parentFolderId": null},
			{"id": "", "name": "Nameless", "parentFolderId": null}
		],
		"tags": [{"id": "t1", "name": "Personal"}, {"id": "t2", "name": "Shared"}, {"id": "t3", "name": " "}],
		"itemTags": [
			{"itemId": "i1", "tagId": "t1"},
			{"itemId": "i2", "tagId": "t2"},
			{"itemId": "i2", "tagId": "t1"},
			{"itemId": "i2", "tagId": "t3"},
			{"itemId": "i2", "tagId": "missing"},
			{"itemId": "i8", "tagId": "t2"}
		],
		"items": [`+strings.Join([]string{
		item("i1", `"f2"`), item("i2", `"f3"`), item("i3", `"c1"`), item("i4", `"s1"`),
		item("i5", `"o1"`), item("i6", `"b1"`), item("i7", `"missing"`), item("i8", `null`),
	}, ",")+`]}`))
	folders := make([][]string, len(export.Items))
	for i, filed := range export.Items {
		folders[i] = filed.Folders
	}
	want := [][]string{
		{"Work/Projects", "Personal"},
		{"Work/Projects/Active", "Shared", "Personal"},
		{"Loop B/Loop A"},
		{"Self"},
		{"Orphan"},
		nil,
		nil,
		{"Shared"},
	}
	if !reflect.DeepEqual(folders, want) {
		t.Fatalf("folders = %q, want %q", folders, want)
	}
}

func TestReadSkipsAnUnknownTypeAndCountsPasskeys(t *testing.T) {
	export := readItems(t,
		`{"id": "u1", "name": " Future ", "itemType": "Identity", "passkeys": [{"rpId": "example.test"}]}`,
		`{"id": "u2", "name": "Typeless", "itemType": null}`,
		`{"id": "u3", "name": "", "itemType": "Login", "passkeys": [{"rpId": "example.test", "privateKey": "{}"}]}`,
		`{"id": "u4", "name": "Two passkeys", "itemType": "Login", "passkeys": [{"rpId": "a.example"}, {"rpId": "b.example"}]}`,
	)
	want := []importers.Skip{
		{Label: "Future", Reason: importers.ReasonUnsupported},
		{Label: "Typeless", Reason: importers.ReasonUnsupported},
		{Origin: importers.OriginLogin, Reason: importers.ReasonUnnamed},
	}
	if !reflect.DeepEqual(export.Skipped, want) || export.Passkeys != 4 || len(export.Items) != 1 {
		t.Fatalf("read %+v", export)
	}
}

func TestReadWritesLabelsInThePersonsLanguage(t *testing.T) {
	opened := openContent(t, avuxOf(t, manifestOf(`{"id": "a1", "name": "Forum", "itemType": "Alias", "fieldValues": [
		{"fieldKey": "alias.first_name", "value": "Alex", "weight": 0},
		{"fieldKey": "alias.gender", "value": "Female", "weight": 0},
		{"fieldKey": "alias.birthdate", "value": "1990-05-17", "weight": 0}
	], "totpCodes": [{"name": "Steam", "secretKey": "steam://ABCDEF"}]}`)))
	russian := english
	russian.OneTimeCode, russian.Name, russian.Gender, russian.Birthday = "Настройка одноразовых кодов", "Имя", "Пол", "Дата рождения"
	export, err := opened.Read(russian)
	if err != nil {
		t.Fatal(err)
	}
	want := "Настройка одноразовых кодов (Steam): steam://ABCDEF\nИмя: Alex\nПол: Female\nДата рождения: 1990-05-17"
	if notes := export.Items[0].Content.Credential.Notes; notes != want {
		t.Fatalf("notes = %q", notes)
	}
}

func TestReadRefusesAValueOfTheWrongType(t *testing.T) {
	for _, items := range []string{
		`{"id": "l1", "name": "Quoted weight", "itemType": "Login", "fieldValues": [{"fieldKey": "login.username", "value": "alex", "weight": "1"}]}`,
		`{"id": "l1", "name": "Numbered type", "itemType": 1}`,
		`{"id": "l1", "name": "Passkeys", "itemType": "Login", "passkeys": "none"}`,
	} {
		if _, err := openContent(t, avuxOf(t, manifestOf(items))).Read(english); !errors.Is(err, importers.ErrUnrecognized) {
			t.Fatalf("Read of %s = %v", items, err)
		}
	}
}
