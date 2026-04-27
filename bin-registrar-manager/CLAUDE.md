# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-registrar-manager` manages SIP registrations for VoIP extensions and trunks in the VoIPbin platform. It handles the lifecycle of Asterisk PJSIP endpoints, contacts, authentication, and Address of Record (AOR) configurations.

**Key Concepts:**
- **Extension**: SIP endpoint for an individual user within a customer account; backed by Asterisk `ps_endpoints`/`ps_aors`/`ps_auths` tables.
- **Trunk**: SIP trunk for carrier/provider connections; supports basic (user/pass) and/or IP-based authentication.
- **Contact**: An active SIP registration (read from Asterisk `ps_contacts`, cached in Redis).
- **Two-database architecture**: Service connects to both the Asterisk DB (`ps_*` tables) and the bin-manager DB (extensions, trunks, sip_auth tables).

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-registrar-manager`.

## Architecture

### Service Communication Pattern

The service follows a handler-based architecture with these key layers:

1. **Listen Handler** (`pkg/listenhandler/`) - Consumes RPC requests from RabbitMQ queue `bin-manager.registrar-manager.request`, processes REST-like URIs (e.g., `/v1/extensions`, `/v1/trunks`), and returns responses
2. **Subscribe Handler** (`pkg/subscribehandler/`) - Subscribes to events from other services (e.g., customer deletions) on queue `bin-manager.registrar-manager.subscribe`
3. **Domain Handlers** - Business logic layers:
   - `extensionhandler` - Manages SIP extensions for customers
   - `trunkhandler` - Manages SIP trunks for customers
   - `contacthandler` - Manages active SIP contacts
4. **Database Handler** (`pkg/dbhandler/`) - Abstracts database operations for both Asterisk and bin-manager databases
5. **Cache Handler** (`pkg/cachehandler/`) - Redis-backed caching for contacts and other data

### Key Concepts

- **Extensions**: SIP endpoints for individual users within a customer account. Each extension has:
  - Asterisk resources: `ps_endpoints`, `ps_aors`, `ps_auths`
  - A unique extension number within the customer's realm
  - Username/password authentication
  - Domain name format: customer-specific subdomain

- **Trunks**: SIP trunks for carrier/provider connections. Each trunk has:
  - Multiple authentication types supported: basic (user/pass) and/or IP-based
  - Custom domain names (validated against `^([a-zA-Z]{1})([a-zA-Z0-9\-\.]{1,30})$`)
  - Optional allowed IP lists for IP-based auth

- **Asterisk Integration**: The service directly manages Asterisk PJSIP tables:
  - `ps_endpoints` - SIP endpoint configurations
  - `ps_aors` - Address of Record mappings
  - `ps_auths` - Authentication credentials
  - `ps_contacts` - Active registrations (read from database, cached in Redis)

### Database Architecture

This service uses **two separate databases**:

- **Asterisk DB** (`config.DatabaseDSNAsterisk`): Asterisk PJSIP tables (`ps_*` tables)
- **Bin Manager DB** (`config.DatabaseDSNBin`): Service's own tables (extensions, trunks, sip_auth)

Both connections are established in `main.go` and passed to handlers.

### Event Flow

**Request Processing**:
```
RabbitMQ Request → ListenHandler.processRequest() → Domain Handler → DBHandler → Database
                                                                                  ↓
                                                  NotifyHandler ← Event Published
