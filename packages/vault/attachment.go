package vault

import (
	"bytes"
	"strings"
)

// A scan is an attachment item holding a JPEG encoded from a PNG or a PDF as given; its entry's label is the file name, its detail the media type.
const (
	MaxDocumentScans  = 4
	MaxScanNameLength = 255
	ScanSide          = 2400
	MediaJPEG         = "image/jpeg"
	MediaPDF          = "application/pdf"
	maxScanBytes      = 960 << 10
	// A PDF carries its end marker within this many bytes of its end.
	pdfEndWindow = 1024
)

var (
	pdfSignature = []byte("%PDF-")
	pdfEnd       = []byte("%%EOF")
)

var scanRule = pictureRule{
	givenBytes:      64 << 20,
	accepts:         func(width, height int) bool { return max(width, height) <= ScanSide },
	budget:          maxScanBytes,
	thumbnailSide:   160,
	thumbnailBudget: 12 << 10,
}

// ScanSummary is what the index holds of a scan; Thumbnail is empty for a PDF.
type ScanSummary struct {
	ID        ID
	Name      string
	MediaType string
	Thumbnail []byte
}

// Scan is a scan with its content.
type Scan struct {
	ScanSummary
	Content []byte
}

// HasPDFSignature reports whether content starts as every PDF does; any other scan is a picture.
func HasPDFSignature(content []byte) bool {
	return bytes.HasPrefix(content, pdfSignature)
}

// validPDF checks the signature and end marker only; the core never parses a PDF.
func validPDF(content []byte) bool {
	return HasPDFSignature(content) && bytes.Contains(content[max(0, len(content)-pdfEndWindow):], pdfEnd)
}

// acceptScan returns the stored content of a given scan, its media type and a picture's thumbnail.
func acceptScan(given []byte) ([]byte, string, []byte, error) {
	if !HasPDFSignature(given) {
		content, thumbnail, err := scanRule.encode(given)
		return content, MediaJPEG, thumbnail, err
	}
	if len(given) > maxScanBytes {
		return nil, "", nil, ErrResourceLimit
	}
	if !validPDF(given) {
		return nil, "", nil, ErrInvalidInput
	}
	return bytes.Clone(given), MediaPDF, nil, nil
}

// acceptScanName takes a file name without its folder.
func acceptScanName(name string) bool {
	return fits(name, MaxScanNameLength) && strings.TrimSpace(name) != "" && !strings.ContainsAny(name, "/\x00")
}

// validScanEntry holds the index rules for an attachment entry that its record does not decide.
func validScanEntry(entry entryMeta) bool {
	if !acceptScanName(entry.label) || entry.pinned || len(entry.groups) != 0 || entry.expiresOn != "" || entry.site != "" {
		return false
	}
	switch entry.detail {
	case MediaJPEG:
		return len(entry.thumbnail) > 0 && len(entry.thumbnail) <= scanRule.thumbnailBudget
	case MediaPDF:
		return len(entry.thumbnail) == 0
	default:
		return false
	}
}

// An attachment record is [7, content].
const recordSchemaAttachment = 7

// attachmentHeadroom bounds the bytes an attachment record adds to its content.
const attachmentHeadroom = 16

type attachmentRecord struct {
	_       struct{} `cbor:",toarray"`
	Schema  uint64
	Content []byte
}

func encodeAttachmentRecord(content []byte) ([]byte, error) {
	return sealableRecord(marshal(attachmentRecord{Schema: recordSchemaAttachment, Content: content}, len(content)+attachmentHeadroom))
}

// decodeAttachmentRecord reads the content and checks it is what the entry's media type says.
func decodeAttachmentRecord(plaintext []byte, mediaType string) ([]byte, error) {
	if err := readSchema(plaintext, recordSchemaAttachment); err != nil {
		return nil, err
	}
	var record attachmentRecord
	if err := unmarshal(plaintext, &record); err != nil {
		return nil, err
	}
	content := record.Content
	if len(content) == 0 || len(content) > maxScanBytes {
		return nil, ErrMalformed
	}
	if mediaType == MediaPDF && !validPDF(content) || mediaType == MediaJPEG && !bytes.HasPrefix(content, []byte{0xff, 0xd8}) {
		return nil, ErrMalformed
	}
	return content, nil
}

// scansOf lists the attachment ids an identity's documents name.
func scansOf(documents []Document) []ID {
	var ids []ID
	for _, document := range documents {
		ids = append(ids, document.Scans...)
	}
	return ids
}

// acceptScanReferences checks that every scan the documents name is a distinct attachment of owner and reports the set named.
func (s *Session) acceptScanReferences(owner ID, documents []Document) (map[ID]struct{}, error) {
	named := make(map[ID]struct{})
	for _, id := range scansOf(documents) {
		index := s.findKind(id, KindAttachment)
		if index < 0 || s.entries[index].owner != owner {
			return nil, ErrInvalidInput
		}
		if _, repeated := named[id]; repeated {
			return nil, ErrInvalidInput
		}
		named[id] = struct{}{}
	}
	return named, nil
}

