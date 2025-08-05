// Package main demonstrates basic usage of the rate limiter.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/viva/rate-limiter/pkg/ratelimit"
)

func main() {
	fmt.Println("🚀 Basic Rate Limiter Example")
	fmt.Println("=============================")

	// Create a rate limiter with default options (in-memory backend)
	limiter := ratelimit.New(ratelimit.DefaultOptions())
	defer limiter.Close()

	ctx := context.Background()
	key := "user123"

	fmt.Printf("Testing rate limiter for key: %s\n", key)
	fmt.Printf("Default limit: 100 requests per hour\n\n")

	// Test allowing requests
	for i := 1; i <= 5; i++ {
		allowed := limiter.Allow(ctx, key)
		fmt.Printf("Request %d: %s\n", i, allowedStatus(allowed))
		
		if i == 3 {
			// Get detailed info
			info, err := limiter.Info(ctx, key)
			if err != nil {
				log.Printf("Error getting info: %v", err)
			} else {
				fmt.Printf("  📊 Usage: %d/%d (remaining: %d)\n", 
					info.Used, info.Limit, info.Remaining)
			}
		}
	}

	fmt.Println("\n📈 Custom Limits Demo")
	fmt.Println("=====================")

	// Set a custom limit for a specific key
	customKey := "premium-user456"
	err := limiter.SetLimit(ctx, customKey, 10, time.Minute)
	if err != nil {
		log.Fatalf("Failed to set custom limit: %v", err)
	}

	fmt.Printf("Set custom limit for %s: 10 requests per minute\n", customKey)

	// Test the custom limit
	for i := 1; i <= 12; i++ {
		allowed := limiter.Allow(ctx, customKey)
		status := allowedStatus(allowed)
		if !allowed {
			status += " ❌"
		}
		fmt.Printf("Request %d: %s\n", i, status)
		
		// Show remaining at key intervals
		if i%5 == 0 || !allowed {
			info, err := limiter.Info(ctx, customKey)
			if err != nil {
				log.Printf("Error getting info: %v", err)
			} else {
				fmt.Printf("  📊 Usage: %d/%d (remaining: %d)\n", 
					info.Used, info.Limit, info.Remaining)
				if info.RetryAfter > 0 {
					fmt.Printf("  ⏰ Retry after: %v\n", info.RetryAfter.Round(time.Second))
				}
			}
		}
	}

	fmt.Println("\n🔄 Reset Demo")
	fmt.Println("=============")

	// Reset the rate limit
	err = limiter.Reset(ctx, customKey)
	if err != nil {
		log.Fatalf("Failed to reset limit: %v", err)
	}
	fmt.Printf("Reset rate limit for %s\n", customKey)

	// Test after reset
	allowed := limiter.Allow(ctx, customKey)
	fmt.Printf("First request after reset: %s\n", allowedStatus(allowed))

	info, err := limiter.Info(ctx, customKey)
	if err != nil {
		log.Printf("Error getting info: %v", err)
	} else {
		fmt.Printf("📊 Usage after reset: %d/%d\n", info.Used, info.Limit)
	}

	fmt.Println("\n✅ Example completed successfully!")
}

func allowedStatus(allowed bool) string {
	if allowed {
		return "✅ ALLOWED"
	}
	return "🚫 DENIED"
}