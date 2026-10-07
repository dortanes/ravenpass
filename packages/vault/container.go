package vault

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/chacha20poly1305"
)

// A container is formatPrefix followed by [header, recovery, index, [record…]], each but the header a box.
type wireContainer struct {
	_        struct{} `cbor:",toarray"`
	Header   wireHeader
	Recovery wireBox
	Index    wireBox
	Records  []wireBox
}

// encode writes the container into a buffer of capacity bytes.
func (c wireContainer) encode(capacity int) ([]byte, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, capacity))
	buffer.WriteString(formatPrefix)
	if err := encoding.MarshalToBuffer(c, buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type rawContainer struct {
	vaultID  ID
	recovery sealedBox
	index    sealedBox
	records  []sealedBox
	data     []byte
}

// parseContainer reads a container in its canonical encoding; the result holds its own copy.
func parseContainer(input []byte) (rawContainer, error) {
	if len(input) > MaxContainerBytes {
		return rawContainer{}, ErrResourceLimit
	}
	if len(input) < len(formatPrefix) || string(input[:len(formatPrefix)-2]) != "RAVENVLT" {
		return rawContainer{}, ErrMalformed
	}
	if string(input[:len(formatPrefix)]) != formatPrefix {
		return rawContainer{}, ErrUnsupported
	}
	var container wireContainer
	if err := decoding.Unmarshal(input[len(formatPrefix):], &container); err != nil {
		return rawContainer{}, decodeError(err)
	}
	data, err := container.encode(len(input))
	if err != nil || !bytes.Equal(data, input) {
		return rawContainer{}, ErrMalformed
	}
	if container.Header.Suite != cipherSuite {
		return rawContainer{}, ErrUnsupported
	}
	recovery, err := container.Recovery.sealed(wrappedKeyBytes)
	if err != nil {
		return rawContainer{}, err
	}
	if len(recovery.ciphertext) != wrappedKeyBytes {
		return rawContainer{}, ErrMalformed
	}
	index, err := container.Index.sealed(maxIndexBytes)
	if err != nil {
		return rawContainer{}, err
	}
	records := make([]sealedBox, len(container.Records))
	for i, record := range container.Records {
		if records[i], err = record.sealed(maxRecordBytes); err != nil {
			return rawContainer{}, err
		}
	}
	return rawContainer{vaultID: container.Header.VaultID, recovery: recovery, index: index, records: records, data: data}, nil
}

// InspectUntrustedVaultID reads the unauthenticated header ID of a bounded, canonical container.
func InspectUntrustedVaultID(container []byte) (ID, error) {
	raw, err := parseContainer(container)
	if err != nil {
		return ID{}, err
	}
	return raw.vaultID, nil
}

func encodeContainer(vaultID ID, recovery, index sealedBox, records []sealedBox) ([]byte, error) {
	if len(records) > maxEntries || len(index.ciphertext) > maxIndexBytes || len(recovery.ciphertext) > wrappedKeyBytes {
		return nil, ErrResourceLimit
	}
	container := wireContainer{Header: wireHeader{VaultID: vaultID, Suite: cipherSuite}, Recovery: recovery.wire(), Index: index.wire(), Records: make([]wireBox, len(records))}
	size := len(formatPrefix) + len(recovery.ciphertext) + len(index.ciphertext)
	for i, record := range records {
		if len(record.ciphertext) > maxRecordBytes {
			return nil, ErrResourceLimit
		}
		size += len(record.ciphertext)
		if size > MaxContainerBytes {
			return nil, ErrResourceLimit
		}
		container.Records[i] = record.wire()
	}
	encoded, err := container.encode(size)
	if err != nil {
		return nil, err
	}
	if len(encoded) > MaxContainerBytes {
		return nil, ErrResourceLimit
	}
	return encoded, nil
}

// A credential record holds everything but the label, which lives in the index; each list may be
// empty:
//
//	[14, [website…], login, email, password, notes, totp, [passkey…], [app…]]
const recordSchemaCredential = 14

type credentialRecord struct {
	_        struct{} `cbor:",toarray"`
	Schema   uint64
	Websites []string
	Login    string
	Email    string
	Password string
	Notes    string
	TOTP     string
	Passkeys []wirePasskey
	Apps     []wireApp
}

// encodeCredentialRecord encodes a credential; the caller clears the result.
func encodeCredentialRecord(input CredentialInput) ([]byte, error) {
	if !validText(slices.Concat([]string{input.Label, input.Login, input.Email, input.Password, input.Notes, input.TOTP}, input.Websites)...) {
		return nil, ErrInvalidInput
	}
	record := credentialRecord{
		Schema: recordSchemaCredential, Websites: input.Websites, Login: input.Login, Email: input.Email, Password: input.Password,
		Notes: input.Notes, TOTP: input.TOTP, Passkeys: wirePasskeys(input.Passkeys), Apps: wireApps(input.Apps),
	}
	capacity := 0
	// A buffer that never grows leaves no copy of a private key behind.
	if len(input.Passkeys) > 0 {
		capacity = maxRecordBytes
	}
	return sealableRecord(marshal(record, capacity))
}

// sealableRecord refuses an encoded record too large to seal, clearing it.
func sealableRecord(encoded []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxRecordBytes-chacha20poly1305.Overhead {
		clear(encoded)
		return nil, ErrResourceLimit
	}
	return encoded, nil
}

func decodeCredentialRecord(plaintext []byte) (CredentialInput, error) {
	if err := readSchema(plaintext, recordSchemaCredential); err != nil {
		return CredentialInput{}, err
	}
	var record credentialRecord
	input, err := record.credential(plaintext)
	if err != nil {
		forgetWirePasskeys(record.Passkeys)
		return CredentialInput{}, err
	}
	return input, nil
}

// credential decodes plaintext into the record; on failure the record may hold private keys the caller clears.
func (r *credentialRecord) credential(plaintext []byte) (CredentialInput, error) {
	if err := unmarshal(plaintext, r); err != nil {
		return CredentialInput{}, err
	}
	if len(r.Websites) > MaxCredentialWebsites || !validText(append([]string{r.Login, r.Email, r.Password, r.Notes, r.TOTP}, r.Websites...)...) {
		return CredentialInput{}, ErrMalformed
	}
	input := CredentialInput{Websites: nilIfEmpty(r.Websites), Login: r.Login, Email: r.Email, Password: r.Password, Notes: r.Notes, TOTP: r.TOTP}
	var err error
	if input.Apps, err = parseApps(r.Apps); err != nil {
		return CredentialInput{}, err
	}
	if input.Passkeys, err = parsePasskeys(r.Passkeys); err != nil {
		return CredentialInput{}, err
	}
	return input, nil
}

func openSession(raw rawContainer, dataKey [32]byte) (*Session, error) {
	indexKey, recordKey, err := deriveContentKeys(dataKey, raw.vaultID)
	if err != nil {
		return nil, err
	}
	plaintext, err := openBox(indexKey, raw.index, indexAAD(raw.vaultID, raw.recovery))
	if err != nil {
		clear(indexKey[:])
		clear(recordKey[:])
		return nil, err
	}
	revision, ancestors, index, err := parseIndex(plaintext, raw.records)
	clear(plaintext)
	if err != nil {
		clear(indexKey[:])
		clear(recordKey[:])
		return nil, err
	}
	return &Session{
		vaultID:    raw.vaultID,
		dataKey:    dataKey,
		indexKey:   indexKey,
		recordKey:  recordKey,
		recovery:   raw.recovery,
		entries:    index.entries,
		groups:     index.groups,
		retention:  index.retention,
		records:    raw.records,
		container:  raw.data,
		head:       Head{VaultID: raw.vaultID, Revision: revision, Hash: sha256.Sum256(raw.data), PreviousHash: ancestors.previous()},
		ancestry:   ancestors,
		generation: 1,
	}, nil
}

// Create makes a new empty vault with a fresh vault key and recovery phrase.
func Create() (Created, error) {
	var vaultID ID
	if err := randomBytes(vaultID[:]); err != nil {
		return Created{}, err
	}
	key, phrase, err := newVaultKey(vaultID)
	if err != nil {
		return Created{}, err
	}
	defer clear(key.data[:])
	indexKey, err := deriveKey(key.data[:], vaultID, "index")
	if err != nil {
		return Created{}, err
	}
	indexPlaintext, err := encodeIndex(1, nil, nil, nil, DefaultTrashRetention)
	if err != nil {
		clear(indexKey[:])
		return Created{}, err
	}
	index, err := seal(indexKey, indexPlaintext, indexAAD(vaultID, key.recovery))
	clear(indexPlaintext)
	clear(indexKey[:])
	if err != nil {
		return Created{}, err
	}
	container, err := encodeContainer(vaultID, key.recovery, index, nil)
	if err != nil {
		return Created{}, err
	}
	raw, err := parseContainer(container)
	if err != nil {
		return Created{}, err
	}
	session, err := openSession(raw, key.data)
	if err != nil {
		return Created{}, err
	}
	return Created{Container: append([]byte(nil), container...), RecoveryPhrase: phrase, Session: session}, nil
}

// OpenWithRecovery opens container with its recovery phrase once every record is verified.
func OpenWithRecovery(container []byte, phrase string) (*Session, error) {
	raw, err := parseContainer(container)
	if err != nil {
		return nil, err
	}
	dataKey, err := openRecovery(raw.vaultID, raw.recovery, phrase)
	if err != nil {
		return nil, err
	}
	session, err := openSession(raw, dataKey)
	clear(dataKey[:])
	if err != nil {
		return nil, err
	}
	if err := session.VerifyAll(); err != nil {
		session.Lock()
		return nil, err
	}
	return session, nil
}

// OpenWithDevice opens container with a device key; a successor opens once persistAdvancedWitness records it, a diverged one as a Divergence.
func OpenWithDevice(container, deviceKey, deviceEnvelope []byte, witness *Witness, persistAdvancedWitness func(Witness) error) (*Session, *Divergence, error) {
	if len(deviceKey) != 32 {
		return nil, nil, ErrInvalidInput
	}
	raw, err := parseContainer(container)
	if err != nil {
		return nil, nil, err
	}
	dataKey, err := openDeviceEnvelope(deviceKey, deviceEnvelope, raw.vaultID, keyIdentity(raw.recovery))
	if err != nil {
		return nil, nil, err
	}
	session, err := openSession(raw, dataKey)
	clear(dataKey[:])
	if err != nil {
		return nil, nil, err
	}
	decision, err := session.ReconcileWitness(witness)
	if errors.Is(err, ErrWitnessDiverged) {
		if err := session.VerifyAll(); err != nil {
			session.Lock()
			return nil, nil, err
		}
		return nil, &Divergence{session: session}, nil
	}
	if err != nil {
		session.Lock()
		return nil, nil, err
	}
	if decision == WitnessAdvance {
		if persistAdvancedWitness == nil {
			session.Lock()
			return nil, nil, ErrWitnessAdvanceRequired
		}
		if err := persistAdvancedWitness(WitnessFor(session.head)); err != nil {
			session.Lock()
			return nil, nil, err
		}
	}
	return session, nil, nil
}

// CheckDeviceEnvelope reports, without a device key, whether envelope holds the key container is sealed under: ErrKeyReplaced when it holds a replaced one.
func CheckDeviceEnvelope(container, envelope []byte) error {
	raw, err := parseContainer(container)
	if err != nil {
		return err
	}
	_, err = currentEnvelopeBox(envelope, raw.vaultID, keyIdentity(raw.recovery))
	return err
}

// CheckDeviceEnvelope is the package's CheckDeviceEnvelope against the key the session uses.
func (s *Session) CheckDeviceEnvelope(envelope []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return ErrLocked
	}
	_, err := currentEnvelopeBox(envelope, s.vaultID, keyIdentity(s.recovery))
	return err
}

