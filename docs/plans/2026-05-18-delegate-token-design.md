# Design: Delegate Token for Project Admin Users

**Date:** 2026-05-18
**Branch:** NOJIRA-Add-delegate-token
**Status:** Approved (3-iteration review)

---

## Problem

Platform superadmins occasionally need full customer admin control over a specific customer's account for support and debugging. Today they have no scoped mechanism to do this — they must either use their own superadmin token (over-privileged) or ask a customer admin to share credentials (insecure).

---

## Solution

Add a new `POST /auth/delegate` endpoint that issues a short-lived JWT (`type: "delegate"`) granting `PermissionCustomerAdmin`-equivalent access scoped to a specific target customer. Only agents with `PermissionProjectSuperAdmin` may call this endpoint.

---

## API Endpoint

```
POST /auth/delegate
```

**Authentication:** Bearer token with `PermissionProjectSuperAdmin` required.

Callers whose identity type is `TypeDelegate` are explicitly rejected with 403 — recursive delegation is not permitted.

**Request body:**
```json
{
  "customer_id": "<uuid>",
  "reason": "<string, 10–200 printable chars, no control characters>"
}
```

**Response:**
```json
{
  "token": "<signed JWT>",
  "customer_id": "<uuid>",
  "expire": "<RFC3339 UTC, e.g. 2026-05-19T06:00:00Z>"
}
```

**Error codes:**

| Code | Condition |
|------|-----------|
| `401` | Not authenticated |
| `403` | Not `PermissionProjectSuperAdmin`, or caller is `TypeDelegate` |
| `404` | Target customer not found or deleted |
| `422` | `reason` fails validation, or `customer_id` is malformed |

---

## JWT Token Structure

```json
{
  "type": "delegate",
  "sub": "<superadmin-agent-id>",
  "aud": "voipbin-api",
  "jti": "<uuid-v4>",
  "iat": 1747526400,
  "nbf": 1747526370,
  "exp": 1747555200,
  "customer_id": "<target-customer-uuid>"
}
```

Key design decisions:
- `reason` is **not** included in JWT claims — stored only in the audit log to prevent forwarding sensitive support context to downstream services.
- `aud` enforcement is scoped to `type=delegate` tokens only in v1 — no backward-compatibility impact on existing `TypeAgent` / `TypeDirect` tokens.
- `nbf = iat - 30s` for clock-skew tolerance.
- `jti` (UUID v4) enables forensic correlation across issuance and use logs.
- Token lifetime: **8 hours** (`DelegateExpiration = 8 * time.Hour`).

---

## Constants

Located in `bin-api-manager` only (NOT `bin-common-handler` — used by a single service):

```go
const (
    TypeAgent    Type = "agent"
    TypeDirect   Type = "direct"
    TypeDelegate Type = "delegate"   // new

    DelegateExpiration = 8 * time.Hour
)
```

---

## Middleware Changes (`lib/middleware/authenticate.go`)

`buildJWTIdentity()` gains a new case:

```go
case auth.TypeDelegate:
    // 1. Validate aud == "voipbin-api" → 401 if mismatch
    // 2. Extract customer_id from claims
    // 3. Extract jti from claims
    // 4. Return AuthIdentity{
    //      Type:       TypeDelegate,
    //      CustomerID: target customer UUID,
    //      JTI:        jti,
    //    }
default:
    return nil, fmt.Errorf("unknown token type: %q", tokenType)
```

### Customer Scope Invariant

**`identity.CustomerID` is the authoritative scope for all downstream queries.** Handlers already filter resources by `identity.CustomerID` and do not look up the subject agent's home customer. A delegate token's `CustomerID = target customer UUID` therefore correctly scopes every query to the target customer without per-handler changes.

### Request-Level Tracing

When `identity.Type == TypeDelegate`, middleware appends to the request-scoped logger:
```go
log = log.WithFields(logrus.Fields{
    "delegate": true,
    "jti":      identity.JTI,
})
```
This enables correlated audit traces from token issuance through every request it authorizes.

### `HasPermission()` for `TypeDelegate`

Returns `true` for customer-level permissions only:
- `PermissionCustomerAdmin` ✅
- `PermissionCustomerManager` ✅
- `PermissionCustomerAgent` ✅
- `PermissionProjectSuperAdmin` ❌
- `PermissionProjectAll` ❌

