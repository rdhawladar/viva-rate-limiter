package cache

import (
	"context"
	"time"

	"github.com/rdhawladar/viva-rate-limiter/internal/services"
)

// cacheService implements the services.CacheService interface
type cacheService struct {
	redis *RedisClient
}

// NewCacheService creates a new cache service
func NewCacheService(redis *RedisClient) services.CacheService {
	return &cacheService{
		redis: redis,
	}
}

// GetCounter retrieves a counter value
func (c *cacheService) GetCounter(ctx context.Context, key string) (int64, error) {
	return c.redis.GetCounter(ctx, key)
}

// SetCounter sets a counter value with expiration
func (c *cacheService) SetCounter(ctx context.Context, key string, value int64, expiration time.Duration) error {
	return c.redis.SetCounter(ctx, key, value, expiration)
}

// IncrementCounter increments a counter and returns the new value
func (c *cacheService) IncrementCounter(ctx context.Context, key string, delta int64, expiration time.Duration) (int64, error) {
	return c.redis.IncrementCounter(ctx, key, delta, expiration)
}

// DeleteKey removes a key
func (c *cacheService) DeleteKey(ctx context.Context, key string) error {
	return c.redis.DeleteKey(ctx, key)
}

// Get retrieves a value by key
func (c *cacheService) Get(ctx context.Context, key string) (string, error) {
	return c.redis.Get(ctx, key)
}

// Set stores a key-value pair with expiration
func (c *cacheService) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.redis.Set(ctx, key, value, expiration)
}

// Delete removes a key
func (c *cacheService) Delete(ctx context.Context, key string) error {
	return c.redis.Delete(ctx, key)
}

// Exists checks if a key exists
func (c *cacheService) Exists(ctx context.Context, key string) (bool, error) {
	return c.redis.Exists(ctx, key)
}