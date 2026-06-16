# Design: Fix transcribe direction drop + harden POST /calls spec

Date: 2026-06-15
Branch: NOJIRA-transcribe-direction-bugfix-calls-spec
Author: CPO (Lux) for pchero

## Context

Two follow-up items surfaced during the PR #992 (openapi optional flow fields)
review loop, deferred as out-of-scope and now executed together:

1. **transcribe direction drop (runtime bug).** `POST /transcribes` accepts a
   `direction` field (required in the spec, enum `both`/`in`/`out`), but the
   server handler hardcodes `tmtranscribe.DirectionBoth` and ignores
   `req.Direction`. A customer asking for `in` or `out` silently gets `both`.
2. **POST /calls spec gaps (doc parity).** `POST /calls` `flow_id`/`actions`
   lack `format: uuid` + `x-go-type: string`, provenance descriptions, the
   `minItems: 1` array rule, and any statement of the flow_id/actions
   relationship. Same class of defect PR #990/#992 fixed for groupcalls.

These are independent (one is a Go runtime fix, one is a spec doc fix) but both
small, both in the same API surface area, and both originate from the same
review. Bundling them into one PR keeps the follow-up overhead low.

## Goals

- transcribe: honor the caller-supplied `direction` instead of hardcoding `both`.
- transcribe: add a regression test that fails on the old hardcoded behavior
  (i.e. exercises `in`/`out`, not just `both`).
- calls: bring `POST /calls` `flow_id`/`actions` to the same spec quality bar as
  groupcalls (format, x-go-type, provenance, minItems, relationship note).

## Non-goals

- No change to `bin-transcribe-manager` itself. The downstream service already
  accepts a `direction` argument; only the api-manager handler drops it.
- No change to `POST /calls` runtime behavior. The handler already reads
  `req.FlowId`/`req.Actions` correctly (verified: `server/calls.go`); only the
  spec text is deficient. Generated types stay `*string` / `*[]FlowManagerAction`
  (already correct on main).
- groupcall `req.Direction`: not applicable. groupcall has no direction field;
  the PR #992 review note about "req.Direction unused" was about transcribe,
  confirmed here.

## Investigation (code-verified)

### Item 1 — transcribe direction

- Spec: `openapi/paths/transcribes/main.yaml:48-49,65` — `direction` is a
  property (`$ref TranscribeManagerTranscribeDirection`) AND in `required`.
- Generated type: `PostTranscribesJSONBody.Direction
  TranscribeManagerTranscribeDirection` (value type, not pointer, because
  required).
- Handler bug: `server/transcribes.go:54` passes literal
  `tmtranscribe.DirectionBoth` into `TranscribeStart`, never reading
  `req.Direction`.
- Service layer already plumbs direction end-to-end:
  `servicehandler/transcribe.go:110-118` `TranscribeStart(... direction
  tmtranscribe.Direction ...)` → `TranscribeV1TranscribeStart`.
- Direction enum (`models/transcribe/transcribe.go:51-53`): `both` / `in` /
  `out`. OpenAPI enum (`openapi.yaml:6981-6988`) matches `both`/`in`/`out`.
- Test gap: `server/transcribes_test.go:53` sends `"direction":"both"` and
  expects `DirectionBoth`. Because the handler hardcodes `DirectionBoth`, this
  test passes regardless of the bug. The regression test MUST use `in` (or
  `out`) so it would fail against the old code.

### Item 2 — calls spec

- `openapi/paths/calls/main.yaml:39-44` `flow_id` is bare `type: string`;
  `actions` array has no `minItems`. No relationship description.
- Generated types already correct on main (`PostCallsJSONBody.FlowId *string`,
  `Actions *[]FlowManagerAction`) because neither is required. Adding
  `format: uuid` WITHOUT `x-go-type: string` would regenerate FlowId as
  `*openapi_types.UUID` and break the handler — same trap PR #992 hit. So
  `x-go-type: string` is mandatory alongside `format: uuid`.
- Relationship: the two API surfaces deliberately differ. The merged #993
  (bin-ai-manager `create_call` LLM tool) HARD-REJECTS "both provided" (exactly
  one required). The REST `POST /calls` handler does NOT hard-reject: it uses
  flow_id precedence, building a temp flow from actions only when flow_id is nil
  (`servicehandler/call.go` CallCreate). The spec text here describes the REST
  runtime (precedence, actions ignored when both set), matching the groupcalls
  wording established in #992. Do not copy #993's "exactly one required"
  rejection language into the REST spec — it would misrepresent REST behavior.

