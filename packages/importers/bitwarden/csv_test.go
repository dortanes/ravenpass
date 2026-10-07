package bitwarden

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dortanes/ravenpass/packages/importers"
	"github.com/dortanes/ravenpass/packages/vault"
)

func TestCSVExportMapsLoginsAndNotes(t *testing.T) {
	export := readContent(t, []byte("\xef\xbb\xbf"+
		"folder,favorite,type,name,notes,fields,reprompt,archivedDate,login_uri,login_username,login_password,login_totp\r\n"+
		"Work,1,login,Example,Own notes,\"Recovery: ABCD\nKey: with: colon\nloose line\",0,,\"https://example.test/login,https://second.example.test\",alex@example.test,secret,JBSWY3DPEHPK3PXP\r\n"+
		",,note,Router,admin panel,,1,,,,,\r\n"+
		"Social,,login,,,,0,,https://www.fallback.example,alex,pw,steam://ABCDEF\r\n"))
	want := []importers.Item{
		{
			Content: vault.NewItem{Pinned: true, Credential: &vault.CredentialInput{
				Label:    "Example",
				Websites: []string{"example.test", "second.example.test"},
				Email:    "alex@example.test",
				Password: "secret",
				TOTP:     "JBSWY3DPEHPK3PXP",
				Notes:    "Own notes\n\nRecovery: ABCD\nKey: with: colon\nloose line",
			}},
			Origin:  importers.OriginLogin,
			Folders: []string{"Work"},
		},
		{
			Content: vault.NewItem{Note: &vault.NoteInput{Label: "Router", Body: "admin panel", Hidden: true}},
			Origin:  importers.OriginNote,
		},
		{
			Content: vault.NewItem{Credential: &vault.CredentialInput{
				Label:    "fallback.example",
				Websites: []string{"fallback.example"},
				Login:    "alex",
				Password: "pw",
				Notes:    "One-time code setup: steam://ABCDEF",
			}},
			Origin:  importers.OriginLogin,
			Folders: []string{"Social"},
		},
	}
	if export.Format != importers.FormatCSV || len(export.Items) != len(want) || len(export.Skipped) != 0 {
		t.Fatalf("read %+v", export)
	}
	for i := range want {
		wantItem(t, export.Items[i], want[i])
	}
}

func TestCSVOrganizationExportFilesItemsUnderCollections(t *testing.T) {
	export := readContent(t, []byte(
		"collections,type,name,notes,fields,reprompt,login_uri,login_username,login_password,login_totp\n"+
			"\"Shared,Team, ,Shared\",login,Org item,,,0,https://org.example,,pw,\n"))
	if folders := export.Items[0].Folders; !reflect.DeepEqual(folders, []string{"Shared", "Team", "Shared"}) {
		t.Fatalf("folders = %q", folders)
	}
}

func TestCSVReadsTheURIsOfALoginAsOneRecord(t *testing.T) {
	export := readContent(t, []byte(
		"login_uri,name,type\n"+
			"\"\"\"https://a.example/?q=1,2\"\", https://b.example,androidapp://com.example\",Quoted,login\n"+
			"\"https://c.example/\"\"odd\",Stray quote,login\n"))
	first := export.Items[0].Content.Credential
	if want := []string{"a.example", "b.example", "androidapp://com.example"}; !reflect.DeepEqual(first.Websites, want) || first.Notes != "" {
		t.Fatalf("read %+v", first)
	}
	if second := export.Items[1].Content.Credential; !reflect.DeepEqual(second.Websites, []string{"c.example"}) {
		t.Fatalf("read %+v", second)
	}
}

func TestCSVReadsAnyColumnOrderAndMissingCells(t *testing.T) {
	export := readContent(t, []byte("name,type\nOnly,login\nShort\n"))
	want := []importers.Item{
		{Content: vault.NewItem{Credential: &vault.CredentialInput{Label: "Only"}}, Origin: importers.OriginLogin},
		{Content: vault.NewItem{Credential: &vault.CredentialInput{Label: "Short"}}, Origin: importers.OriginLogin},
	}
	for i := range want {
		wantItem(t, export.Items[i], want[i])
	}
}

func TestCSVRefusesAMalformedRecord(t *testing.T) {
	opened := openContent(t, []byte("type,name\nlogin,\"unterminated\n"))
	if _, err := opened.Read(english); !errors.Is(err, importers.ErrUnrecognized) {
		t.Fatalf("Read = %v", err)
	}
}
