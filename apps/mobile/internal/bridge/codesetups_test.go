package bridge

import "testing"

func TestAnOpenedLinkReachesTheRegisteredReceiver(t *testing.T) {
	codeSetupOpened([]byte("otpauth://totp/dropped"))
	var got []string
	ReceiveCodeSetups(func(link string) { got = append(got, link) })
	t.Cleanup(func() { codeSetupReceiver.Store(nil) })
	codeSetupOpened([]byte("otpauth://totp/Example:alex?secret=GEZDGNBV"))
	if len(got) != 1 || got[0] != "otpauth://totp/Example:alex?secret=GEZDGNBV" {
		t.Fatalf("received %q", got)
	}
}
