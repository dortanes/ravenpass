package bridge

import "sync/atomic"

var codeSetupReceiver atomic.Pointer[func(link string)]

// ReceiveCodeSetups hands each otpauth link Android opens in Ravenpass to receive, on a worker thread.
func ReceiveCodeSetups(receive func(link string)) {
	codeSetupReceiver.Store(&receive)
}

// codeSetupOpened passes a UTF-8 link to the receiver; a link that arrives before one is registered is dropped.
func codeSetupOpened(link []byte) {
	if receive := codeSetupReceiver.Load(); receive != nil {
		(*receive)(string(link))
	}
}
