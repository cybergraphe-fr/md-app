param(
  [string]$Version = "0.1.0-dev"
)

$ErrorActionPreference = "Stop"
$RootDir = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")

function Get-WailsBin {
  $wails = Get-Command wails -ErrorAction SilentlyContinue
  if ($wails) { return $wails.Source }

  $goPath = go env GOPATH
  $candidate = Join-Path $goPath "bin\wails.exe"
  if (Test-Path $candidate) { return $candidate }

  return $null
}

$wailsBin = Get-WailsBin
if (-not $wailsBin) {
  Write-Error "wails CLI not found. Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
}

Set-Location $RootDir

if (-not $env:GOFLAGS) {
  $env:GOFLAGS = "-p=1"
}

Write-Host "[desktop] building frontend assets"
npm ci --prefix web
npm run build --prefix web

Write-Host "[desktop] building Wails app for windows/amd64"
& $wailsBin build `
  -tags desktop `
  -platform windows/amd64 `
  -clean `
  -trimpath `
  -ldflags "-X main.Version=$Version"

$targetWebDir = Join-Path $RootDir "build\bin\web"
New-Item -ItemType Directory -Force -Path $targetWebDir | Out-Null
Copy-Item -Path (Join-Path $RootDir "web\dist\*") -Destination $targetWebDir -Recurse -Force

Write-Host "[desktop] windows build finished"
Write-Host "[desktop] artifacts: build/bin/"
