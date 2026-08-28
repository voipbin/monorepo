# VOIP-1406: Migrate event subscribers from fanout to topic patterns

Status: PLAN (stage 3 of 4). Issue analysis APPROVED (R1 RC, R2 RC, R3 Approve, R4
Approve). Design APPROVED (docs/plans/2026-08-29-voip-1406-consumer-topic-migration-design.md;
R1 RC, R2 Approve, R3 Approve with live-broker audit).

## Implementation Plan (stage 3) -- rev.1

Normative source: the Approved design (§2 template, §4 rulings, §5 bind sets, §6 tests,
§7 rollout). This plan adds ONLY execution mechanics.

### Waves

- [ ] **W1 -- per-service implementation (20 services, parallel executor batches of ~6;
  no inter-service ordering constraints -- no shared code changes).**
  Per service: add package-level `topicPatterns` (built via `eventtopic.PatternAction`,
  or `"#"` for timeline) and `fanoutUnbindTargets` vars in pkg/subscribehandler; insert
  the §2 template block into Run() (declare -> bind-all-or-nothing with best-effort
  partial rollback -> unbind fanout only on full success), strictly before the
  `go ConsumeMessage`; keep every fanout QueueSubscribe line; keep the sentinel
  defensive declares (call, timeline); add the binding golden test (pin exact pattern
  strings + retained fanout targets + §4 negative assertions where applicable); add
  Run() sequencing test (gomock InOrder; failure-path cases in billing, contact,
  call-manager, timeline); inline cleanups per §4 (number dead const, campaign consumer
  tag, contact stale 1233 comment); the THREE unreachable dispatch cases (ai
  conference_updated, queue customer_deleted, flow call_hangup) each keep a one-line
  comment pointing at **VOIP-1422** (registered before W1 -- ticket number is final);
  update the service's docs/architecture.md events section in the same commit.
  Per-service 5-step verification before each commit.
  Special: webhook string-join targets variance; agent/timeline place the new bin-manager.event block AFTER the existing 1258 webhook-topic block (both before `go ConsumeMessage`) and extend the existing InOrder chain accordingly; timeline replaces 25 subscribe calls
  with `#` (asterisk + 1258 binds untouched); call-manager keeps asterisk subscribe +
  numWorkers=20 untouched; api-manager NOT touched.
- [ ] **W2 -- docs (single commit).** Reference doc (rabbitmq-queues-reference.md):
  Exchanges section gains the "consumers now bind patterns; fanout publish remains until
  VOIP-1407" state note and the runbook commands (broker-binding inspection + manual
  unbind) per §7. (The unreachable-cases follow-up ticket is ALREADY registered:
  VOIP-1422.)
- [ ] **W3 -- verification + PR.** AC evidence below; main conflict check; PR creation.
  The PR body links the reference-doc runbook section and states that post-merge rollout
  follows design §7's wave order (low-volume -> multi-pattern -> timeline -> call-manager
  last).

### Executor protocol (as VOIP-1405/1419)

Executors confined to their service dir; never touch the git index; report
`git status --short -- <dir>` + verification output; orchestrator re-verifies and owns
all commits. `/usr/bin/grep`; vendor regen before tests.

### Acceptance criteria (evidence commands from worktree root)

