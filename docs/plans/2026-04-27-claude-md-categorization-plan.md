# CLAUDE.md and `docs/` Categorization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reorganize the monorepo's documentation into categorized `docs/<category>/` subdirs, slim root `CLAUDE.md` from 746 to ~200-250 lines, split the 1578-line `coding-conventions.md` into 16 per-topic files, add four new `docs/patterns/` docs (circuit-breaker, per-pod-liveness-preflight, per-pod-queues, webhook-message), audit two pilot service `CLAUDE.md` files, and ship a `scripts/check-docs.sh` CI guard — all in one squash-merged PR.

**Architecture:** Pure documentation move + content reshape. No Go code changes, no DB changes, no CI/build changes beyond the new shell script. Filenames preserved on move so `git log --follow` works. The 16-section split is one-to-many (no `git mv` possible) but the original is deleted in the same commit so historical access remains via `git log -- docs/coding-conventions.md`.

**Tech Stack:** Markdown, Bash 5+ (`scripts/check-docs.sh`), `git mv` for moves, `grep` for path audits. No build, no test framework. Verification = "links resolve" + "check-docs.sh exits 0" + "grep finds no stale paths".

**Reference:** [`docs/plans/2026-04-27-claude-md-categorization-design.md`](./2026-04-27-claude-md-categorization-design.md)

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-claude-md-categorization`
**Branch:** `NOJIRA-claude-md-categorization` (already created; design doc already committed at `3419a9cf6`)

---

## Branch sync precheck (run before starting Task 1)

**Step 0.1: Pull latest main into the worktree base for conflict detection**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-claude-md-categorization
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
git log --oneline HEAD..origin/main | head -10
```

Expected: `no conflicts`. If any in-flight PR has touched root `CLAUDE.md` or `docs/coding-conventions.md`, rebase first; if conflicts, do not start the reorg until they're resolved.

**Step 0.2: Heads-up to the team (manual, not by Claude)**

Per design §12: post in the team channel that root `CLAUDE.md` and `docs/coding-conventions.md` are about to be heavily reorganized; in-flight feature branches that touch either should rebase after merge or hold their PRs until this lands. **This is a human action — Claude does not perform it.** The plan task only verifies that the user has done it before opening the final PR (Task 15).

---

## Task 1: Create category skeletons and CI guard

**Files:**
- Create: `docs/architecture/README.md`
- Create: `docs/conventions/README.md`
- Create: `docs/workflows/README.md`
- Create: `docs/patterns/README.md`
- Create: `docs/reference/README.md`
- Create: `docs/README.md`
- Create: `scripts/check-docs.sh`

**Step 1.1: Create the five category README placeholders**

Each README is a single line during this task — content gets filled out in later tasks as files move in.

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-claude-md-categorization
mkdir -p docs/architecture docs/conventions docs/workflows docs/patterns docs/reference scripts
for cat in architecture conventions workflows patterns reference; do
  printf '# %s\n\n> Placeholder — populated in later commits as docs migrate into this directory.\n' "$cat" > "docs/$cat/README.md"
done
```

**Step 1.2: Create top-level `docs/README.md`**

Content:

```markdown
# docs/

This directory holds VoIPbin engineering documentation. The index of categories and the "where to put new docs" decision tree live in [the root CLAUDE.md](../CLAUDE.md).

- `architecture/` — system shape, service interactions
- `conventions/` — coding standards (16 topic files, source of truth)
- `workflows/` — git, verification, deployment, common tasks, gotchas
- `patterns/` — applied infrastructure patterns (circuit breaker, per-pod liveness, etc.)
- `reference/` — quick-lookup tables (queues, glossary, templates)
- `plans/` — feature design docs and implementation plans (existing convention, unchanged)
```

**Step 1.3: Create `scripts/check-docs.sh`**

```bash
#!/usr/bin/env bash
# scripts/check-docs.sh — guards against root CLAUDE.md regrowth and missing category READMEs.
# Run from the monorepo root.

set -euo pipefail

ROOT_CAP=350
ROOT_LINES=$(wc -l < CLAUDE.md)
if [[ $ROOT_LINES -gt $ROOT_CAP ]]; then
  echo "FAIL: root CLAUDE.md is $ROOT_LINES lines (cap $ROOT_CAP). Move detail to docs/<category>/<topic>.md." >&2
  exit 1
fi

for category in architecture conventions workflows patterns reference; do
  if [[ ! -f "docs/$category/README.md" ]]; then
    echo "FAIL: docs/$category/README.md missing." >&2
    exit 1
  fi
done

echo "OK: root CLAUDE.md = $ROOT_LINES lines, all category READMEs present."
```

Make executable:

```bash
chmod +x scripts/check-docs.sh
```

**Step 1.4: Verify**

```bash
./scripts/check-docs.sh
```

Expected: `FAIL: root CLAUDE.md is 746 lines (cap 350). ...` (because root CLAUDE.md hasn't been slimmed yet — that's Task 10). The script *should* fail right now. Confirm the failure message is the line-cap one (not a missing README), then move on. The line cap will start passing in Task 10.

**Step 1.5: Commit**

```bash
git add docs/architecture/ docs/conventions/ docs/workflows/ docs/patterns/ docs/reference/ docs/README.md scripts/check-docs.sh
git status   # confirm only those paths staged
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: add category skeletons and check-docs.sh

