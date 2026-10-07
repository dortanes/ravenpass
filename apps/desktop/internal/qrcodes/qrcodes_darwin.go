//go:build darwin && cgo

package qrcodes

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework UniformTypeIdentifiers -framework Vision
#include "qrcodes_darwin.h"
*/
import "C"

import (
	"errors"
	"unsafe"

	"github.com/dortanes/ravenpass/packages/app/api"
	"github.com/dortanes/ravenpass/packages/app/photos"
)

var errFailed = errors.New("the QR code texts could not be copied")

// Picture fails with photos.ErrUnsupported for content that is not a picture Vision reads.
func (Mac) Picture(content []byte) ([]string, error) {
	if len(content) == 0 {
		return nil, photos.ErrUnsupported
	}
	var found *C.char
	var length C.size_t
	status := C.ravenpass_qr_picture(unsafe.Pointer(unsafe.SliceData(content)), C.size_t(len(content)), &found, &length)
	return answer(status, found, length)
}

func (Mac) Clipboard() ([]string, error) {
	var found *C.char
	var length C.size_t
	return answer(C.ravenpass_qr_clipboard(&found, &length), found, length)
}

// answer copies the adapter's texts and wipes and frees its copy.
func answer(status C.ravenpass_qr_status, found *C.char, length C.size_t) ([]string, error) {
	if found != nil {
		defer C.ravenpass_qr_release(found, length)
	}
	switch status {
	case C.ravenpass_qr_ok:
		joined := C.GoBytes(unsafe.Pointer(found), C.int(length))
		defer clear(joined)
		return texts(joined), nil
	case C.ravenpass_qr_no_picture:
		return nil, api.ErrNoPicture
	case C.ravenpass_qr_unreadable:
		return nil, photos.ErrUnsupported
	default:
		return nil, errFailed
	}
}
