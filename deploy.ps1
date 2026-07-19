# VampYTD Windows 11 Automation Setup Script
# Compiles Go backend (ytd and bridge), syncs files, registers background task, and verifies health.

$ErrorActionPreference = "Stop"

# Define paths
$WorkspaceDir = $PSScriptRoot
$RunDir = "$env:USERPROFILE\apps\vamp-ytd-extension\run"
$StartupPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\VampYTD-Bridge.lnk"

# Ports/Hosts
$Port = 8080
$HostAddr = "localhost"

# Override from environment variables if present
if ($env:VAMPYTD_PORT) { $Port = $env:VAMPYTD_PORT }
elseif ($env:PORT) { $Port = $env:PORT }

if ($env:VAMPYTD_HOST) { $HostAddr = $env:VAMPYTD_HOST }
elseif ($env:HOST) { $HostAddr = $env:HOST }

Write-Host "[VampYTD] Starting Windows 11 automated setup and deployment..." -ForegroundColor Cyan

# 1. Check and Install Go
Write-Host "`n[1/5] Checking Go environment..." -ForegroundColor Blue
$GoPath = (Get-Command go -ErrorAction SilentlyContinue) | Select-Object -ExpandProperty Source -ErrorAction SilentlyContinue
if (-not $GoPath -and (Test-Path "C:\Program Files\Go\bin\go.exe")) {
    $GoPath = "C:\Program Files\Go\bin\go.exe"
}

if (-not $GoPath) {
    Write-Host "[WARN] Go is not found on your path. Attempting auto-installation using winget..." -ForegroundColor Yellow
    
    # Check if winget is available
    $WingetCheck = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $WingetCheck) {
        Write-Error "[ERROR] winget is not available on this system. Please install winget or install Go manually from https://go.dev/dl/"
        exit 1
    }
    
    Write-Host "Installing Go (Golang) via winget, please wait..." -ForegroundColor Yellow
    Start-Process -FilePath "winget" -ArgumentList "install -e --id GoLang.Go --accept-source-agreements --accept-package-agreements" -NoNewWindow -Wait
    
    # Verify the installation path
    $GoPath = "C:\Program Files\Go\bin\go.exe"
    if (-not (Test-Path $GoPath)) {
        Write-Error "[ERROR] Go installation failed or requires a system restart. Please install Go manually and rerun this script."
        exit 1
    }
    Write-Host "[OK] Go installed successfully via winget!" -ForegroundColor Green
} else {
    Write-Host "[OK] Go compiler found: $GoPath" -ForegroundColor Green
}

# Update temporary PATH for the current session to ensure Go is usable
$GoDir = Split-Path $GoPath
if ($env:Path -notlike "*$GoDir*") {
    $env:Path += ";$GoDir"
}

# 2. Compile Golang Server
Write-Host "`n[2/5] Compiling Go binaries (ytd and bridge)..." -ForegroundColor Blue
Push-Location "$WorkspaceDir"
try {
    Write-Host "   - Fetching Go modules..."
    & go mod download
    Write-Host "   - Compiling ytd.exe..."
    & go build -o ytd.exe main.go
    Write-Host "   - Compiling bridge.exe..."
    & go build -o bridge.exe bridge.go
    Write-Host "[OK] Compilation succeeded!" -ForegroundColor Green
} catch {
    Write-Error "[ERROR] Go compilation failed! Please check your Go installation."
    Pop-Location
    exit 1
}
Pop-Location

# 3. Sync Binary and Assets
Write-Host "`n[3/5] Syncing compiled files to runtime path..." -ForegroundColor Blue

# Stop any running process first to ensure we can overwrite the binary files
$Processes = Get-Process -Name "bridge" -ErrorAction SilentlyContinue
if ($Processes) {
    Write-Host "   - Terminating running bridge daemon instances..."
    Stop-Process -Name "bridge" -Force
}

if (-not (Test-Path $RunDir)) {
    New-Item -ItemType Directory -Force -Path $RunDir | Out-Null
}
Copy-Item -Path "$WorkspaceDir\ytd.exe" -Destination "$RunDir\ytd.exe" -Force
Copy-Item -Path "$WorkspaceDir\bridge.exe" -Destination "$RunDir\bridge.exe" -Force

# Chrome Extension Assets sync
$UiDest = "$RunDir\VampYTDExtension"
if (-not (Test-Path $UiDest)) {
    New-Item -ItemType Directory -Force -Path $UiDest | Out-Null
}
Copy-Item -Path "$WorkspaceDir\VampYTDExtension\*" -Destination $UiDest -Recurse -Force
Write-Host "[OK] Files synchronized to operational folder: $RunDir" -ForegroundColor Green

# Add RunDir to User PATH for terminal command availability
Write-Host "   - Adding $RunDir to User PATH env variable..."
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$RunDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$RunDir", "User")
    $env:Path += ";$RunDir"
    Write-Host "[OK] Added to User PATH successfully! Open a new terminal to run 'ytd' anywhere." -ForegroundColor Green
} else {
    Write-Host "[OK] $RunDir is already in User PATH." -ForegroundColor Green
}

# 4. Register background task (Startup Folder or Scheduled Task)
Write-Host "`n[4/5] Checking and registering background startup task..." -ForegroundColor Blue

