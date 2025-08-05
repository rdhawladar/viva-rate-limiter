package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/viva/rate-limiter/pkg/errors"
)

// memoryEntry represents a single entry in the memory backend.
type memoryEntry struct {
	count       int64
	windowStart time.Time
	lastAccess  time.Time
}

// MemoryBackend implements the Backend interface using in-memory storage.
// This is suitable for testing, single-instance applications, or when persistence is not required.
type MemoryBackend struct {
	mu      sync.RWMutex
	data    map[string]*memoryEntry
	closed  bool
	
	// cleanupInterval controls how often expired entries are cleaned up
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewMemoryBackend creates a new in-memory backend.
func NewMemoryBackend() *MemoryBackend {
	backend := &MemoryBackend{
		data:            make(map[string]*memoryEntry),
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}
	
	// Start cleanup goroutine
	go backend.cleanupLoop()
	
	return backend
}

// NewMemoryBackendWithCleanup creates a new in-memory backend with custom cleanup interval.
func NewMemoryBackendWithCleanup(cleanupInterval time.Duration) *MemoryBackend {
	backend := &MemoryBackend{
		data:            make(map[string]*memoryEntry),
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}
	
	go backend.cleanupLoop()
	
	return backend
}

// Increment increments the counter for a key within a time window.
func (m *MemoryBackend) Increment(ctx context.Context, key string, window time.Duration) (int64, time.Time, error) {
	if ctx.Err() != nil {
		return 0, time.Time{}, ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, time.Time{}, errors.ErrBackendClosed
	}

	now := time.Now()
	entry, exists := m.data[key]

	if !exists || m.isWindowExpired(entry.windowStart, window, now) {
		// Create new window
		entry = &memoryEntry{
			count:       1,
			windowStart: m.getWindowStart(now, window),
			lastAccess:  now,
		}
		m.data[key] = entry
		return 1, entry.windowStart, nil
	}

	// Increment existing entry
	entry.count++
	entry.lastAccess = now
	return entry.count, entry.windowStart, nil
}

// Get retrieves the current count for a key within a time window.
func (m *MemoryBackend) Get(ctx context.Context, key string, window time.Duration) (int64, time.Time, error) {
	if ctx.Err() != nil {
		return 0, time.Time{}, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return 0, time.Time{}, errors.ErrBackendClosed
	}

	now := time.Now()
	entry, exists := m.data[key]

	if !exists || m.isWindowExpired(entry.windowStart, window, now) {
		// No entry or expired - return zero count with current window start
		return 0, m.getWindowStart(now, window), nil
	}

	return entry.count, entry.windowStart, nil
}

// Reset resets the counter for a key.
func (m *MemoryBackend) Reset(ctx context.Context, key string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.ErrBackendClosed
	}

	delete(m.data, key)
	return nil
}

// Close closes the backend and releases resources.
func (m *MemoryBackend) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	close(m.stopCleanup)
	m.data = nil
	return nil
}

// Size returns the number of entries currently stored.
func (m *MemoryBackend) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// Clear removes all entries.
func (m *MemoryBackend) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.closed {
		m.data = make(map[string]*memoryEntry)
	}
}

// getWindowStart calculates the start of the current window.
// For sliding window, this is simply the current time.
func (m *MemoryBackend) getWindowStart(now time.Time, window time.Duration) time.Time {
	return now
}

// isWindowExpired checks if a window has expired.
func (m *MemoryBackend) isWindowExpired(windowStart time.Time, window time.Duration, now time.Time) bool {
	return now.Sub(windowStart) >= window
}

// cleanupLoop periodically removes expired entries to prevent memory leaks.
func (m *MemoryBackend) cleanupLoop() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries.
func (m *MemoryBackend) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	now := time.Now()
	maxAge := 24 * time.Hour // Keep entries for at most 24 hours after last access

	for key, entry := range m.data {
		if now.Sub(entry.lastAccess) > maxAge {
			delete(m.data, key)
		}
	}
}