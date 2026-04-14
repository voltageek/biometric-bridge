package demo

import (
	"sync"
	"time"

	"biometric-bridge/internal/driver"
)

// EventSimulator emits synthetic events on a channel at configurable intervals.
type EventSimulator struct {
	cfg    DemoConfig
	ch     chan driver.Event
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewEventSimulator creates an event simulator that emits events on ch.
func NewEventSimulator(cfg DemoConfig, ch chan driver.Event) *EventSimulator {
	return &EventSimulator{
		cfg:    cfg,
		ch:     ch,
		stopCh: make(chan struct{}),
	}
}

// Start begins emitting periodic scan events. Blocks until Stop is called.
func (s *EventSimulator) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.cfg.EventInterval)
		defer ticker.Stop()

		// Emit initial "connected" events for all devices
		for _, dc := range s.cfg.Devices {
			select {
			case <-s.stopCh:
				return
			case s.ch <- driver.Event{Type: "connected", DeviceName: dc.Name}:
			}
		}

		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				// Emit a synthetic scan event for a random device
				if len(s.cfg.Devices) == 0 {
					continue
				}
				idx := generateRandomInt(0, len(s.cfg.Devices)-1)
				dc := s.cfg.Devices[idx]
				select {
				case <-s.stopCh:
					return
				case s.ch <- driver.Event{
					Type:       "scan",
					DeviceName: dc.Name,
					UserID:     "demo-user",
					EventCode:  0,
				}:
				}
			}
		}
	}()
}

// Stop signals the simulator to stop and waits for it to finish.
func (s *EventSimulator) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// EmitAPIEvent pushes an API-triggered event onto the channel (non-blocking).
func (s *EventSimulator) EmitAPIEvent(evt driver.Event) {
	select {
	case <-s.stopCh:
	case s.ch <- evt:
	default:
		// Channel full, drop event
	}
}
