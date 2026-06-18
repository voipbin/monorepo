# Upgrade pipecat-ai 0.0.103 -> 1.4.0 (bin-pipecat-manager Python runner)

**Date:** 2026-06-18
**Service:** bin-pipecat-manager (Python runner under `scripts/pipecat/`)
**Type:** Dependency upgrade + breaking-API migration (hardening class, minimal-change bias)
**Tech Stack:** Python 3.12, pipecat-ai, pipecat-ai-flows, uv

## 1. Problem Statement

The Python Pipecat runner is pinned at `pipecat-ai==0.0.103` (lock), last touched
2026-03-11. Pipecat has since shipped a major release (1.0.0, 2026-04-14) plus
1.1.0 / 1.2.0 / 1.3.0 / 1.4.0 (2026-06-16). The current pin is ~4 months and one
major version behind, which is a security (dependabot) and feature-currency
liability:

- The legacy `OpenAILLMContext` + `create_context_aggregator()` API is removed in
  1.x (deprecated since 0.0.99). The runner still uses it on the openai/grok path.
- Turn-detection, function-calling auto-registration, and STT/TTS service
  improvements accrued across 1.0-1.4 are unavailable.

This is a controlled upgrade, not a feature change. Behavior must be preserved
byte-for-byte where possible (audio path, barge-in, team routing, smart turn).

## 2. Scope

**In scope (this PR):**
- Bump `pipecat-ai` 0.0.103 -> 1.4.0 and `pipecat-ai-flows` 0.0.23 -> 1.2.0 in
  `pyproject.toml` / `requirements.txt`, regenerate `uv.lock`.
- Fix the 7 breaking points enumerated in section 3 (all verified empirically
  against a clean 1.4.0 install, not inferred from changelog).
- Update the Python pytest suite that asserts on the changed APIs.

**Out of scope (deferred, not required for a working upgrade):**
- Adopting new 1.x turn-detection strategies (`user_turn_strategies`,
  `LLMTurnCompletionUserTurnStopStrategy`). Behavior-preserving upgrade first;
  turn-detection tuning is a separate, measurable change.
- Migrating `PipelineTask` -> `PipelineWorker` naming. `PipelineTask` remains a
  working alias in 1.4.0 (deprecation warning only). Renaming is cosmetic churn
  and is deferred.
- Realtime / speech-to-speech services, multi-agent framework, `pipecat eval`.
- Deepgram default model change (1.x default is nova-3; runner hardcodes
  `model="nova-2"`, so no behavior change. Re-evaluating nova-3 is a separate
  decision.)

## 3. Breaking Surface (empirically verified on a clean pipecat-ai 1.4.0 install)

All 7 points were confirmed by installing
`pipecat-ai[silero,deepgram,openai,cartesia,websocket,google,local-smart-turn]==1.4.0`
plus `pipecat-ai-flows==1.2.0` into an isolated venv and importing/inspecting the
actual runner modules. Survivor APIs (custom transport overrides, FrameProcessor
routing, RTVI auto-setup, flows FlowManager/FlowArgs/FlowsFunctionSchema/NodeConfig)
were confirmed present and unchanged; they need no edits.

