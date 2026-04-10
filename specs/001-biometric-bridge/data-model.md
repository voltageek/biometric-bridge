# Data Model: Biometric Bridge

**Phase**: 1 — Design  
**Date**: 2026-04-10  
**Input**: [spec.md](spec.md), [research.md](research.md)

## 1. Core Entities

### 1.1 DeviceConfig (Configuration Input)

Represents a device as declared in `config.yaml`. Immutable after startup.

```go
type DeviceConfig struct {
    Name   string // Human-readable label (unique, used in all API requests)
    Addr   string // IP address of the reader
    Port   int    // TCP port (e.g., 51211)
    UseSSL bool   // Whether to use SSL for the connection
}
```

**Constraints**:
- `Name` must be unique across all devices in the config
- `Addr` must be a valid IP address or hostname
- `Port` must be in range 1–65535

### 1.2 DeviceInfo (Runtime Metadata)

Represents a connected device as returned by `GET /api/devices`.

```go
type DeviceInfo struct {
    Name            string // From DeviceConfig.Name (human-readable)
    ID              string // Opaque; assigned by the driver at connect time (e.g., gateway's uint32 as string)
    Model           string // Hardware model (e.g., "BioStation 2")
    FirmwareVersion string // Firmware version string
    FingerSupported bool   // Whether the device has a fingerprint sensor
}
```

**Source**: Populated at connect time by calling driver-specific info APIs (G-SDK: `DeviceSvc.GetInfo` + `DeviceSvc.GetCapabilityInfo`; BS2: `BS2_GetDeviceInfo`).

### 1.3 EnrollRequest (API Input)

```go
type EnrollRequest struct {
    DeviceName string // Maps to DeviceConfig.Name → resolved to driver device ID
    UserID     string // Unique user identifier (provided by web app)
    UserName   string // Display name (provided by web app)
}
```

**Constraints**:
- `DeviceName` must match a configured and connected device
- `UserID` must be non-empty

### 1.4 ScanRequest (API Input)

```go
type ScanRequest struct {
    DeviceName string // Maps to DeviceConfig.Name → resolved to driver device ID
}
```

### 1.5 ScanResult (API Output)

```go
type ScanResult struct {
    Template []byte // Raw fingerprint template bytes (SDK-dependent format)
    Quality  int    // 0–100 quality score from the SDK
}
```

**JSON representation**: `Template` is base64-encoded in the HTTP response. `Quality` is an integer.

### 1.6 Event (Real-Time)

```go
type Event struct {
    Type        string // "scan" | "error" | "reconnecting" | "connected"
    DeviceName  string // Human-readable name from config (always present)
    UserID      string // Present when Type == "scan"
    EventCode   uint32 // Present when Type == "scan" (SDK event code)
    Attempt     int    // Present when Type == "reconnecting"
    WaitSeconds int    // Present when Type == "reconnecting"
    Message     string // Present when Type == "error"
}
```

### 1.7 Device State (Internal)

```go
type DeviceState int

const (
    DeviceIdle         DeviceState = iota // Ready for operations
    DeviceBusy                            // Scan or enroll in progress
    DeviceDisconnected                    // Reconnecting
)
```

**State Transitions**:

```
                  ┌──────────────────────┐
                  │                      │
   startup ──►  IDLE ──scan/enroll──► BUSY
                  ▲                     │
                  │    op complete/err   │
                  └─────────────────────┘
                  │
           disconnect detected
                  │
                  ▼
             DISCONNECTED
                  │
           reconnect success
                  │
                  ▼
                IDLE
```

- `IDLE → BUSY`: When a scan or enroll request is accepted for this device
- `BUSY → IDLE`: When the operation completes (success or error)
- `IDLE → DISCONNECTED`: When the driver detects loss of connection
- `DISCONNECTED → IDLE`: When reconnection succeeds
- `BUSY → DISCONNECTED`: If connection lost during an operation (operation fails, then reconnect begins)
- A request for a `BUSY` device returns HTTP 409 "device busy" (FR-015)
- A request for a `DISCONNECTED` device returns HTTP 503 "device unavailable"

## 2. Configuration Entities

### 2.1 BridgeConfig (Top-Level)

