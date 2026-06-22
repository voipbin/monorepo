# Timeline Activeflow AI Analysis

Status: Draft (v3)
Services: bin-timeline-manager (analysis owner + storage), bin-ai-manager (generic LLM gateway), bin-common-handler (requesthandler client)
Date: 2026-06-23

## 1. Problem Statement

The admin Timeline view (`/timeline/activeflows/:id`, `square-admin` `execution_viewer.js`) renders everything about a finished activeflow by fetching `aggregated-events`, `sip-analysis`, and PCAP on every visit and computing all derivations client-side. There is no persisted, human-readable interpretation of "what happened in this activeflow". Today an operator must read the raw event/SIP timeline and reason manually every time: which resources were used (call, conferencecall, sms, email, ...), what events fired, what the conversation contained, and whether there was a problem and where.

This design adds an on-demand AI analysis that:
- Runs only on an **ended** activeflow.
- Ingests the activeflow's full aggregated event set + correlation graph (+ available content such as transcripts).
- Produces a **structured JSON** verdict (resources used, event narrative, problem detection with severity and resolved evidence event references).
- Is **persisted once** and shown on revisit without recomputation, with a **manual re-analysis** path to overwrite.

The analysis is owned and stored by timeline-manager (its domain, its format). The LLM connection is owned by ai-manager via a new **generic internal LLM gateway** (prompt + data + json schema -> structured JSON). timeline-manager calls that gateway.

## 2. Scope

### In scope (Phase 1)
- **bin-ai-manager**: a generic internal-only RPC `POST /v1/services/type/analysis` that takes `{prompt, data, schema, schema_name, model?}`, calls `engine_openai_handler.Send` with `ResponseFormat=json_schema` (`Strict:true`), and returns the structured JSON (raw `json.RawMessage`) plus finish-reason/truncation and token usage. Synchronous request/response (the caller owns async). NOT exposed via api-manager / OpenAPI / RST.
- **bin-timeline-manager**: new MySQL dependency + `timeline_analyses` table; a new `requesthandler` capability to call ai-manager; `analysishandler` that orchestrates a **multi-stage** analysis chain over an ended activeflow and persists the structured result; new internal RPC endpoints (trigger / get / list / re-analyze / delete).
- **bin-common-handler**: `AIV1ServiceTypeAnalysisRun` requesthandler method + interface + mock; timeline-manager's own requesthandler wiring (timeline-manager currently has no requesthandler at all).

### Out of scope (Phase 1, deferred)
- **square-admin UI** (analysis panel, trigger button, result rendering). Separate frontend task after backend lands. Backend ships first and is exercised via RPC/CLI.
- **Auto-trigger** on activeflow end. Unbounded LLM cost; Phase 2 minimum, requires a cost estimate first.
- **External REST exposure** of either the gateway or the analysis (explicit pchero decision: gateway internal-only). A customer-facing analysis read endpoint can be a later phase if desired.
- **on_end_flow** trigger and **webhook events** for analysis completion. Not requested; analysis is a pull/diagnostic feature, not an event source. Listed in Open Questions.

### Rationale
- ai-manager already owns LLM credentials, model config, and a generic `engine_openai_handler.Send(*openai.ChatCompletionRequest)` whose `ResponseFormat` field supports `json_schema`. We expose that capability as an internal gateway rather than duplicating LLM plumbing in timeline-manager. The `aiaudithandler` (LLM-as-a-Judge) is the proven async + structured-output template.
- timeline-manager owns the events and the correlation graph and is the natural owner of "an activeflow's analysis". It currently uses ClickHouse (append-only OLAP), which is unsuitable for a single mutable per-activeflow record, so a MySQL OLTP store is added (pchero-approved).

## 3. Domain Model

### 3.1 bin-ai-manager: generic gateway (no new persisted entity)

The gateway is stateless: it does not persist anything. Request/response domain types only.

```go
// bin-ai-manager/models/analysis/analysis.go
package analysis

// Request is the generic LLM gateway input. Internal callers only.
type Request struct {
    Prompt     string          `json:"prompt"`               // system/instruction text
    Data       json.RawMessage `json:"data"`                 // arbitrary caller-supplied payload, rendered into the user message
    Schema     json.RawMessage `json:"schema"`               // JSON Schema for response_format=json_schema (required)
    SchemaName string          `json:"schema_name"`          // required by OpenAI json_schema (response_format.json_schema.name)
    Model      string          `json:"model,omitempty"`      // optional; must be in the allowed model set, else default
}

// Response carries the structured LLM output and accounting.
type Response struct {
    Result       json.RawMessage `json:"result"`         // the schema-conformant JSON object
    Model        string          `json:"model"`          // model actually used
    FinishReason string          `json:"finish_reason"`  // "stop" / "length" / ... so the caller can detect truncation BEFORE Validate()
    Truncated    bool            `json:"truncated"`      // true when FinishReason=="length" (output cut, JSON likely invalid)
    PromptTokens int             `json:"prompt_tokens"`
    OutputTokens int             `json:"output_tokens"`
}
```

