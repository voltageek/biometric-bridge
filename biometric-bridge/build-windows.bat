@echo off
REM build-windows.bat — Build biometric-bridge natively on Windows.
REM
REM Prerequisites:
REM   - Go 1.24.7 or later
REM   - GCC (via MinGW-w64 or similar)
REM   - Windows SDK DLLs in ..\RealScanSDK for Windows_v2.2.0.2311\x64\
REM
REM Usage:
REM   build-windows.bat

setlocal enabledelayedexpansion

set "OUTPUT=bridge-realscan.exe"
set "SDK_DIR=..\RealScanSDK for Windows_v2.2.0.2311"
set "SDK_LIB_DIR=%SDK_DIR%\Bin\x64"

echo ================================================================================
echo Building biometric-bridge for Windows (x64)
echo ================================================================================
echo.

echo Checking prerequisites...

where go >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: Go not found in PATH
    echo Please install Go from https://go.dev/dl/
    exit /b 1
)

where gcc >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: GCC not found in PATH
    echo Please install MinGW-w64 or similar
    exit /b 1
)

if not exist "%SDK_LIB_DIR%" (
    echo Error: Windows SDK not found at %SDK_LIB_DIR%
    echo Expected structure: %SDK_DIR%\Bin\x64\*.dll
    exit /b 1
)

if not exist "%SDK_LIB_DIR%\RS_SDK.dll" (
    echo Error: RS_SDK.dll not found in %SDK_LIB_DIR%
    exit /b 1
)

echo Prerequisites OK
echo.

echo Building...
echo.

set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

go build -tags realscan -o "%OUTPUT%" ./cmd/bridge

if exist "%OUTPUT%" (
    echo.
    echo ================================================================================
    echo Build successful: %OUTPUT%
    echo ================================================================================
    echo.
    echo Next steps:
    echo   1. Copy SDK DLLs from %SDK_LIB_DIR% to same directory
    echo   2. Run bridge-realscan.exe to test
    echo ================================================================================
) else (
    echo.
    echo Build failed
    exit /b 1
)
