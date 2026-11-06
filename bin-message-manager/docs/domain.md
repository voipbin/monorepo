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

SMS: the send logic starts with MessageBird and goes through the SMS provider handlers; see `pkg/messagehandler/send.go`.
MMS: Telnyx is the first provider and MessageBird is used only for the targets Telnyx definitively refused (see the MMS Send Flow below).
The provider identifier is stored on the Message record for webhook routing.

### Webhook Processing

Provider webhooks arrive at `POST /v1/hooks` via `bin-hook-manager`. The URI suffix identifies the provider:
- URI ending in `/telnyx` → Telnyx webhook handler
- URI ending in `/messagebird` → MessageBird webhook handler

Webhooks update the relevant Target's `status` field.

### Inbound Message and Inbound MMS

A Telnyx `message.received` event creates an inbound message at once (`direction` `inbound`, targets in status `received`). The type is `mms` when the provider says `MMS` or the payload lists media, otherwise `sms`; inbound messages are charged through the same `message_created` event as outbound ones (an inbound `mms` at the MMS rate).

The media of an inbound MMS is not downloaded inside the hook: the hook must answer within the RPC timeout (the caller would otherwise answer the provider with an error and the provider delivers the event again). After the message is created, `hookTelnyx` starts one background ingest (`ingestInboundMedias` in `pkg/messagehandler/inbound_media.go`):

1. The payload `media` list is decoded one item at a time (`telnyx.MessageEvent.GetMedias`); an unreadable item is skipped and counted, it never fails the event.
2. Each media is fetched by the `MediaFetcher` (`media_fetcher.go`): https only, host on the allow list `mediaAllowedHostSuffixes` (assumption: `telnyx.com`), the dialled address must be a public address (checked in the dialer, after name resolution), no redirects, no credentials, no compression, content type from an allow list, at most 2 MiB per file, 5 MiB and 10 files per message, 20 seconds per file and 90 seconds for the whole ingest.
3. Each downloaded file is stored through `StorageV1FileCreateContent` (storage-manager `POST /v1/files/content`, file type `media`, generated file name).
4. The stored file ids are written to `medias` once (`dbUpdateMedias`) and `message_updated` is published. When nothing was stored there is no write.

At most two ingests run at the same time in one process; when both slots are taken the message keeps an empty `medias` (result `busy`). The `message_created` event of an inbound MMS carries an empty `medias`; the final list comes with `message_updated`.

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

### MMS Send Flow

`POST /v1/messages/mms` (`messagehandler.SendMMS`) sends a message with media attachments:

1. Identity verification gate and address normalization, the same as SMS.
2. Text length check (2000 characters) and media validation: every media is a storage file id of the same customer, uploaded with file type `media`, with a supported extension and within the size limits. All checks run before the message is created.
3. Balance check with the `mms` billing reference type.
4. A fresh signed download URL is requested for every media. The URLs are handed to the provider only; they are not stored or logged.
5. The message is created with type `mms` and the file ids in `medias`, then dispatched in a goroutine.
6. Every target is sent to Telnyx once. A target that Telnyx definitively refused (HTTP 4xx other than 408), with a `+1` source and destination, is sent once to MessageBird. A target with an unknown result (timeout, 5xx, transport error) is never sent through another provider, so it cannot be delivered twice. A target that no provider accepted becomes `failed`; an MMS is never sent as text only.
7. The final target states and the provider that accepted the targets are written once.
