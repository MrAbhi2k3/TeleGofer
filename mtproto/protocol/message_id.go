package protocol

import (
	"sync"
	"time"
)

// MessageIDGenerator produces monotonically increasing 64-bit Telegram message identifiers.
//
// Per MTProto spec:
// msg_id = (unixtime * 2^32) | (nano * 4)
// For client requests, msg_id % 4 must equal 0.
// Every message ID must be strictly greater than any previously generated message ID.
type MessageIDGenerator struct {
	mu     sync.Mutex
	lastID int64
	offset time.Duration // server time offset
}

// NewMessageIDGenerator creates a message ID generator.
func NewMessageIDGenerator() *MessageIDGenerator {
	return &MessageIDGenerator{}
}

// SetServerTimeOffset sets the clock difference between the local system and Telegram's server.
func (g *MessageIDGenerator) SetServerTimeOffset(offset time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.offset = offset
}

// Next generates the next valid client message ID.
func (g *MessageIDGenerator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().Add(g.offset)
	sec := now.Unix()
	nano := now.Nanosecond()

	// 32 bits for unix seconds, upper 30 bits of nanoseconds, lower 2 bits = 0 (client message)
	id := (sec << 32) | (int64(nano/4) << 2)

	// Ensure strictly monotonic increase
	if id <= g.lastID {
		id = g.lastID + 4
	}

	g.lastID = id
	return id
}
