# Development Guide

> **Quick Reference:** For command summary, see [CLAUDE.md](../CLAUDE.md)

## Prerequisites

Some services require system libraries:
```bash
# ZMQ libraries (required by bin-api-manager and other services)
apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
```

## Building Services

### Build Pattern

Each service follows the same build pattern:

```bash
# Navigate to a service directory
cd bin-<service-name>

# Configure git for private modules (if needed)
git config --global url."https://${GL_DEPLOY_USER}:${GL_DEPLOY_TOKEN}@gitlab.com".insteadOf "https://gitlab.com"
export GOPRIVATE="gitlab.com/voipbin"

# Download dependencies
go mod download

# Vendor dependencies (optional but recommended for reproducible builds)
go mod vendor

# Build the service
go build ./cmd/...

# Run the service with default configuration
./<service-name>

# Show available configuration flags
./<service-name> --help
```

## Testing

### Running Tests

```bash
# From monorepo root: Run tests for all services
find . -name "go.mod" -execdir go test ./... \;

# Test a specific service
cd bin-<service-name>
go test -v ./...

# Run tests with coverage
go test -coverprofile cp.out -v ./...
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Test a specific package
go test -v ./pkg/servicehandler/...
```

## Testing with SQLite Test Databases

**Pattern used across ALL services in the monorepo:**

Each service has `scripts/database_scripts/` directory containing simplified SQL files for unit testing with SQLite.

### Purpose

- **Isolated unit tests** - Test database operations without requiring MySQL connection
- **Fast test execution** - SQLite is lightweight and runs in-memory
- **CI/CD friendly** - No external database dependencies needed
- **Good enough for CRUD** - SQLite doesn't support all MySQL features, but sufficient for basic CRUD operation tests

### Pattern

```
bin-<service-name>/
└── scripts/
    └── database_scripts/
        ├── table_resource1.sql
        ├── table_resource2.sql
        └── table_resource3.sql
```

### SQL File Format

- Simplified CREATE TABLE statements compatible with SQLite
- Copied and adapted from real schema in `bin-dbscheme-manager` Alembic migrations
- MySQL-specific features removed or simplified (e.g., `BINARY(16)` → `BLOB`, `DATETIME(6)` → `TEXT`)
- Primary keys and basic indexes included
- Foreign keys simplified (SQLite has limited FK support in older versions)

**Example (`table_chats.sql`):**
```sql
CREATE TABLE chat_chats (
    id          BLOB PRIMARY KEY,
    customer_id BLOB,
    type        TEXT,
    tm_create   TEXT,
    tm_update   TEXT,
    tm_delete   TEXT
);

CREATE INDEX idx_chat_chats_customer_id ON chat_chats(customer_id);
```

### Usage in Tests

```go
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)

    // Load schema from scripts/database_scripts/
    schema, _ := os.ReadFile("../../scripts/database_scripts/table_chats.sql")
    _, err = db.Exec(string(schema))
    require.NoError(t, err)

    return db
}

func TestChatCreate(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    handler := dbhandler.New(db, nil)
    // ... test CRUD operations
}
```

### Important Notes

- These SQL files are NOT the source of truth (that's `bin-dbscheme-manager` Alembic migrations)
- These are test fixtures, manually created/updated as needed
- Not all MySQL features work in SQLite (stored procedures, full-text search, etc.)
- For integration tests requiring MySQL-specific features, use actual MySQL test database

### When to Update

- After creating new Alembic migration in `bin-dbscheme-manager`
- When adding new tables or significant schema changes
- When database handler tests fail due to schema mismatch

## Code Generation

Many services use code generation for mocks and API specs:

```bash
# Generate mocks for a specific service
cd bin-<service-name>
go generate ./...

# Generate OpenAPI models (from bin-openapi-manager)
cd bin-openapi-manager
go generate ./...

# Update all services in monorepo (run from root)
ls -d */ | xargs -I {} bash -c "cd '{}' && go get -u ./... && go mod vendor && go generate ./... && go test ./..."
```

## Linting

```bash
# Run go vet
go vet ./...

# Run golangci-lint (recommended)
golangci-lint run -v --timeout 5m
```
