# Design: gate team-pipeline TTS/STT on the session's audio mode (VOIP-1481)

Date: 2026-09-07
Status: approved; implemented in VOIP-1481
Analysis: `2026-09-07-team-pipeline-text-mode-issue-analysis.md` (same directory)

## 1. Problem (from the analysis)

`init_team_pipeline` in `bin-pipecat-manager/scripts/pipecat/run.py` builds per-member TTS/STT services from each member AI's voice configuration regardless of whether the pipecatcall is a voice call or a text-only session (webchat/conversation, task, contact_case listen turn). For a text session the request-level `stt_type`/`tts_type` are empty (ai-manager blanks them for every non-call AIcall reference), but `init_pipeline` does not forward them to the team branch, so the team path has no mode signal at all. When a member uses Google TTS/STT the constructor fails on the missing Application Default Credentials and the whole pipeline is rejected; ai-manager's backstop then sends the fallback sentence. For any other vendor the text session still needlessly constructs TTS/STT services and the audio input transport.

## 2. Goal / non-goals

Goal
- A text-only team session (request-level `stt_type` and `tts_type` both empty) builds exactly what the single-AI text path builds: per-member LLM services, the routing LLM, the context aggregator, the output transport, the FlowManager. No TTS, no STT, no input transport.
- A voice team session (request-level types non-empty) behaves exactly as today: per-member TTS/STT from each member's own configuration, routing TTS/STT, input transport with VAD.

Parity caveat: no team pipeline of any kind has completed in production since the containers started (analysis 3.4: 17/17 failed at init), so "builds what the single-AI text path builds" is a structural argument from the code, not an observed one. The only proof is the production verification in section 7 (steps 2-3).

Non-goals
- Provisioning Google ADC in the runner (VOIP-1482). Consequence: after this change a **voice** team call whose members use Google TTS/STT still fails identically until VOIP-1482 lands; matrix row 4 below describes the intended, not the currently reachable, behaviour.
- Validating unsupported TTS/STT vendors in ai-manager (VOIP-1482 scope note).
- Shortening the 18 s gap between a runner 400 and `terminate` (possible follow-up).
- Any Go-side change to the `/run` payload.

## 3. Mode signal

Use the request-level `stt_type` / `tts_type` **presence**, exactly like `init_single_ai_pipeline` does. Facts that make this correct today (all verified in the analysis):

- Only three producers of pipecatcall start requests exist, all in ai-manager. The voice-call path always sends non-empty types (defaults `deepgram`/`elevenlabs` fill any empty AI value); the task, listen and conversation paths send empty types.
- pipecat-manager's own transport dispatch agrees: only `aicall.ReferenceType == call` gets an Asterisk media leg.
- The Go request struct marks both fields `omitempty`, so empty arrives in Python as `None`; the gate is the same `if tts_type:` shape as the single-AI path.
- The `/run` request already carries these fields alongside `resolved_team`; nothing new crosses the Go/Python boundary.

Gate TTS and STT **separately** on their own request-level field (as the single-AI path does), not on a combined "voice" boolean. In practice a voice call sets both, but separate gating keeps the two paths symmetric and costs nothing.

The request-level **values** are never used for members. They are a snapshot of the start member's AI; each member keeps building its service from its own `tts_type` / `tts_voice_id` / `stt_type`.

Rejected alternatives
- Derive mode from a reference type in pipecat-manager or the runner: `pipecatcall.ReferenceType` is always `ai_call` here; the AIcall reference type is not in the payload; re-deriving duplicates a policy ai-manager already applied and would drift.
- Strip member TTS/STT on the Go side when request-level types are empty: works, but moves the decision away from the place that already makes it for single-AI sessions, and leaves the Python function unable to defend itself when called with a team payload directly (7 test call sites do that).
- Add an explicit `mode` field to the request: cleaner in the long run but touches Go + pydantic + tests for no behavioural gain today. Can be revisited if a future producer needs an audio session with empty types.

## 4. Changes

### 4.1 `run.py`

`init_pipeline` (team branch): forward `stt_type=stt_type, tts_type=tts_type` to `init_team_pipeline`. Nothing else changes in the dispatch.

