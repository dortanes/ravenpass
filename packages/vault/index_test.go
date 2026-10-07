package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func stubRecords(count int) []sealedBox {
	records := make([]sealedBox, count)
	for i := range records {
		records[i] = sealedBox{ciphertext: bytes.Repeat([]byte{byte(i + 1)}, 16)}
	}
	return records
}

func withDigests(entries []entryMeta, records []sealedBox) []entryMeta {
	for i := range entries {
		entries[i].digest = sha256.Sum256(encodeBox(records[i]))
	}
	return entries
}

func groupTable(groups ...Group) []byte {
	elements := make([][]byte, len(groups))
	for i, group := range groups {
		elements[i] = encodeArray(encodeBytes(group.ID[:]), encodeBytes([]byte(group.Name)))
	}
	return encodeArray(elements...)
}

func membership(ids ...ID) []byte {
	elements := make([][]byte, len(ids))
	for i, id := range ids {
		elements[i] = encodeBytes(id[:])
	}
	return encodeArray(elements...)
}

// Positions of the fields of an index entry.
const (
	fieldRevision  = 1
	fieldLabel     = 2
	fieldDetail    = 3
	fieldPinned    = 4
	fieldGroups    = 6
	fieldExpiresOn = 8
	fieldThumbnail = 9
	fieldSite      = 10
	fieldOwner     = 11
	fieldEmail     = 12
	fieldCard      = 13
	fieldSummary   = 14
	fieldSites     = 15
	fieldDigits    = 16
	fieldPeriod    = 17
	fieldPasskeys  = 18
	fieldApps      = 19
)

// currentFields are the fields of an otherwise empty entry with the given summary.
func currentFields(id ID, digest [32]byte, kind uint64, summary []byte) [][]byte {
	return [][]byte{encodeBytes(id[:]), encodeUint(1), encodeBytes([]byte("Label")), encodeBytes(nil), encodeUint(0), encodeBytes(digest[:]), encodeArray(), encodeUint(kind), encodeBytes(nil), encodeBytes(nil), encodeBytes(nil), encodeBytes(nil), encodeBytes(nil), encodeArray(), summary, encodeArray(), encodeUint(0), encodeUint(0), encodeArray(), encodeArray()}
}

func currentElement(id ID, digest [32]byte, kind uint64, summary []byte) []byte {
	return encodeArray(currentFields(id, digest, kind, summary)...)
}

// withField is an entry in the current form with field position replaced by value.
func withField(id ID, digest [32]byte, kind uint64, summary []byte, position int, value []byte) []byte {
	fields := currentFields(id, digest, kind, summary)
	fields[position] = value
	return encodeArray(fields...)
}

// manifestElement is a current-form credential entry with the given revision, label, detail and membership.
func manifestElement(id ID, revision uint64, label, detail string, digest [32]byte, groups []byte) []byte {
	fields := currentFields(id, digest, uint64(KindCredential), encodeArray())
	fields[fieldRevision] = encodeUint(revision)
	fields[fieldLabel] = encodeBytes([]byte(label))
	fields[fieldDetail] = encodeBytes([]byte(detail))
	fields[fieldGroups] = groups
	return encodeArray(fields...)
}

// kindedElement is an entry in the current form with the given label, kind and expiry.
func kindedElement(id ID, label string, digest [32]byte, kind uint64, expiresOn string) []byte {
	fields := currentFields(id, digest, kind, encodeArray())
	fields[fieldLabel] = encodeBytes([]byte(label))
	fields[fieldExpiresOn] = encodeBytes([]byte(expiresOn))
	return encodeArray(fields...)
}

// thumbnailElement is an entry in the current form with the given thumbnail.
func thumbnailElement(id ID, digest [32]byte, kind uint64, thumbnail []byte) []byte {
	return withField(id, digest, kind, encodeArray(), fieldThumbnail, encodeBytes(thumbnail))
}

// siteElement is a current-form entry with the given site, also a credential's only site.
func siteElement(id ID, digest [32]byte, kind uint64, site string) []byte {
	fields := currentFields(id, digest, kind, encodeArray())
	fields[fieldSite] = encodeBytes([]byte(site))
	if kind == uint64(KindCredential) && site != "" {
		fields[fieldSites] = encodeArray(encodeBytes([]byte(site)))
	}
	return encodeArray(fields...)
}

