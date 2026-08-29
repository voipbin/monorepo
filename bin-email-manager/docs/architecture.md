# bin-email-manager — Architecture

## Component Overview

`bin-email-manager` is a Class A Standard Go RPC Manager that handles email delivery through external providers (SendGrid, Mailgun) with automatic failover. It tracks delivery status via provider webhooks and supports file attachments via `bin-storage-manager`.

```
cmd/email-manager/main.go
    ├── pkg/cachehandler      (Redis)
    ├── pkg/dbhandler         (MySQL via Squirrel query builder)
    ├── pkg/emailhandler      (Business logic — create, send, hook)
    │   ├── engine_sendgrid.go   (SendGrid provider implementation)
    │   └── engine_mailgun.go    (Mailgun provider implementation)
    └── pkg/listenhandler     (RabbitMQ RPC — emails & hooks API)
```

Supporting binary:
- `cmd/email-control/` — CLI for direct DB/cache operations, bypasses RabbitMQ RPC.

**No SubscribeHandler.** Provider delivery status events arrive via inbound `POST /v1/hooks` webhooks forwarded through `bin-hook-manager`.

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/email-manager` | pflag + Viper config, dependency wiring, daemon start |
| Listen | `pkg/listenhandler` | RabbitMQ RPC routing; dispatches to emailhandler |
| Business | `pkg/emailhandler` | Email create/send, provider failover, attachment fetch, webhook processing |
| Provider | `pkg/emailhandler/engine_sendgrid.go` | SendGrid API send implementation |
| Provider | `pkg/emailhandler/engine_mailgun.go` | Mailgun API send implementation |
| Data | `pkg/dbhandler` | Squirrel SQL builder queries for email records |
| Cache | `pkg/cachehandler` | Redis caching for email lookups |
| Models | `models/email` | Email, Status, ProviderType, Attachment, WebhookMessage |
| Models | `models/sendgrid` | SendGrid-specific webhook event models |

## Event Publishing

Both `cmd/email-manager` and `cmd/email-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`, so every event is published to the global topic exchange `bin-manager.event` with the routing key `email-manager.email.<email-id>.<action>`. **As of VOIP-1407, this is the sole publish path** — the previous per-service fanout exchange `bin-manager.email-manager.event` is no longer published to, and (per the operational runbook in `docs/reference/rabbitmq-queues-reference.md`) will eventually be deleted from the broker. No code in this service changed for VOIP-1407; the behavior change (dual publish → topic-only) comes entirely from `bin-common-handler/pkg/notifyhandler`'s shared library update. The two cmds must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure now propagates to the caller as an error (previously it was swallowed silently).

The third routing-key segment is the *subscription address* — the id consumers bind to. `email.Email` deliberately carries no `eventtopic.SubscriptionIdentifier` override: an email is an independent persistent resource, so its own id is the address and the default JSON `id` extraction covers it. `models/email/routingkey_golden_test.go` pins the exact key of every published event type (`email_created`, `email_updated`, `email_deleted`) plus the deliberate absence of that override. See the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema and `docs/plans/2026-08-27-voip-1405-topic-publisher-rollout-design.md` §2.4 for the address mapping.

## Request Routing

The `listenhandler` consumes from queue `QueueNameEmailRequest` and dispatches by regex-matching the request URI:

| Method | URI Pattern | Handler |
|--------|------------|---------|
| POST | `/v1/emails` | Create and send email |
| GET | `/v1/emails?` | List emails (pagination) |
| GET | `/v1/emails/{uuid}` | Get email details |
| DELETE | `/v1/emails/{uuid}` | Delete email |
| POST | `/v1/hooks` | Process provider delivery webhook (SendGrid or Mailgun) |
