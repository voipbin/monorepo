# Auto AICall Audit — Design

**Date:** 2026-05-30
**Service:** `bin-ai-manager` (+ `bin-common-handler`, `bin-dbscheme-manager`)
**Status:** Draft — under design review

## Problem

Today an AICall audit (`aiaudit`) is created **only on demand** via `POST /v1/aiaudits`
(`aiauditHandler.Create`). There is no way to have audits run automatically when an
AI call finishes. Customers who want every call for a given AI evaluated must build their
own external orchestration that watches for call termination and calls the audit API.

We want auto-auditing to be a **property of the AI** itself: if an AI is configured for
auto-audit, then any AICall involving that AI — whether a normal single-AI call or a team
call — automatically triggers an audit once the call finishes.

## Goals

- Add an opt-in per-AI option that enables automatic auditing of finished AICalls.
- Cover both call types: `AssistanceTypeAI` (normal) and `AssistanceTypeTeam` (team).
- Trigger the audit automatically when the AICall reaches `StatusTerminated`.
- Make triggering **best-effort and fully decoupled** from call teardown: a failure to
  enqueue or run an audit must never affect call termination.
- Reuse the existing audit pipeline (`aiauditHandler.Create`) unchanged.

## Non-Goals

- No change to *how* an audit is computed (Gemini evaluation, scoring, message selection).
- No per-team-member granularity: a team call audits **all** participants if auto-audit
  applies (see "Team semantics" below). Finer control is out of scope for this iteration.
- No retroactive auditing of calls that finished before this feature shipped.
- No new user-facing API to trigger auto-audit; it is driven purely by the AI option.

## Decisions (locked during brainstorming)

1. **Option lives on the AI** (not the team). For team calls, the decision is the logical
   OR across all participating AIs' options ("any participating AI enabled → audit the
   whole call"). This reuses the existing `Create()` team behavior (one audit record per
   participant) without modification.
2. **The decision is frozen at call-creation time** into the AICall `Metadata`, not
   re-read at termination. VoIPbin already loads every participating AI's config when it
   builds the prompt snapshots at call start, so the flag is computed there. This mirrors
   the existing "config frozen at call start" snapshot semantics and avoids any AI-config
   lookups during teardown.
3. **The trigger is published to the queue** (fire-and-forget) via the existing
   `requesthandler`, rather than calling the audit handler in-process. Any `ai-manager`
   pod consumes the scheduled request and runs the audit. This keeps `aicallhandler`
   decoupled from `aiauditHandler` and adds no new dependency to inject.

## Design

### 1. AI option field

Add to `bin-ai-manager/models/ai/main.go` (`AI` struct), following the existing
`SmartTurnEnabled bool` field shape:

```go
AutoAICallAuditEnabled bool `json:"auto_aicall_audit_enabled,omitempty" db:"auto_aicall_audit_enabled"`
```

- Default `false` (opt-in).
- The verbose name (`AutoAICallAuditEnabled`, not `AutoAuditEnabled`) disambiguates it from
  other potential audit concepts in the AI config.

**The field must be settable through the public API end-to-end.** Note: `SmartTurnEnabled`
is an *incomplete* precedent — it appears only in the AI **response** schema
(`bin-openapi-manager/openapi/openapi.yaml:2137`, `AIManagerAI`), is **absent** from the
create/update **request** bodies (`bin-openapi-manager/openapi/paths/ais/main.yaml`,
`id.yaml`), and is not carried by `servicehandler.AICreate/AIUpdate` or
`requesthandler.AIV1AICreate/AIV1AIUpdate`. `VADConfig` is in the request body but is
dropped at the HTTP→servicehandler boundary. Net: neither is actually customer-settable. We
do **not** replicate that gap — `auto_aicall_audit_enabled` is added to both the request
bodies and the response schema and wired through every layer so it persists. (Fixing the
pre-existing `SmartTurnEnabled`/`VADConfig` drop is out of scope.)

Wiring through `requesthandler.AIV1AICreate` / `AIV1AIUpdate` is an **additive signature
change** to two methods used by all 37 consumers — consumers still compile (new trailing
param), but `bin-common-handler` changes require building every consumer per the
shared-library rule. The full layer list is in **Affected Files** below.

