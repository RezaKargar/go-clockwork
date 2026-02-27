package clockwork

import (
	"context"
	"time"
)

// Storage defines persistence behavior for Clockwork request metadata.
// Implement this interface to use a custom storage backend (e.g. Redis, Memcache, or your own).
type Storage interface {
	Store(ctx context.Context, metadata *Metadata) error
	Get(ctx context.Context, id string) (*Metadata, error)
	List(ctx context.Context, limit int) ([]*Metadata, error)
	Cleanup(ctx context.Context, maxAge time.Duration) error
}

// NewStorage creates a Storage from Config using the built-in in-memory backend.
func NewStorage(cfg Config) (Storage, error) {
	cfg.Normalize()
	return NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes), nil
}
