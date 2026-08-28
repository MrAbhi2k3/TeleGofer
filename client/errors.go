package client

import "errors"

// Sentinel errors for client lifecycle.
var (
	ErrClosed        = errors.New("telegofer: client is closed")
	ErrNotAuthorized = errors.New("telegofer: not authorized")
)
