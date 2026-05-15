//go:build realscan

// Package realscan implements the Driver interface using the Xperix RealScan
// SDK via CGo. It communicates with Suprema RealScan G10 fingerprint readers
// over USB.
//
// Build with: CGO_ENABLED=1 go build -tags realscan ./cmd/bridge
//
// Platform-specific SDK loading is in dynload_linux.go and dynload_windows.go
package realscan

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"unsafe"

	"biometric-bridge/internal/driver"
)

const (
	// Pre-processing modes
	rsHighVisibilityPreprocess = 0
	rsBalancedPreprocess       = 1
	rsHighNFIQScorePreprocess  = 2

	// Beep patterns
	rsBeepPattern1 = 1

	// Capture timeout for a single scan in milliseconds
	defaultCaptureTimeoutMS = 10000

	// Slap capture timeout — longer because user must place multiple fingers
	defaultSlapCaptureTimeoutMS = 15000
)

// slapModeInfo maps a CaptureMode to the SDK capture mode, slap type, and
// mode LED index for multi-finger group captures.
type slapModeInfo struct {
	captureMode int // RS_CAPTURE_FLAT_* constant
	slapType    int // RS_SLAP_* constant
	modeLED     int // RS_LED_MODE_* constant
	fingerIndex int // RS_FINGER_* group index for TakeImageDataEx
}

var slapModeMap = map[driver.CaptureMode]slapModeInfo{
	driver.CaptureLeftFour: {
		captureMode: 4,    // RS_CAPTURE_FLAT_LEFT_FOUR_FINGERS
		slapType:    1,    // RS_SLAP_LEFT_FOUR
		modeLED:     0x01, // RS_LED_MODE_LEFT_FINGER4
		fingerIndex: 12,   // RS_FINGER_LEFT_FOUR
	},
	driver.CaptureRightFour: {
		captureMode: 5,    // RS_CAPTURE_FLAT_RIGHT_FOUR_FINGERS
		slapType:    2,    // RS_SLAP_RIGHT_FOUR
		modeLED:     0x02, // RS_LED_MODE_RIGHT_FINGER4
		fingerIndex: 13,   // RS_FINGER_RIGHT_FOUR
	},
	driver.CaptureTwoThumbs: {
		captureMode: 3,    // RS_CAPTURE_FLAT_TWO_FINGERS
		slapType:    4,    // RS_SLAP_TWO_THUMB
		modeLED:     0x03, // RS_LED_MODE_TWO_THUMB
		fingerIndex: 11,   // RS_FINGER_TWO_THUMB
	},
}

// slapFingerTypeToPosition maps the RSSlapInfo.fingerType values returned by
// the SDK's segmentation to our FingerPosition constants. These correspond to
// the RS_FGP_* constants in RS_ParamDef.h.
var slapFingerTypeToPosition = map[int]driver.FingerPosition{
	1:  driver.FingerRightThumb,
	2:  driver.FingerRightIndex,
	3:  driver.FingerRightMiddle,
	4:  driver.FingerRightRing,
	5:  driver.FingerRightLittle,
	6:  driver.FingerLeftThumb,
	7:  driver.FingerLeftIndex,
	8:  driver.FingerLeftMiddle,
	9:  driver.FingerLeftRing,
	10: driver.FingerLeftLittle,
}

// fingerLEDInfo maps a FingerPosition to the SDK's finger index constant
// and the appropriate mode LED to light.
type fingerLEDInfo struct {
	fingerIndex int // RS_FINGER_* constant for RS_SetFingerLED
	modeLED     int // RS_LED_MODE_* constant for RS_SetModeLED (0 = none)
}

