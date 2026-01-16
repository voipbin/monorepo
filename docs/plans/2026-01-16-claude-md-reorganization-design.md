# CLAUDE.md Reorganization Design

**Date:** 2026-01-16
**Status:** Design Complete
**Scope:** Monorepo-wide documentation structure

## Problem Statement

The root `/home/pchero/gitvoipbin/monorepo/CLAUDE.md` has grown to 1,544 lines. It mixes monorepo-wide rules, architectural documentation, and service-specific implementation details. This creates confusion about where to document new information and makes it hard to find service-specific guidance.

**Current issues:**
- Service-specific content (Permission Requirements, Flow-Manager patterns) lives in root file
- Unclear boundaries between root and service CLAUDE.md files
- New documentation gets added to root by default, causing continued growth
- Hard to find service-specific implementation details

## Solution Overview

Establish clear boundaries between root and service-specific CLAUDE.md files:

**Root CLAUDE.md:**
- Monorepo-wide rules and workflows (applies to ALL services)
- Full architectural overview (service categories, RabbitMQ patterns, configuration)
- Shared patterns used across multiple services (with notes explaining scope)
- Cross-service workflows (adding API endpoints, modifying shared models)

**Service CLAUDE.md:**
- Service-specific implementation guide
- API endpoints and RPC methods for this service
- Service-specific handler patterns and testing approaches
- Unique workflows and domain-specific implementation details

**Precedence rule (already documented):**
When conflicts arise, service-specific CLAUDE.md takes precedence over root CLAUDE.md.

## Design Decisions

### Decision 1: Detail Level for Service Files

**Chosen:** Implementation guide level (Option B)

Service-specific CLAUDE.md files include:
- How to use this service (API endpoints, RPC methods)
- Service-specific implementation patterns (handlers, testing, data models)
- Common tasks and workflows for this service

Example: `bin-call-manager/CLAUDE.md` (332 lines) provides comprehensive service documentation.

### Decision 2: Cross-Cutting Patterns

**Chosen:** Keep at root with explanatory notes (Option A + notes)

Patterns like "Parsing Filters from Request Body" affect multiple services but not all:
- Keep pattern documentation at root CLAUDE.md
- Add note explaining what the pattern is for
- Note which services use it and when to apply it

**Example note format:**
```markdown
### Pattern Name

**Note:** This is a [scope] pattern for [which services/use cases].
[Additional context about when/why to use it.]

[Pattern documentation...]
```

### Decision 3: Multi-Service Workflows

**Chosen:** Keep at root (Option A)

Workflows spanning multiple services (like "Adding a New API Endpoint") stay in root:
- These workflows touch 3+ services (bin-openapi-manager, bin-api-manager, target service)
- Root provides end-to-end workflow
- Service-specific files can reference root workflow and add service-specific details

### Decision 4: Root Architecture Section

**Chosen:** Full architectural guide (Option C)

Root CLAUDE.md keeps comprehensive architecture documentation:
- Service categories with one-line descriptions
- Inter-service communication (RabbitMQ RPC patterns)
- Configuration management (Cobra + Viper)
- Package organization patterns
- Database & caching strategies
- Testing patterns
- CircleCI CI/CD setup

This provides the architectural context needed to understand the entire monorepo.

## Root CLAUDE.md Structure

```
1. Scope & Overview
   - Monorepo boundaries and precedence rules
   - VoIPbin platform overview
   - Key characteristics

2. Workflows & Rules (Mandatory processes)
   - Verification workflows (go mod tidy, go test, golangci-lint)
   - Git workflow (commit messages, branching, merging)
   - Dependency update workflows
   - Special cases (bin-common-handler, OpenAPI schemas)
   - Database migrations with Alembic

3. Architecture (Full architectural guide)
   - Service categories with descriptions
   - Inter-service communication (RabbitMQ RPC patterns)
   - Configuration management (Cobra + Viper)
   - Package organization patterns
   - Database & caching strategies
   - Testing patterns
   - CircleCI CI/CD

4. Shared Patterns (Cross-cutting patterns)
   - Parsing Filters from Request Body (with note)
   - UUID Fields and DB Tags (with note)
   - API Design Principles (atomic responses)
   - Other patterns used across multiple services

5. Common Workflows (Multi-service workflows)
   - Adding a New API Endpoint
   - Adding a New Manager Service
   - Modifying Shared Models

6. Decision Framework (For maintainers)
   - Where to document new information
   - Decision tree for root vs service files

7. References
   - Links to resources
   - Links to service-specific CLAUDE.md files
```

