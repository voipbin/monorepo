# Design: `create_call` LLM Tool (independent outbound call origination)

Status: Draft (v3 — post review round 2)
Author: CPO (with pchero)
Date: 2026-06-09
Service: bin-ai-manager (+ reuses bin-call-manager RPC)

## 1. Problem Statement

The AI (voicebot / chatbot / task agent) has no tool to **originate a brand-new, independent outbound call**.

The existing `connect_call` tool looks like it could, but its actual behavior is fundamentally different:

- `connect_call` (`toolHandleConnect` → flow-manager `actionHandleConnect`) creates a confbridge, builds a temporary flow (confbridge_join + hangup), originates the new leg via `CallV1CallsCreate` with `masterCallID = 현재콜`, **bridges the new call into the current call**, and then immediately terminates the AIcall (`AIV1AIcallTerminate`).
- It is gated to `ReferenceType == call` only (`c.ReferenceType != aicall.ReferenceTypeCall` → fail). So it cannot be used from a chat / conversation / task session.

What is missing: a tool that lets the AI place a **standalone** outbound call that is NOT bridged to the current session, runs its own flow, and does NOT end the current AI session. This is required for the AI to act as a true assistant.

Use cases:
- During a text/chat session: "지금 김대리에게 전화 걸어줘" → place an outbound call (current session is chat, not a call).
- During a live call: trigger a callback/notification call to a third party while keeping the current customer call alive.
- An AI worker in a task context triggers an outbound call queue.

## 2. Scope

### In scope (Phase 1)
- New LLM tool `create_call`.
- Originates an independent outbound call (or group call) via the existing `CallV1CallsCreate` RPC with `masterCallID = uuid.Nil` (no bridge).
- The originated call executes a **pre-existing flow referenced by `flow_id`**.
- Works from ANY reference type (call / conversation / task / chat). No reference_type restriction.
- Returns the originated `call_id`(s) and `groupcall_id`(s) in the tool result for tracking.
- `run_llm` default `true` (AI confirms verbally / continues conversation).
- Ownership validation: the referenced `flow_id` MUST belong to the same customer (IDOR prevention).

### Out of scope (Phase 2+)
- Inline `actions[]` array passed by the LLM (LLM-authored flow). Phase 2.
- Linking the originated call to a new AIcall (AI-to-AI calling). Phase 2+.
- Auto-trigger / scheduled / queued origination. Phase 2+.
- Source-number selection logic in ai-manager (delegated to call-manager; see §4).

### Rationale for Phase 1 = `flow_id` reference only
Having the LLM author an `actions[]` array in-band during a voice call carries validation, cost, and latency risk. `flow_id` reference cleanly covers the dominant "AI triggers a predefined scenario call" use case, matches the VoIPBin flow model, and is the smallest safe v1.

## 3. Why no new entity / DB / REST / LLM-logic

This is a **tool-exposure feature**, not a new-entity feature. Therefore the following standard design sections are **N/A**:

- Domain Model (new struct): N/A — reuses `message.ToolCall` / existing tool framework.
- Database Schema: N/A — no new table.
- REST API: N/A — internal tool only; not externally exposed. No `docsdev` RST update needed.
- LLM Logic (own LLM call): N/A — the tool does not itself call an LLM; it is invoked BY the LLM via function calling.
- Webhook Events: N/A — call-manager already emits call lifecycle events for the originated call.
- Flow Variable Integration: N/A.

The remaining relevant sections: tool definition, handler flow, security/ownership, observability, affected services, implementation order, open questions.

## 4. Source number & permission gating: delegated to call-manager (verified)

Confirmed by reading `bin-call-manager/pkg/callhandler/outgoing_call.go::CreateCallOutgoing`:

- **Source validation / default-fill** (`getValidatedSourceForOutgoingCall`, L245): if the supplied source is empty or unusable, call-manager resolves a default / owned source. Only when NO valid source exists does it reject. → ai-manager does NOT need to validate or fill source.
- **Outgoing-call permission** (`validateOutgoingCallPermission`, L169): customer status + identity verification (this is the "verified customer only" gate). → ai-manager does NOT need to re-check.
- **Balance** (`ValidateCustomerBalance`, L175) and **PSTN whitelist + source validation** (L207-218): all fail-closed in call-manager.