---

## `AuthIdentity` Changes (`models/auth/auth.go`)

- Add `TypeDelegate` to the `Type` enum.
- Add `JTI string` field to `AuthIdentity` (populated only for `TypeDelegate`; empty for other types).

---

## Service Handler Flow

`AuthDelegate(ctx, identity, targetCustomerID, reason)` in `bin-api-manager/pkg/servicehandler/`:

1. Reject if `identity.Type == TypeDelegate` → 403 (`denial_reason: "recursive_delegation"`)
2. Verify caller has `PermissionProjectSuperAdmin` → 403 (`denial_reason: "not_superadmin"`)
3. Validate `customer_id` as UUID; normalize to lowercase-with-dashes form → 422 if invalid
4. Validate `reason`: 10–200 chars, printable ASCII + space, no control chars (`\n \r \t \x00`) → 422 if invalid
5. Verify target customer exists and is not deleted via RPC to `bin-customer-manager` → 404 if not
6. Generate `jti` (UUID v4)
7. Generate JWT via `authJWTGenerateWithExpiration()` with all required claims
8. Write audit INFO log (see below)
9. Emit Prometheus metric
10. Return token + customer_id + RFC3339 expire string

---

## Audit Logging

### Issuance success (INFO)

```go
h.log.WithFields(logrus.Fields{
    "audit":               true,
    "event":              "delegate_token_issued",
    "jti":                jtiUUID,
    "sub":                agentID,
    "target_customer_id": customerID,
    "reason":             reason,
    "exp":                expireUnix,
}).Info("Delegate token issued")
```

### Failed attempt (WARN)

```go
h.log.WithFields(logrus.Fields{
    "audit":               true,
    "event":              "delegate_token_denied",
    "sub":                agentID,
    "target_customer_id": customerID,   // included when parseable
    "denial_reason":      reason,       // enum: "recursive_delegation", "not_superadmin",
                                        //       "customer_not_found", "invalid_input", "rate_limited"
    "error":              err.Error(),
}).Warn("Delegate token request denied")
```

---

## Observability

**Prometheus metric:**
```
auth_delegate_token_issued_total{issued_by_agent_id="<id>"}
```
Note: `issued_by_agent_id` label is acceptable while the superadmin set remains small (<50 agents). If it grows, drop the label and rely on audit logs for per-agent attribution.

---

## Rate Limiting

`POST /auth/delegate` is rate-limited to **10 requests/hour per issuing agent** via the existing Gin rate-limit middleware.

---

## Required Artifacts

- [ ] OpenAPI schema in `bin-openapi-manager`
- [ ] RST docs update in `bin-api-manager/docsdev/source/` (auth section) + clean HTML rebuild + `git add -f docsdev/build/`
- [ ] Alembic migration: **not required** — no schema changes in v1

---

## Test Matrix

| Type | Test case |
|------|-----------|
| Unit | `HasPermission` denies `PermissionProjectSuperAdmin` for `TypeDelegate` |
| Unit | `HasPermission` grants `PermissionCustomerAdmin` for `TypeDelegate` |
| Unit | Recursive delegation (`TypeDelegate` caller) is rejected with 403 |
| Unit | Unknown token type in `buildJWTIdentity` returns error (no fallthrough) |
| Unit | `reason` validation rejects control chars, enforces length bounds |
| Integration | Delegate token scopes customer-scoped `GET /v1.0/agents` to target customer only |
| Integration | Rate limit triggers after 10 issuances/hour from same agent |
| Integration | 404 returned when target customer is soft-deleted |
| Regression | `TypeAgent` and `TypeDirect` tokens without `aud` claim continue to validate |
| Regression | Delegate token for customer A cannot access customer B's resources |

---

## Known Limitations (v1)

| Limitation | Mitigation |
|-----------|-----------|
| No revocation: `jti` provides traceability; full DB-backed revocation is a future enhancement | Short enough lifetime for most support sessions |
| Issuer demotion not re-checked per request: a demoted superadmin's delegate tokens remain valid until expiry | Audit log records issuance; revocation is future work |
| Customer state not re-checked per request: a deleted/suspended customer's token remains valid until expiry | 8h window limits blast radius; documented behavior |
| 8-hour lifetime is longer than the security-ideal 1 hour | Chosen per product requirement; revisit if compliance frameworks require shorter |
