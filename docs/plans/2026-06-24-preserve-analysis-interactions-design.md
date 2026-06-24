# VOIP-1200: Preserve Stage 2 interactions in the analysis verdict

Status: Draft v3
Owner: CPO (design) / CTO (decision)
Scope class: enhancement (verdict schema + model + prompt + projection + RST). Minimal-change bias applies.

## 1. Problem Statement

The Timeline activeflow AI analysis (`bin-timeline-manager/pkg/analysishandler`) runs a 3-stage
LLM chain. Stage 2 (`stage2Schema`) produces the substantive content of the analysis:
`interactions[]` (per-resource `{resource_type, summary}`: what was communicated, intent,
outcome) plus an `overall_narrative`.

That Stage 2 output is fed into the Stage 3 prompt as input and then **discarded**. The final
verdict (`verdict.Verdict`, persisted in `analysis.Result` and surfaced to the customer via
`WebhookMessage.Result`) carries only `overall_status`, `resources_used` (counts), `narrative`
(one line), and `issues`. For a normal call with no problems, `issues` is empty, so the
square-admin AI Analysis panel renders a single narrative line and a couple of counts. The user
reaction was "이게 다야?" (is this all?).

Two concrete defects:

1. **Discard (3-stage path, `runStaged`)**: `chain.go:runStaged` computes `resp2.Result`
   (Stage 2 interactions) but `buildFinalVerdict(raw, input)` only uses the Stage 3 `raw`
   verdict + Go-computed inventory. Stage 2 interactions have no field in `verdict.Verdict`,
   so they are collapsed into the Stage 3 one-line `narrative` and thrown away. We pay metered
   GCP/Gemini cost for a 3-stage analysis and persist ~10% of its output.

2. **Asymmetry (single-call path, `runCombined`)**: short activeflows skip staging and call
   `verdictSchema` directly. There is no `interactions` stage at all on this path, so even after
   fix (1) a short call would still render empty. The verdict shape must be made symmetric across
   both paths or short calls remain sparse forever.

## 2. Scope

In scope (backend, `bin-timeline-manager`, this PR):
- Add `interactions[]` to `verdict.Verdict` (+ `RawVerdict`) and to `verdictSchema` (the single-call
  / combined schema), so BOTH the staged and single-call paths produce/persist interactions
  (`stage3VerdictSchema`, the split-off staged diagnosis schema, deliberately stays without).
- Carry Stage 2 `interactions` forward into the final verdict on the staged path; have the
  single-call path emit them directly (same combined schema).
- Bump `verdict.CurrentVersion` 1 → 2 (additive field; version distinguishes pre/post records).
- Surface `interactions` through the customer projection (`WebhookMessage` already passes
  `Result` through verbatim — no scrubbing change, interactions are not internal fields).
- RST struct doc update for the analysis result shape (pre-commit hook requires it).

Out of scope (deferred):
- **square-admin UI** "Call flow summary" section rendering `interactions[]`: a separate SQUAR
  Story (frontend repo, `monorepo-javascript`). Backend ships first; the field is additive so the
  existing UI is unaffected until the UI opts in.
- Re-running analysis on already-stored v1 records (no backfill; new analyses get v2, old v1
  records keep their shape — `version` field disambiguates).
- Any new LLM stage or model change (no cost-tier change).

**Cost (R3):** the staged path is genuinely zero extra LLM cost (Stage 2 interactions are already
computed and paid for; we stop discarding them). The single-call path adds `interactions[]` to the
existing combined call — no new call, no new stage, no model/tier change; only negligible marginal
output tokens (short calls have few resources), bounded by analysis count (on-demand + persist-once).

## 3. Domain Model change

`bin-timeline-manager/models/verdict/verdict.go`.

Add a new `Interaction` type and an `Interactions` slice on `Verdict` and `RawVerdict`. This is a
1:1 carry of the existing `stage2Schema` interaction shape (`resource_type`, `summary`), so no new
LLM concept is introduced.