// KeyIdentity names the vault key the session uses; a committed rekey changes it.
func (s *Session) KeyIdentity() ([32]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return [32]byte{}, ErrLocked
	}
	return keyIdentity(s.recovery), nil
}

// EnvelopeKeyIdentity names the vault key a device envelope for vaultID holds, as Session.KeyIdentity names a session's.
func EnvelopeKeyIdentity(envelope []byte, vaultID ID) ([32]byte, error) {
	held, _, err := decodeDeviceEnvelope(envelope, vaultID)
	return held, err
}

// currentEnvelopeBox decodes a device envelope, failing with ErrKeyReplaced unless it holds the key current names.
func currentEnvelopeBox(envelope []byte, vaultID ID, current [32]byte) (sealedBox, error) {
	held, box, err := decodeDeviceEnvelope(envelope, vaultID)
	if err != nil {
		return sealedBox{}, err
	}
	if held != current {
		return sealedBox{}, ErrKeyReplaced
	}
	return box, nil
}

// openDeviceEnvelope unwraps the vault key an envelope holds for deviceKey, current naming the key the vault uses.
func openDeviceEnvelope(deviceKey, envelope []byte, vaultID ID, current [32]byte) ([32]byte, error) {
	box, err := currentEnvelopeBox(envelope, vaultID, current)
	if err != nil {
		return [32]byte{}, err
	}
	key, err := deriveKey(deviceKey, vaultID, "device-wrap")
	if err != nil {
		return [32]byte{}, err
	}
	plaintext, err := openBox(key, box, deviceAAD(vaultID, current))
	clear(key[:])
	if err != nil {
		return [32]byte{}, err
	}
	defer clear(plaintext)
	if len(plaintext) != 32 {
		return [32]byte{}, ErrAuthentication
	}
	var dataKey [32]byte
	copy(dataKey[:], plaintext)
	return dataKey, nil
}

