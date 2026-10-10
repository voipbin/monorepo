# VOIP-1579: Call recording `out` file must include injected audio (speakings direction=out, ai_talk)

Status: DRAFT v5.5, design review closed (rounds 8 and 9 both APPROVED; round-1..9 reviews responded, see section 8). v4 removed speculative machinery; v5 restored one small cleanup guard (4.4); v5.2 corrected the conference-join statement and added observability; v5.3 limits the new path to platform-internal ExternalMedia; v5.4 assigns files to PRs and fixes factual wording.
Ticket: VOIP-1579. Sibling (separate, not in scope): VOIP-1580 (action timeout precision).

## 1. Issue analysis (validity, evidence)

### 1.1 Symptom
`POST /calls/{id}/recording_start` produces `in` and `out` files. The `out` file is silent for audio that call-manager's call-type external media injects with a speak direction `out` (speakings with `direction=out`, pipecat `ai_talk`). Flow `talk`/`play` (ARI channel playback) is recorded correctly. Product doc `bin-api-manager/docsdev/source/recording_overview.rst:50` defines a call recording as "both directions mixed", so a recording without the AI/TTS side contradicts the documented behaviour (customer impact: contact-center QA/compliance recordings of AI calls hold only the customer).

### 1.2 Evidence (production observations of 2026-10-09 on api.voipbin.net, own validator customer, virtual numbers only; call ids and levels were measured with the skill templates and cannot be re-read by a reviewer, the procedure is reproducible)

| Scenario | Remote party heard | `out` recording |
|---|---|---|
| Flow `talk` (call 75e3f45b) | yes (RMS 2.5k-4.8k) | yes, same level |
| speakings TTS (call faa2d6e1, earlier session) | yes (RMS ~5k) | RMS 0 everywhere |
| `ai_talk` pipecat (new AI, Chirp3-HD voice) | yes (RMS 1.4k-3.8k, whole call) | RMS 0 everywhere |

`in` always matched what the test UA sent.

### 1.3 Root cause
- Asterisk `main/audiohook.c`, `audio_audiohook_write_list` (tag 23.5.0: spy queueing at line 1229, whisper mixing at 1248): spies receive the write-direction frame BEFORE whisper audio is mixed in. A spy on a channel can never see whisper audio on the same channel. Same ordering in master.
- Recording = two snoop channels (`spy=in`, `spy=out`) on the call channel: `bin-call-manager/pkg/recordinghandler/recording.go:73-98`.
- Call-type external media = snoop channel `spy=<listen>`, `whisper=<speak>` plus the ExternalMedia channel in a private "snoop bridge": `bin-call-manager/pkg/externalmediahandler/start.go:104-112`.
- Whisper frame supply is not a factor: Asterisk 23.5.0 has a whisper timer (`AST_WHISPER_TIMER_SRC`) that generates idle frames, and local tests showed noise on the original channel makes the spy record the noise but still not the whisper. (CEO confirmed 2026-10-09 that this needs no further action.)
- The production image is Asterisk 23.5.0: `voipbin/voip-asterisk-call:1787b415...` (the tag running on the production hosts, verified with `asterisk -V` inside the pulled image).

### 1.4 Experiments that fix the direction (call channel in a `mixing,video_sfu` bridge, the type `addCallBridge` creates)
Tones attribute each source by frequency. Harness: skill `voipbin-external-live-call-control` (templates `local_23_5_websocket_extmedia_*`, `local_ari_*`, `tone_analyzer.py`; production-image recipe in `references/local-asterisk-23-5-harness.md`). The websocket channel reported state `Up` after the join in every connected run. Not measured: the call hangup cascade (bridge destroy -> ExternalMedia leaves) and the Destroy-failure case; they are covered by 4.4 and live check 5.

| Injection path | Asterisk | Remote hears | spy=out recording |
|---|---|---|---|
| ARI channel play on the call channel | master | yes | yes |
| ARI bridge play in the call bridge | master | yes | yes |
| snoop whisper (current behaviour) | master | yes | NO |
| RTP ExternalMedia channel in the call bridge | master | yes | yes |
| WebSocket server-mode ExternalMedia (`encapsulation=none, transport=websocket, connection_type=server`) slin16 in the call bridge, recording from the start | 23.5.0 production image | yes | YES |
| same, ulaw format, recording started later | 23.5.0 | yes | YES |
| same, echo check: audio the ExternalMedia client receives holds only the remote party's tone, never its own 2 kHz (softmix excludes a member's own audio; other injectors in the same bridge are heard, see 4.1) | 23.5.0 | n/a | n/a |
| same, `FLUSH_MEDIA` with 8 s of audio queued at once, flush after 1 s | 23.5.0 | tone stops at the flush (~1 s heard), recording matches | YES |
| ExternalMedia created but the client never connects | 23.5.0 | n/a | the channel stays `Down`, is not in Stasis, and `addChannel` to the call bridge returns 422; it never becomes a bridge member, the call is unaffected |
| client disconnects after joining | 23.5.0 | n/a | Asterisk destroys the ExternalMedia channel at once (404); the call bridge keeps only the call channel |

