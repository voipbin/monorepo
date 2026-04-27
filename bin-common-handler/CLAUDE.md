# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-common-handler` is a shared Go library used by every other manager service in this monorepo. It provides reusable handlers, models, and utilities for inter-service RPC, event publishing, RabbitMQ plumbing, database helpers, and shared identity types.

**Key Concepts:**
- **Shared library, not a service.** No `cmd/` entrypoint, no listen queue, no subscriptions of its own. Every change here ripples to 30+ consumers, so the [admission rule](../CLAUDE.md#critical-bin-common-handler-admission-rule) (3+ consumers required) is strict.
- **Inter-service contract.** `requesthandler` is the canonical RPC fan-out — every service-to-service RPC in VoIPbin goes through it.
- **Event publishing.** `notifyhandler` publishes events and webhooks for all services.
- **Cross-cutting infrastructure.** `circuitbreakerhandler`, `sockhandler`, and `rabbitmqhandler` provide the shared transport and resilience layer that consumer services get for free.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-common-handler`.

## Architecture

### Package Organization

**models/** — core data structures and types
- `identity/` — resource identity types (ID, CustomerID)
- `sock/` — message broker types (Request, Response, Event)
- `address/` — address-related models
- `service/` — service definitions
- `outline/` — service names and queue names

**pkg/** — reusable handler packages
- `requesthandler/` — typed inter-service RPC client for every manager service
- `notifyhandler/` — event publishing and webhook notification
- `rabbitmqhandler/` — RabbitMQ connection and queue management
- `sockhandler/` — abstract message-broker interface (currently RabbitMQ)
- `circuitbreakerhandler/` — per-target circuit breaker integrated into every `r.sendRequest()`
- `databasehandler/` — DB utilities and helpers (`PrepareFields`, `GetDBFields`, `ScanRow`)
- `utilhandler/` — general utilities (UUID, hashing, time, email validation, URL)

### Key Architectural Patterns

**Inter-Service Communication (sock model)**
- `sock.Request` — RPC-style request with URI, Method (GET/POST/PUT/DELETE), Publisher, Data
- `sock.Response` — response with StatusCode, DataType, Data
- `sock.Event` — pub/sub event with Type, Publisher, Data

**RequestHandler Pattern**
- Each manager service has dedicated typed methods (e.g., `BillingV1BillingGets`, `CallV1CallCreate`)
- Methods construct `sock.Request` and dispatch via `sockhandler` over RabbitMQ
- Responses are unmarshaled into typed structs from the target service's `models/` package
- All paths go through `r.sendRequest()`, which transparently applies the per-target circuit breaker

**NotifyHandler Pattern**
- `PublishEvent` — publish events to topic exchanges
- `PublishWebhook` — send webhook notifications to customer endpoints
- Supports delayed message delivery via the RabbitMQ delay exchange

**Mock Generation**
All main handler interfaces use `go:generate` with mockgen:

```go
//go:generate mockgen -package packagename -destination ./mock_main.go -source main.go -build_flags=-mod=mod
```

### Queue Types
- **Normal queues** — durable, survive broker restarts. Used for shared per-service request queues.
- **Volatile queues** — auto-delete when unused, used for per-pod queues and temporary operations. See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md).

### Identity Model

All resources follow a common identity pattern:
- `ID` — resource UUID
- `CustomerID` — owning customer UUID

## Patterns hosted in this service

`bin-common-handler` hosts the canonical implementation of several cross-cutting patterns. When you change these packages, also update the corresponding pattern doc.

- **Circuit Breaker** (`pkg/circuitbreakerhandler/`) — every `requesthandler.r.sendRequest()` call is wrapped by a per-target CB with sensible defaults (5 failures → 30s open). See [docs/patterns/circuit-breaker.md](../docs/patterns/circuit-breaker.md) for state machine, metrics, and tuning guidance. Most consumer services should NOT roll their own breaker — they get this for free by routing through `requesthandler`.
- **Per-pod RPC routing** (`pkg/requesthandler/pipecat_message.go` and per-pod method variants) — the client side of the per-pod queue convention. See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md).
- **Per-pod liveness preflight** (`pkg/requesthandler/pipecat_ping.go`) — sub-second `XxxV1Ping` RPCs that share the CB to fast-fail dead pods. See [docs/patterns/per-pod-liveness-preflight.md](../docs/patterns/per-pod-liveness-preflight.md).

## Common Commands

### Test
```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./pkg/utilhandler

# Run a single test
go test ./pkg/utilhandler -run TestIsValidEmail

# Verbose
go test -v ./...
```

### Generate Mocks
```bash
# All packages
go generate ./...

# Single package
go generate ./pkg/notifyhandler
```

### Lint
```bash
golangci-lint run -v --timeout 5m
```

> The pre-commit verification workflow and its rationale live in the root [CLAUDE.md](../CLAUDE.md#critical-verification-before-commit) and [docs/workflows/verification-workflows.md](../docs/workflows/verification-workflows.md). Because this library is consumed by 30+ services, changes here may require running verification in every consuming service — see [docs/workflows/special-cases.md](../docs/workflows/special-cases.md) for the cross-service workflow.

## Monorepo Context

`bin-common-handler` is the only Go module in the monorepo with **no `replace` directives pointing back at it from itself** — it is a leaf dependency. Every other `bin-*-manager` declares it as a `replace` in `go.mod`:

```
replace monorepo/bin-common-handler => ../bin-common-handler
```

When you change a public API (handler interface, exported function signature, model field), you MUST sweep every consumer for build/test breakage. Sed-style bulk edits frequently miss multi-line call sites with different import aliases — see the [Updating Shared Library Function Signatures gotcha](../docs/workflows/common-gotchas.md) before doing one.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces generated in the same package as the interface definition (`mock_*.go`)
- Table-driven tests with struct slices
- Tests verify type conversions (UUID to bytes, JSON marshaling)
- Co-located: `<package>/<feature>_test.go`

Example test structure:

```go
tests := []struct {
    name      string
    input     InputType
    mockSetup func(*MockHandler)
    expectRes ResultType
    expectErr bool
}{
    {"success case", input1, setupMock1, expected1, false},
    {"error case", input2, setupMock2, nil, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // test implementation
    })
}
```

## Key Implementation Details

### `sendRequest` is the single chokepoint
Every `requesthandler` method funnels through `pkg/requesthandler/send_request.go`. The circuit breaker is wrapped here, so any RPC that bypasses `sendRequest` loses CB protection — don't add side channels.

### Identity tags on models
UUID fields on shared models MUST use the `,uuid` db tag; JSON fields MUST use the `,json` db tag. See [docs/conventions/models.md](../docs/conventions/models.md) and [docs/conventions/database.md](../docs/conventions/database.md).

### Request/event helpers must be thin
Anything that grows beyond plumbing (e.g., business logic, retries beyond the CB defaults) belongs in the consumer service, not here. The admission rule (3+ consumers) is the ceiling for what gets pulled into this library.

## Configuration

`bin-common-handler` is a library, not a service — it has no configuration of its own. Consumer services pass configuration (RabbitMQ address, namespace, queue names) into the constructors of the handlers they use (`requesthandler.NewRequestHandler`, `notifyhandler.NewNotifyHandler`, etc.).

## Prometheus Metrics

`bin-common-handler` does not register metrics into its own namespace; instead, when a consumer service constructs a `requesthandler.RequestHandler`, the constructor registers per-namespace metrics scoped to the *consumer*:

- `<consumer_namespace>_request_process_time` — histogram of RPC request processing time
- `<consumer_namespace>_event_publish_total{type}` — counter of events published
- `<consumer_namespace>_circuitbreaker_state{target}` — gauge: 0 closed, 1 open, 2 half-open
- `<consumer_namespace>_circuitbreaker_state_transitions_total{target,from,to}` — counter
- `<consumer_namespace>_circuitbreaker_rejected_total{target}` — counter for Open-state rejections

> **Gotcha:** because these names are registered by `requesthandler.initPrometheus()`, consumer services must NOT also register metrics with the same name in their own `metricshandler` package — `prometheus.MustRegister` panics on duplicate names. See [docs/workflows/common-gotchas.md](../docs/workflows/common-gotchas.md) (Prometheus Metric Name Conflicts).
