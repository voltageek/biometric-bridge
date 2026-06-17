// Package tray provides the system tray GUI for the Biometric Bridge.
package tray

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"biometric-bridge/internal/api"
	"biometric-bridge/internal/auth"
	"biometric-bridge/internal/config"
	"biometric-bridge/internal/device"
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/events"
	"github.com/golang-jwt/jwt/v5"
)

// BridgeState represents the runtime state of the bridge.
type BridgeState int

const (
	StateStopped BridgeState = iota
	StateStarting
	StateRunning
	StateError
)

func (s BridgeState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// BridgeStatus provides runtime information about the bridge.
type BridgeStatus struct {
	State          BridgeState
	ListenAddr     string
	DeviceCount    int
	ConnectedCount int
	ErrorMessage   string
	Uptime         time.Duration
}

// BridgeController manages the embedded bridge lifecycle.
type BridgeController struct {
	cfg         *config.BridgeConfig
	state       BridgeState
	status      BridgeStatus
	statusMu    sync.RWMutex
	startTime   time.Time
	srv         *http.Server
	drv         driver.Driver
	registry    *device.Registry
	broker      *events.Broker
	jwtStore    *JWTStore
	eventBuffer *EventBuffer
	logBuffer   *LogBuffer
	statusSubs  map[chan BridgeStatus]struct{}
	subsMu      sync.RWMutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewBridgeController creates a new bridge controller.
func NewBridgeController(cfg *config.BridgeConfig, jwtStore *JWTStore, eventBuffer *EventBuffer, logBuffer *LogBuffer) *BridgeController {
	return &BridgeController{
		cfg:         cfg,
		state:       StateStopped,
		status:      BridgeStatus{State: StateStopped},
		jwtStore:    jwtStore,
		eventBuffer: eventBuffer,
		logBuffer:   logBuffer,
		statusSubs:  make(map[chan BridgeStatus]struct{}),
		stopCh:      make(chan struct{}),
	}
}

// Start initializes and starts the bridge.
func (c *BridgeController) Start(ctx context.Context) error {
	c.statusMu.Lock()
	if c.state == StateRunning || c.state == StateStarting {
		c.statusMu.Unlock()
		return fmt.Errorf("bridge already running")
	}
	c.state = StateStarting
	c.status.State = StateStarting
	c.statusMu.Unlock()

	c.notifyStatusChange()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("bridge panic recovered", "panic", r)
			c.setError(fmt.Sprintf("panic: %v", r))
		}
	}()

	// Initialize driver
	drv, err := c.initDriver()
	if err != nil {
		c.setError(fmt.Sprintf("driver init: %v", err))
		return err
	}
	c.drv = drv

	// Connect to devices
	if err := c.connectDevices(ctx); err != nil {
		c.setError(fmt.Sprintf("device connection: %v", err))
		return err
	}

	// Load auth key
	pubKey, err := auth.LoadPublicKey(c.cfg.Bridge.PublicKeyFile)
	if err != nil {
		c.setError(fmt.Sprintf("auth: %v", err))
		return err
	}

	validator := auth.NewTokenValidator(
		pubKey,
		c.cfg.Bridge.TokenIssuer,
		c.cfg.Bridge.TokenAudience,
		c.cfg.Bridge.ClockSkewDuration(),
	)

	// Setup auth middleware with JWT capture
	authCfg := auth.MiddlewareConfig{
		Validator: validator,
		SkipPaths: map[string]bool{"/healthz": true, "/events": true},
		OnAuthenticated: func(claims jwt.Claims) {
			if c.jwtStore != nil {
				// Get the raw token from the last request (stored separately)
				// For now, we'll capture claims only
				trayClaims := ExtractClaimsFromJWT(claims, "")
				c.jwtStore.SetClaims(trayClaims)
			}
		},
	}

	// Start event broker
	eventCh := c.drv.Subscribe()
	c.broker = events.NewBroker(eventCh)
	c.broker.Start()

	// Subscribe to events for the tray
	c.wg.Add(1)
	go c.eventSubscriber()

	// Build router
	handler := api.NewRouter(api.RouterDeps{
		TokenValidator: validator,
		Driver:         c.drv,
		Registry:       c.registry,
		Broker:         c.broker,
		AllowedOrigin:  c.cfg.Bridge.AllowedOrigin,
	})

	// Wrap with auth middleware
	handler = auth.MiddlewareWithConfig(authCfg)(handler)

	c.srv = &http.Server{
		Addr:    c.cfg.Bridge.Listen,
		Handler: handler,
	}

	// Start HTTP server
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		slog.Info("bridge listening", "addr", c.cfg.Bridge.Listen)
		if err := c.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			c.setError(fmt.Sprintf("server: %v", err))
		}
	}()

	c.statusMu.Lock()
	c.state = StateRunning
	c.status.State = StateRunning
	c.status.ListenAddr = c.cfg.Bridge.Listen
	c.status.DeviceCount = len(c.cfg.Devices)
	c.status.ConnectedCount = c.registry.Count()
	c.startTime = time.Now()
	c.statusMu.Unlock()

	c.notifyStatusChange()
	slog.Info("bridge started successfully")

	return nil
}

