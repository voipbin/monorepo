# Analysis: Phase 2 — expose MCP tools to the LLM (PR B)

Date: 2026-09-27
Author: CPO (Hermes)
Status: v1 of this document (carries forward rounds 1-4 findings from the split
parent), Analysis Review Loop pending
Scope: **PR B only.** Lifecycle correctness and docs (PR A) and square-admin
parity (PR C) are in `2026-09-27-mcp-lifecycle-and-docs-analysis.md`.

Parent history: this document was split out of
`2026-09-27-mcp-lifecycle-and-phase2-analysis-v4-superseded.md` at the CEO's
direction after round 4, because the lifecycle half had converged while this half
kept producing scope-changing discoveries each round.

## 0. Hard prerequisite: PR A must land first

PR B is what makes `CallTool` reachable by an LLM. Today the dispatch path is
unreachable (§2.1), so a deleted MCP server's exposure is limited to an unwanted
credentialed `ListTools` round trip. **Landing PR B before PR A's revocation and
ownership gates would convert that into live "a deleted or foreign server's tools
are callable by the model" exposure.** This ordering is not negotiable and must be
stated in both PR bodies.

## 1. Issue statement

**D7** `bin-api-manager/docsdev/source/ai_struct_ai.rst:69` and
`bin-openapi-manager/openapi/openapi.yaml:2242-2243` publish that a whitelisted MCP
server's tools are merged into the AI's tool list and presented to the LLM. **No
code path does this.** Customers can register a server (UI shipped), whitelist it
on an AI (UI shipped), read documentation promising tool use (shipped), and nothing
happens.

CEO decision 2026-09-27: **implement it**, do not retract the docs. Verbatim: "Q1.
구현하자." And as one PR, not split: "하나로 가자."

Prerequisites discovered while scoping D7, each of which would independently break
the feature:
- **D9** The MCP tool input schema never unmarshals (json tag mismatch), producing
  `"parameters": null`, which crashes the python schema builder for the **entire**
  tool list including built-ins.
- **D10** The tool list pipecat sends is built by a deny-by-default whitelist no MCP
  name can pass; and ai-manager's same-named resolver has **no** type filter, so the
  obvious alternative reintroduces a known fail-open bug.
- **D11** No dedupe, length, charset, or count validation on namespaced tool names.
  The single-AI and team paths differ: the team path already sanitizes and
  truncates, the single-AI path does not.
- **D15** A fourth AIcall creation path — the realtime voice path, VoIPBin's
  flagship — never writes the `mcp_tool_map` metadata that dispatch depends on, so
  MCP tool calls there would fail closed 100% of the time.

## 2. Validity re-check (code-verified)

### 2.1 D7: the merged tool list is discarded; the live advertising surface is pipecat ALONE.

`resolveTools` (`bin-ai-manager/pkg/aicallhandler/mcp_tool.go:52`) returns
`([]tool.Tool, map[string]aicall.McpToolRef, error)`. **All three callers throw away
the first return value:**

```
pkg/aicallhandler/start.go:1169            _, mcpToolMap, errTools := h.resolveTools(ctx, a)
pkg/aicallhandler/insight_session.go:225   _, mcpToolMap, errTools := h.resolveTools(ctx, a)
pkg/aicallhandler/mcp_tool.go:284          _, mcpToolMap, err := h.resolveTools(ctx, a)
```

`grep -rn 'resolveTools' pkg/ --exclude=*_test.go` returns exactly those three plus
the definition and comments.

**Live surface count is ONE, not two (corrected in round 4).** v4 presented
`engine_openai_handler` as a second surface. It is dead code:
- `StreamingSend` has **no callers** — repo-wide grep (excluding `vendor/`,
  `.worktrees/`, mocks) returns only the interface declaration
  (`pkg/engine_openai_handler/main.go:24`) and the definition
  (`streaming_send.go:24`).
- The only production calls on that handler are `Send`/`SendOnce` with
  caller-built requests that set no `Tools`: `pkg/analysishandler/run.go:98`,
  `pkg/summaryhandler/content.go:238`, `:357`.
- `MessageSend` (`message.go:35-38`) sets no `Tools` and also has no caller.
- Even if it ran, tool names could not escape: `streamingResponseHandleTool`'s
  `Function:` assignment is **commented out** (`streaming_send.go:190-193`).

So `engine_openai_handler/tool.go:8-11`'s static `tools` var is dead code,
explicitly **out of PR B scope**, and should be called out in the PR body so
reviewers do not ask.

**`engine_dialogflow_handler` is provably tool-free.** Its interface is a single
method (`pkg/engine_dialogflow_handler/main.go:11-13`); `getRequest` builds a
`DetectIntentRequest` carrying only `QueryInput_Text` (`message.go:84-94`); the
response collapses to a plain string (`message.go:56-74`). No `Tools` field, no
tool-call channel, no `mcp` reference. **No dialogflow change needed.**

