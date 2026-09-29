# VOIP-1542: Surface LLM pipeline errors on aicall

Status: Final (analysis loop closed: rounds 6+7 APPROVED; design loop closed: rounds 5+6 APPROVED)
Ticket: VOIP-1542
Branch: `VOIP-1542-Surface-LLM-pipeline-errors-on-aicall` (monorepo + monorepo-javascript, one PR per repo)

## 1. Issue analysis

### 1.1 Trigger incident

aicall `443bfb46-fa3a-4dd5-912d-5482d31fed22` (2026-09-29 07:58 UTC, admin test chat, `reference_type=""`).
AI `6e391666` had `engine_key="1234"`, `engine_model=gemini.gemini-2.5-flash`. Three user turns
("안녕", "Hi", "Hello?") each started a pipecatcall (`9c5c6e64`, `0d26685f`, `8e478999`). None
produced an assistant message. The admin timeline showed only the user rows: plain silence.

### 1.2 Evidence (Loki, read-only)

For each of the three pipecatcalls, `voipbin-pipecat-manager-1` logged exactly one frame:

```
{"frame":{"label":"rtvi-ai","type":"error","data":{"error":"Unknown error occurred: 400 Bad Request.
 {... \"message\": \"API key not valid. Please pass a valid API key.\", \"status\": \"INVALID_ARGUMENT\",
 \"reason\": \"API_KEY_INVALID\" ...}","fatal":false}},
 "func":"receiveMessageFrameMessage","message":"Unrecognized RTVI message type: error","severity":"DEBUG"}
```

followed by `BotLLMStopped received but no tokens were received for this generation.` Nothing about the
error reached `bin-ai-manager` (its logs for the aicall contain only message create, pipecatcall
initialized/terminated, and one unrelated 3s send-cooldown rejection).

### 1.3 Code re-check (current `main`, 8e108ac83)

- `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` `receiveMessageFrameTypeMessage`: the `switch frame.Type`
  handles `bot-transcription`, `user-transcription`, `user-llm-text`, `bot-llm-text`, `bot-llm-stopped`.
  Everything else falls to:
  ```go
  default:
      log.WithField("frame", frame).Debugf("Unrecognized RTVI message type: %s", frame.Type)
  ```
  So the RTVI `error` frame is dropped at DEBUG level. This is the drop point for runtime pipeline errors
  (scope: ErrorFrames emitted while a pipeline is live). The same `default:` also drops `error-response`
  frames. pipecat-manager DOES send a client RTVI request on every messaging turn (`SendMessage` ->
  `SendRTVIText`, `pkg/pipecatcallhandler/etc.go:31`, `pipecatframe.go:121-126`, type `send-text`), and
  pipecat 1.4.0 answers it with `error-response` when the message fails validation, raises inside
  `_handle_send_text`, or has an unsupported type (`rtvi/processor.py:357-370`). Those are request/contract
  failures between our own two components (a platform bug, not a provider failure and not actionable by the
  customer), so they are out of scope for the customer notice; this change still adds an explicit
  `error-response` case that WARN-logs and counts them so they stop being invisible to operators.
  Frames whose label is not `rtvi-ai` are dropped at 561-565 (none are emitted today). Startup failures take
  a different path (`/run` returns HTTP 400/500, `scripts/pipecat/main.py:139-144`, surfaced as a Go error
  from `pythonRunner.Start`; a failure inside the background task is only logged, main.py:100-101) and are
  out of scope here.
- `bin-pipecat-manager/models/pipecatframe/helper.go` has no `error` frame-type constant.
  `models/pipecatframe/rtvi.go` already defines `RTVIError{Label, Type, Data RTVIErrorData{Error string, Fatal bool}}`
  (matches pipecat-ai 1.4.0 `RTVIError`, `rtvi.py` `_send_error_frame`: `RTVIError(data=RTVIErrorData(error=frame.error, fatal=frame.fatal))`).
  pipecat-ai pinned version: `uv.lock` `pipecat-ai 1.4.0`.
- In pipecat-ai 1.4.0 the RTVIProcessor (not the observer) converts every `ErrorFrame` it receives into this
  frame (`pipecat/processors/frameworks/rtvi/processor.py:231-232` route, `:517-520`
  `RTVI.Error(data=RTVI.ErrorData(error=frame.error, fatal=frame.fatal))`). PipelineTask prepends it
  automatically (`enable_rtvi=True` default, `pipeline/worker.py:403-406,471`), and `push_error_frame` travels
  upstream (`frame_processor.py:698`), so errors that STT, LLM and TTS services surface via `push_error`
  all reach it, for both the single-AI and team pipelines (`run.py:244,701`). Scope of the claim: provider
  failures that pipecat surfaces as ErrorFrame (the Gemini 400 in 1.2 is one). Services that swallow or
  retry an exception without `push_error` never produce the frame and stay out of reach of this fix.
