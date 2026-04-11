# Tasks: Biometric Bridge

**Input**: Design documents from `/specs/001-biometric-bridge/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Project root**: `biometric-bridge/` (single Go binary project)
- Source code follows the canonical layout from plan.md: `cmd/bridge/`, `internal/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, Go module, and directory structure

- [x] T001 Create project directory structure per plan.md: `biometric-bridge/cmd/bridge/`, `biometric-bridge/internal/{config,driver,auth,api,events,device}/`, `biometric-bridge/install/`
- [x] T002 Initialize Go module (`go mod init`) and add dependencies (`github.com/golang-jwt/jwt/v5`, `github.com/gorilla/websocket`, `gopkg.in/yaml.v3`) in `biometric-bridge/go.mod`
- [x] T003 [P] Create default `biometric-bridge/config.yaml` example file per quickstart.md
- [x] T004 [P] Create `biometric-bridge/install/biometric-bridge.service` (systemd unit) and `biometric-bridge/install/biometric-bridge.plist` (launchd plist)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented. Authentication (US5) and driver abstraction (US7) are architecturally foundational — every endpoint and every hardware operation depends on them.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Implement YAML config parsing and validation (FR-018) in `biometric-bridge/internal/config/config.go` — define `BridgeConfig`, `BridgeSettings`, `DeviceConfig`, `EventSettings`, `LogSettings`, `GSDKSettings`, `BS2Settings` structs per data-model.md; validate required fields, unknown driver, port ranges, listen address format; refuse to start on invalid config
- [x] T006 [P] Implement ECDSA public key loading from PEM file in `biometric-bridge/internal/auth/keys.go` — load and parse P-256 public key; return clear error if file missing or invalid (FR-017)
- [x] T007 [P] Implement JWT validation (ES256, iss, aud, exp, clock skew) in `biometric-bridge/internal/auth/token.go` — validate signature, issuer, audience, expiry; return structured errors for each failure mode per http-api.md auth section
- [x] T008 Implement HTTP auth middleware and WebSocket token extraction in `biometric-bridge/internal/auth/middleware.go` — middleware reads `Authorization: Bearer <token>` header; WS helper reads `token` query parameter; both call token validation from T007; skip auth for `/healthz`
- [x] T009 [P] Define `Driver` interface and shared types in `biometric-bridge/internal/driver/driver.go` — `Driver` interface with `Connect`, `Scan`, `Enroll`, `ListDevices`, `Subscribe`, `Close` methods; define `DeviceConfig`, `DeviceInfo`, `ScanResult`, `Event`, `DeviceState` types per data-model.md
- [x] T010 [P] Implement device registry with per-device busy lock in `biometric-bridge/internal/device/registry.go` — name-to-ID mapping, per-device mutex, state tracking (idle/busy/disconnected), `Acquire`/`Release` methods for FR-015, thread-safe access
- [x] T011 Implement structured logging setup with configurable verbosity in `biometric-bridge/cmd/bridge/main.go` — configure `log/slog` with level from config (error/info/debug), JSON handler to stdout (FR-016)
- [x] T012 Implement route registration, CORS middleware, and health check in `biometric-bridge/internal/api/router.go` — register all routes, apply CORS per http-api.md (single origin, allowed methods/headers), implement `GET /healthz` returning `{"status":"ok"}` (FR-012), apply auth middleware to `/api/*` routes

**Checkpoint**: Foundation ready — config, auth, driver interface, device registry, routing, and logging are all in place. User story implementation can now begin.

---

## Phase 3: User Story 1 — Scan a Fingerprint from the Web App (Priority: P1) 🎯 MVP

**Goal**: An operator clicks "Scan" in the browser, the bridge captures a fingerprint from the specified reader and returns the raw template + quality score.

**Independent Test**: Send `POST /api/scan` with a valid JWT and `{"deviceId":"reception"}` to a bridge with a connected reader. Receive `{"template":"<base64>","quality":82}` back.

### Implementation for User Story 1

- [x] T013 [US1] Implement `POST /api/scan` handler in `biometric-bridge/internal/api/scan.go` — parse `ScanRequest` JSON, resolve device name via registry, acquire device lock (409 if busy, 503 if disconnected), call `Driver.Scan` with 10s timeout context, return base64-encoded template + quality score per http-api.md contract, release device lock on completion; handle 400/502/504 error codes

**Checkpoint**: Scan endpoint is functional. With a driver implementation (Phase 8/9), this delivers end-to-end fingerprint capture from the browser.

---

## Phase 4: User Story 2 — Enroll a User's Fingerprint (Priority: P1)

