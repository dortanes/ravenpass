package vault

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	bip39 "github.com/kslamph/bip39-hdwallet/bip39"
	"golang.org/x/text/unicode/norm"
)

// SeedFormat is what a seed holds, numbered as the vault stores it.
type SeedFormat uint8

const (
	SeedPhrase SeedFormat = iota + 1
	SeedPrivateKey
	SeedBackupCodes
)

// Seed limits are counted in characters; a word is bounded in its stored form.
const (
	MaxSeedWords              = 48
	MaxSeedWordLength         = 32
	MaxSeedPassphraseLength   = 256
	MaxDerivationPathLength   = 128
	MaxPrivateKeyLength       = 1024
	MaxBackupCodes            = 64
	MaxBackupCodeLength       = 64
	MaxWalletNameLength       = 128
	MaxSeedAddresses          = 16
	MaxSeedAddressLabelLength = 32
	MaxSeedAddressLength      = 256
)

// SeedFace is what the list shows of a seed beside its wallet name without decrypting it.
type SeedFace struct {
	Format SeedFormat
	// Total counts a phrase's words or the backup codes, and Used the used codes.
	Total int
	Used  int
}

// validSeedFace holds the index rules for a seed's face.
func validSeedFace(face SeedFace) bool {
	switch face.Format {
	case SeedPhrase:
		return face.Total >= 1 && face.Total <= MaxSeedWords && face.Used == 0
	case SeedPrivateKey:
		return face.Total == 0 && face.Used == 0
	case SeedBackupCodes:
		return face.Total >= 1 && face.Total <= MaxBackupCodes && face.Used >= 0 && face.Used <= face.Total
	default:
		return false
	}
}

// SeedAddress is one labelled address a seed's wallet receives at.
type SeedAddress struct{ Label, Value string }

// BackupCode is one backup code and whether it has been used.
type BackupCode struct {
	Value string
	Used  bool
}

// SeedInput is one seed; only the fields of its Format are set, and Words are stored in NFKD and lower case.
type SeedInput struct {
	Label      string
	Format     SeedFormat
	Words      []string
	Passphrase string
	Path       string
	Key        string
	Codes      []BackupCode
	Wallet     string
	Addresses  []SeedAddress
	Notes      string
	Tags       []string
}

// SeedChecksum is what BIP-39 makes of a phrase.
type SeedChecksum uint8

const (
	SeedChecksumUnknown SeedChecksum = iota
	SeedChecksumValid
	SeedChecksumInvalid
)

// Seed is a seed as read; CheckedOn is the day its copy was last confirmed, and a phrase's Checksum is computed on read.
type Seed struct {
	ID        ID
	CheckedOn string
	Checksum  SeedChecksum
	SeedInput
}

// normalizeSeedWord returns a word in Unicode NFKD and lower case.
func normalizeSeedWord(word string) string {
	return strings.ToLower(norm.NFKD.String(word))
}

// CheckSeedPhrase reports the BIP-39 English checksum of words and the zero-based positions of unlisted words.
func CheckSeedPhrase(words []string) (SeedChecksum, []int) {
	normalized := make([]string, len(words))
	var unknown []int
	for i, word := range words {
		normalized[i] = normalizeSeedWord(word)
		if _, listed := bip39.GetWordIndex(normalized[i]); !listed {
			unknown = append(unknown, i)
		}
	}
	if len(unknown) > 0 || len(words) < 12 || len(words) > 24 || len(words)%3 != 0 {
		return SeedChecksumUnknown, unknown
	}
	if bip39.IsMnemonicValid(strings.Join(normalized, " ")) {
		return SeedChecksumValid, nil
	}
	return SeedChecksumInvalid, nil
}

// SeedWordlist returns the English BIP-39 list.
func SeedWordlist() []string {
	return slices.Clone(bip39.GetWordList())
}

