# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-billing-manager` is a Go service within the VoIPBin monorepo that manages billing operations for telecommunications services. It tracks account balances, processes billing for calls/SMS/numbers, handles payment information, and publishes billing events to the platform.

This service operates as an event-driven microservice using RabbitMQ for message passing and Redis for caching.

## Development Commands

### Testing
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)

# View coverage report
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for a specific package
go test -v ./pkg/accounthandler/...

# Run a single test
go test -v ./pkg/accounthandler -run Test_IsValidBalance
```

### Building
```bash
# Build the service binary
go build -o ./bin/billing-manager ./cmd/billing-manager/

# Build the CLI tool
go build -o ./bin/billing-control ./cmd/billing-control/

# Build using Docker (from monorepo root)
docker build -t billing-manager:latest -f bin-billing-manager/Dockerfile .
```

### CLI Tool: billing-control

A command-line tool for managing billing accounts and viewing billing records directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Account operations (all return JSON)
billing-control account create --customer-id <uuid> [--name] [--detail] [--payment-type] [--payment-method]
billing-control account get --id <uuid>
billing-control account list [--limit 100] [--token] [--customer-id <uuid>]
billing-control account delete --id <uuid>
billing-control account add-balance --id <uuid> --amount <float>
billing-control account subtract-balance --id <uuid> --amount <float>

# Billing operations (read-only, returns JSON)
billing-control billing get --id <uuid>
billing-control billing list [--limit 100] [--token] [--customer-id <uuid>] [--account-id <uuid>]

```

Uses same environment variables as billing-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Code Quality
```bash
# Run go vet
go vet $(go list ./...)

# Run golangci-lint (comprehensive linting)
golangci-lint run -v --timeout 5m
```

## billing-control CLI Tool

A command-line tool for managing billing accounts and records directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Account commands
./bin/billing-control account create --customer-id <uuid> [--name] [--detail] [--payment-type prepaid] [--payment-method]
./bin/billing-control account get --id <uuid>
./bin/billing-control account list [--customer-id <uuid>] [--limit 100] [--token]
./bin/billing-control account update --id <uuid> --name <name> [--detail]
./bin/billing-control account update-payment-info --id <uuid> --payment-type <prepaid> --payment-method <credit card>
./bin/billing-control account delete --id <uuid>
./bin/billing-control account add-balance --id <uuid> --amount <float>
./bin/billing-control account subtract-balance --id <uuid> --amount <float>

# Billing commands
./bin/billing-control billing get --id <uuid>
./bin/billing-control billing list [--customer-id <uuid>] [--account-id <uuid>] [--limit 100] [--token]

```

Uses same environment variables as billing-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Dependency Management
```bash
# Download dependencies
go mod download

# Vendor dependencies
go mod vendor

# Update specific dependency
go get <package>@<version>
go mod tidy
go mod vendor
```

### Mock Generation
```bash
# Generate mocks using go:generate directives
go generate ./...

# Mocks are created for interfaces in:
# - pkg/accounthandler/main.go -> mock_main.go
# - pkg/billinghandler/main.go -> mock_main.go
# - pkg/dbhandler/main.go -> mock_main.go
# - pkg/failedeventhandler/main.go -> mock_main.go
# - pkg/listenhandler/main.go -> mock_main.go
# - pkg/subscribehandler/main.go -> mock_main.go
```

### Run Locally
```bash
# With environment variables
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/billing-manager

# Or with flags
./bin/billing-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/bin_manager" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379" \
  --redis_database 1
```

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.billing-manager.request`, processes them, and returns responses
- **SubscribeHandler** (`pkg/subscribehandler/`): Subscribes to events from other services (call-manager, message-manager, customer-manager, number-manager) to track billable events
- **NotifyHandler**: Publishes events to queue `bin-manager.billing-manager.event` when billing/account state changes

### Core Components

```
cmd/billing-manager/main.go
    ├── Database (MySQL) connection
    ├── Redis cache connection
    ├── run()
        ├── pkg/dbhandler (MySQL + Redis caching)
        ├── pkg/cachehandler (Redis)
        ├── pkg/accounthandler (Account/balance business logic)
        ├── pkg/billinghandler (Billing records business logic)
        ├── runListen() -> pkg/listenhandler
        └── runSubscribe() -> pkg/subscribehandler
```

**Layer Responsibilities:**
- `models/account/`: Account data structures (Account, PaymentType, PaymentMethod)
- `models/billing/`: Billing data structures (Billing, ReferenceType, Status, default costs)
- `models/failedevent/`: Failed event data structures for retry persistence (FailedEvent, Status, Field)
- `pkg/accounthandler/`: Account operations (balance management, payment info, validation)
- `pkg/billinghandler/`: Billing operations (create/track billing records, process events)
- `pkg/dbhandler/`: Database operations for accounts, billings, and failed events
- `pkg/cachehandler/`: Redis caching for account lookups
- `pkg/listenhandler/`: RabbitMQ RPC request routing (REST-like paths: `/v1/accounts`, `/v1/billings`)
- `pkg/subscribehandler/`: Event consumption from other services for billing triggers

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:

**Accounts:**
- `GET /v1/accounts?<filters>` - List accounts (pagination via page_size/page_token)
- `POST /v1/accounts` - Create account
- `GET /v1/accounts/<uuid>` - Get account
- `PUT /v1/accounts/<uuid>` - Update account basic info
- `DELETE /v1/accounts/<uuid>` - Delete account
- `POST /v1/accounts/<uuid>/balance_add_force` - Force add balance
- `POST /v1/accounts/<uuid>/balance_subtract_force` - Force subtract balance
- `POST /v1/accounts/<uuid>/is_valid_balance` - Check if account has sufficient balance
- `PUT /v1/accounts/<uuid>/payment_info` - Update payment information

**Billings:**
- `GET /v1/billings?<filters>` - List billing records (pagination via page_size/page_token)

### Event Subscriptions

SubscribeHandler processes events from:
- **call-manager**: `call_progressing`, `call_hangup` - Creates/updates billing for calls
- **message-manager**: `message_created` - Creates billing for SMS messages
- **customer-manager**: `customer_created`, `customer_deleted` - Creates/deletes billing accounts
- **number-manager**: `number_created`, `number_renewed` - Creates billing for number purchases/renewals

### Configuration

Uses **Viper + pflag** pattern (see `cmd/billing-manager/init.go`):
- Command-line flags and environment variables (e.g., `--database_dsn` or `DATABASE_DSN`)
- Configuration loaded in `init()` via `initVariable()`
- Required: `database_dsn`, `rabbitmq_address`, `redis_address`
- Optional: `redis_password`, `redis_database`, `prometheus_endpoint`, `prometheus_listen_address`

## Key Data Models

### Account (`models/account/account.go`)
Represents a billing account with balance tracking:
- **Type**: `admin` (unlimited balance) or `normal` (requires balance)
- **Balance**: Current balance in USD
- **PaymentType**: `prepaid` or none
- **PaymentMethod**: `credit card` or none
- **Soft Delete**: Uses `tm_delete` timestamp (default `9999-01-01 00:00:00.000000`)

### Billing (`models/billing/billing.go`)
Represents a billing record for a billable event:
- **ReferenceType**: `call`, `sms`, `number`, `number_renew`
- **Status**: `progressing`, `end`, `pending`, `finished`
- **Default Costs**:
  - Call: $0.020 per unit
  - SMS: $0.008 per unit
  - Number: $5.00 per unit
- **BillingUnitCount**: Number of billing units (e.g., minutes for calls)
- **CostTotal**: Total cost calculated from cost_per_unit × billing_unit_count

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler)
- `monorepo/bin-call-manager`: Models for call events
- `monorepo/bin-message-manager`: Models for message events
- `monorepo/bin-customer-manager`: Models for customer events
- `monorepo/bin-number-manager`: Models for number events

**Important**: Builds and Docker images assume parent monorepo directory context is available.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces are generated in same package as interface definition (e.g., `pkg/dbhandler/mock_main.go`)
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods
- Example: `pkg/accounthandler/balance_test.go` tests balance validation with various scenarios

Example test structure:
```go
tests := []test{
    {
        name: "account has enough balance",
        // test fields...
    },
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

### Access Control

**IMPORTANT: Billing and Billing Account resources require CustomerAdmin permission ONLY.**

Access control is handled by `bin-api-manager`, NOT by this service. This service processes RPC requests from bin-api-manager and trusts that authorization has already been performed.

- bin-api-manager validates JWT tokens and checks permissions
- Only users with CustomerAdmin permission can access billing/account endpoints
- Manager-level users do NOT have access to billing resources

See `bin-api-manager/CLAUDE.md` for complete authentication and authorization patterns.

### Balance Validation
Unlimited plan accounts always have valid balance regardless of actual balance:
```go
if a.PlanType == account.PlanTypeUnlimited {
    return true, nil
}
```

### Soft Deletes
Records use `tm_delete` timestamp. Default value `9999-01-01 00:00:00.000000` indicates active records.

### Cache Strategy
Redis cache is used for account lookups. Database is source of truth; cache updates on mutations.

### Event Processing Flow

**Call Billing Flow:**
1. `call_progressing` event received → Create billing record with status `progressing`
2. Track call duration in real-time
3. `call_hangup` event received → Update billing record with final duration and status `end`
4. Subtract cost from account balance

**Number Billing Flow:**
1. `number_created` or `number_renewed` event received
2. Create billing record with reference type `number` or `number_renew`
3. Charge account the number cost immediately

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `billing_manager_receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)
- `billing_manager_receive_subscribe_event_process_time` - Histogram of event processing time (labels: publisher, type)

## Database Schema

While not explicitly in code, the service expects these tables:
- `billing_accounts`: Billing accounts with balance tracking
- `billing_billings`: Billing records for all billable events
- `billing_failed_events`: Failed event records for retry persistence (hard delete on success)

The accounts and billings tables use soft delete pattern with `tm_delete` column. The failed events table uses hard delete (records are removed after successful retry).
