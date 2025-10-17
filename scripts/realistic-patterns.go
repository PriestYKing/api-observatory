package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// Patterns that the analytics engine should detect
type TrafficPattern struct {
	Name        string
	Description string
	Execute     func(string)
}

func main() {
	ingestURL := "http://localhost:8081/api/ingest"

	patterns := []TrafficPattern{
		{
			Name:        "Duplicate Requests",
			Description: "Generate identical requests to trigger duplicate detection",
			Execute: func(url string) {
				generateDuplicates(url, 20)
			},
		},
		{
			Name:        "Cacheable GET Requests",
			Description: "Repeated GET requests that should be cached",
			Execute: func(url string) {
				generateCacheableRequests(url, 50)
			},
		},
		{
			Name:        "Cost Spike",
			Description: "Sudden increase in expensive API calls",
			Execute: func(url string) {
				generateCostSpike(url, 100)
			},
		},
		{
			Name:        "Error Surge",
			Description: "Increased error rate to trigger anomaly detection",
			Execute: func(url string) {
				generateErrorSurge(url, 30)
			},
		},
		{
			Name:        "Rate Limit Abuse",
			Description: "Rapid consecutive calls to same endpoint",
			Execute: func(url string) {
				generateRateLimitAbuse(url, 50)
			},
		},
	}

	fmt.Println("🎯 Realistic Pattern Generator")
	fmt.Println()

	for i, pattern := range patterns {
		fmt.Printf("%d. %s\n", i+1, pattern.Name)
		fmt.Printf("   %s\n", pattern.Description)
		fmt.Println()

		log.Printf("Executing: %s", pattern.Name)
		pattern.Execute(ingestURL)
		log.Printf("✓ Completed: %s\n", pattern.Name)

		time.Sleep(2 * time.Second)
	}

	fmt.Println("✅ All patterns generated!")
	fmt.Println("Check your dashboard for detected issues.")
}

func generateDuplicates(url string, count int) {
	// Same request repeated multiple times
	duplicateReq := createRequest("Stripe", "/v1/charges", "POST", 200, 150)

	for i := 0; i < count; i++ {
		sendRequest(url, duplicateReq)
		time.Sleep(50 * time.Millisecond)
	}
}

func generateCacheableRequests(url string, count int) {
	endpoints := []string{
		"/v1/products/list",
		"/v1/customers/search",
		"/v1/catalog/items",
	}

	for i := 0; i < count; i++ {
		endpoint := endpoints[rand.Intn(len(endpoints))]
		req := createRequest("Stripe", endpoint, "GET", 200, 100)
		sendRequest(url, req)
		time.Sleep(100 * time.Millisecond)
	}
}

func generateCostSpike(url string, count int) {
	// Expensive OpenAI requests
	for i := 0; i < count; i++ {
		req := createRequest("OpenAI", "/v1/chat/completions", "POST", 200, 2000)
		req.RequestSizeBytes = 10000 + rand.Intn(40000)
		req.ResponseSizeBytes = 50000 + rand.Intn(100000)
		sendRequest(url, req)
		time.Sleep(200 * time.Millisecond)
	}
}

func generateErrorSurge(url string, count int) {
	errorCodes := []int{400, 401, 429, 500, 502, 503}

	for i := 0; i < count; i++ {
		statusCode := errorCodes[rand.Intn(len(errorCodes))]
		req := createRequest("SendGrid", "/v3/mail/send", "POST", statusCode, 300)
		req.ErrorMessage = fmt.Sprintf("Error %d occurred", statusCode)
		sendRequest(url, req)
		time.Sleep(100 * time.Millisecond)
	}
}

func generateRateLimitAbuse(url string, count int) {
	// Rapid fire to same endpoint
	endpoint := "/v1/api/resource/123"

	for i := 0; i < count; i++ {
		req := createRequest("Twilio", endpoint, "GET", 429, 50)
		req.ErrorMessage = "Rate limit exceeded"
		sendRequest(url, req)
		time.Sleep(20 * time.Millisecond) // Very fast
	}
}

type APIRequest struct {
	RequestID         string            `json:"request_id"`
	OrganizationID    string            `json:"organization_id"`
	Timestamp         int64             `json:"timestamp"`
	Provider          string            `json:"provider"`
	Endpoint          string            `json:"endpoint"`
	Method            string            `json:"method"`
	StatusCode        int               `json:"status_code"`
	LatencyMS         int               `json:"latency_ms"`
	RequestSizeBytes  int               `json:"request_size_bytes"`
	ResponseSizeBytes int               `json:"response_size_bytes"`
	ErrorMessage      string            `json:"error_message"`
	Metadata          map[string]string `json:"metadata"`
}

func createRequest(provider, endpoint, method string, statusCode, latency int) APIRequest {
	return APIRequest{
		RequestID:         fmt.Sprintf("req_%d", time.Now().UnixNano()),
		OrganizationID:    "1",
		Timestamp:         time.Now().UnixMilli(),
		Provider:          provider,
		Endpoint:          endpoint,
		Method:            method,
		StatusCode:        statusCode,
		LatencyMS:         latency,
		RequestSizeBytes:  rand.Intn(5000) + 100,
		ResponseSizeBytes: rand.Intn(10000) + 500,
		ErrorMessage:      "",
		Metadata: map[string]string{
			"user_id": fmt.Sprintf("user_%d", rand.Intn(100)),
		},
	}
}

func sendRequest(url string, req APIRequest) {
	data, _ := json.Marshal(req)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()
}
