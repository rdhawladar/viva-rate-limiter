package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rdhawladar/viva-rate-limiter/internal/config"
)

// RedisClient wraps redis.Client with additional functionality
type RedisClient struct {
	client *redis.Client
	config *config.RedisConfig
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.RedisConfig) (*RedisClient, error) {
	// Use the first address if multiple are provided
	addr := "localhost:6379" // default
	if len(cfg.Addresses) > 0 {
		addr = cfg.Addresses[0]
	}
	
	options := &redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
	}

	client := redis.NewClient(options)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
		config: cfg,
	}, nil
}

// GetClient returns the underlying Redis client
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Get retrieves a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	result := r.client.Get(ctx, key)
	if result.Err() == redis.Nil {
		return "", fmt.Errorf("key not found")
	}
	return result.Result()
}

// Set stores a key-value pair with expiration
func (r *RedisClient) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Delete removes a key
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetCounter retrieves a counter value
func (r *RedisClient) GetCounter(ctx context.Context, key string) (int64, error) {
	result := r.client.Get(ctx, key)
	if result.Err() == redis.Nil {
		return 0, fmt.Errorf("counter not found")
	}
	if result.Err() != nil {
		return 0, result.Err()
	}

	value, err := result.Int64()
	if err != nil {
		return 0, fmt.Errorf("invalid counter value: %w", err)
	}
	return value, nil
}

// SetCounter sets a counter value with expiration
func (r *RedisClient) SetCounter(ctx context.Context, key string, value int64, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// IncrementCounter increments a counter and returns the new value
func (r *RedisClient) IncrementCounter(ctx context.Context, key string, delta int64, expiration time.Duration) (int64, error) {
	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()
	incrCmd := pipe.IncrBy(ctx, key, delta)
	pipe.Expire(ctx, key, expiration)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}

	return incrCmd.Val(), nil
}

// DecrementCounter decrements a counter and returns the new value
func (r *RedisClient) DecrementCounter(ctx context.Context, key string, delta int64, expiration time.Duration) (int64, error) {
	return r.IncrementCounter(ctx, key, -delta, expiration)
}

// DeleteKey removes a key (alias for Delete for interface compatibility)
func (r *RedisClient) DeleteKey(ctx context.Context, key string) error {
	return r.Delete(ctx, key)
}

// SetHash stores a hash field
func (r *RedisClient) SetHash(ctx context.Context, key, field, value string) error {
	return r.client.HSet(ctx, key, field, value).Err()
}

// GetHash retrieves a hash field
func (r *RedisClient) GetHash(ctx context.Context, key, field string) (string, error) {
	result := r.client.HGet(ctx, key, field)
	if result.Err() == redis.Nil {
		return "", fmt.Errorf("hash field not found")
	}
	return result.Result()
}

// GetAllHash retrieves all fields and values in a hash
func (r *RedisClient) GetAllHash(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// DeleteHashField removes a hash field
func (r *RedisClient) DeleteHashField(ctx context.Context, key, field string) error {
	return r.client.HDel(ctx, key, field).Err()
}

// SetList adds an item to the end of a list
func (r *RedisClient) SetList(ctx context.Context, key, value string) error {
	return r.client.RPush(ctx, key, value).Err()
}

// GetList retrieves all items from a list
func (r *RedisClient) GetList(ctx context.Context, key string) ([]string, error) {
	return r.client.LRange(ctx, key, 0, -1).Result()
}

// GetListRange retrieves a range of items from a list
func (r *RedisClient) GetListRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, key, start, stop).Result()
}

// SetExpire sets expiration for a key
func (r *RedisClient) SetExpire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// GetTTL gets the time to live for a key
func (r *RedisClient) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// FlushDB clears all keys in the current database
func (r *RedisClient) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// Keys returns all keys matching a pattern
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

