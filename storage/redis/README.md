# Redis storage for go-clockwork

Redis-backed storage for Clockwork request metadata.

## Install

```bash
go get github.com/RezaKargar/go-clockwork/storage/redis
```

## Usage

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    cwredis "github.com/RezaKargar/go-clockwork/storage/redis"
)

store, err := cwredis.New(cwredis.Config{
    Endpoint:   "localhost:6379",
    Password:   "",
    DB:        0,
    Prefix:    "clockwork",
    TTL:       time.Hour,
    MaxEntries: 200,
})
if err != nil {
    log.Fatal(err)
}

cfg := clockwork.DefaultConfig()
cw := clockwork.NewClockwork(cfg, store)
```

## Config

- **Endpoint** — Redis address (required).
- **Password** — Optional.
- **DB** — Redis DB index (default 0).
- **Prefix** — Key prefix (default `"clockwork"`).
- **TTL** — Expiration for stored entries.
- **MaxEntries** — Max IDs kept in the index (optional).
