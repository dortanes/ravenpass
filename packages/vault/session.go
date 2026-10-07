package vault

import (
	"bytes"
	"crypto/sha256"
	"math"
	"slices"
	"sync"
)

// Session is an open vault; every method is safe for concurrent use.
type Session struct {
	// mu guards every field below it.
	mu             sync.Mutex
	vaultID        ID
	dataKey        [32]byte
	indexKey       [32]byte
	recordKey      [32]byte
	recovery       sealedBox
	entries        []entryMeta
	groups         []Group
	retention      int
	records        []sealedBox
	container      []byte
	head           Head
	ancestry       ancestry
	generation     uint64
	selectionToken uint64
	selectedID     ID
	selected       bool
	locked         bool
	pending        *Pending
	afterDecrypt   func()
}

// Head is the version the session shows.
func (s *Session) Head() (Head, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return Head{}, ErrLocked
	}
	return s.head, nil
}

// List returns every listed item's entry without decrypting a record; an item in the trash is not listed.
func (s *Session) List() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	result := make([]Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		if entry.kind == KindAttachment || entry.trashed() {
			continue
		}
		result = append(result, entry.listed())
	}
	return result, nil
}

// listed is the entry as a client sees it, holding no memory the session holds.
func (e entryMeta) listed() Entry {
	return Entry{ID: e.id, Kind: e.kind, Label: e.label, Detail: e.detail, ExpiresOn: e.expiresOn, Thumbnail: bytes.Clone(e.thumbnail), Site: e.site, Sites: slices.Clone(e.sites), Email: e.email, Code: e.code, Passkeys: clonedFaces(e.passkeys), Apps: slices.Clone(e.apps), Card: e.card, Note: e.note, Seed: e.seed, Pinned: e.pinned, Groups: append([]ID(nil), e.groups...), Tags: slices.Clone(e.tags), DeletedAt: e.deletedAt}
}

// CurrentContainer returns a copy of the file the session shows, with its head.
func (s *Session) CurrentContainer() ([]byte, Head, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, Head{}, ErrLocked
	}
	return append([]byte(nil), s.container...), s.head, nil
}

// Export returns a copy of the file the session shows once every record is verified.
func (s *Session) Export() ([]byte, Head, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, Head{}, ErrLocked
	}
	if s.pending != nil {
		return nil, Head{}, ErrPendingCommit
	}
	if err := s.verifyEntries(); err != nil {
		return nil, Head{}, err
	}
	return append([]byte(nil), s.container...), s.head, nil
}

// Lock clears the session's keys and drops everything it holds; a locked session stays locked.
func (s *Session) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return
	}
	s.locked = true
	clear(s.dataKey[:])
	clear(s.indexKey[:])
	clear(s.recordKey[:])
	s.entries = nil
	s.groups = nil
	s.retention = 0
	s.records = nil
	s.container = nil
	s.ancestry = nil
	if s.pending != nil {
		s.pending.release()
		s.pending = nil
	}
	s.selected = false
	s.generation++
	s.selectionToken++
}

func (s *Session) find(id ID) int {
	for i, entry := range s.entries {
		if entry.id == id {
			return i
		}
	}
	return -1
}

// findItem is find for a listed item; an attachment is reached only through its document, and an item in the trash
// only through the trash's calls.
func (s *Session) findItem(id ID) int {
	index := s.find(id)
	if index < 0 || s.entries[index].kind == KindAttachment || s.entries[index].trashed() {
		return -1
	}
	return index
}

// BeginSelection selects the item with id and returns the ticket that reads it.
func (s *Session) BeginSelection(id ID) (Selection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return Selection{}, ErrLocked
	}
	if s.findItem(id) < 0 {
		return Selection{}, ErrNotFound
	}
	if s.selectionToken == math.MaxUint64 {
		return Selection{}, ErrResourceLimit
	}
	s.selectionToken++
	s.selectedID = id
	s.selected = true
	return Selection{Generation: s.generation, Token: s.selectionToken, ID: id}, nil
}

// ClearSelection makes every ticket stale.
func (s *Session) ClearSelection() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.locked {
		s.selected = false
		s.selectionToken++
	}
}

// findKind is find for an item of kind alone, outside the trash.
func (s *Session) findKind(id ID, kind Kind) int {
	index := s.find(id)
	if index < 0 || s.entries[index].kind != kind || s.entries[index].trashed() {
		return -1
	}
	return index
}

func (s *Session) ticketCurrent(ticket Selection) bool {
	return !s.locked && s.selected && s.generation == ticket.Generation && s.selectionToken == ticket.Token && s.selectedID == ticket.ID
}

