# Xuroi API (Go)

Write path: auth, events, projections, REST v1.

## Phase 0 (current)

- Event schema: `../docs/event-schema.md`
- Append-only event log + online projections
- Endpoints: health, read (categories/threads), write (category/thread/post), rebuild projections

## Run locally

```bash
# Start infra (from repo root)
cd ../infra && docker compose up -d

# API
cd ../api
go mod tidy
go run ./cmd/xuroi

# Health check
curl http://localhost:8080/health
curl http://localhost:8080/v1/categories

# Auth (dev stub — email only, no password)
curl -X POST http://localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"display_name":"doug","email":"doug@puttertalk.dev"}'
```

## Environment

| Variable | Default |
|---|---|
| `XUROI_ADDR` | `:8080` |
| `DATABASE_URL` | `postgres://xuroi:xuroi_dev@localhost:5433/xuroi?sslmode=disable` |