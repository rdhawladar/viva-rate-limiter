// Package main demonstrates Redis-backed rate limiting.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit"
)

func main() {
	fmt.Println("🔴 Redis Rate Limiter Example")
	fmt.Println("=============================")

	// Configure Redis backend
	redisConfig := ratelimit.DefaultRedisConfig()
	redisConfig.Addresses = []string{"localhost:6379"} // Adjust as needed
	
	redisBackend, err := ratelimit.NewRedisBackend(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create Redis backend: %v", err)
	}
	defer redisBackend.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisBackend.Ping(ctx); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	fmt.Println("✅ Connected to Redis successfully")

	// Create rate limiter with Redis backend
	opts := ratelimit.DefaultOptions()
	opts.Backend = redisBackend
	opts.DefaultLimit = 5
	opts.DefaultWindow = time.Minute
	opts.KeyPrefix = "example:"

	// Add callbacks for logging
	opts.OnAllow = func(key string, remaining int, window time.Duration) {
		fmt.Printf("  ✅ Request allowed for %s (remaining: %d)\n", key, remaining)
	}
	opts.OnLimitExceeded = func(key string, limit int, window time.Duration) {
		fmt.Printf("  🚫 Rate limit exceeded for %s (limit: %d per %v)\n", key, limit, window)
	}

	limiter := ratelimit.New(opts)
	defer limiter.Close()

	fmt.Printf("Rate limiter configured: %d requests per %v\n\n", opts.DefaultLimit, opts.DefaultWindow)

	// Test with multiple keys to show distributed nature
	keys := []string{"user1", "user2", "user3"}

	fmt.Println("🔄 Testing multiple users simultaneously")
	fmt.Println("========================================")

	for round := 1; round <= 3; round++ {
		fmt.Printf("\n--- Round %d ---\n", round)
		
		for _, key := range keys {
			allowed := limiter.Allow(ctx, key)
			
			// Get detailed info
			info, err := limiter.Info(ctx, key)
			if err != nil {
				log.Printf("Error getting info for %s: %v", key, err)
				continue
			}
			
			status := "✅ ALLOWED"
			if !allowed {
				status = "🚫 DENIED"
			}
			
			fmt.Printf("%s: %s (used: %d/%d)\n", 
				key, status, info.Used, info.Limit)
		}
	}

	fmt.Println("\n⚡ Rapid fire test (should hit rate limit)")
	fmt.Println("=========================================")

	rapidKey := "rapid-user"
	for i := 1; i <= 8; i++ {
		allowed := limiter.Allow(ctx, rapidKey)
		status := "✅"
		if !allowed {
			status = "🚫"
		}
		fmt.Printf("Request %d: %s\n", i, status)
		
		// Small delay to make it more readable
		time.Sleep(100 * time.Millisecond)
	}

	// Show final state
	fmt.Println("\n📊 Final Status")
	fmt.Println("===============")
	
	allKeys := append(keys, rapidKey)
	for _, key := range allKeys {
		info, err := limiter.Info(ctx, key)
		if err != nil {
			log.Printf("Error getting info for %s: %v", key, err)
			continue
		}
		
		fmt.Printf("%s: %d/%d used", key, info.Used, info.Limit)
		if info.RetryAfter > 0 {
			fmt.Printf(" (retry after: %v)", info.RetryAfter.Round(time.Second))
		}
		fmt.Println()
	}

	fmt.Println("\n💾 Data persisted in Redis - try running this example again")
	fmt.Println("   to see that the counts persist across application restarts!")
	fmt.Println("\n✅ Redis example completed successfully!")
}