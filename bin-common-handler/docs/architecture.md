# bin-common-handler Architecture

This is a **Class C** service: a shared Go library. It has no `cmd/` entrypoint, no RabbitMQ listener, and is never deployed independently. Every `bin-*-manager` service imports it.

## Package Overview

```
bin-common-handler/
├── models/
│   ├── identity/       — common resource identity (ID, CustomerID)
│   ├── sock/           — message broker types (Request, Response, Event)
│   ├── address/        — address-related models
│   ├── service/        — service definitions
│   └── outline/        — service names and canonical queue names
└── pkg/
    ├── requesthandler/         — typed inter-service RPC client
    ├── notifyhandler/          — event publishing and webhook notification
    ├── sockhandler/            — abstract message-broker interface (RabbitMQ)
    ├── rabbitmqhandler/        — RabbitMQ connection and queue management
    ├── circuitbreakerhandler/  — per-target circuit breaker
    ├── databasehandler/        — DB utilities (PrepareFields, GetDBFields, ScanRow)
    └── utilhandler/            — general utilities (UUID, hashing, time, email, URL)
```

## Package Responsibilities

### models/identity
Defines the common `Identity` struct embedded in every resource:
- `ID` — resource UUID
- `CustomerID` — owning customer UUID

All resources in the monorepo follow this identity pattern.

### models/sock
Wire types for the RabbitMQ RPC transport:
- `sock.Request` — URI, Method (GET/POST/PUT/DELETE), Publisher, Data
- `sock.Response` — StatusCode, DataType, Data
- `sock.Event` — Type, Publisher, Data

### models/outline
Canonical constants for service names and queue names. Services look up their request queue and event queue names from this package — no hardcoded queue strings in individual services.

### pkg/requesthandler
The canonical inter-service RPC client. Provides typed methods for every manager service:
- Example: `BillingV1BillingGets`, `CallV1CallCreate`, `FlowV1ActiveflowGet`
- Each method builds a `sock.Request` and dispatches via `sockhandler`
- All paths go through `sendRequest()`, which wraps the circuit breaker transparently
- Responses are unmarshaled into typed structs from the target service's `models/` package

This is the only correct way for one service to call another. Adding side-channel RPC bypasses the circuit breaker.

### pkg/notifyhandler
Event publishing and webhook delivery:
- `PublishEvent` — publishes to RabbitMQ topic exchange
- `PublishWebhook` — delivers HTTP webhook notifications to customer endpoints
- Supports delayed delivery via RabbitMQ delay exchange

### pkg/sockhandler
Abstract message-broker interface. Currently backed by RabbitMQ. Consumer services receive this as a dependency injection and should not depend on `rabbitmqhandler` directly.

### pkg/rabbitmqhandler
Low-level RabbitMQ connection management, channel lifecycle, queue declarations. Used internally by `sockhandler`. Not intended for direct use by consumer services.

### pkg/circuitbreakerhandler
Per-target circuit breaker. Default tuning: 5 consecutive failures → 30-second open window.
- States: closed → open → half-open → closed
- Integrated into every `requesthandler.sendRequest()` call
- Consumer services get CB protection for free; do not add a second CB on top

See [docs/patterns/circuit-breaker.md](../docs/patterns/circuit-breaker.md) for state machine and tuning guidance.

### pkg/databasehandler
DB helper utilities:
- `PrepareFields` — generates `?` placeholders for SQL INSERT/UPDATE
- `GetDBFields` — reflects struct db tags into column name lists
- `ScanRow` — scans sql.Row/sql.Rows results into structs

### pkg/utilhandler
General-purpose utilities:
- UUID generation and parsing
- SHA-1/SHA-256 hashing helpers
- Timestamp formatting
- Email address validation
- URL manipulation helpers

## Admission Rule

**A package may only live in `bin-common-handler` if it is used by 3 or more services.**

- Single-consumer or dual-consumer packages belong in the consuming service(s).
- `rabbitmqhandler` is exempt — it is internal plumbing used only by `sockhandler`, which itself serves the whole library.
- If a package's usage later grows to 3+ services, it can be promoted here.

Every change to `bin-common-handler` triggers verification across all 37 consumer services. Keep the library lean.

## Prometheus Metrics

`bin-common-handler` does not register metrics in its own namespace. When a consumer service constructs a `requesthandler.RequestHandler`, the constructor registers metrics scoped to the consumer:

| Metric pattern | Type | Description |
|---------------|------|-------------|
| `<ns>_request_process_time` | Histogram | RPC request duration |
| `<ns>_event_publish_total{type}` | Counter | Events published |
| `<ns>_notify_total` / `<ns>_notify_process_time` | Counter/Histogram | Webhook delivery |
| `<ns>_circuitbreaker_state{target}` | Gauge | 0=closed, 1=open, 2=half-open |
| `<ns>_circuitbreaker_state_transitions_total{target,from,to}` | Counter | CB state transitions |
| `<ns>_circuitbreaker_rejected_total{target}` | Counter | Open-state rejections |

Consumer services must not register metrics with the same names — `prometheus.MustRegister` panics on duplicates. See [docs/workflows/common-gotchas.md](../docs/workflows/common-gotchas.md).
