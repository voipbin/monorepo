# Selective Codec for Outbound Calling — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let customers configure a preferred outbound codec list; call-manager embeds it as a `VBOUT-CODECS` SIP header on every outgoing INVITE so Kamailio can filter/transcode accordingly.

**Architecture:** Customer-level `OutboundCodecs` (admin-only metadata) is embedded into call metadata at call-creation time using the same guard pattern as `RTPDebug`. A per-call metadata key `codecs` overrides the customer default. Two small pure helper functions (`embedCustomerCodecs`, `setChannelVariableCodecs`) in a new `codec.go` file translate those values into Asterisk channel variables.

**Tech Stack:** Go, RabbitMQ RPC, Asterisk PJSIP channel variables, existing `bin-customer-manager` + `bin-call-manager` services.

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound`

---

## Task 1: Customer model — add `OutboundCodecs` field

**Files:**
- Modify: `bin-customer-manager/models/customer/metadata.go`
- Test: `bin-customer-manager/models/customer/metadata_test.go` (new file)

### Step 1: Write the failing test

Create `bin-customer-manager/models/customer/metadata_test.go`:

```go
package customer

import (
	"encoding/json"
	"testing"
)

func TestMetadata_OutboundCodecs_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect Metadata
	}{
		{
			"outbound_codecs set",
			`{"rtp_debug":false,"outbound_codecs":"PCMU,PCMA,G729"}`,
			Metadata{RTPDebug: false, OutboundCodecs: "PCMU,PCMA,G729"},
		},
		{
			"outbound_codecs empty",
			`{"rtp_debug":false,"outbound_codecs":""}`,
			Metadata{RTPDebug: false, OutboundCodecs: ""},
		},
		{
			"outbound_codecs absent (zero value)",
			`{"rtp_debug":true}`,
			Metadata{RTPDebug: true, OutboundCodecs: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Metadata
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if got != tt.expect {
				t.Errorf("Got %+v, expected %+v", got, tt.expect)
			}

			// round-trip: marshal back and unmarshal again
			b, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			var got2 Metadata
			if err := json.Unmarshal(b, &got2); err != nil {
				t.Fatalf("Second unmarshal failed: %v", err)
			}
			if got2 != tt.expect {
				t.Errorf("Round-trip mismatch. Got %+v, expected %+v", got2, tt.expect)
			}
		})
	}
}
```

### Step 2: Run to verify it fails

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound/bin-customer-manager
go test ./models/customer/... -run TestMetadata_OutboundCodecs_JSONRoundTrip -v
```

