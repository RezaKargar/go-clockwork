# Migration guide (modular refactor)

If you were using go-clockwork before the modular refactor, update as follows.

## Config and storage

**Before:** Config included `StorageType`, `RedisEndpoint`, `MemcacheEndpoints`, etc., and you used `clockwork.NewStorage(cfg)`.

**After:** Core `Config` no longer has storage-specific fields. Create storage yourself:

- **In-memory (default):** `store := clockwork.NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes)`
- **Redis:** `go get .../storage/redis`, then `store, err := redis.New(redis.Config{Endpoint: "...", TTL: time.Hour, MaxEntries: 200})`
- **Memcache:** `go get .../storage/memcache`, then `store, err := memcache.New(memcache.Config{Endpoints: []string{"..."}, TTL: time.Hour, MaxEntries: 200})`

Then: `cw := clockwork.NewClockwork(cfg, store)`.

## Gin middleware

**Before:** `import "github.com/RezaKargar/go-clockwork"` and `clockwork.Middleware(cw, logger)` / `clockwork.RegisterRoutes(router, cw, logger)`.

**After:** Add the Gin module and use the gin package:

```bash
go get github.com/RezaKargar/go-clockwork/middleware/gin
```

```go
import ginmw "github.com/RezaKargar/go-clockwork/middleware/gin"

router.Use(ginmw.Middleware(cw, logger))
ginmw.RegisterRoutes(router, cw, logger)
```

`logger` can be `nil` or any type that implements `clockwork.Logger` (`Warn(msg string, keysAndValues ...interface{})`).

## net/http middleware

No change: still `import clockworkhttp "github.com/RezaKargar/go-clockwork/middleware/http"` and use `Middleware`, `MetadataHandler`, `RegisterMetadataRoute`. The package remains in the core module.

## Cache integration

**Before:** `clockwork.WrapCache(cache)` and `clockwork.Cache` / `clockwork.CacheWrapper`, or cache in core.

**After:** Add the cache integration module and use the cache package:

```bash
go get github.com/RezaKargar/go-clockwork/integrations/cache
```

```go
import "github.com/RezaKargar/go-clockwork/integrations/cache"

wrapped := cache.Wrap(yourCache)
```

Types: `cache.Cache`, `cache.CacheWrapper`, `cache.Wrap`.

## SQL integration

**Before:** `clockwork.NewSQLObserver(cw, threshold)` and `clockwork.QueryObservation`, or sql in core.

**After:** Add the sql integration module and use the sql package:

```bash
go get github.com/RezaKargar/go-clockwork/integrations/sql
```

```go
import "github.com/RezaKargar/go-clockwork/integrations/sql"

observer := sql.NewObserver(cw, threshold)
observer.OnQuery(ctx, sql.Observation{Query: "...", Duration: d, ...})
```

## Zap integration

**Before:** `clockwork.WrapZapCore(core, cw)` in core.

**After:** Add the zap integration module:

```bash
go get github.com/RezaKargar/go-clockwork/integrations/zap
```

```go
import cwzap "github.com/RezaKargar/go-clockwork/integrations/zap"

core = cwzap.WrapCore(core, cw)
```

## Config loading

**Before:** `config.Load(...)` returned `clockwork.Config` that could include `StorageType`, Redis/Memcache fields.

**After:** Config package is a separate module. Install with `go get github.com/RezaKargar/go-clockwork/config`. `config.Load` still returns `clockwork.Config`; it no longer sets storage-related fields. Configure storage separately (see above).
