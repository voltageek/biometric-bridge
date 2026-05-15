.PHONY: help build-linux build-windows-cross build-tray-linux build-tray-windows-cross deploy-windows deploy-windows-full clean test install

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

# Default target
help:
	@echo "biometric-bridge Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  make build-linux              - Build Linux bridge with RealScan support"
	@echo "  make build-windows-cross      - Cross-compile Windows bridge from Linux"
	@echo "  make build-tray-linux         - Build Linux system tray app"
	@echo "  make build-tray-windows-cross - Cross-compile Windows tray (slow, 15-30 min)"
	@echo "  make deploy-windows           - Create Windows deployment (bridge only)"
	@echo "  make deploy-windows-full      - Create Windows deployment (bridge + tray)"
	@echo "  make clean                    - Remove build artifacts"
	@echo "  make test                     - Run tests"
	@echo "  make install                  - Install bridge to /usr/local/bin"
	@echo ""
	@echo "Examples:"
	@echo "  make build-linux              # Standard Linux build"
	@echo "  make build-windows-cross      # Cross-compile bridge for Windows"
	@echo "  make deploy-windows           # Build + create Windows package"
	@echo "  make deploy-windows-full      # Build bridge + tray + package (very slow)"
	@echo ""
	@echo "Version: $(VERSION) | Commit: $(GIT_COMMIT)"
	@echo ""
	@echo "Note: Tray cross-compilation is VERY slow (15-30+ min) due to Fyne/CGo."
	@echo "      Recommend building tray natively on Windows instead."

# Build Linux bridge with RealScan support
build-linux:
	@echo "Building Linux bridge with RealScan support..."
	CGO_ENABLED=1 go build -tags realscan -ldflags="$(LDFLAGS)" -o bridge-realscan ./cmd/bridge
	@echo "✓ Build complete: bridge-realscan (version $(VERSION))"

# Cross-compile Windows bridge from Linux
build-windows-cross:
	@echo "Cross-compiling Windows bridge..."
	./build-windows-cross.sh $(VERSION)
	@echo "✓ Build complete: bridge-realscan.exe"

# Build Linux system tray app
build-tray-linux:
	@echo "Building Linux system tray app..."
	go build -ldflags="$(LDFLAGS)" -o bridge-tray ./cmd/tray
	@echo "✓ Build complete: bridge-tray (version $(VERSION))"

# Cross-compile Windows tray app (WARNING: Very slow!)
build-tray-windows-cross:
	@echo "⚠ WARNING: This build takes 15-30+ minutes due to Fyne/CGo complexity"
	@echo "⚠ Recommend building natively on Windows instead (see WINDOWS-TRAY.md)"
	@echo ""
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		./build-tray-windows-cross.sh $(VERSION); \
	else \
		echo "Build cancelled"; \
		exit 1; \
	fi

# Create Windows deployment package (bridge only)
deploy-windows:
	@echo "Creating Windows deployment package (bridge only)..."
	@if [ ! -f bridge-realscan.exe ]; then \
		echo "Building Windows bridge first..."; \
		$(MAKE) build-windows-cross; \
	fi
	./create-windows-deployment.sh 1.0.0
	@echo "✓ Deployment package ready: biometric-bridge-windows-v1.0.0.zip"

# Create Windows deployment package (bridge + tray)
deploy-windows-full:
	@echo "Creating Windows deployment package (bridge + tray)..."
	@if [ ! -f bridge-realscan.exe ]; then \
		echo "Building Windows bridge first..."; \
		$(MAKE) build-windows-cross; \
	fi
	@if [ ! -f bridge-tray.exe ]; then \
		echo ""; \
		echo "⚠ bridge-tray.exe not found"; \
		echo "⚠ Cross-compiling tray takes 15-30+ minutes"; \
		echo ""; \
		read -p "Build tray now? [y/N] " -n 1 -r; \
		echo; \
		if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
			$(MAKE) build-tray-windows-cross; \
		else \
			echo "Package will be created without tray app"; \
			echo "Build tray separately on Windows (see WINDOWS-TRAY.md)"; \
		fi \
	fi
	./create-windows-deployment.sh 1.0.0
	@echo "✓ Deployment package ready: biometric-bridge-windows-v1.0.0.zip"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bridge-realscan bridge-realscan.exe bridge-tray bridge-tray.exe
	rm -rf windows-deploy biometric-bridge-windows-*.zip
	go clean
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install bridge to system (Linux only)
install: build-linux
	@echo "Installing bridge to /usr/local/bin..."
	sudo cp bridge-realscan /usr/local/bin/
	sudo chmod 755 /usr/local/bin/bridge-realscan
	@echo "✓ Installed: /usr/local/bin/bridge-realscan"
