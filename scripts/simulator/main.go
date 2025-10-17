package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	IngestURL   string
	Duration    time.Duration
	RPS         int
	Concurrency int
	OrgID       string
	Mode        string
}

type APIProvider struct {
	Name      string
	Weight    int
	Endpoints []string
	Methods   []string
	ErrorRate float64
}

var providers = []APIProvider{
	{
		Name:   "OpenAI",
		Weight: 40,
		Endpoints: []string{
			"/v1/chat/completions",
			"/v1/completions",
			"/v1/embeddings",
			"/v1/models",
		},
		Methods:   []string{"POST", "GET"},
		ErrorRate: 0.02,
	},
	{
		Name:   "Stripe",
		Weight: 25,
		Endpoints: []string{
			"/v1/charges",
			"/v1/customers",
			"/v1/payment_intents",
			"/v1/subscriptions",
			"/v1/invoices",
		},
		Methods:   []string{"POST", "GET", "PUT"},
		ErrorRate: 0.01,
	},
	{
		Name:   "SendGrid",
		Weight: 15,
		Endpoints: []string{
			"/v3/mail/send",
			"/v3/templates",
			"/v3/campaigns",
		},
		Methods:   []string{"POST", "GET"},
		ErrorRate: 0.03,
	},
	{
		Name:   "Twilio",
		Weight: 10,
		Endpoints: []string{
			"/2010-04-01/Accounts/messages",
			"/2010-04-01/Accounts/calls",
		},
		Methods:   []string{"POST", "GET"},
		ErrorRate: 0.02,
	},
	{
		Name:   "AWS S3",
		Weight: 10,
		Endpoints: []string{
			"/bucket/object",
			"/bucket/list",
		},
		Methods:   []string{"GET", "PUT", "DELETE"},
		ErrorRate: 0.01,
	},
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

type Simulator struct {
	config      Config
	client      *http.Client
	stats       *Stats
	rateLimiter *time.Ticker
}

type Stats struct {
	mu              sync.Mutex
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	startTime       time.Time
}

func main() {
	config := parseFlags()

	switch config.Mode {
	case "load":
		runLoadTest(config)
	case "patterns":
		runPatterns(config)
	case "scenarios":
		runScenarios(config)
	default:
		runLoadTest(config)
	}
}

func parseFlags() Config {
	ingestURL := flag.String("url", "http://localhost:8081/api/ingest", "Ingestion API URL")
	duration := flag.Duration("duration", 1*time.Minute, "Simulation duration")
	rps := flag.Int("rps", 50, "Requests per second")
	concurrency := flag.Int("concurrency", 10, "Concurrent workers")
	orgID := flag.String("org", "1", "Organization ID")
	mode := flag.String("mode", "load", "Mode: load, patterns, or scenarios")

	flag.Parse()

	return Config{
		IngestURL:   *ingestURL,
		Duration:    *duration,
		RPS:         *rps,
		Concurrency: *concurrency,
		OrgID:       *orgID,
		Mode:        *mode,
	}
}

// ==================== LOAD TEST MODE ====================

func runLoadTest(config Config) {
	log.Printf("🚀 Starting Load Test...")
	log.Printf("  Target: %s", config.IngestURL)
	log.Printf("  Duration: %s", config.Duration)
	log.Printf("  Target RPS: %d", config.RPS)
	log.Printf("  Concurrency: %d", config.Concurrency)
	log.Println()

	simulator := NewSimulator(config)
	simulator.Run()
}

func NewSimulator(config Config) *Simulator {
	return &Simulator{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		stats: &Stats{
			startTime: time.Now(),
		},
		rateLimiter: time.NewTicker(time.Second / time.Duration(config.RPS)),
	}
}

func (s *Simulator) Run() {
	ctx := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < s.config.Concurrency; i++ {
		wg.Add(1)
		go s.worker(&wg, ctx)
	}

	go s.reportStats(ctx)

	time.Sleep(s.config.Duration)
	close(ctx)
	wg.Wait()

	s.printFinalStats()
}

func (s *Simulator) worker(wg *sync.WaitGroup, ctx chan struct{}) {
	defer wg.Done()

	for {
		select {
		case <-ctx:
			return
		case <-s.rateLimiter.C:
			s.sendRequest()
		}
	}
}

func (s *Simulator) sendRequest() {
	provider := selectProvider()
	endpoint := provider.Endpoints[rand.Intn(len(provider.Endpoints))]
	method := provider.Methods[rand.Intn(len(provider.Methods))]

	baseLatency := map[string]int{
		"OpenAI":   800,
		"Stripe":   150,
		"SendGrid": 200,
		"Twilio":   300,
		"AWS S3":   100,
	}

	latency := baseLatency[provider.Name] + rand.Intn(500)

	statusCode := 200
	errorMsg := ""
	if rand.Float64() < provider.ErrorRate {
		statusCode = []int{400, 401, 403, 404, 429, 500, 502, 503}[rand.Intn(8)]
		errorMsg = fmt.Sprintf("%s error occurred", http.StatusText(statusCode))
	}

	reqSize := rand.Intn(5000) + 100
	respSize := rand.Intn(50000) + 500

	requestID := generateRequestID()
	if rand.Float64() < 0.1 {
		requestID = fmt.Sprintf("dup_%d", rand.Intn(100))
	}

	req := APIRequest{
		RequestID:         requestID,
		OrganizationID:    s.config.OrgID,
		Timestamp:         time.Now().UnixMilli(),
		Provider:          provider.Name,
		Endpoint:          endpoint,
		Method:            method,
		StatusCode:        statusCode,
		LatencyMS:         latency,
		RequestSizeBytes:  reqSize,
		ResponseSizeBytes: respSize,
		ErrorMessage:      errorMsg,
		Metadata: map[string]string{
			"user_id":    fmt.Sprintf("user_%d", rand.Intn(1000)),
			"session_id": fmt.Sprintf("sess_%d", rand.Intn(100)),
			"version":    "v1.0.0",
		},
	}

	if err := s.ingestRequest(req); err != nil {
		s.stats.mu.Lock()
		s.stats.failedRequests++
		s.stats.mu.Unlock()
		return
	}

	s.stats.mu.Lock()
	s.stats.totalRequests++
	s.stats.successRequests++
	s.stats.mu.Unlock()
}

func (s *Simulator) ingestRequest(req APIRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := s.client.Post(s.config.IngestURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (s *Simulator) reportStats(ctx chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx:
			return
		case <-ticker.C:
			s.printStats()
		}
	}
}

func (s *Simulator) printStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	rps := float64(s.stats.totalRequests) / elapsed.Seconds()
	successRate := 0.0
	if s.stats.totalRequests > 0 {
		successRate = float64(s.stats.successRequests) / float64(s.stats.totalRequests) * 100
	}

	log.Printf("[STATS] Total: %d | Success: %d | Failed: %d | RPS: %.2f | Success Rate: %.2f%%",
		s.stats.totalRequests, s.stats.successRequests, s.stats.failedRequests, rps, successRate)
}

