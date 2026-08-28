# VOIP-1419: Enforce explicit EventSubscriptionID on all event types and remove JSON fallback

Status: PLAN (stage 3 of 4). Issue analysis APPROVED (R1+R2 Approve). Design APPROVED
(docs/plans/2026-08-28-voip-1419-explicit-subscription-id-design.md; R1 RC, R2 RC, R3
Approve, R4 Approve — 2 consecutive). Design review round 4 additionally verified: golden
coverage has NO enforcement hole (all 45 new-method types map to rows in the 27 golden
files, mutation-sensitive for the confbridge wrong-id case), the 64-byte cap and
`.`/`*`/`#` sanitization sit downstream in `eventtopic.normalizeSubscriptionID` /
`normalizeSegment` (a bad method cannot emit an invalid AMQP key), `WebhookEventMessage`
has zero name collisions, and zero consumers bind `bin-manager.event` (the only QueueBind
consumers target the VOIP-1258 `bin-manager.webhook-manager.event.topic`, a different
exchange).

Advisory notes carried from the analysis review into design:
- `customer.Customer` does NOT embed `commonidentity.Identity` (own `ID uuid.UUID` field,
  customer.go:22); a mechanical "return h.Identity.ID.String()" template will not compile
  there — the method must use the type's own field. Key value is identical either way.
- Under D1 option (b), `MockWebhookMessage` no longer satisfies the narrowed
  `PublishWebhookEvent` parameter — notifyhandler mock regeneration plus fixes in tests that
  pass it are required (e.g. bin-conversation-manager messagehandler/create_case_id_test.go:60).
- `requesthandler.CallPublishEvent` (publish_event.go:42; sole caller call-manager
  channelhandler/health.go:83, impersonating publisher "asterisk-proxy") bypasses
  notifyhandler entirely (direct sock publish to a subscribe queue) — audited, out of scope.
- The single control-CLI publish site (registrar-control domain_migrate.go:591,
  `*extension.Extension`) is inside the counts and covered by the 45-method set.

## Issue Analysis (2026-08-28)

### 1. Issue validity: VALID, proceed

- The decision this ticket encodes was made explicitly by the CEO on 2026-08-28 after the
  VOIP-1405 merge: the JSON `"id"` fallback is too implicit; every published event data type
  must explicitly implement `eventtopic.SubscriptionIdentifier`, enforced at compile time.
- Code re-check confirms the ticket matches reality on current `main` (post-#1217,
  `60de2f733`): the fallback exists exactly as described, at
  `bin-common-handler/pkg/notifyhandler/publish.go`:
  - `subscriptionIDData` struct: publish.go:23-25
  - `parseSubscriptionID(data json.RawMessage)`: publish.go:241-252
  - fallback invocation: publish.go:213-220 (`if !hasOverride { subscriptionID = parseSubscriptionID(evt.Data) }`)
- Nothing has resolved this in the meantime; no competing implementation exists.

### 2. Code re-check: publish-surface facts (verified against source + full inventory)

The `NotifyHandler` interface (main.go:131-138) has 5 publish methods. What the signature
change touches, per empirical inventory of the whole repo (prod code, vendor/.worktrees
excluded):

| Method | Prod call sites | Topic-publishes? | Impact of narrowing `data` |
|---|---|---|---|
| `PublishEvent(ctx, type, data interface{})` | 116 (115 external + 1 internal) | YES (resolves override at publish.go:99-103) | primary change target |
| `PublishWebhookEvent(ctx, cid, type, data WebhookMessage)` | 118 | YES, indirectly: publish.go:32 does `go h.PublishEvent(ctx, eventType, data)` | **will not compile** unless the value passed also satisfies `SubscriptionIdentifier` |
| `PublishWebhook(ctx, cid, type, data WebhookMessage)` | 1 (internal only, from PublishWebhookEvent) | no (webhook RPC path) | none needed |
| `PublishEventRaw(ctx, type, dataType, data []byte)` | 1 (voip-asterisk-proxy ari_handler.go:76, raw ARI frames) | only if topicEnabled; asterisk-proxy is NOT topic-enabled | `[]byte` cannot implement the interface; see decision D2 |
| `PublishEventWithRoutingKey(ctx, type, key, data interface{})` | 1 (bin-webhook-manager routingkey.go:182, `json.RawMessage` payload) | no (VOIP-1258 scoped path, caller supplies the key) | OUT of scope; see decision D3 |

