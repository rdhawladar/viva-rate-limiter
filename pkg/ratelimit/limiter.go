// Package ratelimit provides a flexible and efficient rate limiting library for Go applications.
// It supports multiple backends (Redis, in-memory) and algorithms (sliding window, token bucket).
package ratelimit

import (
	"context"
	"time"

	"github.com/rdhawladar/viva-rate-limiter/pkg/errors"
)

// Limiter represents a rate limiter that can check and enforce rate limits.
type Limiter interface {
	// Allow checks if a request for the given key is allowed.
	// Returns true if the request is within the rate limit, false otherwise.
	Allow(ctx context.Context, key string) bool

	// AllowN checks if N requests for the given key are allowed.
	// Returns true if all N requests are within the rate limit, false otherwise.
	AllowN(ctx context.Context, key string, n int) bool

	// Info returns detailed information about the current rate limit status for a key.
	Info(ctx context.Context, key string) (*LimitInfo, error)

	// Reset resets the rate limit counter for the given key.
	Reset(ctx context.Context, key string) error

	// SetLimit dynamically updates the rate limit for a specific key.
	// If no specific limit is set, the default limit is used.
	SetLimit(ctx context.Context, key string, limit int, window time.Duration) error

	// Close releases any resources held by the limiter.
	Close() error
}

// LimitInfo contains detailed information about a rate limit status.
type LimitInfo struct {
	// Key is the identifier for this rate limit
	Key string `json:"key"`

	// Limit is the maximum number of requests allowed in the window
	Limit int `json:"limit"`

	// Remaining is the number of requests remaining in the current window
	Remaining int `json:"remaining"`

	// Used is the number of requests used in the current window
	Used int `json:"used"`

	// Window is the time window for the rate limit
	Window time.Duration `json:"window"`

	// WindowStart is when the current window started
	WindowStart time.Time `json:"window_start"`

	// WindowEnd is when the current window ends
	WindowEnd time.Time `json:"window_end"`

	// ResetTime is when the rate limit will reset (same as WindowEnd)
	ResetTime time.Time `json:"reset_time"`

	// RetryAfter is the duration to wait before the next request (in seconds)
	RetryAfter time.Duration `json:"retry_after"`
}

// Backend represents a storage backend for rate limit data.
type Backend interface {
	// Increment increments the counter for a key within a time window.
	// Returns the new count and the time when the window started.
	Increment(ctx context.Context, key string, window time.Duration) (count int64, windowStart time.Time, err error)

	// Get retrieves the current count for a key within a time window.
	// Returns the count and the time when the window started.
	Get(ctx context.Context, key string, window time.Duration) (count int64, windowStart time.Time, err error)

	// Reset resets the counter for a key.
	Reset(ctx context.Context, key string) error

	// Close releases any resources held by the backend.
	Close() error
}

// Options configures a rate limiter.
type Options struct {
	// Backend specifies the storage backend to use.
	// If nil, an in-memory backend will be used.
	Backend Backend

	// DefaultLimit is the default number of requests allowed per window.
	DefaultLimit int

	// DefaultWindow is the default time window for rate limiting.
	DefaultWindow time.Duration

	// KeyPrefix is prepended to all keys stored in the backend.
	// Useful for namespacing when sharing a backend with other applications.
	KeyPrefix string

	// OnLimitExceeded is called when a rate limit is exceeded.
	// This can be used for logging, metrics, or custom actions.
	OnLimitExceeded func(key string, limit int, window time.Duration)

	// OnAllow is called when a request is allowed.
	// This can be used for logging or metrics.
	OnAllow func(key string, remaining int, window time.Duration)
}

// DefaultOptions returns a default configuration.
func DefaultOptions() Options {
	return Options{
		DefaultLimit:  100,
		DefaultWindow: time.Hour,
		KeyPrefix:     "ratelimit:",
	}
}

