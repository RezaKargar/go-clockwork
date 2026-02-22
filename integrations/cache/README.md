# Cache integration for go-clockwork

Wraps any cache that implements the `Cache` interface so Get/Set/Delete are recorded in the active Clockwork request (hit, miss, write, delete with duration).

## Install

```bash
go get github.com/RezaKargar/go-clockwork/integrations/cache
```

## Usage

```go
import (
    "github.com/RezaKargar/go-clockwork/integrations/cache"
)

// Your cache must implement: Get(ctx, key), Set(ctx, key, value, ttl), Delete(ctx, key)
wrapped := cache.Wrap(yourCache)
// Use wrapped in handlers; ensure request context has the Clockwork collector (middleware does this).
```

The wrapper records each operation (type, key, duration) on the collector from `clockwork.CollectorFromContext(ctx)`.
