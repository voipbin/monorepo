# bin-contact-manager — Dependencies

## Runtime Dependencies

### Infrastructure

| Component | Purpose |
|-----------|---------|
| MySQL | Persistent storage for contacts, phone numbers, emails, tag assignments |
| Redis | Lookup cache indexed by phone (E.164) and email; invalidated on mutation |
| RabbitMQ | RPC transport for incoming requests and outgoing events |

### Upstream Services (RabbitMQ RPC clients)

The following services send RPC requests to this service's request queue:

- `bin-api-manager` — exposes the Contact API to external callers
- `bin-campaign-manager` — retrieves contacts for campaign operations
- `bin-call-manager` — performs caller-ID lookup during inbound calls
- Any service performing contact lookup by phone or email

### Downstream Services (called via RabbitMQ RPC)

This service calls the following services:

| Service | Purpose |
|---------|---------|
| `bin-tag-manager` | Tag definition lookup when assigning tags to contacts |

### Event Subscriptions (consumed)

| Queue | Event | Action |
|-------|-------|--------|
| `bin-manager.customer-manager.event` | `customer_deleted` | Cascade-delete all contacts for the removed customer |

### Events Published

| Event | Trigger |
|-------|---------|
| `contact_created` | New contact record created |
| `contact_updated` | Contact fields or child records modified |
| `contact_deleted` | Contact soft-deleted |

## Go Module Dependencies

Direct monorepo dependencies (via `replace` directives in `go.mod`):

| Module | Usage |
|--------|-------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler |
| `monorepo/bin-customer-manager` | Customer event model structs (for subscription deserialization) |
| `monorepo/bin-tag-manager` | Tag model structs (for tag assignment validation) |

Notable third-party dependencies:

| Library | Purpose |
|---------|---------|
| `github.com/gofrs/uuid` | UUID generation and parsing |
| `github.com/Masterminds/squirrel` | SQL query builder |
| `go.uber.org/mock` | Interface mock generation for tests |
| `github.com/prometheus/client_golang` | Metrics exposure |
| `github.com/spf13/viper` + `pflag` | Configuration management |
