# bin-hook-manager Domain Model

## Purpose

`bin-hook-manager` is a **thin proxy** — it has minimal domain logic. Its role is to:
1. Authenticate/verify incoming webhooks (where platform requires it, e.g. Paddle signature)
2. Transform HTTP payloads into RabbitMQ messages
3. Route to the correct internal service based on URL path

## Webhook Types

### Email Webhooks (`/v1.0/emails`)
Inbound email event notifications (e.g. delivery status, bounce, inbound email).
Forwarded to: `email-manager`

### Message Webhooks (`/v1.0/messages`)
Inbound SMS/MMS event notifications from carrier/aggregator.
Forwarded to: `message-manager`

### Conversation Webhooks (`/v1.0/conversation`)
Inbound conversation events (LINE platform webhooks, etc.).
Forwarded to: `conversation-manager`

## Paddle Webhook Verification

`bin-hook-manager` handles billing webhooks from Paddle (payment processor). Paddle webhooks are verified using `paddle_webhook_secret_key` before forwarding. This is the primary reason this service needs a secret key config.

## Hook Payload Model

`models/hook/` contains simple data structures representing the webhook envelope:
- Raw body passthrough with minimal transformation
- No business logic or validation beyond signature verification

## Service Handler Interface

`pkg/servicehandler/` defines:

```go
type ServiceHandler interface {
    SendEmail(ctx, payload) error
    SendMessage(ctx, payload) error
    SendConversation(ctx, payload) error
}
```

Each method publishes the payload to the appropriate internal service queue via `requesthandler` from `bin-common-handler`. The interface is mocked in tests to verify correct dispatch.

## Why a Separate Gateway Service?

- **SSL termination at the application layer**: the gateway holds SSL certificates and handles HTTPS; internal services use plain RabbitMQ
- **Public-facing isolation**: limits external attack surface to a single thin service
- **Protocol translation**: external world speaks HTTPS; internal services speak RabbitMQ RPC
