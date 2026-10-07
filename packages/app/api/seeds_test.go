package api

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

const testPhrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func testPhraseInput() SeedInput {
	return SeedInput{
		Label:      "Cold wallet",
		Tags:       []string{"Cold storage"},
		Format:     "phrase",
		Words:      strings.Fields(testPhrase),
		Passphrase: "extra words",
		Path:       "m/84'/0'/0'",
		Codes:      []BackupCode{},
		Wallet:     "Ledger",
		Addresses:  []SeedAddress{{Label: "Savings", Value: "bc1qsavings"}, {Label: "Spending", Value: "bc1qspending"}},
		Notes:      "paper copy in the safe",
	}
}

func testCodesInput() SeedInput {
	return SeedInput{
		Label:     "Mail codes",
		Tags:      []string{},
		Format:    "codes",
		Words:     []string{},
		Codes:     []BackupCode{{Value: "1111-2222"}, {Value: "3333-4444", Used: true}, {Value: "5555-6666"}},
		Addresses: []SeedAddress{},
	}
}

func testKeyInput() SeedInput {
	return SeedInput{Label: "Hot key", Format: "key", Words: []string{}, Key: "L1aW4aubDFB7yfras2S1mN3bqg9nwySY8nkoLmJebSLD5BWv3ENZ", Codes: []BackupCode{}, Addresses: []SeedAddress{}, Tags: []string{}}
}

func mustCreateSeed(t *testing.T, service *Service, input SeedInput, groups []string) string {
	t.Helper()
	id, err := service.CreateSeed(input, groups)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustReadSeed(t *testing.T, service *Service, id string) Seed {
	t.Helper()
	seed, err := service.ReadSeed(id)
	if err != nil {
		t.Fatal(err)
	}
	return seed
}

func TestSeedsListFromTheIndexAndReadAsWritten(t *testing.T) {
	service := newReadyService(t)
	family := createTestGroup(t, service, "Family")
	phrase := mustCreateSeed(t, service, testPhraseInput(), []string{family})
	codes := mustCreateSeed(t, service, testCodesInput(), nil)
	key := mustCreateSeed(t, service, testKeyInput(), nil)
	if _, err := service.CreateNote(NoteInput{Label: "Note"}, nil); err != nil {
		t.Fatal(err)
	}
	seeds, err := service.ListSeeds()
	if err != nil {
		t.Fatal(err)
	}
	want := []SeedSummary{
		{ID: phrase, Label: "Cold wallet", Format: "phrase", Wallet: "Ledger", Total: 12, Groups: []string{family}, Tags: []string{"Cold storage"}},
		{ID: codes, Label: "Mail codes", Format: "codes", Total: 3, Used: 1, Groups: []string{}, Tags: []string{}},
		{ID: key, Label: "Hot key", Format: "key", Groups: []string{}, Tags: []string{}},
	}
	if !reflect.DeepEqual(seeds, want) {
		t.Fatalf("seeds = %+v", seeds)
	}
	for id, input := range map[string]SeedInput{phrase: testPhraseInput(), codes: testCodesInput(), key: testKeyInput()} {
		seed := mustReadSeed(t, service, id)
		if seed.ID != id || !reflect.DeepEqual(seed.SeedInput, input) || seed.CheckedOn != "" {
			t.Fatalf("seed = %+v, want %+v", seed, input)
		}
		wantChecksum := ""
		if input.Format == "phrase" {
			wantChecksum = "valid"
		}
		if seed.Checksum != wantChecksum {
			t.Fatalf("checksum of a %s = %q", input.Format, seed.Checksum)
		}
	}
	if seeds, err := service.ListSeeds(); err != nil || seeds[0].LastUsedAt <= 0 {
		t.Fatalf("seeds after a read = %+v, error = %v", seeds, err)
	}
	encoded, err := json.Marshal(mustReadSeed(t, service, key))
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`"words":[]`, `"codes":[]`, `"addresses":[]`, `"groups":[]`, `"checksum":""`} {
		if !strings.Contains(string(encoded), value) {
			t.Fatalf("%s missing from %s", value, encoded)
		}
	}
	if notes, err := service.ListNotes(); err != nil || len(notes) != 1 {
		t.Fatalf("notes beside seeds = %+v, error = %v", notes, err)
	}
}

func TestSeedInputIsRefusedWithoutWriting(t *testing.T) {
	service := newReadyService(t)
	cases := map[string]func(*SeedInput){
		"unknown format":         func(input *SeedInput) { input.Format = "mnemonic" },
		"empty format":           func(input *SeedInput) { input.Format = "" },
		"key on a phrase":        func(input *SeedInput) { input.Key = "L1aW4" },
		"space inside a word":    func(input *SeedInput) { input.Words[0] = "aban don" },
		"blank label":            func(input *SeedInput) { input.Label = " " },
		"address without value":  func(input *SeedInput) { input.Addresses[0].Value = "" },
		"codes on a phrase seed": func(input *SeedInput) { input.Codes = []BackupCode{{Value: "1111"}} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := testPhraseInput()
			mutate(&input)
			_, err := service.CreateSeed(input, nil)
			assertFailure(t, err, failureInvalidItem)
		})
	}
	if seeds, err := service.ListSeeds(); err != nil || len(seeds) != 0 {
		t.Fatalf("seeds after refused input = %+v, error = %v", seeds, err)
	}
	id := mustCreateSeed(t, service, testKeyInput(), nil)
	input := testKeyInput()
	input.Format = "Key"
	assertFailure(t, service.UpdateSeed(id, input, nil), failureInvalidItem)
}

