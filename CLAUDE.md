# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Scope

**This CLAUDE.md applies to ALL services in this monorepo.**

Whether you're working in `bin-api-manager`, `bin-flow-manager`, `bin-call-manager`, or any other service directory, the guidelines, commands, and architectural patterns described here apply uniformly across the entire monorepo.

**Service-Specific Documentation:** Some individual services have their own `CLAUDE.md` files. **When conflicts arise, the service-specific CLAUDE.md takes precedence.** Use the root CLAUDE.md for general monorepo patterns and service-specific CLAUDE.md for implementation details.

## Overview

This is the VoIPbin monorepo - a unified backend codebase for a cloud-native CPaaS platform. It contains 37 services (34 `bin-*` + 3 `voip-*`-proxy) providing VoIP, messaging, conferencing, AI integration, and communication workflow orchestration.

**Key Characteristics:**
- **Monorepo architecture** - All backend services in one repository with local module replacements
- **Microservices communication** - Services communicate via RabbitMQ RPC, not HTTP
- **Shared infrastructure** - Common MySQL database, Redis cache, RabbitMQ message broker
- **Event-driven architecture** - Pub/sub events via RabbitMQ and ZeroMQ
- **Kubernetes deployment** - Services designed for GCP GKE with Prometheus monitoring

## CRITICAL: Verification before commit

**⚠️ MANDATORY: ALWAYS run the verification workflow after making ANY code changes and BEFORE committing.**

This applies to ALL changes: code modifications, refactoring, bug fixes, new features, or any other changes. No exceptions.

**For normal code changes (bug fixes, features, refactoring), run this workflow BEFORE committing:**

