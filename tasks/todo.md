# VOIP-1404 Implementation Plan

Design: `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` (Approved)
Branch: `VOIP-1404-Global-topic-exchange-skeleton`

## Acceptance criteria

- [ ] `bin-manager.event` constant exists in `outline`; exchange declared only via shared helper.
- [ ] `models/eventtopic` package: `SubscriptionIdentifier`, `RoutingKey`, 4 pattern builders — all table-tested incl. dot/uppercase/no-underscore/empty types, Nil/empty id → `-`.
- [ ] notifyhandler dual publish behind `WithGlobalTopicPublish()` option on both constructors; fanout behavior byte-identical when option off; topic failure isolated; fanout failure skips topic; declare-failure degradation (non-nil handler, `topicDisabled` field); metrics contract per design §5.2/§7.
- [ ] transcribe-manager pilot: option enabled in both cmds; `Speech`/`Streaming`/`Transcript` implement `EventSubscriptionID()` (pointer receivers) returning `TranscribeID`; golden routing-key test covering all 10 published event types.
- [ ] Reference doc: "Exchanges" section in `docs/reference/rabbitmq-queues-reference.md`; consumer contract (bind-after-start) recorded.
- [ ] Verification workflow green in `bin-common-handler` AND `bin-transcribe-manager` (tidy/vendor/generate/test/lint). Coverage ≥80% on new/changed files, evidenced via `go test -cover` on the affected packages (added to 1.6/2.7).
- [ ] Follow-up Jira tickets registered: A (remaining publishers), B (consumer migration), C (cutover), db.go:33 bug.

## Tasks

### Phase 1 — bin-common-handler skeleton
- [x] 1.1 `models/outline/queuename.go`: add `QueueNameEvent QueueName = "bin-manager.event"` with VOIP-1404 comment.
- [x] 1.2 New package `models/eventtopic`:
  - `identifier.go`: `SubscriptionIdentifier` interface (doc comment: pointer-receiver requirement, "subscription address" semantics, shared-binding/refcount caveat per design §4.3).
  - `routingkey.go`: `RoutingKey(publisher, eventType, subscriptionID string) string` — normalization (lowercase, `.`→`_`), SplitN resource/action, `-` placeholders, residual-char sanitize; `PatternAll/PatternResource/PatternInstance/PatternAction`.
  - `routingkey_test.go`: table-driven tests per design §8 list.
- [x] 1.3 `pkg/notifyhandler`:
  - `main.go`: `Option func(*notifyHandler)`; `WithGlobalTopicPublish()` sets `topicEnabled`; both constructors take `opts ...Option`; ordering: fanout TopicCreate (NewNotifyHandler only) → initPrometheus → (if topicEnabled) global declare via `TopicCreateWithKind`; on declare failure log Error + set `topicDisabled`, return working handler.
  - `main.go` initPrometheus: register `topic_publish_total{type,result}` + `topic_placeholder_total{type}` unconditionally in existing guarded block.
  - `publish.go`: `PublishEvent` resolves subscription-id via type assertion (else `""`); `publishEvent(..., subscriptionID string)` new param; topic publish after successful fanout publish, `delay == 0` guard, direct `h.sockHandler.EventPublish(string(commonoutline.QueueNameEvent), key, evt)` — NOT the promNotifyProcessTime-observing helpers; JSON `id` fallback inside publishEvent when subscriptionID == ""; suppressed publishes (topicDisabled) count `{result="error"}`.
  - `PublishEventRaw` passes `""`.
  - Tests: gomock ordering, isolation (topic error ⇒ no caller error, fanout still done), fanout error ⇒ no topic call, option-off ⇒ zero topic calls, Raw ⇒ fallback key, delay>0 (private fn) ⇒ no topic, metrics assertions.
- [x] 1.4 bin-common-handler docs sync (same commit as code):
  - `docs/architecture.md`: Package Overview tree (+`models/eventtopic/`), `pkg/notifyhandler` responsibilities (dual publish/option), Prometheus Metrics table (+2 new counters). While touching the metrics table: fix its wrong preamble ("constructs a requesthandler.RequestHandler" — the notify counters are registered by notifyhandler.initPrometheus) and the mislabeled "Webhook delivery" purpose on notify_total/notify_process_time rows.
  - `bin-common-handler/CLAUDE.md`: Package layout table +`models/eventtopic` row.
  - Note: bin-common-handler has no `docs/domain.md`; the docs-sync hook warning for `models/*` changes is expected and unsuppressible — not a missed step.
- [x] 1.5 Admission-rule deviation trace: `models/eventtopic` package doc comment carries the 3+-services exemption justification (notifyhandler internal plumbing, Follow-up A adoption); repeat in PR body.
- [x] 1.6 Verification workflow in `bin-common-handler` + `go test -cover` on changed packages (≥80% evidence).

