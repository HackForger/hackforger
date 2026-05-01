#!/bin/bash
# Mac-side: dump the post-clean-slate prod DB and data dirs, rsync to the ECS,
# trigger restore via SSH.
#
# Idempotent for the dump+rsync side. The restore side OVERWRITES the ECS DB
# and data dirs, so guard with --confirm.
#
# Usage:
#   bash deploy/ecs/migrate-data.sh --dry-run     # dump + rsync only, no restore
#   bash deploy/ecs/migrate-data.sh --confirm     # full migration

set -euo pipefail

DRY_RUN=0
CONFIRM=0
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    --confirm) CONFIRM=1 ;;
    *) echo "Usage: $0 (--dry-run|--confirm)" >&2; exit 64 ;;
  esac
done
if [ "$DRY_RUN$CONFIRM" = "00" ]; then
  echo "FATAL: pass --dry-run or --confirm" >&2
  exit 64
fi

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO"

ECS=hackforger@218.91.114.178
TMP_DUMP=/tmp/hackforger-prod.pgc
TMP_TAR=/tmp/hackforger-data.tgz

log() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }

# ---------------------------------------------------------------------------
log "[1/6] Reading PG password from .env"
if ! grep -q "^PGPASSWORD=" .env; then
  echo "FATAL: PGPASSWORD not in .env" >&2
  exit 1
fi
PG_PASS=$(grep "^PGPASSWORD=" .env | cut -d= -f2)
export PGPASSWORD="$PG_PASS"

# ---------------------------------------------------------------------------
log "[2/6] Stop Mac gitea (if running) — only with --confirm"
if [ "$CONFIRM" = "1" ]; then
  PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t 2>/dev/null || true)
  if [ -n "$PID" ]; then
    echo "  Stopping gitea PID $PID"
    kill "$PID"
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      lsof -iTCP:3000 -sTCP:LISTEN -t >/dev/null 2>&1 || break
      sleep 1
    done
  else
    echo "  gitea not running on :3000"
  fi
else
  echo "  --dry-run: leaving Mac gitea running"
fi

# ---------------------------------------------------------------------------
log "[3/6] pg_dump → $TMP_DUMP"
pg_dump -Fc -h 127.0.0.1 -U hackforger hackforger > "$TMP_DUMP"
ls -lh "$TMP_DUMP"

# ---------------------------------------------------------------------------
log "[4/6] tar data dirs + custom assets → $TMP_TAR"
TAR_ITEMS=(data/forgejo-repositories data/attachments data/avatars
           custom/conf/app.ini custom/public custom/templates)
[ -d data/lfs ] && TAR_ITEMS+=(data/lfs)
tar czf "$TMP_TAR" -C "$REPO" "${TAR_ITEMS[@]}" 2>/dev/null
ls -lh "$TMP_TAR"

# ---------------------------------------------------------------------------
log "[5/6] rsync to ECS"
rsync -avz --progress "$TMP_DUMP" "$TMP_TAR" "$ECS:/tmp/"

# ---------------------------------------------------------------------------
log "[6/6] Trigger restore on ECS"
if [ "$CONFIRM" = "1" ]; then
  # Carry over the Mac's INTERNAL_TOKEN and JWT_SECRET verbatim — they MUST
  # NOT change across migration (would invalidate sessions + break runner).
  INTERNAL_TOKEN=$(grep "^INTERNAL_TOKEN" custom/conf/app.ini | cut -d= -f2- | xargs)
  OAUTH_JWT=$(grep "^JWT_SECRET" custom/conf/app.ini | cut -d= -f2- | xargs)

  if [ -z "$INTERNAL_TOKEN" ] || [ -z "$OAUTH_JWT" ]; then
    echo "FATAL: failed to read INTERNAL_TOKEN or JWT_SECRET from custom/conf/app.ini" >&2
    exit 2
  fi

  ssh "$ECS" \
    INTERNAL_TOKEN="'$INTERNAL_TOKEN'" \
    OAUTH_JWT="'$OAUTH_JWT'" \
    bash -s <<'REMOTE_EOF'
set -euo pipefail

log() { printf "\n\033[1;34m  ▶ %s\033[0m\n" "$*"; }

log "Stopping gitea + runner (if running)"
sudo systemctl stop gitea forgejo-runner 2>/dev/null || true

log "Dropping & recreating PG database hackforger"
sudo -u postgres dropdb --if-exists hackforger
sudo -u postgres createdb -O hackforger hackforger

log "pg_restore"
PGPASSWORD=$(sudo cat /root/.hackforger-pg-password) \
  pg_restore -h 127.0.0.1 -U hackforger -d hackforger /tmp/hackforger-prod.pgc

log "Extracting data tar to /var/lib/hackforger"
sudo -u hackforger mkdir -p /var/lib/hackforger
sudo tar xzf /tmp/hackforger-data.tgz -C /var/lib/hackforger
sudo chown -R hackforger:hackforger /var/lib/hackforger

log "Rendering app.ini from template"
PG_PASS=$(sudo cat /root/.hackforger-pg-password)
# Use a tmp file to avoid sed escaping pain with passwords
TMP_INI=$(mktemp)
sed \
  -e "s|__PG_PASSWORD__|$PG_PASS|g" \
  -e "s|__INTERNAL_TOKEN__|$INTERNAL_TOKEN|g" \
  -e "s|__OAUTH2_JWT_SECRET__|$OAUTH_JWT|g" \
  /opt/hackforger/app.ini.tmpl > "$TMP_INI"
sudo install -o hackforger -g hackforger -m 640 "$TMP_INI" \
  /var/lib/hackforger/custom/conf/app.ini
rm -f "$TMP_INI"

log "Smoke-test: gitea --version"
sudo -u hackforger /opt/hackforger/gitea --version || {
  echo "gitea binary missing or not executable" >&2
  exit 2
}
echo "  ✓ Restore complete. Next: sudo systemctl start gitea caddy"
REMOTE_EOF
else
  echo "  --dry-run: skipping restore step"
  echo "  Files staged on ECS at: /tmp/hackforger-prod.pgc, /tmp/hackforger-data.tgz"
fi

log "Done"
