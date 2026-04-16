# Tasks: Slap Enrollment (Multi-Finger)

**Feature**: 004-slap-enroll | **Branch**: `004-slap-enroll` | **Date**: 2026-04-16
**Input**: Design documents from `/specs/004-slap-enroll/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

**Tests**: No TDD requested. Unit tests included as final validation (Phase 8).

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1-US5) from spec.md

## Path Conventions

All paths relative to `biometric-bridge/`:
- Handler: `internal/api/slap_enroll.go`
- Router: `internal/api/router.go`
- Broker: `internal/events/broker.go`
- Tests: `internal/api/slap_enroll_test.go`

---

## Phase 1: Setup

**Purpose**: No project initialization needed - extending existing codebase.

- [ ] T001 Verify existing SlapScan interface in `internal/driver/driver.go` has required types (SlapScanResult, CaptureMode, ErrSlapNotSupported)
- [ ] T002 Verify demo driver implements SlapScan in `internal/driver/demo/driver.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Infrastructure needed before ANY user story can be implemented.

**⚠️ CRITICAL**: US1-US5 cannot begin until this phase is complete.

- [ ] T003 Add Emit() method to events.Broker in `internal/events/broker.go` for direct event injection from handlers
- [ ] T004 Create `internal/api/slap_enroll.go` with request/response type definitions per data-model.md
- [ ] T005 Add handler constants (defaultSlapMinQuality=60, defaultSlapMaxRetries=3, timeouts) in `internal/api/slap_enroll.go`
- [ ] T006 Implement request validation function (validateSlapEnrollRequest) in `internal/api/slap_enroll.go`
- [ ] T007 Implement helper functions (expectedFingerCount, mode validation) in `internal/api/slap_enroll.go`

**Checkpoint**: Foundation ready - handler types defined, validation ready.

---

## Phase 3: User Story 1 - Right-Hand Four-Finger Enrollment (Priority: P1) 🎯 MVP

**Goal**: Capture 2 impressions with mode=right_four, return 4 finger templates per impression.

**Independent Test**: `curl POST /api/slap-enroll` with `mode=right_four`, verify response has `ok=true`, 2 impressions, 4 fingers each.

### Implementation for User Story 1

- [ ] T008 [US1] Create NewSlapEnrollHandler factory function in `internal/api/slap_enroll.go`
- [ ] T009 [US1] Implement device lock acquisition using device.Registry in `internal/api/slap_enroll.go`
- [ ] T010 [US1] Implement 2-impression capture loop calling driver.SlapScan() in `internal/api/slap_enroll.go`
- [ ] T011 [US1] Implement driver result to API response conversion (base64 encoding) in `internal/api/slap_enroll.go`
- [ ] T012 [US1] Implement timeout handling with context (60s overall, 20s per capture) in `internal/api/slap_enroll.go`
- [ ] T013 [US1] Implement error response helpers (writeEnrollError, etc.) in `internal/api/slap_enroll.go`
- [ ] T014 [US1] Register route `POST /api/slap-enroll` in `internal/api/router.go`
- [ ] T015 [US1] Add structured logging for enrollment start, completion in `internal/api/slap_enroll.go`

**Checkpoint**: US1 complete - basic right_four enrollment works with 2 impressions, no quality validation yet.

---

## Phase 4: User Story 2 - Quality Retry on Poor Capture (Priority: P1)

**Goal**: Validate per-finger quality, retry up to maxRetries times, emit enrollment_retry events.

**Independent Test**: Set minQuality=90 (high), verify retry events emitted, totalRetries > 0 on success or 422 on failure.

### Implementation for User Story 2

- [ ] T016 [US2] Implement validateQuality() function checking each finger against minQuality in `internal/api/slap_enroll.go`
- [ ] T017 [US2] Define qualityFailure struct for tracking failed fingers in `internal/api/slap_enroll.go`
- [ ] T018 [US2] Implement per-impression retry loop with attempt counter in `internal/api/slap_enroll.go`
- [ ] T019 [US2] Implement enrollment_retry event emission using broker.Emit() in `internal/api/slap_enroll.go`
- [ ] T020 [US2] Implement 422 response for quality failure with failedFingers array in `internal/api/slap_enroll.go`
- [ ] T021 [US2] Track and return totalRetries in success response in `internal/api/slap_enroll.go`
- [ ] T022 [US2] Add logging for retry attempts with finger quality details in `internal/api/slap_enroll.go`

**Checkpoint**: US2 complete - quality validation works, retries emit events, totalRetries tracked.

---

## Phase 5: User Story 3 - Left-Hand and Thumbs Enrollment (Priority: P2)

**Goal**: Support mode=left_four and mode=two_thumbs with correct finger positions.

**Independent Test**: Call with mode=left_four, verify fingers include left_index, left_middle, left_ring, left_little. Call with mode=two_thumbs, verify left_thumb, right_thumb.

### Implementation for User Story 3

- [ ] T023 [US3] Verify mode parsing handles all three modes (right_four, left_four, two_thumbs) in `internal/api/slap_enroll.go`
- [ ] T024 [US3] Verify expectedFingerCount returns correct values (4, 4, 2) for each mode in `internal/api/slap_enroll.go`
- [ ] T025 [US3] Add mode to response impression objects in `internal/api/slap_enroll.go`

**Checkpoint**: US3 complete - all three capture modes work correctly.

---

## Phase 6: User Story 4 - Strict vs Lenient Missing Finger Handling (Priority: P2)

**Goal**: Reject (strict=true) or accept (strict=false) when detected fingers < expected.

**Independent Test**: With mode=right_four, if 3 fingers detected: strict=true returns 422, strict=false returns 200 with 3 fingers.

