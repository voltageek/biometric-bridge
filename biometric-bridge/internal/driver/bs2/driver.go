//go:build bs2

// Package bs2 implements the Driver interface using the BioStar 2 Device SDK
// via CGo. It communicates directly with Suprema fingerprint readers over TCP.
//
// Build with: CGO_ENABLED=1 go build -tags bs2 ./cmd/bridge
package bs2

/*
#cgo LDFLAGS: -ldl

#include <stdlib.h>
#include <stdint.h>
#include <dlfcn.h>
#include <string.h>

// ──────────────────────────────────────────────────────────────────────────────
// Minimal struct and type declarations matching the BS2 SDK headers.
// We define only what we need to avoid pulling in the full SDK header tree
// which has cross-include issues.
// ──────────────────────────────────────────────────────────────────────────────

typedef uint32_t BS2_DEVICE_ID;
typedef uint16_t BS2_PORT;
typedef uint16_t BS2_DEVICE_TYPE;
typedef uint16_t BS2_EVENT_CODE;
typedef uint32_t BS2_EVENT_ID;
typedef uint32_t BS2_TIMESTAMP;
typedef uint8_t  BS2_CONNECTION_MODE;
typedef uint8_t  BS2_RS485_MODE;
typedef char     BS2_USER_ID[32];

#pragma pack(push, 1)

// BS2SimpleDeviceInfo — subset of fields we care about
typedef struct {
    BS2_DEVICE_ID       id;
    BS2_DEVICE_TYPE     type;
    BS2_CONNECTION_MODE connectionMode;
    uint32_t            ipv4Address;
    BS2_PORT            port;
    uint32_t            maxNumOfUser;
    uint8_t             userNameSupported;
    uint8_t             userPhotoSupported;
    uint8_t             pinSupported;
    uint8_t             cardSupported;
    uint8_t             fingerSupported;
    uint8_t             faceSupported;
    uint8_t             wlanSupported;
    uint8_t             tnaSupported;
    uint8_t             triggerActionSupported;
    uint8_t             wiegandSupported;
    uint8_t             imageLogSupported;
    uint8_t             dnsSupported;
    uint8_t             jobCodeSupported;
    uint8_t             wiegandMultiSupported;
    BS2_RS485_MODE      rs485Mode;
    uint8_t             sslSupported;
    uint8_t             rootCertExist;
    uint8_t             dualIDSupported;
    uint8_t             useAlphanumericID;
    uint32_t            connectedIP;
    uint8_t             phraseSupported;
    uint8_t             card1xSupported;
    uint8_t             systemExtSupported;
    uint8_t             voipSupported;
    uint8_t             rs485ExSupported;
    uint8_t             cardExSupported;
} BS2SimpleDeviceInfo;

// BS2FactoryConfig — for model name and firmware version
typedef struct {
    uint8_t major;
    uint8_t minor;
    uint8_t ext;
    uint8_t reserved;
} BS2Version;

typedef struct {
    BS2_DEVICE_ID deviceID;
    uint8_t       macAddr[6];
    uint8_t       reserved[2];
    char          modelName[32];
    BS2Version    boardVer;
    BS2Version    kernelVer;
    BS2Version    bscoreVer;
    BS2Version    firmwareVer;
    char          kernelRev[32];
    char          bscoreRev[32];
    char          firmwareRev[32];
    uint8_t       reserved2[32];
} BS2FactoryConfig;

// BS2Fingerprint — fingerprint template storage
#define BS2_FINGER_TEMPLATE_SIZE  384
#define BS2_TEMPLATE_PER_FINGER   2

typedef struct {
    uint8_t index;
    uint8_t flag;
    uint8_t reserved[2];
    uint8_t data[BS2_TEMPLATE_PER_FINGER][BS2_FINGER_TEMPLATE_SIZE]; // [2][384]
} BS2Fingerprint;

// BS2User — user header
typedef struct {
    BS2_USER_ID    userID;          // char[32]
    uint8_t        formatVersion;
    uint8_t        flag;
    uint16_t       version;
    uint8_t        numCards;
    uint8_t        numFingers;
    uint8_t        numFaces;
    uint8_t        infoMask;
    uint32_t       authGroupID;
    uint32_t       faceChecksum;
} BS2User;

// BS2UserSetting
typedef struct {
    uint32_t startTime;
    uint32_t endTime;
    uint8_t  fingerAuthMode;
    uint8_t  cardAuthMode;
    uint8_t  idAuthMode;
    uint8_t  securityLevel;
} BS2UserSetting;

// BS2UserPhoto
#define BS2_USER_PHOTO_SIZE 16384

typedef struct {
    uint32_t size;
    uint8_t  data[BS2_USER_PHOTO_SIZE];
} BS2UserPhoto;

// BS2CSNCard
#define BS2_CARD_DATA_SIZE 32

typedef struct {
    uint8_t type;
    uint8_t size;
    uint8_t data[BS2_CARD_DATA_SIZE];
} BS2CSNCard;

// BS2Face
#define BS2_FACE_TEMPLATE_LENGTH 3008
#define BS2_TEMPLATE_PER_FACE    30
#define BS2_FACE_IMAGE_SIZE      16384

typedef struct {
    uint8_t  faceIndex;
    uint8_t  numOfTemplate;
    uint8_t  flag;
    uint8_t  reserved;
    uint16_t imageLen;
    uint8_t  reserved2[2];
    uint8_t  imageData[BS2_FACE_IMAGE_SIZE];
    uint8_t  templateData[BS2_TEMPLATE_PER_FACE][BS2_FACE_TEMPLATE_LENGTH];
} BS2Face;

// BS2UserBlob
#define BS2_MAX_NUM_OF_ACCESS_GROUP_PER_USER 16

typedef struct {
    BS2User         user;
    BS2UserSetting  setting;
    uint8_t         user_name[192];
    BS2UserPhoto    user_photo;
    uint8_t         pin[32];
    BS2CSNCard*     cardObjs;
    BS2Fingerprint* fingerObjs;
    BS2Face*        faceObjs;
    uint32_t        accessGroupId[BS2_MAX_NUM_OF_ACCESS_GROUP_PER_USER];
} BS2UserBlob;

// BS2Event — event from device
typedef struct {
    BS2_EVENT_ID  id;
    BS2_TIMESTAMP dateTime;
    BS2_DEVICE_ID deviceID;
    union {
        BS2_USER_ID userID;
        uint32_t    uid;
        uint8_t     raw[32];
    };
    union {
        BS2_EVENT_CODE code;
        struct {
            uint8_t subCode;
            uint8_t mainCode;
        };
    };
    uint8_t  param;
    uint8_t  image;
} BS2Event;

#pragma pack(pop)

// ──────────────────────────────────────────────────────────────────────────────
// Function pointer typedefs for dynamically loaded SDK functions
// ──────────────────────────────────────────────────────────────────────────────

typedef void* (*fn_AllocateContext)();
typedef void  (*fn_ReleaseContext)(void* ctx);
typedef int   (*fn_Initialize)(void* ctx);
typedef int   (*fn_ConnectDeviceViaIP)(void* ctx, const char* addr, BS2_PORT port, BS2_DEVICE_ID* devId);
typedef int   (*fn_DisconnectDevice)(void* ctx, BS2_DEVICE_ID devId);
typedef int   (*fn_GetDeviceInfo)(void* ctx, BS2_DEVICE_ID devId, BS2SimpleDeviceInfo* info);
typedef int   (*fn_GetFactoryConfig)(void* ctx, BS2_DEVICE_ID devId, BS2FactoryConfig* config);
typedef int   (*fn_ScanFingerprintEx)(void* ctx, BS2_DEVICE_ID devId, BS2Fingerprint* finger,
                                      uint32_t templateIndex, uint32_t quality, uint8_t templateFormat,
                                      uint32_t* outquality, void* readyToScan);
typedef int   (*fn_EnrollUser)(void* ctx, BS2_DEVICE_ID devId, BS2UserBlob* blob,
                               uint32_t userCount, uint8_t overwrite);
typedef int   (*fn_StartMonitoringLog)(void* ctx, BS2_DEVICE_ID devId, void* callback);
typedef int   (*fn_StopMonitoringLog)(void* ctx, BS2_DEVICE_ID devId);
typedef int   (*fn_SetDeviceEventListener)(void* ctx, void* onFound, void* onAccepted,
                                           void* onConnected, void* onDisconnected);

// ──────────────────────────────────────────────────────────────────────────────
// SDK handle and loaded function pointers
// ──────────────────────────────────────────────────────────────────────────────

static void* sdk_handle = NULL;

static fn_AllocateContext        p_AllocateContext;
static fn_ReleaseContext         p_ReleaseContext;
static fn_Initialize             p_Initialize;
static fn_ConnectDeviceViaIP     p_ConnectDeviceViaIP;
static fn_DisconnectDevice       p_DisconnectDevice;
static fn_GetDeviceInfo          p_GetDeviceInfo;
static fn_GetFactoryConfig       p_GetFactoryConfig;
static fn_ScanFingerprintEx      p_ScanFingerprintEx;
static fn_EnrollUser             p_EnrollUser;
static fn_StartMonitoringLog     p_StartMonitoringLog;
static fn_StopMonitoringLog      p_StopMonitoringLog;
static fn_SetDeviceEventListener p_SetDeviceEventListener;

// loadSDK dynamically loads the BS2 shared library and resolves all symbols.
static int loadSDK(const char* libPath) {
    sdk_handle = dlopen(libPath, RTLD_NOW);
    if (!sdk_handle) return -1;

    p_AllocateContext    = (fn_AllocateContext)dlsym(sdk_handle, "BS2_AllocateContext");
    p_ReleaseContext     = (fn_ReleaseContext)dlsym(sdk_handle, "BS2_ReleaseContext");
    p_Initialize         = (fn_Initialize)dlsym(sdk_handle, "BS2_Initialize");
    p_ConnectDeviceViaIP = (fn_ConnectDeviceViaIP)dlsym(sdk_handle, "BS2_ConnectDeviceViaIP");
    p_DisconnectDevice   = (fn_DisconnectDevice)dlsym(sdk_handle, "BS2_DisconnectDevice");
    p_GetDeviceInfo      = (fn_GetDeviceInfo)dlsym(sdk_handle, "BS2_GetDeviceInfo");
    p_GetFactoryConfig   = (fn_GetFactoryConfig)dlsym(sdk_handle, "BS2_GetFactoryConfig");
    p_ScanFingerprintEx  = (fn_ScanFingerprintEx)dlsym(sdk_handle, "BS2_ScanFingerprintEx");
    p_EnrollUser         = (fn_EnrollUser)dlsym(sdk_handle, "BS2_EnrollUser");
    p_StartMonitoringLog = (fn_StartMonitoringLog)dlsym(sdk_handle, "BS2_StartMonitoringLog");
    p_StopMonitoringLog  = (fn_StopMonitoringLog)dlsym(sdk_handle, "BS2_StopMonitoringLog");
    p_SetDeviceEventListener = (fn_SetDeviceEventListener)dlsym(sdk_handle, "BS2_SetDeviceEventListener");

    if (!p_AllocateContext || !p_ReleaseContext || !p_Initialize ||
        !p_ConnectDeviceViaIP || !p_DisconnectDevice || !p_GetDeviceInfo ||
        !p_GetFactoryConfig || !p_ScanFingerprintEx || !p_EnrollUser ||
        !p_StartMonitoringLog || !p_StopMonitoringLog || !p_SetDeviceEventListener) {
        dlclose(sdk_handle);
        sdk_handle = NULL;
        return -2;
    }

    return 0;
}

// SDK wrapper functions called from Go
static void* sdk_allocate_context() { return p_AllocateContext(); }
static void  sdk_release_context(void* ctx) { p_ReleaseContext(ctx); }
static int   sdk_initialize(void* ctx) { return p_Initialize(ctx); }

static int sdk_connect_device(void* ctx, const char* addr, uint16_t port, uint32_t* devId) {
    return p_ConnectDeviceViaIP(ctx, addr, (BS2_PORT)port, (BS2_DEVICE_ID*)devId);
}

static int sdk_disconnect_device(void* ctx, uint32_t devId) {
    return p_DisconnectDevice(ctx, (BS2_DEVICE_ID)devId);
}

static int sdk_get_device_info(void* ctx, uint32_t devId, BS2SimpleDeviceInfo* info) {
    return p_GetDeviceInfo(ctx, (BS2_DEVICE_ID)devId, info);
}

static int sdk_get_factory_config(void* ctx, uint32_t devId, BS2FactoryConfig* config) {
    return p_GetFactoryConfig(ctx, (BS2_DEVICE_ID)devId, config);
}

static int sdk_scan_fingerprint_ex(void* ctx, uint32_t devId, BS2Fingerprint* finger,
                                    uint32_t templateIndex, uint32_t quality,
                                    uint8_t templateFormat, uint32_t* outQuality) {
    return p_ScanFingerprintEx(ctx, (BS2_DEVICE_ID)devId, finger,
                               templateIndex, quality, templateFormat, outQuality, NULL);
}

static int sdk_enroll_user(void* ctx, uint32_t devId, BS2UserBlob* blob,
                           uint32_t count, uint8_t overwrite) {
    return p_EnrollUser(ctx, (BS2_DEVICE_ID)devId, blob, count, overwrite);
}

static int sdk_start_monitoring_log(void* ctx, uint32_t devId, void* callback) {
    return p_StartMonitoringLog(ctx, (BS2_DEVICE_ID)devId, callback);
}

static int sdk_stop_monitoring_log(void* ctx, uint32_t devId) {
    return p_StopMonitoringLog(ctx, (BS2_DEVICE_ID)devId);
}

static int sdk_set_device_event_listener(void* ctx, void* onFound, void* onAccepted,
                                          void* onConnected, void* onDisconnected) {
    return p_SetDeviceEventListener(ctx, onFound, onAccepted, onConnected, onDisconnected);
}

// ──────────────────────────────────────────────────────────────────────────────
// C callback forwarders — these run on C threads and push events to Go via
// a minimal channel-safe path.
// ──────────────────────────────────────────────────────────────────────────────

// Forward declarations of Go functions (defined with //export)
extern void goOnLogReceived(uint32_t deviceId, uint16_t eventCode, char* userID);
extern void goOnDeviceDisconnected(uint32_t deviceId);

// C callback that BS2 SDK calls on log events
static void onLogReceivedCallback(uint32_t deviceId, const BS2Event* event) {
    if (event == NULL) return;
    // Extract userID as a null-terminated string (it's a char[32])
    char uid[33];
    memcpy(uid, event->userID, 32);
    uid[32] = '\0';
    goOnLogReceived(deviceId, event->code, uid);
}

// C callback that BS2 SDK calls on device disconnection
static void onDeviceDisconnectedCallback(uint32_t deviceId) {
    goOnDeviceDisconnected(deviceId);
}

// Convenience wrapper to start monitoring with our callback
static int sdk_start_monitoring_with_callback(void* ctx, uint32_t devId) {
    return p_StartMonitoringLog(ctx, (BS2_DEVICE_ID)devId, (void*)onLogReceivedCallback);
}

// Convenience wrapper to set disconnect listener with our callback
static int sdk_set_disconnect_listener(void* ctx) {
    return p_SetDeviceEventListener(ctx, NULL, NULL, NULL, (void*)onDeviceDisconnectedCallback);
}
*/
import "C"

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"biometric-bridge/internal/driver"
)