func (s *Simulator) printFinalStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	avgRPS := 0.0
	if elapsed.Seconds() > 0 {
		avgRPS = float64(s.stats.totalRequests) / elapsed.Seconds()
	}
	successRate := 0.0
	if s.stats.totalRequests > 0 {
		successRate = float64(s.stats.successRequests) / float64(s.stats.totalRequests) * 100
	}

	fmt.Println("FINAL STATISTICS")

	fmt.Printf("Duration:        %s\n", elapsed.Round(time.Second))
	fmt.Printf("Total Requests:  %d\n", s.stats.totalRequests)
	fmt.Printf("Success:         %d\n", s.stats.successRequests)
	fmt.Printf("Failed:          %d\n", s.stats.failedRequests)
	fmt.Printf("Success Rate:    %.2f%%\n", successRate)
	fmt.Printf("Avg RPS:         %.2f\n", avgRPS)

}

// ==================== PATTERNS MODE ====================

func runPatterns(config Config) {
	log.Println("🎯 Generating Realistic Patterns...")
	log.Println()

	patterns := []struct {
		name string
		fn   func(string)
	}{
		{"Duplicate Requests", func(url string) { generateDuplicates(url, 20) }},
		{"Cacheable GET Requests", func(url string) { generateCacheableRequests(url, 50) }},
		{"Cost Spike", func(url string) { generateCostSpike(url, 100) }},
		{"Error Surge", func(url string) { generateErrorSurge(url, 30) }},
		{"Rate Limit Abuse", func(url string) { generateRateLimitAbuse(url, 50) }},
	}

	for _, pattern := range patterns {
		log.Printf("▶ Executing: %s", pattern.name)
		pattern.fn(config.IngestURL)
		log.Printf("✓ Completed: %s\n", pattern.name)
		time.Sleep(2 * time.Second)
	}

	fmt.Println("✅ All patterns generated!")
}

