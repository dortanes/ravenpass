package vault

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func commitBatch(t *testing.T, session *Session, items []NewItem) BatchResult {
	t.Helper()
	pending, result, err := session.PrepareAddItems(items)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	return result
}

func listedByID(t *testing.T, session *Session) map[ID]Entry {
	t.Helper()
	entries, err := session.List()
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[ID]Entry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	return byID
}

func TestPrepareAddItemsAddsEveryKindInOneSave(t *testing.T) {
	created, held := groupedVault(t, "Work")
	session := created.Session
	defer session.Lock()
	before := mustHead(t, session)
	credential := CredentialInput{Label: "Mail", Websites: []string{"https://mail.example.test/login"}, Login: "alex", Password: "batch-password", TOTP: "JBSWY3DPEHPK3PXP"}
	card := fullCard()
	card.Billing = nil
	identity := withoutAddressIDs(fullIdentity())
	note := NoteInput{Label: "Router", Body: "admin panel\nsecond line"}
	seed := SeedInput{Label: "Wallet", Format: SeedPhrase, Words: strings.Fields("Abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"), Notes: "cold storage"}

	result := commitBatch(t, session, []NewItem{
		{Credential: &credential, Pinned: true, Groups: []string{"Personal"}},
		{Card: &card},
		{Identity: &identity, Groups: []string{"personal ", "  work"}},
		{Note: &note, Groups: []string{"WORK"}},
		{Seed: &seed},
	})

	if after := mustHead(t, session); after.Revision != before.Revision+1 {
		t.Fatalf("revision moved from %d to %d, not by one save", before.Revision, after.Revision)
	}
	if len(result.Items) != 5 || result.Groups != 1 {
		t.Fatalf("result = %+v", result)
	}
	groups, err := session.Groups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].ID != held[0] || groups[1].Name != "Personal" {
		t.Fatalf("groups = %+v", groups)
	}
	personal := groups[1].ID
	listed := listedByID(t, session)
	wants := []struct {
		kind   Kind
		pinned bool
		groups []ID
	}{
		{KindCredential, true, []ID{personal}},
		{KindCard, false, nil},
		{KindIdentity, false, sortedIDs(held[0], personal)},
		{KindNote, false, []ID{held[0]}},
		{KindSeed, false, nil},
	}
	for i, want := range wants {
		entry := listed[result.Items[i]]
		if entry.Kind != want.kind || entry.Pinned != want.pinned || !reflect.DeepEqual(entry.Groups, want.groups) {
			t.Fatalf("item %d lists as %+v, want %+v", i, entry, want)
		}
	}
	if read := selectCredential(t, session, result.Items[0]); read.Password != credential.Password || read.TOTP != credential.TOTP {
		t.Fatalf("credential read back as %+v", read)
	}
	if read := selectCard(t, session, result.Items[1]); read.CardInput.Number != card.Number || read.CardInput.Label != card.Label {
		t.Fatalf("card read back as %+v", read)
	}
	readIdentity := selectIdentity(t, session, result.Items[2])
	if len(readIdentity.Addresses) != 2 || readIdentity.Addresses[0].ID == (ID{}) || readIdentity.Addresses[0].ID == readIdentity.Addresses[1].ID {
		t.Fatalf("addresses were not given distinct ids: %+v", readIdentity.Addresses)
	}
	if !reflect.DeepEqual(withoutAddressIDs(readIdentity.IdentityInput), withoutAddressIDs(identity)) {
		t.Fatalf("identity read back as %+v", readIdentity.IdentityInput)
	}
	if read := selectNote(t, session, result.Items[3]); !reflect.DeepEqual(read.NoteInput, note) {
		t.Fatalf("note read back as %+v", read.NoteInput)
	}
	readSeed := selectSeed(t, session, result.Items[4])
	if readSeed.Checksum != SeedChecksumValid || readSeed.Words[0] != "abandon" || readSeed.Notes != seed.Notes || readSeed.CheckedOn != "" {
		t.Fatalf("seed read back as %+v", readSeed)
	}
}

