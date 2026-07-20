$ErrorActionPreference = "Stop"

$RunDir = [IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA "VampYTD"))
$LocalAppDataRoot = [IO.Path]::GetFullPath($env:LOCALAPPDATA).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
$StartupPath = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Startup\VampYTD-Bridge.lnk"
$NativeHostName = "com.vampytd.bridge"

if (-not $RunDir.StartsWith($LocalAppDataRoot, [StringComparison]::OrdinalIgnoreCase) -or
    (Split-Path -Leaf $RunDir) -ne "VampYTD") {
    throw "Refusing to remove an unexpected installation path: $RunDir"
}

Write-Host "VampYTD Uninstaller" -ForegroundColor Magenta

Write-Host "`n[1/4] Removing browser integration" -ForegroundColor Cyan
$bridgePath = Join-Path $RunDir "bridge.exe"
if (Test-Path -LiteralPath $bridgePath) {
    & $bridgePath --uninstall-native
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "The bridge cleanup returned exit code $LASTEXITCODE; registry cleanup will continue."
    }
}

$registryRoots = @(
    "HKCU:\Software\Google\Chrome\NativeMessagingHosts",
    "HKCU:\Software\Microsoft\Edge\NativeMessagingHosts",
    "HKCU:\Software\Chromium\NativeMessagingHosts",
    "HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts",
    "HKCU:\Software\Vivaldi\NativeMessagingHosts"
)
foreach ($root in $registryRoots) {
    $registration = Join-Path $root $NativeHostName
    if (Test-Path -LiteralPath $registration) {
        Remove-Item -LiteralPath $registration -Recurse -Force
    }
}

Write-Host "`n[2/4] Removing obsolete startup entries" -ForegroundColor Cyan
if (Test-Path -LiteralPath $StartupPath) {
    Remove-Item -LiteralPath $StartupPath -Force
}
$existingTask = Get-ScheduledTask -TaskName "VampYTDBridge" -ErrorAction SilentlyContinue
if ($existingTask) {
    $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
        [Security.Principal.WindowsBuiltInRole]::Administrator
    )
    if ($isAdmin) {
        Unregister-ScheduledTask -TaskName "VampYTDBridge" -Confirm:$false | Out-Null
    } else {
        Write-Warning "The obsolete administrator-created VampYTDBridge task must be removed from an Administrator PowerShell window."
    }
}

Write-Host "`n[3/4] Cleaning the user PATH" -ForegroundColor Cyan
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$pathEntries = @($userPath -split ";" | Where-Object {
    $_ -and -not $_.Equals($RunDir, [StringComparison]::OrdinalIgnoreCase)
})
[Environment]::SetEnvironmentVariable("Path", ($pathEntries -join ";"), "User")

Write-Host "`n[4/4] Removing installed files" -ForegroundColor Cyan
$processes = Get-CimInstance Win32_Process -Filter "Name = 'bridge.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -eq $bridgePath }
$processes | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
if (Test-Path -LiteralPath $RunDir) {
    Remove-Item -LiteralPath $RunDir -Recurse -Force
}

Write-Host "`nVampYTD was removed successfully." -ForegroundColor Green
Write-Host "Remove the unpacked browser extension from the browser's extensions page if it is still listed." -ForegroundColor Yellow
