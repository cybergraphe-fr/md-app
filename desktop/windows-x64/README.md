# Windows 11 x64 Target

Scope:
- Native desktop build for Windows 11 x64
- Installer packaging (MSI or EXE)
- Auto-update strategy (optional second phase)

## Build

- Windows native shell: `powershell -ExecutionPolicy Bypass -File desktop/windows-x64/scripts/build-win-x64.ps1`
- Cross-run from bash: `bash desktop/windows-x64/scripts/build-win-x64.sh`

## Signing

- Script: `powershell -ExecutionPolicy Bypass -File desktop/windows-x64/scripts/sign-win-x64.ps1 -InputExe "build\\bin\\MD.exe"`
- Required secrets/environment:
	- `MD_WIN_CERT_PFX_B64` (base64 PFX)
	- `MD_WIN_CERT_PASSWORD`

Optional:
- `MD_WIN_CERT_BASE64` can be used as fallback key name for certificate payload.
- Timestamp URL can be overridden with `-TimestampUrl`.

## Planned outputs

- md-desktop-x64.exe (or app bundle)
- installer artifact in installer/

## Baseline requirements

- Go toolchain
- Node.js + npm
- Wails CLI (or chosen desktop framework)
- Code signing certificate (for production distribution)
- signtool.exe (Windows SDK Signing Tools)

## Installer status

Installer workspace is prepared under `installer/` for MSI/EXE integration.