// emailElement is an entry in the current form with the given email.
func emailElement(id ID, digest [32]byte, kind uint64, email string) []byte {
	return withField(id, digest, kind, encodeArray(), fieldEmail, encodeBytes([]byte(email)))
}

// scanElement is an attachment entry in the current form.
func scanElement(id ID, digest [32]byte, name, mediaType string, pinned uint64, thumbnail, owner []byte) []byte {
	fields := currentFields(id, digest, uint64(KindAttachment), encodeArray())
	fields[fieldLabel] = encodeBytes([]byte(name))
	fields[fieldDetail] = encodeBytes([]byte(mediaType))
	fields[fieldPinned] = encodeUint(pinned)
	fields[fieldThumbnail] = encodeBytes(thumbnail)
	fields[fieldOwner] = encodeBytes(owner)
	return encodeArray(fields...)
}

func TestIndexHoldsAttachmentsOnlyUnderAnIdentity(t *testing.T) {
	records := stubRecords(3)
	digests := withDigests(make([]entryMeta, 3), records)
	identity, credential, scan := ID{1}, ID{2}, ID{3}
	identityElement := kindedElement(identity, "Me", digests[0].digest, uint64(KindIdentity), "")
	credentialElement := siteElement(credential, digests[1].digest, uint64(KindCredential), "")
	thumbnail := []byte{0xff, 0xd8, 0xff, 0xd9}
	valid := scanElement(scan, digests[2].digest, "passport.jpg", MediaJPEG, 0, thumbnail, identity[:])
	_, _, index, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), identityElement, credentialElement, valid), records)
	entries := index.entries
	if err != nil {
		t.Fatal(err)
	}
	if entries[2].kind != KindAttachment || entries[2].owner != identity || entries[2].detail != MediaJPEG {
		t.Fatalf("attachment entry = %+v", entries[2])
	}
	tests := []struct {
		name    string
		element []byte
	}{
		{"attachment without an owner", scanElement(scan, digests[2].digest, "scan.pdf", MediaPDF, 0, nil, nil)},
		{"attachment of an identity the index does not hold", scanElement(scan, digests[2].digest, "scan.pdf", MediaPDF, 0, nil, bytes.Repeat([]byte{9}, 16))},
		{"attachment of a credential", scanElement(scan, digests[2].digest, "scan.pdf", MediaPDF, 0, nil, credential[:])},
		{"pinned attachment", scanElement(scan, digests[2].digest, "scan.pdf", MediaPDF, 1, nil, identity[:])},
		{"attachment of another media type", scanElement(scan, digests[2].digest, "scan.gif", "image/gif", 0, thumbnail, identity[:])},
		{"picture without a thumbnail", scanElement(scan, digests[2].digest, "scan.jpg", MediaJPEG, 0, nil, identity[:])},
		{"PDF with a thumbnail", scanElement(scan, digests[2].digest, "scan.pdf", MediaPDF, 0, thumbnail, identity[:])},
		{"thumbnail over its budget", scanElement(scan, digests[2].digest, "scan.jpg", MediaJPEG, 0, make([]byte, scanRule.thumbnailBudget+1), identity[:])},
		{"name with a folder", scanElement(scan, digests[2].digest, "folder/scan.pdf", MediaPDF, 0, nil, identity[:])},
		{"empty name", scanElement(scan, digests[2].digest, " ", MediaPDF, 0, nil, identity[:])},
		{"name over its limit", scanElement(scan, digests[2].digest, strings.Repeat("a", MaxScanNameLength+1), MediaPDF, 0, nil, identity[:])},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plaintext := indexPlaintext(1, [32]byte{}, encodeArray(), identityElement, credentialElement, test.element)
			if _, _, _, err := parseIndex(plaintext, records); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
	withOwner := withField(identity, digests[0].digest, uint64(KindIdentity), encodeArray(), fieldOwner, encodeBytes(scan[:]))
	if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), withOwner, credentialElement, valid), records); !errors.Is(err, ErrMalformed) {
		t.Fatalf("an identity with an owner: got %v", err)
	}
}

