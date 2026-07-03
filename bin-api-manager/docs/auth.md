# bin-api-manager Authentication & Authorization

## Authentication Flow

All authentication logic lives exclusively in `bin-api-manager`. Backend services never validate tokens or check ownership — they trust that api-manager has already done so.

### Credential Sources

The middleware (`lib/middleware/authenticate.go`) checks three sources in order:

1. **Cookie** — `token=<jwt>` or `accesskey=<key>`
2. **Query parameter** — `?token=<jwt>` or `?accesskey=<key>`
3. **HTTP Header** — `Authorization: Bearer <jwt>`

If no credential is found, the request is rejected with `401 AUTHENTICATION_REQUIRED`.

### Token Types

There are two credential types, handled by separate code paths:

#### JWT Tokens (Bearer tokens)

JWT tokens are issued by `POST /auth/boot` (login). The token contains a `type` claim:

| JWT `type` | Identity Type | Used by |
|-----------|--------------|---------|
| `"agent"` (or missing) | Agent token — contains an `amagent.Agent` struct | Human users via admin/talk UIs |
| `"direct"` | Direct token — contains a `DirectScope` struct | Machine-to-machine integrations with scoped access |

After parsing:
1. If `type == "agent"`: the embedded `amagent.Agent` is extracted and set as the auth identity.
2. If `type == "direct"`: the embedded `DirectScope` is extracted and set as the auth identity.

#### Access Keys

Access keys are persistent API credentials stored in api-manager's local database. They are validated by:
1. Looking up the key in the database (`pkg/dbhandler/`).
2. Checking `TMExpire` — reject if expired.
3. Checking `TMDelete` — reject if soft-deleted.

Access keys do not go through RabbitMQ; validation is local.

### Customer Frozen Check

After successful authentication, if the customer account status is `frozen`:
- All requests are rejected with `403 PERMISSION_DENIED` / `ACCOUNT_FROZEN`.
- The response includes `deletion_scheduled_at`, `deletion_effective_at`, and `recovery_endpoint` in the error details.
- **Exceptions** (allowed for frozen accounts):
  - `DELETE /auth/unregister`
  - `POST /auth/unregister`
  - Requests from agents with `PermissionProjectSuperAdmin`
  - Direct token requests (skip frozen check)

---

## Authorization Model

### Architecture

```
External Client → bin-api-manager (Auth Layer) → bin-<service>-manager (Business Logic)
```

bin-api-manager is the sole authorization enforcement point. Backend services receive RPC calls and return data without performing any customer_id checks or JWT validation.

### Two-Level Handler Pattern

Every resource follows this pattern in `pkg/servicehandler/`:

```go
// Private helper — fetch resource, no permission check
func (h *serviceHandler) resourceGet(ctx context.Context, resourceID uuid.UUID) (*Resource, error) {
    return h.reqHandler.ServiceV1ResourceGet(ctx, resourceID)
}

// Public method — permission check + WebhookMessage conversion
func (h *serviceHandler) ResourceGet(ctx context.Context, a *amagent.Agent, resourceID uuid.UUID) (*ResourceWebhookMessage, error) {
    r, err := h.resourceGet(ctx, resourceID)
    if err != nil {
        return nil, err
    }
    if !h.hasPermission(ctx, a, r.CustomerID, amagent.PermissionCustomerAdmin) {
        return nil, fmt.Errorf("user has no permission")
    }
    return r.ConvertWebhookMessage(), nil
}
```

### Permission Levels

Permissions are bit flags defined in `bin-agent-manager/models/agent/`. Key levels:

| Permission Constant | Meaning |
|--------------------|---------|
| `PermissionCustomerAdmin` | Customer-level admin (owns all customer resources) |
| `PermissionCustomerManager` | Can manage resources within the customer |
| `PermissionProjectSuperAdmin` | Platform-level superadmin (bypasses frozen check, can access all customers) |

Most endpoints require `PermissionCustomerAdmin`. Some allow `PermissionCustomerAdmin | PermissionCustomerManager`. Billing and billing-account endpoints require `PermissionCustomerAdmin` only.

### How `hasPermission` Works

```
hasPermission(ctx, agent, resourceCustomerID, requiredPermission)
```

- If `agent.CustomerID == resourceCustomerID` AND `agent.Permission & requiredPermission != 0` → allowed.
- If `agent` has `PermissionProjectSuperAdmin` → allowed regardless of customer ID.
- Otherwise → denied.

This means agents can only access resources belonging to their own customer, unless they are a platform superadmin.

### Service Agent Auth (scoped tokens)

`/service_agents/*` endpoints use the same JWT validation but with a narrower scope. The agent identity is the authenticated agent; resources are automatically scoped to `agent.CustomerID`. These endpoints are intended for the agent-facing UI (talk.voipbin.net, square-talk).

**CRITICAL: `square-talk` (and any other Agent-facing frontend) MUST call ONLY `/service_agents/*` paths — never the top-level `/<resource>` path directly, even if the top-level path's permission bitmask happens to allow Agent-level access.** This is a deliberate API-surface separation, not an accident of history:

- **Top-level `/<resource>` endpoints** (e.g. `/transcribes`, `/contacts`) are the Admin/Manager-facing surface (`square-admin`). Their `hasPermission(...)` checks commonly hardcode `PermissionCustomerAdmin|PermissionCustomerManager` and are free to change that bitmask over time to fit admin-console needs.
- **`/service_agents/<resource>` endpoints** are the dedicated Agent-facing surface. They exist as their own path + servicehandler function (e.g. `ServiceAgentContactList`, `ServiceAgentTranscribeList`) specifically so their permission model (`amagent.PermissionAll`, i.e. "any authenticated agent of this customer") can evolve independently of the Admin/Manager surface's needs.

**Do NOT "fix" a missing Agent-facing capability by relaxing the top-level endpoint's permission bitmask instead of adding the missing `/service_agents/<resource>` endpoint.** That shortcut collapses the two intentionally-separate surfaces into one, makes future permission changes on the Admin surface risk silently breaking the Agent-facing app (and vice versa), and violates the frontend's own architectural contract (square-talk's CLAUDE.md/skill states it uses `service_agents/*` only). If a `service_agents/<resource>` endpoint doesn't exist yet for a capability an Agent-facing app needs, add it as a new resource (new OpenAPI path file, new `ServiceAgent*` servicehandler function reusing the existing private `resourceGet`/`resourceList` helpers, new `server/service_agents_<resource>.go` handler) — do not touch the top-level endpoint's permission check. See `bin-openapi-manager/CLAUDE.md`'s "Adding a Service Agent (Agent-facing) resource" section for the concrete checklist, and `pkg/servicehandler/serviceagent_transcribe.go` for a worked example (adds `ServiceAgentTranscribeList`/`ServiceAgentTranscribeStart` alongside the pre-existing Admin/Manager-gated `TranscribeList`/`TranscribeStart`, without changing the latter).

### Direct Token Auth

Direct tokens (`type == "direct"`) carry a `DirectScope` that specifies which resources the token may access. This is used for machine-to-machine integrations where a narrow-scoped credential is preferred over a full agent token.

### External Response Policy

All handlers that return resource data MUST call `.ConvertWebhookMessage()` on the internal model before returning it. This strips internal-only fields (`PodID`, `Username`, `PermissionIDs`, etc.) that must not be exposed externally.

RST struct documentation (`docsdev/source/*_struct_*.rst`) MUST match the `WebhookMessage` fields, not the internal model struct. See `models/<entity>/webhook.go` in each backend service for the authoritative field list.
