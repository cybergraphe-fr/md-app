# macOS Notarization Workspace

This folder contains the signing and notarization helpers for desktop macOS releases.

## Files

- `entitlements.plist`: entitlements used during hardened-runtime signing.
- `notarize-macos.sh`: sign, submit to notary service, staple, and assess.
- `store-notary-profile.sh`: helper to store keychain profile for notarytool.

## Credential modes

Mode A: keychain profile
- `MD_MACOS_SIGN_IDENTITY`
- `MD_MACOS_TEAM_ID`
- `MD_MACOS_NOTARY_KEYCHAIN_PROFILE`

Mode B: Apple ID + app password
- `MD_MACOS_SIGN_IDENTITY`
- `MD_MACOS_TEAM_ID`
- `MD_MACOS_NOTARY_APPLE_ID`
- `MD_MACOS_NOTARY_APP_PASSWORD`

## Typical flow on macOS

1. Build desktop app bundle.
2. Run notarization script.
3. Validate stapled app with Gatekeeper assessment.
