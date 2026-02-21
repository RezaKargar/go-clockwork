# go-clockwork

[![codecov](https://codecov.io/gh/RezaKargar/go-clockwork/graph/badge.svg)](https://codecov.io/gh/RezaKargar/go-clockwork)

**go-clockwork** is a Go port of [Clockwork](https://github.com/itsgoingd/clockwork) — the PHP debugging toolbar and request inspector that works with the [Clockwork browser extension](https://chrome.google.com/webstore/detail/clockwork/dmggabnehkmmfmdffgajcflpdjlnoemp). It collects request metadata from your Go services and exposes it via the same protocol, so you can inspect requests, logs, and performance in the browser or in Chrome DevTools.

## Features

- **Request-scoped collection** — Bounded payload limits and truncation to avoid unbounded memory use
- **Storage backends** — In-memory, Redis, and Memcache for request metadata
- **Framework adapters** — Gin and `net/http` middleware; activates only when the `X-Clockwork` header is present
- **Metadata API** — `GET /__clockwork/:id` to retrieve captured data (compatible with the Clockwork extension)
- **Integrations** — Zap (logging), SQL query observation, and cache wrapper for timeline events
- **Config** — YAML and `.env` loading with `CLOCKWORK_*` env overrides
- **Clean layout** — Core domain, adapter packages, and integration packages kept separate

## Install

```bash
go get github.com/RezaKargar/go-clockwork
```

## Architecture

- `github.com/RezaKargar/go-clockwork`: core domain and use-case logic
- `github.com/RezaKargar/go-clockwork/middleware/gin`: Gin adapter
- `github.com/RezaKargar/go-clockwork/middleware/http`: net/http adapter
- `github.com/RezaKargar/go-clockwork/integrations/*`: external integration adapters
- `github.com/RezaKargar/go-clockwork/config`: yml + `.env` config adapter

## Quick Start (Gin)

```go
cfg, err := config.Load(config.LoadOptions{
    ConfigPath: "./configs",
    ConfigName: "custom",
    ConfigType: "yml",
    EnvPrefix:  "CLOCKWORK",
    EnvFiles:   []string{"./configs/.env"},
})
if err != nil {
    panic(err)
}
store := clockwork.NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes)
cw := clockwork.NewClockwork(cfg, store)

router := gin.New()
router.Use(ginmw.Middleware(cw, logger))
ginmw.RegisterRoutes(router, cw, logger)
```

## HTTP API

- `GET /__clockwork/:id` — Returns captured metadata for the given request ID (used by the Clockwork extension).

## License

MIT
