# Call Extension Billing Type Design

## Problem Statement

All calls are billed at $0.020/second regardless of whether they use PSTN trunks or stay internal. Extension-to-extension calls, agent calls, SIP calls, and conference calls don't incur carrier costs, but customers are charged the same rate as PSTN calls.

## Approach

Add a new billing reference type `call_extension` that represents non-PSTN calls. The billing subscribe handler determines the reference type by inspecting the call's direction and address type:

- Incoming call with `Source.Type == "tel"` → `call` (charged $0.020/sec)
- Outgoing call with `Destination.Type == "tel"` → `call` (charged $0.020/sec)
- Everything else → `call_extension` (free)

## Billing Map

### Destination-based rate determination

| Destination Type | Billing ReferenceType | Rate |
|---|---|---|
| `tel` (PSTN) | `call` | $0.020/sec |
| `extension` | `call_extension` | $0.00 |
| `agent` | `call_extension` | $0.00 |
| `sip` | `call_extension` | $0.00 |
| `conference` | `call_extension` | $0.00 |
| `line` | `call_extension` | $0.00 |

### Direction-based field check

| Direction | Field to check | `tel` | All other types |
|---|---|---|---|
| Incoming | `Source.Type` | `call` ($0.020/sec) | `call_extension` (free) |
| Outgoing | `Destination.Type` | `call` ($0.020/sec) | `call_extension` (free) |

### Call scenario examples

| Scenario | A leg billing | B leg billing |
|---|---|---|
| Extension → Extension | `call_extension` (free) | `call_extension` (free) |
| Extension → PSTN | `call_extension` (free) | `call` (charged) |
| PSTN → Extension | `call` (charged) | `call_extension` (free) |
| PSTN → PSTN | `call` (charged) | `call` (charged) |

## Behavior by Reference Type

| Behavior | `call` | `call_extension` |
|---|---|---|
| Balance check | Check account balance | Always return valid |
| Billing record created | Yes | Yes |
| Cost per unit | $0.020/sec | $0.00/sec |
| Cost total | `cost_per_unit * duration` | $0.00 |
| Balance deduction | Yes | No (skip when cost_total == 0) |

## Reference Type Determination Logic

The subscribe handler in billing-manager determines the reference type from the call object:

```go
func getReferenceTypeForCall(c *cmcall.Call) billing.ReferenceType {
    switch c.Direction {
    case call.DirectionIncoming:
        if c.Source.Type == address.TypeTel {
            return billing.ReferenceTypeCall
        }
        return billing.ReferenceTypeCallExtension

    case call.DirectionOutgoing:
        if c.Destination.Type == address.TypeTel {
            return billing.ReferenceTypeCall
        }
        return billing.ReferenceTypeCallExtension

    default:
        // Safe fallback: charge it
        return billing.ReferenceTypeCall
    }
}
```

## Files to Change

| File | Change |
|---|---|
| `bin-billing-manager/models/billing/billing.go` | Add `ReferenceTypeCallExtension` and `DefaultCostPerUnitReferenceTypeCallExtension` constants |
| `bin-billing-manager/pkg/subscribehandler/call.go` | Add `getReferenceTypeForCall()` helper; use it in `processEventCMCallProgressing` and `processEventCMCallHangup` |
| `bin-billing-manager/pkg/billinghandler/billing.go` | Handle `ReferenceTypeCallExtension` in `BillingStart` (set cost_per_unit to 0); skip balance deduction when cost_total is 0 in `BillingEnd` |
| `bin-billing-manager/pkg/accounthandler/balance.go` | Handle `ReferenceTypeCallExtension` in `IsValidBalance` — return valid immediately |

## What Does NOT Change

- No changes to call-manager or flow-manager — they send the same call events as before
- No database schema changes — `reference_type` is a varchar, so `"call_extension"` works without migration
- No OpenAPI changes — the billing model already has `reference_type` as a string field
- No changes to billing end calculation — `0 * duration = 0` naturally

## Scope

4 files in `bin-billing-manager` only.
