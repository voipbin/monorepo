# CLAUDE.md Split Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Split the 1736-line CLAUDE.md into a streamlined main file plus 7 focused documentation files.

**Architecture:** Extract detailed content from CLAUDE.md into topic-specific files in docs/. Main CLAUDE.md becomes a 200-300 line reference with critical rules and links to detailed docs.

**Tech Stack:** Markdown, manual content extraction and reorganization.

---

## Task 1: Create verification-workflows.md

**Files:**
- Create: `docs/verification-workflows.md`
- Reference: `CLAUDE.md` lines 28-275

**Step 1: Create file with header and structure**

Create `docs/verification-workflows.md`:

```markdown
# Verification Workflows

> **Quick Reference:** For commands, see [CLAUDE.md](../CLAUDE.md#critical-before-committing-changes)

## Overview

This document provides detailed explanations, troubleshooting steps, and special cases for verification workflows that must run before committing any code.

## Regular Code Changes Workflow

### What This Does

### When to Use

### When NOT to Use

## Dependency Update Workflow

## Troubleshooting

### Test Cache Issues

### Build Failures
```

**Step 2: Extract content from CLAUDE.md lines 34-82**

Copy the following sections:
- Regular Code Changes Workflow (lines 34-58)
- Dependency Update Workflow (lines 59-82)

**Step 3: Verify content completeness**

Check that verification-workflows.md includes:
- All bash commands
- All "What this does" explanations
- When to use/not use guidance
- Test cache warnings

**Step 4: Commit**

```bash
git add docs/verification-workflows.md
git commit -m "docs: create verification-workflows.md with detailed verification steps"
```

---

## Task 2: Create special-cases.md

**Files:**
- Create: `docs/special-cases.md`
- Reference: `CLAUDE.md` lines 84-337

**Step 1: Create file with header**

Create `docs/special-cases.md`:

```markdown
# Special Cases

> **Quick Reference:** For triggers, see [CLAUDE.md](../CLAUDE.md#special-cases)

## Changes to bin-common-handler

### What Counts as a Change

### Complete Update Workflow

### Projects Affected

### Why This Is Critical

### Verification Checklist

### Common Mistakes to Avoid

### Troubleshooting

## Changes to Public-Facing Models and OpenAPI Schemas {#openapi-sync}

### What are public-facing models?

### The Rule

### Validation Process

### Why this is critical

### Common scenarios
```

**Step 2: Extract bin-common-handler section (lines 84-274)**

Copy complete bin-common-handler workflow including:
- What counts as a change
- 4-step update workflow
- Projects affected list
- Verification checklist
- Common mistakes
- Troubleshooting scenarios

**Step 3: Extract OpenAPI sync section (lines 276-337)**

Copy public-facing models section with validation process.

**Step 4: Commit**

```bash
git add docs/special-cases.md
git commit -m "docs: create special-cases.md with bin-common-handler and OpenAPI workflows"
```

---

## Task 3: Create git-workflow-guide.md

**Files:**
- Create: `docs/git-workflow-guide.md`
- Reference: `CLAUDE.md` lines 339-592

**Step 1: Create file structure**

Create `docs/git-workflow-guide.md`:

```markdown
# Git Workflow Guide

> **Quick Reference:** For rules summary, see [CLAUDE.md](../CLAUDE.md#git-workflow)

## Commit Message Format

### Structure

### Title Format

### Body Format

### Complete Examples

### Rules

### Good and Bad Examples

## Branch Management {#branch-management}

### Before Making Changes

### Branch Naming Convention

### Correct Workflow

## Never Commit Directly to Main {#main-branch-protection}

### The Rule

### Prohibited Workflow

### Correct Workflow

### Exceptions

## Merging to Main Branch {#merging-rules}

### The Rule

### Prohibited Actions

### Required Workflow

### If User Says "Push It"

### Only Merge When
```

**Step 2: Extract commit message format (lines 339-462)**

Copy all commit message examples, rules, good/bad examples.

**Step 3: Extract branch management (lines 464-592)**

Copy branch management, main branch protection, merge rules.

**Step 4: Commit**

```bash
git add docs/git-workflow-guide.md
git commit -m "docs: create git-workflow-guide.md with commit and branch guidelines"
```

---

## Task 4: Create development-guide.md

**Files:**
- Create: `docs/development-guide.md`
- Reference: `CLAUDE.md` lines 593-751

**Step 1: Create file structure**

Create `docs/development-guide.md`:

```markdown
# Development Guide

> **Quick Reference:** For command summary, see [CLAUDE.md](../CLAUDE.md)

## Prerequisites

## Building Services

### Build Pattern

### Configuration

### Running Services

## Testing

### Running Tests

### Test Coverage

### Testing Specific Packages

## Testing with SQLite Test Databases

### Pattern

### Purpose

### SQL File Format

### Usage in Tests

### Important Notes

### When to Update

## Code Generation

## Linting
```

**Step 2: Extract all sections (lines 593-751)**

Copy prerequisites, building, testing, SQLite patterns, code generation, linting.

**Step 3: Commit**

```bash
git add docs/development-guide.md
git commit -m "docs: create development-guide.md with build and test commands"
```

---

## Task 5: Create code-quality-standards.md

**Files:**
- Create: `docs/code-quality-standards.md`
- Reference: `CLAUDE.md` lines 1482-1722

**Step 1: Create file structure**

Create `docs/code-quality-standards.md`:

```markdown
# Code Quality Standards

> **Quick Reference:** For standards summary, see [CLAUDE.md](../CLAUDE.md#code-quality)

## Overview

All services in the monorepo must follow these standards for consistency.

## Logging Standards

### The Pattern

### Examples

### Key Points

### Import Pattern

### Benefits

## Go Naming Conventions

### The Rule

### Naming Patterns

### Test Function Names

### Function Comments

### Why This Matters

## Common Gotchas

### UUID Fields and DB Tags

#### The Rule

#### Why This Matters

#### Example Bug

#### How to Fix

#### When to Verify
```

**Step 2: Extract logging standards (lines 1490-1621)**

Copy complete logging patterns with examples.

**Step 3: Extract Go naming conventions (lines 1623-1667)**

Copy List vs Gets naming guidance.

**Step 4: Extract UUID gotcha (lines 1669-1722)**

Copy UUID db tag requirements.

**Step 5: Commit**

```bash
git add docs/code-quality-standards.md
git commit -m "docs: create code-quality-standards.md with logging and naming rules"
```

---

## Task 6: Create architecture-deep-dive.md

**Files:**
- Create: `docs/architecture-deep-dive.md`
- Reference: `CLAUDE.md` lines 1024-1228

**Step 1: Create file structure**

Create `docs/architecture-deep-dive.md`:

```markdown
# Architecture Deep Dive

> **Quick Reference:** For architecture summary, see [CLAUDE.md](../CLAUDE.md)

## Service Categories

### Core API & Gateway

### Call & Media Management

### AI & Automation

### Queue & Routing

### Customer & Agent Management

### Campaign & Outbound

### Messaging & Communication

### Infrastructure & Utilities

## Inter-Service Communication

### RabbitMQ RPC Pattern

### Queue Naming Convention

### Making Inter-Service Requests

### Event Publishing

## Configuration Management

### Configuration Precedence

### Common Configuration Patterns

### Common Configuration Fields

## Package Organization Pattern

### Standard Structure

### Handler Pattern

## Database & Caching

### MySQL

### Redis

### Pattern

## Testing Patterns

### Mock Generation

### Test Structure

### Running Tests

## CircleCI Continuous Integration
```

**Step 2: Extract all sections (lines 1024-1228)**

Copy service categories, communication patterns, configuration, package organization, database, testing, CI.

**Step 3: Commit**

```bash
git add docs/architecture-deep-dive.md
git commit -m "docs: create architecture-deep-dive.md with service and system design"
```

---

## Task 7: Create common-workflows.md

**Files:**
- Create: `docs/common-workflows.md`
- Reference: `CLAUDE.md` lines 753-949 and 1230-1393

**Step 1: Create file structure**

Create `docs/common-workflows.md`:

```markdown
# Common Workflows

> **Quick Reference:** For workflow overview, see [CLAUDE.md](../CLAUDE.md#common-workflows)

## Adding a New API Endpoint

## Creating a New Flow Action

## Adding a New Manager Service

## Modifying Shared Models

## Parsing Filters from Request Body

### Pattern

### Why This Is Critical

### Reference Implementation

### How It Works

### Why This Pattern

### Migration Notes

## Database Migrations with Alembic

### Overview

### AI Security Boundaries

### When to Create Migrations

### Migration Workflow

### Migration Best Practices

### Common Migration Patterns

### Troubleshooting

### Important Notes
```

**Step 2: Extract workflow sections (lines 1230-1268)**

