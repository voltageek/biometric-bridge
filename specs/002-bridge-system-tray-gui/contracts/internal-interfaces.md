# Internal Contracts: Bridge System Tray GUI

**Feature**: Bridge System Tray GUI  
**Date**: 2026-04-12  
**Status**: Phase 1 Complete

---

## Overview

This document defines the internal interfaces between the tray GUI (`cmd/tray`) and the bridge packages (`internal/*`). These contracts ensure clean separation while allowing the tray to embed and control the bridge in-process.

---

## Contract 1: Tray ⇄ Bridge Lifecycle

### Interface: `BridgeController`

**Location**: `internal/tray/controller.go`

**Purpose**: Manages the embedded bridge's lifecycle (start, stop, restart) and exposes status for UI consumption.

```go
package tray

import (
    "context"
    "time"
)

// BridgeState represents the runtime state of the bridge
type BridgeState int

const (
    StateStopped BridgeState = iota
    StateStarting
    StateRunning
    StateError
)

// BridgeStatus provides runtime information about the bridge
type BridgeStatus struct {
    State           BridgeState
    ListenAddr      string
    DeviceCount     int
    ConnectedCount  int
    ErrorMessage    string
    Uptime          time.Duration
}

// BridgeController manages the embedded bridge lifecycle
type BridgeController interface {
    // Start initializes and starts the bridge
    // Returns when bridge is ready or fails to start
    Start(ctx context.Context) error
    
    // Stop gracefully shuts down the bridge
    // Waits for in-progress operations to complete
    Stop(ctx context.Context) error
    
    // Restart stops then starts the bridge
    Restart(ctx context.Context) error
    
    // ResyncDevices re-initializes device connections
    // Non-blocking; results appear as events
    ResyncDevices()
    
    // Status returns current bridge status
    Status() BridgeStatus
    
    // Subscribe returns a channel for status updates
    Subscribe() <-chan BridgeStatus
}
```

**Usage**:
```go
ctrl := tray.NewBridgeController(cfg)
if err := ctrl.Start(ctx); err != nil {
    log.Fatal(err)
}

go func() {
    for status := range ctrl.Subscribe() {
        // Update UI with new status
    }
}()
```

**Error Handling**:
- Start errors return descriptive errors with context (e.g., "device connection failed: reception at 192.168.0.110:51211")
- Stop errors indicate graceful shutdown failure; bridge may be in unknown state

---

## Contract 2: Auth Middleware Observer

### Interface: `AuthObserver`

**Location**: `internal/tray/observer.go`

**Purpose**: Captures JWT claims from authenticated requests for display in the tray UI.

```go
package tray

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

// Claims captures relevant JWT claims for display
type Claims struct {
    Subject     string    // sub claim
    Name        string    // name claim
    Issuer      string    // iss claim
    Audience    string    // aud claim
    ExpiresAt   time.Time // exp claim
    RawToken    string    // Full JWT string for COPY JWT feature
}

// AuthObserver receives claims from authenticated requests
type AuthObserver interface {
    OnAuthenticated(claims Claims)
}

// JWTStore holds the most recent JWT information
type JWTStore interface {
    SetClaims(claims Claims)
    GetClaims() (Claims, bool) // bool indicates if claims exist
    GetToken() (string, bool)  // Raw JWT for COPY JWT feature
    Clear()
}
```

**Integration Point**:

The `auth.Middleware` is extended to accept an optional observer:

```go
// In internal/auth/middleware.go
type MiddlewareConfig struct {
    Validator       *TokenValidator
    SkipPaths       map[string]bool
    OnAuthenticated func(claims jwt.Claims) // NEW: Optional callback
}
```

**Data Flow**:
```
[HTTP Request with JWT]
    ↓
[auth.Middleware validation]
    ↓ (success)
[OnAuthenticated callback]
    ↓
[tray.JWTStore.SetClaims()]
    ↓
[UI refresh with ConnectedUser]
```

**Constraints**:
- Observer only captures claims, never stores full request context
- Thread-safe: callback may be invoked from multiple goroutines
- No session state: each request overwrites previous claims

---

## Contract 3: Event Subscription

### Interface: `EventSubscriber`

**Location**: `internal/tray/event_subscriber.go`

**Purpose**: Receives real-time events from the bridge for display in the event log.

