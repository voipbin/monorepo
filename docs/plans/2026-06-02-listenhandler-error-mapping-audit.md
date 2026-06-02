# Listenhandler Error-Mapping — Phase 2 Monorepo Audit

**Date:** 2026-06-02
**Spec:** `docs/superpowers/specs/2026-06-02-listenhandler-error-mapping-design.md`
**Phase 1 (merged):** PR #959 — conversation/message/number/call(groupcalls)
**This doc:** Phase 2 discovery — classify the same swallow pattern across all services; drive Phase 3 remediation.

## Method

For every `bin-*-manager` with a `pkg/listenhandler`, each `return simpleResponse(500), nil`
(and, in call-manager, `simpleResponse(400)`) was classified:

- **SWALLOW** — `if err != nil` block immediately after a handler call `h.X.Method(...)`; discards
  the typed `*cerrors.VoipbinError` / raw `dbhandler.ErrNotFound` the handler emits. **Fix:** replace
  with `return errorResponse(err), nil`.
- **MARSHAL** — after `json.Marshal(...)` (response serialization failure). **Leave** as 500.
- **OTHER** — anything else; noted individually.

`errorResponse` maps typed VoipbinError→true status, `dbhandler.ErrNotFound`(through `errors.Wrap`)→404,
else→500. So converting a SWALLOW is **safe everywhere**: a plain internal error still yields 500
(identical to today), while typed/not-found errors now yield their correct status.

## Key facts

- **Every service already has the `errorResponse` helper.** No helper porting needed anywhere.
- **Central tails:** most are WIRED (route typed + `ErrNotFound`); **call-manager and billing-manager
  are UNWIRED** (bare `simpleResponse(400)`/`(500)`). Irrelevant to the fix — we map at the site, so
  the tail is never relied upon. Do **not** wire an unwired tail inside a de-swallow PR (it would
  silently reclassify unrelated bare-`(nil,err)` endpoints).
- **Handler typing:** by-id `Get` methods almost universally emit typed `cerrors.NotFound`;
  `Delete`/`Update`/action methods generally re-fetch or wrap the raw sentinel, which survives
  `errors.Wrap` → 404 via `errorResponse`.

## Summary table

| Service | Tail | SWALLOW sites | Priority | Notes |
|---|---|---|---|---|
| bin-call-manager (non-groupcall) | UNWIRED | ~33 | **P1** | confbridges swallow as **400** incl. IDGet/IDDelete (404→400 regression); calls IDDelete/digits/recordingID/confbridgeID etc. |
| bin-queue-manager | WIRED | 15 | **P1** | queues IDPut/TagIDs/RoutingMethod/Execute; queuecalls reference-id get/kick |
| bin-talk-manager | WIRED | 13 | **P1** | chats IDPut/IDDelete, participants, messages IDDelete, reactions (many by-id) |
| bin-rag-manager | WIRED | 7 | **P1** | rags IDPut/IDDelete/sources by-id; also latent: IDGet returns 404 for ALL errors |
| bin-ai-manager | WIRED | 19 | P2 | by-id already wired; swallows are List/Create/action (Start/Send/ToolHandle/ServiceStart) |
| bin-conference-manager | WIRED | 6 | P2 | List/Create/Count/DirectHash/ServiceStart |
| bin-customer-manager | WIRED | 4 | P2 | accesskeys+customers List/Create |
| bin-registrar-manager | WIRED | 4 | P2 | extensions Create/Count/DirectHash; trunks Count |
| bin-agent-manager | WIRED | 4 | P2 | List/GetByAddress/DirectHash/Count |
| bin-conversation-manager (accounts/msgs) | WIRED | 3 | P2 | accounts List; messages Post/Create |
| bin-flow-manager | WIRED | 2 | P2 | count; flows direct-hash (DirectHashRegenerate can 404) |
| bin-number-manager (avail/count) | WIRED | 2 | P3 | available-numbers List; count (no by-id 404 semantics) |
| bin-pipecat-manager | WIRED | 3 | P3 | by-id already wired; Start/SendMessage/Ping |
| bin-transcribe-manager | WIRED | 1 | P3 | transcribes List (by-id already wired) |
| bin-timeline-manager | WIRED | 4 | P3 | List/aggregate/SIP — handlers emit only plain errors; `errorResponse`=500 = no behavior change (optional, consistency-only) |
| bin-billing-manager | UNWIRED | 0 to fix | — | CRUD already uses `errorResponse`; the 8 Paddle-webhook 500s are **intentional retry signals — LEAVE** |

