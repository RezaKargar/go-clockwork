package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RezaKargar/go-clockwork"
	redis "github.com/redis/go-redis/v9"
)

// Config holds Redis storage configuration.
type Config struct {
	Endpoint   string
	Password   string
	DB         int
	Prefix     string
	TTL        time.Duration
	MaxEntries int
}

// Storage implements clockwork.Storage using Redis.
type Storage struct {
	client     *redis.Client
	prefix     string
	indexKey   string
	maxEntries int
	ttl        time.Duration
}

// New creates Redis-backed storage for Clockwork.
func New(cfg Config) (clockwork.Storage, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("redis endpoint is required")
	}

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = time.Hour
	}

	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = "clockwork"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     endpoint,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &Storage{
		client:     client,
		prefix:     prefix,
		indexKey:   prefix + ":index",
		maxEntries: cfg.MaxEntries,
		ttl:        ttl,
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

	key := s.reqKey(metadata.ID)
	pipe := s.client.Pipeline()
	pipe.Set(ctx, key, payload, s.ttl)
	pipe.LPush(ctx, s.indexKey, metadata.ID)
	if s.maxEntries > 0 {
		pipe.LTrim(ctx, s.indexKey, 0, int64(s.maxEntries-1))
	}
	pipe.Expire(ctx, s.indexKey, s.ttl)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis store metadata: %w", err)
	}

	return nil
}

// Get fetches metadata by id.
func (s *Storage) Get(ctx context.Context, id string) (*clockwork.Metadata, error) {
	value, err := s.client.Get(ctx, s.reqKey(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("clockwork metadata not found: %s", id)
		}
		return nil, fmt.Errorf("redis get metadata: %w", err)
	}

	var metadata clockwork.Metadata
	if err := json.Unmarshal(value, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// List returns most recent metadata first.
func (s *Storage) List(ctx context.Context, limit int) ([]*clockwork.Metadata, error) {
	if limit <= 0 {
		limit = s.maxEntries
	}
	if limit <= 0 {
		limit = 50
	}

	ids, err := s.client.LRange(ctx, s.indexKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis list index: %w", err)
	}

	out := make([]*clockwork.Metadata, 0, len(ids))
	for _, id := range ids {
		value, err := s.client.Get(ctx, s.reqKey(id)).Bytes()
		if err != nil {
			continue
		}
		var metadata clockwork.Metadata
		if err := json.Unmarshal(value, &metadata); err != nil {
			continue
		}
		out = append(out, &metadata)
	}

	return out, nil
}

// Cleanup is a no-op for Redis since TTL handles expiry.
func (s *Storage) Cleanup(ctx context.Context, maxAge time.Duration) error {
	return nil
}

func (s *Storage) reqKey(id string) string {
	return s.prefix + ":req:" + id
}
