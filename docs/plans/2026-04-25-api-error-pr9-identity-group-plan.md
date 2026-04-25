# PR 9 — Identity group handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the agents, teams, tags, and accesskeys handlers to the canonical error envelope. 27 handlers, 85 sites across 4 files. **No 402, no 409, no 503** modifiers — these are identity/access-management resources with no charging, no state-transition contracts, and no fan-out RPC patterns.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 8 (`NOJIRA-api-error-pr8-voice-group`, merged `43158fbee`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr9-identity-group`
**Branch:** `NOJIRA-api-error-pr9-identity-group` (branched from `origin/main` at `43158fbee`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/agents.go` | 516 | 39 | 11 |
| `server/teams.go` | 295 | 18 | 6 |
| `server/tags.go` | 197 | 15 | 5 |
| `server/accesskeys.go` | 212 | 13 | 5 |

**Total: 85 sites across 27 handlers in 4 files.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetAgents`, `GetTeams`, `GetTags`, `GetAccesskeys`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetAgentsId`, `GetTeamsId`, `GetTagsId`, `GetAccesskeysId`

**Write (no resource ID) → 400, 401, 500:**
- `PostAgents`, `PostTeams`, `PostTags`, `PostAccesskeys`

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `DeleteAgentsId`, `PutAgentsId`, `PutAgentsIdAddresses`, `PutAgentsIdTagIds`, `PutAgentsIdStatus`, `PutAgentsIdPermission`, `PutAgentsIdPassword`, `PostAgentsIdDirectHashRegenerate`
- `DeleteTeamsId`, `PutTeamsId`, `PostTeamsIdDirectHashRegenerate`
- `DeleteTagsId`, `PutTagsId`
- `DeleteAccesskeysId`, `PutAccesskeysId`

**State-transition (+409):** none. `PutAgentsIdStatus` updates an agent's presence (Available/Away/etc.) but doesn't reject transitions.

**RPC-heavy (+503):** none — single-manager hops.

**Billing-sensitive (+402):** none. Identity/access operations don't deduct balance.

## Modifier reachability checks

- `grep -rE "insufficient|enough.balance|has not.*balance" bin-agent-manager/pkg/ bin-tag-manager/pkg/ 2>&1 | head -10` — confirms no balance pre-check anywhere in the upstream identity managers (accesskey lives within bin-customer-manager or similar — agent-manager owns agents/teams/tags collectively).
- `PutAgentsIdStatus` updates a presence flag; idempotent, no state-machine.
- `PutAgentsIdPassword` updates the password hash; no state contract.
- `PostAgentsIdDirectHashRegenerate` / `PostTeamsIdDirectHashRegenerate` regenerate access tokens (admin-only); deterministic.

## Forward-dependency notes

- After PR 9, remaining unmigrated `bin-api-manager/server/` files cluster into 3 logical groups:
  - **Routing/control (~150 sites):** `calls.go` (partial), `campaigncalls.go`, `campaigns.go`, `conferencecalls.go`, `conferences.go`, `outdials.go`, `outplans.go`, `queuecalls.go`, `queues.go`.
  - **Contacts/extensions (~71 sites):** `contacts.go`, `extensions.go`.
  - **Service-agent and small surfaces (~80+ sites):** `service_agents_*.go` (multiple), `storage_files.go`, `aggregated_events.go`, `timelines*.go`, `ws.go`.
- These are out-of-scope for PR 9.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/agents.go` (11 handlers, 39 sites)

This is the biggest single-file migration in the rollout (39 sites). Path-UUID hardening for all by-ID handlers using string IDs:
- `GetAgentsId`, `DeleteAgentsId`, `PutAgentsId`, `PutAgentsIdAddresses`, `PutAgentsIdTagIds`, `PutAgentsIdStatus`, `PutAgentsIdPermission`, `PutAgentsIdPassword`.
- `PostAgentsIdDirectHashRegenerate` already uses `openapi_types.UUID`.

Sample tests in `server/agents_test.go` (file already exists — append):
- `Test_agentsPost_MissingAuthIdentity`
- `Test_agentsPost_InvalidJSONBody`
- `Test_agentsIDPut_InvalidID`
- `Test_agentsIDPasswordPut_InvalidJSONBody`

### Task 3: Migrate `server/teams.go` (6 handlers, 18 sites)

Path-UUID hardening for `GetTeamsId`, `DeleteTeamsId`, `PutTeamsId` (string IDs). `PostTeamsIdDirectHashRegenerate` already uses `openapi_types.UUID`.

Sample test:
- `Test_teamsIDPut_InvalidID`

### Task 4: Migrate `server/tags.go` (5 handlers, 15 sites)

Path-UUID hardening for `GetTagsId`, `DeleteTagsId`, `PutTagsId`.

### Task 5: Migrate `server/accesskeys.go` (5 handlers, 13 sites)

Path-UUID hardening for `GetAccesskeysId`, `DeleteAccesskeysId`, `PutAccesskeysId`.

Sample test:
- `Test_accesskeysPost_MissingAuthIdentity`

### Task 6: OpenAPI path wiring

Files under `bin-openapi-manager/openapi/paths/`:

- `agents/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `agents/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `agents/id_addresses.yaml`, `id_tag_ids.yaml`, `id_status.yaml`, `id_permission.yaml`, `id_password.yaml` — PUT (400, 401, 403, 404, 500)
- `agents/id_direct_hash_regenerate.yaml` — POST (400, 401, 403, 404, 500)
- `teams/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `teams/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `teams/id_direct_hash_regenerate.yaml` — POST (400, 401, 403, 404, 500)
- `tags/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `tags/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `accesskeys/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `accesskeys/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)

Verify exact filenames with `ls`. Match yaml `$ref` style from PR 7-8.

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 7: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Add `agent-manager` section** (new):
   - `AGENT_NOT_FOUND` (404).
   - `TEAM_NOT_FOUND` (404).
   - `TAG_NOT_FOUND` (404).
   - Note: agent/team/tag operations are admin-gated for write surfaces; non-admin callers receive 403 PERMISSION_DENIED via translator's `"no permission"` pattern.

2. **Add `customer-manager` (or matching) section** for accesskeys (new):
   - `ACCESSKEY_NOT_FOUND` (404). (If accesskeys live under agent-manager rather than customer-manager, fold this into the agent-manager section instead.)

3. **Update "Other Domains" deferred list at the bottom**:
   - Remove `agent-manager` (now populated).

Match disclaimer style from PR 4-8.

Rebuild Sphinx HTML.

### Task 8: Full verification

Standard 5-step workflow.

### Task 9: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr9-identity-group` on every commit.
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
- New agent-manager catalog section (covers agents, teams, tags, and possibly accesskeys)
- All 85 sites converted
- Zero 402 / 409 declarations added
