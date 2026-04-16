# Feature Specification: Slap Enrollment (Multi-Finger)

**Feature Branch**: `004-slap-enroll`  
**Created**: 2026-04-16  
**Status**: Draft  
**Input**: User description: "Add a new /api/slap-enroll endpoint to capture 2 impressions of 4 fingers simultaneously, with quality validation and retry logic."

## Overview

**Goal**: Add a new `POST /api/slap-enroll` endpoint to capture 2 impressions of multiple fingers simultaneously (slap capture), validate per-finger quality, and provide retry logic to speed up multi-finger enrollment.

**Why**: Sequential single-finger enrollment is slow for enrolling multiple fingers (~80 seconds for 4 fingers with 2 impressions each). Supporting slap enrollment for devices with a flat platen (RealScan G10) enables enrolling multiple fingers in parallel (~40 seconds) while maintaining the 2-impression reliability standard.

**Scope**: Bridge-side feature only. Widget integration is out-of-scope for this task.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Right-Hand Four-Finger Enrollment (Priority: P1)

An operator uses the web application to enroll a user's right hand (4 fingers) using the slap capture method. The user places all four fingers on the scanner platen twice (2 impressions), and the system validates quality and stores the resulting templates.

**Why this priority**: This is the primary use case for slap enrollment - capturing multiple fingers quickly while maintaining quality standards. It delivers the core speed improvement (50% time reduction).

**Independent Test**: Can be fully tested by calling POST /api/slap-enroll with mode=right_four and verifying 2 impressions with 4 finger templates each are returned.

**Acceptance Scenarios**:

1. **Given** a RealScan device is connected and available, **When** the operator sends POST /api/slap-enroll with deviceId, userId, userName, and mode=right_four, **Then** the bridge captures 2 slap impressions and returns response with ok=true, 2 impressions, each containing 4 finger templates with quality scores.

2. **Given** all captured fingers meet the quality threshold, **When** enrollment completes, **Then** totalRetries=0 in the response.

---

### User Story 2 - Quality Retry on Poor Capture (Priority: P1)

When fingerprint quality is below the configured threshold, the system automatically retries the capture up to the configured maximum retries before failing.

**Why this priority**: Quality validation ensures reliable templates; automatic retry improves UX by not failing immediately on a single poor placement.

**Independent Test**: Can be tested by configuring a low quality threshold and verifying retry events are emitted and totalRetries > 0 when quality improves on subsequent attempt.

**Acceptance Scenarios**:

1. **Given** minQuality is set to 60 and first capture produces a finger with quality 45, **When** the driver validates quality, **Then** it retries the capture and emits an "enrollment_retry" event.

2. **Given** the second attempt produces all fingers with quality >= 60, **When** validation passes, **Then** enrollment succeeds with totalRetries=1.

3. **Given** all retry attempts fail quality validation, **When** maxRetries is exhausted, **Then** return HTTP 422 with error listing low-quality fingers and their scores.

---

### User Story 3 - Left-Hand and Thumbs Enrollment (Priority: P2)

Operators can enroll left hand (4 fingers) or both thumbs using the same slap enrollment endpoint with different mode values.

**Why this priority**: Completes the full hand enrollment capability; follows naturally after right-hand enrollment works.

**Independent Test**: Can be tested by calling POST /api/slap-enroll with mode=left_four or mode=two_thumbs and verifying correct finger positions are returned.

**Acceptance Scenarios**:

1. **Given** mode=left_four, **When** enrollment completes, **Then** returned fingers include left_index, left_middle, left_ring, left_little.

2. **Given** mode=two_thumbs, **When** enrollment completes, **Then** returned fingers include left_thumb, right_thumb.

---

### User Story 4 - Strict vs Lenient Missing Finger Handling (Priority: P2)

Operators can choose whether to reject enrollment when fewer fingers are detected than expected (strict mode) or accept partial enrollment (lenient mode).

**Why this priority**: Accommodates users with missing or injured fingers while maintaining data integrity for standard enrollments.

**Independent Test**: Can be tested by placing only 3 fingers when mode=right_four and verifying strict=true rejects while strict=false accepts.

**Acceptance Scenarios**:

1. **Given** strict=true and mode=right_four, **When** only 3 fingers are detected, **Then** return HTTP 422 with error indicating expected 4, detected 3.

2. **Given** strict=false and mode=right_four, **When** only 3 fingers are detected, **Then** enrollment succeeds with 3 finger templates in each impression.

---

### User Story 5 - Unsupported Device Handling (Priority: P3)

When an operator attempts slap enrollment on a device that doesn't support slap capture (e.g., BS2), the system returns a clear error.

