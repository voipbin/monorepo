# OpenRouter LLM Routing Implementation Plan

> **Status: APPROVED (implementation-plan review loop closed: rounds 3 and 4, both reviewers APPROVED each round).**
>
> **For Hermes:** Use subagent-driven-development to implement task by task. Design: `2026-10-05-openrouter-llm-routing-design.md` (APPROVED). Analysis: `2026-10-05-openrouter-llm-routing-issue-analysis.md` (APPROVED). Section references such as "design 3.2" point to the design doc.

**Goal:** Customers keep choosing familiar model names; Claude, Llama, DeepSeek, Qwen, Mistral models run through a platform-held OpenRouter key (ZDR on), with a designed model picker in square-admin.

**Architecture:** One Go catalog and one fail-closed resolver in `bin-ai-manager/models/ai`. pipecat-manager resolves at every session start and hands the Python runner either an existing direct type or the internal type `platform_openrouter.<slug>` with an empty key. The runner has an OpenRouter branch and no catalog. `GET /ai_models` serves the catalog to the UI.

**Tech Stack:** Go (monorepo, vendored per service), Python 3.11/pipecat-ai 1.12.x, OpenAPI/oapi-codegen, React (square-admin), Sphinx RST.

**Repos and PRs (one PR per repository, no further splitting):** `monorepo` (backend, docs, runner), `monorepo-javascript` (UI, skill.md, llms.txt), `install` (secret schema and manifests), `monorepo-etc` (secret source placeholder, operator-run, see Task 2).

**Global rules for every task:** work only in worktrees; TDD (failing test first); commit per task with the monorepo commit format (summary line max 72 chars, then `- project-name: change` bullets, no AI attribution, author `Sungtae Kim <pchero21@gmail.com>`; the one-line commit messages shown below are the summary line only); never touch `main`. Verification workflow for any service whose Go code or whose vendored `bin-ai-manager` copy changes: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`. `vendor/` is gitignored and every service reaches monorepo modules through `replace => ../`; vendor is a local build artifact, never committed. Any task that makes a service use a symbol newly added to another module (Task 8, 14, 15) runs `go mod tidy && go mod vendor` in that service before its tests.

---

## Phase A: Secrets and config (no behavior change)

### Task 1: Runner env and operator docs (monorepo)
**Files:** Modify `bin-pipecat-manager/komodo/docker-compose.yml` (ONLY the two Python runner services `pipecat-script-runner-1` and `pipecat-script-runner-2`; they are the services that carry `GOOGLE_API_KEY` at ~160/233; `XAI_API_KEY` also appears on the two Go manager services at ~111/190, which never read LLM keys, so they are left alone); Modify `bin-pipecat-manager/docs/operations.md` (env table).
- Add `- OPENROUTER_API_KEY=[[BIN_MANAGER__OPENROUTER_API_KEY]]` on the two script-runner services.
- Document: key read only by the runner, ZDR default; a self-host with an unset or invalid key gets the same provider authentication notice as an invalid OpenAI key (no special casing).
- Check: `grep -c OPENROUTER_API_KEY bin-pipecat-manager/komodo/docker-compose.yml` prints `2`.
- Commit: `bin-pipecat-manager: Add OPENROUTER_API_KEY runner env and ops docs`.

### Task 2: install repo secret plumbing; monorepo-etc operator step
**Files (install repo, own worktree/PR):**
- `scripts/secret_schema.py`: add `"OPENROUTER_API_KEY": {"default": "dummy-openrouter-key", "class": "secret"}` beside `XAI_API_KEY` (~125); update the key-count constant/assertion and the "53-key" comment (~129); add `("OPENROUTER_API_KEY", "OPENROUTER_API_KEY")` to the pipecat entry of `BIN_SERVICE_WIRING` (~489).
- `scripts/secretmgr.py`: update the stale "26 operator-editable" docstring/messages to the real computed count (today's computed seed count is 24, becoming 25; do not trust the old comment, compute from the test). Note: `generate_all_secrets()` seeds every `class=secret` key with a random password (`secretmgr.py:35-52`), exactly like the existing OpenAI/Google keys, so a fresh self-host holds an invalid random OpenRouter key until the operator sets a real one; the runner treats it like any invalid provider key (OpenRouter returns 401, classified as authentication, customer notice as for other providers). The schema default `dummy-openrouter-key` is therefore documentation only. No `dummy` special casing anywhere (design 3.3/3.4 clarified accordingly).
- Do NOT run `scripts/dev/gen_backend_manifests.py`: it hardcodes `image: voipbin/{name}` (`gen_backend_manifests.py:103`) while the committed manifests use kustomize-substituted `image: pipecat-manager-image` plus `imagePullPolicy: Always`; a reviewer reproduced that running it rewrites 31 manifests and drops the pinning (no current test catches it). Instead keep `BIN_SERVICE_WIRING` correct (so the generator stays truthful about env) and HAND-ADD only the env block in `k8s/backend/services/pipecat-manager.yaml` after the `GOOGLE_API_KEY` block (~lines 82-86): `- name: OPENROUTER_API_KEY` / `valueFrom.secretKeyRef` `name: voipbin`, `key: OPENROUTER_API_KEY`, matching the existing blocks. Generator drift is recorded as a separate issue proposal (out of scope). `k8s/backend/secret.yaml` must carry the same key set as `BIN_SECRET_KEYS` (`tests/test_pr4_manifest_invariants.py:70-76`); edit it accordingly.
- Tests to update (counts are asserted): `tests/test_pr4_manifest_invariants.py:76` 53->54; `tests/test_pr_z_secret_schema.py:55,56` 35->36; `tests/test_secretmgr.py:23` 35->36 and `:50` 24->25 (confirm exact values by running the suite before editing).
- Note for the PR description: the installer's `pipecat-manager.yaml` has no runner sidecar (VOIP-1544), so this env entry is a wiring entry following the `GOOGLE_API_KEY` precedent; Task 1 (komodo runner services) is where the key is actually consumed.
- DO NOT touch `secrets.yaml` (it is gitignored, `.gitignore:3`, an operator-local SOPS file).
- Write the failing test updates first, run `cd install && python -m pytest tests -q`, then implement.
- Follow skills `voipbin-k8s-secret-management` and `voipbin-install-parity-audit`.
- `monorepo-etc` (deviation from design 7, deliberate): no PR, because the value is a secret only the CEO can SOPS-encrypt; Hermes writes no secret value. Deliverable is an operator note in the monorepo PR description: add `BIN_MANAGER__OPENROUTER_API_KEY` to `infra-secret/secrets-source/bin-manager/secrets.enc.yml` using a dedicated key with a credit limit and balance alert, before the backend release.
- Self-hosting doc: `bin-api-manager/docsdev/source/self_hosting_providers.rst` (monorepo repo, Task 17).
- Commit in install: `install: Add OPENROUTER_API_KEY secret schema and pipecat wiring`.

---

## Phase B: Catalog and resolver (bin-ai-manager/models/ai)

### Task 3: Catalog data and invariants
**Files:** Create `bin-ai-manager/models/ai/catalog.go`, `bin-ai-manager/models/ai/catalog_test.go`.

Step 1 (test first), `catalog_test.go`:
```go
func TestCatalogInvariants(t *testing.T) {
	seen := map[EngineModel]bool{}
	for _, e := range catalog {
		if seen[e.ID] {
			t.Errorf("duplicate id %s", e.ID)
		}
		seen[e.ID] = true
		if e.Label == "" || e.Vendor == "" {
			t.Errorf("%s: label and vendor are required", e.ID)
		}
		if e.Route == RouteOpenRouter {
			if e.UpstreamSlug == "" || strings.Contains(e.UpstreamSlug, ":") {
				t.Errorf("%s: openrouter entry needs a plain slug (no variant suffix): %q", e.ID, e.UpstreamSlug)
			}
		}
		if e.Route == RouteDirect && e.UpstreamSlug != "" {
			t.Errorf("%s: direct entry must not carry a slug", e.ID)
		}
		for _, tag := range e.Tags {
			if tag != TagLowCost {
				t.Errorf("%s: unknown tag %q", e.ID, tag)
			}
		}
	}
}

