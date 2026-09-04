# Case Insight Assistant: `emit_info_card` tool + chat message role allowlist

Date: 2026-09-04
Status: Issue-analysis stage complete (5 rounds, 2 consecutive approvals at
rounds 4+5). This document is the design-stage artifact, entering its own
design review loop.
Related: VOIP-1455 (this ticket), VOIP-1454 (AI chat Markdown rendering,
merged, PR #447), VOIP-1458 (pre-existing role-filtering defect, Highest
priority, discovered during this analysis and fixed as part of this ticket's
frontend phase)

## 0. Scope

Phase 2a of the "rich interactive AI chat content" roadmap (VOIP-1455's
parent scope: cards, buttons, polls, charts). This document covers **cards
only**. Buttons, polls, and charts are explicitly out of scope — see §8.

Two repositories, two PRs (structural necessity, not a scope split):

- **Backend** (`~/gitvoipbin/monorepo`, this worktree): new Insight-only tool
  `emit_info_card`, message storage/LLM-feedback separation, RST docs.
- **Frontend** (`~/gitvoipbin/monorepo-javascript`, separate worktree/PR):
  `CardBlock` rendering in `square-talk` and `square-admin`'s Insight
  Assistant panels, plus the role-allowlist fix (VOIP-1458) at the same
  render site.

`square-main` (the public marketing chat widget) has no Insight Assistant
integration, so `emit_info_card`/card rendering is out of scope for it in
this ticket. **However** (found in design review round 2): `square-main`
appears to share VOIP-1458's underlying defect class — see the note at the
end of §1.5 and VOIP-1459 (filed separately, since it needs its own
investigation and does not share a code path with this ticket's frontend
fix).

---

## 1. Issue analysis (condensed from the 5-round issue-analysis review loop)

This section summarizes conclusions that were each independently verified by
2+ rounds of adversarial code review before this document was written. Full
round-by-round detail lives in this session's transcript; only the final,
verified conclusions are restated here.

### 1.1 Architecture: `bin-pipecat-manager` needs redeploy, not code changes

`bin-ai-manager`'s `pkg/aicallhandler/tool.go`'s `ToolHandle(ctx, id, toolID,
toolType, function message.FunctionCall)` receives the LLM's tool-call
arguments directly via RPC from `bin-pipecat-manager`, independent of the
final assistant-text generation path. Card data is therefore fully available
at `ToolHandle` time.

Two things gate whether the LLM ever calls the new tool at all:

- **Tool definitions** (name/description/parameters/`run_llm`) are fetched by
  `bin-pipecat-manager` at **runtime** via `AIV1ToolList` RPC
  (`cmd/pipecat-manager/main.go:129-142`), refreshed on a 5-minute ticker.
  No redeploy needed for this part; propagates automatically within 5
  minutes of `bin-ai-manager` deploying the new tool definition. If the
  initial `FetchTools` call fails, `bin-pipecat-manager` continues with an
  empty tool set (graceful, not an error) until the next tick.
- **The Insight-tool whitelist** (`amai.AllowedToolNames(ai.TypeInsight)`,
  consumed by `bin-pipecat-manager/pkg/toolhandler/main.go:91-109`'s
  `GetByNames`) is **statically compiled into `bin-pipecat-manager`** via a
  local Go module replace (`bin-pipecat-manager/go.mod` →
  `replace monorepo/bin-ai-manager => ../bin-ai-manager`). Confirmed: this
  replace is a local path, and the Docker build (`COPY ./ .` +
  `go mod vendor` at build time) re-vendors current monorepo source — **no
  `go.mod` edit is needed**, but `bin-pipecat-manager` must be **rebuilt and
  redeployed** for `emit_info_card` to clear the whitelist filter.

Deployment order is unconstrained in both directions (verified):
`bin-ai-manager` first → `bin-pipecat-manager` still filters the tool out
(safe, tool simply unavailable, no error). `bin-pipecat-manager` first → its
`AIV1ToolList` cache doesn't have the definition yet (safe, same graceful
degradation). No coordination beyond "both eventually deploy" is required.

### 1.2 Message storage / LLM-feedback separation is required

Current structure (`pkg/aicallhandler/tool.go:88-153`) is a round-trip:

```
json.Marshal(messageContent)              // :118
  -> messageHandler.Create(..., RoleTool, string(content), ...)  // :123, DB write
    -> unmarshalToolResponse(msg)          // :94, re-parses the SAME stored string
      -> returned to pipecat-manager as the tool result fed to the LLM
```

The stored string and the LLM-fed string are identical today. Adding a
`Blocks` field naively would put the full card JSON into the LLM's prompt —
wasted tokens, and risk of the LLM re-narrating the card's fields in prose.

**Decision**: break the round-trip for this tool only. The `emit_info_card`
handler builds `tmpContent` (with `Blocks` populated) once; two representations
are derived from it explicitly:

1. **Stored / frontend-facing representation** — includes `Blocks`, persisted
   via `messageHandler.Create`.
2. **LLM-facing representation** — the existing `Message` (human-readable
   text) field only, `Blocks` excluded. Built directly from `tmpContent`,
   not by re-parsing the stored JSON.

`unmarshalToolResponse` remains the source of the LLM-facing value for all
*other* tools (no behavior change there); `emit_info_card`'s handler
constructs its LLM-facing map directly instead of routing through it.

**This alone is insufficient — a second injection point exists (found in
design review round 3) and must be closed in the same change.** Insight
aicalls do not maintain LLM history in-process; `send.go:35-41,138`
(`SendReferenceTypeOthers`, the branch `ReferenceTypeContactCase` always
takes) calls `startPipecatcall` fresh on **every user message**, and
`start.go:705`'s `getPipecatcallMessages` (`start.go:620-661`) rebuilds the
LLM prompt history by reading up to 100 stored messages back out of the DB
and including every non-`RoleNotification` message's raw `Content` string
verbatim (`start.go:637-657`, `RoleTool` included with its `tool_call_id`).
Without a second fix, a card's `blocks` JSON — excluded from the *first*
turn's tool-result feedback by the fix above — reappears in full on **every
subsequent turn** of the same conversation, via this history rebuild. The
original draft's claim that this design keeps card JSON out of the LLM's
prompt context was only true for one turn; verified false as a
standing/N-turn property.

