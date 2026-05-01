# Design: Permission-Aware GET /service_agents/conversations

**Date:** 2026-05-01
**Branch:** NOJIRA-ServiceAgent-conversation-permission

## Problem

`GET /v1.0/service_agents/conversations` and `GET /v1.0/service_agents/conversations/{id}` always
scope results to the calling agent's own owned conversations, regardless of the caller's permission
level. Admin and manager agents cannot see conversations assigned to other agents or unassigned
conversations via the service-agent path.

## Goal

Admin and manager callers at the service-agent endpoints should see all conversations belonging to
their customer, matching the behaviour of the existing `/conversations` endpoint.

## Approach: Direct in-place permission check (Approach A)

### `ServiceAgentConversationList`

**Before:** filter map always contains `{FieldDeleted: false, FieldOwnerID: a.AgentID()}`.

**After:**

| Caller | Filter map |
|--------|-----------|
| Regular agent (`PermissionCustomerAgent`) | `{FieldDeleted: false, FieldOwnerID: a.AgentID()}` (unchanged) |
| Admin / Manager | `{FieldDeleted: false, FieldCustomerID: a.CustomerID}` (no owner restriction) |

The `hasPermission(ctx, a, a.CustomerID, PermissionCustomerAdmin|PermissionCustomerManager)` guard
is the gate — same call used by `ConversationGetsByCustomerID`.

### `ServiceAgentConversationGet`

**Before:** rejects the request if `conversation.OwnerID != a.AgentID()` unconditionally.

**After:**
- Admin / Manager: allowed if `conversation.CustomerID == a.CustomerID` (enforced via `hasPermission`).
- Regular agent: still requires `conversation.OwnerID == a.AgentID()`.

## Files changed

- `bin-api-manager/pkg/servicehandler/serviceagent_conversation.go` — two functions modified
- `bin-api-manager/pkg/servicehandler/serviceagent_conversation_test.go` — new test cases

## Out of scope

- No OpenAPI spec changes (no new params or response shape).
- No RST doc changes (privilege expansion for existing callers, not a new API contract).
- `ServiceAgentConversationUpdate` and `ServiceAgentConversationUnassign` already use
  `hasPermission` correctly — no change needed.

## Security notes

- Cross-customer access remains impossible: `hasPermission` binds to `a.CustomerID`.
- `PermissionProjectSuperAdmin` callers are handled transparently by `hasPermission` (they pass the
  check for any customer).