Copy adding API endpoints, flow actions, services, modifying shared models.

**Step 3: Extract filter parsing (lines 1270-1393)**

Copy complete filter parsing pattern with examples.

**Step 4: Extract Alembic migrations (lines 753-949)**

Copy complete Alembic section with workflows and examples.

**Step 5: Commit**

```bash
git add docs/common-workflows.md
git commit -m "docs: create common-workflows.md with implementation patterns"
```

---

## Task 8: Create reference.md

**Files:**
- Create: `docs/reference.md`
- Reference: `CLAUDE.md` lines 951-1023, 1425-1736

**Step 1: Create file structure**

Create `docs/reference.md`:

```markdown
# Reference

> **Quick Reference:** For reference overview, see [CLAUDE.md](../CLAUDE.md#reference)

## API Design Principles

### Atomic API Responses

#### The Rule

#### Why

#### Examples

#### Exceptions

#### How to Fetch Related Data

## Key Dependencies

### All Services

### Common Tools

### API Gateway Specific

### Cloud Integration

## Deployment

### Kubernetes

### Infrastructure Requirements

## Important Notes

### Monorepo-Specific Practices

### Communication Patterns

## Security Considerations

## Resources
```

**Step 2: Extract API design (lines 951-1022)**

Copy atomic API response principles.

**Step 3: Extract dependencies (lines 1425-1449)**

Copy all dependency lists.

**Step 4: Extract deployment (lines 1451-1465)**

Copy Kubernetes and infrastructure info.

**Step 5: Extract important notes (lines 1467-1480)**

Copy monorepo practices and communication patterns.

**Step 6: Extract security and resources (lines 1724-1736)**

Copy security considerations and resource links.

**Step 7: Commit**

```bash
git add docs/reference.md
git commit -m "docs: create reference.md with API design, dependencies, and deployment"
```

---

## Task 9: Create new streamlined CLAUDE.md

**Files:**
- Create: `CLAUDE.md.new` (temporary)
- Reference: Design doc example structure

**Step 1: Create new CLAUDE.md header**

Create `CLAUDE.md.new`:

```markdown
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Scope

**This CLAUDE.md applies to ALL services in this monorepo.**

Whether you're working in `bin-api-manager`, `bin-flow-manager`, `bin-call-manager`, or any other service directory, the guidelines, commands, and architectural patterns described here apply uniformly across the entire monorepo.

**Service-Specific Documentation:**
- Some individual services have their own `CLAUDE.md` files with service-specific details
- **When conflicts arise, the service-specific CLAUDE.md takes precedence** over this root file
- Use the root CLAUDE.md for general monorepo patterns and service-specific CLAUDE.md for implementation details
- If a service's CLAUDE.md specifies different commands, architecture, or workflows, follow the service-specific guidance

## Overview

This is the VoIPbin monorepo - a unified backend codebase for a cloud-native CPaaS (Communication Platform as a Service) platform. The repository contains 30+ Go microservices that collectively provide VoIP, messaging, conferencing, AI integration, and communication workflow orchestration capabilities.

**Key Characteristics:**
- **Monorepo architecture** - All backend services in one repository with local module replacements
- **Microservices communication** - Services communicate via RabbitMQ RPC, not HTTP
- **Shared infrastructure** - Common MySQL database, Redis cache, RabbitMQ message broker
- **Event-driven architecture** - Pub/sub events via RabbitMQ and ZeroMQ
- **Kubernetes deployment** - Services designed for GCP GKE with Prometheus monitoring
```

**Step 2: Add critical sections with links**

Add the remaining sections following the design doc structure:
- CRITICAL: Before Committing Changes
- Git Workflow
- Build & Development (link)
- Architecture (link)
- Common Workflows (link)
- Code Quality
- Where to Document New Information (preserve decision tree)
- Reference (link)

**Step 3: Verify line count**

Run: `wc -l CLAUDE.md.new`
Expected: 200-300 lines

**Step 4: Commit preparation file**

```bash
git add CLAUDE.md.new
git commit -m "docs: create streamlined CLAUDE.md.new"
```

---

## Task 10: Extract "Where to Document" decision tree

**Files:**
- Read: `CLAUDE.md` lines 1395-1423
- Modify: `CLAUDE.md.new`

**Step 1: Extract decision tree section**

Copy lines 1395-1423 from current CLAUDE.md (the decision tree).

**Step 2: Add to CLAUDE.md.new**

Paste decision tree into appropriate location in CLAUDE.md.new.

