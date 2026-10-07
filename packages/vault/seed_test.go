package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// BIP-39 test vectors for all-zero entropy of 128 and 256 bits and all-one entropy of 256 bits.
var (
	zeroPhrase12 = append(slices.Repeat([]string{"abandon"}, 11), "about")
	zeroPhrase24 = append(slices.Repeat([]string{"abandon"}, 23), "art")
	onePhrase24  = append(slices.Repeat([]string{"zoo"}, 23), "vote")
)

func fullPhraseSeed() SeedInput {
	return SeedInput{
		Label:      "  Cold wallet  ",
		Format:     SeedPhrase,
		Words:      slices.Clone(zeroPhrase12),
		Passphrase: "correct horse battery staple",
		Path:       "m/84'/0'/0'",
		Wallet:     "Ledger Nano",
		Addresses: []SeedAddress{
			{Label: "Savings", Value: "bc1qsavingsaddressexample0001"},
			{Label: "", Value: "bc1qunlabelledaddressexample2"},
		},
		Notes: " stored in the safe\nsecond line ",
	}
}

func fullKeySeed() SeedInput {
	return SeedInput{Label: "Hot key", Format: SeedPrivateKey, Key: "L1privatekeyexampleUnique9f3c", Wallet: "Electrum", Addresses: []SeedAddress{{Label: "Main", Value: "1MainAddressExample"}}, Notes: "note"}
}

func fullCodesSeed() SeedInput {
	return SeedInput{Label: "Exchange", Format: SeedBackupCodes, Codes: []BackupCode{{Value: "code-alpha-7741"}, {Value: "code-bravo-9902", Used: true}, {Value: "code-charlie-3318"}}, Wallet: "Exchange account"}
}

func commitSeed(t *testing.T, session *Session, input SeedInput, groups []ID) ID {
	t.Helper()
	pending, id, err := session.PrepareCreateSeed(input, groups)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	return id
}

