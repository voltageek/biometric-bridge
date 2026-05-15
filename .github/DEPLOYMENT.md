# GitHub Actions Deployment Setup

## Overview

Automated CI/CD pipelines for building and releasing the Biometric Bridge for Windows and Linux platforms.

## Files Created

```
.github/workflows/
├── build-release.yml    # Release workflow (tags + manual)
├── ci.yml              # CI workflow (PR + branch pushes)
└── README.md           # Workflow documentation
```

## Key Features

### 1. Multi-Platform Builds
- **Linux**: Native builds on Ubuntu (bridge + tray)
- **Windows**: Native builds on Windows Server (bridge + tray)
- **Cross-compile**: Windows builds from Linux (validation)

### 2. Version Management
- Automatic versioning from Git tags
- Version info embedded in binaries via ldflags
- Check version: `./bridge-realscan --version`

### 3. Automated Releases
- Triggered by pushing tags: `git push origin v1.0.0`
- Creates GitHub release with:
  - Linux deployment package (.tar.gz)
  - Windows deployment package (.zip)
  - Release notes with checksums
  - Download links for all platforms

### 4. Quality Checks
- Code formatting (`go fmt`)
- Static analysis (`go vet`, `staticcheck`)
- Unit tests with race detection
- Code coverage reporting

### 5. Complete Deployment Packages
Each package includes:
- Bridge and tray executables
- SDK libraries and dependencies
- Configuration templates
- Installation scripts
- Documentation

## Usage

### Creating a Release

**Method 1: Git Tag (Recommended)**
```bash
# Create and push tag
git tag v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# GitHub Actions automatically:
# 1. Builds for Linux and Windows
# 2. Creates deployment packages
# 3. Generates release notes
# 4. Publishes GitHub release
```

**Method 2: Manual Trigger**
1. Go to GitHub Actions tab
2. Select "Build and Release"
3. Click "Run workflow"
4. Enter version: `1.0.0`
5. Click "Run workflow"

### Pull Request Checks

When you create a PR, CI automatically:
1. Runs linting and tests
2. Builds for all platforms
3. Reports status on PR
4. Must pass before merging

## Version Information

All binaries include embedded version info:

```go
// Set via ldflags during build
var (
    Version   = "1.0.0"           // From git tag
    BuildTime = "2026-05-15_..."   // UTC timestamp
    GitCommit = "abc1234"          // Short commit SHA
)
```

**Usage:**
```bash
$ ./bridge-realscan --version
Biometric Bridge
Version:    1.0.0
Build Time: 2026-05-15_14:30:00_UTC
Git Commit: abc1234

$ ./bridge-tray --version
Biometric Bridge Tray
Version:    1.0.0
Build Time: 2026-05-15_14:30:00_UTC
Git Commit: abc1234
```

## Workflow Jobs

### CI Workflow (`ci.yml`)

```yaml
Triggers:
  - Push to main/develop
  - Pull requests to main/develop
  - Only when biometric-bridge/** changes

Jobs:
  1. lint-and-test      # Quality checks + tests
  2. build-linux        # Native Linux build
  3. build-windows      # Native Windows build
  4. build-windows-cross # Cross-compile validation
```

### Release Workflow (`build-release.yml`)

```yaml
Triggers:
  - Tags matching v*
  - Manual workflow_dispatch

Jobs:
  1. build-linux        # Build + package for Linux
  2. build-windows      # Build + package for Windows
  3. create-release     # Generate release + publish
```

## Build Matrix

| Platform | Runner | Compiler | Time | Output |
|----------|--------|----------|------|--------|
| Linux | ubuntu-latest | gcc | 5-10 min | .tar.gz |
| Windows | windows-latest | MinGW | 5-10 min | .zip |
| Windows (cross) | ubuntu-latest | x86_64-w64-mingw32-gcc | 3-7 min | .exe |

## Dependencies

### Linux Build Requirements
```yaml
- Go 1.24.7
- CGO enabled
- Libraries:
  - libgl1-mesa-dev
  - xorg-dev, libx11-dev
  - libxcursor-dev, libxrandr-dev
  - libxinerama-dev, libxi-dev
  - libxxf86vm-dev
  - libgtk-3-dev
```

### Windows Build Requirements
```yaml
- Go 1.24.7
- CGO enabled
- MinGW-w64 (pre-installed)
- Visual C++ Redistributable (runtime)
```