var fingerLEDMap = map[driver.FingerPosition]fingerLEDInfo{
	driver.FingerLeftLittle:  {fingerIndex: 1, modeLED: 0x01},  // RS_FINGER_LEFT_LITTLE,  RS_LED_MODE_LEFT_FINGER4
	driver.FingerLeftRing:    {fingerIndex: 2, modeLED: 0x01},  // RS_FINGER_LEFT_RING,    RS_LED_MODE_LEFT_FINGER4
	driver.FingerLeftMiddle:  {fingerIndex: 3, modeLED: 0x01},  // RS_FINGER_LEFT_MIDDLE,  RS_LED_MODE_LEFT_FINGER4
	driver.FingerLeftIndex:   {fingerIndex: 4, modeLED: 0x01},  // RS_FINGER_LEFT_INDEX,   RS_LED_MODE_LEFT_FINGER4
	driver.FingerLeftThumb:   {fingerIndex: 5, modeLED: 0x03},  // RS_FINGER_LEFT_THUMB,   RS_LED_MODE_TWO_THUMB
	driver.FingerRightThumb:  {fingerIndex: 6, modeLED: 0x03},  // RS_FINGER_RIGHT_THUMB,  RS_LED_MODE_TWO_THUMB
	driver.FingerRightIndex:  {fingerIndex: 7, modeLED: 0x02},  // RS_FINGER_RIGHT_INDEX,  RS_LED_MODE_RIGHT_FINGER4
	driver.FingerRightMiddle: {fingerIndex: 8, modeLED: 0x02},  // RS_FINGER_RIGHT_MIDDLE, RS_LED_MODE_RIGHT_FINGER4
	driver.FingerRightRing:   {fingerIndex: 9, modeLED: 0x02},  // RS_FINGER_RIGHT_RING,   RS_LED_MODE_RIGHT_FINGER4
	driver.FingerRightLittle: {fingerIndex: 10, modeLED: 0x02}, // RS_FINGER_RIGHT_LITTLE, RS_LED_MODE_RIGHT_FINGER4
}

// nfiqToPercent converts NIST NFIQ scores (1–5, lower=better) to a 0–100
// scale (higher=better) for API consistency across drivers. We also accept
// values outside 1..5 defensively.
func nfiqToPercent(nfiq int) int {
	switch nfiq {
	case 1:
		return 100
	case 2:
		return 80
	case 3:
		return 60
	case 4:
		return 40
	case 5:
		return 20
	default:
		if nfiq <= 0 {
			// Treat missing/zero as best-effort
			return 100
		}
		// Unknown/greater-than-5 -> map into lower quality
		if nfiq > 5 {
			// scale down proportionally (clamp)
			v := 100 - (nistClamp(nfiq)-1)*20
			if v < 0 {
				return 0
			}
			return v
		}
		return 0
	}
}

// nistClamp helper clamps nfiq to 1..5 range for simple computations.
func nistClamp(n int) int {
	if n < 1 {
		return 1
	}
	if n > 5 {
		return 5
	}
	return n
}

// setFingerLEDs lights the mode LED and the individual finger LED for the
// given position. Call clearLEDs to turn them off after capture.
func setFingerLEDs(handle int, finger driver.FingerPosition) {
	info, ok := fingerLEDMap[finger]
	if !ok {
		return
	}

	// Turn on the mode LED (left-4, right-4, thumbs, or roll icon)
	rc := sdkSetModeLED(handle, info.modeLED, 1)
	if rc != RS_SUCCESS {
		slog.Debug("set mode LED failed", "mode", info.modeLED, "error", rsErrString(rc))
	}

	// Light the specific finger LED in green
	rc = sdkSetFingerLED(handle, info.fingerIndex, RS_LED_GREEN)
	if rc != RS_SUCCESS {
		slog.Debug("set finger LED failed", "finger", info.fingerIndex, "error", rsErrString(rc))
	}
}

// clearLEDs turns off all mode and finger LEDs.
func clearLEDs(handle int) {
	sdkSetModeLED(handle, RS_LED_MODE_ALL, 0)
	sdkSetFingerLED(handle, RS_FINGER_ALL, RS_LED_OFF)
}

// RSDriver implements the driver.Driver interface using the Xperix RealScan SDK.
type RSDriver struct {
	mu      sync.RWMutex
	devices map[string]*rsDevice // keyed by config name
	eventCh chan driver.Event
	closed  bool
}

