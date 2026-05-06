# Selective Codec for Outbound Calling — Design

**Date:** 2026-05-03
**Branch:** NOJIRA-selective-codec-outbound
**Status:** Approved

## Overview

Allow customers to configure a preferred codec list for outbound calls. When set, call-manager adds a `VBOUT-CODECS` SIP header to the outgoing INVITE so Kamailio can filter or transcode accordingly.

Configuration lives at two levels:
- **Customer-level default** — admin sets `outbound_codecs` on `customer.Metadata`
- **Per-call override** — call metadata key `codecs` takes precedence over the customer default

When neither is set, no header is sent and Kamailio uses current pass-through behavior.

## Codec Format

Comma-separated codec names, e.g. `"PCMU,PCMA,G729"`. Order implies preference. The value is passed verbatim as the SIP header value.

## Affected Services

| Service | Change |
|---|---|
| `bin-customer-manager` | Add `OutboundCodecs string` to `Metadata` struct |
| `bin-call-manager` | New metadata key, new `codec.go` helper file, two call sites |

No DB migrations required — `Metadata` is a JSON column in both services.

## Data Model Changes

### `bin-customer-manager/models/customer/metadata.go`

```go
type Metadata struct {
    RTPDebug       bool   `json:"rtp_debug"`
    OutboundCodecs string `json:"outbound_codecs"` // comma-separated, e.g. "PCMU,PCMA,G729"
}
```

### `bin-call-manager/models/call/metadata.go`

```go
// MetadataKeyCodecs sets the outbound codec preference for the call.
// Overrides the customer-level OutboundCodecs when present.
// Value is a comma-separated string, e.g. "PCMU,PCMA,G729".
MetadataKeyCodecs MetadataKey = "codecs"

// in ValidMetadataKeys:
MetadataKeyCodecs: true,
```

### `bin-call-manager/models/common/sip.go`

```go
SIPHeaderCodecs = "VBOUT-CODECS" // outbound codec preference for Kamailio
```

## New File: `bin-call-manager/pkg/callhandler/codec.go`

Contains two pure, independently testable helper functions:

```go
// embedCustomerCodecs copies OutboundCodecs from customer metadata into call
// metadata if the call does not already carry a codecs override.
// Returns the (possibly new) metadata map.
func embedCustomerCodecs(metadata map[string]any, outboundCodecs string) map[string]any {
    if _, alreadySet := metadata[call.MetadataKeyCodecs]; alreadySet {
        return metadata
    }
    if outboundCodecs == "" {
        return metadata
    }
    if metadata == nil {
        metadata = map[string]any{}
    }
    metadata[call.MetadataKeyCodecs] = outboundCodecs
    return metadata
}

// setChannelVariableCodecs adds the VBOUT-CODECS SIP header to the outgoing
// channel variables if a codec preference is present in call metadata.
// CRLF characters in the value are rejected silently (injection defence).
func setChannelVariableCodecs(variables map[string]string, metadata map[string]any) {
    codecs, ok := metadata[call.MetadataKeyCodecs].(string)
    if !ok || codecs == "" {
        return
    }
    if strings.ContainsAny(codecs, "\r\n") {
        return
    }
    variables["PJSIP_HEADER(add,"+common.SIPHeaderCodecs+")"] = codecs
}
```

## Changes to Existing Files

### `bin-call-manager/pkg/callhandler/tech_headers.go`

Add to `reservedTechHeaderKeys` so provider-supplied `tech_headers` cannot override the customer's codec preference:

```go
"PJSIP_HEADER(add,VBOUT-CODECS)": {},
```

### `bin-call-manager/pkg/callhandler/outgoing_call.go`

**In `CreateCallOutgoing`**, after the existing `RTPDebug` guard (~line 158):

```go
metadata = embedCustomerCodecs(metadata, cu.Metadata.OutboundCodecs)
```

**In `createChannelOutgoing`**, after `setChannelVariablesCallerID`:

```go
setChannelVariableCodecs(channelVariables, c.Metadata)
```

## Data Flow

```
Admin: PUT /v1/customers/{id}/metadata  {"outbound_codecs": "PCMU,PCMA,G729"}
                    ↓
CreateCallOutgoing: embedCustomerCodecs → metadata["codecs"] = "PCMU,PCMA,G729"
  (skipped if metadata["codecs"] already set — per-call override wins)
                    ↓
createChannelOutgoing: setChannelVariableCodecs → channelVariables["PJSIP_HEADER(add,VBOUT-CODECS)"] = "PCMU,PCMA,G729"
                    ↓
Asterisk INVITE → VBOUT-CODECS: PCMU,PCMA,G729
                    ↓
Kamailio (out of scope — separate session)
```

## Failover Behavior

`createFailoverChannel` calls `createChannelOutgoing` again with the same `call.Call` object. Since codecs are stored in `c.Metadata`, they are automatically carried through on route failover — no special handling needed.

## Security

- `PJSIP_HEADER(add,VBOUT-CODECS)` is added to `reservedTechHeaderKeys` — providers cannot override it via `tech_headers`
- CRLF characters in the codec value are rejected in `setChannelVariableCodecs` (same policy as `mergeTechHeaders`)

## Testing Plan

### `bin-customer-manager`
- `TestMetadata_OutboundCodecs_JSONRoundTrip` — verify `outbound_codecs` serialises/deserialises correctly

### `bin-call-manager/pkg/callhandler/codec.go`
- `TestEmbedCustomerCodecs_setsFromCustomer` — customer value embedded when call metadata empty
- `TestEmbedCustomerCodecs_perCallOverrideWins` — existing `metadata["codecs"]` not overwritten
- `TestEmbedCustomerCodecs_emptyCustomerValue_noOp` — no key added when `OutboundCodecs` is empty
- `TestEmbedCustomerCodecs_nilMetadata` — nil input produces new map with codec key
- `TestSetChannelVariableCodecs_addsHeader` — header added when codecs set
- `TestSetChannelVariableCodecs_emptyValue_noHeader` — no header when value is empty string
- `TestSetChannelVariableCodecs_crlfRejected` — CRLF in value produces no header

### `bin-call-manager/pkg/callhandler/tech_headers.go`
- Extend existing `TestReservedTechHeaderKeys` (or equivalent) to assert `PJSIP_HEADER(add,VBOUT-CODECS)` is blocked

### `bin-call-manager/models/call/metadata.go`
- Existing `Test_ValidMetadataKeys_contains_all_declared_constants` will enforce `MetadataKeyCodecs` is registered once declared
