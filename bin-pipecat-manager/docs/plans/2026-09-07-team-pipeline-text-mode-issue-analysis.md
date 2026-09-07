# Issue Analysis: webchat AI Talk (Team) always replies with backstop text

Date: 2026-09-07 (rev 5, after review rounds 1-4; round 4 approved, non-blocking notes folded in)
Reporter: 대표님 (CEO/CTO), observed on voipbin.net webchat widget ("voipbin main webchat", widget 8d566129)
Related AIcall: a9b2c39b-85c3-40db-8fa0-647276f93c2f, pipecatcall 9770d769-1c2a-463e-abc1-521751add341
Observed message: 871c1af6-9d03-4015-aca1-58317833113f, text "Sorry, I'm having trouble responding right now. Please try again." (sender_id nil, direction outbound)

## 1. Symptom

The webchat widget's Message Flow ("webchat message voipbin main") routes every inbound visitor message straight to an `ai_talk` action whose assistance is the Team `voipbin admin helper` (d414dc93). Every visitor message produces the fixed fallback sentence instead of an AI answer. Screenshot shows two visitor messages and two identical fallback bubbles.

## 2. Issue validity (still valid: YES)

Reproduced in production on 2026-09-06 21:12-21:13 UTC. The deployed Python runner scripts are byte-identical to `origin/main` (md5 of `run.py`, `team_flow.py`, `main.py`, `routing_llm.py` match between `voipbin-pipecat-script-runner-1` and monorepo `b41b11ea4`). Nothing on main fixes this yet.

Jira: no ticket describes this defect (searched VOIP, 90 days, for team / credentials / backstop / webchat AI). VOIP-1457 ("square-main public webchat: AI reply hangs indefinitely on Thinking state", Backlog, 2026-09-04) reports a different symptom (hang, no fallback bubble) on the same widget two days earlier. It may share this root cause (the team pipeline never started, so no reply), but the missing fallback bubble is unexplained by this analysis. Decision: create a new ticket for this defect and link VOIP-1457 as "relates to"; do not mark VOIP-1457 as a duplicate. Re-test VOIP-1457 after this fix is deployed and close it only if it no longer reproduces.

## 3. Evidence chain (production logs, bm-nyc-01)

All timestamps UTC, 2026-09-06.

| t | component | fact | source |
|---|---|---|---|
| 21:12:36.42 | bin-ai-manager-1 | `startAIcallByMessaging` created AIcall a9b2c39b: `assistance_type=team`, `assistance_id=d414dc93`, `reference_type=conversation`, `ai_engine_model=gemini.gemini-2.5-flash`, `current_member_id=30cef188` | ai-manager log |
| 21:12:36.43 | bin-ai-manager-1 | `startPipecatcall`: "Determined variables. sttType: , ttsType: , ttsVoiceID: " (empty by design for non-call references, `bin-ai-manager/pkg/aicallhandler/start.go:807-816`) | ai-manager log |
| 21:12:36.45 | pipecat-manager-1 | `runnerStartScript` → `resolveTeamForPython` → "Resolved team for python runner" → "Sending request to python runner" | pipecat-manager log |
| 21:12:36.458 | script-runner-1 | `run_request`: `llm_type=gemini.gemini-2.5-flash`, `stt_type=null`, `tts_type=null`, `has_resolved_team=true` → `[TEAM][INIT] Starting team pipeline` | script-runner log |
| 21:12:48.522 | script-runner-1 | `Pipeline validation failed (id=9770d769): No valid credentials provided.` → HTTP 400. No `[TEAM][INIT] Member ... services created` line was logged, so the first member's service construction failed (see 3.1 for which constructor) | script-runner log |
| 21:12:48.523 | pipecat-manager-1 | `RunnerStart`: "Could not start the pipecat runner script: ... 400 Bad Request, body: {"detail":"No valid credentials provided."}" | pipecat-manager log |
| 21:13:06.45 | pipecat-manager-1 | `terminate` → publishes `pipecatcall_terminated`; python `/stop` returns 404 (no pipeline was ever registered) | pipecat-manager + script-runner log |
| 21:13:09.46 | bin-ai-manager-1 | `EventPMPipecatcallTerminated`: grace 3s, no assistant reply row → "Backstop reply sent" (message 0606d0ed) → `ConversationV1MessageSend` with `backstopReplyText` (`bin-ai-manager/pkg/messagehandler/event.go:35,436-499`) | ai-manager log |

The observed webchat message 871c1af6 (tm_create 21:13:09.461) is that backstop send.

### 3.1 Where "No valid credentials provided." comes from, and which constructor raised it

