package cache

import (
	"context"
	"time"

	"github.com/RezaKargar/go-clockwork"
)

// Cache defines the minimal cache behavior required by Wrap.
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Wrapper wraps Cache to emit cache telemetry to active Clockwork collectors.
type Wrapper struct {
	underlying Cache
}

// Wrap wraps cache operations with Clockwork instrumentation.
func Wrap(underlying Cache) Cache {
	if underlying == nil {
		return nil
	}
	return &Wrapper{underlying: underlying}
}

// Get retrieves a value from cache and records hit/miss.
func (c *Wrapper) Get(ctx context.Context, key string) (interface{}, bool) {
	collector := clockwork.CollectorFromContext(ctx)
	startTime := time.Now()
	value, found := c.underlying.Get(ctx, key)
	duration := time.Since(startTime)

	if collector != nil {
		cacheType := "miss"
		if found {
			cacheType = "hit"
		}
		collector.AddCacheQuery(cacheType, key, duration)
	}

	return value, found
}

// Set stores a value in cache and records write.
func (c *Wrapper) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	collector := clockwork.CollectorFromContext(ctx)
	startTime := time.Now()
	err := c.underlying.Set(ctx, key, value, ttl)
	duration := time.Since(startTime)

	if collector != nil && err == nil {
		collector.AddCacheQuery("write", key, duration)
	}

	return err
}

// Delete removes a value from cache and records delete.
func (c *Wrapper) Delete(ctx context.Context, key string) error {
	collector := clockwork.CollectorFromContext(ctx)
	startTime := time.Now()
	err := c.underlying.Delete(ctx, key)
	duration := time.Since(startTime)

	if collector != nil && err == nil {
		collector.AddCacheQuery("delete", key, duration)
	}

	return err
}
