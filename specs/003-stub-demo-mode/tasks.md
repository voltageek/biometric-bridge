# Tasks: Stub/Demo Mode

**Input**: Design documents from `/specs/003-stub-demo-mode/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/internal-interfaces.md

**Tests**: Not explicitly requested in the feature specification. Test tasks are omitted per task generation rules.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- All paths are relative to `biometric-bridge/` (the Go module root)

---

## Phase 1: Setup

**Purpose**: Project initialization and basic structure for the demo driver package

- [x] T001 Create `biometric-bridge/internal/driver/demo/` package directory
- [x] T002 Generate ECDSA P-256 test key pair and save PEM files to `biometric-bridge/internal/auth/testkeys/test_ec256.pub` and `biometric-bridge/internal/auth/testkeys/test_ec256.priv`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 [P] Add `LoadPublicKeyBytes` function to `biometric-bridge/internal/auth/keys.go` that parses a PEM-encoded ECDSA P-256 public key from `[]byte` (complementing existing file-based `LoadPublicKey`)
- [x] T004 [P] Add `LoadPrivateKeyBytes` function to `biometric-bridge/internal/auth/keys.go` that parses a PEM-encoded ECDSA P-256 private key from `[]byte`
- [x] T005 [P] Embed test key pair via `//go:embed` in `biometric-bridge/internal/auth/testkeys.go` — expose `TestPublicKeyPEM` and `TestPrivateKeyPEM` variables
- [x] T006 [P] Add `DemoJWT` function to `biometric-bridge/internal/auth/testkeys.go` that generates a signed ES256 JWT using the embedded test key pair with configurable issuer, audience, subject, name, and TTL
- [x] T007 Add `LoadDemoDefaults` function to `biometric-bridge/internal/config/config.go` that returns a `BridgeConfig` with sensible defaults (listen `127.0.0.1:7070`, allowed_origin `http://localhost:3000`, issuer `demo`, audience `biometric-bridge`, clock_skew `30s`, driver `demo`, one default device named "Demo Device", log level `info`)
- [x] T008 [P] Define `MockDeviceConfig` and `DemoConfig` structs in `biometric-bridge/internal/driver/demo/config.go` with fields for device metadata (name, model, id, firmwareVersion, fingerSupported, scanWidth, scanHeight, slapWidth, slapHeight) and demo settings (eventInterval, scanDelay, enrollDelay, qualityMin, qualityMax)
- [x] T009 [P] Implement default `DemoConfig()` function in `biometric-bridge/internal/driver/demo/config.go` that returns a `DemoConfig` with one mock device ("Demo Device"), 5s event interval, 200ms scan delay, 800ms enroll delay, quality range 60–95
- [x] T010 Register demo driver factory unconditionally in `biometric-bridge/cmd/bridge/drivers.go` — import `biometric-bridge/internal/driver/demo` and register `registerDriverFactory("demo", ...)` in an `init()` function
- [x] T011 Verify compilation: `go build ./...` succeeds with demo package included (no build tag required)

**Checkpoint**: Foundation ready — demo config defaults, test keys, demo driver factory registered. User story implementation can now begin.

---

## Phase 3: User Story 7 - Authenticate with Demo JWT (Priority: P1) 🎯 MVP

**Goal**: Developers can authenticate API calls in demo mode using a built-in test JWT printed at startup

**Independent Test**: Start bridge with `--demo`, copy the printed JWT, call `/api/devices` with it, verify 200 response

**Why this phase first**: All other P1 user stories (US1, US2, US3, US5) require authentication. Without demo JWT, none of them can be tested independently.

### Implementation for User Story 7

- [x] T012 [US7] Add `--demo` flag to `biometric-bridge/cmd/bridge/main.go` flag parsing (alongside existing `--test`, `--output`, `--finger`, `--mode`)
- [x] T013 [US7] Add demo startup path in `biometric-bridge/cmd/bridge/main.go`: when `--demo` is set, load config via `LoadDemoDefaults()` (or `LoadDriverOnly` if config file exists), override `cfg.Driver = "demo"`, skip driver factory lookup for real SDK, proceed to HTTP server setup
- [x] T014 [US7] In the demo startup path of `biometric-bridge/cmd/bridge/main.go`: generate a demo JWT using `auth.DemoJWT()` with issuer/audience from config (or defaults), print it to stdout with a prominent log line (`INFO  Demo JWT: <token>`)
- [x] T015 [US7] In the demo startup path of `biometric-bridge/cmd/bridge/main.go`: load the embedded test public key for JWT validation (via `auth.LoadPublicKeyBytes`). If a config file provides `bridge.public_key_file`, use that key instead
- [x] T016 [US7] Ensure `--demo` takes precedence over `--test` in `biometric-bridge/cmd/bridge/main.go` — if both flags are set, run demo mode and ignore `--test`