**Goal**: An operator initiates enrollment from the browser, the user places their finger twice, and the enrollment is stored on the device.

**Independent Test**: Send `POST /api/enroll` with `{"deviceId":"reception","userId":"user-001","userName":"Jane Smith"}` and a valid JWT. Complete two finger placements. Receive `{"ok":true,"userId":"user-001"}`.

### Implementation for User Story 2

- [x] T014 [US2] Implement `POST /api/enroll` handler in `biometric-bridge/internal/api/enroll.go` — parse `EnrollRequest` JSON (validate `deviceId`, `userId`, `userName` required), resolve device via registry, acquire device lock (409/503), call `Driver.Enroll` with user ID, user name, and device; 10s timeout per impression; return success or error per http-api.md contract; release device lock on completion; handle 400/502/504 error codes

**Checkpoint**: Enroll endpoint is functional. Combined with US1, the two core biometric operations are complete.

---

## Phase 5: User Story 5 — Authenticate All Bridge Requests (Priority: P1)

**Goal**: Every request (except `/healthz`) is validated for JWT signature, expiry, issuer, and audience. Unauthenticated requests are rejected.

**Independent Test**: Send requests with valid tokens (accepted), expired tokens (rejected), wrong audience (rejected), and no token (rejected). Confirm `/healthz` works without auth.

**Note**: The auth infrastructure was built in Phase 2 (T006–T008). This phase wires it into the application entry point and validates the fail-fast startup behavior.

### Implementation for User Story 5

- [x] T015 [US5] Implement startup auth key validation in `biometric-bridge/cmd/bridge/main.go` — at startup, load public key via `auth.LoadPublicKey`; if missing or invalid, log error and exit non-zero with clear message (FR-017); wire auth middleware from T008 into the router from T012

**Checkpoint**: Auth is fully enforced. All authenticated endpoints reject invalid/missing/expired tokens. Health check remains open. Bridge refuses to start without a valid key file.

---

## Phase 6: User Story 4 — List Connected Devices (Priority: P2)

**Goal**: The web app retrieves all connected readers with names, models, and capabilities for a device picker UI.

**Independent Test**: Send `GET /api/devices` with a valid JWT. Receive a JSON array of device objects per http-api.md contract.

### Implementation for User Story 4

- [x] T016 [US4] Implement `GET /api/devices` handler in `biometric-bridge/internal/api/devices.go` — call `Driver.ListDevices`, map `DeviceInfo` to JSON response per http-api.md contract (id = name, model, firmwareVersion, fingerSupported); return 401 on auth failure

**Checkpoint**: Device listing is functional. The web app can now present a device picker before scan/enroll.

---

## Phase 7: User Story 3 — Monitor Real-Time Scan Events (Priority: P2)

**Goal**: The web app subscribes to a live event stream and receives real-time scan, connection status, and error events from all devices.

**Independent Test**: Open `ws://127.0.0.1:7070/events?token=<JWT>` and trigger a scan on a reader. Receive a scan event within 2 seconds.

### Implementation for User Story 3

- [x] T017 [US3] Implement event broker fan-out hub in `biometric-bridge/internal/events/broker.go` — single input channel from `Driver.Subscribe`; up to 10 subscriber output channels (FR-004); non-blocking fan-out with warning log for slow consumers; subscriber registration/unregistration methods; goroutine lifecycle management
- [x] T018 [US3] Implement WebSocket event handler with token expiry in `biometric-bridge/internal/events/handler.go` — `GET /events` upgrade via gorilla/websocket; extract JWT from `token` query param; validate via auth; enforce 10-subscriber limit (503 on reject per websocket-api.md); register with broker; schedule `time.Timer` from JWT `exp` claim; on expiry send close frame code 4001 `"token expired"` and unregister; on client disconnect unregister; server-push only (no client data messages)

**Checkpoint**: Real-time event streaming is functional. The web app receives scan, reconnecting, connected, and error events from all devices.

---

## Phase 8: User Story 7 — Pluggable Hardware Driver Support (Priority: P3)

**Goal**: The bridge supports interchangeable drivers selected at build time via build tags. The BS2 driver is the primary implementation. The external interface is identical regardless of driver.

**Independent Test**: Build with `-tags bs2` and confirm all API requests produce the expected response formats per contracts.

### Implementation for User Story 7