The string exists only in pipecat's Google service constructors (pipecat-ai 1.4.x as vendored in `scripts/pipecat/venv`; line numbers differ between the two local venvs so functions are cited instead): `pipecat/services/google/tts.py` (`GoogleTTSService._create_client`, called from `GoogleTTSService.__init__`; the sibling class `GoogleHttpTTSService` has its own identical check but is not used by `run.py`, which imports and returns `GoogleTTSService` at `run.py:13,375`), `stt.py` (`GoogleSTTService.__init__`, inline), and the Vertex LLM variants (`llm_vertex.py`, `gemini_live/llm_vertex.py`). In all of them the check is `google.auth.default()` → on failure `raise ValueError("No valid credentials provided.")`, so the ADC lookup happens at construction time.

Constructors ruled out for this run:
- LLM: `run.py:493` builds the non-Vertex `GoogleLLMService(api_key=key or os.getenv("GOOGLE_API_KEY"))`, which only calls `genai.Client(api_key=...)` (`pipecat/services/google/llm.py`, `GoogleLLMService._create_client`) and never `google.auth.default()`. `GOOGLE_API_KEY` is present in the runner env. The empty `engine_key` on both member AIs (3.3) is therefore not the cause: the code falls back to the env key, and single-AI Gemini sessions with the same empty key succeed (3.4).
- Vertex: not referenced by `run.py`.

So the raise came from `create_tts_service("google")` or `create_stt_service("google")` inside the member loop (`run.py:602-611`); the loop creates LLM first (line 599), then TTS, then STT, so TTS is the first to fail for a member with `tts_type=google`.

The ~12 s between run_request and the failure is `google.auth.default()` probing the (absent) GCE metadata server before giving up.

Verified in the container:

```
$ docker exec voipbin-pipecat-script-runner-1 python -c "import google.auth; google.auth.default()"
DefaultCredentialsError: Your default credentials were not found.
```

Env in the runner: `DEEPGRAM_API_KEY XAI_API_KEY GOOGLE_API_KEY CARTESIA_API_KEY ELEVENLABS_API_KEY OPENAI_API_KEY` only. No `GOOGLE_APPLICATION_CREDENTIALS`, no mounts, no `~/.config/gcloud`.

### 3.2 Why the team path builds Google TTS/STT in a text-only session

`bin-pipecat-manager/scripts/pipecat/run.py`:

- `init_pipeline` (line 81): when `resolved_team` is present it calls `init_team_pipeline(id, resolved_team, stt_language, tts_language, llm_messages, vad_config, smart_turn_enabled)` and **drops the request-level `stt_type` / `tts_type` / `tts_voice_id`** (lines 97-107).
- `init_team_pipeline` (line 566): for every member it does `if ai.get("tts_type"): create_tts_service(...)` and `if ai.get("stt_type"): create_stt_service(...)` (lines 594-613) using the member AI's own voice config. There is no session-mode input to this function at all.
- `init_single_ai_pipeline` (line 128), by contrast, only creates STT/TTS when the request-level `stt_type` / `tts_type` is set (lines 157, 175). ai-manager sends none for conversation/task references, so single-AI text sessions never touch Google TTS/STT.

`bin-pipecat-manager/pkg/pipecatcallhandler/run.go:135-213` (`resolveTeamForPython`) copies each member AI's `TTSType`, `TTSVoiceID`, `STTType` into the payload (`run.go:196-206`) regardless of the pipecatcall's mode. `runner.go:173-188` sends both the request-level `pc.STTType`/`pc.TTSType` and `resolvedTeam` in the same `/run` request, so the runner already receives the mode signal it needs for the team branch; it just does not forward it.

Blast radius of `init_team_pipeline`: exactly one production caller (`run.py:98`) and 7 direct call sites in `test_init_pipeline.py` (lines 78, 104, 120, 176, 232, 288, 389). Downstream already tolerates a TTS-less/STT-less team pipeline: `run.py:621-670` builds the stage list conditionally and `team_flow.py:46-47,155-158` handles `routing_tts`/`routing_stt` being `None`.

### 3.3 Team configuration (production DB, read-only)

`ai_ais` for the two members of team d414dc93 (`voipbin admin helper`):

| AI | engine_model | tts_type | tts_voice_id | stt_type | engine_key set |
|---|---|---|---|---|---|
| voipbin Sales Assistant (74b3d982) | gemini.gemini-2.5-flash | google | en-US-Chirp3-HD-Gacrux | google | no (falls back to env `GOOGLE_API_KEY`, not a cause) |
| voipbin General Assistant (dfeb9e05) | gemini.gemini-2.5-flash | google | (empty) | google | no (same) |