// SlidingWindowRateLimit implements sliding window rate limiting
func (r *RedisClient) SlidingWindowRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, error) {
	now := time.Now()
	windowStart := now.Add(-window)
	
	// Use Lua script for atomic operations
	script := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_seconds = tonumber(ARGV[4])
		
		-- Remove expired entries
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)
		
		-- Count current entries
		local current = redis.call('ZCARD', key)
		
		if current < limit then
			-- Add current request
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, window_seconds)
			return {1, limit - current - 1}
		else
			return {0, 0}
		end
	`
	
	result, err := r.client.Eval(ctx, script, []string{key}, 
		windowStart.Unix(), now.Unix(), limit, int(window.Seconds())).Result()
	if err != nil {
		return false, 0, fmt.Errorf("sliding window rate limit failed: %w", err)
	}
	
	results := result.([]interface{})
	allowed := results[0].(int64) == 1
	remaining := results[1].(int64)
	
	return allowed, remaining, nil
}

// FixedWindowRateLimit implements fixed window rate limiting
func (r *RedisClient) FixedWindowRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, error) {
	// Create window-specific key
	now := time.Now()
	windowStart := now.Truncate(window)
	windowKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())
	
	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, window)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("fixed window rate limit failed: %w", err)
	}
	
	current := incrCmd.Val()
	allowed := current <= limit
	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}
	
	return allowed, remaining, nil
}

// TokenBucketRateLimit implements token bucket rate limiting
func (r *RedisClient) TokenBucketRateLimit(ctx context.Context, key string, capacity, refillRate int64, refillPeriod time.Duration) (bool, int64, error) {
	now := time.Now()
	
	// Lua script for atomic token bucket operations
	script := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local refill_period = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1]) or capacity
		local last_refill = tonumber(bucket[2]) or now
		
		-- Calculate tokens to add
		local time_passed = now - last_refill
		local tokens_to_add = math.floor(time_passed / refill_period * refill_rate)
		tokens = math.min(capacity, tokens + tokens_to_add)
		
		if tokens >= 1 then
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, refill_period * 2)
			return {1, tokens}
		else
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, refill_period * 2)
			return {0, tokens}
		end
	`
	
	result, err := r.client.Eval(ctx, script, []string{key}, 
		capacity, refillRate, refillPeriod.Seconds(), now.Unix()).Result()
	if err != nil {
		return false, 0, fmt.Errorf("token bucket rate limit failed: %w", err)
	}
	
	results := result.([]interface{})
	allowed := results[0].(int64) == 1
	remaining := results[1].(int64)
	
	return allowed, remaining, nil
}

// Health checks the Redis connection health
func (r *RedisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Info returns Redis server information
func (r *RedisClient) Info(ctx context.Context) (string, error) {
	return r.client.Info(ctx).Result()
}

// Stats returns connection pool statistics
func (r *RedisClient) Stats() *redis.PoolStats {
	return r.client.PoolStats()
}

// BatchGet retrieves multiple keys in a single operation
func (r *RedisClient) BatchGet(ctx context.Context, keys []string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}
	
	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	
	// Convert []interface{} to []string
	result := make([]string, len(values))
	for i, v := range values {
		if v != nil {
			result[i] = v.(string)
		} else {
			result[i] = ""
		}
	}
	
	return result, nil
}

// BatchSet sets multiple key-value pairs in a single operation
func (r *RedisClient) BatchSet(ctx context.Context, pairs map[string]string, expiration time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}
	
	pipe := r.client.Pipeline()
	for key, value := range pairs {
		pipe.Set(ctx, key, value, expiration)
	}
	
	_, err := pipe.Exec(ctx)
	return err
}

// ScanKeys scans for keys matching a pattern
func (r *RedisClient) ScanKeys(ctx context.Context, pattern string, count int64) ([]string, error) {
	var keys []string
	var cursor uint64
	
	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = r.client.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			return nil, err
		}
		
		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}
	
	return keys, nil
}