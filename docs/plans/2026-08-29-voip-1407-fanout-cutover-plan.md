# VOIP-1407 implementation plan: cutover -- remove fanout dual publish and per-service event exchanges

- Date: 2026-08-29
- Ticket: VOIP-1407
- Stage: 3 of 4 (implementation plan). Stage 1 (issue analysis, `tasks/todo.md` rev.11) and stage 2 (design, `docs/plans/2026-08-29-voip-1407-fanout-cutover-design.md`) are both APPROVED and are the authoritative sources for every technical decision below.
- Branch / worktree: `VOIP-1407-Cutover-remove-fanout-dual-publish`, worktree at `~/gitvoipbin/monorepo/.worktrees/VOIP-1407-Cutover-remove-fanout-dual-publish`
- Status: DRAFT -- pending the 설계 review loop (min 2 rounds, 2 consecutive Approve)

**Citation convention used throughout.** `design §N` = a section of `docs/plans/2026-08-29-voip-1407-fanout-cutover-design.md`. `issue §N` = a section of `tasks/todo.md`. `path:line` = a source citation carried forward from one of those two documents, or (where marked *verified this plan*) re-confirmed against the worktree while writing this plan. **No task in this plan introduces a decision that is not already made in the design doc.** Where this plan found something the design doc does not cover, it is raised in §8 (Open questions) rather than decided here.

---

## 1. Goal restatement

Execute the cutover the design doc specifies: make the 55 real dual-publish call sites (plus `talk-control` as a 56th, per design §2.3.1) publish **only** to the global topic exchange `bin-manager.event` by removing the fanout leg from `bin-common-handler/pkg/notifyhandler`'s `topicEnabled=true` path (design §2.1-§2.6), and delete the now-dead fanout consumer machinery -- the fanout `QueueSubscribe` loop, `subscribeTargets`, and `fanoutUnbindTargets`/`QueueUnbind` step -- from the 20 VOIP-1406 consumer services (design §3.1-§3.4), while leaving `voip-asterisk-proxy` and the `asterisk.all.event` leg entirely untouched (design §0/§3.2) and the VOIP-1258 webhook-topic block retained verbatim with its existing log-only failure semantics (design §3.2). Documentation is updated in the same PR (design §6). The broker-side deletion of the 28 per-service fanout exchanges (design §5) is **operational, post-merge, and not part of this PR's code**; this PR only writes its runbook into `docs/reference/rabbitmq-queues-reference.md`.

---

## 2. Dependency ordering

