// The Kinetic Vault — System tray GUI for the Biometric Bridge
package main

import (
	"fmt"
	"log/slog"
	"os"

	"biometric-bridge/internal/config"
	"biometric-bridge/internal/tray"
)

func main() {
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
	slog.Info("Starting The Kinetic Vault")

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
