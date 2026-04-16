package demo

import (
	"context"
	"encoding/base64"
	"sync"
	"time"

	"biometric-bridge/internal/driver"
)

// Driver implements driver.Driver with mock behavior for demo mode.
type Driver struct {
	cfg       DemoConfig
	eventCh   chan driver.Event
	closeCh   chan struct{}
	closeOnce sync.Once
	simulator *EventSimulator
	// attemptLock protects the attemptCounts map
	attemptLock   sync.Mutex
	attemptCounts map[string]int
}

// New creates a new demo Driver with the given config.
func New(cfg DemoConfig) (driver.Driver, error) {
	d := &Driver{
		cfg:     cfg,
		eventCh: make(chan driver.Event, 64),
		closeCh: make(chan struct{}),
	}
	d.simulator = NewEventSimulator(cfg, d.eventCh)
	return d, nil
}

// Connect is a no-op in demo mode. Always succeeds.
func (d *Driver) Connect(ctx context.Context, devices []driver.DeviceConfig) error {
	return nil
}

// Scan simulates a fingerprint scan. It validates the device name, applies a
// configurable delay, then returns a synthetic ScanResult with random template
// bytes and a quality score in [QualityMin, QualityMax].
func (d *Driver) Scan(ctx context.Context, deviceName string, finger driver.FingerPosition) (*driver.ScanResult, error) {
	if err := ValidateDevice(d.cfg, deviceName); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(d.cfg.ScanDelay):
	}

	dc := FindDevice(d.cfg, deviceName)
	templateB64, quality, width, height := GenerateScanResult(*dc, d.cfg.QualityMin, d.cfg.QualityMax)
	template, _ := base64.StdEncoding.DecodeString(templateB64)

	result := &driver.ScanResult{
		Template: template,
		Quality:  quality,
		Width:    width,
		Height:   height,
		Finger:   finger,
	}

	d.simulator.EmitAPIEvent(driver.Event{
		Type:       "scan",
		DeviceName: deviceName,
		UserID:     "demo-user",
		EventCode:  0,
	})

	return result, nil
}

// SlapScan simulates a multi-finger slap scan.
func (d *Driver) SlapScan(ctx context.Context, deviceName string, mode driver.CaptureMode) (*driver.SlapScanResult, error) {
	if err := ValidateDevice(d.cfg, deviceName); err != nil {
		return nil, err
	}

	dc := FindDevice(d.cfg, deviceName)
	if !dc.FingerSupported {
		return nil, driver.ErrSlapNotSupported
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(d.cfg.ScanDelay):
	}

	// Track attempt counts per-device so we can simulate improving quality
	// across successive SlapScan calls. Use a simple in-memory map stored on
	// the Driver. Initialize lazily.
	d.initAttemptMap()
	attempt := d.incrementAttempt(deviceName)

	slapImageB64, slapWidth, slapHeight, fingers := GenerateSlapResult(*dc, string(mode), d.cfg.QualityMin, d.cfg.QualityMax, attempt, d.cfg)
	slapImage, _ := base64.StdEncoding.DecodeString(slapImageB64)

	slapFingers := make([]driver.ScanResult, 0, len(fingers))
	for _, f := range fingers {
		imgB64, _ := f["image"].(string)
		img, _ := base64.StdEncoding.DecodeString(imgB64)
		slapFingers = append(slapFingers, driver.ScanResult{
			Template: img,
			Quality:  f["quality"].(int),
			Width:    f["width"].(int),
			Height:   f["height"].(int),
			Finger:   driver.FingerPosition(f["finger"].(string)),
		})
	}

	return &driver.SlapScanResult{
		SlapImage:  slapImage,
		SlapWidth:  slapWidth,
		SlapHeight: slapHeight,
		Fingers:    slapFingers,
	}, nil
}

// Enroll simulates a multi-impression enrollment.
func (d *Driver) Enroll(ctx context.Context, deviceName, userID, userName string, fingers []driver.FingerPosition) error {
	if err := ValidateDevice(d.cfg, deviceName); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d.cfg.EnrollDelay):
	}

	d.simulator.EmitAPIEvent(driver.Event{
		Type:       "enrollment_complete",
		DeviceName: deviceName,
		UserID:     userID,
	})

	return nil
}

// ListDevices returns DeviceInfo for each configured mock device.
func (d *Driver) ListDevices() []driver.DeviceInfo {
	infos := make([]driver.DeviceInfo, 0, len(d.cfg.Devices))
	for _, dc := range d.cfg.Devices {
		infos = append(infos, driver.DeviceInfo{
			Name:            dc.Name,
			ID:              dc.ID,
			Model:           dc.Model,
			FirmwareVersion: dc.FirmwareVersion,
			FingerSupported: dc.FingerSupported,
		})
	}
	return infos
}

// Subscribe returns the event channel. The event simulator starts emitting
// periodic events immediately. The channel is closed when Close is called.
func (d *Driver) Subscribe() <-chan driver.Event {
	d.simulator.Start()
	return d.eventCh
}

// Close stops the event simulator and closes the event channel.
func (d *Driver) Close() error {
	d.closeOnce.Do(func() {
		d.simulator.Stop()
		close(d.eventCh)
	})
	return nil
}

// initAttemptMap ensures the attemptCounts map is initialized.
func (d *Driver) initAttemptMap() {
	d.attemptLock.Lock()
	defer d.attemptLock.Unlock()
	if d.attemptCounts == nil {
		d.attemptCounts = make(map[string]int)
	}
}

// incrementAttempt increments and returns the attempt number for deviceName.
func (d *Driver) incrementAttempt(deviceName string) int {
	d.attemptLock.Lock()
	defer d.attemptLock.Unlock()
	c := d.attemptCounts[deviceName]
	c++
	d.attemptCounts[deviceName] = c
	return c
}