const (
	bs2FingerTemplateSize    = 384
	bs2TemplatePerFinger     = 2
	bs2TemplateFormatSuprema = 0x00
)

// ReconnectConfig holds parameters for automatic reconnection with exponential
// backoff (FR-010, US6).
type ReconnectConfig struct {
	Base     time.Duration             // Initial backoff delay (default 1s)
	Cap      time.Duration             // Maximum backoff delay (default 120s)
	Registry driver.DeviceStateUpdater // Registry to update device state
}

// BS2Driver implements the driver.Driver interface using the BioStar 2 Device SDK.
type BS2Driver struct {
	mu      sync.RWMutex
	ctx     unsafe.Pointer        // BS2 SDK context
	devices map[string]*bs2Device // keyed by device name
	eventCh chan driver.Event
	closed  bool

	// Reconnection
	reconnCfg    *ReconnectConfig
	reconnCancel map[string]context.CancelFunc // per-device cancel functions
}

type bs2Device struct {
	name  string
	addr  string // Original address for reconnection
	port  uint16 // Original port for reconnection
	sdkID uint32
	info  driver.DeviceInfo
}

// New creates a new BS2 driver. The libPath is the path to the BS2 shared library.
func New(libPath string) (*BS2Driver, error) {
	cPath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cPath))

	rc := C.loadSDK(cPath)
	if rc != 0 {
		return nil, fmt.Errorf("failed to load BS2 SDK from %s (error: %d)", libPath, rc)
	}

	sdkCtx := C.sdk_allocate_context()
	if sdkCtx == nil {
		return nil, fmt.Errorf("BS2_AllocateContext returned nil")
	}

	rc2 := C.sdk_initialize(sdkCtx)
	if rc2 != 0 {
		C.sdk_release_context(sdkCtx)
		return nil, fmt.Errorf("BS2_Initialize failed: error code %d", rc2)
	}

	d := &BS2Driver{
		ctx:          sdkCtx,
		devices:      make(map[string]*bs2Device),
		eventCh:      make(chan driver.Event, 256),
		reconnCancel: make(map[string]context.CancelFunc),
	}

	// Register the global driver instance for C callbacks
	globalDriverMu.Lock()
	globalDriver = d
	globalDriverMu.Unlock()

	// Set device event listener for disconnect detection
	C.sdk_set_disconnect_listener(sdkCtx)

	return d, nil
}