**Fix, part 1 (tool-result message)**: strip `blocks` at the
LLM-history-build boundary, not at storage/first-feedback time. In
`getPipecatcallMessages`'s loop (`start.go:637-657`), when `m.Role ==
message.RoleTool`, unmarshal `m.Content`, delete `blocks` if present,
re-marshal, and use that stripped string as the history entry's `content`
— for every other tool this is a no-op (no `blocks` key to strip). The
stored DB row and the frontend-facing webhook payload are untouched (still
carry `blocks`, per D3's need for it); only the read-path that feeds the
LLM's next-turn prompt is filtered. `toolCreateResultMessage` (`tool.go:123`)
is confirmed to be the only producer of `role: tool` rows, and it always
marshals a `messageContent` struct, so `m.Content` is always well-formed
JSON for this role in practice — the strip step should still fall back to
the original, unmodified `m.Content` string on an unmarshal error
(defensive, matches this codebase's general parsing conventions) rather
than dropping or erroring the history entry.

**Fix, part 2 (a third injection point — found in design review round 5,
and NOT closed by part 1)**: before a tool even executes,
`tool.go:47`'s `messageHandler.Create(..., RoleAssistant, "",
[]message.ToolCall{*tool}, "", ...)` persists the LLM's original
tool-call request — including `Function.Arguments`
(`models/message/tool.go:15-18`), the raw JSON string of arguments the LLM
generated (for `emit_info_card`, this is the card's title/description/
fields — essentially the same content as `Blocks`, just pre-execution
instead of post-execution). This is stored on the `RoleAssistant` message's
`ToolCalls` field (`models/message/main.go:23`, `db:"tool_calls,json"`),
and `getPipecatcallMessages`'s same loop re-attaches it verbatim on every
subsequent turn (`start.go:648-650`: `if len(m.ToolCalls) > 0 {
tmp["tool_calls"] = m.ToolCalls }` — this `RoleAssistant` message is not
`RoleNotification`, so it isn't skipped). Part 1 alone strips the
*duplicate* copy in the tool-result message; this third copy, in the
*tool-call request* that necessarily precedes it, survives untouched.

Fix: in the same `start.go:637-657` loop, when appending a `ToolCall`
whose `Function.Name == message.FunctionCallNameEmitInfoCard`, replace its
`Arguments` with a minimal placeholder (e.g. `{}`) before attaching to
`tmp["tool_calls"]`. **The `tool_calls` entry and its parent assistant
message must NOT be removed entirely.**

**Rationale, corrected in design review round 6** (the original draft
overstated this as strictly required for API correctness — investigation
found the real picture is more nuanced, and independently surfaced a
separate, serious, pre-existing bug, filed as
[VOIP-1460](https://voipbin.atlassian.net/browse/VOIP-1460)):
`bin-pipecat-manager/scripts/pipecat/run.py` has two separate sites with
this filter (citation corrected, design review round 7): `:450`'s
`valid_messages = [m for m in messages if m.get("role") and
m.get("content")]` in `create_llm_service` (the single-AI path), and an
inlined equivalent at `:637` (`start_messages.extend([m for m in
llm_messages if m.get("role") and m.get("content")])`, variable named
`start_messages` there) in `init_team_pipeline` (the team path). Both
already drop this tool-call-request message today, for **every** Insight
tool (not just `emit_info_card`) — because its `content` is always `""`
(falsy in Python), that message never reaches the LLM regardless of what
`Arguments` contains. So in the *current* code, whether Part 2 keeps or
strips the `tool_calls` entry makes no observable difference to what the
LLM sees; the entry is already being removed one layer downstream, in
Python, before this design's Go-side change would even run.

That downstream removal is itself the problem VOIP-1460 tracks: it leaves
the paired `role: tool` message (non-empty content, so it survives the
filter) orphaned with no preceding `tool_calls` entry. Confirmed via
production log inspection: this exact orphaned pattern exists in live
traffic today. Provider-dependent outcome — the Gemini adapter degrades
gracefully (silently substitutes a placeholder function name, a quality
issue, not a crash), while the OpenAI/Grok path has no equivalent
recovery logic and is a plausible source of hard `400` failures on
providers configured that way (not confirmed live in the log window
inspected, which happened to contain only Gemini traffic).

**Given that**, Part 2 is kept as **defense-in-depth, not a currently-load-
bearing fix**: if/when VOIP-1460 is fixed (e.g. by changing the Python
filter to preserve non-empty-`tool_calls` messages regardless of `content`),
the tool-call-request message will start reaching the LLM again, and at
that point an un-neutered `Arguments` field would reintroduce the N-turn
card-JSON leak this design is trying to close. Keeping the entry (rather
than assuming its removal is permanent/guaranteed) and neutering only
`Arguments` is what makes this fix correct independent of whether/when
VOIP-1460 lands — this is the actual reason to keep the entry, not
API-pairing correctness, which is presently enforced by an accidental
Python-side removal rather than by this design's Go-side behavior.

With both parts, all three injection points are closed:
(1) the tool-call request's `Arguments` (part 2), (2) the immediate
tool-result RPC feedback within the current turn (the earlier
`unmarshalToolResponse`-bypass fix above), and (3) the tool-result message's
`content` on every subsequent turn's history rebuild (part 1). All three
are corrections to the *read* path (`getPipecatcallMessages` and the
immediate RPC-return construction) — the stored DB rows and the
frontend-facing webhook payload are never modified.

### 1.3 `RunLLM: true`, with a prompt guard against duplicate narration

`RunLLM: true` (i.e. the LLM writes a short follow-up after the tool
result) is chosen as a trade-off: a card with no natural-language wrap-up
feels abrupt in a conversational flow, and `RunLLM: true` is technically
straightforward to support (§1.2's separation already produces a
LLM-appropriate summary). The cost is one extra LLM round-trip per card.
This is not "consistency with existing tools" — the six existing Insight
tools are pure data-retrieval tools where `RunLLM: true` is load-bearing
(with no it, the user sees nothing); `emit_info_card`'s output (the card) is
itself the primary artifact, so `RunLLM` is a genuine UX choice, not a
technical requirement.

To prevent the LLM from redundantly restating the card's fields in its
follow-up text, the tool description includes an explicit behavioral guard
— a pattern with direct precedent in this codebase
(`pkg/toolhandler/definitions.go:30,100,181,468` all instruct the LLM on
exactly what to say/not say depending on `run_llm`).

**Multi-call-per-turn**: no artificial cap. If the LLM calls
`emit_info_card` N times in one turn, N cards render in sequence (each is
an independent `RoleTool` message, already how every other tool behaves at
N calls). The tool description asks the LLM to use it once per turn unless
genuinely presenting multiple distinct items, as a soft prompt-level
deterrent against spam — no code-level limit.

### 1.4 Tool categorization: Insight-only, "read-only" invariant reworded

`emit_info_card` performs no DB write and no external API call — it does
not cross a new permission boundary (an Insight AI can already emit
arbitrary text via a normal assistant message on the same UI surface). It
belongs in `AllInsightToolNames`, not a new category, and **not** in
`AllToolNames` (voice/Normal AI has no UI surface to render a card on).

Because "read-only" read literally would exclude it (it does write a
message — but only into its *own* session's message stream, the same
surface a plain assistant-text turn already writes to), the invariant
comment is reworded from "must be read-only" to "must not have side effects
outside the session's own message/expression surface" in **both** places
that state it:

- `models/tool/main.go:62-64` (the canonical comment)
- `models/ai/allowed_tools_test.go:64-71` (the test's doc comment restates
  the same definition — both must move together or the definition is only
  half-updated)

`models/ai/allowed_tools_test.go:73-80`'s `knownReadOnly` allowlist gets
`tool.ToolNameEmitInfoCard` added — required, or
`TestAllInsightToolNamesAreReadOnly` (`allowed_tools_test.go:72`, the
consent-gate test owning `knownReadOnly`) fails red. (Corrected test name,
design review round 3 — the sibling test
`TestValidateToolNames_WriteToolNeverAllowedForInsight`, line 94, checks
`AllToolNames` minus `AllInsightToolNames` and is unaffected by adding an
Insight-only tool.)

### 1.5 Frontend: pre-existing role-filtering defect (VOIP-1458), now Highest priority

Independently discovered during this analysis, filed as VOIP-1458, escalated
to Highest after a second discovery (see below).

**Original finding**: `bin-api-manager`'s `ServiceAgentAImessageList` filter
(`pkg/servicehandler/serviceagent_aimessage.go:52-55` — corrected citation,
design review round 1; a sibling function `AImessageGetsByAIcallID` in
`aimessage.go:64-67` happens to have the identical 2-key filter, so both
paths lack a role filter) is `{deleted, aicall_id}` only — no `role` filter. Both frontend panels (`square-talk`'s
`CaseInsightAssistantPanel.jsx`, `square-admin`'s
`CaseInsightAssistantPanel.js`) branch only on `msg.role === 'user'`;
everything else (including `role: 'assistant'` with empty `content` from a
tool-call request, and `role: 'tool'` with the tool result's raw JSON
string) renders through `MarkdownMessage` as-is. Confirmed by reading both
render sites directly — no role filter exists anywhere in the pipeline (API
-> WebSocket dispatch -> Redux/store -> render).

**Escalating discovery**: `bin-ai-manager/pkg/aicallhandler/start.go:812-819`
stores the Insight system prompt, init prompt, and parameter JSON as
`role: 'system'` messages on the *same* `aicall_id`. With no role filter,
these are equally unfiltered and render as markdown-formatted chat bubbles.
**The actual defect is not "ugly JSON leaks into the chat" — it's "the AI's
system prompt is visible to the agent viewing the panel."** Prompt-injection
material and internal prompt-engineering IP, both exposed. VOIP-1458
re-titled and repriced Highest accordingly.

**Fix, scoped for VOIP-1455's frontend phase (same render site, done
together)**: replace the implicit denylist (`role !== 'user'` -> render)
with an explicit, closed **allowlist**. A message renders if and only if:

1. `role === 'user'`, or
2. `role === 'assistant'` and `content` is non-empty, or
3. `role === 'tool'` and `content` parses as JSON and that JSON has a
   non-empty `blocks` field.

Everything else — `role: 'system'`, `role: 'assistant'` with empty content
(tool-call requests), `role: 'tool'` without `blocks` (every *other* tool's
result, e.g. `get_contact_profile`) — is excluded. This is a closed list, so
any future/unknown role is excluded by default rather than leaking through.

Verified this allowlist doesn't over-exclude: the only other `message.Role`
values in the domain (`RoleNone`, `RoleFunction`, `RoleNotification`) are
either dead code (`RoleFunction`, never constructed anywhere) or
voice/Pipecat-team-switch-only (`RoleNotification`, generated exclusively by
`EventPMTeamMemberSwitched` — not reachable from this text-based Insight
panel). The panel's error banner is a separate local `useState`, not part of
the `msg.content` stream, so it's unaffected by this filter.

**Verified per-app difference** (both apps need the change, but not
identically): `square-admin` has an `_intermediate` streaming branch
(`store.js:107,141`, panel line ~113) that `square-talk` does not have at
all (`_intermediate` — zero matches). The allowlist above applies to
*non*-`_intermediate` (final) messages only; each app's existing
`_intermediate` handling is untouched.

**Deferred to design decisions below** (flagged by review, not yet decided
in the issue-analysis document): which layer applies the allowlist (message
array filter vs. component return `null`), and whether square-admin's
`React.memo` custom comparator (`content`/`role`/`_intermediate` only,
CaseInsightAssistantPanel.js:143-149) needs to include `blocks` in its
comparison once `blocks` exists as a field it should react to. Resolved in
§3 below.

**Related exposure found in `square-main` (design review round 2, filed as
[VOIP-1459](https://voipbin.atlassian.net/browse/VOIP-1459), not fixed in
this ticket)**: `bin-ai-manager/pkg/aicallhandler/start.go:790-819`'s
system-prompt-persisting logic is **not Insight-only** — its `switch` has a
`default: defaultCommonAIcallSystemPrompt` branch that fires for Normal AI
too, and `square-main`'s public chat widget
(`square-main/src/components/widget/ChatMessage.jsx:6-8`:
`const role = message.role || 'assistant'; const isUser = role === 'user'`)
has the identical unfiltered-role rendering defect. **If confirmed
exploitable, this would warrant at least VOIP-1458's Highest severity** —
the blast radius is arguably larger (unauthenticated/public surface vs. an
internal agent panel) — but unlike VOIP-1458, this path is currently
**not known to be live**: it appears to be blocked only by an incidental
timing accident (`initialize()` POSTs, triggering the system message,
before `ws.subscribe()` runs, and there's no history backfill, so the
system-message event is typically published before the widget is
listening — not a deliberate control, and could silently break if that
ordering ever changed). VOIP-1459 is filed at High (not Highest) pending
that confirmation — its own first task is reproducing the exposure before
any fix is designed; escalate to Highest immediately if reproduction
succeeds. Out of scope for VOIP-1455 itself (no shared code path with this
ticket's frontend fix — `square-main` isn't part of the Insight Assistant
surface at all).

**Verification note**: this conclusion is based on exhaustive static
reading (all 4 lines of the API filter, both full render components) plus
an affirmative signal — neither panel's test suite has a single
`role: 'system'` or `role: 'tool'` fixture (grep: 0 matches, vs. 87 for
`role: 'user'`/`'assistant'` combined) — consistent with this path having
never been exercised by tests, which is consistent with the defect having
gone unnoticed. Live browser reproduction was attempted twice in this
session and blocked by browser-automation session/account issues (landed on
an unrelated "Guest" account with no cases), not by the bug failing to
reproduce. **Live confirmation is deferred to this ticket's verification
step** (screenshot the fix working post-implementation), not required as an
issue-analysis gate.

### 1.6 Documentation and schema decisions

- **RST docs** (`bin-api-manager/docsdev/source/ai_overview.rst`,
  `ai_struct_tool.rst`, `ai_struct_ai.rst`) updated per the root CLAUDE.md's
  mandatory RST-sync rule, with a clean Sphinx rebuild
  (`rm -rf build && python3 -m sphinx -M html source build`) and
  `git add -f docsdev/build/`.
- **No Alembic migration.** `Blocks` is embedded as JSON *inside* the
  existing `ai_messages.content` `TEXT` column (confirmed 65,535-byte
  limit, migration `f46d9c5c4438`), not a new column. This trades away
  SQL-level querying of card content (e.g. "find all messages with a card")
  for avoiding a schema change + Alembic review cycle, which is
  disproportionate for this feature's scope. Documented as a known
  limitation; revisit if card search/aggregation becomes a real
  requirement.
- **`messageContent` struct** (`tool.go:102-108`) gets a new `Blocks
  []CardBlock \`json:"blocks,omitempty"\`` field. `omitempty` is mandatory —
  this struct wraps the result of all 21 existing tools; without
  `omitempty` every tool's stored/returned JSON shape changes.