// VerifyDeviceKey checks, without opening anything, that envelope holds this vault's key for deviceKey.
func (s *Session) VerifyDeviceKey(deviceKey, envelope []byte) error {
	if len(deviceKey) != 32 {
		return ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return ErrLocked
	}
	dataKey, err := openDeviceEnvelope(deviceKey, envelope, s.vaultID, keyIdentity(s.recovery))
	if err != nil {
		return err
	}
	defer clear(dataKey[:])
	if subtle.ConstantTimeCompare(dataKey[:], s.dataKey[:]) != 1 {
		return ErrAuthentication
	}
	return nil
}

// WrapDeviceKey seals the vault key for deviceKey into an envelope OpenWithDevice opens.
func (s *Session) WrapDeviceKey(deviceKey []byte) ([]byte, error) {
	if len(deviceKey) != 32 {
		return nil, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, ErrLocked
	}
	return sealDeviceEnvelope(deviceKey, s.vaultID, &s.dataKey, s.recovery)
}

// sealDeviceEnvelope seals dataKey, which recovery seals for the recovery phrase, for deviceKey.
func sealDeviceEnvelope(deviceKey []byte, vaultID ID, dataKey *[32]byte, recovery sealedBox) ([]byte, error) {
	key, err := deriveKey(deviceKey, vaultID, "device-wrap")
	if err != nil {
		return nil, err
	}
	identity := keyIdentity(recovery)
	box, err := seal(key, dataKey[:], deviceAAD(vaultID, identity))
	clear(key[:])
	if err != nil {
		return nil, err
	}
	return encodeDeviceEnvelope(vaultID, identity, box), nil
}