| # | File / Location | Current (0.0.103) | 1.4.0 reality | Fix |
|---|---|---|---|---|
| 1 | `run.py:17` | `from deepgram import LiveOptions` | Deepgram SDK 7.x removed top-level `LiveOptions`; pipecat ships a compat wrapper at `pipecat.services.deepgram.stt.LiveOptions` (DeprecationWarning only) | Change import to `from pipecat.services.deepgram.stt import LiveOptions` |
| 2 | `run.py:28`, `run.py:450-451,463-464` | `from ...openai_llm_context import OpenAILLMContext`; openai/grok paths build `OpenAILLMContext(...)` + `llm.create_context_aggregator(ctx)` | Module `pipecat.processors.aggregators.openai_llm_context` removed; `OpenAILLMService.create_context_aggregator` removed | Replace with universal `LLMContext` + `LLMContextAggregatorPair` (already used on the Gemini path and for team pipelines), converting tools via `ToolsSchema`/`FunctionSchema` the same way |
| 3 | `run.py:18` | `from pipecat.services.whisper.stt import Model, WhisperSTTService` | Module exists but requires the `whisper` extra; **dead import (never referenced)** | Delete the line |
| 4 | `run.py:21` | `from ...stt_mute_filter import STTMuteConfig, STTMuteFilter, STTMuteStrategy` | Module path removed; **dead import (never referenced)** | Delete the line |
| 5 | `requirements.txt:3`, `pyproject.toml:9` | `pipecat-ai[...,local-smart-turn-v3]` | Extra renamed to `local-smart-turn`; the V3 analyzer (`LocalSmartTurnAnalyzerV3`) ships in-package and imports fine | Rename extra `local-smart-turn-v3` -> `local-smart-turn` |
| 6 | `requirements.txt:4`, `pyproject.toml:10`, `uv.lock` | `pipecat-ai-flows>=0.0.10` (locked 0.0.23) | flows 1.2.0 is the pipecat-1.x-compatible line; FlowManager/FlowArgs/FlowsFunctionSchema/NodeConfig API all present | Bump pin so the lock resolves flows 1.2.0 |
| 7 | `routing_llm.py:62-69` | `register_function(self, function_name=None, handler=None, start_callback=None, *, cancel_on_interruption=True, **kwargs)` forwarding `function_name=`, `start_callback=` | 1.4.0 `LLMService.register_function(name, handler, *, cancel_on_interruption=None, timeout_secs=None)` removed `start_callback`. flows 1.2.0 calls `self._llm.register_function(name, transition_func, cancel_on_interruption=, timeout_secs=)` (positional name+handler). **Load-bearing: without this, every team flow dies at `flow_manager.initialize()` with `TypeError: unexpected keyword 'start_callback'`** | Update the forward signature to positional `(name, handler, *, cancel_on_interruption=True, timeout_secs=None, **kwargs)` and forward `timeout_secs`; drop `start_callback` |

### 3a. Survivors that required the SAME empirical rigor (added after review round 1)

Review round 1 (two independent reviewers) correctly flagged that the original
surface stopped at 7 symbols and did not apply the same import+inspect rigor to
three additional load-bearing surfaces. All three were subsequently verified
against the clean 1.4.0 install and need **no code change**, but are recorded here
because the mocked test suite cannot prove their survival:

- **`tools.py` function-calling API (VERIFIED SURVIVOR).** `FunctionCallParams`
  (dataclass, fields incl. `function_name`, `tool_call_id`, `arguments`, `llm`,
  `result_callback`, `app_resources`), `FunctionCallResultProperties(run_llm=,
  on_context_updated=, is_final=)`, and `params.result_callback(result,
  properties=...)` all present and signature-compatible with `tool_execute`.
  This is the single-AI tool-execution path (every connect/message/set_variables
  tool in prod). `conftest.py` mocks the whole `tools` module, so the unit suite
  cannot catch a break here — an **un-mocked import smoke test is added** (see §6).
- **`RoutingLLMService.push_frame` monkeypatch (VERIFIED SURVIVOR).**
  `routing_llm.py:22-23` reassigns `svc.push_frame` per-instance and relies on
  pipecat's `LLMService` emitting downstream output via `self.push_frame`.
  Confirmed 1.4.0 `llm_service.py` still calls `self.push_frame` (2 call sites).
  Team output routing intact. Covered by mandatory live team smoke test (§6a).
- **flows 1.2.0 adapter selection (VERIFIED — and a stale comment to fix).**
  `create_adapter` is **removed** in flows 1.2.0 (0 occurrences); `__init__`
  unconditionally sets `self._adapter = LLMAdapter()` (universal). The
  `flow_manager._llm = routing_llm` swap still works (register_function is
  duck-typed/fanned out, verified `self._llm = llm` still stored), but the
  comment at `run.py:671-675` claiming "FlowManager's create_adapter checks the
  LLM class type ... pass the start member's actual LLM service so the adapter is
  created correctly" is now **false** and must be rewritten. flows 1.2.0 also
  natively accepts `llm: LLMService | LLMSwitcher` (4 `LLMSwitcher` refs) — the
  supported multi-LLM seam now exists; migrating team routing onto `LLMSwitcher`
  is filed as a fast-follow (§9), not done here.

### 3b. Tool-schema fidelity on the openai/grok migration (LATENT risk, NOT an active regression)

The legacy `OpenAILLMContext(tools=openai_tools)` passed native OpenAI tool dicts
straight through. The converged universal path runs them through
`_openai_tools_to_standard()` which builds `FunctionSchema(name, description,
properties, required)` — preserving everything inside `properties` (verified:
**nested** `additionalProperties`, `enum`, `pattern`, nested types all survive the
round-trip) but dropping any **top-level** `parameters.additionalProperties` and
`function.strict`.

