# Fiber middleware for go-clockwork

Clockwork request profiling for [Fiber](https://github.com/gofiber/fiber) v2 apps.

## Install

```bash
go get github.com/RezaKargar/go-clockwork/middleware/fiber
```

## Usage

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    cwfiber "github.com/RezaKargar/go-clockwork/middleware/fiber"
    "github.com/gofiber/fiber/v2"
)

store := clockwork.NewInMemoryStorage(200, 64*1024*1024)
cw := clockwork.NewClockwork(clockwork.DefaultConfig(), store)

app := fiber.New()
app.Use(cwfiber.Middleware(cw))
cwfiber.RegisterRoutes(app, cw)
```

Use `c.UserContext()` in handlers when passing context to DB/cache so the collector is available to integrations.
