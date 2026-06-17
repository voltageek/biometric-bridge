// Package tray provides the system tray GUI for the Biometric Bridge.
package tray

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// LogEntry represents a structured log record.
type LogEntry struct {
	Timestamp time.Time
	Level     slog.Level
	Message   string
	Attrs     map[string]string
}

// LogBuffer is a ring buffer for log entries with subscription support.
type LogBuffer struct {
	buffer *RingBuffer[LogEntry]
	subs   map[chan LogEntry]struct{}
	mu     sync.RWMutex
}

// NewLogBuffer creates a new log buffer with the given capacity.
func NewLogBuffer(capacity int) *LogBuffer {
	return &LogBuffer{
		buffer: NewRingBuffer[LogEntry](capacity),
		subs:   make(map[chan LogEntry]struct{}),
	}
}

// Push adds a log entry to the buffer and notifies subscribers.
func (b *LogBuffer) Push(entry LogEntry) {
	b.buffer.Push(entry)

	// Notify subscribers
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- entry:
		default:
			// Subscriber is slow, skip this entry
		}
	}
}

// GetAll returns all log entries in chronological order.
func (b *LogBuffer) GetAll() []LogEntry {
	return b.buffer.GetAll()
}

// GetFiltered returns log entries filtered by minimum level.
func (b *LogBuffer) GetFiltered(minLevel slog.Level) []LogEntry {
	all := b.buffer.GetAll()
	var filtered []LogEntry
	for _, entry := range all {
		if entry.Level >= minLevel {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// Subscribe returns a channel that receives new log entries in real-time.
// The caller must call Unsubscribe when done.
func (b *LogBuffer) Subscribe() <-chan LogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan LogEntry, 100)
	b.subs[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber channel.
func (b *LogBuffer) Unsubscribe(ch <-chan LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for sub := range b.subs {
		if sub == ch {
			close(sub)
			delete(b.subs, sub)
			return
		}
	}
}

// Clear removes all log entries from the buffer.
func (b *LogBuffer) Clear() {
	b.buffer.Clear()
}

// MultiWriterHandler implements slog.Handler that writes to both a ring buffer and a file.
type MultiWriterHandler struct {
	level  slog.Level
	buffer *LogBuffer
	file   *os.File
	mu     sync.Mutex
	attrs  []slog.Attr
	groups []string
}

// NewMultiWriterHandler creates a new multi-writer handler.
func NewMultiWriterHandler(level slog.Level, buffer *LogBuffer, file *os.File) *MultiWriterHandler {
	return &MultiWriterHandler{
		level:  level,
		buffer: buffer,
		file:   file,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *MultiWriterHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes a log record.
func (h *MultiWriterHandler) Handle(_ context.Context, r slog.Record) error {
	// Build the log entry
	entry := LogEntry{
		Timestamp: r.Time,
		Level:     r.Level,
		Message:   r.Message,
		Attrs:     make(map[string]string),
	}

	// Add accumulated attributes
	for _, attr := range h.attrs {
		entry.Attrs[attr.Key] = attr.Value.String()
	}

	// Add record attributes
	r.Attrs(func(a slog.Attr) bool {
		entry.Attrs[a.Key] = a.Value.String()
		return true
	})

	// Write to ring buffer (non-blocking)
	if h.buffer != nil {
		h.buffer.Push(entry)
	}

	// Write to file
	if h.file != nil {
		h.mu.Lock()
		defer h.mu.Unlock()

		// Format as JSON
		jsonEntry := map[string]interface{}{
			"time":  r.Time.Format(time.RFC3339),
			"level": r.Level.String(),
			"msg":   r.Message,
			"attrs": entry.Attrs,
		}

		data, err := json.Marshal(jsonEntry)
		if err != nil {
			return err
		}

		data = append(data, '\n')
		if _, err := h.file.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// WithAttrs returns a new Handler with the given attributes.
func (h *MultiWriterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &MultiWriterHandler{
		level:  h.level,
		buffer: h.buffer,
		file:   h.file,
		attrs:  append(h.attrs, attrs...),
		groups: h.groups,
	}
	return newHandler
}

// WithGroup returns a new Handler with the given group.
func (h *MultiWriterHandler) WithGroup(name string) slog.Handler {
	newHandler := &MultiWriterHandler{
		level:  h.level,
		buffer: h.buffer,
		file:   h.file,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
	return newHandler
}

// Close closes the file writer.
func (h *MultiWriterHandler) Close() error {
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}

// OpenLogFile opens a log file for writing, creating it if necessary.
func OpenLogFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

// FormatLogEntry formats a log entry as a human-readable string.
func FormatLogEntry(entry LogEntry) string {
	timeStr := entry.Timestamp.Format("15:04:05")
	levelStr := entry.Level.String()
	return fmt.Sprintf("[%s] %s: %s", timeStr, levelStr, entry.Message)
}
