package session

import "errors"

// ErrNotFound is returned when no saved session exists.
var ErrNotFound = errors.New("session: not found")

// Storage persists authorization state between client restarts.
// The data format is opaque to the storage layer; encoding and
// encryption are handled above.
type Storage interface {
	Load() ([]byte, error)
	Save(data []byte) error
}
