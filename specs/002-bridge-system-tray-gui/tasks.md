# Tasks: Bridge System Tray GUI

**Input**: Design documents from `/specs/002-bridge-system-tray-gui/`
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/internal-interfaces.md

**Feature**: The Kinetic Vault — System tray GUI for the Biometric Bridge

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, dependency management, and basic structure

- [x] T001 Add Fyne and clipboard dependencies to `biometric-bridge/go.mod`
- [x] T002 Create `biometric-bridge/cmd/tray/` directory structure
- [x] T003 Create `biometric-bridge/internal/tray/` directory structure
- [x] T004 [P] Add `log.file` field to `internal/config/config.go` LogSettings struct
- [x] T005 Verify dependencies build successfully: `go mod tidy && go build ./...`

**Status**: ✅ Phase 1 Complete

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 [P] Implement generic `RingBuffer[T]` in `internal/tray/ringbuffer.go`
- [x] T007 Define `BridgeState` enum and `BridgeStatus` struct in `internal/tray/controller.go`
- [x] T008 Implement `BridgeController` with Start/Stop/Restart/Status in `internal/tray/controller.go`
- [x] T009 Implement `InstanceLocker` using TCP socket in `internal/tray/singleinstance.go`
- [x] T010 Define color constants and layout tokens in `internal/tray/theme.go`
- [x] T011 Add `OnAuthenticated` callback to `internal/auth/middleware.go` MiddlewareConfig
- [x] T012 Implement `Claims` struct and `JWTStore` in `internal/tray/observer.go`
- [x] T013 Define `EventEntry`, `Severity` enum in `internal/tray/event_subscriber.go`
- [x] T014 Implement `EventBuffer` ring buffer in `internal/tray/event_subscriber.go`
- [x] T015 Define `LogEntry` and `LogBuffer` interface in `internal/tray/logger.go`
- [x] T016 Implement `MultiWriterHandler` slog handler in `internal/tray/logger.go`
- [x] T017 Create base `cmd/tray/main.go` with single-instance check

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Monitor Bridge Status (Priority: P1) 🎯 MVP

**Goal**: Display real-time bridge status in the floating panel with ACTIVE/inactive badge and listening address

**Independent Test**: Launch tray, verify panel shows bridge status. Stop bridge via dropdown, verify status updates to inactive.

### Implementation for User Story 1

