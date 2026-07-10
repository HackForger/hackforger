#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASELINE_REF=${HACKFORGER_BOUNDARY_BASELINE_REF:-refs/remotes/origin/v0.1-dev/hackforger}
[[ "$BASELINE_REF" =~ ^refs/(heads|remotes)/[A-Za-z0-9._/-]+$ \
  || "$BASELINE_REF" =~ ^[0-9a-f]{40}$ \
  || "$BASELINE_REF" =~ ^[0-9a-f]{64}$ ]] \
  || { echo "FATAL: boundary baseline ref has an unsafe form" >&2; exit 2; }
BASELINE_COMMIT=$(git -C "$ROOT" rev-parse --verify "$BASELINE_REF^{commit}" 2>/dev/null) \
  || { echo "FATAL: boundary baseline $BASELINE_REF is unavailable; fetch origin first" >&2; exit 2; }
HEAD_COMMIT=$(git -C "$ROOT" rev-parse --verify 'HEAD^{commit}')
CURRENT_REF=$(git -C "$ROOT" symbolic-ref -q HEAD || printf 'detached/%s\n' "$HEAD_COMMIT")
exec python3 "$ROOT/scripts/ci/check_public_repository_boundary.py" \
  --root "$ROOT" \
  --baseline-ref "$BASELINE_REF" \
  --history-base-ref "$BASELINE_COMMIT" \
  --history-head-ref "$HEAD_COMMIT" \
  --ref-name "$CURRENT_REF" \
  "$@"
