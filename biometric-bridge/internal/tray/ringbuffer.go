// Package tray provides the system tray GUI for the Biometric Bridge.
package tray

import (
	"sync"
)

// RingBuffer is a thread-safe circular buffer with fixed capacity.
type RingBuffer[T any] struct {
	mu     sync.RWMutex
	buffer []T
	size   int
	head   int
	count  int
}

// NewRingBuffer creates a new ring buffer with the given capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = 100
	}
	return &RingBuffer[T]{
		buffer: make([]T, capacity),
		size:   capacity,
		head:   0,
		count:  0,
	}
}

// Push adds an item to the buffer, overwriting the oldest if full.
func (r *RingBuffer[T]) Push(item T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buffer[r.head] = item
	r.head = (r.head + 1) % r.size
	if r.count < r.size {
		r.count++
	}
}

// GetAll returns all items in chronological order (oldest first).
func (r *RingBuffer[T]) GetAll() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return []T{}
	}

	result := make([]T, r.count)
	start := (r.head - r.count + r.size) % r.size
	for i := 0; i < r.count; i++ {
		idx := (start + i) % r.size
		result[i] = r.buffer[idx]
	}
	return result
}

// GetRecent returns the last n items in chronological order.
func (r *RingBuffer[T]) GetRecent(n int) []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n <= 0 || r.count == 0 {
		return []T{}
	}

	if n > r.count {
		n = r.count
	}

	result := make([]T, n)
	start := (r.head - n + r.size) % r.size
	for i := 0; i < n; i++ {
		idx := (start + i) % r.size
		result[i] = r.buffer[idx]
	}
	return result
}

// Len returns the number of items in the buffer.
func (r *RingBuffer[T]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}

// Clear removes all items from the buffer.
func (r *RingBuffer[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	var zero T
	for i := range r.buffer {
		r.buffer[i] = zero
	}
	r.head = 0
	r.count = 0
}

// Capacity returns the maximum capacity of the buffer.
func (r *RingBuffer[T]) Capacity() int {
	return r.size
}
