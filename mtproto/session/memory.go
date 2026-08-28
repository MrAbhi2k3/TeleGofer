package session

import "sync"

// Memory keeps session data in process memory. Safe for concurrent use.
// Data is lost when the process exits.
type Memory struct {
	mu   sync.Mutex
	data []byte
}

func (m *Memory) Load() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		return nil, ErrNotFound
	}
	out := make([]byte, len(m.data))
	copy(out, m.data)
	return out, nil
}

func (m *Memory) Save(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make([]byte, len(data))
	copy(m.data, data)
	return nil
}