- AC1: binding golden tests exist and pass in all 20 subscribehandler packages; the
  pinned pattern totals sum to **65 PatternAction patterns + 1 `#`** (agent4 ai10
  billing14 call4 campaign2 conference2 contact1 conversation4 direct1 flow1 number2
  queue3 registrar2 schedule1 storage2 tag1 transcribe3 transfer3 webhook5 + timeline#).
- AC2: `/usr/bin/grep -rl "TopicCreateWithKind(string(commonoutline.QueueNameEvent)" --include="*.go" --exclude="*_test.go" --exclude-dir=vendor --exclude-dir=.worktrees bin-*/pkg/subscribehandler/ | wc -l` -> **20 files** (one per service; test files excluded because sequencing tests repeat the literal in EXPECT calls). Services whose subscribehandler lacks a commonoutline import today (conference, conversation, transfer, webhook) add it.
- AC3: the 3 exclusions are proven by golden-test NEGATIVE assertions, not repo greps
  (production patterns are PatternAction calls, never literals; the negative rows
  necessarily contain the literals). Evidence: `/usr/bin/grep -Fc "conference-manager.conference.*.updated" bin-ai-manager/pkg/subscribehandler/*_test.go` >= 1 (negative row) while the same literal is absent from the POSITIVE expected set; likewise `customer-manager.customer.*.deleted` negative in queue-manager's golden, `call-manager.call.*.hangup` negative in flow-manager's golden; each golden's pattern-count assertion equals its §5 number.
- AC4: sentinel defensive declares still present (2: call, timeline); asterisk
  QueueSubscribe still present (2); `git diff --stat $(git merge-base HEAD origin/main) -- bin-api-manager/` -> empty.
- AC5: per-service verification workflow green (20 services); docs hook satisfied
  (architecture.md events sections updated in the same commits).
- AC6 (post-merge, live): broker bindings on bin-manager.event match §5 exactly per
  deployed wave; old fanout binds gone from each migrated queue; exchange publish_out
  grows; each service's subscribe-event process-time metric (name varies per service)
  continues advancing for its pairs.

## Results (W1-W3 executed 2026-08-29)

- W1: 20 services in 4 executor batches; every service adds package-level
  topicPatterns/fanoutUnbindTargets + the section-2 template block + a binding golden
  test + a Run sequencing test (failure paths in billing/contact/call/timeline);
  section-4 extras done (number dead const deleted, campaign consumer tag fixed,
  contact 1233 comment refreshed, three VOIP-1422 annotations); per-service docs synced;
  per-service 5-step verification green everywhere; mockgen param-rename drift discarded
  (agent, conference, number, transfer). Two pre-existing observations recorded for
  follow-up, untouched per scope rules: transcribe-manager consumer tag references
  call-manager's constant (same class as campaign's fixed bug); registrar
  subscribesTargets field-name typo.
- W2 (c32d14b26): reference doc gains the consumer-state note and the stale-binding
  runbook (inspection + manual unbind commands, roll-forward alternative).
- W3 AC evidence (from worktree root, merge-base a9c81fa0b):
  AC1 = 20 binding_golden_test.go files; 65 PatternAction calls + timeline "#" (sum
  matches section-5 exactly). AC2 = 20 declare files (test-excluded grep). AC3 = the 3
  negative assertions present in ai/queue/flow goldens. AC4 = sentinel defensive
  declares 2 (call :162, timeline :154), asterisk subscriptions 2, api-manager diff
  EMPTY. AC5 = 20-service verification green (W1). AC6 = post-merge live checks per
  the runbook.

## Working Notes (analysis retained below)

## Issue Analysis (2026-08-29)

### 1. Issue validity: VALID, and the window is open

- Prerequisites are complete: every publisher dual-publishes to `bin-manager.event`
  (VOIP-1404 pilot + VOIP-1405 rollout + VOIP-1419 explicit contract, all merged and the
  1405 rollout deployed 2026-08-28; exchange `publish_in` accumulating, error=0).
- Zero consumers bind `bin-manager.event` today (re-verified in the VOIP-1419 loop) --
  the consumer migration is the missing middle step before Follow-up C (VOIP-1407 cutover
  removes fanout publish; VOIP-1296 precedent).
- The ticket's claims check out empirically: exactly **21 consumer services** (all event
  consumption flows through `sockHandler.ConsumeMessage`; all 21 production call sites
  live inside `pkg/subscribehandler`s, one per service -- no back-door consumers),
  consuming **67 explicit (publisher, event-type) pairs + 2 wildcard arms** ("~70" was
  accurate).
- **Publisher-coverage audit of the non-opted binaries: NO active hole; two documented
  non-issues with one trap.**
  - `tts-control` lacks `WithGlobalTopicPublish` but wires ONLY `ttshandler`, whose
    notifyHandler is injected and never called (zero publish sites; the
    speaking/streaming publishers live only in `cmd/tts-manager`, which has the opt-in).
    Billing's speaking_started/stopped coverage is complete without it. Optional
    defensive hardening only -- NOT a 1406 prerequisite.
  - `talk-control` DOES wire publish-capable handlers (chat/message/participant/
    reaction), but its notifyHandler uses an EMPTY event exchange name, so those CLI
    publishes are black-holed today (fanout consumers never saw them either) -- no 1406
    regression. TRAP: `publishTopicEvent` targets `bin-manager.event` regardless of
    queueNotify, so adding the opt-in to talk-control would surface CLI-driven talk
    events on the topic exchange FOR THE FIRST TIME -- a behavior change. Keep
    talk-control excluded absent an explicit design ruling.
  - transfer/transfer-control publish nothing (verified); webhook-manager and
    asterisk-proxy stay deliberately excluded.

### 2. Code re-check: the facts the design must build on

