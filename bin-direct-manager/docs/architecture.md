# bin-direct-manager — Architecture

## Component Overview

`bin-direct-manager` is a Class A Go RPC microservice that manages direct hash records for SIP URI routing. Each direct record maps a unique, regeneratable hash to a customer resource (extension, conference, AI agent, AI team, or human agent), enabling direct SIP URI dialing without requiring a phone number.

**Binary:** `direct-manager` (daemon) + `direct-control` (CLI tool)

**Packages:**

| Package | Role |
|---------|------|
| `cmd/direct-manager` | Daemon entry point; wires config, DB, cache, and handlers |
| `cmd/direct-control` | CLI tool for direct DB/cache management (bypasses RabbitMQ) |
| `pkg/config` | Configuration singleton via Cobra + Viper |
| `pkg/listenhandler` | RabbitMQ RPC request handler with regex URI routing |
| `pkg/subscribehandler` | Event subscriber for customer deletion cascades — consumes via a pattern binding on the global topic exchange `bin-manager.event`, the sole intake mechanism since VOIP-1407 |
| `pkg/directhandler` | Core business logic for direct hash CRUD and regeneration |
| `pkg/dbhandler` | MySQL operations via `Masterminds/squirrel` |
| `pkg/cachehandler` | Redis cache for hash-based lookups |
| `models/direct` | Direct struct, event types, webhook |

## Layer Responsibilities

```
RabbitMQ
   │
   ├── listenhandler      ← RPC requests (CRUD, hash lookup, regenerate)
   │       │
   │       └── directhandler   ← business logic, hash generation
   │               │
   │               ├── dbhandler      ← MySQL (direct records)
   │               └── cachehandler   ← Redis (hash index)
   │
   └── subscribehandler   ← customer_deleted events → cascade delete
```

- **listenhandler**: URI regex routing only; no business logic.
- **directhandler**: Manages direct hash lifecycle including random hash generation and regeneration. Calls dbhandler and cachehandler.
- **dbhandler**: `Masterminds/squirrel` SQL builder; soft-delete pattern.
- **cachehandler**: Redis hash index for O(1) lookup by hash value.
- **subscribehandler**: Handles `customer_deleted` events by removing all direct records for that customer.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.direct-manager.request`. The `listenhandler` matches the URI against compiled regex patterns:

| Pattern | Methods | Description |
|---------|---------|-------------|
| `/v1/directs$` | POST | Create a new direct hash |
| `/v1/directs\?` | GET | List directs (pagination via page_size/page_token) |
| `/v1/directs/by-hash/` | GET | Get direct by hash value |
| `/v1/directs/{uuid}/regenerate$` | POST | Regenerate hash for existing direct |
| `/v1/directs/{uuid}$` | GET, DELETE | Get or delete direct by ID |

Unmatched URIs return `404`. Mismatched HTTP methods return `405`.

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes from the queue `bin-manager.direct-manager.subscribe`. Since VOIP-1406 the queue is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 1 pattern total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `customer-manager.customer.*.deleted` | Customer deletion — cascade-deletes all direct records of that customer |

As of VOIP-1407 this topic-pattern binding is the **sole intake mechanism**; the old per-service fanout subscription (`QueueSubscribe` to `bin-manager.customer-manager.event`) has been removed from `Run()` entirely, along with the fanout-unbind step that used to follow a successful topic bind. A topic bind failure is now fatal to `Run()` — there is no fanout fallback left to degrade to.

## Events Published

Exchange: the global topic exchange `bin-manager.event`, routing key `direct-manager.<resource>.<subscription-id>.<action>`.

Both `cmd/direct-manager` and `cmd/direct-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`. **As of VOIP-1407, this is the sole publish path** — the previous per-service fanout exchange `bin-manager.direct-manager.event` is no longer published to, and (per the operational runbook in `docs/reference/rabbitmq-queues-reference.md`) will eventually be deleted from the broker. This service's publish-side behavior change comes entirely from `bin-common-handler/pkg/notifyhandler`'s shared library update (its own consumer-side subscribehandler code also changed separately for VOIP-1407, see the Event Subscriptions section above). Both construction sites must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure now propagates to the caller as an error (previously it was swallowed silently).

Every direct event is addressed by the direct's OWN id (the default top-level `id` fallback; no `eventtopic.SubscriptionIdentifier` override exists in this service) — never by the `resource_id` of the agent/queue/conference/... it fronts. A consumer following one direct binds `direct-manager.direct.<direct-id>.#`; a regenerate keeps the same id and only rotates the hash, so an instance binding survives regeneration. The exact keys are pinned by `models/direct/routingkey_golden_test.go`; the schema lives in the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md`.

| Event type | Trigger |
|-----------|---------|
| `direct.EventTypeDirectCreated` | Direct hash created |
| `direct.EventTypeDirectDeleted` | Direct deleted |
| `direct.EventTypeDirectRegenerated` | Direct hash regenerated (same direct id) |
