# Installing Xuroi

Xuroi ships as a **git clone** (downloadable) with a one-command dev bootstrap. An npm package for the web UI may follow; for now the repo is the install unit.

## Requirements

- Go 1.22+
- Node 20+
- Docker (Postgres + Redis via `infra/docker-compose.yml`)
- `psql` client (for health checks in `scripts/dev.sh`)

## Quick start

```bash
git clone https://github.com/douglashardman/xuroi.git
cd xuroi
cp .env.example .env.local   # if present; set SITE_URL, DATABASE_URL, SES keys
make infra                   # Postgres :5433, Redis
make seed                    # optional: seed PutterTalk categories
make dev                     # API :8080 + web :4321
```

Or: `./scripts/dev.sh`

## Site configuration

- **Site config:** `sites/puttertalk/site.json` (name, email, moderation, spam, new-user rules)
- **Admin UI:** `/admin/settings` edits core fields and writes back to `site.json`
- **Production:** set `SITE_JSON` and `SITE_URL` env vars; point `DATABASE_URL` at your Postgres

## Workers (production)

Run alongside the API:

- `go run ./cmd/notify` — email digests + @mentions
- `go run ./cmd/searchindex` — full-text search queue
- `go run ./cmd/intelligence` — thread summaries

## Deploy model

1. Build Astro web (`cd web && npm run build`) with adapter for your host
2. Run `go build -o xuroi ./cmd/xuroi` for the API binary
3. Postgres migrations apply automatically on API start
4. Put uploads on disk or S3; set `MEDIA_UPLOAD_DIR` or future object-store env

Ops (DNS, TLS, CDN) are host-specific — wire those on your iron; the app assumes a reverse proxy in front of API + web.