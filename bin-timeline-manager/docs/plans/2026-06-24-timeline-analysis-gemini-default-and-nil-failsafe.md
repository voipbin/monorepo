# VOIP-1197 — timeline analysis nil-failsafe + Gemini default engine

Status: Draft (in review)

## Problem statement

Two defects, one prod outage:

1. **Prod crash-loop (severity: high).** `bin-timeline-manager` registers the
   `/v1/analyses` RPC routes unconditionally, but only constructs `analysisH`
   when `cfg.DatabaseDSN != ""` (`cmd/timeline-manager/main.go:193-212`). The
   prod Deployment has NO `DATABASE_DSN` env (verified: only `CLICKHOUSE_*`),
   so `analysisHandler` is `nil`. An inbound `POST /v1/analyses` then SIGSEGVs
   at `pkg/listenhandler/v1_analyses.go:66` (`h.analysisHandler.Start(...)`),
   killing the RPC consumer goroutine and crash-looping both pods. A common
   config omission (missing DSN) takes down the whole service.

2. **Analysis feature disabled in prod.** Because `DATABASE_DSN` is unset, the
   analysis feature (square-admin AI Analysis panel) is effectively off in
   prod even though the frontend (square-admin PR #330) correctly calls it.

Separately, the CEO/CTO directs that the analysis LLM gateway should default to
a Google Gemini model instead of OpenAI `gpt-4o`, for cost reasons.

## Goals (numbered, testable)

1. **G1 (code fail-safe).** When `analysisHandler == nil`, the four analysis
   RPC handlers (`v1AnalysesPost`, `v1AnalysesGet`, `v1AnalysesIDGet`,
   `v1AnalysesIDDelete`) MUST return `503 Service Unavailable` instead of
   panicking. Verifiable by a unit test that builds a `listenHandler` with
   `analysisHandler: nil` and asserts 503 + no panic for all four routes.
2. **G2 (analysis gateway default → Gemini).** The ai-manager analysis gateway
   default model becomes `gemini-2.5-flash`, the allow-set becomes Gemini
   models, and the gateway calls the Gemini OpenAI-compatible endpoint using
   `GOOGLE_API_KEY`. Verifiable by config unit tests + a gateway construction
   test asserting the Gemini base URL + key wiring.
3. **G3 (prod enablement, ops).** The timeline-manager Deployment gets
   `DATABASE_DSN` (from secret `voipbin` key `DATABASE_DSN_BIN`) so analysisH
   is constructed and the feature is live. Stage model envs are left unset so
   all three stages resolve to the new Gemini default (G2). Verifiable by
   install-repo manifest + secret_schema parity tests.

## Non-goals (explicit scope cuts)

- **N1.** Switching the conversational AIcall path (real-time call AI,
  `engine_openai_handler.MessageSend` / `StreamingSend`) to Gemini. Tracked
  separately in **VOIP-1198**. This PR only touches the analysis gateway.
- **N2.** Per-stage cost tiering (Stage1=cheap, Stage3=best). Out of scope;
  stages default to one Gemini model for now. Can be tuned later via env.
- **N3.** Removing the `Strict` json_schema flag globally. We change it only on
  the analysis gateway path (see §5.2 D-decision).

## Affected files (table)

| Repo | File | Why |
|---|---|---|
| monorepo | `bin-timeline-manager/pkg/listenhandler/v1_analyses.go` | G1: add nil guard returning 503 in 4 handlers |
| monorepo | `bin-timeline-manager/pkg/listenhandler/v1_analyses_test.go` | G1: nil-handler 503 tests |
| monorepo | `bin-ai-manager/pkg/engine_openai_handler/main.go` | G2: add `NewEngineOpenaiHandlerWithConfig(apiKey, baseURL)` constructor (Gemini-capable) |
| monorepo | `bin-ai-manager/pkg/analysishandler/run.go` | G2: `Strict:false` for Gemini compat |
| monorepo | `bin-ai-manager/pkg/analysishandler/run_test.go` | G2: update Strict + model expectations (see §5.4) |
| monorepo | `bin-ai-manager/cmd/ai-manager/main.go` | G2: construct analysis gateway with Gemini engine (base URL from config) + GOOGLE_API_KEY |
| monorepo | `bin-ai-manager/internal/config/main.go` | G2: change default model + allow-set defaults; add `analysis_engine_base_url` flag |
| monorepo | `bin-ai-manager/internal/config/main_test.go` | G2: add default/base-url expectations |

Note: `bin-ai-manager/pkg/analysishandler/main.go` (`NewAnalysisHandler`) needs
NO change — it already accepts the engine handler as a parameter. The only
analysishandler edit is `run.go` (Strict flag).
| monorepo | `bin-timeline-manager/k8s/deployment.yml` | G3: add DATABASE_DSN env (real internal prod deploy source, replicas=2) |
| install | `k8s/backend/services/timeline-manager.yaml` | G3: add DATABASE_DSN env (external self-hosting installer, replicas=1) |
| install | `scripts/secret_schema.py` | G3: add (DATABASE_DSN, DATABASE_DSN_BIN) to timeline-manager secret_env |
| (live) | GKE prod `deploy/timeline-manager` | G3: apply via the internal deploy pipeline (monorepo k8s), NOT manual edit |

## Exact changes (per-file)

### 5.1 G1 — nil-handler fail-safe (`v1_analyses.go`)

Add a guard at the top of each of the four handlers, before any
`h.analysisHandler` deref. Pattern:

```go
func (h *listenHandler) v1AnalysesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	if h.analysisHandler == nil {
		return simpleResponse(http.StatusServiceUnavailable), nil
	}
	// ... existing body
}
```

Same guard in `v1AnalysesGet`, `v1AnalysesIDGet`, `v1AnalysesIDDelete`. This is
the design §7.2-recommended path (return 503), preferred over conditional route
registration because: (a) it keeps `processRequest` routing table unchanged,
(b) it is unit-testable per-handler, (c) it gives a clear 503 (service exists
but disabled) rather than 404 (route missing), matching the feature-disabled
semantics.

Wire-field note: `simpleResponse(code int) *sock.Response` already exists
(`main.go:85`); `http.StatusServiceUnavailable == 503`.

### 5.2 G2 — analysis gateway → Gemini

**D-decision (Strict flag).** Today `run.go:81` sets `Strict: true`. The verified
in-repo Gemini caller (`geminiaudithandler/main.go:256`) deliberately uses
`Strict: false`, because Gemini's OpenAI-compat layer does not fully support
strict json_schema. Therefore the analysis gateway, once pointed at Gemini, MUST
use `Strict: false`. Decision: change `run.go` to `Strict: false`
unconditionally (the gateway is now Gemini-only). Rationale: avoid a model-name
sniffing branch; the gateway has exactly one provider after this change.

**Engine wiring.** `engine_openai_handler.NewEngineOpenaiHandler(apiKey)` hardwires
`openai.NewClient(apiKey)` → `api.openai.com`. We must NOT mutate that shared
constructor (it is also used by conversational AIcall — N1). Instead, add a
Gemini-configured engine constructor and pass it to the analysis gateway only.

Proposed: add to `engine_openai_handler`:

```go
// NewEngineOpenaiHandlerWithConfig builds an engine against a custom base URL
// (e.g. the Gemini OpenAI-compatible endpoint). Used by the analysis gateway.
func NewEngineOpenaiHandlerWithConfig(apiKey, baseURL string) EngineOpenaiHandler {
	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	return &engineOpenaiHandler{client: openai.NewClientWithConfig(cfg)}
}
```

In `cmd/ai-manager/main.go`, construct a dedicated analysis engine using the
base URL from config (OQ1 adopted — see below):

```go
// Select the analysis-engine API key by provider (base URL). Keeping this the
// single accepted branch makes provider rollback fully env-driven (§5.2 OQ1).
analysisKey := cfg.EngineKeyChatGPT // OpenAI default
if strings.Contains(cfg.AnalysisEngineBaseURL, "generativelanguage") {
	analysisKey = cfg.GoogleAPIKey
}
analysisEngine := engine_openai_handler.NewEngineOpenaiHandlerWithConfig(analysisKey, cfg.AnalysisEngineBaseURL)
analysisHandler := analysishandler.NewAnalysisHandler(
	analysisEngine,
	cfg.AnalysisDefaultModel,
	strings.Split(cfg.AnalysisAllowedModels, ","),
	cfg.AnalysisMaxInputBytes,
	cfg.AnalysisMaxOutputTokens,
)
```

The conversational `engineOpenaiHandler` (OpenAI) is unchanged and still passed
to `messageHandler` / `summaryHandler` (N1 preserved).

**OQ1 adopted: base URL + key as config, for env-only rollback.** Per iter-1
review, hardcoding the Gemini endpoint as a Go `const` makes the documented
"env-only rollback to OpenAI" impossible (an env-set `gpt-4o` would still be
sent to the Gemini endpoint with `GOOGLE_API_KEY`). To make rollback real
WITHOUT a code change, the analysis engine base URL becomes a config flag.
The API key is also selected by config so a full provider swap is env-only.

**Config defaults** (`internal/config/main.go`):

```
- f.String("analysis_default_model", "gpt-4o", ...)
+ f.String("analysis_default_model", "gemini-2.5-flash", ...)
- f.String("analysis_allowed_models", "gpt-4o,gpt-4o-mini,gpt-4-turbo", ...)
+ f.String("analysis_allowed_models", "gemini-2.5-flash,gemini-2.5-pro", ...)
+ f.String("analysis_engine_base_url", "https://generativelanguage.googleapis.com/v1beta/openai/", "Base URL for the analysis gateway LLM engine (Gemini OpenAI-compat by default; clear to use OpenAI)")
```
With a corresponding `ANALYSIS_ENGINE_BASE_URL` binding + `AnalysisEngineBaseURL`
struct field. **Four edit sites in `internal/config/main.go` (all required — the
loader is easy to miss):** (1) `f.String("analysis_engine_base_url", ...)` flag
def; (2) `"analysis_engine_base_url": "ANALYSIS_ENGINE_BASE_URL"` in the bindings
map; (3) `AnalysisEngineBaseURL string` struct field; (4) `AnalysisEngineBaseURL:
viper.GetString("analysis_engine_base_url")` in the `LoadGlobalConfig` return
(config/main.go ~110-131). Omitting (4) leaves the field empty at runtime, which
silently falls back to OpenAI's base URL with the Gemini key — a latent bug.
`NewEngineOpenaiHandlerWithConfig` leaves `cfg.BaseURL` at the
SDK default (OpenAI) when `baseURL == ""`, so clearing the env var reverts the
engine to OpenAI; combined with `ANALYSIS_DEFAULT_MODEL=gpt-4o` +
`ANALYSIS_ALLOWED_MODELS=gpt-4o,...` and pointing the key env back to an OpenAI
key, rollback to OpenAI is fully env-driven.

Key-selection note: `NewEngineOpenaiHandlerWithConfig` takes `cfg.GoogleAPIKey`
for the Gemini default. For an OpenAI rollback the operator would also need the
engine to use `ENGINE_KEY_CHATGPT`. To keep rollback truly code-free, the
construction selects the key by base URL: if `AnalysisEngineBaseURL` is the
Gemini endpoint (or contains `generativelanguage`), use `GoogleAPIKey`; else use
`EngineKeyChatGPT`. This is the one branch we accept (documented), so the
provider is fully env-selectable. Reviewer to confirm this is preferable to a
dedicated `ANALYSIS_ENGINE_API_KEY` env (which would be cleaner but adds a new
secret key). Recommendation: base-URL-based key selection, no new secret.

**`GoogleAPIKey`** is already wired (`cfg.GoogleAPIKey`, env `GOOGLE_API_KEY`),
and is present in prod secret `voipbin` (verified). No new secret key needed for
the default (Gemini) path.

**Wire-field checklist (Gemini OpenAI-compat, verified against in-repo
geminiaudithandler which runs in prod):**

| Field | Value | Source |
|---|---|---|
| base URL | `https://generativelanguage.googleapis.com/v1beta/openai/` | geminiaudithandler/main.go:17 |
| api key env | `GOOGLE_API_KEY` (`AIza...`) | config/main.go:80, prod secret verified |
| model (default) | `gemini-2.5-flash` | geminiaudithandler uses gemini-2.5-flash in prod |
| response_format | `json_schema`, `Strict:false` | geminiaudithandler/main.go:251-257 |

### 5.4 G2 — existing test updates (`run_test.go`, `config/main_test.go`)

Switching to `Strict:false` + Gemini defaults breaks existing ai-manager tests.
Enumerate and update:

- `pkg/analysishandler/run_test.go`:
  - The assertion that `JSONSchema.Strict == true` (the request-shaping test)
    MUST flip to expect `false`.
  - Hardcoded `gpt-4o` / `gpt-4o-mini` expectations for default model, allow-set
    membership, and the empty-model fallback MUST update to
    `gemini-2.5-flash` / `gemini-2.5-pro`. (Exact line numbers to be confirmed
    at implementation against the current file; do not trust stale numbers.)
- `internal/config/main_test.go`: add assertions for the new defaults
  (`analysis_default_model = gemini-2.5-flash`, allow-set, and
  `analysis_engine_base_url` default) — these are net-new (the file currently
  has no analysis-default assertions).

The implementation step MUST re-grep `run_test.go` for `Strict`, `gpt-4o`,
`gpt-4o-mini`, `gpt-4-turbo` and update every hit; the verification grep below
asserts zero residual `gpt-` literals in ai-manager analysis test/code (except
### 5.3 G3 — prod enablement (two deploy manifests)

**Deploy-source distinction (corrected post design-approval).** VoIPBin has TWO
k8s manifest sources for the same service, and both must carry the env change:
- **Internal production**: `monorepo/bin-timeline-manager/k8s/deployment.yml`
  (replicas=2). This is the REAL source the internal prod GKE deploy uses.
- **External self-hosting installer**: `install/k8s/backend/services/timeline-manager.yaml`
  (replicas=1) + `install/scripts/secret_schema.py`. This packages the first-time
  installer for external operators.

An env/secret change like DATABASE_DSN MUST be applied to BOTH or the two drift.
(Initial draft only touched the install repo; corrected to add the monorepo k8s
manifest, which is what actually reaches internal prod.)

**monorepo `bin-timeline-manager/k8s/deployment.yml`**, add as the first `env:`
entry (mirrors `bin-ai-manager/k8s/deployment.yml:27-31`):

```yaml
            - name: DATABASE_DSN
              valueFrom:
                secretKeyRef:
                  name: voipbin
                  key: DATABASE_DSN_BIN
```

**install `k8s/backend/services/timeline-manager.yaml`**, add to the `env:` list:

```yaml
            - name: DATABASE_DSN
              valueFrom:
                secretKeyRef:
                  name: voipbin
                  key: DATABASE_DSN_BIN
```

`install/scripts/secret_schema.py`, in the `"timeline-manager"` block `secret_env`
(line ~630), add as the first entry (matching ai-manager's ordering):

```python
            ("DATABASE_DSN", "DATABASE_DSN_BIN"),
```

This mirrors the existing ai-manager wiring (`secret_schema.py:179`,
`ai-manager.yaml:30-34`) exactly. `DATABASE_DSN_BIN` already exists in the
schema (line 53) and in prod secret `voipbin` (verified).

**Migration precondition (already satisfied).** #1008 design M5 requires the
shared-MySQL `timeline_analyses` table to exist BEFORE enabling. Verified in
prod: table exists, alembic head `a63b82d73655` applied. No migration step
needed in this PR.

**Stage model envs.** Intentionally left UNSET on the Deployment. With G2, an
empty `req.Model` resolves to the gateway default `gemini-2.5-flash`
(`run.go:45-52`: empty model → defaultModel, no warning). Per-stage tiering is
N2.

## Copy / decision rationale

- **503 over 404** for nil handler: the route exists but the feature is
  disabled; 503 (Service Unavailable) is the honest status. 404 would imply the
  endpoint does not exist and confuse the frontend.
- **Separate Gemini engine instance** over mutating the shared constructor:
  keeps conversational AI on OpenAI (N1), zero blast radius on call AI.
- **`Strict:false`**: required for Gemini OpenAI-compat; proven by the prod
  geminiaudithandler.
- **Reuse `DATABASE_DSN_BIN`**: no new secret; identical to ai-manager.

## Verification plan

Code (monorepo), run in EACH changed service dir:
```
cd bin-timeline-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-ai-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Grep checks:
- `grep -n "analysisHandler == nil" bin-timeline-manager/pkg/listenhandler/v1_analyses.go` → 4 hits.
- `grep -n "gemini-2.5-flash" bin-ai-manager/internal/config/main.go` → default + allow-set + base-url default.
- `grep -rn "gpt-4o\|gpt-4o-mini\|gpt-4-turbo" bin-ai-manager/pkg/analysishandler bin-ai-manager/internal/config` → only intentional rollback-doc comments (no live default/allow-set/test literal).
- `grep -n "Strict" bin-ai-manager/pkg/analysishandler/run.go` → `false`.
- `grep -n "NewEngineOpenaiHandlerWithConfig" bin-ai-manager` → defined in engine_openai_handler/main.go, used in cmd/ai-manager/main.go.
- Confirm `engine_openai_handler.NewEngineOpenaiHandler(cfg.EngineKeyChatGPT)` STILL exists and is used by messageHandler/summaryHandler (N1 intact).

Install repo:
```
cd ~/gitvoipbin/install && pytest tests/ -q && bash scripts/dev/check-plan-sensitive.sh <none-committed>
python3 -c "import yaml,sys; list(yaml.safe_load_all(open('k8s/backend/services/timeline-manager.yaml')))"
```

New unit tests:
- `Test_v1Analyses_nilHandler_503` (timeline): build `&listenHandler{analysisHandler: nil}`, call all four routes via `processRequest`, assert 503 + no panic.
- ai-manager: config default-model test update; gateway falls back to default when model empty (already covered by run tests, update expected default).

## Rollout / risk

- **Order:** Merge + deploy the monorepo changes (G1 + G2) FIRST. G1 stops the
  crash-loop regardless of config. G2 changes the gateway provider. THEN merge
  install (G3) and apply, which turns the feature on against Gemini.
- **Risk R1 (Gemini quota/latency).** Analysis is async (timeline chain), not
  on the call path; latency is tolerable. Quota: GOOGLE_API_KEY already serves
  prod gemini-audit; analysis adds load. Mitigation: monitor
  `ai_manager_analysis_gateway_run_duration_seconds` post-deploy.
- **Risk R2 (allow-set coercion).** If a stage env names a model NOT in the
  allow-set, it silently coerces to default (gemini-2.5-flash). Acceptable;
  stages are unset (N2).
- **Risk R3 (Strict:false weaker schema enforcement).** Gemini may
  occasionally emit non-conformant JSON. The chain already guards truncation
  (`callGateway`, chain.go:123) and each stage unmarshals into typed structs;
  malformed output fails the chain cleanly (no panic). Acceptable.
- **Rollback (env-only, no code change — enabled by OQ1).** To revert the
  analysis gateway to OpenAI without redeploying new code, set on the ai-manager
  Deployment: `ANALYSIS_ENGINE_BASE_URL=""` (engine falls back to OpenAI SDK
  default base URL + selects `ENGINE_KEY_CHATGPT`), `ANALYSIS_DEFAULT_MODEL=gpt-4o`,
  `ANALYSIS_ALLOWED_MODELS=gpt-4o,gpt-4o-mini,gpt-4-turbo`. No image rebuild
  needed. (This is why OQ1 was adopted in §5.2.)

## Open questions (for reviewer)

- OQ1 (RESOLVED, adopted in §5.2): analysis engine base URL is now a config
  flag `analysis_engine_base_url` defaulting to the Gemini endpoint, enabling
  env-only rollback to OpenAI. Remaining sub-question for reviewer: confirm
  base-URL-based key selection (Gemini endpoint → GoogleAPIKey, else
  EngineKeyChatGPT) is preferred over adding a dedicated `ANALYSIS_ENGINE_API_KEY`
  secret. Recommendation: base-URL-based, no new secret.
- OQ2: allow-set — include `gemini-2.5-flash-lite` for future Stage1 tiering
  (N2) now, or add later? Recommendation: include flash + pro only now.

## Iter-1 review response summary

iter-1 verdict: CHANGES_REQUESTED (3 items). All addressed:

- **Item 1 (§5.2 + Rollout/Rollback contradiction):** Adopted OQ1. Base URL is
  now a config flag (`analysis_engine_base_url`), key selected by base URL, so
  env-only rollback to OpenAI is actually possible. Rollback bullet rewritten in
  §Rollout/risk to give the exact env triplet. See §5.2 "OQ1 adopted".
- **Item 2 (§Affected files inaccuracy):** Table corrected — added
  `engine_openai_handler/main.go` (new constructor location), removed the
  `analysishandler/main.go` row (no change needed), added explicit note that
  `NewAnalysisHandler` is unchanged. See §"Affected files".
- **Item 3 (existing test breakage understated):** Added new §5.4 enumerating
  `run_test.go` updates (Strict true→false; gpt-4o/gpt-4o-mini → gemini) and
  `config/main_test.go` net-new assertions, with a re-grep mandate at
  implementation (no stale line numbers trusted).

## Iter-2 review response summary

iter-2 verdict: CHANGES_REQUESTED (2 items). iter-1 fixes were all verified
correct (incl. go-openai v1.41.2 DefaultConfig base URL = `https://api.openai.com/v1`,
not empty). Both new items addressed:

- **Item 1 (§5.2 code block self-contradiction):** The construction snippet
  hardcoded `cfg.GoogleAPIKey`, contradicting the base-URL-based key-selection
  prose. Rewrote the §5.2 code block to compute `analysisKey` by base URL
  (`strings.Contains(..., "generativelanguage")` → GoogleAPIKey, else
  EngineKeyChatGPT) before constructing the engine, so the env-only rollback is
  implementable verbatim. Note: `cmd/ai-manager/main.go` already imports
  `strings` (used for `strings.Split` on the allow-set), so no new import.
- **Item 2 (loader assignment omitted):** §5.2 now enumerates all FOUR
  `internal/config/main.go` edit sites explicitly, including the
  `viper.GetString("analysis_engine_base_url")` assignment in `LoadGlobalConfig`,
  with a callout that omitting it causes a silent OpenAI-base-URL + Gemini-key
  fallback bug.

## Iter-3 review

iter-3 verdict: APPROVED. All iter-2 fixes verified correct against the actual
code (strings import line 8, EngineKeyChatGPT/GoogleAPIKey fields, NewAnalysisHandler
signature match, four config edit sites at lines 66/84-85/35-36/125-126, G1 four
handlers + simpleResponse + analysisHandler field). No new contradictions; design
internally consistent and implementation-ready.

## Approval status

APPROVED (design review loop: iter-1 CR → iter-2 CR → iter-3 APPROVED). Ready for
implementation (Phase 4).
