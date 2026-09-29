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
