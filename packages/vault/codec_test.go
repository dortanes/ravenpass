package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

// encodingFixture is the golden encoding of every structure; never regenerate it from this package.
type encodingFixture struct {
	Records      map[string]string `json:"records"`
	IndexHex     string            `json:"indexHex"`
	IndexRecords int               `json:"indexRecords"`
	ContainerHex string            `json:"containerHex"`
	EnvelopeHex  string            `json:"envelopeHex"`
}

func loadEncodingFixture(t *testing.T) encodingFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/encoding.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture encodingFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestRecordsReadAndWriteBackTheirFixtureBytes(t *testing.T) {
	fixture := loadEncodingFixture(t)
	reencode := map[string]func([]byte) ([]byte, error){
		"credential": func(plaintext []byte) ([]byte, error) {
			input, err := decodeCredentialRecord(plaintext)
			if err != nil {
				return nil, err
			}
			return encodeCredentialRecord(input)
		},
		"identity": func(plaintext []byte) ([]byte, error) {
			input, err := decodeIdentityRecord(plaintext)
			if err != nil {
				return nil, err
			}
			return encodeIdentityRecord(input)
		},
		"card": func(plaintext []byte) ([]byte, error) {
			input, err := decodeCardRecord(plaintext)
			if err != nil {
				return nil, err
			}
			return encodeCardRecord(input)
		},
		"note": func(plaintext []byte) ([]byte, error) {
			input, err := decodeNoteRecord(plaintext)
			if err != nil {
				return nil, err
			}
			return encodeNoteRecord(input)
		},
		"seed": func(plaintext []byte) ([]byte, error) {
			input, checkedOn, err := decodeSeedRecord(plaintext)
			if err != nil {
				return nil, err
			}
			return encodeSeedRecord(input, checkedOn)
		},
		"attachment": func(plaintext []byte) ([]byte, error) {
			content, err := decodeAttachmentRecord(plaintext, MediaPDF)
			if err != nil {
				return nil, err
			}
			return encodeAttachmentRecord(content)
		},
	}
	kinds := map[string]string{
		"credential": "credential", "credentialBare": "credential", "identity": "identity",
		"cardBilling": "card", "cardLink": "card", "cardPlain": "card", "note": "note",
		"seedPhrase": "seed", "seedCodes": "seed", "seedKey": "seed", "attachment": "attachment",
	}
	if len(fixture.Records) != len(kinds) {
		t.Fatalf("the fixture holds %d records, want %d", len(fixture.Records), len(kinds))
	}
	for name, kind := range kinds {
		t.Run(name, func(t *testing.T) {
			plaintext := decodeHex(t, fixture.Records[name])
			written, err := reencode[kind](plaintext)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(written, plaintext) {
				t.Fatalf("written back as % x, want % x", written, plaintext)
			}
		})
	}
}

func TestIndexOfEveryFieldWritesBackItsFixtureBytes(t *testing.T) {
	fixture := loadEncodingFixture(t)
	plaintext := decodeHex(t, fixture.IndexHex)
	records := stubRecords(fixture.IndexRecords)
	revision, ancestors, index, err := parseIndex(plaintext, records)
	if err != nil {
		t.Fatal(err)
	}
	entries, groups := index.entries, index.groups
	if len(entries[0].passkeys) == 0 || len(entries[0].apps) == 0 || len(groups) == 0 || len(ancestors) < 2 {
		t.Fatal("the fixture index does not use every field")
	}
	written, err := encodeIndex(revision, ancestors, entries, groups, index.retention)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, plaintext) {
		t.Fatalf("written back as % x, want % x", written, plaintext)
	}
}

func TestContainerAndEnvelopeWriteBackTheirFixtureBytes(t *testing.T) {
	fixture := loadEncodingFixture(t)
	container := decodeHex(t, fixture.ContainerHex)
	raw, err := parseContainer(container)
	if err != nil {
		t.Fatal(err)
	}
	written, err := encodeContainer(raw.vaultID, raw.recovery, raw.index, raw.records)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, container) || !bytes.Equal(raw.data, container) {
		t.Fatalf("container written back as % x", written)
	}
	envelope := decodeHex(t, fixture.EnvelopeHex)
	key, box, err := decodeDeviceEnvelope(envelope, raw.vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if written := encodeDeviceEnvelope(raw.vaultID, key, box); !bytes.Equal(written, envelope) {
		t.Fatalf("envelope written back as % x", written)
	}
}

func TestDecodingRefusesAnyOtherEncodingOfAValue(t *testing.T) {
	type pair struct {
		_      struct{} `cbor:",toarray"`
		Number uint64
		Text   string
	}
	valid := encodeArray(encodeUint(1), encodeBytes([]byte("a")))
	var decoded pair
	if err := unmarshal(valid, &decoded); err != nil || decoded.Number != 1 || decoded.Text != "a" {
		t.Fatalf("the canonical encoding: %+v, %v", decoded, err)
	}
	deep := encodeArray()
	for range maxNesting {
		deep = encodeArray(deep)
	}
	tests := []struct {
		name  string
		input []byte
		want  error
	}{
		{"a number in a longer head", []byte{0x82, 0x18, 0x01, 0x41, 'a'}, ErrMalformed},
		{"an array length in a longer head", []byte{0x98, 0x02, 0x01, 0x41, 'a'}, ErrMalformed},
		{"a text string for bytes", []byte{0x82, 0x01, 0x61, 'a'}, ErrMalformed},
		{"an indefinite array", []byte{0x9f, 0x01, 0x41, 'a', 0xff}, ErrMalformed},
		{"an indefinite byte string", []byte{0x82, 0x01, 0x5f, 0x41, 'a', 0xff}, ErrMalformed},
		{"a tag", []byte{0xc1, 0x82, 0x01, 0x41, 'a'}, ErrMalformed},
		{"a map", []byte{0xa2, 0x01, 0x01, 0x02, 0x41, 'a'}, ErrMalformed},
		{"a trailing byte", append(bytes.Clone(valid), 0), ErrMalformed},
		{"a missing field", encodeArray(encodeUint(1)), ErrMalformed},
		{"nesting past the bound", deep, ErrResourceLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var decoded pair
			if err := unmarshal(test.input, &decoded); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}
