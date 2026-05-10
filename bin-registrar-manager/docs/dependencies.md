# bin-registrar-manager — Dependencies

## Runtime Dependencies

### Infrastructure

| Component | Purpose |
|-----------|---------|
| MySQL (bin-manager DB) | Extensions, trunks, sip_auth tables |
| MySQL (Asterisk DB) | `ps_endpoints`, `ps_aors`, `ps_auths`, `ps_contacts` tables |
| Redis | Active SIP contact cache; invalidated on extension/trunk delete |
| RabbitMQ | RPC transport for incoming requests |

This service is the only service in the monorepo that connects to **two separate MySQL databases**.

### Upstream Services (RabbitMQ RPC clients)

The following services send RPC requests to this service:

- `bin-api-manager` — exposes the Extension and Trunk APIs to external callers
- `bin-call-manager` — resolves extension registrations for call routing
- `voip-kamailio` (SIP proxy) — queries contact registrations for SIP routing decisions

### Downstream Services (called via RabbitMQ RPC)

This service does not make outbound RPC calls to other services.

### Event Subscriptions (consumed)

| Queue | Event | Action |
|-------|-------|--------|
| `bin-manager.customer-manager.event` | `customer_deleted` | Delete all extensions and trunks (and their Asterisk resources) for the deleted customer |

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
| `github.com/sirupsen/logrus` | Structured logging (Joonix Stackdriver formatter) |
