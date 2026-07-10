#!/usr/bin/env bash

set -euo pipefail

CONTENT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
PUBLISH="$CONTENT_DIR/publish-overlay.sh"
GENERATE="$CONTENT_DIR/generate-manifest.sh"
REMOTE_HELPER="$CONTENT_DIR/remote-transaction.sh"
EXCHANGE_HELPER="$CONTENT_DIR/rename-exchange.py"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/content-publisher-tests.XXXXXX")
TMP=$(cd "$TMP" && pwd -P)
trap 'rm -rf "$TMP"' EXIT INT TERM HUP

PASS=0

ok() {
  PASS=$((PASS + 1))
  echo "ok $PASS - $*"
}

fail() {
  echo "not ok - $*" >&2
  exit 1
}

assert_file_text() {
  file=$1
  expected=$2
  actual=$(cat "$file")
  [ "$actual" = "$expected" ] || fail "$file: expected '$expected', got '$actual'"
}

expect_failure() {
  if "$@" >"$TMP/expected-failure.out" 2>&1; then
    cat "$TMP/expected-failure.out" >&2
    fail "command unexpectedly succeeded: $*"
  fi
}

write_config() {
  config=$1
  remote_root=$2
  backup_root=$3
  allow_initial=${4:-false}
  require_clean=${5:-false}
  smoke_asset=${6:-assets/site.css}
  {
    echo 'TRANSPORT=local'
    echo 'DEPLOY_TARGET=local-fixture'
    echo 'PUBLIC_BASE_URL=http://local.invalid'
    echo "REMOTE_ROOT=$remote_root"
    echo "LOCAL_BACKUP_ROOT=$backup_root"
    echo "SMOKE_ASSET=$smoke_asset"
    echo "ALLOW_INITIAL_INSTALL=$allow_initial"
    echo "REQUIRE_CLEAN_SOURCE=$require_clean"
  } > "$config"
}

test_sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

file_mode() {
  if stat -c '%a' "$1" >/dev/null 2>&1; then
    stat -c '%a' "$1"
  else
    stat -f '%Lp' "$1"
  fi
}

write_manifest_unchecked() {
  source_dir=$1
  output=$2
  (
    cd "$source_dir"
    find . -type f -print | sed 's#^\./##' | LC_ALL=C sort
  ) > "$output.paths"
  : > "$output"
  while IFS= read -r path || [ -n "$path" ]; do
    printf '%s  %s\n' "$(test_sha256 "$source_dir/$path")" "$path" >> "$output"
  done < "$output.paths"
  rm -f "$output.paths"
}

init_pushed_repo() {
  repo=$1
  bare=$2
  git init -q "$repo"
  git -C "$repo" checkout -q -b main
  git -C "$repo" config user.name 'Content Test'
  git -C "$repo" config user.email content-test@example.invalid
  git init -q --bare "$bare"
  git -C "$repo" remote add origin "$bare"
}

commit_and_push() {
  repo=$1
  message=$2
  git -C "$repo" add -A
  git -C "$repo" commit -q -m "$message"
  if git -C "$repo" rev-parse --verify '@{upstream}' >/dev/null 2>&1; then
    git -C "$repo" push -q
  else
    git -C "$repo" push -q -u origin main
  fi
}

make_source() {
  source_dir=$1
  label=$2
  mkdir -p "$source_dir/assets"
  printf '%s\n' "$label" > "$source_dir/index.html"
  printf '%s\n' "$label-asset" > "$source_dir/assets/site.css"
  printf '%s\n' "$label-image" > "$source_dir/assets/banner 中文 name.webp"
}

echo '1..24'

# 1. Default invocation is a read-only plan.
CASE1="$TMP/case1"
REMOTE1="$CASE1/remote"
LIVE1="$REMOTE1/custom/public/assets/landing"
SOURCE1="$CASE1/source"
BACKUPS1="$CASE1/local-backups"
mkdir -p "$LIVE1"
printf 'old-live\n' > "$LIVE1/index.html"
printf 'obsolete\n' > "$LIVE1/obsolete.txt"
make_source "$SOURCE1" new-live
bash "$GENERATE" "$SOURCE1" "$CASE1/SHA256SUMS" >/dev/null
write_config "$CASE1/config" "$REMOTE1" "$BACKUPS1"
DRY_OUTPUT=$(bash "$PUBLISH" --config "$CASE1/config" --mount landing --source "$SOURCE1" --manifest "$CASE1/SHA256SUMS")
printf '%s\n' "$DRY_OUTPUT" | grep -q 'DRY RUN' || fail 'dry-run marker missing'
assert_file_text "$LIVE1/index.html" old-live
[ ! -e "$REMOTE1/.last-content-deploy" ] || fail 'dry run wrote remote marker'
[ ! -e "$REMOTE1/.content-backups" ] || fail 'dry run created remote backup'
[ ! -e "$BACKUPS1" ] || fail 'dry run created local backup'
ok 'default invocation is read-only'

