#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0-dev}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"

find_wails() {
  if command -v wails >/dev/null 2>&1; then
    command -v wails
    return 0
  fi
  local gopath_bin
  gopath_bin="$(go env GOPATH)/bin/wails"
  if [[ -x "$gopath_bin" ]]; then
    echo "$gopath_bin"
    return 0
  fi
  return 1
}

WAILS_BIN="$(find_wails || true)"
if [[ -z "$WAILS_BIN" ]]; then
  echo "error: wails CLI not found. Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest" >&2
  exit 1
fi

cd "$ROOT_DIR"

if [[ -z "${GOFLAGS:-}" ]]; then
  export GOFLAGS="-p=1"
fi

echo "[desktop] building frontend assets"
npm ci --prefix web
npm run build --prefix web

echo "[desktop] building Wails app for windows/amd64"
"$WAILS_BIN" build \
  -tags desktop \
  -platform windows/amd64 \
  -clean \
  -trimpath \
  -ldflags "-X main.Version=${VERSION}"

if [[ ! -f build/bin/MD.exe ]]; then
  echo "error: expected artifact build/bin/MD.exe was not produced" >&2
  exit 1
fi

mkdir -p build/bin/web
if command -v rsync >/dev/null 2>&1; then
  rsync -a --delete web/dist/ build/bin/web/
else
  rm -rf build/bin/web/*
  cp -a web/dist/. build/bin/web/
fi

echo "[desktop] windows build finished"
echo "[desktop] artifacts: build/bin/"
