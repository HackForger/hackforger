#!/bin/bash
# Restart the main HackForger gitea dev instance with the latest build.
#
# Safe to automate: this script only kills processes whose binary path matches
# the main repo at $BINARY. It refuses to kill anything else on port 3000.
#
# Usage:  bash scripts/restart-gitea.sh
# Runs from anywhere — always cds to the main repo.

set -euo pipefail

REPO="/Users/h2oslabs/Workspace/hackforger"
BINARY="$REPO/gitea"

cd "$REPO"

# Pre-flight: detect orphan-binary state (gitea running but $BINARY deleted on disk).
# Unix unlink-while-open keeps the process alive but git push fails with ENOENT in
# pre-receive. The rebuild below will recover; this banner just makes the cause
# visible before make's 30s of output buries it. See docs/notes/orphan-binary.md.
ORPHAN_PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$ORPHAN_PID" ] && [ ! -e "$BINARY" ]; then
  echo "" >&2
  echo "  ⚠ Orphan-binary state detected" >&2
  echo "    PID $ORPHAN_PID is listening on :3000 but $BINARY no longer exists on disk." >&2
  echo "    Any git push to this instance has been failing with ENOENT in pre-receive." >&2
  echo "    This restart will recover. See docs/notes/orphan-binary.md" >&2
  echo "" >&2
fi

echo "[1/4] Building backend (bindata + sqlite tags)..."
TAGS="bindata sqlite sqlite_unlock_notify" make build

# Defensive: make build exited 0 but did it actually produce a runnable binary?
# Without this check, a broken Makefile target would let us kill the running
# instance below and leave us with nothing to start.
if [ ! -x "$BINARY" ]; then
  echo "FATAL: make build returned 0 but $BINARY is missing or non-executable." >&2
  echo "       Refusing to kill the running instance — that would leave you with nothing to start." >&2
  exit 1
fi

echo "[2/4] Checking for process on port 3000..."
PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$PID" ]; then
  # Verify the PID belongs to the main repo's gitea binary before killing.
  RUNNING_BIN=$(lsof -p "$PID" 2>/dev/null | awk '/txt.*REG.*\/gitea$/{print $NF; exit}')
  if [ "$RUNNING_BIN" = "$BINARY" ]; then
    echo "    Stopping main instance (PID $PID, binary $RUNNING_BIN)"
    kill "$PID"
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      lsof -iTCP:3000 -sTCP:LISTEN -t >/dev/null 2>&1 || break
      sleep 1
    done
  else
    echo "    Port 3000 is held by a different binary: $RUNNING_BIN" >&2
    echo "    Expected: $BINARY" >&2
    echo "    Not killing. Inspect with: lsof -p $PID" >&2
    exit 1
  fi
else
  echo "    Nothing on port 3000."
fi

echo "[3/4] Starting new instance..."
rm -f data/queues/common/LOCK
./gitea web > /tmp/gitea-main.log 2>&1 &
disown
NEW_PID=$!
echo "    Started PID $NEW_PID, log at /tmp/gitea-main.log"

echo "[4/4] Waiting for HTTP 200..."
for _ in 1 2 3 4 5 6 7 8 9 10; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/ 2>/dev/null || echo "000")
  if [ "$STATUS" = "200" ]; then
    echo "    ✓ HTTP 200 on localhost:3000"
    exit 0
  fi
  sleep 1
done

echo "    ⚠ Timed out waiting for HTTP 200 — check /tmp/gitea-main.log" >&2
tail -20 /tmp/gitea-main.log >&2 || true
exit 1
