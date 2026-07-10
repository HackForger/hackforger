#!/usr/bin/env bash
set -euo pipefail

# This file is source material for scripts/install-public-boundary-hook.sh. The
# effective hook executes only from an immutable, commit-versioned bundle under
# the Git common directory; it never calls back into a mutable checkout.

HOOK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
CHECKER="$HOOK_DIR/check_public_repository_boundary.py"
POLICY="$HOOK_DIR/private-content-markers.txt"
GUARD_POLICY="$HOOK_DIR/boundary_guard_policy.py"
SOURCE_COMMIT_FILE="$HOOK_DIR/source-commit"
REMOTE_NAME=${1:-}
TRUSTED_REMOTE=origin
TMP_REF=

fatal() {
  echo "FATAL: $*" >&2
  exit 2
}

# Git itself exports repository-local variables to hooks. Drop them and any
# caller-provided object/index/config overlays, then rediscover the worktree
# from the cwd that Git sets for a non-bare pre-push hook. SSH transport and
# credential-agent variables remain user-controlled; a user controlling the
# push process can already use --no-verify and is outside this hook's boundary.
unset \
  GIT_DIR \
  GIT_WORK_TREE \
  GIT_IMPLICIT_WORK_TREE \
  GIT_COMMON_DIR \
  GIT_OBJECT_DIRECTORY \
  GIT_ALTERNATE_OBJECT_DIRECTORIES \
  GIT_INDEX_FILE \
  GIT_INDEX_VERSION \
  GIT_NAMESPACE \
  GIT_NO_REPLACE_OBJECTS \
  GIT_REPLACE_REF_BASE \
  GIT_CONFIG \
  GIT_CONFIG_PARAMETERS \
  GIT_CONFIG_COUNT \
  GIT_CONFIG_SYSTEM \
  GIT_CONFIG_GLOBAL \
  GIT_CONFIG_NOSYSTEM \
  GIT_EXEC_PATH \
  GIT_CEILING_DIRECTORIES \
  GIT_DISCOVERY_ACROSS_FILESYSTEM \
  GIT_PREFIX \
  GIT_INTERNAL_SUPER_PREFIX \
  GIT_GRAFT_FILE \
  GIT_SHALLOW_FILE

for artifact in "$CHECKER" "$POLICY" "$GUARD_POLICY" "$SOURCE_COMMIT_FILE"; do
  [ -f "$artifact" ] && [ ! -L "$artifact" ] \
    || fatal "immutable public-boundary hook bundle is incomplete"
done
[ -x "$CHECKER" ] || fatal "immutable boundary checker is not executable"

ROOT=$(git rev-parse --path-format=absolute --show-toplevel 2>/dev/null) \
  || fatal "pre-push hook must run inside a non-bare worktree"
ROOT=$(cd "$ROOT" && pwd -P)
# Do not allow local replace refs to rewrite the commit graph or trusted trees
# inspected by this hook or its checker.
export GIT_NO_REPLACE_OBJECTS=1

reject_graph_overrides() {
  local grafts shallow
  shallow=$(git -C "$ROOT" rev-parse --is-shallow-repository 2>/dev/null) \
    || fatal "cannot determine whether the repository is shallow"
  [ "$shallow" = false ] \
    || fatal "shallow repositories are not allowed while enforcing the public boundary; run git fetch --unshallow origin"
  grafts=$(git -C "$ROOT" rev-parse --path-format=absolute --git-path info/grafts 2>/dev/null) \
    || fatal "cannot resolve the legacy Git grafts path"
  if [ -e "$grafts" ] || [ -L "$grafts" ]; then
    [ -f "$grafts" ] && [ ! -L "$grafts" ] && [ ! -s "$grafts" ] \
      || fatal "legacy Git grafts are not allowed while enforcing the public boundary"
  fi
}

[[ "$REMOTE_NAME" =~ ^[A-Za-z0-9._-]+$ ]] \
  || fatal "pre-push remote name has an unsafe form"

