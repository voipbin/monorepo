# bin-direct-manager — Domain

## Domain Entities

### Direct

A direct record maps a unique hash string to a customer-owned resource, enabling direct SIP URI dialing.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `resource_type` | enum | Target resource type (see below) |
| `resource_id` | UUID | ID of the target resource |
| `hash` | string | Unique hash for SIP URI construction |
| `tm_create` | timestamp | Creation time |
| `tm_update` | timestamp | Last update time |
| `tm_delete` | timestamp | Soft-delete sentinel; active = `9999-01-01` |

**Resource types** (`resource_type`):

| Type | Target service |
|------|---------------|
| `extension` | SIP extension in `bin-registrar-manager` |
| `conference` | Conference room in `bin-conference-manager` |
| `ai` | AI agent in `bin-ai-manager` |
| `ai_team` | AI team (grouped agents) |
| `agent` | Human agent in `bin-agent-manager` |

## Key Business Rules

1. **Hash uniqueness**: Each direct hash is unique across the platform. The hash is used to construct SIP URIs such as `<hash>@<domain>`. Collisions are handled by retrying hash generation.

2. **Hash regeneration**: A direct hash can be replaced without changing the record ID or resource binding. `POST /v1/directs/{id}/regenerate` atomically replaces the hash and invalidates the old cache entry.

3. **Tenant isolation**: All queries filter by `customer_id`. A direct record is never visible to another customer.

4. **Soft deletes**: Delete operations set `tm_delete` to the current timestamp. All active-record queries add `tm_delete = '9999-01-01 00:00:00.000000'` to the filter.

5. **Lookup by hash**: `GET /v1/directs/by-hash/<hash>` is the hot path used by the SIP proxy (Kamailio) to resolve an incoming call destination. This lookup uses the Redis cache for sub-millisecond response.

6. **Cascading delete**: When a `customer_deleted` event is received from `bin-customer-manager`, all direct records for that customer are removed.

7. **No event publishing**: This service does not publish events. Changes are visible only through RPC responses.

8. **Resource validation**: The service stores the resource binding but does not validate that the referenced resource exists. Consumers must handle cases where the target resource has been deleted.