```go
// Interaction is one resource's content summary: what was communicated and the
// intent/outcome. Carried forward from the Stage 2 content pass (staged path) or
// emitted directly (single-call path). Customer-facing (no internal fields).
type Interaction struct {
	ResourceType string `json:"resource_type"`
	Summary      string `json:"summary"`
}

type Verdict struct {
	Version       int            `json:"version"`
	OverallStatus OverallStatus  `json:"overall_status"`
	InputReduced  bool           `json:"input_reduced"`
	ResourcesUsed []ResourceUsed `json:"resources_used"`
	Interactions  []Interaction  `json:"interactions"` // NEW
	Narrative     string         `json:"narrative"`
	Issues        []Issue        `json:"issues"`
}

type RawVerdict struct {
	OverallStatus OverallStatus  `json:"overall_status"`
	ResourcesUsed []ResourceUsed `json:"resources_used"`
	Interactions  []Interaction  `json:"interactions"` // NEW (single-call path emits; staged path leaves empty, carried separately)
	Narrative     string         `json:"narrative"`
	Issues        []RawIssue     `json:"issues"`
}
```

`CurrentVersion` 1 → 2.

`Interaction` has no enum/numeric field, so no `Validate()` is needed. `interactions` is allowed
to be empty (`[]`) — an analysis with no resolvable content is valid, just sparse. We do NOT
require non-empty interactions (that would make a legitimately quiet activeflow fail the chain).

### Carry-forward vs emit (the asymmetry resolution)