func (s *Session) openRecord(index int) ([]byte, error) {
	entry := s.entries[index]
	return openBox(s.recordKey, s.records[index], recordAAD(s.vaultID, entry.id, entry.revision))
}

func (s *Session) decryptCredential(index int) (CredentialInput, error) {
	plaintext, err := s.openRecord(index)
	if err != nil {
		return CredentialInput{}, err
	}
	defer clear(plaintext)
	input, err := decodeCredentialRecord(plaintext)
	input.Label = s.entries[index].label
	input.Tags = slices.Clone(s.entries[index].tags)
	return input, err
}

// verifyEntry authenticates one record, checks its entry against it and adds the scans an identity names to named.
func (s *Session) verifyEntry(index int, named map[ID]ID) error {
	plaintext, err := s.openRecord(index)
	if err != nil {
		return err
	}
	defer clear(plaintext)
	entry := s.entries[index]
	switch entry.kind {
	case KindCredential:
		var input CredentialInput
		input, err = decodeCredentialRecord(plaintext)
		forgetPasskeyKeys(input.Passkeys)
	case KindIdentity:
		var identity IdentityInput
		identity, err = decodeIdentityRecord(plaintext)
		if err == nil && (len(identity.Photo) > 0) != (len(entry.thumbnail) > 0) {
			err = ErrMalformed
		}
		for _, scan := range scansOf(identity.Documents) {
			if _, repeated := named[scan]; repeated {
				return ErrMalformed
			}
			named[scan] = entry.id
		}
	case KindAttachment:
		_, err = decodeAttachmentRecord(plaintext, entry.detail)
	case KindCard:
		err = verifyCardEntry(entry, plaintext)
	case KindNote:
		err = verifyNoteEntry(entry, plaintext)
	case KindSeed:
		err = verifySeedEntry(entry, plaintext)
	default:
		err = ErrUnsupported
	}
	return err
}