```bash
# Navigate to the service directory where changes were made
cd bin-<service-name>

# Run the verification workflow (NO dependency updates)
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

The full verification workflow consists of 5 steps that MUST all be run:

| Step | Command | Purpose |
|------|---------|---------|
| 1 | `go mod tidy` | Sync go.mod/go.sum with imports, remove unused deps, add missing ones |
| 2 | `go mod vendor` | Copy dependencies to vendor/ for local builds (vendor is NOT committed to git; Dockerfiles regenerate it during build) |
| 3 | `go generate ./...` | Run code generators (mocks, OpenAPI types, etc.) |
| 4 | `go test ./...` | Run all unit tests |
| 5 | `golangci-lint run -v --timeout 5m` | Run static analysis and linting |

**Do NOT skip any steps.** Each step can catch different issues. "The change is trivial" is NOT a valid reason to skip — even adding a single stdlib call (e.g., `os.Getenv`) can cause `go.sum` to become stale, breaking Docker builds with `missing go.sum entry` errors. `go build` passing locally does NOT mean the service will deploy successfully; only `go mod tidy` updates `go.sum` with the transitive dependency checksums that Dockerfiles require. Commit the resulting `go.mod`/`go.sum` changes along with the code changes.

**IMPORTANT: Vendor directories are NOT committed to git.** The `.gitignore` excludes `vendor/`. Do NOT use `git add -f` for vendor files. Each service's Dockerfile runs `go mod vendor` during Docker build to regenerate dependencies. The local `go mod vendor` step is only for local development and testing.

→ Detail: [docs/workflows/verification-workflows.md](docs/workflows/verification-workflows.md), [docs/workflows/special-cases.md](docs/workflows/special-cases.md)

## CRITICAL: Worktrees

**🚨 MANDATORY RULE - NO EXCEPTIONS 🚨**

**NEVER edit, create, or modify ANY files directly in the main repository (`~/gitvoipbin/monorepo`).**

This includes: code changes, config file changes (including `.circleci/`), documentation changes (including `docs/`), design documents, ANY file modifications.

**BEFORE making any changes, you MUST:**

1. **Check current directory:**
   ```bash
   pwd
   # If output is ~/gitvoipbin/monorepo → STOP, create worktree first
   # If output is ~/gitvoipbin/monorepo/.worktrees/<branch-name> → OK to proceed
   ```

2. **Create a worktree (if not already in one):**
   ```bash
   cd ~/gitvoipbin/monorepo
   git worktree add ~/gitvoipbin/monorepo/.worktrees/NOJIRA-feature-name -b NOJIRA-feature-name
   cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-feature-name
   ```

3. **Work in the worktree:** All file edits, document creation, and commits happen here. Main repository stays clean on `main` branch.

4. **When done, remove the worktree:**
   ```bash
   cd ~/gitvoipbin/monorepo
   git worktree remove ~/gitvoipbin/monorepo/.worktrees/NOJIRA-feature-name
   ```

**If you accidentally edited files in main repository:** Do NOT commit. Stash or discard changes (`git stash` or `git checkout .`). Create worktree and apply changes there.

→ Detail: [docs/workflows/git-workflow-guide.md](docs/workflows/git-workflow-guide.md)

## CRITICAL: Branch & commit format

**CRITICAL: Commit title MUST match the branch name exactly.**

**Title (first line):**
```
VOIP-[ticket-number]-brief-description-of-change
```
or (when no JIRA ticket)
```
NOJIRA-brief-description-of-change
```

**Body (subsequent lines):** List each affected project with specific changes:
```
- bin-common-handler: Fixed type handling in database mapper
- bin-flow-manager: Updated flow execution to use new types
- bin-call-manager: Refactored call handler to support new interface
```

**Rules:**
1. Always list affected projects with `bin-<service-name>:` prefixes
2. Be specific about what changed in each project
3. Keep title concise; use present tense
4. Use dashes (`-`) for bullet points
5. Do NOT include "Co-Authored-By" lines or AI attribution in commit messages or PR descriptions

**Branch Management:**

**CRITICAL: Before making ANY changes or commits, ALWAYS check the current branch first.** If on `main`: STOP, ask user to create a feature branch (`NOJIRA-brief-description` or `VOIP-1234-brief-description`), wait for confirmation.

**NEVER commit directly to `main` without explicit user permission.**
**NEVER merge any branch to `main` without explicit user permission.**

**CRITICAL: ALL PR merges MUST use squash merge — no exceptions.**

When the user authorizes a merge:
- ✅ **ALWAYS** use squash merge: `gh pr merge <pr-number> --squash --delete-branch`
- ❌ **NEVER** use regular merge commits or rebase merge
- The squashed commit title MUST match the PR title (which matches the branch name)
- The squashed commit body MUST match the PR body
- Do NOT include AI attribution in the squashed commit

**Why:** Keeps `main` history linear, one logical change = one commit, easy to `git revert` if a PR causes issues. Intermediate WIP/review-fix commits stay on the feature branch, not `main`.

**CRITICAL: Before creating a PR or merging, ALWAYS pull the latest `main` and check for conflicts.**

This is mandatory — no exceptions. Run these steps **from the worktree directory** (where your feature branch lives):
1. **Fetch latest main:** `git fetch origin main`
2. **Check for conflicts:** `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`
3. **Review what changed on main:** `git log --oneline HEAD..origin/main`
4. **If conflicts exist:** Rebase or merge main into your branch, resolve conflicts, and re-run the full verification workflow before proceeding.
5. **If no conflicts:** Proceed with PR creation or merge.

**CRITICAL: After a PR is merged on GitHub, ALWAYS sync the local main branch:**

```bash
cd ~/gitvoipbin/monorepo && git pull origin main
```

This keeps the main repository directory in sync with remote so new worktrees start from the latest code.

→ Detail: [docs/workflows/git-workflow-guide.md](docs/workflows/git-workflow-guide.md)

## CRITICAL: Database via Alembic only

**CRITICAL: Database schema changes ONLY through Alembic migrations in `bin-dbscheme-manager`.**

**What AI CAN do:**
- ✅ Create migration files (`alembic -c alembic.ini revision -m "..."`)
- ✅ Edit migration files to add SQL in upgrade()/downgrade() functions
- ✅ Commit migration files to git

**What AI MUST NEVER do:**
- 🚫 Run `alembic upgrade` (applies migrations to database)
- 🚫 Run `alembic downgrade` (rolls back database changes)
- 🚫 Execute any SQL that modifies database schema
- 🚫 Manually create migration files with hand-picked revision IDs — always use `alembic revision` to generate them (see `bin-dbscheme-manager/CLAUDE.md` for details)

**Why:** Database changes are irreversible and require human authorization, testing, and VPN access.

→ Detail: `bin-dbscheme-manager/CLAUDE.md`, [docs/workflows/common-workflows.md#database-migrations-with-alembic](docs/workflows/common-workflows.md#database-migrations-with-alembic)

## CRITICAL: RST docs sync

**CRITICAL: The RST docs in `bin-api-manager/docsdev/source/` are the primary user-facing documentation and the single source of truth for how the platform works. When adding or changing any user-visible feature, you MUST update the relevant RST docs.**

The RST documentation at `bin-api-manager/docsdev/source/` is what customers, developers, and integrators rely on to understand VoIPbin's APIs, billing, features, and behavior. Stale docs are worse than no docs — they actively mislead.

**This applies when you:**
- Add a new billable service type (update rate tables, diagrams, examples in `billing_account_overview.rst`)
- Add or modify API endpoints (update the relevant resource's `*_overview.rst`, `*_tutorial.rst`, `*_struct.rst`)
- Change pricing, rates, or billing behavior
- Add new event types that affect user-visible webhooks
- Add new resource types, statuses, or fields
- Change any behavior documented in the existing RST files

**When updating RST docs:**
1. **Edit the RST source** in `bin-api-manager/docsdev/source/`
2. **Clean rebuild the HTML**: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`
3. **Force-add the build output**: `git add -f bin-api-manager/docsdev/build/` (root `.gitignore` excludes `build/`)
4. **Commit both RST source and built HTML together**

