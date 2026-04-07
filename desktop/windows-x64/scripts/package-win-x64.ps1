param(
  [string]$Version = "0.1.0-dev",
  [string]$AppName = "MD",
  [string]$Manufacturer = "Cybergraphe",
  [string]$UpgradeCode = "{A8AF6E8E-3A6B-4D78-B58F-E34FA9567D1D}",
  [switch]$SkipMsi
)

$ErrorActionPreference = "Stop"
$RootDir = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")
$BinDir = Join-Path $RootDir "build\bin"
$ExePath = Join-Path $BinDir "$AppName.exe"
$WebDir = Join-Path $BinDir "web"

function Normalize-MsiVersion {
  param([string]$RawVersion)

  $parts = @(0, 0, 0)
  $matches = [System.Text.RegularExpressions.Regex]::Matches($RawVersion, "\d+")
  for ($i = 0; $i -lt [Math]::Min(3, $matches.Count); $i++) {
    $value = [int]$matches[$i].Value
    if ($value -gt 65535) {
      $value = 65535
    }
    $parts[$i] = $value
  }
  return "{0}.{1}.{2}" -f $parts[0], $parts[1], $parts[2]
}

function Get-WixBinary {
  param([string]$Name)

  $command = Get-Command $Name -ErrorAction SilentlyContinue
  if ($command) {
    return $command.Source
  }

  $candidates = @(
    "C:\Program Files (x86)\WiX Toolset v3.11\bin\$Name.exe",
    "C:\Program Files\WiX Toolset v3.11\bin\$Name.exe"
  )
  foreach ($candidate in $candidates) {
    if (Test-Path $candidate) {
      return $candidate
    }
  }

  return $null
}

function Invoke-ExternalOrFail {
  param(
    [string]$Tool,
    [string[]]$Args,
    [string]$Step
  )

  & $Tool @Args
  if ($LASTEXITCODE -ne 0) {
    throw "$Step failed with exit code $LASTEXITCODE"
  }
}

function Ensure-WixToolset {
  $heat = Get-WixBinary -Name "heat"
  $candle = Get-WixBinary -Name "candle"
  $light = Get-WixBinary -Name "light"

  if ($heat -and $candle -and $light) {
    return @{
      Heat = $heat
      Candle = $candle
      Light = $light
    }
  }

  $choco = Get-Command choco -ErrorAction SilentlyContinue
  if (-not $choco) {
    Write-Error "WiX binaries missing and Chocolatey is unavailable. Install WiX Toolset v3.11 or run with -SkipMsi."
  }

  Write-Host "[desktop] installing WiX Toolset with Chocolatey"
  Invoke-ExternalOrFail -Tool $choco.Source -Args @("install", "wixtoolset", "--no-progress", "-y") -Step "Chocolatey WiX installation"

  $heat = Get-WixBinary -Name "heat"
  $candle = Get-WixBinary -Name "candle"
  $light = Get-WixBinary -Name "light"

  if (-not ($heat -and $candle -and $light)) {
    Write-Error "WiX installation did not provide heat/candle/light binaries"
  }

  return @{
    Heat = $heat
    Candle = $candle
    Light = $light
  }
}

if (-not (Test-Path $ExePath)) {
  Write-Error "Missing executable artifact: $ExePath"
}
if (-not (Test-Path $WebDir)) {
  Write-Error "Missing web assets directory: $WebDir"
}

$safeVersion = ($Version -replace "[^A-Za-z0-9._-]", "-")
$msiVersion = Normalize-MsiVersion -RawVersion $Version
$ReleaseDir = Join-Path $RootDir "build\releases\windows-x64"
$BundleDir = Join-Path $ReleaseDir ("{0}-{1}-windows-x64" -f $AppName, $safeVersion)

