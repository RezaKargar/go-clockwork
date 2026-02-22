# SQL integration for go-clockwork

Records SQL query observations (query, duration, connection, slow-query flag) on the active Clockwork request collector.

## Install

```bash
go get github.com/RezaKargar/go-clockwork/integrations/sql
```

## Usage

```go
import (
    "github.com/RezaKargar/go-clockwork"
    "github.com/RezaKargar/go-clockwork/integrations/sql"
)

observer := sql.NewObserver(cw, slowQueryThreshold) // or 0 to use cw.Config().SlowQueryThreshold
// After each query, with request context:
observer.OnQuery(ctx, sql.Observation{
    Operation:  "SELECT",
    Query:      "SELECT ...",
    Duration:   elapsed,
    Connection: "dbname",
})
```

Ensure the request context carries the Clockwork collector (middleware does this). Queries longer than the threshold are marked as slow in the timeline.
