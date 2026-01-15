# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Service Overview

**bin-registrar-manager** manages SIP registrations for VoIP extensions and trunks in the voipbin platform. It handles the lifecycle of Asterisk PJSIP endpoints, contacts, authentication, and Address of Record (AOR) configurations.

This is part of a Go monorepo where services communicate via RabbitMQ message passing and share database access (both bin-manager and asterisk databases).

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

## Development Commands

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

The `registrar-control` CLI provides command-line management of SIP extensions and trunks. It follows the same patterns as `agent-control` from `bin-agent-manager`.

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
# Create extension (prompts for password interactively)
registrar-control extension create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --username "user1001" \
  --extension_number "1001"

# Get extension details (includes registration status and contacts)
registrar-control extension get --id "<extension-uuid>"

# List extensions for a customer
registrar-control extension list \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --limit 50

# List with JSON output
registrar-control extension list \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --format json

# Update extension password
registrar-control extension update \
  --id "<extension-uuid>" \
  --password "newpass123"

# Delete extension (prompts for confirmation)
registrar-control extension delete --id "<extension-uuid>"

# Delete without confirmation prompt
registrar-control extension delete --id "<extension-uuid>" --force
```

**Trunk Commands:**

```bash
# Create trunk with basic authentication
registrar-control trunk create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --domain "voip-carrier.example.com" \
  --name "Primary Carrier" \
  --username "trunk_user" \
  --password "trunk_pass"

# Create trunk with IP-based authentication
registrar-control trunk create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --domain "voip-carrier2.example.com" \
  --name "Secondary Carrier" \
  --allowed_ips "203.0.113.1,203.0.113.2"

# Create trunk with both authentication types
registrar-control trunk create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --domain "voip-carrier3.example.com" \
  --name "Hybrid Carrier" \
  --username "trunk_user" \
  --password "trunk_pass" \
  --allowed_ips "203.0.113.1,203.0.113.2"

# Get trunk details (includes auth config and registration status)
registrar-control trunk get --id "<trunk-uuid>"

# List trunks for a customer
registrar-control trunk list \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --format json

# Update trunk allowed IPs
registrar-control trunk update \
  --id "<trunk-uuid>" \
  --allowed_ips "203.0.113.1,203.0.113.2,203.0.113.3"

# Delete trunk (prompts for confirmation)
registrar-control trunk delete --id "<trunk-uuid>"
```

**Interactive Prompts:**
- Missing required fields trigger interactive prompts
- Password fields use `survey.Password` (hidden input)
- Delete operations show resource details and request confirmation
- Use `--force` flag to skip delete confirmations

**Output Formats:**
- Default: Human-readable text with formatted tables
- JSON: Use `--format json` flag for machine-parseable output

## Code Patterns

### Handler Interface Pattern
Most business logic is defined as interfaces (e.g., `ExtensionHandler`, `TrunkHandler`) with mock implementations for testing. When adding new functionality:

1. Add method to interface in `main.go`
2. Implement in same package (e.g., `extension.go`)
3. Regenerate mocks with `go generate`

### Database Pattern
All database operations go through `DBHandler` interface. When adding new DB operations:

1. Add method signature to `pkg/dbhandler/main.go` interface
2. Implement in appropriate file (e.g., `ast_auth.go` for Asterisk auth, `extension.go` for extensions)
3. Use `Masterminds/squirrel` for SQL building
4. Handle `ErrNotFound` for missing records

### Model Pattern
Models are in `models/` directory with separate packages for each entity type. Each model may include:
- Struct definition
- Event types (e.g., `EventExtensionCreated`)
- Webhook types
- Helper methods

### Testing Pattern
Tests use:
- `go.uber.org/mock` for mocking
- Table-driven tests with `t.Run(name, func(t *testing.T){})`
- In-memory SQLite for database tests (see `pkg/dbhandler/main_test.go`)

## Monorepo Dependencies

This service imports from sibling packages:
- `monorepo/bin-common-handler` - Shared handlers (database, request, notify, socket)
- `monorepo/bin-customer-manager` - Customer models for event handling

When modifying these dependencies, remember that Go modules use `replace` directives in `go.mod` to point to local paths.

## Observability

### Prometheus Metrics
Metrics are exposed on the configured prometheus endpoint (default `:2112/metrics`):
- `registrar_manager_extension_create_total` - Total extensions created
- `registrar_manager_extension_delete_total` - Total extensions deleted
- `registrar_manager_trunk_create_total` - Total trunks created
- `registrar_manager_trunk_delete_total` - Total trunks deleted
- `registrar_manager_receive_request_process_time` - Request processing latency histogram

### Logging
- Uses `logrus` with structured fields
- Joonix formatter for stackdriver-compatible JSON logs
- Log level: Debug (configured in `config.initLog()`)

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