**The live surface** is pipecat: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:150`
`tools = h.toolHandler.GetByNames(ai.Type, ai.ToolNames)`, handed to
`h.pythonRunner.Start(...)` at `runner.go:173-189`.
`grep -rni 'mcp' bin-pipecat-manager/ --include=*.go` (excluding `vendor/`) = **zero
matches**.

**Consequence:** the LLM is never told any `mcp_<8hex>_<tool>` name exists, so it
cannot emit one, so `toolHandleMcpCall` is unreachable. This is per design —
`docs/plans/2026-09-11-mcp-tool-integration-design.md:72-74` defers the realtime
path to "Phase 2" and `:290-293` states pipecat is untouched by construction. The
docs simply shipped ahead of the code.

**The receiving half is already complete.** `pkg/aicallhandler/tool.go:141-142`
already routes any unknown tool name carrying `mcpToolNamePrefix` to
`toolHandleMcpCall`, and that function re-reads live state rather than trusting
stored metadata: `mcp_tool.go:144` re-resolves the AI, `:151` re-checks whitelist
membership, `:157` re-fetches the server row, `:163` re-checks `Status`. So PR B's
work is confined to **advertising** (plus D15's metadata write).

### 2.2 D15: the realtime voice path never writes `mcp_tool_map`. BLOCKER.

There are **four** AIcall creation paths, not three. The fourth calls `resolveTools`
zero times:

```go
// pkg/aicallhandler/start.go:1087  startAIcallByRealtime
// …
// :1110-1115
snapshots, autoAudit := h.buildPromptSnapshots(ctx, a, assistanceType, assistanceID, activeflowID)
metadata := map[string]any{
	aicall.MetaKeyPromptSnapshots:  snapshots,
	aicall.MetaKeyAutoAuditEnabled: autoAudit,
}
```

No `resolveTools`, no `MetaKeyMcpToolMap`. This is the `ReferenceTypeCall` path:
`startReferenceTypeCall` → `:272` → `startPipecatcall` at `:279`.

`grep -rn 'MetaKeyMcpToolMap' pkg/ --include=*.go | grep -v _test` returns exactly
three write sites: `start.go:1176` (inside `startAIcallByMessaging`),
`insight_session.go:232`, `mcp_tool.go:298`. **None is reachable from the realtime
branch.**

**Consequence:** PR B advertises at `runner.go:150`, which serves EVERY pipecatcall
including voice. On a voice AIcall, `lookupMcpToolRef` (`mcp_tool.go:186-189`) finds
no `mcp_tool_map` key and returns false, so `toolHandleMcpCall` fails closed with
"unknown mcp tool call" for **100% of MCP tool calls on VoIPBin's flagship path.**
Advertised but non-functional is the worst possible combination.

PR B must add the write at `start.go:1112`, and §5 must cover `ReferenceTypeCall`.

### 2.3 D9: the input schema never unmarshals, and nil poisons the whole list. BLOCKER.

```go
// pkg/mcptoolhandler/main.go:23-27
type McpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}
```

The MCP specification's wire field is **`inputSchema`** (camelCase). Go's
`encoding/json` matches tags case-insensitively but **not**
underscore-insensitively, so `input_schema` never matches and `InputSchema` is nil
for every spec-compliant server.

The nil flows through: `pkg/aicallhandler/mcp_tool.go:97` `Parameters:
mt.InputSchema`, into `tool.Tool.Parameters` whose tag is `json:"parameters"` with
**no `omitempty`** (`bin-ai-manager/models/tool/main.go:108`). So the wire carries
`"parameters": null`.

Python then fails, and fails wide:
- `bin-pipecat-manager/scripts/pipecat/main.py:34` — `parameters: Optional[dict] = None` accepts null; `main.py:121` `model_dump()` emits the key WITH `None`.
- `scripts/pipecat/tools.py:75` — `tool.get("parameters", {"type": "object", ...})`. **The key is PRESENT with value None, so the default is NOT applied.**
- `scripts/pipecat/run.py:434` — `params = func.get("parameters", {})` → `None`; `:439` `params.get("additionalProperties")` → **AttributeError**.
- `run.py:432` loops over **every** tool in the session, so one MCP tool with a nil
  schema takes down the built-in tools with it.

The per-server fail-closed guarantee of `resolveTools:73-88` does NOT extend to the
advertising step. No test covers it (`grep -n schema pkg/mcptoolhandler/*_test.go`
returns nothing).

Fix: correct the tag to `inputSchema`, AND default `Parameters` to
`{"type":"object","properties":{}}` rather than passing nil through, since a server
may legitimately omit the schema.

### 2.4 D10: two different `GetByNames`, and both options are constrained.

**First, a v4 correction.** v4 cited `bin-pipecat-manager/pkg/toolhandler/main.go:96-104`
as the filter `resolveTools` must get past. **`resolveTools` does not call that
function.** There are two distinct implementations:

| | `bin-pipecat-manager/pkg/toolhandler` | `bin-ai-manager/pkg/toolhandler` |
|---|---|---|
| Signature | `GetByNames(aiType amai.Type, names []aitool.ToolName)` | `GetByNames(names []tool.ToolName)` — **no aiType** |
| Type filter | `if !allowed[t.Name]` from `amai.AllowedToolNames(aiType)` (`main.go:100`) | **none** |
| On `all` | still type-filtered | `return toolDefinitions` — everything (`main.go:35-38`) |
| Used by | `runner.go:150`, `run.go:178` | `resolveTools` via the `toolNameResolver` interface (`mcp_tool.go:29-31`), wired at `cmd/ai-manager/main.go:167` |

`amai.AllowedToolNames` (`bin-ai-manager/models/ai/tool_validation.go:36-47`)
returns `toSet(AllToolNames)` / `toSet(AllInsightToolNames)`. **A namespaced MCP
name is by construction absent from those sets**, and pipecat's implementation also
iterates only `h.tools` (from `AIV1ToolList`, the static catalog), so MCP tools are
not even candidates there.

This whitelist is not incidental: `bin-pipecat-manager/pkg/toolhandler/main.go:17-26`
documents it as the closure of a fail-open bug (Normal AIs receiving Insight-only
tools), corroborated at `bin-ai-manager/models/tool/main.go:27` and
`runner.go:136`.

**Both design options are therefore constrained:**
- **(a) Extend pipecat's `GetByNames` / whitelist to admit MCP names.** Must not
  weaken the type control it exists to enforce.
- **(b) Ship ai-manager's merged list to pipecat instead.** This reintroduces the
  fail-open bug by construction, because ai-manager's resolver applies **no** type
  filter. Choosing (b) requires type-filtering ai-manager's built-in resolution
  FIRST.

Either way this is a security-relevant decision requiring explicit review, not an
implementation detail.

### 2.5 D11: name validation is absent in ai-manager and asymmetric in python.

`pkg/aicallhandler/mcp_tool.go:90-105` has no dedupe, no collision detection, no
name validation, no count cap:

- **Prefix collision.** Two servers whose ids share the first 8 hex chars produce
  identical prefixes (`mcpServerIDShort` `:114-120`). If both expose a same-named
  tool, `merged` (`:94`) gets **two entries with identical `Name`** while `toolMap`
  (`:101`) keeps only the last: the LLM sees a duplicate function name (OpenAI
  rejects with 400) and any call resolves to the wrong server. The comment at
  `:113-115` calls the prefix "collision-resistant-enough" — an assumption, not a
  check.
- **Count.** `ListTools` (`pkg/mcptoolhandler/client.go:190-207`) caps the response
  body at 1 MiB (`:152`, constant `1 << 20` at `pkg/mcpserverhandler/ssrf.go:18`)
  but not the tool count, so a verbose or hostile server can flood every prompt.
- **Length / charset — and here the two python paths DIFFER (round 4 correction):**
  - Team path: `scripts/pipecat/team_flow.py:14-29` **already** sanitizes
    (`_INVALID_CHARS_RE.sub('_', name)`) and truncates to 64 chars
    (`_MAX_FUNCTION_NAME_LENGTH`). So an invalid or overlong remote name does not
    produce a provider rejection there; it silently mangles, and two names sharing
    their first 64 sanitized chars **collide silently**.
  - Single-AI path: `scripts/pipecat/tools.py:63-80` passes the name through
    unchanged, so the provider-rejection failure mode is real there.

  `mcp_` + 8 hex + `_` already consumes 13 chars against OpenAI's
  `^[a-zA-Z0-9_-]{1,64}$`, so a 52+ char remote name is enough to trigger either
  failure mode.

**Verified safe, do NOT "fix":**
- `lookupMcpToolRef` (`mcp_tool.go:185-206`) is a whole-string map lookup against
  `MetaKeyMcpToolMap`; it never splits the name, so underscores inside remote tool
  names are harmless.
- A remote tool named `connect` cannot shadow the built-in, because
  `tool.go:139-142` consults `mapFunctions` first and only falls through on the
  `mcp_` prefix.

### 2.6 The pipecat crossing: three hops, TWO advertising surfaces, TWO registration mechanisms.

1. ai-manager `resolveTools` produces the merged list (today discarded).
2. pipecat Go builds its own list via `GetByNames` — **two separate sites**:
   - `runner.go:150` — single AI.
   - `run.go:178` — team, inside `resolveTeamForPython` (function begins at
     `run.go:137`; the member loop is `:171-213`), which ships per-member `Tools`
     inside `resolvedTeam`.
   - `runner.go:126-129` skips the single-AI branch entirely when a team is
     resolved, so **both must change.**
3. `pythonRunner.Start(...)` (`runner.go:173-189`) marshals and **HTTP POSTs to
   `localhost:8000/run`** (`pythonrunner.go:119-120`; field `Tools []aitool.Tool` at
   `:94`) into FastAPI `PipelineRequest.tools` (`main.py:90`).
4. Python registration — **two mechanisms (v4 mis-cited this):**
   - Single-AI: `tool_register` (`tools.py:101-123`), called from `run.py:258`.
     Registers one wrapper per tool name; an unregistered name is dropped with a
     `[missing_tool]` log feeding an alerted Prometheus counter
     (`test_missing_tool_logging.py`).
   - Team: **`tool_register` is never called** — stated explicitly at `tools.py:24-25`
     and `run.py:637-639`. The real mechanism is `team_flow.py:63-70`, where each
     member's tools become
     `FlowsFunctionSchema(name=_sanitize_function_name(tool["name"]),
     properties=tool.get("parameters", {}).get("properties", {}), …)`, wired to the
     LLM services via `FlowManager` with `flow_manager._llm = routing_llm`
     (`run.py:743`).

**`team_flow.py` was absent from every prior version of this analysis, yet it is the
file PR B must change for the team surface.**

**Note `aitool.Tool` IS `bin-ai-manager/models/tool.Tool`** —
`pythonrunner.go:14` imports `aitool "monorepo/bin-ai-manager/models/tool"`. There
is no type-conversion problem; v3's framing of one was wrong.

Team-member-level `mcp_server_ids` semantics are undefined and must be decided here.

### 2.7 Metadata lifecycle: bounded, but the refresh matrix is uneven.

- **No unbounded growth, no truncation risk.** `refreshMcpToolMap`
  (`mcp_tool.go:298`) assigns `MetaKeyMcpToolMap` wholesale, replacing rather than
  merging. The column is `JSON NOT NULL DEFAULT (JSON_OBJECT())`
  (`bin-dbscheme-manager/…/2929f1291813_ai_aicalls_add_metadata.py:20-23`), i.e.
  MySQL JSON with no narrow length cap. A large map is a token/perf concern (D11's
  count cap) not a data-loss risk.
- **But the refresh is not universal (v4 claim corrected).** `refreshMcpToolMap` has
  exactly one caller: `start.go:361`, in the **conversation** reuse branch. The
  contact_case reuse branch (`start.go:583-592`) calls only
  `refreshInsightSessionIfIdle`, which returns `kept` without writing metadata in
  four of its branches (`insight_session.go:68-72`, `:75-79`, `:88-91`, `:92-94`);
  only a genuine refresh reaches `writeInsightSessionMetadata:232`. The
  `StatusInitiating` retry branch (`start.go:575`) writes nothing either.

  So a long-lived reused Insight thread carries a stale `mcp_tool_map`
  indefinitely. Severity is **low once PR A lands** (dispatch re-checks the live
  whitelist at `mcp_tool.go:151` and status at `:163`, and PR A adds the ownership
  and `tm_delete` checks), but PR B must state the refresh matrix per reference type
  rather than claiming blanket pruning.

## 3. Proceed decision

### PROCEED, as ONE PR, AFTER PR A.

CEO decision Q5: one PR rather than B1 (ai-manager prep) + B2 (pipecat wiring),
because B1 alone would be another "no observable effect" PR and reviewers would lose
the context that motivates it.

Scope:

| Area | Work |
|---|---|
| `bin-ai-manager` | D9: fix the `inputSchema` tag + non-nil `Parameters` default. D11: dedupe / length / charset / count validation on namespaced names. D15: write `MetaKeyMcpToolMap` in `startAIcallByRealtime` (`start.go:1112`). D7: stop discarding the merged list; possibly type-filter built-in resolution (if §4 Q1 picks option (b)). |
| `bin-pipecat-manager` Go | Both advertising sites: `runner.go:150` (single AI) and `run.go:178` (team, `resolveTeamForPython`). Resolve how MCP names pass the `AllowedToolNames` control (D10). |
| `bin-pipecat-manager` python | Both registration mechanisms: `tools.py` (`tool_register`, single-AI) and `team_flow.py:63-70` (team). Schema default hardening. Reconcile the sanitize/truncate asymmetry (D11). |

### Explicitly out of scope

- `engine_openai_handler` — dead path (§2.1); its static `tools` var stays as-is.
- `engine_dialogflow_handler` — provably tool-free (§2.1).
- Everything in the PR A document.
- **Tool-level (per-tool) whitelisting.** `resolveTools:91-105` adds every tool a
  server returns. Deferred: no demand signal, and `mcp_server_ids []uuid.UUID`
  extends additively. **Not consequence-free:** server-level whitelisting grants all
  present AND FUTURE tools that server exposes, i.e. a third party can silently
  expand its own scope. Mitigation is one explicit docs sentence, not the feature.
- Making the stale-metadata refresh universal across reuse branches (§2.7) — low
  severity once PR A lands; note it, do not fix it here.

## 4. Decisions to lock before the design doc

1. **D10: how do MCP names reach the model?** (a) extend pipecat's whitelist to
   admit them without weakening the type control, or (b) ship ai-manager's merged
   list — which requires type-filtering ai-manager's built-in resolution first,
   because it currently has none. Recommend (a): it preserves a control that exists
   to close a known fail-open bug, and keeps the blast radius inside pipecat.
2. **Are MCP tools permitted on `TypeInsight` AIs?** This is D13 in the PR A
   document (`ValidateMcpServerIDs` takes no AI type, unlike `ValidateToolNames`).
   **PR A owns the write-time gate; PR B must not advertise a combination PR A
   rejects.** The answer must be consistent across both PRs.
3. **D11 policy.** On an 8-hex prefix collision: extend the prefix, reject the
   second server's tool, or fail the whole resolution? On an invalid/overlong remote
   name: skip that tool or skip that server? What is the per-server tool-count cap?
   And: should the single-AI path adopt `team_flow.py`'s sanitizer, or should the
   team path stop silently mangling and start rejecting, so both behave alike?
   Recommend one shared validation in ai-manager so python never has to sanitize.
4. **Team-member-level semantics.** Does each team member's `mcp_server_ids` govern
   its own tools independently? (Note PR A fixes the start-vs-current member
   whitelist mismatch, D12, which this depends on.)
5. **D15 placement.** Write the tool map in `startAIcallByRealtime` only, or hoist
   the `resolveTools` call so all four creation paths share one code path?
   Recommend hoisting if it does not disturb the other three, since four independent
   call sites is how this defect appeared in the first place.

## 5. Risk table

| Risk | Severity | Mitigation |
|---|---|---|
| PR B lands before PR A → deleted/foreign servers' tools become LLM-callable | **High** | Hard ordering A→B; state it in both PR bodies |
| MCP tools advertised on the realtime voice path but dispatch fails closed 100% of the time (D15) | **High** | Write `MetaKeyMcpToolMap` at `start.go:1112`; §6 must cover `ReferenceTypeCall` |
| One malformed remote schema or name breaks the ENTIRE tool list including built-ins (`run.py:432` loops all tools) | **High** | D9 + D11; validate per tool and default `Parameters` before the list crosses to python; test that built-ins survive one bad MCP tool |
| Option (b) for D10 reintroduces the fail-open bug that `AllowedToolNames` exists to fix | **High** | §4 Q1; if (b) is chosen, type-filter ai-manager's built-in resolution in the same PR |
| Team path silently truncates/mangles names (`team_flow.py:14-29`), so two remote tools can collide invisibly | Medium | D11; decide one shared validation policy (§4 Q3) |
| An unbounded remote tool list inflates every prompt (token cost, latency) | Medium | D11 count cap |
| `mcp_server_ids` grants all present and FUTURE tools of a server | Medium | One explicit docs sentence (§3 out of scope) |
| Team and single-AI registration diverge, so a fix applied to one surface silently misses the other | Medium | §2.6; §6 requires both surfaces tested |
| Advertising an Insight-AI combination PR A's write-time gate rejects | Medium | §4 Q2; keep both PRs consistent |
| Stale `mcp_tool_map` on reused contact_case Insight threads | Low | §2.7; PR A's live re-checks make it inert |
| Reviewer confusion over the dead `engine_openai_handler` tools var | Low | State the exclusion and the evidence in the PR body |

## 6. Verification plan

Go gates in BOTH `bin-ai-manager` and `bin-pipecat-manager`: `go mod tidy && go mod
vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.

**Plus the python suite** at `bin-pipecat-manager/scripts/pipecat/`: `test_run.py`,
`test_init_pipeline.py`, `test_missing_tool_logging.py`, `test_team_flow.py`. PR B
necessarily touches `tools.py`, `run.py`, and `team_flow.py`, so this suite is
mandatory (no prior version of this analysis listed it).

Required cases:
- a whitelisted server's tool actually reaches the model, and a deleted server's
  does not (post-PR A);
- **`ReferenceTypeCall` (realtime voice) end to end** — the D15 regression;
- **built-in tools survive** an MCP tool with a nil / oversized / invalid-charset
  name or schema (the D9 blast-radius guarantee);
- duplicate namespaced names (8-hex prefix collision) are detected, not silently
  last-write-wins;
- the tool-count cap holds;
- **both** advertising surfaces: single-AI (`runner.go:150`) and team
  (`run.go:178` / `resolveTeamForPython`);
- **both** registration mechanisms: `tool_register` (`tools.py:101-123`, via
  `run.py:258`) and `team_flow.py:63-70`;
- a `[missing_tool]` counter assertion confirming MCP tools are *registered* on the
  python side, not merely transmitted;
- the AI-type decision from §4 Q2 is enforced consistently with PR A.

## 7. Inherited lessons

This document carries the failure modes found in rounds 1-4 of the parent analysis.
The recurring shape: **a claim about behavior inferred from the existence or
apparent symmetry of code, instead of traced end-to-end.**

Specifically for PR B, four claims were wrong before verification:
- `resolveTools`' merged list was assumed used; it is discarded.
- `aitool.Tool` vs `models/tool.Tool` was assumed a conversion problem; same type.
- `tools.py:113-116` was cited as the team registration path; the team path never
  calls `tool_register` — it is `team_flow.py:63-70`.
- `engine_openai_handler` was cited as a live second surface; `StreamingSend` has no
  callers.
- Three AIcall creation paths were assumed exhaustive; there are four, and the
  fourth is the flagship voice path.

Rules in effect:
- For any producer function, grep its callers and check whether the return value is
  discarded (`_,`).
- For any struct unmarshalling an external wire format, verify the json tag against
  the actual specification, not local naming convention.
- When a value crosses a process/language boundary, read the receiving side's
  parsing code before asserting the contract holds.
- Enumerate EVERY path that creates the same resource and verify each
  independently — never assume two paths are symmetric.
- Before citing a function, confirm which implementation the caller actually binds
  to (same name, different package, different signature).
- Before citing a code path as live, grep for its callers.

## 8. Review history

### Round 1 (2 reviewers, both CHANGES_REQUESTED)

Both reviewers rejected the analysis. Five of its claims were wrong or
understated, and five prerequisites were missing entirely. Every correction
below was re-verified against real code before being recorded here.

#### 8.1 Corrections to claims this document already made

C1. **D7 was framed as "the value is discarded". That framing hides the real
work.** Two of the three discard sites are correct code:
`writeInsightSessionMetadata` (`insight_session.go:225`) and
`refreshMcpToolMap` (`mcp_tool.go:327`) exist solely to write the name-to-ref
map, so discarding the tool list there is correct by construction. The actual
defect is that **there is no consumer to pass the list to**. The live tool list
is assembled in a different service from its own static catalogue
(`bin-pipecat-manager/pkg/toolhandler`), and the start RPC carries no tools
channel: `PipecatV1PipecatcallStart`
(`bin-common-handler/pkg/requesthandler/pipecat_pipecatcall.go:15-64`) carries
`LLMMessages` and STT/TTS only, and `bin-pipecat-manager/models/pipecatcall/main.go`
has no `Tools` field. Verified: grep for `Tools` in both files returns zero
matches. Fixing D7 is not "stop discarding a value", it is "invent a
transport", and that is the largest single piece of PR B.

C2. **D10's stated cause is false, and it obscured a live defect.** The claim
was that `AllowedToolNames` would drop every MCP name. It cannot: pipecat's
`GetByNames` (`bin-pipecat-manager/pkg/toolhandler/main.go:91-108`) filters
`h.tools`, the global static catalogue fetched once via `AIV1ToolList`. A
per-AI, per-customer MCP tool is never a member of that slice, so it never
reaches the filter at line 100. The filter is not the obstacle; the missing
transport (C1) is.

The real finding, which this document had described as a hypothetical
workaround to avoid, is that **`resolveTools` already routes through the
unfiltered implementation today**: `mcp_tool.go:59-61` calls
`h.toolNameResolver.GetByNames(a.ToolNames)`, wired at
`cmd/ai-manager/main.go:167` to `toolhandler.NewToolHandler()`, whose
`GetByNames` (`bin-ai-manager/pkg/toolhandler/main.go:29`) takes no `aiType`
and applies no whitelist. The built-in half of the merged list is therefore
already fail-open with respect to the AIType separation. It is harmless only
because the list is discarded. **The moment PR B makes that list
authoritative, it reintroduces the exact leak that
`bin-pipecat-manager/pkg/toolhandler/main.go:83-90` was written to close.**
This must be fixed in the same PR, not deferred.

C3. **D9's severity was understated, not overstated.** The nil schema does not
merely lose the tool list. Traced and executed end to end: `main.py:34` accepts
`null`; `tools.py:75` `tool.get("parameters", {default})` returns `None`
because the key exists so the default never fires; `run.py:434-444` then calls
`params.get("additionalProperties")` on `None`, raising
`AttributeError: 'NoneType' object has no attribute 'get'` inside the
`init_llm` task, which `asyncio.gather` re-raises (`run.py:207-213`). The
result is that **pipeline initialisation fails: no STT, no LLM, no TTS**. It is
a total outage of the call, not of tool use. The team path has the same shape
(`team_flow.py:67-68`).

C4. **D15's write-site enumeration was incomplete in two directions.** A fifth
entry point exists that this document's four-function list omitted: `StartTask`
(`start.go:1222`) reaches `startAIcallByMessaging` with `ReferenceTypeTask`
(`start.go:1237`) and does get the map. Separately,
`startReferenceTypeContactCase` gets the map on create (`start.go:502`) but on
the reuse path refreshes only through `refreshInsightSessionIfIdle`, gated on
`AssistanceTypeAI` and `AIcallInsightSessionIdleMinutes > 0` and
idle-past-threshold and `a.Type == TypeInsight` (`insight_session.go:67-111`).
Ordinary non-idle reuse therefore carries a **stale** tool map indefinitely, so
a newly whitelisted server's tools stay uncallable until a refresh. That is the
fail-closed direction, but it is a functional gap this document did not list.
The headline D15 conclusion and its "fails closed 100% of the time" on the
voice path are confirmed precise.

C5. **D11 named the gap but not its consequences.** An invalid name is not a
per-tool degradation, it is a whole-request failure: OpenAI requires
`function.name` to match `^[a-zA-Z0-9_-]{1,64}$`, and `mcp_` plus 8 hex plus
`_` consumes 13 characters, leaving 51 for the remote name. A remote tool whose
name contains a dot, colon or space, or exceeds 51 characters, makes the entire
completion call fail. Same outage class as D9. Also unhandled: a server listing
the same tool name twice produces two identical entries in `merged` (provider
rejects duplicates; python `tool_register` silently overwrites); and
`mcpServerIDShort` (`mcp_tool.go:114-120`) gives a 2^32 namespace, so
cross-server collision is unlikely but undetected.

#### 8.2 Prerequisites this document missed

P7. **Prometheus label cardinality is customer-controlled.**
`bin-ai-manager/pkg/aicallhandler/tool.go:123` does
`promAIcallToolExecuteTotal.WithLabelValues(string(tool.Function.Name)).Inc()`,
declared with `[]string{"tool_name"}` (`main.go:216-223`). Today that value
comes from a closed enum. Once MCP tools are exposed it becomes
`mcp_<8hex>_<arbitrary customer string>`, and Prometheus counters never evict a
label set. A customer with 5 servers advertising 200 tools each adds 1,000
permanent series per replica, per customer, plus whatever a hostile server
invents on each `tools/list`. Must be bucketed to a constant or to the server
id before any MCP name reaches that call site.

P8. **No aggregate time budget, and this is already a live bug.**
`resolveTools` loops `a.McpServerIDs` **sequentially** (`mcp_tool.go:72-106`),
one network round trip per server, bounded only per request by
`McpToolCallTimeoutSeconds` (default 10, `internal/config/main.go:148`). There
is no aggregate cap and no concurrency. Meanwhile `AIV1AIcallStart` runs under
`requestTimeoutDefault = 3000` ms
(`bin-common-handler/pkg/requesthandler/main.go:151`, applied at
`ai_aicalls.go:44`), and `startAIcallByMessaging` **already calls
`resolveTools` today** (`start.go:1184`). So one customer server taking 4s to
answer `tools/list` already makes the API caller's RPC time out at 3s and
return an error while ai-manager continues past the deadline and creates the
AIcall anyway. Three servers at 10s each is 30s of hung handler. This is
shipped behaviour masked only by there being zero customer servers; PR B makes
it the common case and extends it to voice. Needs parallel fan-out, an
aggregate budget derived from the caller's deadline, and a cached tool list so
session start does not depend on a third party's uptime.

P9. **Tool output overruns a hard 64 KiB column and corrupts the
conversation.** `CallTool` joins every content item's text with no cap
(`mcptoolhandler/client.go:267-274`); the only bound is the 1 MiB body limit
(`mcpserverhandler/ssrf.go:18`). That string reaches `messageContent.Message`
and is stored in `ai_messages.content`, which migration
`f46d9c5c4438_ai_messages_alter_column_content.py:22` fixes as **`TEXT`**, so
65,535 bytes. A 300 KB tool result either fails the INSERT, orphaning the
tool-call request row already persisted at `tool.go:88` and triggering the
unpaired-message drop in `filter_valid_messages` (the VOIP-1460 corruption
class), or is silently truncated into invalid JSON so `unmarshalToolResponse`
fails. PR B needs an explicit result cap well under 64 KiB with a truncation
marker fed to the model.

P10. **No MCP initialize handshake, and the `Accept` header violates the
spec.** `doJSONRPCRequest` (`mcptoolhandler/client.go:125-138`) POSTs
`tools/list` directly with `Accept: application/json` only. There is no
`initialize`, no `notifications/initialized`, no `Mcp-Session-Id` handling and
no `MCP-Protocol-Version` header: grep for all four across
`pkg/mcptoolhandler/*.go` returns 0. The 2025-06-18 Streamable HTTP spec makes
three of these **MUST** requirements: the client MUST list both
`application/json` and `text/event-stream` in `Accept`; a server requiring a
session id SHOULD reject a non-initialize request without one with HTTP 400;
and the client MUST send `MCP-Protocol-Version` on all requests after
initialization.

Status of the live check: **unproven, not refuted.** Both hardcoded vendors
(`mcpoauthhandler/vendors.go:26,36`) reject an unauthenticated probe before any
protocol validation runs. `https://mcp.linear.app/mcp` returns 401 with an
empty body and `https://api.githubcopilot.com/mcp/` returns
`401 bad request: missing required Authorization header`, identically for both
the current `Accept` header and the spec-conformant one. So the probe
distinguishes nothing without a real token. The risk stands: PR B could pass
every unit test against an `httptest.Server` that answers a bare `tools/list`
and still return zero tools against every real server. **The acceptance gate
must be an authenticated live call against at least one of the two vendors, not
a mock.** No such test exists today (`client_test.go`, `client_oauth_test.go`
drive local `httptest` servers only).

P11. **Tool descriptions are unsanitized and unbounded straight into the model
context.** `mcp_tool.go:96` passes `Description: mt.Description` verbatim into
`tool.Tool.Description`, which becomes `FunctionSchema(description=...)`
(`tools.py:74`) in the model's tool spec on **every turn**. There is no length
cap, no content inspection, and no tool-count cap. The asymmetry is the
argument: this codebase already caps remote text bound for logs
(`capErrText(err.Error(), 200)` at `mcp_tool.go:180`, `truncateForError` at 512
in `client.go:181`) and already refuses to forward remote error text to the
model at all (`errMcpToolCallFailed`). The one remote-controlled string that
goes directly into the prompt received none of that discipline. A single server
can return one tool whose description is a megabyte of instructions.

#### 8.3 Scope changes accepted from Round 1

S1. **Exclude `TypeInsight` from PR B, and gate it at write time.**
`AllInsightToolNames` (`models/tool/main.go:90-99`) is a closed set whose
documented invariant (`:74-89`) is that every member has no side effects
outside the session. A customer MCP server's tools have an unbounded
side-effect profile, so admitting them to an Insight AI silently voids that
invariant. Note that the write-time gate does not exist either:
`ValidateMcpServerIDs` (`aihandler/mcpserver_validation.go:57`) takes no AI
type, unlike `ValidateToolNames`. PR B must reject `mcp_server_ids` on
`TypeInsight` rather than leave the behaviour emergent.

S2. **Exclude the team surface from PR B.** `resolveTools` resolves exactly one
AI's `McpServerIDs`, while the team pipeline builds tools per member from each
member's own AI (`run.go:178`). `refreshMcpToolMap` already papers over this by
resolving the current member (`start.go:369-373`), so a team's advertised set
and its stored map would disagree the moment the active member switches
mid-call. Team semantics are genuinely undefined and would be designed blind
with zero customers. Ship single-AI first.

S3. **In scope: realtime voice, conversation/messaging, task runs.** Voice is
non-negotiable despite being the larger fix, because it is the flagship surface
and the one a customer tests first. Task runs come along for free through
`startAIcallByMessaging` and are the lowest-risk pilot surface.

#### 8.4 Confirmed sound, do not re-litigate

- **The tool-call return path already carries an `mcp_` name end to end.**
  `tools.py:153-160` posts `{"function":{"name": tool_name}}`, bound at
  `runner.go:447-460` into `ammessage.FunctionCall` whose `Name` is the
  unvalidated string alias `FunctionCallName` (`models/message/tool.go:16,25`),
  through `AIV1AIcallToolExecute` and
  `processV1AIcallsIDToolExecutePost` (`v1_aicalls.go:302-340`, no name
  validation) to `ToolHandle`, where the `mapFunctions` miss falls through to
  the prefix check at `tool.go:141-142` and into `toolHandleMcpCall`. No enum
  gate and no allowlist on any hop.
- **The lifecycle gates from PR A do cover dispatch.** `toolHandleMcpCall`
  trusts stored metadata for nothing but the name-to-ref mapping, then re-reads
  live state: `resolveActiveAIForMcp` (`helpers.go:194`), a whitelist re-check
  against the freshly fetched `tmpAI.McpServerIDs`, `mcpServerHandler.Get`, and
  `mcpServerIsUsable` (`:266-279`, checking nil, `TMDelete`, `Status` and
  `CustomerID`). The transport backstop `refuseDeleted` (`client.go:197-206`)
  is invoked on both `ListTools` (`:214`) and `CallTool` (`:239`).
- **`StreamingSend` is dead code.** Declared at
  `engine_openai_handler/main.go:24`, implemented at `streaming_send.go:24`,
  with no non-test non-mock caller anywhere and no listenhandler route reaching
  it. Its package-level `tools` var (`tool.go:8`) names `connect` and
  `message_send`, which no longer exist in `models/message/tool.go`. It is not
  an advertising surface and must not be treated as one.
- **pipecat is the only live advertising surface.** `runner.go:150` for single
  AIs, `run.go:178` for teams, both over the static catalogue. `grep -rni mcp`
  across `bin-pipecat-manager` excluding vendor returns zero matches.

#### 8.5 Severity restated

The published API reference and docs site make unconditional present-tense
claims that a whitelisted server's tools are merged into the AI's tool list and
presented to the LLM, with the exact namespacing format quoted. Verified in the
served artefacts, not the sources: `gens/openapi_redoc/openapi.json` under
`AIManagerAI.properties.mcp_server_ids.description` and
`paths./mcpservers.post.description`, and the built HTML
`docsdev/build/html/ai_struct_ai.html`, `ai_struct_mcpserver.html`,
`ai_overview.html`. No beta marker, no "coming soon".

A customer following the documentation gets 200 on every call and silence
afterwards. There is no error, no customer-visible log, and **no MCP metric of
any kind** anywhere in the repository. The single accidental signal is that
`aicall.WebhookMessage` carries `Metadata` verbatim
(`models/aicall/webhook.go:41,78`) and the messaging path writes
`mcp_tool_map` into it (`start.go:1191`), so a customer subscribed to aicall
webhooks on a chat AIcall could notice the key. That is undocumented, and on
the voice path the key is never written at all. The rational customer concludes
their own server is broken and debugs their own infrastructure.

Blast radius is currently zero: production has no customer-registered MCP
servers, and all ~100 rows are api-validator data. That is precisely the window
in which to fix this properly rather than hastily.
