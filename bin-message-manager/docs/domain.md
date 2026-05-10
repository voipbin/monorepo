# bin-message-manager — Domain

## Domain Entities

### Message

An SMS message with source/destination addresses and per-target delivery tracking.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `source` | string | Source phone number (E.164) |
| `destinations` | []string | Destination phone numbers (E.164) |
| `text` | string | Message body |
| `type` | string | Message type |
| `direction` | string | `outbound` or `inbound` |
| `provider_name` | string | `telnyx` or `messagebird` |
| `tm_delete` | timestamp | Soft-delete sentinel (`9999-01-01` = active) |

### Target

An individual recipient within a message. Each destination address maps to one Target.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `message_id` | UUID | Parent message |
| `destination` | string | Recipient phone number (E.164) |
| `status` | string | `queued`, `sent`, `delivered`, `failed` |
| `provider_message_id` | string | Provider's own message reference ID |

## Key Business Rules

### Send Flow

1. `messageHandler.Send()` validates customer billing balance via RPC to `bin-billing-manager`.
2. Message record created in database with status `queued` for all targets.
3. `message.EventTypeMessageCreated` published to notify billing-manager.
4. Provider handler runs asynchronously in a goroutine (non-blocking to the caller).
5. Each target dispatched to provider API concurrently via `sync.WaitGroup`.
6. Target statuses updated in database after provider response.
7. Prometheus counters incremented per provider.

### Provider Selection

Currently defaults to MessageBird; Telnyx is the fallback. Provider is hardcoded in the send logic:
```
messagebird → primary
telnyx      → fallback
```
Provider identifier is stored on the Message record for webhook routing.

### Webhook Processing

Provider webhooks arrive at `POST /v1/hooks` via `bin-hook-manager`. The URI suffix identifies the provider:
- URI ending in `/telnyx` → Telnyx webhook handler
- URI ending in `/messagebird` → MessageBird webhook handler

Webhooks update the relevant Target's `status` field.

### Soft Deletes

Messages and Targets use `tm_delete` for soft deletes. Default active sentinel: `9999-01-01 00:00:00.000000`.

### Event Publishing

| Event | Trigger |
|-------|---------|
| `message.EventTypeMessageCreated` | Message created |
| `message.EventTypeMessageDeleted` | Message soft-deleted |
| `message.EventTypeMessageUpdated` | Message or target status updated |

Events are consumed by `bin-billing-manager` to charge accounts for SMS usage.

### Database Queries

Uses Squirrel SQL builder (not raw SQL):
```go
sq.Select("*").From("messages").Where(sq.Eq{"id": id}).Where(sq.Eq{"tm_delete": DefaultTimeStamp})
```
