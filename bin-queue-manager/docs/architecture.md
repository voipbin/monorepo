# Architecture: bin-queue-manager

## Component Overview

```mermaid
graph TD
    CMD["cmd/queue-manager"] --> LH["pkg/listenhandler\n(RabbitMQ RPC server)"]
    CMD --> SH["pkg/subscribehandler\n(Event consumer)"]

    LH --> QH["pkg/queuehandler\n(Queue business logic)"]
    LH --> QCH["pkg/queuecallhandler\n(Queuecall management)"]

    QH --> DBH["pkg/dbhandler\n(MySQL)"]
    QH --> Cache["pkg/cachehandler\n(Redis)"]
    QCH --> DBH
    QCH --> Cache
```

## Layer Responsibilities

| Package | Role | Key Types |
|---------|------|-----------|
| `pkg/queuehandler` | Queue CRUD, routing configuration, agent membership management, queue execution logic | `queue.Queue`, `queue.RoutingMethod` |
| `pkg/queuecallhandler` | Queuecall lifecycle: create, execute, kick, timeout handling, health checks, status transitions | `queuecall.Queuecall`, `queuecall.Status` |
| `pkg/listenhandler` | RabbitMQ RPC request router (regex pattern matching) | `sock.Request`, `sock.Response` |
| `pkg/subscribehandler` | Consumes call-manager events via pattern bindings on the global topic exchange `bin-manager.event` (fanout legs retained until VOIP-1407) | queue event structs |
| `pkg/dbhandler` | MySQL CRUD operations | all model structs |
| `pkg/cachehandler` | Redis fast-path lookups for queues and queuecalls | `queue.Queue`, `queuecall.Queuecall` |
| `models/queue` | Queue data model, routing method constants | `queue.Queue`, `queue.RoutingMethod` |
| `models/queuecall` | Queuecall data model, status constants | `queuecall.Queuecall`, `queuecall.Status` |

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes from the queue `bin-manager.queue-manager.subscribe`. Since VOIP-1406 the queue is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 3 patterns total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `call-manager.call.*.hangup` | Call hangup — kicks the queuecall out of the queue |
| `call-manager.confbridge.*.joined` / `call-manager.confbridge.*.leaved` | Confbridge join/leave — drives queuecall service/done transitions |

The `customer-manager.customer.*.deleted` pair is deliberately NOT bound: its dispatch case is unreachable today (the customer-manager fanout exchange was never subscribed) and stays that way (VOIP-1406 design §4; follow-up VOIP-1422 decides activate-or-delete — a latent-bug candidate, since queue records are likely meant to be cleaned on customer deletion).

The old per-service **fanout subscriptions are retained in code as the rollback surface until VOIP-1407** (`QueueSubscribe` to `bin-manager.call-manager.event`, `bin-manager.agent-manager.event`, `bin-manager.conference-manager.event`); on each boot Run() re-subscribes them, then unbinds all three again after the topic binds succeed. The agent and conference legs were dead binds (zero dispatch cases) and are dropped the same way.

## Event Publishing

All three NotifyHandler construction sites — `cmd/queue-manager`, and both instances in `cmd/queue-control` — are built with `notifyhandler.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange `bin-manager.queue-manager.event` (unchanged, still the system of record) and once to the global topic exchange `bin-manager.event` with the routing key `queue-manager.<resource>.<queue-id>.<action>`. The three sites must stay in lockstep on this option — enabling it in only some would leave consumers with gaps depending on which process published. A topic publish failure never propagates to the caller and never affects the fanout publish. See [docs/domain.md](domain.md) for the per-event routing keys and the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.queue-manager.request`. The `listenhandler` matches each request's URI against regex patterns and dispatches to the appropriate handler function.

| Route Pattern | Method | Description |
|---------------|--------|-------------|
| `/v1/queues/count_by_customer$` | GET | Count queues by customer ID |
| `/v1/queues$` | POST | Create a new queue |
| `/v1/queues\?{{UUID}}$` | GET | List queues with filters/pagination |
| `/v1/queues/{{UUID}}$` | GET/PUT/DELETE | Get, update, or delete a queue |
| `/v1/queues/{{UUID}}/tag_ids$` | PUT | Update queue tag IDs (agent filter) |
| `/v1/queues/{{UUID}}/routing_method$` | PUT | Update queue routing method |
| `/v1/queues/{{UUID}}/agents(\\?.*)$` | GET | List agents eligible for this queue |
| `/v1/queues/{{UUID}}/execute$` | POST | Trigger queue execution (attempt agent routing) |
| `/v1/queues/{{UUID}}/execute_run$` | POST | Run queue execution loop |
| `/v1/queues/{{UUID}}/direct-hash-regenerate$` | POST | Regenerate direct-access hash |
| `/v1/queuecalls\?{{UUID}}$` | GET | List queuecalls with filters/pagination |
| `/v1/queuecalls/{{UUID}}$` | GET/DELETE | Get or delete a queuecall |
| `/v1/queuecalls/{{UUID}}/timeout_wait$` | POST | Trigger wait timeout for a queuecall |
| `/v1/queuecalls/{{UUID}}/timeout_service$` | POST | Trigger service timeout for a queuecall |
| `/v1/queuecalls/{{UUID}}/execute$` | POST | Execute routing for a specific queuecall |
| `/v1/queuecalls/{{UUID}}/health-check$` | POST | Health check for a queuecall |
| `/v1/queuecalls/{{UUID}}/status_waiting$` | POST | Set queuecall back to waiting status |
| `/v1/queuecalls/{{UUID}}/kick$` | POST | Remove a queuecall from the queue |
| `/v1/queuecalls/reference_id/{{UUID}}$` | GET | Look up a queuecall by reference call ID |
| `/v1/queuecalls/reference_id/{{UUID}}/kick$` | POST | Kick a queuecall by reference call ID |
| `/v1/services/type/queuecall$` | POST | Create a queuecall via service type routing |
