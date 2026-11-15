package api

import (
	"reflect"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/messages"
)

func TestImportLabelsAreCompleteInEveryLanguage(t *testing.T) {
	for _, language := range []string{"en", "ru", "zh"} {
		labels := reflect.ValueOf(importLabelsIn(messages.For(language)))
		for i := range labels.NumField() {
			if labels.Field(i).String() == "" {
				t.Errorf("%s has no %s label", language, labels.Type().Field(i).Name)
			}
		}
	}
	if got := importLabelsIn(messages.For("ru")).OneTimeCode; got != "Настройка одноразового кода" {
		t.Fatalf("Russian one-time code label = %q", got)
	}
	if got := importLabelsIn(messages.For("zh")).OneTimeCode; got != "一次性代码设置" {
		t.Fatalf("Chinese one-time code label = %q", got)
	}
}
