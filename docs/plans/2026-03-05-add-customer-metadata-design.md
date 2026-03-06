# Add Customer Metadata Support

## Problem Statement

We need a mechanism to store internal-use configuration flags on a per-customer basis. The first use case is `rtp_debug` — when enabled, RTPEngine captures RTP traffic as PCAP files for debugging audio issues. These options are managed exclusively by VoIPbin admins (ProjectSuperAdmin) and are not visible to regular customers.

## Design

### Data Model

Add a `Metadata` typed struct stored as a JSON column on the `customer_customers` table.

```go
// bin-customer-manager/models/customer/metadata.go
type Metadata struct {
    RTPDebug bool `json:"rtp_debug"` // enable RTPEngine RTP capture (PCAP)
}
```

```go
// on Customer struct in customer.go
Metadata Metadata `json:"metadata" db:"metadata,json"`
```

**Why a typed struct (not `map[string]any`):**
- Compile-time type safety for all consumers
- Self-documenting which options exist
- Adding new options = adding a struct field (no DB migration needed, JSON column)

**Field constant:**
```go
FieldMetadata Field = "metadata"
```

### Database Migration

Alembic migration in `bin-dbscheme-manager`:
```sql
ALTER TABLE customer_customers ADD COLUMN metadata JSON DEFAULT NULL;
```

`NULL` means "no metadata set" (default for all existing customers). The Go `Metadata` struct zero value (`{rtp_debug: false}`) is safe for consumers when the column is NULL.

### API Access

**Read path — no extra work needed:**
- SuperAdmin endpoints (`CustomerGet`, `CustomerList`) already return the raw `*customer.Customer` struct, not `WebhookMessage`. Adding `Metadata` to the struct means SuperAdmin automatically sees it.
- Regular user endpoints (`CustomerSelfGet`) return `*customer.WebhookMessage` which deliberately excludes `Metadata`.

**Write path — new dedicated endpoint:**

| Endpoint | Method | Permission | Purpose |
|---|---|---|---|
| `PUT /v1/customers/{id}/metadata` | PUT | ProjectSuperAdmin | Update customer metadata |

This follows the existing pattern of `PUT /v1/customers/{id}/billing_account_id` — a dedicated endpoint for a specific sub-resource that requires SuperAdmin access.

### What's NOT Exposed

- `Metadata` is NOT added to `WebhookMessage` — regular API consumers never see it
- `Metadata` is NOT added to the public OpenAPI schema for `CustomerManagerCustomer`
- Regular users (`CustomerAdmin`, `CustomerManager`, `CustomerAgent`) cannot read or write metadata
- The self-update endpoint (`PUT /v1.0/customer`) does not accept metadata

### Files to Change

| Layer | File | Change |
|---|---|---|
| DB migration | `bin-dbscheme-manager/alembic/versions/` | New migration: `ADD COLUMN metadata JSON DEFAULT NULL` |
| Metadata type | `bin-customer-manager/models/customer/metadata.go` | New file: `Metadata` struct |
| Model struct | `bin-customer-manager/models/customer/customer.go` | Add `Metadata Metadata` field with `db:"metadata,json"` |
| Field constants | `bin-customer-manager/models/customer/field.go` | Add `FieldMetadata Field = "metadata"` |
| WebhookMessage | No change — metadata deliberately excluded |
| Request struct | `bin-customer-manager/pkg/listenhandler/models/request/customers.go` | Add `V1DataCustomersIDMetadataPut` struct |
| Listen handler | `bin-customer-manager/pkg/listenhandler/v1_customers.go` | Add route + handler for `PUT /v1/customers/{id}/metadata` |
| Business logic | `bin-customer-manager/pkg/customerhandler/db.go` | Add `UpdateMetadata()` method |
| Handler interface | `bin-customer-manager/pkg/customerhandler/main.go` | Add `UpdateMetadata` to interface |
| DB handler | `bin-customer-manager/pkg/dbhandler/customer.go` | `CustomerUpdate` already handles JSON fields via `PrepareFields` — no change needed |
| RPC request handler | `bin-common-handler/pkg/requesthandler/customer_customer.go` | Add `CustomerV1CustomerUpdateMetadata()` |
| API service handler | `bin-api-manager/pkg/servicehandler/customer.go` | Add `CustomerUpdateMetadata()` with SuperAdmin permission check |
| HTTP server | `bin-api-manager/server/customer.go` | Add handler for new endpoint |
| OpenAPI schema | `bin-openapi-manager/openapi/openapi.yaml` | Add metadata update endpoint + `CustomerManagerMetadata` schema (admin section only) |

### Deferred (Out of Scope)

- **Kamailio integration**: How calls read `rtp_debug` and pass `record-call` to RTPEngine. This will be designed separately once the metadata infrastructure is in place.
- **Additional metadata fields**: Only `rtp_debug` for now. New fields can be added to the `Metadata` struct without DB migrations.

## Trade-offs

1. **JSON column vs individual columns**: JSON is more flexible (no migration per option) but loses DB-level type checking. Given these are internal flags with typed Go struct, the trade-off is acceptable.
2. **Dedicated endpoint vs field in existing update**: Dedicated endpoint is cleaner separation — regular update path stays unchanged, admin concerns are isolated.
3. **Metadata excluded from WebhookMessage**: Keeps internal flags invisible to customers. SuperAdmin already gets the full internal struct, so no extra mechanism needed for admin reads.
