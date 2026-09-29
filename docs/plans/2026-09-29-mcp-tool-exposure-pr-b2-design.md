# PR B2: expose MCP tools to the LLM

Status: design, based on `2026-09-27-mcp-phase2-tool-exposure-analysis-v2.md`
(nine review rounds) and `2026-09-29-mcp-pr-b2-deferred-items-design-draft.md`
(a draft written mid-PR-B1 for five items the CEO chose to land there
instead; re-read here for its still-relevant reasoning on B10/B19/B13/B20/B28,
all already merged). This document is normative for what remains: **B2 is the
PR where advertisement turns on** (B12), plus every correctness item the v2
analysis found whose harm was dormant until a consumer exists.

Re-verified against the code at `8e108ac83` (post PR B1 merge, main HEAD as of
this design). Confirmed still true: no transport RPC exists
(`PipecatV1PipecatcallStart` and `models/pipecatcall.Main` carry no tools
field); `resolveTools` has no non-test caller; Insight sessions still call
`resolveMcpToolMap` un-gated (`insight_session.go:225`,
`start.go:1184` for messaging sessions); ai-manager's own `toolhandler.GetByNames`
(the built-in resolver `resolveTools` calls) has no `Type` parameter and applies
no `AllowedToolNames` filter; `CallTool`'s `IsError` field is declared
(`client.go:60`) and never read by any caller; `models/tool.Tool.Parameters`
still lacks `omitempty`.

## 1. Scope

**Round 1 review findings applied throughout this document (summary for the
reader tracking changes):** §2.2/§2.3 corrected the duplicate-built-in-tools
bug (Critical); confirmed `Tool.Parameters` is already `map[string]any`, so
`omitempty` is effective as designed (Critical, no code change needed beyond
the tag); §2.3a adds a session-start discovery latency budget (High); §2.2
documents the structural mcp_-prefix collision guarantee (High); §3's caller
graph is confirmed at implementation time per §2.4's note (High, B11); §8
now states the kill switch location unambiguously; §9 commits to concrete
metric labels; §12 adds the token-cost/DoS risk as an explicit CEO/CTO
sign-off item rather than a silent deferral; §2.3's `persistToolMap` now
documents its concurrency handling.

**Round 2 review findings applied (naming consistency + residual gaps):**
`ResolveMcpTools`/`resolveMcpOnly` is now the single, consistent name used in
every section (§2.1, §2.3, §2.4, §10, §11, §14) -- round 1's fix introduced
the correct MCP-only contract in §2.3 but left four other sections still
describing the old merged-list `ResolveTools`, including §11's test matrix,
which had literally specified "merged list" as the expected test outcome
(the same bug class §2.3 exists to prevent). All now corrected per §2.3's
naming table. §4/§10 add a "verify no other caller" guard on `CallTool`'s
signature change, matching the rigor already applied to `resolveMcpOnly`.
§5 clarifies which timeout bound is already enforced vs. newly added. §2.3a
now requires the PR body to record which branch (new budget vs. existing
coverage) was taken. §12 adds a second day-one risk: repeated discovery
against a flapping session has no aggregate rate limit, only a per-attempt
time budget.

**In scope for this PR (PR B2):**

| # | Item | Why it belongs here |
|---|---|---|
| B12 | Transport: per-AIcall callback RPC `AIV1AIcallToolList` from pipecat into ai-manager | This is what turns advertisement on. Everything else in this list is dormant without it |
| B2b | Non-nil `Parameters` default (add `omitempty`) | A null `parameters` key kills the whole pipecat init (`init_llm`) once one MCP tool is advertised (v2 A.3) |
| B4 | Type-filter ai-manager's own built-in resolver | `resolveTools`'s built-in half is fail-open today (an Insight AI storing `tool_names:["all"]` resolves 25 tools, 17 outside its whitelist); harmless only because nothing consumes the list. B12 makes it load-bearing |
| B10 (residual) | Exclude Insight from discovery too, not just advertisement | The B10 write-gate merged in PR B1 stops a *future* whitelist from being saved on an Insight AI, but `writeInsightSessionMetadata` still calls `resolveMcpToolMap` unconditionally today, discovering (not yet advertising) MCP tools for any Insight AI holding a pre-B1 or exempted-identical whitelist. Close the last hole: gate discovery itself on `a.Type != ai.TypeInsight` |
| B11 | Team symmetry: never write `mcp_tool_map` for `AssistanceTypeTeam` | Same reasoning as B10 -- a team AIcall's map exists today but is unreachable; B12 makes it reachable. `discoverMcpTools`/`resolveMcpToolMap` callers must skip team sessions. **Explicit product-scope decision, not merely deferred (round 1 Medium finding):** this makes MCP tools structurally unavailable to team-typed AIcalls in this PR, with no stated timeline to add them. If team AIs getting MCP tools is on any roadmap, flag that in the PR body as a known scope cut requiring its own follow-up design, rather than a detail buried in implementation |
| B24 | Honour `isError`: a remote tool failure must not read as success | `CallTool` never reads `IsError`; `toolHandleMcpCall`'s `fillSuccess` would label a hostile/broken server's error text a success once the model can act on it |
| B5 (residual) | Duplicate id / duplicate resolved name policy at the resolver | Discovery already validates the tool-name charset (`validMcpToolName`); the missing half is what happens when the SAME resolved name appears twice within one AI's whitelist (two servers whose 8-hex prefix genuinely collided, astronomically unlikely, or a mis-cased duplicate). Drop the second occurrence and log; never last-write-wins into the tool map |
| B25 | Cancel/bound an in-flight `tools/call` when the AIcall ends | Dispatch runs under `context.Background()` (`listenhandler/main.go`); once dispatch is reachable a hangup mid-call leaves a side-effecting remote call running with no consumer. Bound it by the existing `mcp_tool_call_timeout_seconds`, already read by `CallTool`; verify the caller-side RPC (`AIV1AIcallToolExecute`) timeout does not outlive it, and if it does, cap the request context passed into `CallTool` from the callback path (below) |
| B26 | `clearListenState` must not lose `mcp_tool_map` | `pkg/aicallhandler/listen.go`'s metadata-clearing path copies keys from an in-memory snapshot; verify it does not drop `MetaKeyMcpToolMap` written after that snapshot was taken. If it can, fix the same way `refreshMcpToolMap`/`writeInsightSessionMetadata` already do (re-read before merge) |
| B22 (partial) | Webhook projection: drop `mcp_tool_map` from `ConvertWebhookMessage`, add a documented `mcp_tool_status` summary | `models/aicall/webhook.go` copies `Metadata` whole with no key filtering; once tools are actually dispatchable this leaks the customer's remote server UUIDs and tool names onto the messaging webhook (breaking payload change, blast radius nil today per B12 not yet shipped -- ship both in the same PR so no window opens) |
| B27 | Global rollback config key | No way exists today to turn this off if something goes wrong in production, and this is the PR where it turns on. `internal/config/main.go` gains `mcp_tool_exposure_enabled` (default true post-merge, but present so it CAN be flipped false without a deploy of new code) |
| B17/B18 | Metric label cardinality fix + advertisement/outcome metrics | `pkg/aicallhandler/tool.go:123`'s existing metric uses a customer-controlled label; fix it in the same PR that makes the label populate for real (B12), and add the metrics that let an operator see this feature exists and is or isn't working (`mcp_tool_advertised_total`, `mcp_tool_call_outcome_total{outcome}`) |

**Deferred to a follow-up PR (PR B3, hardening), explicitly, with reasoning:**

