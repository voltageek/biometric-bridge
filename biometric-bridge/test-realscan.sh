#!/usr/bin/env bash
# test-realscan.sh — End-to-end test for the RealScan G10 driver.
#
# Prerequisites:
#   1. G10 plugged in via USB (check: lsusb | grep 16d1)
#   2. udev rules installed (see Step 1 below)
#   3. Binary built: CGO_ENABLED=1 go build -tags realscan -o bridge ./cmd/bridge
#
# Usage: bash test-realscan.sh

set -euo pipefail
cd "$(dirname "$0")"

SDK_LIB_DIR="../RealScan_SDK_for_Linux_v2.2.0.2470/Lib"
BRIDGE_BIN="./bridge"
CONFIG="config-realscan-test.yaml"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail()  { echo -e "${RED}[FAIL]${NC} $*"; exit 1; }

# ── Preflight checks ─────────────────────────────────────────────────────────

info "Checking prerequisites..."

# Check G10 on USB bus
if ! lsusb 2>/dev/null | grep -q "16d1"; then
    fail "RealScan G10 not found on USB bus. Plug it in and try again."
fi
info "G10 detected on USB bus"

# Check udev rules
if [ ! -f /etc/udev/rules.d/40-realscan.rules ]; then
    warn "udev rules not installed. USB permissions may fail."
    echo ""
    echo "  To fix, run:"
    echo "    sudo cp ../RealScan_SDK_for_Linux_v2.2.0.2470/realscan.rules /etc/udev/rules.d/40-realscan.rules"
    echo "    sudo udevadm control --reload-rules && sudo udevadm trigger"
    echo ""
    echo "  Or run the bridge with sudo (not recommended for production)."
    echo ""
fi

# Check binary exists
if [ ! -x "$BRIDGE_BIN" ]; then
    fail "Bridge binary not found. Build with: CGO_ENABLED=1 go build -tags realscan -o bridge ./cmd/bridge"
fi

# Check SDK libraries
if [ ! -d "$SDK_LIB_DIR" ]; then
    fail "SDK Lib directory not found at $SDK_LIB_DIR"
fi

# Check test keys
if [ ! -f test-public.pem ] || [ ! -f test-private.pem ]; then
    info "Generating test ECDSA keypair..."
    openssl ecparam -name prime256v1 -genkey -noout -out test-private.pem
    openssl ec -in test-private.pem -pubout -out test-public.pem 2>/dev/null
fi

# Generate a fresh JWT token
info "Generating test JWT token..."
TOKEN=$(go run ./cmd/gentoken -key test-private.pem -ttl 24h)
info "Token: ${TOKEN:0:50}..."

# ── Start bridge ──────────────────────────────────────────────────────────────

info "Starting bridge with RealScan driver..."
echo "  Config:  $CONFIG"
echo "  SDK lib: $SDK_LIB_DIR"
echo ""

# Set LD_LIBRARY_PATH so dlopen can find the SDK's dependencies
export LD_LIBRARY_PATH="${SDK_LIB_DIR}:${LD_LIBRARY_PATH:-}"
export BRIDGE_CONFIG="$CONFIG"

# Start bridge in background
$BRIDGE_BIN &
BRIDGE_PID=$!

cleanup() {
    info "Stopping bridge (PID $BRIDGE_PID)..."
    kill "$BRIDGE_PID" 2>/dev/null || true
    wait "$BRIDGE_PID" 2>/dev/null || true
}
trap cleanup EXIT

# Wait for bridge to start
sleep 2

# Check it's running
if ! kill -0 "$BRIDGE_PID" 2>/dev/null; then
    fail "Bridge exited unexpectedly. Check output above."
fi

info "Bridge is running (PID $BRIDGE_PID)"
echo ""

# ── API tests ─────────────────────────────────────────────────────────────────

BASE="http://127.0.0.1:7070"
AUTH="Authorization: Bearer $TOKEN"

# Test 1: Health check (no auth required)
info "Test 1: GET /healthz"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/healthz")
if [ "$HTTP_CODE" = "200" ]; then
    info "  PASS — 200 OK"
else
    fail "  FAIL — expected 200, got $HTTP_CODE"
fi

# Test 2: List devices (requires auth)
info "Test 2: GET /api/devices"
RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE/api/devices")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
if [ "$HTTP_CODE" = "200" ]; then
    info "  PASS — 200 OK"
    echo "  Response: $BODY"
else
    fail "  FAIL — expected 200, got $HTTP_CODE. Body: $BODY"
fi

# Test 3: Scan (requires auth + finger on scanner)
echo ""
info "Test 3: POST /api/scan"
echo -e "  ${YELLOW}Place your finger on the G10 scanner within 10 seconds...${NC}"
RESP=$(curl -s -w "\n%{http_code}" -X POST \
    -H "$AUTH" \
    -H "Content-Type: application/json" \
    -d '{"deviceId":"g10"}' \
    "$BASE/api/scan")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
if [ "$HTTP_CODE" = "200" ]; then
    info "  PASS — 200 OK"
    # Show quality and size but not the full base64 image
    echo "  Response (truncated): $(echo "$BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(json.dumps({k:v for k,v in d.items() if k!="template"}, indent=2))' 2>/dev/null || echo "$BODY" | head -c 200)"
else
    warn "  Got HTTP $HTTP_CODE (may be timeout if no finger placed)"
    echo "  Body: $(echo "$BODY" | head -c 300)"
fi

echo ""
info "Tests complete. Bridge still running — press Ctrl+C to stop."
echo ""
echo "  Manual test commands:"
echo "    curl -s -H \"$AUTH\" $BASE/api/devices | python3 -m json.tool"
echo "    curl -s -X POST -H \"$AUTH\" -H 'Content-Type: application/json' -d '{\"deviceId\":\"g10\"}' $BASE/api/scan | python3 -m json.tool"
echo ""

# Wait for user to Ctrl+C
wait "$BRIDGE_PID"