func TestUpdateSeedKeepsTheCheckOnlyForTheSameWords(t *testing.T) {
	service := newReadyService(t)
	id := mustCreateSeed(t, service, testPhraseInput(), nil)
	if err := service.SetPinned(id, true); err != nil {
		t.Fatal(err)
	}
	checkedOn := time.Date(2026, time.March, 4, 23, 30, 0, 0, time.Local)
	if err := service.recordSeedCheck(id, checkedOn); err != nil {
		t.Fatal(err)
	}
	if seed := mustReadSeed(t, service, id); seed.CheckedOn != "2026-03-04" {
		t.Fatalf("checked on = %q", seed.CheckedOn)
	}
	input := testPhraseInput()
	input.Wallet = "Trezor"
	if err := service.UpdateSeed(id, input, nil); err != nil {
		t.Fatal(err)
	}
	if seed := mustReadSeed(t, service, id); seed.CheckedOn != "2026-03-04" || seed.Wallet != "Trezor" {
		t.Fatalf("seed after a wallet change = %+v", seed)
	}
	input.Words[11] = "abandon"
	if err := service.UpdateSeed(id, input, nil); err != nil {
		t.Fatal(err)
	}
	if seed := mustReadSeed(t, service, id); seed.CheckedOn != "" || seed.Checksum != "invalid" {
		t.Fatalf("seed after a word change = %+v", seed)
	}
	if seeds, err := service.ListSeeds(); err != nil || !seeds[0].Pinned {
		t.Fatalf("seeds after the update = %+v, error = %v", seeds, err)
	}
	codes := mustCreateSeed(t, service, testCodesInput(), nil)
	assertFailure(t, service.recordSeedCheck(codes, checkedOn), failureInvalidItem)
	assertFailure(t, service.recordSeedCheck("not-an-id", checkedOn), failureItemUnreadable)
}

func TestCopySeedFieldCopiesEachField(t *testing.T) {
	service := newReadyService(t)
	phrase := mustCreateSeed(t, service, testPhraseInput(), nil)
	clipboard := attachPasteboard(service)
	var discard func()
	schedule := func(_ time.Duration, clear func()) { discard = clear }
	spending := SeedField{Kind: "address", Index: 1}
	fields := map[SeedField]string{
		{Kind: "phrase"}:     testPhrase,
		{Kind: "passphrase"}: "extra words",
		{Kind: "path"}:       "m/84'/0'/0'",
		{Kind: "notes"}:      "paper copy in the safe",
		{Kind: "address"}:    "bc1qsavings",
		spending:             "bc1qspending",
	}
	for field, want := range fields {
		if err := service.copySeedField(phrase, field, schedule); err != nil {
			t.Fatalf("copy of %+v failed: %v", field, err)
		}
		if clipboard.text != want {
			t.Fatalf("clipboard after copying %+v = %q", field, clipboard.text)
		}
	}
	if err := service.copySeedField(phrase, spending, schedule); err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.copySeedField(phrase, SeedField{Kind: "key"}, schedule), failureFieldEmpty)
	refused := []SeedField{
		{}, {Kind: "label"}, {Kind: "wallet"}, {Kind: "Notes"}, {Kind: "notes", Index: 1}, {Kind: "words"}, {Kind: "Phrase"}, {Kind: "codes"},
		{Kind: "address:0"}, {Kind: "Address"}, {Kind: "phrase", Index: 1}, {Kind: "key", Index: -1},
		{Kind: "address", Index: -1}, {Kind: "address", Index: 2},
	}
	for _, field := range refused {
		assertFailure(t, service.copySeedField(phrase, field, schedule), failureFieldNotCopyable)
	}
	if clipboard.text != "bc1qspending" {
		t.Fatalf("a refused field changed the clipboard to %q", clipboard.text)
	}
	discard()
	if clipboard.text != "" {
		t.Fatal("the copied value stayed on the clipboard after its lifetime")
	}
	key := mustCreateSeed(t, service, testKeyInput(), nil)
	if err := service.copySeedField(key, SeedField{Kind: "key"}, schedule); err != nil || clipboard.text != testKeyInput().Key {
		t.Fatalf("copy of a key = %q, %v", clipboard.text, err)
	}
	for _, kind := range []string{"phrase", "passphrase", "path"} {
		assertFailure(t, service.copySeedField(key, SeedField{Kind: kind}, schedule), failureFieldEmpty)
	}
	card, err := service.CreateCard(testCardInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.copySeedField(card, SeedField{Kind: "phrase"}, schedule), failureItemUnreadable)
	assertFailure(t, service.copyCardField(phrase, "number", schedule), failureItemUnreadable)
	_, err = service.ReadSeed(card)
	assertFailure(t, err, failureItemUnreadable)
	assertFailure(t, service.UpdateSeed(card, testKeyInput(), nil), failureItemUnreadable)
}

