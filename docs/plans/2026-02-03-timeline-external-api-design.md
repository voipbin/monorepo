# Timeline External API Design

## Overview

Add a public API endpoint for customers to retrieve timeline events for their resources. Events are returned in WebhookMessage format for consistency with webhook deliveries.

## API Endpoint

```
GET /v1/timelines/{resource_type}/{resource_id}/events
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `resource_type` | string | One of: `calls`, `conferences`, `flows`, `activeflows` |
| `resource_id` | UUID | The resource identifier |

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page_size` | int | 100 | Number of events per page (max: 1000) |
| `page_token` | string | - | Cursor for pagination |

### Response

```json
{
  "result": [
    {
      "timestamp": "2024-01-15T10:00:00Z",
      "event_type": "call_created",
      "data": { /* WebhookMessage format */ }
    }
  ],
  "next_page_token": "2024-01-15T09:59:59.999Z"
}
```

## Resource Type Mapping

| resource_type | ServiceName | Validation Helper | Event Pattern | Converter |
|---------------|-------------|-------------------|---------------|-----------|
| `calls` | `ServiceNameCallManager` | `callGet()` | `call_*` | `call.Call.ConvertWebhookMessage()` |
| `conferences` | `ServiceNameConferenceManager` | `conferenceGet()` | `conference_*` | `conference.Conference.ConvertWebhookMessage()` |
| `flows` | `ServiceNameFlowManager` | `flowGet()` | `flow_*` | `flow.Flow.ConvertWebhookMessage()` |
| `activeflows` | `ServiceNameFlowManager` | `activeflowGet()` | `activeflow_*` | `activeflow.Activeflow.ConvertWebhookMessage()` |

## Validation Flow

1. Parse `resource_type` → validate against allowed types
2. Parse `resource_id` → validate UUID format
3. Fetch resource using existing helper (e.g., `callGet()`)
4. Check ownership: `resource.CustomerID == agent.CustomerID`
5. Query timeline-manager with ServiceName and resource ID
6. Convert each event's Data to WebhookMessage using appropriate converter
7. Return paginated response

## Error Handling

| Scenario | HTTP Status | Response |
|----------|-------------|----------|
| Invalid `resource_type` | 400 | `{"message": "invalid resource type"}` |
| Invalid `resource_id` (not UUID) | 400 | `{"message": "invalid resource id"}` |
| Resource not found | 404 | `{"message": "resource not found"}` |
| Resource not owned by customer | 404 | `{"message": "resource not found"}` |
| Timeline query fails | 500 | `{"message": "internal error"}` |
| Event data conversion fails | - | Log error, skip event, continue |

## Permissions

Require `CustomerAdmin` or `CustomerManager` permission (same as viewing the resource directly).

## Implementation Files

### New Files

- `bin-api-manager/pkg/servicehandler/timeline.go` - Main handler logic

### Modified Files

- `bin-openapi-manager/openapi/openapi.yaml` - Add endpoint and schemas
- `bin-api-manager/gens/openapi_server/` - Regenerate after OpenAPI changes
- `bin-api-manager/server/` - Wire up endpoint

### No Changes Needed

- `bin-timeline-manager` - Already has `TimelineV1EventList` in requesthandler
- `bin-common-handler` - Requesthandler already exists
