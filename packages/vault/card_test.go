package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func fullCard() CardInput {
	return CardInput{
		Label:        "  Everyday  ",
		Holder:       "Алекс Ünïcødé Example",
		Number:       "4111111111111111",
		Expiry:       "2029-08",
		SecurityCode: "7391",
		PIN:          "482913",
		Network:      NetworkVisa,
		BankName:     "Example Bank",
		BankSite:     "https://www.Bank.Example/private",
		Color:        "#00a0e1",
		Billing:      &Address{Street: "Vestergade 8-16", City: "Silkeborg", PostalCode: "8600", Country: "Danmark"},
		Notes:        " line 1\nline 2 ",
	}
}

func commitCard(t *testing.T, session *Session, input CardInput, groups []ID) ID {
	t.Helper()
	pending, id, err := session.PrepareCreateCard(input, groups)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	return id
}

func selectCard(t *testing.T, session *Session, id ID) Card {
	t.Helper()
	ticket, err := session.BeginSelection(id)
	if err != nil {
		t.Fatal(err)
	}
	value, err := session.ReadSelectedCard(ticket)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

// cardElement is an entry in the current form with the given expiry and card field.
func cardElement(id ID, digest [32]byte, kind uint64, expiresOn string, face []byte) []byte {
	fields := currentFields(id, digest, kind, encodeArray())
	fields[fieldExpiresOn] = encodeBytes([]byte(expiresOn))
	fields[fieldCard] = face
	return encodeArray(fields...)
}

// cardWithField is a current-form card entry with the given face and field position replaced by value.
func cardWithField(id ID, digest [32]byte, face []byte, position int, value []byte) []byte {
	fields := currentFields(id, digest, uint64(KindCard), encodeArray())
	fields[fieldCard] = face
	fields[position] = value
	return encodeArray(fields...)
}

func faceOf(network uint64, lastFour, color string) []byte {
	return encodeArray(encodeUint(network), encodeBytes([]byte(lastFour)), encodeBytes([]byte(color)))
}

func TestCardRoundTripsEveryValue(t *testing.T) {
	created, ids := vaultWith(t, CredentialInput{Label: "Mail", Login: "alex", Password: "secret"})
	input := fullCard()
	id := commitCard(t, created.Session, input, nil)
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
	card := selectCard(t, reopened, id)
	if card.ID != id || !reflect.DeepEqual(card.CardInput, input) {
		t.Fatalf("card changed across a reopen: %#v", card.CardInput)
	}
	want := Entry{ID: id, Kind: KindCard, Label: input.Label, Detail: "Example Bank", ExpiresOn: "2029-08-31", Site: "bank.example", Card: CardFace{Network: NetworkVisa, LastFour: "1111", Color: "#00a0e1"}}
	if entry := listedEntry(t, reopened, id); !reflect.DeepEqual(entry, want) {
		t.Fatalf("card entry = %+v", entry)
	}
	if credential := listedEntry(t, reopened, ids[0]); credential.Card != (CardFace{}) {
		t.Fatalf("credential entry carries a card face: %+v", credential)
	}
}

func TestCardWithoutOptionalValuesHasABareEntry(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	input := CardInput{Label: "Spare", Number: "000000000000"}
	id := commitCard(t, session, input, nil)
	if card := selectCard(t, session, id); !reflect.DeepEqual(card.CardInput, input) {
		t.Fatalf("bare card = %#v", card.CardInput)
	}
	want := Entry{ID: id, Kind: KindCard, Label: "Spare", Card: CardFace{LastFour: "0000"}}
	if entry := listedEntry(t, session, id); !reflect.DeepEqual(entry, want) {
		t.Fatalf("bare card entry = %+v", entry)
	}
}

func TestCardExpiryEndsOnTheLastDayOfItsMonth(t *testing.T) {
	tests := map[string]string{"2028-02": "2028-02-29", "2029-02": "2029-02-28", "2030-12": "2030-12-31", "1900-04": "1900-04-30", "": ""}
	for expiry, want := range tests {
		if got := expiryEnd(expiry); got != want {
			t.Fatalf("end of %q = %q, want %q", expiry, got, want)
		}
	}
}

func TestDigitsCountsASCIIDigits(t *testing.T) {
	tests := []struct {
		value       string
		least, most int
		want        bool
	}{
		{"123", 3, 4, true},
		{"1234", 3, 4, true},
		{"12", 3, 4, false},
		{"12345", 3, 4, false},
		{"12a", 3, 4, false},
		{" 123", 3, 4, false},
		{"١٢٣", 1, 12, false},
		{"", 0, 4, true},
	}
	for _, test := range tests {
		if got := Digits(test.value, test.least, test.most); got != test.want {
			t.Errorf("Digits(%q, %d, %d) = %v", test.value, test.least, test.most, got)
		}
	}
}

func TestValidDateAndExpiryTakeTheirStoredForms(t *testing.T) {
	dates := map[string]bool{"": true, "2028-02-29": true, "1900-01-01": true, "2029-02-29": false, "1899-12-31": false, "2028-2-3": false, "2028-02": false}
	for value, want := range dates {
		if got := ValidDate(value); got != want {
			t.Errorf("ValidDate(%q) = %v", value, got)
		}
	}
	expiries := map[string]bool{"": true, "2030-12": true, "1900-01": true, "2030-13": false, "0000-01": false, "2030-1": false, "2030-12-31": false}
	for value, want := range expiries {
		if got := ValidExpiry(value); got != want {
			t.Errorf("ValidExpiry(%q) = %v", value, got)
		}
	}
}

func TestCardAtEveryLimitRoundTrips(t *testing.T) {
	at := func(limit int) string { return strings.Repeat("ü", limit) }
	inputs := []CardInput{
		{Label: at(MaxLabelLength), Holder: at(MaxCardHolderLength), Number: strings.Repeat("9", MaxCardNumberLength), Expiry: "1900-01",
			SecurityCode: "1234", PIN: strings.Repeat("7", MaxCardPINLength), Network: NetworkNaranja, BankName: at(MaxBankNameLength),
			BankSite: strings.Repeat("a", MaxOriginLength), Color: "#ffffff", Notes: at(MaxNotesLength)},
		{Label: "Short", Number: strings.Repeat("1", MinCardNumberLength), SecurityCode: "123", PIN: "1234", Color: "#000000"},
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	for _, input := range inputs {
		id := commitCard(t, session, input, nil)
		if card := selectCard(t, session, id); !reflect.DeepEqual(card.CardInput, input) {
			t.Fatalf("a card at its limits changed on its way through the vault: %#v", card.CardInput)
		}
	}
}

func TestCardRefusesEachBoundWithoutWriting(t *testing.T) {
	over := func(limit int) string { return strings.Repeat("ü", limit+1) }
	tests := []struct {
		name   string
		mutate func(*CardInput)
	}{
		{"empty label", func(input *CardInput) { input.Label = "" }},
		{"blank label", func(input *CardInput) { input.Label = " \t\n" }},
		{"label over its limit", func(input *CardInput) { input.Label = over(MaxLabelLength) }},
		{"label that is not text", func(input *CardInput) { input.Label = "\xff" }},
		{"holder over its limit", func(input *CardInput) { input.Holder = over(MaxCardHolderLength) }},
		{"notes over their limit", func(input *CardInput) { input.Notes = over(MaxNotesLength) }},
		{"bank name over its limit", func(input *CardInput) { input.BankName = over(MaxBankNameLength) }},
		{"bank site over its limit", func(input *CardInput) { input.BankSite = strings.Repeat("a", MaxOriginLength+1) }},
		{"empty number", func(input *CardInput) { input.Number = "" }},
		{"number with spaces", func(input *CardInput) { input.Number = "4111 1111 1111 1111" }},
		{"number with a letter", func(input *CardInput) { input.Number = "411111111111111x" }},
		{"number with other digits", func(input *CardInput) { input.Number = "٤١١١١١١١١١١١١١١١" }},
		{"number too short", func(input *CardInput) { input.Number = strings.Repeat("4", MinCardNumberLength-1) }},
		{"number too long", func(input *CardInput) { input.Number = strings.Repeat("4", MaxCardNumberLength+1) }},
		{"thirteenth month", func(input *CardInput) { input.Expiry = "2029-13" }},
		{"month zero", func(input *CardInput) { input.Expiry = "2029-00" }},
		{"expiry with a day", func(input *CardInput) { input.Expiry = "2029-08-31" }},
		{"expiry in another form", func(input *CardInput) { input.Expiry = "08/29" }},
		{"expiry with a one-digit month", func(input *CardInput) { input.Expiry = "2029-8" }},
		{"expiry before 1900", func(input *CardInput) { input.Expiry = "1899-12" }},
		{"expiry in year zero", func(input *CardInput) { input.Expiry = "0000-01" }},
		{"security code too short", func(input *CardInput) { input.SecurityCode = "12" }},
		{"security code too long", func(input *CardInput) { input.SecurityCode = "12345" }},
		{"security code with a letter", func(input *CardInput) { input.SecurityCode = "12a" }},
		{"PIN too short", func(input *CardInput) { input.PIN = "123" }},
		{"PIN too long", func(input *CardInput) { input.PIN = strings.Repeat("1", MaxCardPINLength+1) }},
		{"PIN with a space", func(input *CardInput) { input.PIN = "12 34" }},
		{"unknown network", func(input *CardInput) { input.Network = NetworkNaranja + 1 }},
		{"uppercase colour", func(input *CardInput) { input.Color = "#00A0E1" }},
		{"short colour", func(input *CardInput) { input.Color = "#0ae" }},
		{"colour without its mark", func(input *CardInput) { input.Color = "00a0e1" }},
		{"named colour", func(input *CardInput) { input.Color = "skyblue" }},
		{"billing address with a label", func(input *CardInput) { input.Billing.Label = "Home" }},
		{"billing address with an id", func(input *CardInput) { input.Billing.ID = ID{1} }},
		{"billing address without a part", func(input *CardInput) { input.Billing = &Address{Street: " "} }},
		{"billing street over its limit", func(input *CardInput) { input.Billing.Street = over(MaxStreetLength) }},
		{"billing country over its limit", func(input *CardInput) { input.Billing.Country = over(MaxCountryLength) }},
		{"billing address and a link", func(input *CardInput) { input.BillingLink = &AddressLink{Identity: ID{1}, Address: ID{2}} }},
		{"link to no identity", func(input *CardInput) {
			input.Billing, input.BillingLink = nil, &AddressLink{Identity: ID{0xee}, Address: ID{2}}
		}},
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	existing := commitCard(t, session, fullCard(), nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := fullCard()
			test.mutate(&input)
			if _, _, err := session.PrepareCreateCard(input, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("create: got %v, want ErrInvalidInput", err)
			}
			if _, err := session.PrepareEditCard(existing, input, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("edit: got %v, want ErrInvalidInput", err)
			}
		})
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused card changed the vault")
	}
	if card := selectCard(t, session, existing); !reflect.DeepEqual(card.CardInput, fullCard()) {
		t.Fatalf("a refused edit changed the card: %#v", card.CardInput)
	}
}

func decodeAsCard(record []byte) error {
	_, err := decodeCardRecord(record)
	return err
}

func TestCardRecordSchemaMustMatchItsKind(t *testing.T) {
	card := mustEncode(encodeCardRecord(fullCard()))
	identity, err := encodeIdentityRecord(fullIdentity())
	if err != nil {
		t.Fatal(err)
	}
	credential, err := encodeCredentialRecord(CredentialInput{Label: "Mail", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	unknownNetwork := fullCard()
	unknownNetwork.Network = NetworkNaranja + 1
	empty := encodeBytes(nil)
	cardRecord := func(billing, link []byte) []byte {
		return encodeArray(encodeUint(recordSchemaCard), empty, empty, empty, empty, empty, encodeUint(0), empty, empty, empty, billing, link, empty)
	}
	address := encodeArray(empty, encodeBytes([]byte("Street")), empty, empty, empty, empty)
	link := encodeArray(encodeBytes(bytes.Repeat([]byte{1}, 16)), encodeBytes(bytes.Repeat([]byte{2}, 16)))
	unknownSchema := encodeArray(encodeUint(15), empty, empty, empty, empty, empty, encodeUint(0), empty, empty, empty, encodeArray(), encodeArray(), empty)
	short := encodeArray(encodeUint(recordSchemaCard), empty, empty, empty, empty, empty, encodeUint(0), empty, empty, empty, encodeArray(), empty)
	for _, record := range [][]byte{cardRecord(encodeArray(), encodeArray()), cardRecord(encodeArray(address), encodeArray()), cardRecord(encodeArray(), link)} {
		if _, err := decodeCardRecord(record); err != nil {
			t.Fatalf("a valid card record was refused: %v", err)
		}
	}
	if _, err := decodeCardRecord(card); err != nil {
		t.Fatalf("a valid card record was refused: %v", err)
	}
	tests := []struct {
		name   string
		decode func([]byte) error
		record []byte
		want   error
	}{
		{"identity read as a card", decodeAsCard, identity, ErrMalformed},
		{"credential read as a card", decodeAsCard, credential, ErrMalformed},
		{"card read as an identity", decodeAsIdentity, card, ErrMalformed},
		{"card read as a credential", decodeAsCredential, card, ErrMalformed},
		{"unknown schema read as a card", decodeAsCard, unknownSchema, ErrUnsupported},
		{"network past the known ones", decodeAsCard, mustEncode(encodeCardRecord(unknownNetwork)), ErrUnsupported},
		{"one field short", decodeAsCard, short, ErrMalformed},
		{"billing address and a link", decodeAsCard, cardRecord(encodeArray(address), link), ErrMalformed},
		{"two billing addresses", decodeAsCard, cardRecord(encodeArray(address, address), encodeArray()), ErrMalformed},
		{"billing address with an id", decodeAsCard, cardRecord(encodeArray(encodeArray(encodeBytes(bytes.Repeat([]byte{1}, 16)), empty, empty, empty, empty, empty, empty)), encodeArray()), ErrMalformed},
		{"link of one id", decodeAsCard, cardRecord(encodeArray(), encodeArray(encodeBytes(bytes.Repeat([]byte{1}, 16)))), ErrMalformed},
		{"link with a short id", decodeAsCard, cardRecord(encodeArray(), encodeArray(encodeBytes(bytes.Repeat([]byte{1}, 16)), encodeBytes([]byte{2}))), ErrMalformed},
		{"trailing bytes", decodeAsCard, append(append([]byte(nil), card...), 0), ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode(test.record); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestIndexHoldsACardFaceOnlyForACard(t *testing.T) {
	records := stubRecords(1)
	digest := sha256.Sum256(encodeBox(records[0]))
	id := ID{4}
	card := uint64(KindCard)
	valid := faceOf(uint64(NetworkMir), "6789", "#00a0e1")
	_, _, index, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), cardElement(id, digest, card, "2029-08-31", valid)), records)
	entries := index.entries
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].kind != KindCard || entries[0].card != (CardFace{Network: NetworkMir, LastFour: "6789", Color: "#00a0e1"}) || entries[0].expiresOn != "2029-08-31" {
		t.Fatalf("card entry = %+v", entries[0])
	}
	tests := []struct {
		name    string
		element []byte
		want    error
	}{
		{"card without a face", cardElement(id, digest, card, "", encodeArray()), ErrMalformed},
		{"face on a credential", cardElement(id, digest, uint64(KindCredential), "", valid), ErrMalformed},
		{"face on an identity", cardElement(id, digest, uint64(KindIdentity), "", valid), ErrMalformed},
		{"face of two fields", cardElement(id, digest, card, "", encodeArray(encodeUint(1), encodeBytes([]byte("6789")))), ErrMalformed},
		{"face of four fields", cardElement(id, digest, card, "", encodeArray(encodeUint(1), encodeBytes([]byte("6789")), encodeBytes(nil), encodeBytes(nil))), ErrMalformed},
		{"network past the known ones", cardElement(id, digest, card, "", faceOf(uint64(NetworkNaranja)+1, "6789", "")), ErrUnsupported},
		{"three digits", cardElement(id, digest, card, "", faceOf(1, "789", "")), ErrMalformed},
		{"five digits", cardElement(id, digest, card, "", faceOf(1, "56789", "")), ErrMalformed},
		{"a letter among the digits", cardElement(id, digest, card, "", faceOf(1, "67x9", "")), ErrMalformed},
		{"uppercase colour", cardElement(id, digest, card, "", faceOf(1, "6789", "#00A0E1")), ErrMalformed},
		{"short colour", cardElement(id, digest, card, "", faceOf(1, "6789", "#0ae")), ErrMalformed},
		{"impossible expiry", cardElement(id, digest, card, "2029-02-30", valid), ErrMalformed},
		{"card with a thumbnail", cardWithField(id, digest, valid, fieldThumbnail, encodeBytes([]byte{0xff, 0xd8})), ErrMalformed},
		{"card with an email", cardWithField(id, digest, valid, fieldEmail, encodeBytes([]byte("a@example.test"))), ErrMalformed},
		{"card with an owner", cardWithField(id, digest, valid, fieldOwner, encodeBytes(id[:])), ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, err := parseIndex(indexPlaintext(1, [32]byte{}, encodeArray(), test.element), records); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestCardEntryThatDoesNotMatchItsRecordIsMalformed(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
	session := created.Session
	defer session.Lock()
	commitIdentity(t, session, IdentityInput{Label: "Me", FullName: "Alex"}, nil)
	commitCard(t, session, fullCard(), nil)
	raw, head := currentRaw(t, session)
	tests := []struct {
		name   string
		mutate func([]entryMeta)
	}{
		{"card record listed as a credential", func(entries []entryMeta) {
			entries[2].kind, entries[2].detail, entries[2].expiresOn, entries[2].site = KindCredential, "", "", ""
		}},
		{"card record listed as an identity", func(entries []entryMeta) { entries[2].kind, entries[2].site = KindIdentity, "" }},
		{"identity record listed as a card", func(entries []entryMeta) {
			entries[1].kind, entries[1].card = KindCard, CardFace{LastFour: "0000"}
		}},
		{"other last four digits", func(entries []entryMeta) { entries[2].card.LastFour = "2222" }},
		{"other network", func(entries []entryMeta) { entries[2].card.Network = NetworkMir }},
		{"other colour", func(entries []entryMeta) { entries[2].card.Color = "" }},
		{"other bank name", func(entries []entryMeta) { entries[2].detail = "Other Bank" }},
		{"other expiry", func(entries []entryMeta) { entries[2].expiresOn = "2029-08-30" }},
		{"other site", func(entries []entryMeta) { entries[2].site = "example.com" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := append([]entryMeta(nil), session.entries...)
			test.mutate(entries)
			plaintext, err := encodeIndex(head.Revision, session.ancestry, entries, nil, DefaultTrashRetention)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := OpenWithRecovery(withIndex(t, session, raw, plaintext), created.RecoveryPhrase); !errors.Is(err, ErrMalformed) {
				t.Fatalf("got %v, want ErrMalformed", err)
			}
		})
	}
}

func TestCrossingKindsWithACardIsRefusedAsNotFound(t *testing.T) {
	session, credential := populatedSession(t)
	defer session.Lock()
	identity := commitIdentity(t, session, fullIdentity(), nil)
	card := commitCard(t, session, fullCard(), nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for _, other := range []ID{credential, identity} {
		ticket, err := session.BeginSelection(other)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := session.ReadSelectedCard(ticket); !errors.Is(err, ErrNotFound) {
			t.Fatalf("card read of another kind: %v", err)
		}
		if _, err := session.PrepareEditCard(other, fullCard(), nil); !errors.Is(err, ErrNotFound) {
			t.Fatalf("card edit of another kind: %v", err)
		}
	}
	ticket, err := session.BeginSelection(card)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadSelected(ticket); !errors.Is(err, ErrNotFound) {
		t.Fatalf("credential read of a card: %v", err)
	}
	if _, err := session.ReadSelectedIdentity(ticket); !errors.Is(err, ErrNotFound) {
		t.Fatalf("identity read of a card: %v", err)
	}
	password := "changed"
	if _, err := session.PrepareEdit(card, CredentialPatch{Password: &password}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("credential edit of a card: %v", err)
	}
	if _, err := session.PrepareEditIdentity(card, fullIdentity(), nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("identity edit of a card: %v", err)
	}
	if _, err := session.ScansOf(card); !errors.Is(err, ErrNotFound) {
		t.Fatalf("scans of a card: %v", err)
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused crossing changed the vault")
	}
	if value := selectCard(t, session, card); !reflect.DeepEqual(value.CardInput, fullCard()) {
		t.Fatalf("card after refused crossings = %#v", value.CardInput)
	}
}

func TestItemOperationsActOnCards(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family", "Travel")
	session := created.Session
	defer session.Lock()
	id := commitCard(t, session, fullCard(), []ID{groupIDs[0]})
	raw, _ := currentRaw(t, session)
	sealed := encodeBox(raw.records[0])

	pending, err := session.PrepareSetPinned(id, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	pending, err = session.PrepareSetGroups(id, []ID{groupIDs[1], groupIDs[0]})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	entry := listedEntry(t, session, id)
	if !entry.Pinned || !reflect.DeepEqual(entry.Groups, sortedIDs(groupIDs...)) || entry.Kind != KindCard {
		t.Fatalf("card entry after pin and groups = %+v", entry)
	}
	next, _ := currentRaw(t, session)
	if !bytes.Equal(encodeBox(next.records[0]), sealed) {
		t.Fatal("pinning or grouping sealed the card again")
	}
	pending, err = session.PrepareDelete(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if entries, err := session.List(); err != nil || len(entries) != 0 {
		t.Fatalf("entries after deleting the card = %+v, error = %v", entries, err)
	}
}

func TestPrepareEditCardReplacesContentAndMembershipAndKeepsPin(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family", "Travel")
	session := created.Session
	defer session.Lock()
	credential := commitCredential(t, session, CredentialInput{Label: "Mail", Password: "secret"}, nil)
	id := commitCard(t, session, fullCard(), []ID{groupIDs[0]})
	pending, err := session.PrepareSetPinned(id, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	raw, _ := currentRaw(t, session)
	replacement := CardInput{Label: "Travel card", Number: "2200123412341234", Network: NetworkMir}
	pending, err = session.PrepareEditCard(id, replacement, []ID{groupIDs[1]})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	want := Entry{ID: id, Kind: KindCard, Label: "Travel card", Card: CardFace{Network: NetworkMir, LastFour: "1234"}, Pinned: true, Groups: []ID{groupIDs[1]}}
	if entry := listedEntry(t, session, id); !reflect.DeepEqual(entry, want) {
		t.Fatalf("card entry after the edit = %+v", entry)
	}
	if session.entries[1].revision != 2 {
		t.Fatalf("card revision after one edit = %d", session.entries[1].revision)
	}
	if value := selectCard(t, session, id); !reflect.DeepEqual(value.CardInput, replacement) {
		t.Fatalf("card after the edit = %#v", value.CardInput)
	}
	next, _ := currentRaw(t, session)
	if !bytes.Equal(encodeBox(next.records[0]), encodeBox(raw.records[0])) {
		t.Fatal("editing a card sealed another record again")
	}
	if value := selectCredential(t, session, credential); value.Password != "secret" {
		t.Fatalf("credential after a card edit = %+v", value.CredentialInput)
	}
}

func TestCardValuesNeverReachTheContainerInPlaintext(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	input := fullCard()
	commitCard(t, session, input, nil)
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{input.Label, input.Holder, input.Number, input.Expiry, input.SecurityCode, input.PIN, input.BankName, input.BankSite, input.Color, input.Notes, input.Billing.Street, input.Billing.City, "1111", "bank.example", "2029-08-31"} {
		if bytes.Contains(container, []byte(value)) {
			t.Fatalf("%q appears in the vault file", value)
		}
	}
}

func TestCardsTravelThroughExport(t *testing.T) {
	created, _ := vaultWith(t, CredentialInput{Label: "Mail", Password: "secret"})
	session := created.Session
	defer session.Lock()
	id := commitCard(t, session, fullCard(), nil)
	exported, _, err := session.Export()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := OpenWithRecovery(exported, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Lock()
	if value := selectCard(t, restored, id); !reflect.DeepEqual(value.CardInput, fullCard()) {
		t.Fatalf("exported card = %#v", value.CardInput)
	}
}

func TestListingCardsDecryptsNoRecord(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	card := commitCard(t, session, fullCard(), nil)
	for i := range session.records {
		tampered := append([]byte(nil), session.records[i].ciphertext...)
		tampered[0] ^= 1
		session.records[i].ciphertext = tampered
	}
	if entry := listedEntry(t, session, card); entry.Kind != KindCard || entry.Detail != "Example Bank" || entry.Card.LastFour != "1111" {
		t.Fatalf("card entry over unreadable records = %+v", entry)
	}
	ticket, err := session.BeginSelection(card)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadSelectedCard(ticket); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("reading an unreadable record: %v", err)
	}
}

func TestLockedSessionRefusesCardOperations(t *testing.T) {
	session, _ := populatedSession(t)
	id := commitCard(t, session, fullCard(), nil)
	session.Lock()
	if _, _, err := session.PrepareCreateCard(fullCard(), nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareCreateCard: %v", err)
	}
	if _, err := session.PrepareEditCard(id, fullCard(), nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("PrepareEditCard: %v", err)
	}
	if _, err := session.ReadSelectedCard(Selection{ID: id}); !errors.Is(err, ErrStaleSelection) {
		t.Fatalf("ReadSelectedCard: %v", err)
	}
	if _, err := session.IdentityAddresses(); !errors.Is(err, ErrLocked) {
		t.Fatalf("IdentityAddresses: %v", err)
	}
}

func homeAndWork() IdentityInput {
	return IdentityInput{Label: "Alex", Addresses: []Address{
		{Label: "Home", Street: "1 Example Street", City: "Springfield"},
		{Label: "Work", Street: "2 Office Road", City: "Shelbyville"},
	}}
}

// linkedCard is a card whose billing address is the address of owner at position.
func linkedCard(t *testing.T, session *Session, owner ID, position int, label string) CardInput {
	t.Helper()
	input := fullCard()
	input.Label, input.Billing = label, nil
	input.BillingLink = &AddressLink{Identity: owner, Address: selectIdentity(t, session, owner).Addresses[position].ID}
	return input
}

func TestCardLinkResolvesOnRead(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	owner := commitIdentity(t, session, homeAndWork(), nil)
	input := linkedCard(t, session, owner, 1, "Linked")
	id := commitCard(t, session, input, nil)
	stored := selectIdentity(t, session, owner).Addresses[1]
	card := selectCard(t, session, id)
	if !reflect.DeepEqual(card.CardInput, input) || card.Linked == nil || *card.Linked != stored {
		t.Fatalf("linked card = %#v, linked = %#v", card.CardInput, card.Linked)
	}
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	session.Lock()
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if card := selectCard(t, reopened, id); card.Linked == nil || *card.Linked != stored {
		t.Fatalf("linked address across a reopen = %#v", card.Linked)
	}
	if own := selectCard(t, reopened, commitCard(t, reopened, fullCard(), nil)); own.Linked != nil {
		t.Fatalf("a card with its own address resolved a link: %#v", own.Linked)
	}
}

func TestCardLinkMustNameAnIdentityAddress(t *testing.T) {
	session, credential := populatedSession(t)
	defer session.Lock()
	owner := commitIdentity(t, session, homeAndWork(), nil)
	other := commitIdentity(t, session, IdentityInput{Label: "Other", Addresses: []Address{{City: "Berlin"}}}, nil)
	existing := commitCard(t, session, fullCard(), nil)
	held := selectIdentity(t, session, owner).Addresses[0].ID
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for name, link := range map[string]AddressLink{
		"an address the identity does not hold": {Identity: owner, Address: ID{0xee}},
		"another identity's address":            {Identity: other, Address: held},
		"a credential":                          {Identity: credential, Address: held},
		"a card":                                {Identity: existing, Address: held},
		"no identity":                           {Address: held},
	} {
		input := fullCard()
		input.Billing, input.BillingLink = nil, &link
		if _, _, err := session.PrepareCreateCard(input, nil); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("create with a link to %s: %v", name, err)
		}
		if _, err := session.PrepareEditCard(existing, input, nil); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("edit with a link to %s: %v", name, err)
		}
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused link changed the vault")
	}
}

func TestDeletingAnIdentityDropsItsCardLinksInTheSameSave(t *testing.T) {
	created, groupIDs := groupedVault(t, "Family")
	session := created.Session
	defer session.Lock()
	owner := commitIdentity(t, session, homeAndWork(), nil)
	other := commitIdentity(t, session, homeAndWork(), nil)
	linked := commitCard(t, session, linkedCard(t, session, owner, 0, "Linked"), []ID{groupIDs[0]})
	elsewhere := commitCard(t, session, linkedCard(t, session, other, 1, "Elsewhere"), nil)
	own := commitCard(t, session, fullCard(), nil)
	pending, err := session.PrepareSetPinned(linked, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	raw, head := currentRaw(t, session)
	sealed := map[ID][]byte{}
	for _, id := range []ID{elsewhere, own} {
		sealed[id] = encodeBox(raw.records[session.find(id)])
	}
	pending, err = session.PrepareDelete(owner)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if _, next := currentRaw(t, session); next.Revision != head.Revision+1 {
		t.Fatalf("the delete took %d saves", next.Revision-head.Revision)
	}
	card := selectCard(t, session, linked)
	want := linkedCard(t, session, other, 0, "Linked")
	want.BillingLink = nil
	if !reflect.DeepEqual(card.CardInput, want) || card.Linked != nil {
		t.Fatalf("card after its identity was deleted = %#v", card.CardInput)
	}
	if entry := listedEntry(t, session, linked); !entry.Pinned || !reflect.DeepEqual(entry.Groups, []ID{groupIDs[0]}) {
		t.Fatalf("unlinked card entry = %+v", entry)
	}
	if session.entries[session.find(linked)].revision != 2 {
		t.Fatal("the unlinked card was not written as a new revision")
	}
	next, _ := currentRaw(t, session)
	for id, before := range sealed {
		if !bytes.Equal(encodeBox(next.records[session.find(id)]), before) {
			t.Fatal("a card without a link to the identity was sealed again")
		}
	}
	if card := selectCard(t, session, elsewhere); card.Linked == nil {
		t.Fatal("a link to another identity was dropped")
	}
	if err := session.VerifyAll(); err != nil {
		t.Fatal(err)
	}
}

func TestRemovingALinkedAddressDropsItsCardLinks(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	owner := commitIdentity(t, session, homeAndWork(), nil)
	home := commitCard(t, session, linkedCard(t, session, owner, 0, "Home card"), nil)
	work := commitCard(t, session, linkedCard(t, session, owner, 1, "Work card"), nil)
	edited := selectIdentity(t, session, owner).IdentityInput
	edited.Addresses = []Address{edited.Addresses[1], {Label: "New", City: "Capital City"}}
	pending, err := session.PrepareEditIdentity(owner, edited, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if card := selectCard(t, session, home); card.BillingLink != nil || card.Linked != nil || session.entries[session.find(home)].revision != 2 {
		t.Fatalf("a card linked to a removed address = %#v", card.CardInput)
	}
	if card := selectCard(t, session, work); card.Linked == nil || card.Linked.Label != "Work" || session.entries[session.find(work)].revision != 1 {
		t.Fatalf("a card linked to a kept address = %#v", card.Linked)
	}
}

// resealed returns the container with item id's record resealed at its revision around plaintext.
func resealed(t *testing.T, session *Session, id ID, plaintext []byte) []byte {
	t.Helper()
	raw, head := currentRaw(t, session)
	index := session.find(id)
	box, err := seal(session.recordKey, plaintext, recordAAD(raw.vaultID, id, session.entries[index].revision))
	if err != nil {
		t.Fatal(err)
	}
	entries := append([]entryMeta(nil), session.entries...)
	raw.records = append([]sealedBox(nil), raw.records...)
	raw.records[index] = box
	entries[index].digest = sha256.Sum256(encodeBox(box))
	indexPlaintext, err := encodeIndex(head.Revision, session.ancestry, entries, session.groups, session.retention)
	if err != nil {
		t.Fatal(err)
	}
	return withIndex(t, session, raw, indexPlaintext)
}

func TestDanglingLinkReadsAsNoBillingAddress(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	owner := commitIdentity(t, session, homeAndWork(), nil)
	input := linkedCard(t, session, owner, 0, "Dangling")
	id := commitCard(t, session, input, nil)
	input.BillingLink.Address = ID{0xee}
	container := resealed(t, session, id, mustEncode(encodeCardRecord(input)))
	session.Lock()
	reopened, err := OpenWithRecovery(container, created.RecoveryPhrase)
	if err != nil {
		t.Fatalf("a dangling link was refused: %v", err)
	}
	defer reopened.Lock()
	card := selectCard(t, reopened, id)
	if card.BillingLink != nil || card.Linked != nil || card.Billing != nil {
		t.Fatalf("a dangling link read as %#v", card.CardInput)
	}
	pending, err := reopened.PrepareEditCard(id, card.CardInput, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Commit(pending); err != nil {
		t.Fatal(err)
	}
	plaintext, err := reopened.openRecord(reopened.find(id))
	if err != nil {
		t.Fatal(err)
	}
	if stored, err := decodeCardRecord(plaintext); err != nil || stored.BillingLink != nil {
		t.Fatalf("the dangling link survived the next save: %#v, error = %v", stored.BillingLink, err)
	}
}

func TestIdentityAddressesListsWhatACardCanLinkTo(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	first := commitIdentity(t, session, homeAndWork(), nil)
	commitIdentity(t, session, IdentityInput{Label: "No address"}, nil)
	second := commitIdentity(t, session, IdentityInput{Label: "Second", Addresses: []Address{{Country: "Denmark"}}}, nil)
	commitCard(t, session, fullCard(), nil)
	got, err := session.IdentityAddresses()
	if err != nil {
		t.Fatal(err)
	}
	want := []IdentityAddresses{
		{Identity: first, Label: "Alex", Addresses: selectIdentity(t, session, first).Addresses},
		{Identity: second, Label: "Second", Addresses: selectIdentity(t, session, second).Addresses},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("identity addresses = %#v", got)
	}
}
