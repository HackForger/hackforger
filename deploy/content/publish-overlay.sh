#!/usr/bin/env bash
# Validate and publish externally owned content without rebuilding or restarting
# HackForger. Deployment coordinates come only from an explicit config file.

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
REMOTE_HELPER="$SCRIPT_DIR/remote-transaction.sh"
EXCHANGE_HELPER="$SCRIPT_DIR/rename-exchange.py"

usage() {
  cat <<'EOF'
Usage:
  publish-overlay.sh --config FILE --mount landing --source DIR --manifest FILE
    [--provenance-remote REMOTE --provenance-ref refs/heads/BRANCH] [--apply]

Default behavior is a read-only dry run. SSH apply requires a clean, pushed
source repository and Linux renameat2(RENAME_EXCHANGE). It snapshots live
content while holding the deployment lock, validates the persistent local
copy, then performs a gap-free directory exchange. There is no production
non-atomic fallback.
EOF
}

die() { echo "FATAL: $*" >&2; exit 1; }
log() { printf '\n==> %s\n' "$*"; }

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

quote_arg() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

git_sanitized() {
  env \
    -u GIT_DIR -u GIT_COMMON_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE \
    -u GIT_OBJECT_DIRECTORY -u GIT_ALTERNATE_OBJECT_DIRECTORIES \
    -u GIT_NAMESPACE -u GIT_REPLACE_REF_BASE -u GIT_GRAFT_FILE -u GIT_SHALLOW_FILE \
    -u GIT_CONFIG -u GIT_CONFIG_PARAMETERS -u GIT_EXEC_PATH \
    -u GIT_CONFIG_COUNT -u GIT_CONFIG_KEY_0 -u GIT_CONFIG_VALUE_0 \
    -u GIT_SSH -u GIT_SSH_COMMAND -u GIT_SSH_VARIANT -u GIT_PROXY_COMMAND \
    GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null \
    GIT_NO_REPLACE_OBJECTS=1 GIT_LITERAL_PATHSPECS=1 GIT_OPTIONAL_LOCKS=0 \
    GIT_NO_LAZY_FETCH="${GIT_NO_LAZY_FETCH:-0}" GIT_TERMINAL_PROMPT=0 \
    GIT_SSH_COMMAND='ssh -F /dev/null -o BatchMode=yes -o ConnectTimeout=15 -o ServerAliveInterval=15 -o ServerAliveCountMax=3' \
    GIT_PROTOCOL_FROM_USER=0 git \
    -c protocol.ext.allow=never -c protocol.file.allow=never \
    -c core.fsmonitor=false -c core.hooksPath=/dev/null "$@"
}

validate_inode_types() {
  local tree=$1 special
  special=$(find "$tree" ! -type d ! -type f -print -quit)
  [ -z "$special" ] || die "content tree contains a non-directory/non-regular inode: $special"
}