```go
package tray

import (
    "time"
)

// Severity level for UI styling
type Severity int

const (
    SeverityNormal Severity = iota
    SeverityWarning
    SeverityInfo
    SeverityMuted
)

// EventEntry is a display-friendly event
type EventEntry struct {
    Timestamp   time.Time
    Title       string
    Description string
    Severity    Severity
    DeviceName  string
}

// EventBuffer is a ring buffer for events
type EventBuffer interface {
    Push(event EventEntry)
    GetAll() []EventEntry      // Returns copy in chronological order
    GetRecent(n int) []EventEntry
    Subscribe() <-chan EventEntry // Real-time subscription
    Clear()
    Len() int
}
```

**Integration**:

```go
// Bridge controller subscribes to event broker
broker := events.NewBroker(eventCh)
eventBuffer := tray.NewEventBuffer(100)

// Subscribe bridge events
go func() {
    for event := range broker.Subscribe() {
        entry := tray.EventEntry{
            Timestamp:   event.Timestamp,
            Title:       event.Title,
            Description: event.Description,
            Severity:    mapSeverity(event.Type),
            DeviceName:  event.DeviceName,
        }
        eventBuffer.Push(entry)
    }
}()
```

**Mapping Bridge Events to Display Events**:

| Bridge Event Type | Display Title | Display Severity |
|-------------------|---------------|------------------|
| scan_started | "Scan Started" | SeverityNormal |
| scan_complete | "Scan Complete" | SeverityNormal |
| enrollment_complete | "User Authorized" | SeverityNormal |
| device_disconnected | "Device Disconnected" | SeverityWarning |
| device_reconnecting | "Handshake Pending" | SeverityWarning |
| device_connected | "Bridge Protocol Initialized" | SeverityInfo |
| bridge_started | "System Boot" | SeverityMuted |
| error | Event.Error | SeverityWarning |

---

## Contract 4: Multi-Writer Logger

### Interface: `MultiWriterHandler`

**Location**: `internal/tray/logger.go`

**Purpose**: Writes logs to both in-memory buffer (for UI) and file (for persistence).

```go
package tray

import (
    "context"
    "log/slog"
    "os"
)

// LogEntry represents a structured log record
type LogEntry struct {
    Timestamp time.Time
    Level     slog.Level
    Message   string
    Attrs     map[string]string
}

// LogBuffer is a ring buffer for log entries
type LogBuffer interface {
    Push(entry LogEntry)
    GetAll() []LogEntry
    GetFiltered(level slog.Level) []LogEntry
    Subscribe() <-chan LogEntry
    Clear()
}

// MultiWriterHandler implements slog.Handler
type MultiWriterHandler struct {
    level       slog.Level
    ringBuffer  LogBuffer
    file        *os.File
    formatter   func(LogEntry) string // JSON or text formatting
}

func (h *MultiWriterHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return level >= h.level
}

func (h *MultiWriterHandler) Handle(ctx context.Context, r slog.Record) error {
    entry := LogEntry{
        Timestamp: r.Time,
        Level:     r.Level,
        Message:   r.Message,
        Attrs:     extractAttrs(r),
    }
    
    // Write to ring buffer (non-blocking)
    h.ringBuffer.Push(entry)
    
    // Write to file
    h.file.WriteString(h.formatter(entry))
    
    return nil
}

func (h *MultiWriterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    // Return new handler with added attrs
}

func (h *MultiWriterHandler) WithGroup(name string) slog.Handler {
    // Return new handler with group
}
```

**Integration**:

```go
// In tray main, replace default slog handler
ringBuf := tray.NewLogBuffer(1000)
logFile, _ := os.OpenFile(cfg.Log.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

handler := tray.NewMultiWriterHandler(slog.LevelInfo, ringBuf, logFile)
slog.SetDefault(slog.New(handler))
```

**Log File Format**:

```json
{"time":"2026-04-12T10:30:00Z","level":"INFO","msg":"bridge started","addr":"127.0.0.1:7070"}
```

---

## Contract 5: Single Instance Lock

### Interface: `InstanceLocker`

**Location**: `internal/tray/singleinstance.go`

**Purpose**: Ensures only one tray application instance runs at a time.

```go
package tray

// InstanceLocker provides cross-platform single-instance enforcement
type InstanceLocker interface {
    // TryLock attempts to acquire the lock
    // Returns true if lock acquired, false if another instance is running
    TryLock() bool
    
    // Unlock releases the lock
    Unlock() error
    
    // BringToFront signals the existing instance to show its window
    // Only valid if TryLock() returned false
    BringToFront() error
}

// NewInstanceLocker creates a platform-appropriate locker
func NewInstanceLocker(appName string) InstanceLocker
```

