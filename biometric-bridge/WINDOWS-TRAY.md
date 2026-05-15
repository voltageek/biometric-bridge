# Windows Tray App - Build & Deployment Guide

## Overview

The Kinetic Vault tray application provides a graphical interface for monitoring and managing the RealScan biometric bridge on Windows. It runs in the system tray and provides:

- Real-time bridge status monitoring
- JWT token management (view and copy)
- Event log viewer with filtering
- Bridge start/stop controls
- Configuration file access
- Single-instance enforcement

## Build Approaches

### Option 1: Native Windows Build (RECOMMENDED)

Building natively on Windows is the most reliable approach for the Fyne-based tray app.

**Prerequisites:**
1. Go 1.24.7 or later from https://go.dev/dl/
2. MinGW-w64 or TDM-GCC:
   - MinGW-w64: https://www.mingw-w64.org/downloads/
   - TDM-GCC: https://jmeubank.github.io/tdm-gcc/ (easier installer)
3. Add GCC to PATH

**Build steps:**
```cmd
cd biometric-bridge
build-tray-windows.bat
```

First build takes 2-5 minutes as Go downloads and compiles Fyne dependencies.

**Output:** `bridge-tray.exe` (~25-30MB)

### Option 2: Cross-Compilation from Linux

Cross-compiling Fyne apps with CGo is possible but VERY slow (15-30+ minutes) due to:
- Complex CGo dependencies (OpenGL, Windows GUI libraries)
- Cross-compilation of C/C++ code
- Large dependency tree

**Only recommended if:**
- You have a fast build machine
- You can't build on Windows
- You're willing to wait

**Steps:**
```bash
cd biometric-bridge
./build-tray-windows-cross.sh
# Be patient - this takes 15-30 minutes or more
```

### Option 3: Pre-built Binary

If available, download pre-built `bridge-tray.exe` from releases.

## Installation on Windows

### Manual Installation

1. Extract or copy `bridge-tray.exe` to desired location:
   ```
   C:\Program Files\BiometricBridge\bridge-tray.exe
   ```

2. Create a config file in the same directory:
   ```
   C:\Program Files\BiometricBridge\config.yaml
   ```

3. Run `bridge-tray.exe` - it will start in the system tray

### Auto-Start Setup

To make the tray app start automatically on Windows login:

**Method 1: Startup Folder (Recommended)**

1. Open Run dialog (Win+R)
2. Type: `shell:startup` and press Enter
3. Create a shortcut to `bridge-tray.exe` in this folder

**Method 2: Task Scheduler**

1. Open Task Scheduler
2. Create Basic Task:
   - Name: "Biometric Bridge Tray"
   - Trigger: "When I log on"
   - Action: "Start a program"
   - Program: `C:\Program Files\BiometricBridge\bridge-tray.exe`
3. Set "Run with highest privileges" if needed

**Method 3: Registry (Advanced)**

Add registry key:
```
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run
Name: BiometricBridgeTray
Value: "C:\Program Files\BiometricBridge\bridge-tray.exe"
```

## Usage

### Starting the Tray App

Double-click `bridge-tray.exe` or launch from Startup folder. The app:
- Starts hidden (no window)
- Shows icon in system tray
- Enforces single instance (only one can run at a time)

### System Tray Menu

Right-click the tray icon to access:
- **Show Window** - Open main UI panel
- **Copy JWT** - Copy current JWT token to clipboard
- **Open Config** - Open config.yaml in default editor
- **Show Log** - View bridge log file
- **Quit** - Exit the tray app

### Main UI Panel

Click tray icon or select "Show Window" to see:
- Bridge status (running/stopped)
- Server address and port
- JWT token (click to copy)
- Recent events with filtering
- Start/Stop/Resync buttons

### Single Instance Behavior

If you try to launch a second instance:
- It detects the existing instance
- Brings the existing window to front
- Exits gracefully

## Configuration

The tray app reads configuration from:
1. Path specified in `BRIDGE_CONFIG` environment variable
2. `./config.yaml` (in same directory as exe)

**Example config.yaml:**
```yaml
server:
  port: 8443
  host: "0.0.0.0"

jwt:
  secret: "your-secret-key"
  token_expiry: 24h

realscan_sdk_path: "dlls\\RS_SDK.dll"

log:
  level: "info"
  format: "json"
```

## Deployment Scenarios

### Scenario 1: User Application

