package observatory

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	APIKey    string
	IngestURL string
	OrgID     string
}

type Middleware struct {
	config Config
	client *http.Client
}

func NewMiddleware(config Config) *Middleware {
	return &Middleware{
		config: config,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := uuid.New().String()

		// Capture response
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		// Send to Observatory
		go m.sendRequest(APIRequest{
			RequestID:      requestID,
			OrganizationID: m.config.OrgID,
			Timestamp:      time.Now().UnixMilli(),
			Provider:       "internal",
			Endpoint:       r.URL.Path,
			Method:         r.Method,
			StatusCode:     recorder.statusCode,
			LatencyMS:      int(time.Since(start).Milliseconds()),
		})
	})
}

type APIRequest struct {
	RequestID      string `json:"request_id"`
	OrganizationID string `json:"organization_id"`
	Timestamp      int64  `json:"timestamp"`
	Provider       string `json:"provider"`
	Endpoint       string `json:"endpoint"`
	Method         string `json:"method"`
	StatusCode     int    `json:"status_code"`
	LatencyMS      int    `json:"latency_ms"`
}

func (m *Middleware) sendRequest(req APIRequest) {
	data, _ := json.Marshal(req)
	http.Post(m.config.IngestURL+"/api/ingest", "application/json", bytes.NewBuffer(data))
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}