func selectSeed(t *testing.T, session *Session, id ID) Seed {
	t.Helper()
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	value, err := session.ReadSelectedSeed(ticket)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func commitPending(t *testing.T, session *Session, pending *Pending, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
}

func TestSeedRoundTripsEveryValueOfEachFormat(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
	typed := fullPhraseSeed()
	typed.Words[0], typed.Words[11] = "ABANDON", "About"
	phrase := commitSeed(t, created.Session, typed, nil)
	key := commitSeed(t, created.Session, fullKeySeed(), nil)
	codes := commitSeed(t, created.Session, fullCodesSeed(), nil)
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
	tests := []struct {
		id       ID
		input    SeedInput
		checksum SeedChecksum
		entry    Entry
	}{
		{phrase, fullPhraseSeed(), SeedChecksumValid, Entry{ID: phrase, Kind: KindSeed, Label: "  Cold wallet  ", Detail: "Ledger Nano", Seed: SeedFace{Format: SeedPhrase, Total: 12}}},
		{key, fullKeySeed(), SeedChecksumUnknown, Entry{ID: key, Kind: KindSeed, Label: "Hot key", Detail: "Electrum", Seed: SeedFace{Format: SeedPrivateKey}}},
		{codes, fullCodesSeed(), SeedChecksumUnknown, Entry{ID: codes, Kind: KindSeed, Label: "Exchange", Detail: "Exchange account", Seed: SeedFace{Format: SeedBackupCodes, Total: 3, Used: 1}}},
	}
	for _, test := range tests {
		seed := selectSeed(t, reopened, test.id)
		if seed.ID != test.id || seed.CheckedOn != "" || seed.Checksum != test.checksum || !reflect.DeepEqual(seed.SeedInput, test.input) {
			t.Fatalf("seed changed across a reopen: %#v", seed)
		}
		if entry := listedEntry(t, reopened, test.id); !reflect.DeepEqual(entry, test.entry) {
			t.Fatalf("seed entry = %+v", entry)
		}
	}
	if credential := listedEntry(t, reopened, ids[0]); credential.Seed != (SeedFace{}) {
		t.Fatalf("credential entry carries a seed face: %+v", credential)
	}
}

func TestSeedWithoutOptionalValuesHasABareEntry(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	for _, input := range []SeedInput{
		{Label: "Phrase", Format: SeedPhrase, Words: []string{"one"}},
		{Label: "Key", Format: SeedPrivateKey, Key: "k"},
		{Label: "Codes", Format: SeedBackupCodes, Codes: []BackupCode{{Value: "c"}}},
	} {
		id := commitSeed(t, session, input, nil)
		if seed := selectSeed(t, session, id); !reflect.DeepEqual(seed.SeedInput, input) || seed.Checksum != SeedChecksumUnknown {
			t.Fatalf("bare seed = %#v", seed)
		}
		want := Entry{ID: id, Kind: KindSeed, Label: input.Label, Seed: seedFace(input)}
		if entry := listedEntry(t, session, id); !reflect.DeepEqual(entry, want) {
			t.Fatalf("bare seed entry = %+v", entry)
		}
	}
}

func TestSeedWordsAreStoredNormalized(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	composed, decomposed := "café", "café"
	input := SeedInput{Label: "Mixed", Format: SeedPhrase, Words: []string{"ABANDON", "Zoo", composed, "ａｂ", "K"}, Passphrase: "Keep AS Typed é"}
	id := commitSeed(t, session, input, nil)
	seed := selectSeed(t, session, id)
	if want := []string{"abandon", "zoo", decomposed, "ab", "k"}; !reflect.DeepEqual(seed.Words, want) {
		t.Fatalf("stored words = %q, want %q", seed.Words, want)
	}
	if seed.Passphrase != input.Passphrase || input.Words[0] != "ABANDON" {
		t.Fatalf("a value other than the words was changed: %q, %q", seed.Passphrase, input.Words[0])
	}
}

func TestSeedAtItsLimitsRoundTrips(t *testing.T) {
	at := func(limit int) string { return strings.Repeat("ü", limit) }
	addresses := make([]SeedAddress, MaxSeedAddresses)
	for i := range addresses {
		addresses[i] = SeedAddress{Label: at(MaxSeedAddressLabelLength), Value: at(MaxSeedAddressLength)}
	}
	codes := make([]BackupCode, MaxBackupCodes)
	for i := range codes {
		codes[i] = BackupCode{Value: at(MaxBackupCodeLength), Used: i%2 == 0}
	}
	word := strings.Repeat("w", MaxSeedWordLength)
	inputs := []SeedInput{
		{Label: at(MaxLabelLength), Format: SeedPhrase, Words: slices.Repeat([]string{word}, MaxSeedWords), Passphrase: at(MaxSeedPassphraseLength),
			Path: at(MaxDerivationPathLength), Wallet: at(MaxWalletNameLength), Addresses: addresses, Notes: at(MaxNotesLength)},
		{Label: "Key", Format: SeedPrivateKey, Key: at(MaxPrivateKeyLength), Addresses: addresses},
		{Label: "Codes", Format: SeedBackupCodes, Codes: codes, Wallet: at(MaxWalletNameLength)},
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	for _, input := range inputs {
		id := commitSeed(t, session, input, nil)
		if seed := selectSeed(t, session, id); !reflect.DeepEqual(seed.SeedInput, input) {
			t.Fatal("a seed at its limits changed on its way through the vault")
		}
	}
	if entry := listedEntry(t, session, session.entries[len(session.entries)-1].id); entry.Seed != (SeedFace{Format: SeedBackupCodes, Total: MaxBackupCodes, Used: MaxBackupCodes / 2}) {
		t.Fatalf("codes seed face = %+v", entry.Seed)
	}
}

func TestSeedRefusesEachBoundWithoutWriting(t *testing.T) {
	over := func(limit int) string { return strings.Repeat("ü", limit+1) }
	tooManyAddresses := make([]SeedAddress, MaxSeedAddresses+1)
	for i := range tooManyAddresses {
		tooManyAddresses[i] = SeedAddress{Value: "address"}
	}
	tooManyCodes := make([]BackupCode, MaxBackupCodes+1)
	for i := range tooManyCodes {
		tooManyCodes[i] = BackupCode{Value: "code"}
	}
	phrase := func(mutate func(*SeedInput)) func() SeedInput {
		return func() SeedInput { input := fullPhraseSeed(); mutate(&input); return input }
	}
	key := func(mutate func(*SeedInput)) func() SeedInput {
		return func() SeedInput { input := fullKeySeed(); mutate(&input); return input }
	}
	codes := func(mutate func(*SeedInput)) func() SeedInput {
		return func() SeedInput { input := fullCodesSeed(); mutate(&input); return input }
	}
	tests := []struct {
		name  string
		build func() SeedInput
	}{
		{"empty label", phrase(func(input *SeedInput) { input.Label = "" })},
		{"blank label", phrase(func(input *SeedInput) { input.Label = " \t\n" })},
		{"label over its limit", phrase(func(input *SeedInput) { input.Label = over(MaxLabelLength) })},
		{"label that is not text", phrase(func(input *SeedInput) { input.Label = "\xff" })},
		{"no format", phrase(func(input *SeedInput) { input.Format = 0 })},
		{"unknown format", phrase(func(input *SeedInput) { input.Format = SeedBackupCodes + 1 })},
		{"no words", phrase(func(input *SeedInput) { input.Words = nil })},
		{"too many words", phrase(func(input *SeedInput) { input.Words = slices.Repeat([]string{"zoo"}, MaxSeedWords+1) })},
		{"empty word", phrase(func(input *SeedInput) { input.Words[3] = "" })},
		{"word over its limit", phrase(func(input *SeedInput) { input.Words[3] = strings.Repeat("w", MaxSeedWordLength+1) })},
		{"word over its limit once decomposed", phrase(func(input *SeedInput) { input.Words[3] = strings.Repeat("é", MaxSeedWordLength/2+1) })},
		{"word with a space", phrase(func(input *SeedInput) { input.Words[3] = "aban don" })},
		{"word with a tab", phrase(func(input *SeedInput) { input.Words[3] = "aban\tdon" })},
		{"word with a line break", phrase(func(input *SeedInput) { input.Words[3] = "abandon\n" })},
		{"word with a no-break space", phrase(func(input *SeedInput) { input.Words[3] = "aban don" })},
		{"word with an ideographic space", phrase(func(input *SeedInput) { input.Words[3] = "aban　don" })},
		{"word that is not text", phrase(func(input *SeedInput) { input.Words[3] = "\xff" })},
		{"passphrase over its limit", phrase(func(input *SeedInput) { input.Passphrase = over(MaxSeedPassphraseLength) })},
		{"path over its limit", phrase(func(input *SeedInput) { input.Path = over(MaxDerivationPathLength) })},
		{"phrase with a key", phrase(func(input *SeedInput) { input.Key = "key" })},
		{"phrase with codes", phrase(func(input *SeedInput) { input.Codes = []BackupCode{{Value: "code"}} })},
		{"wallet over its limit", phrase(func(input *SeedInput) { input.Wallet = over(MaxWalletNameLength) })},
		{"notes over their limit", phrase(func(input *SeedInput) { input.Notes = over(MaxNotesLength) })},
		{"too many addresses", phrase(func(input *SeedInput) { input.Addresses = tooManyAddresses })},
		{"address label over its limit", phrase(func(input *SeedInput) { input.Addresses[0].Label = over(MaxSeedAddressLabelLength) })},
		{"address over its limit", phrase(func(input *SeedInput) { input.Addresses[0].Value = over(MaxSeedAddressLength) })},
		{"empty address", phrase(func(input *SeedInput) { input.Addresses[0].Value = "" })},
		{"blank address", phrase(func(input *SeedInput) { input.Addresses[0].Value = " \t" })},
		{"empty key", key(func(input *SeedInput) { input.Key = "" })},
		{"key over its limit", key(func(input *SeedInput) { input.Key = over(MaxPrivateKeyLength) })},
		{"key with words", key(func(input *SeedInput) { input.Words = []string{"zoo"} })},
		{"key with a passphrase", key(func(input *SeedInput) { input.Passphrase = "pass" })},
		{"key with a path", key(func(input *SeedInput) { input.Path = "m/0" })},
		{"key with codes", key(func(input *SeedInput) { input.Codes = []BackupCode{{Value: "code"}} })},
		{"no codes", codes(func(input *SeedInput) { input.Codes = nil })},
		{"too many codes", codes(func(input *SeedInput) { input.Codes = tooManyCodes })},
		{"empty code", codes(func(input *SeedInput) { input.Codes[1].Value = "" })},
		{"blank code", codes(func(input *SeedInput) { input.Codes[1].Value = "  " })},
		{"code over its limit", codes(func(input *SeedInput) { input.Codes[1].Value = over(MaxBackupCodeLength) })},
		{"codes with words", codes(func(input *SeedInput) { input.Words = []string{"zoo"} })},
		{"codes with a passphrase", codes(func(input *SeedInput) { input.Passphrase = "pass" })},
		{"codes with a key", codes(func(input *SeedInput) { input.Key = "key" })},
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	existing := commitSeed(t, session, fullPhraseSeed(), nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := session.PrepareCreateSeed(test.build(), nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("create: got %v, want ErrInvalidInput", err)
			}
			if _, err := session.PrepareEditSeed(existing, test.build(), nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("edit: got %v, want ErrInvalidInput", err)
			}
		})
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused seed changed the vault")
	}
	if seed := selectSeed(t, session, existing); !reflect.DeepEqual(seed.SeedInput, fullPhraseSeed()) {
		t.Fatalf("a refused edit changed the seed: %#v", seed.SeedInput)
	}
}

func TestCheckSeedPhrase(t *testing.T) {
	changedLast := append(slices.Clone(zeroPhrase12[:11]), "abandon")
	unknownWord := slices.Clone(zeroPhrase12)
	unknownWord[3], unknownWord[7] = "bitcoin", "abandonx"
	tests := []struct {
		name    string
		words   []string
		want    SeedChecksum
		unknown []int
	}{
		{"valid 12 words", zeroPhrase12, SeedChecksumValid, nil},
		{"valid 24 words", zeroPhrase24, SeedChecksumValid, nil},
		{"another valid 24 words", onePhrase24, SeedChecksumValid, nil},
		{"valid words in upper case", append(slices.Repeat([]string{"ABANDON"}, 11), "About"), SeedChecksumValid, nil},
		{"valid words in full width", append(slices.Repeat([]string{"ａｂａｎｄｏｎ"}, 11), "about"), SeedChecksumValid, nil},
		{"changed last word", changedLast, SeedChecksumInvalid, nil},
		{"words outside the list", unknownWord, SeedChecksumUnknown, []int{3, 7}},
		{"13 words", append(slices.Clone(zeroPhrase12), "about"), SeedChecksumUnknown, nil},
		{"11 words", zeroPhrase12[:11], SeedChecksumUnknown, nil},
		{"no words", nil, SeedChecksumUnknown, nil},
		{"two words in one", append(slices.Clone(zeroPhrase12[:10]), "abandon about"), SeedChecksumUnknown, []int{10}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, unknown := CheckSeedPhrase(test.words)
			if got != test.want || !reflect.DeepEqual(unknown, test.unknown) {
				t.Fatalf("got %v, %v; want %v, %v", got, unknown, test.want, test.unknown)
			}
		})
	}
}

