// Package main demonstrates using the rate limiter in a web server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rdhawladar/viva-rate-limiter/pkg/ratelimit"
)

type server struct {
	limiter ratelimit.Limiter
}

type response struct {
	Message   string                `json:"message"`
	Timestamp time.Time             `json:"timestamp"`
	RateLimit *ratelimit.LimitInfo  `json:"rate_limit,omitempty"`
	Error     string                `json:"error,omitempty"`
}

func main() {
	fmt.Println("🌐 Web Server Rate Limiting Example")
	fmt.Println("===================================")

	// Create rate limiter with different limits for different endpoints
	opts := ratelimit.DefaultOptions()
	opts.DefaultLimit = 10
	opts.DefaultWindow = time.Minute
	opts.OnLimitExceeded = func(key string, limit int, window time.Duration) {
		log.Printf("Rate limit exceeded for %s: %d per %v", key, limit, window)
	}

	limiter := ratelimit.New(opts)
	defer limiter.Close()

	// Set custom limits for different API tiers
	ctx := context.Background()
	limiter.SetLimit(ctx, "api:free", 5, time.Minute)       // Free tier: 5/min
	limiter.SetLimit(ctx, "api:premium", 50, time.Minute)   // Premium: 50/min
	limiter.SetLimit(ctx, "api:enterprise", 500, time.Minute) // Enterprise: 500/min

	s := &server{limiter: limiter}

	// Routes
	http.HandleFunc("/", s.handleHome)
	http.HandleFunc("/api/free", s.rateLimitMiddleware("api:free")(s.handleAPI))
	http.HandleFunc("/api/premium", s.rateLimitMiddleware("api:premium")(s.handleAPI))
	http.HandleFunc("/api/enterprise", s.rateLimitMiddleware("api:enterprise")(s.handleAPI))
	http.HandleFunc("/api/user/", s.handleUserAPI)
	http.HandleFunc("/status", s.handleStatus)

	port := 8092
	fmt.Printf("🚀 Starting server on http://localhost:%d\n", port)
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  GET  /                    - Home page")
	fmt.Println("  GET  /api/free            - Free tier API (5 req/min)")
	fmt.Println("  GET  /api/premium         - Premium API (50 req/min)")
	fmt.Println("  GET  /api/enterprise      - Enterprise API (500 req/min)")
	fmt.Println("  GET  /api/user/{id}       - User-specific API (10 req/min per user)")
	fmt.Println("  GET  /status              - Rate limit status for all tiers")
	fmt.Println("\nTry making multiple requests to see rate limiting in action!")
	fmt.Printf("Example: curl http://localhost:%d/api/free\n\n", port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func (s *server) handleHome(w http.ResponseWriter, r *http.Request) {
	resp := response{
		Message:   "Welcome to the Rate Limited API Demo!",
		Timestamp: time.Now(),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleAPI(w http.ResponseWriter, r *http.Request) {
	resp := response{
		Message:   "API request successful!",
		Timestamp: time.Now(),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleUserAPI(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	userID := r.URL.Path[len("/api/user/"):]
	if userID == "" {
		userID = "anonymous"
	}

	key := "user:" + userID
	ctx := r.Context()

	// Check rate limit
	allowed := s.limiter.Allow(ctx, key)
	if !allowed {
		info, _ := s.limiter.Info(ctx, key)
		resp := response{
			Error:     "Rate limit exceeded",
			Timestamp: time.Now(),
			RateLimit: info,
		}
		s.writeJSON(w, http.StatusTooManyRequests, resp)
		return
	}

	// Get rate limit info for response headers
	info, err := s.limiter.Info(ctx, key)
	if err != nil {
		log.Printf("Error getting rate limit info: %v", err)
	}

	// Set rate limit headers
	if info != nil {
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))
	}

	resp := response{
		Message:   fmt.Sprintf("Hello, user %s!", userID),
		Timestamp: time.Now(),
		RateLimit: info,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	tiers := []string{"api:free", "api:premium", "api:enterprise"}
	status := make(map[string]*ratelimit.LimitInfo)
	
	for _, tier := range tiers {
		info, err := s.limiter.Info(ctx, tier)
		if err != nil {
			log.Printf("Error getting info for %s: %v", tier, err)
			continue
		}
		status[tier] = info
	}
	
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"timestamp": time.Now(),
		"status":    status,
	})
}

func (s *server) rateLimitMiddleware(key string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Check rate limit
			allowed := s.limiter.Allow(ctx, key)
			if !allowed {
				info, _ := s.limiter.Info(ctx, key)
				
				// Set rate limit headers
				if info != nil {
					w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
					w.Header().Set("X-RateLimit-Remaining", "0")
					w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))
					if info.RetryAfter > 0 {
						w.Header().Set("Retry-After", fmt.Sprintf("%d", int(info.RetryAfter.Seconds())))
					}
				}

				resp := response{
					Error:     "Rate limit exceeded",
					Timestamp: time.Now(),
					RateLimit: info,
				}
				s.writeJSON(w, http.StatusTooManyRequests, resp)
				return
			}

			// Get rate limit info for response headers
			info, err := s.limiter.Info(ctx, key)
			if err != nil {
				log.Printf("Error getting rate limit info: %v", err)
			}

			// Set rate limit headers
			if info != nil {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))
			}

			next(w, r)
		}
	}
}

func (s *server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}