package vault

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// MaxTagLength bounds one tag, in characters.
	MaxTagLength = 32
	// MaxItemTags bounds the tags one item carries.
	MaxItemTags = 8
)

// AcceptTag returns a tag as the vault stores it: trimmed, its inner spaces collapsed, and otherwise as typed.
func AcceptTag(tag string) (string, error) {
	if !utf8.ValidString(tag) {
		return "", ErrInvalidInput
	}
	accepted := strings.Join(strings.Fields(tag), " ")
	if accepted == "" || utf8.RuneCountInString(accepted) > MaxTagLength {
		return "", ErrInvalidInput
	}
	if strings.ContainsFunc(accepted, unicode.IsControl) {
		return "", ErrInvalidInput
	}
	return accepted, nil
}

// TagKey is the form in which two tags are compared; one item holds each key once.
func TagKey(tag string) string {
	return strings.ToLower(tag)
}

// AcceptTags returns an item's tags as the vault stores them, in the order given, dropping blanks and repeats by
// TagKey; nil for none.
func AcceptTags(tags []string) ([]string, error) {
	var accepted []string
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		tag, err := AcceptTag(tag)
		if err != nil {
			return nil, err
		}
		if slices.ContainsFunc(accepted, func(held string) bool { return TagKey(held) == TagKey(tag) }) {
			continue
		}
		if len(accepted) == MaxItemTags {
			return nil, ErrResourceLimit
		}
		accepted = append(accepted, tag)
	}
	return accepted, nil
}

// parseTags reads an index entry's tags, which hold only accepted, distinct tags.
func parseTags(tags []string, kind Kind) ([]string, error) {
	if len(tags) > MaxItemTags {
		return nil, ErrResourceLimit
	}
	if kind == KindAttachment && len(tags) != 0 {
		return nil, ErrMalformed
	}
	accepted, err := AcceptTags(tags)
	if err != nil || !slices.Equal(accepted, tags) {
		return nil, ErrMalformed
	}
	return accepted, nil
}
