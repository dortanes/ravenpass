// Package qrcodes finds QR codes in pictures with the macOS Vision framework.
package qrcodes

import (
	"strings"

	"github.com/dortanes/ravenpass/packages/app/api"
)

var _ api.QRCodeReader = Mac{}

// Mac reads QR codes in an image file's content and in the clipboard's picture.
type Mac struct{}

// texts splits the NUL-terminated UTF-8 texts the Vision adapter returns.
func texts(joined []byte) []string {
	found := strings.Split(string(joined), "\x00")
	return found[:len(found)-1]
}