Expected: **FAIL** — `got.OutboundCodecs` will be empty where `"PCMU,PCMA,G729"` is expected (field doesn't exist yet).

### Step 3: Add the field

Edit `bin-customer-manager/models/customer/metadata.go`:

```go
// Metadata holds configuration flags for a customer.
// Can be updated by ProjectSuperAdmin via PUT /customers/{id}/metadata
// or by CustomerAdmin via PUT /customer/metadata.
type Metadata struct {
	RTPDebug       bool   `json:"rtp_debug"`
	OutboundCodecs string `json:"outbound_codecs"` // comma-separated codec preference, e.g. "PCMU,PCMA,G729"
}
```

### Step 4: Run to verify it passes

```bash
go test ./models/customer/... -run TestMetadata_OutboundCodecs_JSONRoundTrip -v
```

Expected: **PASS**

### Step 5: Verify full service

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all tests pass, lint clean.

### Step 6: Commit

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound
git add bin-customer-manager/models/customer/metadata.go bin-customer-manager/models/customer/metadata_test.go
git commit -m "NOJIRA-selective-codec-outbound

- bin-customer-manager: Add OutboundCodecs field to customer Metadata struct"
```

---

## Task 2: Call model — add `MetadataKeyCodecs` constant

**Files:**
- Modify: `bin-call-manager/models/call/metadata.go`
- Modify: `bin-call-manager/models/call/metadata_test.go`

### Step 1: Write the failing test

The existing `Test_ValidMetadataKeys_contains_all_declared_constants` enforces that every declared constant is registered. Add `MetadataKeyCodecs` to its `required` slice **before** declaring the constant — this makes the test fail with "undefined: MetadataKeyCodecs" (compile error that acts as a RED signal).

Edit `bin-call-manager/models/call/metadata_test.go` — add to the `required` slice:

```go
required := []MetadataKey{
    MetadataKeyRTPDebug,
    MetadataKeyRouteProviderIDs,
    MetadataKeySkipSourceValidation,
    MetadataKeyCodecs, // ← add this line
}
```

### Step 2: Run to verify it fails

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound/bin-call-manager
go test ./models/call/... -v
```

Expected: **compile error** — `undefined: MetadataKeyCodecs`. That is the RED signal.

### Step 3: Add the constant and register it

Edit `bin-call-manager/models/call/metadata.go`:

```go
// MetadataKeyCodecs sets the outbound codec preference for this call.
// Value is a comma-separated string, e.g. "PCMU,PCMA,G729".
// When present, call-manager adds a VBOUT-CODECS SIP header to the outgoing INVITE.
// Overrides the customer-level OutboundCodecs when set per-call.
// Creation-time only — set by CreateCallOutgoing from customer metadata,
// or supplied by the caller to override the customer default.
MetadataKeyCodecs MetadataKey = "codecs"
```

Add to `ValidMetadataKeys`:

```go
var ValidMetadataKeys = map[MetadataKey]bool{
    MetadataKeyRTPDebug:             true,
    MetadataKeyRouteProviderIDs:     true,
    MetadataKeySkipSourceValidation: true,
    MetadataKeyCodecs:               true, // ← add this line
}
```

### Step 4: Run to verify it passes

```bash
go test ./models/call/... -v
```

Expected: **PASS**

### Step 5: Commit

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound
git add bin-call-manager/models/call/metadata.go bin-call-manager/models/call/metadata_test.go
git commit -m "NOJIRA-selective-codec-outbound

- bin-call-manager: Add MetadataKeyCodecs constant and register in ValidMetadataKeys"
```

---

## Task 3: Common SIP constants — add `SIPHeaderCodecs`

**Files:**
- Modify: `bin-call-manager/models/common/sip.go`

No new test needed — this is a constant addition with no logic. The downstream tests in Task 5 will exercise it.

### Step 1: Add the constant

Edit `bin-call-manager/models/common/sip.go`:

```go
const (
    SIPHeaderCallID       = "VB-CALL-ID"
    SIPHeaderConfbridgeID = "VB-CONFBRIDGE-ID"
    SIPHeaderDirection    = "VB-DIRECTION"

    SIPHeaderSDPTransport = "VBOUT-SDP_Transport" // transport for outgoing call
    SIPHeaderCodecs       = "VBOUT-CODECS"        // outbound codec preference for Kamailio
)
```

### Step 2: Commit

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound
git add bin-call-manager/models/common/sip.go
git commit -m "NOJIRA-selective-codec-outbound

- bin-call-manager: Add SIPHeaderCodecs constant to common SIP headers"
```

---

## Task 4: Reserved tech headers — block provider override

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/tech_headers.go`
- Modify: `bin-call-manager/pkg/callhandler/tech_headers_test.go`

### Step 1: Write the failing test

Add a test case to `Test_mergeTechHeaders` in `tech_headers_test.go` that asserts `VBOUT-CODECS` is blocked:

```go
{
    "reserved VBOUT-CODECS header is blocked",

    map[string]string{},
    map[string]string{"VBOUT-CODECS": "PCMU"},

    map[string]string{},
    0,
    1,
},
```

### Step 2: Run to verify it fails

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound/bin-call-manager
go test ./pkg/callhandler/... -run Test_mergeTechHeaders -v
```

Expected: **FAIL** — the `VBOUT-CODECS` entry will be applied (applied=1, skipped=0) instead of skipped.

### Step 3: Add to `reservedTechHeaderKeys`

Edit `bin-call-manager/pkg/callhandler/tech_headers.go`, add to the map:

```go
var reservedTechHeaderKeys = map[string]struct{}{
    "PJSIP_HEADER(add,P-Asserted-Identity)": {},
    "PJSIP_HEADER(add,Privacy)":             {},
    "PJSIP_HEADER(add,VBOUT-SDP_Transport)": {},
    "PJSIP_HEADER(add,VB-CALL-ID)":          {},
    "PJSIP_HEADER(add,VB-CONFBRIDGE-ID)":    {},
    "PJSIP_HEADER(add,VB-DIRECTION)":        {},
    "PJSIP_HEADER(add,VBOUT-CODECS)":        {}, // ← add: prevent provider override
    "CALLERID(name)":                        {},
    "CALLERID(num)":                         {},
    "CALLERID(pres)":                        {},
}
```

### Step 4: Run to verify it passes

```bash
go test ./pkg/callhandler/... -run Test_mergeTechHeaders -v
```

Expected: **PASS**

### Step 5: Commit

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound
git add bin-call-manager/pkg/callhandler/tech_headers.go bin-call-manager/pkg/callhandler/tech_headers_test.go
git commit -m "NOJIRA-selective-codec-outbound

- bin-call-manager: Add VBOUT-CODECS to reserved tech header keys"
```

---

## Task 5: New `codec.go` — implement helper functions

**Files:**
- Create: `bin-call-manager/pkg/callhandler/codec.go`
- Create: `bin-call-manager/pkg/callhandler/codec_test.go`

### Step 1: Write the failing tests

Create `bin-call-manager/pkg/callhandler/codec_test.go`:

```go
package callhandler

import (
	"testing"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/common"
)

func Test_embedCustomerCodecs(t *testing.T) {
	tests := []struct {
		name           string
		metadata       map[string]any
		outboundCodecs string
		expectCodecs   string // expected value of metadata[call.MetadataKeyCodecs]; "" means key absent
		expectSet      bool   // whether key should be present in result
	}{
		{
			"sets from customer when metadata empty",
			map[string]any{},
			"PCMU,PCMA,G729",
			"PCMU,PCMA,G729",
			true,
		},
		{
			"per-call override wins — customer value not applied",
			map[string]any{call.MetadataKeyCodecs: "G722"},
			"PCMU,PCMA",
			"G722",
			true,
		},
		{
			"empty customer value — key not added",
			map[string]any{},
			"",
			"",
			false,
		},
		{
			"nil metadata with customer value — creates map",
			nil,
			"PCMU",
			"PCMU",
			true,
		},
		{
			"nil metadata with empty customer value — returns nil",
			nil,
			"",
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := embedCustomerCodecs(tt.metadata, tt.outboundCodecs)

			val, present := got[call.MetadataKeyCodecs]
			if present != tt.expectSet {
				t.Errorf("Key presence: got %v, expected %v", present, tt.expectSet)
			}
			if tt.expectSet {
				if s, ok := val.(string); !ok || s != tt.expectCodecs {
					t.Errorf("Codec value: got %v, expected %q", val, tt.expectCodecs)
				}
			}
		})
	}
}

