# Architecture: bin-campaign-manager

## Component Overview

```mermaid
graph TD
    CMD["cmd/campaign-manager"] --> LH["pkg/listenhandler\n(RabbitMQ RPC server)"]
    CMD --> SH["pkg/subscribehandler\n(Event consumer)"]

    LH --> CH["pkg/campaignhandler\n(Campaign business logic)"]
    LH --> CCH["pkg/campaigncallhandler\n(Call attempt management)"]
    LH --> OH["pkg/outplanhandler\n(Outplan/dial config)"]

    CH --> DBH["pkg/dbhandler\n(MySQL)"]
    CH --> Cache["pkg/cachehandler\n(Redis)"]
    CCH --> DBH
    CCH --> Cache
    OH --> DBH
```

## Layer Responsibilities

| Package | Role | Key Types |
|---------|------|-----------|
| `pkg/campaignhandler` | Campaign lifecycle: create, execute, status transitions (stop/run/stopping), service level management, next campaign chaining | `campaign.Campaign`, `campaign.Status` |
| `pkg/campaigncallhandler` | Individual call attempt management: create calls, track outcomes, retry logic | `campaigncall.Campaigncall`, `campaigncall.Status` |
| `pkg/outplanhandler` | Outplan CRUD: dialing configuration (timeouts, retries, source), dial list management | `outplan.Outplan`, `outplan.Dial` |
| `pkg/listenhandler` | RabbitMQ RPC request router (regex pattern matching) | `sock.Request`, `sock.Response` |
| `pkg/subscribehandler` | Consumes events from call-manager and flow-manager to track call outcomes | queue event structs |
| `pkg/dbhandler` | MySQL CRUD operations | all model structs |
| `pkg/cachehandler` | Redis fast-path lookups for campaigns and campaigncalls | `campaign.Campaign`, `campaigncall.Campaigncall` |
| `models/campaign` | Campaign data model, status constants, event types | `campaign.Campaign`, `campaign.Status` |
| `models/campaigncall` | Campaigncall data model, status constants | `campaigncall.Campaigncall`, `campaigncall.Status` |
| `models/outplan` | Outplan and dial configuration data model | `outplan.Outplan`, `outplan.Dial` |

## Events Consumed

Since VOIP-1406 the subscribe queue `bin-manager.campaign-manager.subscribe` receives its service events through pattern bindings on the global topic exchange `bin-manager.event` (declared idempotently at boot), one pattern per dispatch pair:

| Pattern | Dispatch |
|---------|----------|
| `call-manager.call.*.hangup` | campaigncall outcome tracking on call hangup |
| `flow-manager.activeflow.*.deleted` | campaigncall completion on activeflow deletion |

The exact pattern set is pinned by `pkg/subscribehandler/binding_golden_test.go`. As of VOIP-1407, topic-pattern binding via `topicPatterns`/`QueueBind` is the sole intake mechanism — the fanout `QueueSubscribe` calls to `bin-manager.call-manager.event` and `bin-manager.flow-manager.event`, and the fanout-unbind rollback surface they fed, have been removed from `Run()` entirely. A `QueueCreate`, `TopicCreateWithKind`, or `QueueBind` failure now fails `Run()` immediately (`return err`); there is no fanout fallback left to degrade to.

## Event Publishing

Both `cmd/campaign-manager` and `cmd/campaign-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`. As of VOIP-1407, this makes the instance topic-ONLY: every event is published exclusively to the global topic exchange `bin-manager.event` with the routing key `campaign-manager.<resource>.<campaign-id>.<action>` — the per-service fanout exchange `bin-manager.campaign-manager.event` is no longer declared or published to (its removal from the broker is a separate, post-deployment operational runbook, see `docs/reference/rabbitmq-queues-reference.md`). A topic publish failure now propagates to the caller (previously logged and swallowed) since this is the sole delivery path, with no fanout publish left to fall back on. See [docs/domain.md](domain.md) for the per-event routing keys and the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema.

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.campaign-manager.request`. The `listenhandler` matches each request's URI against regex patterns and dispatches to the appropriate handler function.

| Route Pattern | Method | Description |
|---------------|--------|-------------|
| `/v1/campaigns$` | POST | Create a new campaign |
| `/v1/campaigns\?` | GET | List campaigns with filters/pagination |
| `/v1/campaigns/{{UUID}}$` | GET/PUT/DELETE | Get, update, or delete a campaign |
| `/v1/campaigns/{{UUID}}/execute$` | POST | Execute (start dialing) a campaign |
| `/v1/campaigns/{{UUID}}/status$` | PUT | Update campaign status (run/stop/stopping) |
| `/v1/campaigns/{{UUID}}/service_level$` | PUT | Update campaign service level throttle |
| `/v1/campaigns/{{UUID}}/actions$` | GET/PUT | Get or update campaign actions (flow actions to run on connect) |
| `/v1/campaigns/{{UUID}}/resource_info$` | GET | Get resource usage info for a campaign |
| `/v1/campaigns/{{UUID}}/next_campaign_id$` | PUT | Set the next campaign to run after this one completes |
| `/v1/campaigncalls\?` | GET | List campaigncalls with filters/pagination |
| `/v1/campaigncalls/{{UUID}}$` | GET/DELETE | Get or delete a campaigncall |
| `/v1/outplans$` | POST | Create a new outplan |
| `/v1/outplans\?` | GET | List outplans with filters/pagination |
| `/v1/outplans/{{UUID}}$` | GET/PUT/DELETE | Get, update, or delete an outplan |
| `/v1/outplans/{{UUID}}/dials$` | GET/POST | List or add dial entries to an outplan |