**Checkpoint**: User Story 7 complete — developers can start bridge in demo mode, get a JWT, and authenticate API calls

---

## Phase 4: User Story 1 - Start Bridge in Demo Mode (Priority: P1) 🎯 MVP

**Goal**: Bridge starts HTTP server in demo mode with zero config, hardware, or SDK libraries

**Independent Test**: Run `./bridge --demo` with no config file, verify HTTP server starts on `127.0.0.1:7070`, `/healthz` returns 200 with `demo: true`

### Implementation for User Story 1

- [x] T017 [US1] Implement stub `Connect` method in `biometric-bridge/internal/driver/demo/driver.go` — no-op, always returns nil, accepts any device config
- [x] T018 [US1] Implement stub `ListDevices` method in `biometric-bridge/internal/driver/demo/driver.go` — returns `driver.DeviceInfo` for each mock device from DemoConfig
- [x] T019 [US1] Implement stub `Close` method in `biometric-bridge/internal/driver/demo/driver.go` — stops event simulator goroutine, closes subscription channel
- [x] T020 [US1] Wire demo startup path in `biometric-bridge/cmd/bridge/main.go` to create the stub driver via factory, call `drv.Connect()`, populate device registry from `drv.ListDevices()`, start event broker from `drv.Subscribe()`, and start HTTP server — following the same flow as production mode
- [x] T021 [US1] Add `"DEMO MODE"` log line at startup in `biometric-bridge/cmd/bridge/main.go` demo path (FR-016)
- [x] T022 [US1] Modify `/healthz` handler in `biometric-bridge/internal/api/router.go` to include `"demo": true` and `"version": "dev"` in the response when running in demo mode — pass a `demo bool` parameter through `RouterDeps`

**Checkpoint**: User Story 1 complete — bridge starts in demo mode with HTTP server, `/healthz` shows demo flag, no hardware required

---

## Phase 5: User Story 2 - Query Mock Devices (Priority: P1) 🎯 MVP

**Goal**: `GET /api/devices` returns realistic mock device data in demo mode

**Independent Test**: Start demo bridge, send `GET /api/devices` with demo JWT, verify response contains mock device with name, model, ID, firmware version

### Implementation for User Story 2

- [x] T023 [P] [US2] Implement mock device metadata generation in `biometric-bridge/internal/driver/demo/config.go` — define default device info (model "BioEntry W2", firmware "v2.6.0", finger supported true, scan 300x400, slap 1600x1500) and generate a deterministic-looking device ID
- [x] T024 [US2] Update `ListDevices` in `biometric-bridge/internal/driver/demo/driver.go` to use `config.go` — if config provides device names, use those names with mock metadata; otherwise use defaults
- [x] T025 [US2] Verify existing `GET /api/devices` handler in `biometric-bridge/internal/api/devices.go` works with demo driver without modifications (SDK-agnostic — should work via `drv.ListDevices()`)

**Checkpoint**: User Story 2 complete — `/api/devices` returns realistic mock devices in demo mode

---

## Phase 6: User Story 3 - Perform Mock Scans (Priority: P1) 🎯 MVP

**Goal**: `POST /api/scan` returns synthetic fingerprint template with realistic quality and dimensions

**Independent Test**: Start demo bridge, send `POST /api/scan` with device ID and JWT, verify 200 response with base64 template, quality 60–95, width, height

### Implementation for User Story 3

