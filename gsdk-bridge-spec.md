# Biometric Bridge — Technical Specification

**Version:** 0.5  
**Status:** Draft

---

## 1. Purpose

The Biometric Bridge is a lightweight local process that runs on a workstation
and provides a web application's **frontend** with a consistent, authenticated
HTTP and WebSocket interface to one or more fingerprint readers — regardless of
which Suprema SDK is in use underneath.

Because the bridge binds to `127.0.0.1`, only software on the same machine can
reach it. The browser running the web app is co-located with the bridge and
calls it directly. The remote backend server cannot reach the bridge and does
not try to — its only role in this flow is to issue signed JWTs that the
frontend presents to the bridge.

---

## 2. Scope

### In scope
- Connecting one or more Suprema BioStar 2 fingerprint readers
- Enrolling fingerprint templates on a specific device
- Scanning a finger and returning the raw template to the frontend
- Streaming real-time scan events from all configured devices over a single
  WebSocket connection
- Listing devices known to the bridge
- JWT-based authentication with server-issued tokens
- Automatic reconnection to event streams with exponential backoff
- Pluggable SDK drivers selectable at build or config time

### Out of scope
- Fingerprint verification and matching (handled server-side)
- 1:N fingerprint search (handled server-side)
- Master gateway scenarios
- Access control zone configuration
- Card or face credential management
- Any storage of biometric data by the bridge itself

---

## 3. Architecture

The bridge is only reachable from the local machine. The backend server is
remote and never calls the bridge. The call chain is:

```
 Remote server                  User's workstation
 ─────────────                  ──────────────────────────────────────────
 Web App Backend
   │
   │  1. User authenticates; backend issues
   │     a bridge-scoped JWT (ES256, short TTL)
   │     and delivers it to the frontend over HTTPS
   │
   ▼
 Web App Frontend (browser) ──── 2. Bearer <JWT> ────►  Biometric Bridge
                             ◄─── JSON responses ──────   127.0.0.1:7070
                            ────  ws://?token=<JWT>  ────►
                            ◄─── scan events (WS) ───────
```

Inside the bridge:

```
┌──────────────────────────────────────────────────────────────┐
│                   Biometric Bridge  (Go binary)              │
│                       127.0.0.1:7070                         │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  HTTP + WebSocket layer  (SDK-agnostic)             │    │
│  │  router · auth middleware · event broker            │    │
│  └──────────────────────┬──────────────────────────────┘    │
│                         │  Driver interface                  │
│            ┌────────────┴────────────┐                       │
│            ▼                         ▼                       │
│  ┌──────────────────┐    ┌───────────────────────┐          │
│  │  G-SDK driver    │    │  BS2 driver            │          │
│  │  (gRPC client)   │    │  (CGo → C library)     │          │
│  └────────┬─────────┘    └──────────┬────────────┘          │
└───────────┼──────────────────────────┼───────────────────────┘
            │                          │
     gRPC + TLS 1.2            direct TCP (BioStar protocol)
            │                          │
   ┌────────────────┐         ┌────────────────┐
   │ Device Gateway │         │  Reader(s)     │
   │  (Suprema bin) │         │  (LAN or USB)  │
   └───────┬────────┘         └────────────────┘
           │  TCP/IP · BioStar protocol
      ┌────┴─────┐
      ▼          ▼
  Reader A   Reader B
```

---

## 4. Driver Interface

The entire SDK surface is captured in a single Go interface. Drivers are
registered at startup; the rest of the bridge never imports an SDK package
directly.

```go
// Driver is the contract every SDK implementation must satisfy.
// All methods receive a context for cancellation and timeout.
type Driver interface {
    // Connect establishes communication with all configured devices.
    // Must be called once before any other method.
    Connect(ctx context.Context, devices []DeviceConfig) error

    // Devices returns the current list of connected devices with their
    // metadata (name, model, firmware version, capability flags).
    Devices(ctx context.Context) ([]DeviceInfo, error)

    // Enroll scans two impressions of one finger on the named device and
    // stores the resulting templates against the given user.
    Enroll(ctx context.Context, req EnrollRequest) error

    // Scan captures one fingerprint impression from the named device and
    // returns the raw template. The caller (server) is responsible for
    // any matching or verification against stored templates.
    Scan(ctx context.Context, req ScanRequest) (ScanResult, error)

    // Subscribe returns a channel that receives Events from all connected
    // devices. The channel remains open until ctx is cancelled.
    // Drivers handle reconnection internally and signal transient states
    // by emitting Events with Type "reconnecting", "connected", or "error"
    // rather than closing the channel.
    Subscribe(ctx context.Context) (<-chan Event, error)

    // Close releases all resources held by the driver.
    Close() error
}
```

