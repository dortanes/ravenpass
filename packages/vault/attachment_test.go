package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"image/jpeg"
	"reflect"
	"testing"
)

var tinyPDF = []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF\n")

func scannedIdentity() IdentityInput {
	return IdentityInput{
		Label: "Alex",
		Documents: []Document{
			{Type: DocumentPassport, Number: "X1234567"},
			{Type: DocumentIDCard, Number: "ID-99"},
		},
	}
}

func prepared(t *testing.T, name string, given []byte) PreparedScan {
	t.Helper()
	scan, err := PrepareScan(name, given)
	if err != nil {
		t.Fatal(err)
	}
	return scan
}

// storedIdentity reads an identity as an editor would start from it.
func storedIdentity(t *testing.T, session *Session, id ID) IdentityInput {
	t.Helper()
	identity := selectIdentity(t, session, id).IdentityInput
	identity.Documents = append([]Document(nil), identity.Documents...)
	return identity
}

func commitEdit(t *testing.T, session *Session, id ID, input IdentityInput) {
	t.Helper()
	pending, err := session.PrepareEditIdentity(id, input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
}

// attachScan adds one scan to a document through an edit and reports its new identifier.
func attachScan(t *testing.T, session *Session, owner ID, document int, name string, given []byte) ID {
	t.Helper()
	input := storedIdentity(t, session, owner)
	input.Documents[document].Attach = []PreparedScan{prepared(t, name, given)}
	commitEdit(t, session, owner, input)
	scans := selectIdentity(t, session, owner).Documents[document].Scans
	return scans[len(scans)-1]
}

func scanSession(t *testing.T) (*Session, ID) {
	t.Helper()
	session, _ := populatedSession(t)
	t.Cleanup(session.Lock)
	return session, commitIdentity(t, session, scannedIdentity(), nil)
}

func assertUnchanged(t *testing.T, session *Session, before []byte, head Head) {
	t.Helper()
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused change altered the vault")
	}
}

func TestPreparedPictureIsStoredAsBareJPEGWithThumbnail(t *testing.T) {
	session, owner := scanSession(t)
	scan := prepared(t, "passport.png", pngBytes(t, patternPicture(300, 200)))
	if scan.Name() != "passport.png" || scan.MediaType() != MediaJPEG {
		t.Fatalf("prepared scan is %s as %s", scan.Name(), scan.MediaType())
	}
	_, head := currentRaw(t, session)
	input := storedIdentity(t, session, owner)
	input.Documents[1].Attach = []PreparedScan{scan}
	commitEdit(t, session, owner, input)
	if _, next := currentRaw(t, session); next.Revision != head.Revision+1 {
		t.Fatalf("attaching took %d saves", next.Revision-head.Revision)
	}
	identity := selectIdentity(t, session, owner)
	if len(identity.Documents[1].Scans) != 1 || identity.Documents[0].Scans != nil {
		t.Fatalf("documents after attaching = %+v", identity.Documents)
	}
	id := identity.Documents[1].Scans[0]
	stored, err := session.ReadScan(id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != "passport.png" || stored.MediaType != MediaJPEG || !bytes.Equal(stored.Thumbnail, scan.Thumbnail()) {
		t.Fatalf("stored scan = %+v", stored.ScanSummary)
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(stored.Content))
	if err != nil || config.Width != 300 || config.Height != 200 {
		t.Fatalf("stored scan is %dx%d, error = %v", config.Width, config.Height, err)
	}
	for _, marker := range markersBeforeScan(t, stored.Content) {
		if marker >= 0xe0 && marker <= 0xef || marker == 0xfe {
			t.Fatalf("stored scan carries metadata segment %#x", marker)
		}
	}
	thumbnail, err := jpeg.DecodeConfig(bytes.NewReader(stored.Thumbnail))
	if err != nil || thumbnail.Width != 160 || thumbnail.Height != 106 {
		t.Fatalf("thumbnail is %dx%d, error = %v", thumbnail.Width, thumbnail.Height, err)
	}
	if scans, err := session.ScansOf(owner); err != nil || len(scans) != 1 || scans[0].ID != id {
		t.Fatalf("scans = %+v, error = %v", scans, err)
	}
	entries, err := session.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == id {
			t.Fatal("an attachment was listed")
		}
	}
}

