package clockwork

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Storage defines persistence behavior for Clockwork request metadata.
type Storage interface {
	Store(ctx context.Context, metadata *Metadata) error
	Get(ctx context.Context, id string) (*Metadata, error)
	List(ctx context.Context, limit int) ([]*Metadata, error)
	Cleanup(ctx context.Context, maxAge time.Duration) error
}

// NewStorage creates a storage backend using Clockwork config.
func NewStorage(cfg Config) (Storage, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.StorageType)) {
	case "", storageMemory:
		return NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes), nil
	case storageRedis:
		return NewRedisStorage(cfg)
	case storageMemcache, "memcached":
		return NewMemcacheStorage(cfg)
	default:
		return nil, fmt.Errorf("unknown clockwork storage type: %s", cfg.StorageType)
	}
}