### Phase 2 — transcribe-manager pilot
- [x] 2.1 `models/streaming/`: `Speech.EventSubscriptionID()`, `Streaming.EventSubscriptionID()` (pointer receivers → `TranscribeID.String()`); `models/transcript/`: `Transcript.EventSubscriptionID()`. Compile-time assertions in tests.
- [x] 2.2 `cmd/transcribe-manager/main.go` + `cmd/transcribe-control/main.go`: add `notifyhandler.WithGlobalTopicPublish()`.
- [x] 2.3 Golden routing-key test: all 10 event types (`transcribe_created/progressing/done/deleted`, `transcribe_speech_started/interim/ended`, `streaming_started/stopped`, `transcript_created`) → exact expected keys. Comment noting it pins current db.go:33 behavior.
- [x] 2.4 Service docs sync (same commit):
  - `bin-transcribe-manager/docs/domain.md` — "Events Published" table (`### Events Published`, domain.md:72; models/* changes fire the domain.md rule). Table currently lists only 3 of 10 published types — complete it while touching it.
  - `bin-transcribe-manager/docs/architecture.md` — line 27 notifyhandler row + line 41: after pilot the service dual-publishes to `bin-manager.event`; cmd/*/main.go changes fire the architecture.md hook rule (suppressible by staging this file).
- [ ] 2.5 Option-wiring test seam: design §8 requires publish-path tests asserting option wiring in both cmds, but `cmd/*/` have no test files. Resolution: golden-key test (2.3) covers key generation; option wiring is asserted via a notifyhandler unit test (option ⇒ topic publish called) + cmd wiring verified by code review. Record this substitution in the PR body (design §8 deviation, rationale: no cmd test seam exists repo-wide).
- [x] 2.6 Cross-consumer compile sweep: `go build ./...` in every `bin-*`/`voip-*` dir **containing a go.mod** (39 Go modules; bin-dbscheme-manager has none and is excluded). CI only tests bin-common-handler itself; bin-common-handler/CLAUDE.md requires consumer builds on public API change — variadic addition is source-compatible but verify, don't assert. Run before any third service gains a vendor/ dir.
- [x] 2.7 Verification workflow in `bin-transcribe-manager` + `go test -cover` on changed packages (≥80% evidence).

### Phase 3 — docs & follow-ups
- [x] 3.1 `docs/reference/rabbitmq-queues-reference.md`: new "Exchanges" section (bin-manager.event vs webhook scope-first topic vs delay; key schema; consumer contract incl. bind-after-start candidates; declaration invariant).
- [ ] 3.2 Register follow-up Jira tickets: Follow-up A/B/C + transcripthandler db.go:33 bug (Korean body, English summary).
- [ ] 3.3 PR runbook section: post-deploy verification per design §7, **including pre-deploy baseline capture** of `transcribe_manager_notify_total{type}` from Prometheus (design §5.5 mitigation 1) — baseline goes into the PR body so the post-deploy comparison is performable.
- [ ] 3.4 RST docs determination: NOT affected — change is internal service-to-service plumbing, no user-visible API/webhook/billing change. Record this determination in PR body.

### Phase 4 — review & PR
- [ ] 4.1 Commit: title matches branch name exactly (`VOIP-1404-Global-topic-exchange-skeleton`), body = project-prefixed bullets, no AI attribution. Verify staged content via `git diff --cached` before committing.
- [ ] 4.2 Code review loop (min 3 rounds, 2 consecutive approvals, per policy). Re-run verification workflow after any review-fix commit.
- [ ] 4.3 Pull main + conflict check from worktree. If rebase/merge needed: resolve, then re-run FULL verification workflow before PR.
- [ ] 4.4 Single PR: title `VOIP-1404-Global-topic-exchange-skeleton`, body per monorepo format (no headers, project-prefixed bullets, no AI attribution; includes §8 deviation note, admission-rule note, RST determination, baseline, runbook). NO merge without explicit instruction.

## Working notes

- Verification runs must happen in BOTH changed services; bin-common-handler change normally implies dependent-service verification — for this PR, transcribe-manager is the only behavioral dependent (option is opt-in, default off ⇒ all other consumer modules unaffected behaviorally; the 2.6 compile sweep covers all 39 Go modules regardless). NotifyHandler interface itself is UNCHANGED — constructors are not interface methods, so no mock regeneration in other services.
- mockgen: notifyhandler mock regenerates only if interface changed — it doesn't. eventtopic has no interface needing mocks (pure functions).
- Do not touch `promNotifyTotal`/`promNotifyProcessTime` observation sites.