- **Size guard**: since `Blocks` shares the 64KB `content` column budget
  with `Message`, the tool's parameter JSON Schema caps field count (≤20)
  and per-field value length (≤500 chars) — enforced by the schema itself
  (LLM-constrained), with a defensive truncation fallback in the handler
  (see §3).

### 1.7 Known, accepted residual risk

`get_aicall_messages` (Normal-AI-only tool, `tool.go:731-779`) marshals
another session's full `message.Message` rows (same-customer, cross-`aicall`)
into the calling session's LLM context, gated only by a `CustomerID` match
(`tool.go:756-758`). Since `Blocks` is embedded in `content`, a card's JSON
could theoretically be re-injected into an unrelated session's LLM prompt
via this path. Impact is narrow (Normal-AI-only tool, same customer, no
rendering surface for it — it becomes inert prompt text there) and this
tool's broader behavior is out of this ticket's scope to change. Recorded
here as a known, accepted limitation of the "embed in `content`" decision.

**All in-process readers of `ai_messages` enumerated (design review round
7).** A third reader beyond `getPipecatcallMessages` and
`get_aicall_messages` exists: `renderAIcall`
(`pkg/aicallhandler/tool_resource.go:591`, used by other tools to summarize
a past aicall). Checked and confirmed safe, no work needed —
`renderAIcallMessageLines` (`tool_resource.go:640-665`) drops `RoleTool`
entries entirely (`default: return nil`) and renders assistant tool-calls
as a bare `[assistant called <name>]` marker with no `Arguments`. This is
also the in-repo precedent both this design's frontend allowlist (§1.5) and
D2's "assistant tool-call rows have `Content == ""`" premise (VOIP-1460)
independently rediscovered.