**IMPORTANT:** Always do a clean rebuild (`rm -rf build` first). Incremental Sphinx builds may miss cross-page references. The built HTML is tracked in git and must stay in sync with the RST sources.

**RST struct docs must match `WebhookMessage`, not internal model structs.** The `WebhookMessage` struct (defined in `models/<entity>/webhook.go`) determines exactly which fields are exposed to external users via the API. RST struct documentation (`*_struct_*.rst`) must only include fields present in `WebhookMessage`. Do not document internal-only fields (e.g., `PodID`, `Username`, `PermissionIDs`) that are stripped by `ConvertWebhookMessage()`. When verifying RST accuracy, always compare against `WebhookMessage` fields, not the internal model struct.

→ Detail: [docs/workflows/special-cases.md](docs/workflows/special-cases.md), [docs/workflows/common-gotchas.md](docs/workflows/common-gotchas.md)

## CRITICAL: Service docs sync

**CRITICAL: Each service's `docs/` suite must stay in sync with its source code. When changing source that drives doc content, update the corresponding doc file in the same commit.**

| Source change | Doc file to update |
|---|---|
| `pkg/listenhandler/main.go` | `docs/architecture.md` — routing table |
| `cmd/*/main.go` or `pkg/subscribehandler/main.go` (subscribeTargets) | `docs/architecture.md` — events section |
| `internal/config/*.go` or `cmd/*/init.go` | `docs/operations.md` — config table |
| `models/.../*.go` (any depth) | `docs/domain.md` — domain entities |
| `go.mod` (replace directives) | `docs/dependencies.md` — local deps |

**To re-extract routing / events / config / deps from source:**
```bash
bash docs/reference/extractor.sh <service-dir>
```

**The pre-commit hook (`scripts/check-service-docs.sh`) will warn (not block) when these source files change without a matching docs update.** Stage the relevant `docs/*.md` alongside the source change to suppress the warning.

→ Script: [scripts/check-service-docs.sh](scripts/check-service-docs.sh)

## CRITICAL: bin-common-handler admission rule

**CRITICAL: A package may only live in `bin-common-handler` if it is used by 3 or more services.**

- Single-consumer or dual-consumer packages belong in the consuming service(s).
- If a package's usage later grows to 3+ services, it can be promoted to `bin-common-handler`.
- Internal plumbing packages (e.g., `rabbitmqhandler` wrapped by `sockhandler`) are exempt since they serve the shared library itself.

**Why:** `bin-common-handler` is a globally shared library — every change triggers verification across all 37 services. Keeping it lean reduces blast radius and maintenance burden.

→ Detail: [docs/conventions/package-structure.md](docs/conventions/package-structure.md)

## Where to find things (index)