**Platform Implementations**:

- **Windows**: Named mutex (`Global\KineticVaultMutex`)
- **macOS/Linux**: Unix socket on localhost (`127.0.0.1:17070`) or pidfile

**Socket Protocol** (for BringToFront):

```
Second instance connects to 127.0.0.1:17070
Sends: "SHOW\n"
First instance receives command and shows panel
```

---

## Contract 6: UI Component Styling

### Constants: `Theme`

**Location**: `internal/tray/theme.go`

**Purpose**: Defines the visual design tokens from the UI specification.

```go
package tray

import "image/color"

// Color tokens
var (
    SurfaceDark    = color.NRGBA{15, 22, 35, 255}      // #0F1623 (header, dropdown)
    SurfaceMenu    = color.NRGBA{30, 37, 53, 255}      // #1E2535 (dropdown background)
    SurfaceBody    = color.NRGBA{244, 245, 248, 255}   // #F4F5F8 (panel body)
    SurfaceCard    = color.NRGBA{255, 255, 255, 255}   // #FFFFFF (buttons, cards)
    SurfaceUser    = color.NRGBA{234, 237, 250, 255}   // #EAEDFA (user card)
    
    BluePrimary    = color.NRGBA{30, 58, 138, 255}     // #1E3A8A (headings, button text)
    BlueAccent     = color.NRGBA{59, 130, 246, 255}    // #3B82F6 (timestamps)
    GreenActive    = color.NRGBA{34, 197, 94, 255}     // #22C55E (active status)
    OrangeWarn     = color.NRGBA{249, 115, 22, 255}    // #F97316 (warning events)
    RedDanger      = color.NRGBA{239, 68, 68, 255}     // #EF4444 (stop actions)
    
    TextPrimary    = color.NRGBA{17, 24, 39, 255}      // #111827 (body text)
    TextMuted      = color.NRGBA{107, 114, 128, 255}   // #6B7280 (secondary text)
    TextLabel      = color.NRGBA{138, 143, 160, 255}   // #8A8FA0 (section labels)
    BorderDefault  = color.NRGBA{209, 213, 219, 255}   // #D1D5DB (borders)
)

// Layout constants
const (
    PanelWidth        = 340 // px
    HeaderHeight      = 52  // px
    SectionGap        = 16  // px
    ButtonHeight      = 44  // px
    ButtonGap         = 10  // px
    ListItemPadding   = 8   // px
    CardPadding       = 12  // px
    IconSize          = 16  // px
    AvatarSize        = 40  // px
    BorderRadius      = 8   // px
    BorderRadiusLarge = 12  // px (panel corners)
)

// Typography
const (
    FontSizeTitle      = 16 // Header title
    FontSizeHeading    = 20 // Bridge address
    FontSizeLabel      = 10 // Section labels
    FontSizeBody       = 13 // Event titles
    FontSizeSmall      = 12 // Descriptions
    FontSizeTimestamp  = 11 // Monospace timestamps
    FontSizeStatus     = 11 // Status badges
)
```

---

## Error Handling Contracts

### Error Classification

| Error Type | Example | User Impact | Recovery Action |
|------------|---------|-------------|-----------------|
| Config Error | Missing public_key_file | Bridge won't start | Open config, fix, restart |
| Device Error | Device unreachable | Bridge won't start | Check network, resync |
| Runtime Error | Driver panic | Bridge stops | Auto-detect, show restart option |
| Network Error | Update check fails | Warning only | Retry later |

### Error Display

Errors are shown in three places:
1. **Status badge**: Changes to error state with brief message
2. **Event log**: Detailed error entry with severity=warning
3. **Log file**: Full error details with stack trace (if available)

---

## Testing Contracts

### Unit Test Requirements

Each interface implementation MUST have tests for:
- Happy path (success scenarios)
- Error handling (failure modes)
- Concurrency (thread safety)
- Resource cleanup (no leaks)

### Integration Test Requirements

- Full startup/shutdown cycle
- Event flow: bridge → tray → UI
- JWT capture: request → auth middleware → UI update
- Resync: button click → device reconnection → events

---

## Version Compatibility

This contract is designed for:
- **Bridge**: v1.0.0+ (existing codebase)
- **Tray**: v1.0.0 (this feature)
- **Go**: 1.21+
- **Fyne**: v2.4+
