# Xuroi Worker

Async jobs: search indexing, thread intelligence, image pipeline.

## Intelligence (Phase 1)

Thread summaries run from a background worker (not on each HTTP post).

```bash
cd xuroi/api
go run ./cmd/intelligence --once    # one batch
go run ./cmd/intelligence           # poll every 30s
```

Summaries land in `thread_intelligence` and surface on thread pages and `/meta.json`.

### Site config (`sites/*/site.json`)

```json
"intelligence": {
  "enabled": true,
  "summary_label": "TL;DR"
}
```

- `enabled: false` — no summary UI, worker skips generation
- `summary_label` — heading above the summary (default: `"Summary"`)

### Optional LLM (env vars)

No API key → free heuristic summaries. With a key → LLM summaries, heuristic fallback on failure.

| Variable | Example |
|----------|---------|
| `XUROI_LLM_PROVIDER` | `openai` or `ollama` |
| `XUROI_LLM_API_KEY` | `sk-...` (use `ollama` for local Ollama) |
| `XUROI_LLM_MODEL` | `gpt-4o-mini` (default) |
| `XUROI_LLM_BASE_URL` | `http://localhost:11434/v1` for Ollama |

## Email notifications

Thread reply digests — **one email per thread** until the recipient visits again.

```bash
cd xuroi/api
go run ./cmd/notify --once
go run ./cmd/notify          # poll every 60s
```

### Site config (`sites/*/site.json`)

```json
"email": {
  "enabled": true,
  "from_address": "noreply@puttertalk.com",
  "from_name": "PutterTalk",
  "reply_to": "doug@puttertalk.com",
  "digest_delay_minutes": 5
}
```

### Provider (env — not committed)

| Provider | Env |
|----------|-----|
| Dev (log only) | default — emails print to stdout |
| Amazon SES | `XUROI_EMAIL_PROVIDER=ses` + `XUROI_EMAIL_FROM` + AWS credentials |
| Any SMTP | `XUROI_EMAIL_PROVIDER=smtp` + `XUROI_SMTP_HOST` + user/pass |

Visiting a thread (signed in) marks it read and cancels any pending digest for that thread.

## Search indexing (H7)

Postgres FTS — decoupled from the write path via `search_index_queue`.

```bash
cd xuroi/api
go run ./cmd/searchindex --rebuild   # full reindex once
go run ./cmd/searchindex --once      # drain queue batch
go run ./cmd/searchindex             # poll every 15s
```

Public search: `GET /v1/search?q=...` · web UI at `/search`.

Phase 3–4: magic link + password reset emails (same templates), AI moderation assist.