// Connect connects to all configured devices. Returns an error if any device
// is unreachable (fail-fast, FR-011).
func (d *BS2Driver) Connect(ctx context.Context, devices []driver.DeviceConfig) error {
	for _, dc := range devices {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		cAddr := C.CString(dc.Addr)
		defer C.free(unsafe.Pointer(cAddr))

		var sdkID C.uint32_t
		rc := C.sdk_connect_device(d.ctx, cAddr, C.uint16_t(dc.Port), &sdkID)
		if rc != 0 {
			return fmt.Errorf("failed to connect to device %q (%s:%d): SDK error %d",
				dc.Name, dc.Addr, dc.Port, rc)
		}

		// Get device info
		var devInfo C.BS2SimpleDeviceInfo
		rc = C.sdk_get_device_info(d.ctx, sdkID, &devInfo)
		if rc != 0 {
			return fmt.Errorf("failed to get device info for %q: SDK error %d", dc.Name, rc)
		}

		// Get factory config for model name and firmware version
		var factoryConfig C.BS2FactoryConfig
		rc = C.sdk_get_factory_config(d.ctx, sdkID, &factoryConfig)
		if rc != 0 {
			return fmt.Errorf("failed to get factory config for %q: SDK error %d", dc.Name, rc)
		}

		modelName := C.GoString(&factoryConfig.modelName[0])
		fwVer := fmt.Sprintf("%d.%d.%d",
			factoryConfig.firmwareVer.major,
			factoryConfig.firmwareVer.minor,
			factoryConfig.firmwareVer.ext,
		)

		dev := &bs2Device{
			name:  dc.Name,
			addr:  dc.Addr,
			port:  uint16(dc.Port),
			sdkID: uint32(sdkID),
			info: driver.DeviceInfo{
				Name:            dc.Name,
				ID:              strconv.FormatUint(uint64(sdkID), 10),
				Model:           modelName,
				FirmwareVersion: fwVer,
				FingerSupported: devInfo.fingerSupported != 0,
			},
		}

		d.mu.Lock()
		d.devices[dc.Name] = dev
		d.mu.Unlock()

		// Start monitoring log events for this device
		rc = C.sdk_start_monitoring_with_callback(d.ctx, sdkID)
		if rc != 0 {
			slog.Warn("failed to start monitoring for device", "name", dc.Name, "error", rc)
		}
	}

	return nil
}

