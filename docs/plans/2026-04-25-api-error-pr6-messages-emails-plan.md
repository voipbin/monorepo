# PR 6 — Messages, emails, AI messages, conversations handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the messages, emails, AI-messages, conversations, and service-agent conversations handlers to the canonical error envelope. Second PR (after PR 4) to apply the §6.1 **402 modifier** — this time for the messaging billing-sensitive write endpoints.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 5 (`NOJIRA-api-error-pr5-billing-customers`, merged `84148d2db`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr6-messages-emails`
**Branch:** `NOJIRA-api-error-pr6-messages-emails` (branched from `origin/main` at `84148d2db`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/messages.go` | 155 | 11 | 4 |
| `server/emails.go` | 160 | 11 | 4 |
| `server/aimessages.go` | 164 | 12 | 4 |
| `server/conversations.go` | 241 | 18 | 5 |
| `server/service_agents_conversations.go` | 183 | 12 | 4 |

**Total: 64 sites across 21 handlers in 5 files.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetMessages`, `GetEmails`, `GetAimessages`, `GetConversations`, `GetServiceAgentsConversations`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetMessagesId`, `GetEmailsId`, `GetAimessagesId`, `GetConversationsId`, `GetServiceAgentsConversationsId`
- `GetConversationsIdMessages`, `GetServiceAgentsConversationsIdMessages` (read with-id; the id is a parent conversation)

**Write (no resource ID, BILLING-SENSITIVE) → 400, 401, 402, 500:**
- `PostMessages` — SMS send. Charges per recipient via message-manager → billing-manager.
- `PostEmails` — email send. Charges per email via email-manager → billing-manager.
- `PostAimessages` — AI message creation (LLM token cost). Charges via ai-manager → billing-manager.

**Write (with resource ID, BILLING-SENSITIVE) → 400, 401, 402, 403, 404, 500:**
- `PostConversationsIdMessages` — sends a message in a conversation thread. Provider cost (LINE/Messenger/etc.) via conversation-manager → billing-manager.
- `PostServiceAgentsConversationsIdMessages` — agent surface for the same conversation message send.

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `DeleteMessagesId`, `DeleteEmailsId`, `DeleteAimessagesId` — message/email retention deletion.
- `PutConversationsId` — update conversation metadata.

**State-transition (+409):** none.

**RPC-heavy (+503):** none — each operation routes to a single manager.

## Translator coverage

The translator's `"insufficient"` substring pattern (added in PR 2) already routes downstream balance failures to **402 / `INSUFFICIENT_BALANCE`**. No new translator work in PR 6 — billing failures from message-manager / email-manager / ai-manager / conversation-manager propagate as `"insufficient balance"` style messages and route correctly today.

## Forward-dependency notes

- This is the last PR in the originally-planned 6-PR rollout (PR 0 → PR 6) for the canonical error envelope. **However**, ~35 additional `bin-api-manager/server/` files still use the old `c.AbortWithStatus` / `c.JSON(http.Status...)` patterns (calls.go, agents.go, contacts.go, queues.go, conferences.go, campaigns.go, ais.go, aicalls.go, aisummaries.go, speakings.go, transcribes.go, etc.). These represent ~700 sites that will require follow-up PRs (PR 7+) outside this rollout's original scope.
- The migration's foundational layer (cerrors, translator, helpers, baseline §6.1 wiring, billing/admin/state-transition modifiers) is now stable and reusable. Subsequent PRs follow the same per-resource-group cadence.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/messages.go` (4 handlers, 11 sites)

Path-UUID hardening for `GetMessagesId`, `DeleteMessagesId`.

Sample tests in `server/messages_test.go`:
- `Test_messagesPost_MissingAuthIdentity`
- `Test_messagesPost_InvalidJSONBody`
- `Test_messagesIDDelete_InvalidID`
- `Test_messagesPost_InsufficientBalance` — mock servicehandler `MessageSend` to return `fmt.Errorf("insufficient balance")`; assert PAYMENT_REQUIRED / INSUFFICIENT_BALANCE end-to-end.

### Task 3: Migrate `server/emails.go` (4 handlers, 11 sites)

Path-UUID hardening for `GetEmailsId`, `DeleteEmailsId`.

Sample test in `server/emails_test.go`:
- `Test_emailsPost_InsufficientBalance` — same pattern as messages.

### Task 4: Migrate `server/aimessages.go` (4 handlers, 12 sites)

Path-UUID hardening for `GetAimessagesId`, `DeleteAimessagesId`.

Sample test in `server/aimessages_test.go`:
- `Test_aimessagesPost_InsufficientBalance`.

### Task 5: Migrate `server/conversations.go` (5 handlers, 18 sites)

Handlers: `GetConversations`, `GetConversationsId`, `PutConversationsId`, `GetConversationsIdMessages`, `PostConversationsIdMessages`.

Path-UUID hardening for all by-ID handlers.

`PostConversationsIdMessages` is billing-sensitive (provider cost); declare 402.

### Task 6: Migrate `server/service_agents_conversations.go` (4 handlers, 12 sites)

Handlers: `GetServiceAgentsConversations`, `GetServiceAgentsConversationsId`, `GetServiceAgentsConversationsIdMessages`, `PostServiceAgentsConversationsIdMessages`.

Same pattern as conversations.go. Path-UUID hardening for by-ID handlers.

### Task 7: OpenAPI path wiring

Wire all `/messages*`, `/emails*`, `/aimessages*`, `/conversations*`, `/service-agents/conversations*` paths per §6.1 baseline:

- `GET /messages`, `GET /emails`, `GET /aimessages`, `GET /conversations`, `GET /service-agents/conversations` → 401, 500
- `GET /messages/{id}`, `GET /emails/{id}`, `GET /aimessages/{id}`, `GET /conversations/{id}`, `GET /service-agents/conversations/{id}` → 400, 401, 403, 404, 500
- `POST /messages`, `POST /emails`, `POST /aimessages` → 400, 401, **402**, 500
- `POST /conversations/{id}/messages`, `POST /service-agents/conversations/{id}/messages` → 400, 401, **402**, 403, 404, 500
- `DELETE /messages/{id}`, `DELETE /emails/{id}`, `DELETE /aimessages/{id}`, `PUT /conversations/{id}` → 400, 401, 403, 404, 500
- `GET /conversations/{id}/messages`, `GET /service-agents/conversations/{id}/messages` → 400, 401, 403, 404, 500

Regenerate `gens/openapi_server/gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 8: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Extend `billing-manager` section** (PR 4 added `INSUFFICIENT_BALANCE`; PR 5 added `BILLING_NOT_FOUND` and `BILLING_ACCOUNT_NOT_FOUND`):
   - Update the `INSUFFICIENT_BALANCE` description's "currently fired for" list to include the PR 6 endpoints: `POST /messages`, `POST /emails`, `POST /aimessages`, `POST /conversations/{id}/messages`, `POST /service-agents/conversations/{id}/messages`.

2. **Add `message-manager` section** (new):
   - `MESSAGE_NOT_FOUND` (404) — reachable via `"not found"` fallback.

3. **Add `email-manager` section** (new):
   - `EMAIL_NOT_FOUND` (404).

4. **Add `ai-manager` section** (new, scoped to ai-message; broader ai-call/ai-summary will be added in a future PR):
   - `AIMESSAGE_NOT_FOUND` (404).

5. **Extend `conversation-manager` section** (added in PR 5 with `CONVERSATION_ACCOUNT_NOT_FOUND`):
   - Add `CONVERSATION_NOT_FOUND` (404) — reachable via `"not found"` fallback.

Match disclaimer style from PR 4/PR 5 (translator-reachable today vs typed-error future).

Rebuild Sphinx HTML.

### Task 9: Full verification

Standard 5-step workflow for both `bin-openapi-manager` and `bin-api-manager`.

### Task 10: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr6-messages-emails` on every commit.
- No AI attribution.
- `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` and `assertErrorResponse` helpers.
- Path-UUID hardening pattern: `uuid.FromStringOrNil(id) == uuid.Nil` (matches PR 3-5).
- `abortWithError` takes `*cerrors.VoipbinError` (verified in PR 5 — actual signature on main).
- Preserve existing `log.Errorf/Infof` site-specific lines.

## Success criteria

- All 10 tasks committed
- `go test -race ./...` green for both `bin-api-manager` and `bin-openapi-manager`
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- Billing-manager catalog `INSUFFICIENT_BALANCE` description updated with the 5 new billing-sensitive endpoints
- New message-manager, email-manager, ai-manager (aimessage-scoped) catalog sections
- conversation-manager section extended with `CONVERSATION_NOT_FOUND`
- All 64 sites converted
- 402 declared on 5 endpoints (messages, emails, aimessages, conversations/{id}/messages, service-agents/conversations/{id}/messages)