Type inventory:
- 67 distinct data types flow into `PublishEvent` + `PublishWebhookEvent`
  (41 + 30, minus 4 overlaps: `transcribe.Transcribe`, `transcript.Transcript`,
  `number.Number`, `agent.Agent`).
- 22 already implement `EventSubscriptionID()` (VOIP-1404 pilot + VOIP-1405 overrides).
- **45 new methods needed.** All are pointer-typed at every call site (zero map, zero
  non-pointer struct, zero nil literal, zero []byte on these two methods), so every one is
  mechanically implementable EXCEPT:
  - **F1** `*corev1.Pod` (bin-sentinel-manager monitoringhandler/run.go:105,117): external
    module type; cannot add a method; needs a local wrapper type. Only genuinely
    unfixable-in-place site in the repo.
  - **F2** `*contact.WebhookMessage` (bin-contact-manager contacthandler/event.go:33): a
    webhook DTO passed to `PublishEvent` (introduced by the VOIP-1405 []byte fix); needs a
    method (its own `ID` is the correct address) or replacement with the domain type.
- Test churn: ~658 `_test.go` publish references, but ~97% are gomock `EXPECT()` recorder
  calls whose parameters are `any` and keep compiling. Known real test-code impact:
  `bin-conversation-manager/pkg/messagehandler/create_case_id_test.go:60` (typed `Do`
  callback referencing `notifyhandler.WebhookMessage`).
- 27 golden routing-key test files contain a `resolveSubscriptionID` helper that mirrors
  production INCLUDING the JSON half; all 27 must be simplified in the same change, and the
  "must NOT implement SubscriptionIdentifier" negative assertions (e.g.
  `TestTranscribeUsesDefaultSubscriptionID`) must be replaced by
  `var _ eventtopic.SubscriptionIdentifier = (*T)(nil)` compile-time assertions, otherwise
  they fail the moment the methods are added.

### 3. Scope decisions surfaced by the analysis (to be settled in the design stage)

- **D1 — how to constrain the webhook path.** Two options:
  (a) embed `SubscriptionIdentifier` into the `WebhookMessage` interface itself → all 55
  `CreateWebhookEvent` implementers must add methods, including 25 types never passed to
  `PublishWebhookEvent` in production;
  (b) leave `WebhookMessage` untouched and change `PublishWebhookEvent`'s param to a new
  intersection interface (`interface { WebhookMessage; eventtopic.SubscriptionIdentifier }`)
  → only the 30 actually-published webhook types need methods.
  Analysis leans (b): precise, 25 fewer dead methods; (a) is more uniform. Decide in design.
- **D2 — `PublishEventRaw`.** Keep `[]byte` signature. With the fallback gone, a
  topic-enabled service calling Raw would always get the `-` placeholder. Today the sole
  caller (asterisk-proxy) is not topic-enabled, so nothing changes in production; document
  the placeholder behavior in the method comment. No API change.
- **D3 — `PublishEventWithRoutingKey`.** Out of scope: it never dual-publishes to
  `bin-manager.event` (caller supplies its own key; VOIP-1258 path), and its only payload is
  a `json.RawMessage` that cannot implement the interface. Signature stays `interface{}`.
- **D4 — internal simplification.** With the interface mandatory, `hasOverride` threading
  becomes constant-true on the PublishEvent path (typed-nil still degrades to placeholder
  via the existing reflect guard, which must stay); `parseSubscriptionID`,
  `subscriptionIDData`, and the `hasOverride` parameter of the internal `publishEvent` can
  be removed/simplified. `PublishEventRaw` hard-codes the placeholder path. The delayed
  path (`delay > 0`, publish.go:160-166) returns before any topic publish and is unaffected.

### 4. Proceeding rationale (priority / risk / sequencing)

- Sequencing confirmed with the CEO and recorded on the ticket: VOIP-1405 merge (done,
  deployed 2026-08-28) → **VOIP-1419** → VOIP-1406 (consumer migration). Topic-exchange
  consumers are still ZERO, so swapping the resolution mechanism now has no subscriber
  impact; this window closes once VOIP-1406 starts. That makes this the right time, not
  just a valid time.
- Routing-key VALUES must not change; the 27 golden test suites (which pin exact keys) are
  the regression fence. The mechanism swap is invisible on the wire.
- Cost: bin-common-handler public-surface change → all 38 services re-verify; 45 new
  methods + wrapper type + 27 golden-helper edits + mock regeneration. Large but
  mechanical; no data migration, no schema change, no consumer coordination.