func TestCatalogPublicViewHidesInternals(t *testing.T) {
	b, _ := json.Marshal(CatalogView())
	for _, banned := range []string{"slug", "route", "openrouter", "meta-llama/"} {
		if strings.Contains(strings.ToLower(string(b)), banned) {
			t.Errorf("public catalog leaks %q", banned)
		}
	}
}

```
The catalog tests must not reference the resolver (defined in Task 4). Run `cd bin-ai-manager && go test ./models/ai/ -run Catalog -v`, expect FAIL (undefined `catalog`/`CatalogView`).

Step 2 (implement) `catalog.go`:
```go
package ai

type Route string

const (
	RouteDirect     Route = "direct"
	RouteOpenRouter Route = "openrouter"
)

const TagLowCost = "low-cost"

// ModelEntry is internal. Route and UpstreamSlug never leave the platform.
type ModelEntry struct {
	ID           EngineModel
	Label        string
	Vendor       string
	Route        Route
	UpstreamSlug string
	Recommended  bool
	Tags         []string
	Description  string
}

// ModelInfo is the customer-facing view returned by GET /ai_models.
type ModelInfo struct {
	ID              EngineModel `json:"id"`
	Label           string      `json:"label"`
	Vendor          string      `json:"vendor"`
	Recommended     bool        `json:"recommended"`
	Tags            []string    `json:"tags"`
	Description     string      `json:"description"`
	PlatformManaged bool        `json:"platform_managed"`
}

