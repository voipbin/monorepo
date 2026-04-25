# PR 13 — Service-agents group handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the agent-scoped resource handlers (`service_agents_*` excluding `_talk`) to the canonical error envelope. 28 handlers, 89 sites across 7 files. **No 402, no 409, no 503** modifiers.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 12 (`NOJIRA-api-error-pr12-contacts-extensions`, merged `75d3c90af`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr13-service-agents`
**Branch:** `NOJIRA-api-error-pr13-service-agents` (branched from `origin/main` at `75d3c90af`)

## Scope

| File | Sites | Handlers |
|---|---|---|
| `server/service_agents_agents.go` | 5 | 2 |
| `server/service_agents_calls.go` | 5 | 2 |
| `server/service_agents_contacts.go` | 53 | 14 |
| `server/service_agents_extensions.go` | 5 | 2 |
| `server/service_agents_files.go` | 15 | 5 |
| `server/service_agents_tags.go` | 5 | 2 |
| `server/service_agents_ws.go` | 1 | 1 |

**Total: 89 sites across 28 handlers in 7 files.**

`service_agents_talk.go` (15 handlers, 50 sites) is deferred to PR 14 — it's a complex single-file migration with WebSocket lifecycle concerns that justify a focused PR.

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetServiceAgentsAgents`, `GetServiceAgentsCalls`, `GetServiceAgentsContacts`, `GetServiceAgentsContactsLookup`, `GetServiceAgentsExtensions`, `GetServiceAgentsFiles`, `GetServiceAgentsTags`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetServiceAgentsAgentsId`, `GetServiceAgentsCallsId`, `GetServiceAgentsContactsId`, `GetServiceAgentsExtensionsId`, `GetServiceAgentsFilesId`, `GetServiceAgentsFilesIdFile`, `GetServiceAgentsTagsId`

**WebSocket connection (read-shaped) → 401, 500:**
- `GetServiceAgentsWs` — establishes a WebSocket. Pre-handshake errors map to 401/500. Once upgraded, errors flow over the WS frame; the HTTP envelope only governs the initial response.

**Write (no resource ID) → 400, 401, 500:**
- `PostServiceAgentsContacts`, `PostServiceAgentsFiles`

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `PutServiceAgentsContactsId`, `DeleteServiceAgentsContactsId`
- `PostServiceAgentsContactsIdPhoneNumbers`, `PostServiceAgentsContactsIdEmails`, `PostServiceAgentsContactsIdTags`
- `PutServiceAgentsContactsIdPhoneNumbersPhoneNumberId`, `DeleteServiceAgentsContactsIdPhoneNumbersPhoneNumberId`
- `PutServiceAgentsContactsIdEmailsEmailId`, `DeleteServiceAgentsContactsIdEmailsEmailId`
- `DeleteServiceAgentsContactsIdTagsTagId`
- `DeleteServiceAgentsFilesId`

### Dual-ID handlers (5 — mirror of PR 12)

The `service_agents_contacts.go` file mirrors `contacts.go` exactly. The same 5 dual-ID handlers exist:
- `PutServiceAgentsContactsIdPhoneNumbersPhoneNumberId`, `DeleteServiceAgentsContactsIdPhoneNumbersPhoneNumberId`
- `PutServiceAgentsContactsIdEmailsEmailId`, `DeleteServiceAgentsContactsIdEmailsEmailId`
- `DeleteServiceAgentsContactsIdTagsTagId`

Apply the same dual-ID validation pattern as PR 12 (parent `id` first, then inner ID with distinguishing message).

## Modifier reachability checks

### 402 PaymentRequired — NOT WIRED

`grep -rE "insufficient|enough.balance|has not.*balance" bin-storage-manager/pkg/` returns zero matches. File upload to storage-manager doesn't surface a balance error. No 402 declarations.

### 409 Conflict — NOT WIRED

No state-transition contracts in service-agent layer.

### 503 ServiceUnavailable — NOT WIRED

Single-manager hops.

## Forward-dependency notes