# 2. Apply swaps the exact tree and creates both backups and a marker.
bash "$PUBLISH" --config "$CASE1/config" --mount landing --source "$SOURCE1" --manifest "$CASE1/SHA256SUMS" --apply >/dev/null
assert_file_text "$LIVE1/index.html" new-live
[ ! -e "$LIVE1/obsolete.txt" ] || fail 'old unlisted file survived exact directory swap'
[ -f "$REMOTE1/.last-content-deploy" ] || fail 'content marker missing'
BACKUP_ID=$(sed -n 's/^backup_id=//p' "$REMOTE1/.last-content-deploy")
[ -n "$BACKUP_ID" ] || fail 'marker lacks backup id'
assert_file_text "$REMOTE1/.content-backups/$BACKUP_ID/live/index.html" old-live
[ -f "$BACKUPS1/$BACKUP_ID/live.tar" ] || fail 'local live tar backup missing'
tar -tf "$BACKUPS1/$BACKUP_ID/live.tar" >/dev/null || fail 'local live tar is invalid'
OLD_FROM_TAR=$(tar -xOf "$BACKUPS1/$BACKUP_ID/live.tar" landing/index.html)
[ "$OLD_FROM_TAR" = old-live ] || fail 'local backup does not contain prior live content'
grep -q '^phase=completed$' "$REMOTE1/.content-backups/$BACKUP_ID/PHASE" || fail 'completed deployment lacks a durable terminal phase'
[ ! -e "$REMOTE1/.content-deploy.lock" ] || fail 'completed deployment retained its lock'
ok 'apply performs exact swap with local and remote backups'

# 3. A modified file invalidates the manifest before target mutation.
CASE3="$TMP/case3"
REMOTE3="$CASE3/remote"
LIVE3="$REMOTE3/custom/public/assets/landing"
SOURCE3="$CASE3/source"
mkdir -p "$LIVE3"
printf 'stable\n' > "$LIVE3/index.html"
make_source "$SOURCE3" candidate
bash "$GENERATE" "$SOURCE3" "$CASE3/SHA256SUMS" >/dev/null
printf 'tampered\n' > "$SOURCE3/index.html"
write_config "$CASE3/config" "$REMOTE3" "$CASE3/backups"
expect_failure bash "$PUBLISH" --config "$CASE3/config" --mount landing --source "$SOURCE3" --manifest "$CASE3/SHA256SUMS"
assert_file_text "$LIVE3/index.html" stable
ok 'checksum mismatch fails closed'

# 4. An unlisted extra file invalidates exact coverage.
CASE4="$TMP/case4"
REMOTE4="$CASE4/remote"
LIVE4="$REMOTE4/custom/public/assets/landing"
SOURCE4="$CASE4/source"
mkdir -p "$LIVE4"
printf 'stable\n' > "$LIVE4/index.html"
make_source "$SOURCE4" candidate
bash "$GENERATE" "$SOURCE4" "$CASE4/SHA256SUMS" >/dev/null
printf 'extra\n' > "$SOURCE4/not-in-manifest.txt"
write_config "$CASE4/config" "$REMOTE4" "$CASE4/backups"
expect_failure bash "$PUBLISH" --config "$CASE4/config" --mount landing --source "$SOURCE4" --manifest "$CASE4/SHA256SUMS"
assert_file_text "$LIVE4/index.html" stable
ok 'extra source file fails exact manifest coverage'

# 5. Symlinked content is rejected.
CASE5="$TMP/case5"
REMOTE5="$CASE5/remote"
LIVE5="$REMOTE5/custom/public/assets/landing"
SOURCE5="$CASE5/source"
mkdir -p "$LIVE5"
printf 'stable\n' > "$LIVE5/index.html"
make_source "$SOURCE5" candidate
bash "$GENERATE" "$SOURCE5" "$CASE5/SHA256SUMS" >/dev/null
ln -s index.html "$SOURCE5/linked.html"
write_config "$CASE5/config" "$REMOTE5" "$CASE5/backups"
expect_failure bash "$PUBLISH" --config "$CASE5/config" --mount landing --source "$SOURCE5" --manifest "$CASE5/SHA256SUMS"
assert_file_text "$LIVE5/index.html" stable
ok 'symbolic links fail closed'

