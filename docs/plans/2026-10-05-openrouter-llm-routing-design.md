# Design: Route additional LLMs through OpenRouter (platform key, hidden from customers)

Status: APPROVED (design review loop closed: rounds 4 and 5, both reviewers APPROVED each round; minor wording items applied)
Branch: NOJIRA-Route-LLM-via-OpenRouter
Date: 2026-10-05
Prerequisite: `2026-10-05-openrouter-llm-routing-issue-analysis.md` (APPROVED x2; F#/R# below refer to it)

## 1. Decisions locked by CEO

- Platform holds the OpenRouter key. Customers never see or supply it (customer `engine_key` ignored for OpenRouter-routed models).
- Free hosted accounts may use the new models. Per-account quota is deferred until abuse signals appear (no speculative build).
- OpenRouter is added as a sub-processor in the privacy policy and DPA (wording is legal work, outside this PR). Requests use ZDR routing by default.
- Add Claude, Llama, DeepSeek, Qwen, Mistral families. The UI must be a designed picker, not a flat list.

## 2. Verified facts used by the design (new in this stage)

| # | Fact | Evidence |
|---|------|----------|
| D-F1 | `OpenAILLMService.build_chat_completion_params` merges `self._settings.extra` into the top-level kwargs of `chat.completions.create(**params)` | pipecat-ai 1.12.0 wheel `pipecat/services/openai/base_llm.py:363-395` |
| D-F2 | Therefore OpenRouter's `provider` object cannot be a top-level kwarg (the OpenAI SDK would reject it); it must travel inside `extra_body`: `extra={"extra_body": {"provider": {...}}}`. To be proven by a unit test and a real call | same file |
| D-F3 | OpenRouter `provider` options exist: `zdr` (bool), `data_collection` ("allow"/"deny"), `require_parameters`, `allow_fallbacks`, `order`, `only`, `ignore` | https://openrouter.ai/docs/features/provider-routing (fetched 2026-10-05) |
| D-F4 | OpenRouter publishes a public ZDR endpoint list (`/api/v1/endpoints/zdr`, 928 endpoints, 335 model IDs, includes `supported_parameters` per endpoint) and a public model list (`/api/v1/models`, 466 models). ZDR still covers current Claude, Llama 3.x/4, DeepSeek V3.x/V4, Qwen3.x, Mistral models with tool support | fetched 2026-10-05 |
| D-F5 | `pipecat-manager` already imports `bin-ai-manager` models (e.g. `amteam`, `aitool`) and is the single Go hop that hands `llm_type` (`runner.go:210`, `pythonrunner.go:100`) and team member `engine_model`/`engine_key` (`run.go:200-201`) to the Python runner | `bin-pipecat-manager/pkg/pipecatcallhandler/` |

## 3. Architecture

### 3.1 One catalog, one resolver (Go, in bin-ai-manager/models/ai)

A single static catalog is the source of truth for every consumer.

```
type ModelEntry struct {
    ID           EngineModel // customer-facing id, e.g. "anthropic.claude-sonnet-4.5"
    Label        string      // "Claude Sonnet 4.5"
    Vendor       string      // display group: "Anthropic", "Google", ...
    Route        Route       // RouteDirect | RouteOpenRouter   (internal, never serialized)
    UpstreamSlug string      // OpenRouter slug, only for RouteOpenRouter (internal, never serialized)
    Recommended  bool        // see 3.7: set only with evidence (platform default or measured)
    Tags         []string    // closed set, v1: "low-cost" only (assigned manually per entry; OpenRouter-routed entries are checked against published OpenRouter pricing, direct entries against the provider price page); "fast" is added only after latency is measured
    Description  string      // one short customer-facing line
}
```

API-visible fields (customer-facing): id, label, vendor, recommended, tags, description, `platform_managed` (derived: true when Route is RouteOpenRouter; meaning "no API key needed"). Route and slug are never serialized.

- Existing 11 models become `RouteDirect` entries with unchanged IDs (no behavior change).
- New models are `RouteOpenRouter`. Customer IDs use the vendor prefix the customer expects
  (`anthropic.`, `meta.`, `deepseek.`, `qwen.`, `mistral.`); the upstream slug lives only in the catalog
  (`meta.llama-3.3-70b-instruct` -> `meta-llama/llama-3.3-70b-instruct`). Single alias location resolves analysis R15/R16.
- `ResolveEngine(EngineModel) (Resolved, Outcome)` is the only resolver. Outcomes (resolves round-1 legacy-value findings, no regression for working customers):
  1. `Catalog`: in the catalog; `Route` is direct or openrouter.
  2. `DirectPassthrough`: prefix is `openai`/`gemini`/`grok` but the model name is not in the catalog (e.g. a stored `openai.gpt-4o`, `gemini.gemini-1.5-pro`). Behaves EXACTLY as today (customer `engine_key`, else platform env fallback). No new cost exposure and no regression.
  3. `Rejected`: everything else, including non-catalog `anthropic.*`/`meta.*`/`deepseek.*`/`qwen.*`/`mistral.*` and any `openrouter.*`. These never worked at runtime, so rejecting them removes nothing, and it guarantees stored free-text values cannot start spending platform OpenRouter money.
- Create/update validation replaces prefix-only `IsValidEngineModel`: accept `Catalog` and `DirectPassthrough`, reject `Rejected`; on update validate only when `engine_model` differs from the stored value so legacy rows stay editable for other fields (implementation note: today `chatbot.go:119` validates before the stored AI is fetched; the check moves after the `preUpdateAI` fetch and compares values, with a test, because the UI always sends `engine_model`). API write contract: tightened only for prefixes that could not work before; direct-prefix behavior is unchanged (explicitly NOT a breaking change for current direct callers).
- `EngineModelTargets` and `GetEngineModelTarget` are left untouched (no longer used for validation; plan stage: `meta` is NOT added); `openrouter` stays defined (legacy reads) but is `Rejected` for new values.
- Pre-merge step 0 (read only, informational): query prod `ai` and team member rows for distinct `engine_model` to confirm no stored `Rejected` values exist (skill voipbin-prod-readonly-inspection). With the policy above, this is a confirmation, not a gate.

### 3.2 Where the resolver is applied (server-side, fail-closed)

Verified facts: every ai-manager Start call passes the customer ID as `LLMType` (`aicallhandler/start.go:~960,998`, `listen.go:~420`); pipecat-manager hands it to Python at `runner.go:210-211` and `pythonrunner.go:100`; the team path is resolved in `resolveTeamForPython` called from `runner.go:~122` INSIDE `runnerStartScript`, which runs in the `RunnerStart` goroutine (`start.go:247/284`), so errors there are only logged (`runner.go:84-90`).

1. ai-manager create/update (`aihandler/chatbot.go:41,119`, `cmd/ai-control`): validation per 3.1. This is the primary gate: every AI row (including team member AIs) is validated when written.
2. pipecat-manager single helper `resolveSessionLLM(llmType pipecatcall.LLMType, aiKey string) (runnerType string, runnerKey string, err error)` used at ALL start sites that set up a session: `startReferenceTypeAIcall`, `startReferenceTypeCall`, and the key derivation in `runGetLLMKey` (`start.go:79-91,187-194`, `run.go:113-130`). It resolves `pc.LLMType` with NO RPC (so an AI lookup failure cannot bypass it) and is fail-closed:
   - `Catalog` direct / `DirectPassthrough`: type and key exactly as today (a failed AI lookup keeps today's warn-and-proceed key behavior for these only).
   - `Catalog` openrouter: `runnerType = "platform_openrouter.<UpstreamSlug>"`, `runnerKey = ""`.
   - `Rejected` (including raw `openrouter.*`): returns an error. Single-AI/Call sessions resolve in the synchronous part of Start BEFORE the DB row (`h.Create`) is created, so the Start RPC returns an error to ai-manager, which already propagates Start errors (`start.go:~279,409,620`, `send.go:138`).
   The resolved value is kept in the in-memory session next to `LLMKey` and `runner.go:210-211` passes it; stored `pc.LLMType`, aicall records, webhooks, variables and `member_switched` keep the customer ID.
3. Distinct internal service name (round-2 finding): the runner-bound type for routed models uses the internal prefix `platform_openrouter.` which no customer can store (not a valid customer prefix), so a raw legacy `openrouter.<model>` still hits `Unsupported LLM service` exactly as today and can never reach the platform key, even during version skew or if a Go-side check were bypassed.
4. Team members: `resolvedAIData` (`run.go:200-201`) gets a NEW `LLMType` field (`llm_type`) computed by the same resolver, with `EngineKey` blanked for routed members; `EngineModel` stays the customer ID. Python `ResolvedAI` (`main.py:40`) gains optional `llm_type`; `run.py:770` uses `ai.get("llm_type") or ai["engine_model"]`, which is safe because a raw `openrouter.*` fallback value is unsupported by the runner (point 3), and `team_flow.py:205` keeps emitting the customer ID (fixes the `member_switched` slug leak, `messagehandler/event.go:370,380`).
   - Team Rejected members (accepted limitation, decided): resolution stays in the async `resolveTeamForPython` (moving it into the synchronous part would serialize `AIV1TeamGet` + one `AIV1AIGet` per member before audio setup and add call-start latency). Fail-closed applies: a Rejected member gets an empty `llm_type`, Python raises at pipeline init, the session ends and an operator error log names the member. This is only reachable for team member AIs whose value was stored before this change and never worked (new writes are validated by point 1). Trigger for a synchronous pre-check: any observed occurrence.
Python guards (round-3 hardening): `platform_openrouter` is honored ONLY when it arrives via the Go-resolved `llm_type` field; the `run.py:770` fallback to `engine_model` rejects any value whose service name (after lowercasing, with either `.` or `:` separator) is `platform_openrouter`, and an empty resolved value (Rejected team member) raises instead of falling back to raw `engine_model` for routed prefixes. Session value empty at `runner.go:210-211` returns an error rather than falling back to `pc.LLMType`. `resolveSessionLLM` roles: a keyless classification call in `Start()` before `h.Create` (rejects), and the keyed call at each session-setup site after the AI lookup (decides key).
5. The Python runner needs NO catalog: it receives direct types it already supports plus `platform_openrouter.<slug>`.

Rollout coupling: pipecat-manager Go and the Python runner ship in the same release; old Go + new Python is harmless (raw `openrouter.*` is unsupported), new Go + old Python fails closed (`platform_openrouter` unsupported).

### 3.3 Python runner (`bin-pipecat-manager/scripts/pipecat/run.py`)

New branch in `create_llm_service` (`service_name == "platform_openrouter"`; raw `openrouter` is intentionally NOT handled):

- `api_key = os.getenv("OPENROUTER_API_KEY")`; the `key` argument is ignored by design. If empty raise `ValueError("OpenRouter is not configured")` (the runner returns 400 via `main.py:139-144`; in pipecat-manager that start error is logged and ends the session, so the user-visible signal is the pipecat-manager sync pre-check below plus an operator log). The synchronous pre-check in 3.2 covers Rejected values only; a missing platform key is an OPERATOR condition surfaced by the runner error log and by the real-call verification (analysis R5; install seeds every secret-class key with a random value, so an unconfigured self-host holds an invalid non-empty key and the call fails with the provider's 401 classified as authentication, exactly like other providers; documented in self-hosting docs. Plan-stage clarification 2026-10-05: no `dummy` special casing).
- `OpenRouterLLMService(api_key=..., settings=OpenRouterLLMService.Settings(model=slug, extra={"extra_body": {"provider": {"zdr": True, "data_collection": "deny", "require_parameters": True}}}))`.
  - `zdr` + `data_collection: deny`: privacy default (CEO decision).
  - `require_parameters`: only endpoints that support `tools`/`tool_choice` answer (analysis R1).
  - `allow_fallbacks` left at default true: fallback is restricted to the ZDR + parameter-compatible set.
- Tools/context built exactly like the grok branch (`_openai_tools_to_standard`, `LLMContext`, `_make_aggregator`).
- Model name with `:` suffix variants is impossible: catalog slugs are fixed strings without variants (analysis R15).
- `stream_idle_timeout_secs`: not set by default for the OpenAI-compatible class; barge-in/idle behavior is verified in the real call test (analysis R14). If the real test shows a stuck stream, add a setting then.

### 3.4 Secrets and config (analysis R8, R8b)

- `OPENROUTER_API_KEY` is read only by the pipecat runner services.
- monorepo: `bin-pipecat-manager/komodo/docker-compose.yml` runner services get `OPENROUTER_API_KEY=[[BIN_MANAGER__OPENROUTER_API_KEY]]`.
- monorepo-etc: add `BIN_MANAGER__OPENROUTER_API_KEY` to `infra-secret/secrets-source/bin-manager/secrets.enc.yml` (SOPS; requires the real key from CEO).
- install repo: `scripts/secret_schema.py` entry (also the service-wiring mapping near line 489 and the `BIN_SECRET_KEYS` count assertion near line 129), (`install/secrets.yaml` is gitignored and is NOT edited), with `{"default": "dummy-openrouter-key", "class": "secret"}`, update the key-count sanity assertion, `k8s/backend/secret.yaml` and `k8s/backend/services/pipecat-manager.yaml` env, and the self-hosting docs `self_hosting_providers.rst`.
- Operator doc `bin-pipecat-manager/docs/operations.md` env table.
- Self-host with an unset key: the runner raises `OpenRouter is not configured` at init (empty key). With the installer's random seeded key, the call fails with OpenRouter's 401 classified as an authentication error (same as every other provider key); the model stays selectable in the catalog (no global flag).

### 3.5 Error classification (analysis R6)

- Extend `pipelineerror.go` phrase lists with OpenRouter credit exhaustion text ("insufficient credits", "more credits") under the rate-limited/quota category so voice sessions notify (rate_limited always notifies, `shouldNotifyPipelineError`).
- No new category and no new customer wording (the existing authentication notice is conditional and the quota notice is accurate).
- Operator signal: the existing truncated raw-error log line (`runner.go:723`) is the operator trail; a dedicated alert rule is a follow-up with a trigger (first real credit exhaustion).

### 3.6 Catalog API (replaces the four hand-synced lists, analysis F8/R12/R17)

Full change chain (round-1 finding: do not under-scope):
- bin-ai-manager: catalog Go package, listenhandler route `GET /v1/ai_models`, handler and tests.
- bin-common-handler `requesthandler`: `AIV1AIModelList` RPC wrapper and mock regeneration.
- Response shape: a deliberate single-page variant of the `GET /ais` list convention (`CommonPagination` + `result: [AIManagerAIModel]`, no `PageSize`/`PageToken` declared; no existing path does this, so it is stated explicitly rather than claimed as convention); the catalog is a small bounded list, so it returns the whole list in one page and ignores `page_size`/`page_token` (no pagination parameters declared). Decided here so the OpenAPI schema is unambiguous.
- bin-api-manager: `servicehandler` method, `server/ai_models.go` handler (top-level `GET /ai_models`, any authenticated customer, read-only, like other top-level resources), tests.
- bin-openapi-manager: `paths/ai_models/main.yaml` registered in `openapi.yaml` `paths:` (next to `/ais`), schema `AIManagerAIModel`; regenerate `bin-openapi-manager/gens` and `bin-api-manager/gens`. `AIManagerAIEngineModel` enum is relaxed to a plain string whose description points to `GET /ai_models`. Verified: api-manager has no OpenAPI request-validator middleware (no `OapiRequestValidator` use) and no non-generated Go code references the `AIManagerAIEngineModelXxx` constants, so request behavior is unchanged; only generated constants and the JS type (`square-admin/src/types/api.ts` `AIEngineModel`, which is regenerated/edited in the same change) change. Other-language SDK consumers lose compile-time enum checking only; server validation stays authoritative.
- RST: API reference for `GET /ai_models`.
- Webhook and aicall payload fields keep carrying the customer ID; catalog IDs are never renamed. Retirement policy (only when a real retirement happens): keep the ID resolvable, mark deprecated with a `replaced_by`, no code now.

### 3.7 UI (monorepo-javascript/square-admin)

One `ModelPicker` component plus a small `useAIModels()` hook (single fetch of `GET /ai_models`, cached per session) replaces every `ENGINE_MODELS` use. Full consumer list (grep-verified by reviewer, all in scope):
- Model SELECTS: `AIEngineFields.js` (line ~149, ~360), `teamgraph/sidebar.js` (create form 128-168/418/509, edit form 648-651/790/946/1307), including `ais_create.js:45,178,194` and `ais_detail.js:80,180,202,411,533,557` state (`customModelInput`, `checkAndSetCustomInput`, dirty comparison) which is REMOVED for the model field only.
- Label lookups: `ais_list.js`, `InsightAIsPanel.js`, `teamgraph/nodes/member.js`, `aicalls_list.js:64`, `aicalls_detail.js:645`, `TestAgentSheet.js:88,95` use `modelLabel(id)` from the hook (raw ID fallback).
- `custom_input` for TTS/STT is untouched; `getFinalValue` stays for those.
- Types: `types/api.ts` `AIEngineModel` becomes `string`.
- Templates: `prompt_templates.js` uses `''` and `gemini.gemini-2.5-flash` only (both fine; `ais_create.js:154-156` fallback keeps working).
- Tests updated (including ais_create/ais_detail engine-key tests for the disabled state, dirty comparison unchanged): `AIEngineFields.test.js:292,312`, `ais_create.test.js:57,749`, `ais_detail.test.js:125`, `sidebar*.test.js`, `member.test.js`, `ais_list.test.js`, `InsightAIsPanel.test.js`, `aicalls_*test.js` plus new ModelPicker tests.
- `square-admin_new`: verify by grep whether it has any AI form; touch it only if it does.

Picker behavior:
- Trigger shows the selected label and vendor. Popover with search (label and vendor).
- Top section "Recommended" (entries with `recommended`), then collapsible vendor groups (Google, OpenAI, Anthropic, Meta, DeepSeek, Qwen, Mistral, xAI). Rows: label, one-line description, tags (`low-cost` in v1). No prices and no latency numbers (not measured).
- `recommended` is set only with evidence: v1 flags `gemini.gemini-2.5-flash` (already the default in all prompt templates) and any candidate that passes the measured first-token test at implementation; nothing else.
- No "Custom..." for the model. A stored value that is not in the catalog (legacy direct-prefix value such as `openai.gpt-4o`, or a never-working legacy value) is NOT classified client-side (round-3 finding: the client must not duplicate server prefix rules). The picker simply injects the stored value as a synthetic first row "Current: <id>" that is selected; no warning, no "unavailable" claim. On update the server validates only a CHANGED `engine_model` (3.1), so an unchanged legacy value saves normally; picking a catalog model replaces it. Replaces the `custom_input` injection done by `checkAndSetCustomInput` (`ais_detail.js:177-203`).
- `platform_managed` models: the Engine Key input stays MOUNTED but is disabled with helper text "API key not required" (in `ais_create.js:84,196,573-576` and `ais_detail.js:173,413,532,1293-1296` it is an uncontrolled ref input; in `teamgraph/sidebar.js` the create form (~975) is a controlled input sent as `trim() || undefined` and the edit form (~1359-1373) uses a masked display with the `engineKeyChanged` state; unmounting would throw on `ref.current.value` and make the dirty comparison always true, so it is never unmounted). Behavior in all four forms: when the selected model is `platform_managed`, the input is disabled and, on submit, the key is omitted/empty (sidebar edit: `engineKeyChanged` is treated as false and the existing key is not re-sent), so a key typed earlier for a direct model is never transmitted for a platform-managed one. Details (round-4 minor items): `PUT /ais/{id}` is a full replace and `engine_key` is required, and the server writes the received value as is (`aihandler/db.go:280`), so in `ais_detail.js` the save payload sends the STORED `engine_key` (`detailData.engine_key`) unchanged and only discards newly typed text; the dirty comparison (`ais_detail.js:532`) skips the engine-key term while a platform-managed model is selected; in the sidebar edit form the "Key will be updated on save" notice (`sidebar.js:1373`) is hidden for platform-managed models. Pre-existing and out of scope (recorded for a separate issue): the sidebar edit form omits `engine_key` when unchanged, which the full-replace server writes as empty. The API still requires the field (empty string allowed) and the server ignores any value.
- Catalog load failure or slow load: create form disables Save with an inline retry message; edit form keeps showing the raw stored value and never blocks other fields. A new UI against an old API (404) behaves as load failure, which the rollout order below avoids.

Illustrative layout:

```
[ Model: Claude Sonnet 4.5  (Anthropic)           v ]
  +---------------------------------------------+
  | Search models...                            |
  | RECOMMENDED                                 |
  |  Gemini 2.5 Flash       low-cost     |
  | ANTHROPIC   (2)                         v   |
  |  Claude Haiku 4.5                            |
  |  Claude Sonnet 4.5                          |
  | GOOGLE (4) / OPENAI (5) / META / DEEPSEEK   |
  | QWEN / MISTRAL / XAI                    >   |
  +---------------------------------------------+
  Engine key: API key not required (platform managed)
```

### 3.8 Docs and derived artifacts (analysis R17)

- RST: `ai_struct_ai.rst` provider table rewritten (remove `openrouter.<model>`, list customer prefixes and point to `GET /ai_models`); sweep `ai_overview.rst`, `ai_tutorial.rst`, `team_tutorial.rst`, `aicall_struct_aicall.rst`, `variable_variable.rst` for stale model examples; rebuild HTML if the repo commits `build/`.
- `monorepo-javascript/square-main/public/skill.md` (805-811 and examples at 411, 621, 909) and `public/llms.txt:49`: corrected to the real catalog, with `build/` copies regenerated by the normal build.
- No customer-facing text mentions OpenRouter. Privacy policy / DPA text is legal work outside this PR (tracked as a follow-up with CEO).

## 4. Initial catalog (candidates; final inclusion requires real verification, analysis R1/R13)

Selection rule: has a ZDR endpoint with `tools` support (D-F4), mainstream, sane cost (no premium tier above ~$5 input / $25 output per M tokens in the first release; Opus-class excluded, analysis R3).

| Customer ID | Upstream slug | Recommended (v1) |
|---|---|---|
| anthropic.claude-haiku-4.5 | anthropic/claude-haiku-4.5 | only if measured |
| anthropic.claude-sonnet-4.5 | anthropic/claude-sonnet-4.5 | no |
| meta.llama-3.3-70b-instruct | meta-llama/llama-3.3-70b-instruct | no |
| meta.llama-4-maverick | meta-llama/llama-4-maverick | no |
| deepseek.deepseek-v3.2 | deepseek/deepseek-v3.2 | no |
| qwen.qwen3-235b-a22b-2507 | qwen/qwen3-235b-a22b-2507 | no |
| qwen.qwen3-30b-a3b-instruct-2507 | qwen/qwen3-30b-a3b-instruct-2507 | no |
| mistral.mistral-medium-3.1 | mistralai/mistral-medium-3.1 | no |
| mistral.mistral-small-3.2-24b-instruct | mistralai/mistral-small-3.2-24b-instruct | no |

Rules: the `recommended` flag is set only after measured first-token latency on real calls (none assumed). A candidate that fails the real tool-call test is dropped from the catalog before merge. Reasoning/"thinking" models and variant suffixes are excluded in v1. Newer slugs seen in the ZDR list (e.g. Claude Sonnet 5.x) are NOT included until verified the same way (the catalog is a curated allow-list by design).

A verification script `bin-pipecat-manager/scripts/verify_openrouter_catalog.py` (manual, network) checks every catalog slug against the public ZDR list and `tools` support and exits non-zero on mismatch; it is run before releases and when OpenRouter retires a slug (R15 detection). It is not part of CI (network dependency).

## 5. Out of scope (with triggers)

- Per-customer attribution/quota on the shared key (R18): trigger = observed 429/402 impact or cost anomaly.
- Billing/metering, per-customer OpenRouter keys, privacy-policy/DPA wording.
- Moving openai/gemini/grok onto OpenRouter; OpenRouter outage fallback to direct providers (R21): trigger = measured outage impact.
- Dedicated Prometheus alert for credit exhaustion (trigger = first occurrence).

## 6. Test plan

Go (ai-manager): catalog invariants (unique IDs, every OpenRouter entry has a slug and no `:` suffix, no entry exposes slug in the API model), `ResolveEngine` table test (direct, openrouter, legacy `openrouter.*`, unknown), create/update validation (reject unknown, allow unchanged legacy on update), listenhandler/api tests for `GET /ai_models`.
Go (pipecat-manager): resolve at both hand-off points (type rewritten, key blanked for openrouter, unchanged for direct, a `Rejected` outcome makes `resolveSessionLLM` return an error and the Start RPC fail), `pipelineerror` classification for credit-exhaustion text incl. voice-session notify.
Python: `create_llm_service` openrouter branch (class, slug, `extra_body.provider` content, missing/empty key raises ValueError, customer key ignored); update the tests that assumed unsupported providers; routing/team tests with a mixed direct + OpenRouter team.
Frontend: ModelPicker unit tests (search, Recommended section, vendor grouping, synthetic "Current: <id>" row for non-catalog values, no custom entry, platform_managed key input disabled), updated existing tests; real-browser verification of the picker (screenshots) per the frontend visual gate.
Real verification (needs the platform OpenRouter key from CEO): for each catalog model, one real call with a tool call and verification that the request carried `provider.zdr` and succeeded; one team call mixing direct and OpenRouter members; one Listen turn; a missing-key negative test.
Added from round 3: `models/ai/main_test.go:341-344` (`anthropic.claude-3` currently valid) is updated for the new validation; update validation ordering test (unchanged legacy value passes, changed invalid value fails); Python guard tests for `platform_openrouter` via fallback and via the `:` separator; empty resolved session value errors.
Added from round 2: AI lookup failure cannot let a raw `openrouter.*` reach Python (single AI), a mutation that passes `pc.LLMType` unresolved at `runner.go:210` turns red, all three start sites use the shared helper (table test), update where stored and requested values are both empty (intent pinned), a team with a Rejected non-current member yields empty `llm_type` and a Python init error, the `run.py:770` fallback does not honor raw `openrouter.*`, a raw `openrouter.*` hits Unsupported in the runner, `SessionCreate`/session struct signature change keeps existing tests compiling (update list).
Added from round 1: `member_switched` `engine_model` is the customer ID for an OpenRouter member (slug must not appear); a `Rejected` model makes the Start RPC return an error (sync) rather than a silent session end; `DirectPassthrough` values (e.g. `openai.gpt-4o`) resolve unchanged with the customer key; `platform_managed` appears in the API model and route/slug never do.
Mutation checks on the new guards: drop key blanking, drop catalog membership check, drop `zdr` flag, each must turn a test red.

Implementation checks (not design gates): confirm whether an aicall created before a failed `startPipecatcall` is cleaned up (pre-existing behavior); confirm `SessionCreate` signature impact.

## 7. Rollout and risks

- One PR in monorepo (backend + docs), plus the required companion PRs in monorepo-javascript, monorepo-etc and install (one PR per repository, structural necessity).
- Operator checklist (no code): set a per-key credit limit and balance alert on the OpenRouter key before enabling (minimum safety net for free-account access with deferred quotas); self-host docs state that the models need the operator's own `OPENROUTER_API_KEY` (the UI line "API key not required" refers to the customer's per-AI key).
- Pre-merge: step 0 prod read-only check of stored `engine_model` values (3.1). No global on/off flag is added: the order below gates exposure instead.
- Service order (round-2 finding): pipecat-manager (Go resolver + Python runner, same release) -> ai-manager (catalog + validation) -> api-manager -> UI. Rollback of the backend: stored AIs that use catalog models fail closed at start (Rejected or unsupported), never spend platform money, and customers can re-pick a model.
- Order: secrets in monorepo-etc and install first (no behavior change), then backend, then UI. UI ships after the API; an old UI keeps working (it still sends existing IDs). Hosted: the OpenRouter secret must be live on the runners BEFORE the backend release, since the catalog exposes the models immediately. Self-host: documented that models need the operator `OPENROUTER_API_KEY`; with an unconfigured key the call fails with a provider authentication error (installer seeds a random key; no dummy special casing).
- Residual risk accepted: sub-provider behavior differences behind one slug (R15), voice latency unknown until measured (R13), shared key rate limits (R18).