```

**Event Subscription**:
```
RabbitMQ Event (e.g., customer.deleted) → SubscribeHandler → Domain Handler → Cleanup Resources
```

## Request Routing

ListenHandler routes RPC requests using regex patterns matching REST-like URIs (see `pkg/listenhandler/main.go`):

**Extensions API (`/v1/extensions/*`):**
- `POST /v1/extensions` — Create extension
- `GET /v1/extensions?<filters>` — List extensions
- `GET /v1/extensions/<uuid>` — Get extension
- `PUT /v1/extensions/<uuid>` — Update extension
- `DELETE /v1/extensions/<uuid>` — Delete extension

**Trunks API (`/v1/trunks/*`):**
- `POST /v1/trunks` — Create trunk
- `GET /v1/trunks?<filters>` — List trunks
- `GET /v1/trunks/<uuid>` — Get trunk
- `PUT /v1/trunks/<uuid>` — Update trunk
- `DELETE /v1/trunks/<uuid>` — Delete trunk

**Contacts API (`/v1/contacts/*`):**
- `GET /v1/contacts?<filters>` — List active SIP contacts
- `GET /v1/contacts/<uuid>` — Get contact details

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.customer-manager.event**: `customer_deleted` → triggers cleanup of extensions/trunks owned by the deleted customer.

## Configuration

The service uses **Cobra + Viper** for configuration management (migrated from flag-based config). Configuration is loaded via:

- Command-line flags (e.g., `--rabbitmq_address`)
- Environment variables (e.g., `RABBITMQ_ADDRESS`)

Key configuration parameters:
- `database_dsn_bin` / `DATABASE_DSN_BIN` - Bin manager database connection
- `database_dsn_asterisk` / `DATABASE_DSN_ASTERISK` - Asterisk database connection
- `rabbitmq_address` / `RABBITMQ_ADDRESS` - RabbitMQ connection string
- `redis_address` / `REDIS_ADDRESS` - Redis cache connection
- `domain_name_extension` / `DOMAIN_NAME_EXTENSION` - Base domain for extensions
- `domain_name_trunk` / `DOMAIN_NAME_TRUNK` - Base domain for trunks
- `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` - Metrics endpoint (default: `/metrics`)
- `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` - Metrics server address (default: `:2112`)

Config singleton is accessed via `config.Get()`.

## Common Commands

### Building
```bash
# Build binary
go build -o bin/registrar-manager ./cmd/registrar-manager

# Docker build (from monorepo root)
docker build -f bin-registrar-manager/Dockerfile -t registrar-manager .
```

### Testing
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/extensionhandler/...

# Run with verbose output
go test -v ./pkg/trunkhandler/...

# Run specific test
go test -v -run TestExtensionCreate ./pkg/extensionhandler/
```

### Mock Generation
Mocks are generated using `mockgen` with go:generate directives:
```bash
# Regenerate all mocks
go generate ./...

# Regenerate mocks for specific package
go generate ./pkg/extensionhandler/
```

### Running Locally
```bash
go run ./cmd/registrar-manager \
  --database_dsn_asterisk "user:pass@tcp(localhost:3306)/asterisk" \
  --database_dsn_bin "user:pass@tcp(localhost:3306)/bin_manager" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "localhost:6379" \
  --redis_database 1 \
  --domain_name_extension "ext.example.com" \
  --domain_name_trunk "trunk.example.com"
```

### registrar-control CLI Tool

The `registrar-control` CLI provides command-line management of SIP extensions and trunks. **All output is JSON format** (stdout), logs go to stderr.

**Build:**
```bash
# From monorepo root or bin-registrar-manager directory
go build -o bin/registrar-control ./cmd/registrar-control

# Verify build
./bin/registrar-control --help
```

**Configuration:**
The CLI uses the same configuration system as the registrar-manager service (Cobra + Viper):
- Command-line flags (e.g., `--database_dsn_bin`, `--rabbitmq_address`)
- Environment variables (e.g., `DATABASE_DSN_BIN`, `RABBITMQ_ADDRESS`)

Required configuration:
- `database_dsn_bin` / `DATABASE_DSN_BIN`
- `database_dsn_asterisk` / `DATABASE_DSN_ASTERISK`
- `rabbitmq_address` / `RABBITMQ_ADDRESS`
- `redis_address` / `REDIS_ADDRESS`
- `domain_name_extension` / `DOMAIN_NAME_EXTENSION`
- `domain_name_trunk` / `DOMAIN_NAME_TRUNK`

**Extension Commands:**

```bash
# Create extension - returns created extension JSON
registrar-control extension create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --username "user1001" \
  --password "secret123" \
  --extension_number "1001"

# Get extension - returns extension JSON
registrar-control extension get --id "<extension-uuid>"

# List extensions - returns JSON array
registrar-control extension list \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --limit 50

# Update extension - returns updated extension JSON
registrar-control extension update \
  --id "<extension-uuid>" \
  --password "newpass123"

# Delete extension - returns deleted extension JSON
registrar-control extension delete --id "<extension-uuid>"
```

**Trunk Commands:**

```bash
# Create trunk with basic authentication - returns created trunk JSON
registrar-control trunk create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --domain "voip-carrier.example.com" \
  --name "Primary Carrier" \
  --username "trunk_user" \
  --password "trunk_pass"

# Create trunk with IP-based authentication - returns created trunk JSON
registrar-control trunk create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --domain "voip-carrier2.example.com" \
  --name "Secondary Carrier" \
  --allowed_ips "203.0.113.1,203.0.113.2"

# Create trunk with both authentication types - returns created trunk JSON
registrar-control trunk create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --domain "voip-carrier3.example.com" \
  --name "Hybrid Carrier" \
  --username "trunk_user" \
  --password "trunk_pass" \
  --allowed_ips "203.0.113.1,203.0.113.2"

# Get trunk - returns trunk JSON
registrar-control trunk get --id "<trunk-uuid>"

# List trunks - returns JSON array
registrar-control trunk list \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b"

# Update trunk - returns updated trunk JSON
registrar-control trunk update \
  --id "<trunk-uuid>" \
  --allowed_ips "203.0.113.1,203.0.113.2,203.0.113.3"

# Delete trunk - returns deleted trunk JSON
registrar-control trunk delete --id "<trunk-uuid>"
```

**Required Flags:**
All required flags must be provided via command line - no interactive prompts.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (database, request, notify, socket)
- `monorepo/bin-customer-manager`: Customer event models

When modifying these dependencies, remember that Go modules use `replace` directives in `go.mod` to point to local paths.

## Testing Patterns

Tests use:
- `go.uber.org/mock` for mocking
- Table-driven tests with `t.Run(name, func(t *testing.T){})`
- In-memory SQLite for database tests (see `pkg/dbhandler/main_test.go`)

```go
tests := []struct {
    name      string
    input     InputType
    mockSetup func(*MockHandler)
    expectRes ResultType
    expectErr bool
}{
    {"success case", input1, setupMock1, expected1, false},
    {"error case", input2, setupMock2, nil, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // test implementation
    })
}
```

## Key Implementation Details

### Handler Interface Pattern
Business logic is defined as interfaces (e.g., `ExtensionHandler`, `TrunkHandler`) with mock implementations. To add functionality: define the method on the interface in `main.go`, implement in the same package, regenerate mocks.

### Database Pattern
All database operations go through the `DBHandler` interface (defined in `pkg/dbhandler/main.go`). New operations belong in that interface; use `Masterminds/squirrel` for SQL building; handle `ErrNotFound` for missing records.

### Models
Models live in `models/` with separate packages per entity. Each model may include the struct definition, event types (e.g., `EventExtensionCreated`), webhook types, and helper methods.

### Multi-Resource Atomicity
Creating an Extension touches three Asterisk tables (`ps_endpoints`, `ps_aors`, `ps_auths`). Deletes must clean up all three. Wrap in a transaction where possible.

### Domain Names
Extensions and trunks must have valid domain names set via `common.SetBaseDomainNames()` during startup.

### Contact Caching
Asterisk contacts are cached in Redis for performance; always invalidate the cache when modifying endpoints.

### Logging
Uses `logrus` with structured fields and the Joonix formatter for Stackdriver-compatible JSON logs. Log level: Debug (configured in `config.initLog()`).

## Prometheus Metrics

Metrics are exposed on the configured Prometheus endpoint (default `:2112/metrics`):
- `registrar_manager_extension_create_total` — counter of extensions created
- `registrar_manager_extension_delete_total` — counter of extensions deleted
- `registrar_manager_trunk_create_total` — counter of trunks created
- `registrar_manager_trunk_delete_total` — counter of trunks deleted
- `registrar_manager_receive_request_process_time` — histogram of request processing latency

## Common Tasks

### Adding a New API Endpoint

1. Add regex pattern in `pkg/listenhandler/main.go` (e.g., `regV1ExtensionsID`)
2. Add case in `processRequest()` switch statement
3. Implement handler method (e.g., `processV1ExtensionsIDGet()`) in appropriate file (`v1_extensions.go`)
4. Add request/response models in `pkg/listenhandler/models/`

### Adding a New Event Handler

1. Add event handling method to domain handler interface (e.g., `EventCUCustomerDeleted`)
2. Implement in domain handler's `event.go` file
3. Wire up in `pkg/subscribehandler/` to process subscribed events
4. Add to `subscribeTargets` in `main.go:runSubscribe()`

### Modifying Asterisk Resources

When changing how Asterisk PJSIP resources are created/updated:
1. Update model structs in `models/ast*` packages
2. Modify corresponding `pkg/dbhandler/ast_*.go` files
3. Be careful with Asterisk's specific requirements (e.g., contact management, AOR settings)
4. Test against actual Asterisk instance, not just SQLite

## Important Notes

- **Domain Names**: Extensions and trunks must have valid domain names set via `common.SetBaseDomainNames()` during startup
- **Contact Caching**: Asterisk contacts are cached in Redis for performance; always invalidate cache when modifying endpoints
- **Graceful Shutdown**: Service handles SIGINT/SIGTERM/SIGHUP via channel-based signal handling
- **Transaction Safety**: Database operations that modify multiple tables (Extension = endpoint + aor + auth) should ideally be in transactions
- **Resource Cleanup**: Deleting extensions/trunks must clean up all related Asterisk resources (endpoint, aor, auth)
