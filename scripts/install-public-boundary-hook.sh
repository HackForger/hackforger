#!/usr/bin/env bash
set -euo pipefail

# Install the public-boundary hook from the trusted origin default commit, never
# from the mutable checkout that happens to invoke this installer.

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)

# Ignore caller-provided repository/object/config views. SSH transport and
# credential-agent variables remain user-controlled; a user who controls the
# push process can already choose --no-verify and is outside this hook's threat
# boundary.
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
ROOT=$(git -C "$ROOT" rev-parse --path-format=absolute --show-toplevel)
COMMON_DIR=$(git -C "$ROOT" rev-parse --path-format=absolute --git-common-dir)
COMMON_DIR=$(cd "$COMMON_DIR" && pwd -P)
CANONICAL_HOOKS="$COMMON_DIR/hooks"
MANAGED_ROOT="$COMMON_DIR/hackforger-boundary-hooks"
REMOTE_NAME=origin

fatal() {
  echo "FATAL: $*" >&2
  exit 1
}

is_oid() {
  [[ "$1" =~ ^[0-9a-f]{40}$ || "$1" =~ ^[0-9a-f]{64}$ ]]
}

canonical_hooks_are_samples_only() {
  local entry name
  [ -e "$CANONICAL_HOOKS" ] || return 0
  [ -d "$CANONICAL_HOOKS" ] && [ ! -L "$CANONICAL_HOOKS" ] || return 1
  while IFS= read -r -d '' entry; do
    name=${entry##*/}
    [ -f "$entry" ] && [ ! -L "$entry" ] && [[ "$name" == *.sample ]] || return 1
  done < <(find "$CANONICAL_HOOKS" -mindepth 1 -maxdepth 1 -print0)
}

managed_hook_path() {
  local candidate=$1 suffix
  [[ "$candidate" == "$MANAGED_ROOT/"* ]] || return 1
  suffix=${candidate#"$MANAGED_ROOT/"}
  is_oid "$suffix" || return 1
  [ "$candidate" = "$MANAGED_ROOT/$suffix" ]
}

verify_bundle() {
  local directory=$1 expected=$2 count
  [ -d "$directory" ] && [ ! -L "$directory" ] || return 1
  count=$(find "$directory" -mindepth 1 -maxdepth 1 -print | wc -l | tr -d '[:space:]')
  [ "$count" = 5 ] || return 1
  for name in \
    pre-push \
    check_public_repository_boundary.py \
    boundary_guard_policy.py \
    private-content-markers.txt \
    source-commit
  do
    [ -f "$directory/$name" ] && [ ! -L "$directory/$name" ] || return 1
    cmp -s "$directory/$name" "$expected/$name" || return 1
  done
  [ -x "$directory/pre-push" ] && [ -x "$directory/check_public_repository_boundary.py" ]
}

restore_local_config() {
  if [ "$PREVIOUS_LOCAL_SET" = 1 ]; then
    git -C "$ROOT" config --local core.hooksPath "$PREVIOUS_LOCAL"
  else
    git -C "$ROOT" config --local --unset-all core.hooksPath 2>/dev/null || true
  fi
}

command -v git >/dev/null 2>&1 || fatal "git is required"
command -v python3 >/dev/null 2>&1 || fatal "python3 is required"
ORIGIN_URL=$(git -C "$ROOT" config --local --get-all remote.origin.url 2>/dev/null || true)
case "$ORIGIN_URL" in
  git@github.com:HackForger/hackforger.git|https://github.com/HackForger/hackforger.git)
    ;;
  *)
    fatal "origin must be the canonical HackForger/hackforger SSH or HTTPS URL"
    ;;
esac

# Disable local replace refs while identifying and extracting the trusted tree.
export GIT_NO_REPLACE_OBJECTS=1
REMOTE_HEAD=$(git -C "$ROOT" ls-remote --symref "$REMOTE_NAME" HEAD) \
  || fatal "cannot read the trusted origin default ref"
DEFAULT_REF=$(printf '%s\n' "$REMOTE_HEAD" | awk '$1 == "ref:" && $3 == "HEAD" { print $2 }')
DEFAULT_OID=$(printf '%s\n' "$REMOTE_HEAD" | awk '$2 == "HEAD" && $1 ~ /^[0-9a-f]+$/ { print $1 }')
[[ "$DEFAULT_REF" =~ ^refs/heads/[A-Za-z0-9._/-]+$ ]] \
  || fatal "origin HEAD did not advertise exactly one safe default branch"
git -C "$ROOT" check-ref-format "$DEFAULT_REF" >/dev/null 2>&1 \
  || fatal "origin HEAD advertised an invalid default branch"
is_oid "$DEFAULT_OID" \
  || fatal "origin HEAD did not advertise exactly one valid commit object id"

TMP_REF="refs/hackforger-boundary/install/$$-$RANDOM"
STAGING=
cleanup() {
  git -C "$ROOT" update-ref -d "$TMP_REF" 2>/dev/null || true
  if [ -n "${STAGING:-}" ] && [ -d "$STAGING" ]; then
    chmod -R u+w "$STAGING" 2>/dev/null || true
    rm -rf "$STAGING"
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP

git -C "$ROOT" fetch --quiet --no-tags "$REMOTE_NAME" "+$DEFAULT_REF:$TMP_REF" \
  || fatal "cannot fetch the trusted origin default ref"
FETCHED_OID=$(git -C "$ROOT" rev-parse --verify "$TMP_REF^{commit}" 2>/dev/null || true)
[ "$FETCHED_OID" = "$DEFAULT_OID" ] \
  || fatal "origin default ref changed during installation; rerun the installer"

umask 077
mkdir -p "$MANAGED_ROOT"
[ -d "$MANAGED_ROOT" ] && [ ! -L "$MANAGED_ROOT" ] \
  || fatal "managed hook root is not a real directory"
chmod 0700 "$MANAGED_ROOT"
STAGING=$(mktemp -d "$MANAGED_ROOT/.install.XXXXXX")

extract_blob() {
  local source_path=$1 expected_mode=$2 destination=$3 entry metadata tracked_path mode type oid
  entry=$(git -C "$ROOT" ls-tree "$DEFAULT_OID" -- ":(literal)$source_path")
  [ -n "$entry" ] || fatal "trusted default commit is missing $source_path"
  metadata=${entry%%$'\t'*}
  tracked_path=${entry#*$'\t'}
  [ "$tracked_path" = "$source_path" ] || fatal "unexpected trusted tree path for $source_path"
  read -r mode type oid <<< "$metadata"
  [ "$mode" = "$expected_mode" ] && [ "$type" = blob ] && is_oid "$oid" \
    || fatal "trusted default commit has an unsafe entry for $source_path"
  git -C "$ROOT" cat-file blob "$oid" > "$destination"
}

extract_blob scripts/pre-push-public-boundary.sh 100755 "$STAGING/pre-push"
extract_blob scripts/ci/check_public_repository_boundary.py 100755 \
  "$STAGING/check_public_repository_boundary.py"
extract_blob scripts/ci/boundary_guard_policy.py 100644 "$STAGING/boundary_guard_policy.py"
extract_blob scripts/ci/private-content-markers.txt 100644 "$STAGING/private-content-markers.txt"
printf '%s\n' "$DEFAULT_OID" > "$STAGING/source-commit"
chmod 0555 "$STAGING/pre-push" "$STAGING/check_public_repository_boundary.py"
chmod 0444 \
  "$STAGING/boundary_guard_policy.py" \
  "$STAGING/private-content-markers.txt" \
  "$STAGING/source-commit"

INSTALL_DIR="$MANAGED_ROOT/$DEFAULT_OID"
if [ -e "$INSTALL_DIR" ] || [ -L "$INSTALL_DIR" ]; then
  verify_bundle "$INSTALL_DIR" "$STAGING" \
    || fatal "existing immutable hook bundle does not match trusted commit $DEFAULT_OID"
else
  chmod 0555 "$STAGING"
  mv "$STAGING" "$INSTALL_DIR"
  STAGING=
fi
chmod 0555 "$INSTALL_DIR" "$INSTALL_DIR/pre-push" "$INSTALL_DIR/check_public_repository_boundary.py"
chmod 0444 \
  "$INSTALL_DIR/boundary_guard_policy.py" \
  "$INSTALL_DIR/private-content-markers.txt" \
  "$INSTALL_DIR/source-commit"

CURRENT=$(git -C "$ROOT" config --get-all core.hooksPath 2>/dev/null || true)
[[ "$CURRENT" != *$'\n'* ]] || fatal "multiple or malformed core.hooksPath values are configured"
if [ -z "$CURRENT" ] || [ "$CURRENT" = "$CANONICAL_HOOKS" ]; then
  canonical_hooks_are_samples_only \
    || fatal "canonical hooks directory contains a real hook; refusing to replace it"
elif [ "$CURRENT" = "$INSTALL_DIR" ]; then
  :
elif managed_hook_path "$CURRENT"; then
  [ -f "$CURRENT/pre-push" ] && [ ! -L "$CURRENT/pre-push" ] && [ -x "$CURRENT/pre-push" ] \
    || fatal "configured managed hooks path is not a valid installed bundle"
else
  fatal "core.hooksPath is already set to '$CURRENT'; refusing to replace it"
fi

if PREVIOUS_LOCAL=$(git -C "$ROOT" config --local --get-all core.hooksPath 2>/dev/null); then
  PREVIOUS_LOCAL_SET=1
else
  PREVIOUS_LOCAL_SET=0
  PREVIOUS_LOCAL=
fi
[[ "$PREVIOUS_LOCAL" != *$'\n'* ]] || fatal "multiple local core.hooksPath values are configured"

# Keep the source commit reachable so the immutable hook can use it as its
# trusted opaque-file baseline even after remote-tracking refs move.
git -C "$ROOT" update-ref "refs/hackforger-boundary/installed/$DEFAULT_OID" "$DEFAULT_OID"
git -C "$ROOT" config --local core.hooksPath "$INSTALL_DIR"
EFFECTIVE=$(git -C "$ROOT" rev-parse --path-format=absolute --git-path hooks 2>/dev/null || true)
if [ "$EFFECTIVE" != "$INSTALL_DIR" ] \
  || [ ! -f "$EFFECTIVE/pre-push" ] \
  || [ -L "$EFFECTIVE/pre-push" ] \
  || [ ! -x "$EFFECTIVE/pre-push" ]; then
  restore_local_config
  fatal "effective pre-push hook is not the installed immutable executable"
fi

echo "Installed HackForger public-boundary pre-push hook from $DEFAULT_OID"
echo "core.hooksPath=$INSTALL_DIR"
