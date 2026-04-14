# Quickstart: Stub/Demo Mode

**Feature**: 003-stub-demo-mode
**Date**: 2026-04-14

---

## Prerequisites

- Go 1.24.7+
- No hardware, SDK libraries, or config files required

## Build

```bash
# Build with any driver tag (realscan shown here — demo driver is always included)
cd biometric-bridge
go build -tags realscan -o bridge ./cmd/bridge
```

## Run

```bash
# Start in demo mode with zero config
./bridge --demo

# Start in production mode (normal behavior, requires hardware + config)
./bridge

# Start in demo mode with a config file (uses config's listen/origin)
./bridge --demo --config /path/to/config.yaml
```

### Expected Startup Output

```
INFO  Starting Biometric Bridge in DEMO mode
INFO  Demo JWT: eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...
INFO  Demo devices: [Demo Device]
INFO  bridge listening addr=127.0.0.1:7070
```

Copy the `Demo JWT` value for use with API calls.

## Test the API

```bash
# Health check (no auth required)
curl http://127.0.0.1:7070/healthz

# List devices (use the demo JWT from startup)
export TOKEN="eyJhbGci..."  # paste the demo JWT here
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:7070/api/devices

# Trigger a scan
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"deviceId": "Demo Device", "finger": "right_index"}' \
  http://127.0.0.1:7070/api/scan

# Trigger an enrollment
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"deviceId": "Demo Device", "userId": "user-001", "userName": "Test User"}' \
  http://127.0.0.1:7070/api/enroll
```

## WebSocket Events

Connect to the event stream:

```bash
# Using wscat or similar
wscat -c "ws://127.0.0.1:7070/events?token=$TOKEN"
```

Expected events:
1. Immediately: `connected` events for each mock device
2. Every ~5 seconds: synthetic `scan` events
3. On API calls: corresponding `scan` or `enrollment_complete` events

## Development with Web Application

1. Start the demo bridge: `./bridge --demo`
2. Copy the demo JWT from startup output
3. Configure your web app to point to `http://127.0.0.1:7070`
4. Set the JWT in your web app's auth header
5. All API endpoints respond with mock data — no hardware needed

## Run Tests

```bash
# Run all tests
go test ./...

# Run only demo driver tests
go test ./internal/driver/demo/...
```

## Cleanup

Press `Ctrl+C` to stop. The bridge shuts down gracefully.
