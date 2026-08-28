package rpc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RPCError represents a Telegram RPC error response.
type RPCError struct {
	Code    int
	Type    string
	Message string
}

func (e *RPCError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("rpc error %d: %s (%s)", e.Code, e.Type, e.Message)
	}
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Type)
}

// FloodWaitError means the server wants the client to back off.
type FloodWaitError struct {
	Wait time.Duration
}

func (e *FloodWaitError) Error() string {
	return fmt.Sprintf("flood wait: retry after %s", e.Wait)
}

// MigrateError means the request must be resent to a different DC.
type MigrateError struct {
	DC   int
	Kind string // "PHONE", "FILE", "USER", etc.
}

func (e *MigrateError) Error() string {
	return fmt.Sprintf("%s_MIGRATE: switch to DC %d", e.Kind, e.DC)
}

// FileRefExpiredError indicates a file reference needs refreshing.
type FileRefExpiredError struct{}

func (e *FileRefExpiredError) Error() string { return "file reference expired" }

// ParseError constructs a typed error from a Telegram RPC error code
// and message string. Flood waits, DC migrations, and file reference
// errors are returned as their specific types.
func ParseError(code int, msg string) error {
	switch code {
	case 303:
		return parseMigrate(msg)
	case 420:
		return parseFloodWait(msg)
	}
	if msg == "FILE_REFERENCE_EXPIRED" {
		return &FileRefExpiredError{}
	}
	return &RPCError{Code: code, Type: msg}
}

func parseMigrate(msg string) error {
	// PHONE_MIGRATE_2, FILE_MIGRATE_4, USER_MIGRATE_1
	idx := strings.Index(msg, "_MIGRATE_")
	if idx < 0 {
		return &RPCError{Code: 303, Type: msg}
	}
	dc, err := strconv.Atoi(msg[idx+len("_MIGRATE_"):])
	if err != nil {
		return &RPCError{Code: 303, Type: msg}
	}
	return &MigrateError{DC: dc, Kind: msg[:idx]}
}

func parseFloodWait(msg string) error {
	// FLOOD_WAIT_300, FLOOD_PREMIUM_WAIT_120
	idx := strings.LastIndexByte(msg, '_')
	if idx < 0 {
		return &RPCError{Code: 420, Type: msg}
	}
	secs, err := strconv.Atoi(msg[idx+1:])
	if err != nil {
		return &RPCError{Code: 420, Type: msg}
	}
	return &FloodWaitError{Wait: time.Duration(secs) * time.Second}
}

// IsFloodWait reports whether err is a FloodWaitError. Returns the
// required wait duration if true.
func IsFloodWait(err error) (time.Duration, bool) {
	var e *FloodWaitError
	if errors.As(err, &e) {
		return e.Wait, true
	}
	return 0, false
}

// IsMigrate reports whether err is a MigrateError. Returns the
// target DC if true.
func IsMigrate(err error) (int, bool) {
	var e *MigrateError
	if errors.As(err, &e) {
		return e.DC, true
	}
	return 0, false
}

// AsRPCError extracts an *RPCError from err if one is present.
func AsRPCError(err error) (*RPCError, bool) {
	var e *RPCError
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