// readSelected decrypts the selected record outside the lock and returns only if the ticket is still current afterwards.
func (s *Session) readSelected(ticket Selection, kind Kind, decode func([]byte) error) (entryMeta, error) {
	s.mu.Lock()
	if !s.ticketCurrent(ticket) {
		s.mu.Unlock()
		return entryMeta{}, ErrStaleSelection
	}
	index := s.find(ticket.ID)
	if index < 0 {
		s.mu.Unlock()
		return entryMeta{}, ErrStaleSelection
	}
	entry := s.entries[index]
	if entry.kind != kind {
		s.mu.Unlock()
		return entryMeta{}, ErrNotFound
	}
	box := sealedBox{nonce: s.records[index].nonce, ciphertext: append([]byte(nil), s.records[index].ciphertext...)}
	key := s.recordKey
	vaultID := s.vaultID
	hook := s.afterDecrypt
	s.mu.Unlock()

	plaintext, err := openBox(key, box, recordAAD(vaultID, entry.id, entry.revision))
	clear(key[:])
	if err != nil {
		return entryMeta{}, err
	}
	decodeErr := decode(plaintext)
	clear(plaintext)
	if hook != nil {
		hook()
	}
	s.mu.Lock()
	current := s.ticketCurrent(ticket)
	s.mu.Unlock()
	if !current {
		return entryMeta{}, ErrStaleSelection
	}
	if decodeErr != nil {
		return entryMeta{}, decodeErr
	}
	return entry, nil
}

// ReadSelected decrypts the selected credential. Its passkeys come without their private keys.
func (s *Session) ReadSelected(ticket Selection) (Credential, error) {
	var input CredentialInput
	entry, err := s.readSelected(ticket, KindCredential, func(plaintext []byte) (err error) {
		input, err = decodeCredentialRecord(plaintext)
		forgetPasskeyKeys(input.Passkeys)
		return err
	})
	if err != nil {
		return Credential{}, err
	}
	input.Label = entry.label
	input.Tags = slices.Clone(entry.tags)
	return Credential{ID: entry.id, CredentialInput: input}, nil
}

// ReadCredential decrypts one credential without taking the selection; its passkeys come without private keys.
func (s *Session) ReadCredential(id ID) (Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return Credential{}, ErrLocked
	}
	index := s.findKind(id, KindCredential)
	if index < 0 {
		return Credential{}, ErrNotFound
	}
	input, err := s.decryptCredential(index)
	if err != nil {
		return Credential{}, err
	}
	forgetPasskeyKeys(input.Passkeys)
	return Credential{ID: id, CredentialInput: input}, nil
}

// uniqueID draws random IDs until one is free; eight collisions in a row mean a broken random source.
func uniqueID(taken func(ID) bool) (ID, error) {
	var id ID
	for attempt := 0; attempt < 8; attempt++ {
		if err := randomBytes(id[:]); err != nil {
			return ID{}, err
		}
		if !taken(id) {
			return id, nil
		}
	}
	return ID{}, ErrResourceLimit
}

// freshID draws an identifier no item holds and none reserved for this save does.
func (s *Session) freshID(reserved map[ID]struct{}) (ID, error) {
	return uniqueID(func(candidate ID) bool {
		_, taken := reserved[candidate]
		return taken || s.find(candidate) >= 0
	})
}

// sealFirst seals plaintext as the first revision of a new item with identifier id.
func (s *Session) sealFirst(id ID, entry entryMeta, plaintext []byte) (entryMeta, sealedBox, error) {
	box, err := seal(s.recordKey, plaintext, recordAAD(s.vaultID, id, 1))
	if err != nil {
		return entryMeta{}, sealedBox{}, err
	}
	entry.id, entry.revision, entry.digest = id, 1, sha256.Sum256(encodeBox(box))
	return entry, box, nil
}

// sealNext seals plaintext as the next revision of the item at index, keeping its ID and pin.
func (s *Session) sealNext(index int, entry entryMeta, plaintext []byte) (entryMeta, sealedBox, error) {
	current := s.entries[index]
	revision, err := nextRevision(current.revision)
	if err != nil {
		return entryMeta{}, sealedBox{}, err
	}
	box, err := seal(s.recordKey, plaintext, recordAAD(s.vaultID, current.id, revision))
	if err != nil {
		return entryMeta{}, sealedBox{}, err
	}
	entry.id, entry.revision, entry.pinned, entry.digest = current.id, revision, current.pinned, sha256.Sum256(encodeBox(box))
	return entry, box, nil
}

