# Insight Q&A tools (get_contact_interactions / get_conversation_content) design memo

- Date: 2026-07-15
- Ticket: VOIP-1234 (last remaining subtask)
- Service: bin-ai-manager (models/tool, pkg/toolhandler, pkg/aicallhandler)
- Status: **FINAL — all in-scope decisions resolved by 대표님 2026-07-15. Prompt-injection framing (§5/§6-3) explicitly deferred to a separate consolidated ticket, not in scope here. Ready for TDD implementation.**

## 1. Context (already shipped, not touched by this memo)

VOIP-1234's 11 prior subtasks are already merged to `main`: AIcall-Case link
(`aicall.ReferenceTypeContactCase`, `ReferenceID = Case.ID`), concurrency
guard, rate limit, `ai.Type` (Normal/Insight) fan-out, `in_reply_to_message_id`
cross-talk fix, team-membership rejection for Insight AIs, and the
`service_agents/aicalls` + `service_agents/aimessages` REST surface.

What remains: `tool.ToolNameGetContactInteractions` /
`tool.ToolNameGetConversationContent` are reserved as names in
`models/tool/main.go` (`AllInsightToolNames`) but have **no toolDefinitions
entry and no execution handler**. Today an Insight AI has literally zero
usable tools. This memo scopes the two missing tools.

## 2. Precedent this design copies

`get_resource` (`pkg/aicallhandler/tool_resource.go`, `docs/plans/
2026-06-11-add-get-resource-tool-design.md`) already solved the same
shape of problem: fetcher-map dispatch, single masked "not found" message
for every not-accessible path (`msgResourceNotFound` / `ErrResourceNotAccessible`
— IDOR-safe, not an existence oracle), a whole-message rune cap
(`maxResourceSummaryRunes = 4000`), and a `resourceListPageSize` probe-one-extra
pagination pattern. Both new tools reuse these primitives verbatim rather than
inventing new ones.

`get_correlation` -> `get_resource` also sets the precedent this design now
explicitly follows for Tool 2: a **discover-then-fetch chain** — one tool
lists candidate ids, a second tool takes an explicit id argument the LLM
picked from that list. See §5.

## 3. Case scoping — DECIDED: implicit, no id argument, for BOTH tools' target Case

Both tools run only inside an Insight AIcall
(`c.ReferenceType == aicall.ReferenceTypeContactCase`, `c.ReferenceID = Case.ID`).
Neither tool accepts a `case_id` or `contact_id` argument. The Case itself is
always resolved server-side from `c.ReferenceID` — a value the LLM never sees
and never supplies.

**Rationale (confirmed 2026-07-15):** the Insight AI's system prompt already
frames its job as "analyze the Case you were opened for." There is no
approved use case for an Insight AI to inspect a different Case, so accepting
a case/contact id as an LLM-supplied argument would (a) require re-deriving
an IDOR-safe ownership check that doesn't need to exist otherwise, (b) create
a route for prompt-injection or hallucination to redirect the tool at the
wrong Case, and (c) add test/review surface for a capability nobody asked
for. Removing the argument removes the bug class entirely rather than
defending against it. If a future ticket wants cross-Case lookup, that is a
new, explicitly-scoped tool — not a parameter bolted onto this one.

This decision is scoped to **which Case** the tools operate on. It does
**not** extend to which conversation/message Tool 2 reads — see §5, which
reverses the original draft's recommendation on exactly that point.

## 4. Tool 1: get_contact_interactions

**Purpose:** list past interactions (calls, conversation messages) tied to
the Case's peer, so the Insight AI can answer "has this customer contacted
us before" / "what's the interaction history" — and to give Tool 2 a set of
concrete candidate ids to choose from (see §5).

**Resolution path:**
1. `h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)` — fetch
   the Case. Not-found or cross-customer (defensive; tenant is already
   embedded in the RPC) both fold into `msgResourceNotFound`, mirroring
   `get_resource`.