Design consequence: **ai-manager performs NO source/verified/balance/whitelist checks.** The customer enabling this tool (`tool_names`) is sufficient authorization at the ai-manager layer; call-manager enforces all origination safety fail-closed. ai-manager's only security duty is `flow_id` ownership (§7).

`CreateCallOutgoing` generates a fresh `call_id` when `id == uuid.Nil` (L138) and returns it on the `*call.Call`, so the originated IDs are available for the tool result (requirement: tracking).

## 5. Tool definition

### 5.1 Names (touch points 1-2)

`models/tool/main.go`:
```go
ToolNameCreateCall ToolName = "create_call"
// add to AllToolNames slice (REQUIRED — TestAllToolNames is hardcoded)
```

`models/message/tool.go`:
```go
FunctionCallNameCreateCall FunctionCallName = "create_call"
```

### 5.2 Definition (`pkg/toolhandler/definitions.go`, touch point 3)

```go
{
    Name:   tool.ToolNameCreateCall,
    RunLLM: true,
    Description: `Places a NEW, INDEPENDENT outbound call that is NOT connected/bridged
to the current conversation. The new call runs its own predefined flow. The current
AI session continues normally (it is NOT ended).

WHEN TO USE:
- User wants a separate call placed to someone: "call John and remind him about the meeting"
- A callback / notification call should be triggered to a third party
- You need to start an outbound call that runs a predefined scenario (flow)

WHEN NOT TO USE:
- User wants to be transferred / connected to someone in THIS call (use connect_call)
- User wants to end the current call (use stop_flow / stop_service)

DIFFERS FROM connect_call:
- create_call = NEW independent call, NOT bridged, current session continues
- connect_call = bridges another party INTO the current call, ends the AI session

run_llm: Set true (default) to confirm verbally ("I've placed the call").`,
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "run_llm": map[string]any{
                "type": "boolean",
                "description": "Set true to confirm verbally after placing the call.",
                "default": true,
            },
            "flow_id": map[string]any{
                "type": "string",
                "description": "UUID of the pre-existing flow the new call will execute. Must belong to your account.",
            },
            "source": map[string]any{ /* optional address object: type, target, target_name */ },
            "destinations": map[string]any{ /* array of address objects: type(tel|sip|...), target, target_name */ },
            "anonymous": map[string]any{
                "type": "string",
                "description": "Optional caller-ID privacy: yes | no | auto (default auto).",
            },
        },
        "required": []string{"flow_id", "destinations"},
    },
},
```

`source` and `destinations` reuse the same address-object shape already used by `connect_call` / `send_message` for LLM consistency. `source` is optional (call-manager fills default).

**`run_llm` semantics (clarified, addresses round-2 concern).** `run_llm` governs ONLY whether the CURRENT AI session speaks/continues after the tool returns (e.g. "I've placed the call"). It does NOT arm or attach an LLM to the originated call. The originated call's behavior is fully determined by its own `flow_id` (which may or may not contain an `ai_talk` action). Therefore `run_llm` default `true` does NOT itself create the recursion vector; the amplification surface comes from the originated `flow_id` content (§7), not from this flag. `run_llm` default `true` is intentional (decision #5): the AI confirms placement to the user, which is the desired UX. `anonymous` is normalized in call-manager (only `yes`/`no` honored; anything else → `auto`), so an out-of-enum value is safe; the handler passes it through verbatim.

## 6. Handler flow (`pkg/aicallhandler/tool.go`, touch point 4)

Dispatch entry in `mapFunctions`:
```go
message.FunctionCallNameCreateCall: h.toolHandleCreateCall,
```

