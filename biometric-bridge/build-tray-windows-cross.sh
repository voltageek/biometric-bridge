#!/bin/bash
#
# build-tray-windows-cross.sh — Cross-compile tray app for Windows from Linux.
#
# Prerequisites:
#   - mingw-w64 toolchain (x86_64-w64-mingw32-gcc)
#   - Fyne v2 dependencies
#
# Usage:
#   ./build-tray-windows-cross.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Configuration
OUTPUT="bridge-tray.exe"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Building tray app for Windows (x64) using cross-compilation${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}→ Checking prerequisites...${NC}"

if ! command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    echo -e "${RED}✗ Error: x86_64-w64-mingw32-gcc not found${NC}"
    echo -e "${RED}  Install mingw-w64 toolchain:${NC}"
    echo -e "${RED}    Fedora/RHEL: sudo dnf install mingw64-gcc${NC}"
    echo -e "${RED}    Ubuntu/Debian: sudo apt install gcc-mingw-w64-x86-64${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites OK${NC}"
echo ""

# Build
echo -e "${YELLOW}→ Cross-compiling tray app for Windows x64...${NC}"
echo ""

export CGO_ENABLED=1
export GOOS=windows
export GOARCH=amd64
export CC=x86_64-w64-mingw32-gcc
export CXX=x86_64-w64-mingw32-g++

# Fyne requires these for Windows
export CGO_LDFLAGS="-lOpenGL32 -lGdi32 -lComdlg32 -lOle32 -lShell32 -lwinmm"

# Build the tray app
go build -ldflags="-H windowsgui" -o "$OUTPUT" ./cmd/tray

if [ -f "$OUTPUT" ]; then
    SIZE=$(du -h "$OUTPUT" | cut -f1)
    echo ""
    echo -e "${GREEN}✓ Build successful: $OUTPUT ($SIZE)${NC}"
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Next steps:${NC}"
    echo -e "${GREEN}  1. Copy $OUTPUT to Windows machine${NC}"
    echo -e "${GREEN}  2. Run on Windows to test${NC}"
    echo -e "${GREEN}  3. Add to Startup folder for auto-start${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