The first member iterated triggers `create_tts_service("google")` → `GoogleTTSService()` → ADC lookup → `ValueError`.

### 3.4 Correlation across all recent runs (40 h window, both runners)

- 17 run_requests with `has_resolved_team=true`: **17/17** failed with `No valid credentials provided.` (all with `stt_type=null`, i.e. text mode).
- ~20 run_requests with `has_resolved_team=false` (Gemini and OpenAI, text mode): **0** validation failures.
- 0 voice run_requests (non-null `stt_type`/`tts_type`) since the containers started on 2026-09-05 04:31 UTC.

What this evidence supports: the team path, which constructs the members' Google TTS/STT services, fails 100% of the time in this environment, while the single-AI text path, which constructs no TTS/STT, succeeds with the same Gemini model, same env and same empty `engine_key`. The evidence does **not** separate "team + text" from "team, any mode": with zero voice samples, a voice team call with these members would fail identically today because the ADC lookup fails regardless of mode (secondary finding, section 4). That the text session should never have built TTS/STT is a code-based design argument (3.2), not an empirical one.

## 4. Root cause

**Primary (this ticket):** `init_team_pipeline` has no notion of session mode and instantiates per-member TTS/STT services from the member AI's voice configuration even when the pipecatcall is text-only. Because the team's members are configured with Google TTS/STT, construction fails at ADC lookup and the whole pipeline is rejected (HTTP 400) before any LLM turn runs; ai-manager's backstop then delivers the fallback sentence. Any team whose members use Google TTS/STT is unusable for webchat/SMS/conversation AI Talk in the current environment; teams whose members use other vendors still needlessly build TTS/STT services and the audio input transport for text sessions.

**Secondary (separate ticket):** the pipecat script runner on bm-nyc-01 has no Google Application Default Credentials, so `GoogleTTSService`/`GoogleSTTService` cannot be constructed for **any** pipeline, single-AI or team, voice or text. This affects voice AI calls whose AI has `tts_type=google` or `stt_type=google`, independent of this ticket. Not proven from logs (no voice run in the retained log window), but the ADC probe is conclusive about the container state. Open question for that ticket: whether Google TTS/STT ever worked on GKE (k8s `deployment.yml` sets only `GOOGLE_API_KEY`, no `serviceAccountName`/volume; credentials could have come from the node service account via the metadata server, or the feature may never have worked). That answer decides whether it is a migration regression or a never-supported configuration. Reach confirmation from code: `pipecatcall.TTSType`/`STTType` define no `google` constant (`models/pipecatcall/main.go:46-59`) while `ai.TTSType`/`STTType` do, and `getPipecatcallTTSInfo`/`getPipecatcallSTTType` cast the AI value unchecked (`start.go:759-777`), so a voice single-AI call whose AI has `tts_type=google` sends `tts_type="google"` at request level and hits the identical ADC failure in `create_tts_service`. The same unchecked cast also lets every other `ai.TTSType`/`STTType` vendor (`models/ai/main.go:247-268,303-307` define 21 non-empty TTS and 4 non-empty STT vendors; the runner implements only cartesia/elevenlabs/google TTS and deepgram/google STT at `run.py:346-407`, leaving 18 TTS and 2 STT vendors unsupported), failing with `Unsupported TTS/STT service`; the secondary ticket should scope that too. Suggested priority: High, because it silently breaks voice AI for any Google-TTS/STT AI, and VOIP-1351 covered credential materialisation only for api/storage/rag/transcribe-manager.

## 5. Why now / should we proceed (YES, high priority)

