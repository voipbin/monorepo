# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the `bin-customer-manager` service, part of the VoIPbin monorepo. It manages customers (tenant organizations) and their access keys (API credentials), including CRUD operations, billing integration, and event publishing.

**Key Concepts:**
- **Customer**: Top-level tenant organization with billing account, contact info, and webhook configuration for receiving event notifications
- **Access Key**: API credentials (tokens) associated with a customer for authenticating API requests
- **Soft Deletes**: Records use `tm_delete` timestamp (default `"9999-01-01 00:00:00.000000"` for active records)
- **Webhook Events**: Customer changes trigger webhook notifications via RabbitMQ for async delivery

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.customer-manager.request`, processes them, and returns responses using REST-like URI patterns
- **NotifyHandler**: Publishes events to exchange `bin-manager.customer-manager.event` when customer/accesskey state changes
- **RequestHandler**: Makes RPC calls to other services (billing-manager, agent-manager) for cross-service validation

### Core Components

```
cmd/customer-manager/main.go
    ├── initCache() -> pkg/cachehandler (Redis)
    ├── startServices()
        ├── pkg/dbhandler (MySQL via raw SQL queries)
        ├── pkg/customerhandler (Business logic for customers)
        ├── pkg/accesskeyhandler (Business logic for access keys)
        └── pkg/listenhandler (RabbitMQ RPC routing)
```

**Layer Responsibilities:**
- `models/customer/`: Customer data structures, webhook events, validation
- `models/accesskey/`: Access key data structures, webhook events
- `pkg/customerhandler/`: Business logic for customer operations, cross-service validation
- `pkg/accesskeyhandler/`: Business logic for access key operations
- `pkg/dbhandler/`: Database operations using parameterized SQL queries
- `pkg/cachehandler/`: Redis caching for customer/accesskey lookups (cache-first pattern)
- `pkg/listenhandler/`: RabbitMQ RPC request routing with REST-like path patterns

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:

**Customers:**
- `POST /v1/customers` - Create customer
- `GET /v1/customers?<filters>` - List customers (pagination via page_size/page_token)
- `GET /v1/customers/<uuid>` - Get customer
- `PUT /v1/customers/<uuid>` - Update customer basic info
- `DELETE /v1/customers/<uuid>` - Delete customer
- `PUT /v1/customers/<uuid>/billing_account_id` - Link billing account

**Access Keys:**
- `POST /v1/accesskeys` - Create access key
- `GET /v1/accesskeys?<filters>` - List access keys
- `GET /v1/accesskeys/<id>` - Get access key
- `PUT /v1/accesskeys/<id>` - Update access key basic info
- `DELETE /v1/accesskeys/<id>` - Delete access key

### Event Publishing

Customer operations publish events to `bin-manager.customer-manager.event`:
- `customer_created`, `customer_updated`, `customer_deleted`
- `accesskey_created`, `accesskey_updated`, `accesskey_deleted`

Webhook notifications are created and queued for async delivery to customer webhook URIs.

### Configuration

Uses **Cobra + Viper** pattern (see `internal/config/`):
- Command-line flags and environment variables (e.g., `--rabbitmq_address` or `RABBITMQ_ADDRESS`)
- Config loaded once via `sync.Once` in `LoadGlobalConfig()`
- Required: `database_dsn`, `rabbitmq_address`, `redis_address`, `prometheus_endpoint`

## Common Commands

### Build
```bash
# From monorepo root (expects parent directory context for replacements)
cd /path/to/monorepo/bin-customer-manager
go build -o bin/customer-manager ./cmd/customer-manager
go build -o bin/customer-control ./cmd/customer-control
```

### Test
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for specific package
go test -v ./pkg/customerhandler/...

# Run single test
go test -v ./pkg/customerhandler -run Test_Delete
```

### Generate Mocks
```bash
# Generate all mocks (uses go:generate directives)
go generate ./...

# Mocks are created via mockgen for interfaces in:
# - pkg/customerhandler/main.go -> mock_main.go
# - pkg/accesskeyhandler/main.go -> mock_main.go
# - pkg/dbhandler/main.go -> mock_main.go
# - pkg/listenhandler/main.go -> mock_main.go
# - pkg/cachehandler/main.go -> mock_main.go
```