| Path | How interactions are produced | Where they enter the verdict |
|---|---|---|
| Staged (`runStaged`) | Stage 2 already produces them (`stage2Schema.interactions`) | `buildFinalVerdict` carries `stage2.Interactions` forward; Stage 3 `verdictSchema` does NOT re-emit them (keeps Stage 3 focused on diagnosis, avoids the LLM rewriting Stage 2's content) |
| Single-call (`runCombined`) | `verdictSchema` (now with `interactions`) emits them directly | resolved from `raw.Interactions` in `buildFinalVerdict` |

So `buildFinalVerdict` takes interactions from one of two sources. **Source selection is by PATH
(structural), not by `len()` (value)** — this was a v1→v2 review correction (R1 low). Keying off
`len()==0` was correct only by accident: a staged analysis whose Stage 2 legitimately returns zero
interactions would fall through to `raw.Interactions`, which happens to also be empty on the staged
path — right answer, wrong reason, and a latent trap if `stage3VerdictSchema` ever emitted
interactions. We pass the interactions explicitly per path instead:

```go
// buildFinalVerdict(raw *RawVerdict, interactions []verdict.Interaction, input *collectedInput)
//   runStaged   passes the parsed Stage 2 interactions (never consults raw.Interactions)
//   runCombined passes raw.Interactions (the combined schema emitted them)
```

`runStaged` and `runCombined` each select their own source and pass it in; `buildFinalVerdict` does
NOT branch on `len()`. This removes the structural ambiguity.

**Empty-slice normalization (R2 HIGH):** a Go `nil` slice marshals to JSON `null`, an empty non-nil
slice to `[]`. On the staged path `s2.Interactions` may be nil (a quiet activeflow's Stage 2 returns
zero interactions) and on the single-call path `raw.Interactions` may be empty, so without
normalization a v2 record could serialize `interactions: null` on the staged path while the
single-call path serializes `interactions: []` — two shapes inside the SAME `version: 2`
contract, breaking any consumer doing `.interactions.map(...)`. `buildFinalVerdict` MUST guarantee a
non-nil slice:

```go
if interactions == nil {
	interactions = []verdict.Interaction{}
}
```

This delivers the parity the design asserts (every v2 record carries `interactions: []` at minimum).
Note: `ResourcesUsed`/`Issues` avoid this only because they are always model-emitted into a non-nil
slice; `interactions` needs the explicit guard because the staged path leaves it nil.

(Reviewer note: an alternative is to have Stage 3 also emit interactions and drop the carry —
rejected because it re-spends tokens and lets Stage 3 paraphrase/lose Stage 2 detail.)

## 4. Schema changes (`pkg/analysishandler/schemas.go`)

### 4a. `verdictSchema` — add `interactions` (single-call / combined path only). Because
OpenAI/Gemini strict json_schema requires every property listed in `required` and
`additionalProperties:false`, add `interactions` to both `properties` and `required`:

```jsonc
"required": ["overall_status", "resources_used", "interactions", "narrative", "issues"],
"properties": {
  ...
  "interactions": {
    "type": "array",
    "items": {
      "type": "object",
      "additionalProperties": false,
      "required": ["resource_type", "summary"],
      "properties": {
        "resource_type": { "type": "string" },
        "summary":       { "type": "string" }
      }
    }
  },
  ...
}
```

RATIONALE for the split: putting `interactions` in `required` on the schema the staged Stage 3 used
would FORCE Stage 3 to emit interactions we then discard in favor of the Stage 2 carry (wasteful, and
lets Stage 3 hallucinate content). **Resolution:** Stage 3 runs on a dedicated `stage3VerdictSchema`
WITHOUT interactions; the interactions-bearing `verdictSchema` is used only by the single-call
combined path. This means:

- `verdictSchema` (combined/single-call): HAS interactions.
- `stage3VerdictSchema` (staged Stage 3 diagnosis): NO interactions (diagnosis only;
  interactions come from the Stage 2 carry).

Both marshal into the same `verdict.Verdict` Go type; the staged path fills `Interactions` from the
Stage 2 carry, the single-call path from the combined emission. `ValidateRaw` must tolerate an
absent/empty `interactions` (it already only validates status/severity/evidence; no change needed
beyond the struct field).

This splits the previously-shared diagnosis schema, so update `runStaged` Stage 3 + `runCombined`
to reference the correct schema name each.

**Drift control (R1 low):** the two schemas share four properties (`overall_status`,
`resources_used`, `narrative`, `issues`); only `interactions` differs. To avoid the two raw JSON
blobs drifting on the shared fields, derive `verdictSchema` from `stage3VerdictSchema` by injecting
the single `interactions` property + required entry at construction (Go-side composition of the
`json.RawMessage`), rather than maintaining two independent verbatim JSON literals. `stage3VerdictSchema`
is byte-identical to today's `verdictSchema`, so the diagnosis contract does not change.

## 5. Handler flow changes (`pkg/analysishandler/chain.go`)

1. `runStaged`: unmarshal `resp2.Result` into a Stage 2 wrapper struct, NOT a bare slice. Stage 2's
   schema is the OBJECT `{interactions:[{resource_type,summary}], overall_narrative}`, so the parse
   target is:
   ```go
   type stage2Result struct {
       Interactions     []verdict.Interaction `json:"interactions"`
       OverallNarrative string                `json:"overall_narrative"`
   }
   ```
   Unmarshaling into a bare `[]verdict.Interaction` would fail (object→array mismatch). Pass
   `s2.Interactions` to `buildFinalVerdict`. `overall_narrative` is tolerated/ignored (not persisted).
   Stage 3 now uses `stage3VerdictSchema`.
2. `runCombined`: uses `verdictSchema` (with interactions); passes `raw.Interactions` to
   `buildFinalVerdict` as the interactions argument.
3. `buildFinalVerdict(raw, interactions, input)`: applies the `nil → []verdict.Interaction{}` guard
   (§3) uniformly to its `interactions` argument; sets `Verdict.Interactions` from it and
   `Version: verdict.CurrentVersion` (now 2). It does NOT branch on `len()` or re-read
   `raw.Interactions` — each caller supplies the correct source.
4. Stage 2 prompt: no change needed (already produces interactions). Combined prompt: add
   instructions to also produce `interactions[]` (per-resource content summary) with the SAME shape
   and depth expectations as the Stage 2 pass, so a short call's interactions are never degenerate
   (R3 low — keeps the single-call output from re-triggering "is this all?" on the very calls most
   likely to look thin). Minimal copy edit, no new stage.

Failure handling unchanged: a malformed Stage 2 `interactions` parse fails the chain (same as any
gateway parse error), which is correct — we do not silently drop content on the staged path.

## 6. Customer projection / Webhook / API

No code change to `models/analysis/webhook.go`. `WebhookMessage.Result = h.Result` passes the
verdict JSON through verbatim. Since `interactions` lives INSIDE the verdict and carries no
internal fields (no model id, no token counts), it is automatically and safely exposed through the
existing GET / webhook surface. The `version:2` field tells consumers the shape changed.

REST: `GET /v1.0/timeline-analyses/{id}` and the list endpoint return `result` unchanged in
contract (it is already an opaque versioned JSON document); the only difference is an additional
`interactions` key inside it.

## 7. RST documentation

The struct doc page that documents the analysis `result` shape
(`docsdev/source/*_struct_*.rst` for timeline-analyses) must add the `interactions[]` field +
`version: 2` note in the SAME commit (pre-commit hook blocks the projection/model change
otherwise). Update the result JSON example to include an `interactions` array.

## 8. Observability

No new metric. The analysis chain already has its counter/latency instrumentation; adding a field
to the verdict does not change the lifecycle. (Minimal-change: a per-interaction-count metric is
deferred / optional, not in v1.)

## 9. Security & Compliance

`interactions[]` content is the LLM's summary of the activeflow's own events/transcripts, scoped to
the activeflow's customer. It is exposed only on the customer-owned analysis record (same ownership
predicate as `result` today — no new exposure surface, no new PII class beyond what `narrative` and
the transcripts already represent). No external-LLM PII change: the transcripts were already sent to
the gateway to produce the existing narrative; we are persisting more of the result we already
generate, not sending more input.