The websocket rows use the transport options of `bin-pipecat-manager/pkg/pipecatcallhandler/main.go:55-60` and `bin-tts-manager/pkg/streaminghandler/main.go:59-61`. Conclusion: an ExternalMedia channel that is a member of the call bridge records into `out`, receives the remote party's `in` audio without a snoop, and barge-in flush still works.

### 1.5 Verdict
Valid, customer-impacting defect (documented behaviour violated), rooted in an Asterisk ordering rule so no config-only fix exists. Not a duplicate. Proceed, on the condition that the cleanup (4.4) and rollout (4.6) risks of the new topology are handled as designed.

## 2. Callers of `CallV1ExternalMediaStart` (verified in code)

| Caller | listen / speak | Path after this change |
|---|---|---|
| pipecat `pipecatcallhandler/start.go:111` and `:232` | In / Out | bridge path |
| tts-manager speakings `streaminghandler/start.go:106-107` | None / `st.Direction` (API request: "", in, out, both; default none) | bridge path only when direction=out; otherwise unchanged |
| transcribe-manager `streaminghandler/start.go:56-68` | request direction (flow/api default Both) / None | unchanged (legacy) |
| api-manager stream `streamhandler/start.go:37-50` | Both / Both | unchanged (legacy); NOT fixed here (non-goal) |
| call-manager flow action `external_media_start` and API `POST /v1/calls/{id}/external-media` (`callhandler/action.go`, `v1_calls.go`) | customer options, customer-supplied `external_host` | unchanged (legacy): the new path is limited to the platform-internal host `INCOMING` (4.1) |

## 3. Goals / non-goals

Goals
1. `out` recording contains audio injected by speakings (direction=out) and pipecat ai_talk when the call is a plain single call (eligibility in 4.1).
2. Same API and events; callers unchanged.
3. No change to `in` recording, confbridge external media, Flow `talk`/`play`. (Transcribe with `out`/`both` intentionally also hears injected audio, see 4.9.)

Non-goals
- Customer-supplied external hosts (flow action `external_media_start`, `/calls/{id}/external-media`) and api-manager streams (Both/Both): remain unrecorded in `out`; needs a different design (a bridge member cannot hear its own output); separate ticket if wanted.
- Calls that are conference, connect or groupcall participants, or already have a bridge-path injector: legacy path, recording gap stays, documented.
- Changes in Asterisk, tts-manager, pipecat-manager, transcribe-manager, api-manager.
- VOIP-1580.

## 4. Design (minimal)

Four small code changes (selection rule, new path, leave branch and cleanup guard, join log) plus tests and one doc note. Everything else considered in earlier drafts was removed because no measured harm backs it (project rule: no speculative expansion; triggers are listed in 4.8).

### 4.1 Selection rule, decided from the call record only
Implemented as a pure function `isCallBridgeEligible(c *call.Call, externalHost string, listen, speak externalmedia.Direction) (bool, string)` (the string is the first failed condition, empty when eligible) in `externalmediahandler/start.go` (no mocks needed to test it). `startReferenceTypeCall` keeps its current order: `CallV1CallGet(callID)`, then `channelHandler.Get(c.ChannelID)` (this is where `AsteriskID` comes from, as today), then it calls the function once with the loaded `c`; the function evaluates the cheap direction test first, then the call fields:

