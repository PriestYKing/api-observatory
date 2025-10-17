#!/bin/bash

set -e

echo "Initializing Go modules for all services..."

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Create go.mod files for each service
services=("ingestion" "analytics" "cost-tracker" "api-gateway")

for service in "${services[@]}"; do
    echo -e "${BLUE}Creating go.mod for $service service...${NC}"

    mkdir -p services/$service

    cat > services/$service/go.mod << EOF
module github.com/yourusername/api-observatory/$service

go 1.22

require (
	github.com/go-redis/redis/v8 v8.11.5
	github.com/lib/pq v1.10.9
	google.golang.org/grpc v1.60.1
	google.golang.org/protobuf v1.32.0
	github.com/gorilla/websocket v1.5.1
	github.com/google/uuid v1.5.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240125205218-1f4bbc51befe // indirect
)
EOF

    # Create empty go.sum
    touch services/$service/go.sum

    echo -e "${GREEN}✓ Created go.mod for $service${NC}"
done

echo -e "${GREEN}All Go modules initialized!${NC}"
