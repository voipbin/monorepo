# bin-customer-manager — Architecture

## Component Overview

`bin-customer-manager` is a Class A Standard Go RPC Manager and the **foundational identity service** for the VoIPbin platform. It manages tenant organizations (customers) and their API credentials (access keys). Almost every other service depends on customer context for authorization and isolation — this service is the root of the tenant hierarchy.

```
cmd/customer-manager/main.go
    ├── pkg/cachehandler      (Redis — cache-first reads)
    ├── pkg/dbhandler         (MySQL — parameterized raw SQL)
    ├── pkg/customerhandler   (Customer CRUD, cross-service validation, email verification)
    ├── pkg/accesskeyhandler  (Access key CRUD)
    └── pkg/listenhandler     (RabbitMQ RPC — customers & accesskeys API)
```

Supporting binary:
- `cmd/customer-control/` — CLI tool for direct DB/cache operations, bypasses RabbitMQ RPC.

**Foundational role:** `bin-customer-manager` has no SubscribeHandler — it does not consume events from other services. All state changes are driven by inbound RPC requests. Because every other service scopes resources by `customer_id`, this service must be healthy for the platform to function.

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/customer-manager` | Cobra + Viper config, dependency wiring, daemon start |
| Listen | `pkg/listenhandler` | RabbitMQ RPC routing; dispatches to customerhandler / accesskeyhandler |
| Business | `pkg/customerhandler` | Customer lifecycle, email verification, freeze/recover, cross-service validation |
| Business | `pkg/accesskeyhandler` | Access key creation, rotation, deletion |
| Data | `pkg/dbhandler` | Parameterized raw SQL queries (no query builder) |
| Cache | `pkg/cachehandler` | Redis cache-first reads; invalidated on mutations |
| Models | `models/customer` | Customer, WebhookMessage, validation |
| Models | `models/accesskey` | AccessKey, WebhookMessage |

## Request Routing

The `listenhandler` consumes from queue `bin-manager.customer-manager.request` and dispatches by regex-matching the request URI:

| Method | URI Pattern | Handler |
|--------|------------|---------|
| POST | `/v1/customers` | Create customer |
| GET | `/v1/customers?` | List customers (cursor pagination) |
| GET | `/v1/customers/{uuid}` | Get customer |
| PUT | `/v1/customers/{uuid}` | Update customer |
| DELETE | `/v1/customers/{uuid}` | Delete customer |
| PUT | `/v1/customers/{uuid}/billing_account_id` | Link billing account |
| PUT | `/v1/customers/{uuid}/metadata` | Update metadata |
| POST | `/v1/customers/signup` | New customer self-registration |
| POST | `/v1/customers/email_verify` | Email verification callback |
| POST | `/v1/customers/cleanup_unverified` | Sweep endpoint: expire unverified customers past `unverifiedMaxAge` (invoked by schedule-manager cron, replaces the old in-process ticker) |
| POST | `/v1/customers/cleanup_frozen_expired` | Sweep endpoint: anonymize PII and publish `customer_deleted` for frozen customers past the grace period (invoked by schedule-manager cron, replaces the old in-process ticker) |
| POST | `/v1/customers/{uuid}/freeze` | Freeze customer account |
| POST | `/v1/customers/{uuid}/recover` | Recover frozen customer |
| POST | `/v1/customers/{uuid}/freeze_and_delete` | Freeze and schedule deletion |
| POST | `/v1/accesskeys` | Create access key |
| GET | `/v1/accesskeys?` | List access keys |
| GET | `/v1/accesskeys/{uuid}` | Get access key |
| PUT | `/v1/accesskeys/{uuid}` | Update access key |
| DELETE | `/v1/accesskeys/{uuid}` | Delete access key |

## Events Published

Exchange: the global topic exchange `bin-manager.event`, routing key `customer-manager.<resource>.<subscription-id>.<action>`.

`cmd/customer-manager` and BOTH NotifyHandler instances inside `cmd/customer-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`. **As of VOIP-1407, this is the sole publish path** — the previous per-service fanout exchange `bin-manager.customer-manager.event` is no longer published to, and (per the operational runbook in `docs/reference/rabbitmq-queues-reference.md`) will eventually be deleted from the broker. No code in this service changed for VOIP-1407; the behavior change (dual publish → topic-only) comes entirely from `bin-common-handler/pkg/notifyhandler`'s shared library update. All three construction sites must stay in lockstep on this option — enabling it in only some would leave consumers with gaps depending on which process published. A topic publish failure now propagates to the caller as an error (previously it was swallowed silently).

Both published resources are addressed by their OWN id (the default top-level `id` fallback; no `eventtopic.SubscriptionIdentifier` override exists in this service). An accesskey is an independent persistent resource and is NOT addressed by its owning customer. Consumers bind `customer-manager.customer.<customer-id>.#` and `customer-manager.accesskey.<accesskey-id>.#` as two independent streams. The exact keys are pinned by `models/customer/routingkey_golden_test.go`; the schema lives in the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md`.

**CRITICAL — do not add an `EventSubscriptionID()` override to `*customer.Customer`.** `customer.CustomerCreatedEvent` anonymously embeds `*Customer`, so any method on `*Customer` is promoted to the wrapper as well: an override would silently re-address BOTH types at once. The wrapper works correctly today precisely because the embed's `id` is promoted into its JSON, which the default fallback picks up. A wrapper with a nil embed marshals without an `id` and correctly collapses to the `-` placeholder segment. All three facts are asserted in the golden test.

| Event type | Trigger | Payload type |
|-----------|---------|--------------|
| `customer.EventTypeCustomerCreated` | Customer created (admin create or self-signup) | `*customer.CustomerCreatedEvent` (wraps `*Customer` + `headless`) |
| `customer.EventTypeCustomerUpdated` | Customer info, billing account, or webhook config updated | `*customer.Customer` |
| `customer.EventTypeCustomerDeleted` | Customer deleted, or frozen customer past the grace period | `*customer.Customer` |
| `customer.EventTypeCustomerFrozen` | Customer account frozen | `*customer.Customer` |
| `customer.EventTypeCustomerRecovered` | Frozen customer recovered | `*customer.Customer` |
| `customer.EventTypeCustomerIdentityVerificationUpdated` | Identity verification status changed | `*customer.Customer` |
| `accesskey.EventTypeAccesskeyCreated` | Access key created | `*accesskey.Accesskey` |
| `accesskey.EventTypeAccesskeyUpdated` | Access key updated | `*accesskey.Accesskey` |
| `accesskey.EventTypeAccesskeyDeleted` | Access key deleted | `*accesskey.Accesskey` |
