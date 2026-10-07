package api

import "github.com/dortanes/ravenpass/packages/vault"

// TagLimits bound the tags of one item: each tag's length in characters and their count.
type TagLimits struct {
	Tag  int `json:"tag"`
	Tags int `json:"tags"`
}

// GetTagLimits reports the vault core's tag bounds.
func (s *Service) GetTagLimits() (TagLimits, error) {
	return TagLimits{Tag: vault.MaxTagLength, Tags: vault.MaxItemTags}, nil
}
