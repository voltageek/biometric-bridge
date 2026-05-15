# Windows Deployment Package for RealScan Bridge

## Package Contents

After building, create a deployment directory with the following structure:

```
biometric-bridge-windows/
├── bridge-realscan.exe          # Main bridge executable
├── config.yaml                  # Configuration file (copy from Linux version)
├── dlls/                        # RealScan SDK DLLs
│   ├── RS_SDK.dll               # Main RealScan SDK library
│   ├── RS_SDK.ini               # SDK configuration
│   ├── NFIQ2.dll                # NIST Fingerprint Image Quality library
│   ├── opencv_world4100.dll     # OpenCV library
│   ├── tensorflowlite_c.dll     # TensorFlow Lite library
│   └── LICENSE.TXT              # SDK license
├── README.txt                   # Deployment instructions
└── install-service.bat          # Optional: Install as Windows service
```

## Creating the Deployment Package

Run from the `biometric-bridge` directory on Linux (after cross-compilation):

```bash
./create-windows-deployment.sh
```

Or manually:

```bash
# Create deployment directory
mkdir -p windows-deploy/dlls

# Copy executable
cp bridge-realscan.exe windows-deploy/

# Copy SDK DLLs
cp "../RealScanSDK for Windows_v2.2.0.2311/Bin/x64"/*.dll windows-deploy/dlls/
cp "../RealScanSDK for Windows_v2.2.0.2311/Bin/x64"/*.ini windows-deploy/dlls/
cp "../RealScanSDK for Windows_v2.2.0.2311/Bin/x64"/*.TXT windows-deploy/dlls/

# Copy configuration
cp config.yaml.example windows-deploy/config.yaml

# Create archive
cd windows-deploy
zip -r ../biometric-bridge-windows-v1.0.zip .
cd ..
```

## Deployment on Windows

### Option 1: Desktop Application

1. Extract the deployment package to `C:\Program Files\BiometricBridge\`
2. Copy all DLLs from `dlls\` directory to the same directory as `bridge-realscan.exe`
3. Edit `config.yaml` with your settings
4. Run `bridge-realscan.exe` from Command Prompt or PowerShell

### Option 2: Windows Service

1. Extract the deployment package to `C:\Program Files\BiometricBridge\`
2. Copy all DLLs from `dlls\` directory to the same directory as `bridge-realscan.exe`
3. Edit `config.yaml` with your settings
4. Run Command Prompt as Administrator
5. Install the service:
   ```cmd
   sc create BiometricBridge binPath= "C:\Program Files\BiometricBridge\bridge-realscan.exe" start= auto
   ```
6. Start the service:
   ```cmd
   sc start BiometricBridge
   ```

### Option 3: System Tray Application

(Coming soon - requires building the tray app for Windows)

## Configuration

Edit `config.yaml`:

```yaml
# RealScan SDK library path (Windows uses LoadLibrary with search path)
realscan_sdk_path: "dlls\\RS_SDK.dll"

# Server configuration
server:
  port: 8443
  host: "0.0.0.0"
  
# JWT authentication
jwt:
  secret: "your-secret-key-here"
  
# TLS (optional)
tls:
  enabled: true
  cert_file: "cert.pem"
  key_file: "key.pem"
```

## Verifying Installation

1. Check that all DLLs are in place:
   ```cmd
   dir "C:\Program Files\BiometricBridge\dlls"
   ```

2. Test the bridge (run from Command Prompt):
   ```cmd
   cd "C:\Program Files\BiometricBridge"
   bridge-realscan.exe
   ```

3. Check logs:
   - Console output shows startup messages
   - Look for "RealScan SDK loaded successfully"
   - Look for device detection messages

## Troubleshooting

### Bridge won't start

- **Check DLL paths**: Ensure all SDK DLLs are in the `dlls\` directory or same directory as executable
- **Check config.yaml**: Verify the `realscan_sdk_path` points to correct DLL location
- **Check permissions**: Run as Administrator if needed
- **Check USB driver**: Install RealScan USB driver from SDK's `Driver\` directory

### SDK won't load

- **Missing dependencies**: Ensure all DLL dependencies are present (NFIQ2.dll, opencv_world4100.dll, tensorflowlite_c.dll)
- **Architecture mismatch**: Verify you're using x64 DLLs with x64 executable
- **Visual C++ Runtime**: Install Visual C++ Redistributable 2015-2022 (x64)

### Device not detected

- **USB driver**: Install the RealScan USB driver:
  1. Go to SDK directory: `RealScanSDK for Windows_v2.2.0.2311\Driver\`
  2. Right-click on `realscan_driver_win10_x64.inf`
  3. Select "Install"
  4. Reboot if prompted
- **Device permissions**: Some Windows versions require admin rights for USB access
- **Hot-plug support**: Unplug and replug the device while bridge is running

## Uninstallation

### Desktop Application
Simply delete the installation directory

### Windows Service
1. Stop the service:
   ```cmd
   sc stop BiometricBridge
   ```
2. Delete the service:
   ```cmd
   sc delete BiometricBridge
   ```
3. Delete the installation directory

## Building from Source on Windows

If you want to build natively on Windows instead of cross-compiling:

1. Install Go 1.24.7 or later from https://go.dev/dl/
2. Install MinGW-w64 from https://www.mingw-w64.org/
3. Add MinGW bin directory to PATH
4. Run `build-windows.bat`

## SDK Version

- RealScan SDK for Windows v2.2.0.2311
- Compatible with RealScan G-10, G-10F, G-10I devices
- Requires Windows 10 or later (x64 only)

## License

See LICENSE.TXT in the dlls\ directory for RealScan SDK license terms.
