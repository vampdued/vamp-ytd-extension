[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$WorkspaceDir = $PSScriptRoot
$ExtensionSource = Join-Path $WorkspaceDir "VampYTDExtension"

Write-Host "VampYTD Dev Mode Setup" -ForegroundColor Magenta
Write-Host "Configures your local Git workspace as the active browser extension and native host." -ForegroundColor Gray

# 1. Build binaries locally
Write-Host "`n[1/3] Building local binaries in workspace..." -ForegroundColor Cyan
$goCommand = Get-Command "go.exe" -ErrorAction SilentlyContinue
$goPath = if ($goCommand) { $goCommand.Source } else { $null }
if (-not $goPath -and (Test-Path -LiteralPath "C:\Program Files\Go\bin\go.exe")) {
    $goPath = "C:\Program Files\Go\bin\go.exe"
}
if (-not $goPath) {
    throw "Go compiler was not found in PATH or standard directories. Please install Go to use Dev Mode."
}

Push-Location $WorkspaceDir
try {
    Write-Host "Running unit tests..." -ForegroundColor Gray
    & $goPath test ./...
    if ($LASTEXITCODE -ne 0) { throw "Unit tests failed with exit code $LASTEXITCODE." }

    & $goPath build -o (Join-Path $WorkspaceDir "ytd.exe") ./cmd/ytd
    if ($LASTEXITCODE -ne 0) { throw "Building ytd.exe failed." }
    & $goPath build -o (Join-Path $WorkspaceDir "bridge.exe") ./cmd/bridge
    if ($LASTEXITCODE -ne 0) { throw "Building bridge.exe failed." }
    Write-Host "Unit tests passed and binaries compiled successfully in $WorkspaceDir" -ForegroundColor Green
} finally {
    Pop-Location
}

# 2. Register native host pointing to workspace
Write-Host "`n[2/3] Registering native messaging host to local repository..." -ForegroundColor Cyan
& (Join-Path $WorkspaceDir "bridge.exe") --install-native
if ($LASTEXITCODE -ne 0) { throw "Native messaging registration failed with exit code $LASTEXITCODE." }
Write-Host "Native messaging host pointed to Git workspace." -ForegroundColor Green

# 3. Copy unpacked extension path
Write-Host "`n[3/3] Setting up browser extension..." -ForegroundColor Cyan
try {
    Set-Clipboard -Value $ExtensionSource
    Write-Host "Copied workspace extension path to clipboard: $ExtensionSource" -ForegroundColor Green
} catch {
    Write-Warning "Could not copy path to clipboard automatically."
}

Write-Host "`n========================================================" -ForegroundColor Yellow
Write-Host " [DEV MODE ACTIVE]" -ForegroundColor Yellow
Write-Host "========================================================" -ForegroundColor Yellow
Write-Host "- Native host binary: $(Join-Path $WorkspaceDir 'bridge.exe')"
Write-Host "- Browser extension:  $ExtensionSource"
Write-Host "- To test changes:    Edit files in the repo, then click 'Reload' in chrome://extensions"
Write-Host "- To restore normal:  Run Install-VampYTD.cmd to switch back to %LOCALAPPDATA%\VampYTD"
Write-Host "========================================================`n" -ForegroundColor Yellow
