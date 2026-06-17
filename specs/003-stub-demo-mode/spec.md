# Feature Specification: Stub/Demo Mode

**Feature Branch**: `003-stub-demo-mode`
**Created**: 2026-04-14
**Status**: Draft
**Input**: User description: "Implement a stub/demo mode for the biometric bridge that serves mock data and simulated device events so the web application can be developed and tested without real hardware"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start Bridge in Demo Mode (Priority: P1)

A developer starts the biometric bridge with a demo flag. The bridge starts the HTTP server and WebSocket event stream without requiring real biometric hardware, SDK libraries, or configuration files. All API endpoints respond with realistic mock data, allowing the web application to be fully exercised.

**Why this priority**: Without this, developers cannot iterate on the web frontend without physical devices connected — blocking all UI development and testing.

**Independent Test**: Run the bridge with `--demo` flag and no config file, verify the HTTP server starts, `/healthz` returns 200, and `/api/devices` returns mock device data.

**Acceptance Scenarios**:

1. **Given** no config file exists, **When** the developer runs the bridge with `--demo`, **Then** the HTTP server starts on the default address with sensible defaults (no error about missing config, driver libraries, or hardware).
2. **Given** a config file exists, **When** the developer runs the bridge with `--demo`, **Then** the bridge uses the config's listen address and CORS origin but ignores driver, device, and auth validation (no SDK loaded, no hardware required).
3. **Given** the bridge is running in demo mode, **When** a request hits `/healthz`, **Then** a 200 response is returned indicating the bridge is operational.
4. **Given** the bridge is running in demo mode, **When** the developer sends a SIGINT/SIGTERM, **Then** the bridge shuts down gracefully.

---

### User Story 2 - Query Mock Devices (Priority: P1)

A developer fetches the device list from the running demo bridge. The response returns one or more realistic mock devices with names, models, IDs, and firmware versions matching what a real deployment would produce.

**Why this priority**: The web application's device listing page cannot be built without this endpoint returning realistic data.

**Independent Test**: Start the bridge in demo mode, send `GET /api/devices` with a valid JWT, verify the response contains mock devices with expected fields.

**Acceptance Scenarios**:

1. **Given** the bridge is running in demo mode, **When** a developer sends `GET /api/devices` with a valid token, **Then** the response includes at least one mock device with name, model, ID, and firmware version fields populated.
2. **Given** the bridge is running in demo mode with a config file specifying devices, **When** a developer sends `GET /api/devices`, **Then** the response returns devices matching the names from the config file (with mock metadata).

---

### User Story 3 - Perform Mock Scans (Priority: P1)

A developer triggers a scan via the API from the web application. The demo bridge returns a synthetic fingerprint template with a realistic quality score and dimensions, simulating a successful biometric capture.

**Why this priority**: Scan is the primary user-facing interaction — the web UI cannot be developed without this working end-to-end.

**Independent Test**: Start bridge in demo mode, send `POST /api/scan` with a valid JWT and device ID, verify the response contains a base64 template, quality score, width, and height.

**Acceptance Scenarios**:

1. **Given** the bridge is running in demo mode, **When** a developer sends `POST /api/scan` with a valid device ID and token, **Then** the response returns a 200 with a base64-encoded template, quality score (0–100), width, and height.
2. **Given** the bridge is running in demo mode, **When** a developer sends `POST /api/scan` with an unknown device ID, **Then** the response returns a 400 error indicating the device was not found.
3. **Given** a scan is in progress, **When** a second scan request arrives for the same device, **Then** the response returns a 409 error indicating the device is busy.
4. **Given** the bridge is running in demo mode, **When** a developer sends `POST /api/scan` with a finger position, **Then** the response includes the finger position in the result.

---

### User Story 4 - Perform Mock Enrollments (Priority: P2)

A developer triggers an enrollment via the API. The demo bridge simulates a successful multi-impression enrollment, returning an acknowledgment.

**Why this priority**: Enrollment is a secondary workflow. Developers need scans working first, then can build enrollment UI.

**Independent Test**: Start bridge in demo mode, send `POST /api/enroll` with valid parameters, verify 200 response with `ok: true`.

**Acceptance Scenarios**:

1. **Given** the bridge is running in demo mode, **When** a developer sends `POST /api/enroll` with a valid device ID, user ID, user name, and token, **Then** the response returns `{"ok": true, "userId": "<provided userId>"}`.
2. **Given** the bridge is running in demo mode, **When** a developer sends `POST /api/enroll` with a finger position list, **Then** the enrollment simulates a brief delay (to mimic real hardware timing) before returning success.

---

### User Story 5 - Receive Simulated Real-Time Events (Priority: P1)

A developer connects the web application to the WebSocket event stream. The demo bridge emits a realistic sequence of simulated events (device connected, scan events, periodic heartbeats) at configurable intervals, allowing the web UI's event log and notification features to be developed.

**Why this priority**: Real-time events are a core feature of the bridge. The web application's event panel and notifications cannot be tested without a live event stream.

**Independent Test**: Start bridge in demo mode, connect to `/events` WebSocket with a valid token, verify events of type "connected" and "scan" are received.

**Acceptance Scenarios**:

1. **Given** the bridge is running in demo mode, **When** a developer connects to the WebSocket endpoint with a valid token, **Then** the bridge immediately emits a "connected" event for each mock device.
2. **Given** a WebSocket connection is active, **When** time passes, **Then** the bridge periodically emits synthetic scan events (every few seconds by default).
3. **Given** the bridge is running in demo mode, **When** a developer triggers a scan via `POST /api/scan`, **Then** a corresponding "scan" event is emitted on the WebSocket stream.
4. **Given** the bridge is running in demo mode, **When** a developer triggers an enrollment via `POST /api/enroll`, **Then** a corresponding enrollment-related event is emitted on the WebSocket stream.

---

### User Story 6 - Perform Mock Slap Scans (Priority: P2)

A developer triggers a slap scan via the API. The demo bridge returns a synthetic slap image and segmented finger results.

**Why this priority**: Slap scanning is a specialized workflow used by some device models. Lower priority than single-finger scan.

**Independent Test**: Start bridge in demo mode, send `POST /api/slap-scan` with a valid mode, verify response contains slap image and finger array.

**Acceptance Scenarios**:

1. **Given** the bridge is running in demo mode, **When** a developer sends `POST /api/slap-scan` with a valid device ID and mode, **Then** the response returns a 200 with slap image data and an array of finger results, each with image, quality, dimensions, and finger position.

---

### User Story 7 - Authenticate with Demo JWT (Priority: P1)

A developer needs to make authenticated API calls to the demo bridge without generating real JWTs with an ECDSA key pair. The demo mode provides a built-in way to authenticate — either by accepting any token, or by providing a default test token.

**Why this priority**: All API endpoints (except `/healthz`) require JWT auth. Without a way to authenticate, the web application cannot call any endpoint.

**Independent Test**: Start bridge in demo mode, use the provided demo credentials/token to call `/api/devices`, verify 200 response.

**Acceptance Scenarios**:

1. **Given** the bridge is running in demo mode, **When** the bridge starts, **Then** it prints a demo JWT (or the secret/key needed to generate one) to the console so the developer can use it immediately.
2. **Given** the bridge is running in demo mode with a config file that specifies auth fields, **When** a developer sends a request with a valid JWT signed by the configured key, **Then** the request is authenticated normally.
3. **Given** the bridge is running in demo mode without a config file, **When** a developer sends a request with the printed demo token, **Then** the request is authenticated successfully.

---

### Edge Cases

