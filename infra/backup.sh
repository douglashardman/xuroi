#!/usr/bin/env bash
# Xuroi Postgres backup — run early, run often.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$ROOT/backups}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
FILENAME="xuroi-${TIMESTAMP}.sql.gz"

PGHOST="${PGHOST:-127.0.0.1}"
PGPORT="${PGPORT:-5433}"
PGUSER="${PGUSER:-xuroi}"
PGDATABASE="${PGDATABASE:-xuroi}"
export PGPASSWORD="${PGPASSWORD:-xuroi_dev}"

mkdir -p "$BACKUP_DIR"

echo "Backing up ${PGDATABASE}@${PGHOST}:${PGPORT} → ${BACKUP_DIR}/${FILENAME}"
pg_dump -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" --no-owner --no-acl | gzip -9 > "${BACKUP_DIR}/${FILENAME}"

# Keep last 14 daily-style backups (newest first)
ls -1t "${BACKUP_DIR}"/xuroi-*.sql.gz 2>/dev/null | tail -n +15 | while IFS= read -r old; do
  [ -n "$old" ] && rm -f "$old"
done

echo "Done. $(du -h "${BACKUP_DIR}/${FILENAME}" | cut -f1) written."