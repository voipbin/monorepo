# PR 11 — Conferences & queues handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the conferences, conferencecalls, queues, and queuecalls handlers to the canonical error envelope. 27 handlers, 84 sites across 4 files. **No 402, no 409, no 503** modifiers — verified upstream behavior.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 10 (`NOJIRA-api-error-pr10-campaigns-outbound`, merged `50d4338f9`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr11-conferences-queues`
**Branch:** `NOJIRA-api-error-pr11-conferences-queues` (branched from `origin/main` at `50d4338f9`)

## Scope

| File | Sites | Handlers |
|---|---|---|
| `server/conferences.go` | 36 | 11 |
| `server/conferencecalls.go` | 8 | 3 |
| `server/queues.go` | 26 | 8 |
| `server/queuecalls.go` | 14 | 5 |

**Total: 84 sites across 27 handlers in 4 files.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetConferences`, `GetConferencecalls`, `GetQueues`, `GetQueuecalls`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetConferencesId`, `GetConferencecallsId`, `GetQueuesId`, `GetQueuecallsId`
- `GetConferencesIdMediaStream`

**Write (no resource ID) → 400, 401, 500:**
- `PostConferences`, `PostQueues`

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `DeleteConferencesId`, `PutConferencesId`
- `PostConferencesIdRecordingStart`, `PostConferencesIdRecordingStop`
- `PostConferencesIdTranscribeStart`, `PostConferencesIdTranscribeStop`
- `PostConferencesIdDirectHashRegenerate`
- `DeleteConferencecallsId`
- `DeleteQueuesId`, `PutQueuesId`, `PutQueuesIdTagIds`, `PutQueuesIdRoutingMethod`
- `PostQueuesIdDirectHashRegenerate`
- `DeleteQueuecallsId`, `PostQueuecallsIdKick`, `PostQueuecallsReferenceIdIdKick`

## Modifier reachability checks

### 402 PaymentRequired — NOT WIRED

`grep -rE "insufficient|enough.balance|has not.*balance" bin-conference-manager/pkg/ bin-queue-manager/pkg/ 2>&1` returns zero matches. Conference creation and queue ops don't surface balance errors at the manager layer (downstream call charging is handled in call-manager). No 402 declarations.

### 409 Conflict — NOT WIRED

State-mutation operations are idempotent at the manager layer:
- `bin-conference-manager/pkg/conferencehandler/terminate.go:27` — comment "if the conference is already terminated or stopping, just return at here" (no error).
- `bin-queue-manager/pkg/queuecallhandler/kick.go:69` — comment "already ended" (no error).

Recording and transcribe Start/Stop on conferences also follow the manager's idempotent pattern. So `PostConferencesIdRecordingStop` and `PostConferencesIdTranscribeStop` don't surface 409 today. Do not declare 409.

### 503 ServiceUnavailable — NOT WIRED

Conference and queue operations route to single managers. No fan-out justifying 503.

## Forward-dependency notes

- After PR 11, remaining unmigrated `bin-api-manager/server/` files cluster into:
  - **calls.go** (partial PR 2 — 10 stragglers in 17 handlers) — likely the most billing-sensitive remaining file (`POST /calls` for outbound dialing).
  - **Contacts/extensions (~71 sites):** `contacts.go`, `extensions.go`.
  - **Service-agent surfaces (~80+ sites):** `service_agents_*.go` (multiple), `service_agents_talk.go` notably has 50 sites.
  - **Small surfaces (~30 sites):** `storage_files.go`, `aggregated_events.go`, `timelines*.go`, `ws.go`.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/conferences.go` (11 handlers, 36 sites)

Path-UUID hardening for all by-ID handlers using string IDs. `PostConferencesIdDirectHashRegenerate` already uses `openapi_types.UUID`.

Sample tests in `server/conferences_test.go` (file already exists — append):
- `Test_conferencesPost_MissingAuthIdentity`
- `Test_conferencesPost_InvalidJSONBody`
- `Test_conferencesIDPut_InvalidID`
- `Test_conferencesIDRecordingStartPost_InvalidID`

### Task 3: Migrate `server/conferencecalls.go` (3 handlers, 8 sites)

Path-UUID hardening on `GetConferencecallsId`, `DeleteConferencecallsId`.

Sample test:
- `Test_conferencecallsIDDelete_InvalidID`

### Task 4: Migrate `server/queues.go` (8 handlers, 26 sites)

Path-UUID hardening for all by-ID handlers (string IDs). `PostQueuesIdDirectHashRegenerate` already uses `openapi_types.UUID`.

Sample tests:
- `Test_queuesPost_MissingAuthIdentity`
- `Test_queuesIDPut_InvalidID`

### Task 5: Migrate `server/queuecalls.go` (5 handlers, 14 sites)

Path-UUID hardening for `GetQueuecallsId`, `DeleteQueuecallsId`, `PostQueuecallsIdKick`. The `PostQueuecallsReferenceIdIdKick` handler uses an inner string `id` (which is the reference ID per the route name) — apply hardening with appropriate message.

Sample test:
- `Test_queuecallsIDKickPost_InvalidID`

### Task 6: OpenAPI path wiring

Files under `bin-openapi-manager/openapi/paths/`:

- `conferences/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `conferences/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `conferences/id_recording_start.yaml`, `id_recording_stop.yaml`, `id_transcribe_start.yaml`, `id_transcribe_stop.yaml` — POST (400, 401, 403, 404, 500)
- `conferences/id_media_stream.yaml` — GET (400, 401, 403, 404, 500)
- `conferences/id_direct_hash_regenerate.yaml` — POST (400, 401, 403, 404, 500)
- `conferencecalls/main.yaml` — GET (401, 500)
- `conferencecalls/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `queues/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `queues/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `queues/id_tag_ids.yaml`, `id_routing_method.yaml` — PUT (400, 401, 403, 404, 500)
- `queues/id_direct_hash_regenerate.yaml` — POST (400, 401, 403, 404, 500)
- `queuecalls/main.yaml` — GET (401, 500)
- `queuecalls/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `queuecalls/id_kick.yaml`, `reference_id_id_kick.yaml` — POST (400, 401, 403, 404, 500)

Verify exact filenames with `ls`. **No 402, no 409 anywhere.**

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 7: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Add `conference-manager` section** (new):
   - `CONFERENCE_NOT_FOUND` (404)
   - `CONFERENCECALL_NOT_FOUND` (404)
   - Note: conference state-mutation operations (`POST /conferences/{id}/recording_stop`, etc.) are idempotent at the manager layer; future state-transition typing may add 409.

2. **Add `queue-manager` section** (new):
   - `QUEUE_NOT_FOUND` (404)
   - `QUEUECALL_NOT_FOUND` (404)
   - Similar idempotent note for `POST /queuecalls/{id}/kick`.

3. **Update "Other Domains" deferred list at the bottom**:
   - Remove `conference-manager` and `queue-manager` (now populated).

Match disclaimer style from PR 4-10.

Rebuild Sphinx HTML.

### Task 8: Full verification

Standard 5-step workflow.

### Task 9: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr11-conferences-queues` on every commit.
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
- `go test -race ./...` green
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- New conference-manager and queue-manager catalog sections
- All 84 sites converted
- Zero 402 / 409 declarations added
