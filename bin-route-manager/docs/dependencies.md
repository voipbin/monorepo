# bin-route-manager — Dependencies

## Runtime Dependencies

### Infrastructure

| Component | Purpose |
|-----------|---------|
| MySQL | Persistent storage for providers and routes |
| Redis | Cache for dialroute lookups (reduces DB round-trips on call setup) |
| RabbitMQ | RPC transport for incoming requests |

### Upstream Services (RabbitMQ RPC clients)

The following services send RPC requests to this service:

- `bin-api-manager` — exposes Provider and Route APIs to external callers
- `bin-call-manager` — queries `GET /v1/dialroutes` during outbound call setup to select the carrier
- `bin-outboundcall-manager` — queries dialroutes for outbound PSTN routing decisions

### Downstream Services (called via RabbitMQ RPC)

This service does not make outbound RPC calls to other services.

### Event Subscriptions (consumed)

None. This service has no subscribe handler.

### Events Published

None. This service publishes no events.

## Go Module Dependencies

Direct monorepo dependencies (via `replace` directives in `go.mod`):

| Module | Usage |
|--------|-------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler |

Notable third-party dependencies:

| Library | Purpose |
|---------|---------|
| `github.com/gofrs/uuid` | UUID generation and parsing |
| `github.com/Masterminds/squirrel` | SQL query builder |
| `go.uber.org/mock` | Interface mock generation for tests |
| `github.com/prometheus/client_golang` | Metrics exposure |
| `github.com/spf13/viper` + `pflag` | Configuration management |
