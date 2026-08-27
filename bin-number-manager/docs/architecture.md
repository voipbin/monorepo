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
| Subscribe | `pkg/subscribehandler` | Consumes customer/flow events; cascading deletes and flow-ref cleanup |
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

Both `cmd/number-manager` and `cmd/number-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange `bin-manager.number-manager.event` (unchanged, still the system of record) and once to the global topic exchange `bin-manager.event` with the routing key `number-manager.number.<number-id>.<action>`. The two cmds must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure never propagates to the caller and never affects the fanout publish.

`numberhandler.dbUpdate` picks its publish call from the event type: `number_renewed` goes out through `PublishEvent` (internal bookkeeping, no customer webhook) while `number_updated` goes out through `PublishWebhookEvent`. Both paths generate the same topic routing-key shape with the same `*number.Number` payload, so one instance binding `number-manager.number.<number-id>.#` follows the whole lifecycle.

The third routing-key segment is the *subscription address* — the id consumers bind to. `number.Number` deliberately carries no `eventtopic.SubscriptionIdentifier` override: a number is an independent persistent resource, so its own id is the address and the default JSON `id` extraction covers it. `models/number/routingkey_golden_test.go` pins the exact key of every published event type (`number_created`, `number_deleted`, and both `dbUpdate` branches) plus the deliberate absence of that override. See the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema and `docs/plans/2026-08-27-voip-1405-topic-publisher-rollout-design.md` §2.4 for the address mapping.

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