**Transport is ready -- zero bin-common-handler changes needed.** The sockhandler
interface already has everything: `QueueBind/QueueUnbind` (VOIP-1258, idempotent tracked
binds, replayed by `redeclareAll()` on reconnect), `TopicCreateWithKind` (idempotent
declare), and `QueueSubscribe(q, topic)` is literally `QueueBind(q, "", topic, ...)` --
a fanout binding IS a topic binding with an empty key. Pattern builders exist in
`eventtopic` (`PatternAll/PatternResource/PatternInstance/PatternAction`).

**The in-repo migration template exists (VOIP-1258, 4a static form)** --
bin-agent-manager and bin-timeline-manager already migrated their webhook-manager
subscription to a topic exchange with a proven idiom whose load-bearing properties are:
1. bind-new-BEFORE-unbind-old (no unbound window);
2. bind failure aborts the unbind (degrade to fanout, never lose events);
3. unbind failure logs CRITICAL but is not fatal (double-processing beats loss);
4. the block lives INSIDE `Run()`, after `QueueCreate`, BEFORE the `go ConsumeMessage`
   goroutine -- an AMQP 503 channel race killed a pod in prod (2026-07-14) when it lived
   elsewhere. 1406 must keep this sequencing in all 21 services.
5. Momentary double-delivery between bind-new and unbind-old is accepted (1258 precedent).
6. **The template is 1 pattern / 1 unbind; 1406 services are N patterns / M unbinds.**
   The partial-failure semantics (pattern 3 of 5 fails to bind: abort remaining binds?
   abort ALL unbinds?) are a load-bearing decision the template does not answer -- the
   design must specify an explicit all-or-nothing rule (analysis lean: any bind failure
   -> bind nothing further and unbind NOTHING, stay fully on fanout for that service).

**The consumer contract is already written** (reference doc "Declaration invariant"):
both sides declare idempotently via `TopicCreateWithKind(QueueNameEvent, "topic")` --
subscriber-side declare is what makes start order irrelevant. Note: the 1258 consumers
(agent/timeline) do NOT declare the webhook topic exchange -- a live start-order gap the
reference doc tells 1406 not to repeat. `topicExchangeKind` ("topic") is currently a
notifyhandler-private const; hoisting one shared const/helper is nice-to-have.
The existing consumer-side defensive declare precedent: call-manager and timeline-manager
already `TopicCreate(target)` for the sentinel fanout exchange before subscribing.

**Structural exclusions (must be in scope table):**
- `asterisk.all.event` -> call-manager: voip-asterisk-proxy publishes raw ARI frames and
  is NOT on the topic exchange (permanent exclusion per 1404 design). call-manager's
  biggest consumer path (16 ARI types, numWorkers=20) keeps its fanout QueueSubscribe;
  only its customer/flow/sentinel subscriptions migrate.
- bin-api-manager: already topic-native (empty subscribeTargets; consumes the
  client-facing VOIP-1258 exchange via scoperefcount, which permanently coexists) --
  OUT of 1406 scope entirely.
- bin-webhook-manager: consumer side migrates like any other; its publisher side must
  keep NOT using WithGlobalTopicPublish (triple-publish ban).
- timeline-manager: the one true `#` consumer -- 25 of its 26 QueueSubscribe calls
  collapse to ONE `QueueBind(q, "#", bin-manager.event)`, keeping only the asterisk
  fanout subscription. **CAUTION -- the `#` bind is a SUPERSET, not a 1:1 collapse**:
  27 services publish to the topic exchange, including direct/schedule/webchat whose
  fanout exchanges timeline never subscribed, so `#` starts delivering their events (new
  ClickHouse rows) and auto-includes every FUTURE publisher. By this analysis's own
  standard that is a behavior change needing an explicit design ruling (likely desirable
  for a timeline service, but it must be a decision, not a side effect). Also: timeline's
  `QueueNameTransferEvent` target is a dead bind today -- transfer-manager publishes
  nothing.

**Pattern mapping:** `PatternAction(pub, res, action)` maps 1:1 onto today's
(publisher, event-type) switch pairs via the same `SplitN(type, "_", 2)` normalization
(`confbridge_joined` -> `call-manager.confbridge.*.joined`). Payload on the topic
exchange is the SAME sock.Event, so dispatch switches (Publisher/Type keys) need zero
logic changes -- 1406 is a bindings-only migration per service.

**Binding/dispatch mismatches discovered (design must rule on each):**
- ai-manager: binds transcribe/tts fanout exchanges it never dispatches (dead binds;
  transcribemanager.go is fully commented out) AND has an unreachable
  `conference-manager/conference_updated` case (never binds the conference exchange).
