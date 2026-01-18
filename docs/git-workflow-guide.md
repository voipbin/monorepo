# Git Workflow Guide

> **Quick Reference:** For rules summary, see [CLAUDE.md](../CLAUDE.md#git-workflow)

## Commit Message Format

**CRITICAL: This is a monorepo containing multiple projects. Commit messages MUST specify which projects were affected.**

**CRITICAL: Commit title MUST match the branch name exactly.**

### Commit Message Structure

**Title (first line):**

The commit title MUST match the branch name exactly (same format, all lowercase with hyphens):

```
VOIP-[ticket-number]-brief-description-of-change
```
or (when no JIRA ticket)
```
NOJIRA-brief-description-of-change
```

**Important:** Title uses the same format as branch name (all lowercase with hyphens)

**Body (subsequent lines):**
List each affected project with specific changes:
```
- bin-common-handler: Fixed type handling in database mapper
- bin-flow-manager: Updated flow execution to use new types
- bin-call-manager: Refactored call handler to support new interface
- bin-conference-manager: Updated conference creation logic
```

### Complete Examples

**Complete Example (Narrative Style):**
```
VOIP-1190-refactor-database-handlers-to-use-commondatabasehandler

Successfully refactored 22 microservices to adopt standardized commondatabasehandler
pattern from bin-common-handler, improving type safety and code consistency across
the entire monorepo.

- bin-common-handler: Provides PrepareFields(), ScanRow(), ApplyFields(), and
  GetDBFields() utilities for standardized database operations
- bin-ai-manager: Adds db tags to ai, aicall, message, and summary models with
  typed Field constants for type-safe database operations
- bin-ai-manager: Migrates all dbhandler operations to use PrepareFields() for
  INSERT/UPDATE and ScanRow() for result scanning
- bin-call-manager: Adds field.go files for call, confbridge, groupcall, and
  recording models with typed Field constants
- bin-call-manager: Refactors all database queries to use Squirrel SetMap()
  with PrepareFields() instead of Columns().Values()
- bin-conference-manager: Adds ConvertStringMapToFieldMap() helper functions
  for filter conversion from external APIs
... (continue for all affected services)

Test results: All 28 services passing (477 files modified)
```

**Commit Format (matches branch name):**
```
VOIP-[ticket-number]-brief-description-of-change

[Narrative summary paragraph explaining what was accomplished, the impact,
and high-level context. 2-3 sentences recommended for significant changes.]

- bin-service-1: What changed
- bin-service-2: What changed
- bin-service-3: What changed
```

**Example:**
```
NOJIRA-add-claude-md-reorganization-design

Created comprehensive design document to reorganize CLAUDE.md documentation structure
across the monorepo. Establishes clear boundaries between monorepo-wide rules in root
CLAUDE.md and service-specific implementation details in service CLAUDE.md files,
resolving confusion about where to document new information.

- docs: Define clear boundaries between root and service-specific CLAUDE.md files
- docs: Establish decision framework for where to document new information
- docs: Plan content migration from root to service-specific files (api-manager, flow-manager, billing-manager)
- docs: Add notes strategy for shared patterns used across multiple services
- docs: Document implementation plan with 5 phases and risk mitigation
```

### Rules

1. **Always list affected projects** - Even if it's just one project
2. **Be specific** - Describe what changed in each project, not just "updated"
3. **Keep title concise** - Detailed changes go in the body
4. **Use present tense** - "Add feature" not "Added feature"
5. **Use dashes (`-`) for bullet points** - Never use asterisks (`*`)
6. **Add narrative summary** - For significant changes (new features, refactoring, multi-service updates), include 2-3 sentence summary paragraph before the bullet list explaining what was accomplished and the impact
7. **Multiple bullets per service** - Complex changes should have multiple detailed bullets
8. **Include test results** - For large changes, add summary line at the end (e.g., "Test results: All 28 services passing")

### Good and Bad Examples

**Good examples:**
```
VOIP-1234-add-jwt-authentication-support

- bin-api-manager: Implement JWT middleware and token validation
- bin-customer-manager: Add token generation endpoints
- bin-common-handler: Add JWT utility functions
```

```
NOJIRA-fix-database-connection-leak

- bin-call-manager: Close database connections in defer statements
- bin-conference-manager: Add connection pool timeout handling
```

**Bad examples:**
```
Fixed bug  ❌ (No ticket number, no affected projects)
```

```
VOIP-1234: Updated everything  ❌ (Old format with colon, not specific, no project list)
```

```
VOIP-1234-add-feature
- Updated files  ❌ (Not specific about which projects)
```

## Branch Management {#branch-management}

**CRITICAL: Before making ANY changes or commits, ALWAYS check the current branch first.**

### Before Making Changes

**If the current branch is `main`:**
1. **STOP - DO NOT make commits on main**
2. Ask the user to create a feature branch first
3. Suggest a branch name following this convention: `NOJIRA-brief-description` or `VOIP-1234-brief-description`
4. Wait for user confirmation before proceeding with any code changes

**Example prompt when starting work:**
```
You're currently on the main branch. It's recommended to create a feature branch before making changes.

Suggested branch name: NOJIRA-fix-conference-customer-id

Would you like to:
1. Create and switch to this branch
2. Use a different branch name
3. Work on main anyway (not recommended)
```

### Branch Naming Convention

- Format: `NOJIRA-brief-description` or `VOIP-[ticket]-brief-description`
- Use lowercase with hyphens separating words
- Commit title MUST match branch name exactly
- Examples:
  - `NOJIRA-fix-conference-customer-id`
  - `NOJIRA-add-user-authentication`
  - `VOIP-1234-refactor-flow-manager`
- Keep it concise but descriptive

### Correct Workflow

```bash
# Step 1: ALWAYS check current branch BEFORE making any changes
git branch --show-current

# Step 2: If on main, create feature branch BEFORE any edits
git checkout -b NOJIRA-descriptive-change-summary

# Step 3: Make your code changes

# Step 4: Run the verification workflow BEFORE committing (from section above)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Step 5: Commit changes (title matches branch name)
git add .
git commit -m "NOJIRA-descriptive-change-summary

- project-name: What changed
- project-name: What changed"

# Step 6: Push to remote
git push -u origin NOJIRA-descriptive-change-summary
```

**Only proceed with working on main if the user explicitly confirms.**

## Never Commit Directly to Main {#main-branch-protection}

**ALWAYS work on feature branches. NEVER commit directly to `main` without explicit user permission.**

### The Rule

**Before making ANY changes or commits:**
1. Check current branch: `git branch --show-current`
2. If on `main`, create a feature branch FIRST
3. Only commit to main if user explicitly approves

**This applies to ALL changes:**
- Code changes
- Documentation updates (including CLAUDE.md)
- Configuration files
- Any other modifications

### Prohibited Workflow

```bash
# ❌ WRONG - Committing directly to main
git branch --show-current  # shows: main
# ... make changes to files ...
git add .
git commit -m "some change"  # NEVER DO THIS ON MAIN
```

### Correct Workflow

```bash
# ✅ CORRECT - Create branch first
git branch --show-current  # shows: main
git checkout -b NOJIRA-descriptive-change-name  # Create feature branch FIRST
# ... make changes to files ...
git add .
git commit -m "NOJIRA-descriptive-change-name

- project-name: What changed"  # Safe - on feature branch, title matches branch
```

### Exceptions

Only commit directly to main when user explicitly says:
- "commit this to main"
- "yes, commit directly to main" (in response to your question)

## Merging to Main Branch {#merging-rules}

**NEVER merge any branch to `main` without explicit user permission.**

### Prohibited Actions

**Prohibited actions without user approval:**
- `git merge <branch>` while on main
- `git merge <branch> --no-ff` while on main
- Any operation that merges commits into main

### Required Workflow

1. ✅ Push feature branches to remote: `git push -u origin <branch-name>`
2. ✅ Create pull requests on GitHub for review
3. ❌ **DO NOT** merge to main directly - always ask user first

### If User Says "Push It"

**If user says "push it" or similar:**
- **ONLY push the current branch** to remote
- **DO NOT assume** this means merge to main
- **ASK explicitly** if merge to main is intended

**Example - What to do:**
```
User: "push it"
Claude: "I'll push the current branch to remote. Should I also merge it to main, or just push the feature branch?"
```

### Only Merge When

**Only merge to main when the user explicitly says:**
- "merge to main"
- "merge it to main and push"
- "yes, merge to main" (in response to your question)