```go
type BridgeConfig struct {
    Bridge  BridgeSettings   `yaml:"bridge"`
    Devices []DeviceConfig   `yaml:"devices"`
    Events  EventSettings    `yaml:"events"`
    Log     LogSettings      `yaml:"log"`
    Driver  string           `yaml:"driver"` // "gsdk" or "bs2"
    GSDK    *GSDKSettings    `yaml:"gsdk,omitempty"`
    BS2     *BS2Settings     `yaml:"bs2,omitempty"`
}
```

### 2.2 BridgeSettings

```go
type BridgeSettings struct {
    Listen        string `yaml:"listen"`          // "127.0.0.1:7070"
    AllowedOrigin string `yaml:"allowed_origin"`  // "https://yourapp.com"
    PublicKeyFile string `yaml:"public_key_file"` // PEM file path
    TokenIssuer   string `yaml:"token_issuer"`    // Must match JWT iss claim
    TokenAudience string `yaml:"token_audience"`  // Must match JWT aud claim (default: "biometric-bridge")
    ClockSkew     string `yaml:"clock_skew"`      // Duration string, e.g., "30s"
}
```

### 2.3 LogSettings

```go
type LogSettings struct {
    Level string `yaml:"level"` // "error" | "info" | "debug" (default: "info")
}
```

### 2.4 EventSettings

```go
type EventSettings struct {
    ReconnectBase string `yaml:"reconnect_base"` // Duration string, e.g., "1s"
    ReconnectCap  string `yaml:"reconnect_cap"`  // Duration string, e.g., "120s"
}
```

### 2.5 GSDKSettings

```go
type GSDKSettings struct {
    GatewayAddr   string `yaml:"gateway_addr"`    // "127.0.0.1:4000"
    GatewayCACert string `yaml:"gateway_ca_cert"` // PEM file path for gateway CA
}
```

### 2.6 BS2Settings

```go
type BS2Settings struct {
    LibPath string `yaml:"lib_path"` // Path to libBS2SDK.so/.dll/.dylib
}
```

## 3. API Entities (JSON Wire Format)

### 3.1 Devices Response

```json
{
  "devices": [
    {
      "id": "reception",
      "name": "reception",
      "model": "BioStation 2",
      "firmwareVersion": "2.8.0",
      "fingerSupported": true
    }
  ]
}
```

Note: `id` in the API response is the human-readable `Name` from config (not the internal SDK device ID). This keeps the API SDK-agnostic per Principle VI.

### 3.2 Enroll Request/Response

```json
// Request
{ "deviceId": "reception", "userId": "user-001", "userName": "Jane Smith" }

// Response (success)
{ "ok": true, "userId": "user-001" }
```

### 3.3 Scan Request/Response

```json
// Request
{ "deviceId": "reception" }

// Response (success)
{ "template": "<base64>", "quality": 82 }
```

### 3.4 Error Response

```json
{ "error": "device busy", "code": 409 }
```

### 3.5 WebSocket Event Messages

```json
{ "type": "scan", "deviceId": "reception", "userId": "user-001", "eventCode": 4354 }
{ "type": "reconnecting", "deviceId": "reception", "attempt": 3, "waitSeconds": 4 }
{ "type": "connected", "deviceId": "reception" }
{ "type": "error", "deviceId": "reception", "message": "stream closed" }
```

## 4. Relationships

```
BridgeConfig (1) ──has──► (N) DeviceConfig
BridgeConfig (1) ──has──► (1) BridgeSettings
BridgeConfig (1) ──has──► (1) EventSettings
BridgeConfig (1) ──has──► (0..1) GSDKSettings
BridgeConfig (1) ──has──► (0..1) BS2Settings

DeviceConfig (1) ──connects──► (1) DeviceInfo  [at startup]
DeviceInfo   (1) ──tracked-by──► (1) DeviceState [in registry]

Driver (1) ──manages──► (N) DeviceConfig → DeviceInfo
Driver (1) ──produces──► (N) Event [via Subscribe channel]

EventBroker (1) ──receives-from──► (1) Driver.Subscribe channel
EventBroker (1) ──fans-out-to──► (0..10) WebSocket subscribers
```
