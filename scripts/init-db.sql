-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Organizations table
CREATE TABLE organizations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- API providers pricing table
CREATE TABLE api_providers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    base_cost_per_request DECIMAL(10, 6) DEFAULT 0,
    rate_limit_per_minute INTEGER DEFAULT 60,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert sample providers
INSERT INTO api_providers (name, base_cost_per_request, rate_limit_per_minute) VALUES
('OpenAI', 0.002, 3500),
('Stripe', 0.0001, 100),
('SendGrid', 0.0005, 600),
('Twilio', 0.0075, 1000),
('AWS S3', 0.0004, 3500);

-- API requests table (hypertable for time-series data)
CREATE TABLE api_requests (
    time TIMESTAMPTZ NOT NULL,
    organization_id INTEGER NOT NULL,
    request_id VARCHAR(255) NOT NULL,
    provider VARCHAR(100) NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER,
    latency_ms INTEGER,
    request_size_bytes INTEGER,
    response_size_bytes INTEGER,
    cost DECIMAL(10, 6),
    error_message TEXT,
    metadata JSONB,
    PRIMARY KEY (time, request_id)
);

-- Convert to hypertable
SELECT create_hypertable('api_requests', 'time');

-- Create indexes
CREATE INDEX idx_api_requests_org_time ON api_requests (organization_id, time DESC);
CREATE INDEX idx_api_requests_provider ON api_requests (provider, time DESC);
CREATE INDEX idx_api_requests_endpoint ON api_requests (endpoint, time DESC);
CREATE INDEX idx_api_requests_status ON api_requests (status_code, time DESC);

-- Duplicate requests detection table
CREATE TABLE duplicate_requests (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    count INTEGER DEFAULT 1,
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL,
    potential_savings DECIMAL(10, 4),
    UNIQUE(organization_id, request_hash)
);

-- Cache recommendations table
CREATE TABLE cache_recommendations (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    cache_hit_ratio DECIMAL(5, 2),
    potential_savings DECIMAL(10, 4),
    recommendation TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Cost aggregations (continuous aggregate)
CREATE MATERIALIZED VIEW api_costs_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    organization_id,
    provider,
    COUNT(*) as request_count,
    SUM(cost) as total_cost,
    AVG(latency_ms) as avg_latency,
    COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_count
FROM api_requests
GROUP BY bucket, organization_id, provider
WITH NO DATA;

-- Refresh policy for continuous aggregate
SELECT add_continuous_aggregate_policy('api_costs_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

-- Anomaly detection results
CREATE TABLE anomalies (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL,
    anomaly_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT,
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    metadata JSONB
);

-- Create sample organization
INSERT INTO organizations (name, api_key) VALUES
('Demo Organization', 'demo_api_key_12345');