validate_manifest_path() {
  local path=$1
  [ -n "$path" ] || die "empty manifest path"
  if [[ "$path" =~ [[:cntrl:]] ]]; then
    die "control characters are not allowed in manifest paths"
  fi
  case "$path" in
    /*|./*|../*|*/../*|*/..|*/./*|*//*|*\\*) die "unsafe manifest path: $path" ;;
  esac
}

verify_tree() {
  local tree=$1 manifest=$2 entrypoint=$3 work=$4
  local line hash separator path duplicate directory parent
  validate_inode_types "$tree"
  [ -f "$manifest" ] && [ ! -L "$manifest" ] || die "manifest is not a regular file"
  : > "$work/expected"
  while IFS= read -r line || [ -n "$line" ]; do
    [ -n "$line" ] || die "manifest contains a blank line"
    [ "${#line}" -ge 67 ] || die "malformed manifest line"
    hash=${line:0:64}
    separator=${line:64:2}
    path=${line:66}
    [[ "$hash" =~ ^[0-9a-f]{64}$ ]] || die "invalid manifest digest"
    [ "$separator" = "  " ] || die "manifest requires two spaces before each path"
    validate_manifest_path "$path"
    [ -f "$tree/$path" ] && [ ! -L "$tree/$path" ] || die "manifest file is not regular: $path"
    [ "$(sha256_file "$tree/$path")" = "$hash" ] || die "checksum mismatch: $path"
    printf '%s\n' "$path" >> "$work/expected"
  done < "$manifest"
  [ -s "$work/expected" ] || die "manifest is empty"
  [ -s "$tree/$entrypoint" ] || die "entrypoint is missing or empty"
  LC_ALL=C sort "$work/expected" > "$work/expected.sorted"
  duplicate=$(uniq -d "$work/expected.sorted" | head -1 || true)
  [ -z "$duplicate" ] || die "duplicate manifest path: $duplicate"
  (
    cd "$tree"
    find . -type f -print | sed 's#^\./##' | LC_ALL=C sort
  ) > "$work/actual.sorted"
  if ! cmp -s "$work/expected.sorted" "$work/actual.sorted"; then
    echo "FATAL: manifest does not exactly cover the content tree" >&2
    diff -u "$work/expected.sorted" "$work/actual.sorted" >&2 || true
    exit 1
  fi
  : > "$work/expected-dirs"
  printf '.\n' >> "$work/expected-dirs"
  while IFS= read -r path || [ -n "$path" ]; do
    directory=$(dirname "$path")
    while [ "$directory" != . ]; do
      printf '%s\n' "$directory" >> "$work/expected-dirs"
      parent=$(dirname "$directory")
      [ "$parent" != "$directory" ] || die "could not normalize manifest directory: $directory"
      directory=$parent
    done
  done < "$work/expected.sorted"
  LC_ALL=C sort -u "$work/expected-dirs" > "$work/expected-dirs.sorted"
  (
    cd "$tree"
    find . -type d -print | sed 's#^\./##' | LC_ALL=C sort
  ) > "$work/actual-dirs.sorted"
  if ! cmp -s "$work/expected-dirs.sorted" "$work/actual-dirs.sorted"; then
    echo "FATAL: archive/source contains directories not implied by the manifest" >&2
    diff -u "$work/expected-dirs.sorted" "$work/actual-dirs.sorted" >&2 || true
    exit 1
  fi
}

is_lfs_pointer() {
  local file=$1
  [ "$(sed -n '1p' "$file")" = 'version https://git-lfs.github.com/spec/v1' ] \
    && grep -q '^oid sha256:[0-9a-f]\{64\}$' "$file" \
    && grep -q '^size [0-9][0-9]*$' "$file"
}

CONFIG=""
MOUNT=""
SOURCE_INPUT=""
MANIFEST_INPUT=""
PROVENANCE_REMOTE=""
PROVENANCE_REF=""
APPLY=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --config) [ "$#" -ge 2 ] || die "--config needs a value"; CONFIG=$2; shift 2 ;;
    --mount) [ "$#" -ge 2 ] || die "--mount needs a value"; MOUNT=$2; shift 2 ;;
    --source) [ "$#" -ge 2 ] || die "--source needs a value"; SOURCE_INPUT=$2; shift 2 ;;
    --manifest) [ "$#" -ge 2 ] || die "--manifest needs a value"; MANIFEST_INPUT=$2; shift 2 ;;
    --provenance-remote) [ "$#" -ge 2 ] || die "--provenance-remote needs a value"; PROVENANCE_REMOTE=$2; shift 2 ;;
    --provenance-ref) [ "$#" -ge 2 ] || die "--provenance-ref needs a value"; PROVENANCE_REF=$2; shift 2 ;;
    --apply) APPLY=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; die "unknown argument: $1" ;;
  esac
done

[ -n "$CONFIG" ] && [ -f "$CONFIG" ] && [ ! -L "$CONFIG" ] \
  || die "--config must name a regular non-symlink file"
[ -n "$MOUNT" ] || die "--mount is required"
[ -n "$SOURCE_INPUT" ] && [ -d "$SOURCE_INPUT" ] || die "--source must name an existing directory"
[ -n "$MANIFEST_INPUT" ] && [ -f "$MANIFEST_INPUT" ] && [ ! -L "$MANIFEST_INPUT" ] \
  || die "--manifest must name a regular non-symlink file"

CONFIG_DIR=$(cd "$(dirname "$CONFIG")" && pwd -P)
CONFIG="$CONFIG_DIR/$(basename "$CONFIG")"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/publish-content.XXXXXX")
WORK=$(cd "$WORK" && pwd -P)
trap 'rm -rf "$WORK"' EXIT INT TERM HUP
CONFIG_SNAPSHOT="$WORK/config.input"
cp "$CONFIG" "$CONFIG_SNAPSHOT"
CONFIG_READ_HASH=$(sha256_file "$CONFIG_SNAPSHOT")

TRANSPORT=""
DEPLOY_TARGET=""
PUBLIC_BASE_URL=""
REMOTE_ROOT=""
LOCAL_BACKUP_ROOT=""
SMOKE_ASSET=""
ALLOW_INITIAL_INSTALL=false
REQUIRE_CLEAN_SOURCE=true
line_number=0
while IFS= read -r line || [ -n "$line" ]; do
  line_number=$((line_number + 1))
  case "$line" in ''|'#'*) continue ;; esac
  key=${line%%=*}
  [ "$key" != "$line" ] || die "config line $line_number is not KEY=value"
  value=${line#*=}
  [ -n "$value" ] || die "config key $key cannot be empty"
  case "$key" in
    TRANSPORT) TRANSPORT=$value ;;
    DEPLOY_TARGET) DEPLOY_TARGET=$value ;;
    PUBLIC_BASE_URL) PUBLIC_BASE_URL=$value ;;
    REMOTE_ROOT) REMOTE_ROOT=$value ;;
    LOCAL_BACKUP_ROOT) LOCAL_BACKUP_ROOT=$value ;;
    SMOKE_ASSET) SMOKE_ASSET=$value ;;
    ALLOW_INITIAL_INSTALL) ALLOW_INITIAL_INSTALL=$value ;;
    REQUIRE_CLEAN_SOURCE) REQUIRE_CLEAN_SOURCE=$value ;;
    *) die "unknown config key on line $line_number: $key" ;;
  esac
done < "$CONFIG_SNAPSHOT"

[ "$TRANSPORT" = ssh ] || [ "$TRANSPORT" = local ] || die "TRANSPORT must be ssh or local"
[ -n "$DEPLOY_TARGET" ] || die "DEPLOY_TARGET is required in --config"
[ -n "$PUBLIC_BASE_URL" ] || die "PUBLIC_BASE_URL is required in --config"
[ -n "$REMOTE_ROOT" ] || die "REMOTE_ROOT is required in --config"
[ -n "$LOCAL_BACKUP_ROOT" ] || die "LOCAL_BACKUP_ROOT is required in --config"
[ -n "$SMOKE_ASSET" ] || die "SMOKE_ASSET is required in --config"
[ "$ALLOW_INITIAL_INSTALL" = true ] || [ "$ALLOW_INITIAL_INSTALL" = false ] || die "ALLOW_INITIAL_INSTALL must be true or false"
[ "$REQUIRE_CLEAN_SOURCE" = true ] || [ "$REQUIRE_CLEAN_SOURCE" = false ] || die "REQUIRE_CLEAN_SOURCE must be true or false"
[[ "$DEPLOY_TARGET" =~ ^[A-Za-z0-9_.@:-]+$ ]] || die "DEPLOY_TARGET contains unsupported characters"
[[ "$PUBLIC_BASE_URL" =~ ^https?://[A-Za-z0-9._:-]+/?$ ]] || die "PUBLIC_BASE_URL must be an http(s) origin without a path"
[[ "$SMOKE_ASSET" =~ ^[A-Za-z0-9._/-]+$ ]] || die "SMOKE_ASSET must be URL-safe without encoding"
validate_manifest_path "$SMOKE_ASSET"
[ "$SMOKE_ASSET" != index.html ] || die "SMOKE_ASSET must be distinct from index.html"
case "$REMOTE_ROOT" in /*) ;; *) die "REMOTE_ROOT must be absolute" ;; esac
case "$LOCAL_BACKUP_ROOT" in /*) ;; *) die "LOCAL_BACKUP_ROOT must be absolute" ;; esac
[ "$REMOTE_ROOT" != / ] && [ "$LOCAL_BACKUP_ROOT" != / ] || die "configured roots cannot be /"
case "$REMOTE_ROOT$LOCAL_BACKUP_ROOT" in
  *"'"*|*'..'*|*$'\t'*|*$'\r'*|*$'\n'*) die "configured roots contain unsafe characters" ;;
esac
if [ "$TRANSPORT" = ssh ] && [ "$REQUIRE_CLEAN_SOURCE" != true ]; then
  die "ssh deployments require REQUIRE_CLEAN_SOURCE=true"
fi
if [ "$REQUIRE_CLEAN_SOURCE" = true ]; then
  [ -n "$PROVENANCE_REMOTE" ] && [ -n "$PROVENANCE_REF" ] \
    || die "clean-source publishing requires --provenance-remote and --provenance-ref trust anchors"
elif [ -n "$PROVENANCE_REMOTE" ] || [ -n "$PROVENANCE_REF" ]; then
  die "provenance trust anchors require REQUIRE_CLEAN_SOURCE=true"
fi

case "$MOUNT" in
  landing) MOUNT_PATH=custom/public/assets/landing; ENTRYPOINT=index.html ;;
  *) die "unsupported mount: $MOUNT" ;;
esac

SOURCE=$(cd "$SOURCE_INPUT" && pwd -P)
MANIFEST_DIR=$(cd "$(dirname "$MANIFEST_INPUT")" && pwd -P)
MANIFEST="$MANIFEST_DIR/$(basename "$MANIFEST_INPUT")"
MANIFEST_SNAPSHOT="$WORK/manifest.input"
cp "$MANIFEST" "$MANIFEST_SNAPSHOT"
MANIFEST_READ_HASH=$(sha256_file "$MANIFEST_SNAPSHOT")
case "$MANIFEST" in "$SOURCE"/*) die "manifest must live outside the content source tree" ;; esac
validate_inode_types "$SOURCE"

REMOTE_ARCHIVE=""
REMOTE_MANIFEST=""
REMOTE_EXCHANGE=""
REMOTE_UPLOAD_DIR=""
LOCK_HELD=0
LOCK_TOKEN=""
BACKUP_ID=""

run_helper() {
  if [ "$TRANSPORT" = local ]; then
    bash "$REMOTE_HELPER" "$@"
    return
  fi
  command="bash -s --"
  for arg in "$@"; do command="$command $(quote_arg "$arg")"; done
  ssh "$DEPLOY_TARGET" "$command" < "$REMOTE_HELPER"
}

cleanup() {
  rc=$?
  trap - EXIT INT TERM HUP
  set +e
  if [ "$LOCK_HELD" -eq 1 ] && [ -n "$LOCK_TOKEN" ]; then
    if [ -n "$BACKUP_ID" ]; then
      run_helper abort "$REMOTE_ROOT" "$LOCK_TOKEN" "$BACKUP_ID" \
        "$APPLY_EXCHANGE" "$EXCHANGE_HASH" publisher-preactivation-failure >/dev/null 2>&1
    else
      run_helper lock-release "$REMOTE_ROOT" "$LOCK_TOKEN" "$APPLY_EXCHANGE" "$EXCHANGE_HASH" >/dev/null 2>&1
    fi || \
      echo "WARN: deployment lock could not be released: $REMOTE_ROOT/.content-deploy.lock" >&2
  fi
  if [ "$TRANSPORT" = ssh ]; then
    [ -n "$REMOTE_UPLOAD_DIR" ] \
      && run_helper upload-clean "$REMOTE_UPLOAD_DIR" >/dev/null 2>&1 || true
  fi
  rm -rf "$WORK"
  exit "$rc"
}
trap cleanup EXIT INT TERM HUP

: > "$WORK/normalized.SHA256SUMS"
: > "$WORK/source-paths"
ENTRYPOINT_HASH=""
SMOKE_ASSET_HASH=""
while IFS= read -r line || [ -n "$line" ]; do
  [ -n "$line" ] && [ "${#line}" -ge 67 ] || die "malformed or blank manifest line"
  hash=$(printf '%s' "${line:0:64}" | tr A-F a-f)
  separator=${line:64:2}
  path=${line:66}
  [[ "$hash" =~ ^[0-9a-f]{64}$ ]] || die "invalid manifest digest"
  [ "$separator" = "  " ] || die "manifest requires two spaces before each path"
  validate_manifest_path "$path"
  [ -f "$SOURCE/$path" ] && [ ! -L "$SOURCE/$path" ] || die "manifest file is not regular: $path"
  [ "$(sha256_file "$SOURCE/$path")" = "$hash" ] || die "checksum mismatch: $path"
  is_lfs_pointer "$SOURCE/$path" && die "Git LFS pointer files are not deployable content: $path"
  printf '%s\n' "$path" >> "$WORK/source-paths"
  printf '%s  %s\n' "$hash" "$path" >> "$WORK/normalized.SHA256SUMS"
  [ "$path" = "$ENTRYPOINT" ] && ENTRYPOINT_HASH=$hash
  [ "$path" = "$SMOKE_ASSET" ] && SMOKE_ASSET_HASH=$hash
done < "$MANIFEST_SNAPSHOT"

[ -n "$ENTRYPOINT_HASH" ] && [ -s "$SOURCE/$ENTRYPOINT" ] || die "manifest entrypoint is missing or empty"
[ -n "$SMOKE_ASSET_HASH" ] && [ -s "$SOURCE/$SMOKE_ASSET" ] || die "configured SMOKE_ASSET is missing or empty"
VERIFY_SOURCE="$WORK/verify-source"
mkdir "$VERIFY_SOURCE"
verify_tree "$SOURCE" "$WORK/normalized.SHA256SUMS" "$ENTRYPOINT" "$VERIFY_SOURCE"

SOURCE_REVISION=unversioned
if [ "$REQUIRE_CLEAN_SOURCE" = true ]; then
  [ -z "${GIT_REPLACE_REF_BASE:-}" ] && [ -z "${GIT_GRAFT_FILE:-}" ] \
    || die "replace/graft environment overrides are forbidden"
  SOURCE_REPO=$(GIT_NO_LAZY_FETCH=1 git_sanitized -C "$SOURCE" rev-parse --show-toplevel 2>/dev/null) || die "source is not inside a git repository"
  MANIFEST_REPO=$(GIT_NO_LAZY_FETCH=1 git_sanitized -C "$MANIFEST_DIR" rev-parse --show-toplevel 2>/dev/null) || die "manifest is not inside a git repository"
  CONFIG_REPO=$(GIT_NO_LAZY_FETCH=1 git_sanitized -C "$CONFIG_DIR" rev-parse --show-toplevel 2>/dev/null) || die "config is not inside a git repository"
  SOURCE_REPO=$(cd "$SOURCE_REPO" && pwd -P)
  MANIFEST_REPO=$(cd "$MANIFEST_REPO" && pwd -P)
  CONFIG_REPO=$(cd "$CONFIG_REPO" && pwd -P)
  [ "$SOURCE_REPO" = "$MANIFEST_REPO" ] && [ "$SOURCE_REPO" = "$CONFIG_REPO" ] \
    || die "source, manifest, and config must belong to the same git repository"

  git_source() {
    GIT_NO_LAZY_FETCH=1 git_sanitized -C "$SOURCE_REPO" "$@"
  }

  COMMON_DIR=$(git_source rev-parse --git-common-dir)
  case "$COMMON_DIR" in /*) ;; *) COMMON_DIR="$SOURCE_REPO/$COMMON_DIR" ;; esac
  COMMON_DIR=$(cd "$COMMON_DIR" && pwd -P)
  SOURCE_GIT_DIR=$(git_source rev-parse --absolute-git-dir)
  SOURCE_GIT_DIR=$(cd "$SOURCE_GIT_DIR" && pwd -P)
  [ ! -e "$COMMON_DIR/info/grafts" ] && [ ! -L "$COMMON_DIR/info/grafts" ] \
    || die "legacy Git grafts are forbidden"
  [ ! -e "$COMMON_DIR/objects/info/alternates" ] \
    && [ ! -L "$COMMON_DIR/objects/info/alternates" ] \
    || die "Git object alternates are forbidden for provenance"
  if [ "$SOURCE_GIT_DIR" != "$COMMON_DIR" ]; then
    [ ! -e "$SOURCE_GIT_DIR/objects/info/alternates" ] \
      && [ ! -L "$SOURCE_GIT_DIR/objects/info/alternates" ] \
      || die "Git worktree object alternates are forbidden for provenance"
  fi
  [ -z "$(git_source for-each-ref --format='%(refname)' refs/replace)" ] || die "Git replace refs are forbidden"
  [ "$(git_source rev-parse --is-shallow-repository)" = false ] || die "shallow source repositories are forbidden"
  SOURCE_UNTRUSTED_CONFIG=$(git_source config --show-origin --get-regexp \
    '^(extensions\.partialclone|remote\..*\.(promisor|partialclonefilter)|filter\..*\.(clean|smudge|process))$' \
    2>/dev/null || true)
  [ -z "$SOURCE_UNTRUSTED_CONFIG" ] \
    || die "partial-clone, promisor, or conversion-filter config is forbidden for provenance"
  SOURCE_REVISION=$(git_source rev-parse --verify HEAD)
  SOURCE_GITLINK=$(git_source ls-files --stage | awk '$1 == "160000" {print "present"; exit}')
  [ -z "$SOURCE_GITLINK" ] || die "Git submodules are forbidden for provenance"
  git_source diff-index --cached --quiet --no-ext-diff --no-textconv \
    --ignore-submodules=all "$SOURCE_REVISION" -- \
    || die "source repository index differs from HEAD"
  git_source ls-files --others --exclude-standard -z > "$WORK/source-untracked"
  [ ! -s "$WORK/source-untracked" ] || die "source repository contains untracked files"
  BRANCH=$(git_source symbolic-ref -q --short HEAD) || die "source repository must be on a branch"

  [[ "$PROVENANCE_REF" =~ ^refs/heads/[A-Za-z0-9._/-]+$ ]] \
    && git_sanitized check-ref-format "$PROVENANCE_REF" >/dev/null 2>&1 \
    || die "--provenance-ref must be a valid refs/heads/* branch"
  [ "$PROVENANCE_REF" = "refs/heads/$BRANCH" ] \
    || die "checked-out branch does not match --provenance-ref"
  if [ "$TRANSPORT" = ssh ]; then
    [[ "$PROVENANCE_REMOTE" =~ ^git@github\.com:[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+\.git$ ]] \
      || die "SSH provenance remote must be exactly git@github.com:OWNER/REPO.git"
    PROVENANCE_FILE_POLICY=never
  else
    case "$PROVENANCE_REMOTE" in /*) ;; *) die "local-test provenance remote must be an absolute bare repository path" ;; esac
    [ -d "$PROVENANCE_REMOTE" ] && [ ! -L "$PROVENANCE_REMOTE" ] \
      || die "local-test provenance remote must be a real bare repository directory"
    PROVENANCE_REMOTE_REAL=$(cd "$PROVENANCE_REMOTE" && pwd -P)
    [ "$PROVENANCE_REMOTE_REAL" = "$PROVENANCE_REMOTE" ] \
      || die "local-test provenance remote must be canonical"
    [ "$(git_sanitized -C "$PROVENANCE_REMOTE" rev-parse --is-bare-repository 2>/dev/null)" = true ] \
      || die "local-test provenance remote is not bare"
    PROVENANCE_FILE_POLICY=always
  fi

  PROVENANCE_CWD="$WORK/provenance-cwd"
  mkdir "$PROVENANCE_CWD"

  git_provenance() {
    git_sanitized -C "$PROVENANCE_CWD" -c protocol.ext.allow=never \
      -c "protocol.file.allow=$PROVENANCE_FILE_POLICY" \
      -c fetch.fsckObjects=true -c transfer.fsckObjects=true "$@"
  }

  query_provenance_tip() {
    local result tip
    result=$(git_provenance ls-remote --refs --exit-code "$PROVENANCE_REMOTE" "$PROVENANCE_REF") \
      || die "could not query the provenance branch tip"
    tip=$(printf '%s\n' "$result" | awk -v ref="$PROVENANCE_REF" '$2 == ref {print $1}')
    [[ "$tip" =~ ^([0-9a-f]{40}|[0-9a-f]{64})$ ]] \
      || die "provenance remote returned an invalid branch tip"
    printf '%s\n' "$tip"
  }

  REMOTE_TIP=$(query_provenance_tip)
  [ "$SOURCE_REVISION" = "$REMOTE_TIP" ] \
    || die "source HEAD is not equal to the provenance branch tip"

  case "$SOURCE" in "$SOURCE_REPO"/*) SOURCE_REL=${SOURCE#"$SOURCE_REPO"/} ;; *) die "source is outside its repository" ;; esac
  case "$MANIFEST" in "$SOURCE_REPO"/*) MANIFEST_REL=${MANIFEST#"$SOURCE_REPO"/} ;; *) die "manifest is outside its repository" ;; esac
  case "$CONFIG" in "$SOURCE_REPO"/*) CONFIG_REL=${CONFIG#"$SOURCE_REPO"/} ;; *) die "config is outside its repository" ;; esac

  # Fetch the remote commit and tree graph without the potentially large blob
  # history. The remote config and manifest blobs are fetched on demand; local
  # source blobs are accepted only when both their remote Git object IDs and
  # the SHA-256 hashes from that remote manifest agree.
  TRUST_REPO="$WORK/provenance.git"
  EMPTY_TEMPLATE="$WORK/empty-git-template"
  mkdir "$EMPTY_TEMPLATE"
  git_sanitized -C "$PROVENANCE_CWD" init -q --bare --template="$EMPTY_TEMPLATE" "$TRUST_REPO"
  git_provenance --git-dir "$TRUST_REPO" remote add provenance "$PROVENANCE_REMOTE"
  git_sanitized --git-dir "$TRUST_REPO" config extensions.partialClone provenance
  git_sanitized --git-dir "$TRUST_REPO" config remote.provenance.promisor true
  git_sanitized --git-dir "$TRUST_REPO" config remote.provenance.partialclonefilter blob:none
  git_provenance --git-dir "$TRUST_REPO" fetch -q --no-tags --force \
    --depth=1 --filter=blob:none provenance \
    "+$PROVENANCE_REF:refs/provenance/source"
  FETCHED_REVISION=$(git_sanitized --git-dir "$TRUST_REPO" rev-parse --verify 'refs/provenance/source^{commit}')
  [ "$FETCHED_REVISION" = "$SOURCE_REVISION" ] \
    || die "provenance branch changed while the authoritative tree was fetched"
  [ -z "$(git_sanitized --git-dir "$TRUST_REPO" for-each-ref --format='%(refname)' refs/replace)" ] \
    || die "authoritative provenance repository contains replacement refs"
  GIT_NO_LAZY_FETCH=1 git_sanitized --git-dir "$TRUST_REPO" fsck --connectivity-only --strict \
    --no-reflogs "$FETCHED_REVISION" >/dev/null

  verify_source_worktree() {
    local entry metadata mode oid stage path actual_oid link_target
    git_source diff-index --cached --quiet --no-ext-diff --no-textconv \
      --ignore-submodules=all "$SOURCE_REVISION" -- \
      || die "source repository index differs from HEAD"
    git_source ls-files --others --exclude-standard -z > "$WORK/source-untracked"
    [ ! -s "$WORK/source-untracked" ] \
      || die "source repository contains untracked files"
    git_source ls-files --stage -z > "$WORK/source-index"
    while IFS= read -r -d '' entry; do
      metadata=${entry%%$'\t'*}
      path=${entry#*$'\t'}
      read -r mode oid stage <<< "$metadata"
      [ "$stage" = 0 ] || die "source repository index contains unresolved entries"
      case "$mode" in
        100644|100755)
          [ -f "$SOURCE_REPO/$path" ] && [ ! -L "$SOURCE_REPO/$path" ] \
            || die "tracked working file is missing or not regular: $path"
          actual_oid=$(GIT_NO_LAZY_FETCH=1 git_sanitized --git-dir "$TRUST_REPO" \
            hash-object --stdin < "$SOURCE_REPO/$path")
          ;;
        120000)
          [ -L "$SOURCE_REPO/$path" ] || die "tracked working symlink is missing: $path"
          link_target=$(readlink "$SOURCE_REPO/$path")
          actual_oid=$(printf '%s' "$link_target" \
            | GIT_NO_LAZY_FETCH=1 git_sanitized --git-dir "$TRUST_REPO" hash-object --stdin)
          ;;
        160000) die "Git submodules are forbidden for provenance" ;;
        *) die "source repository index contains an unsupported mode: $path" ;;
      esac
      [ "$actual_oid" = "$oid" ] || die "tracked working file differs from the index: $path"
    done < "$WORK/source-index"
  }

  verify_source_worktree

  AUTHORITATIVE_ROOT="$WORK/authoritative"
  AUTHORITATIVE_SOURCE="$AUTHORITATIVE_ROOT/source"
  mkdir -p "$AUTHORITATIVE_SOURCE"

  remote_blob_oid() {
    local repository_path=$1 entry metadata mode type oid returned_path
    entry=$(git_sanitized --git-dir "$TRUST_REPO" ls-tree "$SOURCE_REVISION" -- "$repository_path")
    metadata=${entry%%$'\t'*}
    returned_path=${entry#*$'\t'}
    read -r mode type oid <<< "$metadata"
    [ "$returned_path" = "$repository_path" ] \
      || die "provenance commit lacks a regular file: $repository_path"
    case "$mode:$type" in 100644:blob|100755:blob) ;; *) die "provenance commit lacks a regular file: $repository_path" ;; esac
    printf '%s\n' "$oid"
  }

  require_promised_blob() {
    local oid=$1 repository_path=$2
    if GIT_NO_LAZY_FETCH=1 git_sanitized --git-dir "$TRUST_REPO" \
      cat-file -e "$oid^{blob}" 2>/dev/null; then
      die "provenance server did not honor blobless filtering: $repository_path"
    fi
  }

  materialize_authoritative_file() {
    local working_file=$1 oid=$2 repository_path=$3 output=$4
    mkdir -p "$(dirname "$output")"
    GIT_NO_LAZY_FETCH=0 git_provenance --git-dir "$TRUST_REPO" cat-file blob "$oid" > "$output"
    cmp -s "$output" "$working_file" || die "working file differs from provenance commit: $repository_path"
  }

  AUTHORITATIVE_TREE="$WORK/authoritative-tree"
  git_sanitized --git-dir "$TRUST_REPO" ls-tree -r -z "$SOURCE_REVISION" -- "$SOURCE_REL" \
    > "$AUTHORITATIVE_TREE"
  [ -s "$AUTHORITATIVE_TREE" ] || die "provenance source subtree is empty"

  CONFIG_OID=$(remote_blob_oid "$CONFIG_REL")
  MANIFEST_OID=$(remote_blob_oid "$MANIFEST_REL")
  require_promised_blob "$CONFIG_OID" "$CONFIG_REL"
  require_promised_blob "$MANIFEST_OID" "$MANIFEST_REL"
  while IFS= read -r -d '' tree_entry; do
    tree_metadata=${tree_entry%%$'\t'*}
    repository_path=${tree_entry#*$'\t'}
    read -r tree_mode tree_type tree_oid <<< "$tree_metadata"
    case "$tree_mode:$tree_type" in
      100644:blob|100755:blob) ;;
      *) die "provenance source contains a non-regular Git entry: $repository_path" ;;
    esac
    require_promised_blob "$tree_oid" "$repository_path"
  done < "$AUTHORITATIVE_TREE"

  materialize_authoritative_file "$CONFIG_SNAPSHOT" "$CONFIG_OID" \
    "$CONFIG_REL" "$AUTHORITATIVE_ROOT/config"
  materialize_authoritative_file "$MANIFEST_SNAPSHOT" "$MANIFEST_OID" \
    "$MANIFEST_REL" "$AUTHORITATIVE_ROOT/manifest"
  [ "$(sha256_file "$AUTHORITATIVE_ROOT/config")" = "$CONFIG_READ_HASH" ] \
    || die "config changed while it was being read"
  [ "$(sha256_file "$AUTHORITATIVE_ROOT/manifest")" = "$MANIFEST_READ_HASH" ] \
    || die "manifest changed while it was being read"

  : > "$WORK/authoritative-source-paths"
  while IFS= read -r -d '' tree_entry; do
    tree_metadata=${tree_entry%%$'\t'*}
    repository_path=${tree_entry#*$'\t'}
    read -r tree_mode tree_type tree_oid <<< "$tree_metadata"
    case "$repository_path" in
      "$SOURCE_REL"/*) path=${repository_path#"$SOURCE_REL"/} ;;
      *) die "provenance tree contains a path outside the source subtree" ;;
    esac
    case "$tree_mode:$tree_type" in
      100644:blob|100755:blob) ;;
      *) die "provenance source contains a non-regular Git entry: $repository_path" ;;
    esac
    validate_manifest_path "$path"
    [ -f "$SOURCE/$path" ] && [ ! -L "$SOURCE/$path" ] \
      || die "working source lacks a provenance file: $repository_path"
    mkdir -p "$(dirname "$AUTHORITATIVE_SOURCE/$path")"
    cp "$SOURCE/$path" "$AUTHORITATIVE_SOURCE/$path"
    case "$tree_mode" in
      100644) chmod 0644 "$AUTHORITATIVE_SOURCE/$path" ;;
      100755) chmod 0755 "$AUTHORITATIVE_SOURCE/$path" ;;
    esac
    COPIED_OID=$(GIT_NO_LAZY_FETCH=1 git_sanitized --git-dir "$TRUST_REPO" \
      hash-object --stdin < "$AUTHORITATIVE_SOURCE/$path")
    [ "$COPIED_OID" = "$tree_oid" ] \
      || die "working source differs from provenance commit: $repository_path"
    printf '%s\n' "$path" >> "$WORK/authoritative-source-paths"
  done < "$AUTHORITATIVE_TREE"
  [ -s "$WORK/authoritative-source-paths" ] || die "provenance source subtree is empty"
  VERIFY_AUTHORITATIVE="$WORK/verify-authoritative"
  mkdir "$VERIFY_AUTHORITATIVE"
  verify_tree "$AUTHORITATIVE_SOURCE" "$WORK/normalized.SHA256SUMS" "$ENTRYPOINT" "$VERIFY_AUTHORITATIVE"
  SOURCE=$AUTHORITATIVE_SOURCE

  verify_source_worktree

  VERIFIED_REMOTE_TIP=$(query_provenance_tip)
  [ "$VERIFIED_REMOTE_TIP" = "$SOURCE_REVISION" ] \
    || die "provenance branch changed while the authoritative tree was materialized"
else
  if git -C "$SOURCE" rev-parse HEAD >/dev/null 2>&1; then
    SOURCE_REVISION=$(git -C "$SOURCE" rev-parse HEAD)
  fi
fi

MANIFEST_HASH=$(sha256_file "$WORK/normalized.SHA256SUMS")
RELEASE_ID=$MANIFEST_HASH
FILE_COUNT=$(wc -l < "$WORK/source-paths" | tr -d ' ')

log "Validating deployment target"
INSPECTION=$(run_helper inspect "$REMOTE_ROOT" "$MOUNT_PATH" "$ENTRYPOINT")
printf '%s\n' "$INSPECTION" | sed 's/^/  /'
LIVE_EXISTS=$(printf '%s\n' "$INSPECTION" | sed -n 's/^live_exists=//p')
[ "$LIVE_EXISTS" = 1 ] || [ "$LIVE_EXISTS" = 0 ] || die "target inspection returned invalid state"
[ "$LIVE_EXISTS" = 1 ] || [ "$ALLOW_INITIAL_INSTALL" = true ] || die "live content is absent and initial installation is disabled"

ACTIVATION_MODE=local-test
[ "$TRANSPORT" = ssh ] && ACTIVATION_MODE=exchange
echo
echo "Content deployment plan"
echo "  transport:       $TRANSPORT"
echo "  activation:      $ACTIVATION_MODE"
echo "  target:          $DEPLOY_TARGET"
echo "  public URL:      ${PUBLIC_BASE_URL%/}/"
echo "  remote root:     $REMOTE_ROOT"
echo "  mount:           $MOUNT_PATH"
echo "  smoke asset:     $SMOKE_ASSET"
echo "  source revision: $SOURCE_REVISION"
echo "  release id:      $RELEASE_ID"
echo "  files:           $FILE_COUNT"
echo "  local backups:   $LOCAL_BACKUP_ROOT"
if [ "$APPLY" -eq 0 ]; then
  echo
  echo "DRY RUN: no backup, lock, staging, live, or marker files were changed."
  exit 0
fi

log "Building and re-extracting immutable archive"
ARCHIVE="$WORK/content.tar"
COPYFILE_DISABLE=1 tar -cf "$ARCHIVE" -C "$SOURCE" .
ARCHIVE_CHECK="$WORK/archive-check"
mkdir "$ARCHIVE_CHECK"
COPYFILE_DISABLE=1 tar -xf "$ARCHIVE" -C "$ARCHIVE_CHECK"
VERIFY_ARCHIVE="$WORK/verify-archive"
mkdir "$VERIFY_ARCHIVE"
verify_tree "$ARCHIVE_CHECK" "$WORK/normalized.SHA256SUMS" "$ENTRYPOINT" "$VERIFY_ARCHIVE"
ARCHIVE_HASH=$(sha256_file "$ARCHIVE")
EXCHANGE_HASH=$(sha256_file "$EXCHANGE_HELPER")
BACKUP_ID="$(date -u +%Y%m%dT%H%M%SZ)-${RELEASE_ID:0:12}-$$"
LOCK_TOKEN=$BACKUP_ID

if [ "$TRANSPORT" = ssh ]; then
  REMOTE_UPLOAD_DIR=$(run_helper upload-create)
  [[ "$REMOTE_UPLOAD_DIR" =~ ^/tmp/hackforger-content-upload\.[A-Za-z0-9]+$ ]] \
    || die "remote upload allocator returned an unsafe path"
  REMOTE_ARCHIVE="$REMOTE_UPLOAD_DIR/content.tar"
  REMOTE_MANIFEST="$REMOTE_UPLOAD_DIR/manifest.SHA256SUMS"
  REMOTE_EXCHANGE="$REMOTE_UPLOAD_DIR/rename-exchange.py"
  scp -q "$ARCHIVE" "$DEPLOY_TARGET:$REMOTE_ARCHIVE"
  scp -q "$WORK/normalized.SHA256SUMS" "$DEPLOY_TARGET:$REMOTE_MANIFEST"
  scp -q "$EXCHANGE_HELPER" "$DEPLOY_TARGET:$REMOTE_EXCHANGE"
  run_helper upload-verify "$REMOTE_UPLOAD_DIR" \
    "$ARCHIVE_HASH" "$MANIFEST_HASH" "$EXCHANGE_HASH"
  APPLY_ARCHIVE=$REMOTE_ARCHIVE
  APPLY_MANIFEST=$REMOTE_MANIFEST
  APPLY_EXCHANGE=$REMOTE_EXCHANGE
  SMOKE_MODE=http
else
  APPLY_ARCHIVE=$ARCHIVE
  APPLY_MANIFEST="$WORK/normalized.SHA256SUMS"
  APPLY_EXCHANGE=$EXCHANGE_HELPER
  SMOKE_MODE=local
fi

log "Acquiring deployment lock and probing activation capability"
run_helper lock-acquire "$REMOTE_ROOT" "$MOUNT_PATH" "$ENTRYPOINT" "$LOCK_TOKEN" \
  "$ACTIVATION_MODE" "$APPLY_EXCHANGE" "$EXCHANGE_HASH" "$ALLOW_INITIAL_INSTALL"
LOCK_HELD=1

log "Creating authoritative snapshot while lock is held"
SNAPSHOT=$(run_helper snapshot "$REMOTE_ROOT" "$MOUNT_PATH" "$ENTRYPOINT" \
  "$LOCK_TOKEN" "$BACKUP_ID" "$ALLOW_INITIAL_INSTALL" "$APPLY_EXCHANGE" "$EXCHANGE_HASH")
printf '%s\n' "$SNAPSHOT" | sed 's/^/  /'
SNAPSHOT_LIVE=$(printf '%s\n' "$SNAPSHOT" | sed -n 's/^live_exists=//p')
SNAPSHOT_ARCHIVE_HASH=$(printf '%s\n' "$SNAPSHOT" | sed -n 's/^backup_archive_sha256=//p')
SNAPSHOT_MANIFEST_HASH=$(printf '%s\n' "$SNAPSHOT" | sed -n 's/^previous_manifest_sha256=//p')
SNAPSHOT_MARKER_HASH=$(printf '%s\n' "$SNAPSHOT" | sed -n 's/^previous_marker_sha256=//p')
[ "$SNAPSHOT_LIVE" = 1 ] || [ "$SNAPSHOT_LIVE" = 0 ] || die "snapshot returned invalid live state"
REMOTE_BACKUP="$REMOTE_ROOT/.content-backups/$BACKUP_ID"
LOCAL_BACKUP="$LOCAL_BACKUP_ROOT/$BACKUP_ID"
mkdir -p "$LOCAL_BACKUP_ROOT"
[ -d "$LOCAL_BACKUP_ROOT" ] && [ ! -L "$LOCAL_BACKUP_ROOT" ] || die "LOCAL_BACKUP_ROOT is not a real directory"
LOCAL_BACKUP_PARENT=$(dirname "$LOCAL_BACKUP_ROOT")
[ -d "$LOCAL_BACKUP_PARENT" ] && [ ! -L "$LOCAL_BACKUP_PARENT" ] || die "LOCAL_BACKUP_ROOT parent is not a real directory"
[ ! -e "$LOCAL_BACKUP" ] && [ ! -L "$LOCAL_BACKUP" ] || die "local backup id already exists"
mkdir "$LOCAL_BACKUP"
printf 'version=1\ntarget=%s\nmount=%s\nnew_release_id=%s\nsource_revision=%s\ncreated_at=%s\n' \
  "$DEPLOY_TARGET" "$MOUNT_PATH" "$RELEASE_ID" "$SOURCE_REVISION" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$LOCAL_BACKUP/metadata"

if [ "$SNAPSHOT_LIVE" = 1 ]; then
  if [ "$TRANSPORT" = local ]; then
    cp "$REMOTE_BACKUP/live.tar" "$LOCAL_BACKUP/live.tar"
    cp "$REMOTE_BACKUP/previous.SHA256SUMS" "$LOCAL_BACKUP/previous.SHA256SUMS"
  else
    scp -q "$DEPLOY_TARGET:$REMOTE_BACKUP/live.tar" "$LOCAL_BACKUP/live.tar"
    scp -q "$DEPLOY_TARGET:$REMOTE_BACKUP/previous.SHA256SUMS" "$LOCAL_BACKUP/previous.SHA256SUMS"
  fi
  [ "$(sha256_file "$LOCAL_BACKUP/live.tar")" = "$SNAPSHOT_ARCHIVE_HASH" ] || die "persistent local backup hash differs from locked snapshot"
  [ "$(sha256_file "$LOCAL_BACKUP/previous.SHA256SUMS")" = "$SNAPSHOT_MANIFEST_HASH" ] || die "persistent local manifest hash differs from locked snapshot"
  LOCAL_OLD_CHECK="$WORK/local-old-check"
  mkdir "$LOCAL_OLD_CHECK"
  COPYFILE_DISABLE=1 tar -xf "$LOCAL_BACKUP/live.tar" -C "$LOCAL_OLD_CHECK"
  VERIFY_OLD="$WORK/verify-old"
  mkdir "$VERIFY_OLD"
  verify_tree "$LOCAL_OLD_CHECK/landing" "$LOCAL_BACKUP/previous.SHA256SUMS" "$ENTRYPOINT" "$VERIFY_OLD"
  printf '%s  %s\n' "$SNAPSHOT_ARCHIVE_HASH" live.tar > "$LOCAL_BACKUP/SHA256SUMS"
else
  SNAPSHOT_ARCHIVE_HASH=absent
  SNAPSHOT_MANIFEST_HASH=absent
  : > "$LOCAL_BACKUP/live-was-absent"
fi

if [ "$TRANSPORT" = local ]; then
  if [ -f "$REMOTE_BACKUP/previous-marker" ]; then
    cp "$REMOTE_BACKUP/previous-marker" "$LOCAL_BACKUP/previous-marker"
    [ "$(sha256_file "$LOCAL_BACKUP/previous-marker")" = "$SNAPSHOT_MARKER_HASH" ] || die "persistent local marker hash differs from locked snapshot"
  else
    [ "$SNAPSHOT_MARKER_HASH" = absent ] || die "locked snapshot marker disappeared"
    : > "$LOCAL_BACKUP/marker-was-absent"
  fi
else
  if ssh "$DEPLOY_TARGET" "test -f '$REMOTE_BACKUP/previous-marker'"; then
    scp -q "$DEPLOY_TARGET:$REMOTE_BACKUP/previous-marker" "$LOCAL_BACKUP/previous-marker"
    [ "$(sha256_file "$LOCAL_BACKUP/previous-marker")" = "$SNAPSHOT_MARKER_HASH" ] || die "persistent local marker hash differs from locked snapshot"
  else
    [ "$SNAPSHOT_MARKER_HASH" = absent ] || die "locked snapshot marker disappeared"
    : > "$LOCAL_BACKUP/marker-was-absent"
  fi
fi

python3 "$EXCHANGE_HELPER" fsync-tree "$LOCAL_BACKUP"
python3 "$EXCHANGE_HELPER" fsync "$LOCAL_BACKUP" "$LOCAL_BACKUP_ROOT" "$LOCAL_BACKUP_PARENT"

run_helper backup-ack "$REMOTE_ROOT" "$LOCK_TOKEN" "$BACKUP_ID" \
  "$SNAPSHOT_ARCHIVE_HASH" "$SNAPSHOT_MANIFEST_HASH" "$SNAPSHOT_MARKER_HASH" \
  "$APPLY_EXCHANGE" "$EXCHANGE_HASH"

if [ "$REQUIRE_CLEAN_SOURCE" = true ]; then
  FINAL_REMOTE_TIP=$(query_provenance_tip)
  [ "$FINAL_REMOTE_TIP" = "$SOURCE_REVISION" ] \
    || die "provenance branch changed before activation; refusing the stale release"
fi

log "Activating verified release"
# From here the remote transaction owns lock release. A rollback failure keeps
# the lock and writes MANUAL_RECOVERY; the outer cleanup must not remove it.
LOCK_HELD=0
run_helper apply \
  "$REMOTE_ROOT" "$MOUNT_PATH" "$ENTRYPOINT" \
  "$APPLY_ARCHIVE" "$APPLY_MANIFEST" \
  "$ARCHIVE_HASH" "$MANIFEST_HASH" "$ENTRYPOINT_HASH" \
  "$SMOKE_ASSET" "$SMOKE_ASSET_HASH" \
  "$RELEASE_ID" "$SOURCE_REVISION" "$BACKUP_ID" "${PUBLIC_BASE_URL%/}" \
  "$SMOKE_MODE" "$ALLOW_INITIAL_INSTALL" "$LOCK_TOKEN" "$ACTIVATION_MODE" \
  "$APPLY_EXCHANGE" "$EXCHANGE_HASH"

echo
echo "Published content release $RELEASE_ID"
echo "  local backup:  $LOCAL_BACKUP"
echo "  remote backup: $REMOTE_BACKUP"
echo "  remote marker: $REMOTE_ROOT/.last-content-deploy"
