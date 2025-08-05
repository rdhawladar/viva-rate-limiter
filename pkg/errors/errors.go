// Package errors provides custom error types for the rate limiter.
package errors

import "errors"

// Common rate limiter errors.
var (
	// ErrInvalidLimit is returned when an invalid limit is provided.
	ErrInvalidLimit = errors.New("invalid limit: must be greater than 0")

	// ErrInvalidWindow is returned when an invalid time window is provided.
	ErrInvalidWindow = errors.New("invalid window: must be greater than 0")

	// ErrKeyNotFound is returned when a key is not found in the backend.
	ErrKeyNotFound = errors.New("key not found")

	// ErrBackendUnavailable is returned when the backend is unavailable.
	ErrBackendUnavailable = errors.New("backend unavailable")

	// ErrInvalidKey is returned when an invalid key is provided.
	ErrInvalidKey = errors.New("invalid key: cannot be empty")

	// ErrLimitExceeded is returned when a rate limit is exceeded.
	ErrLimitExceeded = errors.New("rate limit exceeded")

	// ErrBackendClosed is returned when trying to use a closed backend.
	ErrBackendClosed = errors.New("backend is closed")
)