# Fix: AIAcceptProposal leaves stale AI cache — Design Spec

**Date:** 2026-05-29
**Status:** Approved (2 review iterations)
**Owner:** bin-ai-manager

---

## Problem Statement

After calling `POST /v1.0/aipromptproposals/{id}/accept`, the database is updated correctly (new `ai_ai_prompt_histories` row, `ai_ais.init_prompt` set to the proposed prompt, `ai_ais.current_prompt_history_id` set to the new history id, proposal status flipped to `accepted`). However, subsequent reads via `GET /v1.0/ais/{id}` return the **pre-accept** state of the AI: old `init_prompt`, old `current_prompt_history_id`.

The symptom is reproducible: a user accepted proposal `f97cb187-e572-49e9-88df-544833583415` for AI `74b3d982-8a5a-43b5-9c1d-51e6ce508b0a` at 2026-05-29 06:54:01 UTC (api-manager pod `vtfcm` logged the POST returning 200 in 136 ms). Subsequent GET on that AI continued to return the old values for over 25 minutes.

---

## Root Cause

`bin-ai-manager/pkg/dbhandler/ai.go` uses a write-through cache for the `ai_ais` table. Every existing write method in that file refreshes the cache after the DB write returns:

| Method | DB write | Cache invalidation |
|---|---|---|
| `AICreate` (line 36-44) | `h.db.Exec` | `_ = h.aiUpdateToCache(ctx, c.ID)` (line 41) |
| `AIDelete` (line 128-151) | `h.db.Exec` | `_ = h.aiUpdateToCache(ctx, id)` (line 148) |
| `AIUpdate` (line 196-225) | `h.db.Exec` | `_ = h.aiUpdateToCache(ctx, id)` (line 222) |

`AIGet` (line 109-125) reads from cache first; on miss it reads from the DB and seeds the cache. The cache has no automatic short-TTL invalidation — it persists until explicitly refreshed.

`AIAcceptProposal` in `bin-ai-manager/pkg/dbhandler/aipromptproposal.go:240-330` is a transactional method that ends with `tx.Commit()` and a bare `return nil`. It modifies `ai_ais` inside the transaction (step 4 at lines 306-313), but it never calls `aiUpdateToCache`. The cache invariant required by the rest of `dbhandler` is therefore broken.

The bug is best characterized as **a code path that mutates a cached table outside the cache-invalidation convention of the package it lives in**. Every other writer in the package honors the convention; this transactional method silently does not.

---

## Why This Slipped Through

- The original design spec (`2026-05-29-ai-prompt-proposal-design.md`) focused on transactional correctness of the Accept flow (lock order, drift detection, idempotency). It did not mention the AI cache.
- All three iterations of the original design review (architect agent) focused on the transactional invariants and did not notice the missing cache invalidation.
- All three iterations of the post-implementation PR code review checked transaction semantics, error mapping, and test coverage. None of them grepped for "cache" or compared the new write path against the existing `ai.go` convention.
- The unit tests use a mocked DBHandler, so they never exercise the cache write path.

This is a class of defect worth flagging in future reviews: **whenever a new code path writes to a row whose table has a cache layer, the cache invalidation must be exercised.** Adding a checklist item to the architect review prompt is out of scope for this PR but worth a follow-up.

---

## Goals

- Reads through `AIGet` after a successful Accept must reflect the new `init_prompt`, `current_prompt_history_id`, and any other AI fields the Accept mutated.
- The fix must follow the existing cache-invalidation convention so the next reader of `aipromptproposal.go` does not have to re-invent the pattern.
- The fix must be testable without a real Redis or MySQL.
- No regression to the transactional correctness already shipped.

## Non-Goals

- Restructuring the dbhandler's cache architecture.
- Adding cache layers for tables that don't have them today (`ai_ai_prompt_proposals`, `ai_ai_prompt_histories` — both read straight from DB).
- Eliminating the convention of mixing cache calls into dbhandler write methods. That's a larger refactor; this PR only follows the existing pattern.
- Adding TTL-based cache expiry as a defensive measure. Out of scope.

---

## Fix

### Approach

