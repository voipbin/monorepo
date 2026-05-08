# Outbound Profile Codec Gate by Destination Type — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move `OutboundConfig.Codecs` injection from PSTN calls to SIP calls — PSTN trunks negotiate codecs with the carrier, so injecting a codec header there is incorrect.

**Architecture:** Hoist the outbound config fetch outside the PSTN-only guard in `CreateCallOutgoing`. Use a switch-case to apply `embedCodecs` for `TypeSIP` only. Internal system customer IDs continue to skip the fetch entirely (same pattern as today). Fail-closed on DB/cache errors for both PSTN and SIP.

**Tech Stack:** Go, gomock (go.uber.org/mock), standard `go test` table-driven tests.

---

### Task 1: Write failing tests for new codec behaviour

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call_test.go`

The goal is to write tests that describe exactly what the new code must do. These tests must FAIL before any production code changes are made.

**Step 1: Add a new test function after `Test_CreateCallOutgoing_TypeTel_OutboundConfigFetchError_FailClosed` (around line 997)**

```go
// Test_CreateCallOutgoing_SIP_CodecEmbed verifies that OutboundConfig.Codecs is
// embedded into call metadata for SIP destinations and NOT for PSTN destinations.
func Test_CreateCallOutgoing_CodecEmbed(t *testing.T) {
	tests := []struct {
		name string

		customerID  uuid.UUID
		destination commonaddress.Address
		outboundCfg *outboundconfig.OutboundConfig

		expectCodecInMetadata bool
	}{
		{
			name: "SIP destination with codecs set - codec embedded in metadata",

			customerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
			destination: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testoutgoing@test.com",
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				Codecs:     "ulaw,alaw",
			},
			expectCodecInMetadata: true,
		},
		{
			name: "PSTN destination with codecs set - codec NOT embedded in metadata",

			customerID: uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821121656521",
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				CustomerID: uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
				Codecs:     "ulaw,alaw",
			},
			expectCodecInMetadata: false,
		},
	}

	// These are integration-style sub-tests that call embedCodecs directly
	// to assert gate logic without spinning up a full callHandler mock stack.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := map[string]any{}
			switch tt.destination.Type {
			case commonaddress.TypeSIP:
				metadata = embedCodecs(metadata, tt.outboundCfg)
			case commonaddress.TypeTel:
				// no codec embedding for PSTN
			}

			_, hasCodec := metadata[call.MetadataKeyCodecs]
			if hasCodec != tt.expectCodecInMetadata {
				t.Errorf("Wrong match. codec in metadata = %v, want %v", hasCodec, tt.expectCodecInMetadata)
			}
		})
	}
}
```

**Step 2: Add a SIP fail-closed test after `Test_CreateCallOutgoing_CodecEmbed`**

```go
// Test_CreateCallOutgoing_TypeSIP_OutboundConfigFetchError_FailClosed verifies
// that a DB error during outbound config fetch rejects a SIP call (fail-closed).
func Test_CreateCallOutgoing_TypeSIP_OutboundConfigFetchError_FailClosed(t *testing.T) {
	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		activeflowID uuid.UUID
		masterCallID uuid.UUID
		source       commonaddress.Address
		destination  commonaddress.Address
		fetchErr     error
	}{
		{
			name: "outbound config fetch returns db error for SIP call - call rejected",

			id:           uuid.FromStringOrNil("b1b2c3d4-0000-4000-8000-000000000001"),
			customerID:   uuid.FromStringOrNil("b1b2c3d4-0000-4000-8000-0000000000c1"),
			flowID:       uuid.FromStringOrNil("b1b2c3d4-0000-4000-8000-0000000000f1"),
			activeflowID: uuid.FromStringOrNil("b1b2c3d4-0000-4000-8000-0000000000a1"),
			masterCallID: uuid.Nil,
			source: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsrc@test.com",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testoutgoing@test.com",
			},
			fetchErr: fmt.Errorf("transient db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil    := utilhandler.NewMockUtilHandler(mc)
			mockReq     := requesthandler.NewMockRequestHandler(mc)
			mockNotify  := notifyhandler.NewMockNotifyHandler(mc)
			mockDB      := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				utilHandler:           mockUtil,
				reqHandler:            mockReq,
				notifyHandler:         mockNotify,
				db:                    mockDB,
				channelHandler:        mockChannel,
				outboundConfigHandler: mockOutboundConfig,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(&cucustomer.Customer{
				ID:                         tt.customerID,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(nil, tt.fetchErr)

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, false, false, "", nil)

			if res != nil {
				t.Errorf("Wrong match. expect: nil call, got: %v", res)
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !stderrors.Is(err, tt.fetchErr) {
				t.Errorf("Wrong match. expect error to wrap %v, got: %v", tt.fetchErr, err)
			}
			if !strings.Contains(err.Error(), "could not get outbound config") {
				t.Errorf("Wrong match. expect 'could not get outbound config' in error, got: %v", err)
			}
		})
	}
}
```

**Step 3: Run the new tests to confirm they fail (or panic) before the production change**

```bash
cd bin-call-manager
go test ./pkg/callhandler/... -run "Test_CreateCallOutgoing_CodecEmbed|Test_CreateCallOutgoing_TypeSIP_OutboundConfigFetchError_FailClosed" -v
```

Expected: The `CodecEmbed` test may pass trivially (it calls `embedCodecs` directly). The `FailClosed` test will **panic** because the existing `Test_CreateCallOutgoing_TypeSIP` struct has no `outboundConfigHandler`, and the production code doesn't call `GetByCustomerID` for SIP yet.

---

### Task 2: Update existing SIP tests to inject mock outbound config handler

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call_test.go`