func TestPrepareAddItemsCreatesOneGroupPerName(t *testing.T) {
	created, _ := groupedVault(t)
	session := created.Session
	defer session.Lock()
	first := NoteInput{Label: "One"}
	second := NoteInput{Label: "Two"}
	result := commitBatch(t, session, []NewItem{
		{Note: &first, Groups: []string{"Travel", "travel "}},
		{Note: &second, Groups: []string{" TRAVEL"}},
	})
	groups, err := session.Groups()
	if err != nil {
		t.Fatal(err)
	}
	if result.Groups != 1 || len(groups) != 1 || groups[0].Name != "Travel" {
		t.Fatalf("result %+v, groups %+v", result, groups)
	}
	listed := listedByID(t, session)
	for _, id := range result.Items {
		if !reflect.DeepEqual(listed[id].Groups, []ID{groups[0].ID}) {
			t.Fatalf("%s is in %v", id, listed[id].Groups)
		}
	}
}

func TestPrepareAddItemsRefusesTheWholeBatch(t *testing.T) {
	valid := NoteInput{Label: "Fine"}
	long := NoteInput{Label: strings.Repeat("x", MaxLabelLength+1)}
	credential := CredentialInput{Label: "Mail"}
	linked := fullCard()
	linked.Billing = nil
	linked.BillingLink = &AddressLink{Identity: ID{1}, Address: ID{2}}
	withPhoto := IdentityInput{Label: "Photo", Photo: []byte{1}}
	withScans := IdentityInput{Label: "Scans", Documents: []Document{{Type: DocumentPassport, Number: "P1", Scans: []ID{{3}}}}}
	withAttach := IdentityInput{Label: "Attach", Documents: []Document{{Type: DocumentPassport, Number: "P1", Attach: []PreparedScan{{}}}}}
	withAddressID := IdentityInput{Label: "Address", Addresses: []Address{{ID: ID{4}, Street: "Main"}}}
	tooManyNames := make([]string, MaxCredentialGroups+1)
	for i := range tooManyNames {
		tooManyNames[i] = fmt.Sprintf("Group %d", i)
	}
	beyondMaxGroups := make([]NewItem, 0, MaxGroups+1)
	for i := range MaxGroups + 1 {
		beyondMaxGroups = append(beyondMaxGroups, NewItem{Note: &valid, Groups: []string{fmt.Sprintf("Group %d", i)}})
	}
	tests := []struct {
		name  string
		items []NewItem
		want  error
	}{
		{"empty batch", nil, ErrInvalidInput},
		{"no content", []NewItem{{Note: &valid}, {Pinned: true}}, ErrInvalidInput},
		{"two contents", []NewItem{{Note: &valid, Credential: &credential}}, ErrInvalidInput},
		{"label over the bound", []NewItem{{Note: &valid}, {Note: &long}}, ErrInvalidInput},
		{"card link", []NewItem{{Card: &linked}}, ErrInvalidInput},
		{"identity photo", []NewItem{{Identity: &withPhoto}}, ErrInvalidInput},
		{"identity scans", []NewItem{{Identity: &withScans}}, ErrInvalidInput},
		{"identity scans to attach", []NewItem{{Identity: &withAttach}}, ErrInvalidInput},
		{"address with an id", []NewItem{{Identity: &withAddressID}}, ErrInvalidInput},
		{"blank group name", []NewItem{{Note: &valid, Groups: []string{"  "}}}, ErrInvalidInput},
		{"group name over the bound", []NewItem{{Note: &valid, Groups: []string{strings.Repeat("g", MaxGroupNameLength+1)}}}, ErrInvalidInput},
		{"too many groups on one item", []NewItem{{Note: &valid, Groups: tooManyNames}}, ErrResourceLimit},
		{"too many groups in the vault", beyondMaxGroups, ErrResourceLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			created, _ := groupedVault(t, "Work")
			session := created.Session
			defer session.Lock()
			before := mustHead(t, session)
			pending, _, err := session.PrepareAddItems(test.items)
			if !errors.Is(err, test.want) || pending != nil {
				t.Fatalf("PrepareAddItems = %v, %v; want %v", pending, err, test.want)
			}
			if after := mustHead(t, session); after != before {
				t.Fatal("a refused batch changed the vault")
			}
			commitNote(t, session, valid, nil)
		})
	}
}

