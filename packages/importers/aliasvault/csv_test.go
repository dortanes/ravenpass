package aliasvault

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strings"
	"testing"

	"github.com/dortanes/ravenpass/packages/importers"
	"github.com/dortanes/ravenpass/packages/vault"
)

// csvOf writes a CSV export as AliasVault lays it out, with CRLF line endings.
func csvOf(t *testing.T, header []string, rows ...map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	writer.UseCRLF = true
	records := [][]string{header}
	for _, row := range rows {
		record := make([]string, len(header))
		for i, column := range header {
			record[i] = row[column]
		}
		records = append(records, record)
	}
	if err := writer.WriteAll(records); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func csvColumns() []string {
	return strings.Split(strings.TrimSuffix(csvHeader, "\r\n"), ",")
}

func TestCSVReadsEachRowKind(t *testing.T) {
	phrase := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	export := readContent(t, csvOf(t, csvColumns(),
		map[string]string{
			columnName: "Visa", columnFolder: "Finance", columnHolder: "Alex Doe", columnNumber: "4111 1111 1111 1111",
			columnExpiryMonth: "01", columnExpiryYear: "2027", columnSecurityCode: "123", columnPIN: "1234",
			columnNotes: "Own notes",
		},
		map[string]string{
			columnName: "Mail", columnURLs: "https://mail.example.test, https://webmail.example.test,", columnUsername: "alexd",
			columnPassword: "secret", columnEmail: "alex@example.test", columnTOTP: "JBSWY3DPEHPK3PXP",
			columnGender: "Female", columnFirstName: "Alex", columnLastName: "Doe", columnBirthdate: "05/17/1990 00:00:00",
		},
		map[string]string{
			columnName: "Example", columnFolder: "Work/Projects", columnURLs: "https://example.test",
			columnUsername: "alex@example.test", columnPassword: "pw", columnBirthdate: "01/01/0001 00:00:00",
		},
		map[string]string{columnName: "Code only", columnTOTP: "steam://ABCDEF"},
		map[string]string{columnName: "Router", columnFolder: " ", columnNotes: "admin panel"},
		map[string]string{columnName: "Wallet", columnNotes: phrase},
	))
	want := []importers.Item{
		{
			Content: vault.NewItem{Card: &vault.CardInput{
				Label: "Visa", Holder: "Alex Doe", Number: "4111111111111111", Expiry: "2027-01", SecurityCode: "123", PIN: "1234", Notes: "Own notes",
			}},
			Origin:  importers.OriginCard,
			Folders: []string{"Finance"},
		},
		{
			Content: vault.NewItem{Credential: &vault.CredentialInput{
				Label:    "Mail",
				Websites: []string{"mail.example.test", "webmail.example.test"},
				Login:    "alexd",
				Email:    "alex@example.test",
				Password: "secret",
				TOTP:     normalized(t, "JBSWY3DPEHPK3PXP"),
				Notes:    "Name: Alex Doe\nGender: Female\nDate of birth: 1990-05-17",
			}},
			Origin: importers.OriginAlias,
		},
		{
			Content: vault.NewItem{Credential: &vault.CredentialInput{
				Label: "Example", Websites: []string{"example.test"}, Email: "alex@example.test", Password: "pw",
			}},
			Origin:  importers.OriginLogin,
			Folders: []string{"Work/Projects"},
		},
		{
			Content: vault.NewItem{Credential: &vault.CredentialInput{Label: "Code only", Notes: "One-time code setup: steam://ABCDEF"}},
			Origin:  importers.OriginLogin,
		},
		{
			Content: vault.NewItem{Note: &vault.NoteInput{Label: "Router", Body: "admin panel"}},
			Origin:  importers.OriginNote,
		},
		{
			Content: vault.NewItem{Seed: &vault.SeedInput{Label: "Wallet", Format: vault.SeedPhrase, Words: strings.Fields(phrase)}},
			Origin:  importers.OriginNote,
		},
	}
	if export.Format != importers.FormatCSV || len(export.Items) != len(want) || len(export.Skipped) != 0 {
		t.Fatalf("read %+v", export)
	}
	for i := range want {
		wantItem(t, export.Items[i], want[i])
	}
}

func TestCSVReadsAnExportMadeBeforeCards(t *testing.T) {
	header := []string{
		columnName, columnFolder, columnURLs, columnUsername, columnPassword, columnEmail, columnTOTP, columnGender,
		columnFirstName, columnLastName, columnNickname, columnBirthdate, columnNotes, "CreatedAt", "UpdatedAt",
	}
	export := readContent(t, csvOf(t, header,
		map[string]string{
			columnName: "Forum", columnURLs: "https://forum.example.test", columnUsername: "alexd", columnPassword: "pw",
			columnEmail: "alex@example.test", columnGender: "Male", columnFirstName: "Alex", columnLastName: "Doe",
			columnNickname: "Lexi", columnBirthdate: "01/01/1990 00:00:00", columnNotes: "Own notes",
		},
		map[string]string{columnName: "Nickname only", columnNickname: "Lexi"},
	))
	want := []importers.Item{
		{
			Content: vault.NewItem{Credential: &vault.CredentialInput{
				Label:    "Forum",
				Websites: []string{"forum.example.test"},
				Login:    "alexd",
				Email:    "alex@example.test",
				Password: "pw",
				Notes:    "Own notes\n\nName: Alex Doe\nNickname: Lexi\nGender: Male\nDate of birth: 1990-01-01",
			}},
			Origin: importers.OriginAlias,
		},
		{
			Content: vault.NewItem{Credential: &vault.CredentialInput{Label: "Nickname only", Notes: "Nickname: Lexi"}},
			Origin:  importers.OriginAlias,
		},
	}
	if len(export.Items) != len(want) {
		t.Fatalf("read %+v", export)
	}
	for i := range want {
		wantItem(t, export.Items[i], want[i])
	}
}

func TestCSVReadsAnyColumnOrderAndMissingCells(t *testing.T) {
	export := readContent(t, []byte("CurrentPassword,ServiceName\npw,Only\n,Note only\nsecret\n"))
	want := []importers.Item{
		{Content: vault.NewItem{Credential: &vault.CredentialInput{Label: "Only", Password: "pw"}}, Origin: importers.OriginLogin},
		{Content: vault.NewItem{Note: &vault.NoteInput{Label: "Note only"}}, Origin: importers.OriginNote},
	}
	if len(export.Items) != len(want) || len(export.Skipped) != 1 || export.Skipped[0] != (importers.Skip{Origin: importers.OriginLogin, Reason: importers.ReasonUnnamed}) {
		t.Fatalf("read %+v", export)
	}
	for i := range want {
		wantItem(t, export.Items[i], want[i])
	}
}

func TestCSVRefusesAMalformedRecord(t *testing.T) {
	opened := openContent(t, []byte("ServiceName,CurrentPassword\nRouter,\"unterminated\n"))
	if _, err := opened.Read(english); !errors.Is(err, importers.ErrUnrecognized) {
		t.Fatalf("Read = %v", err)
	}
}
