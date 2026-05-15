#!/bin/bash
#
# create-windows-deployment.sh — Create Windows deployment package
#
# This script creates a ready-to-deploy Windows package with the bridge
# executable and all required SDK DLLs.
#
# Usage:
#   ./create-windows-deployment.sh [version]
#
# Example:
#   ./create-windows-deployment.sh 1.0.0

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Configuration
VERSION="${1:-dev}"
SDK_DIR="../RealScanSDK for Windows_v2.2.0.2311"
SDK_BIN_DIR="$SDK_DIR/Bin/x64"
DEPLOY_DIR="windows-deploy"
PACKAGE_NAME="biometric-bridge-windows-v${VERSION}.zip"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Creating Windows Deployment Package${NC}"
echo -e "${GREEN}Version: ${VERSION}${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}→ Checking prerequisites...${NC}"

if [ ! -f "bridge-realscan.exe" ]; then
    echo -e "${RED}✗ Error: bridge-realscan.exe not found${NC}"
    echo -e "${RED}  Run ./build-windows-cross.sh first${NC}"
    exit 1
fi

if [ ! -d "$SDK_BIN_DIR" ]; then
    echo -e "${RED}✗ Error: Windows SDK not found at $SDK_BIN_DIR${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites OK${NC}"
echo ""

# Clean old deployment directory
if [ -d "$DEPLOY_DIR" ]; then
    echo -e "${YELLOW}→ Cleaning old deployment directory...${NC}"
    rm -rf "$DEPLOY_DIR"
fi

# Create deployment directory structure
echo -e "${YELLOW}→ Creating deployment structure...${NC}"
mkdir -p "$DEPLOY_DIR/dlls"
echo -e "${GREEN}✓ Created $DEPLOY_DIR/${NC}"

# Copy executable
echo -e "${YELLOW}→ Copying bridge executable...${NC}"
cp bridge-realscan.exe "$DEPLOY_DIR/"
echo -e "${GREEN}✓ Copied bridge-realscan.exe${NC}"

# Copy tray app if available
if [ -f "bridge-tray.exe" ]; then
    echo -e "${YELLOW}→ Copying tray app...${NC}"
    cp bridge-tray.exe "$DEPLOY_DIR/"
    echo -e "${GREEN}✓ Copied bridge-tray.exe${NC}"
    HAS_TRAY=true
else
    echo -e "${YELLOW}⚠ bridge-tray.exe not found - skipping (build separately if needed)${NC}"
    HAS_TRAY=false
fi

