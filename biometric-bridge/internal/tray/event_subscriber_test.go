package tray

import "testing"

func TestMapBridgeEvent(t *testing.T) {
	tests := []struct {
		in       string
		wantType EventType
	}{
		{"scan_started", EventTypeScan},
		{"scan_complete", EventTypeScan},
		{"enrollment_complete", EventTypeEnrollment},
		{"device_disconnected", EventTypeConnection},
		{"device_connected", EventTypeConnection},
		{"bridge_started", EventTypeSystem},
		{"error", EventTypeError},
		{"unknown_event_xyz", EventTypeSystem},
	}

	for _, tt := range tests {
		_, _, et := MapBridgeEvent(tt.in)
		if et != tt.wantType {
			t.Fatalf("MapBridgeEvent(%q) = %v, want %v", tt.in, et, tt.wantType)
		}
	}
}
