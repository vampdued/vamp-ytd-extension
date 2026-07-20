@echo off
setlocal
title VampYTD Uninstaller
choice /C YN /N /M "Remove VampYTD from this computer? [Y/N] "
if errorlevel 2 exit /b 0
echo Starting VampYTD removal...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall.ps1"
set "setupExit=%ERRORLEVEL%"
echo.
pause
exit /b %setupExit%
