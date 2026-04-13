# Data Model: Bridge System Tray GUI

**Feature**: Bridge System Tray GUI  
**Date**: 2026-04-12  
**Status**: Phase 1 Complete

---

## Entities

### TrayState

The top-level state container for the tray application.

| Field | Type | Description |
|-------|------|-------------|
| `bridgeStatus` | `BridgeStatus` | Current operational state of the embedded bridge |
| `panelOpen` | `bool` | Whether the floating panel is currently visible |
| `settingsOpen` | `bool` | Whether the settings dropdown is currently visible |
| `eventFilter` | `EventFilter` | Active filter for the event log (severity/type) |
| `configPath` | `string` | Path to the active config.yaml file |

### BridgeStatus

The runtime state of the embedded bridge.

| Field | Type | Description |
|-------|------|-------------|
| `state` | `BridgeState` | Enum: `stopped`, `starting`, `running`, `error` |
| `listenAddr` | `string` | Current HTTP listen address (e.g., "127.0.0.1:7070") |
| `deviceCount` | `int` | Number of configured devices |
| `connectedCount` | `int` | Number of currently connected devices |
| `errorMessage` | `string` | Error details if state is `error` |
| `uptime` | `time.Duration` | Time since bridge started (if running) |

**BridgeState enum values:**
- `stopped` — Bridge not running
- `starting` — Bridge initialization in progress
- `running` — Bridge fully operational
- `error` — Bridge stopped due to error

### ConnectedUser

Identity extracted from the most recent authenticated JWT.

| Field | Type | Description |
|-------|------|-------------|
| `userID` | `string` | Subject claim (`sub`) from JWT |
| `displayName` | `string` | Name claim (`name`) from JWT, fallback to userID |
| `avatarURL` | `string` | Optional avatar URL (empty for initials fallback) |
| `lastSeen` | `time.Time` | Timestamp of the authenticated request |
| `hasIdentity` | `bool` | True if any authenticated request has been received |

**Constraints:**
- All fields except `hasIdentity` are empty if no authenticated request received
- `displayName` defaults to `userID` if `name` claim not present in JWT

### EventEntry

A single real-time event from the bridge event broker.

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | `time.Time` | Event occurrence time |
| `title` | `string` | Short event title (e.g., "Scan Started") |
| `description` | `string` | Detailed description |
| `severity` | `EventSeverity` | Severity level affecting UI styling |
| `deviceName` | `string` | Source device name (optional) |

