# bin-outdial-manager — Domain

## Domain Entities

### Outdial

The top-level container for an outbound dialing campaign batch.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `campaign_id` | UUID | Associated campaign (nullable) |
| `name` | string | Human-readable label |
| `detail` | string | Description |
| `data` | JSON | Custom data payload (arbitrary JSON) |
| `tm_create` | timestamp | |
| `tm_update` | timestamp | |
| `tm_delete` | timestamp | Soft-delete; active = `9999-01-01` |

An outdial is a batch of targets. It belongs to a customer and optionally to a campaign.

### OutdialTarget

An individual call target within an outdial. Supports up to 5 destination numbers.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `outdial_id` | UUID | Parent outdial |
| `name` | string | Label for target |
| `detail` | string | Description |
| `data` | JSON | Custom data (passed to call handler) |
| `destination_0` – `destination_4` | string | Up to 5 E.164/SIP destinations |
| `try_count_0` – `try_count_4` | int | Attempt count per destination |
| `status` | enum | `idle` / `processing` / `done` |
| `tm_create` | timestamp | |
| `tm_update` | timestamp | |
| `tm_delete` | timestamp | Soft-delete sentinel |

### OutdialTargetCall

A single call attempt record linked to an OutdialTarget.

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | Primary key |
| `outdialtarget_id` | UUID | Parent target |
| `call_id` | UUID | Associated call from bin-call-manager |
| `destination` | string | Actual dialed destination for this attempt |
| `try_count` | int | Try index used |
| `tm_create` | timestamp | |
| `tm_end` | timestamp | When the call attempt concluded |

## Key Business Rules

1. **Multi-destination retry**: Each `OutdialTarget` carries up to 5 destination addresses (`destination_0`–`destination_4`) with independent attempt counts (`try_count_0`–`try_count_4`). The `available` endpoint accepts per-destination try-count thresholds so the campaign manager can request only targets eligible for a specific retry level.

2. **Target status machine**:
   - `idle` — target has not been picked up
   - `processing` — target is currently being dialed (locked via `POST /progressing`)
   - `done` — all attempts exhausted or target manually completed

3. **Available query**: `GET /v1/outdials/{id}/available?try_count_0=N&...&limit=N` returns targets whose per-destination try counts are below the given thresholds. Used exclusively by `bin-campaign-manager` to claim work.

4. **Soft deletes**: `tm_delete` sentinel `9999-01-01 00:00:00.000000` marks active records. All active-record queries include this filter.

5. **Custom data**: Both `Outdial.data` and `OutdialTarget.data` accept arbitrary JSON. The platform passes this data through to downstream call handlers, enabling per-target scripting or metadata.

6. **Campaign association**: An outdial may be linked to a campaign via `campaign_id`. This is updated separately via `PUT /v1/outdials/{id}/campaign_id` and does not affect target status.

7. **Event publishing**: The service publishes `outdial_created`, `outdial_updated`, and `outdial_deleted` events to `QueueNameOutdialEvent`. No events are published for individual targets.

8. **No event subscriptions**: This service has no `subscribehandler`. It does not listen to events from other services.
