[CmdletBinding()]
param(
    [switch]$InstallDependencies,
    [switch]$InstallFZF
)

$ErrorActionPreference = "Stop"
$WorkspaceDir = $PSScriptRoot
$RunDir = Join-Path $env:LOCALAPPDATA "VampYTD"
$ExtensionSource = Join-Path $WorkspaceDir "VampYTDExtension"
$ExtensionDestination = Join-Path $RunDir "VampYTDExtension"
$StartupPath = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Startup\VampYTD-Bridge.lnk"

$RequiredDependencies = @(
    [pscustomobject]@{ Name = "yt-dlp"; Command = "yt-dlp"; Package = "yt-dlp.yt-dlp" },
    [pscustomobject]@{ Name = "FFmpeg"; Command = "ffmpeg"; Package = "Gyan.FFmpeg" },
    [pscustomobject]@{ Name = "Node.js"; Command = "node"; Package = "OpenJS.NodeJS.LTS" }
)
$OptionalDependencies = @(
    [pscustomobject]@{ Name = "FZF"; Command = "fzf"; Package = "junegunn.fzf" }
)

function Write-Step([string]$Text) {
    Write-Host "`n$Text" -ForegroundColor Cyan
}

function Refresh-ProcessPath {
    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = "$machinePath;$userPath"
}

function Get-MissingDependencies {
    param([array]$List)
    return @($List | Where-Object {
        -not (Get-Command $_.Command -ErrorAction SilentlyContinue)
    })
}

function Install-MissingDependencies {
    param([array]$Missing)

    if ($Missing.Count -eq 0) { return }
    $winget = Get-Command "winget.exe" -ErrorAction SilentlyContinue
    if (-not $winget) {
        Write-Warning "Windows Package Manager (winget) is unavailable, so dependencies cannot be installed automatically."
        return
    }

    foreach ($dependency in $Missing) {
        Write-Host "Installing $($dependency.Name)..." -ForegroundColor Blue
        & $winget.Path install --exact --id $dependency.Package --accept-package-agreements --accept-source-agreements --silent --disable-interactivity
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Automatic installation of $($dependency.Name) returned exit code $LASTEXITCODE."
        }
    }
    Refresh-ProcessPath
}

Write-Host "VampYTD Setup" -ForegroundColor Magenta
Write-Host "Installs or repairs the downloader and browser connection for Windows & Chromium browsers."

Write-Step "[1/5] Checking required tools"
$missingRequired = Get-MissingDependencies -List $RequiredDependencies
$toInstall = @()

if ($InstallDependencies) {
    $toInstall += $missingRequired
}
if ($InstallDependencies -or $InstallFZF) {
    $missingOptional = Get-MissingDependencies -List $OptionalDependencies
    $toInstall += $missingOptional
}

if ($toInstall.Count -gt 0) {
    Install-MissingDependencies -Missing $toInstall
    $missingRequired = Get-MissingDependencies -List $RequiredDependencies
}

if ($missingRequired.Count -eq 0) {
    Write-Host "All required tools are available." -ForegroundColor Green
} else {
    Write-Warning "Missing required tools: $($missingRequired.Name -join ', '). Run Install-VampYTD.cmd for automatic setup."
}

Write-Step "[2/5] Preparing VampYTD binaries"
if (-not (Test-Path -LiteralPath (Join-Path $WorkspaceDir "ytd.exe")) -or
    -not (Test-Path -LiteralPath (Join-Path $WorkspaceDir "bridge.exe"))) {
    throw "This installation archive is missing ytd.exe or bridge.exe. Please download the complete release archive from GitHub Releases."
}
Write-Host "Found pre-built VampYTD binaries." -ForegroundColor Green

Write-Step "[3/5] Installing application files"
New-Item -ItemType Directory -Path $RunDir -Force | Out-Null
New-Item -ItemType Directory -Path $ExtensionDestination -Force | Out-Null

$processes = Get-CimInstance Win32_Process -Filter "Name = 'bridge.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -eq (Join-Path $RunDir "bridge.exe") }
$processes | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }

Copy-Item -LiteralPath (Join-Path $WorkspaceDir "ytd.exe") -Destination (Join-Path $RunDir "ytd.exe") -Force
Copy-Item -LiteralPath (Join-Path $WorkspaceDir "bridge.exe") -Destination (Join-Path $RunDir "bridge.exe") -Force

# Copy Chromium extension directory
Copy-Item -Path (Join-Path $ExtensionSource "*") -Destination $ExtensionDestination -Recurse -Force
if (Test-Path -LiteralPath (Join-Path $ExtensionDestination "manifest.firefox.json")) {
    Remove-Item -LiteralPath (Join-Path $ExtensionDestination "manifest.firefox.json") -Force
}

Write-Host "Installed to $RunDir" -ForegroundColor Green

Write-Step "[4/5] Adding the command to your user PATH"
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$pathEntries = @($userPath -split ";" | Where-Object { $_ })
if ($pathEntries -notcontains $RunDir) {
    $newUserPath = (@($pathEntries) + $RunDir) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
    Refresh-ProcessPath
    Write-Host "Added VampYTD to PATH." -ForegroundColor Green
} else {
    Write-Host "PATH is already configured." -ForegroundColor Green
}

Write-Step "[5/5] Registering the browser connection"
if (Test-Path -LiteralPath $StartupPath) {
    Remove-Item -LiteralPath $StartupPath -Force -ErrorAction SilentlyContinue
}
try {
    $existingTask = Get-ScheduledTask -TaskName "VampYTDBridge" -ErrorAction SilentlyContinue
    if ($existingTask) {
        $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
            [Security.Principal.WindowsBuiltInRole]::Administrator
        )
        if ($isAdmin) {
            Unregister-ScheduledTask -TaskName "VampYTDBridge" -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
        }
    }
} catch {}

& (Join-Path $RunDir "bridge.exe") --install-native
if ($LASTEXITCODE -ne 0) { throw "Native messaging registration failed with exit code $LASTEXITCODE." }

$browserKeys = @(
    "HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.vampytd.bridge",
    "HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.vampytd.bridge",
    "HKCU:\Software\Chromium\NativeMessagingHosts\com.vampytd.bridge",
    "HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\com.vampytd.bridge",
    "HKCU:\Software\Vivaldi\NativeMessagingHosts\com.vampytd.bridge"
)
$verified = $false
foreach ($key in $browserKeys) {
    $reg = Get-ItemPropertyValue -LiteralPath $key -Name "(default)" -ErrorAction SilentlyContinue
    if ($reg -and (Test-Path -LiteralPath $reg)) {
        $verified = $true
        break
    }
}
if (-not $verified) {
    throw "The native host registration could not be verified in any supported Chromium browser registry key."
}
Write-Host "Browser connection registered successfully." -ForegroundColor Green

try {
    Set-Clipboard -Value $ExtensionDestination
    Write-Host "Copied extension folder path to clipboard." -ForegroundColor Green
} catch {
    Write-Warning "The extension path could not be copied to the clipboard."
}

$missingReq = Get-MissingDependencies -List $RequiredDependencies
$missingFzf = Get-MissingDependencies -List $OptionalDependencies

Write-Host "`nInstallation complete." -ForegroundColor Green
Write-Host "Extension folder: $ExtensionDestination"

Write-Host "`nTo activate in Chromium browsers (Chrome, Edge, Brave, Vivaldi):" -ForegroundColor Cyan
Write-Host "  1. Open chrome://extensions (or edge://extensions / brave://extensions)."
Write-Host "  2. Enable 'Developer mode' in the top right."
Write-Host "  3. Click 'Load unpacked' and select the extension folder (copied to clipboard)."

if ($missingReq.Count -gt 0) {
    Write-Warning "`nDownloads will not work until these tools are installed: $($missingReq.Name -join ', ')."
} else {
    Write-Host "`nAll required tools are ready. Reload the extension and open its Status tab to verify." -ForegroundColor Green
}
if ($missingFzf.Count -gt 0) {
    Write-Host "Note: FZF is not installed. Interactive format selection will fall back to numbered lists." -ForegroundColor DarkGray
}
