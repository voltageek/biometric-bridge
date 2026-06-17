// Package tray provides the system tray GUI for the Biometric Bridge.
package tray

import (
	"sync"
	"time"
)

// Severity represents the severity level of an event for UI styling.
type Severity int

const (
	SeverityNormal Severity = iota
	SeverityWarning
	SeverityInfo
	SeverityMuted
)

// String returns the string representation of the severity.
func (s Severity) String() string {
	switch s {
	case SeverityNormal:
		return "normal"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	case SeverityMuted:
		return "muted"
	default:
		return "unknown"
	}
}

// EventType categorizes events for filtering in the UI.
type EventType int

const (
	EventTypeScan EventType = iota
	EventTypeEnrollment
	EventTypeConnection
	EventTypeSystem
	EventTypeError
)

func (t EventType) String() string {
	switch t {
	case EventTypeScan:
		return "scan"
	case EventTypeEnrollment:
		return "enrollment"
	case EventTypeConnection:
		return "connection"
	case EventTypeSystem:
		return "system"
	case EventTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// EventEntry is a display-friendly event from the bridge.
type EventEntry struct {
	Timestamp   time.Time
	Title       string
	Description string
	Severity    Severity
	DeviceName  string
	Type        EventType
}

// EventBuffer is a ring buffer for events with subscription support.
type EventBuffer struct {
	buffer *RingBuffer[EventEntry]
	subs   map[chan EventEntry]struct{}
	mu     sync.RWMutex
}

// NewEventBuffer creates a new event buffer with the given capacity.
func NewEventBuffer(capacity int) *EventBuffer {
	return &EventBuffer{
		buffer: NewRingBuffer[EventEntry](capacity),
		subs:   make(map[chan EventEntry]struct{}),
	}
}

// Push adds an event to the buffer and notifies subscribers.
func (b *EventBuffer) Push(event EventEntry) {
	b.buffer.Push(event)

	// Notify subscribers
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- event:
		default:
			// Subscriber is slow, skip this event
		}
	}
}

// GetFiltered returns events matching the provided type and/or severity.
// Use zero values to indicate "no filter" for that field.
func (b *EventBuffer) GetFiltered(t EventType, s Severity) []EventEntry {
	all := b.buffer.GetAll()
	var out []EventEntry
	for _, e := range all {
		if t != EventType(-1) { // sentinel: -1 means no type filter
			if e.Type != t {
				continue
			}
		}
		if s != Severity(-1) { // sentinel: -1 means no severity filter
			if e.Severity != s {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

// GetAll returns all events in chronological order.
func (b *EventBuffer) GetAll() []EventEntry {
	return b.buffer.GetAll()
}

// GetRecent returns the last n events in chronological order.
func (b *EventBuffer) GetRecent(n int) []EventEntry {
	return b.buffer.GetRecent(n)
}

// Subscribe returns a channel that receives new events in real-time.
// The caller must call Unsubscribe when done.
func (b *EventBuffer) Subscribe() <-chan EventEntry {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan EventEntry, 10)
	b.subs[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber channel.
func (b *EventBuffer) Unsubscribe(ch <-chan EventEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Convert <-chan to chan for map lookup
	for sub := range b.subs {
		if sub == ch {
			close(sub)
			delete(b.subs, sub)
			return
		}
	}
}

// Clear removes all events from the buffer.
func (b *EventBuffer) Clear() {
	b.buffer.Clear()
}

// Len returns the number of events in the buffer.
func (b *EventBuffer) Len() int {
	return b.buffer.Len()
}

// MapBridgeEventType maps bridge event types to display events.
func MapBridgeEventType(eventType string) (string, Severity) {
	switch eventType {
	case "scan_started":
		return "Scan Started", SeverityNormal
	case "scan_complete":
		return "Scan Complete", SeverityNormal
	case "enrollment_complete":
		return "User Authorized", SeverityNormal
	case "device_disconnected":
		return "Device Disconnected", SeverityWarning
	case "device_reconnecting":
		return "Handshake Pending", SeverityWarning
	case "device_connected":
		return "Bridge Protocol Initialized", SeverityInfo
	case "bridge_started":
		return "System Boot", SeverityMuted
	case "error":
		return "Error", SeverityWarning
	default:
		return eventType, SeverityNormal
	}
}

// MapBridgeEvent maps a raw bridge event type to a display title, severity and EventType.
func MapBridgeEvent(eventType string) (title string, sev Severity, et EventType) {
	title, sev = MapBridgeEventType(eventType)
	// map underlying string categories to EventType
	switch eventType {
	case "scan_started", "scan_complete":
		et = EventTypeScan
	case "enrollment_complete":
		et = EventTypeEnrollment
	case "device_disconnected", "device_reconnecting", "device_connected", "resync_started", "resync_complete":
		et = EventTypeConnection
	case "bridge_started", "bridge_stopped":
		et = EventTypeSystem
	case "error":
		et = EventTypeError
	default:
		// default mapping: treat unknown as system
		et = EventTypeSystem
	}
	return
}
