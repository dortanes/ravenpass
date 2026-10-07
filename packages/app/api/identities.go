package api

import (
	"encoding/base64"
	"time"

	"github.com/dortanes/ravenpass/packages/vault"
)

// IdentityDocument is one document of an identity.
type IdentityDocument struct {
	// Type is "passport", "drivers-license", "id-card", "tax-number" or "other".
	Type      string `json:"type"`
	Label     string `json:"label"`
	Number    string `json:"number"`
	Issuer    string `json:"issuer"`
	IssuedOn  string `json:"issuedOn"`
	ExpiresOn string `json:"expiresOn"`
	// Scans are attachment ids, passed back as read on edit, or "new:" tokens of staged scans.
	Scans []string `json:"scans"`
}

// IdentityInput is the editable content of an identity.
type IdentityInput struct {
	Label     string             `json:"label"`
	FullName  string             `json:"fullName"`
	Birthday  string             `json:"birthday"`
	Emails    []string           `json:"emails"`
	Phones    []string           `json:"phones"`
	Addresses []Address          `json:"addresses"`
	Documents []IdentityDocument `json:"documents"`
	Notes     string             `json:"notes"`
	// Photo is base64, empty for none: the stored photo as read or the PNG CropIdentityPhoto returned.
	Photo string `json:"photo"`
	// Tags tell the item apart from others like it.
	Tags []string `json:"tags"`
}

// IdentitySummary is an identity as the list shows it; empty strings mean none.
type IdentitySummary struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Email is the first email.
	Email      string   `json:"email"`
	Pinned     bool     `json:"pinned"`
	LastUsedAt int64    `json:"lastUsedAt"`
	Groups     []string `json:"groups"`
	Tags       []string `json:"tags"`
	// ExpiresOn is the earliest document expiry as YYYY-MM-DD.
	ExpiresOn string `json:"expiresOn"`
	// Thumbnail is the photo as a base64 JPEG.
	Thumbnail string `json:"thumbnail"`
}

// Identity is an identity as read.
type Identity struct {
	ID     string   `json:"id"`
	Groups []string `json:"groups"`
	// Attachments describes every scan its documents name.
	Attachments []ScanSummary `json:"attachments"`
	IdentityInput
}

// ListIdentities reports every identity from the index.
func (s *Service) ListIdentities() ([]IdentitySummary, error) {
	entries, usage, err := s.listKind(vault.KindIdentity)
	if err != nil {
		return nil, err
	}
	result := make([]IdentitySummary, len(entries))
	for i, entry := range entries {
		result[i] = IdentitySummary{
			ID:         entry.ID.String(),
			Label:      entry.Label,
			Email:      entry.Detail,
			Pinned:     entry.Pinned,
			LastUsedAt: usage[entry.ID],
			Groups:     idStrings(entry.Groups),
			Tags:       append([]string{}, entry.Tags...),
			ExpiresOn:  entry.ExpiresOn,
			Thumbnail:  base64.StdEncoding.EncodeToString(entry.Thumbnail),
		}
	}
	return result, nil
}

// ReadIdentity opens the identity id with its attachment summaries and records the use.
func (s *Service) ReadIdentity(id string) (Identity, error) {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return Identity{}, fail(failureItemUnreadable)
	}
	item, groups, err := openItem(s, parsed, s.vault.ReadSelectedIdentity)
	if err != nil {
		return Identity{}, err
	}
	scans, err := s.vault.ScansOf(parsed)
	if err != nil {
		return Identity{}, present(err)
	}
	attachments := make([]ScanSummary, len(scans))
	for i, scan := range scans {
		attachments[i] = scanSummary(scan)
	}
	return Identity{ID: item.ID.String(), Groups: groups, Attachments: attachments, IdentityInput: fromVaultIdentity(item.IdentityInput)}, nil
}

// CreateIdentity creates an identity in groups with its staged scans and returns its id.
func (s *Service) CreateIdentity(input IdentityInput, groups []string) (string, error) {
	membership, err := s.knownGroups(groups)
	if err != nil {
		return "", err
	}
	identity, err := s.identityToVault(input)
	if err != nil {
		return "", err
	}
	id, err := s.vault.CreateIdentity(identity, membership)
	if err != nil {
		return "", present(err)
	}
	s.scans.clear()
	return id.String(), nil
}

// UpdateIdentity replaces an identity's content, membership and scans in one save.
func (s *Service) UpdateIdentity(id string, input IdentityInput, groups []string) error {
	parsed, err := vault.ParseID(id)
	if err != nil {
		return fail(failureItemUnreadable)
	}
	membership, err := s.knownGroups(groups)
	if err != nil {
		return err
	}
	identity, err := s.identityToVault(input)
	if err != nil {
		return err
	}
	if err := s.vault.EditIdentity(parsed, identity, membership); err != nil {
		return present(err)
	}
	s.scans.clear()
	return nil
}

// IdentityField names one copyable identity value.
type IdentityField struct {
	// Kind is "fullName", "birthday", "notes", "email", "phone", "address" or "document".
	Kind string `json:"kind"`
	// Index is the zero-based position in the Kind's list.
	Index int `json:"index,omitempty"`
	// Part names one part of an address alone.
	Part string `json:"part,omitempty"`
}

