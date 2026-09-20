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
$ExtensionDestinationFirefox = Join-Path $RunDir "VampYTDExtension-Firefox"
$StartupPath = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Startup\VampYTD-Bridge.lnk"
$SourceBuild = (Test-Path -LiteralPath (Join-Path $WorkspaceDir "cmd\ytd")) -and
    (Test-Path -LiteralPath (Join-Path $WorkspaceDir "cmd\bridge"))

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
Write-Host "Installs or repairs the downloader and browser connection for the current user."

Write-Step "[1/6] Checking required tools"
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

Write-Step "[2/6] Preparing VampYTD binaries"
$BinarySourceDir = $WorkspaceDir
$TemporaryBuildDir = $null
if ($SourceBuild) {
    $goCommand = Get-Command "go.exe" -ErrorAction SilentlyContinue
    $goPath = if ($goCommand) { $goCommand.Source } else { $null }
    if (-not $goPath -and (Test-Path -LiteralPath "C:\Program Files\Go\bin\go.exe")) {
        $goPath = "C:\Program Files\Go\bin\go.exe"
    }
    if (-not $goPath) {
        throw "Go is required for a source checkout. Use a prebuilt release or install Go first."
    }

    $TemporaryBuildDir = Join-Path $env:TEMP ("vampytd-build-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $TemporaryBuildDir | Out-Null
    $BinarySourceDir = $TemporaryBuildDir
    Push-Location $WorkspaceDir
    try {
        & $goPath build -o (Join-Path $TemporaryBuildDir "ytd.exe") ./cmd/ytd
        if ($LASTEXITCODE -ne 0) { throw "Building ytd.exe failed with exit code $LASTEXITCODE." }
        & $goPath build -o (Join-Path $TemporaryBuildDir "bridge.exe") ./cmd/bridge
        if ($LASTEXITCODE -ne 0) { throw "Building bridge.exe failed with exit code $LASTEXITCODE." }
    } finally {
        Pop-Location
    }
    Write-Host "Source build completed." -ForegroundColor Green
} elseif (-not (Test-Path -LiteralPath (Join-Path $WorkspaceDir "ytd.exe")) -or
          -not (Test-Path -LiteralPath (Join-Path $WorkspaceDir "bridge.exe"))) {
    throw "This release is missing ytd.exe or bridge.exe. Download the complete Windows archive."
} else {
    Write-Host "Using the included release binaries; Go is not required." -ForegroundColor Green
}

try {
    Write-Step "[3/6] Installing application files"
    New-Item -ItemType Directory -Path $RunDir -Force | Out-Null
    New-Item -ItemType Directory -Path $ExtensionDestination -Force | Out-Null
    New-Item -ItemType Directory -Path $ExtensionDestinationFirefox -Force | Out-Null

    $processes = Get-CimInstance Win32_Process -Filter "Name = 'bridge.exe'" -ErrorAction SilentlyContinue |
        Where-Object { $_.ExecutablePath -eq (Join-Path $RunDir "bridge.exe") }
    $processes | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }

    Copy-Item -LiteralPath (Join-Path $BinarySourceDir "ytd.exe") -Destination (Join-Path $RunDir "ytd.exe") -Force
    Copy-Item -LiteralPath (Join-Path $BinarySourceDir "bridge.exe") -Destination (Join-Path $RunDir "bridge.exe") -Force

    # 1. Chromium extension directory (uses service_worker)
    Copy-Item -Path (Join-Path $ExtensionSource "*") -Destination $ExtensionDestination -Recurse -Force
    if (Test-Path -LiteralPath (Join-Path $ExtensionDestination "manifest.firefox.json")) {
        Remove-Item -LiteralPath (Join-Path $ExtensionDestination "manifest.firefox.json") -Force
    }

    # 2. Firefox extension directory (uses background.scripts)
    Copy-Item -Path (Join-Path $ExtensionSource "*") -Destination $ExtensionDestinationFirefox -Recurse -Force
    Copy-Item -LiteralPath (Join-Path $ExtensionSource "manifest.firefox.json") -Destination (Join-Path $ExtensionDestinationFirefox "manifest.json") -Force
    if (Test-Path -LiteralPath (Join-Path $ExtensionDestinationFirefox "manifest.firefox.json")) {
        Remove-Item -LiteralPath (Join-Path $ExtensionDestinationFirefox "manifest.firefox.json") -Force
    }

    # 3. Package Firefox archive from the Firefox directory
    $ZipDestination = Join-Path $RunDir "VampYTD-Extension.zip"
    $XpiDestination = Join-Path $RunDir "VampYTD-Extension.xpi"
    if (Test-Path -LiteralPath $ZipDestination) { Remove-Item -LiteralPath $ZipDestination -Force -ErrorAction SilentlyContinue }
    if (Test-Path -LiteralPath $XpiDestination) { Remove-Item -LiteralPath $XpiDestination -Force -ErrorAction SilentlyContinue }

    Add-Type -AssemblyName System.IO.Compression
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zipStream = [System.IO.File]::Create($ZipDestination)
    $zipArch = [System.IO.Compression.ZipArchive]::new($zipStream, [System.IO.Compression.ZipArchiveMode]::Create)
    $extFiles = Get-ChildItem -Path $ExtensionDestinationFirefox -Recurse -File
    foreach ($f in $extFiles) {
        $rel = $f.FullName.Substring($ExtensionDestinationFirefox.Length).TrimStart("\", "/").Replace("\", "/")
        [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zipArch, $f.FullName, $rel) | Out-Null
    }
    $zipArch.Dispose()
    $zipStream.Dispose()
    Copy-Item -LiteralPath $ZipDestination -Destination $XpiDestination -Force

    Write-Host "Installed to $RunDir" -ForegroundColor Green
} finally {
    if ($TemporaryBuildDir -and (Test-Path -LiteralPath $TemporaryBuildDir)) {
        Remove-Item -LiteralPath $TemporaryBuildDir -Recurse -Force
    }
}

Write-Step "[4/6] Adding the command to your user PATH"
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

Write-Step "[5/6] Registering the browser connection"
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
    "HKCU:\Software\Vivaldi\NativeMessagingHosts\com.vampytd.bridge",
    "HKCU:\Software\Mozilla\NativeMessagingHosts\com.vampytd.bridge"
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
    throw "The native host registration could not be verified in any supported browser registry key."
}
Write-Host "Browser connection registered successfully." -ForegroundColor Green

