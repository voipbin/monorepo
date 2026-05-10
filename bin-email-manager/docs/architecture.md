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

## Request Routing

The `listenhandler` consumes from queue `QueueNameEmailRequest` and dispatches by regex-matching the request URI:

| Method | URI Pattern | Handler |
|--------|------------|---------|
| POST | `/v1/emails` | Create and send email |
| GET | `/v1/emails?` | List emails (pagination) |
| GET | `/v1/emails/{uuid}` | Get email details |
| DELETE | `/v1/emails/{uuid}` | Delete email |
| POST | `/v1/hooks` | Process provider delivery webhook (SendGrid or Mailgun) |
