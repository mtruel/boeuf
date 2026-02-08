# Makefile for Boeuf Project Standardized Workflow

COMPOSE_FILE_PROD := -f docker-compose.yml
COMPOSE_FILE_DEV  := -f docker-compose.yml -f docker-compose.dev.yml

.PHONY: help watch dev-restart prod build down logs test test-backend test-frontend test-playwright docker-test docker-test-backend docker-test-frontend test-quick clean

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

dev-up: ## Start development environment without rebuilding (faster)
	@echo "Starting Dev Environment (No Rebuild)..."
	docker compose $(COMPOSE_FILE_DEV) up -d --remove-orphans

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

test: test-backend test-frontend test-playwright ## Run all tests (local)

test-quick: ## Run all local tests without rebuilding
	@echo "Running All Tests (Local Quick Mode)..."
	@$(MAKE) test-backend
	@$(MAKE) test-frontend

test-backend: ## Run Backend tests (Go, local)
	@echo "Running Backend Tests (Local)..."
	cd backend && go test ./... -v

test-frontend: ## Run Frontend tests (Vitest, local)
	@echo "Running Frontend Tests (Local)..."
	@echo "Installing frontend dependencies..."
	pnpm --dir frontend install
	pnpm --dir frontend exec vitest run

test-playwright: ## Run Playwright E2E tests (local; requires browser install)
	@echo "Running Playwright E2E Tests (Local)..."
	@echo "Installing frontend dependencies..."
	pnpm --dir frontend install
	pnpm --dir frontend exec playwright test --project=chromium --project=firefox

docker-test: docker-test-backend docker-test-frontend ## Run all tests via Docker

docker-test-backend: ## Run Backend tests (Go, Docker)
	@echo "Running Backend Tests (Docker)..."
	docker compose $(COMPOSE_FILE_DEV) run --rm --no-deps backend go test ./... -v

docker-test-frontend: ## Run Frontend tests (Vitest, Docker)
	@echo "Running Frontend Tests (Docker)..."
	# Run tests using the built dev container with all deps pre-installed
	docker compose $(COMPOSE_FILE_DEV) run --rm --no-deps frontend sh -c "pnpm run test:unit --run"

clean: ## Remove artifacts and volumes
	docker compose down -v
	rm -rf frontend/node_modules
	rm -rf frontend/dist