// acceptSeed returns a seed as stored with its words normalized, refusing any value out of bounds.
func acceptSeed(input SeedInput) (SeedInput, error) {
	if !fits(input.Label, MaxLabelLength) || strings.TrimSpace(input.Label) == "" {
		return SeedInput{}, ErrInvalidInput
	}
	tags, err := AcceptTags(input.Tags)
	if err != nil {
		return SeedInput{}, err
	}
	input.Tags = tags
	if !fits(input.Wallet, MaxWalletNameLength) || !fits(input.Notes, MaxNotesLength) || len(input.Addresses) > MaxSeedAddresses {
		return SeedInput{}, ErrInvalidInput
	}
	for _, address := range input.Addresses {
		if !fits(address.Label, MaxSeedAddressLabelLength) || !fits(address.Value, MaxSeedAddressLength) || strings.TrimSpace(address.Value) == "" {
			return SeedInput{}, ErrInvalidInput
		}
	}
	phrase := len(input.Words) > 0 || input.Passphrase != "" || input.Path != ""
	switch input.Format {
	case SeedPhrase:
		if input.Key != "" || len(input.Codes) > 0 || len(input.Words) == 0 || len(input.Words) > MaxSeedWords {
			return SeedInput{}, ErrInvalidInput
		}
		if !fits(input.Passphrase, MaxSeedPassphraseLength) || !fits(input.Path, MaxDerivationPathLength) {
			return SeedInput{}, ErrInvalidInput
		}
		words := make([]string, len(input.Words))
		for i, word := range input.Words {
			if !utf8.ValidString(word) {
				return SeedInput{}, ErrInvalidInput
			}
			words[i] = normalizeSeedWord(word)
			if words[i] == "" || !fits(words[i], MaxSeedWordLength) || strings.ContainsFunc(words[i], unicode.IsSpace) {
				return SeedInput{}, ErrInvalidInput
			}
		}
		input.Words = words
	case SeedPrivateKey:
		if phrase || len(input.Codes) > 0 || input.Key == "" || !fits(input.Key, MaxPrivateKeyLength) {
			return SeedInput{}, ErrInvalidInput
		}
	case SeedBackupCodes:
		if phrase || input.Key != "" || len(input.Codes) == 0 || len(input.Codes) > MaxBackupCodes {
			return SeedInput{}, ErrInvalidInput
		}
		for _, code := range input.Codes {
			if !fits(code.Value, MaxBackupCodeLength) || strings.TrimSpace(code.Value) == "" {
				return SeedInput{}, ErrInvalidInput
			}
		}
		input.Codes = slices.Clone(input.Codes)
	default:
		return SeedInput{}, ErrInvalidInput
	}
	input.Addresses = slices.Clone(input.Addresses)
	return input, nil
}

func seedFace(input SeedInput) SeedFace {
	face := SeedFace{Format: input.Format}
	switch input.Format {
	case SeedPhrase:
		face.Total = len(input.Words)
	case SeedBackupCodes:
		face.Total = len(input.Codes)
		for _, code := range input.Codes {
			if code.Used {
				face.Used++
			}
		}
	}
	return face
}

// seedEntry is what the index shows of an accepted seed.
func seedEntry(input SeedInput, groups []ID) entryMeta {
	return entryMeta{kind: KindSeed, label: input.Label, detail: input.Wallet, seed: seedFace(input), groups: groups, tags: input.Tags}
}

// A seed record holds everything but the label, which lives in the index; checkedOn is empty or
// YYYY-MM-DD:
//
//	[11, format, [word…], passphrase, path, key, [[code, used]…], wallet, [[label, value]…],
//	 checkedOn, notes]
const recordSchemaSeed = 11

type seedRecord struct {
	_          struct{} `cbor:",toarray"`
	Schema     uint64
	Format     uint64
	Words      []string
	Passphrase string
	Path       string
	Key        string
	Codes      []wireBackupCode
	Wallet     string
	Addresses  []wireSeedAddress
	CheckedOn  string
	Notes      string
}