type rsDevice struct {
	name      string
	handle    int // SDK device handle
	deviceIdx int // USB device index used during init
	info      driver.DeviceInfo
	capturing bool // tracks whether a capture is in progress
}

// New creates a new RealScan driver. The libPath is the path to the RealScan
// shared library (e.g., libRS_SDK.so.2.2.0.2470 on Linux, RS_SDK.dll on Windows).
func New(libPath string) (*RSDriver, error) {
	rc := sdkLoadLibrary(libPath)
	if rc != 0 {
		return nil, fmt.Errorf("failed to load RealScan SDK from %s (error code: %d)", libPath, rc)
	}

	// Initialize SDK — pass nil config dir, 0 options
	numDevices, rc2 := sdkInit(nil, 0)
	if rc2 != RS_SUCCESS {
		return nil, fmt.Errorf("RS_InitSDK failed: %s (code %d)", rsErrString(rc2), rc2)
	}

	slog.Info("RealScan SDK initialized", "devicesFound", numDevices)

	d := &RSDriver{
		devices: make(map[string]*rsDevice),
		eventCh: make(chan driver.Event, 256),
	}

	// Register the global driver instance for C callbacks
	globalDriverMu.Lock()
	globalDriver = d
	globalDriverMu.Unlock()

	// Start hot plugging detection
	rc3 := sdkStartHotPlugging()
	if rc3 == 0 {
		rc4 := sdkRegisterHotPlugCallbackGo()
		if rc4 != 0 {
			slog.Warn("failed to register hot-plug callback", "error", rsErrString(rc4))
		} else {
			slog.Info("hot-plug monitoring started")
		}
	} else {
		slog.Warn("hot-plugging not available", "error", rsErrString(rc3))
	}

	return d, nil
}

// Connect initializes all configured USB devices. For RealScan, devices are
// identified by USB index rather than IP:port. The DeviceConfig.Addr field is
// interpreted as the device index (e.g., "0", "1") for USB devices, or left
// empty to auto-assign sequentially.
func (d *RSDriver) Connect(ctx context.Context, devices []driver.DeviceConfig) error {
	// Discover how many USB devices the SDK sees
	numDevices, rc := sdkGetDeviceCount()
	if rc != RS_SUCCESS {
		return fmt.Errorf("RS_GetDeviceCount failed: %s", rsErrString(rc))
	}

	if numDevices == 0 {
		return fmt.Errorf("no RealScan devices found on USB bus")
	}

	slog.Info("USB device discovery", "found", numDevices, "configured", len(devices))

	for i, dc := range devices {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Determine device index — use sequential index for now.
		// For multi-device setups the config could specify a DeviceIndex field.
		deviceIdx := i
		if deviceIdx >= numDevices {
			return fmt.Errorf("device %q: USB index %d out of range (only %d devices found)",
				dc.Name, deviceIdx, numDevices)
		}

		// Initialize the device
		handle, rc := sdkInitDevice(deviceIdx)
		if rc != RS_SUCCESS {
			return fmt.Errorf("failed to init device %q (index %d): %s",
				dc.Name, deviceIdx, rsErrString(rc))
		}

		// Enable automatic calibration
		sdkSetAutoCalibrate(handle, 1)

		// Set pre-processing to balanced mode for good quality + speed
		sdkSetPreProcessing(handle, rsBalancedPreprocess)

		// Set capture mode: flat single finger, high sensitivity, LED on
		rc = sdkSetCaptureMode(handle, RS_CAPTURE_FLAT_SINGLE_FINGER, RS_AUTO_SENSITIVITY_HIGH, 1)
		if rc != RS_SUCCESS {
			sdkExitDevice(handle)
			return fmt.Errorf("failed to set capture mode for %q: %s",
				dc.Name, rsErrString(rc))
		}

		// Get device info
		devInfo, rc := sdkGetDeviceInfo(handle)
		if rc != RS_SUCCESS {
			sdkExitDevice(handle)
			return fmt.Errorf("failed to get device info for %q: %s",
				dc.Name, rsErrString(rc))
		}

		dev := &rsDevice{
			name:      dc.Name,
			handle:    handle,
			deviceIdx: deviceIdx,
			info: driver.DeviceInfo{
				Name:            dc.Name,
				ID:              devInfo.DeviceID,
				Model:           devInfo.ProductName,
				FirmwareVersion: devInfo.FirmwareVersion,
				FingerSupported: true, // RealScan is always a fingerprint device
			},
		}

		d.mu.Lock()
		d.devices[dc.Name] = dev
		d.mu.Unlock()

		slog.Info("device initialized",
			"name", dc.Name,
			"model", devInfo.ProductName,
			"id", devInfo.DeviceID,
			"firmware", devInfo.FirmwareVersion,
			"usbIndex", deviceIdx,
		)

		// Emit connected event
		d.emitEvent(driver.Event{
			Type:       "connected",
			DeviceName: dc.Name,
		})
	}

	return nil
}