func Test_setChannelVariableCodecs(t *testing.T) {
	headerKey := "PJSIP_HEADER(add," + common.SIPHeaderCodecs + ")"

	tests := []struct {
		name         string
		metadata     map[string]any
		expectHeader string // expected value; "" means key absent
		expectSet    bool
	}{
		{
			"adds header when codecs set",
			map[string]any{call.MetadataKeyCodecs: "PCMU,PCMA,G729"},
			"PCMU,PCMA,G729",
			true,
		},
		{
			"no header when codecs key absent",
			map[string]any{},
			"",
			false,
		},
		{
			"no header when codecs value is empty string",
			map[string]any{call.MetadataKeyCodecs: ""},
			"",
			false,
		},
		{
			"CRLF in value rejected — no header",
			map[string]any{call.MetadataKeyCodecs: "PCMU\r\nX-Inject: evil"},
			"",
			false,
		},
		{
			"CR alone in value rejected",
			map[string]any{call.MetadataKeyCodecs: "PCMU\rPCMA"},
			"",
			false,
		},
		{
			"LF alone in value rejected",
			map[string]any{call.MetadataKeyCodecs: "PCMU\nPCMA"},
			"",
			false,
		},
		{
			"non-string metadata value — no header",
			map[string]any{call.MetadataKeyCodecs: 42},
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variables := map[string]string{}
			setChannelVariableCodecs(variables, tt.metadata)

			val, present := variables[headerKey]
			if present != tt.expectSet {
				t.Errorf("Header presence: got %v, expected %v. variables=%v", present, tt.expectSet, variables)
			}
			if tt.expectSet && val != tt.expectHeader {
				t.Errorf("Header value: got %q, expected %q", val, tt.expectHeader)
			}
		})
	}
}
```

### Step 2: Run to verify it fails

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound/bin-call-manager
go test ./pkg/callhandler/... -run "Test_embedCustomerCodecs|Test_setChannelVariableCodecs" -v
```

Expected: **compile error** — `embedCustomerCodecs` and `setChannelVariableCodecs` are undefined.

### Step 3: Create `codec.go`

Create `bin-call-manager/pkg/callhandler/codec.go`:

```go
package callhandler

import (
	"strings"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/common"
)

// embedCustomerCodecs copies OutboundCodecs from customer metadata into call
// metadata if the call does not already carry a codecs override.
// Returns the (possibly newly allocated) metadata map.
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

// setChannelVariableCodecs adds the VBOUT-CODECS SIP header to outgoing channel
// variables if a codec preference is present in call metadata.
// CRLF characters in the value are rejected silently (header-injection defence).
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

### Step 4: Run to verify it passes

```bash
go test ./pkg/callhandler/... -run "Test_embedCustomerCodecs|Test_setChannelVariableCodecs" -v
```

Expected: **PASS** (all 12 cases green).

### Step 5: Commit

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound
git add bin-call-manager/pkg/callhandler/codec.go bin-call-manager/pkg/callhandler/codec_test.go
git commit -m "NOJIRA-selective-codec-outbound

- bin-call-manager: Add codec.go with embedCustomerCodecs and setChannelVariableCodecs helpers"
```

---

## Task 6: Wire helpers into `outgoing_call.go`

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go`

No new tests needed — the helpers are fully tested in Task 5. This task is two one-line wiring changes.

### Step 1: Wire `embedCustomerCodecs` in `CreateCallOutgoing`

In `outgoing_call.go`, find the `RTPDebug` guard block (around line 155–165):

```go
if _, alreadySet := metadata[call.MetadataKeyRTPDebug]; !alreadySet {
    if cu.Metadata.RTPDebug {
        if metadata == nil {
            metadata = map[string]any{}
        }
        metadata[call.MetadataKeyRTPDebug] = true
    }
}
```

Add immediately after it:

```go
metadata = embedCustomerCodecs(metadata, cu.Metadata.OutboundCodecs)
```

### Step 2: Wire `setChannelVariableCodecs` in `createChannelOutgoing`

Find the callerID setup block in `createChannelOutgoing` (around line 571–573):

```go
if err := setChannelVariablesCallerID(channelVariables, c, anonymous); err != nil {
    log.Errorf("Could not set caller ID variables. err: %v", err)
    return err
}
```

Add immediately after it:

```go
setChannelVariableCodecs(channelVariables, c.Metadata)
```

### Step 3: Run full callhandler tests

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound/bin-call-manager
go test ./pkg/callhandler/... -v
```

Expected: **PASS** — no regressions.

### Step 4: Commit

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound
git add bin-call-manager/pkg/callhandler/outgoing_call.go
git commit -m "NOJIRA-selective-codec-outbound

- bin-call-manager: Wire embedCustomerCodecs and setChannelVariableCodecs into outgoing call path"
```

---

## Task 7: Verify `bin-call-manager` end-to-end

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound/bin-call-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps exit 0. Commit any `go.mod`/`go.sum` changes if `go mod tidy` produced a diff:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound
git add bin-call-manager/go.mod bin-call-manager/go.sum
git diff --cached --quiet || git commit -m "NOJIRA-selective-codec-outbound

- bin-call-manager: go mod tidy after codec feature addition"
```

---

## Task 8: Verify `bin-customer-manager` end-to-end

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound/bin-customer-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps exit 0. Commit any `go.mod`/`go.sum` changes:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound
git add bin-customer-manager/go.mod bin-customer-manager/go.sum
git diff --cached --quiet || git commit -m "NOJIRA-selective-codec-outbound

- bin-customer-manager: go mod tidy after OutboundCodecs addition"
```

---

## Task 9: Pre-PR checks

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-selective-codec-outbound

# 1. Fetch latest main and check for conflicts
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
# Expected: no output (no conflicts)

# 2. Review what changed on main since branch point
git log --oneline HEAD..origin/main

# 3. Confirm all commits look right
git log --oneline origin/main..HEAD
```

If conflicts exist: rebase onto main, resolve, re-run verifications from Task 7 and 8.

---

## Summary of all changed files

| File | Change |
|---|---|
| `bin-customer-manager/models/customer/metadata.go` | Add `OutboundCodecs string` field |
| `bin-customer-manager/models/customer/metadata_test.go` | New — JSON round-trip test |
| `bin-call-manager/models/call/metadata.go` | Add `MetadataKeyCodecs` constant + `ValidMetadataKeys` entry |
| `bin-call-manager/models/call/metadata_test.go` | Add `MetadataKeyCodecs` to required list |
| `bin-call-manager/models/common/sip.go` | Add `SIPHeaderCodecs` constant |
| `bin-call-manager/pkg/callhandler/tech_headers.go` | Add `PJSIP_HEADER(add,VBOUT-CODECS)` to reserved map |
| `bin-call-manager/pkg/callhandler/tech_headers_test.go` | Add reserved-header test case |
| `bin-call-manager/pkg/callhandler/codec.go` | New — `embedCustomerCodecs`, `setChannelVariableCodecs` |
| `bin-call-manager/pkg/callhandler/codec_test.go` | New — full test coverage for both helpers |
| `bin-call-manager/pkg/callhandler/outgoing_call.go` | Two one-line wiring calls |