Audited the actual ai-manager tool catalog (`bin-ai-manager/pkg/toolhandler/
definitions.go`): the only `additionalProperties` uses are **nested** inside
`properties.variables` (set_variables, create_call) — both preserved. **No tool
sets top-level `strict` or top-level `additionalProperties` today.** So this is a
latent fidelity gap, not a current behavior change. Mitigation: a fidelity unit
test feeding a tool carrying top-level `strict`+`additionalProperties` asserts the
known drop (making it explicit/reviewed), and a code comment in
`_openai_tools_to_standard` documents the four-field constraint so a future
upstream addition doesn't silently vanish.

## 4. Fix Detail (before / after)

### 4.1 run.py imports (points 1, 3, 4)

```python
# BEFORE
from pipecat.services.deepgram.stt import DeepgramSTTService
from deepgram import LiveOptions
from pipecat.services.whisper.stt import Model, WhisperSTTService
from pipecat.services.google.stt import GoogleSTTService
from pipecat.transcriptions.language import Language
from pipecat.processors.filters.stt_mute_filter import STTMuteConfig, STTMuteFilter, STTMuteStrategy

# AFTER
from pipecat.services.deepgram.stt import DeepgramSTTService, LiveOptions
from pipecat.services.google.stt import GoogleSTTService
from pipecat.transcriptions.language import Language
```

`OpenAILLMContext` import (point 2) is also removed; `LLMContext`,
`LLMContextAggregatorPair`, `FunctionSchema`, `ToolsSchema`, `NOT_GIVEN` are
already imported.

### 4.2 create_llm_service openai/grok paths (point 2)

The openai and grok branches currently build `OpenAILLMContext(messages, tools)`
and `llm.create_context_aggregator(ctx)`. The Gemini branch already shows the
target pattern (universal `LLMContext` + `LLMContextAggregatorPair` with tools
converted through `ToolsSchema(standard_tools=...)`). Converge openai/grok onto
the same helper:

```python
# AFTER (shared by openai and grok branches)
standard_tools = _openai_tools_to_standard(tools)
tools_schema = ToolsSchema(standard_tools=standard_tools) if standard_tools else NOT_GIVEN
ctx = LLMContext(messages=valid_messages, tools=tools_schema)
aggregator = LLMContextAggregatorPair(ctx)
return llm, aggregator
```

This is the safest convergence because the team pipeline path already runs the
universal context with these exact provider services in production; openai/grok
were the only branches still on the legacy context.

### 4.3 routing_llm.register_function forward (point 7)

```python
# BEFORE
def register_function(self, function_name=None, handler=None, start_callback=None, *, cancel_on_interruption=True, **kwargs):
    for svc in self._services.values():
        svc.register_function(
            function_name=function_name,
            handler=handler,
            start_callback=start_callback,
            cancel_on_interruption=cancel_on_interruption,
        )

# AFTER
def register_function(self, name=None, handler=None, *, cancel_on_interruption=None, timeout_secs=None, **kwargs):
    if kwargs:
        raise TypeError(f"RoutingLLMService.register_function got unexpected kwargs: {list(kwargs)}")
    for svc in self._services.values():
        svc.register_function(
            name,
            handler,
            cancel_on_interruption=cancel_on_interruption,
            timeout_secs=timeout_secs,
        )
```

`cancel_on_interruption` default is `None` to mirror 1.4.0
`LLMService.register_function` exactly (round-2 finding: do not invent `True`
where upstream is `None`; every real caller — flows 1.2.0 — passes it explicitly,
so the default is effectively moot, but it must not diverge from upstream). `**kwargs`
is kept only to **reject** unknown args loudly (e.g. a re-introduced
`start_callback`) rather than silently swallow them — this is what makes the
Tier-1 regression test meaningful. `tools.py:81` registers on the **raw**
single-AI service positionally (`llm_service.register_function(tool_name,
wrapper)`) and is never routed (single-AI has no router); a code comment notes this.

### 4.4 Fix the now-false FlowManager adapter comment + add a loud guard (point 3a)

