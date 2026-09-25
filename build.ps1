[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$WorkspaceDir = $PSScriptRoot

Write-Host "VampYTD Local Build" -ForegroundColor Cyan

$goCommand = Get-Command "go.exe" -ErrorAction SilentlyContinue
$goPath = if ($goCommand) { $goCommand.Source } else { $null }
if (-not $goPath -and (Test-Path -LiteralPath "C:\Program Files\Go\bin\go.exe")) {
    $goPath = "C:\Program Files\Go\bin\go.exe"
}
if (-not $goPath) {
    throw "Go compiler was not found. Please install Go to build binaries locally."
}

Push-Location $WorkspaceDir
try {
    Write-Host "Running tests..." -ForegroundColor Gray
    & $goPath test ./...
    if ($LASTEXITCODE -ne 0) { throw "Tests failed with exit code $LASTEXITCODE." }

    Write-Host "Building ytd.exe..." -ForegroundColor Gray
    & $goPath build -o (Join-Path $WorkspaceDir "ytd.exe") ./cmd/ytd
    if ($LASTEXITCODE -ne 0) { throw "Building ytd.exe failed." }

    Write-Host "Building bridge.exe..." -ForegroundColor Gray
    & $goPath build -o (Join-Path $WorkspaceDir "bridge.exe") ./cmd/bridge
    if ($LASTEXITCODE -ne 0) { throw "Building bridge.exe failed." }

    Write-Host "Build completed successfully." -ForegroundColor Green
} finally {
    Pop-Location
}
