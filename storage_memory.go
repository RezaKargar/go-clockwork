package clockwork

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type memoryEntry struct {
	id        string
	metadata  *Metadata
	createdAt time.Time
	bytes     int64
}

// InMemoryStorage keeps bounded metadata in-memory.
type InMemoryStorage struct {
	mu         sync.RWMutex
	entries    *list.List
	byID       map[string]*list.Element
	maxEntries int
	maxBytes   int64
	totalBytes int64
}

// NewInMemoryStorage builds a bounded in-memory storage.
func NewInMemoryStorage(maxEntries int, maxBytes int64) *InMemoryStorage {
	if maxEntries <= 0 {
		maxEntries = 200
	}
	if maxBytes <= 0 {
		maxBytes = 64 * 1024 * 1024
	}

	return &InMemoryStorage{
		entries:    list.New(),
		byID:       make(map[string]*list.Element, maxEntries),
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
	}
}

// Store saves metadata and applies count/size eviction.
func (s *InMemoryStorage) Store(ctx context.Context, metadata *Metadata) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}
	if metadata.ID == "" {
		return fmt.Errorf("metadata id cannot be empty")
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byID[metadata.ID]; ok {
		s.removeElementLocked(existing)
	}

	entry := &memoryEntry{
		id:        metadata.ID,
		metadata:  metadata,
		createdAt: time.Now(),
		bytes:     int64(len(payload)),
	}
	elem := s.entries.PushBack(entry)
	s.byID[metadata.ID] = elem
	s.totalBytes += entry.bytes

	s.evictLocked()
	return nil
}

// Get fetches metadata by id.
func (s *InMemoryStorage) Get(ctx context.Context, id string) (*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	elem, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("clockwork metadata not found: %s", id)
	}

	entry, _ := elem.Value.(*memoryEntry)
	if entry == nil {
		return nil, fmt.Errorf("clockwork metadata not found: %s", id)
	}

	return entry.metadata, nil
}

// List returns most recent metadata first.
func (s *InMemoryStorage) List(ctx context.Context, limit int) ([]*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = s.maxEntries
	}

	out := make([]*Metadata, 0, min(limit, s.entries.Len()))
	for elem := s.entries.Back(); elem != nil && len(out) < limit; elem = elem.Prev() {
		entry, _ := elem.Value.(*memoryEntry)
		if entry != nil && entry.metadata != nil {
			out = append(out, entry.metadata)
		}
	}

	return out, nil
}

// Cleanup removes entries older than maxAge.
func (s *InMemoryStorage) Cleanup(ctx context.Context, maxAge time.Duration) error {
	if maxAge <= 0 {
		return nil
	}

	cutoff := time.Now().Add(-maxAge)

	s.mu.Lock()
	defer s.mu.Unlock()

	for elem := s.entries.Front(); elem != nil; {
		next := elem.Next()
		entry, _ := elem.Value.(*memoryEntry)
		if entry != nil && entry.createdAt.Before(cutoff) {
			s.removeElementLocked(elem)
		}
		elem = next
	}

	return nil
}

func (s *InMemoryStorage) evictLocked() {
	for s.entries.Len() > s.maxEntries || (s.maxBytes > 0 && s.totalBytes > s.maxBytes) {
		elem := s.entries.Front()
		if elem == nil {
			break
		}
		s.removeElementLocked(elem)
	}
}

func (s *InMemoryStorage) removeElementLocked(elem *list.Element) {
	entry, _ := elem.Value.(*memoryEntry)
	if entry != nil {
		delete(s.byID, entry.id)
		s.totalBytes -= entry.bytes
		if s.totalBytes < 0 {
			s.totalBytes = 0
		}
	}
	s.entries.Remove(elem)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
