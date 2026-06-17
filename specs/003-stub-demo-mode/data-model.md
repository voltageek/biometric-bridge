# Data Model: Stub/Demo Mode

**Feature**: 003-stub-demo-mode
**Date**: 2026-04-14
**Status**: Phase 1 Complete

---

## Entities

### MockDeviceConfig

Configuration for a single mock device used by the stub driver.

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Device display name (default: "Demo Device") |
| `model` | `string` | Device model string (default: "BioEntry W2") |
| `id` | `string` | Opaque device identifier (default: auto-generated UUID-like string) |
| `firmwareVersion` | `string` | Firmware version string (default: "v2.6.0") |
| `fingerSupported` | `bool` | Whether the device has a fingerprint sensor (default: true) |
| `scanWidth` | `int` | Single-finger scan image width in pixels (default: 300) |
| `scanHeight` | `int` | Single-finger scan image height in pixels (default: 400) |
| `slapWidth` | `int` | Slap scan image width in pixels (default: 1600) |
| `slapHeight` | `int` | Slap scan image height in pixels (default: 1500) |

**Constraints**:
- `name` must be non-empty
- `scanWidth`, `scanHeight`, `slapWidth`, `slapHeight` must be positive
- When devices are loaded from config, the config `name` is used; other fields use defaults

### DemoConfig

Top-level demo mode configuration.

| Field | Type | Description |
|-------|------|-------------|
| `devices` | `[]MockDeviceConfig` | List of mock devices (default: 1 device) |
| `eventInterval` | `time.Duration` | Interval between simulated events (default: 5s) |
| `scanDelay` | `time.Duration` | Simulated scan processing delay (default: 200ms) |
| `enrollDelay` | `time.Duration` | Simulated enrollment processing delay (default: 800ms) |
| `qualityMin` | `int` | Minimum scan quality score (default: 60) |
| `qualityMax` | `int` | Maximum scan quality score (default: 95) |
| `jwtIssuer` | `string` | JWT issuer claim for demo token (default: "demo") |
| `jwtTTL` | `time.Duration` | Demo JWT validity duration (default: 1h) |

**Constraints**:
- `devices` must have at least 1 entry
- `eventInterval` must be > 0
- `qualityMin` < `qualityMax`
- `qualityMin` >= 0, `qualityMax` <= 100

### EmbeddedTestKeys

Compile-time embedded ECDSA P-256 key pair for demo JWT generation/validation.

| Field | Type | Description |
|-------|------|-------------|
| `publicKeyPEM` | `[]byte` | PEM-encoded ECDSA P-256 public key |
| `privateKeyPEM` | `[]byte` | PEM-encoded ECDSA P-256 private key |

**Constraints**:
- Keys are embedded via Go `//go:embed` directive
- Private key is ONLY used to sign demo JWTs at startup
- Private key MUST NOT be exported, logged, or exposed via API
- Keys are for local development only; MUST NOT be used in production

---

## Relationships

```
DemoConfig
├── MockDeviceConfig (1..n) — Virtual devices exposed via ListDevices
└── EmbeddedTestKeys (1:1) — Used for demo JWT generation/validation

MockDeviceConfig
└── maps to: driver.DeviceInfo (via ListDevices)
└── maps to: driver.ScanResult (via Scan)
└── maps to: driver.SlapScanResult (via SlapScan)

StubDriver (implements driver.Driver)
├── reads: DemoConfig
├── reads: EmbeddedTestKeys (for JWT generation, if driver handles it)
└── emits: driver.Event (via Subscribe channel)
```

---

## State Transitions

### Mock Device State Machine

Mock devices follow the same state machine as real devices, managed by the `device.Registry`:

```
         +-----------+
         |   Idle    |<------------------+
         +-----------+                   |
              |                          |
              | Acquire()                |
              v                          |
         +-----------+     Error()       |
         |   Busy    |------------------>+
         +-----------+                   |
              |                          |
              | Release()               |
              v                          |
         +-----------+                   |
         |   Idle    |-------------------+
         +-----------+
```

**Differences from real devices**:
- No `Disconnected` state in demo mode (no physical device to disconnect)
- State transitions are instantaneous (no hardware delay except simulated `scanDelay`/`enrollDelay`)

---

## Validation Rules

### MockDeviceConfig Validation
- `name` non-empty, max 100 characters
- `scanWidth` > 0, `scanHeight` > 0
- `slapWidth` > 0, `slapHeight` > 0

### DemoConfig Validation
- At least 1 mock device
- `eventInterval` > 0 and <= 60s
- `scanDelay` >= 0 and <= 5s
- `enrollDelay` >= 0 and <= 10s
- `qualityMin` >= 0, `qualityMax` <= 100, `qualityMin` < `qualityMax`
- `jwtTTL` > 0

---

## Persistence

### In-Memory Only (Ephemeral)
- DemoConfig (runtime defaults or loaded from config)
- MockDeviceConfig (derived from config or defaults)
- Synthetic scan/enroll results (generated per-request, not cached)
- Event simulator state (timer goroutine)

### No Database
Demo mode requires no database. All state is in-memory and transient.

### Embedded (Compile-Time)
- Test ECDSA P-256 key pair (PEM files via `//go:embed`)
