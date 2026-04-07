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

Optional (connected sync with web backend):

- Set `MD_DESKTOP_REMOTE_API_URL=https://md.cybergraphe.fr` before build to embed remote sync API target.

The script currently targets `darwin/amd64,darwin/arm64`.
For native notarized releases, run from macOS with signing credentials configured.

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