- Not every `error` frame is a provider failure. Also observed in 1.4.0 source: malformed inbound RTVI
  messages (`processor.py:280-292`, `:162-168`), exceptions inside our own function-call handlers
  (`services/llm_service.py:1462-1464`, text `Error executing function call [<name>]: ...`), and transient
  STT reconnect errors (Google STT re-raises its recoverable 409 inactivity abort into the reconnect loop,
  which then calls `push_error("Unknown error occurred: ...")`, `services/google/stt.py:936-937,1016-1027`;
  Deepgram `_on_error` pushes errors for drops it retries, `services/deepgram/stt.py:678-680`). The design
  (3.1) must therefore decide per error which ones become customer notices.
- Provider timeouts: OpenAI/Grok (`OpenAILLMService`) push the literal `LLM completion timeout` on an httpx
  timeout (`services/openai/base_llm.py:566-568`); an openai `APITimeoutError` surfaces as
  `Error during completion: Request timed out.` (`base_llm.py:570`, openai `_exceptions.py:99-101`).
  Gemini: `services/google/llm.py:607` catches `google.api_core.exceptions.DeadlineExceeded`, but the pinned
  google-genai 1.75.0 client never raises that type, so that branch is effectively dead. pipecat calls
  `generate_content_stream` (`google/llm.py:435`, SSE over aiohttp), so Gemini errors arrive via
  `llm.py:609-610` in two forms: (a) HTTP error response, the common case: aiohttp `.json()` raises `ContentTypeError`
  (content-type mismatch), so genai falls back to the HTTP reason phrase with the body text in details (genai `errors.py:227-236`), e.g.
  `Unknown error occurred: 504 Gateway Timeout. {...DEADLINE_EXCEEDED...}`; the incident's
  `400 Bad Request.` in 1.2 is this form; (b) an error chunk inside a 200 stream
  (`_api_client.py:1672-1681`), e.g. `Unknown error occurred: 504 DEADLINE_EXCEEDED. {...}`; or httpx/aiohttp
  timeout text. A client-side hang
  with no response may never error at all (genai default `http_options.timeout=None`; inference, not fully traced).
  google-genai's default is a single attempt, so 400/429 are pushed on first failure.
- Which services exist per session: non-call aicalls (conversation, task, contact_case, `""`, Insight listen
  turns) start their pipecatcall with no STT and no TTS (`bin-ai-manager/pkg/aicallhandler/start.go:952-956,1007`,
  `listen.go:428-431`), so every error frame there is LLM- or function-call-origin. Only voice (call) pipecatcalls
  carry STT/TTS, and only they see the transient STT reconnect errors.
- Insight listen turns each start a fresh ai_call pipecatcall with a throwaway id (`listen.go:379`) that is
  never written to `AIcall.PipecatcallID`. They are debounced (call kind on transcript arrival, leaky bucket,
  min interval flag default 20s, `internal/config/main.go:124`, `ListenTurnTryLock` `listen.go:742-765`;
  conversation kind on message arrival, `listen_conversation.go:347,406,428`) and hard-capped at 60 turns per
  aicall for both kinds (`main.go:127`, `listen.go:318`). Per-pipecatcall dedup alone allows up to 60
  identical notices per aicall from listen turns.
- `isForeignPipecatcall` (`messagehandler/main.go:215`, `ac.PipecatcallID != evt.PipecatcallID`) is true for
  (a) listen turns and (b) a stale superseded turn whose event arrives after the next `Send` rotated the id
  (`send.go:116-119`, `start.go:349-350`; interrupting the previous pipecatcall is best-effort; the helper's own
  comment names both cases). It is false for the initial contact_case turn, which is registered as a listen
  turn (`start.go:632-651`) but runs on the aicall's current id. The exact listen-turn registry is
  `cache.ListenTurnPipecatcallIDIsMember` (`tool.go:55`), which lives in aicallhandler's cache handler, not in
  messagehandler. The existing `isForeignPipecatcall` callers (`event.go:169,182,315`) gate on
  `ReferenceTypeContactCase`.