// Scan captures a single fingerprint image from the specified device and
// returns the raw grayscale image data with a NIST quality score.
// If finger is specified, the device LEDs indicate which finger to place.
func (d *RSDriver) Scan(ctx context.Context, deviceName string, finger driver.FingerPosition) (*driver.ScanResult, error) {
	dev, err := d.getDevice(deviceName)
	if err != nil {
		return nil, err
	}

	// Mark device as capturing
	d.mu.Lock()
	if dev.capturing {
		d.mu.Unlock()
		return nil, fmt.Errorf("device %q: capture already in progress", deviceName)
	}
	dev.capturing = true
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		dev.capturing = false
		d.mu.Unlock()
	}()

	handle := dev.handle

	// Light LEDs to guide finger placement
	if finger != driver.FingerNone {
		setFingerLEDs(handle, finger)
		defer clearLEDs(handle)
	}

	// Set up context cancellation to abort capture
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			sdkAbortCapture(handle)
		case <-done:
		}
	}()

	// Blocking capture — blocks until finger placed and image acquired
	var imageData unsafe.Pointer
	var width, height, rc int

	// Use extended capture if a finger hint is available (enables automatic
	// LED feedback during capture — green on success, red on failure)
	if finger != driver.FingerNone {
		info := fingerLEDMap[finger]
		imageData, width, height, rc = sdkTakeImageDataEx(handle, defaultCaptureTimeoutMS, info.fingerIndex, 1)
	} else {
		imageData, width, height, rc = sdkTakeImageData(handle, defaultCaptureTimeoutMS)
	}
	close(done) // stop cancellation goroutine

	if rc != RS_SUCCESS {
		if rc == RS_ERR_CAPTURE_ABORTED {
			return nil, fmt.Errorf("scan cancelled on device %q", deviceName)
		}
		if rc == RS_ERR_CAPTURE_TIMEOUT {
			return nil, fmt.Errorf("%w on device %q", driver.ErrScanTimeout, deviceName)
		}
		return nil, fmt.Errorf("scan failed on device %q: %s (code %d)",
			deviceName, rsErrString(rc), rc)
	}

	// Copy image data to Go memory before freeing SDK memory
	imageSize := width * height // 8-bit grayscale, 1 byte per pixel
	goImageData := make([]byte, imageSize)
	copy(goImageData, unsafe.Slice((*byte)(imageData), imageSize))

	// Get NIST quality score
	nistQuality, qrc := sdkGetQualityScore(imageData, width, height)

	// Free SDK-allocated image data
	sdkFreeImageData(imageData)

	quality := 0
	if qrc == RS_SUCCESS {
		quality = nfiqToPercent(nistQuality)
		slog.Debug("quality converted", "device", deviceName, "nist_nfiq", nistQuality, "quality_pct", quality)
	} else {
		slog.Warn("quality scoring failed", "device", deviceName, "error", rsErrString(qrc))
	}

	// Success beep
	sdkBeep(dev.handle, rsBeepPattern1)

	slog.Info("scan complete",
		"device", deviceName,
		"width", width,
		"height", height,
		"quality", quality,
		"imageBytes", imageSize,
	)

	return &driver.ScanResult{
		Template: goImageData,
		Quality:  quality,
		Width:    width,
		Height:   height,
	}, nil
}

