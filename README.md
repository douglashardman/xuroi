# Xuroi

Community knowledge engine — event-sourced forum platform. [PutterTalk](https://puttertalk.com) is Customer #1.

**Building in the open.** Fresh codebase (2026); earlier experiments on this repo were scrapped.

## What this is

- **Xuroi** — forum platform (Go API, Astro web, Postgres, Redis, Mustache themes)
- **PutterTalk** — flagship site and dogfood deployment

## Repo layout

```
api/              Go — API, auth, event log, projections
web/              Astro — public SSR + authenticated shell
worker/           Async jobs (search, summaries, digests)
theme-contract/   Design handoff — schema, fixtures, THEMING.md
themes/           Site themes (puttertalk/)
sites/            Per-site config (categories, feature flags)
infra/            Docker Compose, backups, uploads
```

## Stack

| Layer | Tech |
|---|---|
| Database | PostgreSQL 16 |
| API | Go |
| Public site | Astro (SSR) |
| Themes | Mustache + CSS tokens |
| Queue | Redis (planned) |
| Cache | Redis |

## Local dev

```bash
# 1. Data services
cd infra && docker compose up -d

# 2. API
cd ../api
cp ../.env.example ../.env.local   # optional — edit as needed
go run ./cmd/xuroi                 # :8080

# 3. Seed PutterTalk categories + welcome thread
go run ./cmd/seed

# 4. Web
cd ../web
npm install
npm run dev                        # :4321
```

Public API: `http://localhost:8080` · Web: `http://localhost:4321`

## Docs

| Doc | Purpose |
|---|---|
| [PROJECT-STATE.md](./PROJECT-STATE.md) | Current focus and handoff |
| [WISH-LIST.md](./WISH-LIST.md) | Master backlog (P0–P4) |
| [theme-contract/THEMING.md](./theme-contract/THEMING.md) | Theme builder contract |

## Distribution

Every public page includes **Powered by [Xuroi](https://xuroi.com)** in the footer (engine-injected).

## License

TBD — will choose after PutterTalk launch proves the platform.