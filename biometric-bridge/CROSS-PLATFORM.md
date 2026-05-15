# Cross-Platform Support: Windows + Linux

This document describes the cross-platform architecture for RealScan G-10 biometric bridge driver.

## Architecture Overview

The bridge now supports both Linux and Windows through a platform-abstraction layer:

```
┌─────────────────────────────────────────────────────────────┐
│ driver.go (platform-independent business logic)             │
│ - Device management, scanning, enrollment                    │
│ - Zero C references, pure Go                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────┴───────────────────┐
        ↓                                       ↓
┌──────────────────┐                 ┌──────────────────┐
│ dynload_linux.go │                 │ dynload_windows.go│
│ //go:build linux │                 │ //go:build windows│
├──────────────────┤                 ├──────────────────┤
│ - dlopen/dlsym   │                 │ - LoadLibrary    │
│ - C preamble     │                 │ - GetProcAddress │
│ - Go wrappers    │                 │ - __stdcall      │
└──────────────────┘                 └──────────────────┘
        ↓                                       ↓
┌──────────────────┐                 ┌──────────────────┐
│ libRS_SDK.so     │                 │ RS_SDK.dll       │
│ (v2.2.0.2470)    │                 │ (v2.2.0.2311)    │
└──────────────────┘                 └──────────────────┘
```

### Key Design Decisions

1. **Platform files expose identical Go function signatures**: Each platform file (`dynload_linux.go`, `dynload_windows.go`) exports the same set of Go wrapper functions with identical signatures.

2. **All CGo isolated to platform files**: Only platform-specific files have `import "C"` and C preambles. The main `driver.go` has zero C references.

3. **Constants duplicated in platform files**: Instead of sharing constants via a common file, each platform file defines its own constants to avoid CGo cross-file symbol issues.

4. **Dynamic loading**: Both platforms use dynamic library loading (dlopen on Linux, LoadLibrary on Windows) to avoid static linking and allow flexible SDK paths.

5. **SDK version tolerance**: Linux uses v2.2.0.2470, Windows uses v2.2.0.2311 (159 builds older). The API is compatible across these versions.

## Platform-Specific Details

### Linux (dynload_linux.go)

- **Build tag**: `//go:build linux`
- **SDK loading**: `dlopen()` with `RTLD_LAZY | RTLD_LOCAL`
- **Function loading**: `dlsym()` to get function pointers
- **Calling convention**: Standard C calling convention
- **SDK path**: `/usr/local/lib/libRS_SDK.so` or via `LD_LIBRARY_PATH`
- **SDK cleanup**: `RS_ExitSDK()` exported as mangled symbol `_Z10RS_ExitSDKv`

### Windows (dynload_windows.go)

- **Build tag**: `//go:build windows`
- **SDK loading**: `LoadLibraryA()` with standard Windows search path
- **Function loading**: `GetProcAddress()` to get function pointers
- **Calling convention**: `__stdcall` for all SDK functions
- **SDK path**: `RS_SDK.dll` in same directory or PATH
- **SDK cleanup**: `RS_ExitSDK()` NOT exported by DLL - cleanup is automatic

## Go Wrapper Functions

Both platform files implement these Go wrapper functions:

