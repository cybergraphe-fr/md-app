#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0-dev}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
APP_NAME="${2:-MD}"
DESKTOP_PROJECT_DIR="${ROOT_DIR}/cmd/desktop"
DESKTOP_BIN_DIR="${DESKTOP_PROJECT_DIR}/build/bin"
MAC_PLATFORMS="${MD_DESKTOP_MACOS_PLATFORMS:-darwin/arm64}"

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

echo "[md-desktop] macOS build scaffold"
echo "Version: ${VERSION}"

echo "[desktop] building frontend assets"
npm ci --prefix web
npm run build --prefix web

LDFLAGS="-X main.Version=${VERSION}"
if [[ -n "${MD_DESKTOP_REMOTE_API_URL:-}" ]]; then
	LDFLAGS+=" -X main.RemoteAPIURL=${MD_DESKTOP_REMOTE_API_URL}"
fi

echo "[desktop] building Wails app for ${MAC_PLATFORMS}"
(
	cd "$DESKTOP_PROJECT_DIR"
	"$WAILS_BIN" build \
		-tags desktop \
		-platform "$MAC_PLATFORMS" \
		-skipbindings \
		-s \
		-clean \
		-trimpath \
		-ldflags "$LDFLAGS"
)

SOURCE_APP_BUNDLE="${DESKTOP_BIN_DIR}/${APP_NAME}.app"
APP_BUNDLE="build/bin/${APP_NAME}.app"
if [[ ! -d "$SOURCE_APP_BUNDLE/Contents/Resources" ]]; then
	echo "error: app bundle not found at $SOURCE_APP_BUNDLE" >&2
	echo "hint: run this packaging target on macOS with the required toolchain/signing setup" >&2
	exit 1
fi

mkdir -p build/bin
rm -rf "$APP_BUNDLE"
cp -R "$SOURCE_APP_BUNDLE" "$APP_BUNDLE"

mkdir -p "$APP_BUNDLE/Contents/Resources/web"
if command -v rsync >/dev/null 2>&1; then
	rsync -a --delete web/dist/ "$APP_BUNDLE/Contents/Resources/web/"
else
	rm -rf "$APP_BUNDLE/Contents/Resources/web"/*
	cp -a web/dist/. "$APP_BUNDLE/Contents/Resources/web/"
fi

echo "[desktop] macOS build finished"
echo "[desktop] artifacts: build/bin/"
