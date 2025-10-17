package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

type AnalyticsServer struct {
	db    *sql.DB
	redis *redis.Client
}

type DuplicateGroup struct {
	Endpoint  string    `json:"endpoint"`
	Count     int       `json:"count"`
	Cost      float64   `json:"cost"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

type CacheRecommendation struct {
	Endpoint         string  `json:"endpoint"`
	CacheHitRatio    float64 `json:"cache_hit_ratio"`
	PotentialSavings float64 `json:"potential_savings"`
	Recommendation   string  `json:"recommendation"`
}

type Anomaly struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	DetectedAt  time.Time `json:"detected_at"`
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	redisURL := os.Getenv("REDIS_URL")
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	defer rdb.Close()

	server := &AnalyticsServer{
		db:    db,
		redis: rdb,
	}

	// Start background analysis jobs
	go server.continuousAnalysis()

	log.Println("Analytics-service running and analyzing usage patterns...")
	select {} // block forever
}

func (s *AnalyticsServer) continuousAnalysis() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		log.Println("Running continuous analysis...")
		s.detectDuplicates()
		s.analyzeCacheOpportunities()
		s.detectAnomalies()
		<-ticker.C
	}
}

func (s *AnalyticsServer) detectDuplicates() {
	ctx := context.Background()

	// Find duplicate requests within 1-hour window
	query := `
        WITH request_hashes AS (
            SELECT
                organization_id,
                endpoint,
                method,
                MD5(endpoint || method || COALESCE(metadata::text, '')) as request_hash,
                time,
                cost
            FROM api_requests
            WHERE time > NOW() - INTERVAL '1 hour'
        ),
        duplicates AS (
            SELECT
                organization_id,
                request_hash,
                endpoint,
                COUNT(*) as duplicate_count,
                SUM(cost) as total_cost,
                MIN(time) as first_seen,
                MAX(time) as last_seen
            FROM request_hashes
            GROUP BY organization_id, request_hash, endpoint
            HAVING COUNT(*) > 1
        )
        SELECT
            organization_id,
            request_hash,
            endpoint,
            duplicate_count,
            total_cost,
            first_seen,
            last_seen
        FROM duplicates
        ORDER BY total_cost DESC
        LIMIT 100
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Failed to detect duplicates: %v", err)
		return
	}
	defer rows.Close()

	duplicates := []DuplicateGroup{}
	for rows.Next() {
		var orgID int
		var hash, endpoint string
		var count int
		var cost float64
		var firstSeen, lastSeen time.Time

		if err := rows.Scan(&orgID, &hash, &endpoint, &count, &cost, &firstSeen, &lastSeen); err != nil {
			continue
		}

		duplicates = append(duplicates, DuplicateGroup{
			Endpoint:  endpoint,
			Count:     count,
			Cost:      cost,
			FirstSeen: firstSeen,
			LastSeen:  lastSeen,
		})
	}

	// Cache results in Redis
	if len(duplicates) > 0 {
		data, _ := json.Marshal(duplicates)
		s.redis.Set(ctx, "analytics:duplicates", data, 10*time.Minute)
		log.Printf("Detected %d duplicate patterns", len(duplicates))
	}
}

func (s *AnalyticsServer) analyzeCacheOpportunities() {
	ctx := context.Background()

	// Identify GET requests with high repeat rates
	query := `
        WITH endpoint_stats AS (
            SELECT
                endpoint,
                COUNT(*) as total_requests,
                COUNT(DISTINCT MD5(endpoint || COALESCE(metadata::text, ''))) as unique_requests,
                SUM(cost) as total_cost,
                AVG(latency_ms) as avg_latency
            FROM api_requests
            WHERE
                method = 'GET'
                AND time > NOW() - INTERVAL '24 hours'
                AND status_code < 400
            GROUP BY endpoint
            HAVING COUNT(*) > 10
        )
        SELECT
            endpoint,
            total_requests,
            unique_requests,
            total_cost,
            avg_latency,
            ROUND(100.0 * (total_requests - unique_requests) / total_requests, 2) as cache_hit_ratio
        FROM endpoint_stats
        WHERE unique_requests < total_requests
        ORDER BY cache_hit_ratio DESC
        LIMIT 50
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Failed to analyze cache opportunities: %v", err)
		return
	}
	defer rows.Close()

	recommendations := []CacheRecommendation{}
	for rows.Next() {
		var endpoint string
		var totalReqs, uniqueReqs int
		var totalCost, avgLatency, cacheRatio float64

		if err := rows.Scan(&endpoint, &totalReqs, &uniqueReqs, &totalCost, &avgLatency, &cacheRatio); err != nil {
			continue
		}

		potentialSavings := totalCost * (cacheRatio / 100.0) * 0.8
		recommendation := fmt.Sprintf("Cache this endpoint with TTL of %d seconds. Could save %.2f%% of requests.",
			int(avgLatency/1000)*10, cacheRatio)

		recommendations = append(recommendations, CacheRecommendation{
			Endpoint:         endpoint,
			CacheHitRatio:    cacheRatio,
			PotentialSavings: potentialSavings,
			Recommendation:   recommendation,
		})
	}

	if len(recommendations) > 0 {
		data, _ := json.Marshal(recommendations)
		s.redis.Set(ctx, "analytics:cache_recommendations", data, 10*time.Minute)
		log.Printf("Generated %d cache recommendations", len(recommendations))
	}
}

func (s *AnalyticsServer) detectAnomalies() {
	ctx := context.Background()

	// Detect cost spikes (spending >3x the average)
	query := `
        WITH hourly_costs AS (
            SELECT
                time_bucket('1 hour', time) as hour,
                organization_id,
                SUM(cost) as hourly_cost,
                COUNT(*) as request_count
            FROM api_requests
            WHERE time > NOW() - INTERVAL '24 hours'
            GROUP BY hour, organization_id
        ),
        avg_costs AS (
            SELECT
                organization_id,
                AVG(hourly_cost) as avg_cost,
                STDDEV(hourly_cost) as stddev_cost
            FROM hourly_costs
            GROUP BY organization_id
        )
        SELECT
            hc.organization_id,
            hc.hour,
            hc.hourly_cost,
            ac.avg_cost,
            hc.request_count
        FROM hourly_costs hc
        JOIN avg_costs ac ON hc.organization_id = ac.organization_id
        WHERE hc.hourly_cost > (ac.avg_cost + 3 * ac.stddev_cost)
        ORDER BY hc.hour DESC
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Failed to detect anomalies: %v", err)
		return
	}
	defer rows.Close()

	anomalies := []Anomaly{}
	for rows.Next() {
		var orgID int
		var hour time.Time
		var hourlyCost, avgCost float64
		var requestCount int

		if err := rows.Scan(&orgID, &hour, &hourlyCost, &avgCost, &requestCount); err != nil {
			continue
		}

		spike := ((hourlyCost - avgCost) / avgCost) * 100
		description := fmt.Sprintf("Cost spike: $%.2f (%.0f%% above average). %d requests/hour.",
			hourlyCost, spike, requestCount)

		anomaly := Anomaly{
			Type:        "cost_spike",
			Severity:    "high",
			Description: description,
			DetectedAt:  hour,
		}

		anomalies = append(anomalies, anomaly)
	}

	if len(anomalies) > 0 {
		data, _ := json.Marshal(anomalies)
		s.redis.Set(ctx, "analytics:anomalies", data, 10*time.Minute)
		log.Printf("Detected %d anomalies", len(anomalies))
	}
}
