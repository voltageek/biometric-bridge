// Package driver defines the Driver interface and shared types that abstract
// over different Suprema SDK backends (BS2, G-SDK). No SDK-specific imports
// appear outside of the driver sub-packages (Constitution Principle II).
package driver

import "context"

// FingerPosition identifies which finger(s) should be scanned. When passed to
// Scan or Enroll, drivers that support LED indicators will light up the
// corresponding mode and finger LEDs on the device before capture.
type FingerPosition string

const (
	FingerNone        FingerPosition = "" // No LED guidance
	FingerLeftLittle  FingerPosition = "left_little"
	FingerLeftRing    FingerPosition = "left_ring"
	FingerLeftMiddle  FingerPosition = "left_middle"
	FingerLeftIndex   FingerPosition = "left_index"
	FingerLeftThumb   FingerPosition = "left_thumb"
	FingerRightThumb  FingerPosition = "right_thumb"
	FingerRightIndex  FingerPosition = "right_index"
	FingerRightMiddle FingerPosition = "right_middle"
	FingerRightRing   FingerPosition = "right_ring"
	FingerRightLittle FingerPosition = "right_little"
)

// ValidFingerPositions is the set of recognized finger position strings.
var ValidFingerPositions = map[FingerPosition]bool{
	FingerLeftLittle:  true,
	FingerLeftRing:    true,
	FingerLeftMiddle:  true,
	FingerLeftIndex:   true,
	FingerLeftThumb:   true,
	FingerRightThumb:  true,
	FingerRightIndex:  true,
	FingerRightMiddle: true,
	FingerRightRing:   true,
	FingerRightLittle: true,
}

// DeviceConfig holds the configuration for connecting to a single device.
// This mirrors config.DeviceConfig but lives in the driver package to avoid
// a circular dependency.
type DeviceConfig struct {
	Name   string
	Addr   string
	Port   int
	UseSSL bool
}

// DeviceInfo holds runtime metadata about a connected device, as returned by
// GET /api/devices.
type DeviceInfo struct {
	Name            string // Human-readable name from config
	ID              string // Opaque SDK-assigned identifier
	Model           string // Hardware model (e.g., "BioStation 2")
	FirmwareVersion string // Firmware version string
	FingerSupported bool   // Whether the device has a fingerprint sensor
}

// ScanResult holds the output of a single fingerprint scan.
type ScanResult struct {
	Template []byte // Raw fingerprint image/template bytes
	Quality  int    // 0–100 quality score from the SDK
	Width    int    // Image width in pixels (0 if not applicable)
	Height   int    // Image height in pixels (0 if not applicable)
}

// DeviceState represents the operational state of a device.
type DeviceState int

const (
	DeviceIdle         DeviceState = iota // Ready for operations
	DeviceBusy                            // Scan or enroll in progress
	DeviceDisconnected                    // Reconnecting
)

func (s DeviceState) String() string {
	switch s {
	case DeviceIdle:
		return "idle"
	case DeviceBusy:
		return "busy"
	case DeviceDisconnected:
		return "disconnected"
	default:
		return "unknown"
	}
}

// Event is a real-time event emitted by a driver and fanned out to WebSocket
// subscribers.
type Event struct {
	Type        string `json:"type"`                  // "scan" | "error" | "reconnecting" | "connected"
	DeviceName  string `json:"deviceId"`              // Human-readable name from config
	UserID      string `json:"userId,omitempty"`      // Present for "scan" events
	EventCode   uint32 `json:"eventCode,omitempty"`   // Present for "scan" events
	Attempt     int    `json:"attempt,omitempty"`     // Present for "reconnecting" events
	WaitSeconds int    `json:"waitSeconds,omitempty"` // Present for "reconnecting" events
	Message     string `json:"message,omitempty"`     // Present for "error" events
}

// DeviceStateUpdater is implemented by the device registry to allow drivers
// to update device state on disconnect/reconnect without a circular dependency.
type DeviceStateUpdater interface {
	SetState(name string, state DeviceState)
}

// Driver is the interface that SDK-specific backends must implement.
// Only one driver is active per build (selected via build tags).
type Driver interface {
	// Connect establishes connections to all configured devices.
	// Returns an error if any device is unreachable (fail-fast, FR-011).
	Connect(ctx context.Context, devices []DeviceConfig) error

	// Scan captures a single fingerprint from the specified device.
	// If finger is not empty, the driver should light the corresponding LED
	// indicators before capture and clear them afterward.
	// The context should carry a 10-second timeout.
	Scan(ctx context.Context, deviceName string, finger FingerPosition) (*ScanResult, error)

	// Enroll performs a multi-impression enrollment on the specified device.
	// The fingers slice indicates which finger LED to light for each
	// impression (e.g., ["right_index", "right_index"] for two takes of the
	// same finger). Nil or empty means no LED guidance.
	// The context should carry a 10-second timeout per impression.
	Enroll(ctx context.Context, deviceName, userID, userName string, fingers []FingerPosition) error

	// ListDevices returns metadata for all connected devices.
	ListDevices() []DeviceInfo

	// Subscribe returns a channel that receives real-time events from all
	// connected devices. The channel is closed when the driver shuts down.
	Subscribe() <-chan Event

	// Close disconnects all devices and releases all resources.
	Close() error
}
