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
$TempInstaller = Join-Path $env:TEMP "VampYTD-Setup.exe"

if (Test-Path -LiteralPath $TempInstaller) {
    Remove-Item -LiteralPath $TempInstaller -Force -ErrorAction SilentlyContinue
}

Write-Host "`nDownloading latest VampYTD-Setup.exe..." -ForegroundColor Gray
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempInstaller -UseBasicParsing
} catch {
    Write-Error "Failed to download installer from GitHub Releases: $_"
    return
}

Write-Host "Starting setup..." -ForegroundColor Green
try {
    & $TempInstaller
} finally {
    if (Test-Path -LiteralPath $TempInstaller) {
        Remove-Item -LiteralPath $TempInstaller -Force -ErrorAction SilentlyContinue
    }
}