func TestIndexRejectsAnEntryOfAnyOtherFieldCount(t *testing.T) {
	records := stubRecords(1)
	digest := sha256.Sum256(encodeBox(records[0]))
	fields := currentFields(ID{5}, digest, uint64(KindCredential), encodeArray())
	if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), encodeArray(fields...)), records); err != nil {
		t.Fatalf("an entry of %d fields: %v", len(fields), err)
	}
	for count := range len(fields) + 2 {
		if count == len(fields) {
			continue
		}
		element := encodeArray(append(append([][]byte(nil), fields...), encodeUint(0))[:count]...)
		if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), element), records); !errors.Is(err, ErrMalformed) {
			t.Fatalf("an entry of %d fields: got %v, want ErrMalformed", count, err)
		}
	}
}

func TestVaultWithAnEntryMissingItsLastFieldsIsMalformed(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "Mail", Websites: []string{"https://mail.example/"}, Email: "user@example.test", Password: "one"})
	session := created.Session
	defer session.Lock()
	raw, head := currentRaw(t, session)
	entry := session.entries[0]
	fields := currentFields(entry.id, entry.digest, uint64(KindCredential), encodeArray())
	fields[fieldSite] = encodeBytes([]byte(entry.site))
	fields[fieldSites] = encodeArray(encodeBytes([]byte(entry.site)))
	fields[fieldEmail] = encodeBytes([]byte(entry.email))
	for _, count := range []int{len(fields) - 2, len(fields) - 1} {
		truncated := withIndex(t, session, raw, indexPlaintext(head.Revision, head.PreviousHash, encodeArray(), encodeArray(fields[:count]...)))
		if _, err := OpenWithRecovery(truncated, created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
			t.Fatalf("an entry of %d fields: got %v, want ErrMalformed", count, err)
		}
	}
}

func TestEditingACredentialEmailUpdatesItsEntry(t *testing.T) {
	created, ids := vaultWith(t,
		CredentialInput{Label: "Mail", Email: "old@example.test", Password: "one"},
	)
	session := created.Session
	defer session.Lock()
	if entry := listedEntry(t, session, ids[0]); entry.Email != "old@example.test" {
		t.Fatalf("created email = %q", entry.Email)
	}
	email := "user@example.test"
	pending, err := session.PrepareEdit(ids[0], CredentialPatch{Email: &email})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if entry := listedEntry(t, session, ids[0]); entry.Email != email {
		t.Fatalf("edited email = %q", entry.Email)
	}
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if entry := listedEntry(t, reopened, ids[0]); entry.Email != email {
		t.Fatalf("email across a reopen = %q", entry.Email)
	}
}

func indexPlaintext(revision uint64, previous [32]byte, table []byte, elements ...[]byte) []byte {
	return encodeArray(encodeUint(revision), encodeBytes(previous[:]), encodeArray(elements...), table, encodeArray())
}

