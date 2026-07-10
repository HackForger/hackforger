#!/bin/bash
# Restart the HackForger test gitea instance with the latest build.
#
# Mirror of restart-gitea.sh but targets the test instance:
#   port:        3001
#   custom path: /tmp/hackforger-test-custom
#   work path:   /tmp/hackforger-test-data
#   database:    hackforger_test (PG)
#
# Safe to automate: only kills processes whose binary path matches $BINARY.
# Refuses to touch port 3001 if it's held by anything else.
#
# Usage:  bash scripts/restart-gitea-test.sh

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="$REPO/gitea"
CUSTOM_PATH="/tmp/hackforger-test-custom"
WORK_PATH="/tmp/hackforger-test-data"
PORT=3001
LOG="/tmp/gitea-test.log"

cd "$REPO"

# Pre-flight: detect orphan-binary state. See docs/notes/orphan-binary.md.
ORPHAN_PID=$(lsof -iTCP:$PORT -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$ORPHAN_PID" ] && [ ! -e "$BINARY" ]; then
  echo "" >&2
  echo "  ⚠ Orphan-binary state detected" >&2
  echo "    PID $ORPHAN_PID is listening on :$PORT but $BINARY no longer exists on disk." >&2
  echo "    Any git push to this instance has been failing with ENOENT in pre-receive." >&2
  echo "    This restart will recover. See docs/notes/orphan-binary.md" >&2
  echo "" >&2
fi

echo "[1/4] Building backend (bindata + sqlite tags)..."
TAGS="bindata sqlite sqlite_unlock_notify" make build

# Defensive: make build exited 0 but did it actually produce a runnable binary?
if [ ! -x "$BINARY" ]; then
  echo "FATAL: make build returned 0 but $BINARY is missing or non-executable." >&2
  echo "       Refusing to kill the running instance — that would leave you with nothing to start." >&2
  exit 1
fi

echo "[2/4] Checking for process on port $PORT..."
PID=$(lsof -iTCP:$PORT -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$PID" ]; then
  RUNNING_BIN=$(lsof -p "$PID" 2>/dev/null | awk '/txt.*REG.*\/gitea$/{print $NF; exit}')
  if [ "$RUNNING_BIN" = "$BINARY" ]; then
    echo "    Stopping test instance (PID $PID, binary $RUNNING_BIN)"
    kill "$PID"
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      lsof -iTCP:$PORT -sTCP:LISTEN -t >/dev/null 2>&1 || break
      sleep 1
    done
  else
    echo "    Port $PORT is held by a different binary: $RUNNING_BIN" >&2
    echo "    Expected: $BINARY" >&2
    echo "    Not killing. Inspect with: lsof -p $PID" >&2
    exit 1
  fi
else
  echo "    Nothing on port $PORT."
fi

echo "[3/4] Starting new instance..."
rm -f "$WORK_PATH/data/queues/common/LOCK"
./gitea web --custom-path "$CUSTOM_PATH" > "$LOG" 2>&1 &
disown
NEW_PID=$!
echo "    Started PID $NEW_PID, log at $LOG"

echo "[4/4] Waiting for HTTP 200..."
for _ in 1 2 3 4 5 6 7 8 9 10; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT/ 2>/dev/null || echo "000")
  if [ "$STATUS" = "200" ]; then
    echo "    ✓ HTTP 200 on localhost:$PORT"
    exit 0
  fi
  sleep 1
done

echo "    ⚠ Timed out waiting for HTTP 200 — check $LOG" >&2
tail -20 "$LOG" >&2 || true
exit 1