if (Test-Path $BundleDir) {
  Remove-Item -Path $BundleDir -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $BundleDir | Out-Null

Copy-Item -Path $ExePath -Destination (Join-Path $BundleDir "$AppName.exe") -Force
Copy-Item -Path $WebDir -Destination (Join-Path $BundleDir "web") -Recurse -Force

$PortableExe = Join-Path $ReleaseDir ("{0}-{1}-windows-x64.exe" -f $AppName, $safeVersion)
Copy-Item -Path $ExePath -Destination $PortableExe -Force
Copy-Item -Path $PortableExe -Destination (Join-Path $ReleaseDir ("{0}-latest-windows-x64.exe" -f $AppName)) -Force

$ZipPath = Join-Path $ReleaseDir ("{0}-{1}-windows-x64.zip" -f $AppName, $safeVersion)
if (Test-Path $ZipPath) {
  Remove-Item -Path $ZipPath -Force
}
Compress-Archive -Path (Join-Path $BundleDir "*") -DestinationPath $ZipPath -CompressionLevel Optimal -Force
Copy-Item -Path $ZipPath -Destination (Join-Path $ReleaseDir ("{0}-latest-windows-x64.zip" -f $AppName)) -Force

$MsiPath = Join-Path $ReleaseDir ("{0}-{1}-windows-x64.msi" -f $AppName, $safeVersion)
if (-not $SkipMsi) {
  $wix = Ensure-WixToolset

  $WixWorkDir = Join-Path $ReleaseDir ("_wix-{0}" -f $safeVersion)
  if (Test-Path $WixWorkDir) {
    Remove-Item -Path $WixWorkDir -Recurse -Force
  }
  New-Item -ItemType Directory -Force -Path $WixWorkDir | Out-Null

  $productWxs = Join-Path $WixWorkDir "Product.wxs"
  $appFilesWxs = Join-Path $WixWorkDir "AppFiles.wxs"
  $productWixobj = Join-Path $WixWorkDir "Product.wixobj"
  $appFilesWixobj = Join-Path $WixWorkDir "AppFiles.wixobj"

  $productWxsContent = @"
<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
  <Product Id="*"
           Name="$AppName"
           Language="1033"
           Version="$msiVersion"
           Manufacturer="$Manufacturer"
           UpgradeCode="$UpgradeCode">
    <Package InstallerVersion="500" Compressed="yes" InstallScope="perMachine" />
    <MajorUpgrade DowngradeErrorMessage="A newer version of [ProductName] is already installed." />
    <MediaTemplate EmbedCab="yes" CompressionLevel="high" />
    <Icon Id="AppIcon" SourceFile="`$(var.SourceDir)\$AppName.exe" />
    <Property Id="ARPPRODUCTICON" Value="AppIcon" />
    <Feature Id="MainFeature" Title="$AppName" Level="1">
      <ComponentGroupRef Id="AppFiles" />
      <ComponentRef Id="ApplicationShortcuts" />
    </Feature>
    <UIRef Id="WixUI_Minimal" />
  </Product>

  <Fragment>
    <Directory Id="TARGETDIR" Name="SourceDir">
      <Directory Id="ProgramFiles64Folder">
        <Directory Id="INSTALLDIR" Name="$AppName" />
      </Directory>
      <Directory Id="ProgramMenuFolder">
        <Directory Id="ApplicationProgramsFolder" Name="$AppName" />
      </Directory>
      <Directory Id="DesktopFolder" />
    </Directory>
  </Fragment>

  <Fragment>
    <Component Id="ApplicationShortcuts" Directory="ApplicationProgramsFolder" Guid="*" Win64="yes">
      <Shortcut Id="ApplicationStartMenuShortcut"
                Name="$AppName"
                Description="$AppName Desktop"
                Target="[INSTALLDIR]$AppName.exe"
                WorkingDirectory="INSTALLDIR" />
      <Shortcut Id="ApplicationDesktopShortcut"
                Name="$AppName"
                Description="$AppName Desktop"
                Directory="DesktopFolder"
                Target="[INSTALLDIR]$AppName.exe"
                WorkingDirectory="INSTALLDIR" />
      <RemoveFolder Id="ApplicationProgramsFolder" On="uninstall" />
      <RegistryValue Root="HKLM"
                     Key="Software\\Cybergraphe\\$AppName"
                     Name="installed"
                     Type="integer"
                     Value="1"
                     KeyPath="yes" />
    </Component>
  </Fragment>
</Wix>
"@
  Set-Content -Path $productWxs -Value $productWxsContent -Encoding UTF8

  Invoke-ExternalOrFail -Tool $wix["Heat"] -Args @("dir", $BundleDir, "-nologo", "-cg", "AppFiles", "-dr", "INSTALLDIR", "-srd", "-scom", "-sreg", "-sfrag", "-gg", "-var", "var.SourceDir", "-out", $appFilesWxs) -Step "WiX heat"
  Invoke-ExternalOrFail -Tool $wix["Candle"] -Args @("-nologo", "-arch", "x64", "-dSourceDir=$BundleDir", "-out", $productWixobj, $productWxs) -Step "WiX candle (product)"
  Invoke-ExternalOrFail -Tool $wix["Candle"] -Args @("-nologo", "-arch", "x64", "-dSourceDir=$BundleDir", "-out", $appFilesWixobj, $appFilesWxs) -Step "WiX candle (files)"
  if (Test-Path $MsiPath) {
    Remove-Item -Path $MsiPath -Force
  }
  Invoke-ExternalOrFail -Tool $wix["Light"] -Args @("-nologo", "-ext", "WixUIExtension", "-cultures:en-us", "-out", $MsiPath, $productWixobj, $appFilesWixobj) -Step "WiX light"

  if (-not (Test-Path $MsiPath)) {
    Write-Error "MSI artifact was not produced: $MsiPath"
  }
  Copy-Item -Path $MsiPath -Destination (Join-Path $ReleaseDir ("{0}-latest-windows-x64.msi" -f $AppName)) -Force
}

Write-Host "[desktop] windows package artifacts created"
Write-Host "[desktop] portable exe: $PortableExe"
Write-Host "[desktop] bundle zip:  $ZipPath"
if (-not $SkipMsi) {
  Write-Host "[desktop] msi:         $MsiPath"
}
