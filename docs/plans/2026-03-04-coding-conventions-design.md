# Design: Consolidated Coding Conventions Document

**Date:** 2026-03-04
**Branch:** NOJIRA-Add-coding-conventions-document
**Status:** Draft

## Problem Statement

The VoIPbin monorepo has coding conventions scattered across multiple locations:
- Root `CLAUDE.md` (~31KB) with inline coding rules mixed with workflow/process rules
- 8 supplementary docs in `docs/` with overlapping content (e.g., `reference.md` duplicates `code-quality-standards.md` logging section verbatim)
- 36 service-specific `CLAUDE.md` files with local patterns
- Auto-memory files capturing 30+ patterns from past sessions
- Implicit conventions in the code that are not documented anywhere

This causes problems:
1. **Inconsistency across Claude sessions** — Different sessions may follow different rules depending on which docs they read
2. **Contradictions** — `database-patterns-checklist.md` shows manual `rows.Scan()` examples while root CLAUDE.md mandates `commondatabasehandler.ScanRow()` only
3. **Undocumented patterns** — Import ordering, error handling style, test naming, variable naming conventions exist in code but aren't formally documented
4. **Duplication** — Same rules appear in multiple places, leading to drift when one is updated but not others

## Goal

Create a single, authoritative `docs/coding-conventions.md` that:
- Codifies ALL coding conventions (both currently documented and implicit in the codebase)
- Uses a consistent format: rule statement + correct example + anti-pattern + rationale
- Eliminates the need to check multiple docs for "how should I write this code?"
- Is referenced from root CLAUDE.md so every Claude session picks it up automatically

## Approach

### Deliverable 1: `docs/coding-conventions.md`

A comprehensive document organized into 15 sections:

#### 1. Package Structure & File Organization
- Standard service directory layout (`cmd/`, `internal/`, `models/`, `pkg/`)
- Two binaries per service: daemon + control CLI
- Model file organization: `<entity>.go`, `field.go`, `event.go`, `webhook.go`
- Where each type of code lives (business logic vs DB vs routing)

#### 2. Naming Conventions
- Function naming: `Get`/`List`/`Create`/`Update`/`Delete`, `GetBy<Criteria>`, `Update<Field>`
- Event handler naming: `Event<ServicePrefix><EventName>`
- Private DB helper naming: `db*` prefix
- Type naming: `Type` for enums, `Field` for map keys, `EventType` prefix for events
- Table name variables: unexported `<entity>Table`
- Import aliases: 2-3 letter prefix from service+model name
- Test variables: `mc`, `h`, `tt`, `mock<Name>`

#### 3. Import Ordering
- Five groups: stdlib → bin-common-handler → cross-service models → third-party → local
- Aliasing convention for cross-service imports
- Examples from actual codebase

#### 4. Error Handling
- Sentinel errors as package-level vars in `dbhandler/main.go`
- Wrapping with `fmt.Errorf("context: %w", err)` or `errors.Wrap(err, "context")`
- Log-then-return pattern
- When to use `errors.Is()` vs direct comparison
- Never return unwrapped errors from business handlers

#### 5. Logging
- Function-scoped logrus logger pattern (first statement in every function)
- `logrus.WithFields` for multiple fields, `logrus.WithField` for single
- Log level guidelines: Debug/Info/Warn/Error with examples
- Structured object logging after data retrieval
- Error format: `"Could not ..., err: %v"`

#### 6. Model Definitions
- `commonidentity.Identity` embedding for ID/CustomerID
- `db` tag conventions: plain, `,uuid`, `,json`, `-`
- `json` tag conventions: `omitempty` for optional fields, `-` for internal fields
- `Field` type definition pattern
- Event constant naming pattern
- Timestamp fields: `TMCreate`, `TMUpdate`, `TMDelete` as `*time.Time`
- `WebhookMessage` struct + `ConvertWebhookMessage()` method
- `CreateWebhookEvent()` for webhook delivery

#### 7. Database Patterns
- Squirrel query builder mandatory (no raw SQL)
- `commondatabasehandler.PrepareFields()` for INSERT/UPDATE
- `commondatabasehandler.GetDBFields()` + `ScanRow()` for SELECT
- Empty slice initialization: `res := []*Type{}`
- Soft delete via TMDelete timestamp
- Cache-aside: cache-first reads, write-through on mutations
- Cursor-based pagination via TMCreate token
- `commondatabasehandler.ApplyFields()` for typed filter maps
- All DB operations in `pkg/dbhandler/` only

#### 8. Handler Architecture
- Interface in `main.go` with `//go:generate mockgen` directive
- Private struct implementing public interface
- Constructor returns interface type
- Two-layer split: public (validation + permission + events) vs private `db*` (DB + cache + notify)
- "Get-after-write + publish" pattern universal for all mutations
- Sub-handler composition for complex services

