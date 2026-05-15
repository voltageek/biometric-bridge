//go:build windows && realscan

package realscan

/*
#cgo LDFLAGS: -lkernel32

#include <windows.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

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
typedef int (__stdcall *fn_RS_InitSDK)(const char* configDir, int options, int* numOfDevice);
typedef int (__stdcall *fn_RS_ExitSDK)(void);

// Device lifecycle
typedef int (__stdcall *fn_RS_InitDevice)(int deviceIndex, int* deviceHandle);
typedef int (__stdcall *fn_RS_ExitDevice)(int deviceHandle);
typedef int (__stdcall *fn_RS_GetNumOfDevice)(int* numOfDevice);
typedef int (__stdcall *fn_RS_GetDeviceInfo)(int deviceHandle, RSDeviceInfo* deviceInfo);

// Capture mode
typedef int (__stdcall *fn_RS_SetCaptureMode)(int deviceHandle, int captureMode, int autoSensitivity, int withLED);

// Blocking capture
typedef int (__stdcall *fn_RS_TakeImageData)(int deviceHandle, int timeout,
                                   unsigned char** imageData, int* width, int* height);

// Memory management
typedef int (__stdcall *fn_RS_FreeImageData)(unsigned char* imageData);

// Quality
typedef int (__stdcall *fn_RS_GetQualityScore)(unsigned char* imageData, int width, int height, int* nistQuality);

// Abort
typedef int (__stdcall *fn_RS_AbortCapture)(int deviceHandle);
typedef int (__stdcall *fn_RS_IsCapturing)(int deviceHandle, int* isRunning);

// Calibration & pre-processing
typedef int (__stdcall *fn_RS_SetAutomaticCalibrate)(int deviceHandle, int enable);
typedef int (__stdcall *fn_RS_SetPreProcessing)(int deviceHandle, int mode);

// Hot plugging
typedef int (__stdcall *fn_RS_StartHotPlugging)(void);

// Beep
typedef int (__stdcall *fn_RS_Beep)(int deviceHandle, int beepPattern);

// Error string
typedef int (__stdcall *fn_RS_GetErrString)(int errorCode, char* errorMsg);

// Hot plug callback: void(int deviceId, int isConnected)
typedef void (__stdcall *RSHotPlugCallback)(int deviceId, int isConnected);
typedef int (__stdcall *fn_RS_RegisterHotPluggingCallback)(RSHotPlugCallback callback);

// LED control
typedef int (__stdcall *fn_RS_SetFingerLED)(int deviceHandle, int fingerIndex, int ledColor);
typedef int (__stdcall *fn_RS_SetModeLED)(int deviceHandle, int ledIndex, int isOn);

// Extended capture with finger index + LED
typedef int (__stdcall *fn_RS_TakeImageDataEx)(int deviceHandle, int timeout,
                                     int fingerIndex, int withLED,
                                     unsigned char** imageData, int* width, int* height);

// Segmentation — all-in-one capture + segment
typedef int (__stdcall *fn_RS_TakeImageDataSegment)(int deviceHandle, int timeout,
                                          unsigned char** imageData, int* imageWidth, int* imageHeight,
                                          int* captureResult, int slapType, int* numOfFinger,
                                          RSSlapInfo** slapInfo,
                                          unsigned char*** fingerImageData,
                                          int** fingerImageWidth, int** fingerImageHeight);

// ──────────────────────────────────────────────────────────────────────────────
// SDK handle and loaded function pointers
// ──────────────────────────────────────────────────────────────────────────────

static HMODULE sdk_handle = NULL;

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

// loadSDK dynamically loads the RealScan DLL and resolves symbols.
static int loadSDK(const char* libPath) {
    sdk_handle = LoadLibraryA(libPath);
    if (!sdk_handle) return -1;

    p_InitSDK             = (fn_RS_InitSDK)GetProcAddress(sdk_handle, "RS_InitSDK");
    p_ExitSDK             = (fn_RS_ExitSDK)GetProcAddress(sdk_handle, "RS_ExitSDK");
    p_InitDevice          = (fn_RS_InitDevice)GetProcAddress(sdk_handle, "RS_InitDevice");
    p_ExitDevice          = (fn_RS_ExitDevice)GetProcAddress(sdk_handle, "RS_ExitDevice");
    p_GetNumOfDevice      = (fn_RS_GetNumOfDevice)GetProcAddress(sdk_handle, "RS_GetNumOfDevice");
    p_GetDeviceInfo       = (fn_RS_GetDeviceInfo)GetProcAddress(sdk_handle, "RS_GetDeviceInfo");
    p_SetCaptureMode      = (fn_RS_SetCaptureMode)GetProcAddress(sdk_handle, "RS_SetCaptureMode");
    p_TakeImageData       = (fn_RS_TakeImageData)GetProcAddress(sdk_handle, "RS_TakeImageData");
    p_FreeImageData       = (fn_RS_FreeImageData)GetProcAddress(sdk_handle, "RS_FreeImageData");
    p_GetQualityScore     = (fn_RS_GetQualityScore)GetProcAddress(sdk_handle, "RS_GetQualityScore");
    p_AbortCapture        = (fn_RS_AbortCapture)GetProcAddress(sdk_handle, "RS_AbortCapture");
    p_IsCapturing         = (fn_RS_IsCapturing)GetProcAddress(sdk_handle, "RS_IsCapturing");
    p_SetAutomaticCalibrate = (fn_RS_SetAutomaticCalibrate)GetProcAddress(sdk_handle, "RS_SetAutomaticCalibrate");
    p_SetPreProcessing    = (fn_RS_SetPreProcessing)GetProcAddress(sdk_handle, "RS_SetPreProcessing");
    p_StartHotPlugging    = (fn_RS_StartHotPlugging)GetProcAddress(sdk_handle, "RS_StartHotPlugging");
    p_Beep                = (fn_RS_Beep)GetProcAddress(sdk_handle, "RS_Beep");
    p_GetErrString        = (fn_RS_GetErrString)GetProcAddress(sdk_handle, "RS_GetErrString");
    p_RegisterHotPluggingCallback = (fn_RS_RegisterHotPluggingCallback)GetProcAddress(sdk_handle, "RS_RegisterHotPluggingCallback");
    p_SetFingerLED        = (fn_RS_SetFingerLED)GetProcAddress(sdk_handle, "RS_SetFingerLED");
    p_SetModeLED          = (fn_RS_SetModeLED)GetProcAddress(sdk_handle, "RS_SetModeLED");
    p_TakeImageDataEx     = (fn_RS_TakeImageDataEx)GetProcAddress(sdk_handle, "RS_TakeImageDataEx");
    p_TakeImageDataSegment = (fn_RS_TakeImageDataSegment)GetProcAddress(sdk_handle, "RS_TakeImageDataSegment");

    // Required symbols (hot plugging is optional — may not be present in all SDK versions)
    // Note: RS_ExitSDK may not be exported on Windows - treat as optional
    if (!p_InitSDK || !p_InitDevice || !p_ExitDevice ||
        !p_GetNumOfDevice || !p_GetDeviceInfo || !p_SetCaptureMode ||
        !p_TakeImageData || !p_FreeImageData || !p_GetQualityScore ||
        !p_AbortCapture || !p_IsCapturing || !p_SetAutomaticCalibrate) {
        FreeLibrary(sdk_handle);
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
    // RS_ExitSDK may not be exported on Windows - treat as optional
    if (p_ExitSDK == NULL) return 0;
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

static void __stdcall onHotPlugCallback(int deviceId, int isConnected) {
    goOnHotPlug(deviceId, isConnected);
}

static int sdk_register_hot_plug_callback_go(void) {
    return sdk_register_hot_plug_callback(onHotPlugCallback);
}
*/
import "C"
import "unsafe"

