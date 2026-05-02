#!/bin/bash
# Cross-compile gitea for linux/amd64 using docker, without polluting the Mac
# with a Linux Go toolchain.
#
# Output: ./gitea-linux-amd64 in the repo root.
# Usage:  bash deploy/ecs/build-linux.sh
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO"

OUT="gitea-linux-amd64"
IMAGE="golang:1.25-bookworm"

if ! command -v docker >/dev/null 2>&1; then
  echo "FATAL: docker not found on PATH. Install Docker Desktop." >&2
  exit 1
fi

echo "[1/3] Pulling build image $IMAGE (no-op if cached)..."
docker pull "$IMAGE"

echo "[2/3] Cross-compiling gitea (this takes 3-6 minutes)..."
docker run --rm --platform linux/amd64 \
  -v "$REPO":/src -w /src \
  "$IMAGE" bash -ec '
    # Use Aliyun mirror — deb.debian.org is unreliable from CN
    sed -i "s|deb.debian.org|mirrors.aliyun.com|g; s|security.debian.org|mirrors.aliyun.com|g" \
        /etc/apt/sources.list.d/debian.sources 2>/dev/null || \
      sed -i "s|deb.debian.org|mirrors.aliyun.com|g; s|security.debian.org|mirrors.aliyun.com|g" \
        /etc/apt/sources.list 2>/dev/null || true
    apt-get update -qq
    apt-get install -y -qq -o Acquire::Retries=3 build-essential nodejs npm git
    git config --global --add safe.directory /src
    TAGS="bindata sqlite sqlite_unlock_notify" \
      GOOS=linux GOARCH=amd64 \
      GOPROXY="https://goproxy.cn,direct" \
      make build
    mv gitea '"$OUT"'
  '

echo "[3/3] Verifying output is a Linux amd64 ELF..."
if file "$OUT" | grep -q "ELF 64-bit LSB.*x86-64"; then
  ls -lh "$OUT"
  sha256sum "$OUT" 2>/dev/null || shasum -a 256 "$OUT"
  echo "✓ Build OK"
else
  echo "FATAL: $OUT is not a Linux x86-64 ELF" >&2
  file "$OUT" >&2
  exit 2
fi
