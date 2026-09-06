# Implementation Plan: gate team-pipeline TTS/STT on session audio mode (VOIP-1481)

Date: 2026-09-07
Status: executed (rev 4; test 2 vendor and mirror test 3b amended during code review)
Design: `2026-09-07-team-pipeline-text-mode-gate-design.md` (approved). Analysis: `2026-09-07-team-pipeline-text-mode-issue-analysis.md` (approved).
Branch/worktree: `VOIP-1481-Gate-team-pipeline-tts-stt-on-text-sessions` at `.worktrees/` of `~/gitvoipbin/monorepo`, based on `origin/main` `b41b11ea4`.

All paths below are relative to `bin-pipecat-manager/`. Only Python and docs change; no Go, no payload, no schema.

## 0. Acceptance criteria

- A1. `init_team_pipeline(..., stt_type=None, tts_type=None)` with members that declare `tts_type`/`stt_type` creates no TTS/STT services, no input transport, returns `routing_tts is None`, `routing_stt is None`, `transport_input is None`; output transport and FlowManager path unchanged.
- A2. `init_team_pipeline(..., stt_type="deepgram", tts_type="elevenlabs")` creates per-member TTS/STT from each member's own config exactly as today (members without a type are skipped as today; request-level values never reach `create_tts_service`/`create_stt_service`).
- A3. TTS and STT are gated independently by their own request-level field.
- A4. `init_pipeline` forwards request-level `stt_type`/`tts_type` to `init_team_pipeline`.
- A5. The transition handler works with both routers `None`.
- A6. All pre-existing tests keep their baseline result: 140 pass; the 2 pre-existing failures in `test_run.py::TestCreateTTSService` (`test_google_tts_service_creation`, `test_google_tts_default_voice`: stale assertions from before the Chirp3 default-voice change, unrelated to this work, never caught because CI runs no pytest) remain exactly 2 and are not touched by this PR (recorded on VOIP-1356).
- A7. Docs: analysis, design, plan in `docs/plans/` with README index rows; `docs/domain.md` Pipecat Pipeline section states the mode rule.
- A8. Manual mutation check recorded: with the signature kept but the two guards and the forwarding reverted, tests 1, 3, 3b and 4 fail (test 2 keeps passing because the voice path is unchanged). Final suite: `2 failed, 146 passed` = 140 baseline + 6 new tests (tests 1-4, 3b, and the transition-handler test).

## 1. Test environment

```bash
cd bin-pipecat-manager/scripts/pipecat
uv run --project . --with pytest --with pytest-asyncio python -m pytest -q
```

Baseline at b41b11ea4 (worktree, untouched code): `2 failed, 140 passed in 0.42s`. The 2 failures are `test_run.py::TestCreateTTSService::test_google_tts_default_voice` (expects `voice_id='default_voice_id'`; code now defaults to `en-US-Chirp3-HD-Charon` at `run.py:365-367`) and `::test_google_tts_service_creation` (asserts `assert_called_once_with(voice_id=...)` only; code now also passes `params=GoogleTTSService.InputParams(language=lang)` at `run.py:377-380`); both pre-existing stale assertions, out of scope, recorded on VOIP-1356 as a precondition for wiring pytest into CI. `uv run --project .` (re)creates `.venv` from `uv.lock` (python 3.11); `.venv`/`venv` are gitignored.

Note: `conftest.py` stubs the whole `pipecat` tree, `routing_tts`/`routing_stt`/`team_flow`, `aiohttp`, so tests observe the gate, not the Google ADC failure (design 6). `test_team_flow.py` pops the `team_flow` stub to import the real module (`test_team_flow.py:9`).

## 2. Steps (TDD, one commit at the end)

### Step 1. Baseline

Done: `2 failed, 140 passed` (see section 1). Every later run must show the same 2 failures and only them, plus the new tests passing.

### Step 2. RED: add tests 1-4 to `scripts/pipecat/test_init_pipeline.py`

