# Call RTP Debug Capture

## Problem

The `Customer.Metadata.RTPDebug` flag exists and can be toggled via `PUT /v1/customers/{id}/metadata`, but nothing consumes it. When a customer has RTP debug enabled, we need to instruct RTPEngine to capture RTP traffic for that customer's calls, making the captures available in Homer for later analysis.

## Approach

When a call starts for a customer with `RTPDebug=true`, bin-call-manager sends RTPEngine a `"start recording"` command. When the call ends, it sends `"stop recording"`. The call model gets a new `Metadata map[string]interface{}` field to track that RTP debug was enabled for that specific call.

### Trigger points — per direction

Channel SIPData is stored on the Channel model, populated from Kamailio's Redis hash (`kamailio:<sip-call-id>`). Kamailio writes this metadata to Redis **before** routing the call to Asterisk, so the data is available in Redis by the time any ARI event fires.

SIPData cannot be populated at `ChannelCreated` time because the channel has not entered the Stasis application yet — ARI channel variable queries (`CHANNEL(pjsip,call-id)`) are not reliably available before `StasisStart`.

- **Incoming calls**: `UpdateSIPInfo()` runs during `StasisStart` in `channelhandler/arievent.go:ARIStasisStart()`. It receives the SIP Call-ID from the stasis args, reads the Kamailio Redis hash, and sets SIPData on the channel. The updated channel (with SIPData) is then passed to `callHandler.ARIStasisStart(ctx, cn)`, which eventually calls `startCallTypeFlow()` to create the call struct. The RTP debug check happens at **call creation time** — the channel already has SIPData.
- **Outgoing calls**: `UpdateSIPInfoByChannelVariable()` runs during `ChannelStateChange` (state Up) in `channelhandler/arievent.go:ARIChannelStateChange()`. It uses `variableGet("CHANNEL(pjsip,call-id)")` to get the SIP Call-ID (now available because the channel is in Stasis), reads the Kamailio Redis hash, and sets SIPData. The updated channel is then passed to `callHandler.ARIChannelStateChange(ctx, cn)`. The RTP debug check happens at **answer time**.
- **Stop recording**: On hangup, if `call.Metadata["rtp_debug"]` is set, fetch a fresh channel from DB (the channel parameter in `Hangup()` may be stale) and send `"stop recording"`.

All RTP debug logic lives in `callhandler` (not `channelhandler`) to maintain separation of concerns — call-domain logic stays in the call handler layer.

### Flow

```
Incoming call:

1. Kamailio receives SIP INVITE
   → writes SIP metadata to Redis hash (kamailio:<sip-call-id>)
   → routes call to Asterisk

2. Asterisk fires ChannelCreated
   → channelHandler.Create() — bare channel, no SIPData
   (channel not in Stasis yet, can't query ARI variables)

3. Asterisk fires StasisStart
   → arieventhandler.EventHandlerStasisStart()
     → channelHandler.ARIStasisStart(ctx, e)
       → UpdateSIPInfo(sipCallID from stasis args)
         → cache.KamailioMetadataGet() — reads Redis hash
         → db.ChannelSetSIPData() — stores on channel
       → returns channel WITH SIPData ✓
     → callHandler.ARIStasisStart(ctx, cn)  ← channel has SIPData
       → startCallTypeFlow()
         → fetch customer, check Metadata.RTPDebug
         → if true: set call.Metadata["rtp_debug"] = true
         → create call in DB
         → send "start recording" to RTPEngine (best-effort)

Outgoing call:

1. Call created via RPC (flow-manager, groupcall, etc.)
   → Asterisk dials out, channel created
   (no SIPData yet — Kamailio hasn't processed the outbound SIP)

2. Kamailio processes outbound SIP
   → writes SIP metadata to Redis hash

3. Asterisk fires ChannelStateChange (state Up — call answered)
   → arieventhandler.EventHandlerChannelStateChange()
     → channelHandler.ARIChannelStateChange(ctx, e)
       → UpdateSIPInfoByChannelVariable(ctx, cn)
         → variableGet("CHANNEL(pjsip,call-id)") — gets SIP Call-ID
         → cache.KamailioMetadataGet() — reads Redis hash
         → db.ChannelSetSIPData() — stores on channel
       → returns channel WITH SIPData ✓
     → callHandler.ARIChannelStateChange(ctx, cn)  ← channel has SIPData
       → fetch customer, check Metadata.RTPDebug
       → if true: update call metadata["rtp_debug"] = true via DB
       → send "start recording" to RTPEngine (best-effort)

Hangup (both directions):

1. Asterisk fires hangup event
   → callHandler.Hangup(ctx, cn)
     → get call via CallGetByChannelID
     → check call.Metadata["rtp_debug"]
     → if true:
       → fetch fresh channel from DB (cn parameter may be stale)
       → get rtpengineID from Channel.SIPData["rtpengine_address"]
       → send "stop recording" to RTPEngine (best-effort)
     → proceed with normal hangup cleanup
```