`run.py:671-675`'s comment about `create_adapter` selecting the provider adapter
is false in flows 1.2.0 (adapter is universal `LLMAdapter()`). Rewrite it to state
the real reason for the swap (register_function fan-out to all member services;
adapter is universal regardless of the passed llm). Add an explicit guard so a
future flows rename of `_llm` fails loudly instead of silently no-op'ing the
fan-out (a silent no-op = only the start member ever gets tool registrations, so
transitions appear to work but other members' tools are missing):

```python
# AFTER (before the swap)
if not hasattr(flow_manager, "_llm"):
    raise RuntimeError("FlowManager no longer exposes _llm; team tool-routing fan-out broken")
flow_manager._llm = routing_llm
```

### 4.5 Dependency pins (points 5, 6) + reconciliation

Round 1 found the dep files are **inconsistent today** and §3.5 is an *add* in
pyproject, not a rename:
- `requirements.txt:3` has `local-smart-turn-v3`; `pyproject.toml:9` has **no
  smart-turn extra at all** (yet the code imports `LocalSmartTurnAnalyzerV3`).
- `requirements.txt:5` hard-pins `onnxruntime==1.20.1`; `pyproject.toml:11` has
  `onnxruntime>=1.20.1`.

```
# requirements.txt
pipecat-ai[silero,deepgram,openai,cartesia,websocket,google,local-smart-turn]>=1.4.0
pipecat-ai-flows>=1.2.0
onnxruntime>=1.24.3   # pipecat-ai 1.4.0 silero extra requires >=1.24.3,<1.25

# pyproject.toml
requires-python = ">=3.11"   # pipecat-ai 1.4.0 requires >=3.11 (Docker base is python:3.12-slim)
"pipecat-ai[silero,deepgram,openai,cartesia,websocket,google,local-smart-turn]>=1.4.0",  # ADD local-smart-turn
"pipecat-ai-flows>=1.2.0",
"onnxruntime>=1.24.3",
```

**Implementation note (resolved during `uv lock`):** the original plan kept the
exact `onnxruntime==1.20.1` pin, but `uv lock` surfaced a HARD conflict (exactly
as §4.5 anticipated): pipecat-ai 1.4.0's `silero` extra requires
`onnxruntime>=1.24.3,<1.25`. Per the design's stated reconciliation rule (resolve
the conflict explicitly by bumping to the version 1.4.0 needs, not by silently
loosening to a floor that picks something arbitrary), the pin was raised to
`>=1.24.3`; `uv.lock` resolves it to 1.24.4 (the same build the clean 1.4.0 venv
pulled). A second `uv lock` error surfaced that pipecat-ai 1.4.0 requires Python
`>=3.11` while `pyproject.toml` declared `requires-python = ">=3.10"`; bumped to
`>=3.11` (the runner Docker base is `python:3.12-slim`, so no runtime impact).
Both files carry the identical pipecat extras set and onnxruntime floor.

## 5. Survivors (no change required, verified)

- Custom transport: `UnpacedWebsocketClientOutputTransport` overrides
  `_write_audio_sleep()`, `process_frame()`, calls `_write_frame()`. All three
  exist on 1.4.0 `WebsocketClientOutputTransport`. `WebsocketClientTransport.output()`
  still constructs `WebsocketClientOutputTransport(self, self._session, self._params)`,
  matching the `UnpacedWebsocketClientTransport.output()` override exactly. Barge-in
  FLUSH_MEDIA path intact.
- `audio_out_sample_rate=16000` in `PipelineParams` preserved (mandatory 16 kHz rule).
- Routing services (`RoutingLLMService/TTS/STT`) subclass `FrameProcessor` — unchanged.
- `PipelineTask`, `PipelineRunner`, `PipelineParams`, `ProtobufFrameSerializer`,
  `SileroVADAnalyzer`, `LocalSmartTurnAnalyzerV3`, `LLMRunFrame`, RTVI auto-setup,
  `FlowManager(task=, llm=, context_aggregator=)` + `flow_manager._llm` reassignment
  (flows 1.2.0 still stores `self._llm = llm`) — all confirmed present.

## 6. Test Strategy

**Critical limitation of the existing suite (round-1 finding).** `conftest.py`
replaces the entire `pipecat.*` tree plus `tools`, `routing_*`, `team_flow`,
`pipecat_flows`, `deepgram`, `aiohttp` with MagicMocks before import. So a green
pytest run proves **wiring** (the runner calls the right names with the right
args), NOT **behavior** (what those names now do). The mocked suite is
structurally incapable of catching a 1.4.0 API break. The gate therefore has
three tiers, and tiers 2 and 3 are mandatory, not optional.

### 6.0 Tier 1 — update the mocked unit suite (wiring regression)

- `conftest.py` mocks `pipecat.processors.aggregators.openai_llm_context` and the
  `stt_mute_filter` module. Remove the openai_llm_context mock (module gone) and
  the stt_mute_filter mock (import deleted).
