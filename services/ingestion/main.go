package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

type Server struct {
	db                *sql.DB
	redis             *redis.Client
	requestsProcessed atomic.Int64
	startTime         time.Time
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

type IngestResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func main() {
	log.Println("Starting Ingestion Service...")

	// Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/api_observatory?sslmode=disable"
	}

	log.Printf("Connecting to database...")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		db = nil
	} else {
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		if err := db.Ping(); err != nil {
			log.Printf("Warning: Failed to ping database: %v", err)
			db = nil
		} else {
			log.Println("✓ Connected to TimescaleDB")
		}
	}

	// Connect to Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	log.Printf("Connecting to Redis...")
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		rdb = nil
	} else {
		log.Println("✓ Connected to Redis")
	}

	server := &Server{
		db:        db,
		redis:     rdb,
		startTime: time.Now(),
	}

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/api/ingest", server.handleIngest)
	mux.HandleFunc("/api/health", server.handleHealth)
	mux.HandleFunc("/", server.handleRoot)

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8081"
	}

	log.Printf("✓ HTTP server listening on port %s", httpPort)
	log.Println("✓ Registered routes:")
	log.Println("    POST /api/ingest")
	log.Println("    GET  /api/health")
	log.Println("Ready to accept requests!")

	if err := http.ListenAndServe(":"+httpPort, handler); err != nil {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "API Observatory Ingestion Service",
		"status":  "running",
		"version": "1.0.0",
		"routes": []string{
			"POST /api/ingest",
			"GET  /api/health",
		},
	})
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received ingest request from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Processing request: %s for provider %s", req.RequestID, req.Provider)

	// Validate required fields
	if req.Provider == "" || req.Endpoint == "" {
		log.Printf("Missing required fields: provider=%s, endpoint=%s", req.Provider, req.Endpoint)
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Calculate cost
	cost := s.calculateCost(req.Provider, req.RequestSizeBytes, req.ResponseSizeBytes)

	// Store in database if available
	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		metadataJSON, _ := json.Marshal(req.Metadata)

		query := `
			INSERT INTO api_requests (
				time, organization_id, request_id, provider, endpoint, method,
				status_code, latency_ms, request_size_bytes, response_size_bytes,
				cost, error_message, metadata
			) VALUES (
				to_timestamp($1), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			)
		`

		_, err := s.db.ExecContext(ctx, query,
			float64(req.Timestamp)/1000.0,
			req.OrganizationID,
			req.RequestID,
			req.Provider,
			req.Endpoint,
			req.Method,
			req.StatusCode,
			req.LatencyMS,
			req.RequestSizeBytes,
			req.ResponseSizeBytes,
			cost,
			req.ErrorMessage,
			metadataJSON,
		)

		if err != nil {
			log.Printf("Failed to insert request: %v", err)
		} else {
			log.Printf("Stored request in database: %s", req.RequestID)
		}
	}

	// Increment counter
	s.requestsProcessed.Add(1)

	// Publish to Redis if available
	if s.redis != nil {
		event := map[string]interface{}{
			"type":      "new_request",
			"provider":  req.Provider,
			"cost":      cost,
			"timestamp": req.Timestamp,
		}
		eventJSON, _ := json.Marshal(event)
		s.redis.Publish(context.Background(), "api_events", eventJSON)
	}

	// Send success response
	resp := IngestResponse{
		Success:   true,
		Message:   "Request ingested successfully",
		RequestID: req.RequestID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
	}

	log.Printf("Successfully processed request: %s", req.RequestID)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime)

	dbStatus := "disconnected"
	if s.db != nil {
		if err := s.db.Ping(); err == nil {
			dbStatus = "connected"
		}
	}

	redisStatus := "disconnected"
	if s.redis != nil {
		if err := s.redis.Ping(context.Background()).Err(); err == nil {
			redisStatus = "connected"
		}
	}

	health := map[string]interface{}{
		"status":             "healthy",
		"requests_processed": s.requestsProcessed.Load(),
		"uptime_seconds":     uptime.Seconds(),
		"database":           dbStatus,
		"redis":              redisStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (s *Server) calculateCost(provider string, reqSize, respSize int) float64 {
	baseCost := 0.001

	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		query := "SELECT base_cost_per_request FROM api_providers WHERE name = $1"
		err := s.db.QueryRowContext(ctx, query, provider).Scan(&baseCost)
		if err != nil {
			baseCost = 0.001
		}
	}

	totalSize := float64(reqSize+respSize) / 1024.0
	sizeCost := totalSize * 0.00001

	return baseCost + sizeCost
}