// New creates a new rate limiter with the given options.
func New(opts Options) Limiter {
	if opts.DefaultLimit <= 0 {
		opts.DefaultLimit = 100
	}
	if opts.DefaultWindow <= 0 {
		opts.DefaultWindow = time.Hour
	}
	if opts.KeyPrefix == "" {
		opts.KeyPrefix = "ratelimit:"
	}
	if opts.Backend == nil {
		opts.Backend = NewMemoryBackend()
	}

	return &slidingWindowLimiter{
		backend:       opts.Backend,
		defaultLimit:  opts.DefaultLimit,
		defaultWindow: opts.DefaultWindow,
		keyPrefix:     opts.KeyPrefix,
		onLimitExceeded: opts.OnLimitExceeded,
		onAllow:       opts.OnAllow,
		customLimits:  make(map[string]limitConfig),
	}
}

// limitConfig holds custom limit configuration for a specific key.
type limitConfig struct {
	limit  int
	window time.Duration
}

// slidingWindowLimiter implements the Limiter interface using a sliding window algorithm.
type slidingWindowLimiter struct {
	backend         Backend
	defaultLimit    int
	defaultWindow   time.Duration
	keyPrefix       string
	onLimitExceeded func(string, int, time.Duration)
	onAllow         func(string, int, time.Duration)
	customLimits    map[string]limitConfig
}

// Allow checks if a request for the given key is allowed.
func (l *slidingWindowLimiter) Allow(ctx context.Context, key string) bool {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks if N requests for the given key are allowed.
func (l *slidingWindowLimiter) AllowN(ctx context.Context, key string, n int) bool {
	if n <= 0 {
		return true
	}

	limit, window := l.getLimitAndWindow(key)
	prefixedKey := l.keyPrefix + key

	// Get current count without incrementing first
	currentCount, _, err := l.backend.Get(ctx, prefixedKey, window)
	if err != nil {
		// On error, be conservative and deny the request
		return false
	}

	// Check if adding N requests would exceed the limit
	if currentCount+int64(n) > int64(limit) {
		if l.onLimitExceeded != nil {
			l.onLimitExceeded(key, limit, window)
		}
		return false
	}

	// Increment by N
	for i := 0; i < n; i++ {
		_, _, err = l.backend.Increment(ctx, prefixedKey, window)
		if err != nil {
			// If we fail partway through, still deny to be safe
			return false
		}
	}

	// Calculate remaining for callback
	remaining := limit - int(currentCount) - n
	if remaining < 0 {
		remaining = 0
	}

	if l.onAllow != nil {
		l.onAllow(key, remaining, window)
	}

	return true
}

// Info returns detailed information about the current rate limit status for a key.
func (l *slidingWindowLimiter) Info(ctx context.Context, key string) (*LimitInfo, error) {
	limit, window := l.getLimitAndWindow(key)
	prefixedKey := l.keyPrefix + key

	count, windowStart, err := l.backend.Get(ctx, prefixedKey, window)
	if err != nil {
		return nil, err
	}

	used := int(count)
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}

	windowEnd := windowStart.Add(window)
	now := time.Now()
	
	var retryAfter time.Duration
	if remaining == 0 && windowEnd.After(now) {
		retryAfter = windowEnd.Sub(now)
	}

	return &LimitInfo{
		Key:         key,
		Limit:       limit,
		Remaining:   remaining,
		Used:        used,
		Window:      window,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		ResetTime:   windowEnd,
		RetryAfter:  retryAfter,
	}, nil
}

// Reset resets the rate limit counter for the given key.
func (l *slidingWindowLimiter) Reset(ctx context.Context, key string) error {
	prefixedKey := l.keyPrefix + key
	return l.backend.Reset(ctx, prefixedKey)
}

// SetLimit dynamically updates the rate limit for a specific key.
func (l *slidingWindowLimiter) SetLimit(ctx context.Context, key string, limit int, window time.Duration) error {
	if limit <= 0 {
		return errors.ErrInvalidLimit
	}
	if window <= 0 {
		return errors.ErrInvalidWindow
	}

	l.customLimits[key] = limitConfig{
		limit:  limit,
		window: window,
	}
	return nil
}

// Close releases any resources held by the limiter.
func (l *slidingWindowLimiter) Close() error {
	return l.backend.Close()
}

// getLimitAndWindow returns the limit and window for a given key.
// If a custom limit is set for the key, it returns that; otherwise, it returns the default.
func (l *slidingWindowLimiter) getLimitAndWindow(key string) (int, time.Duration) {
	if config, exists := l.customLimits[key]; exists {
		return config.limit, config.window
	}
	return l.defaultLimit, l.defaultWindow
}