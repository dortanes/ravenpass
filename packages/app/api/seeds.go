package api

import (
	"strings"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

// SeedAddress is one labelled address a seed's wallet receives at.
type SeedAddress struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// BackupCode is one backup code and whether it was used.
type BackupCode struct {
	Value string `json:"value"`
	Used  bool   `json:"used"`
}

// SeedInput is one seed; fields of a format other than Format are empty.
type SeedInput struct {
	Label string `json:"label"`
	// Format is "phrase" (Words, Passphrase, Path), "key" (Key) or "codes" (Codes).
	Format     string        `json:"format"`
	Words      []string      `json:"words"`
	Passphrase string        `json:"passphrase"`
	Path       string        `json:"path"`
	Key        string        `json:"key"`
	Codes      []BackupCode  `json:"codes"`
	Wallet     string        `json:"wallet"`
	Addresses  []SeedAddress `json:"addresses"`
	Notes      string        `json:"notes"`
	// Tags tell the item apart from others like it.
	Tags []string `json:"tags"`
}

// SeedSummary is a seed as the list shows it.
type SeedSummary struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Format string `json:"format"`
	Wallet string `json:"wallet"`
	// Total is the word count of a phrase, 0 for a private key and the code count of backup codes.
	Total int `json:"total"`
	// Used is the used backup code count, 0 for other formats.
	Used       int      `json:"used"`
	Pinned     bool     `json:"pinned"`
	LastUsedAt int64    `json:"lastUsedAt"`
	Groups     []string `json:"groups"`
	Tags       []string `json:"tags"`
}

// Seed is a seed as read.
type Seed struct {
	ID     string   `json:"id"`
	Groups []string `json:"groups"`
	// CheckedOn is empty or the YYYY-MM-DD day its copy was last confirmed.
	CheckedOn string `json:"checkedOn"`
	// Checksum is "valid", "invalid" or "unknown" for a phrase, and empty for another format.
	Checksum string `json:"checksum"`
	SeedInput
}

// PhraseCheck is the BIP-39 check of a typed phrase.
type PhraseCheck struct {
	Checksum string `json:"checksum"`
	// UnknownWords are the zero-based positions of words outside the BIP-39 list.
	UnknownWords []int `json:"unknownWords"`
}

var seedFormatNames = map[vault.SeedFormat]string{
	vault.SeedPhrase:      "phrase",
	vault.SeedPrivateKey:  "key",
	vault.SeedBackupCodes: "codes",
}

var seedChecksumNames = map[vault.SeedChecksum]string{
	vault.SeedChecksumValid:   "valid",
	vault.SeedChecksumInvalid: "invalid",
	vault.SeedChecksumUnknown: "unknown",
}

func seedFormatNamed(name string) (vault.SeedFormat, bool) {
	for format, known := range seedFormatNames {
		if known == name {
			return format, true
		}
	}
	return 0, false
}

// ListSeeds reports every seed from the index.
func (s *Service) ListSeeds() ([]SeedSummary, error) {
	entries, usage, err := s.listKind(vault.KindSeed)
	if err != nil {
		return nil, err
	}
	result := make([]SeedSummary, len(entries))
	for i, entry := range entries {
		result[i] = SeedSummary{
			ID:         entry.ID.String(),
			Label:      entry.Label,
			Format:     seedFormatNames[entry.Seed.Format],
			Wallet:     entry.Detail,
			Total:      entry.Seed.Total,
			Used:       entry.Seed.Used,
			Pinned:     entry.Pinned,
			LastUsedAt: usage[entry.ID],
			Groups:     idStrings(entry.Groups),
			Tags:       append([]string{}, entry.Tags...),
		}
	}
	return result, nil
}

// ReadSeed opens the seed id and records the use.
func (s *Service) ReadSeed(id string) (Seed, error) {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return Seed{}, fail(failureItemUnreadable)
	}
	item, groups, err := openItem(s, parsed, s.vault.ReadSelectedSeed)
	if err != nil {
		return Seed{}, err
	}
	seed := Seed{ID: item.ID.String(), Groups: groups, CheckedOn: item.CheckedOn, SeedInput: fromVaultSeed(item.SeedInput)}
	if item.Format == vault.SeedPhrase {
		seed.Checksum = seedChecksumNames[item.Checksum]
	}
	return seed, nil
}

// CreateSeed creates a seed in groups and returns its id.
func (s *Service) CreateSeed(input SeedInput, groups []string) (string, error) {
	membership, err := s.knownGroups(groups)
	if err != nil {
		return "", err
	}
	seed, err := input.toVault()
	if err != nil {
		return "", err
	}
	id, err := s.vault.CreateSeed(seed, membership)
	if err != nil {
		return "", present(err)
	}
	return id.String(), nil
}

// UpdateSeed replaces a seed's content and membership; the checked day survives only unchanged phrase words.
func (s *Service) UpdateSeed(id string, input SeedInput, groups []string) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	membership, err := s.knownGroups(groups)
	if err != nil {
		return err
	}
	seed, err := input.toVault()
	if err != nil {
		return err
	}
	return present(s.vault.EditSeed(parsed, seed, membership))
}

// SeedField names one copyable seed value.
type SeedField struct {
	// Kind is "phrase", "passphrase", "path", "key", "notes" or "address".
	Kind string `json:"kind"`
	// Index is the zero-based position of an address.
	Index int `json:"index,omitempty"`
}

