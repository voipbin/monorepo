# Call SIP Timeline & PCAP API Design

## Overview

Add API endpoints to retrieve SIP message timeline and PCAP download for calls. This enables customers to debug their own calls and support staff to diagnose issues.

## API Endpoints

### GET /v1/timelines/call/{call-id}/sip-messages

Returns SIP messages for a call as JSON.

**Authorization:**
- Customers: Can access their own calls (validated via `call.CustomerID`)
- Support staff: Can access any call with admin permissions

**Response:**
```json
{
  "call_id": "uuid",
  "sip_call_id": "abc123@host",
  "messages": [
    {
      "timestamp": "2026-02-05T10:30:00.123456Z",
      "method": "INVITE",
      "src_ip": "10.0.0.1",
      "src_port": 5060,
      "dst_ip": "10.0.0.2",
      "dst_port": 5060,
      "raw": "INVITE sip:user@host SIP/2.0\r\n..."
    }
  ]
}
```

**Limits:**
- Maximum 50 messages returned
- Silent truncation (no indicator when messages exceed limit)
- Users should download PCAP for full analysis

### GET /v1/timelines/call/{call-id}/pcap

Returns a signed download URL for PCAP file.

**Response:**
```json
{
  "call_id": "uuid",
  "download_uri": "https://storage.googleapis.com/...",
  "expires_at": "2026-02-05T10:45:00Z"
}
```

## Service Architecture

### New Service: bin-timeline-manager

Dedicated service for timeline/trace features with Homer integration.

**Responsibilities:**
- Query Homer API for SIP messages
- Generate PCAP downloads via Homer's export endpoint
- Upload PCAP files to GCS via storage-manager
- Cache PCAP download URLs in Redis

### Request Flow

```
User Request
     │
     ▼
bin-api-manager
     │  1. Validate auth
     │  2. Fetch call via RPC to call-manager
     │  3. Check permissions (customer owns call OR admin)
     │  4. Extract: sip_call_id, tm_create, tm_hangup
     │  5. RPC to timeline-manager
     │
     ▼
bin-timeline-manager
     │  1. Query Homer API with sip_call_id + time range
     │  2. For messages: return JSON (max 50)
     │  3. For PCAP: fetch from Homer, upload via storage-manager, return URL
     │
     ▼
bin-storage-manager (PCAP only)
     │  1. Store PCAP in GCS
     │  2. Return signed download URL (15 min TTL)
     │
     ▼
Response to User
```

### Timeline-manager Internal RPC Methods

- `TimelineV1SIPMessagesGet(sip_call_id, from_time, to_time)` - Returns SIP messages
- `TimelineV1PcapGet(sip_call_id, from_time, to_time)` - Returns download URL

## Homer Integration

### SIP Messages Endpoint

Uses existing Homer API pattern:

```
POST /api/v3/call/transaction
```

### PCAP Export Endpoint

Uses Homer's native PCAP export:

```
POST /api/v3/export/call/messages/pcap
```

**Request Body:**
```json
{
  "param": {
    "transaction": {},
    "limit": 200,
    "search": {
      "1_call": {
        "callid": ["<sip-call-id>"],
        "type": "string",
        "hepid": 1
      }
    }
  },
  "timestamp": {
    "from": 1683637401000,
    "to": 1683641001000
  }
}
```

### Configuration

Environment variables (same pattern as call-manager recovery):
- `HOMER_API_ADDRESS` - Homer server URL
- `HOMER_AUTH_TOKEN` - Authentication token

### Time Range

- Uses call's `tm_create` and `tm_hangup` timestamps
- Adds ±30 seconds buffer to capture setup and teardown messages
- If call not found, return 404 (no fallback to arbitrary time window)

## PCAP Generation

### Flow

1. Timeline-manager calls Homer's `POST /api/v3/export/call/messages/pcap`
2. Homer returns PCAP bytes directly
3. Timeline-manager uploads to GCS via storage-manager
4. Returns signed URL with 15-minute expiry

### Caching Strategy

- Cache key: `pcap:{sip_call_id}:{from_ts}:{to_ts}`
- Store download URL in Redis for 15 minutes
- Check cache first, return cached URL if valid and not expired
- Avoids regenerating PCAP for repeated requests

## Error Handling

| Scenario | HTTP Status | Response |
|----------|-------------|----------|
| Call ID not found | 404 | `{"error": "call not found"}` |
| Call exists but no SIP Call-ID | 404 | `{"error": "no SIP data available for this call"}` |
| User doesn't own call (and not admin) | 403 | `{"error": "permission denied"}` |
| Homer unavailable/timeout | 502 | `{"error": "upstream service unavailable"}` |
| Homer returns empty data | 200 | Empty messages array / PCAP with no packets |
| Storage-manager fails | 502 | `{"error": "failed to generate download"}` |

### Timeouts

- Homer API call: 30 seconds
- Storage upload: 30 seconds
- Total request timeout: 60 seconds

## Implementation

### New Service Structure

```
bin-timeline-manager/
├── cmd/main.go
├── internal/config/main.go
├── pkg/timelinehandler/
│   ├── main.go
│   ├── sip_messages.go
│   └── pcap.go
├── pkg/homerhandler/
│   ├── main.go
│   ├── messages.go
│   └── pcap.go
├── models/sipmessage/main.go
├── go.mod
└── go.sum
```

### Database

No new tables required. Timeline-manager is stateless:
- Homer is the source of truth for SIP data
- Redis for PCAP URL caching only

### Configuration (bin-timeline-manager)

- `HOMER_API_ADDRESS` - Homer server URL
- `HOMER_AUTH_TOKEN` - Authentication token
- `REDIS_ADDRESS` - For PCAP URL caching

### Dependencies

- `bin-call-manager` - Call lookup (via api-manager)
- `bin-storage-manager` - GCS upload for PCAP files
- Homer API - External SIP capture system

### API Manager Changes

Add two new route handlers in `bin-api-manager`:
- `TimelineCallSIPMessagesGet` - GET /v1/timelines/call/{call-id}/sip-messages
- `TimelineCallPcapGet` - GET /v1/timelines/call/{call-id}/pcap

### OpenAPI Changes

Add endpoint definitions in `bin-openapi-manager` for the new routes.