Reuse the success-path harness from `test_init_team_pipeline_swaps_flowmanager_llm_to_router` (`test_init_pipeline.py:238-293`): patches for `run.create_llm_service`, `run.RoutingLLMService`, `run.create_websocket_transport`, `run.Pipeline`, `run.PipelineTask`, `run.build_team_flow`, `run.FlowManager`, `run.task_manager`, with `mock_routing.active_service = mock_llm`, `flow_manager_stub._llm = MagicMock()`, `flow_manager_stub.initialize = AsyncMock()`. Extract a small module-level helper `_run_team_init(resolved_team, **kwargs)` that builds those mocks, additionally patches `run.create_tts_service`, `run.create_stt_service`, `run.RoutingTTSService`, `run.RoutingSTTService`, `run.SileroVADAnalyzer`, `run.build_vad_params`, awaits `init_team_pipeline(id=..., resolved_team=resolved_team, **kwargs)`, and returns `(ctx, mocks)` where `mocks` exposes the TTS/STT/transport mocks. Existing tests are left untouched (no refactor of the 11 existing tests).

Fixture: `_two_google_members()` → members `member-1` (start) `{engine_model: "gemini.gemini-2.5-flash", engine_key: "", tts_type: "google", tts_voice_id: "en-US-Chirp3-HD-Gacrux", stt_type: "google"}` and `member-2` `{..., tts_type: "google", tts_voice_id: "", stt_type: "google"}` (mirrors the production team). Every `resolved_team` used by tests 1-3 must set `start_member_id` to a member present in `members` (`member-1`), otherwise `run.py:632-634` raises `ValueError("start_member_id ... not found")` and masks the intended RED/GREEN outcome.

All four new tests are `async def` and MUST carry `@pytest.mark.asyncio` (pytest-asyncio strict mode: no `asyncio_mode` in `pyproject.toml`; all 11 existing tests are marked). An unmarked async test is skipped with a warning instead of failing, which would silently break RED.

