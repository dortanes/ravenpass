//go:build !darwin || !arm64 || !cgo

package sandbox

func contained() bool { return false }

type system struct{}

func (system) create(string) ([]byte, error) { return nil, ErrUnavailable }

func (system) resolve([]byte) (string, bool, error) { return "", false, ErrUnavailable }

func (system) replace(string, []byte) error { return ErrUnavailable }
