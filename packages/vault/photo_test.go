package vault

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand/v2"
	"testing"
)

func patternPicture(width, height int) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			picture.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: uint8((x ^ y) * 3), A: 255})
		}
	}
	return picture
}

func noisePicture(size int) *image.NRGBA {
	random := rand.New(rand.NewPCG(1, 2))
	picture := image.NewNRGBA(image.Rect(0, 0, size, size))
	for i := range picture.Pix {
		picture.Pix[i] = uint8(random.Uint32())
	}
	return picture
}

func pngBytes(t *testing.T, picture image.Image) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func jpegBytes(t *testing.T, picture image.Image) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, picture, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

// withExif puts an APP1 Exif segment right after the start of image marker.
func withExif(jpegData []byte) []byte {
	payload := []byte("Exif\x00\x00MM\x00\x2a\x00\x00\x00\x08\x00\x00secret camera serial")
	segment := append([]byte{0xff, 0xe1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}, payload...)
	return append(append(append([]byte(nil), jpegData[:2]...), segment...), jpegData[2:]...)
}

// markersBeforeScan lists the JPEG segment markers up to the start of scan.
func markersBeforeScan(t *testing.T, data []byte) []byte {
	t.Helper()
	if len(data) < 2 || data[0] != 0xff || data[1] != 0xd8 {
		t.Fatal("not a JPEG")
	}
	var markers []byte
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xff {
			t.Fatalf("no marker at %d", i)
		}
		marker := data[i+1]
		markers = append(markers, marker)
		if marker == 0xda {
			return markers
		}
		i += 2 + (int(data[i+2])<<8 | int(data[i+3]))
	}
	t.Fatal("no start of scan")
	return nil
}

func assertStoredJPEG(t *testing.T, data []byte, size, budget int) {
	t.Helper()
	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != size || config.Height != size {
		t.Fatalf("stored picture is %dx%d, error = %v", config.Width, config.Height, err)
	}
	if len(data) > budget {
		t.Fatalf("stored picture is %d bytes, over %d", len(data), budget)
	}
	for _, marker := range markersBeforeScan(t, data) {
		if marker >= 0xe0 && marker <= 0xef || marker == 0xfe {
			t.Fatalf("stored picture carries metadata segment %#x", marker)
		}
		if marker == 0xc2 {
			t.Fatal("stored picture is progressive")
		}
	}
}

func TestPhotoIsStoredAsBareJPEGWithThumbnail(t *testing.T) {
	sources := map[string][]byte{
		"png":            pngBytes(t, patternPicture(PhotoSize, PhotoSize)),
		"jpeg with exif": withExif(jpegBytes(t, patternPicture(PhotoSize, PhotoSize))),
	}
	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			session, _ := populatedSession(t)
			defer session.Lock()
			id := commitIdentity(t, session, IdentityInput{Label: "Alex", Photo: source}, nil)
			identity := selectIdentity(t, session, id)
			if bytes.Equal(identity.Photo, source) {
				t.Fatal("the given picture was stored as it came")
			}
			if bytes.Contains(identity.Photo, []byte("secret camera serial")) {
				t.Fatal("the Exif segment survived")
			}
			assertStoredJPEG(t, identity.Photo, PhotoSize, maxPhotoBytes)
			assertStoredJPEG(t, listedEntry(t, session, id).Thumbnail, photoRule.thumbnailSide, maxThumbnailBytes)
		})
	}
}

