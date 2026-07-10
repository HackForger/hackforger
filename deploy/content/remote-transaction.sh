#!/usr/bin/env bash
# Transaction engine used locally for fixtures and over ssh for production.
# SSH activation requires Linux renameat2(RENAME_EXCHANGE); no non-atomic
# production fallback exists.

set -euo pipefail

die() {
  echo "FATAL: $*" >&2
  exit 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

device_id() {
  if stat -c '%d' "$1" >/dev/null 2>&1; then
    stat -c '%d' "$1"
  else
    stat -f '%d' "$1"
  fi
}

verify_durability_helper() {
  local helper=$1 expected_hash=$2
  command -v python3 >/dev/null 2>&1 || die "python3 is required for durable transactions"
  [ -f "$helper" ] && [ ! -L "$helper" ] || die "durability helper is not a regular file"
  [ "$(sha256_file "$helper")" = "$expected_hash" ] || die "durability helper checksum changed"
}

durable_paths() {
  local helper=$1
  shift
  python3 "$helper" fsync "$@"
}

durable_tree() {
  python3 "$1" fsync-tree "$2"
}

file_mode() {
  if stat -c '%a' "$1" >/dev/null 2>&1; then
    stat -c '%a' "$1"
  else
    stat -f '%Lp' "$1"
  fi
}

link_count() {
  if stat -c '%h' "$1" >/dev/null 2>&1; then
    stat -c '%h' "$1"
  else
    stat -f '%l' "$1"
  fi
}

validate_upload_dir() {
  local upload_dir=$1
  [[ "$upload_dir" =~ ^/tmp/hackforger-content-upload\.[A-Za-z0-9]+$ ]] \
    || die "upload directory is outside the managed private namespace"
  [ -d "$upload_dir" ] && [ ! -L "$upload_dir" ] && [ -O "$upload_dir" ] \
    || die "upload directory is not a real directory owned by the deployment user"
  [ "$(file_mode "$upload_dir")" = 700 ] \
    || die "upload directory permissions are not 0700"
}

upload_create() {
  [ "$#" -eq 0 ] || die "upload-create takes no arguments"
  local upload_dir
  umask 077
  upload_dir=$(mktemp -d /tmp/hackforger-content-upload.XXXXXXXXXXXX)
  chmod 700 "$upload_dir"
  validate_upload_dir "$upload_dir"
  printf '%s\n' "$upload_dir"
}

upload_verify() {
  [ "$#" -eq 4 ] \
    || die "upload-verify expects DIR ARCHIVE_HASH MANIFEST_HASH HELPER_HASH"
  local upload_dir=$1 archive_hash=$2 manifest_hash=$3 helper_hash=$4
  local archive manifest helper path count
  validate_upload_dir "$upload_dir"
  archive="$upload_dir/content.tar"
  manifest="$upload_dir/manifest.SHA256SUMS"
  helper="$upload_dir/rename-exchange.py"
  count=$(find "$upload_dir" -mindepth 1 -maxdepth 1 -print | wc -l | tr -d ' ')
  [ "$count" = 3 ] || die "upload directory does not contain exactly three inputs"
  for path in "$archive" "$manifest" "$helper"; do
    [ -f "$path" ] && [ ! -L "$path" ] && [ -O "$path" ] \
      || die "uploaded input is not a regular file owned by the deployment user"
    [ "$(link_count "$path")" = 1 ] || die "uploaded input has multiple hard links"
    chmod 600 "$path"
  done
  [ "$(sha256_file "$archive")" = "$archive_hash" ] || die "uploaded archive checksum mismatch"
  [ "$(sha256_file "$manifest")" = "$manifest_hash" ] || die "uploaded manifest checksum mismatch"
  verify_durability_helper "$helper" "$helper_hash"
  durable_paths "$helper" "$archive" "$manifest" "$helper" "$upload_dir"
}

upload_clean() {
  [ "$#" -eq 1 ] || die "upload-clean expects DIR"
  local upload_dir=$1 path
  validate_upload_dir "$upload_dir"
  for path in \
    "$upload_dir/content.tar" \
    "$upload_dir/manifest.SHA256SUMS" \
    "$upload_dir/rename-exchange.py"; do
    if [ -e "$path" ] || [ -L "$path" ]; then
      rm -f -- "$path"
    fi
  done
  [ -z "$(find "$upload_dir" -mindepth 1 -maxdepth 1 -print -quit)" ] \
    || die "upload directory contains an unexpected entry"
  rmdir -- "$upload_dir"
}

write_phase() {
  local backup=$1 helper=$2 phase=$3 temporary
  shift 3
  temporary=$(mktemp "$backup/.PHASE.XXXXXX")
  printf 'phase=%s\nrecorded_at=%s\n' "$phase" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$temporary"
  while [ "$#" -gt 0 ]; do
    printf '%s\n' "$1" >> "$temporary"
    shift
  done
  durable_paths "$helper" "$temporary"
  mv "$temporary" "$backup/PHASE"
  durable_paths "$helper" "$backup/PHASE" "$backup"
}

validate_root_and_mount() {
  local root=$1 mount_path=$2 entrypoint=$3 root_real current component
  case "$root" in /*) ;; *) die "remote root must be absolute" ;; esac
  [ "$root" != "/" ] || die "remote root cannot be /"
  [ "$mount_path" = "custom/public/assets/landing" ] || die "unsupported content mount"
  [ "$entrypoint" = "index.html" ] || die "unsupported content entrypoint"
  [ -d "$root" ] || die "remote root does not exist: $root"
  [ ! -L "$root" ] || die "remote root cannot be a symbolic link"
  root_real=$(cd "$root" && pwd -P)
  [ "$root_real" = "$root" ] || die "remote root must be canonical: $root"
  current=$root
  for component in custom public assets; do
    current="$current/$component"
    if [ -e "$current" ] || [ -L "$current" ]; then
      [ -d "$current" ] && [ ! -L "$current" ] || die "mount ancestor is not a real directory: $current"
    else
      break
    fi
  done
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
  local tree=$1 manifest=$2 entrypoint=$3 work line hash separator path
  local actual_hash duplicate directory parent
  work=$(mktemp -d "${TMPDIR:-/tmp}/verify-content.XXXXXX")
  validate_inode_types "$tree"
  [ -f "$manifest" ] && [ ! -L "$manifest" ] || { rm -rf "$work"; die "manifest is not a regular file"; }

  : > "$work/expected"
  while IFS= read -r line || [ -n "$line" ]; do
    [ -n "$line" ] || { rm -rf "$work"; die "blank manifest line"; }
    [ "${#line}" -ge 67 ] || { rm -rf "$work"; die "malformed manifest line"; }
    hash=${line:0:64}
    separator=${line:64:2}
    path=${line:66}
    [[ "$hash" =~ ^[0-9a-f]{64}$ ]] || { rm -rf "$work"; die "invalid manifest digest"; }
    [ "$separator" = "  " ] || { rm -rf "$work"; die "manifest requires two spaces before each path"; }
    validate_manifest_path "$path"
    [ -f "$tree/$path" ] && [ ! -L "$tree/$path" ] || { rm -rf "$work"; die "manifest file is not regular: $path"; }
    actual_hash=$(sha256_file "$tree/$path")
    [ "$actual_hash" = "$hash" ] || { rm -rf "$work"; die "checksum mismatch: $path"; }
    printf '%s\n' "$path" >> "$work/expected"
  done < "$manifest"

  [ -s "$work/expected" ] || { rm -rf "$work"; die "manifest is empty"; }
  [ -s "$tree/$entrypoint" ] || { rm -rf "$work"; die "entrypoint is missing or empty"; }
  LC_ALL=C sort "$work/expected" > "$work/expected.sorted"
  duplicate=$(uniq -d "$work/expected.sorted" | head -1 || true)
  [ -z "$duplicate" ] || { rm -rf "$work"; die "duplicate manifest path: $duplicate"; }
  (
    cd "$tree"
    find . -type f -print | sed 's#^\./##' | LC_ALL=C sort
  ) > "$work/actual.sorted"
  if ! cmp -s "$work/expected.sorted" "$work/actual.sorted"; then
    echo "FATAL: manifest does not exactly cover the content tree" >&2
    diff -u "$work/expected.sorted" "$work/actual.sorted" >&2 || true
    rm -rf "$work"
    exit 1
  fi
  : > "$work/expected-dirs"
  printf '.\n' >> "$work/expected-dirs"
  while IFS= read -r path || [ -n "$path" ]; do
    directory=$(dirname "$path")
    while [ "$directory" != . ]; do
      printf '%s\n' "$directory" >> "$work/expected-dirs"
      parent=$(dirname "$directory")
      [ "$parent" != "$directory" ] || { rm -rf "$work"; die "could not normalize manifest directory"; }
      directory=$parent
    done
  done < "$work/expected.sorted"
  LC_ALL=C sort -u "$work/expected-dirs" > "$work/expected-dirs.sorted"
  (
    cd "$tree"
    find . -type d -print | sed 's#^\./##' | LC_ALL=C sort
  ) > "$work/actual-dirs.sorted"
  if ! cmp -s "$work/expected-dirs.sorted" "$work/actual-dirs.sorted"; then
    echo "FATAL: content tree has directories not implied by the manifest" >&2
    diff -u "$work/expected-dirs.sorted" "$work/actual-dirs.sorted" >&2 || true
    rm -rf "$work"
    exit 1
  fi
  rm -rf "$work"
}

generate_tree_manifest() {
  local tree=$1 output=$2 work path
  work=$(mktemp -d "${TMPDIR:-/tmp}/snapshot-manifest.XXXXXX")
  validate_inode_types "$tree"
  (
    cd "$tree"
    find . -type f -print | sed 's#^\./##' | LC_ALL=C sort
  ) > "$work/files"
  : > "$work/manifest"
  while IFS= read -r path || [ -n "$path" ]; do
    validate_manifest_path "$path"
    printf '%s  %s\n' "$(sha256_file "$tree/$path")" "$path" >> "$work/manifest"
  done < "$work/files"
  mv "$work/manifest" "$output"
  rm -rf "$work"
}

validate_token() {
  [[ "$1" =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid lock token"
}

assert_lock() {
  local root=$1 token=$2 lock
  validate_token "$token"
  lock="$root/.content-deploy.lock"
  [ -d "$lock" ] && [ ! -L "$lock" ] || die "content deployment lock is absent"
  [ -f "$lock/token" ] && [ ! -L "$lock/token" ] || die "content deployment lock token is absent"
  [ "$(cat "$lock/token")" = "$token" ] || die "content deployment lock token does not match"
}

release_lock() {
  local root=$1 token=$2 helper=$3 helper_hash=$4 lock
  assert_lock "$root" "$token"
  verify_durability_helper "$helper" "$helper_hash"
  lock="$root/.content-deploy.lock"
  rm -f "$lock/token"
  durable_paths "$helper" "$lock"
  rmdir "$lock"
  durable_paths "$helper" "$root"
}

exchange_paths() {
  local mode=$1 helper=$2 left=$3 right=$4 temporary
  [ -d "$left" ] && [ ! -L "$left" ] || return 1
  [ -d "$right" ] && [ ! -L "$right" ] || return 1
  [ "$(device_id "$left")" = "$(device_id "$right")" ] || return 1
  if [ "$mode" = "exchange" ]; then
    python3 "$helper" exchange "$left" "$right"
    return
  fi
  [ "$mode" = "local-test" ] || return 1
  temporary="${left}.local-test-exchange.$$"
  [ ! -e "$temporary" ] || return 1
  mv "$left" "$temporary" || return 1
  if ! mv "$right" "$left"; then
    mv "$temporary" "$left" >/dev/null 2>&1 || true
    return 1
  fi
  if ! mv "$temporary" "$right"; then
    return 1
  fi
  if [ "${CONTENT_TEST_EXCHANGE_FAIL_AFTER_SWAP:-0}" = 1 ]; then
    return 70
  fi
}

inspect() {
  [ "$#" -eq 3 ] || die "inspect expects ROOT MOUNT ENTRYPOINT"
  validate_root_and_mount "$1" "$2" "$3"
  root=$1
  live="$root/$2"
  marker="$root/.last-content-deploy"
  if [ -L "$live" ]; then
    die "live content path cannot be a symbolic link"
  elif [ -e "$live" ] && [ ! -d "$live" ]; then
    die "live content path exists but is not a directory"
  elif [ -d "$live" ]; then
    validate_inode_types "$live"
    [ -s "$live/$3" ] || die "live content entrypoint is missing or empty"
    echo "live_exists=1"
    echo "live_files=$(find "$live" -type f | wc -l | tr -d ' ')"
  else
    echo "live_exists=0"
    echo "live_files=0"
  fi
  if [ -e "$marker" ]; then
    [ -f "$marker" ] && [ ! -L "$marker" ] || die "content marker is not a regular file"
    echo "marker_present=1"
    echo "marker_sha256=$(sha256_file "$marker")"
  else
    echo "marker_present=0"
    echo "marker_sha256=none"
  fi
}

lock_acquire() {
  [ "$#" -eq 8 ] || die "lock-acquire expects 8 arguments"
  root=$1
  mount_path=$2
  entrypoint=$3
  token=$4
  activation_mode=$5
  exchange_helper=$6
  exchange_helper_hash=$7
  allow_initial=$8
  validate_root_and_mount "$root" "$mount_path" "$entrypoint"
  validate_token "$token"
  [ "$allow_initial" = true ] || [ "$allow_initial" = false ] || die "invalid initial-install setting"
  verify_durability_helper "$exchange_helper" "$exchange_helper_hash"
  lock="$root/.content-deploy.lock"
  if [ -e "$lock" ] || [ -L "$lock" ]; then
    echo "FATAL: deployment lock already exists: $lock" >&2
    echo "       inspect $root/.content-backups/*/PHASE and MANUAL_RECOVERY before removing it" >&2
    exit 1
  fi
  backup_root="$root/.content-backups"
  if [ -d "$backup_root" ] && [ ! -L "$backup_root" ]; then
    for transaction in "$backup_root"/*; do
      [ -d "$transaction" ] || continue
      phase_file="$transaction/PHASE"
      if [ ! -f "$phase_file" ] || [ -L "$phase_file" ]; then
        die "stale transaction without a durable phase journal: $transaction"
      fi
      phase=$(sed -n 's/^phase=//p' "$phase_file" | head -1)
      case "$phase" in
        completed|rolled-back|aborted) ;;
        *) die "stale incomplete transaction ($phase): $transaction; inspect PHASE/MANUAL_RECOVERY" ;;
      esac
    done
  elif [ -e "$backup_root" ] || [ -L "$backup_root" ]; then
    die "remote backup root is not a real directory"
  fi
  if ! mkdir "$lock" 2>/dev/null; then
    die "another content deployment holds $lock"
  fi
  printf '%s\n' "$token" > "$lock/token"
  durable_paths "$exchange_helper" "$lock/token" "$lock" "$root"

  live="$root/$mount_path"
  live_parent=$(dirname "$live")
  probe_parent=$root
  [ -d "$live_parent" ] && probe_parent=$live_parent
  if [ "$activation_mode" = exchange ]; then
    if [ "$(uname -s)" != Linux ] || ! command -v python3 >/dev/null 2>&1 \
       || [ ! -f "$exchange_helper" ] || [ -L "$exchange_helper" ] \
       || [ "$(sha256_file "$exchange_helper")" != "$exchange_helper_hash" ] \
       || ! python3 "$exchange_helper" probe "$probe_parent"; then
      release_lock "$root" "$token" "$exchange_helper" "$exchange_helper_hash" >/dev/null 2>&1 || true
      die "Linux renameat2(RENAME_EXCHANGE) capability probe failed"
    fi
  elif [ "$activation_mode" != local-test ]; then
    release_lock "$root" "$token" "$exchange_helper" "$exchange_helper_hash" >/dev/null 2>&1 || true
    die "unsupported activation mode"
  fi
  echo "lock_token=$token"
  echo "activation_mode=$activation_mode"
}

snapshot() {
  [ "$#" -eq 8 ] || die "snapshot expects 8 arguments"
  root=$1
  mount_path=$2
  entrypoint=$3
  token=$4
  backup_id=$5
  allow_initial=$6
  durability_helper=$7
  durability_helper_hash=$8
  validate_root_and_mount "$root" "$mount_path" "$entrypoint"
  assert_lock "$root" "$token"
  verify_durability_helper "$durability_helper" "$durability_helper_hash"
  [[ "$backup_id" =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid backup id"
  live="$root/$mount_path"
  live_parent=$(dirname "$live")
  leaf=$(basename "$live")
  marker="$root/.last-content-deploy"
  backup_root="$root/.content-backups"
  backup="$backup_root/$backup_id"
  if [ -e "$backup_root" ] || [ -L "$backup_root" ]; then
    [ -d "$backup_root" ] && [ ! -L "$backup_root" ] || die "remote backup root is not a real directory"
  else
    mkdir "$backup_root"
  fi
  [ ! -e "$backup" ] || die "remote backup already exists: $backup"
  mkdir "$backup"
  [ "$(device_id "$root")" = "$(device_id "$backup_root")" ] || die "remote backup must share the live filesystem"

  if [ -L "$live" ]; then
    die "live content path cannot be a symbolic link"
  elif [ -e "$live" ] && [ ! -d "$live" ]; then
    die "live content path exists but is not a directory"
  elif [ -d "$live" ]; then
    validate_inode_types "$live"
    [ -s "$live/$entrypoint" ] || die "live content entrypoint is missing or empty"
    generate_tree_manifest "$live" "$backup/previous.SHA256SUMS"
    COPYFILE_DISABLE=1 tar -cf "$backup/live.tar" -C "$live_parent" "$leaf"
    check=$(mktemp -d "$backup/.snapshot-check.XXXXXX")
    COPYFILE_DISABLE=1 tar -xf "$backup/live.tar" -C "$check"
    verify_tree "$check/$leaf" "$backup/previous.SHA256SUMS" "$entrypoint"
    rm -rf "$check"
    echo "live_exists=1"
    echo "backup_archive=$backup/live.tar"
    echo "backup_archive_sha256=$(sha256_file "$backup/live.tar")"
    echo "previous_manifest=$backup/previous.SHA256SUMS"
    echo "previous_manifest_sha256=$(sha256_file "$backup/previous.SHA256SUMS")"
  else
    [ "$allow_initial" = true ] || die "live content is absent and initial installation is disabled"
    : > "$backup/live-was-absent"
    echo "live_exists=0"
    echo "backup_archive=absent"
    echo "backup_archive_sha256=absent"
    echo "previous_manifest=absent"
    echo "previous_manifest_sha256=absent"
  fi

  if [ -e "$marker" ]; then
    [ -f "$marker" ] && [ ! -L "$marker" ] || die "content marker is not a regular file"
    cp "$marker" "$backup/previous-marker"
    echo "previous_marker=$backup/previous-marker"
    echo "previous_marker_sha256=$(sha256_file "$backup/previous-marker")"
  else
    : > "$backup/marker-was-absent"
    echo "previous_marker=absent"
    echo "previous_marker_sha256=absent"
  fi
  printf 'backup_id=%s\nlock_token=%s\ncreated_at=%s\n' \
    "$backup_id" "$token" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$backup/snapshot"
  durable_tree "$durability_helper" "$backup"
  durable_paths "$durability_helper" "$backup" "$backup_root" "$root"
  write_phase "$backup" "$durability_helper" snapshot-created \
    "lock_token=$token" "live_exists=$([ -f "$backup/live-was-absent" ] && echo 0 || echo 1)"
  echo "backup_dir=$backup"
}

backup_ack() {
  [ "$#" -eq 8 ] || die "backup-ack expects ROOT TOKEN BACKUP_ID ARCHIVE_HASH MANIFEST_HASH MARKER_HASH HELPER HELPER_HASH"
  root=$1
  token=$2
  backup_id=$3
  expected_hash=$4
  expected_manifest_hash=$5
  expected_marker_hash=$6
  durability_helper=$7
  durability_helper_hash=$8
  validate_root_and_mount "$root" custom/public/assets/landing index.html
  assert_lock "$root" "$token"
  verify_durability_helper "$durability_helper" "$durability_helper_hash"
  [[ "$backup_id" =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid backup id"
  backup="$root/.content-backups/$backup_id"
  [ -d "$backup" ] || die "authoritative remote backup is absent"
  if [ "$expected_hash" = absent ]; then
    [ -f "$backup/live-was-absent" ] || die "initial-install backup state does not match"
    [ "$expected_manifest_hash" = absent ] || die "initial-install manifest state does not match"
  else
    [[ "$expected_hash" =~ ^[0-9a-f]{64}$ ]] || die "invalid backup hash"
    [ -f "$backup/live.tar" ] && [ "$(sha256_file "$backup/live.tar")" = "$expected_hash" ] \
      || die "authoritative remote backup hash changed"
    [[ "$expected_manifest_hash" =~ ^[0-9a-f]{64}$ ]] || die "invalid previous manifest hash"
    [ -f "$backup/previous.SHA256SUMS" ] \
      && [ "$(sha256_file "$backup/previous.SHA256SUMS")" = "$expected_manifest_hash" ] \
      || die "authoritative previous manifest hash changed"
  fi
  if [ "$expected_marker_hash" = absent ]; then
    [ -f "$backup/marker-was-absent" ] || die "previous marker absence state changed"
  else
    [[ "$expected_marker_hash" =~ ^[0-9a-f]{64}$ ]] || die "invalid previous marker hash"
    [ -f "$backup/previous-marker" ] \
      && [ "$(sha256_file "$backup/previous-marker")" = "$expected_marker_hash" ] \
      || die "authoritative previous marker hash changed"
  fi
  printf 'archive_sha256=%s\nmanifest_sha256=%s\nmarker_sha256=%s\nvalidated_at=%s\n' \
    "$expected_hash" "$expected_manifest_hash" "$expected_marker_hash" \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$backup/local-backup-validated"
  durable_paths "$durability_helper" "$backup/local-backup-validated"
  write_phase "$backup" "$durability_helper" local-backup-validated \
    "archive_sha256=$expected_hash" "manifest_sha256=$expected_manifest_hash" "marker_sha256=$expected_marker_hash"
  echo "local_backup_ack=$expected_hash"
}

abort_transaction() {
  [ "$#" -eq 6 ] || die "abort expects ROOT TOKEN BACKUP_ID HELPER HELPER_HASH REASON"
  root=$1
  token=$2
  backup_id=$3
  durability_helper=$4
  durability_helper_hash=$5
  reason=$6
  validate_root_and_mount "$root" custom/public/assets/landing index.html
  assert_lock "$root" "$token"
  verify_durability_helper "$durability_helper" "$durability_helper_hash"
  [[ "$backup_id" =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid backup id"
  [[ "$reason" =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid abort reason"
  backup="$root/.content-backups/$backup_id"
  if [ -d "$backup" ] && [ ! -L "$backup" ]; then
    write_phase "$backup" "$durability_helper" aborted "reason=$reason"
  elif [ -e "$backup" ] || [ -L "$backup" ]; then
    die "remote backup is not a real directory"
  fi
  release_lock "$root" "$token" "$durability_helper" "$durability_helper_hash"
}

apply_transaction() {
  [ "$#" -eq 20 ] || die "apply expects 20 arguments"
  root=$1
  mount_path=$2
  entrypoint=$3
  archive=$4
  manifest=$5
  expected_archive_hash=$6
  expected_manifest_hash=$7
  expected_entrypoint_hash=$8
  smoke_asset=$9
  expected_asset_hash=${10}
  release_id=${11}
  source_revision=${12}
  backup_id=${13}
  public_base_url=${14}
  smoke_mode=${15}
  allow_initial=${16}
  token=${17}
  activation_mode=${18}
  exchange_helper=${19}
  exchange_helper_hash=${20}

  validate_root_and_mount "$root" "$mount_path" "$entrypoint"
  assert_lock "$root" "$token"
  validate_manifest_path "$smoke_asset"
  [[ "$release_id" =~ ^[0-9a-f]{64}$ ]] || die "invalid release id"
  [[ "$source_revision" =~ ^([0-9a-f]{40,64}|unversioned)$ ]] || die "invalid source revision"
  [[ "$backup_id" =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid backup id"
  [ "$allow_initial" = true ] || [ "$allow_initial" = false ] || die "invalid initial-install setting"
  [ "$smoke_mode" = http ] || [ "$smoke_mode" = local ] || die "invalid smoke mode"
  [ "$activation_mode" = exchange ] || [ "$activation_mode" = local-test ] || die "invalid activation mode"

  live="$root/$mount_path"
  live_parent=$(dirname "$live")
  leaf=$(basename "$live")
  marker="$root/.last-content-deploy"
  backup="$root/.content-backups/$backup_id"
  stage="$live_parent/.${leaf}.staging-${backup_id}"
  marker_tmp=""
  smoke_index=""
  smoke_asset_file=""
  activated=0
  committed=0
  ambiguous=0
  had_live=0
  recovery_peer=$stage

  write_manual_recovery() {
    local manual_status=${1:-ROLLBACK_FAILED}
    mkdir -p "$backup" 2>/dev/null || true
    printf '%s\n' \
      "status=$manual_status" \
      "live=$live" \
      "staging_peer=$stage" \
      "backup_peer=$backup/live" \
      "failed_peer=$backup/failed-live" \
      "previous_manifest=$backup/previous.SHA256SUMS" \
      "lock=$root/.content-deploy.lock" \
      "recorded_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$backup/MANUAL_RECOVERY" 2>/dev/null || true
    durable_paths "$exchange_helper" "$backup/MANUAL_RECOVERY" "$backup" >/dev/null 2>&1 || true
    write_phase "$backup" "$exchange_helper" manual-recovery \
      "status=$manual_status" "live=$live" "staging_peer=$stage" \
      "backup_peer=$backup/live" >/dev/null 2>&1 || true
  }

  matches_incoming() {
    ( verify_tree "$1" "$manifest" "$entrypoint" ) >/dev/null 2>&1
  }

  matches_previous() {
    local candidate=$1
    if [ "$had_live" -eq 1 ]; then
      ( verify_tree "$candidate" "$backup/previous.SHA256SUMS" "$entrypoint" ) >/dev/null 2>&1
    else
      [ -d "$candidate" ] && [ -z "$(find "$candidate" -mindepth 1 -print -quit)" ]
    fi
  }

  restore_marker_verified() {
    if [ -f "$backup/previous-marker" ]; then
      cp "$backup/previous-marker" "$marker_tmp" || return 1
      durable_paths "$exchange_helper" "$marker_tmp" || return 1
      mv "$marker_tmp" "$marker" || return 1
      durable_paths "$exchange_helper" "$marker" "$root" || return 1
      [ "$(sha256_file "$marker")" = "$(sha256_file "$backup/previous-marker")" ] || return 1
    elif [ -f "$backup/marker-was-absent" ]; then
      rm -f "$marker" || return 1
      durable_paths "$exchange_helper" "$root" || return 1
      [ ! -e "$marker" ] || return 1
    else
      return 1
    fi
  }

  rollback_verified() {
    local exchange_result
    if [ "${CONTENT_TEST_FAIL_ROLLBACK:-0}" = 1 ]; then
      return 1
    fi
    if [ ! -d "$recovery_peer" ]; then
      if [ -d "$backup/live" ]; then
        recovery_peer="$backup/live"
      elif [ -d "$stage" ]; then
        recovery_peer=$stage
      else
        return 1
      fi
    fi
    set +e
    exchange_paths "$activation_mode" "$exchange_helper" "$live" "$recovery_peer"
    exchange_result=$?
    set -e
    if [ "$exchange_result" -ne 0 ]; then
      if matches_previous "$live" && matches_incoming "$recovery_peer"; then
        : # The exchange completed even though its helper reported an error.
      else
        return 1
      fi
    fi
    durable_paths "$exchange_helper" "$live_parent" || return 1
    if [ "$had_live" -eq 1 ]; then
      ( verify_tree "$live" "$backup/previous.SHA256SUMS" "$entrypoint" ) || return 1
    else
      [ -z "$(find "$live" -mindepth 1 -print -quit)" ] || return 1
      rmdir "$live" || return 1
      [ ! -e "$live" ] || return 1
    fi
    restore_marker_verified || return 1
    ( verify_tree "$recovery_peer" "$manifest" "$entrypoint" ) || return 1
    [ ! -e "$backup/failed-live" ] || return 1
    mv "$recovery_peer" "$backup/failed-live" || return 1
    [ -d "$backup/failed-live" ] || return 1
    return 0
  }

  finish() {
    rc=$?
    trap - EXIT INT TERM HUP
    set +e
    [ -n "$smoke_index" ] && rm -f "$smoke_index"
    [ -n "$smoke_asset_file" ] && rm -f "$smoke_asset_file"
    rm -f "$marker_tmp"
    if [ "$rc" -ne 0 ] && [ "$committed" -eq 1 ]; then
      write_manual_recovery COMMIT_FINALIZATION_FAILED
      echo "Content was committed but transaction finalization failed; further deploys are blocked. See $backup/MANUAL_RECOVERY" >&2
      exit 75
    fi
    if [ "$rc" -ne 0 ] && [ "$committed" -eq 0 ]; then
      if [ "$ambiguous" -eq 1 ]; then
        write_manual_recovery ATOMIC_EXCHANGE_AMBIGUOUS
        echo "ATOMIC EXCHANGE STATE AMBIGUOUS; lock retained. See $backup/MANUAL_RECOVERY" >&2
        exit 75
      fi
      if [ "$activated" -eq 1 ]; then
        if rollback_verified; then
          printf 'rolled_back_at=%s\nexit_code=%s\nverified=true\n' \
            "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$rc" > "$backup/ROLLBACK"
          durable_paths "$exchange_helper" "$backup/ROLLBACK"
          write_phase "$backup" "$exchange_helper" rolled-back "exit_code=$rc"
          if ( release_lock "$root" "$token" "$exchange_helper" "$exchange_helper_hash" ); then
            echo "Content activation failed; previous live content and marker were restored and verified." >&2
            exit "$rc"
          fi
        fi
        write_manual_recovery
        echo "ROLLBACK FAILED; lock retained. See $backup/MANUAL_RECOVERY" >&2
        exit 75
      fi
      if [ -d "$stage" ] && [ -d "$backup" ] && [ ! -e "$backup/failed-staging" ]; then
        if mv "$stage" "$backup/failed-staging" >/dev/null 2>&1; then
          durable_paths "$exchange_helper" "$live_parent" "$backup" >/dev/null 2>&1 || true
        fi
      fi
      write_phase "$backup" "$exchange_helper" aborted "exit_code=$rc" >/dev/null 2>&1 || true
      if ( release_lock "$root" "$token" "$exchange_helper" "$exchange_helper_hash" ); then
        exit "$rc"
      fi
      write_manual_recovery PREACTIVATION_CLEANUP_FAILED
      echo "Pre-activation failure could not release the lock; manual recovery required." >&2
      exit 75
    fi
    exit "$rc"
  }
  trap finish EXIT INT TERM HUP

  [ -d "$backup" ] || die "authoritative remote backup is absent"
  [ -f "$backup/local-backup-validated" ] || die "validated local backup acknowledgement is absent"
  verify_durability_helper "$exchange_helper" "$exchange_helper_hash"
  [ -f "$archive" ] && [ ! -L "$archive" ] || die "uploaded archive is not regular"
  [ -f "$manifest" ] && [ ! -L "$manifest" ] || die "uploaded manifest is not regular"
  [ "$(sha256_file "$archive")" = "$expected_archive_hash" ] || die "uploaded archive checksum mismatch"
  [ "$(sha256_file "$manifest")" = "$expected_manifest_hash" ] || die "uploaded manifest checksum mismatch"
  durable_paths "$exchange_helper" "$archive" "$manifest"
  cp "$manifest" "$backup/incoming.SHA256SUMS"
  if [ "$activation_mode" = exchange ]; then
    [ "$(uname -s)" = Linux ] && command -v python3 >/dev/null 2>&1 \
      && [ -f "$exchange_helper" ] && [ ! -L "$exchange_helper" ] \
      && [ "$(sha256_file "$exchange_helper")" = "$exchange_helper_hash" ] \
      || die "atomic exchange helper is unavailable or changed"
  fi
  marker_tmp=$(mktemp "$root/.last-content-deploy.tmp.XXXXXX")

  mkdir -p "$live_parent"
  [ ! -e "$stage" ] || die "staging path already exists: $stage"
  mkdir "$stage"
  COPYFILE_DISABLE=1 tar -xf "$archive" -C "$stage"
  verify_tree "$stage" "$manifest" "$entrypoint"
  [ "$(sha256_file "$stage/$entrypoint")" = "$expected_entrypoint_hash" ] || die "entrypoint digest argument does not match"
  [ -f "$stage/$smoke_asset" ] && [ ! -L "$stage/$smoke_asset" ] || die "smoke asset is missing"
  [ "$(sha256_file "$stage/$smoke_asset")" = "$expected_asset_hash" ] || die "smoke asset digest argument does not match"
  find "$stage" -type d -exec chmod 755 {} +
  find "$stage" -type f -exec chmod 644 {} +
  durable_tree "$exchange_helper" "$stage"
  durable_paths "$exchange_helper" "$live_parent"
  write_phase "$backup" "$exchange_helper" staging-durable \
    "stage=$stage" "archive_sha256=$expected_archive_hash"
  [ "$(device_id "$live_parent")" = "$(device_id "$backup")" ] || die "live, staging, and backup must share one filesystem"

  if [ -d "$live" ] && [ ! -L "$live" ]; then
    had_live=1
    verify_tree "$live" "$backup/previous.SHA256SUMS" "$entrypoint"
  elif [ ! -e "$live" ] && [ "$allow_initial" = true ] && [ -f "$backup/live-was-absent" ]; then
    mkdir "$live"
  else
    die "live content state changed after the authoritative snapshot"
  fi

  set +e
  exchange_paths "$activation_mode" "$exchange_helper" "$live" "$stage"
  exchange_result=$?
  set -e
  recovery_peer=$stage
  if [ "$exchange_result" -eq 0 ]; then
    activated=1
  elif matches_incoming "$live" && matches_previous "$stage"; then
    activated=1
    durable_paths "$exchange_helper" "$live_parent"
    write_phase "$backup" "$exchange_helper" activated \
      "exchange_reported_error=$exchange_result" "live=$live" "recovery_peer=$stage"
    die "atomic exchange completed but helper returned $exchange_result; rolling back"
  elif matches_previous "$live" && matches_incoming "$stage"; then
    die "atomic exchange failed before changing live (exit $exchange_result)"
  else
    ambiguous=1
    die "atomic exchange returned $exchange_result and resulting paths are ambiguous"
  fi
  durable_paths "$exchange_helper" "$live_parent"
  write_phase "$backup" "$exchange_helper" activated \
    "exchange_reported_error=0" "live=$live" "recovery_peer=$stage"
  verify_tree "$live" "$manifest" "$entrypoint"
  if [ "${CONTENT_TEST_FAIL_AFTER_SWAP:-0}" = 1 ]; then
    die "injected post-swap failure"
  fi

  if [ "$smoke_mode" = http ]; then
    case "$public_base_url" in http://*|https://*) ;; *) die "public base URL must use http or https" ;; esac
    smoke_index=$(mktemp "${TMPDIR:-/tmp}/content-smoke-index.XXXXXX")
    smoke_asset_file=$(mktemp "${TMPDIR:-/tmp}/content-smoke-asset.XXXXXX")
    curl -fsSL --max-time 30 -H 'Accept-Encoding: identity' "${public_base_url%/}/" -o "$smoke_index"
    [ "$(sha256_file "$smoke_index")" = "$expected_entrypoint_hash" ] || die "public entrypoint checksum mismatch"
    curl -fsSL --max-time 30 -H 'Accept-Encoding: identity' \
      "${public_base_url%/}/assets/landing/$smoke_asset" -o "$smoke_asset_file"
    [ "$(sha256_file "$smoke_asset_file")" = "$expected_asset_hash" ] || die "public smoke asset checksum mismatch"
  fi
  write_phase "$backup" "$exchange_helper" smoke-verified \
    "entrypoint_sha256=$expected_entrypoint_hash" "asset=$smoke_asset" "asset_sha256=$expected_asset_hash"

  [ ! -e "$backup/live" ] || die "remote recovery directory already exists"
  mv "$stage" "$backup/live"
  recovery_peer="$backup/live"
  durable_paths "$exchange_helper" "$live_parent" "$backup"
  if [ "$had_live" -eq 1 ]; then
    verify_tree "$backup/live" "$backup/previous.SHA256SUMS" "$entrypoint"
  else
    [ -z "$(find "$backup/live" -mindepth 1 -print -quit)" ] || die "initial-install recovery placeholder is not empty"
  fi
  write_phase "$backup" "$exchange_helper" old-live-backed-up "recovery_peer=$recovery_peer"

  printf '%s\n' \
    version=1 \
    "mount=$mount_path" \
    "release_id=$release_id" \
    "source_revision=$source_revision" \
    "manifest_sha256=$expected_manifest_hash" \
    "backup_id=$backup_id" \
    "deployed_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$marker_tmp"
  durable_paths "$exchange_helper" "$marker_tmp"
  mv "$marker_tmp" "$marker"
  durable_paths "$exchange_helper" "$marker" "$root"
  [ -f "$marker" ] && grep -q "^release_id=$release_id$" "$marker" || die "new content marker verification failed"
  write_phase "$backup" "$exchange_helper" marker-committed "marker=$marker"
  committed=1
  write_phase "$backup" "$exchange_helper" completed "release_id=$release_id"
  if ! ( release_lock "$root" "$token" "$exchange_helper" "$exchange_helper_hash" ); then
    die "content committed but durable lock release failed"
  fi
  echo "release_id=$release_id"
  echo "backup_id=$backup_id"
  echo "marker=$marker"
}

ACTION=${1:-}
shift || true
case "$ACTION" in
  upload-create) upload_create "$@" ;;
  upload-verify) upload_verify "$@" ;;
  upload-clean) upload_clean "$@" ;;
  inspect) inspect "$@" ;;
  lock-acquire) lock_acquire "$@" ;;
  snapshot) snapshot "$@" ;;
  backup-ack) backup_ack "$@" ;;
  abort) abort_transaction "$@" ;;
  lock-release) [ "$#" -eq 4 ] || die "lock-release expects ROOT TOKEN HELPER HELPER_HASH"; release_lock "$1" "$2" "$3" "$4" ;;
  apply) apply_transaction "$@" ;;
  *) die "unknown transaction action: $ACTION" ;;
esac
