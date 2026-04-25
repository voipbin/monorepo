# PR 14 — Service-agents talk handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the agent-scoped talk surface (chats, channels, messages, participants, reactions) to the canonical error envelope. 15 handlers, 50 sites in a single file. **No 402, no 409, no 503** modifiers.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 13 (`NOJIRA-api-error-pr13-service-agents`, merged `fd81d0571`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr14-service-agents-talk`
**Branch:** `NOJIRA-api-error-pr14-service-agents-talk` (branched from `origin/main` at `fd81d0571`)

## Scope

| File | Sites | Handlers |
|---|---|---|
| `server/service_agents_talk.go` | 50 | 15 |

**Total: 50 sites across 15 handlers in 1 file.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetServiceAgentsTalkChats`, `GetServiceAgentsTalkChannels`, `GetServiceAgentsTalkMessages`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetServiceAgentsTalkChatsId`, `GetServiceAgentsTalkChatsIdParticipants`, `GetServiceAgentsTalkMessagesId`

**Write (no resource ID) → 400, 401, 500:**
- `PostServiceAgentsTalkChats`, `PostServiceAgentsTalkMessages`

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `PutServiceAgentsTalkChatsId`, `DeleteServiceAgentsTalkChatsId`
- `PostServiceAgentsTalkChatsIdJoin`, `PostServiceAgentsTalkChatsIdParticipants`
- `DeleteServiceAgentsTalkChatsIdParticipantsParticipantId` (dual ID)
- `DeleteServiceAgentsTalkMessagesId`, `PostServiceAgentsTalkMessagesIdReactions`

### Dual-ID handler (1)

- `DeleteServiceAgentsTalkChatsIdParticipantsParticipantId` — id + participantId. Apply same dual-ID validation pattern as PR 12/13 with distinguishing message: "The provided participant_id is not a valid UUID."

## Modifier reachability checks

### 402 PaymentRequired — NOT WIRED

`grep -rE "insufficient|enough.balance|has not.*balance" bin-chat-manager/pkg/ bin-talk-manager/pkg/` returns zero matches. Internal agent talk doesn't deduct balance. No 402 declarations.

### 409 Conflict — NOT WIRED

No state-transition contracts on these endpoints. The Join endpoint is idempotent (joining an already-joined chat returns success).

### 503 ServiceUnavailable — NOT WIRED

Single-manager hops (chat-manager + talk-manager).

## Forward-dependency notes

- After PR 14, remaining unmigrated `bin-api-manager/server/` files:
  - **calls.go** — partial PR 2 migration with 10 stragglers in 17 handlers — PR 15 candidate (likely the most billing-sensitive remaining file with `POST /calls`).
  - **Small surfaces:** `storage_files.go` (15 sites), `aggregated_events.go` (1 site), `timelines.go` (1 site), `timelines_sip.go` (1 site), `ws.go` (1 site) — could batch into PR 15 along with `calls.go` cleanup.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/service_agents_talk.go` (15 handlers, 50 sites)

Path-UUID hardening on all by-ID handlers. **Dual-ID validation** on `DeleteServiceAgentsTalkChatsIdParticipantsParticipantId` with distinguishing `participant_id` message.

Add tests in `service_agents_talk_test.go` (file already exists — append):
- `Test_serviceAgentsTalkChatsPost_MissingAuthIdentity`
- `Test_serviceAgentsTalkChatsPost_InvalidJSONBody`
- `Test_serviceAgentsTalkChatsIDPut_InvalidID`
- `Test_serviceAgentsTalkChatsIDParticipantsParticipantIDDelete_InvalidParticipantID` — dual-ID test
- `Test_serviceAgentsTalkMessagesPost_MissingAuthIdentity`

### Task 3: OpenAPI path wiring

Files under `bin-openapi-manager/openapi/paths/service_agents/talk/` (or matching layout):

- `chats.yaml` (or `chats/main.yaml`) — GET (401, 500), POST (400, 401, 500)
- `chats/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `chats/id_join.yaml` — POST (400, 401, 403, 404, 500)
- `chats/id_participants.yaml` — GET, POST (400, 401, 403, 404, 500)
- `chats/id_participants_participant_id.yaml` — DELETE (400, 401, 403, 404, 500)
- `channels.yaml` — GET (401, 500)
- `messages.yaml` — GET (401, 500), POST (400, 401, 500)
- `messages/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `messages/id_reactions.yaml` — POST (400, 401, 403, 404, 500)

Verify exact filenames with `ls bin-openapi-manager/openapi/paths/service_agents/talk/`. **No 402, no 409 anywhere.**

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 4: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Add `talk-manager` (or `chat-manager`) section** (new) — verify upstream manager ownership:
   - `CHAT_NOT_FOUND` (404)
   - `CHANNEL_NOT_FOUND` (404) — if channels are a distinct resource
   - `MESSAGE_NOT_FOUND` (404) — for talk messages (separate from PR 6's `MESSAGE_NOT_FOUND` for SMS messages — disambiguate via `TALK_MESSAGE_NOT_FOUND` if needed)
   - `PARTICIPANT_NOT_FOUND` (404)

2. **Update "Other Domains" deferred list at the bottom**:
   - Remove `talk-manager` (now populated).

Match disclaimer style from PR 4-13.

Rebuild Sphinx HTML.

### Task 5: Full verification

Standard 5-step workflow.

### Task 6: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr14-service-agents-talk` on every commit.
- No AI attribution.
- `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` and `assertErrorResponse` helpers.
- Path-UUID hardening pattern: `uuid.FromStringOrNil(id) == uuid.Nil`.
- Dual-ID handler: validate both UUIDs separately with distinguishing messages.
- `getAuthIdentity(c)` (NOT `commonmiddleware.AuthIdentityGet`).
- `abortWithError` takes `*cerrors.VoipbinError`.
- Standard message strings: "The request body is not valid JSON.", "The provided id is not a valid UUID.", "Authentication is required."
- Preserve existing `log.Errorf/Infof` site-specific lines.

## Success criteria

- All 6 tasks committed
- `go test -race ./...` green
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- New talk-manager (or matching) catalog section
- All 50 sites converted
- Zero 402 / 409 declarations added