- queue-manager: dead binds (agent, conference) AND an unreachable
  `customer-manager/customer_deleted` case (never binds customer).
- flow-manager: an unreachable `call-manager/call_hangup` case (subscribeTargets is
  customer-only; `EventCallHangup` has never run) -- found in design review R1, the
  third unreachable case and a latent-bug candidate alongside queue's.
- number-manager: a dead wrong-valued const (`publisherCustomerManager =
  ServiceNameQueueManager`), harmless today.
- campaign-manager: consumer tag passes `ServiceNameQueueManager` (copy-paste, harmless)
  -- adjacent cleanup candidate while the file is touched.
- Publisher identity strings are a mix of commonoutline constants and per-service
  hardcoded consts -- topic patterns should be built from one identity source.
Under topic bindings "bind exactly what you dispatch" forces a per-case decision:
dropping dead binds is pure cleanup; binding for a today-unreachable case would be a
BEHAVIOR change (events start arriving) and needs an explicit keep-or-delete ruling per
case in the design, not a silent side effect.

**Bind-after-start race (1404 design §5.4):** moot for 1406 -- every existing consumer
binds static publisher/resource-level patterns at boot, not instance addresses learned
from RPCs. The (a)/(b)/(c) options apply only if a service later opts into
PatternInstance; no such consumer exists in this migration.

### 3. Scope decision: VOIP-1233 stays OUT

The ticket suggests considering coupling with VOIP-1233 (subscribe callback error
propagation; 17 of 21 services are fire-and-forget `go processEvent; return nil`, which
makes the library's already-built ack-after-process/retry machinery dead code). Analysis
recommendation: keep it out. The two changes touch the same files but are orthogonal --
1406 edits `Run()`'s binding block; 1233 changes `processEventRun` semantics and
requires per-service idempotency review (a retried `customer_deleted` must be safe).
Coupling turns 21 mechanical bind-swap diffs into 21 semantic-change diffs. The 1406 PR
should note the adjacency (and fix the now-stale VOIP-1233 comment in contact-manager's
subscribehandler if touched) so the follow-up stays unambiguous.

### 4. Proceeding rationale

- Right time: dual publish is live and soaking; consumers migrating per service with
  independent rollback (restore fanout bind) is exactly the design's staging. Follow-up C
  (fanout removal) is blocked on this ticket.
- Risk profile: bindings-only change, per-service; the 1258 template is proven in prod;
  reconnect semantics come free from tracked binds; rollback = redeploy previous image
  (its Run() re-subscribes fanout; the topic binding left on the durable queue must be
  considered -- design should specify whether rollback needs a manual unbind or tolerates
  double delivery until roll-forward). The same stale-binding ruling must also cover the
  rolling-deploy window on 2-replica services: an OLD-image pod hitting a broker
  reconnect after the NEW-image pod unbound fanout replays its tracked fanout bind onto
  the shared durable queue, leaving silent double delivery with no rollback having
  happened (trigger differs, end state and remediation identical; broker-binding
  inspection is the detection step).
- Cost: ~20 services' subscribehandler Run() blocks + cmd subscribeTargets wiring +
  per-service docs sync; no bch changes required (optional tiny declare helper/const
  hoist would touch bch -- design decides whether it clears the 3+-consumer admission
  rule, which it trivially does at 20 consumers).

### 5. Acceptance criteria (draft, finalized in design/plan)

- Each of the 20 in-scope services (21 minus api-manager) binds `bin-manager.event`
  with patterns covering EXACTLY its dispatch set (or `#` for timeline, per the design's
  explicit superset ruling), declares the exchange idempotently before binding, and
  unbinds its old fanout exchanges only per the design's multi-pattern all-or-nothing
  rule, inside Run() before ConsumeMessage.
- asterisk fanout subscription in call-manager (and timeline) preserved untouched.
- Live verification: broker shows the expected bindings on `bin-manager.event`;
  `publish_out` grows; per-service event processing continues (existing
  promEventProcessTime metrics by Publisher/Type unchanged).
- Full verification workflow per touched service; no bch behavior change.

## Working Notes

- Worktree: `.worktrees/VOIP-1406-Migrate-subscribers-to-topic-bindings` (from main
  a9c81fa0b, post-VOIP-1419).
- Full topology map (21 services, 67 pairs, per-service file:line, API surface, 1258
  template details, 1233 adjacency) produced 2026-08-29 by repo scan; commands recorded
  in the scan report.
- Env quirks: /usr/bin/grep; vendor regen before tests; RST hook on models/*/webhook.go
  (not expected to trigger -- no model changes); PreToolUse hook blocks gh pr merge.
