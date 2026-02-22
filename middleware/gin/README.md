# Gin middleware for go-clockwork

Clockwork request profiling for [Gin](https://github.com/gin-gonic/gin) apps.

## Install

```bash
go get github.com/RezaKargar/go-clockwork/middleware/gin
```

## Usage

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    ginmw "github.com/RezaKargar/go-clockwork/middleware/gin"
    "github.com/gin-gonic/gin"
)

store := clockwork.NewInMemoryStorage(200, 64*1024*1024)
cw := clockwork.NewClockwork(clockwork.DefaultConfig(), store)

router := gin.New()
router.Use(ginmw.Middleware(cw, nil))
ginmw.RegisterRoutes(router, cw, nil)
```

Pass a `clockwork.Logger` as the second argument to log persistence failures, or `nil` to ignore.
