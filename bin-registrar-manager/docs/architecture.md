# bin-registrar-manager — Architecture

## Component Overview

`bin-registrar-manager` is a Class A Go RPC microservice that manages SIP registrations for the VoIPbin platform. It handles the lifecycle of Asterisk PJSIP extensions (user endpoints) and trunks (carrier connections), including provisioning of Asterisk database tables and caching of active contact registrations.

This service is unique in using **two separate databases**: the Asterisk DB (`ps_*` tables) and the bin-manager DB (extensions, trunks, sip_auth).

**Binary:** `registrar-manager` (daemon) + `registrar-control` (CLI tool)

**Packages:**

| Package | Role |
|---------|------|
| `cmd/registrar-manager` | Daemon entry point; establishes both DB connections |
| `cmd/registrar-control` | CLI tool for direct management (bypasses RabbitMQ); hosts the VOIP-1385 domain migration batch (`domain-migrate`, `domain-migrate-rollback`) |
| `pkg/listenhandler` | RabbitMQ RPC request handler with regex URI routing |
| `pkg/subscribehandler` | Event subscriber: customer domain provisioning on create, cascading cleanup on delete |
| `pkg/extensionhandler` | Business logic for SIP extension lifecycle |
| `pkg/trunkhandler` | Business logic for SIP trunk lifecycle |
| `pkg/customerdomainhandler` | Customer SIP domain (label/realm) lifecycle: get-or-create with 4-char base36 label generation, realm lookup, endpoint composition |
| `pkg/contacthandler` | Reads active SIP contacts from Asterisk DB; Redis-cached |
| `pkg/dbhandler` | Unified DB abstraction over both MySQL databases |
| `pkg/cachehandler` | Redis-backed contact cache + realm -> customer-domain cache |

## Layer Responsibilities

```
RabbitMQ
   │
   ├── listenhandler       ← RPC requests (extensions, trunks, contacts)
   │       │
   │       ├── extensionhandler  ← SIP extension CRUD
   │       │       │
   │       │       └── dbhandler ← bin-manager DB (extensions table)
   │       │                       + Asterisk DB (ps_endpoints, ps_aors, ps_auths)
   │       │
   │       ├── trunkhandler      ← SIP trunk CRUD
   │       │       │
   │       │       └── dbhandler ← bin-manager DB (trunks table)
   │       │                       + Asterisk DB (ps_endpoints, ps_aors, ps_auths)
   │       │
   │       ├── contacthandler    ← Active SIP registrations (read-only)
   │       │       │
   │       │       └── dbhandler + cachehandler ← Asterisk ps_contacts + Redis
   │       │
   │       └── customerdomainhandler ← customer SIP domain (label/realm)
   │               │
   │               └── dbhandler ← bin-manager DB (registrar_customer_domains)
   │                               + Redis (realm -> row cache)
   │
   └── subscribehandler    ← customer_created → ensure customer domain row
                             customer_deleted → cleanup extensions + trunks
                                                + customer domain row
```

- **listenhandler**: URI regex routing; no business logic.
- **extensionhandler**: Creates/deletes extension records in bin-manager DB AND corresponding `ps_endpoints`/`ps_aors`/`ps_auths` entries in Asterisk DB atomically.
- **trunkhandler**: Same pattern for trunks; supports basic (user/pass) and IP-based authentication modes.
- **contacthandler**: Read-only view of active registrations from Asterisk `ps_contacts` table; Redis-cached. Endpoint strings are reconstructed via `customerdomainhandler.EndpointGet` (lookup-only — read paths never create a domain row).
- **customerdomainhandler**: Owns the `registrar_customer_domains` mapping (one row per customer: `domain_label` + full `realm`). Rows are created on the `customer_created` event, lazily on the first extension create (`EnsureByCustomerID`), and hard-deleted on `customer_deleted`. New-row shape is gated by `domain_short_label_enabled` (default `false`): `<uuid>.<base>` rows when disabled, 4-char base36 short labels when enabled (the production setting since the VOIP-1385 short-domain cutover); the `domain-migrate` batch generates short labels regardless of the flag. The realm lookup backs the `/v1/customer_domains/realm/{realm}` RPC used by bin-call-manager's incoming-call resolution and is Redis-cached (realm -> row).
- **dbhandler**: Abstracts both DB connections; uses `Masterminds/squirrel` for query building.

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes from the queue `bin-manager.registrar-manager.subscribe`. Since VOIP-1406 the queue is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 2 patterns total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `customer-manager.customer.*.created` | Ensure the customer's SIP domain row exists |
| `customer-manager.customer.*.deleted` | Cascading cleanup: extensions + trunks + customer domain row |

