[CmdletBinding()]
param(
    [switch]$SkipInstaller
)

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

$PayloadDir = Join-Path $WorkspaceDir "cmd\installer\payload"

Push-Location $WorkspaceDir
try {
    Write-Host "[1/3] Running tests..." -ForegroundColor Gray
    & $goPath test ./...
    if ($LASTEXITCODE -ne 0) { throw "Tests failed with exit code $LASTEXITCODE." }

    Write-Host "[2/3] Building core binaries (ytd.exe & bridge.exe)..." -ForegroundColor Gray
    & $goPath build -o (Join-Path $WorkspaceDir "ytd.exe") ./cmd/ytd
    if ($LASTEXITCODE -ne 0) { throw "Building ytd.exe failed." }

    & $goPath build -o (Join-Path $WorkspaceDir "bridge.exe") ./cmd/bridge
    if ($LASTEXITCODE -ne 0) { throw "Building bridge.exe failed." }

    if (-not $SkipInstaller) {
        Write-Host "[3/3] Staging payload and building VampYTD-Setup.exe..." -ForegroundColor Gray
        New-Item -ItemType Directory -Path $PayloadDir -Force | Out-Null
        Copy-Item -LiteralPath (Join-Path $WorkspaceDir "ytd.exe") -Destination (Join-Path $PayloadDir "ytd.exe") -Force
        Copy-Item -LiteralPath (Join-Path $WorkspaceDir "bridge.exe") -Destination (Join-Path $PayloadDir "bridge.exe") -Force

        $ExtPayload = Join-Path $PayloadDir "extension"
        if (Test-Path -LiteralPath $ExtPayload) {
            Remove-Item -LiteralPath $ExtPayload -Recurse -Force -ErrorAction SilentlyContinue
        }
        New-Item -ItemType Directory -Path $ExtPayload -Force | Out-Null
        Copy-Item -Path (Join-Path $WorkspaceDir "VampYTDExtension\*") -Destination $ExtPayload -Recurse -Force
        if (Test-Path -LiteralPath (Join-Path $ExtPayload "manifest.firefox.json")) {
            Remove-Item -LiteralPath (Join-Path $ExtPayload "manifest.firefox.json") -Force
        }

        & $goPath build -o (Join-Path $WorkspaceDir "VampYTD-Setup.exe") ./cmd/installer
        if ($LASTEXITCODE -ne 0) { throw "Building VampYTD-Setup.exe failed." }
    }

    Write-Host "`nBuild completed successfully." -ForegroundColor Green
    Write-Host "Generated binaries:"
    Write-Host "  - $(Join-Path $WorkspaceDir 'ytd.exe')"
    Write-Host "  - $(Join-Path $WorkspaceDir 'bridge.exe')"
    if (-not $SkipInstaller) {
        Write-Host "  - $(Join-Path $WorkspaceDir 'VampYTD-Setup.exe')"
    }
} finally {
    # Clean up staged binaries from payload folder, keeping placeholder.txt
    if (Test-Path -LiteralPath (Join-Path $PayloadDir "ytd.exe")) {
        Remove-Item -LiteralPath (Join-Path $PayloadDir "ytd.exe") -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path -LiteralPath (Join-Path $PayloadDir "bridge.exe")) {
        Remove-Item -LiteralPath (Join-Path $PayloadDir "bridge.exe") -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path -LiteralPath (Join-Path $PayloadDir "extension")) {
        Remove-Item -LiteralPath (Join-Path $PayloadDir "extension") -Recurse -Force -ErrorAction SilentlyContinue
    }
    Pop-Location
}
