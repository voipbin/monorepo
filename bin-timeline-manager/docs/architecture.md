# bin-timeline-manager Architecture

## Component Overview

`bin-timeline-manager` is the platform-wide audit log and event timeline service. It subscribes to events from 27 other services across the platform, stores them in ClickHouse for high-throughput time-series queries, and exposes a read API over RabbitMQ RPC.

```
cmd/timeline-manager/main.go
    ├── ClickHouse connection (pkg/dbhandler)
    ├── RabbitMQ connection (sockhandler)
    ├── runServiceListen()     → pkg/listenhandler
    ├── runServiceSubscribe()  → (implicit — via subscription queues)
    └── Prometheus metrics endpoint (:2112)
```

Key packages:

| Package | Role |
|---------|------|
| `pkg/listenhandler` | RabbitMQ RPC routing for read queries |
| `pkg/eventhandler` | Business logic — event queries with cursor-based pagination |
| `pkg/dbhandler` | ClickHouse read/write operations |
| `models/event` | Domain Event struct (basic Go types for ClickHouse driver compatibility) |
| `pkg/listenhandler/models/request` | API request DTOs (richer types such as `ServiceName`) |
| `pkg/listenhandler/models/response` | API response DTOs |

## Layer Responsibilities

```
listenhandler           — deserializes RabbitMQ RPC, routes by URI+method regex
    │
    └─ eventhandler     — applies filter validation, cursor pagination logic
            │
            └─ dbhandler — ClickHouse SQL (uses basic Go types only)
```

Event ingestion path (separate from query path):
```
27 service event queues  →  subscribehandler  →  ClickHouse batch insert
```

The subscribe path is write-only; the listen path is read-only. They share only the ClickHouse database.

### Event Subscriptions

`pkg/subscribehandler` declares the durable queue `bin-manager.timeline-manager.subscribe`. Since VOIP-1406, the primary subscription is a single `#` catch-all pattern binding on the global `bin-manager.event` topic exchange (the package-level `topicPatterns` var, pinned by `binding_golden_test.go`): timeline receives **everything on `bin-manager.event`** — all current and future topic publishers, a superset of the old 25 per-service fanout subscriptions — **plus the retained asterisk fanout leg** (`asterisk.all.event`; asterisk-proxy does not publish to the topic exchange) **plus the VOIP-1258 `#` bind on the webhook-manager topic exchange** (`bin-manager.webhook-manager.event.topic`, a separate, permanently coexisting exchange).

At boot, `Run()` still subscribes the queue to every exchange in the package-level `subscribeTargets` list (see `docs/dependencies.md` for the full queue list), then declares `bin-manager.event`, binds `#`, and — only after the bind succeeds — unbinds the 25 non-asterisk fanout exchanges (`fanoutUnbindTargets`). If the declare or bind fails, the service stays fully on the fanout subscriptions; a failed fanout unbind is CRITICAL-logged but not fatal (double delivery beats loss). The fanout subscriptions are retained as the rollback surface until VOIP-1407 removes them.

**Sentinel exchange declare before bind.** `sentinel-manager` requires the Kubernetes API and is therefore only deployed in Kubernetes environments. In every other deployment (for example Docker Compose based self-hosting) nothing declares the `bin-manager.sentinel-manager.event` exchange, so binding to it fails with an AMQP 404, which closes the channel shared by all of this queue's bindings and makes `Run()` return a fatal error at boot. To avoid that, `Run()` calls `sockHandler.TopicCreate` for the sentinel target specifically, immediately before that target's `QueueSubscribe` call. `TopicCreate` declares a durable fanout exchange, the same parameters `sentinel-manager`'s own `notifyhandler` uses, so the declare is an idempotent no-op when sentinel-manager is deployed and creates the exchange when it is not.

The guard is scoped to the sentinel target only, not applied to every target. A blanket fanout declare would silently paper over a future non-fanout (topic-kind) target rather than surface the mismatch. The remaining targets are each declared by their own owning service at that service's boot; because nothing sequences this service after its event-owning peers, an unlucky boot order can still produce a transient, self-resolving restart on those targets. That is distinct from the sustained crash loop the sentinel declare removes.

**ClickHouse type constraint**: The ClickHouse Go driver only supports basic Go types (`string`, `int`, `time.Time`) for column scanning. Domain models in `models/event/` use `string` for `Publisher` and `EventType`. Conversion to richer types (e.g., `ServiceName`) happens in `eventhandler` at the API boundary.

## Request Routing

Requests arrive on queue `bin-manager.timeline-manager.request`. The listenhandler routes using regex patterns:

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| POST | `/v1/events` | `v1EventsPost` — list events with filters (publisher, resource ID, event-type wildcards) |
| POST | `/v1/aggregated-events` | `v1AggregatedEventsPost` — aggregated event view |
| GET | `/v1/correlations/<resource_id>` | `v1CorrelationsGet` — resolve a resource id to its activeflow and return all correlated resources grouped by publisher |
| GET | `/v1/sip/analysis` | `v1SIPAnalysisGet` — SIP call analysis (via Homer) |
| GET | `/v1/sip/pcap` | `v1SIPPcapGet` — SIP PCAP capture retrieval (via Homer) |

Event listing uses cursor-based pagination with `PageSize` (default 100, max 1000) and `PageToken`.

No per-pod queue routing — all replicas share the same ClickHouse database.