```go
// Library management
func sdkLoadLibrary(path string) error
func sdkInit() int
func sdkExit() int

// Device management
func sdkInitDevice(handle int, deviceType int) int
func sdkExitDevice(handle int) int
func sdkGetDeviceCount() (int, int)
func sdkGetDeviceInfo(index int, info *DeviceInfo) int

// Configuration
func sdkSetCaptureMode(handle int, mode int) int
func sdkSetAutoCalibrate(handle int, sensitivity int) int
func sdkSetPreProcessing(handle int, enable int) int

// Hot-plug support
func sdkStartHotPlugging() int
func sdkRegisterHotPlugCallbackGo() int

// LED control
func sdkSetModeLED(handle int, mode int, color int) int
func sdkSetFingerLED(handle int, finger int, color int) int

// Image capture
func sdkAbortCapture(handle int) int
func sdkTakeImageDataEx(handle int, timeout int, fingerIndex int, nCount int) (unsafe.Pointer, int, int, int)
func sdkTakeImageData(handle int, timeout int) (unsafe.Pointer, int, int, int)
func sdkTakeImageDataSegment(handle int, timeout int, slapType int) (imageData unsafe.Pointer, imageWidth int, imageHeight int, captureResult int, numOfFinger int, rc int, slapInfos []SlapInfo, fingerImages []unsafe.Pointer, fingerWidths []int, fingerHeights []int)

// Quality and utilities
func sdkGetQualityScore(imageData unsafe.Pointer, width int, height int) int
func sdkFreeImageData(imageData unsafe.Pointer)
func sdkBeep(handle int, beepPattern int) int
func sdkGetErrString(code int) string
```

## Building

### Linux

```bash
# Standard build
go build -tags realscan -o bridge-realscan ./cmd/bridge

# Or use Makefile
make build-linux
```

### Windows (Cross-compilation from Linux)

```bash
# Prerequisites: mingw-w64 toolchain
sudo dnf install mingw64-gcc mingw64-winpthreads-static  # Fedora
sudo apt install gcc-mingw-w64-x86-64                     # Ubuntu

# Build
./build-windows-cross.sh

# Or use Makefile
make build-windows-cross
```

### Windows (Native build)

```cmd
REM Prerequisites: Go 1.24.7+, MinGW-w64

REM Build
build-windows.bat
```

## Deployment

### Linux

```bash
# Copy bridge and SDK
sudo cp bridge-realscan /usr/local/bin/
sudo cp ../RealScan_SDK_for_Linux_v2.2.0.2470/lib/libRS_SDK.so /usr/local/lib/

# Or use startup script
./start-realscan.sh
```

### Windows

```bash
# Create deployment package (on Linux)
./create-windows-deployment.sh 1.0.0

# Transfer to Windows and extract
# Then install as service or run as desktop app
# See WINDOWS-DEPLOYMENT.md for details
```

## Testing

### Linux

```bash
# With device connected
./bridge-realscan

# Check for:
# - "RealScan SDK loaded successfully"
# - Device detection messages
# - WebSocket server startup
```

### Windows

```cmd
REM Copy DLLs first
copy dlls\*.dll .

REM Run bridge
bridge-realscan.exe

REM Check for same messages as Linux
```

## SDK Compatibility

| Platform | SDK Version    | Build Date | Notes                          |
|----------|---------------|------------|--------------------------------|
| Linux    | v2.2.0.2470   | Recent     | Latest Linux SDK               |
| Windows  | v2.2.0.2311   | Older      | 159 builds older, API-compatible|

