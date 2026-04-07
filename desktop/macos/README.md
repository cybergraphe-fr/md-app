# macOS Target

Scope:
- Desktop build for macOS
- Codesign and notarization pipeline

## Architecture strategy

- Intel x64 (darwin/amd64)
- Apple Silicon (darwin/arm64)
- Optional universal app bundle in a later iteration

## Build

- `bash desktop/macos/scripts/build-macos.sh`
- `bash desktop/macos/scripts/package-macos-installers.sh "0.1.0-dev" "MD"`

Optional (connected sync with web backend):

- Set `MD_DESKTOP_REMOTE_API_URL=https://md.cybergraphe.fr` before build to embed remote sync API target.

The script defaults to `darwin/arm64` (aligned with `macos-14` runners).
Override target list when needed with `MD_DESKTOP_MACOS_PLATFORMS`.
For native notarized releases, run from macOS with signing credentials configured.

Installer outputs are written to `build/releases/macos/`:

- `MD-<version>-macos.dmg`
- `MD-<version>-macos.pkg`
- `MD-<version>-macos.zip`
- stable aliases `MD-latest-macos.*`

## Notarization

- Script: `bash desktop/macos/notarization/notarize-macos.sh build/bin/MD.app`
- Required environment:
	- `MD_MACOS_SIGN_IDENTITY`
	- `MD_MACOS_TEAM_ID`
	- and either:
		- `MD_MACOS_NOTARY_KEYCHAIN_PROFILE`
		- or `MD_MACOS_NOTARY_APPLE_ID` + `MD_MACOS_NOTARY_APP_PASSWORD`

Optional:
- `MD_MACOS_BUNDLE_ID` (default: `fr.cybergraphe.md`)

## Baseline requirements

- Xcode Command Line Tools
- Go toolchain
- Node.js + npm
- Wails CLI (or chosen desktop framework)
- Apple Developer certificate and notarization credentials
