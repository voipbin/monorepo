# Provider Codec Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `codecs` field to the Provider model so PSTN calls are SDP-constrained to the carrier's supported codec list, injected at dial time via the `VBOUT-CODECS` SIP header.

**Architecture:** Provider gains a `codecs string` DB column and API field. In `bin-call-manager`, `getDialURI` and its three variants change to return a `dialTarget` struct carrying the provider's codecs, which `createChannelOutgoing` injects into per-attempt channel variables (not persisted call metadata). Provider codecs apply only to PSTN (`TypeTel`) paths; SIP-to-SIP paths are unchanged.

**Tech Stack:** Go 1.22, MySQL (Alembic migrations via Python), RabbitMQ RPC, Asterisk/PJSIP, OpenAPI 3, Sphinx RST docs.

**Design doc:** `docs/plans/2026-05-09-provider-codec-support-design.md`

**Verification command (run in each service dir after every change):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Deployment note:** DB migration MUST be applied before deploying any code that references the new `codecs` column. The migration adds `codecs VARCHAR(255) NOT NULL DEFAULT ''` to `route_providers`.

---

## Task 1: Database Migration

**Files:**
- Modify: `bin-dbscheme-manager/` (run alembic to generate, then edit)

**Step 1: Generate migration file**
```bash
cd bin-dbscheme-manager
alembic -c alembic.ini revision -m "add_codecs_to_route_providers"
```
Expected: new file created at `alembic/versions/<hash>_add_codecs_to_route_providers.py`

**Step 2: Edit the generated migration file**

Open the newly created file and fill in `upgrade()` and `downgrade()`:
```python
def upgrade():
    op.execute("""
        ALTER TABLE route_providers
        ADD COLUMN codecs VARCHAR(255) NOT NULL DEFAULT ''
    """)

def downgrade():
    op.execute("""
        ALTER TABLE route_providers
        DROP COLUMN codecs
    """)
```

**Step 3: Commit**
```bash
git add alembic/versions/
git commit -m "NOJIRA-Provider-codec-support

- bin-dbscheme-manager: Add codecs column to route_providers table"
```

---

## Task 2: Route-Manager — Model and Field Constants

**Files:**
- Modify: `bin-route-manager/models/provider/provider.go`
- Modify: `bin-route-manager/models/provider/field.go`

**Step 1: Add `Codecs` field to Provider struct**

In `provider.go`, inside `type Provider struct { ... }`, add after the `Metadata` field:
```go
Codecs string `json:"codecs" db:"codecs"` // comma-separated; empty = server default
```

**Step 2: Add `FieldCodecs` constant**

In `field.go`, add:
```go
FieldCodecs Field = "codecs"
```

**Step 3: Run tests to verify no breakage**
```bash
cd bin-route-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all tests pass (struct change is backward-compatible; JSON unmarshal tolerates new fields)

**Step 4: Commit**
```bash
git add models/provider/
git commit -m "NOJIRA-Provider-codec-support

- bin-route-manager: Add Codecs field to Provider model and FieldCodecs constant"
```

---

## Task 3: Route-Manager — Webhook

**Files:**
- Modify: `bin-route-manager/models/provider/webhook.go`
- Modify: `bin-route-manager/models/provider/webhook_test.go`

**Step 1: Read the current webhook.go**

Open `models/provider/webhook.go` and find:
- The `WebhookMessage` struct definition
- The `ConvertWebhookMessage()` function

**Step 2: Write a failing test**

In `webhook_test.go`, add a test case verifying the `Codecs` field is copied:
```go
func Test_ConvertWebhookMessage_Codecs(t *testing.T) {
    p := &Provider{
        ID:     uuid.Must(uuid.NewV4()),
        Codecs: "PCMU,PCMA",
        // fill other required fields to avoid zero-value noise
    }
    got := ConvertWebhookMessage(p)
    if got.Codecs != "PCMU,PCMA" {
        t.Errorf("expected Codecs %q, got %q", "PCMU,PCMA", got.Codecs)
    }
}
```

**Step 3: Run test to verify it fails**
```bash
cd bin-route-manager
go test ./models/provider/... -run Test_ConvertWebhookMessage_Codecs -v
```
Expected: compile error — `got.Codecs` undefined

**Step 4: Add `Codecs` to `WebhookMessage` struct**

In `webhook.go`, add to `WebhookMessage`:
```go
Codecs string `json:"codecs"`
```

**Step 5: Add `Codecs` copy to `ConvertWebhookMessage()`**

In `ConvertWebhookMessage`, add:
```go
res.Codecs = p.Codecs
```

**Step 6: Run test to verify it passes**
```bash
go test ./models/provider/... -run Test_ConvertWebhookMessage_Codecs -v
```
Expected: PASS

**Step 7: Run full verification**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 8: Commit**
```bash
git add models/provider/
git commit -m "NOJIRA-Provider-codec-support

