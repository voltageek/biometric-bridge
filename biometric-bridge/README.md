# Biometric Bridge

A lightweight Go daemon that gives web applications authenticated HTTP and WebSocket access to Suprema fingerprint readers. It abstracts the BioStar 2 Device SDK (BS2), the Suprema G-SDK, and the RealScan G10 SDK behind a single, SDK-agnostic API surface. All communication except `/healthz` is protected by short-lived ES256 JWTs.

## Prerequisites

| Tool | Version | Check command |
|------|---------|---------------|
| Go | 1.22+ | `go version` |
| GCC / CGo toolchain | any | `gcc --version` |
| BS2 Device SDK | V2 | shared library at `../biostar-device-sdk/Lib/Linux/lib/x64/libBS_SDK_V2.so` |
| OpenSSL | any | `openssl version` |

A Suprema BioStar 2 compatible fingerprint reader (e.g., BioStation 2, BioEntry W2) must be reachable on the network, **or** a RealScan G10 must be plugged in via USB.

## Build

Choose one build tag that matches your reader SDK.

```bash
# BS2 (network-attached BioStar 2 readers — primary driver)
CGO_ENABLED=1 go build -tags bs2 -o bridge ./cmd/bridge

# RealScan G10 (USB-attached flat-bed scanner)
CGO_ENABLED=1 go build -tags realscan -o bridge ./cmd/bridge

# G-SDK (Suprema Device Gateway — requires a license key)
go build -tags gsdk -o bridge ./cmd/bridge
```

## Configure

Copy or edit `config.yaml`:

```yaml
bridge:
  listen:          "127.0.0.1:7070"
  allowed_origin:  "http://localhost:3000"   # your web app's dev origin
  public_key_file: "./bridge-public.pem"     # ECDSA P-256 public key (PEM)
  token_issuer:    "dev-server"              # must match JWT iss claim
  token_audience:  "biometric-bridge"        # must match JWT aud claim
  clock_skew:      "30s"

devices:
  - name: "reception"
    addr: "192.168.0.110"
    port: 51211
    use_ssl: false

events:
  reconnect_base: "1s"
  reconnect_cap:  "120s"

log:
  level: "info"   # "error" | "info" | "debug"

driver: bs2

bs2:
  lib_path: "./biostar-device-sdk/Lib/Linux/lib/x64/libBS_SDK_V2.so"
```

For the BS2 driver, set the shared library path at runtime so the binary can find it:

```bash
export LD_LIBRARY_PATH="$(pwd)/../biostar-device-sdk/Lib/Linux/lib/x64:$LD_LIBRARY_PATH"
```

