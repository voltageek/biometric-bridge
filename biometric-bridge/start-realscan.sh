#!/usr/bin/env bash
# start-realscan.sh — Start the biometric bridge with RealScan G10 driver
#
# Usage: ./start-realscan.sh [config-file]
#
# Prerequisites:
#   1. G10 plugged in via USB (check: lsusb | grep 16d1)
#   2. udev rules installed for USB permissions
#   3. Binary built with: CGO_ENABLED=1 go build -tags realscan -o bridge ./cmd/bridge

set -euo pipefail
cd "$(dirname "$0")"

SDK_LIB_DIR="../RealScan_SDK_for_Linux_v2.2.0.2470/Lib"
CONFIG="${1:-config.yaml}"

# Set LD_LIBRARY_PATH so dlopen can find the SDK's dependencies
export LD_LIBRARY_PATH="${SDK_LIB_DIR}:${LD_LIBRARY_PATH:-}"

echo "Starting biometric bridge with RealScan driver..."
echo "  Config: $CONFIG"
echo "  SDK:    $SDK_LIB_DIR"
echo ""

# Run the bridge
exec ./bridge
