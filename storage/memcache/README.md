# Memcache storage for go-clockwork

Memcached-backed storage for Clockwork request metadata.

## Install

```bash
go get github.com/RezaKargar/go-clockwork/storage/memcache
```

## Usage

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    cwmemcache "github.com/RezaKargar/go-clockwork/storage/memcache"
    "time"
)

store, err := cwmemcache.New(cwmemcache.Config{
    Endpoints:  []string{"127.0.0.1:11211"},
    Prefix:     "clockwork",
    TTL:        time.Hour,
    MaxEntries: 200,
})
if err != nil {
    log.Fatal(err)
}

cfg := clockwork.DefaultConfig()
cw := clockwork.NewClockwork(cfg, store)
```

## Config

- **Endpoints** — Memcached server addresses (required).
- **Prefix** — Key prefix (default `"clockwork"`).
- **TTL** — Expiration for stored entries.
- **MaxEntries** — Max IDs kept in the index (optional).
