# Insight Assistant: draft_message, send_message and draft-decision

Date: 2026-11-10
Ticket: VOIP-1704
Services: bin-ai-manager, bin-pipecat-manager, bin-api-manager, bin-openapi-manager, bin-common-handler (request client and mock)

## 1. Goal and non-goals

An Insight Assistant can propose an SMS to a person and, depending on the customer's own prompt, either wait for the agent to confirm it or send it directly.

- Two tools: `draft_message` (shows an SMS draft card, sends nothing) and `send_message` (the existing tool, reused as is).
- When the agent presses OK on a draft card, the platform runs the send_message handler in code, without an LLM turn. The agent can also reject the draft. A draft that is not decided within 10 minutes can no longer be decided.
- The policy (confirm first, send directly, never send) is decided by the customer's prompt and `tool_names`. The platform only provides the mechanics.

Non-goals (not implemented): a mode field, a schema migration, a recipient restriction, a call limit, a refusal during a listen turn, an AI-type gate inside `ToolHandle`, any change to the MCP path, email or call tools, the frontend.

## 2. Tool set

| Name | Where | Notes |
|---|---|---|
| `draft_message` | new, Insight only | Records a card block in the session message stream. No external call. |
| `send_message` | existing definition and handler | Allowed for Insight only when named in `tool_names`. |

`tool.InsightOptInToolNames = {send_message, draft_message}` is a separate list. **Revised 2026-11-11 (VOIP-1708, section 8): the list is renamed `tool.InsightMessageToolNames` and the `["all"]` selector now includes it. The statements below that say `["all"]` excludes these tools are superseded.**

- `AllowedToolNames(Insight)` is `AllInsightToolNames` plus `InsightOptInToolNames`. This is the set `ValidateToolNames` accepts for an explicit name.
- The `["all"]` selector expands to `AllInsightToolNames` only. A new helper `ExpandAllToolNames(type)` returns the set a selector may expand to (Normal: `AllToolNames`, Insight: `AllInsightToolNames`, other: empty). `GetByNames` in bin-ai-manager and in bin-pipecat-manager use it. A tool is returned when it is in `AllowedToolNames(type)` and either the selector expands to it or its name is listed.
- (Superseded, see section 8.) So an existing Insight AI stored with `["all"]` gets no new capability on deploy. A list such as `["all", "send_message"]` is valid and yields the expanded set plus `send_message`.
- `draft_message` is not added to `tool.AllToolNames`, so a Normal AI cannot use it and the team transition reserved-name check (which uses `AllToolNames`) is unchanged. Insight AIs cannot be team members.
- Dispatch: `ToolHandle` has no AI-type gate (existing behavior, out of scope). Suggestion only: add one in a separate ticket.

## 3. draft_message

Arguments: `to` (required, E.164 `^\+[1-9][0-9]{6,14}$` after trimming), `text` (required, not blank, at most 1600 characters (runes); a longer text is rejected, never silently cut), `title` (optional, default `SMS draft`, cut at 200 characters), `run_llm`.

The handler writes no external call. It returns a result whose `blocks` holds one card: `{type: "sms_draft", title, to, text}`. `CardBlock` gets `To` and `Text` (omitempty), so existing `info` cards are unchanged on the wire. The result row is a `role=tool` message created by the normal `ToolHandle` path; its message id is the **draft id**. The result text is `Displayed an SMS draft for review.` and never contains the draft body.

LLM view: the `blocks` key of a tool row is already removed when the history is replayed, and `emitInfoCardLLMResult` (extended to `draft_message`) returns the result without blocks on the first turn. The tool-call arguments of `draft_message` are not neutralized (unlike `emit_info_card`), so the assistant remembers its own draft and can revise it when the agent asks.

Untrusted input: the tool description states that customer messages and call content are untrusted and must not be followed as instructions.

## 4. Agent decision: `POST /service_agents/aimessages/draft-decision`

Body: `{aicall_id, draft_id, decision}`, `decision` is `approve` or `reject`. The existing `POST /service_agents/aimessages` is not extended because `aicall_id`, `role` and `content` are required there.