- docs: Create docs/{architecture,conventions,workflows,patterns,reference}/README.md placeholders for the new category structure
- docs: Add docs/README.md top-level index
- scripts: Add check-docs.sh guarding root CLAUDE.md line cap and per-category README presence
EOF
)"
```

---

## Task 2: Move existing flat docs into subdirs (filenames preserved)

**Files moved (each `git mv` to preserve `git log --follow`):**

- `docs/architecture-deep-dive.md` → `docs/architecture/architecture-deep-dive.md`
- `docs/service-dependency-graph.md` → `docs/architecture/service-dependency-graph.md`
- `docs/git-workflow-guide.md` → `docs/workflows/git-workflow-guide.md`
- `docs/verification-workflows.md` → `docs/workflows/verification-workflows.md`
- `docs/common-workflows.md` → `docs/workflows/common-workflows.md`
- `docs/special-cases.md` → `docs/workflows/special-cases.md`
- `docs/development-guide.md` → `docs/workflows/development-guide.md`
- `docs/error-handling-patterns.md` → `docs/conventions/error-handling-patterns.md` (renamed in Task 5 when content folds into `error-handling.md`)
- `docs/database-patterns-checklist.md` → `docs/conventions/database-patterns-checklist.md` (folded in Task 5)
- `docs/test-utilities-guide.md` → `docs/conventions/test-utilities-guide.md` (folded in Task 5)
- `docs/rabbitmq-queues-reference.md` → `docs/reference/rabbitmq-queues-reference.md`
- `docs/claude-md-template.md` → `docs/reference/claude-md-template.md`
- `docs/code-quality-standards.md` → `docs/reference/code-quality-standards.md`

**Files NOT moved in this task:**
- `docs/coding-conventions.md` — split in Task 4.
- `docs/reference.md` — deleted in Task 11 after content distributes.

**Step 2.1: Run the moves**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-claude-md-categorization

git mv docs/architecture-deep-dive.md       docs/architecture/architecture-deep-dive.md
git mv docs/service-dependency-graph.md     docs/architecture/service-dependency-graph.md
git mv docs/git-workflow-guide.md           docs/workflows/git-workflow-guide.md
git mv docs/verification-workflows.md       docs/workflows/verification-workflows.md
git mv docs/common-workflows.md             docs/workflows/common-workflows.md
git mv docs/special-cases.md                docs/workflows/special-cases.md
git mv docs/development-guide.md            docs/workflows/development-guide.md
git mv docs/error-handling-patterns.md      docs/conventions/error-handling-patterns.md
git mv docs/database-patterns-checklist.md  docs/conventions/database-patterns-checklist.md
git mv docs/test-utilities-guide.md         docs/conventions/test-utilities-guide.md
git mv docs/rabbitmq-queues-reference.md    docs/reference/rabbitmq-queues-reference.md
git mv docs/claude-md-template.md           docs/reference/claude-md-template.md
git mv docs/code-quality-standards.md       docs/reference/code-quality-standards.md
```

**Step 2.2: Verify the renames**

```bash
git status --short
ls docs/
ls docs/architecture/ docs/conventions/ docs/workflows/ docs/reference/
```

Expected: 13 files renamed (`R` in `git status`); old `docs/*.md` files are gone except `coding-conventions.md` and `reference.md`. New subdir listings show the moved files.

**Step 2.3: Verify history is preserved**

```bash
git log --follow --oneline -3 docs/workflows/git-workflow-guide.md
```

Expected: shows the original commit that created `docs/git-workflow-guide.md`. If empty, the rename detection isn't working — investigate before continuing.

**Step 2.4: Commit**

```bash
git add -A docs/
git status   # only renames staged
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: move flat docs into category subdirs

- docs: git mv 13 existing flat docs/*.md into docs/{architecture,conventions,workflows,reference}/ — filenames preserved so git log --follow works for blame continuity. docs/coding-conventions.md and docs/reference.md left in place; they're handled in Tasks 4 and 11 respectively
EOF
)"
```

---

## Task 3: Update internal links to reflect the moves

**Files modified:**
- `CLAUDE.md` (root) — uses `docs/<file>.md` paths in many places; will be heavily rewritten in Task 10 but update now for atomicity
- `bin-billing-manager/CLAUDE.md` — verified main-branch reference
- `.claude/scripts/check-error-log-return.sh` — verified main-branch reference

**Step 3.1: Find every stale reference**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-claude-md-categorization
grep -rln "docs/architecture-deep-dive\|docs/git-workflow-guide\|docs/verification-workflows\|docs/common-workflows\|docs/special-cases\|docs/development-guide\|docs/error-handling-patterns\|docs/database-patterns-checklist\|docs/test-utilities-guide\|docs/rabbitmq-queues-reference\|docs/claude-md-template\|docs/code-quality-standards\|docs/service-dependency-graph" --include="*.md" --include="*.sh" --include="*.go" --include="*.yml" --include="*.yaml" 2>/dev/null | grep -v "/docs/plans/"
```

Expected: 3-5 file paths. If any are unexpected (e.g., a `.go` source file), stop and ask before mass-editing.

**Step 3.2: Update each reference**

For each match, rewrite the path. The mapping is mechanical — append the right subdir between `docs/` and the filename:

| Old | New |
|---|---|
| `docs/architecture-deep-dive.md` | `docs/architecture/architecture-deep-dive.md` |
| `docs/service-dependency-graph.md` | `docs/architecture/service-dependency-graph.md` |
| `docs/git-workflow-guide.md` | `docs/workflows/git-workflow-guide.md` |
| `docs/verification-workflows.md` | `docs/workflows/verification-workflows.md` |
| `docs/common-workflows.md` | `docs/workflows/common-workflows.md` |
| `docs/special-cases.md` | `docs/workflows/special-cases.md` |
| `docs/development-guide.md` | `docs/workflows/development-guide.md` |
| `docs/error-handling-patterns.md` | `docs/conventions/error-handling-patterns.md` (will become `error-handling.md` in Task 5) |
| `docs/database-patterns-checklist.md` | `docs/conventions/database-patterns-checklist.md` (will become `database.md` in Task 5) |
| `docs/test-utilities-guide.md` | `docs/conventions/test-utilities-guide.md` (will become `testing.md` in Task 5) |
| `docs/rabbitmq-queues-reference.md` | `docs/reference/rabbitmq-queues-reference.md` |
| `docs/claude-md-template.md` | `docs/reference/claude-md-template.md` |
| `docs/code-quality-standards.md` | `docs/reference/code-quality-standards.md` |

Note: `docs/coding-conventions.md` references stay unchanged in this task — that file still exists. They'll be updated to point into `docs/conventions/<topic>.md` in Task 4.

Use Edit tool per file (small number of files; safer than sed).

**Step 3.3: Re-grep to confirm zero stale paths remain (excluding plans and `docs/coding-conventions.md`)**

```bash
grep -rln "docs/architecture-deep-dive\|docs/git-workflow-guide\|docs/verification-workflows\|docs/common-workflows\|docs/special-cases\|docs/development-guide\|docs/error-handling-patterns\|docs/database-patterns-checklist\|docs/test-utilities-guide\|docs/rabbitmq-queues-reference\|docs/claude-md-template\|docs/code-quality-standards\|docs/service-dependency-graph" --include="*.md" --include="*.sh" --include="*.go" --include="*.yml" --include="*.yaml" 2>/dev/null | grep -v "/docs/plans/"
```

Expected: empty output.

**Step 3.4: Commit**

```bash
git add CLAUDE.md bin-billing-manager/CLAUDE.md .claude/scripts/check-error-log-return.sh
git status
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: update internal references to moved docs

