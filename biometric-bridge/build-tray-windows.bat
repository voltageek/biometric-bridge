@echo off
REM build-tray-windows.bat — Build tray app natively on Windows
REM
REM Prerequisites:
REM   - Go 1.24.7 or later
REM   - GCC (MinGW-w64 or TDM-GCC)
REM   - Fyne dependencies (automatically handled by Go modules)
REM
REM Usage:
REM   build-tray-windows.bat

setlocal enabledelayedexpansion

set "OUTPUT=bridge-tray.exe"

echo ================================================================================
echo Building Tray App for Windows (x64)
echo ================================================================================
echo.

echo Checking prerequisites...

where go >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: Go not found in PATH
    echo Please install Go from https://go.dev/dl/
    pause
    exit /b 1
)

where gcc >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: GCC not found in PATH
    echo Please install MinGW-w64 from:
    echo   https://www.mingw-w64.org/downloads/
    echo Or TDM-GCC from:
    echo   https://jmeubank.github.io/tdm-gcc/
    pause
    exit /b 1
)

echo Prerequisites OK
echo.

echo Building tray app...
echo This may take 2-5 minutes on first build...
echo.

set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

REM Build with GUI subsystem (no console window)
go build -ldflags="-H windowsgui" -o "%OUTPUT%" ./cmd/tray

if exist "%OUTPUT%" (
    echo.
    echo ================================================================================
    echo Build successful: %OUTPUT%
    echo ================================================================================
    echo.
    echo Next steps:
    echo   1. Run bridge-tray.exe to test
    echo   2. Copy to desired location
    echo   3. Add to Startup folder for auto-start:
    echo      Copy to: %%APPDATA%%\Microsoft\Windows\Start Menu\Programs\Startup
    echo ================================================================================
) else (
    echo.
    echo Build failed
    echo Check that all prerequisites are installed correctly
    pause
    exit /b 1
)

pause
