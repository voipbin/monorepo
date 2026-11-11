# Insight Assistant: draft_call and the call draft decision

Date: 2026-11-11
Verified against: origin/main `346df1cba`, then rebased onto the latest origin/main before the PR (no conflict in sources)
New test files: `pkg/aicallhandler/tool_draft_call_test.go`, `pkg/aicallhandler/draft_decision_call_test.go`
Ticket: VOIP-1717
Services: bin-ai-manager, bin-pipecat-manager (tests only), bin-api-manager (docs and tests only), bin-openapi-manager
Related: `2026-11-10-insight-assistant-sms-design.md` (VOIP-1704, 1708), `2026-11-11-insight-assistant-email-design.md` (VOIP-1710). This document extends that pattern to outbound phone calls. It does not revise those documents.

## 1. Ticket review (issue analysis)

All statements below were checked against origin/main `346df1cba` (fetched 2026-11-11).

| Claim | Result | Evidence |
|---|---|---|
| `create_call` exists as an AI tool | Valid | `FunctionCallNameCreateCall` -> `toolHandleCreateCall` (`pkg/aicallhandler/tool.go`). It places a new independent outbound call, not bridged to the session, and does not end the AIcall. Arguments: `flow_id` XOR `actions`, `source`, `destinations` (many), `anonymous`, `variables` |
| An Insight AI cannot use `create_call` | Valid | not in `tool.AllInsightToolNames`, `InsightMessageToolNames`, `InsightEmailToolNames`; `ai.AllowedToolNames(TypeInsight)` is the union of those three, so a save with `create_call` is rejected. The comment on `AllInsightToolNames` still says "Calls and CRM writes remain unavailable to Insight AIs" |
| `draft_call`, `call_draft` do not exist | Valid | `git grep -i -E "draft_call|call_draft|BlockTypeCallDraft"` on origin/main: no hit |
| No duplicate ticket or PR | Valid | Jira text search "call draft", "draft_call", "create_call insight": no open match (SQUARE-136 is the email UI); no PR |
| The decision path is generic enough to extend | Partly | `DraftDecision` selects the send by `isEmail` else SMS (two-way `if`). A third block type needs a real switch. `draftDecisionLoad` accepts only `sms_draft` with `To` or `email_draft` with `Recipients` |

What call-manager already enforces for every `CallV1CallsCreate` (`CreateCallOutgoing`, `bin-call-manager/pkg/callhandler`), so a draft does not have to repeat it:

- customer status active, and for a tel destination the customer identity must be verified (`validateOutgoingCallPermission`);
- account balance (`ValidateCustomerBalance`);
- a tel destination must be in the customer's `OutboundConfig.DestinationWhitelist` (country level, fail closed: no config, empty list or unknown country denies) (`ValidateDestination`);
- the source number must be an active normal number owned by the customer, otherwise it falls back to `DefaultOutgoingSourceNumberID`, otherwise the call is rejected (`getValidatedSourceForOutgoingCall`). The tool path passes the LLM's `source` through, so the owned-number check is the only protection against a chosen caller id.

What call-manager does not do: no per-customer call rate limit (`grep -i ratelimit` in `pkg` finds none), no destination count limit in `CreateCallsOutgoing`.

Risks specific to a call (beyond SMS and email):

