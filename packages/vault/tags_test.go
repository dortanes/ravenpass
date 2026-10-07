package vault

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestAcceptTagsTidiesAndDeduplicates(t *testing.T) {
	tags, err := AcceptTags([]string{"  Work ", "", "work  account", "work", "Работа"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Work", "work account", "Работа"}; !slices.Equal(tags, want) {
		t.Fatalf("tags = %q, want %q", tags, want)
	}
	if tags, err := AcceptTags([]string{" ", ""}); err != nil || tags != nil {
		t.Fatalf("blank tags = %q, %v", tags, err)
	}
}

func TestAcceptTagsRefusesEachBound(t *testing.T) {
	many := make([]string, MaxItemTags+1)
	for i := range many {
		many[i] = strings.Repeat("t", i+1)
	}
	tests := []struct {
		name string
		tags []string
		want error
	}{
		{"overlong tag", []string{strings.Repeat("я", MaxTagLength+1)}, ErrInvalidInput},
		{"control character", []string{"a\x07b"}, ErrInvalidInput},
		{"invalid text", []string{"\xff"}, ErrInvalidInput},
		{"too many tags", many, ErrResourceLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := AcceptTags(test.tags); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
	if tags, err := AcceptTags([]string{strings.Repeat("я", MaxTagLength)}); err != nil || len(tags) != 1 {
		t.Fatalf("tag at its limit = %q, %v", tags, err)
	}
}

func TestTagsLiveInTheIndexAcrossAReopen(t *testing.T) {
	created, _ := vaultWith(t)
	credential := commitCredential(t, created.Session, CredentialInput{Label: "Example", Password: "p", Tags: []string{"Personal"}}, nil)
	note := commitNote(t, created.Session, NoteInput{Label: "Note", Body: "text", Tags: []string{"Work"}}, nil)
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
	if entry := listedEntry(t, reopened, credential); !slices.Equal(entry.Tags, []string{"Personal"}) {
		t.Fatalf("credential entry tags = %q", entry.Tags)
	}
	if read, err := reopened.ReadCredential(credential); err != nil || !slices.Equal(read.Tags, []string{"Personal"}) {
		t.Fatalf("credential read tags = %q, %v", read.Tags, err)
	}
	if read := selectNote(t, reopened, note); !slices.Equal(read.Tags, []string{"Work"}) {
		t.Fatalf("note read tags = %q", read.Tags)
	}
}

func TestEditKeepsTagsUnlessThePatchSetsThem(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	id := commitCredential(t, session, CredentialInput{Label: "Example", Password: "p", Tags: []string{"Personal"}}, nil)
	password := "next"
	pending, err := session.PrepareEdit(id, CredentialPatch{Password: &password})
	commitPending(t, session, pending, err)
	if entry := listedEntry(t, session, id); !slices.Equal(entry.Tags, []string{"Personal"}) {
		t.Fatalf("tags after an edit = %q", entry.Tags)
	}
	tags := []string{"Shared", "shared"}
	pending, err = session.PrepareEdit(id, CredentialPatch{Tags: &tags})
	commitPending(t, session, pending, err)
	if entry := listedEntry(t, session, id); !slices.Equal(entry.Tags, []string{"Shared"}) {
		t.Fatalf("tags after setting them = %q", entry.Tags)
	}
	none := []string{}
	pending, err = session.PrepareEdit(id, CredentialPatch{Tags: &none})
	commitPending(t, session, pending, err)
	if entry := listedEntry(t, session, id); entry.Tags != nil {
		t.Fatalf("tags after clearing them = %q", entry.Tags)
	}
}

func TestSeedChangesFromWithinKeepTags(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	input := fullCodesSeed()
	input.Tags = []string{"Exchange"}
	id := commitSeed(t, session, input, nil)
	pending, err := session.PrepareUseBackupCode(id, 0)
	commitPending(t, session, pending, err)
	if entry := listedEntry(t, session, id); !slices.Equal(entry.Tags, []string{"Exchange"}) {
		t.Fatalf("tags after using a code = %q", entry.Tags)
	}
}

func TestAnEntryWithoutTagsKeepsTheEarlierEncoding(t *testing.T) {
	entry := entryMeta{id: ID{1}, revision: 1, label: "Mail", kind: KindCredential}
	plain, err := encoding.Marshal(entry.wire())
	if err != nil {
		t.Fatal(err)
	}
	if plain[0] != untaggedHead {
		t.Fatalf("untagged entry head = %#x", plain[0])
	}
	entry.tags = []string{"Personal"}
	tagged, err := encoding.Marshal(entry.wire())
	if err != nil {
		t.Fatal(err)
	}
	if tagged[0] != untaggedHead+1 || !slices.Equal(tagged[1:len(plain)], plain[1:]) {
		t.Fatalf("tagged entry = %x, untagged %x", tagged, plain)
	}
}

func TestIndexReadsTagsOnlyAsTheVaultWritesThem(t *testing.T) {
	records := stubRecords(1)
	digest := sha256.Sum256(encodeBox(records[0]))
	id := ID{5}
	credential := uint64(KindCredential)
	tagged := func(tags ...string) []byte {
		values := make([][]byte, len(tags))
		for i, tag := range tags {
			values[i] = encodeBytes([]byte(tag))
		}
		return encodeArray(append(currentFields(id, digest, credential, encodeArray()), encodeArray(values...))...)
	}
	_, _, entries, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), tagged("Personal", "Work")), records)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(entries[0].tags, []string{"Personal", "Work"}) {
		t.Fatalf("tags = %q", entries[0].tags)
	}
	tests := []struct {
		name    string
		element []byte
	}{
		{"empty tags element", tagged()},
		{"untrimmed tag", tagged(" Personal")},
		{"repeated tag", tagged("Personal", "personal")},
		{"blank tag", tagged("")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), test.element), records); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestIndexTagsRefuseAnAttachmentAndMoreThanAnItemHolds(t *testing.T) {
	if _, err := parseTags([]string{"Work"}, KindAttachment); !errors.Is(err, ErrMalformed) {
		t.Fatalf("an attachment's tags: got %v, want ErrMalformed", err)
	}
	many := make([]string, MaxItemTags+1)
	for i := range many {
		many[i] = fmt.Sprintf("Tag %d", i)
	}
	if _, err := parseTags(many, KindCredential); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("%d tags: got %v, want ErrResourceLimit", len(many), err)
	}
	if _, err := AcceptTags(many); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("accepting %d tags: got %v, want ErrResourceLimit", len(many), err)
	}
}
