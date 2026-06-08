# bin-timeline-manager Domain

## Domain Entities

### Event

A timeline record representing something that happened to a resource in the platform. Stored in ClickHouse `events` table.

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | `DateTime64(3)` | When the event occurred |
| `publisher` | `String` | Service that emitted the event (e.g., `call-manager`, `flow-manager`) |
| `type` | `String` | Event type identifier (e.g., `call_hangup`, `activeflow_created`) |
| `resource_id` | `UUID` | ID of the affected resource |
| `data` | `String` | JSON payload with event details |

ClickHouse schema uses `MergeTree()` engine ordered by `(publisher, resource_id, timestamp)` for efficient per-resource queries.

**Go domain model** (`models/event/event.go`) uses basic Go types for ClickHouse driver compatibility:
- `Publisher string` (not `ServiceName`)
- `Type string`
- `Data json.RawMessage`

### Event Query Request

API DTO for listing events (richer types used here, converted before DB call):

| Field | Type | Description |
|-------|------|-------------|
| `Publisher` | `ServiceName` | Required — filter by publishing service |
| `ID` | `uuid.UUID` | Required — filter by resource ID |
| `Events` | `[]string` | Required — event-type patterns (supports wildcards like `activeflow_*`) |
| `PageSize` | `int` | Optional — results per page (default: 100, max: 1000) |
| `PageToken` | `string` | Optional — opaque cursor for next page |

### Resource Correlation

A correlation graph links every resource that shares the same `activeflow_id`.
`activeflow_id` is the universal correlation key across most VoIPBin resources
(call, recording, confbridge, aicall, ai message, summary, transcribe,
conferencecall, queuecall). Given any one resource id, the service reverse-looks
up its `activeflow_id` (from the MATERIALIZED column) and returns all sibling
resources grouped by publisher.

| Field | Type | Description |
|-------|------|-------------|
| `ResourceID` | `uuid.UUID` | The queried resource id |
| `ResourceFound` | `bool` | Whether any event exists for the id (distinguishes "never seen" from "seen but no activeflow") |
| `ActiveflowID` | `uuid.UUID` | The resolved activeflow; `uuid.Nil` when none |
| `Truncated` | `bool` | `true` when the graph exceeded the 100-resource cap |
| `Resources` | `[]PublisherGroup` | Deduplicated resources grouped by publisher, each with `event_types`, `first_seen`, `last_seen` |

Known gap: `message-manager` (SMS) and `transfer-manager` resources do not carry
`activeflow_id` in their event payloads, so they are not captured. If they later
emit `activeflow_id`, they are picked up automatically (MATERIALIZED extraction).

## Key Business Rules

### Read-Only Query API

This service does **not** write events via its RPC API. Events enter ClickHouse through a separate subscription path (subscribe handler consuming 27 event queues). The RPC listen path only reads.

### Event-Type Wildcard Matching

Event type filters support suffix wildcards (e.g., `activeflow_*` matches `activeflow_created`, `activeflow_updated`, `activeflow_ended`). Pattern matching is applied at the ClickHouse query layer.

### Batch Insert on Subscribe

The subscribe handler processes incoming events in batches before inserting into ClickHouse. Metrics track both batch insert time and batch size to monitor ingestion performance.

### SIP Analysis via Homer

The `/v1/sip/analysis` and `/v1/sip/pcap` endpoints proxy requests to a Homer API instance (configured via `homer_api_address` and `homer_auth_token`). Homer stores SIP signalling capture data separately from the event timeline.

### Event Sources Subscribed

This service subscribes to 27 queues covering all platform services:

- AI, agent, asterisk, billing, call, campaign, conference, contact, conversation, customer
- Email, flow, message, number, outdial, pipecat, queue, registrar, route, sentinel
- Storage, tag, talk, transcribe, transfer, tts, webhook

Events published through these queues are stored verbatim in ClickHouse.