- CLAUDE.md: Update references to docs/<file>.md to point at the new docs/<category>/<filename>.md paths (root CLAUDE.md gets a fuller rewrite in a later commit, this is the mechanical link refresh)
- bin-billing-manager: Update doc references to the new paths
- .claude: Update check-error-log-return.sh script references
EOF
)"
```

---

## Task 4: Split `coding-conventions.md` into 16 per-topic files

**Files:**
- Read source: `docs/coding-conventions.md` (1578 lines, 16 numbered top-level sections)
- Create 16 files in `docs/conventions/` (filenames per design §6)
- Delete: `docs/coding-conventions.md` after split

**Section → file mapping** (verified):

| § | Heading | Target filename |
|---|---|---|
| 1 | Package Structure & File Organization | `package-structure.md` |
| 2 | Naming Conventions | `naming.md` |
| 3 | Import Ordering | `imports.md` |
| 4 | Error Handling | `error-handling.md` |
| 5 | Logging | `logging.md` |
| 6 | Model Definitions | `models.md` |
| 7 | Database Patterns | `database.md` |
| 8 | Handler Architecture | `handlers.md` |
| 9 | Inter-Service Communication | `rpc.md` |
| 10 | API & External Interfaces | `api-design.md` |
| 11 | Event Publishing | `events.md` |
| 12 | Configuration | `configuration.md` |
| 13 | Testing | `testing.md` |
| 14 | Prometheus Metrics | `metrics.md` |
| 15 | Security | `security.md` |
| 16 | No Magic Strings for Direct Resource Types | `direct-resource-types.md` |

**Step 4.1: Read the source file end-to-end**

```bash
wc -l docs/coding-conventions.md
grep -n "^## " docs/coding-conventions.md
```

Expected: 1578 lines, 16 numbered headings (matches mapping above).

**Step 4.2: For each section, extract content into its target file**

For each `(start_line, end_line, target_file)` pair derived from the headers, do:

1. Extract the lines from `docs/coding-conventions.md`.
2. Write them to `docs/conventions/<target_file>` with a leading `# <Heading>` (drop the `## N. ` numbered prefix; replace with `# <Heading>` so the file is a self-contained doc).
3. Preserve all sub-headings (`### ...`), code blocks, tables, and links verbatim.
4. If the section references another section by number (e.g., "see §5 above"), update the link to reference the new file path (e.g., `see [Logging](logging.md)`).

This is a multi-step manual extraction. Prefer one file per micro-commit if convenient, but a single commit for all 16 is acceptable given the operation is mechanical.

**Step 4.3: Update `docs/conventions/README.md` to be the new index**

Replace the placeholder content with:

```markdown
# Conventions

This directory is the **single source of truth** for VoIPbin coding conventions. It supersedes the previous monolithic `docs/coding-conventions.md`.

Recommended reading order for new engineers:

1. [package-structure.md](package-structure.md) — file layout and the bin-common-handler 3-service admission rule
2. [naming.md](naming.md) — Go naming conventions
3. [imports.md](imports.md) — import grouping and ordering
4. [error-handling.md](error-handling.md) — wrap, log, propagate
5. [logging.md](logging.md) — Debug for retrieved data, Info for external events
6. [models.md](models.md) — struct definitions, field tags, JSON marshaling
7. [database.md](database.md) — squirrel + commondatabasehandler + dbhandler-only access
8. [handlers.md](handlers.md) — handler interface + struct + constructor pattern
9. [rpc.md](rpc.md) — inter-service RabbitMQ RPC
10. [api-design.md](api-design.md) — atomic responses, WebhookMessage pattern reference
11. [events.md](events.md) — pub/sub
12. [configuration.md](configuration.md) — env vars, viper
13. [testing.md](testing.md) — mockgen, table-driven tests, coverage targets
14. [metrics.md](metrics.md) — Prometheus naming, no name collisions with shared library
15. [security.md](security.md) — secrets, input validation, authorization
16. [direct-resource-types.md](direct-resource-types.md) — no magic strings for direct resource types
```

**Step 4.4: Delete the original**

```bash
git rm docs/coding-conventions.md
```

**Step 4.5: Verify**

```bash
ls docs/conventions/
wc -l docs/conventions/*.md
grep -rln "docs/coding-conventions" --include="*.md" --include="*.sh" --include="*.go" --include="*.yml" --include="*.yaml" 2>/dev/null | grep -v "/docs/plans/"
```

Expected: 17 files in `docs/conventions/` (16 topic files + 1 README); per-file line counts roughly correlate with section sizes; no stale `docs/coding-conventions` references outside `docs/plans/` (historical plan docs may keep them — leave alone).

**Step 4.6: Commit**

```bash
git add docs/conventions/ 
git rm docs/coding-conventions.md
git status
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: split coding-conventions.md into per-topic files

- docs: Split docs/coding-conventions.md into 16 per-topic files in docs/conventions/, one per existing top-level section (package-structure, naming, imports, error-handling, logging, models, database, handlers, rpc, api-design, events, configuration, testing, metrics, security, direct-resource-types)
- docs: Add docs/conventions/README.md as the new source-of-truth index with recommended reading order
- docs: Delete the original docs/coding-conventions.md; historical access remains via git log -- docs/coding-conventions.md
EOF
)"
```

---

## Task 5: Merge duplicate-topic files and delete the originals

The duplicate-topic files (`error-handling-patterns.md`, `database-patterns-checklist.md`, `test-utilities-guide.md`) were moved to `docs/conventions/` in Task 2 with their original filenames. In this task they get folded into the canonical conventions files and deleted.

**Step 5.1: Fold `error-handling-patterns.md` into `error-handling.md`**

Read both files. For any content in `error-handling-patterns.md` that's NOT already in `error-handling.md` (e.g., extra examples, edge cases, anti-patterns), add it to `error-handling.md` in an appropriate section (typically a new `## Examples` or `## Anti-patterns` block). Then `git rm docs/conventions/error-handling-patterns.md`.

**Step 5.2: Fold `database-patterns-checklist.md` into `database.md`**

Same approach. The `-checklist.md` file is likely a checkbox-style summary; add it as a `## Checklist` section in `database.md` if not already covered, then delete the source.

**Step 5.3: Fold `test-utilities-guide.md` into `testing.md`**

Same approach. The utilities guide is likely a how-to-use the test helpers; merge as `## Test Utilities` section in `testing.md` if missing, then delete the source.