Gateway guards (abuse control, since it is generic):
- Reachable only over the internal RPC queue `bin-manager.ai-manager.request`. Not added to api-manager routing, OpenAPI, or RST.
- `Model` validated against an allowed set (`analysisAllowedModels`); unknown -> default model `analysisDefaultModel` (config: `ENGINE_MODEL_ANALYSIS`, default a cost-appropriate model). No arbitrary model passthrough. **The allowed set MUST be a superset of every timeline-manager stage model (§6.3); a startup assertion / unit test enforces `{stage1,stage2,stage3} ⊆ analysisAllowedModels` so config drift cannot fail a chain mid-run (review H4).**
- `Schema` required and non-empty; `SchemaName` required (OpenAI mandates `response_format.json_schema.name`). Missing -> 400-class error (the whole point is shape-enforced output).
- **`json_schema` plumbing (review M1):** set `Strict: true` on `ChatCompletionResponseFormatJSONSchema`; the caller-supplied schema MUST declare `additionalProperties:false` and mark all keys required (OpenAI strict-mode requirement). The gateway rejects a schema missing these (cheap structural pre-check) rather than letting OpenAI 400 at request time.
- Input size cap: `Data` + `Prompt` byte length bounded (`analysisMaxInputBytes`, e.g. 256 KiB) -> reject oversize before hitting the LLM. **The caller (timeline-manager) is responsible for fitting under this via the truncation strategy in §6.1; the gateway cap is the backstop, not the primary control (review H2).**
- **Output ceiling vs cost ceiling are SEPARATE (review H4).** The gateway sets `max_tokens = analysisMaxOutputTokens` sized ABOVE the worst-case structured verdict (so a normal large verdict never hits it), purely as a runaway-output guard. It is NOT the cost-control lever: clipping the output produces `finish_reason=length` -> `failed` (§6.5), which would defeat cost control by failing exactly the big analyses it targets. Cost control is instead the run-count/cadence cap (Q7), not output clipping.
- Per-call LLM timeout via `context.WithTimeout`.
- **Response shape note (review M4):** this endpoint deliberately returns a bespoke `analysis.Response`, NOT the `service.Service` shape that the existing `/v1/services/type/{summary,task,aicall}` endpoints return. The new `bin-common-handler` `AIV1ServiceTypeAnalysisRun` unmarshals `analysis.Response`. Called out so reviewers do not expect `service.Service`.

### 3.2 bin-timeline-manager: Analysis entity

```go
// bin-timeline-manager/models/analysis/analysis.go
package analysis

type Status string

const (
    StatusProgressing Status = "progressing" // running the stage chain
    StatusCompleted   Status = "completed"   // result persisted
    StatusFailed      Status = "failed"      // chain failed; error recorded
)
// NOTE: no StatusNone="" (zero-value hazard, VoIPBin convention).

type Analysis struct {
    ID           uuid.UUID       `json:"id"`
    CustomerID   uuid.UUID       `json:"customer_id"`
    ActiveflowID uuid.UUID       `json:"activeflow_id"`

    Status Status          `json:"status"`
    Result json.RawMessage `json:"result"`         // structured verdict (see 6.4); includes "version"
    Model  string          `json:"model"`          // model used for the final (diagnostic) stage
    Error  string          `json:"error"`          // failure reason when Status=failed

    TMCreate string `json:"tm_create"`
    TMUpdate string `json:"tm_update"`
    TMDelete string `json:"tm_delete"`
}
```

Lifecycle:
```
(trigger on ended activeflow) -> progressing
    -> [stage chain succeeds] -> completed (Result set)
    -> [stage chain fails]    -> failed   (Error set, Result empty)
(manual re-analyze) -> overwrite same row -> progressing -> completed/failed
```

One row per activeflow (`UNIQUE(activeflow_id, tm_delete)`). Re-analysis overwrites in place (status back to progressing, then result replaced). Customer ownership carried on the row for read filtering.

## 4. Database Schema (bin-timeline-manager, NEW MySQL — shared instance, Alembic)

