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

`pkg/subscribehandler` declares the durable queue `bin-manager.timeline-manager.subscribe`. As of VOIP-1407, this queue's event intake is composed of three independent mechanisms that must not be conflated:

1. **Topic-pattern binding (sole intake for the VOIP-1406 migrated events).** A single `#` catch-all pattern binding on the global `bin-manager.event` topic exchange (the package-level `topicPatterns` var, `[]string{"#"}`, pinned by `binding_golden_test.go`): timeline receives **everything on `bin-manager.event`** — all current and future topic publishers, a superset of the old ~25 per-service fanout subscriptions it replaces. `Run()` declares the exchange (`TopicCreateWithKind`) and binds `#`; the old per-service fanout `QueueSubscribe` loop and the matching fanout-unbind step have been removed entirely, so there is no fanout fallback left — a declare or bind failure here is now fatal and `Run()` returns an error at boot.
2. **The retained asterisk fanout leg (permanent, unaffected by this migration).** `Run()` separately calls `QueueSubscribe` on `asterisk.all.event` (`commonoutline.QueueNameAsteriskEventAll`). This is fed by `voip-asterisk-proxy`, which does not publish to the topic exchange and is entirely out of scope for the VOIP-1406/1407 cutover (design §3.2 calls this leg permanent) — it is not going away.
3. **The VOIP-1258 webhook-topic-bind block (pre-existing, unrelated system — left as-is).** `Run()` also binds `#` on the separate `bin-manager.webhook-manager.event.topic` exchange and, only on success, unbinds the old `bin-manager.webhook-manager.event` fanout exchange. This predates and is independent of the VOIP-1406/1407 global-topic migration; its failure semantics stay log-only/non-fatal (stays on the old exchange rather than risk being bound to neither).

The former "sentinel exchange declare before bind" guard (a `TopicCreate` call for `bin-manager.sentinel-manager.event` immediately before that target's `QueueSubscribe`, needed because `sentinel-manager` is Kubernetes-only and its exchange might not exist) no longer applies — it lived inside the per-service fanout `QueueSubscribe` loop, which VOIP-1407 removed along with the rest of that loop.

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
