# bin-number-manager — Architecture

## Component Overview

`bin-number-manager` is a Class A Standard Go RPC Manager responsible for PSTN phone number lifecycle management. It purchases, manages, and releases DID (Direct Inward Dial) phone numbers through external providers (Telnyx, Twilio) and routes inbound calls/messages to flows.

```
cmd/number-manager/main.go
    ├── pkg/cachehandler            (Redis — number lookups)
    ├── pkg/dbhandler               (MySQL — number records, provider mappings)
    ├── pkg/requestexternal         (HTTP clients for Telnyx/Twilio APIs)
    ├── pkg/numberhandlertelnyx     (Telnyx provider implementation)
    ├── pkg/numberhandlertwilio     (Twilio provider implementation)
    ├── pkg/numberhandler           (Core business logic, provider dispatch)
    ├── pkg/listenhandler           (RabbitMQ RPC — numbers & available_numbers API)
    └── pkg/subscribehandler        (RabbitMQ event consumer — cascading deletions)
```

Supporting binary:
- `cmd/number-control/` — CLI for direct DB/cache operations, bypasses RabbitMQ RPC.

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/number-manager` | Cobra + Viper config, dependency wiring, daemon start |
| Listen | `pkg/listenhandler` | RabbitMQ RPC routing; dispatches to numberhandler |
| Subscribe | `pkg/subscribehandler` | Consumes customer/flow events via pattern bindings on the global topic exchange `bin-manager.event` (sole intake mechanism since VOIP-1407); cascading deletes and flow-ref cleanup |
| Business | `pkg/numberhandler` | Number CRUD, provider dispatch, billing validation |
| Provider | `pkg/numberhandlertelnyx` | Telnyx API: purchase, release, list available numbers |
| Provider | `pkg/numberhandlertwilio` | Twilio API: purchase, release, list available numbers |
| External | `pkg/requestexternal` | HTTP client wrapper for provider APIs |
| Data | `pkg/dbhandler` | MySQL queries for numbers and provider mappings |
| Cache | `pkg/cachehandler` | Redis cache for number lookups |
| Models | `models/number` | Number, Status, EventType |
| Models | `models/availablenumber` | AvailableNumber (search results from providers) |
| Models | `models/providernumber` | ProviderNumber (internal provider-to-number mapping) |

## Event Publishing

Both `cmd/number-manager` and `cmd/number-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`. **As of VOIP-1407, this is the sole publish path** — the previous per-service fanout exchange `bin-manager.number-manager.event` is no longer published to, and (per the operational runbook in `docs/reference/rabbitmq-queues-reference.md`) will eventually be deleted from the broker. Events publish to the global topic exchange `bin-manager.event` with the routing key `number-manager.number.<number-id>.<action>`. This service's publish-side behavior change comes entirely from `bin-common-handler/pkg/notifyhandler`'s shared library update (its own consumer-side subscribehandler code also changed separately for VOIP-1407, see the Event Subscriptions section below). The two cmds must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure now propagates to the caller as an error (previously it was swallowed silently).

`numberhandler.dbUpdate` picks its publish call from the event type: `number_renewed` goes out through `PublishEvent` (internal bookkeeping, no customer webhook) while `number_updated` goes out through `PublishWebhookEvent`. Both paths generate the same topic routing-key shape with the same `*number.Number` payload, so one instance binding `number-manager.number.<number-id>.#` follows the whole lifecycle.

The third routing-key segment is the *subscription address* — the id consumers bind to. `number.Number` deliberately carries no `eventtopic.SubscriptionIdentifier` override: a number is an independent persistent resource, so its own id is the address and the default JSON `id` extraction covers it. `models/number/routingkey_golden_test.go` pins the exact key of every published event type (`number_created`, `number_deleted`, and both `dbUpdate` branches) plus the deliberate absence of that override. See the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema and `docs/plans/2026-08-27-voip-1405-topic-publisher-rollout-design.md` §2.4 for the address mapping.

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes from the queue `bin-manager.number-manager.subscribe`. Since VOIP-1406 the queue is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 2 patterns total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `customer-manager.customer.*.deleted` | Customer deletion — releases all of the customer's numbers back to the provider |
| `flow-manager.flow.*.deleted` | Flow deletion — clears the deleted flow's references from numbers |

As of VOIP-1407 this topic-pattern binding is the **sole intake mechanism**; the old per-service fanout subscriptions (`QueueSubscribe` to `bin-manager.flow-manager.event` and `bin-manager.customer-manager.event`) have been removed from `Run()` entirely, along with the fanout-unbind step that used to follow a successful topic bind.

## Request Routing

The `listenhandler` consumes from queue `bin-manager.number-manager.request` and dispatches by regex-matching the request URI:

| Method | URI Pattern | Handler |
|--------|------------|---------|
| GET | `/v1/available_numbers` | Search provider for purchasable numbers by country code |
| GET | `/v1/numbers?` | List owned numbers (pagination) |
| POST | `/v1/numbers` | Purchase (create) number from provider |
| GET | `/v1/numbers/{uuid}` | Get number details |
| PUT | `/v1/numbers/{uuid}` | Update number (flow IDs, status, name) |
| DELETE | `/v1/numbers/{uuid}` | Release number back to provider |
| PUT | `/v1/numbers/{uuid}/flow_ids` | Update call/message flow associations |
| PUT | `/v1/numbers/{uuid}/metadata` | Update metadata |
| POST | `/v1/numbers/renew` | Renew number subscription |
| GET | `/v1/numbers/count_virtual_by_customer` | Count virtual numbers for a customer |
