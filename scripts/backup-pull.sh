#!/bin/bash
# Mac-side: pull the ECS's nightly backups to ~/Backups/hackforger/.
# Invoked by launchd job com.h2os.hackforger-backup-pull at 04:30 daily.

set -euo pipefail

DEST="${HOME}/Backups/hackforger"
mkdir -p "$DEST"/{db,data,custom}

REMOTE="hackforger@203.119.115.130"
LOG="$DEST/backup.log"

log() {
  printf "%s %s\n" "$(date -Iseconds)" "$*" | tee -a "$LOG"
}

log "===== backup-pull start ====="

rsync -az --delete \
  "$REMOTE:/var/lib/hackforger/pg-backups/" "$DEST/db/" \
  2>&1 | tail -5 | sed 's/^/  /' | tee -a "$LOG"

rsync -az --delete \
  "$REMOTE:/var/lib/hackforger/data/" "$DEST/data/" \
  2>&1 | tail -5 | sed 's/^/  /' | tee -a "$LOG"

rsync -az --delete \
  "$REMOTE:/var/lib/hackforger/custom/" "$DEST/custom/" \
  2>&1 | tail -5 | sed 's/^/  /' | tee -a "$LOG"

NEWEST=$(find "$DEST/db" -name 'hf-*.pgc' -mtime -1 | head -1 || true)
if [ -z "$NEWEST" ]; then
  log "WARN: no .pgc file <24h old in $DEST/db/"
else
  log "newest dump: $NEWEST"
fi

log "===== backup-pull OK ====="
