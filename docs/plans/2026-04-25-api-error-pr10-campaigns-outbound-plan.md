# PR 10 — Campaigns & outbound dialing handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the campaigns, campaigncalls, outdials, and outplans handlers to the canonical error envelope. 31 handlers, 101 sites across 4 files — largest single-PR site count in the rollout. **No 402, no 409, no 503** modifiers — verified upstream behavior.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 9 (`NOJIRA-api-error-pr9-identity-group`, merged `9e15691c5`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr10-campaigns-outbound`
**Branch:** `NOJIRA-api-error-pr10-campaigns-outbound` (branched from `origin/main` at `9e15691c5`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/campaigns.go` | ~ | 38 | 11 |
| `server/campaigncalls.go` | ~ | 8 | 3 |
| `server/outdials.go` | ~ | 36 | 11 |
| `server/outplans.go` | ~ | 19 | 6 |

**Total: 101 sites across 31 handlers in 4 files.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetCampaigns`, `GetCampaigncalls`, `GetOutdials`, `GetOutplans`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetCampaignsId`, `GetCampaigncallsId`, `GetOutdialsId`, `GetOutplansId`
- `GetCampaignsIdCampaigncalls`, `GetOutdialsIdTargets`, `GetOutdialsIdTargetsTargetId`

**Write (no resource ID) → 400, 401, 500:**
- `PostCampaigns`, `PostOutdials`, `PostOutplans`

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `DeleteCampaigncallsId`
- `DeleteCampaignsId`, `PutCampaignsId`, `PutCampaignsIdStatus`, `PutCampaignsIdServiceLevel`, `PutCampaignsIdActions`, `PutCampaignsIdResourceInfo`, `PutCampaignsIdNextCampaignId`
- `DeleteOutdialsId`, `PutOutdialsId`, `PutOutdialsIdCampaignId`, `PutOutdialsIdData`, `PostOutdialsIdTargets`, `DeleteOutdialsIdTargetsTargetId`
- `DeleteOutplansId`, `PutOutplansId`, `PutOutplansIdDialInfo`

## Modifier reachability checks

### 402 PaymentRequired — NOT WIRED

`grep -rE "insufficient|enough.balance|has not.*balance" bin-campaign-manager/pkg/ bin-outdial-manager/pkg/ 2>&1 | head -10` returns zero matches. Campaign / outdial creation doesn't deduct balance directly — the per-call charges happen downstream when individual calls are dialed. No 402 declarations.

### 409 Conflict — NOT WIRED

Campaign state-transition operations are idempotent at the manager layer:
- `bin-campaign-manager/pkg/campaignhandler/status_stop.go:69` — comment "Status is already stop or stopping" returns success without error.
- Same idempotent pattern across other state mutations.

So `PutCampaignsIdStatus` and similar state mutations don't surface 409 today. Do not declare 409.

### 503 ServiceUnavailable — NOT WIRED

Campaign operations route to single managers (campaign-manager or outdial-manager). No fan-out justifying 503.

## Forward-dependency notes

- After PR 10, remaining unmigrated `bin-api-manager/server/` files cluster into:
  - **Routing/control (~110 sites):** `calls.go` (partial), `conferencecalls.go`, `conferences.go`, `queuecalls.go`, `queues.go`.
  - **Contacts/extensions (~71 sites):** `contacts.go`, `extensions.go`.
  - **Service-agent and small surfaces (~80+ sites):** `service_agents_*.go` (multiple), `storage_files.go`, `aggregated_events.go`, `timelines*.go`, `ws.go`.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/campaigns.go` (11 handlers, 38 sites)

Path-UUID hardening for all by-ID handlers (string IDs).

Sample tests in `server/campaigns_test.go` (file already exists — append):
- `Test_campaignsPost_MissingAuthIdentity`
- `Test_campaignsPost_InvalidJSONBody`
- `Test_campaignsIDPut_InvalidID`
- `Test_campaignsIDStatusPut_InvalidJSONBody`

### Task 3: Migrate `server/campaigncalls.go` (3 handlers, 8 sites)

Path-UUID hardening for `GetCampaigncallsId`, `DeleteCampaigncallsId`.

Sample test:
- `Test_campaigncallsIDDelete_InvalidID`

### Task 4: Migrate `server/outdials.go` (11 handlers, 36 sites)

Path-UUID hardening for all by-ID handlers including the dual-ID `GetOutdialsIdTargetsTargetId` and `DeleteOutdialsIdTargetsTargetId` (both `id` and `targetId` need validation).

Sample test:
- `Test_outdialsIDTargetsTargetIDDelete_InvalidTargetID`

### Task 5: Migrate `server/outplans.go` (6 handlers, 19 sites)

Path-UUID hardening for `GetOutplansId`, `DeleteOutplansId`, `PutOutplansId`, `PutOutplansIdDialInfo`.

### Task 6: OpenAPI path wiring

Files under `bin-openapi-manager/openapi/paths/`:

- `campaigns/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `campaigns/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `campaigns/id_status.yaml`, `id_service_level.yaml`, `id_actions.yaml`, `id_resource_info.yaml`, `id_next_campaign_id.yaml` — PUT (400, 401, 403, 404, 500)
- `campaigns/id_campaigncalls.yaml` — GET (400, 401, 403, 404, 500)
- `campaigncalls/main.yaml` — GET (401, 500)
- `campaigncalls/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `outdials/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `outdials/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `outdials/id_campaign_id.yaml`, `id_data.yaml` — PUT (400, 401, 403, 404, 500)
- `outdials/id_targets.yaml` — GET, POST (400, 401, 403, 404, 500)
- `outdials/id_targets_target_id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `outplans/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `outplans/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `outplans/id_dial_info.yaml` — PUT (400, 401, 403, 404, 500)

Verify exact filenames with `ls`. **No 402, no 409 anywhere.**

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 7: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Add `campaign-manager` section** (new):
   - `CAMPAIGN_NOT_FOUND` (404)
   - `CAMPAIGNCALL_NOT_FOUND` (404)
   - Note: campaign state-transition operations (`PUT /campaigns/{id}/status`) are idempotent today; future state-transition typing may add 409.

2. **Add `outdial-manager` section** (new):
   - `OUTDIAL_NOT_FOUND` (404)
   - `OUTDIAL_TARGET_NOT_FOUND` (404)
   - `OUTPLAN_NOT_FOUND` (404)
   - (If outplans live under a different manager, fold accordingly.)

3. **Update "Other Domains" deferred list at the bottom**:
   - Remove `campaign-manager` (now populated).

Match disclaimer style from PR 4-9.

Rebuild Sphinx HTML.

### Task 8: Full verification

Standard 5-step workflow.

### Task 9: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr10-campaigns-outbound` on every commit.
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
- New campaign-manager and outdial-manager catalog sections
- All 101 sites converted
- Zero 402 / 409 declarations added