- Fatal errors: the worker queues a `CancelFrame` on `fatal=True` (`pipeline/worker.py:1146-1151`), which
  may race the RTVI message out through the transport; a fatal notice is best-effort. The subsequent
  `pipecatcall_terminated` still fires.
- The frame is delivered over the runner's output WebSocket. If that WebSocket itself is what failed, the
  error cannot arrive; that case is already visible as a transport disconnect and is not addressed here.
- `bin-ai-manager`: `messagehandler.EventPMPipecatcallTerminated` (event.go:436) sends a canned
  "Sorry, I'm having trouble responding right now" backstop reply, but only for
  `reference_type=conversation`. For `""`/`call`/`task`/`contact_case` aicalls there is no signal at all,
  and even for conversation the operator never learns WHY.

### 1.4 Validity and go/no-go

- Valid and reproducible: every provider failure surfaced as ErrorFrame, on every aicall reference type, is
  silent to the customer/operator today. The drop point is the missing switch case in 1.3; closing the gap
  additionally needs a new event and an ai-manager consumer (the missing downstream half).
- Not already fixed on main (checked 8e108ac83). Open PRs checked with
  `gh pr list --repo voipbin/monorepo --state open --limit 50` filtered for error/pipecat/rtvi: none.
- Worth doing now: low cost (one new event, one handler, one UI card), high diagnostic value, and
  it is the prerequisite for any future "engine_key validation" UX (follow-up item 2 in the incident report).
- Out of scope (explicit): validating `engine_key` at AI save time; falling back to the platform key when a
  customer key fails (BYOK cost-attribution issue, rejected); changing the conversation backstop text.

## 2. Goals / non-goals

Goals
1. An actionable provider failure (see the notice policy in 3.1) inside a pipecatcall that belongs to an
   aicall becomes a persisted, customer-visible aicall message, delivered through the existing
   `aimessage_created` webhook/WS path. Every error frame, actionable or not, is WARN-logged and counted.
2. The message tells the customer the class of failure (auth / rate limit / timeout / other) in plain text,
   without leaking platform-internal provider detail.
3. Operators keep the full raw error in logs at WARN (first occurrence), so it is visible to default
   operator log views and WARN-based alerting instead of being buried in DEBUG noise.
4. The rows never enter the pipecat prompt context of the conversing AI and the agent panel (square-talk)
   never shows them. Offline analysis consumers (audit evaluator, prompt proposals, get_aicall_messages
   tool) DO see them on purpose: an evaluator should know the AI did not answer because the provider failed.
5. Admin renders them as a recognisable error card in both places that show aicall messages: the aicall
   detail timeline and the admin test chat (`TestAgentSheet`), which is where the trigger incident happened.
6. Voice calls: this is a timeline/webhook notice only. The caller on the phone still hears silence; nothing
   is played or spoken. (Spoken fallback is a separate product decision, not in scope.)

Non-goals: see 1.4.

## 3. Design

### 3.1 pipecat-manager: recognise and publish

- `models/pipecatframe/helper.go`: add `RTVIFrameTypeError = "error"`.
- `models/message`: add event type `EventTypePipelineError = "pipeline_error"` and payload
  ```go
  type PipelineErrorEvent struct {
      CustomerID               uuid.UUID                 `json:"customer_id,omitempty"`
      PipecatcallID            uuid.UUID                 `json:"pipecatcall_id,omitempty"`
      PipecatcallReferenceType pipecatcall.ReferenceType `json:"pipecatcall_reference_type,omitempty"`
      PipecatcallReferenceID   uuid.UUID                 `json:"pipecatcall_reference_id,omitempty"`
      ActiveflowID             uuid.UUID                 `json:"activeflow_id,omitempty"`
      Category                 ErrorCategory             `json:"category,omitempty"`
      Fatal                    bool                      `json:"fatal"`
  }
  func (h *PipelineErrorEvent) EventSubscriptionID() string { return h.PipecatcallID.String() }
  ```
  Routing key: `pipecat-manager.pipeline.<pipecatcall-id>.error` (mechanical split of `pipeline_error`).
  The five routing fields mirror `MemberSwitchedEvent` (the existing precedent in the same package);
  there is no shared base struct to embed today, and introducing one plus refactoring `MemberSwitchedEvent`
  is out of scope.
  `Category` enum (payload): `authentication`, `rate_limited`, `timeout`, `unknown`. `function_call` and `internal`
  never reach the payload (policy step 1).
  The raw provider string is deliberately NOT in the payload (see 3.3).