func TestIndexRoundTripsDetailKindAndPinned(t *testing.T) {
	ancestors := ancestry{{9}, {8}, {7}}
	tests := []struct {
		name    string
		entries []entryMeta
	}{
		{"no entries", nil},
		{"empty detail unpinned", []entryMeta{{id: ID{1}, revision: 1, label: "One", detail: "", pinned: false, kind: KindCredential}}},
		{"detail pinned", []entryMeta{{id: ID{2}, revision: 4, label: "Two", detail: "alex@example.test", pinned: true, kind: KindCredential}}},
		{"mixed", []entryMeta{
			{id: ID{3}, revision: 1, label: "Three", detail: "  Ünïcødé  ", pinned: true, kind: KindCredential},
			{id: ID{4}, revision: 2, label: "", detail: "", pinned: false, kind: KindCredential},
		}},
		{"identities with and without an expiry", []entryMeta{
			{id: ID{5}, revision: 3, label: "Me", detail: "me@example.test", kind: KindIdentity, expiresOn: "2031-07-15", thumbnail: []byte{0xff, 0xd8, 0xff, 0xd9}},
			{id: ID{6}, revision: 1, label: "Partner", kind: KindIdentity},
			{id: ID{7}, revision: 1, label: "Mail", detail: "alex", kind: KindCredential},
		}},
		{"credentials with and without a site", []entryMeta{
			{id: ID{8}, revision: 1, label: "Google", kind: KindCredential, site: "google.com", sites: []string{"google.com"}},
			{id: ID{9}, revision: 1, label: "Longest", kind: KindCredential, site: strings.Repeat("a", MaxOriginLength), sites: []string{strings.Repeat("a", MaxOriginLength)}},
			{id: ID{10}, revision: 1, label: "Local", kind: KindCredential},
		}},
		{"credentials with several sites", []entryMeta{
			{id: ID{21}, revision: 2, label: "Example", kind: KindCredential, site: "example.com", sites: []string{"example.com", "example.org", "login.example.net"}},
			{id: ID{22}, revision: 1, label: "Everywhere", kind: KindCredential, site: "site0.example", sites: numberedSites(MaxCredentialWebsites)},
		}},
		{"credentials with and without a code", []entryMeta{
			{id: ID{23}, revision: 1, label: "Coded", kind: KindCredential, site: "example.com", sites: []string{"example.com"}, code: CodeFace{Digits: 6, Period: 30}},
			{id: ID{24}, revision: 3, label: "Bare", kind: KindCredential, site: "example.com", sites: []string{"example.com"}},
			{id: ID{25}, revision: 2, label: "Longest", kind: KindCredential, code: CodeFace{Digits: maxCodeDigits, Period: maxCodePeriod}},
		}},
		{"credentials with and without an email", []entryMeta{
			{id: ID{11}, revision: 1, label: "Mail", detail: "alex", kind: KindCredential, email: "user@example.test"},
			{id: ID{12}, revision: 1, label: "Longest", kind: KindCredential, email: strings.Repeat("a", MaxEmailLength)},
			{id: ID{13}, revision: 1, label: "None", kind: KindCredential},
		}},
		{"cards with and without optional values", []entryMeta{
			{id: ID{14}, revision: 2, label: "Everyday", detail: "Example Bank", kind: KindCard, expiresOn: "2029-08-31", site: "bank.example", card: CardFace{Network: NetworkNaranja, LastFour: "6789", Color: "#00a0e1"}},
			{id: ID{15}, revision: 1, label: "Spare", kind: KindCard, card: CardFace{LastFour: "0000"}},
		}},
		{"notes shown and hidden", []entryMeta{
			{id: ID{16}, revision: 1, label: "Shopping", detail: "Milk", kind: KindNote},
			{id: ID{17}, revision: 3, label: "Diary", kind: KindNote, note: NoteFace{Hidden: true}},
		}},
		{"seeds of every format", []entryMeta{
			{id: ID{18}, revision: 1, label: "Cold", detail: "Ledger", kind: KindSeed, seed: SeedFace{Format: SeedPhrase, Total: MaxSeedWords}},
			{id: ID{19}, revision: 1, label: "Key", kind: KindSeed, seed: SeedFace{Format: SeedPrivateKey}},
			{id: ID{20}, revision: 2, label: "Codes", kind: KindSeed, seed: SeedFace{Format: SeedBackupCodes, Total: MaxBackupCodes, Used: MaxBackupCodes}},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records := stubRecords(len(test.entries))
			entries := withDigests(append([]entryMeta(nil), test.entries...), records)
			plaintext, err := encodeIndex(7, ancestors, entries, nil, DefaultTrashRetention)
			if err != nil {
				t.Fatal(err)
			}
			revision, gotAncestors, index, err := parseIndex(plaintext, records)
			if err != nil {
				t.Fatal(err)
			}
			gotEntries, gotGroups := index.entries, index.groups
			if revision != 7 || !reflect.DeepEqual(gotAncestors, ancestors) {
				t.Fatalf("head fields changed: revision %d, ancestry %x", revision, gotAncestors)
			}
			if len(gotGroups) != 0 {
				t.Fatalf("unexpected groups: %+v", gotGroups)
			}
			if len(entries) == 0 {
				if len(gotEntries) != 0 {
					t.Fatalf("unexpected entries: %+v", gotEntries)
				}
				return
			}
			if !reflect.DeepEqual(gotEntries, entries) {
				t.Fatalf("entries changed: %+v", gotEntries)
			}
		})
	}
}

