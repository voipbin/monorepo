# CLAUDE.md Reorganization Design

## Problem

The root CLAUDE.md file has grown to 1736 lines (64KB). This size creates cognitive load—the file is hard to scan, navigate, and maintain. Users seeking specific guidance must scroll through extensive content, and contributors struggle to locate where new information belongs.

## Goal

Split CLAUDE.md into focused documentation files while preserving all content. The main file will retain critical rules that Claude must always see. Detailed examples, workflows, and reference material will move to separate files linked from the main document.

## Proposed Structure

### Main File: CLAUDE.md (~200-300 lines)

The main file keeps:
- Critical rules (git workflow, verification commands, special case triggers)
- Brief summaries of each topic
- Clear links to detailed documentation
- Decision trees and scope definitions

The main file removes:
- Extended examples and code snippets
- Detailed troubleshooting sections
- Step-by-step tutorials
- Architecture descriptions
- Long explanations

### New Documentation Files in docs/

```
docs/
├── verification-workflows.md      # Verification steps, troubleshooting, test cache
├── special-cases.md              # bin-common-handler updates, OpenAPI sync
├── git-workflow-guide.md         # Commit examples, branch strategies, merge rules
├── development-guide.md          # Build commands, testing, code generation
├── code-quality-standards.md     # Logging patterns, naming conventions
├── architecture-deep-dive.md     # Services, communication, configuration
├── common-workflows.md           # API endpoints, migrations, filter parsing
└── reference.md                  # Dependencies, deployment, security
```

Each detailed file will:
- Start with "Quick Reference" link back to main CLAUDE.md
- Organize content with clear headings
- Include all code examples and troubleshooting steps
- Use anchors for cross-linking (e.g., `#openapi-sync`)

## Content Distribution

### verification-workflows.md

Gets from CLAUDE.md:
- "What this does" explanations for each command
- Dependency update workflow details
- Test cache warnings
- All troubleshooting scenarios

### special-cases.md

Gets from CLAUDE.md:
- Complete bin-common-handler update workflow
- Projects affected list
- Verification checklist
- Common mistakes section
- UUID filter bug examples
- OpenAPI schema sync process
- Validation workflow

### git-workflow-guide.md

Gets from CLAUDE.md:
- Extended commit message examples
- Narrative commit format samples
- Branch management scenarios
- Multiple good/bad examples
- Main branch protection rules
- Merge permission workflows

### development-guide.md

Gets from CLAUDE.md:
- Prerequisites (ZMQ libraries)
- Building services (complete examples)
- Testing commands (coverage, specific packages)
- SQLite test database pattern
- Code generation commands
- Linting commands

### code-quality-standards.md

Gets from CLAUDE.md:
- Complete logging standards section
- Function-scoped log examples
- Go naming conventions
- UUID fields and DB tags gotcha
- Import patterns
- All code examples

### architecture-deep-dive.md

Gets from CLAUDE.md:
- Service categories (complete list with descriptions)
- Inter-service communication patterns
- RabbitMQ RPC examples
- Configuration management
- Package organization pattern
- Database & caching
- Testing patterns
- CircleCI integration

### common-workflows.md

Gets from CLAUDE.md:
- Adding a new API endpoint (6 steps)
- Creating a new flow action (5 steps)
- Adding a new manager service (6 steps)
- Modifying shared models (5 steps)
- Parsing filters from request body (complete section)
- Database migrations with Alembic (entire section)

### reference.md

Gets from CLAUDE.md:
- API design principles
- Key dependencies (categorized lists)
- Deployment information
- Important notes (monorepo practices, communication)
- Security considerations
- Resources (URLs)

## Main CLAUDE.md Example Structure