**Database migration** (`bin-dbscheme-manager`, Alembic, generated via `alembic revision` —
never hand-authored revision IDs): add a `auto_aicall_audit_enabled` `TINYINT(1) NOT NULL
DEFAULT 0` column to table `ai_ais` (matching the `smart_turn_enabled` column shape). The
`smart_turn_enabled` precedent migration used a plain `ALTER TABLE ... ADD ... DEFAULT 0`
with no `ALGORITHM`/`LOCK` clause; if an `ALGORITHM` is specified use `INSTANT` **alone**
(MySQL rejects `INSTANT` combined with any `LOCK` mode). The migration must deploy before
the code that selects the column.

**`table_ai_ais.sql` (mandatory, paired with the `db:` tag).** Add the same column to
`bin-ai-manager/scripts/database_scripts_test/table_ai_ais.sql`. `GetDBFields` reflects the
struct, so a `db:` tag without the matching test-schema column breaks **every** `ai_ais`
`SELECT` in `go test ./...`.

### 2. Freeze the flag into AICall metadata at creation

A new metadata key in `bin-ai-manager/models/aicall/main.go`:

```go
// MetaKeyAutoAuditEnabled is the Metadata map key (bool) recording whether this
// AICall should be auto-audited when it terminates. Frozen at call-creation time.
const MetaKeyAutoAuditEnabled = "auto_audit_enabled"
```

`buildPromptSnapshots()` (in `pkg/aicallhandler/start.go`) already resolves the single AI
(`AssistanceTypeAI`) or every team member's AI (`AssistanceTypeTeam`, via
`resolveAIForTeam`). Compute:

```
autoAudit = OR over all resolved participant AIs of ai.AutoAICallAuditEnabled
```

Store the result under `MetaKeyAutoAuditEnabled` in the AICall `Metadata` map, written
alongside `MetaKeyPromptSnapshots` at both creation sites (`start.go:613`, `start.go:668`).

**The OR must be computed from the same resolved AI set used to build the snapshots** — not
a second fetch. Change `buildPromptSnapshots` to return `([]aicall.PromptSnapshot, bool)`:
it already holds the single `*ai.AI` (`AssistanceTypeAI`) or the resolved member map
(`AssistanceTypeTeam`, via `resolveAIForTeam`), so it ORs `AutoAICallAuditEnabled` over
exactly that set and returns the flag. Update both call sites
(`startAIcallByRealtime`, `startAIcallByMessaging`) to capture and store the bool.

