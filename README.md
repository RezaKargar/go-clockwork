# go-clockwork

`go-clockwork` is a Go package for collecting request metadata compatible with the Clockwork browser extension workflow.

## Features

- Request-scoped collectors with bounded payload limits
- Storage backends: memory, Redis, Memcache
- Gin and net/http middleware
- Minimal metadata API: `GET /__clockwork/:id`
- Zap, SQL, and cache integration adapters

## Install

```bash
go get github.com/RezaKargar/go-clockwork
```

## Quick Start (Gin)

```go
cfg := clockwork.Config{Enabled: true}
cfg.Normalize()
store := clockwork.NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes)
cw := clockwork.NewClockwork(cfg, store)

router := gin.New()
router.Use(clockwork.Middleware(cw, logger))
clockwork.RegisterRoutes(router, cw, logger)
```

## HTTP API

- `GET /__clockwork/:id`

## License

MIT
