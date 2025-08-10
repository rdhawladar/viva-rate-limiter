package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit"
)

func main() {
	// Create a memory backend
	backend := ratelimit.NewMemoryBackend()
	
	ctx := context.Background()
	key := "test-user"
	window := time.Minute
	limit := int64(10)
	
	fmt.Println("Testing rate limiter with limit of 10 requests per minute...")
	
	// Test the rate limiter
	for i := 1; i <= 12; i++ {
		count, windowStart, err := backend.Increment(ctx, key, window)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		if count <= limit {
			fmt.Printf("Request %d: ALLOWED (count: %d, window started: %v)\n", 
				i, count, windowStart.Format("15:04:05"))
		} else {
			fmt.Printf("Request %d: BLOCKED - Rate limit exceeded (count: %d)\n", i, count)
		}
	}
	
	// Get current status
	currentCount, windowStart, err := backend.Get(ctx, key, window)
	if err != nil {
		fmt.Printf("Error getting status: %v\n", err)
	} else {
		fmt.Printf("\nCurrent status: %d/%d requests used (window started: %v)\n", 
			currentCount, limit, windowStart.Format("15:04:05"))
	}
	
	fmt.Println("\n✅ Package imported and working successfully!")
	fmt.Println("You can now use: go get github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit@v0.3.0")
}