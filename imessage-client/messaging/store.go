package messaging

import (
	"errors"
	"sync"
	"time"
)

// Store tracks last seen message IDs or timestamps to filter unread results.
type Store interface {
	LastSeen(chat string) time.Time
	SetLastSeen(chat string, ts time.Time) error
}

// MemoryStore is a simple in-memory implementation suitable for short-lived sessions.
type MemoryStore struct {
	mu   sync.RWMutex
	seen map[string]time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{seen: make(map[string]time.Time)}
}

func (s *MemoryStore) LastSeen(chat string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.seen[chat]
}

func (s *MemoryStore) SetLastSeen(chat string, ts time.Time) error {
	if chat == "" {
		return errors.New("chat identifier is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[chat] = ts
	return nil
}
