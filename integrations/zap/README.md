# Zap integration for go-clockwork

Mirrors [Zap](https://go.uber.org/zap) logs into Clockwork request collectors (with trace correlation).

## Install

```bash
go get github.com/RezaKargar/go-clockwork/integrations/zap
```

## Usage

```go
import (
    clockwork "github.com/RezaKargar/go-clockwork"
    cwzap "github.com/RezaKargar/go-clockwork/integrations/zap"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

cw := clockwork.NewClockwork(cfg, store)
core := zapcore.NewCore(...)
core = cwzap.WrapCore(core, cw)
logger := zap.New(core, ...)
```

Logs that include `trace_id` are associated with the matching request; otherwise they are attached to the single active request when exactly one is active.