// CopySeedField copies one value of a seed.
func (s *Service) CopySeedField(id string, field SeedField) error {
	return s.copySeedField(id, field, clearAfter)
}

func (s *Service) copySeedField(id string, field SeedField, schedule func(time.Duration, func())) error {
	read, supported := field.reader()
	if !supported {
		return fail(failureFieldNotCopyable)
	}
	return s.copyItemValue(id, schedule, func(parsed vault.ID) (string, error) {
		item, err := readSelected(s, parsed, s.vault.ReadSelectedSeed)
		if err != nil {
			return "", err
		}
		value, found := read(item.SeedInput)
		if !found {
			return "", fail(failureFieldNotCopyable)
		}
		return value, nil
	})
}

var seedFields = map[string]func(vault.SeedInput) string{
	"phrase":     func(seed vault.SeedInput) string { return strings.Join(seed.Words, " ") },
	"passphrase": func(seed vault.SeedInput) string { return seed.Passphrase },
	"path":       func(seed vault.SeedInput) string { return seed.Path },
	"key":        func(seed vault.SeedInput) string { return seed.Key },
	"notes":      func(seed vault.SeedInput) string { return seed.Notes },
}

// reader returns the reader of the one value f names; another format's value reads as empty.
func (f SeedField) reader() (func(vault.SeedInput) (string, bool), bool) {
	if f.Kind == "address" {
		if f.Index < 0 {
			return nil, false
		}
		return func(seed vault.SeedInput) (string, bool) {
			address, found := at(seed.Addresses, f.Index)
			return address.Value, found
		}, true
	}
	read, known := seedFields[f.Kind]
	if !known || f.Index != 0 {
		return nil, false
	}
	return func(seed vault.SeedInput) (string, bool) { return read(seed), true }, true
}

// SpendBackupCode copies the unused backup code at index, then records it as used.
func (s *Service) SpendBackupCode(id string, index int) error {
	return s.spendBackupCode(id, index, clearAfter)
}

func (s *Service) spendBackupCode(id string, index int, schedule func(time.Duration, func())) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	err = s.copyItemValue(id, schedule, func(selected vault.ID) (string, error) {
		item, err := readSelected(s, selected, s.vault.ReadSelectedSeed)
		if err != nil {
			return "", err
		}
		if item.Format != vault.SeedBackupCodes || index < 0 || index >= len(item.Codes) || item.Codes[index].Used {
			return "", fail(failureInvalidItem)
		}
		return item.Codes[index].Value, nil
	})
	if err != nil {
		return err
	}
	return present(s.vault.SpendBackupCode(parsed, index))
}

// RecordSeedCheck records today, in local time, as the day a phrase seed's copy was confirmed.
func (s *Service) RecordSeedCheck(id string) error {
	return s.recordSeedCheck(id, time.Now())
}

func (s *Service) recordSeedCheck(id string, today time.Time) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	return present(s.vault.RecordSeedCheck(parsed, today.Format("2006-01-02")))
}

// CheckSeedPhrase reports the BIP-39 checksum of words as typed. It needs no open vault.
func (s *Service) CheckSeedPhrase(words []string) (PhraseCheck, error) {
	checksum, unknown := vault.CheckSeedPhrase(words)
	return PhraseCheck{Checksum: seedChecksumNames[checksum], UnknownWords: append([]int{}, unknown...)}, nil
}

// SeedWordlist returns the English BIP-39 list. It needs no open vault.
func (s *Service) SeedWordlist() ([]string, error) {
	return vault.SeedWordlist(), nil
}

func (input SeedInput) toVault() (vault.SeedInput, error) {
	format, known := seedFormatNamed(input.Format)
	if !known {
		return vault.SeedInput{}, fail(failureInvalidItem)
	}
	seed := vault.SeedInput{
		Label: input.Label, Format: format, Words: input.Words, Passphrase: input.Passphrase,
		Path: input.Path, Key: input.Key, Wallet: input.Wallet, Notes: input.Notes, Tags: input.Tags,
	}
	if len(input.Codes) > 0 {
		seed.Codes = make([]vault.BackupCode, len(input.Codes))
		for i, code := range input.Codes {
			seed.Codes[i] = vault.BackupCode{Value: code.Value, Used: code.Used}
		}
	}
	if len(input.Addresses) > 0 {
		seed.Addresses = make([]vault.SeedAddress, len(input.Addresses))
		for i, address := range input.Addresses {
			seed.Addresses[i] = vault.SeedAddress{Label: address.Label, Value: address.Value}
		}
	}
	return seed, nil
}

func fromVaultSeed(seed vault.SeedInput) SeedInput {
	input := SeedInput{
		Label: seed.Label, Format: seedFormatNames[seed.Format], Words: append([]string{}, seed.Words...),
		Passphrase: seed.Passphrase, Path: seed.Path, Key: seed.Key, Codes: make([]BackupCode, len(seed.Codes)),
		Wallet: seed.Wallet, Addresses: make([]SeedAddress, len(seed.Addresses)), Notes: seed.Notes,
		Tags: append([]string{}, seed.Tags...),
	}
	for i, code := range seed.Codes {
		input.Codes[i] = BackupCode{Value: code.Value, Used: code.Used}
	}
	for i, address := range seed.Addresses {
		input.Addresses[i] = SeedAddress{Label: address.Label, Value: address.Value}
	}
	return input
}