### Lint
```bash
# Run golangci-lint (CI uses golangci/golangci-lint:latest image)
golangci-lint run -v --timeout 5m

# Run vet
go vet $(go list ./...)
```

## customer-control CLI Tool

A command-line tool for managing customers directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create customer - returns created customer JSON
./bin/customer-control customer create --email <email> [--name] [--detail] [--phone-number] [--address] [--webhook-method POST] [--webhook-uri]

# Get customer - returns customer JSON
./bin/customer-control customer get --id <uuid>

# List customers - returns JSON array
./bin/customer-control customer list [--limit 100] [--token]

# Update customer basic info - returns updated customer JSON
./bin/customer-control customer update --id <uuid> --email <email> [--name] [--detail] [--phone-number] [--address] [--webhook-method] [--webhook-uri]

# Update customer billing account ID - returns updated customer JSON
./bin/customer-control customer update-billing-account --id <uuid> --billing-account-id <uuid>

# Delete customer - returns deleted customer JSON
./bin/customer-control customer delete --id <uuid>
```

Uses same environment variables as customer-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Run Locally
```bash
# With environment variables
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/customer-manager

# Or with flags
./bin/customer-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/voipbin" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379"
```

### CLI Tool (customer-control)

A command-line tool for managing customers. **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create customer - returns created customer JSON
./bin/customer-control customer create --email user@example.com [--name] [--detail] [--phone_number] [--address] [--webhook_method] [--webhook_uri]

# Get customer - returns customer JSON
./bin/customer-control customer get --id <uuid>

# List customers - returns JSON array
./bin/customer-control customer list [--limit 100] [--token]

# Delete customer - returns deleted customer JSON
./bin/customer-control customer delete --id <uuid>
```

Uses same environment variables as customer-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Docker
```bash
# Build (expects monorepo root context)
docker build -f Dockerfile -t customer-manager:latest ../..

# CI builds from monorepo root with:
# docker build --tag $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
```

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler)
- `monorepo/bin-billing-manager`: Billing account models and validation
- `monorepo/bin-agent-manager`: Agent username validation for customer creation

**Important**: Builds and Docker images assume parent monorepo directory context is available.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces are generated in same package as interface definition (e.g., `pkg/dbhandler/mock_main.go`)
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods

Example test structure:
```go
tests := []struct {
    name    string
    // inputs
    // expected results
}{
    {"normal", ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // Setup mocks and test
    })
}
```

## Key Implementation Details

### Database Queries
Uses **parameterized raw SQL** (not Squirrel builder):
```go
query := `SELECT * FROM customer_customers
          WHERE id = ? AND tm_delete = ?`
err := db.QueryRowContext(ctx, query, id, DefaultTimeStamp).Scan(...)
```

### Pagination
Uses **cursor-based pagination** with timestamp tokens:
```go
// Query pattern: tm_create < token ORDER BY tm_create DESC
pageToken := req.GetPageToken()  // Timestamp string
pageSize := req.GetPageSize()    // Limit
```

### Cache Strategy
Cache-first pattern for reads:
1. Attempt Redis cache lookup by ID
2. On miss, query database
3. Mutations invalidate cache and update database

### Soft Deletes
Records use `tm_delete` timestamp. Default value `"9999-01-01 00:00:00.000000"` indicates active records. Deletion sets `tm_delete` to current timestamp.

### Cross-Service Validation
Customer creation validates:
- Email uniqueness (local database check)
- Username conflicts with agent-manager (RPC call)
- Billing account existence via billing-manager (optional)

### Webhook Configuration
Customers can configure webhook endpoints:
- `webhook_method`: HTTP method (GET/POST/PUT/DELETE)
- `webhook_uri`: Target URL
- Events serialized to JSON and published to RabbitMQ for async delivery

## CI/CD

GitLab CI pipeline (`.gitlab-ci.yml` - deleted, check sibling services):
1. **ensure**: `go mod download && go mod vendor`
2. **test**: `golangci-lint`, `go vet`, `go test`
3. **build**: Docker build and push to registry
4. **release**: Deploy to k8s using kustomize

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)

## Kubernetes Deployment

Deployment configuration in `k8s/deployment.yml`:
- Replicas: 2 (high availability)
- Resource limits: 20m CPU, 30M memory
- Prometheus scraping on `:2112/metrics`
- Consumer group: "call-manager" for RabbitMQ shared processing
