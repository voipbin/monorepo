# VoIPBin Monorepo Developer Operations
#
# Usage:
#   make help                           Show all targets
#   make doctor                         Check prerequisites
#   make verify SERVICE=bin-call-manager Full 5-step verification
#   make verify-all                     Verify all services
#   make verify-common                  Verify common-handler + all dependents
#   make test SERVICE=bin-call-manager   Run tests (clears cache first)
#   make lint SERVICE=bin-call-manager   Run golangci-lint
#   make generate SERVICE=bin-call-manager Run go generate
#   make build SERVICE=bin-call-manager  Build service binary
#   make dev-up                         Start local dev infrastructure
#   make dev-down                       Stop local dev infrastructure
#   make services                       List all services

.PHONY: help doctor verify verify-all verify-common test lint generate build \
        dev-up dev-down dev-logs services clean

SHELL := /bin/bash
PARALLEL ?= 4
SERVICE ?=

# ── Help ────────────────────────────────────────────────────────────────────

help: ## Show this help
	@echo "VoIPBin Monorepo Developer Operations"
	@echo ""
	@echo "Usage: make <target> [SERVICE=bin-xxx-manager] [PARALLEL=N]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make verify SERVICE=bin-call-manager"
	@echo "  make verify-all PARALLEL=8"
	@echo "  make verify-common --skip-lint"
	@echo "  make dev-up"

# ── Prerequisites ───────────────────────────────────────────────────────────

doctor: ## Check development environment prerequisites
	@./scripts/doctor.sh

# ── Verification (the 5-step workflow) ──────────────────────────────────────

verify: _require-service ## Run full 5-step verification for SERVICE
	@./scripts/verify.sh $(SERVICE)

verify-all: ## Verify ALL Go services (use PARALLEL=N for concurrency)
	@./scripts/verify.sh --all --parallel $(PARALLEL)

verify-common: ## Verify bin-common-handler + update all dependent services
	@./scripts/update-common.sh --parallel $(PARALLEL)

verify-quick: _require-service ## Verify SERVICE without linting (faster)
	@./scripts/verify.sh --skip-lint $(SERVICE)

# ── Individual Steps ────────────────────────────────────────────────────────

test: _require-service ## Run tests for SERVICE (clears cache first)
	@echo "Testing $(SERVICE)..."
	@cd $(SERVICE) && go clean -testcache && go test ./...

lint: _require-service ## Run golangci-lint for SERVICE
	@echo "Linting $(SERVICE)..."
	@cd $(SERVICE) && golangci-lint run -v --timeout 5m

generate: _require-service ## Run go generate for SERVICE
	@echo "Generating code for $(SERVICE)..."
	@cd $(SERVICE) && go generate ./...

tidy: _require-service ## Run go mod tidy + vendor for SERVICE
	@echo "Tidying $(SERVICE)..."
	@cd $(SERVICE) && go mod tidy && go mod vendor

build: _require-service ## Build SERVICE binary
	@echo "Building $(SERVICE)..."
	@cd $(SERVICE) && go build -o /dev/null ./cmd/...

# ── Local Development Infrastructure ────────────────────────────────────────

dev-up: ## Start local dev infrastructure (MySQL, Redis, RabbitMQ)
	@echo "Starting local development infrastructure..."
	@docker compose -f docker-compose.dev.yml up -d
	@echo ""
	@echo "Services:"
	@echo "  MySQL:    localhost:3306  (user: voipbin, pass: voipbin)"
	@echo "  Redis:    localhost:6379  (pass: voipbin)"
	@echo "  RabbitMQ: localhost:5672  (user: guest, pass: guest)"
	@echo "  RabbitMQ: localhost:15672 (management UI)"

dev-down: ## Stop local dev infrastructure
	@docker compose -f docker-compose.dev.yml down

dev-logs: ## Show logs from local dev infrastructure
	@docker compose -f docker-compose.dev.yml logs -f

dev-reset: ## Reset local dev infrastructure (destroy volumes)
	@docker compose -f docker-compose.dev.yml down -v
	@echo "Volumes destroyed. Run 'make dev-up' to recreate."

# ── Information ─────────────────────────────────────────────────────────────

services: ## List all services in the monorepo
	@echo "Go services:"
	@for dir in bin-*/; do \
		svc=$$(basename "$$dir"); \
		if [ -f "$$dir/go.mod" ]; then \
			echo "  $$svc"; \
		fi; \
	done
	@echo ""
	@echo "Other:"
	@for dir in voip-*/; do \
		echo "  $$(basename $$dir)"; \
	done

# ── Cleanup ─────────────────────────────────────────────────────────────────

clean: _require-service ## Clean build artifacts and test cache for SERVICE
	@echo "Cleaning $(SERVICE)..."
	@cd $(SERVICE) && go clean -testcache -cache

clean-all: ## Clean test cache for ALL services
	@echo "Cleaning all services..."
	@for dir in bin-*/; do \
		if [ -f "$$dir/go.mod" ]; then \
			(cd "$$dir" && go clean -testcache) 2>/dev/null; \
		fi; \
	done
	@echo "Done."

# ── Internal ────────────────────────────────────────────────────────────────

_require-service:
ifndef SERVICE
	$(error SERVICE is required. Usage: make <target> SERVICE=bin-xxx-manager)
endif
	@if [ ! -d "$(SERVICE)" ]; then \
		echo "Error: $(SERVICE) directory not found"; \
		exit 1; \
	fi
