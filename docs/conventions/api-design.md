# API & External Interfaces

### 10.1 Atomic API Responses

API endpoints return single resource types without combining data from other services:

```go
// CORRECT — single resource
func (h *serviceHandler) BillingGet(ctx context.Context, ...) (*bmbilling.WebhookMessage, error) {
    // Returns just the billing record
}

// WRONG — combined resources
func (h *serviceHandler) BillingGet(ctx context.Context, ...) (*BillingWithAccount, error) {
    // Returns billing + account + call details
}
```

**Exceptions to the atomic-response rule:**

1. **Pagination Metadata** — List responses can include `next_page_token` as it's directly related to the query:
   ```json
   {
     "result": [...],
     "next_page_token": "2024-01-15T10:30:00"
   }
   ```

2. **Atomic Operation Responses** — When a single operation creates multiple related resources, the response can include all created resources:
   ```
   POST /v1/calls (with groupcall option)
   Returns: { "call": {...}, "groupcall": {...} }

   Reason: Call and groupcall are created atomically in one transaction,
   so returning both is appropriate.
   ```

**How to fetch related data:** For all other cases, clients make separate requests:
```
1. GET /v1/billings/{billing-id}   → Get billing record (⚠️ known naming exception, see §10.5)
2. GET /v1/billing_accounts/{id}   → Get account details (if needed)
3. GET /v1/calls/{id}              → Get call details (if needed)
```

For authentication and authorization patterns, see `bin-api-manager/CLAUDE.md`.

### 10.2 Two-Level ServiceHandler

In `bin-api-manager/pkg/servicehandler/`, private helpers return internal structs; public methods return `*WebhookMessage`:

```go
// CORRECT — private: internal struct for permission checks
func (h *serviceHandler) agentGet(ctx context.Context, id uuid.UUID) (*amagent.Agent, error) {
    res, err := h.reqHandler.AgentV1AgentGet(ctx, id)
    log.WithField("agent", res).Debug("Received result.")
    return res, nil
}

// CORRECT — public: WebhookMessage for API response, with permission check
func (h *serviceHandler) AgentGet(ctx context.Context, a *amagent.Agent, agentID uuid.UUID) (*amagent.WebhookMessage, error) {
    tmp, err := h.agentGet(ctx, agentID)
    if a.ID != agentID && !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
        return nil, fmt.Errorf("user has no permission")
    }
    return tmp.ConvertWebhookMessage(), nil
}
```

### 10.3 Filters from Request Body

**CRITICAL: Pagination parameters go in the URL. Filter parameters go in the request body JSON.**

```go
// ✅ CORRECT — pagination from URL, filters from request body
func (h *listenHandler) processV1AgentsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    u, err := url.Parse(m.URI)

    // Pagination from URL
    tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
    pageSize := uint64(tmpSize)
    pageToken := u.Query().Get(PageToken)

    // Filters from request body
    tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
    filters, err := utilhandler.ConvertFilters[agent.FieldStruct, agent.Field](agent.FieldStruct{}, tmpFilters)

    tmp, err := h.agentHandler.List(ctx, pageSize, pageToken, filters)
}

// ❌ WRONG — never parse filters from URL query parameters
customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))  // Will be uuid.Nil!
```

Filter fields are defined via `FieldStruct` with `filter:` tags in `models/<resource>/filters.go`.
For the complete implementation guide, see [common-workflows.md](../workflows/common-workflows.md#parsing-filters-from-request-body).

### 10.4 OpenAPI Schema Sync

When modifying API-facing structs, update the OpenAPI schema to match `WebhookMessage` fields (not internal struct). See the [verification workflow](../../CLAUDE.md#critical-before-committing-changes) for the required regeneration steps.

### 10.5 URL Path Segment Naming

**CRITICAL: Public REST API path segments and path parameters (the routes `bin-api-manager`/`bin-openapi-manager` expose to clients) use underscore (`snake_case`), never hyphen (`kebab-case`).**

```
// CORRECT
PUT  /v1.0/agents/{id}/tag_ids
POST /v1.0/webchat_widgets/{id}/direct_hash_regenerate
GET  /v1.0/contact_cases

// WRONG — do not introduce new hyphenated segments or parameter names
POST /v1.0/agents/{id}/tag-ids
GET  /v1.0/contact-cases
```

This scopes only to the **public REST API surface**. Internal inter-service RPC `requestType` URIs (the strings passed to `bin-common-handler/pkg/requesthandler`) are a separate, unrelated contract and are not covered by this rule.

**Why underscore:** it has been VoIPbin's convention since `bin-openapi-manager`'s first commit — every resource path added at that point used either underscore or a single word, with zero hyphens. Hyphenated segments only appeared during a ~5-month window (roughly Jan–Jun 2026) in the absence of a written rule, and every resource added since (`webchat_widgets`, `webchat_sessions`, `webchat_messages`, `contact_peer_events`) reverted to underscore — including `webchat_widgets` reimplementing an action that exists elsewhere as hyphenated (`direct-hash-regenerate`) using underscore (`direct_hash_regenerate`) instead.

**Known exceptions (do not copy these into new work):** these hyphenated paths/parameters predate this rule and are live, in-use API surface. **Decision (VOIP-1312): these are permanent exceptions, not a pending migration.** They stay hyphenated indefinitely — renaming them is a breaking change for production email links, `voipbin-python-sdk`, `voipbin-go`, the `square-main` public API docs, and multiple `square-admin` views, and the migration cost outweighs the consistency benefit. Do not migrate these; do not reference them as "to be renamed" in other docs or tickets. The rule above applies to all work going forward — every resource added since this window (`webchat_widgets`, `webchat_sessions`, `webchat_messages`, `contact_peer_events`) has already used underscore, and that must continue:

- `POST /auth/email-verify`, `/auth/password-forgot`, `/auth/password-reset` — no version prefix (registered outside the `v1.0` router group)
- `GET /v1.0/timelines/calls/{call_id}/sip-analysis`
- `GET /v1.0/timeline-analyses`
- `GET /v1.0/aggregated-events`
- `POST /v1.0/{agents,ais,conferences,extensions,flows,queues,teams}/{id}/direct-hash-regenerate`
- Path parameter `{billing-id}` on the `billings` resource (`GET /v1.0/billings/{billing-id}`) — parameter name only, not exposed in the URL clients send (`/v1.0/billings/<uuid>`), so this one carries no client-migration cost either way

---
