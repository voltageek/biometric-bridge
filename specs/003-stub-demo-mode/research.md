# Research: Stub/Demo Mode

**Feature**: 003-stub-demo-mode
**Date**: 2026-04-14

## Research Topics

### R1: Stub Driver Registration Pattern

**Decision**: Always compile the demo driver into the binary. Register its factory unconditionally in `cmd/bridge/drivers.go`. Select it at runtime via the `--demo` flag.

**Rationale**: The demo driver has zero SDK imports and adds negligible binary size. Always compiling it in means a single binary supports both production and demo modes — `./bridge` starts normally, `./bridge --demo` starts with the mock driver. This avoids forcing developers to rebuild just to switch modes (e.g., `go build -tags realscan` then `go build -tags demo`). Driver selection happens at config time (the `--demo` flag overrides `cfg.Driver` to `"demo"`), which is consistent with Constitution Principle II ("selection resolved at build time or config time"). The demo driver is registered in `cmd/bridge/drivers.go` via a standard `init()` that imports `internal/driver/demo`.

**Alternatives considered**:
- Build-tag gated (`//go:build demo`): Would exclude the demo driver from production builds, but requires a separate build just for demo mode. A developer working with realscan hardware who wants to quickly test the web UI would need to rebuild. Unnecessary friction.
- Separate binary for demo mode: Duplicates the entire HTTP/auth/event layer. High maintenance cost for no benefit.

### R2: Demo Mode Auth Strategy

**Decision**: Embed a test ECDSA P-256 key pair at compile time. Generate a demo JWT at startup using the embedded private key. Load the embedded public key for JWT validation.

**Rationale**: All authenticated endpoints require JWT validation per Constitution Principle IV. Skipping auth would create a behavioral difference between demo and production, risking bugs that only manifest in production. By embedding a test key pair, the full auth pipeline runs identically in demo mode. The private key is only used to generate the demo JWT printed to console at startup — it never leaves the process. The key pair is generated once and committed as PEM files under `internal/auth/testkeys/`.

**Alternatives considered**:
- No-op validator (accept any token): Violates Principle IV. Would mask auth bugs.
- Skip auth middleware entirely on all routes: Same violation. Web app would not test its JWT handling.
- Require user to generate a key pair and config file: Defeats the "zero config" goal (FR-002).

### R3: Demo Startup Path

**Decision**: Add `--demo` flag to `cmd/bridge/main.go`. When set, the startup flow diverges after flag parsing: load config (with relaxed validation via `LoadDriverOnly`), skip driver factory lookup, instantiate the stub driver directly, and continue with the standard HTTP server + event broker startup path.

**Rationale**: The existing startup flow already has a divergence point for `--test` mode. Demo mode follows the same pattern but continues to start the HTTP server (unlike `--test` which exits after a single scan). By reusing the same event broker, router, and graceful shutdown logic, we avoid duplicating code. The demo driver is selected via the factory registry like any other driver — `--demo` sets `cfg.Driver = "demo"`, and the unconditionally-registered `"demo"` factory is looked up.

**Key flow**:
1. Parse `--demo` flag
2. Load config (fallback to defaults if file missing)
3. Override `cfg.Driver = "demo"` if not already set
4. Look up `driverFactories["demo"]` (always registered in `drivers.go`)
5. Create stub driver, skip real Connect
6. Populate device registry from stub's ListDevices()
7. Load embedded test public key (or config key if available)
8. Start event broker (stub's Subscribe() feeds simulated events)
9. Start HTTP server (standard path)

### R4: Synthetic Data Generation

**Decision**: Generate random but realistic data on each API call using `crypto/rand`.

**Rationale**: Templates must look realistic but must never contain real biometric data (Constitution Principle III). `crypto/rand` provides non-deterministic bytes suitable for synthetic fingerprint templates. Quality scores use a random range (60–95) to simulate natural variation. Dimensions are fixed per mock device (e.g., 300x400 for single finger, 1600x1500 for slap) to match real device output.

**Alternatives considered**:
- Cached/static responses: Would not exercise the web app's handling of varying data (e.g., different quality scores, different template sizes).
- Real template images: Would risk including actual biometric data. Also unnecessary for frontend development.

### R5: Event Simulation Strategy

**Decision**: The stub driver's `Subscribe()` method returns a channel backed by a goroutine that emits events on a timer. Default interval: 5 seconds. API-triggered events (scan, enroll) are pushed to the same channel immediately when the corresponding method is called.

**Rationale**: The existing event broker reads from `drv.Subscribe()` and fans out to WebSocket subscribers. By having the stub driver own the event generation, the broker and WebSocket handler remain unchanged. The goroutine-based timer keeps the driver self-contained. API-triggered events use a shared channel that both the timer goroutine and the API method goroutines write to.

**Alternatives considered**:
- Separate event injector outside the driver: Would require modifying the broker or adding a parallel event path. More complex and violates the Driver interface encapsulation.
- No periodic events (only API-triggered): Would not allow testing the web app's event log with ongoing activity.

### R6: Config Handling in Demo Mode

**Decision**: When `--demo` is set and no config file exists, use an all-defaults `BridgeConfig`. When a config file exists, load it with relaxed validation (using `LoadDriverOnly`-style relaxation that also skips auth field validation).

**Rationale**: The spec requires demo mode to work with zero config (FR-002). The existing `LoadDriverOnly` already skips auth validation but still requires driver/device config. Demo mode needs a further-relaxed loader that provides default devices and skips driver-specific config validation (no `lib_path` or `gateway_addr` required).

**Implementation**: Add a `LoadDemoDefaults()` function to `internal/config/config.go` that returns a config with:
- `bridge.listen`: `127.0.0.1:7070`
- `bridge.allowed_origin`: `http://localhost:3000` (common dev server port)
- `bridge.token_issuer`: `demo`
- `bridge.token_audience`: `biometric-bridge`
- `bridge.clock_skew`: `30s`
- `driver`: `demo`
- `devices`: One default device named "Demo Device"
- `log.level`: `info`
