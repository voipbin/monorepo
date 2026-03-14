# Design: Aggregated Events Endpoint

## Problem Statement

Currently, the timeline API (`GET /v1/{resource_type}/{resource_id}/timeline`) returns events for a single resource type — e.g., all `call_*` events for one call, or all `activeflow_*` events for one activeflow. There is no way to get a unified view of everything that happened during an activeflow execution: calls, AI calls, recordings, transcriptions, etc.

Customers need a single endpoint that aggregates all events related to an activeflow execution to analyze call flows, debug issues, and understand the full lifecycle of a communication session.

## Approach

Add an aggregated events endpoint that, given an activeflow ID (or call ID), returns all related events in a single chronological list. The activeflow ID is the anchor — all events are correlated through this ID.

### Key Design Decisions

1. **No new service** — implemented as a new handler in `bin-timeline-manager` + new endpoint in `bin-api-manager`
2. **Materialized column in ClickHouse** — add `activeflow_id` as a materialized column extracted from the JSON `data` field. Zero changes to existing services or the `PublishEvent` interface.
3. **Activeflow ID is the key** — all related resource models (Call, AICall, Recording, Transcribe, Pipecatcall, Conferencecall) already carry `activeflow_id` in their WebhookMessage JSON stored in ClickHouse.
4. **API manager resolves call_id** — when queried by `call_id`, the API manager fetches the call, extracts `activeflow_id`, and passes only `activeflow_id` to timeline-manager. Timeline-manager has no dependency on call-manager.
5. **Raw data returned as-is** — ClickHouse stores WebhookMessage JSON (external-safe). No type-specific conversion needed unlike the existing timeline endpoint.
6. **Single activeflow scope** — chained activeflows (via `on_complete_flow_id`) are separate queries. The customer can discover child activeflows via `reference_activeflow_id` and query them individually.

## API

### Endpoint

```
GET /v1/aggregated-events?activeflow_id={uuid}
GET /v1/aggregated-events?call_id={uuid}
```

- Exactly one query parameter required. Both provided or neither → 400.
- Cursor-based pagination (consistent with existing timeline API).
- Events sorted by timestamp DESC (newest first).

### Response

```json
{
  "result": [
    {
      "timestamp": "2026-03-14T10:00:05.123Z",
      "event_type": "aicall_status_progressing",
      "data": {"id": "...", "activeflow_id": "...", "status": "progressing", ...}
    },
    {
      "timestamp": "2026-03-14T10:00:02.000Z",
      "event_type": "activeflow_created",
      "data": {"id": "...", "flow_id": "...", "status": "running", ...}
    },
    {
      "timestamp": "2026-03-14T10:00:01.000Z",
      "event_type": "call_created",
      "data": {"id": "...", "activeflow_id": "...", "from": {...}, "to": {...}, ...}
    }
  ],
  "next_page_token": "2026-03-14T10:00:01.000Z"
}
```

Fields per event: `timestamp`, `event_type`, `data`. Consistent with the existing timeline response format.

### Error Handling

- Both or neither query params → 400 Bad Request
- Activeflow/call not found → 404 Not Found
- Call has no activeflow → 404 Not Found
- No events found → 200 with empty `result: []`

## ClickHouse Changes

### Migration: Add `activeflow_id` materialized column

```sql
ALTER TABLE events
ADD COLUMN IF NOT EXISTS activeflow_id String
MATERIALIZED if(
    event_type LIKE 'activeflow_%',
    JSONExtractString(data, 'id'),
    JSONExtractString(data, 'activeflow_id')
);
```

**Why the compound expression:**
- Activeflow events store their own ID as `"id"` in the data JSON (not `"activeflow_id"`)
- All other resource events (call, aicall, recording, etc.) store the activeflow reference as `"activeflow_id"`
- The `if()` expression handles both cases

**Notes:**
- Materialized columns are computed on INSERT — only new events will have this column populated
- Historical events will not be queryable through this endpoint (accepted trade-off)
- No secondary index added in v1 — can be added later if performance requires it
- Follows the existing pattern of the `resource_id` materialized column

### Query

```sql
SELECT timestamp, event_type, publisher, data_type, data
FROM events
WHERE activeflow_id = ?
  AND timestamp < ?  -- pagination cursor (if present)
ORDER BY timestamp DESC
LIMIT ?
```

## Timeline Manager Changes

### New RPC: `TimelineV1AggregatedEventList`

**Request:**
- `ActiveflowID` (uuid.UUID, required)
- `PageToken` (string, optional — timestamp cursor)
- `PageSize` (int, optional — default 100, max 1000)

**Response:**
- `Result` ([]*Event — timestamp, event_type, publisher, data_type, data)
- `NextPageToken` (string, omitted if no more results)