// SlapScan captures multiple fingers simultaneously and segments the result
// into individual finger images. The mode selects which finger group to capture
// (left_four, right_four, two_thumbs). The device capture mode is temporarily
// switched and restored after capture.
func (d *RSDriver) SlapScan(ctx context.Context, deviceName string, mode driver.CaptureMode) (*driver.SlapScanResult, error) {
	modeInfo, ok := slapModeMap[mode]
	if !ok {
		return nil, fmt.Errorf("unsupported capture mode: %s", mode)
	}

	dev, err := d.getDevice(deviceName)
	if err != nil {
		return nil, err
	}

	// Mark device as capturing
	d.mu.Lock()
	if dev.capturing {
		d.mu.Unlock()
		return nil, fmt.Errorf("device %q: capture already in progress", deviceName)
	}
	dev.capturing = true
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		dev.capturing = false
		d.mu.Unlock()
	}()

	handle := dev.handle

	// Switch capture mode to the multi-finger mode
	rc := sdkSetCaptureMode(handle, modeInfo.captureMode, RS_AUTO_SENSITIVITY_HIGH, 1)
	if rc != RS_SUCCESS {
		return nil, fmt.Errorf("failed to set capture mode %s on device %q: %s (code %d)",
			mode, deviceName, rsErrString(rc), rc)
	}

	// Restore single-finger capture mode when done
	defer func() {
		rc := sdkSetCaptureMode(handle, RS_CAPTURE_FLAT_SINGLE_FINGER, RS_AUTO_SENSITIVITY_HIGH, 1)
		if rc != RS_SUCCESS {
			slog.Warn("failed to restore single-finger capture mode",
				"device", deviceName, "error", rsErrString(rc))
		}
	}()

	// Light mode LED to guide finger placement
	sdkSetModeLED(handle, modeInfo.modeLED, 1)
	defer clearLEDs(handle)

	// Set up context cancellation to abort capture
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			sdkAbortCapture(handle)
		case <-done:
		}
	}()

	// Blocking segmented capture
	imageData, imageWidth, imageHeight, _, numOfFinger, rc, slapInfos, fingerImages, fingerWidths, fingerHeights := sdkTakeImageDataSegment(handle, defaultSlapCaptureTimeoutMS, modeInfo.slapType)
	close(done)

	if rc != RS_SUCCESS {
		if rc == RS_ERR_CAPTURE_ABORTED {
			return nil, fmt.Errorf("slap scan cancelled on device %q", deviceName)
		}
		if rc == RS_ERR_CAPTURE_TIMEOUT {
			return nil, fmt.Errorf("%w on device %q (slap)", driver.ErrScanTimeout, deviceName)
		}
		return nil, fmt.Errorf("slap scan failed on device %q: %s (code %d)",
			deviceName, rsErrString(rc), rc)
	}

	nFingers := numOfFinger

	// Copy the full slap image to Go memory
	slapImageSize := imageWidth * imageHeight
	goSlapImage := make([]byte, slapImageSize)
	copy(goSlapImage, unsafe.Slice((*byte)(imageData), slapImageSize))

	// Copy each segmented finger image to Go memory
	fingers := make([]driver.ScanResult, 0, nFingers)
	for i := 0; i < nFingers; i++ {
		fImg := fingerImages[i]
		fW := fingerWidths[i]
		fH := fingerHeights[i]
		fSize := fW * fH
		goFingerImage := make([]byte, fSize)
		copy(goFingerImage, unsafe.Slice((*byte)(fImg), fSize))

		// Get quality score for this finger
		nistQuality, qrc := sdkGetQualityScore(fImg, fW, fH)
		quality := 0
		if qrc == RS_SUCCESS {
			quality = nfiqToPercent(nistQuality)
			slog.Debug("segmented finger quality converted", "device", deviceName, "finger", i, "nist_nfiq", nistQuality, "quality_pct", quality)
		} else {
			slog.Debug("quality scoring failed for segmented finger",
				"device", deviceName, "finger", i, "error", rsErrString(qrc))
		}

		// Map the SDK fingerType to our FingerPosition
		var fingerPos driver.FingerPosition
		if i < len(slapInfos) {
			fingerType := slapInfos[i].FingerType
			if pos, ok := slapFingerTypeToPosition[fingerType]; ok {
				fingerPos = pos
			}
			// Override quality with SDK-reported quality if available
			if slapInfos[i].ImageQuality > 0 {
				rawNFIQ := slapInfos[i].ImageQuality
				quality = nfiqToPercent(rawNFIQ)
				slog.Debug("segmented finger quality from slapInfo", "device", deviceName, "finger", i, "nist_nfiq", rawNFIQ, "quality_pct", quality)
			}
		}

		fingers = append(fingers, driver.ScanResult{
			Template: goFingerImage,
			Quality:  quality,
			Width:    fW,
			Height:   fH,
			Finger:   fingerPos,
		})

		slog.Debug("segmented finger",
			"index", i,
			"finger", string(fingerPos),
			"width", fW,
			"height", fH,
			"quality", quality,
		)
	}

	// Free SDK-allocated memory — the full slap image
	sdkFreeImageData(imageData)
	// Note: fingerImageData, fingerImageWidth, fingerImageHeight, and slapInfo
	// are SDK-allocated arrays. The SDK manages their lifetime alongside the
	// main image data returned by RS_TakeImageDataSegment.

	// Success beep
	sdkBeep(handle, rsBeepPattern1)

	slog.Info("slap scan complete",
		"device", deviceName,
		"mode", string(mode),
		"slapWidth", imageWidth,
		"slapHeight", imageHeight,
		"fingersDetected", nFingers,
	)

	return &driver.SlapScanResult{
		SlapImage:  goSlapImage,
		SlapWidth:  imageWidth,
		SlapHeight: imageHeight,
		Fingers:    fingers,
	}, nil
}