// Scan captures a single fingerprint from the specified device.
// The fingerHint parameter is ignored — BS2 devices do not have finger LEDs.
func (d *BS2Driver) Scan(ctx context.Context, deviceName string, fingerHint driver.FingerPosition) (*driver.ScanResult, error) {
	dev, err := d.getDevice(deviceName)
	if err != nil {
		return nil, err
	}

	var finger C.BS2Fingerprint
	var outQuality C.uint32_t

	// Scan template index 0
	rc := C.sdk_scan_fingerprint_ex(d.ctx, C.uint32_t(dev.sdkID), &finger,
		0, // templateIndex
		0, // quality threshold (0 = accept any)
		C.uint8_t(bs2TemplateFormatSuprema),
		&outQuality,
	)
	if rc != 0 {
		return nil, fmt.Errorf("scan failed on device %q: SDK error %d", deviceName, rc)
	}

	// Extract template data (first template slot)
	templateData := C.GoBytes(unsafe.Pointer(&finger.data[0][0]), C.int(bs2FingerTemplateSize))

	return &driver.ScanResult{
		Template: templateData,
		Quality:  int(outQuality),
	}, nil
}

// SlapScan is not supported by the BS2 driver — BS2 devices do not support
// multi-finger slap capture with segmentation.
func (d *BS2Driver) SlapScan(ctx context.Context, deviceName string, mode driver.CaptureMode) (*driver.SlapScanResult, error) {
	return nil, driver.ErrSlapNotSupported
}

