# Windows Installer Workspace

Place installer-related assets here:
- MSI/WiX project files, or
- EXE installer scripts (Inno Setup / NSIS), and
- branding resources.

Suggested first milestone:
- produce a signed installer for Windows 11 x64.

Current baseline:
- `desktop/windows-x64/scripts/package-win-x64.ps1` generates WiX sources in a temporary build workspace,
- compiles MSI output under `build/releases/windows-x64/`,
- and keeps stable `MD-latest-windows-x64.*` aliases for web download URLs.
