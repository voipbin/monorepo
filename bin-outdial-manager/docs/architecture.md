# bin-outdial-manager — Architecture

## Component Overview

`bin-outdial-manager` is a Class A Go RPC microservice that manages outbound dialing campaigns. It stores outdial containers (campaigns), their individual targets, and per-target call attempt tracking. The primary consumer is `bin-campaign-manager`, which fetches available targets and updates their status during campaign execution.

**Binary:** `outdial-manager` (daemon) + `outdial-control` (CLI tool)

**Packages:**

| Package | Role |
|---------|------|
| `cmd/outdial-manager` | Daemon entry point; wires config, DB, cache, and handlers |
| `cmd/outdial-control` | CLI tool for direct DB/cache management (bypasses RabbitMQ) |
| `pkg/listenhandler` | RabbitMQ RPC request handler with regex URI routing |
| `pkg/outdialhandler` | Business logic for outdial container CRUD |
| `pkg/outdialtargethandler` | Business logic for outdial target management and status transitions |
| `pkg/outdialtargetcallhandler` | Tracks individual call attempts per target |
| `pkg/dbhandler` | MySQL + Redis coordination |
| `pkg/cachehandler` | Redis cache for target lookups |
| `models/outdial` | Outdial struct, event types |
| `models/outdialtarget` | OutdialTarget struct, status constants |
| `models/outdialtargetcall` | Call attempt tracking struct |

## Layer Responsibilities

```
RabbitMQ
   │
   └── listenhandler     ← RPC requests (outdials, targets, status)
           │
           ├── outdialhandler           ← outdial container CRUD + event publishing
           ├── outdialtargethandler     ← target CRUD, available-targets query
           └── outdialtargetcallhandler ← call attempt tracking
                   │
                   ├── dbhandler    ← MySQL queries (squirrel)
                   └── cachehandler ← Redis (target state cache)
```

- **listenhandler**: URI regex routing only; no business logic.
- **outdialhandler**: Manages the outdial container lifecycle; publishes create/update/delete events.
- **outdialtargethandler**: Manages individual targets; handles the `available` query (filters by try-count thresholds per destination).
- **outdialtargetcallhandler**: Tracks each call attempt against a target for retry accounting.
- **dbhandler**: SQLite-compatible schema for tests; MySQL in production.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.outdial-manager.request`. The `listenhandler` matches URI against compiled regex patterns:

| Pattern | Methods | Description |
|---------|---------|-------------|
| `/v1/outdials$` | POST | Create outdial |
| `/v1/outdials(\?.*)?$` | GET | List outdials |
| `/v1/outdials/{uuid}$` | GET, PUT, DELETE | Get / update / delete outdial |
| `/v1/outdials/{uuid}/available(\?.*)?$` | GET | Get available targets (filtered by try counts) |
| `/v1/outdials/{uuid}/targets$` | POST | Create target in outdial |
| `/v1/outdials/{uuid}/targets(\?.*)?$` | GET | List targets in outdial |
| `/v1/outdials/{uuid}/campaign_id$` | PUT | Update campaign association |
| `/v1/outdials/{uuid}/data$` | PUT | Update custom JSON data |
| `/v1/outdialtargets/{uuid}$` | GET, DELETE | Get / delete target |
| `/v1/outdialtargets/{uuid}/progressing$` | POST | Mark target as in-progress |
| `/v1/outdialtargets/{uuid}/status$` | PUT | Update target status |

Unmatched URIs return `404`. Mismatched HTTP methods return `405`.