- A call cannot be undone. A rung phone is a customer-visible event the moment the call is created, and PSTN minutes are charged to the customer account.
- `create_call` accepts `actions`: an LLM written action chain run on the callee's answer. Untrusted call or message content can steer it (prompt injection) toward any action, including speaking arbitrary text to the callee and connecting to other numbers.
- `create_call` accepts a list of destinations, so one tool call can ring many numbers.
- Listen turns (no agent present) run with the AI's own `tool_names`; `ToolHandle` has no AI-type or listen-turn gate (unchanged by this ticket).
- Approver authority is lower than the action's: `ServiceAgentAImessageDraftDecision` requires `PermissionAll` (api-manager `serviceagent_aimessage.go`; `Agent.HasPermission` returns true for any agent when `perm == PermissionAll`), so any agent of the customer can decide any draft. `POST /calls` and flow create/update require `PermissionCustomerAdmin|PermissionCustomerManager` (`servicehandler/call.go`, `flow.go`). A call draft approval would therefore let an ordinary agent cause a PSTN call that agent could not place directly. For SMS and email this gap was accepted; for a call it is a privilege escalation with real minutes.
- A stored flow is not a single-destination action: a flow may hold `call`, `connect` (several destinations), `message_send`, `email_send` actions (`bin-flow-manager/models/action/action.go`). Choosing `flow_id` therefore does not bound the outbound effects to one number. Read in flow-manager `actionhandle.go`: `connect` (line ~548) and `call` (~958) go through `CallV1CallsCreate`, so they get the same call-manager checks (customer status, identity, balance, country whitelist, source ownership). `message_send` and `email_send` go to message-manager and email-manager, which check identity and balance; no destination country whitelist was found in either, so a flow-driven SMS or email is not whitelist-checked. Flows are NOT admin/manager only: `servicehandler/flow.go` lets any agent create a non-persistent (cache-only, TTL) flow with arbitrary actions (`persist=false` needs `PermissionAll`), Existence plus same customer is therefore not a trust check. (Corrected in design review: `Flow.Persist` is not a database column but is set on read, `true` for database rows and false for cache-only flows, and it is serialized as `persist`, so ai-manager can tell them apart; the card shows it as `flow_persistent`, a label only, no restriction.) The decision (section 2, item 3) is that any agent of the customer may approve, as for SMS and email; the approver therefore does not need the authority to place the call directly, and that gap is an accepted risk (section 5). The card also lists the action types of the flow (for example `connect, message_send`) so the approver sees what the flow can send; the card does not claim the flow is vetted.
- TOCTOU: re-reading the flow at approval proves existence and ownership, not that its actions are unchanged since the card was shown. Flows referenced from the top flow (a `call` action's `flow_id`, `on_complete_flow_id`) can change without changing the top flow's `tm_update`; that indirect change is not covered. `flow_name` on the card is the draft-time value.
- A destination-level failure in call-manager (identity, whitelist, balance) is logged and skipped; the tool receives only `all destinations failed to create outgoing calls`. The claim row is already consumed, so the draft is spent and the reason is not shown to the agent.
- api-manager applies `middleware.RateLimit("v1")` and a per-customer limit to the HTTP decision call, but the AI tool path (`CallV1CallsCreate` over RPC) bypasses them. Scope of these limits on the decision route was not verified.
- `ToolHandle` dispatches by name with no gate, so keeping `create_call` away from Insight AIs relies only on the tool exposure layer (`AllowedToolNames`, runtime expansion), not on the handler.
- The `["all"]` selector of an Insight AI already expands to `send_message` and `send_email` (VOIP-1708, VOIP-1710), so an AI stored with `["all"]` can send an SMS or an email directly with no human step. That behavior is already shipped. The call tools below are specified explicitly so that it is clear what `["all"]` includes.

Verdict: valid. Not already solved, not a duplicate, no changed premise.

## 2. Decisions

1. Design B: the Insight Assistant gets both `create_call` (sends at once, the existing tool reused as is) and a draft tool, as for SMS and email. The customer chooses by the AI prompt and `tool_names`. Naming: the new tool is `draft_call`, the card block is `call_draft`. Basis: tool names in this code base are verb first (`send_message`, `send_email`, `create_call`, `connect_call`, and the drafts `draft_message`, `draft_email`) and card blocks are noun first (`sms_draft`, `email_draft`); `create_call` already exists for Normal AIs and renaming it would break stored AIs. A resource-first scheme (`call_create`, `call_draft`) is possible only for the new names and would break the pair with `draft_message`/`draft_email`. Renaming `draft_call` before merge is a mechanical change.
2. The `["all"]` selector of an Insight AI includes every call tool: `create_call`, `draft_call` (and `describe_action`, see section 3).
3. Any agent of the customer may approve or reject a call draft (the permission of the existing decision endpoint, `PermissionAll`), as for SMS and email. No api-manager or common-handler code change is needed.
4. No v1 restriction on the call arguments: every destination type, `flow_id` or `actions`, `source`, `variables`, `anonymous`, as `create_call` accepts them. Misuse is controlled by the customer's AI prompt. The only limits are technical (section 4).

## 3. Goal, tool set and contract

Goal: an Insight Assistant can place an outbound call directly (`create_call`) or propose one for the agent to review (`draft_call`). When the agent approves a draft card, the platform places the stored call in code, without an LLM turn. Reject places nothing.

Non-goals: a mode field, a schema migration, a per-hour limit, a destination count limit, a refusal during a listen turn, an AI-type gate inside `ToolHandle`, a change of `InsightSystemPrompt`, the frontend.

Tool lists. A new `tool.InsightCallToolNames = [create_call, draft_call, describe_action]`, separate from `AllInsightToolNames` so `TestAllInsightToolNamesAreReadOnly` is not touched (`create_call` has outside effects). `describe_action` is read-only (a static action catalog lookup) and is needed because `create_call` with `actions` tells the model to look up option shapes with it; it stays in `tool.AllToolNames` for Normal AIs. `insightToolSet()` adds the new list, so `AllowedToolNames(TypeInsight)` and `ExpandAllToolNames(TypeInsight)` (the `["all"]` set) include it. `Test_InsightCallToolNames` pins the exact names. `draft_call` is not in `tool.AllToolNames`, so a Normal AI rejects it (as `draft_email`).

### draft_call arguments

The same as `create_call` plus a title: `flow_id` XOR `actions`, `destinations` (at least one), `source`, `anonymous`, `variables`, `title` (optional, at most 200 characters, default `Call draft`), `run_llm`. `draft_call` places nothing and makes one read call: when `flow_id` is given the flow is read (`FlowV1FlowGet`) to fill the card; a flow that is missing or belongs to another customer is refused with the same masked error as `create_call` (`could not resolve flow`).

### The call_draft card block (frontend contract)

```
{"type":"call_draft","title":"Call draft","call":{
  "destinations":[{"type":"tel","target":"+...","target_name":"..."}],
  "source":{"type":"tel","target":"+..."},            // omitted when not given
  "anonymous":"auto",                                  // omitted when not given
  "variables":{"k":"v"},                               // omitted when empty
  "flow_id":"<uuid>","flow_name":"...","flow_action_types":["connect"],"flow_persistent":true,"flow_destinations":[{"type":"tel","target":"+..."}],"flow_tm_update":"<RFC3339 nanoseconds or empty>",
  "actions":[{...}]                                    // exactly one of flow_id and actions is present
}}
```

`destinations` and `source` keep the whole address object (`type`, `target`, `target_name`). `flow_destinations` lists the destination addresses of the `connect` and `call` actions of the flow (read from the action options; an action whose option cannot be parsed contributes nothing and `flow_action_types` still shows its type), so the approver sees where the flow itself dials. When `source` is omitted call-manager uses the customer's default outgoing number; the frontend must say so instead of showing an empty source. `flow_name`, `flow_action_types` and `flow_destinations` are labels read at draft time and are not proof of trust (an ordinary agent can build a temporary flow with any name). The `tool` row id is the draft id. `describe_action`, `create_call` need no contract change.

### Decision endpoint

`POST /service_agents/aimessages/draft-decision` is unchanged (request, response, errors). The block type selects the action: `call_draft` approve runs the `create_call` handler in code with the stored arguments, from the aicall that owns the draft. Result message of an approved call is the handler's own result unchanged (a JSON string with `call_ids`, `groupcall_ids`, `requested`, `created`, `partial`); no fixed wording replaces it, and no ` Delivery may be unconfirmed.` suffix is added on failure because a `failed` result means no call was created (the handler returns an error only when every destination failed). A created call is not an answered call, and the frontend text must not say "connected". The execution metric label is `create_call` (the handler that ran), which is intended. Reject records the constant `draftCallRejectedMessage` = `The agent rejected this draft. The call was not placed.` A failed placement (the handler returned `failed`: every destination failed) places no call and consumes the draft (the claim row is already written); call-manager's reason is not in the result (it logs per-destination failures and returns one generic error).

Errors and guards, all existing: `DRAFT_NOT_FOUND` (same for every refusal about the row), `DRAFT_EXPIRED` (`aicall_draft_decision_expiry_seconds`, default 600, no new setting), `DRAFT_ALREADY_DECIDED` (claim row primary key), `INVALID_DECISION`, and `DRAFT_INVALID` (below).

## 4. Validation and limits

Only technical limits, no policy limits (decision 4):

- shape (the same function at draft time and at approval): exactly one of `flow_id` (a non-nil UUID) and `actions` (non-empty); at least one destination; no destination with an empty `target`; every action `type` is in `actioncatalog.ActionTypeEnum()` (local check, flow-manager validates the rest when the ephemeral flow is created); `variables` are stored as strings (same coercion as `create_call`, `fmvariable.NewVariablesFromMap`); `anonymous` is stored as given and call-manager normalizes it.
- `variables` limits of flow-manager (`activeflowhandler/variable.go`): flow-manager drops the whole `variables` map without any error when it has more than 100 keys, a value above its per-value limit (32 KiB) or a total of keys and values above 64 KiB, so an approved call would run without variables and still succeed. `validateCallDraft` therefore refuses a draft over those limits (the constants are in `activeflowhandler/main.go`: 100 keys, 32 KiB per value, 64 KiB total; a test pins them); flow-manager ignores a key that is empty or that starts with `voipbin.` after `TrimSpace` and lower-casing (`Voipbin.x` and ` voipbin.x` too); the draft refuses the same keys with the same normalization, and the limits are checked on the stringified map (`NewVariablesFromMap`), as flow-manager does so that the card never shows a variable that is not applied.
- stored size: the JSON of the stored tool row must not exceed `draftStoredMaxBytes` (60000), as for email, because `ai_messages.content` is a 65,535-byte text column. A larger draft is refused, never cut.
- flow binding (when `flow_id` is given): at approval, before the claim, the flow is read again; not found or another customer or a `tm_update` different from `flow_tm_update` gives `DRAFT_INVALID`. `tm_update` nil on both sides matches; nil against non-nil does not; the value is compared as the stored RFC 3339 nanosecond string, any parse or format loss counts as a mismatch (fail closed). A transient read error is returned as an error before the claim and does not consume the draft. This covers the top flow only; a flow referenced from it (`call` action `flow_id`, `on_complete_flow_id`) can change without changing the top flow's `tm_update` (stated gap).
- numbers (variables are stored as strings after `NewVariablesFromMap`, so the number problem is only in `actions` options; the tests pin the two separately): `draft_call` decodes its arguments with `UseNumber` too (a plain `json.Unmarshal` would already cut an integer above 2^53 to float64 before it is stored, and `NewVariablesFromMap` turns float64 `1.0` into `"1"` but keeps json.Number `"1.0"`); the round-trip test runs the whole path draft arguments, stored row, approval.
- numbers in `actions` options: `toolHandleCreateCall` decodes with `UseNumber`, so the stored row must be decoded the same way at approval (`draftDecisionLoad` decodes the stored content with `UseNumber`, and a round-trip test with an integer above 2^53 and `1.0` pins it); otherwise the stored arguments would not behave as shown on the card.
- `actions` have no snapshot problem: the actions are stored on the card and the ephemeral flow is created from the stored actions at approval.

Not limited, by decision: destination types and count, source, `variables`, `anonymous`, `actions` content, calls per time.

## 5. Accepted risks (stated plainly)

- Direct `create_call` for an Insight AI places a real call at once: PSTN minutes are charged to the customer account and the callee is called; it cannot be undone. An AI stored with `["all"]` can do this with no human step, including on a listen turn (no agent present) driven by untrusted call or message content. This is the same exposure as `send_message` and `send_email` (VOIP-1708, VOIP-1710) and is accepted by the decision; the customer controls it with the AI prompt and by naming tools instead of `["all"]` (for example `draft_call` together with `describe_action`, without `create_call`; `describe_action` must be named too, otherwise the model is told to use a tool it does not have).
- Retroactive widening: on deploy, every Insight AI already stored with `["all"]` gains `create_call`, `draft_call` and `describe_action` with no migration.
- `actions` on the callee's answer can say or do anything the action set allows, including `connect` to other numbers; `source` is checked by call-manager only for ownership (an owned number, otherwise the default number); `destinations` have no count limit in ai-manager or call-manager (call-manager enforces identity, balance and the country whitelist per call).
- Any agent of the customer can approve a call that agent could not place directly with `POST /calls` (that needs admin or manager). Accepted by decision 3.
- No rate limit on the AI tool path (RPC bypasses the api-manager limiter; call-manager has none).
- A decided draft is consumed even when placement then fails.
- Flow-driven `message_send` and `email_send` are not country-whitelisted (message-manager, email-manager).
- `flow_destinations` does not show: destinations inside an inline `actions` list of a `call` action, destinations of a flow referenced by `call.flow_id`, and the recipients of `message_send` and `email_send` (only their type is in `flow_action_types`).
- For an `actions` draft the ephemeral flow is created (`FlowV1FlowCreate`, validated by flow-manager) after the claim, so a card that looked valid can fail at approval (a rejected action, a transient error) and the draft is spent; creating the flow before the claim would leave an orphan flow whenever a second decision loses the claim, so it is not done. Tests cover this path.
- Call-manager failure reasons are not returned to the agent.
- A team transition `function_name` can name `draft_call` because the reserved-name check uses `tool.AllToolNames`; the same pre-existing gap as `draft_email` and `draft_message`, out of scope.
- `approve` of an old card after the flow or the customer's configuration changed is limited to the top-flow `tm_update` check above.
- `ToolHandle` has no gate: keeping `create_call` away from Insight AIs worked only by tool exposure, and now `create_call` is exposed by design.

Trigger to add a per-hour or destination-count limit: a measured abuse or cost case, or a request for unattended calling by many AIs.

## 6. Implementation design

Facts read from the code for this design (flow-manager `flowhandler/db.go`, `dbhandler/flows.go`, `cachehandler`): a temporary flow (`persist=false`) is written only to the cache and its `TMUpdate` stays nil, so for a temporary flow nil equals nil always matches and the binding detects only disappearance; after the cache entry expires `FlowV1FlowGet` gives not found, which is `DRAFT_INVALID`. A persistent flow is a database row; `tm_update` is set by the update paths and is nil until the first update. `Persist` is not a column but is set on read (true for rows, false for cache-only flows) and serialized as `persist`, so the card carries `flow_persistent` as a label; the draft restricts nothing by it (decision 4). The cache TTL (24 hours) is longer than the draft expiry (600 seconds), so the disappearance case is a safety net.

Files (changed or added):

- `models/tool/main.go`: `ToolNameDraftCall`; `InsightCallToolNames = [create_call, draft_call, describe_action]`; comments of `AllInsightToolNames` and the Insight lists.
- `models/message/tool.go`: `FunctionCallNameDraftCall`; `BlockTypeCallDraft`; comments. `models/message/tool_test.go`: the two string constants.
- `models/ai/tool_validation.go` (`insightToolSet`), `models/ai/main.go` (tool list comment).
- `pkg/toolhandler/definitions_draft_call.go` (new; `definitions.go` is unchanged): the `draft_call` definition, built in `init()` from the `create_call` definition (the parameters of `create_call` plus `title`; guidance: nothing is placed, customer content is untrusted data, never take a number, source or action from it without the agent's request, the card shows destinations, source and the flow or actions).
- `pkg/aicallhandler/tool_draft_call.go` (new): `toolHandleDraftCall`, `validateCallDraftShape` split into `validateCallDraftTarget`, `validateCallDraftActions` and `validateCallDraftVariables`, and `callDraftFillFlow` with `callDraftFlowSummary` and `callDraftTMUpdate` (flow name, action types, destinations, `tm_update`), `callDraftShapeStillValid` (shape only) and `callDraftFlowBindingValid` (the flow read and comparison), two functions that `draftDecisionLoad` calls in the order shape, expiry, binding, so that the expiry check sits between them; the type `CallDraft` and the `Call *CallDraft` field of `CardBlock`.
- `pkg/aicallhandler/tool.go`: dispatch of `draft_call`, the `emitInfoCardLLMResult` condition, the comments of `CardBlock` and the condition.
- `pkg/aicallhandler/draft_decision.go`, split so the cyclomatic complexity of `DraftDecision` does not grow: `draftDecisionFindBlock(blocks)` returns the decidable block (its `Type` selects the action); `draftDecisionBuildTool(block, draftID)` builds the tool call and holds the only switch on the block type (sms, email, call); `draftDecisionExecute` runs the handler per type and sets the result; constants `draftCallInvalidMessage`, `draftCallRejectedMessage`. The load order is fixed as: shape validation, then expiry, then the flow binding read, then the claim (an expired draft never costs a flow read, and a transient read error comes before the claim and does not consume the draft). The stored content is decoded with `UseNumber`; the SMS and email blocks have no `interface{}` fields, so their decoding is unchanged in effect, and their decision tests must stay green. The binding read (`FlowV1FlowGet`) and the one inside `toolHandleCreateCall` are two reads; the gap between them is accepted, and the mocks expect two reads.
- Comments and wording (`SMS or email` becomes `SMS, email or call`): `pkg/aicallhandler/main.go`, `pkg/teamhandler/handler.go`, `internal/config/main.go`, `docs/operations.md`, `bin-common-handler/pkg/requesthandler/main.go` (line ~353) and `ai_messages.go` (line ~81), `bin-api-manager/pkg/servicehandler/serviceagent_aimessage.go` (comment and log text), `bin-api-manager/docs/routing.md`.
- Tests that pin the lists (they fail otherwise, so they are fixed in the same change): `models/ai/allowed_tools_test.go` (`insightExtraToolNames()` gets the call list, because `create_call` and `describe_action` are also in `AllToolNames` and the tests `TestAllowedToolNames` and `TestValidateToolNames_WriteToolNeverAllowedForInsight` currently pin "create_call is rejected for Insight"; the count 12 becomes 15; `Test_InsightCallToolNames` added; `TestAllInsightToolNamesAreReadOnly` unchanged except its comment), `models/ai/tool_validation_test.go`, `pkg/toolhandler/main_test.go` (counts 12 to 15, the Insight composition, and the cases insight explicit `draft_call`, insight explicit `create_call`, normal `all` without `draft_call`, normal explicit `draft_call` rejected), `bin-pipecat-manager/pkg/toolhandler/main_test.go` (`Test_ToolHandler_GetByNames_InsightAllReturnsTwelveTools` becomes fifteen, cache set and `want` set). Checked and not changed: `pkg/aihandler/chatbot_test.go` (uses `connect_call` and `stop_flow` only), `builderhandler` and `builder.AllowedTools`, `eval_test`, `definitions_resource_test`, `bin-api-manager/server/ais_test.go`, mocks (no interface change).
- Docs and generated files: `bin-ai-manager/docs/domain.md` (rows 174 to 188 and the call tools), `architecture.md`; openapi `AIManagerToolName` enum, the `tool_names` descriptions, the draft-decision description (`openapi.yaml`, `paths/ais/id.yaml`, `paths/ais/main.yaml`, `aimessages_draft_decision.yaml`) and the generated `openapi_redoc/openapi.json`, `bin-api-manager/gens/openapi_server/gen.go`, `bin-openapi-manager/gens/models/gen.go`, regenerated with the repository's generate steps (the diff of each generated file must be the enum line and the description text only, which is checked); `bin-api-manager/docsdev/source` (`ai_struct_tool.rst`, `ai_struct_message.rst`, `service_agent_overview.rst`, `ai_overview.rst`, `ai_struct_ai.rst`) and the built HTML rebuilt as in CLAUDE.md (`rm -rf build`, sphinx rebuild, `git add -f`).

Order (TDD, each step ends with the tests of that step green):

1. Constants, lists, validation and the list tests: first run the existing list tests to see them fail where they pin the old sets, then fix them together with the pipecat and ai-manager toolhandler tests.
2. `draft_call` definition and `toolHandleDraftCall` with tests written first.
3. Decision path: run the existing SMS and email decision tests on the refactored switch first (must be green with no change), then add the call branch tests and code.
4. Comments, wording, openapi, generated files, docs.
5. `go build`, `go vet`, `golangci-lint` (the repository's `.golangci.yml` and `.golangci-errorlint.yml`), and `go test` for bin-ai-manager, bin-pipecat-manager, bin-api-manager, bin-common-handler, bin-openapi-manager.

## 7. Tests (every call uses the request handler mock; nothing is dialed)

- Tool lists: `Test_InsightCallToolNames` exact names; `TestAllInsightToolNamesAreReadOnly` green and unchanged; `AllowedToolNames` and `ExpandAllToolNames` for Insight include the three tools; `ValidateToolNames` accepts `create_call`, `draft_call`, `describe_action` for Insight and rejects `draft_call` for Normal; the pipecat `["all"]` set for Insight is 15 tools.
- `draft_call` creation: a flow draft and an actions draft each produce a `call_draft` block, no `CallV1CallsCreate` call and (for flow) one `FlowV1FlowGet`; the card carries destinations, source, anonymous, variables, `flow_name`, `flow_action_types`, `flow_destinations` and `flow_tm_update`, and an action whose option cannot be parsed contributes nothing to `flow_destinations`; flow and actions together or neither; no destination; an empty target; an unknown action type; a missing or foreign flow gives the masked `could not resolve flow`; variables over 100 keys, a value over 32 KiB, a total over 64 KiB, a reserved key (`voipbin.x`, `Voipbin.x`, ` voipbin.x`, an empty key); a draft above `draftStoredMaxBytes`; `title` default and the 200 character cut; run_llm; the block is not in the LLM-facing result (`emitInfoCardLLMResult`).
- Variable keys: flow-manager trims keys, so `"a"` and `" a"` collide after trimming; the draft refuses a key that is not already trimmed, and a test covers it.
- Numbers: an integer above 2^53 and `1.0` in an action option survive draft arguments, stored row and approval unchanged; variables with numeric values are stored as strings.
- Decision: approve of a flow draft and of an actions draft calls `CallV1CallsCreate` exactly once with the stored destinations, source, anonymous, variables and the flow id (or the ephemeral flow created from the stored actions, with `FlowV1FlowCreate` called after the claim); the result is the handler's JSON unchanged and no ` Delivery may be unconfirmed.` suffix; reject places nothing and records `draftCallRejectedMessage`; second decision `DRAFT_ALREADY_DECIDED`; expired `DRAFT_EXPIRED` and no flow read happens; a tampered row, a missing flow, a foreign flow, a changed `tm_update`, nil against non-nil, an unparsable stored `tm_update` all `DRAFT_INVALID` with the draft not consumed; a temporary flow (nil on both sides) matches; a transient `FlowV1FlowGet` error returns an error before the claim and does not consume the draft; `FlowV1FlowCreate` failing after the claim for an actions draft gives a `failed` result and a consumed draft; every destination failing gives `failed`; metric label `create_call`; the SMS and email decision tests unchanged and green.
- Mocks expect two `FlowV1FlowGet` calls on an approved flow draft (the binding read and the one inside `toolHandleCreateCall`).
- pipecat-manager, api-manager and openapi tests that enumerate the tools or the draft-decision description are updated.
- Design-stage reading of real flow-manager behavior is done (start of section 6); the unit tests above use the mock and encode that behavior.