### Shared types

```go
type DeviceConfig struct {
    Name   string
    Addr   string
    Port   int
    UseSSL bool
}

type DeviceInfo struct {
    Name            string
    ID              string // opaque; assigned by the driver at connect time
    Model           string
    FirmwareVersion string
    FingerSupported bool
}

type EnrollRequest struct {
    DeviceName string
    UserID     string
    UserName   string
}

type ScanRequest struct {
    DeviceName string
}

type ScanResult struct {
    // Template is the raw fingerprint template bytes returned by the SDK.
    // Format is SDK-dependent (Suprema proprietary or ISO 19794-2).
    // The server is responsible for all matching against this data.
    Template []byte
    // Quality is a 0–100 score reported by the SDK. The bridge does not
    // enforce a minimum; the server may reject low-quality scans.
    Quality  int
}

type Event struct {
    Type        string // "scan" | "error" | "reconnecting" | "connected"
    DeviceName  string // human-readable name from config, always
    UserID      string // set when Type == "scan"
    EventCode   uint32 // set when Type == "scan"
    Attempt     int    // set when Type == "reconnecting"
    WaitSeconds int    // set when Type == "reconnecting"
    Message     string // set when Type == "error"
}
```

---

## 5. Driver Implementations

### 5.1 G-SDK driver  (`internal/driver/gsdk`)

| Concern | Detail |
|---------|--------|
| Dependency | `github.com/supremainc/g-sdk/api/go` — gRPC generated stubs |
| Prerequisite | Suprema Device Gateway process running on the same host |
| Connection | TLS 1.2 gRPC channel to the gateway; `ConnectSvc.Connect` per device |
| Enroll | `FingerSvc.GetConfig` → `FingerSvc.Scan` × 2 → `UserSvc.Enroll` |
| Scan | `FingerSvc.Scan` × 1 → return `TemplateData` + `QualityScore` |
| Events | `EventSvc.EnableMonitoringMulti` + `EventSvc.SubscribeRealtimeLog` with all device IDs in one stream |
| Reconnect | Re-call `EnableMonitoringMulti`, reopen stream; backoff per section 9 |
| Build | Pure Go — cross-compilation supported |

Driver-specific config block:

```yaml
driver: gsdk
gsdk:
  gateway_addr:    "127.0.0.1:4000"
  gateway_ca_cert: "/etc/biometric-bridge/gateway-ca.crt"
```

### 5.2 BS2 driver  (`internal/driver/bs2`)

| Concern | Detail |
|---------|--------|
| Dependency | Suprema BioStar2 Device SDK C shared library |
| Prerequisite | Platform-matching `.so` / `.dll` / `.dylib` present at `lib_path` |
| Connection | `BS2_Initialize` + `BS2_ConnectDeviceViaIP` per device; no gateway process |
| Enroll | `BS2_ScanFingerprint` × 2 → `BS2_EnrolUser` with `BS2UserBlob` |
| Scan | `BS2_ScanFingerprintEx` × 1 → return template bytes + quality score |
| Events | `BS2_SetNotificationReceiver` registers `OnLogReceived` C callback; a bridge goroutine forwards callbacks into the `Event` channel |
| Reconnect | Re-call `BS2_ConnectDeviceViaIP`; backoff per section 9 |
| Build | Requires CGo; cross-compilation not supported |

Driver-specific config block:

```yaml
driver: bs2
bs2:
  lib_path: "/usr/lib/libBS2SDK.so"  # or .dll / .dylib
```

> **Note:** The G-SDK driver is pure Go and preferred unless the Device
> Gateway is not available or the BS2 SDK is specifically required. BS2
> requires CGo and a platform-matching binary from Suprema.

---

## 6. HTTP + WebSocket API

This section is SDK-agnostic and does not change when the driver changes.

### 6.1 Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET`  | `/healthz`     | None  | Returns `200 OK`. Used by the installer to confirm the bridge is running. |
| `GET`  | `/api/devices` | JWT   | Returns the list of connected devices and their capability flags. |
| `POST` | `/api/enroll`  | JWT   | Scans two impressions and enrolls the user on the specified device. |
| `POST` | `/api/scan`    | JWT   | Captures one fingerprint and returns the raw template. The server is responsible for all matching. |
| `GET`  | `/events`      | JWT † | WebSocket. Streams real-time events from all devices. |

† JWT supplied via `?token=` query parameter — browsers cannot set custom
headers on WebSocket connections.

### 6.2 Enroll

**Request**
```json
POST /api/enroll
Authorization: Bearer <token>

{
  "deviceId": "reception",
  "userId":   "user-001",
  "userName": "Jane Smith"
}
```

