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
- macOS amd64 + arm64: `bash desktop/macos/scripts/build-macos.sh`

Notes:
- macOS artifacts generally require running on macOS for final signed/notarized outputs.
- Windows artifacts are best produced on Windows for installer/signing finalization.

## Quick Make targets

- `make desktop-bin-win-x64`
- `make desktop-bin-macos-amd64`
- `make desktop-bin-macos-arm64`
- `make desktop-bin-all`
- `make desktop-package-win-x64`
- `make desktop-package-macos`
