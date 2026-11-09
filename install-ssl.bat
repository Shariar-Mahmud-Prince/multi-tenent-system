@echo off
:: Check for administrative permissions
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo =======================================================
    echo ERROR: Administrator privileges required!
    echo Please right-click this file and choose:
    echo "Run as administrator"
    echo =======================================================
    pause
    exit /b 1
)

echo Installing Caddy Local Root Certificate into Windows Trusted Root Store...
certutil -addstore -f "Root" "%~dp0caddy-root-ca.crt"

echo.
echo =======================================================
echo SUCCESS! Caddy Local Root Certificate is installed!
echo.
echo 1. Close and reopen Google Chrome.
echo 2. Visit https://testdir.local or https://ecom.local
echo You will see the secure green/gray padlock.
echo =======================================================
echo.
pause