# 6. Missing config coordinates fail before inspection.
CASE6="$TMP/case6"
mkdir -p "$CASE6/remote/custom/public/assets/landing"
printf 'stable\n' > "$CASE6/remote/custom/public/assets/landing/index.html"
make_source "$CASE6/source" candidate
bash "$GENERATE" "$CASE6/source" "$CASE6/SHA256SUMS" >/dev/null
{
  echo 'TRANSPORT=local'
  echo 'DEPLOY_TARGET=local-fixture'
  echo "REMOTE_ROOT=$CASE6/remote"
  echo "LOCAL_BACKUP_ROOT=$CASE6/backups"
  echo 'SMOKE_ASSET=assets/site.css'
  echo 'REQUIRE_CLEAN_SOURCE=false'
} > "$CASE6/config"
expect_failure bash "$PUBLISH" --config "$CASE6/config" --mount landing --source "$CASE6/source" --manifest "$CASE6/SHA256SUMS"
ok 'missing public URL in config fails closed'

# 7. A post-swap failure restores both content and marker.
CASE7="$TMP/case7"
REMOTE7="$CASE7/remote"
LIVE7="$REMOTE7/custom/public/assets/landing"
SOURCE7="$CASE7/source"
mkdir -p "$LIVE7"
printf 'before-failure\n' > "$LIVE7/index.html"
printf 'version=1\nrelease_id=previous\n' > "$REMOTE7/.last-content-deploy"
make_source "$SOURCE7" after-failure
bash "$GENERATE" "$SOURCE7" "$CASE7/SHA256SUMS" >/dev/null
write_config "$CASE7/config" "$REMOTE7" "$CASE7/backups"
if CONTENT_TEST_FAIL_AFTER_SWAP=1 bash "$PUBLISH" --config "$CASE7/config" --mount landing --source "$SOURCE7" --manifest "$CASE7/SHA256SUMS" --apply >"$CASE7/output" 2>&1; then
  cat "$CASE7/output" >&2
  fail 'injected post-swap failure unexpectedly succeeded'
fi
assert_file_text "$LIVE7/index.html" before-failure
grep -q '^release_id=previous$' "$REMOTE7/.last-content-deploy" || fail 'previous marker was not restored'
FAILED_NEW=$(find "$REMOTE7/.content-backups" -path '*/failed-live/index.html' -print -quit)
[ -n "$FAILED_NEW" ] || fail 'failed replacement tree was not preserved'
assert_file_text "$FAILED_NEW" after-failure
BACKUP7=$(dirname "$(dirname "$FAILED_NEW")")
grep -q '^phase=rolled-back$' "$BACKUP7/PHASE" || fail 'verified rollback lacks a durable terminal phase'
ok 'post-swap failure automatically restores prior live state'

# 8. Initial installation is denied unless explicitly enabled.
CASE8="$TMP/case8"
REMOTE8="$CASE8/remote"
SOURCE8="$CASE8/source"
mkdir -p "$REMOTE8"
make_source "$SOURCE8" initial
bash "$GENERATE" "$SOURCE8" "$CASE8/SHA256SUMS" >/dev/null
write_config "$CASE8/config" "$REMOTE8" "$CASE8/backups" false
expect_failure bash "$PUBLISH" --config "$CASE8/config" --mount landing --source "$SOURCE8" --manifest "$CASE8/SHA256SUMS" --apply
[ ! -e "$REMOTE8/custom/public/assets/landing" ] || fail 'denied initial install created live content'
ok 'initial installation requires explicit config approval'

# 9. FIFO and other special inodes are rejected in source and live trees.
CASE9="$TMP/case9"
REMOTE9="$CASE9/remote"
LIVE9="$REMOTE9/custom/public/assets/landing"
SOURCE9="$CASE9/source"
mkdir -p "$LIVE9"
printf 'stable\n' > "$LIVE9/index.html"
make_source "$SOURCE9" fifo-candidate
bash "$GENERATE" "$SOURCE9" "$CASE9/SHA256SUMS" >/dev/null
mkfifo "$SOURCE9/assets/pipe"
write_config "$CASE9/config" "$REMOTE9" "$CASE9/backups"
expect_failure bash "$PUBLISH" --config "$CASE9/config" --mount landing --source "$SOURCE9" --manifest "$CASE9/SHA256SUMS"
rm "$SOURCE9/assets/pipe"
mkfifo "$LIVE9/live-pipe"
expect_failure bash "$PUBLISH" --config "$CASE9/config" --mount landing --source "$SOURCE9" --manifest "$CASE9/SHA256SUMS"
ok 'source and remote FIFO inodes fail closed'

