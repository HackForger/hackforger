#!/bin/bash
# Atomic redeploy of HackForger to the production ECS.
#
# Build (or reuse existing) gitea-linux-amd64, sync custom/templates/ and
# custom/public/, install the binary, restart gitea, record .last-deploy.
#
# This is the script you run AFTER the initial migrate-data.sh — for every
# subsequent change. Use it instead of ad-hoc scp + restart, because:
#
# - It rsyncs custom/templates/ and custom/public/ alongside the binary.
#   Forgetting these is a real bug we hit: PRs that only touch templates
#   or landing assets won't take effect from a binary-only deploy.
#   (Locale .ini files ARE in the binary via bindata, so they don't need
#   rsync — but template files do, since custom/ overrides bindata at
#   runtime.)
#
# - It updates /var/lib/hackforger/.last-deploy so deploy/ecs/preflight.sh
#   knows what was shipped.
#
# - It uses a single sudo invocation per remote step, so it works in
#   non-TTY ssh sessions (just like ecs-bootstrap.sh).
#
# Usage:
#   bash deploy/ecs/redeploy.sh [--skip-build] [--skip-preflight]
#
# Inputs:
#   .env on Mac:  ECS_SUDO_PASS=...   (the box's hackforger sudo password)
#   git HEAD:     used to record .last-deploy on the box

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO"

ECS=hackforger@203.119.115.130
SKIP_BUILD=0
SKIP_PREFLIGHT=0
for arg in "$@"; do
  case "$arg" in
    --skip-build) SKIP_BUILD=1 ;;
    --skip-preflight) SKIP_PREFLIGHT=1 ;;
    *) echo "Usage: $0 [--skip-build] [--skip-preflight]" >&2; exit 64 ;;
  esac
done

log() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }
ok()  { printf "  \033[32m✓\033[0m %s\n" "$*"; }

# ---------------------------------------------------------------------------
log "[0/7] Reading sudo password from .env"
if ! grep -q "^ECS_SUDO_PASS=" .env; then
  echo "FATAL: ECS_SUDO_PASS not in .env" >&2
  exit 1
fi
SUDO_PASS=$(grep "^ECS_SUDO_PASS=" .env | cut -d= -f2-)

# ---------------------------------------------------------------------------
if [ "$SKIP_PREFLIGHT" = "0" ]; then
  log "[1/7] Preflight (use --skip-preflight to bypass for hotfixes)"
  if [ -x deploy/ecs/preflight.sh ]; then
    bash deploy/ecs/preflight.sh
  else
    echo "  WARN: deploy/ecs/preflight.sh not present — gate not yet merged"
    echo "        skipping (this is OK for early deploys; merge the gitflow PR to enforce)"
  fi
else
  log "[1/7] Preflight SKIPPED (--skip-preflight)"
fi

# ---------------------------------------------------------------------------
log "[2/7] Build linux binary"
if [ "$SKIP_BUILD" = "1" ] && [ -f gitea-linux-amd64 ]; then
  ok "reusing existing gitea-linux-amd64 ($(ls -lh gitea-linux-amd64 | awk '{print $5}'))"
else
  bash deploy/ecs/build-linux.sh
fi

# ---------------------------------------------------------------------------
log "[3/7] Sync custom/templates + custom/public to ECS"
# These override bindata-embedded templates at runtime. A binary-only deploy
# is a bug — locale .ini files are in the binary, but template/asset
# overrides are in the filesystem.
#
# DELIBERATELY excluded:
#   custom/conf/app.ini   (per-instance config; renders on bootstrap, never sync)
#   custom/options/       (locale overrides; if used, add separately)
rsync -az --delete custom/templates/ "$ECS:/tmp/custom-templates-staged/"
rsync -az --delete custom/public/    "$ECS:/tmp/custom-public-staged/"
ok "staged at /tmp/custom-{templates,public}-staged/"

# ---------------------------------------------------------------------------
log "[4/7] Stage binary on ECS"
scp -q gitea-linux-amd64 "$ECS:/tmp/gitea-new"
ok "staged at /tmp/gitea-new"

# ---------------------------------------------------------------------------
log "[5/7] Atomic swap + restart (single ssh, single sudo)"
SHA=$(git rev-parse HEAD)
ssh "$ECS" "SUDO_PASS='$SUDO_PASS' SHA='$SHA' bash -s" <<'REMOTE_EOF'
set -euo pipefail
echo "$SUDO_PASS" | sudo -S -v
( while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null ) &
KEEPER=$!
trap 'kill $KEEPER 2>/dev/null || true' EXIT

# Templates + public assets
sudo rsync -a --delete /tmp/custom-templates-staged/ /var/lib/hackforger/custom/templates/
sudo rsync -a --delete /tmp/custom-public-staged/    /var/lib/hackforger/custom/public/
sudo chown -R hackforger:hackforger /var/lib/hackforger/custom/templates /var/lib/hackforger/custom/public
sudo rm -rf /tmp/custom-templates-staged /tmp/custom-public-staged

# Binary swap (install is atomic — old binary keeps serving until restart)
sudo install -m 755 -o hackforger -g hackforger /tmp/gitea-new /opt/hackforger/gitea
sudo rm -f /tmp/gitea-new

# Restart
sudo systemctl restart gitea

# Record what we shipped
echo "$SHA" | sudo tee /var/lib/hackforger/.last-deploy >/dev/null

echo "  ✓ binary + custom/ swapped, gitea restarting"
REMOTE_EOF

# ---------------------------------------------------------------------------
log "[6/7] Wait for gitea to come back"
for _ in $(seq 1 30); do
  if ssh "$ECS" 'curl -fsS http://127.0.0.1:3000/api/v1/version' 2>/dev/null > /dev/null; then
    ok "gitea responding"
    break
  fi
  sleep 2
done

# ---------------------------------------------------------------------------
log "[7/7] Public smoke tests"
echo -n "  /api/v1/version → "
curl -fsS https://www.synnovator.com/api/v1/version
echo

echo -n "  navbar zh-CN '登录/注册' present → "
if curl -fsS -H "Accept-Language: zh-CN,zh;q=0.9" "https://www.synnovator.com/explore/repos" \
   | grep -qE '登录\s?/\s?注册'; then
  ok "yes"
else
  echo "NOT FOUND (template may not have synced)"
fi

echo -n "  HTTPS landing returns 200 → "
status=$(curl -sI https://www.synnovator.com/ 2>&1 | grep -oE 'HTTP/[0-9.]+ [0-9]+' | tail -1)
echo "$status"

echo
ok "Deploy complete: SHA $SHA"
echo "  cloud .last-deploy:"
ssh "$ECS" 'cat /var/lib/hackforger/.last-deploy'