- What happens when the web application sends concurrent scan requests to the same mock device? (Should return 409 device busy, consistent with real behavior.)
- What happens when the WebSocket connection drops during event streaming? (Should clean up subscriber and allow reconnection.)
- What happens when no config file exists and `--demo` is used? (Should use all-sensible defaults with no errors.)
- What happens when `--demo` and `--test` are both specified? (Demo mode should take precedence and ignore --test.)
- What happens when a device configured in the config file is referenced in a scan request during demo mode but the stub driver doesn't know about it? (Should return 400 device not found.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The bridge MUST support a `--demo` command-line flag that starts the HTTP server with a mock driver instead of a real hardware driver.
- **FR-002**: Demo mode MUST NOT require a valid config file. When no config is provided, the bridge MUST start with sensible defaults (listen on `127.0.0.1:7070`, mock devices, relaxed auth).
- **FR-003**: Demo mode MUST skip driver library loading, device connection, and SDK initialization — no native libraries or hardware are required.
- **FR-004**: The mock driver MUST implement the full driver interface: Connect, Scan, SlapScan, Enroll, ListDevices, Subscribe, Close.
- **FR-005**: `GET /api/devices` MUST return at least one mock device with realistic metadata (name, model, ID, firmware version, finger supported flag).
- **FR-006**: `POST /api/scan` MUST return a synthetic fingerprint template (base64-encoded random bytes), a quality score between 60–95, and realistic image dimensions.
- **FR-007**: `POST /api/enroll` MUST simulate a brief processing delay (0.5–1 second) then return success with the provided user ID.
- **FR-008**: `POST /api/slap-scan` MUST return a synthetic slap image and an array of individual finger results matching the requested mode.
- **FR-009**: The WebSocket event stream MUST emit simulated events at periodic intervals (configurable, default every 5 seconds) when no API actions are triggered.
- **FR-010**: Demo mode MUST emit a "connected" event for each mock device immediately upon WebSocket connection.
- **FR-011**: API-triggered actions (scan, enroll) MUST emit corresponding events on the WebSocket stream, consistent with real driver behavior.
- **FR-012**: Device locking MUST work identically to production mode — concurrent operations on the same device MUST return 409 (busy).
- **FR-013**: Demo mode MUST provide a way to authenticate API calls. When no config file is present, it MUST print a usable demo JWT to the console at startup.
- **FR-014**: Demo mode MUST respect the listen address and CORS origin from the config file when one is provided, allowing developers to match their web application's origin.
- **FR-015**: The mock driver MUST generate deterministic-looking but random data on each request (no cached responses).
- **FR-016**: Demo mode MUST log that it is running in demo/stub mode prominently at startup.
- **FR-017**: Demo mode MUST be clearly indicated in the `/healthz` response (e.g., include a `demo: true` field).
- **FR-018**: The mock driver MUST support device state transitions (idle ↔ busy ↔ disconnected) consistent with the device registry, so that the web application can display device state accurately.
- **FR-019**: Demo mode MUST NOT require a public key file. When no key is available, it MUST use a built-in test key pair or skip signature verification.

### Key Entities

- **Mock Device**: A virtual device with configurable name, model, ID, and firmware version. Supports the same state machine as real devices (idle, busy, disconnected).
- **Synthetic Scan Result**: A generated fingerprint template (random bytes), quality score (60–95), and dimensions (e.g., 300×400). Different on each call.
- **Synthetic Slap Result**: A generated slap image plus segmented finger results matching the requested capture mode.
- **Demo JWT**: A pre-generated JWT token printed at startup, using a built-in test key pair, valid for a configurable duration (default 1 hour).
- **Event Simulator**: A timer-based component that periodically emits synthetic device events (scan, connected, error) on the driver's subscription channel.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can start the bridge in demo mode with a single command (`--demo`) and zero configuration files, hardware, or SDK libraries.
- **SC-002**: All existing API endpoints (`/healthz`, `/api/devices`, `/api/scan`, `/api/slap-scan`, `/api/enroll`, `/events`) respond with realistic mock data within 2 seconds.
- **SC-003**: The WebSocket event stream delivers simulated events within 5 seconds of connection and continues at regular intervals.
- **SC-004**: A developer can complete a full scan workflow from the web application (authenticate, list devices, trigger scan, receive events) without any real hardware.
- **SC-005**: Demo mode startup time is under 1 second (no SDK initialization delays).
- **SC-006**: Device locking behavior in demo mode is identical to production mode (409 on concurrent operations).

## Assumptions

- Developers are running on the same machine as the bridge (localhost development workflow).
- The web application already has the ability to configure the bridge URL and provide JWT tokens.
- A single default mock device is sufficient for most development scenarios; additional mock devices can be configured via the config file.
- The built-in test key pair for demo JWT is acceptable for local development only and MUST NOT be used in production.
- Demo mode is for development and testing only — it MUST be clearly marked and never accidentally deployed to production.
- The existing `--test` flag serves a different purpose (single scan test with real hardware) and should not be modified.
- Event simulation timing does not need to be precisely calibrated — approximate intervals are acceptable.
- The stub driver is always compiled into the binary and selected at runtime via the `--demo` flag. It has zero SDK imports and negligible binary size.