**Step 3: Commit**

```bash
git add CLAUDE.md.new
git commit -m "docs: add decision tree to streamlined CLAUDE.md"
```

---

## Task 11: Replace old CLAUDE.md with new version

**Files:**
- Delete: `CLAUDE.md` (old)
- Rename: `CLAUDE.md.new` → `CLAUDE.md`

**Step 1: Create backup**

```bash
cp CLAUDE.md CLAUDE.md.backup
```

**Step 2: Replace with new version**

```bash
mv CLAUDE.md.new CLAUDE.md
```

**Step 3: Verify**

Check line count:
```bash
wc -l CLAUDE.md
```
Expected: 200-300 lines (down from 1736)

**Step 4: Commit**

```bash
git add CLAUDE.md CLAUDE.md.backup
git commit -m "docs: replace CLAUDE.md with streamlined version"
```

---

## Task 12: Validate all cross-references

**Files:**
- Test: All `.md` files in docs/ and root

**Step 1: Check links in CLAUDE.md**

Verify these links exist and point to correct files:
- `docs/verification-workflows.md`
- `docs/special-cases.md`
- `docs/git-workflow-guide.md`
- `docs/development-guide.md`
- `docs/code-quality-standards.md`
- `docs/architecture-deep-dive.md`
- `docs/common-workflows.md`
- `docs/reference.md`

**Step 2: Check anchors**

Verify these anchors work:
- `docs/special-cases.md#openapi-sync`
- `docs/git-workflow-guide.md#branch-management`
- `docs/git-workflow-guide.md#main-branch-protection`
- `docs/git-workflow-guide.md#merging-rules`

**Step 3: Check back-references**

Verify all detailed files have "Quick Reference" links back to CLAUDE.md.

**Step 4: Document validation results**

Create checklist in commit message showing all links validated.

**Step 5: Commit if fixes needed**

```bash
git add docs/*.md CLAUDE.md
git commit -m "docs: fix cross-reference links and anchors"
```

---

## Task 13: Final review and cleanup

**Files:**
- Review: All modified files

**Step 1: Review main CLAUDE.md**

Check:
- Line count is 200-300
- All critical rules present
- All links clear and explicit
- Decision tree preserved
- Scope and Overview unchanged

**Step 2: Spot-check detailed files**

Verify:
- verification-workflows.md has all workflows
- special-cases.md has bin-common-handler section
- git-workflow-guide.md has commit examples
- development-guide.md has build commands
- code-quality-standards.md has logging patterns
- architecture-deep-dive.md has service list
- common-workflows.md has Alembic section
- reference.md has dependencies

**Step 3: Check for orphaned content**

Search for sections in CLAUDE.md.backup that might be missing from new structure.

**Step 4: Remove backup**

```bash
rm CLAUDE.md.backup
```

**Step 5: Final commit**

```bash
git add CLAUDE.md.backup
git commit -m "docs: remove CLAUDE.md backup after successful migration"
```

---

## Task 14: Verify success criteria

**Files:**
- Check: All documentation files

**Step 1: Check line counts**

Run:
```bash
wc -l CLAUDE.md
wc -l docs/verification-workflows.md
wc -l docs/special-cases.md
wc -l docs/git-workflow-guide.md
wc -l docs/development-guide.md
wc -l docs/code-quality-standards.md
wc -l docs/architecture-deep-dive.md
wc -l docs/common-workflows.md
wc -l docs/reference.md
```

Expected:
- CLAUDE.md: 200-300 lines
- All detailed files: populated with content

**Step 2: Verify all files exist**

```bash
ls -lh docs/*.md CLAUDE.md
```

Expected: 8 files (7 detailed + 1 main)

**Step 3: Check total content preservation**

Compare total lines:
```bash
wc -l CLAUDE.md docs/*.md | tail -1
```

Should be roughly equal to original 1736 lines (content redistributed, not lost).

**Step 4: Create summary commit**

```bash
git commit --allow-empty -m "docs: CLAUDE.md split complete - 1736 lines → 8 focused files"
```

---

## Success Criteria

- ✅ Main CLAUDE.md is 200-300 lines (down from 1736)
- ✅ All 7 detailed files created and populated
- ✅ No content lost during migration
- ✅ All links functional
- ✅ Critical rules remain in main file
- ✅ All commits follow monorepo standards

## Next Steps

After implementation:
1. Push branch to remote
2. Create pull request
3. Request review from team
4. Merge to main after approval