Services with no listenhandler swallows (clean): campaign, contact, direct, email, message(done), outdial, route, storage, tag, transfer, tts, webhook.

## Per-service SWALLOW sites (file:line — function — handler method)

### P1

**bin-call-manager** (exclude v1_groupcalls.go — done)
- v1_calls.go:60 — processV1CallsGet — callHandler.List
- v1_calls.go:144 — processV1CallsPost — callHandler.CreateCallsOutgoing
- v1_calls.go:200 — processV1CallsIDPost — callHandler.CreateCallOutgoing
- v1_calls.go:237 — processV1CallsIDDelete — callHandler.Delete  *(by-id 404)*
- v1_calls.go:274 — processV1CallsIDHangupPost — callHandler.HangingUp  *(by-id 404)*
- v1_calls.go:430 — processV1CallsIDChainedCallIDsPost — callHandler.ChainedCallIDAdd
- v1_calls.go:466 — processV1CallsIDChainedCallIDsDelete — callHandler.ChainedCallIDRemove
- v1_calls.go:596 — processV1CallsIDDigitsGet — callHandler.DigitsGet  *(by-id 404)*
- v1_calls.go:638 — processV1CallsIDDigitsSet — callHandler.DigitsSet  *(by-id 404)*
- v1_calls.go:670 — processV1CallsIDRecordingIDPut — callHandler.UpdateRecordingID  *(by-id 404)*
- v1_calls.go:709 — processV1CallsIDConfbridgeIDPut — callHandler.UpdateConfbridgeID  *(by-id 404)*
- v1_calls.go:748 — processV1CallsIDRecordingStartPost — callHandler.RecordingStart
- v1_calls.go:783 — processV1CallsIDRecordingStopPost — callHandler.RecordingStop
- v1_calls.go:822 — processV1CallsIDTalkPost — callHandler.Talk
- v1_calls.go:879 — processV1CallsIDMediaStopPost — callHandler.MediaStop
- v1_confbridges.go:38 — processV1ConfbridgesPost — confbridgeHandler.Create  *(currently **400**)*
- v1_confbridges.go:73 — processV1ConfbridgesIDGet — confbridgeHandler.Get  *(currently **400**; 404 regression)*
- v1_confbridges.go:107 — processV1ConfbridgesIDDelete — confbridgeHandler.Delete  *(currently **400**)*
- v1_confbridges.go:142 — processV1ConfbridgesIDTerminatePost — confbridgeHandler.Terminate  *(currently **400**)*
- v1_confbridges.go:177 — processV1ConfbridgesIDCallsIDDelete — (kick) *(currently **400**)*
- v1_confbridges.go:201 — processV1ConfbridgesIDCallsIDPost — (join) *(currently **400**)*
- v1_confbridges.go:324 — processV1ConfbridgesIDRecordingStartPost — confbridgeHandler.RecordingStart
- v1_confbridges.go:359 — processV1ConfbridgesIDRecordingStopPost — confbridgeHandler.RecordingStop
- v1_confbridges.go:399 — processV1ConfbridgesIDFlagsPost — confbridgeHandler.FlagAdd
- v1_confbridges.go:439 — processV1ConfbridgesIDFlagsDelete — confbridgeHandler.FlagRemove
- v1_confbridges.go:473 — processV1ConfbridgesIDRingPost — confbridgeHandler.Ring
- v1_confbridges.go:495 — processV1ConfbridgesIDAnswerPost — confbridgeHandler.Answer
- v1_recordings.go:54 — processV1RecordingsGet — recordingHandler.List
- v1_recordings.go:88 — processV1RecordingsPost — recordingHandler.Start
- v1_external_medias.go:54 — processV1ExternalMediasGet — externalMediaHandler.List
- v1_external_medias.go:101 — processV1ExternalMediasPost — externalMediaHandler.Start
- v1_outbound_configs.go:45 — processV1OutboundConfigsPost — outboundConfigHandler.Create  *(preserve the existing "Duplicate entry"→409 special-case before the fallback)*
- v1_outbound_configs.go:84 — processV1OutboundConfigsGet — outboundConfigHandler.List

