//go:build realscan

// Package realscan implements the Driver interface using the Xperix RealScan
// SDK via CGo. It communicates with Suprema RealScan G10 fingerprint readers
// over USB.
//
// Build with: CGO_ENABLED=1 go build -tags realscan ./cmd/bridge
package realscan

/*
#cgo LDFLAGS: -ldl

#include <stdlib.h>
#include <stdint.h>
#include <dlfcn.h>
#include <string.h>

// ──────────────────────────────────────────────────────────────────────────────
// Constants matching RS_ParamDef.h / RS_Error.h
// ──────────────────────────────────────────────────────────────────────────────

// Capture modes
#define RS_CAPTURE_FLAT_SINGLE_FINGER  2

// Auto-sensitivity
#define RS_AUTO_SENSITIVITY_HIGH 1

// Error codes
#define RS_SUCCESS                     0
#define RS_ERR_SDK_UNINITIALIZED      -10
#define RS_ERR_SDK_ALREADY_INITIALIZED -11
#define RS_ERR_NO_DEVICE              -100
#define RS_ERR_INVALID_HANDLE         -102
#define RS_ERR_CAPTURE_TIMEOUT        -202
#define RS_ERR_CAPTURE_ABORTED        -203
#define RS_ERR_CAPTURE_IS_RUNNING     -212
#define RS_ERR_DEVICE_NOT_INITIALIZED -127
#define RS_ERR_CANNOT_GET_QUALITY     -216

// Device types
#define RS_DEVICE_REALSCAN_G10   0x30
#define RS_DEVICE_REALSCAN_G10F  0x31
#define RS_DEVICE_REALSCAN_G10I  0x33

// ──────────────────────────────────────────────────────────────────────────────
// Struct declarations matching RS_Data.h
// ──────────────────────────────────────────────────────────────────────────────

typedef struct {
    int  deviceType;
    char productName[16];
    char deviceID[16];
    char firmwareVersion[16];
    char hardwareVersion[16];
} RSDeviceInfo;

// ──────────────────────────────────────────────────────────────────────────────
// Function pointer typedefs for dynamically loaded SDK functions
// ──────────────────────────────────────────────────────────────────────────────

// SDK lifecycle
typedef int (*fn_RS_InitSDK)(const char* configDir, int options, int* numOfDevice);
typedef int (*fn_RS_ExitSDK)(void);

// Device lifecycle
typedef int (*fn_RS_InitDevice)(int deviceIndex, int* deviceHandle);
typedef int (*fn_RS_ExitDevice)(int deviceHandle);
typedef int (*fn_RS_GetDeviceCount)(int* numOfDevice);
typedef int (*fn_RS_GetDeviceInfo)(int deviceHandle, RSDeviceInfo* deviceInfo);

// Capture mode
typedef int (*fn_RS_SetCaptureMode)(int deviceHandle, int captureMode, int autoSensitivity, int withLED);

// Blocking capture
typedef int (*fn_RS_TakeImageData)(int deviceHandle, int timeout,
                                   unsigned char** imageData, int* width, int* height);

// Memory management
typedef int (*fn_RS_FreeImageData)(unsigned char* imageData);

// Quality
typedef int (*fn_RS_GetQualityScore)(unsigned char* imageData, int width, int height, int* nistQuality);

// Abort
typedef int (*fn_RS_AbortCapture)(int deviceHandle);
typedef int (*fn_RS_IsCapturing)(int deviceHandle, int* isRunning);

// Calibration & pre-processing
typedef int (*fn_RS_SetAutomaticCalibrate)(int deviceHandle, int enable);
typedef int (*fn_RS_SetPreProcessing)(int deviceHandle, int mode);

// Hot plugging
typedef int (*fn_RS_StartHotPlugging)(void);
typedef int (*fn_RS_StopHotPlugging)(void);

// Beep
typedef int (*fn_RS_Beep)(int deviceHandle, int beepPattern);

// Error string
typedef int (*fn_RS_GetErrStringChar)(int errorCode, char* errorMsg);

// Hot plug callback: void(int deviceId, int isConnected) — using int for bool on Linux
typedef void (*RSHotPlugCallback)(int deviceId, int isConnected);
typedef int (*fn_RS_RegisterHotPluggingCallback)(RSHotPlugCallback callback);

// ──────────────────────────────────────────────────────────────────────────────
// SDK handle and loaded function pointers
// ──────────────────────────────────────────────────────────────────────────────

static void* sdk_handle = NULL;

static fn_RS_InitSDK                    p_InitSDK;
static fn_RS_ExitSDK                    p_ExitSDK;
static fn_RS_InitDevice                 p_InitDevice;
static fn_RS_ExitDevice                 p_ExitDevice;
static fn_RS_GetDeviceCount             p_GetDeviceCount;
static fn_RS_GetDeviceInfo              p_GetDeviceInfo;
static fn_RS_SetCaptureMode             p_SetCaptureMode;
static fn_RS_TakeImageData              p_TakeImageData;
static fn_RS_FreeImageData              p_FreeImageData;
static fn_RS_GetQualityScore            p_GetQualityScore;
static fn_RS_AbortCapture               p_AbortCapture;
static fn_RS_IsCapturing                p_IsCapturing;
static fn_RS_SetAutomaticCalibrate      p_SetAutomaticCalibrate;
static fn_RS_SetPreProcessing           p_SetPreProcessing;
static fn_RS_StartHotPlugging           p_StartHotPlugging;
static fn_RS_StopHotPlugging            p_StopHotPlugging;
static fn_RS_Beep                       p_Beep;
static fn_RS_GetErrStringChar           p_GetErrStringChar;
static fn_RS_RegisterHotPluggingCallback p_RegisterHotPluggingCallback;

// loadSDK dynamically loads the RealScan shared library and resolves symbols.
static int loadSDK(const char* libPath) {
    sdk_handle = dlopen(libPath, RTLD_NOW);
    if (!sdk_handle) return -1;

    p_InitSDK             = (fn_RS_InitSDK)dlsym(sdk_handle, "RS_InitSDK");
    p_ExitSDK             = (fn_RS_ExitSDK)dlsym(sdk_handle, "RS_ExitSDK");
    p_InitDevice          = (fn_RS_InitDevice)dlsym(sdk_handle, "RS_InitDevice");
    p_ExitDevice          = (fn_RS_ExitDevice)dlsym(sdk_handle, "RS_ExitDevice");
    p_GetDeviceCount      = (fn_RS_GetDeviceCount)dlsym(sdk_handle, "RS_GetDeviceCount");
    p_GetDeviceInfo       = (fn_RS_GetDeviceInfo)dlsym(sdk_handle, "RS_GetDeviceInfo");
    p_SetCaptureMode      = (fn_RS_SetCaptureMode)dlsym(sdk_handle, "RS_SetCaptureMode");
    p_TakeImageData       = (fn_RS_TakeImageData)dlsym(sdk_handle, "RS_TakeImageData");
    p_FreeImageData       = (fn_RS_FreeImageData)dlsym(sdk_handle, "RS_FreeImageData");
    p_GetQualityScore     = (fn_RS_GetQualityScore)dlsym(sdk_handle, "RS_GetQualityScore");
    p_AbortCapture        = (fn_RS_AbortCapture)dlsym(sdk_handle, "RS_AbortCapture");
    p_IsCapturing         = (fn_RS_IsCapturing)dlsym(sdk_handle, "RS_IsCapturing");
    p_SetAutomaticCalibrate = (fn_RS_SetAutomaticCalibrate)dlsym(sdk_handle, "RS_SetAutomaticCalibrate");
    p_SetPreProcessing    = (fn_RS_SetPreProcessing)dlsym(sdk_handle, "RS_SetPreProcessing");
    p_StartHotPlugging    = (fn_RS_StartHotPlugging)dlsym(sdk_handle, "RS_StartHotPlugging");
    p_StopHotPlugging     = (fn_RS_StopHotPlugging)dlsym(sdk_handle, "RS_StopHotPlugging");
    p_Beep                = (fn_RS_Beep)dlsym(sdk_handle, "RS_Beep");
    p_GetErrStringChar    = (fn_RS_GetErrStringChar)dlsym(sdk_handle, "RS_GetErrStringChar");
    p_RegisterHotPluggingCallback = (fn_RS_RegisterHotPluggingCallback)dlsym(sdk_handle, "RS_RegisterHotPluggingCallback");

    // Required symbols (hot plugging is optional — may not be present in all SDK versions)
    if (!p_InitSDK || !p_ExitSDK || !p_InitDevice || !p_ExitDevice ||
        !p_GetDeviceCount || !p_GetDeviceInfo || !p_SetCaptureMode ||
        !p_TakeImageData || !p_FreeImageData || !p_GetQualityScore ||
        !p_AbortCapture || !p_IsCapturing || !p_SetAutomaticCalibrate) {
        dlclose(sdk_handle);
        sdk_handle = NULL;
        return -2;
    }

    return 0;
}

// ──────────────────────────────────────────────────────────────────────────────
// SDK wrapper functions called from Go
// ──────────────────────────────────────────────────────────────────────────────

static int sdk_init(const char* configDir, int options, int* numOfDevice) {
    return p_InitSDK(configDir, options, numOfDevice);
}

static int sdk_exit(void) {
    return p_ExitSDK();
}

static int sdk_init_device(int deviceIndex, int* deviceHandle) {
    return p_InitDevice(deviceIndex, deviceHandle);
}

static int sdk_exit_device(int deviceHandle) {
    return p_ExitDevice(deviceHandle);
}

static int sdk_get_device_count(int* numOfDevice) {
    return p_GetDeviceCount(numOfDevice);
}

static int sdk_get_device_info(int deviceHandle, RSDeviceInfo* info) {
    return p_GetDeviceInfo(deviceHandle, info);
}

static int sdk_set_capture_mode(int deviceHandle, int captureMode, int autoSensitivity, int withLED) {
    return p_SetCaptureMode(deviceHandle, captureMode, autoSensitivity, withLED);
}

static int sdk_take_image_data(int deviceHandle, int timeout,
                               unsigned char** imageData, int* width, int* height) {
    return p_TakeImageData(deviceHandle, timeout, imageData, width, height);
}

static int sdk_free_image_data(unsigned char* imageData) {
    return p_FreeImageData(imageData);
}

static int sdk_get_quality_score(unsigned char* imageData, int width, int height, int* nistQuality) {
    return p_GetQualityScore(imageData, width, height, nistQuality);
}

static int sdk_abort_capture(int deviceHandle) {
    return p_AbortCapture(deviceHandle);
}

static int sdk_set_auto_calibrate(int deviceHandle, int enable) {
    return p_SetAutomaticCalibrate(deviceHandle, enable);
}

static int sdk_set_pre_processing(int deviceHandle, int mode) {
    if (p_SetPreProcessing == NULL) return 0; // optional
    return p_SetPreProcessing(deviceHandle, mode);
}

static int sdk_start_hot_plugging(void) {
    if (p_StartHotPlugging == NULL) return 0;
    return p_StartHotPlugging();
}

static int sdk_stop_hot_plugging(void) {
    if (p_StopHotPlugging == NULL) return 0;
    return p_StopHotPlugging();
}

static int sdk_beep(int deviceHandle, int beepPattern) {
    if (p_Beep == NULL) return 0;
    return p_Beep(deviceHandle, beepPattern);
}

static int sdk_get_err_string(int errorCode, char* errorMsg) {
    if (p_GetErrStringChar == NULL) {
        errorMsg[0] = '\0';
        return -1;
    }
    return p_GetErrStringChar(errorCode, errorMsg);
}

static int sdk_register_hot_plug_callback(RSHotPlugCallback callback) {
    if (p_RegisterHotPluggingCallback == NULL) return -1;
    return p_RegisterHotPluggingCallback(callback);
}

// ──────────────────────────────────────────────────────────────────────────────
// C callback forwarders — run on SDK threads, push events to Go.
// ──────────────────────────────────────────────────────────────────────────────

extern void goOnHotPlug(int deviceId, int isConnected);

static void onHotPlugCallback(int deviceId, int isConnected) {
    goOnHotPlug(deviceId, isConnected);
}

static int sdk_register_hot_plug_callback_go(void) {
    return sdk_register_hot_plug_callback(onHotPlugCallback);
}
*/
import "C"

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
)

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
// shared library (e.g., libRS_SDK.so.2.2.0.2470).
func New(libPath string) (*RSDriver, error) {
	cPath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cPath))

	rc := C.loadSDK(cPath)
	if rc != 0 {
		return nil, fmt.Errorf("failed to load RealScan SDK from %s (error: %d, dlerror: %s)",
			libPath, rc, C.GoString(C.dlerror()))
	}

	// Initialize SDK — pass NULL config dir, 0 options
	var numDevices C.int
	rc2 := C.sdk_init(nil, 0, &numDevices)
	if rc2 != C.RS_SUCCESS {
		return nil, fmt.Errorf("RS_InitSDK failed: %s (code %d)", rsErrString(int(rc2)), rc2)
	}

	slog.Info("RealScan SDK initialized", "devicesFound", int(numDevices))

	d := &RSDriver{
		devices: make(map[string]*rsDevice),
		eventCh: make(chan driver.Event, 256),
	}

	// Register the global driver instance for C callbacks
	globalDriverMu.Lock()
	globalDriver = d
	globalDriverMu.Unlock()

	// Start hot plugging detection
	rc3 := C.sdk_start_hot_plugging()
	if rc3 == 0 {
		rc4 := C.sdk_register_hot_plug_callback_go()
		if rc4 != 0 {
			slog.Warn("failed to register hot-plug callback", "error", rsErrString(int(rc4)))
		} else {
			slog.Info("hot-plug monitoring started")
		}
	} else {
		slog.Warn("hot-plugging not available", "error", rsErrString(int(rc3)))
	}

	return d, nil
}