- bin-route-manager: Add Codecs to WebhookMessage and ConvertWebhookMessage"
```

---

## Task 4: Route-Manager — Codec Validation

**Files:**
- Create: `bin-route-manager/pkg/providerhandler/validate.go`
- Create: `bin-route-manager/pkg/providerhandler/validate_test.go`

**Step 1: Write failing tests first**

Create `validate_test.go`:
```go
package providerhandler

import (
    "testing"
)

func Test_validateCodecs(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {name: "empty is valid", input: "", wantErr: false},
        {name: "single codec", input: "PCMU", wantErr: false},
        {name: "comma separated", input: "PCMU,PCMA,G729", wantErr: false},
        {name: "too long", input: string(make([]byte, 256)), wantErr: true},
        {name: "CRLF injection CR", input: "PCMU\rPCMA", wantErr: true},
        {name: "CRLF injection LF", input: "PCMU\nPCMA", wantErr: true},
        {name: "open paren rejected", input: "PCMU(bad", wantErr: true},
        {name: "close paren rejected", input: "PCMU)bad", wantErr: true},
        {name: "double comma rejected", input: "PCMU,,PCMA", wantErr: true},
        {name: "whitespace trimmed and valid", input: " PCMU , PCMA ", wantErr: false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := validateCodecs(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateCodecs(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
            }
            if err == nil && tt.input != "" {
                _ = got // normalized output; main assertion is no error
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**
```bash
cd bin-route-manager
go test ./pkg/providerhandler/... -run Test_validateCodecs -v
```
Expected: compile error — `validateCodecs` undefined

**Step 3: Implement `validateCodecs` in validate.go**

Create `validate.go`:
```go
package providerhandler

import (
    "fmt"
    "strings"
)

// validateCodecs checks and normalizes a comma-separated codec string.
// Returns the trimmed string and nil on success, or an error on invalid input.
// Rules: max 255 chars, no CRLF, no parens, no empty list elements.
func validateCodecs(s string) (string, error) {
    if len(s) > 255 {
        return "", fmt.Errorf("codecs string exceeds 255 characters")
    }
    if strings.ContainsAny(s, "\r\n") {
        return "", fmt.Errorf("codecs must not contain CRLF characters")
    }
    if strings.ContainsAny(s, "()") {
        return "", fmt.Errorf("codecs must not contain parentheses")
    }
    if s == "" {
        return "", nil
    }
    parts := strings.Split(s, ",")
    normalized := make([]string, 0, len(parts))
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p == "" {
            return "", fmt.Errorf("codecs must not contain empty list elements (double comma)")
        }
        normalized = append(normalized, p)
    }
    return strings.Join(normalized, ","), nil
}
```

**Step 4: Run tests to verify they pass**
```bash
go test ./pkg/providerhandler/... -run Test_validateCodecs -v
```
Expected: PASS

**Step 5: Run full verification**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**
```bash
git add pkg/providerhandler/
git commit -m "NOJIRA-Provider-codec-support

- bin-route-manager: Add validateCodecs helper in providerhandler"
```

---

## Task 5: Route-Manager — Provider Handler (Create and Update)

**Files:**
- Modify: `bin-route-manager/pkg/providerhandler/main.go`
- Modify: `bin-route-manager/pkg/providerhandler/provider.go`
- Modify: `bin-route-manager/pkg/providerhandler/mock_main.go` (regenerated by go generate)

**Step 1: Add `codecs string` to `Create` in the interface**

In `main.go`, find `Create(ctx context.Context, ...)` in the `ProviderHandler` interface and add `codecs string` as the last parameter before the closing `)`.

**Step 2: Add `codecs string` to `Update` in the interface**

Same file — find `Update(ctx context.Context, id uuid.UUID, ...)` and add `codecs string` as the last parameter.

**Step 3: Update `Create` implementation in provider.go**

In `provider.go`, `Create` function:
- Add `codecs string` to the signature
- Before creating the provider, add validation:
  ```go
  normalizedCodecs, err := validateCodecs(codecs)
  if err != nil {
      return nil, cerrors.InvalidArgument(
          commonoutline.ServiceNameRouteManager,
          "INVALID_CODECS",
          fmt.Sprintf("Invalid codecs value: %v", err),
      )
  }
  ```
- Set `Codecs: normalizedCodecs` in the `&provider.Provider{...}` literal

**Step 4: Update `Update` implementation in provider.go**

In `provider.go`, `Update` function:
- Add `codecs string` to the signature
- Add validation (same pattern as Create)
- Add to the `fields` map:
  ```go
  provider.FieldCodecs: normalizedCodecs,
  ```

**Step 5: Regenerate mocks**
```bash
cd bin-route-manager
go generate ./pkg/providerhandler/...
```
Expected: `mock_main.go` regenerated with new `Create`/`Update` signatures

**Step 6: Run full verification**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Note: listen handler tests will fail (caller updated in next task). Fix any compile errors from the interface change.

**Step 7: Commit**
```bash
git add pkg/providerhandler/
git commit -m "NOJIRA-Provider-codec-support

- bin-route-manager: Add codecs param to providerhandler Create and Update with validation"
```

---

## Task 6: Route-Manager — Wire DTOs and Listen Handler

**Files:**
- Modify: `bin-route-manager/pkg/listenhandler/models/request/providers.go`
- Modify: `bin-route-manager/pkg/listenhandler/v1_providers.go` (or equivalent)

**Step 1: Add `Codecs` to wire DTOs**

In `listenhandler/models/request/providers.go`, add to both structs:
```go
// V1DataProvidersPost
Codecs string `json:"codecs"`

// V1DataProvidersIDPut
Codecs string `json:"codecs"`
```
`V1DataProvidersSetupPost` is intentionally NOT changed.

**Step 2: Find the listen handler provider create and update**
```bash
grep -n "providerhandler\.\|ProviderCreate\|ProviderUpdate" bin-route-manager/pkg/listenhandler/v1_providers.go | head -20
```

**Step 3: Update listen handler to pass `Codecs` to handler**

In the POST handler: pass `req.Codecs` as the last argument to `h.providerHandler.Create(...)`.
In the PUT handler: pass `req.Codecs` as the last argument to `h.providerHandler.Update(...)`.

**Step 4: Run full verification**
```bash
cd bin-route-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**
```bash
git add pkg/listenhandler/
git commit -m "NOJIRA-Provider-codec-support

- bin-route-manager: Add codecs to wire DTOs and listen handler for Create/Update"
```

---

## Task 7: Common-Handler — RPC Wrappers

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/route_providers.go`

**Step 1: Add `codecs string` to `RouteV1ProviderCreate`**

In `route_providers.go`, find `RouteV1ProviderCreate` and:
- Add `codecs string` as the last parameter
- Add `Codecs: codecs` to the `V1DataProvidersPost{...}` struct literal

**Step 2: Add `codecs string` to `RouteV1ProviderUpdate`**

Find `RouteV1ProviderUpdate` and:
- Add `codecs string` as the last parameter
- Add `Codecs: codecs` to the `V1DataProvidersIDPut{...}` struct literal

**Step 3: Update the interface in main.go of requesthandler**

Find where `RouteV1ProviderCreate` and `RouteV1ProviderUpdate` are declared in the `RequestHandler` interface (likely `bin-common-handler/pkg/requesthandler/main.go`) and add `codecs string` to their signatures there too.

**Step 4: Regenerate the common-handler mock**
```bash
cd bin-common-handler
go generate ./pkg/requesthandler/...
```

**Step 5: Run full verification**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Note: `bin-api-manager` and any other callers will now have compile errors — fix in next task.

**Step 6: Commit**
```bash
git add pkg/requesthandler/
git commit -m "NOJIRA-Provider-codec-support

- bin-common-handler: Add codecs param to RouteV1ProviderCreate and RouteV1ProviderUpdate"
```

---

## Task 8: OpenAPI Spec

**Files:**
- Modify: `bin-openapi-manager/` — find and update the Provider schema YAML/JSON

**Step 1: Find the Provider schema**
```bash
grep -rn "tech_prefix\|tech_postfix" bin-openapi-manager/ --include="*.yaml" --include="*.json" | head -10
```

**Step 2: Add `codecs` to the Provider response schema**

In the Provider object definition, add:
```yaml
codecs:
  type: string
  description: >
    Comma-separated codec list offered to this provider (e.g. "PCMU,PCMA").
    Empty means server-default negotiation.
    Applied to outgoing PSTN dial attempts only; has no effect on SIP-to-SIP traffic.
  example: "PCMU,PCMA"
```

**Step 3: Add `codecs` to ProviderCreateRequest and ProviderUpdateRequest**

Same field definition in both request body schemas (optional — no `required` annotation).

**Step 4: Regenerate API types**
```bash
cd bin-openapi-manager
go generate ./...
# OR check if there's a Makefile target
make generate 2>/dev/null || go generate ./...
```

**Step 5: Run full verification**
```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**
```bash
git add .
git commit -m "NOJIRA-Provider-codec-support

- bin-openapi-manager: Add codecs field to Provider schemas"
```

---

## Task 9: API Manager — Handler Updates

**Files:**
- Modify: `bin-api-manager/server/providers.go`
- Modify: `bin-api-manager/gens/` (regenerated from openapi-manager)

**Step 1: Regenerate API types from updated spec**
```bash
cd bin-api-manager
go generate ./...
```
Expected: generated types now include `Codecs` field on provider request/response types.

**Step 2: Update `PostProviders` handler**

In `server/providers.go`, find `PostProviders`. After parsing `req`, pass `req.Codecs` (or `""` if nil pointer — check the generated type) as the last argument to `h.serviceHandler.ProviderCreate(...)`.

If the generated type uses `*string` for optional fields:
```go
codecs := ""
if req.Codecs != nil {
    codecs = *req.Codecs
}
```

**Step 3: Update `PutProvidersId` handler**

Same pattern — pass `codecs` to `h.serviceHandler.ProviderUpdate(...)`.

**Step 4: Run full verification**
```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**
```bash
git add .
git commit -m "NOJIRA-Provider-codec-support

- bin-api-manager: Pass codecs field in PostProviders and PutProvidersId handlers"
```

---

## Task 10: Call-Manager — `dialTarget` Struct and `setProviderCodecs`

**Files:**
- Create: `bin-call-manager/pkg/callhandler/dial_target.go`
- Modify: `bin-call-manager/pkg/callhandler/codec.go`
- Create/Modify: `bin-call-manager/pkg/callhandler/codec_test.go`

**Step 1: Write failing tests for `setProviderCodecs`**

In `codec_test.go`, add:
```go
func Test_setProviderCodecs(t *testing.T) {
    tests := []struct {
        name      string
        codecs    string
        wantKey   bool
        wantValue string
    }{
        {
            name:      "codecs set - header written",
            codecs:    "PCMU,PCMA",
            wantKey:   true,
            wantValue: "PCMU,PCMA",
        },
        {
            name:    "empty codecs - no header",
            codecs:  "",
            wantKey: false,
        },
        {
            name:    "CRLF in codecs - no header",
            codecs:  "PCMU\r\nPCMA",
            wantKey: false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            variables := map[string]string{}
            setProviderCodecs(variables, tt.codecs)
            key := "PJSIP_HEADER(add," + common.SIPHeaderCodecs + ")"
            got, exists := variables[key]
            if exists != tt.wantKey {
                t.Errorf("key present=%v, want %v", exists, tt.wantKey)
            }
            if tt.wantKey && got != tt.wantValue {
                t.Errorf("value=%q, want %q", got, tt.wantValue)
            }
        })
    }
}

func Test_setProviderCodecs_PrecedenceOverMetadata(t *testing.T) {
    // Provider codecs win over per-call metadata codecs (setChannelVariableCodecs runs first).
    variables := map[string]string{}
    metadata := map[string]any{call.MetadataKeyCodecs: "OPUS"}
    setChannelVariableCodecs(variables, metadata) // writes OPUS
    setProviderCodecs(variables, "PCMU")          // must overwrite with PCMU

    key := "PJSIP_HEADER(add," + common.SIPHeaderCodecs + ")"
    if got := variables[key]; got != "PCMU" {
        t.Errorf("expected provider codec PCMU to win, got %q", got)
    }
}
```

**Step 2: Run tests to verify they fail**
```bash
cd bin-call-manager
go test ./pkg/callhandler/... -run Test_setProviderCodecs -v
```
Expected: compile error — `setProviderCodecs` undefined

**Step 3: Create `dial_target.go`**
```go
package callhandler

import "github.com/gofrs/uuid"

// dialTarget carries per-attempt dial parameters resolved at channel-creation time.
// Each call to createChannelOutgoing produces a fresh dialTarget, so failover
// to a different provider automatically picks up the new provider's Codecs.
type dialTarget struct {
    URI         string
    TechHeaders map[string]string
    Codecs      string    // provider codec list; empty = no constraint
    ProviderID  uuid.UUID // uuid.Nil for non-provider (SIP) paths
}
```

**Step 4: Add `setProviderCodecs` to `codec.go`**

Add to `bin-call-manager/pkg/callhandler/codec.go`:
```go
// setProviderCodecs writes the VBOUT-CODECS channel variable for PSTN dial
// attempts. It is separate from setChannelVariableCodecs (which reads from
// call metadata for SIP paths) so that provider codecs stay scoped to a
// single dial attempt and do not enter persisted call metadata.
//
// Precedence: setProviderCodecs runs after setChannelVariableCodecs in
// createChannelOutgoing, so provider codecs win on map-key collision.
// The normal case has no collision: OutboundConfig codecs are only embedded
// into call metadata for TypeSIP destinations (not TypeTel), so
// setChannelVariableCodecs writes nothing for PSTN calls. However, an operator
// could set MetadataKeyCodecs directly in per-call PSTN metadata — if they do,
// provider codecs deliberately overwrite that value, because the provider's
// accepted codec list is the authoritative constraint for the carrier trunk.
func setProviderCodecs(variables map[string]string, codecs string) {
    if codecs == "" {
        return
    }
    if strings.ContainsAny(codecs, "\r\n") {
        return
    }
    variables["PJSIP_HEADER(add,"+common.SIPHeaderCodecs+")"] = codecs
}
```

**Step 5: Run tests to verify they pass**
```bash
go test ./pkg/callhandler/... -run Test_setProviderCodecs -v
```
Expected: PASS

**Step 6: Run full verification**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**
```bash
git add pkg/callhandler/
git commit -m "NOJIRA-Provider-codec-support

- bin-call-manager: Add dialTarget struct and setProviderCodecs function"
```

---

## Task 11: Call-Manager — `getDialURI*` Signature Changes

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go`

**Step 1: Update `getDialURITel` to return `(*dialTarget, error)`**

Find the function at line ~357. Change:
```go
func (h *callHandler) getDialURITel(ctx context.Context, c *call.Call) (string, map[string]string, error) {
```
to:
```go
func (h *callHandler) getDialURITel(ctx context.Context, c *call.Call) (*dialTarget, error) {
```

Update the body:
- Remove `return res, pr.TechHeaders, nil`
- Replace with:
  ```go
  return &dialTarget{
      URI:         res,
      TechHeaders: pr.TechHeaders,
      Codecs:      pr.Codecs,
      ProviderID:  pr.ID,
  }, nil
  ```
- Update the error returns from `return "", nil, err` to `return nil, err`

**Step 2: Update `getDialURISIP` to return `(*dialTarget, error)`**

Change signature and body:
```go
func (h *callHandler) getDialURISIP(ctx context.Context, c *call.Call) (*dialTarget, error) {
    // ... existing URI construction ...
    return &dialTarget{URI: res}, nil
}
```
Error returns: `return nil, err`

**Step 3: Update `getDialURISIPDirect` to return `(*dialTarget, error)`**

Same pattern — build URI, return `&dialTarget{URI: res}`, errors return `nil, err`.

**Step 4: Update `getDialURI` dispatcher**

Change signature:
```go
func (h *callHandler) getDialURI(ctx context.Context, c *call.Call) (*dialTarget, error) {
```
The switch cases now return the result of the sub-functions directly (no unpacking needed since they all return `(*dialTarget, error)`).

**Step 5: Verify compile**
```bash
cd bin-call-manager
go build ./pkg/callhandler/...
```
Expected: compile errors in `createChannelOutgoing` (uses old signature) — fix in next task.

**Step 6: Commit (even if compile fails — next task fixes callers)**
```bash
git add pkg/callhandler/outgoing_call.go
git commit -m "NOJIRA-Provider-codec-support

- bin-call-manager: Change getDialURI and variants to return dialTarget struct"
```

---

## Task 12: Call-Manager — `createChannelOutgoing` Integration

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go`

**Step 1: Update `createChannelOutgoing` to use `*dialTarget`**

Find `createChannelOutgoing` at line ~585. Replace:
```go
dialURI, techHeaders, err := h.getDialURI(ctx, c)
if err != nil { ... }
channelVariables := map[string]string{}
techApplied, techSkipped := mergeTechHeaders(channelVariables, techHeaders, log)
transport := getDestinationTransport(dialURI)
```
with:
```go
target, err := h.getDialURI(ctx, c)
if err != nil { ... }
channelVariables := map[string]string{}
techApplied, techSkipped := mergeTechHeaders(channelVariables, target.TechHeaders, log)
transport := getDestinationTransport(target.URI)
```

After `setChannelVariableCodecs(channelVariables, c.Metadata)`, add:
```go
setProviderCodecs(channelVariables, target.Codecs)
if target.Codecs != "" {
    log.Debugf("Provider codec applied for dial attempt. provider_id: %s, codecs: %s",
        target.ProviderID, target.Codecs)
}
```

Update all remaining references from `dialURI` to `target.URI` (e.g., in `StartChannel` call and `appArgs`).

**Step 2: Verify compile**
```bash
cd bin-call-manager
go build ./...
```
Expected: clean build

**Step 3: Run full test suite**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Write integration tests for the PSTN codec injection**

In `outgoing_call_test.go`, add tests (using existing mock patterns in the file):

```go
// Test_createChannelOutgoing_providerCodecs verifies that provider codecs are
// injected into channel variables for PSTN calls.
func Test_createChannelOutgoing_providerCodecs(t *testing.T) {
    // ... set up handler with mocked reqHandler returning provider with Codecs="PCMU" ...
    // Assert: StartChannel was called with variables containing
    //   "PJSIP_HEADER(add,VBOUT-CODECS)" = "PCMU"
}

// Test_createChannelOutgoing_noCodecs verifies no VBOUT-CODECS header when provider has no codecs.
func Test_createChannelOutgoing_noCodecs(t *testing.T) {
    // ... provider with Codecs="" ...
    // Assert: VBOUT-CODECS key is absent from variables
}
```

Follow the existing test patterns in `outgoing_call_test.go` for mock setup.

**Step 5: Run all tests**
```bash
go test ./pkg/callhandler/... -v
```
Expected: all pass

**Step 6: Run full verification**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**
```bash
git add pkg/callhandler/
git commit -m "NOJIRA-Provider-codec-support

- bin-call-manager: Inject provider codecs into channel variables at dial time in createChannelOutgoing"
```

---

## Task 13: RST Documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/` — provider RST files

**Step 1: Find the relevant RST files**
```bash
ls bin-api-manager/docsdev/source/ | grep provider
```

**Step 2: Update `provider_struct_*.rst`**

Add `codecs` to the field table under `WebhookMessage`. Match the existing table format. Example row:
```
codecs         string  Comma-separated codec list (e.g. ``PCMU,PCMA``). Empty = server default. Applied to PSTN dial attempts only.
```

**Step 3: Update `provider_overview.rst`**

Add a note about the `codecs` field — describe its purpose and that it applies to PSTN-routed calls only (not SIP-to-SIP).

**Step 4: Update `provider_tutorial.rst`**

Add an example section showing:
1. Create a provider (codecs omitted — defaults to no restriction)
2. Update codecs via `PUT /providers/{id}` with `{"codecs": "PCMU,PCMA"}`
3. Note that `POST /providers/setup` does not accept codecs; use PUT afterward.

**Step 5: Clean rebuild**
```bash
cd bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```
Expected: clean build with no warnings about undefined references

**Step 6: Force-add build output**
```bash
cd bin-api-manager
git add -f docsdev/build/
git add docsdev/source/
```

**Step 7: Commit**
```bash
git commit -m "NOJIRA-Provider-codec-support

- bin-api-manager: Update RST docs to document provider codecs field"
```

---

## Task 14: Run Full Multi-Service Verification

Run the verification workflow for every modified service:

```bash
# bin-route-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Provider-codec-support/bin-route-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-call-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Provider-codec-support/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-common-handler
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Provider-codec-support/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Provider-codec-support/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Provider-codec-support/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

All must pass before creating the PR.

---

## Task 15: Create Pull Request

**Step 1: Fetch latest main and check for conflicts**
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Provider-codec-support
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

If conflicts: rebase, resolve, re-run Task 14 verification.

**Step 2: Create PR**
```bash
gh pr create \
  --title "NOJIRA-Provider-codec-support" \
  --body "$(cat <<'EOF'
Add selective codec support to the Provider model, enabling operators to constrain SDP negotiation for PSTN calls through a specific carrier.

- bin-dbscheme-manager: Add codecs VARCHAR(255) NOT NULL DEFAULT '' column to route_providers via Alembic migration
- bin-route-manager: Add Codecs field to Provider model, FieldCodecs constant, WebhookMessage, validateCodecs helper, and thread codecs through Create/Update handler and listen handler
- bin-route-manager: Extend V1DataProvidersPost and V1DataProvidersIDPut wire DTOs with Codecs field
- bin-common-handler: Add codecs param to RouteV1ProviderCreate and RouteV1ProviderUpdate RPC wrappers
- bin-openapi-manager: Add optional codecs field to Provider, ProviderCreateRequest, ProviderUpdateRequest schemas
- bin-api-manager: Pass codecs through PostProviders and PutProvidersId handlers; update RST docs
- bin-call-manager: Introduce dialTarget struct, change getDialURI/getDialURITel/getDialURISIP/getDialURISIPDirect to return *dialTarget, inject provider codecs into per-attempt channel variables via setProviderCodecs in createChannelOutgoing
EOF
)"
```

---

## Task 16: api-validator Tests

**Files:**
- Modify: `~/gitvoipbin/monorepo-monitoring/api-validator/`

**Step 1: Find existing provider tests**
```bash
find ~/gitvoipbin/monorepo-monitoring/api-validator/ -name "*.go" | xargs grep -l "provider\|Provider" | head -5
```

**Step 2: Add test for POST /providers with codecs**

Following the existing test patterns, add a test that:
1. Creates a provider with `codecs: "PCMU,PCMA"`
2. Asserts the response includes `"codecs": "PCMU,PCMA"`
3. Cleans up by deleting the created provider

**Step 3: Add test for PUT /providers/{id} to update codecs**

1. Create a provider without codecs
2. PUT with `{"codecs": "PCMU"}` 
3. GET and assert `codecs == "PCMU"`
4. PUT with `{"codecs": ""}` to clear
5. GET and assert `codecs == ""`
6. Delete the provider

**Step 4: Add GET coverage**

Existing GET tests should now naturally return `codecs` — add an assertion that the field is present in the response (even if empty).

**Step 5: Run api-validator suite**
```bash
cd ~/gitvoipbin/monorepo-monitoring/api-validator
go test ./... -v -run TestProvider
```

**Step 6: Commit api-validator changes**
```bash
git -C ~/gitvoipbin/monorepo-monitoring/api-validator add .
git -C ~/gitvoipbin/monorepo-monitoring/api-validator commit -m "NOJIRA-Provider-codec-support

- api-validator: Add provider codecs tests for POST, PUT, and GET endpoints"
```
