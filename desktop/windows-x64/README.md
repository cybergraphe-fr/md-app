# Windows 11 x64 Target

Scope:
- Native desktop build for Windows 11 x64
- Installer packaging (MSI or EXE)
- Auto-update strategy (optional second phase)

## Build

- Windows native shell: `powershell -ExecutionPolicy Bypass -File desktop/windows-x64/scripts/build-win-x64.ps1`
- Cross-run from bash: `bash desktop/windows-x64/scripts/build-win-x64.sh`

## Planned outputs

- md-desktop-x64.exe (or app bundle)
- installer artifact in installer/

## Baseline requirements

- Go toolchain
- Node.js + npm
- Wails CLI (or chosen desktop framework)
- Code signing certificate (for production distribution)

## Installer status

Installer workspace is prepared under `installer/` for MSI/EXE integration.