# 10. Strict provenance accepts a clean, pushed, same-repository source.
CASE10="$TMP/case10"
REPO10="$CASE10/private"
BARE10="$CASE10/origin.git"
mkdir -p "$CASE10"
init_pushed_repo "$REPO10" "$BARE10"
make_source "$REPO10/overlays/landing" strict-clean
bash "$GENERATE" "$REPO10/overlays/landing" "$REPO10/overlays/SHA256SUMS" >/dev/null
REMOTE10="$CASE10/remote"
mkdir -p "$REMOTE10/custom/public/assets/landing"
printf 'stable\n' > "$REMOTE10/custom/public/assets/landing/index.html"
mkdir -p "$REPO10/deploy"
CONFIG10="$REPO10/deploy/production.env"
write_config "$CONFIG10" "$REMOTE10" "$CASE10/backups" false true
commit_and_push "$REPO10" 'initial content and deployment config'
bash "$PUBLISH" --config "$CONFIG10" --mount landing \
  --provenance-remote "$BARE10" --provenance-ref refs/heads/main \
  --source "$REPO10/overlays/landing" --manifest "$REPO10/overlays/SHA256SUMS" >/dev/null
ok 'strict provenance accepts clean pushed HEAD content'

# 11. Strict provenance requires source and manifest in the same repository.
mkdir -p "$CASE10/outside"
cp "$REPO10/overlays/SHA256SUMS" "$CASE10/outside/SHA256SUMS"
expect_failure bash "$PUBLISH" --config "$CONFIG10" --mount landing \
  --provenance-remote "$BARE10" --provenance-ref refs/heads/main \
  --source "$REPO10/overlays/landing" --manifest "$CASE10/outside/SHA256SUMS"
cp "$CONFIG10" "$CASE10/outside/production.env"
expect_failure bash "$PUBLISH" --config "$CASE10/outside/production.env" --mount landing \
  --provenance-remote "$BARE10" --provenance-ref refs/heads/main \
  --source "$REPO10/overlays/landing" --manifest "$REPO10/overlays/SHA256SUMS"
ok 'strict provenance rejects a manifest or config outside the source repository'

# 12. An ignored source file is rejected even when status is clean and the
# updated manifest itself is committed and pushed.
printf 'overlays/landing/ignored.bin\n' > "$REPO10/.gitignore"
commit_and_push "$REPO10" 'ignore fixture file'
printf 'ignored-but-present\n' > "$REPO10/overlays/landing/ignored.bin"
bash "$GENERATE" "$REPO10/overlays/landing" "$REPO10/overlays/SHA256SUMS" >/dev/null
git -C "$REPO10" add overlays/SHA256SUMS
git -C "$REPO10" commit -q -m 'manifest references ignored file'
git -C "$REPO10" push -q
[ -z "$(git -C "$REPO10" status --porcelain --untracked-files=all)" ] || fail 'ignored-file fixture repository is unexpectedly dirty'
expect_failure bash "$PUBLISH" --config "$CONFIG10" --mount landing \
  --provenance-remote "$BARE10" --provenance-ref refs/heads/main \
  --source "$REPO10/overlays/landing" --manifest "$REPO10/overlays/SHA256SUMS"
ok 'strict provenance rejects ignored or otherwise untracked source files'

# 13. A tracked Git LFS pointer is never accepted as deployable content.
CASE13="$TMP/case13"
REPO13="$CASE13/private"
BARE13="$CASE13/origin.git"
mkdir -p "$CASE13"
init_pushed_repo "$REPO13" "$BARE13"
make_source "$REPO13/overlays/landing" lfs
printf '%s\n' \
  'version https://git-lfs.github.com/spec/v1' \
  'oid sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef' \
  'size 1234' > "$REPO13/overlays/landing/assets/site.css"
write_manifest_unchecked "$REPO13/overlays/landing" "$REPO13/overlays/SHA256SUMS"
REMOTE13="$CASE13/remote"
mkdir -p "$REMOTE13/custom/public/assets/landing"
printf 'stable\n' > "$REMOTE13/custom/public/assets/landing/index.html"
mkdir -p "$REPO13/deploy"
CONFIG13="$REPO13/deploy/production.env"
write_config "$CONFIG13" "$REMOTE13" "$CASE13/backups" false true
commit_and_push "$REPO13" 'tracked lfs pointer fixture and deployment config'
expect_failure bash "$PUBLISH" --config "$CONFIG13" --mount landing \
  --provenance-remote "$BARE13" --provenance-ref refs/heads/main \
  --source "$REPO13/overlays/landing" --manifest "$REPO13/overlays/SHA256SUMS"
