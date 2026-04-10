# Feature Specification: Biometric Bridge

**Feature Branch**: `001-biometric-bridge`  
**Created**: 2026-04-10  
**Status**: Draft  
**Input**: User description: "Build a Biometric Bridge — a lightweight local process on a workstation that gives a web application's frontend authenticated access to one or more Suprema fingerprint readers for enrollment, scanning, device listing, and real-time event streaming, with pluggable hardware driver support and JWT-based security."

## Clarifications

### Session 2026-04-10

- Q: What happens when the bridge receives a scan request while another scan is already in progress on the same device? → A: Reject with "device busy" error — the second request fails immediately.
- Q: How long should the bridge wait for a user to place their finger before timing out a scan? → A: 10 seconds — balanced for typical visitor-facing workflows.
- Q: How does the system behave when the configured authentication key file is missing or corrupted at startup? → A: Refuse to start — exit with a clear error identifying the missing/invalid key file.
- Q: Should the bridge limit the number of concurrent event stream connections? → A: Limit to 10 concurrent connections — reject with error beyond that.
- Q: What level of observability (logging) should the bridge provide? → A: Structured logging with configurable verbosity (error/info/debug levels).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scan a Fingerprint from the Web App (Priority: P1)

A frontline operator (e.g., receptionist) uses the web application to capture a visitor's fingerprint. The operator clicks "Scan" in the browser, the web app requests a fingerprint from the locally-running bridge, the visitor places their finger on the reader, and the raw fingerprint data is returned to the web app for server-side processing. The operator sees immediate feedback throughout — prompt to place finger, scan in progress, and result received.

**Why this priority**: Fingerprint scanning is the core value proposition. Without it, no other biometric workflow is possible. Every downstream use case (enrollment, verification, identification) depends on the ability to capture a fingerprint from the browser.

**Independent Test**: Can be fully tested by sending a scan request to the bridge with a valid token and a connected reader. Delivers a fingerprint template back to the caller, confirming end-to-end hardware access from the browser.

**Acceptance Scenarios**:

1. **Given** the bridge is running and a reader is connected, **When** the web app sends an authenticated scan request specifying a device, **Then** the bridge prompts the reader to capture a fingerprint and returns the raw template data and a quality score to the web app.
2. **Given** the bridge is running and a reader is connected, **When** a scan request is sent without valid authentication, **Then** the bridge rejects the request and returns an authentication error.
3. **Given** the bridge is running, **When** a scan request specifies a device that is not connected, **Then** the bridge returns a clear error indicating the device is unavailable.
4. **Given** the bridge is running and a reader is connected, **When** the user does not place their finger within 10 seconds, **Then** the bridge returns an error indicating the scan timed out.
5. **Given** a scan is already in progress on a device, **When** another scan request arrives for the same device, **Then** the bridge rejects the second request with a "device busy" error.

---

### User Story 2 - Enroll a User's Fingerprint (Priority: P1)

An operator uses the web application to enroll a user's fingerprint. The enrollment process captures two impressions of the same finger to create a reliable template. The operator initiates enrollment from the browser, the user places their finger twice when prompted, and the enrollment is stored on the device for the specified user.

**Why this priority**: Enrollment is the prerequisite for all future fingerprint-based interactions for a user. It is equally critical as scanning because without enrollment, there are no stored templates to verify against.

**Independent Test**: Can be fully tested by sending an enrollment request with a user identifier, having a person scan their finger twice on the reader, and confirming the enrollment completes successfully.

**Acceptance Scenarios**:

1. **Given** the bridge is running and a reader is connected, **When** the web app sends an authenticated enrollment request with a user ID, user name, and device name, **Then** the bridge captures two fingerprint impressions and enrolls the user on the specified device.
2. **Given** an enrollment is in progress, **When** the user fails to provide a second impression within 10 seconds, **Then** the bridge returns an error indicating enrollment was incomplete.
3. **Given** a valid enrollment request, **When** the specified device is not connected, **Then** the bridge returns an error indicating the device is unavailable.
4. **Given** an enrollment is in progress on a device, **When** another scan or enrollment request arrives for the same device, **Then** the bridge rejects the second request with a "device busy" error.

---

### User Story 3 - Monitor Real-Time Scan Events (Priority: P2)

A supervisor or the web application itself subscribes to a live event stream from the bridge to monitor fingerprint scan activity across all connected devices in real time. Events appear as they happen — a finger is scanned, a device reconnects, or an error occurs — enabling the web app to update its UI immediately without polling.

**Why this priority**: Real-time event streaming enables responsive user interfaces and operational dashboards. While the core scan and enroll workflows can function with request-response alone, event streaming significantly improves user experience and is essential for multi-device monitoring.