### RTPEngine command format

```go
rtpengineID := channel.SIPData[channel.SIPDataKeyRTPEngineAddress]
command := map[string]interface{}{
    "command": "start recording",
    "call-id": channel.SIPCallID,
}
reqHandler.RTPEngineV1CommandsSend(ctx, rtpengineID, command)
```

RTPEngine exports captured RTP to Homer via HEP protocol. Homer storage and PCAP download are out of scope (separate tickets).

## Changes

### 1. Call model — new Metadata field

**File:** `bin-call-manager/models/call/call.go`
- Add `Metadata map[string]interface{}` with `db:"metadata,json"` tag.

**File:** `bin-call-manager/models/call/metadata.go` (new)
- Define `MetadataKey` type and `MetadataKeyRTPDebug` constant.

```go
type MetadataKey = string

const (
    MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)
```

**File:** `bin-call-manager/models/call/field.go`
- Add `FieldMetadata Field = "metadata"`.

**File:** `bin-call-manager/models/call/webhook.go`
- Add `Metadata map[string]interface{}` to `WebhookMessage`.
- Update `ConvertWebhookMessage()` to copy the field.

### 2. Channel model — SIPData key constants

**File:** `bin-call-manager/models/channel/sipdata.go` (new)

```go
type SIPDataKey = string

const (
    SIPDataKeyCallID            SIPDataKey = "call_id"
    SIPDataKeyFromUser          SIPDataKey = "from_user"
    SIPDataKeyFromName          SIPDataKey = "from_name"
    SIPDataKeyFromDomain        SIPDataKey = "from_domain"
    SIPDataKeyFromURI           SIPDataKey = "from_uri"
    SIPDataKeyToUser            SIPDataKey = "to_user"
    SIPDataKeyToName            SIPDataKey = "to_name"
    SIPDataKeyToDomain          SIPDataKey = "to_domain"
    SIPDataKeyToURI             SIPDataKey = "to_uri"
    SIPDataKeyPAI               SIPDataKey = "pai"
    SIPDataKeyRTPEngineAddress  SIPDataKey = "rtpengine_address"
    SIPDataKeyDirection         SIPDataKey = "direction"
    SIPDataKeySourceIP          SIPDataKey = "source_ip"
    SIPDataKeyTransport         SIPDataKey = "transport"
    SIPDataKeyDomain            SIPDataKey = "domain"
)
```

### 3. Customer model — MetadataKey constants

**File:** `bin-customer-manager/models/customer/metadata.go`
- Add `MetadataKey` type and `MetadataKeyRTPDebug` constant (if not already present).

```go
type MetadataKey = string

const (
    MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)
```

### 4. Call dbhandler — handle Metadata in CRUD

