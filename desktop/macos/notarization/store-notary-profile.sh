#!/usr/bin/env bash
set -euo pipefail

PROFILE_NAME="${1:-md-notary}"
APPLE_ID="${MD_MACOS_NOTARY_APPLE_ID:-}"
APP_PASSWORD="${MD_MACOS_NOTARY_APP_PASSWORD:-}"
TEAM_ID="${MD_MACOS_TEAM_ID:-}"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "error: notary profile setup must run on macOS" >&2
  exit 1
fi

if [[ -z "$APPLE_ID" || -z "$APP_PASSWORD" || -z "$TEAM_ID" ]]; then
  echo "error: set MD_MACOS_NOTARY_APPLE_ID, MD_MACOS_NOTARY_APP_PASSWORD and MD_MACOS_TEAM_ID" >&2
  exit 1
fi

echo "[desktop] storing keychain profile: $PROFILE_NAME"
/usr/bin/xcrun notarytool store-credentials "$PROFILE_NAME" --apple-id "$APPLE_ID" --password "$APP_PASSWORD" --team-id "$TEAM_ID"

echo "[desktop] profile stored"
