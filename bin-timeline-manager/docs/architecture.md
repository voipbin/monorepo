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

**ClickHouse type constraint**: The ClickHouse Go driver only supports basic Go types (`string`, `int`, `time.Time`) for column scanning. Domain models in `models/event/` use `string` for `Publisher` and `EventType`. Conversion to richer types (e.g., `ServiceName`) happens in `eventhandler` at the API boundary.

## Request Routing

Requests arrive on queue `bin-manager.timeline-manager.request`. The listenhandler routes using regex patterns:

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| POST | `/v1/events` | `v1EventsPost` — list events with filters (publisher, resource ID, event-type wildcards) |
| POST | `/v1/aggregated-events` | `v1AggregatedEventsPost` — aggregated event view |
| GET | `/v1/sip/analysis` | `v1SIPAnalysisGet` — SIP call analysis (via Homer) |
| GET | `/v1/sip/pcap` | `v1SIPPcapGet` — SIP PCAP capture retrieval (via Homer) |

Event listing uses cursor-based pagination with `PageSize` (default 100, max 1000) and `PageToken`.

No per-pod queue routing — all replicas share the same ClickHouse database.