- `test_run.py` has ~11 `@patch("run.OpenAILLMContext")` decorators (lines incl.
  12, 31, 54, 78, 97, 117, 136, 155, 174, 206, 227) that will raise
  `AttributeError` at patch time once the import is removed. Rewrite the
  openai/grok/gemini tests to patch `run.LLMContext` / `run.LLMContextAggregatorPair`
  and assert the `ToolsSchema`/`FunctionSchema` conversion. **Baseline caveat
  (round-2):** the existing Gemini tests assert `create_context_aggregator` /
  `OpenAILLMContext`, which the *current* (0.0.103) Gemini branch already no longer
  calls — so the suite is likely **not fully green on 0.0.103 today**. Step one is
  therefore: run the suite on 0.0.103, record exactly which tests are already red
  and why (stale Gemini assertions), fix/justify those as baseline cleanup, and
  only then treat "full pytest green" as a meaningful before/after gate. Do not
  claim a green baseline the same plan shows is red.
- `test_routing_llm.py` mocks `svc.register_function = MagicMock()`, which
  swallows any signature including the removed `start_callback`. A plain
  `Mock(spec=...)` restricts attribute access but does **not** validate call
  kwargs — use **`create_autospec`/`autospec=True`** bound to the real 1.4.0
  `LLMService.register_function` signature so a regressed forward of
  `start_callback` raises. Add an explicit test asserting
  `RoutingLLMService.register_function(..., start_callback=...)` raises `TypeError`
  (the wrapper now rejects unknown kwargs, §4.3). Assert `timeout_secs` is
  forwarded.
- Add a **tool-schema fidelity test** (point 3b): feed a tool carrying top-level
  `strict` + top-level `additionalProperties` through `_openai_tools_to_standard`
  and assert the documented drop, so the lossy re-encode is explicit and reviewed.
- The AST guard tests (RTVI auto-setup, UnpacedWebsocketClient* presence) are
  unaffected and must still pass unchanged.

### 6a. Tier 2 — un-mocked import + offline behavioral checks (API-compat gate)

Run in the clean 1.4.0 venv with **no conftest on the path**:
1. `python -c "import run, tools, team_flow, routing_llm, routing_tts, routing_stt"`
   succeeds with no ImportError (only DeprecationWarnings acceptable). This is the
   real import-compat gate; the mocked pytest run is not.
2. Offline (no API key) behavioral probes that the author already ran and that
   become committed checks: (a) `_openai_tools_to_standard` -> `ToolsSchema` ->
   `OpenAILLMAdapter` round-trip preserves nested `additionalProperties`; (b)
   `DeepgramSTTService(live_options=LiveOptions(model="nova-2", interim_results=
   True))` resolves `punctuate=True, profanity_filter=True, smart_format=False`
   (parity with the old inline defaults), `model` overrides the new nova-3 default.
3. `nova-2` still hardcoded; no top-level `from deepgram import` remains.

### 6b. Tier 3 — MANDATORY live smoke matrix before merge (real calls vs Asterisk)

The mocked suite + a single post-deploy api-validator AICall is insufficient
(round-1, both reviewers). None of the unit tests exercise 16 kHz audio, the
`_write_audio_sleep` no-op pacing, `_write_frame(TextFrame("FLUSH_MEDIA"))`
barge-in, smart-turn, or team routing against real 1.4.0. Minimum 6 calls,
results reported in the PR before merge:

| # | Pipeline | Provider | Exercises | Pass criteria |
|---|---|---|---|---|
| A | single-AI | openai | call with >=1 tool (required args) + system prompt; trigger tool by voice | tool fires; Go `/tools` gets correct args; system prompt obeyed; 16 kHz (not robotic) |
| B | single-AI | grok (x.ai) | same as A | tool fires on x.ai endpoint; system prompt obeyed |
| C | single-AI | openai | barge-in: interrupt mid-TTS | TTS stops immediately; FLUSH_MEDIA sent; new turn aggregated cleanly |
| D | single-AI | openai | rapid back-to-back utterances | no double-fire / dropped turn (new universal aggregator timing) |
| E | team | multi-member | transition between >=2 members, each with its own tool | transition works (point 7); post-transition member's tools actually fire (fan-out) |
| F | team | multi-member | start with non-empty handed-off conversation history | first member is history-aware (flows 1.2.0 context strategy) |
| G | single-AI | gemini | call with >=1 tool + system prompt | tool fires; system prompt obeyed (gemini also moves to 1.4.0 — its aggregator/context is not exempt from the bump) |

Rollback is `git revert` of the pin + `uv.lock` commit (single logical change).

## 7. Affected Files