func TestIndexRoundTripsGroupsAndMembership(t *testing.T) {
	work := Group{ID: ID{1}, Name: "Work"}
	home := Group{ID: ID{2}, Name: "Home"}
	travel := Group{ID: ID{3}, Name: "Travel"}
	ancestors := ancestry{{5}}
	tests := []struct {
		name    string
		entries []entryMeta
		groups  []Group
	}{
		{"no groups at all", []entryMeta{{id: ID{9}, revision: 1, label: "Alone", kind: KindCredential}}, nil},
		{"groups without members", []entryMeta{{id: ID{9}, revision: 1, label: "Alone", kind: KindCredential}}, []Group{work, home}},
		{"one entry in several groups", []entryMeta{
			{id: ID{9}, revision: 2, label: "Everywhere", groups: []ID{work.ID, home.ID, travel.ID}, kind: KindCredential},
		}, []Group{work, home, travel}},
		{"a member and a loose entry", []entryMeta{
			{id: ID{9}, revision: 1, label: "Member", detail: "alex", groups: []ID{home.ID}, kind: KindCredential},
			{id: ID{10}, revision: 3, label: "Loose", pinned: true, kind: KindIdentity},
		}, []Group{work, home}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records := stubRecords(len(test.entries))
			entries := withDigests(append([]entryMeta(nil), test.entries...), records)
			plaintext, err := encodeIndex(11, ancestors, entries, test.groups, DefaultTrashRetention)
			if err != nil {
				t.Fatal(err)
			}
			revision, gotAncestors, index, err := parseIndex(plaintext, records)
			if err != nil {
				t.Fatal(err)
			}
			gotEntries, gotGroups := index.entries, index.groups
			if revision != 11 || !reflect.DeepEqual(gotAncestors, ancestors) {
				t.Fatalf("head fields changed: revision %d, ancestry %x", revision, gotAncestors)
			}
			if !reflect.DeepEqual(gotEntries, entries) {
				t.Fatalf("entries changed: %+v", gotEntries)
			}
			if !reflect.DeepEqual(gotGroups, test.groups) {
				t.Fatalf("groups changed: %+v", gotGroups)
			}
		})
	}
}

