# Architecture

`go-clockwork` follows clean architecture boundaries.

## Core Layer

Package: `github.com/RezaKargar/go-clockwork`

Responsibilities:

- Request collector lifecycle
- Metadata model
- Storage abstractions and implementations
- Core business rules (limits, truncation, telemetry aggregation)

This layer does not depend on framework-specific adapters.

## Adapter Layer

Packages:

- `github.com/RezaKargar/go-clockwork/middleware/gin`
- `github.com/RezaKargar/go-clockwork/middleware/http`
- `github.com/RezaKargar/go-clockwork/config`

Responsibilities:

- Translate framework/runtime I/O to core use cases
- Expose HTTP metadata endpoint
- Load config from `yml` and `.env`

## Integration Layer

Packages:

- `github.com/RezaKargar/go-clockwork/integrations/zap`
- `github.com/RezaKargar/go-clockwork/integrations/sql`
- `github.com/RezaKargar/go-clockwork/integrations/cache`

Responsibilities:

- Bridge external systems into core collector events
- Keep third-party coupling out of core business logic