The old per-service **fanout subscription is retained in code as the rollback surface until VOIP-1407** (`QueueSubscribe` to `bin-manager.customer-manager.event`); on each boot Run() re-subscribes it, then unbinds it again after the topic binds succeed.

## Event Publishing

All four NotifyHandler construction sites — `cmd/registrar-manager`, and all three in `cmd/registrar-control` (`initExtensionHandler`, `initDomainMigrationDeps`, `initTrunkHandler`) — are built with `notifyhandler.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange `bin-manager.registrar-manager.event` (unchanged, still the system of record) and once to the global topic exchange `bin-manager.event` with the routing key `registrar-manager.<resource>.<resource-id>.<action>`. The four sites must stay in lockstep on this option — enabling it in only some would leave consumers with gaps depending on which process published. A topic publish failure never propagates to the caller and never affects the fanout publish.

`initDomainMigrationDeps` is included for publisher-stream completeness rather than handler wiring: the `domain-migrate` batch publishes `extension_updated` directly from cmd code (`cmd/registrar-control/domain_migrate.go`), so leaving it fanout-only would make an extension's topic stream silently incomplete across a migration.

Two resource namespaces publish: `trunk` and `extension`. Both are independent top-level resources addressed by their own id, returned by each type's explicit `EventSubscriptionID()` method (`eventtopic.SubscriptionIdentifier`, mandatory since VOIP-1419 — there is no JSON fallback; an empty return degrades to the `-` placeholder). `models/trunk/routingkey_golden_test.go` pins the exact key of every published event type (including the migration-batch publish path). See the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the key schema and `docs/plans/2026-08-28-voip-1419-explicit-subscription-id-design.md` for the resolution mechanism.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.registrar-manager.request`. The `listenhandler` matches the URI against compiled regex patterns:

| Pattern | Methods | Description |
|---------|---------|-------------|
| `/v1/contacts(\?.*)?$` | GET | List active SIP contacts (from Asterisk ps_contacts) |
| `/v1/customer_domains/realm/{realm}$` | GET | Get customer domain (customer mapping) by SIP realm |
| `/v1/extensions/count_by_customer$` | GET | Count extensions per customer |
| `/v1/extensions$` | POST | Create SIP extension |
| `/v1/extensions\?` | GET | List extensions with filters |
| `/v1/extensions/{uuid}/direct-hash-regenerate$` | POST | Regenerate extension's direct hash |
| `/v1/extensions/{uuid}$` | GET, PUT, DELETE | Get / update / delete extension |
| `/v1/extensions/endpoint/{uuid}$` | GET | Get extension by Asterisk endpoint ID |
| `/v1/extensions/extension/{uuid}(\?.*)?$` | GET | Get extension by extension number |
| `/v1/trunks/count_by_customer$` | GET | Count trunks per customer |
| `/v1/trunks$` | POST | Create SIP trunk |
| `/v1/trunks\?` | GET | List trunks with filters |
| `/v1/trunks/{uuid}$` | GET, PUT, DELETE | Get / update / delete trunk |
| `/v1/trunks/domain_name/` | GET | Get trunk by domain name |

Unmatched URIs return `404`. Mismatched HTTP methods return `405`.