**Independent Test**: Can be tested by opening an event stream connection and performing a scan on any connected reader. The event should appear in the stream within seconds, confirming real-time delivery.

**Acceptance Scenarios**:

1. **Given** the bridge is running and the web app has an open event stream connection, **When** a fingerprint is scanned on any connected device, **Then** the web app receives a scan event containing the device name, user ID, and event details within 2 seconds.
2. **Given** an active event stream, **When** a device temporarily loses connection, **Then** the web app receives status events (reconnecting, connected) rather than the stream being dropped, allowing the UI to display device status.
3. **Given** an active event stream, **When** a device encounters an error, **Then** the web app receives an error event with a descriptive message identifying the affected device.
4. **Given** a user opens the event stream without valid authentication, **Then** the connection is rejected.
5. **Given** 10 event stream connections are already open, **When** an 11th authenticated client attempts to connect, **Then** the bridge rejects the connection with an error indicating the maximum number of subscribers has been reached.
6. **Given** an active event stream connection, **When** the authentication token used to open the connection expires, **Then** the bridge closes the event stream connection and the client must reconnect with a fresh token.

---

### User Story 4 - List Connected Devices (Priority: P2)

An operator or administrator views the list of all fingerprint readers currently connected to the bridge, including device names, models, and capabilities. This allows the web app to present a device picker before initiating a scan or enrollment.

**Why this priority**: Device listing supports scan and enrollment by letting users select which reader to use. In single-device setups this is less critical, but for multi-device environments it is necessary for a complete workflow.

**Independent Test**: Can be tested by sending a device list request and confirming all configured and connected readers are returned with their metadata.

**Acceptance Scenarios**:

1. **Given** the bridge is running with multiple readers connected, **When** the web app sends an authenticated device list request, **Then** the bridge returns all connected devices with their names, models, and capability information.
2. **Given** the bridge is running with no readers connected (startup failure), **When** the bridge cannot connect to all configured devices, **Then** the bridge does not start accepting requests and exits with a clear error.

---

### User Story 5 - Authenticate All Bridge Requests (Priority: P1)

Every request from the web app to the bridge must carry a valid, short-lived token issued by the backend server. The bridge verifies the token's signature, expiry, issuer, and audience on every request. This ensures that only authenticated users of the web application can access fingerprint hardware, and that tokens cannot be reused across different services. For long-lived event stream connections, the bridge enforces token expiry by closing the connection when the token expires, requiring the client to reconnect with a fresh token.

**Why this priority**: Security is non-negotiable for a system handling biometric data. Without authentication, any process on the workstation could access fingerprint readers, creating a significant security and privacy risk.

**Independent Test**: Can be tested by sending requests with valid tokens (accepted), expired tokens (rejected), tokens with wrong audience (rejected), and no token (rejected).

**Acceptance Scenarios**:

1. **Given** the bridge is running, **When** a request arrives with a valid, non-expired token containing the correct issuer and audience, **Then** the request is processed normally.
2. **Given** the bridge is running, **When** a request arrives with an expired token, **Then** the request is rejected with an authentication error.
3. **Given** the bridge is running, **When** a request arrives with a token issued for a different audience or by an unrecognized issuer, **Then** the request is rejected.
4. **Given** the bridge is running, **When** a request arrives with no token, **Then** the request is rejected with an authentication error.
5. **Given** the bridge is running, **When** a health check request is sent (no authentication required), **Then** the bridge responds confirming it is operational.
6. **Given** an event stream connection is open with a token that has 15 minutes TTL, **When** the token expires, **Then** the bridge closes the event stream connection, requiring the client to reconnect with a new token.

---

### User Story 6 - Automatic Recovery from Device Disconnection (Priority: P3)

When a fingerprint reader becomes temporarily unreachable (network glitch, USB disconnect/reconnect, device restart), the bridge automatically attempts to reconnect using increasing wait intervals. During reconnection, the event stream remains open and reports device status. Other connected devices continue operating normally.

**Why this priority**: Automatic recovery reduces operator intervention and improves system reliability. While the core scan/enroll workflows function without it (they would simply fail until the device is manually reconnected), automatic recovery is important for production deployments.

**Independent Test**: Can be tested by disconnecting a reader, observing reconnection status events in the stream, then reconnecting the reader and confirming the bridge re-establishes the connection.

**Acceptance Scenarios**:

1. **Given** a reader becomes unreachable, **When** the bridge detects the disconnection, **Then** it begins automatic reconnection attempts with increasing wait times (starting at 1 second, capping at 2 minutes).
2. **Given** a reader is reconnecting, **When** the event stream is open, **Then** the web app receives status events for each reconnection attempt indicating the attempt number and wait time.
3. **Given** one reader disconnects in a multi-device setup, **When** other readers are still connected, **Then** scan and enrollment operations on the remaining readers continue to work normally.
4. **Given** a reader successfully reconnects, **When** the connection is re-established, **Then** the bridge emits a "connected" event and resumes normal operations for that device.

---

### User Story 7 - Pluggable Hardware Driver Support (Priority: P3)

The bridge supports multiple hardware communication methods through interchangeable drivers. An administrator selects which driver to use at configuration time. The web application's behavior and the bridge's external interface remain identical regardless of which driver is active — the same requests, responses, and events work with any driver.

**Why this priority**: Driver pluggability future-proofs the system and supports diverse deployment environments. The initial deployment may only need one driver, but the architecture must support swapping without changing the web application.

**Independent Test**: Can be tested by configuring the bridge with different drivers and confirming that the same set of operations (scan, enroll, list devices, events) produce consistent behavior.

**Acceptance Scenarios**:

1. **Given** the bridge is configured with any supported driver, **When** the web app sends a scan request, **Then** the response format and behavior are identical regardless of driver.
2. **Given** the bridge is configured with a driver, **When** the bridge starts up, **Then** it connects to all configured devices using the selected driver before accepting requests.
3. **Given** the bridge is running, **When** the web app interacts with the bridge, **Then** there is no way for the web app to determine which driver is in use — the interface is identical.

---

### Edge Cases

- If a scan or enrollment is in progress on a device and another request arrives for the same device, the bridge rejects the second request with a "device busy" error.
- If the configured authentication key file is missing or corrupted at startup, the bridge refuses to start and exits with a clear error identifying the problem.
- If a device is configured but physically absent (never connects) during startup, the bridge fails startup and exits with a clear error (all-or-nothing startup policy per FR-011).
- If a token has the correct signature but is missing required claims (issuer or audience), the bridge rejects the request with an authentication error.
- If the event stream receives a burst of events from multiple devices simultaneously, all events are delivered to subscribers in arrival order without dropping any.
- If enrollment captures two impressions that differ significantly (e.g., different fingers), the device-level quality check may reject the enrollment; the bridge returns the device's error to the caller.
- If the authentication token expires while an event stream connection is open, the bridge closes the connection; the client must reconnect with a fresh token.
- If the configuration file is malformed, contains unrecognized fields, or references an unknown driver, the bridge refuses to start and exits with a clear error describing the configuration problem.
- If the bridge process receives a termination signal while operations are in progress, it closes event stream connections, cancels in-progress scans/enrollments, releases device connections, and exits cleanly.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST capture a single fingerprint impression from a specified device and return the raw template data and quality score to the requesting web application. The scan operation MUST time out after 10 seconds if no finger is placed.
- **FR-002**: System MUST enroll a user by capturing two fingerprint impressions from a specified device and storing the resulting template on that device against the provided user identifier and name. Each impression MUST time out after 10 seconds if no finger is placed.
- **FR-003**: System MUST return a list of all connected devices with their names, models, and capability information upon request.
- **FR-004**: System MUST stream real-time events (scan occurrences, connection status changes, errors) from all connected devices over a persistent connection to subscribed clients, up to a maximum of 10 concurrent event stream connections. Connections beyond this limit MUST be rejected with an error.
- **FR-005**: System MUST validate a short-lived, server-issued authentication token on every request (except health checks), rejecting requests with missing, expired, invalid, or incorrectly scoped tokens (including tokens missing required issuer or audience claims). For event stream connections, the system MUST close the connection when the token expires; the client must reconnect with a fresh token.
- **FR-006**: System MUST only accept connections from the local machine — it MUST NOT be reachable from the network.
- **FR-007**: System MUST restrict cross-origin requests to a single configured web application domain.
- **FR-008**: System MUST automatically attempt to reconnect to a device that becomes unreachable, using exponential backoff starting at 1 second and capping at 2 minutes.
- **FR-009**: System MUST emit status events (reconnecting with attempt count and wait time, connected, error) during device reconnection, keeping the event stream open throughout.
- **FR-010**: System MUST support multiple simultaneously connected fingerprint readers, with each device independently operational (a failure on one device does not affect others after startup).
- **FR-011**: System MUST connect to all configured devices at startup before accepting any requests; if any device fails to connect, the system MUST exit with a clear error.
- **FR-012**: System MUST provide a health check endpoint that requires no authentication, allowing installers and monitoring tools to confirm the bridge is running.
- **FR-013**: System MUST support interchangeable hardware drivers selected at configuration time, with no change to the external interface regardless of which driver is active.
- **FR-014**: System MUST NOT store, cache, or persist any biometric data — all fingerprint templates are returned to the caller immediately and not retained.
- **FR-015**: System MUST reject scan or enrollment requests for a device that already has an operation in progress, returning a "device busy" error immediately.
- **FR-016**: System MUST provide structured logging with configurable verbosity levels (error, info, debug), allowing administrators to control log detail for troubleshooting.
- **FR-017**: System MUST refuse to start if the authentication key file is missing or invalid, exiting with a clear error identifying the problem.
- **FR-018**: System MUST validate the configuration file at startup, refusing to start if the file is malformed, contains invalid values, or references an unrecognized driver. The error message MUST identify the specific configuration problem.
- **FR-019**: System MUST shut down gracefully when receiving a termination signal: close all event stream connections, cancel any in-progress scan or enrollment operations, release all device connections, and exit cleanly.

