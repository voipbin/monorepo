# Design: Use "activeflow" terminology in get_correlation output and description

Status: Draft (v2)
Author: CPO (with pchero)
Date: 2026-06-09
Service: bin-ai-manager

## 1. Problem Statement

The `get_correlation` tool returns a human/LLM-readable summary whose first line was:

```
Call flow 923aba67-78be-46b8-9b9d-d8ef53cda9cd is linked to:
```

The UUID printed after the words "Call flow" is the **activeflow_id** (`corr.ActiveflowID`, see `pkg/aicallhandler/tool.go::formatCorrelationSummary`). Two problems:

1. **Ambiguous label.** In a real diagnostic session (2026-06-09) a reader could not tell that the value behind the "Call flow" label was the activeflow_id, because a single trace contains several flow-related ids: the current session's `activeflow_id`, the `flow_id` passed to `create_call` (a flow *definition*, NOT an activeflow), the originated `call_id`, and the originated call's own `activeflow_id` (surfaced only via `get_correlation`).

2. **"Call" bias is factually wrong.** An activeflow is the running instance of a flow, and its reference is NOT always a call. The activeflow `ReferenceType` enum (`bin-flow-manager/models/activeflow/activeflow.go`) is: none, ai, api, call, campaign, conversation, transcribe, recording. Calling it a "call flow" wrongly narrows it to telephony. The same bias appears throughout the tool description ("call flow execution", "the current session's call flow"), which can push the LLM to assume the session is a phone call even when it is not.

The data is correct; this is a terminology/observability fix.

## 2. Goal

Use the precise domain term **activeflow** consistently in (a) the summary output and (b) the tool description, removing the telephony ("call") bias. Minimal, behavior-preserving copy change.

## 3. Scope

### In scope
- `formatCorrelationSummary` first line: `Activeflow %s is linked to:`.
- The own-session unlinked message: `This resource exists but is not linked to any activeflow.`
- `get_correlation` tool description (`pkg/toolhandler/definitions.go`): replace "call flow" wording with "activeflow", and add an explicit note that the activeflow reference may be call, conversation, ai, api, campaign, transcribe, or recording (counter the call-only assumption).
- Update test expectations in `pkg/aicallhandler/tool_correlation_test.go` (summary strings + unlinked message).
- Refresh the historical design doc example (`2026-06-08-add-correlation-llm-tool-design.md`).

### Out of scope
- No change to `create_call` result payload (surfacing originated activeflow_id is a separate, deferred item).
- No change to resource-line formatting (`- call <id>`, `- transcribe <id>`, etc.). Those carry a correct type label already.
- No new entity / DB / REST / webhook. N/A.

## 4. Proposed change

`pkg/aicallhandler/tool.go`:

Summary line:
```go
fmt.Fprintf(&sb, "Activeflow %s is linked to:\n", corr.ActiveflowID)
```
Resulting first line:
```
Activeflow 923aba67-78be-46b8-9b9d-d8ef53cda9cd is linked to:
```

Unlinked message:
```go
fillSuccess(res, "correlation", resourceID.String(), "This resource exists but is not linked to any activeflow.")
```

Rationale for `Activeflow %s` (form 1) over `Call flow (activeflow_id: %s)`:
- The word "Activeflow" itself tells the reader the UUID is an activeflow_id; no parenthetical needed.
- It is the actual domain term and is reference-type neutral (not "call").
- A transcript search for "activeflow" now matches.

`pkg/toolhandler/definitions.go` description: replace "call flow" with "activeflow" throughout and add a sentence clarifying the reference may be call, conversation, ai, api, campaign, transcribe, or recording (and may be unset), so the LLM does not over-assume telephony.

## 5. Affected files

| File | Change |
|---|---|
| `bin-ai-manager/pkg/aicallhandler/tool.go` | summary line + unlinked message |
| `bin-ai-manager/pkg/toolhandler/definitions.go` | get_correlation description (de-bias from "call") |
| `bin-ai-manager/pkg/aicallhandler/tool_correlation_test.go` | expected summary + unlinked strings |
| `bin-ai-manager/docs/plans/2026-06-08-add-correlation-llm-tool-design.md` | historical example line |

## 6. Verification

Full mandatory workflow in `bin-ai-manager`:
```
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Focus: `Test_toolHandleGetCorrelation` (summary + unlinked cases) and `Test_toolHandleGetCorrelation_maskingInvariant` (asserts `msgCorrelationNotFound`, an unrelated constant, must stay byte-identical).

## 7. Open question (deferred)

Should `create_call` tool result also include the originated call's `activeflow_id`? Deferred: at create_call return time the outbound call may not yet have an activeflow (created on answer via flow execution). Tracked separately.

## 8. Review Summary

### Design review round 1 (2 independent reviewers)

Both APPROVE on the earlier "Call flow (activeflow_id: %s)" form. Reviewer A's "keep the call flow concept word" guidance was subsequently overruled by the domain owner (pchero): "call flow" is factually wrong because an activeflow reference is not always a call (the activeflow ReferenceType enum is none, ai, api, call, campaign, conversation, transcribe, recording). v2 therefore uses the reference-type-neutral term "activeflow" in both output and description, and de-biases the description away from telephony.

### Design review round 2 (2 independent reviewers, v2)

Both CHANGES REQUESTED: the de-bias sentence I added first listed invented activeflow reference types ("chat", "task"). "task" is an aicall reference type, not an activeflow one, and "chat" is not in any enum. Corrected to the authoritative activeflow ReferenceType set (none, ai, api, call, campaign, conversation, transcribe, recording) in both the description and this doc.