ok 'tracked Git LFS pointers fail closed'

# 14. A pre-existing lock blocks apply before any authoritative or local backup.
CASE14="$TMP/case14"
REMOTE14="$CASE14/remote"
LIVE14="$REMOTE14/custom/public/assets/landing"
SOURCE14="$CASE14/source"
mkdir -p "$LIVE14" "$REMOTE14/.content-deploy.lock"
printf 'stable\n' > "$LIVE14/index.html"
printf 'foreign-token\n' > "$REMOTE14/.content-deploy.lock/token"
make_source "$SOURCE14" locked
bash "$GENERATE" "$SOURCE14" "$CASE14/SHA256SUMS" >/dev/null
write_config "$CASE14/config" "$REMOTE14" "$CASE14/backups"
expect_failure bash "$PUBLISH" --config "$CASE14/config" --mount landing \
  --source "$SOURCE14" --manifest "$CASE14/SHA256SUMS" --apply
[ ! -e "$REMOTE14/.content-backups" ] || fail 'blocked apply created a remote backup outside the lock'
[ ! -e "$CASE14/backups" ] || fail 'blocked apply created a local backup outside the lock'
grep -q '^foreign-token$' "$REMOTE14/.content-deploy.lock/token" || fail 'publisher disturbed a foreign lock'
ok 'deployment lock covers authoritative and persistent backup creation'

# 15. If recovery itself fails, retain the lock and exact manual paths and
# never claim that the old tree was restored.
CASE15="$TMP/case15"
REMOTE15="$CASE15/remote"
LIVE15="$REMOTE15/custom/public/assets/landing"
SOURCE15="$CASE15/source"
mkdir -p "$LIVE15"
printf 'before-unrecoverable\n' > "$LIVE15/index.html"
printf 'version=1\nrelease_id=previous\n' > "$REMOTE15/.last-content-deploy"
make_source "$SOURCE15" after-unrecoverable
bash "$GENERATE" "$SOURCE15" "$CASE15/SHA256SUMS" >/dev/null
write_config "$CASE15/config" "$REMOTE15" "$CASE15/backups"
set +e
CONTENT_TEST_FAIL_AFTER_SWAP=1 CONTENT_TEST_FAIL_ROLLBACK=1 \
  bash "$PUBLISH" --config "$CASE15/config" --mount landing \
    --source "$SOURCE15" --manifest "$CASE15/SHA256SUMS" --apply > "$CASE15/output" 2>&1
STATUS15=$?
set -e
[ "$STATUS15" -eq 75 ] || { cat "$CASE15/output" >&2; fail "rollback failure returned $STATUS15 instead of 75"; }
grep -q 'ROLLBACK FAILED; lock retained' "$CASE15/output" || fail 'rollback failure did not report retained lock'
if grep -q 'restored and verified' "$CASE15/output"; then fail 'rollback failure falsely claimed restoration'; fi
[ -d "$REMOTE15/.content-deploy.lock" ] || fail 'rollback failure did not retain deployment lock'
MANUAL15=$(find "$REMOTE15/.content-backups" -name MANUAL_RECOVERY -print -quit)
[ -n "$MANUAL15" ] || fail 'rollback failure did not write manual recovery paths'
grep -q '^phase=manual-recovery$' "$(dirname "$MANUAL15")/PHASE" || fail 'manual recovery lacks a durable blocking phase'
assert_file_text "$LIVE15/index.html" after-unrecoverable
ok 'rollback failure is explicit, verifiable, and keeps the manual lock'

# 16. Production exchange capability is probed before snapshot or live mutation.
CASE16="$TMP/case16"
REMOTE16="$CASE16/remote"
mkdir -p "$REMOTE16/custom/public/assets/landing"
printf 'stable\n' > "$REMOTE16/custom/public/assets/landing/index.html"
EXCHANGE_HASH16=$(test_sha256 "$EXCHANGE_HELPER")
if [ "$(uname -s)" = Linux ]; then
  bash "$REMOTE_HELPER" lock-acquire "$REMOTE16" custom/public/assets/landing index.html \
    atomic-probe exchange "$EXCHANGE_HELPER" "$EXCHANGE_HASH16" false >/dev/null
  [ -d "$REMOTE16/.content-deploy.lock" ] || fail 'successful exchange probe did not retain its lock'
  bash "$REMOTE_HELPER" lock-release "$REMOTE16" atomic-probe "$EXCHANGE_HELPER" "$EXCHANGE_HASH16"