## Changes

### bin-api-manager (runtime fix)

1. `server/transcribes.go`: stop hardcoding `tmtranscribe.DirectionBoth`; honor
   the caller-supplied direction. IMPORTANT (found in PR review): the previous
   handler ignored direction and always used `both`, so callers that OMIT
   direction relied on the implicit-both behavior. There is no request
   validation middleware and oapi-codegen does not enforce required scalars, so
   honoring direction naively would turn an omitted field into `""` and produce
   a broken empty-direction stream downstream (`bin-transcribe-manager`
   `transcribehandler/start.go:152-155` only expands `both`→`[in,out]`; any other
   value, including `""`, opens a single literal-direction stream). To preserve
   back-compat we make direction OPTIONAL in the spec (see change 6) and default
   to `both` in the handler:
   ```go
   direction := tmtranscribe.DirectionBoth
   if req.Direction != nil {
       direction = tmtranscribe.Direction(*req.Direction)
   }
   ```
2. `server/transcribes_test.go`: KEEP the existing `both` success case and ADD
   table rows for `"direction":"in"` (assert `DirectionIn`), `"direction":"out"`
   (assert `DirectionOut`), and `direction` OMITTED (assert `DirectionBoth`, the
   back-compat default). The mock matches the direction arg exactly, so an
   `in`/`out` row fails against the old hardcoded `DirectionBoth` and the omitted
   row guards the default-path regression. The mocked response does not echo
   direction, so `expectRes` is identical across rows.

### bin-openapi-manager (spec doc fix + transcribe optional)

6. `openapi/paths/transcribes/main.yaml`: remove `direction` from the `required`
   array (it now defaults to `both` when omitted) and document the default via an
   `allOf`-wrapped `$ref` + description ("Which audio legs to transcribe. If
   omitted, defaults to \"both\"."). This regenerates `Direction` as
   `*TranscribeManagerTranscribeDirection` (pointer + omitempty), matching the
   nil-check handler above. This mirrors PR #992's "spec matches runtime" intent.

3. `openapi/paths/calls/main.yaml` `flow_id`:
   ```yaml
   flow_id:
     type: string
     format: uuid
     x-go-type: string
     description: >-
       The flow to execute for this call. The flow ID returned from the
       POST /flows or GET /flows response. Provide either flow_id or actions.
       If both are supplied, flow_id takes precedence and actions is ignored.
     example: "f1a2b3c4-d5e6-7890-1234-567890abcdef"
   ```
4. `openapi/paths/calls/main.yaml` `actions`:
   ```yaml
   actions:
     type: array
     minItems: 1
     items:
       $ref: '#/components/schemas/FlowManagerAction'
     description: >-
       Inline flow actions used to build an ephemeral flow for this call.
       Provide either flow_id or actions. Ignored if flow_id is set.
   ```
5. Regenerate `bin-openapi-manager/gens/models/gen.go` and
   `bin-api-manager/gens/openapi_server/gen.go` via `go generate ./...`. Expect
   FlowId to STAY `*string` (guard: must not become `*openapi_types.UUID`).
   Actions stays `*[]FlowManagerAction`. No other type drift.

## Verification

For each changed service run the full 5-step workflow:
`go mod tidy && go mod vendor && go generate ./... && go test ./... &&
golangci-lint run -v --timeout 5m`.

- bin-openapi-manager: build + generate reproducibility, gen.go diff = calls
  field comments only, FlowId stays `*string`, go.mod/go.sum unchanged after
  tidy.
- bin-api-manager: build + full test + lint. The transcribe regression test
  must fail on old handler code and pass on new (verify by reasoning / local
  revert check). server gen.go regenerated for calls description.

## Risks

- `tmtranscribe.Direction(req.Direction)` is a raw string cast. If a caller
  sends an invalid enum value, the generated type's unmarshalling already
  rejects unknown enum values at BindJSON for a typed enum? No — oapi-codegen
  enum types are plain string aliases without unmarshal validation. Same as
  existing `provider`/`reference_type` passthrough. Downstream
  bin-transcribe-manager validates. Acceptable and consistent with current
  patterns. Not introducing new validation in this PR (out of scope).
- calls example UUID must be distinct from groupcall/transcribe examples used in
  #992 to avoid the "all same UUID" nit raised in PR #992 review.