Handler pseudocode:
```go
func (h *aicallHandler) toolHandleCreateCall(ctx, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
    res := newToolResult(tool.ID)

    // 1. parse args
    var args struct {
        FlowID       uuid.UUID                `json:"flow_id"`
        Source       *commonaddress.Address   `json:"source,omitempty"`
        Destinations []commonaddress.Address  `json:"destinations"`
        Anonymous    string                   `json:"anonymous,omitempty"`
    }
    if err := json.Unmarshal([]byte(tool.Function.Arguments), &args); err != nil { fillFailed(res, err); return res }
    if args.FlowID == uuid.Nil { fillFailed(res, fmt.Errorf("flow_id is required")); return res }
    if len(args.Destinations) == 0 { fillFailed(res, fmt.Errorf("at least one destination is required")); return res }
    for _, d := range args.Destinations { // input hygiene: empty target would be silently skipped by call-manager
        if d.Target == "" { fillFailed(res, fmt.Errorf("destination target must not be empty")); return res }
    }

    // 2. SECURITY: flow ownership (IDOR prevention)
    f, err := h.reqHandler.FlowV1FlowGet(ctx, args.FlowID)
    if err != nil { fillFailed(res, errCouldNotResolveFlow); return res }            // bare constant, no %w
    if f.CustomerID != c.CustomerID { fillFailed(res, errCouldNotResolveFlow); return res } // byte-identical to not-found

    // 3. originate — masterCallID = uuid.Nil (no bridge), source nil-safe
    src := commonaddress.Address{}
    if args.Source != nil { src = *args.Source }
    calls, groupcalls, err := h.reqHandler.CallV1CallsCreate(
        ctx, c.CustomerID, args.FlowID, uuid.Nil /*masterCallID: no master*/, &src, args.Destinations,
        false /*earlyExecution*/, false /*connect/executeNextMasterOnHangup: load-bearing — see Key points*/, args.Anonymous, nil)
    // CONTRACT (verified): CreateCallsOutgoing returns an error ONLY when ALL destinations fail
    // (returns nil,nil,err). Partial/full success returns (calls,groupcalls,nil). There is NO
    // (err + non-empty slices) case, so a single err!=nil check is correct and total.
    if err != nil { fillFailed(res, err); return res }

    // 4. return originated IDs for tracking (decision #5)
    //   Per-destination failures are SWALLOWED by call-manager (logged + continue), so the only
    //   honest partial signal is requested-vs-created count. Surface it explicitly.
    //   INVARIANT (load-bearing): each destination yields at most ONE leg — one call OR one
    //   groupcall (fan-out is encapsulated inside a single groupcall, returning one groupcall_id).
    //   So created <= requested always. If the call-manager contract ever fans a single
    //   destination into multiple call_ids, this `partial` computation must be revisited.
    ids := struct {
        CallIDs      []string `json:"call_ids"`
        GroupcallIDs []string `json:"groupcall_ids"`
        Requested    int      `json:"requested"`
        Created      int      `json:"created"`
        Partial      bool     `json:"partial,omitempty"`
    }{Requested: len(args.Destinations)}
    for _, cc := range calls      { ids.CallIDs = append(ids.CallIDs, cc.ID.String()) }
    for _, gc := range groupcalls { ids.GroupcallIDs = append(ids.GroupcallIDs, gc.ID.String()) }
    ids.Created = len(ids.CallIDs) + len(ids.GroupcallIDs)
    ids.Partial = ids.Created < ids.Requested
    body, _ := json.Marshal(ids)

    // primary handle + correct resource type (groupcall-only must not be tagged "call")
    primaryID, primaryType := "", "call"
    if len(calls) > 0 {
        primaryID = calls[0].ID.String()
    } else if len(groupcalls) > 0 {
        primaryID, primaryType = groupcalls[0].ID.String(), "groupcall"
    }
    fillSuccess(res, primaryType, primaryID, string(body))
    return res
}
```

`errCouldNotResolveFlow` is a package-level bare sentinel (`errors.New("could not resolve flow")`) so both the not-found and cross-customer paths return a byte-identical message (no `%w` wrap, no interpolation) — the tool cannot be used as a flow-existence oracle.