else
  expect_failure bash "$REMOTE_HELPER" lock-acquire "$REMOTE16" custom/public/assets/landing index.html \
    atomic-probe exchange "$EXCHANGE_HELPER" "$EXCHANGE_HASH16" false
  [ ! -e "$REMOTE16/.content-deploy.lock" ] || fail 'failed exchange capability probe left a lock'
fi
[ ! -e "$REMOTE16/.content-backups" ] || fail 'exchange probe created a backup before capability was proven'
assert_file_text "$REMOTE16/custom/public/assets/landing/index.html" stable
ok 'production atomic exchange capability probe is fail-closed before mutation'

# 17. A second deterministic URL-safe smoke asset is mandatory and manifest-backed.
CASE17="$TMP/case17"
REMOTE17="$CASE17/remote"
LIVE17="$REMOTE17/custom/public/assets/landing"
SOURCE17="$CASE17/source"
mkdir -p "$LIVE17"
printf 'stable\n' > "$LIVE17/index.html"
make_source "$SOURCE17" smoke
bash "$GENERATE" "$SOURCE17" "$CASE17/SHA256SUMS" >/dev/null
write_config "$CASE17/config" "$REMOTE17" "$CASE17/backups" false false assets/not-present.css
expect_failure bash "$PUBLISH" --config "$CASE17/config" --mount landing \
  --source "$SOURCE17" --manifest "$CASE17/SHA256SUMS"
assert_file_text "$LIVE17/index.html" stable
ok 'second smoke asset must be URL-safe, nonempty, and present in the manifest'

# 18. A branch configured with the local-dot pseudo-remote is not proof that
# content exists in an independently hosted upstream.
expect_failure bash "$PUBLISH" --config "$CONFIG10" --mount landing \
  --provenance-remote . --provenance-ref refs/heads/main \
  --source "$REPO10/overlays/landing" --manifest "$REPO10/overlays/SHA256SUMS"
grep -q 'absolute bare repository path' "$TMP/expected-failure.out" || fail 'local-dot provenance anchor failed for the wrong reason'
ok 'strict provenance rejects a local-dot upstream'

# 19. Replacement refs are rejected before object provenance is evaluated.
REPLACED_COMMIT=$(git -C "$REPO10" rev-parse HEAD)
REPLACEMENT_COMMIT=$(git -C "$REPO10" rev-parse HEAD^)
git -C "$REPO10" replace "$REPLACED_COMMIT" "$REPLACEMENT_COMMIT"
expect_failure bash "$PUBLISH" --config "$CONFIG10" --mount landing \
  --provenance-remote "$BARE10" --provenance-ref refs/heads/main \
  --source "$REPO10/overlays/landing" --manifest "$REPO10/overlays/SHA256SUMS"
grep -q 'Git replace refs are forbidden' "$TMP/expected-failure.out" || fail 'replacement ref failed for the wrong reason'
git -C "$REPO10" replace -d "$REPLACED_COMMIT" >/dev/null
ok 'strict provenance rejects Git replacement refs'

# 20. A nonzero exchange result is classified by both directory manifests. If
# the swap happened, the verified rollback runs and releases the lock.
CASE20="$TMP/case20"
REMOTE20="$CASE20/remote"
LIVE20="$REMOTE20/custom/public/assets/landing"
SOURCE20="$CASE20/source"
mkdir -p "$LIVE20"
printf 'before-exchange-error\n' > "$LIVE20/index.html"
printf 'version=1\nrelease_id=previous\n' > "$REMOTE20/.last-content-deploy"
make_source "$SOURCE20" after-exchange-error
bash "$GENERATE" "$SOURCE20" "$CASE20/SHA256SUMS" >/dev/null
write_config "$CASE20/config" "$REMOTE20" "$CASE20/backups"
set +e
CONTENT_TEST_EXCHANGE_FAIL_AFTER_SWAP=1 \
  bash "$PUBLISH" --config "$CASE20/config" --mount landing \
    --source "$SOURCE20" --manifest "$CASE20/SHA256SUMS" --apply > "$CASE20/output" 2>&1