func CatalogView() []ModelInfo {
	res := make([]ModelInfo, 0, len(catalog))
	for _, e := range catalog {
		tags := e.Tags
		if tags == nil {
			tags = []string{}
		}
		res = append(res, ModelInfo{e.ID, e.Label, e.Vendor, e.Recommended, tags, e.Description, e.Route == RouteOpenRouter})
	}
	return res
}
```
`catalog` is a package var slice: the 11 existing models as `RouteDirect` (IDs from the existing constants, `Recommended: true` only for `EngineModelGeminiGemini2Dot5Flash`; `Tags: []string{TagLowCost}` assigned manually to: gemini-2.5-flash, gpt-5-mini, gpt-5-nano, grok-3-mini, and the OpenRouter entries whose published input price is at most $0.50 per M tokens at implementation time (llama-3.3-70b, llama-4-maverick, deepseek-v3.2, qwen3-235b-a22b-2507, qwen3-30b-a3b-instruct-2507, mistral-small-3.2); verify against the price pages and adjust), followed by the design 4 candidates as `RouteOpenRouter` (final inclusion is decided by Task 22; start with all nine, remove any that fail). Descriptions are one short customer-facing line each. Keep the existing `EngineModel*` constants and add constants for the new IDs only if referenced elsewhere (YAGNI: use string literals inside the catalog).
Run the tests, expect PASS. Commit `bin-ai-manager: Add model catalog`.

### Task 4: Resolver
**Files:** Create `bin-ai-manager/models/ai/resolve.go`, `resolve_test.go`; Modify `models/ai/main.go` (DELETE the existing `IsValidEngineModel` at ~205-219, since the new one lives in `resolve.go`; leave `EngineModelTargets` and the `EngineModelTarget*` constants and `GetEngineModelTarget` untouched, they no longer gate anything, and do not add `meta` (YAGNI; this supersedes the design 3.1 sentence about adding `meta`, because the list is no longer used for validation)).

Step 1 tests: also add here the test moved out of Task 3:
```go
func TestExistingElevenModelsAreDirectCatalogEntries(t *testing.T) {
	for _, id := range []EngineModel{EngineModelGeminiGemini2Dot5Flash, EngineModelOpenaiGPT5, EngineModelGrok3} {
		r, o := ResolveEngine(id)
		if o != OutcomeCatalog || r.Entry.Route != RouteDirect {
			t.Errorf("%s must stay a direct catalog entry", id)
		}
	}
}
```
and the table: `openai.gpt-5` -> Catalog direct; `anthropic.claude-haiku-4.5` -> Catalog openrouter with `RunnerType == "platform_openrouter.anthropic/claude-haiku-4.5"` and `BlankKey`; `openai.gpt-4o` -> DirectPassthrough with `RunnerType == "openai.gpt-4o"` and not `BlankKey`; `anthropic.claude-opus-4` (not in catalog) -> Rejected; `openrouter.meta-llama/llama-3-70b` -> Rejected; `platform_openrouter.x` -> Rejected; `""`, `openai.`, `unknown.x` -> Rejected.
Step 2 implement:
```go
const RunnerServicePlatformOpenRouter = "platform_openrouter"

type Outcome int

const (
	OutcomeRejected Outcome = iota
	OutcomeCatalog
	OutcomeDirectPassthrough
)

type Resolved struct {
	Entry      *ModelEntry // nil for passthrough/rejected
	RunnerType string      // what the Python runner receives
	BlankKey   bool        // true: never send the customer engine_key
}

var directPassthroughPrefixes = map[string]bool{"openai": true, "gemini": true, "grok": true}

func ResolveEngine(m EngineModel) (Resolved, Outcome) {
	for i := range catalog {
		if catalog[i].ID == m {
			e := &catalog[i]
			if e.Route == RouteOpenRouter {
				return Resolved{Entry: e, RunnerType: RunnerServicePlatformOpenRouter + "." + e.UpstreamSlug, BlankKey: true}, OutcomeCatalog
			}
			return Resolved{Entry: e, RunnerType: string(m)}, OutcomeCatalog
		}
	}
	parts := strings.SplitN(string(m), ".", 2)
	if len(parts) == 2 && parts[1] != "" && directPassthroughPrefixes[parts[0]] {
		return Resolved{RunnerType: string(m)}, OutcomeDirectPassthrough
	}
	return Resolved{}, OutcomeRejected
}