| # | Item | Why deferred |
|---|---|---|
| B14 | Tool-list cache with invalidation | Correctness does not require it -- discovery already runs at session start under B13's budget (merged in B1). Caching is a latency/load optimization once B12 is live and real usage exists to size it against |
| B7 | Token-based (not character-based) description cap | v2 measured worst case at ~132K tokens/turn for 64 Korean-heavy tools; real MCP servers a customer registers are far fewer in practice, and the existing `mcpMaxToolsPerResolution=256` plus per-tool description already bounds the pathological case somewhat. Needs its own token-budget design (tiktoken-equivalent for whichever provider); not blocking B12's correctness |
| B15 | Registration-time `tools/list` probe | Turns discovery-at-session-start into look-up; a UX improvement once the core path is live, not a correctness item |
| B23 | `tools/list` pagination (`cursor`/`nextCursor`) | No `toolsListResult.Tools` cursor handling exists; real-world MCP servers in scope today (customer's own or common vendors) do not paginate at the tool counts VoIPBin caps (128/server); revisit if a customer reports truncation |
| B8 | Auditor (`aiaudithandler`) exposure cap for untrusted remote tool text | Second LLM consumer of MCP content; real but independent of whether the primary conversational path is correct. Needs its own review of the auditor's existing trust model |
| O1/O2/O3/O6/O7/O8/O9 | Open questions the v2 analysis left open, not required to close for this PR's correctness | Tracked below in Open Questions, carried forward |

Rationale for this split: B12 is the item that makes everything else's harm
real. The items kept in this PR are the ones the v2 analysis proved are
correctness gaps that activate *the moment* B12 ships (fail-open type filter,
Insight/team leakage, a success-labeled failure, an unbounded webhook leak, no
kill switch). The deferred items are real but their absence does not create an
incorrect behavior on day one -- they bound cost, latency, or a second-order
consumer. This mirrors the split criterion PR B1 already used successfully
("does the item fix something whose harm requires this PR's own change to be
live").

## 2. B12: the transport RPC

### 2.1 Shape (D1, D12, D13 from the v2 analysis, unchanged)

A new per-AIcall callback RPC, **not** a field added to `PipecatV1PipecatcallStart`.
Late-bound: pipecat calls back into ai-manager for the merged tool list at the
point it currently calls `h.toolHandler.GetByNames(ai.Type, ai.ToolNames)`
(`bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:150` for the single-AI
path). D12: **this supplements, it does not replace**, that call -- the
built-in half keeps going through pipecat's own `GetByNames`
(`AllowedToolNames`-filtered there already); only the MCP half is new.

```
AIV1AIcallToolList(ctx, aicallID uuid.UUID) ([]tool.Tool, error)
```

- `bin-common-handler/pkg/requesthandler/ai_aicalls.go`: new client method,
  same shape as the existing `AIV1AIcallToolExecute` (RPC to
  `bin-manager.ai-manager.request`, `ContentTypeNone`).
- `bin-ai-manager/pkg/listenhandler/main.go`: new route + dispatch case.
- `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`: new handler function
  reading the aicall, resolving its AI (or refusing for a team AIcall, D13
  below), and calling `aicallHandler.ResolveMcpTools` (§2.3's naming is
  authoritative for this method throughout this document; see the round-2
  correction note in §2.3).
- Response is the domain `tool.Tool` slice marshaled directly (style A, no new
  `response.*` DTO -- `tool.Tool` already carries json tags for the existing
  `AIV1ToolList` response).

**D13, corrected from the draft's note:** on failure (RPC error, AI resolution
failure, `resolveTools` internal error), the caller (pipecat's `runner.go`)
**fails open to built-ins only**, exactly mirroring the existing fail-closed
policy for AI-resolution failure that already exists just above this call
site (VOIP-1234 §6 v4: "a degraded tool-less session is acceptable, silently
granting write-capable tools when the AI type can't be determined is not").
Concretely: if the callback RPC fails, `tools` stays as whatever
`h.toolHandler.GetByNames(ai.Type, ai.ToolNames)` already produced (built-ins
only); the MCP half is simply absent for that session. This does **not**
touch the existing `errAI != nil` fail-closed branch (which already zeroes
`tools` entirely) -- a callback failure is a narrower, later failure than not
being able to resolve the AI at all, and must not regress to that stricter
posture.

### 2.2 Where it plugs into `runner.go`

```go
} else {
    ai, errAI := h.resolveAIFromAIcall(se.Ctx, aicall)
    if errAI != nil {
        // unchanged fail-closed branch
        tools = []aitool.Tool{}
    } else {
        tools = h.toolHandler.GetByNames(ai.Type, ai.ToolNames)
        if ai.RagID == uuid.Nil { /* unchanged search_knowledge filter */ }

        // NEW: supplement with the AIcall's MCP tools, best-effort. The RPC
        // returns MCP-derived tools ONLY (never a repeat of the built-in set
        // above) -- see §2.3's round-1 correction. Appending here must never
        // duplicate a name already in `tools`; belt-and-suspenders dedup by
        // name is applied even though the server side should never emit one
        // (defense against a future server-side regression reintroducing
        // built-ins into this response).
        mcpTools, errMcp := h.requestHandler.AIV1AIcallToolList(se.Ctx, aicall.ID)
        if errMcp != nil {
            metricsMcpToolListFallbackTotal.Inc()
            log.WithError(errMcp).Warnf("Could not list mcp tools for pipecat session %s; continuing with built-ins only", pc.ID)
        } else {
            existing := make(map[aitool.ToolName]struct{}, len(tools))
            for _, t := range tools {
                existing[t.Name] = struct{}{}
            }
            for _, t := range mcpTools {
                if _, dup := existing[t.Name]; dup {
                    log.Warnf("Dropped a duplicate mcp tool name colliding with an existing tool. tool_name: %s", t.Name)
                    continue
                }
                tools = append(tools, t)
            }
        }
    }
}
```

**Confirmed (round 1 High finding #3 resolved by inspection):** `h.requestHandler`
is already a field on `pipecatcallhandler`'s struct
(`bin-pipecat-manager/pkg/pipecatcallhandler/main.go:71`, set at construction
from `NewPipecatcallHandler`'s `reqHandler` parameter), used elsewhere in the
package already. No new wiring needed for the field itself; only the new
`AIV1AIcallToolList` method on the `requesthandler.RequestHandler` interface
(and its mock) is new.

**Name-collision defense-in-depth (round 1 High finding #5):** every MCP tool
name is already namespaced with the reserved `mcp_` prefix plus an 8-hex
server-id segment before it ever reaches this merge point
(`mcpToolNamePrefix = "mcp_"`, `mcp_tool.go:27`, applied at
`prefix := mcpToolNamePrefix + mcpServerIDShort(serverID) + "_"` in
`discoverMcpTools`). No built-in tool name starts with `mcp_` (verify by
grepping `tool.AllToolNames`/`tool.AllInsightToolNames` at implementation
time as a one-line regression guard, since this is exactly the kind of
constant list that could silently gain an `mcp_`-prefixed member later).
Given that, a same-name collision between a built-in and an MCP tool is
structurally prevented, not merely policy; the dedup-by-name loop above is
retained anyway as defense-in-depth against a future change to the prefix
convention, not because a collision is expected to occur.

### 2.3 What `AIV1AIcallToolList`'s handler does, server side (ai-manager)

**Corrected after round 1 review (Critical finding).** `resolveTools`
(existing code, `mcp_tool.go:64`) already returns **built-ins merged with MCP
tools** in one slice (`merged = append(merged, builtins...)`, then the MCP
loop appends onto the same `merged`) -- it is not, and was never designed to
be, an MCP-only view. The first draft of this section returned `merged`
verbatim from the RPC and had `runner.go` additionally keep its own
`h.toolHandler.GetByNames(ai.Type, ai.ToolNames)` call and append the RPC
result onto it (§2.2). That duplicates every built-in tool in the array
handed to Python's `init_llm` -- the same class of malformed-tool-list bug
B2b exists to prevent, just via double entries instead of a null field.
Fixed by making the RPC MCP-only and never touching pipecat's own built-in
resolution:

```go
// ResolveMcpTools returns ONLY this AIcall's MCP-derived tools (already
// namespaced, schema-decoded, capped) -- never the built-in set. Pipecat's
// runner.go keeps resolving built-ins itself via its own toolHandler.GetByNames,
// unchanged; this RPC supplements that list, so it must not repeat it.
func (h *aicallHandler) ResolveMcpTools(ctx context.Context, aicallID uuid.UUID) ([]tool.Tool, error) {
    if !h.config.McpToolExposureEnabled() { // B27, checked first, before any DB/RPC work
        return nil, nil
    }

    c, err := h.Get(ctx, aicallID) // existing method, ownership-neutral read
    if err != nil {
        return nil, err
    }
    if c.AssistanceType == aicall.AssistanceTypeTeam {
        // B11: a team AIcall's per-member tools come from the team
        // resolution path (resolvedTeam in runner.go), never from here.
        return nil, nil
    }
    a, err := h.aiHandler.Get(ctx, c.AssistanceID)
    if err != nil {
        return nil, err
    }
    // B10: Insight AIs never get MCP tools. resolveMcpOnly below also gets
    // its own internal Insight gate (2.4) as defense-in-depth, so this
    // branch is redundant belt-and-suspenders, not the only enforcement.
    if a.Type == ai.TypeInsight {
        return nil, nil
    }

    mcpTools, toolMap, err := h.resolveMcpOnly(ctx, a) // see below: extracted from resolveTools
    if err != nil {
        return nil, err
    }
    // D15: the map the dispatch path reads must be built from exactly this
    // capped, filtered list, not a wider pre-cap one.
    if errRefresh := h.persistToolMap(ctx, c, toolMap); errRefresh != nil {
        log.WithError(errRefresh).Warnf("could not persist refreshed mcp tool map for aicall %s", c.ID)
        // Non-fatal: return the resolved tools anyway. A stale/absent map
        // only affects dispatch of the MCP half, not this response.
    }
    return mcpTools, nil
}
```

**`resolveMcpOnly` is the final, authoritative name for the internal helper
(round 2 correction).** The existing `resolveTools` (built-ins + MCP merged,
used by nothing today) is refactored into `resolveMcpOnly(ctx, a)
([]tool.Tool, map[string]aicall.McpToolRef, error)` that does only the
MCP-discovery half (the `for _, d := range h.discoverMcpTools(ctx, a, true)`
loop and everything after it in the current function body), dropping the
`builtins := h.toolNameResolver.GetByNames(...)` prelude entirely. The old
`resolveTools` is deleted outright (confirmed at implementation time: grep
shows no caller outside its own tests today, per the codebase context this
design was built against), not kept as a wrapper -- there is exactly one
concept this design needs ("resolve this AI's MCP tools"), and keeping two
near-identical names in flight is what caused round 2's naming-consistency
finding. If implementation-time grepping surfaces an unexpected caller that
needs the merged view, that is new information changing this design's premise
and should trigger a fresh review round, not a silent reintroduction of
`resolveTools`.

**Naming table, authoritative for the whole document (round 2 fix):**

| Old/rejected name | Final name | Meaning |
|---|---|---|
| `ResolveTools` (RPC-facing, wrapper around merged `resolveTools`) | `ResolveMcpTools` | Exported `aicallHandler` method backing the RPC; returns MCP tools ONLY |
| `resolveTools` (existing, built-ins + MCP merged) | *(deleted)* | Superseded; no longer exists after this PR |
| `resolveMcpOnly` (round-1 draft name) | `resolveMcpOnly` | Unexported helper: MCP-only discovery + tool-map resolution, called by `ResolveMcpTools` |

Every other section of this document (§2.1, §2.4, §10, §11, §14) uses
`ResolveMcpTools`/`resolveMcpOnly` per this table; any remaining occurrence of
bare `ResolveTools` or `resolveTools` elsewhere in this document is a defect
in this document, not an intentional second concept.

This makes `AIV1AIcallToolList` (renamed for clarity to
`AIV1AIcallToolList` returning `ResolveMcpTools`'s output -- the RPC name and
the Go method name need not match; keep the RPC name generic since it is the
public wire contract) the single place that both answers "what MCP tools
does this session have" and keeps `MetaKeyMcpToolMap` fresh for dispatch,
resolving O3: **yes**, the transport RPC also writes the map.
`refreshMcpToolMap`'s existing `resolveMcpToolMap` call on the messaging
reuse path becomes redundant once this exists (both now do the same
resolution+persist work; kept for now, an independent cleanup, not required
for correctness). For Insight, the relationship is different, not redundant.
`ResolveMcpTools`'s own path has TWO independent, redundant Insight gates:
its own early-return in this section's code sample, plus `discoverMcpTools`'s
internal check (§2.4). Both are kept deliberately as defense-in-depth -- the
code comment above labels the early-return itself "redundant
belt-and-suspenders... not the only enforcement," which is precisely a
description of two independent gates for that path.

`writeInsightSessionMetadata`'s `resolveMcpToolMap` call is different. It is
a separate call site that never routes through `ResolveMcpTools` at all, so
`ResolveMcpTools`'s early-return cannot be what gates it. Only
`discoverMcpTools`'s check (§2.4) applies there, because that is the sole
gate that call site's execution path passes through.

The net effect is the same for both paths (Insight never gets MCP tools),
but the mechanism differs: two independent gates on the `ResolveMcpTools`
path, one gate on the `writeInsightSessionMetadata` path. `ResolveMcpTools`
never returns anything for Insight regardless of whether
`writeInsightSessionMetadata` also calls `resolveMcpToolMap`, so it does not
supersede that call the way it does on the messaging path. Both remain in
place for now.

**Concurrency on `persistToolMap` (round 1 Medium finding resolved).**
`ResolveMcpTools` can run more than once for the same AIcall in quick
succession (e.g. a Python-side reconnect within one call, or a retry after
the RPC's own timeout expired on the caller side but the server-side call
completed anyway). `persistToolMap` MUST follow the same
read-modify-write-by-key discipline `refreshMcpToolMap` and
`writeInsightSessionMetadata` already use today (re-read `c.Metadata` via
`h.db.AIcallGet` immediately before merging in the new
`MetaKeyMcpToolMap` value, write back only that key alongside whatever else
the fresh read shows) -- it must NOT write from the `c` this function's
caller already had in hand, which can be stale relative to a concurrent
writer. This is not a new pattern; `persistToolMap`'s implementation is
expected to be a thin wrapper reusing the exact merge logic
`refreshMcpToolMap` (`mcp_tool.go:602`) already has, parameterized to take
the tool map directly rather than re-resolving it (since `ResolveMcpTools`
already has `toolMap` from `resolveMcpOnly` in hand and must not call
`discoverMcpTools` a second time just to get the same result again).

### 2.3a Latency bound on this new hot-path call (round 1 High finding)

`ResolveMcpTools` runs synchronously on pipecat's call-setup path, before the
Python process starts, for every MCP-enabled AI's session -- this is new: no
production caller reaches `resolveMcpOnly`/`discoverMcpTools` today. The
existing bound on this path is `mcpDiscoverySlotWait = 2s` (time spent
waiting for one of two shared discovery slots), but the per-server
`tools/list` call itself is bounded by the separately configured
`mcp_tool_call_timeout_seconds` (default 10s, `internal/config/main.go:148`),
which is NOT a session-start-path budget -- it exists to bound one MCP call
in isolation (originally written for the dispatch/`CallTool` path). With up
to 8 whitelisted servers per AI (existing `ai.MaxMcpServerIDs` cap merged in
B1) and each potentially taking up to 10s before failing, discovery can add
tens of seconds to call setup in the worst case, on the critical path to
first audio.

**Fix: add a dedicated, tighter aggregate budget for the discovery-at-session-start
path**, separate from `mcp_tool_call_timeout_seconds` (which stays as-is for
dispatch). Add `mcpSessionStartDiscoveryBudgetSeconds` (suggested default
2s aggregate, matching `mcpDiscoverySlotWait`'s existing order of magnitude)
as a `context.WithTimeout` wrapping the whole `discoverMcpTools` loop inside
`resolveMcpOnly`, so a slow or hung customer server can delay
a call's setup by at most this bound, not by up to
`8 * mcp_tool_call_timeout_seconds`. This is a real gap the v2 analysis's own
B13 (merged in B1) was supposed to close but did not cover this new call
site, since B13 predates B12's existence. Confirm at implementation time
whether B13's existing budget mechanism already wraps `discoverMcpTools`
generally (it may -- re-read `discoverMcpTools`'s current context handling in
full before assuming this needs new code) and only add a new budget constant
if none currently applies to this call path. **The PR body MUST state
explicitly which branch was taken** (a new `mcpSessionStartDiscoveryBudgetSeconds`
was added, or an existing mechanism already covered this call path and no new
code was needed) -- this is a High-severity latency item, and "checked, no
action needed" must be recorded as a stated fact, not left as an implicit
absence of a diff (round 2 finding resolved).

### 2.4 B10/B11 residual gates, precisely

```go
// discoverMcpTools, top of function, after existing nil-handler guard:
if a.Type == ai.TypeInsight {
    return res // never discover for Insight -- B10 residual
}
```

Callers of `resolveMcpToolMap`/`resolveMcpOnly` for a team AIcall:
`ResolveMcpTools` above already returns before resolving an AI at all when
`AssistanceType == AssistanceTypeTeam`, which is earlier and sufficient --
`discoverMcpTools` takes an `*ai.AI`, and a team session's "AI" is per-member,
resolved elsewhere (`resolveTeamForPython`). No change needed inside
`discoverMcpTools` itself for B11; the gate belongs at the caller that would
otherwise resolve a single top-level AI for a team AIcall, which per the code
already read (`insight_session.go`, `start.go:1184`) only ever fires for
`AssistanceTypeAI` sessions today. Verify this holds for every one of
`resolveMcpToolMap`'s three call sites during implementation
(`insight_session.go:225`, `start.go:1184`, `mcp_tool.go:327`
`refreshMcpToolMap`) -- if any can run for a team AIcall, add the same guard
`discoverMcpTools` now has.

## 3. B4: type-filter ai-manager's own built-in resolver

`bin-ai-manager/pkg/toolhandler.GetByNames` currently has no `Type` parameter
(only a `names []tool.ToolName` selector with an `"all"` special case) and is
called from `resolveTools` as `h.toolNameResolver.GetByNames(a.ToolNames)`
with no type filtering at all. Bring it to parity with
`bin-pipecat-manager/pkg/toolhandler.GetByNames`, which already re-applies
`amai.AllowedToolNames(aiType)` regardless of what `names` claims:

```go
// bin-ai-manager/pkg/toolhandler
type ToolHandler interface {
    GetAll() []tool.Tool
    GetByNames(aiType ai.Type, names []tool.ToolName) []tool.Tool // aiType added
}

func (h *toolHandler) GetByNames(aiType ai.Type, names []tool.ToolName) []tool.Tool {
    allowed := ai.AllowedToolNames(aiType)
    hasAll := containsName(names, tool.ToolNameAll)
    var result []tool.Tool
    for _, t := range toolDefinitions {
        if !allowed[t.Name] { continue }
        if hasAll || containsName(names, t.Name) { result = append(result, t) }
    }
    return result
}
```

`toolNameResolver` (the local structural interface in `mcp_tool.go`) gains the
`aiType` parameter to match; its sole caller, `resolveTools`, already has `a`
in scope (`h.toolNameResolver.GetByNames(a.Type, a.ToolNames)`). This is a
signature change with exactly one production call site plus its mock and
tests -- small, mechanical.

## 4. B24: honour `isError`

```go
// pkg/mcptoolhandler/client.go, CallTool
func (h *mcpToolHandler) CallTool(ctx context.Context, serverID uuid.UUID, toolName string, argumentsJSON string) (string, bool, error) {
    // ... existing request/parse logic ...
    return text, result.IsError, nil // new second return: isError
}
```

```go
// pkg/aicallhandler/mcp_tool.go, toolHandleMcpCall
msg, isError, err := h.mcptoolHandler.CallTool(ctx, ref.ServerID, ref.ToolName, tc.Function.Arguments)
if err != nil {
    // unchanged
}
if isError {
    log.Warnf("Mcp tool call reported an error result. mcp_server_id: %s, tool_name: %s", ref.ServerID, ref.ToolName)
    fillFailed(res, errMcpToolCallFailed(capErrText(msg, 200))) // same 200-char cap already used for transport errors -- never forward the remote server's raw text uncapped
    return res
}
fillSuccess(res, "mcp_tool", ref.ServerID.String(), msg)
```

`CallTool`'s signature change touches one production caller
(`toolHandleMcpCall`) today; **verify this at implementation time with the
same rigor §2.3 applies to `resolveMcpOnly`'s caller graph** (a deferred
consumer such as B8's future auditor exposure work, or anything else calling
`CallTool` directly, would silently break on this signature change if missed
-- grep for all callers before assuming the arity change is contained to one
call site). Update its tests/mock accordingly.

## 5. B25: cancellation bound

`toolHandleMcpCall` is invoked from `listenhandler` under whatever context the
RPC consumer loop gives it. **Clarification (round 2 finding resolved):**
`mcp_tool_call_timeout_seconds` is already read inside `CallTool` itself
(`pkg/mcptoolhandler/client.go`, per `internal/config/main.go:148`'s comment
"Bounds one whole MCP call ... its handshake, the method and one
re-initialisation") -- this bounds the TRANSPORT round-trip to the remote MCP
server, which is the dominant cost of a `tools/call`. It does NOT bound
whatever `toolHandleMcpCall` does with the result afterward (response
parsing, `fillSuccess`/`fillFailed`, the DB/cache writes those trigger), nor
does it bound the RPC consumer path itself if `listenhandler/main.go:279`'s
`context.Background()` means the request has no deadline until `CallTool`
imposes its own. Verify at implementation time whether that outer context is
already tied to the AIcall's lifetime or is unbounded as the v2 analysis
states for the *listing* path. If dispatch runs the same way (unbounded
outer context, `CallTool`'s internal timeout bounding only the transport
leg): wrap the WHOLE `toolHandleMcpCall` call in
`context.WithTimeout(ctx, h.config.McpToolCallTimeoutSeconds())` so post-transport
work cannot run unbounded even though the transport leg already is. This is
additive, not redundant, with `CallTool`'s internal bound -- it closes the gap
between "the remote call itself is bounded" and "everything this process does
around that call is also bounded." This is a belt-and-suspenders bound, not a
listen-for-hangup mechanism -- true cancellation-on-hangup would require
threading the AIcall's termination signal into this call, which is out of
scope; capping the duration is what "must not run forever" requires for this
PR.

## 6. B26: `clearListenState` metadata-clobber check

Read `pkg/aicallhandler/listen.go`'s `clearListenState` (or equivalent) in
full at implementation time. If it writes `c.Metadata` wholesale from an
in-memory snapshot taken before a concurrent `MetaKeyMcpToolMap` write, apply
the same re-read-then-merge pattern `refreshMcpToolMap` and
`writeInsightSessionMetadata` already use. If it already re-reads (verify,
do not assume), this item is a no-op confirmation, documented as such in the
PR body rather than silently dropped.

## 7. B22: webhook projection

`models/aicall/webhook.go`'s `ConvertWebhookMessage` copies `Metadata` as a
whole field. Add an explicit `mcp_tool_map` key removal and, when the map is
non-empty, a derived `mcp_tool_status` summary (documented shape: e.g.
`{"servers": 2, "tools": 5}` -- no server UUIDs, no tool names, no schemas).
Update `ai_struct_aicall.rst` to match. This is a **breaking payload change**;
state so explicitly in the PR body, with the same "blast radius nil, no AI
has both MCP tools whitelisted and Insight excluded from getting the raw map
in the first place" framing PR B1 used for its own breaking changes, since
this PR is what first makes a non-empty map possible in production traffic.

## 8. B27: rollback switch

`internal/config/main.go` (4-edit pattern: struct field, flag registration,
env binding, default) gains `mcpToolExposureEnabled bool`, default `true`.
**Single check location (round 1 Medium finding resolved):** the flag is
checked exactly once, as the FIRST line of `ResolveMcpTools` (§2.3, ai-manager
side) -- before any DB read, AI resolution, or discovery. It is never checked
inside `resolveMcpOnly`/`discoverMcpTools` themselves (those stay unaware of
the flag, callable directly by tests without needing to stub config) and
never checked in `runner.go`/pipecat (which only sees the RPC's return value,
`(nil, nil)` when disabled, indistinguishable from "this AI has no MCP tools"
-- correct, since disabling the feature should look exactly like no MCP tools
existing, not like an error). This is the only kill switch; nothing else in
the codebase can disable the feature once shipped.

## 9. B17/B18: metrics

**Concrete label fix, confirmed against the code (round 1 Medium finding
resolved).** `pkg/aicallhandler/tool.go:123`:
`promAIcallToolExecuteTotal.WithLabelValues(string(tool.Function.Name)).Inc()`
labels by the raw LLM-called function name -- for a built-in tool this is one
of a fixed ~20-member enum (bounded), but for an MCP tool it is
`mcp_<8hex>_<remote_tool_name>` (§2.2's namespacing), where `<remote_tool_name>`
is chosen by the customer's own MCP server and unbounded. Fix: split the
label into a bounded `tool_class` (`"builtin"` or `"mcp"`, two values) and,
for the builtin case only, keep the existing concrete `tool_name` label;
for the MCP case, do NOT label by the resolved name at all -- log it instead
(Prometheus label, not log line, is the cardinality-sensitive surface). Concretely:

```go
if strings.HasPrefix(string(tool.Function.Name), mcpToolNamePrefix) {
    promAIcallToolExecuteTotal.WithLabelValues("mcp").Inc()
} else {
    promAIcallToolExecuteTotal.WithLabelValues(string(tool.Function.Name)).Inc()
}
```

This requires either widening `promAIcallToolExecuteTotal`'s existing label
set (adding a value, not a new label dimension -- no metric re-registration
needed, since the vector already has one label) or, if simpler given the
existing metric's declared name, leaving the existing metric untouched for
built-ins and adding it is own single-value-labeled counter for the MCP case.
Decide the exact shape at implementation time against the existing
`prometheus.NewCounterVec` declaration for this metric; both are equivalent
in effect, and this doc's purpose is to fix cardinality, not mandate the
Prometheus API shape.

- Add `mcp_tool_advertised_total` (counter, incremented once per
  `ResolveMcpTools` resolution that returns a non-empty slice) and
  `mcp_tool_call_outcome_total{outcome}` (`success`/`error`/`failed`, where
  `error` is B24's `isError:true` case and `failed` is a transport/dispatch
  failure) so this feature is observable from day one rather than requiring a
  customer email to notice it is silently broken again. **§1's summary table
  entry is corrected to match this name** (round 2 finding resolved -- the
  prior "trust this section, not that one" footnote is removed; there is now
  one metric name in the document, `mcp_tool_call_outcome_total`).

**`metricsMcpToolListFallbackTotal` (bin-pipecat-manager side, round 3
finding resolved).** §2.2's `runner.go` code sample increments
`metricsMcpToolListFallbackTotal` whenever the `AIV1AIcallToolList` callback
fails and the session falls back to built-ins only. This is declared here as
a bin-pipecat-manager-owned Prometheus counter with no labels (the failure
reason is already captured in the accompanying log line, not the metric --
matching this section's own cardinality discipline). It belongs in
bin-pipecat-manager's metrics registration alongside its existing
`metricsToolResolveFallbackTotal` (the sibling counter for the pre-existing
AI-resolution-failure branch, `runner.go`), not in ai-manager's B17/B18
metrics (which are entirely ai-manager-side). §13's bin-pipecat-manager row
is updated to name it explicitly.

## 10. Domain model changes

- `models/tool/main.go`: add `omitempty` to `Tool.Parameters`'s json tag
  (B2b).
- `bin-common-handler/pkg/requesthandler/ai_aicalls.go` +
  `pkg/requesthandler/main.go` (interface) + `mock_main.go`: new
  `AIV1AIcallToolList` method.
- `bin-ai-manager/pkg/aicallhandler/main.go`: `ResolveMcpTools` added to the
  `AIcallHandler` interface; mocks regenerated. (The old unexported
  `resolveTools` is deleted; see §2.3's naming table.)
- `bin-ai-manager/pkg/toolhandler/main.go`: `GetByNames` gains `aiType
  ai.Type`; `toolNameResolver` (local interface in `mcp_tool.go`) updated to
  match; mocks regenerated.
- `bin-ai-manager/pkg/mcptoolhandler/client.go` +
  `pkg/mcptoolhandler/main.go` (interface): `CallTool` gains a second return
  value `isError bool`; mocks regenerated.
- `internal/config/main.go`: new `mcp_tool_exposure_enabled` flag.
- `models/aicall/webhook.go`: `mcp_tool_map` removed from the projection,
  `mcp_tool_status` added.

## 11. Test matrix

- `ResolveMcpTools`: normal AI with MCP tools (**MCP-only** list returned --
  never the built-in set, per §2.3's round-1 correction; assert the returned
  slice contains no built-in tool name -- this specific assertion is what
  would have caught the round-1 duplicate-built-ins bug had it shipped, so it
  is not optional), map persisted via `persistToolMap`; Insight AI (`nil, nil`,
  discovery not even attempted -- assert `discoverMcpTools` is not reached,
  e.g. via a mock that would fail the test if called); team AIcall (`nil, nil`,
  no AI resolution attempted); config disabled (`nil, nil`, no DB/RPC calls at
  all); AI resolution failure (error propagated, caller's fail-open is
  exercised at the `runner.go` level, not here).
- `toolHandleMcpCall`: `isError:true` fails the call with capped text,
  `isError:false` succeeds as today, transport error unchanged.
- `pkg/toolhandler.GetByNames` (ai-manager): Insight AI with `tool_names:["all"]`
  no longer returns Normal-only tools; Normal AI unaffected; unknown type
  denies all (matches `AllowedToolNames`'s existing default).
- `runner.go` (pipecat, Go side): callback success appends MCP tools; callback
  RPC error falls back to built-ins only (not empty); AI-resolution failure
  still zeroes tools entirely (regression guard for the existing branch).
- Webhook: a non-empty `mcp_tool_map` never appears in `ConvertWebhookMessage`'s
  output; `mcp_tool_status` reflects server/tool counts.
- Config: `mcp_tool_exposure_enabled=false` makes `ResolveMcpTools` return
  `(nil, nil)` without calling `resolveMcpOnly` or touching the DB.
- One live/manual check per L4 in the v2 analysis: register the reference MCP
  server, whitelist it on a Normal AI, place one call to a virtual/internal
  number, confirm the model can invoke the tool and the result is not
  mislabeled on an injected `isError:true` response.

## 12. Open questions carried forward (not blocking this PR)

**Risk requiring explicit CEO/CTO sign-off before merge (round 1 High finding,
not previously called out this plainly):** the v2 analysis measured a
worst-case ~132,000 tokens per conversational turn for 64 MCP tools with
Korean-heavy descriptions at the existing per-tool cap. B7 (a token-based
description cap) is deferred to a follow-up PR (§1) on the reasoning that
real customer-registered MCP servers today have far fewer tools in practice.
That reasoning is an assumption about customer behavior, not a control: any
customer (or anyone able to register an MCP server against an AI they
control) can construct this pathological case the moment B12 ships, before
B3/B7 exists to cap it. **This PR ships with no upper bound on
per-session prompt token cost from MCP tool descriptions beyond the existing
`mcpMaxToolsPerResolution=256` count cap and the untyped per-tool description
length (not token count).** The only mitigation available on day one is B27's
global kill switch (an all-or-nothing rollback, not a per-customer or
per-severity control). Recommend one of: (a) accept this as a known, bounded-
blast-radius risk (worst case is an expensive/slow LLM call for the customer
who configured it, not a platform-wide outage, since `mcpMaxToolsPerResolution`
already caps the pathological case's tool COUNT even if not its token cost),
explicitly logged as an accepted risk in the PR body, or (b) pull a minimal
version of B7 (a conservative flat per-tool description CHARACTER cap well
under today's implicit limit, e.g. 512 characters, cheap to add, not the full
token-aware design) into this PR as a stopgap. Recommend (a) with the PR body
calling it out explicitly, since (b) risks scope creep into B7's real design
work and the blast radius is per-customer, not platform-wide -- but this is a
CEO/CTO call, not an engineering default.

**Second day-one risk, no aggregate rate limit on repeated discovery (round 2
High finding):** §2.3's `persistToolMap` concurrency note observes that
`ResolveMcpTools` can run more than once for the same AIcall in quick
succession (a Python-side reconnect, or a retry after the caller's own RPC
timeout). Each such run performs a live `tools/list` round-trip to the
customer's own MCP server(s), bounded per-attempt by §2.3a's session-start
budget, but with **no aggregate rate limit** across repeated attempts for one
flapping session. A pipecat session that repeatedly reconnects (for reasons
unrelated to MCP -- network flakiness, a crashing Python process, etc.) can
hammer a customer's MCP server with repeated discovery calls with no backoff.
This is the same shape of gap as the token-cost risk above: the only
available mitigation on day one is B27's global kill switch, not a
per-session or per-customer throttle. Recommend accepting this as a known
risk alongside the token-cost one (same reasoning: blast radius is the
customer's own configured server, not platform-wide), explicitly logged in
the PR body rather than silently left unaddressed.

| # | Question | Status |
|---|---|---|
| O1 | Cache location/invalidation (deferred to B3/B14) | OPEN, deferred |
| O2 | Final token-based description cap numbers (deferred to B3/B7) | OPEN, deferred -- see the risk callout above |
| O6 | Per-AI tool count cap interaction with a future per-server cache | OPEN, deferred (no cache yet in this PR, so moot until B14) |
| O7 | square-admin picker surfacing the Insight/over-cap error reason | OPEN, PR C (frontend), out of this backend PR's scope |
| O8 | Does B15's probe result persist (new schema)? | OPEN, deferred with B15 |
| O9 | Bound on the transport RPC's response payload size | Partially answered: `mcpMaxToolsPerResolution=256` and per-tool schema caps (`mcpMaxToolSchemaBytes`, `mcpToolSchemaBudgetBytes`) already bound it from the discovery side (merged in B1); no additional cap needed for the RPC itself since it carries exactly what `resolveTools` already capped |
| O10 | Is `ai_struct_aicall.rst`'s `mcp_tool_map` field already relied on by an external integrator despite always being empty in production today? | OPEN -- round 1 Medium finding. No way to answer definitively from the codebase alone (an external integrator's code is not visible); recommend treating the RST as the authoritative public contract per existing CLAUDE.md convention ("RST docs are the primary user-facing documentation ... single source of truth"), and since it already documents `mcp_tool_map` as a real field, its removal should go through the same breaking-change communication B1 used (PR body states the field is removed and why), not be assumed silently safe purely because production traffic never populated it |
| O11 | Does the new `AIV1AIcallToolList` RPC need an explicit cross-customer ownership check on `aicallID`, or is it safe under the existing internal-RPC trust model? | OPEN -- round 1 Medium finding. `h.Get(ctx, aicallID)` is described as "ownership-neutral" in §2.3, consistent with how every other existing `AIV1Aicall*` RPC in this codebase behaves (the RPC layer trusts its caller, which is always another VoIPBin service on an internal queue, never a customer-facing edge) -- this RPC introduces no NEW trust boundary beyond what `AIV1AIcallToolExecute` (the existing, structurally identical dispatch RPC) already crosses. Recorded as an open question rather than closed outright because this RPC is the first one whose response can contain a customer's MCP server-derived tool schemas, which is new *content* even though not a new *trust boundary*; if this reasoning is accepted, no code change is needed, but it should be stated in the PR body rather than left implicit |


## 13. Affected services

| Service | Change |
|---|---|
| bin-ai-manager | `ResolveMcpTools`, `AIV1AIcallToolList` handler, B4/B10/B11/B24/B25/B26 fixes, config flag, webhook projection, metrics |
| bin-common-handler | New RPC client method + interface + mock |
| bin-pipecat-manager | `runner.go` callback wiring, fail-open-to-built-ins on callback failure, new `metricsMcpToolListFallbackTotal` counter (§9) |
| bin-api-manager | RST/OpenAPI: correct the "not yet available" caveat added by
`NOJIRA-Caveat-mcp-tool-use-not-yet-available` back to accurate present-tense
text, scoped precisely (Normal-type AIs, single-AI sessions, the existing
caps) |

## 14. Implementation order

1. Domain/interface signature changes first (B4's `GetByNames`, B24's
   `CallTool`, `omitempty`) -- small, mechanical, each with its own tests, so
   later steps build on a stable interface.
2. B10/B11 residual gates in `discoverMcpTools` and the `ResolveMcpTools`
   team/Insight short-circuit.
3. B25/B26 hardening at the existing dispatch/listen-clear sites.
4. B27 config flag (must exist before B12 is wired, so B12 is disable-able
   from the moment it merges).
5. B12: `ResolveMcpTools`, the RPC client method, listenhandler route, mocks.
6. `runner.go` wiring in bin-pipecat-manager (the actual "turn it on").
7. B17/B18 metrics.
8. B22 webhook projection + RST.
9. Full test matrix, then the L4 manual live check, recorded in the PR body.

---

This is one PR. It closes every correctness gap the v2 analysis identified as
activating "the moment a consumer exists" and adds the consumer (B12) in the
same change, so no window opens between "advertisement exists" and "the known
bugs around it are fixed." Everything deferred (§1's second table) is real
work but does not create an incorrect behavior that this PR's own change makes
newly reachable.

## 15. Addendum: provider-safe MCP tool schemas (post-deploy finding)

Status: design addendum, revised after design review round 1 and amended after code
review rounds 1 to 4 of `0aa260c84` (R4 dedupe, R7a, R10a, `maxWork` in R12). Source analysis:
`2026-09-29-mcp-tool-schema-provider-compat-analysis.md` (revision 6, review loop
closed; "the analysis" below, section numbers prefixed "an."). CEO decision: A (Go
allowlist normalization in ai-manager) plus E (per-tool Gemini validation in the
Python runner) plus the Go-side RTVI `error` frame log raised to WARN, all in this PR.
B (pipecat upgrade) is a separate later track and out of scope here. Code references
are to the branch head at `4e5015cc9`.

### 15.1 Problem, in one paragraph

`resolveMcpOnly` (`bin-ai-manager/pkg/aicallhandler/mcp_tool.go:101`) passes each MCP
tool's raw `inputSchema`, checked only for size and JSON-object shape by
`decodeToolSchema` (`mcp_tool.go:508`), straight into `tool.Tool.Parameters`
(`mcp_tool.go:120`). On the Gemini path the pinned runner (pipecat 1.4.0, google-genai
1.75.0) validates the whole `GenerateContentConfig` at once, so one tool carrying an
`x-mcp-header` key or a list-valued `type` fails every turn of the session before
any network request, built-ins included (an.2, an.3). The D13 property "MCP half is
best-effort, built-ins always survive" (§2.1) held at the RPC layer only. This
section extends D13 to the provider layer.

### 15.2 A: where normalization runs (bin-ai-manager)

**Call site.** Exactly one: inside `resolveMcpOnly`'s loop, immediately after
`decodeToolSchema` returns `schemaOK` (`mcp_tool.go:101-114`) and before the
`seenNames`/`tool.Tool` append (`mcp_tool.go:116-123`). This is the only place the
full raw schema, including top-level `$defs`/`definitions`, is in memory: pipecat's
`run.py _openai_tools_to_standard` (`bin-pipecat-manager/scripts/pipecat/run.py:414-451`)
later keeps only `properties` and `required`, so `$ref` targets are lost past this
point (an.3, an.5 option C).

```go
params, why := decodeToolSchema(d.inputSchema, &schemaBudget)
if why != schemaOK { /* unchanged skip accounting */ }

norm, rep := mcpschema.Normalize(params, mcpMaxToolSchemaBytes)   // NEW
logSchemaReport(log, d, rep)                    // NEW, 15.4
if rep.ToolDropped {
    continue                                    // tool NOT advertised, NOT in toolMap
}
// outBudget := mcpToolSchemaBudgetBytes is declared next to schemaBudget (NEW, R12)
if rep.OutBytes > outBudget {                   // NEW
    /* same skip accounting as schemaOverBudget: s.overBudget++ */
    continue
}
outBudget -= rep.OutBytes                       // NEW; schemaBudget is untouched
// ... unchanged: seenNames, append tool.Tool{Parameters: norm}, toolMap[d.name] = d.ref
```

A dropped tool is skipped before `toolMap[d.name] = d.ref`, so it is neither advertised
nor dispatchable, consistent with D15 (the map is built from exactly the advertised
list, §2.3). A tool whose output charge does not fit what is left of `outBudget`
is counted as `overBudget` and reported by the existing per-server line
(`Skipped mcp tools whose input schema could not be used. ... over_shared_budget`,
`mcp_tool.go:126-128`); no new log line for it. The budget accounting is in R12. A nil
`params` (absent or `null` schema, `decodeToolSchema` returns
`nil, schemaOK`, `mcp_tool.go:510-511`) is passed through unchanged as nil: the
`omitempty` tag (`models/tool/main.go:108`) already keeps it off the wire, and a
missing `parameters` is the no-argument shape Gemini accepts (an.5 notes top-level
`properties: {}` from `stop_flow`/`stop_service` in the control request).

**Paths that do not need it.**
- `resolveMcpToolMap` (`mcp_tool.go:45`) calls `discoverMcpTools(ctx, a, false)`,
  so no schema is ever read (`mcp_tool.go:458-461`); its callers
  (`start.go:1193`, `insight_session.go:225`, `refreshMcpToolMap` at `mcp_tool.go:766`)
  store names only and advertise nothing.
- The persisted `mcp_tool_map` (`persistToolMap`, `mcp_tool.go:207`) stores
  `McpToolRef{ServerID, ToolName}` only; no schema is cached anywhere, so there is no
  second cached path to normalize.
- Dispatch (`toolHandleMcpCall`, `CallTool` at `mcp_tool.go:599`) forwards
  `tc.Function.Arguments` verbatim. **Call arguments are never rewritten.** The MCP
  server's own validation plus B24 (`isError`) handles any constraint the
  normalized schema no longer carries.

One known gap, pre-existing at HEAD and widened by A: the name-only paths store the
name of every discovered tool, including tools `decodeToolSchema` would skip today
(too large or malformed), so an `mcp_tool_map` entry can name a tool that is not
advertised, and an LLM that invents that exact namespaced name could dispatch it. A
adds its dropped tools to that set. Two variants:
- (a) A tool that A keeps but E drops (15.5) stays in `toolMap` by design (A cannot
  see E's verdict), and Python still registers its handler by name (`tools.py:101`).
- (b) `ResolveMcpTools` normally replaces the name-only map with the advertised list
  when pipecat resolves tools, but `persistToolMap` failure is only logged and not
  fatal (`mcp_tool.go:188-193`); then the name-only map written at session start
  (`start.go:1193`) stays for that session.

In every variant the tool is never advertised (the LLM is never shown it) and the
server still validates the arguments. Accepted; see R-4 in 15.10.

**Package.** New pure package `bin-ai-manager/pkg/mcpschema` (files `main.go`,
`normalize.go`, `normalize_test.go`, `testdata/`). Reasons:
- It is a pure function over `map[string]any` with no handler, DB, or config
  dependency, the same shape as the existing pure helper package
  `bin-ai-manager/pkg/actioncatalog` (no handler struct, table tests in
  `main_test.go`). `docs/conventions/package-structure.md` §1.1 lists `<domain>handler`
  packages for stateful logic; it has no rule forbidding a helper package and
  `actioncatalog` is the in-service precedent.
- `aicallhandler` is already large (`mcp_tool.go` alone is 782 lines) and its tests
  need the full gomock handler setup; a separate package keeps the normalizer's tests
  fast, table-driven, and free of mocks.
- No import cycle: `mcpschema` imports only the standard library; `aicallhandler`
  imports it.

API (exported, the only surface):

```go
package mcpschema

// Report describes what Normalize changed. Paths are JSON-pointer-like
// ("/properties/owner/x-mcp-header"). It never contains schema values.
type Report struct {
    ToolDropped     bool
    DropReason      string   // one of the Reason* constants, "" when kept
    DropPath        string   // where the fatal unusable subschema was found
    DroppedProps    []string // first maxReportedPaths (8) paths of optional properties removed (cascade)
    DroppedPropsN   int      // total count of optional properties removed (may exceed len(DroppedProps))
    DroppedKeys     int      // count of non-allowlisted keys removed
    Rewrites        int      // oneOf, const, allOf, $ref, list type, type inference
    OutBytes        int      // output charge (R12); 0 for a nil schema
}

// Normalize returns a provider-neutral copy of schema (the decoded top-level
// inputSchema object). It never mutates its input. A nil schema returns (nil,
// Report{}). maxOutBytes is the per-tool output cap (R12); the caller passes
// mcpMaxToolSchemaBytes, so mcpschema needs no import from aicallhandler.
func Normalize(schema map[string]any, maxOutBytes int) (map[string]any, Report)
```

The input is not mutated (build a new map at every level) so that a future caller
holding the raw map (for example a log dump) is not surprised, and so tests can
compare input and output.

### 15.3 A: the ruleset (implementable spec)

Terms. A *subschema* is any JSON value in a schema position: the root, each value of
`properties`, `items`, each member of `anyOf`/`oneOf`/`allOf`, each `$defs` target.
Processing is recursive, bottom-up for the cascade, with a depth counter starting at
0 at the root. Output is built fresh; any key not listed in R1 never appears in it.

**R0. Root.** The root must be a JSON object (guaranteed by `decodeToolSchema`). After
normalization the root must have `type: "object"`. At the root, an absent `type` is
always treated as `object` (before R6, which would otherwise find a bare `{}` or
`{"description": ...}` typeless and drop no-argument tools that work today because
`_openai_tools_to_standard` defaults `properties` to `{}`); any other root type drops
the tool with `ReasonRootNotObject` (this includes a root list type such as
`["object","null"]`; exotic, and accepted as a drop). At the root, R5 (`$ref` and
single-member `allOf` merge) runs before the absent-type default. A root-level `anyOf`/`oneOf` is dropped
(counted in `DroppedKeys`), because the runner keeps only `properties` and
`required` of the root anyway (`run.py` `_openai_tools_to_standard`). The root always emits
`properties` (possibly `{}`) and `required` only when non-empty. The top-level empty
`properties: {}` is kept (proven by `stop_flow`/`stop_service` in the 08:02 control,
an.5). Root-level `$defs`, `definitions`, `$schema`, `$id`, `title`,
`additionalProperties` are consumed or dropped like any other key.

**R1. Keep list.** Only these keys are emitted: `type`, `description`, `properties`,
`required`, `items`, `enum`, `anyOf`, `format`, `minimum`, `maximum`, `minItems`,
`maxItems`. Everything else is dropped and counted in `Report.DroppedKeys`, including
`x-*`, `$schema`, `$id`, `$comment`, `examples`, `title`, `default`,
`additionalProperties`, `patternProperties`, `exclusiveMinimum`, `exclusiveMaximum`,
`multipleOf`, `pattern`, `minLength`, `maxLength`, `nullable`, `readOnly`,
`writeOnly`, `deprecated`, `not`, `if`/`then`/`else`, `dependentRequired`,
`prefixItems`, `contains`, `uniqueItems`. `oneOf`, `allOf`, `const`, `$ref` are
consumed by the conversions below and never emitted.

**R2. Value shapes.** A kept key with a wrong value shape is dropped (the key only,
not the subschema), except where R7 says the subschema becomes unusable:
- `type`: a string in {`string`, `number`, `integer`, `boolean`, `object`, `array`,
  `null`}, or a list handled by R4. Any other string or value makes the subschema
  unusable (`ReasonBadType`).
- `description`: string, else dropped. Not truncated here (description caps are B7,
  deferred, §1).
- `properties`: object whose values are subschemas; a non-object `properties` is
  treated as absent.
- `required`: list of strings; non-strings removed; then pruned by R8.
- `items`: must be a single object subschema. Tuple `items: [...]` and boolean
  `items` make the array unusable.
- Boolean subschemas (`true`/`false`) anywhere are unusable.
- `minimum`/`maximum`: JSON numbers (Go `float64` after decode) kept only on
  `number`/`integer`; `minItems`/`maxItems`: non-negative integral numbers kept only
  on `array`. Anything else dropped.

**R3. Server-side-conservative rules** (the genai client accepts these, the Gemini
server is reported to reject them, an.5A sources):
- `format` kept only for these (type, format) pairs: string: `date-time`, `enum`;
  integer: `int32`, `int64`; number: `float`, `double`. Otherwise dropped (for
  example `uri`, `email`, `uuid`, `date`). `format: enum` is kept only if an `enum`
  is also emitted on the same subschema.
- `enum` kept only when the emitted `type` is `string` and every member is a
  string, and the list is non-empty. Otherwise dropped (the subschema stays usable
  with its type). Duplicate members are kept as is (not deduplicated; order
  preserved).

**R4. List `type`.** `type: [t1, t2, ...]` becomes `anyOf: [{type: t1, ...}, ...]`,
one member per listed type, each member carrying the type-appropriate sibling
constraints of the original (`enum`, `format`, `items`, `properties`, `required`,
bounds), filtered by R2/R3 for that member's type. `description` moves to the
parent, not the members. A `"null"` entry becomes a bare `{type: "null"}` member
(R9). A single-element list is treated as the plain string. Duplicate entries are
collapsed (one member per distinct type, first-occurrence order). An empty list is
unusable. This is the incident shape
`issue_fields.items.properties.value.type = ["string","number","boolean"]` (an.2).

**R5. Conversions.**
- `oneOf` becomes `anyOf` (same members). If both are present, the members are
  concatenated (`anyOf` first).
- `const`: a string becomes `enum: [c]` with `type: string` (if a `type` other than
  string is present, the `const` is dropped instead). A non-string `const` is dropped.
- `allOf` with exactly one member: the member is merged into the parent, parent keys
  winning on conflict, then the result is normalized (this is the Pydantic
  `allOf: [{$ref}]` wrapper). `allOf` with 0 members is ignored. `allOf` with 2 or
  more members is unusable (`ReasonAllOf`).
- `$ref`: only local refs `#/$defs/<name>` and `#/definitions/<name>` (root-level
  maps, JSON pointer `~0`/`~1` unescaped). The target is merged under the referencing
  subschema's siblings (siblings win), then normalized. Any other ref form
  (remote, `#`, nested pointer), a missing target, or a cycle is unusable
  (`ReasonRef`).
  - Cycle detection: a per-path stack of ref names being expanded; re-entering a name
    already on the stack is a cycle.
  - Bounds: at most **8** nested `$ref` expansions on any one path
    (`maxRefDepth = 8`) and at most **256** `$ref` expansions per tool in total
    (`maxRefExpansions = 256`, stops exponential fan-out from a DAG of shared defs).
    Exceeding either is unusable at that subschema. These bound the work, not the
    output size: 256 expansions of one large `$defs` target still multiply the
    output (measured below in R12); the output cap in R12 is what bounds that.

**R6. Missing `type` inference** (after R5 has run on the subschema):
- `properties` present: `object`;
- else `items` present: `array`;
- else a non-empty `enum` of all strings: `string` (repairs the Zod `{enum: [...]}`
  shape);
- else `anyOf` present: no `type` emitted, the subschema is its `anyOf`;
- else unusable (`ReasonTypeless`, "any JSON value", for example GitHub MCP's
  `projects_write.updated_field.*.value`).

**R7. Unusable-subschema cascade (local semantics, an.5A).** A subschema is
*unusable* when R2, R4, R5 or R6 say so, when it is an `array` whose `items` is
absent or unusable, or when it is an `anyOf` left with no usable member. Handling:
- an unusable `anyOf` member is removed from its `anyOf`; an `anyOf` reduced to
  exactly one member is kept as a one-member `anyOf` (not flattened; flattening is a
  possible later refinement, not needed for acceptance);
- an unusable `items` makes its array unusable;
- an unusable property of an object: if the object's `required` does not list it,
  the property is removed, `Report.DroppedPropsN` is incremented, and its path is
  added to `Report.DroppedProps` only while fewer than `maxReportedPaths` (8) paths
  are held (bounded memory; 15.4 logs at most 3); if it is
  required, the enclosing object becomes unusable;
- the tool is dropped (`Report.ToolDropped`) only when the root becomes unusable.
  Required-ness therefore propagates only up an unbroken `required` chain.
- An `anyOf` member that is `{type: "null"}` is usable but does not count as the
  "usable member" that keeps the `anyOf` alive: an `anyOf` whose only surviving
  member is `{type: null}` is unusable. (Decision taken here; the analysis does not
  settle it. Reason: a property that can only be null carries no argument, and a
  null-only `anyOf` is not in the probe-verified set.)

**R7a. Constraint-only `anyOf`/`oneOf` next to a type** (added after code review
round 1). When a subschema has a usable `type` (string or list) and at least one of
its `anyOf`/`oneOf` members, after its own `$ref` and single-member `allOf` are
resolved, has none of `type`, `properties`, `items`, `enum`, `const` (a member that
is itself only an `anyOf`/`oneOf` is followed into its members, and counts when any
of those does; a member that cannot be resolved does not count and is left to the
normal evaluation), the combinator only refines the parent. The check is a
look-ahead (amended after code review round 3): it leaves `Report` unchanged, is
charged to `maxWork`, and counts its `$ref` expansions against its own budget of
`maxRefExpansions` (256), separate from the build's; once that budget is spent a
member behind a further `$ref` is not a refinement. The budget is per subschema, not
per tool (amended after code review round 4), so the verdict for one property never
depends on how much another property's look-ahead spent (a per-tool budget let an
optional property early in sort order turn a later required property's refinement
into a dropped tool); `maxWork` bounds the total. Without the separate budget a
1.2 KB `$defs` DAG (7 levels, 6 `$ref` members each) behind one `anyOf` member cost
about 85 ms and 97 MB and dropped a tool the round-2 code kept
(for example `anyOf: [{required: [a]}, {required: [b]}]` "at least one of", or
`oneOf: [{format: date}, {format: date-time}]`). The whole combinator is removed
(counted once in `DroppedKeys`) and never evaluated, so it cannot make a usable
subschema unusable. Without this rule such a member is typeless under R6, the `anyOf`
has no usable member, and a required property drops a tool that OpenAI and Grok
accept today. A typeless subschema's `anyOf` is not a refinement and is evaluated
as usual.

**R8. `required` pruning.** After properties are processed, `required` keeps only
names present in the emitted `properties`, deduplicated, original order. Emitted
only when non-empty.

**R9. Nullability.** Expressed only as an `anyOf` member `{type: "null"}`
(probe-verified on genai 1.75 and 2.25, an.5A). `nullable: true` (OpenAPI) is
dropped by R1, never produced.

**R10. Objects.**
- Root: always `properties` emitted (R0).
- Nested `object` with absent or empty `properties`, or whose properties all
  cascaded away: normalized to exactly `{type: "object", description?}` with no
  `properties` key and no `required` key. This is the free-form shape already sent by
  built-ins (`set_variables.variables`, `create_call.actions[].option`) and proven on
  gemini-2.5-flash in the 08:02 control (an.5A). Explicit nested `properties: {}` is
  stripped to this shape.
- A nested `object` without usable `properties` that also carries `anyOf`/`oneOf`
  (GitHub MCP: `projects_write` `/properties/updated_field`,
  `projects_write` `/properties/items/items`,
  `custom_properties_write` `/properties/properties/items`): the `anyOf` is evaluated first
  under R7 (after R5 conversion). If it has no usable member, the object is unusable
  (cascade). Otherwise the `anyOf` is discarded and the result is the free-form shape
  above. Order matters: stripping first would silently keep `updated_field`.
  Expected results are pinned in 15.7 item 4.
- **R10a. No output carries `type` and `anyOf` together** (added after code review
  round 1). For any typed subschema (not only objects) that also carries a shaped
  `anyOf`/`oneOf`, the `anyOf` is evaluated as a usability gate under R7: with no
  usable member the subschema is unusable; otherwise the `anyOf` is discarded
  (counted in `DroppedKeys`) and only the type (with its properties, items, enum,
  bounds) is emitted. Exception (added after code review round 2), the
  documented-enum shape: when the type is a scalar (`string`, `number`, `integer`,
  `boolean`), the parent carries no `enum`/`const`/`format`/`minimum`/`maximum` of
  its own, and every kept member has that same type (for example
  `{type: string, oneOf: [{const: a, description}, {const: b, description}]}`), the
  members are emitted as the `anyOf` and the parent `type` is dropped, which is the
  same output the schema gets without the parent `type`. Still no `type`+`anyOf`
  siblings. Amended after code review rounds 3 and 4: under a parent whose type (or
  list type) includes a scalar, a member that, after its own `$ref` and single-member
  `allOf` are resolved, has no `type`, `properties`, `items`, `anyOf` or `oneOf`,
  whose `const` (if any) is a value of that scalar type and whose `enum` (if any) has
  at least one value of it, takes the first such parent type before it is evaluated
  (JSON Schema applies the parent type to it), so
  `{type: integer, oneOf: [{const: 1, description}, ...]}`, the same members behind
  `$ref`s, and `{type: [integer, null], oneOf: [{const: 1}, ...]}` are usable instead
  of unusable; and the members replace the type only when every kept member carries its
  own `enum`, `format`, `minimum` or `maximum`. A non-string `const` is dropped (R5),
  so integer, number and boolean documented consts keep the bare type rather than
  emitting value-less duplicate members. Reason: `{type, anyOf}` siblings are accepted by the genai
  client but are in no probe-verified or live-verified payload, so the server-side
  acceptance on Gemini is unknown (R-1); the conservative shape is the type alone.
  The loss (for example a nullable variant, or per-variant `required`) is the same
  kind of constraint loss as R-3.
- A nested object that was made unusable by a required unusable property (R7) is
  unusable, not turned into a free-form object. (A free-form object would let the
  LLM omit the required field silently; failing the enclosing branch is the local
  semantics the analysis chose.)

**R11. Arrays.** `items` required (R7). An array without `items` is not re-typed
(for example as a JSON string), because arguments go raw to the MCP server (an.5A).

**R12. Limits.** Inputs are already bounded by `decodeToolSchema`
(`mcpMaxToolSchemaBytes = 64 KiB` per tool, `mcpToolSchemaBudgetBytes = 256 KiB` per
resolution, raw bytes, `mcp_tool.go:492-495`, charged at `mcp_tool.go:524`). Input
bounds alone do not bound the output: `$ref` inlining copies a target once per
reference. Measured (`~/.hermes/cache/scratch/pc/rr1_s15_ref_amplification.py`): a
64,429-byte input (one `$defs` string with a 56,000-byte description, 256 properties
each `$ref` to it) stays within every node and ref limit below and normalizes to
14.3 MB (x223); four such tools fit the raw budget, about 55 MiB in one RPC reply,
against a 40M memory limit (`bin-ai-manager/k8s/deployment.yml:81`). So:
- `maxDepth = 32` subschema nesting levels. One level is one step into `properties`
  values, `items`, or an `anyOf`/`oneOf` member. A `$ref` expansion does not add a
  level by itself: the target's content sits at the referencing subschema's depth,
  and nesting inside the target counts normally (ref-to-ref chains are bounded by
  `maxRefDepth`). Deeper subschemas are unusable. For scale: the deepest GitHub MCP
  tool normalizes to depth 4 (`issue_write`), the largest has 35 subschemas
  (`projects_write`), measured on `rv6_normalized.json`.
- `maxNodes = 4096` emitted subschemas per tool. Exceeding it drops the tool
  (`ReasonTooLarge`). With the production cap (`maxOutBytes` = 64 KiB) it cannot
  trip, because every emitted node is also a 32-byte visit and the output cap stops
  the build at 2,049 visits or fewer; it is a backstop for callers passing a larger
  `maxOutBytes` (tests do so to exercise it).
- `maxWork = 262,144` units of transient work per tool (added after code review
  round 1, extended after round 2). One unit is one map key copied by a
  `$ref`/`allOf` merge or scanned when counting dropped keys, or one list entry
  scanned in `required`, `anyOf`/`oneOf`, `enum` or a list `type`. List scans are
  counted because they run once per `$ref` use: a `$def` holding a 7,500-name
  `required` list, referenced 256 times, cost 350 ms and 394 MB per tool without
  it. Exceeding it drops the tool (`ReasonTooLarge`), like the
  output cap. Reason: resolution work is not emitted, so the output charge does not
  see it. A 2,000-level single-member `allOf` chain (one dropped sibling key per
  level) in one `$def`, referenced 256 times, is 49,756 raw bytes and without this
  cap costs about 23 s of CPU and 32 GB of allocation per resolution while the tool
  is KEPT (every merged key is dropped, the output is `{type: string}`). With the cap
  it is dropped in about 14 ms and 16 MB. The largest GitHub MCP tool needs 117 units
  (2,240x headroom).
- **Output charge.** While building, `Normalize` keeps a running charge, reported
  as `Report.OutBytes`:
  - 32 bytes per VISITED subschema, usable or not, emitted or later dropped
    (braces, key names, separators). Charging visits, not only emissions, is what
    makes the charge bound work and transient memory: without it, content the
    cascade drops is free, and a crafted 65,532-byte schema (one `$def` with 5,411
    typeless optional properties, referenced 256 times) stays under every other
    limit while producing 1,385,216 dropped paths (~64 MiB, probe
    `rr2_s15_droppedprops_amplification.py`). With the visit charge it trips the
    output cap after about 2,000 visits;
  - for each emitted string (property name, `type`, `description`, `format`,
    `enum` member, `required` member): its UTF-8 byte length plus 3;
  - 24 bytes per emitted number (`minimum`, `maximum`, `minItems`, `maxItems`).
  A subschema's own content is charged when it is emitted, before the cascade can
  remove it; a later removal does not refund (conservative). Together with the visit
  charge above, the running charge bounds both work and retained memory per tool.
  The charge is an estimate, not the marshalled size. On the 125 GitHub MCP tools
  (normalized by the oracle, emission-only charge) it is 108,903 bytes against
  102,364 bytes of compact JSON (ratio 0.94; per tool, JSON is at most 1.15x the
  charge; probe `~/.hermes/cache/scratch/pc/rr1_s15_output_charge.py`); the visit
  charge adds 32 bytes per dropped subschema, negligible for real schemas. JSON
  escaping is not counted: Go's `json.Marshal` escapes `<`, `>`, `&` and control
  characters as 6-byte `\uXXXX`, so a pathological schema can marshal to up to about
  6x the charge (about 384 KiB per tool, about 1.5 MiB per resolution). That is
  acceptable for memory (well under the 40M pod limit) and is stated here so no one
  reads the charge as an exact reply-size bound.
- **Per-tool output cap.** When the charge exceeds `maxOutBytes` (the caller passes
  `mcpMaxToolSchemaBytes`, 64 KiB, the same cap the raw input has), `Normalize`
  stops building and drops the tool (`ReasonTooLarge`, WARN per 15.4). The amplified
  shape above is dropped after about 64 KiB of work instead of 14.3 MB.
- **Per-resolution output budget.** `resolveMcpOnly` keeps a second budget,
  `outBudget := mcpToolSchemaBudgetBytes` (256 KiB), next to `schemaBudget`, and
  charges each kept tool's `Report.OutBytes` to it (15.2 code block). A tool that
  does not fit is skipped and counted as `overBudget` in the existing per-server log line. Decision: two pools,
  not one. `schemaBudget` stays exactly as today (raw bytes, charged inside
  `decodeToolSchema`, its tests unchanged); it bounds decode memory, which is
  transient. `outBudget` bounds what is retained and marshalled into the RPC reply.
  One shared pool charged with raw plus normalized would count most schemas twice
  (the GitHub MCP set alone is 105,983 raw plus 108,903 charged, 82% of 256 KiB)
  and would drop tools from a second normal-sized server that fits today. Worst case per resolution is
  therefore 256 KiB raw decoded plus 256 KiB (charged) retained, the same order as
  today's raw-only bound.
- Output can be somewhat larger than input without any `$ref` (R4 copies sibling
  constraints into each `anyOf` member, R0 adds an empty `properties`): 8 of the 125
  GitHub MCP tools grow slightly under the oracle, for example `issue_write` and
  `get_me`. The caps above are far from that.

**Resource-limit overflow drops the whole tool** (`maxNodes`, `maxWork`, output cap), unlike
the shape rules, which cascade (R7). Decision: a resource limit says nothing about
which part of the schema is at fault, so there is no local subschema to remove, and
a partial build cut at an arbitrary point would advertise a schema that depends on
traversal order. `maxDepth`, `maxRefDepth`, `maxRefExpansions` stay local (unusable
at that subschema, then R7), because each of them names a specific subschema.

**R13. Determinism.** Output maps have no ordering (JSON object order is irrelevant
to every provider). Everything with semantic order is preserved: `required`, `enum`,
`anyOf` member order. Property iteration inside the normalizer sorts keys, so the
capped `DroppedProps` (the first 8 in traversal order), `DroppedPropsN`, and the
first-failure `DropPath` are deterministic across runs without a post-sort.
Tests compare with `reflect.DeepEqual` on decoded maps, or on `json.Marshal` output
(Go sorts map keys on marshal), never on raw string concatenation.

### 15.4 A: observability

**Logs** (in `resolveMcpOnly`, same `log` entry with `func`, `ai_id`; fields added
per line: `mcp_server_id`, `tool_name` (the namespaced name, capped via
`capErrText(..., 80)` like `sampleToolNames`)):
- Tool dropped: `Warnf("Dropped an mcp tool whose input schema cannot be made provider-safe. mcp_server_id: %s, tool_name: %s, reason: %s, path: %s", ...)`.
  WARN, one line per dropped tool. `reason` is the fixed `Reason*` constant, `path`
  capped at 200 bytes. This includes `ReasonTooLarge` (node cap, work cap or per-tool
  output cap, R12; `path` is empty for it). A tool skipped for the per-resolution
  `outBudget` is not a normalization drop; it is counted in the existing
  `over_shared_budget` line (15.2).
- Optional properties dropped: one WARN line per KEPT tool (not per property), listing
  `dropped_properties` count (`Report.DroppedPropsN`) and at most the first 3 paths (reuse the
  `sampleToolNames` style: quoted, each capped at 80). A dropped tool (any reason,
  including `ReasonTooLarge`) emits only the tool-dropped line above, never this one,
  since the partial `DroppedProps` of an aborted build would mislead.
- Key stripping and rewrites (`x-*`, `title`, `oneOf` conversions, and so on):
  not logged per tool. They are routine on generated schemas (Pydantic `title`,
  `$defs`, `oneOf`; `x-mcp-header` on the owner/repo properties of 37 of the
  incident server's tools, an.2),
  so logging them at WARN is noise. One DEBUG line per resolution with the totals.
WARN matches `docs/conventions/logging.md` ("Warn: safe-default fallbacks"): the
session continues with fewer tools.

**No new metric.** Observability for A is the WARN lines above only. A counter can be
added later if the logs show real volume.

### 15.5 E: per-tool Gemini validation (bin-pipecat-manager Python runner)

**Hook.** `run.py create_llm_service`, `gemini` branch
(`bin-pipecat-manager/scripts/pipecat/run.py:495-512`), immediately after
`standard_tools = _openai_tools_to_standard(tools)` (`run.py:503`) and before
`ToolsSchema(...)`/`LLMContext(...)` (`run.py:504-509`). OpenAI and Grok branches
(`run.py:465-493`) are untouched.

```python
standard_tools = _openai_tools_to_standard(tools)
standard_tools = drop_gemini_invalid_tools(standard_tools, pipeline_id)   # NEW, gemini_tool_filter.py (15.7)
```

**Behavior of `drop_gemini_invalid_tools(schemas: list[FunctionSchema], pipeline_id: str = "") -> list[FunctionSchema]`**
(standalone module `scripts/pipecat/gemini_tool_filter.py`, 15.7). Every log line E
emits includes `pipeline id={pipeline_id}` in the text, following the runner's
existing convention (`run.py:146`, `:169`), because loguru has no bound context and
one runner process serves many sessions; `create_llm_service` gains an optional
`pipeline_id` argument fed by its caller, which already has `id`. An empty
`schemas` list returns immediately (no imports, no validation).
1. Import `GeminiLLMAdapter` (`pipecat.adapters.services.gemini_adapter`) and
   `GenerateContentConfig` (`google.genai.types`) inside the function, and only for
   a factory argument that is None (an injected `adapter_factory`/`config_factory`
   skips its real import, so the mocked control-flow tests do not fall into the
   fail-open path; `conftest.py:54` mocks `pipecat.adapters` as a plain MagicMock,
   under which the real adapter import always fails). If a real import
   itself fails, log WARN once per call (once per session, since the filter runs
   once at pipeline build) and return `schemas` unchanged (fail open). These
   and `ToolsSchema` are imported inside the function so the module imports without
   pipecat (15.7).
2. Fast path: build `GeminiLLMAdapter().to_provider_tools_format(ToolsSchema(standard_tools=schemas))`
   and `GenerateContentConfig(tools=...)` for the whole list. If it validates, return
   `schemas` unchanged (one validation per session in the normal case).
3. Otherwise, validate each `FunctionSchema` alone the same way. Keep those that
   pass. For each that raises `pydantic.ValidationError`, drop it and log
   `logger.warning(f"Dropped tool '{fs.name}' rejected by the Gemini schema validator: {n} error(s), first: {loc}: {msg}")`,
   with `loc`/`msg` from `e.errors()[0]`, the whole message capped at 300 chars.
   Never log the schema itself.
4. After filtering, validate the kept set once more. If it still fails (an
   interaction between tools, not seen so far), log WARN and return the original
   `schemas` unchanged (fail open to current behavior; the session behaves exactly as
   it would without E).
5. Any exception other than `pydantic.ValidationError` from the validator (adapter
   API change, unexpected type) at any step: log WARN with the exception type and
   return `schemas` unchanged. **A failure of the validator itself never removes
   tools.**
6. Log one INFO line when anything was dropped: `"Gemini tool validation dropped
   {k} of {n} tools"`.

It applies to all tools, built-ins included. Built-ins are expected to pass (they are
in the 08:02 control request); a dropped built-in is a regression signal and gets
the same WARN. The adapter is instantiated per call (it is stateless in 1.4.0,
`gemini_adapter.py:84`), and using the installed adapter makes E track whatever
conversion the installed pipecat does, including after track B upgrades it (an.5E).

**Dropped tool, still registered.** `tool_register` (`tools.py:101`) registers
handlers by name from the Go-provided tool list, independent of the advertised
schema. A handler for a tool the LLM was never shown is inert. No change there.

**Limit (stated).** E catches only client-side pydantic rejection. A server-side 400
(for example the `format`/`enum` cases R3 guards against) still fails the whole
request and E cannot see it. A's conservative rules and the live call (15.8) are
what cover that residual (an.5E, an.6). E also does nothing for OpenAI/Grok.

### 15.6 RTVI `error` frame at WARN (bin-pipecat-manager Go)

`receiveMessageFrameTypeMessage` (`bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:582`)
logs every unknown RTVI type at DEBUG in its `default:` branch (`runner.go:706-707`),
including pipecat's `error` message (`{label: "rtvi-ai", type: "error", data: {error:
str, fatal: bool}}`, pipecat 1.4.0 `processors/frameworks/rtvi/models.py:141-158`).

Change: add an explicit case before `default:`.
- New constant `RTVIFrameTypeError = "error"` in
  `bin-pipecat-manager/models/pipecatframe/helper.go` next to the other RTVI types
  (`helper.go:13-28`), plus a row in `helper_test.go`'s constants table and in the
  constants table of `rtvi_test.go` (~line 38).
- No new struct: unmarshal into the existing `pipecatframe.RTVIError`
  (`rtvi.go:166-177`, `Data RTVIErrorData{Error, Fatal}`, already tested in
  `rtvi_test.go`).
- Case body: unmarshal; on unmarshal failure fall back to logging `frame.Type` only.
  Log `log.WithFields(logrus.Fields{"pipecatcall_reference_type": se.PipecatcallReferenceType,
  "pipecatcall_reference_id": se.PipecatcallReferenceID, "fatal": msg.Data.Fatal}).Warnf("Pipecat runner reported an error. error: %s", capText(msg.Data.Error, 2048))`.
  `log` already carries `func` and `pipecatcall_id` (`runner.go:583-586`); the
  reference fields (`models/pipecatcall/session.go:16-17`) give the aicall id for a
  Loki join with ai-manager.
- Truncation: 2048 bytes, rune-boundary safe (a local helper equivalent to
  ai-manager's `capErrText`, `mcp_tool.go:729`; pipecat-manager has no such helper
  today, so add a small unexported one). The incident message was ~40 KB with 76
  errors; 2 KB keeps the first several errors, which is enough to identify the
  construct, while the full text stays in the runner's own ERROR record.
- Data in the logged text. The error string can carry customer data: pydantic's
  message includes `input_value=...` fragments of the rejected tool declaration
  (customer MCP schema text), and any other pipeline `ErrorFrame` text passes through
  the same field. This is acceptable at WARN in internal logs because the same data
  is already logged today: the runner's own ERROR record holds the full ~40 KB
  message (an.2), the current `default:` branch logs the whole frame, error text
  included, at DEBUG (`runner.go:706-707`), and ai-manager logs customer MCP server
  error text at WARN (`mcp_tool.go:437`, capped at 1024). The 2 KB cap keeps this
  line smaller than the existing ERROR record. Not logged anywhere customer-visible.
- No metric in this PR (an.6 marks it optional). The runner-side E WARN plus this
  WARN are sufficient for diagnosis; a counter can follow if alerting needs it.
- Not surfaced to the customer or to ai-manager. That is a separate product
  question (15.10).

### 15.7 Test plan (written at implementation, TDD, tests first)

**Go, `pkg/mcpschema` (new, CI-run via `bin-ai-manager-test`).**
1. Table test `Test_Normalize_Rules`, one or more rows per rule, each row
   `{name, in (JSON string), want (JSON string or nil), wantReport fields}`:
   keep list and `x-*`/`title`/`default`/`$schema`/`additionalProperties` removal;
   value-shape failures (bad `type` string, boolean subschema, tuple `items`,
   non-string `required` members, non-numeric bounds, non-string description);
   `format` allow pairs and a dropped `uri`/`email`; `format: enum` with and without
   `enum`; string enum kept, integer enum dropped, mixed enum dropped; list `type`
   including `["string","null"]`; `oneOf`, `oneOf`+`anyOf`; string and non-string
   `const`; single-member `allOf` (with `$ref` inside), multi-member `allOf` optional
   and required; `$ref` to `$defs` and `definitions`, missing ref, remote ref, direct
   cycle, indirect cycle, depth 9 chain, fan-out over 256; type inference for
   properties/items/string enum/anyOf and typeless unusable; nested free-form object,
   nested `properties: {}`, object whose props all cascade away; top-level
   `properties: {}` kept; array without items (optional removed, required drops the
   tool); cascade: unusable required inside optional nested object (object removed,
   tool kept), unusable required chain to root (tool dropped), `anyOf` with one bad
   member, `anyOf` with only `{type: null}` left; `required` pruning and dedupe; depth
   33 (and a `$ref` chain that does not add depth, R12); node cap drops the whole
   tool (this row passes an explicit `maxOutBytes` large enough that the output cap
   does not trip first); empty `type` list; nil input.
   Output-cap rows (R12): the amplification shape from
   `rr1_s15_ref_amplification.py` built in the test (one `$defs` string with a
   56,000-byte description, 256 properties each `$ref` to it; raw under 64 KiB, within
   `maxNodes` and `maxRefExpansions`) returns `ToolDropped` with `ReasonTooLarge`
   and a nil schema; a schema whose charge is just under the cap is kept and its
   `OutBytes` equals the hand-computed charge; a small plain schema's `OutBytes`
   matches a hand-computed value (pins the charge formula). Dropped-subtree row
   (visit charge): the shape from `rr2_s15_droppedprops_amplification.py` built in
   the test (one `$def` with thousands of typeless optional properties, referenced
   256 times; raw under 64 KiB, within `maxNodes` and `maxRefExpansions`) returns
   `ToolDropped` with `ReasonTooLarge`, and `len(Report.DroppedProps) <= 8` on every
   path. A row with 20 dropped optional properties on a kept tool asserts
   `DroppedPropsN == 20` and `len(DroppedProps) == 8`.
2. `Test_Normalize_DoesNotMutateInput`: deep-copy input, normalize, compare.
3. `Test_Normalize_Deterministic`: run 50 times on a fixture with many optional
   unusable properties; `Report.DroppedProps`, `DroppedPropsN` and `DropPath` identical every run.
4. Fixture test `Test_Normalize_GitHubMCPFixtures`: `testdata/github/*.json` holds
   a small copied subset of GitHub MCP `pkg/github/__toolsnaps__` snapshots (MIT,
   commit `85598ba`; keep the upstream LICENSE notice in `testdata/github/README.md`):
   `issue_write` (list `type` inside `issue_fields.items.properties.value`, nullable
   `anyOf` in `type`), `projects_write` (typeless `value` in every `oneOf` member,
   expected single dropped property `/properties/updated_field`), `update_issue_labels`
   (`oneOf` in `items`), `custom_properties_write` (`oneOf` of objects), and
   `get_file_contents` (plain). Plus one synthetic `incident_x_mcp_header.json`
   reproducing the prod-only shape (`owner`/`repo` with `x-mcp-header`), since the
   local snapshots carry none (an.5A). Golden outputs in `testdata/github/*.golden.json`,
   compared as decoded maps. Pinned expectations: `update_issue_labels`
   `/properties/labels/items` is NOT an object-with-`anyOf` (it has only `oneOf`, no
   `type`, no `properties`), so R10 does not apply: R5 converts `oneOf` to `anyOf`, R6
   emits no `type`, and the result is `{anyOf: [{type: string, description}, {type:
   object, properties: {...}, required: [name]}]}` (both variants kept). For the
   object-with-`anyOf` shape (R10): `projects_write` `/properties/items/items` becomes
   `{type: object}`; `custom_properties_write` `/properties/properties/items` becomes
   `{type: object}`; `projects_write` `/properties/updated_field` is removed (its
   `anyOf` has no usable member), and the tool is kept. (Paths per the snapshots;
   confirm exact JSON pointers when copying the fixtures, and treat a mismatch as a
   spec question, not a golden update.)
   **Goldens are generated from the Go implementation** (a one-off `go test` helper
   run at implementation time, then reviewed by hand against this spec and
   checked in). The Python oracle `rv6_local_cascade.py` / `rv6_normalized.json`
   (`~/.hermes/cache/scratch/pc/`) is not the source of the goldens, because its
   output shape differs from this spec in known ways (probe
   `rr1_s15_oracle_vs_spec.py`):
   - it copies `description` into each `anyOf` member produced from a list `type` or
     `oneOf` (R4 moves it to the parent only): 3 members in `issue_write`. The
     `update_issue_assignees`/`update_issue_labels` member descriptions ("GitHub
     username", "Label name") are the members' own in the raw schema and are kept;
   - it emits `required: []` on nested objects (R8 omits an empty `required`): 2
     objects in `actions_list`;
   - its depth limit is 20 and it drops the tool from any depth (this spec: 32,
     local cascade, R7/R12).
   The oracle is used once, at implementation time, only to cross-check outcomes:
   the same tools dropped (0) and the same properties dropped (1,
   `projects_write.updated_field`), and that google-genai accepts the Go output
   (the real-library run below). The test never reads scratch paths.
5. Synthetic fixtures from an.7 as their own golden cases: nested free-form object
   (normalized `{type: object}`), explicit nested `properties: {}`, nullable
   `anyOf` with `{type: null}` (Pydantic `Optional`), typeless string enum. These are
   also the payloads for the live call (15.8), checked in as
   `testdata/live/*.json` (normalized output; the test asserts they equal the
   current `Normalize` result, so they cannot drift).

**Go, `pkg/aicallhandler`.** Extend `Test_resolveMcpOnly` (`mcp_tool_test.go:51`)
with a new `expectParams` field on the table (the table today has only
`expectToolNames` and `expectToolMap`), and rows: a tool whose schema normalizes (asserts `Parameters` equals the normalized map, not
the raw one); a tool dropped by normalization (asserts absent from both the returned
slice and `toolMap`, and the server's other tools kept); `outBudget`: tools whose
charge is well above their raw size (a `$ref` fan-out of about 35 KiB raw that
charges about 60 KiB (under the 64 KiB per-tool cap); five of them are about 175 KiB
raw, under the 256 KiB `schemaBudget`, while four charge about 240 KiB and the fifth
would reach about 300 KiB, over the 256 KiB charge budget. The `$ref` variant is the
one to use: a short-property variant (about 22 raw bytes and 45 charged per unique
property) would need each tool kept under about 31 KiB raw so it does not trip the
per-tool cap first); assert only four kept, `toolMap` and the
returned slice the same size, and that `schemaBudget` still has bytes left (proving
`outBudget`, not `schemaBudget`, fired; mirrors `Test_resolveMcpOnly_SchemaLimits`,
`mcp_tool_test.go:1026`); and a no-argument tool with root `{}` kept as
`{type: object, properties: {}}` (R0); and `schemaBudget` accounting unchanged
(`Test_decodeToolSchema` rows at `mcp_tool_test.go:929-935` stay as they are).

**Go, `bin-pipecat-manager`.** Table rows for `receiveMessageFrameTypeMessage` with an
`error` frame (fatal true/false, oversize error text truncated, malformed data
falls back), asserting no error returned. No test in `pkg/pipecatcallhandler` uses a
logrus test hook today, so the rows assert behavior (no error, no panic, truncation
via the helper's own unit test), not the log level. `helper_test.go` and `rtvi_test.go` get the new constant row.

**Python, E.** Current layout: tests are flat files next to `run.py`
(`scripts/pipecat/test_run.py` and siblings), and `scripts/pipecat/conftest.py`
installs `MagicMock` modules over the pipecat tree, including `pipecat.adapters` and
`FunctionSchema`/`ToolsSchema` (`conftest.py:28-60`), into `sys.modules` for every
test collected in that directory (`conftest.py:100-102`). Under it a real-library
import of `pipecat.adapters.services.gemini_adapter` fails with
`'pipecat.adapters' is not a package` (probe `rr1_s15_conftest_shadow.py`), so a
real-library test placed there would neither skip cleanly nor test real pipecat.
CI does not run pytest at all: `.circleci/config_work.yml` has no pytest step (the
pipecat job, `go-test-pipecat-manager`, runs Go only). So:
- Put the filter in its own small module `scripts/pipecat/gemini_tool_filter.py`
  (about 40 lines) with `drop_gemini_invalid_tools(schemas, pipeline_id="", adapter_factory=None,
  config_factory=None)`. It imports only `loguru` at module level; pipecat,
  google-genai and pydantic are imported inside the function (15.5 step 1), so
  importing the module needs neither real `run.py` nor pipecat. `run.py` imports it
  (`from gemini_tool_filter import drop_gemini_invalid_tools`, next to the existing
  local import `from message_filters import ...` at `run.py:44`), and the Dockerfile
  already copies the whole directory (`bin-pipecat-manager/Dockerfile:20`). The two
  factory parameters default to the real `GeminiLLMAdapter` and
  `GenerateContentConfig` (with `ToolsSchema`, also imported inside the function).
- Control-flow unit tests in `scripts/pipecat/test_gemini_tool_filter.py` (runs under
  the existing mocked `conftest.py`) with injected fake adapter and config factories:
  fast path, per-tool drop, fail open on a non-pydantic exception, fail open on
  import error, fail open when the filtered set still fails. The fake raises a real
  `pydantic.ValidationError` if pydantic is importable, otherwise the test module
  skips (pydantic is not mocked by conftest).
- Real-library test in a directory outside `scripts/pipecat/`, so that conftest is
  never collected:
  `bin-pipecat-manager/scripts/pipecat_realtest/test_gemini_tool_filter_real.py`
  (a subdirectory of `scripts/pipecat/` would not do: pytest also loads
  parent-directory conftests, checked with a two-file probe,
  `~/.hermes/cache/scratch/pc/rr1_conftest_probe/`). It is not shipped (the
  Dockerfile copies only `scripts/pipecat`, `bin-pipecat-manager/Dockerfile:20`). It
  adds `scripts/pipecat` to `sys.path` and imports `gemini_tool_filter` only (never
  `run.py`). It starts with
  `pytest.importorskip("pipecat.adapters.services.gemini_adapter")` (the adapter
  module, not `google.genai`: the adapter is what E calls, and it pulls in genai).
  It reproduces `rv2_probe_e_hook.py`: a good tool plus an `x-mcp-header` tool fail
  as a set, the filter keeps only the good one, the filtered set validates; plus one
  case feeding the Go goldens of 15.7 item 4 through the adapter and
  `GenerateContentConfig` (all accepted). Command:
  `pytest --noconftest bin-pipecat-manager/scripts/pipecat_realtest` (`--noconftest`
  as a second guard). Run manually during implementation in a venv with the pinned
  versions (pipecat-ai 1.4.0 and google-genai 1.75.0 per
  `scripts/pipecat/uv.lock`, the prod set per an.4) plus pytest (install it into that
  venv first; the scratch venv has none); record the command and result
  in the PR body. The mocked control-flow test for "fail open on import error" must
  use `monkeypatch.setitem(sys.modules, "pipecat.adapters.services.gemini_adapter",
  None)` as a deterministic guard (conftest mocks `pipecat.adapters*` but not that
  submodule, so without the guard the outcome would depend on the environment).
- State plainly in the PR body that CI does not run pytest, so the Python change is
  not CI-tested and the real-library result is a manual record. Adding a pytest CI
  job is out of scope (it belongs with track B, whose upgrade needs it).

### 15.8 Pre-merge live verification

Who: the CEO triggers it (paid provider calls; per the no-cost-test rule these are
not automated). Claude prepares the payloads and the exact commands, and records the
results in the PR body.

Checklist:
- [ ] Gemini (`gemini-2.5-flash`, the incident model) one request whose tool list is
  the normalized real fixture set (15.7 item 4, including the list-`type` case) plus
  the four synthetic tools (nested free-form `{type: object}`, nested
  `properties: {}` as normalized, nullable `anyOf` with `{type: null}`, typeless
  string enum) plus the current built-ins, through the pinned runner path (pipecat
  1.4.0 adapter). Pass: HTTP 200 with a normal completion. If the nested
  `properties: {}` probe is sent both raw and stripped, record which one the server
  accepts (an.7 asks this); the rule R10 stays "strip" unless raw is also accepted.
- [ ] Repeat the prod scenario end to end: the AI `6e391666` setup (or an equivalent
  test AI with the GitHub-style MCP server) on the deployed branch, one messaging
  turn, assistant reply received, runner log shows no `GenerateContentConfig`
  validation error.
- [ ] OpenAI (`openai.gpt-4o-mini` or the model in use): one call with the same
  normalized tool set. Pass: 200.
- [ ] Grok: one call with the same set. Pass: 200.
- [ ] Negative check for E: temporarily feed one raw (unnormalized) `x-mcp-header`
  tool to the runner locally (no paid call needed, validation is client-side):
  built-ins survive, WARN logged. In the same local run, bypass E once (call the
  runner without the filter) to confirm the new pipecat-manager WARN
  `Pipecat runner reported an error` appears with `pipecatcall_reference_id` (15.6).

Rollback: `MCP_TOOL_EXPOSURE_ENABLED=false` on ai-manager (§8, B27) makes
`ResolveMcpTools` return nothing, so no MCP schema reaches any provider. The variable
is not in `bin-ai-manager/k8s/` today, so flipping it means adding it to the
Deployment env (or `kubectl set env deployment/<ai-manager> MCP_TOOL_EXPOSURE_ENABLED=false`
for an immediate change, followed by the manifest change). Per-AI
mitigation: clear `mcp_server_ids` on the affected AI. Neither A nor E changes the
built-in path, so rollback of this addendum alone is a code revert.

### 15.9 Docs to update at implementation

- `bin-ai-manager/docs/operations.md`: no metrics table change (no new metric,
  15.4; the `mcp_tool_advertised_total` row stays as is). Add a note under the
  `## Alerting Guidance` heading: repeated `Dropped an mcp tool whose input schema
  cannot be made provider-safe` WARN lines for one `mcp_server_id` mean that customer's MCP server
  exposes schemas VoIPBin cannot advertise.
- `bin-ai-manager/docs/domain.md`: extend the "MCP tools" paragraph (line 154) with
  one sentence: input schemas are normalized to a provider-neutral subset before
  advertisement; a tool whose schema cannot be normalized is not advertised; call
  arguments are forwarded unchanged.
- `bin-pipecat-manager/docs/operations.md`: note the Gemini per-tool validation
  filter and the WARN `Pipecat runner reported an error` log line (troubleshooting
  entry, no new metric).
- Customer-facing RST (behavior change: tools can be silently omitted from what the
  AI sees): `bin-api-manager/docsdev/source/ai_struct_mcpserver.rst`. The file has
  two "Tool use scope" notes that disagree today: the first (lines 29-35) omits
  realtime voice, the second (lines 97-99, under Status) excludes realtime voice
  sessions. At implementation they are merged into one note at the first location,
  keeping the second's scope text (realtime voice, `type=insight`, and team AI
  calls excluded), and the Status copy is replaced by a `:ref:` to it. The schema
  text goes into that merged note. Add: VoIPBin advertises each tool's input schema
  in a provider-neutral subset (listed keywords), unsupported keywords are removed, an optional parameter whose
  schema cannot be expressed is omitted, a tool whose required parameters cannot be
  expressed is not offered to the AI, and the arguments the AI sends are forwarded to
  the server unchanged, so the server's own validation still applies. Include a
  short "schema tips" list (give every property a `type`; arrays need `items`; avoid
  multi-member `allOf`; prefer string enums). Rebuild HTML per
  `bin-api-manager/CLAUDE.md` (`python3 -m sphinx -M html source build`,
  `git add -f docsdev/build/`).

### 15.10 Risks and open questions

| # | Item | Status / recommendation |
|---|---|---|
| R-1 | Server-side rejection of shapes the client accepts (format, enum, free-form object on non-2.5-flash Gemini models: 2.5-pro, 2.0-flash, pro-latest, `models/ai/main.go:156-159`) | Residual. Covered only by R3 conservatism and the 15.8 live call on 2.5-flash. Other models unverified (no traffic in 7 days, an.5A). Accept; note in PR body |
| R-2 | OpenAI/Grok acceptance of the normalized subset unverified offline | 15.8 live calls are mandatory before merge |
| R-3 | Constraint loss (`pattern`, `minLength`, `additionalProperties: false`, `format: uri`) lets the LLM send values the server rejects | By design; server validation plus B24 `isError` returns the failure to the LLM. Documented to customers (15.9) |
| R-4 | Pre-existing at HEAD, widened by A: name-only paths store every discovered tool's name, including tools `decodeToolSchema` skips, so `mcp_tool_map` can name an unadvertised tool that is dispatched if the LLM invents the exact name. A adds its dropped tools. Variants: (a) a tool E drops stays in `toolMap` and its Python handler is registered; (b) when `persistToolMap` fails (`mcp_tool.go:188-193`, logged, not fatal) the name-only map from session start stays for that session (15.2) | Accept: never advertised, normally replaced by the advertised list on `ResolveMcpTools`, server validates. Fixing it would mean decoding and normalizing on name-only paths, which never decode schemas by design (memory, `mcp_tool.go:356-360`) |
| R-5 | E is not CI-tested (no pytest job; conftest mocks pipecat) | Mitigated by a small standalone injectable module, mocked control-flow tests, one manual real-library test outside the mocked conftest's directory, recorded in the PR. CI pytest belongs to track B |
| R-6 | Customer invisibility: the customer still gets no signal when a tool is dropped or a turn fails | Out of scope. O7 (square-admin surfacing) is the natural home; the WARN logs exist for support |
| Q-1 | Decisions made in this addendum, not in the analysis: `mcpschema` package placement; limits `maxDepth=32`, `maxRefDepth=8`, `maxRefExpansions=256`, `maxNodes=4096`; output charge formula, per-tool output cap 64 KiB, `maxWork=262144` (round-1 code review), R7a constraint-only combinator removal and R10a no `type`+`anyOf` siblings (round-1 code review) with the same-typed scalar documented-enum exception and R7a judged on resolved members (round-2 code review), scalar type inheritance for typeless members and the look-ahead's own `$ref` budget (round-3 code review), inheritance judged on resolved members, for list types and gated on enum values, and the look-ahead budget per subschema (round-4 code review), duplicate list-type collapse, separate per-resolution `outBudget` of 256 KiB (R12); a `$ref` expansion adds no depth level (R12); bad `type` string, boolean subschemas, and tuple `items` are unusable and cascade (R2, R7); an empty `type` list is unusable (R4); resource-limit overflow (`maxNodes`, `maxWork`, output cap) drops the whole tool instead of cascading, while depth and ref limits cascade (R12); null-only `anyOf` unusable (R7); one-member `anyOf` not flattened; required-unusable nested object stays unusable rather than free-form (R10); `const` with a non-string `type` dropped; no new metric (15.4); E in a standalone module, fast path, fail-open-when-filtered-set-still-fails; RTVI error text cap 2048 | For design review |
| Q-2 | Should E also run for the team flow path (`team_flow.py`)? | No: team AIcalls never receive MCP tools (§2.4, an.7), and team built-ins already pass. Revisit with track B |
