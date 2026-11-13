# bin-hook-manager Domain Model

## Purpose

`bin-hook-manager` is a **thin proxy** — it has minimal domain logic. Its role is to:
1. Reject requests that can not be genuine (for Paddle: a missing `Paddle-Signature` header or an oversized body). Signatures are verified by the receiving service, not here
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

## Paddle Webhook Forwarding

`bin-hook-manager` handles billing webhooks from Paddle (payment processor). It does not verify the signature and holds no Paddle secret. It rejects a body larger than 1 MiB and a request without the `Paddle-Signature` header (400), and forwards the header unchanged to billing as `received_signature`. The billing service verifies the signature with the secret stored for it and is the only place that does. A 401 answer from billing is returned to Paddle as 400; every other billing error (including billing answering 500 while no secret is stored) stays 500. The deployment order and the rollback rule are in [operations.md](operations.md).

## Hook Payload Model

`models/hook/` contains simple data structures representing the webhook envelope:
- Raw body passthrough with minimal transformation
- No business logic or validation beyond the body size and the signature header check

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