// This file contains Windows-specific SDK loading and wrapper functions.
// Uses LoadLibrary/GetProcAddress instead of dlopen/dlsym, with __stdcall convention.

// ──────────────────────────────────────────────────────────────────────────────
// Go wrapper functions - these are called by driver.go
// ──────────────────────────────────────────────────────────────────────────────

// SDK error codes
const (
	RS_SUCCESS             = 0
	RS_ERR_CAPTURE_TIMEOUT = -202
	RS_ERR_CAPTURE_ABORTED = -203
)

// Capture modes
const (
	RS_CAPTURE_FLAT_SINGLE_FINGER = 2
	RS_AUTO_SENSITIVITY_HIGH      = 1
)

// LED constants
const (
	RS_LED_OFF      = 0x00
	RS_LED_GREEN    = 0x01
	RS_LED_MODE_ALL = 0x00
	RS_FINGER_ALL   = 0
)

// DeviceInfo represents device information from the SDK
type DeviceInfo struct {
	DeviceType      int
	ProductName     string
	DeviceID        string
	FirmwareVersion string
	HardwareVersion string
}

// SlapInfo represents segmentation info for a single finger
type SlapInfo struct {
	FingerType      int
	FingerPositions [4][2]int // 4 corner points (x,y)
	ImageQuality    int
	Rotation        int
}