2. If `case.ContactID != nil`: call `ContactV1InteractionList(ctx, customerID,
   size, token, "", "", *case.ContactID, uuid.Nil, since)` (filter by
   contact, the strongest signal — covers every address the contact owns,
   not just this Case's peer).
3. Else: fall back to `ContactV1InteractionList(ctx, customerID, size, token,
   string(case.PeerType), case.PeerTarget, uuid.Nil, uuid.Nil, since)` (no
   linked Contact yet — filter by the Case's own peer address).

**Arguments (LLM-facing):**
```
{
  "run_llm": bool (default true),
  "limit": int, optional, default 20, hard cap 50 — mirrors resourceListPageSize convention but smaller (this is a summary list, not a resource dump)
}
```
No `contact_id`/`case_id` argument (§3). No `since`/date-range argument in
v1 — scope is "recent interaction history for this Case," not an open-ended
CRM query tool; add pagination/date-range only if real usage demands it
(same "add per demand" philosophy as the `get_resource` design doc's Phase 2
list).

**Output:** one line per interaction — direction, peer address, reference
type, **reference id** (this is the value Tool 2 consumes, see §5 — it MUST
be surfaced, not just used internally), `tm_interaction` — reusing the flat
"N of M shown" truncation convention `get_resource` and `get_aicall_messages`
already use. Total output capped at `maxResourceSummaryRunes` (import/reuse
the constant from `tool_resource.go`, don't redefine it).

**Failure modes:** Case RPC transient failure -> honest tool failure
(`fillFailed`), never masked. Empty interaction list -> success with
"no interactions found" (not a failure — a genuinely new contact is a valid
answer, not an error).

## 5. Tool 2: get_conversation_content — DECIDED: explicit `reference_id` argument, 2-RPC resolution

### Reversed from the original draft

The original draft recommended Tool 2 stay argument-less like Tool 1
(walk every interaction, fetch every message, N+1 RPCs). 대표님 rejected this
on two grounds: (a) it is wasteful (an unbounded-shaped RPC fan-out gated
only by a page-size cap), and (b) more importantly, **which conversation
to read is a judgment call the LLM should make explicitly, not one the
server should silently resolve for it** (e.g. picking "the most recent
interaction" would silently pick the wrong thread if the customer asks
about an older conversation). This is the same discover-then-fetch shape
`get_correlation` -> `get_resource` already establishes elsewhere in this
codebase — Tool 2 should follow it, not diverge from it.

### Final design

**Arguments (LLM-facing):**
```go
{
    Name:   tool.ToolNameGetConversationContent,
    RunLLM: true,
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "run_llm": map[string]any{
                "type": "boolean", "default": true,
            },
            "reference_id": map[string]any{
                "type":        "string",
                "description": "The reference id of a conversation_message-type interaction, as returned by get_contact_interactions. Call get_contact_interactions first to discover candidate ids.",
            },
            "limit": map[string]any{
                "type": "integer", "default": 20,
            },
        },
        "required": []string{"reference_id"},
    },
},
```
`reference_id` is **required**. The LLM must call `get_contact_interactions`
first, see the `conversation_message`-type rows and their reference ids, and
pass the one relevant to the question being asked (most recent, or a
specific one the user/agent references). There is no "just give me
everything" mode — that keeps this tool aligned with the CRM Q&A use case
(answer a specific question about a specific thread) rather than becoming a
bulk export.

**Resolution path (fixed 2 RPCs, independent of message/thread count):**
1. `ConversationV1MessageGet(ctx, referenceID)` — resolve the message the
   LLM pointed at. **Ownership check here is load-bearing**: if the message
   is absent OR `msg.CustomerID != c.CustomerID`, mask to
   `msgResourceNotFound` via the exact same `ErrResourceNotAccessible` /
   single-masking-site pattern `tool_resource.go` uses for `get_resource` —
   `reference_id` is an LLM-suppliable value now, so it needs the same
   IDOR defense `get_resource`'s arbitrary `resource_id` already has. This
   is the one place in this design where that defense is actually required
   (§3's implicit-Case decision doesn't cover this argument).
2. `ConversationV1MessageList(ctx, pageToken, pageSize, filters{
   message.FieldConversationID: msg.ConversationID})` — one list call,
   filtered by the resolved `conversation_id`, capped at `limit` (default
   20, hard cap 50, same convention as Tool 1). Returns the surrounding
   thread, not just the single referenced message.

Total: **2 RPC calls, fixed**, regardless of how many messages are in the
conversation — this was the original inefficiency concern (대표님's "무작정
모든 conversation을 다 조회할 순 없어") and it's resolved by filtering server-side
on `conversation_id` in one list call rather than fetching messages one at a
time.

**Output:** chronological message list (direction, text, timestamp),
truncated to `maxResourceSummaryRunes`. Messages themselves are user content
— in principle this is attacker-influenceable text flowing into an LLM
context (a customer could type prompt-injection text into a chat message),
the same class of concern `tool_resource.go`'s config-block framing
(`configFrameOpen`/`configFrameClose`) addresses for session-config data.
**Explicitly OUT OF SCOPE for this subtask (decided 2026-07-15): no
framing/escaping is added here.** 대표님 wants prompt-injection defense handled
as a separate, consolidated pass across all tools that return
user-authored/external content (not just these two), rather than
one-off per new tool. Ship Tool 1 and Tool 2 WITHOUT this framing now; track
the consolidated defense as its own follow-up ticket/session. Do not block
this subtask's implementation or review loop on it, and do not silently add
ad-hoc framing "just in case" — that would fragment the eventual
consolidated design.

**Failure modes:** invalid/missing `reference_id` -> `fillFailed` (LLM
self-correction case, same as `toolHandleSearchKnowledge`'s unmarshal-error
handling). Not-found / cross-customer message -> masked `msgResourceNotFound`
(never a distinguishable error — anti-IDOR-oracle). Transient RPC failure on
either call -> honest `fillFailed`, never masked.

## 6. Decisions — ALL CONFIRMED 2026-07-15

1. **Case scoping**: implicit, no `case_id`/`contact_id` argument on either
   tool. ✅ Confirmed, unchanged from draft.
2. **get_conversation_content target resolution**: **explicit `reference_id`
   argument** (reversed from the draft's Option B). The LLM must discover the
   id via `get_contact_interactions` first, then request the specific
   conversation it needs. Resolution is 2 fixed RPC calls
   (`ConversationV1MessageGet` -> `ConversationV1MessageList` filtered by
   `conversation_id`), not N+1 per-message fetches and not a silent
   "most-recent" auto-pick. ✅ Confirmed.
3. **Prompt-injection framing on quoted message content**: **DEFERRED, not
   in scope for this subtask.** 대표님 decided (2026-07-15) to handle this as a
   separate, consolidated effort across all tools that surface
   user-authored/external content, not as a one-off addition to these two
   new tools. Tool 1 and Tool 2 ship without this framing. Track separately.

## 7. Scope confirmation

Single service (`bin-ai-manager`). No new RPC (both tools compose existing
`ContactV1CaseGet`, `ContactV1InteractionList`, `ConversationV1MessageGet`,
`ConversationV1MessageList`). No OpenAPI/REST change (tools are internal LLM
function-calling surface only, exposed via existing `GET /v1/tools`). No
DB/migration change. Matches this skill's "single-file/single-service, not
the full cross-service fan-out" scoping guidance
(`voipbin-cross-service-phased-feature-implementation`).

Estimated shape: `models/tool/main.go` (remove TODO comments once real),
`pkg/toolhandler/definitions.go` (2 new Tool entries, Tool 2's schema has a
required `reference_id` field per §5), `pkg/aicallhandler/tool.go` (2 new
dispatch map entries), `pkg/aicallhandler/tool_insight.go` (new file, 2
handler functions + shared Case-fetch helper for Tool 1 + shared
IDOR-masking helper for Tool 2, mirroring `resolveResource`'s shape), plus
test files mirroring `tool_resource.go`'s test conventions (ownership
masking — now needed for Tool 2's `reference_id` path — rune-cap truncation,
empty-result-is-success, and a fixed-RPC-count assertion for Tool 2 proving
no N+1 fan-out regression). No prompt-injection framing/escaping code in this
subtask (see §5/§6-3 — explicitly deferred).
