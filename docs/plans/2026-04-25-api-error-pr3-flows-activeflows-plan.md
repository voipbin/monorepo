# PR 3 — Flows & Activeflows handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the flows + activeflows handlers to the canonical error envelope. Smaller than PR 2 — 29 error sites across 11 handlers in 2 files. Uses the §6.1 convention as locked by PR 1b/PR 2.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 2 (`NOJIRA-api-error-pr2-calls-group`, merged `15c651c87`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr3-flows-activeflows`
**Branch:** `NOJIRA-api-error-pr3-flows-activeflows` (branched from `origin/main` at `15c651c87`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/flows.go` | 241 | 18 | 6 |
| `server/activeflows.go` | 194 | 11 | 5 |

**Total: 29 sites across 11 handlers.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetFlows`, `GetActiveflows`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetFlowsId`, `GetActiveflowsId`

**Write (no resource ID) → 400, 401, 500:**
- `PostFlows`, `PostActiveflows` — not billing-sensitive (flow definitions don't deduct credits; activeflow creation triggers downstream resources but the activeflow itself is free), not RPC-heavy (single-manager fan-out to flow-manager)

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `PutFlowsId`, `DeleteFlowsId`, `PostFlowsIdDirectHashRegenerate`, `DeleteActiveflowsId`

**State-transition (+409):**
- `PostActiveflowsIdStop` — stopping an already-stopped activeflow → `FLOW_STATE_INVALID` / `ACTIVEFLOW_ALREADY_STOPPED`

**RPC-heavy:** none — flow/activeflow operations all hit flow-manager only. Single-hop. Per §6.1 "single-manager endpoints can omit 503", these endpoints don't add 503.

## Forward-dependency note from PR 2 Round 1

PR 2 added `"already"` and `"deleted"` substring patterns to the translator specifically anticipating activeflow state-transition errors. So `fmt.Errorf("activeflow already stopped")` and `fmt.Errorf("deleted activeflow")` already route to `FAILED_PRECONDITION/STATE_INVALID` (409) today.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/flows.go`

6 handlers, 18 sites. Standard mappings (auth → AUTHENTICATION_REQUIRED, BindJSON → INVALID_JSON_BODY, UUID parse → INVALID_ID, servicehandler → abortWithServiceError). Sampled error-path tests.

### Task 3: Migrate `server/activeflows.go`

5 handlers, 11 sites. Same pattern. Add a state-transition test for `PostActiveflowsIdStop` if mockable.

### Task 4: OpenAPI path wiring

Wire all `/flows*` and `/activeflows*` paths per §6.1 baseline. Specifically:

- `GET /flows`, `GET /activeflows` → 401, 500
- `POST /flows`, `POST /activeflows` → 400, 401, 500
- `GET /flows/{id}`, `GET /activeflows/{id}` → 400, 401, 403, 404, 500
- `PUT /flows/{id}`, `DELETE /flows/{id}`, `DELETE /activeflows/{id}`, `POST /flows/{id}/direct_hash_regenerate` → 400, 401, 403, 404, 500
- `POST /activeflows/{id}/stop` → 400, 401, 403, 404, **409**, 500

Regenerate `gens/openapi_server/gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 5: RST catalog — add flow-manager section

Add a new `flow-manager` section to `restful_api_errors.rst` between `call-manager` and the existing placeholders. Reasons:

- `FLOW_NOT_FOUND` (404)
- `ACTIVEFLOW_NOT_FOUND` (404)
- `FLOW_STATE_INVALID` (409) — generic state restriction
- `ACTIVEFLOW_ALREADY_STOPPED` (409) — specific state-transition reason

Match the same disclaimer style introduced in PR 2: enumerate which reasons are reachable today via translator fallback (the `_NOT_FOUND` ones via "not found", state ones via "already" / "not active") vs which require typed-error migration.

Rebuild Sphinx HTML.

### Task 6: Full verification

Standard 5-step workflow for both `bin-openapi-manager` and `bin-api-manager`.

### Task 7: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr3-flows-activeflows` on every commit.
- No AI attribution.
- `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` helper.
- Preserve existing `log.Errorf/Infof` site-specific lines.

## Success criteria

- All 7 tasks committed
- `go test -race ./server/...` green
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- New `flow-manager` catalog section
- All 29 sites converted