**Response**
```json
{ "ok": true, "userId": "user-001" }
```

### 6.3 Scan

Captures one fingerprint impression and returns the raw template to the
caller. All matching — 1:1 verification, 1:N search, or any other comparison
— is performed server-side. The bridge has no knowledge of stored templates.

**Request**
```json
POST /api/scan
Authorization: Bearer <token>

{
  "deviceId": "reception"
}
```

**Response**
```json
{
  "template": "<base64-encoded bytes>",
  "quality":  82
}
```

`template` is base64-encoded raw bytes in the format returned by the SDK
(Suprema proprietary or ISO 19794-2, depending on device configuration).
`quality` is a 0–100 score reported by the SDK. The bridge does not enforce
a minimum — the server may reject low-quality scans before attempting a match.

### 6.4 WebSocket events

```
ws://127.0.0.1:7070/events?token=<JWT>
```

The bridge never closes the WebSocket during a driver reconnection. Clients
stay connected and receive status events throughout.

```json
{ "type": "scan",         "deviceId": "reception", "userId": "user-001", "eventCode": 4354 }
{ "type": "reconnecting", "deviceId": "reception", "attempt": 3, "waitSeconds": 4 }
{ "type": "connected",    "deviceId": "reception" }
{ "type": "error",        "deviceId": "reception", "message": "stream closed" }
```

`deviceId` is always the human-readable name from config.

### 6.5 Error responses

| Status | Meaning |
|--------|---------|
| 400 | Malformed request body |
| 401 | Missing, invalid, or expired JWT |
| 502 | Driver-level failure (SDK call failed) |
| 503 | Named device not connected |

---

## 7. Security Model

### Token flow

The backend server is the only token issuer. The private signing key never
leaves the server. The sequence is:

1. The user authenticates to the web application normally (session cookie,
   OAuth, etc.).
2. When the frontend needs biometric access, it requests a bridge token from
   the backend via an authenticated endpoint on the web app's own API.
3. The backend mints a short-lived JWT signed with the ECDSA private key and
   returns it to the frontend over HTTPS.
4. The frontend holds the JWT in memory (not `localStorage`) and attaches it
   to every bridge request for its lifetime.
5. The bridge validates the JWT signature against the public key on disk. It
   cannot mint tokens and has no knowledge of the private key.

```
  Backend                Frontend (browser)            Bridge (localhost)
  ───────                ──────────────────            ──────────────────
  Sign JWT ──HTTPS──►  hold in memory
                         │
                         ├── Bearer <JWT> ─────────►  validate signature
                         │                             execute SDK call
                         │◄── result ─────────────────
                         │
                         └── ws://?token=<JWT> ──────► validate signature
                          ◄── scan events ─────────────
```

### Why the backend cannot proxy

The bridge listens on `127.0.0.1` and is intentionally unreachable from the
network. The backend runs on a remote server with no path to the user's
workstation. This is a feature, not a limitation — it means biometric hardware
is never exposed to the internet even indirectly.

### Token design

The JWT must be scoped to prevent misuse if extracted from browser memory.
Recommended claims beyond the standard set:

| Claim | Value | Purpose |
|-------|-------|---------|
| `iss` | configured server identifier | Validated by bridge |
| `exp` | now + 15 min | Limits replay window |
| `sub` | authenticated user ID | Ties token to a specific user session |
| `aud` | `"biometric-bridge"` | Ensures tokens issued for other purposes are rejected |

The bridge must validate `iss`, `exp`, and `aud` on every request.

### JWT requirements

| Field | Value |
|-------|-------|
| Algorithm | `ES256` |
| Issuer (`iss`) | Configured string, validated on every request |
| Audience (`aud`) | `"biometric-bridge"`, validated on every request |
| Expiry (`exp`) | Required; tokens without expiry are rejected |
| Recommended TTL | 15 minutes |

### Network binding

The bridge listens on `127.0.0.1` only. Not reachable from the network without
explicit reconfiguration.

### CORS

`Access-Control-Allow-Origin` is set to the single configured origin,
matching the web application's domain. Requests from any other origin are
rejected at the preflight stage. The JWT is the sole guard for `/events` —
CORS does not protect WebSocket connections.

---

## 8. Configuration

`config.yaml` — lives next to the binary. The `driver` key selects the
implementation; only the matching driver block is read.