- `pkg/pipecatcallhandler/runner.go` `receiveMessageFrameTypeMessage`: new case
  ```go
  case pipecatframe.RTVIFrameTypeError:
      msg := pipecatframe.RTVIError{}
      if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil { return errors.Wrapf(...) }
      category := classifyPipelineError(msg.Data.Error)
      metricsPipelineErrorTotal.WithLabelValues(string(category), strconv.FormatBool(msg.Data.Fatal)).Inc()
      firstSeen := se.MarkPipelineErrorSeen(category)
      if firstSeen {
          log.WithField("category", category).Warnf("Pipeline error. fatal: %v, error: %s", msg.Data.Fatal, truncate(msg.Data.Error, 2000))
      } else {
          log.WithField("category", category).Debugf("Pipeline error (repeat). fatal: %v, error: %s", msg.Data.Fatal, truncate(msg.Data.Error, 2000))
      }
      if firstSeen && shouldNotifyPipelineError(category, msg.Data.Fatal, se.HasSTT) {
          go h.notifyHandler.PublishEvent(se.Ctx, message.EventTypePipelineError, h.newPipelineErrorEvent(se, category, msg.Data.Fatal))
      }
  ```
  `PublishEvent` in a goroutine matches every other case in this switch (read loop must not block).
  Plus an `error-response` case (`RTVIFrameTypeErrorResponse = "error-response"`): WARN log with the text,
  separate counter `pipecat_manager_rtvi_error_response_total` (no labels), no event (see 1.3).