**Setup:**
- User installs to `C:\Users\<username>\AppData\Local\BiometricBridge\`
- Config in same directory
- Auto-start via Startup folder

**Pros:**
- No admin rights needed
- Per-user configuration
- Easy uninstall

### Scenario 2: System-Wide Installation

**Setup:**
- Install to `C:\Program Files\BiometricBridge\`
- Config in `C:\ProgramData\BiometricBridge\`
- Auto-start via Task Scheduler (all users)

**Pros:**
- Single installation for all users
- Centralized configuration
- Professional deployment

### Scenario 3: Portable Installation

**Setup:**
- Extract to any folder (USB drive, network share, etc.)
- Config in same folder
- Run manually

**Pros:**
- No installation needed
- Move between machines
- No registry changes

## Troubleshooting

### Tray app won't start

**Check for existing instance:**
```powershell
Get-Process bridge-tray
```

**Kill existing process:**
```powershell
Stop-Process -Name bridge-tray -Force
```

**Check port 17070:**
The app uses TCP port 17070 for single-instance enforcement.
```powershell
netstat -ano | findstr :17070
```

### Missing DLL errors

Fyne requires Windows system DLLs that should be present on Windows 10/11:
- `OpenGL32.dll`
- `Gdi32.dll`
- `Comdlg32.dll`
- `Ole32.dll`
- `Shell32.dll`

If missing, install latest Windows updates.

### Bridge won't start from tray

The tray app launches `bridge-realscan.exe` as a subprocess. Ensure:
1. `bridge-realscan.exe` is in the same directory or in PATH
2. All SDK DLLs are present
3. Config file is valid
4. No other instance of bridge is running

**Check manually:**
```cmd
cd "C:\Program Files\BiometricBridge"
bridge-realscan.exe
```

### No system tray icon

Some Windows configurations hide system tray icons. Check:
1. Click the up arrow (^) in system tray to show hidden icons
2. Right-click taskbar → Taskbar settings → Select which icons appear on taskbar
3. Enable "Biometric Bridge" or "The Kinetic Vault"

### High CPU usage

First launch compiles and caches UI components. Subsequent launches are faster.

If CPU stays high:
- Check for bridge errors in event log
- Verify device is not continuously scanning
- Check Windows Task Manager for details

## Uninstallation

### Manual Uninstall

1. Stop the tray app (right-click → Quit)
2. Remove from Startup folder if configured
3. Delete installation directory
4. Optional: Clean up config:
   ```powershell
   Remove-Item -Recurse "$env:APPDATA\BiometricBridge"
   ```

### Registry Cleanup (if used)

Remove auto-start registry key:
```
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run\BiometricBridgeTray
```

## Build Troubleshooting

### "gcc: command not found"

Install MinGW-w64 or TDM-GCC and add to PATH:
```cmd
setx PATH "%PATH%;C:\TDM-GCC-64\bin"
```

### "cgo: C compiler not found"

Ensure CGO_ENABLED=1 and GCC is in PATH:
```cmd
set CGO_ENABLED=1
where gcc
```

### Build takes forever

This is normal for first build. Fyne downloads and compiles:
- OpenGL bindings
- Image libraries
- GUI components
- Platform-specific code

Subsequent builds use Go's cache and are much faster (~30 seconds).

### Out of memory during build

Fyne builds can use significant RAM (2-4GB). Close other applications or:
- Increase virtual memory (pagefile)
- Build on machine with more RAM
- Use pre-built binary

## Architecture Notes

### Cross-Platform Code

The tray app uses the same codebase for Linux and Windows:
- `internal/tray/*.go` - Platform-independent logic
- Fyne handles platform-specific GUI rendering
- Single-instance uses TCP socket (works on both platforms)

### Dependencies

- **Fyne v2.5.5** - GUI framework
- **fyne.io/systray** - System tray integration  
- Standard Go libraries

### Communication with Bridge

The tray app can:
1. Launch bridge as subprocess
2. Monitor bridge stdout/stderr
3. Send signals to bridge process
4. Read bridge config file

It does NOT:
- Embed bridge code
- Communicate via websocket (future enhancement)
- Require bridge to be running (can start/stop it)

## Future Enhancements

- [ ] WebSocket connection to running bridge (real-time status)
- [ ] Installer package (MSI/NSIS)
- [ ] Update notifications
- [ ] Settings dialog (edit config in UI)
- [ ] Device status in tray tooltip
- [ ] Scan history visualization
- [ ] Multi-language support

## Support

For issues:
1. Check this guide's troubleshooting section
2. Review main documentation: `WINDOWS-DEPLOYMENT.md`
3. Check system logs and tray app console output
4. Report issues with:
   - Windows version
   - Error messages
   - Steps to reproduce

## Technical Details

**Binary size:** ~25-30MB (includes Fyne UI framework)
**Memory usage:** ~50-100MB (typical)
**CPU usage:** <1% when idle, 2-5% when updating UI
**Network:** TCP port 17070 (localhost only, for single-instance)
**Dependencies:** Windows 10/11 system DLLs only

**Build output includes:**
- Compiled Go code
- Embedded icon resources
- Fyne UI components
- OpenGL rendering code
- Windows GUI bindings

All dependencies are statically linked - no external DLLs required except Windows system libraries.