## 10. Affected files

| File | Change | Phase |
|---|---|---|
| `models/verdict/verdict.go` | Add `Interaction`, `Interactions` on `Verdict`+`RawVerdict`, bump `CurrentVersion`→2 | this PR |
| `pkg/analysishandler/schemas.go` | Add `interactions` to combined `verdictSchema`; split out `stage3VerdictSchema` (no interactions) | this PR |
| `pkg/analysishandler/chain.go` | Parse Stage 2 interactions; source-select in `buildFinalVerdict`; schema-name wiring | this PR |
| `pkg/analysishandler/prompts.go` | Combined prompt: one line to also emit `interactions[]` | this PR |
| `*_test.go` (verdict + analysishandler) | Assert interactions present on both paths; version=2; empty-interactions tolerated | this PR |
| `docsdev/source/*_struct_*.rst` | Document `interactions[]` + version 2 in result | this PR |
| square-admin panel | Render "Call flow summary" from `interactions[]` | SQUAR (separate) |

## 11. Implementation order

1. `verdict.go`: add types, bump version.
2. `schemas.go`: add interactions to combined schema; introduce `stage3VerdictSchema`.
3. `chain.go`: parse Stage 2 wrapper into `stage2Result`; wire schema names; update
   `buildFinalVerdict` to the new `(raw, interactions, input)` signature and update ALL existing
   callers + their unit tests for the added argument.
4. `prompts.go`: combined prompt copy edit.
5. Tests:
   - **Staged path carries Stage 2 interactions**: a staged run whose Stage 2 returns N>0
     interactions produces a verdict with those N interactions (extracted from the `stage2Result`
     wrapper, `overall_narrative` ignored).
   - **Single-call path emits interactions**: `runCombined` produces a verdict with `raw.Interactions`.
   - **Empty-interactions null-vs-`[]` pin (R2 HIGH)**: when Stage 2 (staged) OR the combined call
     (single-call) returns ZERO interactions, MARSHAL the persisted verdict and assert the
     `interactions` key serializes to `[]`, NOT `null`. A `len()==0` / `assert.Len(...,0)` check is
     INSUFFICIENT (passes for both nil and empty). Use a JSON-string/`json.RawMessage` compare on the
     marshaled output, or `assert.Equal([]verdict.Interaction{}, v.Interactions)` (testify `Equal`
     distinguishes nil from empty). Cover BOTH paths (the guard lives in `buildFinalVerdict`).
   - **Malformed Stage 2 parse fails the chain** (object→wrapper unmarshal error surfaces).
   - **Schema-split guard**: `stage3VerdictSchema` does NOT contain `interactions`; `verdictSchema`
     DOES. Repoint any existing staged-path test that asserted against `verdictSchema` to
     `stage3VerdictSchema`.
   - **Version**: persisted verdict `version == 2`.
6. RST struct doc.
7. Full verification (`go mod tidy && vendor && generate && test && golangci-lint`).
8. PR review loop (min 3 rounds).

## 12. Open Questions