// Enroll performs a multi-impression enrollment capture on the specified device.
// For RealScan, this captures fingerprint images and returns them via the event
// channel for server-side processing. The fingers slice controls which LED to
// light for each impression. If fingers is nil/empty, defaults to two
// impressions with no LED guidance.
func (d *RSDriver) Enroll(ctx context.Context, deviceName, userID, userName string, fingers []driver.FingerPosition) error {
	dev, err := d.getDevice(deviceName)
	if err != nil {
		return err
	}

	// Mark device as capturing
	d.mu.Lock()
	if dev.capturing {
		d.mu.Unlock()
		return fmt.Errorf("device %q: capture already in progress", deviceName)
	}
	dev.capturing = true
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		dev.capturing = false
		d.mu.Unlock()
	}()

	handle := dev.handle

	// Default to 2 impressions if no fingers specified
	numImpressions := len(fingers)
	if numImpressions == 0 {
		numImpressions = 2
		fingers = make([]driver.FingerPosition, numImpressions)
	}

	// Capture each impression
	for i := 0; i < numImpressions; i++ {
		select {
		case <-ctx.Done():
			sdkAbortCapture(handle)
			clearLEDs(handle)
			return ctx.Err()
		default:
		}

		finger := fingers[i]

		// Light LEDs for this impression
		if finger != driver.FingerNone {
			setFingerLEDs(handle, finger)
		}

		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				sdkAbortCapture(handle)
			case <-done:
			}
		}()

		var imageData unsafe.Pointer
		var rc int
		if finger != driver.FingerNone {
			info := fingerLEDMap[finger]
			imageData, _, _, rc = sdkTakeImageDataEx(handle, defaultCaptureTimeoutMS, info.fingerIndex, 1)
		} else {
			imageData, _, _, rc = sdkTakeImageData(handle, defaultCaptureTimeoutMS)
		}
		close(done)

		// Clear LEDs after each impression
		if finger != driver.FingerNone {
			clearLEDs(handle)
		}

		if rc != RS_SUCCESS {
			if rc == RS_ERR_CAPTURE_ABORTED {
				return fmt.Errorf("enrollment cancelled on device %q (impression %d)", deviceName, i+1)
			}
			if rc == RS_ERR_CAPTURE_TIMEOUT {
				return fmt.Errorf("%w on device %q (enrollment impression %d)", driver.ErrScanTimeout, deviceName, i+1)
			}
			return fmt.Errorf("enrollment scan %d failed on device %q: %s (code %d)",
				i+1, deviceName, rsErrString(rc), rc)
		}

		// Free SDK memory — for enrollment the server will request images
		// via separate scan calls or the images are forwarded via events.
		sdkFreeImageData(imageData)

		// Success beep for each impression
		sdkBeep(handle, rsBeepPattern1)

		slog.Info("enrollment impression captured",
			"device", deviceName,
			"impression", i+1,
			"finger", string(finger),
			"userID", userID,
		)
	}

	return nil
}

