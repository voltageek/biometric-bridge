# Research: Biometric Bridge

**Phase**: 0 — SDK Research & Spike  
**Date**: 2026-04-10  
**Input**: Bridge spec (`gsdk-bridge-spec.md`), G-SDK Go examples (`g-sdk/client/go/`), BS2 Device SDK headers (`biostar-device-sdk/Include/`), Device Gateway binary (`device_gateway_linux_x64_V1.9.0_20260127/`)

## 1. G-SDK Research

### 1.1 Gateway Process

The Suprema Device Gateway (`device_gateway_linux_x64`) is a standalone binary that manages device connections and exposes a gRPC API on `localhost:4000` (configurable via `config.json` → `rpc_server.port`). It uses self-signed TLS certificates generated in its `cert/` directory.

Key config values from `device_gateway_linux_x64_V1.9.0_20260127/config.json`:
- gRPC server: `localhost:4000`
- Device server port: `51212` (plaintext), `51213` (SSL)
- Command timeout: `10000ms` (matches our 10s scan timeout)
- Keep-alive: `32000ms`

### 1.2 gRPC Client Connection

From `g-sdk/client/go/src/example/client/grpc.go`:
- TLS connection using `credentials.NewClientTLSFromFile(certFile, "")`
- Connection timeout: 10 seconds (`CONN_TIMEOUT = 10000 * time.Millisecond`)
- Uses `grpc.WithBlock()` for synchronous connection
- The CA certificate from the gateway's `cert/ca.crt` is required

### 1.3 Device Connection (ConnectSvc)

From `g-sdk/client/go/src/example/connect/sync.go`:
- `ConnectSvc.Connect(deviceIP, devicePort, useSSL)` → returns `uint32` device ID
- Uses `connect.ConnectRequest` with `ConnectInfo{IPAddr, Port, UseSSL}`
- Device ID is assigned by the gateway, not configurable
- `ConnectSvc.GetDeviceList()` returns `[]*connect.DeviceInfo` with device metadata
- `ConnectSvc.Disconnect(deviceIDs)` for cleanup

**Bridge mapping**: Our `DeviceConfig.Name` is a bridge-local human-readable label. The gateway-assigned `uint32` device ID is the opaque `DeviceInfo.ID` we return. The `device/registry.go` must maintain a bidirectional `name ↔ deviceID` mapping.

### 1.4 Fingerprint Scanning (FingerSvc)

From `g-sdk/client/go/src/example/finger/finger.go`:
- `FingerSvc.Scan(deviceID, templateFormat, qualityThreshold)` → `([]byte, uint32, error)`
- Returns `TemplateData` (raw bytes) and `QualityScore`
- Template formats: `TEMPLATE_FORMAT_SUPREMA` (0), `TEMPLATE_FORMAT_ISO` (1), `TEMPLATE_FORMAT_ANSI` (2)
- `FingerSvc.GetConfig(deviceID)` returns `FingerConfig` (security level, fast mode, etc.)

**Bridge mapping**: `Driver.Scan()` wraps `FingerSvc.Scan()`. Quality threshold can default to 0 (let the server decide). Template format should be configurable or default to Suprema proprietary.

### 1.5 User Enrollment (UserSvc)

From `g-sdk/client/go/src/example/user/user.go`:
- `UserSvc.Enroll(deviceID, []*user.UserInfo)` — enrolls one or more users
- `UserInfo` contains user header + finger data
- Enrollment requires: scan finger twice → build `UserInfo` with `UserFinger` → call `Enroll`
- `UserSvc.SetFinger(deviceID, []*user.UserFinger)` can also set fingerprints separately

**Bridge mapping**: `Driver.Enroll()` must: (1) call `FingerSvc.Scan` twice, (2) build a `UserInfo` with the two templates, (3) call `UserSvc.Enroll`. The two scans are sequential with 10s timeout each.

### 1.6 Device Info (DeviceSvc)

From `g-sdk/client/go/src/example/device/device.go`:
- `DeviceSvc.GetInfo(deviceID)` → `*device.FactoryInfo` (model, firmware, MAC, etc.)
- `DeviceSvc.GetCapabilityInfo(deviceID)` → `*device.CapabilityInfo` (finger/face/card support flags)

