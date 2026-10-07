//go:build !darwin || !cgo

package qrcodes

import "errors"

var errUnavailable = errors.New("QR codes are read on macOS only")

func (Mac) Picture([]byte) ([]string, error) { return nil, errUnavailable }

func (Mac) Clipboard() ([]string, error) { return nil, errUnavailable }
