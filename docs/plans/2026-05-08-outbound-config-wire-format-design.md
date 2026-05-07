# Outbound Config RPC Wire-Format Fix — Design

**Date:** 2026-05-08
**Status:** Approved
**Branch:** `NOJIRA-Fix-outbound-config-wire-format`

## Problem

`PUT /v1.0/outbound_config` returns HTTP 200 but the row is never updated. Reproduced in production (api-manager pod `f7t5k` at 16:13:34 UTC: `PUT 200`, `GET 200` returns unchanged data).

### Root cause — JSON wire-format mismatch on the internal RPC

| Side | Code | Wire shape produced/expected |
|---|---|---|
| Client | `bin-common-handler/pkg/requesthandler/call_outbound_configs.go:99` — `json.Marshal(req)` | `{"name":"...","detail":"...","destination_whitelist":[...],"codecs":"..."}` |
| Listener | `bin-call-manager/pkg/listenhandler/v1_outbound_configs.go:179-180` — `json.Unmarshal(m.Data, &V1DataOutboundConfigsIDPut)` | `{"request":{...}}` |

`json.Unmarshal` silently ignores unknown top-level keys, so `req.Request` is the zero value. Every pointer field on `*UpdateRequest` is nil. The DB layer (`bin-call-manager/pkg/dbhandler/outbound_config.go::OutboundConfigUpdate`) treats nil pointers as "no change," updates only `tm_update`, and returns the unchanged row → 200 OK with stale data.

### Why this slipped through

- `Test_processV1OutboundConfigsIDPut` (`v1_outbound_configs_test.go:281`) hard-codes the wrapped shape `{"request":{"name":"updated"}}`, then asserts via `Update(gomock.Any(), tt.expectID, gomock.Any())` — the third arg matcher is `gomock.Any()`, so the parsed `*UpdateRequest` is never validated. The test would pass even with all fields zeroed.
- No client-side test exists for `CallV1OutboundConfigUpdate` in `bin-common-handler` (`ls bin-common-handler/pkg/requesthandler/call_outbound_configs*` shows only the source file).
- The handler logs only on the error path; a silent no-op produces no logs.

### Why the wire format diverged

`outbound_configs.go` is the only request file in `bin-call-manager/pkg/listenhandler/models/request/` that uses a `Request` wrapper field. Every other resource (`calls.go`, `recordings.go`, `confbridge.go`, `channels.go`, `externalmedias.go`, `groupcalls.go`, `recovery.go`) is flat. The wrapper was introduced to import `outboundconfig.UpdateRequest` directly into the listener request package, which conflated the **wire format** (RPC contract) with the **domain model** (handler/DB partial-update shape).

## Goals

1. Fix `PUT /v1.0/outbound_config` so it actually persists changes.
2. Align outbound_config's RPC wire format with the rest of the service (flat structs in `pkg/listenhandler/models/request/`).
3. Tighten tests so this class of bug fails CI next time.

## Non-goals

- Changing the public REST API (`PUT /v1.0/outbound_config` body shape stays the same).
- Schema or migration changes.
- RST documentation changes (user-facing contract is unchanged).
- Refactoring other resources' wire formats.

## Design

### Wire format (call-manager listener)

`bin-call-manager/pkg/listenhandler/models/request/outbound_configs.go` becomes:

```go
package request

import "github.com/gofrs/uuid"

// V1DataOutboundConfigsPost is the request body for POST /v1/outbound_configs.
type V1DataOutboundConfigsPost struct {
    CustomerID           uuid.UUID `json:"customer_id"`
    Name                 *string   `json:"name,omitempty"`
    Detail               *string   `json:"detail,omitempty"`
    DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
    Codecs               *string   `json:"codecs,omitempty"`
}

// V1DataOutboundConfigsIDPut is the request body for PUT /v1/outbound_configs/<id>.
type V1DataOutboundConfigsIDPut struct {
    Name                 *string   `json:"name,omitempty"`
    Detail               *string   `json:"detail,omitempty"`
    DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
    Codecs               *string   `json:"codecs,omitempty"`
}
```

- No `Request` wrapper — flat, matches `V1DataCallsPost`, `V1DataRecordingsPost`, etc.
- Drops the import of `outboundconfig.UpdateRequest` from this package (separates wire shape from domain model).
- Pointer fields preserve the "absent" vs. "explicit empty" distinction.

### Listener handlers

`bin-call-manager/pkg/listenhandler/v1_outbound_configs.go`:

```go
// PUT
var req request.V1DataOutboundConfigsIDPut
if err := json.Unmarshal(m.Data, &req); err != nil { ... }

updateReq := &outboundconfig.UpdateRequest{
    Name:                 req.Name,
    Detail:               req.Detail,
    DestinationWhitelist: req.DestinationWhitelist,
    Codecs:               req.Codecs,
}
c, err := h.outboundConfigHandler.Update(ctx, id, updateReq)
```

POST gets the same translation. The `outboundconfig.UpdateRequest` model in `models/outboundconfig/outboundconfig.go` is **kept where it is** — it's a legitimate handler/DB-layer model used by `OutboundConfigUpdate` to build dynamic `UPDATE … SET` SQL. Only the listener boundary changes.

### Client (bin-common-handler)

`bin-common-handler/pkg/requesthandler/call_outbound_configs.go`:

```go
// CallV1OutboundConfigCreate
m, err := json.Marshal(cmrequest.V1DataOutboundConfigsPost{
    CustomerID:           customerID,
    Name:                 req.Name,
    Detail:               req.Detail,
    DestinationWhitelist: req.DestinationWhitelist,
    Codecs:               req.Codecs,
})

// CallV1OutboundConfigUpdate
m, err := json.Marshal(cmrequest.V1DataOutboundConfigsIDPut{
    Name:                 req.Name,
    Detail:               req.Detail,
    DestinationWhitelist: req.DestinationWhitelist,
    Codecs:               req.Codecs,
})
```

- Drops the inline anonymous struct in `CallV1OutboundConfigCreate`.
- Drops the bare `json.Marshal(req)` in `CallV1OutboundConfigUpdate` (the bug).
- Public function signatures are unchanged — `bin-api-manager` callers are not touched.

### Test hardening

1. **`bin-call-manager/pkg/listenhandler/v1_outbound_configs_test.go`:**
   - Update `Data` payloads to flat shape: `{"name":"updated"}` (PUT), `{"customer_id":"...","name":"new"}` (POST).
   - Replace `Update(gomock.Any(), tt.expectID, gomock.Any())` with a struct-literal matcher (gomock falls back to `reflect.DeepEqual` for non-`Matcher` args), so the parsed `*UpdateRequest` is verified field-by-field.
   - Same for the POST test (`Test_processV1OutboundConfigsPost`).

2. **`bin-common-handler/pkg/requesthandler/call_outbound_configs_test.go` (new):**
   - For `CallV1OutboundConfigUpdate`: use mock `sockHandler.EXPECT().RequestPublish(...)` capturing the published message; assert `m.Data` unmarshals into `V1DataOutboundConfigsIDPut` with the expected pointer values.
   - Same shape for `CallV1OutboundConfigCreate`.

## Trade-offs and risks

### Mixed-version deploy window

Both POST and PUT change wire format. During a rolling deploy:
- Old api-manager pod (sends wrapped) → new call-manager pod (expects flat): unknown keys silently dropped → POST creates a config with all fields nil; PUT no-ops.
- New api-manager pod (sends flat) → old call-manager pod (expects wrapped): same silent failure.

**Impact:** outbound_config is admin-frequency, not high-volume; the window is the duration of a rolling deploy (typically 1–3 minutes per service). Customer-visible failure mode is "POST/PUT returned 200 but data wasn't saved" — same as the existing PUT bug, just briefly extended to POST.

**Mitigation:**
1. Deploy `bin-call-manager` first (the listener accepting the new shape). At this point old api-manager is still sending wrapped → broken, but PUT was already broken so this is no regression for PUT; POST briefly degrades.
2. Deploy `bin-api-manager` second, immediately after.
3. Pause heavy outbound_config CRUD during the deploy window if practical (admin operation, easy to schedule).

Alternative considered and rejected: have the listener accept both shapes during transition, then drop wrapped support in a follow-up. Adds complexity and a temporary code path that must be cleaned up; not worth it given the small impact window.

### Cross-service verification

`bin-common-handler` is changed → must run the verification workflow in `bin-common-handler`, `bin-api-manager`, and `bin-call-manager`. Public signatures (`CallV1OutboundConfigCreate`, `CallV1OutboundConfigUpdate`) are unchanged, so callers don't recompile differently — but `go mod vendor` must be refreshed in api-manager and call-manager, and tests must pass in all three.

### Scope of refactor

We are touching the working POST path to keep both methods consistent. If consistency is not worth the deploy-window risk, the alternative (PR 1 from earlier discussion) is a one-line wrap in `CallV1OutboundConfigUpdate` only. **User approved the full refactor** (option C) on 2026-05-08, accepting the trade-off.

## Acceptance criteria

- `PUT /v1.0/outbound_config` with `{"name": "x"}` results in `name = 'x'` in the DB and the GET response reflects it.
- `POST /v1/outbound_configs` (admin) continues to work after redeploy.
- `golangci-lint run -v --timeout 5m` passes in `bin-common-handler`, `bin-call-manager`, `bin-api-manager`.
- `go test ./...` passes in all three services.
- `Test_processV1OutboundConfigsIDPut` and `Test_processV1OutboundConfigsPost` both fail (with a clear assertion message) if the listener's `*UpdateRequest` would be all-nil — i.e., the test would have caught the original bug.
- New test `Test_CallV1OutboundConfigUpdate` in `bin-common-handler` asserts the marshaled JSON contains the expected top-level fields and does NOT contain a `request` wrapper key.

## Files changed

```
bin-call-manager/pkg/listenhandler/models/request/outbound_configs.go    # flatten structs
bin-call-manager/pkg/listenhandler/v1_outbound_configs.go                # translate flat → UpdateRequest
bin-call-manager/pkg/listenhandler/v1_outbound_configs_test.go           # flat data + tightened matchers
bin-common-handler/pkg/requesthandler/call_outbound_configs.go           # marshal flat structs
bin-common-handler/pkg/requesthandler/call_outbound_configs_test.go      # NEW: assert wire shape
docs/plans/2026-05-08-outbound-config-wire-format-design.md              # this doc
```

No files in `bin-api-manager/`, `models/outboundconfig/`, RST docs, OpenAPI spec, migrations, or other services.

## Verification workflow

For each of `bin-common-handler`, `bin-call-manager`, `bin-api-manager` (in that order):

```
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
