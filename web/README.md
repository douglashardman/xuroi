# Xuroi Web (Astro)

Public SSR read path. Fetches from Go API at build/request time.

## Run locally

```bash
# API must be running on :8080
npm install
npm run dev    # http://localhost:4321
```

## Environment

| Variable | Default |
|---|---|
| `PUBLIC_API_URL` | `http://localhost:8080` |

## Pages

- `/` — category index
- `/c/{slug}` — threads in category
- `/t/{slug}--{id}` — thread with posts