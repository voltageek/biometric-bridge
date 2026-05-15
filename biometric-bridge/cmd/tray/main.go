// The Kinetic Vault — System tray GUI for the Biometric Bridge
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"biometric-bridge/internal/config"
	"biometric-bridge/internal/tray"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"

	flagVersion = flag.Bool("version", false, "Print version information and exit")
)

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Printf("Biometric Bridge Tray\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Check for single instance
	locker := tray.NewInstanceLocker()
	if !locker.TryLock() {
		// Another instance is running, bring it to front
		if err := locker.BringToFront(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to bring existing instance to front: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Another instance is already running. Bringing it to front...")
		os.Exit(0)
	}
	defer locker.Unlock()

	// Initialize logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("Starting The Kinetic Vault", "version", Version)

	// Initialize minimal buffers and stores
	jwtStore := tray.NewJWTStore()
	evBuf := tray.NewEventBuffer(100)
	logBuf := tray.NewLogBuffer(1000)

	// Determine config path (respect BRIDGE_CONFIG env var)
	cfgPath := os.Getenv("BRIDGE_CONFIG")
	if cfgPath == "" {
		cfgPath = "./config.yaml"
	}

	// Create controller (uses stubbed driver init for now)
	cfg := &config.BridgeConfig{}
	// Use zero-value cfg - later we'll load from file
	controller := tray.NewBridgeController(cfg, jwtStore, evBuf, logBuf)

	// Start UI with system tray. This call will block on the Fyne main loop.
	tray.StartUIWithTray(controller, jwtStore, evBuf, logBuf, cfgPath)
}