// prepareAdd prepares the index that lists a new item.
func (s *Session) prepareAdd(entry entryMeta, plaintext []byte) (*Pending, ID, error) {
	id, err := s.freshID(nil)
	if err != nil {
		return nil, ID{}, err
	}
	entry, box, err := s.sealFirst(id, entry, plaintext)
	if err != nil {
		return nil, ID{}, err
	}
	entries := append(append([]entryMeta(nil), s.entries...), entry)
	records := append(append([]sealedBox(nil), s.records...), box)
	pending, err := s.prepare(entries, records)
	return pending, entry.id, err
}

// prepareReplace prepares a new revision of the item at index. No other record is sealed again.
func (s *Session) prepareReplace(index int, entry entryMeta, plaintext []byte) (*Pending, error) {
	entry, box, err := s.sealNext(index, entry, plaintext)
	if err != nil {
		return nil, err
	}
	entries := append([]entryMeta(nil), s.entries...)
	records := append([]sealedBox(nil), s.records...)
	entries[index] = entry
	records[index] = box
	return s.prepare(entries, records)
}

// PrepareCreate prepares a new credential in groups.
func (s *Session) PrepareCreate(input CredentialInput, groups []ID) (*Pending, ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, ID{}, err
	}
	input, err := acceptInput(input)
	if err != nil {
		return nil, ID{}, err
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, ID{}, err
	}
	plaintext, err := encodeCredentialRecord(input)
	if err != nil {
		return nil, ID{}, err
	}
	defer clear(plaintext)
	return s.prepareAdd(credentialEntry(input, membership), plaintext)
}

// credentialEntry is what the index shows of an accepted credential.
func credentialEntry(input CredentialInput, groups []ID) entryMeta {
	return entryMeta{kind: KindCredential, label: input.Label, detail: input.Login, groups: groups, site: input.Site(), sites: input.Sites(), email: input.Email, code: input.CodeFace(), passkeys: passkeyFaces(input.Passkeys), apps: slices.Clone(input.Apps), tags: input.Tags}
}

// credentialToChange decrypts credential id for its next revision; the caller holds the lock and forgets the returned private keys.
func (s *Session) credentialToChange(id ID) (int, CredentialInput, error) {
	if err := s.readyToWrite(); err != nil {
		return 0, CredentialInput{}, err
	}
	index := s.findKind(id, KindCredential)
	if index < 0 {
		return 0, CredentialInput{}, ErrNotFound
	}
	if _, err := nextRevision(s.entries[index].revision); err != nil {
		return 0, CredentialInput{}, err
	}
	input, err := s.decryptCredential(index)
	if err != nil {
		return 0, CredentialInput{}, err
	}
	return index, input, nil
}

// PrepareEdit prepares the credential with id changed by patch.
func (s *Session) PrepareEdit(id ID, patch CredentialPatch) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, input, err := s.credentialToChange(id)
	if err != nil {
		return nil, err
	}
	defer forgetPasskeyKeys(input.Passkeys)
	input, membership, err := s.patched(index, input, patch)
	if err != nil {
		return nil, err
	}
	plaintext, err := encodeCredentialRecord(input)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	return s.prepareReplace(index, credentialEntry(input, membership), plaintext)
}

// patched is the accepted credential at index, read as input, changed by patch, with its membership. The returned
// passkeys share their private keys with input's. The caller holds the lock.
func (s *Session) patched(index int, input CredentialInput, patch CredentialPatch) (CredentialInput, []ID, error) {
	var err error
	if input.Passkeys, err = withoutPasskeys(input.Passkeys, patch.RemovePasskeys); err != nil {
		return CredentialInput{}, nil, err
	}
	if patch.Label != nil {
		input.Label = *patch.Label
	}
	if patch.Websites != nil {
		input.Websites = *patch.Websites
	}
	if patch.Login != nil {
		input.Login = *patch.Login
	}
	if patch.Email != nil {
		input.Email = *patch.Email
	}
	if patch.Password != nil {
		input.Password = *patch.Password
	}
	if patch.Notes != nil {
		input.Notes = *patch.Notes
	}
	if patch.TOTP != nil {
		input.TOTP = *patch.TOTP
	}
	if patch.Apps != nil {
		input.Apps = *patch.Apps
	}
	if patch.Tags != nil {
		input.Tags = *patch.Tags
	}
	input, err = acceptInput(input)
	if err != nil {
		return CredentialInput{}, nil, err
	}
	membership := s.entries[index].groups
	if patch.Groups != nil {
		membership, err = acceptMembership(*patch.Groups, s.groups)
		if err != nil {
			return CredentialInput{}, nil, err
		}
	}
	return input, membership, nil
}