#### 9. Inter-Service Communication
- `requesthandler.RequestHandler` typed RPC via RabbitMQ
- `listenhandler` regex-based URI+method routing
- Queue naming: `bin-manager.<service-name>.request/event/subscribe`
- `sock.Request`/`sock.Response`/`sock.Event` message types
- Fresh `context.Background()` per incoming RPC request
- HTTP-style status codes in responses

#### 10. API & External Interfaces
- Atomic API responses (single resource type, never combined)
- Two-level servicehandler: private returns internal struct, public returns `*WebhookMessage`
- Permission checks between private fetch and public return
- Filters from request body (not URL params)
- OpenAPI schema must match `WebhookMessage` fields

#### 11. Event Publishing
- `notifyHandler.PublishWebhookEvent()` for both internal event + customer webhook
- Fire-and-forget via goroutines
- Delayed events via `EventPublishWithDelay`
- Event types defined as constants in model package
- ClickHouse replication for analytics (automatic)

#### 12. Configuration
- Cobra + Viper with `sync.Once` singleton
- `Bootstrap()` → `LoadGlobalConfig()` → `Get()` lifecycle
- Environment variable binding via `viper.BindEnv()`
- Standard config fields: RabbitMQ, Database, Redis, Prometheus
- Service-specific fields added to Config struct

#### 13. Testing
- Table-driven tests with anonymous struct slices
- `gomock` from `go.uber.org/mock` for mocks
- Tests instantiate private struct directly (not constructor)
- Hardcoded real-looking UUID strings
- `t.Errorf("Wrong match. expect: ..., got: ...")` assertion style
- `reflect.DeepEqual` for struct comparison
- Test function naming: `Test_<MethodName>`
- Mock naming: `mock_main.go` co-located with interface
- `gomock.Any()` is a matcher only (never in `Return()`)

#### 14. Prometheus Metrics
- Service metrics via `init()` in `metricshandler`
- Must not collide with `requesthandler` auto-registered metrics
- Naming: `<namespace>_<unique_metric_name>`
- Check `requesthandler.initPrometheus()` before adding new metrics

#### 15. Security
- XSS prevention: never inject user input into HTML via `fmt.Sprintf`
- Token generation: `crypto/rand` only, never `math/rand`
- Username enumeration prevention: password-forgot always returns 200
- Guest agent protection in all mutation operations
- Validation at system boundaries only
- No secrets in code, logs, or chat output

### Deliverable 2: Root CLAUDE.md Updates

Update root `CLAUDE.md` to:
1. Add prominent reference to `docs/coding-conventions.md` in the "Code Quality" section
2. Remove inline convention content that's now in the conventions doc:
   - "Debug Logging for Retrieved Data" section → covered in conventions Section 5
   - Parts of "Database Handling Rules" that duplicate conventions Section 7
   - "WebhookMessage Pattern" section → covered in conventions Section 6 and 10
   - "Common Gotchas" subsections that are pure coding conventions
3. Keep these in CLAUDE.md (they're workflow/process, not coding conventions):
   - Verification workflow
   - Git workflow / branch management / commit format / PR format
   - Worktree rules
   - Design & implementation workflow
   - Database migration workflow (Alembic)
   - RST documentation rebuild workflow
   - API testing guidelines
   - OpenAPI update workflow
   - "Where to Document New Information" decision tree
4. Add cross-references: CLAUDE.md → conventions doc, conventions doc → workflow docs where relevant

### Deliverable 3: Fix Known Contradictions

1. Update `docs/database-patterns-checklist.md` to remove manual `rows.Scan()` examples and replace with `commondatabasehandler.ScanRow()` pattern (aligning with root CLAUDE.md mandate)
2. Deduplicate `docs/reference.md` logging section (link to conventions doc instead of duplicating `code-quality-standards.md`)

## What's NOT Changing

- 36 service-specific `CLAUDE.md` files stay as-is (service-specific details remain local)
- 8 `docs/*.md` files remain for workflow/reference content (deduplicated where they overlap with conventions)
- Auto-memory files are unaffected (they serve a different purpose: session-specific patterns)
- No code changes — this is documentation only

## Risk Assessment

- **Low risk**: Documentation-only change, no code modifications
- **Maintenance burden**: One more file to keep in sync. Mitigated by making it THE authoritative source (other docs link to it, don't duplicate)
- **Size concern**: Comprehensive format with examples will make this ~2000-3000 lines. Acceptable for a reference doc that's read on-demand, not loaded into every context window
- **CLAUDE.md size**: After removing duplicated content, root CLAUDE.md should shrink by ~15-20%

## Success Criteria

1. Every coding convention used in the codebase is documented in `docs/coding-conventions.md`
2. No contradictions between conventions doc and CLAUDE.md
3. Root CLAUDE.md references the conventions doc prominently
4. A new Claude session can find the correct convention for any coding question by reading CLAUDE.md → following link to conventions doc
