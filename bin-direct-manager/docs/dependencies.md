# bin-direct-manager — Dependencies

## Runtime Dependencies

### Infrastructure

| Component | Purpose |
|-----------|---------|
| MySQL | Persistent storage for direct hash records |
| Redis | Hash-to-record index for O(1) lookup by hash value |
| RabbitMQ | RPC transport for incoming requests |

### Upstream Services (RabbitMQ RPC clients)

The following services send RPC requests to this service:

- `bin-api-manager` — exposes the Direct API to external callers
- `voip-kamailio` (SIP proxy) — resolves incoming direct-dialed URIs via the hash lookup endpoint

### Downstream Services (called via RabbitMQ RPC)

This service does not make outbound RPC calls to other services.

### Event Subscriptions (consumed)

| Queue | Event | Action |
|-------|-------|--------|
| `bin-manager.customer-manager.event` | `customer_deleted` | Remove all direct records for the deleted customer |

### Events Published

None. This service publishes no events.

## Go Module Dependencies

Direct monorepo dependencies (via `replace` directives in `go.mod`):

| Module | Usage |
|--------|-------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler |
| `monorepo/bin-customer-manager` | Customer event model structs (for subscription deserialization) |

Notable third-party dependencies:

| Library | Purpose |
|---------|---------|
| `github.com/gofrs/uuid` | UUID generation and parsing |
| `github.com/Masterminds/squirrel` | SQL query builder |
| `go.uber.org/mock` | Interface mock generation for tests |
| `github.com/prometheus/client_golang` | Metrics exposure |
| `github.com/spf13/cobra` + `viper` | Configuration management |
