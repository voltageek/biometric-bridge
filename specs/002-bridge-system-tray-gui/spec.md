# Feature Specification: Bridge System Tray GUI

**Feature Branch**: `002-bridge-system-tray-gui`
**Created**: 2026-04-12
**Status**: Draft
**Input**: User description: "Build a GUI for bridge. Will be a system tray application using Fyne (screenshots attached)"

## Clarifications

### Session 2026-04-12

- Q: How should the tray GUI integrate with the bridge — as an in-process Go library or as a subprocess? → A: Embed bridge as a Go library — the GUI binary imports bridge packages directly, runs bridge logic in-process.
- Q: How should the Connected User identity be obtained — the bridge is currently stateless with no session tracking? → A: Extract from the last JWT received by the bridge HTTP server (observer pattern — no session state added). The tray hooks into the auth middleware to capture the most recent authenticated request's claims.
- Q: Should the event log persist events across panel close/reopen and bridge restarts, or be ephemeral? → A: Ephemeral — in-memory ring buffer (last N events), cleared on bridge restart, panel shows events since last bridge start.
- Q: How should the "View Logs" detailed log viewer capture bridge logs written to stdout/stderr? → A: Custom slog handler writes to both an in-memory ring buffer (for the log viewer UI) and a log file on disk (for persistence and troubleshooting).

## UI Overview

