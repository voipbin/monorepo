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

## Event Publishing

Both NotifyHandler construction sites — `cmd/outdial-manager` and `cmd/outdial-control` — are built with `notifyhandler.WithGlobalTopicPublish()`, so every event is published to the global topic exchange `bin-manager.event` with the routing key `outdial-manager.outdial.<outdial-id>.<action>`. **As of VOIP-1407, this is the sole publish path** — the previous per-service fanout exchange `bin-manager.outdial-manager.event` is no longer published to, and (per the operational runbook in `docs/reference/rabbitmq-queues-reference.md`) will eventually be deleted from the broker. No code in this service changed for VOIP-1407; the behavior change (dual publish → topic-only) comes entirely from `bin-common-handler/pkg/notifyhandler`'s shared library update. The two sites must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure now propagates to the caller as an error (previously it was swallowed silently).

The subscription address (third key segment) is the outdial's own id, resolved by the default JSON `id` fallback: this service declares no `eventtopic.SubscriptionIdentifier` override. `models/outdial/routingkey_golden_test.go` pins the exact key of every published event type and asserts that absence. The `outdialtarget_*` constants in `models/outdialtarget` are dead (never published) and are deliberately outside that table. See the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the key schema.

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