timeline-manager has no MySQL today (ClickHouse only; its `dbhandler` is ClickHouse-typed). This adds a **second, distinct persistence engine**. To avoid confusion with the ClickHouse handler, the MySQL access lives in a **separate handler package** (e.g. `pkg/analysisdbhandler` or `pkg/dbhandler/mysql`) with its own connection pool, lifecycle, config (`DATABASE_DSN`, pool sizing), and its own `go generate` mock target. The existing ClickHouse `dbhandler` is untouched. (Review #4 / blast-radius.)

**Migration ownership (review C2/#2 — corrected from v1).** The monorepo manages ALL MySQL schema via **Alembic** in `bin-dbscheme-manager/bin-manager` against the **shared** MySQL instance. The v1 Open Question "Alembic vs golang-migrate" is resolved: **Alembic, shared DB.** timeline-manager's in-process `runMigrations()` stays ClickHouse/golang-migrate only and does NOT own this table. The `timeline_analyses` table is added to the shared Alembic tree; timeline-manager only reads/writes rows, never issues DDL. This is a shared-DB table (not a new timeline-private database).

```sql
-- Added to bin-dbscheme-manager/bin-manager Alembic revision (shared MySQL instance).
CREATE TABLE timeline_analyses (
  id            BINARY(16)   NOT NULL,
  customer_id   BINARY(16)   NOT NULL,
  activeflow_id BINARY(16)   NOT NULL,

  status        VARCHAR(32)  NOT NULL,
  result        JSON         NULL,
  model         VARCHAR(255) NOT NULL DEFAULT '',
  error         TEXT         NULL,

  tm_create     DATETIME(6)  NOT NULL,
  tm_update     DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',
  tm_delete     DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX idx_timeline_analyses_activeflow ON timeline_analyses (activeflow_id, tm_delete);
CREATE INDEX idx_timeline_analyses_customer_create   ON timeline_analyses (customer_id, tm_create);
CREATE INDEX idx_timeline_analyses_customer_delete   ON timeline_analyses (customer_id, tm_delete);
```

Notes:
- `tm_delete DEFAULT '9999-01-01 00:00:00.000000'` (soft-delete convention). Live rows ALWAYS carry this constant sentinel (never zero-value), so `UNIQUE(activeflow_id, tm_delete)` means "at most one live analysis per activeflow" and a soft-deleted historical row (distinct `tm_delete`) does not collide.
- `result` JSON includes a top-level `"version"` for schema evolution.
- **Deploy ordering (review M5):** the Alembic migration creating `timeline_analyses` MUST run before bin-timeline-manager's new MySQL dbhandler serves `Start`/`Get`/etc. Standard monorepo ordering (dbscheme-manager migration precedes dependent service rollout) applies; called out because timeline-manager is gaining its first MySQL table.

## 5. Handler Interface

### 5.1 bin-ai-manager analysisHandler (gateway)

```go
// pkg/analysishandler/main.go
type AnalysisHandler interface {
    Run(ctx context.Context, req *analysis.Request) (*analysis.Response, error)
}
```

Flow (`Run`):
1. Validate: `Schema` non-empty; `SchemaName` non-empty; schema declares `additionalProperties:false` + all-required; input bytes <= cap; resolve `Model` (allowed-set or default).
2. Build `openai.ChatCompletionRequest{ Model, MaxTokens: analysisMaxOutputTokens, Messages: [system=Prompt, user=string(Data)], ResponseFormat: {Type: json_schema, JSONSchema: {Name: SchemaName, Schema: Schema, Strict: true}} }`.
3. `context.WithTimeout(ctx, analysisLLMTimeout)`, call `engineOpenaiHandler.Send`.
4. Extract choice content + `finish_reason` (already schema-conformant JSON), wrap into `Response` with `Truncated = finish_reason=="length"` and token usage.
5. Errors: LLM error / timeout / empty choice -> wrapped error to caller. No persistence, no retry here (caller decides).

This is synchronous. The async lifecycle lives in the timeline-manager caller (it owns the progressing/completed record).

### 5.2 bin-timeline-manager analysisHandler (orchestrator + store)

```go
// pkg/analysishandler/main.go
type AnalysisHandler interface {
    // Trigger: validates ended-state + ownership, creates progressing row, kicks the async chain.
    Start(ctx context.Context, customerID, activeflowID uuid.UUID, reanalyze bool) (*analysis.Analysis, error)
    // Reads/Delete ALL take customerID and enforce ownership (review C1).
    Get(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error)
    GetByActiveflowID(ctx context.Context, customerID, activeflowID uuid.UUID) (*analysis.Analysis, error)
    List(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[analysis.Field]any) ([]*analysis.Analysis, error)
    Delete(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error)
}
```

**Ownership on every read/delete (review C1 — IDOR fix).** `Get`/`GetByActiveflowID`/`Delete` resolve the row, then compare `row.CustomerID == customerID`; on mismatch return the SAME not-found error as a truly-absent row (no existence oracle). `List` ALWAYS injects `customer_id = <caller>` into the filter set server-side and never trusts a client-supplied `customer_id` as the authority. The internal RPC endpoints (§7) therefore carry `customer_id` on every analysis operation, not just the trigger.

`Start` internal flow:
1. **Ended gate (mandatory).** Resolve the activeflow via `reqHandler.FlowV1ActiveflowGet(activeflowID)`. If `Status != ended` -> reject (`ErrActiveflowNotEnded`). This is also the **ownership** source: `activeflow.CustomerID` must equal the requesting `customerID` (timeline events carry no customer_id; flow-manager is the authority, same pattern as the correlation IDOR fix). Mismatch and not-found both masked as not-found.
2. **Existing-record policy (with race handling — review H3/#5).**
   - `GetByActiveflowID` exists and `reanalyze=false` and status `completed` -> return existing (idempotent, no LLM).
   - exists and status `failed` and `reanalyze=false` -> return existing failed (caller decides; re-run requires `reanalyze=true`).
   - exists and `reanalyze=true` -> **conditionally** reset row to `progressing` via `UPDATE ... SET status='progressing', result=NULL, error='' WHERE id=? AND status != 'progressing'`. If 0 rows affected, another reanalyze already won the transition -> return the in-flight row (no second goroutine, no double-spend — review H2). On success, continue.
   - exists and status `progressing` -> return existing (a run is in flight; do not double-spend). `reanalyze=true` while `progressing` is also a no-op return of the in-flight row (does NOT restart; documented for callers). The conditional UPDATE above makes the completed->progressing transition atomic so two concurrent reanalyze calls cannot both spawn a job.
   - not exists -> create `progressing` row. **The create wraps the `UNIQUE(activeflow_id, tm_delete)` violation: if a concurrent trigger won the insert, catch the duplicate-key error, re-SELECT, and return the in-flight `progressing` row instead of surfacing a 500.** (Check-then-act race made safe by the unique constraint + dup-key catch.)
3. **Async chain** (goroutine, `aiaudithandler` pattern): bounded by a `semaphore`, `context.WithTimeout(context.Background(), analysisJobTimeout)`, `defer recover()+debug.Stack()`, final write on a fresh 10s `context.Background()` timeout so the result persists even if the parent ctx is gone. **The final write-back guards on `tm_delete = <sentinel>` so a row soft-deleted while the goroutine ran is NOT resurrected (review #8).** The persisted `Error` string is a sanitized, operator-safe message (no raw provider errors or stack traces; the stack goes to logs only — review L2).
4. Return the `progressing` row immediately to the caller (poll `Get`/`GetByActiveflowID` for completion).

The async chain (see 6) collects inputs, runs the multi-stage LLM gateway calls, validates the final JSON, then `Update` the row to `completed`+result or `failed`+error.

## 6. LLM Logic (multi-stage chain)

pchero requirement: analyze everything, but split into **stages**, each with its own prompt; later stages consume earlier stage output.

### 6.1 Input collection (no LLM)
**Strict ordering (review C1/M2 — the index space must be frozen before anything references it).** The pipeline is: (1) collect → (2) reduce/truncate → (3) freeze + index the FINAL list → (4) deterministic pre-extraction tags indices on that frozen list → (5) send to LLM → (6) raw-output validation → (7) resolve indices → (8) persist. An event's `evidence_index` is assigned ONCE, on the post-reduction frozen list, and never renumbered. Pre-extraction (§6.2), the LLM-visible list, `Validate()` (§6.4), and the resolver all reference that single frozen list, so a cited index can never map to a different event.

- **Events**: `eventHandler.AggregatedList(ctx, activeflow_id, pageToken, pageSize)` is **paginated** (cursor = last row timestamp). The orchestrator loops pages to assemble the full event set, bounded by `analysisMaxEvents` (hard cap, e.g. 5000) and `analysisMaxPages` so a pathological flow cannot spin forever (review H1). **Same-timestamp page-boundary handling (review M2):** ClickHouse timestamps are not unique, so the page cursor uses `>=` and the assembler de-duplicates boundary rows by a stable composite key `(timestamp, event_type, resource_id, hash(data))`; this keeps the canonical list complete AND deterministic across re-analysis (otherwise pre-extraction could miss a boundary error event and two runs could disagree).
- The assembled, de-duplicated, reduced slice is the **canonical event list** for this run, held in Go memory. Go assigns each entry a 0-based `evidence_index` AFTER reduction (review C1). The list is passed to the LLM as `[ {idx, timestamp, event_type, publisher, resource_id, summary_of_data}, ... ]`. Stage 3 (or the SMALL combined call) cites `evidence_index` values.
- **Correlation graph**: the existing correlation resolution (resource graph grouped by publisher) for the activeflow's resources -> "which resources were used".
- **Content (best-effort)**: for call/conference resources present, pull available transcripts via `reqHandler.TranscribeV1TranscriptList` when present. Missing content is non-fatal (analyze what exists). PCAP/RTP audio is NOT fetched server-side in Phase 1 (heavy; client already does it). MOS/quality, if already present as events, is included as event data.
- **Truncation strategy instead of hard-fail (review H2-input).** Large flows (long transcripts + many events) can exceed the timeline-side reduce target `analysisReduceTargetBytes` (which MUST be strictly smaller than the gateway's `analysisMaxInputBytes` cap minus prompt+schema overhead — review M3; the two are distinct constants in two services). Reduction is deterministic and happens in step (2), BEFORE indexing, in priority order: (1) ALWAYS keep the deterministic inventory + ALL error-class events from §6.2 (these are never elided, so their frozen index is stable); (2) cap transcript text per resource; (3) sample/elide low-signal repetitive non-error events; (4) set `"input_reduced": true` in the result so the operator knows the verdict was on a reduced view. Stage 1 inventory must always survive truncation.

### 6.2 Deterministic pre-extraction (no LLM) — hybrid guard
Before the LLM stages, extract decision-deterministic signals in Go and pass them as structured facts:
- resource inventory with counts per publisher/type (from correlation graph).
- error-class signals: `*_hangup` reasons, any `*_error`/`*_failed` event types, abnormal activeflow termination, MOS-below-threshold events if present. Each error signal carries its `evidence_index` so Stage 3 can cite it directly.
This keeps problem-detection grounded (the LLM narrates and contextualizes signals rather than inventing them), reducing hallucination and cost.

### 6.3 Stages (adaptive — review #6)
Each stage = one gateway call, own prompt, own json_schema. **Staging is adaptive, not unconditional, to avoid paying 3x cost/latency on a trivial activeflow.** A size threshold (`analysisStageThresholdEvents` + transcript length over `analysisShortTranscriptRunes`) decides:
- **Small activeflow (below threshold, short/no transcripts):** a **single** gateway call with a richer combined schema producing the full §6.4 verdict directly. One LLM round-trip. **This combined call uses `ENGINE_MODEL_ANALYSIS_STAGE3` (the best/diagnostic model) so it too is covered by the startup allow-set assertion (review H3) — no separate untracked model.**
- **Large activeflow (at/above threshold, or long transcripts):** the full 3-stage chain:
  - **Stage 1 — Inventory.** Prompt: identify resources/channels used and the chronological event outline. Input: deterministic inventory + indexed event outline. Output: `{ resources_used:[{type,count}], event_outline:[{evidence_index, label}] }` (the outline carries per-event indices so later stages can cite non-error events too — review L5). Model: cheap (`ENGINE_MODEL_ANALYSIS_STAGE1`).
  - **Stage 2 — Content.** Prompt: summarize what was communicated and intent/outcome. Input: Stage 1 **structured output** (NOT raw events again — avoids token duplication, review #6) + transcripts/message content. Output: `{ interactions:[{resource_type, summary}], overall_narrative }`. Model: capable (`ENGINE_MODEL_ANALYSIS_STAGE2`).
  - **Stage 3 — Diagnosis.** Prompt: given inventory, narrative, and the deterministic error signals, determine problems/where/severity, citing `evidence_index` values (from the error signals OR the Stage 1 outline). Input: Stage 1+2 structured output + deterministic error signals (with indices) + the error-class events. Output: the final verdict (6.4). Model: best (`ENGINE_MODEL_ANALYSIS_STAGE3`).

A stage failure fails the whole chain (`failed` + error). Stage prompts are constants in timeline-manager (`pkg/analysishandler/prompts.go`). All stage models AND the SMALL combined model MUST be in the gateway allow-set (§3.1, review H3/H4). Token usage is summed across stages for the `timeline_analysis` cost log.

### 6.4 Final result JSON (persisted in `result`)

**Two-phase validation (review H1).** The gateway returns raw LLM JSON containing integer `evidence_index` arrays. timeline-manager runs, in order: (a) **raw-output validation** on the LLM JSON — enum membership (`overall_status`, `severity`), non-empty evidence on every non-`ok` issue, and **every cited `evidence_index` within range of the frozen canonical list** (hallucination guard, review C2); fail any -> `failed`, do not persist. Then (b) **resolution** — replace each `evidence_index` with the concrete `{event_type, timestamp, resource_id}` tuple. Then (c) **deterministic overwrite** — `resources_used` is replaced with the Go-computed inventory counts from §6.2, NOT trusted from the LLM (review M1; the LLM may narrate but must not invent counts). The persisted shape below has no `evidence_index` left (already resolved), which is why range-checking happens in phase (a) on the raw output, not on the persisted object.

```json
{
  "version": 1,
  "overall_status": "warning",
  "input_reduced": false,
  "resources_used": [
    {"type": "call", "count": 2},
    {"type": "conferencecall", "count": 1},
    {"type": "transcribe", "count": 2}
  ],
  "narrative": "Two inbound calls joined a conference; ...",
  "issues": [
    {
      "severity": "warning",
      "area": "media",
      "summary": "call A MOS degraded to 2.8",
      "evidence": [
        {"evidence_index": 42, "event_type": "call_hangup", "timestamp": "2026-06-23T...", "resource_id": "<call-id>"}
      ]
    }
  ]
}
```
- `overall_status` enum: `ok` / `warning` / `error` (holistic, not derived by averaging).
- `issues[].severity` enum: `info` / `warning` / `error`.
- `issues[].evidence`: **mandatory non-empty for any non-`ok` issue**. The resolved tuple is shown for human readability; the original `evidence_index` is ALSO persisted alongside it (review M4) so a downstream UI can highlight the exact frozen-list event without re-deriving from the non-unique `(event_type,timestamp,resource_id)` triple. Empty issues array when `overall_status=ok`.
- `input_reduced`: true when input was reduced per §6.1 (renamed from `truncated` to avoid collision with the gateway's output-`Truncated` flag — review L1).
- `resources_used`: Go-computed (phase c), authoritative.

### 6.5 Failure handling matrix
| Failure | Behavior |
|---|---|
| activeflow not ended | reject `Start` (`ErrActiveflowNotEnded`), no row, no LLM |
| activeflow not owned by customer | reject as not-found (IDOR mask) |
| read/delete on another customer's analysis | not-found (IDOR mask, review C1) |
| input collection error (events) | `failed` + error |
| input over size cap | truncate per §6.1 (NOT fail), mark `truncated:true` |
| gateway call error/timeout (any stage) | `failed` + error |
| gateway returns `truncated`/`finish_reason=length` | `failed` + error (output JSON unreliable) |
| final JSON fails `Validate()` (enum / evidence / index range) | `failed` + error |
| concurrent `Start` losing the unique race | catch dup-key, return in-flight progressing row |
| panic in chain | recover -> `failed` + error, stack logged |

## 7. REST API

None for customers. Phase 1 is internal RPC only (pchero decision). Internal RPC over `bin-manager.timeline-manager.request`. **Every analysis operation carries `customer_id` for ownership enforcement (review C1), not just the trigger:**

| Method/URI | Purpose |
|---|---|
| `POST /v1/analyses` | Trigger analysis. Body `{customer_id, activeflow_id, reanalyze}`. Returns the (progressing or existing) record. |
| `GET /v1/analyses/<uuid>?customer_id=` | Get one analysis by id; ownership-checked, masked not-found on mismatch. |
| `GET /v1/analyses?customer_id=&activeflow_id=&status=&page_token=&page_size=` | List; `customer_id` is mandatory and server-injected as the authority filter. |
| `DELETE /v1/analyses/<uuid>?customer_id=` | Soft-delete; ownership-checked. |

(The admin UI later calls these via api-manager only if/when a Phase 2 customer-facing read endpoint is added. Phase 1 ships backend RPC + CLI exercise.)

ai-manager gateway internal RPC:

| Method/URI | Purpose |
|---|---|
| `POST /v1/services/type/analysis` | Generic gateway: `{prompt, data, schema, schema_name, model?}` -> `{result, model, finish_reason, truncated, prompt_tokens, output_tokens}`. Internal only. |

## 8. Webhook Events

None in Phase 1. Analysis is a diagnostic pull feature, not an event source. (Open Question: emit `timeline_analysis_completed` if a future UI wants push.)

## 9. Flow Variable Integration

None. No on_end_flow in Phase 1 (analysis is operator-facing, not a flow step). Listed in Open Questions.

## 10. RabbitMQ Integration

- timeline-manager gains a **requesthandler** (it has NONE today — this is its first outbound RPC client; it was a read-only ClickHouse service). Wiring blast radius (review #3): a publisher `ServiceName` for timeline-as-caller; `NewRequestHandler` registers Prometheus collectors on construction, so verify no duplicate-registration panic against the existing `initProm`; circuit-breaker + RPC timeout config become part of timeline's runtime; three new outbound dependencies mean new failure modes (downstream-down, CB-open) on what was a pure-local read path. Calls:
  - `FlowV1ActiveflowGet` (ended-gate + ownership) — bin-flow-manager.
  - `AIV1ServiceTypeAnalysisRun` (the gateway) — bin-ai-manager (new requesthandler method in bin-common-handler).
  - `TranscribeV1TranscriptList` (content) — bin-transcribe-manager.
- bin-common-handler: add `AIV1ServiceTypeAnalysisRun(ctx, *analysis.Request) (*analysis.Response, error)` to the requesthandler interface + impl + regenerate mock. (3+ consumers rule is satisfied as a method on the existing ai-manager client surface.)
- ai-manager listenhandler: route `POST /v1/services/type/analysis` -> `analysisHandler.Run`.

## 11. Observability

bin-ai-manager gateway:
- `analysis_gateway_run_total{model}` counter.
- `analysis_gateway_run_duration_seconds{model}` histogram.
- token counts logged per call (cost visibility).

bin-timeline-manager:
- `timeline_analysis_start_total{result}` counter (result = progressing/reused).
- `timeline_analysis_done_total{status}` counter (completed/failed).
- `timeline_analysis_duration_seconds` histogram (full chain).
- per-stage debug logs with trace id propagated into the async goroutine (the goroutine uses `context.Background()` + explicit trace id, not the RPC ctx, so logs correlate after the response is sent).

## 12. Security & Compliance

- **Ownership**: every `Start` resolves `FlowV1ActiveflowGet(activeflow_id).CustomerID == customerID`; mismatch and not-found both masked as not-found (no existence oracle). `Get`/`GetByActiveflowID`/`List`/`Delete` all take `customer_id` and enforce row ownership with the same masked not-found (§5.2).
- **Gateway is internal-only**: not on api-manager/OpenAPI/RST; only internal managers can reach the RPC queue. Generic prompt+data is therefore not a customer-exposed LLM injection surface. Model passthrough is restricted to an allowed set; input size capped.
- **PII / external LLM**: transcripts and event data (which may contain PII) are sent to the external LLM provider (OpenAI/compatible) via the gateway, exactly as the existing `summaryhandler`/`aiaudithandler` already do. This is consistent with current platform behavior but must be acknowledged. **Flag for CEO/CTO**: confirm activeflow analysis sending full transcripts + event payloads to the external provider is acceptable (same posture as summary/audit) — yes/no, and whether a per-customer opt-out is needed.

## 13. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-ai-manager | generic analysis gateway: `models/analysis`, `pkg/analysishandler`, listenhandler route, config (allowed models, default model, input cap, timeout), metrics; unit tests | 1 |
| bin-timeline-manager | NEW MySQL dep as a SECOND persistence engine (separate handler pkg + pool + lifecycle + mock target; ClickHouse `dbhandler` untouched) + `timeline_analyses` (Alembic, shared DB); `models/analysis` (+`Validate()`); analysis CRUD; `pkg/analysishandler` orchestration + adaptive stage chain + prompts; FIRST requesthandler wiring (publisher name, prom-collision check, CB/timeout config); listenhandler routes; metrics; unit tests | 1 |
| bin-common-handler | `AIV1ServiceTypeAnalysisRun` requesthandler method + interface + mock | 1 |
| square-admin | analysis panel + trigger/re-analyze button + structured render (status badge, resource chips, issues with evidence links) | 2 (separate) |

## 14. Implementation Order

1. bin-ai-manager: `models/analysis` (Request/Response).
2. bin-ai-manager: `pkg/analysishandler.Run` (gateway) + config + metrics + unit tests (schema-required, model allow/default, size cap, timeout, success, LLM-error).
3. bin-ai-manager: listenhandler route `POST /v1/services/type/analysis`.
4. bin-common-handler: `AIV1ServiceTypeAnalysisRun` + interface + mock.
5. bin-timeline-manager: add MySQL connection + `timeline_analyses` migration (Alembic convention) + `models/analysis` (+ `Validate()`).
6. bin-timeline-manager: `pkg/dbhandler` analysis CRUD (Create/Get/GetByActiveflowID/List/Update/Delete).
7. bin-timeline-manager: add requesthandler wiring (Flow/AI/Transcribe clients).
8. bin-timeline-manager: `pkg/analysishandler` (Start gate + ownership + async chain + deterministic pre-extraction + adaptive single/3-stage chain + evidence-index resolution) + metrics + unit tests (ended-gate, ownership-mask, idempotent-existing, reanalyze-overwrite, in-flight-skip, dup-key-race, truncation, single-call-small, 3-stage-large, stage-fail->failed, validate-fail-evidence-index->failed, success).
9. bin-timeline-manager: listenhandler routes (POST/GET/LIST/DELETE analyses).
10. Full verification workflow per touched service; PR review loop.

## 15. Open Questions

| # | Question | Recommendation | Owner |
|---|---|---|---|
| Q1 | External LLM receives transcripts + event payloads (PII) for analysis | Same posture as summary/audit (already do this); confirm acceptable, consider per-customer opt-out later | CEO/CTO (immediate) |
| Q2 | ~~Alembic vs golang-migrate~~ RESOLVED | Alembic, shared MySQL instance, table in `bin-dbscheme-manager/bin-manager`; timeline-manager does no DDL. (review C2) | resolved |
| Q3 | Stage model defaults (cost) | Stage1 cheap, Stage2 mid, Stage3 best; all overridable via env; all ⊆ gateway allow-set; document defaults | CPO/CTO |
| Q4 | Auto-trigger on activeflow end | Defer to Phase 2 with a cost estimate (unbounded) | CPO/CTO |
| Q5 | Customer-facing read endpoint (api-manager) + webhook on completion | Defer; add when the UI phase lands if customers (not just admins) should see it | CPO |
| Q6 | Gateway genericity vs lock-down | Internal-only + schema-required + model allow-set + input cap is the agreed control; keep prompt/data free-form for reuse | CPO/CTO |
| Q7 | Cost cap / abuse ceiling (review M2/H4) | Generic internal gateway + manual re-analysis + adaptive 1-3 LLM calls per run = unbounded token spend if a caller loops. P1: per-call `analysisMaxOutputTokens` (runaway guard, not cost lever) + a per-activeflow re-analysis cooldown (`analysisReanalyzeCooldown`) so repeated manual reanalyze on one activeflow cannot loop-spend. Defer a coarse per-customer/day analysis-run cap to Phase 2 (with the auto-trigger cost work). Decide whether the daily cap is needed before any non-controlled rollout | CPO/CTO |

## 16. Review Summary (v1 -> v2)

Two independent reviewers (both with live codebase access) returned CHANGES REQUESTED on v1. All Critical/High addressed in v2:

- **C1 IDOR on read/delete (both reviewers):** `Get`/`GetByActiveflowID`/`List`/`Delete` now take `customer_id` and enforce ownership with masked not-found; `List` server-injects the authority filter. RPC endpoints carry `customer_id` on every op. (§5.2, §7, §6.5)
- **C1/#1 evidence cannot be "event ids" — ClickHouse `Event` has no id field (both reviewers, hard blocker):** redesigned to a Go-assigned synthetic `evidence_index` over the canonical event list; LLM cites indices; Go resolves to `{event_type, timestamp, resource_id}` tuples and `Validate()` rejects out-of-range indices (also closes the C2 hallucinated-evidence hole). (§6.1, §6.4)
- **C2/#2 migration tooling:** resolved to Alembic + shared MySQL instance; timeline-manager does no DDL. (§4, Q2)
- **H1 AggregatedList is paginated:** explicit page loop with `analysisMaxEvents`/`analysisMaxPages` bounds. (§6.1)
- **H2 256 KiB hard-fail makes big flows un-analyzable:** replaced with a deterministic truncation strategy (keep inventory + error events, cap transcripts, mark `truncated:true`). (§6.1, §6.5)
- **H3/#5 concurrent Start race:** dup-key catch on the `UNIQUE(activeflow_id, tm_delete)` insert returns the in-flight row, no 500. (§5.2)
- **H4 stage models must be in gateway allow-set:** startup assertion/test that `{stage1,2,3} ⊆ analysisAllowedModels`. (§3.1, §6.3)
- **#4/#3 blast radius (second reviewer):** documented that timeline-manager gains a SECOND persistence engine (separate MySQL handler package + pool + lifecycle + mock target, ClickHouse handler untouched) AND its FIRST outbound requesthandler (publisher ServiceName, Prometheus metric-collision check, circuit-breaker/timeout config, new downstream failure modes). (§4, §10, §13)
- **#6 3-stage cost/latency on trivial flows:** staging made ADAPTIVE — single combined call below a size threshold, full 3-stage chain above; Stage 2/3 consume prior STRUCTURED output, not raw events again. (§6.3)
- **M1 json_schema plumbing:** added `schema_name` (required by OpenAI), `Strict:true`, and the `additionalProperties:false` + all-required schema requirement. (§3.1)
- **#7/M4 gateway shape:** added `finish_reason`/`truncated` so the caller detects length-truncation before `Validate()`; documented the deliberate divergence from the `service.Service` shape. (§3.1, §6.5)
- **#8 write-back resurrection:** final async write guards on `tm_delete = <sentinel>`. (§5.2)
- **M2 cost cap:** added as Q7 (per-call token ceiling now, per-customer/day cap before rollout).

Deferred (Medium/Low, Open Questions or Phase 2): per-customer/day run cap (Q7), customer-facing read endpoint + completion webhook (Q5), auto-trigger (Q4).

## 17. Review Summary (v2 -> v3)

Round-2 review (fresh, post-v2) returned CHANGES REQUESTED: the v2 adaptive-staging + evidence-index edits introduced ordering/consistency bugs. All Critical/High addressed in v3:

- **C1 evidence_index ordering bug:** pinned the strict pipeline order collect -> reduce -> freeze+index -> pre-extract -> send -> validate -> resolve -> persist; index assigned ONCE on the post-reduction frozen list; error events never elided so their index is stable. (§6.1)
- **H1 Validate() self-contradiction:** split into two phases - (a) raw-output validation (enum + index-in-range + evidence non-empty) on the LLM JSON, then (b) index->tuple resolution, then (c) deterministic `resources_used` overwrite, then persist. Range-check happens on raw output, not the persisted object. (§6.4)
- **H2 reanalyze UPDATE race:** conditional `UPDATE ... WHERE status != 'progressing'`; 0 rows affected -> return in-flight, no second goroutine. (§5.2)
- **H3 SMALL combined-call model escaped allow-set:** combined call pinned to `ENGINE_MODEL_ANALYSIS_STAGE3`, covered by the startup allow-set assertion. (§6.3)
- **H4 cost-cap vs truncated-fail collision + missing max_tokens:** decoupled `analysisMaxOutputTokens` (runaway guard, sized above worst-case) from cost control (Q7 run-count/cooldown); added `MaxTokens` to the gateway request build; promoted per-activeflow reanalyze cooldown to P1. (§3.1, §5.1, Q7)
- **M1 LLM-emitted resources_used:** Go overwrites `resources_used` with deterministic counts (phase c). (§6.4)
- **M2 same-timestamp page-boundary:** cursor `>=` + composite-key de-dup keeps the canonical list complete and deterministic across runs. (§6.1)
- **M3 two `analysisMaxInputBytes` constants:** distinct constants; timeline `analysisReduceTargetBytes` < gateway cap - overhead. (§3.1, §6.1)
- **M4 non-unique persisted evidence tuple:** persist `evidence_index` alongside the resolved tuple. (§6.4)
- **M5 deploy ordering:** Alembic migration before timeline-manager serves Start. (§4)
- **L1 `truncated` overload:** result flag renamed `input_reduced` vs gateway `Truncated`. (§6.4)
- **L2 error hygiene / L3 §5.1 Name+Strict / L4 short-transcript const / L5 Stage1 outline indices:** all applied. (§5.1, §5.2, §6.3)

Round-2 confirmed the round-1 fixes hold and the storage/ownership/security architecture is sound.
