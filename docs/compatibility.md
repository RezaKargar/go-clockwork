# Compatibility Notes

## Implemented in v1

- Request capture activation via `X-Clockwork`
- Response headers: `X-Clockwork-Id`, `X-Clockwork-Version`
- Metadata retrieval: `GET /__clockwork/:id`
- Storage backends: memory, Redis, Memcache
- Integrations: zap, SQL observer, cache wrapper

## Deliberately omitted in v1

- `GET /__clockwork/latest`
- `GET /__clockwork/:id/previous`
- `GET /__clockwork/:id/next`
- `GET /__clockwork` and `GET /__clockwork/app`