`init_team_pipeline`: add keyword parameters `stt_type: str = None, tts_type: str = None`. Defaults keep the 7 existing test call sites valid and mean "text-only", mirroring `init_single_ai_pipeline`'s defaults. Trade-off acknowledged: a future caller that forgets to forward the types gets a silent text-only pipeline rather than a loud failure; the docstring must state that `None` means text-only, and test 4 pins the one production caller's forwarding. In the member loop:

```python
if tts_type and ai.get("tts_type"):
    tts_services[mid] = create_tts_service(ai["tts_type"], voice_id=ai.get("tts_voice_id"), language=tts_language)
if stt_type and ai.get("stt_type"):
    stt_services[mid] = create_stt_service(ai["stt_type"], language=stt_language)
```

Everything downstream is already conditional on `tts_services` / `stt_services` being empty (`routing_tts`/`routing_stt` None at `run.py:621-629`, input transport gated on `routing_stt` at `run.py:650-657`, stage list, `build_team_flow`, transition handler guards `team_flow.py:155-158`, cleanup). One INFO log line `[TEAM][INIT] Text-only session; skipping per-member TTS/STT`, emitted once per pipeline (not per member) and only when **both** request-level `stt_type` and `tts_type` are falsy, so the partial-types shape (test 3) does not log a misleading "text-only" line.

Update the docstring of `init_team_pipeline` to state the gate and the invariant it relies on.

### 4.2 Go side

No change. `resolveTeamForPython` keeps sending full member configuration; the payload stays a complete description of the team, and the runner decides what to instantiate for the session, as it does for single AIs.

### 4.3 Docs

- Add the analysis and this design under `bin-pipecat-manager/docs/plans/`.
- One or two sentences in `bin-pipecat-manager/docs/domain.md` under `## Pipecat Pipeline` (the section that describes what `run.py` constructs; `architecture.md` does not mention teams at all) stating that request-level `stt_type`/`tts_type` presence selects voice vs text-only, that the single-AI and team paths both honour it, and that team members' TTS/STT are built from member config only in voice mode.

## 5. Behaviour matrix

| session | request-level stt/tts | members | TTS/STT built | input transport | output transport |
|---|---|---|---|---|---|
| conversation (webchat) | empty/empty | google, google | none | no | yes |
| task | empty/empty | mixed vendors | none | no | yes |
| contact_case listen turn | empty/empty | any | none | no | yes |
| voice call | deepgram/elevenlabs (defaults or snapshot) | google, google | per member, google | yes | yes |
| voice call | deepgram/elevenlabs | start member without TTS, other member with TTS | only for the member that has one (unchanged from today) | yes | yes |
| voice call | deepgram/elevenlabs | mixed vendors | per member, each with its own vendor | yes | yes |

Rows 1-3 are the new behaviour; rows 4-6 must be byte-for-byte today's behaviour. Pre-existing and out of scope: in voice mode a team whose members all lack `stt_type` gets no input transport (deaf call), and one whose members all lack `tts_type` gets no TTS (`run.py:621-629,650-657` today); the gate does not change that. Rows 1-3 collapse into a single unit test (test 1): `init_team_pipeline` has no notion of the AIcall reference type, and all three sessions arrive as empty request-level types, so one text-mode test covers them.

## 6. Tests (pytest, `scripts/pipecat/test_init_pipeline.py`)

Run with `uv run --project . --with pytest --with pytest-asyncio python -m pytest -q` from `scripts/pipecat` (verified: 11 existing tests pass at b41b11ea4).

New tests (tests 1-4 must fail before the change and pass after; test 6 is coverage for a newly reachable path and passes both before and after):

