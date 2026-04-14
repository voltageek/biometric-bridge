# Internal Contracts: Stub/Demo Mode

**Feature**: 003-stub-demo-mode
**Date**: 2026-04-14

---

## Contract 1: Stub Driver Interface Compliance

The stub driver MUST implement the full `driver.Driver` interface:

```go
type Driver interface {
    Connect(ctx context.Context, devices []driver.DeviceConfig) error
    Scan(ctx context.Context, deviceName string, finger driver.FingerPosition) (*driver.ScanResult, error)
    SlapScan(ctx context.Context, deviceName string, mode driver.CaptureMode) (*driver.SlapScanResult, error)
    Enroll(ctx context.Context, deviceName, userID, userName string, fingers []driver.FingerPosition) error
    ListDevices() []driver.DeviceInfo
    Subscribe() <-chan driver.Event
    Close() error
}
```

### Behavioral Contract

| Method | Behavior |
|--------|----------|
| `Connect` | No-op in demo mode. Always returns nil. Accepts any device config. |
| `Scan` | Simulates a delay (configurable, default 200ms), then returns a `ScanResult` with random template bytes, quality score in [qualityMin, qualityMax], and device-specific dimensions. |
| `SlapScan` | Simulates a delay, then returns a `SlapScanResult` with slap image and per-finger results matching the requested `CaptureMode`. Returns `driver.ErrSlapNotSupported` if called on a device with `fingerSupported: false`. |
| `Enroll` | Simulates a delay (configurable, default 800ms), then returns nil (success). |
| `ListDevices` | Returns `DeviceInfo` for each configured mock device. Called once at startup to populate the device registry. |
| `Subscribe` | Returns a channel that receives simulated `driver.Event` values. The channel is closed when `Close()` is called. Events include periodic synthetic scans and API-triggered events. |
| `Close` | Stops the event simulator goroutine and closes the subscription channel. |

### Error Behavior

| Condition | Returned Error |
|-----------|---------------|
| Unknown device name in Scan/SlapScan/Enroll | Error wrapping "device not found: <name>" |
| SlapScan on non-finger device | `driver.ErrSlapNotSupported` |
| Context cancelled during delay | Context cancellation error |

---

## Contract 2: Demo JWT Generation

Demo mode generates a single JWT at startup using the embedded test key pair.

### JWT Claims

| Claim | Value |
|-------|-------|
| `iss` | `demo` (or `config.Bridge.TokenIssuer` if config provided) |
| `aud` | `biometric-bridge` (or `config.Bridge.TokenAudience` if config provided) |
| `exp` | `now + jwtTTL` (default 1 hour) |
| `iat` | `now` |
| `sub` | `demo-user` |
| `name` | `Demo User` |

### Key Pair

- Algorithm: ES256 (ECDSA P-256)
- Public key: embedded at compile time in `internal/auth/testkeys/test_ec256.pub`
- Private key: embedded at compile time in `internal/auth/testkeys/test_ec256.priv`
- The private key is ONLY used to sign the demo JWT at startup
- JWT validation uses the same public key (or config-provided key if available)

---

## Contract 3: Demo Config Defaults

When no config file is provided (or `--demo` is used with missing config), the following defaults apply:

| Config Field | Default Value |
|-------------|---------------|
| `bridge.listen` | `127.0.0.1:7070` |
| `bridge.allowed_origin` | `http://localhost:3000` |
| `bridge.token_issuer` | `demo` |
| `bridge.token_audience` | `biometric-bridge` |
| `bridge.clock_skew` | `30s` |
| `driver` | `demo` |
| `log.level` | `info` |
| `devices[0].name` | `Demo Device` |

### Override Behavior

When a config file IS provided with `--demo`:
- `bridge.listen` and `bridge.allowed_origin` are respected from config
- `bridge.public_key_file`, `bridge.token_issuer`, `bridge.token_audience` are respected if present (fallback to embedded key)
- `driver` is overridden to `demo` regardless of config value
- `devices` from config are used (names preserved, metadata mocked)
- Driver-specific sections (`gsdk`, `bs2`, `realscan`) are ignored

---

## Contract 4: /healthz Response (Demo Mode)

The `/healthz` endpoint returns an extended response in demo mode:

```json
{
  "status": "ok",
  "demo": true,
  "version": "dev"
}
```

The `demo: true` field is the only addition. Production mode omits this field (or sets it to `false`).

---

## Contract 5: Event Simulation

The stub driver emits events on the subscription channel according to these rules:

| Trigger | Event Type | Fields |
|---------|-----------|--------|
| WebSocket connection established | `connected` | `deviceId`: each mock device name |
| Periodic timer (every N seconds) | `scan` | `deviceId`: random mock device, `userId`: "demo-user", `eventCode`: 0 |
| `POST /api/scan` called | `scan` | `deviceId`: requested device, `userId`: "demo-user", `eventCode`: 0 |
| `POST /api/enroll` called | `enrollment_complete` | `deviceId`: requested device, `userId`: enrolled user ID |

**Note**: The event broker fans these out to all WebSocket subscribers. The web app sees the same event format as production.