func TestSeedWordlistIsTheEnglishList(t *testing.T) {
	list := SeedWordlist()
	if len(list) != 2048 || list[0] != "abandon" || list[2047] != "zoo" {
		t.Fatalf("word list of %d words from %q to %q", len(list), list[0], list[len(list)-1])
	}
	list[0] = "changed"
	if SeedWordlist()[0] != "abandon" {
		t.Fatal("changing a returned list changed the next one")
	}
}

func TestPhraseIsSavedWhateverItsChecksum(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	input := fullPhraseSeed()
	input.Words[11] = "abandon"
	id := commitSeed(t, session, input, nil)
	if seed := selectSeed(t, session, id); seed.Checksum != SeedChecksumInvalid || !reflect.DeepEqual(seed.Words, input.Words) {
		t.Fatalf("seed with a failing checksum = %#v", seed)
	}
	unknown := commitSeed(t, session, SeedInput{Label: "Other", Format: SeedPhrase, Words: []string{"not", "bip39"}}, nil)
	if seed := selectSeed(t, session, unknown); seed.Checksum != SeedChecksumUnknown {
		t.Fatalf("checksum of a phrase outside BIP-39 = %v", seed.Checksum)
	}
	twentyFour := commitSeed(t, session, SeedInput{Label: "Long", Format: SeedPhrase, Words: onePhrase24}, nil)
	if seed := selectSeed(t, session, twentyFour); seed.Checksum != SeedChecksumValid {
		t.Fatalf("checksum of a valid 24-word phrase = %v", seed.Checksum)
	}
}

