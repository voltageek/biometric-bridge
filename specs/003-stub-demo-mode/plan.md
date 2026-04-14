# Implementation Plan: Stub/Demo Mode

**Branch**: `003-stub-demo-mode` | **Date**: 2026-04-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-stub-demo-mode/spec.md`

## Summary

Add a `--demo` flag to the bridge that starts the full HTTP server and WebSocket event stream using a mock driver — no hardware, SDK libraries, or config file required. The mock driver implements the existing `Driver` interface, returns synthetic scan/enroll results, and emits simulated device events on a timer. Demo mode provides a built-in test JWT so the web application can authenticate and exercise all API endpoints end-to-end without real biometric devices.

## Technical Context

**Language/Version**: Go 1.24.7 (existing project version)
**Primary Dependencies**: Existing bridge stack (net/http, gorilla/websocket, gopkg.in/yaml.v3, golang-jwt/jwt/v5, fyne.io/fyne/v2 for tray)
**Storage**: N/A (in-memory only, no persistent state)
**Testing**: `go test` (standard Go testing), integration tests via `httptest`
**Target Platform**: Linux, macOS, Windows (cross-platform, no native SDK dependency in demo mode)
**Project Type**: cli / web-service (addition to existing bridge CLI)
**Performance Goals**: Startup <1s, API responses <2s, WebSocket events within 5s of connection
**Constraints**: Must respect Constitution security principles (localhost-only, JWT auth); demo driver always compiled in (runtime-selected via --demo flag)
**Scale/Scope**: Single developer workflow; 1-3 mock devices; 1-10 concurrent WebSocket subscribers

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Localhost-Only Security | PASS | Demo mode defaults to `127.0.0.1:7070`. If config provided, listen addr still validated as localhost. |
| II. Driver Abstraction | PASS | Stub driver implements full `Driver` interface. Always compiled in, selected at runtime via `--demo` flag (config-time selection per Constitution Principle II). No SDK imports in the demo package. |
| III. Zero Biometric Storage | PASS | Synthetic templates are generated in-memory, returned to caller, never persisted. |
| IV. JWT-Only Authentication | **CONDITIONAL** | Demo mode MUST still require JWTs. When no config key is available, a built-in test key pair generates a demo JWT. This does not introduce an alternative auth mechanism — JWT validation still occurs. The test key pair is embedded at compile time and MUST NOT be used in production. See Complexity Tracking. |
| V. Resilient Reconnection | N/A | Stub driver has no physical devices to disconnect/reconnect. Simulated reconnection events are optional (deferred). |
| VI. SDK-Agnostic API Surface | PASS | Same routes, same JSON shapes, same event types. The web app cannot distinguish demo from production by API alone (except `/healthz` demo flag). |
| VII. Fail-Fast Startup | **CONDITIONAL** | Demo mode intentionally relaxes fail-fast: no device connection required. This is justified because demo mode is explicitly a development-only mode. The `--demo` flag is a deliberate opt-in that bypasses hardware requirements. See Complexity Tracking. |

**Gate result**: PASS with documented conditions. No unresolved violations.

## Project Structure

### Documentation (this feature)

```text
specs/003-stub-demo-mode/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
biometric-bridge/
├── cmd/
│   └── bridge/
│       ├── main.go                  # Modified: add --demo flag, demo startup path
│       ├── drivers.go               # Modified: register demo factory unconditionally
│       ├── driver_bs2.go            # Existing: bs2 build tag
│       └── driver_realscan.go       # Existing: realscan build tag
├── internal/
│   ├── driver/
│   │   ├── driver.go                # Existing: Driver interface
│   │   └── demo/                    # NEW: stub driver package
│   │       ├── driver.go            # NEW: mock Driver implementation
│   │       ├── devices.go           # NEW: mock device definitions
│   │       ├── events.go            # NEW: event simulator
│   │       └── driver_test.go       # NEW: unit tests
│   ├── auth/
│   │   ├── keys.go                  # Existing: public key loading
│   │   └── testkeys/                # NEW: embedded test key pair
│   │       ├── test_ec256.pub       # NEW: embedded ECDSA P-256 public key
│   │       └── test_ec256.priv      # NEW: embedded ECDSA P-256 private key (demo only)
│   ├── config/
│   │   └── config.go                # Modified: add demo defaults helper
│   └── api/
│       └── router.go                # No changes needed (SDK-agnostic)
```

**Structure Decision**: The stub driver package (`internal/driver/demo/`) implements the `Driver` interface, analogous to `internal/driver/realscan/` and `internal/driver/bs2/`. Unlike real drivers, it is NOT gated behind a build tag — it is always compiled in. The demo driver factory is registered unconditionally in `cmd/bridge/drivers.go`. This allows a single binary to support both production and demo modes: `./bridge` starts normally, `./bridge --demo` starts with the mock driver. The demo package has zero SDK imports and adds negligible binary size. Driver selection happens at config time (the `--demo` flag overrides `cfg.Driver` to `"demo"`), consistent with Constitution Principle II.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| IV. Built-in test key pair for demo JWT | Demo mode must work without a config file, but all authenticated endpoints require JWT validation. Providing a test key pair at compile time allows the bridge to generate and validate demo JWTs without external files. | Using a no-op validator would bypass JWT auth entirely, violating Principle IV and creating a development/production behavioral difference in the auth layer. |
| VII. Relaxed fail-fast in demo mode | Demo mode has no physical devices to connect. Requiring device connection would defeat the purpose of demo mode entirely. | Running without devices but still validating driver config would require a dummy config — adding friction with no benefit. The `--demo` flag is an explicit developer opt-in. |
