# Case Insight Assistant: tool whitelist expansion + "all" support (NOJIRA)

**Date:** 2026-07-30
**Ticket:** none (NOJIRA)
**Services:** `bin-ai-manager`, `bin-pipecat-manager`, `square-admin` (frontend)
**Status:** Design reviewed (5 rounds each, 2 consecutive approvals) for both parts below. Not yet implemented.

## Problem

Case Insight Assistant (`type=insight` AI, `bin-ai-manager`) currently exposes
exactly two tools to the LLM: `get_contact_interactions` and
`get_conversation_content` (`tool.AllInsightToolNames`,
`bin-ai-manager/models/tool/main.go:50`). Two separate gaps were raised in
this design pass:

1. **The whitelist is too narrow.** Agents want the assistant to answer "has
   this contact had other cases?" and "what did the last agent write in their
   notes?" — neither is answerable today.
2. **Insight AI cannot use `tool_names=["all"]`.** Normal AI supports this
   shorthand in square-admin's AI editor ("Enable All Tools"); Insight AI's
   editor has no equivalent, and the backend rejects `"all"` for
   `type=insight` outright.

Investigating (2) surfaced a pre-existing production gap unrelated to the
original ask: the *actual* runtime tool-expansion code lives in
`bin-pipecat-manager`, not `bin-ai-manager`, and it has no `AIType` awareness
at all — see §2 below. Fixing that is a prerequisite for shipping "all"
support safely, so it's included in this design.

## Non-goals

- **Tier 2/3 candidate tools** surveyed during requirements discussion
  (`get_call_transcript`, `get_call_history`, `get_message_history`,
  `get_queue_wait_status`, `get_campaign_membership`, `get_contact_profile`,
  `get_billing_status`, `get_customer_profile`) — deferred. Most depend on
  filter-map RPCs (`CallV1CallList`, `TranscribeV1*List`, etc.) that have no
  server-enforced tenant filter; they need a shared scope-verification
  helper before they're safe to whitelist. Not designed here.
- **`get_case_tags`** — deferred. Depends on `TagV1TagList`, a free-form
  filter API with no `customer_id` parameter; needs its own
  ownership-reverification design.
- **`get_ai_summary_history`** — deferred, and currently **not viable as
  specified**: `summary.ReferenceType` (`bin-ai-manager/models/summary/main.go:32-36`)
  only covers `call/conference/transcribe/recording`, not `contact_case`, so
  a naive `reference_id=c.ReferenceID` filter always returns empty. Needs a
  multi-hop design (Case → interaction → call/transcribe reference) before
  reconsideration.
- **Rate limiting / automated repeated-attempt detection** on tool calls —
  out of scope for this change; audit logging (§1.3) is the only control
  landing now. Tracked as a follow-up.
- **Automated read-only enforcement** on future `AllInsightToolNames`
  entries — this design documents the requirement and recommends one cheap
  unit test (§2.6); a structural guard (e.g. a `Tool` metadata flag) is a
  follow-up, not required to ship.

## Part 1 — Two new Insight tools

### 1.1 `get_related_cases`

Returns metadata (not full body) for other cases belonging to the same
contact as the current case, so the assistant can answer "has this contact
had other cases?" without exposing unrelated case content.

**Decision matrix** (mirrors the existing `get_contact_interactions` pattern
in `bin-ai-manager/pkg/aicallhandler/tool_insight.go`):