**Key differences:**
- Windows DLL does NOT export `RS_ExitSDK()` - cleanup is automatic
- Windows DLL uses `__stdcall` convention vs standard C on Linux
- Function exports are UNMANGLED on Windows (unlike Linux's mangled C++ symbols)

## File Structure

```
internal/driver/realscan/
├── driver.go              # Platform-independent logic (NO C code)
├── dynload_linux.go       # Linux dlopen implementation
├── dynload_windows.go     # Windows LoadLibrary implementation
└── [types.go removed]     # Constants now in platform files

internal/tray/
├── ui.go                  # Cross-platform Fyne UI
├── controller.go          # Bridge lifecycle management
├── event_subscriber.go    # Event handling
├── logger.go              # Log capture
├── singleinstance.go      # Cross-platform instance locking
├── observer.go            # Observer pattern
├── ringbuffer.go          # Circular buffer
└── theme.go               # Custom theme

Build scripts:
├── build-windows-cross.sh          # Cross-compile bridge from Linux
├── build-windows.bat               # Native Windows bridge build
├── build-tray-windows-cross.sh     # Cross-compile tray from Linux (slow)
├── build-tray-windows.bat          # Native Windows tray build (recommended)
├── create-windows-deployment.sh    # Create deployment package
├── Makefile                        # Build automation
└── start-realscan.sh              # Linux startup script

Documentation:
├── WINDOWS-DEPLOYMENT.md       # Windows bridge deployment guide
├── WINDOWS-TRAY.md            # Windows tray app guide
└── CROSS-PLATFORM.md          # This file
```

## System Tray Application

### Cross-Platform Architecture

The tray app is built with **Fyne v2.5.5** - a cross-platform Go GUI framework:
- Same Go codebase for Linux and Windows
- Platform-native rendering (GTK on Linux, Windows API on Windows)
- System tray integration via `fyne.io/systray`
- ~1,800 lines of pure Go code (no platform-specific code needed)

### Features

- Real-time bridge status monitoring
- JWT token management (view/copy)
- Event log viewer with filtering (Scan/Enrollment/Connection/System/Error)
- Bridge start/stop controls
- Configuration file editor
- Single-instance enforcement (TCP port 17070)
- Auto-start support

### Building Tray App

#### Linux
```bash
go build -o bridge-tray ./cmd/tray
# Output: ~10MB binary
```

#### Windows (Native - Recommended)
```cmd
REM Prerequisites: Go + MinGW-w64/TDM-GCC
build-tray-windows.bat
REM Output: ~25-30MB binary
REM Build time: 2-5 minutes (first time)
```

#### Windows (Cross-Compile - Not Recommended)
```bash
# Very slow due to Fyne CGo dependencies (15-30+ minutes)
./build-tray-windows-cross.sh
# Only use if native Windows build is not possible
```

### Deployment

**Linux:**
```bash
# System-wide
sudo cp bridge-tray /usr/local/bin/

# Auto-start (systemd user service or XDG autostart)
```

**Windows:**
```
1. Copy bridge-tray.exe to installation directory
2. Auto-start via Startup folder:
   - Press Win+R
   - Type: shell:startup
   - Create shortcut to bridge-tray.exe
```

See `WINDOWS-TRAY.md` for detailed Windows tray documentation.

## Troubleshooting

### Linux

**SDK won't load:**
- Check `LD_LIBRARY_PATH` includes SDK directory
- Verify `libRS_SDK.so` exists and is readable
- Check dependencies: `ldd libRS_SDK.so`

**Device not detected:**
- Check USB permissions: add user to `plugdev` group
- Verify device shows in `lsusb`
- Check kernel logs: `dmesg | grep -i usb`

### Windows

**SDK won't load:**
- Ensure all DLLs are in same directory as executable or in PATH
- Install Visual C++ Redistributable 2015-2022 (x64)
- Check DLL dependencies with Dependency Walker

**Device not detected:**
- Install RealScan USB driver from SDK's `Driver\` directory
- Check Device Manager for driver issues
- Try running as Administrator

## Performance Notes

- **Memory**: Both platforms use same image buffer management
- **Threading**: SDK callbacks run on SDK threads, events forwarded to Go via channels
- **Hot-plug**: Both platforms support hot-plug detection via SDK callbacks
- **Image formats**: Both return raw 8-bit grayscale WSQ-compatible images

## Future Enhancements

- [ ] macOS support (if SDK becomes available)
- [ ] ARM64 Windows support (when SDK available)
- [x] System tray app for Windows (completed - uses Fyne)
- [ ] Windows service auto-start configuration
- [ ] Installer packages (MSI for Windows, DEB/RPM for Linux)
- [ ] Tray WebSocket connection to bridge (real-time status)
- [ ] Tray settings dialog (edit config in UI)

## References

- RealScan SDK Documentation: `Document/` in SDK directories
- Windows bridge deployment: `WINDOWS-DEPLOYMENT.md`
- Windows tray app: `WINDOWS-TRAY.md`
- Build system: `Makefile`