// IsValidEngineModel keeps its name so existing callers keep compiling; semantics are the new policy.
func IsValidEngineModel(m EngineModel) bool {
	_, o := ResolveEngine(m)
	return o != OutcomeRejected
}
```
Update `models/ai/main_test.go:324-344` (`TestIsValidEngineModel`) (`anthropic.claude-3` was valid, now rejected; replace with the new table) and any other test using the old prefix-only behavior (grep `IsValidEngineModel` in tests). Run `go test ./models/ai/...`. Commit `bin-ai-manager: Add fail-closed engine resolver`.

### Task 5: Validation call sites and update ordering
**Files:** Modify `bin-ai-manager/pkg/aihandler/chatbot.go` (create ~line 41, update ~line 119), `bin-ai-manager/cmd/ai-control/main.go` (~213, ~337); Test `pkg/aihandler/chatbot_test.go` (or the file holding Create/Update tests).
- Create: unchanged call `ai.IsValidEngineModel(engineModel)` (now the new policy).
- Update: REMOVE the check at ~line 119 (before `preUpdateAI` fetch) and re-add after `preUpdateAI, errGet := h.db.AIGet(...)`:
```go
if engineModel != preUpdateAI.EngineModel && !ai.IsValidEngineModel(engineModel) {
	return nil, fmt.Errorf("invalid engine model: %s", engineModel)
}
```
- ai-control update: exact rule: when `--engine-model` is non-empty, fetch the stored AI first and validate only if the value differs from the stored `EngineModel`; when empty, skip validation (as today). Create: validate when non-empty (as today). Pin with a test per case.
- The existing update test case `fails_with_invalid_model` (`chatbot_test.go` ~254, "Should not call database") must gain the `AIGet` mock expectation because validation now runs after the stored-AI fetch.
- Tests (write first): create rejects `anthropic.claude-opus-4`, accepts catalog and `openai.gpt-4o`; update with unchanged legacy `anthropic.claude-opus-4` passes; update changing to `anthropic.claude-opus-4` fails; stored value empty and requested empty passes (unchanged; an empty-model AI cannot be created, so this is unreachable in practice; pin it so the intent is explicit).
- Run `cd bin-ai-manager && go test ./pkg/aihandler/... -v`. Commit `bin-ai-manager: Validate engine model by catalog policy`.

### Task 6: ai-manager catalog RPC
**Files:** Modify `bin-ai-manager/pkg/listenhandler/main.go` (add `regV1AIModels = regexp.MustCompile("/v1/ai_models$")` near the other `/v1/ais` regexes and a `case regV1AIModels.MatchString(m.URI) && m.Method == sock.RequestMethodGet:` mirroring the `regV1AIsGet` case at ~line 302; metrics label per the existing pattern), Create `bin-ai-manager/pkg/listenhandler/v1_ai_models.go`, `v1_ai_models_test.go`.
- Handler `processV1AIModelsGet` returns `simpleResponse`-style JSON of `ai.CatalogView()` (use the same marshal/response pattern as `processV1AIsGet`, no pagination, no DB).
- Test first: request `GET /v1/ai_models` returns 200, JSON array whose items have `id`, `label`, `platform_managed`, and whose body contains no `slug`/`openrouter`.
- Also confirm `regV1AIsGet` (`/v1/ais\?`) does not match `/v1/ai_models` (add an assertion).
- Run `go test ./pkg/listenhandler/...`. Commit `bin-ai-manager: Add ai_models RPC`.

---

## Phase C: pipecat-manager (Go)

### Task 7: Session carries the resolved runner type
**Files:** Modify `bin-pipecat-manager/models/pipecatcall/session.go` (add `LLMRunnerType string \`json:"-"\`` next to `LLMKey`), `bin-pipecat-manager/pkg/pipecatcallhandler/session.go` (`SessionCreate` gains `llmRunnerType string` before `llmKey`), `runner.go` (~207-212), plus every caller and mock/test (`start.go:94,200,277`, tests ~20 call sites; update via grep `SessionCreate(`).
- Interim wiring until Task 8: every `SessionCreate` call site temporarily passes `string(pc.LLMType)` as `llmRunnerType` so the tree compiles and behaves as today (Task 8 replaces it with the resolver output).
- `runner.go`: pass `se.LLMRunnerType` to `pythonRunner.Start` instead of `string(pc.LLMType)`. If `se.LLMRunnerType == ""` return an error `"session has no resolved llm type"` (never fall back to `pc.LLMType`).
- Existing tests that build `pipecatcall.Session` literals without the new field must set `LLMRunnerType` or they hit the new empty-value error: `runner_mcptools_test.go:67,123`, `runner_toolfallback_test.go:50`, `start_test.go:283,361`, plus the 16 `SessionCreate(` calls in `session_test.go` (grep to confirm).
- Because `llmRunnerType` and `llmKey` are adjacent strings, add a session test asserting both are stored in the right fields (a swapped order must fail).
- Tests first: a `runner_test.go` case (create the file if the existing runner tests live elsewhere; place it beside `runner_flush_test.go`) with empty `LLMRunnerType` expects error and no `pythonRunner.Start` call; case with value passes it.
- `SessionCreate` is a method on the handler and is not part of the mocked interface (`main.go`), so no mock regeneration is needed; update the ~16 test call sites (grep `SessionCreate(`).
- Commit `bin-pipecat-manager: Carry resolved llm type on the session`.

### Task 8: resolveSessionLLM and fail-closed Start
**Before the tests:** `cd bin-pipecat-manager && go mod tidy && go mod vendor` (it now imports the new `ai.ResolveEngine`).
**Files:** Create `bin-pipecat-manager/pkg/pipecatcallhandler/llmresolve.go`, `llmresolve_test.go`; Modify `start.go` (`Start` before `h.Create`; `startReferenceTypeAIcall` ~187-200; `startReferenceTypeCall` ~79-94; `run.go:113-130` `runGetLLMKey`).
```go
// resolveSessionLLM maps the customer-facing model to what the runner receives.
// aiKey is the customer's engine_key ("" when unknown).
func resolveSessionLLM(llmType pipecatcall.LLMType, aiKey string) (runnerType string, runnerKey string, err error) {
	r, outcome := amai.ResolveEngine(amai.EngineModel(llmType))
	if outcome == amai.OutcomeRejected {
		return "", "", fmt.Errorf("engine model is not available: %s", llmType)
	}
	if r.BlankKey {
		return r.RunnerType, "", nil
	}
	return r.RunnerType, aiKey, nil
}
```
- `Start()`: first statement `if _, _, err := resolveSessionLLM(llmType, ""); err != nil { return nil, errors.Wrap(err, "...") }` (keyless classification, before `h.Create`, so no DB row is written for a rejected model).
- Each of the three `SessionCreate` sites: call `resolveSessionLLM(pc.LLMType, <key from existing lookup or "">)` and pass `(runnerType, runnerKey)`; an error returns from that start path. A failed AI lookup keeps today's warn-and-proceed ONLY for direct outcomes (the helper resolves from `pc.LLMType` with no RPC, so a failed lookup cannot bypass it).
- Tests first (table in `llmresolve_test.go` plus start tests): Rejected returns error and `h.Create` is never called; catalog openrouter returns `platform_openrouter.<slug>` and empty key even when the AI has a key; direct keeps the key; passthrough keeps the key; AI lookup failure + stored `openrouter.x` still errors; a table test that all three session-setup sites call the helper (use a mutation: temporarily pass `string(pc.LLMType)` and confirm a test goes red).
- Update existing fixtures that used legacy values the resolver now rejects, in the same task: `anthropic.claude-2` in `bin-pipecat-manager/models/pipecatcall/main_test.go:95` and any `anthropic.*`/other non-catalog, non-direct-prefix model in `run_test.go`/`start_test.go` (use catalog or `openai.*` values).
- Commit `bin-pipecat-manager: Resolve llm type fail-closed at session start`.

