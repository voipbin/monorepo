# PR 12 — Contacts & extensions handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the contacts (with nested phone-number/email/tag sub-resources) and extensions handlers to the canonical error envelope. 20 handlers, 71 sites across 2 files. **No 402, no 409, no 503** modifiers.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 11 (`NOJIRA-api-error-pr11-conferences-queues`, merged `46a6e1ea0`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr12-contacts-extensions`
**Branch:** `NOJIRA-api-error-pr12-contacts-extensions` (branched from `origin/main` at `46a6e1ea0`)

## Scope

| File | Sites | Handlers |
|---|---|---|
| `server/contacts.go` | 53 | 14 |
| `server/extensions.go` | 18 | 6 |

**Total: 71 sites across 20 handlers in 2 files.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetContacts`, `GetContactsLookup`, `GetExtensions`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetContactsId`, `GetExtensionsId`

**Write (no resource ID) → 400, 401, 500:**
- `PostContacts`, `PostExtensions`

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `PutContactsId`, `DeleteContactsId`
- `PostContactsIdPhoneNumbers`, `PostContactsIdEmails`, `PostContactsIdTags`
- `PutContactsIdPhoneNumbersPhoneNumberId`, `DeleteContactsIdPhoneNumbersPhoneNumberId`
- `PutContactsIdEmailsEmailId`, `DeleteContactsIdEmailsEmailId`
- `DeleteContactsIdTagsTagId`
- `PutExtensionsId`, `DeleteExtensionsId`
- `PostExtensionsIdDirectHashRegenerate`

### Dual-ID handlers (5)

Five contacts handlers use a parent contact id PLUS a sub-resource id (phoneNumberId, emailId, or tagId):
- `PutContactsIdPhoneNumbersPhoneNumberId` — id + phoneNumberId
- `DeleteContactsIdPhoneNumbersPhoneNumberId` — id + phoneNumberId
- `PutContactsIdEmailsEmailId` — id + emailId
- `DeleteContactsIdEmailsEmailId` — id + emailId
- `DeleteContactsIdTagsTagId` — id + tagId

Apply path-UUID hardening to **both** UUIDs separately, with a distinguishing message for the inner ID (e.g., "The provided phone_number_id is not a valid UUID."). Pattern matches PR 10's `GetOutdialsIdTargetsTargetId`.

## Modifier reachability checks

### 402 PaymentRequired — NOT WIRED

Contact and extension operations don't deduct balance. No 402 declarations.

### 409 Conflict — NOT WIRED

No state-transition contracts. No 409 declarations.

### 503 ServiceUnavailable — NOT WIRED

Single-manager hops.

## Forward-dependency notes

- After PR 12, remaining unmigrated `bin-api-manager/server/` files cluster into:
  - **calls.go** (partial PR 2 — 10 stragglers in 17 handlers) — likely the most billing-sensitive remaining file (`POST /calls` for outbound dialing).
  - **Service-agent surfaces (~80+ sites):** `service_agents_*.go` (multiple files, including `service_agents_talk.go` with 50 sites alone, `service_agents_contacts.go` with 53 sites mirror of contacts, `service_agents_files.go`, `service_agents_agents.go`, `service_agents_calls.go`, `service_agents_extensions.go`, `service_agents_tags.go`, `service_agents_ws.go`).
  - **Small surfaces (~30 sites):** `storage_files.go`, `aggregated_events.go`, `timelines*.go`, `ws.go`.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/contacts.go` (14 handlers, 53 sites)

Path-UUID hardening for all by-ID handlers. **Dual-ID validation** on the 5 nested-sub-resource handlers using distinguishing messages.

Sample tests in `server/contacts_test.go` (file already exists — append):
- `Test_contactsPost_MissingAuthIdentity`
- `Test_contactsPost_InvalidJSONBody`
- `Test_contactsIDPut_InvalidID`
- `Test_contactsIDPhoneNumbersPhoneNumberIDDelete_InvalidPhoneNumberID` — dual-ID test (valid id, malformed phoneNumberId)

### Task 3: Migrate `server/extensions.go` (6 handlers, 18 sites)

Path-UUID hardening for `GetExtensionsId`, `PutExtensionsId`, `DeleteExtensionsId` (string IDs). `PostExtensionsIdDirectHashRegenerate` already uses `openapi_types.UUID`.

Sample test:
- `Test_extensionsIDPut_InvalidID`

### Task 4: OpenAPI path wiring

Files under `bin-openapi-manager/openapi/paths/`:

- `contacts/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `contacts/lookup.yaml` — GET (401, 500)
- `contacts/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `contacts/id_phone_numbers.yaml` — POST (400, 401, 403, 404, 500)
- `contacts/id_phone_numbers_phone_number_id.yaml` — PUT, DELETE (400, 401, 403, 404, 500)
- `contacts/id_emails.yaml` — POST (400, 401, 403, 404, 500)
- `contacts/id_emails_email_id.yaml` — PUT, DELETE (400, 401, 403, 404, 500)
- `contacts/id_tags.yaml` — POST (400, 401, 403, 404, 500)
- `contacts/id_tags_tag_id.yaml` — DELETE (400, 401, 403, 404, 500)
- `extensions/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `extensions/id.yaml` — GET, PUT, DELETE (400, 401, 403, 404, 500)
- `extensions/id_direct_hash_regenerate.yaml` — POST (400, 401, 403, 404, 500)

Verify exact filenames with `ls`. **No 402, no 409 anywhere.**

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 5: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Add `contact-manager` section** (new):
   - `CONTACT_NOT_FOUND` (404)
   - `CONTACT_PHONE_NUMBER_NOT_FOUND` (404)
   - `CONTACT_EMAIL_NOT_FOUND` (404)
   - `CONTACT_TAG_NOT_FOUND` (404)
   - (If sub-resources don't have distinct typed errors yet, fold them into a single `CONTACT_SUB_RESOURCE_NOT_FOUND` or list with a translator-fallback note.)

2. **Add `registrar-manager` section** (new) for extensions, OR fold into an appropriate existing section if extensions live elsewhere:
   - `EXTENSION_NOT_FOUND` (404).

3. **Update "Other Domains" deferred list at the bottom**:
   - Remove `contact-manager` (now populated). `registrar-manager` likely was never in the list.

Verify upstream manager ownership: `ls bin-customer-manager/models/contact/`, `ls bin-registrar-manager/models/extension/` (or whatever managers own these resources).

Match disclaimer style from PR 4-11.

Rebuild Sphinx HTML.

### Task 6: Full verification

Standard 5-step workflow.

### Task 7: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr12-contacts-extensions` on every commit.
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

- All 7 tasks committed
- `go test -race ./...` green
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- New contact-manager catalog section (with sub-resource entries)
- New registrar-manager (or matching) catalog section for extensions
- All 71 sites converted
- Zero 402 / 409 declarations added