1. `test_init_team_pipeline_text_mode_skips_member_tts_stt`: two members both `tts_type="google"`, `stt_type="google"`; call with no `stt_type`/`tts_type`. Patch `run.create_tts_service` and `run.create_stt_service` and assert neither was called; assert `create_websocket_transport` was called only for `"output"`; assert returned `routing_tts` and `routing_stt` are `None` and `transport_input` is `None`.
2. `test_init_team_pipeline_voice_mode_builds_member_tts_stt_from_member_config`: request-level `stt_type="deepgram"`, `tts_type="elevenlabs"`; member A `tts_type="google"`, `tts_voice_id="v-a"`, `stt_type="google"`; member B no `tts_type`, `stt_type="whisper"` (distinct from A's vendor and from the request-level `deepgram`). Assert `create_tts_service` called once with `("google", voice_id="v-a", language=...)`, `create_stt_service` called twice with the members' own vendors in member order, and the request-level values never appear in those calls.
3. `test_init_team_pipeline_partial_request_types_gate_independently`: `stt_type="deepgram"`, `tts_type=None` → STT services created, TTS not (mirrors single-AI). Plus the mirror case (added in code review) `test_init_team_pipeline_partial_request_types_tts_only_builds_no_stt`: `tts_type="elevenlabs"`, `stt_type=None` → TTS created, no STT, no input transport.
4. `test_init_pipeline_forwards_request_types_to_team_branch`: patch `run.init_team_pipeline` and assert `init_pipeline(..., stt_type="deepgram", tts_type="elevenlabs", resolved_team=...)` forwards both as kwargs; and with both `None` forwards `None`.
5. Existing 11 tests keep passing unchanged (they call `init_team_pipeline` without types; those members have no TTS/STT so behaviour is identical).
6. `test_team_flow.py`: the handler returned by `_create_transition_handler` invoked with `routing_tts=None, routing_stt=None` switches only the LLM router, updates `current_state`, and does not raise. This path (`team_flow.py:145-169`, guards at 155-158) becomes reachable in production for the first time with this change and is currently untested: every team test in `test_init_pipeline.py` patches `build_team_flow` away and `test_team_flow.py` never awaits a handler. Mechanics: needs `@pytest.mark.asyncio` (strict mode, no `asyncio_mode` in pyproject) and a patch of `team_flow._notify_member_switched` (the handler fires it with `asyncio.create_task`, `team_flow.py:163`) so no pending task is destroyed at test end.

Test 4 detail: production calls `init_pipeline` with positional arguments (`main.py:124-138`), so the test asserts on what the patched `init_team_pipeline` receives (`stt_type`/`tts_type` kwargs), not on how `init_pipeline` itself was called.

What the tests cannot show: `conftest.py` stubs the whole `pipecat` package tree (including `pipecat.services.google.tts/stt`), so `create_tts_service("google")` returns a MagicMock in tests and never touches ADC. The tests pin the gate; only section 7 proves the production effect.

CI: `.circleci/config_work.yml` currently runs no tests at all for this service (the Go test job is commented out since 2026-03, tracked as VOIP-1356) and has never run the pytest suite. Both the Go verification and the pytest run are local-only for this PR; the PR must state both runs and their results. Adding the Python step to CI belongs with VOIP-1356, out of scope here.

Mutation check: revert the gate and confirm test 1 fails on the `create_tts_service` assertion and test 4 fails on the missing kwargs.

## 7. Verification plan (production)

1. Merge → CI image → Komodo deploy of `bin-pipecat-manager` stack (both manager+runner pairs). Confirm the two runner containers restarted and `md5sum run.py` in the container matches the merged commit.
2. Send a visitor message through the voipbin.net webchat widget (public widget boot → session → inbound message). Expect an AI answer, not the backstop sentence. Check `script-runner` logs: `run_request ... has_resolved_team: true` followed by `[TEAM][INIT] Text-only session; skipping per-member TTS/STT` and `[TEAM][INIT][total]`, no `Pipeline validation failed`.
3. In the same session, send a message that should make the start member hand off to the other member (for this team: a sales question). Expect a `member_switched` event / reply from the other member with no error, since the text-mode transition path (`team_flow.py:145-169` with both routers `None`) has never run in production. Also ask something that triggers a member tool, so team-mode tool execution (`team_flow.py` `_call_go_tool_endpoint`) runs once in production, and confirm `[TEAM][CLEANUP]` appears in the runner log after the turn ends (normal team teardown, `run.py:779-794`, has also never run in production).
4. Check ai-manager logs: no new `Backstop reply sent` for the new AIcall.
5. Re-test VOIP-1457 on the square-main widget; close it if it no longer reproduces.
6. Voice regression: production has had zero voice run_requests since 2026-09-05, and a voice team call with these (google) members would fail on VOIP-1482 regardless, so the voice rows are protected by tests 2, 3 and 6 plus the unchanged code path. A zero-cost live check is possible (call to a virtual number with a throwaway team whose members use elevenlabs/deepgram, allowed by the repo's API testing guidelines); do it if time permits, otherwise state the omission in the PR.

## 8. Rollback

One Python source file plus tests and docs; revert the PR and redeploy. No schema, no payload, no Go change.
