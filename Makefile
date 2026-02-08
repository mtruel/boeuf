# Makefile for Boeuf Project Standardized Workflow

COMPOSE_FILE_PROD := -f docker-compose.yml
COMPOSE_FILE_DEV  := -f docker-compose.yml -f docker-compose.dev.yml

.PHONY: help watch dev-restart prod build down logs test test-backend test-frontend test-quick clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

watch: ## Start development environment with Hot Reload (Watch Mode - Blocking)
	@echo "Starting Dev Environment (Blocking with Watch)..."
	docker compose $(COMPOSE_FILE_DEV) up -d --build --remove-orphans
	@echo "Starting File Watcher..."
	docker compose $(COMPOSE_FILE_DEV) watch

dev-restart: ## Rebuild and start development environment (Non-blocking)
	@echo "Rebuilding and Starting Dev Environment..."
	docker compose $(COMPOSE_FILE_DEV) up -d --build --remove-orphans

prod: ## Start production-like environment
	@echo "Starting Production Environment..."
	docker compose $(COMPOSE_FILE_PROD) up -d --build --remove-orphans

build: ## Build all containers
	docker compose $(COMPOSE_FILE_PROD) build

down: ## Stop all containers
	docker compose down --remove-orphans

logs: ## Show logs (snapshot, non-blocking)
	docker compose logs

watch-logs: ## Follow logs (blocking)
	docker compose logs -f

test: test-backend test-frontend ## Run all tests

test-quick: ## Run all tests without rebuilding (faster, use after first build)
	@echo "Running All Tests (Quick Mode - No Rebuild)..."
	@$(MAKE) test-backend
	@$(MAKE) test-frontend

test-backend: ## Run Backend tests (Go)
	@echo "Running Backend Tests..."
	docker compose $(COMPOSE_FILE_DEV) run --rm --no-deps backend go test ./... -v

test-frontend: ## Run Frontend tests (Vitest)
	@echo "Running Frontend Tests..."
	# Run tests using the built dev container with all deps pre-installed
	docker compose $(COMPOSE_FILE_DEV) run --rm --no-deps frontend sh -c "pnpm run test:unit --run"

clean: ## Remove artifacts and volumes
	docker compose down -v
	rm -rf frontend/node_modules
	rm -rf frontend/dist