type wireBackupCode struct {
	_     struct{} `cbor:",toarray"`
	Value string
	Used  flag
}

type wireSeedAddress struct {
	_     struct{} `cbor:",toarray"`
	Label string
	Value string
}

func encodeSeedRecord(input SeedInput, checkedOn string) ([]byte, error) {
	record := seedRecord{
		Schema: recordSchemaSeed, Format: uint64(input.Format), Words: input.Words, Passphrase: input.Passphrase, Path: input.Path, Key: input.Key,
		Codes: make([]wireBackupCode, len(input.Codes)), Wallet: input.Wallet, Addresses: make([]wireSeedAddress, len(input.Addresses)),
		CheckedOn: checkedOn, Notes: input.Notes,
	}
	for i, code := range input.Codes {
		record.Codes[i] = wireBackupCode{Value: code.Value, Used: flag(code.Used)}
	}
	for i, address := range input.Addresses {
		record.Addresses[i] = wireSeedAddress{Label: address.Label, Value: address.Value}
	}
	return sealableRecord(marshal(record, 0))
}

// decodeSeedRecord returns the seed a record holds and its checkedOn.
func decodeSeedRecord(plaintext []byte) (SeedInput, string, error) {
	if err := readSchema(plaintext, recordSchemaSeed); err != nil {
		return SeedInput{}, "", err
	}
	var record seedRecord
	if err := unmarshal(plaintext, &record); err != nil {
		return SeedInput{}, "", err
	}
	if record.Format == 0 {
		return SeedInput{}, "", ErrMalformed
	}
	if record.Format > uint64(SeedBackupCodes) {
		return SeedInput{}, "", ErrUnsupported
	}
	texts := slices.Concat([]string{record.Passphrase, record.Path, record.Key, record.Wallet, record.CheckedOn, record.Notes}, record.Words)
	if len(record.Words) > MaxSeedWords || len(record.Codes) > MaxBackupCodes || len(record.Addresses) > MaxSeedAddresses || !validText(texts...) || !ValidDate(record.CheckedOn) {
		return SeedInput{}, "", ErrMalformed
	}
	input := SeedInput{
		Format: SeedFormat(record.Format), Words: nilIfEmpty(record.Words), Passphrase: record.Passphrase, Path: record.Path, Key: record.Key,
		Wallet: record.Wallet, Notes: record.Notes,
	}
	for _, code := range record.Codes {
		if !validText(code.Value) {
			return SeedInput{}, "", ErrMalformed
		}
		input.Codes = append(input.Codes, BackupCode{Value: code.Value, Used: bool(code.Used)})
	}
	for _, address := range record.Addresses {
		if !validText(address.Label, address.Value) {
			return SeedInput{}, "", ErrMalformed
		}
		input.Addresses = append(input.Addresses, SeedAddress{Label: address.Label, Value: address.Value})
	}
	return input, record.CheckedOn, nil
}

// verifySeedEntry checks that a record holds an accepted seed matching its entry, with a checked day only for a phrase.
func verifySeedEntry(entry entryMeta, plaintext []byte) error {
	input, checkedOn, err := decodeSeedRecord(plaintext)
	if err != nil {
		return err
	}
	input.Label = entry.label
	accepted, err := acceptSeed(input)
	if err != nil || !slices.Equal(accepted.Words, input.Words) || input.Format != SeedPhrase && checkedOn != "" {
		return ErrMalformed
	}
	want := seedEntry(accepted, entry.groups)
	if entry.detail != want.detail || entry.seed != want.seed {
		return ErrMalformed
	}
	return nil
}

