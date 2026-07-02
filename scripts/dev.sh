#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "→ Starting Postgres + Redis (docker compose)…"
docker compose -f infra/docker-compose.yml up -d

echo "→ Waiting for Postgres…"
for i in {1..30}; do
  if PGPASSWORD=xuroi_dev psql -h localhost -p 5433 -U xuroi -d xuroi -c 'SELECT 1' >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if [[ -f .env.local ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env.local
  set +a
fi

echo "→ API (migrations run on start)…"
(cd api && go run ./cmd/xuroi) &
API_PID=$!

sleep 2

echo "→ Web…"
(cd web && npm run dev) &
WEB_PID=$!

trap 'kill $API_PID $WEB_PID 2>/dev/null || true' EXIT

echo ""
echo "Xuroi dev:"
echo "  Web   http://localhost:4321"
echo "  API   http://localhost:8080"
echo ""
echo "Workers (optional, separate terminals):"
echo "  cd api && go run ./cmd/notify"
echo "  cd api && go run ./cmd/searchindex"
echo "  cd api && go run ./cmd/intelligence"
echo ""
wait