// Stop gracefully shuts down the bridge.
func (c *BridgeController) Stop(ctx context.Context) error {
	c.statusMu.Lock()
	if c.state == StateStopped {
		c.statusMu.Unlock()
		return nil
	}
	c.state = StateStopped
	c.status.State = StateStopped
	c.statusMu.Unlock()

	c.notifyStatusChange()

	// Signal stop
	close(c.stopCh)

	// Shutdown HTTP server
	if c.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := c.srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}

	// Stop event broker
	if c.broker != nil {
		c.broker.Stop()
	}

	// Wait for goroutines
	c.wg.Wait()

	// Close driver
	if c.drv != nil {
		if err := c.drv.Close(); err != nil {
			slog.Error("driver close error", "error", err)
		}
	}

	slog.Info("bridge stopped")
	return nil
}

// Restart stops then starts the bridge.
func (c *BridgeController) Restart(ctx context.Context) error {
	if err := c.Stop(ctx); err != nil {
		return err
	}
	// Reset stop channel
	c.stopCh = make(chan struct{})
	return c.Start(ctx)
}

// ResyncDevices re-initializes device connections.
func (c *BridgeController) ResyncDevices() {
	c.statusMu.RLock()
	state := c.state
	c.statusMu.RUnlock()

	if state != StateRunning {
		slog.Warn("cannot resync: bridge not running")
		return
	}

	// Emit event
	if c.eventBuffer != nil {
		c.eventBuffer.Push(EventEntry{
			Timestamp:   time.Now(),
			Title:       "Device Re-sync Started",
			Description: "Re-initializing connections to all configured devices",
			Severity:    SeverityInfo,
		})
	}

	// TODO: Implement actual device reconnection logic
	// This would involve closing and reopening driver connections
	slog.Info("device resync requested")
}

// Status returns current bridge status.
func (c *BridgeController) Status() BridgeStatus {
	c.statusMu.RLock()
	defer c.statusMu.RUnlock()

	status := c.status
	if c.state == StateRunning && !c.startTime.IsZero() {
		status.Uptime = time.Since(c.startTime)
	}
	return status
}

// Subscribe returns a channel for status updates.
func (c *BridgeController) Subscribe() <-chan BridgeStatus {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()

	ch := make(chan BridgeStatus, 1)
	c.statusSubs[ch] = struct{}{}

	// Send current status immediately
	go func() {
		ch <- c.Status()
	}()

	return ch
}

// Unsubscribe removes a status subscriber.
func (c *BridgeController) Unsubscribe(ch <-chan BridgeStatus) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()

	for sub := range c.statusSubs {
		if sub == ch {
			close(sub)
			delete(c.statusSubs, sub)
			return
		}
	}
}

// eventSubscriber subscribes to bridge events and forwards to the event buffer.
func (c *BridgeController) eventSubscriber() {
	defer c.wg.Done()

	// Subscribe to broker events
	// Note: This is a simplified version - actual implementation would
	// need to handle the specific event types from the bridge
	if c.broker == nil {
		// nothing to subscribe to
		<-c.stopCh
		return
	}

	ch, ok := c.broker.Subscribe()
	if !ok {
		<-c.stopCh
		return
	}

	for {
		select {
		case <-c.stopCh:
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			rawType := ev.Type
			desc := ev.Message
			if desc == "" {
				desc = fmt.Sprintf("device=%s", ev.DeviceName)
			}

			title, sev, et := MapBridgeEvent(rawType)
			if c.eventBuffer != nil {
				c.eventBuffer.Push(EventEntry{
					Timestamp:   time.Now(),
					Title:       title,
					Description: desc,
					Severity:    sev,
					DeviceName:  ev.DeviceName,
					Type:        et,
				})
			}
		}
	}
}

func (c *BridgeController) notifyStatusChange() {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	status := c.Status()
	for ch := range c.statusSubs {
		select {
		case ch <- status:
		default:
		}
	}
}

func (c *BridgeController) setError(msg string) {
	c.statusMu.Lock()
	c.state = StateError
	c.status.State = StateError
	c.status.ErrorMessage = msg
	c.statusMu.Unlock()
	c.notifyStatusChange()
}

func (c *BridgeController) initDriver() (driver.Driver, error) {
	// This would use the driver factory pattern from the existing bridge
	// For now, return nil - actual implementation would import the driver packages
	return nil, fmt.Errorf("driver initialization not yet implemented")
}

func (c *BridgeController) connectDevices(ctx context.Context) error {
	c.registry = device.NewRegistry()
	// Device connection logic would go here
	return nil
}