## Service-Specific CLAUDE.md Structure

```
1. Overview
   - What this service does
   - Key concepts

2. Architecture
   - Service communication pattern (RabbitMQ queues)
   - Core components and handler dependency chain
   - Request routing (API endpoints)
   - Event subscriptions

3. Common Commands
   - Build, test, generate, lint, run locally

4. Service-Specific Patterns
   - Unique implementation patterns
   - Service-specific workflows
   - Testing patterns specific to this service

5. Key Implementation Details
   - Domain-specific logic
   - Integration points
   - Handler dependencies
```

## Content Migration Plan

### Content Moving from Root to Service Files

**→ bin-api-manager/CLAUDE.md:**
- "Authentication & Authorization Pattern" section
  - JWT validation
  - Customer ID extraction from JWT
  - Authorization flow examples
- "Permission Requirements" section
  - Permission checking in servicehandler
  - Resource access patterns with hasPermission()
  - Example code for authorization

**→ bin-flow-manager/CLAUDE.md:**
- "Working with Flow-Manager" section
  - Flow execution pattern (Flow vs Activeflow)
  - Stack-based execution for nested flows
  - Variable substitution patterns (${variable.key})
- "Creating a New Flow Action" workflow
  - Define action type in models
  - Add action handler
  - Implement execution logic

**→ bin-billing-manager/CLAUDE.md:**
- Billing-specific permission note
  - Extract from "Permission Requirements"
  - CustomerAdmin-only requirement for billing resources

### Content Staying at Root with Added Notes

**"Parsing Filters from Request Body":**
- Stays at root (monorepo-wide pattern)
- Add note: "**Note:** This is a monorepo-wide pattern for all services that expose list endpoints (GET /v1/resources). Implemented using `bin-common-handler/pkg/utilhandler`. See individual service CLAUDE.md files for service-specific filter field definitions."

**"UUID Fields and DB Tags":**
- Stays at root (affects all services using commondatabasehandler)
- Add note: "**Note:** This affects all services using the `commondatabasehandler` pattern. Critical for database queries to work correctly across the monorepo."

**"API Design Principles":**
- General principles stay at root (Atomic API Responses)
- Service-specific auth implementation moves to bin-api-manager

### Content to Remove

**"Creating a New Flow Action":**
- Remove from root "Common Workflows"
- Move to bin-flow-manager/CLAUDE.md

## Decision Framework for Future Maintainers

Add this section to root CLAUDE.md to guide future documentation:

```
## Where to Document New Information

Use this decision tree when adding new documentation:

Does this apply to ALL services in the monorepo?
├─ YES → Add to Root CLAUDE.md
│   └─ Examples:
│       - New verification step required for all services
│       - New git workflow rule
│       - New shared pattern from bin-common-handler
│       - Changes to RabbitMQ communication pattern
│
└─ NO → Does it affect 2+ services?
    ├─ YES → Is it a workflow spanning services?
    │   ├─ YES → Add to Root CLAUDE.md "Common Workflows"
    │   │   └─ Example: "Adding a New API Endpoint"
    │   │
    │   └─ NO → Is it a shared pattern/gotcha?
    │       ├─ YES → Add to Root "Shared Patterns" with note
    │       │   └─ Example: "Parsing Filters from Request Body"
    │       │
    │       └─ NO → Add to each affected service's CLAUDE.md
    │           └─ Example: Recording handling in conference/call managers
    │
    └─ NO → Add to Service-Specific CLAUDE.md
        └─ Examples:
            - Service-specific API endpoints
            - Handler patterns unique to this service
            - Service-specific testing approaches
            - Domain-specific implementation details
```

## Implementation Plan

### Phase 1: Preparation
1. Create backup branch: `git checkout -b NOJIRA-Claude_md_reorganization_backup`
2. Identify services needing CLAUDE.md updates:
   - bin-api-manager (exists, needs auth sections)
   - bin-flow-manager (check if exists, add flow sections)
   - bin-billing-manager (exists, needs permission note)
3. Switch to feature branch: `git checkout -b NOJIRA-Reorganize_CLAUDE_md_structure`

### Phase 2: Content Extraction
1. Extract sections from root CLAUDE.md to temporary files:
   - Save "Authentication & Authorization Pattern" section
   - Save "Permission Requirements" section
   - Save "Working with Flow-Manager" section
   - Save "Creating a New Flow Action" section
   - Extract billing-specific permission note

