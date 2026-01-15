# Design: registrar-control CLI Tool

**Date**: 2026-01-15
**Status**: Approved
**Author**: Claude Code (Brainstorming Session)

## Overview and Architecture

### Purpose

Manages SIP extensions and trunks in bin-registrar-manager through a command-line interface. Follows patterns established by agent-control.

### Key Characteristics

- Cobra-based CLI with subcommands for `extension` and `trunk` resources
- Viper configuration management (flags + environment variables)
- Interactive prompts for missing required inputs
- Direct database, cache, and RabbitMQ access (like agent-control)
- Dual database connectivity: Asterisk DB and Bin Manager DB
- Human-readable output by default, optional JSON format

### Architecture Pattern

```
registrar-control (Cobra root)
├── Config loading (Viper + flags/env vars)
├── Handler initialization (extensionhandler + trunkhandler)
│   ├── Database connections (Asterisk + Bin)
│   ├── Redis cache
│   ├── RabbitMQ (sockhandler)
│   └── Request/Notify handlers
└── Subcommands
    ├── extension (create, get, list, update, delete)
    └── trunk (create, get, list, update, delete)
```

### Dependencies

- Uses registrar-manager's config package (database_dsn_bin, database_dsn_asterisk, rabbitmq_address, redis_address, domain_name_extension, domain_name_trunk)
- Uses registrar-manager's extensionhandler and trunkhandler
- Survey library (github.com/AlecAivazis/survey/v2) for interactive prompts

## Extension Commands Structure

### 1. `registrar-control extension create`

- **Required**: `--customer_id` (UUID)
- **Optional**: `--extension_number`, `--username`, `--password`, `--domain`
- Prompts for username and password if not provided
- Generates extension_number if omitted; uses config domain as default
- **Output**: Extension details (text + JSON, or JSON only with `--format json`)

### 2. `registrar-control extension get`

- **Required**: `--id` (extension UUID)
- **Output**: Extension details, Asterisk resources (ps_endpoints, ps_aors, ps_auths), active contacts, registration status, contact addresses, and expiry times

### 3. `registrar-control extension list`

- **Required**: `--customer_id` (UUID)
- **Optional**: `--domain`, `--username`, `--extension_number` (filters); `--limit` (default 100), `--token` (pagination)
- **Output**: Table with ID, extension_number, username, domain, registration status; `--format json` returns array

### 4. `registrar-control extension update`

- **Required**: `--id` (extension UUID); at least one of `--password`, `--username`, `--extension_number`, `--domain`
- Updates Bin Manager and Asterisk PJSIP tables
- **Output**: Updated extension details

### 5. `registrar-control extension delete`

- **Required**: `--id` (extension UUID)
- Shows extension details and prompts for confirmation; `--force` skips prompt
- Removes Asterisk resources (ps_endpoints, ps_aors, ps_auths), Bin Manager records, and invalidates Redis cache
- **Output**: Success message

## Trunk Commands Structure

### 1. `registrar-control trunk create`

- **Required**: `--customer_id` (UUID), `--domain` (must match `^([a-zA-Z]{1})([a-zA-Z0-9\-\.]{1,30})$`)
- **Optional**: `--name`, `--username`, `--password`, `--allowed_ips` (comma-separated); any combination allowed
- **Output**: Trunk details (text + JSON, or JSON only with `--format json`)

### 2. `registrar-control trunk get`

- **Required**: `--id` (trunk UUID)
- **Output**: Trunk details, Asterisk resources (ps_endpoints, ps_aors, ps_auths), active contacts, authentication config, registration status, contact addresses

### 3. `registrar-control trunk list`

- **Required**: `--customer_id` (UUID)
- **Optional**: `--domain`, `--name` (filters); `--limit` (default 100), `--token` (pagination)
- **Output**: Table with ID, name, domain, auth type, registration status; `--format json` returns array

### 4. `registrar-control trunk update`

- **Required**: `--id` (trunk UUID); at least one of `--name`, `--domain`, `--username`, `--password`, `--allowed_ips`
- Updates Bin Manager and Asterisk PJSIP tables
- **Output**: Updated trunk details

### 5. `registrar-control trunk delete`

- **Required**: `--id` (trunk UUID)
- Shows trunk details and prompts for confirmation; `--force` skips prompt
- Removes Asterisk resources (ps_endpoints, ps_aors, ps_auths, sip_auth), Bin Manager records, and invalidates Redis cache
- **Output**: Success message

## Implementation Details

### File Structure

```
bin-registrar-manager/
├── cmd/
│   └── registrar-control/
│       └── main.go          # Root command, initialization, all subcommands
```

### Code Organization Pattern

Single main.go file (like agent-control):

**1. Initialization Functions**:
- `main()` - Entry point, executes root command
- `initCommand()` - Creates Cobra root + subcommands, binds config
- `initHandlers()` - Initializes extension + trunk handlers with DB/cache/RabbitMQ
- `initCache()` - Connects to Redis
- `initExtensionHandler()` - Creates extensionhandler with dependencies
- `initTrunkHandler()` - Creates trunkhandler with dependencies

**2. Helper Functions**:
- `resolveUUID(flagName, label)` - Gets UUID from flag or prompts user
- `resolveString(flagName, label, required)` - Gets string from flag or prompts
- `formatOutput(data, format)` - Outputs human-readable or JSON based on `--format` flag
- `confirmDelete(resourceType, id, details)` - Shows details and prompts for confirmation

**3. Command Constructors** (one per operation):
- Extension: `cmdExtensionCreate()`, `cmdExtensionGet()`, `cmdExtensionList()`, `cmdExtensionUpdate()`, `cmdExtensionDelete()`
- Trunk: `cmdTrunkCreate()`, `cmdTrunkGet()`, `cmdTrunkList()`, `cmdTrunkUpdate()`, `cmdTrunkDelete()`

