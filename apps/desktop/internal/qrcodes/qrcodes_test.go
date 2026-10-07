package qrcodes

import (
	"slices"
	"testing"
)

func TestTextsSplitTheAdaptersNULTerminatedTexts(t *testing.T) {
	for joined, want := range map[string][]string{
		"":                     {},
		"otpauth://totp/a\x00": {"otpauth://totp/a"},
		"first\x00second\x00":  {"first", "second"},
	} {
		if got := texts([]byte(joined)); !slices.Equal(got, want) {
			t.Errorf("texts(%q) = %q, want %q", joined, got, want)
		}
	}
}