func TestSeedCheckIsKeptForTheSameWordsAndClearedOtherwise(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family")
	session := created.Session
	defer session.Lock()
	id := commitSeed(t, session, fullPhraseSeed(), []ID{groupIDs[0]})
	pending, err := session.PrepareSetPinned(id, true)
	commitPending(t, session, pending, err)
	pending, err = session.PrepareRecordSeedCheck(id, "2026-09-21")
	commitPending(t, session, pending, err)
	seed := selectSeed(t, session, id)
	if seed.CheckedOn != "2026-09-21" || !reflect.DeepEqual(seed.SeedInput, fullPhraseSeed()) || session.entries[0].revision != 2 {
		t.Fatalf("seed after a check = %#v", seed)
	}
	if entry := listedEntry(t, session, id); !entry.Pinned || !reflect.DeepEqual(entry.Groups, []ID{groupIDs[0]}) {
		t.Fatalf("seed entry after a check = %+v", entry)
	}

	sameWords := fullPhraseSeed()
	sameWords.Words[0], sameWords.Wallet, sameWords.Passphrase = "ABANDON", "Trezor", "changed"
	pending, err = session.PrepareEditSeed(id, sameWords, nil)
	commitPending(t, session, pending, err)
	if seed := selectSeed(t, session, id); seed.CheckedOn != "2026-09-21" || seed.Wallet != "Trezor" {
		t.Fatalf("check after an edit with the same words = %q", seed.CheckedOn)
	}

	changed := fullPhraseSeed()
	changed.Words[5] = "zoo"
	pending, err = session.PrepareEditSeed(id, changed, nil)
	commitPending(t, session, pending, err)
	if seed := selectSeed(t, session, id); seed.CheckedOn != "" {
		t.Fatalf("check after an edit with changed words = %q", seed.CheckedOn)
	}

	pending, err = session.PrepareRecordSeedCheck(id, "2026-09-22")
	commitPending(t, session, pending, err)
	pending, err = session.PrepareEditSeed(id, fullKeySeed(), nil)
	commitPending(t, session, pending, err)
	if seed := selectSeed(t, session, id); seed.CheckedOn != "" || seed.Format != SeedPrivateKey {
		t.Fatalf("check after an edit to another format = %#v", seed)
	}
	if err := session.VerifyAll(); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareRecordSeedCheckRefusals(t *testing.T) {
	session, credential := populatedSession(t)
	defer session.Lock()
	phrase := commitSeed(t, session, fullPhraseSeed(), nil)
	key := commitSeed(t, session, fullKeySeed(), nil)
	codes := commitSeed(t, session, fullCodesSeed(), nil)
	note := commitNote(t, session, fullNote(), nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		id   ID
		on   string
		want error
	}{
		{"private key seed", key, "2026-09-21", ErrInvalidInput},
		{"backup codes seed", codes, "2026-09-21", ErrInvalidInput},
		{"empty day", phrase, "", ErrInvalidInput},
		{"impossible day", phrase, "2026-02-30", ErrInvalidInput},
		{"day in another form", phrase, "21.09.2026", ErrInvalidInput},
		{"day before 1900", phrase, "1899-12-31", ErrInvalidInput},
		{"credential", credential, "2026-09-21", ErrNotFound},
		{"note", note, "2026-09-21", ErrNotFound},
		{"no item", ID{0xee}, "2026-09-21", ErrNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := session.PrepareRecordSeedCheck(test.id, test.on); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused check changed the vault")
	}
}

func TestPrepareUseBackupCode(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family")
	session := created.Session
	defer session.Lock()
	credential := commitCredential(t, session, CredentialInput{Label: "Mail", Password: "secret"}, nil)
	phrase := commitSeed(t, session, fullPhraseSeed(), nil)
	key := commitSeed(t, session, fullKeySeed(), nil)
	id := commitSeed(t, session, fullCodesSeed(), []ID{groupIDs[0]})
	pending, err := session.PrepareSetPinned(id, true)
	commitPending(t, session, pending, err)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		id    ID
		index int
		want  error
	}{
		{"code already used", id, 1, ErrInvalidInput},
		{"index below range", id, -1, ErrInvalidInput},
		{"index past range", id, 3, ErrInvalidInput},
		{"phrase seed", phrase, 0, ErrInvalidInput},
		{"private key seed", key, 0, ErrInvalidInput},
		{"credential", credential, 0, ErrNotFound},
		{"no item", ID{0xee}, 0, ErrNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := session.PrepareUseBackupCode(test.id, test.index); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused use changed the vault")
	}

	raw, _ := currentRaw(t, session)
	pending, err = session.PrepareUseBackupCode(id, 2)
	commitPending(t, session, pending, err)
	want := fullCodesSeed()
	want.Codes[2].Used = true
	if seed := selectSeed(t, session, id); !reflect.DeepEqual(seed.SeedInput, want) {
		t.Fatalf("seed after a code was used = %#v", seed.SeedInput)
	}
	entry := listedEntry(t, session, id)
	if entry.Seed != (SeedFace{Format: SeedBackupCodes, Total: 3, Used: 2}) || !entry.Pinned || !reflect.DeepEqual(entry.Groups, []ID{groupIDs[0]}) {
		t.Fatalf("seed entry after a code was used = %+v", entry)
	}
	index := session.find(id)
	if session.entries[index].revision != 2 {
		t.Fatalf("seed revision after one use = %d", session.entries[index].revision)
	}
	next, nextHead := currentRaw(t, session)
	if nextHead.Revision != head.Revision+1 {
		t.Fatalf("using a code took %d saves", nextHead.Revision-head.Revision)
	}
	for i := range index {
		if !bytes.Equal(encodeBox(next.records[i]), encodeBox(raw.records[i])) {
			t.Fatal("using a code sealed another record again")
		}
	}
	if _, err := session.PrepareUseBackupCode(id, 2); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("using a code twice: %v", err)
	}
	if err := session.VerifyAll(); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareEditSeedReplacesContentAndMembershipAndKeepsPin(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family", "Travel")
	session := created.Session
	defer session.Lock()
	id := commitSeed(t, session, fullPhraseSeed(), []ID{groupIDs[0]})
	pending, err := session.PrepareSetPinned(id, true)
	commitPending(t, session, pending, err)
	replacement := fullCodesSeed()
	pending, err = session.PrepareEditSeed(id, replacement, []ID{groupIDs[1]})
	commitPending(t, session, pending, err)
	want := Entry{ID: id, Kind: KindSeed, Label: "Exchange", Detail: "Exchange account", Seed: SeedFace{Format: SeedBackupCodes, Total: 3, Used: 1}, Pinned: true, Groups: []ID{groupIDs[1]}}
	if entry := listedEntry(t, session, id); !reflect.DeepEqual(entry, want) {
		t.Fatalf("seed entry after the edit = %+v", entry)
	}
	if seed := selectSeed(t, session, id); !reflect.DeepEqual(seed.SeedInput, replacement) {
		t.Fatalf("seed after the edit = %#v", seed.SeedInput)
	}
}

func decodeAsSeed(record []byte) error {
	_, _, err := decodeSeedRecord(record)
	return err
}

func TestSeedRecordSchemaMustMatchItsKind(t *testing.T) {
	seed := mustEncode(encodeSeedRecord(fullPhraseSeed(), "2026-09-21"))
	note := mustEncode(encodeNoteRecord(fullNote()))
	card := mustEncode(encodeCardRecord(fullCard()))
	if input, checkedOn, err := decodeSeedRecord(seed); err != nil || checkedOn != "2026-09-21" || !reflect.DeepEqual(input, func() SeedInput { v := fullPhraseSeed(); v.Label = ""; return v }()) {
		t.Fatalf("a valid seed record read as %#v, %q, %v", input, checkedOn, err)
	}
	empty := encodeBytes(nil)
	record := func(format uint64, codes, addresses []byte, checkedOn string) []byte {
		return encodeArray(encodeUint(recordSchemaSeed), encodeUint(format), encodeArray(), empty, empty, empty, codes, empty, addresses, encodeBytes([]byte(checkedOn)), empty)
	}
	code := func(used uint64) []byte { return encodeArray(encodeBytes([]byte("code")), encodeUint(used)) }
	tests := []struct {
		name   string
		decode func([]byte) error
		record []byte
		want   error
	}{
		{"note read as a seed", decodeAsSeed, note, ErrMalformed},
		{"card read as a seed", decodeAsSeed, card, ErrMalformed},
		{"seed read as a note", decodeAsNote, seed, ErrMalformed},
		{"seed read as a card", decodeAsCard, seed, ErrMalformed},
		{"seed read as a credential", decodeAsCredential, seed, ErrMalformed},
		{"seed read as an identity", decodeAsIdentity, seed, ErrMalformed},
		{"unknown schema read as a seed", decodeAsSeed, encodeArray(encodeUint(15), encodeUint(1), encodeArray(), empty, empty, empty, encodeArray(), empty, encodeArray(), empty, empty), ErrUnsupported},
		{"format zero", decodeAsSeed, record(0, encodeArray(), encodeArray(), ""), ErrMalformed},
		{"format past the known ones", decodeAsSeed, record(uint64(SeedBackupCodes)+1, encodeArray(), encodeArray(), ""), ErrUnsupported},
		{"used flag of two", decodeAsSeed, record(3, encodeArray(code(2)), encodeArray(), ""), ErrMalformed},
		{"code of one field", decodeAsSeed, record(3, encodeArray(encodeArray(encodeBytes([]byte("code")))), encodeArray(), ""), ErrMalformed},
		{"address of one field", decodeAsSeed, record(1, encodeArray(), encodeArray(encodeArray(encodeBytes([]byte("a")))), ""), ErrMalformed},
		{"impossible checked day", decodeAsSeed, record(1, encodeArray(), encodeArray(), "2026-02-30"), ErrMalformed},
		{"too many codes", decodeAsSeed, record(3, encodeArray(slices.Repeat([][]byte{code(0)}, MaxBackupCodes+1)...), encodeArray(), ""), ErrMalformed},
		{"one field short", decodeAsSeed, encodeArray(encodeUint(recordSchemaSeed), encodeUint(1), encodeArray(), empty, empty, empty, encodeArray(), empty, encodeArray(), empty), ErrMalformed},
		{"trailing bytes", decodeAsSeed, append(append([]byte(nil), seed...), 0), ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode(test.record); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func seedSummary(format, total, used uint64) []byte {
	return encodeArray(encodeUint(format), encodeUint(total), encodeUint(used))
}

func TestIndexHoldsASeedSummaryOnlyForASeed(t *testing.T) {
	records := stubRecords(1)
	digest := sha256.Sum256(encodeBox(records[0]))
	id := ID{6}
	seed := uint64(KindSeed)
	valid := seedSummary(uint64(SeedBackupCodes), 10, 3)
	_, _, entries, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), currentElement(id, digest, seed, valid)), records)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].kind != KindSeed || entries[0].seed != (SeedFace{Format: SeedBackupCodes, Total: 10, Used: 3}) {
		t.Fatalf("seed entry = %+v", entries[0])
	}
	tests := []struct {
		name    string
		element []byte
		want    error
	}{
		{"seed without a summary", currentElement(id, digest, seed, encodeArray()), ErrMalformed},
		{"note summary on a seed", currentElement(id, digest, seed, encodeArray(encodeUint(0))), ErrMalformed},
		{"summary of four fields", currentElement(id, digest, seed, encodeArray(encodeUint(1), encodeUint(12), encodeUint(0), encodeUint(0))), ErrMalformed},
		{"format zero", currentElement(id, digest, seed, seedSummary(0, 12, 0)), ErrMalformed},
		{"format past the known ones", currentElement(id, digest, seed, seedSummary(uint64(SeedBackupCodes)+1, 1, 0)), ErrUnsupported},
		{"phrase without words", currentElement(id, digest, seed, seedSummary(uint64(SeedPhrase), 0, 0)), ErrMalformed},
		{"phrase of too many words", currentElement(id, digest, seed, seedSummary(uint64(SeedPhrase), MaxSeedWords+1, 0)), ErrMalformed},
		{"phrase with used codes", currentElement(id, digest, seed, seedSummary(uint64(SeedPhrase), 12, 1)), ErrMalformed},
		{"key with a total", currentElement(id, digest, seed, seedSummary(uint64(SeedPrivateKey), 1, 0)), ErrMalformed},
		{"codes without a code", currentElement(id, digest, seed, seedSummary(uint64(SeedBackupCodes), 0, 0)), ErrMalformed},
		{"too many codes", currentElement(id, digest, seed, seedSummary(uint64(SeedBackupCodes), MaxBackupCodes+1, 0)), ErrMalformed},
		{"more used than held", currentElement(id, digest, seed, seedSummary(uint64(SeedBackupCodes), 3, 4)), ErrMalformed},
		{"huge total", currentElement(id, digest, seed, seedSummary(uint64(SeedBackupCodes), 1<<63, 0)), ErrMalformed},
		{"seed summary on a credential", currentElement(id, digest, uint64(KindCredential), valid), ErrMalformed},
		{"seed summary on an identity", currentElement(id, digest, uint64(KindIdentity), valid), ErrMalformed},
		{"seed with an expiry", withField(id, digest, seed, valid, fieldExpiresOn, encodeBytes([]byte("2030-01-01"))), ErrMalformed},
		{"seed with a thumbnail", withField(id, digest, seed, valid, fieldThumbnail, encodeBytes([]byte{0xff, 0xd8})), ErrMalformed},
		{"seed with a site", withField(id, digest, seed, valid, fieldSite, encodeBytes([]byte("example.com"))), ErrMalformed},
		{"seed with an owner", withField(id, digest, seed, valid, fieldOwner, encodeBytes(id[:])), ErrMalformed},
		{"seed with an email", withField(id, digest, seed, valid, fieldEmail, encodeBytes([]byte("a@example.test"))), ErrMalformed},
		{"seed with a card face", withField(id, digest, seed, valid, fieldCard, faceOf(1, "6789", "")), ErrMalformed},
		{"seed with a code", codeElement(id, digest, seed, valid, 6, 30), ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), test.element), records); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestSeedEntryThatDoesNotMatchItsRecordIsMalformed(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
	session := created.Session
	defer session.Lock()
	commitSeed(t, session, fullPhraseSeed(), nil)
	commitSeed(t, session, fullCodesSeed(), nil)
	commitNote(t, session, NoteInput{Label: "Note"}, nil)
	raw, head := currentRaw(t, session)
	tests := []struct {
		name   string
		mutate func([]entryMeta)
	}{
		{"other word count", func(entries []entryMeta) { entries[1].seed.Total = 13 }},
		{"other format", func(entries []entryMeta) { entries[1].seed = SeedFace{Format: SeedPrivateKey} }},
		{"other wallet", func(entries []entryMeta) { entries[1].detail = "Trezor" }},
		{"other code count", func(entries []entryMeta) { entries[2].seed.Total = 4 }},
		{"other used count", func(entries []entryMeta) { entries[2].seed.Used = 0 }},
		{"seed record listed as a note", func(entries []entryMeta) {
			entries[1].kind, entries[1].detail, entries[1].seed = KindNote, "", SeedFace{}
		}},
		{"note record listed as a seed", func(entries []entryMeta) {
			entries[3].kind, entries[3].seed = KindSeed, SeedFace{Format: SeedPrivateKey}
		}},
		{"seed record listed as a credential", func(entries []entryMeta) {
			entries[1].kind, entries[1].detail, entries[1].seed = KindCredential, "", SeedFace{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := append([]entryMeta(nil), session.entries...)
			test.mutate(entries)
			plaintext, err := encodeIndex(head.Revision, session.ancestry, entries, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := OpenWithRecovery(withIndex(t, session, raw, plaintext), created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestSeedRecordNotAsTheVaultStoresItIsMalformed(t *testing.T) {
	tests := []struct {
		name      string
		input     SeedInput
		checkedOn string
	}{
		{"words not normalized", func() SeedInput { v := fullPhraseSeed(); v.Words[0] = "ABANDON"; return v }(), ""},
		{"checked private key", fullKeySeed(), "2026-09-21"},
		{"blank code", func() SeedInput { v := fullCodesSeed(); v.Codes[0].Value = " "; return v }(), ""},
		{"phrase with a key", func() SeedInput { v := fullPhraseSeed(); v.Key = "key"; return v }(), ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			created, _ := vaultWith(t)
			session := created.Session
			id := commitSeed(t, session, SeedInput{Label: test.input.Label, Format: SeedPrivateKey, Key: "k", Wallet: test.input.Wallet}, nil)
			session.entries[session.find(id)].seed = seedFace(test.input)
			container := resealed(t, session, id, mustEncode(encodeSeedRecord(test.input, test.checkedOn)))
			session.Lock()
			if _, err := OpenWithRecovery(container, created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestCrossingKindsWithNotesAndSeedsIsRefusedAsNotFound(t *testing.T) {
	session, credential := populatedSession(t)
	defer session.Lock()
	identity := commitIdentity(t, session, fullIdentity(), nil)
	card := commitCard(t, session, fullCard(), nil)
	note := commitNote(t, session, fullNote(), nil)
	seed := commitSeed(t, session, fullCodesSeed(), nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	read := func(id ID, operation func(Selection) error) error {
		ticket, err := session.BeginSelection(id)
		if err != nil {
			t.Fatal(err)
		}
		return operation(ticket)
	}
	readNote := func(ticket Selection) error { _, err := session.ReadSelectedNote(ticket); return err }
	readSeed := func(ticket Selection) error { _, err := session.ReadSelectedSeed(ticket); return err }
	for _, other := range []ID{credential, identity, card, seed} {
		if err := read(other, readNote); !errors.Is(err, ErrNotFound) {
			t.Fatalf("note read of another kind: %v", err)
		}
		if _, err := session.PrepareEditNote(other, fullNote(), nil); !errors.Is(err, ErrNotFound) {
			t.Fatalf("note edit of another kind: %v", err)
		}
	}
	for _, other := range []ID{credential, identity, card, note} {
		if err := read(other, readSeed); !errors.Is(err, ErrNotFound) {
			t.Fatalf("seed read of another kind: %v", err)
		}
		if _, err := session.PrepareEditSeed(other, fullCodesSeed(), nil); !errors.Is(err, ErrNotFound) {
			t.Fatalf("seed edit of another kind: %v", err)
		}
		if _, err := session.PrepareUseBackupCode(other, 0); !errors.Is(err, ErrNotFound) {
			t.Fatalf("code use of another kind: %v", err)
		}
		if _, err := session.PrepareRecordSeedCheck(other, "2026-09-21"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("seed check of another kind: %v", err)
		}
	}
	password := "changed"
	for _, item := range []ID{note, seed} {
		if err := read(item, func(ticket Selection) error { _, err := session.ReadSelected(ticket); return err }); !errors.Is(err, ErrNotFound) {
			t.Fatalf("credential read of a note or seed: %v", err)
		}
		if err := read(item, func(ticket Selection) error { _, err := session.ReadSelectedIdentity(ticket); return err }); !errors.Is(err, ErrNotFound) {
			t.Fatalf("identity read of a note or seed: %v", err)
		}
		if err := read(item, func(ticket Selection) error { _, err := session.ReadSelectedCard(ticket); return err }); !errors.Is(err, ErrNotFound) {
			t.Fatalf("card read of a note or seed: %v", err)
		}
		if _, err := session.PrepareEdit(item, CredentialPatch{Password: &password}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("credential edit of a note or seed: %v", err)
		}
		if _, err := session.PrepareEditIdentity(item, fullIdentity(), nil); !errors.Is(err, ErrNotFound) {
			t.Fatalf("identity edit of a note or seed: %v", err)
		}
		if _, err := session.PrepareEditCard(item, fullCard(), nil); !errors.Is(err, ErrNotFound) {
			t.Fatalf("card edit of a note or seed: %v", err)
		}
		if _, err := session.ScansOf(item); !errors.Is(err, ErrNotFound) {
			t.Fatalf("scans of a note or seed: %v", err)
		}
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused crossing changed the vault")
	}
}

func TestNoteAndSeedValuesNeverReachTheContainerInPlaintext(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	hidden := NoteInput{Label: "Diary", Body: "Plaintextfirstline of the hidden note\nand a second plaintext line", Hidden: true}
	shown := NoteInput{Label: "Shopping", Body: "Shownfirstline preview text\nmore"}
	commitNote(t, session, hidden, nil)
	commitNote(t, session, shown, nil)
	phrase := SeedInput{Label: "Cold", Format: SeedPhrase, Words: []string{"uniquewordalpha", "uniquewordbravo"}, Passphrase: "passphrase-never-plain", Path: "m/44'/60'/7'", Wallet: "WalletNameUnique",
		Addresses: []SeedAddress{{Label: "LabelUnique", Value: "AddressValueUnique"}}, Notes: "SeedNotesUnique"}
	commitSeed(t, session, phrase, nil)
	commitSeed(t, session, fullKeySeed(), nil)
	commitSeed(t, session, fullCodesSeed(), nil)
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	values := []string{hidden.Label, hidden.Body, "Plaintextfirstline", "second plaintext line", shown.Label, "Shownfirstline preview text",
		"uniquewordalpha", "uniquewordbravo", phrase.Passphrase, phrase.Path, phrase.Wallet, "LabelUnique", "AddressValueUnique", phrase.Notes,
		fullKeySeed().Key, "1MainAddressExample", "code-alpha-7741", "code-bravo-9902", "code-charlie-3318", "Exchange account"}
	for _, value := range values {
		if bytes.Contains(container, []byte(value)) {
			t.Fatalf("%q appears in the vault file", value)
		}
	}
}

func TestListingSeedsDecryptsNoRecord(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	id := commitSeed(t, session, fullCodesSeed(), nil)
	for i := range session.records {
		tampered := append([]byte(nil), session.records[i].ciphertext...)
		tampered[0] ^= 1
		session.records[i].ciphertext = tampered
	}
	if entry := listedEntry(t, session, id); entry.Kind != KindSeed || entry.Detail != "Exchange account" || entry.Seed.Used != 1 {
		t.Fatalf("seed entry over unreadable records = %+v", entry)
	}
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadSelectedSeed(ticket); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("reading an unreadable record: %v", err)
	}
}

func TestLockedSessionRefusesSeedOperations(t *testing.T) {
	session, _ := populatedSession(t)
	id := commitSeed(t, session, fullCodesSeed(), nil)
	session.Lock()
	if _, _, err := session.PrepareCreateSeed(fullCodesSeed(), nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareCreateSeed: %v", err)
	}
	if _, err := session.PrepareEditSeed(id, fullCodesSeed(), nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareEditSeed: %v", err)
	}
	if _, err := session.PrepareUseBackupCode(id, 0); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareUseBackupCode: %v", err)
	}
	if _, err := session.PrepareRecordSeedCheck(id, "2026-09-21"); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareRecordSeedCheck: %v", err)
	}
	if _, err := session.ReadSelectedSeed(Selection{ID: id}); !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("ReadSelectedSeed: %v", err)
	}
}

func TestSeedsTravelThroughExport(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
	session := created.Session
	defer session.Lock()
	seed := commitSeed(t, session, fullPhraseSeed(), nil)
	note := commitNote(t, session, fullNote(), nil)
	pending, err := session.PrepareRecordSeedCheck(seed, "2026-09-21")
	commitPending(t, session, pending, err)
	exported, _, err := session.Export()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := OpenWithRecovery(exported, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Lock()
	if value := selectSeed(t, restored, seed); !reflect.DeepEqual(value.SeedInput, fullPhraseSeed()) || value.CheckedOn != "2026-09-21" {
		t.Fatalf("exported seed = %#v", value)
	}
	if value := selectNote(t, restored, note); !reflect.DeepEqual(value.NoteInput, fullNote()) {
		t.Fatalf("exported note = %#v", value.NoteInput)
	}
}
