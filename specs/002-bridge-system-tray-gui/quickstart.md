# Quick Start Guide: Bridge System Tray GUI

**The Kinetic Vault** — System tray application for the Biometric Bridge

---

## Installation

### Prerequisites

- Go 1.21 or later
- Fyne dependencies (see below)
- Existing Biometric Bridge configuration (`config.yaml`)

### Platform-Specific Dependencies

**macOS:**
```bash
# No additional dependencies required
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install libgl1-mesa-dev xorg-dev
```

**Linux (Fedora):**
```bash
sudo dnf install mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
```

**Windows:**
```powershell
# No additional dependencies required
# Windows 10 or later recommended
```

### Build

```bash
cd biometric-bridge

# Build tray application
go build -o kinetic-vault ./cmd/tray

# (Optional) Build with version
go build -ldflags "-X main.version=1.0.0" -o kinetic-vault ./cmd/tray
```

---

## Configuration

### Step 1: Bridge Configuration

Ensure you have a valid `config.yaml` in the working directory:

```yaml
bridge:
  listen: "127.0.0.1:7070"
  allowed_origin: "http://localhost:3000"
  public_key_file: "./bridge-public.pem"
  token_issuer: "dev-server"
  token_audience: "biometric-bridge"
  clock_skew: "30s"

devices:
  - name: "reception"
    addr: "192.168.0.110"
    port: 51211
    use_ssl: false

events:
  reconnect_base: "1s"
  reconnect_cap: "120s"

log:
  level: "info"
  file: "bridge.log"  # NEW: Log file path (optional, defaults to bridge.log)

driver: realscan

realscan:
  lib_path: "../RealScan_SDK_for_Linux_v2.2.0.2470/Lib/libRS_SDK_x86_64.so.2.2.0.2470"
```

**New config field:** `log.file` specifies where structured logs are written for persistence.

### Step 2: Environment Variables

```bash
# Use custom config path (optional)
export BRIDGE_CONFIG=/path/to/config.yaml

# Development mode (opens panel on start for easier testing)
export KINETIC_VAULT_DEV=1
```

---

## Running

### From Command Line

```bash
# Start the tray application
./kinetic-vault

# Application starts minimized to system tray
# Bridge starts automatically
```

### Auto-Start on Login

**macOS:**
```bash
# Add to Login Items manually via System Settings
# Or use launchd plist (see install/macos/)
```

**Windows:**
```powershell
# Add shortcut to Startup folder
# %APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup
```

**Linux:**
```bash
# Add .desktop file to ~/.config/autostart/
# See install/linux/kinetic-vault.desktop
```

---

## Usage

### Basic Operations

| Action | How To |
|--------|--------|
| **View bridge status** | Click the tray icon — panel shows status, device count, and address |
| **Copy JWT token** | Click "COPY JWT" button in panel |
| **Re-sync devices** | Click "RE-SYNC" button in panel |
| **Restart bridge** | Click gear icon → "Restart Service" |
| **Stop bridge** | Click gear icon → "Stop Service" |
| **View detailed logs** | Click gear icon → "View Logs" |
| **Edit configuration** | Click gear icon → "Open Config" |
| **Quit application** | Right-click tray icon → "Quit" |

### Keyboard Shortcuts

None currently defined. Application is mouse-driven via tray interactions.

---

## Troubleshooting

### Application Won't Start

**Symptom:** No tray icon appears

**Check:**
```bash
# Verify config exists and is valid
cat config.yaml

# Test bridge independently first
./bridge --test

# Check for missing native libraries
ldd kinetic-vault  # Linux
otool -L kinetic-vault  # macOS
```

### Bridge Fails to Start

**Symptom:** Panel shows error status

**Check:**
```bash
# View logs
./kinetic-vault
# Then: Click gear → "View Logs"

# Or check log file
tail -f bridge.log

# Common causes:
# - Missing public_key_file
# - Device unreachable (check network/USB)
# - Invalid config.yaml
```

### COPY JWT Not Working

**Symptom:** Button disabled or "No token available"

**Solution:**
- JWT is only available after an authenticated request
- Use your web application to make a request to the bridge
- The token from that request will be captured and available to copy

### Tray Icon Not Visible (Linux)

**Symptom:** App running but no icon in system tray

**Solutions:**
```bash
# GNOME: Install AppIndicator extension
# KDE: Check system tray settings
# Check desktop environment supports system tray
```

---

## Development

### Project Structure

```
biometric-bridge/
├── cmd/
│   ├── bridge/main.go          # CLI bridge (existing)
│   └── tray/main.go            # Tray GUI (new)
├── internal/
│   ├── tray/
│   │   ├── controller.go       # Bridge lifecycle management
│   │   ├── observer.go         # JWT claims capture
│   │   ├── logger.go           # Multi-writer slog handler
│   │   ├── ringbuffer.go       # Generic ring buffer
│   │   ├── singleinstance.go   # Single-instance lock
│   │   └── ui.go               # Fyne UI components
│   ├── [existing packages]     # Reused from bridge
│   └── ...
└── go.mod
```

### Running in Development Mode

```bash
# Opens panel immediately on start for easier iteration
go run ./cmd/tray -dev

# Or with environment variable
KINETIC_VAULT_DEV=1 go run ./cmd/tray
```

### Testing

```bash
# Run all tests
go test ./...

# Run tray-specific tests
go test ./internal/tray/...

# Integration test with real bridge
go test -tags=integration ./internal/tray/...
```

---

## Uninstallation

### macOS
```bash
# Remove from Login Items (System Settings)
rm -f /usr/local/bin/kinetic-vault
rm -rf ~/.config/kinetic-vault
```

### Linux
```bash
rm -f ~/.config/autostart/kinetic-vault.desktop
rm -f ~/bin/kinetic-vault
rm -rf ~/.config/kinetic-vault
```

### Windows
```powershell
# Remove from Startup folder
# Remove executable
```

---

## Support

- **Issues**: [GitHub Issues](https://github.com/yourorg/biometric-bridge/issues)
- **Documentation**: `specs/001-biometric-bridge/` for bridge documentation
- **Logs**: Check `bridge.log` in the working directory
