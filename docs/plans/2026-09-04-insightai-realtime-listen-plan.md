# InsightAI Realtime Listen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a Case Insight AI follow a live call's transcript in real time and proactively push a short note to the agent's Insight Assistant panel when the customer-configured conditions are met, without polluting the Q&A thread, evicting the LLM system prompt, or scaling LLM cost with speech volume.

**Architecture:** Two decoupled layers on one AIcall. Layer 1 (intake) consumes `transcribe-manager.transcript.*.created` from the global topic exchange, resolves ownership through a Redis set written at listen-start, and buffers each line into two Redis lists — no DB write, no LLM, no webhook. Layer 2 (evaluation) runs at most once per debounce interval per AIcall: it atomically drains the pending buffer, assembles a bounded, explicitly-built LLM context (never `getPipecatcallMessages`), and runs it on a **throwaway pipecatcall id that is never written to the AIcall row** — so the agent's own Q&A pipecatcall is never rotated, never interrupted, and never cooldown-blocked. Everything the listen turn emits is dropped except a deliberate `notify_agent` tool call, which writes one `role=assistant`, `Origin=proactive` message row that flows out over the existing webhook/WebSocket/poll path.

**Tech Stack:** Go 1.x (monorepo microservices, RabbitMQ RPC + topic-exchange events, MySQL via squirrel, Redis via go-redis v8, gomock table-driven tests), Python (pipecat runner, read-only here), Alembic (schema), OpenAPI 3 + Sphinx RST (docs), React (square-admin CRA/Jest, square-talk Vite/Vitest).

**Design document (read it before starting):** `docs/plans/2026-09-03-insight-ai-realtime-listen-design.md` (**rev 23**, Status: Approved — the rev-15 review sub-loop closed on review round 20's second consecutive approval). This plan implements that design. Where this plan and the design disagree on a *file path, line, or signature*, this plan wins — it was re-verified against the working tree on 2026-09-04. Where they disagree on a *decision*, stop and ask.

**This plan was synced to design rev 23 on 2026-09-04.** Design revisions 15-23 changed three things this plan originally got from rev 14, and Tasks 0, 2, 10, 11, 20, 26, 27, 28, 29, 30, 31, 32 and 33 were rewritten accordingly (see the Self-review section's sync note for the itemized account):

1. **The trigger is an explicit API, not a `Start` hook** (design §5.1, rev 15/16). `Start` no longer triggers listening at all. A new `POST /service_agents/aicalls/{id}/listen` — on the **Agent-facing** `/service_agents/*` surface, never the top-level Admin-console `/v1/aicalls/{id}/listen` path — calls a single exported `ProcessListen`, which runs `checkListenEligible` (steps 1-6) synchronously and spawns `runListenStart` (steps 7-8) detached. `ensureListen` / `ensureListenAsync` do not exist in this design any more.
2. **An event-ordering fix** (design §5.2.2, rev 15/16): the transcribe id is pre-generated and the DB + Redis state written *before* `TranscribeV1TranscribeStart`, with an explicit `rollbackListenState` undo path.
3. **A per-AIcall mutual-exclusion lock** (design §5.2.2, rev 17-20) wrapping the whole create-or-reuse sequence, with an ownership token, an atomic compare-and-delete release run on a context detached from the goroutine's own cancellation, and a best-effort release on the acquire-error path.

Design rev 15's own closing sentence bounds the blast radius: *"Neither change touches §5.2-§5.9's Layer 1/2 architecture, §5.4-§5.7's turn/tool/lifecycle mechanisms, or §5.9's speaker-mapping conclusions."* Tasks 1, 3-9, 12-19 and 21-25 are therefore unchanged by this sync.

---

## Repository and PR structure

This feature spans exactly two git repositories, so it is exactly two PRs. Root `CLAUDE.md` requires one PR per task; the multi-repository split is a structural necessity, not a discretionary split. **Do not split further within a repository.**

| Phase | Repository | Worktree | Branch | Tasks |
|---|---|---|---|---|
| **A** | `~/gitvoipbin/monorepo` | `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen` | `NOJIRA-Insight-AI-realtime-listen` | 0 – 30 |
| **B** | `~/gitvoipbin/monorepo-javascript` | `~/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen` | `NOJIRA-Insight-AI-realtime-listen` | 31 – 33 |

There is **no Jira ticket for this work.** Branch names, commit titles, and PR titles all use `NOJIRA-Insight-AI-realtime-listen`. Never invent a `VOIP-####` number anywhere in code, commits, PR text, or docs.

Phase A's worktree and branch already exist and are rebased onto `main`. Confirm before Task 1:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && git branch --show-current
```
Expected: `NOJIRA-Insight-AI-realtime-listen`

Phase B's worktree does not exist yet; Task 31 creates it.

## Commit convention (every task)

Every task ends in one commit. Title is always the branch name; body lists each affected project:

```
NOJIRA-Insight-AI-realtime-listen

- bin-ai-manager: <what changed>
- bin-common-handler: <what changed>
```

Never add `Co-Authored-By`, "Generated with Claude Code", or any other AI attribution.

## Verification convention (every Go task)

After editing any Go service, run the **full five-step workflow in that service's directory** before committing. No exceptions, no "it's trivial".

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-<service> && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

`vendor/` is gitignored — never `git add -f` it. Commit the resulting `go.mod` / `go.sum` changes with the code.

**`bin-common-handler` is special.** It is consumed by every service through `replace` directives, so a change there requires the full workflow in **every** service. Tasks 3, 4 and 5 touch it. Use this sweep, from the worktree root:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
for d in bin-*-manager voip-*-proxy bin-common-handler; do
  [ -f "$d/go.mod" ] || continue
  echo "=== $d ==="
  ( cd "$d" && go mod tidy && go mod vendor && go generate ./... && go test ./... ) || { echo "FAILED: $d"; break; }
done
```

Expected: every service prints `=== <dir> ===` followed by `ok` / `no test files` lines, and no `FAILED:` line.

## File structure

New files created by this plan:

| Path | Responsibility |
|---|---|
| `bin-ai-manager/pkg/cachehandler/listen.go` | All listen-specific Redis primitives (sets, lists, lock, counter). Kept out of `handler.go` because that file is a pure JSON entity-snapshot cache and these are ephemeral buffers + a distributed rate limiter. |
| `bin-ai-manager/pkg/cachehandler/listen_test.go` | Key-format golden tests for the above. |
| `bin-ai-manager/pkg/aicallhandler/listen_trigger.go` | **The trigger path (design §9's own file name).** `ProcessListen`, `checkListenEligible`, `runListenStart`, `waitForConfbridgeReady`, `startListenTranscribe` (the locked create-or-reuse sequence), `UpdateListenState`, `rollbackListenState`, `isAlreadyProgressing`. |
| `bin-ai-manager/pkg/aicallhandler/listen_trigger_test.go` | Unit tests for the trigger path. |
| `bin-ai-manager/pkg/aicallhandler/listen.go` | The listening *session*: the small shared predicates/metadata readers (`isListenableCallStatus`, `listenTranscribeIDFromMetadata`, `listenOwnsTranscribeFromMetadata`), then `EventTMTranscriptCreated`, `RunListenTurn`, `runListenTurnWithLines`, `buildListenTurnMessages`, `startListenPipecatcall`, `stopListening`, `stopListenByCallID`, `clearListenState`. |
| `bin-ai-manager/pkg/aicallhandler/listen_test.go` | Unit tests for the above. |
| `bin-ai-manager/pkg/aicallhandler/metrics_listen.go` | **Six** of the seven new listen metrics. The seventh, `aicall_foreign_pipecatcall_dropped_total`, lives in `metrics_foreign.go` below because it is emitted from `messagehandler`, not `aicallhandler`, and Task 26 audits it separately (its Step 3, a grep, rather than the Step 1 golden test that pins these six). |
| `bin-ai-manager/pkg/messagehandler/metrics_foreign.go` | `promForeignPipecatcallDroppedTotal` (`aicall_foreign_pipecatcall_dropped_total`) — the seventh new metric, audited separately from the six above. |
| `bin-dbscheme-manager/bin-manager/main/versions/<generated>_ai_aicalls_add_column_listen_call_id.py` | Schema: `ai_aicalls.listen_call_id` + index. |
| `bin-dbscheme-manager/bin-manager/main/versions/<generated>_ai_messages_add_column_origin.py` | Schema: `ai_messages.origin`. |

Existing files are modified in place; each task names them exactly.

---

## Task 0: Pre-flight checks — HUMAN GATE, do not proceed without answers

**This task produces no code and no commit.** It resolves the two BLOCKING open items from design §11, one that this plan discovered, and — **added to reflect design rev 11-14** — the confbridge-readiness retry defaults design §11 item 13 flags as needing confirmation before implementation. **A coding agent must not decide any of these on its own and must not continue to Task 1 until a human has answered all four in writing.** Getting item (a) wrong silently attributes the customer's words to the agent and vice versa, which produces confidently wrong proactive notifications. Getting item (b) wrong silently loses transcript lines to a race.

- [ ] **Step 1: (a) Empirically verify the `in`/`out` → `[CUSTOMER]`/`[AGENT]` speaker mapping (design §5.9, §11 item 1 — narrowed in design rev 11-14, not closed)**

The design's *structural* reading is `in = CUSTOMER`, `out = AGENT`. This is now backed by more than an assumption: `in` is always the listened channel's own party, which is `Case.Peer` — and `case_create` itself guarantees that peer is CRM-eligible (never an internal agent/extension/SIP/conference/AI endpoint), so `in=CUSTOMER` follows from code, not merely from "`Case.ReferenceID` happens to be the customer-facing leg." The general channel-relative *mechanism* (that `direction` is relative to the transcribed channel's own read/write direction, not the call as a whole) was also independently confirmed against a real production transcript sample during design review, not just reasoned from documentation. **What is still open, and this step still exists to close:** neither of those closes the actual empirical gap — one real agent-bridged call's transcript segments have **never been confirmed against known speaker identity end-to-end**, and a reversed mapping is a silent correctness failure regardless of how well-supported the structural reasoning is.

Ask a human to place or stage one agent-bridged call on a Case, with a known script (e.g. customer says "this is the customer speaking", agent says "this is the agent speaking"), then read back the transcripts:

```bash
# Human runs this (needs VPN / production DB access), substituting the transcribe id:
#   SELECT direction, message, tm_transcript
#     FROM ai_transcripts
#    WHERE transcribe_id = UNHEX(REPLACE('<transcribe-uuid>','-',''))
#    ORDER BY tm_transcript ASC;
```

Record the answer here before continuing:

```
in  = CUSTOMER   (PROVISIONAL — proceeding on the design's structural
out = AGENT       reading, not an empirical confirmation. Decided by
                   대표님, 2026-09-04: "구조적 근거로 잠정 진행, 실증은
                   후속 티켓으로." Empirical verification tracked as
                   VOIP-1461. If VOIP-1461 finds the mapping reversed,
                   Task 23's speakerTag function and its golden test
                   must be corrected before this ships to production.)
```

If the confirmed mapping is the reverse of the design's assumption, Task 23's `speakerTag` function must be written with the confirmed mapping, and the golden test in Task 23 pins the confirmed values.

- [ ] **Step 2: (b) Confirm the deployed Redis is ≥ 6.2 (design §11 item 2)**

`LPOP key count` — the single atomic command that drains the pending buffer — was added in Redis 6.2. Without it, a `LLEN` + `LPOP` pair reintroduces exactly the lost-line race the atomic drain exists to prevent, and Task 11 would need a `MULTI`/`EXEC` wrapping `LRANGE` + `LTRIM` instead.

Ask a human to run, against the production Redis instance:

```bash
redis-cli INFO server | grep redis_version
```

Expected: `redis_version:6.2.0` or higher.

```
redis_version = ASSUMED >= 6.2   (PROVISIONAL — not empirically confirmed
                                   against production. Decided by 대표님,
                                   2026-09-04: proceed on the structural
                                   assumption; verification tracked as
                                   VOIP-1462. Task 11 is implemented
                                   against `LPOP key count`. If VOIP-1462
                                   finds the deployed Redis is below 6.2,
                                   ListenPendingPopAll needs a follow-up
                                   PR reworking it to MULTI/EXEC +
                                   LRANGE + LTRIM before this feature is
                                   enabled in that environment.)
```

If it is below 6.2, **stop and report** — Task 11's `ListenPendingPopAll` needs a redesign decision before implementation starts.

- [ ] **Step 3: (c) Confirm the `IDAIManagerListen` sentinel needs no `customer_customers` row (design §11 item 5)**

This plan already gathered the evidence; a human only needs to confirm the conclusion.

Evidence, verified 2026-09-04 against the working tree:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  sed -n '74,92p' bin-customer-manager/models/customer/customer.go
```

`IDAIManager = uuid.FromStringOrNil("00000000-0000-0000-0001-00000000002")`. That literal's last group has **11** hex digits, not 12, so `FromStringOrNil` returns `uuid.Nil`. `bin-customer-manager/models/customer/customer_test.go`'s `TestSpecialIDConstants` says so explicitly in its own comment. `bin-ai-manager/pkg/summaryhandler/start.go` passes this value as the `customerID` of every AI-summary transcribe today, and those sessions work — which proves the sentinel is a **bare UUID constant with no FK-backed `customer_customers` row**, exactly the question §11 item 5 asked.

Two consequences a human must sign off on:

1. `IDAIManagerListen` also needs no row. Task 1 adds a bare constant.
2. `IDAIManagerListen` **must be a well-formed UUID literal.** If it were malformed the same way, it would also parse to `uuid.Nil`, become *identical* to `IDAIManager`, and silently destroy design §5.2.1's entire premise (that listen and `ai_summary` can never collide in `startLive`'s `(customer_id, reference_id, language, ...)` dedup guard). Task 1 pins this with a test.

```
Confirmed: IDAIManagerListen ships as a bare, well-formed UUID constant with no seed row.
(confirmed by: 대표님, 2026-09-04, on the code evidence already presented above —
 IDAIManager's own working precedent in summaryhandler/start.go, and the malformed-
 literal collision risk Task 1's test pins directly.)
```

- [ ] **Step 4: (d) Confirm the five proposed listen timing defaults (design §5.1.1 step 7, §5.2.2, §11 item 13 — three from rev 11-14, two added by rev 17-19)**

Task 20's `waitForConfbridgeReady` (design §5.1.1 step 7) polls, bounded, until a Case's call settles into a live, exactly-2-party confbridge before listening starts — needed because the agent's leg only joins on answer, so a call opened at panel-open/ring time would otherwise read as 1 party for the whole ring window. Task 20's per-AIcall start lock (design §5.2.2) serializes two concurrent `runListenStart` goroutines for the same AIcall. Their proposed defaults (Task 10) are the design's own proposal, **not yet validated against real hold/ring-time distributions**:

```
aicall_listen_confbridge_ready_poll_interval_seconds = 2
aicall_listen_confbridge_ready_max_wait_seconds      = 30
aicall_listen_ensure_goroutine_timeout_seconds       = 45  (must stay > max_wait above)
aicall_listen_start_lock_ttl_seconds                 = 60  (must stay > goroutine timeout above)
aicall_listen_start_lock_release_timeout_seconds     = 3
```

Two design notes worth knowing before signing off.

**On the max-wait budget.** This retry's give-up condition does **not** distinguish "stably wrong topology" from "just a slow ring" (design review round 10 finding HIGH-A — an earlier fast-fail attempt broke a legitimate multi-destination `early_media` scenario, so it was removed). Both share one `skipped_confbridge_not_ready` outcome and one wait budget. That is a reason to err toward a **longer** `max_wait_seconds` default rather than a shorter one, per design §11 item 13's own reasoning — an unusually long real ring time would otherwise look identical, in the metrics, to a genuine topology problem.

**On the lock TTL (design §5.2.2, review round 15 finding HIGH-1).** `60` is *not* derived by summing the RPC timeouts inside the lock — that derivation was tried and withdrawn. It is derived from the goroutine's own outer timeout: no call inside the lock can outlive the `ctx` it runs under, so a TTL strictly greater than `aicall_listen_ensure_goroutine_timeout_seconds` (`45`) can never expire out from under a goroutine that is still legitimately working. **Raising the max-wait value therefore cascades**: max-wait < goroutine timeout < lock TTL must hold, and Task 10 pins both inequalities as standing test invariants. The accepted cost of the larger TTL is slower recovery from a genuine crash (pod loss, where the release `defer` never runs at all) — the lock is then held for the full TTL. A shorter TTL only reopens review round 14's HIGH-2 race in exchange for that faster recovery, which is the wrong trade for a lock whose whole purpose is preventing two writers clobbering a live, billed STT session.

Ask a human whether 30s (max-wait) is long enough for this deployment's actual queue/ring-time distribution, and — if it is raised — that the other two values are raised to preserve the ordering above.

```
Confirmed: the proposed defaults above are acceptable to ship with, unchanged
(2/30/45/60/3). These are config values, not code paths — revisable via a
config-only change post-deploy once real hold/ring-time distributions are
observed, so this does not block implementation the way (a)/(b) do.
(confirmed by: 대표님, 2026-09-04)
```

- [x] **Step 5: Report the four answers and wait**

**Human gate cleared, 2026-09-04.** (a) and (b) proceed provisionally on
structural grounds per 대표님's explicit decision ("구조적 근거로 잠정
진행, 실증은 후속 티켓으로"), tracked as VOIP-1461 and VOIP-1462
respectively. (c) and (d) are confirmed outright. Task 1 may proceed.

**Not a human gate, but flagged here so nobody is surprised later — design §11's other open items and how this plan handles them:**

| Design §11 item | Handling in this plan |
|---|---|
| 1 (`in`/`out` empirical check), 2 (Redis ≥ 6.2), 5 (`IDAIManagerListen` row), 13 (timing defaults) | Steps 1-4 above — the human gate |
| 3, 6, 7, 8, 10 (follow-up tickets) | The "Follow-ups to file as separate Jira tickets" table at the end of this plan |
| 4 (no Jira ticket) | Deliberate: this work is `NOJIRA-*` throughout; never invent a `VOIP-####` |
| 9 (where the listen Redis primitives live) | **Decided in Task 11**: a new `listen.go` file inside the existing `cachehandler`, not a new package |
| 14 (the endpoint returns no listening-status field) | Accepted as-is. Tasks 31/32 fire the `listen` call fire-and-forget precisely because there is nothing to branch on. Recorded as follow-up row 7 |
| 15 (naming) | This plan adopts the design's names for everything the design's own §5.2.2 snippet calls by name; the one deliberate divergence is recorded in Task 11 |
| 16 (`skipped_start_locked` and the other new metric labels) | **Decided in Task 20/26**: the new outcomes get explicit `result` label values rather than folding silently into `skipped_not_listenable`/`failed`. They are label values on the existing `aicall_listen_start_total` family, so Task 26's name-pinning audit is unaffected |

---

## Task 1: `bin-customer-manager` — add the `IDAIManagerListen` system customer id

**Files:**
- Modify: `bin-customer-manager/models/customer/customer.go`
- Test: `bin-customer-manager/models/customer/customer_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bin-customer-manager/models/customer/customer_test.go`:

```go
// TestIDAIManagerListenIsDistinctAndWellFormed pins the two properties the
// InsightAI realtime-listen design (docs/plans/
// 2026-09-03-insight-ai-realtime-listen-design.md §5.2.1) depends on:
//
//  1. IDAIManagerListen parses to a real, non-nil UUID. Several older sentinels
//     in this file (IDEmpty, IDCallManager, IDAIManager) have malformed literals
//     -- their last group has 11 hex digits instead of 12 -- so FromStringOrNil
//     silently yields uuid.Nil for them. A malformed IDAIManagerListen would
//     collapse onto IDAIManager and break property 2.
//  2. IDAIManagerListen != IDAIManager. bin-transcribe-manager's startLive
//     duplicate guard is scoped by (customer_id, reference_id, language,
//     status, deleted). Insight listening and AI summary must own SEPARATE
//     transcribe sessions on the same call; equal owner ids would make them
//     collide and share one session's lifecycle.
func TestIDAIManagerListenIsDistinctAndWellFormed(t *testing.T) {
	if IDAIManagerListen == uuid.Nil {
		t.Errorf("IDAIManagerListen parsed to uuid.Nil -- the literal is malformed (a UUID's last group needs exactly 12 hex digits)")
	}

	if IDAIManagerListen == IDAIManager {
		t.Errorf("IDAIManagerListen must differ from IDAIManager. got both: %s", IDAIManagerListen)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-customer-manager && \
  go test ./models/customer/ -run TestIDAIManagerListenIsDistinctAndWellFormed -v
```
Expected: FAIL — `undefined: IDAIManagerListen`.

- [ ] **Step 3: Add the constant**

In `bin-customer-manager/models/customer/customer.go`, inside the existing `var (...)` block, immediately after the `IDAIManager` line:

```go
	// IDAIManagerListen is the customer id that owns transcribe sessions started
	// by the Insight AI's realtime call-listening feature (docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.2.1). It is deliberately
	// SEPARATE from IDAIManager -- bin-transcribe-manager's startLive duplicate
	// guard is scoped by customer_id, so sharing an owner with the AI-summary
	// feature would make the two collide on one session and entangle their
	// lifecycles. Like IDAIManager, this is a bare sentinel with no
	// customer_customers row behind it.
	//
	// NOTE on the literal: unlike the three sentinels above, this one is
	// well-formed (12 hex digits in the last group). The older literals are not,
	// so they all parse to uuid.Nil -- see TestSpecialIDConstants' own comment.
	// Copying their shape here would silently make this equal to IDAIManager.
	IDAIManagerListen = uuid.FromStringOrNil("00000000-0000-0000-0001-000000000003")
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-customer-manager && \
  go test ./models/customer/ -run TestIDAIManagerListenIsDistinctAndWellFormed -v
```
Expected: PASS.

- [ ] **Step 5: Confirm the whitelist is deliberately NOT touched**

`bin-customer-manager/models/customer/ids.go`'s `IsInternalSystemID` gates certain call-origination paths. Listening never traverses it (design §5.2.1). Verify nothing on the listen path calls it, then leave it alone:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  grep -rn "IsInternalSystemID" --include='*.go' bin-ai-manager bin-transcribe-manager
```
Expected: no output. If there is output, stop and report — the design's scope assumption is wrong.

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-customer-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-customer-manager/models/customer/customer.go bin-customer-manager/models/customer/customer_test.go bin-customer-manager/go.mod bin-customer-manager/go.sum && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-customer-manager: Add IDAIManagerListen system customer id for Insight AI call listening
- bin-customer-manager: Pin the new sentinel as non-nil and distinct from IDAIManager"
```

---

## Task 2: `bin-contact-manager` — add the `kase.ReferenceTypeCall` constant

`Case.ReferenceType` is a plain `string` (there is no typed enum), and every call site today uses the bare literal `"call"`. `checkListenEligible` (Task 20, design §5.1.1 step 5) needs to compare against it, so give it a name in the owning model rather than repeating the literal in `bin-ai-manager`.

**Files:**
- Modify: `bin-contact-manager/models/kase/kase.go`
- Test: `bin-contact-manager/models/kase/kase_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Append to `bin-contact-manager/models/kase/kase_test.go` (create the file with `package kase` and a `"testing"` import if it does not exist):

```go
// TestReferenceTypeCallValue pins the stored value of a Case created from a
// call. Case.ReferenceType is a plain string mirroring
// contact_interactions.reference_type's existing vocabulary, so this constant
// is the single named spelling of a value that was previously written as a bare
// "call" literal at every call site. Changing it silently orphans every stored
// Case row.
func TestReferenceTypeCallValue(t *testing.T) {
	if ReferenceTypeCall != "call" {
		t.Errorf("ReferenceTypeCall mismatch. expected: %q, got: %q", "call", ReferenceTypeCall)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-contact-manager && \
  go test ./models/kase/ -run TestReferenceTypeCallValue -v
```
Expected: FAIL — `undefined: ReferenceTypeCall`.

- [ ] **Step 3: Add the constant**

In `bin-contact-manager/models/kase/kase.go`, immediately after the `Case` struct's closing brace:

```go
// ReferenceTypeCall is the stored ReferenceType value for a Case created from a
// call. Case.ReferenceType is a plain string (it mirrors
// contact_interactions.reference_type's existing vocabulary, not a typed enum),
// so this is an untyped string constant rather than a typed enum member.
//
// Introduced by docs/plans/2026-09-03-insight-ai-realtime-listen-design.md
// §5.1 step 5, which needs to test "was this Case created from a call?" from
// bin-ai-manager without repeating a bare "call" literal across services.
const ReferenceTypeCall = "call"
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-contact-manager && \
  go test ./models/kase/ -run TestReferenceTypeCallValue -v
```
Expected: PASS.

- [ ] **Step 5: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-contact-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-contact-manager/models/kase/ bin-contact-manager/go.mod bin-contact-manager/go.sum && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-contact-manager: Add kase.ReferenceTypeCall constant for the call-created Case reference type"
```

---

## Task 3: `bin-common-handler` — add the `databasehandler.NotEq` filter wrapper

`ApplyFields` today builds only `squirrel.Eq{...}` per field, plus one hardcoded special case for the `"deleted"` key. Task 15 needs `origin != 'listen_internal'` and `role != 'system'` at the SQL layer. Adding another string-keyed special case would not generalise; a typed wrapper does.

**Files:**
- Modify: `bin-common-handler/pkg/databasehandler/main.go`
- Test: `bin-common-handler/pkg/databasehandler/main_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bin-common-handler/pkg/databasehandler/main_test.go` (create it with `package databasehandler` plus `"testing"` and `sq "github.com/Masterminds/squirrel"` imports if it does not exist — check the file's existing import alias first with `head -20`):

```go
// Test_ApplyFields_NotEq pins the NotEq wrapper introduced by docs/plans/
// 2026-09-03-insight-ai-realtime-listen-design.md §5.4.5 step 3. It renders a
// "<>" comparison instead of ApplyFields' default "=", without needing a
// hardcoded field-name special case the way "deleted" has.
func Test_ApplyFields_NotEq(t *testing.T) {
	tests := []struct {
		name        string
		fields      map[string]any
		expectQuery string
		expectArgs  []any
	}{
		{
			name:        "string NotEq renders a not-equal clause",
			fields:      map[string]any{"origin": NotEq{Value: "listen_internal"}},
			expectQuery: "SELECT id FROM t WHERE origin <> ?",
			expectArgs:  []any{"listen_internal"},
		},
		{
			name:        "plain string still renders an equal clause",
			fields:      map[string]any{"origin": "proactive"},
			expectQuery: "SELECT id FROM t WHERE origin = ?",
			expectArgs:  []any{"proactive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := ApplyFields(sq.Select("id").From("t"), tt.fields)
			if err != nil {
				t.Fatalf("ApplyFields returned an unexpected error. err: %v", err)
			}

			query, args, err := builder.ToSql()
			if err != nil {
				t.Fatalf("ToSql returned an unexpected error. err: %v", err)
			}

			if query != tt.expectQuery {
				t.Errorf("query mismatch.\nexpected: %s\ngot:      %s", tt.expectQuery, query)
			}
			if len(args) != len(tt.expectArgs) {
				t.Fatalf("args count mismatch. expected: %d, got: %d (%v)", len(tt.expectArgs), len(args), args)
			}
			for i := range args {
				if args[i] != tt.expectArgs[i] {
					t.Errorf("args[%d] mismatch. expected: %v, got: %v", i, tt.expectArgs[i], args[i])
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-common-handler && \
  go test ./pkg/databasehandler/ -run Test_ApplyFields_NotEq -v
```
Expected: FAIL — `undefined: NotEq`.

- [ ] **Step 3: Add the wrapper type**

In `bin-common-handler/pkg/databasehandler/main.go`, immediately above `func ApplyFields`:

```go
// NotEq wraps a filter value to signal "<>" instead of ApplyFields' default
// "=". It is a typed wrapper rather than another hardcoded field-name special
// case (the "deleted" branch below is not a pattern to extend), so any caller
// can express an exclusion on any field without this package learning that
// field's name.
//
// SCOPE WARNING -- this is deliberately narrower than it looks. ApplyFields'
// per-field switch normalizes some value types before comparing (a uuid.UUID
// goes through .Bytes(); a bool named "deleted" maps onto tm_delete IS [NOT]
// NULL). NotEq bypasses all of that and hands the raw value to squirrel. Only
// use it with string-kind values until it is extended to route through the same
// normalization. A NotEq{Value: someUUID} would silently compare against the
// wrong byte representation.
//
// Introduced by docs/plans/2026-09-03-insight-ai-realtime-listen-design.md
// §5.4.5 step 3.
type NotEq struct{ Value any }
```

- [ ] **Step 4: Handle it in `ApplyFields`**

Inside `ApplyFields`, add a new first case to the existing `switch val := v.(type)` — it must come before `case uuid.UUID:`:

```go
		case NotEq:
			// See NotEq's own doc comment for the value-type scope warning.
			sb = sb.Where(squirrel.NotEq{key: val.Value})
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-common-handler && \
  go test ./pkg/databasehandler/ -run Test_ApplyFields_NotEq -v
```
Expected: PASS, both subtests.

- [ ] **Step 6: Monorepo-wide verification (mandatory — `bin-common-handler` is shared)**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
for d in bin-*-manager voip-*-proxy bin-common-handler; do
  [ -f "$d/go.mod" ] || continue
  echo "=== $d ==="
  ( cd "$d" && go mod tidy && go mod vendor && go generate ./... && go test ./... ) || { echo "FAILED: $d"; break; }
done
```
Expected: every service passes, no `FAILED:` line. Then lint the changed service:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-common-handler && \
  golangci-lint run -v --timeout 5m
```

- [ ] **Step 7: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-common-handler/pkg/databasehandler/ && \
git add $(git diff --name-only -- '*/go.mod' '*/go.sum') && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-common-handler: Add databasehandler.NotEq filter wrapper for SQL not-equal comparisons"
```

---

## Task 4: `bin-common-handler` — add a cache-bypassing `AIV1AIcallGetSkipCache` sibling

Task 19's stale-reply guard drops a bot-LLM message when its pipecatcall id does not match the AIcall's currently-bound one. `AIcallGet` is cache-first and `AIcallUpdate`'s cache refresh discards its own error, so a transient Redis write failure right after a real `Send()` leaves a stale `PipecatcallID` cached — and the guard would then drop the agent's genuine answer. The guard therefore re-reads authoritatively before dropping.

**A sibling method, not a changed signature.** `AIV1AIcallGet` has production callers in `bin-timeline-manager` and three places in `bin-ai-manager`, plus a dozen mock expectations. Changing its signature churns all of them for one new call site.

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/ai_aicalls.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (the `RequestHandler` interface)
- Test: `bin-common-handler/pkg/requesthandler/ai_aicalls_test.go`

- [ ] **Step 1: Write the failing test**

Read the existing `Test_AIV1AIcallGet` in `bin-common-handler/pkg/requesthandler/ai_aicalls_test.go` first so the new test matches its table shape and mock setup exactly:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-common-handler && \
  sed -n '170,230p' pkg/requesthandler/ai_aicalls_test.go
```

Then append a sibling test built on that same shape, asserting only what differs — the URI:

```go
// Test_AIV1AIcallGetSkipCache pins the one thing that differs from
// AIV1AIcallGet: the request URI carries skip_cache=true, which ai-manager's
// listenhandler routes to a cache-bypassing read. Introduced by docs/plans/
// 2026-09-03-insight-ai-realtime-listen-design.md §5.4.4(b).
func Test_AIV1AIcallGetSkipCache(t *testing.T) {
	aicallID := uuid.FromStringOrNil("8f4b9c2a-1d3e-4f5a-9b8c-7d6e5f4a3b2c")

	expectTarget := string(outline.QueueNameAIRequest)
	expectRequest := &sock.Request{
		URI:    "/v1/aicalls/8f4b9c2a-1d3e-4f5a-9b8c-7d6e5f4a3b2c?skip_cache=true",
		Method: sock.RequestMethodGet,
	}
	responseAIcall := amaicall.AIcall{
		Identity: commonidentity.Identity{ID: aicallID},
	}
	responseBody, err := json.Marshal(responseAIcall)
	if err != nil {
		t.Fatalf("could not marshal the response body. err: %v", err)
	}
	response := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       responseBody,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sockHandler: mockSock}
	ctx := context.Background()

	mockSock.EXPECT().RequestPublishWithDelay(gomock.Any(), expectTarget, expectRequest, gomock.Any()).Return(response, nil)

	res, err := reqHandler.AIV1AIcallGetSkipCache(ctx, aicallID)
	if err != nil {
		t.Errorf("Wrong match. expected: ok, got: %v", err)
	}
	if res.ID != aicallID {
		t.Errorf("Wrong match. expected: %s, got: %s", aicallID, res.ID)
	}
}
```

If the existing `Test_AIV1AIcallGet` uses different mock plumbing (e.g. a different `sockHandler` method or a helper constructor), mirror **that** file's actual shape rather than the sketch above — the assertion that matters is the `?skip_cache=true` URI.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-common-handler && \
  go test ./pkg/requesthandler/ -run Test_AIV1AIcallGetSkipCache -v
```
Expected: FAIL — `reqHandler.AIV1AIcallGetSkipCache undefined`.

- [ ] **Step 3: Add the client method**

In `bin-common-handler/pkg/requesthandler/ai_aicalls.go`, immediately after `AIV1AIcallGet`:

```go
// AIV1AIcallGetSkipCache sends a request to ai-manager to get the aicall,
// bypassing ai-manager's own Redis snapshot cache and reading the row from the
// database.
//
// Use it ONLY where a stale read would produce a wrong, irreversible decision.
// The one such site today is bin-ai-manager's messagehandler stale-reply guard
// (docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.4(b)): the
// guard drops a bot-LLM message whose pipecatcall id does not match the
// AIcall's bound one, and AIcallUpdate's cache refresh discards its own error,
// so a transiently-stale cached PipecatcallID would make the guard drop the
// agent's genuine answer. The guard confirms against the database before
// dropping. Everywhere else, prefer AIV1AIcallGet -- this variant defeats the
// cache on purpose and costs a real query.
func (r *requestHandler) AIV1AIcallGetSkipCache(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {

	uri := fmt.Sprintf("/v1/aicalls/%s?skip_cache=true", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls/<aicall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

- [ ] **Step 4: Add it to the `RequestHandler` interface**

In `bin-common-handler/pkg/requesthandler/main.go`, immediately after the existing `AIV1AIcallGet(...)` line:

```go
	AIV1AIcallGetSkipCache(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error)
```

- [ ] **Step 5: Regenerate the mock and run the test**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-common-handler && \
  go generate ./... && go test ./pkg/requesthandler/ -run Test_AIV1AIcallGetSkipCache -v
```
Expected: PASS.

- [ ] **Step 6: Monorepo-wide verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
for d in bin-*-manager voip-*-proxy bin-common-handler; do
  [ -f "$d/go.mod" ] || continue
  echo "=== $d ==="
  ( cd "$d" && go mod tidy && go mod vendor && go generate ./... && go test ./... ) || { echo "FAILED: $d"; break; }
done
```
Expected: every service passes. Adding a method to an interface breaks any hand-written (non-generated) implementation of it; if a service fails here, it has one, and it needs the new method too.

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-common-handler && \
  golangci-lint run -v --timeout 5m
```

- [ ] **Step 7: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-common-handler/pkg/requesthandler/ && \
git add $(git diff --name-only -- '*/go.mod' '*/go.sum') && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-common-handler: Add AIV1AIcallGetSkipCache RPC client for cache-bypassing aicall reads"
```

---

## Task 5: Thread the invoking pipecatcall id into `ToolHandle` (atomic, three services)

`bin-pipecat-manager` knows which pipecatcall a tool call arrived on (`pc.ID`) but never forwards it: `runner.go` calls `AIV1AIcallToolExecute(ctx, pc.ReferenceID, request.ID, ...)`. `ToolHandle` therefore cannot tell a listen evaluation turn from the agent's own Q&A turn — the signal both Task 17's `Origin` tagging and Task 18's `notify_agent` reject-guard need.

**This must be one commit.** Changing `AIV1AIcallToolExecute`'s signature in `bin-common-handler` breaks `bin-pipecat-manager`'s build immediately. In this task `ToolHandle` only *accepts and logs* the new value; Task 17 makes it load-bearing. That keeps this commit green on its own.

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/ai_aicalls.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_aicalls_test.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`
- Modify: `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go` (the `AIcallHandler` interface)
- Modify: `bin-ai-manager/pkg/aicallhandler/tool.go`

- [ ] **Step 1: Add the wire field**

In `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`, replace the `V1DataAIcallsIDToolExecutePost` struct with:

```go
// V1DataAIcallsIDToolExecutePost is
// v1 data type request struct for
// /v1/aicalls/<ai-id>/tool_execute POST
type V1DataAIcallsIDToolExecutePost struct {
	ID       string               `json:"id,omitempty"`
	Type     message.ToolType     `json:"type,omitempty"`
	Function message.FunctionCall `json:"function,omitempty"`

	// PipecatcallID is the pipecatcall session this tool call arrived on.
	//
	// DELIBERATELY OPTIONAL (omitempty, no required-field validation): during a
	// rolling deploy an old bin-pipecat-manager sends no such field, which
	// unmarshals to uuid.Nil. Every consumer treats uuid.Nil as "this is the
	// agent's own Q&A turn" -- the fail-safe direction, since guessing that way
	// costs at most one rejected notify_agent call, whereas guessing "listen
	// turn" would permanently mistag real conversational content. The reverse
	// direction (new pipecat-manager, old ai-manager) is safe too: an old
	// ai-manager ignores an unknown JSON field. No deployment order is forced.
	//
	// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.3a.
	PipecatcallID uuid.UUID `json:"pipecatcall_id,omitempty"`
}
```

Add `uuid "github.com/gofrs/uuid"` to that file's imports if it is not already there.

- [ ] **Step 2: Change the RPC client**

In `bin-common-handler/pkg/requesthandler/ai_aicalls.go`, change `AIV1AIcallToolExecute` to:

```go
func (r *requestHandler) AIV1AIcallToolExecute(
	ctx context.Context,
	aicallID uuid.UUID,
	toolID string,
	toolType ammessage.ToolType,
	function *ammessage.FunctionCall,
	pipecatcallID uuid.UUID,
) (map[string]any, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s/tool_execute", aicallID)

	data := &cbrequest.V1DataAIcallsIDToolExecutePost{
		ID: toolID,

		Type:     toolType,
		Function: *function,

		// The session this tool call arrived on. ai-manager uses it to tell a
		// listen evaluation turn from the agent's own Q&A turn. See
		// docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.3a.
		PipecatcallID: pipecatcallID,
	}
```

Leave the rest of the function body unchanged.

Then update the interface line in `bin-common-handler/pkg/requesthandler/main.go` to match — it is a multi-line signature, so add `pipecatcallID uuid.UUID,` as the last parameter before the closing `)`.

- [ ] **Step 3: Update the existing common-handler test**

In `bin-common-handler/pkg/requesthandler/ai_aicalls_test.go`, `Test_AIV1AIcallToolExecute` calls the method at roughly line 472. Add a `pipecatcallID` field to its test table, pass it as the sixth argument, and assert it lands in the marshalled request body. Read the table's current shape first:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-common-handler && \
  sed -n '412,485p' pkg/requesthandler/ai_aicalls_test.go
```

The call site becomes:

```go
			res, err := reqHandler.AIV1AIcallToolExecute(ctx, tt.aicallID, tt.toolID, tt.toolType, tt.function, tt.pipecatcallID)
```

and each table entry gains `pipecatcallID: uuid.FromStringOrNil("2c1d4e6f-8a9b-4c0d-9e1f-3a5b7c9d1e2f"),` with the expected request body's JSON gaining `"pipecatcall_id":"2c1d4e6f-8a9b-4c0d-9e1f-3a5b7c9d1e2f"`.

- [ ] **Step 4: Forward it from pipecat-manager**

In `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`, change the `AIV1AIcallToolExecute` call to pass `pc.ID`:

```go
	// pc.ID -- the pipecatcall this tool call arrived on -- lets ai-manager tell
	// a listen evaluation turn from the agent's own Q&A turn. See
	// docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.3a.
	res, err := h.requestHandler.AIV1AIcallToolExecute(ctx, pc.ReferenceID, request.ID, ammessage.ToolType(request.Type), &request.Function, pc.ID)
```

- [ ] **Step 5: Change `ToolHandle`'s signature**

In `bin-ai-manager/pkg/aicallhandler/main.go`, replace the `ToolHandle` line in the `AIcallHandler` interface with:

```go
	ToolHandle(ctx context.Context, id uuid.UUID, toolID string, toolType message.ToolType, function message.FunctionCall, pipecatcallID uuid.UUID) (map[string]any, error)
```

In `bin-ai-manager/pkg/aicallhandler/tool.go`, change the implementation's signature and its log fields to match:

```go
func (h *aicallHandler) ToolHandle(ctx context.Context, id uuid.UUID, toolID string, toolType message.ToolType, function message.FunctionCall, pipecatcallID uuid.UUID) (map[string]any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ToolHandle",
		"aicall_id":      id,
		"tool_id":        toolID,
		"tool_type":      toolType,
		"function":       function,
		"pipecatcall_id": pipecatcallID,
	})
```

`pipecatcallID` is consumed only by `ToolHandle` itself (Task 17). The `mapFunctions` value type — `func(context.Context, *aicall.AIcall, *message.ToolCall) *messageContent`, shared by all 21 `toolHandleXxx` functions — is **not** changed; Task 18 passes a resolved `listenTurn bool` to `toolHandleNotifyAgent` outside that map instead, so the other 20 handlers' signatures stay untouched.

- [ ] **Step 6: Pass it through the listenhandler**

In `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`, in `processV1AIcallsIDToolExecutePost`, change the handler call to:

```go
	tmp, err := h.aicallHandler.ToolHandle(ctx, id, req.ID, req.Type, req.Function, req.PipecatcallID)
```

This stays compliant with root `CLAUDE.md`'s transport-DTO-ownership rule: the `request.*` value is unmarshalled here and only unwrapped domain arguments cross into the handler.

- [ ] **Step 7: Update the ai-manager listenhandler tests**

`bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go` has two `ToolHandle` expectations (around lines 399 and 505). Add the sixth argument:

```go
			mockAIcall.EXPECT().ToolHandle(gomock.Any(), tt.expectedID, tt.expectedToolID, tt.expectedToolType, tt.expectedToolFunction, tt.expectedPipecatcallID).Return(tt.responseToolHandle, nil)
```

for the first, adding `expectedPipecatcallID uuid.UUID` to that table and a `"pipecatcall_id":"..."` value in the corresponding request `Data` JSON; and for the second (the error-path test, which already uses `gomock.Any()` throughout) simply add one more `gomock.Any()`.

Add a new table entry to the first test that omits `pipecatcall_id` from the request body entirely, with `expectedPipecatcallID: uuid.Nil` — this pins the rolling-deploy compatibility promise in Step 1's comment.

- [ ] **Step 8: Regenerate mocks and run the touched packages**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
( cd bin-common-handler && go generate ./... ) && \
( cd bin-ai-manager && go generate ./... ) && \
( cd bin-pipecat-manager && go generate ./... ) && \
( cd bin-common-handler && go test ./pkg/requesthandler/ ) && \
( cd bin-ai-manager && go test ./pkg/listenhandler/ ./pkg/aicallhandler/ ) && \
( cd bin-pipecat-manager && go test ./pkg/pipecatcallhandler/ )
```
Expected: all `ok`.

- [ ] **Step 9: Monorepo-wide verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
for d in bin-*-manager voip-*-proxy bin-common-handler; do
  [ -f "$d/go.mod" ] || continue
  echo "=== $d ==="
  ( cd "$d" && go mod tidy && go mod vendor && go generate ./... && go test ./... ) || { echo "FAILED: $d"; break; }
done
```
Then lint the three changed services:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
for d in bin-common-handler bin-pipecat-manager bin-ai-manager; do
  echo "=== $d ==="; ( cd "$d" && golangci-lint run -v --timeout 5m ) || break
done
```

- [ ] **Step 10: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-common-handler/pkg/requesthandler/ bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-ai-manager/pkg/listenhandler/ bin-ai-manager/pkg/aicallhandler/main.go bin-ai-manager/pkg/aicallhandler/tool.go bin-ai-manager/pkg/aicallhandler/mock_main.go && \
git add $(git diff --name-only -- '*/go.mod' '*/go.sum') && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-common-handler: Add pipecatcallID parameter to AIV1AIcallToolExecute
- bin-pipecat-manager: Forward the invoking pipecatcall id on tool_execute
- bin-ai-manager: Accept pipecatcall_id on the tool_execute wire DTO and thread it into ToolHandle"
```

---

## Task 6: `bin-dbscheme-manager` — generate the two schema migrations

**AI drafts these files and commits them. A human applies them.** Never run `alembic upgrade` or `alembic downgrade`. Never hand-pick a revision id — always generate the file with `alembic revision` so the id is unique and `down_revision` chains to the real head.

**Deploy-order warning to carry into Task 30's PR body:** `commondatabasehandler.GetDBFields` builds every `SELECT` column list by reflecting the Go struct. The moment `Message.Origin` and `AIcall.ListenCallID` exist in Go, *every* message and aicall query selects those columns — including queries this feature never touches. **A code deploy landing before its migration is a hard `Unknown column` outage across `bin-ai-manager`, not a soft degradation.** The migrations must be applied before the code deploy reaches any pod.

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<generated>_ai_aicalls_add_column_listen_call_id.py`
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<generated>_ai_messages_add_column_origin.py`

- [ ] **Step 1: Confirm there is exactly one head before starting**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-dbscheme-manager/bin-manager && \
  alembic -c alembic.ini heads
```
Expected: exactly one revision line. If there are two, stop and report — a merge migration is needed first and that is not this feature's job.

- [ ] **Step 2: Generate the first migration file**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-dbscheme-manager/bin-manager && \
  alembic -c alembic.ini revision -m "ai_aicalls_add_column_listen_call_id"
```
Expected: prints `Generating .../main/versions/<hash>_ai_aicalls_add_column_listen_call_id.py ... done`. Note the generated path.

- [ ] **Step 3: Fill in the first migration**

Open the generated file and replace its `upgrade()` / `downgrade()` bodies (leave the auto-generated `revision` / `down_revision` header lines exactly as generated):

```python
def upgrade():
    # listen_call_id records which live call a contact_case Insight AIcall is
    # currently listening to (docs/plans/
    # 2026-09-03-insight-ai-realtime-listen-design.md §5.8).
    #
    # It is a real column rather than a Metadata JSON key for exactly one
    # reason: EventCMCallHangup has to answer "which AIcalls are listening to
    # THIS call id?" and that is a WHERE clause, which JSON metadata cannot
    # serve. Hence the index.
    #
    # NOT NULL DEFAULT 0x00... (not NULL) matches how every other binary(16)
    # id column on this table stores "unset" -- Go's uuid.Nil round-trips
    # through it, and the Go struct field is a plain uuid.UUID, not a pointer.
    op.execute("""
        ALTER TABLE ai_aicalls
          ADD COLUMN listen_call_id BINARY(16) NOT NULL DEFAULT 0x00000000000000000000000000000000
    """)

    op.execute("""
        CREATE INDEX idx_ai_aicalls_listen_call_id ON ai_aicalls(listen_call_id)
    """)


def downgrade():
    op.execute("""
        DROP INDEX idx_ai_aicalls_listen_call_id ON ai_aicalls
    """)

    op.execute("""
        ALTER TABLE ai_aicalls
          DROP COLUMN listen_call_id
    """)
```

- [ ] **Step 4: Generate and fill in the second migration**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-dbscheme-manager/bin-manager && \
  alembic -c alembic.ini revision -m "ai_messages_add_column_origin"
```

Then replace the generated file's bodies:

```python
def upgrade():
    # origin distinguishes a message the AI produced on its own initiative from
    # one that answers or asks something (docs/plans/
    # 2026-09-03-insight-ai-realtime-listen-design.md §5.6.2).
    #
    # Three values in use:
    #   ''                -- OriginNone, the default: every ordinary message.
    #   'proactive'       -- an AI-initiated notification the agent should see;
    #                        the frontends badge it, and it IS replayed into
    #                        future LLM context so the AI remembers what it said.
    #   'listen_internal' -- mechanical tool-call/tool-result rows written during
    #                        a listen evaluation turn; excluded from every future
    #                        LLM replay so they cannot evict the system prompt or
    #                        the agent's own Q&A history.
    #
    # varchar(16) NOT NULL DEFAULT '' matches the role/direction columns' shape
    # on this table, and makes every pre-existing row read back as OriginNone
    # with no backfill.
    op.execute("""
        ALTER TABLE ai_messages
          ADD COLUMN origin VARCHAR(16) NOT NULL DEFAULT ''
    """)


def downgrade():
    op.execute("""
        ALTER TABLE ai_messages
          DROP COLUMN origin
    """)
```

- [ ] **Step 5: Verify there is still exactly one head**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-dbscheme-manager/bin-manager && \
  alembic -c alembic.ini heads
```
Expected: exactly one revision, and it is the `ai_messages_add_column_origin` one.

- [ ] **Step 6: Confirm the column shapes match the Go struct tags they will serve**

The Go fields land in Tasks 7 and 8 with tags `db:"listen_call_id,uuid"` and `db:"origin"`. Sanity-check the naming against the existing table conventions:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  grep -rn "binary(16)\|BINARY(16)" bin-dbscheme-manager/bin-manager/main/versions/ | grep -i "ai_aicalls" | head -3
```
Expected: prior `ai_aicalls` id columns use the same `BINARY(16) NOT NULL DEFAULT 0x00...` form. If they differ, match what is actually there.

- [ ] **Step 7: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-dbscheme-manager/bin-manager/main/versions/ && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-dbscheme-manager: Add ai_aicalls.listen_call_id column and its index
- bin-dbscheme-manager: Add ai_messages.origin column"
```

---

## Task 7: `bin-ai-manager` — add `Message.Origin`

**Files:**
- Modify: `bin-ai-manager/models/message/main.go`
- Modify: `bin-ai-manager/models/message/field.go`
- Modify: `bin-ai-manager/models/message/filters.go`
- Modify: `bin-ai-manager/models/message/webhook.go`
- Test: `bin-ai-manager/models/message/field_test.go`
- Test: `bin-ai-manager/models/message/webhook_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `bin-ai-manager/models/message/field_test.go`:

```go
// Test_FieldOrigin pins the origin field's database column name. It is compared
// against a literal because the value is written into SQL WHERE clauses by
// getPipecatcallMessages (via databasehandler.NotEq) -- a rename here without a
// matching migration is a silent query failure, not a compile error.
func Test_FieldOrigin(t *testing.T) {
	if FieldOrigin != "origin" {
		t.Errorf("FieldOrigin mismatch. expected: %q, got: %q", "origin", FieldOrigin)
	}
}

// Test_OriginValues pins the three Origin values. 'proactive' reaches the
// frontends (they badge on it) and 'listen_internal' reaches tenant webhook
// payloads, so both are external contract, not internal naming.
func Test_OriginValues(t *testing.T) {
	tests := []struct {
		name   string
		origin Origin
		expect string
	}{
		{"none", OriginNone, ""},
		{"proactive", OriginProactive, "proactive"},
		{"listen_internal", OriginListenInternal, "listen_internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.origin) != tt.expect {
				t.Errorf("Origin mismatch. expected: %q, got: %q", tt.expect, string(tt.origin))
			}
		})
	}
}
```

Append to `bin-ai-manager/models/message/webhook_test.go`:

```go
// Test_ConvertWebhookMessage_Origin pins that Origin reaches the external
// surface. The frontends render a proactive notification differently from an
// answer, and they key that entirely off this field
// (docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.10.1) -- if it
// were stripped by ConvertWebhookMessage the badge would silently never appear.
func Test_ConvertWebhookMessage_Origin(t *testing.T) {
	tests := []struct {
		name   string
		origin Origin
	}{
		{"proactive survives conversion", OriginProactive},
		{"listen_internal survives conversion", OriginListenInternal},
		{"empty origin survives conversion", OriginNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{Origin: tt.origin}

			res := m.ConvertWebhookMessage()
			if res.Origin != tt.origin {
				t.Errorf("Origin mismatch. expected: %q, got: %q", tt.origin, res.Origin)
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./models/message/ -run 'Test_FieldOrigin|Test_OriginValues|Test_ConvertWebhookMessage_Origin' -v
```
Expected: FAIL — `undefined: FieldOrigin`, `undefined: Origin`.

- [ ] **Step 3: Add the field and the type**

In `bin-ai-manager/models/message/main.go`, add to the `Message` struct immediately after the `ToolCallID` line:

```go
	// Origin marks whether this message was the AI's own initiative rather than
	// a reply to, or a question from, somebody. Empty for every ordinary
	// message. See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md
	// §5.6.2.
	Origin Origin `json:"origin,omitempty" db:"origin"`
```

and add the type, after the `Direction` block:

```go
// Origin marks how a message came to exist, orthogonally to Role.
//
// It is a string enum rather than a bool, matching Role / Direction /
// DeliveryStatus, so a future third origin does not need another column.
type Origin string

// list of origins
const (
	// OriginNone is the default: every message that answers or asks something.
	OriginNone Origin = ""

	// OriginProactive marks an AI-initiated notification -- something the AI
	// chose to say without being asked, via the notify_agent tool while
	// listening to a live call. It is REAL conversational content: it is stored
	// as role=assistant, it is replayed into future LLM context (so the AI
	// remembers what it told the agent when they ask "what did you mean?"), and
	// the frontends render it with a distinct treatment.
	OriginProactive Origin = "proactive"

	// OriginListenInternal marks the mechanical tool-call and tool-result rows
	// written during a listen evaluation turn. These are NEVER replayed into any
	// future context -- getPipecatcallMessages excludes them at the SQL layer.
	//
	// Without that exclusion they would accumulate (up to 2 rows per turn, up to
	// the per-AIcall turn cap) and push the AIcall's own system prompt and the
	// agent's real Q&A history out of the capped replay window the next time the
	// agent asks a question. See the design doc §5.4.5.
	OriginListenInternal Origin = "listen_internal"
)
```

- [ ] **Step 4: Add the field constant, the filter entry, and the webhook exposure**

In `bin-ai-manager/models/message/field.go`, after `FieldToolCallID`:

```go
	FieldOrigin Field = "origin"
```

In `bin-ai-manager/models/message/filters.go`, add to `FieldStruct` after the `Role` line:

```go
	Origin       Origin    `filter:"origin"`
```

In `bin-ai-manager/models/message/webhook.go`, add to `WebhookMessage` after the `Direction` line:

```go
	Origin    Origin    `json:"origin,omitempty"`
```

and to `ConvertWebhookMessage`'s literal, after `Direction: h.Direction,`:

```go
		Origin:    h.Origin,
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./models/message/ -v
```
Expected: PASS, including any pre-existing golden test over the field list. If a golden test fails because it pins the exact set of `Field` constants or `FieldStruct` members, update its expected list to include `origin` — that is the test doing its job.

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass. If `go test ./...` fails in `pkg/dbhandler` with `Unknown column 'origin'`, the tests are hitting a real database — stop and report, because that means Task 6's migration has not been applied to whatever local instance the tests use, and applying it is a human's call.

- [ ] **Step 7: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-ai-manager/models/message/ bin-ai-manager/go.mod bin-ai-manager/go.sum && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-ai-manager: Add Message.Origin field with proactive and listen_internal values
- bin-ai-manager: Expose origin on the message webhook payload"
```

---

## Task 8: `bin-ai-manager` — add `AIcall.ListenCallID` and the two listen metadata keys

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go`
- Modify: `bin-ai-manager/models/aicall/field.go`
- Modify: `bin-ai-manager/models/aicall/filters.go`
- Test: `bin-ai-manager/models/aicall/field_test.go`
- Test: `bin-ai-manager/models/aicall/filters_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `bin-ai-manager/models/aicall/field_test.go`:

```go
// Test_FieldListenCallID pins the listen_call_id column name. EventCMCallHangup
// filters on it (WHERE listen_call_id = ?) to find every AIcall listening to a
// call that just ended, so a rename without a matching migration means hangup
// cleanup silently stops finding anything.
func Test_FieldListenCallID(t *testing.T) {
	if FieldListenCallID != "listen_call_id" {
		t.Errorf("FieldListenCallID mismatch. expected: %q, got: %q", "listen_call_id", FieldListenCallID)
	}
}

// Test_ListenMetaKeys pins the two Metadata map keys the listen lifecycle
// writes. They are read back by the idempotency check and by every stop path;
// a rename orphans a live listening session's bookkeeping.
func Test_ListenMetaKeys(t *testing.T) {
	if MetaKeyListenTranscribeID != "listen_transcribe_id" {
		t.Errorf("MetaKeyListenTranscribeID mismatch. expected: %q, got: %q", "listen_transcribe_id", MetaKeyListenTranscribeID)
	}
	if MetaKeyListenOwnsTranscribe != "listen_owns_transcribe" {
		t.Errorf("MetaKeyListenOwnsTranscribe mismatch. expected: %q, got: %q", "listen_owns_transcribe", MetaKeyListenOwnsTranscribe)
	}
}
```

Append to `bin-ai-manager/models/aicall/filters_test.go`:

```go
// Test_FieldStruct_ListenCallID pins that listen_call_id is a filterable field.
// Without the struct tag, ConvertFilters drops it and stopListenByCallID's
// AIcallList silently returns every contact_case AIcall on the platform instead
// of the ones listening to the hung-up call.
func Test_FieldStruct_ListenCallID(t *testing.T) {
	field, ok := reflect.TypeOf(FieldStruct{}).FieldByName("ListenCallID")
	if !ok {
		t.Fatalf("FieldStruct has no ListenCallID member")
	}

	if got := field.Tag.Get("filter"); got != "listen_call_id" {
		t.Errorf("filter tag mismatch. expected: %q, got: %q", "listen_call_id", got)
	}
}
```

Add `"reflect"` to that test file's imports if it is not already there.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./models/aicall/ -run 'Test_FieldListenCallID|Test_ListenMetaKeys|Test_FieldStruct_ListenCallID' -v
```
Expected: FAIL — `undefined: FieldListenCallID`.

- [ ] **Step 3: Add the field and the metadata keys**

In `bin-ai-manager/models/aicall/main.go`, add to the `AIcall` struct immediately after the `CurrentMemberID` line:

```go
	// ListenCallID is the live call this contact_case Insight AIcall is
	// currently listening to, or uuid.Nil when it is not listening.
	//
	// A real column rather than a Metadata key for exactly one reason: when a
	// call hangs up, EventCMCallHangup must run WHERE listen_call_id = ? to find
	// every AIcall watching it (plural -- two Cases can share one call), and
	// JSON metadata is not usefully indexable. The transcribe id and ownership
	// flag stay in Metadata precisely because they are only ever read with the
	// row already in hand.
	//
	// Deliberately NOT exposed on the webhook -- internal plumbing, same
	// treatment as Message.PipecatcallID. See docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.8.
	ListenCallID uuid.UUID `json:"listen_call_id,omitempty" db:"listen_call_id,uuid"`
```

and add the two key constants after `MetaKeyAutoAuditEnabled`:

```go
// MetaKeyListenTranscribeID is the Metadata map key (string, a UUID) holding the
// transcribe session this AIcall is reading while listening to a live call.
// Read by the listen-start idempotency check and by every stop path, always with
// the AIcall row already in hand -- hence Metadata rather than a column.
const MetaKeyListenTranscribeID = "listen_transcribe_id"

// MetaKeyListenOwnsTranscribe is the Metadata map key (bool) recording whether
// THIS AIcall started the transcribe session named by MetaKeyListenTranscribeID,
// as opposed to reusing one another AIcall already had running on the same call.
// Only the owner may stop it; a non-owner must never touch a session another
// listener still depends on.
const MetaKeyListenOwnsTranscribe = "listen_owns_transcribe"
```

- [ ] **Step 4: Add the field constant and the filter entry**

In `bin-ai-manager/models/aicall/field.go`, after `FieldCurrentMemberID`:

```go
	FieldListenCallID Field = "listen_call_id"
```

In `bin-ai-manager/models/aicall/filters.go`, add to `FieldStruct` after the `PipecatcallID` line:

```go
	ListenCallID  uuid.UUID      `filter:"listen_call_id"`
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./models/aicall/ -v
```
Expected: PASS. If a pre-existing golden test pins the exact `Field` constant list, add `listen_call_id` to its expected set.

- [ ] **Step 6: Confirm it is NOT on the webhook**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -n "ListenCallID" models/aicall/webhook.go
```
Expected: no output. If there is any, remove it — this field is internal plumbing and adding it would force an OpenAPI and RST change this design deliberately avoids.

- [ ] **Step 7: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-ai-manager/models/aicall/ bin-ai-manager/go.mod bin-ai-manager/go.sum && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-ai-manager: Add AIcall.ListenCallID indexed field for hangup-time listener lookup
- bin-ai-manager: Add listen_transcribe_id and listen_owns_transcribe metadata keys"
```

---

## Task 9: `bin-ai-manager` — add the `WithOrigin` message create option

`messageHandler.Create` already uses a variadic `CreateOption` pattern (`WithPipecatcallID`, `WithDeliveryStatus`, `WithActiveAIID`, `WithInReplyToMessageID`). `Origin` follows it. **Do not add a positional parameter** — `Create` already takes ten and every one of its many call sites would change.

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/main.go`
- Modify: `bin-ai-manager/pkg/messagehandler/db.go`
- Test: `bin-ai-manager/pkg/messagehandler/db_test.go`

- [ ] **Step 1: Write the failing test**

Read the existing `Create` test's table and mock shape first so the new case matches it:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -n "func Test_Create" -A 60 pkg/messagehandler/db_test.go | head -70
```

Then append a focused test:

```go
// Test_Create_WithOrigin pins that WithOrigin lands on the persisted row, and
// that omitting it still yields OriginNone. Origin drives both the frontend
// badge (proactive) and the LLM-replay exclusion (listen_internal), so a
// silently-dropped option is a silent correctness failure in two places.
func Test_Create_WithOrigin(t *testing.T) {
	tests := []struct {
		name string

		opts []CreateOption

		expectOrigin message.Origin
	}{
		{
			name:         "no option yields OriginNone",
			opts:         nil,
			expectOrigin: message.OriginNone,
		},
		{
			name:         "WithOrigin proactive",
			opts:         []CreateOption{WithOrigin(message.OriginProactive)},
			expectOrigin: message.OriginProactive,
		},
		{
			name:         "WithOrigin listen_internal",
			opts:         []CreateOption{WithOrigin(message.OriginListenInternal)},
			expectOrigin: message.OriginListenInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			id := uuid.FromStringOrNil("6b1e8a40-2c7d-4f19-9a03-5e8c1d2f4b60")
			mockUtil.EXPECT().UUIDCreate().Return(id)

			mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).DoAndReturn(
				func(_ context.Context, m *message.Message) error {
					if m.Origin != tt.expectOrigin {
						t.Errorf("Origin mismatch. expected: %q, got: %q", tt.expectOrigin, m.Origin)
					}
					return nil
				})
			mockDB.EXPECT().MessageGet(ctx, id).Return(&message.Message{Origin: tt.expectOrigin}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), message.EventTypeMessageCreated, gomock.Any())

			res, err := h.Create(ctx, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil,
				message.DirectionIncoming, message.RoleAssistant, "hello", nil, "", tt.opts...)
			if err != nil {
				t.Fatalf("Create returned an unexpected error. err: %v", err)
			}
			if res.Origin != tt.expectOrigin {
				t.Errorf("returned Origin mismatch. expected: %q, got: %q", tt.expectOrigin, res.Origin)
			}
		})
	}
}
```

Match the file's real mock constructors and import aliases; if `messageHandler` has extra dependencies the constructor sets, leave them nil as the existing tests do.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/messagehandler/ -run Test_Create_WithOrigin -v
```
Expected: FAIL — `undefined: WithOrigin`.

- [ ] **Step 3: Add the option**

In `bin-ai-manager/pkg/messagehandler/main.go`, add a field to `createParams` after `inReplyToMessageID`:

```go
	origin             message.Origin
```

and the option function after `WithInReplyToMessageID`:

```go
// WithOrigin sets the message origin on createParams.
//
// message.OriginProactive marks an AI-initiated notification (notify_agent);
// message.OriginListenInternal marks the mechanical tool-call/tool-result rows a
// listen evaluation turn writes, which are excluded from every future LLM
// replay. Omitting the option leaves message.OriginNone, which is what every
// ordinary message wants. See docs/plans/
// 2026-09-03-insight-ai-realtime-listen-design.md §5.6.2 and §5.4.5.
func WithOrigin(o message.Origin) CreateOption {
	return func(p *createParams) { p.origin = o }
}
```

- [ ] **Step 4: Plumb it through `Create`**

In `bin-ai-manager/pkg/messagehandler/db.go`, add to the `message.Message` literal inside `Create`, immediately after the `ToolCallID: toolCallID,` line:

```go

		Origin: p.origin,
```

The `createParams` defaults literal needs no change: `message.OriginNone` is the zero value.

- [ ] **Step 5: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/messagehandler/ -run Test_Create_WithOrigin -v
```
Expected: PASS, all three subtests.

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-ai-manager/pkg/messagehandler/ bin-ai-manager/go.mod bin-ai-manager/go.sum && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-ai-manager: Add WithOrigin message create option following the existing CreateOption pattern"
```

---

## Task 10: `bin-ai-manager` — add the thirteen listen configuration flags

Every flag ships with a default that makes the feature inert. `aicall_listen_enabled` defaults to **false**: the feature ships dark and is enabled deliberately.

**Thirteen, matching design §5.12's table row-for-row** (the design's own §9 file list says "thirteen flags" too). Five of them size Task 20's trigger path and are the ones Task 0 Step 4 asks a human to sign off on:

| Flag | Default | Added by |
|---|---|---|
| `aicall_listen_confbridge_ready_poll_interval_seconds` | `2` | design rev 11 (review round 9 BLOCKING-1) |
| `aicall_listen_confbridge_ready_max_wait_seconds` | `30` | design rev 11 |
| `aicall_listen_ensure_goroutine_timeout_seconds` | `45` | design rev 12 (review round 10 MEDIUM-B) |
| `aicall_listen_start_lock_ttl_seconds` | `60` | design rev 17, corrected to `60` in rev 18/19 (review round 14 HIGH-2, round 15 HIGH-1) |
| `aicall_listen_start_lock_release_timeout_seconds` | `3` | design rev 19 (review round 16 MEDIUM-2) |

**Two ordering invariants, both pinned as standing test assertions below, not just as one-time default checks.**

1. `ensure_goroutine_timeout > confbridge_ready_max_wait` — the goroutine encloses the retry loop and needs headroom for the RPC calls each poll makes.
2. `start_lock_ttl > ensure_goroutine_timeout` — design §5.2.2's own derivation (review round 15 finding HIGH-1). No call inside the lock can outlive the `ctx` it runs under, so a TTL above the outer goroutine timeout can never expire out from under a goroutine that is still legitimately working. **The withdrawn derivation was "sum the RPC timeouts inside the lock"** (which would have given ~65s from `TranscribeV1TranscribeList`'s hardcoded 30000ms client timeout alone); do not reintroduce it if these values are ever retuned.

The five defaults above are the design's own proposal, not yet empirically validated against real ring-time distributions (design §11 item 13) — confirm before this task is considered final for implementation, alongside Task 0's other pre-flight items.

**Files:**
- Modify: `bin-ai-manager/internal/config/main.go`
- Test: `bin-ai-manager/internal/config/main_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Append to `bin-ai-manager/internal/config/main_test.go` (create it with `package config` and `"testing"` if absent):

```go
// Test_ListenConfigDefaults pins the shipped defaults for the Insight AI
// realtime-listen flags. The first assertion is the important one: the feature
// must ship dark. The rest guard against a zero-value default silently
// disabling the debounce (interval 0 = a turn per transcript segment, the exact
// unbounded-cost shape the design exists to avoid) or the turn cap.
func Test_ListenConfigDefaults(t *testing.T) {
	SetListenDefaultsForTest()

	cfg := Get()

	if cfg.AIcallListenEnabled {
		t.Errorf("AIcallListenEnabled must default to false -- the feature ships dark")
	}
	if cfg.AIcallListenEvaluateIntervalSeconds != 20 {
		t.Errorf("AIcallListenEvaluateIntervalSeconds mismatch. expected: 20, got: %d", cfg.AIcallListenEvaluateIntervalSeconds)
	}
	if cfg.AIcallListenWindowSize != 40 {
		t.Errorf("AIcallListenWindowSize mismatch. expected: 40, got: %d", cfg.AIcallListenWindowSize)
	}
	if cfg.AIcallListenQAContextSize != 10 {
		t.Errorf("AIcallListenQAContextSize mismatch. expected: 10, got: %d", cfg.AIcallListenQAContextSize)
	}
	if cfg.AIcallListenMaxTurnsPerAIcall != 60 {
		t.Errorf("AIcallListenMaxTurnsPerAIcall mismatch. expected: 60, got: %d", cfg.AIcallListenMaxTurnsPerAIcall)
	}
	if cfg.AIcallListenBufferTTLHours != 6 {
		t.Errorf("AIcallListenBufferTTLHours mismatch. expected: 6, got: %d", cfg.AIcallListenBufferTTLHours)
	}
	if cfg.AIcallListenTurnPipecatcallIDTTLSeconds != 180 {
		t.Errorf("AIcallListenTurnPipecatcallIDTTLSeconds mismatch. expected: 180, got: %d", cfg.AIcallListenTurnPipecatcallIDTTLSeconds)
	}
	if cfg.AIcallListenDefaultLanguage != "en-US" {
		t.Errorf("AIcallListenDefaultLanguage mismatch. expected: %q, got: %q", "en-US", cfg.AIcallListenDefaultLanguage)
	}
	if cfg.AIcallListenConfbridgeReadyPollIntervalSeconds != 2 {
		t.Errorf("AIcallListenConfbridgeReadyPollIntervalSeconds mismatch. expected: 2, got: %d", cfg.AIcallListenConfbridgeReadyPollIntervalSeconds)
	}
	if cfg.AIcallListenConfbridgeReadyMaxWaitSeconds != 30 {
		t.Errorf("AIcallListenConfbridgeReadyMaxWaitSeconds mismatch. expected: 30, got: %d", cfg.AIcallListenConfbridgeReadyMaxWaitSeconds)
	}
	if cfg.AIcallListenEnsureGoroutineTimeoutSeconds != 45 {
		t.Errorf("AIcallListenEnsureGoroutineTimeoutSeconds mismatch. expected: 45, got: %d", cfg.AIcallListenEnsureGoroutineTimeoutSeconds)
	}
	if cfg.AIcallListenStartLockTTLSeconds != 60 {
		t.Errorf("AIcallListenStartLockTTLSeconds mismatch. expected: 60, got: %d", cfg.AIcallListenStartLockTTLSeconds)
	}
	if cfg.AIcallListenStartLockReleaseTimeoutSeconds != 3 {
		t.Errorf("AIcallListenStartLockReleaseTimeoutSeconds mismatch. expected: 3, got: %d", cfg.AIcallListenStartLockReleaseTimeoutSeconds)
	}
	// The goroutine timeout must have headroom over the max-wait budget it
	// encloses -- pinned here as a standing invariant, not just a one-time
	// default check, since the two are set independently.
	if cfg.AIcallListenEnsureGoroutineTimeoutSeconds <= cfg.AIcallListenConfbridgeReadyMaxWaitSeconds {
		t.Errorf("AIcallListenEnsureGoroutineTimeoutSeconds (%d) must be strictly greater than AIcallListenConfbridgeReadyMaxWaitSeconds (%d)", cfg.AIcallListenEnsureGoroutineTimeoutSeconds, cfg.AIcallListenConfbridgeReadyMaxWaitSeconds)
	}
	// And the start lock's TTL must in turn exceed that goroutine timeout
	// (design §5.2.2, review round 15 finding HIGH-1): the lock must never be
	// able to expire out from under a goroutine that is still working inside
	// its own legitimate budget, because a second goroutine acquiring it would
	// reopen exactly the same-AIcall clobbering race the lock exists to close.
	if cfg.AIcallListenStartLockTTLSeconds <= cfg.AIcallListenEnsureGoroutineTimeoutSeconds {
		t.Errorf("AIcallListenStartLockTTLSeconds (%d) must be strictly greater than AIcallListenEnsureGoroutineTimeoutSeconds (%d)", cfg.AIcallListenStartLockTTLSeconds, cfg.AIcallListenEnsureGoroutineTimeoutSeconds)
	}
	// The release bound is a small, independent timeout on the DETACHED
	// context the lock's Release call runs under -- it must stay far below the
	// TTL, and must never be conflated with it.
	if cfg.AIcallListenStartLockReleaseTimeoutSeconds >= cfg.AIcallListenStartLockTTLSeconds {
		t.Errorf("AIcallListenStartLockReleaseTimeoutSeconds (%d) must stay well below AIcallListenStartLockTTLSeconds (%d)", cfg.AIcallListenStartLockReleaseTimeoutSeconds, cfg.AIcallListenStartLockTTLSeconds)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./internal/config/ -run Test_ListenConfigDefaults -v
```
Expected: FAIL — `undefined: SetListenDefaultsForTest`.

- [ ] **Step 3: Add the `Config` fields**

In `bin-ai-manager/internal/config/main.go`, add to the `Config` struct after `AIcallSendCooldownSeconds`:

```go
	// Insight AI realtime call listening (docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.12).
	AIcallListenEnabled                     bool   // Master kill switch. False ships the feature dark; a rollback to false stops in-flight sessions at their next evaluated turn.
	AIcallListenEvaluateIntervalSeconds     int    // Debounce window. One listen evaluation turn per AIcall per this many seconds, regardless of how many sentences were spoken -- this is what decouples LLM cost from speech volume.
	AIcallListenWindowSize                  int    // Rolling transcript lines kept for continuity across turns and fed into each turn's context.
	AIcallListenQAContextSize               int    // Q&A message rows replayed into a listen turn's context, so the AI has continuity with what the agent asked.
	AIcallListenMaxTurnsPerAIcall           int    // Hard per-AIcall turn cap. Reaching it stops listening cleanly -- the backstop against a pathologically long call.
	AIcallListenBufferTTLHours              int    // TTL on the pending/window/lock/turn-count Redis keys. NOT the turn-id set, which has its own much shorter TTL below.
	AIcallListenTurnPipecatcallIDTTLSeconds int    // TTL on the registered listen-turn pipecatcall id set entries. Only needs to outlive one turn; generous headroom is cheap and self-expiring.
	AIcallListenDefaultLanguage             string // STT language used when the AIcall carries no STTLanguage.

	// The trigger path's own timings (design §5.1.1 step 7, §5.2.2, §5.12) --
	// they size waitForConfbridgeReady's bounded retry, the goroutine that
	// encloses it, and the per-AIcall create-or-reuse lock.
	//
	// ORDERING INVARIANT, pinned by Test_ListenConfigDefaults:
	//   ConfbridgeReadyMaxWait < EnsureGoroutineTimeout < StartLockTTL
	AIcallListenConfbridgeReadyPollIntervalSeconds int // Poll interval for waitForConfbridgeReady.
	AIcallListenConfbridgeReadyMaxWaitSeconds      int // Total wait budget before giving up with skipped_confbridge_not_ready. Must stay strictly less than AIcallListenEnsureGoroutineTimeoutSeconds below.
	AIcallListenEnsureGoroutineTimeoutSeconds      int // runListenStart's own detached-goroutine timeout -- purpose-built for this feature, not inherited from any other detached-goroutine pattern in this package.
	AIcallListenStartLockTTLSeconds                int // TTL on ai:listen:startlock:<aicall_id>, the per-AIcall lock serializing concurrent runListenStart create-or-reuse sequences. Must strictly EXCEED AIcallListenEnsureGoroutineTimeoutSeconds so it can never expire under a goroutine still working inside its own budget -- NOT derived by summing the RPC timeouts inside the lock, a derivation the design tried and withdrew.
	AIcallListenStartLockReleaseTimeoutSeconds     int // Bound on the DETACHED context (context.WithTimeout(context.WithoutCancel(ctx), ...)) the lock's Release call runs under, so a stuck Redis call during cleanup cannot hang the releasing goroutine. Independent of, and far below, the TTL above.
```

- [ ] **Step 4: Register the flags and env bindings**

In `bindConfig`, after the `aicall_send_cooldown_seconds` line:

```go
	f.Bool("aicall_listen_enabled", false, "Master kill switch for Insight AI realtime call listening")
	f.Int("aicall_listen_evaluate_interval_seconds", 20, "Debounce window (seconds) between Insight AI listen evaluation turns on one AIcall")
	f.Int("aicall_listen_window_size", 40, "Rolling transcript lines kept in a listen turn's context")
	f.Int("aicall_listen_qa_context_size", 10, "Q&A message rows replayed into a listen turn's context")
	f.Int("aicall_listen_max_turns_per_aicall", 60, "Hard cap on listen evaluation turns per AIcall")
	f.Int("aicall_listen_buffer_ttl_hours", 6, "TTL (hours) on the listen buffer, lock and turn-count Redis keys")
	f.Int("aicall_listen_turn_pipecatcall_id_ttl_seconds", 180, "TTL (seconds) on registered listen-turn pipecatcall id set entries")
	f.String("aicall_listen_default_language", "en-US", "STT language for listening when the AIcall has none set")
	f.Int("aicall_listen_confbridge_ready_poll_interval_seconds", 2, "Poll interval (seconds) for the bounded confbridge-readiness retry before listening starts")
	f.Int("aicall_listen_confbridge_ready_max_wait_seconds", 30, "Total wait budget (seconds) for the confbridge-readiness retry before giving up")
	f.Int("aicall_listen_ensure_goroutine_timeout_seconds", 45, "Timeout (seconds) for runListenStart's own detached goroutine; must stay strictly greater than aicall_listen_confbridge_ready_max_wait_seconds")
	f.Int("aicall_listen_start_lock_ttl_seconds", 60, "TTL (seconds) on the per-AIcall listen-start lock; must stay strictly greater than aicall_listen_ensure_goroutine_timeout_seconds")
	f.Int("aicall_listen_start_lock_release_timeout_seconds", 3, "Timeout (seconds) on the detached context the listen-start lock's release runs under")
```

and to the `bindings` map, after the `"aicall_send_cooldown_seconds"` entry:

```go
		"aicall_listen_enabled":                                    "AICALL_LISTEN_ENABLED",
		"aicall_listen_evaluate_interval_seconds":                  "AICALL_LISTEN_EVALUATE_INTERVAL_SECONDS",
		"aicall_listen_window_size":                                "AICALL_LISTEN_WINDOW_SIZE",
		"aicall_listen_qa_context_size":                             "AICALL_LISTEN_QA_CONTEXT_SIZE",
		"aicall_listen_max_turns_per_aicall":                        "AICALL_LISTEN_MAX_TURNS_PER_AICALL",
		"aicall_listen_buffer_ttl_hours":                            "AICALL_LISTEN_BUFFER_TTL_HOURS",
		"aicall_listen_turn_pipecatcall_id_ttl_seconds":             "AICALL_LISTEN_TURN_PIPECATCALL_ID_TTL_SECONDS",
		"aicall_listen_default_language":                            "AICALL_LISTEN_DEFAULT_LANGUAGE",
		"aicall_listen_confbridge_ready_poll_interval_seconds":      "AICALL_LISTEN_CONFBRIDGE_READY_POLL_INTERVAL_SECONDS",
		"aicall_listen_confbridge_ready_max_wait_seconds":           "AICALL_LISTEN_CONFBRIDGE_READY_MAX_WAIT_SECONDS",
		"aicall_listen_ensure_goroutine_timeout_seconds":            "AICALL_LISTEN_ENSURE_GOROUTINE_TIMEOUT_SECONDS",
		"aicall_listen_start_lock_ttl_seconds":                      "AICALL_LISTEN_START_LOCK_TTL_SECONDS",
		"aicall_listen_start_lock_release_timeout_seconds":          "AICALL_LISTEN_START_LOCK_RELEASE_TIMEOUT_SECONDS",
```

- [ ] **Step 5: Load them in `LoadGlobalConfig`**

Add to the `globalConfig = Config{...}` literal, after `AIcallSendCooldownSeconds`:

```go

			AIcallListenEnabled:                     viper.GetBool("aicall_listen_enabled"),
			AIcallListenEvaluateIntervalSeconds:     viper.GetInt("aicall_listen_evaluate_interval_seconds"),
			AIcallListenWindowSize:                  viper.GetInt("aicall_listen_window_size"),
			AIcallListenQAContextSize:               viper.GetInt("aicall_listen_qa_context_size"),
			AIcallListenMaxTurnsPerAIcall:           viper.GetInt("aicall_listen_max_turns_per_aicall"),
			AIcallListenBufferTTLHours:              viper.GetInt("aicall_listen_buffer_ttl_hours"),
			AIcallListenTurnPipecatcallIDTTLSeconds: viper.GetInt("aicall_listen_turn_pipecatcall_id_ttl_seconds"),
			AIcallListenDefaultLanguage:             viper.GetString("aicall_listen_default_language"),
			AIcallListenConfbridgeReadyPollIntervalSeconds: viper.GetInt("aicall_listen_confbridge_ready_poll_interval_seconds"),
			AIcallListenConfbridgeReadyMaxWaitSeconds:      viper.GetInt("aicall_listen_confbridge_ready_max_wait_seconds"),
			AIcallListenEnsureGoroutineTimeoutSeconds:      viper.GetInt("aicall_listen_ensure_goroutine_timeout_seconds"),
			AIcallListenStartLockTTLSeconds:                viper.GetInt("aicall_listen_start_lock_ttl_seconds"),
			AIcallListenStartLockReleaseTimeoutSeconds:     viper.GetInt("aicall_listen_start_lock_release_timeout_seconds"),
```

- [ ] **Step 6: Add the test helpers**

At the end of `bin-ai-manager/internal/config/main.go`, following the existing `SetXxxForTest` convention:

```go
// SetListenDefaultsForTest populates the Insight AI listen flags in the global
// config with their shipped defaults, without going through the
// Bootstrap+LoadGlobalConfig path (LoadGlobalConfig is sync.Once-guarded, so a
// test cannot re-run it).
// USE ONLY FROM TESTS.
func SetListenDefaultsForTest() {
	globalConfig.AIcallListenEnabled = false
	globalConfig.AIcallListenEvaluateIntervalSeconds = 20
	globalConfig.AIcallListenWindowSize = 40
	globalConfig.AIcallListenQAContextSize = 10
	globalConfig.AIcallListenMaxTurnsPerAIcall = 60
	globalConfig.AIcallListenBufferTTLHours = 6
	globalConfig.AIcallListenTurnPipecatcallIDTTLSeconds = 180
	globalConfig.AIcallListenDefaultLanguage = "en-US"
	globalConfig.AIcallListenConfbridgeReadyPollIntervalSeconds = 2
	globalConfig.AIcallListenConfbridgeReadyMaxWaitSeconds = 30
	globalConfig.AIcallListenEnsureGoroutineTimeoutSeconds = 45
	globalConfig.AIcallListenStartLockTTLSeconds = 60
	globalConfig.AIcallListenStartLockReleaseTimeoutSeconds = 3
}

// SetAIcallListenStartLockTTLForTest overrides the per-AIcall listen-start
// lock's TTL in tests, so Task 20's "simulated crash, lock held for the full
// TTL" row does not have to wait 60 real seconds.
// USE ONLY FROM TESTS.
func SetAIcallListenStartLockTTLForTest(seconds int) {
	globalConfig.AIcallListenStartLockTTLSeconds = seconds
}

// SetAIcallListenConfbridgeReadyPollIntervalForTest overrides
// waitForConfbridgeReady's poll interval in tests. Combine with
// SetAIcallListenConfbridgeReadyMaxWaitForTest to keep the timeout-driven
// rows of Test_waitForConfbridgeReady (Task 20) fast.
// USE ONLY FROM TESTS.
func SetAIcallListenConfbridgeReadyPollIntervalForTest(seconds int) {
	globalConfig.AIcallListenConfbridgeReadyPollIntervalSeconds = seconds
}

// SetAIcallListenConfbridgeReadyMaxWaitForTest overrides
// waitForConfbridgeReady's wait budget in tests.
// USE ONLY FROM TESTS.
func SetAIcallListenConfbridgeReadyMaxWaitForTest(seconds int) {
	globalConfig.AIcallListenConfbridgeReadyMaxWaitSeconds = seconds
}

// SetAIcallListenEnabledForTest overrides the listen kill switch in tests.
// USE ONLY FROM TESTS.
func SetAIcallListenEnabledForTest(enabled bool) {
	globalConfig.AIcallListenEnabled = enabled
}

// SetAIcallListenMaxTurnsPerAIcallForTest overrides the per-AIcall turn cap in
// tests, so a cap-exceeded path can be exercised without running 60 turns.
// USE ONLY FROM TESTS.
func SetAIcallListenMaxTurnsPerAIcallForTest(turns int) {
	globalConfig.AIcallListenMaxTurnsPerAIcall = turns
}

// SetAIcallListenWindowSizeForTest overrides the rolling transcript window size
// in tests.
// USE ONLY FROM TESTS.
func SetAIcallListenWindowSizeForTest(size int) {
	globalConfig.AIcallListenWindowSize = size
}

// SetAIcallListenQAContextSizeForTest overrides the Q&A context row budget in
// tests.
// USE ONLY FROM TESTS.
func SetAIcallListenQAContextSizeForTest(size int) {
	globalConfig.AIcallListenQAContextSize = size
}
```

- [ ] **Step 7: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./internal/config/ -run Test_ListenConfigDefaults -v
```
Expected: PASS.

- [ ] **Step 8: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 9: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
git add bin-ai-manager/internal/config/ bin-ai-manager/go.mod bin-ai-manager/go.sum && \
git commit -m "NOJIRA-Insight-AI-realtime-listen

- bin-ai-manager: Add thirteen aicall_listen_* configuration flags, defaulting the feature to disabled"
```

---

## Task 11: `bin-ai-manager` — add the listen Redis primitives to `cachehandler`

`cachehandler` today is a pure JSON entity-snapshot cache: two primitives (`getSerialize` / `setSerialize`), a fixed 24h TTL, no raw Redis data structures. Listening needs sets, lists, a distributed lock and a counter — a structurally different responsibility.

**Decision (design §11 item 9, decided here):** extend `cachehandler` with a **new file**, `listen.go`, rather than create a new package. It shares the same `*redis.Client` and the same `CacheHandler` interface, so `aicallHandler` takes one dependency instead of two, and the generated mock covers everything in one place. The separate file keeps the snapshot cache uncontaminated.

**These keys are NOT part of `AIcallSet`'s snapshot-index scheme.** That scheme writes secondary keys and never invalidates a stale one when the indexed field changes. These keys are explicitly written at listen start and explicitly removed at listen stop.

**The per-AIcall start lock is here too (design §5.2.2, rev 18/19).** `ListenStartLockAcquire` / `ListenStartLockRelease` are that lock's **only** two entry points, and the `ai:listen:startlock:<aicall_id>` key format is built in exactly one place — inside these two functions. Design review round 16 (finding LOW-6) specifically rejected the earlier shape, a raw inline `SetNX` at the call site paired with a named release helper, because it duplicated the key format across two files where the two could drift apart. **Do not call `SetNX` for this lock from `aicallHandler`.** Acquire is a thin `SetNX` wrapper (the same underlying Redis command the debounce lock below already uses — no new primitive); Release is genuinely new: an atomic compare-and-delete via a single Redis `EVAL`.

**One deliberate naming divergence from the design, recorded rather than silently taken.** Design §9's `cachehandler` bullet names five listen primitives beyond the buffers: `ListenTurnPipecatcallIDAdd`, `ListenTurnPipecatcallIDIsMember`, `ListenTranscribeAIcallRemove`, `ListenStartLockAcquire`, `ListenStartLockRelease`. This plan uses the design's exact names for four of them. The fifth, `ListenTranscribeAIcallRemove`, ships here as **`ListenAIcallIDRemove`** — the same primitive (SREM this AIcall's id from `ai:listen:transcribe:<transcribe_id>`), named to sit symmetrically alongside its own `ListenAIcallIDsGet` / `ListenAIcallIDAdd` siblings, which the design never names at all. Design §11 item 15 states plainly that these names are "this design's proposed names … fine to bikeshed at implementation time; not a design-level decision," and this plan's own front matter says the plan wins on signatures. Wherever the design's §5.2.2 snippet says `ListenTranscribeAIcallRemove`, this plan says `ListenAIcallIDRemove` and says so at the call site.

**Files:**
- Create: `bin-ai-manager/pkg/cachehandler/listen.go`
- Create: `bin-ai-manager/pkg/cachehandler/listen_test.go`
- Modify: `bin-ai-manager/pkg/cachehandler/main.go` (the `CacheHandler` interface)

- [ ] **Step 1: Write the failing key-format test**

Create `bin-ai-manager/pkg/cachehandler/listen_test.go`:

```go
package cachehandler

import (
	"testing"

	uuid "github.com/gofrs/uuid"
)

// Test_listenKeys pins every listen Redis key format. These keys are the
// contract between the intake path (which resolves ownership by SMEMBERS on a
// key built from a transcribe id) and the listen lifecycle (which writes and
// removes membership on the same key). A format drift between the two would
// silently stop every transcript segment from ever being matched -- with no
// error and no metric, since "not one of ours" is the overwhelmingly common
// case the intake path is designed to drop cheaply.
func Test_listenKeys(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")

	tests := []struct {
		name   string
		got    string
		expect string
	}{
		{"transcribe resolver set", listenTranscribeKey(transcribeID), "ai:listen:transcribe:11111111-2222-3333-4444-555555555555"},
		{"pending buffer list", listenPendingKey(aicallID), "ai:listen:pending:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"rolling window list", listenWindowKey(aicallID), "ai:listen:window:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"debounce lock", listenLockKey(aicallID), "ai:listen:lock:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"turn counter", listenTurnsKey(aicallID), "ai:listen:turns:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"turn pipecatcall id set", listenTurnPipecatcallIDKey(aicallID), "ai:listen:turnpcid:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		// The start lock's key format is built in exactly ONE place, inside
		// ListenStartLockAcquire/Release (design §5.2.2, review round 16
		// finding LOW-6). Pinning it here is what keeps a future caller from
		// re-deriving it inline and drifting.
		{"start lock", listenStartLockKey(aicallID), "ai:listen:startlock:66666666-7777-8888-9999-aaaaaaaaaaaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expect {
				t.Errorf("key mismatch.\nexpected: %s\ngot:      %s", tt.expect, tt.got)
			}
		})
	}
}

// Test_listenPendingPopMax pins the atomic-drain bound. LPOP key count (Redis
// >= 6.2) is what makes draining the pending buffer a single atomic command, so
// a concurrent appender can never lose a line between a read and a trim. The
// count argument is REQUIRED by go-redis v8's LPopCount, so there must be a
// bound; it exists to cap one turn's context, not to be tuned.
func Test_listenPendingPopMax(t *testing.T) {
	if listenPendingPopMax != 500 {
		t.Errorf("listenPendingPopMax mismatch. expected: 500, got: %d", listenPendingPopMax)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/cachehandler/ -run 'Test_listenKeys|Test_listenPendingPopMax' -v
```
Expected: FAIL — `undefined: listenTranscribeKey`.

- [ ] **Step 3: Write the implementation**

Create `bin-ai-manager/pkg/cachehandler/listen.go`:

```go
package cachehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	uuid "github.com/gofrs/uuid"
)

// Redis state backing the Insight AI's realtime call listening
// (docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.2.4, §5.3,
// §5.4).
//
// These keys are deliberately NOT part of AIcallSet's snapshot-index scheme in
// handler.go. That scheme writes secondary keys pointing at a serialized entity
// and never invalidates a stale one when the indexed field changes; reusing it
// here would leave stale pointers and collide every non-listening AIcall on a
// shared nil-UUID key. These are purpose-built, explicitly-managed pointers
// with explicit lifecycles: written at listen start, removed at listen stop,
// and TTL'd as a backstop against a lost stop.
//
// Cache-loss behaviour is deliberate and stated: a Redis flush drops the
// resolver keys, so in-flight calls stop being listened to until the panel is
// reopened (which re-runs the listen-start path and repopulates). There is no
// DB fallback on a miss, because a DB fallback would put a query on the
// platform-wide transcript_created hot path -- exactly the cost this design
// removes.

// listenPendingPopMax bounds a single atomic drain of the pending buffer.
//
// go-redis v8's LPopCount requires a count argument, and the atomicity is the
// whole point: LLEN followed by a separate LPOP would reintroduce the race
// where a line pushed between the two calls is silently lost. 500 lines is far
// beyond any realistic debounce interval's worth of speech, so in practice this
// drains everything in one command.
const listenPendingPopMax = 500

func listenTranscribeKey(transcribeID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:transcribe:%s", transcribeID)
}

func listenPendingKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:pending:%s", aicallID)
}

func listenWindowKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:window:%s", aicallID)
}

func listenLockKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:lock:%s", aicallID)
}

func listenTurnsKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:turns:%s", aicallID)
}

func listenTurnPipecatcallIDKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:turnpcid:%s", aicallID)
}

func listenStartLockKey(aicallID uuid.UUID) string {
	return fmt.Sprintf("ai:listen:startlock:%s", aicallID)
}

// listenStartLockReleaseScript is the start lock's compare-and-delete.
//
// It MUST be one EVAL, not a GET followed by a DEL. A separate GET-then-DEL
// could observe our own token, then have this goroutine's TTL lapse and a
// second goroutine acquire the key, and then delete THAT goroutine's still-live
// lock -- which is precisely the clobbering the lock exists to prevent
// (design §5.2.2, review round 15 finding HIGH-1(b)).
const listenStartLockReleaseScript = `if redis.call("GET",KEYS[1])==ARGV[1] then return redis.call("DEL",KEYS[1]) else return 0 end`

// ListenAIcallIDsGet returns every AIcall id currently listening to the given
// transcribe session.
//
// A SET, not a single value: N AIcalls can share one transcribe session (two
// Cases open on one call each get their own AIcall, and the second reuses the
// first's session). A single-valued key would let the second listener silently
// overwrite the first's mapping -- the first would stop receiving segments for
// the rest of the call, with no error and no metric.
//
// This is the ONE Redis round trip the platform-wide transcript_created hot
// path pays per final STT result. An empty result means "not a session we
// started" and is the overwhelmingly common outcome.
func (h *handler) ListenAIcallIDsGet(ctx context.Context, transcribeID uuid.UUID) ([]uuid.UUID, error) {
	tmp, err := h.Cache.SMembers(ctx, listenTranscribeKey(transcribeID)).Result()
	if err != nil {
		return nil, err
	}

	res := []uuid.UUID{}
	for _, m := range tmp {
		id := uuid.FromStringOrNil(m)
		if id == uuid.Nil {
			// A malformed member cannot address an AIcall; skip rather than
			// fail the whole resolution for the other listeners.
			continue
		}
		res = append(res, id)
	}

	return res, nil
}

// ListenAIcallIDAdd registers this AIcall as a listener on the transcribe
// session. Every listener adds only itself, so cleanup can remove only itself.
func (h *handler) ListenAIcallIDAdd(ctx context.Context, transcribeID uuid.UUID, aicallID uuid.UUID, ttl time.Duration) error {
	key := listenTranscribeKey(transcribeID)

	if err := h.Cache.SAdd(ctx, key, aicallID.String()).Err(); err != nil {
		return err
	}

	return h.Cache.Expire(ctx, key, ttl).Err()
}

// ListenAIcallIDRemove removes only THIS AIcall's membership.
//
// Never DEL the key: another AIcall may still be listening to the same shared
// transcribe session, and deleting the whole key would cut it off silently.
// Redis removes the key itself once the set empties.
func (h *handler) ListenAIcallIDRemove(ctx context.Context, transcribeID uuid.UUID, aicallID uuid.UUID) error {
	return h.Cache.SRem(ctx, listenTranscribeKey(transcribeID), aicallID.String()).Err()
}

// ListenPendingPush appends one transcript line to the not-yet-evaluated
// buffer.
func (h *handler) ListenPendingPush(ctx context.Context, aicallID uuid.UUID, line string, ttl time.Duration) error {
	key := listenPendingKey(aicallID)

	if err := h.Cache.RPush(ctx, key, line).Err(); err != nil {
		return err
	}

	return h.Cache.Expire(ctx, key, ttl).Err()
}

// ListenPendingPopAll atomically drains the pending buffer.
//
// One LPOP key count command (Redis >= 6.2), not LLEN followed by LPOP: the
// atomicity is what guarantees no concurrent appender's line is lost between a
// read and a trim. Returns an empty slice (not an error) when the buffer is
// empty -- redis.Nil is the normal "nothing there" signal for LPOP.
func (h *handler) ListenPendingPopAll(ctx context.Context, aicallID uuid.UUID) ([]string, error) {
	res, err := h.Cache.LPopCount(ctx, listenPendingKey(aicallID), listenPendingPopMax).Result()
	if err == redis.Nil {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ListenWindowPush appends one transcript line to the rolling window and trims
// it back to windowSize.
//
// A second list rather than a counter on the first: both operations here are
// single atomic Redis commands, so no cross-command consistency reasoning is
// needed. A line briefly present in the window but not yet popped from pending
// is harmless -- it is context either way.
func (h *handler) ListenWindowPush(ctx context.Context, aicallID uuid.UUID, line string, windowSize int, ttl time.Duration) error {
	key := listenWindowKey(aicallID)

	if err := h.Cache.RPush(ctx, key, line).Err(); err != nil {
		return err
	}

	if err := h.Cache.LTrim(ctx, key, int64(-windowSize), -1).Err(); err != nil {
		return err
	}

	return h.Cache.Expire(ctx, key, ttl).Err()
}

// ListenWindowGet returns the rolling window, oldest line first.
func (h *handler) ListenWindowGet(ctx context.Context, aicallID uuid.UUID) ([]string, error) {
	res, err := h.Cache.LRange(ctx, listenWindowKey(aicallID), 0, -1).Result()
	if err == redis.Nil {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ListenTurnTryLock reports whether this caller may run an evaluation turn now.
//
// SET NX EX: a leaky-bucket debounce, not a mutex. It works across replicas
// (both ai-manager pods share Redis), needs no timers and no per-AIcall
// goroutine, and self-heals on pod loss when the TTL expires. Losing the race
// is the normal case and is not an error -- the line stays buffered for the
// turn that did win.
func (h *handler) ListenTurnTryLock(ctx context.Context, aicallID uuid.UUID, ttl time.Duration) (bool, error) {
	return h.Cache.SetNX(ctx, listenLockKey(aicallID), "1", ttl).Result()
}

// ListenTurnCountIncr increments and returns this AIcall's evaluation-turn
// count. The hard cap it feeds is the backstop against a pathologically long
// call burning LLM spend indefinitely.
func (h *handler) ListenTurnCountIncr(ctx context.Context, aicallID uuid.UUID, ttl time.Duration) (int64, error) {
	key := listenTurnsKey(aicallID)

	res, err := h.Cache.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if errExpire := h.Cache.Expire(ctx, key, ttl).Err(); errExpire != nil {
		return 0, errExpire
	}

	return res, nil
}

// ListenTurnPipecatcallIDAdd registers a pipecatcall id as a genuine listen
// evaluation turn, at the moment that id is minted.
//
// This is the POSITIVE signal ToolHandle needs. The tempting alternative --
// "this id is not the AIcall's currently-bound one, so it must be a listen
// turn" -- is wrong: an agent's own tool call can arrive after Send() has
// best-effort-interrupted its turn and rotated the bound id away, and would be
// indistinguishable from a listen turn. An id that was never SADD'd here is
// provably not a listen turn, whatever the AIcall row happens to say.
//
// Self-expiring: the entry only needs to outlive one turn, so it uses its own
// short TTL and needs no explicit cleanup.
func (h *handler) ListenTurnPipecatcallIDAdd(ctx context.Context, aicallID uuid.UUID, pipecatcallID uuid.UUID, ttl time.Duration) error {
	key := listenTurnPipecatcallIDKey(aicallID)

	if err := h.Cache.SAdd(ctx, key, pipecatcallID.String()).Err(); err != nil {
		return err
	}

	return h.Cache.Expire(ctx, key, ttl).Err()
}

// ListenTurnPipecatcallIDIsMember reports whether the given pipecatcall id was
// registered as a listen evaluation turn for this AIcall.
func (h *handler) ListenTurnPipecatcallIDIsMember(ctx context.Context, aicallID uuid.UUID, pipecatcallID uuid.UUID) (bool, error) {
	return h.Cache.SIsMember(ctx, listenTurnPipecatcallIDKey(aicallID), pipecatcallID.String()).Result()
}

// ListenStartLockAcquire takes the per-AIcall listen-start lock (design §5.2.2).
//
// A thin SET NX EX wrapper -- the same Redis command ListenTurnTryLock above
// already uses -- but with a CALLER-SUPPLIED OWNERSHIP TOKEN rather than a
// constant value, and that difference is the whole point. The debounce lock's
// "anyone may release it" shape is safe only because stealing it merely delays
// a turn. Stealing THIS lock lets two goroutines clobber each other's DB and
// Redis state for a live, billed STT session, so it must be releasable only by
// the goroutine that took it.
//
// This function and ListenStartLockRelease are the lock's only two entry
// points, and they are the only two places the key format exists (design
// review round 16 finding LOW-6). Never call SetNX for this key from a handler.
//
// Returns false, nil when another goroutine already holds it. That is the
// normal, expected outcome of a second panel re-open during one long ring, not
// an error.
func (h *handler) ListenStartLockAcquire(ctx context.Context, aicallID uuid.UUID, token string, ttl time.Duration) (bool, error) {
	return h.Cache.SetNX(ctx, listenStartLockKey(aicallID), token, ttl).Result()
}

// ListenStartLockRelease releases the per-AIcall listen-start lock, but ONLY if
// this caller still holds it.
//
// Compare-and-delete against the caller's own token, atomically, in one EVAL
// (see listenStartLockReleaseScript). A token mismatch means this goroutine's
// TTL already lapsed and someone else legitimately acquired the key since --
// so the call is a deliberate NO-OP, not an error and not a delete. Deleting
// there would take a lock this goroutine no longer holds away from a goroutine
// that does.
//
// ALWAYS CALLED ON A CONTEXT DETACHED FROM THE ACQUIRING GOROUTINE'S OWN ctx
// (design §5.2.2, review round 16 finding MEDIUM-2). Acquire must respect the
// caller's deadline like any other RPC in the trigger path; Release deliberately
// must not, or the one case the TTL-vs-timeout margin exists for -- a goroutine
// reaching its own outer timeout while still finishing legitimate work -- is
// exactly the case where the release silently fails and strands the lock. The
// caller owns that detaching (Task 20), not this function.
func (h *handler) ListenStartLockRelease(ctx context.Context, aicallID uuid.UUID, token string) error {
	return h.Cache.Eval(ctx, listenStartLockReleaseScript, []string{listenStartLockKey(aicallID)}, token).Err()
}

// ListenStateClear removes this AIcall's own per-AIcall listen keys.
//
// It deliberately does NOT touch ai:listen:transcribe:<id> -- that set may be
// shared with another listening AIcall, and only this AIcall's own membership
// may be removed (ListenAIcallIDRemove does that, separately, before this is
// called).
//
// It also deliberately does NOT delete ai:listen:turnpcid:<id>. Those entries
// are short-TTL and self-expiring, and leaving a stale one past a stop causes
// no incorrect behaviour: a tool call arriving late for an already-stopped
// listen turn still correctly resolves as a listen turn, which is exactly what
// it was.
func (h *handler) ListenStateClear(ctx context.Context, aicallID uuid.UUID) error {
	return h.Cache.Del(ctx,
		listenPendingKey(aicallID),
		listenWindowKey(aicallID),
		listenLockKey(aicallID),
		listenTurnsKey(aicallID),
	).Err()
}
```

- [ ] **Step 4: Add the methods to the `CacheHandler` interface**

In `bin-ai-manager/pkg/cachehandler/main.go`, add `"time"` to the imports and append to the `CacheHandler` interface, after the `TeamSet` line:

```go

	// Insight AI realtime call listening (see listen.go).
	ListenAIcallIDsGet(ctx context.Context, transcribeID uuid.UUID) ([]uuid.UUID, error)
	ListenAIcallIDAdd(ctx context.Context, transcribeID uuid.UUID, aicallID uuid.UUID, ttl time.Duration) error
	ListenAIcallIDRemove(ctx context.Context, transcribeID uuid.UUID, aicallID uuid.UUID) error

	ListenPendingPush(ctx context.Context, aicallID uuid.UUID, line string, ttl time.Duration) error
	ListenPendingPopAll(ctx context.Context, aicallID uuid.UUID) ([]string, error)

	ListenWindowPush(ctx context.Context, aicallID uuid.UUID, line string, windowSize int, ttl time.Duration) error
	ListenWindowGet(ctx context.Context, aicallID uuid.UUID) ([]string, error)

	ListenTurnTryLock(ctx context.Context, aicallID uuid.UUID, ttl time.Duration) (bool, error)
	ListenTurnCountIncr(ctx context.Context, aicallID uuid.UUID, ttl time.Duration) (int64, error)

	ListenTurnPipecatcallIDAdd(ctx context.Context, aicallID uuid.UUID, pipecatcallID uuid.UUID, ttl time.Duration) error
	ListenTurnPipecatcallIDIsMember(ctx context.Context, aicallID uuid.UUID, pipecatcallID uuid.UUID) (bool, error)

	// The per-AIcall create-or-reuse lock (design §5.2.2). A matched, symmetric
	// pair -- the key format lives in these two functions and nowhere else.
	ListenStartLockAcquire(ctx context.Context, aicallID uuid.UUID, token string, ttl time.Duration) (bool, error)
	ListenStartLockRelease(ctx context.Context, aicallID uuid.UUID, token string) error

	ListenStateClear(ctx context.Context, aicallID uuid.UUID) error
```

- [ ] **Step 5: Regenerate the mock and run the tests**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go generate ./pkg/cachehandler/... && \
  go test ./pkg/cachehandler/ -v
```
Expected: PASS, and `pkg/cachehandler/mock_main.go` now carries the **fourteen** new methods — twelve buffer/resolver/turn primitives plus the start lock's `Acquire`/`Release` pair. (An earlier revision of this plan said "thirteen" against a twelve-method list; the count is now the actual one.)

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 7: Commit**

Commit message body:

```
- bin-ai-manager: Add listen Redis primitives (resolver set, buffers, debounce lock, turn counter) to cachehandler
- bin-ai-manager: Add the per-AIcall listen-start lock as a matched ListenStartLockAcquire/Release pair with an atomic compare-and-delete release
```

Stage `bin-ai-manager/pkg/cachehandler/`, `bin-ai-manager/go.mod`, `bin-ai-manager/go.sum`, then commit with the branch name as the title and the body above.

---

## Task 12: `bin-ai-manager` — give `aicallHandler` its `cache` dependency

**This is the gap the design assumes away.** Design §5.2.4, §5.3, §5.4 and §5.4.5 all write `h.cache.*` calls inside `aicallHandler`, but the `aicallHandler` struct has **no `cache` field today** and `NewAIcallHandler` takes no cache parameter. Verified 2026-09-04: the struct's fields are `utilHandler`, `reqHandler`, `notifyHandler`, `db`, `aiHandler`, `teamHandler`, `messageHandler`, `participantHandler` — nothing else. Every later listen task depends on this one.

There are **two** call sites of `NewAIcallHandler`, and both must be updated:
- `bin-ai-manager/cmd/ai-manager/main.go` — the real service; a `cache` is already constructed there and passed to `initAIHandler`, so it just needs threading one level further.
- `bin-ai-manager/cmd/ai-control/main.go` — the CLI, which passes `nil` for the four handler dependencies it does not need.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go`
- Modify: `bin-ai-manager/cmd/ai-manager/main.go`
- Modify: `bin-ai-manager/cmd/ai-control/main.go`
- Test: `bin-ai-manager/pkg/aicallhandler/main_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Create or append to `bin-ai-manager/pkg/aicallhandler/main_test.go`:

```go
// Test_NewAIcallHandler_WiresCache pins that the cache dependency reaches the
// struct. Every listen path (the transcribe resolver set, the transcript
// buffers, the debounce lock, the turn counter, and ToolHandle's listen-turn
// membership check) goes through it; a constructor that silently dropped it
// would nil-panic at the first transcript segment, in production, on a code
// path no unit test with an explicitly-constructed handler would ever reach.
func Test_NewAIcallHandler_WiresCache(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := NewAIcallHandler(nil, nil, nil, mockCache, nil, nil, nil, nil)

	concrete, ok := h.(*aicallHandler)
	if !ok {
		t.Fatalf("NewAIcallHandler did not return an *aicallHandler")
	}
	if concrete.cache != mockCache {
		t.Errorf("cache dependency was not wired through the constructor")
	}
}
```

Add the imports the file needs: `"testing"`, the gomock package, and `"monorepo/bin-ai-manager/pkg/cachehandler"`. Confirm the gomock import path matches what the other test files in this package use:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -n "gomock" pkg/aicallhandler/start_test.go | head -2
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_NewAIcallHandler_WiresCache -v
```
Expected: FAIL — too many arguments to `NewAIcallHandler`, and `concrete.cache` undefined.

- [ ] **Step 3: Add the field and the constructor parameter**

In `bin-ai-manager/pkg/aicallhandler/main.go`, add `"monorepo/bin-ai-manager/pkg/cachehandler"` to the imports, add the struct field:

```go
// aicallHandler define
type aicallHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler

	// cache backs the Insight AI's realtime call listening: the transcribe ->
	// AIcall resolver set, the per-AIcall transcript buffers, the cross-replica
	// debounce lock, the turn counter, and ToolHandle's listen-turn membership
	// check. It is NOT a read-through cache for anything in this handler --
	// entity snapshots still go through db (dbhandler owns that). See
	// pkg/cachehandler/listen.go.
	cache cachehandler.CacheHandler

	aiHandler          aihandler.AIHandler
	teamHandler        teamhandler.TeamHandler
	messageHandler     messagehandler.MessageHandler
	participantHandler participanthandler.ParticipantHandler
}
```

and change the constructor:

```go
func NewAIcallHandler(
	req requesthandler.RequestHandler,
	notify notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	cache cachehandler.CacheHandler,
	aiHandler aihandler.AIHandler,
	teamHandler teamhandler.TeamHandler,
	messageHandler messagehandler.MessageHandler,
	participantHandler participanthandler.ParticipantHandler,
) AIcallHandler {
	return &aicallHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,
		cache:         cache,

		aiHandler:          aiHandler,
		teamHandler:        teamHandler,
		messageHandler:     messageHandler,
		participantHandler: participantHandler,
	}
}
```

`cache` goes fourth, right after `db`, because it is infrastructure like the three before it — the handler dependencies stay grouped together after it.

- [ ] **Step 4: Update `cmd/ai-manager/main.go`**

The `cache` is already in scope there (`run(sqlDB *sql.DB, cache cachehandler.CacheHandler)`). Change the construction line to pass it:

```go
	aicallHandler := aicallhandler.NewAIcallHandler(requestHandler, notifyHandler, db, cache, aiHandler, teamHandler, messageHandler, participantHandler)
```

- [ ] **Step 5: Update `cmd/ai-control/main.go`**

That call site passes `nil` for the four handler dependencies. Add one more `nil` in the new fourth position:

```go
	return aicallhandler.NewAIcallHandler(reqHandler, notifyHandler, dbHandler, nil, nil, nil, nil, nil), nil
```

`ai-control` is a CLI that never listens to calls, so a nil cache is correct there — but confirm it, and if it turns out to invoke any listen path, wire the real cache instead (the file already has an `initCache` function):

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -rn "ToolHandle\|EventCMCallHangup\|ProcessTerminate" cmd/ai-control/
```
Expected: no output, or only `ProcessTerminate`. If `ToolHandle` appears, pass the real cache.

- [ ] **Step 6: Fix the existing tests that construct the handler**

Tests in `pkg/aicallhandler` build `&aicallHandler{...}` struct literals directly with named fields, so they compile unchanged. Any that call `NewAIcallHandler` need the extra argument. Find them:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -rn "NewAIcallHandler(" --include='*.go' . | grep -v "/vendor/"
```
Add `nil` (or a `cachehandler.NewMockCacheHandler(mc)`) in the fourth position at each hit outside `main.go`.

- [ ] **Step 7: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_NewAIcallHandler_WiresCache -v && go build ./cmd/...
```
Expected: PASS, and both commands build.

- [ ] **Step 8: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 9: Commit**

Commit message body:

```
- bin-ai-manager: Add cachehandler dependency to aicallHandler and wire it through both cmd entrypoints
```

Stage `bin-ai-manager/pkg/aicallhandler/main.go`, `bin-ai-manager/pkg/aicallhandler/main_test.go`, `bin-ai-manager/cmd/`, then commit with the branch name as the title and the body above.

---

## Task 13: `bin-ai-manager` — accept `?skip_cache=true` on the aicall GET route

Task 4 added the RPC client that sends `GET /v1/aicalls/<uuid>?skip_cache=true`. Nothing serves it yet, and **the existing route regex actively blocks it**: `regV1AIcallsID` is `"/v1/aicalls/" + regUUID + "$"`, anchored at end-of-string, so a URI carrying a query string matches nothing and the request falls through to the switch's default.

`processV1AIcallsIDGet` also parses the id with `strings.Split(m.URI, "/")`, which would hand back `"<uuid>?skip_cache=true"` as element 3 and yield `uuid.Nil`. It must parse the URL properly.

**Do Task 14 before this one** (or commit the two together). This task calls `h.db.AIcallGetSkipCache`, which Task 14 adds.

**Files:**
- Modify: `bin-ai-manager/pkg/listenhandler/main.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go` (the `AIcallHandler` interface)
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go`
- Test: `bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go`, matching the file's existing table/mock shape:

```go
// Test_processV1AIcallsIDGet_SkipCache pins that a GET carrying a query string
// still routes and still parses its id.
//
// Two independent things would break it. (1) regV1AIcallsID is anchored with
// "$", so a query string makes it match nothing and the request silently falls
// through to the router's default -- hence the separate query-tolerant pattern.
// (2) The handler used to split the URI on "/" and take element 3, which for a
// query-bearing URI is "<uuid>?skip_cache=true" and parses to uuid.Nil -- hence
// url.Parse.
func Test_processV1AIcallsIDGet_SkipCache(t *testing.T) {
	aicallID := uuid.FromStringOrNil("3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c")

	tests := []struct {
		name string

		request *sock.Request

		expectSkipCache bool
	}{
		{
			name: "no query string reads through the cache",
			request: &sock.Request{
				URI:    "/v1/aicalls/3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c",
				Method: sock.RequestMethodGet,
			},
			expectSkipCache: false,
		},
		{
			name: "skip_cache=true bypasses the cache",
			request: &sock.Request{
				URI:    "/v1/aicalls/3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c?skip_cache=true",
				Method: sock.RequestMethodGet,
			},
			expectSkipCache: true,
		},
		{
			name: "skip_cache=false reads through the cache",
			request: &sock.Request{
				URI:    "/v1/aicalls/3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c?skip_cache=false",
				Method: sock.RequestMethodGet,
			},
			expectSkipCache: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)
			h := &listenHandler{aicallHandler: mockAIcall}

			if tt.expectSkipCache {
				mockAIcall.EXPECT().GetSkipCache(gomock.Any(), aicallID).Return(&aicall.AIcall{}, nil)
			} else {
				mockAIcall.EXPECT().Get(gomock.Any(), aicallID).Return(&aicall.AIcall{}, nil)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("processRequest returned an unexpected error. err: %v", err)
			}
			if res.StatusCode != 200 {
				t.Errorf("status mismatch. expected: 200, got: %d", res.StatusCode)
			}
		})
	}
}
```

If `listenHandler`'s dispatch entry point is named something other than `processRequest`, use the real name (`grep -n "func (h \*listenHandler) process" pkg/listenhandler/main.go | head -3`). Routing through the dispatcher rather than calling `processV1AIcallsIDGet` directly is deliberate: the regex anchoring bug lives in the dispatcher, so a direct call would pass while production still 404s.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/listenhandler/ -run Test_processV1AIcallsIDGet_SkipCache -v
```
Expected: FAIL — `GetSkipCache` undefined on the mock, and the skip_cache subtest gets a non-200 because nothing routes it.

- [ ] **Step 3: Add the query-tolerant route pattern**

In `bin-ai-manager/pkg/listenhandler/main.go`, add to the aicalls regex block, immediately above `regV1AIcallsID`:

```go
	regV1AIcallsIDQuery             = regexp.MustCompile("/v1/aicalls/" + regUUID + `\?`)
```

and add a dispatch case **before** the existing `regV1AIcallsID` GET case:

```go
	// GET /aicalls/<aicall-id>?<query>
	// Split from the anchored route below because regV1AIcallsID ends in "$",
	// which no query-bearing URI can match. Both land in the same handler; the
	// handler reads the query itself.
	case regV1AIcallsIDQuery.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AIcallsIDGet(ctx, m)
		requestType = "/v1/aicalls/<aicall-id>"
```

- [ ] **Step 4: Rewrite the handler's URI parsing**

In `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`, replace `processV1AIcallsIDGet`'s parsing block and handler call:

```go
// processV1AIcallsIDGet handles GET /v1/aicalls/<aicall-id> request.
//
// The optional skip_cache=true query parameter forces a database-authoritative
// read, bypassing ai-manager's own Redis snapshot cache. Its one caller today is
// messagehandler's stale-reply guard, which must not drop the agent's genuine
// answer because of a transiently-stale cached PipecatcallID. See
// docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.4(b).
func (h *listenHandler) processV1AIcallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsIDGet",
		"request": m,
	})

	// url.Parse, not strings.Split(m.URI, "/"): with a query string attached,
	// splitting on "/" yields "<uuid>?skip_cache=true" as the id element, which
	// parses to uuid.Nil.
	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the uri. err: %v", err)
		return simpleResponse(400), nil
	}

	uriItems := strings.Split(u.Path, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid AIcall ID.")
		return simpleResponse(400), nil
	}

	skipCache := u.Query().Get("skip_cache") == "true"

	var tmp *aicall.AIcall
	if skipCache {
		tmp, err = h.aicallHandler.GetSkipCache(ctx, id)
	} else {
		tmp, err = h.aicallHandler.Get(ctx, id)
	}
	if err != nil {
		log.Errorf("Could not get ai. err: %v", err)
		return errorResponse(err), nil
	}
```

Leave the rest of the function (marshal, response) unchanged. `net/url`, `strings`, `uuid` and `aicall` are all already imported in this file.

- [ ] **Step 5: Add `GetSkipCache` to the `AIcallHandler` interface and implement it**

In `bin-ai-manager/pkg/aicallhandler/main.go`, add to the interface immediately after the `Get(...)` line:

```go
	GetSkipCache(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
```

In `bin-ai-manager/pkg/aicallhandler/db.go`, immediately after `Get`, add a sibling that mirrors it exactly except for the read:

```go
// GetSkipCache returns the aicall read from the database, bypassing the Redis
// snapshot cache. Error handling is identical to Get -- only the read differs.
//
// Use it only where a stale read would cause a wrong, irreversible decision;
// see AIV1AIcallGetSkipCache's own doc comment in bin-common-handler for the
// single such site.
func (h *aicallHandler) GetSkipCache(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "GetSkipCache",
			"aicall_id": id,
		},
	)

	res, err := h.db.AIcallGetSkipCache(ctx, id)
	if err != nil {
		log.Errorf("Could not get aicall info. err: %v", err)
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"AICALL_NOT_FOUND",
				"The AI call was not found.",
			)
		}
		return nil, err
	}

	return res, nil
}
```

Copy the *exact* `cerrors.NotFound(...)` argument list from the existing `Get` — if it differs from the sketch above, the real one wins.

- [ ] **Step 6: Regenerate mocks and run the test**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go generate ./... && go test ./pkg/listenhandler/ -run Test_processV1AIcallsIDGet_SkipCache -v
```
Expected: PASS, all three subtests.

- [ ] **Step 7: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 8: Commit**

Commit message body:

```
- bin-ai-manager: Add a query-tolerant aicall GET route and url.Parse-based id extraction
- bin-ai-manager: Add AIcallHandler.GetSkipCache serving GET /v1/aicalls/<id>?skip_cache=true
```

Stage `bin-ai-manager/pkg/listenhandler/` and `bin-ai-manager/pkg/aicallhandler/`, then commit with the branch name as the title and the body above.

---

## Task 14: `bin-ai-manager` — add the two `dbhandler` variants listening needs

Two narrow additions to `DBHandler`:

1. **`AIcallGetSkipCache`** — the database-authoritative read Task 13's route serves.
2. **`AIcallUpdateNoTouchTMUpdate`** — an update that writes the listen bookkeeping **without** bumping `tm_update`.

The second exists for a real, measured reason. `AIcallUpdate` unconditionally sets `tm_update = now()`, and `Send()`'s cooldown reads `tm_update` to decide whether to reject a message. Listening writes the AIcall row exactly twice per session (once at start, once at stop) — but the stop happens on call hangup, which is precisely when an agent is most likely to ask a follow-up ("what was that about?"). A `Send()` landing inside that window would be rejected by a cooldown it did nothing to deserve. Skipping the bump for listen's own two writes fixes that narrowly, without changing `Send`'s cooldown semantics for anyone else.

**Files:**
- Modify: `bin-ai-manager/pkg/dbhandler/main.go`
- Modify: `bin-ai-manager/pkg/dbhandler/aicall.go`
- Test: `bin-ai-manager/pkg/dbhandler/aicall_test.go`

- [ ] **Step 1: Write the failing test**

Read how the existing `AIcallUpdate` test sets up its harness first — this package's tests may use a live test database:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -n "func Test_AIcallUpdate" -A 40 pkg/dbhandler/aicall_test.go | head -50
```

Then add a test in that same harness's style. Two assertions carry the whole meaning: `listen_call_id` took the new value, and `tm_update` is identical to what it was before the call.

```go
// Test_AIcallUpdateNoTouchTMUpdate pins that this variant leaves tm_update
// alone, unlike AIcallUpdate which always bumps it.
//
// This is not a micro-optimisation. Send()'s cooldown reads tm_update to decide
// whether to reject a message. Listening stops on call hangup -- exactly when an
// agent is most likely to ask the Insight AI a follow-up question -- so a
// tm_update bump from listen's own stop-time bookkeeping would reject a genuine
// question the agent just typed, for no reason the agent could understand.
func Test_AIcallUpdateNoTouchTMUpdate(t *testing.T) {
	// Harness setup identical to Test_AIcallUpdate.
	//
	// 1. Create an aicall.
	// 2. Read back its tm_update.
	// 3. Call AIcallUpdateNoTouchTMUpdate with a listen_call_id change.
	// 4. Read back again.
	// 5. Assert listen_call_id changed AND tm_update did not.
}
```

Fill the body in against the real harness before running it.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/dbhandler/ -run Test_AIcallUpdateNoTouchTMUpdate -v
```
Expected: FAIL — `AIcallUpdateNoTouchTMUpdate` undefined.

- [ ] **Step 3: Add both methods to the `DBHandler` interface**

In `bin-ai-manager/pkg/dbhandler/main.go`, add after the existing `AIcallGet` line:

```go
	AIcallGetSkipCache(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
```

and after `AIcallUpdate`:

```go
	AIcallUpdateNoTouchTMUpdate(ctx context.Context, id uuid.UUID, fields map[aicall.Field]any) error
```

- [ ] **Step 4: Implement them**

In `bin-ai-manager/pkg/dbhandler/aicall.go`, immediately after `AIcallGet`:

```go
// AIcallGetSkipCache gets the aicall straight from the database, ignoring the
// cache entirely.
//
// It still refreshes the cache with what it read -- a caller reaching for this
// has just established that the cached copy was suspect, so leaving the stale
// copy in place would make the next ordinary AIcallGet wrong again.
func (h *handler) AIcallGetSkipCache(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	res, err := h.aicallGetFromDB(id)
	if err != nil {
		return nil, err
	}

	_ = h.aicallSetToCache(ctx, res)

	return res, nil
}
```

and immediately after `AIcallUpdate`:

```go
// AIcallUpdateNoTouchTMUpdate updates the given aicall fields WITHOUT bumping
// tm_update, unlike AIcallUpdate.
//
// This is deliberately narrow, and exists for one caller: the Insight AI's
// realtime-listen bookkeeping (docs/plans/
// 2026-09-03-insight-ai-realtime-listen-design.md §5.2.4, §5.7.3), which writes
// listen_call_id and two metadata keys exactly twice per listening session.
//
// Send()'s cooldown reads tm_update to decide whether to reject a message.
// Listening stops on call hangup -- exactly when an agent is most likely to ask
// the Insight AI a follow-up question -- so a tm_update bump here would reject a
// genuine question the agent just typed. Narrowing the fix to listen's own two
// writes is safer than changing Send's cooldown semantics for every other write
// path in this service.
//
// Do not reach for this for ordinary updates: tm_update is the general
// last-modified signal and skipping it hides real changes from anything that
// reads it.
func (h *handler) AIcallUpdateNoTouchTMUpdate(ctx context.Context, id uuid.UUID, fields map[aicall.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	// Deliberately no updateFields["tm_update"] = h.utilHandler.TimeNow() here.
	// That single omission is this method's entire reason to exist.

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("AIcallUpdateNoTouchTMUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(aicallTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIcallUpdateNoTouchTMUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIcallUpdateNoTouchTMUpdate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, id)

	return nil
}
```

- [ ] **Step 5: Regenerate the mock and run the test**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go generate ./pkg/dbhandler/... && go test ./pkg/dbhandler/ -run Test_AIcallUpdateNoTouchTMUpdate -v
```
Expected: PASS.

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 7: Commit**

Commit message body:

```
- bin-ai-manager: Add dbhandler.AIcallGetSkipCache for database-authoritative aicall reads
- bin-ai-manager: Add dbhandler.AIcallUpdateNoTouchTMUpdate so listen bookkeeping never triggers the Send cooldown
```

Stage `bin-ai-manager/pkg/dbhandler/`, then commit with the branch name as the title and the body above.

---

## Task 15: `bin-ai-manager` — restructure `getPipecatcallMessages` into two fetches

This is a **general context-assembly fix**, not listen-specific machinery. It is active for every `call` / `conversation` / `task` / `contact_case` AIcall the moment the code deploys, regardless of the listen kill switch — there is no listen state to gate it on for AIcalls that were never going to listen.

Today `getPipecatcallMessages` fetches the newest 100 message rows for an AIcall and replays them. The system prompt is message row #1 (written once by `startInitMessages` at AIcall creation and never again), so it competes for that same 100-row window against every subsequent row — and once enough rows accumulate, **the AI's own instructions get evicted from its own context.** Listen turns' tool rows make this reachable much faster, but proactive rows and the agent's own Q&A tool rows would do it eventually anyway.

Two fixes, both needed: exclude `listen_internal` rows at the SQL layer, and fetch the leading system row(s) **independently of the capped window** so they can never be evicted at all.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go`
- Test: `bin-ai-manager/pkg/aicallhandler/start_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bin-ai-manager/pkg/aicallhandler/start_test.go`:

```go
// Test_getPipecatcallMessages_TwoFetch pins the two properties the restructure
// exists for.
//
// (a) The system prompt survives unbounded conversation volume. It is written
// once at AIcall creation and never again, so under the old single capped fetch
// it was simply row number N-100 eventually, and the AI silently lost its own
// instructions. The system rows now come from their own fetch, independent of
// the capped window, so no amount of subsequent traffic can evict them.
//
// (b) Listen-internal rows are excluded at the SQL layer, not filtered in Go
// after the fact. Filtering in Go would not help: the rows would still have
// consumed the 100-row budget before being discarded, which is the whole
// problem.
//
// Both fetches return newest-first (MessageList orders tm_create DESC) and both
// must be reversed before use -- reversing only one silently emits the system
// prompt in reverse order relative to the conversation.
func Test_getPipecatcallMessages_TwoFetch(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockMessage := messagehandler.NewMockMessageHandler(mc)
	h := &aicallHandler{messageHandler: mockMessage}
	ctx := context.Background()

	c := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("7e2c9a10-4b3d-4f8e-9c1a-2b3c4d5e6f70"),
		},
	}

	// Fetch 1: the system rows, newest-first. In production there are at most
	// three (the type-specific system prompt, the substituted init prompt, and
	// an optional parameter-JSON block), all written at creation time.
	mockMessage.EXPECT().List(ctx, uint64(5), "", map[message.Field]any{
		message.FieldAIcallID: c.ID,
		message.FieldRole:     message.RoleSystem,
	}).Return([]*message.Message{
		{Role: message.RoleSystem, Content: "init prompt"},
		{Role: message.RoleSystem, Content: "system prompt"},
	}, nil)

	// Fetch 2: the newest 100 non-system, non-listen-internal rows.
	mockMessage.EXPECT().List(ctx, uint64(100), "", map[message.Field]any{
		message.FieldAIcallID: c.ID,
		message.FieldRole:     commondatabasehandler.NotEq{Value: message.RoleSystem},
		message.FieldOrigin:   commondatabasehandler.NotEq{Value: message.OriginListenInternal},
	}).Return([]*message.Message{
		{Role: message.RoleAssistant, Content: "answer"},
		{Role: message.RoleUser, Content: "question"},
	}, nil)

	res, err := h.getPipecatcallMessages(ctx, c)
	if err != nil {
		t.Fatalf("getPipecatcallMessages returned an unexpected error. err: %v", err)
	}

	expect := []map[string]any{
		{"role": "system", "content": "system prompt"},
		{"role": "system", "content": "init prompt"},
		{"role": "user", "content": "question"},
		{"role": "assistant", "content": "answer"},
	}

	if !reflect.DeepEqual(res, expect) {
		t.Errorf("message assembly mismatch.\nexpected: %v\ngot:      %v", expect, res)
	}
}
```

Add `"reflect"` and `commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"` to the test file's imports if absent, and match the file's existing aliases for `commonidentity` / `message`.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_getPipecatcallMessages_TwoFetch -v
```
Expected: FAIL — the mock sees one `List` call with different arguments.

- [ ] **Step 3: Rewrite the function**

In `bin-ai-manager/pkg/aicallhandler/start.go`, replace `getPipecatcallMessages` entirely:

```go
// getPipecatcallMessages assembles the LLM context replayed into a pipecatcall.
//
// TWO fetches, not one. The reason is a real defect the single-fetch shape had:
// startInitMessages writes the AIcall's system prompt(s) exactly once, at
// creation, so under a single "newest 100 rows" fetch they are simply row
// number N-100 eventually -- and the AI silently loses its own instructions
// partway through a long conversation. Fetching them separately, uncapped,
// makes that structurally impossible.
//
// The second fetch also excludes Origin=listen_internal at the SQL layer. Those
// are the mechanical tool-call/tool-result rows a listen evaluation turn writes;
// they are never useful context, and at up to two rows per turn they would
// consume the capped budget that real Q&A history needs. Excluding them in Go
// after the fact would not help -- they would already have taken the slots.
//
// Both fetches return NEWEST-first (MessageList orders tm_create DESC), so both
// are reversed before use. Reversing only one is a subtle ordering bug that
// still produces plausible-looking output.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.5 step 4.
func (h *aicallHandler) getPipecatcallMessages(ctx context.Context, c *aicall.AIcall) ([]map[string]any, error) {

	// (1) The system row(s), independent of the capped window below. In
	// production there are never more than three (the type-specific system
	// prompt, the substituted init prompt, and an optional parameter-JSON
	// block), all written once by startInitMessages and never again -- so
	// "newest 5" and "all of them" are the same fetch. 5 is headroom, not a
	// truncation risk.
	systemRowsDesc, err := h.messageHandler.List(ctx, 5, "", map[message.Field]any{
		message.FieldAIcallID: c.ID,
		message.FieldRole:     message.RoleSystem,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Could not get system messages")
	}

	// (2) The newest 100 non-system, non-listen-internal rows: Q&A history and
	// proactive notifications, exactly as before, minus the listen-internal
	// exclusion.
	restDesc, err := h.messageHandler.List(ctx, 100, "", map[message.Field]any{
		message.FieldAIcallID: c.ID,
		message.FieldRole:     commondatabasehandler.NotEq{Value: message.RoleSystem},
		message.FieldOrigin:   commondatabasehandler.NotEq{Value: message.OriginListenInternal},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Could not get messages")
	}

	reverseMessages(systemRowsDesc)
	reverseMessages(restDesc)

	ordered := make([]*message.Message, 0, len(systemRowsDesc)+len(restDesc))
	ordered = append(ordered, systemRowsDesc...)
	ordered = append(ordered, restDesc...)

	res := []map[string]any{}
	for _, m := range ordered {
		// skip non-LLM roles (e.g. notification) that would cause API errors
		if m.Role == message.RoleNotification {
			continue
		}

		tmp := map[string]any{
			"role":    string(m.Role),
			"content": string(m.Content),
		}

		if len(m.ToolCalls) > 0 {
			tmp["tool_calls"] = m.ToolCalls
		}

		if len(m.ToolCallID) > 0 {
			tmp["tool_call_id"] = m.ToolCallID
		}

		res = append(res, tmp)
	}

	return res, nil
}

// reverseMessages flips a newest-first slice in place to oldest-first.
func reverseMessages(messages []*message.Message) {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
}
```

Add `commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"` to `start.go`'s imports if it is not already there.

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_getPipecatcallMessages_TwoFetch -v
```
Expected: PASS.

- [ ] **Step 5: Fix every other test that mocks the old single fetch**

Every existing test that exercises `startPipecatcall`, `startPipecatcallTask`, or anything downstream of them now sees **two** `List` calls instead of one. Find and fix them:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ 2>&1 | head -40
```

For each failure, add the second `List` expectation alongside the first, matching the argument maps in Step 3 exactly. This is a large mechanical sweep across `start_test.go` (177KB) — expect many hits, and do not shortcut it with `gomock.Any()` on the filter map: the filter map is the thing this task changes, and pinning it is what stops the exclusion from silently regressing.

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 7: Commit**

Commit message body:

```
- bin-ai-manager: Fetch AIcall system prompt rows independently of the capped replay window
- bin-ai-manager: Exclude listen_internal rows from LLM context replay at the SQL layer
```

Stage `bin-ai-manager/pkg/aicallhandler/start.go` and `bin-ai-manager/pkg/aicallhandler/start_test.go`, then commit with the branch name as the title and the body above.

---

## Task 16: `bin-ai-manager` — register the `notify_agent` tool

`notify_agent` is the **only** way the AI can reach the agent proactively, and it is the one sanctioned write in an otherwise strictly read-only Insight tool set. Registering it touches five places plus the invariant test that exists precisely to make an addition like this loud.

**Files:**
- Modify: `bin-ai-manager/models/tool/main.go`
- Modify: `bin-ai-manager/models/message/tool.go`
- Modify: `bin-ai-manager/pkg/toolhandler/definitions.go`
- Modify: `bin-ai-manager/models/ai/allowed_tools_test.go`
- Modify: `docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md`

- [ ] **Step 1: Watch the invariant test fail first**

Add the tool name to `AllInsightToolNames` **before** touching the test, so the guard fires as designed. In `bin-ai-manager/models/tool/main.go`, add the constant after `ToolNameGetCallTranscript`:

```go
	// Insight AI proactive notification (NOJIRA, docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.5).
	ToolNameNotifyAgent ToolName = "notify_agent"
```

and add it to `AllInsightToolNames`:

```go
	ToolNameGetCallTranscript,
	ToolNameNotifyAgent,
```

**Do NOT add it to `AllToolNames`.** Gating works through the existing, verified mechanism: `ai.AllowedToolNames(TypeInsight)` returns the Insight set, and `bin-pipecat-manager`'s `toolhandler.GetByNames` re-applies it at expansion time. Membership in `AllToolNames` would offer it to every Normal AI too.

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./models/ai/ -run TestAllInsightToolNamesAreReadOnly -v
```
Expected: FAIL — `tool.AllInsightToolNames contains "notify_agent", which is not in this test's known-read-only allowlist`. That failure is the guard working.

- [ ] **Step 2: Relax the invariant narrowly, keeping it loud**

In `bin-ai-manager/models/tool/main.go`, replace the comment above `AllInsightToolNames`:

```go
// AllInsightToolNames defines the tool set available to ai.TypeInsight AIs.
//
// Every entry must be read-only with respect to customer data and external
// systems. The single sanctioned exception is notify_agent, whose only effect is
// to write a message into the AIcall's own conversation thread -- the same
// thread the agent is already reading. It cannot place calls, send email or SMS,
// mutate CRM records, or spend money.
//
// See docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md §2.6
// and docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.5.2.
```

In `bin-ai-manager/models/ai/allowed_tools_test.go`, rewrite `TestAllInsightToolNamesAreReadOnly`'s body so a *different* write tool would still fail:

```go
	knownReadOnly := map[tool.ToolName]bool{
		tool.ToolNameGetContactInteractions: true,
		tool.ToolNameGetConversationContent: true,
		tool.ToolNameGetRelatedCases:        true,
		tool.ToolNameGetCaseNotes:           true,
		tool.ToolNameGetContactProfile:      true,
		tool.ToolNameGetCallTranscript:      true,
	}

	// Sanctioned write exceptions -- a SEPARATE map, deliberately, so this test
	// keeps failing loudly for any other write tool added to
	// AllInsightToolNames. Adding a name here requires the same explicit
	// design-level justification notify_agent got (docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.5.2): its only effect
	// must be on the AIcall's own conversation thread -- no external action, no
	// customer-data mutation, no spend.
	knownSanctionedWrite := map[tool.ToolName]bool{
		tool.ToolNameNotifyAgent: true,
	}

	for _, n := range tool.AllInsightToolNames {
		if !knownReadOnly[n] && !knownSanctionedWrite[n] {
			t.Errorf("tool.AllInsightToolNames contains %q, which is in neither this test's known-read-only allowlist nor its sanctioned-write allowlist -- "+
				"verify what it actually does, then add it to the right map", n)
		}
	}
```

`TestValidateToolNames_WriteToolNeverAllowedForInsight` needs **no** change: it iterates `AllToolNames`, and `notify_agent` is deliberately not a member.

- [ ] **Step 3: Add the `FunctionCallName`**

In `bin-ai-manager/models/message/tool.go`, after `FunctionCallNameGetCallTranscript`:

```go
	FunctionCallNameNotifyAgent            FunctionCallName = "notify_agent"
```

- [ ] **Step 4: Add the tool definition**

Append a new entry to the end of the `toolDefinitions` slice in `bin-ai-manager/pkg/toolhandler/definitions.go`, after the `tool.ToolNameGetCallTranscript` entry:

```go
	{
		Name: tool.ToolNameNotifyAgent,
		// RunLLM:false deliberately: the notification IS the output, there is no
		// follow-up text to generate. This is a best-effort hint to the Python
		// runner, not a guarantee -- every error path there drops the properties
		// entirely, and the model can override it per call with its own run_llm
		// argument. toolHandleNotifyAgent's own reject-if-not-a-listen-turn
		// guard is what actually holds.
		RunLLM: false,
		Description: `Pushes a short, actionable note to the human agent's Insight
Assistant panel, without the agent having asked anything.

WHEN TO USE:
- You are watching a live call transcript and something just happened that the
  agent needs to know right now, per your configured instructions.

WHEN NOT TO USE:
- The agent asked you a question — answer normally instead; do not call this.
- You have nothing new or actionable to say. Saying nothing is the correct and
  expected outcome for most checks.
- You want to repeat something you already notified about on this call.

ARGUMENTS:
- message (required): one or two sentences, written for a busy human mid-call.

This is the only way to reach the agent proactively. It writes into the same
panel thread the agent is already reading; it cannot place calls, send email or
SMS, change CRM records, or spend money.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "The note to show the agent. One or two sentences.",
				},
			},
			"required": []string{"message"},
		},
	},
```

**Note on `RunLLM` in this file:** the six pre-existing Insight tools (`get_contact_interactions`, `get_conversation_content`, `get_related_cases`, `get_case_notes`, `get_contact_profile`, `get_call_transcript`) all use `RunLLM: true`. `notify_agent` being the one Insight tool with `RunLLM: false` is a deliberate outlier. Locate those entries **by tool name, never by line number** — they have shifted across recent refactors.

- [ ] **Step 5: Cross-reference the older design doc so the two do not silently disagree**

In `docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md`, find §2.6 (the "every Insight tool must be read-only" section) and append one line to it:

```markdown
> **Exception, added 2026-09-04:** `notify_agent` is a sanctioned write, scoped
> entirely to the AIcall's own conversation thread. See
> `docs/plans/2026-09-03-insight-ai-realtime-listen-design.md` §5.5.2 for the
> justification and for how `TestAllInsightToolNamesAreReadOnly` still fails
> loudly for any other write tool.
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./models/ai/ ./models/tool/ ./models/message/ ./pkg/toolhandler/ -v
```
Expected: PASS, including `TestValidateToolNames_WriteToolNeverAllowedForInsight` unchanged.

- [ ] **Step 7: Confirm the pipecat-manager gating still holds**

`notify_agent` must be offered to Insight AIs and to nobody else:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-pipecat-manager && \
  go test ./pkg/toolhandler/ -v
```
Expected: PASS.

- [ ] **Step 8: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass. `mapFunctions` has no `notify_agent` entry yet, so a call would return `unknown tool call` — Task 18 adds the handler. That is fine: nothing calls the tool until listening runs, which is still gated off.

- [ ] **Step 9: Commit**

Commit message body:

```
- bin-ai-manager: Register the notify_agent tool in the Insight-only tool set
- bin-ai-manager: Relax the Insight read-only invariant with a separate sanctioned-write allowlist
```

Stage `bin-ai-manager/models/`, `bin-ai-manager/pkg/toolhandler/definitions.go` and `docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md`, then commit with the branch name as the title and the body above.

---

## Task 17: `bin-ai-manager` — resolve `listenTurn` in `ToolHandle` and tag listen-internal rows

Task 5 threaded the invoking pipecatcall id into `ToolHandle` but only logged it. Now it becomes load-bearing.

**Why a Redis membership check and not a comparison against `AIcall.PipecatcallID`.** The tempting predicate — "this id is not the AIcall's currently-bound one, therefore it is a listen turn" — is wrong, and it fails in a way that permanently corrupts data. Concretely: the agent asks Q1 (pipecatcall A is minted and bound); before `ToolHandle` processes Q1's tool call, the agent asks Q2, `Send` best-effort-interrupts A and rotates the binding to B; Q1's tool call now arrives with `pipecatcallID = A`, and `A != B` is true. Indistinguishable from a genuine listen turn. The rows get tagged `listen_internal` and are excluded from replay **forever**.

A set of ids explicitly registered at listen-turn mint time (Task 22) has no such window: A was never registered, whatever the AIcall row happens to say at read time.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/tool.go`
- Modify: `bin-ai-manager/pkg/messagehandler/main.go` (a test helper)
- Create: `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`
- Test: `bin-ai-manager/pkg/aicallhandler/tool_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bin-ai-manager/pkg/aicallhandler/tool_test.go`:

```go
// Test_ToolHandle_listenTurnResolution pins every branch of the listen-turn
// decision. It is worth this much coverage because a wrong answer here is
// permanent: a mistagged row is excluded from LLM replay forever, silently.
func Test_ToolHandle_listenTurnResolution(t *testing.T) {
	aicallID := uuid.FromStringOrNil("0a1b2c3d-4e5f-4a6b-8c9d-0e1f2a3b4c5d")
	turnPCID := uuid.FromStringOrNil("1b2c3d4e-5f6a-4b7c-8d9e-1f2a3b4c5d6e")
	boundPCID := uuid.FromStringOrNil("2c3d4e5f-6a7b-4c8d-9e0f-2a3b4c5d6e7f")

	tests := []struct {
		name string

		referenceType aicall.ReferenceType
		pipecatcallID uuid.UUID

		expectRedisCall  bool
		responseIsMember bool
		responseErr      error

		expectOrigin message.Origin
	}{
		{
			name: "non contact_case never touches redis",
			// Pins the cost gate: without it, every tool call on every AIcall
			// type pays a Redis round trip for a decision that only ever
			// matters for contact_case.
			referenceType:   aicall.ReferenceTypeCall,
			pipecatcallID:   turnPCID,
			expectRedisCall: false,
			expectOrigin:    message.OriginNone,
		},
		{
			name:             "registered turn id is a listen turn",
			referenceType:    aicall.ReferenceTypeContactCase,
			pipecatcallID:    turnPCID,
			expectRedisCall:  true,
			responseIsMember: true,
			expectOrigin:     message.OriginListenInternal,
		},
		{
			name: "unregistered id is a real Q&A turn even when it is not the bound one",
			// The race this design exists to close: a Q&A tool call delayed
			// behind a best-effort pipecatcall rotation arrives with an id that
			// is no longer bound, but was never registered as a listen turn.
			referenceType:    aicall.ReferenceTypeContactCase,
			pipecatcallID:    boundPCID,
			expectRedisCall:  true,
			responseIsMember: false,
			expectOrigin:     message.OriginNone,
		},
		{
			name: "uuid.Nil is treated as a real Q&A turn",
			// Rolling-deploy window: an old pipecat-manager sends no
			// pipecatcall_id. Fail toward doing nothing new, never toward
			// mistagging real content.
			referenceType:    aicall.ReferenceTypeContactCase,
			pipecatcallID:    uuid.Nil,
			expectRedisCall:  true,
			responseIsMember: false,
			expectOrigin:     message.OriginNone,
		},
		{
			name: "a redis error degrades to a real Q&A turn, it does not fail the tool call",
			// During a Redis outage the listen-turn registration and the
			// debounce lock are failing too, so no genuine listen turn can
			// exist -- listenTurn=false is not a guess, it is provably correct.
			// Failing closed here would take ordinary Insight Q&A tool use down
			// with Redis.
			referenceType:   aicall.ReferenceTypeContactCase,
			pipecatcallID:   turnPCID,
			expectRedisCall: true,
			responseErr:     fmt.Errorf("redis unavailable"),
			expectOrigin:    message.OriginNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{db: mockDB, cache: mockCache, messageHandler: mockMessage}
			ctx := context.Background()

			c := &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: aicallID},
				ReferenceType: tt.referenceType,
				PipecatcallID: boundPCID,
			}
			mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(c, nil)

			if tt.expectRedisCall {
				mockCache.EXPECT().
					ListenTurnPipecatcallIDIsMember(ctx, aicallID, tt.pipecatcallID).
					Return(tt.responseIsMember, tt.responseErr)
			}

			// Both the tool-call row and the tool-result row carry the tag.
			mockMessage.EXPECT().
				Create(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					message.DirectionIncoming, message.RoleAssistant, "", gomock.Any(), "",
					gomock.Any(), gomock.Any()).
				DoAndReturn(assertOriginOption(t, tt.expectOrigin))
			mockMessage.EXPECT().
				Create(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					message.DirectionOutgoing, message.RoleTool, gomock.Any(), nil, gomock.Any(),
					gomock.Any(), gomock.Any()).
				DoAndReturn(assertOriginOption(t, tt.expectOrigin))

			_, _ = h.ToolHandle(ctx, aicallID, "tool-1", message.ToolTypeFunction,
				message.FunctionCall{Name: message.FunctionCallNameGetCaseNotes, Arguments: "{}"},
				tt.pipecatcallID)
		})
	}
}

// assertOriginOption returns a MessageHandler.Create stub that applies the
// supplied CreateOptions to a fresh createParams and asserts the resulting
// Origin. Asserting on the applied result rather than on the opaque option
// closure is the only way to see what WithOrigin actually set.
func assertOriginOption(t *testing.T, expect message.Origin) func(
	ctx context.Context, id, customerID, aicallID, activeflowID uuid.UUID,
	direction message.Direction, role message.Role, content string,
	toolCalls []message.ToolCall, toolCallID string, opts ...messagehandler.CreateOption,
) (*message.Message, error) {
	t.Helper()

	return func(_ context.Context, _, _, _, _ uuid.UUID,
		_ message.Direction, _ message.Role, _ string,
		_ []message.ToolCall, _ string, opts ...messagehandler.CreateOption,
	) (*message.Message, error) {
		got := messagehandler.ResolveOriginForTest(opts...)
		if got != expect {
			t.Errorf("Origin mismatch. expected: %q, got: %q", expect, got)
		}
		return &message.Message{Content: "{}"}, nil
	}
}
```

Add the exported test helper it calls to `bin-ai-manager/pkg/messagehandler/main.go`:

```go
// ResolveOriginForTest applies the given options and reports the resulting
// Origin. createParams is unexported, so a test in another package cannot
// otherwise observe what an option actually set.
// USE ONLY FROM TESTS.
func ResolveOriginForTest(opts ...CreateOption) message.Origin {
	p := createParams{}
	for _, opt := range opts {
		opt(&p)
	}
	return p.origin
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_ToolHandle_listenTurnResolution -v
```
Expected: FAIL — no `ListenTurnPipecatcallIDIsMember` call is made and Origin is always `OriginNone`.

- [ ] **Step 3: Resolve `listenTurn` and tag both rows**

In `bin-ai-manager/pkg/aicallhandler/tool.go`, insert the resolution **between** the existing `h.Get` and the first `messageHandler.Create` — before any row for this tool call is written, so an error here never leaves an orphaned row behind:

```go
	// Resolve, exactly once, whether this tool call arrived on a listen
	// evaluation turn. Two consumers share this one answer -- the Origin tag
	// below and toolHandleNotifyAgent's reject-guard -- deliberately, so they
	// can never disagree.
	//
	// The ReferenceType pre-gate is a cached-field comparison, and it is safe to
	// trust the cache for it: ReferenceType is immutable (it is never among
	// AIcallUpdate's written fields). It confines the Redis round trip to the
	// one reference type that can ever be listening, instead of charging every
	// tool call on every AIcall type for it.
	//
	// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.5.
	listenTurn := false
	if c.ReferenceType == aicall.ReferenceTypeContactCase {
		isMember, errMember := h.cache.ListenTurnPipecatcallIDIsMember(ctx, c.ID, pipecatcallID)
		if errMember != nil {
			// Degrade, do not fail closed. A Redis outage means the listen-turn
			// registration and the debounce lock are failing too, so no genuine
			// listen turn can be running at this moment -- every tool call
			// arriving during an outage is structurally a real Q&A call.
			// listenTurn=false is therefore not a guess; it is the provably
			// correct value under this specific failure. Failing the tool call
			// here would take ordinary Insight Q&A tool use down with Redis.
			log.Warnf("Listen-turn membership check failed, assuming a real Q&A turn. err: %v", errMember)
			promListenMembershipCheckFailedTotal.Inc()
		} else {
			listenTurn = isMember
		}
	}

	// Mechanical tool-call/tool-result rows from a listen turn are tagged so
	// getPipecatcallMessages can exclude them from every future replay. A
	// proactive notify_agent OUTPUT row is not tagged -- that is real
	// conversational content the agent sees and the AI should remember.
	rowOrigin := message.OriginNone
	if listenTurn {
		rowOrigin = message.OriginListenInternal
	}
```

Then add `messagehandler.WithOrigin(rowOrigin)` to the first `Create` call:

```go
	tmp, errCreate := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, "",
		messagehandler.WithActiveAIID(toolCallActiveAIID),
		messagehandler.WithOrigin(rowOrigin))
```

- [ ] **Step 4: Thread the tag into the tool-result row**

Change `toolCreateResultMessage`'s signature and its `Create` call in the same file:

```go
func (h *aicallHandler) toolCreateResultMessage(
	ctx context.Context,
	c *aicall.AIcall,
	tool *message.ToolCall,
	tmpContent *messageContent,
	activeAIID uuid.UUID,
	origin message.Origin,
) (*message.Message, error) {

	content, err := json.Marshal(tmpContent)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the tool result content")
	}

	tmp, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(content), nil, tool.ID,
		messagehandler.WithActiveAIID(activeAIID),
		messagehandler.WithOrigin(origin))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the tool message")
	}
	return tmp, nil
}
```

and its call site in `ToolHandle`:

```go
	msg, err := h.toolCreateResultMessage(ctx, c, tool, tmpMessageContent, toolCallActiveAIID, rowOrigin)
```

Check for other callers before changing it:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -rn "toolCreateResultMessage" --include='*.go' pkg/
```
Update every non-test hit; pass `message.OriginNone` at any call site outside `ToolHandle`.

- [ ] **Step 5: Add the metric**

Create `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`:

```go
package aicallhandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics for the Insight AI's realtime call listening. The namespace is
// prepended by the Prometheus client library from metricsNamespace in main.go,
// so Name values here are bare -- writing "ai_manager_" into the name string
// would render as ai_manager_ai_manager_...
var (
	promListenMembershipCheckFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_membership_check_failed_total",
			Help:      "Total number of listen-turn membership checks that errored and degraded to treating the tool call as a real Q&A turn. Near-zero expected; a sustained non-zero rate means Redis is unhealthy, not that anything listen-specific is wrong.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promListenMembershipCheckFailedTotal,
	)
}
```

Confirm the registration idiom matches this package's existing one:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -n "MustRegister" pkg/aicallhandler/*.go | grep -v _test
```
If the package registers metrics somewhere central instead of in a per-file `init()`, follow that instead.

- [ ] **Step 6: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_ToolHandle_listenTurnResolution -v
```
Expected: PASS, all five subtests.

- [ ] **Step 7: Fix the existing `ToolHandle` tests**

Every existing test exercising `ToolHandle` on a `contact_case` AIcall now needs a `ListenTurnPipecatcallIDIsMember` expectation, and every one needs a non-nil `cache` on the handler struct:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ 2>&1 | head -40
```
Work through the failures. Non-`contact_case` tests need no Redis expectation — that is the pre-gate doing its job, and a test that fails because it expected one is telling you the gate broke.

- [ ] **Step 8: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 9: Commit**

Commit message body:

```
- bin-ai-manager: Resolve listen-turn membership once in ToolHandle via a Redis positive check
- bin-ai-manager: Tag listen-turn tool-call and tool-result rows with Origin=listen_internal
```

Stage `bin-ai-manager/pkg/aicallhandler/` and `bin-ai-manager/pkg/messagehandler/main.go`, then commit with the branch name as the title and the body above.

---

## Task 18: `bin-ai-manager` — implement `toolHandleNotifyAgent`

The handler that turns a `notify_agent` tool call into the one proactive message row the agent actually sees.

**It rejects outright when it did not arrive on a listen turn.** If the model calls `notify_agent` during the agent's own Q&A turn and `run_llm=false` takes effect, the agent's real question gets **no answer at all** — just an unrelated notification. That is not harmless; rejecting the tool call is strictly better, because then the agent simply gets their real answer.

`toolHandleNotifyAgent` cannot go in `mapFunctions` unchanged: that map's value type is shared by all 21 handlers and none of the other 20 need `listenTurn`. Dispatch it as a special case instead.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/tool.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/tool_insight.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`
- Test: `bin-ai-manager/pkg/aicallhandler/tool_insight_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bin-ai-manager/pkg/aicallhandler/tool_insight_test.go`:

```go
// Test_toolHandleNotifyAgent covers the whole contract: the happy path writes
// exactly one proactive row, and every rejection path writes none.
func Test_toolHandleNotifyAgent(t *testing.T) {
	aicallID := uuid.FromStringOrNil("4d5e6f7a-8b9c-4d0e-9f1a-2b3c4d5e6f70")
	customerID := uuid.FromStringOrNil("5e6f7a8b-9c0d-4e1f-8a2b-3c4d5e6f7a8b")

	tests := []struct {
		name string

		listenTurn bool
		arguments  string

		expectProactiveRow bool
		expectResult       string
	}{
		{
			name:               "happy path writes one proactive row",
			listenTurn:         true,
			arguments:          `{"message":"Customer just mentioned cancelling."}`,
			expectProactiveRow: true,
			expectResult:       "success",
		},
		{
			name: "rejected outside a listen turn, with no row written",
			// The important case. Allowing it would let RunLLM's best-effort
			// suppression silently eat the agent's real answer.
			listenTurn:         false,
			arguments:          `{"message":"Customer just mentioned cancelling."}`,
			expectProactiveRow: false,
			expectResult:       "failed",
		},
		{
			name:               "empty message rejected",
			listenTurn:         true,
			arguments:          `{"message":""}`,
			expectProactiveRow: false,
			expectResult:       "failed",
		},
		{
			name:               "whitespace-only message rejected",
			listenTurn:         true,
			arguments:          `{"message":"   \n\t  "}`,
			expectProactiveRow: false,
			expectResult:       "failed",
		},
		{
			name:               "oversized message rejected",
			listenTurn:         true,
			arguments:          `{"message":"` + strings.Repeat("x", notifyAgentMaxMessageLen+1) + `"}`,
			expectProactiveRow: false,
			expectResult:       "failed",
		},
		{
			name:               "malformed arguments rejected",
			listenTurn:         true,
			arguments:          `{"message":`,
			expectProactiveRow: false,
			expectResult:       "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &aicallHandler{messageHandler: mockMessage}
			ctx := context.Background()

			c := &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: aicallID, CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeContactCase,
			}

			if tt.expectProactiveRow {
				mockMessage.EXPECT().
					Create(ctx, uuid.Nil, customerID, aicallID, gomock.Any(),
						message.DirectionIncoming, message.RoleAssistant, "Customer just mentioned cancelling.",
						nil, "", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _, _ uuid.UUID,
						_ message.Direction, _ message.Role, _ string,
						_ []message.ToolCall, _ string, opts ...messagehandler.CreateOption,
					) (*message.Message, error) {
						if got := messagehandler.ResolveOriginForTest(opts...); got != message.OriginProactive {
							t.Errorf("Origin mismatch. expected: %q, got: %q", message.OriginProactive, got)
						}
						return &message.Message{}, nil
					})
			}
			// No EXPECT() at all in the rejection cases: gomock fails the test
			// if Create is called anyway, which is exactly the assertion.

			tc := &message.ToolCall{
				ID:       "tool-1",
				Function: message.FunctionCall{Name: message.FunctionCallNameNotifyAgent, Arguments: tt.arguments},
			}

			res := h.toolHandleNotifyAgent(ctx, c, tc, tt.listenTurn)
			if res.Result != tt.expectResult {
				t.Errorf("result mismatch. expected: %q, got: %q (message: %q)", tt.expectResult, res.Result, res.Message)
			}
		})
	}
}
```

Add `"strings"` to the test file's imports if absent.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_toolHandleNotifyAgent -v
```
Expected: FAIL — `h.toolHandleNotifyAgent undefined`.

- [ ] **Step 3: Implement the handler**

Append to `bin-ai-manager/pkg/aicallhandler/tool_insight.go`:

```go
// notifyAgentMaxMessageLen bounds a proactive note's length.
//
// The tool description asks for one or two sentences written for a busy human
// mid-call; this is the backstop for when the model ignores that. It is
// generous on purpose -- the point is to stop a runaway generation from landing
// a wall of text in the agent's panel mid-call, not to police phrasing.
const notifyAgentMaxMessageLen = 500

// parseNotifyAgentMessage extracts and validates the note from a notify_agent
// tool call's arguments.
func parseNotifyAgentMessage(arguments string) (string, error) {
	var args struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", errors.Wrap(err, "invalid arguments")
	}

	msg := strings.TrimSpace(args.Message)
	if msg == "" {
		return "", fmt.Errorf("message is required and must not be empty")
	}

	if len(msg) > notifyAgentMaxMessageLen {
		return "", fmt.Errorf("message is too long: %d characters, maximum %d", len(msg), notifyAgentMaxMessageLen)
	}

	return msg, nil
}

// toolHandleNotifyAgent pushes a proactive note into the agent's Insight
// Assistant panel.
//
// It takes listenTurn as a parameter rather than deriving it: ToolHandle
// resolves it once, from Redis set membership, and shares that one answer with
// the Origin tagging decision. Two independent derivations could disagree.
//
// The row it writes is role=assistant with Origin=proactive, NOT
// role=notification. That distinction is load-bearing: getPipecatcallMessages
// skips RoleNotification when assembling LLM context, so a notification-role row
// would mean that when the agent replies "what did you mean by that?", the AI
// would have no memory of its own notification. It is a genuine assistant
// utterance and is stored as one.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.5, §5.6.
func (h *aicallHandler) toolHandleNotifyAgent(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall, listenTurn bool) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleNotifyAgent",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool notify_agent.")

	res := newToolResult(tc.ID)

	if !listenTurn {
		// This tool fired on the agent's own conversational turn, or the
		// membership check could not run at all (ToolHandle degrades a Redis
		// failure to listenTurn=false, which is provably correct during an
		// outage). Reject rather than let RunLLM's best-effort suppression
		// silently eat the agent's real question: with run_llm=false in effect,
		// a notify_agent call during a Q&A turn means the agent gets an
		// unrelated notification INSTEAD of the answer they asked for.
		fillFailed(res, fmt.Errorf("notify_agent is only usable while proactively monitoring a call; you were asked a question — answer it directly instead"))
		return res
	}

	msg, err := parseNotifyAgentMessage(tc.Function.Arguments)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	tmp, errCreate := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID,
		message.DirectionIncoming, message.RoleAssistant, msg, nil, "",
		messagehandler.WithActiveAIID(h.resolveActiveAIIDFromAIcall(ctx, c)),
		messagehandler.WithOrigin(message.OriginProactive))
	if errCreate != nil {
		log.Errorf("Could not create the proactive message. err: %v", errCreate)
		fillFailed(res, fmt.Errorf("could not deliver the notification"))
		return res
	}
	log.WithField("message", tmp).Debugf("Created the proactive notification message. message_id: %s", tmp.ID)

	promListenNotifyTotal.Inc()

	fillSuccess(res, "message", tmp.ID.String(), "Notification delivered to the agent.")
	return res
}
```

Add `"strings"` and `"monorepo/bin-ai-manager/pkg/messagehandler"` to `tool_insight.go`'s imports if absent.

- [ ] **Step 4: Add the metric**

In `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`, add to the `var` block and to `MustRegister`:

```go
	promListenNotifyTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_notify_total",
			Help:      "Total number of proactive notifications actually delivered to an agent's Insight panel.",
		},
	)
```

- [ ] **Step 5: Dispatch it from `ToolHandle`**

In `bin-ai-manager/pkg/aicallhandler/tool.go`, replace the `mapFunctions` lookup block with a special case ahead of it:

```go
	promAIcallToolExecuteTotal.WithLabelValues(string(tool.Function.Name)).Inc()

	var tmpMessageContent *messageContent
	switch {
	case tool.Function.Name == message.FunctionCallNameNotifyAgent:
		// Dispatched outside mapFunctions deliberately. That map's value type is
		// func(ctx, *aicall.AIcall, *message.ToolCall) *messageContent, shared by
		// all 21 handlers; notify_agent is the only one that needs listenTurn,
		// and widening the shared type for one handler would churn the other 20
		// signatures for a value none of them use.
		tmpMessageContent = h.toolHandleNotifyAgent(ctx, c, tool, listenTurn)

	default:
		fn, exists := mapFunctions[tool.Function.Name]
		if !exists {
			log.Debugf("unknown tool call: %s", tool.Function.Name)
			return nil, fmt.Errorf("unknown tool call: %s", tool.Function.Name)
		}
		tmpMessageContent = fn(ctx, c, tool)
	}
```

- [ ] **Step 6: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_toolHandleNotifyAgent -v
```
Expected: PASS, all six subtests.

- [ ] **Step 7: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 8: Commit**

Commit message body:

```
- bin-ai-manager: Implement toolHandleNotifyAgent writing one role=assistant Origin=proactive row
- bin-ai-manager: Reject notify_agent outright when it did not arrive on a listen evaluation turn
```

Stage `bin-ai-manager/pkg/aicallhandler/`, then commit with the branch name as the title and the body above.

---

## Task 19: `bin-ai-manager` — drop pipecat messages from a foreign pipecatcall

A listen evaluation turn runs on a throwaway pipecatcall id. If its LLM emits ordinary text instead of calling `notify_agent`, that text arrives as a normal pipecat message event and would be persisted and webhook-published as if the AI had said it to the agent. It must be dropped.

**This is a strict improvement beyond this feature, and it is NOT flag-gated.** The `conversation` reference type already has exactly this stale-reply guard; `contact_case` does not, so genuinely stale replies are silently persisted there today. Adding it fixes that too, from the moment the code deploys.

**Only two handlers need it.** `EventPMMessageUserLLM` and `EventPMMessageUserTranscription` are both driven by an STT leg, and a listen turn starts with `STTTypeNone` — so neither can ever originate from one. They are left exactly as they are.

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/main.go`
- Modify: `bin-ai-manager/pkg/messagehandler/event.go`
- Create: `bin-ai-manager/pkg/messagehandler/metrics_foreign.go`
- Test: `bin-ai-manager/pkg/messagehandler/event_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bin-ai-manager/pkg/messagehandler/event_test.go`:

```go
// Test_EventPMMessageBotLLM_ForeignPipecatcall covers the guard's three
// outcomes on the one handler that persists.
//
// The re-read case is the subtle one and the reason the guard is not a simple
// comparison: AIcallUpdate's cache refresh discards its own error, so a
// transient Redis write failure right after a real Send() leaves a stale
// PipecatcallID cached for up to its TTL -- and a cache-only guard would then
// drop the agent's genuine answer. Confirming against the database before
// dropping is what makes the guard safe to turn on for contact_case at all.
func Test_EventPMMessageBotLLM_ForeignPipecatcall(t *testing.T) {
	aicallID := uuid.FromStringOrNil("8a9b0c1d-2e3f-4a5b-8c9d-0e1f2a3b4c5d")
	boundPCID := uuid.FromStringOrNil("9b0c1d2e-3f4a-4b5c-8d9e-1f2a3b3c4d5e")
	foreignPCID := uuid.FromStringOrNil("0c1d2e3f-4a5b-4c6d-9e0f-2a3b4c5d6e7f")

	tests := []struct {
		name string

		eventPipecatcallID  uuid.UUID
		cachedPipecatcallID uuid.UUID

		expectReRead              bool
		expectReReadPipecatcallID uuid.UUID
		expectPersist             bool
	}{
		{
			name:                "matching id persists with no re-read",
			eventPipecatcallID:  boundPCID,
			cachedPipecatcallID: boundPCID,
			expectReRead:        false,
			expectPersist:       true,
		},
		{
			name:                      "mismatch that still disagrees on re-read is dropped",
			eventPipecatcallID:        foreignPCID,
			cachedPipecatcallID:       boundPCID,
			expectReRead:              true,
			expectReReadPipecatcallID: boundPCID,
			expectPersist:             false,
		},
		{
			name:                      "mismatch that agrees on re-read still persists",
			eventPipecatcallID:        foreignPCID,
			cachedPipecatcallID:       boundPCID,
			expectReRead:              true,
			expectReReadPipecatcallID: foreignPCID,
			expectPersist:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &messageHandler{reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify, utilHandler: mockUtil}
			ctx := context.Background()

			evt := &pmmessage.Message{
				ID:                       uuid.FromStringOrNil("1d2e3f4a-5b6c-4d7e-8f90-3a4b5c6d7e8f"),
				Text:                     "some assistant text",
				PipecatcallID:            tt.eventPipecatcallID,
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   aicallID,
			}

			mockReq.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(&aicall.AIcall{
				Identity:      commonidentity.Identity{ID: aicallID},
				ReferenceType: aicall.ReferenceTypeContactCase,
				PipecatcallID: tt.cachedPipecatcallID,
			}, nil)

			if tt.expectReRead {
				mockReq.EXPECT().AIV1AIcallGetSkipCache(ctx, aicallID).Return(&aicall.AIcall{
					Identity:      commonidentity.Identity{ID: aicallID},
					ReferenceType: aicall.ReferenceTypeContactCase,
					PipecatcallID: tt.expectReReadPipecatcallID,
				}, nil)
			}

			if tt.expectPersist {
				mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().MessageGet(ctx, gomock.Any()).Return(&message.Message{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), message.EventTypeMessageCreated, gomock.Any())
			}
			// No EXPECT() in the drop case: gomock fails if anything persists.

			h.EventPMMessageBotLLM(ctx, evt)
		})
	}
}
```

Adapt the mock plumbing to whatever `event_test.go` already uses — `resolveActiveAIIDFromAIcall` may add another expectation on the persist paths.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/messagehandler/ -run Test_EventPMMessageBotLLM_ForeignPipecatcall -v
```
Expected: FAIL — the drop case persists anyway; no `AIV1AIcallGetSkipCache` is ever called.

- [ ] **Step 3: Add the shared helper**

Append to `bin-ai-manager/pkg/messagehandler/main.go`:

```go
// isForeignPipecatcall reports whether an inbound pipecat message event came
// from a pipecatcall session the AIcall does not consider its current
// conversational turn -- a listen evaluation turn, or a genuinely stale reply.
// Such an event must not be persisted or delivered.
//
// It is applied only for aicall.ReferenceTypeContactCase, and only in the two
// handlers a listen turn can actually reach. EventPMMessageUserLLM and
// EventPMMessageUserTranscription are both driven by an STT leg, and a listen
// turn starts with STTTypeNone, so the condition this checks for structurally
// cannot occur on those paths.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.4(b).
func (h *messageHandler) isForeignPipecatcall(ac *aicall.AIcall, evtPipecatcallID uuid.UUID) bool {
	return ac.PipecatcallID != evtPipecatcallID
}
```

- [ ] **Step 4: Apply it in `EventPMMessageBotLLM`**

In `bin-ai-manager/pkg/messagehandler/event.go`, inside the `ac.ReferenceType != aicall.ReferenceTypeConversation` branch (the voice/task/contact_case "persist, no delivery" path), insert before the `Create` call:

```go
		if ac.ReferenceType == aicall.ReferenceTypeContactCase && h.isForeignPipecatcall(ac, evt.PipecatcallID) {
			// The cached AIcall says this event is foreign -- but AIcallUpdate's
			// cache refresh discards its own error, so a transient Redis write
			// failure right after a real Send() leaves a stale PipecatcallID
			// cached, and dropping on that alone would discard the agent's
			// genuine answer. Confirm against the database before dropping.
			fresh, errFresh := h.reqHandler.AIV1AIcallGetSkipCache(ctx, evt.PipecatcallReferenceID)
			if errFresh != nil {
				log.Errorf("Could not re-read the aicall to confirm a foreign pipecatcall — dropping. err: %v", errFresh)
				promForeignPipecatcallDroppedTotal.WithLabelValues("bot_llm").Inc()
				return
			}

			if h.isForeignPipecatcall(fresh, evt.PipecatcallID) {
				log.Infof("Dropping message from a foreign pipecatcall. aicall_id: %s, current_pcc: %s, event_pcc: %s",
					fresh.ID, fresh.PipecatcallID, evt.PipecatcallID)
				promForeignPipecatcallDroppedTotal.WithLabelValues("bot_llm").Inc()
				return
			}

			// The cache was stale; this is a genuine reply. Fall through and
			// persist it, using the authoritative row.
			ac = fresh
		}
```

Then add `WithPipecatcallID(evt.PipecatcallID)` to that branch's `Create` options, so the row records which session produced it:

```go
		tmp, errCreate := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
			message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "",
			WithActiveAIID(activeAIID),
			WithPipecatcallID(evt.PipecatcallID),
			WithInReplyToMessageID(evt.InReplyToMessageID))
```

- [ ] **Step 5: Apply it in `EventPMMessageBotLLMIntermediate`, without a re-read**

That handler fires once per streamed token chunk. Putting an uncached database read on it would charge one per chunk for the entire duration of every listen-turn or stale reply. It only ever publishes a webhook and never persists a row, so a false-positive drop costs one skipped intermediate-token webhook — not user-visible, since only the final `EventPMMessageBotLLM` message matters to the agent.

Insert after the `PipecatcallReferenceType` check:

```go
	// Deliberately NO cache-bypass re-read here, unlike EventPMMessageBotLLM.
	// This fires once per streamed token chunk; an uncached read per chunk would
	// be a real hot-path cost. This handler never persists a row -- it only
	// publishes an intermediate webhook -- so the worst case of a false-positive
	// drop is one skipped intermediate-token webhook, which no user sees.
	if ac, errGet := h.reqHandler.AIV1AIcallGet(ctx, evt.PipecatcallReferenceID); errGet == nil &&
		ac.ReferenceType == aicall.ReferenceTypeContactCase &&
		h.isForeignPipecatcall(ac, evt.PipecatcallID) {
		promForeignPipecatcallDroppedTotal.WithLabelValues("bot_llm_intermediate").Inc()
		return
	}
```

If `resolveActiveAIID` already fetches the AIcall on this path, reuse that value instead of adding a second `AIV1AIcallGet`.

- [ ] **Step 6: Add the metric**

Create `bin-ai-manager/pkg/messagehandler/metrics_foreign.go`:

```go
package messagehandler

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	promForeignPipecatcallDroppedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_foreign_pipecatcall_dropped_total",
			Help:      "Total number of pipecat message events dropped because they came from a pipecatcall the AIcall no longer considers its conversational turn. Covers both Insight listen-turn output and pre-existing stale contact_case replies, which used to be persisted silently.",
		},
		[]string{"handler"},
	)
)

func init() {
	prometheus.MustRegister(
		promForeignPipecatcallDroppedTotal,
	)
}
```

Confirm `metricsNamespace` is reachable from this package:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -rn "metricsNamespace" pkg/messagehandler/
```
If `messagehandler` has no such constant, use `Namespace: "ai_manager"` and add a comment saying it matches `aicallhandler`'s namespace so the metric family stays uniform.

- [ ] **Step 7: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/messagehandler/ -run Test_EventPMMessageBotLLM_ForeignPipecatcall -v
```
Expected: PASS, all three subtests.

- [ ] **Step 8: Confirm the two untouched handlers really are untouched**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -n "isForeignPipecatcall" pkg/messagehandler/event.go
```
Expected: hits only inside `EventPMMessageBotLLM` and `EventPMMessageBotLLMIntermediate`. If `EventPMMessageUserLLM` or `EventPMMessageUserTranscription` appear, remove them — guarding a condition that cannot occur only adds cost.

- [ ] **Step 9: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass. Existing `EventPMMessageBotLLM` tests on `contact_case` AIcalls may now need an `AIV1AIcallGetSkipCache` expectation — add it where the ids genuinely mismatch.

- [ ] **Step 10: Commit**

Commit message body:

```
- bin-ai-manager: Drop contact_case pipecat messages arriving from a foreign pipecatcall
- bin-ai-manager: Confirm against the database before dropping, so a stale cache never discards a genuine reply
```

Stage `bin-ai-manager/pkg/messagehandler/`, then commit with the branch name as the title and the body above.

---

## Task 20: `bin-ai-manager` — `ProcessListen`, `checkListenEligible`, `runListenStart` and the per-AIcall start lock

The trigger. **Rewritten against design rev 23** — rev 15-20 replaced this task's original shape entirely, so read this section as new rather than as a delta.

**There is no `Start` hook any more, and `ensureListen` / `ensureListenAsync` do not exist.** Design rev 1-14 started listening as an implicit side effect of `Start` creating or reusing the Q&A AIcall, which forced this task to solve a real problem — `startReferenceTypeContactCase` has three success returns and only two transition status, so a hook keyed on the transition would essentially never fire on the common panel-re-open path. **Design rev 15 removed that hook entirely, so the problem no longer needs solving here** (design §5.1). Bundling "start listening" into "create/reuse the Q&A AIcall" conflated two independently-callable, independently-observable capabilities. The trigger is now an explicit `POST /service_agents/aicalls/{id}/listen` (public path; Task 27), routed inside ai-manager at `POST /v1/aicalls/{id}/listen` (this task), landing on **one exported method**:

```
ProcessListen(ctx, id)                        -- sole exported entry point, mirrors ProcessTerminate's one-call shape
  ├─ h.Get(ctx, id)                           -- cache-first, same as every other single-AIcall route
  ├─ checkListenEligible(ctx, c)              -- §5.1.1 steps 1-6, SYNCHRONOUS, before the HTTP response
  │    └─ returns (a, kase, callID, call, proceed, err)
  └─ if proceed: go runListenStart(ctx', a, c, kase, callID, call)   -- §5.1.1 steps 7-8, DETACHED
```

**Why one method and not two, and why every resolved value crosses into the goroutine.** Design rev 15's first draft split this into `EnsureListenPrecheck` (steps 1-6) and `ensureListenAsync` (steps 7-8), connected by a bare `bool`. Review round 13 found two defects in that split, both of which this shape exists to avoid (findings HIGH-1/HIGH-2): steps 7-8 need `kase`, `callID` and `call` — a bare bool discards them, forcing the goroutine to **silently re-fetch the Case and the call**, duplicating an RPC pair and re-deriving the tenant boundary that step 4 calls "the tenant boundary for the whole feature"; and an unexported `ensureListenAsync` is unreachable from `pkg/listenhandler` and unmockable on the `AIcallHandler` interface, where every method is exported. So: **`runListenStart` takes every value `checkListenEligible` already resolved, and nothing is re-fetched.** Step 1's test asserts exactly that, by counting `ContactV1CaseGet`/`CallV1CallGet` calls inside the goroutine.

**`ProcessListen` must not block on step 7's confbridge wait.** Steps 1-6 are all cache-first reads or single RPCs; steps 7-8 include an up-to-30s bounded poll. The endpoint returns the AIcall as-is once steps 1-6 pass. Blocking an HTTP request on that wait is a bad pattern on its own merits, and the API deliberately carries no listening-status field to justify it (design §5.1, §11 item 14). Step 1's test pins this as "the single most safety-critical property of the sync/async split."

**The AIcall gate is stricter than it used to be (design §5.1.1 step 2, review round 13 finding MEDIUM-2).** Under the old `Start` hook, the AIcall was by construction one that had just been created or reused as active. A public, arbitrarily-callable endpoint removes that guarantee: any caller can `POST` against any AIcall id it owns, including a terminated or deleted one. So the type gate is now a combined **liveness** gate — `a.Type == ai.TypeInsight && c.Status == aicall.StatusProgressing && c.TMDelete == nil`. Without it a terminated AIcall could pass steps 3-6, spawn the 45s goroutine, and start a **billed** STT session.

**Step 3's idempotency predicate approximates the design, and this plan flags that rather than resolving it silently.** Design §5.1.1 step 3 words the check as comparing the existing transcribe's `ReferenceID` against *"the call we are about to resolve."* That call id is resolved by steps 4-5 (Case lookup, then reference typing) — which the design's own step numbering places **after** step 3. So the requirement, read literally, is a forward reference to a value that does not exist yet at the point the check runs. This is a tension inside the design itself, not something the plan introduced.

Step 3 below therefore compares against `c.ListenCallID`, the call id a *prior* successful listen-start already persisted on this AIcall row. For the common case — repeated panel opens where the Case's call linkage has not changed — the two values are identical and the check is **exact**, not merely close. It could diverge in exactly one scenario: if a Case's associated call somehow changed between two `ProcessListen` calls on the same AIcall, a stale prior session would read as "still listening, skip" where the design's literal wording appears to want a mismatch and a fresh start.

**Do not resolve this by moving the idempotency check after steps 4-5.** Reordering the design's explicit step numbering is a larger divergence than the approximation, and it would put an RPC pair in front of the cheap short-circuit that exists precisely to make the common path free. Per this plan's own convention of surfacing forced implementation choices rather than deciding them alone (see the sync note's "flagged rather than silently decided" list, which carries this as its third item), it is recorded here for whoever confirms whether a Case's reference call is fixed for the Case's lifetime. **If it is, the divergent scenario is provably impossible and this predicate is exact rather than approximate** — and this note can be deleted.

**The confbridge participant-count guard, with bounded retry (design §5.1.1 step 7), is unchanged by the pivot** — it just lives in `runListenStart` now. `Case.ReferenceID` names the customer-facing leg, and `in == Case.Peer` is a code-checked invariant (`case_create`'s own `isCRMEligiblePeer`), but the `out=AGENT` half only holds when the call is in a live, exactly-2-party confbridge — and `Call.ConfbridgeID` is only set once *that leg itself* joins, which for the agent's B-leg happens on answer. `ProcessListen` can run as early as panel-open (a screen-pop UI at ring time is entirely plausible), so this cannot be a one-shot check. **Do not implement a fast-fail on a 3+ party count** — an early version tried that and broke a documented, legitimate flow (`connect` with `early_media: true` and multiple destinations transiently bridges several ringing legs, `bin-call-manager/pkg/confbridgehandler/joined.go:87-97`); the retry below deliberately never distinguishes "still converging" from "stably wrong."

**Everything from the reuse check through the state write is inside a per-AIcall lock (design §5.2.2, rev 17-20).** Step 7's retry means the *same* AIcall can have several concurrent `runListenStart` goroutines in flight from repeated panel re-opens during one long ring — the idempotency check (step 3) cannot short-circuit them, because `listen_transcribe_id` is not set while step 7 is still polling. Combined with the event-ordering fix below (which makes each goroutine mint its **own** speculative transcribe id and pre-write against it), two of them racing can `SREM` each other's *live* session out of the resolver set, or roll back DB/Redis state belonging to the other's live, billed session. **That is not fixable by write-ordering; it needs mutual exclusion.** Three details of the lock are load-bearing and each closes a defect a review round found in the previous attempt:

- **A per-goroutine ownership token, released by atomic compare-and-delete** (round 15 HIGH-1(b)) — never an unconditional `Del`, which could delete a different holder's lock after a TTL lapse.
- **A release context detached from the goroutine's own `ctx`** via `context.WithoutCancel` (round 16 MEDIUM-2) — otherwise the one case the TTL margin exists for (a goroutine hitting its outer timeout while legitimately finishing) is exactly the case where the release fails.
- **A best-effort release on the acquire-*error* path** (round 17 B-7) — a `SET NX` can land server-side while the client sees a timeout or reset, and that path registers no `defer`, so nothing would ever release it.

**And the state write happens *before* the transcribe is created (design §5.2.2/§5.2.4, rev 15/16).** Registering after creation left a window where the new transcribe was already emitting `transcript_created` events for lines nobody had registered to receive — silently dropped as `dropped_unknown`. Pre-registering only the Redis `SADD` (rev 15's narrower fix) then reopened a *worse* race: an early event could resolve through the registered set, fail `RunListenTurn`'s precondition (which requires the **metadata** set, not just Redis membership), and trigger `stopListening`/`clearListenState`, deleting the state the fix had just created. So **both** writes land together, speculatively, against a caller-pre-generated id, with an explicit `rollbackListenState` undo path.

**Files:**
- Create: `bin-ai-manager/pkg/aicallhandler/listen_trigger.go`
- Create: `bin-ai-manager/pkg/aicallhandler/listen_trigger_test.go`
- Create: `bin-ai-manager/pkg/aicallhandler/listen.go` (only the shared predicates and metadata readers in this task; Tasks 21-25 fill in the rest)
- Create: `bin-ai-manager/pkg/aicallhandler/listen_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go` (the `AIcallHandler` interface gains `ProcessListen`)
- Modify: `bin-ai-manager/pkg/listenhandler/main.go` (the route)
- Modify: `bin-ai-manager/pkg/listenhandler/v1_aicalls.go` (the handler)
- Modify: `bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go` (the route's own dispatcher-level test — the file Task 13 already appends to)
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`
- Modify: `bin-ai-manager/docs/architecture.md` (the routing-table entry for the new route — Step 5; Task 29 covers the rest of that file)
- **Not** modified: `bin-ai-manager/pkg/aicallhandler/start.go` — `Start` gains nothing. If you find yourself editing it for this task, re-read design §5.1.

- [ ] **Step 1: Write the failing tests**

Create `bin-ai-manager/pkg/aicallhandler/listen_trigger_test.go` (six tests) and append one more to the existing `bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go`. **Seven tests total**, and none of the coverage lists is optional — each row is a distinct way listening must NOT start, or a distinct way it must. Together they implement design §7 items 1 and 2 in full:

| Test | File | Design coverage |
|---|---|---|
| `Test_checkListenEligible` | `pkg/aicallhandler/listen_trigger_test.go` | §7 item 1's early-return list, plus rev 16's new AIcall-liveness rows |
| `Test_ProcessListen` | `pkg/aicallhandler/listen_trigger_test.go` | §7 item 2's handler-layer list, including the "does not block on the confbridge wait" and "no re-fetch inside the goroutine" assertions |
| `Test_runListenStart_EventOrdering` | `pkg/aicallhandler/listen_trigger_test.go` | §7 item 2's event-ordering block |
| `Test_runListenStart_StartLock` | `pkg/aicallhandler/listen_trigger_test.go` | §7 item 2's lock block |
| `Test_UpdateListenState_OwnsMerge` | `pkg/aicallhandler/listen_trigger_test.go` | §7 item 2's `owns`-merge block |
| `Test_waitForConfbridgeReady` | `pkg/aicallhandler/listen_trigger_test.go` | §7 item 1's step-7 sub-paragraph |
| `Test_processV1AIcallsIDListenPost` | `pkg/listenhandler/v1_aicalls_test.go` | Step 5's new route/dispatcher entry — §7 item 2's "unknown `id` → 404" row at the transport layer |

```go
// Test_checkListenEligible covers every early return of design §5.1.1 steps
// 1-6. Each row is a distinct way listening must not proceed, and every one of
// them must return proceed=false SYNCHRONOUSLY with ZERO goroutines spawned.
func Test_checkListenEligible(t *testing.T) {
	tests := []struct {
		name string

		flagEnabled  bool
		aiType       ai.Type
		aicallStatus aicall.Status

		expectTranscribeGet   bool
		expectCaseGet         bool
		expectCallGet         bool
		expectProceed         bool
	}{
		{name: "flag disabled returns immediately"},
		{name: "non-insight AI returns immediately"},
		// New in design rev 16 (review round 13 finding MEDIUM-2). These two
		// rows are the direct regression test for the public endpoint's loss
		// of the old Start hook's implicit "just created, therefore live"
		// guarantee: they must make ZERO transcribe list/start calls, and must
		// not even reach the idempotency check.
		//
		// "cross-service RPC" is the precise claim, and it is what these rows
		// assert: step 2 necessarily calls the IN-PROCESS aiHandler.Get first,
		// because it cannot check a.Type / c.Status / c.TMDelete without the
		// struct. What must NOT happen is any TranscribeV1*/CallV1*/ContactV1*
		// call -- assert zero of those, not zero calls of any kind.
		{name: "terminated aicall is refused before any cross-service RPC"},
		{name: "soft-deleted aicall (TMDelete set) is refused before any cross-service RPC"},
		{name: "already listening on a valid session makes zero transcribe-start calls"},
		{name: "case reference type is not call"},
		{name: "case reference id does not parse as a uuid"},
		{name: "cross-customer case is refused"},
		{name: "cross-customer call is refused"},
		{name: "call status hangup is not listenable"},
		{name: "call status dialing is listenable and proceeds"},
		{name: "call status ringing is listenable and proceeds"},
		{name: "call status progressing is listenable and proceeds"},
	}
	// Fill in each row's mock expectations and assertions against Step 3.
	_ = tests
}

// Test_ProcessListen covers the sync/async split itself.
//
// This is the handler-layer surface for the whole trigger -- ONE exported
// method, not two (design rev 16, review round 13 findings HIGH-2/MEDIUM-4).
func Test_ProcessListen(t *testing.T) {
	// - unknown id -> the error h.Get returns, and ZERO checkListenEligible
	//   calls.
	//
	// - checkListenEligible returning proceed=false -> 200 with the UNCHANGED
	//   AIcall and ZERO runListenStart invocations. This asserts the method
	//   actually branches on the returned bool rather than always firing the
	//   goroutine.
	//
	// - HAPPY PATH: returns within test-reasonable latency, asserting
	//   ProcessListen does NOT block on step 7's confbridge wait. This is the
	//   single most safety-critical property of the sync/async split
	//   (design §5.1, §7 item 2).
	//
	// - HAPPY PATH, no re-fetch: runListenStart is invoked exactly once and
	//   receives the already-resolved a/c/kase/callID/call values directly --
	//   assert ZERO additional ContactV1CaseGet/CallV1CallGet calls inside the
	//   goroutine. This is the direct regression test for review round 13's
	//   HIGH-1, where a two-function split connected by a bare bool forced the
	//   async stage to silently re-fetch both.
	//
	// - Repeated calls on an already-listening AIcall are free: two
	//   consecutive ProcessListen calls, the second short-circuiting at step
	//   3's idempotency check with zero transcribe-start calls.
	//
	// runListenStart is detached, so assert on a seam (an injected hook or a
	// mock on the handler interface), never on wall-clock timing.
}

// Test_runListenStart_EventOrdering pins design §5.2.2/§5.2.4's ordering fix.
//
// The whole point is that the DB write and the Redis SADD land BEFORE the
// transcribe exists, so the session cannot emit events for a listener nobody
// has registered yet.
func Test_runListenStart_EventOrdering(t *testing.T) {
	// - UpdateListenState's speculative pre-write (DB + Redis, against the
	//   PRE-GENERATED id) happens before TranscribeV1TranscribeStart is
	//   called. Assert call ORDER with gomock.InOrder, not merely that both
	//   eventually happen.
	//
	// - Pre-write failure -> TranscribeV1TranscribeStart is NEVER called (fail
	//   closed; nothing was created, so there is nothing to roll back).
	//
	// - TranscribeV1TranscribeStart failing with a NON-
	//   TRANSCRIBE_ALREADY_PROGRESSING error -> rollbackListenState is called
	//   with the pre-generated id.
	//
	// - TranscribeV1TranscribeStart failing WITH TRANSCRIBE_ALREADY_PROGRESSING
	//   and the re-run list finding a winner -> UpdateListenState is called
	//   again with the WINNER's id and owns=false, ListenAIcallIDRemove is
	//   called for this AIcall's own never-created pre-generated id, and
	//   rollbackListenState is NOT called. This is the direct regression test
	//   for review round 13's MEDIUM-3: an earlier draft rolled back and gave
	//   up here, silently dropping the reuse-on-conflict behaviour the design's
	//   own §6 promises.
	//
	// - Same case but the re-run list also comes up empty -> rollbackListenState,
	//   give up.
	//
	// - An early transcript_created event arriving between the pre-write and
	//   TranscribeV1TranscribeStart returning must NOT trigger
	//   stopListening/clearListenState (the direct regression test for review
	//   round 13's HIGH-3): assert RunListenTurn's precondition reads the
	//   pre-written listen_transcribe_id correctly and processes the segment
	//   normally, not as a skipped_invalid teardown.
	//
	// - Happy path: the transcribe id the mocked TranscribeV1TranscribeStart
	//   reports back equals the pre-generated id already written.
}

// Test_runListenStart_StartLock pins the per-AIcall create-or-reuse lock
// (design §5.2.2, §7 item 2). Every case below is a regression test for a
// specific defect a review round found in an earlier version of this lock.
func Test_runListenStart_StartLock(t *testing.T) {
	// - TWO CONCURRENT runListenStart INVOCATIONS FOR THE SAME AICALL: the
	//   second ListenStartLockAcquire returns acquired=false while the first
	//   still holds ai:listen:startlock:<aicall_id>, and the second goroutine
	//   returns immediately with ZERO TranscribeV1TranscribeStart and ZERO
	//   UpdateListenState calls of its own -- assert the FIRST goroutine's
	//   session survives untouched. Metered skipped_start_locked. This is the
	//   direct regression test for review round 14's HIGH-2 clobbering
	//   scenario.
	//
	// - NORMAL COMPLETION, INCLUDING AFTER ctx IS ALREADY CANCELLED by
	//   AIcallListenEnsureGoroutineTimeoutSeconds: ListenStartLockRelease is
	//   always called and the key is gone immediately after. Assert it by
	//   mocking ListenStartLockRelease to CAPTURE the context it receives and
	//   confirming that context is NOT ctx and is not already Done(). This is
	//   the direct regression test for round 16's MEDIUM-2 -- a release still
	//   keyed off the cancelled ctx would silently no-op.
	//
	// - COMPARE-AND-DELETE SEMANTICS: a release call whose token no longer
	//   matches the key's current value (this goroutine's TTL lapsed and a
	//   different goroutine has since acquired the same key) is a NO-OP, and
	//   the second goroutine's still-live lock is unaffected. This exercises
	//   the exact clobbering the lock exists to prevent, at the release layer
	//   directly rather than only inferring it from the create-path test above
	//   (round 15 HIGH-1(b), re-verified round 16).
	//
	// - SIMULATED CRASH (the defer never runs at all, e.g. the goroutine is
	//   killed mid-sequence rather than merely timing out): the lock is held
	//   for the full AIcallListenStartLockTTLSeconds. A goroutine attempting
	//   Acquire BEFORE that window elapses observes acquired=false
	//   (skipped_start_locked, not an error); one attempting AFTER it elapses
	//   acquires normally and proceeds. Note the "after" half is only true once
	//   the TTL has ACTUALLY elapsed -- the TTL now exceeds a single
	//   goroutine's own outer timeout budget (round 16 MEDIUM-3 corrected an
	//   earlier claim that any later goroutine could just acquire). Use
	//   SetAIcallListenStartLockTTLForTest (Task 10) rather than waiting 60
	//   real seconds.
	//
	// - ACQUIRE-ERROR PATH, INCLUDING ITS BEST-EFFORT RELEASE (round 17 B-7):
	//   ListenStartLockAcquire returning a Redis error -> runListenStart (NOT
	//   checkListenEligible, which never reaches this lock -- round 17 B-6)
	//   fails closed, metered failed, ZERO TranscribeV1TranscribeStart calls,
	//   REGARDLESS of whether the best-effort ListenStartLockRelease attempt on
	//   this path itself succeeds. Assert it by making that release call fail
	//   too and confirming runListenStart still returns the ORIGINAL acquire
	//   error.
	//
	// - DEFERRED-RELEASE-ERROR PATH (round 17 B-6, extending round 16's
	//   MEDIUM-4 coverage): a ListenStartLockRelease error on the normal,
	//   successful-acquire path is swallowed by design -- assert it does NOT
	//   propagate as a runListenStart failure and does NOT get separately
	//   metered, distinguishing it from the acquire-error path above.
}

// Test_UpdateListenState_OwnsMerge pins the SCOPED owns-merge rule
// (design §5.2.4, review round 14 finding HIGH-1).
func Test_UpdateListenState_OwnsMerge(t *testing.T) {
	// - SAME transcribeID as the row's current one, owns=false written after a
	//   prior owns=true -> merged result is TRUE (rev 14's original
	//   same-AIcall race, unchanged).
	//
	// - DIFFERENT transcribeID (the create-then-fall-back-to-reuse branch),
	//   owns=false written after a prior owns=true AGAINST THE OLD ID ->
	//   merged result is FALSE, not carried forward. This is the direct
	//   regression test for HIGH-1: a stale carried-forward owns=true makes
	//   this AIcall believe it owns a session it fell back away from, and
	//   design §5.7.2's stop path would then tear down another Case's
	//   still-live session (in the two-Cases-one-call scenario) -- because
	//   `!owns` evaluates false and the "never touch it" branch is SKIPPED.
	//
	// - DIFFERENT transcribeID -> the OLD id's resolver membership is SREM'd
	//   before the new one is SADD'd.
	//
	// - The merge decision reads a FRESH AIcallGet inside UpdateListenState,
	//   not the caller's in-hand copy (design §5.2.4, review round 15 finding
	//   LOW-7). Assert by giving the mock DB a row whose metadata differs from
	//   whatever the test passes around, and confirming the fresh value wins
	//   for the SREM-old-id half of the rule.
}

// Test_waitForConfbridgeReady covers the bounded retry design §5.1.1 step 7
// added in rev 11-14, and specifically its rev-12 fix for review round 10's
// finding HIGH-A.
//
// It asserts BOTH return values. The second -- the last observed party count --
// is what design §6 requires in the NOT-READY give-up branch's log line (this
// plan also logs it on the error branch, which §6 does not mandate), and it is
// the only thing that distinguishes a stuck-at-1 timeout from a stuck-at-3 one,
// since both deliberately share the skipped_confbridge_not_ready label.
func Test_waitForConfbridgeReady(t *testing.T) {
	tests := []struct {
		name string

		// Sequenced CallV1CallGet / CallV1ConfbridgeGet responses, one pair
		// per poll iteration the test should observe.
		callStatuses      []cmcall.Status
		confbridgeIDs     []uuid.UUID // uuid.Nil means "not yet bridged"
		channelCallCounts []int       // len(ChannelCallIDs) per poll, ignored where confbridgeIDs[i] == uuid.Nil

		expectResult confbridgeReadyResult
		expectPolls  int

		// The LAST OBSERVED party count the function returns alongside the
		// outcome (design §6 requires it in the not-ready branch's log line;
		// this plan also logs it on the error branch).
		// -1 means no confbridge was ever observed -- assert that explicitly on
		// the "ConfbridgeID never resolves" row and the "CallV1ConfbridgeGet
		// errors on the first poll" row, since collapsing it to 0 would make a
		// never-bridged call indistinguishable from an empty bridge in the log.
		// (The "still queued" row is NOT one of them: it starts at uuid.Nil but
		// does resolve, so its expected value is 2.)
		expectLastPartyCount int
	}{
		{
			name:                 "already 2 parties on the first poll -- zero extra polls",
			expectResult:         confbridgeReady,
			expectPolls:          1,
			expectLastPartyCount: 2,
		},
		{
			name: "ringing, then answers: 1 party for several polls, then 2 -- must actually re-poll, not just eventually succeed",
			// This is the direct regression test for review round 9's
			// BLOCKING-1: a one-shot check on this exact sequence would
			// silently never listen.
			expectResult:         confbridgeReady,
			expectPolls:          4,
			expectLastPartyCount: 2,
		},
		{
			name: "confbridge not yet assigned (still queued) -- ConfbridgeID stays uuid.Nil for several polls, then resolves to 2 parties",
			expectResult:         confbridgeReady,
			expectLastPartyCount: 2,
		},
		{
			name: "wait budget exhausted while ConfbridgeID never resolves -- last observed count is -1, not 0",
			// Pins that "never bridged" stays distinguishable from "bridged but
			// empty" in the give-up log line (design §6).
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: -1,
		},
		{
			name: "wait budget exhausted while still stuck at 1 party",
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: 1,
		},
		{
			name: "3 parties while the call is progressing, then settles back to 2 within the wait budget -- proceeds normally, does NOT give up",
			// The direct regression test for review round 10's HIGH-A: an
			// earlier version of this function fast-failed on exactly this
			// sequence (call.Status == progressing plus len >= 3), which
			// broke the early_media multi-destination connect scenario.
			expectResult:         confbridgeReady,
			expectLastPartyCount: 2,
		},
		{
			name: "3 parties for the entire wait budget -- times out the same way a stuck 1-party count would, NOT a distinct outcome",
			// This design deliberately does not have a separate
			// "invalid topology" result -- see the function's own doc
			// comment for why. The last-observed count of 3 in the give-up log
			// line is the ONLY thing that distinguishes this from the stuck-at-1
			// row above, which is exactly why design §6 requires it there.
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: 3,
		},
		{
			name: "call ends mid-poll (hangup during the wait) after a successful 1-party read -- last observed count is 1, not reset",
			// Scenario pinned deliberately: one poll observes a live 1-party
			// bridge, and the NEXT poll's CallV1CallGet comes back hung up. The
			// liveness check runs before the confbridge read, so the count the
			// function carries out is the one the earlier poll saw.
			expectResult:         confbridgeCallEnded,
			expectLastPartyCount: 1,
		},
		{
			name: "CallV1ConfbridgeGet errors on the first poll -- last observed count is -1",
			expectResult:         confbridgeError,
			expectLastPartyCount: -1,
		},
		{
			name: "CallV1ConfbridgeGet errors after a successful 1-party read -- last observed count is 1, not reset",
			expectResult:         confbridgeError,
			expectLastPartyCount: 1,
		},
		{
			name:         "confbridge exists but TMDelete is set or Status is not progressing -- treated as not-ready, not as an error",
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: 2,
		},
	}
	// Fill in each row's sequenced mock expectations against Step 3's
	// implementation. Use a short interval/max-wait override
	// (SetAIcallListenConfbridgeReadyPollIntervalForTest /
	// SetAIcallListenConfbridgeReadyMaxWaitForTest, added in Task 10) so the
	// timeout-driven rows do not make the test suite slow.
	_ = tests
}
```

Then append the seventh test to `bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go` (`package listenhandler`), matching that file's existing table/mock shape — the same file and the same shape Task 13 already appends `Test_processV1AIcallsIDGet_SkipCache` to:

```go
// Test_processV1AIcallsIDListenPost pins Step 5's new route end to end at the
// transport layer: the regex, the dispatcher case, the id parse, and the one
// business-handler call.
//
// IT ROUTES THROUGH processRequest, NOT processV1AIcallsIDListenPost DIRECTLY.
// That is the same deliberate choice Task 13 made, for the same reason: the
// known regex-anchoring behaviour of this dispatcher (regV1AIcallsID ends in
// "$", and Task 13 had to add a separate query-tolerant pattern because of it)
// lives in the dispatcher, so a test that calls the handler function directly
// would pass while production never routes the request at all. If listenHandler's
// dispatch entry point is named something other than processRequest, use the
// real name (grep -n "func (h \*listenHandler) process" pkg/listenhandler/main.go | head -3).
func Test_processV1AIcallsIDListenPost(t *testing.T) {
	// - VALID id -> 200, and aicallHandler.ProcessListen is invoked EXACTLY
	//   ONCE with that parsed uuid. Assert the status code and the mock
	//   expectation; the response body is the marshaled *aicall.AIcall the
	//   mock returns (style (A), no response.* DTO).
	//
	// - UNKNOWN id (ProcessListen returns a cerrors NotFound) -> 404, per
	//   design §7 item 2's "unknown `id` -> 404" row. Assert the status code
	//   comes back as 404 and not as a generic 500, i.e. that the handler
	//   routes the error through errorResponse the way
	//   processV1AIcallsIDTerminatePost does.
	//
	// - UNPARSEABLE id in the URI -> the request must not reach ProcessListen
	//   at all (zero mock calls). Whether that surfaces as the dispatcher's own
	//   no-match default or as the handler's 400, assert the invariant that
	//   matters: no business-handler call is made on a malformed id.
	//
	// - WRONG METHOD (GET on the same path) -> does NOT dispatch to
	//   processV1AIcallsIDListenPost; zero ProcessListen calls. This pins the
	//   `&& m.Method == sock.RequestMethodPost` half of Step 5's switch case.
}
```

Write out every row's mock expectations and assertions concretely against the Step 3 and Step 5 implementations before running them — the sketches above are the coverage contract, not the finished tests.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ ./pkg/listenhandler/ -run 'Test_checkListenEligible|Test_ProcessListen|Test_runListenStart|Test_UpdateListenState_OwnsMerge|Test_waitForConfbridgeReady|Test_processV1AIcallsIDListenPost' -v
```
Expected: FAIL — `h.ProcessListen undefined` in `aicallhandler`, and `ProcessListen` undefined on the `listenhandler` mock (the interface method arrives in Step 4, the route in Step 5).

**Confirm the pattern actually matched something.** `go test -run <pattern>` exits 0 when the pattern matches zero tests, so a typo in a test name reads as a silent pass rather than a failure. The `-v` output must name all seven tests; if any is missing from the run list, fix the name before continuing — and re-check the same pattern in Step 6, which is the verification this task ultimately relies on.

- [ ] **Step 3: Write the trigger implementation**

Create `bin-ai-manager/pkg/aicallhandler/listen_trigger.go`:

```go
package aicallhandler

import (
	"context"
	"errors"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cerrors "monorepo/bin-common-handler/models/errors"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// defaultListenTranscribeStartTimeout is the transcribe-start RPC timeout, in
// milliseconds. Matches summaryhandler's own transcribe start.
const defaultListenTranscribeStartTimeout = 5000

// listenResolverTTL bounds how long a transcribe -> AIcall resolver entry can
// outlive a lost cleanup. Twelve hours comfortably exceeds any real call while
// still guaranteeing the key cannot leak forever.
//
// Deliberately a constant and not a config flag (design §5.2.4): unlike the
// timing flags in internal/config, this bounds a worst-case safety margin --
// how long a genuinely orphaned resolver entry can outlive its transcribe --
// rather than a value anyone is expected to tune.
const listenResolverTTL = 12 * time.Hour

// transcribeReasonAlreadyProgressing is transcribe-manager's rejection reason
// when a session for this (customer_id, reference_id, language) is already
// live. If bin-transcribe-manager ever exports this as a constant, use that
// instead of the literal.
const transcribeReasonAlreadyProgressing = "TRANSCRIBE_ALREADY_PROGRESSING"

// ProcessListen is the sole exported entry point for the listen trigger,
// called once by processV1AIcallsIDListenPost -- the same one-call shape as
// ProcessTerminate (design §5.1, rev 16).
//
// It resolves the AIcall, runs design §5.1.1 steps 1-6 INLINE (synchronously;
// nothing slower than the three RPCs the caller's own longer timeout already
// budgets for), and -- only if every step passes -- spawns a detached
// goroutine for steps 7-8, closing over the already-resolved a/c/kase/callID/
// call values directly. No value crosses a function boundary by itself, so
// there is nothing to re-fetch and nothing to silently lose (review round 13
// finding HIGH-1).
//
// It returns the AIcall UNCHANGED by steps 1-6 themselves; steps 7-8 write
// asynchronously. The response deliberately carries no listening-status field
// (design §5.1, §11 item 14) -- the caller cannot tell "started" from "reused"
// from "not eligible" from "still waiting on the confbridge", and MUST NOT
// block waiting to find out.
func (h *aicallHandler) ProcessListen(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	c, err := h.Get(ctx, id) // cache-first, same as every other single-AIcall route
	if err != nil {
		return nil, err
	}

	a, kase, callID, call, proceed, err := h.checkListenEligible(ctx, c) // §5.1.1 steps 1-6, inline
	if err != nil {
		return nil, err
	}
	if proceed {
		go func() {
			timeout := time.Duration(config.Get().AIcallListenEnsureGoroutineTimeoutSeconds) * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			h.runListenStart(ctx, a, c, kase, callID, call) // §5.1.1 steps 7-8
		}()
	}

	return c, nil
}

// checkListenEligible runs design §5.1.1 steps 1-6 and reports whether the
// detached steps 7-8 may proceed.
//
// THE TENANT BOUNDARY FOR THE WHOLE FEATURE IS HERE. The transcribe session
// runs under a platform system customer id, so its transcript events carry
// that system id, never a tenant id -- an event-time "does this transcript
// belong to this AIcall's customer?" check would ALWAYS fail and is
// impossible. Instead the tenant is checked once, here (customer-scoped
// CaseGet plus a CustomerID recheck on both the Case and the call), and the
// event path verifies PROVENANCE instead: is this transcribe id one we
// ourselves started and recorded? That is a stronger property -- the id is one
// ai-manager generated and persisted, not anything an attacker can influence.
//
// IT RETURNS EVERY VALUE STEPS 7-8 NEED, not a bare bool. a/kase/callID/call
// are all resolved here and handed straight to runListenStart; a bool-only
// boundary would force the goroutine to re-fetch the Case and the call, which
// is exactly the defect review round 13's HIGH-1 found in the first draft of
// this split.
//
// A LOOKUP FAILURE IS NOT AN ERROR RETURN. Design §6's first row is explicit
// that a Case/call/transcribe lookup failure is "logged, metered, listening
// simply does not start" and must never fail the triggering call -- which,
// since rev 15, is the POST itself. So those paths return proceed=false with a
// NIL error and a metered outcome; the error return exists for genuinely
// unexpected conditions only.
//
// CONSEQUENCE, STATED SO NOBODY "SIMPLIFIES" IT AWAY: no branch in the body
// below ever returns a non-nil error. That is intentional, not an oversight.
// The six-value signature matches design §5.1's own snippet shape, and the
// error return is the seam a genuinely unexpected condition would use if one
// is ever added. Dropping it to five values would make this function's
// contract diverge from the design for no gain, and would have to be undone
// the first time a real error case appears.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.1, §5.1.1.
func (h *aicallHandler) checkListenEligible(ctx context.Context, c *aicall.AIcall) (*ai.AI, *kmkase.Kase, uuid.UUID, *cmcall.Call, bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "checkListenEligible",
		"aicall_id": c.ID,
	})

	// Step 1: feature gate.
	//
	// NOT METERED, and that is a gap this plan records rather than papers
	// over (review round 1 finding LOW-8). Design §5.13 enumerates
	// aicall_listen_start_total's `result` values as started / reused /
	// skipped_not_listenable / skipped_confbridge_not_ready /
	// skipped_confbridge_error / failed, and never says which one covers "the
	// feature flag is off." Folding it into skipped_not_listenable is the
	// plausible reading, but the design does not state it and inventing a
	// seventh value here would be exactly the kind of unilateral decision Task
	// 20 flags elsewhere. Leaving it unmetered is also defensible on its own:
	// during a flag-off rollout stage EVERY call takes this branch, so the
	// counter would say nothing the flag's own value does not already say. If
	// a reviewer wants it metered, decide the label in the design first.
	if !config.Get().AIcallListenEnabled {
		return nil, nil, uuid.Nil, nil, false, nil
	}

	// Step 2: AIcall gate -- type AND liveness, combined.
	//
	// The liveness half is NEW with the public endpoint (design rev 16, review
	// round 13 finding MEDIUM-2). Start's old hook only ever ran against an
	// AIcall it had just created or reused as active; an arbitrarily-callable
	// POST removes that guarantee. Without this, a terminated AIcall could
	// pass steps 3-6, spawn the 45s goroutine, and start a BILLED STT session
	// that only RunListenTurn's own unrelated precondition would eventually
	// reap, on the first transcript segment -- later and less directly than
	// catching it here.
	//
	// Deny by default on the type: contact_case AIcalls are Insight in
	// practice, but this does not rely on that.
	a, err := h.aiHandler.Get(ctx, c.AIID)
	if err != nil {
		log.Errorf("Could not get the ai. err: %v", err)
		promListenStartTotal.WithLabelValues("failed").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	if a.Type != ai.TypeInsight || c.Status != aicall.StatusProgressing || c.TMDelete != nil {
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	// Step 3: idempotency. This is what makes repeated panel opens free -- and
	// the panel re-open path is the common one.
	//
	// THE REFERENCE-ID COMPARISON IS AN APPROXIMATION, DELIBERATELY, AND IS
	// FLAGGED RATHER THAN SILENTLY RESOLVED. Design §5.1.1 step 3 words the
	// predicate as comparing the existing transcribe's ReferenceID against
	// "the call we are about to resolve" -- but that call id does not exist
	// yet at step 3: steps 4-5 (Case lookup, reference typing) are what resolve
	// it, and the design's own step numbering puts them AFTER this check. The
	// requirement is therefore a forward reference to a value that, at this
	// point in the design's own ordering, is structurally unavailable.
	//
	// This compares against c.ListenCallID instead -- the call id a PRIOR
	// successful listen-start already persisted on this AIcall row. For the
	// overwhelmingly common case (repeated panel opens where the Case's call
	// linkage has not changed) the two are the same value, so the check is
	// exact, not merely close. It diverges in exactly one scenario: if a Case's
	// associated call somehow changed between two ProcessListen calls on the
	// same AIcall, this would read a stale prior session as "still listening,
	// skip," where §5.1.1 step 3's literal wording appears to want it treated
	// as a mismatch and a fresh start.
	//
	// DO NOT "FIX" THIS BY MOVING THE IDEMPOTENCY CHECK AFTER STEPS 4-5. That
	// reorders the design's explicit step numbering, which is a larger
	// divergence than the approximation itself, and it would put an RPC pair
	// ahead of the cheap short-circuit the common path exists for. Whoever
	// confirms whether a Case's reference call is fixed for the Case's lifetime
	// should settle this: if it is, the divergent scenario is provably
	// impossible and this predicate is exact.
	if existingID := listenTranscribeIDFromMetadata(c); existingID != uuid.Nil {
		if tr, errGet := h.reqHandler.TranscribeV1TranscribeGet(ctx, existingID); errGet == nil &&
			tr.Status == tmtranscribe.StatusProgressing && tr.TMDelete == nil && tr.ReferenceID == c.ListenCallID {
			// Metered as "reused": this AIcall is listening, via a session it
			// or another AIcall already started. Without this increment the
			// design's own stated common path (repeated panel opens) would
			// never appear in aicall_listen_start_total at all (review round 1
			// finding LOW-8).
			promListenStartTotal.WithLabelValues("reused").Inc()
			return nil, nil, uuid.Nil, nil, false, nil
		}
	}

	// Step 4: Case lookup, and the tenant boundary.
	kase, err := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
	if err != nil {
		log.Errorf("Could not get the case. err: %v", err)
		promListenStartTotal.WithLabelValues("failed").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	if kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil {
		// Defensive: the tenant is already embedded in the RPC, but fail closed
		// on any mismatch rather than trust a foreign response shape.
		log.Warnf("Cross-customer case access blocked. case_customer_id: %s", kase.CustomerID)
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	// Step 5: reference typing.
	if kase.ReferenceType != kmkase.ReferenceTypeCall {
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	callID := uuid.FromStringOrNil(kase.ReferenceID)
	if callID == uuid.Nil {
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	// Step 6: call liveness + ownership.
	call, err := h.reqHandler.CallV1CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get the call. err: %v", err)
		promListenStartTotal.WithLabelValues("failed").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	if call.CustomerID != c.CustomerID {
		log.Warnf("Cross-customer call access blocked. call_customer_id: %s", call.CustomerID)
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	if call.TMDelete != nil || !isListenableCallStatus(call.Status) {
		// The call is over. The agent can still read its finished transcript
		// with get_call_transcript, unchanged.
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	return a, kase, callID, call, true, nil
}

// runListenStart runs design §5.1.1 steps 7-8 in its own detached goroutine.
//
// Every argument is already resolved by checkListenEligible and passed
// directly -- NOTHING here re-fetches the Case or the call (review round 13
// finding HIGH-1). `a` and `kase` are unused by the steps below today; they are
// still passed because the design's own signature passes them, and because
// re-deriving either later would silently re-open that same defect.
//
// Fire-and-forget by design: no listening failure may ever fail the POST that
// triggered it, and the POST has already returned by the time this runs.
//
// THE TIMEOUT ITS CALLER GIVES IT IS PURPOSE-BUILT FOR THIS FEATURE, NOT
// INHERITED (design §5.1.1 intro, corrected in review round 11 finding LOW-3).
// It must stay strictly greater than AIcallListenConfbridgeReadyMaxWaitSeconds,
// since waitForConfbridgeReady's bounded retry runs inside this same goroutine
// and needs headroom for the RPC calls each poll makes -- and strictly less
// than AIcallListenStartLockTTLSeconds, so the lock below can never expire
// under a goroutine still working inside its own budget.
func (h *aicallHandler) runListenStart(ctx context.Context, a *ai.AI, c *aicall.AIcall, kase *kmkase.Kase, callID uuid.UUID, call *cmcall.Call) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "runListenStart",
		"aicall_id": c.ID,
		"call_id":   callID,
	})

	// Step 7: the bounded confbridge-readiness retry.
	//
	// lastPartyCount is the LAST OBSERVED len(cb.ChannelCallIDs), or -1 if no
	// confbridge was ever observed. Design §6 requires it in the LOG LINE of
	// the NOT-READY branch specifically (§6's `skipped_confbridge_not_ready`
	// row), explicitly NOT as a metric label -- the count is unbounded-ish and
	// would be cardinality-bearing, but without it in the log there is no way
	// to tell a stuck-at-1 (slow ring) timeout from a stuck-at-3 (genuinely
	// wrong topology) one, since §5.1.1 step 7 deliberately gives both the same
	// label. This plan also logs it on the ERROR branch, for the same
	// diagnostic value, though §6's `skipped_confbridge_error` row does not
	// mandate it there. The third give-up path -- confbridgeCallEnded -- has no
	// log line at all, matching §6's own silence on that outcome.
	//
	// Design §5.1.1 step 7's own prose is what makes this load-bearing: it is
	// what keeps "the false-negative rate this retry exists to bound visible in
	// production rather than merely bounded on paper."
	// Named readyResult, not result: `result` is taken below by step 8's
	// metric label string, which is a different type.
	readyResult, lastPartyCount := h.waitForConfbridgeReady(ctx, callID)
	switch readyResult {
	case confbridgeReady:
		// proceed to step 8
	case confbridgeCallEnded:
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return
	case confbridgeNotReady:
		log.Warnf("The confbridge did not become ready within the wait budget. Listening does not start. last_party_count: %d", lastPartyCount)
		promListenStartTotal.WithLabelValues("skipped_confbridge_not_ready").Inc()
		return
	case confbridgeError:
		log.Warnf("Could not check the confbridge readiness. Listening does not start. last_party_count: %d", lastPartyCount)
		promListenStartTotal.WithLabelValues("skipped_confbridge_error").Inc()
		return
	}

	// Step 8: the locked create-or-reuse sequence.
	result, err := h.startListenTranscribe(ctx, c, call, callID)
	promListenStartTotal.WithLabelValues(result).Inc()
	if err != nil {
		log.Errorf("Could not start listening. result: %s, err: %v", result, err)
		return
	}

	log.Debugf("Listen start finished. result: %s", result)
}

// startListenTranscribe is design §5.2.2's create-or-reuse sequence for one
// AIcall, wrapped in that section's per-AIcall lock.
//
// The returned string is this attempt's aicall_listen_start_total `result`
// label; runListenStart emits it. Every branch below is the design's own
// snippet, kept structurally line-for-line, because each branch is a
// regression fix from a specific review round and reordering or collapsing
// them reopens the corresponding race.
//
// WHY THE LOCK EXISTS (design §5.2.2, review round 14 finding HIGH-2). Design
// §5.1.1 step 7's retry means the SAME AIcall can have several concurrent
// runListenStart goroutines in flight from repeated panel re-opens during one
// long ring -- step 3's idempotency check cannot short-circuit them, because
// listen_transcribe_id is not set while step 7 is still polling. Since the
// event-ordering fix makes each goroutine mint its OWN speculative transcribe
// id and pre-write against it, two of them can both pass the List check below
// before either finishes writing, and then either (a) have the second SREM the
// first's ALREADY-LIVE session out of the resolver set, or (b) have a later
// rollback delete DB/Redis state belonging to the first's live, billed
// session. Neither is fixable by write-ordering; both need mutual exclusion.
//
// SCOPE OF THE LOCK, stated so nobody widens it by accident: it serializes ONE
// AIcall's own create-or-reuse attempts. It does not serialize a different
// AIcall reusing a session this one created (that session was already running
// and emitting before this AIcall ever looked -- a narrower, effectively
// unclosable race shared by every revision of this design), and teardown paths
// (clearListenState, stopListenByCallID) do NOT take this lock and can still
// interleave with it.
func (h *aicallHandler) startListenTranscribe(ctx context.Context, c *aicall.AIcall, call *cmcall.Call, callID uuid.UUID) (string, error) {
	// dupFilters -- bound once, referenced by name from BOTH
	// TranscribeV1TranscribeList calls below. Keyed by the typed
	// tmtranscribe.Field, not a bare string: TranscribeV1TranscribeList's
	// actual parameter is `filters map[tmtranscribe.Field]any`, a distinct
	// named type Go does not implicitly convert to (review round 18 finding
	// MEDIUM-2 -- an earlier draft used map[string]any and would not compile).
	dupFilters := map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:  cmcustomer.IDAIManagerListen,
		tmtranscribe.FieldReferenceID: callID,
		tmtranscribe.FieldStatus:      tmtranscribe.StatusProgressing,
		tmtranscribe.FieldDeleted:     false,
	}

	lockToken := h.utilHandler.UUIDCreate() // this goroutine's own identity for the
	                                        //   lock -- independent of
	                                        //   newTranscribeID, minted below
	lockTTL := time.Duration(config.Get().AIcallListenStartLockTTLSeconds) * time.Second
	releaseTimeout := time.Duration(config.Get().AIcallListenStartLockReleaseTimeoutSeconds) * time.Second

	acquired, err := h.cache.ListenStartLockAcquire(ctx, c.ID, lockToken.String(), lockTTL)
	if err != nil {
		// Ambiguous outcome (review round 17 finding B-7): the SET NX may have
		// landed server-side even though the client saw an error (timeout,
		// connection reset mid-response -- a Redis client cannot always tell
		// "definitely not set" from "set, but the response was lost"). Attempt
		// a best-effort release with our own token so an ambiguous acquire
		// error doesn't strand the lock for the full TTL the same way a
		// genuine crash does. If this second call also fails, the outcome
		// collapses into that same crash case -- accepted, not specially
		// handled further.
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
		_ = h.cache.ListenStartLockRelease(releaseCtx, c.ID, lockToken.String())
		cancel()
		return "failed", err // fail closed, same as every other §5.2 RPC failure --
		                     //   no transcribe list/start call has been made yet
	}
	if !acquired {
		// Another goroutine for this exact AIcall is already inside this
		// sequence (§5.1.1 step 7's own retry, or a second panel-open during
		// the same ring). Let it finish -- this goroutine's job is now
		// redundant, and racing it is exactly the race above.
		return "skipped_start_locked", nil
	}
	defer func() {
		// Detached from ctx's own cancellation/deadline (review round 16
		// finding MEDIUM-2) so a goroutine that reaches its own outer timeout
		// still releases promptly instead of stranding the lock for the full
		// TTL -- combined with the best-effort release on the acquire-error
		// path above, stranding for the full TTL is now reserved for an actual
		// crash (pod loss, process kill -- anywhere this defer itself never
		// runs), not merely an error return from either call.
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
		defer cancel()
		_ = h.cache.ListenStartLockRelease(releaseCtx, c.ID, lockToken.String()) // compare-and-delete, best-effort
	}()

	// REUSE IS LANGUAGE-TOLERANT ON PURPOSE. Any progressing
	// IDAIManagerListen session on this call is reused regardless of its
	// language string -- starting a second session only because a language
	// string differs would double the STT cost on one call to gain nothing,
	// since the LLM reads whatever language comes out.
	//
	// A session ai-manager does NOT own -- one the customer started under
	// their own customer_id, or an AI summary's under IDAIManager -- is never
	// reused and never touched. The owner scoping makes that structural, not a
	// convention: this list simply cannot see them.
	existing, errList := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 10, dupFilters)
	if errList != nil {
		return "failed", errList // fail closed -- an unhandled error here previously
		                         //   read as "no existing session found" (review round
		                         //   15 finding LOW-4) and could have started a
		                         //   duplicate session
	}
	if len(existing) > 0 {
		if _, errUpdate := h.UpdateListenState(ctx, c.ID, callID, existing[0].ID, false); errUpdate != nil {
			return "failed", errUpdate
		}
		return "reused", nil
	}

	language := c.STTLanguage
	if language == "" {
		language = config.Get().AIcallListenDefaultLanguage
	}

	// THE STATE WRITE COMES FIRST, SPECULATIVELY, AGAINST AN ID WE GENERATE
	// (design §5.2.2/§5.2.4, review round 13 finding HIGH-3). Both the DB
	// write and the Redis SADD land before the transcribe exists, so the
	// session cannot emit a single event for a listener nobody has registered.
	// Do not "simplify" this back to writing after creation.
	newTranscribeID := h.utilHandler.UUIDCreate()
	if _, err := h.UpdateListenState(ctx, c.ID, callID, newTranscribeID, true); err != nil {
		return "failed", err // fail closed: no transcribe created yet, nothing to roll back
	}

	_, err = h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		newTranscribeID,              // id -- caller-specified, not uuid.Nil; this
		                              //   ordering fix is the one and only reason
		                              //   this design uses that capability
		cmcustomer.IDAIManagerListen, // customerID: the platform sentinel, never the tenant
		call.ActiveflowID,            // the CALL's activeflow, not the AIcall's -- a
		                              //   panel-started contact_case AIcall has
		                              //   ActiveflowID == uuid.Nil
		uuid.Nil,                     // onEndFlowID: no on-end flow for listening
		tmtranscribe.ReferenceTypeCall,
		callID,
		language,
		tmtranscribe.DirectionBoth, // both legs; the speaker tag comes from each segment's own direction
		tmtranscribe.ProviderEmpty, // default provider order
		defaultListenTranscribeStartTimeout,
	)
	switch {
	case err == nil:
		// The created transcribe's id equals newTranscribeID (caller-specified,
		// above), and the DB/Redis state written above already matches it, so
		// there is nothing further to write on this path -- and nothing to
		// capture from the return value.
		return "started", nil

	case isAlreadyProgressing(err):
		// The read-then-create race design §6 already documents: this AIcall's
		// own List above ran just before another writer (a DIFFERENT AIcall on
		// the same call -- the lock only serializes writers sharing this same
		// AIcall) won the create. Re-run the list once and, if a winner is
		// found, rewrite our state to point at it instead of giving up. A
		// blanket rollback-and-fail here would silently drop the
		// reuse-on-conflict behaviour §6 promises (review round 13 finding
		// MEDIUM-3).
		existingRetry, errListRetry := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 10, dupFilters)
		if errListRetry != nil || len(existingRetry) == 0 {
			_ = h.rollbackListenState(ctx, c.ID, newTranscribeID) // no winner found either; give up
			return "failed", err
		}
		if _, errUpdate := h.UpdateListenState(ctx, c.ID, callID, existingRetry[0].ID, false); errUpdate != nil {
			_ = h.rollbackListenState(ctx, c.ID, newTranscribeID)
			return "failed", errUpdate
		}
		// Our own speculative id never got created -- remove only THAT
		// membership, never the winner's (UpdateListenState above already
		// registered us against the winner correctly).
		//
		// This is design §5.2.2's ListenTranscribeAIcallRemove; see Task 11 for
		// why this plan calls it ListenAIcallIDRemove.
		_ = h.cache.ListenAIcallIDRemove(ctx, newTranscribeID, c.ID)
		return "reused", nil

	default:
		// Any other TranscribeV1TranscribeStart failure: give up, undo the
		// speculative write.
		_ = h.rollbackListenState(ctx, c.ID, newTranscribeID)
		return "failed", err
	}
}

// isAlreadyProgressing reports whether err is transcribe-manager's
// "a session for this reference is already live" rejection.
//
// This is the established pattern in this monorepo -- errors.As against
// *cerrors.VoipbinError plus a Reason comparison
// (bin-common-handler/models/errors/voipbin_error.go, used at
// bin-transcribe-manager/pkg/transcribehandler/stop.go and, in exactly this
// one-line-wrapper shape, at bin-storage-manager/pkg/filehandler/signing.go).
// There is no cerrors.IsReason helper in this codebase; an earlier design
// draft invented one (review round 14 finding MEDIUM-1). Do not add one to
// bin-common-handler for this -- it is a local wrapper, named for readability
// in the switch above.
func isAlreadyProgressing(err error) bool {
	var ve *cerrors.VoipbinError
	return errors.As(err, &ve) && ve.Reason == transcribeReasonAlreadyProgressing
}

// rollbackListenState undoes the speculative pre-write for a transcribe that
// was never actually created (design §5.2.2).
//
// A small, dedicated helper -- NOT a reuse of clearListenState (Task 25),
// whose contract reads listen_transcribe_id off an AIcall struct "already in
// hand," an assumption that does not hold here since UpdateListenState writes
// through the DB rather than mutating the caller's in-memory c. This one takes
// the known transcribeID directly, so it can only ever remove the membership
// it was told about.
func (h *aicallHandler) rollbackListenState(ctx context.Context, aicallID uuid.UUID, transcribeID uuid.UUID) error {
	if errRem := h.cache.ListenAIcallIDRemove(ctx, transcribeID, aicallID); errRem != nil {
		logrus.Warnf("Could not remove the speculative listen resolver membership. aicall_id: %s, transcribe_id: %s, err: %v", aicallID, transcribeID, errRem)
	}

	// Read fresh, then drop only the two listen keys. FieldMetadata is a
	// whole-column write, so building the map from scratch here would silently
	// destroy every other metadata key on the row (prompt snapshots, the
	// auto-audit flag). Same reasoning as UpdateListenState's own copy loop
	// below -- including its choice of the cache-first AIcallGet over
	// AIcallGetSkipCache, for the reason spelled out in that function's doc
	// comment.
	cur, err := h.db.AIcallGet(ctx, aicallID)
	if err != nil {
		return err
	}

	metadata := map[string]any{}
	for k, v := range cur.Metadata {
		metadata[k] = v
	}
	delete(metadata, aicall.MetaKeyListenTranscribeID)
	delete(metadata, aicall.MetaKeyListenOwnsTranscribe)

	// Same targeted, tm_update-bypassing write UpdateListenState uses -- a
	// rollback must not bump tm_update either, or it feeds Send()'s cooldown
	// for a session that never even started.
	return h.db.AIcallUpdateNoTouchTMUpdate(ctx, aicallID, map[aicall.Field]any{
		aicall.FieldListenCallID: uuid.Nil,
		aicall.FieldMetadata:     metadata,
	})
}

// confbridgeReadyResult is why waitForConfbridgeReady stopped polling.
type confbridgeReadyResult int

const (
	confbridgeReady confbridgeReadyResult = iota
	// confbridgeNotReady: the wait budget elapsed without the call ever
	// settling into a live, exactly-2-party confbridge. Covers a stuck
	// 1-party count (still ringing) and a stuck 3+-party count (a genuinely
	// non-standard topology) identically -- see the function's own doc
	// comment for why this design does not try to tell them apart.
	confbridgeNotReady
	// confbridgeCallEnded: the call itself stopped being listenable during
	// the wait (step 6's own check, re-run every poll).
	confbridgeCallEnded
	// confbridgeError: a CallV1CallGet or CallV1ConfbridgeGet RPC failed.
	confbridgeError
)

// waitForConfbridgeReady polls, bounded, until the given call is live inside a
// confbridge with exactly 2 parties -- the shape speakerTag's in=CUSTOMER/
// out=AGENT mapping assumes.
//
// WHY THIS POLLS INSTEAD OF CHECKING ONCE (design §5.1.1 step 7; review round
// 9 finding BLOCKING-1). Call.ConfbridgeID and the confbridge's own
// ChannelCallIDs are only updated once THIS leg's own join channel enters the
// bridge. For the A-leg that happens at queue-forward time, well before the
// agent (the B-leg) answers -- so the confbridge reads as exactly 1 party for
// the whole queue-wait-plus-ring window. ProcessListen can run as early as
// panel-open (a screen-pop UI opening the Case panel at ring time is entirely
// plausible), so a one-shot check would silently never start listening on a
// perfectly ordinary call, with nothing recorded to explain why.
//
// WHY THIS NEVER FAST-FAILS ON A NON-2 PARTY COUNT (review round 10 finding
// HIGH-A). An earlier version of this function gave up immediately once
// call.Status was progressing and the party count was >= 3, reasoning that a
// progressing call was past the point where an extra party could still be
// transient pre-answer noise. That reasoning is unsound: the LISTENED leg
// (the call this function is checking) is progressing for this entire wait --
// queue-wait through agent-ring -- so the fast-fail condition was true the
// instant ANY 3rd party appeared, not just once one had lingered. A
// documented, legitimate flow hits exactly this: a connect action with
// early_media=true and multiple destinations bridges every ringing
// destination before the losing ones hang up
// (bin-call-manager/pkg/confbridgehandler/joined.go:87-97 explicitly
// iterates looking for a ringing/dialing member -- i.e. this state is
// anticipated, not a bug). The earlier fast-fail would have permanently
// killed listening on such a call, on possibly the only ProcessListen
// invocation it ever gets. So: any non-2 count, at any call status, just
// keeps polling until the wait budget runs out. This means a stably-wrong
// topology and a merely-slow ring share one outcome
// (confbridgeNotReady/skipped_confbridge_not_ready) and one budget --
// accepted, and documented in design §11 item 13 as a reason to err toward a
// longer AIcallListenConfbridgeReadyMaxWaitSeconds default rather than a
// shorter one.
//
// NOTE ON CONCURRENT CALLERS: repeated panel re-opens during one long ring
// spawn multiple concurrent, independently-bounded calls to this function for
// the SAME AIcall (design §5.1.1 step 7's closing note). Step 3's idempotency
// check cannot short-circuit them, because listen_transcribe_id is not set
// while this function is still polling. That is expected, not a bug, and it is
// bounded -- but it is NOT harmless on its own: what makes it safe is
// startListenTranscribe's per-AIcall lock, which serializes the create-or-reuse
// sequence those goroutines then race into (design §5.2.2). Do not weaken that
// lock on the theory that transcribe-manager's own cross-AIcall dedup guard
// already covers this; review round 14 (HIGH-2) found that reasoning covers
// only cross-AIcall duplication, not this same-AIcall race.
//
// One consequence worth knowing when reading the metric:
// skipped_confbridge_not_ready's raw rate can be inflated by repeated re-opens
// of the same still-ringing call, not just by distinct calls.
//
// IT TAKES ONLY callID, NOT THE AICALL'S CUSTOMER ID, AND THAT IS DELIBERATE.
// Design §5.1.1 step 6 pairs a liveness check with an ownership check
// (call.CustomerID == c.CustomerID), but only the LIVENESS half needs
// re-checking on each poll: a live call's CustomerID is immutable, so
// checkListenEligible's single step-6 ownership check still holds for the whole
// wait. Re-checking it every poll would cost nothing and prove nothing.
//
// IT RETURNS THE LAST OBSERVED PARTY COUNT alongside the outcome (design §6).
// -1 means no confbridge was ever observed (ConfbridgeID stayed uuid.Nil, or
// the very first CallV1CallGet failed), which is a materially different
// diagnosis from "observed, but stuck at 1" and must not be collapsed into 0.
// The caller logs it; it is deliberately NOT a metric label.
func (h *aicallHandler) waitForConfbridgeReady(ctx context.Context, callID uuid.UUID) (confbridgeReadyResult, int) {
	interval := time.Duration(config.Get().AIcallListenConfbridgeReadyPollIntervalSeconds) * time.Second
	deadline := time.Now().Add(time.Duration(config.Get().AIcallListenConfbridgeReadyMaxWaitSeconds) * time.Second)

	lastPartyCount := -1 // -1: no confbridge observed yet, distinct from an observed 0

	for {
		call, err := h.reqHandler.CallV1CallGet(ctx, callID)
		if err != nil {
			return confbridgeError, lastPartyCount
		}
		if call.TMDelete != nil || !isListenableCallStatus(call.Status) {
			return confbridgeCallEnded, lastPartyCount
		}

		if call.ConfbridgeID != uuid.Nil {
			cb, errCb := h.reqHandler.CallV1ConfbridgeGet(ctx, call.ConfbridgeID)
			if errCb != nil {
				return confbridgeError, lastPartyCount
			}
			lastPartyCount = len(cb.ChannelCallIDs)
			if cb.TMDelete == nil && cb.Status == cmconfbridge.StatusProgressing && lastPartyCount == 2 {
				return confbridgeReady, lastPartyCount
			}
			// Not yet a live 2-party bridge: ConfbridgeID unset (not yet
			// joined), a live 1-party bridge (still ringing the other leg),
			// or a transient 3+-party state that has not yet settled (see the
			// HIGH-A note above). All three fall through to the same "keep
			// polling" behaviour below.
		}

		if time.Now().After(deadline) {
			return confbridgeNotReady, lastPartyCount
		}

		select {
		case <-ctx.Done():
			// CATEGORY MISMATCH, CURRENTLY UNREACHABLE -- REVISIT IF THE
			// DEFAULTS CHANGE. This branch fires when the goroutine's OWN outer
			// timeout (AIcallListenEnsureGoroutineTimeoutSeconds) expires, which
			// is not an RPC failure -- and design §5.13 defines
			// confbridgeError/skipped_confbridge_error as the RPC-failure
			// outcome specifically. At the shipped defaults it cannot happen:
			// the poll budget (AIcallListenConfbridgeReadyMaxWaitSeconds, 30s)
			// is strictly less than the goroutine timeout (45s), so the deadline
			// check above always wins and returns confbridgeNotReady first.
			// Task 10 pins that ordering as a standing invariant. If either
			// default ever moves such that the poll budget can outlast the
			// goroutine, this becomes reachable and must be given its own
			// outcome rather than mislabelled as an RPC error.
			return confbridgeError, lastPartyCount
		case <-time.After(interval):
		}
	}
}

// UpdateListenState persists that this AIcall is now listening: one AIcall row
// write plus one Redis set membership.
//
// It takes the AIcall ID, NOT the caller's *aicall.AIcall (design §5.2.4,
// review round 15 finding LOW-7). Both merge rules below turn on "the row's
// CURRENT value," and the calling goroutine's own in-hand copy can be stale --
// it is never mutated by this write, and a concurrent goroutine for the same
// AIcall may have written since. So the current values come from a fresh read
// here, immediately before the merge decision.
//
// THE FRESH READ IS THE ORDINARY CACHE-FIRST h.db.AIcallGet, NOT
// AIcallGetSkipCache. Design §5.2.4's contrast is between this function's own
// read and the CALLER's stale in-hand struct -- not between the cache and the
// database. Nothing in this feature writes this AIcall's cache entry from a
// concurrent path that could race within one request: the only writers are
// UpdateListenState and rollbackListenState themselves, both of which go
// through AIcallUpdateNoTouchTMUpdate and both of which are serialized per
// AIcall by startListenTranscribe's own lock. AIcallGetSkipCache does exist
// (Task 14), and it is deliberately not used here -- its one justified caller
// is messagehandler's stale-reply guard, where a stale PipecatcallID would
// cause a wrong, irreversible decision.
//
// This is the ONLY ai_aicalls write the feature makes during a listening session
// (one at start, one at stop) -- never per turn. And it goes through
// AIcallUpdateNoTouchTMUpdate specifically, so listen's own bookkeeping never
// bumps tm_update and therefore never feeds Send()'s cooldown.
//
// TWO CALLING CONVENTIONS (design §5.2.4, rev 16), and they are not
// interchangeable. On the REUSE path it is called AFTER the List call found an
// existing session -- there is nothing to pre-write ahead of when reusing
// someone else's already-running session. On the CREATE path it is called
// BEFORE TranscribeV1TranscribeStart, speculatively, against the id that call
// then generates for itself.
func (h *aicallHandler) UpdateListenState(ctx context.Context, aicallID uuid.UUID, callID uuid.UUID, transcribeID uuid.UUID, owns bool) (*aicall.AIcall, error) {
	cur, err := h.db.AIcallGet(ctx, aicallID)
	if err != nil {
		return nil, err
	}

	oldID := listenTranscribeIDFromMetadata(cur)

	// Drop a stale membership first, when step 3's idempotency check found the
	// old session invalid and started a fresh one, or when the create path fell
	// back to reusing a winner. Without this the stale entry survives until its
	// 12h TTL -- harmless functionally, since the old transcribe's events have
	// stopped, but a dangling entry nobody can explain.
	if oldID != uuid.Nil && oldID != transcribeID {
		if errRem := h.cache.ListenAIcallIDRemove(ctx, oldID, aicallID); errRem != nil {
			logrus.Warnf("Could not remove the stale listen resolver membership. aicall_id: %s, err: %v", aicallID, errRem)
		}
	}

	// THE OWNS-MERGE IS SCOPED TO SAME-ID WRITES ONLY (design §5.2.4, review
	// round 12 finding MEDIUM-2, scoped in review round 14 finding HIGH-1).
	//
	// Same id: step 7's bounded retry means the SAME AIcall can have two
	// concurrent runListenStart goroutines racing to write this field on the
	// same row, in an unspecified order. A blind overwrite could persist
	// owns=false for the goroutine that actually started the session, so OR a
	// true in and never let a true already on this row be overwritten.
	//
	// DIFFERENT id: set owns directly, with NO carry-over. This is the branch
	// review round 14 added, and getting it wrong is worse than the race it
	// guards. The create-then-fall-back-to-reuse branch legitimately writes
	// this row against two different transcribe ids in sequence; an
	// unconditional OR would carry a stale owns=true from the abandoned
	// speculative id onto a row now describing a DIFFERENT session this AIcall
	// does not own. Design §5.7.2's stop path then reads `!owns` as false,
	// SKIPS its "never touch it" branch, and tears the session down -- in the
	// two-Cases-one-call scenario, out from under the other Case that is still
	// listening to it.
	if oldID == transcribeID {
		owns = owns || listenOwnsTranscribeFromMetadata(cur)
	}

	// FieldMetadata is a whole-column write, so copy the existing map rather
	// than building a fresh one -- otherwise every other metadata key on the
	// row (prompt snapshots, the auto-audit flag) is silently destroyed.
	metadata := map[string]any{}
	for k, v := range cur.Metadata {
		metadata[k] = v
	}
	metadata[aicall.MetaKeyListenTranscribeID] = transcribeID.String()
	metadata[aicall.MetaKeyListenOwnsTranscribe] = owns

	if errUpdate := h.db.AIcallUpdateNoTouchTMUpdate(ctx, aicallID, map[aicall.Field]any{
		aicall.FieldListenCallID: callID,
		aicall.FieldMetadata:     metadata,
	}); errUpdate != nil {
		return nil, errUpdate
	}

	if errAdd := h.cache.ListenAIcallIDAdd(ctx, transcribeID, aicallID, listenResolverTTL); errAdd != nil {
		return nil, errAdd
	}

	res := *cur
	res.ListenCallID = callID
	res.Metadata = metadata

	return &res, nil
}
```

Then create `bin-ai-manager/pkg/aicallhandler/listen.go` with the three helpers the trigger path and the session path both use. Tasks 21-25 add the rest of this file:

```go
package aicallhandler

import (
	"monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
)

// isListenableCallStatus reports whether a call is in a state transcribe-manager
// will accept. It mirrors transcribehandler.isValidReference's own set exactly;
// diverging would mean starting a transcribe that is then refused.
func isListenableCallStatus(status cmcall.Status) bool {
	return status == cmcall.StatusDialing || status == cmcall.StatusRinging || status == cmcall.StatusProgressing
}

// listenTranscribeIDFromMetadata reads the listen transcribe id off the AIcall's
// metadata, returning uuid.Nil when absent or unparseable.
func listenTranscribeIDFromMetadata(c *aicall.AIcall) uuid.UUID {
	if c.Metadata == nil {
		return uuid.Nil
	}

	tmp, ok := c.Metadata[aicall.MetaKeyListenTranscribeID].(string)
	if !ok {
		return uuid.Nil
	}

	return uuid.FromStringOrNil(tmp)
}

// listenOwnsTranscribeFromMetadata reports whether this AIcall started the
// transcribe session it is listening to, and may therefore stop it.
func listenOwnsTranscribeFromMetadata(c *aicall.AIcall) bool {
	if c.Metadata == nil {
		return false
	}

	owns, ok := c.Metadata[aicall.MetaKeyListenOwnsTranscribe].(bool)
	if !ok {
		return false
	}

	return owns
}
```

Create `bin-ai-manager/pkg/aicallhandler/listen_test.go` as `package aicallhandler` with a small table test for `isListenableCallStatus` (each of dialing/ringing/progressing true; hangup/terminating/canceled false) and for both metadata readers (absent key, wrong type, well-formed value). Tasks 21-25 append to it.

Add `promListenStartTotal` to `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`'s `var` block and its `MustRegister` call:

```go
	promListenStartTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_start_total",
			Help:      "Total number of listen-start attempts by outcome. result values: started, reused, skipped_not_listenable, skipped_confbridge_not_ready, skipped_confbridge_error, skipped_start_locked, failed. All are values of the existing 'result' label -- no new CounterVec.",
		},
		[]string{"result"},
	)
```

**No new metric family.** `skipped_confbridge_not_ready`, `skipped_confbridge_error` and `skipped_start_locked` are additional `result` label values on this same `promListenStartTotal` — Task 26's metric-surface audit does not need a new entry, since it pins metric *names*, not label values.

**On `skipped_start_locked` specifically (design §11 item 16).** The design leaves this as a *proposed* label, not yet added to §5.13's enumerated set, and asks for a deliberate choice before implementation: either give the new outcomes explicit labels, or state plainly that they fold into `skipped_not_listenable`/`failed` and accept the reduced observability. **This plan takes the explicit label**, for the reason the design itself gives: unlike the other new outcomes, a sustained non-zero rate here is a genuinely useful operational signal (heavy concurrent re-open pressure on one AIcall), not just a fail-closed edge case. The other two new outcomes the design flags — the AIcall-liveness rejection and the pre-write failure — deliberately do **not** get their own labels; they fold into `skipped_not_listenable` and `failed` respectively, which is the accepted-reduced-observability half of that same choice.

**Do not meter the deferred release's own error** (design §6, review round 17 finding B-6). It is discarded by design, it runs after `TranscribeV1TranscribeStart` may already have succeeded, and if the `DEL` genuinely did not happen the lock simply falls back to its own TTL — the same outcome the TTL is already sized to tolerate. Only the *acquire* error is metered, as `failed`.

- [ ] **Step 4: Add `ProcessListen` to the `AIcallHandler` interface**

In `bin-ai-manager/pkg/aicallhandler/main.go`, add to the `AIcallHandler` interface, alongside `ProcessTerminate`:

```go
	ProcessListen(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
```

**It must be exported.** `pkg/listenhandler` calls it across a package boundary and mocks it on this interface; design review round 13's HIGH-2 found an unexported `ensureListenAsync` unreachable and unmockable, which made the design's own test items unimplementable as written. Regenerate the mock (`go generate ./pkg/aicallhandler/...`).

- [ ] **Step 5: Add the internal ai-manager route**

**The internal route keeps the plain `/v1/aicalls/{id}/listen` path** (design §5.1). Only the *public, api-manager-facing* path moves to `/service_agents/aicalls/{id}/listen` (Task 27). These are two different services' routes that happen to share a trailing segment, not one route reachable two ways — do not "fix" this one to match the public path.

In `bin-ai-manager/pkg/listenhandler/main.go`, add the regex alongside the existing `regV1AIcallsIDTerminate`:

```go
	regV1AIcallsIDListen = regexp.MustCompile("/v1/aicalls/" + regUUID + "/listen$")
```

and a switch case routing `POST` on it to `processV1AIcallsIDListenPost`, following `processV1AIcallsIDTerminatePost`'s existing entry exactly.

In `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`, add the handler. **Keep it thin** — parse the id, one business-handler call, marshal, return 200, matching `processV1AIcallsIDTerminatePost`'s shape. No orchestration logic belongs in `listenhandler` (design review round 13 finding MEDIUM-4), and per root `CLAUDE.md`'s layering rule the handler returns a domain `*aicall.AIcall` which this layer marshals directly (style (A)) — no `response.*` DTO.

**This step has its own test requirement:** `Test_processV1AIcallsIDListenPost` (Step 1) covers exactly this route, and it must go through `processRequest` so the regex and the dispatcher case are actually exercised. Task 13 documented why that matters in this specific dispatcher — `regV1AIcallsID`'s `$` anchor made a whole route unreachable there — so do not settle for a direct call to `processV1AIcallsIDListenPost`.

Update `bin-ai-manager/docs/architecture.md`'s routing table in the same change (Task 29 covers the rest of that file; the `check-service-docs.sh` hook fires on `pkg/listenhandler/main.go`).

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ ./pkg/listenhandler/ -run 'Test_checkListenEligible|Test_ProcessListen|Test_runListenStart|Test_UpdateListenState_OwnsMerge|Test_waitForConfbridgeReady|Test_processV1AIcallsIDListenPost' -v
```
Expected: PASS, every subtest. **Read the `-v` run list, do not just read the exit code:** all seven tests from Step 1's table must appear (`Test_checkListenEligible`, `Test_ProcessListen`, `Test_runListenStart_EventOrdering`, `Test_runListenStart_StartLock`, `Test_UpdateListenState_OwnsMerge`, `Test_waitForConfbridgeReady`, `Test_processV1AIcallsIDListenPost`). `go test -run` exits 0 when its pattern matches nothing, so a missing or misspelled test — `Test_processV1AIcallsIDListenPost` in particular, since it is the only one in a different package — would make this step pass vacuously without the new route ever having been tested.

- [ ] **Step 7: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass. **Existing `Start` tests must be completely unaffected** — `Start` is not touched by this task and spawns no goroutine. If any `Start` test changes behaviour, something hooked listening into `Start`; remove it (design §5.1).

- [ ] **Step 8: Commit**

Commit message body:

```
- bin-ai-manager: Add ProcessListen, checkListenEligible and runListenStart as the explicit listen trigger
- bin-ai-manager: Add waitForConfbridgeReady, a bounded retry gating listen-start on an exactly-2-party confbridge
- bin-ai-manager: Write the listen state speculatively before starting the transcribe, with an explicit rollback path
- bin-ai-manager: Serialize one AIcall's create-or-reuse sequence behind a per-AIcall Redis lock with a token-checked release
- bin-ai-manager: Scope UpdateListenState's owns merge to same-transcribe-id writes only
- bin-ai-manager: Route POST /v1/aicalls/<aicall-id>/listen to ProcessListen
```

Stage `bin-ai-manager/pkg/aicallhandler/`, `bin-ai-manager/pkg/listenhandler/` and `bin-ai-manager/docs/architecture.md`, then commit with the branch name as the title and the body above.

---

## Task 21: `bin-ai-manager` — `ListenTurnSystemPrompt` and the listen-turn context assembly

A listen turn's LLM context is built **explicitly**, from known-bounded inputs. `getPipecatcallMessages` is deliberately not called: its window would put transcript-driven tool rows in competition with the AIcall's own history, which is the cost this whole design exists to avoid.

Five messages, and the total is constant-shaped regardless of how long the call runs:

| # | Role | Content | Bound |
|---|---|---|---|
| 1 | `system` | `InsightSystemPrompt` — the platform's own Insight guardrails | 1 message |
| 2 | `system` | The frozen, already-substituted prompt snapshot from `Metadata[prompt_snapshots]` | 1 message |
| 3 | `system` | `ListenTurnSystemPrompt` — the watch task and the `notify_agent` contract | 1 message |
| 4 | `user`/`assistant` | The last N Q&A rows, oldest-first | ≤ `AIcallListenQAContextSize` |
| 5 | `user` | The rolling transcript window, with a marker separating seen from new lines | 1 message, ≤ `AIcallListenWindowSize` lines |

**Message 1 is easy to forget and matters.** `startInitMessages` puts `InsightSystemPrompt` first for every Insight AIcall, ahead of the customer's own prompt. The prompt snapshot in metadata holds **only** the substituted `init_prompt` — it never captured `InsightSystemPrompt`. Without message 1, a listen turn would run with none of the platform's guardrails ("base every answer strictly on retrieved data", "never expose raw JSON or tool responses", "never mention tool names") — exactly the rules that keep *unsolicited* output sane.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go` (the new prompt constant)
- Modify: `bin-ai-manager/pkg/aicallhandler/listen.go`
- Test: `bin-ai-manager/pkg/aicallhandler/listen_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bin-ai-manager/pkg/aicallhandler/listen_test.go`:

```go
// Test_buildListenTurnMessages is a golden test on the exact shape of a listen
// turn's context.
//
// It asserts message COUNT and ORDER, not just presence, because the failure
// mode this guards is silent: an Insight AI missing InsightSystemPrompt still
// answers, it just answers without the platform's hallucination and
// tool-leakage guardrails -- and unsolicited output is exactly where those
// matter most.
//
// It also asserts getPipecatcallMessages is NOT called and c.PipecatcallID is
// NOT written. Both are load-bearing: reading the replay window would reintroduce
// the context-eviction problem, and writing the pipecatcall id would rotate the
// agent's own conversational turn out from under an in-flight answer.
func Test_buildListenTurnMessages(t *testing.T) {
	config.SetListenDefaultsForTest()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockMessage := messagehandler.NewMockMessageHandler(mc)
	h := &aicallHandler{messageHandler: mockMessage}
	ctx := context.Background()

	aicallID := uuid.FromStringOrNil("5f6a7b8c-9d0e-4f1a-8b2c-3d4e5f6a7b8c")
	c := &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: aicallID},
		AssistanceType: aicall.AssistanceTypeAI,
		Metadata: map[string]any{
			aicall.MetaKeyPromptSnapshots: []any{
				map[string]any{"prompt": "CUSTOMER INIT PROMPT"},
			},
		},
	}

	// Q&A rows come back newest-first and are reversed to oldest-first. Tool and
	// system rows are filtered out in-process: ApplyFields has no IN support, so
	// the role filter cannot be expressed in the query.
	mockMessage.EXPECT().List(ctx, uint64(30), "", map[message.Field]any{
		message.FieldAIcallID: aicallID,
		message.FieldDeleted:  false,
	}).Return([]*message.Message{
		{Role: message.RoleAssistant, Content: "A1"},
		{Role: message.RoleTool, Content: `{"result":"noise"}`},
		{Role: message.RoleUser, Content: "Q1"},
		{Role: message.RoleSystem, Content: "system noise"},
	}, nil)

	window := []string{"[CUSTOMER] hello", "[AGENT] hi there"}
	newLines := []string{"[CUSTOMER] I want to cancel"}

	res, err := h.buildListenTurnMessages(ctx, c, window, newLines)
	if err != nil {
		t.Fatalf("buildListenTurnMessages returned an unexpected error. err: %v", err)
	}

	if len(res) != 6 {
		t.Fatalf("message count mismatch. expected: 6, got: %d (%v)", len(res), res)
	}

	if res[0]["role"] != "system" || res[0]["content"] != InsightSystemPrompt {
		t.Errorf("message 1 must be InsightSystemPrompt. got: %v", res[0])
	}
	if res[1]["role"] != "system" || res[1]["content"] != "CUSTOMER INIT PROMPT" {
		t.Errorf("message 2 must be the frozen prompt snapshot. got: %v", res[1])
	}
	if res[2]["role"] != "system" || res[2]["content"] != ListenTurnSystemPrompt {
		t.Errorf("message 3 must be ListenTurnSystemPrompt. got: %v", res[2])
	}
	if res[3]["role"] != "user" || res[3]["content"] != "Q1" {
		t.Errorf("message 4 must be the oldest Q&A row. got: %v", res[3])
	}
	if res[4]["role"] != "assistant" || res[4]["content"] != "A1" {
		t.Errorf("message 5 must be the newest Q&A row. got: %v", res[4])
	}
	if res[5]["role"] != "user" {
		t.Errorf("message 6 must be the transcript block. got: %v", res[5])
	}

	transcript, _ := res[5]["content"].(string)
	for _, want := range []string{"[CUSTOMER] hello", "[AGENT] hi there", listenTranscriptNewMarker, "[CUSTOMER] I want to cancel"} {
		if !strings.Contains(transcript, want) {
			t.Errorf("transcript block is missing %q. got:\n%s", want, transcript)
		}
	}

	// The new-lines marker must sit between the seen lines and the new ones --
	// without it the model cannot tell what it has already evaluated and will
	// re-notify about the same thing every turn.
	if strings.Index(transcript, "[AGENT] hi there") > strings.Index(transcript, listenTranscriptNewMarker) {
		t.Errorf("the new-lines marker must come after the already-seen window lines. got:\n%s", transcript)
	}
}
```

Add `"strings"` and the `config` import to the test file if absent.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_buildListenTurnMessages -v
```
Expected: FAIL — `h.buildListenTurnMessages undefined`.

- [ ] **Step 3: Add `ListenTurnSystemPrompt`**

In `bin-ai-manager/pkg/aicallhandler/main.go`, add beside `InsightSystemPrompt` in the same `const` block:

```go
	// ListenTurnSystemPrompt supplies the MECHANICS of a listen evaluation turn
	// and nothing else. The business conditions -- what actually warrants
	// speaking up -- come entirely from the customer's own init_prompt, which is
	// message #2 of every listen turn. That split is deliberate and is the whole
	// point of the feature: triggering is customer-configurable without a schema
	// change and without a hardcoded rule set.
	ListenTurnSystemPrompt = `You are silently monitoring a live phone call in progress. You are NOT talking to anyone right now.

Below you will see a rolling window of what has been said so far, tagged by speaker. Lines after the "--- NEW SINCE YOUR LAST CHECK ---" marker are what you have not evaluated yet; everything before it you have already considered on a previous check.

Your task on each check:
1. Read the new lines in the context of the conversation so far.
2. Decide whether the instructions in your configured prompt warrant alerting the human agent RIGHT NOW.
3. If and only if they do, call the notify_agent tool with one or two sentences written for a busy human mid-call.

CRITICAL RULES:
- Saying nothing is the correct and expected outcome for most checks. Do not manufacture something to say.
- notify_agent is the ONLY way to reach the agent. Any text you produce instead of a tool call is discarded and nobody will ever see it.
- Never repeat a notification you already sent on this call. Check the conversation above before notifying.
- Do not summarize the call, do not narrate what is happening, and do not greet anyone. You are not a participant.
- Do not use other tools unless answering the alert genuinely requires information the transcript does not contain.`
```

- [ ] **Step 4: Implement the assembly**

Append to `bin-ai-manager/pkg/aicallhandler/listen.go`:

```go
// listenTranscriptNewMarker separates the lines a previous turn already
// evaluated from the ones this turn is seeing for the first time.
//
// Without it the model re-reads the whole window every turn with no way to tell
// what is new, and re-notifies about the same thing repeatedly -- the single
// most likely way this feature becomes annoying rather than useful.
const listenTranscriptNewMarker = "--- NEW SINCE YOUR LAST CHECK ---"

// buildListenTurnMessages assembles a listen evaluation turn's LLM context.
//
// It is built EXPLICITLY, from known-bounded inputs, and getPipecatcallMessages
// is deliberately never called. Two reasons, both structural:
//
//   - The transcript is not, and must never become, message rows. Rows would be
//     webhook-published per spoken sentence, rendered in the agent's panel, and
//     would consume the replay window.
//   - The context size here is a constant, independent of call length. A replay
//     window is not.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.2.
func (h *aicallHandler) buildListenTurnMessages(ctx context.Context, c *aicall.AIcall, window []string, newLines []string) ([]map[string]any, error) {
	res := []map[string]any{}

	// (1) The platform's own Insight guardrails.
	//
	// startInitMessages writes this first for every Insight AIcall, ahead of the
	// customer's prompt -- but the frozen prompt snapshot in Metadata holds ONLY
	// the substituted init_prompt and never captured this. Omitting it would run
	// unsolicited output with none of the "base answers strictly on retrieved
	// data / never expose raw JSON or tool responses / never mention tool names"
	// rules, which is exactly where they matter most. It is a fixed platform
	// constant, so this costs no DB read.
	res = append(res, map[string]any{
		"role":    string(message.RoleSystem),
		"content": InsightSystemPrompt,
	})

	// (2) The customer's own prompt, frozen and already substituted at AIcall
	// start -- so no DB read and no re-substitution here.
	if snapshot := listenPromptSnapshot(c); snapshot != "" {
		res = append(res, map[string]any{
			"role":    string(message.RoleSystem),
			"content": snapshot,
		})
	}

	// (3) The mechanics of a listen turn.
	res = append(res, map[string]any{
		"role":    string(message.RoleSystem),
		"content": ListenTurnSystemPrompt,
	})

	// (4) Recent Q&A, so the AI has continuity with what the agent asked and
	// with its own earlier notifications.
	//
	// Over-fetch and filter in process: ApplyFields builds equality clauses per
	// field and has no IN support, so "role in (user, assistant)" cannot be
	// expressed in the query. FieldDeleted:false IS expressible and is applied,
	// unlike getPipecatcallMessages which does not filter deleted rows today --
	// this is a new code path, so it gets the correct filter rather than
	// inheriting that gap.
	qaRowsDesc, err := h.messageHandler.List(ctx, 30, "", map[message.Field]any{
		message.FieldAIcallID: c.ID,
		message.FieldDeleted:  false,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not get the qa messages")
	}

	budget := config.Get().AIcallListenQAContextSize
	qa := []map[string]any{}
	// qaRowsDesc is newest-first; walk it that way, take the newest `budget`
	// conversational rows, then reverse into chronological order for the LLM.
	for _, m := range qaRowsDesc {
		if len(qa) >= budget {
			break
		}
		if m.Role != message.RoleUser && m.Role != message.RoleAssistant {
			continue
		}
		if strings.TrimSpace(m.Content) == "" {
			// Empty-content rows are the tool-call carriers; they have no
			// conversational value here and would waste the budget.
			continue
		}
		qa = append(qa, map[string]any{
			"role":    string(m.Role),
			"content": m.Content,
		})
	}
	for i, j := 0, len(qa)-1; i < j; i, j = i+1, j-1 {
		qa[i], qa[j] = qa[j], qa[i]
	}
	res = append(res, qa...)

	// (5) The transcript block.
	res = append(res, map[string]any{
		"role":    string(message.RoleUser),
		"content": buildListenTranscriptBlock(window, newLines),
	})

	return res, nil
}

// buildListenTranscriptBlock renders the rolling window with the new lines
// marked off.
//
// The window already contains the new lines (both lists are appended to on
// intake), so the seen portion is the window minus its own tail.
func buildListenTranscriptBlock(window []string, newLines []string) string {
	seen := window
	if len(newLines) > 0 && len(window) >= len(newLines) {
		seen = window[:len(window)-len(newLines)]
	}

	var sb strings.Builder
	sb.WriteString("Live call transcript so far:\n")
	for _, line := range seen {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString(listenTranscriptNewMarker)
	sb.WriteString("\n")
	for _, line := range newLines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// listenPromptSnapshot returns the frozen, already-substituted customer prompt
// for this AIcall.
//
// For AssistanceTypeAI there is exactly one snapshot. For AssistanceTypeTeam
// there is one per member, and the right one is whichever matches
// CurrentMemberID -- falling back to the first, because a listen turn with the
// wrong team member's prompt is still far better than one with no customer
// instructions at all.
func listenPromptSnapshot(c *aicall.AIcall) string {
	if c.Metadata == nil {
		return ""
	}

	raw, ok := c.Metadata[aicall.MetaKeyPromptSnapshots].([]any)
	if !ok || len(raw) == 0 {
		return ""
	}

	first := ""
	for _, item := range raw {
		snapshot, okItem := item.(map[string]any)
		if !okItem {
			continue
		}

		prompt, _ := snapshot["prompt"].(string)
		if prompt == "" {
			continue
		}
		if first == "" {
			first = prompt
		}

		memberID, _ := snapshot["member_id"].(string)
		if c.CurrentMemberID != uuid.Nil && memberID == c.CurrentMemberID.String() {
			return prompt
		}
	}

	return first
}
```

Add `"strings"`, `"monorepo/bin-ai-manager/models/message"` and `"github.com/pkg/errors"` to `listen.go`'s imports.

`Metadata` round-trips through JSON, so a `[]aicall.PromptSnapshot` comes back as `[]any` of `map[string]any` — hence the untyped reads above. Confirm the JSON tag on `PromptSnapshot.MemberID` is `member_id` and on `.Prompt` is `prompt` before relying on those keys:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -n "type PromptSnapshot" -A 8 models/aicall/main.go
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_buildListenTurnMessages -v
```
Expected: PASS.

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 7: Commit**

Commit message body:

```
- bin-ai-manager: Add ListenTurnSystemPrompt describing the watch task and the notify_agent contract
- bin-ai-manager: Assemble listen-turn LLM context explicitly, never through getPipecatcallMessages
```

Stage `bin-ai-manager/pkg/aicallhandler/`, then commit with the branch name as the title and the body above.

---

## Task 22: `bin-ai-manager` — `RunListenTurn`, `runListenTurnWithLines` and `startListenPipecatcall`

The evaluation turn itself.

**The split into two functions is not cosmetic.** The design calls for a final flush turn at hangup that evaluates the last few buffered lines. But an ordinary `RunListenTurn` drains the buffer *itself* and respects the debounce lock, so the hangup path has no way to say "evaluate these lines I already drained, and do it now." Extracting the turn body as `runListenTurnWithLines(ctx, c, lines)` gives both callers what they need: `RunListenTurn` checks preconditions, drains atomically, then calls it; the hangup path calls it directly with lines it already holds.

**The turn's pipecatcall id is never written to the AIcall row.** That single decision is what makes the whole design safe: no `AIcallUpdate` per turn means no `tm_update` bump and no `Send` cooldown interference; `interruptPreviousPipecatcall` is never called so an in-flight answer to the agent is never killed; and the id mismatch itself becomes the drop signal for any text the turn emits. Tool calls still route correctly, because pipecat-manager resolves the AIcall from the *pipecatcall's* `ReferenceID`, not from `AIcall.PipecatcallID`.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/listen.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`
- Test: `bin-ai-manager/pkg/aicallhandler/listen_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `bin-ai-manager/pkg/aicallhandler/listen_test.go`:

```go
// Test_RunListenTurn covers every precondition and outcome.
func Test_RunListenTurn(t *testing.T) {
	tests := []struct {
		name string

		flagEnabled bool
		status      aicall.Status
		refType     aicall.ReferenceType
		metadata    map[string]any
		pendingLines []string
		turnCount    int64
		registerErr  error

		expectStopListening bool
		expectPipecatStart  bool
		expectResult        string
	}{
		{
			name:                "flag off stops listening entirely",
			// Not merely "clears bookkeeping": a bare state clear would leave a
			// still-running owned STT session with its handle lost, so a
			// rollback would strand a billed stream until the call ended.
			flagEnabled:         false,
			expectStopListening: true,
			expectResult:        "skipped_disabled",
		},
		{name: "terminated aicall stops listening", expectStopListening: true, expectResult: "skipped_invalid"},
		{name: "non contact_case reference type stops listening", expectStopListening: true, expectResult: "skipped_invalid"},
		{name: "missing listen_transcribe_id metadata stops listening", expectStopListening: true, expectResult: "skipped_invalid"},
		{name: "empty pending buffer skips without stopping", expectResult: "skipped_empty"},
		{
			name:                "turn cap exceeded stops listening",
			expectStopListening: true,
			expectResult:        "skipped_cap",
		},
		{
			name: "turn-id registration failure aborts before starting a pipecatcall",
			// Proceeding unregistered would make every tool call from this turn
			// resolve listenTurn=false: its rows get permanently tagged
			// OriginNone and its notify_agent call gets rejected -- the exact
			// failure the registration exists to prevent.
			registerErr:        fmt.Errorf("redis unavailable"),
			expectPipecatStart: false,
			expectResult:       "skipped_register_failed",
		},
		{
			name:               "happy path runs one turn",
			expectPipecatStart: true,
			expectResult:       "ran",
		},
	}
	// Fill in each row against Step 3. Assert on the metric label as the
	// outcome signal, plus the mock expectations named in each row.
	_ = tests
}

// Test_RunListenTurn_DoesNotWritePipecatcallID pins the load-bearing decision.
//
// Writing the turn's throwaway id to the AIcall row would rotate the agent's own
// conversational turn out from under an in-flight answer, bump tm_update into
// Send's cooldown, and destroy the id mismatch that is the drop signal for
// anything the turn emits. All three at once, silently.
func Test_RunListenTurn_DoesNotWritePipecatcallID(t *testing.T) {
	// Drive one happy-path turn with a dbhandler mock that has NO
	// AIcallUpdate / AIcallUpdateIfActive / AIcallUpdateNoTouchTMUpdate
	// expectation at all -- gomock fails the test if the turn writes the row.
	//
	// Also assert the id passed to PipecatV1PipecatcallStart differs from
	// c.PipecatcallID and matches the id passed to ListenTurnPipecatcallIDAdd.
}

// Test_runListenTurnWithLines_HangupPath pins that the hangup flush can
// evaluate lines it already holds, bypassing both the drain and the debounce
// lock.
//
// This is why the turn body is a separate function at all: RunListenTurn drains
// the buffer itself and respects the lock, so the hangup path -- which has
// already drained -- would otherwise have no way in.
func Test_runListenTurnWithLines_HangupPath(t *testing.T) {
	// Call runListenTurnWithLines directly with a non-empty lines slice and
	// assert: no ListenPendingPopAll, no ListenTurnTryLock, one
	// PipecatV1PipecatcallStart.
}
```

Fill each row in concretely before running.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run 'Test_RunListenTurn|Test_runListenTurnWithLines' -v
```
Expected: FAIL — `h.RunListenTurn undefined`.

- [ ] **Step 3: Implement the turn**

Append to `bin-ai-manager/pkg/aicallhandler/listen.go`:

```go
// defaultListenTurnTimeout is how long a listen evaluation turn's pipecatcall
// may live before it is terminated, in milliseconds.
const defaultListenTurnTimeout = 60000

// RunListenTurn evaluates whatever has been said since the last turn.
//
// Preconditions first, then an ATOMIC drain, then the turn body. The drain is a
// single LPOP-count command precisely so a line pushed concurrently cannot be
// lost between a read and a trim.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.
func (h *aicallHandler) RunListenTurn(ctx context.Context, aicallID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "RunListenTurn",
		"aicall_id": aicallID,
	})

	c, err := h.Get(ctx, aicallID)
	if err != nil {
		// No AIcall means nothing to stop and nothing to evaluate.
		promListenTurnTotal.WithLabelValues("skipped_invalid").Inc()
		return
	}

	// The flag check lives HERE, in the require-list, not in a separate earlier
	// step: everything a failing condition does next needs `c`, which does not
	// exist until the fetch above. It is also what makes a rollback real -- with
	// no flag read on this path, a session that started while the flag was on
	// would run to call-end or the turn cap regardless.
	if !config.Get().AIcallListenEnabled {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues("skipped_disabled").Inc()
		return
	}

	if c.Status != aicall.StatusProgressing ||
		c.ReferenceType != aicall.ReferenceTypeContactCase ||
		listenTranscribeIDFromMetadata(c) == uuid.Nil {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues("skipped_invalid").Inc()
		return
	}

	// Hard backstop against a pathologically long call. Reaching it stops
	// listening cleanly; the Q&A panel keeps working normally.
	turns, errCount := h.cache.ListenTurnCountIncr(ctx, aicallID, listenBufferTTL())
	if errCount != nil {
		log.Warnf("Could not increment the listen turn counter. err: %v", errCount)
	} else if turns > int64(config.Get().AIcallListenMaxTurnsPerAIcall) {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues("skipped_cap").Inc()
		return
	}

	lines, err := h.cache.ListenPendingPopAll(ctx, aicallID)
	if err != nil {
		log.Errorf("Could not drain the pending buffer. err: %v", err)
		promListenTurnTotal.WithLabelValues("failed").Inc()
		return
	}
	if len(lines) == 0 {
		promListenTurnTotal.WithLabelValues("skipped_empty").Inc()
		return
	}

	h.runListenTurnWithLines(ctx, c, lines)
}

// runListenTurnWithLines is the turn body, taking the lines to evaluate rather
// than draining them.
//
// EXTRACTED DELIBERATELY, and this is not a refactoring nicety. The hangup path
// must evaluate the last words of a call immediately, without waiting for the
// debounce lock and without a buffer left to drain -- it has already drained
// it. RunListenTurn owns the preconditions, the counter, the lock and the
// drain; this owns the evaluation. Two callers, one turn body, no duplicated
// LLM-invocation logic.
func (h *aicallHandler) runListenTurnWithLines(ctx context.Context, c *aicall.AIcall, lines []string) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "runListenTurnWithLines",
		"aicall_id": c.ID,
	})

	window, errWindow := h.cache.ListenWindowGet(ctx, c.ID)
	if errWindow != nil {
		log.Warnf("Could not read the transcript window; evaluating the new lines alone. err: %v", errWindow)
		window = lines
	}

	llmMessages, err := h.buildListenTurnMessages(ctx, c, window, lines)
	if err != nil {
		log.Errorf("Could not build the listen turn context. err: %v", err)
		promListenTurnTotal.WithLabelValues("failed").Inc()
		return
	}

	// A FRESH, THROWAWAY id -- never written to c.PipecatcallID. That single
	// decision is what keeps this whole design safe:
	//   - no AIcallUpdate per turn, so no tm_update bump and no Send cooldown
	//     interference,
	//   - interruptPreviousPipecatcall is never called, so an in-flight answer
	//     to the agent is never killed,
	//   - and the id mismatch itself becomes the drop signal for any text this
	//     turn emits (see messagehandler's foreign-pipecatcall guard).
	//
	// Tool calls still route correctly: pipecat-manager resolves the AIcall from
	// the PIPECATCALL's ReferenceID (= c.ID), not from AIcall.PipecatcallID.
	turnPipecatcallID := h.utilHandler.UUIDCreate()

	// Register it as a genuine listen turn BEFORE starting the session, and
	// ABORT if that fails. Proceeding unregistered would make every tool call
	// this turn issues resolve listenTurn=false: its rows would be permanently
	// tagged OriginNone (never excluded from future Q&A replay) and its
	// notify_agent call would be rejected -- precisely the failure the
	// registration exists to prevent, reintroduced through the one write this
	// function owns.
	ttl := time.Duration(config.Get().AIcallListenTurnPipecatcallIDTTLSeconds) * time.Second
	if errAdd := h.cache.ListenTurnPipecatcallIDAdd(ctx, c.ID, turnPipecatcallID, ttl); errAdd != nil {
		log.Warnf("Could not register the listen turn id; skipping this turn. err: %v", errAdd)
		promListenTurnTotal.WithLabelValues("skipped_register_failed").Inc()
		return
	}

	pc, err := h.startListenPipecatcall(ctx, c, turnPipecatcallID, llmMessages)
	if err != nil {
		log.Errorf("Could not start the listen pipecatcall. err: %v", err)
		promListenTurnTotal.WithLabelValues("failed").Inc()
		return
	}

	if errTerm := h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultListenTurnTimeout); errTerm != nil {
		// Non-fatal: the turn already ran. A missed terminate leaves one idle
		// session that its own timeout will reap.
		log.Warnf("Could not schedule the listen pipecatcall terminate. err: %v", errTerm)
	}

	promListenTurnTotal.WithLabelValues("ran").Inc()
}

// startListenPipecatcall is a sibling of startPipecatcall that takes the
// pipecatcall id and the message list as parameters, instead of reading
// c.PipecatcallID and calling getPipecatcallMessages.
//
// STTTypeNone and TTSTypeNone are not incidental: a listen turn has no audio
// legs at all. It is a text-in, tool-call-out evaluation. That is also why the
// two STT-driven pipecat message handlers need no foreign-pipecatcall guard --
// a listen turn structurally cannot produce their events.
func (h *aicallHandler) startListenPipecatcall(ctx context.Context, c *aicall.AIcall, pipecatcallID uuid.UUID, llmMessages []map[string]any) (*pmpipecatcall.Pipecatcall, error) {
	res, err := h.reqHandler.PipecatV1PipecatcallStart(
		ctx,
		pipecatcallID,
		c.CustomerID,
		c.ActiveflowID,
		pmpipecatcall.ReferenceTypeAICall,
		c.ID,
		pmpipecatcall.LLMType(c.AIEngineModel),
		llmMessages,
		pmpipecatcall.STTTypeNone,
		"",
		pmpipecatcall.TTSTypeNone,
		"",
		"",
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not start the listen pipecatcall")
	}

	return res, nil
}

// listenBufferTTL is the TTL applied to the pending buffer, the rolling window,
// the debounce lock and the turn counter. The listen-turn id set uses its own,
// much shorter TTL -- it only needs to outlive one turn.
func listenBufferTTL() time.Duration {
	return time.Duration(config.Get().AIcallListenBufferTTLHours) * time.Hour
}
```

Add `pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"` to `listen.go`'s imports.

Add `RunListenTurn` to the `AIcallHandler` interface in `bin-ai-manager/pkg/aicallhandler/main.go`:

```go
	RunListenTurn(ctx context.Context, aicallID uuid.UUID)
```

- [ ] **Step 4: Add the metric**

Add to `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`'s `var` block and `MustRegister`:

```go
	promListenTurnTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_turn_total",
			Help:      "Total listen evaluation turns by outcome. skipped_locked measured against ran is the direct read on how much LLM spend the debounce is saving -- near-zero skipped_locked means the interval is too short for the traffic.",
		},
		[]string{"result"},
	)
```

Label values in use: `ran`, `skipped_locked`, `skipped_empty`, `skipped_cap`, `skipped_disabled`, `skipped_invalid`, `skipped_register_failed`, `failed`.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run 'Test_RunListenTurn|Test_runListenTurnWithLines' -v
```
Expected: PASS. `stopListening` is Task 25 — if it is not written yet, stub it as a no-op with a `// TODO(Task 25)` and remove the stub there. Prefer doing Task 25 first if that ordering is easier.

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 7: Commit**

Commit message body:

```
- bin-ai-manager: Add RunListenTurn with an atomic pending-buffer drain and turn cap
- bin-ai-manager: Extract runListenTurnWithLines so the hangup flush can evaluate already-drained lines
- bin-ai-manager: Add startListenPipecatcall running each turn on a throwaway pipecatcall id
```

Stage `bin-ai-manager/pkg/aicallhandler/`, then commit with the branch name as the title and the body above.

---

## Task 23: `bin-ai-manager` — `EventTMTranscriptCreated` intake and the speaker tags

Layer 1. **No LLM, no DB write, no webhook** — the whole point is that the per-event cost is bounded to one Redis round trip, because this handler sees every final STT result on the platform, not just the calls being listened to.

**Use the speaker mapping confirmed in Task 0 Step 1.** If the confirmed mapping is the reverse of the design's structural assumption, write the reverse here and pin the confirmed values in the test. Do not guess.

**Updated to reflect design rev 11-14 (§5.1.1 step 7, §5.9).** The mapping is no longer a bare assumption resting only on "`Case.ReferenceID` happens to be the A-leg": it now follows from a code-checked invariant confirmed while answering a direct architectural question — `in` is always the listened channel's own remote party, i.e. `Case.Peer`, and `case_create` itself already guarantees that peer is CRM-eligible (never an internal agent/extension/SIP/conference/AI endpoint — `bin-flow-manager/pkg/activeflowhandler/actionhandle.go`'s `isCRMEligiblePeer`). Task 20's `waitForConfbridgeReady` (design §5.1.1 step 7) additionally guards against the topology this mapping assumes (a live, exactly-2-party confbridge) not yet holding when listening would otherwise start. What Task 0 Step 1 still must confirm empirically is narrower than before: not *whether* the mechanism is channel-relative (confirmed via a real production transcript sample during design review) or *which* leg is transcribed (confirmed structurally), but whether one real agent-bridged call's actual transcript segments match known speaker identity end-to-end. Two residual risks the structural invariant does **not** close, both documented as open in design §11 items 11-12 rather than fixed here: (a) an inbound call whose peer address is CRM-eligible but is actually staff calling in via a plain DID rather than the normal agent-dial path, and (b) a call-transfer mid-listen, which was not investigated. Neither is in this plan's scope.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/listen.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go` (the interface)
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`
- Test: `bin-ai-manager/pkg/aicallhandler/listen_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `bin-ai-manager/pkg/aicallhandler/listen_test.go`:

```go
// Test_speakerTag pins the in/out -> CUSTOMER/AGENT mapping.
//
// This is a golden test on purpose. A reversed mapping is a SILENT correctness
// failure: the AI keeps working and keeps notifying, it just attributes the
// customer's words to the agent and vice versa -- which can produce a
// confidently wrong proactive message (e.g. telling the agent the customer
// threatened to cancel when it was the agent who said it).
//
// The values below are the ones confirmed empirically in Task 0 Step 1. If that
// check found the reverse, this test and speakerTag both flip together.
func Test_speakerTag(t *testing.T) {
	tests := []struct {
		name      string
		direction tmtranscript.Direction
		expect    string
	}{
		{"in is the customer", tmtranscript.DirectionIn, "[CUSTOMER]"},
		{"out is the agent", tmtranscript.DirectionOut, "[AGENT]"},
		{"both is not guessed", tmtranscript.DirectionBoth, "[SPEAKER]"},
		{"unknown is not guessed", tmtranscript.Direction("weird"), "[SPEAKER]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := speakerTag(tt.direction); got != tt.expect {
				t.Errorf("speakerTag(%q) mismatch. expected: %q, got: %q", tt.direction, tt.expect, got)
			}
		})
	}
}

// Test_EventTMTranscriptCreated covers intake's drops and its fan-out.
func Test_EventTMTranscriptCreated(t *testing.T) {
	tests := []struct {
		name string

		transcript *tmtranscript.Transcript

		expectResolve bool
		resolvedIDs   []uuid.UUID
		lockAcquired  []bool

		expectBuffered int
		expectTurns    int
	}{
		{
			name: "a deleted transcript is dropped before any redis call",
			// transcripthandler.dbDelete publishes transcript_created on DELETE
			// too (a known upstream bug). Without this guard a deleted line
			// replays into the LLM as freshly-spoken content.
			expectResolve: false,
		},
		{
			name:          "an empty message is dropped before any redis call",
			expectResolve: false,
		},
		{
			name: "an unknown transcribe id is dropped after one redis call and no DB call",
			// This is 99.9% of platform events. It must cost one SMEMBERS and
			// nothing else -- no DB query, no RPC.
			expectResolve: true,
			resolvedIDs:   nil,
		},
		{
			name: "two AIcalls sharing one transcribe each get the segment buffered",
			// Two Cases open on one call. A single-valued resolver key would let
			// the second listener silently steal the first's mapping; the set
			// makes both independent.
			expectResolve:  true,
			resolvedIDs:    []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
			lockAcquired:   []bool{true, true},
			expectBuffered: 2,
			expectTurns:    2,
		},
		{
			name:           "a buffered-but-locked segment runs no turn",
			expectResolve:  true,
			resolvedIDs:    []uuid.UUID{uuid.Must(uuid.NewV4())},
			lockAcquired:   []bool{false},
			expectBuffered: 1,
			expectTurns:    0,
		},
	}
	// Fill in each row against Step 2's implementation.
	_ = tests
}
```

- [ ] **Step 2: Run the tests to verify they fail, then implement**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run 'Test_speakerTag|Test_EventTMTranscriptCreated' -v
```
Expected: FAIL — `undefined: speakerTag`.

Append to `bin-ai-manager/pkg/aicallhandler/listen.go`:

```go
// speakerTag renders a transcript segment's direction as a structural speaker
// label.
//
// The labels are STRUCTURAL, not localized, so prompt behaviour does not fork by
// call language.
//
// The mapping (design §5.9, wording corrected in review round 11 finding
// LOW-2 to match Asterisk's actual read/write convention unambiguously):
// transcript.Direction is relative to the transcribed CHANNEL's own read/write
// direction -- "in" is audio Asterisk reads FROM that channel (what the
// channel's own party said), "out" is audio Asterisk writes TO it (what was
// played to them). The listened channel's own party is always Case.Peer --
// case_create only ever creates a Case from a CRM-eligible peer (never an
// internal agent/extension/SIP/conference/AI endpoint), so in=CUSTOMER is a
// code-checked invariant, not a bare assumption resting on which leg happens
// to be transcribed. Once that leg is bridged to an agent, out=AGENT follows.
//
// Depends on Task 20's waitForConfbridgeReady already having confirmed a live,
// exactly-2-party confbridge before listening starts -- with 3+ parties in the
// bridge, "out" stops reliably meaning "the agent" (design §5.1.1 step 7's
// closing note); "in" is unaffected regardless of party count, since it never
// depended on who else is in the bridge.
//
// VERIFIED EMPIRICALLY before this shipped (Task 0 Step 1 of the implementation
// plan): the general channel-relative mechanism was independently confirmed
// against real production transcript data during design review (design §5.9),
// and which leg is transcribed is confirmed structurally, not assumed -- but
// neither substitutes for Task 0 Step 1's own confirmation that one real
// agent-bridged call's transcript segments match known speaker identity
// end-to-end. Test_speakerTag pins whichever mapping that check confirms.
//
// DirectionBoth (and anything unrecognised) is tagged [SPEAKER] rather than
// guessed -- a wrong attribution is worse than an unattributed line.
func speakerTag(direction tmtranscript.Direction) string {
	switch direction {
	case tmtranscript.DirectionIn:
		return "[CUSTOMER]"
	case tmtranscript.DirectionOut:
		return "[AGENT]"
	default:
		return "[SPEAKER]"
	}
}

// EventTMTranscriptCreated is layer 1: transcript intake.
//
// NO LLM, NO DB WRITE, NO WEBHOOK. This runs for every final STT result
// PLATFORM-WIDE -- flow-driven, summary-driven, customer-started -- not just for
// calls being listened to, so the per-event cost has to stay at one Redis round
// trip. An empty resolver set means "not a session we started" and is the
// overwhelmingly common outcome.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.3.
func (h *aicallHandler) EventTMTranscriptCreated(ctx context.Context, evt *tmtranscript.Transcript) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "EventTMTranscriptCreated",
		"transcribe_id": evt.TranscribeID,
	})

	// transcripthandler.dbDelete publishes EventTypeTranscriptCreated on DELETE
	// as well as on create -- a known upstream bug this design defends against
	// rather than fixes, because changing the emitted event type is a
	// routing-key-visible change affecting every current subscriber. Without
	// this guard a deleted line replays into the LLM as freshly-spoken content.
	if evt.TMDelete != nil || strings.TrimSpace(evt.Message) == "" {
		promListenSegmentTotal.WithLabelValues("dropped_deleted").Inc()
		return
	}

	aicallIDs, err := h.cache.ListenAIcallIDsGet(ctx, evt.TranscribeID)
	if err != nil {
		log.Warnf("Could not resolve the listening aicalls. err: %v", err)
		promListenSegmentTotal.WithLabelValues("dropped_unknown").Inc()
		return
	}
	if len(aicallIDs) == 0 {
		promListenSegmentTotal.WithLabelValues("dropped_unknown").Inc()
		return
	}

	line := fmt.Sprintf("%s %s", speakerTag(evt.Direction), strings.TrimSpace(evt.Message))
	ttl := listenBufferTTL()
	windowSize := config.Get().AIcallListenWindowSize
	interval := time.Duration(config.Get().AIcallListenEvaluateIntervalSeconds) * time.Second

	// Fan out per listening AIcall. Two Cases open on one call each get their
	// own AIcall and each buffers and debounces independently.
	for _, aicallID := range aicallIDs {
		if errPending := h.cache.ListenPendingPush(ctx, aicallID, line, ttl); errPending != nil {
			log.Warnf("Could not buffer the pending line. aicall_id: %s, err: %v", aicallID, errPending)
			continue
		}
		if errWindow := h.cache.ListenWindowPush(ctx, aicallID, line, windowSize, ttl); errWindow != nil {
			log.Warnf("Could not buffer the window line. aicall_id: %s, err: %v", aicallID, errWindow)
		}
		promListenSegmentTotal.WithLabelValues("buffered").Inc()

		// Leaky-bucket debounce. Losing the race is the NORMAL case and is not
		// an error -- the line stays buffered for whichever turn did win, which
		// is exactly what decouples LLM invocations from speech volume.
		acquired, errLock := h.cache.ListenTurnTryLock(ctx, aicallID, interval)
		if errLock != nil {
			log.Warnf("Could not take the listen turn lock. aicall_id: %s, err: %v", aicallID, errLock)
			continue
		}
		if !acquired {
			promListenTurnTotal.WithLabelValues("skipped_locked").Inc()
			continue
		}

		// Detached: this handler must return promptly, and the turn's own
		// lifetime is bounded by its pipecatcall terminate.
		go func(id uuid.UUID) {
			turnCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			h.RunListenTurn(turnCtx, id)
		}(aicallID)
	}
}
```

Add `"fmt"` and `tmtranscript "monorepo/bin-transcribe-manager/models/transcript"` to `listen.go`'s imports, and add to the `AIcallHandler` interface in `main.go`:

```go
	EventTMTranscriptCreated(ctx context.Context, evt *tmtranscript.Transcript)
```

Add the metric to `metrics_listen.go`:

```go
	promListenSegmentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_segment_total",
			Help:      "Total transcript segments seen by listen intake, by outcome. dropped_unknown dominates by design -- this handler sees every final STT result platform-wide.",
		},
		[]string{"result"},
	)
```

- [ ] **Step 3: Run the tests to verify they pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run 'Test_speakerTag|Test_EventTMTranscriptCreated' -v
```
Expected: PASS, every subtest.

- [ ] **Step 4: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 5: Commit**

Commit message body:

```
- bin-ai-manager: Add EventTMTranscriptCreated intake buffering segments with a cross-replica debounce
- bin-ai-manager: Tag transcript lines by speaker using the empirically confirmed in/out mapping
```

Stage `bin-ai-manager/pkg/aicallhandler/`, then commit with the branch name as the title and the body above.

---

## Task 24: `bin-ai-manager` — subscribe to `transcript_created`

Three coupled edits plus a golden test that pins all of them. The golden test is position-sensitive, so **append the new pattern at the end** of `topicPatterns`.

**Files:**
- Modify: `bin-ai-manager/pkg/subscribehandler/main.go`
- Modify: `bin-ai-manager/pkg/subscribehandler/transcribemanager.go`
- Modify: `bin-ai-manager/pkg/subscribehandler/binding_golden_test.go`

- [ ] **Step 1: Update the golden test first, and watch it fail**

In `bin-ai-manager/pkg/subscribehandler/binding_golden_test.go`, append the new pattern to `expected` and change both occurrences of `11` to `12`:

```go
		"conference-manager.conference.*.deleted",
		"transcribe-manager.transcript.*.created",
	}

	// design §5 + VOIP-1422 + NOJIRA Insight AI realtime listen: ai-manager
	// binds exactly 12 patterns.
	if len(topicPatterns) != 12 {
		t.Fatalf("topicPatterns count mismatch. expected: 12, got: %d (%v)", len(topicPatterns), topicPatterns)
	}
```

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/subscribehandler/ -run Test_topicPatterns_golden -v
```
Expected: FAIL — count mismatch, expected 12 got 11. The golden test is doing its job.

- [ ] **Step 2: Add the pattern**

In `bin-ai-manager/pkg/subscribehandler/main.go`, append to `topicPatterns` (last position — the golden test pins order):

```go
	// Insight AI realtime call listening (docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.3.1). A static wildcard
	// rather than a dynamic per-transcribe binding: the wildcard's cost is one
	// AMQP delivery, one goroutine, one JSON unmarshal and one Redis SMEMBERS
	// per final STT result platform-wide, with no DB query and no RPC, whereas a
	// bind/unbind lifecycle's failure mode is a permanently leaked binding.
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameTranscribeManager), tmtranscript.EventTypeTranscriptCreated),
```

Add `tmtranscript "monorepo/bin-transcribe-manager/models/transcript"` to that file's imports, and update the doc comment above `topicPatterns` to mention the new pair.

- [ ] **Step 3: Add the dispatch case**

In `processEvent`'s switch, after the pipecat-manager cases, add a new section using the already-declared `publisherTranscribeManager` constant (it exists and is currently unused):

```go
	// transcribe-manager
	case m.Publisher == publisherTranscribeManager && m.Type == tmtranscript.EventTypeTranscriptCreated:
		err = h.processEventTMTranscriptCreated(ctx, m)
```

- [ ] **Step 4: Replace the commented-out ghost**

`bin-ai-manager/pkg/subscribehandler/transcribemanager.go` currently contains **nothing but** a fully commented-out `processEventTMTranscriptCreated` that does `GetByTranscribeID` then `ChatMessage` — i.e. precisely the naive one-LLM-call-per-segment design this feature rejects. Delete the commented block entirely (leaving it would read as an endorsed alternative) and replace the file's contents with:

```go
package subscribehandler

import (
	"context"
	"encoding/json"

	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processEventTMTranscriptCreated handles transcribe-manager's
// transcript_created event.
//
// It does as little as possible on purpose. This fires for EVERY final STT
// result on the platform -- flow-driven, summary-driven, customer-started -- and
// processEventRun spawns a goroutine per event, so anything expensive here is
// multiplied by total platform transcription volume, not by the number of calls
// actually being listened to. Unmarshal, hand off, return.
//
// The ownership filter (one Redis SMEMBERS) lives in
// aicallHandler.EventTMTranscriptCreated, which is where the listen state is.
func (h *subscribeHandler) processEventTMTranscriptCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventTMTranscriptCreated",
		"event": m,
	})

	var evt tmtranscript.Transcript
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.aicallHandler.EventTMTranscriptCreated(ctx, &evt)

	return nil
}
```

- [ ] **Step 5: Run the golden test to verify it passes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/subscribehandler/ -v
```
Expected: PASS, with the golden test showing 12 patterns.

- [ ] **Step 6: Confirm the ghost is really gone**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -n "GetByTranscribeID\|ChatMessage" pkg/subscribehandler/transcribemanager.go
```
Expected: no output.

- [ ] **Step 7: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 8: Commit**

Commit message body:

```
- bin-ai-manager: Bind transcribe-manager.transcript.*.created and dispatch it to listen intake
- bin-ai-manager: Replace the commented-out per-segment transcript handler with the real implementation
```

Stage `bin-ai-manager/pkg/subscribehandler/`, then commit with the branch name as the title and the body above.

---

## Task 25: `bin-ai-manager` — the stop paths

Three entry points, one shared helper.

**`stopListening` never calls `ProcessTerminate`.** That function ends the *AIcall itself* — status to terminated plus an activeflow service stop — which would kill the agent's entire Insight Q&A session. Stopping *listening* must leave the panel working normally. The naming is close enough that this deserves its own regression test.

**Step order in `clearListenState` matters.** The resolver-set removal needs the transcribe id, and the metadata clear destroys it. Read first, `SREM` second, clear third.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/listen.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/event.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/process.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`
- Test: `bin-ai-manager/pkg/aicallhandler/listen_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `bin-ai-manager/pkg/aicallhandler/listen_test.go`:

```go
// Test_stopListening_NeverTerminatesTheAIcall is a regression test for a naming
// hazard, not a hypothetical.
//
// "Stop listening" and "terminate the AIcall" are one word apart and worlds
// apart in effect: ProcessTerminate ends the AIcall itself (status terminated +
// activeflow service stop), which would kill the agent's entire Insight Q&A
// session. Every stop path must leave the panel working normally.
func Test_stopListening_NeverTerminatesTheAIcall(t *testing.T) {
	// Drive stopListening with a requesthandler mock that has NO
	// FlowV1ActiveflowServiceStop expectation and a dbhandler mock with no
	// UpdateStatus expectation -- gomock fails if either is called.
	//
	// Assert the two calls it SHOULD make, in order: the owned-transcribe stop,
	// then clearListenState.
}

// Test_clearListenState_StepOrder pins that the resolver-set removal happens
// BEFORE the metadata clear.
//
// The SREM needs the transcribe id, and the metadata clear destroys it. Doing
// them in the other order leaves a stale (transcribe_id, aicall_id) pairing that
// section 5.3 can still match, feeding segments to an AIcall that has stopped
// listening.
func Test_clearListenState_StepOrder(t *testing.T) {
	// Use gomock.InOrder to require ListenAIcallIDRemove before
	// AIcallUpdateNoTouchTMUpdate.
}

// Test_stopListenByCallID_ClearsEveryMatch pins the plural lookup.
//
// Two Cases open on one call each get their own AIcall (the active-reference
// unique key is per Case, not per customer), and BOTH must be cleared when the
// call hangs up. Clearing only the first leaves the second listening to a
// transcribe session that has stopped producing.
func Test_stopListenByCallID_ClearsEveryMatch(t *testing.T) {
	// AIcallList returns two rows; assert both get cleared.
}

// Test_stopListenByCallID_FinalFlush pins that the last words of a call are
// still evaluated.
//
// The debounce means the final lines before a hangup sit unevaluated in the
// pending buffer. This is the only chance to read them, and it deliberately
// bypasses the debounce lock -- there is no "next segment" coming.
func Test_stopListenByCallID_FinalFlush(t *testing.T) {
	// Non-empty pending buffer -> runListenTurnWithLines is reached with those
	// lines, WITHOUT a ListenTurnTryLock call. Empty buffer -> no turn.
}
```

- [ ] **Step 2: Run the tests to verify they fail, then implement**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run 'Test_stopListening|Test_clearListenState|Test_stopListenByCallID' -v
```
Expected: FAIL — undefined functions.

Append to `bin-ai-manager/pkg/aicallhandler/listen.go`:

```go
// stopListening stops this AIcall listening. It is EXACTLY two steps, in order,
// and nothing else:
//
//  1. If this AIcall owns the transcribe session, stop it.
//  2. Clear the listen state.
//
// IT NEVER CALLS ProcessTerminate. That function ends the AIcall ITSELF (status
// to terminated plus an activeflow service stop), which would kill the agent's
// entire Insight Q&A session. Stopping listening must leave the panel working
// normally -- the agent can still ask questions about a call that just ended,
// and that is often exactly when they do.
func (h *aicallHandler) stopListening(ctx context.Context, c *aicall.AIcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "stopListening",
		"aicall_id": c.ID,
	})

	if listenOwnsTranscribeFromMetadata(c) {
		transcribeID := listenTranscribeIDFromMetadata(c)
		if transcribeID != uuid.Nil {
			// HostID is fetched FRESH rather than read from a persisted column,
			// precisely because it is regenerated on every transcribe-manager
			// restart -- a stale one addresses a queue that no longer exists.
			tr, errGet := h.reqHandler.TranscribeV1TranscribeGet(ctx, transcribeID)
			if errGet != nil {
				log.Warnf("Could not get the transcribe to stop it. err: %v", errGet)
				promListenStopFailedTotal.Inc()
			} else if tr.Status == tmtranscribe.StatusProgressing {
				if _, errStop := h.reqHandler.TranscribeV1TranscribeStop(ctx, tr.HostID, tr.ID); errStop != nil {
					// NON-FATAL, and the fallback is stated rather than assumed:
					// if the owning pod restarted, its per-pod request queue no
					// longer exists and this times out. The session's audio
					// transport still ends when the call itself ends (hanging up
					// closes the Asterisk WebSocket feeding the STT stream), so
					// the failure mode is a slightly-longer-than-necessary STT
					// session, never a permanently orphaned one.
					log.Warnf("Could not stop the transcribe. err: %v", errStop)
					promListenStopFailedTotal.Inc()
				}
			}
		}
	}

	h.clearListenState(ctx, c)
}

// clearListenState removes every trace of this AIcall's listening state.
//
// STEP ORDER IS LOAD-BEARING. Step 2 needs the transcribe id and step 3 destroys
// it, so reading and using it must come first. Getting this backwards leaves a
// stale (transcribe_id, aicall_id) pairing that intake can still match, feeding
// segments to an AIcall that stopped listening.
func (h *aicallHandler) clearListenState(ctx context.Context, c *aicall.AIcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "clearListenState",
		"aicall_id": c.ID,
	})

	// 1. Read the transcribe id from the AIcall already in hand -- no extra
	//    fetch; every caller holds `c`.
	transcribeID := listenTranscribeIDFromMetadata(c)

	// 2. Remove ONLY this AIcall's resolver membership. Never DEL the key: a
	//    shared transcribe must stay resolvable for whichever AIcalls are still
	//    listening to it. Redis drops the key once the set empties.
	if transcribeID != uuid.Nil {
		if errRem := h.cache.ListenAIcallIDRemove(ctx, transcribeID, c.ID); errRem != nil {
			log.Warnf("Could not remove the listen resolver membership. err: %v", errRem)
		}
	}

	// The per-AIcall keys are never shared, so a plain delete is correct for
	// them. The turn-id set is deliberately NOT deleted: it is short-TTL and
	// self-expiring, and a tool call arriving late for an already-stopped turn
	// still correctly resolves as a listen turn -- which is exactly what it was.
	if errClear := h.cache.ListenStateClear(ctx, c.ID); errClear != nil {
		log.Warnf("Could not clear the listen state keys. err: %v", errClear)
	}

	// 3. Clear the DB bookkeeping LAST, since step 2 consumed the value it
	//    holds. AIcallUpdateNoTouchTMUpdate, not AIcallUpdate: listening stops
	//    on hangup, which is exactly when an agent is most likely to type a
	//    follow-up question, and a tm_update bump here would have Send's cooldown
	//    reject it.
	metadata := map[string]any{}
	for k, v := range c.Metadata {
		if k == aicall.MetaKeyListenTranscribeID || k == aicall.MetaKeyListenOwnsTranscribe {
			continue
		}
		metadata[k] = v
	}

	if errUpdate := h.db.AIcallUpdateNoTouchTMUpdate(ctx, c.ID, map[aicall.Field]any{
		aicall.FieldListenCallID: uuid.Nil,
		aicall.FieldMetadata:     metadata,
	}); errUpdate != nil {
		log.Warnf("Could not clear the listen state on the aicall row. err: %v", errUpdate)
	}
}

// stopListenByCallID stops every AIcall listening to the given call.
//
// PLURAL ON PURPOSE: the active-AIcall unique key is per Case, not per customer,
// so two Cases open on one call each get their own AIcall and both must be
// cleared.
//
// Before clearing, it runs one final flush turn per AIcall with a non-empty
// buffer. The debounce means the last few lines before a hangup are still
// sitting unevaluated, and this is the only chance to read them -- there is no
// next segment coming. It bypasses the debounce lock deliberately, which is why
// the turn body was extracted as runListenTurnWithLines.
func (h *aicallHandler) stopListenByCallID(ctx context.Context, callID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "stopListenByCallID",
		"call_id": callID,
	})

	listening, err := h.List(ctx, 10, "", map[aicall.Field]any{
		aicall.FieldReferenceType: aicall.ReferenceTypeContactCase,
		aicall.FieldListenCallID:  callID,
		aicall.FieldDeleted:       false,
	})
	if err != nil {
		log.Errorf("Could not list the listening aicalls. err: %v", err)
		return
	}

	for _, c := range listening {
		lines, errPop := h.cache.ListenPendingPopAll(ctx, c.ID)
		if errPop != nil {
			log.Warnf("Could not drain the pending buffer for the final flush. aicall_id: %s, err: %v", c.ID, errPop)
		} else if len(lines) > 0 {
			h.runListenTurnWithLines(ctx, c, lines)
		}

		h.stopListening(ctx, c)
	}
}
```

Add the metric to `metrics_listen.go`:

```go
	promListenStopFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_listen_stop_failed_total",
			Help:      "Total listen transcribe-stop RPCs that failed and fell back to the call-hangup-ends-the-audio-transport backstop.",
		},
	)
```

- [ ] **Step 3: Hook the hangup path**

In `bin-ai-manager/pkg/aicallhandler/event.go`, replace `EventCMCallHangup`:

```go
// EventCMCallHangup handles the call-manager's call_hangup event
func (h *aicallHandler) EventCMCallHangup(ctx context.Context, evt *cmcall.Call) {
	// Existing path, unchanged: the AIcall whose reference IS this call.
	if cc, err := h.GetByReferenceID(ctx, evt.ID); err == nil {
		_, _ = h.ProcessTerminate(ctx, cc.ID)
	}

	// New path: every contact_case AIcall LISTENING to this call.
	//
	// A second lookup is genuinely required, not a nicety. For an Insight
	// AIcall, ReferenceID is the CASE id, so GetByReferenceID can never find a
	// listening AIcall from a call id -- which is exactly why listen_call_id
	// exists as an indexed column.
	h.stopListenByCallID(ctx, evt.ID)
}
```

- [ ] **Step 4: Hook the terminate path**

In `bin-ai-manager/pkg/aicallhandler/process.go`, inside `ProcessTerminate`, after the AIcall is fetched and before the pipecatcall teardown:

```go
	// Release any listening state this AIcall holds. The agent closed the panel,
	// the session idled out, or the turn cap tripped -- either way the live call
	// may still be running, and an owned STT session must not be left behind
	// with its handle about to be forgotten.
	if tmp.ReferenceType == aicall.ReferenceTypeContactCase && tmp.ListenCallID != uuid.Nil {
		h.stopListening(ctx, tmp)
	}
```

This is the AIcall-is-ending path reusing the same helper; it is not a call *to* `ProcessTerminate` from `stopListening`, which would be a loop.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run 'Test_stopListening|Test_clearListenState|Test_stopListenByCallID' -v
```
Expected: PASS, every subtest.

- [ ] **Step 6: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass. Existing `EventCMCallHangup` and `ProcessTerminate` tests need an `AIcallList` expectation returning an empty slice — with no listening AIcalls, both new paths are no-ops.

- [ ] **Step 7: Commit**

Commit message body:

```
- bin-ai-manager: Add stopListening, clearListenState and stopListenByCallID
- bin-ai-manager: Stop listening for every AIcall watching a call that hung up, after a final flush turn
- bin-ai-manager: Release listening state when a contact_case AIcall terminates
```

Stage `bin-ai-manager/pkg/aicallhandler/`, then commit with the branch name as the title and the body above.

---

## Task 26: `bin-ai-manager` — verify the metric surface is complete

Every metric was added alongside the code that emits it. This task audits the set as a whole and pins the naming convention.

**The namespace is prepended by the Prometheus client library**, from `metricsNamespace` in `pkg/aicallhandler/main.go`. Writing `ai_manager_` into a `Name` string renders as `ai_manager_ai_manager_...`.

**Seven new metrics total, six of them here.** The six names below all live in `pkg/aicallhandler/metrics_listen.go` and match design §5.13's table exactly; the seventh, `aicall_foreign_pipecatcall_dropped_total`, lives in `pkg/messagehandler/metrics_foreign.go` and is audited separately by Step 3's grep rather than by Step 1's golden test (it is registered from a different package, so gathering it here would prove nothing about where it is emitted). **None of the seven names changed in design rev 15-23.** What rev 15-23 added were new *outcomes*, and every one of them is a `result` **label value** on `aicall_listen_start_total`, not a new family: `skipped_confbridge_not_ready` and `skipped_confbridge_error` (rev 11), and `skipped_start_locked` (rev 18-20). This test pins names, so it is unaffected — that is deliberate, and the reason label values are not pinned here is that they change far more often than families do. Task 20 records the design §11 item 16 decision behind `skipped_start_locked`; the acquire-error path meters `failed`, and the deferred-release error path is deliberately **not** metered at all (design §6, review round 17 finding B-6).

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/metrics_listen.go`
- Test: `bin-ai-manager/pkg/aicallhandler/metrics_listen_test.go` (create)

- [ ] **Step 1: Write the golden test**

Create `bin-ai-manager/pkg/aicallhandler/metrics_listen_test.go`:

```go
package aicallhandler

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// Test_listenMetricNames pins the full listen metric surface and its naming.
//
// Two things it guards. (1) The set is complete -- an outcome with no metric is
// an outcome nobody can alert on, and every one of these is a real operational
// signal. (2) No Name string carries the "ai_manager_" prefix itself: the
// Prometheus client library prepends the namespace, so a literal prefix renders
// as ai_manager_ai_manager_..., which silently breaks every dashboard query
// written against the documented name.
func Test_listenMetricNames(t *testing.T) {
	expected := map[string]bool{
		"ai_manager_aicall_listen_start_total":                    false,
		"ai_manager_aicall_listen_segment_total":                  false,
		"ai_manager_aicall_listen_turn_total":                     false,
		"ai_manager_aicall_listen_notify_total":                   false,
		"ai_manager_aicall_listen_stop_failed_total":              false,
		"ai_manager_aicall_listen_membership_check_failed_total":  false,
	}

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("could not gather metrics. err: %v", err)
	}

	for _, f := range families {
		name := f.GetName()
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
		if strings.HasPrefix(name, "ai_manager_ai_manager_") {
			t.Errorf("metric %q has a doubled namespace -- the Name field must not repeat the namespace prefix", name)
		}
	}

	for name, seen := range expected {
		if !seen {
			t.Errorf("metric %q is not registered", name)
		}
	}
}
```

A counter with no observations may not appear in `Gather()` output. If the test fails for that reason rather than a real omission, force each counter to register a zero sample first — for a `CounterVec`, call `.WithLabelValues(...)` on one label set; for a plain `Counter`, `.Add(0)`. Do that in the test, not in production code.

- [ ] **Step 2: Run the test**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go test ./pkg/aicallhandler/ -run Test_listenMetricNames -v
```
Expected: PASS if Tasks 17–25 added every metric. Any failure names the missing one — add it in `metrics_listen.go` and emit it from the path it describes.

- [ ] **Step 3: Confirm the messagehandler metric too**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  grep -rn "aicall_foreign_pipecatcall_dropped_total" pkg/messagehandler/
```
Expected: one hit in `metrics_foreign.go` plus its emit sites in `event.go`.

- [ ] **Step 4: Full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-ai-manager && \
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass.

- [ ] **Step 5: Commit**

Commit message body:

```
- bin-ai-manager: Add a golden test pinning the listen metric surface and its namespace convention
```

Stage `bin-ai-manager/pkg/aicallhandler/`, then commit with the branch name as the title and the body above.

---

## Task 27: `bin-openapi-manager` and `bin-api-manager` — public API surface

Three additions to public surface: **the `POST /service_agents/aicalls/{id}/listen` endpoint** (new in design rev 15, its path corrected in rev 16), the `notify_agent` tool name, and the `origin` field on a message.

**🚨 THE ENDPOINT GOES ON THE AGENT-FACING `/service_agents/*` SURFACE, NOT THE TOP-LEVEL ADMIN ONE. 🚨** This is design review round 13's BLOCKING-1, and getting it wrong makes the feature not work at all for its actual primary user. Design rev 15's first draft put it at the top-level `POST /v1/aicalls/{id}/listen`, mirroring `terminate` — but `AIcallTerminate` (`servicehandler/aicall.go`) gates on `amagent.PermissionCustomerAdmin|PermissionCustomerManager`, an **Admin-console** tier. The panel's own existing "Start" call, `ServiceAgentAIcallCreate`, is on the **Agent** surface, gated only on `amagent.PermissionAll`. `bin-api-manager/docs/auth.md` states the rule in the imperative: *"`square-talk` (and any other Agent-facing frontend) MUST call ONLY `/service_agents/*` paths — never the top-level `/<resource>` path directly, even if the top-level path's permission bitmask happens to allow Agent-level access,"* and *"Do NOT 'fix' a missing Agent-facing capability by relaxing the top-level endpoint's permission bitmask."* At the top-level path an ordinary agent in square-talk would get `ErrPermissionDenied` and **listening would never start in the feature's actual primary use case**.

**A top-level Admin-console `/v1/aicalls/{id}/listen` public route is deliberately NOT added.** Nothing in scope needs an admin-console caller for this action, and adding an unused surface is scope creep. If a genuine admin-console use case appears, it gets its own ticket.

**Do not confuse this with Task 20's internal route.** ai-manager's own RPC surface stays at `POST /v1/aicalls/{id}/listen`; it is the public, api-manager-facing path that is `/service_agents/aicalls/{id}/listen`. Two services, two routes, one shared trailing segment.

**The precedent to mirror is `POST /service_agents/contact_addresses/:id/claim`**, not `POST /v1/aicalls/{id}/terminate`: same surface, same id-scoped action-verb idiom, verb as the trailing path segment, `POST`, no request body.

**`listen_internal` gets documented too, not hidden.** Those rows go out through the same `ConvertWebhookMessage` path as any other message, so the value genuinely reaches a tenant's webhook payload. Leaving an on-the-wire value undocumented is worse than documenting it plainly as internal bookkeeping.

**Files:**
- Create: `bin-openapi-manager/openapi/paths/service_agents/aicalls/id_listen.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml`
- Modify: `bin-openapi-manager/openapi/paths/ais/main.yaml`
- Modify: `bin-openapi-manager/openapi/paths/ais/id.yaml`
- Modify: `bin-api-manager/server/service_agents_aicalls.go`
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go` and the `ServiceHandler` interface in `pkg/servicehandler/main.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_aicalls.go` + mock regen
- Regenerated: `bin-api-manager/gens/openapi_server/gen.go`

**⚠️ This task touches `bin-common-handler`**, like Tasks 3, 4 and 5 — so its verification is the full every-service sweep from the Verification convention section, not just the two services in this task's title.

- [ ] **Step 1: Add the `AIV1AIcallListen` RPC client**

In `bin-common-handler/pkg/requesthandler/ai_aicalls.go`, add alongside `AIV1AIcallTerminate`, whose shape it mirrors — `POST` to `/v1/aicalls/<id>/listen` with `ContentTypeNone` (no request body, same as `terminate`):

```go
AIV1AIcallListen(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error)
```

**Give it an explicit `10000` (10s) timeout rather than inheriting `requestTimeoutDefault` (3000ms).** This is design review round 13's MEDIUM-1, and the reasoning was corrected in rev 17 (round 14 MEDIUM-3), so use the *corrected* one: `ProcessListen` runs up to three **sequential** cross-service RPCs (`TranscribeV1TranscribeGet`, `ContactV1CaseGet`, `CallV1CallGet`), and **each hop can independently take up to its own default timeout** — so three hops can add up to roughly 3× a single hop's timeout worst-case, failing the *client's* request even when ai-manager's own precheck later succeeds. (The earlier justification, "none of the three is cache-first," was withdrawn: `CallV1CallGet` *is* cache-first. Do not reintroduce it if this value is ever revisited.) The per-call-override pattern is the one `TranscribeV1TranscribeStart` already uses for its own `5000`.

Regenerate the mock, then run the full every-service sweep before moving on — a `bin-common-handler` change that only compiles in two services is a broken deploy.

- [ ] **Step 2: Add the OpenAPI path**

Create `bin-openapi-manager/openapi/paths/service_agents/aicalls/id_listen.yaml`, mirroring `service_agents/contact_addresses/id_claim.yaml`'s shape: `POST`, id path parameter, **no request body**, `200` returning the existing `AIManagerAIcall` schema. Wire it into `openapi.yaml`'s paths exactly the way `id_claim.yaml` is wired, and regenerate.

Confirm the generated router picked it up — the registration should sit alongside the existing `PostServiceAgentsAicalls` one:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  grep -n "PostServiceAgentsAicalls" bin-api-manager/gens/openapi_server/gen.go | head
```
Expected: both `PostServiceAgentsAicalls` and `PostServiceAgentsAicallsIdListen`, the latter registered at `options.BaseURL+"/service_agents/aicalls/:id/listen"`.

- [ ] **Step 3: Add the api-manager handler and service method**

In `bin-api-manager/server/service_agents_aicalls.go`, add `PostServiceAgentsAicallsIdListen`, following the file's existing handlers.

In `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go`, add:

```go
ServiceAgentAIcallListen(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)
```

and its `ServiceHandler` interface entry in `pkg/servicehandler/main.go`. The body, in this exact order:

1. `a.IsAgent()` — else `ErrAuthenticationRequired`. Matches `ServiceAgentTranscribeList`'s and `ServiceAgentAIcallList`'s own first line.
2. `h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll)` — else `ErrPermissionDenied`. **`PermissionAll`, not the admin/manager bitmask** (see BLOCKING-1 above).
3. `tmp, err := h.aicallGet(ctx, id)` — **the private two-level helper, not `AIV1AIcallGet` directly** (design review round 14 finding LOW-2). `serviceagent_aicall.go` already wraps this exact fetch, and `bin-api-manager/CLAUDE.md`'s two-level handler pattern expects it reused rather than re-inlined.
4. **The ownership compare happens here, in the public method, not inside `aicallGet`** (design rev 18, review round 15 finding LOW-5): `tmp.CustomerID != a.CustomerID → ErrPermissionDenied`. `aicallGet` only fetches; the sibling `ServiceAgentAIcallGet` does the compare itself, and this method follows that same division of labour. Do not move the compare into the helper — that would change `ServiceAgentAIcallGet`'s behaviour too.
5. `h.reqHandler.AIV1AIcallListen(ctx, id)`, then `.ConvertWebhookMessage()`.

Test coverage for this layer (design §7 item 2): non-agent identity → `ErrAuthenticationRequired` with **zero** RPC calls; agent without `PermissionAll` → `ErrPermissionDenied` with **zero** `AIV1AIcallListen` calls (the direct regression test for BLOCKING-1); cross-customer AIcall (fetched, but `CustomerID != a.CustomerID`) → rejected **before** `AIV1AIcallListen` is called.

- [ ] **Step 4: Add `notify_agent` to the tool-name enum**

In `bin-openapi-manager/openapi/openapi.yaml`, find the tool-name enum (locate it by its `get_call_transcript` entry, not by line number) and append to both the `enum` list and the parallel `x-enum-varnames` list, keeping them index-aligned:

```yaml
        - notify_agent
```

```yaml
        - AIManagerToolNameNotifyAgent
```

Confirm alignment after editing — the two lists are positional:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  python3 -c "
import yaml,sys
d=yaml.safe_load(open('bin-openapi-manager/openapi/openapi.yaml'))
def walk(o):
    if isinstance(o,dict):
        if 'enum' in o and 'x-enum-varnames' in o and 'notify_agent' in (o.get('enum') or []):
            print('enum len', len(o['enum']), 'varnames len', len(o['x-enum-varnames']))
            print('last pair:', o['enum'][-1], o['x-enum-varnames'][-1])
        for v in o.values(): walk(v)
    elif isinstance(o,list):
        for v in o: walk(v)
walk(d)
"
```
Expected: equal lengths, and the last pair is `notify_agent` / `AIManagerToolNameNotifyAgent`.

- [ ] **Step 5: Add `origin` to the message schema**

In the same file, find the AI message schema (the one carrying `tool_calls` and `tool_call_id`) and add:

```yaml
        origin:
          type: string
          description: >-
            How this message came to exist, orthogonally to role. Empty for
            ordinary messages. "proactive" marks a note the Insight AI sent on
            its own initiative while monitoring a live call, rather than in
            answer to anything. "listen_internal" marks internal bookkeeping
            rows produced while monitoring; do not depend on their presence or
            meaning.
          enum:
            - ""
            - proactive
            - listen_internal
```

Match the surrounding indentation and the file's existing style for enums with descriptions.

- [ ] **Step 6: Update the Insight tool-list prose**

`bin-openapi-manager/openapi/paths/ais/main.yaml` and `paths/ais/id.yaml` both carry a prose list of Insight-allowed tools ending in `get_call_transcript);`. Add `notify_agent` to both, with a brief clause noting it is the one write tool and only usable while monitoring a live call.

- [ ] **Step 7: Regenerate**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-openapi-manager && \
  go generate ./... && go test ./... && \
cd ../bin-api-manager && \
  go generate ./... && go test ./...
```
Expected: both regenerate cleanly and pass.

- [ ] **Step 8: Confirm the generated Go carries the new values**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  grep -rn "NotifyAgent" bin-api-manager/ --include='*.gen.go' | head -3 && \
  grep -rn "PostServiceAgentsAicallsIdListen" bin-api-manager/ | head -3
```
Expected: at least one hit for each. If the generated files use a different suffix, adjust the glob — the point is that regeneration actually picked both changes up.

- [ ] **Step 9: Confirm the endpoint did NOT land on the top-level surface**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  grep -rn "aicalls/:id/listen\|aicalls/{id}/listen" bin-openapi-manager/openapi/ bin-api-manager/gens/ | grep -v service_agents
```
Expected: **no output.** Any hit outside `service_agents` means the route was added to the Admin-console surface too — that is design review round 13's BLOCKING-1 reintroduced, and an ordinary agent would be denied. Remove it.

- [ ] **Step 10: Full verification**

`bin-common-handler` was touched in Step 1, so the sweep is every service, not just these two:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
for d in bin-*-manager voip-*-proxy bin-common-handler; do
  [ -f "$d/go.mod" ] || continue
  echo "=== $d ==="
  ( cd "$d" && go mod tidy && go mod vendor && go generate ./... && go test ./... ) || { echo "FAILED: $d"; break; }
done && \
for d in bin-openapi-manager bin-api-manager bin-common-handler; do
  echo "=== lint $d ==="
  ( cd "$d" && golangci-lint run -v --timeout 5m ) || break
done
```
Expected: no `FAILED:` line, and all three lint clean.

- [ ] **Step 11: Commit**

Commit message body:

```
- bin-common-handler: Add the AIV1AIcallListen RPC client with an explicit 10s timeout
- bin-openapi-manager: Add POST /service_agents/aicalls/{id}/listen on the agent-facing surface
- bin-openapi-manager: Add notify_agent to the tool-name enum and the Insight tool-list docs
- bin-openapi-manager: Add the message origin field with its proactive and listen_internal values
- bin-api-manager: Add ServiceAgentAIcallListen and its generated route, regenerating the OpenAPI types
```

Stage `bin-common-handler/`, `bin-openapi-manager/`, `bin-api-manager/` (generated files included), then commit with the branch name as the title and the body above.

---

## Task 28: `bin-api-manager` — RST documentation

The RST docs are the single source of truth for how the platform works, for customers and integrators. A user-visible change without a docs update actively misleads.

**Always clean-rebuild.** Incremental Sphinx builds miss cross-page references.

**Three things to document, not two** (design §5.10.2): the `origin` field, the `notify_agent` tool, and — new with design rev 15/16 — **the `POST /service_agents/aicalls/{id}/listen` endpoint itself**, which is genuinely new public API surface.

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_struct_message.rst`
- Modify: `bin-api-manager/docsdev/source/ai_struct_tool.rst`
- Modify: `bin-api-manager/docsdev/source/ai_overview.rst`
- Modify: `bin-api-manager/docsdev/build/**` (regenerated)

- [ ] **Step 1: Document the `origin` field**

In `bin-api-manager/docsdev/source/ai_struct_message.rst`, add `"origin": "<string>"` to the `Message` JSON block (after `direction`), add a bullet describing it, and add a new value table after the existing Direction table:

```rst
* ``origin`` (enum string): How this message came to exist, orthogonally to ``role``. See :ref:`Origin <ai-struct-message-origin>`.
```

```rst
.. _ai-struct-message-origin:

Origin
------

All possible values for the ``origin`` field:

=============== ===========
Origin          Description
=============== ===========
(empty)         An ordinary message: an answer, or a question
proactive       A note the Insight Assistant sent on its own initiative while monitoring a live call, rather than in answer to anything
listen_internal Internal bookkeeping produced while monitoring a live call. Do not depend on the presence or meaning of these messages
=============== ===========
```

**Compare against `WebhookMessage`, not the internal struct.** `Origin` was added to `WebhookMessage` and `ConvertWebhookMessage` in Task 7, so it belongs here. `AIcall.ListenCallID` was deliberately kept off the webhook, so it does **not** get an RST entry.

- [ ] **Step 2: Document the `notify_agent` tool**

In `bin-api-manager/docsdev/source/ai_struct_tool.rst`, add a `notify_agent` entry alongside the other Insight tools, matching the file's existing per-tool format. State plainly that it is the one Insight tool that writes, that its only effect is a message in the AIcall's own thread, and that it is only usable while the assistant is monitoring a live call.

- [ ] **Step 3: Add the feature overview section**

In `bin-api-manager/docsdev/source/ai_overview.rst`, add a subsection near the existing Insight Assistant material:

```rst
Insight Assistant: live call listening
--------------------------------------

When an agent opens a Case whose linked call is still in progress, the Insight
Assistant can follow that call's live transcript and speak up on its own if the
situation warrants it, instead of waiting to be asked.

**What triggers a proactive note is entirely up to you.** It is defined in the
AI's ``init_prompt``, the same field you already edit — for example: *"if the
customer mentions cancellation, a compliance keyword, or requests something
requiring approval, call notify_agent with a short actionable note; otherwise
say nothing."* There is no separate rule engine and no extra configuration.

Proactive notes arrive in the same Case Insight Assistant thread the agent is
already reading, over the same delivery path as any other message, and are
marked ``origin: "proactive"`` so they can be told apart from an answer to a
question.

Listening stops automatically when the call ends. Saying nothing is the normal
outcome of most checks — the assistant is expected to stay quiet unless your
instructions genuinely call for an alert.

Starting to listen
^^^^^^^^^^^^^^^^^^

Listening is triggered explicitly, by a single call:

.. code::

    POST /service_agents/aicalls/<aicall-id>/listen

There is no request body. The response is the current AI call
(:ref:`AIcall <ai-struct-aicall>`), returned immediately — it deliberately
carries no listening-status field, so there is nothing to poll on it. The call
is safe to repeat: opening the same Case panel again is free.

Listening does **not** start automatically when an AI call is created. The two
are separate actions on purpose, so either can be used without the other.
```

**Do not describe listening as automatic on AIcall creation** (design §5.10.2, corrected in rev 15/17). That was true of an earlier design and is not true now, and a doc that says so sends integrators looking for a switch that does not exist.

- [ ] **Step 3a: Document the endpoint itself**

Add the endpoint's own doc entry — method, path, **empty request body**, 200 response shape (the existing `AIManagerAIcall` struct, already documented elsewhere; **no new `*_struct_*.rst` file**, since this endpoint does not introduce a new shape).

**Follow whichever existing pattern documents `POST /service_agents/contact_addresses/:id/claim` today** — same directory, same doc generation path. **Not** `POST /v1/aicalls/{id}/terminate`'s pattern: `terminate` is on the Admin-console surface this endpoint deliberately is not (Task 27's BLOCKING-1 note). Locate the `claim` docs first and mirror them:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  grep -rn "contact_addresses" bin-api-manager/docsdev/source/ | grep -i claim | head
```

If `claim` turns out to have no dedicated endpoint page of its own (its OpenAPI entry may be the only place it is described), say so explicitly rather than inventing a new documentation shape for this one endpoint — the "Starting to listen" subsection added in Step 3 then carries the whole endpoint doc, and that is sufficient.

- [ ] **Step 4: Clean-rebuild the HTML**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen/bin-api-manager/docsdev && \
  rm -rf build && python3 -m sphinx -M html source build
```
Expected: `build succeeded`. Warnings about pre-existing pages are acceptable; warnings naming the three files just edited are not — fix those.

- [ ] **Step 5: Force-add the build output**

The root `.gitignore` excludes `build/`, but this build output is tracked and must stay in sync with its sources:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git add bin-api-manager/docsdev/source/ && \
  git add -f bin-api-manager/docsdev/build/
```

- [ ] **Step 6: Verify the rendered output actually changed**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git diff --cached --stat -- bin-api-manager/docsdev/build/ | tail -3 && \
  grep -rl "notify_agent" bin-api-manager/docsdev/build/html/ | head -3 && \
  grep -rl "service_agents/aicalls" bin-api-manager/docsdev/build/html/ | head -3
```
Expected: staged build changes, at least one built HTML page mentioning `notify_agent`, and at least one mentioning the listen endpoint's path.

- [ ] **Step 7: Commit**

Commit message body:

```
- bin-api-manager: Document the message origin field and its values in the RST struct docs
- bin-api-manager: Document the notify_agent tool and the Insight live-call-listening feature
- bin-api-manager: Document the POST /service_agents/aicalls/{id}/listen trigger endpoint
```

RST source and built HTML go in the same commit. Commit with the branch name as the title and the body above.

---

## Task 29: `bin-ai-manager` — service documentation

`scripts/check-service-docs.sh` warns when source that drives doc content changes without a matching docs update. This task clears those warnings and, more importantly, records the operational surface an on-call engineer needs.

**Files:**
- Modify: `bin-ai-manager/docs/architecture.md`
- Modify: `bin-ai-manager/docs/domain.md`
- Modify: `bin-ai-manager/docs/operations.md`

- [ ] **Step 1: Re-extract the mechanical sections**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  bash docs/reference/extractor.sh bin-ai-manager
```
Then reconcile the output into the three files. If the extractor writes them directly, review its diff rather than re-typing.

- [ ] **Step 2: Update `docs/architecture.md`**

Add `transcribe-manager.transcript.*.created` to the events section, with a one-line note that its handler is intentionally cheap because it fires for every final STT result platform-wide, not only for calls being listened to. Add both new routing-table entries: the `GET /v1/aicalls/<aicall-id>?skip_cache=true` variant, and **`POST /v1/aicalls/<aicall-id>/listen`** (Task 20 should already have added the latter in the same commit that added the route; confirm it is there rather than adding it twice).

- [ ] **Step 3: Update `docs/domain.md`**

Add `AIcall.ListenCallID` and the two metadata keys (`listen_transcribe_id`, `listen_owns_transcribe`) to the AIcall entity, and `Message.Origin` with its three values to the Message entity. For `Origin`, say which values are user-visible (`proactive`) and which are internal bookkeeping (`listen_internal`).

- [ ] **Step 4: Update `docs/operations.md`**

Add **all thirteen** config flags with their defaults (Task 10's list — including the two lock flags, and both ordering invariants between them), and all **seven** new metrics: the six in `pkg/aicallhandler/metrics_listen.go` plus `aicall_foreign_pipecatcall_dropped_total`, which lives separately in `pkg/messagehandler/metrics_foreign.go`. Then add a short runbook section, because these are the questions an on-call engineer will actually have:

```markdown
### Insight AI live call listening

Kill switch: `aicall_listen_enabled` / `AICALL_LISTEN_ENABLED`, default `false`.

**How listening starts.** Explicitly, by `POST /service_agents/aicalls/<aicall-id>/listen`
(routed internally to ai-manager's `POST /v1/aicalls/<aicall-id>/listen`). It is
**not** a side effect of creating or reusing the Q&A AIcall — creating an AIcall
never starts listening. The panels make the two calls in sequence when the Case
panel opens, and the second is fire-and-forget: its response carries no
listening-status field, so "did listening actually start?" is answered by the
metrics below, not by the API.

**Turning it off mid-call.** A rollback takes effect on an in-flight session at
its next *evaluated turn*, and turns are triggered by transcript segments, not by
a timer — so for an active conversation that is typically within one
`aicall_listen_evaluate_interval_seconds` (default 20s), but a call that has gone
quiet may not stop until it ends. Call hangup is the guaranteed backstop, and it
is independent of the flag.

**What the flag does NOT gate.** Two changes shipped with this feature are
general fixes and are active regardless: the two-fetch LLM context assembly
(which guarantees an AIcall's system prompt is never evicted), and the
foreign-pipecatcall guard on `contact_case` bot-LLM messages (which also drops
genuinely stale replies that used to be persisted silently). Expect
`aicall_foreign_pipecatcall_dropped_total` to become non-zero and Insight answer
*shape* to change slightly the moment the code deploys, independent of the flag.

**What to watch:**

| Signal | Reading |
|---|---|
| `aicall_listen_turn_total{result="skipped_locked"}` vs `{result="ran"}` | How much LLM spend the debounce is saving. Near-zero `skipped_locked` means the interval is too short for the traffic |
| `aicall_listen_turn_total{result="skipped_cap"}` | Calls hitting the hard turn cap. A rising rate means the cap or the interval needs revisiting |
| `aicall_listen_notify_total` | Proactive notes actually delivered. Zero with non-zero `ran` means prompts are not triggering — a prompt problem, not a system one |
| `aicall_listen_membership_check_failed_total` | Should be ~0. Sustained non-zero means Redis is unhealthy, not that anything listen-specific is wrong |
| `aicall_listen_stop_failed_total` | Stop RPCs that missed their pod. Tolerated — the audio transport ends with the call regardless — but a high rate suggests transcribe-manager instability |
| `aicall_listen_start_total{result="skipped_confbridge_not_ready"}` | The confbridge never settled to a live 2-party bridge within the wait budget. Note this **cannot** distinguish a slow ring from a genuinely non-2-party topology, and repeated panel re-opens on one still-ringing call inflate it. A sustained rate means `aicall_listen_confbridge_ready_max_wait_seconds` is likely too short for real ring times |
| `aicall_listen_start_total{result="skipped_start_locked"}` | A second concurrent start attempt for the *same* AIcall found the lock held and stood down. Expected in small numbers (an agent re-opening a panel during a long ring); a sustained high rate means heavy concurrent re-open pressure, not a fault |

**Redis dependency.** Listening degrades to today's reactive-only behaviour if
Redis is unavailable; Insight Q&A keeps working. A Redis flush silently stops
listening for in-flight calls until the panel is reopened, which repopulates the
state. This is deliberate: there is no DB fallback on a resolver miss, because
that would put a query on a platform-wide hot path.

**The `ai:listen:startlock:<aicall_id>` key.** Held only for the duration of one
listen-start sequence, released by the goroutine that took it via a
token-checked compare-and-delete. A goroutine that genuinely crashes (pod loss)
leaves it to expire on its own `aicall_listen_start_lock_ttl_seconds` — for that
one AIcall, further start attempts stand down as `skipped_start_locked` until
then, and the next panel open after expiry works normally. Do not delete this key
by hand to "unstick" a call: if a live goroutine still holds it, doing so
reintroduces exactly the double-writer race the lock exists to prevent.
```

- [ ] **Step 5: Confirm the docs hook is satisfied**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  bash scripts/check-service-docs.sh 2>&1 | head -20
```
Expected: no warning naming `bin-ai-manager`. The hook is advisory, but an unaddressed warning here means a doc genuinely drifted.

- [ ] **Step 6: Commit**

Commit message body:

```
- bin-ai-manager: Document the transcript_created subscription and the skip_cache aicall route
- bin-ai-manager: Document ListenCallID, the listen metadata keys and Message.Origin
- bin-ai-manager: Document the listen config flags, metrics and an operational runbook
```

Stage `bin-ai-manager/docs/`, then commit with the branch name as the title and the body above.

---

## Task 30: Phase A — final verification and the `monorepo` PR

- [ ] **Step 1: Full-monorepo verification**

`bin-common-handler` was touched, so every service must pass:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
for d in bin-*-manager voip-*-proxy bin-common-handler; do
  [ -f "$d/go.mod" ] || continue
  echo "=== $d ==="
  ( cd "$d" && go mod tidy && go mod vendor && go generate ./... && go test ./... ) || { echo "FAILED: $d"; break; }
done
```
Expected: no `FAILED:` line.

- [ ] **Step 2: Lint every service this feature touched**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
for d in bin-ai-manager bin-common-handler bin-pipecat-manager bin-contact-manager bin-customer-manager bin-api-manager bin-openapi-manager; do
  echo "=== $d ==="
  ( cd "$d" && golangci-lint run -v --timeout 5m ) || break
done
```
Expected: all clean.

- [ ] **Step 3: Confirm no vendor directory got staged**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git diff --cached --name-only | grep "/vendor/" | head
```
Expected: no output. If there is any, unstage it — `vendor/` is gitignored and Dockerfiles regenerate it.

- [ ] **Step 4: Confirm no fabricated ticket number anywhere**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git log origin/main..HEAD --format='%s%n%b' | grep -E "VOIP-[0-9]+" ; \
  git diff origin/main..HEAD | grep -E "^\+.*VOIP-[0-9]{4}" | grep -v "VOIP-1234\|VOIP-1253\|VOIP-1404\|VOIP-1405\|VOIP-1406\|VOIP-1422\|VOIP-1257\|VOIP-1453" | head
```
Expected: no output from the first command. The second may show references to genuinely pre-existing tickets in comments you copied; anything else is a fabricated number and must be removed. This work has **no** Jira ticket.

- [ ] **Step 5: Pull the latest main and check for conflicts**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git fetch origin main && \
  echo "--- conflicts ---" && \
  git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" ; \
  echo "--- new on main ---" && \
  git log --oneline HEAD..origin/main
```
If conflicts exist: rebase, resolve, and **re-run Steps 1 and 2 in full** before proceeding. If `main` has moved at all, re-run Step 1 regardless — a clean merge is not a passing build.

- [ ] **Step 6: Review the whole diff against the design**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git diff origin/main...HEAD --stat
```

Confirm the touched services match the design's impacted-file list: `bin-ai-manager`, `bin-common-handler`, `bin-pipecat-manager`, `bin-contact-manager`, `bin-customer-manager`, `bin-dbscheme-manager`, `bin-openapi-manager`, `bin-api-manager`, plus `docs/plans/`. Anything else is scope creep — check why it is there.

- [ ] **Step 7: Push and open the PR**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git push -u origin NOJIRA-Insight-AI-realtime-listen
```

Then open the PR with title `NOJIRA-Insight-AI-realtime-listen` and this body. No markdown headers, no test-plan section, no AI attribution:

```
Let the Case Insight Assistant follow a live call's transcript in real time and
proactively push a short note to the agent's panel when the customer-configured
conditions are met, without polluting the Q&A thread, evicting the LLM system
prompt, or scaling LLM cost with speech volume.

DEPLOY ORDER: the two Alembic migrations MUST be applied before this code
reaches any pod. commondatabasehandler.GetDBFields builds every SELECT column
list by reflecting the Go struct, so the moment Message.Origin and
AIcall.ListenCallID exist, every message and aicall query selects those columns.
Code landing first is a hard Unknown column outage across bin-ai-manager, not a
soft degradation.

Ships dark: aicall_listen_enabled defaults to false.

- bin-customer-manager: Add IDAIManagerListen system customer id for Insight AI call listening
- bin-contact-manager: Add kase.ReferenceTypeCall constant for the call-created Case reference type
- bin-common-handler: Add databasehandler.NotEq filter wrapper for SQL not-equal comparisons
- bin-common-handler: Add AIV1AIcallGetSkipCache RPC client for cache-bypassing aicall reads
- bin-common-handler: Add pipecatcallID parameter to AIV1AIcallToolExecute
- bin-common-handler: Add the AIV1AIcallListen RPC client with an explicit 10s timeout
- bin-pipecat-manager: Forward the invoking pipecatcall id on tool_execute
- bin-dbscheme-manager: Add ai_aicalls.listen_call_id column with its index, and ai_messages.origin
- bin-ai-manager: Add Message.Origin and AIcall.ListenCallID with their field and filter plumbing
- bin-ai-manager: Add the WithOrigin message create option and thirteen aicall_listen_* config flags
- bin-ai-manager: Add listen Redis primitives to cachehandler and wire the dependency into aicallHandler
- bin-ai-manager: Add a query-tolerant aicall GET route serving skip_cache=true
- bin-ai-manager: Add dbhandler AIcallGetSkipCache and AIcallUpdateNoTouchTMUpdate
- bin-ai-manager: Fetch system prompt rows independently of the capped replay window and exclude listen_internal rows
- bin-ai-manager: Register the notify_agent tool as the one sanctioned Insight write
- bin-ai-manager: Resolve listen-turn membership in ToolHandle and tag listen-internal rows
- bin-ai-manager: Implement toolHandleNotifyAgent, rejecting calls outside a listen turn
- bin-ai-manager: Drop contact_case pipecat messages from a foreign pipecatcall, confirming against the database first
- bin-ai-manager: Add ProcessListen and its POST /v1/aicalls/<aicall-id>/listen route as the explicit listen trigger
- bin-ai-manager: Add the bounded confbridge-readiness retry, the per-AIcall start lock, the transcript intake path, RunListenTurn and every stop path
- bin-ai-manager: Bind transcribe-manager.transcript.*.created and replace the commented-out per-segment handler
- bin-ai-manager: Add listen metrics, service docs and an operational runbook
- bin-openapi-manager: Add POST /service_agents/aicalls/{id}/listen, notify_agent in the tool-name enum, and the message origin field
- bin-api-manager: Add ServiceAgentAIcallListen, regenerate OpenAPI types, and document the listen endpoint, origin and notify_agent in RST
```

**Do not merge.** Wait for explicit authorization. When it comes, use `gh pr merge <n> --squash --delete-branch`.

---

## Task 31: `monorepo-javascript` — create the worktree and update square-admin

Phase B. A separate repository, therefore a separate PR — this is a structural necessity, not a discretionary split.

**Three changes here, not two** (design §5.10.1, §5.10.1a): the tool-noise render filter, the `origin=proactive` treatment, and — new with design rev 15/16 — **firing the `listen` call explicitly, as a second call after `Start` resolves.**

**Phase B now depends on Phase A being deployed.** Under the old design the panel called nothing new and this PR was safe to deploy in any order. It is not any more: the panel calls an endpoint that does not exist until the backend ships. The call is fire-and-forget and its failure must not block the panel, so a wrong order degrades rather than breaks — but say so in the PR body (Task 33) rather than repeating the old "safe to deploy at any time" claim.

**Files:**
- Modify: `square-admin/src/views/contacts/CaseInsightAssistantPanel.js`
- Modify: whichever module holds the aicall API wrappers this panel already imports (the `Start` call's own home) — one new wrapper for the `listen` call
- Test: `square-admin/src/views/contacts/CaseInsightAssistantPanel.test.js` (create)

- [ ] **Step 1: Create the worktree**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript && \
  git fetch origin main && \
  git worktree add /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen -b NOJIRA-Insight-AI-realtime-listen origin/main && \
  cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git branch --show-current
```
Expected: `NOJIRA-Insight-AI-realtime-listen`.

- [ ] **Step 2: Write the failing test**

square-admin uses `react-scripts test` (Jest). Create `square-admin/src/views/contacts/CaseInsightAssistantPanel.test.js`:

```js
import { buildTimelineItems } from './CaseInsightAssistantPanel'

// The panel renders whatever the aimessages endpoint returns, and every tool
// call the assistant makes produces TWO rows nobody wants to see: an
// empty-content assistant row carrying the tool_calls array, and a tool row
// holding raw JSON. That has always been true for Insight Q&A; live call
// listening makes it far more visible, because it is the first thing that can
// trigger a tool call without the agent having asked for anything -- so the
// noise now appears unprompted, mid-call.
//
// This is a CLIENT-SIDE RENDER FILTER ONLY. It does not touch webhook delivery:
// a tenant's own automation still receives every aimessage_created event exactly
// as before.
describe('buildTimelineItems tool-noise filter', () => {
  const at = (s) => ({ tm_create: s })

  it('hides the empty-content assistant row that carries tool_calls', () => {
    const items = buildTimelineItems(
      [{ id: 'm1', role: 'assistant', content: '', tool_calls: [{ id: 't1' }], ...at('2026-09-04T00:00:00Z') }],
      'UTC',
    )
    expect(items.filter((i) => i.type === 'message')).toHaveLength(0)
  })

  it('hides the tool row holding the raw JSON result', () => {
    const items = buildTimelineItems(
      [{ id: 'm2', role: 'tool', content: '{"result":"success"}', ...at('2026-09-04T00:00:00Z') }],
      'UTC',
    )
    expect(items.filter((i) => i.type === 'message')).toHaveLength(0)
  })

  it('keeps a normal assistant answer', () => {
    const items = buildTimelineItems(
      [{ id: 'm3', role: 'assistant', content: 'Here is the answer.', ...at('2026-09-04T00:00:00Z') }],
      'UTC',
    )
    expect(items.filter((i) => i.type === 'message')).toHaveLength(1)
  })

  it('keeps an assistant row that has both content and tool_calls', () => {
    // Only the EMPTY-content carrier row is noise. A row with real content is
    // something the agent should read, whatever else it carries.
    const items = buildTimelineItems(
      [{ id: 'm4', role: 'assistant', content: 'Let me look that up.', tool_calls: [{ id: 't2' }], ...at('2026-09-04T00:00:00Z') }],
      'UTC',
    )
    expect(items.filter((i) => i.type === 'message')).toHaveLength(1)
  })

  it('keeps a proactive notification', () => {
    const items = buildTimelineItems(
      [{ id: 'm5', role: 'assistant', content: 'Customer mentioned cancelling.', origin: 'proactive', ...at('2026-09-04T00:00:00Z') }],
      'UTC',
    )
    expect(items.filter((i) => i.type === 'message')).toHaveLength(1)
  })
})
```

`buildTimelineItems` is currently module-private. Export it (`export function buildTimelineItems`) — the panel's default export is unaffected.

- [ ] **Step 3: Run the test to verify it fails**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen/square-admin && \
  npx react-scripts test --watchAll=false --testPathPattern=CaseInsightAssistantPanel
```
Expected: FAIL on the two hiding cases — every message currently becomes an item.

- [ ] **Step 4: Add the filter**

In `square-admin/src/views/contacts/CaseInsightAssistantPanel.js`, add above `buildTimelineItems`:

```js
// Rows produced by the assistant's tool use that the agent should never see:
// the empty-content assistant row that merely carries the tool_calls array, and
// the tool row holding the raw JSON result. Both have always been rendered;
// live call listening makes them far more visible because it is the first thing
// that can trigger a tool call without the agent having asked for anything.
//
// Field names are snake_case (tool_calls, not toolCalls) -- that is the actual
// wire shape, matching msg.role / msg.content / msg.tm_create alongside it. A
// camelCase spelling here would simply never match and the filter would
// silently do nothing.
//
// Render-only. Webhook delivery is untouched: a tenant's own automation still
// receives every one of these as an aimessage_created event, exactly as before.
function isToolNoise(msg) {
  if (msg.role === 'tool') return true
  return msg.role === 'assistant' && !msg.content && msg.tool_calls?.length > 0
}
```

and skip them at the top of the loop in `buildTimelineItems`:

```js
  for (const msg of messages) {
    if (isToolNoise(msg)) continue

```

Placing the skip *before* the session- and date-separator logic is deliberate: a hidden message must not emit a separator either, or the thread grows dividers with nothing under them.

- [ ] **Step 5: Add the proactive treatment**

Replace `MessageBubble`:

```js
// A proactive note gets a visually distinct surface and an accessible label, so
// it is never mistaken for an answer to something the agent asked. `origin`
// comes straight off the wire (WebhookMessage.origin) and is absent on every
// message written before this feature, so the fallback is the ordinary
// assistant styling.
const MessageBubble = ({ msg, formatTimestamp }) => {
  const isUser = msg.role === 'user'
  const isProactive = msg.origin === 'proactive'

  if (isProactive) {
    return (
      <div
        className="rounded-md border border-primary/30 bg-primary/5 p-3 text-sm mr-8"
        role="note"
        aria-label="Proactive insight"
      >
        <div className="mb-1 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-primary">
          <Sparkles className="h-3 w-3" aria-hidden="true" />
          Proactive insight
        </div>
        <p className="whitespace-pre-wrap">{msg.content}</p>
        <span className={CAPTION}>{formatTimestamp(msg.tm_create)}</span>
      </div>
    )
  }

  return (
    <div className={`rounded-md p-3 text-sm ${isUser ? 'bg-primary/10 ml-8' : 'bg-muted mr-8'}`}>
      <p className="whitespace-pre-wrap">{msg.content}</p>
      <span className={CAPTION}>{formatTimestamp(msg.tm_create)}</span>
    </div>
  )
}
```

`Sparkles` is already imported in this file (`SessionSeparator` uses it). Confirm before relying on it:

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen/square-admin && \
  grep -n "Sparkles" src/views/contacts/CaseInsightAssistantPanel.js | head -2
```

- [ ] **Step 5a: Fire the explicit `listen` call after `Start` resolves**

**This is the trigger.** Design rev 15 removed the backend's implicit `Start` hook, so if the panel does not make this call, **listening never starts at all** — no error, no metric, nothing to notice. It is the single most load-bearing line in Phase B.

The panel now makes **two** sequential calls when it opens (design §5.10.1a):

1. the existing one that creates/reuses the Q&A AIcall (`Start`, already `POST /service_agents/aicalls`), then
2. `POST /service_agents/aicalls/{id}/listen`, using the AIcall id `Start`'s response returned.

Add the API wrapper alongside the existing aicall wrappers, then call it in the **same effect that already calls `Start`**, immediately after `Start` resolves:

```js
  // Design rev 15 made listening an explicit, separately-callable action
  // rather than a side effect of creating the Q&A AIcall. Two calls, in
  // order, because the second needs the first's id.
  //
  // FIRE-AND-FORGET on purpose: the response is the current AIcall and
  // carries NO listening-status field to branch on, so there is nothing to
  // await for the UI's sake. A failed or slow response must never block
  // rendering the panel, and a repeated call (a fast double-open) is free --
  // the backend's own idempotency check makes it a no-op.
  const aicall = await createAicallAPI(/* ...existing args... */)
  listenAicallAPI(aicall.id).catch(() => {
    // Deliberately swallowed. Listening is an enhancement; the panel works
    // without it, and the backend records its own outcome either way.
  })
```

**Not** the top-level `/v1/aicalls/{id}/listen` path — that one is Admin-console-gated and would reject an ordinary agent (Task 27's BLOCKING-1 note). If the wrapper module has a base path helper for `/service_agents/*`, use it.

Add a test alongside the render-filter tests above: mock the `Start` wrapper to resolve with a known id, render the panel, and assert the `listen` wrapper was called **once, with that id**. Add a second case where the `listen` wrapper rejects, asserting the panel still renders its messages — that is the fire-and-forget contract, and it is exactly the kind of thing an unhandled rejection would otherwise turn into a broken panel.

- [ ] **Step 6: Run the tests and lint**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen/square-admin && \
  npx react-scripts test --watchAll=false --testPathPattern=CaseInsightAssistantPanel && \
  npm run lint
```
Expected: all cases pass (the five render-filter ones plus Step 5a's two), lint clean.

- [ ] **Step 7: Commit**

Commit message body:

```
- square-admin: Fire POST /service_agents/aicalls/{id}/listen after Start resolves, as a separate fire-and-forget call
- square-admin: Hide tool-call and tool-result noise rows from the Case Insight Assistant thread
- square-admin: Render origin=proactive notifications with a distinct surface and an accessible label
```

Stage `square-admin/src/views/contacts/` and the API wrapper module, then commit with the branch name as the title and the body above.

---

## Task 32: `monorepo-javascript` — update square-talk

square-talk uses Vite/Vitest and already has a panel test file to extend. It keeps its 2s poll — no transport change; the note simply appears on the next poll.

**Same three changes as Task 31**, including the explicit `listen` call after `Start` resolves (design §5.10.1a) — same pattern, same call site as this panel's own `Start` invocation. **This is the panel most affected by getting it wrong**: square-talk is the Agent-facing app, and it is precisely the ordinary agent (holding neither admin nor manager permission) that design review round 13's BLOCKING-1 was about. Use `/service_agents/aicalls/{id}/listen`, never the top-level path.

**No new WebSocket/poll subscription is needed for this call itself** — it is a one-shot trigger, not a data source. Any resulting proactive message still arrives over this panel's existing 2s poll, unchanged.

**Files:**
- Modify: `square-talk/src/features/cases/CaseInsightAssistantPanel.jsx`
- Modify: `square-talk/src/api/services/aicalls` (the module this panel already mocks in its tests) — one new wrapper for the `listen` call
- Modify: `square-talk/src/features/cases/CaseInsightAssistantPanel.test.jsx`

- [ ] **Step 1: Write the failing tests**

Append to `square-talk/src/features/cases/CaseInsightAssistantPanel.test.jsx`, inside the existing `describe`:

```jsx
  it('hides tool-call and tool-result noise rows and renders a proactive note distinctly', async () => {
    // Same two noise rows square-admin hides, same reasoning: they have always
    // been rendered, and live call listening makes them appear unprompted
    // mid-call rather than only after a question the agent typed.
    createAicallAPI.mockResolvedValue({ id: 'aicall-1' });
    getAicallsAPI.mockResolvedValue([{ id: 'aicall-1', status: 'progressing' }]);
    getAimessagesAPI.mockResolvedValue([
      { id: 'm4', role: 'assistant', content: 'Customer mentioned cancelling.', origin: 'proactive', tm_create: '2026-09-04T00:00:03Z' },
      { id: 'm3', role: 'tool', content: '{"result":"success"}', tm_create: '2026-09-04T00:00:02Z' },
      { id: 'm2', role: 'assistant', content: '', tool_calls: [{ id: 't1' }], tm_create: '2026-09-04T00:00:01Z' },
      { id: 'm1', role: 'user', content: 'What is going on?', tm_create: '2026-09-04T00:00:00Z' },
    ]);

    render(<CaseInsightAssistantPanel caseId="case-1" />);

    await waitFor(() => {
      expect(screen.getByText('Customer mentioned cancelling.')).toBeInTheDocument();
    });

    expect(screen.getByText('What is going on?')).toBeInTheDocument();
    expect(screen.queryByText('{"result":"success"}')).not.toBeInTheDocument();

    // The proactive note carries its own accessible label, so it is never
    // mistaken for an answer to the question above it.
    expect(screen.getByLabelText('Proactive insight')).toBeInTheDocument();
  });

  it('renders a message with no origin field unchanged', async () => {
    // Backward compatibility: every message written before this feature has no
    // origin field at all.
    createAicallAPI.mockResolvedValue({ id: 'aicall-1' });
    getAicallsAPI.mockResolvedValue([{ id: 'aicall-1', status: 'progressing' }]);
    getAimessagesAPI.mockResolvedValue([
      { id: 'm1', role: 'assistant', content: 'An ordinary answer.', tm_create: '2026-09-04T00:00:00Z' },
    ]);

    render(<CaseInsightAssistantPanel caseId="case-1" />);

    await waitFor(() => {
      expect(screen.getByText('An ordinary answer.')).toBeInTheDocument();
    });
    expect(screen.queryByLabelText('Proactive insight')).not.toBeInTheDocument();
  });

  it('fires the listen call once with the id Start returned', async () => {
    // THE TRIGGER. Design rev 15 removed the backend's implicit Start hook, so
    // without this call listening never starts at all -- silently, with no
    // error anywhere for anyone to notice.
    createAicallAPI.mockResolvedValue({ id: 'aicall-1' });
    getAicallsAPI.mockResolvedValue([{ id: 'aicall-1', status: 'progressing' }]);
    getAimessagesAPI.mockResolvedValue([]);

    render(<CaseInsightAssistantPanel caseId="case-1" />);

    await waitFor(() => {
      expect(listenAicallAPI).toHaveBeenCalledTimes(1);
    });
    expect(listenAicallAPI).toHaveBeenCalledWith('aicall-1');
  });

  it('still renders when the listen call fails', async () => {
    // Fire-and-forget: the response carries no listening-status field to
    // branch on, so a failure must never block the panel or surface an error
    // to the agent.
    createAicallAPI.mockResolvedValue({ id: 'aicall-1' });
    getAicallsAPI.mockResolvedValue([{ id: 'aicall-1', status: 'progressing' }]);
    getAimessagesAPI.mockResolvedValue([
      { id: 'm1', role: 'assistant', content: 'An ordinary answer.', tm_create: '2026-09-04T00:00:00Z' },
    ]);
    listenAicallAPI.mockRejectedValue(new Error('listen unavailable'));

    render(<CaseInsightAssistantPanel caseId="case-1" />);

    await waitFor(() => {
      expect(screen.getByText('An ordinary answer.')).toBeInTheDocument();
    });
  });
```

Match the existing tests' setup — this file already mocks `@/api/services/aicalls` and uses fake timers for the poll. Add `listenAicallAPI` to that same mock.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen/square-talk && \
  npx vitest run src/features/cases/CaseInsightAssistantPanel.test.jsx
```
Expected: FAIL — the raw JSON renders, there is no `Proactive insight` label, and `listenAicallAPI` is never called.

- [ ] **Step 3: Add the filter and the treatment**

In `square-talk/src/features/cases/CaseInsightAssistantPanel.jsx`, add above the thread component:

```jsx
// Rows produced by the assistant's tool use that the agent should never see:
// the empty-content assistant row that merely carries tool_calls, and the tool
// row holding the raw JSON result. Both have always been rendered; live call
// listening makes them far more visible because it is the first thing that can
// trigger a tool call without the agent having asked for anything.
//
// Field names are snake_case (tool_calls, not toolCalls) -- that is the actual
// wire shape, matching msg.role / msg.content / msg.tm_create alongside it.
//
// Render-only. Webhook delivery is untouched.
function isToolNoise(msg) {
  if (msg.role === 'tool') return true;
  return msg.role === 'assistant' && !msg.content && msg.tool_calls?.length > 0;
}
```

and change the render to filter and branch:

```jsx
        messages
          .filter((msg) => !isToolNoise(msg))
          .slice()
          .reverse()
          .map((msg) =>
            msg.origin === 'proactive' ? (
              <div
                key={msg.id}
                className="rounded-lg border border-primary-500/40 bg-primary-600/10 p-2.5 text-sm text-surface-100 mr-6"
                role="note"
                aria-label="Proactive insight"
              >
                <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-primary-300">
                  Proactive insight
                </div>
                <p className="whitespace-pre-wrap">{msg.content}</p>
                <span className="text-xs text-surface-400">{formatThreadTimestamp(msg.tm_create)}</span>
              </div>
            ) : (
              <div
                key={msg.id}
                className={`rounded-lg p-2.5 text-sm ${
                  msg.role === 'user' ? 'ml-6 bg-primary-600/15 text-surface-100' : 'mr-6 bg-surface-600 text-surface-100'
                }`}
              >
                <p className="whitespace-pre-wrap">{msg.content}</p>
                <span className="text-xs text-surface-400">{formatThreadTimestamp(msg.tm_create)}</span>
              </div>
            ),
          )
```

The `.filter()` goes before `.slice().reverse()` — the existing `messages.length === 0` empty-state check above still keys off the unfiltered array, which is correct: a thread containing only noise rows is not empty, it is just quiet.

- [ ] **Step 3a: Fire the explicit `listen` call after `Start` resolves**

Add the wrapper to `square-talk/src/api/services/aicalls` alongside the existing ones — `POST /service_agents/aicalls/{id}/listen`, no request body — and call it in the **same effect that already calls `Start`**, immediately after `Start` resolves:

```jsx
  // Design rev 15 made listening an explicit, separately-callable action
  // rather than a side effect of creating the Q&A AIcall. Two calls, in
  // order, because the second needs the first's id.
  //
  // FIRE-AND-FORGET on purpose: the response is the current AIcall and
  // carries NO listening-status field to branch on. A failed or slow response
  // must never block rendering the panel, and a repeated call (a fast
  // double-open) is free -- the backend's own idempotency check makes it a
  // no-op.
  const aicall = await createAicallAPI(/* ...existing args... */);
  listenAicallAPI(aicall.id).catch(() => {
    // Deliberately swallowed. Listening is an enhancement; the panel works
    // without it, and the backend records its own outcome either way.
  });
```

**`/service_agents/aicalls/{id}/listen`, never the top-level `/v1/aicalls/{id}/listen`.** square-talk is the Agent-facing app; the top-level path is Admin-console-gated and would return `ErrPermissionDenied` for an ordinary agent, which is the exact failure design review round 13's BLOCKING-1 exists to prevent.

- [ ] **Step 4: Run the tests and lint**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen/square-talk && \
  npx vitest run src/features/cases/CaseInsightAssistantPanel.test.jsx && \
  npm run lint
```
Expected: the whole suite passes, including the pre-existing tests, and lint is clean.

- [ ] **Step 5: Commit**

Commit message body:

```
- square-talk: Fire POST /service_agents/aicalls/{id}/listen after Start resolves, as a separate fire-and-forget call
- square-talk: Hide tool-call and tool-result noise rows from the Case Insight Assistant thread
- square-talk: Render origin=proactive notifications with a distinct surface and an accessible label
```

Stage `square-talk/src/features/cases/` and `square-talk/src/api/services/`, then commit with the branch name as the title and the body above.

---

## Task 33: Phase B — final verification and the `monorepo-javascript` PR

- [ ] **Step 1: Build and test both apps**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
( cd square-admin && npm run lint && npx react-scripts test --watchAll=false && npm run build ) && \
( cd square-talk && npm run lint && npx vitest run && npm run build )
```
Expected: both lint, test and build cleanly.

- [ ] **Step 2: Pull the latest main and check for conflicts**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git fetch origin main && \
  echo "--- conflicts ---" && \
  git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" ; \
  echo "--- new on main ---" && \
  git log --oneline HEAD..origin/main
```
If conflicts exist: rebase, resolve, and re-run Step 1 in full.

- [ ] **Step 3: Confirm no fabricated ticket number**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git log origin/main..HEAD --format='%s%n%b' | grep -E "VOIP-[0-9]+"
```
Expected: no output.

- [ ] **Step 4: Push and open the PR**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git push -u origin NOJIRA-Insight-AI-realtime-listen
```

Then open the PR with title `NOJIRA-Insight-AI-realtime-listen` and this body:

```
Frontend half of Insight AI realtime call listening. Fires the explicit listen
trigger when a Case panel opens, renders proactive notifications distinctly from
answers, and hides the tool-call noise rows that every Insight tool call has
always produced.

DEPLOY ORDER: land the monorepo PR first. Both panels now call
POST /service_agents/aicalls/{id}/listen when they open, and that endpoint does
not exist until the backend ships. The call is fire-and-forget and its failure is
swallowed, so deploying this first degrades rather than breaks -- the panels work
exactly as they do today and nothing starts listening -- but there is no reason
to take that state deliberately.

Nothing else here depends on deploy order. No message will carry origin=proactive
until the backend flag is enabled, and the noise filter only ever hides rows that
were noise before this feature existed. It is a client-side render filter only --
webhook delivery is untouched, so a tenant's own automation still receives every
aimessage_created event exactly as before.

- square-admin: Fire POST /service_agents/aicalls/{id}/listen after Start resolves, as a separate fire-and-forget call
- square-admin: Hide tool-call and tool-result noise rows from the Case Insight Assistant thread
- square-admin: Render origin=proactive notifications with a distinct surface and an accessible label
- square-talk: Fire POST /service_agents/aicalls/{id}/listen after Start resolves, as a separate fire-and-forget call
- square-talk: Hide tool-call and tool-result noise rows from the Case Insight Assistant thread
- square-talk: Render origin=proactive notifications with a distinct surface and an accessible label
```

**Do not merge.** Wait for explicit authorization; then squash-merge.

- [ ] **Step 5: Clean up the worktree once the PR is merged**

```bash
cd /home/pchero/gitvoipbin/monorepo-javascript && \
  git worktree remove /home/pchero/gitvoipbin/monorepo-javascript/.worktrees/NOJIRA-Insight-AI-realtime-listen && \
  git pull origin main
```

---

## Follow-ups to file as separate Jira tickets

These are real, but out of scope. **File them in Jira, not GitHub Issues** — Jira is the issue tracker for this project. Do this after Phase A's PR is open, not before.

| # | Item | Why it is not in this plan |
|---|---|---|
| 1 | `transcripthandler.dbDelete` publishes `transcript_created` on DELETE as well as create | Changing the emitted event type is routing-key-visible and affects every current subscriber. This design defends against it at intake instead. |
| 2 | An orphaned `tool`-role message reaches the LLM with no preceding `tool_calls` entry in the same request | `ToolHandle` writes the tool-call row with empty content, and pipecat-manager's own context filter drops empty-content rows — so what reaches the LLM is a `tool` result with nothing to reference. This predates and is independent of this feature; every existing Insight tool call already produces it. **Worth confirming against production traffic promptly** — OpenAI may be lenient in practice, or few sessions may exercise a tool call followed by a genuine follow-up question. Confirm before deciding urgency. |
| 3 | Decouple `Send`'s cooldown from `tm_update` onto a dedicated `tm_last_send` | Pre-existing fragility. This plan bounds it (listen's own two writes skip the bump) but does not remove it for other write paths. |
| 4 | Tool-call rows still reach a tenant's webhook consumer | The panels now hide them, but `aimessage_created` still fires for each. Whether to suppress those webhooks depends on whether any tenant automation relies on receiving them — unknown, and a genuinely separate decision. |
| 5 | `get_aicall_messages` can leak `listen_internal` rows into an answer | That tool reads by `aicall_id` alone and bypasses `getPipecatcallMessages`, so this plan's SQL-layer exclusion does not cover it. Lower severity: mechanical tool-call JSON in an answer, not a lost system prompt. |
| 6 | Whether Insight listening becomes a billed line item, and under which meter | A pricing decision, not an architecture one. The architecture deliberately keeps the STT cost off the customer's transcription bill, which makes this a clean choice rather than an accident. |
| 7 | A "🎧 listening" indicator in the Case panel (design §11 item 14) | `POST /service_agents/aicalls/{id}/listen` deliberately returns no listening-status field, so the caller cannot tell "started" from "reused" from "not eligible" from "still waiting on the confbridge". That matches the API's stated scope (separation of concerns, not status visibility). Closing the gap does **not** require blocking the endpoint's response on the confbridge wait: the AIcall's own `ListenCallID` / `Metadata[listen_transcribe_id]` already carry the eventual outcome and could be surfaced over the transport each panel already uses for messages. |

---

## Self-review

Run against the design document with fresh eyes after the plan is written. Findings and fixes below; issues found were fixed inline.

**1. Spec coverage.** Walked each design section against a task:

| Design section | Task |
|---|---|
| §5.1 the explicit trigger API, §5.1.1 `ProcessListen` / `checkListenEligible` / `runListenStart` | 20 (ai-manager side), 27 (public surface) |
| §5.2.1 `IDAIManagerListen` | 1, and Task 0 Step 3 resolves §11 item 5 |
| §5.2.2 reuse rule + the per-AIcall start lock, §5.2.3 language, §5.2.4 `UpdateListenState` | 20, with the lock's Redis primitives in 11 |
| §5.3.1 binding, §5.3.2 intake, §5.3.3 buffering, §5.3.4 debounce | 11, 23, 24 |
| §5.4.1 preconditions, §5.4.2 context, §5.4.3 session start | 21, 22 |
| §5.4.3a pipecatcall-id threading | 5 |
| §5.4.4(a) `RunLLM:false`, (b) foreign-pipecatcall guard, (c) reject guard | 16, 19, 18 |
| §5.4.5 `Origin` tagging + two-fetch context | 7, 15, 17 |
| §5.5 `notify_agent` definition, §5.5.2 invariant, §5.5.3 configurability | 16, 18, 21 |
| §5.6 message storage | 7, 18 |
| §5.7 lifecycle and cleanup | 25 |
| §5.8 data model | 6, 7, 8 |
| §5.9 speaker mapping | 0, 23 |
| §5.10.1 frontend render treatment, §5.10.1a the explicit two-call trigger | 31, 32 |
| §5.10.2 RST | 28 |
| §5.12 config, §5.13 metrics | 10, 26 |
| §5.14 the commented-out ghost | 24 |
| §7 testing items 1–19 | distributed across 1–32 |
| §8 rollout | 30 (PR body), 29 (runbook) |
| §11 open items | 0 (blocking), follow-ups table (non-blocking) |

Gaps found and closed: §5.14's ghost deletion had no home until it was folded into Task 24; §5.12's `SetXxxForTest` helpers were missing until added to Task 10 Step 6.

**2. Placeholder scan.** Fixed: Task 22 and Task 25's test bodies were left as bare `// ...` comments; each now carries a named coverage list stating exactly what to assert. Tasks 20 and 23's test tables likewise. Task 14's DB test still delegates its harness setup to the existing `AIcallUpdate` test rather than duplicating unknown fixture plumbing — that is a deliberate read-the-real-thing instruction, not a placeholder, and the two assertions that matter are spelled out.

Removed a real placeholder: an earlier draft of Task 27 said "add the field to the schema" without the YAML. It now carries the block.

**3. Type consistency.** Checked names across tasks and fixed three drifts:

- `AIcallUpdateWithoutTouchingTMUpdate` (design's phrasing) vs. `AIcallUpdateNoTouchTMUpdate` — unified on the latter everywhere (Tasks 14, 20, 25).
- `ListenPendingPopAll` returning `[]string` is consumed by `runListenTurnWithLines(ctx, c, lines []string)` in Tasks 22 and 25 — consistent.
- `stopListening(ctx, c *aicall.AIcall)` takes the AIcall, not an id, in all four call sites (Task 22's two, Task 25's two) — consistent, and it matters: `clearListenState` reads metadata off the value the caller already holds.

Also verified `speakerTag` output strings (`[CUSTOMER]` / `[AGENT]` / `[SPEAKER]`) match what `buildListenTranscriptBlock`'s test asserts in Task 21, and that `listenTranscriptNewMarker` is the same constant in both the prompt text (Task 21 Step 3) and the block builder.

**4. Ordering hazards.** Two tasks have a real dependency the checkbox order would otherwise hide, and both now say so explicitly: Task 13 needs Task 14's `dbhandler.AIcallGetSkipCache`, and Task 22 needs Task 25's `stopListening`. Each names the other and offers the swap.

**5. One design assumption this plan contradicts, deliberately.** Design §5.2.1 flags as an open question whether `IDAIManagerListen` needs a backing customer row. Task 0 Step 3 answers it with evidence the design did not have: `IDAIManager`'s literal is malformed and parses to `uuid.Nil`, yet `summaryhandler` uses it as a transcribe owner in production today — which proves no row is required. That same finding makes Task 1's "must be a well-formed literal" test necessary rather than pedantic: copying the neighbouring constants' shape would silently collapse the new sentinel onto `IDAIManager` and destroy the separation §5.2.1 exists to create.

---

## Sync to design rev 23 (2026-09-04)

This plan was originally written against design **rev 14**. The design then reopened its review loop for eight more rounds (13-20, revisions 15-23) and closed it Approved at rev 23. This revision syncs the plan to rev 23. It is a targeted sync, not a rewrite: only what the design actually changed was touched.

**What the design changed, in three items.**

1. **The trigger became an explicit API** (rev 15, endpoint surface corrected in rev 16). `Start`'s hook is gone. `POST /service_agents/aicalls/{id}/listen` → `ProcessListen` → `checkListenEligible` (steps 1-6, synchronous) → `runListenStart` (steps 7-8, detached). `ensureListen`/`ensureListenAsync` are not part of the design any more, and the endpoint sits on the Agent-facing surface, not the top-level Admin one.
2. **An event-ordering fix** (rev 15, restructured rev 16): the DB write and the Redis `SADD` land *before* `TranscribeV1TranscribeStart`, against a caller-pre-generated transcribe id, with an explicit `rollbackListenState`.
3. **A per-AIcall mutual-exclusion lock** (rev 17, hardened through rev 20): ownership token, atomic compare-and-delete release, a release context detached via `context.WithoutCancel`, and a best-effort release on the acquire-error path.

**Tasks touched, and why.**

| Task | Change |
|---|---|
| Front matter | Design reference `rev 14` → `rev 23`; added a summary of the three changes and rev 15's own scope-bound sentence |
| File structure | Added `listen_trigger.go` / `listen_trigger_test.go` per design §9; `listen.go`'s stated contents narrowed to the session path plus the shared helpers |
| **0** | Step 4 now covers **five** timing defaults, not three — added the two lock flags with the TTL's actual (re-derived) rationale and the max-wait → goroutine-timeout → lock-TTL cascade. Added a table mapping every design §11 open item to where this plan handles it |
| **2** | One stale prose reference: `ensureListen` → `checkListenEligible` |
| **10** | Eleven flags → **thirteen**, matching a fresh count of design §5.12. Added `aicall_listen_start_lock_ttl_seconds` (60) and `aicall_listen_start_lock_release_timeout_seconds` (3), their default assertions, a second standing ordering invariant (lock TTL > goroutine timeout), and a TTL override helper for tests |
| **11** | Added `ListenStartLockAcquire` / `ListenStartLockRelease` (key format in one place; `EVAL` compare-and-delete release), the start-lock key golden-test row, the interface entries, and an explicit note recording the one naming divergence from design §9 (`ListenAIcallIDRemove` for the design's `ListenTranscribeAIcallRemove`) |
| **20** | **Rewritten.** `ensureListen`/`ensureListenAsync` → `ProcessListen`/`checkListenEligible`/`runListenStart`; the `Start` hook removed entirely; the combined AIcall-liveness gate added; `ensureListenTranscribe` replaced by `startListenTranscribe` carrying design §5.2.2's lock sequence, `dupFilters`, the acquire-error best-effort release, the deferred detached release, and the `isAlreadyProgressing` discrimination; `rollbackListenState` added; `UpdateListenState` re-signatured to take the AIcall id and read fresh, with the `owns` merge **scoped to same-transcribe-id writes**; the ai-manager `/v1/aicalls/{id}/listen` route added; tests restructured into seven suites implementing design §7 items 1 and 2 (six in `pkg/aicallhandler/listen_trigger_test.go`, plus `Test_processV1AIcallsIDListenPost` in `pkg/listenhandler/v1_aicalls_test.go` — the last of those added by review round 1's MEDIUM-2) |
| **26** | Recorded that the new outcomes are `result` **label values**, not new families, and the design §11 item 16 decision behind `skipped_start_locked` |
| **27** | **Grew a whole new half.** Was only `notify_agent` + `origin`; now also the `AIV1AIcallListen` RPC client (with its explicit 10s timeout and the *corrected* per-hop justification), the `service_agents/aicalls/id_listen.yaml` path, `ServiceAgentAIcallListen` with the `IsAgent` → `PermissionAll` → `aicallGet` → ownership-compare → RPC sequence, and a verification step that fails if the route ever appears outside `service_agents` |
| **28** | Added the endpoint's own RST doc (mirroring `contact_addresses/:id/claim`, **not** `terminate`), a "Starting to listen" subsection, and an explicit instruction not to describe listening as automatic on AIcall creation |
| **29** | Config-flag count eight → thirteen; the new route in the routing table; a "how listening starts" runbook paragraph; runbook rows for `skipped_confbridge_not_ready` and `skipped_start_locked`; and a note about the start-lock key |
| **30** | PR body bullets updated for the new trigger, the lock, the RPC client and the endpoint |
| **31, 32** | Both panels now fire the `listen` call as a **separate, explicit, fire-and-forget** call after `Start` resolves — with tests pinning that it is called once with `Start`'s own id, and that a rejected `listen` call still renders the panel |
| **33** | The old "safe to deploy at any time, independently of the backend" claim is no longer true and was replaced with an explicit deploy-order note |
| Follow-ups | New row 7 for design §11 item 14 (a listening indicator), so the deliberate absence of a status field is recorded rather than forgotten |

**Tasks deliberately NOT touched: 1, 3-9, 12-19, 21-25.** Design §0's rev-15 row is explicit that neither of its changes touches §5.2-§5.9's Layer 1/2 architecture, §5.4-§5.7's turn/tool/lifecycle mechanisms, or §5.9's speaker-mapping conclusions. Task 6's migrations were re-checked against design §5.8 and still match it exactly (one column `listen_call_id` + index, two metadata keys) — that part of §5.8 did not change.

**Three things flagged rather than silently decided.**

1. **`checkListenEligible`'s error return.** Design §5.1's snippet propagates an error out of `ProcessListen`, while design §6's first row says a Case/call/transcribe lookup failure is logged and metered and "never fails the triggering call." This plan resolves it the §6 way: lookup failures return `proceed=false` with a **nil** error and a metered outcome, and the error return is reserved for genuinely unexpected conditions. **§6's first row already resolves this**; it is recorded here only because §5.1's own snippet doesn't restate it, and that snippet merely plumbs an error return without ever saying lookup failures produce a non-nil one. The plan's implementation (Task 20) returns a nil error from every early-exit branch in `checkListenEligible`, consistent with §6, and Task 20's doc comment says so explicitly so nobody later "simplifies" the unused return away.
2. **`UpdateListenState`'s file home.** Design §9 assigns `listen_trigger.go` and `listen.go` their contents by name but never says where `UpdateListenState` lives. This plan puts it in `listen_trigger.go`, with the create-or-reuse sequence that is its only caller.
3. **Step 3's idempotency predicate is an approximation, because the design's own step ordering makes the literal version impossible.** Design §5.1.1 step 3 compares the existing transcribe's `ReferenceID` against "the call we are about to resolve," but steps 4-5 are what resolve that call id and they run *after* step 3. This plan compares against the already-persisted `c.ListenCallID` instead — exact for the common repeated-panel-open case, divergent only if a Case's associated call somehow changed between two `ProcessListen` calls on the same AIcall. It deliberately does **not** reorder the design's steps to close the gap. **Full explanation, including the "do not move the check after steps 4-5" instruction and the code comment carrying it, is in Task 20** (the "Step 3's idempotency predicate approximates the design" prose block, and step 3's own comment in the `checkListenEligible` implementation). Whoever confirms whether a Case's reference call is fixed for the Case's lifetime should settle it: if it is, the approximation is exact.

### Review round 1 (plan-stage review loop) — findings fixed

Round 1 of the plan-stage review loop returned **REQUEST_CHANGES** with 3 MEDIUM and 9 LOW findings, all inside Task 20 or immediately adjacent to it. All twelve were fixed; the itemized account:

| Finding | Fix |
|---|---|
| **MEDIUM-1** — step 3's idempotency predicate silently diverges from design §5.1.1 and isn't flagged | Task 20 gained a prose block ("Step 3's idempotency predicate approximates the design…") and step 3's implementation gained a matching code comment, both naming the design's own forward-reference tension, the `c.ListenCallID` approximation, the one scenario it could be wrong in, and an explicit "do not reorder the design's steps to fix this." Recorded as item 3 of "flagged rather than silently decided" above |
| **MEDIUM-2** — Task 20's `go test -run` pattern named `Test_processV1AIcallsIDListenPost`, which no step ever specified, so the verification would pass vacuously (`-run` exits 0 on zero matches) | Added `pkg/listenhandler/v1_aicalls_test.go` to Task 20's Files block; added `Test_processV1AIcallsIDListenPost` to Step 1 (valid id → 200 with `ProcessListen` invoked; unknown id → 404 per design §7 item 2; unparseable id and wrong method → zero handler calls), **routed through `processRequest`** following the precedent Task 13 set for this exact dispatcher's regex-anchoring behaviour; Step 1's table is now seven tests with a file column; Steps 2 and 6 both run `./pkg/listenhandler/` too and both now instruct the reader to check the `-v` run list rather than the exit code; Step 5 carries its own test requirement |
| **MEDIUM-3** — the confbridge give-up paths didn't log the party count design §6 requires | `waitForConfbridgeReady` now returns `(confbridgeReadyResult, int)`, tracking the last observed `len(cb.ChannelCallIDs)` (`-1` = never bridged, deliberately distinct from an observed 0); `runListenStart`'s `confbridgeNotReady` and `confbridgeError` branches each emit a `log.Warnf` carrying `last_party_count` before incrementing the metric. `Test_waitForConfbridgeReady`'s table gained an `expectLastPartyCount` column and three rows exercising it (never-bridged → -1, error-after-a-read → the retained count, stable-3 → 3) |
| **LOW-1** — the rev-23 rewritten-task list omitted Task 33 | Front matter now reads "Tasks 0, 2, 10, 11, 20, 26, 27, 28, 29, 30, 31, 32 and 33", matching the sync table and the complement list |
| **LOW-2** — three places disagreed on "seven" vs "six" listen metrics and where they live | Reconciled on one statement everywhere: **seven new metrics total, six of them in `metrics_listen.go`**, with `aicall_foreign_pipecatcall_dropped_total` in `metrics_foreign.go` and audited separately. Fixed at the File-structure table (both rows), Task 26's intro, and Task 29's `operations.md` step |
| **LOW-3** — `waitForConfbridgeReady` re-checks only liveness, not ownership, undocumented | Its doc comment now states this is intentional: a live call's `CustomerID` is immutable, so step 6's single ownership check still holds for the whole wait |
| **LOW-4** — `ctx.Done()` reported as `confbridgeError`, a category mismatch with design §5.13's RPC-failure definition | The branch now carries a comment stating it is unreachable at the shipped defaults (poll budget 30s < goroutine timeout 45s, an invariant Task 10 pins) and must be given its own outcome if those defaults ever move |
| **LOW-5** — the `checkListenEligible` error-return question was framed as "a design ambiguity to settle before implementation" when §6 already settles it | Softened to: §6's first row already resolves it; recorded only because §5.1's snippet doesn't restate it, and the plan's own implementation returns nil from every early-exit branch, consistent with §6 |
| **LOW-6** — the error return is never set non-nil anywhere, with nothing saying so | `checkListenEligible`'s doc comment now states the consequence explicitly and instructs an implementer not to drop the six-value signature, which matches design §5.1's own snippet shape |
| **LOW-7** — `UpdateListenState` / `rollbackListenState` read cache-first where §5.2.4 says "a fresh `AIcallGet`" | `UpdateListenState`'s doc comment now notes §5.2.4's contrast is with the caller's in-hand struct, not with cache freshness, and that `AIcallGetSkipCache` (Task 14) exists but isn't needed here because nothing else writes this cache entry concurrently within one request; `rollbackListenState`'s comment points at that reasoning |
| **LOW-8** — neither early return incremented `aicall_listen_start_total`, so the design's own common path was invisible in the metric | Step 3's idempotency short-circuit now increments `result="reused"`. Step 1's flag-off return stays unmetered, with a comment recording *why*: design §5.13 never says which label covers "flag disabled", inventing a seventh value would be exactly the unilateral decision this task flags elsewhere, and during a flag-off stage the counter would say nothing the flag's own value doesn't |
| **LOW-9** — a test row named "refused before any RPC" when `aiHandler.Get` (in-process) necessarily runs first | Both liveness rows renamed to "…before any cross-service RPC", with a comment stating the assertion that actually matters: zero `TranscribeV1*` / `CallV1*` / `ContactV1*` calls |

### Review round 2 (plan-stage review loop) — findings fixed

Round 2 returned **APPROVE** with 0 BLOCKING/HIGH/MEDIUM findings and 4 LOW nitpicks, all in Task 20. All four were fixed; the itemized account:

| Finding | Fix |
|---|---|
| **LOW-a** — `expectLastPartyCount`'s field comment told the reader to assert `-1` on the "still queued" row, which actually resolves to 2 | The comment now names the two rows where `-1` genuinely matters (ConfbridgeID never resolves; `CallV1ConfbridgeGet` errors on the first poll) and adds a parenthetical stating that the "still queued" row starts at `uuid.Nil` but does resolve, so its expected value is 2 |
| **LOW-b** — two rows omitted `expectLastPartyCount`, silently defaulting to `0`, which is wrong for both | The "ringing, then answers" row now asserts `2` (it reaches `confbridgeReady`). The "call ends mid-poll" row now asserts `1`, and its name/comment pin the scenario that makes that unambiguous: one poll observes a live 1-party bridge, the next poll's `CallV1CallGet` comes back hung up, and the liveness check runs before the confbridge read |
| **LOW-c** — `runListenStart`'s comment claimed design §6 requires the party count "in the log line of both give-up branches" | Reworded to the accurate split: §6 requires it only for the not-ready branch (§6's `skipped_confbridge_not_ready` row); this plan also logs it on the error branch for the same diagnostic value, though §6's `skipped_confbridge_error` row does not mandate it there; and the third give-up path (`confbridgeCallEnded`) has no log line, matching §6's own silence on that outcome |
| **LOW-d** — Task 20's Files block omitted `bin-ai-manager/docs/architecture.md`, which its own Step 5 and Step 8 both touch | Added to the Files block with a parenthetical scoping it to the routing-table entry for the new route and pointing at Task 29 for the rest of that file, matching Task 29's own listing of the same file |

### Review round 3 (plan-stage review loop) — APPROVE, second consecutive approval, loop closes

Round 3 independently re-verified all four of round 2's fixes against current text and the actual `waitForConfbridgeReady` implementation (not the fix agent's self-report), re-derived every one of the 11 `expectLastPartyCount` test-table rows from the implementation end to end (not just the two rounds 1-2 touched), and re-checked design §6's exact rows a third time. All four fixes hold; no new defect was found inside them. **0 BLOCKING / 0 HIGH / 0 MEDIUM. This is the second consecutive APPROVE (round 2 also approved) — the plan-stage (설계) review loop is CLOSED per policy (min 2 rounds, 2 consecutive approvals, max 20; closed in 3).**

Eight non-blocking nitpicks were recorded (an undocumented `expectPolls` zero-default sentinel in 9 of 11 test rows — pre-existing, not fix-introduced; a gofmt alignment nit in one test-table literal; a residual plural ("give-up path**s**"/"require**s**") in round 1's own ledger row, directly corrected 19 lines below; the round-2 ledger under-reporting that LOW-c's fix touched three locations, not one; a slightly loose function-doc-comment phrasing; a comment placed one branch too early; an unstated poll/RPC-pairing asymmetry for two edge-case rows; and a runbook row that could cross-reference the log-line diagnostic it describes elsewhere). None require reopening the loop; explicitly reviewer-approved to fold into the first implementation commit touching Task 20, or drop.

**Loop history:** round 1 REQUEST_CHANGES (3 MEDIUM + 9 LOW, fixed) → round 2 APPROVE (4 LOW nitpicks, fixed) → round 3 APPROVE (second consecutive, closes). Proceeding to implementation per this plan as it now stands.
