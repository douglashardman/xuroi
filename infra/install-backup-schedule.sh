#!/usr/bin/env bash
# Install macOS launchd agent — backs up Xuroi Postgres every 6 hours.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKUP_SCRIPT="$ROOT/backup.sh"
LOG_DIR="$ROOT/backups/logs"
PLIST_LABEL="com.puttertalk.xuroi-backup"
PLIST_DEST="$HOME/Library/LaunchAgents/${PLIST_LABEL}.plist"
TEMPLATE="$ROOT/com.puttertalk.xuroi-backup.plist.template"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "launchd install is macOS-only. Run backup.sh from cron on Linux:"
  echo "  0 */6 * * * $BACKUP_SCRIPT >> $LOG_DIR/backup.log 2>&1"
  exit 1
fi

chmod +x "$BACKUP_SCRIPT"
mkdir -p "$LOG_DIR" "$ROOT/backups"

sed -e "s|__BACKUP_SCRIPT__|$BACKUP_SCRIPT|g" \
    -e "s|__LOG_DIR__|$LOG_DIR|g" \
    "$TEMPLATE" > "$PLIST_DEST"

launchctl bootout "gui/$(id -u)/$PLIST_LABEL" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$PLIST_DEST"
launchctl enable "gui/$(id -u)/$PLIST_LABEL"
launchctl kickstart -k "gui/$(id -u)/$PLIST_LABEL"

echo "Installed $PLIST_LABEL"
echo "  Script:  $BACKUP_SCRIPT"
echo "  Every:   6 hours (+ once at load)"
echo "  Logs:    $LOG_DIR/"
echo "  Backups: $ROOT/backups/"
echo ""
echo "Manual run: $BACKUP_SCRIPT"
echo "Uninstall:  launchctl bootout gui/$(id -u)/$PLIST_LABEL && rm $PLIST_DEST"