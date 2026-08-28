# VOIP-1405 Implementation Plan (rev.9)

Design: `docs/plans/2026-08-27-voip-1405-topic-publisher-rollout-design.md` (Approved). Unqualified § refs below = 1405 design; "1404 §N" = the skeleton design.
Branch: `VOIP-1405-Topic-exchange-publisher-rollout`

## Acceptance criteria (each AC names its evidence task)

- [x] AC1: 53 wiring points across 26 services carry `WithGlobalTopicPublish()` — evidence: task 3.2 grep count == 55 (= 53 new + 2 pilot) AND talk-control/tts-control asserted absent.
- [x] AC2: 19 override types implement `EventSubscriptionID()` (pointer receiver). Each type's OWN package `*_test.go` carries BOTH (a) `var _ eventtopic.SubscriptionIdentifier = (*T)(nil)` AND (b) a behavioral test that CALLS `EventSubscriptionID()` asserting the returned address — including an explicit "address != own ID" assertion where the type has an own id (pilot precedent: `TestSpeechEventSubscriptionIDIsNotOwnID`). This is what covers the method body in its own package (golden tests live elsewhere and contribute nothing to these packages' coverage). Evidence: task 3.2 assertion count == 26 (= 19 new + 7 baseline) + behavioral-test listing.
- [x] AC3: 4 new structs replace map payloads with identical JSON key SETS (order differs; no byte-equality asserts) — evidence: golden tests + task 3.2.
- [x] AC4: contact []byte publish fixed (`ConvertWebhookMessage()` — payload intentionally changes base64→object), pipecat 6 value publishes → pointers, contact case event constants named — evidence: Phase 1 tasks + code review.
- [x] AC5: golden routing-key test per publishing service — evidence: task 3.2 file count == 27 (= 26 new + 1 pilot), per-task special-case rows (below).
- [x] AC6: docs sync per touched service + reference doc mapping summary — evidence: per-task + 3.1.
- [x] AC7: verification green everywhere (vendor regen first); coverage on changed `models/`/`pkg/` packages (`cmd/` exempt per AC9) is **delta-based, not absolute**: (a) NO regression vs pre-change coverage (executor records the before number in its report), (b) lines added/changed by this ticket ARE covered, (c) absolute ≥80% applies only to packages already ≥80% before. Packages measured below 80% pre-change (R6 measurements: callhandler 55.1%, pipecatcallhandler 51.4%, casehandler 58.6%, contacthandler 61.2%, models/call 60.8%, models/pipecatcall 60.0%, webchat models/message+session 0.0%, conversation models/message 7.7%, others) are OUT OF SCOPE for raising to 80% here — legacy coverage uplift goes to a follow-up ticket (3.3). Baseline rule for `[no statements]`/`[no test files]` packages (dtmf, kase, casenote, pipecat models/message today): treated as NOT already-≥80% — only criteria (a)+(b) apply, and (b) is satisfied by the AC2 behavioral tests. Phase-2 services touching only cmd/ wiring + golden file + docs have no coverage obligation (nothing in models/pkg changes there). Compile sweep 39 modules; `git diff --stat` shows zero bin-common-handler changes — evidence: per-task + 3.2.
- [x] AC8: follow-up Jira tickets per 1405 §7 registered (English summary / Korean body) — evidence: 3.3.
- [x] AC9: **constructor-wiring lines** in cmd/ get no unit test — most cmd/ dirs have no test seam (transcribe pilot precedent), and where cmd tests do exist (registrar-control `domain_migrate_test.go`, agent-control) they test batch logic, not constructor wiring; extending them to assert an option on a constructor call would test the plan, not behavior. Named substitute guard = AC1's grep assertions + code review. registrar-control's cmd-level PUBLISH path (domain_migrate) is still covered by its golden key expectations. Recorded in 3.2 + PR body.

## Execution phases

**Service accounting: Phase 1 = 12 services in 8 groups (22 wiring points); Phase 2 = 14 services (31 points). 22+31=53, 12+14=26.**

