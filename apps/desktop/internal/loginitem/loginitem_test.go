package loginitem

import "testing"

func TestOnlyAnAppBundlesExecutableIsBundled(t *testing.T) {
	for executable, want := range map[string]bool{
		"/Applications/Ravenpass.app/Contents/MacOS/Ravenpass": true,
		"/tmp/bin/Ravenpass-dev":                               false,
		"/tmp/Ravenpass.app/Ravenpass":                         false,
		"/tmp/Contents/MacOS/Ravenpass":                        false,
	} {
		if got := bundled(executable); got != want {
			t.Errorf("bundled(%q) = %v, want %v", executable, got, want)
		}
	}
}
