#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0-dev}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
DESKTOP_PROJECT_DIR="${ROOT_DIR}/cmd/desktop"
DESKTOP_BIN_DIR="${DESKTOP_PROJECT_DIR}/build/bin"

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

LDFLAGS="-X main.Version=${VERSION}"
if [[ -n "${MD_DESKTOP_REMOTE_API_URL:-}" ]]; then
  LDFLAGS+=" -X main.RemoteAPIURL=${MD_DESKTOP_REMOTE_API_URL}"
fi

echo "[desktop] building Wails app for windows/amd64"
(
  cd "$DESKTOP_PROJECT_DIR"
  "$WAILS_BIN" build \
    -tags desktop \
    -platform windows/amd64 \
    -skipbindings \
    -s \
    -clean \
    -trimpath \
    -ldflags "$LDFLAGS"
)

if [[ ! -f "${DESKTOP_BIN_DIR}/MD.exe" ]]; then
  echo "error: expected artifact ${DESKTOP_BIN_DIR}/MD.exe was not produced" >&2
  exit 1
fi

mkdir -p build/bin
cp "${DESKTOP_BIN_DIR}/MD.exe" build/bin/MD.exe

mkdir -p build/bin/web
if command -v rsync >/dev/null 2>&1; then
  rsync -a --delete web/dist/ build/bin/web/
else
  rm -rf build/bin/web/*
  cp -a web/dist/. build/bin/web/
fi

echo "[desktop] windows build finished"
echo "[desktop] artifacts: build/bin/"