| File | Change |
|---|---|
| `bin-pipecat-manager/scripts/pipecat/run.py` | imports (1,2,3,4), openai/grok context migration (2), rewrite false FlowManager adapter comment + add `_llm` guard (3a/4.4) |
| `bin-pipecat-manager/scripts/pipecat/routing_llm.py` | register_function forward signature (7) |
| `bin-pipecat-manager/scripts/pipecat/tools.py` | doc comment: four-field FunctionSchema constraint in `_openai_tools_to_standard` is in run.py; note single-AI register bypasses router |
| `bin-pipecat-manager/scripts/pipecat/requirements.txt` | pins (5,6), relax onnxruntime `==`->`>=` |
| `bin-pipecat-manager/scripts/pipecat/pyproject.toml` | pins (5,6) — **ADD** local-smart-turn extra (absent today) |
| `bin-pipecat-manager/scripts/pipecat/uv.lock` | regenerate |
| `bin-pipecat-manager/scripts/pipecat/conftest.py` | drop removed-module mocks |
| `bin-pipecat-manager/scripts/pipecat/test_run.py` | repoint `@patch` to LLMContext path; fidelity test |
| `bin-pipecat-manager/scripts/pipecat/test_routing_llm.py` | spec-bound register_function mock; assert timeout_secs forwarded |

No Go changes. Go side (transport, session lifecycle, DB) is unaffected by the
Python library upgrade.

## 8. Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Deepgram 7.x compat-wrapper `LiveOptions` later removed (deprecated, slated 2.0.0) | Low | We pin 1.4.0; the wrapper is present and functional. Fast-follow to migrate to `settings=Settings(...)` (§9). |
| Subtle runtime behavior change in universal context / new aggregator timing vs legacy on openai/grok | Medium | gemini + team already run the universal context in prod; openai/grok converge onto that proven path. Caught by Tier 3 live smoke rows A-D (barge-in, rapid turns) before merge, not after deploy. |
| Tool-schema fidelity drop (top-level strict/additionalProperties) | Low | No current tool uses them (audited); fidelity unit test + code comment make the constraint explicit (§3b). |
| `flow_manager._llm` private-attr reassignment breaks if flows internals change | Low | Verified flows 1.2.0 stores `self._llm = llm`; added a loud `hasattr` guard (§4.4) so a future rename fails fast instead of silently dropping tool fan-out. |
| `register_function` signature break (start_callback removed) silently breaks team flows | Medium | Load-bearing fix (§3.7/§4.3); wrapper rejects unknown kwargs and `cancel_on_interruption` default mirrors upstream `None`. Gated by Tier-1 autospec regression test + Tier-3 rows E-F. |
| Mocked unit suite cannot catch a 1.4.0 API break | Medium | Tier 2 un-mocked import gate + Tier 3 live smoke matrix are mandatory before merge (§6a/§6b); the mocked pytest run is treated as wiring-only. |

## 9. Open Questions

| Question | Recommendation | Owner |
|---|---|---|
| Adopt 1.x turn-detection strategies now or defer? | Defer to a separate measurable change; this PR is behavior-preserving | CPO -> CEO/CTO |
| Move to Deepgram nova-3 default? | Separate decision; keep nova-2 here to isolate the upgrade | CEO/CTO |
| Rename PipelineTask -> PipelineWorker? | Defer; alias works, renaming is cosmetic churn | CPO |
| Migrate team routing onto native `LLMSwitcher` (flows 1.2.0 seam) instead of the `_llm` private-attr swap? | Fast-follow PR; keeps this PR minimal, removes the private-attr fragility before the next upgrade | CPO |
| Migrate Deepgram to `settings=Settings(...)` and services to `settings=Settings(model=...)` off the deprecated `LiveOptions`/`model=` kwargs before 2.0.0? | Fast-follow; functional now, deprecation clock running | CPO |

## 10. Review Summary (v1 -> v2)

Two independent adversarial reviewers (fresh `delegate_task`, no chat/worktree
access, full doc inlined) returned **CHANGES REQUESTED**. Both ran their own
empirical probes against a real 1.4.0 / flows 1.2.0 / deepgram 7.3.1 install. The
author independently re-verified every code-level claim before accepting it (did
not trust self-reports). Changes applied in v2:

- **Added §3a** — three load-bearing surfaces the original 7-point list missed,
  now empirically verified as survivors: `tools.py` function-calling API,
  `RoutingLLMService.push_frame` monkeypatch (1.4.0 still emits via
  `self.push_frame`), flows 1.2.0 adapter (`create_adapter` removed -> universal
  `LLMAdapter()`; the run.py comment is now false and will be rewritten + guarded).