**Bridge mapping**: After connecting each device, call `GetInfo` + `GetCapabilityInfo` to populate `DeviceInfo.Model`, `DeviceInfo.FirmwareVersion`, `DeviceInfo.FingerSupported`.

### 1.7 Event Monitoring (EventSvc)

From `g-sdk/client/go/src/example/event/event.go`:
- `EventSvc.EnableMonitoring(deviceID)` enables log monitoring for a device
- `EventSvc.SubscribeRealtimeLog(ctx, subReq)` opens a streaming gRPC call
  - `subReq.QueueSize = 8`, `subReq.DeviceIDs = []uint32{...}`
  - Returns `Event_SubscribeRealtimeLogClient` (stream)
  - `eventStream.Recv()` blocks until next event
- `EventSvc.StopMonitoring(deviceID)` disables monitoring

**Bridge mapping**: Use `EnableMonitoringMulti` (mentioned in spec) with all device IDs in a single stream. The goroutine receiving events maps `deviceID` → device name and publishes to the event broker. On stream error, reconnect with exponential backoff.

### 1.8 Multi-device Event Monitoring

The spec mentions `EventSvc.EnableMonitoringMulti` + `EventSvc.SubscribeRealtimeLog` with all device IDs in one stream. The example shows single-device `EnableMonitoring`, but the proto has `EnableMonitoringMulti` which takes a list of device IDs. The `SubscribeRealtimeLog` request already accepts `DeviceIDs []uint32`, so a single stream can carry events from all devices.

## 2. BS2 Device SDK Research

### 2.1 SDK Architecture

From `biostar-device-sdk/Include/BS_API.h`:
- Pure C shared library (`.so`/`.dll`/`.dylib`)
- Requires CGo for Go integration
- Context-based API: `BS2_AllocateContext()` → `BS2_Initialize()` → operations → `BS2_ReleaseContext()`
- All functions return `int` error codes (see `BS_Errno.h`)

### 2.2 Device Connection

- `BS2_ConnectDeviceViaIP(context, deviceAddress, defaultDevicePort, &deviceId)` — connects and returns device ID
- `BS2_DisconnectDevice(context, deviceId)` — disconnects
- `BS2_IsConnected(context, deviceId, &connected)` — connection check
- `BS2_GetDeviceInfo(context, deviceId, &deviceInfo)` — returns `BS2SimpleDeviceInfo` with capability flags

Key `BS2SimpleDeviceInfo` fields relevant to us:
- `id`, `type`, `connectionMode`
- `fingerSupported`, `faceSupported` (capability flags)
- `maxNumOfUser`

### 2.3 Fingerprint Scanning

- `BS2_ScanFingerprintEx(context, deviceId, &finger, templateIndex, quality, templateFormat, &outquality, readyToScanCallback)` — scans one fingerprint with quality output
- `BS2_ScanFingerprint(context, deviceId, &finger, templateIndex, quality, templateFormat, readyToScanCallback)` — without quality output
- `readyToScanCallback`: `OnReadyToScan(BS2_DEVICE_ID deviceId, uint32_t sequence)` — called when device is ready for finger placement
- Template format: `BS2_FINGER_TEMPLATE_FORMAT` enum

**Bridge mapping**: Use `BS2_ScanFingerprintEx` to get the quality score. The `OnReadyToScan` callback is internal to the driver (not exposed to the HTTP API). Timeout is handled by the SDK's default response timeout (`DEFAULT_RESPONSE_TIMEOUT_MS = 10000`).

### 2.4 User Enrollment

- Build a `BS2UserBlob` struct containing: `BS2User`, `BS2UserSetting`, `BS2_USER_NAME`, `BS2Fingerprint*`
- `BS2_MAX_NUM_OF_FINGER_PER_USER = 10`
- Call enroll API with the populated blob

**Bridge mapping**: `Driver.Enroll()` scans twice via `BS2_ScanFingerprintEx`, builds the `BS2UserBlob`, and calls the enroll function.

### 2.5 Event Monitoring

- `BS2_StartMonitoringLog(context, deviceId, OnLogReceived)` — registers a C callback
- `OnLogReceived(BS2_DEVICE_ID deviceId, const BS2Event* event)` — called on each event
- `BS2_StopMonitoringLog(context, deviceId)` — stops monitoring
- Also: `BS2_SetDeviceEventListener(context, onFound, onAccepted, onConnected, onDisconnected)` for connection lifecycle events