STATUS20=$?
set -e
[ "$STATUS20" -ne 0 ] || fail 'post-syscall exchange error unexpectedly committed'
assert_file_text "$LIVE20/index.html" before-exchange-error
grep -q '^release_id=previous$' "$REMOTE20/.last-content-deploy" || fail 'exchange-error rollback did not restore the marker'
[ ! -e "$REMOTE20/.content-deploy.lock" ] || fail 'verified exchange-error rollback retained its lock'
PHASE20=$(find "$REMOTE20/.content-backups" -name PHASE -print -quit)
[ -n "$PHASE20" ] || fail 'exchange-error rollback lacks a phase journal'
grep -q '^phase=rolled-back$' "$PHASE20" || fail 'exchange-error rollback did not reach its verified terminal phase'
ok 'post-syscall exchange errors are classified and rolled back safely'

# 21. Linux-only fault injection proves that renameat2 may mutate both names
# before the helper can report success to its caller.
CASE21="$TMP/case21"
mkdir -p "$CASE21/left" "$CASE21/right"
printf 'left\n' > "$CASE21/left/marker"
printf 'right\n' > "$CASE21/right/marker"
if [ "$(uname -s)" = Linux ]; then
  set +e
  CONTENT_TEST_RENAME_FAIL_AFTER_SYSCALL=1 \
    python3 "$EXCHANGE_HELPER" exchange "$CASE21/left" "$CASE21/right" > "$CASE21/output" 2>&1
  STATUS21=$?
  set -e
  [ "$STATUS21" -eq 70 ] || { cat "$CASE21/output" >&2; fail "post-syscall injection returned $STATUS21 instead of 70"; }
  assert_file_text "$CASE21/left/marker" right
  assert_file_text "$CASE21/right/marker" left
  python3 "$EXCHANGE_HELPER" exchange "$CASE21/left" "$CASE21/right"
else
  assert_file_text "$CASE21/left/marker" left
  assert_file_text "$CASE21/right/marker" right
fi
ok 'rename helper exposes a post-syscall failure injection point'

# 22. A crashed transaction journal blocks a new deployment even when its
# lock directory is no longer present, while an orderly pre-activation abort
# records a durable terminal phase before releasing the lock.
CASE22="$TMP/case22"
ABORT22="$CASE22/aborted-remote"
mkdir -p "$ABORT22/custom/public/assets/landing"
printf 'stable\n' > "$ABORT22/custom/public/assets/landing/index.html"
EXCHANGE_HASH22=$(test_sha256 "$EXCHANGE_HELPER")
bash "$REMOTE_HELPER" lock-acquire "$ABORT22" \
  custom/public/assets/landing index.html abort-lock local-test \
  "$EXCHANGE_HELPER" "$EXCHANGE_HASH22" false >/dev/null
bash "$REMOTE_HELPER" snapshot "$ABORT22" custom/public/assets/landing index.html \
  abort-lock abort-backup false "$EXCHANGE_HELPER" "$EXCHANGE_HASH22" >/dev/null
bash "$REMOTE_HELPER" abort "$ABORT22" abort-lock abort-backup \
  "$EXCHANGE_HELPER" "$EXCHANGE_HASH22" test-preactivation >/dev/null
grep -q '^phase=aborted$' "$ABORT22/.content-backups/abort-backup/PHASE" || fail 'pre-activation abort lacks a durable terminal phase'
[ ! -e "$ABORT22/.content-deploy.lock" ] || fail 'pre-activation abort retained its lock'
bash "$REMOTE_HELPER" lock-acquire "$ABORT22" \
  custom/public/assets/landing index.html after-abort local-test \
  "$EXCHANGE_HELPER" "$EXCHANGE_HASH22" false >/dev/null
bash "$REMOTE_HELPER" lock-release "$ABORT22" after-abort "$EXCHANGE_HELPER" "$EXCHANGE_HASH22"

REMOTE22="$CASE22/remote"
mkdir -p "$REMOTE22/custom/public/assets/landing" "$REMOTE22/.content-backups/crashed"
printf 'stable\n' > "$REMOTE22/custom/public/assets/landing/index.html"
printf 'phase=activated\n' > "$REMOTE22/.content-backups/crashed/PHASE"
expect_failure bash "$REMOTE_HELPER" lock-acquire "$REMOTE22" \
  custom/public/assets/landing index.html stale-journal local-test \
  "$EXCHANGE_HELPER" "$EXCHANGE_HASH22" false
grep -q 'stale incomplete transaction' "$TMP/expected-failure.out" || fail 'stale journal failed for the wrong reason'
[ ! -e "$REMOTE22/.content-deploy.lock" ] || fail 'stale journal check created a new lock'
ok 'durable aborts allow retry while stale nonterminal journals block re-entry'

