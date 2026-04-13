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

// EventEntry is a display-friendly event from the bridge.
type EventEntry struct {
	Timestamp   time.Time
	Title       string
	Description string
	Severity    Severity
	DeviceName  string
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
