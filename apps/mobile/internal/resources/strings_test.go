package resources

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestTheAppsStringNamesAreResourceNamesThatNameCatalogKeys(t *testing.T) {
	data, err := os.ReadFile("../../strings.json")
	if err != nil {
		t.Fatal(err)
	}
	var names map[string]string
	if err := json.Unmarshal(data, &names); err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("strings.json names no string")
	}
	for name, key := range names {
		if !namePattern.MatchString(name) || key == "" {
			t.Errorf("%q names %q", name, key)
		}
	}
}

func TestAStringSourceInEveryLanguageParses(t *testing.T) {
	catalog, err := Parse([]byte(`{"app_name": {"en": "Ravenpass", "ru": "Ravenpass"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(catalog.Languages(), []string{"en", "ru"}) {
		t.Fatalf("languages %v", catalog.Languages())
	}
}

func TestEachLanguageGoesToItsOwnDirectory(t *testing.T) {
	if Directory("en") != "values" || Directory("ru") != "values-ru" {
		t.Fatalf("directories %s, %s", Directory("en"), Directory("ru"))
	}
}

func TestResourcesReadBackAsTheSourceText(t *testing.T) {
	catalog, err := Parse([]byte(`{
		"b_quote": {"en": "Can't \"fill\" C:\\ <now> & %1$s", "ru": "Нельзя «%1$s»"},
		"a_reference": {"en": "@home?", "ru": "?дом"}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	rendered := catalog.Resources("en")
	var file struct {
		Strings []struct {
			Name string `xml:"name,attr"`
			Text string `xml:",chardata"`
		} `xml:"string"`
	}
	if err := xml.Unmarshal(rendered, &file); err != nil {
		t.Fatalf("%v in\n%s", err, rendered)
	}
	if len(file.Strings) != 2 || file.Strings[0].Name != "a_reference" || file.Strings[1].Name != "b_quote" {
		t.Fatalf("strings %+v", file.Strings)
	}
	if got, want := file.Strings[0].Text, `\@home?`; got != want {
		t.Fatalf("a leading @ reads %q, want %q", got, want)
	}
	if got, want := file.Strings[1].Text, `Can\'t \"fill\" C:\\ <now> & %1$s`; got != want {
		t.Fatalf("quotes read %q, want %q", got, want)
	}
	if !strings.Contains(string(catalog.Resources("ru")), `\?дом`) {
		t.Fatalf("a leading ? is not escaped in\n%s", catalog.Resources("ru"))
	}
}

func TestAStringSourceMustAgreeAcrossLanguages(t *testing.T) {
	for name, source := range map[string]string{
		"missing translation": `{"a": {"en": "One", "ru": "Один"}, "b": {"en": "Two"}}`,
		"no English":          `{"a": {"ru": "Один"}}`,
		"other formats":       `{"a": {"en": "%1$d left", "ru": "Осталось %1$s"}}`,
		"padded text":         `{"a": {"en": " One"}}`,
		"empty text":          `{"a": {"en": ""}}`,
		"bad name":            `{"A-name": {"en": "One"}}`,
		"bad language":        `{"a": {"en": "One", "RU": "Один"}}`,
		"nothing":             `{}`,
		"not JSON":            `strings`,
	} {
		if _, err := Parse([]byte(source)); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
}