// Enroll performs a two-impression enrollment on the specified device.
// The fingers parameter is ignored — BS2 devices do not have finger LEDs.
func (d *BS2Driver) Enroll(ctx context.Context, deviceName, userID, userName string, fingers []driver.FingerPosition) error {
	dev, err := d.getDevice(deviceName)
	if err != nil {
		return err
	}

	// Scan two templates for the same finger
	var finger C.BS2Fingerprint
	var outQuality C.uint32_t

	// First impression (template index 0)
	rc := C.sdk_scan_fingerprint_ex(d.ctx, C.uint32_t(dev.sdkID), &finger,
		0, 0, C.uint8_t(bs2TemplateFormatSuprema), &outQuality)
	if rc != 0 {
		return fmt.Errorf("enrollment scan 1 failed on device %q: SDK error %d", deviceName, rc)
	}

	// Second impression (template index 1)
	rc = C.sdk_scan_fingerprint_ex(d.ctx, C.uint32_t(dev.sdkID), &finger,
		1, 0, C.uint8_t(bs2TemplateFormatSuprema), &outQuality)
	if rc != 0 {
		return fmt.Errorf("enrollment scan 2 failed on device %q: SDK error %d", deviceName, rc)
	}

	// Build BS2UserBlob
	var blob C.BS2UserBlob
	C.memset(unsafe.Pointer(&blob), 0, C.size_t(unsafe.Sizeof(blob)))

	// Set user ID (char[32])
	cUID := C.CString(userID)
	defer C.free(unsafe.Pointer(cUID))
	C.strncpy(&blob.user.userID[0], cUID, 31)

	// Set user name (uint8_t[192])
	cName := C.CString(userName)
	defer C.free(unsafe.Pointer(cName))
	nameLen := len(userName)
	if nameLen > 191 {
		nameLen = 191
	}
	C.memcpy(unsafe.Pointer(&blob.user_name[0]), unsafe.Pointer(cName), C.size_t(nameLen))

	// Set finger data
	blob.user.numFingers = 1
	blob.fingerObjs = &finger

	// Set auth mode to fingerprint only
	blob.setting.fingerAuthMode = 255 // BS2_AUTH_MODE_NONE -> use default

	// Enroll
	rc = C.sdk_enroll_user(d.ctx, C.uint32_t(dev.sdkID), &blob, 1, 1)
	if rc != 0 {
		return fmt.Errorf("enrollment failed on device %q: SDK error %d", deviceName, rc)
	}

	return nil
}

