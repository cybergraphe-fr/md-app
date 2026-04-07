#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0-dev}"
APP_NAME="${2:-MD}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
APP_BUNDLE="${ROOT_DIR}/build/bin/${APP_NAME}.app"

if [[ ! -d "${APP_BUNDLE}" ]]; then
  echo "error: app bundle not found: ${APP_BUNDLE}" >&2
  exit 1
fi

if ! command -v hdiutil >/dev/null 2>&1; then
  echo "error: hdiutil not found (required on macOS)" >&2
  exit 1
fi
if ! command -v pkgbuild >/dev/null 2>&1; then
  echo "error: pkgbuild not found (required on macOS)" >&2
  exit 1
fi

safe_version="${VERSION//[^A-Za-z0-9._-]/-}"
out_dir="${ROOT_DIR}/build/releases/macos"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

mkdir -p "${out_dir}"
cp -R "${APP_BUNDLE}" "${work_dir}/"

DMG_PATH="${out_dir}/${APP_NAME}-${safe_version}-macos.dmg"
PKG_PATH="${out_dir}/${APP_NAME}-${safe_version}-macos.pkg"
ZIP_PATH="${out_dir}/${APP_NAME}-${safe_version}-macos.zip"
LATEST_DMG_PATH="${out_dir}/${APP_NAME}-latest-macos.dmg"
LATEST_PKG_PATH="${out_dir}/${APP_NAME}-latest-macos.pkg"
LATEST_ZIP_PATH="${out_dir}/${APP_NAME}-latest-macos.zip"

rm -f "${DMG_PATH}" "${PKG_PATH}" "${ZIP_PATH}" "${LATEST_DMG_PATH}" "${LATEST_PKG_PATH}" "${LATEST_ZIP_PATH}"

hdiutil create \
  -volname "${APP_NAME}" \
  -srcfolder "${work_dir}/${APP_NAME}.app" \
  -ov \
  -format UDZO \
  "${DMG_PATH}"

pkgbuild \
  --component "${work_dir}/${APP_NAME}.app" \
  --install-location "/Applications" \
  "${PKG_PATH}"

(
  cd "${work_dir}"
  ditto -c -k --sequesterRsrc --keepParent "${APP_NAME}.app" "${ZIP_PATH}"
)

cp "${DMG_PATH}" "${LATEST_DMG_PATH}"
cp "${PKG_PATH}" "${LATEST_PKG_PATH}"
cp "${ZIP_PATH}" "${LATEST_ZIP_PATH}"

echo "[desktop] macOS installer artifacts created"
echo "[desktop] dmg: ${DMG_PATH}"
echo "[desktop] pkg: ${PKG_PATH}"
echo "[desktop] zip: ${ZIP_PATH}"