### Task 9: Team members carry llm_type
**Files:** Modify `bin-pipecat-manager/pkg/pipecatcallhandler/run.go` (`resolvedAIData` add `LLMType string \`json:"llm_type"\`; in `resolveTeamForPython` ~line 200-201 compute with the same resolver: on Rejected set `LLMType: ""` and log an operator error naming the member id and customer id; on BlankKey set `EngineKey: ""`), `run_test.go` (~line 1024 literal).
- `EngineModel` stays `string(ai.EngineModel)` (customer id; feeds `member_switched`).
- Tests first: openrouter member gets `LLMType == "platform_openrouter.<slug>"`, empty key, customer `EngineModel`; direct member unchanged; Rejected member gets empty `LLMType` and an error log; the non-current member case.
- Commit `bin-pipecat-manager: Add llm_type to resolved team members`.

### Task 10: Error classification for OpenRouter credit exhaustion
**Files:** Modify `bin-pipecat-manager/pkg/pipecatcallhandler/pipelineerror.go` (add `"insufficient credits"`, `"more credits"` to `pipelineErrorPhrasesRateLimitedText`), `pipelineerror_test.go`.
- Tests first: text containing `insufficient credits` classifies as rate_limited; `shouldNotifyPipelineError(rate_limited, false, true)` is true (voice session notifies).
- Commit `bin-pipecat-manager: Classify OpenRouter credit exhaustion as quota`.

---

## Phase D: Python runner (bin-pipecat-manager/scripts/pipecat)

### Task 11: platform_openrouter branch
**Files:** Modify `run.py` (import `from pipecat.services.openrouter.llm import OpenRouterLLMService`; add branch before the final `else` in `create_llm_service`), `test_run.py`.
```python
    elif service_name == "platform_openrouter":
        # The key argument is intentionally ignored: customers never supply it.
        api_key = os.getenv("OPENROUTER_API_KEY", "")
        if not api_key:
            raise ValueError("OpenRouter is not configured")
        llm = OpenRouterLLMService(
            api_key=api_key,
            settings=OpenRouterLLMService.Settings(
                model=model_name,
                extra={"extra_body": {"provider": {
                    "zdr": True,
                    "data_collection": "deny",
                    "require_parameters": True,
                }}},
            ),
        )
        standard_tools = _openai_tools_to_standard(tools)
        tools_schema = ToolsSchema(standard_tools=standard_tools) if standard_tools else NOT_GIVEN
        ctx = LLMContext(messages=valid_messages, tools=tools_schema)
        aggregator = _make_aggregator(ctx)
        return llm, aggregator
```
- Tests first (mock `OpenRouterLLMService` like the existing grok test): called with env key and not the customer key; `settings.extra["extra_body"]["provider"]` equals the dict above; model is the slug (e.g. `anthropic/claude-haiku-4.5`, split on the first `.` only); missing/empty env raises `ValueError` (an invalid non-empty key is NOT special-cased; it surfaces as the provider's 401, as for other providers); raw `openrouter` service raises `Unsupported LLM service`; `platform_openrouter:a.b` (colon) with a customer fallback source raises per Task 12.
- Also add a test that proves `extra` reaches the OpenAI client as `extra_body` (instantiate the real `OpenRouterLLMService` with a fake key, call `build_chat_completion_params` with minimal context params, assert `params["extra_body"]["provider"]["zdr"] is True`).
- Run `cd bin-pipecat-manager/scripts/pipecat && python -m pytest test_run.py -q`. Commit `bin-pipecat-manager: Add platform_openrouter runner branch`.

### Task 12: Team llm_type and fallback guard
**Files:** Modify `main.py` (`ResolvedAI` add `llm_type: Optional[str] = None`), `run.py:770`, `team_flow.py` (no change expected, assert in test), `test_team_flow.py`, `test_routing_llm.py`, `test_init_pipeline.py:111-128,184-201`, `test_run.py`.
- At `run.py:770` replace `create_llm_service(ai["engine_model"], ai["engine_key"], ...)` with a helper. `ResolvedAI.llm_type` is `Optional[str] = None`: `None` means an older Go that does not send the field; `""` means Go rejected the member (Go always sends the field, no `omitempty`).
```python
def _member_llm_type(ai: dict) -> str:
    resolved = ai.get("llm_type")
    if resolved is None:
        # Older Go (version skew): legacy behavior, but never honor routed/internal services.
        fallback = ai["engine_model"]
        svc = fallback.replace(":", ".", 1).split(".", 1)[0].lower() if fallback else ""
        if svc in ("platform_openrouter", "openrouter"):
            raise ValueError("engine model is not available")
        return fallback
    if resolved == "":
        raise ValueError("engine model is not available")
    return resolved
```
- This closes the round-1 plan finding: a Go-rejected member (`""`) never falls back to the raw `engine_model` for ANY provider (including case variants such as `OpenAI.gpt-4o` that Go rejects).
- Tests first: member with `llm_type` uses it; `llm_type == ""` raises for ANY `engine_model` (openrouter-like and direct); `llm_type is None` with a direct `engine_model` keeps legacy behavior and with an openrouter-like one raises; a mixed direct + OpenRouter team builds both services; `team_flow._build_member_info` still returns the customer `engine_model`; update the existing tests that assumed unsupported providers.
- Commit `bin-pipecat-manager: Use resolved llm_type for team members`.

### Task 13: Catalog verification script (manual)
**Files:** Create `bin-pipecat-manager/scripts/pipecat/verify_openrouter_catalog.py` and `test_verify_openrouter_catalog.py`.
- Matching rules: compare slugs by exact `model_id` equality (never substring, `...-v3.2` must not match `...-v3.2-exp`); parse `UpstreamSlug:` with a regex tolerant of gofmt alignment (`UpstreamSlug:\s*"([^"]+)"`); put the unit test next to the script under `bin-pipecat-manager/scripts/pipecat/` so the existing pytest invocation picks it up (place the script there too).
- Reads slugs directly from `bin-ai-manager/models/ai/catalog.go` (`go run` helper not required: the script reads it by regex for `UpstreamSlug: "..."`), fetches `https://openrouter.ai/api/v1/endpoints/zdr`, exits non-zero if a slug has no ZDR endpoint with `tools` in `supported_parameters`. Not wired into CI. Add a unit test for the parsing function using a small fixture.
- Commit `bin-pipecat-manager: Add manual OpenRouter catalog verifier`.