// CopyIdentityField copies one value of an identity. A document copies its number.
func (s *Service) CopyIdentityField(id string, field IdentityField) error {
	return s.copyIdentityField(id, field, clearAfter)
}

func (s *Service) copyIdentityField(id string, field IdentityField, schedule func(time.Duration, func())) error {
	read, supported := field.reader()
	if !supported {
		return fail(failureFieldNotCopyable)
	}
	return s.copyItemValue(id, schedule, func(parsed vault.ID) (string, error) {
		item, err := readSelected(s, parsed, s.vault.ReadSelectedIdentity)
		if err != nil {
			return "", err
		}
		value, found := read(item.IdentityInput)
		if !found {
			return "", fail(failureFieldNotCopyable)
		}
		return value, nil
	})
}

var identityListFields = map[string]func(vault.IdentityInput, int) (string, bool){
	"email": func(identity vault.IdentityInput, index int) (string, bool) { return at(identity.Emails, index) },
	"phone": func(identity vault.IdentityInput, index int) (string, bool) { return at(identity.Phones, index) },
	"address": func(identity vault.IdentityInput, index int) (string, bool) {
		address, found := at(identity.Addresses, index)
		return formatAddress(address), found
	},
	"document": func(identity vault.IdentityInput, index int) (string, bool) {
		document, found := at(identity.Documents, index)
		return document.Number, found
	},
}

// identitySingleFields read a value an identity holds once; "birthday" is the stored YYYY-MM-DD date.
var identitySingleFields = map[string]func(vault.IdentityInput) string{
	"fullName": func(identity vault.IdentityInput) string { return identity.FullName },
	"birthday": func(identity vault.IdentityInput) string { return identity.Birthday },
	"notes":    func(identity vault.IdentityInput) string { return identity.Notes },
}

// reader returns the reader of the one value f names.
func (f IdentityField) reader() (func(vault.IdentityInput) (string, bool), bool) {
	if read, single := identitySingleFields[f.Kind]; single {
		if f.Index != 0 || f.Part != "" {
			return nil, false
		}
		return func(identity vault.IdentityInput) (string, bool) { return read(identity), true }, true
	}
	if f.Index < 0 {
		return nil, false
	}
	if f.Part != "" {
		part, known := addressPart(f.Part)
		if f.Kind != "address" || !known {
			return nil, false
		}
		return func(identity vault.IdentityInput) (string, bool) {
			address, found := at(identity.Addresses, f.Index)
			return part(address), found
		}, true
	}
	read, known := identityListFields[f.Kind]
	if !known {
		return nil, false
	}
	return func(identity vault.IdentityInput) (string, bool) { return read(identity, f.Index) }, true
}

func at[T any](items []T, index int) (T, bool) {
	if index >= len(items) {
		var none T
		return none, false
	}
	return items[index], true
}

func (s *Service) identityToVault(input IdentityInput) (vault.IdentityInput, error) {
	photo, err := base64.StdEncoding.DecodeString(input.Photo)
	if err != nil {
		return vault.IdentityInput{}, fail(failureInvalidItem)
	}
	identity := vault.IdentityInput{
		Label: input.Label, FullName: input.FullName, Birthday: input.Birthday,
		Emails: input.Emails, Phones: input.Phones, Notes: input.Notes, Photo: photo, Tags: input.Tags,
	}
	for _, address := range input.Addresses {
		converted, err := address.toVault()
		if err != nil {
			return vault.IdentityInput{}, err
		}
		identity.Addresses = append(identity.Addresses, converted)
	}
	used := make(map[string]struct{})
	for _, document := range input.Documents {
		documentType, known := vault.DocumentTypeNamed(document.Type)
		if !known {
			return vault.IdentityInput{}, fail(failureInvalidItem)
		}
		stored, attached, err := s.resolveScans(document.Scans, used)
		if err != nil {
			return vault.IdentityInput{}, err
		}
		identity.Documents = append(identity.Documents, vault.Document{
			Type: documentType, Label: document.Label, Number: document.Number,
			Issuer: document.Issuer, IssuedOn: document.IssuedOn, ExpiresOn: document.ExpiresOn,
			Scans: stored, Attach: attached,
		})
	}
	return identity, nil
}

// fromVaultIdentity reports every list as an array, never null.
func fromVaultIdentity(identity vault.IdentityInput) IdentityInput {
	input := IdentityInput{
		Label: identity.Label, FullName: identity.FullName, Birthday: identity.Birthday,
		Emails:    append([]string{}, identity.Emails...),
		Phones:    append([]string{}, identity.Phones...),
		Addresses: make([]Address, len(identity.Addresses)),
		Documents: make([]IdentityDocument, len(identity.Documents)),
		Notes:     identity.Notes,
		Photo:     base64.StdEncoding.EncodeToString(identity.Photo),
		Tags:      append([]string{}, identity.Tags...),
	}
	for i, address := range identity.Addresses {
		input.Addresses[i] = fromVaultAddress(address)
	}
	for i, document := range identity.Documents {
		input.Documents[i] = IdentityDocument{
			Type: document.Type.Name(), Label: document.Label, Number: document.Number,
			Issuer: document.Issuer, IssuedOn: document.IssuedOn, ExpiresOn: document.ExpiresOn,
			Scans: idStrings(document.Scans),
		}
	}
	return input
}