**Step 5.4: Re-grep for stale references to the deleted files**

```bash
grep -rln "docs/conventions/error-handling-patterns\|docs/conventions/database-patterns-checklist\|docs/conventions/test-utilities-guide\|docs/error-handling-patterns\|docs/database-patterns-checklist\|docs/test-utilities-guide" --include="*.md" --include="*.sh" --include="*.go" --include="*.yml" --include="*.yaml" 2>/dev/null | grep -v "/docs/plans/"
```

Expected: empty (any references should now point to `docs/conventions/{error-handling,database,testing}.md`).

**Step 5.5: Verify scripts/check-docs.sh still passes the categories**

```bash
./scripts/check-docs.sh || true   # will still fail on root CLAUDE.md size (expected until Task 10); confirm the error is the line-cap one
```

**Step 5.6: Commit**

```bash
git add docs/conventions/
git rm docs/conventions/error-handling-patterns.md docs/conventions/database-patterns-checklist.md docs/conventions/test-utilities-guide.md 2>/dev/null || true
git status
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: fold duplicate-topic files into canonical conventions docs

- docs: Merge content from error-handling-patterns.md into docs/conventions/error-handling.md and delete the source
- docs: Merge content from database-patterns-checklist.md into docs/conventions/database.md and delete the source
- docs: Merge content from test-utilities-guide.md into docs/conventions/testing.md and delete the source
EOF
)"
```

---

## Task 6: Add `docs/patterns/circuit-breaker.md`

**Files:**
- Create: `docs/patterns/circuit-breaker.md`

**Step 6.1: Read the existing CB code so the doc cites real files and line ranges**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-claude-md-categorization
ls bin-common-handler/pkg/circuitbreakerhandler/
cat bin-common-handler/pkg/circuitbreakerhandler/option.go
grep -n "cb.Allow\|cb.RecordSuccess\|cb.RecordFailure" bin-common-handler/pkg/requesthandler/send_request.go
```

Confirm: `defaultFailureThreshold = 5`, `defaultOpenDuration = 30 * time.Second`, integration in `send_request.go:50-67`.

**Step 6.2: Write the doc**

Content (verbatim — no need to invent):

```markdown
# Circuit Breaker

VoIPbin has a per-target (per-RabbitMQ-queue) circuit breaker built into every `r.sendRequest()` call in `bin-common-handler/pkg/requesthandler/`. You almost never need to add your own retry / timeout / fast-fail logic — the shared CB covers all RPC paths.

## Where it lives

- Implementation: `bin-common-handler/pkg/circuitbreakerhandler/`
- Integration point: `bin-common-handler/pkg/requesthandler/send_request.go:50-67`
- Defaults: `bin-common-handler/pkg/circuitbreakerhandler/option.go`
  - `defaultFailureThreshold = 5` (consecutive failures before opening)
  - `defaultOpenDuration = 30 * time.Second` (window during which requests fast-fail)

## State machine

- **Closed** — `cb.Allow()` returns nil; the underlying RPC runs.
- **Open** — `cb.Allow()` returns `ErrCircuitOpen` immediately; no RPC is issued. Triggered by 5 consecutive failures recorded against the same target.
- **Half-open** — entered automatically 30s after Open. One probe is allowed; success → Closed, failure → Open for another 30s.

State is per-target (per RabbitMQ queue name). One dead pipecat-manager pod's per-pod queue trips its own breaker without affecting the shared `pipecat-manager.request` queue or any other service.

## Free Prometheus metrics

The CB auto-exports per-target metrics on the namespace of the consuming service (e.g., `ai_manager_*`):

- `<namespace>_circuitbreaker_state{target="<queue>"}` — gauge: 0 closed, 1 open, 2 half-open
- `<namespace>_circuitbreaker_state_transitions_total{target,from,to}` — counter
- `<namespace>_circuitbreaker_rejected_total{target}` — counter incremented on each Open-state rejection

Add these to your service Grafana dashboard if you handle critical RPC paths.

## When NOT to add a new circuit breaker

Don't, unless your RPC path bypasses `r.sendRequest()`. The vast majority of services route through it; just call your `r.<Service>V1<Method>(...)` and you get the breaker for free.

## When to tune the threshold

The defaults (5 failures / 30s) suit ~90% of cases. If telemetry shows you need a more aggressive threshold for a specific target (e.g., per-pod queues that should fast-fail after one timeout), file an issue first; tuning is currently per-target and requires a change inside `circuitbreakerhandler/`. Don't roll a parallel breaker — extend the shared one.

## Reference: how PR #832 leveraged this

The per-pod liveness preflight in PR #832 (see `docs/patterns/per-pod-liveness-preflight.md`) routes its 1-second `PipecatV1Ping` through `r.sendRequest()` so 5 consecutive ping failures against the same dead `HostID` open the breaker, giving 30 seconds of microsecond fast-fail with zero new state. That design choice — "ping is just another sendRequest" — let v1 ship without a new CB or cache.
```

**Step 6.3: Verify**

```bash
ls -la docs/patterns/circuit-breaker.md
wc -l docs/patterns/circuit-breaker.md
```

Expected: file exists, ~50 lines.

**Step 6.4: Commit**

```bash
git add docs/patterns/circuit-breaker.md
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: document the per-target circuit breaker pattern

- docs: Add docs/patterns/circuit-breaker.md describing the existing bin-common-handler/pkg/circuitbreakerhandler — defaults, state machine, integration point in r.sendRequest, free Prometheus metrics, when NOT to add a parallel breaker, and a reference to PR #832 as the canonical leverage example
EOF
)"
```

---

## Task 7: Add `docs/patterns/per-pod-liveness-preflight.md`

**Files:**
- Create: `docs/patterns/per-pod-liveness-preflight.md`

**Step 7.1: Write the doc, citing PR #832 design**

Pull the rules from `docs/plans/2026-04-26-pipecat-pod-ping-design.md`. Content:

```markdown
# Per-pod Liveness Preflight

When a per-pod RPC's stored `HostID` may go stale (e.g., the target pod restarts on K8s and gets a new IP), gate the call with a sub-second preflight `/v1/ping` against the per-pod queue. This pattern was introduced in PR #832 for `bin-pipecat-manager` and applies to any future per-pod service.

## When to use

- The consumer holds a `HostID` (typically `POD_IP`) that points to a specific pod owning in-memory state.
- The pod may die between the time the `HostID` was stored and the time it's used.
- The cost of a 3s RabbitMQ RPC timeout per failed call is unacceptable on a customer hot path.