```markdown
# CLAUDE.md

## Scope
[Unchanged - applies to all services]

## Overview
[Unchanged - 2-3 sentence description]

## CRITICAL: Before Committing Changes

⚠️ MANDATORY: Run verification workflow before committing.

### Regular Code Changes
```bash
cd bin-<service-name>
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v
```

**For detailed explanations and troubleshooting, see [docs/verification-workflows.md](docs/verification-workflows.md)**

### Special Cases

**Changes to bin-common-handler:** Update ALL 30+ services.
**Changes to public-facing models:** Update OpenAPI schemas.

**For complete workflows and troubleshooting, see [docs/special-cases.md](docs/special-cases.md)**

## Git Workflow

### Commit Message Format

Commit title MUST match branch name exactly.

**Format:**
```
NOJIRA-brief-description

- bin-service: What changed
```

**For extended examples, see [docs/git-workflow-guide.md](docs/git-workflow-guide.md)**

## Build & Development

**See [docs/development-guide.md](docs/development-guide.md)**

## Architecture

**See [docs/architecture-deep-dive.md](docs/architecture-deep-dive.md)**

## Common Workflows

**See [docs/common-workflows.md](docs/common-workflows.md)**

## Code Quality

Follow these standards:
- Generate mocks after interface changes
- Write table-driven tests
- Use function-scoped logging
- Follow Go naming (List not Gets)

**For detailed standards and examples, see [docs/code-quality-standards.md](docs/code-quality-standards.md)**

## Where to Document New Information

[Keeps existing decision tree - unchanged]

## Reference

**See [docs/reference.md](docs/reference.md)**
```

## Detailed File Example: verification-workflows.md

```markdown
# Verification Workflows

> **Quick Reference:** For commands, see [CLAUDE.md](../CLAUDE.md#critical-before-committing-changes)

## Overview

This document explains verification workflows that run before committing code. Every code change requires verification to maintain monorepo consistency.

## Regular Code Changes Workflow

### What Each Command Does

1. `go mod tidy` - Cleans up go.mod and go.sum files
2. `go mod vendor` - Vendors dependencies for reproducible builds
3. `go generate ./...` - Regenerates mocks and generated code
4. `go test ./...` - Runs all tests to ensure nothing broke
5. `golangci-lint run -v --timeout 5m` - Lints code for quality

### When to Use

Use this workflow for:
- Bug fixes
- New features
- Refactoring
- Any code modification

### When NOT to Use

Skip this if updating dependencies. Use Dependency Update Workflow instead.

## Dependency Update Workflow

[Full section from current CLAUDE.md...]

## Troubleshooting

### Test Cache Issues

⚠️ CRITICAL: Test cache hides failures.

[Current troubleshooting content...]
```

## Implementation Plan

### Phase 1: Create Documentation Files
1. Create 7 new files in docs/ directory
2. Copy relevant sections from CLAUDE.md to each file
3. Add "Quick Reference" links back to main CLAUDE.md
4. Preserve all code examples and detailed content

### Phase 2: Update Main CLAUDE.md
1. Replace detailed sections with brief summaries
2. Add clear links to detailed documentation
3. Keep all critical rules and commands
4. Maintain decision trees and scope sections

### Phase 3: Validate Cross-References
1. Test all links
2. Check for orphaned content
3. Verify anchors work (e.g., `#openapi-sync`)
4. Confirm Claude can find referenced sections

### Phase 4: Review & Test
1. Review main CLAUDE.md for completeness
2. Spot-check detailed files for accuracy
3. Verify content preservation
4. Check line counts (main should be ~200-300 lines)

## Risks & Mitigation

**Risk:** Claude misses detailed docs if links are unclear.
**Mitigation:** Use explicit "See [link]" format, never vague references.

**Risk:** Content duplication or loss during migration.
**Mitigation:** Systematic copy-paste with verification checklist.

**Risk:** Breaking external references to CLAUDE.md sections.
**Mitigation:** Check for external references before moving content.

## Success Criteria

- Main CLAUDE.md shrinks to 200-300 lines (from 1736)
- All 7 detailed files created and populated
- No content lost during migration
- All links functional
- Critical rules remain in main file

## Benefits

**Reduced cognitive load:** Main file becomes scannable.
**Easier maintenance:** Update specific topics in focused files.
**Better navigation:** Jump directly to relevant detailed documentation.
**Preserved content:** All examples and workflows remain accessible.
**Clear structure:** Explicit organization aids both humans and Claude.