## Local Build Commands

Match CI environment locally:

```bash
# Linux builds
cd biometric-bridge
make build-linux              # Bridge
make build-tray-linux         # Tray

# Windows cross-compile
make build-windows-cross      # Bridge (slow)
# Tray: Build natively on Windows

# Check version
./bridge-realscan --version
./bridge-tray --version
```

## Deployment Packages

### Linux: `biometric-bridge-linux-{version}.tar.gz`
```
bridge-realscan           # 10MB - Bridge executable
bridge-tray               # 10MB - Tray executable
config.example.yaml       # Configuration template
start-realscan.sh         # Startup script with LD_LIBRARY_PATH
lib/                      # RealScan SDK .so files
README.txt                # Installation guide
```

**Installation:**
```bash
tar -xzf biometric-bridge-linux-1.0.0.tar.gz
cd biometric-bridge-linux-1.0.0
cp config.example.yaml config.yaml
# Edit config.yaml
./start-realscan.sh
```

### Windows: `biometric-bridge-windows-{version}.zip`
```
bridge-realscan.exe       # 18MB - Bridge executable
bridge-tray.exe           # 25MB - Tray executable
config.example.yaml       # Configuration template
*.dll                     # SDK libraries (RS_SDK.dll, etc.)
install-service.bat       # Windows service installer
uninstall-service.bat     # Windows service remover
start-bridge.bat          # Quick start script
README.txt                # Installation guide
```

**Installation:**
```powershell
# Extract ZIP
Expand-Archive biometric-bridge-windows-1.0.0.zip

# Run bridge
cd biometric-bridge-windows-1.0.0
copy config.example.yaml config.yaml
# Edit config.yaml
start-bridge.bat

# Optional: Install as service
Right-click install-service.bat → Run as Administrator
```

## Release Notes Template

Generated automatically with:
- Version and build info
- Download links (Linux + Windows)
- Features included
- Installation instructions
- Requirements
- SHA256 checksums

## Troubleshooting

### "Build failed: cannot find package"
**Solution:**
```bash
go mod tidy
git add go.mod go.sum
git commit -m "Update dependencies"
```

### "CGO_ENABLED=1 required"
**Solution:**
- Verify C compiler installed
- Check CGO_ENABLED in workflow
- Ensure build tags correct

### "Tag already exists"
**Solution:**
```bash
# Delete tag locally and remotely
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0

# Re-create and push
git tag v1.0.0
git push origin v1.0.0
```

### "Artifact too large"
**Solution:**
- Check SDK library sizes
- Remove unnecessary files from package
- Split into multiple artifacts if needed

## Security

- No secrets required (uses `GITHUB_TOKEN`)
- Workflows run in isolated containers
- Artifacts retained for 7 days
- Releases are public (adjust repo settings if needed)

## Best Practices

1. **Tag from stable branches** - Only tag `main` for releases
2. **Use semantic versioning** - v{major}.{minor}.{patch}
3. **Test locally first** - Run `make build-linux build-tray-linux`
4. **Review CI results** - Check all jobs pass before merging
5. **Download artifacts** - Test packages before announcing release

## Future Enhancements

Potential additions:
- [ ] macOS builds (native Apple Silicon + Intel)
- [ ] ARM64 Linux builds (Raspberry Pi, etc.)
- [ ] Docker image builds
- [ ] Automated security scanning
- [ ] Performance benchmarks in CI
- [ ] Automated changelog generation
- [ ] Codesigning for Windows executables
- [ ] Notarization for macOS binaries

## Maintenance

### Update Go version
Edit both workflow files:
```yaml
env:
  GO_VERSION: '1.24.7'  # Update here
```

### Update runner OS
```yaml
jobs:
  build-linux:
    runs-on: ubuntu-24.04  # Update here
```

### Add new platforms
1. Create new job in `build-release.yml`
2. Add CI job in `ci.yml`
3. Test thoroughly
4. Update documentation

## References

- [GitHub Actions Documentation](https://docs.github.com/actions)
- [Go Release Best Practices](https://goreleaser.com/quick-start/)
- [Semantic Versioning](https://semver.org/)
- [CGo Documentation](https://pkg.go.dev/cmd/cgo)

---

**Created:** 2026-05-15  
**Last Updated:** 2026-05-15  
**Status:** ✅ Ready for production
