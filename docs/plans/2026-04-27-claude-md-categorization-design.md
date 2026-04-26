# CLAUDE.md and `docs/` categorization

**Date:** 2026-04-27
**Status:** Design (pre-implementation)
**Owner:** voipbin engineering
**Scope:** Root CLAUDE.md, `docs/`, pilot 2 service-level `CLAUDE.md` (full 30+ service audit deferred to a follow-up PR series)
**Migration strategy:** Single squash-merged PR (per Q5 decision), with a documented escape-hatch if review-time diff size demands a split

---

## 1. Problem

Documentation in this monorepo has accreted past the point where any single file is comfortably scannable.

- **Root `CLAUDE.md`: 746 lines.** Critical safety rails (verification workflow, no-merge-without-permission, squash-merge-only) coexist with ~150 lines of inline gotchas, ~30 lines of database rules, ~30 lines of WebhookMessage pattern, full git workflow examples, etc. Many of those sections are *also* present in `docs/coding-conventions.md`, with the root file declaring "duplicated here for high visibility."
- **`docs/coding-conventions.md`: 1578 lines** in **16 numbered sections** (verified). Declared as "the source of truth" for code patterns; covers everything from package structure to security in one buffer.
- **`docs/` is flat with 14 topic files.** Three are functional duplicates of conventions sections (`error-handling-patterns.md`, `database-patterns-checklist.md`, `test-utilities-guide.md`). One (`reference.md`, 231 lines) is itself a multi-topic grab-bag overlapping with both root CLAUDE.md and `coding-conventions.md`.
- **30+ service-level `bin-*-manager/CLAUDE.md` files** vary in length and structure; `docs/claude-md-template.md` exists but isn't enforced.
- **No bucket for cross-cutting infrastructure patterns.** The existing per-target circuit breaker in `bin-common-handler/pkg/circuitbreakerhandler/`, the new per-pod liveness ping pattern (PR #832), and the WebhookMessage pattern all live as inline sections in CLAUDE.md or buried inside `coding-conventions.md`. Future engineers either reinvent them or miss them entirely.

The next addition (the per-pod liveness ping that just merged in PR #832) makes this concrete: there is no obvious place to document the pattern as a reusable template. It currently lives only in its own `docs/plans/2026-04-26-pipecat-pod-ping-design.md`, which is feature-specific and won't be found by anyone solving a similar problem.

## 2. Goals & non-goals

### Goals
- Reorganize docs so future cross-cutting patterns have an obvious home.
- Reduce root `CLAUDE.md` to a focused safety-rails-plus-index file (~200-250 lines).
- Split the 1578-line `coding-conventions.md` into per-topic files of 50-200 lines each.
- Pilot the service-level `CLAUDE.md` audit on 2 high-traffic services (`bin-common-handler`, `bin-pipecat-manager`) to validate the template; full 30+ audit ships as a follow-up PR series.
- Land structural reorg + new patterns content as a single squash-merged PR.
- Ship a lightweight `scripts/check-docs.sh` CI check so the file doesn't regrow.

### Non-goals
- Rewriting content. The reorganization preserves wording wherever possible; new content (CB, liveness preflight, webhook-message-as-pattern) is additive.
- Moving `docs/plans/` (design docs and implementation plans). That convention is fine and stays.
- Touching `bin-api-manager/docsdev/source/` (RST-source customer-facing docs). Out of scope.
- Auditing all 30+ service `CLAUDE.md` files in this PR. Pilot 2; defer the rest.

## 3. Decisions log (from brainstorming)

| Q | Decision | Rationale |
|---|---|---|
| 1 | **B** — categorized subdirs in `docs/` | Scales as new patterns get added; lazy migration tends to stall. |
| 2 | **B** — critical-rails inline (~200 lines) | Saved feedback memory shows the rules left inline are exactly the ones that caused incidents when skipped. Reference material is fine one click away. |
| 3 | **B** — service `CLAUDE.md` stays self-contained, standardized to template | Engineers working in `bin-X` want service-specific knowledge inline; clicking out for "what's special about this service" is wrong UX. |
| 4 | **B** — split `coding-conventions.md` into `docs/conventions/<topic>.md` | A 1578-line single file is exactly the bloat problem we're solving at the top level. |
| 5 | **A** — single squash-merged PR | Phases 1-4 are interlocking; doing them sequentially means rewriting the same lines multiple times. Escape hatch documented in §11. |
| 2a | Decision tree stays inline in root CLAUDE.md | Every contributor needs it before writing new docs. |
| 2b | bin-common-handler admission rule stays in CRITICAL block | Deny-list rule; needs visibility before the offending code is written. |

## 4. Final directory structure

Filenames preserve current names where possible to keep `git log --follow` clean. Renames only when the new path is dramatically clearer.

```
monorepo/
├── CLAUDE.md                              ~200-250 lines: critical safety rails + index
├── docs/
│   ├── README.md                          one line: "see ../CLAUDE.md for the index"
│   │
│   ├── architecture/
│   │   ├── README.md
│   │   ├── architecture-deep-dive.md      MOVED from docs/architecture-deep-dive.md (filename preserved)
│   │   └── service-dependency-graph.md    MOVED from docs/service-dependency-graph.md
│   │
│   ├── conventions/                       SPLIT from docs/coding-conventions.md
│   │   ├── README.md                      reading order + topic links
│   │   ├── package-structure.md           §1 of coding-conventions
│   │   ├── naming.md                      §2
│   │   ├── imports.md                     §3
│   │   ├── error-handling.md              §4 + merges old docs/error-handling-patterns.md
│   │   ├── logging.md                     §5 + extracts debug + external-event logging from root CLAUDE.md
│   │   ├── models.md                      §6
│   │   ├── database.md                    §7 + merges old docs/database-patterns-checklist.md + extracts DB rules from root CLAUDE.md
│   │   ├── handlers.md                    §8
│   │   ├── rpc.md                         §9
│   │   ├── api-design.md                  §10 + extracts API Design Principles from root CLAUDE.md and reference.md
│   │   ├── events.md                      §11
│   │   ├── configuration.md               §12
│   │   ├── testing.md                     §13 + merges old docs/test-utilities-guide.md
│   │   ├── metrics.md                     §14
│   │   ├── security.md                    §15
│   │   └── direct-resource-types.md       §16 ("No Magic Strings for Direct Resource Types")
│   │
│   ├── workflows/
│   │   ├── README.md
│   │   ├── git-workflow-guide.md          MOVED from docs/git-workflow-guide.md (filename preserved)
│   │   ├── verification-workflows.md      MOVED from docs/verification-workflows.md
│   │   ├── common-workflows.md            MOVED from docs/common-workflows.md
│   │   ├── special-cases.md               MOVED from docs/special-cases.md
│   │   ├── development-guide.md           MOVED from docs/development-guide.md
│   │   └── common-gotchas.md              NEW — extracts the gotchas section (~150 lines) from root CLAUDE.md
│   │
│   ├── patterns/                          NEW DIRECTORY — applied infrastructure patterns with reference code
│   │   ├── README.md
│   │   ├── circuit-breaker.md             NEW — documents existing CB in bin-common-handler
│   │   ├── per-pod-liveness-preflight.md  NEW — pipecat ping pattern, "any response = alive"
│   │   ├── per-pod-queues.md              NEW — generalizes bin-*.request.<host_id> queue pattern
│   │   └── webhook-message.md             NEW — extracts WebhookMessage section (~30 lines) from root CLAUDE.md
│   │   (NOTE: debug-logging is NOT in patterns/. Style rules belong in conventions/logging.md.)
│   │
│   ├── reference/
│   │   ├── README.md
│   │   ├── rabbitmq-queues-reference.md   MOVED from docs/rabbitmq-queues-reference.md (filename preserved)
│   │   ├── claude-md-template.md          MOVED from docs/claude-md-template.md (refreshed for new doc paths)
│   │   └── code-quality-standards.md      MOVED from docs/code-quality-standards.md
│   │
│   └── plans/                             UNCHANGED
│       └── ... (existing design + plan docs)
│
├── scripts/check-docs.sh                  NEW — line-cap + per-category-README check
│
├── bin-common-handler/CLAUDE.md           AUDITED (pilot) — references new circuit-breaker pattern doc
├── bin-pipecat-manager/CLAUDE.md          AUDITED (pilot) — references new per-pod-queues pattern doc
└── ... (28 more bin-*-manager/CLAUDE.md UNCHANGED — follow-up audit PR series)
```

**`docs/reference.md` (231-line grab-bag) is DELETED**, not renamed. Its content distributes:
- API Design Principles → `docs/conventions/api-design.md`
- Key Dependencies → `docs/architecture/architecture-deep-dive.md`
- Deployment → already in `docs/architecture/architecture-deep-dive.md` (verify before delete)
- Important Notes / Common Gotchas → `docs/workflows/common-gotchas.md`
- Resources / Quick links → root CLAUDE.md `## Reference` section

**Net file count:**
- Today: ~45 docs (1 root CLAUDE.md + 14 docs/*.md + 30 service CLAUDE.md).
- After v1: ~50 docs (1 root + ~30 docs/<category>/<topic>.md including 6 README indices + 30 service CLAUDE.md, 2 audited).
- After full follow-up audits: ~50 docs (service file count unchanged; just standardized).

## 5. Root `CLAUDE.md` target shape (~200-250 lines)

Keep inline only what has documented incident history per the saved feedback memory.

```
# CLAUDE.md

## Scope
[1 paragraph: applies to all services; service-specific CLAUDE.md takes precedence on conflict]

## Overview
[1 paragraph: VoIPbin monorepo, 30+ Go microservices, RabbitMQ RPC, GKE]

## CRITICAL: Verification before commit
[5-step `go mod tidy && go mod vendor && go generate && go test && golangci-lint` block, verbatim]
[Why: caused full deployment outage when skipped — feedback_verification_bulk_changes.md]
→ Detail: docs/workflows/verification-workflows.md

## CRITICAL: Worktrees
[Never edit in main repo; worktree creation/removal commands, verbatim]
→ Detail: docs/workflows/git-workflow-guide.md

## CRITICAL: Branch & commit format
[Title = branch name; bulleted body with bin-* prefixes; no AI attribution]
[Never push to main; never merge without explicit user permission; squash-merge only]
[Pre-merge: fetch origin/main, conflict precheck, sync after merge]
→ Detail: docs/workflows/git-workflow-guide.md

## CRITICAL: Database via Alembic only
[AI can create/edit migration files; AI must never run upgrade/downgrade or alter schema]
[Always use `alembic revision` to generate IDs — never handpick]
→ Detail: bin-dbscheme-manager/CLAUDE.md

## CRITICAL: RST docs sync
[When user-visible feature changes: edit RST under bin-api-manager/docsdev/source/, clean rebuild HTML, force-add build/, commit both]
[Compare struct docs against WebhookMessage, not internal model]
→ Detail: docs/workflows/special-cases.md

## CRITICAL: bin-common-handler admission rule
[Package may live in bin-common-handler only when 3+ services consume it]
→ Detail: docs/conventions/package-structure.md

## Where to find things (index)
- Architecture          → docs/architecture/
- Coding conventions    → docs/conventions/
- Workflows             → docs/workflows/
- Shared patterns       → docs/patterns/
- Reference             → docs/reference/
- Plans (designs/plans) → docs/plans/

## Where to document new information
[The decision tree, kept here because every contributor needs it before writing new docs]

## Reference
[Quick links: admin console, agent interface, API docs, project site]
```

**What moves out** (no content changes — just relocate):

| Currently in root CLAUDE.md | Moves to |
|---|---|
| `### Database Handling Rules` (~30 lines) | `docs/conventions/database.md` |
| `### Debug Logging for Retrieved Data` (~20 lines) | `docs/conventions/logging.md` |
| `### External Event & Webhook Processing Logs` (~10 lines) | `docs/conventions/logging.md` |
| `### WebhookMessage Pattern for External API Responses` (~30 lines) | `docs/patterns/webhook-message.md` |
| `### Common Gotchas` (~150 lines, 4 sub-sections) | `docs/workflows/common-gotchas.md` |
| Detailed git workflow body (commit examples, branch strategies, merge rules) | `docs/workflows/git-workflow-guide.md` |
| `## Architecture` body | `docs/architecture/architecture-deep-dive.md` |
| `## API Design Principles` body | `docs/conventions/api-design.md` |
| `## Build & Development` body | `docs/workflows/development-guide.md` |

## 6. `coding-conventions.md` split mapping (verified against actual headers)

Source: `docs/coding-conventions.md`, 1578 lines, 16 numbered top-level sections (verified by `grep "^## "`).

| § | Current heading | Target file | Notes |
|---|---|---|---|
| 1 | Package Structure & File Organization | `docs/conventions/package-structure.md` | Includes detailed bin-common-handler 3-service admission rule |
| 2 | Naming Conventions | `docs/conventions/naming.md` | |
| 3 | Import Ordering | `docs/conventions/imports.md` | Could merge with naming.md if §3 is short |
| 4 | Error Handling | `docs/conventions/error-handling.md` | Absorbs old `docs/error-handling-patterns.md` |
| 5 | Logging | `docs/conventions/logging.md` | Absorbs the two logging blocks (debug, external-event) currently in root CLAUDE.md |
| 6 | Model Definitions | `docs/conventions/models.md` | |
| 7 | Database Patterns | `docs/conventions/database.md` | Absorbs old `docs/database-patterns-checklist.md` and DB rules from root CLAUDE.md |
| 8 | Handler Architecture | `docs/conventions/handlers.md` | |
| 9 | Inter-Service Communication | `docs/conventions/rpc.md` | |
| 10 | API & External Interfaces | `docs/conventions/api-design.md` | Absorbs `## API Design Principles` from root CLAUDE.md and overlapping content from `docs/reference.md` |
| 11 | Event Publishing | `docs/conventions/events.md` | |
| 12 | Configuration | `docs/conventions/configuration.md` | |
| 13 | Testing | `docs/conventions/testing.md` | Absorbs old `docs/test-utilities-guide.md` |
| 14 | Prometheus Metrics | `docs/conventions/metrics.md` | |
| 15 | Security | `docs/conventions/security.md` | |
| 16 | No Magic Strings for Direct Resource Types | `docs/conventions/direct-resource-types.md` | Could merge into models.md if short |

`docs/conventions/README.md` becomes the new "single source of truth" entry point — lists topics in a recommended reading order with a one-line summary each.

The original `docs/coding-conventions.md` is **deleted** in the same commit that creates the per-topic files (so the file's git history shows the deletion alongside the equivalent additions, making blame intent clear). `git log -- docs/coding-conventions.md` continues to work for historical lookup.

## 7. Pilot service `CLAUDE.md` audits

**Pilot scope (2 services in this PR):**
- `bin-common-handler/CLAUDE.md` — adds explicit reference to `docs/patterns/circuit-breaker.md` (the CB lives here, but isn't documented in its own service docs today).
- `bin-pipecat-manager/CLAUDE.md` — adds reference to `docs/patterns/per-pod-queues.md` and `docs/patterns/per-pod-liveness-preflight.md`. Strips any duplicated verification-workflow / git-rules content (use root CLAUDE.md as canonical).

**Audit checklist for the pilot (and future batches):**
1. Compare against `docs/reference/claude-md-template.md` — add missing required sections.
2. Strip duplicated content — remove anything copy-pasted from root CLAUDE.md.
3. Promote cross-cutting patterns — if a pattern in this service is also in another, move to `docs/patterns/<topic>.md` and replace inline copies with a 1-line link.
4. Refresh stale references — file paths to old `docs/<file>.md` updated to `docs/<category>/<topic>.md`.

**Follow-up audit batches (out of scope for v1):** Group remaining ~28 service CLAUDE.md files into 4-5 follow-up PRs of 5-7 services each. Each follow-up applies the same checklist; the pilot output serves as reference.

## 8. New shared-patterns content

Four new files in `docs/patterns/`. Pure additions — no existing content moves into them except the WebhookMessage section.

### 8.1 `docs/patterns/circuit-breaker.md`
Documents the existing `bin-common-handler/pkg/circuitbreakerhandler/`:
- Per-target (queue name) breaker state.
- Default failure threshold (5 consecutive) → `Closed → Open`.
- Default open duration (30 s) → `cb.Allow()` rejects immediately.
- Half-open probe after 30 s.
- Auto-integrated with every `r.sendRequest()` call (no opt-in).
- Free Prometheus metrics: `*_circuitbreaker_state{target}`, `_state_transitions_total`, `_rejected_total`.
- When NOT to add a new CB: don't, unless you're not using `r.sendRequest()`. The shared CB covers all RPC paths.
- When to tune the threshold: file an issue first; tuning is per-target, requires code change in `circuitbreakerhandler`. Default suits 90% of cases.

### 8.2 `docs/patterns/per-pod-liveness-preflight.md`
Generalizes the pattern introduced in PR #832 (bin-pipecat-manager `/v1/ping`):
- When to use: per-pod RPC where the consumer holds a `HostID` that may go stale (pod restart on K8s).
- Server side: lightweight `GET /v1/ping` on the per-pod queue, returns process identity. No DB I/O, no business logic.
- Client side: small helper that issues a 1 s preflight, distinguishes `ErrCircuitOpen` / `DeadlineExceeded` / other-error via `errors.Is` (relies on `pkg/errors v0.9.0+` Unwrap).
- Rules:
  - **Any response = alive.** Old pods returning 404 must be treated as alive (rolling-deploy compat). Do not add status-code checks.
  - **Best-effort `host_id` echo** detects routing bugs but does NOT detect Calico POD_IP recycle.
  - **Run preflight before any DB write** so a dead-pod failure doesn't orphan rows (lesson from PR #832 final review).
- Reference implementation: links to `docs/plans/2026-04-26-pipecat-pod-ping-design.md`.

### 8.3 `docs/patterns/per-pod-queues.md`
Documents the per-pod RabbitMQ queue convention:
- Queue name pattern: `<service>.request.<host_id>` (e.g., `bin-manager.pipecat-manager.request.<POD_IP>`).
- Declared as **volatile** (auto-delete when last consumer disconnects).
- Used when an RPC must reach a *specific* pod that owns in-memory state (e.g., a streaming session).
- Combined with shared queue (`<service>.request`) for stateless calls.
- `HostID` typically equals `POD_IP` from the K8s Downward API. Limitations: Calico recycles IPs within minutes — pair with the per-pod-liveness-preflight pattern above.
- Future option: store `POD_UID` for IP-recycle-safe identity (v2 candidate noted in PR #832).

### 8.4 `docs/patterns/webhook-message.md`
Extracts the `WebhookMessage` section from root CLAUDE.md (~30 lines), expands with concrete examples already present in `bin-api-manager/pkg/servicehandler/` callers.

**Excluded from `docs/patterns/`:**
- **debug-logging** — this is a style convention (when to log Debug vs Info), not an applied infrastructure pattern. Lives in `docs/conventions/logging.md` only. (Earlier draft v1 had this in patterns/; reviewer correctly flagged the type mismatch.)

## 9. CI guard: `scripts/check-docs.sh`

A 30-line shell script wired into the verification workflow (developer-runnable, optionally CI-runnable). Checks:

```bash
#!/usr/bin/env bash
# scripts/check-docs.sh — guards against root CLAUDE.md regrowth and missing category READMEs

set -euo pipefail

ROOT_CAP=350
ROOT_LINES=$(wc -l < CLAUDE.md)
if [[ $ROOT_LINES -gt $ROOT_CAP ]]; then
  echo "FAIL: root CLAUDE.md is $ROOT_LINES lines (cap $ROOT_CAP). Move detail to docs/<category>/<topic>.md."
  exit 1
fi

for category in architecture conventions workflows patterns reference; do
  if [[ ! -f "docs/$category/README.md" ]]; then
    echo "FAIL: docs/$category/README.md missing."
    exit 1
  fi
done

echo "OK: root CLAUDE.md = $ROOT_LINES lines, all category READMEs present."
```

Wired in v1 as a manual run in the verification workflow (`scripts/check-docs.sh` mentioned in `docs/workflows/verification-workflows.md`). Adding it to `.circleci/` or git pre-commit is a follow-up enhancement.

## 10. Cross-cutting policies

- **Every `docs/<category>/` has a `README.md`** (enforced by `scripts/check-docs.sh`).
- **Every `README.md` follows the same flat shape**: 1-paragraph purpose, file table (filename + 1-line description), nothing else. No long-form content in READMEs — they exist to navigate, not to teach.
- **Cross-references between docs** use repo-relative paths (`docs/conventions/database.md`), not URLs.
- **Plan docs** stay in `docs/plans/` with the `YYYY-MM-DD-<topic>-design.md` and `-plan.md` convention.
- **Service `CLAUDE.md` files** link out using the new `docs/<category>/<topic>.md` paths.
- **`docs/patterns/` admission criteria**: an applied infrastructure pattern with reference code, ideally consumed by 2+ services or an obvious candidate for that. Distinguish from style conventions in `docs/conventions/`.

## 11. Migration sequence (single PR, ordered for branch reviewability)

The squashed commit on `main` is one unit. The feature branch keeps logically grouped commits so reviewers can step through:

1. **Foundation** — Create `docs/<category>/` directory skeletons with empty README.md placeholders. Add `scripts/check-docs.sh`. Update the still-flat `docs/verification-workflows.md` to mention the new script — that edit then rides along when step 2 `git mv`s the file into `docs/workflows/`.
2. **Move existing flat docs** — `git mv` each existing `docs/*.md` into its target subdir, **preserving filenames** (no renames). This commit is pure renames so `git log --follow` is reliable.
3. **Update internal links** — Search-and-replace stale `docs/<file>.md` references in:
   - Root `CLAUDE.md` (will be rewritten anyway in step 6, but do it now for atomicity)
   - `bin-billing-manager/CLAUDE.md` (the only main-branch service file with such references — verified)
   - `.claude/scripts/check-error-log-return.sh` (verified)
   - Optional: historical `docs/plans/2026-03-04-coding-conventions-*.md` (low value — they're historical; leave or update). Recommendation: leave alone.
4. **Split `coding-conventions.md` and merge duplicate-topic files** — One commit creating all 16 `docs/conventions/<topic>.md` files; delete the original `docs/coding-conventions.md` in the same commit. The duplicate-topic files (`error-handling-patterns.md`, `database-patterns-checklist.md`, `test-utilities-guide.md`) were already moved into their target subdirs by step 2 — in this step their content is folded into the corresponding `docs/conventions/<topic>.md` and the moved originals are deleted, leaving exactly one canonical doc per topic.
5. **Add `docs/patterns/` content** — Four new files (`circuit-breaker.md`, `per-pod-liveness-preflight.md`, `per-pod-queues.md`, `webhook-message.md`) plus `docs/patterns/README.md`.
6. **Slim root `CLAUDE.md`** to the ~200-250 line target. Move duplicated sections out, update all `→ docs/<file>.md` links to `→ docs/<category>/<topic>.md`. Delete `docs/reference.md` (content distributed in step 4).
7. **Audit pilot service files** — `bin-common-handler/CLAUDE.md` and `bin-pipecat-manager/CLAUDE.md`.
8. **Final audit** — Run `scripts/check-docs.sh` locally, run `grep -rn "docs/coding-conventions\|docs/architecture-deep-dive\|..." --include="*.md"` once more to catch missed references.

Each commit on the feature branch follows the standard commit message format (title = branch name, bulleted body with `bin-*` / `docs:` / `scripts:` prefixes).

**Escape hatch:** if at code-review time the squash diff exceeds ~5000 lines or reviewers explicitly ask for a split, the natural break is between steps 1-3 (mechanical: directory + moves + link updates) and steps 4-7 (substantive: split + new content + slim + audit). Step 3 leaves the repo in a working state where everything still resolves. Document this option in the PR description.

## 12. Risk and rollback

- **External path references**: only 5 main-branch hits identified by audit (1 service `CLAUDE.md`, 1 CI script, 1 root CLAUDE.md, 2 historical plan docs). Step 3 covers them. Worktree branches for other in-flight features will have stale paths — those branches will need to rebase after the merge; note in the PR description for early warning.
- **`coding-conventions.md` blame continuity**: post-split, target file history starts at the split commit; source file history persists via `git log -- docs/coding-conventions.md` even after deletion. Acceptable — git itself doesn't follow one-to-many splits cleanly in any tool.
- **In-flight merge conflicts on root `CLAUDE.md`**: ~50 worktrees exist (per `git worktree list`). Most don't touch CLAUDE.md, but any that do will conflict massively after merge. Mitigation: post a heads-up in the team channel before opening the PR; encourage in-flight feature branches to either land first or hold rebases until after. Add to PR description.
- **Reviewer overload**: per Q5, the user accepted the single-PR risk. Escape hatch (split into mechanical + substantive) documented in §11.
- **Rollback**: revert the squash commit. All file moves are pure renames except step 4 (split) and steps 5-6 (additive + slim). Revert restores all original files; the only loss is new patterns content, which is additive and trivially re-introducible.

## 13. Open questions / future work

- **Full 30+ service `CLAUDE.md` audit** — follow-up PR series, ~5 services per PR. Pilot output (step 7) is the reference.
- **CI integration of `scripts/check-docs.sh`** — wire into `.circleci/` or git pre-commit hook. Follow-up.
- **Should `bin-api-manager/docsdev/` (customer-facing RST) follow a similar reorg?** Out of scope; tracked separately.
- **Service-level `bin-*/docs/` subdirectories?** Not in v1. Service-specific detail stays in service `CLAUDE.md`. Promoted to `docs/patterns/` when 3+ services share the pattern.

## 14. Decisions log (consolidated)

- Categorized subdirs in `docs/`.
- Root `CLAUDE.md` ~200-250 lines: critical-rails-inline, everything else linked.
- Service `CLAUDE.md` standardized to template; pilot 2 services in v1, full audit deferred.
- `docs/coding-conventions.md` split into 16 files in `docs/conventions/` (one per existing top-level section), original deleted.
- Single squash-merged PR; documented split escape-hatch.
- "Where to document new information" decision tree stays in root CLAUDE.md.
- bin-common-handler admission rule stays in root CRITICAL block.
- `docs/plans/` convention unchanged.
- `docs/reference.md` deleted (content distributes).
- Filenames preserved on move (no renaming during the moves) for `git log --follow` continuity.
- `scripts/check-docs.sh` ships in v1 as a manual check; CI wiring is follow-up.
- `debug-logging` lives only in `docs/conventions/logging.md`, not duplicated in `docs/patterns/`.
- `bin-api-manager/docsdev/` (customer-facing RST) out of scope.
