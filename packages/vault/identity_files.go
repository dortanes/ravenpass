package vault

import "bytes"

// IdentityFiles is what an identity holds that can leave the vault as a file; Documents are only those naming scans.
type IdentityFiles struct {
	Identity  ID
	Label     string
	Thumbnail []byte
	Documents []DocumentScans
}

// DocumentScans is one document with the scans it names, in their order.
type DocumentScans struct {
	Type  DocumentType
	Label string
	Scans []ScanSummary
}

// IdentityFiles lists every identity outside the trash with its photo thumbnail and document scans without taking the
// selection.
func (s *Session) IdentityFiles() ([]IdentityFiles, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	scans := make(map[ID]ScanSummary)
	for _, entry := range s.entries {
		if entry.kind == KindAttachment {
			scans[entry.id] = ScanSummary{ID: entry.id, Name: entry.label, MediaType: entry.detail, Thumbnail: bytes.Clone(entry.thumbnail)}
		}
	}
	var result []IdentityFiles
	for i, entry := range s.entries {
		if entry.kind != KindIdentity || entry.trashed() {
			continue
		}
		identity, err := s.decryptIdentity(i)
		if err != nil {
			return nil, err
		}
		clear(identity.Photo)
		files := IdentityFiles{Identity: entry.id, Label: entry.label, Thumbnail: bytes.Clone(entry.thumbnail)}
		for _, document := range identity.Documents {
			if len(document.Scans) == 0 {
				continue
			}
			held := DocumentScans{Type: document.Type, Label: document.Label}
			for _, id := range document.Scans {
				scan, found := scans[id]
				if !found {
					return nil, ErrMalformed
				}
				held.Scans = append(held.Scans, scan)
			}
			files.Documents = append(files.Documents, held)
		}
		result = append(result, files)
	}
	return result, nil
}

// ReadIdentityPhoto decrypts one identity's JPEG photo without taking the selection.
func (s *Session) ReadIdentityPhoto(id ID) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	index := s.findKind(id, KindIdentity)
	if index < 0 {
		return nil, ErrNotFound
	}
	identity, err := s.decryptIdentity(index)
	if err != nil {
		return nil, err
	}
	if len(identity.Photo) == 0 {
		return nil, ErrNotFound
	}
	return identity.Photo, nil
}
