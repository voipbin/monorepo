# Issue Analysis: Route additional LLMs through OpenRouter (platform key)

Status: Issue-analysis review loop, round 5 pending (rounds 1-3 CHANGES_REQUESTED applied; round 4 APPROVED with minor citation fixes applied)
Branch: NOJIRA-Route-LLM-via-OpenRouter
Date: 2026-10-05

## 1. Request (CEO decisions, 2026-10-05)

- Today only Gemini, OpenAI (and Grok) are usable as AI engine models. Extend the
  selectable LLMs (Claude, Llama, DeepSeek, Qwen, Mistral, ...) by routing them
  through OpenRouter internally.
- Customers must NOT see "OpenRouter". They only see provider/model names such as
  Claude, Llama, Gemini, GPT.
- Key model A: the PLATFORM holds the OpenRouter key. Customers do not supply one.
- Add all reasonable models, but the UI must present them with deliberate UX, not a flat
  list.

## 2. Verified current state (live code, base 440b71f9d)

| # | Fact | Evidence |
|---|------|----------|
| F1 | Pipecat runner builds the LLM in `create_llm_service(type, key, ...)`; supports only `openai`, `grok`, `gemini`. Anything else raises `Unsupported LLM service`. | `bin-pipecat-manager/scripts/pipecat/run.py:570-640` |
| F2 | `type` is split on the FIRST `.` (or `:`), so `service.model` where model contains `/` or further dots parses correctly. | `run.py:575-578` |
| F3 | Key path: Go reads the DB `ai.EngineKey` and passes it as `llm_key` (`pipecatcallhandler/start.go:187-194`, `run.go:113-130`; team path `run.go:44,201`, where `ResolvedAI` strips the key in `models/team/resolved.go:12-22` and pipecat-manager re-attaches it). Python then does `key or os.getenv(<PROVIDER>_API_KEY)`. If the AI lookup fails, `start.go:191-193` proceeds with an empty key, which silently falls back to the PLATFORM env key. `engine_key` is `required` in the create API (`bin-openapi-manager/openapi/paths/ais/main.yaml:113`, `id.yaml:153`, RST `ai_struct_ai.rst:54`) but may be an empty string, and the UI has an Engine Key input (`ais_create.js:573`). Platform env keys: `komodo/docker-compose.yml` runner services carry OPENAI/XAI keys (XAI at lines 111,159,190,232; GOOGLE only on the script-runner services at 160/233). | `run.py:582,597,612`; compose above |
| F4 | pipecat-ai is pinned `>=1.12.0,<1.13` with the `openai` extra, uv.lock = 1.12.0. The 1.12.0 wheel ships `pipecat/services/openrouter/llm.py` (`OpenRouterLLMService`, subclass of `OpenAILLMService`, base_url `https://openrouter.ai/api/v1`, `supports_developer_role=False`). Verified by listing the downloaded 1.12.0 wheel. | wheel `pipecat_ai-1.12.0`, `uv.lock:1894` |
| F5 | Go enum lists anthropic, deepseek, mistral, qwen, openrouter, aws, azure, groq, ...; the effective gate is `IsValidEngineModel` (prefix-only check) called from `aihandler/chatbot.go:41,119` and `cmd/ai-control`. The OpenAPI enum has only 11 values and `bin-api-manager/server/ais.go:33,252` uses `BindJSON` on the generated type; whether the generated type enforces the enum was not proven, so treat the Go prefix check as the effective gate. `GetEngineModelTarget` is a fixed map with no non-test caller. | `bin-ai-manager/models/ai/main.go:107-221` |
| F6 | The `engine_model` value is passed unchanged as `LLMType` to pipecat in `startPipecatcall` and `startPipecatcallTask` (`aicallhandler/start.go`, grep `pmpipecatcall.LLMType(c.AIEngineModel)`, around 936/985). There is a THIRD entry point: `aicallhandler/listen.go:419-437` (`startListenPipecatcall`, Listen turn: text-in, tool-call-out, no STT/TTS) also passes `LLMType(c.AIEngineModel)`. LLM construction is `run.py:299` (single AI) and `run.py:770` (per team member); `team_flow.py:205` only builds member-switched metadata. | as listed |
| F7 | The text-chat engine (`engine_openai_handler.MessageSend/StreamingSend`) has NO non-test callers. Summary/analysis use the platform engine configuration (`cmd/ai-manager/main.go:163-190`), not per-AI `engine_model`. Reproduce: `cd bin-ai-manager && grep -rn "\.MessageSend(\|\.StreamingSend(" --include=*.go pkg cmd \| grep -v _test \| grep -v mock` (empty). So conversation LLM traffic is pipecat-only. | command above |
| F8 | `engine_model` is enumerated in FOUR hand-maintained places: Go constants, OpenAPI enum (`AIManagerAIEngineModel`) plus generated code, TS type `AIEngineModel` and `ENGINE_MODELS` in square-admin. The UI list is a flat array rendered in several places (`AIEngineFields.js` shared by create/detail, `ais_detail.js`, `ais_list.js`, `InsightAIsPanel.js`, `teamgraph/nodes/member.js`, `teamgraph/sidebar.js`). UI repos `square-admin` and `square-admin_new` both exist; target must be confirmed (design). | `openapi.yaml:2013-2040`; `square-admin/src/types/api.ts:465`; `views/ais/constants.js` |
| F9 | Customer-provided custom model strings are tolerated in UI (`custom_input` path). | `teamgraph/sidebar.js:648-654` |

