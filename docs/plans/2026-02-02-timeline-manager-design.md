# timeline-manager Design

## Overview

**timeline-manager** is a read-only OLAP query service that provides a unified interface for querying historical events stored in ClickHouse.

### Responsibilities

- Accept event queries via RabbitMQ RPC from internal services
- Translate query parameters into efficient ClickHouse SQL
- Return paginated event results

### What it does NOT do

- Write events (ingestion handled by notifyhandler)
- Enforce customer_id isolation (caller's responsibility)
- Aggregate or transform data (pass-through from ClickHouse)

### Primary Use Cases

1. **Debugging/Support** - "What happened with activeflow X?" → Get full event history
2. **Internal Service Queries** - Other managers querying event history for their entities
3. **Activity Feeds** - Building timelines of call/flow activity

### Communication

- RabbitMQ RPC: `bin-manager.timeline-manager.request` queue
- Follows existing monorepo REST-like RPC pattern

### Data Source

- ClickHouse `default.events` table (read-only)
- Schema includes: `timestamp`, `event_type`, `publisher`, `data_type`, `data`, `resource_id` (materialized)

## Project Structure

```
bin-timeline-manager/
├── cmd/
│   ├── timeline-manager/      # Main service entry point
│   │   └── main.go
│   └── timeline-control/      # CLI tool for queries + migrations
│       └── main.go
├── internal/
│   └── config/
│       └── config.go
├── migrations/                 # ClickHouse schema migrations (golang-migrate)
│   ├── 000001_create_events_table.up.sql
│   ├── 000001_create_events_table.down.sql
│   ├── 000002_add_resource_id_column.up.sql
│   └── 000002_add_resource_id_column.down.sql
├── models/
│   └── event/
│       ├── event.go
│       └── request.go
├── pkg/
│   ├── listenhandler/         # RabbitMQ RPC handler
│   │   ├── main.go
│   │   └── v1_events.go
│   ├── eventhandler/          # Business logic
│   │   ├── main.go
│   │   └── event.go
│   └── dbhandler/             # ClickHouse queries
│       ├── main.go
│       └── event.go
├── Dockerfile
├── go.mod
└── README.md
```

### Key Differences from Other Managers

- Uses ClickHouse instead of MySQL
- Read-only (no Create/Update/Delete operations)
- No cache layer needed (ClickHouse handles it)
- No subscribehandler (doesn't consume events)
- Owns ClickHouse schema via golang-migrate

## API Design

### RPC Endpoint

```
Queue:  bin-manager.timeline-manager.request
URI:    /v1/events
Method: POST
```

### Request Model

```go
import (
    "github.com/gofrs/uuid"
    commonoutline "monorepo/bin-common-handler/models/outline"
)

type EventListRequest struct {
    Publisher commonoutline.ServiceName `json:"publisher"`
    ID        uuid.UUID                 `json:"id"`
    Events    []string                  `json:"events"`

    // Pagination
    PageToken string `json:"page_token,omitempty"`
    PageSize  int    `json:"page_size,omitempty"`
}
```

### Response Model

```go
type EventListResponse struct {
    Result        []*Event `json:"result"`
    NextPageToken string   `json:"next_page_token,omitempty"`
}

type Event struct {
    Timestamp  string                    `json:"timestamp"`
    EventType  string                    `json:"event_type"`
    Publisher  commonoutline.ServiceName `json:"publisher"`
    DataType   string                    `json:"data_type"`
    Data       json.RawMessage           `json:"data"`
}
```

### Event Pattern Matching

- `activeflow_created` → exact match
- `activeflow_*` → prefix match (translates to `LIKE 'activeflow_%'`)
- `*` → all events (no filter)

### Example Request/Response

```json
// Request
{
  "publisher": "flow-manager",
  "id": "9225c7a8-0017-11f1-bff8-abe1dfdf5c9d",
  "events": ["activeflow_*"],
  "page_size": 50
}

// Response
{
  "result": [
    {
      "timestamp": "2024-01-15T10:30:00.123Z",
      "event_type": "activeflow_created",
      "publisher": "flow-manager",
      "data_type": "application/json",
      "data": {"id": "9225c7a8-...", "customer_id": "...", "status": "..."}
    }
  ],
  "next_page_token": "2024-01-15T10:29:00.000Z"
}
```

## Database Layer

### ClickHouse Schema

```sql
CREATE TABLE IF NOT EXISTS events (
    timestamp DateTime64(3),
    event_type LowCardinality(String),
    publisher LowCardinality(String),
    data_type LowCardinality(String),
    data String,
    resource_id String MATERIALIZED JSONExtractString(data, 'id')
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (event_type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 1 YEAR;
```

### Materialized Column

The `resource_id` column is automatically populated from `JSONExtractString(data, 'id')` at insert time:
- Enables fast filtering by resource ID without JSON parsing at query time
- No changes to ingestion pipeline required
- Existing rows need backfill: `ALTER TABLE events UPDATE resource_id = resource_id WHERE 1`

### Query Generation

```go
func (h *dbHandler) EventList(
    ctx context.Context,
    publisher commonoutline.ServiceName,
    resourceID uuid.UUID,
    events []string,
    pageToken string,
    pageSize int,
) ([]*event.Event, error) {
    query := `
        SELECT timestamp, event_type, publisher, data_type, data
        FROM events
        WHERE publisher = ?
          AND resource_id = ?
    `
    args := []interface{}{string(publisher), resourceID.String()}

    // Add event type filters
    if len(events) > 0 {
        eventConditions := buildEventConditions(events)
        query += " AND (" + eventConditions + ")"
    }

    // Pagination by timestamp
    if pageToken != "" {
        query += " AND timestamp < ?"
        args = append(args, pageToken)
    }

    query += " ORDER BY timestamp DESC LIMIT ?"
    args = append(args, pageSize)

    // Execute and scan...
}

// buildEventConditions converts ["activeflow_*", "flow_created"] to SQL
// "event_type LIKE 'activeflow_%' OR event_type = 'flow_created'"
func buildEventConditions(events []string) string {
    // Handle wildcards: * -> LIKE '%', activeflow_* -> LIKE 'activeflow_%'
}
```

## Migrations

Schema ownership moves from `monorepo-etc/infra/clickhouse` to `bin-timeline-manager/migrations/`.

### Migration Tool

Using `golang-migrate` with ClickHouse driver.

### Migration Files

**000001_create_events_table.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS events (
    timestamp DateTime64(3),
    event_type LowCardinality(String),
    publisher LowCardinality(String),
    data_type LowCardinality(String),
    data String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (event_type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 1 YEAR;
```

**000001_create_events_table.down.sql:**
```sql
DROP TABLE IF EXISTS events;
```

**000002_add_resource_id_column.up.sql:**
```sql
ALTER TABLE events
ADD COLUMN resource_id String MATERIALIZED JSONExtractString(data, 'id');
```

**000002_add_resource_id_column.down.sql:**
```sql
ALTER TABLE events
DROP COLUMN resource_id;
```

### CLI Commands

```bash
# Run all pending migrations
./timeline-control migrate up

# Rollback last migration
./timeline-control migrate down

# Show migration status
./timeline-control migrate status

# Query events
./timeline-control event list --publisher flow-manager --id <uuid> --events "activeflow_*"
```

## Configuration

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CLICKHOUSE_ADDRESS` | ClickHouse server address | `clickhouse:9000` |
| `CLICKHOUSE_DATABASE` | Database name | `default` |
| `RABBITMQ_ADDRESS` | RabbitMQ connection string | `amqp://guest:guest@rabbitmq:5672` |
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics server address | `:8080` |

### Config Struct

```go
type Config struct {
    // ClickHouse
    ClickHouseAddress  string `mapstructure:"clickhouse_address"`
    ClickHouseDatabase string `mapstructure:"clickhouse_database"`

    // RabbitMQ
    RabbitMQAddress string `mapstructure:"rabbitmq_address"`

    // Prometheus
    PrometheusEndpoint      string `mapstructure:"prometheus_endpoint"`
    PrometheusListenAddress string `mapstructure:"prometheus_listen_address"`
}
```

## Dependencies

```
github.com/ClickHouse/clickhouse-go/v2
github.com/golang-migrate/migrate/v4
github.com/spf13/cobra
github.com/spf13/viper
github.com/streadway/amqp
github.com/gofrs/uuid
github.com/sirupsen/logrus
github.com/prometheus/client_golang

// Local monorepo modules
monorepo/bin-common-handler
```

## Summary

| Aspect | Decision |
|--------|----------|
| Purpose | Read-only query facade over ClickHouse |
| Communication | RabbitMQ RPC only |
| Architecture | Same as flow-manager pattern |
| API | Single `/v1/events` endpoint with flexible filtering |
| Pagination | Timestamp-based cursor |
| Schema management | golang-migrate with up/down migrations |
| Schema change | Materialized column `resource_id` from JSON |
| Response format | `json.RawMessage` for clean nested JSON |

## Out of Scope

- ClickHouse ingestion (handled by notifyhandler)
- Customer ID filtering (caller's responsibility)
- Event aggregation/analytics
