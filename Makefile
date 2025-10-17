.PHONY: help build up down restart logs clean test migrate seed

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)API Observatory - Available Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

build: ## Build all Docker images
	@echo "$(BLUE)Building all services...$(NC)"
	docker compose build --parallel

build-nocache: ## Build all Docker images without cache
	@echo "$(BLUE)Building all services (no cache)...$(NC)"
	docker compose build --no-cache --parallel

up: ## Start all services
	@echo "$(BLUE)Starting all services...$(NC)"
	docker compose up -d
	@echo "$(GREEN)✓ All services started$(NC)"
	@echo "Dashboard: http://localhost:3000"
	@echo "API Gateway: http://localhost:8080"
	@echo "Ingestion API: http://localhost:8081"

up-build: ## Build and start all services
	@echo "$(BLUE)Building and starting all services...$(NC)"
	docker compose up -d --build

down: ## Stop all services
	@echo "$(BLUE)Stopping all services...$(NC)"
	docker compose down
	@echo "$(GREEN)✓ All services stopped$(NC)"

down-volumes: ## Stop all services and remove volumes
	@echo "$(RED)Stopping all services and removing volumes...$(NC)"
	docker compose down -v
	@echo "$(GREEN)✓ All services stopped and volumes removed$(NC)"

restart: ## Restart all services
	@echo "$(BLUE)Restarting all services...$(NC)"
	docker compose restart
	@echo "$(GREEN)✓ All services restarted$(NC)"

logs: ## Tail logs from all services
	docker compose logs -f

logs-ingestion: ## Tail logs from ingestion service
	docker compose logs -f ingestion-service

logs-analytics: ## Tail logs from analytics service
	docker compose logs -f analytics-service

logs-cost: ## Tail logs from cost-tracker service
	docker compose logs -f cost-tracker-service

logs-gateway: ## Tail logs from API gateway
	docker compose logs -f api-gateway

logs-dashboard: ## Tail logs from dashboard
	docker compose logs -f dashboard

ps: ## Show running services
	docker compose ps

stats: ## Show container resource usage
	docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}"

clean: ## Remove all containers, networks, and volumes
	@echo "$(RED)Cleaning up all resources...$(NC)"
	docker compose down -v --remove-orphans
	docker system prune -f
	@echo "$(GREEN)✓ Cleanup complete$(NC)"

shell-db: ## Open PostgreSQL shell
	docker compose exec timescaledb psql -U postgres -d api_observatory

shell-redis: ## Open Redis CLI
	docker compose exec redis redis-cli

seed: ## Generate test data
	@echo "$(BLUE)Generating test data...$(NC)"
	@bash scripts/generate-test-data.sh
	@echo "$(GREEN)✓ Test data generated$(NC)"

health: ## Check health of all services
	@echo "$(BLUE)Checking service health...$(NC)"
	@echo ""
	@echo "TimescaleDB:"
	@curl -s http://localhost:5432 > /dev/null 2>&1 && echo "  $(GREEN)✓ Healthy$(NC)" || echo "  $(RED)✗ Unhealthy$(NC)"
	@echo "Redis:"
	@docker compose exec -T redis redis-cli ping > /dev/null 2>&1 && echo "  $(GREEN)✓ Healthy$(NC)" || echo "  $(RED)✗ Unhealthy$(NC)"
	@echo "Ingestion Service:"
	@curl -s http://localhost:8081/api/health > /dev/null 2>&1 && echo "  $(GREEN)✓ Healthy$(NC)" || echo "  $(RED)✗ Unhealthy$(NC)"
	@echo "API Gateway:"
	@curl -s http://localhost:8080/health > /dev/null 2>&1 && echo "  $(GREEN)✓ Healthy$(NC)" || echo "  $(RED)✗ Unhealthy$(NC)"
	@echo "Dashboard:"
	@curl -s http://localhost:3000 > /dev/null 2>&1 && echo "  $(GREEN)✓ Healthy$(NC)" || echo "  $(RED)✗ Unhealthy$(NC)"

dev: ## Start services in development mode with hot reload
	@echo "$(BLUE)Starting in development mode...$(NC)"
	docker compose watch

backup-db: ## Backup database
	@echo "$(BLUE)Backing up database...$(NC)"
	docker compose exec -T timescaledb pg_dump -U postgres api_observatory > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)✓ Database backed up$(NC)"

restore-db: ## Restore database from backup (usage: make restore-db FILE=backup.sql)
	@echo "$(BLUE)Restoring database from $(FILE)...$(NC)"
	docker compose exec -T timescaledb psql -U postgres api_observatory < $(FILE)
	@echo "$(GREEN)✓ Database restored$(NC)"

install-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "$(GREEN)✓ Tools installed$(NC)"

proto: ## Generate protobuf files
	@echo "$(BLUE)Generating protobuf files...$(NC)"
	protoc --go_out=. --go-grpc_out=. shared/proto/*.proto
	@echo "$(GREEN)✓ Protobuf files generated$(NC)"

test: ## Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	@cd services/ingestion && go test -v ./...
	@cd services/analytics && go test -v ./...
	@cd services/cost-tracker && go test -v ./...
	@cd services/api-gateway && go test -v ./...
	@echo "$(GREEN)✓ All tests passed$(NC)"

benchmark: ## Run performance benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@cd services/ingestion && go test -bench=. -benchmem ./...

lint: ## Run linters
	@echo "$(BLUE)Running linters...$(NC)"
	golangci-lint run ./services/...
	@cd dashboard && npm run lint

init: build up seed ## Initialize the project (build, start, and seed data)
	@echo "$(GREEN)✓ API Observatory is ready!$(NC)"
	@echo ""
	@echo "Access points:"
	@echo "  Dashboard:    http://localhost:3000"
	@echo "  API Gateway:  http://localhost:8080"
	@echo "  Ingestion:    http://localhost:8081"
	@echo ""
	@echo "Useful commands:"
	@echo "  make logs     - View all logs"
	@echo "  make health   - Check service health"
	@echo "  make down     - Stop all services"


simulate-quick: ## Quick 1-minute test
	@go run scripts/simulator/main.go -duration=1m -rps=50

simulate-patterns: ## Generate realistic problem patterns
	@go run scripts/simulator/main.go -mode=patterns

simulate-load: ## Heavy load test (5 min, 200 RPS)
	@go run scripts/simulator/main.go -duration=5m -rps=200 -concurrency=30

simulate-scenarios: ## Interactive scenario menu
	@go run scripts/simulator/main.go -mode=scenarios

simulate-continuous: ## Run continuous load (10 minutes)
	@go run scripts/simulator/main.go -duration=10m -rps=50 -concurrency=10