- After PR 13, remaining unmigrated `bin-api-manager/server/` files:
  - **service_agents_talk.go** — 15 handlers, 50 sites — PR 14 candidate (WebSocket-heavy talk surface).
  - **calls.go** — partial PR 2 migration with 10 stragglers in 17 handlers — PR 15 candidate (likely the most billing-sensitive remaining file with `POST /calls`).
  - **Small surfaces:** `storage_files.go` (15 sites), `aggregated_events.go` (1 site), `timelines.go` (1 site), `timelines_sip.go` (1 site), `ws.go` (1 site) — could batch into PR 16 along with `calls.go` cleanup.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/service_agents_contacts.go` (14 handlers, 53 sites)

Apply path-UUID hardening on all by-ID handlers. **Dual-ID validation** on the 5 nested-sub-resource handlers using distinguishing messages (mirror PR 12's pattern exactly).

Add tests in `service_agents_contacts_test.go` (file already exists — append):
- `Test_serviceAgentsContactsPost_MissingAuthIdentity`
- `Test_serviceAgentsContactsPost_InvalidJSONBody`
- `Test_serviceAgentsContactsIDPut_InvalidID`
- `Test_serviceAgentsContactsIDPhoneNumbersPhoneNumberIDDelete_InvalidPhoneNumberID`

### Task 3: Migrate small read-only files (`service_agents_agents.go`, `service_agents_calls.go`, `service_agents_extensions.go`, `service_agents_tags.go`)

8 handlers total, 20 sites. All read-only (Get/Get-by-id pattern). Path-UUID hardening on by-ID handlers.

Sample tests (one per file optional):
- `Test_serviceAgentsAgentsIDGet_InvalidID`
- `Test_serviceAgentsCallsIDGet_InvalidID`

### Task 4: Migrate `server/service_agents_files.go` (5 handlers, 15 sites)

Handlers: `PostServiceAgentsFiles`, `GetServiceAgentsFiles`, `GetServiceAgentsFilesId`, `DeleteServiceAgentsFilesId`, `GetServiceAgentsFilesIdFile`. Path-UUID hardening on by-ID handlers (`GetServiceAgentsFilesIdFile` already uses `openapi_types.UUID`).

Sample test:
- `Test_serviceAgentsFilesPost_MissingAuthIdentity`

### Task 5: Migrate `server/service_agents_ws.go` (1 handler, 1 site)

`GetServiceAgentsWs` — WebSocket upgrade endpoint. Pre-handshake auth/setup errors return 401/500. Trivial — the single abort site is for missing auth identity → 401.

### Task 6: OpenAPI path wiring

Files under `bin-openapi-manager/openapi/paths/`:

- `service_agents/agents/main.yaml`, `id.yaml` (read patterns)
- `service_agents/calls/main.yaml`, `id.yaml`
- `service_agents/contacts/*.yaml` (mirror of contacts/*.yaml structure including dual-ID paths)
- `service_agents/extensions/main.yaml`, `id.yaml`
- `service_agents/files/main.yaml`, `id.yaml`, `id_file.yaml`
- `service_agents/tags/main.yaml`, `id.yaml`
- `service_agents/ws.yaml`

Verify exact filenames with `ls bin-openapi-manager/openapi/paths/service_agents/`. **No 402, no 409 anywhere.**

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 7: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

The service-agents endpoints are agent-scoped views of existing resources. Their NOT_FOUND reasons reuse the resource manager's catalog entries (e.g., `CONTACT_NOT_FOUND` from PR 12's contact-manager section). No new manager sections required — but add a brief note in the **agent-manager** section (or new `service-agents` section) explaining that these endpoints reuse the underlying manager's reasons.

Possibly add `STORAGE_FILE_NOT_FOUND` (404) to a new **storage-manager** section if files are owned by `bin-storage-manager` (verify ownership). If `STORAGE_FILE_NOT_FOUND` is the only reason from storage-manager today, this section can be very short.

Match disclaimer style from PR 4-12.

Rebuild Sphinx HTML.

### Task 8: Full verification

Standard 5-step workflow.

### Task 9: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr13-service-agents` on every commit.
- No AI attribution.
- `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` and `assertErrorResponse` helpers.
- Path-UUID hardening pattern: `uuid.FromStringOrNil(id) == uuid.Nil`.
- Dual-ID handlers: validate both UUIDs separately with distinguishing messages.
- `getAuthIdentity(c)` (NOT `commonmiddleware.AuthIdentityGet`).
- `abortWithError` takes `*cerrors.VoipbinError`.
- Standard message strings: "The request body is not valid JSON.", "The provided id is not a valid UUID.", "Authentication is required."
- Preserve existing `log.Errorf/Infof` site-specific lines.

## Success criteria

- All 9 tasks committed
- `go test -race ./...` green
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- All 89 sites converted
- Zero 402 / 409 declarations added
