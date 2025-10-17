package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type Gateway struct {
	redis *redis.Client
}

func main() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		log.Println("Connected to Redis")
	}

	gateway := &Gateway{
		redis: rdb,
	}

	// Middleware
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/costs", gateway.handleGetCosts)
	mux.HandleFunc("/api/analytics/duplicates", gateway.handleGetDuplicates)
	mux.HandleFunc("/api/analytics/cache-recommendations", gateway.handleGetCacheRecommendations)
	mux.HandleFunc("/api/analytics/anomalies", gateway.handleGetAnomalies)
	mux.HandleFunc("/api/dashboard/summary", gateway.handleGetDashboardSummary)

	// WebSocket for real-time updates
	mux.HandleFunc("/ws", gateway.handleWebSocket)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Root handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "API Observatory Gateway",
			"status":  "running",
			"version": "1.0.0",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	log.Printf("API Gateway listening on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// CORS Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (g *Gateway) handleGetCosts(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Try to get from Redis
	data, err := g.redis.Get(ctx, "costs:24h:by_provider").Result()
	if err == redis.Nil || err != nil {
		// Return default data if Redis is empty
		defaultData := map[string]interface{}{
			"breakdown":  []map[string]interface{}{},
			"total_cost": 0.0,
			"updated_at": time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(defaultData)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (g *Gateway) handleGetDuplicates(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	data, err := g.redis.Get(ctx, "analytics:duplicates").Result()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (g *Gateway) handleGetCacheRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	data, err := g.redis.Get(ctx, "analytics:cache_recommendations").Result()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (g *Gateway) handleGetAnomalies(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	data, err := g.redis.Get(ctx, "analytics:anomalies").Result()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (g *Gateway) handleGetDashboardSummary(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get all cached data
	costs := g.getRedisData(ctx, "costs:24h:by_provider")
	duplicates := g.getRedisData(ctx, "analytics:duplicates")
	cacheRecs := g.getRedisData(ctx, "analytics:cache_recommendations")
	anomalies := g.getRedisData(ctx, "analytics:anomalies")

	summary := map[string]interface{}{
		"costs":                 costs,
		"duplicates":            duplicates,
		"cache_recommendations": cacheRecs,
		"anomalies":             anomalies,
		"updated_at":            time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (g *Gateway) getRedisData(ctx context.Context, key string) interface{} {
	data, err := g.redis.Get(ctx, key).Result()
	if err != nil {
		// Return empty default based on key
		if key == "costs:24h:by_provider" {
			return map[string]interface{}{
				"breakdown":  []interface{}{},
				"total_cost": 0.0,
			}
		}
		return []interface{}{}
	}

	var result interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return []interface{}{}
	}
	return result
}

func (g *Gateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	ctx := context.Background()
	pubsub := g.redis.Subscribe(ctx, "api_events")
	defer pubsub.Close()

	log.Println("WebSocket client connected")

	// Send initial data
	summary := g.getInitialData(ctx)
	if err := conn.WriteJSON(summary); err != nil {
		log.Printf("Failed to send initial data: %v", err)
		return
	}

	// Create channels
	done := make(chan struct{})

	// Handle incoming messages (keepalive)
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Stream updates from Redis
	ch := pubsub.Channel()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			log.Println("WebSocket client disconnected")
			return
		case msg := <-ch:
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}

			if err := conn.WriteJSON(event); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		case <-ticker.C:
			// Send ping to keep connection alive
			if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
				return
			}
		}
	}
}

func (g *Gateway) getInitialData(ctx context.Context) map[string]interface{} {
	costs := g.getRedisData(ctx, "costs:24h:by_provider")
	return map[string]interface{}{
		"type":      "initial_data",
		"data":      costs,
		"timestamp": time.Now().Unix(),
	}
}