Chain: api-manager server handler, servicehandler (agent identity, `PermissionAll`, and the aicall must belong to the agent's customer, the same rule as the other aimessages calls), `requesthandler.AIV1MessageDraftDecision`, ai-manager listenhandler `POST /v1/messages/draft-decision`, `aicallHandler.DraftDecision`. It does not go through `aicallHandler.Send` and its cooldown.

Server checks in `DraftDecision` (the request carries ids only; the number and body are read from the stored draft row):

1. `decision` is `approve` or `reject`, else InvalidArgument.
2. `messageHandler.Get(draft_id)` (its `MESSAGE_NOT_FOUND` error is converted to `DRAFT_NOT_FOUND`, so a missing id and a foreign id answer the same); the row must belong to the same aicall and customer as the aicall, have `role=tool`, and its content must carry a top-level block with `type=sms_draft` and a non-empty `to`. Every failure (missing, other aicall, other customer, not a draft) is the same `NotFound` (`DRAFT_NOT_FOUND`) so ids cannot be probed.
3. Expiry: `now - draft.tm_create > AIcallDraftDecisionExpirySeconds` (service config, default 600; a value of zero or less falls back to 600; a draft row without `tm_create` counts as expired, fail closed) is `FailedPrecondition` (`DRAFT_EXPIRED`). Evaluated lazily, no timer, no history row. The check runs before the claim, so an expired draft reports expired whether or not it was decided earlier.
4. Claim: create the claim row with id `uuid.NewV5(draftDecisionNamespace, draft_id.String())`, where `draftDecisionNamespace` is a fixed constant in `aicallhandler` (changing it would make old claims invisible, so it is never changed). A primary key conflict (MySQL 1062 through `errors.As`, or the sqlite unique error in tests) is `AlreadyExists` (`DRAFT_ALREADY_DECIDED`); any other error is Internal. Approve and reject use the same claim id, so the first call wins and the other is refused.
5. Approve: re-validate `to` with the same E.164 rule, build the arguments with `json.Marshal` (never string concatenation) from the draft row (`source` omitted so the account's default sender is used, one `tel` destination, `text`, no medias) and run `toolHandleMessageSend`; when it reports `failed`, the stored message gets the suffix ` Delivery may be unconfirmed.`. Reject: do not send.
6. Record the result row (below) and return it.

Rows written for one decision (no row is updated after creation):

- Claim row (written first, before any send): `role=notification`, content `{"type":"draft_decision","draft_id":...,"decision":...}`, id from step 4. The replay already skips `role=notification` rows, so the claim never reaches the LLM and a slow send cannot leave a dangling tool call in the replayed history.
- Call row (written after the send, immediately before the result row so the pair is adjacent): `role=assistant`, tool call `send_message` with the arguments above and the synthetic tool call id `dd_` + the 32 hex characters of the draft id (35 characters, non-empty). Origin none, `ActiveAIID` from the aicall, no pipecat call id.
- Result row: `role=tool`, same tool call id. Approve: the `send_message` result (`result` is `success` or `failed`, `resource_type=message`, `resource_id` is the message id on success). Reject: `result=rejected`, message `The agent rejected this draft. The message was not sent.`.
- `promAIcallToolExecuteTotal{tool="send_message"}` is incremented on approve.

A send failure is a recorded `failed` result, returned with 200. The same draft cannot be retried; the assistant makes a new draft. If the process stops after the claim and before the pair is written, only the claim row remains; nothing resends automatically and replay is unaffected (the claim is not replayed). If the send succeeded but writing the pair fails, the caller gets an Internal error, the failure is logged with the draft id, a repeated decision answers `DRAFT_ALREADY_DECIDED`, and the card stays in the processing state. The pair is two consecutive inserts; a row of another turn could in theory land between them, which is the same exposure the existing tool path has between its call and result rows.

Card state is computed by the frontend from history and the draft creation time: no decision rows is pending; a claim row (`role=notification`, `type=draft_decision`) without a result row is processing (a send in flight, or a result that was never recorded after a crash; a repeated decision is refused with `DRAFT_ALREADY_DECIDED`, so this state does not return to pending); a result row with tool call id `dd_<draft id hex>` is sent (`success`), failed (`failed`) or cancelled (`rejected`); no decision rows and more than the expiry elapsed is expired. Each decision row (claim, call, result) publishes a message-created event.

A decision is not tied to the aicall state: a draft from an earlier assistant session, from a listen turn, or of an aicall that has since ended can still be decided within the expiry. The api-manager layer behaves like the other aimessages calls (tenant check through the aicall lookup); the identical-NotFound rule above applies to the draft checks in ai-manager only.

## 5. Accepted risks (stated plainly)

- Reusing `send_message` opens MMS attachments and an arbitrary source number when the assistant calls it directly. The decision path is narrower (default sender, one destination, no medias).
- Prompt injection through customer input (messages, call content) can make an assistant send to an arbitrary number when `send_message` is enabled. The cost is charged to the customer account. Mitigation is the customer's prompt and enabling only `draft_message`. (Revised by section 8: `["all"]` now enables `send_message`, so a customer who does not want direct sending must name `draft_message` explicitly instead of `["all"]`.)
- The call row is a `send_message` tool call in the replayed history, so an assistant that has only `draft_message` enabled can see a `send_message` call after an agent approved a draft, and a call to a tool it was not given would still run because `ToolHandle` has no AI-type gate. The arguments are kept on purpose so the assistant knows what was sent.
- A `failed` result can hide an unknown delivery state (for example an RPC timeout after the message was accepted), and the follow-up is a new draft, so a duplicate SMS is possible in that case. The result text says delivery may be unconfirmed.
- Drafts the assistant replaced with a revised draft stay decidable until they expire; marking a draft as superseded is a frontend concern.
- The decision rows are ordered by creation time among the session rows and can land inside a turn that is in progress; the pair is written adjacent to each other, and unpaired rows are dropped before the LLM sees them.
- Any agent of the same customer can decide any draft of that customer (same permission as the existing aimessages calls).
- Tools of an MCP server in `mcp_server_ids` are not filtered for write capability; unchanged.

## 6. API and documentation changes

- OpenAPI: new path schema `service_agents/aimessages/draft-decision`; `AIManagerToolName` gets `draft_message`; the `tool_names` description lists the opt-in tools and says `["all"]` does not include them. Generated code is regenerated in bin-openapi-manager and bin-api-manager.
- RST: `ai_struct_ai` (`tool_names` text), `ai_struct_tool` (the `draft_message` section, the opt-in note, the `sms_draft` block and card-state rules), `ai_overview` (tool list note), `service_agent_overview` (new endpoint); the built HTML are updated, including the statement that `["all"]` is valid for Insight AIs.
- Service docs: bin-ai-manager `architecture.md` (route), `operations.md` (new setting), `domain.md` (tools); bin-api-manager `routing.md`.
- The `AllInsightToolNames` comment is rewritten: the list is still side-effect free; SMS sending exists only as the opt-in list; email, calls and CRM writes remain unavailable.

## 7. Implementation design

bin-ai-manager
- `models/tool/main.go`: `ToolNameDraftMessage`, `InsightOptInToolNames`, comment update.
- `models/ai/tool_validation.go`: `AllowedToolNames` union for Insight; `ExpandAllToolNames`.
- `models/message/tool.go`: `FunctionCallNameDraftMessage`; `DraftDecision` type with `approve` and `reject`; draft-related constants.
- `pkg/toolhandler/main.go` and `definitions.go`: `GetByNames` via `ExpandAllToolNames`; the `draft_message` definition (RunLLM true).
- `pkg/aicallhandler/tool_draft_message.go`: handler; `tool.go`: `mapFunctions` entry, `emitInfoCardLLMResult` branch, `CardBlock` fields.
- `pkg/aicallhandler/draft_decision.go` (claim marker, send, pair): `DraftDecision` (added to the `AIcallHandler` interface and mock).
- `pkg/dbhandler/main.go`: exported duplicate-key helper wrapping the existing `errors.As` check.
- `internal/config`: `AIcallDraftDecisionExpirySeconds` (`aicall_draft_decision_expiry_seconds`, default 600) and a test setter.
- `pkg/listenhandler`: route, handler, request DTO (wire type used only inside listenhandler); the response is the domain message marshaled directly.

bin-pipecat-manager: `pkg/toolhandler/main.go` `GetByNames` via `ExpandAllToolNames`.

bin-common-handler: `AIV1MessageDraftDecision` in `requesthandler/ai_messages.go`, interface and mock.

bin-api-manager: servicehandler `ServiceAgentAImessageDraftDecision`, server handler `PostServiceAgentsAimessagesDraftDecision`, routing doc.

Signatures and flow.

- `message.DraftDecision` (string) with `DraftDecisionApprove`, `DraftDecisionReject`.
- `aicallHandler.DraftDecision(ctx, aicallID, draftID uuid.UUID, decision message.DraftDecision) (*message.Message, error)` returns the result row.
- `request.V1DataMessagesDraftDecisionPost{AIcallID, DraftID uuid.UUID; Decision message.DraftDecision}`, used only in listenhandler. Route regex `^/v1/messages/draft-decision$`, method POST, placed next to the other `/v1/messages` routes. Typed errors go through the existing `errorResponse`.
- `requesthandler.AIV1MessageDraftDecision(ctx, aicallID, draftID uuid.UUID, decision cbmessage.DraftDecision) (*cbmessage.Message, error)`, 30 second timeout like `AIV1MessageSend`.
- `servicehandler.ServiceAgentAImessageDraftDecision(ctx, a *auth.AuthIdentity, aicallID, draftID uuid.UUID, decision ammessage.DraftDecision) (*ammessage.WebhookMessage, error)`: `IsAgent`, `PermissionAll`, `ServiceAgentAIcallGet`, RPC, `ConvertWebhookMessage`. The deciding agent is written to the log only.
- `toolHandleDraftMessage(ctx, c, tool) *messageContent` is registered in `mapFunctions`; `ToolHandle` treats `draft_message` like `emit_info_card` when building the LLM-facing result.

`DraftDecision` outline: validate decision; `c := h.Get(aicallID)`; `d := messageHandler.Get(draftID)` and convert not-found; check `d.AIcallID == c.ID`, `d.CustomerID == c.CustomerID`, `d.Role == tool`, parse content into `messageContent` and find the `sms_draft` block with a non-empty `to`; expiry check; claim `messageHandler.Create(claimID, ..., RoleNotification, claimJSON, nil, "")` and map a duplicate key to `DRAFT_ALREADY_DECIDED`; approve builds `message.ToolCall{ID: "dd_"+hex, Type: function, Function{send_message, json}}` and runs `toolHandleMessageSend`, increments `promAIcallToolExecuteTotal`; reject builds the rejected result; write the call row (`Create` with the tool call, `WithActiveAIID`) and the result row (`toolCreateResultMessage`); return the result row.

Additional tests: the claim row is absent from `getPipecatcallMessages`, ignored by the unanswered-message timeout query and skipped by the pipeline-error notice scan; the two decision tool rows are adjacent in replay.

Implementation details fixed by the design review.

- Validation order: every check (decision value, draft lookup and ownership, block kind and non-empty `to`, E.164 re-validation, expiry) runs before the claim row is written, so a request that is refused never consumes the draft. Only the claim, send and pair-writing steps come after.
- Time source: `h.utilHandler.TimeNow()` (mockable), not `time.Since`, so the 599 and 601 second boundaries are deterministic.
- Duplicate detection: a new exported `dbhandler.IsDuplicateKey` wraps the existing unexported `errors.As` check for MySQL error 1062 with the sqlite text fallback. The existing text-based `IsErrDuplicate` is left alone.
- RPC timeout: `sendRequestAI` takes milliseconds, so the new request uses 30000, like `AIV1MessageSend`.
- Unknown AI type: `ExpandAllToolNames` returns an empty set without a second warning or metric increment (`AllowedToolNames` already reports it).
- Request body: `decision` is a named component schema `AIManagerMessageDraftDecision` (enum `approve`, `reject`) so the generated type name stays short; the server handler casts with `uuid.UUID(req.AicallId)` and `ammessage.DraftDecision(req.Decision)`.
- Listen handler: the metric label branch for `/v1/messages/draft-decision` is added; routing tests cover that the UUID route of `/v1/messages/<id>` does not capture it.
- Files not listed above: `bin-openapi-manager/openapi/openapi.yaml`, `openapi/paths/service_agents/aimessages_draft_decision.yaml`, both generated files (`bin-openapi-manager/gens/models/gen.go`, `bin-api-manager/gens/openapi_server/gen.go`), the `ServiceHandler` interface and mock in bin-api-manager, the `AIcallHandler` and request handler mocks, the four RST files and the rebuilt `docsdev/build`. No `go.mod` or vendor change (all consumers already import the ai-manager model packages).
- The claim row (`role=notification`, `type=draft_decision`) is visible in the aimessages list and in the message-created event; the RST message documentation describes it with the card state rules.
- Two rows created in the same microsecond are ordered by the store, so the adjacent pair can in theory be listed in reverse; the existing tool path has the same exposure.

Existing tests that change: in `models/ai/allowed_tools_test.go`, the Insight case of `TestAllowedToolNames` (disallowed set is `AllToolNames` minus the opt-in list) and `TestValidateToolNames_WriteToolNeverAllowedForInsight` (skips the opt-in list; a new guard asserts the opt-in list contains exactly the two known names); the pipecat `toolhandler` test fixtures get the two tool definitions.

Test harness: the concurrent approve and reject test, the claim id test and the duplicate-versus-other-error test use the real `messagehandler` on the sqlite `dbhandler` with only the request handler mocked (a mocked message handler cannot reproduce a primary key conflict). The assertions count send calls (exactly one or zero) and decision row sets. Further cases: a `role=tool` row with an `info` block, an `sms_draft` block with empty `to`, and a failed E.164 re-validation are refused before the claim.

Tests (all sends use a request-handler mock; nothing is sent): selector exclusion and explicit allow in both services; mixed `["all","send_message"]`; `draft_message` rejected for Normal; validation and truncation rules of the draft arguments; `draft_message` writes only a block and makes no send call; blocks absent from the LLM result and from replay while the arguments stay; approve and reject paths with paired rows; second decision refused in both orders; concurrent approve and reject (one wins, one send); other aicall, other customer, missing id and non-draft row refused identically; expiry at 599 and 601 seconds and expired-before-decided order; send failure recorded as `failed`; duplicate-key versus other database errors; the deterministic claim id; the argument JSON shape accepted by the send handler; api-manager permission and tenant isolation.

Additions from the second design review.

- Generated files also changed: `bin-api-manager/gens/openapi_redoc/openapi.json`. `api.html` and the extraction files under `docs/.docs-gen` are not regenerated: the checked-in copies were already out of date with their sources (a regeneration rewrites unrelated parts), so they are left to their own refresh.
- `internal/config` change is complete only with the flag, the environment variable mapping, the `Get()` read, the setter and the setters coverage test entry. The fallback for a value of zero or less is applied by the reader (`DraftDecision`).
- `dbhandler.IsDuplicateKey` godoc states that it matches the database error number (and the sqlite text in tests), unlike the text based `IsErrDuplicate`.
- The sqlite based tests in `aicallhandler` load the repository database scripts through their own helper; the expiry boundary tests use a mocked message handler and a mocked `TimeNow` instead.
- Rollout order: ai-manager is deployed before bin-pipecat-manager so the tool definition exists; the reverse order only makes pipecat skip the unknown name. (Revised by section 8: the order does not matter for `["all"]`.)
- Team transition reserved names (`tool.AllToolNames`) are unchanged: `draft_message` is an Insight-only tool and Insight AIs cannot be team members, like the other Insight tools.

## 8. Revision 1 (VOIP-1708, 2026-11-11): `["all"]` includes the SMS tools

Decision (2026-11-11): an Insight AI stored with `tool_names=["all"]` gets every Insight tool: the 8 read-only and card tools plus `draft_message` and `send_message`. "all" means all tools; the opt-in concept no longer applies to the selector. Naming a tool explicitly stays valid. Reason: with opt-in, an Insight assistant created through the admin console ("Enable All Tools" sends `["all"]`) could not send an SMS and answered that it had no such capability, which contradicted what "all" promises. The earlier text of sections 1 to 7 is kept above for history. Every statement in sections 2 to 7 that says `["all"]` excludes the SMS tools, calls them opt-in, expects a selector exclusion test, or fixes a rollout order is superseded by this section 8.

Design (supersedes the "all expands to `AllInsightToolNames` only" bullet of section 2):
- `tool.AllInsightToolNames` stays the 8 tools without outside side effects. `TestAllInsightToolNamesAreReadOnly` keeps guarding exactly that list; the SMS tools are not added to it and so never pass through that guard.
- `tool.InsightOptInToolNames` is renamed `tool.InsightMessageToolNames` (`send_message`, `draft_message`): the Insight tools that send or propose an SMS. It is a separate, pinned list (`Test_InsightMessageToolNames`).
- `ai.ExpandAllToolNames(TypeInsight)` returns the union of the two lists (10 tools). `AllowedToolNames(TypeInsight)` is unchanged in content (also the union). Normal AI is unchanged: `AllToolNames` for `["all"]`, `draft_message` rejected by name and by selector.
- The function body of `ExpandAllToolNames` changes (Insight branch). The ai-manager and pipecat-manager `GetByNames` already derive the selector set from it, so their logic does not change; only comments and tests change. The pipecat Python runner has no hard-coded tool list (it receives the tool definitions from Go).
- Deploy impact: an Insight AI already stored with `["all"]` receives `draft_message` and `send_message` when the service is deployed. `send_message` sends an SMS or MMS at once without agent confirmation and the cost is charged to the customer account. A customer who wants approval before sending lists the tools by name and leaves out `send_message`.
- Listen turns: a listen turn (no agent present) runs with the AI's `tool_names`, so an Insight AI with `["all"]` can now call `send_message` there too, driven by untrusted call or message content. The prompt wording is an instruction, not a block. This widens the prompt-injection risk of section 5 to every AI stored with `["all"]`; it is accepted with the decision and not mitigated in code here. `ToolHandle` has no AI-type gate, which is unchanged.
- Rollout and rollback: no schema change or data migration. The order of the two services does not matter: if pipecat-manager runs the new code first it simply returns the two tools it finds in its cache; if it runs old code it keeps the old 8 tools. Rollback is a code rollback.
- Tests: `Test_InsightOptInToolNames` becomes `Test_InsightMessageToolNames` (pins the 2 names, both are in `ExpandAllToolNames(Insight)` and accepted by `ValidateToolNames`); `Test_ExpandAllToolNames` expects 10 for Insight; a new invariant test requires `ExpandAllToolNames(Insight)` to equal `AllowedToolNames(Insight)`; both `GetByNames` tests expect `["all"]` to return exactly the 10 tools, explicit single names to work as before, `["all", "send_message"]` to return 10 without duplicates, and Normal to reject `draft_message`. `Test_InsightMessageToolNames` is the guard against the message list growing unnoticed, because `TestValidateToolNames_WriteToolNeverAllowedForInsight` skips every name in it.
- Docs synced in the same change: `bin-ai-manager/docs/domain.md` (draft_message row), the RST files `ai_struct_ai` (tool_names item and the `["all"]` row of the tool_names table), `ai_struct_tool` (the opt-in paragraph, the `draft_message` section, the table note), `ai_overview` (tool list note and the two `["all"]` cautions, which gain a line that `send_message` sends at once) and their built HTML, and the three `tool_names` descriptions in bin-openapi-manager with the regenerated code of bin-openapi-manager and bin-api-manager. The admin console "Enable All Tools" check sends `["all"]` and therefore enables both tools.
- Not changed: `InsightSystemPrompt`, `ToolHandle` (still no AI-type gate), call limits, recipient limits, frontend.
