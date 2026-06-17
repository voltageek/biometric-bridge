// Command bridge is the Biometric Bridge daemon — a local gateway that gives
// web applications authenticated HTTP/WebSocket access to Suprema fingerprint
// readers.
package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"biometric-bridge/internal/api"
	"biometric-bridge/internal/auth"
	"biometric-bridge/internal/config"
	"biometric-bridge/internal/device"
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/events"
)

var (
	flagTest   = flag.Bool("test", false, "Run a single scan test and exit (no HTTP server)")
	flagDemo   = flag.Bool("demo", false, "Start bridge in demo mode (mock driver, no hardware required)")
	flagOutput = flag.String("output", "", "Save captured fingerprint image (PNG if .png extension, raw grayscale otherwise; only with --test)")
	flagFinger = flag.String("finger", "", "Finger position for LED guidance during --test scan (e.g., right_index, left_thumb)")
	flagMode   = flag.String("mode", "", "Multi-finger capture mode for --test: left_four, right_four, two_thumbs")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// T016: --demo takes precedence over --test
	if *flagDemo {
		return runDemo()
	}

	// --- Load config ---
	cfgPath := "config.yaml"
	if p := os.Getenv("BRIDGE_CONFIG"); p != "" {
		cfgPath = p
	}
	slog.Info("loading config", "path", cfgPath)

	cfg, err := config.Load(cfgPath)
	if err != nil && !*flagTest {
		return fmt.Errorf("config: %w", err)
	}
	// In test mode, fall back to relaxed loading if full validation failed
	// (e.g., missing public_key_file which isn't needed for --test).
	if err != nil && *flagTest {
		cfg, err = config.LoadDriverOnly(cfgPath)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
	}

	// --- Setup logging (T011) ---
	setupLogging(cfg.Log.Level)

	// --- Initialize driver ---
	drv, err := initDriver(cfg)
	if err != nil {
		return fmt.Errorf("driver init: %w", err)
	}

	// --- Connect all devices (FR-011: fail-fast) ---
	devConfigs := make([]driver.DeviceConfig, len(cfg.Devices))
	for i, d := range cfg.Devices {
		devConfigs[i] = driver.DeviceConfig{
			Name:   d.Name,
			Addr:   d.Addr,
			Port:   d.Port,
			UseSSL: d.UseSSL,
		}
		slog.Info("connecting to device", "name", d.Name, "addr", fmt.Sprintf("%s:%d", d.Addr, d.Port))
	}

	connectCtx, connectCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer connectCancel()

	if err := drv.Connect(connectCtx, devConfigs); err != nil {
		return fmt.Errorf("device connection failed: %w", err)
	}

	// --- Populate device registry ---
	registry := device.NewRegistry()
	for _, info := range drv.ListDevices() {
		registry.Register(info)
		slog.Info("device connected", "name", info.Name, "model", info.Model, "id", info.ID)
	}
	slog.Info("all devices connected", "count", len(cfg.Devices))

	// --- Test mode: scan once and exit ---
	if *flagTest {
		return runTestScan(drv, cfg)
	}

	// --- Load auth public key (T015, FR-017) ---
	slog.Info("loading public key", "path", cfg.Bridge.PublicKeyFile)
	pubKey, err := auth.LoadPublicKey(cfg.Bridge.PublicKeyFile)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	slog.Info("loaded public key", "path", cfg.Bridge.PublicKeyFile)

	tokenValidator := auth.NewTokenValidator(
		pubKey,
		cfg.Bridge.TokenIssuer,
		cfg.Bridge.TokenAudience,
		cfg.Bridge.ClockSkewDuration(),
	)

	// --- Configure reconnection (T021, FR-010) ---
	configureReconnect(drv, cfg, registry)

	// --- Start event broker ---
	slog.Info("starting event monitor")
	eventCh := drv.Subscribe()
	broker := events.NewBroker(eventCh)
	broker.Start()

	// --- Build router and start HTTP server ---
	handler := api.NewRouter(api.RouterDeps{
		TokenValidator: tokenValidator,
		Driver:         drv,
		Registry:       registry,
		Broker:         broker,
		AllowedOrigin:  cfg.Bridge.AllowedOrigin,
		RequestLogging: cfg.Log.RequestLogging,
	})

	srv := &http.Server{
		Addr:    cfg.Bridge.Listen,
		Handler: handler,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Bridge.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// --- Graceful shutdown (T024, FR-019) ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("received signal, shutting down", "signal", sig)
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	// Ordered teardown
	slog.Info("shutdown: stopping HTTP server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	slog.Info("shutdown: stopping event broker")
	broker.Stop()

	slog.Info("shutdown: closing driver")
	if err := drv.Close(); err != nil {
		slog.Error("driver close error", "error", err)
	}

	slog.Info("shutdown complete")
	return nil
}

// runDemo starts the bridge in demo mode with a mock driver.
// No hardware, SDK libraries, or config file is required.
func runDemo() error {
	slog.Info("========================================")
	slog.Info("  Starting Biometric Bridge in DEMO mode")
	slog.Info("========================================")

	// --- Load config (with fallback to defaults) ---
	cfgPath := "config.yaml"
	if p := os.Getenv("BRIDGE_CONFIG"); p != "" {
		cfgPath = p
	}

	cfg := config.LoadDemoDefaults()

	if _, err := os.Stat(cfgPath); err == nil {
		slog.Info("config file found, loading with relaxed validation", "path", cfgPath)
		loaded, err := config.LoadDriverOnly(cfgPath)
		if err != nil {
			slog.Warn("config load failed, using defaults", "path", cfgPath, "error", err)
		} else {
			cfg = loaded
		}
	}

	// Override driver to demo regardless of config
	cfg.Driver = "demo"

	// --- Setup logging ---
	setupLogging(cfg.Log.Level)

	// --- Initialize demo driver ---
	drv, err := initDriver(cfg)
	if err != nil {
		return fmt.Errorf("demo driver init: %w", err)
	}

	// --- Populate device registry ---
	registry := device.NewRegistry()
	for _, info := range drv.ListDevices() {
		registry.Register(info)
	}
	slog.Info("demo devices registered", "count", len(cfg.Devices))

	// --- Generate and print demo JWT (T014, FR-013) ---
	issuer := cfg.Bridge.TokenIssuer
	if issuer == "" {
		issuer = "demo"
	}
	audience := cfg.Bridge.TokenAudience
	if audience == "" {
		audience = "biometric-bridge"
	}

	jwtToken, err := auth.DemoJWT(issuer, audience, "demo-user", "Demo User", 1*time.Hour)
	if err != nil {
		return fmt.Errorf("generate demo JWT: %w", err)
	}
	fmt.Println()
	fmt.Printf("  Demo JWT: %s\n", jwtToken)
	fmt.Println()

	// --- Load auth public key (T015, FR-019) ---
	var pubKey *ecdsa.PublicKey
	if cfg.Bridge.PublicKeyFile != "" {
		slog.Info("loading public key from config", "path", cfg.Bridge.PublicKeyFile)
		pubKey, err = auth.LoadPublicKey(cfg.Bridge.PublicKeyFile)
		if err != nil {
			slog.Warn("config public key failed, using embedded test key", "error", err)
		}
	}
	if pubKey == nil {
		slog.Info("using embedded test public key for demo mode")
		pubKey, err = auth.LoadPublicKeyBytes(auth.TestPublicKeyPEM())
		if err != nil {
			return fmt.Errorf("load embedded test public key: %w", err)
		}
	}

	tokenValidator := auth.NewTokenValidator(
		pubKey,
		issuer,
		audience,
		cfg.Bridge.ClockSkewDuration(),
	)

	// --- Start event broker ---
	slog.Info("starting event monitor")
	eventCh := drv.Subscribe()
	broker := events.NewBroker(eventCh)
	broker.Start()

	// --- Build router and start HTTP server ---
	handler := api.NewRouter(api.RouterDeps{
		TokenValidator: tokenValidator,
		Driver:         drv,
		Registry:       registry,
		Broker:         broker,
		AllowedOrigin:  cfg.Bridge.AllowedOrigin,
		Demo:           true,
		RequestLogging: cfg.Log.RequestLogging,
	})

	srv := &http.Server{
		Addr:    cfg.Bridge.Listen,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Bridge.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// --- Graceful shutdown ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("received signal, shutting down", "signal", sig)
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	slog.Info("shutdown: stopping HTTP server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	slog.Info("shutdown: stopping event broker")
	broker.Stop()

	slog.Info("shutdown: closing driver")
	if err := drv.Close(); err != nil {
		slog.Error("driver close error", "error", err)
	}

	slog.Info("shutdown complete")
	return nil
}

// runTestScan performs a single fingerprint capture for testing purposes.
// It prints device info, captures one image, displays results (quality, size,
// hex preview), optionally saves the raw image to a file, then cleans up.
// When --mode is specified, performs a multi-finger slap capture instead.
func runTestScan(drv driver.Driver, cfg *config.BridgeConfig) error {
	defer func() {
		slog.Info("test: closing driver")
		drv.Close()
	}()

	devices := drv.ListDevices()
	if len(devices) == 0 {
		return fmt.Errorf("no devices available for test scan")
	}

	dev := devices[0]
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║          BIOMETRIC BRIDGE — TEST MODE        ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Device:   %s\n", dev.Name)
	fmt.Printf("  Model:    %s\n", dev.Model)
	fmt.Printf("  ID:       %s\n", dev.ID)
	fmt.Printf("  Firmware: %s\n", dev.FirmwareVersion)
	fmt.Println()

	// Multi-finger slap capture mode
	if *flagMode != "" {
		return runTestSlapScan(drv, dev)
	}

	fmt.Println("  Place your finger on the scanner...")
	fmt.Println()

	// Determine finger position for LED guidance
	finger := driver.FingerPosition(*flagFinger)
	if *flagFinger != "" {
		if !driver.ValidFingerPositions[finger] {
			return fmt.Errorf("invalid --finger value %q; valid values: left_little, left_ring, left_middle, left_index, left_thumb, right_thumb, right_index, right_middle, right_ring, right_little", *flagFinger)
		}
		fmt.Printf("  LED guidance: %s\n", *flagFinger)
		fmt.Println()
	}

	scanCtx, scanCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer scanCancel()

	result, err := drv.Scan(scanCtx, dev.Name, finger)
	if err != nil {
		return fmt.Errorf("test scan failed: %w", err)
	}

	// Display results
	fmt.Println("  ─── SCAN RESULT ───")
	fmt.Println()
	fmt.Printf("  Quality:    %d (NIST)\n", result.Quality)
	fmt.Printf("  Dimensions: %d x %d pixels\n", result.Width, result.Height)
	fmt.Printf("  Image size: %d bytes\n", len(result.Template))

	// Show hex preview (first 64 bytes)
	previewLen := 64
	if len(result.Template) < previewLen {
		previewLen = len(result.Template)
	}
	fmt.Printf("  Hex preview (first %d bytes):\n", previewLen)
	fmt.Println()
	fmt.Println(hex.Dump(result.Template[:previewLen]))

	// Save to file if requested
	if *flagOutput != "" {
		if err := saveImage(*flagOutput, result); err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}
		fmt.Printf("  Image saved to: %s\n", *flagOutput)
		fmt.Println()
	}

	fmt.Println("  Test complete.")
	return nil
}

// runTestSlapScan performs a multi-finger slap capture test.
func runTestSlapScan(drv driver.Driver, dev driver.DeviceInfo) error {
	mode := driver.CaptureMode(*flagMode)
	if !driver.ValidCaptureModes[mode] {
		return fmt.Errorf("invalid --mode value %q; valid values: left_four, right_four, two_thumbs", *flagMode)
	}

	fmt.Printf("  Capture mode: %s\n", *flagMode)
	fmt.Println()
	fmt.Println("  Place your fingers on the scanner...")
	fmt.Println()

	scanCtx, scanCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer scanCancel()

	result, err := drv.SlapScan(scanCtx, dev.Name, mode)
	if err != nil {
		return fmt.Errorf("test slap scan failed: %w", err)
	}

	// Display results
	fmt.Println("  ─── SLAP SCAN RESULT ───")
	fmt.Println()
	fmt.Printf("  Slap image:  %d x %d pixels (%d bytes)\n",
		result.SlapWidth, result.SlapHeight, len(result.SlapImage))
	fmt.Printf("  Fingers:     %d detected\n", len(result.Fingers))
	fmt.Println()

	for i, f := range result.Fingers {
		fingerLabel := string(f.Finger)
		if fingerLabel == "" {
			fingerLabel = "unknown"
		}
		fmt.Printf("  Finger %d: %-15s %4d x %4d  quality=%d  (%d bytes)\n",
			i+1, fingerLabel, f.Width, f.Height, f.Quality, len(f.Template))
	}
	fmt.Println()

	// Save output if requested
	if *flagOutput != "" {
		// Save the full slap image
		slapResult := &driver.ScanResult{
			Template: result.SlapImage,
			Width:    result.SlapWidth,
			Height:   result.SlapHeight,
		}
		slapPath := *flagOutput
		if err := saveImage(slapPath, slapResult); err != nil {
			return fmt.Errorf("failed to save slap image: %w", err)
		}
		fmt.Printf("  Slap image saved to: %s\n", slapPath)

		// Save each finger as finger_N.png
		base := strings.TrimSuffix(*flagOutput, ".png")
		base = strings.TrimSuffix(base, ".raw")
		for i, f := range result.Fingers {
			fingerResult := &driver.ScanResult{
				Template: f.Template,
				Width:    f.Width,
				Height:   f.Height,
			}
			fingerLabel := string(f.Finger)
			if fingerLabel == "" {
				fingerLabel = fmt.Sprintf("%d", i+1)
			}
			fingerPath := fmt.Sprintf("%s_finger_%s.png", base, fingerLabel)
			if err := saveImage(fingerPath, fingerResult); err != nil {
				slog.Warn("failed to save finger image", "path", fingerPath, "error", err)
				continue
			}
			fmt.Printf("  Finger %d saved to: %s\n", i+1, fingerPath)
		}
		fmt.Println()
	}

	fmt.Println("  Test complete.")
	return nil
}

// saveImage writes a fingerprint image to disk. If the path ends in ".png",
// it encodes a proper PNG from the raw 8-bit grayscale pixels. Otherwise it
// writes the raw bytes directly.
func saveImage(path string, result *driver.ScanResult) error {
	if strings.HasSuffix(strings.ToLower(path), ".png") {
		return savePNG(path, result)
	}
	return os.WriteFile(path, result.Template, 0644)
}

// savePNG encodes raw 8-bit grayscale pixel data as a PNG file.
func savePNG(path string, result *driver.ScanResult) error {
	if result.Width == 0 || result.Height == 0 {
		return fmt.Errorf("cannot write PNG: image dimensions unknown (width=%d, height=%d)", result.Width, result.Height)
	}

	img := &image.Gray{
		Pix:    result.Template,
		Stride: result.Width,
		Rect:   image.Rect(0, 0, result.Width, result.Height),
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

// setupLogging configures slog with JSON output at the specified level.
func setupLogging(level string) {
	var lvl slog.Level
	switch level {
	case "error":
		lvl = slog.LevelError
	case "debug":
		lvl = slog.LevelDebug
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})))
}

// initDriver creates the appropriate driver based on config. The actual driver
// implementations register themselves via build tags and init() functions.
func initDriver(cfg *config.BridgeConfig) (driver.Driver, error) {
	factory := getDriverFactory(cfg.Driver)
	if factory == nil {
		return nil, fmt.Errorf("driver %q not available (was this binary built with -tags %s?)", cfg.Driver, cfg.Driver)
	}

	var libPath string
	switch cfg.Driver {
	case "bs2":
		if cfg.BS2 != nil {
			libPath = cfg.BS2.LibPath
		}
	case "gsdk":
		if cfg.GSDK != nil {
			libPath = cfg.GSDK.GatewayAddr
		}
	case "realscan":
		if cfg.RealScan != nil {
			libPath = cfg.RealScan.LibPath
		}
	}

	return factory(libPath)
}

// reconnectable is an optional interface that drivers can implement to support
// automatic reconnection with exponential backoff (FR-010).
type reconnectable interface {
	SetReconnectConfig(base, cap time.Duration, registry driver.DeviceStateUpdater)
}

// configureReconnect sets up reconnection parameters if the driver supports it.
func configureReconnect(drv driver.Driver, cfg *config.BridgeConfig, registry *device.Registry) {
	if r, ok := drv.(reconnectable); ok {
		base := cfg.Events.ReconnectBaseDuration()
		cap_ := cfg.Events.ReconnectCapDuration()
		r.SetReconnectConfig(base, cap_, registry)
		slog.Info("reconnection configured", "base", base, "cap", cap_)
	}
}