---

## Phase E: API chain and docs

### Task 14: requesthandler RPC
**Files:** Modify `bin-common-handler/pkg/requesthandler/ai_ais.go` (add `AIV1AIModelList(ctx) ([]amai.ModelInfo, error)` (`go mod tidy && go mod vendor` first; the module must see the new `ModelInfo`) using `sendRequestAI(ctx, "/v1/ai_models", sock.RequestMethodGet, "ai/ai_models", requestTimeoutDefault, 0, ContentTypeNone, nil)` and `parseResponse`), `main.go` (interface ~line 217), regenerate `mock_main.go`, add a test in `ai_ais_test.go` following the `AIV1AIList` test.
- Commit `bin-common-handler: Add AIV1AIModelList`.

### Task 15: OpenAPI, api-manager handler, generated code
**Files:** Create `bin-openapi-manager/openapi/paths/ai_models/main.yaml` (GET, tag AI, 200: `allOf [CommonPagination, {result: array of AIManagerAIModel}]`, no page params, 401/500 refs); Modify `bin-openapi-manager/openapi/openapi.yaml` (`paths:` register `/ai_models` next to `/ais`; add schema `AIManagerAIModel` {id, label, vendor, recommended, tags, description, platform_managed}; relax `AIManagerAIEngineModel` to `type: string` with description pointing to `GET /ai_models`, remove enum and x-enum-varnames); Create `bin-api-manager/server/ai_models.go` (`GetAiModels`, auth identity check as in `GetAis`, call `serviceHandler.AIModelList`), `bin-api-manager/pkg/servicehandler/ai.go` (`AIModelList(ctx, a) ([]*amai.ModelInfo, error)` calls `reqHandler.AIV1AIModelList` and converts to a pointer slice because `GenerateListResponse[T]` (`server/main.go:91`) takes `[]*T`; any authenticated identity may read it, no direct-hash restriction since the catalog is non-sensitive and read-only), interface in `pkg/servicehandler/main.go`, regenerate mock; tests for server and servicehandler.
- Regenerate in order: `cd bin-openapi-manager && go generate ./...` then `cd ../bin-api-manager && go generate ./...`; commit generated files (`gens/`) as the repo does.
- Verify no non-generated Go code referenced `AIManagerAIEngineModelXxx` (`grep -rn AIManagerAIEngineModel --include=*.go . | grep -v gens/`), expect empty.
- Run the full verification workflow in `bin-openapi-manager` and `bin-api-manager`.
- Commit `bin-openapi-manager, bin-api-manager: Add GET /ai_models and relax engine model enum`.

### Task 16: Generated artifacts check
- After Task 15, run `git status --short bin-openapi-manager/gens bin-api-manager/gens` and confirm the redoc `openapi.json` and `api.html` changed (they require `npx`; if `npx` is missing, install it or regenerate elsewhere, never commit a stale redoc silently). Confirm `gens/openapi_server/gen.go` and `bin-openapi-manager/gens/models/gen.go` contain `AIManagerAIModel` and no `AIManagerAIEngineModelGemini...` constants.
- Commit generated files with their source change if not already committed in Task 15.