### New Files/Changes
- `pkg/dbhandler/event.go` — add `AggregatedEventList()` query function
- `pkg/eventhandler/event.go` — add `AggregatedList()` business logic
- `pkg/listenhandler/` — add RPC handler for the new endpoint
- `models/` — add request/response structs for the new RPC

## API Manager Changes

### New Endpoint Handler

Location: `bin-api-manager/pkg/servicehandler/`

**Flow for `?activeflow_id={id}`:**
1. Parse `activeflow_id` from query params
2. Fetch activeflow from flow-manager (permission check — verify customer owns it)
3. Call `TimelineV1AggregatedEventList` with `activeflow_id`
4. Return raw events as-is (data is already WebhookMessage JSON)

**Flow for `?call_id={id}`:**
1. Parse `call_id` from query params
2. Fetch call from call-manager (permission check — verify customer owns it)
3. Extract `activeflow_id` from the call
4. If `activeflow_id` is empty → return 404
5. Call `TimelineV1AggregatedEventList` with `activeflow_id`
6. Return raw events as-is

### OpenAPI Changes
- Add `GET /v1/aggregated-events` endpoint definition in `bin-openapi-manager/openapi/openapi.yaml`
- Define query parameters (`activeflow_id`, `call_id`)
- Define response schema (reuse existing timeline event format)
- Regenerate types in `bin-openapi-manager` and server code in `bin-api-manager`

## Event Coverage

### v1 — Top-level resources (have `activeflow_id` AND `ConvertWebhookMessage()`)

| Resource | Events | Service |
|----------|--------|---------|
| Call | `call_created`, `call_updated`, `call_dialing`, `call_ringing`, `call_hangup`, etc. | call-manager |
| Activeflow | `activeflow_created`, `activeflow_updated`, `activeflow_deleted` | flow-manager |
| AICall | `aicall_created`, `aicall_started`, `aicall_progressing`, `aicall_terminated`, etc. | ai-manager |
| Recording | `recording_created`, `recording_started`, `recording_stopped`, `recording_ended` | call-manager |
| Transcribe | `transcribe_created`, `transcribe_progressing`, `transcribe_done`, `transcribe_deleted` | transcribe-manager |
| Conferencecall | events from conference-manager | conference-manager |
| Campaigncall | `campaigncall_created`, `campaigncall_updated`, `campaigncall_deleted` | campaign-manager |
| Summary | `summary_created`, `summary_updated`, `summary_deleted` | ai-manager |

### Future — Resources with `activeflow_id` but no `ConvertWebhookMessage()`

These resources publish events with `activeflow_id` in their data, so ClickHouse will correlate them.
However, they lack a `ConvertWebhookMessage()` method, so the aggregated endpoint cannot safely
return them without potentially leaking internal fields. They are silently skipped in v1.
To add support: create `webhook.go` with `WebhookMessage` struct and `ConvertWebhookMessage()`,
then add a case to `convertAggregatedEventData()` in `bin-api-manager`.

| Resource | Events | Service | Missing |
|----------|--------|---------|---------|
| Confbridge | `confbridge_created`, `confbridge_deleted`, etc. | call-manager | No `webhook.go` |
| Pipecatcall | `pipecatcall_initialized`, `pipecatcall_started`, etc. | pipecat-manager | No `webhook.go` |
| Email | `email_created`, `email_updated`, etc. | email-manager | No `webhook.go` |

### Future — Sub-resources (do NOT have `activeflow_id` yet)

These can be added incrementally by adding `activeflow_id` to their model structs:

| Resource | Events | Parent |
|----------|--------|--------|
| DTMF | `dtmf_received` | Call |
| AIMessage | `aimessage_created` | AICall |
| Speaking | `speaking_started`, `speaking_stopped` | Call/Confbridge |
| Groupcall | `groupcall_created`, `groupcall_progressing`, `groupcall_hangup` | Call |
| ExternalMedia | external media lifecycle events | Call/Confbridge |

## What Does NOT Change

- Existing `/v1/{resource_type}/{resource_id}/timeline` endpoint — unchanged
- `PublishEvent` / `PublishWebhookEvent` function signatures — unchanged
- RabbitMQ real-time event pub/sub — unchanged
- No changes to any of the 26 services that publish events
- ClickHouse existing schema (additive column only)

## Risks and Trade-offs

1. **Historical data gap** — Events inserted before the migration won't have `activeflow_id` populated. Accepted.
2. **No secondary index** — Queries on `activeflow_id` scan within partitions. Acceptable for v1 given low event volume per activeflow. Add index later if needed.
3. **Sub-resource gaps** — DTMF, AIMessage, Speaking, Groupcall, ExternalMedia events won't appear in v1. Acceptable — top-level events cover the primary use case.
4. **Chained activeflows** — Each activeflow is queried separately. Customers can traverse chains via `reference_activeflow_id` in activeflow events.