| # | Question | Recommendation | Owner |
|---|---|---|---|
| Q1 | Bump `version` to 2, or keep 1 and treat interactions as optional-absent on old records? | **Bump to 2.** Additive but the shape genuinely changed; version is the disambiguator we already designed for. | CTO |
| Q2 | Stage 3 emit interactions vs Stage 2 carry-forward? | **Carry forward** (no token re-spend, no Stage 3 paraphrase loss). Split `stage3VerdictSchema`. | CTO |
| Q3 | Should the single-call (short-call) path also produce interactions, or accept short calls staying sparse? | **Produce them.** A short call rendering empty is the same defect; combined schema already nearly free. | CTO |
| Q4 | Backfill v1 records to v2 by re-analysis? | **No.** Manual re-analysis already resets a row in place; no mass backfill (unbounded LLM cost). `version` disambiguates. | CTO |
| Q5 | UI in this PR or separate SQUAR ticket — and does backend-only resolve the complaint? | **Separate SQUAR, but co-sequenced (R3 HIGH).** The "이게 다야?" complaint is a UI experience; shipping the backend field alone leaves the panel visually identical until the square-admin "Call flow summary" section ships, so backend-only spends (marginal) tokens with zero user-visible improvement. Backend stays a separate PR (different repo, additive field), but the SQUAR frontend ticket must be committed to the same release/sprint window, not left open-ended. New analyses created in the gap carry `interactions[]` and the UI reads them retroactively (persist-once, no backfill needed). **Decision needed from CEO: commit the SQUAR UI to this window?** | CEO |

## 13. Review Summary (v1 → v2)

Round 1: three independent reviewers (correctness/domain, schema/compat, cost/product).

| Finding | Severity | Resolution in v2 |
|---|---|---|
| R2: staged-path nil slice marshals to `null`, not `[]` — two shapes inside one `version:2` contract | HIGH | §3: `buildFinalVerdict` normalizes `nil → []verdict.Interaction{}`; every v2 record carries `interactions:[]` at minimum |
| R3: backend-only field does not resolve the UI-level "이게 다야?" complaint (sequencing) | HIGH | §2 + Q5: SQUAR UI ticket co-sequenced to the same release window; surfaced as explicit CEO decision |
| R1: source-selection keyed on `len()` is correct-by-accident, latent trap | low | §3: selection is now by PATH (structural) — `runStaged`/`runCombined` each pass their own source; `buildFinalVerdict` does not branch on `len()` |
| R1: two json_schema literals can drift on shared fields | low | §4: derive `verdictSchema` from `stage3VerdictSchema` by injecting only the `interactions` property; shared fields have one source |
| R3: "no extra LLM cost" overstated for single-call path | low | §2 cost line reworded: no new call/stage/model; negligible marginal output tokens on the single-call path |
| R3: single-call interactions may be degenerate/shallow | low | §5.4: combined prompt instructs same shape/depth as Stage 2; non-degenerate floor |

R1 (correctness) verdict was APPROVE; R2 and R3 were CHANGES REQUESTED, both HIGH items now resolved. v2 ready for re-review.

## 14. Review Summary (v2 → v3)

Round 2: two independent reviewers (internal-consistency/contradiction-hunt, test-strategy/impl-completeness).

| Finding | Severity | Resolution in v3 |
|---|---|---|
| Stage 2 parse target wrong: `resp2.Result` is the OBJECT `{interactions, overall_narrative}`, not a bare `[]Interaction` (bare unmarshal fails) | HIGH | §5.1: parse into `stage2Result` wrapper struct, pass `.Interactions`; `overall_narrative` ignored |
| Test plan said only "empty allowed; version=2" — does NOT pin the null-vs-`[]` bug (`len()==0` passes for both) | HIGH | §11.5: explicit marshal-and-assert `interactions:[]` (not null) on BOTH paths; `assert.Equal([]Interaction{}, ...)` or JSON compare, never `assert.Len(...,0)` |
| §4a stale v1 sentence ("Stage 3 uses verdictSchema... FORCED to emit") contradicted the resolution below it | medium | §4a reworded to past-tense RATIONALE; header second clause dropped |
| §2 in-scope "verdictSchema + the combined-call schema" redundant (verdictSchema IS the combined schema) | medium | §2 reworded: interactions in `verdictSchema` (single-call/combined); `stage3VerdictSchema` stays without |
| §5 combined-path data-flow described inconsistently (caller-passes vs callee-reads) | low-med | §5: both callers PASS their source; `buildFinalVerdict` applies the guard uniformly, no `len()` branch, no `raw.Interactions` re-read |
| schema split may break existing staged-path tests asserting `verdictSchema` | medium | §11.5 schema-split guard test + repoint instruction |
| `buildFinalVerdict` 3rd-param signature change | (noted sound) | §11.3: explicit "update ALL callers + tests" step |

VERIFIED FIXED by R2: both prior HIGH items (R2 nil→[] normalization placement; R1 by-path source selection) confirmed correct and internally consistent. v3 ready for final (Round 3) re-review.