Write-Step "[6/6] Finishing extension setup"
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
Write-Host "Firefox package:  $XpiDestination"

Write-Host "`nTo activate in Chromium browsers (Chrome, Edge, Brave, Vivaldi):" -ForegroundColor Cyan
Write-Host "  1. Open chrome://extensions (or edge://extensions / brave://extensions)."
Write-Host "  2. Enable 'Developer mode'."
Write-Host "  3. Click 'Load unpacked' and select the extension folder (copied to clipboard)."

Write-Host "`nTo activate in Firefox:" -ForegroundColor Cyan
Write-Host "  Method 1 (All Firefox versions - Developer / Temporary):" -ForegroundColor White
Write-Host "    1. Open about:debugging#/runtime/this-firefox"
Write-Host "    2. Click 'Load Temporary Add-on...'"
Write-Host "    3. Select manifest.json inside: $ExtensionDestinationFirefox"
Write-Host "  Method 2 (Firefox Developer Edition / ESR / Unbranded - Install from File):" -ForegroundColor White
Write-Host "    1. Open about:addons and click the gear icon."
Write-Host "    2. Select 'Install Add-on From File...' and choose: $XpiDestination"
Write-Host "    (Note: Standard retail Firefox requires xpinstall.signatures.required=false in about:config for unsigned .xpi files)." -ForegroundColor DarkGray

if ($missingReq.Count -gt 0) {
    Write-Warning "`nDownloads will not work until these tools are installed: $($missingReq.Name -join ', ')."
} else {
    Write-Host "`nAll required tools are ready. Reload the extension and open its Status tab to verify." -ForegroundColor Green
}
if ($missingFzf.Count -gt 0) {
    Write-Host "Note: FZF is not installed. Interactive format selection will fall back to numbered lists." -ForegroundColor DarkGray
}