# Clean up existing Task Scheduler task if it was registered previously to prevent conflicts
$IsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
$ExistingTask = Get-ScheduledTask -TaskName "VampYTDBridge" -ErrorAction SilentlyContinue
if ($ExistingTask) {
    if ($IsAdmin) {
        Write-Host "   - Removing existing Task Scheduler task..."
        Unregister-ScheduledTask -TaskName "VampYTDBridge" -Confirm:$false | Out-Null
    } else {
        Write-Warning "   - Found existing Task Scheduler task 'VampYTDBridge' but cannot remove it without Administrator privileges."
    }
}

# Clean up existing startup shortcut if present
if (Test-Path $StartupPath) {
    Write-Host "   - Removing existing startup shortcut..."
    Remove-Item -Path $StartupPath -Force
}



# Create hidden startup VBScript to run bridge detached and completely silent
$VbsPath = "$RunDir\start_bridge_hidden.vbs"
$VbsContent = @"
Set objShell = CreateObject("WScript.Shell")
objShell.CurrentDirectory = "$RunDir"
objShell.Run """$RunDir\bridge.exe""", 0, False
"@
Set-Content -Path $VbsPath -Value $VbsContent -Encoding ASCII

if ($IsAdmin) {
    Write-Host "   - Administrator privileges detected. Registering Scheduled Task as background service..."
    $Action = New-ScheduledTaskAction -Execute "wscript.exe" -Argument "`"$VbsPath`""
    $Trigger = New-ScheduledTaskTrigger -AtLogOn
    $Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    Register-ScheduledTask -TaskName "VampYTDBridge" -Action $Action -Trigger $Trigger -Settings $Settings -Description "VampYTD Browser Bridge Server" -Force
    Write-Host "[OK] Scheduled Task 'VampYTDBridge' successfully registered to run on login!" -ForegroundColor Green
} else {
    Write-Host "   - Standard user privileges detected. Registering background startup shortcut..."
    Write-Host "     (Run as Administrator to register as a system Scheduled Task instead)" -ForegroundColor Gray
    $WshShell = New-Object -ComObject WScript.Shell
    $Shortcut = $WshShell.CreateShortcut($StartupPath)
    $Shortcut.TargetPath = "wscript.exe"
    $Shortcut.Arguments = "`"$VbsPath`""
    $Shortcut.WindowStyle = 7 # Minimized/Hidden
    $Shortcut.Save()
    Write-Host "[OK] Background startup shortcut successfully registered!" -ForegroundColor Green
}

# Start the task now so it runs immediately and remains online after terminal exits
Write-Host "   - Launching daemon..."
Start-Process -FilePath "wscript.exe" -ArgumentList "`"$VbsPath`""

# 5. Extension load helpers and health checks
Write-Host "`n[5/5] Checking endpoint health and opening extension..." -ForegroundColor Blue

# Attempt to launch Google Chrome extensions page
$ChromePath = "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe"
if (-not (Test-Path $ChromePath)) { $ChromePath = "${env:ProgramFiles}\Google\Chrome\Application\chrome.exe" }
if (-not (Test-Path $ChromePath)) { $ChromePath = "$env:LocalAppData\Google\Chrome\Application\chrome.exe" }

if (Test-Path $ChromePath) {
    Write-Host "   - Opening Chrome to extensions page..."
    Start-Process -FilePath $ChromePath -ArgumentList "chrome://extensions/"
} else {
    Write-Host "[INFO] Please open Chrome and navigate to: chrome://extensions/" -ForegroundColor Yellow
}

# Open workspace folder in explorer so the user can easily drag-and-drop
Write-Host "   - Opening extension source folder in File Explorer..."
Start-Process -FilePath "explorer.exe" -ArgumentList "`"$WorkspaceDir`""
Write-Host "[INFO] Inside Chrome extensions page: enable 'Developer mode' (top right), then drag and drop the 'VampYTDExtension' folder into Chrome!" -ForegroundColor Yellow

# Endpoint health check
$HealthHost = "127.0.0.1"
if ($HostAddr -ne "0.0.0.0" -and $HostAddr -ne "localhost") {
    $HealthHost = $HostAddr
}
$HealthUrl = "http://${HealthHost}:${Port}/"
$MaxAttempts = 10
$Success = $false

# Small pause to allow the daemon to spin up
Start-Sleep -Milliseconds 800

for ($i = 1; $i -le $MaxAttempts; $i++) {
    Write-Host "   - Pinging $HealthUrl (Attempt $i/$MaxAttempts)..."
    try {
        $Response = Invoke-RestMethod -Uri $HealthUrl -Method Get -TimeoutSec 2 -ErrorAction Stop
        if ($Response) {
            $Success = $true
            break
        }
    } catch {
        # Retry
    }
    Start-Sleep -Seconds 1
}

if ($Success) {
    Write-Host "`n[SUCCESS] [VampYTD] App is alive and responding!" -ForegroundColor Green
    if ($HostAddr -eq "0.0.0.0") {
        Write-Host "[INFO] Deployment complete! UI/Bridge is available at http://localhost:${Port} and on your local network." -ForegroundColor Green
    } else {
        Write-Host "[INFO] Deployment complete! UI/Bridge is available at http://${HostAddr}:${Port}" -ForegroundColor Green
    }
} else {
    Write-Warning "`n[ERROR] [VampYTD] Health check failed! The server is not responding at ${HealthHost}:${Port}."
    Write-Host "[INFO] To debug, try starting the server manually from: $RunDir\bridge.exe" -ForegroundColor Yellow
}
