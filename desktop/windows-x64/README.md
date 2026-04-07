# Windows 11 x64 Target

Scope:
- Native desktop build for Windows 11 x64
- Installer packaging (MSI or EXE)
- Auto-update strategy (optional second phase)

## Build

- Windows native shell: `powershell -ExecutionPolicy Bypass -File desktop/windows-x64/scripts/build-win-x64.ps1`
- Cross-run from bash: `bash desktop/windows-x64/scripts/build-win-x64.sh`
- Installer packaging (requires WiX on Windows host): `powershell -ExecutionPolicy Bypass -File desktop/windows-x64/scripts/package-win-x64.ps1 -Version "0.1.0-dev" -AppName "MD"`

Optional (connected sync with web backend):

- Set `MD_DESKTOP_REMOTE_API_URL=https://md.cybergraphe.fr` before build to embed remote sync API target.

## Signing

- Script: `powershell -ExecutionPolicy Bypass -File desktop/windows-x64/scripts/sign-win-x64.ps1 -InputExe "build\\bin\\MD.exe"`
- Required secrets/environment:
	- `MD_WIN_CERT_PFX_B64` (base64 PFX)
	- `MD_WIN_CERT_PASSWORD`
- Release expectation: sign both the built executable and generated MSI before publication.

Optional:
- `MD_WIN_CERT_BASE64` can be used as fallback key name for certificate payload.
- Timestamp URL can be overridden with `-TimestampUrl`.

## Planned outputs

- `build/releases/windows-x64/MD-<version>-windows-x64.exe` (portable)
- `build/releases/windows-x64/MD-<version>-windows-x64.zip` (bundle exe + web assets)
- `build/releases/windows-x64/MD-<version>-windows-x64.msi` (installer)
- `build/releases/windows-x64/MD-latest-windows-x64.*` stable aliases

## Baseline requirements

- Go toolchain
- Node.js + npm
- Wails CLI (or chosen desktop framework)
- Code signing certificate (for production distribution)
- signtool.exe (Windows SDK Signing Tools)

## Installer status

Installer workspace is prepared under `installer/` for MSI/EXE integration.