# 23. The authoritative remote subtree, not a sparse/skip-worktree local view,
# defines exact source coverage. A remote extra file omitted by the manifest
# must fail even when the local working tree hides it and reports clean.
CASE23="$TMP/case23"
REPO23="$CASE23/private"
BARE23="$CASE23/origin.git"
mkdir -p "$CASE23"
init_pushed_repo "$REPO23" "$BARE23"
make_source "$REPO23/overlays/landing" authoritative-extra
bash "$GENERATE" "$REPO23/overlays/landing" "$REPO23/overlays/SHA256SUMS" >/dev/null
printf 'tracked-but-not-manifested\n' > "$REPO23/overlays/landing/remote-extra.txt"
REMOTE23="$CASE23/remote"
mkdir -p "$REMOTE23/custom/public/assets/landing" "$REPO23/deploy"
printf 'stable\n' > "$REMOTE23/custom/public/assets/landing/index.html"
CONFIG23="$REPO23/deploy/production.env"
write_config "$CONFIG23" "$REMOTE23" "$CASE23/backups" false true
commit_and_push "$REPO23" 'remote source contains an extra tracked file'
git -C "$REPO23" update-index --skip-worktree overlays/landing/remote-extra.txt
rm "$REPO23/overlays/landing/remote-extra.txt"
[ -z "$(git -C "$REPO23" status --porcelain --untracked-files=all)" ] \
  || fail 'skip-worktree provenance fixture is unexpectedly dirty'
expect_failure bash "$PUBLISH" --config "$CONFIG23" --mount landing \
  --provenance-remote "$BARE23" --provenance-ref refs/heads/main \
  --source "$REPO23/overlays/landing" --manifest "$REPO23/overlays/SHA256SUMS"
grep -q 'working source lacks a provenance file' "$TMP/expected-failure.out" \
  || fail 'authoritative extra-file fixture failed for the wrong reason'
ok 'authoritative remote subtree defeats sparse or skip-worktree omissions'

# 24. SSH inputs live in an unpredictable, mode-0700 directory and are checked
# before the deployment lock or any live-content operation can use them.
UPLOAD24=$(bash "$REMOTE_HELPER" upload-create)
[[ "$UPLOAD24" =~ ^/tmp/hackforger-content-upload\.[A-Za-z0-9]+$ ]] \
  || fail 'upload allocator returned an unsafe path'
[ "$(file_mode "$UPLOAD24")" = 700 ] || fail 'upload directory is not mode 0700'
cp "$CASE1/SHA256SUMS" "$UPLOAD24/manifest.SHA256SUMS"
cp "$EXCHANGE_HELPER" "$UPLOAD24/rename-exchange.py"
tar -cf "$UPLOAD24/content.tar" -C "$SOURCE1" .
UPLOAD_ARCHIVE_HASH24=$(test_sha256 "$UPLOAD24/content.tar")
UPLOAD_MANIFEST_HASH24=$(test_sha256 "$UPLOAD24/manifest.SHA256SUMS")
UPLOAD_HELPER_HASH24=$(test_sha256 "$UPLOAD24/rename-exchange.py")
bash "$REMOTE_HELPER" upload-verify "$UPLOAD24" \
  "$UPLOAD_ARCHIVE_HASH24" "$UPLOAD_MANIFEST_HASH24" "$UPLOAD_HELPER_HASH24"
[ "$(file_mode "$UPLOAD24/content.tar")" = 600 ] || fail 'verified upload is not mode 0600'
bash "$REMOTE_HELPER" upload-clean "$UPLOAD24"
[ ! -e "$UPLOAD24" ] || fail 'verified upload directory was not removed'

UPLOAD_SYMLINK24=$(bash "$REMOTE_HELPER" upload-create)
VICTIM24="$TMP/upload-symlink-victim"
printf 'unchanged\n' > "$VICTIM24"
ln -s "$VICTIM24" "$UPLOAD_SYMLINK24/content.tar"
cp "$CASE1/SHA256SUMS" "$UPLOAD_SYMLINK24/manifest.SHA256SUMS"
cp "$EXCHANGE_HELPER" "$UPLOAD_SYMLINK24/rename-exchange.py"
expect_failure bash "$REMOTE_HELPER" upload-verify "$UPLOAD_SYMLINK24" \
  "$UPLOAD_ARCHIVE_HASH24" "$UPLOAD_MANIFEST_HASH24" "$UPLOAD_HELPER_HASH24"
assert_file_text "$VICTIM24" unchanged
bash "$REMOTE_HELPER" upload-clean "$UPLOAD_SYMLINK24"
ok 'private upload directory rejects symlink pre-placement before lock acquisition'

echo "All $PASS content publisher tests passed."
