# Quickstart: Biometric Bridge

**Phase**: 1 — Design  
**Date**: 2026-04-10

## Prerequisites

| Tool | Version | Check Command |
|------|---------|---------------|
| Go | 1.22+ | `go version` |
| Suprema Device Gateway | 1.9.0+ | Binary in `device_gateway_linux_x64_V1.9.0_20260127/` |
| Fingerprint reader | Any Suprema BioStar 2 compatible | Connected to network or USB |

## 1. Start the Device Gateway (G-SDK driver only)

```bash
cd device_gateway_linux_x64_V1.9.0_20260127/
./device_gateway_linux_x64
```

Verify it's running — the gRPC server should be listening on `localhost:4000`.

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

driver: gsdk

gsdk:
  gateway_addr:    "127.0.0.1:4000"
  gateway_ca_cert: "./device_gateway_linux_x64_V1.9.0_20260127/cert/ca.crt"
```

Adjust `devices[].addr` to match your reader's IP address.

## 4. Build and Run

```bash
# G-SDK driver (pure Go)
go build -tags gsdk -o biometric-bridge ./cmd/bridge
./biometric-bridge

# BS2 driver (requires CGo + shared library)
CGO_ENABLED=1 go build -tags bs2 -o biometric-bridge ./cmd/bridge
./biometric-bridge
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

## 5. Generate a Test JWT

Use a Go script, `jwt.io`, or any JWT library to create a token signed with `bridge-private.pem`:

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
TOKEN="<your-jwt-here>"

# Health check (no auth)
curl http://127.0.0.1:7070/healthz

# List devices
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:7070/api/devices

# Scan a fingerprint (place finger on reader when prompted)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"deviceId":"reception"}' \
     http://127.0.0.1:7070/api/scan

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
