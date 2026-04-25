# PR 2 — Calls group handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the calls/groupcalls/recordings/recordingfiles/transfers handlers (the heart of the platform) to the canonical error envelope. Largest single migration in the rollout — 92 error sites across 27 handlers in 5 files. Also extends the §6.1 convention with three additions surfaced during PR 1b review: billing-sensitive 402 in practice, state-transition 409 rule, RPC-heavy 503 add-on.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 1b (`NOJIRA-api-error-pr1b-admin-agent-ui`, merged `6f1485a31`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr2-calls-group`
**Branch:** `NOJIRA-api-error-pr2-calls-group` (branched from `origin/main` at `6f1485a31`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/calls.go` | 689 | 64 | 17 |
| `server/groupcalls.go` | 192 | 14 | 5 |
| `server/recordings.go` | 127 | 8 | 3 |
| `server/recordingfiles.go` | 41 | 3 | 1 |
| `server/transfers.go` | 59 | 3 | 1 |

**Total: 92 sites across 27 handlers.**

### Handler classification (for OpenAPI wiring per §6.1)

**Read (GET, no path param):** `GetCalls`, `GetGroupcalls`, `GetRecordings` — baseline `401, 500`.

**Read (GET with resource ID):** `GetCallsId`, `GetGroupcallsId`, `GetRecordingsId`, `GetRecordingfilesId` — baseline `400, 401, 403, 404, 500`. Plus 503 (RPC-heavy).

**Write — billing-sensitive (deducts credits):** `PostCalls`, `PostGroupcalls` — baseline `400, 401, 500` + `402` (insufficient balance) + `503` (RPC-heavy).

**Write (POST/PUT/PATCH/DELETE with resource ID, no state restriction):** `DeleteRecordingsId` — baseline `400, 401, 403, 404, 500` + `503` (RPC-heavy). No 409: recording deletion has no state machine to violate (unlike call hangup which can only happen once on a non-terminated call).

**Write — state-transition (resource state machine):** `PostCallsIdHangup`, `PostCallsIdTalk`, `PostCallsIdHold`, `DeleteCallsIdHold`, `PostCallsIdMute`, `DeleteCallsIdMute`, `PostCallsIdMoh`, `DeleteCallsIdMoh`, `PostCallsIdSilence`, `DeleteCallsIdSilence`, `PostCallsIdRecordingStart`, `PostCallsIdRecordingStop`, `PostGroupcallsIdHangup`, `PostTransfers`, `DeleteCallsId`, `DeleteGroupcallsId` — baseline `400, 401, 403, 404, 500` + `409` (wrong state) + `503` (RPC-heavy).

**WebSocket upgrade (counter-example):** `GetCallsIdMediaStream` — handshake errors get baseline; post-upgrade errors use WS close codes. Per §6.1 counter-example.

## Design-doc amendments to land first (Task 1)

PR 1b's review surfaced three §6.1 gaps that PR 2 will hit immediately. Pre-codify as part of this PR's first commit so the migration code references concrete rules.

Changes to `docs/plans/2026-04-24-api-error-response-codes-design.md` §6.1:

1. **State-transition row.** Add to the convention table:

   | State-transition (operation invalid for current state) | baseline + `409` |

   With explicit examples in the rationale: "POST /calls/{id}/hangup on an already-hung-up call → 409 FAILED_PRECONDITION; recording start on an already-recording call → 409; transfer from a non-progressing call → 409. Use the dedicated reason codes `CALL_ALREADY_HANGUP`, `RECORDING_ALREADY_ACTIVE`, `CALL_STATE_INVALID`, etc."

2. **RPC-heavy 503 row.** Add:

   | RPC-heavy (fan-out to ≥2 managers) | baseline + `503` |

   Rationale: "PostCalls fans out to call-manager + flow-manager + number-manager + billing-manager. Any can fail mid-fan-out, returning UNAVAILABLE / SERVICE_UNAVAILABLE. Single-manager endpoints (e.g., GetRecordings) can omit 503 since the failure mode is more typically INTERNAL (single hop)."

3. **Billing-sensitive concrete examples.** Update the existing billing-sensitive row's rationale to list `POST /calls`, `POST /groupcalls`, `POST /messages`, `POST /emails`, `POST /numbers` explicitly. Today's text says "any endpoint whose success path deducts credits" — add the enumerated list so PR 2+ authors know which paths to flag.

4. **§13 audit trail entry.** Append "PR 2 calls-group refinements" subsection.

## Tasks

### Task 1: Design-doc amendments (state-transition 409, RPC-heavy 503, billing examples)

Apply the 4 changes above. No code touched — docs only. Locks the convention BEFORE the migration code references it.

Commit: `NOJIRA-api-error-pr2-calls-group: amend §6.1 convention with state-transition, RPC-heavy, and billing-sensitive specifics`.

### Task 2: Migrate `server/calls.go` (64 sites, 17 handlers)

Biggest single file. Apply standard mappings from PR 1/1b. Specific reasons to use:

- Auth identity → `UNAUTHENTICATED / AUTHENTICATION_REQUIRED`
- BindJSON → `INVALID_ARGUMENT / INVALID_JSON_BODY`
- UUID parse → `INVALID_ARGUMENT / INVALID_ID`
- Servicehandler → `abortWithServiceError` (translator classifies; substring fallback now covers `"direct access"`, `"does not belong"`, `"no permission"`, `"not found"`, `"unavailable"`)

Add error-path tests sampling the patterns. Required:
- 1 `_MissingAuthIdentity` per HTTP method (GET, POST, DELETE)
- 1 `_InvalidJSONBody` for `PostCalls` (body parse)
- 1 `_InvalidID` for `GetCallsId` or `DeleteCallsId` (path UUID)
- 1 `_ServiceError` for `GetCallsId` (translator routes "call not found" → NOT_FOUND)
- 1 `_BillingError` for `PostCalls` if mockable — passes a typed `cerrors.PaymentRequired(...)` and asserts the response

### Task 3: Migrate `server/groupcalls.go` (14 sites, 5 handlers) and `server/transfers.go` (3 sites, 1 handler)

Group together since they share the state-transition pattern. Tests sampled.

### Task 4: Migrate `server/recordings.go` (8 sites, 3 handlers) and `server/recordingfiles.go` (3 sites, 1 handler)

Smaller files. Tests sampled. `GetRecordingfilesId` returns a download URL — straightforward read pattern.

### Task 5: OpenAPI path wiring

Wire all calls/groupcalls/recordings/recordingfiles/transfers paths per §6.1:

- `POST /calls` → 400, 401, 402, 500, 503 (write-no-ID + billing + RPC-heavy)
- `GET /calls` → 401, 500 (read-no-ID; 503 NOT added — list endpoints aren't RPC-heavy in the same way; document as deviation if needed)
- `GET /calls/{id}` → 400, 401, 403, 404, 500, 503 (read-with-ID + RPC-heavy)
- `DELETE /calls/{id}` → 400, 401, 403, 404, 409, 500, 503 (write-with-ID + state-transition + RPC-heavy)
- `POST /calls/{id}/hangup`, `/talk`, `/hold`, `/mute`, `/moh`, `/silence`, `/recording-start`, `/recording-stop` → 400, 401, 403, 404, 409, 500, 503 (write-with-ID + state-transition + RPC-heavy)
- `DELETE /calls/{id}/hold`, `/mute`, `/moh`, `/silence` → same as their POST counterparts
- `GET /calls/{id}/media-stream` → counter-example (WebSocket upgrade); declare 401, 500 for handshake errors only
- `POST /groupcalls` → 400, 401, 402, 500, 503 (billing + RPC-heavy)
- `GET /groupcalls`, `GET /groupcalls/{id}`, `DELETE /groupcalls/{id}`, `POST /groupcalls/{id}/hangup` → standard per class
- `GET /recordings`, `GET /recordings/{id}` → standard per class
- `DELETE /recordings/{id}` → 400, 401, 403, 404, 500, 503 (write-with-ID + RPC-heavy; no 409 — no state machine on recording deletion)
- `GET /recordingfiles/{id}` → 400, 401, 403, 404, 500, 503
- `POST /transfers` → 400, 401, 403, 404, 409, 500, 503 (write-with-ID — operates on source call; state-transition; RPC-heavy)

Regenerate `gens/openapi_server/gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 6: RST catalog updates

Append new reason codes to `restful_api_errors.rst`. New reasons emitted by PR 2:

- `CALL_NOT_FOUND` (404) — call ID doesn't exist or belongs to another customer
- `CALL_ALREADY_HANGUP` (409) — operation invalid because call already ended
- `CALL_STATE_INVALID` (409) — operation invalid for current call state (generic)
- `RECORDING_NOT_FOUND` (404)
- `RECORDING_ALREADY_ACTIVE` (409) — start on already-recording call
- `RECORDING_NOT_ACTIVE` (409) — stop on non-recording call
- `INSUFFICIENT_BALANCE` (402) — billing balance below required
- `GROUPCALL_NOT_FOUND` (404)

If servicehandler/calls.go currently uses generic strings, the substring fallback will route them; new reason codes will be added to the catalog as PR 2 introduces typed constructors. For PR 2 we add the catalog entries up front so they're documented; the migration to typed errors in servicehandler is a future PR's scope (deferred per the design doc).

Add new "call-manager" section to the catalog (currently only "api-manager" exists). Populate with the reasons above.

Rebuild Sphinx HTML.

### Task 7: Full verification

Standard 5-step workflow for both `bin-openapi-manager` and `bin-api-manager`. Same as prior PRs.

### Task 8: Push + open PR

Conflict check, push, open PR with body linking back to design doc + plan.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr2-calls-group` on every commit.
- No AI attribution.
- Preserve existing `log.Errorf/Infof` lines.
- Use `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` helper from `customer_test.go` (same package).

## Success criteria

- All 8 tasks committed
- `go test -race ./...` green in both services
- `golangci-lint` 0 issues in both
- Design doc §6.1 has state-transition, RPC-heavy rows; §13 has PR 2 audit subsection
- New reason codes catalogued in `restful_api_errors.rst`
- All 27 handlers migrated, all 92 sites converted

## Out of scope (deferred to future PRs)

- Migrating `pkg/servicehandler/calls.go` etc. to use typed sentinels instead of `fmt.Errorf` — translator substring fallback handles them; principled migration deferred.
- Retrofitting `/me`, `/customer*`, `/customers*`, `/service-agents/*` to add 503 baseline — coordinated with §10.5 shrink-fallback PR.
- The 405 METHOD_NOT_ALLOWED canonical status — separate schema-bump PR.