The application presents a single floating panel titled **"The Kinetic Vault"** that appears when the user clicks the system tray icon. The panel is 340px wide, with a dark navy header (#0F1623) and a light gray body (#F4F5F8). It is not a traditional multi-window application — all primary information and controls live in this one panel. Secondary actions are accessed through a settings dropdown menu triggered by a gear button in the header.

The panel has four body sections stacked vertically:

1. **Bridge Instance** — status badge and listening address
2. **Action Buttons** — COPY JWT and RE-SYNC
3. **Connected User** — avatar, name, and ID of the currently authenticated user
4. **Real-Time Event Log** — scrollable list of timestamped bridge events with severity styling

The settings dropdown provides: Open Config, View Logs, Check for Updates, Restart Service, and Stop Service.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Monitor Bridge Status at a Glance (Priority: P1)

An operator clicks the system tray icon and the floating panel appears. The Bridge Instance section shows a green "ACTIVE" status badge with the listening address (e.g., "Running on 127.0.0.1:7070"). If the bridge is stopped, the badge shows an inactive state and the address reflects that the bridge is not running. The panel updates in real time — when a device disconnects, the status and event log reflect it immediately without manual refresh.

**Why this priority**: The primary purpose of the tray GUI is giving operators instant visibility into bridge health. Without real-time status, the panel is just decoration. Everything else depends on the operator knowing the bridge is running.

**Independent Test**: Launch the application, start the bridge, click the tray icon, and verify the panel shows "ACTIVE" with the correct listening address. Then stop the bridge and verify the status updates.

**Acceptance Scenarios**:

1. **Given** the tray application is running, **When** the user clicks the tray icon, **Then** the floating panel opens with the dark header showing "The Kinetic Vault" title and a gear settings button.
2. **Given** the bridge is running, **When** the panel is open, **Then** the Bridge Instance section displays a green status dot and "ACTIVE" label alongside the heading "Running on 127.0.0.1:7070" (or the configured address).
3. **Given** the bridge is stopped, **When** the panel is open, **Then** the Bridge Instance section displays an inactive status indicator and a heading indicating the bridge is not running.
4. **Given** the bridge is running, **When** the bridge encounters an error, **Then** the status badge updates to reflect an error state and an event appears in the event log describing the problem.
5. **Given** the panel is open, **When** the user clicks outside the panel, **Then** the panel closes (standard tray popup behavior).

---

### User Story 2 - Copy JWT Token (Priority: P1)

An operator needs the JWT token from a recent authenticated request to the bridge — for example, to paste into a testing tool or share with a developer. They click the tray icon to open the panel and press the "COPY JWT" button. The most recent JWT token received by the bridge is copied to the system clipboard and a brief visual confirmation appears on the button.

**Why this priority**: JWT token access is a frequent operational and debugging task. The bridge already validates tokens on every request; capturing the last received token for clipboard copy is a lightweight addition that eliminates manual extraction from browser dev tools or logs.

**Independent Test**: Open the panel, click "COPY JWT", and verify the clipboard contains a valid JWT token.

**Acceptance Scenarios**:

1. **Given** the bridge is running and a token is available, **When** the user clicks "COPY JWT", **Then** the current JWT token is copied to the system clipboard.
2. **Given** the bridge is not running, **When** the panel is open, **Then** the "COPY JWT" button is disabled or indicates that no token is available.
3. **Given** the user clicks "COPY JWT", **When** the copy succeeds, **Then** the button provides brief visual feedback (e.g., text change or color flash) confirming the copy.

---

### User Story 3 - Re-Sync Devices (Priority: P1)

An operator connects a new fingerprint reader or restarts an existing one. They click the tray icon to open the panel and press the "RE-SYNC" button. The bridge re-initializes its device connections — connecting to all configured devices, updating the device registry, and refreshing the event stream. The event log shows sync-related events as they happen.

**Why this priority**: Device reconnection is a common operational need. Rather than restarting the entire bridge, re-sync provides a targeted action to re-establish device connections, which is faster and less disruptive.

**Independent Test**: Disconnect a reader, click "RE-SYNC", reconnect the reader, and verify the device appears in the event log as connected.

**Acceptance Scenarios**:

1. **Given** the bridge is running, **When** the user clicks "RE-SYNC", **Then** the bridge re-initializes connections to all configured devices.
2. **Given** a re-sync is in progress, **When** device connection attempts complete (success or failure), **Then** the results appear as events in the Real-Time Event Log.
3. **Given** the bridge is not running, **When** the panel is open, **Then** the "RE-SYNC" button is disabled.
4. **Given** a re-sync is in progress, **When** the user clicks "RE-SYNC" again, **Then** the second request is ignored or the button shows a loading state to prevent duplicate triggers.

---

### User Story 4 - View Connected User (Priority: P2)

An operator opens the panel and the Connected User section shows the identity of the user currently authenticated to the bridge — their avatar (or initials fallback), display name, and user ID/handle. This confirms which operator session is active and is especially useful in shared workstation environments.

**Why this priority**: User identity display confirms the active session but is not strictly required for bridge operation. It adds situational awareness in multi-operator settings.

**Independent Test**: Authenticate to the bridge, open the panel, and verify the Connected User section shows the correct user name and ID.

**Acceptance Scenarios**:

1. **Given** a user is authenticated to the bridge, **When** the panel is open, **Then** the Connected User section displays the user's avatar (or initials on a light blue-gray background), their name, and their ID/handle.
2. **Given** no authenticated request has been received by the bridge since it started, **When** the panel is open, **Then** the Connected User section shows a placeholder state (e.g., "No active session").
3. **Given** a new authenticated request is received by the bridge, **When** the panel is open, **Then** the Connected User section updates to reflect the new user's identity from the JWT claims.

---

### User Story 5 - Monitor Real-Time Events (Priority: P2)

An operator opens the panel and scrolls through the Real-Time Event Log. Each event shows a monospaced timestamp in blue, a bold event title, and a secondary description. Events are color-coded by severity: normal events in dark text, warning/pending events in orange, and system boot events in muted gray. New events appear at the top of the list as they occur. The operator can click the filter icon to narrow events by type or severity.

**Why this priority**: The event log transforms the tray panel from a simple status indicator into an operational dashboard. It provides immediate context for what the bridge is doing without requiring a separate log viewer or terminal.

**Independent Test**: Start the bridge, perform a scan, open the panel, and verify the scan event appears in the log with correct timestamp, title, and description.

**Acceptance Scenarios**:

1. **Given** the bridge is running, **When** an event occurs (scan, enrollment, connection change, error), **Then** it appears in the event log within 2 seconds with timestamp, title, and description.
2. **Given** the event log is displayed, **When** a warning-level event occurs, **Then** its title is styled in orange (#F97316) to distinguish it from normal events.
3. **Given** the event log is displayed, **When** the user clicks the filter icon, **Then** filter options appear allowing the user to narrow events by severity or type.
4. **Given** the event log has many entries, **When** the user scrolls, **Then** older events remain accessible and the list performs smoothly without lag.
5. **Given** the bridge is stopped, **When** the panel is open, **Then** the event log shows the last events from before the bridge stopped (from the in-memory buffer) with a visual indicator that the log is not live.

---

### User Story 6 - Access Settings and Service Controls (Priority: P1)

An operator clicks the gear button in the panel header and a dropdown menu appears with dark background (#1E2535). The menu provides: Open Config (opens the config file for editing), View Logs (opens a detailed log viewer), Check for Updates, Restart Service (restarts the bridge process), and Stop Service (stops the bridge, styled in red). The operator can restart or stop the bridge directly from this menu without opening a separate window.

**Why this priority**: Service lifecycle control (restart, stop) and config access are essential operator actions grouped under the settings menu. Without them, the panel is read-only and the operator must fall back to the CLI for management tasks.

**Independent Test**: Open the settings dropdown, click "Restart Service", and verify the bridge stops and starts again with the status updating accordingly.

**Acceptance Scenarios**:

1. **Given** the panel is open, **When** the user clicks the gear button in the header, **Then** the settings dropdown appears below the button with a dark background and the menu items: Open Config, View Logs, Check for Updates, Restart Service, Stop Service.
2. **Given** the settings dropdown is open, **When** the user clicks "Restart Service", **Then** the bridge stops gracefully and restarts, with the status badge and event log reflecting the restart sequence.
3. **Given** the settings dropdown is open, **When** the user clicks "Stop Service", **Then** the bridge stops gracefully and the status badge updates to inactive. The "Stop Service" item is styled with red text and icon to signal a destructive action.
4. **Given** the settings dropdown is open, **When** the user clicks "Open Config", **Then** the config.yaml file opens in the system's default text editor or a built-in config editor.
5. **Given** the settings dropdown is open, **When** the user clicks "View Logs", **Then** a log viewer window opens with detailed, filterable log output.
6. **Given** the settings dropdown is open, **When** the user clicks "Check for Updates", **Then** the application checks for a newer version and displays the result (up to date or update available).
7. **Given** the settings dropdown is open, **When** the user clicks outside the dropdown, **Then** the dropdown closes without triggering an action.

---

### User Story 7 - Launch and Minimize to System Tray (Priority: P1)

When the application starts, it creates a system tray icon and the bridge starts automatically. The application runs in the background — no terminal window or main application window is visible. The operator interacts exclusively through the tray icon (click to open the floating panel) and the settings dropdown.

**Why this priority**: A system tray application must actually live in the system tray. If it opens a visible window on launch, it defeats the purpose of being a background utility.

**Independent Test**: Launch the application and verify no visible window appears — only a tray icon. Click the icon and verify the floating panel opens.

**Acceptance Scenarios**:

1. **Given** the application is launched, **When** it starts, **Then** a tray icon appears in the system tray/notification area and the bridge starts automatically using the configured settings.
2. **Given** the application is running, **When** the user clicks the tray icon, **Then** the floating panel appears. Clicking again or clicking outside closes it.
3. **Given** the application is running, **When** the user right-clicks the tray icon, **Then** a minimal context menu provides Quit (and optionally Restart, Show Panel).
4. **Given** the user selects "Quit" from the tray context menu, **When** the bridge is running, **Then** the bridge stops gracefully and the application exits completely.

---

### User Story 8 - Open and Edit Configuration (Priority: P2)

An operator opens the settings dropdown and selects "Open Config". The bridge's config.yaml file opens for editing. After making changes and saving the file, the operator restarts the bridge from the settings dropdown to apply the new configuration.

**Why this priority**: Config editing is a necessary administrative task but can be handled by an external editor. The GUI's role is providing quick access to the file and indicating when a restart is needed.

**Independent Test**: Select "Open Config", modify the log level, save the file, restart the bridge, and verify the new log level takes effect.

**Acceptance Scenarios**:

1. **Given** the settings dropdown is open, **When** the user clicks "Open Config", **Then** the config.yaml file opens in the system's default YAML/text editor.
2. **Given** the config file has been modified externally, **When** the user restarts the bridge, **Then** the new configuration is loaded and the status reflects any changes (e.g., new listening address).
3. **Given** the config file is modified to invalid values, **When** the bridge restarts, **Then** the bridge fails to start and the panel shows an error state with a descriptive message.

---

### User Story 9 - View Detailed Logs (Priority: P3)

An operator opens the settings dropdown and selects "View Logs". A separate log viewer window opens showing structured, filterable log output from the bridge. This provides more detail than the compact event log in the main panel.

**Why this priority**: Detailed log viewing aids troubleshooting but the main panel's event log covers most day-to-day needs. This is a power-user feature.

**Independent Test**: Start the bridge, open View Logs, perform operations, and verify detailed log entries appear with filtering capability.

**Acceptance Scenarios**:

1. **Given** the settings dropdown is open, **When** the user clicks "View Logs", **Then** a log viewer window opens displaying structured log entries with timestamps, severity levels, and messages.
2. **Given** the log viewer is open, **When** the user applies a severity filter, **Then** only matching entries are displayed.
3. **Given** the log viewer is open and the bridge is running, **When** new log entries are produced, **Then** they appear in the viewer within 2 seconds.

---

### User Story 10 - Check for Updates (Priority: P3)

An operator opens the settings dropdown and selects "Check for Updates". The application queries for the latest available version and displays the result — either confirming the current version is up to date or indicating a newer version is available.

**Why this priority**: Update checking is a nice-to-have for operational hygiene but does not block any core bridge functionality.

**Independent Test**: Click "Check for Updates" and verify a result is displayed (version check succeeds or fails gracefully).

**Acceptance Scenarios**:

1. **Given** the settings dropdown is open, **When** the user clicks "Check for Updates", **Then** the application checks for a newer version and displays the result within 10 seconds.
2. **Given** the version check fails (no network, update server unreachable), **When** the check completes, **Then** a message indicates the check could not be completed and suggests trying later.

---

### Edge Cases

- If the config file is deleted or moved while the tray application is running, the application detects the issue on the next config access or bridge restart and displays an error in the panel.
- If the bridge encounters an unrecoverable error while running in-process (e.g., driver crash, server panic), the tray panel updates to an error state, the event log records the failure, and the operator can use "Restart Service" from the settings dropdown.
- If the user clicks "RE-SYNC" while a scan or enrollment is in progress, the action is deferred or rejected to avoid disrupting active operations.
- If the user clicks "COPY JWT" but no token is available (bridge not running, token expired), the button provides feedback indicating no token is available.
- If the bridge fails to start on application launch (e.g., missing public key, unreachable devices), the panel shows an error state and the operator can fix the config via "Open Config" and retry via "Restart Service".
- If the application cannot find a required native library (driver SDK, Fyne dependencies), it displays a clear error at startup identifying the missing dependency.
- If multiple instances of the tray application are launched, the second instance detects the first and brings it to focus rather than running a duplicate.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display a system tray icon that launches the bridge automatically on application start and reflects the current bridge state (running, stopped, error) with distinct icon visuals.
- **FR-002**: System MUST show a 340px-wide floating panel when the tray icon is clicked, with a dark navy header (#0F1623) titled "The Kinetic Vault" containing a gear settings button, and a light gray body (#F4F5F8).
- **FR-003**: System MUST display a Bridge Instance section in the panel body showing a status badge (green dot + "ACTIVE" when running, inactive when stopped) and the listening address as a bold heading.
- **FR-004**: System MUST provide two equal-width action buttons — "COPY JWT" and "RE-SYNC" — in a horizontal row below the Bridge Instance section.
- **FR-005**: System MUST copy the most recent JWT token received by the bridge's HTTP server to the system clipboard when "COPY JWT" is clicked, providing visual confirmation on the button. If no authenticated request has been received, the button MUST be disabled with a "No token available" indicator.
- **FR-006**: System MUST re-initialize all device connections when "RE-SYNC" is clicked, with results appearing in the event log.
- **FR-007**: System MUST display a Connected User section showing the identity extracted from the most recent authenticated JWT received by the bridge — avatar (or initials on a light blue-gray #EAEDFA card), display name, and user ID/handle from the token's claims (sub, name). If no authenticated request has been received since the bridge started, a placeholder state (e.g., "No active session") MUST be shown.
- **FR-008**: System MUST display a Real-Time Event Log section showing timestamped events from an ephemeral in-memory ring buffer (minimum 100 events). Each event displays a monospaced blue timestamp, bold title, and secondary description, styled by severity (normal, warning/orange, muted/gray). The buffer is cleared when the bridge restarts.
- **FR-009**: System MUST provide a filter icon in the event log section header that allows filtering events by severity or type.
- **FR-010**: System MUST provide a settings dropdown triggered by the header gear button, with a dark background (#1E2535), containing: Open Config, View Logs, Check for Updates, Restart Service, and Stop Service.
- **FR-011**: System MUST open the config.yaml file in the system's default editor when "Open Config" is selected from the settings dropdown.
- **FR-012**: System MUST open a detailed log viewer window when "View Logs" is selected, displaying bridge logs from an in-memory ring buffer populated by a custom slog handler. The slog handler MUST also write logs to a file on disk for persistence. The log viewer MUST support severity filtering.
- **FR-013**: System MUST check for application updates when "Check for Updates" is selected and display the result.
 - **FR-013**: System MUST provide a "Check for Updates" action in the settings dropdown. For the MVP this action will display the current version or a message "Update checking not configured" (no network requests). A configurable update source and network-based check will be considered post‑MVP.
- **FR-014**: System MUST restart the bridge (graceful teardown of HTTP server, event broker, and driver, then re-initialization from config) when "Restart Service" is selected, with status and event log reflecting the restart sequence.
- **FR-015**: System MUST stop the bridge process gracefully when "Stop Service" is selected, following the same ordered teardown as the CLI (HTTP server stop, event broker stop, driver close). The "Stop Service" menu item MUST be styled with red text and icon.
- **FR-016**: System MUST detect unexpected bridge failure (server error, driver crash) and update the panel status to an error state with an event log entry and a "Restart Service" option.
- **FR-017**: System MUST disable the COPY JWT and RE-SYNC buttons when the bridge is not running.
- **FR-018**: System MUST exit completely (stopping the bridge if running) when "Quit" is selected from the tray icon context menu.
- **FR-019**: System MUST support Windows, macOS, and Linux with platform-appropriate system tray integration.
- **FR-020**: System MUST load configuration from the same config.yaml path used by the CLI, with the path configurable via the BRIDGE_CONFIG environment variable.
- **FR-021**: System MUST enforce single-instance behavior — if already running, a second launch brings the existing instance to focus.
- **FR-022**: System MUST apply the defined color tokens and spacing consistently: dark surface (#0F1623) for header and dropdown, light surface (#F4F5F8) for body, blue-primary (#1E3A8A) for headings and button text, blue-accent (#3B82F6) for timestamps, green-active (#22C55E) for active status, orange-warn (#F97316) for warning events, red-danger (#EF4444) for stop actions.

### Key Entities

- **Panel State**: The current state of the floating panel — open or closed, which section is in view, filter state for the event log. Managed entirely by the tray application.
- **Bridge Status**: The runtime state of the bridge process — running, stopped, starting, error. Includes the listening address, device count, and uptime. Displayed in the Bridge Instance section.
- **Connected User**: The identity of the most recent authenticated caller — avatar image (or initials), display name, and user ID/handle extracted from the JWT claims (sub, name) of the last authenticated HTTP request received by the bridge. No session state is added to the bridge; the tray observes the auth middleware to capture the latest token's claims. Displayed in the Connected User section. Empty if no authenticated request has been received since the bridge started.
- **Event Log Entry**: A single real-time event from the bridge — timestamp, title, description, and severity (normal, warning, info, muted). Stored in an ephemeral in-memory ring buffer (last 100 events) within the tray application, cleared when the bridge restarts. Displayed in the scrollable Real-Time Event Log section. Source is the bridge's event broker.
- **JWT Token**: The most recent JWT token received by the bridge's HTTP server from an authenticated request. Used by the COPY JWT action to place the token on the clipboard. The token was originally issued by the backend server; the tray does not generate tokens.
- **Settings Menu**: The dropdown menu state — open or closed. Contains service lifecycle actions (restart, stop) and utility actions (open config, view logs, check updates).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The floating panel opens within 1 second of clicking the tray icon, displaying current bridge status, user info, and recent events.
- **SC-002**: Bridge state changes (start, stop, error, device disconnect/reconnect) are reflected in the panel within 2 seconds.
- **SC-003**: Events appear in the Real-Time Event Log within 2 seconds of occurring on the bridge.
- **SC-004**: The COPY JWT button copies a valid token to the clipboard in under 1 second with visual confirmation.
- **SC-005**: The RE-SYNC action completes device reconnection within 15 seconds for reachable devices, with results visible in the event log.
- **SC-006**: The application launches, starts the bridge, and shows a tray icon within 10 seconds (assuming configured devices are reachable).
- **SC-007**: The bridge managed by the tray application exhibits identical behavior (device connection, HTTP serving, event streaming, graceful shutdown) to the CLI bridge.
- **SC-008**: The application runs reliably for at least 8 hours of continuous operation without memory leaks or UI degradation.
 - **SC-008**: Soak testing (8-hour reliability test) is deferred to post‑MVP. For the MVP the application will rely on bounded in‑memory buffers, unit tests, and code review to reduce the risk of memory leaks or UI degradation. A formal 8‑hour soak test will be added to the polish phase after MVP.
- **SC-009**: The settings dropdown provides access to all service lifecycle and utility actions within 2 clicks from the tray icon.
- **SC-010**: The panel renders correctly at 340px width with proper section spacing, color tokens, and typography as specified in the UI design.

## Assumptions

- The tray GUI application embeds the bridge as a Go library — the GUI binary imports bridge packages (config, driver, events, auth, device, api) directly and runs bridge logic in-process rather than launching a separate bridge process. The GUI does not reimplement bridge functionality; it calls existing bridge functions to start/stop the server, manage devices, and subscribe to events.
- The tray GUI is a separate Go binary (`cmd/tray/main.go`) within the same `biometric-bridge` Go module, sharing all internal packages with the CLI bridge.
- The user has a desktop environment supporting system tray icons (Windows taskbar, macOS menu bar, Linux desktop with notification area).
- The config.yaml format and validation rules are stable and defined by the existing bridge implementation. The GUI reuses the same config loading and validation logic.
- The JWT token for the COPY JWT function is the most recent JWT received by the bridge's HTTP server — originally issued by the backend server. The tray does not generate or sign tokens; it copies the last received token to the clipboard. No private key is required by the tray application.
- The "Connected User" reflects the identity extracted from the most recent authenticated JWT received by the bridge's HTTP server. No session state is added to the bridge — the tray observes the auth middleware to capture the latest token's claims (sub, name). If no authenticated request has been received, the section shows a placeholder.
- Auto-start (launch on login) is handled by the application installer or OS-level configuration, not through a toggle in the GUI settings.
- Update checking requires a defined update source (URL or mechanism) — the specific source will be determined during implementation.
 - Update checking: by default no remote update source is configured for the MVP; the UI shows "Update checking not configured". If an update source is later configured, the application will query it to determine available versions.
- The tray application replaces the bridge's default slog handler with a custom multi-writer handler that writes structured logs to both an in-memory ring buffer (for the View Logs UI) and a log file on disk (for persistence and troubleshooting). The log file path defaults to `bridge.log` in the same directory as config.yaml and is configurable via a new `log.file` field in config.yaml.
- Only one instance of the bridge tray application runs at a time. Duplicate launches are detected and redirected.