## Server side

Add a lightweight `GET /v1/ping` route on the per-pod queue. Return process identity only — no DB I/O, no business logic.

```go
type PingResult struct {
    HostID    string    `json:"host_id"`
    Timestamp time.Time `json:"timestamp"`
}

func (h *handler) Ping(ctx context.Context) (*PingResult, error) {
    return &PingResult{HostID: h.hostID, Timestamp: time.Now().UTC()}, nil
}
```

Wire into the listenhandler's per-pod queue route table.

## Client side

Add a `XxxV1Ping(ctx, hostID) error` method to `bin-common-handler/pkg/requesthandler/` (alongside the existing per-service RPCs). 1-second timeout. Routes through `r.sendRequest()` so it shares the existing per-target circuit breaker (see [circuit-breaker.md](circuit-breaker.md)).

In the consumer service, add a small helper that distinguishes error classes via `errors.Is` (relies on `pkg/errors v0.9.0+` Unwrap):

```go
func (h *handler) pingHost(ctx context.Context, hostID string) bool {
    if hostID == "" {
        return false
    }
    cctx, cancel := context.WithTimeout(ctx, 1100*time.Millisecond)
    defer cancel()
    err := h.reqHandler.XxxV1Ping(cctx, hostID)
    if err == nil {
        return true
    }
    switch {
    case errors.Is(err, circuitbreakerhandler.ErrCircuitOpen):
        // CB is already open; fast-fail
    case errors.Is(err, context.DeadlineExceeded):
        // pod is dead
    default:
        // broker / transport error — distinguish from pod death
    }
    return false
}
```

## Rules

1. **Any response = alive.** Old pods returning 404 (because they predate the route) must be treated as alive — they responded. Do not add status-code checks. The only "dead" signal is `err != nil` (timeout or `ErrCircuitOpen`).
2. **Best-effort `host_id` echo.** Compare the response `HostID` to the requested one to catch routing bugs. This does NOT detect Calico POD_IP recycle (where a different pod takes the dead pod's IP).
3. **Run preflight before any DB write.** A dead-pod failure should not orphan rows. Lesson from PR #832 final review: persisting the message before the ping created orphan rows when the pod was dead.
4. **Outer 1.1s context wraps inner 1s RPC.** The inner sendRequest enforces the 1s hard timeout; the outer 100ms slack is a safety net for uncancellable upstream contexts.

## What this pattern does NOT solve

- **Calico POD_IP recycle.** Within minutes, a dead pod's IP may be reassigned to a new pod that responds to the ping. The `host_id` echo matches (both equal POD_IP) so detection fails. Fallback: the new pod's listenhandler returns 4xx because the session isn't in its session map; one wasted real RPC. Same as today's behavior. v2 candidate: store `POD_UID` for IP-recycle-safe identity.
- **Session-level liveness.** This is process-level only — the pod is up, but the specific session may have been cleaned up in-memory. Treat as the same edge case as Calico recycle.

## Reference implementation

- Design: `docs/plans/2026-04-26-pipecat-pod-ping-design.md`
- Code: `bin-pipecat-manager/pkg/listenhandler/v1_ping.go`, `bin-common-handler/pkg/requesthandler/pipecat_ping.go`, `bin-ai-manager/pkg/aicallhandler/ping.go`
```

**Step 7.2: Verify and commit**

```bash
wc -l docs/patterns/per-pod-liveness-preflight.md
git add docs/patterns/per-pod-liveness-preflight.md
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: document the per-pod liveness preflight pattern

- docs: Add docs/patterns/per-pod-liveness-preflight.md generalizing the PR #832 pipecat ping pattern — server-side /v1/ping shape, client-side helper with errors.Is differentiation, the four hard rules (any response = alive, best-effort echo, preflight-before-DB-write, double-timeout layering), known limitations (Calico recycle, session loss), and reference implementation citations
EOF
)"
```

---

## Task 8: Add `docs/patterns/per-pod-queues.md`

**Step 8.1: Write the doc**

Content:

```markdown
# Per-pod RabbitMQ Queues

Some services need to route RPCs to a *specific* pod that owns in-memory state (e.g., a streaming session, an audio bridge). VoIPbin handles this with per-pod queues alongside the standard shared per-service queue.

## Naming convention

```
<service>.request                       — shared queue, any consumer wins
<service>.request.<host_id>             — per-pod queue, owned by exactly one pod
```

Example from `bin-pipecat-manager`:
- `bin-manager.pipecat-manager.request` — call creation, lookup, etc.
- `bin-manager.pipecat-manager.request.<POD_IP>` — message-send, terminate, ping

## Declaration

Per-pod queues MUST be declared as **volatile** (auto-delete when the last consumer disconnects). The volatility is what guarantees the queue disappears when its owning pod dies, so a publisher can detect death by RPC timeout.

```go
sockHandler.QueueCreate(queue, "volatile")
```

Shared queues use `"normal"` (durable) declaration.

## Identity source

`HostID` is set at pod startup from the K8s Downward API — typically `POD_IP`:

```yaml
env:
- name: POD_IP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
```

Then in `cmd/<service>/main.go`:

```go
listenIP := os.Getenv("POD_IP")
if listenIP == "" {
    return fmt.Errorf("could not get the listen ip address")
}
listenQueue := fmt.Sprintf("%s.%s", commonoutline.QueueName<Service>Request, listenIP)
```

Persist this `HostID` on the resource (e.g., `pipecatcall.HostID`) so the consumer service can route follow-up RPCs to the right pod.

## Limitations

- **Calico POD_IP recycle.** Calico CNI reassigns released pod IPs within minutes. A stored `HostID` may resolve to a different pod after a restart. Pair this pattern with [per-pod-liveness-preflight.md](per-pod-liveness-preflight.md) to detect dead pods quickly, but be aware that recycle gives a matching IP and bypasses the simple echo check. v2 candidate: store `POD_UID` (immutable across IP recycle) instead of or alongside `POD_IP`.
- **No load balancing.** A per-pod queue routes to one pod by definition; if that pod is busy or backlogged, the request waits. Use the shared queue for stateless calls and reserve per-pod for genuine session affinity.
- **No persistence.** Volatile queues lose messages if the pod dies before consuming them. Acceptable for the use case (the session is also gone), but don't use this pattern for important non-session work.

## When to use