Correction to an earlier statement in this thread: `engine_key` is not purely BYOK. It is
required-but-may-be-empty, and empty falls back to platform env keys (F3). Operating under
key model A needs a new platform secret `OPENROUTER_API_KEY` in the pipecat runner
services, but the key-selection logic must change for OpenRouter-routed providers
(see R2).

## 3. Issue validity

Valid. The official docs already advertise `anthropic.claude-3-5-sonnet` and `openrouter.meta-llama/llama-3-70b` (`bin-api-manager/docsdev/source/ai_struct_ai.rst:52,208`) while the runtime rejects them. A request for a non-listed provider (for example `anthropic.claude-...`) passes the Go prefix validation (F5), is stored, and fails only at call time in the runner with
`Unsupported LLM service` (F1). Nothing in the repo gives any non-openai/gemini/grok engine a working path.

## 4. Feasibility / proceed judgement

Proceed. Backend change is small and uses an upstream-supported class (F4).

Proposed scope (to be refined in the design doc):

1. pipecat runner: add an OpenRouter-backed branch. Customer-facing IDs stay
   `<provider>.<model>`; the runner maps them to OpenRouter model IDs
   (`<vendor>/<model>`), with a small alias table for vendors whose OpenRouter slug
   differs from our prefix. Existing `openai` / `gemini` / `grok` stay on their direct
   paths (no latency/cost change, no regression risk).
2. Key: `OPENROUTER_API_KEY` platform env (see R8 checklist). Customer `engine_key` is
   always ignored for OpenRouter-routed providers (R2, decided).
3. Single source of truth for the model catalog (label, vendor, tags such as
   recommended-for-voice / reasoning / tool-calling verified), to stop the four-way drift (F8).
   Options (design decision): static catalog in ai-manager exposed by API, vs.
   continuing hand-sync of four lists.
4. UI: replace the flat list with a grouped, searchable picker (vendor groups,
   "Recommended for voice" section, short capability/speed hints). Applies to all
   ENGINE_MODELS consumers.
5. Docs (RST/OpenAPI), including removal of the customer-visible `openrouter.<model>`
   row.

## 5. Risks and required handling (round 1 additions included)

- R1 tool calling: not every OpenRouter model reliably supports function calling (core to
  Flow). Curated list, each model verified with a real tool call. Also verify message
  shape constraints (system-first, no consecutive same role) against `filter_valid_messages`
  per vendor, and decide whether to pin OpenRouter provider routing
  (`require_parameters` / `allow_fallbacks`) so a sub-provider without tool support cannot answer.
- R2 customer key contamination (MUST): an AI keeps a stored `engine_key` (e.g. an OpenAI
  key) after its model is changed to `anthropic.*`; the `key or env` pattern would send
  that key to OpenRouter. Decision: for OpenRouter-routed providers the customer
  `engine_key` is ALWAYS ignored and only `OPENROUTER_API_KEY` is used. Also close the
  silent-fallback path (`start.go:191-193`) for these providers.