**bin-queue-manager**
- v1_queues.go:101 — processV1QueuesGet — queueHandler.List
- v1_queues.go:232 — processV1QueuesIDPut — queueHandler.UpdateBasicInfo  *(by-id 404)*
- v1_queues.go:274 — processV1QueuesIDTagIDsPut — queueHandler.UpdateTagIDs  *(by-id 404)*
- v1_queues.go:316 — processV1QueuesIDRoutingMethodPut — queueHandler.UpdateRoutingMethod  *(by-id 404)*
- v1_queues.go:408 — processV1QueuesIDAgentsGet — queueHandler.GetAgents  *(by-id 404)*
- v1_queues.go:474 — processV1QueuesIDExecutePut — queueHandler.UpdateExecute  *(by-id 404)*
- v1_queues.go:509 — processV1QueuecallsIDStatusWaitingPost — queuecallHandler.UpdateStatusWaiting  *(by-id 404)*
- v1_queues_direct_hash.go:31 — processV1QueuesIDDirectHashRegeneratePost — queueHandler.DirectHashRegenerate  *(by-id 404)*
- v1_count.go:37 — processV1QueuesCountByCustomerGet — queueHandler.CountByCustomerID
- v1_queuecalls.go:54 — processV1QueuecallsGet — queuecallHandler.List
- v1_queuecalls.go:90 — processV1QueuecallsReferenceIDIDGet — queuecallHandler.GetByReferenceID  *(by-id 404)*
- v1_queuecalls.go:258 — processV1QueuecallsIDExecutePost — queuecallHandler.Execute  *(by-id 404)*
- v1_queuecalls.go:293 — processV1QueuecallsIDKickPost — queuecallHandler.Kick  *(by-id 404)*
- v1_queuecalls.go:328 — processV1QueuecallsReferenceIDIDKickPost — queuecallHandler.KickByReferenceID  *(by-id 404)*
- v1_services.go:33 — processV1ServicesTypeQueuecallPost — queuecallHandler.ServiceStart

**bin-talk-manager**
- v1_chats.go:51 — v1ChatsPost — chatHandler.ChatCreate
- v1_chats.go:92 — v1ChatsGet — chatHandler.ChatList
- v1_chats.go:137 — v1ChatsIDPut — chatHandler.ChatUpdate  *(by-id 404)*
- v1_chats.go:154 — v1ChatsIDDelete — chatHandler.ChatDelete  *(by-id 404)*
- v1_chats_participants.go:39 — v1ChatsIDParticipantsPost — participantHandler.ParticipantAdd  *(by-id 404)*
- v1_chats_participants.go:70 — v1ChatsIDParticipantsGet — participantHandler.ParticipantList  *(by-id 404)*
- v1_chats_participants.go:102 — v1ChatsIDParticipantsIDDelete — participantHandler.ParticipantRemove  *(by-id 404)*
- v1_messages.go:54 — v1MessagesPost — messageHandler.MessageCreate
- v1_messages.go:98 — v1MessagesGet — messageHandler.MessageList
- v1_messages.go:140 — v1MessagesIDDelete — messageHandler.MessageDelete  *(by-id 404)*
- v1_participants.go:46 — v1ParticipantsGet — participantHandler.ParticipantListWithFilters
- v1_reactions.go:34 — v1MessagesIDReactionsPost — reactionHandler.ReactionAdd  *(by-id 404)*
- v1_reactions.go:65 — v1MessagesIDReactionsDelete — reactionHandler.ReactionRemove  *(by-id 404)*

**bin-rag-manager**
- v1_rags.go:54 — processV1RagsPost — ragHandler.RagCreate
- v1_rags.go:90 — processV1RagsGet — ragHandler.RagList
- v1_rags.go:162 — processV1RagsIDPut — ragHandler.RagUpdate  *(by-id 404)*
- v1_rags.go:186 — processV1RagsIDDelete — ragHandler.RagDelete  *(by-id 404)*
- v1_rags.go:226 — processV1RagsIDSourcesPost — ragHandler.RagAddSources  *(by-id 404)*
- v1_rags.go:259 — processV1RagsIDSourcesIDDelete — ragHandler.RagRemoveSource  *(by-id 404)*
- v1_query.go:41 — processV1QueryPost — ragHandler.QueryRag
- *Latent (separate fix, out of scope): v1_rags.go:115 processV1RagsIDGet returns `simpleResponse(404)` for ALL RagGet errors — should use `errorResponse(err)` so non-not-found errors are 500.*