Use per-pod queues when **all** of these are true:
1. Some operation must reach a specific pod that holds in-memory state.
2. That state cannot be reconstructed cheaply on another pod.
3. You have a way to handle pod death (typically the liveness preflight pattern + caller error surfacing).

If the state is reconstructable from DB or shared cache, use the shared queue and let any pod handle the request.

## Reference implementations

- `bin-pipecat-manager/cmd/pipecat-manager/main.go:116-120` (HostID = POD_IP wiring)
- `bin-pipecat-manager/pkg/listenhandler/main.go` (volatile per-pod queue declaration)
- `bin-common-handler/pkg/requesthandler/pipecat_message.go` (per-pod RPC routing)
```

**Step 8.2: Verify and commit**

```bash
wc -l docs/patterns/per-pod-queues.md
git add docs/patterns/per-pod-queues.md
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: document the per-pod RabbitMQ queue pattern

- docs: Add docs/patterns/per-pod-queues.md describing the <service>.request.<host_id> naming convention, volatile declaration requirement, POD_IP-from-Downward-API identity source, Calico recycle and no-persistence limitations, when-to-use criteria, and reference implementation citations from bin-pipecat-manager
EOF
)"
```

---

## Task 9: Add `docs/patterns/webhook-message.md`

**Step 9.1: Read the WebhookMessage section currently in root CLAUDE.md (~30 lines)**

```bash
grep -n "WebhookMessage Pattern" CLAUDE.md
```

Read the full block (use Read tool on the surrounding lines).

**Step 9.2: Write the doc, expanding with bin-api-manager examples**

Content shape (paste the existing CLAUDE.md content verbatim, then add an `## Examples` section pointing at `bin-api-manager/pkg/servicehandler/` callers — read a couple of those files to confirm the example shape is current):

```markdown
# WebhookMessage Pattern for External API Responses

[VERBATIM COPY of the WebhookMessage Pattern section currently in root CLAUDE.md]

## Examples in the codebase

- `bin-api-manager/pkg/servicehandler/speaking.go` — `SpeakingGet` returning `*WebhookMessage`
- `bin-api-manager/pkg/servicehandler/customer_signup.go` — `SignupResultWebhookMessage` compound result
- `bin-talk-manager/models/message/webhook.go` — typical `webhook.go` neighbor file

## Why the pattern exists

Internal model structs may carry infrastructure fields (e.g., `PodID`, `Username`, `PermissionIDs`) that must not leak to external clients. Returning the internal struct directly from an HTTP handler exposes those fields by default. `WebhookMessage` is the explicit public-facing representation; `ConvertWebhookMessage()` is the explicit narrowing.

This pattern pairs with the RST struct documentation rule: `*_struct_*.rst` docs MUST describe `WebhookMessage` fields, not the internal model.
```

**Step 9.3: Verify and commit**

```bash
wc -l docs/patterns/webhook-message.md
git add docs/patterns/webhook-message.md
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: document the WebhookMessage external-API-response pattern

- docs: Add docs/patterns/webhook-message.md extracting the existing WebhookMessage section from root CLAUDE.md and expanding it with concrete examples from bin-api-manager/pkg/servicehandler/ callers and the RST documentation rule pairing
EOF
)"
```

---

## Task 10: Slim root `CLAUDE.md` to ~200-250 lines

**Files:**
- Modify: `CLAUDE.md` (root)

This is the largest single content edit. Approach: rewrite the file from scratch following the §5 target shape in the design doc. Preserve every CRITICAL safety rail verbatim; everything else becomes a one-paragraph stub with a `→ docs/<category>/<file>.md` link.

**Step 10.1: Read the current CLAUDE.md end-to-end** (Read tool on the whole file).

**Step 10.2: Rewrite using the §5 target shape**

The target structure is in `docs/plans/2026-04-27-claude-md-categorization-design.md` §5. Copy the structure verbatim. Fill in:

- `Scope` — copy the current one-paragraph version.
- `Overview` — copy the current one-paragraph version.
- 7 CRITICAL blocks — copy the current verbatim text for each (verification workflow, worktrees, branch/commit, Alembic, RST sync, common-handler admission), with `→ docs/<category>/<file>.md` link added at the end of each block.
- `Where to find things (index)` — verbatim from §5.
- `Where to document new information` — copy the current decision tree section.
- `Reference` — copy the current quick-links list.

Drop entirely (now lives in topic docs):
- `### Database Handling Rules` (now in `docs/conventions/database.md`)
- `### Debug Logging for Retrieved Data` (now in `docs/conventions/logging.md`)
- `### External Event & Webhook Processing Logs` (now in `docs/conventions/logging.md`)
- `### WebhookMessage Pattern for External API Responses` (now in `docs/patterns/webhook-message.md`)
- `### Common Gotchas` and all sub-sections (now in `docs/workflows/common-gotchas.md` — see Task 11)
- Detailed `## Build & Development` body (now in `docs/workflows/development-guide.md`)
- Detailed `## Architecture` body (now in `docs/architecture/architecture-deep-dive.md`)
- Detailed `## API Design Principles` body (now in `docs/conventions/api-design.md`)

**Step 10.3: Verify line count and check-docs.sh passes**

```bash
wc -l CLAUDE.md
./scripts/check-docs.sh
```

Expected: ~200-250 lines; script exits 0 with `OK: root CLAUDE.md = N lines, all category READMEs present.`

**Step 10.4: Spot-check the CRITICAL blocks**

Read the rewritten file. Confirm each of the 7 CRITICAL blocks is present and the wording matches the original (no accidental softening). The 7 blocks are: Verification, Worktrees, Branch & commit format, Database via Alembic, RST docs sync, bin-common-handler admission rule, plus the implicit one in the index header callout.

**Step 10.5: Commit**

```bash
git add CLAUDE.md
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: slim root CLAUDE.md to critical safety rails plus index

- CLAUDE.md: Rewrite root CLAUDE.md to keep only the seven CRITICAL safety-rail blocks (verification workflow, worktrees, branch/commit format, Alembic-only DB changes, RST docs sync, bin-common-handler admission rule) plus the where-to-find-things index, the where-to-document decision tree, and the quick-link reference; everything else moved into topic docs in earlier commits
- CLAUDE.md: Drops from 746 to ~200-250 lines, enforced by scripts/check-docs.sh going forward
EOF
)"
```

---

## Task 11: Move common gotchas, distribute `docs/reference.md`, then delete it