- Risk of NOT doing it now: every new event type added meanwhile silently relies on the
  fallback; the longer the wait, the bigger the one-shot migration.

### 5. Acceptance criteria (draft, to be finalized in design/plan)

- `PublishEvent`'s `data` parameter (and the webhook-path equivalent per D1) requires
  `eventtopic.SubscriptionIdentifier`; the repo compiles ⇒ every published type implements.
- `parseSubscriptionID` / `subscriptionIDData` deleted; no JSON id extraction remains in
  notifyhandler.
- All 27 golden routing-key suites pass UNCHANGED in their expected key strings (only the
  resolution helper and negative assertions change).
- 38-module verification workflow passes (tidy/vendor/generate/test/lint per service).
- Placeholder metric semantics unchanged; typed-nil still resolves to `-` without panic.

## Implementation Plan (stage 3) — rev.1

Normative source: the Approved design doc. This plan adds ONLY execution mechanics.

### Ordering principle (load-bearing)

The signature narrowing breaks all 234 production call sites the moment it lands, so the
branch is built in waves where **method addition (additive, compiles against the old
signature) precedes narrowing (lands only when every type already implements)**. Every
commit on the branch compiles and passes its touched services' tests.

### Waves

- [ ] **W1 — methods + tests, per service (additive; ~26 services, parallel executors).**
  Per service: add the `EventSubscriptionID()` methods for its share of the 45 types
  (template + special cases per design §3; placement rule: sibling file when the type
  lives in `models/<entity>/webhook.go`); add `var _ eventtopic.SubscriptionIdentifier`
  assertions in sibling `_test.go`; add per-type behavioral tests (address == own id,
  mutation-checked; nil-embed `""`-no-panic tests for CustomerCreatedEvent and pod.Event);
  invert the negative assertions **whose types gain methods (27 of the 30)** in the same
  commit as those methods. **The 3 negatives on types that REMAIN non-implementers are NOT
  inverted** — `*corev1.Pod` (pod golden :138), `*pod.Pod` (pod golden :159, pins the
  promotion hazard on the OLD wrapper), and `kase.Case`
  (bin-contact-manager/models/kase/event_test.go:207, never published, not in the 45).
  They stay as runtime negatives with their semantics intact, but their message strings
  are reworded to drop the exact phrase "must not implement SubscriptionIdentifier" (AC5
  depends on it). Note kase/event_test.go is not a golden file — do not miss it.
  **"Invert" means: DELETE the runtime negative test; the type's single compile assertion
  lives in its sibling `_test.go`, not the golden file** (a duplicate `var _` in the
  golden file would push AC6 past 71). While editing, also refresh prose comments that
  describe the deleted fallback (helper doc comments, e.g. tts golden :51-56; free-standing
  ones like tts golden :48-49 "resolves through the default JSON fallback" and the
  surviving kase.Case test's doc comment) — cosmetic, but stale mechanism descriptions are
  exactly what this ticket exists to kill.
  Simplify the golden `resolveSubscriptionID` helper: drop the JSON half BUT **keep the
  `data any` parameter signature** (assert → typed-nil guard → method, else return `""`) —
  the pod golden legitimately feeds non-implementing payloads (bare `*corev1.Pod`);
  typing the parameter as the interface would not compile. Valid pre-narrowing because
  for implementing types the override already wins, so resolution results are unchanged.
  Sentinel specifics: publish sites wrap in `pod.Event`; the pod golden's TABLE rows
  switch from bare `*corev1.Pod` to `pod.Event` (expected keys unchanged:
  `sentinel-manager.pod.-.updated` / `.deleted`); the two supplementary pod tests stay
  (bare-Pod negative reworded per above; `TestPodPayloadHasNoSubscriptionAddress` keeps
  feeding a bare `*corev1.Pod` through the `any`-typed helper and now additionally gains
  a `pod.Event` row asserting the wrapper's explicit `""`); header comment rewritten per
  design §3. Expect the service-docs hook to warn on the new `models/pod` type — stage the
  corresponding `docs/domain.md` line rather than dismissing the warning.
  Per-service verification workflow (tidy/vendor/generate/test/lint) before each commit.
- [ ] **W2 — bin-common-handler narrowing (single commit).**
  `WebhookEventMessage` interface; `PublishEvent` param → `eventtopic.SubscriptionIdentifier`;
  `PublishWebhookEvent` param → `WebhookEventMessage`; `resolveSubscriptionID` two-part
  guard (nil-interface FIRST, then typed-nil reflect guard); delete `parseSubscriptionID`
  + `subscriptionIDData`; drop `hasOverride` threading; `PublishEventRaw` passes `""` +
  comment; `topicEnabled` gate stays; identifier.go doc comment rewritten wholesale;
  mock regen; bch unit tests (fixture updates, nil-interface case, typed-nil case,
  Raw-placeholder case); conversation-manager typed callback update
  (create_case_id_test.go:60) rides this commit because the name `WebhookEventMessage`
  only exists from W2 — note it is a CONSISTENCY update, not compile-required (the
  callback's `notifyhandler.WebhookMessage` param still exists post-W2 and gomock invokes
  Do reflectively, so a green build without it is not an anomaly). bch verification +
  compile of ALL dependents. Post-merge: comment on VOIP-1409 that its identifier.go
  checklist item is closed by this PR (the checklist lives on the Jira ticket).
- [ ] **W3 — docs (single commit).** rabbitmq-queues-reference.md (fallback clause removed;
  explicit-method contract; "Deliberate non-overrides" block rewritten per design §5);
  one-line supersession pointers in the VOIP-1404/1405 design docs.
- [ ] **W4 — global verification + evidence.** Full 38-module compile sweep; per-service
  verification for every touched service; AC evidence commands below; PR creation after
  main conflict check.

### Executor protocol (as VOIP-1405)

Parallel executors are confined to their service directory, never touch the git index,
and report `git status --short -- <dir>` + test output. The orchestrator owns all commits
and re-verifies before committing. All greps use `/usr/bin/grep` with
`--exclude-dir=vendor --exclude-dir=.worktrees`.

### Acceptance criteria (evidence commands, run from worktree root)

- AC1 fallback gone: `/usr/bin/grep -rn "parseSubscriptionID\|subscriptionIDData" --include="*.go" --exclude-dir=vendor --exclude-dir=.worktrees bin-common-handler/` → **0 hits**.
- AC2 narrowing landed: `PublishEvent(ctx context.Context, eventType string, data eventtopic.SubscriptionIdentifier)` appears in notifyhandler interface + impl; `data interface{}` remains ONLY on `PublishEventWithRoutingKey` (D3).
- AC3 implementer count: `/usr/bin/grep -rn "func (.*) EventSubscriptionID()" --include="*.go" --exclude-dir=vendor --exclude-dir=.worktrees . | /usr/bin/grep -v "_test.go" | wc -l` → **67** (22 existing + 45 new).
- AC4 golden suites: all 27 `routingkey_golden_test.go` pass with **zero changes to any `expect` key string** — diff basis is the merge-base with main, not HEAD~: expect-line diff vs `$(git merge-base HEAD origin/main)` is empty; no golden helper contains a JSON-unmarshal half.
- AC5 negative-assertion string retired: `/usr/bin/grep -rn "must not implement SubscriptionIdentifier" --include="*_test.go" --exclude-dir=vendor --exclude-dir=.worktrees .` → **0 hits** (27 negatives inverted; the 3 surviving runtime negatives are reworded per W1 so the sentinel string disappears while their semantics stay).
- AC6 compile-time assertions: `/usr/bin/grep -rn "_ eventtopic.SubscriptionIdentifier = (" --include="*_test.go" --exclude-dir=vendor --exclude-dir=.worktrees . | wc -l` → **71** (26 existing incl. 4 bch test-local + 45 new). The pattern deliberately omits the `var ` prefix — 4 existing contact-manager assertions sit inside `var (...)` blocks and a `var _`-prefixed grep undercounts by 4.
- AC7 build: 38-module `go build ./...` sweep → 0 failures; full verification workflow per touched service passes.
- AC8 behavior: `go test ./...` green in bch including the new nil-interface, typed-nil, and Raw-placeholder cases.

## Working Notes

- Worktree: `.worktrees/VOIP-1419-Enforce-explicit-event-subscription-id` (branch same name, from `60de2f733`).
- Full inventory (counts per service, per method, flagged sites F1/F2 above plus the Raw
  and RoutingKey sites covered by D2/D3, 22 implementer list)
  produced 2026-08-28 by repo-wide scan; re-runnable commands recorded in the scan report.
- Env quirks: use `/usr/bin/grep`; exclude vendor/.worktrees; regen vendor before tests;
  RST pre-commit hook fires on staged `models/*/webhook.go` (keep methods in sibling files,
  e.g. `subscription.go`, as done for ai-manager in VOIP-1405).
