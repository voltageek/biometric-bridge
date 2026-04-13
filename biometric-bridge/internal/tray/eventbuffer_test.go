package tray

import (
	"testing"
	"time"
)

func TestEventBufferFiltering(t *testing.T) {
	buf := NewEventBuffer(10)

	now := time.Now()
	buf.Push(EventEntry{Timestamp: now, Title: "Scan1", Description: "ok", Severity: SeverityNormal, Type: EventTypeScan})
	buf.Push(EventEntry{Timestamp: now.Add(time.Second), Title: "Enroll1", Description: "ok", Severity: SeverityNormal, Type: EventTypeEnrollment})
	buf.Push(EventEntry{Timestamp: now.Add(2 * time.Second), Title: "Conn1", Description: "warn", Severity: SeverityWarning, Type: EventTypeConnection})
	buf.Push(EventEntry{Timestamp: now.Add(3 * time.Second), Title: "Sys1", Description: "info", Severity: SeverityMuted, Type: EventTypeSystem})
	buf.Push(EventEntry{Timestamp: now.Add(4 * time.Second), Title: "Err1", Description: "err", Severity: SeverityWarning, Type: EventTypeError})

	// Type-only filter
	scans := buf.GetFiltered(EventTypeScan, Severity(-1))
	if len(scans) != 1 || scans[0].Title != "Scan1" {
		t.Fatalf("expected 1 scan, got %v", scans)
	}

	// Severity-only filter
	warns := buf.GetFiltered(EventType(-1), SeverityWarning)
	if len(warns) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(warns))
	}

	// Combined filter (type + severity)
	errs := buf.GetFiltered(EventTypeError, SeverityWarning)
	if len(errs) != 1 || errs[0].Title != "Err1" {
		t.Fatalf("expected Err1, got %v", errs)
	}
}