For a RealScan G10, see the [RealScan configuration note](#realscan-g10) below.

## Generate a Keypair

The bridge validates requests using an ECDSA P-256 public key. Generate a development keypair once:

```bash
# Private key — used to sign JWTs (kept on your backend server)
openssl ecparam -genkey -name prime256v1 -noout -out bridge-private.pem

# Public key — given to the bridge for validation
openssl ec -in bridge-private.pem -pubout -out bridge-public.pem
```

Set `public_key_file: "./bridge-public.pem"` in `config.yaml`.

## Run

```bash
./bridge
```

Expected startup output:

```
{"level":"INFO","msg":"loading config","path":"config.yaml"}
{"level":"INFO","msg":"loaded public key","path":"./bridge-public.pem"}
{"level":"INFO","msg":"connecting to device","name":"reception","addr":"192.168.0.110:51211"}
{"level":"INFO","msg":"device connected","name":"reception","model":"BioStation 2","id":"542"}
{"level":"INFO","msg":"all devices connected","count":1}
{"level":"INFO","msg":"starting event monitor"}
{"level":"INFO","msg":"listening","addr":"127.0.0.1:7070"}
```

The bridge fails fast on startup if any configured device is unreachable.

## Testing Attached Readers

The bridge has a built-in `--test` flag that connects to the configured device, captures one fingerprint, prints a quality report, and exits — without starting the HTTP server. Use it to verify hardware is working before integrating with a web app.

### Single-finger test

```bash
./bridge --test
```

Optionally light the LED for a specific finger while scanning:

```bash
./bridge --test --finger right_index
```

Valid `--finger` values: `left_little`, `left_ring`, `left_middle`, `left_index`, `left_thumb`, `right_thumb`, `right_index`, `right_middle`, `right_ring`, `right_little`.

Save the captured image for inspection:

```bash
# Save as PNG (requires width/height support in driver)
./bridge --test --output capture.png

# Save as raw 8-bit grayscale bytes
./bridge --test --output capture.raw
```

Example output:

```
╔══════════════════════════════════════════════╗
║          BIOMETRIC BRIDGE — TEST MODE        ║
╚══════════════════════════════════════════════╝

  Device:   reception
  Model:    BioStation 2
  ID:       542
  Firmware: 2.8.0

  Place your finger on the scanner...

  ─── SCAN RESULT ───

  Quality:    82 (NIST)
  Dimensions: 300 x 400 pixels
  Image size: 120000 bytes
  Hex preview (first 64 bytes):

  ...

  Test complete.
```

### Multi-finger (slap) test

For flat-bed readers that support simultaneous multi-finger capture:

```bash
./bridge --test --mode left_four
./bridge --test --mode right_four
./bridge --test --mode two_thumbs
```

Save all segmented finger images at once:

```bash
./bridge --test --mode right_four --output capture.png
# Saves: capture.png (full slap), capture_finger_right_index.png, etc.
```

### RealScan G10

For the USB-attached G10, use the provided end-to-end test script instead:

```bash
# Prerequisites: G10 plugged in, udev rules installed
bash test-realscan.sh
```

The script builds the binary (if needed), generates a test JWT, starts the bridge, and runs health, device list, and scan tests automatically.

RealScan config (`config-realscan-test.yaml`):

```yaml
driver: realscan

realscan:
  lib_path: "../RealScan_SDK_for_Linux_v2.2.0.2470/Lib/libRS_SDK.so.2.2.0.2470"

devices:
  - name: "g10"   # addr/port are ignored — USB auto-discovery
```

## Generate a Test JWT

Use the included `gentoken` tool (faster than `jwt.io` for repeated testing):

```bash
# Generate a token signed with the development private key
go run ./cmd/gentoken -key bridge-private.pem

# Custom issuer / audience to match your config
go run ./cmd/gentoken -key bridge-private.pem -iss dev-server -aud biometric-bridge

# Longer-lived token for a long test session
go run ./cmd/gentoken -key bridge-private.pem -ttl 8h
```

The token is printed to stdout; capture it with:

```bash
TOKEN=$(go run ./cmd/gentoken -key bridge-private.pem)
```

## Test Endpoints

```bash
TOKEN=$(go run ./cmd/gentoken -key bridge-private.pem)

# Health check (no auth required)
curl http://127.0.0.1:7070/healthz

# List devices
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:7070/api/devices

# Single-finger scan (place finger on reader)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"deviceId":"reception"}' \
     http://127.0.0.1:7070/api/scan

# Multi-finger slap scan (place four fingers simultaneously)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"deviceId":"reception","mode":"right_four"}' \
     http://127.0.0.1:7070/api/slap-scan

# Enroll a user (place same finger twice when prompted)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"deviceId":"reception","userId":"user-001","userName":"Jane Smith"}' \
     http://127.0.0.1:7070/api/enroll

# Event stream (WebSocket — requires websocat or similar)
websocat "ws://127.0.0.1:7070/events?token=$TOKEN"
```

## Run Tests

```bash
# All unit tests
go test ./...

# With verbose output
go test -v ./...

# Auth package only
go test -v ./internal/auth/...
```

## API Reference

- [HTTP API](../specs/001-biometric-bridge/contracts/http-api.md) — REST endpoints: `/healthz`, `/api/devices`, `/api/scan`, `/api/slap-scan`, `/api/enroll`
- [WebSocket API](../specs/001-biometric-bridge/contracts/websocket-api.md) — `/events` event stream: scan, reconnecting, connected, error events

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `exit: device connection failed: reception` | Reader unreachable | Check IP, port, and network |
| `exit: public key file not found` | Missing PEM file | Run the keypair generation step or check `public_key_file` in config |
| `401 Unauthorized` | JWT invalid or expired | Regenerate token with correct `iss`, `aud`, and fresh `exp` |
| `503 device unavailable` | Device disconnected | Check physical connection; bridge auto-reconnects |
| `409 device busy` | Concurrent request on same device | Wait for the current scan/enroll to finish |
| `libBS_SDK_V2.so: cannot open shared object` | LD_LIBRARY_PATH not set | Export `LD_LIBRARY_PATH` pointing to the SDK `Lib/Linux/lib/x64/` directory |
| `RealScan G10 not found on USB bus` | Unplugged or udev rules missing | Install udev rules from the RealScan SDK package |