- [x] T019 [US7] Implement BS2 driver in `biometric-bridge/internal/driver/bs2/driver.go` — build tag `//go:build bs2`; implement `Driver` interface via CGo; `Connect`: `BS2_AllocateContext` + `BS2_ConnectDeviceViaIP`; `Scan`: `BS2_ScanFingerprintEx` with quality output; `Enroll`: two scans + `BS2UserBlob` enrollment; `ListDevices`: `BS2_GetDeviceInfo`; `Subscribe`: `BS2_StartMonitoringLog` callback forwarded via channel; `Close`: `BS2_DisconnectDevice` + `BS2_ReleaseContext`; CGo callback safety (minimal C-side forwarder to Go channel); register via `init()`; shared library at `biostar-device-sdk/Lib/Linux/lib/x64/libBS_SDK_V2.so`
- [ ] T020 [DEFERRED] [US7] Implement G-SDK driver in `biometric-bridge/internal/driver/gsdk/driver.go` — (pending G-SDK license key) build tag `//go:build gsdk`; implement `Driver` interface; `Connect`: gRPC TLS connection to gateway, `ConnectSvc.Connect` per device, populate `DeviceInfo` via `DeviceSvc.GetInfo` + `GetCapabilityInfo`; `Scan`: `FingerSvc.Scan` with 10s context timeout; `Enroll`: two sequential scans + `UserSvc.Enroll`; `ListDevices`: return cached device info; `Subscribe`: `EventSvc.EnableMonitoringMulti` + `SubscribeRealtimeLog` streaming to channel; `Close`: `ConnectSvc.Disconnect` all; register via `init()`

**Checkpoint**: BS2 driver implements the `Driver` interface. Build tag `bs2` selects it at compile time. The external API is SDK-agnostic (FR-013, Constitution Principle VI). G-SDK driver deferred until license key is available.

---

## Phase 9: User Story 6 — Automatic Recovery from Device Disconnection (Priority: P3)

**Goal**: When a reader becomes unreachable, the bridge automatically reconnects with exponential backoff while keeping the event stream open and other devices operational.

**Independent Test**: Disconnect a reader, observe reconnecting events in the stream, reconnect the reader, confirm a "connected" event appears.

### Implementation for User Story 6

- [x] T021 [US6] Implement reconnection with exponential backoff in BS2 driver (`biometric-bridge/internal/driver/bs2/driver.go`) — detect disconnection via `OnDeviceDisconnected` callback; start reconnection goroutine with backoff from `EventSettings` (base 1s, cap 2m); emit `reconnecting` events with attempt count and wait time; emit `connected` event on success; update device state in registry to disconnected/idle; isolate per-device failures so other devices remain operational (FR-010)
- [ ] T022 [DEFERRED] [US6] Implement reconnection with exponential backoff in G-SDK driver (`biometric-bridge/internal/driver/gsdk/driver.go`) — (pending G-SDK license key) detect gRPC stream errors / connection loss; start reconnection goroutine with same backoff parameters; emit `reconnecting`/`connected` events; update device state in registry; isolate per-device failures (FR-010)

**Checkpoint**: BS2 device disconnections are handled automatically. The event stream reports reconnection status. Other devices continue operating normally. G-SDK reconnection deferred until license key is available.

---

## Phase 10: Application Entry Point & Graceful Shutdown

**Purpose**: Wire everything together in the main function and implement graceful shutdown.

- [x] T023 Implement application entry point in `biometric-bridge/cmd/bridge/main.go` — load config (T005), validate config (FR-018), load auth key (T006/T015), initialize driver based on config `driver` field, connect all devices (FR-011: fail-fast if any device unreachable), initialize device registry (T010), start event broker (T017), register routes with auth middleware (T008/T012), start HTTP server on configured listen address, log startup complete
- [x] T024 Implement graceful shutdown in `biometric-bridge/cmd/bridge/main.go` — listen for `SIGINT`/`SIGTERM`; ordered teardown per FR-019: (1) stop accepting new HTTP connections, (2) close all WebSocket connections via broker drain, (3) cancel in-progress scan/enroll via context cancellation, (4) call `Driver.Close` to release device connections, (5) exit 0; log each shutdown step

