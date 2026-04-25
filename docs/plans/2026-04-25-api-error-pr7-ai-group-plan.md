# PR 7 — AI group handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the AI configuration (`ais`), AI calls (`aicalls`), AI summaries (`aisummaries`), and RAG (`rags`) handlers to the canonical error envelope. 21 handlers, 66 sites across 4 files. **No 402 modifier** — verified that `bin-ai-manager` has no balance pre-check today (lesson from PR 6 Round 1).

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 6 (`NOJIRA-api-error-pr6-messages-emails`, merged `92082e002`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr7-ai-group`
**Branch:** `NOJIRA-api-error-pr7-ai-group` (branched from `origin/main` at `92082e002`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/ais.go` | 304 | 18 | 6 |
| `server/aicalls.go` | 158 | 11 | 4 |
| `server/aisummaries.go` | 160 | 11 | 4 |
| `server/rags.go` | 325 | 26 | 7 |

**Total: 66 sites across 21 handlers in 4 files.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetAis`, `GetAicalls`, `GetAisummaries`, `GetRags`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetAisId`, `GetAicallsId`, `GetAisummariesId`, `GetRagsId`

**Write (no resource ID) → 400, 401, 500:**
- `PostAis`, `PostAicalls`, `PostAisummaries`, `PostRags` — none are billing-sensitive at the api-manager layer (see "402 reachability check" below).

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `PutAisId`, `DeleteAisId`, `PostAisIdDirectHashRegenerate`
- `DeleteAicallsId`
- `DeleteAisummariesId`
- `PutRagsId`, `PostRagsIdSources`, `DeleteRagsId`, `DeleteRagsIdSourcesSourceId`

**State-transition (+409):** none.

**RPC-heavy (+503):** none — each operation routes to a single manager (ai-manager).

**Billing-sensitive (+402):** **NONE** — confirmed by `grep -rE "insufficient|enough.balance|has not.*balance" bin-ai-manager/pkg/` returning zero matches. The ai-manager does not currently emit balance pre-check errors. While voice AI calls and LLM token usage are conceptually chargeable (downstream metering), the api-manager-facing layer doesn't surface a balance error today. Per PR 6 Round 1 lesson: do NOT declare 402 on endpoints whose upstream managers don't actually emit `"insufficient"`-style errors. Defer 402 wiring on `POST /aicalls` and `POST /aisummaries` to a future PR that also adds the balance pre-check in ai-manager.

## Forward-dependency notes

- **Voice group (next PR candidate):** `speakings.go`, `transcribes.go`, `transcripts.go` — TTS and STT surfaces. Speaking and transcribe are billing-sensitive (charge per character / per second). Need to verify upstream wording before wiring 402.
- **AI catalog completion:** PR 6 added an `ai-manager` catalog section scoped to `AIMESSAGE_NOT_FOUND`. PR 7 extends it with `AI_NOT_FOUND`, `AICALL_NOT_FOUND`, `AISUMMARY_NOT_FOUND`, and adds `rag-manager` section.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/ais.go` (6 handlers, 18 sites)

Path-UUID hardening for `GetAisId`, `PutAisId`, `DeleteAisId` (string IDs). `PostAisIdDirectHashRegenerate` already takes `openapi_types.UUID`.

Sample tests in `server/ais_test.go` (file already exists — append):
- `Test_aisPost_MissingAuthIdentity`
- `Test_aisPost_InvalidJSONBody`
- `Test_aisIDPut_InvalidID`

### Task 3: Migrate `server/aicalls.go` (4 handlers, 11 sites)

Path-UUID hardening for `GetAicallsId`, `DeleteAicallsId`.

Sample test in `server/aicalls_test.go`:
- `Test_aicallsIDDelete_InvalidID`

### Task 4: Migrate `server/aisummaries.go` (4 handlers, 11 sites)

Path-UUID hardening for `GetAisummariesId`, `DeleteAisummariesId`.

### Task 5: Migrate `server/rags.go` (7 handlers, 26 sites)

All by-ID handlers use `openapi_types.UUID` (router-validated). No string-to-UUID hardening needed in the handler bodies, but if the handler converts via `uuid.FromString(id.String())` and checks errors that can never fail in practice, simplify.

`DeleteRagsIdSourcesSourceId` takes two UUIDs — both already `openapi_types.UUID`.

Sample test in `server/rags_test.go`:
- `Test_ragsIDPut_InvalidJSONBody`

### Task 6: OpenAPI path wiring

Files to edit / create under `bin-openapi-manager/openapi/paths/`:

- `ais/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `ais/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `ais/id_direct_hash_regenerate.yaml` — POST (400, 401, 403, 404, 500)
- `aicalls/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `aicalls/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `aisummaries/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `aisummaries/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `rags/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `rags/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `rags/id_sources.yaml` — POST (400, 401, 403, 404, 500)
- `rags/id_sources_source_id.yaml` — DELETE (400, 401, 403, 404, 500)

Match existing yaml structure conventions (`ls` to verify exact filenames).

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 7: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Extend `ai-manager` section** (added in PR 6 with `AIMESSAGE_NOT_FOUND`):
   - Add `AI_NOT_FOUND` (404) — for `/ais/{id}`.
   - Add `AICALL_NOT_FOUND` (404) — for `/aicalls/{id}`.
   - Add `AISUMMARY_NOT_FOUND` (404) — for `/aisummaries/{id}`.
   - Update the deferral note: now only `aicall billing pre-check` and `aisummary billing pre-check` are pending in ai-manager (the 402 deferred list).

2. **Add `rag-manager` section** (new):
   - `RAG_NOT_FOUND` (404).

Match disclaimer style (translator-reachable today via `"not found"` substring; typed-error future).

Rebuild Sphinx HTML.

### Task 8: Full verification

Standard 5-step workflow for both `bin-openapi-manager` and `bin-api-manager`.

### Task 9: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr7-ai-group` on every commit.
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

- All 9 tasks committed
- `go test -race ./...` green for both `bin-api-manager` and `bin-openapi-manager`
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- ai-manager catalog section extended with `AI_NOT_FOUND`, `AICALL_NOT_FOUND`, `AISUMMARY_NOT_FOUND`
- New rag-manager catalog section
- All 66 sites converted
- Zero 402 declarations added (consistent with PR 6 Round 1 lesson — no over-claiming)
