# create_call inline action assembly design

Date: 2026-06-15
Service: bin-ai-manager
Type: LLM tool extension (feature)
Branch: NOJIRA-Add-create-call-actions-design

## 1. Problem Statement

The `create_call` LLM tool places a NEW, independent outbound call that runs a
pre-existing flow. Today it **requires** a `flow_id` referencing a flow the
customer built in advance. This means the AI can only originate calls whose
behavior was anticipated and authored ahead of time.

There is no way for the AI to assemble an ad-hoc call scenario at runtime. A
request like "call John and tell him the meeting moved to 3pm" is impossible
unless a matching flow already exists, because the AI cannot express "talk this
text, then hang up" as part of the call origination.

We want the AI to be able to **assemble flow actions inline** and originate a
call that executes those actions, without a pre-built flow.

## 2. Scope

### In scope (Phase 1)

- Add an optional `actions` array parameter to the `create_call` tool.
- `flow_id` and `actions` become a oneof: exactly one must be provided.
- When `actions` is provided, ai-manager creates an ephemeral (non-persisted)
  flow from those actions and originates the call against it, reusing the
  existing `FlowV1FlowCreate(..., persist=false)` + `CallV1CallsCreate` path.
- No action-type whitelist and no denylist: the full action-type set is allowed
  without exception. Type-level hygiene is enforced by flow-manager
  `ValidateActions`; verification gating for send actions is each resource
  manager's responsibility (#989), not this tool's.

### Out of scope

- Action-type whitelisting / denylisting / per-type cost guards. Decision:
  expose all action types without exception. Type-level hygiene is enforced by
  flow-manager `ValidateActions` (rejects unknown types, validates `anonymous`
  on call/connect). Deeper option validation is the responsibility of each
  action executor, not this tool.
- verified-customer gating for SMS/Email/SNS send actions. Decision: gating is
  enforced (or fixed) at each resource manager (message-manager, email-manager,
  conversation-manager), NOT in this tool. The current asymmetry (call-manager
  gates PSTN; the others do not) is tracked in #989 and is out of scope here.
- **Composition-level guards** (per-flow action-count cap, nested call/connect
  recursion-depth limit, goto/branch loop-iteration budget, ephemeral-flow TTL/
  size cap). Decision: these are enforced at the activeflow/flow layer in
  flow-manager, NOT in this tool, because they apply uniformly to every flow
  creation path (REST `CallCreate` inline actions, campaign flows, this tool)
  and no single action executor can observe the whole assembled graph. Tracked
  in issue #991.
- A separate `create_call_with_actions` tool. We extend the existing tool.
- Batch / multi-step "assemble many calls" semantics. One call origination per
  invocation (already the contract).

## 3. Design

### 3.1 Mechanism (no new RPC, no new engine)

The full "assemble actions then originate a call" path already exists in the
codebase. The user-facing REST `CallCreate` (bin-api-manager
`servicehandler/call.go`) already accepts both `flow_id` and `actions` and does:

```go
targetFlowID := flowID
if targetFlowID == uuid.Nil {
    f, _ := h.FlowCreate(ctx, a, "tmp", "tmp outbound flow", actions, uuid.Nil, false) // persist=false
    targetFlowID = f.ID
}
// verify flow ownership
// CallV1CallsCreate(ctx, customerID, targetFlowID, uuid.Nil, source, destinations, false, false, anonymous, nil, variables)
```

We mirror this pattern inside `toolHandleCreateCall`, calling
`reqHandler.FlowV1FlowCreate(...)` directly (ai-manager has no `FlowCreate`
helper).

```
LLM assembles actions[]
  -> ai-manager toolHandleCreateCall
       if actions present: FlowV1FlowCreate(customerID, TypeFlow, "tmp", "...", actions, uuid.Nil, false)
                           -> ephemeral flow (cache-only, see 3.4)
       targetFlowID = ephemeral flow id (or provided flow_id)
  -> CallV1CallsCreate(customerID, targetFlowID, uuid.Nil, source, destinations, false, false, anonymous, nil, variables)
```

### 3.2 oneof handling (flow_id XOR actions)

| flow_id | actions | result |
|---|---|---|
| set | empty | use flow_id (current behavior, unchanged) |
| nil | non-empty | create ephemeral flow from actions, use it |
| set | non-empty | reject: "provide either flow_id or actions, not both" |
| nil | empty | reject: "either flow_id or actions is required" |

Validation happens in `toolHandleCreateCall` before any RPC. Both rejection
paths return a `fillFailed` tool result (the LLM sees a clear error string and
can retry), not a Go-level error that aborts the turn.

### 3.3 Action parsing