| # | Condition | Result |
|---|---|---|
| 1 | `c.ReferenceType != aicall.ReferenceTypeContactCase` | tool failure (entry guard, shared with `get_case_notes`) |
| 2 | `ContactV1CaseGet(case_id=c.ReferenceID)` RPC errors/times out | tool failure — **never** silently treated as "no data" |
| 3 | CaseGet succeeds, case not found | success, empty result (not-found) |
| 4 | CaseGet succeeds, `kase.CustomerID != c.CustomerID` | mask as not-found (empty result) + `log.Warnf` audit (actor/customer_id/case_id/reason/timestamp) — cross-tenant attempt |
| 5 | CaseGet succeeds, ownership confirmed, `kase.ContactID == nil` **or** `*kase.ContactID == uuid.Nil` | success, empty result (legitimate: no linked contact) — no audit needed, not a denial |
| 6 | CaseGet succeeds, ownership confirmed, `ContactID` present and non-nil-UUID | `ContactV1CaseList(ctx, customerID=c.CustomerID, status="", ownerType="", ownerID=uuid.Nil, contactID=*kase.ContactID, size=insightMaxListLimit, token="", referenceID="")`. RPC error → tool failure. Success → map to `{case_id, title, status, created_at}` only (no body/notes), **exclude `case_id == c.ReferenceID`** (the current case itself) from the result, apply `insightMaxListLimit` **after** exclusion (so the truncation flag reflects the post-exclusion count, not a stale pre-exclusion count) |

Step 5's nil-`ContactID` handling exists specifically to close a real bug
found in review: `ContactV1CaseList`'s underlying query
(`bin-common-handler/pkg/requesthandler/contact_cases.go:103`,
`if contactID != uuid.Nil { ... }`) silently *drops* the `contact_id` filter
when given `uuid.Nil` — calling it with a zero-value UUID would return every
case for the tenant, not "no related contact." Step 5 must never reach the
RPC in that case.

**Accepted residual risk (documented, not IDOR):** the assistant can see
metadata (title/status/date, not body) of other cases from the *same*
contact, even if unrelated to the current conversation topic. This is
same-tenant, same-contact data, not a cross-customer leak, but is a
deliberate minimum-privilege trade-off — recorded here per review
requirement rather than silently accepted.

### 1.2 `get_case_notes`

Returns internal agent notes on the *current* case (not other cases) —
useful for handoffs between agents.

Same guard (row 1) and CaseGet ownership check (rows 2-4) as
`get_related_cases`. On ownership confirmed:
`ContactV1CaseNoteList(ctx, customerID=c.CustomerID, caseID=c.ReferenceID)`
(`bin-common-handler/pkg/requesthandler/contact_cases.go:314` — this RPC
takes no `size`/`token` args, unlike the case list one above). RPC error →
tool failure. Success → truncate client-side to `resolveInsightListLimit(args.Limit)`
(the same clamp-and-truncate helper the two existing tools already use,
**not** a new hardcoded constant), keeping the **most recent** N notes and
using the existing `renderBodyLines` "showing the most recent N" truncation
marker — confirm note sort order (`tm_create` asc/desc) before slicing so
the marker's claim ("most recent") is actually true.

### 1.3 Cross-cutting for both tools

- Response list size: reuse `insightMaxListLimit` (case list) /
  `resolveInsightListLimit` (case notes) — no new constants.
- Audit log fields (row 4 in both): actor (AI session/aicall id),
  customer_id, case_id, denial reason, timestamp — matches the existing
  `log.Warnf("Cross-customer ... access blocked")` convention in
  `tool_insight.go`.
- Registration work (not just the whitelist): `tool.AllInsightToolNames`
  (`models/tool/main.go:50`) entries, `toolDefinitions` schema entries in
  `bin-ai-manager/pkg/toolhandler/definitions.go`, dispatch branches in
  `tool_insight.go`, and a `ValidateToolNames` unit test asserting no
  write-capable tool name can ever be added to `AllInsightToolNames`.

## Part 2 — Insight AI `"all"` support (and the pipecat fail-open fix it depends on)

### 2.1 Why this isn't a one-line change

The obvious fix — allow `"all"` in `bin-ai-manager`'s
`ValidateToolNames` — is necessary but not sufficient. The actual runtime
tool-list expansion for a live LLM session happens in a **different
service**, `bin-pipecat-manager`, whose `GetByNames` has no `AIType`
parameter at all today:

```go
// bin-pipecat-manager/pkg/toolhandler/main.go:74-103 (current, unmodified)
func (h *toolHandler) GetByNames(names []aitool.ToolName) []aitool.Tool {
    if len(names) == 0 { return []aitool.Tool{} }
    for _, name := range names {
        if name == aitool.ToolNameAll {
            return h.GetAll() // no type check — returns all 17 cached tools
        }
    }
    // ... else: return cached tools whose name is in `names`, again no type check
}
```

`h.tools` is a flat, untyped cache of all 17 tool definitions (Normal +
Insight combined) fetched once via `AIV1ToolList`
(`bin-ai-manager/pkg/listenhandler/v1_tools.go:12`, itself untyped —
`toolDefinitions` in `definitions.go` has no type tag).

**Confirmed pre-existing bugs this surfaces, independent of the "all"
feature request:**

- Normal AI's existing `tool_names=["all"]` already returns the two
  Insight-only tools alongside all 15 Normal tools — the type separation
  `ValidateToolNames` enforces at save time is not re-checked at pipecat's
  expansion time.
- `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`'s
  `resolveAIFromAIcall` failure branch falls back to `GetAll()` —
  **every** tool, including write-capable ones (`create_call`,
  `send_email`, ...), whenever the AI record lookup fails for *any*
  reason, regardless of the AI's actual type. This is a documented,
  intentional decision (VOIP-1234 §6 v4) that predates this design and is
  being revisited here because it directly contradicts the
  defense-in-depth this change introduces.
- A second `GetAll()` call site (non-`AICall` reference types,
  `runner.go`) is provably Normal-only by code inspection but doesn't say
  so — it just happens to never be reached by Insight sessions today.

### 2.2 Single source of truth for the whitelist

Add to `bin-ai-manager/models/ai` (not `models/tool`, to key on the real
`Type` enum rather than a boolean — see rationale below):

```go
// AllowedToolNames returns the tool whitelist for a given AI type.
// Unknown/future types deny-by-default (empty set) rather than falling
// back to the widest (Normal, write-capable) set.
func AllowedToolNames(t Type) map[tool.ToolName]bool {
    switch t {
    case TypeNormal:
        return toSet(tool.AllToolNames)
    case TypeInsight:
        return toSet(tool.AllInsightToolNames)
    default:
        log.Warnf("unknown AI type %v encountered in tool whitelist resolution, denying all tools", t)
        metrics.UnknownAITypeToolDenialTotal.Inc() // new counter
        return map[tool.ToolName]bool{}
    }
}
```

`models/ai` already imports `models/tool` (for `ValidateToolNames`), so this
has no import cycle. A `bool isInsight` parameter was considered and
rejected: it collapses any future third `AIType` into "Normal" (the
write-capable set) by construction, which is the opposite of
deny-by-default.

**Important semantic note** (this caused confusion across review rounds and
must be stated explicitly): `tool.ToolNameAll` (`"all"`) is a **selector**,
not a concrete tool name — `tool.AllToolNames` / `tool.AllInsightToolNames`
correctly do **not** contain it. `AllowedToolNames(t)` only ever answers
"is this concrete tool name allowed for type t"; it is never asked whether
`"all"` itself is a member.

`ValidateToolNames` (`bin-ai-manager/models/ai/tool_validation.go`) is
refactored to consume this same helper instead of building its own inline
sets:

```go
func ValidateToolNames(t Type, toolNames []tool.ToolName) error {
    allowed := AllowedToolNames(t)
    for _, name := range toolNames {
        if name == tool.ToolNameAll {
            continue // selector: structurally valid input for any type that supports "all"
        }
        if !allowed[name] {
            return fmt.Errorf("invalid tool_names for type=%s: %q is not allowed", t, name)
        }
    }
    return nil
}
```

This is what actually grants Insight AI the ability to save `["all"]` —
today `case TypeInsight:` rejects it outright.

### 2.3 Pipecat-side defense-in-depth