After the transactional method finishes — both on the happy path (commit) AND on the `ErrProposalAlreadyAccepted` early-return — refresh the AI cache. This is done in a single deferred block so there is exactly one cache-refresh call-site, regardless of which exit path the function takes. The deferred refresh also serves as a self-healing mechanism: if a previous Accept committed the AI row but its cache refresh never ran (pod crash, SIGTERM, network hiccup between commit and cache write), the next caller hitting `ErrProposalAlreadyAccepted` will repair the stale cache.

### Code change

`bin-ai-manager/pkg/dbhandler/aipromptproposal.go` — restructure `AIAcceptProposal` to extract `aiID` into a function-scope variable populated as soon as we read it from the locked proposal row, and add a refresh flag set on the two exit paths that should trigger a cache refresh (commit success and `ErrProposalAlreadyAccepted`):

```go
func (h *handler) AIAcceptProposal(ctx context.Context, proposalID uuid.UUID, newHistoryID uuid.UUID, proposedPrompt string) error {
    var (
        aiID         uuid.UUID
        refreshCache bool
    )

    tx, err := h.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("AIAcceptProposal: BeginTx: %w", err)
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback()
        }
        // Refresh the AI cache after the function exits, on these two paths:
        //   - happy commit (refreshCache=true was set just before tx.Commit())
        //   - ErrProposalAlreadyAccepted (a previous winner already advanced the AI;
        //     self-heal in case that winner crashed before its cache refresh ran)
        // Use context.Background() so a cancelled request context does not skip
        // the cache refresh.
        if refreshCache && aiID != uuid.Nil {
            _ = h.aiUpdateToCache(context.Background(), aiID)
        }
    }()

    // ... step 1: SELECT FOR UPDATE proposal, populate pAIID, pBasis, pStatus, pTMDelete ...

    aiID, _ = uuid.FromBytes(pAIID)   // captured for the deferred cache refresh

    if pStatus == string(aipromptproposal.StatusAccepted) {
        refreshCache = true   // self-heal stale cache on the idempotent path
        err = ErrProposalAlreadyAccepted
        return err
    }
    // ... existing not-completed / drifted / not-found checks ...

    // ... steps 2-5: lock AI, INSERT history, UPDATE AI, UPDATE proposal ...

    if err = tx.Commit(); err != nil {
        return fmt.Errorf("AIAcceptProposal: commit: %w", err)
    }
    refreshCache = true   // happy path: cache must reflect post-accept state
    return nil
}
```

The diff is roughly 8-10 lines: two new local variables, one new deferred refresh block, two `refreshCache = true` assignments, and moving `aiID, _ = uuid.FromBytes(pAIID)` so it's set inside the function-scope `aiID`. Everything else in the existing method body remains unchanged.

### Why this shape

- **Single refresh call-site.** Future maintainers see one place that decides "should we refresh", not multiple `aiUpdateToCache` calls scattered through return paths.
- **`context.Background()` for the refresh.** The 136 ms commit observed in the incident shows the request context can plausibly be cancelled between `tx.Commit()` and the cache write under slow-network conditions. Using `context.Background()` for the post-commit refresh ensures it always runs to completion (or fails on its own, in which case the next AIGet re-seeds from DB).
- **`_ =` discard.** Matches the silent-discard convention in `ai.go` (lines 41, 122, 148, 222). A transient cache write failure is non-fatal — the DB is the source of truth. Adding a log here would diverge from convention; the existing approach is intentional.
- **Self-heal on `ErrProposalAlreadyAccepted`.** Belt-and-suspenders: if the *winning* Accept's pod crashed between `tx.Commit()` and the cache refresh, the cache is stale. The *next* Accept attempt — which a polling UI or retry will produce — short-circuits with `ErrProposalAlreadyAccepted` AND refreshes the cache. This closes the failure window without requiring crash-recovery infrastructure.

### Why `aiUpdateToCache` and not `aiSetToCache`

- `aiSetToCache(ctx, c *ai.AI)` requires the caller to supply the AI struct. We don't have one in this scope; we'd have to construct it from the fields we just wrote, with risk of drift.
- `aiUpdateToCache(ctx, id uuid.UUID)` re-reads the AI from the DB via `aiGetFromDB` and writes that snapshot to the cache. It's the same helper the other write methods use. One extra `SELECT * FROM ai_ais WHERE id=?` per Accept is acceptable; the Accept path already runs five DB statements in the transaction.