- [ ] T018 [P] Create `TrayState` struct and state management in `internal/tray/ui.go`
- [ ] T019 [P] Create `BridgeStatus` display component with status badge in `internal/tray/ui.go`
- [ ] T020 Implement status subscription and UI update loop in `internal/tray/ui.go`
- [ ] T021 Implement "The Kinetic Vault" header with dark background (#0F1623) in `internal/tray/ui.go`
- [ ] T022 Implement Bridge Instance section with status dot and "ACTIVE"/inactive label in `internal/tray/ui.go`
- [ ] T023 Implement listening address display in `internal/tray/ui.go`
- [ ] T024 Wire BridgeController status updates to UI components in `cmd/tray/main.go`
- [ ] T025 Implement click-outside-to-close panel behavior in `internal/tray/ui.go`

**Checkpoint**: At this point, User Story 1 should be fully functional — panel opens, shows bridge status, updates when bridge state changes

---

## Phase 4: User Story 2 - Copy JWT Token (Priority: P1) 🎯 MVP

**Goal**: Copy the most recent JWT token to clipboard with visual feedback

**Independent Test**: Make authenticated request to bridge, open panel, click COPY JWT, verify clipboard contains token.

### Implementation for User Story 2

- [ ] T026 Implement auth middleware callback registration in `internal/tray/observer.go`
- [ ] T027 Implement JWT capture from `OnAuthenticated` callback in `internal/tray/controller.go`
- [ ] T028 Implement token validity checking (expiry) in `internal/tray/observer.go`
- [ ] T029 Implement COPY JWT button with clipboard integration in `internal/tray/ui.go`
- [ ] T030 Implement button disabled state when no token available in `internal/tray/ui.go`
- [ ] T031 Implement visual feedback (color flash) on successful copy in `internal/tray/ui.go`
- [ ] T032 Wire JWTStore to UI component in `internal/tray/ui.go`

**Checkpoint**: User Story 2 complete — COPY JWT button works, shows visual feedback, disabled when appropriate

---

## Phase 5: User Story 3 - Re-Sync Devices (Priority: P1) 🎯 MVP

**Goal**: Re-initialize device connections via RE-SYNC button with event log feedback

**Independent Test**: Disconnect device, click RE-SYNC, verify reconnection events appear.

### Implementation for User Story 3

- [ ] T033 Implement `ResyncDevices()` method in `internal/tray/controller.go`
- [ ] T034 Implement RE-SYNC button with loading state in `internal/tray/ui.go`
- [ ] T035 Implement button disabled state when bridge not running in `internal/tray/ui.go`
- [ ] T036 Implement duplicate request prevention in `internal/tray/ui.go`
- [ ] T037 Wire resync to trigger device reconnection in `internal/tray/controller.go`
- [ ] T038 Add resync-related events to event mapping in `internal/tray/event_subscriber.go`

**Checkpoint**: User Story 3 complete — RE-SYNC button triggers device reconnection, shows loading state, disabled appropriately

---

## Phase 6: User Story 4 - View Connected User (Priority: P2)

**Goal**: Display connected user identity (avatar/initials, name, ID) from JWT claims

**Independent Test**: Authenticate to bridge, open panel, verify Connected User section shows correct identity.

### Implementation for User Story 4

- [ ] T039 Implement `ConnectedUser` display component in `internal/tray/ui.go`
- [ ] T040 Implement avatar with initials fallback in `internal/tray/ui.go`
- [ ] T041 Implement user card with light blue-gray (#EAEDFA) background in `internal/tray/ui.go`
- [ ] T042 Implement placeholder state ("No active session") in `internal/tray/ui.go`
- [ ] T043 Wire JWTStore claims extraction to UI in `internal/tray/ui.go`
- [ ] T044 Implement real-time update when new JWT received in `internal/tray/ui.go`

**Checkpoint**: User Story 4 complete — Connected User section displays identity, updates on new auth, shows placeholder when empty

---

## Phase 7: User Story 5 - Monitor Real-Time Events (Priority: P2)

**Goal**: Display scrollable event log with timestamps, severity styling, and filtering

**Independent Test**: Start bridge, perform scan, verify event appears in log with correct styling.

### Implementation for User Story 5

- [ ] T045 Implement event subscription from bridge broker in `internal/tray/controller.go`
- [ ] T046 Implement bridge event to EventEntry mapping in `internal/tray/event_subscriber.go`
- [ ] T047 Implement severity-based color styling (normal/warning/info/muted) in `internal/tray/ui.go`
- [ ] T048 Implement event list with monospaced blue timestamps in `internal/tray/ui.go`
- [ ] T049 Implement scrollable event list container in `internal/tray/ui.go`
- [ ] T050 Implement filter icon and severity filter dropdown in `internal/tray/ui.go`
- [ ] T051 Implement event filter logic in `internal/tray/event_subscriber.go`
 - [ ] T107 Define EventType enum (Scan, Enrollment, Connection, System, Error) in `internal/tray/event_subscriber.go` and data-model.md
 - [ ] T108 Implement event type filter UI in `internal/tray/ui.go`
 - [ ] T109 Map bridge events to EventType in `internal/tray/event_subscriber.go`/`controller.go`
- [ ] T052 Implement "not live" indicator when bridge stopped in `internal/tray/ui.go`
- [ ] T053 Add batch UI updates (100ms ticker) for performance in `internal/tray/ui.go`

**Checkpoint**: User Story 5 complete — Event log displays real-time events, scrollable, filterable, styled by severity

---

## Phase 8: User Story 6 - Access Settings and Service Controls (Priority: P1) 🎯 MVP

**Goal**: Settings dropdown with Restart Service, Stop Service, Open Config, View Logs, Check for Updates

**Independent Test**: Click gear button, verify dropdown appears. Click Restart, verify bridge restarts. Click Stop, verify bridge stops.

### Implementation for User Story 6

- [ ] T054 Implement gear button with hover state in header in `internal/tray/ui.go`
- [ ] T055 Implement settings dropdown with dark background (#1E2535) in `internal/tray/ui.go`
- [ ] T056 Implement menu items: Open Config, View Logs, Check for Updates, Restart Service, Stop Service in `internal/tray/ui.go`
- [ ] T057 Implement Restart Service action with confirmation in `internal/tray/controller.go`
- [ ] T058 Implement Stop Service action with graceful shutdown in `internal/tray/controller.go`
- [ ] T059 Implement Stop Service red styling (#EF4444) in `internal/tray/ui.go`
- [ ] T060 Implement dropdown close on outside click in `internal/tray/ui.go`
- [ ] T061 Implement error state display when bridge fails in `internal/tray/ui.go`

**Checkpoint**: User Story 6 complete — Settings dropdown works, all menu items functional, Restart/Stop work correctly

---

## Phase 9: User Story 7 - Launch and Minimize to System Tray (Priority: P1) 🎯 MVP

**Goal**: Auto-start bridge on launch, run in background, tray icon click opens panel, right-click shows Quit

**Independent Test**: Launch application, verify no window appears, only tray icon. Click icon, verify panel opens. Right-click, verify Quit works.

### Implementation for User Story 7

- [ ] T062 Implement single-instance check in `cmd/tray/main.go`
- [ ] T063 Implement "bring to front" when second instance launches in `internal/tray/singleinstance.go`
- [ ] T064 Implement system tray icon setup in `cmd/tray/main.go`
- [ ] T065 Implement bridge auto-start on application launch in `cmd/tray/main.go`
- [ ] T066 Implement tray icon click to toggle panel in `internal/tray/ui.go`
- [ ] T067 Implement right-click context menu with Quit in `internal/tray/ui.go`
- [ ] T068 Implement Quit action with graceful bridge shutdown in `cmd/tray/main.go`
- [ ] T069 Implement hidden window on startup (no visible window) in `cmd/tray/main.go`
- [ ] T070 Add KINETIC_VAULT_DEV env var for dev mode (show panel on start) in `cmd/tray/main.go`

**Checkpoint**: User Story 7 complete — Application launches minimized, bridge auto-starts, tray interactions work, Quit works

---

## Phase 10: User Story 8 - Open and Edit Configuration (Priority: P2)

**Goal**: Open config.yaml in system default editor from settings dropdown

**Independent Test**: Click Open Config, verify editor opens. Edit config, restart bridge, verify changes applied.

### Implementation for User Story 8

- [ ] T071 Implement `OpenConfig()` with platform-specific commands in `internal/tray/util.go`
- [ ] T072 Wire Open Config menu item to file opener in `internal/tray/ui.go`
- [ ] T073 Implement config file change detection (optional enhancement) in `internal/tray/controller.go`
- [ ] T074 Add validation error display when bridge restart fails due to config error in `internal/tray/ui.go`

**Checkpoint**: User Story 8 complete — Open Config works, config changes can be applied via restart

---

## Phase 11: User Story 9 - View Detailed Logs (Priority: P3)

**Goal**: Separate log viewer window showing structured, filterable logs

**Independent Test**: Click View Logs, verify window opens. Verify logs appear, filtering works.

### Implementation for User Story 9

- [ ] T075 Implement log file creation/opening in `internal/tray/logger.go`
- [ ] T076 Implement JSON log formatter in `internal/tray/logger.go`
- [ ] T077 Implement log viewer window with separate Fyne window in `internal/tray/ui.go`
- [ ] T078 Implement log list with structured display in `internal/tray/ui.go`
- [ ] T079 Implement severity filter dropdown in log viewer in `internal/tray/ui.go`
- [ ] T080 Implement real-time log subscription and display in `internal/tray/ui.go`
- [ ] T081 Wire View Logs menu item to open viewer in `internal/tray/ui.go`

**Checkpoint**: User Story 9 complete — Log viewer opens, shows structured logs, supports filtering

---

## Phase 12: User Story 10 - Check for Updates (Priority: P3)

**Goal**: Check for newer application version and display result

**Independent Test**: Click Check for Updates, verify result displayed (up to date or available).

### Implementation for User Story 10

- [ ] T082 Implement version embedding via ldflags in `cmd/tray/main.go`
- [ ] T083 Implement a no-network update-check stub (show current version or "Update checking not configured") in `internal/tray/updater.go` and `internal/tray/ui.go`

**Checkpoint**: User Story 10 complete — Update check works, displays result, handles errors gracefully

---

## Phase 13: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T088 [P] Add comprehensive logging throughout tray components
- [ ] T089 [P] Implement panic recovery in BridgeController to show error state
- [ ] T090 Add platform-specific tray icon assets
- [ ] T091 [P] Write unit tests for RingBuffer in `internal/tray/ringbuffer_test.go`
- [ ] T092 [P] Write unit tests for JWTStore in `internal/tray/observer_test.go`
- [ ] T093 Add build scripts for cross-platform compilation
- [ ] T094 Create installation packages (optional: installers for Windows/macOS/Linux)
- [ ] T095 Update `specs/002-bridge-system-tray-gui/quickstart.md` with final usage instructions
- [ ] T096 Run quickstart validation: follow steps and verify everything works
- [ ] T097 Code cleanup: remove debug logging, add comments where needed
- [ ] T098 Performance profiling: verify <1s panel open, <2s state updates (POST-MVP)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

| Story | Priority | Depends On | Notes |
|-------|----------|------------|-------|
| US1 (Monitor) | P1 | Foundational | Base panel functionality |
| US2 (Copy JWT) | P1 | US1 + Foundational | Needs panel + auth middleware |
| US3 (Re-Sync) | P1 | US1 + Foundational | Needs panel + controller |
| US6 (Settings) | P1 | US1 + Foundational | Needs panel + controller |
| US7 (Launch) | P1 | Foundational | Entry point, can be done in parallel with US1 |
| US4 (User) | P2 | US1 + Foundational | Enhances panel with user info |
| US5 (Events) | P2 | US1 + Foundational | Enhances panel with event log |
| US8 (Config) | P2 | US6 + Foundational | Depends on settings dropdown |
| US9 (Logs) | P3 | US6 + Foundational | Depends on settings dropdown |
| US10 (Updates) | P3 | US6 + Foundational | Depends on settings dropdown |

### Within Each User Story

- Models/components before wiring
- UI components before integration
- Core implementation before polish
- Story complete before moving to next priority

### Parallel Opportunities

- **Setup phase**: All tasks marked [P] can run in parallel
- **Foundational phase**: All tasks marked [P] can run in parallel (within Phase 2)
  - Exception: T011 (auth middleware) should complete before T012 (JWTStore) starts
  - Exception: T006 (RingBuffer) should complete before T009, T014, T015, T016
- **Once Foundational completes**:
  - US1, US7 can start immediately (parallel)
  - US2, US3, US6 can start after US1 completes
  - US4, US5 can start after US1 completes
  - US8, US9, US10 can start after US6 completes
- **Different user stories**: Can be worked on in parallel by different developers

---

## Implementation Strategy

### MVP First (User Stories 1, 2, 3, 6, 7)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Monitor Bridge) + Phase 9: User Story 7 (Launch)
4. Complete Phase 4: User Story 2 (Copy JWT)
5. Complete Phase 5: User Story 3 (Re-Sync)
6. Complete Phase 8: User Story 6 (Settings)
7. **STOP and VALIDATE**: Test all P1 stories independently
8. Deploy/demo if ready — this is your MVP!

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add US1 (Monitor) → Test independently → Core panel works
3. Add US7 (Launch) → Test independently → Tray integration works
4. Add US2 (Copy JWT) → Test independently → Token feature works
5. Add US3 (Re-Sync) → Test independently → Device control works
6. Add US6 (Settings) → Test independently → Service control works
7. **MVP Complete!** — Deploy/Demo
8. Add US4 (User) → Enhances panel
9. Add US5 (Events) → Full dashboard experience
10. Add US8, US9, US10 → Administrative features

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US1 + US7 (core panel + tray integration)
   - Developer B: US2 + US4 (JWT features)
   - Developer C: US3 + US5 (device + event features)
3. Merge when US1 and US7 complete
4. Then:
   - Developer A: US6 (settings)
   - Developer B: US8 + US9 (config + logs)
   - Developer C: US10 (updates)
5. Stories complete and integrate independently

---

## MVP Scope Definition

**Minimum Viable Product includes**:
- Phase 1: Setup
- Phase 2: Foundational
- Phase 3: US1 (Monitor Bridge Status)
- Phase 4: US2 (Copy JWT Token)
- Phase 5: US3 (Re-Sync Devices)
- Phase 8: US6 (Access Settings and Service Controls)
- Phase 9: US7 (Launch and Minimize to System Tray)

**Total tasks for MVP**: T001-T070 (approximately)

**Post-MVP (P2/P3 stories)**:
- US4: Connected User (nice-to-have identity display)
- US5: Real-Time Event Log (enhanced dashboard)
- US8: Open Config (administrative convenience)
- US9: View Logs (troubleshooting aid)
- US10: Check for Updates (maintenance feature)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Fyne's system tray support is experimental — test early on target platforms
- Linux desktop environment variations may require platform-specific adjustments
- Cross-compilation for Windows/macOS from Linux is supported by Fyne
