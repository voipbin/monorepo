# bin-route-manager — Domain

## Domain Entities

### Provider

A SIP gateway or PSTN carrier used for outbound call routing.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `name` | string | Human-readable label |
| `detail` | string | Description |
| `type` | enum | Currently only `sip` |
| `hostname` | string | SIP proxy/gateway hostname or IP |
| `tech_prefix` | string | Dial string prefix prepended to the dialed number |
| `tech_postfix` | string | Dial string suffix appended to the dialed number |
| `tech_headers` | []string | Custom SIP headers injected for calls via this provider |
| `tm_create` | timestamp | |
| `tm_update` | timestamp | |
| `tm_delete` | timestamp | Soft-delete sentinel; active = `9999-01-01` |

Table: `route_manager_providers`

### Route

Maps a customer destination to a provider with priority ordering.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer (or `CustomerIDBasicRoute` for system defaults) |
| `name` | string | Human-readable label |
| `detail` | string | Description |
| `provider_id` | UUID | Target provider |
| `target` | string | Country code (e.g., `1`, `44`) or `all` for catch-all |
| `priority` | int | Lower number = higher priority |
| `tm_create` | timestamp | |
| `tm_update` | timestamp | |
| `tm_delete` | timestamp | Soft-delete sentinel |

Table: `route_manager_routes`

**System default routes** use a reserved customer ID: `CustomerIDBasicRoute = "00000000-0000-0000-0000-000000000001"`. These serve as the global fallback when a customer has no route for a given destination.

### Dialroute

Not a stored entity — a computed result. The `GET /v1/dialroutes?customer_id=<uuid>&target=<target>` endpoint returns the **effective merged route list** for a (customer, destination) pair.

## Key Business Rules

1. **Dialroute merge (fallback chain)**:
   - `routehandler.DialrouteGets` first retrieves customer-specific routes for the target destination.
   - It then retrieves default routes (`CustomerIDBasicRoute`) for the same target.
   - Results are merged: customer routes take precedence; default routes are appended for providers not already represented in customer routes.
   - This lets customers override specific providers while falling back to system defaults for others.

2. **Target matching**: Routes are matched by exact `target` value. A target of `all` acts as a catch-all and is used as the fallback when no country-code-specific route exists. The dialroute lookup typically queries both the specific country code and `all`.

3. **Priority ordering**: Within a customer's routes for a given target, lower `priority` values are returned first.

4. **Provider type**: Currently all providers are type `sip`. The type field exists for future extensibility.

5. **Tech prefix/postfix**: These fields allow dial string manipulation per provider. For example, a provider requiring `+` prefix: `tech_prefix = "+"`. Applied by the call manager when constructing the SIP URI.

6. **Tech headers**: Custom SIP headers injected into outbound INVITE messages. Used for carrier authentication, routing hints, or trunk-specific requirements.

7. **Soft deletes**: Both `providers` and `routes` tables use `tm_delete` sentinel `9999-01-01 00:00:00.000000`.

8. **No event subscriptions or publishing**: This service neither subscribes to nor publishes events.

9. **External SIP gateway addresses**: The `external_sip_gateway_addresses` config provides the list of SIP gateway IPs used for provider call routing. Referenced during provider setup.
