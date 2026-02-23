# Fix Billing Call Cost Type Detection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix `getCostTypeForCall()` logic to correctly classify incoming VN, PSTN, and direct extension calls, and lower the VN credit rate to 1,000 micros.

**Architecture:** Two changes in `bin-billing-manager`: (1) rewrite `getCostTypeForCall()` decision tree in `pkg/billinghandler/event.go`, (2) update `DefaultCreditPerUnitCallVN` constant in `models/billing/cost_type.go`.

**Tech Stack:** Go, gomock

---

### Task 1: Update VN credit rate constant

**Files:**
- Modify: `bin-billing-manager/models/billing/cost_type.go:22`

**Step 1: Change the constant**

```go
// Before:
DefaultCreditPerUnitCallVN int64 = 4500    // $0.0045/min

// After:
DefaultCreditPerUnitCallVN int64 = 1000    // $0.001/min
```

**Step 2: Run tests to check for breakage**

Run: `cd bin-billing-manager && go test ./...`
Expected: Some tests in `pkg/billinghandler/` may fail if they hardcode 4500 for VN scenarios. Note which tests fail — they'll be fixed in Task 3.

**Step 3: Commit (defer until Task 3 completes)**

Do NOT commit yet — this change pairs with the logic fix.

---

### Task 2: Rewrite getCostTypeForCall() logic

**Files:**
- Modify: `bin-billing-manager/pkg/billinghandler/event.go:54-83`

**Step 1: Replace the getCostTypeForCall function**

New logic:

```go
func getCostTypeForCall(c *cmcall.Call) billing.CostType {
	switch c.Direction {
	case cmcall.DirectionIncoming:
		if c.Destination.Type == commonaddress.TypeTel {
			if strings.HasPrefix(c.Destination.Target, nmnumber.VirtualNumberPrefix) {
				return billing.CostTypeCallVN
			}
			return billing.CostTypeCallPSTNIncoming
		}
		if c.Source.Type == commonaddress.TypeSIP && c.Destination.Type == commonaddress.TypeExtension {
			return billing.CostTypeCallDirectExt
		}
		return billing.CostTypeCallExtension

	case cmcall.DirectionOutgoing:
		if c.Destination.Type == commonaddress.TypeTel {
			return billing.CostTypeCallPSTNOutgoing
		}
		return billing.CostTypeCallExtension

	default:
		return billing.CostTypeCallPSTNOutgoing
	}
}
```

Key changes from current logic:
1. **PSTN Incoming** — no longer requires `src=tel`, only checks `dst=tel` and not `+899`
2. **VN** — now requires `dst=tel` type check before prefix check (was only checking prefix regardless of type)
3. **Direct Extension** — new: `src=sip AND dst=extension` returns `CostTypeCallDirectExt` (was falling through to free extension)

**Step 2: Run tests**

Run: `cd bin-billing-manager && go test ./pkg/billinghandler/ -run Test_getCostTypeForCall -v`
Expected: Some existing test cases will fail because the logic changed. Fix in Task 3.

---

### Task 3: Update tests for getCostTypeForCall

**Files:**
- Modify: `bin-billing-manager/pkg/billinghandler/event_test.go`

**Step 1: Update the `Test_getCostTypeForCall` test cases**

Replace the test table in `Test_getCostTypeForCall` with cases matching the new decision tree:

```go
{
    name: "incoming call to virtual number (dst=tel with +899 prefix)",
    call: &cmcall.Call{
        Direction: cmcall.DirectionIncoming,
        Source: commonaddress.Address{
            Type:   commonaddress.TypeSIP,
            Target: "sip:user@example.com",
        },
        Destination: commonaddress.Address{
            Type:   commonaddress.TypeTel,
            Target: nmnumber.VirtualNumberPrefix + "1234567",
        },
    },
    expectCostType: billing.CostTypeCallVN,
},
{
    name: "incoming PSTN call (dst=tel, no +899 prefix)",
    call: &cmcall.Call{
        Direction: cmcall.DirectionIncoming,
        Source: commonaddress.Address{
            Type: commonaddress.TypeTel,
        },
        Destination: commonaddress.Address{
            Type:   commonaddress.TypeTel,
            Target: "+14155551234",
        },
    },
    expectCostType: billing.CostTypeCallPSTNIncoming,
},
{
    name: "incoming PSTN call (src=sip, dst=tel, no +899 prefix)",
    call: &cmcall.Call{
        Direction: cmcall.DirectionIncoming,
        Source: commonaddress.Address{
            Type:   commonaddress.TypeSIP,
            Target: "sip:trunk@provider.com",
        },
        Destination: commonaddress.Address{
            Type:   commonaddress.TypeTel,
            Target: "+14155551234",
        },
    },
    expectCostType: billing.CostTypeCallPSTNIncoming,
},
{
    name: "incoming direct extension (src=sip, dst=extension)",
    call: &cmcall.Call{
        Direction: cmcall.DirectionIncoming,
        Source: commonaddress.Address{
            Type:   commonaddress.TypeSIP,
            Target: "sip:user@example.com",
        },
        Destination: commonaddress.Address{
            Type:   commonaddress.TypeExtension,
            Target: "1001",
        },
    },
    expectCostType: billing.CostTypeCallDirectExt,
},
{
    name: "incoming call from extension - free",
    call: &cmcall.Call{
        Direction: cmcall.DirectionIncoming,
        Source: commonaddress.Address{
            Type: commonaddress.TypeExtension,
        },
        Destination: commonaddress.Address{
            Type:   commonaddress.TypeExtension,
            Target: "1001",
        },
    },
    expectCostType: billing.CostTypeCallExtension,
},
{
    name: "incoming call from agent - free",
    call: &cmcall.Call{
        Direction: cmcall.DirectionIncoming,
        Source: commonaddress.Address{
            Type: commonaddress.TypeAgent,
        },
        Destination: commonaddress.Address{
            Type:   commonaddress.TypeExtension,
            Target: "1002",
        },
    },
    expectCostType: billing.CostTypeCallExtension,
},
{
    name: "outgoing call to PSTN",
    call: &cmcall.Call{
        Direction: cmcall.DirectionOutgoing,
        Destination: commonaddress.Address{
            Type:   commonaddress.TypeTel,
            Target: "+14155551234",
        },
    },
    expectCostType: billing.CostTypeCallPSTNOutgoing,
},
{
    name: "outgoing call to extension - free",
    call: &cmcall.Call{
        Direction: cmcall.DirectionOutgoing,
        Destination: commonaddress.Address{
            Type:   commonaddress.TypeExtension,
            Target: "1003",
        },
    },
    expectCostType: billing.CostTypeCallExtension,
},
{
    name: "outgoing call to agent - free",
    call: &cmcall.Call{
        Direction: cmcall.DirectionOutgoing,
        Destination: commonaddress.Address{
            Type: commonaddress.TypeAgent,
        },
    },
    expectCostType: billing.CostTypeCallExtension,
},
{
    name: "outgoing call to SIP - free",
    call: &cmcall.Call{
        Direction: cmcall.DirectionOutgoing,
        Destination: commonaddress.Address{
            Type: commonaddress.TypeSIP,
        },
    },
    expectCostType: billing.CostTypeCallExtension,
},
{
    name: "outgoing call to conference - free",
    call: &cmcall.Call{
        Direction: cmcall.DirectionOutgoing,
        Destination: commonaddress.Address{
            Type: commonaddress.TypeConference,
        },
    },
    expectCostType: billing.CostTypeCallExtension,
},
{
    name: "unknown direction defaults to PSTN outgoing",
    call: &cmcall.Call{
        Direction: "",
    },
    expectCostType: billing.CostTypeCallPSTNOutgoing,
},
```

Also update the VN test case in `Test_EventCMCallProgressing` — the existing "incoming call to virtual number" test has `Destination.Type: commonaddress.TypeExtension` which will no longer match VN under the new logic (VN now requires `dst=tel`). Update to `commonaddress.TypeTel`.

**Step 2: Run all tests**

Run: `cd bin-billing-manager && go test ./... -v`
Expected: All PASS

**Step 3: Run linter**

Run: `cd bin-billing-manager && golangci-lint run -v --timeout 5m`
Expected: No new issues

**Step 4: Commit all changes**

```bash
git add bin-billing-manager/models/billing/cost_type.go bin-billing-manager/pkg/billinghandler/event.go bin-billing-manager/pkg/billinghandler/event_test.go
git commit -m "NOJIRA-fix-billing-call-cost-type-detection

- bin-billing-manager: Fix getCostTypeForCall to check dst=tel type before VN prefix
- bin-billing-manager: Add direct extension detection for incoming SIP-to-extension calls
- bin-billing-manager: Remove src=tel requirement for PSTN incoming classification
- bin-billing-manager: Lower DefaultCreditPerUnitCallVN from 4500 to 1000 micros
- bin-billing-manager: Update tests for new call cost type detection logic"
```