### Task 17: RST docs
**Files:** `bin-api-manager/docsdev/source/ai_struct_ai.rst` (rewrite the provider table ~lines 196-211: customer prefixes only, no `openrouter.<model>`, point to `GET /ai_models`), `ai_overview.rst`, `ai_tutorial.rst`, `team_tutorial.rst`, `aicall_struct_aicall.rst`, `variable_variable.rst` (fix stale model examples), a new `ai_models` API reference page registered in the `ai.rst` toctree (`ai.rst:10-24`; adding the toctree entry is enough, the top-level-section skill is more than needed), `self_hosting_providers.rst` (add the OpenRouter key row near lines 28, 63, 73; do not overstate: the installer's k8s `pipecat-manager.yaml` has no Python runner container (removed in VOIP-1544), so the OpenRouter env there is a wiring entry like `GOOGLE_API_KEY`, and where the runner executes is deployment-specific; describe the real behavior: the installer seeds a random key, and an unconfigured OpenRouter key yields OpenRouter's 401 classified as an authentication error, like any other provider key).
- Build docs per the repo procedure (skill `sphinx-rst-audit`); commit the rebuilt HTML if the repo tracks `docsdev/build`.
- No customer-facing text may name OpenRouter except self-hosting operator docs.
- Commit `bin-api-manager: Update AI model docs for catalog`.

---

## Phase F: UI (monorepo-javascript/square-admin; own worktree and PR)

### Task 18: Hook and ModelPicker
**Files:** Create `square-admin/src/views/ais/useAIModels.js`, `square-admin/src/views/ais/ModelPicker.js`, tests `useAIModels.test.js`, `ModelPicker.test.js`; Modify `aisApi.js` (add `fetchAIModels`).
- `useAIModels()`: also returns `getModel(id)` (full entry or undefined) and `isPlatformManaged(id)` so parents that own the engine-key input (`ais_create.js`, `ais_detail.js`, sidebar) can react; single fetch of `GET /ai_models` cached at module level for the session (cache the in-flight promise at module level so concurrently mounted team-graph member nodes trigger one `GET /ai_models`; expose a `resetAIModelsCache()` used by tests in `beforeEach`); returns `{models, status: 'loading'|'ready'|'error', retry, modelLabel(id)}`; `modelLabel` returns the catalog label or the raw id.
- `ModelPicker` props: `value`, `onChange`, `disabled`. Behavior per design 3.7: trigger shows label and vendor; popover with search; "Recommended" section then vendor groups; rows show label, description and tags; synthetic first row "Current: <id>" when `value` is non-empty and not in the catalog; no "Custom..." entry; loading state and error state with Retry.
- Tests first for each behavior listed (search, Recommended, grouping, synthetic row, no custom, error/retry).
- Catalog load failure at form level (design 3.7): create forms disable Save with an inline retry message; edit forms keep showing the stored raw value and never block other fields. Tests for both.
- Commit `square-admin: Add ModelPicker and AI model catalog hook`.

### Task 19: Replace every consumer
**Files:** `AIEngineFields.js` (~149, ~360), `ais_create.js` (45, 178, 194, engine key ref 84/196/573-576), `ais_detail.js` (80, 177-203, 202, 411, 533, 557, engine key 173/413/532/1293-1296), `teamgraph/sidebar.js` (create form 128-168/418/509/975, edit form 648-651/790/946/1307/1359-1373), label lookups in `ais_list.js`, `InsightAIsPanel.js`, `teamgraph/nodes/member.js`, `aicalls_list.js:64`, `aicalls_detail.js:645`, `TestAgentSheet.js:88,95`, `constants.js` (remove `ENGINE_MODELS` after the last consumer is gone), `types/api.ts` (`AIEngineModel = string`; add `AIModel` interface).
- Remove `customModelInput`/`checkAndSetCustomInput`/`getFinalValue` use for the MODEL only (TTS/STT `custom_input` untouched).
- Engine key: when the selected model is `platform_managed` the input stays mounted and `disabled` with helper text "API key not required"; `ais_create.js:196` submits an empty `engine_key` when the selected model is `platform_managed` (never the stale typed text); `ais_detail` save payload sends the stored `engine_key` unchanged and ignores new typed text; dirty comparison (`ais_detail.js:532`) skips the engine-key term for platform-managed models; sidebar edit form treats `engineKeyChanged` as false and hides "Key will be updated on save" (~line 1373); sidebar create form omits the key.
- `ais_list.test.js` does not mock `src/provider` today; add the `jest.mock('src/provider', ...)` for `Get('ai_models')` pattern used by the other tests.
- Tests: update `AIEngineFields.test.js:292,312`, `ais_create.test.js:57,749`, `ais_detail.test.js:125`, `sidebar*.test.js`, `member.test.js`, `ais_list.test.js`, `InsightAIsPanel.test.js`, `aicalls_*test.js`; add tests for the disabled engine-key behavior in all four forms and for the unchanged legacy value saving without a dirty flag.
- `aicalls_list.js:64` renders the raw `ai_engine_model` accessor today; converting it to a label is optional (a hook must be called inside a component, e.g. a cell renderer component).
- `teamgraph/nodes/member.js` `getProviderStyle` (lines ~36-39) colors by model-id prefix: add styles for `anthropic`, `meta`, `deepseek`, `qwen`, `mistral` (fallback exists, add explicit entries). `aicalls_detail.js:374,379` (team member `engine_model`) render through `modelLabel(id)` too.
- Confirm by grep that `square-admin_new`, `square-talk`, `square-main` have no AI model form (reviewers already found none); do not touch them.
- Run `cd square-admin && npm test -- --watchAll=false` and the repo lint/build; follow skill `voipbin-frontend-visual-verification-gate` (screenshots of create, detail, team sidebar and the picker, including search and the legacy row, using mock catalog data via `voipbin-frontend-mock-render-screenshot`).
- Commit `square-admin: Replace engine model lists with ModelPicker`.