**File:** `bin-call-manager/pkg/dbhandler/call.go`
- Metadata is handled automatically by `commondatabasehandler.PrepareFields()` and `ScanRow()` via the `db:"metadata,json"` tag.
- Initialize nil metadata as empty map in `callGetFromRow()` (same pattern as `Data`, `ChainedCallIDs`, etc.): `if res.Metadata == nil { res.Metadata = map[string]interface{}{} }`
- Ensure `CallUpdate` supports `FieldMetadata`.

### 5. Call handler — RTP debug trigger logic

**File:** `bin-call-manager/pkg/callhandler/start.go` (incoming call trigger)
- In the incoming call creation flow (e.g., `startCallTypeFlow()`), after the channel with SIPData is available:
  1. Fetch customer: `reqHandler.CustomerV1CustomerGet(ctx, customerID)`
  2. Check `customer.Metadata.RTPDebug`
  3. If true:
     - Set `call.Metadata[MetadataKeyRTPDebug] = true` on the call struct before DB insert
     - After call creation, send `"start recording"` to RTPEngine using `Channel.SIPData[SIPDataKeyRTPEngineAddress]` and `Channel.SIPCallID`
     - Log the result; do not block the call on failure

**File:** `bin-call-manager/pkg/callhandler/arievent.go` (outgoing call trigger)
- In `ARIChannelStateChange()`, when an outgoing call transitions to Up state (channel has SIPData populated by `UpdateSIPInfoByChannelVariable()`):
  1. Fetch customer: `reqHandler.CustomerV1CustomerGet(ctx, call.CustomerID)`
  2. Check `customer.Metadata.RTPDebug`
  3. If true:
     - Update call metadata via DB: set `metadata["rtp_debug"] = true`
     - Send `"start recording"` to RTPEngine using `Channel.SIPData[SIPDataKeyRTPEngineAddress]` and `Channel.SIPCallID`
     - Log the result; do not block the call on failure

**File:** `bin-call-manager/pkg/callhandler/hangup.go` (stop trigger)
- In `Hangup()`, before or after existing cleanup:
  1. Check `call.Metadata[MetadataKeyRTPDebug]`
  2. If true, fetch a **fresh channel from DB** (the `cn` parameter may be stale and missing SIPData)
  3. Send `"stop recording"` to RTPEngine; log result, do not block hangup

### 6. Database migration

**File:** `bin-dbscheme-manager/` — new Alembic migration
- Add `metadata JSON DEFAULT NULL` column to `call_calls` table.

### 7. OpenAPI schema

**File:** `bin-openapi-manager/openapi/openapi.yaml`
- Add `metadata` field to the call schema (object type, nullable).
- Regenerate: `go generate ./...` in both `bin-openapi-manager` and `bin-api-manager`.

### 8. RST documentation

**File:** `bin-api-manager/docsdev/source/` — call struct docs
- Add `metadata` field description to the call resource documentation.
- Clean rebuild HTML and commit both source + build.

## Out of scope

- RTPEngine-to-Homer HEP configuration (separate infra ticket).
- PCAP download API from Homer (separate ticket).
- RTP debug for conference bridges (future enhancement).

## Risks

- **Additional RPC call**: Fetching customer metadata adds latency to call creation (incoming) or answer handling (outgoing). This is a single RPC call and should be negligible compared to SIP signaling time.
- **RTPEngine command failure**: If `"start recording"` fails, we log the error but do not block the call. RTP debug is best-effort.
- **Orphaned captures**: If hangup handler fails to send `"stop recording"`, RTPEngine's recording continues until the call-id is cleaned up by RTPEngine's own timeout. Acceptable for a debug feature.
- **Stale channel at hangup**: The channel passed to `Hangup()` via ARI event may not have the latest SIPData. Mitigated by fetching a fresh channel from DB before sending `"stop recording"`.
- **Nil metadata serialization**: A nil `map[string]interface{}` serializes to `null` in JSON. Mitigated by initializing as empty map `{}` in `callGetFromRow()`, following existing patterns for `Data`, `ChainedCallIDs`, etc.
