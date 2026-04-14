package demo

import "time"

// MockDeviceConfig holds metadata for a single mock device.
type MockDeviceConfig struct {
	Name            string
	Model           string
	ID              string
	FirmwareVersion string
	FingerSupported bool
	ScanWidth       int
	ScanHeight      int
	SlapWidth       int
	SlapHeight      int
}

// DemoConfig holds all demo mode settings.
type DemoConfig struct {
	Devices       []MockDeviceConfig
	EventInterval time.Duration
	ScanDelay     time.Duration
	EnrollDelay   time.Duration
	QualityMin    int
	QualityMax    int
}

// DefaultDemoConfig returns a DemoConfig with sensible defaults:
// one "Demo Device", 5s event interval, 200ms scan delay, 800ms enroll delay,
// quality range 60–95.
func DefaultDemoConfig() DemoConfig {
	return DemoConfig{
		Devices: []MockDeviceConfig{
			{
				Name:            "Demo Device",
				Model:           "BioEntry W2",
				ID:              "demo-device-001",
				FirmwareVersion: "v2.6.0",
				FingerSupported: true,
				ScanWidth:       300,
				ScanHeight:      400,
				SlapWidth:       1600,
				SlapHeight:      1500,
			},
		},
		EventInterval: 5 * time.Second,
		ScanDelay:     200 * time.Millisecond,
		EnrollDelay:   800 * time.Millisecond,
		QualityMin:    60,
		QualityMax:    95,
	}
}
