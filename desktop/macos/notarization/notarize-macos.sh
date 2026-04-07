#!/usr/bin/env bash
set -euo pipefail

APP_BUNDLE="${1:-build/bin/MD.app}"
BUNDLE_ID="${MD_MACOS_BUNDLE_ID:-fr.cybergraphe.md}"
TEAM_ID="${MD_MACOS_TEAM_ID:-}"
SIGN_IDENTITY="${MD_MACOS_SIGN_IDENTITY:-}"
NOTARY_PROFILE="${MD_MACOS_NOTARY_KEYCHAIN_PROFILE:-}"
APPLE_ID="${MD_MACOS_NOTARY_APPLE_ID:-}"
APP_PASSWORD="${MD_MACOS_NOTARY_APP_PASSWORD:-}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
ENTITLEMENTS_FILE="$ROOT_DIR/desktop/macos/notarization/entitlements.plist"
ZIP_PATH="$ROOT_DIR/build/bin/MD-notarize.zip"

cd "$ROOT_DIR"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "error: notarization must run on macOS" >&2
  exit 1
fi

if [[ ! -d "$APP_BUNDLE" ]]; then
  echo "error: app bundle not found: $APP_BUNDLE" >&2
  exit 1
fi

if [[ -z "$SIGN_IDENTITY" ]]; then
  echo "error: missing MD_MACOS_SIGN_IDENTITY" >&2
  exit 1
fi
if [[ -z "$TEAM_ID" ]]; then
  echo "error: missing MD_MACOS_TEAM_ID" >&2
  exit 1
fi
if [[ ! -f "$ENTITLEMENTS_FILE" ]]; then
  echo "error: entitlements file missing: $ENTITLEMENTS_FILE" >&2
  exit 1
fi

if [[ -z "$NOTARY_PROFILE" ]]; then
  if [[ -z "$APPLE_ID" || -z "$APP_PASSWORD" ]]; then
    echo "error: set MD_MACOS_NOTARY_KEYCHAIN_PROFILE or both MD_MACOS_NOTARY_APPLE_ID and MD_MACOS_NOTARY_APP_PASSWORD" >&2
    exit 1
  fi
fi

echo "[desktop] codesign app bundle"
/usr/bin/codesign --force --deep --options runtime --timestamp --sign "$SIGN_IDENTITY" --entitlements "$ENTITLEMENTS_FILE" "$APP_BUNDLE"

/usr/bin/codesign --verify --deep --strict --verbose=2 "$APP_BUNDLE"

rm -f "$ZIP_PATH"
echo "[desktop] creating notarization zip"
/usr/bin/ditto -c -k --sequesterRsrc --keepParent "$APP_BUNDLE" "$ZIP_PATH"

if [[ -n "$NOTARY_PROFILE" ]]; then
  echo "[desktop] submitting with keychain profile"
  /usr/bin/xcrun notarytool submit "$ZIP_PATH" --keychain-profile "$NOTARY_PROFILE" --team-id "$TEAM_ID" --wait
else
  echo "[desktop] submitting with apple id credentials"
  /usr/bin/xcrun notarytool submit "$ZIP_PATH" --apple-id "$APPLE_ID" --password "$APP_PASSWORD" --team-id "$TEAM_ID" --wait
fi

echo "[desktop] stapling ticket"
/usr/bin/xcrun stapler staple "$APP_BUNDLE"

echo "[desktop] gatekeeper assessment"
/usr/sbin/spctl --assess --type execute --verbose "$APP_BUNDLE"

echo "[desktop] notarization completed for bundle id $BUNDLE_ID"
