<#
.SYNOPSIS
    VampYTD 1-Liner Web Installer for Windows (Chromium).
.DESCRIPTION
    Downloads the latest pre-compiled VampYTD-Setup.exe from GitHub Releases
    and runs the interactive setup.
.EXAMPLE
    irm https://raw.githubusercontent.com/vampdued/vamp-ytd-extension/main/install.ps1 | iex
#>

[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"

if (-not [System.Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([System.Runtime.InteropServices.OSPlatform]::Windows)) {
    Write-Error "VampYTD installer currently supports Windows."
    return
}

Write-Host "========================================================" -ForegroundColor Cyan
Write-Host " VampYTD Web Installer (Windows Chromium)" -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan

$DownloadUrl = "https://github.com/vampdued/vamp-ytd-extension/releases/latest/download/VampYTD-Setup.exe"
$ChecksumsUrl = "https://github.com/vampdued/vamp-ytd-extension/releases/latest/download/checksums.txt"
$TempInstaller = Join-Path $env:TEMP "VampYTD-Setup.exe"
$TempChecksums = Join-Path $env:TEMP "VampYTD-checksums.txt"

foreach ($path in @($TempInstaller, $TempChecksums)) {
    if (Test-Path -LiteralPath $path) {
        Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "`nDownloading latest VampYTD-Setup.exe..." -ForegroundColor Gray
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempInstaller -UseBasicParsing
    Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $TempChecksums -UseBasicParsing
} catch {
    Write-Error "Failed to download installer from GitHub Releases: $_"
    return
}

Write-Host "Verifying installer checksum..." -ForegroundColor Gray
$ExpectedHash = $null
foreach ($line in Get-Content -LiteralPath $TempChecksums) {
    # checksums.txt format: "<sha256>  <filename>" (or "*<filename>")
    if ($line -match '^(?<hash>[0-9a-fA-F]{64})\s+\*?(?<file>\S+)$') {
        if ($Matches['file'] -eq 'VampYTD-Setup.exe') {
            $ExpectedHash = $Matches['hash'].ToLower()
            break
        }
    }
}
if (-not $ExpectedHash) {
    Write-Error "Could not find VampYTD-Setup.exe entry in checksums.txt. Aborting for safety."
    return
}
$ActualHash = (Get-FileHash -LiteralPath $TempInstaller -Algorithm SHA256).Hash.ToLower()
if ($ActualHash -ne $ExpectedHash) {
    Write-Error "Checksum mismatch! Expected $ExpectedHash but got $ActualHash. The installer may be corrupted or tampered with. Aborting."
    return
}
Write-Host "Checksum OK." -ForegroundColor Green

Write-Host "Starting setup..." -ForegroundColor Green
try {
    & $TempInstaller
} finally {
    foreach ($path in @($TempInstaller, $TempChecksums)) {
        if (Test-Path -LiteralPath $path) {
            Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
        }
    }
}
