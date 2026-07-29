# Insight AI multi-config: implementation plan

**Date:** 2026-07-29
**Design:** `2026-07-29-insight-ai-multi-config-design.md` (approved, r4/r5 consecutive APPROVE)
**Branches:** `NOJIRA-Insight-ai-multi-config` in both `monorepo` and
`monorepo-javascript` (worktrees already created).

## Scope note

`bin-dbscheme-manager`'s CLAUDE.md prohibits AI-run `alembic upgrade`
/`downgrade` and hand-picked revision IDs — the migration file is
generated via `alembic revision` and committed, never applied by this
session.

## Step 1 — Migration (`bin-dbscheme-manager`)

- [ ] Run `alembic revision -m "ai_ais_insight_multi_config"` against
      current head (verify head is still `f4c4c2407cee` before running —
      design §1 "Open items" pin).
- [ ] `upgrade()`: add `is_insight_active BOOLEAN NOT NULL DEFAULT FALSE`;
      drop `uq_ai_active_insight_key` index + `active_insight_key` column;
      re-add `active_insight_key` generated column with the
      `is_insight_active = TRUE` condition; recreate the unique index.
      Guard each step with `_column_exists`/`_index_exists` per
      `f4c4c2407cee`'s existing pattern.
- [ ] `downgrade()`: mirror in reverse, restoring SQUARE-23's original
      generated-column definition (no `is_insight_active` condition) and
      dropping the new column. Add a code comment noting this is a
      one-way door once any customer has 2+ insight AIs (design §1).

## Step 2 — `bin-ai-manager` domain + dbhandler

- [ ] `models/ai/main.go`: add `IsInsightActive bool` (`db:"is_insight_active"`).
- [ ] `models/ai/field.go`: add `FieldIsInsightActive`.
- [ ] `models/ai/filters.go`: add `IsInsightActive bool` with
      `filter:"is_insight_active"` — **do not skip**, this is the fix for
      the listenhandler allowlist gap found in design review round 3.
- [ ] `models/ai/webhook.go`: add `IsInsightActive bool` to
      `WebhookMessage`, no `omitempty`.
- [ ] `scripts/database_scripts_test/table_ai_ais.sql`: add
      `is_insight_active` column and the SQLite `CASE WHEN` stand-in for
      the redefined generated column (mirrors `table_ai_aicalls.sql`'s
      existing pattern).
- [ ] `pkg/dbhandler/ai.go`:
  - New `AIActivateInsight(ctx, id uuid.UUID) (*ai.AI, error)` — follow
    `AIAcceptProposal`'s transaction pattern
    (`pkg/dbhandler/aipromptproposal.go:221-356`): `BeginTx` → `SELECT ...
    FOR UPDATE` scoped to the customer's currently-active insight row →
    clear it → set target row `TRUE` → `Commit` → deferred dual-row cache
    refresh via `aiUpdateToCache` with `context.Background()`.
  - `AIDelete`: add unconditional `is_insight_active = false` to its
    `SetMap`.
  - `AIUpdate`/`buildUpdateFields` (`pkg/aihandler/db.go`): when resolved
    `aiType != insight`, unconditionally include `is_insight_active =
    false` in the same update statement.
- [ ] `pkg/aihandler`:
  - New `ActivateInsight(ctx, id)` handler method — 400 if target
    `type != insight`, else delegate to dbhandler.
  - Remove `aiInsightDuplicateErr` translation call sites (`db.go:212`,
    `chatbot.go:78`, `chatbot.go:172`, `chatbot.go:199`) and the helper
    itself — dead once creates default inactive (design §2). Remove the
    four corresponding `chatbot_test.go` assertions (lines 1210, 1243,
    1277, 1311) and their enclosing test functions; check whether any
    shared `assertAlreadyExists`-style helper becomes unused afterward
    (would fail `golangci-lint`'s unused-code check) and remove it too if
    so.
  - Add new error reason `AI_INSIGHT_ACTIVATION_CONFLICT`, produced in
    `aihandler.ActivateInsight` (the new handler method, this same file)
    when `dbhandler.AIActivateInsight` returns an `IsErrDuplicate` error —
    this replaces the deleted `aiInsightDuplicateErr` helper's role for
    this one remaining reachable path.
- [ ] `pkg/listenhandler`: new route for
      `POST /ais/{id}/activate_insight`, unwrapped domain args in, `*ai.AI`
      out (style A).
- [ ] `docs/architecture.md` (bin-ai-manager): routing-table sync for the
      new route.
- [ ] `docs/domain.md` (bin-ai-manager): domain-entity sync for the
      `IsInsightActive` field addition to `models/ai/*.go` (root CLAUDE.md
      service-docs-sync table: `models/.../*.go` changes → `docs/domain.md`;
      the PostToolUse hook warns if this is skipped).

## Step 3 — Cross-service plumbing

- [ ] `bin-openapi-manager/openapi/paths/ais/`: new
      `POST /ais/{id}/activate_insight` path spec; add `is_insight_active`
      to the `AI` schema.
- [ ] `bin-common-handler/pkg/requesthandler`: new RPC client method for
      the activation call. This regenerates the shared mock consumed by
      `bin-api-manager` and other callers — run `bin-common-handler`'s
      `go generate ./...` (Step 6) before `bin-api-manager`'s, so the
      mock is current when api-manager's own tests compile against it.
- [ ] `bin-api-manager`: route + handler passthrough for the new endpoint,
      **including a customer-ownership check** in
      `pkg/servicehandler/ai.go` before calling the new activation RPC —
      follow this file's existing pattern for other AI methods (`AIGet`
      the target, verify its `customer_id` matches the requesting agent's
      customer, reject otherwise). Design §6 treats this as the primary
      authz layer and the dbhandler-level `FOR UPDATE` scoping as a
      backstop only — without this check here, a cross-customer
      activation is reachable at the API boundary.