**Bridge mapping**: Register `OnLogReceived` for each device. The C callback runs on a C thread; use a channel to forward events to Go goroutines safely. The `OnDeviceDisconnected` callback triggers reconnection logic.

### 2.6 CGo Considerations

- All BS2 callbacks run on C threads — must not call Go functions that might block or grow the stack
- Use `//export` + minimal C-side forwarder to push events into a Go channel
- The shared library must be present at `lib_path` configured in `config.yaml`
- Build requires `CGO_ENABLED=1` and platform-matching library
- Cross-compilation is not supported for BS2 builds

## 3. Key Design Decisions

### 3.1 Driver Registration

Build tags (`//go:build gsdk` / `//go:build bs2`) control which driver is compiled in. Each driver registers itself via an `init()` function in its package. The `cmd/bridge/main.go` imports the driver package, and the `init()` function sets the active driver in a package-level registry.

### 3.2 Device Registry

`internal/device/registry.go` provides:
- `name → deviceID (uint32)` mapping (populated at connect time)
- Per-device `sync.Mutex` for busy-lock (FR-015: reject concurrent scan/enroll on same device)
- Device state tracking: `idle` / `busy` / `disconnected`
- Thread-safe access from HTTP handlers and event monitor

### 3.3 Event Broker Fan-Out

`internal/events/broker.go` implements a pub-sub hub:
- Single input channel from the driver's `Subscribe()` call
- Up to 10 subscriber channels (FR-004)
- Non-blocking send to subscribers (drop events for slow consumers with warning log)
- Subscriber lifecycle tied to WebSocket connection

### 3.4 WebSocket Token Expiry

The auth middleware extracts the JWT's `exp` claim at connection time and sets a `time.Timer`. When the timer fires, the broker sends a close frame to the WebSocket and removes the subscriber. This satisfies FR-005's continuous token expiry enforcement.

### 3.5 Graceful Shutdown

`cmd/bridge/main.go` listens for `SIGINT`/`SIGTERM`:
1. Stop accepting new HTTP connections
2. Close all WebSocket connections (broker drains subscribers)
3. Cancel in-progress scan/enroll operations (context cancellation)
4. Call `Driver.Close()` to release device connections
5. Exit with status 0

### 3.6 Scan Timeout

Both drivers must implement a 10-second timeout for scan operations. For G-SDK, this is handled by `context.WithTimeout`. For BS2, the SDK's `DEFAULT_RESPONSE_TIMEOUT_MS` is already 10s; we can also set it explicitly via `BS2_SetDeviceSearchingTimeout` or by using context cancellation on the Go side.

## 4. Dependency Assessment

| Dependency | Version | Purpose | Risk |
|-----------|---------|---------|------|
| `google.golang.org/grpc` | latest stable | G-SDK gRPC client | Low — mature, widely used |
| `github.com/golang-jwt/jwt/v5` | v5.x | JWT parsing + ES256 validation | Low — standard JWT library for Go |
| `github.com/gorilla/websocket` | v1.5.x | WebSocket server | Low — de facto standard |
| `gopkg.in/yaml.v3` | v3.x | Config file parsing | Low — mature |
| `log/slog` | stdlib (Go 1.21+) | Structured logging | None — standard library |
| G-SDK proto stubs | vendored from `g-sdk/client/go/src/biostar/` | gRPC service clients | Medium — must vendor and maintain |
| BS2 C shared library | Suprema-provided | Direct device communication | Medium — platform-specific, requires CGo |

## 5. Open Research Items (Resolved)

1. **EnableMonitoringMulti proto**: Confirmed exists in the proto definitions. Will use multi-device monitoring with a single stream.
2. **BS2 scan timeout**: `DEFAULT_RESPONSE_TIMEOUT_MS = 10000` in `BS_API.h` matches our 10s requirement.
3. **G-SDK TLS cert path**: Gateway generates certs in `cert/` subdirectory; `ca.crt` is the required CA certificate for client connections.
4. **Template format selection**: Default to `TEMPLATE_FORMAT_SUPREMA` (0) for G-SDK; equivalent for BS2. Server-side matching must know the format, but this is configured at the server level, not the bridge.