### Implementation for User Story 4

- [ ] T026 [US4] Implement checkFingerCount() function comparing detected vs expected in `internal/api/slap_enroll.go`
- [ ] T027 [US4] Handle strict pointer default (nil → true) in request parsing in `internal/api/slap_enroll.go`
- [ ] T028 [US4] Implement 422 response for strict mode failure with detected/expected counts in `internal/api/slap_enroll.go`
- [ ] T029 [US4] In lenient mode (strict=false), proceed with available fingers in `internal/api/slap_enroll.go`
- [ ] T030 [US4] Add logging for finger count validation results in `internal/api/slap_enroll.go`

**Checkpoint**: US4 complete - strict/lenient mode works for partial enrollments.

---

## Phase 7: User Story 5 - Unsupported Device Handling (Priority: P3)

**Goal**: Return HTTP 501 when driver returns ErrSlapNotSupported.

**Independent Test**: Call slap-enroll against BS2 device (or mock), verify 501 response.

### Implementation for User Story 5

- [ ] T031 [US5] Check for driver.ErrSlapNotSupported error from SlapScan() in `internal/api/slap_enroll.go`
- [ ] T032 [US5] Return 501 Not Implemented with standard error message in `internal/api/slap_enroll.go`
- [ ] T033 [US5] Add logging for unsupported driver attempts in `internal/api/slap_enroll.go`

**Checkpoint**: US5 complete - BS2 and other non-slap devices return clean 501.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Testing, documentation, validation.

- [ ] T034 [P] Create unit tests for validateSlapEnrollRequest() in `internal/api/slap_enroll_test.go`
- [ ] T035 [P] Create unit tests for validateQuality() in `internal/api/slap_enroll_test.go`
- [ ] T036 [P] Create unit tests for expectedFingerCount() in `internal/api/slap_enroll_test.go`
- [ ] T037 [P] Create unit tests for checkFingerCount() in `internal/api/slap_enroll_test.go`
- [ ] T038 Create integration test with demo driver for full enrollment flow in `internal/api/slap_enroll_test.go`
- [ ] T039 Run quickstart.md validation with demo driver
- [ ] T040 Manual hardware testing with RealScan G10 (all modes, quality retry, strict/lenient)

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (Setup) → Phase 2 (Foundational) → [User Stories in parallel or priority order] → Phase 8 (Polish)
```

- **Setup (Phase 1)**: Verification only, no blocking
- **Foundational (Phase 2)**: BLOCKS all user stories - types and helpers must exist first
- **User Stories (Phases 3-7)**: Depend on Phase 2 completion
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

| Story | Priority | Depends On | Independent? |
|-------|----------|------------|--------------|
| US1 (Right-Four) | P1 | Phase 2 | ✅ Yes |
| US2 (Quality Retry) | P1 | Phase 2 + T008-T015 (US1 handler skeleton) | Builds on US1 |
| US3 (Left/Thumbs) | P2 | Phase 2 | ✅ Yes (mode validation only) |
| US4 (Strict/Lenient) | P2 | Phase 2 | ✅ Yes (finger count logic only) |
| US5 (Unsupported) | P3 | Phase 2 | ✅ Yes (error handling only) |

**Note**: US2 builds on US1's handler structure. US3, US4, US5 can be developed in parallel with US1 as they add independent logic.

### Within Each User Story

- Earlier tasks define structure used by later tasks
- Tasks within same story are sequential unless marked [P]
- Commit after each logical group

### Parallel Opportunities

**Phase 2 (Foundational)**:
```
T003 (broker.Emit) can run parallel to T004-T007 (handler setup)
```

**Phase 8 (Testing)**:
```
T034, T035, T036, T037 all marked [P] - unit tests for different functions
```

---

## Parallel Example: Phase 2 Foundation

```bash
# These can run in parallel (different files):
Task T003: "Add Emit() to broker.go"
Task T004: "Create slap_enroll.go with types"
```

## Parallel Example: Phase 8 Testing

```bash
# All unit test tasks can run in parallel:
Task T034: "Unit tests for validateSlapEnrollRequest()"
Task T035: "Unit tests for validateQuality()"
Task T036: "Unit tests for expectedFingerCount()"
Task T037: "Unit tests for checkFingerCount()"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (verification)
2. Complete Phase 2: Foundational (types, helpers)
3. Complete Phase 3: User Story 1 (basic enrollment)
4. **STOP and VALIDATE**: Test right_four enrollment with demo driver
5. Deploy/demo if ready - basic functionality works

### Incremental Delivery

| Milestone | Stories Complete | Capability |
|-----------|-----------------|------------|
| MVP | US1 | Basic right_four enrollment (2 impressions) |
| v1.1 | US1 + US2 | Quality validation with retry |
| v1.2 | US1-US4 | All modes, strict/lenient |
| v1.3 | US1-US5 + Tests | Full feature with error handling and tests |

### Recommended Sequence

1. **MVP Sprint**: Phase 1 → Phase 2 → Phase 3 (US1) → Validate
2. **Quality Sprint**: Phase 4 (US2) → Validate
3. **Modes Sprint**: Phase 5 (US3) + Phase 6 (US4) in parallel → Validate  
4. **Hardening Sprint**: Phase 7 (US5) + Phase 8 (Tests) → Final validation

---

## Notes

- All implementation is in single file `internal/api/slap_enroll.go` (following existing handler pattern)
- Broker modification is minimal (single method addition)
- Router modification is single line (route registration)
- Demo driver already implements SlapScan - no driver changes needed
- Tests are in Phase 8 as final validation (no TDD requested)
- Manual hardware test (T040) is critical - do not skip