- [x] T026 [US3] Implement synthetic scan result generation in `biometric-bridge/internal/driver/demo/generate.go` — generate random template bytes via `crypto/rand` (e.g., 2048 bytes), random quality score in [qualityMin, qualityMax], device-specific dimensions
- [x] T027 [US3] Implement `Scan` method in `biometric-bridge/internal/driver/demo/driver.go` — validate device name exists, simulate delay (configurable, default 200ms via `time.Sleep`), return `driver.ScanResult` from `generate.go`, return error for unknown devices
- [x] T028 [US3] Verify existing `POST /api/scan` handler in `biometric-bridge/internal/api/scan.go` works with demo driver without modifications — device registry handles locking (409 busy), driver returns scan result

**Checkpoint**: User Story 3 complete — scan workflow works end-to-end in demo mode with device locking

---

## Phase 7: User Story 5 - Receive Simulated Real-Time Events (Priority: P1) 🎯 MVP

**Goal**: WebSocket event stream emits simulated events (connected, periodic scans, API-triggered events)

**Independent Test**: Start demo bridge, connect to `/events?token=<jwt>` via WebSocket, verify "connected" events for each device and periodic "scan" events

### Implementation for User Story 5

- [x] T029 [US5] Implement event simulator in `biometric-bridge/internal/driver/demo/events.go` — goroutine-based timer that emits synthetic `driver.Event` values on a channel at configurable interval (default 5s). Events include: periodic `scan` events with random device, and API-triggered events pushed from Scan/Enroll methods
- [x] T030 [US5] Implement `Subscribe` method in `biometric-bridge/internal/driver/demo/driver.go` — start event simulator goroutine, return channel from `events.go`. Immediately emit `connected` event for each mock device
- [x] T031 [US5] Update `Scan` method in `biometric-bridge/internal/driver/demo/driver.go` to push a `scan` event on the subscription channel after successful scan (FR-011)
- [x] T032 [US5] Wire `Close` in `biometric-bridge/internal/driver/demo/driver.go` to stop the event simulator goroutine and close the subscription channel cleanly
- [x] T033 [US5] Verify existing WebSocket handler in `biometric-bridge/internal/events/handler.go` works with demo driver events without modifications — broker fans out events, handler writes JSON

**Checkpoint**: User Story 5 complete — WebSocket event stream delivers connected events, periodic scan events, and API-triggered events

---

## Phase 8: User Story 4 - Perform Mock Enrollments (Priority: P2)

**Goal**: `POST /api/enroll` simulates enrollment with realistic delay and returns success

**Independent Test**: Start demo bridge, send `POST /api/enroll` with device ID, user ID, user name, JWT, verify 200 response with `ok: true`

### Implementation for User Story 4

- [x] T034 [US4] Implement `Enroll` method in `biometric-bridge/internal/driver/demo/driver.go` — validate device name exists, simulate delay (configurable, default 800ms), return nil (success), push `enrollment_complete` event on subscription channel with enrolled user ID
- [x] T035 [US4] Verify existing `POST /api/enroll` handler in `biometric-bridge/internal/api/enroll.go` works with demo driver without modifications

**Checkpoint**: User Story 4 complete — enrollment workflow works end-to-end in demo mode

---

## Phase 9: User Story 6 - Perform Mock Slap Scans (Priority: P2)

**Goal**: `POST /api/slap-scan` returns synthetic slap image and segmented finger results

**Independent Test**: Start demo bridge, send `POST /api/slap-scan` with device ID and mode, verify 200 response with slap image and finger array

### Implementation for User Story 6

- [x] T036 [US6] Implement synthetic slap scan result generation in `biometric-bridge/internal/driver/demo/generate.go` — generate slap image bytes via `crypto/rand`, generate per-finger results matching the requested `CaptureMode` (left_four → 4 fingers, right_four → 4 fingers, two_thumbs → 2 fingers)
- [x] T037 [US6] Implement `SlapScan` method in `biometric-bridge/internal/driver/demo/driver.go` — validate device name and mode, check `fingerSupported`, simulate delay, return `driver.SlapScanResult` from `generate.go`. Return `driver.ErrSlapNotSupported` if device has `fingerSupported: false`
- [x] T038 [US6] Verify existing `POST /api/slap-scan` handler in `biometric-bridge/internal/api/slap_scan.go` works with demo driver without modifications

**Checkpoint**: User Story 6 complete — slap scan workflow works end-to-end in demo mode

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Hardening, verification, and documentation