**EventSeverity enum values:**
- `normal` — Standard events (dark text)
- `warning` — Attention needed (orange text, #F97316)
- `info` — Informational (dark text)
- `muted` — System/internal events (gray text, #374151)

**Ring buffer constraints:**
- Max 100 entries
- FIFO eviction when full
- Cleared on bridge restart

### LogEntry

A single structured log entry from the bridge slog output.

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | `time.Time` | Log timestamp |
| `level` | `LogLevel` | Severity: error, warn, info, debug |
| `message` | `string` | Log message |
| `attrs` | `map[string]string` | Structured attributes (optional) |

**LogLevel enum values:**
- `error` — Error conditions
- `warn` — Warning conditions
- `info` — Informational messages
- `debug` — Debug detail

**Ring buffer constraints:**
- Max 1000 entries
- FIFO eviction when full
- Written to both memory buffer and log file

### JWTToken

The captured JWT from the most recent authenticated request.

| Field | Type | Description |
|-------|------|-------------|
| `tokenString` | `string` | Raw JWT string (signed) |
| `claims` | `jwt.Claims` | Parsed claims (sub, name, iss, aud, exp) |
| `receivedAt` | `time.Time` | When token was received |
| `isValid` | `bool` | Whether token is still valid (not expired) |

**Constraints:**
- Only stores the single most recent token
- Overwritten on each new authenticated request
- Expiry checked before copy to clipboard

### TrayConfig

Tray-specific configuration extending bridge config.

| Field | Type | Description |
|-------|------|-------------|
| `logFilePath` | `string` | Path to log file (default: `bridge.log` next to config) |
| `eventBufferSize` | `int` | Max events in ring buffer (default: 100) |
| `logBufferSize` | `int` | Max logs in ring buffer (default: 1000) |
| `version` | `string` | Current application version |
| `updateCheckURL` | `string` | Optional URL for version check (optional) |

---

## State Transitions

### BridgeStatus State Machine

```
          +-----------+
          |  stopped  |<------------------+
          +-----------+                   |
               |                          |
               | Start()                  |
               v                          |
          +-----------+     Error()       |
          | starting  |------------------>+
          +-----------+                   |
               |                          |
               | Ready()                  |
               v                          |
          +-----------+     Error()       |
          |  running  |------------------>+
          +-----------+                   |
               |                          |
               | Stop()                   |
               v                          |
          +-----------+                   |
          |  stopped  |-------------------+
          +-----------+        ^
               ^               |
               |               |
               +---------------+
               Restart()
```

**Transitions:**
- `stopped → starting`: User clicks tray icon (auto-start), or Restart Service
- `starting → running`: All devices connected, HTTP server listening
- `starting → error`: Device connection failure, config error, or panic
- `running → stopped`: User clicks Stop Service, or graceful shutdown
- `running → error`: Driver crash, panic, or unrecoverable error
- `error → stopped`: Error acknowledged or auto-cleanup
- Any state → starting: Restart Service invoked

### Panel State Transitions

```
Closed --(click tray icon)--> Open --(click outside)--> Closed
Closed --(click tray icon)--> Open --(click tray icon)--> Closed
Open --(click gear)--> SettingsOpen --(click outside)--> Open
Open --(click gear)--> SettingsOpen --(click gear)--> Open
```

---

## Relationships

```
TrayState
├── BridgeStatus (1:1) - Current bridge operational state
├── ConnectedUser (0..1) - Latest authenticated user (optional)
├── EventBuffer (1:1) - Ring buffer of EventEntry (max 100)
├── LogBuffer (1:1) - Ring buffer of LogEntry (max 1000)
└── JWTToken (0..1) - Latest captured token (optional)

BridgeStatus
└── references: internal/driver.Device (via registry)

EventEntry
└── references: internal/events.Event (from bridge broker)

LogEntry
└── produced by: internal/tray.MultiWriterHandler
```

---

## Validation Rules

### BridgeStatus Validation
- `listenAddr` must be valid host:port format
- `deviceCount` >= 0
- `connectedCount` >= 0 && <= `deviceCount`
- `uptime` only valid when state is `running`

### EventEntry Validation
- `title` non-empty, max 100 characters
- `severity` must be valid enum value
- `timestamp` not in future

### LogEntry Validation
- `message` non-empty
- `level` must be valid enum value

### JWTToken Validation
- `tokenString` must be valid JWT format (3 base64 parts)
- `isValid` checks `exp` claim against current time with clock skew

---

## Data Flow

### Event Capture Flow
```
[Bridge Driver] 
    ↓ (emits event)
[internal/events.Broker]
    ↓ (subscribers)
[internal/tray.EventSubscriber]
    ↓ (push)
[RingBuffer[EventEntry]]
    ↓ (UI refresh)
[Fyne widget.List] (display)
```

### Log Capture Flow
```
[Bridge Code]
    ↓ (slog call)
[internal/tray.MultiWriterHandler]
    ├──→ [RingBuffer[LogEntry]] (for UI)
    └──→ [os.File] (bridge.log for persistence)
```

### JWT Capture Flow
```
[HTTP Request]
    ↓
[auth.Middleware]
    ↓ (validation success)
[OnAuthenticated callback]
    ↓ (claims extraction)
[ConnectedUser + JWTToken storage]
    ↓ (UI refresh)
[Fyne UI components]
```

---

## Persistence

### In-Memory Only (Ephemeral)
- EventEntry ring buffer
- LogEntry ring buffer (UI portion)
- ConnectedUser (current session only)
- JWTToken (current token only)
- TrayState (runtime state)

### File System Persistence
- `config.yaml` — Bridge configuration (read-only by tray, edited externally)
- `bridge.log` — Structured log output (append-only, rotated by tray)

### No Database
The tray application requires no database. All state is either ephemeral (in-memory) or stored in the existing config.yaml and log files.