```go
// bin-pipecat-manager/pkg/toolhandler/main.go
func (h *toolHandler) GetByNames(aiType amai.Type, names []aitool.ToolName) []aitool.Tool {
    allowed := amai.AllowedToolNames(aiType) // re-resolved every call, never cached across calls
    hasAll := containsName(names, aitool.ToolNameAll)
    h.mu.RLock()
    defer h.mu.RUnlock()
    result := make([]aitool.Tool, 0, len(h.tools))
    for _, t := range h.tools {
        if !allowed[t.Name] {
            continue // concrete-name check only; "all" is never itself checked for set membership
        }
        if hasAll || containsName(names, t.Name) {
            result = append(result, t)
        }
    }
    return result
}
```

Whether `names` contains `"all"` or an explicit list, every returned tool
must first be a member of `allowed`. This closes both the "Normal `all`
leaks Insight tools" bug and the "`ValidateToolNames` bypassed via direct
DB write" gap — the type check is re-applied at expansion time regardless
of what's stored.

`aiType` is derived **fresh** at each call site from the AI record already
in hand (`run.go:178`'s `resolveTeamForPython`, `runner.go:148`) —
`ai.Type == amai.TypeInsight` — not threaded through as a long-lived
cached value. This mirrors the existing pattern one line away, where the
same functions already read `ai.RagID` to conditionally include
`search_knowledge`.

### 2.4 Fail-open → fail-closed policy change

`runner.go`'s `resolveAIFromAIcall` failure branch currently does:

```go
// current: "Could not resolve AI, returning all tools"
tools = h.toolHandler.GetAll()
```

This is changed to fail-closed: on AI-lookup failure, return `[]aitool.Tool{}`
(no tools) rather than every tool. This is an explicit policy reversal of
VOIP-1234 §6 v4's original fail-open decision, made because tool-access
control is a case where least-privilege must outweigh availability — a
degraded (tool-less) session is an acceptable outcome; silently granting
write-capable tools to a session whose type couldn't even be determined is
not. The trade-off (a transient AI-lookup RPC failure now produces a
tool-less session instead of a fully-capable one) is accepted as-is;
retry-before-fail-closed was considered and explicitly rejected as
unnecessary scope for this change.

The second call site, `runner.go`'s non-`AICall` reference-type branch
(provably Normal-only), becomes `GetByNames(amai.TypeNormal, []aitool.ToolName{aitool.ToolNameAll})`
instead of raw `GetAll()`.

`ToolHandler.GetAll()` is removed from the exported interface entirely (or
unexported to an internal `getAll()` helper used only inside `GetByNames`'s
own implementation) once both call sites above are migrated, so a future
caller cannot silently reintroduce a fail-open path by calling it directly.
This does **not** affect the unrelated, same-named
`bin-ai-manager/pkg/toolhandler.GetAll()` (different package, different
purpose — backs the admin-facing `AIV1ToolList` RPC, not runtime
expansion).

### 2.5 Mechanical follow-through (implementation checklist, not exhaustive)

- `bin-pipecat-manager/pkg/toolhandler/mock_main.go` — regenerate for the
  new `GetByNames` signature.
- `bin-pipecat-manager/pkg/toolhandler/main_test.go` (`TestToolHandler_GetByNames`
  and `TestToolHandler_GetAll`, if `GetAll` is removed) — update.
- `bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go` (lines using
  single-arg `GetByNames(gomock.Any())`) — update to 2-arg form.
- `bin-pipecat-manager/pkg/pipecatcallhandler/runner_toolfallback_test.go` —
  currently asserts the fail-open fallback (`mockTool.EXPECT().GetAll().Return(...)`);
  invert to assert fail-closed (empty tool list), and add
  `TestToolResolve_AIResolveFailure_FailsClosed` (name indicative) pinning
  the new behavior.
- `bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go` (`GetAll().AnyTimes()`
  expectations) — audit and update now that the non-AICall path calls
  `GetByNames` instead.
- `bin-pipecat-manager/pkg/pipecatcallhandler/metrics.go` and
  `docs/operations.md` — both currently document the fallback as
  intentional fail-open (VOIP-1234 §6); update to describe the fail-closed
  policy and add a superseding note on the original VOIP-1234 §6 v4 design
  doc.

### 2.6 Live-whitelist policy (explicit decision record)

`tool_names=["all"]` is stored as the literal string, re-expanded via
`AllowedToolNames(t)` at every call. This means any future addition to
`tool.AllInsightToolNames` (e.g. Part 1's `get_related_cases`/
`get_case_notes` once merged) is **automatically** granted to every
existing Insight AI already storing `["all"]`, with no re-consent step.

This is accepted as the intended model: Insight AI's whitelist is
constructed to only ever contain read-only tools, so scope growth carries
low risk. This assumption must hold going forward — **recommended (not
blocking) for the implementation PR:** add one unit test asserting every
entry in `tool.AllInsightToolNames` maps to a tool definition with no
side effects (however "read-only" is representable in the `Tool`
struct/definition — introduce a metadata flag if none exists). Automating
this via a structural guard (vs. a single assertion test) is a follow-up,
not a requirement to ship.

### 2.7 Frontend (`square-admin`)

- `src/views/ais/AIEngineFields.js`: extend the "Enable All Tools" checkbox
  (currently rendered only for `aiType === 'normal'`) to also render for
  `'insight'`. The underlying state (`enableAllTools`, `selectedTools`) and
  save/load logic in `ais_create.js` / `ais_detail.js` (`toolNamesValue =
  enableAllTools ? ['all'] : selectedTools`, and the reverse mapping on
  load) is already type-agnostic and needs no change.
- `ais_create.js` / `ais_detail.js`: both currently force
  `setEnableAllTools(false)` on switching `aiType` to `'insight'` — this
  reset logic must be removed (or made conditional) now that Insight
  supports the flag.
- `src/views/ais/constants.js`: update the comment documenting that
  Insight rejects `"all"` (no longer true).
- `AIEngineFields.test.js`: update the existing assertions that Insight has
  no "Enable All" checkbox.

## Review history (condensed)

Both parts went through 5 rounds of independent architecture + security
review (parallel reviewers each round, `Agent` tool, `architect` +
`security-reviewer` subagent types) before reaching 2 consecutive
approvals, per this repo's mandatory review-loop policy. Substantive
findings that changed the design, in order:

**Part 1** (new tools): tier misclassification (two "safe" candidates
actually shared Tier 2's IDOR risk) → batch size cut from 10 candidates to
2 → `get_ai_summary_history` found non-viable by code inspection (dropped)
→ `get_customer_cases` scope cut from tenant-wide to same-contact,
renamed `get_related_cases` → fail-open bug found in the nil-`ContactID`
path (§1.1 row 5) → fail-closed semantics for RPC errors vs. not-found
clarified (§1.1 row 2 vs. 3) → self-inclusion bug found (current case
appearing in its own "related cases" list, §1.1 row 6).

**Part 2** (`"all"` support): initial design targeted dead code
(`bin-ai-manager`'s unused `GetByNames`) — real expansion logic in
`bin-pipecat-manager` found by the first review round → surfaced the
pre-existing Normal/Insight leak and the `resolveAIFromAIcall` fail-open
bug (§2.1) → whitelist helper relocated from a `bool` in `models/tool` to
a `Type`-keyed function in `models/ai` for deny-by-default safety on future
types → fail-open→fail-closed policy change proposed and reviewed
specifically for its availability trade-off → missing test/mock/doc call
sites found by direct grep in round 3.

## Open questions for the next stage (implementation plan)

1. Should Part 1 and Part 2 ship as one PR or two? They touch different
   services (`bin-ai-manager` models/tools vs. `bin-pipecat-manager`
   runtime) and are independently valuable; splitting reduces review
   surface per PR but Part 2's whitelist refactor (`AllowedToolNames`) is
   a clean dependency Part 1's new tools should register against either
   way.
2. Confirm current sort order of `ContactV1CaseNoteList` results before
   implementing the "most recent N" truncation in `get_case_notes`
   (§1.2) — flagged by review as needing code-level verification, not
   assumed here.
