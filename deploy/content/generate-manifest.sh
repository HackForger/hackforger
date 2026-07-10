#!/usr/bin/env bash
# Generate a deterministic SHA-256 manifest for an external content tree.

set -euo pipefail

usage() {
  echo "Usage: $0 SOURCE_DIR OUTPUT_SHA256SUMS" >&2
}

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

[ "$#" -eq 2 ] || { usage; exit 64; }

SOURCE_INPUT=$1
OUTPUT_INPUT=$2
[ -d "$SOURCE_INPUT" ] || die "source directory does not exist: $SOURCE_INPUT"

SOURCE=$(cd "$SOURCE_INPUT" && pwd -P)
OUTPUT_DIR=$(cd "$(dirname "$OUTPUT_INPUT")" && pwd -P)
OUTPUT="$OUTPUT_DIR/$(basename "$OUTPUT_INPUT")"

case "$OUTPUT" in
  "$SOURCE"/*) die "manifest must live outside the content source tree" ;;
esac

special=$(find "$SOURCE" ! -type d ! -type f -print -quit)
[ -z "$special" ] || die "content tree contains a non-directory/non-regular inode: $special"
empty_directory=$(find "$SOURCE" -type d -empty -print -quit)
[ -z "$empty_directory" ] || die "content tree contains an empty directory not representable by the manifest: $empty_directory"

WORK=$(mktemp -d "${TMPDIR:-/tmp}/content-manifest.XXXXXX")
trap 'rm -rf "$WORK"' EXIT INT TERM HUP

(
  cd "$SOURCE"
  find . -type f -print | sed 's#^\./##' | LC_ALL=C sort
) > "$WORK/files"

[ -s "$WORK/files" ] || die "content source is empty"

: > "$WORK/SHA256SUMS"
while IFS= read -r path || [ -n "$path" ]; do
  [ -n "$path" ] || die "empty path in generated file list"
  if [[ "$path" =~ [[:cntrl:]] ]]; then
    die "control characters are not allowed in content paths"
  fi
  case "$path" in
    /*|./*|../*|*/../*|*/..|*/./*|*//*|*\\*)
      die "unsafe content path: $path"
      ;;
  esac
  if [ "$(sed -n '1p' "$SOURCE/$path")" = 'version https://git-lfs.github.com/spec/v1' ] \
     && grep -q '^oid sha256:[0-9a-f]\{64\}$' "$SOURCE/$path" \
     && grep -q '^size [0-9][0-9]*$' "$SOURCE/$path"; then
    die "Git LFS pointer files are not deployable content: $path"
  fi
  printf '%s  %s\n' "$(sha256_file "$SOURCE/$path")" "$path" >> "$WORK/SHA256SUMS"
done < "$WORK/files"

mv "$WORK/SHA256SUMS" "$OUTPUT"
echo "Wrote $(wc -l < "$OUTPUT" | tr -d ' ') entries to $OUTPUT"
