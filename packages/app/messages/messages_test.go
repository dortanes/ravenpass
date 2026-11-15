package messages

import (
	"encoding/json"
	"os"
	"testing"
)

func TestForMatchesTheLanguageWithEnglishAsFallback(t *testing.T) {
	for tag, want := range map[string]string{
		"en": "en", "en-GB": "en", "ru": "ru", "ru-RU": "ru", "RU": "ru", "zh": "zh", "zh-CN": "zh", "ZH": "zh", "de-DE": "en", "": "en", "not a tag": "en",
	} {
		if got := For(tag).Language(); got != want {
			t.Errorf("For(%q) = %s, want %s", tag, got, want)
		}
	}
}

func TestTextReadsTheLanguageAndFallsBackToEnglish(t *testing.T) {
	if got := For("ru").Text("system.menu.quit"); got != "Выйти из Ravenpass" {
		t.Fatalf("Russian text = %q", got)
	}
	if got := For("zh").Text("system.menu.quit"); got != "退出 Ravenpass" {
		t.Fatalf("Chinese text = %q", got)
	}
	if got := (Catalog{language: "xx"}).Text("system.menu.quit"); got != "Quit Ravenpass" {
		t.Fatalf("fallback text = %q", got)
	}
	if got := For("en").Text("no.such.key"); got != "" {
		t.Fatalf("an unknown key read %q", got)
	}
}

func TestTheCatalogCarriesEveryNamedMessageInEveryLanguage(t *testing.T) {
	data, err := os.ReadFile("names.json")
	if err != nil {
		t.Fatal(err)
	}
	var names map[string]string
	if err := json.Unmarshal(data, &names); err != nil {
		t.Fatal(err)
	}
	catalog := loaded()
	if len(catalog.texts) != len(names) {
		t.Fatalf("the catalog holds %d messages, names.json names %d: regenerate it", len(catalog.texts), len(names))
	}
	for name := range names {
		for _, language := range catalog.languages {
			if catalog.texts[name][language] == "" {
				t.Errorf("%s has no %s text", name, language)
			}
		}
	}
}