The `actions` argument arrives as JSON. It is unmarshaled into
`[]fmaction.Action` (the flow-manager action model, already imported in
`tool.go` as `fmaction`). The `Action` struct is:

```go
type Action struct {
    ID     uuid.UUID      `json:"id,omitempty"`
    NextID uuid.UUID      `json:"next_id,omitempty"`
    Type   Type           `json:"type,omitempty"`
    Option map[string]any `json:"option,omitempty"`
}
```

The LLM supplies `type` and `option`, and may also supply `id`/`next_id` when
it wants to express non-linear control flow. These are **preserved as-is** and
passed through to `FlowV1FlowCreate`; flow-manager `GenerateFlowActions` assigns
an `id` only when one is nil. We do NOT reset ids.

Rationale: `goto`/`branch`/`condition_*` actions carry their jump targets inside
`option` (e.g. `goto.target_id`, `branch.target_ids`, `condition_*.false_target_id`)
referencing other actions by their `id`. If the handler reset `id` to nil,
flow-manager would re-assign fresh UUIDs and those option-level target
references would dangle, silently breaking every branch/goto the LLM built.
Preserving ids is therefore required for the assembled flow to be functional at
all. This is identical to how REST `CallCreate` inline actions already behave.

Consequence: the LLM CAN author non-linear flows, including back-edge loops via
`goto`/`branch`. This is intentional and consistent with the owner decision that
loop / recursion / count budgets are enforced uniformly at the flow-manager
activeflow layer (#991), not in this tool.

Reuse the existing `json.Decoder` + `UseNumber()` pattern already used for the
`variables` field, so numeric option values (e.g. `sleep` duration) keep
integer precision instead of becoming float64.

An `actions` key present but empty (`[]`) is treated as "actions not provided":
the oneof check uses `len(actions) > 0`, so an empty array falls into the
"neither provided" rejection path, never into ephemeral-flow creation.

### 3.4 Ephemeral flow lifecycle (no cleanup needed)

`FlowV1FlowCreate(..., persist=false)` does NOT write to the database. In
flow-manager `flowhandler/db.go`, the `persist=false` branch calls
`FlowSetToCache` only. The flow lives in Redis and expires by TTL. There is no
orphaned DB row, no explicit delete required. This matches every existing
ephemeral-flow caller (call-manager SIP/agent/queue/conference direct dialing,
flow-manager connect action).

### 3.5 Validation / hygiene (where each check lives)

| Concern | Enforced by | Notes |
|---|---|---|
| Unknown action type | flow-manager `ValidateActions` (via `FlowV1FlowCreate` -> `GenerateFlowActions`) | rejects type not in `TypeListAll` |
| anonymous value on call/connect | flow-manager `ValidateActions` | rejects invalid anonymous |
| action option field correctness | each action executor at run time | not validated at assembly time today |
| Composition limits (count/recursion/loop) | flow-manager activeflow layer (#991) | not in this tool, deferred |
| PSTN origination permission | call-manager `ValidateCustomerIdentityVerified` (fail-closed) | unchanged |
| SMS/Email/SNS send permission | each resource manager (#989) | call-manager gates PSTN; message/email/conversation managers do not yet. Not this tool's concern |
| flow_id ownership (when flow_id used) | existing check in `toolHandleCreateCall` (`f.CustomerID == c.CustomerID`, masked error) | unchanged |

Note on ephemeral-flow ownership: when `actions` is used, the ephemeral flow is
created with `customerID = c.CustomerID`, so it is owned by the aicall's
customer by construction. The inline `actions` path has no caller-supplied
top-level flow id, so it adds no flow-existence-oracle IDOR surface. This claim
is scoped to the TOP-LEVEL flow only: action `option` fields may themselves
reference other resources (e.g. a nested flow id, conference id, queue id), and
ownership of those referenced resources is the responsibility of each action
executor at run time, not validated here.

## 4. Tool definition changes (definitions.go)

Add `actions` to the `create_call` parameter schema and relax the `required`
constraint. Because JSON Schema `required` cannot express XOR directly, keep
`required: ["destinations"]` and document the flow_id-XOR-actions rule in the
descriptions; enforce the XOR in code (3.2).

`actions` schema (array of objects):

```
"actions": {
  "type": "array",
  "description": "Assemble the call scenario inline as an ordered list of flow
    actions, INSTEAD OF flow_id. Provide either flow_id (reuse a pre-built flow)
    or actions (build the scenario now), not both. Each item is a flow action
    with a 'type' and a 'type-specific 'option' object. Example to speak a
    message then hang up:
    [{\"type\":\"talk\",\"option\":{\"text\":\"Hi, the meeting moved to 3pm\",
    \"language\":\"en-US\"}},{\"type\":\"hangup\"}].",
  "items": {
    "type": "object",
    "properties": {
      "type":   {"type": "string",
                 "description": "Flow action type, e.g. talk, play, hangup,
                   connect, variable_set, branch, goto, sleep, digits_receive."},
      "option": {"type": "object",
                 "description": "Action-type-specific options. Shape depends on
                   type. e.g. talk -> {text, language, gender}. Omit for actions
                   with no options (e.g. hangup)."}
    },
    "required": ["type"]
  }
}
```

Update `flow_id` description to: "UUID of a pre-existing flow the new call will
execute. Provide either flow_id OR actions, not both. Must belong to your
account." Update the tool's top-level description to mention inline assembly.

LLM-facing copy uses the term "flow" / "call scenario", not telephony-narrowed
wording, but `create_call` is by definition a call-origination tool so "call"
is accurate here.

## 5. Handler changes (aicallhandler/tool.go)

In `toolHandleCreateCall`, after parsing args:

1. Parse a new `Actions []fmaction.Action` field (`json:"actions,omitempty"`)
   from the tool arguments alongside the existing fields.
2. XOR validation (3.2), using `len(args.Actions) > 0` for "actions provided"
   (so an empty `[]` counts as not provided): reject if both or neither of
   `flow_id`/`actions` set.
3. If `actions` provided:
   - **Preserve ids (3.3):** pass `args.Actions` through unchanged. The LLM's
     `id`/`next_id` and option-level target references are kept so branch/goto
     control flow stays intact.
   - `f, err := h.reqHandler.FlowV1FlowCreate(ctx, c.CustomerID, fmflow.TypeFlow,
     "tmp", "tmp flow for ai create_call action assembly", args.Actions,
     uuid.Nil, false)` — on error `fillFailed(res, err)` and return. The error
     from `ValidateActions` (unknown type / bad anonymous) surfaces here as a
     clear message.
   - `targetFlowID = f.ID`
4. If `flow_id` provided: keep the existing ownership check
   (`FlowV1FlowGet` + masked `errCouldNotResolveFlow`), `targetFlowID = flow_id`.
5. `CallV1CallsCreate(ctx, c.CustomerID, targetFlowID, uuid.Nil, &src,
   args.Destinations, false, false, args.Anonymous, nil, variables)` — unchanged
   from here down (partial-failure counting, primary id/type, result body).

New imports: `fmflow "monorepo/bin-flow-manager/models/flow"` for `TypeFlow`.

### Failure handling matrix

| Failure | Behavior |
|---|---|
| both flow_id and actions set | `fillFailed` "provide either flow_id or actions, not both" |
| neither set (incl. empty `actions: []`) | `fillFailed` "either flow_id or actions is required" |
| actions JSON malformed | existing decode error -> `fillFailed` |
| FlowV1FlowCreate rejects (bad action type/anonymous) | `fillFailed` with flow-manager error message |
| flow_id not owned / not found | existing masked `errCouldNotResolveFlow` |
| all destinations fail | existing `fillFailed(res, err)` |
| partial destination failure | existing requested-vs-created `partial:true` |

## 6. Security & Compliance

### 6.1 Blast-radius change (honest framing)

This change qualitatively shifts `create_call` from "execute a flow a HUMAN
authored and reviewed in advance" to "execute a call program the LLM ASSEMBLES
at runtime." Because the LLM is driven by an end caller who may attempt prompt
injection, the effective actor for the assembled program is partly the caller.
The mitigations below are scoped accordingly; this section does not claim the
surface is unchanged.

### 6.2 What is closed here

- Ephemeral flow is created under `c.CustomerID`; no cross-customer top-level
  flow surface. The inline path has no caller-supplied top-level flow id, so no
  flow-existence oracle is added.
- flow_id path retains the masked not-found/cross-customer error.
- PSTN origination stays gated by call-manager (fail-closed). An assembled flow
  that dials PSTN still passes through `CallV1CallsCreate` -> call-manager
  identity verification.

### 6.3 Residual risks (acknowledged, deferred to named owners)

These are NOT closed by this PR and are explicitly recorded:

- **Composition-level abuse** (nested call/connect fan-out, unbounded action
  count, goto/branch back-edge loops, branch/sleep occupancy): deferred to
  flow-manager activeflow guards, #991. The LLM CAN author non-linear flows
  (ids are preserved so branch/goto work); the loop/recursion/count budget that
  bounds this is flow-manager's responsibility and applies uniformly to REST
  `CallCreate` inline actions too.
- **SMS/Email/SNS send without verification**: all action types including
  `message_send`/`email_send`/`conversation_send` are exposed (owner decision).
  Verification gating for those is each resource manager's responsibility,
  tracked in #989. This tool does not gate them.
- **webhook_send exfiltration / SSRF**: `webhook_send` remains allowed; option
  contents (URL) are not validated here. Egress allowlist / private-IP blocking
  is tracked in #979 (webhook SSRF hardening). Acknowledged residual.
- **Toll fraud via premium/international destinations**: identity verification
  gates the SOURCE identity, not the DESTINATION. Premium-rate dialing is a
  pre-existing concern not specific to this tool.
- **Nested-resource IDOR in action options** (a nested flow id / conference id
  referenced inside an `option`): ownership is each executor's responsibility at
  run time, not validated at assembly. Pre-existing for all inline-action paths.

## 7. Observability

- Existing debug logging in `toolHandleCreateCall` is retained. When the inline
  `actions` path is taken, log the ephemeral flow id AND the assembled action
  types + count, so abuse triage / forensics can reconstruct what the LLM built.
- No new Prometheus metrics for this change (it is a tool-surface extension on
  an existing tool, not a new async subsystem). Tool-call volume is already
  observable via existing aicall tool handling. (A dedicated assembled-call
  counter can be added later if abuse monitoring needs it.)

## 8. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-ai-manager | `definitions.go`: add `actions` to create_call schema; `tool.go`: XOR + ephemeral-flow branch in `toolHandleCreateCall` | 1 |

No backend RPC changes. No DB migration. No OpenAPI change (this is an internal
LLM tool, not a REST endpoint). Per the backend-design convention,
internal-RPC/tool-only features do not require RST `docsdev` updates.

## 9. Implementation Order

1. `models/tool` — no change (create_call already registered).
2. `definitions.go` — add `actions` param + update descriptions.
3. `tool.go` `toolHandleCreateCall` — add `Actions` field, XOR validation
   (`len > 0`), ephemeral-flow branch (ids preserved), new `fmflow` import,
   enriched success log.
4. Unit test: extend `toolHandleCreateCall` test (or sibling) with table cases:
   actions-only success, flow_id-only success (regression), both-set rejection,
   neither-set/empty-array rejection, bad-action-type rejection via
   FlowV1FlowCreate mock returning error, and a case asserting the assembled
   actions (incl. any LLM-supplied id/next_id) are passed through unchanged to
   FlowV1FlowCreate. Use gomock; assert RPC arity.
5. Full verification workflow in bin-ai-manager.

## 10. Open Questions

| Question | Recommendation | Owner |
|---|---|---|
| Composition guards (action count / recursion / loop) | Deferred to flow-manager activeflow layer, #991 (decided) | flow-manager |
| SMS/Email/SNS verification gap | Fixed at resource managers, #989 (decided); not gated in this tool | CEO/CTO |
| webhook_send SSRF / egress | Tracked in #979; not addressed here | CEO/CTO |
| Should `talk` language default be injected if LLM omits it? | No; leave to the talk action executor's existing defaults | CPO |

## 11. Review Summary

### v1 -> v2 (first two design reviews)

Two independent design reviews (general soundness + adversarial security) both
returned CHANGES REQUESTED. v2 applied: schema fix (`item.required = ["type"]`
only so option-less actions like `hangup` are valid), empty-array handling
(`len > 0`), corrected security framing (removed inaccurate "not widened" claim;
added blast-radius framing 6.1 + residual-risk register 6.3), enriched
observability log, IDOR claim scoped to the top-level flow. v2 also added a
send-action denylist and an id/next_id graph-reset, which were superseded in v3.

### v2 -> v3 (owner decision + re-review fix)

- **Send-action denylist removed (owner decision):** message_send / email_send /
  conversation_send are no longer blocked. All action types are exposed without
  exception. Verification gating for send actions is each resource manager's
  responsibility (#989), not this tool's.
- **Graph-reset removed (re-review finding, was a correctness bug):** the v2
  plan to reset `id`/`next_id` to nil would have re-assigned fresh UUIDs in
  flow-manager while leaving option-level jump targets (`goto.target_id`,
  `branch.target_ids`, `condition_*.false_target_id`) pointing at the old ids,
  silently breaking every branch/goto the LLM built. v2's claim that reset made
  the flow "linear / no jumps" was false because those actions carry targets in
  `option`, not in `NextID`. v3 preserves ids as-is (matching REST `CallCreate`
  behavior); branch/goto work, and loop/recursion budgets are flow-manager's
  concern (#991).

Deferred (not blocking, by owner decision): composition guards (#991), webhook
egress hardening (#979), resource-manager send gating (#989).
