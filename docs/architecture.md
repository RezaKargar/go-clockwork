# Architecture

`go-clockwork` uses a multi-module layout. The core module has minimal dependencies; storage backends, framework adapters, and optional integrations live in separate modules so you only depend on what you use.

## Core Layer

Package: `github.com/RezaKargar/go-clockwork`

- Request collector lifecycle and `DataCollector` interface
- Metadata model (`Metadata`, `LogTraceFrame`, `UserData`, etc.)
- `Storage` interface and in-memory implementation only
- `DataSource` interface for pluggable data (e.g. custom metrics) via `RegisterDataSource`
- Helper functions for middleware (`ShouldSkipPath`, `ShouldCapture`, `BuildRequestURL`, `ExtractSafeHeaders`, `TraceFromContext`, `NewRequestCapture`)
- net/http middleware (`middleware/http` package, part of core module)

Core does not import Gin, Chi, Fiber, Echo, Redis, Memcache, Zap, or Viper. Integrations (cache, SQL, zap) are separate modules.

## Storage modules

- `github.com/RezaKargar/go-clockwork/storage/redis` — Redis-backed storage
- `github.com/RezaKargar/go-clockwork/storage/memcache` — Memcache-backed storage

Implement the `Storage` interface to add a custom backend.

## Adapter layer (middleware)

- `github.com/RezaKargar/go-clockwork/middleware/http` — net/http (in core)
- `github.com/RezaKargar/go-clockwork/middleware/gin`
- `github.com/RezaKargar/go-clockwork/middleware/chi`
- `github.com/RezaKargar/go-clockwork/middleware/fiber`
- `github.com/RezaKargar/go-clockwork/middleware/echo`

Each adapter uses the core helpers and implements framework-specific middleware and route registration.

**Middleware contract:** To add support for another framework, (1) call `clockwork.NewRequestCapture(cw, method, path, uri, headers)`; if it returns `(nil, false)`, skip profiling and run the next handler; (2) otherwise set headers, URL, and trace on the collector, put it in request context via `ContextWithCollector`, set response headers `X-Clockwork-Id` and `X-Clockwork-Version`, run the handler, then call `cw.CompleteRequest(ctx, collector, status, duration)`.

## Integration layer

- `github.com/RezaKargar/go-clockwork/integrations/cache` — Cache wrapper (separate module)
- `github.com/RezaKargar/go-clockwork/integrations/sql` — SQL observer (separate module)
- `github.com/RezaKargar/go-clockwork/integrations/zap` — Zap core wrapper (separate module)

## Config module

- `github.com/RezaKargar/go-clockwork/config` — Load config from YAML and `.env` with `CLOCKWORK_*` overrides.

## Interfaces

- **Storage** — `Store`, `Get`, `List`, `Cleanup`. Implement for custom backends.
- **DataCollector** — Methods to record queries, logs, timeline events, and `SetUserData` for custom key-value data. The built-in `*Collector` implements it; custom collectors can implement it for alternate data sources.
- **DataSource** — `Name() string`, `Resolve(ctx, collector)`. Register with `Clockwork.RegisterDataSource` to run when each request completes; use `collector.SetUserData` to attach data to `Metadata.UserData`.
- **Logger** — `Warn(msg string, keysAndValues ...interface{})`. Used by middleware when persistence fails.
