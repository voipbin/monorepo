# bin-registrar-manager — Domain

## Domain Entities

### Extension

A SIP endpoint for an individual user within a customer account. Creating an extension provisions three Asterisk PJSIP records atomically.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key (bin-manager DB) |
| `customer_id` | UUID | Owning customer |
| `username` | string | SIP username |
| `password` | string | SIP auth password (hashed) |
| `extension_number` | string | Dialable extension within the customer realm |
| `domain_name` | string | SIP domain (customer-specific subdomain of `domain_name_extension`) |
| `tm_create` | timestamp | |
| `tm_update` | timestamp | |
| `tm_delete` | timestamp | Soft-delete sentinel; active = `9999-01-01` |

**Asterisk tables provisioned per extension:**

| Table | Purpose |
|-------|---------|
| `ps_endpoints` | Endpoint config (codec, context, auth reference) |
| `ps_aors` | Address of Record — maps SIP URI to registration |
| `ps_auths` | Authentication credentials |

The three Asterisk records share the same identifier derived from the extension UUID.

### Trunk

A SIP trunk for carrier or provider connections. Supports basic authentication (username/password), IP-based authentication, or both simultaneously.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `name` | string | Human-readable label |
| `domain_name` | string | Carrier SIP domain (validated: `^([a-zA-Z]{1})([a-zA-Z0-9\-\.]{1,30})$`) |
| `username` | string | SIP username (basic auth; optional) |
| `password` | string | SIP password (basic auth; optional) |
| `allowed_ips` | []string | IP allowlist (IP-based auth; optional) |
| `tm_create` | timestamp | |
| `tm_update` | timestamp | |
| `tm_delete` | timestamp | Soft-delete sentinel |

### Contact (SIP Registration)

A read-only view of an active SIP registration. Sourced from Asterisk `ps_contacts` and cached in Redis.

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Asterisk contact key |
| `uri` | string | Registered contact URI (SIP URI with IP/port/transport) |
| `expiration_time` | timestamp | When the registration expires |
| `endpoint_id` | string | References the `ps_endpoints` entry |
| `via_addr` | string | Client IP address (NAT traversal) |
| `via_port` | int | Client port |

## Key Business Rules

1. **Two-database atomicity**: Creating or deleting an extension touches the bin-manager DB and three Asterisk tables (`ps_endpoints`, `ps_aors`, `ps_auths`). All operations must clean up all records on failure — partial states cause Asterisk to malfunction.

2. **Domain name assignment**: Extensions use a subdomain of `domain_name_extension` config value. Trunks use a custom domain validated by regex. Domain names are set globally at startup via `common.SetBaseDomainNames()`.

3. **Trunk domain uniqueness**: Each trunk's `domain_name` must be unique within the platform (used as an Asterisk endpoint identifier).

4. **Trunk auth modes**: A trunk may use basic auth, IP auth, or both. At least one auth mode must be configured.

5. **Contact caching**: Active SIP registrations from `ps_contacts` are cached in Redis. Cache is invalidated when the corresponding extension or trunk is deleted.

6. **Cascading delete**: When a `customer_deleted` event is received, all extensions and trunks for that customer are deleted along with their Asterisk resources.

7. **Soft deletes on bin-manager tables**: The `extensions` and `trunks` tables use `tm_delete` sentinel. Asterisk tables do not use soft deletes — Asterisk records are hard-deleted.

8. **Extension number uniqueness**: Extension numbers must be unique within the customer's domain (enforced at DB level).