Team partial-resolution note: `resolveAIForTeam` is partial-failure tolerant (a member whose
AI can't be loaded is skipped), and `buildPromptSnapshots` degrades a total team-resolve
failure to empty snapshots. The OR is computed over the AIs that actually resolved; if none
resolve, `autoAudit` is `false`.

**Reference-type scope.** The flag is frozen for **every** AICall regardless of
`ReferenceType` (`call`, `conversation`, `task`) — both creation paths funnel through
`buildPromptSnapshots`. Any terminated AICall whose participating AI opted in is audited;
there is no special-casing of task/conversation calls in this iteration. (The user framed
this as "AI call," but applying it uniformly is simpler and the audit pipeline already
accepts any terminated AICall.)

### 3. Trigger on termination, via the queue

**New `requesthandler` method** in `bin-common-handler/pkg/requesthandler/ai_aiaudits.go`,
mirroring `AIV1AIcallTerminateWithDelay`:

```go
// AIV1AIAuditCreateWithDelay asks ai-manager to create audit job(s) for an aicall
// after a delay, fire-and-forget (no response awaited).
func (r *requestHandler) AIV1AIAuditCreateWithDelay(
    ctx context.Context, customerID uuid.UUID, aicallID uuid.UUID, language string, delay int,
) error {
    uri := "/v1/aiaudits"
    data := &amrequest.V1DataAIAuditsPost{
        CustomerID: customerID,
        AIcallID:   aicallID,
        Language:   language,
    }
    m, err := json.Marshal(data)
    if err != nil {
        return err
    }
    tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aiaudits", requestTimeoutDefault, delay, ContentTypeJSON, m)
    if err != nil {
        return err
    }
    return parseResponse(tmp, nil)
}
```

- Added to the `RequestHandler` interface in `main.go`; `mock_main.go` regenerated via
  `go generate`. Purely additive — existing consumers still compile.
- With `delay > 0`, `sendRequest` routes to `sendDelayedRequest`, which publishes a
  scheduled message and returns `(nil, nil)` immediately without awaiting processing.

**Hook in `ProcessTerminate`** (`bin-ai-manager/pkg/aicallhandler/process.go`), **after**
`UpdateStatus(ctx, id, aicall.StatusTerminated)` succeeds:

```
res := <the terminated aicall>
if enabled, _ := res.Metadata[aicall.MetaKeyAutoAuditEnabled].(bool); enabled {
    if err := h.reqHandler.AIV1AIAuditCreateWithDelay(ctx, res.CustomerID, res.ID, "", autoAuditTriggerDelay); err != nil {
        // best-effort: log and continue. Never fail termination.
        logrus.WithField(...).Errorf(...)
    }
}
```

- `h.reqHandler` is **already present** on `aicallHandler` (used for
  `FlowV1ActiveflowServiceStop`, etc.) — no new injection, no import-cycle risk.
- **Double-fire safety.** `ProcessTerminate` has four callers and can be invoked twice for
  one call (e.g. both `EventCMCallHangup` and `EventCMConfbridgeLeaved`). It returns early
  when the AICall is *already* `StatusTerminated` (`process.go:47-50`), **before** reaching
  this hook, so the normal second invocation never enqueues a second audit. A rare
  concurrent two-pod race could pass the early-return check on both before either commits and
  enqueue twice; the conditional dedupe in §4 makes that safe (at worst a redundant eval).
- `autoAuditTriggerDelay`: a package constant, proposed **1000 ms**. Its primary role is to
  select the async publish path (`delay > 0`). It is *not* needed for state visibility:
  `UpdateStatus` writes the terminated AICall to **shared Redis** synchronously before
  `ProcessTerminate` returns, and the consuming pod reads cache-first, so even a 0 ms delay
  would see `StatusTerminated`. The small delay is a harmless settle margin for trailing
  message persistence. Value is tunable; not load-bearing.
- `language` is `""` so `Create` applies its existing resolution (`aicall STT language →
  en-US`).
- Best-effort: the metadata read uses a safe type assertion (absent/wrong-type key → `false`),
  and any publish error is logged only.

### 4. Audit execution (unchanged)

The consuming `ai-manager` pod processes the scheduled request through the existing
`/v1/aiaudits` POST route → `aiauditHandler.Create(ctx, customerID, aicallID, "")`, which:

- Requires the AICall to be `StatusTerminated` (it is, by the time the delayed message fires).
- For `AssistanceTypeTeam`, lists participants and creates one audit per member; for
  `AssistanceTypeAI`, one audit.
- Enforces the 10-concurrent-audits-per-customer rate limit (over-limit → handled/logged,
  does not crash).
- Dedupes (conditionally): the audit row has a unique key on `(aicall_id, ai_id)`, and
  `AIAuditUpsert` is an `INSERT ... ON DUPLICATE KEY UPDATE`. A duplicate trigger fired
  while a prior audit for the same AI is **still `progressing` with identical inputs** is a
  no-op (no column changes → 0 rows affected). It is **not** an unconditional no-op: if the
  prior audit already completed and reset columns, a later duplicate flips the row back to
  `progressing` and re-runs. At the ~1000 ms trigger delay this is unlikely (Gemini eval has
  a 30 s window), and the only cost of a rare re-run is a redundant evaluation — never a
  correctness or termination problem.
- Resolves language and runs the Gemini evaluation in background goroutines, as today.

No changes to `aiauditHandler`, `geminiaudithandler`, or the audit DB schema.

## Data Flow

```
Call creation (start.go)
  buildPromptSnapshots() resolves participant AI config(s)
    autoAudit = OR(ai.AutoAICallAuditEnabled)
    AICall.Metadata[MetaKeyAutoAuditEnabled] = autoAudit
  AICall persisted (initiating)
        │
        ▼  ... call proceeds ...
Call ends → ProcessTerminate()
  ... stop activeflow / pipecat / confbridge ...
  UpdateStatus(StatusTerminated)  → sets TMEnd, publishes EventTypeStatusTerminated (webhook)
        │
        ▼  if Metadata[MetaKeyAutoAuditEnabled] == true
  reqHandler.AIV1AIAuditCreateWithDelay(customerID, aicallID, "", 1000ms)   [fire-and-forget]
        │  (scheduled message on QueueNameAIRequest)
        ▼
Any ai-manager pod consumes → POST /v1/aiaudits → aiauditHandler.Create()
        │
        ▼
  one audit per participant → Gemini evaluation → audit records updated
```

## Edge Cases

| Case | Behavior |
|------|----------|
| AICall created before this feature (in flight) | Metadata key absent → read as `false` → no audit. |
| AI option toggled mid-call | Ignored; the flag is frozen at call start (consistent with prompt snapshots). |
| Team call, only some members enabled | OR semantics → whole call audited (one record per participant). |
| Team call, no member enabled | `false` → no audit. |
| `ProcessTerminate` fires twice (hangup + confbridge-leaved) | Second call early-returns before the hook; no second enqueue (rare concurrent race deduped per §4). |
| Non-`call` reference types (`conversation`, `task`) | Treated identically — audited if a participating AI opted in. |
| Publish/enqueue fails | Logged; termination unaffected. |
| Audit already in progress for an AI | `AIAuditUpsert` deduped via the `(aicall_id, ai_id)` unique key while still `progressing` (see §4). |
| Customer over audit rate limit | Existing `Create` rate-limit handling; no termination impact. |
| No messages / Gemini unavailable | Surfaced in the audit record status/error only. |

## Testing

**`bin-common-handler`**
- `Test_AIV1AIAuditCreateWithDelay`: asserts a delayed publish to the AI request queue with
  the correct URI/method/body (mirrors `AIV1AIcallTerminateWithDelay` test).
- Regenerate `mock_main.go`; ensure all consumers still build (`go build ./...`).

**`bin-ai-manager` (unit)**
- Snapshot flag computation in `buildPromptSnapshots`:
  - single AI enabled → `true`; single AI disabled → `false`;
  - team any-enabled → `true`; team all-disabled → `false`; team partial-resolve → OR over
    resolved set.
- `ProcessTerminate`:
  - metadata flag `true` → calls `reqHandler.AIV1AIAuditCreateWithDelay` once with expected args;
  - flag `false`/absent → never calls it;
  - publish returns error → `ProcessTerminate` still completes successfully (best-effort);
  - already-`StatusTerminated` AICall → early return at `process.go:47-50`, audit **not**
    enqueued (covers the double-fire short-circuit).
- Existing terminate/create tests updated for the new metadata key where they assert on
  `Metadata`.

**api-validator** (`monorepo-monitoring/api-validator`)
- AI create with `auto_aicall_audit_enabled: true` → response echoes it; AI update toggling
  the field → read-back reflects the change. Read/CRUD only — no real calls, no audit cost.

**RST docs**
- Update the AI struct doc (`*_struct_*`) and AI overview/tutorial to document
  `auto_aicall_audit_enabled`. Clean rebuild (`rm -rf build && python3 -m sphinx -M html
  source build`) and commit the built HTML.

## Affected Files

**`bin-ai-manager`**
- `models/ai/main.go` — add `AutoAICallAuditEnabled bool` field.
- `models/ai/field.go` — add `FieldAutoAICallAuditEnabled` DB-field constant.
- `models/ai/webhook.go` — add field to `WebhookMessage` + `ConvertWebhookMessage()`.
- `models/aicall/main.go` — add `MetaKeyAutoAuditEnabled` constant.
- `pkg/aicallhandler/start.go` — `buildPromptSnapshots` returns the OR flag; both creation
  sites store it under `MetaKeyAutoAuditEnabled`.
- `pkg/aicallhandler/process.go` — trigger hook after `UpdateStatus(StatusTerminated)`.
- `pkg/aihandler/main.go` — add param to `Create`/`Update` interface.
- `pkg/aihandler/db.go` — thread param through create/update field maps.
- `pkg/aihandler/mock_main.go` — regenerated (`go generate`).
- `pkg/listenhandler/models/request/ais.go` — add field to `V1DataAIsPost` + `V1DataAIsIDPut`.
- `pkg/listenhandler/v1_ais.go` — pass field at create + update routes.
- `scripts/database_scripts_test/table_ai_ais.sql` — add the column.
- `docs/domain.md` — sync the AI entity (PostToolUse hook warns on `models/**` changes).

**`bin-common-handler`**
- `pkg/requesthandler/ai_aiaudits.go` — add the new `AIV1AIAuditCreateWithDelay` method.
- `pkg/requesthandler/ai_ais.go` — add `autoAICallAuditEnabled` param to `AIV1AICreate` +
  `AIV1AIUpdate` (additive) and into the `V1DataAIsPost`/`V1DataAIsIDPut` payloads.
- `pkg/requesthandler/main.go` — add `AIV1AIAuditCreateWithDelay` to the `RequestHandler`
  interface **and** update the `AIV1AICreate`/`AIV1AIUpdate` signatures (three edits).
- `pkg/requesthandler/mock_main.go` — regenerated.
- Build all consumers (shared-library rule).

**`bin-openapi-manager`** (the OpenAPI schema source — *not* `bin-api-manager`)
- `openapi/paths/ais/main.yaml` — add `auto_aicall_audit_enabled` to the POST request body.
- `openapi/paths/ais/id.yaml` — add it to the PUT request body.
- `openapi/openapi.yaml` — add it to the `AIManagerAI` response schema.
- `gens/models/gen.go` — regenerated via `go generate ./...`.

**`bin-api-manager`**
- `gens/openapi_server/gen.go` — regenerated from the updated `bin-openapi-manager` schema
  (`go generate ./...`). (`openapi/config_server/config.generate.yaml` is the oapi-codegen
  *tool* config, not the schema — no manual schema edit there.)
- `server/ais.go` — read `req.AutoAicallAuditEnabled` in `PostAis`/`PutAisId` and pass it on.
- `pkg/servicehandler/ai.go` + `pkg/servicehandler/main.go` (interface) +
  `pkg/servicehandler/mock_main.go` — add param to `AICreate`/`AIUpdate`.
- `docsdev/source/*` — RST struct/overview/tutorial for the AI resource; clean rebuild +
  commit built HTML.

**`bin-dbscheme-manager`**
- `bin-manager/main/versions/<generated>.py` — Alembic migration for
  `ai_ais.auto_aicall_audit_enabled` (created, **not** applied by the agent).

## Verification

Per monorepo rules, run the full workflow in each changed Go service before committing:

```
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Run in `bin-common-handler` (interface + mock change → also build all consumers),
`bin-ai-manager`, `bin-openapi-manager` (after editing the schema and `go generate`), and
`bin-api-manager` (after `go generate` regenerates `gens/openapi_server/gen.go`).
`bin-dbscheme-manager` migration is created via `alembic revision` and committed but
**not** applied by the agent.

## Rollout / Rollback

- The feature is inert until a customer sets `auto_aicall_audit_enabled = true` on an AI, so
  shipping the code is low risk (default off).
- Migration must deploy before the code that selects the new column.
- Rollback: revert the PR. Calls created while it was enabled simply carry a now-unused
  metadata key; no cleanup required.

## Open Questions

- Trigger delay value — defaulted to **1000 ms** (tunable, not load-bearing).
- Field/column name — defaulted to **`auto_aicall_audit_enabled`**.
- Whether to also fix the pre-existing `SmartTurnEnabled`/`VADConfig` requesthandler-drop
  gap — **out of scope** here; flagged for a separate change.
