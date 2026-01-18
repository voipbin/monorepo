# Verification Workflows

> **Quick Reference:** For commands, see [CLAUDE.md](../CLAUDE.md#critical-before-committing-changes)

## Overview

This document provides detailed explanations for verification workflows that must run before committing any code changes in the monorepo.

**⚠️ MANDATORY: ALWAYS run the verification workflow after making ANY code changes and BEFORE committing.**

This applies to ALL changes: code modifications, refactoring, bug fixes, new features, or any other changes. No exceptions.

## Regular Code Changes Workflow

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

**What this does:**
1. `go mod tidy` - Cleans up go.mod and go.sum files
2. `go mod vendor` - Vendors dependencies for reproducible builds
3. `go generate ./...` - Regenerates mocks and generated code
4. `go test ./...` - Runs all tests to ensure nothing broke
5. `golangci-lint run -v --timeout 5m` - Lints code for quality issues

**This runs AFTER making changes but BEFORE `git commit`.**

## Dependency Update Workflow

**Only when specifically updating dependencies, run this workflow:**

```bash
# Navigate to the service directory
cd bin-<service-name>

# Run the full update workflow (WITH dependency updates)
go get -u ./... && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Why separate workflows?**
- `go get -u ./...` updates ALL dependencies to latest versions
- Mixing dependency updates with feature changes makes PR review harder
- Dependency updates should be separate commits/PRs when possible
- For regular code changes, only update dependencies if needed

**Both workflows are MANDATORY before committing** - Do not skip any step. The monorepo's interdependencies require this to maintain consistency and catch issues early.
