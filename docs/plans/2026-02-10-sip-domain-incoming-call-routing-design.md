# SIP Domain Incoming Call Routing

**Date:** 2026-02-10

## Problem

When a SIP INVITE arrives at `sip.voipbin.net`, the call-manager's `getDomainTypeIncomingCall()` function
does not recognize the `sip.voipbin.net` domain. It falls through to `domainTypeNone` and the call is
hung up with a "no route found" error.

The system currently supports four domain types for incoming calls:
- `conference.voipbin.net` → Conference routing
- `pstn.voipbin.net` → Number-based flow lookup
- `*.trunk.voipbin.net` → Trunk-based routing with temp Connect flow
- `*.registrar.voipbin.net` → Customer-scoped routing by destination type

There is no handler for `sip.voipbin.net` itself.

## Approach

Add a new `sip` domain type that routes incoming calls by looking up the destination phone number
in number-manager, identical to the PSTN path but implemented as a separate function for future
divergence.

The `domainTypeSIP` constant already exists in `start.go` (line 41) but is unused. We will wire it up.

## Changes

### 1. bin-common-handler/pkg/projectconfig/main.go

Add `DomainSIP` field to `ProjectConfig` struct, initialized as `"sip." + baseDomain`.

### 2. bin-common-handler/pkg/projectconfig/main_test.go

Add `expectedDomainSIP` field to all test cases in `Test_load` and verify it.

### 3. bin-call-manager/models/common/domain.go

Add `DomainSIP = projectconfig.Get().DomainSIP` variable.

### 4. bin-call-manager/pkg/callhandler/start.go

- Add `case common.DomainSIP` in `getDomainTypeIncomingCall()` returning `domainTypeSIP`
- Add `case domainTypeSIP` in `startContextIncomingCall()` routing to `startIncomingDomainTypeSIP()`

### 5. bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go (new file)

New file with `startIncomingDomainTypeSIP()` function. Same logic as `startIncomingDomainTypePSTN()`:
1. Get source/destination addresses as TypeTel
2. Look up destination number in number-manager
3. If not found, hangup with NO_ROUTE_DESTINATION
4. Call `startCallTypeFlow()` with the number's CustomerID and CallFlowID

### 6. bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_test.go (new file)

Test for `startIncomingDomainTypeSIP()`, following the same pattern as the trunk test file.

### 7. bin-call-manager/pkg/callhandler/start_test.go

Add test case for `sip.voipbin.net` domain in `Test_GetTypeContextIncomingCall`.

## Trade-offs

- The SIP handler is a copy of the PSTN handler. This is intentional — keeping them separate
  allows independent evolution without affecting existing PSTN behavior.