// verifyScanReferences checks that named, from attachment to owner, matches the attachments exactly.
func (s *Session) verifyScanReferences(named map[ID]ID) error {
	for _, entry := range s.entries {
		if entry.kind != KindAttachment {
			continue
		}
		owner, found := named[entry.id]
		if !found || owner != entry.owner {
			return ErrMalformed
		}
		delete(named, entry.id)
	}
	if len(named) != 0 {
		return ErrMalformed
	}
	return nil
}

// ScansOf reports the scans an identity holds, in the order they were attached.
func (s *Session) ScansOf(owner ID) ([]ScanSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	if s.findKind(owner, KindIdentity) < 0 {
		return nil, ErrNotFound
	}
	var scans []ScanSummary
	for _, entry := range s.entries {
		if entry.kind == KindAttachment && entry.owner == owner {
			scans = append(scans, ScanSummary{ID: entry.id, Name: entry.label, MediaType: entry.detail, Thumbnail: bytes.Clone(entry.thumbnail)})
		}
	}
	return scans, nil
}

// ReadScan decrypts one scan under the lock.
func (s *Session) ReadScan(id ID) (Scan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return Scan{}, ErrLocked
	}
	index := s.findKind(id, KindAttachment)
	// A scan of an identity in the trash stays with it until the identity returns.
	if index < 0 || s.findKind(s.entries[index].owner, KindIdentity) < 0 {
		return Scan{}, ErrNotFound
	}
	plaintext, err := s.openRecord(index)
	if err != nil {
		return Scan{}, err
	}
	defer clear(plaintext)
	entry := s.entries[index]
	content, err := decodeAttachmentRecord(plaintext, entry.detail)
	if err != nil {
		return Scan{}, err
	}
	return Scan{ScanSummary: ScanSummary{ID: entry.id, Name: entry.label, MediaType: entry.detail, Thumbnail: bytes.Clone(entry.thumbnail)}, Content: content}, nil
}

// PreparedScan is a scan in its stored form from PrepareScan; a save refuses the zero value.
type PreparedScan struct {
	name      string
	mediaType string
	content   []byte
	thumbnail []byte
}

// PrepareScan turns a PNG or a PDF named by its folder-less file name into its stored form without writing.
func PrepareScan(name string, given []byte) (PreparedScan, error) {
	if !acceptScanName(name) {
		return PreparedScan{}, ErrInvalidInput
	}
	content, mediaType, thumbnail, err := acceptScan(given)
	if err != nil {
		return PreparedScan{}, err
	}
	return PreparedScan{name: name, mediaType: mediaType, content: content, thumbnail: thumbnail}, nil
}

// Copy is the scan in its stored form, to attach to another document without converting it again.
func (s Scan) Copy() PreparedScan {
	return PreparedScan{name: s.Name, mediaType: s.MediaType, content: bytes.Clone(s.Content), thumbnail: bytes.Clone(s.Thumbnail)}
}

// Name is the scan's file name.
func (p PreparedScan) Name() string { return p.name }

// MediaType is MediaJPEG or MediaPDF.
func (p PreparedScan) MediaType() string { return p.mediaType }

// Thumbnail is a small JPEG for a picture and empty for a PDF.
func (p PreparedScan) Thumbnail() []byte { return bytes.Clone(p.thumbnail) }

// attachPrepared seals each scan the documents attach as an attachment of owner; it mutates the caller's document copies.
func (s *Session) attachPrepared(owner ID, documents []Document, reserved map[ID]struct{}) ([]entryMeta, []sealedBox, error) {
	var entries []entryMeta
	var records []sealedBox
	for i := range documents {
		for _, scan := range documents[i].Attach {
			if scan.mediaType == "" {
				return nil, nil, ErrInvalidInput
			}
			id, err := s.freshID(reserved)
			if err != nil {
				return nil, nil, err
			}
			reserved[id] = struct{}{}
			plaintext, err := encodeAttachmentRecord(scan.content)
			if err != nil {
				return nil, nil, err
			}
			entry, box, err := s.sealFirst(id, entryMeta{kind: KindAttachment, label: scan.name, detail: scan.mediaType, thumbnail: bytes.Clone(scan.thumbnail), owner: owner}, plaintext)
			clear(plaintext)
			if err != nil {
				return nil, nil, err
			}
			documents[i].Scans = append(documents[i].Scans, id)
			entries = append(entries, entry)
			records = append(records, box)
		}
		documents[i].Attach = nil
	}
	return entries, records, nil
}

// withoutItems drops the items drop selects, keeping entries and records aligned.
func withoutItems(entries []entryMeta, records []sealedBox, drop func(entryMeta) bool) ([]entryMeta, []sealedBox) {
	keptEntries := make([]entryMeta, 0, len(entries))
	keptRecords := make([]sealedBox, 0, len(records))
	for i, entry := range entries {
		if !drop(entry) {
			keptEntries = append(keptEntries, entry)
			keptRecords = append(keptRecords, records[i])
		}
	}
	return keptEntries, keptRecords
}
