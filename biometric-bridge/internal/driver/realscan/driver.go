//go:build realscan

// Package realscan implements the Driver interface using the Xperix RealScan
// SDK via CGo. It communicates with Suprema RealScan G10 fingerprint readers
// over USB.
//
// Build with: CGO_ENABLED=1 go build -tags realscan ./cmd/bridge
package realscan

/*
#cgo linux LDFLAGS: -ldl
#cgo windows LDFLAGS: -lkernel32

#include <stdlib.h>
#include <stdint.h>
#include <string.h>

#ifdef _WIN32
    #include <windows.h>
    typedef HMODULE dl_handle;
    #define RTLD_NOW 0
#else
    #include <dlfcn.h>
    typedef void* dl_handle;
#endif

// ──────────────────────────────────────────────────────────────────────────────
// Constants matching RS_ParamDef.h / RS_Error.h
// ──────────────────────────────────────────────────────────────────────────────

// Capture modes
#define RS_CAPTURE_FLAT_SINGLE_FINGER  2
#define RS_CAPTURE_FLAT_TWO_FINGERS    3
#define RS_CAPTURE_FLAT_LEFT_FOUR      4
#define RS_CAPTURE_FLAT_RIGHT_FOUR     5
#define RS_CAPTURE_FLAT_TWO_THUMBS     6
#define RS_CAPTURE_ROLL_FINGER         1

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

// LED mode indices (for RS_SetModeLED)
#define RS_LED_MODE_ALL             0x00
#define RS_LED_MODE_LEFT_FINGER4    0x01
#define RS_LED_MODE_RIGHT_FINGER4   0x02
#define RS_LED_MODE_TWO_THUMB       0x03
#define RS_LED_MODE_ROLL            0x04

// LED colors (for RS_SetFingerLED)
#define RS_LED_OFF    0x00
#define RS_LED_GREEN  0x01
#define RS_LED_RED    0x02
#define RS_LED_YELLOW 0x03

// Finger indices (for RS_SetFingerLED)
#define RS_FINGER_ALL           0
#define RS_FINGER_LEFT_LITTLE   1
#define RS_FINGER_LEFT_RING     2
#define RS_FINGER_LEFT_MIDDLE   3
#define RS_FINGER_LEFT_INDEX    4
#define RS_FINGER_LEFT_THUMB    5
#define RS_FINGER_RIGHT_THUMB   6
#define RS_FINGER_RIGHT_INDEX   7
#define RS_FINGER_RIGHT_MIDDLE  8
#define RS_FINGER_RIGHT_RING    9
#define RS_FINGER_RIGHT_LITTLE  10
#define RS_FINGER_TWO_THUMB     11
#define RS_FINGER_LEFT_FOUR     12
#define RS_FINGER_RIGHT_FOUR    13

// Slap types (for RS_TakeImageDataSegment)
#define RS_SLAP_LEFT_FOUR       1
#define RS_SLAP_RIGHT_FOUR      2
#define RS_SLAP_TWO_THUMB       4

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

// Segmentation data structures (from RS_Data.h)
typedef struct {
    int x;
    int y;
} RSPoint;

typedef struct {
    int     fingerType;
    RSPoint fingerPosition[4];
    int     imageQuality;
    int     rotation;
    int     reserved[3];
} RSSlapInfo;

// ──────────────────────────────────────────────────────────────────────────────
// Function pointer typedefs for dynamically loaded SDK functions
// ──────────────────────────────────────────────────────────────────────────────

// SDK lifecycle
typedef int (*fn_RS_InitSDK)(const char* configDir, int options, int* numOfDevice);
typedef int (*fn_RS_ExitSDK)(void);  // mangled as _Z10RS_ExitSDKv in the .so

// Device lifecycle
typedef int (*fn_RS_InitDevice)(int deviceIndex, int* deviceHandle);
typedef int (*fn_RS_ExitDevice)(int deviceHandle);
typedef int (*fn_RS_GetNumOfDevice)(int* numOfDevice);
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
// Note: RS_StopHotPlugging does not exist in the SDK binary

// Beep
typedef int (*fn_RS_Beep)(int deviceHandle, int beepPattern);

// Error string — exported as RS_GetErrString (not RS_GetErrStringChar)
typedef int (*fn_RS_GetErrString)(int errorCode, char* errorMsg);

// Hot plug callback: void(int deviceId, int isConnected) — using int for bool on Linux
typedef void (*RSHotPlugCallback)(int deviceId, int isConnected);
typedef int (*fn_RS_RegisterHotPluggingCallback)(RSHotPlugCallback callback);

// LED control
typedef int (*fn_RS_SetFingerLED)(int deviceHandle, int fingerIndex, int ledColor);
typedef int (*fn_RS_SetModeLED)(int deviceHandle, int ledIndex, int isOn);

// Extended capture with finger index + LED
typedef int (*fn_RS_TakeImageDataEx)(int deviceHandle, int timeout,
                                     int fingerIndex, int withLED,
                                     unsigned char** imageData, int* width, int* height);

// Segmentation — all-in-one capture + segment
typedef int (*fn_RS_TakeImageDataSegment)(int deviceHandle, int timeout,
                                          unsigned char** imageData, int* imageWidth, int* imageHeight,
                                          int* captureResult, int slapType, int* numOfFinger,
                                          RSSlapInfo** slapInfo,
                                          unsigned char*** fingerImageData,
                                          int** fingerImageWidth, int** fingerImageHeight);

// ──────────────────────────────────────────────────────────────────────────────
// SDK handle and loaded function pointers
// ──────────────────────────────────────────────────────────────────────────────

static dl_handle sdk_handle = NULL;

static fn_RS_InitSDK                    p_InitSDK;
static fn_RS_ExitSDK                    p_ExitSDK;
static fn_RS_InitDevice                 p_InitDevice;
static fn_RS_ExitDevice                 p_ExitDevice;
static fn_RS_GetNumOfDevice             p_GetNumOfDevice;
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
static fn_RS_Beep                       p_Beep;
static fn_RS_GetErrString               p_GetErrString;
static fn_RS_RegisterHotPluggingCallback p_RegisterHotPluggingCallback;
static fn_RS_SetFingerLED               p_SetFingerLED;
static fn_RS_SetModeLED                 p_SetModeLED;
static fn_RS_TakeImageDataEx            p_TakeImageDataEx;
static fn_RS_TakeImageDataSegment       p_TakeImageDataSegment;

// Platform-specific dynamic loading wrappers
#ifdef _WIN32
static dl_handle dl_open(const char* path) {
    return LoadLibraryA(path);
}

static void* dl_sym(dl_handle handle, const char* symbol) {
    return (void*)GetProcAddress(handle, symbol);
}

static void dl_close(dl_handle handle) {
    FreeLibrary(handle);
}

static const char* dl_error(void) {
    static char buf[512];
    DWORD err = GetLastError();
    FormatMessageA(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
                   NULL, err, 0, buf, sizeof(buf), NULL);
    return buf;
}
#else
static dl_handle dl_open(const char* path) {
    return dlopen(path, RTLD_NOW);
}

static void* dl_sym(dl_handle handle, const char* symbol) {
    return dlsym(handle, symbol);
}

static void dl_close(dl_handle handle) {
    dlclose(handle);
}

static const char* dl_error(void) {
    return dlerror();
}
#endif

// loadSDK dynamically loads the RealScan shared library and resolves symbols.
static int loadSDK(const char* libPath) {
    sdk_handle = dl_open(libPath);
    if (!sdk_handle) return -1;

    p_InitSDK             = (fn_RS_InitSDK)dl_sym(sdk_handle, "RS_InitSDK");
    p_ExitSDK             = (fn_RS_ExitSDK)dl_sym(sdk_handle, "_Z10RS_ExitSDKv");
    p_InitDevice          = (fn_RS_InitDevice)dl_sym(sdk_handle, "RS_InitDevice");
    p_ExitDevice          = (fn_RS_ExitDevice)dl_sym(sdk_handle, "RS_ExitDevice");
    p_GetNumOfDevice      = (fn_RS_GetNumOfDevice)dl_sym(sdk_handle, "RS_GetNumOfDevice");
    p_GetDeviceInfo       = (fn_RS_GetDeviceInfo)dl_sym(sdk_handle, "RS_GetDeviceInfo");
    p_SetCaptureMode      = (fn_RS_SetCaptureMode)dl_sym(sdk_handle, "RS_SetCaptureMode");
    p_TakeImageData       = (fn_RS_TakeImageData)dl_sym(sdk_handle, "RS_TakeImageData");
    p_FreeImageData       = (fn_RS_FreeImageData)dl_sym(sdk_handle, "RS_FreeImageData");
    p_GetQualityScore     = (fn_RS_GetQualityScore)dl_sym(sdk_handle, "RS_GetQualityScore");
    p_AbortCapture        = (fn_RS_AbortCapture)dl_sym(sdk_handle, "RS_AbortCapture");
    p_IsCapturing         = (fn_RS_IsCapturing)dl_sym(sdk_handle, "RS_IsCapturing");
    p_SetAutomaticCalibrate = (fn_RS_SetAutomaticCalibrate)dl_sym(sdk_handle, "RS_SetAutomaticCalibrate");
    p_SetPreProcessing    = (fn_RS_SetPreProcessing)dl_sym(sdk_handle, "RS_SetPreProcessing");
    p_StartHotPlugging    = (fn_RS_StartHotPlugging)dl_sym(sdk_handle, "RS_StartHotPlugging");
    p_Beep                = (fn_RS_Beep)dl_sym(sdk_handle, "RS_Beep");
    p_GetErrString        = (fn_RS_GetErrString)dl_sym(sdk_handle, "RS_GetErrString");
    p_RegisterHotPluggingCallback = (fn_RS_RegisterHotPluggingCallback)dl_sym(sdk_handle, "RS_RegisterHotPluggingCallback");
    p_SetFingerLED        = (fn_RS_SetFingerLED)dl_sym(sdk_handle, "RS_SetFingerLED");
    p_SetModeLED          = (fn_RS_SetModeLED)dl_sym(sdk_handle, "RS_SetModeLED");
    p_TakeImageDataEx     = (fn_RS_TakeImageDataEx)dl_sym(sdk_handle, "RS_TakeImageDataEx");
    p_TakeImageDataSegment = (fn_RS_TakeImageDataSegment)dl_sym(sdk_handle, "RS_TakeImageDataSegment");

    // Required symbols (hot plugging is optional — may not be present in all SDK versions)
    if (!p_InitSDK || !p_ExitSDK || !p_InitDevice || !p_ExitDevice ||
        !p_GetNumOfDevice || !p_GetDeviceInfo || !p_SetCaptureMode ||
        !p_TakeImageData || !p_FreeImageData || !p_GetQualityScore ||
        !p_AbortCapture || !p_IsCapturing || !p_SetAutomaticCalibrate) {
        dl_close(sdk_handle);
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
    return p_GetNumOfDevice(numOfDevice);
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
    // RS_StopHotPlugging does not exist in this SDK version — no-op
    return 0;
}

static int sdk_beep(int deviceHandle, int beepPattern) {
    if (p_Beep == NULL) return 0;
    return p_Beep(deviceHandle, beepPattern);
}

static int sdk_get_err_string(int errorCode, char* errorMsg) {
    if (p_GetErrString == NULL) {
        errorMsg[0] = '\0';
        return -1;
    }
    return p_GetErrString(errorCode, errorMsg);
}

static int sdk_register_hot_plug_callback(RSHotPlugCallback callback) {
    if (p_RegisterHotPluggingCallback == NULL) return -1;
    return p_RegisterHotPluggingCallback(callback);
}

static int sdk_set_finger_led(int deviceHandle, int fingerIndex, int ledColor) {
    if (p_SetFingerLED == NULL) return 0; // optional
    return p_SetFingerLED(deviceHandle, fingerIndex, ledColor);
}

static int sdk_set_mode_led(int deviceHandle, int ledIndex, int isOn) {
    if (p_SetModeLED == NULL) return 0; // optional
    return p_SetModeLED(deviceHandle, ledIndex, isOn);
}

static int sdk_take_image_data_ex(int deviceHandle, int timeout,
                                   int fingerIndex, int withLED,
                                   unsigned char** imageData, int* width, int* height) {
    if (p_TakeImageDataEx == NULL) {
        // Fall back to non-ex version if not available
        return p_TakeImageData(deviceHandle, timeout, imageData, width, height);
    }
    return p_TakeImageDataEx(deviceHandle, timeout, fingerIndex, withLED, imageData, width, height);
}

static int sdk_take_image_data_segment(int deviceHandle, int timeout,
                                        unsigned char** imageData, int* imageWidth, int* imageHeight,
                                        int* captureResult, int slapType, int* numOfFinger,
                                        RSSlapInfo** slapInfo,
                                        unsigned char*** fingerImageData,
                                        int** fingerImageWidth, int** fingerImageHeight) {
    if (p_TakeImageDataSegment == NULL) return -999;
    return p_TakeImageDataSegment(deviceHandle, timeout, imageData, imageWidth, imageHeight,
                                   captureResult, slapType, numOfFinger, slapInfo,
                                   fingerImageData, fingerImageWidth, fingerImageHeight);
}

// Helper to access segmented finger image data from Go.
// The SDK returns unsigned char** (array of pointers) and int* (arrays).
// Go cannot directly index triple-pointer types, so these helpers do it in C.
static unsigned char* get_finger_image(unsigned char** images, int index) {
    return images[index];
}
static int get_finger_dim(int* dims, int index) {
    return dims[index];
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

// setFingerLEDs lights the mode LED and the individual finger LED for the
// given position. Call clearLEDs to turn them off after capture.
func setFingerLEDs(handle C.int, finger driver.FingerPosition) {
	info, ok := fingerLEDMap[finger]
	if !ok {
		return
	}

	// Turn on the mode LED (left-4, right-4, thumbs, or roll icon)
	rc := C.sdk_set_mode_led(handle, C.int(info.modeLED), 1)
	if rc != C.RS_SUCCESS {
		slog.Debug("set mode LED failed", "mode", info.modeLED, "error", rsErrString(int(rc)))
	}

	// Light the specific finger LED in green
	rc = C.sdk_set_finger_led(handle, C.int(info.fingerIndex), C.RS_LED_GREEN)
	if rc != C.RS_SUCCESS {
		slog.Debug("set finger LED failed", "finger", info.fingerIndex, "error", rsErrString(int(rc)))
	}
}

// clearLEDs turns off all mode and finger LEDs.
func clearLEDs(handle C.int) {
	C.sdk_set_mode_led(handle, C.RS_LED_MODE_ALL, 0)
	C.sdk_set_finger_led(handle, C.RS_FINGER_ALL, C.RS_LED_OFF)
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
// shared library (e.g., libRS_SDK.so.2.2.0.2470).
func New(libPath string) (*RSDriver, error) {
	cPath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cPath))

	rc := C.loadSDK(cPath)
	if rc != 0 {
		return nil, fmt.Errorf("failed to load RealScan SDK from %s (error: %d, dlerror: %s)",
			libPath, rc, C.GoString(C.dl_error()))
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

	handle := C.int(dev.handle)

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
			C.sdk_abort_capture(handle)
		case <-done:
		}
	}()

	// Blocking capture — blocks until finger placed and image acquired
	var imageData *C.uchar
	var width, height C.int

	// Use extended capture if a finger hint is available (enables automatic
	// LED feedback during capture — green on success, red on failure)
	var rc C.int
	if finger != driver.FingerNone {
		info := fingerLEDMap[finger]
		rc = C.sdk_take_image_data_ex(handle, C.int(defaultCaptureTimeoutMS),
			C.int(info.fingerIndex), 1, // withLED=true
			&imageData, &width, &height)
	} else {
		rc = C.sdk_take_image_data(handle, C.int(defaultCaptureTimeoutMS),
			&imageData, &width, &height)
	}
	close(done) // stop cancellation goroutine

	if rc != C.RS_SUCCESS {
		if rc == C.RS_ERR_CAPTURE_ABORTED {
			return nil, fmt.Errorf("scan cancelled on device %q", deviceName)
		}
		if rc == C.RS_ERR_CAPTURE_TIMEOUT {
			return nil, fmt.Errorf("%w on device %q", driver.ErrScanTimeout, deviceName)
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
		Width:    int(width),
		Height:   int(height),
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

	handle := C.int(dev.handle)

	// Switch capture mode to the multi-finger mode
	rc := C.sdk_set_capture_mode(handle, C.int(modeInfo.captureMode),
		C.RS_AUTO_SENSITIVITY_HIGH, 1)
	if rc != C.RS_SUCCESS {
		return nil, fmt.Errorf("failed to set capture mode %s on device %q: %s (code %d)",
			mode, deviceName, rsErrString(int(rc)), rc)
	}

	// Restore single-finger capture mode when done
	defer func() {
		rc := C.sdk_set_capture_mode(handle,
			C.RS_CAPTURE_FLAT_SINGLE_FINGER,
			C.RS_AUTO_SENSITIVITY_HIGH, 1)
		if rc != C.RS_SUCCESS {
			slog.Warn("failed to restore single-finger capture mode",
				"device", deviceName, "error", rsErrString(int(rc)))
		}
	}()

	// Light mode LED to guide finger placement
	C.sdk_set_mode_led(handle, C.int(modeInfo.modeLED), 1)
	defer clearLEDs(handle)

	// Set up context cancellation to abort capture
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			C.sdk_abort_capture(handle)
		case <-done:
		}
	}()

	// Blocking segmented capture
	var imageData *C.uchar
	var imageWidth, imageHeight C.int
	var captureResult C.int
	var numOfFinger C.int
	var slapInfo *C.RSSlapInfo
	var fingerImageData **C.uchar
	var fingerImageWidth *C.int
	var fingerImageHeight *C.int

	rc = C.sdk_take_image_data_segment(handle, C.int(defaultSlapCaptureTimeoutMS),
		&imageData, &imageWidth, &imageHeight,
		&captureResult, C.int(modeInfo.slapType), &numOfFinger,
		&slapInfo, &fingerImageData, &fingerImageWidth, &fingerImageHeight)
	close(done)

	if rc != C.RS_SUCCESS {
		if rc == C.RS_ERR_CAPTURE_ABORTED {
			return nil, fmt.Errorf("slap scan cancelled on device %q", deviceName)
		}
		if rc == C.RS_ERR_CAPTURE_TIMEOUT {
			return nil, fmt.Errorf("%w on device %q (slap)", driver.ErrScanTimeout, deviceName)
		}
		return nil, fmt.Errorf("slap scan failed on device %q: %s (code %d)",
			deviceName, rsErrString(int(rc)), rc)
	}

	nFingers := int(numOfFinger)

	// Copy the full slap image to Go memory
	slapImageSize := int(imageWidth) * int(imageHeight)
	goSlapImage := C.GoBytes(unsafe.Pointer(imageData), C.int(slapImageSize))

	// Copy each segmented finger image to Go memory
	fingers := make([]driver.ScanResult, 0, nFingers)
	for i := 0; i < nFingers; i++ {
		fImg := C.get_finger_image(fingerImageData, C.int(i))
		fW := int(C.get_finger_dim(fingerImageWidth, C.int(i)))
		fH := int(C.get_finger_dim(fingerImageHeight, C.int(i)))
		fSize := fW * fH
		goFingerImage := C.GoBytes(unsafe.Pointer(fImg), C.int(fSize))

		// Get quality score for this finger
		var nistQuality C.int
		quality := 0
		qrc := C.sdk_get_quality_score(fImg, C.int(fW), C.int(fH), &nistQuality)
		if qrc == C.RS_SUCCESS {
			quality = int(nistQuality)
		} else {
			slog.Debug("quality scoring failed for segmented finger",
				"device", deviceName, "finger", i, "error", rsErrString(int(qrc)))
		}

		// Map the SDK fingerType to our FingerPosition
		var fingerPos driver.FingerPosition
		if slapInfo != nil {
			// Access the i-th RSSlapInfo element
			slapInfoPtr := (*C.RSSlapInfo)(unsafe.Pointer(
				uintptr(unsafe.Pointer(slapInfo)) + uintptr(i)*unsafe.Sizeof(*slapInfo)))
			fingerType := int(slapInfoPtr.fingerType)
			if pos, ok := slapFingerTypeToPosition[fingerType]; ok {
				fingerPos = pos
			}
			// Override quality with SDK-reported quality if available
			if slapInfoPtr.imageQuality > 0 {
				quality = int(slapInfoPtr.imageQuality)
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
	C.sdk_free_image_data(imageData)
	// Note: fingerImageData, fingerImageWidth, fingerImageHeight, and slapInfo
	// are SDK-allocated arrays. The SDK manages their lifetime alongside the
	// main image data returned by RS_TakeImageDataSegment.

	// Success beep
	C.sdk_beep(handle, C.int(rsBeepPattern1))

	slog.Info("slap scan complete",
		"device", deviceName,
		"mode", string(mode),
		"slapWidth", int(imageWidth),
		"slapHeight", int(imageHeight),
		"fingersDetected", nFingers,
	)

	return &driver.SlapScanResult{
		SlapImage:  goSlapImage,
		SlapWidth:  int(imageWidth),
		SlapHeight: int(imageHeight),
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

	handle := C.int(dev.handle)

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
			C.sdk_abort_capture(handle)
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
				C.sdk_abort_capture(handle)
			case <-done:
			}
		}()

		var imageData *C.uchar
		var width, height C.int

		var rc C.int
		if finger != driver.FingerNone {
			info := fingerLEDMap[finger]
			rc = C.sdk_take_image_data_ex(handle, C.int(defaultCaptureTimeoutMS),
				C.int(info.fingerIndex), 1,
				&imageData, &width, &height)
		} else {
			rc = C.sdk_take_image_data(handle, C.int(defaultCaptureTimeoutMS),
				&imageData, &width, &height)
		}
		close(done)

		// Clear LEDs after each impression
		if finger != driver.FingerNone {
			clearLEDs(handle)
		}

		if rc != C.RS_SUCCESS {
			if rc == C.RS_ERR_CAPTURE_ABORTED {
				return fmt.Errorf("enrollment cancelled on device %q (impression %d)", deviceName, i+1)
			}
			if rc == C.RS_ERR_CAPTURE_TIMEOUT {
				return fmt.Errorf("%w on device %q (enrollment impression %d)", driver.ErrScanTimeout, deviceName, i+1)
			}
			return fmt.Errorf("enrollment scan %d failed on device %q: %s (code %d)",
				i+1, deviceName, rsErrString(int(rc)), rc)
		}

		// Free SDK memory — for enrollment the server will request images
		// via separate scan calls or the images are forwarded via events.
		C.sdk_free_image_data(imageData)

		// Success beep for each impression
		C.sdk_beep(handle, C.int(rsBeepPattern1))

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
