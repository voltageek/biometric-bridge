// Package events provides the event broker (fan-out hub) and WebSocket handler
// for real-time event streaming from device drivers to browser clients.
package events

import (
	"log/slog"
	"sync"

	"biometric-bridge/internal/driver"
)

const maxSubscribers = 10

// Broker receives events from the driver's Subscribe channel and fans them out
// to up to 10 WebSocket subscriber channels (FR-004).
type Broker struct {
	mu          sync.Mutex
	subscribers map[chan driver.Event]struct{}
	source      <-chan driver.Event
	done        chan struct{}
}

// NewBroker creates a broker that reads from the given source channel.
func NewBroker(source <-chan driver.Event) *Broker {
	return &Broker{
		subscribers: make(map[chan driver.Event]struct{}),
		source:      source,
		done:        make(chan struct{}),
	}
}

// Start begins reading from the source channel and fanning out to subscribers.
// It runs until the source channel is closed or Stop is called.
func (b *Broker) Start() {
	go b.run()
}

// Stop signals the broker to stop processing events.
func (b *Broker) Stop() {
	close(b.done)
}

// Subscribe registers a new subscriber. Returns a channel for receiving events
// and a boolean indicating success. Returns nil, false if the subscriber limit
// is reached.
func (b *Broker) Subscribe() (chan driver.Event, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.subscribers) >= maxSubscribers {
		return nil, false
	}

	ch := make(chan driver.Event, 64) // buffered to absorb bursts
	b.subscribers[ch] = struct{}{}
	return ch, true
}

// Unsubscribe removes a subscriber and closes its channel.
func (b *Broker) Unsubscribe(ch chan driver.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// SubscriberCount returns the current number of subscribers.
func (b *Broker) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subscribers)
}

func (b *Broker) run() {
	for {
		select {
		case <-b.done:
			b.drainSubscribers()
			return
		case evt, ok := <-b.source:
			if !ok {
				b.drainSubscribers()
				return
			}
			b.fanOut(evt)
		}
	}
}

func (b *Broker) fanOut(evt driver.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.subscribers {
		select {
		case ch <- evt:
		default:
			slog.Warn("slow subscriber, dropping event",
				"type", evt.Type,
				"device", evt.DeviceName,
			)
		}
	}
}

func (b *Broker) drainSubscribers() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, ch)
	}
}

// Emit allows direct injection of driver.Event into the broker's fan-out
// mechanism. This is used by API handlers to emit workflow-level events
// (e.g., enrollment_retry) that are not produced by drivers.
func (b *Broker) Emit(evt driver.Event) {
	// Fan out synchronously to preserve ordering relative to handler actions.
	b.fanOut(evt)
}
