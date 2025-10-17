package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

type CostTrackerServer struct {
	db    *sql.DB
	redis *redis.Client
}

type CostBreakdown struct {
	Label        string  `json:"label"`
	Cost         float64 `json:"cost"`
	RequestCount int64   `json:"request_count"`
	AvgLatency   float64 `json:"avg_latency"`
	ErrorCount   int     `json:"error_count"`
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

	server := &CostTrackerServer{
		db:    db,
		redis: rdb,
	}

	// Start background cost aggregation
	go server.aggregateCosts()

	log.Println("Cost-tracker running and aggregating costs...")
	select {} // block forever
}

func (s *CostTrackerServer) aggregateCosts() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		s.calculateRealTimeCosts()
		<-ticker.C
	}
}

func (s *CostTrackerServer) calculateRealTimeCosts() {
	ctx := context.Background()

	// Get costs by provider for last 24 hours
	query := `
        SELECT
            provider,
            COUNT(*) as request_count,
            SUM(cost) as total_cost,
            AVG(latency_ms) as avg_latency,
            COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_count
        FROM api_requests
        WHERE time > NOW() - INTERVAL '24 hours'
        GROUP BY provider
        ORDER BY total_cost DESC
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Failed to calculate costs: %v", err)
		return
	}
	defer rows.Close()

	breakdown := []CostBreakdown{}
	totalCost := 0.0

	for rows.Next() {
		var item CostBreakdown
		if err := rows.Scan(&item.Label, &item.RequestCount, &item.Cost, &item.AvgLatency, &item.ErrorCount); err != nil {
			continue
		}
		breakdown = append(breakdown, item)
		totalCost += item.Cost
	}

	// Cache in Redis for dashboard
	data := map[string]interface{}{
		"breakdown":  breakdown,
		"total_cost": totalCost,
		"updated_at": time.Now(),
	}
	jsonData, _ := json.Marshal(data)
	s.redis.Set(ctx, "costs:24h:by_provider", jsonData, 5*time.Minute)

	log.Printf("Cost aggregation complete: $%.4f across %d providers", totalCost, len(breakdown))
}