Everything below ships in **ONE PR** (root `CLAUDE.md`: "one PR per task"; design §1's "can ship in separate PRs/commits" describes *deploy-time independence*, not a mandate to split the PR -- see §7/OQ-4 for how this plan reconciles the two). "Ordering" here means the sequence of commits/work inside that one PR, chosen so that `go build`/`go test` surfaces integration breaks as early as possible.

### 2.1 The actual dependency graph

```
  Unit A  bin-common-handler/pkg/notifyhandler          [STRICTLY FIRST]
     |      (breaking behavioral change; consumed by all 62 publish call sites
     |       and, via go.mod replace, compiled into every one of the ~38 services)
     |
     +-----------------------------+-----------------------------+
     |                             |                             |
  Unit B  bin-talk-manager      Unit C  3 dead-wiring      Unit D  bin-webhook-manager
  talk-control companion fix    deletions (transfer x2,    comment-only (design §2.6)
  (design §2.3.1)               tts-control) (design §4)
     |                             |                             |
     +-----------------------------+-----------------------------+
                                   |
  Unit E  20 consumer services' pkg/subscribehandler + cmd wiring
          (design §3.1-§3.3)  -- INDEPENDENT of A/B/C/D at the source level,
          and the 20 services are mutually independent of each other
                                   |
  Unit F  Documentation (design §6)  [LAST -- needs the final shape of A-E]
```

**Strictly sequential:**

- **A before B.** `talk-control`'s fix exists *only because* Unit A introduces `logrus.Fatalf` in `NewNotifyHandler` (design §2.3.1: "Why §2.3 makes this materially worse"). Doing B first has no meaning and no way to verify.
- **A before C and D.** Both C and D are edits to call sites of the constructor A changes. C's deletions and D's comment must describe A's post-change semantics.
- **A before the *verification* of B/C/D/E.** `bin-common-handler` is consumed by every service through a `replace` directive; until A compiles and its tests pass, every downstream `go test ./...` is testing against a stale shared library.
- **A-E before F.** Design §6's doc updates describe the end state; writing them against a half-finished code state guarantees rework.

**Parallelizable:**

- **B, C, D are mutually independent** (three different services, no shared files).
- **The 20 services inside Unit E are mutually independent.** Each service's `pkg/subscribehandler/main.go` + `cmd/*-manager/main.go` + its own tests are a self-contained edit with no cross-service file overlap. They can be done in any order, or fanned out across parallel workers, with one caveat: the **3 exception services (call-manager, agent-manager, timeline-manager) must not be batched into a mechanical sweep** -- design §3.2 exists precisely because a wholesale loop deletion would destroy a live production binding in those three. Do the 17 typical services as a sweep; do the 3 exceptions individually, by hand, each with its own review pass.
- **Unit E is independent of A/B/C/D at the source level.** No file in Unit E imports anything Unit A changes in a way A's edits alter. In practice, still run Unit E's verification *after* A lands locally, so the `go mod vendor` step picks up the same `bin-common-handler` tree the PR ships.

### 2.2 Recommended commit sequence inside the single PR

| # | Commit scope | Unit | Rationale |
|---|---|---|---|
| 1 | `bin-common-handler` (code + tests) | A | Breaking change first; everything downstream compiles against it |
| 2 | `bin-talk-manager` | B | Directly motivated by commit 1 |
| 3 | `bin-transfer-manager`, `bin-tts-manager` | C | Independent deletions |
| 4 | `bin-webhook-manager` | D | Comment-only |
| 5 | 17 typical consumer services | E (sweep) | Mechanical, uniform |
| 6 | `bin-call-manager`, `bin-agent-manager`, `bin-timeline-manager` | E (exceptions) | Isolated so the diff for the risky three is reviewable on its own |
| 7 | Docs (shared + per-service) | F | Reflects the final state |

Commits 5 and 6 are deliberately separated so a reviewer can read the three exception services' diff without it being buried in a 17-service mechanical sweep. This is a *reviewability* choice inside one PR, not a PR split.

---

## 3. Task breakdown

### Unit A -- `bin-common-handler/pkg/notifyhandler`

**Design authority: §2.1, §2.2, §2.3, §2.4, §2.5, §2.6. Test authority: §8.**

This is the whole behavioral change. Everything else in the PR is a consequence of it.

#### A1. `main.go` -- comment rewrites (design §2.1, three comments)

| File:line (design's citation) | Change |
|---|---|
| `bin-common-handler/pkg/notifyhandler/main.go:175-186` | `WithGlobalTopicPublish()`'s doc comment. Rewrite per design §2.1 bullet 1: enabling the option makes the instance topic-ONLY (no fanout publish, no fanout exchange declared); the default (option omitted) is unchanged fanout-only. |
| `main.go:70-76` | `initPrometheus` guard comment. The claim that webhook-manager/webhook-control make "a fanout-bound `NewNotifyHandler` call in the same process" is **false** (issue §3, design §2.1 bullet 2). Correct it. |
| `main.go:64-66` | `promTopicPublishTotal`/`promTopicPlaceholderTotal` var-block comment ("the topic publish path must never touch `promNotifyTotal`/`promNotifyProcessTime`"). A2/A5 make both of those statements false. Rewrite to the new invariant per design §2.1 bullet 3: one active publish path per instance, so double-counting is impossible by construction. *Verified this plan:* the comment is at `main.go:65-66`; locate by content, not by line. |

#### A2. `publish.go` -- `publishEvent()` control-flow change (design §2.2)

- File: `bin-common-handler/pkg/notifyhandler/publish.go`, `publishEvent()` (*verified this plan:* declared at `publish.go:137`).
- Replace the current two-case `switch` with the three-case form in design §2.2's "New" snippet: `case delay > 0` (unchanged, evaluated FIRST -- design §8 depends on this ordering), `case h.topicEnabled` (new topic-only path, calling `publishTopicEventOrErr` and returning its error **unwrapped**, per design §2.2's R2 finding 11 note), `default` (fanout-only, unchanged).
- `promNotifyTotal.WithLabelValues(evt.Type).Inc()` moves to after the switch, covering both live paths (design §2.2 snippet).
- Delete the old trailing `h.publishTopicEvent(evt, subscriptionID)` call.
- `publishDirectEvent` / `publishDirectEventWithKey` are **not** modified (design §2.2).

#### A3. `main.go` -- `NewNotifyHandler` construction-time change (design §2.3)

- Wrap the existing `sockHandler.TopicCreate(string(queueEvent))` in `if !h.topicEnabled { ... }` (design §2.3 snippet). *Verified this plan:* `NewNotifyHandler` is at `main.go:192`.
- On declare failure inside that branch: `logrus.Fatalf` instead of log-and-return-nil (design §2.3's Decision).
- `h.initGlobalTopicExchange()` call site is **UNCHANGED** -- it keeps relying on the method's own internal `!h.topicEnabled` guard, so `NewNotifyHandlerForExistingExchange`'s call at `main.go:250` is unaffected (design §2.3's inline comment, and design §2.4's rationale for not refactoring to a bool-returning form).
- Signature unchanged: `queueEvent`/`h.queueNotify` are **kept** (design §2.3's "`queueEvent`/`h.queueNotify` field" paragraph).
- Add the doc comment design §2.3 specifies: "`queueEvent` is ignored on the `WithGlobalTopicPublish()` path -- no per-service fanout exchange is declared or published to there."

#### A4. `main.go` -- `initGlobalTopicExchange` failure semantics (design §2.4)

- *Verified this plan:* `initGlobalTopicExchange` is at `main.go:269`, its `!h.topicEnabled` guard at `:270`, the `h.topicDisabled = true` assignment at `:276`.
- Replace the degrade branch with `logrus.Fatalf` (design §2.4 snippet), carrying the doc comment that snippet includes.
- **Delete the `topicDisabled` field** (`main.go:163-168`) and every read of it. Known reads to remove: `publish.go:200` (`if h.topicDisabled`) and the `promTopicPublishTotal{result="error"}` "suppressed publish" increment at `publish.go:203` (design §2.4, design §7 item 6).
- Design §7 item 6 requires an explicit post-change sweep: `grep -rn topicDisabled` and confirm **zero** survivors across the repo (including tests -- see A7).

#### A5. `publish.go` -- `publishTopicEvent` -> `publishTopicEventOrErr` (design §2.5)

- *Verified this plan:* current `publishTopicEvent` at `publish.go:190`, its now-obsolete "reusing them would pollute the existing fanout metrics" comment at `publish.go:186-189`.
- Rewrite to the design §2.5 snippet: returns `error`, observes `promNotifyProcessTime` (same metric name and `type` label, design §2.5's Decision), keeps `promTopicPlaceholderTotal` / `promTopicPublishTotal` behavior, and folds `routing_key` into the returned error string (design §2.5's "Diagnostic-log fidelity" paragraph).
- The internal `if !h.topicEnabled { return }` guard at `publish.go:191` is no longer the gate (the `switch` in A2 is); follow design §2.5's snippet, which does not carry it.
- Carry an updated doc comment explaining why observing `promNotifyProcessTime` is now correct (design §2.1 bullet 3, last sentence).
- **Do not touch** `PublishEvent`'s existing `if h.topicEnabled` subscription-identifier gate (*verified this plan:* `publish.go:99`) -- the design does not change it.

#### A6. `main.go` -- webhook-manager option-surface safeguard (design §2.6)

- Replace `NewNotifyHandlerForExistingExchange`'s existing warning comment with the strengthened text in design §2.6's snippet. **Comment only; no code-level guard** (design §2.6's Decision). *Verified this plan:* `NewNotifyHandlerForExistingExchange` is at `main.go:229`.

#### A7. Test rewrites, deletions, and additions (design §8)

> **Citation drift note.** Design §8's `main_test.go` line citations carry a small offset against the current worktree (*verified this plan:* `Test_NewNotifyHandler_globalTopicDeclareFailure` is at `main_test.go:261`, design cites `:258-294`; `Test_NewNotifyHandler_withoutOption` is at `:297`, design cites `:296-318`; `Test_WithGlobalTopicPublish_declaresGlobalExchange` is at `:187`). `publish_test.go`'s citations point at *fixture* lines inside tests and check out consistently. **Locate every test below by NAME, and treat the design's line numbers as a cross-check, not a lookup key.**

**`main_test.go`:**

| Test | Design §8 citation | Action |
|---|---|---|
| `Test_NewNotifyHandler_globalTopicDeclareFailure` | `:258-294` (actual `:261`) | **REWRITE.** Currently pins the exact degrade-not-abort contract §2.4 deletes. Under `logrus.Fatalf` it would kill the whole test binary if left as-is. Rewrite to the `ExitFunc`-override strategy (design §8 bullet 1): override `logrus.StandardLogger().ExitFunc` to a non-exiting stub, attach a hook/formatter capturing the entry, assert it was logged at `logrus.FatalLevel` with the expected message. |
| `Test_WithGlobalTopicPublish_declaresGlobalExchange` -- `topicDisabled` assertions | `:251-253` | **DELETE** those assertions along with the field (A4). |
| `Test_WithGlobalTopicPublish_declaresGlobalExchange` -- `"new notify handler"` subtest | `:199-210`, field `expectFanoutDeclare` at `:209`, consumed at `:232-233` | **DROP the `TopicCreate` expectation entirely** for the `topicEnabled=true` construction (A3 moves that call inside `if !h.topicEnabled`). Both subtests become "no fanout declare", so the `expectFanoutDeclare` table field itself is vestigial -- **remove the field from the table**, do not leave it dangling (design §8, R4 finding). |
| `Test_NewNotifyHandler_withoutOption` | `:296-318` (actual `:297`) | **KEEP AS-IS.** Design §8 explicitly says this is the already-passing form of regression pin #2 (the `voip-asterisk-proxy` bit-identical-behavior contract) -- cite it, do not rewrite it. |

**New in `main_test.go`:** a test for `initGlobalTopicExchange`'s fatal path (A4), same `ExitFunc`-override strategy, plus the two **regression pins** design §8 names:

- Pin 1: `NewNotifyHandlerForExistingExchange` with `topicEnabled=false` still does NOT declare `bin-manager.event` and still returns a non-nil handler.
- Pin 2: satisfied by the retained `Test_NewNotifyHandler_withoutOption` (above).

**`publish_test.go` -- design §8 states this is a COMPLETE accounting (9 invalidated fixtures), not a partial list requiring a further sweep:**

| Fixture | Design §8 citation | Action |
|---|---|---|
| Dual-publish table test (`gomock.InOrder(fanout, topic)`) | `~:400-446`, `topicEnabled: true` at `:413` (inside `Test_PublishEvent_globalTopicPublish`, actual func `:318`) | **REWRITE** to expect a single topic-only `EventPublish`. |
| `Test_publishEvent_globalTopicPublishFailureIsolated` | `:536-579`, assertions at `:569-571` | **REWRITE with inverted semantics.** Not mechanical: the topic-path error is now RETURNED to the caller, not swallowed. Both the test name and the assertion direction flip (design §8). |
| `Test_publishEvent_fanoutFailureSkipsTopic` | `:583-603` | **DELETE.** The scenario (fanout attempted at all when topic-enabled) ceases to exist. Design §8: "do not attempt to adapt it." |
| `Test_PublishEvent_globalTopicPublishPlaceholderMetric` | fixture `:516` | **MECHANICAL:** delete the now-nonexistent fanout expectation. |
| `Test_PublishEventRaw_globalTopicPublish` | fixture `:715` | **MECHANICAL:** same. |
| `Test_PublishEvent_nilInterface` | fixture `:819` | **MECHANICAL:** same. |
| `Test_PublishEvent_emptyAddressIgnoresPayloadID` | fixture `:887` | **MECHANICAL:** same. |
| `Test_PublishEvent_typedNilSubscriptionIdentifier` | fixture `:778` | **STRUCTURAL:** two-arm table with a shared, unconditional fanout `EventPublish` expectation. Post-A2 the arms publish to different exchanges. **Split the shared expectation into the option-off arm only** -- not a one-line deletion (design §8). |
| `Test_PublishEvent_optionOffSkipsSubscriptionIdentifier` | fixture `:957` | **STRUCTURAL:** same shape, same treatment. |
| `Test_publishEvent_delayedSkipsTopic` | fixture `:638` (actual func `:628`) | **NO CHANGE -- explicitly confirmed by design §8.** A2 keeps `case delay > 0` before `case h.topicEnabled`, so the `.Times(0)` assertion at `:642` still holds. Verify it still passes; do not edit it. |
| `Test_publishEvent_optionOffSkipsTopic` | `:607` | **NO CHANGE** -- sets no `topicEnabled` field, so stays `false`, unaffected (design §8). |

**New in `publish_test.go`** (design §8 bullet 1): table-driven coverage of `publishEvent()`'s new branch split (`topicEnabled` true/false × `delay` zero/nonzero = 4 cells), and `publishTopicEventOrErr`'s metric observations, extending the existing `promNotifyProcessTime`/`promTopicPublishTotal` assertions onto the topic-only path.

#### A8. Verification for Unit A

```bash
cd ~/gitvoipbin/monorepo/.worktrees/VOIP-1407-Cutover-remove-fanout-dual-publish/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Plus the design §7 item 6 sweep:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/VOIP-1407-Cutover-remove-fanout-dual-publish
/usr/bin/grep -rn "topicDisabled" --include="*.go" .        # must return zero
/usr/bin/grep -rn "publishTopicEvent\b" --include="*.go" .  # must return zero (renamed)
```

> **Methodology constraint carried from issue §3:** every enumeration/sweep in this implementation phase MUST use `/usr/bin/grep`, **not** the IDE-integrated Grep tool, which was independently observed by two reviewers to silently drop at least one match (`bin-pipecat-manager/cmd/pipecat-manager/main.go:125`). Re-verify results against the issue-analysis counts rather than trusting either blindly.

---

### Unit B -- `bin-talk-manager`: the `talk-control` companion fix

**Design authority: §2.3.1 (and §4's disposition row). Test authority: §8 bullet "`bin-talk-manager/cmd/talk-control`".**

#### B1. Fix the construction

- File: `bin-talk-manager/cmd/talk-control/main.go:57`.
- Replace the current `notifyhandler.NewNotifyHandler(sockHandler, nil, "", serviceName)` with the exact two-line form in design §2.3.1's Decision snippet: construct a real `reqHandler` via `commonreq.NewRequestHandler(sockHandler, commonoutline.ServiceNameTalkManager)`, pass `commonoutline.QueueNameTalkEvent` as `queueEvent`, and add `notifyhandler.WithGlobalTopicPublish()`.
- Add the two imports design §2.3.1 names: `monorepo/bin-common-handler/models/outline` aliased `commonoutline`, and `monorepo/bin-common-handler/pkg/requesthandler` aliased `commonreq`. Design §2.3.1 confirms neither is currently imported and there is no alias collision -- re-confirm at edit time.
- Reference shape: `bin-talk-manager/cmd/talk-manager/main.go:81-88` (the daemon).
- **Explicitly NOT changed** (design §2.3.1's last paragraph): `serviceName`'s untyped string constant type. Leave it.

#### B2. Tests / verification for Unit B

Design §8 is explicit that there is **no unit-test harness** around `cmd/talk-control/main.go`'s `initHandlers()`, so this is a build + manual smoke check:

1. Standard workflow:
   ```bash
   cd bin-talk-manager
   go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
   ```
2. Manual smoke (design §8): run `talk-control chat list` against a real environment and confirm it exits 0 rather than calling `os.Exit(1)` during startup. This confirms both that `initHandlers()` no longer reaches `logrus.Fatalf` and that the `reqHandler` fix did not break construction.
3. Manual smoke, **best-effort** (design §8 marks this explicitly non-binding): run a write subcommand (e.g. `talk-control chat create`) and confirm `bin-manager.talk-manager.event` gains zero new bindings/messages. Because `PublishWebhookEvent` is fire-and-forget across two goroutines the CLI does not wait on (design §2.3.1's "Remaining caveat"), allow retry/wait margin and do **not** treat a single fast check as conclusive in either direction.

---

### Unit C -- the 3 dead-wiring deletions

**Design authority: §4's table rows 3 and 4, and §4's "Decision on the 3 dead-wiring deletions."**

| # | File |
|---|---|
| 1 | `bin-transfer-manager/cmd/transfer-manager/main.go:137` |
| 2 | `bin-transfer-manager/cmd/transfer-control/main.go:66` |
| 3 | `bin-tts-manager/cmd/tts-control/main.go:38` |

Per design §4: delete "the `notifyHandler` field, its constructor parameter, and both `NewNotifyHandler` construction sites" (plus the now-unused imports).

**Scope note (not a new decision -- a size disclosure).** *Verified this plan:* `bin-transfer-manager/pkg/transferhandler/main.go` holds the `notifyHandler` field, and **8 test files** in `pkg/transferhandler` reference it (`attended_test.go`, `blind_test.go`, `db_test.go`, `service_start_test.go`, `transfer_test.go`, `transferee_test.go`, `transferer_comprehensive_test.go`, `transferer_test.go`). Removing the field and the constructor parameter therefore ripples mechanically into all of them (drop the mock-notify argument from each `transferHandler` construction, drop any `mockNotify` declarations that become unused). This is the direct, unavoidable mechanical consequence of design §4's instruction, not extra scope -- but it is materially larger than the word "trivial" in design §4 implies, and the implementer should budget for it. See §8 OQ-2.

For `bin-tts-manager`: design §4 and issue §3 both state `pkg/ttshandler` makes zero publish calls. Re-verify at edit time whether `pkg/ttshandler` carries a `notifyHandler` field at all (design §4's row says only "`pkg/ttshandler` never calls it"); if it does not, the deletion is confined to `cmd/tts-control/main.go`.

**Verification (run from EACH service directory):**

```bash
cd bin-transfer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-tts-manager      && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Unit D -- `bin-webhook-manager`: comment-only

**Design authority: §2.6, and §4's table row 6 ("Comment-only change").**

- Files: `bin-webhook-manager/cmd/webhook-manager/main.go`, `bin-webhook-manager/cmd/webhook-control/main.go`.
- **No code change at either call site.** Both keep `NewNotifyHandlerForExistingExchange` with no `WithGlobalTopicPublish()`.
- The strengthened warning comment itself lives in `bin-common-handler` (A6). Design §4 classifies the webhook-manager *disposition* as comment-only; if any local explanatory comment at these two call sites describes the old dual-publish meaning, update it to match A6's text. Otherwise this unit is a no-op verification pass confirming these two sites were correctly left alone.

**Note:** `bin-webhook-manager` also appears in Unit E (it is one of the 20 consumer services). Those are different files (`pkg/subscribehandler` + `cmd/webhook-manager`'s `strings.Join`); do not conflate the two units.

**Verification:**

```bash
cd bin-webhook-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Unit E -- the 20 VOIP-1406 consumer services

**Design authority: §3.1 (the four deletions + the variance table), §3.2 (the two exception mechanisms), §3.3 (fatal failure semantics + the canonical `Run()` snippet). Test authority: §8's consumer bullet.**

The 20 services (issue §2b, design §1(b)): agent, ai, billing, call, campaign, conference, contact, conversation, direct, flow, number, queue, registrar, schedule, storage, tag, timeline, transcribe, transfer, webhook.

#### E1. The generic edit (applies to all 20, modulo §3.1's variance table)

In `pkg/subscribehandler/main.go`, delete:

1. Whatever constructs the fanout target list feeding `Run()`'s subscribe loop (design §3.1 item 1 -- shape varies, see E2).
2. The fanout `QueueSubscribe` loop in `Run()` (design §3.1 item 2). **Carrying with it the sentinel defensive `TopicCreate`**, which *verified this plan* sits INSIDE that loop as an `if target == QueueNameSentinelEvent` special case (`bin-call-manager/pkg/subscribehandler/main.go:161-162`, `bin-timeline-manager/pkg/subscribehandler/main.go:153-154`) -- so deleting the loop deletes the sentinel declare automatically, exactly as design §3.2 requires. Do not re-add it.
3. `fanoutUnbindTargets` and the `QueueUnbind` step that consumed it (design §3.1 item 3). *Verified this plan:* `fanoutUnbindTargets` is a package-level var even in otherwise-typical services (e.g. `bin-call-manager/pkg/subscribehandler/main.go:53-57`, `bin-timeline-manager/pkg/subscribehandler/main.go:67-75`), separate from `subscribeTargets`' shape -- delete both, independently.
4. In `cmd/*-manager/main.go`: the slice literal and the constructor-parameter wiring (design §3.1 item 4). Does NOT apply to timeline-manager (no `cmd/` wiring exists) and takes a different form for webhook-manager (see E2).

Then apply design §3.3:

- `topicPatterns` + its `QueueBind` loop become the **sole** intake path, unconditional, not gated on any "if the fanout declare failed, stay on fanout" branch.
- Every failure in `Run()`'s startup sequence returns the error immediately (`return fmt.Errorf(...)`), matching design §3.3's snippet.
- **Delete the "bound so far, roll back on partial failure" machinery entirely** (design §3.3's closing paragraph). *Verified this plan:* this is the per-pattern rollback `QueueUnbind` at e.g. `bin-call-manager/pkg/subscribehandler/main.go:206` and `bin-timeline-manager/pkg/subscribehandler/main.go:207`.

#### E2. Per-service variance (design §3.1's corrected table)

| Group | Services | `subscribeTargets` shape | What to delete |
|---|---|---|---|
| Typical (18) | agent, ai, billing, call, campaign, conference, contact, conversation, direct, flow, number, queue, registrar, schedule, storage, tag, transcribe, transfer | `cmd/*-manager/main.go`-local slice literal (design cites `bin-call-manager/cmd/call-manager/main.go:180-185`) passed as a constructor param into a `pkg/subscribehandler` struct field (design cites `bin-call-manager/pkg/subscribehandler/main.go:63,121,130`) | The `cmd/`-local literal + the constructor parameter + the struct field. Any `string(commonoutline.QueueName*Event)` conversions used to build the slice disappear with it -- nothing extra. |
| timeline-manager | timeline | Package-level `var subscribeTargets = []commonoutline.QueueName{...}` (`bin-timeline-manager/pkg/subscribehandler/main.go:27`, typed `QueueName`, not `[]string`); **no `cmd/` wiring** (`NewSubscribeHandler(sockHandler, dbHandler)`, `main.go:119-128`) | The package-level var only. **No item-4 counterpart.** |
| webhook-manager | webhook | Comma-joined `string` constructor param (`pkg/subscribehandler/main.go:97`), split via `strings.Split(h.subscribesTargets, ",")` at `main.go:126`; built with `strings.Join([]string{...}, ",")` in `cmd/webhook-manager/main.go:193` | The `strings.Join` construction in `cmd/`, the constructor parameter, AND the `strings.Split` call in `Run()`. |

**timeline-manager additionally**: its `Run(ctx) (<-chan struct{}, error)` is two-value (`pkg/subscribehandler/main.go:109,134`), so **every** `return err` in design §3.3's snippet becomes `return nil, err` for this service only (design §3.1 variance table, design §3.3's preamble). *Verified this plan:* `func (h *subscribeHandler) Run(ctx context.Context) (<-chan struct{}, error)` at `bin-timeline-manager/pkg/subscribehandler/main.go:134`.

#### E3. The 3 exception services -- handle individually, NOT in the sweep

**Design authority: §3.2. This is the single highest-risk item in the whole PR -- design §3.2 opens by noting an earlier draft omitted one of these entirely and would have caused webhook-event intake loss in two production services.**

| Service | Exception(s) it carries | Exact requirement |
|---|---|---|
| **call-manager** | asterisk fanout leg (has it); VOIP-1258 webhook block (does NOT have it) | Deleting the loop must NOT remove the `QueueNameAsteriskEventAll` iteration. Re-add it as a **standalone** `QueueSubscribe(subscribeQueue, string(commonoutline.QueueNameAsteriskEventAll))` statement **in the exact position the loop used to occupy** -- immediately after `QueueCreate`, BEFORE the topic-declare line. Design §3.2 explicitly corrects an earlier "immediately before `go ConsumeMessage`" placement as wrong for this service's actual source layout. Failure semantics: fatal (`return fmt.Errorf`), per design §3.3's snippet. |
| **agent-manager** | VOIP-1258 webhook block (has it); asterisk leg (does NOT have it) | The `QueueBind("#", QueueNameWebhookEventTopic) / else if QueueUnbind(QueueNameWebhookEvent)` block at `bin-agent-manager/pkg/subscribehandler/main.go:97-120` is **RETAINED VERBATIM AND IN POSITION** (between where the deleted fanout loop was and the `topicPatterns` block). Its **log-only, non-fatal** failure semantics are **NOT** promoted to fatal (design §3.2 sub-ruling 1). Its legacy `QueueUnbind(..., QueueNameWebhookEvent, ...)` call is **left untouched** (design §3.2 sub-ruling 2). |
| **timeline-manager** | **BOTH** | Both of the above, plus the two-value `Run()` signature change from E2. |

Design §3.3's snippet is the canonical target shape; it shows the *union* of both exceptions for illustration -- only timeline-manager carries both. The other 17 services skip straight from `QueueCreate` to the topic-declare line.

#### E4. Per-service test work

Design §8's consumer bullet specifies the target shape:

- `binding_golden_test.go`: drop the fanout-target pins. *Verified this plan:* all 20 services have this file.
- The `Run()` sequencing test's `gomock.InOrder` becomes: `QueueCreate` -> [asterisk `QueueSubscribe`, call-manager/timeline-manager only] -> [VOIP-1258 webhook `QueueBind`+`QueueUnbind`, agent-manager/timeline-manager only, **asserting §3.2's unchanged log-only failure behavior is preserved, not just its happy path**] -> `TopicCreateWithKind` -> `QueueBind`×N -> `ConsumeMessage`.
- Failure-path cases assert `Run()` returns the error immediately (`return nil, err` for timeline-manager, `return err` elsewhere), replacing the old roll-back-and-degrade assertions.

Design §8 states this requirement generically rather than naming each service's tests. The concrete inventory below is **derived** from that requirement by locating every existing `Run()`-related test in the worktree (*verified this plan*); it introduces no new decision, and a reviewer should check the derivation rather than take it on trust (flagged as OQ-3).

| Service | Test files / functions | Derived action |
|---|---|---|
| agent | `main_test.go`: `Test_Run_BindsTopicExchangeBeforeReturning:27`, `Test_Run_QueueBindFailure_DoesNotUnbind:160`; `binding_golden_test.go` | Rewrite `:27` to the new InOrder (retaining the VOIP-1258 block assertions). `:160` pins "bind failure does not unbind fanout" -- rework to "bind failure returns the error immediately" per §3.3. Add a case asserting the VOIP-1258 block's log-only behavior survives. |
| ai | `main_test.go`: `Test_Run_sequencing:23`; `binding_golden_test.go` | Mechanical rewrite. |
| billing | `run_sequencing_test.go`: `_success:35`, `_declareFailure:73`, `_bindFailure_rollsBackPartialBinds:106`, `_fanoutUnbindFailure_continues:150`; `binding_golden_test.go` | `:35` rewrite; `:73` confirm; `:106` rework -- **no rollback exists any more**, assert immediate error return; `:150` **DELETE**. `main_test.go`'s `processEventRun*` tests untouched. |
| call | `main_test.go`: `Test_Run:19`, `Test_Run_error:90`, `Test_Run_topicDeclareFails:161`, `Test_Run_topicBindFails:203`, `Test_Run_fanoutUnbindFails:278`; `binding_golden_test.go` | `:19` rewrite **retaining the standalone asterisk `QueueSubscribe`** in the InOrder; `:90` rewrite; `:161`/`:203` rework to assert immediate error return; `:278` **DELETE**. |
| campaign | `main_test.go`: `Test_Run_BindsTopicExchangeBeforeConsuming:23`; `binding_golden_test.go` | Mechanical rewrite. |
| conference | `main_test.go`: `Test_Run_sequencing:23`; `binding_golden_test.go` | Mechanical rewrite. |
| contact | `run_sequencing_test.go`: `_success:30`, `_declareFailure:68`, `_bindFailure_noRollbackNeeded:101`, `_fanoutUnbindFailure_continues:135`; `subscribehandler_test.go`: `Test_Run_QueueCreateError:216`, `Test_Run_SubscribeError:238`, `Test_Run_Success:261`; `binding_golden_test.go` | `:30`/`:261` rewrite; `:68` confirm; `:101` rework; `:135` **DELETE**; `:216` keep (QueueCreate unchanged); `:238` **DELETE** (fanout `QueueSubscribe` gone). |
| conversation | `main_test.go`: `Test_Run_BindsTopicExchangeBeforeConsuming:152`; `binding_golden_test.go` | Mechanical rewrite. |
| direct | `main_test.go`: `Test_Run_sequencing:23`; `binding_golden_test.go` | Mechanical rewrite. |
| flow | `main_test.go`: `Test_Run_sequencing:23`; `binding_golden_test.go` | Mechanical rewrite. |
| number | `main_test.go`: `Test_Run_sequencing:23`; `binding_golden_test.go` | Mechanical rewrite. |
| queue | `main_test.go`: `Test_Run_sequencing:23`; `binding_golden_test.go` | Mechanical rewrite. |
| registrar | `main_test.go`: `Test_Run_sequencing:23`; `binding_golden_test.go` | Mechanical rewrite. |
| schedule | `main_test.go`: `Test_Run_sequencing:43`; `binding_golden_test.go` | Mechanical rewrite. |
| storage | `main_test.go`: `Test_Run_sequencing:23`; `binding_golden_test.go` | Mechanical rewrite. |
| tag | `main_test.go`: `Test_Run_sequencing:135`; `error_test.go`: `Test_Run_QueueCreateError:14`, `Test_Run_QueueSubscribeError:36`; `binding_golden_test.go` | `:135` rewrite; `error_test.go:14` keep; `error_test.go:36` **DELETE**. |
| timeline | `run_topic_migration_test.go`: `:27`, `_DeclareFailure:94`, `_BindFailure:133`, `_FanoutUnbindFailure:172`; `run_sentinel_test.go`: `:22`, `:79`; `run_ordering_test.go`: `:26`; `binding_golden_test.go` | `:27` rewrite (**both** exceptions in the InOrder, two-value `Run()`); `:94`/`:133` rework to `return nil, err`; `:172` **DELETE**; `run_sentinel_test.go` **DELETE ENTIRELY** (the sentinel defensive declare it pins is removed by E1 item 2); `run_ordering_test.go:26` rewrite. `main_test.go`'s `processEventRun*` untouched. |
| transcribe | `main_test.go`: `Test_Run_sequencing:25`; `binding_golden_test.go` | Mechanical rewrite. |
| transfer | `main_test.go`: `Test_Run_sequencing:110` (`TestProcessEventRun:59` untouched); `binding_golden_test.go` | Mechanical rewrite. **Note:** transfer-manager is ALSO in Unit C -- keep the two edits separate. |
| webhook | `main_test.go`: `Test_Run_TopicMigrationSequencing:23`; `binding_golden_test.go` | Rewrite, accounting for the `strings.Split` deletion from E2. |

Also delete the `QueueNameWebhookEvent` gomock expectations that become unreachable **only if** their surrounding tests are deleted. Issue §4 item 1 catalogues 8 such expectations (`bin-agent-manager/pkg/subscribehandler/main_test.go:76,174`; `bin-timeline-manager/pkg/subscribehandler/run_topic_migration_test.go:49,109,148,191`; `run_sentinel_test.go:49`; `run_ordering_test.go:48`). **Design §7 item 2 explicitly does NOT delete the `QueueNameWebhookEvent` constant in this PR**, and design §3.2 sub-ruling 2 keeps its `QueueUnbind` call site -- so surviving tests keep their expectations. Only expectations inside tests this plan deletes go away with them.

#### E5. Verification for Unit E

```bash
for s in agent ai billing call campaign conference contact conversation direct flow \
         number queue registrar schedule storage tag timeline transcribe transfer webhook; do
  ( cd bin-$s-manager \
    && go mod tidy && go mod vendor && go generate ./... \
    && go test ./... && golangci-lint run -v --timeout 5m ) \
    || echo "FAILED: bin-$s-manager"
done
```

Plus a residue sweep:

```bash
/usr/bin/grep -rn "fanoutUnbindTargets\|subscribesTargets" --include="*.go" .
# expected survivors: NONE in bin-*/pkg/subscribehandler
/usr/bin/grep -rn "subscribeTargets" --include="*.go" .
# expected survivors: bin-api-manager only (see OQ-1)
```

---

### Unit F -- Documentation

**Design authority: §6 (plus §5 for the runbook content this PR writes but does not execute).**

#### F1. `docs/reference/rabbitmq-queues-reference.md`

Per design §6, rewrite the dual-publish framing throughout, specifically these sections design §6 names: the **routing-key section**, the **publish-path section**, the **declaration-invariant section**, and the **"Consumer state (VOIP-1406)"** note -- for a topic-only world across the 55+20 services this ticket touches.

Additionally:

- Document `asterisk.all.event` / `voip-asterisk-proxy` as a **permanently-retained, deliberately-excluded exception** -- explicitly not migration debris (design §6, design §0).
- Add the §5 exchange-deletion runbook: `curl` against the management API on **port 80** (not 15672) on bm-nyc-01, matching the existing runbook pattern already at `docs/reference/rabbitmq-queues-reference.md:301-312` -- **NOT `rabbitmqadmin`**, which that doc does not use (design §5's Mechanism; design §3.4's R2 finding 5 correction). Per-exchange form: `curl -u "$RABBITMQ_USER:$RABBITMQ_PASS" -X DELETE "http://<host>/api/exchanges/%2f/bin-manager.<x>-manager.event"`.
- **Enumerate all 28 exchange names VERBATIM** in the runbook -- design §5 is explicit that describing them "by range" is not acceptable. The normative list is issue §1's 28-name `for ex in ...` enumeration; copy it, do not re-derive it.
- Document the runbook's precondition (design §5: every publisher daemon + control binary AND all 20 consumer services redeployed, with a per-service confirmed restart-survival check) and its **post-deletion re-check** after a defined soak window.
- Document the §3.4 pre-rollout AND post-rollout stale-binding sweeps, including why the post-rollout sweep is required (the automatic self-heal disappears with the `QueueUnbind` loop; manual remediation via the `curl` procedure at `:301-312` still works but is no longer optional).
- Note (design §5's exclusion list) the exchanges explicitly NOT deleted: `asterisk.all.event`, `bin-manager.event`, `bin-manager.webhook-manager.event.topic`, `bin-manager.delay`, and the 5 already-absent `QueueName*Event`-family exchanges.

#### F2. Per-service docs

Design §6: publish-side prose for the ~27 publisher services' `docs/architecture.md`, events-section prose for the 20 consumer services' `docs/architecture.md`, and `docs/dependencies.md` where affected -- **union, not sum**, since most services are both.

Root `CLAUDE.md`'s service-docs-sync table makes two mandatory:

- `cmd/*/main.go` or `pkg/subscribehandler/main.go` (subscribeTargets) changed → `docs/architecture.md` events section MUST be updated in the same commit.
- `go.mod` replace directives changed → `docs/dependencies.md`.

Use the repo's own extractor rather than hand-editing generated sections:

```bash
bash docs/reference/extractor.sh bin-<service>
```

The PostToolUse hook `scripts/check-service-docs.sh` warns (does not block) when these source files change without a matching docs update -- stage the docs alongside the source, and treat any surviving warning as a missed doc.

#### F3. `bin-common-handler/docs/architecture.md`

Update the `notifyhandler` section to describe the topic-only default and the `WithGlobalTopicPublish()` **meaning change** (design §6's last sentence): the option now means "topic-ONLY, fanout removed," and the option-omitted default remains fanout-only (the `voip-asterisk-proxy` case).

#### F4. Verification for Unit F

- Docs-only; no Go verification needed for F1/F3.
- For F2, per-service docs are staged with that service's code commit, so they are covered by that service's already-run workflow.
- Confirm no RST work is triggered: this change is entirely internal message-bus plumbing with **no user-visible API/webhook/billing surface change**, so root `CLAUDE.md`'s RST-sync rule does not apply. State this explicitly in the PR body so a reviewer does not have to re-derive it.

---

## 4. Cross-cutting verification after all units land

### 4.1 Full-repo build/test/lint across every touched service

Touched Go modules (23 distinct directories; `bin-transfer-manager` and `bin-webhook-manager` appear in two units each but are one module apiece):

```bash
cd ~/gitvoipbin/monorepo/.worktrees/VOIP-1407-Cutover-remove-fanout-dual-publish
for d in bin-common-handler bin-talk-manager bin-tts-manager \
         bin-agent-manager bin-ai-manager bin-billing-manager bin-call-manager \
         bin-campaign-manager bin-conference-manager bin-contact-manager \
         bin-conversation-manager bin-direct-manager bin-flow-manager \
         bin-number-manager bin-queue-manager bin-registrar-manager \
         bin-schedule-manager bin-storage-manager bin-tag-manager \
         bin-timeline-manager bin-transcribe-manager bin-transfer-manager \
         bin-webhook-manager; do
  echo "=== $d ==="
  ( cd "$d" && go mod tidy && go mod vendor && go generate ./... \
      && go test ./... && golangci-lint run -v --timeout 5m ) || echo "FAILED: $d"
done
```

**Blast-radius check (not optional).** `bin-common-handler` is consumed by every service via a `replace` directive, and Unit A is a *behavioral* change to a shared library. Even services this PR does not edit compile against it. Run at minimum `go build ./...` in every remaining `bin-*`/`voip-*` module that constructs a `NotifyHandler` -- notably `voip-asterisk-proxy` (the excluded caller, whose behavior must be provably inert) and the ~27 publisher daemons' `-control` binaries. A repo-wide loop over every module directory is the safest form.

### 4.2 Traceability checklist -- every design §8 named item

Regression pins (design §8):

- [ ] Pin 1: `NewNotifyHandlerForExistingExchange` with `topicEnabled=false` still does NOT declare `bin-manager.event` and still returns a non-nil handler.
- [ ] Pin 2: a `topicEnabled=false` `NewNotifyHandler` construction still calls `TopicCreate(queueEvent)` and routes every publish through `publishDirectEvent`/`publishDirectEventWithKey`, **bit-identical to pre-PR behavior** (`Test_NewNotifyHandler_withoutOption`, retained unchanged).

`main_test.go` (design §8):

- [ ] `Test_NewNotifyHandler_globalTopicDeclareFailure` REWRITTEN to `ExitFunc`-override + `logrus.FatalLevel` assertion.
- [ ] `Test_WithGlobalTopicPublish_declaresGlobalExchange`: `topicDisabled` assertions removed.
- [ ] `Test_WithGlobalTopicPublish_declaresGlobalExchange`: `"new notify handler"` subtest's `TopicCreate` expectation DROPPED, and the `expectFanoutDeclare` table field REMOVED (not left dangling).
- [ ] `Test_NewNotifyHandler_withoutOption` UNCHANGED and still passing.
- [ ] NEW: `initGlobalTopicExchange` fatal-path test.

`publish_test.go` -- all 9 invalidated fixtures + 2 explicit non-changes (design §8's "complete accounting"):

- [ ] Dual-publish table test (`:413`) rewritten to a single topic-only `EventPublish`.
- [ ] `Test_publishEvent_globalTopicPublishFailureIsolated` (`:536`) rewritten with INVERTED semantics (error now reaches the caller) + renamed.
- [ ] `Test_publishEvent_fanoutFailureSkipsTopic` (`:583`) DELETED.
- [ ] `Test_PublishEvent_globalTopicPublishPlaceholderMetric` (`:516`) fanout expectation deleted.
- [ ] `Test_PublishEventRaw_globalTopicPublish` (`:715`) fanout expectation deleted.
- [ ] `Test_PublishEvent_nilInterface` (`:819`) fanout expectation deleted.
- [ ] `Test_PublishEvent_emptyAddressIgnoresPayloadID` (`:887`) fanout expectation deleted.
- [ ] `Test_PublishEvent_typedNilSubscriptionIdentifier` (`:778`) shared expectation STRUCTURALLY SPLIT into the option-off arm only.
- [ ] `Test_PublishEvent_optionOffSkipsSubscriptionIdentifier` (`:957`) same structural split.
- [ ] `Test_publishEvent_delayedSkipsTopic` (`:638`) UNCHANGED and still passing (confirms `case delay > 0` still precedes `case h.topicEnabled`).
- [ ] `Test_publishEvent_optionOffSkipsTopic` (`:607`) UNCHANGED and still passing.
- [ ] NEW: table-driven `publishEvent()` branch-split coverage (topicEnabled × delay, 4 cells).
- [ ] NEW: `publishTopicEventOrErr` metric-observation coverage (`promNotifyProcessTime` + `promTopicPublishTotal` on the topic-only path).

Consumer side (design §8's consumer bullet):

- [ ] All 20 `binding_golden_test.go` files updated (fanout pins dropped).
- [ ] All 20 `Run()` sequencing tests rewritten to the new InOrder shape.
- [ ] Failure-path cases assert immediate error return (`return nil, err` for timeline-manager; `return err` for the other 19).
- [ ] agent-manager and timeline-manager: a case asserting the VOIP-1258 block's **log-only, non-fatal** failure behavior is preserved (design §8 requires this explicitly, "not just its happy path").
- [ ] `bin-timeline-manager/pkg/subscribehandler/run_sentinel_test.go` DELETED.

`talk-control` (design §8):

- [ ] Build + smoke: `talk-control chat list` exits 0.
- [ ] Best-effort: a write subcommand produces zero new `bin-manager.talk-manager.event` bindings/messages (with retry/wait margin; not conclusive on a single fast check).

Residue sweeps:

- [ ] `/usr/bin/grep -rn "topicDisabled" --include="*.go" .` returns zero (design §7 item 6).
- [ ] `/usr/bin/grep -rn "publishTopicEvent\b" --include="*.go" .` returns zero.
- [ ] `/usr/bin/grep -rn "fanoutUnbindTargets" --include="*.go" .` returns zero.

---

## 5. Explicit scope boundary -- what this plan does NOT include

- **Design §5's 28-exchange deletion is NOT executed by this PR.** It is an operational, post-merge, post-full-deployment runbook. This PR only **writes** that runbook into `docs/reference/rabbitmq-queues-reference.md` (F1). No broker mutation is performed at any point in this plan. Handing off the runbook after deployment is a separate, post-merge activity gated on design §5's precondition (every publisher and consumer redeployed, per-service restart-survival confirmed) and design §3.4's pre/post-rollout stale-binding sweeps.
- **`voip-asterisk-proxy` is not touched at all** (design §0/§2.1/§4). Zero edits to `voip-asterisk-proxy/cmd/asterisk-proxy/main.go:107`. Its behavior must be provably inert -- that is exactly what regression pin #2 exists to prove.
- **`asterisk.all.event` and its two consumer bindings are permanent** (design §3.2).
- **VOIP-1258's runtime path stays out** (design §1's "Out of scope"): the `NewNotifyHandlerForExistingExchange` / `PublishEventWithRoutingKey` path is confirmed orthogonal; only its option-surface comment is touched (A6/D).
- **Design §7's non-blocking open items stay OPEN, untouched by this PR:**
  - §7 item 2: the 5 vestigial `QueueName*Event` outline constants (4 fully dead + 1 legacy `QueueNameWebhookEvent`) are **NOT deleted**, and neither are the 8 gomock expectations or 4 doc mentions their deletion would touch.
  - §7 item 3: delayed-publish dead code (`publishDelayedEvent`, `DelaySecond`/`DelayMinute`/`DelayHour` in `notifyhandler`) is **NOT touched**.
  - §7 items 1, 4, 5, 6 are listed there for completeness and are already decided IN-scope; they are covered by A3, C, B, and A4 respectively.
- **`serviceName`'s untyped-string type at `talk-control`** stays as-is (design §2.3.1's closing paragraph).
- **No database, Alembic, or RST work.** No user-visible surface changes (F4).

---

## 6. PR plan

**Single PR** per root `CLAUDE.md`'s "one PR per task" rule. This whole ticket is one logical change spanning `bin-common-handler` + ~24 services + docs.

- **Branch:** `VOIP-1407-Cutover-remove-fanout-dual-publish` (already exists; worktree at `~/gitvoipbin/monorepo/.worktrees/VOIP-1407-Cutover-remove-fanout-dual-publish`).
- **PR title:** must match the branch name exactly: `VOIP-1407-Cutover-remove-fanout-dual-publish`.
- **Pre-PR gate** (root `CLAUDE.md`, mandatory, run from the worktree):
  ```bash
  git fetch origin main
  git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
  git log --oneline HEAD..origin/main
  ```
  If conflicts exist: rebase/merge, resolve, and **re-run the full verification workflow in every affected service** before proceeding.
- **Merge:** do NOT merge. A review loop ends at approval. Wait for the 대표님's explicit "merge" instruction, then `gh pr merge <n> --squash --delete-branch`.

### 6.1 PR body

No markdown headers (`## Summary` etc.), no generic test-plan section, no AI attribution. Narrative opener, then `bin-<service>:`-prefixed bullets:

```
Remove fanout dual publish and delete the per-service fanout consumer machinery, making
the global topic exchange bin-manager.event the sole event delivery path for every
publisher that opts into WithGlobalTopicPublish() and for all 20 VOIP-1406 consumer
services. voip-asterisk-proxy and asterisk.all.event are deliberately excluded and
unchanged.

- bin-common-handler: Make WithGlobalTopicPublish() mean topic-only publishing; remove
  the fanout leg from publishEvent()'s topic-enabled path
- bin-common-handler: Make the fanout-declare and global-topic-declare failure paths
  fatal via logrus.Fatalf instead of returning an unchecked nil handler
- bin-common-handler: Delete the topicDisabled degrade branch and its suppressed-publish
  metric accounting
- bin-common-handler: Refactor publishTopicEvent into publishTopicEventOrErr, returning
  the error to the caller and observing notify_process_time
- bin-common-handler: Strengthen the NewNotifyHandlerForExistingExchange warning against
  enabling WithGlobalTopicPublish()
- bin-talk-manager: Fix talk-control's empty exchange name and match the daemon's
  construction exactly (real reqHandler, QueueNameTalkEvent, WithGlobalTopicPublish)
- bin-transfer-manager: Delete dead notifyHandler wiring from transfer-manager and
  transfer-control
- bin-tts-manager: Delete dead notifyHandler wiring from tts-control
- bin-webhook-manager: Update the notify-handler option-surface comment for the new
  topic-only meaning
- bin-agent-manager: Delete the fanout QueueSubscribe loop, subscribeTargets, and
  fanoutUnbindTargets; retain the VOIP-1258 webhook-topic block verbatim
- bin-call-manager: Delete the fanout loop and sentinel defensive declare; retain the
  asterisk.all.event subscription as a standalone call in its original position
- bin-timeline-manager: Delete the fanout loop, package-level subscribeTargets, and
  sentinel declare; retain both the asterisk leg and the VOIP-1258 block
- bin-webhook-manager: Delete the comma-joined subscribesTargets wiring and its
  strings.Split in Run()
- bin-ai-manager: Delete the fanout QueueSubscribe loop, subscribeTargets wiring, and
  fanoutUnbindTargets/QueueUnbind step; make topic declare and bind failures fatal
- bin-billing-manager: (same)
- bin-campaign-manager: (same)
- bin-conference-manager: (same)
- bin-contact-manager: (same)
- bin-conversation-manager: (same)
- bin-direct-manager: (same)
- bin-flow-manager: (same)
- bin-number-manager: (same)
- bin-queue-manager: (same)
- bin-registrar-manager: (same)
- bin-schedule-manager: (same)
- bin-storage-manager: (same)
- bin-tag-manager: (same)
- bin-transcribe-manager: (same)
- bin-transfer-manager: (same)
- docs: Rewrite rabbitmq-queues-reference.md for a topic-only world, document
  asterisk.all.event as a permanent exception, and add the 28-exchange deletion runbook
```

(Expand each `(same)` into its own full sentence when writing the actual PR body; it is collapsed here only for plan readability.)

### 6.2 Testing / verification section for the PR description

```
Verification run (per service directory):
  go mod tidy && go mod vendor && go generate ./... && go test ./... && \
    golangci-lint run -v --timeout 5m

Services verified: bin-common-handler, bin-talk-manager, bin-tts-manager,
bin-transfer-manager, bin-webhook-manager, and the 20 VOIP-1406 consumer services
(agent, ai, billing, call, campaign, conference, contact, conversation, direct, flow,
number, queue, registrar, schedule, storage, tag, timeline, transcribe, transfer,
webhook). All five steps passed in each.

Blast-radius build check: go build ./... across every remaining module that compiles
against bin-common-handler, including voip-asterisk-proxy (the deliberately excluded
caller).

notifyhandler unit tests: <N> fixtures rewritten, <N> deleted, <N> added; the two
regression pins (topicEnabled=false NewNotifyHandler stays bit-identical;
NewNotifyHandlerForExistingExchange still declares nothing) pass.

Consumer services: all 20 binding_golden_test.go files and Run() sequencing tests
updated; failure-path cases now assert immediate error return. agent-manager and
timeline-manager additionally assert the VOIP-1258 webhook-topic block's log-only,
non-fatal failure behavior is unchanged.

Residue sweeps (/usr/bin/grep, not the IDE tool): zero remaining references to
topicDisabled, publishTopicEvent, or fanoutUnbindTargets.

talk-control smoke: `talk-control chat list` exits 0 (no startup Fatalf).

Not run / deferred: the 28-exchange broker deletion (docs/reference/
rabbitmq-queues-reference.md's new runbook) is an operational post-merge step gated on
full redeployment, not part of this PR.
```

---

## 7. Working notes and risk register

The design doc spent 10 review rounds eliminating a small number of specific failure modes. Each below is a **mechanical checkbox** an implementer or reviewer can tick against the diff. If any is ambiguous in the diff, that is a Request Changes.

### R1 -- `talk-control`'s blast radius (design §2.3.1, CRITICAL)

Unit A's `logrus.Fatalf` turns a narrow pre-existing bug (9 write-path subcommands panic when invoked) into a **guaranteed boot crash for all 14 subcommands**, including the 5 read-only ones that work perfectly today. This is a regression **introduced by this ticket** if Unit B is not landed in the same PR.

- [ ] Unit B is in this PR. If any circumstance would drop it, Unit A **must not ship either**.
- [ ] `talk-control`'s `NewNotifyHandler` call passes `commonoutline.QueueNameTalkEvent` (not `""`), a real `reqHandler` (not `nil`), AND `WithGlobalTopicPublish()` -- all three. Two out of three is not sufficient: the option is what removes `TopicCreate` from `talk-control`'s path and preserves §5's precondition; the `reqHandler` is what prevents a nil-interface panic in `PublishWebhook`'s unrecovered goroutine (design §2.3.1's "Why the `nil` `reqHandler` argument can no longer be left alone").
- [ ] `talk-control` boots (`chat list` exits 0).

### R2 -- the two exception services' block positions (design §3.2, CRITICAL)

Design §3.2 was added after an earlier draft omitted the VOIP-1258 block entirely; following §3.1 mechanically would have deleted a live production binding in two services.

- [ ] **call-manager**: the `asterisk.all.event` `QueueSubscribe` survives, as a standalone statement **immediately after `QueueCreate` and before the topic-declare line** -- NOT before `go ConsumeMessage` (design §3.2 explicitly corrects that placement).
- [ ] **timeline-manager**: same asterisk statement, same position, AND the VOIP-1258 block.
- [ ] **agent-manager**: the VOIP-1258 block survives verbatim, positioned where the deleted fanout loop used to sit, i.e. before the `topicPatterns` block.
- [ ] **No other service** gained or kept a fanout `QueueSubscribe` line.
- [ ] The three exception services' diffs are in their own commit (§2.2 commit 6), reviewable independently of the 17-service sweep.

### R3 -- fatal-vs-log-only failure semantics split (design §2.4, §3.2, §3.3)

Three different failure-handling regimes now coexist. Mixing them up is silent and dangerous.

| Site | Regime | Authority |
|---|---|---|
| `NewNotifyHandler` fanout declare (topicEnabled=false only) | **FATAL** (`logrus.Fatalf`) | design §2.3 |
| `initGlobalTopicExchange` topic declare | **FATAL** (`logrus.Fatalf`) | design §2.4 |
| Consumer `TopicCreateWithKind` + `topicPatterns` `QueueBind` | **FATAL** (`return err`) | design §3.3 |
| Consumer asterisk `QueueSubscribe` (2 services) | **FATAL** (`return err`) | design §3.3 snippet |
| Consumer VOIP-1258 `QueueBind`/`QueueUnbind` (2 services) | **LOG-ONLY, NON-FATAL -- UNCHANGED** | design §3.2 sub-ruling 1 |

- [ ] The VOIP-1258 block's `log.Errorf` calls were **not** converted to `return` statements anywhere. This is the one place the "make it fatal" sweep must stop.
- [ ] The VOIP-1258 block's legacy `QueueUnbind(..., QueueNameWebhookEvent, ...)` call still exists (design §3.2 sub-ruling 2).
- [ ] A test explicitly asserts the log-only behavior survives, in both agent-manager and timeline-manager (design §8).

### R4 -- rollback machinery removal (design §3.3)

- [ ] The "bound so far, roll back on partial failure" per-pattern `QueueUnbind` is gone from all 20 services. Leaving it behind while the fanout leg is gone produces a service that unbinds its only intake path on a partial bind failure.
- [ ] Every test named `*rollsBackPartialBinds*` / `*noRollbackNeeded*` was reworked, not just renamed.

### R5 -- the excluded caller stays inert (design §0, §2.1, §8 pin 2)

- [ ] `git diff --stat` shows **zero** lines changed under `voip-asterisk-proxy/`.
- [ ] Regression pin 2 passes.
- [ ] After Unit C deletes the 3 dead-wiring sites and Unit B moves `talk-control` to `topicEnabled=true`, `voip-asterisk-proxy` is the **only** remaining `topicEnabled=false` `NewNotifyHandler` construction in the repo (design §2.1's table, R4 finding 5). Verify with a `/usr/bin/grep` sweep for `NewNotifyHandler(` call sites lacking `WithGlobalTopicPublish()`.

### R6 -- enumeration methodology (issue §3's search-methodology paragraph)

- [ ] Every sweep used `/usr/bin/grep`, never the IDE Grep tool.
- [ ] A dedicated **string-literal** grep was run for hardcoded exchange names, since the constant-based sweep cannot see them -- `bin-conversation-manager/cmd/conversation-control/main.go:58` hardcodes `"bin-manager.conversation-manager.event"` inline (issue §3, R4 finding 7). Confirm this site is correctly handled by Unit A (it also passes `WithGlobalTopicPublish()`, so it becomes topic-only and the literal becomes inert), and that no *other* literal site exists.
- [ ] Counts re-verified against issue §3's numbers (55 dual-publish across 49 files, 5 fanout-only, 2 existing-exchange, total 62) rather than trusted blindly from either source.

### R7 -- observability (design §2.5)

- [ ] `promNotifyProcessTime` is observed on the topic-only path under its **existing** name and `type` label. Renaming or adding a metric would contradict design §2.5's Decision.
- [ ] `routing_key` survives into the returned error string (design §2.5's "Diagnostic-log fidelity"), not dropped at the `promTopicPublishTotal` increment.

### R8 -- docs sync hook and vendor hygiene

- [ ] Every service whose `cmd/*/main.go` or `pkg/subscribehandler/main.go` changed has its `docs/architecture.md` events section staged in the same commit (`scripts/check-service-docs.sh` warns otherwise).
- [ ] Vendor directories are **not** committed (`.gitignore` excludes `vendor/`; never `git add -f` them). The `go.mod`/`go.sum` changes from `go mod tidy` **are** committed.

---

## 8. Open questions for the 대표님 (design doc does not settle these)

Per this plan's constraint, these are raised rather than decided.

**OQ-1 -- `bin-api-manager`'s residual fanout-subscribe machinery.**
*Verified this plan:* `bin-api-manager/pkg/subscribehandler/main.go` carries a `subscribeTargets []string` struct field (`:33`), constructor parameter (`:65`), and a fanout `QueueSubscribe` loop in `Run()` (`:94-95`) -- structurally identical to what design §3.1 deletes in the 20 services. It is fed an **empty** slice literal at `bin-api-manager/cmd/api-manager/main.go:206` (`subscribeTargets := []string{}`), so it binds zero fanout exchanges and design §5's exchange deletion **cannot** break it. That is presumably why it is correctly absent from the 20-service list. But the design doc does not mention this 21st instance of the same machinery at all, so it is unclear whether leaving it is intentional (genuinely harmless dead code, out of this ticket's stated scope) or an oversight (exactly the code shape this ticket exists to remove). **Recommendation: leave it, and file a separate Jira cleanup follow-up** -- deleting it adds a 21st service to an already 23-module PR for zero functional gain, and it cannot cause the outage design §5 guards against. Needs a ruling either way so a reviewer does not flag it as a miss.

**OQ-2 -- the test-file blast radius of Unit C's dead-wiring deletion.**
Design §4 calls the 3 dead-wiring deletions "trivial (remove an unused field, an unused constructor parameter, and the `NewNotifyHandler` call plus its now-unused imports)." *Verified this plan:* removing `bin-transfer-manager/pkg/transferhandler`'s `notifyHandler` field and constructor parameter mechanically touches **8 test files** in that package. The mechanical fix (drop the mock-notify argument from each construction) is unambiguous, but a second question is not: several of those tests may currently set `mockNotify.EXPECT()` expectations that would become unused variables. **Recommendation: proceed with the mechanical fix and delete any now-unused mock declarations** -- that is the only compiling outcome. Flagged because the design doc's "trivial" framing under-describes the diff size, and a reviewer comparing plan-to-design may read the larger diff as scope creep.

**OQ-3 -- the derived per-service consumer test inventory (§3 E4).**
Design §8 specifies the consumer-side test requirement generically ("update `binding_golden_test.go` and the `Run()` sequencing test") rather than naming each service's tests, unlike the exhaustive treatment it gives `notifyhandler`'s tests. §3 E4 above **derives** the concrete per-service list by locating every existing `Run()`-related test in the worktree. The derivation contains judgment calls the design doc does not explicitly authorize -- specifically the **deletions**: `Test_Run_fanoutUnbindFails` (call), `Test_Run_sequencing_fanoutUnbindFailure_continues` (billing, contact), `Test_Run_TopicMigration_FanoutUnbindFailure` (timeline), `Test_Run_QueueSubscribeError` (tag), `Test_Run_SubscribeError` (contact), and the whole of `bin-timeline-manager/pkg/subscribehandler/run_sentinel_test.go`. Each targets behavior design §3.1/§3.2 removes, so deletion follows logically, but "delete the test" vs "keep the file and rewrite it to pin the new behavior" is a real choice. **This is the part of the plan a reviewer should scrutinize hardest.** No decision is made here beyond "follow the design's own logic"; if the 대표님 prefers rewriting over deleting for any of these, say so before implementation starts.

**OQ-4 -- deploy sequencing is intentionally out of this plan.**
Design §1 notes the publish-side and consumer-side changes "do not depend on EACH OTHER's code being live first, and can ship in separate PRs/commits," while design §3.4 and §5 impose per-service rollout preconditions. This plan follows root `CLAUDE.md` and ships one PR, which means **the whole change deploys as one unit**, and the per-service rollout gates of §3.4 apply to the deployment of that one artifact set rather than to separately merged PRs. That is compatible with both documents, but it does mean the deployment runbook (post-merge, out of this plan's scope) has to be written against a single-artifact rollout. Flagging so the deployment step is not assumed to inherit a two-phase shape the PR structure does not provide.

---

## 9. Definition of done for this plan's execution

- [ ] All 7 commits (§2.2) on `VOIP-1407-Cutover-remove-fanout-dual-publish`.
- [ ] §4.1's full verification workflow passes in all 23 touched modules, and the blast-radius `go build` passes repo-wide.
- [ ] Every checkbox in §4.2 (design §8 traceability) ticked.
- [ ] Every checkbox in §7 (risk register) ticked.
- [ ] §8's open questions answered by the 대표님 before implementation starts.
- [ ] Docs (Unit F) staged with their corresponding code commits; `scripts/check-service-docs.sh` emits no unresolved warnings.
- [ ] `git fetch origin main` + conflict check clean, PR opened with the §6.1 body and §6.2 verification section.
- [ ] **NOT merged.** Awaiting the 대표님's explicit merge instruction.
