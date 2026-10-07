package api

import (
	"slices"
	"strings"
	"testing"

	"github.com/dortanes/ravenpass/packages/vault"
)

func TestTheTrashListsRestoresAndEmpties(t *testing.T) {
	service := newReadyService(t)
	mail, err := service.CreateCredential(CredentialInput{Label: "Mail", Login: "alex", Password: "secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	note, err := service.CreateNote(NoteInput{Label: "Door", Body: "4821"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{mail, note} {
		if err := service.TrashItem(id); err != nil {
			t.Fatal(err)
		}
	}
	if credentials, err := service.ListCredentials(); err != nil || len(credentials) != 0 {
		t.Fatalf("credentials beside the trash = %+v, %v", credentials, err)
	}
	trash, err := service.ListTrash()
	if err != nil || trash.RetentionDays != vault.DefaultTrashRetention || len(trash.Items) != 2 {
		t.Fatalf("trash = %+v, %v", trash, err)
	}
	if first := trash.Items[1]; first.ID != mail || first.Kind != "credential" || first.Label != "Mail" || first.Detail != "alex" || first.DeletedAt == 0 {
		t.Fatalf("the trashed credential = %+v", first)
	}
	if trash.Items[0].DeletedAt < trash.Items[1].DeletedAt {
		t.Fatalf("the trash is not latest first: %+v", trash.Items)
	}
	_, err = service.ReadCredential(mail)
	assertFailure(t, err, failureItemUnreadable)

	if err := service.RestoreItem(mail); err != nil {
		t.Fatal(err)
	}
	if err := service.EmptyTrash(); err != nil {
		t.Fatal(err)
	}
	if trash, err := service.ListTrash(); err != nil || len(trash.Items) != 0 {
		t.Fatalf("trash after emptying = %+v, %v", trash, err)
	}
	if credential, err := service.ReadCredential(mail); err != nil || credential.Password != "secret" {
		t.Fatalf("the restored credential = %+v, %v", credential, err)
	}
	assertFailure(t, service.RestoreItem(note), failureItemUnreadable)
	assertFailure(t, service.TrashItem("not an id"), failureItemUnreadable)
}

func TestTheTrashRetentionStaysWithinItsBounds(t *testing.T) {
	service := newReadyService(t)
	if err := service.SetTrashRetention(7); err != nil {
		t.Fatal(err)
	}
	if trash, err := service.ListTrash(); err != nil || trash.RetentionDays != 7 {
		t.Fatalf("trash = %+v, %v", trash, err)
	}
	assertFailure(t, service.SetTrashRetention(vault.MaxTrashRetention+1), failureInvalidItem)
}

func TestMergeCredentialsKeepsLinkedAppsOfBoth(t *testing.T) {
	service := newReadyService(t)
	into, err := service.vault.CreateCredential(vault.CredentialInput{Label: "Mail", Password: "current", Apps: []vault.App{{Package: "com.example.mail", Signer: [32]byte{0xab}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	from, err := service.vault.CreateCredential(vault.CredentialInput{Label: "Mail old", Login: "alex", Apps: []vault.App{{Package: "com.example.chat", Signer: [32]byte{0xcd}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	kept, err := service.ReadCredential(into.String())
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.ReadCredential(from.String())
	if err != nil {
		t.Fatal(err)
	}
	input := kept.CredentialInput
	input.Login = other.Login
	input.Apps = append(input.Apps, other.Apps...)
	if err := service.MergeCredentials(into.String(), from.String(), input, nil); err != nil {
		t.Fatal(err)
	}
	merged, err := service.ReadCredential(into.String())
	if err != nil {
		t.Fatal(err)
	}
	if merged.Login != "alex" || merged.Password != "current" || len(merged.Apps) != 2 {
		t.Fatalf("merged = %+v", merged.CredentialInput)
	}
	if trash, err := service.ListTrash(); err != nil || len(trash.Items) != 1 || trash.Items[0].ID != from.String() {
		t.Fatalf("trash = %+v, %v", trash, err)
	}

	input.Apps = []LinkedApp{{Package: "com.attacker.mail", Signer: strings.Repeat("ab", 32)}}
	third, err := service.CreateCredential(CredentialInput{Label: "Third"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.MergeCredentials(into.String(), third, input, nil), failureInvalidItem)
	assertFailure(t, service.MergeCredentials(into.String(), from.String(), merged.CredentialInput, nil), failureItemUnreadable)
	if trash, err := service.ListTrash(); err != nil || !slices.ContainsFunc(trash.Items, func(item TrashedItem) bool { return item.ID == from.String() }) || len(trash.Items) != 1 {
		t.Fatalf("trash after refused merges = %+v, %v", trash, err)
	}
}