Key points:
- **No `ReferenceType` restriction** (works from any session) — satisfies decision #4. `customerID` is sourced from `c.CustomerID` (the aicall record), NOT from any call-only field. `source` comes only from args or call-manager default-fill — never dereferenced from the current call. The handler touches NO call-only field, so it is safe from conversation/task/chat. A regression test originating from a conversation aicall (no call reference) is mandatory.
- **`masterCallID = uuid.Nil`** → the originated call is NOT bridged into the current session, and the current AIcall is NOT terminated. This is the core differentiator. Termination in `connect_call` is a SEPARATE explicit goroutine (`AIV1AIcallTerminate`); by simply not replicating it, the aicall survives.
- **8th bool = `false` (CRITICAL correction from v1).** v1 wrongly set this `true` with the rationale "run the call's own flow". That rationale was factually wrong. Verified in code: this positional arg maps to call-manager `CreateCallOutgoing(executeNextMasterOnHangup)` and is stored as `call.Data["execute_next_master_on_hangup"]`. It governs ONLY whether a MASTER call's flow advances when this (chained) call hangs up. It is NOT a bridge trigger and is NOT what makes the originated call run its own flow. The originated call runs its own `flow_id` automatically on answer via `status.go updateStatusProgressing → ActionNext`, independent of this bool. Since `create_call` has NO master (`masterCallID=Nil`), the correct value is `false`. Setting `true` is semantically meaningless (no master to advance) and misleading; add an inline comment so a future reader doesn't "helpfully" flip it.
- **Return value**: full list of originated `call_ids` + `groupcall_ids` in `Message` (JSON), plus `requested`/`created` counts and a `partial` bool (true when `created < requested`); primary id in `ResourceID` and `ResourceType` set to `call` or `groupcall` accordingly. `CreateCallsOutgoing` returns an error ONLY when ALL destinations fail, so total failure → `fillFailed`; any success (full or partial) → `fillSuccess` with the counts revealing partial drops. Satisfies decision #5 (tracking).
- `source` omitted/empty is intentional — call-manager fills default (§4).

## 7. Security & Compliance

The ONLY ai-manager-layer security duty is **flow ownership (IDOR)**:
- The LLM supplies `flow_id`. Without validation, an attacker-influenced prompt could reference another customer's flow. → resolve `FlowV1FlowGet(flow_id)` and reject when `f.CustomerID != c.CustomerID`.
- Use a **single masked error message** ("could not resolve flow") for both "not found" and "cross-customer" so the tool is not an existence oracle for flow IDs.

Destination targets are NOT an IDOR concern (they are phone numbers / SIP URIs, not customer-scoped resource IDs); destination/source/permission/balance safety is fully enforced fail-closed by call-manager (§4).

PII / external-LLM: N/A — this tool sends no transcript to an external LLM; it only invokes an internal RPC.

`tool_names:["all"]` reality: putting `create_call` in `definitions.go` auto-exposes it to any AI configured with `"all"`. This is accepted; data-access safety is the flow-ownership check + call-manager fail-closed gating, not tool gating. Do NOT add `create_call` to the not-yet-wired `ConversationSafeTools`.

### Abuse / amplification risk (residual, accepted for Phase 1)

Because the originated call runs its own `flow_id`, that flow could itself launch another AI (e.g. an `ai_talk` action) whose aicall has `create_call` enabled, which originates another call, and so on. With `run_llm` default `true` the new call's AI is "armed" to chain further. This is a recursion/fan-out amplification vector specific to giving the AI call-origination power.

Why this is acceptable for Phase 1 WITHOUT new controls (consistent with the decision to delegate all gating to call-manager):
- Every originated leg traverses call-manager `CreateCallOutgoing`, which is fail-closed on **balance** (`ValidateCustomerBalance`), verified-customer status, and PSTN whitelist. Runaway origination drains balance and is then rejected — there is a hard money backstop.
- Phase 1 uses `flow_id` reference (not LLM-authored actions), so the recursion requires the customer to have deliberately built a flow that re-launches an AI with `create_call`. It is not trivially LLM-inducible.