func TestPhotoRefusalsLeaveTheVaultUnchanged(t *testing.T) {
	square := pngBytes(t, patternPicture(PhotoSize, PhotoSize))
	squareJPEG := jpegBytes(t, patternPicture(PhotoSize, PhotoSize))
	refused := map[string][]byte{
		"narrower PNG":   pngBytes(t, patternPicture(PhotoSize-1, PhotoSize)),
		"taller JPEG":    jpegBytes(t, patternPicture(PhotoSize, PhotoSize+1)),
		"smaller PNG":    pngBytes(t, patternPicture(96, 96)),
		"truncated PNG":  square[:len(square)/2],
		"truncated JPEG": squareJPEG[:len(squareJPEG)/2],
		"WebP":           []byte("RIFF\x10\x00\x00\x00WEBPVP8 "),
		"GIF":            []byte("GIF89a\x00\x02\x00\x02"),
		"text":           []byte("not a picture"),
		"too many bytes": append(append([]byte(nil), pngSignature...), make([]byte, photoRule.givenBytes)...),
	}
	session, _ := populatedSession(t)
	defer session.Lock()
	existing := commitIdentity(t, session, IdentityInput{Label: "Alex"}, nil)
	before, head, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for name, photo := range refused {
		t.Run(name, func(t *testing.T) {
			if _, _, err := session.PrepareCreateIdentity(IdentityInput{Label: "Alex", Photo: photo}, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("create: got %v, want ErrInvalidInput", err)
			}
			if _, err := session.PrepareEditIdentity(existing, IdentityInput{Label: "Alex", Photo: photo}, nil); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("edit: got %v, want ErrInvalidInput", err)
			}
		})
	}
	after, nextHead, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || nextHead != head {
		t.Fatal("a refused photo changed the vault")
	}
}

