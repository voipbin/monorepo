# Reconnect vad_config / smart_turn_enabled to the pipecat user aggregator (runner: `scripts/pipecat/`)

Ticket VOIP-1547. Branch `VOIP-1547-Fix-pipecat-vad-smart-turn`. Class: bug fix (settings accepted and stored, but ignored at call time). The settings are reconnected; removing the feature is out of scope.

## 1. Issue analysis

### 1.1 Ticket review (branch base `2ec2fac22`, re-checked against origin/main before the PR; pipecat-ai 1.12.0 in an isolated venv with the runner's extras)

Verdict: valid.

- `TransportParams` (`pipecat/transports/base_transport.py:66`) declares no `vad_analyzer`, `turn_analyzer` or `session_timeout`, and its `model_config` is `ConfigDict(arbitrary_types_allowed=True)` only (pydantic v2 default `extra=ignore`). `WebsocketClientParams(vad_analyzer=..., turn_analyzer=...)` in `create_websocket_transport` (run.py) therefore drops both without an error. `transports/websocket/client.py` and `base_input.py` never read them either. The same file of pipecat 1.4.0 (a local venv) has none of the three names, so this is not an upgrade regression.
- The user aggregator is where VAD now lives: `LLMUserAggregatorParams` has `vad_analyzer` and `user_turn_strategies`; `LLMUserAggregator.__init__` builds a `VADController` only when `params.vad_analyzer` is set (`llm_response_universal.py:767`). `_make_aggregator` passes only `empty_user_turn=None`. It is the only place that builds `LLMContextAggregatorPair` (run.py:72); the 7 call sites in `create_llm_service` and the team path (run.py:1110) all go through it.
- Data path is intact: bin-ai-manager stores `vad_config` / `smart_turn_enabled`; `pkg/pipecatcallhandler/runner.go:115-125` copies them from the aicall (`aicall.AIVADConfig`, `AISmartTurnEnabled`), `pythonrunner.go:104-128` posts them as `vad_config` / `smart_turn_enabled`, `main.py:126-127` (`PipelineRequest`) and `main.py:195-196` hand them to `init_pipeline` / `init_team_pipeline`. The loss is only inside run.py.
- Team calls: Go also fills per member `vad_config` / `smart_turn_enabled` (`run.go:58-59,265-266`; Python model `ResolvedAI`, main.py:75-76), but `init_team_pipeline` uses only the top-level values (run.py:1115-1120, the aicall's own AI) and builds one aggregator for the whole team (run.py:1110). Per-member values are not read today and stay unused here (a team call has one user aggregator; per-member switching would be a mid-call parameter change, out of scope).
- Not affected: the Go side. Out of scope and untouched: `session_timeout` (VOIP-1548), STT failover, mid-call parameter changes.

### 1.2 What a voice call does today (simulated STT timing on real pipecat 1.12.0 components, section 4)

Without `vad_analyzer` the aggregator gets no VAD frames. Default strategies still apply (start: VAD + Transcription; stop: a `TurnAnalyzerUserTurnStopStrategy` with a `LocalSmartTurnAnalyzerV3` built by `default_user_turn_stop_strategies()`, one model load per aggregator). With Deepgram or Google STT (`interim_results=True` at run.py:586, `enable_interim_results=True` at run.py:600):

- A turn starts (and the bot is interrupted) at the first interim or final transcript, not at the start of speech.
- The stop strategy runs its fallback path for "transcript without VAD stop" (`turn_analyzer_user_turn_stop_strategy.py`): after each transcript it waits `max(0, ttfs_p99_latency - stop_secs)` with `stop_secs=0` (no VAD), then ends the turn. `ttfs_p99_latency` comes from the STT service (`STTMetadataFrame`): Deepgram 0.35 s, Google 1.57 s.
- Each final transcript (`TranscriptionFrame`; interims do not touch the timer) restarts this wait. When the gap between consecutive finals is longer than `ttfs_p99 - stop_secs`, the turn ends in between, so one long utterance is cut into several turns, each triggering an LLM run and an interruption (reproduced in the harness with a final every 2.5 s). The STT vendor's endpointing, not the customer's setting, decides where a turn ends.
- `vad_config.confidence/start_secs/stop_secs/min_volume` have no effect, and the smart-turn model is loaded but never gets VAD signals, so `smart_turn_enabled` has no effect either (it is not "off", it is unused).

## 2. Design

### 2.1 Behavior

For calls that have audio input (single AI with `stt_type`; team call when `routing_stt` exists, i.e. at least one member has an `stt_type`), the user aggregator gets:

- `vad_analyzer = SileroVADAnalyzer(params=build_vad_params(vad_config, smart_turn_enabled))` (unchanged builder, so `smart_turn_enabled` still forces `stop_secs=0.2`).
- `user_turn_strategies = UserTurnStrategies(start=default_user_turn_start_strategies(), stop=[...])` (VAD + Transcription start, the library default list; passed explicitly because `UserTurnStrategies` fills only an empty side and the test pins both lists). Stop:
  - `smart_turn_enabled=True`: `TurnAnalyzerUserTurnStopStrategy(turn_analyzer=LocalSmartTurnAnalyzerV3())`.
  - `smart_turn_enabled=False`: `SpeechTimeoutUserTurnStopStrategy()` (`user_speech_timeout=0.6`, library default). It ends a turn from VAD stop, so `vad_config.stop_secs` takes effect again. The library default (smart turn) is not used here because the setting says smart turn is off.
  - Wait after speech end (smart turn off) = VAD `stop_secs` + 0.6 s, and the STT safety wait `max(0, ttfs_p99 - stop_secs)` runs in parallel, so the larger one wins. The strategy logs one WARN per session when `stop_secs != 0.2` ("built-in p99 values assume 0.2"); that is expected for customers who set `stop_secs` and is not suppressed.
- Voice calls without audio input and text-only sessions (no `stt_type`) keep exactly today's parameters (`empty_user_turn=None`, no VAD, library default strategies).

### 2.2 Code changes (all in `scripts/pipecat/run.py`)

1. New `build_user_turn_analyzers(vad_config, smart_turn_enabled) -> (vad_analyzer, turn_analyzer)`; replaces the two duplicated blocks (init_pipeline, init_team_pipeline).
2. `_make_aggregator(ctx, vad_analyzer=None, turn_analyzer=None)`: with no `vad_analyzer` returns today's params unchanged; otherwise builds the params of 2.1 (`empty_user_turn=None` is kept in both cases, 1.4.x parity pin).
3. `init_single_ai_pipeline` (called by `init_pipeline`): when `stt_type` is set build the analyzers before the init tasks (about 40 ms for both models, measured), inside a `try` that logs the same `[INIT] Pipeline initialization failed: <_error_text>` line and re-raises as the existing `gather` handler does (no task exists yet, nothing to cancel), pass them to `create_llm_service` through `**options`; the 7 `_make_aggregator(ctx)` calls inside `create_llm_service` forward `options.get("vad_analyzer")` / `options.get("turn_analyzer")`. The returned dict key `"vad_analyzer"` of `init_stt_and_input_ws` is read nowhere and is removed with the block it lives in. `init_team_pipeline`: build when `routing_stt` is set, before `_make_aggregator`; that code is outside any try today and its error path does not change; the per-member `create_llm_service` calls (run.py:1040, aggregator discarded) get no analyzers.
4. `create_websocket_transport`: drop the `vad_analyzer` / `turn_analyzer` parameters and the two ignored fields. `session_timeout` stays as it is (VOIP-1548).
5. Docs: `docs/domain.md` (new subsection on VAD and end of turn, including the stop_secs WARN).
6. Unit-test mocks: `conftest.py` replaces pipecat with MagicMocks; the new imports are added there: parent `pipecat.turns` (MagicMock), `pipecat.turns.user_turn_strategies` (`UserTurnStrategies`, `default_user_turn_start_strategies`), `pipecat.turns.user_stop` (`SpeechTimeoutUserTurnStopStrategy`, `TurnAnalyzerUserTurnStopStrategy`), and `LLMUserAggregatorParams` is added to the existing `llm_response_universal` mock (conftest.py:82 has only `LLMContextAggregatorPair`). Unit tests only assert the wiring (which classes and arguments); real behavior is the job of the real-library guards.

No new flags: behavior follows the existing two settings.

### 2.3 Default behavior change (must be in the PR text)

Default call = `vad_config` empty, `smart_turn_enabled=false`, Deepgram STT (section 4 numbers):

- Barge-in: today at the first STT transcript; after the change at VAD speech start (first transcript latency earlier).
- Turn splitting: today a gap between STT finals longer than 0.35 s (Deepgram) ends the turn and triggers an LLM run plus an interruption, so the 8.3 s utterance of the harness became 3 turns; after the change it is one turn.
- First response after the end of speech: about 0.4 s today (last transcript + 0.35 s) vs about 0.8 s (`stop_secs` 0.2 + 0.6 s) with smart turn off; about 0.2 s with smart turn on when the model judges the sentence complete, or the library timeout (3 s) when it judges the phrase incomplete.

Whether the change needs a customer notice and whether the 0.6 s should be tuned are product decisions outside this change.

### 2.4 Compatibility and risk

- No DB, API, OpenAPI, RST or Go change. No schema change. The request fields already exist.
- Rolling deploy: pure runner change, independent of the Go containers.
- Risk: pipecat internals. Pinned by the real-library guards (section 3).
- Rollback: revert the commit.

### 2.5 Alternatives rejected

- Remove the settings: rejected, the feature stays.
- Pass `vad_analyzer` to the transport: no such field in 1.x.
- Keep the library default stop strategy for smart-turn-off: it would run smart turn when the customer disabled it.

## 3. Verification plan

- Unit tests (`scripts/pipecat/test_user_turn_wiring.py`, `test_init_pipeline.py`): analyzers built once per audio call and forwarded; a failure of `build_user_turn_analyzers` in the single-AI path logs the `[INIT]` error and raises; text-only and no-STT team calls unchanged; member `create_llm_service` calls get none; transport params no longer carry the two fields.
- Real-library guards in a new file `scripts/pipecat_realtest/test_vad_turn_real.py` (`pytest --noconftest`, not run by CI like the other files there): (a) `TransportParams.model_fields` has no vad/turn field (the premise); (b) `_make_aggregator` result: `agg._params.vad_analyzer` is the analyzer and `agg._params.user_turn_strategies` has the expected start and stop classes for both `smart_turn_enabled` values, the turn analyzer is the passed instance, and `agg._vad_controller` exists (private attribute, pinned to 1.12 on purpose); (c) VADParams equal `vad_config`; smart turn forces `stop_secs=0.2`; no analyzers gives today's params; (d) behavior test with the real Silero and smart-turn models. A speech clip cannot be synthesized (Silero ignores tones) and is not committed (binary), so (d) takes the clip path from `VOIP1547_SPEECH_WAV` and skips without it; the PR records the measured numbers from that run (section 4), which are the acceptance record for the default-behavior comparison.

## 4. Measurement

Harness (scratch, not committed): real pipecat 1.12.0 aggregator, real Silero and smart-turn models, a 16 kHz clip of one 8.3 s utterance fed in real time with the bot marked as speaking; STT is simulated as Deepgram (`STTMetadataFrame` ttfs_p99 0.35 s, first interim at 1.2 s, a final segment at 2.5 s and 5.0 s, last final 0.3 s after speech end). Times are seconds from the start of the clip; speech ends at about 8.55 s.

| Mode | Barge-in (interruption) | Turns / LLM runs | Last LLM run, after speech end |
|---|---|---|---|
| Today (no VAD in aggregator) | 1.20 | 3 (2.85, 5.35, 8.96) | +0.43 |
| VAD + speech timeout (smart turn off, the new default) | 0.74 | 1 | +0.77 |
| VAD + smart turn on | 0.74 | 1 | +0.23 |

The interruption gain (0.46 s) is the simulated first-interim delay; with real STT it is the real first-transcript latency. Three runs of each mode on the implemented code gave the same values within 0.05 s (barge-in 1.20 / 0.75 / 0.75; last LLM run after speech end 0.40-0.41 / 0.77 / 0.23-0.28; LLM runs 3 / 1 / 1).
