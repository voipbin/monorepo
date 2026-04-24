# PR 1b — Admin & Agent-UI handler migration (finishes the auth-and-identity group)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Close out the "auth & identity" PR group by migrating the admin `/customers*` surface and the `/service-agents/me` + `/service-agents/customer` surface. Also codify the "default error-block per endpoint class" convention in the design doc so PR 2+ stops guessing.

**Parent:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 1 (`NOJIRA-api-error-pr1-auth-identity`, merged `331c05c09`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr1b-admin-agent-ui`
**Branch:** `NOJIRA-api-error-pr1b-admin-agent-ui` (branched from `origin/main` at `331c05c09`)

## Scope

| File | LoC | `c.AbortWithStatus` sites | Handlers |
|---|---|---|---|
| `server/customers.go` | 401 | 35 | 10 |
| `server/service_agents_me.go` | 175 | 14 | 5 |
| `server/service_agents_customer.go` | 33 | 2 | 1 |

Total: **51 sites across 16 handlers.**

Admin-only `customers.go` is *not* a normal customer endpoint — it handles the full CRUD for customers by an admin. The HTTP status meaning for 404 / 403 is real here (resource-keyed paths exist).

## Migration patterns

Same as PR 1:
- `getAuthIdentity(c)` failure → `abortWithError(Unauthenticated, AUTHENTICATION_REQUIRED)`
- `c.BindJSON` failure → `abortWithError(InvalidArgument, INVALID_JSON_BODY)`
- `uuid.FromStringOrNil(...) == uuid.Nil` → `abortWithError(InvalidArgument, INVALID_ID)`
- `serviceHandler.X(...)` failure → `abortWithServiceError(c, err)`

## Tasks

### Task 1: Migrate `server/customers.go`

10 handlers, 35 sites. Same pattern as `customer.go` in PR 1 but scaled. Add error-path tests covering:
- One `_MissingAuthIdentity` per GET, one per PUT, one per DELETE (sample, not exhaustive — the pattern is uniform).
- One `_InvalidJSONBody` test for a PUT handler.
- One `_InvalidID` test for a handler that parses UUIDs from the path (`/customers/{id}`).
- One `_ServiceError` test for a GET handler with a resource-keyed path (translator mapping to NOT_FOUND).

### Task 2: Migrate `server/service_agents_me.go` + `server/service_agents_customer.go`

5 handlers in `service_agents_me.go` (14 sites), 1 handler in `service_agents_customer.go` (2 sites).
Add error-path tests mirroring Task 1's patterns but sampled tighter (these are agent-UI endpoints, lower blast radius).

### Task 3: OpenAPI path wiring

Wire 400/401/500 into `/customers` and `/customers/{id}` paths. Add 403 + 404 to `/customers/{id}` (resource-keyed). Same for `/service-agents/me/*` and `/service-agents/customer`.

Regenerate `bin-api-manager/gens/openapi_server/gen.go`. Confirm loose `ServerInterface` signatures unchanged (same pattern as PR 1).

### Task 4: RST catalog + convention codification

Two docs updates in one commit:

**4a. RST catalog** — if PR 1b emits any new reason codes (unlikely since the patterns are identical to PR 1), append. Otherwise skip RST.

**4b. Design-doc convention** — append a new subsection under §7 or §10 documenting the default error-block per endpoint class:

```markdown
### OpenAPI default error-block convention (established as of PR 1/1b)

For each endpoint under `/v1.0/*`, the OpenAPI path `responses:` block SHOULD reference the named error responses below as a baseline. Endpoints with special semantics (billing → 402, conflict-able writes → 409, rate-limited paths → 429) add the corresponding response on top of this baseline.

| Endpoint class | Baseline error responses |
|---|---|
| Read (GET, no path param) | `401`, `500` |
| Read (GET with resource ID) | `401`, `403`, `404`, `500` |
| Write (POST / PUT / PATCH / DELETE, no resource ID) | `400`, `401`, `500` |
| Write (POST / PUT / PATCH / DELETE with resource ID) | `400`, `401`, `403`, `404`, `500` |
| Billing-sensitive | baseline + `402` |
| Rate-limited | baseline + `429` |

Rationale: matches the actual translator output — every endpoint can plausibly return INTERNAL (500), any authenticated endpoint can return UNAUTHENTICATED (401), writes can return INVALID_ARGUMENT (400), resource-keyed endpoints can return NOT_FOUND (404) and PERMISSION_DENIED (403). Adding 409/429/402 only where the code path exists keeps the schema honest.

`/me` (read, no path param) uses the minimum `401, 500`. `/customer*` (write, no path param) uses `400, 401, 500`. `/customers/{id}` (write, path param) adds `403, 404`.
```

Rebuild Sphinx HTML.

### Task 5: Verification

Standard 5-step workflow for both `bin-openapi-manager` and `bin-api-manager` (6 for api-manager including `-race`). Same as prior PRs.

### Task 6: Push + open PR

Conflict check, push, open PR #804 with body linking back to PR 1 and noting the convention codification.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr1b-admin-agent-ui`.
- No AI attribution.
- Preserve existing `log.Errorf/Infof` site-specific lines.
- Use `commonoutline.ServiceName*` constants (never string literals).
- 3-group import layout in new test files (stdlib / monorepo / third-party).
- Retrofit `/me` OpenAPI path to add `400` response ref per the new convention (read without path param normally wouldn't need it, but `/me` is special — it's not path-keyed but the existing handler has an auth-missing-identity path that can return 401). Actually per the table, `/me` is "Read (GET, no path param)" → 401+500. Current `/me` already has 401+500 only. No retrofit needed.

## Success criteria

- All 6 tasks committed
- `go test -race ./...` green
- `golangci-lint` 0 issues
- Design doc §§ updated with convention table
- PR 2 author has a clear default error-block convention to follow
