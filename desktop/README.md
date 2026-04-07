# MD Desktop Packaging

Desktop packaging baseline for MD:
- Windows 11 x64
- macOS Intel + Apple Silicon

## Runtime strategy

Desktop shell: Wails.

Application model:
- an internal HTTP handler is built from existing MD API + static frontend,
- Wails AssetServer serves the app from this handler,
- platform-specific packaging scripts add web assets beside desktop artifacts.

This preserves the current backend and frontend architecture with minimal divergence.

## Tree

- common/: shared assets and checklists
- windows-x64/: Windows build + installer workspace
- macos/: macOS build + notarization workspace

## Prerequisites

- Go 1.25+
- Node.js 22+
- npm
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

## Build commands

From repository root:

- Windows x64: `bash desktop/windows-x64/scripts/build-win-x64.sh`
- macOS (default arm64, override with `MD_DESKTOP_MACOS_PLATFORMS`): `bash desktop/macos/scripts/build-macos.sh`
- Windows installers (exe/zip/msi, on Windows host): `powershell -ExecutionPolicy Bypass -File desktop/windows-x64/scripts/package-win-x64.ps1 -Version "0.1.0-dev" -AppName "MD"`
- macOS installers (dmg/pkg/zip, on macOS host): `bash desktop/macos/scripts/package-macos-installers.sh "0.1.0-dev" "MD"`

To embed remote web sync support in desktop binaries, set:

- `MD_DESKTOP_REMOTE_API_URL=https://md.cybergraphe.fr`

When this variable is set at build time, desktop API calls are proxied to the remote backend,
so web and desktop can link the same workspace with the generated sync code.

## Signing and Notarization

- Windows signing script: `powershell -ExecutionPolicy Bypass -File desktop/windows-x64/scripts/sign-win-x64.ps1 -InputExe "build\\bin\\MD.exe"`
- macOS notarization script: `bash desktop/macos/notarization/notarize-macos.sh build/bin/MD.app`

## GitHub workflow

- Manual workflow: `.github/workflows/desktop-release.yml`
- Trigger: `workflow_dispatch`
- Inputs: `version`, `sign_windows`, `notarize_macos`, `sync_api_base_url`

Required secrets for Windows signing:
- `MD_WIN_CERT_PFX_B64`
- `MD_WIN_CERT_PASSWORD`

Windows signing gate:
- If `sign_windows=true`, the workflow fails when either Windows signing secret is missing.
- MSI outputs are signed after packaging.
- Use `sign_windows=false` only for internal unsigned test builds.

Required secrets for macOS notarization:
- `MD_MACOS_SIGN_IDENTITY`
- `MD_MACOS_TEAM_ID`
- and either:
	- `MD_MACOS_NOTARY_KEYCHAIN_PROFILE`
	- or `MD_MACOS_NOTARY_APPLE_ID` + `MD_MACOS_NOTARY_APP_PASSWORD`

Notes:
- macOS artifacts generally require running on macOS for final signed/notarized outputs.
- Windows artifacts are best produced on Windows for installer/signing finalization.
- If remote sync input/variable is omitted, desktop runs in local mode with local-only workspace storage.

Installer outputs are generated in:

- `build/releases/windows-x64/`
- `build/releases/macos/`

Both versioned and stable `*-latest-*` filenames are emitted for web URL wiring.

## Quick Make targets

- `make desktop-bin-win-x64`
- `make desktop-bin-macos-amd64`
- `make desktop-bin-macos-arm64`
- `make desktop-bin-all`
- `make desktop-package-win-x64`
- `make desktop-package-macos`
- `make desktop-sign-win-x64`
- `make desktop-notary-profile`
- `make desktop-notarize-macos`
