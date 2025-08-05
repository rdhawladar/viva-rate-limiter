package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	DefaultBaseURL = "http://localhost:8090"
)

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

type APIKey struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tier        string                 `json:"tier"`
	RateLimit   int                    `json:"rate_limit"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Services  map[string]interface{} `json:"services"`
}

func NewAPIClient(baseURL string) *APIClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	
	return &APIClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *APIClient) makeRequest(method, endpoint string, headers map[string]string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}
	
	req, err := http.NewRequest(method, c.BaseURL+endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	
	return resp, nil
}

func (c *APIClient) CheckHealth() (*HealthResponse, error) {
	resp, err := c.makeRequest("GET", "/health", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}
	
	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}
	
	return &health, nil
}

func (c *APIClient) ValidateAPIKey(apiKey string) error {
	payload := map[string]string{
		"api_key": apiKey,
	}
	
	resp, err := c.makeRequest("POST", "/api/public/v1/rate-limit/validate", nil, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("validation failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (c *APIClient) CreateAPIKey(apiKey string, payload map[string]interface{}) (*APIKey, error) {
	headers := map[string]string{
		"X-API-Key": apiKey,
	}
	
	resp, err := c.makeRequest("POST", "/api/v1/api-keys", headers, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result APIKey
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

func (c *APIClient) ListAPIKeys(apiKey string) ([]APIKey, error) {
	headers := map[string]string{
		"X-API-Key": apiKey,
	}
	
	resp, err := c.makeRequest("GET", "/api/v1/api-keys", headers, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result []APIKey
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result, nil
}

func (c *APIClient) CheckRateLimit(payload map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest("POST", "/api/v1/rate-limit/check", nil, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result, nil
}

func printJSON(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_client.go <command> [args...]")
		fmt.Println("Commands:")
		fmt.Println("  health                     - Check API health")
		fmt.Println("  validate <api-key>         - Validate API key")
		fmt.Println("  list <api-key>             - List API keys")
		fmt.Println("  create <api-key> <name>    - Create new API key")
		fmt.Println("  rate-check <api-key-id>    - Check rate limit")
		os.Exit(1)
	}
	
	client := NewAPIClient(DefaultBaseURL)
	command := os.Args[1]
	
	switch command {
	case "health":
		health, err := client.CheckHealth()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ API is healthy!")
		printJSON(health)
		
	case "validate":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run test_client.go validate <api-key>")
			os.Exit(1)
		}
		apiKey := os.Args[2]
		
		err := client.ValidateAPIKey(apiKey)
		if err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ API key '%s' is valid!\n", apiKey)
		
	case "list":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run test_client.go list <api-key>")
			os.Exit(1)
		}
		apiKey := os.Args[2]
		
		keys, err := client.ListAPIKeys(apiKey)
		if err != nil {
			fmt.Printf("❌ List failed: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("✅ Found %d API keys:\n", len(keys))
		printJSON(keys)
		
	case "create":
		if len(os.Args) < 4 {
			fmt.Println("Usage: go run test_client.go create <api-key> <name>")
			os.Exit(1)
		}
		apiKey := os.Args[2]
		name := os.Args[3]
		
		payload := map[string]interface{}{
			"name":        name,
			"description": "Created by test client",
			"tier":        "standard",
			"rate_limit":  5000,
		}
		
		result, err := client.CreateAPIKey(apiKey, payload)
		if err != nil {
			fmt.Printf("❌ Create failed: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("✅ Created API key:\n")
		printJSON(result)
		
	case "rate-check":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run test_client.go rate-check <api-key-id>")
			os.Exit(1)
		}
		apiKeyID := os.Args[2]
		
		payload := map[string]interface{}{
			"api_key_id": apiKeyID,
			"endpoint":   "/test",
			"method":     "GET",
		}
		
		result, err := client.CheckRateLimit(payload)
		if err != nil {
			fmt.Printf("❌ Rate check failed: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("✅ Rate limit check result:\n")
		printJSON(result)
		
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}