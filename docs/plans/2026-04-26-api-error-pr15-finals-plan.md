# PR 15 — Final migration & cleanup (rollout completion)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the last remaining error sites and clean up `c.AbortWithStatus(200)` success-path code smell. Final PR in the canonical-error-envelope rollout.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 14 (`NOJIRA-api-error-pr14-service-agents-talk`, merged `af32906d7`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr15-finals`
**Branch:** `NOJIRA-api-error-pr15-finals` (branched from `origin/main` at `af32906d7`)

## Scope

| File | Real error sites | Cleanup sites | Handlers touched |
|---|---|---|---|
| `server/storage_files.go` | 15 | 0 | 5 |
| `server/ws.go` | 1 | 0 | 1 |
| `server/calls.go` | 0 | 10 (`c.AbortWithStatus(200)` → `c.Status(200)`) | 10 |
| `server/conferences.go` | 0 | 1 (`c.AbortWithStatus(200)` → `c.Status(200)`) | 1 |

**Total: 16 real error sites to migrate, plus 11 success-path code-smell cleanups.**

The other files reported by the grep — `aggregated_events.go`, `timelines.go`, `timelines_sip.go` — only have `c.JSON(http.StatusOK, res)` SUCCESS responses, which are correct as-is (they're not error sites; the grep pattern matched any `c.JSON(http.Status...)` including success paths).

### Real error site classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetStorageFiles`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetStorageFilesId`, `GetStorageFilesIdFile`

**WebSocket (read-shaped) → 401, 500:**
- `GetWs` — pre-handshake auth check is the single 400 abort site (legacy bug — should be 401 for missing auth, not 400). Migrate to canonical 401 / `AUTHENTICATION_REQUIRED`.

**Write (no resource ID) → 400, 401, 500:**
- `PostStorageFiles`

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `DeleteStorageFilesId`

### Cleanup site classification

`c.AbortWithStatus(200)` is a code smell — `AbortWithStatus` should be reserved for early termination on error paths. For success-path 200-no-body responses, the idiomatic Gin pattern is `c.Status(http.StatusOK)` (or just letting the handler return without setting status). Migration is functionally equivalent but makes the code-base consistent post-rollout.

Touched handlers:
- `calls.go`: `PostCallsIdHangup`, `PostCallsIdTalk`, `PostCallsIdHold`, `DeleteCallsIdHold`, `PostCallsIdMute`, `DeleteCallsIdMute`, `PostCallsIdMoh`, `DeleteCallsIdMoh`, `PostCallsIdSilence`, `DeleteCallsIdSilence` (10 sites)
- `conferences.go`: One success-path site (likely `PostConferencesIdRecordingStop` or similar)

## Modifier reachability checks

### 402 PaymentRequired — NOT WIRED

Storage operations don't deduct balance directly (storage cost is metered out-of-band). No 402.

### 409 Conflict — NOT WIRED

No state-transition contracts.

### 503 ServiceUnavailable — NOT WIRED

Single-manager hops.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/storage_files.go` (5 handlers, 15 error sites)

Path-UUID hardening on `GetStorageFilesId`, `DeleteStorageFilesId` (string IDs). `GetStorageFilesIdFile` already uses `openapi_types.UUID`.

Sample tests in `storage_files_test.go` (file already exists — append):
- `Test_storageFilesPost_MissingAuthIdentity`
- `Test_storageFilesIDDelete_InvalidID`

### Task 3: Migrate `server/ws.go` (1 handler, 1 error site)

`GetWs` pre-handshake — currently emits `c.AbortWithStatus(400)` for missing auth. Migrate to canonical `401` / `AUTHENTICATION_REQUIRED` (the 400 was a legacy bug).

### Task 4: Cleanup `c.AbortWithStatus(200)` in `server/calls.go` and `server/conferences.go`

Replace `c.AbortWithStatus(200)` with `c.Status(http.StatusOK)` on the 11 affected sites. Preserve all surrounding logic.

### Task 5: OpenAPI path wiring

Files under `bin-openapi-manager/openapi/paths/`:

- `storage_files/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `storage_files/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `storage_files/id_file.yaml` — GET (400, 401, 403, 404, 500)
- `ws.yaml` — GET (401, 500) — confirm this exists or is auto-generated.

Verify exact filenames with `ls`. **No 402, no 409 anywhere.**

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 6: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Storage-manager catalog section** was extended in PR 13 with `STORAGE_FILE_NOT_FOUND` (covering `service_agents/files`). PR 15 just adds the public `/storage_files` endpoints to that same description.

2. **Final rollout completion note** — add a closing paragraph or update the "Other Domains" deferred list at the bottom of the file. The deferred list should now be empty (or contain only intentionally-deferred typed-error future work). Note that the canonical error envelope is now uniformly applied across all `bin-api-manager/server/` files.

Match disclaimer style from PR 4-14.

Rebuild Sphinx HTML.

### Task 7: Full verification

Standard 5-step workflow.

### Task 8: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr15-finals` on every commit.
- No AI attribution.
- `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` and `assertErrorResponse` helpers.
- Path-UUID hardening pattern: `uuid.FromStringOrNil(id) == uuid.Nil`.
- `getAuthIdentity(c)` (NOT `commonmiddleware.AuthIdentityGet`).
- `abortWithError` takes `*cerrors.VoipbinError`.
- Standard message strings: "The request body is not valid JSON.", "The provided id is not a valid UUID.", "Authentication is required."
- Preserve existing `log.Errorf/Infof` site-specific lines.

## Success criteria

- All 8 tasks committed
- `go test -race ./...` green
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- Zero `c.AbortWithStatus(...)` or `c.JSON(http.Status..., gin.H{...})` ERROR sites remaining anywhere in `bin-api-manager/server/*.go`
- All 16 real error sites migrated, all 11 success-path sites cleaned up
- Rollout completion note in RST catalog
- Zero 402 / 409 declarations added