### P2

**bin-ai-manager** — v1_ais.go:61,113 ; v1_aicalls.go:60,94,252 ; v1_participants.go:47,94 ; v1_ais_direct_hash.go:30 ; v1_teams_direct_hash.go:30 ; v1_teams.go:61,103 ; v1_summaries.go:60,94 ; v1_messages.go:60,94 ; v1_services.go:30,72,106

**bin-conference-manager** — v1_conferences.go:54,99 ; v1_conferences_direct_hash.go:30 ; v1_count.go:36 ; v1_conferencecalls.go:54 ; v1_services.go:30

**bin-customer-manager** — v1_accesskeys.go:56,99 ; v1_customers.go:55,101

**bin-registrar-manager** — v1_extensions.go:48 ; v1_count.go:37 ; v1_count.go:69 ; v1_extensions_direct_hash.go:30

**bin-agent-manager** — v1_agents.go:61,241 ; v1_agents_direct_hash.go:30 ; v1_count.go:37

**bin-conversation-manager** (exclude v1_conversations.go) — v1_accounts.go:54 ; v1_messages.go:33,132

**bin-flow-manager** — v1_count.go:37 ; v1_flows_direct_hash.go:40

### P3 (low 404 impact; full-fidelity/consistency only)

**bin-number-manager** (exclude v1_numbers.go) — v1_available_numbers.go:75 ; v1_count.go:37
**bin-pipecat-manager** — v1_pipecatcalls.go:45 ; v1_messages.go:27 ; v1_ping.go:23
**bin-transcribe-manager** — v1_transcribes.go:94
**bin-timeline-manager** — v1_events.go:30 ; v1_aggregated_events.go:30 ; v1_sip.go:46,92  *(behavior-neutral: handlers emit only plain errors → still 500; convert for consistency only, or skip)*

## Explicitly LEAVE

- **bin-billing-manager** Paddle webhook 500s (`v1_hooks_paddle.go:157,184,223,250,286,308,360,373`) — intentional retry signaling for the Paddle webhook; converting to 404/typed would change retry semantics. Leave.
- All MARSHAL 500s (after `json.Marshal`) across every service.
- Pre-handler request-parse paths returning `simpleResponse(400)`.

## Latent issues found (separate, out of Phase 3 scope — track individually)

- **bin-rag-manager** `v1_rags.go:115` — `processV1RagsIDGet` returns 404 for *every* `RagGet` error (internal errors masked as 404). Fix: `errorResponse(err)`.
- **bin-pipecat-manager** — `v1_pipecatcalls.go:88,123`, `v1_messages.go:32` return `simpleResponse(404)` after a `json.Marshal` failure (wrong code; should be 500).
- **bin-call-manager** `outboundConfigHandler.GetByID` returns `(nil, nil)` for a missing row, so `processV1OutboundConfigsIDGet` yields 200 with `null` body instead of 404 (not a swallow; a handler-layer not-found gap).

## Phase 3 remediation plan

Each service is an independent PR (TDD: add not-found/typed test → convert SWALLOW sites to
`errorResponse(err)` → leave MARSHAL/parse → full 5-step verification → review). Recommended batching
by priority:

- **Batch 1 (P1):** call-manager (non-groupcall), queue-manager, talk-manager, rag-manager.
- **Batch 2 (P2):** ai, conference, customer, registrar, agent, conversation(accounts/msgs), flow.
- **Batch 3 (P3):** number(avail/count), pipecat, transcribe; timeline optional (behavior-neutral).

For each PR: replace handler-call `simpleResponse(500)`/`simpleResponse(400)` swallows with
`errorResponse(err)`; preserve special-cases (e.g. outbound-config `Duplicate entry`→409); add
per-endpoint not-found tests for the by-id endpoints flagged `(by-id 404)`. Do not change central
tails. Latent issues above are tracked as their own follow-ups.