SOURCE_COMMIT=$(cat "$SOURCE_COMMIT_FILE")
[[ "$SOURCE_COMMIT" =~ ^[0-9a-f]{40}$ || "$SOURCE_COMMIT" =~ ^[0-9a-f]{64}$ ]] \
  || fatal "immutable hook bundle has an invalid source commit"
[ "${HOOK_DIR##*/}" = "$SOURCE_COMMIT" ] \
  || fatal "immutable hook bundle path does not match its source commit"

COMMITS=$(mktemp "${TMPDIR:-/tmp}/hackforger-boundary-commits.XXXXXX")
cleanup() {
  rm -f "$COMMITS"
  if [ -n "${TMP_REF:-}" ]; then
    git -C "$ROOT" update-ref -d "$TMP_REF" 2>/dev/null || true
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP
REF_ARGS=()
HAS_UPDATE=0
GRAPH_CHECKED=0

is_zero_oid() {
  [[ "$1" =~ ^0{40}$ || "$1" =~ ^0{64}$ ]]
}

while read -r local_ref local_oid remote_ref remote_oid; do
  [ -n "${local_ref:-}" ] || continue
  # A deletion removes public metadata. Do not scan or reject the name/type of
  # the ref being deleted, and do not require a baseline for deletion-only pushes.
  is_zero_oid "${local_oid:-}" && continue
  HAS_UPDATE=1
  if [ "$GRAPH_CHECKED" = 0 ]; then
    reject_graph_overrides
    GRAPH_CHECKED=1
  fi
  REF_ARGS+=(--ref-name "$local_ref" --ref-name "$remote_ref")
  [[ "$local_oid" =~ ^[0-9a-f]{40}$ || "$local_oid" =~ ^[0-9a-f]{64}$ ]] \
    || fatal "pre-push local object id is invalid"
  [ "$(git -C "$ROOT" cat-file -t "$local_oid" 2>/dev/null)" = commit ] \
    || fatal "annotated tags or non-commit refs require dedicated security review"
  local_commit=$(git -C "$ROOT" rev-parse --verify "$local_oid^{commit}" 2>/dev/null) \
    || fatal "only commit-backed refs may be pushed"
  git -C "$ROOT" merge-base --is-ancestor "$SOURCE_COMMIT" "$local_commit" 2>/dev/null \
    || fatal "pushed refs must contain the installed boundary source commit; rebase or merge the trusted default"
  if is_zero_oid "${remote_oid:-}"; then
    git -C "$ROOT" rev-list --reverse "$SOURCE_COMMIT..$local_commit" >> "$COMMITS"
  else
    [[ "$remote_oid" =~ ^[0-9a-f]{40}$ || "$remote_oid" =~ ^[0-9a-f]{64}$ ]] \
      || fatal "pre-push remote object id is invalid"
    remote_commit=$(git -C "$ROOT" rev-parse --verify "$remote_oid^{commit}" 2>/dev/null) \
      || fatal "remote ref does not resolve to a local commit"
    git -C "$ROOT" rev-list --reverse "$remote_commit..$local_commit" >> "$COMMITS"
  fi
done

if [ "$HAS_UPDATE" = 0 ]; then
  echo "public repository boundary: PASS (ref deletions only)"
  exit 0
fi

ORIGIN_URL=$(git -C "$ROOT" config --local --get-all remote.origin.url 2>/dev/null || true)
case "$ORIGIN_URL" in
  git@github.com:HackForger/hackforger.git|https://github.com/HackForger/hackforger.git)
    ;;
  *)
    fatal "origin must be the canonical HackForger/hackforger SSH or HTTPS URL"
    ;;
esac