- **Added §3b** — openai/grok tool-schema fidelity: top-level `strict` /
  `additionalProperties` are dropped by `_openai_tools_to_standard`. Audited the
  ai-manager catalog: all `additionalProperties` are **nested** (preserved), no
  tool sets top-level `strict`. Latent gap, not an active regression; documented +
  fidelity test added.
- **§4.4** — rewrite the false FlowManager adapter comment, add a loud
  `hasattr(flow_manager, "_llm")` guard (silent no-op would leave non-start
  members tool-less).
- **§4.5** — dependency reconciliation: pyproject has NO smart-turn extra today
  (it's an ADD not a rename), and onnxruntime `==1.20.1` (req) vs `>=1.20.1`
  (pyproject) diverge; relax to `>=` and align both files.
- **§6** — restructured into three tiers, flagging that the mocked unit suite
  cannot catch API breaks. Added Tier 2 (un-mocked import + offline behavioral
  probes) and Tier 3 (mandatory 6-call live smoke matrix before merge, since the
  legacy "unit tests + one post-deploy AICall" gate is structurally blind to live
  voice regressions). Noted the ~11 `@patch("run.OpenAILLMContext")` mass-failure
  and the need for a green 0.0.103 baseline first.
- **§9** — added two fast-follow items (LLMSwitcher migration, settings-API
  migration off deprecated LiveOptions).

Net: scope and code changes are unchanged in spirit (still a minimal
behavior-preserving upgrade), but the verification rigor is materially raised and
two previously-invisible risk classes (tool-schema fidelity, mock-blind testing)
are now explicit with concrete gates.

### Round 2 (re-review of v2) — applied to v3

A fresh independent reviewer confirmed all five round-1 findings were resolved
with no remaining factual error, and raised test-gate/dep-pin precision defects,
all applied here:
- `cancel_on_interruption` wrapper default `True` -> **`None`** to mirror upstream
  1.4.0 exactly (no invented value).
- onnxruntime reconciled toward the **pinned** `==1.20.1` in both files (not the
  floor) for Docker reproducibility.
- Tier-1 register_function regression mock: `Mock(spec=)` does not validate
  kwargs -> use **`create_autospec`** + an explicit `start_callback`-raises-TypeError
  test; the wrapper now rejects unknown `**kwargs`.
- §6 baseline contradiction clarified: 0.0.103 suite is likely NOT green today
  (stale Gemini assertions); baseline cleanup is step one, not an assumed green.
- §8 risk table gained a register_function row mapped to Tier-3 E-F; Tier-3 gained
  a Gemini smoke row G (gemini also moves to 1.4.0).
- `LLMContext(tools=NOT_GIVEN)` sentinel confirmed against the existing Gemini
  branch (run.py:623 already passes `tools=NOT_GIVEN`), so openai/grok inherit a
  proven call shape.

After v3 edits the design is internally consistent and the load-bearing fixes are
each backed by an executable gate. Treated as design-approved; remaining risk is
execution (implementation + the mandatory live smoke matrix), not design.

## 11. Implementation + PR Review Loop (round 1)

Implementation landed on this branch. Three independent PR reviewers (fresh
`delegate_task`) reviewed the verbatim code diff + the updated tests; the author
re-verified every code-level claim against the real 1.4.0 install rather than
trusting self-reports. Findings and dispositions:

- **smart-turn extra (R1/R3, raised as High):** the `local-smart-turn-v3` ->
  `local-smart-turn` rename was flagged as a possible real-call ImportError since
  the lazy `local_smart_turn_v3.LocalSmartTurnAnalyzerV3` import isn't exercised by
  the mocked suite. **Disposition: false alarm, verified.** In a `uv sync --frozen`
  env built from the new pins, `LocalSmartTurnAnalyzerV3` imports AND instantiates.
  The extra is named `local-smart-turn` in 1.4.0 and the V3 model ships inside it.
- **cancel_on_interruption None vs True (R1, Medium):** flagged as a possible
  behavior change. **Disposition: confirmed safe.** 1.4.0
  `LLMService.register_function` resolves `cancel_on_interruption=None` via
  `_resolve_tool_option(..., default=True)`, so `None` means "use the default
  (True)" — identical to the 0.0.x behavior. The `None` default is the correct
  upstream mirror.
- **FlowManager _llm guard untested (R2, High):** **fixed.** Added
  `test_init_team_pipeline_flowmanager_missing_llm_attr_guard` (RuntimeError when
  `_llm` absent, via `MagicMock(spec=[])`) and
  `test_init_team_pipeline_swaps_flowmanager_llm_to_router` (asserts
  `flow_manager._llm is routing_llm` on success).
- **Two in-scope team-init tests left red (R2, High):** **fixed.**
  `test_init_team_pipeline_start_member_not_found` had a stale regex
  (`"Unknown member_id"` -> `"start_member_id .* not found"`);
  `test_init_team_pipeline_active_service_none_guard` needed
  `mock_routing.cleanup = AsyncMock()` (awaited in the except path). Both now green.
- **requirements.txt unbounded floors (R3, High):** **fixed.** Capped majors:
  `pipecat-ai>=1.4.0,<2.0`, `pipecat-ai-flows>=1.2.0,<2.0`,
  `onnxruntime>=1.24.3,<1.25` in both requirements.txt and pyproject.toml. This
  neutralizes the named horizon breaks (pipecat 2.0 removes the deepgram
  LiveOptions wrapper; onnxruntime 1.25 exits the silero ceiling).
- **torch/libgomp on python:3.12-slim (R3, Medium):** the 1.4.0 silero/local-smart-turn
  extras newly pull torch 2.12.x (0.0.103 did NOT). Verified on a bare
  `python:3.12-slim` container running the exact Dockerfile install path: torch,
  `LocalSmartTurnAnalyzerV3` (loads `smart-turn-v3.2-cpu.onnx`), and deepgram
  `LiveOptions` all import/instantiate with no libgomp or missing-syslib error.
  However the default wheel is CUDA (`2.12.1+cu130`), ballooning site-packages to
  **5.5GB**. Since the smart-turn model is CPU-only and the runner does no GPU
  work, the **Dockerfile now installs CPU torch/torchaudio first**
  (`--index-url https://download.pytorch.org/whl/cpu`), which pip then satisfies
  before the requirements install, cutting site-packages to **1.7GB** (verified in
  the same container test). Functionally identical, 3.8GB smaller image.
- **tool-schema fidelity runtime guard (R1, Medium):** **added.**
  `_openai_tools_to_standard` now emits a WARNING when a tool carries top-level
  `strict`/`additionalProperties` (which FunctionSchema cannot carry), so a future
  catalog change fails loud instead of silently dropping them.
- **Pre-existing out-of-scope failures left as-is:** `test_google_tts_service_creation`
  and `test_google_tts_default_voice` are byte-identical to origin/main and fail on
  a mock-env TTS `InputParams` mismatch unrelated to this PR (both reviewers
  concurred these are acceptable to leave under minimal-change bias).

Mock suite after round 1: 132 passed, 2 failed (the two out-of-scope google_tts
pre-existing failures only). The R1 "test files not in the diff" finding was an
artifact of the reviewer receiving only the production-code diff; the test changes
are in the same commit/PR.

## 12. PR Review Loop (rounds 2 and 3) — APPROVE

Two further independent reviewers (fresh `delegate_task`) reviewed the round-1
fixes and the complete change holistically. Both returned **APPROVE**.

- **Round 2 (round-1 fix verification):** confirmed all round-1 fixes mechanically
  correct with no new defect. Raised one Medium (the CPU-torch Dockerfile line was
  unpinned -> reproducibility drift) and Lows (onnxruntime minor-cap rationale
  comment; warn log-noise). **Applied:** pinned `torch==2.12.1 torchaudio==2.11.0`
  in the Dockerfile (the exact versions the validated 1.7GB image resolved) and
  added a comment to requirements.txt explaining the deliberate onnxruntime `<1.25`
  minor cap (coupled to the smart-turn ONNX model). Re-verified in a real
  `python:3.12-slim` container that the pinned CPU torch survives the subsequent
  requirements.txt install (not re-pulled as CUDA) and smart-turn still loads.
- **Round 3 (holistic production-readiness):** APPROVE. Confirmed the 132/2 mock
  result, that the 2 failures are pre-existing google_tts tests identical to main,
  that the custom transport (barge-in FLUSH_MEDIA, push_frame monkeypatch) and
  16kHz are unchanged from main, that the universal-context migration is
  behavior-preserving (gemini+team already run it), the deferral list is
  appropriate, and `git revert` rollback is adequate. The only remaining gate is
  the documented Tier-3 live-smoke matrix, which is a separate pre-merge
  operational gate, not a code defect.

Review loop status: 3 rounds completed (round 1 = 3 reviewers with fixes applied,
rounds 2 and 3 = APPROVE). All Critical/High/Medium findings resolved. The PR is
code-complete and merge-eligible pending the mandatory Tier-3 live-smoke matrix
(7 real calls) and explicit merge authorization.