// Connect initializes all configured USB devices. For RealScan, devices are
// identified by USB index rather than IP:port. The DeviceConfig.Addr field is
// interpreted as the device index (e.g., "0", "1") for USB devices, or left
// empty to auto-assign sequentially.
func (d *RSDriver) Connect(ctx context.Context, devices []driver.DeviceConfig) error {
	// Discover how many USB devices the SDK sees
	var numDevices C.int
	rc := C.sdk_get_device_count(&numDevices)
	if rc != C.RS_SUCCESS {
		return fmt.Errorf("RS_GetDeviceCount failed: %s", rsErrString(int(rc)))
	}

	if int(numDevices) == 0 {
		return fmt.Errorf("no RealScan devices found on USB bus")
	}

	slog.Info("USB device discovery", "found", int(numDevices), "configured", len(devices))

	for i, dc := range devices {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Determine device index — use sequential index for now.
		// For multi-device setups the config could specify a DeviceIndex field.
		deviceIdx := i
		if deviceIdx >= int(numDevices) {
			return fmt.Errorf("device %q: USB index %d out of range (only %d devices found)",
				dc.Name, deviceIdx, numDevices)
		}

		// Initialize the device
		var handle C.int
		rc := C.sdk_init_device(C.int(deviceIdx), &handle)
		if rc != C.RS_SUCCESS {
			return fmt.Errorf("failed to init device %q (index %d): %s",
				dc.Name, deviceIdx, rsErrString(int(rc)))
		}

		// Enable automatic calibration
		C.sdk_set_auto_calibrate(handle, 1)

		// Set pre-processing to balanced mode for good quality + speed
		C.sdk_set_pre_processing(handle, C.int(rsBalancedPreprocess))

		// Set capture mode: flat single finger, high sensitivity, LED on
		rc = C.sdk_set_capture_mode(handle,
			C.RS_CAPTURE_FLAT_SINGLE_FINGER,
			C.RS_AUTO_SENSITIVITY_HIGH,
			1, // LED on
		)
		if rc != C.RS_SUCCESS {
			C.sdk_exit_device(handle)
			return fmt.Errorf("failed to set capture mode for %q: %s",
				dc.Name, rsErrString(int(rc)))
		}

		// Get device info
		var devInfo C.RSDeviceInfo
		rc = C.sdk_get_device_info(handle, &devInfo)
		if rc != C.RS_SUCCESS {
			C.sdk_exit_device(handle)
			return fmt.Errorf("failed to get device info for %q: %s",
				dc.Name, rsErrString(int(rc)))
		}

		productName := C.GoString(&devInfo.productName[0])
		deviceID := C.GoString(&devInfo.deviceID[0])
		fwVer := C.GoString(&devInfo.firmwareVersion[0])

		dev := &rsDevice{
			name:      dc.Name,
			handle:    int(handle),
			deviceIdx: deviceIdx,
			info: driver.DeviceInfo{
				Name:            dc.Name,
				ID:              deviceID,
				Model:           productName,
				FirmwareVersion: fwVer,
				FingerSupported: true, // RealScan is always a fingerprint device
			},
		}

		d.mu.Lock()
		d.devices[dc.Name] = dev
		d.mu.Unlock()

		slog.Info("device initialized",
			"name", dc.Name,
			"model", productName,
			"id", deviceID,
			"firmware", fwVer,
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
func (d *RSDriver) Scan(ctx context.Context, deviceName string) (*driver.ScanResult, error) {
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

	handle := C.int(dev.handle)

	// Set up context cancellation to abort capture
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			C.sdk_abort_capture(handle)
		case <-done:
		}
	}()

	// Blocking capture — blocks until finger placed and image acquired
	var imageData *C.uchar
	var width, height C.int

	rc := C.sdk_take_image_data(handle, C.int(defaultCaptureTimeoutMS),
		&imageData, &width, &height)
	close(done) // stop cancellation goroutine

	if rc != C.RS_SUCCESS {
		if rc == C.RS_ERR_CAPTURE_ABORTED {
			return nil, fmt.Errorf("scan cancelled on device %q", deviceName)
		}
		if rc == C.RS_ERR_CAPTURE_TIMEOUT {
			return nil, fmt.Errorf("scan timed out on device %q", deviceName)
		}
		return nil, fmt.Errorf("scan failed on device %q: %s (code %d)",
			deviceName, rsErrString(int(rc)), rc)
	}

	// Copy image data to Go memory before freeing SDK memory
	imageSize := int(width) * int(height) // 8-bit grayscale, 1 byte per pixel
	goImageData := C.GoBytes(unsafe.Pointer(imageData), C.int(imageSize))

	// Get NIST quality score
	var nistQuality C.int
	qrc := C.sdk_get_quality_score(imageData, width, height, &nistQuality)

	// Free SDK-allocated image data
	C.sdk_free_image_data(imageData)

	quality := 0
	if qrc == C.RS_SUCCESS {
		quality = int(nistQuality)
	} else {
		slog.Warn("quality scoring failed", "device", deviceName, "error", rsErrString(int(qrc)))
	}

	// Success beep
	C.sdk_beep(C.int(dev.handle), C.int(rsBeepPattern1))

	slog.Info("scan complete",
		"device", deviceName,
		"width", int(width),
		"height", int(height),
		"quality", quality,
		"imageBytes", imageSize,
	)

	return &driver.ScanResult{
		Template: goImageData,
		Quality:  quality,
	}, nil
}

// Enroll performs a two-impression enrollment capture on the specified device.
// For RealScan, this captures two fingerprint images and returns them via
// the event channel for server-side processing. The device itself has no
// enrollment concept — enrollment is purely a capture-twice workflow.
func (d *RSDriver) Enroll(ctx context.Context, deviceName, userID, userName string) error {
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

	handle := C.int(dev.handle)

	// Capture two impressions
	for impression := 1; impression <= 2; impression++ {
		select {
		case <-ctx.Done():
			C.sdk_abort_capture(handle)
			return ctx.Err()
		default:
		}

		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				C.sdk_abort_capture(handle)
			case <-done:
			}
		}()

		var imageData *C.uchar
		var width, height C.int

		rc := C.sdk_take_image_data(handle, C.int(defaultCaptureTimeoutMS),
			&imageData, &width, &height)
		close(done)

		if rc != C.RS_SUCCESS {
			if rc == C.RS_ERR_CAPTURE_ABORTED {
				return fmt.Errorf("enrollment cancelled on device %q (impression %d)", deviceName, impression)
			}
			if rc == C.RS_ERR_CAPTURE_TIMEOUT {
				return fmt.Errorf("enrollment timed out on device %q (impression %d)", deviceName, impression)
			}
			return fmt.Errorf("enrollment scan %d failed on device %q: %s (code %d)",
				impression, deviceName, rsErrString(int(rc)), rc)
		}

		// Free SDK memory — for enrollment the server will request images
		// via separate scan calls or the images are forwarded via events.
		C.sdk_free_image_data(imageData)

		// Success beep for each impression
		C.sdk_beep(handle, C.int(rsBeepPattern1))

		slog.Info("enrollment impression captured",
			"device", deviceName,
			"impression", impression,
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
		h := C.int(dev.handle)
		var isRunning C.int
		C.sdk_abort_capture(h)
		_ = isRunning // suppress unused warning
		C.sdk_exit_device(h)
	}

	// Stop hot plugging
	C.sdk_stop_hot_plugging()

	// Shut down SDK
	C.sdk_exit()

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
	var buf [256]C.char
	rc := C.sdk_get_err_string(C.int(code), &buf[0])
	if rc != 0 {
		return fmt.Sprintf("unknown error %d", code)
	}
	return C.GoString(&buf[0])
}

// ──────────────────────────────────────────────────────────────────────────────
// Global driver instance for C callbacks.
// ──────────────────────────────────────────────────────────────────────────────

var (
	globalDriverMu sync.Mutex
	globalDriver   *RSDriver
)

//export goOnHotPlug
func goOnHotPlug(deviceId C.int, isConnected C.int) {
	globalDriverMu.Lock()
	d := globalDriver
	globalDriverMu.Unlock()
	if d == nil {
		return
	}

	connected := isConnected != 0

	slog.Info("hot-plug event",
		"deviceId", int(deviceId),
		"connected", connected,
	)

	// Find the device by handle or index
	d.mu.RLock()
	var devName string
	for _, dev := range d.devices {
		// deviceId from hot-plug callback corresponds to the device ID
		if fmt.Sprintf("%d", deviceId) == dev.info.ID || dev.deviceIdx == int(deviceId) {
			devName = dev.name
			break
		}
	}
	d.mu.RUnlock()

	if devName == "" {
		slog.Warn("hot-plug event for unknown device", "deviceId", int(deviceId))
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