// ListDevices returns metadata for all connected devices.
func (d *BS2Driver) ListDevices() []driver.DeviceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	infos := make([]driver.DeviceInfo, 0, len(d.devices))
	for _, dev := range d.devices {
		infos = append(infos, dev.info)
	}
	return infos
}

// Subscribe returns a channel that receives real-time events from all devices.
func (d *BS2Driver) Subscribe() <-chan driver.Event {
	return d.eventCh
}

// Close disconnects all devices and releases the SDK context.
func (d *BS2Driver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	d.closed = true

	// Cancel all reconnection goroutines
	for name, cancel := range d.reconnCancel {
		cancel()
		delete(d.reconnCancel, name)
	}

	// Stop monitoring and disconnect each device
	for _, dev := range d.devices {
		C.sdk_stop_monitoring_log(d.ctx, C.uint32_t(dev.sdkID))
		C.sdk_disconnect_device(d.ctx, C.uint32_t(dev.sdkID))
	}

	// Release SDK context
	C.sdk_release_context(d.ctx)
	d.ctx = nil

	// Close event channel
	close(d.eventCh)

	// Clear global driver reference
	globalDriverMu.Lock()
	globalDriver = nil
	globalDriverMu.Unlock()

	return nil
}

// SetReconnectConfig configures automatic reconnection behaviour. Must be
// called before Connect. If not called, disconnections emit error events but
// do not trigger automatic reconnection.
func (d *BS2Driver) SetReconnectConfig(base, cap time.Duration, registry driver.DeviceStateUpdater) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reconnCfg = &ReconnectConfig{
		Base:     base,
		Cap:      cap,
		Registry: registry,
	}
}

