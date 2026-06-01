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

echo "[1/4] Building frontend on host (Node) ..."
# The cross-compile image (golang:1.25-bookworm) only has Debian's Node 18 via
# apt, but Forgejo's frontend build requires Node >= 20. Mounting the repo into
# the container would also clobber the Mac's darwin-native node_modules. So build
# the frontend here on the Mac (Node 20+, correct platform); the container step
# below compiles only the Go backend and embeds these assets via bindata.
make frontend

echo "[2/4] Pulling build image $IMAGE (no-op if cached)..."
docker pull "$IMAGE"

echo "[3/4] Cross-compiling backend (embeds host-built frontend via bindata; 3-6 min)..."
docker run --rm --platform linux/amd64 \
  -v "$REPO":/src -w /src \
  "$IMAGE" bash -ec '
    # Use Aliyun mirror — deb.debian.org is unreliable from CN
    sed -i "s|deb.debian.org|mirrors.aliyun.com|g; s|security.debian.org|mirrors.aliyun.com|g" \
        /etc/apt/sources.list.d/debian.sources 2>/dev/null || \
      sed -i "s|deb.debian.org|mirrors.aliyun.com|g; s|security.debian.org|mirrors.aliyun.com|g" \
        /etc/apt/sources.list 2>/dev/null || true
    apt-get update -qq
    apt-get install -y -qq -o Acquire::Retries=3 build-essential git
    git config --global --add safe.directory /src
    # make backend (NOT make build): compiles Go + embeds public/ via bindata,
    # without running the Node frontend target. No node needed in the container.
    TAGS="bindata sqlite sqlite_unlock_notify" \
      GOOS=linux GOARCH=amd64 \
      GOPROXY="https://goproxy.cn,direct" \
      make backend
    mv gitea '"$OUT"'
  '

echo "[4/4] Verifying output is a Linux amd64 ELF..."
if file "$OUT" | grep -q "ELF 64-bit LSB.*x86-64"; then
  ls -lh "$OUT"
  sha256sum "$OUT" 2>/dev/null || shasum -a 256 "$OUT"
  echo "✓ Build OK"
else
  echo "FATAL: $OUT is not a Linux x86-64 ELF" >&2
  file "$OUT" >&2
  exit 2
fi