---

## 2. Requirements

1. Insight Assistant (`square-talk`, `square-admin`) can render a
   structured, single "info card" (title + description + key/value fields)
   as part of an assistant turn, driven by a new LLM tool `emit_info_card`.
2. The card is not narrated redundantly by the LLM's own follow-up text.
3. No DB schema change (Alembic) required.
4. No raw/system/tool-call debris renders in either panel any longer
   (VOIP-1458, folded into this ticket's frontend phase).
5. No regression to VOIP-1454's Markdown rendering, existing tool behavior,
   or existing panel tests.

## 3. Design decisions

### D1 — `CardBlock` shape (wire format, shared by backend and frontend)

```json
{
  "type": "info",
  "title": "string, required, <=200 chars",
  "description": "string, optional, <=1000 chars",
  "fields": [
    { "label": "string, <=50 chars", "value": "string, <=500 chars" }
  ]
}
```

`type` is a discriminator, fixed to `"info"` for this ticket — allows future
card types (e.g. `"list"`, `"contact"`) to share the `blocks` array without
a wire-format break. `fields` capped at 20 entries (§1.6's size guard).

**Enforcement (corrected in design review round 1 — the original draft had
this backwards):** the JSON Schema's `maxItems`/`maxLength` on the tool's
parameters is a **hint to the model only, not an enforced constraint**.
Verified: `bin-pipecat-manager/scripts/pipecat/run.py:418-421,434-439` never
sets `function.strict`, and explicitly warns if a caller tries to — without
strict mode, OpenAI-style function calling does not guarantee schema
compliance. **The handler-side truncation is therefore the primary and only
real enforcement** (drop trailing fields past 20, truncate individual
values past 500 chars with a trailing `...`), following this codebase's own
established pattern of handler-level clamping over schema-level limits
(`tool_insight.go:63-70`'s `resolveInsightListLimit`,
`tool_insight.go:603-608`'s `insightContactAddressLimit` — neither of which
rely on JSON Schema `maxItems`/`maxLength`; `definitions.go` has zero uses
of either keyword anywhere in its 939 lines). The schema's `maxItems`/
`maxLength` are still included as a model-facing hint (cheap, and reduces
how often truncation actually triggers), but §7's security table must
credit the handler truncation, not the schema, as the actual DoS control.

### D2 — Backend: where `Blocks` lives in the message pipeline

`message.Message` (`models/message/main.go`) does **not** get a new field.
Instead:

- `messageContent` (`tool.go:102-108`, the tool-result wrapper) gets
  `Blocks []CardBlock \`json:"blocks,omitempty"\``.
- `emit_info_card`'s handler is the only one that ever populates `Blocks`.
- Storage: `toolCreateResultMessage` marshals the full `messageContent`
  (with `Blocks`) into `content` exactly as today — no new code path, just
  a new field flowing through the existing `json.Marshal`. This reaches the
  frontend unchanged: `models/message/webhook.go:29-46`'s
  `ConvertWebhookMessage` passes `Content` straight through with no
  re-shaping, so the `blocks`-bearing JSON string arrives at the frontend
  inside `msg.content` exactly as stored (confirmed in design review round 1).
- LLM feedback: `emit_info_card`'s handler does **not** call
  `unmarshalToolResponse` on its own stored message. It builds the
  RPC-return map directly from `tmpContent`. All 21 other tools are
  unaffected — they still go through `unmarshalToolResponse` unchanged.

**What `tmpContent.Message` holds (resolved in design review round 7 — the
original draft left this genuinely undecided across three sections, and
both implicit readings were broken).** `tmpContent` is marshaled wholesale
into the stored `content` (`toolCreateResultMessage`, `tool.go:118-123`),
so `Message` is simultaneously the frontend-facing value, the first-turn
LLM feedback, and (after Part 1's strip, below) the N-turn LLM history
value — one field, three consumers, so it cannot both be empty (clean
frontend, but the LLM has zero record a card existed) and be the full
field values (LLM has too much, and D5's frontend would need to actively
suppress rendering it to avoid a literal "Card 'X' displayed to the user."
string appearing in the UI). **Decision: `Message` is a short, title-only
trace** — e.g. `"Displayed an info card titled '<title>'."` — identical in
all three roles, no divergence between turn 1 and turn N, and containing
no field values (preserves §1.3's anti-duplicate-narration goal). D5 is
amended accordingly: a card-bearing `role: 'tool'` message renders **only**
the `CardBlock`(s), never `content.Message` — the trace exists for the
LLM's benefit (so a later "what was on that card again?" has *some* context
to work from, short of the full field values), not for display.

**Confirmed locally implementable (design review round 1)**: `tool.go`'s
control flow is a single branch point — `tmpMessageContent` (built at
line ~82 by the dispatched handler) is still in scope at line 94 where
`unmarshalToolResponse(msg)` is called, so
`if function.Name == message.FunctionCallNameEmitInfoCard { <build map
directly from tmpMessageContent> } else { unmarshalToolResponse(msg) }` is a
clean, local addition. No larger refactor is needed.

**LLM-facing map key set (must match the existing wire shape exactly, to
avoid pipecat-manager/Python seeing a tool-dependent shape)**: the existing
`unmarshalToolResponse` path produces keys from `messageContent`'s JSON
tags — `tool_call_id`, `result`, `message`, `resource_type`,
`resource_id` (no `omitempty` on any of them, since `json.Unmarshal`
populates every key present in the marshaled source). `emit_info_card`'s
direct-construction path must populate the same five keys. **`resource_type`/
`resource_id` values (corrected in design review round 7 — the original
draft's claim that empty strings here "match what other info-only tools
already do" is false: every existing `fillSuccess` call site across
`pkg/aicallhandler/*.go` passes a non-empty `resource_type`, e.g.
`"contact_profile"`, `"call_transcript"`; the only empty *values* anywhere
belong to `resource_id`, in `case_create`, which still has a real
`resource_type: "case"`)**: follow that actual precedent —
`resource_type: "card"`, `resource_id: ""` (no addressable resource ID for
a card the way a case/call/contact has one). No `blocks` key in this map,
and no key added or removed relative to what every other tool's
pipecat-facing payload looks like.

### D3 — Frontend: allowlist layer

Applied as an **array filter**, not a component-return-`null`: build the
allowlist-filtered array once (alongside `mergedMessages`/`buildTimelineItems`
in each panel), before date/session-separator generation. Rationale
(resolving the deferred item from §1.5): filtering post-`buildTimelineItems`
risks an orphaned separator (a date/session marker with zero visible
messages under it, if the only message on a given day was filtered out).
Filtering the raw message array first means separators are only ever
generated for days/sessions that have at least one renderable message.

**`blocks` is not a wire field — it must be derived, not read** (corrected
in design review round 1). `WebhookMessage` has no `blocks` property; a
card's data lives *inside* the `content` JSON string (D2). So rule 3 of the
allowlist ("`role === 'tool'` && `content` parses as JSON && that JSON has
a non-empty `blocks` field") means: `JSON.parse(msg.content)` and check
`.blocks`, not read `msg.blocks` directly. A shared helper,
`parseToolCardBlocks(content)`, returns the parsed `blocks` array or `null`
(on parse failure or absence) and is used both by the filter (truthiness
check) and by the render path (§5) to get the actual array to hand to
`CardBlock`. This is a small, cheap pure function on a <64KB string, but
each of its two call sites still needs its own memoization boundary so
it isn't re-run on every render: the filter step is one `useMemo` over the
whole message array (deps: `[mergedMessages]`, mirroring the existing
`mergedMessages` -> `timelineItems` chain already in both panels — see §5),
and the render step is a separate, per-bubble
`useMemo(() => parseToolCardBlocks(msg.content), [msg.content])` inside the
bubble component. Both are required; neither is optional for the
"negligible cost" characterization to hold.

**Explicit `_intermediate` pass-through rule** (missing from the original
draft, added in review): a message with `_intermediate === true` always
passes the filter regardless of rule 2's "content non-empty" check. Reason:
`_intermediate` messages are mid-stream deltas (their `content` starts
empty and grows token-by-token in square-admin; `store.js:107,141`) and are
rendered through each app's existing pre-VOIP-1454 streaming-bubble path,
which is orthogonal to this allowlist. Excluding empty-content intermediate
messages would make the streaming bubble flicker in and out as tokens
arrive. (`square-talk` has no `_intermediate` messages at all — see §1.5 —
so this rule is a no-op there, included for parity/clarity.)

**Empty-state gate must read the filtered array, not the raw one**
(blocking fix from review round 1). Both panels currently gate their empty
state on the raw, unfiltered message count:
`square-admin/CaseInsightAssistantPanel.js:784` and the equivalent line in
`square-talk` both check `mergedMessages.length === 0`. Every Insight
session starts with 2-3 `role: 'system'` messages persisted immediately
(`bin-ai-manager/pkg/aicallhandler/start.go:812-819`), all of which the
allowlist excludes. Left as-is, `mergedMessages.length` would be nonzero
(gate does not fire) while the *visible* timeline is empty until the first
real assistant reply — the panel would render a false empty scroll region
instead of the intended "Analyzing..." loading state. **Fix**: the
empty-state gate must check the length of the post-allowlist array, not
`mergedMessages`.

### D4 — Frontend: `square-admin`'s memo comparator (withdrawn as originally
written; replaced with a simpler fix)

The original draft proposed extending `MessageBubble`'s custom `React.memo`
comparator (`CaseInsightAssistantPanel.js:143-149`) to also compare
`msg.blocks`. **This is wrong and was caught in design review round 1**:
since `blocks` is not a wire field (D3), there is no stable `msg.blocks` to
compare — comparing `undefined === undefined` is a silent no-op, and naively
attaching a freshly-`JSON.parse`d array to the message object before
comparing would make the comparator see a new array reference on every
render, permanently defeating the memo (the exact failure mode the
comparator's own inline comment warns against).

**Actual fix: no comparator change needed.** Because `parseToolCardBlocks`
is called from a `useMemo` scoped *inside* `MessageBubbleImpl`, keyed on
`msg.content` (D3), the parsing itself is already gated by React's
dependency comparison at the point of use — not by the outer `React.memo`
comparator. The existing comparator's `content` comparison
(`CaseInsightAssistantPanel.js:143-149`, unchanged) is sufficient: if
`content` hasn't changed, `MessageBubbleImpl` doesn't re-render, so the
inner `useMemo` doesn't re-run either. `square-talk` has no memo comparator
at all and needs no change for the same reason (the local `useMemo` pattern
in `CardBlock`'s render path is app-agnostic).

### D5 — Frontend: `CardBlock` component

New component in each app (no shared package — same rationale as VOIP-1454:
no npm workspace root, three independent design-token systems). Renders
`title` (semibold), `description` (muted), `fields` as a 2-column
label/value list, inside a bordered card container distinct from the plain
message bubble background (an actual box, not just formatted markdown —
this is the point of the feature per the original product ask). **A
card-bearing `role: 'tool'` message renders only the `CardBlock`(s) — never
`content.Message`** (corrected in design review round 7, alongside D2's
`Message` decision above): `Message` is a short trace that exists solely
for the LLM's benefit on later turns, not a caption meant for display;
rendering it in the UI would show the literal internal string
`"Displayed an info card titled '<title>'."` next to the card, which is not
useful to the agent viewing the panel.

---

## 4. Backend change plan (`bin-ai-manager`)

- `models/tool/main.go`: add `ToolNameEmitInfoCard ToolName = "emit_info_card"`;
  add to `AllInsightToolNames`; reword the read-only invariant comment
  (§1.4).
- `pkg/toolhandler/definitions.go`: add the tool definition — name,
  description (with the RunLLM/no-duplicate-narration guard, §1.3), JSON
  Schema parameters (`title` required, `description` optional, `fields`
  array with `maxItems`/`maxLength` hints per D1, plus a `run_llm` boolean
  property matching every other `RunLLM: true` tool's parameter shape,
  e.g. `definitions.go:472-476,771-775,865-869`), `RunLLM: true` as the
  tool's default metadata.
- `models/message/tool.go:25-52` (confirmed sole location of
  `FunctionCallName` constants, design review round 1): add
  `FunctionCallNameEmitInfoCard`.
- `pkg/aicallhandler/tool.go`:
  - Add `CardField{Label, Value string}` and `CardBlock{Type, Title,
    Description string; Fields []CardField}` types in this package (no
    existing shared `models/message` home for card-specific types; keep
    them next to `messageContent` since only `emit_info_card`'s handler
    constructs them).
  - Add `Blocks []CardBlock \`json:"blocks,omitempty"\`` to `messageContent`
    (`tool.go:102-108`).
  - Add `toolHandleEmitInfoCard(ctx, aicall, tool) *messageContent` (new
    file `tool_emit_info_card.go`, following the existing per-tool file
    convention e.g. `tool_insight.go`): parses/validates/truncates
    parameters per D1 (handler-side truncation is the actual enforcement,
    per D1's correction), builds `tmpContent` with `Blocks` populated.
  - Add its dispatch entry to the `mapFunctions` map (`tool.go:54-76`,
    confirmed exact location and signature
    `map[message.FunctionCallName]func(context.Context, *aicall.AIcall,
    *message.ToolCall) *messageContent`, 21 existing entries).
  - Bypass `unmarshalToolResponse` for this tool specifically (D2,
    confirmed locally implementable — see D2's "Confirmed locally
    implementable" note).
- `models/ai/tool_validation.go`: **no code change** (confirmed, design
  review round 1: `tool_validation.go:36-47`'s `case TypeInsight: return
  toSet(tool.AllInsightToolNames)` derives directly from
  `models/tool`'s list — no duplicate list exists to keep in sync).
- `pkg/aicallhandler/start.go:637-657` (`getPipecatcallMessages`'s history-
  build loop), two changes (D2's fix parts 1+2, found in design review
  rounds 3 and 5 — together close all three card-JSON re-injection paths
  into the LLM's N-turn prompt; both are no-ops for every other tool):
  - Part 1: when `m.Role == message.RoleTool`, unmarshal `m.Content`,
    delete the `blocks` key if present, re-marshal before using it as the
    history entry's content (fall back to the original string on an
    unmarshal error).
  - Part 2: when attaching `m.ToolCalls` to the history entry, for any
    `ToolCall` whose `Function.Name ==
    message.FunctionCallNameEmitInfoCard`, replace `Function.Arguments`
    with a minimal placeholder (e.g. `{}`) in a **copied** `[]message.ToolCall`
    (build a new slice/values; do not mutate `m.ToolCalls` in place — safe
    today since `messageHandler.List` returns freshly-built structs with no
    cache layer, but mutation-free is the correct default regardless) — the
    `ToolCall` entry itself (id/type/name) must be preserved, only its
    `Arguments` payload is neutered. **Why keep the entry at all, given
    Python currently drops it anyway (§1.2's `Message` note above,
    VOIP-1460): defense-in-depth, not current API-pairing necessity** — if
    VOIP-1460 changes the Python filter to stop dropping this message,
    keeping the (neutered) entry here is what prevents the card's raw
    `Arguments` from reaching the LLM at that point; stripping the entry
    entirely would be simpler today but would silently reopen this leak
    the moment VOIP-1460 ships.
- `models/ai/allowed_tools_test.go`: add `tool.ToolNameEmitInfoCard` to
  `knownReadOnly` (§1.4); reword the doc comment (§1.4).
- RST docs: `bin-api-manager/docsdev/source/ai_overview.rst`,
  `ai_struct_tool.rst`, `ai_struct_ai.rst` — add `emit_info_card` following
  the existing tool documentation pattern; clean Sphinx rebuild; force-add
  `build/`.
- OpenAPI: `bin-openapi-manager/openapi/openapi.yaml` — add
  `emit_info_card` to the `AIManagerToolName` enum (+ `x-enum-varnames`,
  per the get_contact_profile design doc's rev4 note that both must move
  together).

## 5. Frontend change plan (`monorepo-javascript`, separate PR)

Both `square-talk/src/features/cases/CaseInsightAssistantPanel.jsx` and
`square-admin/src/views/contacts/CaseInsightAssistantPanel.js`:

- Add a shared `parseToolCardBlocks(content)` helper (small pure function,
  co-located with each panel or in a shared chat-utils module per app):
  attempts `JSON.parse(content)`, returns `.blocks` if present and
  non-empty, else `null`. Used by both the allowlist filter (D3, truthiness
  check) and the render path (below).
- Add the allowlist filter (D3) applied to the raw message array before
  `buildTimelineItems`, using `parseToolCardBlocks` for rule 3 and the
  explicit `_intermediate` pass-through rule. Implement as its own
  `useMemo` step with deps `[mergedMessages]` (i.e. it depends on
  `mergedMessages`'s *output*, the same way `timelineItems` already does —
  `square-talk` panel line 453 pattern — not on `mergedMessages`'s own
  upstream deps) so the filter (and its `parseToolCardBlocks` calls) only
  re-runs when the underlying message data actually changes, not on every
  render — this is what keeps the "two independent `parseToolCardBlocks`
  call sites" cost negligible in practice (see D3's corresponding note).
- **Switch the empty-state gate** (`mergedMessages.length === 0` in both
  panels) to check the post-allowlist array's length instead (D3 blocking
  fix).
- Add `CardBlock` component (D5) per app (`square-talk/src/components/chat/`,
  `square-admin/src/components/`, matching VOIP-1454's established
  locations for chat-adjacent components).
- In the message-bubble render path: for a `role: 'tool'` message that
  passed the allowlist, call `useMemo(() =>
  parseToolCardBlocks(msg.content), [msg.content])` to get the actual
  `blocks` array (memoized per D4's replacement fix — no comparator change
  needed), and render `CardBlock` for each entry instead of the existing
  `MarkdownMessage`/plain-text branch.
- `square-admin` only: **`src/views/ais/constants.js`'s `AVAILABLE_TOOLS`**
  (found missing from the original draft in design review round 1) — this
  list mirrors `bin-ai-manager`'s `tool.AllToolNames`/`AllInsightToolNames`
  per its own in-file comment and `square-admin/CLAUDE.md`'s "Field Sync
  Points" section; every existing Insight tool (`get_contact_profile`,
  `get_call_transcript`, etc.) is registered there. Add `emit_info_card` so
  admins configuring an AI with an explicit tool list (not `["all"]`) can
  enable it. `square-talk` has no equivalent tool-selector UI — no change
  needed there.
- Tests: extend each panel's existing test suite with cases for (a) a
  `role: 'system'` message is not rendered, (b) an empty-content
  `role: 'assistant'` tool-call message is not rendered, (c) a `role:
  'tool'` message without `blocks` (i.e. every other tool's result) is not
  rendered, (d) a `role: 'tool'` message with `blocks` renders a
  `CardBlock` with the expected title/description/fields, (e) existing
  `role: 'user'`/content-bearing-`assistant` cases still render exactly as
  before (regression guard), (f) the empty-state gate still shows
  "Analyzing..." when only `role: 'system'` messages exist (new session,
  D3's blocking fix), (g) an `_intermediate: true` message with empty
  content still renders the streaming bubble (D3's pass-through rule).

---

## 6. Testing strategy

**Backend** (Go, `bin-ai-manager`):
- Unit tests for `toolHandleEmitInfoCard`: valid input -> correct
  `messageContent.Blocks`; field/value truncation at the D1 caps; empty
  `fields` array; missing optional `description`; `Message` is the
  title-only trace (`"Displayed an info card titled '<title>'."`), never
  the field values (D2's `Message` decision, design review round 7).
- Unit test confirming the LLM-facing return value for this tool excludes
  `Blocks`, includes the same title-only `Message` trace (not empty, not
  the field values), and uses `resource_type: "card"` / `resource_id: ""`
  (regression guard for D2's first-turn bypass).
- Unit tests for `getPipecatcallMessages`'s two strip steps (D2's fix parts
  1+2, design review rounds 3 and 5): after a card-bearing turn (tool-call
  request + tool-result both persisted), a subsequent call to
  `getPipecatcallMessages` (i.e. the history rebuilt for the *next* user
  turn) must show (a) the `role: 'tool'` entry's `content` string contains
  no `blocks` key, (b) the preceding `role: 'assistant'` entry's
  `tool_calls[].function.arguments` is the placeholder, not the original
  card JSON, while (c) that same `tool_calls` entry is still present with
  its original `id`/`type`/`function.name` intact (defense-in-depth guard
  for D2 Part 2 — Python currently drops this entry downstream regardless,
  per VOIP-1460, but this test asserts Go's own output is already correct
  independent of that, so the fix keeps working once VOIP-1460 changes the
  Python side). All other stored fields (role, tool_call_id, the
  human-readable `message` text) unchanged. This three-part test is the
  actual regression guard for the N-turn property — the first-turn-only
  test above is necessary but not sufficient on its own, and neither is
  part 1 alone.
- `models/ai/allowed_tools_test.go`'s existing consent-gate test passes
  with `emit_info_card` in `knownReadOnly`.
- Full verification workflow per root CLAUDE.md: `go mod tidy && go mod
  vendor && go generate ./... && go test ./... && golangci-lint run -v
  --timeout 5m` in `bin-ai-manager`.

**Frontend**: see §5's test list. Full existing suites for both panels must
pass unmodified (regression signal), plus the new allowlist/`CardBlock`
cases.

## 7. Security

| Vector | Control |
|---|---|
| System prompt / internal params exposure (VOIP-1458) | Closed allowlist (D3) — `role: 'system'` never renders |
| Raw tool-result JSON exposure (VOIP-1458) | Closed allowlist — `role: 'tool'` only renders with valid `blocks` |
| Card content is LLM-generated, untrusted-by-policy | `CardBlock` renders as plain React text nodes (no raw-HTML-injection props, no markdown re-parse of field values) — auto-escaped by React, consistent with VOIP-1454's security posture |
| Content-length DoS via oversized card | D1's handler-side truncation (20 fields, 500 chars/value) — the actual enforcement, since function calls aren't in `strict` mode and the JSON Schema caps are a model hint only |
| Cross-session data re-injection via `get_aicall_messages` | Known, accepted, narrow-impact residual risk (§1.7) — not fixed in this ticket |

## 8. Out of scope / follow-ups

1. Buttons, polls, charts (VOIP-1455's parent scope, phase 2b+) — separate
   design cycles.
2. Additional card `type` values beyond `"info"` (D1 leaves room for this).
3. Fixing `get_aicall_messages`'s cross-session re-injection path more
   generally (§1.7) — file separately if it becomes a priority.
4. **Fixing the orphaned-tool-message defect in `bin-pipecat-manager`'s
   Python message filter** (found in design review round 6, confirmed via
   production log inspection, affects all 6 existing Insight tools, not
   just `emit_info_card`) — tracked in
   [VOIP-1460](https://voipbin.atlassian.net/browse/VOIP-1460), Highest
   priority, independent fix. D2's Part 2 (§3 D2) is deliberately designed
   to remain correct whether or not VOIP-1460 is fixed, but does not fix
   VOIP-1460 itself.
5. `square-main` — no Insight Assistant integration exists there, so
   `emit_info_card`/card rendering is not applicable. Its separate,
   higher-severity variant of VOIP-1458 (see §1.5) is tracked in
   [VOIP-1459](https://voipbin.atlassian.net/browse/VOIP-1459), not fixed
   here.

## 9. Success criteria

- [ ] `emit_info_card` tool defined, dispatched, and stores a card-bearing
      `role: 'tool'` message without putting card JSON into the LLM's
      prompt context — verified as an N-turn property across all three
      injection points (the tool-call request's `Arguments`, the immediate
      tool-result feedback, and the tool-result's `content` on every
      subsequent turn's rebuilt history), per D2's fix parts 1+2.
- [ ] `bin-pipecat-manager` redeployed; tool usable by Insight AI within 5
      minutes of `bin-ai-manager`'s deploy.
- [ ] Both `square-talk` and `square-admin` render the card as a bordered
      component, not raw text.
- [ ] Neither panel renders `role: 'system'` messages, empty-content
      tool-call `role: 'assistant'` messages, or non-card `role: 'tool'`
      messages, in any theme.
- [ ] All existing tests (both apps, `bin-ai-manager`) pass unmodified;
      new tests cover the allowlist and card-rendering paths.
- [ ] RST docs + OpenAPI enum updated.
- [ ] No Alembic migration.