// startReconnect spawns a goroutine that attempts to reconnect to a device
// using exponential backoff. It emits "reconnecting" events with attempt count
// and wait time, and a "connected" event on success.
func (d *BS2Driver) startReconnect(dev *bs2Device) {
	d.mu.Lock()
	if d.closed || d.reconnCfg == nil {
		d.mu.Unlock()
		return
	}

	// Don't start a second reconnection goroutine for the same device
	if _, exists := d.reconnCancel[dev.name]; exists {
		d.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.reconnCancel[dev.name] = cancel
	cfg := *d.reconnCfg // copy under lock
	d.mu.Unlock()

	// Mark device as disconnected in registry
	if cfg.Registry != nil {
		cfg.Registry.SetState(dev.name, driver.DeviceDisconnected)
	}

	go d.reconnectLoop(ctx, dev, cfg)
}

// reconnectLoop is the per-device reconnection goroutine. It runs until
// the device is reconnected or the context is cancelled (e.g., on Close).
func (d *BS2Driver) reconnectLoop(ctx context.Context, dev *bs2Device, cfg ReconnectConfig) {
	defer func() {
		d.mu.Lock()
		delete(d.reconnCancel, dev.name)
		d.mu.Unlock()
	}()

	for attempt := 1; ; attempt++ {
		// Exponential backoff: base * 2^(attempt-1), capped
		backoff := time.Duration(float64(cfg.Base) * math.Pow(2, float64(attempt-1)))
		if backoff > cfg.Cap {
			backoff = cfg.Cap
		}
		waitSec := int(backoff.Seconds())
		if waitSec < 1 {
			waitSec = 1
		}

		slog.Info("reconnecting to device",
			"name", dev.name,
			"attempt", attempt,
			"waitSeconds", waitSec,
		)

		// Emit reconnecting event
		d.emitEvent(driver.Event{
			Type:        "reconnecting",
			DeviceName:  dev.name,
			Attempt:     attempt,
			WaitSeconds: waitSec,
		})

		// Wait for backoff or cancellation
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		// Attempt reconnection
		cAddr := C.CString(dev.addr)
		var newSDKID C.uint32_t
		rc := C.sdk_connect_device(d.ctx, cAddr, C.uint16_t(dev.port), &newSDKID)
		C.free(unsafe.Pointer(cAddr))

		if rc != 0 {
			slog.Warn("reconnection attempt failed",
				"name", dev.name,
				"attempt", attempt,
				"sdkError", int(rc),
			)
			continue
		}

		// Success — update device SDK ID, restart monitoring
		d.mu.Lock()
		if d.closed {
			d.mu.Unlock()
			// Driver closed while we were reconnecting
			C.sdk_disconnect_device(d.ctx, newSDKID)
			return
		}
		dev.sdkID = uint32(newSDKID)
		d.mu.Unlock()

		// Restart monitoring for this device
		rc = C.sdk_start_monitoring_with_callback(d.ctx, newSDKID)
		if rc != 0 {
			slog.Warn("failed to restart monitoring after reconnect",
				"name", dev.name,
				"sdkError", int(rc),
			)
		}

		// Mark device as idle in registry
		if cfg.Registry != nil {
			cfg.Registry.SetState(dev.name, driver.DeviceIdle)
		}

		slog.Info("device reconnected", "name", dev.name, "attempt", attempt)

		// Emit connected event
		d.emitEvent(driver.Event{
			Type:       "connected",
			DeviceName: dev.name,
		})

		return
	}
}

// emitEvent sends an event to the event channel, dropping it if full.
func (d *BS2Driver) emitEvent(evt driver.Event) {
	select {
	case d.eventCh <- evt:
	default:
		slog.Warn("event channel full, dropping event",
			"type", evt.Type,
			"device", evt.DeviceName,
		)
	}
}

func (d *BS2Driver) getDevice(name string) (*bs2Device, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	dev, ok := d.devices[name]
	if !ok {
		return nil, fmt.Errorf("unknown device: %s", name)
	}
	return dev, nil
}

// deviceNameBySDKID looks up a device name by its SDK-assigned ID.
func (d *BS2Driver) deviceNameBySDKID(sdkID uint32) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, dev := range d.devices {
		if dev.sdkID == sdkID {
			return dev.name
		}
	}
	return ""
}