```yaml
bridge:
  listen:          "127.0.0.1:7070"
  allowed_origin:  "https://yourapp.com"
  public_key_file: "/etc/biometric-bridge/bridge-public.pem"
  token_issuer:    "your-app-server"   # must match iss in server-issued JWTs
  token_audience:  "biometric-bridge"  # must match aud in server-issued JWTs
  clock_skew:      "30s"

devices:
  - name:    "reception"
    addr:    "192.168.0.110"
    port:    51211
    use_ssl: false
  - name:    "server-room"
    addr:    "192.168.0.111"
    port:    51211
    use_ssl: false

events:
  reconnect_base: "1s"
  reconnect_cap:  "120s"

driver: gsdk

gsdk:
  gateway_addr:    "127.0.0.1:4000"
  gateway_ca_cert: "/etc/biometric-bridge/gateway-ca.crt"

# driver: bs2
# bs2:
#   lib_path: "/usr/lib/libBS2SDK.so"
```

---

## 9. Reconnection behaviour

Both drivers implement exponential backoff internally against the same
schedule. The HTTP layer and WebSocket clients are unaffected during
reconnection.

| Attempt | Wait before retry |
|---------|-------------------|
| 1 | 1 s |
| 2 | 2 s |
| 3 | 4 s |
| 4 | 8 s |
| 5 | 16 s |
| 6+ | 120 s (cap) |

Base and cap are configurable via the `events` block. On each attempt the
driver emits a `reconnecting` event. On success it emits `connected` and
resets the counter.

---

## 10. Error handling

- Driver errors from enroll or verify are returned as `502 Bad Gateway` with a
  descriptive plain-text body.
- If a device becomes unreachable, the driver emits an `error` event and begins
  backoff reconnection for that device independently of others.
- Startup is all-or-nothing: all configured devices must connect successfully
  before the HTTP server begins accepting requests. Any failure exits with a
  non-zero status and a descriptive log line.

---

## 11. Deployment

### Workstation dependencies

| Component | Source | Required by |
|-----------|--------|-------------|
| `biometric-bridge` binary | This project | All |
| `bridge-public.pem` | Server / installer | All |
| `device-gateway` binary | Suprema | G-SDK driver only |
| `gateway-ca.crt` | Device Gateway installer | G-SDK driver only |
| `libBS2SDK.so` / `.dll` / `.dylib` | Suprema | BS2 driver only |

### Build tags

```bash
# G-SDK driver (pure Go, cross-compile friendly)
go build -tags gsdk -o biometric-bridge ./cmd/bridge

# BS2 driver (requires CGo and platform SDK binary)
go build -tags bs2 -o biometric-bridge ./cmd/bridge
```

### Startup order

**G-SDK:** `device-gateway` must be running before `biometric-bridge`.  
**BS2:** `biometric-bridge` only — connects to readers directly.

Both should be registered as system services. Example unit files for systemd
and launchd are in `install/`.

---

## 12. Project layout

```
biometric-bridge/
├── cmd/
│   └── bridge/
│       └── main.go
├── internal/
│   ├── driver/
│   │   ├── driver.go          ← Driver interface + shared types
│   │   ├── gsdk/
│   │   │   └── driver.go      ← G-SDK implementation  (build tag: gsdk)
│   │   └── bs2/
│   │       └── driver.go      ← BS2 implementation    (build tag: bs2)
│   ├── auth/
│   │   ├── keys.go            ← public key loading only
│   │   ├── token.go           ← validation only
│   │   └── middleware.go
│   ├── api/
│   │   ├── router.go
│   │   ├── devices.go
│   │   ├── enroll.go
│   │   └── scan.go
│   └── events/
│       ├── broker.go          ← fan-out hub (driver-agnostic)
│       └── monitor.go         ← forwards Driver.Subscribe → broker
├── install/
│   ├── biometric-bridge.service   (systemd)
│   └── biometric-bridge.plist     (launchd)
├── config.yaml
└── go.mod
```

---

## 13. Open questions

1. ~~**Verify method**~~ — **Superseded.** Verification is out of scope for
   the bridge entirely. The bridge exposes a `Scan` endpoint that returns a
   raw template; all matching is performed server-side.

2. ~~**Event reconnection**~~ — **Resolved.** Exponential backoff inside each
   driver. WebSocket clients stay connected and receive status events.

3. ~~**Multi-device**~~ — **Resolved.** Config takes a `devices` list. Drivers
   connect all devices on startup. Events carry `deviceId`.

4. ~~**Token delivery to frontend**~~ — **Resolved.** The frontend is always
   the bridge caller — the backend cannot reach localhost on the user's
   machine. The backend issues a short-lived JWT (scoped with `aud:
   "biometric-bridge"`) and delivers it to the frontend over HTTPS.

5. ~~**BS2 verify round-trip**~~ — **Superseded.** Verification is out of
   scope; the BS2 driver only needs to scan and return a template.
