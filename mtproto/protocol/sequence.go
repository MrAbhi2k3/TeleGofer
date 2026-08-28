package protocol

import (
	"sync"
)

// SequenceGenerator manages Telegram message sequence numbers (seq_no) for an MTProto session.
//
// Per MTProto spec:
// If a message is content-related (requires acknowledgement):
// seq_no = content_related_count * 2 + 1
// If a message is service/not content-related (e.g. msgs_ack, ping):
// seq_no = content_related_count * 2
type SequenceGenerator struct {
	mu           sync.Mutex
	contentCount int32
}

// NewSequenceGenerator creates a sequence generator initialized to 0.
func NewSequenceGenerator() *SequenceGenerator {
	return &SequenceGenerator{}
}

// Next returns the appropriate seq_no and updates internal counters.
func (s *SequenceGenerator) Next(contentRelated bool) int32 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if contentRelated {
		seqNo := s.contentCount*2 + 1
		s.contentCount++
		return seqNo
	}

	return s.contentCount * 2
}

// Reset resets the content count to zero (used on new session creation).
func (s *SequenceGenerator) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contentCount = 0
}
