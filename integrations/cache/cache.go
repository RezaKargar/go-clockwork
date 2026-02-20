package cache

import "github.com/RezaKargar/go-clockwork"

// Cache mirrors the core cache interface.
type Cache = clockwork.Cache

// CacheWrapper mirrors the core cache wrapper.
type CacheWrapper = clockwork.CacheWrapper

// Wrap wraps cache operations with Clockwork instrumentation.
func Wrap(underlying Cache) Cache {
	return clockwork.WrapCache(underlying)
}