- **Architecture** → [docs/architecture/](docs/architecture/) — service boundaries, inter-service communication, deployment topology, dependency graph
- **Coding conventions** → [docs/conventions/](docs/conventions/) — Go coding rules (package structure, naming, errors, logging, database, handlers, testing, ...). Run `ls docs/conventions/` for the current set.
- **Workflows** → [docs/workflows/](docs/workflows/) — git, verification, multi-service feature workflows, common gotchas
- **Shared patterns** → [docs/patterns/](docs/patterns/) — applied infrastructure patterns with reference implementations (e.g. circuit breaker, WebhookMessage)
- **Reference** → [docs/reference/](docs/reference/) — queue catalog, service CLAUDE.md template, code-quality standards
- **Plans** → [docs/plans/](docs/plans/) — dated design documents and implementation plans

## Where to Document New Information

The `docs/` directory is the **live, authoritative shared documentation** for the monorepo. Subdirectories evolve over time — new categories may be added at any time.

**Before adding new documentation, ALWAYS run `ls docs/` and `ls docs/<subdir>/` to inspect the current state of categories.** Do not rely on a memorized list — the categories listed below are a snapshot, not an exhaustive contract. If a topic clearly fits an existing category, place it there. If no category fits, propose a new `docs/<category>/` subdirectory with a brief `README.md`, and update this decision tree in the same change.

Use this decision tree when adding new documentation, in order:

**1. Is it a coding/style/language convention or pattern that applies broadly to Go code in this monorepo?**
- **YES** → Add to `docs/conventions/<topic>.md`
  - Examples: error-handling rules (e.g. variable naming in if-init blocks), import grouping, naming conventions, package structure, testing patterns, RPC conventions, database patterns
  - **Run `ls docs/conventions/` to see current files** — the set evolves; do not rely on a memorized list

**2. Is it an architectural detail (service boundaries, inter-service communication, deployment topology, dependency graph)?**
- **YES** → Add to `docs/architecture/<topic>.md`

**3. Is it a workflow spanning services (git, verification, deployment, multi-service feature work)?**
- **YES** → Add to `docs/workflows/<topic>.md`
  - Promote to root CLAUDE.md if it is a CRITICAL safety rail (e.g. mandatory verification step, branch protection rule)
  - Example: "Adding a New API Endpoint" (touches openapi-manager + api-manager + target service)

**4. Is it a shared infrastructure pattern with a reference implementation (e.g. circuit breaker, WebhookMessage)?**
- **YES** → Add to `docs/patterns/<topic>.md`
  - For one-off cross-cutting gotchas without a reference implementation, use `docs/workflows/common-gotchas.md` instead
  - Example: "Parsing Filters from Request Body"

**5. Is it reference material (queue catalog, service CLAUDE.md template, code-quality standards)?**
- **YES** → Add to `docs/reference/<topic>.md`

**6. Is it a dated design document or implementation plan for a non-trivial change?**
- **YES** → Add to `docs/plans/YYYY-MM-DD-<topic>-design.md` (or `-plan.md` for execution plans)

**7. Is it specific to one service?**
- **YES** → Add to that service's `<service>/CLAUDE.md`
  - Examples: service-specific API endpoints, handler patterns unique to this service, domain-specific implementation details

**8. Otherwise — does it apply universally to ALL services AND is it a CRITICAL safety rail?**
- **YES** → Add to root CLAUDE.md
  - Examples: mandatory verification step required for all services, repo-wide git workflow rule, change to RabbitMQ communication pattern

**Maintenance rule:** Whenever a new `docs/<subdir>/` is introduced or removed, update this decision tree **and** the "Where to find things (index)" section above in the same change. The category descriptions in both sections are intentionally brief — they describe *kinds* of content, not exhaustive file lists, so adding individual files within an existing subdirectory does **not** require updating either section. Stale routing here is the most common failure mode — it sends new content to private memory or service CLAUDE.md files instead of the right shared category.

## Reference

Quick links:
- Admin Console: https://admin.voipbin.net/
- Agent Interface: https://talk.voipbin.net/
- API Documentation: https://api.voipbin.net/docs/
- Project Site: http://voipbin.net/
- Architecture diagram: `architecture_overview_all.png` in repo root
