# GitHub Actions Workflows

This directory contains automated workflows for building and releasing the Biometric Bridge.

## Workflows

### `build-release.yml` - Build and Release

**Triggers:**
- Push to tags matching `v*` (e.g., `v1.0.0`, `v2.1.3`)
- Manual trigger via workflow_dispatch

**What it does:**
1. Builds bridge and tray applications for Linux and Windows
2. Creates deployment packages with SDK libraries and documentation
3. Generates release notes with checksums
4. Creates a GitHub release with downloadable artifacts

**Usage:**

**Automated release** (recommended):
```bash
# Tag and push
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions will automatically build and create a release
```

**Manual trigger:**
1. Go to Actions tab in GitHub
2. Select "Build and Release" workflow
3. Click "Run workflow"
4. Enter version number (e.g., `1.0.0`)
5. Click "Run workflow"

**Artifacts produced:**
- `biometric-bridge-linux-{version}.tar.gz` - Linux deployment package
- `biometric-bridge-windows-{version}.zip` - Windows deployment package
- Both include bridge + tray applications with SDK libraries

### `ci.yml` - Continuous Integration

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches
- Only runs when `biometric-bridge/**` or `.github/workflows/**` files change

**What it does:**
1. **Lint and Test** - Runs code quality checks and tests
   - `go fmt` - Code formatting
   - `go vet` - Static analysis
   - `staticcheck` - Advanced linting
   - `go test` - Unit tests with race detection and coverage

2. **Build Linux** - Builds both applications on Ubuntu
   - Bridge with RealScan support
   - Tray application

3. **Build Windows** - Native Windows build on windows-latest
   - Bridge with RealScan support
   - Tray application

4. **Build Windows (Cross-compile)** - Cross-compilation from Linux
   - Validates cross-compile toolchain
   - Ensures build scripts work correctly

**All builds must pass before merging PRs**

## Version Information

All builds include version information:
- **Version**: Git tag or "dev" for untagged builds
- **Build Time**: UTC timestamp of build
- **Git Commit**: Short commit SHA

Version info is embedded via ldflags:
```bash
-X main.Version={version}
-X main.BuildTime={timestamp}
-X main.GitCommit={commit}
```

View version in applications:
```bash
# Linux
./bridge-realscan --version
./bridge-tray --version

# Windows
bridge-realscan.exe --version
bridge-tray.exe --version
```

## Requirements

### Linux Build
- Ubuntu 20.04+ (latest runner)
- Go 1.24.7
- CGo enabled
- Dependencies:
  - libgl1-mesa-dev
  - xorg-dev
  - libx11-dev, libxcursor-dev, libxrandr-dev
  - libxinerama-dev, libxi-dev, libxxf86vm-dev
  - libgtk-3-dev

### Windows Build
- Windows Server 2022+ (latest runner)
- Go 1.24.7
- CGo enabled
- MinGW-w64 (pre-installed on windows-latest)

### Cross-Compile Build
- Ubuntu 20.04+ (latest runner)
- Go 1.24.7
- mingw-w64 cross-compiler

## Secrets Required

Currently, no custom secrets are required. The workflows use:
- `GITHUB_TOKEN` (automatically provided by GitHub Actions)

## Build Times

Approximate build times per job:

| Job | Time |
|-----|------|
| Lint and Test | 2-5 min |
| Build Linux | 5-10 min |
| Build Windows | 5-10 min |
| Build Windows (Cross) | 3-7 min |
| Full Release | 15-25 min |

**Note:** Tray app builds are slow (~5-10 min) due to Fyne CGo dependencies (OpenGL, GUI libs).

## Deployment Package Contents

### Linux Package (`biometric-bridge-linux-{version}.tar.gz`)
```
bridge-realscan          # Main bridge executable
bridge-tray              # System tray executable
config.example.yaml      # Configuration template
start-realscan.sh        # Startup script
lib/                     # RealScan SDK libraries
README.txt               # Installation instructions
```

### Windows Package (`biometric-bridge-windows-{version}.zip`)
```
bridge-realscan.exe      # Main bridge executable
bridge-tray.exe          # System tray executable
config.example.yaml      # Configuration template
*.dll                    # RealScan SDK + dependencies
install-service.bat      # Service installation
uninstall-service.bat    # Service removal
start-bridge.bat         # Quick start script
README.txt               # Installation instructions
```

## Troubleshooting

### Build Failures

**"CGO_ENABLED=1 required"**
- Ensure CGO is enabled in build environment
- Check that C compiler is available (gcc/mingw)

**"Cannot find SDK libraries"**
- Verify SDK paths in workflow files
- Check that SDK is committed or available in runner

**"go.mod dependency errors"**
- Run `go mod tidy` locally
- Commit updated go.mod/go.sum files

### Release Failures

**"Tag already exists"**
- Delete the tag locally and remotely
- Re-create with new commit

**"Artifact upload failed"**
- Check artifact size (max 2GB per artifact)
- Verify network connectivity

**"Release creation failed"**
- Ensure `GITHUB_TOKEN` has write permissions
- Check that tag matches version

## Manual Build Locally

To build locally matching CI environment:

```bash
# Linux
cd biometric-bridge
make build-linux
make build-tray-linux

# Windows (from Linux)
make build-windows-cross

# Windows (native)
cd biometric-bridge
build-windows.bat
build-tray-windows.bat
```

## Maintenance

### Updating Go Version

Edit `env.GO_VERSION` in both workflow files:
```yaml
env:
  GO_VERSION: '1.24.7'  # Update this
```

### Updating Dependencies

1. Update in workflow files (e.g., Ubuntu version, package versions)
2. Test locally or in a branch
3. Merge after CI passes

### Adding New Platforms

To add macOS, ARM, etc.:
1. Add new job to `build-release.yml`
2. Add matching CI job to `ci.yml`
3. Update deployment scripts
4. Test thoroughly before release

## Best Practices

1. **Always test locally before pushing**
   - Run `make build-linux build-tray-linux`
   - Run `go test ./...`
   - Verify version flags work

2. **Use semantic versioning**
   - v1.0.0 - Major releases
   - v1.1.0 - Minor features
   - v1.1.1 - Patches

3. **Create releases from stable branches**
   - Tag from `main` for stable releases
   - Tag from `develop` for beta/RC releases

4. **Review artifacts before publishing**
   - Download from Actions artifacts
   - Test on target platform
   - Verify checksums

5. **Write clear release notes**
   - Highlight breaking changes
   - Document new features
   - List bug fixes
