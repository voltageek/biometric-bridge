// Package device provides a thread-safe device registry with per-device busy
// locks (FR-015), name-to-ID mapping, and state tracking.
package device

import (
	"fmt"
	"sync"

	"biometric-bridge/internal/driver"
)

// Registry tracks device state and provides per-device mutual exclusion for
// scan and enroll operations.
type Registry struct {
	mu      sync.RWMutex
	devices map[string]*entry // keyed by device name
}

type entry struct {
	mu    sync.Mutex
	info  driver.DeviceInfo
	state driver.DeviceState
}

// NewRegistry creates an empty device registry.
func NewRegistry() *Registry {
	return &Registry{
		devices: make(map[string]*entry),
	}
}

// Register adds a device to the registry with an initial idle state.
// This is called at startup after successful connection.
func (r *Registry) Register(info driver.DeviceInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[info.Name] = &entry{
		info:  info,
		state: driver.DeviceIdle,
	}
}

// Acquire attempts to lock a device for an operation (scan or enroll).
// Returns nil on success.
// Returns an error with the appropriate message for:
//   - unknown device name
//   - device disconnected (503)
//   - device busy (409)
func (r *Registry) Acquire(name string) error {
	r.mu.RLock()
	e, ok := r.devices[name]
	r.mu.RUnlock()
	if !ok {
		return &DeviceError{Code: 400, Message: fmt.Sprintf("unknown device: %s", name)}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	switch e.state {
	case driver.DeviceDisconnected:
		return &DeviceError{Code: 503, Message: fmt.Sprintf("device unavailable: %s", name)}
	case driver.DeviceBusy:
		return &DeviceError{Code: 409, Message: "device busy"}
	}

	e.state = driver.DeviceBusy
	return nil
}

// Release marks a device as idle after an operation completes.
func (r *Registry) Release(name string) {
	r.mu.RLock()
	e, ok := r.devices[name]
	r.mu.RUnlock()
	if !ok {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Only transition back to idle if still busy (not if disconnected during op)
	if e.state == driver.DeviceBusy {
		e.state = driver.DeviceIdle
	}
}

// SetState updates the state of a named device. Used by the driver to report
// disconnection and reconnection.
func (r *Registry) SetState(name string, state driver.DeviceState) {
	r.mu.RLock()
	e, ok := r.devices[name]
	r.mu.RUnlock()
	if !ok {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.state = state
}

// State returns the current state of a named device.
func (r *Registry) State(name string) (driver.DeviceState, bool) {
	r.mu.RLock()
	e, ok := r.devices[name]
	r.mu.RUnlock()
	if !ok {
		return 0, false
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state, true
}

// DeviceError is returned by Acquire when a device cannot be locked.
type DeviceError struct {
	Code    int    // HTTP status code (400, 409, 503)
	Message string // Human-readable error message
}

func (e *DeviceError) Error() string {
	return e.Message
}