func generateDuplicates(url string, count int) {
	duplicateReq := createRequest("Stripe", "/v1/charges", "POST", 200, 150)

	for i := 0; i < count; i++ {
		sendRequestSync(url, duplicateReq)
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
		sendRequestSync(url, req)
		time.Sleep(100 * time.Millisecond)
	}
}

func generateCostSpike(url string, count int) {
	for i := 0; i < count; i++ {
		req := createRequest("OpenAI", "/v1/chat/completions", "POST", 200, 2000)
		req.RequestSizeBytes = 10000 + rand.Intn(40000)
		req.ResponseSizeBytes = 50000 + rand.Intn(100000)
		sendRequestSync(url, req)
		time.Sleep(200 * time.Millisecond)
	}
}

func generateErrorSurge(url string, count int) {
	errorCodes := []int{400, 401, 429, 500, 502, 503}

	for i := 0; i < count; i++ {
		statusCode := errorCodes[rand.Intn(len(errorCodes))]
		req := createRequest("SendGrid", "/v3/mail/send", "POST", statusCode, 300)
		req.ErrorMessage = fmt.Sprintf("Error %d occurred", statusCode)
		sendRequestSync(url, req)
		time.Sleep(100 * time.Millisecond)
	}
}

func generateRateLimitAbuse(url string, count int) {
	endpoint := "/v1/api/resource/123"

	for i := 0; i < count; i++ {
		req := createRequest("Twilio", endpoint, "GET", 429, 50)
		req.ErrorMessage = "Rate limit exceeded"
		sendRequestSync(url, req)
		time.Sleep(20 * time.Millisecond)
	}
}

// ==================== SCENARIOS MODE ====================

func runScenarios(config Config) {
	log.Println("📊 Running Interactive Scenarios...")
	fmt.Println()
	fmt.Println("Select a scenario:")
	fmt.Println("  1) Morning Spike (2 min, 100 RPS)")
	fmt.Println("  2) Steady Traffic (5 min, 50 RPS)")
	fmt.Println("  3) Spike Event (1 min, 200 RPS)")
	fmt.Println("  4) Night Traffic (1 min, 10 RPS)")
	fmt.Println("  5) Full Day Simulation")
	fmt.Println()

	var choice int
	fmt.Print("Enter choice [1-5]: ")
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		config.Duration = 2 * time.Minute
		config.RPS = 100
		config.Concurrency = 20
	case 2:
		config.Duration = 5 * time.Minute
		config.RPS = 50
		config.Concurrency = 10
	case 3:
		config.Duration = 1 * time.Minute
		config.RPS = 200
		config.Concurrency = 30
	case 4:
		config.Duration = 1 * time.Minute
		config.RPS = 10
		config.Concurrency = 2
	case 5:
		runFullDaySimulation(config)
		return
	default:
		log.Println("Invalid choice")
		return
	}

	runLoadTest(config)
}

func runFullDaySimulation(config Config) {
	scenarios := []struct {
		name        string
		duration    time.Duration
		rps         int
		concurrency int
	}{
		{"Morning Spike", 2 * time.Minute, 100, 20},
		{"Steady Traffic", 5 * time.Minute, 50, 10},
		{"Spike Event", 1 * time.Minute, 200, 30},
		{"Evening Traffic", 3 * time.Minute, 75, 15},
		{"Night Traffic", 1 * time.Minute, 10, 2},
	}

	for _, scenario := range scenarios {
		log.Printf("▶ Running: %s", scenario.name)
		config.Duration = scenario.duration
		config.RPS = scenario.rps
		config.Concurrency = scenario.concurrency
		runLoadTest(config)
		time.Sleep(5 * time.Second)
	}

	log.Println("✅ Full day simulation completed!")
}

// ==================== HELPER FUNCTIONS ====================

func selectProvider() APIProvider {
	totalWeight := 0
	for _, p := range providers {
		totalWeight += p.Weight
	}

	r := rand.Intn(totalWeight)
	sum := 0
	for _, p := range providers {
		sum += p.Weight
		if r < sum {
			return p
		}
	}

	return providers[0]
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().Unix(), rand.Intn(1000000))
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

func sendRequestSync(url string, req APIRequest) {
	client := &http.Client{Timeout: 10 * time.Second}
	data, _ := json.Marshal(req)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
