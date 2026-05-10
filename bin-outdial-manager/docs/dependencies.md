# bin-outdial-manager — Dependencies

## Runtime Dependencies

### Infrastructure

| Component | Purpose |
|-----------|---------|
| MySQL | Persistent storage for outdials, targets, and call records |
| Redis | Target state cache |
| RabbitMQ | RPC transport for incoming requests; event publishing |

### Upstream Services (RabbitMQ RPC clients)

The following services send RPC requests to this service:

- `bin-campaign-manager` — primary consumer; fetches available targets, marks targets in-progress, updates target status
- `bin-api-manager` — exposes the Outdial API to external callers

### Downstream Services (called via RabbitMQ RPC)

This service does not make RPC calls to other services.

### Events Published

| Queue / Exchange | Event | Trigger |
|-----------------|-------|---------|
| `bin-manager.outdial-manager.event` | `outdial_created` | New outdial created |
| `bin-manager.outdial-manager.event` | `outdial_updated` | Outdial fields changed |
| `bin-manager.outdial-manager.event` | `outdial_deleted` | Outdial deleted |

### Events Subscribed

None. This service has no subscribe handler.

## Go Module Dependencies

Direct monorepo dependencies (via `replace` directives in `go.mod`):

| Module | Usage |
|--------|-------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler |
| `monorepo/bin-campaign-manager` | Campaign event model structs (event types) |

Notable third-party dependencies:

| Library | Purpose |
|---------|---------|
| `github.com/gofrs/uuid` | UUID generation and parsing |
| `github.com/Masterminds/squirrel` | SQL query builder |
| `go.uber.org/mock` | Interface mock generation for tests |
| `github.com/smotes/purse` | SQL script loader for test DB setup |
| `github.com/prometheus/client_golang` | Metrics exposure |
| `github.com/spf13/viper` + `pflag` | Configuration management |
