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
1. GET /v1/billings/{billing-id}        → Get billing record
2. GET /v1/billing_accounts/{account-id} → Get account details (if needed)
3. GET /v1/calls/{call-id}              → Get call details (if needed)
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

---
