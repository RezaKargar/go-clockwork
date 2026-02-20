# go-clockwork

`go-clockwork` is a Go package for collecting request metadata compatible with the Clockwork browser extension workflow.

## Features

- Request-scoped collectors with bounded payload limits
- Storage backends: memory, Redis, Memcache
- Clean architecture layers: core domain, adapter packages, and integration packages
- Gin and net/http middleware adapters
- Minimal metadata API: `GET /__clockwork/:id`
- Zap, SQL, and cache integration adapters
- Config loading from `yml` and `.env`

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

- `GET /__clockwork/:id`

## License

MIT
