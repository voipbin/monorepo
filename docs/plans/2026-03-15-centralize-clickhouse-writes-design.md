# Design: Centralize ClickHouse Writes in Timeline-Manager

**Date:** 2026-03-15
**Branch:** NOJIRA-Centralize-clickhouse-writes-in-timeline-manager

## Problem

All 27 services maintain their own ClickHouse connection via `notifyhandler` in `bin-common-handler`. When an event is published, it goes to both RabbitMQ (sync) and ClickHouse (async fire-and-forget). This causes:

- **Operational burden:** Changing the ClickHouse URL requires redeploying all 27 services.
- **Resource waste:** 27 services × ClickHouse connections, each with a background retry loop.
- **Tight coupling:** Every service depends on `clickhouse-go/v2` transitively through `bin-common-handler`, even though ClickHouse is a secondary analytics store.

ClickHouse is only read by `bin-timeline-manager`. No other service queries it.

## Solution

Make `bin-timeline-manager` the sole ClickHouse writer. It subscribes to all service event exchanges via RabbitMQ and inserts events into ClickHouse. Remove all ClickHouse code from `bin-common-handler/pkg/notifyhandler` and all services.

## Data Flow

### Before

```
Service → RabbitMQ event exchange (sync) + ClickHouse (async, 27 connections)
Timeline-manager → reads ClickHouse
```

### After

```
Service → RabbitMQ event exchange (sync only)
Timeline-manager → subscribes to all event exchanges via RabbitMQ → writes ClickHouse (1 connection)
Timeline-manager → reads ClickHouse
```

## Changes

### 1. bin-common-handler/pkg/notifyhandler — Remove ClickHouse entirely

- Remove `clickhouseAddress` parameter from `NewNotifyHandler()` (5th param).
- Remove `clickhouseConnectionLoop()`, `newClickHouseClient()`, `publishToClickHouse()`.
- Remove `clickhouseAddress` and `chClient` fields from `notifyHandler` struct.
- Remove `clickhouseRetryInterval` constant.
- Remove `clickhouse-go/v2` import from `main.go` and `publish.go`.
- Remove `clickhouse-go/v2` from `bin-common-handler/go.mod`.
- Regenerate mock: `go generate ./pkg/notifyhandler/...`.

The `NotifyHandler` interface is unchanged — `PublishEvent`, `PublishEventRaw`, `PublishWebhook`, `PublishWebhookEvent` remain as-is. Only the constructor and internal implementation change.

### 2. All 27 services + voip-asterisk-proxy — Mechanical updates

For each service's `cmd/*/main.go` (~55 call sites):
- Remove the last argument (`os.Getenv("CLICKHOUSE_ADDRESS")`) from `NewNotifyHandler()` calls.
- No other code changes needed — the `NotifyHandler` interface is the same.

Note: `bin-talk-manager/cmd/talk-manager/main.go` uses multi-line format with `commonnotify` alias. All others use single-line with `notifyhandler` alias.

K8s deployment changes (optional, can be done later):
- Remove `CLICKHOUSE_ADDRESS` env var from all service deployments except timeline-manager.

### 3. bin-timeline-manager — Add event subscription

Add a `subscribehandler` package following the established pattern (see `bin-webhook-manager/pkg/subscribehandler/`):

- Create `pkg/subscribehandler/main.go`:
  - `NewSubscribeHandler(sockHandler, dbHandler)` constructor.
  - `Run()` method that:
    1. Creates queue `bin-manager.timeline-manager.subscribe` (durable).
    2. Subscribes to all service event exchanges using `QueueSubscribe()`.
    3. Starts consuming events via `ConsumeMessage()`.
  - `processEvent(evt *sock.Event)` that inserts into ClickHouse via dbHandler.

- Subscribe targets (all event exchanges from `commonoutline`):
  ```
  QueueNameAIEvent, QueueNameAgentEvent, QueueNameAsteriskEventAll,
  QueueNameBillingEvent, QueueNameCallEvent, QueueNameCampaignEvent,
  QueueNameConferenceEvent, QueueNameContactEvent, QueueNameConversationEvent,
  QueueNameCustomerEvent, QueueNameEmailEvent, QueueNameFlowEvent,
  QueueNameMessageEvent, QueueNameNumberEvent, QueueNameOutdialEvent,
  QueueNamePipecatEvent, QueueNameQueueEvent, QueueNameRegistrarEvent,
  QueueNameRouteEvent, QueueNameSentinelEvent, QueueNameStorageEvent,
  QueueNameTagEvent, QueueNameTalkEvent, QueueNameTimelineEvent,
  QueueNameTranscribeEvent, QueueNameTransferEvent, QueueNameTTSEvent,
  QueueNameWebhookEvent
  ```

- Add `EventInsert(ctx, event)` to `pkg/dbhandler/main.go` interface:
  - Uses `PrepareBatch()` (not `Exec()`) for DateTime64(3) millisecond precision.
  - Insert columns: `timestamp`, `event_type`, `publisher`, `data_type`, `data`.

- Update `cmd/timeline-manager/main.go`:
  - Create and run `subscribehandler` alongside existing `listenhandler`.

## Behavioral Notes

### Delayed events

Currently, delayed events (`delay > 0`) skip ClickHouse in `publishEvent()`. After this change, delayed events will arrive at the service's event exchange after the RabbitMQ delay expires, and timeline-manager will receive and store them. This is a minor behavior change that results in a more complete event history.

### Resilience improvement

Currently, if a service's ClickHouse connection drops, events are silently lost (fire-and-forget goroutine). After this change, events queue up in RabbitMQ when timeline-manager is down and get processed when it recovers. This is strictly better.

### Multiple replicas

Timeline-manager K8s deployment has 2 replicas. With a shared durable queue, RabbitMQ load-balances events across replicas — each event processed exactly once.

## Trade-offs

| Pro | Con |
|-----|-----|
| Single point of ClickHouse config | Timeline-manager becomes the bottleneck for event ingestion |
| 27 fewer ClickHouse connections | Brief event gap during cutover (acceptable) |
| Simpler services (no ClickHouse awareness) | Delayed events now stored (minor behavior change) |
| Better resilience (RabbitMQ buffering vs fire-and-forget) | |
| Reduced dependency tree (remove clickhouse-go from 27 services) | |

## Scope

- Remove ClickHouse from `bin-common-handler/pkg/notifyhandler`.
- Update all ~55 `NewNotifyHandler()` call sites.
- Add `subscribehandler` + `EventInsert` to `bin-timeline-manager`.
- Run verification workflow on all affected services.
- K8s deployment env var cleanup can be done in a follow-up.