```
useCallBridge :=
     directionSpeak == DirectionOut
  && (directionListen == DirectionNone || directionListen == DirectionIn)
  && c.Status == call.StatusProgressing
  && c.BridgeID != ""
  && c.ConfbridgeID == uuid.Nil           // not in a conference
  && c.MasterCallID == uuid.Nil           // not the slave leg of a connect
  && len(c.ChainedCallIDs) == 0           // not the master leg of a connect
  && c.GroupcallID == uuid.Nil            // not a groupcall leg
  && externalHost == "INCOMING"           // platform-internal consumers only (constant externalHostIncoming)
```
Why `externalHost == "INCOMING"`: pipecat, tts-manager and transcribe all pass this literal (`pipecatcallhandler/start.go:116,237`, `streaminghandler/start.go:100` in tts, `:61` in transcribe) for the platform websocket server mode where the consumer is a platform service reached over the internal network. On the bridge path an ExternalMedia channel receives the other bridge members' audio even with `listen=none`; for platform services that is acceptable (tts-manager drains and discards inbound frames, `runWebSocketRead`), but for a customer-supplied host (flow action; `/calls/{id}/external-media` is internal RPC only, no caller today) it would send the remote party's audio to the customer endpoint although `direction_listen` says none, a contract and privacy change. So customer-supplied hosts keep the legacy path. Other transport combinations are rejected by Asterisk. A customer who passes the literal `INCOMING` gets a server-mode websocket reachable only on the internal network, so this is no loophole (`MediaURI` is `ws://<asterisk internal ip>:port/media/<id>`, an existing behaviour). The literal lives in four services while call-manager would hold the constant; if a caller changes it the path silently falls back to legacy, visible in the path log.
Anything else takes the unchanged legacy snoop path (including `dialing` calls and `early_execution` flows that run while the call is still `ringing`; those remain unrecorded on `out` and are out of scope here). `c.ExternalMediaIDs` and the reference-id cache key are NOT used: the former is filled only by the `/v1/calls/{id}/external-media` request path and the flow action (the only `CallAddExternalMediaID` caller is `callhandler/external_media.go`), the latter is a single overwritten key that other records' updates and deletes also rewrite (`cachehandler/handler.go:261-297`), so neither can identify "an injector already exists" reliably.
`addCallBridge` creates the call bridge as `mixing,video_sfu` (`callhandler/start.go:461`), the same type the experiments used.

Accepted behaviours (bounded, documented, trigger-based follow-up, not locked):
- Several injectors on one call (for example ai_talk plus speakings) all join the call bridge and hear each other in softmix. Not seen in known flows.
- `ConfbridgeID`, `MasterCallID` and `ChainedCallIDs` are set after the call/join channel is created (`confbridgehandler/joined.go`, chained-call linking), so an injector started in that short window is treated as eligible; a conference joined while an injector runs hears it and is heard by it. This is NOT guaranteed to be rare: the AI `connect` tool adds the `connect` action to the flow and terminates the AI call asynchronously (`bin-ai-manager` `toolHandleConnect`), so the later `confbridge_join` can overlap pipecat's `ExternalMediaStop`. (Flow `external_media_start` is no longer on the new path, so the intentional long-lived case does not exist for it.) Customer-visible effect: the joined party can hear the injected audio and the injector can hear the party. The legacy whisper path did not expose the joined party. Decision: accept now, make it observable (`Warnf` below), and treat the first occurrence as the trigger for the guard 'a TypeJoin entering a call bridge that holds external members hangs those members up' (a guard that ends platform injectors at join; since customer hosts stay legacy it does not change customer-visible media contracts). Open for the CEO: add that guard now instead? Recommendation: wait for the first logged occurrence.
- Recovery (VOIP-1556) happens after the Asterisk that held the call is lost; the old bridge and anything in it disappear with it, as the old snoop did. `recoverySkipReason` skips on `ExternalMediaIDs`, which pipecat/tts do not fill; that asymmetry already exists today and is unchanged.

### 4.2 New path
`startReferenceTypeCallBridge(ctx, id, c, ch, ...)` calls `h.startExternalMedia(...)` with `ch.AsteriskID` and `bridgeID = c.BridgeID`. If `startExternalMedia` fails there is nothing to clean up (no snoop channel, no new bridge were created) and the call bridge is never destroyed on this path. The existing `startContextExternalMedia` already joins the ExternalMedia channel to `em.BridgeID` (a websocket server-mode channel enters Stasis, and can be joined, only after the client connected; section 1.4). No snoop, no new bridge. One log line (`Infof`: call id, chosen path `call_bridge`|`snoop`, and for `snoop` the first failed condition) states the chosen path, so a fallback is visible; no metric or label change (the existing counter is incremented before dispatch and the Grafana dashboard queries it).

