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
