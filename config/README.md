# Config loader for go-clockwork

Load Clockwork config from YAML and `.env` with `CLOCKWORK_*` env overrides.

## Install

```bash
go get github.com/RezaKargar/go-clockwork/config
```

## Usage

```go
import (
    "github.com/RezaKargar/go-clockwork"
    "github.com/RezaKargar/go-clockwork/config"
)

cfg, err := config.Load(config.LoadOptions{
    ConfigPath: "./configs",
    ConfigName: "clockwork",
    ConfigType: "yml",
    EnvPrefix:  "CLOCKWORK",
    EnvFiles:   []string{".env"},
})
if err != nil {
    log.Fatal(err)
}

store := clockwork.NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes)
cw := clockwork.NewClockwork(cfg, store)
```

Storage (Redis, Memcache, etc.) is configured separately; see the main README and storage package docs.
