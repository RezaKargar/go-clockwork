# Compatibility Notes

## Implemented

- Request capture activation via `X-Clockwork`
- Response headers: `X-Clockwork-Id`, `X-Clockwork-Version`
- Metadata retrieval: `GET /__clockwork/:id`
- Storage: in-memory (core), Redis and Memcache (separate modules)
- Middleware: net/http (core), Gin, Chi, Fiber, Echo (separate modules)
- Integrations: cache, SQL (core), Zap (separate module)
- Config loader (separate module)

## Deliberately omitted

- `GET /__clockwork/latest`
- `GET /__clockwork/:id/previous`
- `GET /__clockwork/:id/next`
- `GET /__clockwork` and `GET /__clockwork/app`
