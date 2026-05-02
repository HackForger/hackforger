#!/bin/bash
# One-shot pull of the production DB into the Mac dev instance.
#
# Use this when you want the Mac instance to mirror prod state — e.g. after
# someone made an important change in production (admin user rename, new
# hackathon, content edit) and you want to reproduce/test against that on
# your laptop.
#
# Mechanism:
#   1. Trigger a FRESH pg_dump on the ECS via the existing backup systemd unit
#      (so we don't depend on the nightly schedule).
#   2. rsync the new .pgc file to ~/Backups/hackforger/db/ on the Mac.
#   3. Stop Mac gitea (port 3000).
#   4. Drop + recreate the local `hackforger` DB.
#   5. pg_restore.
#   6. Restart Mac gitea.
#
# This is DESTRUCTIVE on the Mac: any local-only DB changes are wiped.
# That's intentional — the Mac is the dev/standby instance, prod is truth.
#
# Repos and attachments are NOT touched by this script. Use the rsync from
# scripts/backup-pull.sh if you also want repo/attachment files synced.
#
# Usage:
#   bash scripts/sync-from-prod.sh --confirm

set -euo pipefail

CONFIRM=0
for arg in "$@"; do
  case "$arg" in
    --confirm) CONFIRM=1 ;;
    *) echo "Usage: $0 --confirm" >&2; exit 64 ;;
  esac
done
if [ "$CONFIRM" = "0" ]; then
  cat >&2 <<'EOF'
This script will:
  - Stop Mac gitea on port 3000
  - DROP and recreate the local 'hackforger' Postgres database
  - pg_restore from a fresh ECS dump (any local DB changes will be lost)
  - Restart Mac gitea

Pass --confirm if you understand and want to proceed.
EOF
  exit 1
fi

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"

ECS=hackforger@203.119.115.130
DEST="$HOME/Backups/hackforger/db"

log() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }

# ---------------------------------------------------------------------------
log "[1/6] Read Mac PG password + ECS sudo password from .env"
if ! grep -q "^PGPASSWORD=" .env; then
  echo "FATAL: PGPASSWORD not in .env" >&2
  exit 1
fi
PGPASSWORD=$(grep "^PGPASSWORD=" .env | cut -d= -f2)
export PGPASSWORD
if ! grep -q "^ECS_SUDO_PASS=" .env; then
  echo "FATAL: ECS_SUDO_PASS not in .env" >&2
  exit 1
fi
ECS_SUDO_PASS=$(grep "^ECS_SUDO_PASS=" .env | cut -d= -f2-)

# ---------------------------------------------------------------------------
log "[2/6] Trigger fresh pg_dump on ECS (systemctl start hackforger-backup.service)"
ssh "$ECS" "echo '$ECS_SUDO_PASS' | sudo -S systemctl start hackforger-backup.service" 2>&1 | tail -2
sleep 2
NEWEST_REMOTE=$(ssh "$ECS" 'ls -1t /var/lib/hackforger/pg-backups/hf-*.pgc | head -1')
echo "  newest dump on ECS: $NEWEST_REMOTE"

# ---------------------------------------------------------------------------
log "[3/6] rsync the new dump to Mac"
mkdir -p "$DEST"
rsync -az "$ECS:$NEWEST_REMOTE" "$DEST/"
LOCAL_DUMP="$DEST/$(basename "$NEWEST_REMOTE")"
ls -lh "$LOCAL_DUMP"

# ---------------------------------------------------------------------------
log "[4/6] Stop Mac gitea (if running on :3000)"
PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$PID" ]; then
  echo "  stopping PID $PID"
  kill "$PID"
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    lsof -iTCP:3000 -sTCP:LISTEN -t >/dev/null 2>&1 || break
    sleep 1
  done
else
  echo "  (no gitea on :3000)"
fi

# ---------------------------------------------------------------------------
log "[5/6] Restore into local 'hackforger' DB (pg_restore --clean --if-exists)"
# --clean drops the existing schema objects before recreating them; --if-exists
# silences the "doesn't exist yet" errors on first run. This avoids needing
# CREATEDB privilege on the `hackforger` PG role, since we never DROP/CREATE
# the database itself — only its contents.
pg_restore --clean --if-exists --no-owner --no-acl \
  -h 127.0.0.1 -U hackforger -d hackforger "$LOCAL_DUMP" 2>&1 | tail -8
echo "  restored — verify a few rows:"
psql -h 127.0.0.1 -U hackforger -d hackforger -c \
  "SELECT 'users' AS tbl, count(*) FROM \"user\" UNION ALL SELECT 'hackathon', count(*) FROM hackathon;"

# ---------------------------------------------------------------------------
log "[6/6] Restart Mac gitea"
bash scripts/restart-gitea.sh 2>&1 | tail -5

# ---------------------------------------------------------------------------
echo
echo "✓ Mac DB now mirrors ECS as of $(basename "$LOCAL_DUMP")"
echo
echo "  Note: this script does NOT sync repos / attachments / lfs files."
echo "  If you need those too, run:"
echo "    rsync -az --delete $ECS:/var/lib/hackforger/data/ ~/Backups/hackforger/data/"
echo "  then manually copy the bits you need into ./data/."