# Copy SDK DLLs
echo -e "${YELLOW}→ Copying SDK DLLs...${NC}"
cp "$SDK_BIN_DIR"/*.dll "$DEPLOY_DIR/dlls/"
cp "$SDK_BIN_DIR"/*.ini "$DEPLOY_DIR/dlls/" 2>/dev/null || true
cp "$SDK_BIN_DIR"/*.TXT "$DEPLOY_DIR/dlls/" 2>/dev/null || true
echo -e "${GREEN}✓ Copied SDK DLLs:${NC}"
ls -1 "$DEPLOY_DIR/dlls/" | sed 's/^/  - /'

# Copy configuration example
echo -e "${YELLOW}→ Creating configuration file...${NC}"
if [ -f "config.yaml.example" ]; then
    cp config.yaml.example "$DEPLOY_DIR/config.yaml"
elif [ -f "config.yaml" ]; then
    cp config.yaml "$DEPLOY_DIR/config.yaml"
else
    # Create minimal config
    cat > "$DEPLOY_DIR/config.yaml" << 'EOF'
# RealScan SDK Configuration
realscan_sdk_path: "dlls\\RS_SDK.dll"

# Server Configuration
server:
  port: 8443
  host: "0.0.0.0"
  
# JWT Authentication
jwt:
  secret: "CHANGE-ME-IN-PRODUCTION"
  token_expiry: 24h
  
# TLS Configuration (optional)
tls:
  enabled: false
  cert_file: ""
  key_file: ""

# Logging
log:
  level: "info"
  format: "json"
EOF
fi
echo -e "${GREEN}✓ Created config.yaml${NC}"

# Create README.txt
echo -e "${YELLOW}→ Creating README.txt...${NC}"
if [ "$HAS_TRAY" = true ]; then
cat > "$DEPLOY_DIR/README.txt" << 'EOF'
================================================================================
Biometric Bridge for Windows - RealScan G-10 Support
================================================================================

QUICK START
-----------

1. Extract all files to C:\Program Files\BiometricBridge\

2. Copy all DLLs from the dlls\ directory to the same directory as 
   bridge-realscan.exe:
   
   copy dlls\*.dll .
   copy dlls\*.ini .

3. Edit config.yaml with your settings

4. Choose your preferred mode:
   
   Option A - GUI Tray App (Recommended):
   - Run bridge-tray.exe
   - System tray icon appears
   - Bridge starts automatically
   - Easy status monitoring
   
   Option B - Command Line:
   - Run bridge-realscan.exe
   - Console window shows logs
   - Manual management

REQUIREMENTS
------------

- Windows 10 or later (x64)
- RealScan G-10 USB driver installed
- Visual C++ Redistributable 2015-2022 (x64)

TRAY APP AUTO-START
-------------------

To start tray app automatically on login:

1. Press Win+R
2. Type: shell:startup
3. Create shortcut to bridge-tray.exe in this folder

See WINDOWS-TRAY.md for detailed tray app documentation.

INSTALLATION
------------

See WINDOWS-DEPLOYMENT.md for detailed installation instructions including:
- Desktop application setup
- Windows service installation
- System tray application setup
- Troubleshooting
EOF
else
cat > "$DEPLOY_DIR/README.txt" << 'EOF'
================================================================================
Biometric Bridge for Windows - RealScan G-10 Support
================================================================================

QUICK START
-----------

1. Extract all files to C:\Program Files\BiometricBridge\

2. Copy all DLLs from the dlls\ directory to the same directory as 
   bridge-realscan.exe:
   
   copy dlls\*.dll .
   copy dlls\*.ini .

3. Edit config.yaml with your settings

4. Run bridge-realscan.exe

NOTE: Tray app (bridge-tray.exe) not included in this package.
      Build it separately on Windows using build-tray-windows.bat
      See WINDOWS-TRAY.md for instructions.

REQUIREMENTS
------------

- Windows 10 or later (x64)
- RealScan G-10 USB driver installed
- Visual C++ Redistributable 2015-2022 (x64)

INSTALLATION
------------

See WINDOWS-DEPLOYMENT.md for detailed installation instructions including:
- Desktop application setup
- Windows service installation
- System tray application setup
- Troubleshooting

SUPPORT
-------

For issues and documentation, see:
- WINDOWS-DEPLOYMENT.md
- GitHub repository

LICENSE
-------

See LICENSE.TXT in dlls\ directory for RealScan SDK license.

================================================================================
EOF
fi
echo -e "${GREEN}✓ Created README.txt${NC}"

# Copy deployment documentation
if [ -f "WINDOWS-DEPLOYMENT.md" ]; then
    cp WINDOWS-DEPLOYMENT.md "$DEPLOY_DIR/"
    echo -e "${GREEN}✓ Copied WINDOWS-DEPLOYMENT.md${NC}"
fi

# Copy tray documentation
if [ -f "WINDOWS-TRAY.md" ]; then
    cp WINDOWS-TRAY.md "$DEPLOY_DIR/"
    echo -e "${GREEN}✓ Copied WINDOWS-TRAY.md${NC}"
fi

# Copy build scripts for reference
if [ "$HAS_TRAY" != true ] && [ -f "build-tray-windows.bat" ]; then
    cp build-tray-windows.bat "$DEPLOY_DIR/"
    echo -e "${GREEN}✓ Copied build-tray-windows.bat (for building tray on Windows)${NC}"
fi

# Create install-service.bat
echo -e "${YELLOW}→ Creating install-service.bat...${NC}"
cat > "$DEPLOY_DIR/install-service.bat" << 'EOF'
@echo off
REM install-service.bat — Install biometric-bridge as Windows service

echo ================================================================================
echo Installing Biometric Bridge as Windows Service
echo ================================================================================
echo.

REM Check for admin rights
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: This script requires administrator privileges.
    echo Please right-click and select "Run as administrator"
    pause
    exit /b 1
)

set "INSTALL_DIR=%~dp0"
set "INSTALL_DIR=%INSTALL_DIR:~0,-1%"

echo Installation directory: %INSTALL_DIR%
echo.

echo Creating service...
sc create BiometricBridge binPath= "%INSTALL_DIR%\bridge-realscan.exe" start= auto DisplayName= "Biometric Bridge Service"

if %errorlevel% equ 0 (
    echo.
    echo Service created successfully!
    echo.
    echo Starting service...
    sc start BiometricBridge
    echo.
    echo ================================================================================
    echo Service installed and started
    echo ================================================================================
    echo.
    echo To manage the service:
    echo   - Start:   sc start BiometricBridge
    echo   - Stop:    sc stop BiometricBridge
    echo   - Remove:  sc delete BiometricBridge
    echo.
) else (
    echo.
    echo Failed to create service
    echo Check that no service with this name already exists
    echo.
)

pause
EOF
echo -e "${GREEN}✓ Created install-service.bat${NC}"

# Create uninstall-service.bat
echo -e "${YELLOW}→ Creating uninstall-service.bat...${NC}"
cat > "$DEPLOY_DIR/uninstall-service.bat" << 'EOF'
@echo off
REM uninstall-service.bat — Uninstall biometric-bridge Windows service

echo ================================================================================
echo Uninstalling Biometric Bridge Windows Service
echo ================================================================================
echo.

REM Check for admin rights
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: This script requires administrator privileges.
    echo Please right-click and select "Run as administrator"
    pause
    exit /b 1
)

echo Stopping service...
sc stop BiometricBridge

echo.
echo Removing service...
sc delete BiometricBridge

if %errorlevel% equ 0 (
    echo.
    echo ================================================================================
    echo Service removed successfully
    echo ================================================================================
) else (
    echo.
    echo Failed to remove service
)

echo.
pause
EOF
echo -e "${GREEN}✓ Created uninstall-service.bat${NC}"

# Create version info
echo -e "${YELLOW}→ Creating VERSION.txt...${NC}"
cat > "$DEPLOY_DIR/VERSION.txt" << EOF
Biometric Bridge for Windows
Version: ${VERSION}
Build Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
Platform: Windows x64
SDK: RealScan v2.2.0.2311
Go Version: $(go version | cut -d' ' -f3)
EOF
echo -e "${GREEN}✓ Created VERSION.txt${NC}"

# Create ZIP package
echo ""
echo -e "${YELLOW}→ Creating ZIP package...${NC}"
cd "$DEPLOY_DIR"
zip -r "../$PACKAGE_NAME" . > /dev/null
cd ..

if [ -f "$PACKAGE_NAME" ]; then
    SIZE=$(du -h "$PACKAGE_NAME" | cut -f1)
    echo -e "${GREEN}✓ Created package: $PACKAGE_NAME ($SIZE)${NC}"
else
    echo -e "${RED}✗ Failed to create package${NC}"
    exit 1
fi

# Summary
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Deployment package created successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}Package contents:${NC}"
echo -e "  - bridge-realscan.exe (biometric bridge)"
if [ "$HAS_TRAY" = true ]; then
    echo -e "  - bridge-tray.exe (system tray app)"
fi
echo -e "  - config.yaml (configuration file)"
echo -e "  - dlls/ (RealScan SDK libraries)"
echo -e "  - README.txt (quick start guide)"
echo -e "  - WINDOWS-DEPLOYMENT.md (detailed bridge documentation)"
if [ "$HAS_TRAY" = true ]; then
    echo -e "  - WINDOWS-TRAY.md (tray app documentation)"
fi
echo -e "  - install-service.bat (service installer)"
echo -e "  - uninstall-service.bat (service uninstaller)"
echo -e "  - VERSION.txt (build information)"
echo ""
echo -e "${BLUE}Package location:${NC}"
echo -e "  $PACKAGE_NAME"
echo ""
echo -e "${BLUE}Deploy directory:${NC}"
echo -e "  $DEPLOY_DIR/"
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo -e "  1. Transfer $PACKAGE_NAME to Windows machine"
echo -e "  2. Extract to C:\\Program Files\\BiometricBridge\\"
if [ "$HAS_TRAY" = true ]; then
    echo -e "  3. Run bridge-tray.exe for GUI experience (recommended)"
    echo -e "     OR run bridge-realscan.exe for command-line mode"
else
    echo -e "  3. Run bridge-realscan.exe"
    echo -e "  4. Optional: Build tray app on Windows (see WINDOWS-TRAY.md)"
fi
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