- R3 cost exposure (MUST include minimal guard): platform pays. Free-text model strings are accepted today (`custom_input` in `ais_create.js:46,177-196`, `sidebar.js:640-656`; prefix-only `IsValidEngineModel`), so a UI-only catalog bounds nothing. The allow-list MUST be enforced server-side in BOTH places: at AI create/update (ai-manager) and in the runner mapping (reject any model not in the catalog for OpenRouter-routed providers). Billing/metering stays
  out of scope, but at least: curated allow-list (no unrestricted high-cost models), a
  spend cap on the OpenRouter key, and a credit-low operator alert.
- R4 hosted vs self-host: no code gating found. Decide whether free/trial hosted accounts may use platform-paid OpenRouter models at all (D5). Self-hosters supply their own key. The install default is a dummy key (`secret_schema.py:125` pattern), so an unset OpenRouter key yields a 401 at call time, not at startup; define a startup/health check or an explicit pre-call error (R5).
- R5 missing/invalid key: `OpenAILLMService` raises on `api_key=None`; `main.py:139-144`
  maps `ValueError` to 400 and others to 500. Add an explicit pre-check that raises a
  clear error. A missing key is an init-time exception, not an RTVI error, so it never
  enters `pipelineerror` classification.
- R6 error classification gap (MUST): `pipelineerror.go` tokens target OpenAI/Grok/Gemini.
  OpenRouter 402 "insufficient credits" is not matched, becomes `unknown`, and
  `shouldNotifyPipelineError` returns false for unknown in voice sessions
  (`pipelineerror.go:96-114`), so credit exhaustion would be a silent call in VOICE sessions. Text-only sessions (`hasSTT=false`, which includes Listen turns, F6) do notify on unknown, so the gap is voice-specific. Decide whether
  to add a category/tokens for OpenRouter credit and auth errors.
- R7 team, routing and Listen: the Listen turn path (`listen.go:427`) must be verified with an OpenRouter model for tool-call output. Team and routing: `run.py:770` builds each member LLM; mixed direct + OpenRouter teams
  must work. `routing_llm.py:25,57-62` overrides `push_frame` and relies on
  `broadcast_service_metadata`; `run.py:777` sets `_has_async_tools` (private API).
  All need a real test with `OpenRouterLLMService`. Existing tests that assume
  unsupported providers (`test_run.py`, `test_init_pipeline.py:111-128,184-201`,
  `test_routing_llm.py`, `test_team_flow.py`) must be updated.
- R8 secret plumbing parity (checklist): monorepo `bin-pipecat-manager/komodo/docker-compose.yml`
  (runner services; manager services to be decided), `monorepo-etc/infra-secret/secrets-source/bin-manager/secrets.enc.yml`
  (SOPS source of `BIN_MANAGER__*`), install repo `scripts/secret_schema.py` (dummy default
  pattern, key-count sanity), `k8s/backend/secret.yaml`, `k8s/backend/services/pipecat-manager.yaml`,
  self-hosting docs `self_hosting_providers.rst`, `bin-pipecat-manager/docs/operations.md`. Follow skill
  voipbin-k8s-secret-management.
- R8b (minor): `bin-ai-manager/k8s/deployment.yml` injects OPENAI/GOOGLE keys for ai-manager's own summary/analysis use; design confirms which services read `OPENROUTER_API_KEY` (pipecat runners only expected). Pre-existing, out of scope: empty `engine_key` on direct `openai.*`/`grok.*` already falls back to platform keys (cost exposure shared with R3/D5).
- R9 exposure of the mapped vendor ID: customer-facing `engine_model` already appears in `member_switched.go:39` and `resolved.go:16` (fine, it is the customer ID). The internal `<vendor>/<model>` must not leak via logs,
  metrics labels (`pipecatcallhandler/metrics.go`) or `member_switched` events; those keep
  the customer-facing `engine_model`.
- R10 existing data and validation policy: decide what happens to already-stored values
  `openrouter.*`, `anthropic.*` etc. and to unsupported prefixes (`aws`, `azure`, `groq`...)
  that today pass `IsValidEngineModel`. Proposal: replace prefix-only check with a catalog
  allow-list at create/update; keep reading legacy values.
- R11 library notes: `OpenRouterLLMService` prefers `settings=` over `model=`. The Gemini
  system-message handling inside it only applies to model names containing "gemini"; since
  `gemini.*` stays on the direct path it is not exercised here.
- R12 UI/test blast radius: grouped picker touches the six ENGINE_MODELS consumers above and
  existing tests (e.g. `ais_detail.test.js`). A catalog API changes OpenAPI and generated
  clients (`bin-api-manager/gens`, `bin-openapi-manager/gens`).