### Key Entities

- **Device**: A fingerprint reader connected to the bridge. Has a human-readable name (from configuration), model, firmware version, and capability flags (e.g., fingerprint supported). Each device is independently addressable for scan and enrollment operations. A device can be in one of three states: idle (ready for operations), busy (scan or enrollment in progress), or disconnected (reconnecting).
- **Fingerprint Template**: Raw biometric data captured from a device during a scan or enrollment. Represented as opaque binary data with an associated quality score (0–100). The bridge never interprets, matches, or stores templates.
- **User (Enrollment Context)**: A person whose fingerprint is being enrolled on a device. Identified by a user ID and name provided by the web application. The bridge does not manage user accounts — it only passes identifiers to the device during enrollment.
- **Event**: A real-time occurrence from a connected device. Types include scan (finger detected), connection status changes (reconnecting, connected), and errors. Each event identifies the originating device.
- **Authentication Token**: A short-lived credential issued by the backend server and presented by the web application to authorize bridge requests. Validated for signature, expiry, issuer, and audience on every request. For event stream connections, the token's expiry is enforced continuously — the connection is closed when the token expires.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can capture a fingerprint from a connected reader and receive the result in the web application within 5 seconds of placing their finger.
- **SC-002**: Operators can complete a full two-impression enrollment in under 30 seconds.
- **SC-003**: The web application receives real-time scan events within 2 seconds of the physical scan occurring.
- **SC-004**: The bridge maintains stable connections to at least 2 simultaneously connected readers without degraded performance.
- **SC-005**: 100% of requests without valid authentication tokens are rejected — zero unauthorized access to biometric operations, including event stream connections with expired tokens.
- **SC-006**: The bridge automatically recovers from temporary device disconnections without operator intervention in 95% of transient failure scenarios.
- **SC-007**: The external interface (requests, responses, events) behaves identically regardless of which hardware driver is configured — zero driver-specific behavior visible to the web application.
- **SC-008**: Health check responses return within 500 milliseconds, enabling reliable automated monitoring.
- **SC-009**: The bridge is unreachable from any machine other than the one it is running on — zero external network exposure.
- **SC-010**: The bridge supports up to 10 concurrent event stream subscribers without degraded event delivery latency.
- **SC-011**: The bridge is ready to accept requests within 15 seconds of process launch (assuming all configured devices are reachable on the network).

## Assumptions

- The web application frontend and the bridge always run on the same physical workstation. The bridge is never accessed over a network.
- The backend server that issues authentication tokens is a separate, remote system. The bridge has no direct communication with the backend — it only validates tokens using a pre-shared public key.
- The bridge does not perform any fingerprint matching, verification, or identification. All biometric comparison logic resides on the server side.
- Fingerprint readers are Suprema BioStar 2 compatible devices. Other manufacturers' readers are out of scope.
- A single driver is active at any given time. The bridge does not run multiple drivers simultaneously.
- The bridge is deployed on workstations running Windows, macOS, or Linux. Cross-platform support is expected.
- The configured authentication key is pre-deployed to the workstation by an installer or administrator before the bridge starts. If missing, the bridge will not start.
- Token TTL of 15 minutes is sufficient for typical operator sessions. The web application is responsible for refreshing tokens before they expire, including reconnecting event streams with fresh tokens.
- The bridge does not manage user accounts, permissions, or roles. It relies entirely on the token's validity to authorize requests.
- At startup, all configured devices must be reachable. The bridge does not support a "partial startup" mode where some devices are unavailable.
- Only one biometric operation (scan or enrollment) can be in progress on a single device at a time. Concurrent requests to the same device are rejected.
- Structured logs are written to standard output/error. Log aggregation and forwarding are the responsibility of the deployment environment.
