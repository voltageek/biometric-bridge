package demo

import (
	"sync"

	"biometric-bridge/internal/driver"
)

// EventSimulator manages API-triggered events for the demo driver.
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

// Start keeps the goroutine alive to handle API-triggered events via EmitAPIEvent.
// No automatic events are emitted — the WebSocket only broadcasts events triggered
// by actual API calls (scan, enroll), just like a real device would.
func (s *EventSimulator) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-s.stopCh
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