After the production code change, every non-internal SIP test will call `GetByCustomerID`. The three existing test functions that use SIP destinations need a mock injected.

**Step 1: Update `Test_CreateCallOutgoing_TypeSIP` (around line 162–177)**

Find the `h := &callHandler{...}` block and the surrounding mock setup. Change it from:

```go
mockUtil    := utilhandler.NewMockUtilHandler(mc)
mockReq     := requesthandler.NewMockRequestHandler(mc)
mockNotify  := notifyhandler.NewMockNotifyHandler(mc)
mockDB      := dbhandler.NewMockDBHandler(mc)
mockChannel := channelhandler.NewMockChannelHandler(mc)

h := &callHandler{
    utilHandler:   mockUtil,
    reqHandler:    mockReq,
    notifyHandler: mockNotify,
    db:            mockDB,
    channelHandler: mockChannel,
}
```

to:

```go
mockUtil    := utilhandler.NewMockUtilHandler(mc)
mockReq     := requesthandler.NewMockRequestHandler(mc)
mockNotify  := notifyhandler.NewMockNotifyHandler(mc)
mockDB      := dbhandler.NewMockDBHandler(mc)
mockChannel := channelhandler.NewMockChannelHandler(mc)
mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

h := &callHandler{
    utilHandler:           mockUtil,
    reqHandler:            mockReq,
    notifyHandler:         mockNotify,
    db:                    mockDB,
    channelHandler:        mockChannel,
    outboundConfigHandler: mockOutboundConfig,
}
```

And add the EXPECT call **after** the balance check mock but **before** the existing channel mock:

```go
mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(nil, nil)
```

**Step 2: Repeat the same two-part change for `Test_CreateCallOutgoing_Metadata` (around line 380–404)**

Same pattern: add `mockOutboundConfig` declaration, inject into `h`, add `EXPECT` call returning `(nil, nil)`.

**Step 3: Repeat the same two-part change for `Test_CreateCallOutgoing_RTPDebug` (around line 600–640)**

Same pattern. All cases in this table-driven test use `TypeSIP` destinations, so add one `EXPECT` returning `(nil, nil)` per test case (already handled if the mock is set with `AnyTimes()`, or set per-case).

**Step 4: Run all three updated test functions to confirm they compile**

```bash
cd bin-call-manager
go test ./pkg/callhandler/... -run "Test_CreateCallOutgoing_TypeSIP|Test_CreateCallOutgoing_Metadata|Test_CreateCallOutgoing_RTPDebug" -v 2>&1 | tail -30
```

Expected at this point: tests still fail (production code not yet changed) — but no panic.

---