### Why the deferred-block placement is safe

The defer block reads two function-scope variables (`err`, `refreshCache`, `aiID`) — none of which it writes. Critically, `_ = h.aiUpdateToCache(...)` discards the cache error; it does NOT assign to the named `err` return. This means:

- If `tx.Commit()` succeeded and the cache refresh fails, the function still returns `nil`. The user sees a successful Accept. The cache will self-heal on the next `aiUpdateToCache` call.
- If the function is exiting via an error path (any `return err`), the defer first rolls back the (possibly no-op SELECT-only) transaction, then attempts a cache refresh only when `refreshCache=true` (currently only the idempotent path). The rollback runs on a transaction that did no writes, so it is harmless.

A future maintainer who adds a new error-return path must NOT set `refreshCache = true` on that path. The spec assumes only the two paths above set it.

### Caching scope clarification

- `ai_ais` — **cached** via `cachehandler`. Affected by this fix.
- `ai_ai_prompt_histories` — **not cached**. The new history row inserted by `AIAcceptProposal` step 3 is visible immediately via `GET /v1.0/ais/{id}/prompt_histories`.
- `ai_ai_prompt_proposals` — **not cached**. The proposal's `status='accepted'` and `applied_prompt_history_id` are visible immediately via `GET /v1.0/aipromptproposals/{id}`.

Only the `ai_ais` row is affected by the bug. The fix targets only that.

---

## Tests

### Test framework

The existing dbhandler tests (`ai_test.go`, `aiaudit_test.go`, etc.) use:
- The package-level `dbTest` variable — a real `*sql.DB` connected to a test MySQL/MariaDB schema loaded from `scripts/database_scripts_test/table_*.sql`. **NOT** sqlmock.
- `cachehandler.NewMockCacheHandler(mc)` from gomock — the cache layer is mocked.
- `utilhandler.NewMockUtilHandler(mc)` — for deterministic `TimeNow()` and `UUIDCreate()` values.
- `handler{utilHandler, db: dbTest, cache: mockCache}` constructed inline per test.

New tests added in this PR follow that exact pattern.

### New test file: `bin-ai-manager/pkg/dbhandler/aipromptproposal_test.go`

Does not yet exist. Create it. Required fixtures: the test schema file `table_ai_ai_prompt_proposals.sql` already exists from the original feature PR; verify it is loaded by the package's `TestMain` (it should be — all `scripts/database_scripts_test/*.sql` are loaded). The test schema for `ai_ai_prompt_histories` must include the `proposal_id` column (added in PR #948).

### Required tests (4 new test functions)

**1. `Test_AIAcceptProposal_HappyPath_RefreshesAICache`** — the regression guard.

Setup (using gomock + `dbTest`):
- Pre-insert into `ai_ais`: one AI with `current_prompt_history_id = basisHistID`.
- Pre-insert into `ai_ai_prompt_histories`: one row with `id = basisHistID`.
- Pre-insert into `ai_ai_prompt_proposals`: one row with `status='completed'`, `basis_prompt_history_id = basisHistID`.
- Wire mocks: `mockUtil.EXPECT().TimeNow().Return(curTime).AnyTimes()`; `mockCache.EXPECT().AISet(ctx, aiMatcher).Return(nil).Times(1)` where `aiMatcher` is a `gomock.Matcher` that asserts the AI's `current_prompt_history_id == newHistID` and `init_prompt == "new prompt"`.

Action: `h.AIAcceptProposal(ctx, proposalID, newHistID, "new prompt")`

Assertions:
- Return value is `nil`.
- Query the DB directly: `ai_ais.init_prompt = "new prompt"`, `ai_ais.current_prompt_history_id = newHistID`.
- Query the DB directly: a new `ai_ai_prompt_histories` row exists with `id = newHistID` and `proposal_id = proposalID`.
- Query the DB directly: proposal row has `status='accepted'`, `applied_prompt_history_id = newHistID`.
- The `mockCache.AISet` Times(1) expectation is satisfied by `mc.Finish()`.

**2. `Test_AIAcceptProposal_AlreadyAccepted_RefreshesAICache_SelfHeal`** — covers the idempotent self-heal path.