**Checkpoint**: The bridge starts, validates all prerequisites, connects devices, serves requests, and shuts down cleanly.

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T025 [P] Verify all error responses follow the `{"error":"..."}` JSON format across all handlers in `biometric-bridge/internal/api/`
- [x] T026 [P] Add structured log statements at key points across all packages: config load, key load, device connect/disconnect, scan start/complete, enroll start/complete, auth reject, event stream open/close, shutdown steps
- [ ] T027 Validate end-to-end flow using quickstart.md steps: set up BS2 shared library, generate keypair, configure bridge, build with `-tags bs2`, start bridge, generate test JWT, test all endpoints (`/healthz`, `/api/devices`, `/api/scan`, `/api/enroll`, `/events`)
- [x] T028 [P] Review all handlers for consistent Content-Type `application/json` headers and CORS header application

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **US1 Scan (Phase 3)**: Depends on Phase 2 — core MVP
- **US2 Enroll (Phase 4)**: Depends on Phase 2 — can run in parallel with US1
- **US5 Auth wiring (Phase 5)**: Depends on Phase 2 — can run in parallel with US1/US2
- **US4 Devices (Phase 6)**: Depends on Phase 2 — can run in parallel with US1/US2/US5
- **US3 Events (Phase 7)**: Depends on Phase 2 — can run in parallel with US1/US2/US4/US5
- **US7 Drivers (Phase 8)**: Depends on Phase 2 (driver interface) — can run in parallel with US1–US5
- **US6 Reconnection (Phase 9)**: Depends on Phase 8 (driver implementations) + Phase 7 (event broker)
- **Entry Point (Phase 10)**: Depends on ALL previous phases
- **Polish (Phase 11)**: Depends on Phase 10

### User Story Dependencies

```
Phase 2 (Foundational)
  ├──► US1 (Scan) ──────────────────────────┐
  ├──► US2 (Enroll) ────────────────────────┤
  ├──► US5 (Auth wiring) ──────────────────┤
  ├──► US4 (Devices) ──────────────────────┤
  ├──► US3 (Events) ───────────┐            ├──► Phase 10 (Entry Point) ──► Phase 11 (Polish)
  └──► US7 (Drivers) ──────────┤            │
                                └──► US6 ───┘
```

- US1, US2, US4, US5 are fully independent after Phase 2
- US3 (Events) is independent of US1/US2/US4/US5
- US7 (Drivers) is independent of US1–US5 (implements the interface they consume)
- US6 (Reconnection) requires both US7 (driver implementations) and US3 (event broker)

### Within Each User Story

- Models/types before services
- Services before handlers
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- T003 + T004 can run in parallel (Setup phase)
- T006 + T007 + T009 + T010 can all run in parallel (Foundational — different files)
- After Phase 2: US1, US2, US4, US5 can all start in parallel
- US3 and US7 can start in parallel with each other and with US1/US2/US4/US5
- T020 (G-SDK driver) and T022 (G-SDK reconnection) are deferred pending license key
- T025 + T026 + T028 can run in parallel (Polish phase — different concerns)

---

## Parallel Example: Foundational Phase

```
# Launch all independent foundational tasks together:
Task T006: "Implement ECDSA public key loading in internal/auth/keys.go"
Task T007: "Implement JWT validation in internal/auth/token.go"
Task T009: "Define Driver interface in internal/driver/driver.go"
Task T010: "Implement device registry in internal/device/registry.go"
```

## Parallel Example: User Stories after Foundational

```
# Launch independent user stories in parallel:
Task T013 [US1]: "Implement POST /api/scan handler"
Task T014 [US2]: "Implement POST /api/enroll handler"
Task T015 [US5]: "Implement startup auth key validation"
Task T016 [US4]: "Implement GET /api/devices handler"
```

## Driver Implementation Note

```
# Primary driver (BS2 — CGo, requires shared library):
Task T019 [US7]: "Implement BS2 driver"

# Deferred (pending G-SDK license key):
Task T020 [DEFERRED] [US7]: "Implement G-SDK driver"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1 (Scan)
4. Complete Phase 8: BS2 driver (requires `CGO_ENABLED=1` + BS2 shared library at `biostar-device-sdk/Lib/Linux/lib/x64/libBS_SDK_V2.so`)
5. Complete Phase 10: Entry point (minimal wiring)
6. **STOP and VALIDATE**: Test scan end-to-end with a real reader
7. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. US1 (Scan) + US5 (Auth) + BS2 driver → Test scan independently → **MVP!**
3. Add US2 (Enroll) → Test enrollment independently
4. Add US4 (Devices) → Device picker available
5. Add US3 (Events) → Real-time monitoring
6. Add US6 (Reconnection) → Production resilience
7. Add G-SDK driver → Multi-SDK support (when license key available)
8. Polish → Production ready

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US1 (Scan) + US2 (Enroll)
   - Developer B: US3 (Events) + US4 (Devices)
   - Developer C: US7 (BS2 driver)
3. After BS2 driver is ready: Developer C takes US6 (Reconnection)
4. Phase 10 (Entry Point) integrates all work
5. Team reviews Polish phase together
6. G-SDK driver (T020) and G-SDK reconnection (T022) are picked up when license key is available

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [DEFERRED] tasks = blocked on external dependency (G-SDK license key); not on the critical path
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable (given the BS2 driver)
- Tests are not included — not explicitly requested in the spec
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- BS2 is the primary driver; G-SDK is deferred pending license key acquisition
