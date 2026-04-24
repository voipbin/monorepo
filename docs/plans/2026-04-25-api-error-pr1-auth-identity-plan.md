# PR 1 — Auth & Identity handler migration (trimmed scope)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to execute this plan.

**Goal:** Migrate `bin-api-manager`'s customer-facing auth/identity handlers from bare `c.AbortWithStatus(...)` to the new canonical error envelope delivered by PR 0a/0b. This is the first real handler migration in the 9-group rollout.

**Parent:** `docs/plans/2026-04-24-api-error-response-codes-design.md`

**Dependency:** PR 0a (merged `6d5ce5412`), PR 0b (merged `f301e96b9`), NoRoute hygiene (merged `88900760c`). All shipped. Branch is based on latest `main`.

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr1-auth-identity`
**Branch:** `NOJIRA-api-error-pr1-auth-identity` (branched from `origin/main` at `88900760c`)

## Scope (trimmed per user instruction)

**In scope for this PR:**
- `server/auth_boot.go` — 1 stub (`PostAuthBoot`)
- `server/auth_signup.go` — 2 stubs (`PostAuthSignup`, `PostAuthEmailVerify`)
- `server/auth_unregister.go` — 2 stubs (`PostAuthUnregister`, `DeleteAuthUnregister`)
- `server/customer.go` — 5 real handlers, 19 `c.AbortWithStatus(400)` sites
- Accompanying `*_test.go` files — add coverage where needed
- OpenAPI spec updates for `/customer*` paths (add 401/500 response refs; the auth stubs live outside `/v1.0/` so OpenAPI doesn't wire them)
- Regenerate `bin-api-manager/gens/openapi_server/gen.go`
- RST catalog updates: append new reasons to `restful_api_errors.rst` under `api-manager` domain
- Sphinx HTML rebuild

**Out of scope (deferred to PR 1b):**
- `server/customers.go` — admin-only `/customers` and `/customers/{id}` routes
- `server/service_agents_me.go`, `server/service_agents_customer.go` — agent-UI surface

**Not needed (already migrated):**
- `server/me.go` + `me_test.go` — done in PR 0b

## Migration patterns

### Pattern A: Stub handler (auth_*.go)

The 5 auth stubs currently do:

```go
func (h *server) PostAuthBoot(c *gin.Context) {
    c.AbortWithStatus(404)
}
```

Migrate to use `abortWithError` with `ROUTE_NOT_FOUND` reason (same as the NoRoute handler — these stubs effectively return "wrong route" because the real handler lives elsewhere):

```go
func (h *server) PostAuthBoot(c *gin.Context) {
    abortWithError(c, cerrors.NotFound(
        commonoutline.ServiceNameAPIManager,
        "ROUTE_NOT_FOUND",
        "The requested endpoint does not exist on this path; see /auth/boot.",
    ))
}
```

Adding a pointer to the correct path in the message helps clients debug. Keep the doc comment (currently says "stub to satisfy the generated ServerInterface" — keep this).

### Pattern B: Real handler (customer.go)

Each handler has up to 4 error sites. Mapping:

| Site | Before | After |
|---|---|---|
| `getAuthIdentity(c)` fails | `c.AbortWithStatus(400)` | `abortWithError(c, cerrors.Unauthenticated(SN, "AUTHENTICATION_REQUIRED", "Authentication is required."))` |
| `c.BindJSON(&req)` fails | `c.AbortWithStatus(400)` | `abortWithError(c, cerrors.InvalidArgument(SN, "INVALID_JSON_BODY", "The request body is not valid JSON."))` |
| UUID parse fails | `c.AbortWithStatus(400)` | `abortWithError(c, cerrors.InvalidArgument(SN, "INVALID_ID", "The provided id is not a valid UUID."))` |
| `serviceHandler.X` fails | `c.AbortWithStatus(400)` | `abortWithServiceError(c, err)` (translator handles) |

`SN` = `commonoutline.ServiceNameAPIManager` for auth/parse errors originating in the server layer. The servicehandler layer owns its own domain attribution when it constructs typed errors.

**Per-site log message preservation:** keep the existing `log.Errorf(...)` / `log.Infof(...)` lines in place — the translator logs a structured line too, but site-specific context (`"Could not parse the request"`) is still useful for grep.

### Pattern C: OpenAPI path-level error responses

For `/customer*` paths, add to each endpoint's `responses:` block:

```yaml
        '401':
          $ref: '#/components/responses/Unauthenticated'
        '500':
          $ref: '#/components/responses/InternalError'
```

No 400/404/403 refs yet — too noisy and the spike proved additive refs don't break gen.go. Add the minimum.

## Tasks

### Task 1: Migrate auth stubs (auth_boot.go, auth_signup.go, auth_unregister.go)

5 handlers across 3 files. All identical pattern — replace `c.AbortWithStatus(404)` with `abortWithError(ROUTE_NOT_FOUND)`.

Add minimal tests — assert one stub returns the new envelope (no need to test all 5; they're copy-paste).

Add imports: `cerrors`, `commonoutline`.

### Task 2: Migrate customer.go

5 handlers, 19 error sites. Apply Pattern B table above. Each handler gets:
- Auth-identity check migrated
- BindJSON migrated
- UUID parse migrated (where present)
- servicehandler error migrated via `abortWithServiceError`

Update the existing `customer_test.go` (if any) or add new tests covering at least the auth-identity and BindJSON error paths via `assertErrorResponse`.

### Task 3: OpenAPI spec updates

Wire 401 + 500 response refs into each `/customer*` path in `bin-openapi-manager/openapi/openapi.yaml` (or wherever the customer paths live). Regenerate:

```bash
cd bin-openapi-manager && go generate ./...
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./...
```

Verify `gens/openapi_server/gen.go` regenerates cleanly.

### Task 4: RST catalog updates

`bin-api-manager/docsdev/source/restful_api_errors.rst` — append to the `api-manager` section:

- `INVALID_JSON_BODY` (400) — request body is not valid JSON
- `INVALID_ID` (400) — path/body parameter is not a valid UUID

(`AUTHENTICATION_REQUIRED`, `ROUTE_NOT_FOUND`, `RESOURCE_NOT_FOUND` already cataloged.)

Clean rebuild Sphinx HTML.

### Task 5: Full verification

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test -race ./... && golangci-lint run -v --timeout 5m
```

### Task 6: Push + open PR

Conflict check, push, open PR #802 with body linking back to the design doc and noting the trimmed scope (PR 1b follow-up for `customers.go` + `service_agents_*`).

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr1-auth-identity` on every commit.
- No AI attribution.
- Use `outline.ServiceName` constants (per PR 0a Round 1 refinement), never string literals.
- Do NOT use `fmt.Errorf("...")` for new errors — construct typed `VoipbinError` via constructors.
- Existing `log.Errorf/Infof` site-specific context lines stay in place.
- RST catalog entries follow the AI-native format (Cause → Fix pairs).
- Sphinx HTML rebuilt + force-added.

## Success criteria

- All 6 tasks committed
- `go test -race ./...` green in both services
- `golangci-lint run -v --timeout 5m` 0 issues in both
- Live test against staging/prod: a bad JSON body on `PUT /v1.0/customer` returns the new envelope
- PR 1b plan noted as follow-up