Setup: pre-insert proposal with `status='accepted'` (simulating a winner that already committed).

Action: `h.AIAcceptProposal(ctx, proposalID, newHistID, "anything")`

Assertions:
- Return value is `dbhandler.ErrProposalAlreadyAccepted`.
- `mockCache.AISet` called exactly once with the AI's CURRENT state (post-winner). The DB query inside `aiUpdateToCache` will read whatever the winner already wrote.

**3. `Test_AIAcceptProposal_PromptVersionDrifted_NoCacheRefresh`** — negative control.

Setup: pre-insert proposal with `basis_prompt_history_id` that does not match the AI's current.

Action: `h.AIAcceptProposal(...)`

Assertions:
- Return value is `dbhandler.ErrPromptVersionDrifted`.
- **`mockCache.AISet` is NOT called.** This catches a future regression where someone moves the cache refresh out of the conditional and into an unconditional defer.

**4. `Test_AIAcceptProposal_ProposalNotFound_NoCacheRefresh`** — negative control.

Setup: do not insert the proposal row.

Action: `h.AIAcceptProposal(...)` with a random proposalID.

Assertions:
- Return value is `dbhandler.ErrNotFound`.
- `mockCache.AISet` is NOT called.

### Negative controls are load-bearing

Tests 3 and 4 are the protection against regressions where someone restructures the function and accidentally refreshes the cache on error paths. Without them, a future change that always refreshes (even on drift) could silently overwrite the cache with the un-changed AI state, which would mask other bugs. The negative-control assertions enforce: cache refresh runs ONLY on `tx.Commit() + nil` and `ErrProposalAlreadyAccepted` paths.

### Behavior tests already in place

The existing `TestAccept_*` tests in `aipromptproposalhandler/accept_test.go` use a mocked dbhandler and therefore do not exercise the cache. They remain valid and continue to pass unchanged. They cover the handler-layer error mapping; cache behavior is verified at the dbhandler layer where it lives.

### Manual verification on staging

After the fix lands:
1. Create an audit, then a proposal, wait for completion.
2. Accept the proposal.
3. Immediately `GET /v1.0/ais/{id}` — confirm `init_prompt` and `current_prompt_history_id` reflect the accept.
4. Run the same sequence twice in quick succession against the same proposal (the second hits the idempotent path) and confirm the AI state stays correct.

---

## Concurrency Analysis

Adding a cache write after the transaction does not change any concurrency properties:

- The transaction's serializability is unchanged — the cache write is outside the tx boundary.
- If two Accepts race on the same proposal, the loser's transaction returns `ErrProposalAlreadyAccepted` and the `aiUpdateToCache` call below it is bypassed (the `return err` paths short-circuit before we reach it). The winner's `aiUpdateToCache` runs and refreshes the cache.
- If an Accept and a manual `AIUpdate` race: both go through their respective DB writes under different transactions. Whichever commits second wins the DB. Whichever calls `aiUpdateToCache` second wins the cache. The two are independently consistent — the cache will eventually reflect the actual DB state because both writers refresh from the DB.
- A `tx.Commit()` followed by an `aiUpdateToCache` is technically a TOCTOU window: between commit and re-read, another writer could have advanced the AI. The cache will then briefly reflect that other writer's state — not stale, just newer than what this Accept produced. That's the correct behavior: caches should reflect the latest committed state, not the writer's mental model. No fix needed.

---

## Rollout

Single-file change to `bin-ai-manager`. Standard deploy:

1. Land PR.
2. CI runs `go mod tidy`, `go test`, `golangci-lint`. All must pass.
3. After merge, redeploy `ai-manager` to staging.
4. Verify the manual scenario above on staging.
5. Promote to prod.

No DB migration. No backwards-compatibility concern (the cache key shape is unchanged; we are only triggering an extra refresh).

---

## Risk

- **Risk: `aiUpdateToCache` itself has a bug.** Unlikely — three other write methods call it on every invocation and the system works.
- **Risk: cache thrash.** One extra `SELECT * FROM ai_ais WHERE id=?` per Accept. Accept is a low-frequency human-initiated operation (max 3 per customer concurrent, hard global cap of 30). Negligible.
- **Risk: hidden coupling.** `aiUpdateToCache` reads the AI from the DB using `aiGetFromDB`. If `aiGetFromDB` has ever been miscompiled or returns a partial row, the cache will be poisoned. But this is the convention the rest of the package follows, so we're inheriting any such risk rather than introducing new exposure.

