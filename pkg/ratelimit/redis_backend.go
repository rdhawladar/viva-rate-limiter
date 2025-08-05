package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/viva/rate-limiter/pkg/errors"
)

// RedisBackend implements the Backend interface using Redis.
// This is suitable for distributed applications and production use.
type RedisBackend struct {
	client redis.UniversalClient
	closed bool
}

// RedisConfig contains configuration for Redis backend.
type RedisConfig struct {
	// Addresses of Redis servers. For single instance, use one address.
	// For cluster, provide multiple addresses.
	Addresses []string

	// Password for Redis authentication (optional).
	Password string

	// Database number (only used for single instance, ignored in cluster mode).
	DB int

	// PoolSize is the maximum number of socket connections.
	PoolSize int

	// MinIdleConns is the minimum number of idle connections.
	MinIdleConns int

	// MaxRetries is the maximum number of retries before giving up.
	MaxRetries int

	// DialTimeout is the timeout for establishing a connection.
	DialTimeout time.Duration

	// ReadTimeout is the timeout for socket reads.
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for socket writes.
	WriteTimeout time.Duration

	// ClusterMode enables Redis cluster mode.
	ClusterMode bool
}

// DefaultRedisConfig returns a default Redis configuration.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addresses:    []string{"localhost:6379"},
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// NewRedisBackend creates a new Redis backend with the given configuration.
func NewRedisBackend(config RedisConfig) (*RedisBackend, error) {
	var client redis.UniversalClient

	if config.ClusterMode {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Addresses,
			Password:     config.Password,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
			MaxRetries:   config.MaxRetries,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		})
	} else {
		addr := "localhost:6379"
		if len(config.Addresses) > 0 {
			addr = config.Addresses[0]
		}

		client = redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     config.Password,
			DB:           config.DB,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
			MaxRetries:   config.MaxRetries,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		})
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisBackend{
		client: client,
	}, nil
}

// NewRedisBackendFromClient creates a Redis backend from an existing Redis client.
func NewRedisBackendFromClient(client redis.UniversalClient) *RedisBackend {
	return &RedisBackend{
		client: client,
	}
}

// Increment increments the counter for a key within a time window using a sliding window approach.
func (r *RedisBackend) Increment(ctx context.Context, key string, window time.Duration) (int64, time.Time, error) {
	if ctx.Err() != nil {
		return 0, time.Time{}, ctx.Err()
	}

	if r.closed {
		return 0, time.Time{}, errors.ErrBackendClosed
	}

	now := time.Now()
	windowStart := now
	
	// Use a Lua script for atomic sliding window operation
	luaScript := `
		local key = KEYS[1]
		local window_ms = tonumber(ARGV[1])
		local now_ms = tonumber(ARGV[2])
		local window_start_ms = now_ms
		
		-- Remove expired entries (older than window)
		local min_timestamp = now_ms - window_ms
		redis.call('ZREMRANGEBYSCORE', key, '-inf', min_timestamp)
		
		-- Add current request with timestamp as score
		redis.call('ZADD', key, now_ms, now_ms)
		
		-- Set expiration to window duration + buffer
		redis.call('EXPIRE', key, math.ceil(window_ms / 1000) + 10)
		
		-- Count requests in current window
		local count = redis.call('ZCARD', key)
		
		-- Get the oldest timestamp to calculate window start
		local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
		if oldest[2] then
			window_start_ms = tonumber(oldest[2])
		end
		
		return {count, window_start_ms}
	`

	windowMs := window.Milliseconds()
	nowMs := now.UnixMilli()

	result, err := r.client.Eval(ctx, luaScript, []string{key}, windowMs, nowMs).Result()
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("Redis increment failed: %w", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 2 {
		return 0, time.Time{}, fmt.Errorf("unexpected Redis response format")
	}

	count, err := strconv.ParseInt(fmt.Sprintf("%v", resultSlice[0]), 10, 64)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to parse count: %w", err)
	}

	windowStartMs, err := strconv.ParseInt(fmt.Sprintf("%v", resultSlice[1]), 10, 64)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to parse window start: %w", err)
	}

	windowStart = time.UnixMilli(windowStartMs)
	return count, windowStart, nil
}

// Get retrieves the current count for a key within a time window.
func (r *RedisBackend) Get(ctx context.Context, key string, window time.Duration) (int64, time.Time, error) {
	if ctx.Err() != nil {
		return 0, time.Time{}, ctx.Err()
	}

	if r.closed {
		return 0, time.Time{}, errors.ErrBackendClosed
	}

	now := time.Now()
	
	// Use a Lua script for atomic sliding window read
	luaScript := `
		local key = KEYS[1]
		local window_ms = tonumber(ARGV[1])
		local now_ms = tonumber(ARGV[2])
		
		-- Remove expired entries (older than window)
		local min_timestamp = now_ms - window_ms
		redis.call('ZREMRANGEBYSCORE', key, '-inf', min_timestamp)
		
		-- Count requests in current window
		local count = redis.call('ZCARD', key)
		
		-- Get the oldest timestamp to calculate window start
		local window_start_ms = now_ms
		local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
		if oldest[2] then
			window_start_ms = tonumber(oldest[2])
		end
		
		return {count, window_start_ms}
	`

	windowMs := window.Milliseconds()
	nowMs := now.UnixMilli()

	result, err := r.client.Eval(ctx, luaScript, []string{key}, windowMs, nowMs).Result()
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("Redis get failed: %w", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 2 {
		return 0, time.Time{}, fmt.Errorf("unexpected Redis response format")
	}

	count, err := strconv.ParseInt(fmt.Sprintf("%v", resultSlice[0]), 10, 64)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to parse count: %w", err)
	}

	windowStartMs, err := strconv.ParseInt(fmt.Sprintf("%v", resultSlice[1]), 10, 64)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to parse window start: %w", err)
	}

	windowStart := time.UnixMilli(windowStartMs)
	return count, windowStart, nil
}

// Reset resets the counter for a key.
func (r *RedisBackend) Reset(ctx context.Context, key string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return errors.ErrBackendClosed
	}

	return r.client.Del(ctx, key).Err()
}

// Close closes the Redis client and releases resources.
func (r *RedisBackend) Close() error {
	if r.closed {
		return nil
	}

	r.closed = true
	return r.client.Close()
}

// Ping tests the connection to Redis.
func (r *RedisBackend) Ping(ctx context.Context) error {
	if r.closed {
		return errors.ErrBackendClosed
	}

	return r.client.Ping(ctx).Err()
}

// Info returns information about the Redis server.
func (r *RedisBackend) Info(ctx context.Context) (string, error) {
	if r.closed {
		return "", errors.ErrBackendClosed
	}

	return r.client.Info(ctx).Result()
}