- R14 streaming and interruption: barge-in cancellation and idle timeout behavior must be verified per vendor; the Gemini path sets `stream_idle_timeout_secs=None` (`run.py:613`) and the OpenRouter path needs an explicit per-vendor decision.
- R15 model ID mapping lifecycle: the stored `engine_model` is a permanent identifier, but OpenRouter slugs get renamed, retired and versioned, and carry variant suffixes (`:free`, `:thinking`, `:online`, which must never be reachable). Needs: an explicit customer-ID to slug alias table kept in ONE place; a deprecation policy (alias redirect to a successor model rather than breaking saved AIs); a rule on sub-provider variance (same slug, different behavior). `engine_model` is also emitted in customer webhooks (`aicall/webhook.go:20`, `ai/webhook.go:27`), so changing a stored/emitted value is an external contract change.
- R16 customer-visible prefixes: the Go enum has no prefix for Llama/Meta (nor `google`, which `square-main/public/skill.md:809` already documents). Adding Llama and similar requires deciding customer-facing prefixes and extending `EngineModelTargets`/validation (`models/ai/main.go:107-171`).
- R17 documentation and agent-facing derivatives that list models and must be refreshed with the catalog: `monorepo-javascript/square-main/public/skill.md:805-811` (already inconsistent: `google.*`, `openai.gpt-4o`, `anthropic.claude-3-5-sonnet`), also `skill.md:411,621,909`, `public/llms.txt:49` and the `build/` copies; RST `ai_overview.rst`, `ai_tutorial.rst`, `team_tutorial.rst`, `aicall_struct_aicall.rst`, `variable_variable.rst` under `bin-api-manager/docsdev/source`; UI display-only consumers `aicalls_list.js:64`, `aicalls_detail.js:645`, `TestAgentSheet.js:88,95` (decide label vs raw value). Other-language SDK clients may reject enum values not in their generated type (F5 unproven).
- R18 shared platform key: one OpenRouter key means shared rate limit and credit across all customers (noisy neighbor: one customer's volume yields 429/402 for others). Needs per-customer attribution (OpenRouter `user` field or header) and a decision on per-customer quota, beyond the R3 total cap.
- R19 data privacy and compliance (needs CEO/legal decision, D8): call transcripts (PII) go to OpenRouter and its sub-providers. Evaluate OpenRouter data-collection / zero-data-retention routing options, update the privacy policy and DPA sub-processor list. "Hide OpenRouter from the UI" must not conflict with sub-processor disclosure obligations.
- R20 customer notice wording and error text exposure: the fixed notice for authentication errors reads "If this AI uses a custom engine key, verify that the key is valid..." (`messagehandler/pipeline_error.go:35`). It is conditional and does not blame a key, but for platform-key OpenRouter failures it is not actionable. Decide on wording for platform-side failures. Separately, raw provider error text (which may contain the mapped slug or `openrouter.ai` URLs) is logged up to 2000 chars (`pipelineerror.go:12`, `runner.go:723`) and `bin-ai-manager/pkg/aicallhandler/send.go:170` logs the engine model; customer-facing surfaces never carry raw error text (`PipelineErrorEvent`), so the exposure is operator logs only (acceptable, but state it).
- R21 availability: single router outage disables all OpenRouter-routed models. Out of scope; direct openai/gemini/grok paths are unaffected. Fallback to direct provider is a future option only on a measured need.
- R13 voice latency: extra proxy hop; measure first-token latency on real calls before
  labeling a model "recommended for voice". No numbers assumed.

## 6. Out of scope

Billing/metering, per-customer OpenRouter keys, exposing "openrouter" as a customer
provider, moving existing openai/gemini/grok onto OpenRouter.

## 7. Decision requests for design stage

- D1 Catalog single-source via API vs. static lists (R12).
- D2 DECIDED in analysis: customer `engine_key` ignored for OpenRouter-routed providers (R2).
- D3 Initial model list (verified-with-tool-call only, cost-aware per R3).
- D4 Validation policy and legacy data (R10).
- D5 Hosted gating and cost guards (R3, R4).
- D6 Error classification extension (R6).
- D8 Privacy/sub-processor policy and DPA wording (R19).
- D9 Alias and deprecation policy for model IDs (R15), customer prefixes for Llama etc. (R16).
- D7 Which UI repo(s): square-admin vs square-admin_new (R12).
