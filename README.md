# go-clockwork

[![codecov](https://codecov.io/gh/RezaKargar/go-clockwork/graph/badge.svg)](https://codecov.io/gh/RezaKargar/go-clockwork)

**go-clockwork** is a Go port of [Clockwork](https://github.com/itsgoingd/clockwork) — the PHP debugging toolbar and request inspector that works with the [Clockwork browser extension](https://chrome.google.com/webstore/detail/clockwork/dmggabnehkmmfmdffgajcflpdjlnoemp). It collects request metadata from your Go services and exposes it via the same protocol.

## Features

- **Request-scoped collection** — Bounded payload limits and truncation
- **Pluggable storage** — In-memory (default), with optional Redis and Memcache in separate modules
- **Pluggable framework adapters** — net/http (in core), Gin, Chi, Fiber, Echo in separate modules
- **Metadata API** — `GET /__clockwork/:id` (Clockwork extension compatible)
- **Integrations** — Cache, SQL, and Zap in separate modules; implement `DataSource` for custom data
- **Interfaces** — Implement `Storage`, `DataCollector`, `DataSource`, or add your own middleware

## Install

**Core only (in-memory storage, net/http middleware):**

```bash
go get github.com/RezaKargar/go-clockwork
```

**Optional storage (install only what you need):**

```bash
go get github.com/RezaKargar/go-clockwork/storage/redis
go get github.com/RezaKargar/go-clockwork/storage/memcache
```

**Optional framework middleware:**

```bash
go get github.com/RezaKargar/go-clockwork/middleware/gin
go get github.com/RezaKargar/go-clockwork/middleware/chi
go get github.com/RezaKargar/go-clockwork/middleware/fiber
go get github.com/RezaKargar/go-clockwork/middleware/echo
```

**Optional integrations:**

```bash
go get github.com/RezaKargar/go-clockwork/integrations/cache
go get github.com/RezaKargar/go-clockwork/integrations/sql
go get github.com/RezaKargar/go-clockwork/integrations/zap
go get github.com/RezaKargar/go-clockwork/config
```

## Quick Start (Gin)

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    ginmw "github.com/RezaKargar/go-clockwork/middleware/gin"
    "github.com/gin-gonic/gin"
)

cfg := clockwork.DefaultConfig()
cfg.Normalize()
store := clockwork.NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes)
cw := clockwork.NewClockwork(cfg, store)

router := gin.New()
router.Use(ginmw.Middleware(cw, nil)) // pass clockwork.Logger or nil
ginmw.RegisterRoutes(router, cw, nil)
```

## Quick Start (net/http)

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    clockworkhttp "github.com/RezaKargar/go-clockwork/middleware/http"
    "net/http"
)

store := clockwork.NewInMemoryStorage(200, 64*1024*1024)
cw := clockwork.NewClockwork(clockwork.DefaultConfig(), store)

mux := http.NewServeMux()
mux.Handle("/", yourHandler)
clockworkhttp.RegisterMetadataRoute(mux, cw)

handler := clockworkhttp.Middleware(cw, mux)
http.ListenAndServe(":8080", handler)
```

## Custom storage

Implement the `Storage` interface and pass it to `NewClockwork`:

```go
type MyStorage struct{}
func (s *MyStorage) Store(ctx context.Context, m *clockwork.Metadata) error { ... }
func (s *MyStorage) Get(ctx context.Context, id string) (*clockwork.Metadata, error) { ... }
func (s *MyStorage) List(ctx context.Context, limit int) ([]*clockwork.Metadata, error) { ... }
func (s *MyStorage) Cleanup(ctx context.Context, maxAge time.Duration) error { ... }

cw := clockwork.NewClockwork(cfg, &MyStorage{})
```

## DataSource and custom data

Register a `DataSource` to attach custom data when each request completes. Use `SetUserData` on the collector to add key-value data that appears in `Metadata.UserData`:

```go
type myDataSource struct{}
func (d *myDataSource) Name() string { return "custom" }
func (d *myDataSource) Resolve(ctx context.Context, c clockwork.DataCollector) {
    c.SetUserData("memory_alloc_mb", runtimeMemMB())
}

cw := clockwork.NewClockwork(cfg, store)
cw.RegisterDataSource(&myDataSource{})
```

## Extending middleware

Framework adapters use the same flow: call `clockwork.NewRequestCapture(cw, method, path, uri, headers)`; if it returns `(collector, true)`, set headers/URL/trace on the collector, put it in context, run the handler, then `cw.CompleteRequest(ctx, collector, status, duration)`. See [docs/architecture.md](docs/architecture.md) for the middleware contract.

## HTTP API

- `GET /__clockwork/:id` — Returns captured metadata for the given request ID.

## Module layout

| Path | Description |
|------|-------------|
| `github.com/RezaKargar/go-clockwork` | Core: collector, in-memory storage, interfaces, net/http middleware |
| `.../storage/redis` | Redis storage backend |
| `.../storage/memcache` | Memcache storage backend |
| `.../middleware/gin` | Gin middleware and routes |
| `.../middleware/chi` | Chi middleware and routes |
| `.../middleware/fiber` | Fiber middleware and routes |
| `.../middleware/echo` | Echo middleware and routes |
| `.../middleware/http` | net/http middleware (part of core module) |
| `.../integrations/cache` | Cache wrapper (separate module) |
| `.../integrations/sql` | SQL observer (separate module) |
| `.../integrations/zap` | Zap log integration |
| `.../config` | YAML + env config loader |

See [docs/architecture.md](docs/architecture.md) and [docs/migration.md](docs/migration.md) for details.

## License

MIT