func sdkLoadLibrary(libPath string) int {
	cPath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cPath))
	return int(C.loadSDK(cPath))
}

func sdkInit(configDir *string, options int) (numDevices int, rc int) {
	var cConfigDir *C.char
	if configDir != nil {
		cConfigDir = C.CString(*configDir)
		defer C.free(unsafe.Pointer(cConfigDir))
	}
	var num C.int
	rc = int(C.sdk_init(cConfigDir, C.int(options), &num))
	return int(num), rc
}

func sdkExit() int {
	return int(C.sdk_exit())
}

func sdkStopHotPlugging() int {
	return int(C.sdk_stop_hot_plugging())
}

func sdkGetDeviceCount() (numDevices int, rc int) {
	var num C.int
	rc = int(C.sdk_get_device_count(&num))
	return int(num), rc
}

func sdkInitDevice(deviceIndex int) (deviceHandle int, rc int) {
	var handle C.int
	rc = int(C.sdk_init_device(C.int(deviceIndex), &handle))
	return int(handle), rc
}

func sdkExitDevice(deviceHandle int) int {
	return int(C.sdk_exit_device(C.int(deviceHandle)))
}

func sdkGetDeviceInfo(deviceHandle int) (info DeviceInfo, rc int) {
	var cInfo C.RSDeviceInfo
	rc = int(C.sdk_get_device_info(C.int(deviceHandle), &cInfo))
	if rc == 0 {
		info.DeviceType = int(cInfo.deviceType)
		info.ProductName = C.GoString(&cInfo.productName[0])
		info.DeviceID = C.GoString(&cInfo.deviceID[0])
		info.FirmwareVersion = C.GoString(&cInfo.firmwareVersion[0])
		info.HardwareVersion = C.GoString(&cInfo.hardwareVersion[0])
	}
	return info, rc
}

func sdkSetCaptureMode(deviceHandle, captureMode, autoSensitivity, withLED int) int {
	return int(C.sdk_set_capture_mode(C.int(deviceHandle), C.int(captureMode),
		C.int(autoSensitivity), C.int(withLED)))
}

func sdkSetAutoCalibrate(deviceHandle, enable int) {
	C.sdk_set_auto_calibrate(C.int(deviceHandle), C.int(enable))
}

func sdkSetPreProcessing(deviceHandle, mode int) int {
	return int(C.sdk_set_pre_processing(C.int(deviceHandle), C.int(mode)))
}

func sdkStartHotPlugging() int {
	return int(C.sdk_start_hot_plugging())
}

func sdkRegisterHotPlugCallbackGo() int {
	return int(C.sdk_register_hot_plug_callback_go())
}

func sdkSetModeLED(deviceHandle, ledIndex, isOn int) int {
	return int(C.sdk_set_mode_led(C.int(deviceHandle), C.int(ledIndex), C.int(isOn)))
}

func sdkSetFingerLED(deviceHandle, fingerIndex, ledColor int) int {
	return int(C.sdk_set_finger_led(C.int(deviceHandle), C.int(fingerIndex), C.int(ledColor)))
}

func sdkAbortCapture(deviceHandle int) int {
	return int(C.sdk_abort_capture(C.int(deviceHandle)))
}

func sdkTakeImageDataEx(deviceHandle, timeout, fingerIndex, withLED int) (imageData unsafe.Pointer, width, height, rc int) {
	var cImageData *C.uchar
	var cWidth, cHeight C.int
	rc = int(C.sdk_take_image_data_ex(C.int(deviceHandle), C.int(timeout),
		C.int(fingerIndex), C.int(withLED),
		&cImageData, &cWidth, &cHeight))
	return unsafe.Pointer(cImageData), int(cWidth), int(cHeight), rc
}