func TestPreparedPDFIsStoredAsGiven(t *testing.T) {
	session, owner := scanSession(t)
	id := attachScan(t, session, owner, 0, "tax.pdf", tinyPDF)
	scan, err := session.ReadScan(id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(scan.Content, tinyPDF) || scan.MediaType != MediaPDF || scan.Thumbnail != nil {
		t.Fatalf("scan = %+v", scan.ScanSummary)
	}
}

func TestPrepareScanRefusesWhatTheRulesDoNotAllow(t *testing.T) {
	largePDF := append(append([]byte("%PDF-1.4\n"), make([]byte, maxScanBytes)...), "%%EOF\n"...)
	tests := []struct {
		name  string
		file  string
		given []byte
		want  error
	}{
		{"PDF without its end marker", "fake.pdf", []byte("%PDF-1.4\nnot really"), ErrInvalidInput},
		{"text", "notes.pdf", []byte("plain text"), ErrInvalidInput},
		{"JPEG handed to the core", "scan.jpg", jpegBytes(t, patternPicture(64, 64)), ErrInvalidInput},
		{"picture over its side", "wide.png", pngBytes(t, patternPicture(ScanSide+1, 8)), ErrInvalidInput},
		{"PDF over its budget", "large.pdf", largePDF, ErrResourceLimit},
		{"name with a folder", "a/scan.pdf", tinyPDF, ErrInvalidInput},
		{"empty name", " ", tinyPDF, ErrInvalidInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := PrepareScan(test.file, test.given); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestNewIdentityTakesItsScansInOneSave(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	_, head := currentRaw(t, session)
	input := scannedIdentity()
	input.Documents[0].Attach = []PreparedScan{prepared(t, "front.png", pngBytes(t, patternPicture(40, 30))), prepared(t, "back.pdf", tinyPDF)}
	id := commitIdentity(t, session, input, nil)
	if _, next := currentRaw(t, session); next.Revision != head.Revision+1 {
		t.Fatalf("a new identity with scans took %d saves", next.Revision-head.Revision)
	}
	scans := selectIdentity(t, session, id).Documents[0].Scans
	if len(scans) != 2 {
		t.Fatalf("scans on the new document = %v", scans)
	}
	back, err := session.ReadScan(scans[1])
	if err != nil || back.Name != "back.pdf" || !bytes.Equal(back.Content, tinyPDF) {
		t.Fatalf("second scan = %+v, error = %v", back.ScanSummary, err)
	}
	if err := session.VerifyAll(); err != nil {
		t.Fatal(err)
	}
}

func TestScanLimitCountsStoredAndNewScans(t *testing.T) {
	session, owner := scanSession(t)
	for range MaxDocumentScans - 1 {
		attachScan(t, session, owner, 1, "id.pdf", tinyPDF)
	}
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	input := storedIdentity(t, session, owner)
	input.Documents[1].Attach = []PreparedScan{prepared(t, "fourth.pdf", tinyPDF), prepared(t, "fifth.pdf", tinyPDF)}
	if _, err := session.PrepareEditIdentity(owner, input, nil); !errors.Is(err, ErrScanLimit) {
		t.Fatalf("a fifth scan beside stored ones: %v", err)
	}
	crowded := scannedIdentity()
	crowded.Documents[0].Attach = make([]PreparedScan, MaxDocumentScans+1)
	if _, _, err := session.PrepareCreateIdentity(crowded, nil); !errors.Is(err, ErrScanLimit) {
		t.Fatalf("a new document with five scans: %v", err)
	}
	unprepared := storedIdentity(t, session, owner)
	unprepared.Documents[0].Attach = []PreparedScan{{}}
	if _, err := session.PrepareEditIdentity(owner, unprepared, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("a scan the core did not prepare: %v", err)
	}
	assertUnchanged(t, session, before, head)
	input.Documents[1].Attach = input.Documents[1].Attach[:1]
	commitEdit(t, session, owner, input)
	if scans := selectIdentity(t, session, owner).Documents[1].Scans; len(scans) != MaxDocumentScans {
		t.Fatalf("scans at the limit = %d", len(scans))
	}
}

func TestEditingAnIdentityKeepsOrRemovesItsScans(t *testing.T) {
	session, owner := scanSession(t)
	passport := attachScan(t, session, owner, 0, "passport.pdf", tinyPDF)
	card := attachScan(t, session, owner, 1, "card.png", pngBytes(t, patternPicture(40, 40)))
	other := commitIdentity(t, session, scannedIdentity(), nil)
	foreign := attachScan(t, session, other, 0, "other.pdf", tinyPDF)
	raw, _ := currentRaw(t, session)
	sealedCard := encodeBox(raw.records[session.find(card)])

	read := storedIdentity(t, session, owner)
	refused := map[string]func(*IdentityInput){
		"another identity's scan": func(input *IdentityInput) { input.Documents[0].Scans = []ID{passport, foreign} },
		"one scan named twice":    func(input *IdentityInput) { input.Documents[0].Scans = []ID{passport, card} },
		"an identity as a scan":   func(input *IdentityInput) { input.Documents[0].Scans = []ID{other} },
	}
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range refused {
		t.Run(name, func(t *testing.T) {
			input := read
			input.Documents = append([]Document(nil), read.Documents...)
			mutate(&input)
			if _, err := session.PrepareEditIdentity(owner, input, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("got %v, want ErrInvalidInput", err)
			}
		})
	}
	named := scannedIdentity()
	named.Documents[0].Scans = []ID{foreign}
	if _, _, err := session.PrepareCreateIdentity(named, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("a new identity naming a stored scan: %v", err)
	}
	assertUnchanged(t, session, before, head)

	renamed := read
	renamed.Label = "Alex renamed"
	commitEdit(t, session, owner, renamed)
	raw, _ = currentRaw(t, session)
	if !bytes.Equal(encodeBox(raw.records[session.find(card)]), sealedCard) {
		t.Fatal("an edit that kept a scan sealed it again")
	}

	dropped := renamed
	dropped.Documents = []Document{read.Documents[1]}
	commitEdit(t, session, owner, dropped)
	if _, err := session.ReadScan(passport); !errors.Is(err, ErrNotFound) {
		t.Fatalf("the scan of a dropped document survived: %v", err)
	}
	if _, err := session.ReadScan(card); err != nil {
		t.Fatalf("the scan of a kept document was lost: %v", err)
	}
	if _, err := session.ReadScan(foreign); err != nil {
		t.Fatalf("another identity's scan was lost: %v", err)
	}

	removed := storedIdentity(t, session, owner)
	removed.Documents[0].Scans = nil
	commitEdit(t, session, owner, removed)
	if _, err := session.ReadScan(card); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a scan taken off its document survived: %v", err)
	}
	plaintext, err := session.openRecord(session.find(owner))
	if err != nil {
		t.Fatal(err)
	}
	if plaintext[1] != recordSchemaIdentity {
		t.Fatalf("an identity without scans was written as schema %d", plaintext[1])
	}
}

func TestDeletingAnIdentityRemovesItsScans(t *testing.T) {
	session, owner := scanSession(t)
	scan := attachScan(t, session, owner, 0, "passport.pdf", tinyPDF)
	for name, refuse := range map[string]func() error{
		"delete":    func() error { _, err := session.PrepareDelete(scan); return err },
		"pin":       func() error { _, err := session.PrepareSetPinned(scan, true); return err },
		"groups":    func() error { _, err := session.PrepareSetGroups(scan, nil); return err },
		"selection": func() error { _, err := session.BeginSelection(scan); return err },
	} {
		if err := refuse(); !errors.Is(err, ErrNotFound) {
			t.Fatalf("%s on an attachment: %v", name, err)
		}
	}
	pending, err := session.PrepareDelete(owner)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if session.find(scan) >= 0 {
		t.Fatal("a deleted identity's scan survived")
	}
	if err := session.VerifyAll(); err != nil {
		t.Fatal(err)
	}
}

func TestScansTravelThroughExportAndRecovery(t *testing.T) {
	created, _ := vaultWith(t)
	defer created.Session.Lock()
	owner := commitIdentity(t, created.Session, scannedIdentity(), nil)
	pdf := attachScan(t, created.Session, owner, 0, "tax.pdf", tinyPDF)
	picture := attachScan(t, created.Session, owner, 1, "card.png", pngBytes(t, noisePicture(64)))
	stored, err := created.Session.ReadScan(picture)
	if err != nil {
		t.Fatal(err)
	}
	exported, _, err := created.Session.Export()
	if err != nil {
		t.Fatal(err)
	}
	for _, window := range [][]byte{[]byte("/Type /Catalog"), []byte("card.png"), stored.Content[len(stored.Content)/2 : len(stored.Content)/2+32]} {
		if bytes.Contains(exported, window) {
			t.Fatalf("%q appears in the vault file", window)
		}
	}
	restored, err := OpenWithRecovery(exported, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Lock()
	for id, want := range map[ID][]byte{pdf: tinyPDF, picture: stored.Content} {
		scan, err := restored.ReadScan(id)
		if err != nil || !bytes.Equal(scan.Content, want) {
			t.Fatalf("scan %s after recovery: error = %v", id, err)
		}
	}
}

// resealIdentity returns the container with the identity's record resealed at its revision around documents.
func resealIdentity(t *testing.T, session *Session, owner ID, documents []Document) []byte {
	t.Helper()
	raw, head := currentRaw(t, session)
	index := session.find(owner)
	plaintext, err := session.openRecord(index)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := decodeIdentityRecord(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	identity.Documents = documents
	plaintext, err = encodeIdentityRecord(identity)
	if err != nil {
		t.Fatal(err)
	}
	box, err := seal(session.recordKey, plaintext, recordAAD(raw.vaultID, owner, session.entries[index].revision))
	if err != nil {
		t.Fatal(err)
	}
	records := append([]sealedBox(nil), raw.records...)
	entries := append([]entryMeta(nil), session.entries...)
	records[index] = box
	entries[index].digest = sha256.Sum256(encodeBox(box))
	indexPlaintext, err := encodeIndex(head.Revision, session.ancestry, entries, nil, DefaultTrashRetention)
	if err != nil {
		t.Fatal(err)
	}
	raw.records = records
	return withIndex(t, session, raw, indexPlaintext)
}

func TestScanReferencesMustMatchTheIndex(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	defer session.Lock()
	owner := commitIdentity(t, session, scannedIdentity(), nil)
	other := commitIdentity(t, session, scannedIdentity(), nil)
	scan := attachScan(t, session, owner, 0, "passport.pdf", tinyPDF)
	foreign := attachScan(t, session, other, 0, "other.pdf", tinyPDF)
	documents := func(first, second []ID) []Document {
		result := scannedIdentity().Documents
		result[0].Scans, result[1].Scans = first, second
		return result
	}
	tests := map[string][]Document{
		"a scan no document names": documents(nil, nil),
		"a scan named twice":       documents([]ID{scan}, []ID{scan}),
		"a name that is no item":   documents([]ID{scan, {0xee}}, nil),
		"another identity's scan":  documents([]ID{scan, foreign}, nil),
	}
	for name, documents := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := OpenWithRecovery(resealIdentity(t, session, owner, documents), created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestAttachmentRecordSchema(t *testing.T) {
	record := mustEncode(encodeAttachmentRecord(tinyPDF))
	if content, err := decodeAttachmentRecord(record, MediaPDF); err != nil || !bytes.Equal(content, tinyPDF) {
		t.Fatalf("attachment record = %q, error = %v", content, err)
	}
	identity, err := encodeIdentityRecord(IdentityInput{Documents: []Document{{Type: DocumentPassport, Number: "1", Scans: []ID{{1}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if identity[1] != recordSchemaIdentity {
		t.Fatalf("an identity with scans was written as schema %d", identity[1])
	}
	tests := []struct {
		name   string
		decode func() error
		want   error
	}{
		{"attachment read as an identity", func() error { return decodeAsIdentity(record) }, ErrMalformed},
		{"attachment read as a credential", func() error { return decodeAsCredential(record) }, ErrMalformed},
		{"identity with scans read as a credential", func() error { return decodeAsCredential(identity) }, ErrMalformed},
		{"identity read as an attachment", func() error { _, err := decodeAttachmentRecord(identity, MediaPDF); return err }, ErrMalformed},
		{"PDF content for a picture", func() error { _, err := decodeAttachmentRecord(record, MediaJPEG); return err }, ErrMalformed},
		{"unknown schema", func() error {
			_, err := decodeAttachmentRecord(encodeArray(encodeUint(15), encodeBytes(tinyPDF)), MediaPDF)
			return err
		}, ErrUnsupported},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode(); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
	decoded, err := decodeIdentityRecord(identity)
	if err != nil || !reflect.DeepEqual(decoded.Documents[0].Scans, []ID{{1}}) {
		t.Fatalf("schema 9 record = %+v, error = %v", decoded.Documents, err)
	}
}
