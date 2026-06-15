# Design: Make flow-related request fields optional in the OpenAPI spec

## Problem

The OpenAPI spec marks several request-body fields as `required` that the
backend handlers do NOT actually require at runtime. The spec is the source of
truth for the generated Go types (`oapi-codegen`), the ReDoc/Swagger public
docs, and any client SDKs. The mismatch means:

- Public docs tell users a field is mandatory when it is not.
- Generated request structs use value types (non-pointer) for fields the
  handler treats as optional, forcing the handler to rely on zero-value
  coercion (`uuid.FromStringOrNil("")` -> `uuid.Nil`).

Two endpoints are affected:

### 1. `POST /transcribes` — `on_end_flow_id`

- Spec: `openapi/paths/transcribes/main.yaml` lists `on_end_flow_id` in
  `required`.
- Runtime: `server/transcribes.go:38` does
  `onEndFlowID := uuid.FromStringOrNil(req.OnEndFlowId)` with no rejection when
  empty. `pkg/servicehandler/transcribe.go` passes it through; `uuid.Nil` is a
  valid "no follow-up flow" value.
- Conclusion: `on_end_flow_id` is optional at runtime. (Already documented as
  optional in the transcribe tutorial via PR #987.)

### 2. `POST /groupcalls` — `flow_id` and `actions`

- Spec: `openapi/paths/groupcalls/main.yaml` lists
  `source, destinations, flow_id, actions, ring_method, answer_method` all in
  `required`.
- Runtime: `pkg/servicehandler/groupcall.go:152-163` uses `flow_id` if set,
  otherwise builds a temp flow from `actions`. Neither is individually
  mandatory; the caller provides one OR the other (flow_id wins if both set).
- `server/groupcalls.go:50,52` reads `req.FlowId` (value string) and
  `req.Actions` (value slice) with no rejection.
- Conclusion: `flow_id` and `actions` are each optional at runtime (the real
  constraint is "provide at least one", enforced implicitly, not a per-field
  `required`).

## Why not `oneOf`

The spec rule file recommends `oneOf` for polymorphism, but:

- `oneOf` is currently used NOWHERE in the spec (grep: 0 matches). Introducing
  it for groupcall would be the first use and would change the oapi-codegen
  output shape into a union type, forcing a handler rewrite of
  `server/groupcalls.go`.
- `oneOf` semantically means "exactly one", but the runtime does NOT enforce
  exactly-one (both-empty passes through, both-set lets flow_id win). So `oneOf`
  would itself be inaccurate to the code.
- The accurate, minimal, code-truthful change is: remove these fields from
  `required` (make them optional). This matches the handler exactly.

## Change set

### Spec (`bin-openapi-manager`)

1. `openapi/paths/transcribes/main.yaml`:
   - Remove `on_end_flow_id` from the `required` list (keep the property).
   - Enrich the `on_end_flow_id` property. Use `x-go-type: string` alongside
     `format: uuid` so oapi-codegen keeps the field as a Go `string` (NOT
     `openapi_types.UUID`), matching the `gofrs/uuid.FromStringOrNil(string)`
     handler usage. This mirrors the existing `start_member_id` precedent in
     `paths/teams/main.yaml:48-53` (`format: uuid` + `x-go-type: string`
     -> generated `string`):
     ```yaml
     on_end_flow_id:
       type: string
       format: uuid
       x-go-type: string
       description: >-
         The flow to execute when the transcription ends. The flow ID returned
         from the POST /flows or GET /flows response. If omitted, no follow-up
         flow is executed.
     ```

2. `openapi/paths/groupcalls/main.yaml`:
   - Remove `flow_id` and `actions` from the `required` list (keep both
     properties).
   - **Add descriptions that state the at-least-one constraint and the
     precedence**, since "required removal alone" would wrongly imply both are
     freely omittable. The real constraint ("provide flow_id OR actions;
     flow_id wins") is not expressible by `required` and MUST live in the
     description. Use `x-go-type: string` with `format: uuid` on `flow_id` for
     the same reason as above:
     ```yaml
     flow_id:
       type: string
       format: uuid
       x-go-type: string
       description: >-
         The flow to execute for the groupcall. Provide either flow_id or
         actions; if both are set, flow_id takes precedence and actions is
         ignored. The flow ID returned from the POST /flows or GET /flows
         response.
     actions:
       type: array
       minItems: 1
       items:
         $ref: '#/components/schemas/FlowManagerAction'
       description: >-
         Inline actions used to build a temporary flow when flow_id is not
         provided. Provide either flow_id or actions.
     ```
   - Also add a request-body-level description on the POST schema for
     discoverability (a reader scanning only one field still sees the rule):
     `description: "Provide either flow_id or actions. If both are set, flow_id
     takes precedence and actions is ignored."` on the request schema object.
   - Keep `source`, `destinations`, `ring_method`, `answer_method` as required
     (out of scope; see Open question).

3. Regenerate: `cd bin-openapi-manager && go generate ./...` to update
   `gens/models/gen.go`.

### Generated types + handlers (`bin-api-manager`)

4. `cd bin-api-manager && go generate ./...` to regenerate
   `gens/openapi_server/gen.go`. Expected change: `OnEndFlowId`, `FlowId`
   become `*string` (NOT `*openapi_types.UUID`, because of `x-go-type: string`);
   `Actions` becomes `*[]FlowManagerAction`. Confirm the generated types are
   exactly these before editing handlers; if any field comes out as a
   `*openapi_types.UUID`, the `x-go-type: string` is missing and must be added.
5. Adjust handlers to the established nil-check pattern already used in
   `server/activeflows.go:90-99`:
   - `server/transcribes.go:38`:
     ```go
     onEndFlowID := uuid.Nil
     if req.OnEndFlowId != nil {
         onEndFlowID = uuid.FromStringOrNil(*req.OnEndFlowId)
     }
     ```
   - `server/groupcalls.go:50,52`:
     ```go
     flowID := uuid.Nil
     if req.FlowId != nil {
         flowID = uuid.FromStringOrNil(*req.FlowId)
     }
     actions := []fmaction.Action{}
     if req.Actions != nil {
         for _, v := range *req.Actions {
             actions = append(actions, ConvertFlowManagerAction(v))
         }
     }
     ```
6. Handler unit tests for these endpoints build the request body as raw JSON
   bytes (`reqBody: []byte(...)` passed to `c.BindJSON`), so the value-to-pointer
   type flip does NOT break them. No test edits are expected; confirm via
   `go test ./...` rather than pre-emptively editing tests.

## Verification

- `bin-openapi-manager`: `go generate ./...`, `go build ./...`.
- `bin-api-manager`: full 5-step workflow — `go mod tidy && go mod vendor &&
  go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.
- Confirm `gen.go` diff shows only the three fields flipping to pointer types
  (no unrelated churn).
- Confirm no other service consumes these specific generated request structs
  (they are bin-api-manager server types).

## Scope guard / non-goals

- Do NOT touch the RST docs in this PR (already corrected in PRs #987/#990).
- Do NOT introduce `oneOf`.
- Do NOT change `ring_method`/`answer_method` requiredness in this PR unless a
  reviewer flags it (see Open question).
- Separate known mismatch, out of scope: `POST /transcribes` `direction` is
  `required` in the spec but the handler ignores `req.Direction` and hardcodes
  `DirectionBoth` (`server/transcribes.go:51`). Track as a follow-up ticket;
  not addressed here.

## Open question for reviewers

`ring_method`/`answer_method` are `required` in the spec but the handler also
accepts empty values (`RingMethodNone = ""`, `AnswerMethodNone = ""`). Should
they also be made optional for full code-truth, or kept required as a
"normal usage" guard? Default position: keep them required (out of scope), since
this PR targets the flow_id/actions/on_end_flow_id mismatch specifically.
