# Chi middleware for go-clockwork

Clockwork request profiling for [Chi](https://github.com/go-chi/chi) routers.

## Install

```bash
go get github.com/RezaKargar/go-clockwork/middleware/chi
```

## Usage

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    cwchi "github.com/RezaKargar/go-clockwork/middleware/chi"
    "github.com/go-chi/chi/v5"
)

store := clockwork.NewInMemoryStorage(200, 64*1024*1024)
cw := clockwork.NewClockwork(clockwork.DefaultConfig(), store)

r := chi.NewRouter()
r.Use(cwchi.Middleware(cw))
cwchi.RegisterRoutes(r, cw)
```