// PrepareSetPinned prepares the item's pin set to pinned, sealing no record again.
func (s *Session) PrepareSetPinned(id ID, pinned bool) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	index := s.findItem(id)
	if index < 0 {
		return nil, ErrNotFound
	}
	entries := append([]entryMeta(nil), s.entries...)
	entries[index].pinned = pinned
	records := append([]sealedBox(nil), s.records...)
	return s.prepare(entries, records)
}

// PrepareDelete prepares the permanent removal of an item, in the trash or not; an identity takes its scans and every
// card link to it along.
func (s *Session) PrepareDelete(id ID) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	index := s.find(id)
	if index < 0 || s.entries[index].kind == KindAttachment {
		return nil, ErrNotFound
	}
	return s.prepareRemoval(map[ID]struct{}{id: {}})
}

// prepareRemoval prepares the permanent removal of every item in removed with its scans, and of every card link to a
// removed identity. The caller holds the lock.
func (s *Session) prepareRemoval(removed map[ID]struct{}) (*Pending, error) {
	entries := append([]entryMeta(nil), s.entries...)
	records := append([]sealedBox(nil), s.records...)
	if slices.ContainsFunc(s.entries, func(entry entryMeta) bool {
		_, gone := removed[entry.id]
		return gone && entry.kind == KindIdentity
	}) {
		if err := s.unlinkCards(entries, records, func(link AddressLink) bool {
			_, gone := removed[link.Identity]
			return gone
		}); err != nil {
			return nil, err
		}
	}
	entries, records = withoutItems(entries, records, func(entry entryMeta) bool {
		_, gone := removed[entry.id]
		_, ownerGone := removed[entry.owner]
		return gone || entry.kind == KindAttachment && ownerGone
	})
	return s.prepare(entries, records)
}

func (s *Session) readyToWrite() error {
	if s.locked {
		return ErrLocked
	}
	if s.pending != nil {
		return ErrPendingCommit
	}
	if s.head.Revision == math.MaxUint64 {
		return ErrResourceLimit
	}
	return nil
}

func (s *Session) prepare(entries []entryMeta, records []sealedBox) (*Pending, error) {
	return s.prepareWithGroups(entries, records, append([]Group(nil), s.groups...))
}

func (s *Session) prepareWithGroups(entries []entryMeta, records []sealedBox, groups []Group) (*Pending, error) {
	return s.prepareSealed(entries, records, groups, s.retention, s.indexKey, s.recovery)
}

// prepareSealed prepares the next revision keeping items in the trash for retention days, with its index sealed by
// indexKey beside recovery.
func (s *Session) prepareSealed(entries []entryMeta, records []sealedBox, groups []Group, retention int, indexKey [32]byte, recovery sealedBox) (*Pending, error) {
	revision := s.head.Revision + 1
	ancestors := s.ancestry.after(s.head.Hash)
	indexPlaintext, err := encodeIndex(revision, ancestors, entries, groups, retention)
	if err != nil {
		return nil, err
	}
	indexBox, err := seal(indexKey, indexPlaintext, indexAAD(s.vaultID, recovery))
	clear(indexPlaintext)
	if err != nil {
		return nil, err
	}
	container, err := encodeContainer(s.vaultID, recovery, indexBox, records)
	if err != nil {
		return nil, err
	}
	pending := &Pending{session: s, parentHash: s.head.Hash, entries: entries, groups: groups, retention: retention, records: records, container: container, head: Head{VaultID: s.vaultID, Revision: revision, Hash: sha256.Sum256(container), PreviousHash: s.head.Hash}, ancestry: ancestors}
	s.pending = pending
	return pending, nil
}

// Commit makes pending the session's version once its container is stored.
func (s *Session) Commit(pending *Pending) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return ErrLocked
	}
	if pending == nil || s.pending != pending || pending.session != s || pending.parentHash != s.head.Hash {
		return ErrStaleCommit
	}
	raw, err := parseContainer(pending.container)
	if err != nil {
		return err
	}
	if pending.rekeyed != nil {
		s.dataKey, s.indexKey, s.recordKey = pending.rekeyed.data, pending.rekeyed.index, pending.rekeyed.record
	}
	s.entries = pending.entries
	s.groups = pending.groups
	s.retention = pending.retention
	s.records = raw.records
	s.recovery = raw.recovery
	s.container = raw.data
	s.head = pending.head
	s.ancestry = pending.ancestry
	s.pending = nil
	pending.release()
	s.selected = false
	s.selectionToken++
	return nil
}

// Abort drops pending, leaving the session as it was.
func (s *Session) Abort(pending *Pending) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return ErrLocked
	}
	if pending == nil || s.pending != pending {
		return ErrStaleCommit
	}
	s.pending = nil
	pending.release()
	return nil
}