2. Add notes to shared patterns in root CLAUDE.md:
   - Add note to "Parsing Filters from Request Body" (line ~1291)
   - Add note to "UUID Fields and DB Tags" (under Common Gotchas)

### Phase 3: Service File Updates
1. Update `bin-api-manager/CLAUDE.md`:
   - Read existing file to understand current structure
   - Add "Authentication & Authorization" section
   - Add "Permission Requirements" section
   - Ensure consistency with service structure template

2. Update `bin-flow-manager/CLAUDE.md`:
   - Check if file exists, create if needed
   - Add "Flow Execution Patterns" section
   - Add "Creating Flow Actions" workflow
   - Document variable substitution

3. Update `bin-billing-manager/CLAUDE.md`:
   - Read existing file
   - Add "Permission Requirements" section
   - Note CustomerAdmin-only access

### Phase 4: Root File Reorganization
1. Remove migrated content from root CLAUDE.md:
   - Delete "Authentication & Authorization Pattern" (keep reference in API Design Principles)
   - Delete "Permission Requirements"
   - Delete "Working with Flow-Manager"
   - Delete "Creating a New Flow Action"

2. Add "Decision Framework" section:
   - Insert after "Common Workflows"
   - Before "References" section

3. Add notes to shared patterns:
   - "Parsing Filters from Request Body"
   - "UUID Fields and DB Tags"

4. Update References section:
   - Add links to service-specific CLAUDE.md files
   - List all services with CLAUDE.md files

### Phase 5: Verification
1. Check all cross-references:
   - Search for broken section references
   - Verify service names are accurate
   - Check code examples are complete

2. Verify content accuracy:
   - Ensure no context lost during migration
   - Check technical details preserved
   - Verify examples still make sense

3. Review for consistency:
   - Consistent heading levels
   - Consistent code block formatting
   - Consistent note format for shared patterns

4. Test Claude Code behavior:
   - Verify precedence rule works
   - Check that service-specific CLAUDE.md loads correctly

## Files Affected

```
/home/pchero/gitvoipbin/monorepo/
├── CLAUDE.md (modify - reorganize, remove migrated sections, add notes)
├── bin-api-manager/CLAUDE.md (modify - add auth sections)
├── bin-flow-manager/CLAUDE.md (create or modify - add flow sections)
├── bin-billing-manager/CLAUDE.md (modify - add permission note)
└── docs/plans/2026-01-16-claude-md-reorganization-design.md (create)
```

## Risks & Mitigation

**Risk: Breaking existing references in service files**
- Mitigation: Search for references to moved sections before deleting
- Search pattern: `grep -r "Authentication & Authorization Pattern" bin-*/CLAUDE.md`

**Risk: Losing important context during migration**
- Mitigation: Review each section carefully, preserve all technical details
- Keep backup branch for reference

**Risk: Future confusion about where to add documentation**
- Mitigation: Add explicit "Decision Framework" section to root CLAUDE.md
- Make decision tree simple and clear

**Risk: Service-specific files becoming inconsistent**
- Mitigation: Define standard structure template for all service CLAUDE.md files
- Review existing files for compliance

## Success Criteria

**Root CLAUDE.md:**
- Contains only monorepo-wide rules, architecture, and shared patterns
- All shared patterns have explanatory notes
- Decision framework guides future documentation
- Total length reduced by ~300-400 lines

**Service CLAUDE.md files:**
- bin-api-manager has complete auth/permission documentation
- bin-flow-manager has flow execution patterns and workflows
- bin-billing-manager has permission requirements
- All follow consistent structure template

**Maintainability:**
- Clear decision tree for where to document new information
- No ambiguity about root vs service boundaries
- Easy to find service-specific implementation details

## Future Enhancements

**Phase 2 (Optional):**
- Create service CLAUDE.md files for services that don't have them
- Standardize all existing service CLAUDE.md files to match template
- Add cross-reference links between related services

**Phase 3 (Optional):**
- Extract architecture documentation to separate `docs/architecture.md`
- Create `docs/development-guide.md` for onboarding new developers
- Add diagrams for RabbitMQ communication patterns

## References

- Current root CLAUDE.md: `/home/pchero/gitvoipbin/monorepo/CLAUDE.md` (1,544 lines)
- Example service file: `/home/pchero/gitvoipbin/monorepo/bin-call-manager/CLAUDE.md` (332 lines)
- Filter parsing plan: `docs/plans/2026-01-14-listenhandler-filter-parsing-implementation-plan.md`