- [x] T039 [P] Run `go vet ./...` and `go build ./...` to verify no issues across the full module including demo package
- [x] T040 [P] Verify demo mode works when a real config file is provided with `--demo` — listen addr and CORS origin from config are used, driver overridden to "demo"
- [x] T041 [P] Verify `--demo` and `--test` coexistence: `--demo` takes precedence, `--test` is ignored
- [x] T042 Run quickstart.md validation: follow each step (build, run, healthz, devices, scan, enroll, websocket events) and verify all pass
- [ ] T043 Update `specs/003-stub-demo-mode/quickstart.md` if any quickstart steps need correction based on validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US7 (Phase 3)**: Depends on Phase 2 — provides JWT auth needed by all subsequent stories
- **US1 (Phase 4)**: Depends on Phase 2 and US7 — needs demo driver skeleton and auth
- **US2 (Phase 5)**: Depends on US1 (needs running HTTP server and registered devices)
- **US3 (Phase 6)**: Depends on US1 (needs running HTTP server and device registry)
- **US5 (Phase 7)**: Depends on US1 (needs running HTTP server and event broker)
- **US4 (Phase 8)**: Depends on US1 (needs running HTTP server)
- **US6 (Phase 9)**: Depends on US1 (needs running HTTP server)
- **Polish (Phase 10)**: Depends on all user stories being complete

### User Story Dependencies

| Story | Priority | Depends On | Notes |
|-------|----------|------------|-------|
| US7 (Demo JWT) | P1 | Phase 2 | Auth foundation — all other stories need JWT |
| US1 (Start Demo) | P1 | Phase 2 + US7 | HTTP server + driver skeleton |
| US2 (Devices) | P1 | US1 | Needs running server + registered devices |
| US3 (Scan) | P1 | US1 | Needs running server + device registry |
| US5 (Events) | P1 | US1 | Needs running server + event broker |
| US4 (Enroll) | P2 | US1 | Needs running server |
| US6 (Slap Scan) | P2 | US1 | Needs running server |

### Parallel Opportunities

- Phase 1: T001, T002 can run in parallel
- Phase 2: T003, T004, T005, T006, T007, T008, T009 can run in parallel (different files). T010 depends on T008. T011 depends on all above.
- Phase 5 (US2), Phase 6 (US3), Phase 7 (US5): Can be implemented in parallel after US1 is complete (they touch different methods in the demo driver, but share `driver.go` — coordinate via sequential commits or small PRs)
- Phase 8 (US4), Phase 9 (US6): Can be implemented in parallel (independent methods)

---

## Implementation Strategy

### MVP First (P1 Stories Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: US7 (Demo JWT) — auth foundation
4. Complete Phase 4: US1 (Start Demo) — HTTP server running
5. Complete Phases 5–7: US2 + US3 + US5 (Devices, Scan, Events) — can be done in parallel
6. **STOP and VALIDATE**: Full scan workflow (auth → devices → scan → events) works end-to-end
7. Deploy/demo — this is MVP

### Incremental Delivery

1. Setup + Foundational → Demo config, test keys, factory registered
2. US7 (Demo JWT) → Can authenticate API calls
3. US1 (Start Demo) → HTTP server running in demo mode
4. US2 (Devices) → Device listing works
5. US3 (Scan) → Scan workflow works
6. US5 (Events) → WebSocket event stream works
7. **MVP Complete!** — Full frontend development workflow supported
8. US4 (Enroll) → Enrollment workflow
9. US6 (Slap Scan) → Slap scan workflow

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US7 (Demo JWT) + US1 (Start Demo) — sequential, blocks everyone
3. Once US1 is complete:
   - Developer A: US2 (Devices)
   - Developer B: US3 (Scan)
   - Developer C: US5 (Events)
4. Once P1 stories are complete:
   - Developer A: US4 (Enroll)
   - Developer B: US6 (Slap Scan)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- The demo driver package (`internal/driver/demo/`) has zero SDK imports — safe to always compile
- Existing API handlers (`scan.go`, `enroll.go`, `slap_scan.go`, `devices.go`, `events/handler.go`) should NOT need modification — the demo driver implements the same `Driver` interface
- `go:embed` requires files to be relative to the embedding Go file — place PEM files in `internal/auth/testkeys/` and embed from `internal/auth/testkeys.go`