Reverting is trivial — remove the three lines.

---

## Documentation Updates

### Service docs (`bin-ai-manager/docs/`)

Per the root CLAUDE.md service-docs-sync table, this change touches `pkg/dbhandler/aipromptproposal.go` only. None of the columns in that table list `pkg/dbhandler/*` as a trigger, so **no `docs/*.md` updates are required** for the service.

### RST docs (`bin-api-manager/docsdev/source/`)

This fix restores the documented behavior (Accept updates `init_prompt` and `current_prompt_history_id`) — it does not change any user-visible API contract. **No RST changes are required.**

### `bin-ai-manager/CLAUDE.md` — new "Cache invariants" section

The original feature shipped this bug because the design and three rounds of design review never noticed the cache-invalidation convention in `ai.go`. To prevent recurrence, this PR adds a new section to `bin-ai-manager/CLAUDE.md`:

```markdown
## Cache invariants

Any code path that writes to `ai_ais` — transactional or not — MUST call
`h.aiUpdateToCache(ctx, aiID)` after the write succeeds. See `dbhandler/ai.go`
(`AICreate`, `AIDelete`, `AIUpdate`) for the convention and
`dbhandler/aipromptproposal.go` (`AIAcceptProposal`) for the transactional case.

To audit: `grep -n 'UPDATE ai_ais\|INSERT INTO ai_ais' bin-ai-manager/pkg/dbhandler/*.go`
and confirm every match is followed by an `aiUpdateToCache` call.

The same pattern does NOT apply to `ai_ai_prompt_proposals` or
`ai_ai_prompt_histories` (those are not cached today).
```

Two paragraphs. Forces the next person who writes a new mutator to see the rule before they reproduce the bug.

## Resolved Open Questions (was: Open Questions for Reviewers)

- **"Should the cache invalidation be moved up to the handler layer?"** Resolved: no. Co-locating cache invariants with the writes that require them matches the existing `ai.go` convention and avoids forcing every future caller to remember.
- **"Should we add a defensive log line if `aiUpdateToCache` returns an error?"** Resolved: no. Match the silent-discard convention used by all four sibling call-sites (`ai.go` lines 41, 122, 148, 222). Operator-visibility regressions are caught by the AI's eventual cache miss → DB re-read, not by application logs.

---

## Alternative Considered: Cache invalidation at the handler layer

Put the cache invalidation in `aipromptproposalhandler.Accept` instead of inside `AIAcceptProposal`.

**Pros:**
- Keeps the transactional dbhandler method pure (DB only).
- Forces an explicit, visible cache invalidation step in the handler that future maintainers will see.

**Cons:**
- Diverges from the convention in `dbhandler/ai.go` where every write method invalidates its own cache.
- Adds a new exported method on the DBHandler interface (`AICacheRefresh(id)` or similar) — surface area growth for one caller.
- Future callers of `AIAcceptProposal` (e.g., a bulk-accept admin tool) would need to remember to invalidate the cache too, repeating the same bug class.

**Rejected** for the same reason `AIUpdate` does its own cache invalidation: the cache-invariant should be co-located with the write that requires it, not with one specific caller of that write.

---

## References

- Source: `bin-ai-manager/pkg/dbhandler/ai.go` lines 36-44 (AICreate), 128-151 (AIDelete), 196-225 (AIUpdate), 109-125 (AIGet)
- Source: `bin-ai-manager/pkg/dbhandler/aipromptproposal.go` lines 240-330 (AIAcceptProposal)
- Original feature spec: `docs/superpowers/specs/2026-05-29-ai-prompt-proposal-design.md`
- Incident timeline: api-manager pod `api-manager-866c48f8f7-vtfcm` logs at 2026-05-29 06:43–06:55 UTC for proposal `f97cb187-e572-49e9-88df-544833583415` on AI `74b3d982-8a5a-43b5-9c1d-51e6ce508b0a`.
