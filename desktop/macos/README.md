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

The script currently targets `darwin/amd64,darwin/arm64`.
For native notarized releases, run from macOS with signing credentials configured.

## Baseline requirements

- Xcode Command Line Tools
- Go toolchain
- Node.js + npm
- Wails CLI (or chosen desktop framework)
- Apple Developer certificate and notarization credentials