**Files:**
- Create: `docs/workflows/common-gotchas.md` (extract the gotchas section from the current/previous root CLAUDE.md history)
- Distribute and delete: `docs/reference.md`

**Step 11.1: Recover the gotchas content**

The `### Common Gotchas` section was dropped in Task 10. Recover it from git:

```bash
git show HEAD~1:CLAUDE.md > /tmp/old-claude.md
grep -n "### Common Gotchas" /tmp/old-claude.md
# Read the surrounding ~150 lines, copy into docs/workflows/common-gotchas.md
```

Write `docs/workflows/common-gotchas.md`:

```markdown
# Common Gotchas

Hard-won lessons from production incidents. Keep this list short and high-signal — the ones that have actually bitten engineers.

[Verbatim copy of the four sub-sections from the old root CLAUDE.md ### Common Gotchas:
 - Updating Shared Library Function Signatures
 - Prometheus Metric Name Conflicts
 - UUID Fields and DB Tags
 - Model/Struct Changes Require OpenAPI Updates
 - Feature Changes Require RST Documentation Updates]
```

**Step 11.2: Distribute `docs/reference.md` content**

Read `docs/reference.md` end-to-end. Map each section:

| Section in reference.md | Target |
|---|---|
| `## API Design Principles` (and `### Atomic API Responses`) | Already covered in `docs/conventions/api-design.md` (no action). |
| `## Key Dependencies` (### All Services, ### Common Tools, ### API Gateway Specific, ### Cloud Integration) | Append to `docs/architecture/architecture-deep-dive.md` as a new `## Key Dependencies` section (or merge into existing dependencies content if duplicated). |
| `## Deployment` (### Kubernetes, ### Infrastructure Requirements) | Append to `docs/architecture/architecture-deep-dive.md` as `## Deployment` if not already there. |
| `## Important Notes` (### Monorepo-Specific Practices, ### Communication Patterns, ### Code Quality, ### Common Gotchas) | These are summaries — likely covered by `docs/workflows/common-gotchas.md` and `docs/conventions/`. Drop unless unique content exists. |
| `## Security Considerations` | Already covered in `docs/conventions/security.md`. Drop. |
| `## Resources` | Already in root CLAUDE.md `## Reference`. Drop. |

For each "append" target, be careful not to duplicate existing content. Read the target file, decide if reference.md adds anything; only append the genuinely-new bits.

**Step 11.3: Delete `docs/reference.md`**

```bash
git rm docs/reference.md
```

**Step 11.4: Final grep audit**

```bash
grep -rln "docs/reference\.md\|coding-conventions\.md" --include="*.md" --include="*.sh" --include="*.go" --include="*.yml" --include="*.yaml" 2>/dev/null | grep -v "/docs/plans/"
```

Expected: empty (any historical references in `docs/plans/` are fine to leave).

**Step 11.5: Run check-docs.sh**

```bash
./scripts/check-docs.sh
```

Expected: PASS.

**Step 11.6: Commit**

```bash
git add docs/workflows/common-gotchas.md docs/architecture/architecture-deep-dive.md
git rm docs/reference.md
git status
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: extract common-gotchas and distribute docs/reference.md

- docs: Add docs/workflows/common-gotchas.md preserving the four sub-sections from the old root CLAUDE.md ### Common Gotchas (shared-library signatures, prometheus name conflicts, UUID db tags, model/openapi sync, RST docs sync)
- docs: Append unique Key Dependencies and Deployment content from docs/reference.md into docs/architecture/architecture-deep-dive.md
- docs: Delete docs/reference.md (multi-topic grab-bag); content now lives in api-design.md, architecture-deep-dive.md, common-gotchas.md, and security.md
EOF
)"
```

---

## Task 12: Audit `bin-common-handler/CLAUDE.md` (pilot 1 of 2)

**Step 12.1: Read the current file**

```bash
wc -l bin-common-handler/CLAUDE.md
```

**Step 12.2: Apply the audit checklist** (per design §7):

1. **Compare against `docs/reference/claude-md-template.md`** — add missing required sections (Overview, Build/Test Commands, Architecture, etc.).
2. **Strip duplicated content** — anything copy-pasted from root CLAUDE.md (verification workflow, git rules) gets removed; service files only carry service-specific content.
3. **Add CB pattern reference** — explicit link to `docs/patterns/circuit-breaker.md` in a new `## Patterns` section or under Architecture.
4. **Refresh stale references** — `docs/<file>.md` paths updated to `docs/<category>/<topic>.md`.

**Step 12.3: Commit**

```bash
git add bin-common-handler/CLAUDE.md
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: audit bin-common-handler/CLAUDE.md (pilot 1 of 2)

- bin-common-handler: Standardize CLAUDE.md against docs/reference/claude-md-template.md, strip content duplicated from root CLAUDE.md, add explicit reference to docs/patterns/circuit-breaker.md (the breaker lives in this service but was previously undocumented in its own service docs), and refresh stale docs/<file>.md path references
EOF
)"
```

---

## Task 13: Audit `bin-pipecat-manager/CLAUDE.md` (pilot 2 of 2)

**Step 13.1-13.3:** Same checklist as Task 12, applied to `bin-pipecat-manager/CLAUDE.md`. Add references to `docs/patterns/per-pod-queues.md` and `docs/patterns/per-pod-liveness-preflight.md`.

**Step 13.4: Commit**

```bash
git add bin-pipecat-manager/CLAUDE.md
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: audit bin-pipecat-manager/CLAUDE.md (pilot 2 of 2)

- bin-pipecat-manager: Standardize CLAUDE.md against docs/reference/claude-md-template.md, strip content duplicated from root CLAUDE.md, add explicit references to docs/patterns/per-pod-queues.md and docs/patterns/per-pod-liveness-preflight.md (the per-pod queue and liveness ping patterns originated in this service), and refresh stale docs/<file>.md path references
EOF
)"
```

---

## Task 14: Final sweep — check-docs.sh + grep audit + sync precheck

**Step 14.1: Run the CI guard**

```bash
./scripts/check-docs.sh
```

Expected: `OK: root CLAUDE.md = N lines, all category READMEs present.` (N ≤ 350)

**Step 14.2: Grep for any stale legacy paths**

```bash
grep -rln "docs/architecture-deep-dive\.md\|docs/git-workflow-guide\.md\|docs/verification-workflows\.md\|docs/common-workflows\.md\|docs/special-cases\.md\|docs/development-guide\.md\|docs/error-handling-patterns\.md\|docs/database-patterns-checklist\.md\|docs/test-utilities-guide\.md\|docs/rabbitmq-queues-reference\.md\|docs/claude-md-template\.md\|docs/code-quality-standards\.md\|docs/service-dependency-graph\.md\|docs/coding-conventions\.md\|docs/reference\.md" --include="*.md" --include="*.sh" --include="*.go" --include="*.yml" --include="*.yaml" 2>/dev/null | grep -v "/docs/plans/"
```

Expected: empty (any historical references inside `docs/plans/` are intentional — those are dated design/plan docs that capture state at that time).

**Step 14.3: Verify all category READMEs got populated**

```bash
for f in docs/architecture docs/conventions docs/workflows docs/patterns docs/reference; do
  echo "=== $f/README.md ==="
  head -10 "$f/README.md"
done
```

Each should have a real file table by now (filled in via the per-category populate steps). If any is still the placeholder text from Task 1.1, populate it now (1-paragraph purpose + `| filename | description |` table).

**Step 14.4: Pre-merge conflict precheck**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
git log --oneline HEAD..origin/main | head -10
```

If any new commits on origin/main touch CLAUDE.md or `docs/`, rebase and re-run the audit greps before pushing.

**Step 14.5: Commit any catch-up edits from Steps 14.3 / 14.4**

If READMEs needed populating or rebase introduced fixes, commit:

```bash
git add -A
git commit -m "$(cat <<'EOF'
NOJIRA-claude-md-categorization: final sweep populates category READMEs and resolves catch-up edits

- docs: Populate any category README that was still showing the Task 1 placeholder content
- docs: Resolve stale path references caught by the final grep sweep
EOF
)"
```

(If nothing to commit, skip this commit.)

---

## Task 15: Code review and fix loop, then push branch and open PR

**Step 15.1: Run code review on the full diff**

Per saved feedback memory: ALWAYS run code review after finishing work; fix HIGH+ severity issues before commit/PR.

```bash
git diff origin/main...HEAD --stat
git diff origin/main...HEAD | wc -l
```

If line count exceeds 5000, consider invoking the design's escape hatch (split into mechanical PR + substantive PR — see design §11 escape hatch). Otherwise, proceed.

Dispatch a code reviewer subagent against the diff range. Capture HIGH and CRITICAL findings; fix and re-verify each before pushing.

**Step 15.2: Push the branch**

```bash
git push -u origin NOJIRA-claude-md-categorization
```

**Step 15.3: Open the PR**

```bash
gh pr create --title "NOJIRA-claude-md-categorization" --body "$(cat <<'EOF'
Reorganize the monorepo's documentation into categorized docs/ subdirectories,
slim root CLAUDE.md to a focused safety-rails-plus-index file, split the 1578-line
docs/coding-conventions.md into 16 per-topic files, add four new docs/patterns/
docs covering shared infrastructure (circuit breaker, per-pod liveness preflight,
per-pod queues, WebhookMessage), audit two pilot service CLAUDE.md files, and
ship a scripts/check-docs.sh CI guard.

This is structural-only: no Go code changes, no DB changes, no behavior changes.
Filenames are preserved on move so git log --follow remains useful for blame
continuity. The remaining ~28 service CLAUDE.md audits ship as a follow-up PR
series of ~5 services per PR.

Heads-up: in-flight feature branches that touch root CLAUDE.md or
docs/coding-conventions.md will need to rebase after this lands.

- docs: Create docs/{architecture,conventions,workflows,patterns,reference}/ subdirectories with READMEs
- docs: Move 13 existing flat docs/*.md into their target subdirectories with filenames preserved
- docs: Update internal references in root CLAUDE.md, bin-billing-manager/CLAUDE.md, and .claude/scripts/check-error-log-return.sh to the new docs/<category>/<file>.md paths
- docs: Split docs/coding-conventions.md into 16 docs/conventions/<topic>.md files (one per top-level section), add docs/conventions/README.md as the new source-of-truth index, delete the original
- docs: Fold docs/error-handling-patterns.md, docs/database-patterns-checklist.md, and docs/test-utilities-guide.md into their canonical docs/conventions/ counterparts and delete the originals
- docs: Add docs/patterns/circuit-breaker.md documenting the existing per-target CB in bin-common-handler/pkg/circuitbreakerhandler
- docs: Add docs/patterns/per-pod-liveness-preflight.md generalizing the PR #832 pipecat ping pattern (any-response-alive, best-effort host_id echo, preflight-before-DB-write)
- docs: Add docs/patterns/per-pod-queues.md describing the <service>.request.<host_id> volatile-queue convention
- docs: Add docs/patterns/webhook-message.md extracting the WebhookMessage external-API-response pattern from root CLAUDE.md with bin-api-manager examples
- CLAUDE.md: Slim root CLAUDE.md from 746 to ~200-250 lines (critical safety rails plus index plus where-to-document decision tree); enforced going forward by scripts/check-docs.sh
- docs: Extract ### Common Gotchas (~150 lines) from root CLAUDE.md into docs/workflows/common-gotchas.md
- docs: Distribute the multi-topic docs/reference.md into docs/conventions/api-design.md and docs/architecture/architecture-deep-dive.md and delete the original
- bin-common-handler: Audit CLAUDE.md against docs/reference/claude-md-template.md, strip duplicated root content, add explicit reference to docs/patterns/circuit-breaker.md
- bin-pipecat-manager: Audit CLAUDE.md against docs/reference/claude-md-template.md, strip duplicated root content, add explicit references to docs/patterns/per-pod-queues.md and docs/patterns/per-pod-liveness-preflight.md
- scripts: Add scripts/check-docs.sh enforcing root CLAUDE.md line cap (350) and per-category README presence
EOF
)"
```

**Step 15.4: Wait for explicit user authorization before merging.** Per saved feedback memory and CLAUDE.md, NEVER merge without explicit user request, and when authorized use squash merge: `gh pr merge <pr-number> --squash --delete-branch`.

---

## Post-merge cleanup (after user authorizes merge)

```bash
cd /home/pchero/gitvoipbin/monorepo
git pull origin main
git worktree remove /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-claude-md-categorization
```

## Follow-up work (separate PR series, not in scope here)

Audit the remaining ~28 `bin-*-manager/CLAUDE.md` files in batches of 5 services per PR using the same checklist applied in Tasks 12-13. Pilot output (`bin-common-handler/CLAUDE.md`, `bin-pipecat-manager/CLAUDE.md`) serves as the reference template.
