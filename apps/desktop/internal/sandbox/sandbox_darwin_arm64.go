//go:build cgo

package sandbox

/*
#cgo LDFLAGS: -L/usr/lib/swift -framework Foundation
#include <stdint.h>
#include <stdlib.h>

int32_t ravenpass_sandboxed(void);
int32_t ravenpass_bookmark_create(const uint8_t *path, intptr_t pathLength, uint8_t **bookmark, intptr_t *bookmarkLength);
int32_t ravenpass_bookmark_resolve(const uint8_t *bookmark, intptr_t bookmarkLength, uint8_t **path, intptr_t *pathLength);
int32_t ravenpass_file_replace(const uint8_t *path, intptr_t pathLength, const uint8_t *data, intptr_t dataLength);
*/
import "C"

import "unsafe"

// Result codes shared with sandbox.swift.
const (
	resultSucceeded = 0
	resultStale     = 2
)

func contained() bool { return C.ravenpass_sandboxed() == 1 }

type system struct{}

func (system) create(folder string) ([]byte, error) {
	if folder == "" {
		return nil, ErrBookmark
	}
	var out *C.uint8_t
	var length C.intptr_t
	if C.ravenpass_bookmark_create(stringPointer(folder), C.intptr_t(len(folder)), &out, &length) != resultSucceeded {
		return nil, ErrBookmark
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoBytes(unsafe.Pointer(out), C.int(length)), nil
}

func (system) resolve(bookmark []byte) (string, bool, error) {
	if len(bookmark) == 0 {
		return "", false, ErrResolve
	}
	var out *C.uint8_t
	var length C.intptr_t
	result := C.ravenpass_bookmark_resolve((*C.uint8_t)(unsafe.Pointer(&bookmark[0])), C.intptr_t(len(bookmark)), &out, &length)
	if result != resultSucceeded && result != resultStale {
		return "", false, ErrResolve
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoStringN((*C.char)(unsafe.Pointer(out)), C.int(length)), result == resultStale, nil
}

func (system) replace(path string, data []byte) error {
	if path == "" || len(data) == 0 {
		return ErrWrite
	}
	if C.ravenpass_file_replace(stringPointer(path), C.intptr_t(len(path)),
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.intptr_t(len(data))) != resultSucceeded {
		return ErrWrite
	}
	return nil
}

func stringPointer(value string) *C.uint8_t {
	return (*C.uint8_t)(unsafe.Pointer(unsafe.StringData(value)))
}