### 4.2a Observability
`startContextJoinCall` (`callhandler/start.go:215`, TypeJoin entering the call's `BridgeID`) logs `Warnf` (call id, bridge id, ids of members whose channel type is `TypeExternal`) when the call bridge already holds external members; no behaviour change. Placement: after the existing `ChannelJoin` has succeeded, best effort: `bridgeHandler.Get` (which can wait for an existing record) or `channelHandler.Get` errors are ignored (debug log) and never change the return value; the bridge record lists ids only, so each member is looked up with `channelHandler.Get` to read its type. The bridge membership view is event-driven, so a member that joined an instant earlier may be missing: the signal is best effort, not complete. The leave branch (4.3), the guard (4.4) and the path line (4.2) log call id and the external channel id (the 4.3 leave-branch line is Debug level, since the guard and 4.3 both hang up the same channel on normal call end). These logs are the signals for the triggers in 4.8; the guard also fires on every normal hangup, so its log is not a Destroy-failure signal (that is the existing `Errorf` in `Hangup`).

### 4.3 Leave handling (required)
`callhandler.bridgeLeftExternal` (`pkg/callhandler/bridge.go:57-82`) hangs up an external channel and, when channels remain, kicks all of them (`removeAllChannelsInBridge`). Correct for the private snoop bridge, destructive for the call bridge (the call channel would be removed from its own bridge when the ExternalMedia channel stops or the client disconnects). New first branch inside `bridgeLeftExternal`: if `br.ReferenceType == bridge.ReferenceTypeCall`, only `HangingUp(cn.ID)` and return; log the bridge id and the number of remaining channels. No `Destroy` even when no channel remains, and no kick (the call bridge is owned by `Hangup`; if its `Destroy` failed an empty Asterisk bridge may remain, which is part of the accepted residual of that failure). Snoop bridge behaviour is untouched; existing tests build bridges with an empty `ReferenceType` and keep passing. This branch is unreachable before the selection rule ships, so it is safe to deploy alone (4.6).

### 4.4 Cleanup when the call ends (small guard replacing the legacy cascade)
Legacy chain: the call channel dies, its snoop channel dies, `bridgeLeftExternal` hangs up the ExternalMedia channel in the private snoop bridge, the websocket closes. With the call-bridge path that cascade is gone and the only thing removing the channel is `Hangup`'s `bridgeHandler.Destroy(c.BridgeID)`, whose error is only logged, and `Hangup` can return before it (`CallGetByChannelID` failure, `TMHangup != nil`). pipecat's own teardown does not cover it: `terminateReferenceTypeCall` skips `ExternalMediaStop` when the call is no longer progressing, and tts-manager has no call-hangup subscription; ai-manager does subscribe to call hangup and terminates the ai_talk pipecat session (including `ExternalMediaStop`) as a second path, so only speakings depend solely on call-manager. The guard is still needed because call-manager must not rely on other services for its own cleanup. So a failed `Destroy` would leave a live WebSocket channel (and a pipecat session with LLM/STT/TTS cost) behind, which legacy did not.
Guard: in `callHandler.ARIChannelLeftBridge`, the `channel.TypeCall` case (currently `return nil`) additionally runs, when `br.ReferenceType == bridge.ReferenceTypeCall`, `hangupExternalMembers(br)`: for each id in `br.ChannelIDs` (the view after `RemoveChannelID`), `channelHandler.Get`, and if `Type == channel.TypeExternal` then `HangingUp(id)` (a `channelHandler.Get` or `HangingUp` error is logged and the loop continues with the next member; errors are never returned). `channelHandler.Get` can wait up to its existence timeout for a record that does not exist; members come from the bridge record so they exist in normal operation. The ExternalMedia hangup then triggers its own `ChannelLeftBridge`, which 4.3 handles. This restores the legacy property "call channel gone implies injector gone" independent of `Destroy`. Like 4.3 it does nothing before the selection rule ships (no external member exists in a call bridge), so it can be part of the guard PR.
Race: `br.ChannelIDs` is filled by `ChannelEnteredBridge`, so an ExternalMedia channel that joined an instant before the call ended may not be listed yet; then `Hangup`'s `Destroy` and 4.3 are the remaining cleanup (accepted, only with a simultaneous `Destroy` failure). The call channel leaves a call bridge at call end and, mid-call, when `echo`/`stream_echo` hand it to the dialplan with `channelHandler.Continue` (`callhandler/action.go:355,435`); there the guard also ends the injector, which is the desired result because the call is no longer in Stasis. Otherwise it does not leave (`ChannelKick` is used only by `removeAllChannelsInBridge`, which only external-channel leaves trigger; confbridge join adds a TypeJoin channel without moving the call channel; recovery builds a new bridge).
Residual (same as legacy): ExternalMedia records are not deleted by this cascade; pipecat/tts run their own `Stop` when their websocket closes, a flow-action started record may stay on the call record.
Other situations: client gone (Asterisk destroys the channel at once, section 1.4); client never connects (the channel never reaches Stasis and never joins, 422, call unaffected; pipecat/tts already call `Stop` on connect failure); join failure (the Stasis handler already hangs the channel up; a stale record stays in the 24 h cache keyed by call id and never read again; same as legacy).

### 4.5 Readers of call-bridge membership (audit, grep over the monorepo excluding vendor, mocks, tests, worktrees)
`callhandler/arievent.go` `answerCallBridgePeers`, `confbridgehandler/ari_event.go` (TypeJoin Up peer answer), `callhandler/bridge.go` `bridgeLeftExternal`/`removeAllChannelsInBridge`. The answer loops skip Up channels; an external channel joins only after its client connected and was `Up` in every connected run of section 1.4. `callhandler` has no `bridgeHandler.Play` caller (only `channelHandler.Play`). Every `ChannelJoin` caller: `callhandler/start.go:165` (`startContextExternalSoop`, snoop bridge), `:193` (external media, `em.BridgeID`), `:215` (TypeJoin channel into the call's `BridgeID`), `:470` (`addCallBridge`, also used for the recovery channel), `confbridgehandler/start.go:48` (confbridge bridge). No other path adds a member to a call bridge. The recording snoops are not bridge members. Unit tests cover both loops with an external member.

### 4.6 Rollout safety (decision pending with the CEO)
Mixed versions are the hazard. Old `bridgeLeftExternal` ignores the bridge type and kicks every remaining channel. Two replicas compete on one RabbitMQ queue for ARI events (`subscribehandler`, `QueueNameCallSubscribe`). If a new pod puts an ExternalMedia channel into a call bridge and an old pod handles its `ChannelLeftBridge`, the call channel is removed from its bridge (the call stays up in the database but has no media path; not measured). Rollback has the same exposure.
Options: (1) two PRs: PR-A = 4.3 and 4.4 (with their tests), deployed and confirmed on all replicas; PR-B = 4.1, 4.2, docs, live verification. PR-A is safe alone because no ExternalMedia channel can be in a call bridge before PR-B (4.3 and 4.4 are unreachable until then), and it changes no snoop behaviour (snoop bridges are `ReferenceTypeCallSnoop`). (2) one PR with a default-off switch; weaker than it looks, because enabling it still needs the guard on every replica first, so it does not replace (1). (3) one PR, accept a window of minutes, deploy at a quiet hour, watch logs. Rollback order: revert PR-B first, let open calls finish, then PR-A (reverting only PR-A restores the old kick risk). Confirming PR-A on all replicas: `ListStacks` image SHA per call-manager replica equals the PR-A merge commit (`voipbin-prod-readonly-inspection` skill). Recommendation: (1). Needs the CEO's explicit approval because the project rules forbid splitting PRs and adding switches on my own.

### 4.7 Concrete change list
| File | Change |
|---|---|
| `bin-call-manager/pkg/externalmediahandler/start.go` | pure `isCallBridgeEligible`, branch in `startReferenceTypeCall`, new `startReferenceTypeCallBridge`, log line |
| `bin-call-manager/pkg/callhandler/bridge.go` | `bridgeLeftExternal`: call-bridge branch (4.3); new `hangupExternalMembers` (4.4) |
| `bin-call-manager/pkg/callhandler/start.go` | `startContextJoinCall`: log-only warning when the call bridge holds external members (4.2a); needs `bridgeHandler.Get` (already a field) |
| `bin-call-manager/pkg/callhandler/arievent.go` | `ARIChannelLeftBridge`: `TypeCall` case calls `hangupExternalMembers` for call bridges |
| tests | `externalmediahandler/start_test.go` (matrix, new path, legacy regression), `callhandler/bridge_test.go` (call-bridge leave, snoop unchanged, `hangupExternalMembers`), `callhandler/start_test.go` (join with and without external members; log-only, no behaviour change), `callhandler/arievent_test.go` (new `ARIChannelLeftBridge` cases for TypeCall and TypeExternal, plus answer-loop cases); `start_test.go` also covers the `bridgeHandler.Get` failure being ignored, `confbridgehandler/ari_event_test.go` (answer loop with an external member) |
| `bin-api-manager/docsdev/source/recording_overview.rst` | states, next to the existing sentence that calls a call recording "both directions mixed": for plain single calls only, `out` contains injected AI/TTS audio (speakings with direction out, ai_talk); not for conference, connect or groupcall legs, `dialing`/early-execution flows, or api-manager streams; transcribe (streaming and recording-based) `out`/`both` also transcribes injected audio; earlier recordings stay unchanged. Follow the repo's RST docs sync rule in CLAUDE.md (clean rebuild, `git add -f` of the tracked build output) |
No schema, API, mock-interface, metric or config change. Existing `Test_Hangup` and `join` tests are untouched because those functions are not modified.

### 4.7a PR assignment (when split, see 4.6; with a single PR everything ships together)
- PR-A (safe alone, unreachable before PR-B): `bridge.go` (4.3), `arievent.go` (4.4) and their tests (`bridge_test.go`, `arievent_test.go`), plus the `confbridgehandler` answer-loop test with an external member.
- PR-B: `externalmediahandler/start.go` (4.1, 4.2) and `start_test.go`, `callhandler/start.go` (4.2a log) and `callhandler/start_test.go`, the RST change, live verification.

### 4.8 Considered and removed (with triggers)
| Removed | Why | Trigger to add |
|---|---|---|
| "first injector wins" via reference-id lookup | the key is overwritten and deleted by other records, stale after crashes, NotFound surfaces as `redis.Nil` | two concurrent injectors observed |
| join-failure `Stop` | no harm observed, legacy has the same residual | leaked records or channels observed |
| `Hangup` cleanup of external media (reference lookup + `Stop`) | adds a Redis read to every hangup and breaks all `Test_Hangup` mocks; replaced by the event-driven guard in 4.4 | guard proves insufficient (orphan WebSocket channel after calls ended) |
| conference-join guard (hang up injectors when a TypeJoin enters) | needs a product decision; the log in 4.2a promotes it when it fires | first logged occurrence in 4.2a |
| metric label | counter incremented before dispatch, dashboard dependency | fallback rate becomes a question |

### 4.9 Intentional behaviour changes
- Transcribe with `out`/`both` on the same call now also transcribes injected AI/TTS audio (it records the call channel's write side, which now contains it). Recording-based transcription (`transcripthandler/recording.go`) transcribes the `out` file as `DirectionOut`, so for eligible calls it now contains the AI/TTS speech where it used to be empty; downstream consumers of that transcript (for example summaries) may now see the AI turns twice if they also use the AI's own transcript. Not changed here; noted in the docs.
- Existing recordings are not changed (no retroactive fix); billing is unaffected (recordings are billed on start/finish timestamps, not content).
- Recordings that were silent on `out` now contain injected audio; `in` unchanged.

### 4.10 PR code-review starting points
- `isCallBridgeEligible` is pure, evaluates the direction test first, and the existing `(Both,Both)` test is unchanged.
- The new path has no `bridgeHandler.Start`, `bridgeHandler.Destroy` or `StartSnoop`.
- In `bridgeLeftExternal` the call-bridge branch precedes the hangup/destroy/kick logic and applies only to `ReferenceTypeCall`.
- `hangupExternalMembers` never returns an error and a failing member does not stop the loop.
- `startContextJoinCall` change is log-only.
- RST updated, clean rebuild, tracked build output added with `git add -f`.
- Transcribe behaviour change (4.9) is in the docs.

## 5. Test plan

Unit (gomock, repo conventions: `Test_` names, controller `mc`; read `docs/conventions/testing.md` before writing, `scripts/check-test-conventions.sh` checks changed lines). Honest limit: unit tests prove branch selection and call order only; "out is recorded" and "the call survives" are proven only by the live checks.
- `Test_isCallBridgeEligible` (pure table test, no mocks): all 16 (listen x speak) combos on an eligible call, plus one `(None,Out)` row for each ineligibility reason (status not progressing incl. dialing, empty BridgeID, ConfbridgeID, MasterCallID, ChainedCallIDs, GroupcallID set, `externalHost` other than `INCOMING`). `ineligible (None,Out)` handler test uses a customer host.
- handler level (gomock, two tests only): new path (mock order `CallV1CallGet`, `channelHandler.Get`, `UUIDCreate`, external media create, `StartExternalMedia`; no `bridgeHandler.Start`, no `StartSnoop`; bridge argument `c.BridgeID`) and ineligible `(None,Out)` call taking the legacy snoop path; the existing `(Both,Both)` test stays unchanged.
- `bridgeLeftExternal`: ReferenceTypeCall + remaining call channel -> hangup only, no kick, no destroy; ReferenceTypeCallSnoop unchanged (existing cases).
- `ARIChannelLeftBridge` is new test territory (no existing test in `callhandler`): TypeCall + ReferenceTypeCall bridge holding an external channel -> `HangingUp` on it; only non-external members -> no call; no remaining channels -> no call; `channelHandler.Get` error on one member -> the next member is still handled; call channel leaving mid-call (echo/Continue) -> same handling; TypeCall + snoop bridge -> no action; TypeExternal + call bridge -> hangup only, no kick, no destroy even with zero remaining channels.
- `answerCallBridgePeers` and the confbridge answer loop with an external member.

Live, production after deploy (PR-B if split), own validator customer, virtual number + SIP UA from the skill templates, no PSTN. Criteria are numeric and computed per 1-second window with the template analyzer (energy at the tone frequency, or RMS for speech):
1. ai_talk with `recording_start`: in the windows where the UA received AI audio (UA RMS > 500), the out file RMS > 500 in the same windows (offset tolerance 1 s); in file unchanged; AI answers the UA's speech.
2. speakings `direction=out`: same. `direction=""` stays unrecorded.
3. Call survival: after the AI session ends (or external media is stopped via API) the call stays up for 10 more seconds, the UA still receives a Flow `talk` that follows, and the `out` recording keeps recording it.
4. Barge-in on a pipecat call stops speech immediately. Hold/MOH and mute-out on an ai_talk call: record what the UA hears and what the out file holds (read-only commands only, no `channel request hangup`; unmeasured; whisper vs bridge-injected audio may differ under `MuteOn`/`HoldOn`); any difference is reported, not assumed.
5. Cleanup: after hangup (repeat 5 times, including immediate hangup after ai_talk start) `core show channels` on the call Asterisk (ssh, `docker exec`, production read access as in the `voipbin-prod-readonly-inspection` skill) shows no leftover WebSocket channel for the call; the call-manager log shows the chosen-path line for every tested call and the leave-branch line with remaining channels. A `Destroy` failure cannot be injected in production; it is covered by the unit test of 4.4 and the local harness.
6. Regression: transcribe with default direction still produces transcripts; api-manager stream still works.
7. If the single-PR option is chosen: mixed-version window with one old and one new replica, ai_talk start/stop repeated, count kicks.
Cleanup checklist: temporary AI, virtual number, flow, recordings. Cost: about 40 s of LLM/STT/TTS per scenario.

## 6. Risks and trade-offs
- Topology change for pipecat/speakings: eligibility defaults to the legacy path, proof on the production Asterisk image, live checklist. Rollback = revert, but see 4.6 for the mixed-version hazard.
- Bridge members hear each other: only the call channel is present for an eligible call (4.1); later conference join is observable (4.7).
- ExternalMedia now also receives the call's `in` audio when listen=none (speakings). tts-manager drains inbound websocket messages (`runWebSocketRead`, `websocket.go:113`), so no backpressure and no use of the audio; customer-supplied hosts are excluded from the new path (4.1).
- Bridge-level ARI playback into the call bridge would be heard by the injector's listener (a snoop spy=in did not hear it). Verified by grep: `callhandler` has no `bridgeHandler.Play` caller.

## 7. Open points
- Blocking: the CEO's rollout decision (4.6).
- Not blocking: api-manager Both/Both streams remain a documented gap; concurrent injectors (4.1).

## 8. Review history
Section numbers cited in section 8 are those of the draft being reviewed (v1-v4) and do not map to the current numbering; the current design is sections 1-7.

### Round-1 review response summary (2026-10-10)
| Finding | Resolution |
|---|---|
| B1 evidence not representative (RTP vs websocket slin16, version) | Re-ran on the production Asterisk 23.5.0 image: websocket server mode, slin16 and ulaw, late recording, echo, FLUSH_MEDIA (1.4) |
| B2 cleanup depends on Destroy; join failure leaks | 4.3 and 4.5 added |
| M1-M3, M4/M5 callers table, goal vs Both/Both | section 2 rewritten, api streams moved to non-goals |
| M3 empty bridge / TOCTOU / DB view | rule uses synchronous call-record fields (4.1), no bridge lookup |
| M6 echo to STT | measured: none (1.4) |
| M7 / m8 leave branch semantics | 4.4: no destroy, no kick, position inside bridgeLeftExternal |
| m9 TMDelete / 3 s wait | removed with the bridge lookup |
| m11 conference after start | 4.7 policy |
| m13 mute/DTMF | live checklist item 4 |
| m14 test gaps, short-circuit order | 4.1 and 5 |
| 3.5 audit incomplete, connect/groupcall wording | 4.6 filled and corrected |
| m12 line citations | fixed (`bridge.go:57-82`) |

### Round-2 review response summary (2026-10-10)
| Finding | Resolution |
|---|---|
| B1 `c.ExternalMediaIDs` not filled by pipecat/tts/transcribe/api | no use of it anywhere; record lookup by reference id (4.1, 4.5, 4.7); live check changed |
| B2 `ConfbridgeID` set asynchronously | accepted race documented with bounds (4.1) |
| B3 `MasterCallID` missing | added (4.1) |
| M1 4.3 premise | corrected: channel already hung up by the Stasis handler, record cleanup only; scope stated |
| M2 duplicate Stop | NotFound ignored, 4.5 reworded as belt and braces |
| M3 file list | 4.10 |
| M4 recovery | recovery follows Asterisk loss; stale record has a different BridgeID (4.1) |
| R2-B2 rolling deploy kick | 4.9, CEO decision requested; split plan prepared |
| R2-M3 client gone/never connected | experiments added to 1.4: never-connected channel cannot join (422) and stays Down; disconnect destroys the channel |
| R2-M4 conference after start | 4.7 policy using record lookup |
| R2-M5 transcribe behaviour | 4.8 documented as intentional |
| R2-M6 logic leaps | experiments added (1.4); end-of-life behaviour covered by 4.4 tests and live item 5 |
| R2 minor: stale record, id reuse, metrics, MOH, mute | partly accepted/documented (4.1, 4.8), path log and metric label (4.2), mute expectation (live 4) |

### Round-3 review response summary (2026-10-10)
| Finding | Resolution |
|---|---|
| B (reviewer 1) NotFound is `redis.Nil`, not `ErrNotFound`; M1 reference key rewritten/deleted by other records | the reference lookup is removed entirely (4.1, 4.8) |
| B2 (reviewer 2) stale reference record disables the new path mid-call | same removal |
| M2/M3 (reviewer 1) Hangup change breaks all `Test_Hangup`, runs for every hangup | Hangup cleanup removed (4.4, 4.8); no existing test touched |
| M1 over-engineering (reviewer 2): 4.3, 4.5, 4.7, metric label | all removed with explicit triggers (4.8); log line kept |
| M2 rollout wording, PR-A independence, flag is not a substitute | 4.6 rewritten with the facts and a recommendation |
| M3 recovery asymmetry | stated as an existing, unchanged fact (4.1) |
| M4 call survival and numeric live criteria, Redis NotFound check | live items 1-3 rewritten; Redis check dropped with the lookup |
| m3/m4 mock interface, docs for transcribe change | 4.7 |
| m1 status contradictions | section 7 rewritten; status line updated |

### Round-4 review response summary (2026-10-10)
| Finding | Resolution |
|---|---|
| M (both reviewers) cleanup claim false: pipecat/tts do not close the socket on call hangup; failed Destroy leaves a live channel, unlike legacy | 4.4 rewritten with the real chain and a small event-driven guard (call channel leaves call bridge -> hang up external members) |
| M early_execution / dialing calls fall back silently | stated as out of scope (4.1) and path log checked in live check 5 |
| M production-image websocket harness not saved | saved as templates and `references/local-asterisk-23-5-harness.md` (1.4) |
| m `MasterCallID`/`ChainedCallIDs` race | added to 4.1 |
| m Up state evidence | wording tied to the observed runs (1.4, 4.5) |
| m wrong cross references, numbering gap, history numbers | fixed; history note added |
| m ChannelJoin caller list, missing-bridge-record test, Destroy failure not injectable, leave log fields, 1.5 caveat | added (4.3, 4.5, 4.7, live 5, 1.5) |

### Round-5 and round-6 review response summary (2026-10-10)
| Finding | Resolution |
|---|---|
| spec gaps: AsteriskID source, pure function, Get failure handling, test composition, docs procedure, 16 combos | 4.1, 4.2, 4.4, 4.7, 5 updated |
| round 6: conference join after injector is not rare (AI connect tool, flow action) and exposes the joined party | statement corrected, observable (4.2a), decision recorded with trigger and an explicit open question for the CEO (4.1) |
| mid-call leave via echo/Continue | 4.4 corrected, test added |
| triggers not observable, no review checklist, records not cleaned | 4.2a, 4.10, residual line in 4.4 |
| stale status/history | updated |

### Round-7 review response summary (2026-10-10)
| Finding | Resolution |
|---|---|
| MAJOR listen=none no longer means no inbound audio for customer-supplied hosts (flow action; `/calls/{id}/external-media` is internal RPC only, no caller today) | new path limited to the platform-internal host `INCOMING`; flow action and API request path stay legacy (2, 3, 4.1) |
| recording-based transcribe sees injected audio; existing recordings unchanged; billing; RST placement | 4.7, 4.9 |
| hold/MOH/mute unmeasured | live check 4 |
| stale 4.8 row, wrong cross reference, connect wording, 4.2a spec details, "three changes" | fixed |

### Round-8 response (2026-10-10)
PR assignment table 4.7a; ai-manager hangup subscription acknowledged in 4.4; internal-RPC wording, label and test corrections; unlimited session lifetime is the same as legacy.

### Round-9 (APPROVED x2) minor items applied
Section 6 reference, rollback order, 4.3 log level. Open decisions for the CEO (non-code): PR split (4.6) and the conference-join guard timing (4.1); the reviewer advises deciding the guard before PR-B merges.
