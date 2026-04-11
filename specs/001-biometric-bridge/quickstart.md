# Quickstart: Biometric Bridge

**Phase**: 1 — Design  
**Date**: 2026-04-10

## Prerequisites

| Tool | Version | Check Command |
|------|---------|---------------|
| Go | 1.22+ | `go version` |
| GCC / CGo toolchain | Any | `gcc --version` |
| BS2 Device SDK | V2 | Shared library at `biostar-device-sdk/Lib/Linux/lib/x64/libBS_SDK_V2.so` |
| Fingerprint reader | Any Suprema BioStar 2 compatible | Connected to network or USB |

## 1. Set Up the BS2 Shared Library

Verify the shared library exists:

```bash
ls -la biostar-device-sdk/Lib/Linux/lib/x64/libBS_SDK_V2.so
```

Set the library path so the bridge can find it at runtime:

```bash
export LD_LIBRARY_PATH="$(pwd)/biostar-device-sdk/Lib/Linux/lib/x64:$LD_LIBRARY_PATH"
```

> **Note**: For the G-SDK driver (deferred — requires license key), you would instead start the Suprema Device Gateway binary on `localhost:4000`. See the G-SDK section in research.md for details.

## 2. Generate a Test Keypair

For development, generate an ECDSA P-256 keypair:

```bash
# Private key (stays on "server" — used to sign test JWTs)
openssl ecparam -genkey -name prime256v1 -noout -out bridge-private.pem

# Public key (given to the bridge for validation)
openssl ec -in bridge-private.pem -pubout -out bridge-public.pem
```

## 3. Configure the Bridge

Create or edit `config.yaml` in the project root:

```yaml
bridge:
  listen:          "127.0.0.1:7070"
  allowed_origin:  "http://localhost:3000"   # your web app's dev origin
  public_key_file: "./bridge-public.pem"
  token_issuer:    "dev-server"
  token_audience:  "biometric-bridge"
  clock_skew:      "30s"

devices:
  - name:    "reception"
    addr:    "192.168.0.110"
    port:    51211
    use_ssl: false

events:
  reconnect_base: "1s"
  reconnect_cap:  "120s"

driver: bs2

bs2:
  lib_path: "./biostar-device-sdk/Lib/Linux/lib/x64/libBS_SDK_V2.so"
```

Adjust `devices[].addr` to match your reader's IP address.

## 4. Build and Run

```bash
# BS2 driver (primary — requires CGo + shared library)
CGO_ENABLED=1 go build -tags bs2 -o biometric-bridge ./cmd/bridge
./biometric-bridge

# G-SDK driver (deferred — requires license key + Device Gateway running)
# go build -tags gsdk -o biometric-bridge ./cmd/bridge
# ./biometric-bridge
```

Expected output:
```
level=INFO msg="loading config" path=config.yaml
level=INFO msg="loaded public key" path=./bridge-public.pem
level=INFO msg="connecting to device" name=reception addr=192.168.0.110:51211
level=INFO msg="device connected" name=reception model="BioStation 2" id=542
level=INFO msg="all devices connected" count=1
level=INFO msg="starting event monitor"
level=INFO msg="listening" addr=127.0.0.1:7070
```

## 4a. Test an Attached Reader (--test mode)

Use the built-in `--test` flag to verify the reader is working without starting the HTTP server. The bridge connects to the first configured device, captures one fingerprint, prints a quality/hex report, and exits.

```bash
# Basic test — place finger when prompted
./biometric-bridge --test

# Light the LED for a specific finger during capture
./biometric-bridge --test --finger right_index

# Valid --finger values:
#   left_little  left_ring  left_middle  left_index  left_thumb
#   right_thumb  right_index  right_middle  right_ring  right_little

# Save the captured image to disk
./biometric-bridge --test --output capture.png     # PNG (requires dimensions in driver)
./biometric-bridge --test --output capture.raw     # raw 8-bit grayscale bytes
```

For flat-bed readers that support simultaneous multi-finger capture, use `--mode`:

```bash
./biometric-bridge --test --mode right_four       # four right-hand fingers
./biometric-bridge --test --mode left_four        # four left-hand fingers
./biometric-bridge --test --mode two_thumbs       # both thumbs

# Save full slap image + individual finger images
./biometric-bridge --test --mode right_four --output capture.png
# Produces: capture.png, capture_finger_right_index.png, etc.
```

## 5. Generate a Test JWT

Use the included `gentoken` tool for quick test tokens:

```bash
TOKEN=$(go run ./cmd/gentoken -key bridge-private.pem)

# Custom issuer and audience (must match config)
TOKEN=$(go run ./cmd/gentoken -key bridge-private.pem -iss dev-server -aud biometric-bridge)

# Longer-lived token
TOKEN=$(go run ./cmd/gentoken -key bridge-private.pem -ttl 8h)
```

Alternatively, use `jwt.io` or any JWT library to create a token signed with `bridge-private.pem`:

```json
{
  "iss": "dev-server",
  "aud": "biometric-bridge",
  "sub": "test-user",
  "exp": 1744382400
}
```

Algorithm: ES256. Sign with the private key from step 2.

## 6. Test Endpoints

```bash
TOKEN=$(go run ./cmd/gentoken -key bridge-private.pem)

# Health check (no auth)
curl http://127.0.0.1:7070/healthz

# List devices
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:7070/api/devices

# Scan a fingerprint (place finger on reader when prompted)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"deviceId":"reception"}' \
     http://127.0.0.1:7070/api/scan

# Multi-finger slap scan (place four fingers simultaneously)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"deviceId":"reception","mode":"right_four"}' \
     http://127.0.0.1:7070/api/slap-scan

# Enroll a user (place finger twice when prompted)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"deviceId":"reception","userId":"user-001","userName":"Jane Smith"}' \
     http://127.0.0.1:7070/api/enroll

# Event stream (WebSocket)
websocat "ws://127.0.0.1:7070/events?token=$TOKEN"
```

## 7. Run Tests

```bash
# All tests
go test ./...

# With verbose output
go test -v ./...

# Only G-SDK driver tests
go test -tags gsdk -v ./internal/driver/gsdk/...

# Only auth tests
go test -v ./internal/auth/...
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `exit: device connection failed: reception` | Reader unreachable | Check reader IP, port, and network connectivity |
| `exit: public key file not found` | Missing PEM file | Run step 2 or check `public_key_file` path in config |
| `401 Unauthorized` | JWT invalid or expired | Regenerate token with correct `iss`, `aud`, and fresh `exp` |
| `503 device unavailable` | Device disconnected | Check physical connection; bridge will auto-reconnect |
| `409 device busy` | Concurrent request to same device | Wait for current operation to complete |
