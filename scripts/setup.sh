#!/bin/bash

set -e

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}   API Observatory - Automated Setup                    ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

# Check dependencies
echo -e "${YELLOW}Checking dependencies...${NC}"

command -v docker >/dev/null 2>&1 || { echo -e "${RED}Docker is required but not installed. Aborting.${NC}" >&2; exit 1; }
command -v docker compose >/dev/null 2>&1 || { echo -e "${RED}Docker Compose is required but not installed. Aborting.${NC}" >&2; exit 1; }

echo -e "${GREEN}✓ Docker found${NC}"
echo -e "${GREEN}✓ Docker Compose found${NC}"
echo ""

# Create necessary directories
echo -e "${YELLOW}Creating directory structure...${NC}"
mkdir -p services/{ingestion,analytics,cost-tracker,api-gateway}
mkdir -p dashboard/{app,components,lib}
mkdir -p sdk/{go,node}
mkdir -p shared/proto
mkdir -p scripts
echo -e "${GREEN}✓ Directories created${NC}"
echo ""

# Initialize Go modules
echo -e "${YELLOW}Initializing Go modules...${NC}"
bash scripts/init-go-modules.sh
echo ""

# Check for .env file
if [ ! -f .env ]; then
    echo -e "${YELLOW}Creating .env file...${NC}"
    cat > .env << 'EOF'
# Database
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/api_observatory?sslmode=disable
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=api_observatory

# Redis
REDIS_URL=localhost:6379

# Services
INGESTION_SERVICE_URL=localhost:50051
ANALYTICS_SERVICE_URL=localhost:50052
COST_TRACKER_SERVICE_URL=localhost:50053

# API Gateway
API_GATEWAY_URL=http://localhost:8080

# Dashboard
NEXT_PUBLIC_API_GATEWAY_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws

# Timezone
TZ=Asia/Kolkata
EOF
    echo -e "${GREEN}✓ .env file created${NC}"
else
    echo -e "${GREEN}✓ .env file already exists${NC}"
fi
echo ""

# Update dashboard package.json if it doesn't exist
if [ ! -f dashboard/package.json ]; then
    echo -e "${YELLOW}Creating dashboard package.json...${NC}"
    cat > dashboard/package.json << 'EOF'
{
  "name": "api-observatory-dashboard",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "next": "^14.1.0",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "recharts": "^2.10.0"
  },
  "devDependencies": {
    "@types/node": "^20.10.0",
    "@types/react": "^18.2.0",
    "typescript": "^5.3.0",
    "tailwindcss": "^3.4.0",
    "autoprefixer": "^10.4.16",
    "postcss": "^8.4.32"
  }
}
EOF
    echo -e "${GREEN}✓ Dashboard package.json created${NC}"
fi
echo ""

# Build Docker images
echo -e "${YELLOW}Building Docker images (this may take a few minutes)...${NC}"
docker compose build --parallel
echo -e "${GREEN}✓ Docker images built successfully${NC}"
echo ""

# Start services
echo -e "${YELLOW}Starting all services...${NC}"
docker compose up -d
echo -e "${GREEN}✓ All services started${NC}"
echo ""

# Wait for services to be healthy
echo -e "${YELLOW}Waiting for services to be ready...${NC}"
sleep 15

# Check health
echo -e "${YELLOW}Checking service health...${NC}"

check_service() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ $name is healthy${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    echo -e "${RED}✗ $name failed to start${NC}"
    return 1
}

check_service "http://localhost:8081/api/health" "Ingestion Service"
check_service "http://localhost:8080/health" "API Gateway"
check_service "http://localhost:3000" "Dashboard"

echo ""

# Success message
echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}   API Observatory is ready!                           ${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "Access points:"
echo -e "  ${BLUE}Dashboard:${NC}     http://localhost:3000"
echo -e "  ${BLUE}API Gateway:${NC}   http://localhost:8080"
echo -e "  ${BLUE}Ingestion API:${NC} http://localhost:8081"
echo ""
echo -e "Useful commands:"
echo -e "  ${YELLOW}make logs${NC}      - View all logs"
echo -e "  ${YELLOW}make health${NC}    - Check service health"
echo -e "  ${YELLOW}make down${NC}      - Stop all services"
echo ""
echo -e "View logs: ${YELLOW}docker compose logs -f${NC}"
echo ""
