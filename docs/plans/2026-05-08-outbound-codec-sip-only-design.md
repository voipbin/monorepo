# Design: Outbound Profile Codec Gate by Destination Type

**Date:** 2026-05-08  
**Branch:** NOJIRA-Outbound-codec-sip-only  
**Status:** Approved

## Problem

The outbound profile's `Codecs` field currently controls codec injection for PSTN
(`TypeTel`) outgoing calls. This is incorrect: PSTN trunks negotiate codecs directly
with the carrier via SDP, so injecting a codec header overrides carrier-negotiated
SDP and can cause call failures. Codec injection should apply to SIP calls only,
where the platform controls both sides of the SDP negotiation.

## Goal

`OutboundConfig.Codecs` is embedded into outgoing call metadata — and thus into the
outgoing SIP channel — for `TypeSIP` destinations only. PSTN calls are unaffected.

## Scope

Single file change: `bin-call-manager/pkg/callhandler/outgoing_call.go`, function
`CreateCallOutgoing`. No model changes, no API changes, no new packages.

## Design

### Core change

Replace the PSTN-only outbound config fetch block with a unified fetch and a
switch-case for codec embedding:

```go
// Fetch outbound config once for all non-internal customers.
// Used for: codec embedding (SIP only), whitelist + source validation (PSTN).
var outboundCfg *outboundconfig.OutboundConfig
if !cucustomer.IsInternalSystemID(customerID) {
    var cfgErr error
    outboundCfg, cfgErr = h.outboundConfigHandler.GetByCustomerID(ctx, customerID)
    if cfgErr != nil {
        log.Errorf("Could not get outbound config; rejecting call (fail-closed). err: %v", cfgErr)
        outboundconfighandler.IncFetchError("db_error")
        return nil, fmt.Errorf("could not get outbound config: %w", cfgErr)
    }
    // Codec embedding is destination-type-specific.
    // PSTN trunks negotiate codecs directly with the carrier; injecting a
    // codec header into PSTN calls overrides carrier SDP, which is incorrect.
    // Internal system IDs (IDCallManager, IDAIManager, etc.) skip this entire
    // block — they have no OutboundConfig row and must not have codecs injected.
    switch destination.Type {
    case commonaddress.TypeSIP:
        metadata = embedCodecs(metadata, outboundCfg)
    case commonaddress.TypeTel:
        // no codec embedding for PSTN
    }
}

// PSTN-only: whitelist + source number validation.
// outboundCfg is nil for internal system IDs — ValidateDestination allows
// all destinations in that case (internal callers bypass the whitelist check).
if destination.Type == commonaddress.TypeTel {
    if !h.ValidateDestination(ctx, customerID, outboundCfg, destination) {
        // ... rejection logic unchanged
    }
}
```

Supporting functions `embedCodecs` and `setChannelVariableCodecs` are unchanged.

### Behavior matrix

| Destination  | Internal system ID | Outbound config fetch  | Codec embedded |
|--------------|--------------------|------------------------|----------------|
| PSTN (Tel)   | no                 | yes (whitelist/source) | **no**         |
| PSTN (Tel)   | yes                | no                     | no             |
| SIP          | no                 | yes                    | **yes**        |
| SIP          | yes                | no                     | no             |

### Error handling

| Condition                                    | Outcome                                                    |
|----------------------------------------------|------------------------------------------------------------|
| DB/cache error on fetch                      | Call rejected (fail-closed) — same for PSTN and SIP        |
| No outbound config row (`nil, nil`)          | `embedCodecs` no-ops; PSTN whitelist denies if no list set |
| Per-call metadata codec override already set | `embedCodecs` respects existing value (unchanged)          |
| Internal system ID + TypeTel                 | `outboundCfg` nil → `ValidateDestination` allows (bypass)  |

## Pre-ship verification

The `Codecs` field was previously embedded into PSTN call metadata. Before deploying,
run the following query to confirm no live customer has `Codecs` set for PSTN use:

```sql
SELECT id, customer_id, codecs
FROM outbound_configs
WHERE codecs != '' AND tm_delete IS NULL;
```

If any rows exist, those customers will silently lose codec injection on PSTN calls.
Coordinate before deploying.

## Tests

### Existing tests to update

All existing `TypeSIP` destination test cases in `outgoing_call_test.go` currently
have `outboundConfigHandler: nil` in the test handler struct. After this change,
`CreateCallOutgoing` calls `GetByCustomerID` for every non-internal SIP call, causing
a nil-pointer panic. Every SIP test case must inject a mock:

```go
mockOutboundConfig.EXPECT().GetByCustomerID(ctx, customerID).Return(nil, nil)
```

### New test cases

1. SIP call, outbound config has codecs → codec present in call metadata
2. PSTN call, outbound config has codecs → codec **not** in call metadata
3. SIP call, `GetByCustomerID` returns error → call rejected (fail-closed)
4. Internal system ID + SIP destination → no fetch called, no codec in metadata
