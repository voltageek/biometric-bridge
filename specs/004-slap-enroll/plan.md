# Implementation Plan: Slap Enrollment (Multi-Finger)

**Branch**: `004-slap-enroll` | **Date**: 2026-04-16 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-slap-enroll/spec.md`

## Summary

Add a new `POST /api/slap-enroll` endpoint that captures 2 impressions of multiple fingers simultaneously using the existing `driver.SlapScan()` method, validates per-finger quality against a configurable threshold, implements retry logic with WebSocket notifications, and supports strict/lenient mode for missing fingers. This speeds up multi-finger enrollment from ~80s (sequential) to ~40s (parallel slap capture).

## Technical Context

**Language/Version**: Go 1.24.7  
**Primary Dependencies**: net/http, gorilla/websocket, gopkg.in/yaml.v3, golang-jwt/jwt/v5, fyne.io/fyne/v2 (tray)  
**Storage**: N/A (in-memory only, no persistent state - templates returned to caller)  
**Testing**: go test (standard library testing package)  
**Target Platform**: Linux workstations (primary), Windows/macOS (cross-compile)
**Project Type**: Local web service (bridge between browser and biometric hardware)  
**Performance Goals**: 95% of successful slap enrollments complete within 50 seconds (SC-001)  
**Constraints**: 60s overall timeout, 20s per-slap timeout, localhost-only binding, JWT auth required  
**Scale/Scope**: Single workstation, 1 device connected at a time for enrollment operations

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. Localhost-Only Security** | ✅ PASS | No changes to network binding. New endpoint uses existing router infrastructure bound to 127.0.0.1 |
| **II. Driver Abstraction** | ✅ PASS | Uses existing `driver.SlapScan()` interface method. No SDK-specific code in API layer. Quality validation at API layer keeps drivers simple. |
| **III. Zero Biometric Storage** | ✅ PASS | Templates returned in HTTP response, not persisted. No disk writes of biometric data. |
| **IV. JWT-Only Authentication** | ✅ PASS | New endpoint registered with existing auth middleware. No alternative auth mechanisms. |
| **V. Resilient Reconnection** | ✅ PASS | No changes to reconnection logic. Device registry handles lock/unlock. |
| **VI. SDK-Agnostic API Surface** | ✅ PASS | Same request/response shapes for all drivers. BS2 returns standardized 501 error. |
| **VII. Fail-Fast Startup** | ✅ PASS | No changes to startup behavior. Endpoint only available after successful device connection. |

**Gate Result**: All 7 principles pass. No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/004-slap-enroll/
├── plan.md              # This file
├── research.md          # Phase 0 output (no NEEDS CLARIFICATION items)
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── slap-enroll.md   # Phase 1 output - API contract
└── tasks.md             # Phase 2 output (from /speckit.tasks)
```

### Source Code (repository root)

```text
biometric-bridge/
├── cmd/
│   └── bridge/
│       └── main.go              # Entry point (no changes needed)
├── internal/
│   ├── api/
│   │   ├── router.go            # Route registration (MODIFY: add slap-enroll route)
│   │   ├── slap_enroll.go       # NEW: Handler implementation
│   │   └── slap_enroll_test.go  # NEW: Unit tests
│   ├── auth/
│   │   └── middleware.go        # (no changes)
│   ├── config/
│   │   └── config.go            # (no changes - defaults in handler)
│   ├── device/
│   │   └── registry.go          # (no changes - reuse existing locking)
│   ├── driver/
│   │   ├── driver.go            # (no changes - SlapScan already exists)
│   │   ├── demo/
│   │   │   └── driver.go        # (no changes - SlapScan implemented)
│   │   ├── realscan/
│   │   │   └── driver.go        # (no changes - SlapScan implemented)
│   │   └── bs2/
│   │       └── driver.go        # (no changes - returns ErrSlapNotSupported)
│   └── events/
│       └── broker.go            # MODIFY: add Emit() method for direct event injection
└── config.yaml                  # (optional: document defaults)
```

**Structure Decision**: Single Go module (`biometric-bridge`). New handler file `internal/api/slap_enroll.go` follows existing pattern (see `scan.go`, `enroll.go`). Broker extended with `Emit()` method to allow API handlers to inject events directly without going through driver channel.

## Complexity Tracking

> No violations to justify - all Constitution gates pass.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none) | — | — |

## Key Design Decisions

1. **Quality validation at API layer** (not driver): Keeps drivers simple and focused on hardware interaction. Quality thresholds are business logic, not SDK concerns.

2. **Per-impression retry** (not per-finger): Matches hardware behavior - user places all fingers again, not individual finger adjustments.

3. **Reuse existing SlapScan()**: No driver interface changes needed. The existing method provides all required functionality (slap image + segmented fingers + quality scores).

4. **Broker.Emit() for retry events**: The broker currently only reads from driver's Subscribe channel. Adding `Emit(event)` allows API handlers to inject events directly for WebSocket broadcast.

5. **Defaults in handler constants**: Default values (minQuality=60, maxRetries=3, timeout=60s) defined as constants in handler file rather than config.yaml to minimize configuration surface.

## Implementation Approach

**Phase 1 (Core Implementation)**:
- Create `slap_enroll.go` with request/response types matching contract
- Implement 2-impression capture loop using `driver.SlapScan()`
- Register route in `router.go` with existing auth middleware
- Handle error mapping (501 for unsupported, 409 for busy, 504 for timeout)

**Phase 2 (Quality & Retry Logic)**:
- Add per-finger quality validation against `minQuality` threshold
- Implement retry loop with configurable `maxRetries`
- Handle strict/lenient mode for missing fingers (422 vs success)
- Track and return `totalRetries` count

**Phase 3 (WebSocket Events)**:
- Add `Emit()` method to events.Broker for direct event injection
- Emit `enrollment_retry` events on quality failure with retry details

**Phase 4 (Testing)**:
- Unit tests for quality validation logic
- Unit tests for request validation
- Integration tests with demo driver
- Manual hardware testing with RealScan G10