Residual risk acknowledged: balance is a backstop, not a rate limiter; a customer can still burn money/concurrency quickly. **Phase 2 hardening (Open Question #5):** per-customer rate limit on AI-originated calls, an origination depth/hop counter propagated via call metadata (cap ~2-3), and a destinations-per-invocation cap. Deferred, not built in Phase 1, to keep the change minimal.

## 8. Observability

- `promAIcallToolExecuteTotal{function="create_call"}` is incremented automatically by the existing dispatch path (`ToolHandle`). No new metric required.
- Errors logged via the handler's `logrus.WithFields` (aicall_id, flow_id, customer_id). Mirror the existing `toolHandle*` logging convention.

## 9. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-ai-manager | `models/tool/main.go`: `ToolNameCreateCall` + `AllToolNames`; `models/message/tool.go`: `FunctionCallNameCreateCall`; `toolhandler/definitions.go`: tool def; `aicallhandler/tool.go`: dispatch + `toolHandleCreateCall` | 1 |
| bin-ai-manager | Tests: `toolhandler/main_test.go::TestAllToolNames` update; `aicallhandler/tool_test.go` new cases | 1 |
| bin-call-manager | None (reuses `CallV1CallsCreate`) | - |
| bin-flow-manager | None (reuses `FlowV1FlowGet`) | - |
| bin-pipecat-manager | None (tool framework already feeds `run_llm`) | - |

## 10. Implementation Order

1. `models/tool/main.go`: add `ToolNameCreateCall` + append to `AllToolNames`.
2. `models/message/tool.go`: add `FunctionCallNameCreateCall`.
3. `pkg/toolhandler/definitions.go`: add the `create_call` tool definition.
4. `pkg/aicallhandler/tool.go`: add dispatch entry + `toolHandleCreateCall`.
5. Tests: update `TestAllToolNames`; add `toolHandleCreateCall` table-driven tests (success single, success multi, success groupcall-only → `rType="groupcall"`, success mixed call+groupcall → `rType="call"`, partial success → `partial:true` + `requested>created`, total failure → fillFailed, missing flow_id, missing destinations, empty destination target → fillFailed, cross-customer flow → masked, flow-get error → masked, byte-identical masked-message assertion for not-found vs cross-customer, **originate from conversation/task aicall with no call reference → no call-only deref**, **assert aicall remains Active after success (8th bool=false, no termination)**). Verify `run_llm` default true.
6. `bin-ai-manager/docs/`: update `architecture.md` (tool list) + `domain.md` (Tool section) for the new tool.
7. Full verification: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.

## 11. Open Questions

| # | Question | Recommendation | Owner |
|---|---|---|---|
| 1 | 8th positional bool semantics (RESOLVED in v2) | Set `false`. Confirmed it maps to `executeNextMasterOnHangup` (master-flow advance on chained hangup), irrelevant with no master. Originated call self-executes via `ActionNext` on answer. | Resolved |
| 2 | Should `create_call` pre-check `flow.Status` before originating? | Defer; `FlowV1FlowGet` already resolves the flow (ownership), and call-manager activeflow creation fail-closes on an unusable flow. Phase 2 polish. | Phase 2 |
| 3 | Multi-destination: one tool call → N independent calls vs single | Allow N (reuse `CallV1CallsCreate` array). Tool result returns all ids + partial_error. | Confirmed |
| 4 | Phase 2 inline `actions[]` (LLM-authored flow) | Deferred; document migration (flow_id and actions mutually exclusive). | Phase 2 |
| 5 | AI-origination amplification (recursion/fan-out) | Phase 1: balance fail-closed is the backstop, no new control. Phase 2: per-customer rate limit + origination depth/hop counter (cap ~2-3) + destinations-per-call cap. | Phase 2 |
| 6 | Partial-success contract (some legs created, some failed) | Return created IDs + `partial_error` in tool result; `fillFailed` only when zero legs created. | Confirmed (v2) |

## 12. Review Summary

### Round 1 (2 independent reviewers, fresh delegate_task)

Reviewer A produced no usable output (aborted early). Reviewer B (skeptical Go engineer + CPO) returned CHANGES REQUESTED with the following, all addressed in v2:

| Severity | Finding | Resolution in v2 |
|---|---|---|
| CRITICAL | 8th bool `connect=true` contradicts `masterCallID=Nil`. | Verified in call-manager code: arg maps to `executeNextMasterOnHangup` (stored as `call.Data["execute_next_master_on_hangup"]`), governs master-flow advance only, NOT bridging and NOT self-flow execution. Originated call self-executes via `status.go → ActionNext`. Changed to `false` with load-bearing inline comment. v1's "run own flow" rationale was factually wrong and removed. |
| HIGH | AI-originates-AI recursion/fan-out, no depth/rate caps. | Added "Abuse / amplification risk" subsection. Phase 1 relies on call-manager balance fail-closed backstop (consistent with delegate-gating decision); Phase 2 hardening (rate limit + depth counter + dest cap) recorded as Open Question #5. |
| HIGH | Call-only field assumptions (`customerID`/`source`) when run from conversation/task. | Clarified: `customerID` from `c.CustomerID` (aicall record), `source` from args or call-manager default-fill only. Handler touches no call-only field. Mandatory regression test: originate from a conversation aicall (no call reference). |
| MEDIUM | Groupcall return + partial-success/both-empty handling. | Handler now: `fillFailed` only when err AND zero legs; partial success returns created IDs + `partial_error`; primary id = first call else first groupcall. Open Question #6. |
| LOW | `masterCallID=Nil` sufficiency for non-termination. | Confirmed: termination is a separate `AIV1AIcallTerminate` goroutine in connect_call; not replicating it keeps the aicall alive. Added assertion to test plan. |
| LOW | `earlyExecution=false` hardcoded. | Documented as intentional for an independent call. |

Verification of the CRITICAL finding was done directly against call-manager source (`outgoing_call.go`, `status.go`, `hangup.go`, `chained_call.go`), not taken on the reviewer's assertion alone — the reviewer's conclusion (`false`) was correct but its stated mechanism (bridge trigger) was inaccurate; the real mechanism is master-flow advance.

### Round 2 (2 independent reviewers, fresh delegate_task, pure document review)

Both reviewers independently returned CHANGES REQUESTED and converged on the same top finding. All addressed in v3.

| Severity | Finding | Resolution in v3 |
|---|---|---|
| MEDIUM (both) | `partial_error` field + `if err && len==0` guard is DEAD CODE. Verified contract: `CreateCallsOutgoing` returns an error ONLY when ALL destinations fail (`nil,nil,err`); partial success returns `(calls,groupcalls,nil)` with NO error. There is no `err + non-empty slices` state, so `partial_error` could never populate and the `len==0` conjunct was tautological. | Removed `partial_error` and the conjunct. Error check is now a single total `if err != nil { fillFailed }`. Real partial-failure (per-destination skips, swallowed by call-manager) is now surfaced honestly via `requested` vs `created` counts + a `partial` bool. |
| LOW–MEDIUM (R-A) | `fillSuccess("call", ...)` hardcoded `rType="call"` even when `primaryID` falls back to a groupcall id (groupcall-only result mislabeled). | `primaryType` now `"groupcall"` when only groupcalls returned, `"call"` otherwise. Test added. |
| MEDIUM (both) | No per-destination input validation; empty target silently swallowed by call-manager. | Added empty-target rejection (`fillFailed`) before the RPC. Test added. |
| LOW (both) | Masked-error oracle only holds if message is a bare constant (no `%w`). | Introduced package-level `errCouldNotResolveFlow` sentinel; both paths return it byte-identically. Added byte-identical assertion test. |
| MEDIUM (R-B) / N/A | Reviewer B recommended `run_llm` default `false`. | Kept `true` per decision #5, but clarified semantics: `run_llm` governs ONLY the current session's verbal confirmation; it does NOT arm/attach an LLM to the originated call. The recursion vector is the originated `flow_id` content, not this flag. Reviewer's amplification concern is thus decoupled from the default. |
| LOW (both) | `anonymous` enum not validated. | Documented that call-manager normalizes it (only `yes`/`no` honored, else `auto`), so passthrough is safe; no handler-side validation needed. |
| LOW (R-B) | Confirm strict `CustomerID` equality (not loose). | Handler uses `f.CustomerID != c.CustomerID` strict equality. Confirmed. |
| Deferred (R-B) | Self-dial dedupe / `dest==source` rejection. | Noted as Phase 2 input-hardening alongside Open Question #5 (out of P1 minimal scope; not a tenant-isolation break). |

Both rounds confirm: tenant isolation (flow ownership), the 8th-bool correctness, aicall-stays-alive, and the masked-error path are all sound. No Critical items remain open.

### Round 3 (2 independent reviewers, fresh delegate_task, confirmation pass on v3)

Both reviewers returned **APPROVE**. Confirmed: count logic (`partial = created < requested`) correct for full/partial/groupcall-only/mixed; masking still byte-identical; empty-Target guard does not over-reject groupcall-type addresses (they carry a non-empty group/agent identifier in Target); no stale `partial_error` reference; primaryType selection correct in mixed case. Two non-blocking notes, both resolved:
- `fillSuccess` rType is a free-form `string` field on `messageContent` (verified in code: `pkg/aicallhandler/tool.go` L93/127-129), NOT validated against `aicall.ReferenceType`. So `"groupcall"` is safe — no regression.
- Added a load-bearing INVARIANT comment (`created <= requested`, one destination ⇒ ≤1 leg) so a future call-manager fan-out change doesn't silently break the `partial` flag.

Final status: v3 APPROVED across 3 review rounds (6 reviewer passes). Ready for implementation.