- `Session` gains `HasSTT bool` (json:"-"), set in `SessionCreate` from `pc.STTType != STTTypeNone`.
- Notice policy `shouldNotifyPipelineError(category, fatal, hasSTT)` (pure, table-tested, first match wins):
  1. `function_call`, `internal` -> no notice (our own bug, operator signal only), regardless of fatal.
  2. `fatal=true` -> notify. Future-proofing: none of the services VoIPBin wires today (OpenAI, Grok,
     Gemini, Deepgram, Google STT/TTS, ElevenLabs, Cartesia) push `fatal=True`.
  3. `authentication`, `rate_limited`, `timeout` -> notify.
  4. `unknown` -> notify only when `hasSTT == false`.
  Rationale: in no-STT sessions (all text/messaging aicalls, including the incident's admin test chat) every
  error frame is LLM-origin, so wrong model (`404 NOT_FOUND`), region `FAILED_PRECONDITION`, provider 5xx
  and Gemini `504 DEADLINE_EXCEEDED` must be announced. In voice sessions non-fatal `unknown` includes
  transient STT reconnects (Google STT 409 inactivity abort when the caller is silent ~10s), which would be
  false alarms, so there it stays log + metric only. Cost: a generic non-fatal LLM 5xx during a voice call
  is not announced (still counted; conversation backstop unaffected).
- `models/pipecatcall/session.go`: add a small per-session set of reported categories
  (`seenPipelineErrors map[ErrorCategory]struct{}` guarded by a mutex, accessed only through
  `MarkPipelineErrorSeen(category) bool`, returns true the first time a category is seen; the map is created
  lazily inside that method under the mutex, so a zero-value Session never panics). One set drives both the
  WARN-once log and the notify gate. A later fatal frame of an already-seen category is not re-announced
  (acceptable: fatal is future-proofing only).
  Dedup: at most one event per (pipecatcall, category). Accepted limitation: a later, different error of
  the same category (e.g. a TTS 5xx after an STT reconnect, both `unknown`) is logged at DEBUG only; the
  metric still counts it. Messaging aicalls use one pipecatcall per turn, so a
  failing turn yields one notice. A voice call is one pipecatcall; a key that fails on every utterance
  yields one notice for the whole call instead of one per utterance, while a different failure class later
  in the call (e.g. auth then timeout) still surfaces. Known limitation: in a team aicall, if a member switch
  moves to another AI that fails with the SAME category, that second failure is not re-announced (the
  first notice already tells the operator that class of failure is occurring). Later errors are still
  WARN-logged and counted.
- `pkg/pipecatcallhandler/metrics.go`: `pipecat_manager_pipeline_error_total{category,fatal}` counter
  (`category` in authentication/rate_limited/timeout/function_call/internal/unknown, `fatal` "true"/"false") and
  `pipecat_manager_rtvi_error_response_total` (names checked against
  `bin-common-handler/pkg/requesthandler/main.go#initPrometheus()`, no collision).
- Logging: WARN (text truncated to 2000 chars) for the first frame of each category in a session, DEBUG
  afterwards (Google STT with a persistent auth/quota failure re-pushes every ~1s, `google/stt.py:936-940`).
  The metric counts every frame. Applies to every category, including non-notified ones.
- `classifyPipelineError(raw string) ErrorCategory` (pure function, runner-side helper file), lower-cased,
  tiers evaluated in order, first hit wins:
  0. Prefix `error executing function call [` -> `function_call` (our own tool handler raised); prefix
     `invalid rtvi transport message` -> `internal` (malformed inbound RTVI envelope, `processor.py:291`).
  1. Structured provider status/reason tokens: `api_key_invalid`, `invalid_api_key`, `permission_denied`,
     `unauthenticated` -> authentication; `resource_exhausted`, `rate_limit_exceeded` -> rate_limited.
  2. HTTP status phrases (anchored to the phrase, not a bare number): `401 unauthorized`, `403 forbidden`,
     `http 401`, `http 403` (websockets `InvalidStatus`: `server rejected WebSocket connection: HTTP 401`,
     ElevenLabs/Cartesia handshake), regex `(status|code)["':= ]{0,4}(401|403)\b` -> authentication; `429 too many requests`,
     regex `(status|code)["':= ]{0,4}429\b` -> rate_limited.
  3. Free-text phrases: `api key not valid`, `invalid api key`, `incorrect api key`, `invalid authentication`
     -> authentication; `rate limit`, `exceeded your current quota`, `quota exceeded` -> rate_limited;
     `llm completion timeout`, `error during completion: request timed out` (OpenAI/Grok literals),
     `504 gateway timeout`, `504 deadline_exceeded` (Gemini, both forms in 1.3) -> timeout.
  Generic `timed out` / `deadline exceeded` (space) text is NOT matched: Google STT inactivity reconnects use
  it, so it stays `unknown` (then the policy notifies only in no-STT sessions, where it cannot be STT).
  Otherwise `unknown`. Bare numbers and bare words like `unauthorized`/`authentication`/`quota` are NOT
  matched on their own (request ids and URLs can contain them). Deterministic, table-tested. Code decides
  the category; nothing LLM-generated.

### 3.2 ai-manager: persist as an aicall notification

- `pkg/subscribehandler/main.go`: add pattern
  `eventtopic.PatternForEventType(ServiceNamePipecatManager, pmmessage.EventTypePipelineError)`
  (`pipecat-manager.pipeline.*.error`) and a dispatch case. Update `binding_golden_test.go` (13 -> 14).
- `pkg/subscribehandler/pipecat_message.go`: `processEventPMPipelineError` unmarshals and calls
  `messageHandler.EventPMPipelineError(ctx, &evt)`.
- `pkg/messagehandler/event.go` `EventPMPipelineError`:
  - skip unless `evt.PipecatcallReferenceType == ai_call` (non-aicall pipecatcalls have no message timeline).
  - content = JSON (same convention as `member_switched`):
    ```json
    {"type":"pipeline_error","category":"authentication","fatal":false,
     "pipecatcall_id":"9c5c6e64-...","message":"The AI provider rejected the API key. Check the AI's engine key."}
    ```
    `message` is a fixed English plain-text sentence per category (no markdown). The RTVI error frame does
    not say which service (LLM, STT or TTS) failed, and the key may be the customer's or the platform's, so
    the wording names the failure class without blaming a specific key:
    - authentication: `An AI service provider rejected the credentials or denied access. If this AI uses a custom engine key, verify that the key is valid and permitted for the selected model.`
    - rate_limited: `An AI service provider rejected the request due to a rate limit or quota. Try again later.`
    - timeout: `The AI language model did not respond in time.`
    - unknown: `An AI service provider returned an error.`
    Accepted imprecision: `permission_denied` / 403 also covers "API not enabled", region blocks and
    unsupported-country errors; the "or denied access" wording keeps the sentence true in those cases.
  - `h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
     message.DirectionOutgoing, message.RoleNotification, content, nil, "",
     WithPipecatcallID(evt.PipecatcallID), WithActiveAIID(h.resolveActiveAIID(ctx, evt.PipecatcallReferenceID)))`
    Direction/role mirror `EventPMTeamMemberSwitched`; `WithPipecatcallID` is an addition (that handler does
    not set it) so the row carries the originating pipecatcall. `Create` already publishes the
    `aimessage_created` webhook, so admin/WS/webhook consumers get it with no new plumbing.
  - Errors: log and return (same as `EventPMTeamMemberSwitched`), no requeue loop.
  - Foreign-pipecatcall dedup: fetch the aicall once (`AIV1AIcallGet`) and reuse it for
    `resolveActiveAIIDFromAIcall(ctx, ac)` (no second fetch). If the fetch fails, fail open: create the row
    with a nil active AI and skip dedup. If `isForeignPipecatcall(ac, evt.PipecatcallID)`, first confirm with
    `AIV1AIcallGetSkipCache` (same stale-cache guard as `EventPMMessageBotLLM`, `event.go:169-192`; the
    cached id can lag an update whose cache refresh failed); if the fresh row says it is current, treat it as
    current; if SkipCache itself fails, also treat it as current and create the row (fail-open, the opposite
    of the bot-LLM path, which drops, because a missed error notice is worse than a duplicate). Keep the
    nil-`reqHandler` guard that `resolveActiveAIID` has. Only for a confirmed foreign pipecatcall, list the newest
    notification rows of this aicall (`MessageList`, filters aicall_id + role=notification, size 50,
    `tm_create desc`; rows whose content fails to parse as JSON count as no match) and skip if one has
    `type=pipeline_error` and the same `category` within the last
    10 minutes (`pipelineErrorNoticeWindow`). Bounds a bad-key Insight session to about one notice per
    category per 10 minutes instead of up to 60. Size 50 can in theory miss an in-window row buried under
    many `member_switched` rows in a team aicall; negligible, accepted.
    Current-pipecatcall turns (messaging, task, voice, the initial contact_case turn) are NOT windowed: every
    failing turn yields its own notice, so an operator who edits the engine key and retries in the admin test
    chat immediately sees whether the new key also fails. Voice is already one-per-category per call on the
    pipecat side. Accepted edge: a stale superseded messaging turn whose error arrives after the next `Send`
    (1.3 case b) is also foreign and therefore windowed; the newer current turn still gets its own row, so
    nothing actionable is hidden. The discriminator applies to every reference type (the existing helper is
    used only for contact_case today); for non-listening types only case (b) can occur.
    Why not `ListenTurnPipecatcallIDIsMember`: it would add a cache-handler dependency to messagehandler for
    no customer-visible gain over the accepted edge above.
    MessageList failure -> create anyway (fail-open). A two-pod race can still create two rows; accepted.
  - Delivery is at-least-once; a redelivered event can create a duplicate notification row. Same accepted
    behavior as `member_switched`; no idempotency key added.
- Safety properties already provided by existing code (no change needed, verified):
  - LLM context: `aicallhandler/start.go:828` skips `RoleNotification` when building pipecatcall messages.
  - Conversation backstop: `MessageAssistantReplyExists` checks assistant rows only, so the backstop reply
    still fires for conversation aicalls; the notice complements it.
  - square-talk agent panel: `isRenderableMessage` is an allowlist (user/assistant/tool-card), notification hidden.
  - square-admin teams view (`teams_detail.js:126-137`) ignores non-`member_switched` notifications.
- Offline consumers that render all rows (intentional, Goal 4): audit evaluator transcript
  (`aiaudithandler/main.go:264`, `buildTranscript:414-432` prints `[notification]: {json}`), prompt proposals
  (`aipromptproposalhandler/prompt_builder.go:64-67`), `get_aicall_messages` tool (`aicallhandler/tool.go:997`).
  A test pins that the audit transcript includes a `pipeline_error` row. `aiaudits_detail.js:68` shows it as
  raw JSON, acceptable.

### 3.3 Why category, not raw provider text, in the customer-visible row

When the AI has no `engine_key`, pipecat uses the platform `GOOGLE_API_KEY` / platform keys. Raw provider
errors can then contain platform-internal detail (GCP project numbers in quota errors, masked key suffixes,
internal model routing). Exposing that to every customer is a leak. The category plus a fixed sentence gives
the customer the actionable part ("your key was rejected"), and the raw text stays in the WARN log for
operators. If later we want raw detail for BYOK keys only, it can be added behind that condition; not now.

### 3.4 square-admin

- `src/views/aicalls/aicalls_detail.js` `renderMessage`, `notification` branch: add a
  `parsed?.type === 'pipeline_error'` card before the generic JSON fallback: red left border, title
  "AI Error", category badge, the `message` sentence, small `Fatal` marker when true, timestamp.
  `getFilterClass` treats it like `member_switched` (never dimmed, it is session-level). `switchCount` stays
  `member_switched`-only.
- `src/components/TestAgentSheet.js` `ChatNotificationMessage` (`:70-106`): same `pipeline_error` card in the
  chat style, so the admin test chat (the incident's screen) shows the error instead of raw JSON.
- The card markup lives in one shared component (e.g. `src/components/PipelineErrorNotice.js`) used by both.
- Unit tests in `__tests__/aicalls_detail.test.js` and a TestAgentSheet test.

### 3.5 Docs

- `bin-api-manager/docsdev/source/ai_struct_message.rst`: notification row gains a note that `content`
  is a JSON object with a `type` field, and a short subsection documenting `pipeline_error`
  (fields, categories, example). Clean rebuild + commit HTML.
- `bin-pipecat-manager/docs/domain.md` events table: add the `pipeline_error` row; update the prose that
  says "all three resource namespaces" (lines 33, 48) to four (`pipeline`), and add `PipelineErrorEvent` to
  the pointer-receiver `SubscriptionIdentifier` explanation. Code adds
  `var _ eventtopic.SubscriptionIdentifier = (*PipelineErrorEvent)(nil)` and an `event_test.go` entry.
- `bin-ai-manager/docs/architecture.md` subscriptions table: add `pipecat-manager.pipeline.*.error`
  (re-extract with `bash docs/reference/extractor.sh bin-ai-manager` per CLAUDE.md) and change
  "13 patterns total" (line 89) to 14.
- `bin-pipecat-manager/docs/operations.md` metrics table: add the new counter.
- pipecat-manager routing-key golden test: add the `pipeline_error` key and the shared-address case.

## 4. Tests

- pipecat-manager: `classifyPipelineError` table test (the real Gemini string from 1.2 -> authentication,
  OpenAI "Incorrect API key provided" -> authentication, "429 Too Many Requests" -> rate_limited,
  "RESOURCE_EXHAUSTED" -> rate_limited, "500 Internal" -> unknown,
  ElevenLabs/Deepgram 401 bodies -> authentication, a pipecat transport/WebSocket error -> unknown,
  literal "LLM completion timeout" -> timeout, "Error during completion: Request timed out." -> timeout,
  "Unknown error occurred: 504 DEADLINE_EXCEEDED. {...}" -> timeout, "Unknown error occurred: 504 Gateway
  Timeout. {...}" -> timeout, "Invalid RTVI transport message: x" -> internal, "Request timed out" -> unknown, "Unknown error occurred: 409 Stream
  timed out after receiving no more client requests." -> unknown, "Unknown error occurred: server rejected
  WebSocket connection: HTTP 401" -> authentication, "Error executing function call [send_email]: boom" ->
  function_call, "Unknown error occurred: 404 NOT_FOUND. {...}" -> unknown;
  `shouldNotifyPipelineError` table (every category x fatal x hasSTT); `HasSTT` set from pc.STTType;
  WARN-once-per-category then DEBUG (including non-notified categories); non-notified error publishes nothing but
  still increments the metric; error-response frame -> WARN + metric, no event;
  "req-401-x" and a URL containing "unauthorized" in a timeout body must NOT be authentication);
  per-category dedup (same category twice -> one event, different category -> second event);
  metric increments on every frame;
  `receiveMessageFrameTypeMessage` with an error frame publishes exactly one `pipeline_error` event with
  the session routing fields; a second error frame on the same session publishes nothing;
  malformed error frame returns an error; event type / routing key golden.
- ai-manager: `EventPMPipelineError` creates one notification row with the exact JSON for each category;
  audit `buildTranscript` includes the pipeline_error row; listen-turn (foreign pipecatcall) dedup: same
  category within window -> no row, outside window -> row, different category -> row, MessageList error ->
  still create (fail-open); current-pipecatcall event with an in-window same-category row -> row created
  (not windowed); cached aicall says foreign but SkipCache says current -> treated as current; AIcallGet
  failure -> row created with nil active AI, no dedup; SkipCache failure -> row created; stale foreign event
  with no in-window same-category row -> row created; unparseable notification content -> ignored by dedup;
  created row carries `pipecatcall_id`;
  non-aicall reference type creates nothing; Create failure is logged, no panic; subscribe dispatch +
  binding golden.
- square-admin: aicall detail and TestAgentSheet render the AI Error card with category and message;
  unknown notification types still hit the generic fallback.

## 5. Rollout / compatibility

- New event type on the existing topic exchange; old ai-manager simply has no binding and ignores it,
  so deploy order does not matter.
- No DB schema change (existing `role=notification` rows).
- No OpenAPI change: `role` enum already includes `notification`; `content` is a string.

## 6. Open questions for review

- (resolved in v2) Dedup granularity: per (pipecatcall, category), see 3.1.
- (resolved in v2) `fatal`: kept in the JSON, shown only as a small marker.
- No open questions remain.

## 7. Review log

- Round 1 analysis review: CHANGES_REQUESTED. RTVIProcessor vs observer naming, scope of "single drop
  point", unverified "every provider failure", output-WS caveat, open-PR query not recorded. All fixed in §1.
- Round 1 design review: CHANGES_REQUESTED. MAJOR: Goal 4 overclaim (audit/proposal/tool see rows) ->
  narrowed + test; auth text blamed customer key for platform-key and STT/TTS failures -> neutral wording;
  TestAgentSheet (incident screen) missing -> added. MINOR: classifier tiers, per-category dedup + team
  limitation, Prometheus counter, voice semantics note, extractor re-run. All applied.
- Round 2 analysis review: CHANGES_REQUESTED. `error-response` claim false (pipecat-manager sends
  send-text every turn) -> corrected, kept out of notice scope with the true reason, WARN+metric added.
  Non-provider error sources and fatal cancel race -> documented in 1.3.
- Round 2 design review: CHANGES_REQUESTED. MAJOR: timeout category unreachable (real string is
  "LLM completion timeout", Gemini pushes nothing) -> matched literally, generic timeout text demoted;
  not every error frame is a provider failure (STT 409 reconnect, function-call exceptions) -> notice policy
  (3.1) + function_call category. MINOR: unknown wording, 403 wording, docs prose (namespaces count,
  13->14, pointer-receiver + interface assertion), lazy map init, at-least-once duplicates. All applied.
- Round 3 analysis review: CHANGES_REQUESTED. Gemini timeout "no frame" claim wrong (dead DeadlineExceeded
  branch; real text `504 DEADLINE_EXCEEDED`) -> corrected, literal matched as timeout.
- Round 3 design review: CHANGES_REQUESTED. MAJOR: non-fatal unknown suppressed in text sessions where all
  errors are LLM-origin -> policy now keyed on HasSTT; Insight listen-turn flooding -> aicall-level 10-min
  dedup in ai-manager. MINOR: timeout test strings, websockets HTTP 401 phrase, enum vs function_call,
  separate error-response counter, fatal future-proofing note, WARN-once logging. All applied.
- Round 4 analysis review: CHANGES_REQUESTED. MAJOR: Gemini streaming path yields `504 Gateway Timeout.`
  (reason phrase) not only `504 DEADLINE_EXCEEDED.` -> both forms documented and matched. MINOR: listen turns
  are transcript-debounced and capped at 60, default flag at main.go:124 -> corrected. NIT worker.py cite.
- Round 4 design review: CHANGES_REQUESTED. MAJOR: 10-min window hid per-turn notices in admin test chat
  -> window limited to listen turns (foreign pipecatcall). MINOR: single MarkPipelineErrorSeen drives
  WARN-once + gate; stale "Request timed out" test fixed; same-category distinct errors documented; Goal 3
  wording; `internal` category for malformed RTVI envelope; page size note. All applied.
- Round 5 analysis review: CHANGES_REQUESTED. "listen turns are the only foreign case" false (stale
  superseded turns also foreign; initial contact_case turn is current) -> 1.3 corrected, 3.2 documents the
  accepted edge and the scope. Nits (conversation-kind debounce cite, ContentTypeError wording) applied.
- Round 5 design review: APPROVED. MINOR/NIT applied: SkipCache re-check before suppressing, AIcallGet
  fail-open, single aicall fetch, WithPipecatcallID noted as addition, pipecatcall_id asserted in tests.
- Round 6 analysis review: APPROVED (nits: contact_case-gated callers noted).
- Round 6 design review: APPROVED. Design loop closed (rounds 5, 6 consecutive APPROVED). MINOR/NIT applied
  (SkipCache failure fail-open, stale-turn and parse-failure tests, nil reqHandler guard, switchCount note).
- Round 7 analysis review: APPROVED. Analysis loop closed (rounds 6, 7 consecutive APPROVED). Nit applied
  (third isForeignPipecatcall caller event.go:182).