- [ ] `bin-api-manager/pkg/servicehandler/ai.go` `convertAIFilters`: no
      change needed — it types against `amai.AI{}` reflectively, so
      Step 2's `models/ai/main.go` field addition already makes
      `is_insight_active` filterable at this hop. (The *other*, separate
      allowlist that does need an explicit change is
      `bin-ai-manager/models/ai/filters.go`, already listed in Step 2 —
      see design §2 for why both layers exist and only one needs a plan
      bullet here.)
- [ ] `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go`
      (`resolveInsightAIID`): switch to two-query resolution — size-1
      query with `is_insight_active=true, deleted=false, type=insight`
      first, fall back to existing 100-row most-recent-created query on
      empty result. Rewrite the function's stale doc comment
      (SQUARE-23-era "single AI" description).
- [ ] `bin-api-manager/docsdev/source/`: RST docs for the new endpoint and
      the `is_insight_active` field; clean rebuild
      (`rm -rf build && python3 -m sphinx -M html source build`);
      `git add -f` the build output.

## Step 4 — Frontend (`square-admin`)

- [ ] Insight AI list/card view (`ais_detail.js` area): "Active" badge,
      "Activate" button on inactive cards, optimistic update + rollback
      on failure.
- [ ] Zero-active-state affordance: when the dedicated active-query
      returns none, show "Currently used (no assistant activated yet)" on
      the fallback-selected (most-recent) card.
- [ ] English-only copy (admin default locale is English).
- [ ] No change needed to `TestAgentSheet.js` — verify only.
- [ ] Add the `POST /ais/{id}/activate_insight` API-client call (wherever
      this app's existing `ais` API client methods live) and the
      dedicated `GET /ais?type=insight&is_insight_active=true` query used
      by both the "Active" badge rendering and the zero-active-state
      affordance above.

## Step 5 — Tests (per design §6)

- [ ] `dbhandler`: dual-create-succeeds, activate-swaps-and-refreshes-cache,
      idempotent-reactivate, activate-soft-deleted-fails, normal-type
      unaffected, delete-clears-flag, type-change-clears-flag,
      concurrency test skip-gated like `aipromptproposal_test.go`.
- [ ] `aihandler`: `ActivateInsight` 400-on-normal-type + success path.
- [ ] `bin-api-manager`: resolution test (active wins, zero-active
      fallback, active-beyond-100-page still found).
- [ ] Filter-hop regression test: `GET /ais?is_insight_active=true`
      integration-level, confirming the filter survives
      `pkg/listenhandler/v1_ais.go`'s allowlist conversion (design round-3
      finding).
- [ ] `WebhookMessage`/RST field presence check.
- [ ] Cross-customer activation guard test at dbhandler level.
- [ ] square-admin manual/E2E per design §6.

## Step 6 — Verification (before any commit, per root CLAUDE.md)

For each modified Go service (`bin-ai-manager`, `bin-api-manager`,
`bin-openapi-manager`, `bin-common-handler`, run in each service dir):

```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

For `square-admin`: existing lint/type-check/test scripts (per
`package.json`), plus a manual browser check per design §6's E2E list
(local dev server).

`bin-dbscheme-manager` has no Go build to run, but syntax-check the new
migration file (`python3 -m py_compile <file>`) and confirm it lints
clean per that service's existing conventions — this session never runs
`alembic upgrade`/`downgrade` (root CLAUDE.md), so this static check is
the only verification available before merge.

**Rollback note:** Step 1's migration is a one-way door once applied to a
database with 2+ insight AIs on any customer (design §1). If a problem is
found post-merge but pre-apply, revert the PR. If found post-apply,
`downgrade()` is only safe while every customer still has ≤1 active-status
insight AI — do not run it blindly against a populated table; this is an
ops decision, not something this plan resolves.

## Step 7 — Code review loop

Minimum 3 rounds, 2 consecutive Approvals, per CLAUDE.md Review Loop
Policy (max 30). Reviewers: `code-reviewer` (general), plus one dedicated
pass with `security-reviewer` given this touches authz-adjacent
activation logic and a new cross-customer-scoped DB write.

## Step 8 — PR creation (no merge)

- [ ] Pull latest `origin/main` in both worktrees, check for conflicts
      (`git merge-tree`), resolve if any, re-run Step 6.
- [ ] Commit with `NOJIRA-Insight-ai-multi-config` title, project-prefixed
      body bullets (`bin-ai-manager:`, `bin-api-manager:`,
      `bin-openapi-manager:`, `bin-common-handler:`,
      `bin-dbscheme-manager:`, `square-admin:` as applicable).
- [ ] Push branch, open PR via `gh pr create` — title matches branch name,
      narrative summary + project-prefixed bullets, no "Test plan"
      section, no AI attribution. One PR per repo (`monorepo`,
      `monorepo-javascript`) since they're separate GitHub repos.
- [ ] **Do not merge.** Report PR URLs and stop; wait for 대표님's explicit
      merge instruction.
