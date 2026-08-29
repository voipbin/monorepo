# bin-transfer-manager Architecture

## Component Overview

`bin-transfer-manager` handles call transfer operations — both blind (immediate) and attended (consultative) — in the VoIPbin VoIP platform. It coordinates with `bin-call-manager` to manipulate confbridge state and groupcall lifecycles, and reacts to call events to progress or roll back transfer state machines.

```
cmd/transfer-manager/main.go
    ├── MySQL connection (pkg/dbhandler)
    ├── Redis cache connection (pkg/cachehandler)
    ├── RabbitMQ connection (sockhandler)
    ├── runServiceListen()     → pkg/listenhandler
    ├── runServiceSubscribe()  → pkg/subscribehandler
    └── Prometheus metrics endpoint (:2112)
```

Key packages:

| Package | Role |
|---------|------|
| `pkg/listenhandler` | RabbitMQ RPC routing via regex patterns |
| `pkg/subscribehandler` | Consumes call-manager events for state transitions via pattern bindings on the global topic exchange `bin-manager.event` — the sole intake mechanism since VOIP-1407 removed the old per-service fanout subscription |
| `pkg/transferhandler` | Core transfer state machine (attended + blind) |
| `pkg/dbhandler` | MySQL persistence |
| `pkg/cachehandler` | Redis transfer lookups by call ID |
| `models/transfer` | Transfer struct, Type enum, Webhook model |

## Layer Responsibilities

```
listenhandler           — deserializes RPC, routes by URI+method regex
subscribehandler        — consumes call-manager events, drives state transitions
    │
    └─ transferhandler  — attended/blind block-execute-unblock workflows
            │
            ├─ dbhandler     — MySQL persistence
            ├─ cachehandler  — Redis lookups by call ID or groupcall ID
            └─ requesthandler → bin-call-manager RPC (confbridge flags, call mute/MOH)
```

Rules:
- `transferhandler` owns all business logic; listen and subscribe handlers only route.
- State transitions are driven by call-manager events (not timers or polling).
- Rollback (`unblock`) is always idempotent — safe to call on failure paths.

## Request Routing

Requests arrive on queue `bin-manager.transfer-manager.request`. The listenhandler routes using regex patterns:

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| POST | `/v1/transfers` | `v1TransfersPost` — start a transfer (attended or blind) |
| GET | `/v1/transfers/{id}` | `v1TransfersIDGet` — get transfer by ID |
| GET | `/v1/transfers` | `v1TransfersGet` — list transfers |
| DELETE | `/v1/transfers/{id}` | `v1TransfersIDDelete` — cancel / rollback transfer |

No per-pod queue routing — all replicas share MySQL + Redis state.

Event subscriptions drive state transitions. Since VOIP-1406 the subscribe queue (`bin-manager.transfer-manager.subscribe`) is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 3 patterns total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `call-manager.groupcall.*.progressing` | `groupcall_progressing` — advance attended/blind transfer state |
| `call-manager.groupcall.*.hangup` | `groupcall_hangup` — roll back or finalize transfer |
| `call-manager.call.*.hangup` | `call_hangup` — transferer/transferee hangup handling |

As of VOIP-1407, this topic-pattern binding is the **sole intake mechanism** — the old per-service fanout `QueueSubscribe` call (to `bin-manager.call-manager.event`) and the fanout-unbind step that used to follow a successful topic bind have both been removed. A declare or bind failure is now fatal; there is no fanout fallback left to degrade to.
