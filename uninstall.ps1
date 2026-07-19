# VampYTD Windows 11 Automation Uninstallation Script
# Stops running daemon, deletes startup task/shortcut, removes PATH variable, and removes operational folder.

$ErrorActionPreference = "Stop"

$RunDir = "$env:USERPROFILE\apps\vamp-ytd-extension\run"
$StartupPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\VampYTD-Bridge.lnk"

Write-Host "[VampYTD] Starting uninstallation..." -ForegroundColor Cyan

# 1. Stop bridge process
$Processes = Get-Process -Name "bridge" -ErrorAction SilentlyContinue
if ($Processes) {
    Write-Host "[STOP] Stopping bridge daemon..." -ForegroundColor Yellow
    Stop-Process -Name "bridge" -Force
} else {
    Write-Host "[INFO] No running bridge process found."
}

# 2. Clean up Startup shortcut and Scheduled Task
if (Test-Path $StartupPath) {
    Write-Host "[CLEANUP] Removing startup shortcut..." -ForegroundColor Yellow
    Remove-Item -Path $StartupPath -Force
    Write-Host "[OK] Startup shortcut removed!" -ForegroundColor Green
} else {
    Write-Host "[INFO] No startup shortcut found at $StartupPath."
}

# Clean up Scheduled Task if present
$IsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
$ExistingTask = Get-ScheduledTask -TaskName "VampYTDBridge" -ErrorAction SilentlyContinue
if ($ExistingTask) {
    if ($IsAdmin) {
        Write-Host "[CLEANUP] Removing Scheduled Task 'VampYTDBridge'..." -ForegroundColor Yellow
        Unregister-ScheduledTask -TaskName "VampYTDBridge" -Confirm:$false | Out-Null
        Write-Host "[OK] Scheduled Task removed!" -ForegroundColor Green
    } else {
        Write-Warning "[WARN] Found Scheduled Task 'VampYTDBridge' but uninstallation requires Administrator privileges to remove it."
        Write-Host "[INFO] To remove it, please run uninstall.ps1 from an Administrator PowerShell window." -ForegroundColor Yellow
    }
}

# 3. Remove RunDir from User PATH
Write-Host "[PATH] Cleaning up PATH environment variable..."
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -like "*$RunDir*") {
    # Remove $RunDir and any trailing/leading duplicate semicolons
    $NewPath = $UserPath -replace [Regex]::Escape(";$RunDir"), ""
    $NewPath = $NewPath -replace [Regex]::Escape("$RunDir;"), ""
    $NewPath = $NewPath -replace [Regex]::Escape($RunDir), ""
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    Write-Host "[OK] Removed $RunDir from User PATH!" -ForegroundColor Green
} else {
    Write-Host "[INFO] $RunDir was not in User PATH."
}

# 4. Clean up active runtime folder
if (Test-Path $RunDir) {
    Write-Host "[CLEANUP] Removing active runtime folder at $RunDir..." -ForegroundColor Yellow
    Remove-Item -Path $RunDir -Recurse -Force
    Write-Host "[OK] Runtime folder deleted!" -ForegroundColor Green
} else {
    Write-Host "[INFO] No runtime folder found at $RunDir."
}

Write-Host "`n[SUCCESS] [VampYTD] Uninstallation complete!" -ForegroundColor Green