func (s *Session) decryptSeed(index int) (SeedInput, string, error) {
	plaintext, err := s.openRecord(index)
	if err != nil {
		return SeedInput{}, "", err
	}
	defer clear(plaintext)
	input, checkedOn, err := decodeSeedRecord(plaintext)
	input.Label = s.entries[index].label
	input.Tags = slices.Clone(s.entries[index].tags)
	return input, checkedOn, err
}

// ReadSelectedSeed decrypts the selected seed.
func (s *Session) ReadSelectedSeed(ticket Selection) (Seed, error) {
	var input SeedInput
	var checkedOn string
	entry, err := s.readSelected(ticket, KindSeed, func(plaintext []byte) (err error) {
		input, checkedOn, err = decodeSeedRecord(plaintext)
		return err
	})
	if err != nil {
		return Seed{}, err
	}
	input.Label = entry.label
	input.Tags = slices.Clone(entry.tags)
	seed := Seed{ID: entry.id, CheckedOn: checkedOn, SeedInput: input}
	if input.Format == SeedPhrase {
		seed.Checksum, _ = CheckSeedPhrase(input.Words)
	}
	return seed, nil
}

// PrepareCreateSeed prepares a new seed in groups.
func (s *Session) PrepareCreateSeed(input SeedInput, groups []ID) (*Pending, ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, ID{}, err
	}
	input, err := acceptSeed(input)
	if err != nil {
		return nil, ID{}, err
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, ID{}, err
	}
	plaintext, err := encodeSeedRecord(input, "")
	if err != nil {
		return nil, ID{}, err
	}
	defer clear(plaintext)
	return s.prepareAdd(seedEntry(input, membership), plaintext)
}

// PrepareEditSeed replaces a seed's content and membership, keeping its checked day only for the same normalized phrase.
func (s *Session) PrepareEditSeed(id ID, input SeedInput, groups []ID) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	index := s.findKind(id, KindSeed)
	if index < 0 {
		return nil, ErrNotFound
	}
	input, err := acceptSeed(input)
	if err != nil {
		return nil, err
	}
	membership, err := acceptMembership(groups, s.groups)
	if err != nil {
		return nil, err
	}
	stored, checkedOn, err := s.decryptSeed(index)
	if err != nil {
		return nil, err
	}
	if stored.Format != SeedPhrase || input.Format != SeedPhrase || !slices.Equal(stored.Words, input.Words) {
		checkedOn = ""
	}
	plaintext, err := encodeSeedRecord(input, checkedOn)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	return s.prepareReplace(index, seedEntry(input, membership), plaintext)
}

// PrepareUseBackupCode marks the unused backup code at index used.
func (s *Session) PrepareUseBackupCode(id ID, index int) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	position := s.findKind(id, KindSeed)
	if position < 0 {
		return nil, ErrNotFound
	}
	input, checkedOn, err := s.decryptSeed(position)
	if err != nil {
		return nil, err
	}
	if input.Format != SeedBackupCodes || index < 0 || index >= len(input.Codes) || input.Codes[index].Used {
		return nil, ErrInvalidInput
	}
	input.Codes[index].Used = true
	plaintext, err := encodeSeedRecord(input, checkedOn)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	return s.prepareReplace(position, seedEntry(input, s.entries[position].groups), plaintext)
}

// PrepareRecordSeedCheck sets a phrase seed's checkedOn to on, a valid YYYY-MM-DD.
func (s *Session) PrepareRecordSeedCheck(id ID, on string) (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyToWrite(); err != nil {
		return nil, err
	}
	index := s.findKind(id, KindSeed)
	if index < 0 {
		return nil, ErrNotFound
	}
	if on == "" || !ValidDate(on) {
		return nil, ErrInvalidInput
	}
	input, _, err := s.decryptSeed(index)
	if err != nil {
		return nil, err
	}
	if input.Format != SeedPhrase {
		return nil, ErrInvalidInput
	}
	plaintext, err := encodeSeedRecord(input, on)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	return s.prepareReplace(index, seedEntry(input, s.entries[index].groups), plaintext)
}