verify_guard_tree() {
  local commit=$1 source_path=$2 expected_mode=$3 installed=$4
  local entry metadata tracked_path mode type oid
  entry=$(git -C "$ROOT" ls-tree "$commit" -- ":(literal)$source_path")
  [ -n "$entry" ] || fatal "trusted default is missing boundary guard $source_path; reinstall is required"
  metadata=${entry%%$'\t'*}
  tracked_path=${entry#*$'\t'}
  [ "$tracked_path" = "$source_path" ] \
    || fatal "trusted default returned an unexpected boundary guard path"
  read -r mode type oid <<< "$metadata"
  [ "$mode" = "$expected_mode" ] && [ "$type" = blob ] \
    && [[ "$oid" =~ ^[0-9a-f]{40}$ || "$oid" =~ ^[0-9a-f]{64}$ ]] \
    || fatal "trusted default has an unsafe boundary guard entry; reinstall is required"
  git -C "$ROOT" cat-file blob "$oid" | cmp -s - "$installed" \
    || fatal "installed boundary guard is stale or modified; rerun scripts/install-public-boundary-hook.sh"
}

verify_guard_bundle_at() {
  local commit=$1
  verify_guard_tree "$commit" scripts/pre-push-public-boundary.sh 100755 "$HOOK_DIR/pre-push"
  verify_guard_tree "$commit" scripts/ci/check_public_repository_boundary.py 100755 "$CHECKER"
  verify_guard_tree "$commit" scripts/ci/boundary_guard_policy.py 100644 "$GUARD_POLICY"
  verify_guard_tree "$commit" scripts/ci/private-content-markers.txt 100644 "$POLICY"
}

# First prove that the non-checkout bundle still matches its pinned source.
verify_guard_bundle_at "$SOURCE_COMMIT"
REMOTE_HEAD=$(git -C "$ROOT" ls-remote --symref "$TRUSTED_REMOTE" HEAD) \
  || fatal "cannot verify the trusted origin default guard; push is blocked"
DEFAULT_REF=$(printf '%s\n' "$REMOTE_HEAD" | awk '$1 == "ref:" && $3 == "HEAD" { print $2 }')
LIVE_OID=$(printf '%s\n' "$REMOTE_HEAD" | awk '$2 == "HEAD" && $1 ~ /^[0-9a-f]+$/ { print $1 }')
[[ "$DEFAULT_REF" =~ ^refs/heads/[A-Za-z0-9._/-]+$ ]] \
  || fatal "origin HEAD did not advertise exactly one safe default branch"
git -C "$ROOT" check-ref-format "$DEFAULT_REF" >/dev/null 2>&1 \
  || fatal "origin HEAD advertised an invalid default branch"
[[ "$LIVE_OID" =~ ^[0-9a-f]{40}$ || "$LIVE_OID" =~ ^[0-9a-f]{64}$ ]] \
  || fatal "origin HEAD did not advertise exactly one valid commit object id"
if [ "$LIVE_OID" != "$SOURCE_COMMIT" ]; then
  TMP_REF="refs/hackforger-boundary/pre-push/$$-$RANDOM"
  git -C "$ROOT" fetch --quiet --no-tags "$TRUSTED_REMOTE" "+$DEFAULT_REF:$TMP_REF" \
    || fatal "cannot fetch the trusted origin default guard; push is blocked"
  FETCHED_OID=$(git -C "$ROOT" rev-parse --verify "$TMP_REF^{commit}" 2>/dev/null || true)
  [ "$FETCHED_OID" = "$LIVE_OID" ] \
    || fatal "origin default changed while validating the boundary guard; retry"
  verify_guard_bundle_at "$LIVE_OID"
fi

BASELINE_COMMIT=$(git -C "$ROOT" rev-parse --verify "$SOURCE_COMMIT^{commit}" 2>/dev/null) \
  || fatal "installed boundary source commit is unavailable; reinstall the hook"
# Recheck immediately before the checker starts; it also performs Git graph
# operations while scanning the immutable commit list.
reject_graph_overrides
LC_ALL=C sort -u "$COMMITS" -o "$COMMITS"
python3 "$CHECKER" \
  --root "$ROOT" \
  --policy "$POLICY" \
  --baseline-ref "$BASELINE_COMMIT" \
  --history-commit-list "$COMMITS" \
  --history-only \
  "${REF_ARGS[@]}"