- Impact: voipbin.net's own public webchat AI (the product demo surface) answers every visitor with an apology. Same for any customer using a Team in a Message Flow.
- Fix shape: local to the pipecat runner request path; the design must give the team path an explicit session-mode signal and skip per-member TTS/STT (and the input transport) when the session is text-only, mirroring what the single-AI path already does. Existing pytest harnesses for the team pipeline exist to extend.
- Mode signal available today (verified across all three `PipecatV1PipecatcallStart` call sites in ai-manager, the only producers of pipecatcall requests):
  - voice AI call, `startPipecatcall` (`start.go:807-816`): request-level STT/TTS types are filled from the AIcall snapshot with **non-empty defaults** when the snapshot is empty (`getPipecatcallSTTType` → `defaultPipecatcallSTTType` = `deepgram`, `getPipecatcallTTSInfo` → `defaultPipecatcallTTSType` = `elevenlabs`; `start.go:759-777`, `main.go:95,99`). So for a voice call they are always non-empty.
  - AI task, `startPipecatcallTask` (`start.go:856-869`): `STTTypeNone`/`TTSTypeNone`.
  - listen turn, `startListenPipecatcall` (`listen.go:412-427`): `STTTypeNone`/`TTSTypeNone`, documented as "a listen turn has no audio legs at all". Listen turns run only on `contact_case` AIcalls (`listen.go:270-276`, `listen_trigger.go:219`, `listen_kind.go:58-66`); the underlying Case may point at a call, but the AIcall itself is never `ReferenceTypeCall`.
  - every non-call AIcall reference (`conversation` as in this bug, `task`, `contact_case`, empty): the gate at `start.go:811` is `c.ReferenceType == aicall.ReferenceTypeCall`, so `startPipecatcall` leaves both `None` for all of them.
  Invariant on main: **request-level `stt_type`/`tts_type` non-empty ⟺ the pipeline needs audio; both empty ⟺ text-only.** The single-AI path already relies on exactly this (`run.py:157,175`). pipecat-manager's own transport decision agrees on every path: only `aicall.ReferenceType == call` gets an Asterisk external-media leg (`pkg/pipecatcallhandler/start.go:197` vs the audio-less `default` branch at `:276`). Because the request struct uses `omitempty` (`pythonrunner.go:89,91`), empty types arrive in Python as `None`, so a team gate would be literally the same `if stt_type:` shape as the single-AI path. The invariant is implicit and unenforced, so the design may choose to make it explicit, but a presence gate is correct today.
- Two things the design must NOT do:
  - Do not use the request-level type **values** as the per-member TTS/STT type. For a team AIcall they are a create-time snapshot of the **start member's** AI (`start.go:67-77` `resolveAI`; `db.go:52-54`), while members are heterogeneous (3.3). Presence is the mode flag; per-member values stay per-member.
  - Do not re-derive the mode from a reference type inside pipecat-manager or the runner. `pipecatcall.ReferenceType` is always `ai_call` for AI-driven sessions (`models/pipecatcall/main.go:34-39`; all three call sites pass it), so it carries no mode. The AIcall's `ReferenceType` does correlate with mode today (`call` ⟺ audio, identical to the presence gate on every current path), but it is not part of the `/run` payload, and re-deriving it in a second service duplicates a policy ai-manager already computed when it filled or blanked the request-level types; the two would drift the next time ai-manager adds an audio-less start path. Consume the signal ai-manager already sends.
- Behaviour matrix for design: {text session (conversation / task / contact_case listen turn: request-level types empty), voice call (request-level types non-empty)} × {members with google TTS/STT, members mixed vendors, start member without TTS but another member with TTS}. Every text-session row must build no TTS/STT and no input transport, while the output transport stays unconditional on both paths (`run.py:194-201,235` single; `run.py:658,670` team) because text sessions deliver the LLM reply through it; every voice row must keep today's per-member TTS/STT from member config.
- Dependencies/risks: none on other in-flight work. Voice team calls must keep creating per-member TTS/STT in voice mode.
- Alternatives considered: (a) configure Google ADC in the runner only. Unblocks this case but leaves TTS/STT being built for text sessions and is an infra-level change (secondary ticket). (b) the Go side already blanks the request-level types for non-audio sessions; the remaining gap is the per-member payload in `resolveTeamForPython` (`run.go:196-205`) plus the absent Python-side gate. Design phase decides between: Python presence gate in `init_team_pipeline` (mirrors the single-AI path, smallest change), Go-side stripping of member TTS/STT when the request-level types are empty, an explicit mode field on the `/run` request, or a combination.
- Latency note (not a blocker): between the runner's 400 (21:12:48) and `terminate` (21:13:06) ~18 s pass, so the user waits ~33 s for the fallback. `RunnerStart` only logs and returns on script-start failure (`runner.go:86-89`). Possible follow-up ticket, not in scope.

## 6. Next steps

1. Create Jira ticket (VOIP, Bug): summary in English, body in Korean; link VOIP-1457 (relates to); create the secondary ticket (Google ADC missing in pipecat runner) with the open question recorded.
2. Design: choose where the mode signal is computed and how it is transported; write the behaviour matrix above; confirm the voice team path is unchanged.
3. Plan → TDD in `scripts/pipecat` (failing tests: team text session (request-level types empty) with google-TTS/STT members must not call `create_tts_service`/`create_stt_service` and must build no input transport; voice team session (request-level types non-empty) still builds per-member TTS/STT from member config, including when the start member lacks TTS but another member has it; a contact_case listen turn on a team (request-level types empty) behaves like the text row).
4. Deploy, verify on voipbin.net webchat (visitor inbound via widget token), re-test VOIP-1457, close tickets.