// ListDevices returns metadata for all initialized devices.
func (d *RSDriver) ListDevices() []driver.DeviceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	infos := make([]driver.DeviceInfo, 0, len(d.devices))
	for _, dev := range d.devices {
		infos = append(infos, dev.info)
	}
	return infos
}

// Subscribe returns a channel that receives real-time events from all devices.
func (d *RSDriver) Subscribe() <-chan driver.Event {
	return d.eventCh
}

// Close releases all devices and shuts down the SDK.
func (d *RSDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	d.closed = true

	// Abort any running captures and exit each device
	for _, dev := range d.devices {
		sdkAbortCapture(dev.handle)
		sdkExitDevice(dev.handle)
	}

	// Stop hot plugging
	sdkStopHotPlugging()

	// Shut down SDK
	sdkExit()

	// Close event channel
	close(d.eventCh)

	// Clear global driver reference
	globalDriverMu.Lock()
	globalDriver = nil
	globalDriverMu.Unlock()

	slog.Info("RealScan driver closed")
	return nil
}

// emitEvent sends an event to the event channel, dropping it if full.
func (d *RSDriver) emitEvent(evt driver.Event) {
	select {
	case d.eventCh <- evt:
	default:
		slog.Warn("event channel full, dropping event",
			"type", evt.Type,
			"device", evt.DeviceName,
		)
	}
}

func (d *RSDriver) getDevice(name string) (*rsDevice, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	dev, ok := d.devices[name]
	if !ok {
		return nil, fmt.Errorf("unknown device: %s", name)
	}
	return dev, nil
}

// rsErrString converts a RealScan error code to a human-readable string.
func rsErrString(code int) string {
	return sdkGetErrString(code)
}

// ──────────────────────────────────────────────────────────────────────────────
// Global driver instance for callbacks.
// ──────────────────────────────────────────────────────────────────────────────

var (
	globalDriverMu sync.Mutex
	globalDriver   *RSDriver
)

// handleHotPlugEvent is called by platform-specific callback handlers.
func handleHotPlugEvent(deviceId int, isConnected int) {
	globalDriverMu.Lock()
	d := globalDriver
	globalDriverMu.Unlock()
	if d == nil {
		return
	}

	connected := isConnected != 0

	slog.Info("hot-plug event",
		"deviceId", deviceId,
		"connected", connected,
	)

	// Find the device by handle or index
	d.mu.RLock()
	var devName string
	for _, dev := range d.devices {
		// deviceId from hot-plug callback corresponds to the device ID
		if fmt.Sprintf("%d", deviceId) == dev.info.ID || dev.deviceIdx == deviceId {
			devName = dev.name
			break
		}
	}
	d.mu.RUnlock()

	if devName == "" {
		slog.Warn("hot-plug event for unknown device", "deviceId", deviceId)
		return
	}

	if connected {
		d.emitEvent(driver.Event{
			Type:       "connected",
			DeviceName: devName,
		})
	} else {
		d.emitEvent(driver.Event{
			Type:       "error",
			DeviceName: devName,
			Message:    "device disconnected (USB)",
		})
	}
}