func TestIndexRejectsMalformedManifestElements(t *testing.T) {
	records := stubRecords(1)
	digest := sha256.Sum256(encodeBox(records[0]))
	id := ID{5}
	credential, identity := uint64(KindCredential), uint64(KindIdentity)
	work := Group{ID: ID{1}, Name: "Work"}
	home := Group{ID: ID{2}, Name: "Home"}
	table := groupTable(work, home)
	none := encodeArray()
	crowded := make([]ID, MaxCredentialGroups+1)
	for i := range crowded {
		crowded[i] = ID{byte(i + 1)}
	}
	tests := []struct {
		name    string
		element []byte
		want    error
	}{
		{"pinned two", withField(id, digest, credential, none, fieldPinned, encodeUint(2)), ErrMalformed},
		{"pinned large", withField(id, digest, credential, none, fieldPinned, encodeUint(255)), ErrMalformed},
		{"membership unsorted", manifestElement(id, 1, "Label", "alex", digest, membership(home.ID, work.ID)), ErrMalformed},
		{"membership repeats an id", manifestElement(id, 1, "Label", "alex", digest, membership(work.ID, work.ID)), ErrMalformed},
		{"membership over the limit", manifestElement(id, 1, "Label", "alex", digest, membership(crowded...)), ErrResourceLimit},
		{"thumbnail that is not bytes", withField(id, digest, credential, none, fieldThumbnail, encodeUint(0)), ErrMalformed},
		{"card field that is not an array", withField(id, digest, identity, none, fieldCard, encodeBytes(nil)), ErrMalformed},
		{"summary that is not an array", withField(id, digest, identity, none, fieldSummary, encodeBytes(nil)), ErrMalformed},
		{"passkeys that are not an array", withField(id, digest, identity, none, fieldPasskeys, encodeBytes(nil)), ErrMalformed},
		{"digits that are not a number", withField(id, digest, credential, none, fieldDigits, encodeBytes(nil)), ErrMalformed},
		{"period that is not a number", withField(id, digest, credential, none, fieldPeriod, encodeBytes(nil)), ErrMalformed},
		{"too few digits", codeElement(id, digest, credential, none, minCodeDigits-1, 30), ErrMalformed},
		{"too many digits", codeElement(id, digest, credential, none, maxCodeDigits+1, 30), ErrMalformed},
		{"digits without a period", codeElement(id, digest, credential, none, 6, 0), ErrMalformed},
		{"a period without digits", codeElement(id, digest, credential, none, 0, 30), ErrMalformed},
		{"period over its limit", codeElement(id, digest, credential, none, 6, maxCodePeriod+1), ErrMalformed},
		{"huge period", codeElement(id, digest, credential, none, 6, 1<<63), ErrMalformed},
		{"identity with a code", codeElement(id, digest, identity, none, 6, 30), ErrMalformed},
		{"sites that are not an array", withField(id, digest, credential, none, fieldSites, encodeBytes(nil)), ErrMalformed},
		{"identity with sites", sitesElement(id, digest, identity, "", "example.com"), ErrMalformed},
		{"credential with a site and no sites", sitesElement(id, digest, credential, "example.com"), ErrMalformed},
		{"credential with sites and no site", sitesElement(id, digest, credential, "", "example.com"), ErrMalformed},
		{"sites that do not start with the site", sitesElement(id, digest, credential, "example.com", "example.org", "example.com"), ErrMalformed},
		{"sites that repeat one", sitesElement(id, digest, credential, "example.com", "example.com", "example.org", "example.com"), ErrMalformed},
		{"sites with an empty one", sitesElement(id, digest, credential, "example.com", "example.com", ""), ErrMalformed},
		{"sites with one over its limit", sitesElement(id, digest, credential, "example.com", "example.com", strings.Repeat("a", MaxOriginLength+1)), ErrMalformed},
		{"sites that are not text", sitesElement(id, digest, credential, "example.com", "example.com", "\xff"), ErrMalformed},
		{"more sites than websites", sitesElement(id, digest, credential, "site0.example", numberedSites(MaxCredentialWebsites+1)...), ErrMalformed},
		{"email that is not bytes", withField(id, digest, credential, none, fieldEmail, encodeUint(0)), ErrMalformed},
		{"identity with an email", emailElement(id, digest, identity, "me@example.test"), ErrMalformed},
		{"email over its limit", emailElement(id, digest, credential, strings.Repeat("a", MaxEmailLength+1)), ErrMalformed},
		{"email that is not text", emailElement(id, digest, credential, "\xff"), ErrMalformed},
		{"site that is not bytes", withField(id, digest, credential, none, fieldSite, encodeUint(0)), ErrMalformed},
		{"identity with a site", siteElement(id, digest, identity, "example.com"), ErrMalformed},
		{"site over its limit", siteElement(id, digest, credential, strings.Repeat("a", MaxOriginLength+1)), ErrMalformed},
		{"site that is not text", siteElement(id, digest, credential, "\xff"), ErrMalformed},
		{"credential with a thumbnail", thumbnailElement(id, digest, credential, []byte{0xff, 0xd8}), ErrMalformed},
		{"thumbnail over its budget", thumbnailElement(id, digest, identity, make([]byte, maxThumbnailBytes+1)), ErrMalformed},
		{"kind zero", kindedElement(id, "Label", digest, 0, ""), ErrUnsupported},
		{"kind seven", kindedElement(id, "Label", digest, 7, ""), ErrUnsupported},
		{"kind past one byte", kindedElement(id, "Label", digest, 257, ""), ErrUnsupported},
		{"credential with an expiry", kindedElement(id, "Label", digest, credential, "2030-01-01"), ErrMalformed},
		{"identity with an impossible expiry", kindedElement(id, "Label", digest, identity, "2026-02-30"), ErrMalformed},
		{"identity with an expiry in another form", kindedElement(id, "Label", digest, identity, "2026-2-3"), ErrMalformed},
		{"identity with an expiry before 1900", kindedElement(id, "Label", digest, identity, "0000-01-01"), ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, table, test.element), records); !errors.Is(err, test.want) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestIndexRejectsInconsistentGroups(t *testing.T) {
	records := stubRecords(1)
	digest := sha256.Sum256(encodeBox(records[0]))
	id := ID{7}
	work := Group{ID: ID{1}, Name: "Work"}
	member := manifestElement(id, 1, "Label", "alex", digest, membership(work.ID))
	loose := manifestElement(id, 1, "Label", "alex", digest, encodeArray())
	tests := []struct {
		name    string
		element []byte
		table   []byte
	}{
		{"membership names a group the table does not hold", member, groupTable(Group{ID: ID{2}, Name: "Home"})},
		{"two groups share an identifier", loose, groupTable(work, Group{ID: work.ID, Name: "Home"})},
		{"two names differ only by case", loose, groupTable(work, Group{ID: ID{2}, Name: "WORK"})},
		{"a name carries space around it", loose, groupTable(Group{ID: ID{2}, Name: " Work "})},
		{"an empty name", loose, groupTable(work, Group{ID: ID{2}, Name: ""})},
		{"a whitespace name", loose, groupTable(work, Group{ID: ID{2}, Name: " \t "})},
		{"a name over the limit", loose, groupTable(Group{ID: ID{2}, Name: strings.Repeat("ü", MaxGroupNameLength+1)})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, test.table, test.element), records); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestContainerVersionRejection(t *testing.T) {
	created, err := Create()
	if err != nil {
		t.Fatal(err)
	}
	defer created.Session.Lock()
	tests := []struct {
		name   string
		mutate func([]byte)
		want   error
	}{
		{"version 0x0002", func(input []byte) { input[9] = 2 }, ErrUnsupported},
		{"version 0x0001", func(input []byte) { input[9] = 1 }, ErrUnsupported},
		{"version 0x0100", func(input []byte) { input[8], input[9] = 1, 0 }, ErrUnsupported},
		{"wrong magic", func(input []byte) { input[0] ^= 1 }, ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modified := append([]byte(nil), created.Container...)
			test.mutate(modified)
			if _, err := parseContainer(modified); !errors.Is(err, test.want) {
				t.Fatalf("got %v", err)
			}
			if _, err := OpenWithRecovery(modified, created.RecoveryPhrase); !errors.Is(err, test.want) {
				t.Fatalf("open got %v", err)
			}
		})
	}
}

func TestPrepareSetPinnedKeepsRecordSealed(t *testing.T) {
	tests := []struct {
		name   string
		steps  []bool
		expect bool
	}{
		{"pin", []bool{true}, true},
		{"pin then unpin", []bool{true, false}, false},
		{"repeat pin", []bool{true, true}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, id := populatedSession(t)
			defer session.Lock()
			before, head, err := session.CurrentContainer()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := parseContainer(before)
			if err != nil {
				t.Fatal(err)
			}
			sealedRecord := encodeBox(raw.records[0])
			for step, pinned := range test.steps {
				pending, err := session.PrepareSetPinned(id, pinned)
				if err != nil {
					t.Fatalf("step %d: %v", step, err)
				}
				if err := session.Commit(pending); err != nil {
					t.Fatalf("step %d: %v", step, err)
				}
			}
			after, nextHead, err := session.CurrentContainer()
			if err != nil {
				t.Fatal(err)
			}
			if nextHead.Revision != head.Revision+uint64(len(test.steps)) {
				t.Fatalf("revision did not advance: %d then %d", head.Revision, nextHead.Revision)
			}
			nextRaw, err := parseContainer(after)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encodeBox(nextRaw.records[0]), sealedRecord) {
				t.Fatal("record was re-encrypted")
			}
			entries, err := session.List()
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Pinned != test.expect {
				t.Fatalf("pin flag not listed: %+v", entries)
			}
			if value := selectCredential(t, session, id); value.Password != "secret" {
				t.Fatalf("credential unreadable after pin change: %+v", value)
			}
		})
	}
}

func TestPrepareSetPinnedRejectsUnknownEntry(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	if _, err := session.PrepareSetPinned(ID{0xff}, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}