func TestPrepareAddItemsNeedsAnOpenIdleSession(t *testing.T) {
	created, _ := groupedVault(t)
	session := created.Session
	note := NoteInput{Label: "Note"}
	pending, _, err := session.PrepareCreateNote(note, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := session.PrepareAddItems([]NewItem{{Note: &note}}); !errors.Is(err, ErrPendingCommit) {
		t.Fatalf("with a save awaiting confirmation: %v", err)
	}
	if err := session.Abort(pending); err != nil {
		t.Fatal(err)
	}
	session.Lock()
	if _, _, err := session.PrepareAddItems([]NewItem{{Note: &note}}); !errors.Is(err, ErrLocked) {
		t.Fatalf("locked: %v", err)
	}
}

func TestPreviewNewItemListsWhatAnItemWouldShow(t *testing.T) {
	credential := CredentialInput{Label: "Mail", Websites: []string{"https://www.Mail.example.test/login"}, Login: "alex", Email: "alex@example.test"}
	card := fullCard()
	card.Billing = nil
	identity := withoutAddressIDs(fullIdentity())
	note := NoteInput{Label: "Router", Body: "\n  first line  \nsecond"}
	seed := SeedInput{Label: "Wallet", Format: SeedPhrase, Words: []string{"abandon", "about"}, Wallet: "Ledger"}
	tests := []struct {
		item NewItem
		want Entry
	}{
		{NewItem{Credential: &credential, Pinned: true, Groups: []string{"Work"}}, Entry{Kind: KindCredential, Label: "Mail", Detail: "alex", Site: "mail.example.test", Sites: []string{"mail.example.test"}, Email: "alex@example.test", Pinned: true}},
		{NewItem{Card: &card}, Entry{Kind: KindCard, Label: card.Label, Detail: card.BankName, ExpiresOn: "2029-08-31", Site: "bank.example", Card: CardFace{Network: NetworkVisa, LastFour: "1111", Color: card.Color}}},
		{NewItem{Identity: &identity}, Entry{Kind: KindIdentity, Label: identity.Label, Detail: "alex@example.test", ExpiresOn: "2024-02-29"}},
		{NewItem{Note: &note}, Entry{Kind: KindNote, Label: "Router", Detail: "first line"}},
		{NewItem{Seed: &seed}, Entry{Kind: KindSeed, Label: "Wallet", Detail: "Ledger", Seed: SeedFace{Format: SeedPhrase, Total: 2}}},
	}
	for _, test := range tests {
		got, err := PreviewNewItem(test.item)
		if err != nil {
			t.Fatal(err)
		}
		if got.Thumbnail != nil || got.Groups != nil {
			t.Fatalf("preview carries memory it should not: %+v", got)
		}
		got.Thumbnail, got.Groups = nil, nil
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("PreviewNewItem = %+v, want %+v", got, test.want)
		}
	}
	invalid := CredentialInput{Label: "Mail", TOTP: "not a setup"}
	if _, err := PreviewNewItem(NewItem{Credential: &invalid}); !errors.Is(err, ErrInvalidTOTP) {
		t.Fatalf("an invalid one-time code setup previewed: %v", err)
	}
}

func TestBatchValuesNeverReachTheContainerInPlaintext(t *testing.T) {
	created, _ := groupedVault(t)
	session := created.Session
	defer session.Lock()
	credential := CredentialInput{Label: "Batch-label-credential", Login: "batch-login", Password: "batch-password-secret", Notes: "batch-credential-notes"}
	note := NoteInput{Label: "Batch-label-note", Body: "batch-note-body"}
	commitBatch(t, session, []NewItem{{Credential: &credential, Groups: []string{"Batch-group"}}, {Note: &note}})
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{credential.Label, credential.Login, credential.Password, credential.Notes, note.Label, note.Body, "Batch-group"} {
		if bytes.Contains(container, []byte(value)) {
			t.Fatalf("%q appears in the vault file", value)
		}
	}
}
