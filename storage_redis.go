package clockwork

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// RedisStorage stores Clockwork payloads in Redis.
type RedisStorage struct {
	client     *redis.Client
	prefix     string
	indexKey   string
	maxEntries int
	ttl        time.Duration
}

// NewRedisStorage creates Redis-backed storage.
func NewRedisStorage(cfg Config) (Storage, error) {
	endpoint := strings.TrimSpace(cfg.RedisEndpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("redis_endpoint is required for redis storage")
	}

	ttl := cfg.RequestRetentionTime
	if ttl <= 0 {
		ttl = time.Hour
	}

	prefix := cfg.RedisPrefix
	if prefix == "" {
		prefix = "clockwork"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     endpoint,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	return &RedisStorage{
		client:     client,
		prefix:     prefix,
		indexKey:   prefix + ":index",
		maxEntries: cfg.MaxRequests,
		ttl:        ttl,
	}, nil
}

// Store saves metadata and updates recency index.
func (s *RedisStorage) Store(ctx context.Context, metadata *Metadata) error {
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
func (s *RedisStorage) Get(ctx context.Context, id string) (*Metadata, error) {
	value, err := s.client.Get(ctx, s.reqKey(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("clockwork metadata not found: %s", id)
		}
		return nil, fmt.Errorf("redis get metadata: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(value, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// List returns most recent metadata first.
func (s *RedisStorage) List(ctx context.Context, limit int) ([]*Metadata, error) {
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

	out := make([]*Metadata, 0, len(ids))
	for _, id := range ids {
		value, err := s.client.Get(ctx, s.reqKey(id)).Bytes()
		if err != nil {
			continue
		}
		var metadata Metadata
		if err := json.Unmarshal(value, &metadata); err != nil {
			continue
		}
		out = append(out, &metadata)
	}

	return out, nil
}

// Cleanup is a no-op for Redis since TTL handles expiry.
func (s *RedisStorage) Cleanup(ctx context.Context, maxAge time.Duration) error {
	return nil
}

func (s *RedisStorage) reqKey(id string) string {
	return s.prefix + ":req:" + id
}