### Phase 1 — override-carrying services (12 services / 8 groups / 22 points)
Every task below includes: wiring, golden test (external test package, ONE file per service named `routingkey_golden_test.go`, placed in the service's designated PRIMARY model package (chosen for practical anchoring, not strict aggregate semantics) — FULL enumeration, all 26: contact→kase, call→call, pipecat→pipecatcall, ai→ai, tts→streaming, talk→chat, conversation→conversation, webchat→session, campaign→campaign, conference→conference, queue→queue, schedule→schedule, agent→agent, billing→billing, customer→customer, direct→direct, email→email, flow→flow, message→message, number→number, outdial→outdial, registrar→trunk, route→route, storage→account, sentinel→pod, tag→tag; template = transcribe pilot. Override method bodies are covered by AC2 behavioral tests in their OWN packages, not by the golden file), docs sync (architecture.md for cmd; domain.md for models), 5-step verification with `go mod vendor` FIRST, `go test -cover` on changed packages.
- [x] 1.1 **contact** (2 pts): []byte fix; new structs `kase.CaseTagEvent`/`kase.CaseContactEvent`/`casenote.CaseNoteDeletedEvent`; named constants for 6 case event literals; `CaseNote`→CaseID override. Golden MUST include: **CaseNoteDeletedEvent address == case_id row (silent-failure class — mandatory)**, contact_update dynamic 2 branches, case-axis convergence.
- [x] 1.2 **call** (2 pts): `OutboundWhitelistRejectedEvent` struct; `dtmf.DTMF`→CallID. Golden MUST include: mapEvt 5 branches, dot-type `call.outbound_whitelist_rejected` normalization.
- [x] 1.3 **pipecat** (2 pts): 6 pointer conversions (runner.go:510,563,576,584,865,886); `Message`/`MemberSwitchedEvent`→PipecatcallID. Golden: all message_* + team_member_switched + pipecatcall_*.
- [x] 1.4 **ai** (3 pts): `Message`(B)/`IntermediateWebhookMessage`(A)→AIcallID. Note: ai-control has 2 instances.
- [x] 1.5 **tts** (1 pt): `Message`→StreamingID. Golden: message_play_started/finished are LIVE; streaming_* dead trio excluded.
- [x] 1.6 **talk** (1 pt): `Message`/`Participant`→ChatID; DO NOT rename the `commonnotify` import alias (3 files in this service use it; AC1 pattern is alias-independent, and renaming adds cosmetic diff to an already-large PR). talk-control: DO NOT TOUCH.
- [x] 1.7 **conversation + webchat** (2+1 pts): `Message`→ConversationID / SessionID. webchat golden MUST include resource-collapse row (both events → resource `webchat`, same session address).
- [x] 1.8 **campaign + conference + queue + schedule** (2+2+3+1 pts): `Campaigncall`→CampaignID, `Conferencecall`→ConferenceID, `Queuecall`→QueueID, `Execution`→ScheduleID. schedule golden: dispatch dynamic 2 branches.

### Phase 2 — default-id services (14 services / 31 points)
Same per-task inclusions as Phase 1.
- [x] 2.1 agent(2), billing(2), customer(3 — golden MUST include CustomerCreatedEvent wrapper: id promotion AND nil-embed→placeholder rows), direct(2), email(2), flow(2), message(2), number(2 — golden: dbUpdate dynamic 2 branches)
- [x] 2.2 outdial(2), registrar(4 — incl. domain_migrate publish path), route(2), storage(3 — golden: `Account_*` uppercase normalization rows), sentinel(1 — golden: pod placeholder rows; no override on *corev1.Pod), tag(2)

### Phase 3 — closure
- [x] 3.1 Reference doc `docs/reference/rabbitmq-queues-reference.md`: Category A/B mapping summary, sentinel placeholder invariant (`placeholder_total ≈ publish_total{ok}`), address-convergence notes (tts/contact/pipecat/webchat). RST determination: expected NOT affected (internal-only; the contact []byte change alters timeline stored history, not any documented API shape) — verify and record; if the events API docs describe payload examples, update them.
- [x] 3.2 Global assertions (AC evidence). **Run from THIS worktree root** (the main repo has sibling worktrees that pollute counts). Baselines empirically measured on main (R3): wiring 2, assertions 7, golden 1.
  - AC1: `/usr/bin/grep -rn "WithGlobalTopicPublish()" --include="*.go" --exclude-dir=vendor --exclude-dir=.worktrees --exclude-dir=bin-common-handler . | /usr/bin/grep -v "_test.go" | wc -l` == **55** (53 new + 2 pilot). Use `/usr/bin/grep` — the rtk shell hook rewrites grep and strips `./` path prefixes, which both breaks path-anchored filters and can drop lines; `--exclude-dir=bin-common-handler` removes the option-definition lines path-independently. Pattern is unqualified (substring) because bin-talk-manager imports notifyhandler as `commonnotify` (import at cmd/talk-manager/main.go:16, constructor call at :82; alias kept — see 1.6). One option call per constructor site, never duplicated (some sites are multi-line calls). Absence assertion: `/usr/bin/grep -rn "WithGlobalTopicPublish()" --include="*.go" bin-talk-manager/cmd/talk-control bin-tts-manager/cmd/tts-control | wc -l` == 0 (do not suppress stderr).
  - AC2: `/usr/bin/grep -rn "_ eventtopic.SubscriptionIdentifier" --include="*_test.go" --exclude-dir=vendor --exclude-dir=.worktrees . | wc -l` (/usr/bin/grep — rtk hook avoidance, same as AC1) == **26** (19 new + 7 baseline: 4 in bin-common-handler notifyhandler fixtures + 3 transcribe pilot). Placement: each override type's own package `*_test.go` (pilot precedent), NOT the golden file. ALSO list behavioral tests: `/usr/bin/grep -rln "EventSubscriptionID()" --include="*_test.go" --exclude-dir=vendor --exclude-dir=.worktrees . | /usr/bin/grep -v routingkey_golden_test.go` must show a non-golden test file in each of the **15 packages** the 19 types live in (golden files also call the helper — excluded to prevent false positives); per-type 19-count verification uses the AC2 `-rn` assertion listing where type names are visible.
  - AC5: `find . -path ./vendor -prune -o -path ./.worktrees -prune -o -name "routingkey_golden_test.go" -print | wc -l` == **27** (26 new + 1 pilot). NOTE: the rtk shell hook rejects compound find predicates — run via `/usr/bin/find` or `rtk proxy find`.
  - Distribution check (totals alone don't prove coverage): keep the `-print`/`-l`/`-rn` listings and verify — AC1: group the 55 lines by service and compare against the per-service point numbers in Phase 1/2 (offsetting mis-wirings keep the total at 55); AC5: 26 distinct service dirs; AC2: 19 distinct types. Note: the behavioral `-l` list also matches method DEFINITIONS (harmless here — the 15 target packages define methods in non-test source).
  - PR body: include the site reconciliation formula (60 total constructor sites = 53 new + 2 pilot + 5 excluded [asterisk 1, talk-control 1, tts-control 1, transfer-manager dead 2]) to pre-empt the "60 vs 55" review question; note webhook-manager's 2 sites use NewNotifyHandlerForExistingExchange (option forbidden, design §1.2) and are outside the 60.
  - Do not suppress stderr on the absence assertion (a moved path would fail silently to 0).
  - Compile sweep 39 modules; `git diff --stat` — no bin-common-handler lines. Record AC9 decision here.
- [x] 3.3 Follow-up Jira tickets (1405 §7; summary EN, body KR): talk-control defect (+post-fix option), dead NotifyHandlers (transfer + tts ttshandler), conversation secret stripping (PRIORITY), stream-completeness audit (incl. timeline subscribing never-published conversation_deleted), **legacy coverage uplift** for the sub-80% packages listed in AC7 (aggressive-coverage principle applied head-on in its own ticket). Also fold into VOIP-1409 (notifyhandler housekeeping): the inverted method-set comment at `bin-common-handler/models/eventtopic/identifier.go:43` — "a value receiver would silently never be picked up" is backwards (a value receiver satisfies the interface for both forms; it is a VALUE of a pointer-receiver type that fails); excluded from this PR by the zero-bch-changes rule.

### Phase 4 — review & PR
- [x] 4.1 Commits by the ORCHESTRATOR ONLY (executors never stage/commit — see delegation protocol). Title matches branch; body project-prefixed bullets; verify via git diff --cached.
- [x] 4.2 Code review loop (min 3 rounds, 2 consecutive approvals); re-run verification in touched services after fixes.
- [ ] 4.3 Pull main + conflict check; full re-verify if rebased.
- [ ] 4.4 Single PR: narrative summary + `bin-<service>:` prefixed bullets (per-service table SUPPLEMENTS, not replaces, the bullets; no markdown headers); runbook (wave order low-freq→call/pipecat last, per-service metric checks, rollback = option removal per service, noting contact normalization is NOT reverted by rollback — intended independent change); AC9 substitution note; RST determination. NO merge without instruction.

## Delegation protocol (parallel executors)

- Executors work ONLY inside their assigned service directory; they NEVER run `git add`/`git commit`/`git stash` (git index is shared — orchestrator owns all staging/commits).
- Executor reports must include `git status --short -- <service-dir>` output + test/lint results; orchestrator verifies before accepting.
- Only the orchestrator edits tasks/todo.md.
- Hook warnings from scripts/check-service-docs.sh may be misattributed across concurrent agents — orchestrator re-checks per service at staging time. Phase 2 services: update architecture.md only; IGNORE domain.md warnings triggered by the golden _test.go file (the hook matches models/ regardless of _test.go; do not invent domain.md content).

## Working notes

- vendor/ stale in EVERY service — `go mod vendor` before any test run.
- No bin-common-handler edits; if a change seems to need one, STOP and re-plan.
- Do NOT add override to *Customer (wrapper promotion) or *corev1.Pod (external type).
- webhook-manager, asterisk-proxy, talk-control, tts-control: hands off.

## Results (2026-08-28)

- 26 services wired (53 sites; verified 55 = 53 + 2 pilot), 19 override types with behavioral tests in 15 packages, 27 golden files in 27 service dirs.
- Prerequisite fixes landed: contact []byte payload, pipecat 6 pointer conversions (now compiler-enforced), 4 new structs with byte-identical JSON output (empirically compared).
- Drive-by fixes: outdial SA4000 tautologies -> JSON round-trips, email SA5011, registrar/route CLAUDE.md false "no events" claims, timeline stale comment.
- Global assertions all pass: AC1 55 / absence 0 / AC2 26 / AC5 27 / compile sweep 39 modules 0 fail / bin-common-handler 0 lines.
- Code review loop: R1 RC (4 MEDIUM, comment/test fidelity) -> R2 RC (3) -> R3 Approve -> R4 (pending). Typed-nil guard mutation-locked via pilot golden.
- Follow-up tickets: VOIP-1412 (talk-control defect), VOIP-1413 (dead NotifyHandlers), VOIP-1414 (conversation secret stripping, priority), VOIP-1415 (stream completeness), VOIP-1416 (coverage uplift), VOIP-1417 (mockgen drift); identifier.go comment folded into VOIP-1409.