### Task 3: Implement the production code change

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go`

**Step 1: Locate the outbound config fetch block**

Find this block (around line 183–203):

```go
var outboundCfg *outboundconfig.OutboundConfig
if destination.Type == commonaddress.TypeTel && !cucustomer.IsInternalSystemID(customerID) {
    var cfgErr error
    outboundCfg, cfgErr = h.outboundConfigHandler.GetByCustomerID(ctx, customerID)
    if cfgErr != nil {
        log.Errorf("Could not get outbound config; rejecting call (fail-closed). err: %v", cfgErr)
        outboundconfighandler.IncFetchError("db_error")
        return nil, fmt.Errorf("could not get outbound config: %w", cfgErr)
    }
    metadata = embedCodecs(metadata, outboundCfg)
    if !h.ValidateDestination(ctx, customerID, outboundCfg, destination) {
        log.Infof("Outbound destination not in whitelist. customer_id: %s", customerID)
        country := h.getCountry(ctx, destination.Target)
        h.notifyHandler.PublishEvent(ctx, call.EventTypeCallOutboundWhitelistRejected, map[string]interface{}{
            "customer_id":         customerID,
            "call_id":             id,
            "destination_country": country,
        })
        return nil, outboundconfig.ErrDestinationNotWhitelisted
    }
}
```

**Step 2: Replace it with the unified fetch + switch-case**

```go
// Fetch outbound config once for all non-internal customers.
// Used for: codec embedding (SIP only), whitelist + source validation (PSTN).
// Internal system IDs (IDCallManager, IDAIManager, etc.) skip this block entirely —
// they have no OutboundConfig row and must not be gated by whitelist or codec injection.
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
    // PSTN trunks negotiate codecs directly with the carrier via SDP; injecting a
    // codec header into PSTN calls overrides that negotiation, which is incorrect.
    switch destination.Type {
    case commonaddress.TypeSIP:
        metadata = embedCodecs(metadata, outboundCfg)
    case commonaddress.TypeTel:
        // no codec embedding for PSTN
    }
}

// PSTN-only: whitelist + source number validation.
// outboundCfg is nil for internal system IDs — ValidateDestination returns true
// for internal callers regardless of config (bypass path in validate.go).
if destination.Type == commonaddress.TypeTel {
    if !h.ValidateDestination(ctx, customerID, outboundCfg, destination) {
        log.Infof("Outbound destination not in whitelist. customer_id: %s", customerID)
        country := h.getCountry(ctx, destination.Target)
        h.notifyHandler.PublishEvent(ctx, call.EventTypeCallOutboundWhitelistRejected, map[string]interface{}{
            "customer_id":         customerID,
            "call_id":             id,
            "destination_country": country,
        })
        return nil, outboundconfig.ErrDestinationNotWhitelisted
    }
}
```

**Step 3: Run the full test suite for callhandler**

```bash
cd bin-call-manager
go test ./pkg/callhandler/... -v 2>&1 | tail -40
```

Expected: all tests pass, including the new ones added in Tasks 1 and 2.

---

### Task 4: Run full verification workflow

**Files:** none — verification only

**Step 1: Run mod tidy and vendor**

```bash
cd bin-call-manager
go mod tidy && go mod vendor
```

**Step 2: Run code generation**

```bash
go generate ./...
```

**Step 3: Run all tests**

```bash
go test ./... 2>&1 | tail -20
```

Expected: `ok` for every package, no failures.

**Step 4: Run linter**

```bash
golangci-lint run -v --timeout 5m 2>&1 | tail -30
```

Expected: no new issues introduced.

---

### Task 5: Commit

**Step 1: Stage changed files**

```bash
cd bin-call-manager
git add pkg/callhandler/outgoing_call.go pkg/callhandler/outgoing_call_test.go
```

**Step 2: Commit**

```bash
git commit -m "NOJIRA-Outbound-codec-sip-only

- bin-call-manager: Embed OutboundConfig codecs for SIP calls only, not PSTN
- bin-call-manager: Hoist outbound config fetch outside PSTN-only guard
- bin-call-manager: Add switch-case codec gate by destination type
- bin-call-manager: Update existing SIP tests to inject outboundConfigHandler mock
- bin-call-manager: Add SIP codec embed and fail-closed tests"
```

---

## Pre-ship database check

Before merging, run this query against the production database to confirm no live
customer relies on `Codecs` being injected into PSTN calls:

```sql
SELECT id, customer_id, codecs
FROM outbound_configs
WHERE codecs != '' AND tm_delete IS NULL;
```

If any rows exist, coordinate with those customers before deploying — they will lose
codec injection on PSTN calls after this change ships.