**4. Run Functions** (execution logic):
- Extension: `runExtensionCreate()`, `runExtensionGet()`, `runExtensionList()`, `runExtensionUpdate()`, `runExtensionDelete()`
- Trunk: `runTrunkCreate()`, `runTrunkGet()`, `runTrunkList()`, `runTrunkUpdate()`, `runTrunkDelete()`

### Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Config management
- `github.com/AlecAivazis/survey/v2` - Interactive prompts
- Registrar-manager packages: `internal/config`, `pkg/extensionhandler`, `pkg/trunkhandler`, `pkg/dbhandler`, `pkg/cachehandler`

## Configuration and Error Handling

### Configuration Requirements

**Required Config Fields**:
- `database_dsn_bin` / `DATABASE_DSN_BIN` - Bin manager database
- `database_dsn_asterisk` / `DATABASE_DSN_ASTERISK` - Asterisk database
- `rabbitmq_address` / `RABBITMQ_ADDRESS` - RabbitMQ connection
- `redis_address` / `REDIS_ADDRESS` - Redis cache
- `domain_name_extension` / `DOMAIN_NAME_EXTENSION` - Base domain for extensions
- `domain_name_trunk` / `DOMAIN_NAME_TRUNK` - Base domain for trunks

**Optional Config Fields**:
- `redis_password` / `REDIS_PASSWORD`
- `redis_database` / `REDIS_DATABASE` (default: 1)

### Config Loading

PersistentPreRunE hook binds flags to Viper, loads config via `config.LoadGlobalConfig()`, validates required fields, and initializes domain names via `common.SetBaseDomainNames()`.

### Error Handling Patterns

**1. Input Validation Errors**:
- UUID format validation with helpful messages: `"invalid format for Agent ID: 'abc' is not a valid UUID"`
- Domain validation for trunks: `"domain must match pattern: ^([a-zA-Z]{1})([a-zA-Z0-9\-\.]{1,30})$"`
- Required field validation via survey validators

**2. Handler Errors**:
- Wrap all errors with context: `errors.Wrap(err, "failed to create extension")`
- Handle common cases:
  - `ErrNotFound` → "Extension not found with ID: <uuid>"
  - Duplicates → "Extension with number X already exists for customer"
  - Connection errors → "Failed to connect to database, check configuration"

**3. Confirmation Cancellation**:
- Delete operations: `"Deletion canceled"` message when user declines
- Survey interrupts (Ctrl+C): `"Input canceled"` with graceful exit

**4. Output on Errors**:
- Return error from RunE functions (Cobra handles display)
- Use logrus for success messages: `logrus.WithField("res", res).Infof("Created extension")`
- Structured logging with relevant fields (customer_id, extension_id, etc.)

## Testing and Usage Examples

### Testing Strategy

**Unit Tests** (same pattern as agent-control handlers):
- Mock extensionhandler and trunkhandler interfaces
- Test command initialization and flag parsing
- Test helper functions (resolveUUID, resolveString, formatOutput)
- Test error handling paths
- Use gomock for mocking

**Integration Tests** (manual):
- Test against development environment with real databases
- Verify Asterisk PJSIP tables are correctly updated
- Verify Redis cache invalidation works
- Test interactive prompts work correctly
- Test both flag-based and interactive workflows

### Usage Examples

```bash
# Extension management
registrar-control extension create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --username "user1001" \
  --extension_number "1001"
  # Will prompt for password interactively

registrar-control extension get --id "uuid-here"

registrar-control extension list \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --limit 50 \
  --format json

registrar-control extension update \
  --id "uuid-here" \
  --password "newpass123"

registrar-control extension delete --id "uuid-here" --force

# Trunk management
registrar-control trunk create \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --domain "voip-carrier.example.com" \
  --name "Primary Carrier" \
  --username "trunk_user" \
  --password "trunk_pass" \
  --allowed_ips "203.0.113.1,203.0.113.2"

registrar-control trunk get --id "uuid-here"

registrar-control trunk list \
  --customer_id "5e4a0680-804e-11ec-8477-2fea5968d85b" \
  --domain "voip-carrier.example.com"

registrar-control trunk update \
  --id "uuid-here" \
  --allowed_ips "203.0.113.1,203.0.113.2,203.0.113.3"

registrar-control trunk delete --id "uuid-here"
  # Will prompt for confirmation
```

### Build and Deployment

```bash
# Build binary
cd bin-registrar-manager
go build -o bin/registrar-control ./cmd/registrar-control

# Install for system-wide use
sudo cp bin/registrar-control /usr/local/bin/

# Run with config file or environment variables
export DATABASE_DSN_BIN="user:pass@tcp(localhost:3306)/voipbin"
export DATABASE_DSN_ASTERISK="user:pass@tcp(localhost:3306)/asterisk"
registrar-control extension list --customer_id "uuid-here"
```

## Implementation Checklist

- [ ] Create `cmd/registrar-control/main.go` with Cobra root command
- [ ] Implement initialization functions (handlers, cache, config)
- [ ] Implement helper functions (resolveUUID, resolveString, formatOutput, confirmDelete)
- [ ] Implement extension commands (create, get, list, update, delete)
- [ ] Implement trunk commands (create, get, list, update, delete)
- [ ] Add extension output formatting (human-readable + JSON)
- [ ] Add trunk output formatting (human-readable + JSON)
- [ ] Add interactive prompts for required fields
- [ ] Add confirmation prompts for delete operations
- [ ] Test all commands against development environment
- [ ] Update bin-registrar-manager/CLAUDE.md with registrar-control documentation
- [ ] Add build configuration to Dockerfile/Makefile if needed