1. `test_init_team_pipeline_text_mode_skips_member_tts_stt` (A1): call with no `stt_type`/`tts_type`. Assert `create_tts_service.assert_not_called()`, `create_stt_service.assert_not_called()`, `RoutingTTSService`/`RoutingSTTService` not called, `create_websocket_transport` called exactly once with first positional `"output"`, `ctx["routing_tts"] is None`, `ctx["routing_stt"] is None`, `ctx["transport_input"] is None`, `create_llm_service.call_count == 2`.
2. `test_init_team_pipeline_voice_mode_builds_member_tts_stt_from_member_config` (A2): call with `stt_type="deepgram"`, `tts_type="elevenlabs"`, `tts_language="en-US"`, `stt_language="en-US"`; members: A `tts_type="google"`, `tts_voice_id="v-a"`, `stt_type="google"`; B no `tts_type`, `stt_type="whisper"` (a third vendor, distinct from both A's and the request-level `deepgram`, so mixed-vendor routing and request-level leaks are both detectable). Assert `create_tts_service.assert_called_once_with("google", voice_id="v-a", language="en-US")`; `create_stt_service` call args list `== [call("google", language="en-US"), call("whisper", language="en-US")]`; `"elevenlabs"` does not appear in any `create_tts_service` call and `"deepgram"` in no `create_stt_service` call; `RoutingTTSService`/`RoutingSTTService` each constructed once; `create_websocket_transport` called twice (`"input"` then `"output"`); `ctx["transport_input"] is not None`.
3. `test_init_team_pipeline_partial_request_types_gate_independently` (A3): `stt_type="deepgram"`, `tts_type=None`, members both with google TTS+STT. Assert `create_stt_service.call_count == 2`, `create_tts_service.assert_not_called()`, `ctx["routing_stt"] is not None`, `ctx["routing_tts"] is None`, `ctx["transport_input"] is not None`.
3b. `test_init_team_pipeline_partial_request_types_tts_only_builds_no_stt` (A3, mirror, added during code review): `stt_type=None`, `tts_type="elevenlabs"`. Assert `create_tts_service.call_count == 2`, `create_stt_service.assert_not_called()`, `ctx["routing_tts"] is not None`, `ctx["routing_stt"] is None`, `ctx["transport_input"] is None`, exactly one `create_websocket_transport` call for `"output"`. Together with test 3 it fails a combined `(tts_type or stt_type)` gate.
4. `test_init_pipeline_forwards_request_types_to_team_branch` (A4): `patch("run.init_team_pipeline", new=AsyncMock(side_effect=lambda *a, **k: {}))` (fresh dict per call, since `init_pipeline` mutates the returned ctx at `run.py:106`); call `init_pipeline("id", "openai.gpt-4o", "k", [], "deepgram", "en-US", "elevenlabs", "en-US", "voice", [], resolved_team={"id": "t", "start_member_id": "m", "members": []})` positionally, mirroring `main.py:124-138`. Assert the mock was awaited once and `call_args.kwargs["stt_type"] == "deepgram"`, `call_args.kwargs["tts_type"] == "elevenlabs"`; second call with `None`/`None` asserts both kwargs are `None` (present, not missing).

Run: all four must FAIL at RED. Test 1 fails on `create_tts_service.assert_not_called()` (the current code builds member TTS). Tests 2 and 3 fail with `TypeError: init_team_pipeline() got an unexpected keyword argument ...` (whichever of `stt_type`/`tts_type` CPython reports first) because the parameters do not exist yet (`run.py:566-574`); the pass criterion for RED is the `TypeError`, not the exact keyword name. Test 4 fails with `KeyError` on the missing kwargs. Record the four failure modes.

### Step 3. Coverage test 6 in `scripts/pipecat/test_team_flow.py` (A5)

The file currently imports only `sys` and `MagicMock` (`test_team_flow.py:3-4`); add `import asyncio`, `import pytest`, `from unittest.mock import AsyncMock, patch`, and extend the `team_flow` import with `_create_transition_handler`.

`test_transition_handler_with_no_tts_stt_routers`: `routing_llm = MagicMock()`; `member_nodes = {"m2": {"name": "m2"}}`; `current_state = {"active_member_id": "m1"}`; `resolved_team = _make_team([...], "m1")`; build the handler with `next_member_id="m2"`, `routing_tts=None, routing_stt=None`, `pipecatcall_id="pc-1"`, `function_name="transfer_to_m2"`; `with patch("team_flow._notify_member_switched", new=AsyncMock())`; await `handler({}, MagicMock())`. Assert result `== ({"status": "transferred"}, member_nodes["m2"])`, `routing_llm.set_active_member.assert_called_once_with("m2")`, `current_state["active_member_id"] == "m2"`, and the notify mock awaited once with `("pc-1", "m1", "m2", "transfer_to_m2", resolved_team)`. Mark `@pytest.mark.asyncio`; let the event loop settle (`await asyncio.sleep(0)`) before asserting the notify call. Expected to PASS before and after the change (documented as coverage, not RED).

### Step 4. GREEN: `scripts/pipecat/run.py`

- `init_pipeline` (lines 97-105): add `stt_type=stt_type, tts_type=tts_type,` to the `init_team_pipeline` call.
- `init_team_pipeline` signature (lines 566-574): add `stt_type: str = None, tts_type: str = None,` after `tts_language`. Docstring: "`stt_type`/`tts_type` are the request-level types from bin-ai-manager. Their presence selects the session's audio mode: both `None` (the default) means a text-only session (conversation, task, contact_case listen turn) and no per-member TTS/STT or input transport is built; non-empty means a voice call and each member's TTS/STT is built from that member's own config. The values themselves are never used for members."
- Member loop (lines 602-611): change the two guards to `if tts_type and ai.get("tts_type"):` and `if stt_type and ai.get("stt_type"):`.
- After the loop's summary log (line 615), add: `if not stt_type and not tts_type: logger.info(f"[TEAM][INIT] Text-only session; skipping per-member TTS/STT. pipeline id={id}")`. Emitted once, only when both are falsy.

Run the full suite: everything passes.

### Step 5. Mutation check (A8)

Temporarily revert the two guards and the forwarding (git stash of `run.py` or manual), run tests 1-4: tests 1, 3, 4 fail; restore. Record the failing assertion names in the PR body.

### Step 6. Docs (A7)

- `docs/domain.md`, `## Pipecat Pipeline`: after the STT/TTS provider lines add a short paragraph: "Audio mode is selected by the request-level `stt_type`/`tts_type` that bin-ai-manager sends: both empty means a text-only session (conversation, task, contact_case listen turn) and no STT/TTS or audio input is built; non-empty means a voice call. Both the single-AI and the team pipeline honour this. In team mode each member's TTS/STT is built from that member's own configuration, and only in voice mode (VOIP-1481)."
- `docs/plans/README.md`: add index rows for the three 2026-09-07 documents.
- Design doc status line → "approved; implemented in VOIP-1481" (status metadata only, no content change to the signed-off text); plan doc status → "executed".

### Step 7. Verification before commit

```bash
cd bin-pipecat-manager/scripts/pipecat && uv run --project . --with pytest --with pytest-asyncio python -m pytest -q
cd ../.. && git status --short && git diff --stat
```

No Go files touched → the full Go verification chain (`go mod tidy && go mod vendor && go generate ./... && golangci-lint run`) is not required, but run `go build ./... && go test ./...` in `bin-pipecat-manager` once as a sanity check that nothing else in the worktree changed. RST docs (`bin-api-manager/docsdev/source/`) are not affected: no API, webhook or billing surface changes; state this in the PR body.

### Step 8. Commit, push, PR

- Pull-main/conflict check from the worktree: `git fetch origin main`, `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`, `git log --oneline HEAD..origin/main`.
- One commit. Per the monorepo `CLAUDE.md` the commit title MUST match the branch name exactly; body bullets carry the project prefix:
  ```
  VOIP-1481-Gate-team-pipeline-tts-stt-on-text-sessions

  - bin-pipecat-manager: skip per-member TTS/STT and the input transport in init_team_pipeline when the request-level stt_type/tts_type are empty (text-only session)
  - bin-pipecat-manager: forward request-level stt_type/tts_type from init_pipeline to the team branch
  - bin-pipecat-manager: add pytest coverage for text-mode/voice-mode/partial gating, init_pipeline forwarding, and the transition handler with no TTS/STT routers
  - bin-pipecat-manager: document the audio-mode rule in docs/domain.md and add the VOIP-1481 analysis/design/plan docs
  ```
- Push branch, open PR titled `VOIP-1481-Gate-team-pipeline-tts-stt-on-text-sessions`. Body per the monorepo/global format: a narrative paragraph first (which also states, in prose, the pytest run result, that Go is untouched, that CI runs no tests for this service (VOIP-1356), that voice regression is covered by tests only, the VOIP-1482 caveat for Google voice teams, and that RST docs are unaffected), then the `bin-pipecat-manager:` bullets. No markdown headers, no "Test plan" section, no AI attribution.
- Do not merge; wait for 대표님's explicit instruction.

### Step 9. After merge (separate approval)

Follow design section 7: confirm runner containers picked up the image (`md5sum run.py` vs merged commit), send a visitor message on the voipbin.net widget, drive a member transition and a tool call, check runner/ai-manager logs, re-test VOIP-1457, then transition VOIP-1481 to Done.

## 3. Risks and rollback

- Risk: a hidden consumer expects `routing_tts`/`routing_stt` non-None in team mode. Mitigation: design review enumerated every consumer (`run.py:621-629,650-657,661-670,753-763,779-794`, `team_flow.py:46-47,155-158`); test 1 exercises the return path; test 6 the transition path.
- Risk: voice regression. Mitigation: tests 2 and 3 pin per-member creation with the members' own values; the guards only add a conjunct that is always true in voice mode.
- Rollback: revert the PR; no data or payload change.

## 4. Out of scope (tickets)

- VOIP-1482: Google ADC missing in the runner; unsupported vendor cast in ai-manager.
- VOIP-1356: CI test step for this service.
- Follow-up candidate (no ticket yet): fail-fast on runner 400 in `RunnerStart` to cut the ~18 s wait before the backstop.
