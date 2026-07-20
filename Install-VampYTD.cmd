@echo off
setlocal
title VampYTD Installer
echo Starting VampYTD setup...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0deploy.ps1" -InstallDependencies
set "setupExit=%ERRORLEVEL%"
echo.
if not "%setupExit%"=="0" echo Setup did not finish successfully. Review the message above.
pause
exit /b %setupExit%