func (s *Session) verifyEntries() error {
	named := make(map[ID]ID)
	for i := range s.entries {
		if err := s.verifyEntry(i, named); err != nil {
			return err
		}
	}
	return s.verifyScanReferences(named)
}

// VerifyAll authenticates and decodes every record of the vault.
func (s *Session) VerifyAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return ErrLocked
	}
	return s.verifyEntries()
}

// acceptInput returns a credential as stored, never shortened, with empty lists nil for a deterministic encoding.
func acceptInput(input CredentialInput) (CredentialInput, error) {
	fields := []struct {
		value string
		limit int
	}{
		{input.Label, MaxLabelLength},
		{input.Login, MaxLoginLength},
		{input.Email, MaxEmailLength},
		{input.Password, MaxPasswordLength},
		{input.Notes, MaxNotesLength},
		{input.TOTP, MaxTOTPLength},
	}
	for _, field := range fields {
		if !fits(field.value, field.limit) {
			return CredentialInput{}, ErrInvalidInput
		}
	}
	if strings.TrimSpace(input.Label) == "" || !acceptValues(input.Websites, MaxCredentialWebsites, MaxOriginLength) {
		return CredentialInput{}, ErrInvalidInput
	}
	setup, err := NormalizeTOTP(input.TOTP)
	if err != nil {
		return CredentialInput{}, err
	}
	// The canonical link can outgrow what was typed where it escapes characters.
	if utf8.RuneCountInString(setup) > MaxTOTPLength {
		return CredentialInput{}, ErrInvalidInput
	}
	passkeys, err := acceptPasskeys(input.Passkeys)
	if err != nil {
		return CredentialInput{}, err
	}
	apps, err := acceptApps(input.Apps)
	if err != nil {
		return CredentialInput{}, err
	}
	if input.Tags, err = AcceptTags(input.Tags); err != nil {
		return CredentialInput{}, err
	}
	input.TOTP = setup
	input.Websites = nilIfEmpty(slices.Clone(input.Websites))
	input.Passkeys = passkeys
	input.Apps = apps
	return input, nil
}

// fits reports whether a value is valid text of at most limit characters.
func fits(value string, limit int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= limit
}

func nextRevision(current uint64) (uint64, error) {
	if current == math.MaxUint64 {
		return 0, ErrResourceLimit
	}
	return current + 1, nil
}
