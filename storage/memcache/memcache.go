package memcache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/RezaKargar/go-clockwork"
	"github.com/bradfitz/gomemcache/memcache"
)

// Config holds Memcache storage configuration.
type Config struct {
	Endpoints   []string
	Prefix      string
	TTL         time.Duration
	MaxEntries int
}

// Storage implements clockwork.Storage using Memcached.
type Storage struct {
	client     *memcache.Client
	prefix     string
	indexKey   string
	maxEntries int
	ttlSeconds int32
	mu         sync.Mutex
}

// New creates Memcached-backed storage for Clockwork.
func New(cfg Config) (clockwork.Storage, error) {
	endpoints := make([]string, 0, len(cfg.Endpoints))
	for _, endpoint := range cfg.Endpoints {
		trimmed := strings.TrimSpace(endpoint)
		if trimmed != "" {
			endpoints = append(endpoints, trimmed)
		}
	}
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("memcache endpoints are required")
	}

	ttl := int32(cfg.TTL.Seconds())
	if ttl <= 0 {
		ttl = int32((time.Hour).Seconds())
	}

	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = "clockwork"
	}

	return &Storage{
		client:     memcache.New(endpoints...),
		prefix:     prefix,
		indexKey:   prefix + ":index",
		maxEntries: cfg.MaxEntries,
		ttlSeconds: ttl,
	}, nil
}

// Store saves metadata and updates recency index.
func (s *Storage) Store(ctx context.Context, metadata *clockwork.Metadata) error {
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

	item := &memcache.Item{
		Key:        s.reqKey(metadata.ID),
		Value:      payload,
		Expiration: s.ttlSeconds,
	}
	if err := s.client.Set(item); err != nil {
		return fmt.Errorf("memcache set metadata: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids, err := s.loadIndexLocked()
	if err != nil {
		return err
	}

	ids = prependUnique(ids, metadata.ID)
	evicted := make([]string, 0)
	if s.maxEntries > 0 && len(ids) > s.maxEntries {
		evicted = append(evicted, ids[s.maxEntries:]...)
		ids = ids[:s.maxEntries]
	}

	if err := s.saveIndexLocked(ids); err != nil {
		return err
	}

	for _, id := range evicted {
		_ = s.client.Delete(s.reqKey(id))
	}

	return nil
}

// Get fetches metadata by id.
func (s *Storage) Get(ctx context.Context, id string) (*clockwork.Metadata, error) {
	item, err := s.client.Get(s.reqKey(id))
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return nil, fmt.Errorf("clockwork metadata not found: %s", id)
		}
		return nil, fmt.Errorf("memcache get metadata: %w", err)
	}

	var metadata clockwork.Metadata
	if err := json.Unmarshal(item.Value, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// List returns most recent metadata first.
func (s *Storage) List(ctx context.Context, limit int) ([]*clockwork.Metadata, error) {
	s.mu.Lock()
	ids, err := s.loadIndexLocked()
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > len(ids) {
		limit = len(ids)
	}

	out := make([]*clockwork.Metadata, 0, limit)
	for i := 0; i < limit; i++ {
		item, err := s.client.Get(s.reqKey(ids[i]))
		if err != nil {
			continue
		}
		var metadata clockwork.Metadata
		if err := json.Unmarshal(item.Value, &metadata); err != nil {
			continue
		}
		out = append(out, &metadata)
	}

	return out, nil
}

// Cleanup is a no-op for Memcached since TTL handles expiry.
func (s *Storage) Cleanup(ctx context.Context, maxAge time.Duration) error {
	return nil
}

func (s *Storage) reqKey(id string) string {
	return s.prefix + ":req:" + id
}

func (s *Storage) loadIndexLocked() ([]string, error) {
	item, err := s.client.Get(s.indexKey)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return []string{}, nil
		}
		return nil, fmt.Errorf("memcache get index: %w", err)
	}

	var ids []string
	if err := json.Unmarshal(item.Value, &ids); err != nil {
		return nil, fmt.Errorf("unmarshal index: %w", err)
	}
	return ids, nil
}

func (s *Storage) saveIndexLocked(ids []string) error {
	payload, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	item := &memcache.Item{
		Key:        s.indexKey,
		Value:      payload,
		Expiration: s.ttlSeconds,
	}
	if err := s.client.Set(item); err != nil {
		return fmt.Errorf("memcache set index: %w", err)
	}
	return nil
}

func prependUnique(in []string, id string) []string {
	out := make([]string, 0, len(in)+1)
	out = append(out, id)
	for _, v := range in {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}
