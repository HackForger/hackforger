#!/bin/bash
# Pre-deploy gate: refuse to ship commits that don't have a smoke-test report.
#
# Usage:    bash deploy/ecs/preflight.sh
# Inputs:   reads .last-deploy SHA from the ECS via SSH
# Outputs:  exit 0 + "ready to deploy: <SHA>" on success
#           exit 1 + per-commit failure listing on failure
#
# See: docs/notes/gitflow.md for the full process.

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO"

ECS=hackforger@203.119.115.130
REPORTS_DIR=docs/tests/e2e/reports
PROD_BRANCH="${PROD_BRANCH:-prod}"

# Paths that DON'T need a smoke-test report (infra/docs/tooling only)
INFRA_PATHS_RE='^(deploy/|scripts/|docs/|\.github/|\.forgejo/|\.gitignore$|\.editorconfig$|Makefile$|CHANGELOG\.md$|[^/]+\.md$|go\.mod$|go\.sum$|package-lock\.json$|package\.json$)'

log() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }
fail() { printf "  \033[31m✗\033[0m %s\n" "$*"; }
ok() { printf "  \033[32m✓\033[0m %s\n" "$*"; }

# ---------------------------------------------------------------------------
log "Checking prod branch"
if ! git rev-parse --verify "$PROD_BRANCH" >/dev/null 2>&1; then
  echo "FATAL: branch '$PROD_BRANCH' does not exist locally." >&2
  echo "  First time? Create it from the desired baseline:" >&2
  echo "    git branch $PROD_BRANCH origin/v0.1-dev/hackforger" >&2
  exit 2
fi
PROD_HEAD=$(git rev-parse "$PROD_BRANCH")
echo "  $PROD_BRANCH HEAD: $PROD_HEAD"

# ---------------------------------------------------------------------------
log "Reading last-deploy marker from $ECS"
LAST_DEPLOY=$(ssh -o ConnectTimeout=10 "$ECS" \
  'cat /var/lib/hackforger/.last-deploy 2>/dev/null || echo MISSING' | tr -d '[:space:]')
if [ "$LAST_DEPLOY" = "MISSING" ]; then
  echo "  no .last-deploy file on ECS — treating as bootstrap (must verify ALL commits on $PROD_BRANCH)"
  RANGE="$PROD_BRANCH"
else
  echo "  last deployed: $LAST_DEPLOY"
  RANGE="${LAST_DEPLOY}..${PROD_BRANCH}"
fi

# ---------------------------------------------------------------------------
log "Commits to verify"
COMMITS=$(git rev-list --reverse "$RANGE" 2>/dev/null || true)
if [ -z "$COMMITS" ]; then
  ok "no new commits to deploy — already up to date"
  exit 0
fi
COUNT=$(echo "$COMMITS" | wc -l | tr -d ' ')
echo "  $COUNT commit(s) in $RANGE"

# ---------------------------------------------------------------------------
log "Per-commit smoke-test check"
FAIL=0
PASS=0
SKIP=0

for sha in $COMMITS; do
  short=$(git rev-parse --short "$sha")
  subject=$(git log -1 --format='%s' "$sha")
  files=$(git show --name-only --format= "$sha")

  # Is this commit infra-only?
  user_facing=0
  for f in $files; do
    [ -z "$f" ] && continue
    if ! echo "$f" | grep -qE "$INFRA_PATHS_RE"; then
      user_facing=1
      break
    fi
  done

  if [ "$user_facing" = "0" ]; then
    printf "  \033[33m·\033[0m %s %s — INFRA-ONLY (skip)\n" "$short" "${subject:0:60}"
    SKIP=$((SKIP+1))
    continue
  fi

  # Look for a report referencing this SHA in frontmatter
  report=$(grep -lE "^\s*-\s+${sha}\b|^\s*-\s+${short}\b" "$REPORTS_DIR"/*.md 2>/dev/null | head -1)
  if [ -z "$report" ]; then
    fail "$short $subject"
    echo "        no report references $sha (or $short) in $REPORTS_DIR/"
    FAIL=$((FAIL+1))
    continue
  fi

  # Check admin_signoff: non-null AND not empty.
  # YAML allows two forms:
  #   admin_signoff: null              ← unsigned, FAIL
  #   admin_signoff:                   ← unsigned (empty value), FAIL
  #   admin_signoff:\n  by: alice...   ← signed (mapping value), PASS
  if grep -qE '^admin_signoff:\s*(null|~)?\s*$' "$report"; then
    fail "$short $subject"
    echo "        report exists ($report) but admin_signoff is null"
    FAIL=$((FAIL+1))
    continue
  fi

  ok "$short $subject  ($(basename "$report"))"
  PASS=$((PASS+1))
done

# ---------------------------------------------------------------------------
echo
echo "summary: $PASS pass, $FAIL fail, $SKIP infra-only-skip"
if [ "$FAIL" -gt 0 ]; then
  echo
  echo "DEPLOY BLOCKED. Each failing commit needs a report at:"
  echo "  docs/tests/e2e/reports/YYYY-MM-DD-<slug>.md"
  echo "with frontmatter:"
  echo "  commits: [<sha>]"
  echo "  admin_signoff: { by: <name>, at: <ISO-time> }"
  echo
  echo "See docs/notes/gitflow.md for the full process."
  exit 1
fi

echo
echo "READY TO DEPLOY: $PROD_HEAD"
echo
echo "Next:"
echo "  bash deploy/ecs/build-linux.sh"
echo "  scp gitea-linux-amd64 $ECS:/opt/hackforger/gitea-new"
echo "  ssh $ECS 'sudo install -m 755 -o hackforger -g hackforger /opt/hackforger/gitea-new /opt/hackforger/gitea && sudo systemctl restart gitea && echo $PROD_HEAD | sudo tee /var/lib/hackforger/.last-deploy'"
