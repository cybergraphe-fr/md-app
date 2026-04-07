param(
  [string]$InputExe = "build\bin\MD.exe",
  [string]$TimestampUrl = "http://timestamp.digicert.com",
  [switch]$SkipVerify,
  [switch]$DryRun
)

$ErrorActionPreference = "Stop"
$RootDir = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")
Set-Location $RootDir

function Get-SignTool {
  $cmd = Get-Command signtool.exe -ErrorAction SilentlyContinue
  if ($cmd) { return $cmd.Source }

  $kitsRoot = Join-Path ${env:ProgramFiles(x86)} "Windows Kits\10\bin"
  if (-not (Test-Path $kitsRoot)) {
    return $null
  }

  $candidate = Get-ChildItem $kitsRoot -Recurse -Filter signtool.exe -ErrorAction SilentlyContinue |
    Sort-Object FullName -Descending |
    Select-Object -First 1
  if ($candidate) { return $candidate.FullName }

  return $null
}

$exePath = if ([System.IO.Path]::IsPathRooted($InputExe)) {
  $InputExe
} else {
  Join-Path $RootDir $InputExe
}

if (-not (Test-Path $exePath)) {
  Write-Error "Input executable not found: $exePath"
}

$pfxBase64 = if ($env:MD_WIN_CERT_PFX_B64) { $env:MD_WIN_CERT_PFX_B64 } else { $env:MD_WIN_CERT_BASE64 }
$pfxPassword = $env:MD_WIN_CERT_PASSWORD

if (-not $pfxBase64) {
  Write-Error "Missing signing certificate. Set MD_WIN_CERT_PFX_B64 (or MD_WIN_CERT_BASE64)."
}
if (-not $pfxPassword) {
  Write-Error "Missing certificate password. Set MD_WIN_CERT_PASSWORD."
}

$signTool = Get-SignTool
if (-not $signTool) {
  Write-Error "signtool.exe not found. Install Windows SDK Signing Tools."
}

$tempPfx = Join-Path $env:TEMP ("md-signing-" + [Guid]::NewGuid().ToString() + ".pfx")
try {
  [IO.File]::WriteAllBytes($tempPfx, [Convert]::FromBase64String($pfxBase64))

  $signArgs = @(
    "sign",
    "/fd", "SHA256",
    "/tr", $TimestampUrl,
    "/td", "SHA256",
    "/f", $tempPfx,
    "/p", $pfxPassword,
    "/v", $exePath
  )

  if ($DryRun) {
    Write-Host "[dry-run] $signTool $($signArgs -join ' ')"
  } else {
    Write-Host "[desktop] signing $exePath"
    & $signTool @signArgs
  }

  if (-not $SkipVerify) {
    $verifyArgs = @("verify", "/pa", "/v", $exePath)
    if ($DryRun) {
      Write-Host "[dry-run] $signTool $($verifyArgs -join ' ')"
    } else {
      Write-Host "[desktop] verifying signature"
      & $signTool @verifyArgs
    }
  }

  Write-Host "[desktop] windows signing completed"
}
finally {
  if (Test-Path $tempPfx) {
    Remove-Item -Force $tempPfx
  }
}