func sdkTakeImageData(deviceHandle, timeout int) (imageData unsafe.Pointer, width, height, rc int) {
	var cImageData *C.uchar
	var cWidth, cHeight C.int
	rc = int(C.sdk_take_image_data(C.int(deviceHandle), C.int(timeout),
		&cImageData, &cWidth, &cHeight))
	return unsafe.Pointer(cImageData), int(cWidth), int(cHeight), rc
}

func sdkGetQualityScore(imageData unsafe.Pointer, width, height int) (nistQuality, rc int) {
	var quality C.int
	rc = int(C.sdk_get_quality_score((*C.uchar)(imageData), C.int(width), C.int(height), &quality))
	return int(quality), rc
}

func sdkFreeImageData(imageData unsafe.Pointer) {
	C.sdk_free_image_data((*C.uchar)(imageData))
}

func sdkBeep(deviceHandle, beepPattern int) {
	C.sdk_beep(C.int(deviceHandle), C.int(beepPattern))
}

func sdkGetErrString(errorCode int) string {
	var buf [256]C.char
	C.sdk_get_err_string(C.int(errorCode), &buf[0])
	return C.GoString(&buf[0])
}

func sdkTakeImageDataSegment(deviceHandle, timeout, slapType int) (
	imageData unsafe.Pointer, imageWidth, imageHeight, captureResult, numOfFinger, rc int,
	slapInfos []SlapInfo, fingerImages []unsafe.Pointer, fingerWidths, fingerHeights []int,
) {
	var cImageData *C.uchar
	var cImageWidth, cImageHeight, cCaptureResult, cNumOfFinger C.int
	var cSlapInfo *C.RSSlapInfo
	var cFingerImageData **C.uchar
	var cFingerImageWidth, cFingerImageHeight *C.int

	rc = int(C.sdk_take_image_data_segment(
		C.int(deviceHandle), C.int(timeout),
		&cImageData, &cImageWidth, &cImageHeight,
		&cCaptureResult, C.int(slapType), &cNumOfFinger,
		&cSlapInfo,
		&cFingerImageData,
		&cFingerImageWidth, &cFingerImageHeight,
	))

	imageData = unsafe.Pointer(cImageData)
	imageWidth = int(cImageWidth)
	imageHeight = int(cImageHeight)
	captureResult = int(cCaptureResult)
	numOfFinger = int(cNumOfFinger)

	// Convert slap info array
	if numOfFinger > 0 && cSlapInfo != nil {
		slapInfos = make([]SlapInfo, numOfFinger)
		for i := 0; i < numOfFinger; i++ {
			// Access array elements via pointer arithmetic
			info := (*C.RSSlapInfo)(unsafe.Pointer(uintptr(unsafe.Pointer(cSlapInfo)) + uintptr(i)*unsafe.Sizeof(*cSlapInfo)))
			slapInfos[i].FingerType = int(info.fingerType)
			slapInfos[i].ImageQuality = int(info.imageQuality)
			slapInfos[i].Rotation = int(info.rotation)
			for j := 0; j < 4; j++ {
				slapInfos[i].FingerPositions[j][0] = int(info.fingerPosition[j].x)
				slapInfos[i].FingerPositions[j][1] = int(info.fingerPosition[j].y)
			}
		}

		// Convert finger image arrays
		fingerImages = make([]unsafe.Pointer, numOfFinger)
		fingerWidths = make([]int, numOfFinger)
		fingerHeights = make([]int, numOfFinger)
		for i := 0; i < numOfFinger; i++ {
			fingerImages[i] = unsafe.Pointer(C.get_finger_image(cFingerImageData, C.int(i)))
			fingerWidths[i] = int(C.get_finger_dim(cFingerImageWidth, C.int(i)))
			fingerHeights[i] = int(C.get_finger_dim(cFingerImageHeight, C.int(i)))
		}
	}

	return
}

// ──────────────────────────────────────────────────────────────────────────────
// C callback — called by SDK on hot-plug events.
// ──────────────────────────────────────────────────────────────────────────────

//export goOnHotPlug
func goOnHotPlug(deviceId C.int, isConnected C.int) {
	handleHotPlugEvent(int(deviceId), int(isConnected))
}
