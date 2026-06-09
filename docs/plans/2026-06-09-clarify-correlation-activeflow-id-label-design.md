# Design: Clarify activeflow_id label in get_correlation summary output

Status: Draft (v1)
Author: CPO (with pchero)
Date: 2026-06-09
Service: bin-ai-manager

## 1. Problem Statement

The `get_correlation` tool returns a human/LLM-readable summary whose first line is:

```
Call flow 923aba67-78be-46b8-9b9d-d8ef53cda9cd is linked to:
```

The UUID printed after the words "Call flow" is in fact the **activeflow_id** (`corr.ActiveflowID`, see `pkg/aicallhandler/tool.go::formatCorrelationSummary` L781). However the label "Call flow" does not make that explicit. In a real diagnostic session (2026-06-09) this caused confusion: when looking for "the activeflow_id" in a transcript, the reader could not tell that the value behind the "Call flow" label *was* the activeflow_id, especially because several flow-related IDs coexist in one trace:

- the current AI session's `activeflow_id` (on every message record),
- the `flow_id` passed to `create_call` (a flow *definition* / template, NOT an activeflow),
- the originated outbound `call_id`,
- the originated call's own `activeflow_id` (surfaced only via `get_correlation` as "Call flow ...").

The ambiguity is purely a labeling/observability issue. The data is correct.

## 2. Goal

Make the summary output state, unambiguously, that the leading UUID is the `activeflow_id`, WITHOUT renaming the broader "call flow" concept (the tool's own description teaches the LLM that a "call flow execution" is the unit of correlation, see `pkg/toolhandler/definitions.go` L487-501). Minimal, behavior-preserving copy change.

## 3. Scope

### In scope
- Change the first line of `formatCorrelationSummary` to label the UUID as `activeflow_id`.
- Update the two test expectations in `pkg/aicallhandler/tool_correlation_test.go` (L92, L136) that assert the exact summary string.

### Out of scope
- No change to the tool definition/description (the "call flow execution" concept wording stays).
- No change to `create_call` result payload (whether to also surface activeflow_id there is a separate, deferred question, see §7).
- No change to the resource-line formatting (`- call <id>`, `- transcribe <id>`, etc.).
- No new entity / DB / REST / webhook. N/A.

## 4. Proposed change

`pkg/aicallhandler/tool.go` L781:

Current:
```go
fmt.Fprintf(&sb, "Call flow %s is linked to:\n", corr.ActiveflowID)
```

Proposed:
```go
fmt.Fprintf(&sb, "Call flow (activeflow_id: %s) is linked to:\n", corr.ActiveflowID)
```

Resulting first line:
```
Call flow (activeflow_id: 923aba67-78be-46b8-9b9d-d8ef53cda9cd) is linked to:
```

Rationale for this exact form (vs alternatives):
- `Activeflow %s is linked to:` — drops the "call flow" concept word the tool description relies on; risks LLM/term inconsistency. Rejected.
- `Call flow (activeflow_id: %s) is linked to:` — keeps the concept word AND names the identifier precisely. A human or LLM searching for "activeflow_id" now finds it verbatim. Chosen.
- The phrasing also disambiguates from `flow_id` (the definition), which is the other flow-related identifier in the same trace.

## 5. Affected files

| File | Change |
|---|---|
| `bin-ai-manager/pkg/aicallhandler/tool.go` | L781 format string |
| `bin-ai-manager/pkg/aicallhandler/tool_correlation_test.go` | L92, L136 expected `Message` strings |

## 6. Verification

Run the full mandatory workflow in `bin-ai-manager`:
```
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Focus: `Test_toolHandleGetCorrelation` and `Test_toolHandleGetCorrelation_maskingInvariant` must pass with the new expected strings. The masking invariant test asserts the not-accessible path returns `msgCorrelationNotFound` (a constant unaffected by this change), so it must remain byte-identical.

## 7. Open question (deferred)

Should `create_call` tool result also include the originated call's `activeflow_id` so a follow-up `get_correlation` is not required? Deferred: at `create_call` return time the outbound call may not yet have an activeflow (it is created on answer via flow execution), so this is non-trivial and out of this minimal scope. Tracked separately.

## 8. Review Summary

### Design review (2 independent reviewers, fresh delegate_task)

Both reviewers returned APPROVE.

- Reviewer A (skeptical Go engineer): confirmed `Call flow (activeflow_id: %s)` is the correct minimal fix (preserves the "call flow execution" concept the tool description teaches the LLM while naming the identifier verbatim); all affected sites identified (tool.go L781 + test L92/L136, no other renderer/asserter); no risk to the masking-invariant test (asserts `msgCorrelationNotFound`, a separate constant) or the unlinked-resource branch; deferred open-question correctly scoped. Non-blocking note: the prior design doc (2026-06-08) L237 documents the old format and could be updated for consistency.
- Reviewer B (product/observability): confirmed the parenthetical "label (field_name: value)" pattern resolves the confusion without term drift; resource lines (`- call <id>`, `- transcribe <id>`) already carry a type label and were never the confusion source, correctly out of scope; flow_id-vs-activeflow_id distinction belongs to the deferred §7 create_call item, not here.

Action taken on the non-blocking note: prior design doc L237 updated to the new format.

