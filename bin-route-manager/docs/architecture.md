# bin-route-manager — Architecture

## Component Overview

`bin-route-manager` is a Class A Go RPC microservice that manages outbound call routing. It stores provider configurations (SIP gateways) and routing rules, and provides a dialroute resolution API that merges customer-specific routes with system-default fallback routes.

**Binary:** `route-manager` (daemon) + `route-control` (CLI tool)

**Packages:**

| Package | Role |
|---------|------|
| `cmd/route-manager` | Daemon entry point; wires config, DB, cache, and handlers |
| `cmd/route-control` | CLI tool for direct DB/cache management (bypasses RabbitMQ) |
| `pkg/listenhandler` | RabbitMQ RPC request handler with regex URI routing |
| `pkg/routehandler` | Route CRUD and dialroute merge/selection logic |
| `pkg/providerhandler` | Provider CRUD |
| `pkg/dbhandler` | MySQL + Redis coordination |
| `pkg/cachehandler` | Redis cache for provider/route lookups |
| `models/provider` | Provider struct and type constants |
| `models/route` | Route struct, `CustomerIDBasicRoute` constant |

## Layer Responsibilities

```
RabbitMQ
   │
   └── listenhandler       ← RPC requests (providers, routes, dialroutes)
           │
           ├── providerhandler   ← provider CRUD
           └── routehandler      ← route CRUD + dialroute merge
                   │
                   ├── dbhandler    ← MySQL (providers, routes tables)
                   └── cachehandler ← Redis (provider/route cache)
```

- **listenhandler**: URI regex routing only; no business logic.
- **providerhandler**: Manages SIP provider configurations (hostname, tech prefix/postfix, SIP headers).
- **routehandler**: Manages per-customer route mappings. The `DialrouteGets` function implements fallback merge logic between customer routes and the system default.
- **dbhandler**: `Masterminds/squirrel` query builder; soft-delete pattern for both tables.
- **cachehandler**: Redis cache reduces DB load for frequent dialroute lookups during call setup.

## Event Publishing

Both NotifyHandler construction sites — `cmd/route-manager` and `cmd/route-control` — are built with `notifyhandler.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange `bin-manager.route-manager.event` (unchanged, still the system of record) and once to the global topic exchange `bin-manager.event` with the routing key `route-manager.<resource>.<resource-id>.<action>`. The two sites must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure never propagates to the caller and never affects the fanout publish.

Three resource namespaces publish: `route`, `provider`, and `providercall`. Each is an independent top-level resource addressed by its OWN id, resolved by the default JSON `id` fallback — this service declares no `eventtopic.SubscriptionIdentifier` override, even though its models declare `ID` directly instead of embedding `commonidentity.Identity` (the fallback reads the `json:"id"` tag, so the two forms behave identically). Note that `provider` and `providercall` are separate namespaces despite the string-prefix relationship: AMQP topic matching is per-segment, so a `route-manager.provider.#` binding never receives providercall events. `models/route/routingkey_golden_test.go` pins the exact key of every published event type, that override absence, and the namespace separation. See the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the key schema.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.route-manager.request`. The `listenhandler` matches the URI against compiled regex patterns:

| Pattern | Methods | Description |
|---------|---------|-------------|
| `/v1/providers/setup$` | POST | Initial provider setup |
| `/v1/providers$` | POST | Create provider |
| `/v1/providers\?` | GET | List providers with pagination |
| `/v1/providers/{uuid}$` | GET, PUT, DELETE | Get / update / delete provider |
| `/v1/providercalls$` | POST | Create provider call record |
| `/v1/providercalls(\?.*)?$` | GET | List provider call records |
| `/v1/providercalls/{uuid}$` | GET | Get provider call record |
| `/v1/routes$` | POST | Create route |
| `/v1/routes(\?.*)?$` | GET | List routes with pagination |
| `/v1/routes/{uuid}$` | GET, PUT, DELETE | Get / update / delete route |
| `/v1/dialroutes(\?.*)?$` | GET | Get effective routes for a (customer, target) pair |

Unmatched URIs return `404`. Mismatched HTTP methods return `405`.