// ──────────────────────────────────────────────────────────────────────────────
// Global driver instance for C callbacks.
// C callbacks run on C threads and cannot carry Go pointers, so we use a global.
// ──────────────────────────────────────────────────────────────────────────────

var (
	globalDriverMu sync.Mutex
	globalDriver   *BS2Driver
)

//export goOnLogReceived
func goOnLogReceived(deviceId C.uint32_t, eventCode C.uint16_t, userID *C.char) {
	globalDriverMu.Lock()
	d := globalDriver
	globalDriverMu.Unlock()
	if d == nil {
		return
	}

	name := d.deviceNameBySDKID(uint32(deviceId))
	if name == "" {
		return
	}

	d.emitEvent(driver.Event{
		Type:       "scan",
		DeviceName: name,
		UserID:     C.GoString(userID),
		EventCode:  uint32(eventCode),
	})
}

//export goOnDeviceDisconnected
func goOnDeviceDisconnected(deviceId C.uint32_t) {
	globalDriverMu.Lock()
	d := globalDriver
	globalDriverMu.Unlock()
	if d == nil {
		return
	}

	name := d.deviceNameBySDKID(uint32(deviceId))
	if name == "" {
		return
	}

	slog.Warn("device disconnected", "name", name, "sdkId", uint32(deviceId))

	// Emit error event for the disconnection
	d.emitEvent(driver.Event{
		Type:       "error",
		DeviceName: name,
		Message:    "device disconnected",
	})

	// Look up the device and start reconnection
	d.mu.RLock()
	dev, ok := d.devices[name]
	d.mu.RUnlock()
	if ok {
		d.startReconnect(dev)
	}
}
