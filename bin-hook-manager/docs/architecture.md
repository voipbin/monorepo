# bin-hook-manager Architecture

## Component Overview

`bin-hook-manager` is a public-facing HTTPS webhook gateway. It receives inbound HTTP webhook requests from external platforms and forwards them as RabbitMQ messages to internal VoIPbin services. There is no inbound RPC queue — it exposes HTTP, not RabbitMQ.

```
cmd/hook-manager/main.go
    ├── SSL bootstrap (base64 cert/key → /tmp/)
    ├── Gin HTTP server (:80 HTTP + :443 HTTPS)
    │     ├── api/v1.0/emails/         (email webhook endpoint)
    │     ├── api/v1.0/messages/       (SMS/MMS webhook endpoint)
    │     └── api/v1.0/conversation/   (conversation webhook endpoint)
    └── pkg/servicehandler/            (RabbitMQ publish abstraction)
          └── requesthandler (bin-common-handler) → RabbitMQ
```

**Supporting binaries:**
- `cmd/hook-control/` — CLI for testing webhook delivery

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| HTTP | `api/v1.0/*/` | Gin route handlers; parse incoming webhook payloads |
| Service | `pkg/servicehandler` | Interface abstracting RabbitMQ publish; one method per webhook type |
| Transport | requesthandler (bin-common-handler) | Publishes messages to destination service queues |
| Data | `models/hook/` | Simple webhook payload data structures |

No SubscribeHandler. No ListenHandler. No database reads/writes on the request path.

## Execution Model

`bin-hook-manager` does **not** use the standard RabbitMQ RPC pattern. It is an HTTP-to-RabbitMQ bridge:

```
External platform
    │ HTTPS POST /v1.0/<resource>
    ▼
Gin HTTP router (port 443 / 80)
    │ parse payload
    ▼
servicehandler.SendEmail/SendMessage/SendConversation()
    │ publish to RabbitMQ
    ▼
Destination service queue (email-manager / message-manager / conversation-manager)
    │ consume and process
    ▼
Internal service processes the event
```

Routing is by URL path — not by RabbitMQ queue pattern:

| HTTP path | Forwards to |
|-----------|-------------|
| `POST /v1.0/emails` | email-manager |
| `POST /v1.0/messages` | message-manager |
| `POST /v1.0/conversation` | conversation-manager |
| `GET /ping` | health check (no RabbitMQ) |

CORS is configured to allow all origins (`AllowOrigins: ["*"]`) — appropriate for a public webhook receiver.

## SSL

Certificates are passed via `SSL_CERT_BASE64` and `SSL_PRIVKEY_BASE64` environment variables, decoded at startup, and written to `/tmp/`. The service starts both HTTP (:80) and HTTPS (:443) listeners simultaneously.

## Event Subscriptions

This service does **not** subscribe to RabbitMQ events. There is no SubscribeHandler.

## Events Published

None. This service publishes messages to destination service request queues, not to event exchanges.