func TestEditKeepsTheStoredPhotoBytes(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	id := commitIdentity(t, session, IdentityInput{Label: "Alex", Photo: pngBytes(t, patternPicture(PhotoSize, PhotoSize))}, nil)
	stored := selectIdentity(t, session, id)
	thumbnail := listedEntry(t, session, id).Thumbnail

	kept := stored.IdentityInput
	kept.Label = "Alex renamed"
	pending, err := session.PrepareEditIdentity(id, kept, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if again := selectIdentity(t, session, id); !bytes.Equal(again.Photo, stored.Photo) || again.Label != "Alex renamed" {
		t.Fatal("an edit that passed the stored photo back changed it")
	}
	if !bytes.Equal(listedEntry(t, session, id).Thumbnail, thumbnail) {
		t.Fatal("an edit that kept the photo changed the thumbnail")
	}

	replaced := kept
	replaced.Photo = pngBytes(t, noisePicture(PhotoSize))
	pending, err = session.PrepareEditIdentity(id, replaced, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if again := selectIdentity(t, session, id); bytes.Equal(again.Photo, stored.Photo) {
		t.Fatal("a new photo did not replace the stored one")
	}
	if bytes.Equal(listedEntry(t, session, id).Thumbnail, thumbnail) {
		t.Fatal("a new photo kept the old thumbnail")
	}

	removed := kept
	removed.Photo = nil
	pending, err = session.PrepareEditIdentity(id, removed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	if again := selectIdentity(t, session, id); again.Photo != nil {
		t.Fatal("a removed photo is still stored")
	}
	if listedEntry(t, session, id).Thumbnail != nil {
		t.Fatal("a removed photo left its thumbnail")
	}
	plaintext, err := session.openRecord(session.find(id))
	if err != nil {
		t.Fatal(err)
	}
	if schema := plaintext[1]; schema != recordSchemaIdentity {
		t.Fatalf("an identity without a photo was written as schema %d", schema)
	}
}

func TestPhotoOfAnotherIdentityIsEncodedAgain(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	first := commitIdentity(t, session, IdentityInput{Label: "First", Photo: pngBytes(t, patternPicture(PhotoSize, PhotoSize))}, nil)
	second := commitIdentity(t, session, IdentityInput{Label: "Second"}, nil)
	photo := selectIdentity(t, session, first).Photo
	pending, err := session.PrepareEditIdentity(second, IdentityInput{Label: "Second", Photo: photo}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Commit(pending); err != nil {
		t.Fatal(err)
	}
	assertStoredJPEG(t, selectIdentity(t, session, second).Photo, PhotoSize, maxPhotoBytes)
	assertStoredJPEG(t, listedEntry(t, session, second).Thumbnail, photoRule.thumbnailSide, maxThumbnailBytes)
}

func TestEncodingStepsQualityDownToFitItsBudget(t *testing.T) {
	picture := noisePicture(64)
	sizes := map[int]int{}
	for quality := 85; quality >= 45; quality -= qualityStep {
		var encoded bytes.Buffer
		if err := jpeg.Encode(&encoded, picture, &jpeg.Options{Quality: quality}); err != nil {
			t.Fatal(err)
		}
		sizes[quality] = encoded.Len()
	}
	fitted, err := encodeWithin(picture, 85, 45, sizes[65])
	if err != nil {
		t.Fatal(err)
	}
	if len(fitted) != sizes[65] {
		t.Fatalf("fitted size %d, want the quality 65 size %d", len(fitted), sizes[65])
	}
	if _, err := encodeWithin(picture, 85, 45, sizes[45]-1); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("a picture over budget at the last quality: %v", err)
	}
}

func TestPhotoNeverReachesTheContainerInPlaintext(t *testing.T) {
	session, _ := populatedSession(t)
	defer session.Lock()
	id := commitIdentity(t, session, IdentityInput{Label: "Alex", Photo: pngBytes(t, noisePicture(PhotoSize))}, nil)
	photo := selectIdentity(t, session, id).Photo
	thumbnail := listedEntry(t, session, id).Thumbnail
	container, _, err := session.CurrentContainer()
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"photo": photo, "thumbnail": thumbnail} {
		window := data[len(data)/2 : len(data)/2+32]
		if bytes.Contains(container, window) {
			t.Fatalf("the %s appears in the vault file", name)
		}
	}
}

func TestThumbnailMustMatchThePhotoInTheRecord(t *testing.T) {
	created, _ := vaultWith(t)
	session := created.Session
	defer session.Lock()
	commitIdentity(t, session, IdentityInput{Label: "With", Photo: pngBytes(t, patternPicture(PhotoSize, PhotoSize))}, nil)
	commitIdentity(t, session, IdentityInput{Label: "Without"}, nil)
	raw, head := currentRaw(t, session)
	tests := []struct {
		name      string
		index     int
		thumbnail []byte
	}{
		{"photo without a thumbnail", 0, nil},
		{"thumbnail without a photo", 1, session.entries[0].thumbnail},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := append([]entryMeta(nil), session.entries...)
			entries[test.index].thumbnail = test.thumbnail
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

func TestIdentityPhotoRecordSchema(t *testing.T) {
	withPhoto, err := encodeIdentityRecord(IdentityInput{Photo: []byte{0xff, 0xd8, 0xff, 0xd9}})
	if err != nil {
		t.Fatal(err)
	}
	if decoded, err := decodeIdentityRecord(withPhoto); err != nil || !bytes.Equal(decoded.Photo, []byte{0xff, 0xd8, 0xff, 0xd9}) {
		t.Fatalf("schema 9 record = %+v, error = %v", decoded, err)
	}
	withoutPhoto, err := encodeIdentityRecord(IdentityInput{})
	if err != nil {
		t.Fatal(err)
	}
	if decoded, err := decodeIdentityRecord(withoutPhoto); err != nil || decoded.Photo != nil {
		t.Fatalf("schema 9 record without a photo = %+v, error = %v", decoded, err)
	}
	empty := encodeBytes(nil)
	fields := [][]byte{encodeUint(recordSchemaIdentity), empty, empty, encodeArray(), encodeArray(), encodeArray(), encodeArray(), empty}
	tests := []struct {
		name   string
		decode func([]byte) error
		record []byte
		want   error
	}{
		{"schema 9 without its photo field", decodeAsIdentity, encodeArray(fields...), ErrMalformed},
		{"schema 9 photo over its budget", decodeAsIdentity, encodeArray(append(fields, encodeBytes(make([]byte, maxPhotoBytes+1)))...), ErrMalformed},
		{"schema 9 read as a credential", decodeAsCredential, withPhoto, ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode(test.record); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}