### Task 20: Agent-facing documents (same JS PR)
**Files:** `square-main/public/skill.md` (LLM Providers table ~805-811 and examples at ~411, 621, 909), `square-main/public/llms.txt:~49`; do not hand-edit `build/` copies (regenerated by the normal build).
- Align with the real catalog and `GET /ai_models`.
- Commit `square-main: Align agent docs with the AI model catalog`.

---

## Phase G: Integration, real verification, delivery

### Task 21: Re-vendor and full backend verification
- For every Go service that vendors `monorepo/bin-ai-manager` or `monorepo/bin-common-handler` (find with `grep -rl "monorepo/bin-ai-manager" --include=go.mod .`): `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`. `vendor/` is gitignored, so nothing is committed from this step; it is verification only (each service resolves monorepo modules through `replace => ../`). Also include services that consume `bin-openapi-manager` gens besides api-manager.

### Task 22: Real-call verification (uses the existing OpenRouter key in `~/.hermes/.env` as authorized by the CEO on 2026-10-05; read from the environment only, never echoed, printed, committed or logged; it is used for verification only, not placed in any production secret)
- For each catalog candidate: one chat completion with a tool definition through the real `OpenRouterLLMService` configuration (same `extra_body.provider`), assert a tool call returns, record the sub-provider that answered; drop any model that fails from `catalog.go` (and its test expectations) and rerun Task 3 tests.
- One Listen turn (`startListenPipecatcall` path) with an OpenRouter model.
- One pipecat runner text session (`create_llm_service` with `platform_openrouter.<slug>`) and one mixed direct + OpenRouter team init.
- Negative: an empty key produces `OpenRouter is not configured`; an invalid non-empty key yields the provider authentication error classified as authentication.
- Measure first-token latency for 3 representative models over repeated calls; set `Recommended` only for models with evidence, otherwise none beyond `gemini.gemini-2.5-flash`.
- Run `verify_openrouter_catalog.py` against the final catalog.
- Streaming and barge-in: for one OpenRouter model start a text session, interrupt mid-stream, confirm the stream cancels and the next turn works; record the idle-timeout decision (design 3.3). Confirm ZDR actually applied by checking that the answering sub-provider reported in the response metadata is a provider listed with a ZDR endpoint for that slug in the public ZDR list.
- `~/.hermes/.env` holds two `OPENROUTER_API_KEY=` lines; confirm which one the shell loads without printing values.
- Record results in the PR description (no keys, no raw transcripts).

### Task 23: Mutation checks
Temporarily break each guard (also the empty `LLMRunnerType` error at the runner and the update-validates-only-on-change condition; backend and UI: also the disabled engine key, the dirty-comparison skip, `engineKeyChanged=false`, and the synthetic legacy row), confirm a test fails, then revert: drop key blanking (Task 8/9), pass `string(pc.LLMType)` unresolved at the runner (Task 7), remove the catalog membership check (Task 4/5), remove `zdr` (Task 11), remove the Python fallback guard (Task 12), remove the `ResolveEngine` rejection of raw `openrouter.*` (Task 4). Record each in the PR notes.

### Task 24: Review loop, delivery
- Gate semantics (CLAUDE.md): a CHANGES_REQUESTED round resets the consecutive-APPROVE count to zero; the loop stops at 30 rounds without 2 consecutive APPROVEs and the state is reported to the CEO. Author check covers every commit: `git log origin/main..HEAD --format=%ae` must list only `pchero21@gmail.com`.
- Per policy: code review loop (min 3 rounds, then 2 consecutive APPROVE, 3 fresh parallel reviewers per round with distinct angles) on each PR before reporting it ready; fix and re-review on any CHANGES_REQUESTED.
- Before creating each PR: `git fetch origin main`, conflict check, `git log -1 --format=%ae` equals `pchero21@gmail.com`.
- PR titles equal branch names (`NOJIRA-Route-LLM-via-OpenRouter`), body per the monorepo convention (narrative + `project: change` bullets, no headers, no test plan, no AI attribution). One PR per repository: monorepo, monorepo-javascript, install.
- Never merge without the CEO's explicit instruction; squash merge only; after merge pull main in the base checkout.
- Read the PR's real CI checks before reporting ready.

## Scope notes
Out of scope and unaffected: `bin-ai-manager/pkg/engine_openai_handler` (`message.go:31`, `streaming_send.go:43` call `GetEngineModelName`) has no production callers (analysis F7: summary and analysis use the platform engine configuration), so catalog models never reach it.

## Rollout notes (carry into PR descriptions)
Service order: pipecat-manager (Go + runner, same release) -> ai-manager -> api-manager -> UI. Operator checklist: dedicated OpenRouter key with credit limit and balance alert registered in SOPS before the backend release; pre-merge read-only query of stored `engine_model` values (skill `voipbin-prod-readonly-inspection`) as a confirmation step. Out of scope with triggers (also record as follow-up issue proposals: OpenRouter 404 "no endpoints" under ZDR + require_parameters is classified `unknown` and silent in voice sessions; install manifest generator drift): per-account quotas, usage metering, outage fallback, credit-exhaustion alert rule, synchronous team pre-check, separate issue for the sidebar edit form omitting `engine_key`.
