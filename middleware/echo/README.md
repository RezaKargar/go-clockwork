# Echo middleware for go-clockwork

Clockwork request profiling for [Echo](https://github.com/labstack/echo) v4 apps.

## Install

```bash
go get github.com/RezaKargar/go-clockwork/middleware/echo
```

## Usage

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    cwecho "github.com/RezaKargar/go-clockwork/middleware/echo"
    "github.com/labstack/echo/v4"
)

store := clockwork.NewInMemoryStorage(200, 64*1024*1024)
cw := clockwork.NewClockwork(clockwork.DefaultConfig(), store)

e := echo.New()
e.Use(cwecho.Middleware(cw))
cwecho.RegisterRoutes(e, cw)
```