**Why this priority**: Error handling for unsupported configurations; not on the happy path but necessary for robustness.

**Independent Test**: Can be tested by attempting slap enrollment against a BS2 device and verifying HTTP 501 is returned.

**Acceptance Scenarios**:

1. **Given** the device is a BS2 reader (no slap support), **When** POST /api/slap-enroll is called, **Then** return HTTP 501 Not Implemented with message "slap enrollment not supported by this driver".

---

### Edge Cases

- What happens when the user removes fingers mid-capture? The SDK returns a timeout or partial detection; handled by strict/lenient mode.
- How does the system handle quality scoring failure? If SDK cannot compute quality, use SDK-reported quality or default to 0 and log warning.
- What if the device is busy with another operation? Return HTTP 409 Conflict (existing device registry behavior).
- What happens on timeout? Return HTTP 504 Gateway Timeout after 60s overall or 20s per slap capture.
- What if context is cancelled (client disconnect)? Abort capture and return context error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a new endpoint `POST /api/slap-enroll` that accepts deviceId, userId, userName, mode (required) and optional minQuality, maxRetries, strict parameters.

- **FR-002**: System MUST capture exactly 2 impressions per slap enrollment request (maintaining the 2-impression quality standard).

- **FR-003**: Each impression MUST include the full slap image (base64), dimensions (width, height), and an array of segmented finger templates with position, image data, dimensions, and NIST quality score.

- **FR-004**: System MUST validate per-finger quality when minQuality > 0, rejecting impressions where any detected finger has quality below threshold.

- **FR-005**: System MUST retry failed quality validations up to maxRetries times (default: 3, configurable), emitting an "enrollment_retry" event via WebSocket for each retry attempt.

- **FR-006**: System MUST support three capture modes: left_four (4 left-hand fingers), right_four (4 right-hand fingers), two_thumbs (both thumbs).

- **FR-007**: System MUST respect the strict parameter: when true, reject if detected fingers < expected; when false, accept partial enrollment.

- **FR-008**: System MUST return HTTP 501 Not Implemented for drivers that do not support slap capture (e.g., BS2).

- **FR-009**: System MUST enforce a timeout of 60 seconds overall and 20 seconds per slap capture to prevent indefinite hangs.

- **FR-010**: System MUST require valid JWT authentication (existing auth middleware) for the slap-enroll endpoint.

- **FR-011**: System MUST log enrollment start, completion, retries, and per-finger quality results with sufficient metadata for auditing.

- **FR-012**: System MUST acquire device lock before capture and release after completion to prevent concurrent operations (existing device registry).

### Key Entities

- **SlapEnrollRequest**: Enrollment request containing deviceId, userId, userName, mode, optional minQuality (0-100), maxRetries (0-10), strict (boolean).

- **SlapEnrollResult**: Response containing ok status, userId, impressions array, and totalRetries count.

- **SlapEnrollImpression**: Single impression containing impressionNumber (1 or 2), mode, slapImage (base64), dimensions, and fingers array.

- **SlapEnrolledFinger**: Individual finger data containing finger position (e.g., right_index), template (base64), dimensions, and quality score.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of successful slap enrollments (2 impressions of 4 fingers) complete within 50 seconds under normal device conditions (target: ~40 seconds baseline).

- **SC-002**: Quality validation correctly enforces configured thresholds: enrolled finger quality >= minQuality in >= 95% of successful enrollments.

- **SC-003**: Average retry attempts per successful enrollment <= 1 under typical operator conditions (proper finger placement).

- **SC-004**: BS2 devices return HTTP 501 Not Implemented for 100% of slap enrollment requests.

- **SC-005**: Zero breaking changes to existing `/api/enroll` single-finger enrollment behavior.

## Assumptions

- RealScan driver supports slap capture via `RS_TakeImageDataSegment` and automatic finger segmentation; BS2 driver does not support slap capture.
- Bridge returns base64-encoded templates in HTTP response; widget/server is responsible for persistent storage.
- Default quality threshold is 60 (NIST score); configurable via config.yaml.
- Default maxRetries is 3; configurable via config.yaml.
- Device registry and locking mechanism already exist and will be reused.
- WebSocket event emission for "enrollment_retry" uses existing event broker infrastructure.
- Demo driver will simulate slap enrollment for testing without hardware.

## Non-Goals

- Widget-side UI changes or template storage (out of scope for this task)
- Implementing dedicated server-side template database
- Changing existing single-finger enrollment (`/api/enroll`) behavior
- Supporting slap capture on BS2 devices (hardware limitation)