func TestSpendBackupCodeCopiesThenMarks(t *testing.T) {
	service := newReadyService(t)
	codes := mustCreateSeed(t, service, testCodesInput(), nil)
	phrase := mustCreateSeed(t, service, testPhraseInput(), nil)
	clipboard := attachPasteboard(service)
	var discard func()
	schedule := func(_ time.Duration, clear func()) { discard = clear }
	if err := service.spendBackupCode(codes, 2, schedule); err != nil {
		t.Fatal(err)
	}
	if clipboard.text != "5555-6666" {
		t.Fatalf("clipboard after a code use = %q", clipboard.text)
	}
	seed := mustReadSeed(t, service, codes)
	if used := []bool{seed.Codes[0].Used, seed.Codes[1].Used, seed.Codes[2].Used}; !slices.Equal(used, []bool{false, true, true}) {
		t.Fatalf("codes after a use = %+v", seed.Codes)
	}
	if seeds, err := service.ListSeeds(); err != nil || seeds[0].Used != 2 || seeds[0].LastUsedAt <= 0 {
		t.Fatalf("seeds after a use = %+v, error = %v", seeds, err)
	}
	discard()
	for name, call := range map[string]func() error{
		"used code":          func() error { return service.spendBackupCode(codes, 1, schedule) },
		"index out of range": func() error { return service.spendBackupCode(codes, 3, schedule) },
		"negative index":     func() error { return service.spendBackupCode(codes, -1, schedule) },
		"phrase seed":        func() error { return service.spendBackupCode(phrase, 0, schedule) },
	} {
		t.Run(name, func(t *testing.T) { assertFailure(t, call(), failureInvalidItem) })
	}
	if clipboard.text != "" {
		t.Fatalf("a refused code reached the clipboard: %q", clipboard.text)
	}
	card, err := service.CreateCard(testCardInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.spendBackupCode(card, 0, schedule), failureItemUnreadable)
	assertFailure(t, service.spendBackupCode("not-an-id", 0, schedule), failureItemUnreadable)
	clipboard.failing = true
	assertFailure(t, service.spendBackupCode(codes, 0, schedule), failureCopyFailed)
	if seed := mustReadSeed(t, service, codes); seed.Codes[0].Used {
		t.Fatal("a code that never reached the clipboard was marked used")
	}
}

func TestCheckSeedPhraseNeedsNoVault(t *testing.T) {
	service := &Service{}
	cases := []struct {
		words []string
		want  PhraseCheck
	}{
		{words: strings.Fields(testPhrase), want: PhraseCheck{Checksum: "valid", UnknownWords: []int{}}},
		{words: strings.Fields(strings.Replace(testPhrase, "about", "abandon", 1)), want: PhraseCheck{Checksum: "invalid", UnknownWords: []int{}}},
		{words: []string{"abandon", "ravenpass", "about", "zzz"}, want: PhraseCheck{Checksum: "unknown", UnknownWords: []int{1, 3}}},
		{words: nil, want: PhraseCheck{Checksum: "unknown", UnknownWords: []int{}}},
	}
	for _, testCase := range cases {
		check, err := service.CheckSeedPhrase(testCase.words)
		if err != nil || !reflect.DeepEqual(check, testCase.want) {
			t.Fatalf("check of %q = %+v, %v", testCase.words, check, err)
		}
	}
	encoded, err := json.Marshal(PhraseCheck{Checksum: "unknown", UnknownWords: []int{}})
	if err != nil || string(encoded) != `{"checksum":"unknown","unknownWords":[]}` {
		t.Fatalf("encoded check = %s, %v", encoded, err)
	}
	wordlist, err := service.SeedWordlist()
	if err != nil || len(wordlist) != 2048 || wordlist[0] != "abandon" || wordlist[2047] != "zoo" {
		t.Fatalf("wordlist of %d words, error = %v", len(wordlist), err)
	}
}

func TestSeedLimitsMatchTheVault(t *testing.T) {
	limits, err := (&Service{}).GetSeedLimits()
	if err != nil {
		t.Fatal(err)
	}
	want := SeedLimits{
		Label: vault.MaxLabelLength, Words: vault.MaxSeedWords, Word: vault.MaxSeedWordLength,
		Passphrase: vault.MaxSeedPassphraseLength, Path: vault.MaxDerivationPathLength, Key: vault.MaxPrivateKeyLength,
		Codes: vault.MaxBackupCodes, Code: vault.MaxBackupCodeLength, Wallet: vault.MaxWalletNameLength,
		Addresses: vault.MaxSeedAddresses, AddressLabel: vault.MaxSeedAddressLabelLength, Address: vault.MaxSeedAddressLength,
		Notes: vault.MaxNotesLength,
	}
	if limits != want {
		t.Fatalf("limits = %+v", limits)
	}
}
