# Customer Account Deletion (Unregister) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement graceful customer account deletion with 30-day grace period, three enforcement layers, and self-service recovery.

**Architecture:** Customer-manager owns account lifecycle state (active/frozen/deleted). API gateway returns 403 for frozen accounts. Call-manager terminates active calls and rejects new ones. Billing-manager freezes charges and subscriptions. Events drive cross-service coordination.

**Tech Stack:** Go, RabbitMQ (events/RPC), MySQL (Squirrel query builder), Redis (cache), Gin (HTTP), Alembic (migrations)

**Design Doc:** `docs/plans/2026-02-16-customer-account-deletion-design.md`

---

## Task 1: Database Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_customer_customers_add_column_status_tm_deletion_scheduled.py`

**Step 1: Create migration file**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Customer-account-deletion-design/bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "customer_customers add column status tm_deletion_scheduled"
```

**Step 2: Edit the generated migration file**

```python
def upgrade():
    op.execute("""alter table customer_customers add column status varchar(16) not null default 'active' after email_verified;""")
    op.execute("""alter table customer_customers add column tm_deletion_scheduled datetime(6) default null after status;""")
    op.execute("""create index idx_customer_customers_status on customer_customers(status);""")

def downgrade():
    op.execute("""alter table customer_customers drop index idx_customer_customers_status;""")
    op.execute("""alter table customer_customers drop column tm_deletion_scheduled;""")
    op.execute("""alter table customer_customers drop column status;""")
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-dbscheme-manager: Add status and tm_deletion_scheduled columns to customer_customers table"
```

---

## Task 2: Customer Model & Event Types

**Files:**
- Modify: `bin-customer-manager/models/customer/customer.go`
- Modify: `bin-customer-manager/models/customer/event.go`
- Modify: `bin-customer-manager/models/customer/field.go`
- Modify: `bin-customer-manager/models/customer/webhook.go`

**Step 1: Add Status type and constants to customer model**

In `bin-customer-manager/models/customer/customer.go`, add:

```go
// Status type
type Status string

const (
    StatusActive  Status = "active"
    StatusFrozen  Status = "frozen"
    StatusDeleted Status = "deleted"
)
```

Add fields to `Customer` struct (after `EmailVerified`):

```go
Status               Status     `json:"status" db:"status"`
TMDeletionScheduled  *time.Time `json:"tm_deletion_scheduled" db:"tm_deletion_scheduled"`
```

**Step 2: Add new event types**

In `bin-customer-manager/models/customer/event.go`, add:

```go
EventTypeCustomerFrozen    string = "customer_frozen"
EventTypeCustomerRecovered string = "customer_recovered"
```

**Step 3: Add new field constants**

In `bin-customer-manager/models/customer/field.go`, add:

```go
FieldStatus              Field = "status"
FieldTMDeletionScheduled Field = "tm_deletion_scheduled"
```

**Step 4: Update webhook message**

In `bin-customer-manager/models/customer/webhook.go`, add the new fields to `WebhookMessage` struct and `ConvertWebhookMessage()`.

**Step 5: Run verification**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-customer-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-customer-manager: Add Status and TMDeletionScheduled fields to customer model
- bin-customer-manager: Add customer_frozen and customer_recovered event types
- bin-customer-manager: Add field constants for status and tm_deletion_scheduled"
```

---

## Task 3: Customer Handler - Freeze & Recover Logic

**Files:**
- Create: `bin-customer-manager/pkg/customerhandler/freeze.go`
- Create: `bin-customer-manager/pkg/customerhandler/freeze_test.go`
- Modify: `bin-customer-manager/pkg/customerhandler/main.go` (interface)
- Modify: `bin-customer-manager/pkg/dbhandler/main.go` (interface)
- Modify: `bin-customer-manager/pkg/dbhandler/customer.go` (new DB methods)

**Step 1: Add Freeze/Recover to CustomerHandler interface**

In `bin-customer-manager/pkg/customerhandler/main.go`, add to interface:

```go
Freeze(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
Recover(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
```

**Step 2: Add DB methods to DBHandler interface**

In `bin-customer-manager/pkg/dbhandler/main.go`, add:

```go
CustomerFreeze(ctx context.Context, id uuid.UUID) error
CustomerRecover(ctx context.Context, id uuid.UUID) error
CustomerListFrozenExpired(ctx context.Context, expiredBefore time.Time) ([]*customer.Customer, error)
CustomerAnonymizePII(ctx context.Context, id uuid.UUID, anonName, anonEmail string) error
```

**Step 3: Implement DB methods**

In `bin-customer-manager/pkg/dbhandler/customer.go`, add:

`CustomerFreeze`: UPDATE customer_customers SET status='frozen', tm_deletion_scheduled=NOW(), tm_update=NOW() WHERE id=? AND status='active'

`CustomerRecover`: UPDATE customer_customers SET status='active', tm_deletion_scheduled=NULL, tm_update=NOW() WHERE id=? AND status='frozen'

`CustomerListFrozenExpired`: SELECT * FROM customer_customers WHERE status='frozen' AND tm_deletion_scheduled < ? AND tm_delete IS NULL

`CustomerAnonymizePII`: UPDATE customer_customers SET name=?, email=?, phone_number='', address='', webhook_uri='', status='deleted', tm_delete=NOW(), tm_update=NOW() WHERE id=?

Follow the existing pattern in `CustomerDelete` for metrics, squirrel query building, and cache invalidation.

**Step 4: Implement Freeze handler**

In `bin-customer-manager/pkg/customerhandler/freeze.go`:

```go
func (h *customerHandler) Freeze(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
    // 1. Get customer, validate status is active
    // 2. If already frozen, return current state (idempotent)
    // 3. Call h.db.CustomerFreeze(ctx, id)
    // 4. Get updated customer
    // 5. Publish customer_frozen event
    // 6. Return updated customer
}

func (h *customerHandler) Recover(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
    // 1. Get customer, validate status is frozen
    // 2. If not frozen, return error
    // 3. Call h.db.CustomerRecover(ctx, id)
    // 4. Get updated customer
    // 5. Publish customer_recovered event
    // 6. Return updated customer
}
```

**Step 5: Write tests**

In `bin-customer-manager/pkg/customerhandler/freeze_test.go`:
- Test Freeze on active customer → status becomes frozen, event published
- Test Freeze on already-frozen customer → returns current state (idempotent)
- Test Freeze on deleted customer → returns error
- Test Recover on frozen customer → status becomes active, event published
- Test Recover on active customer → returns error

**Step 6: Run verification**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**

```bash
git add bin-customer-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-customer-manager: Implement Freeze and Recover customer handler methods
- bin-customer-manager: Add CustomerFreeze, CustomerRecover, CustomerListFrozenExpired, CustomerAnonymizePII DB methods
- bin-customer-manager: Add tests for freeze and recover logic"
```

---

## Task 4: Customer Handler - Expiry Cron

**Files:**
- Create: `bin-customer-manager/pkg/customerhandler/expiry.go`
- Create: `bin-customer-manager/pkg/customerhandler/expiry_test.go`
- Modify: `bin-customer-manager/pkg/customerhandler/main.go` (interface)
- Modify: `bin-customer-manager/cmd/customer-manager/main.go` (start cron)

**Step 1: Add RunCleanupFrozenExpired to interface**

In `bin-customer-manager/pkg/customerhandler/main.go`, add:

```go
RunCleanupFrozenExpired(ctx context.Context)
```

**Step 2: Implement expiry cron**

In `bin-customer-manager/pkg/customerhandler/expiry.go`:

```go
const (
    expiryCheckInterval = 24 * time.Hour
    gracePeriod         = 30 * 24 * time.Hour // 30 days
)

func (h *customerHandler) RunCleanupFrozenExpired(ctx context.Context) {
    // Same ticker pattern as RunCleanupUnverified in cleanup.go
    // On each tick: call h.cleanupFrozenExpired(ctx)
}

func (h *customerHandler) cleanupFrozenExpired(ctx context.Context) {
    // 1. cutoff := time.Now().Add(-gracePeriod)
    // 2. customers := h.db.CustomerListFrozenExpired(ctx, cutoff)
    // 3. For each customer:
    //    a. Anonymize PII: name → "deleted_user_{short_id}", email → "deleted_{short_id}@removed.voipbin.net"
    //    b. h.db.CustomerAnonymizePII(ctx, id, anonName, anonEmail)
    //    c. Get updated customer
    //    d. Publish customer_deleted event (reuse existing event type)
    //    e. Log each processed customer
}
```

Short ID: use first 8 chars of customer UUID for anonymized identifiers.

**Step 3: Start cron in main.go**

In `bin-customer-manager/cmd/customer-manager/main.go`, add alongside existing cleanup goroutine:

```go
go customerHandler.RunCleanupFrozenExpired(context.Background())
```

**Step 4: Write tests**

In `bin-customer-manager/pkg/customerhandler/expiry_test.go`:
- Test cleanupFrozenExpired with expired frozen customers → PII anonymized, customer_deleted event published
- Test cleanupFrozenExpired with non-expired frozen customers → no action
- Test cleanupFrozenExpired with no frozen customers → no action

**Step 5: Run verification and commit**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
git add bin-customer-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-customer-manager: Add daily cron for frozen account expiry after 30-day grace period
- bin-customer-manager: Implement PII anonymization on expiry
- bin-customer-manager: Reuse customer_deleted event for cascading cleanup on expiry"
```

---

## Task 5: Customer ListenHandler - New RPC Routes

**Files:**
- Modify: `bin-customer-manager/pkg/listenhandler/main.go` (regex + routing)
- Create: `bin-customer-manager/pkg/listenhandler/v1_customers_deletion.go`

**Step 1: Add regex patterns**

In `bin-customer-manager/pkg/listenhandler/main.go`, add:

```go
regV1CustomersIDDeletion = regexp.MustCompile("/v1/customers/" + regUUID + "/deletion$")
regV1CustomersIDRecover  = regexp.MustCompile("/v1/customers/" + regUUID + "/recover$")
```

**Step 2: Add route cases**

In the request routing switch in `main.go`, add:

```go
// POST /v1/customers/<customer-id>/deletion (freeze)
case regV1CustomersIDDeletion.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1CustomersIDDeletionPost(ctx, m)
    requestType = "/v1/customers/deletion"

// POST /v1/customers/<customer-id>/recover
case regV1CustomersIDRecover.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1CustomersIDRecoverPost(ctx, m)
    requestType = "/v1/customers/recover"
```

**Important:** Place these BEFORE the existing `regV1CustomersID` cases to prevent regex conflicts (since `/v1/customers/{id}/deletion` would also match `/v1/customers/{id}` regex if it's checked first).

**Step 3: Implement handlers**

In `bin-customer-manager/pkg/listenhandler/v1_customers_deletion.go`:

```go
func (h *listenHandler) processV1CustomersIDDeletionPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    // Parse customer ID from URI
    // Call h.customerHandler.Freeze(ctx, id)
    // Return JSON response
}

func (h *listenHandler) processV1CustomersIDRecoverPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    // Parse customer ID from URI
    // Call h.customerHandler.Recover(ctx, id)
    // Return JSON response
}
```

Follow the exact pattern of `processV1CustomersIDDelete` in `v1_customers.go`.

**Step 4: Run verification and commit**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
git add bin-customer-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-customer-manager: Add RPC routes for POST /v1/customers/{id}/deletion and POST /v1/customers/{id}/recover
- bin-customer-manager: Implement freeze and recover listenhandler endpoints"
```

---

## Task 6: RequestHandler RPC Methods (bin-common-handler)

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (interface)
- Modify: `bin-common-handler/pkg/requesthandler/customer_customer.go` (implementations)

**Step 1: Add methods to RequestHandler interface**

In `bin-common-handler/pkg/requesthandler/main.go`, add:

```go
CustomerV1CustomerFreeze(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error)
CustomerV1CustomerRecover(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error)
```

**Step 2: Implement RPC methods**

In `bin-common-handler/pkg/requesthandler/customer_customer.go`, add:

```go
func (r *requestHandler) CustomerV1CustomerFreeze(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error) {
    uri := fmt.Sprintf("/v1/customers/%s/deletion", customerID)
    tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPost, "customer/customers/deletion", requestTimeoutDefault, 0, ContentTypeJSON, nil)
    if err != nil {
        return nil, err
    }
    var res cscustomer.Customer
    if errParse := parseResponse(tmp, &res); errParse != nil {
        return nil, errParse
    }
    return &res, nil
}

func (r *requestHandler) CustomerV1CustomerRecover(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error) {
    uri := fmt.Sprintf("/v1/customers/%s/recover", customerID)
    tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPost, "customer/customers/recover", requestTimeoutDefault, 0, ContentTypeJSON, nil)
    if err != nil {
        return nil, err
    }
    var res cscustomer.Customer
    if errParse := parseResponse(tmp, &res); errParse != nil {
        return nil, errParse
    }
    return &res, nil
}
```

Follow the exact pattern of `CustomerV1CustomerGet` and `CustomerV1CustomerDelete`.

**Step 3: Run verification**

```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Update vendor in dependent services**

After changing bin-common-handler, every service that uses it needs vendor update. Focus on the services we're modifying:

```bash
for svc in bin-customer-manager bin-api-manager bin-call-manager bin-billing-manager; do
    cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Customer-account-deletion-design/$svc
    go mod tidy && go mod vendor
done
```

**Step 5: Commit**

```bash
git add bin-common-handler/ bin-customer-manager/vendor/ bin-api-manager/vendor/ bin-call-manager/vendor/ bin-billing-manager/vendor/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-common-handler: Add CustomerV1CustomerFreeze and CustomerV1CustomerRecover RPC methods
- bin-customer-manager: Update vendor
- bin-api-manager: Update vendor
- bin-call-manager: Update vendor
- bin-billing-manager: Update vendor"
```

---

## Task 7: OpenAPI Spec Updates

**Files:**
- Create: `bin-openapi-manager/openapi/paths/customers/id_deletion.yaml`
- Create: `bin-openapi-manager/openapi/paths/customers/id_recover.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add path refs, update CustomerManagerCustomer schema)

**Step 1: Create id_deletion.yaml**

In `bin-openapi-manager/openapi/paths/customers/id_deletion.yaml`:

```yaml
post:
  summary: Schedule customer deletion (freeze account)
  description: |
    Marks the customer for deletion. The account enters 'frozen' state immediately.
    Active calls are terminated and new operations are blocked.
    The customer has 30 days to recover before permanent deletion.
    Admin-only endpoint.
  tags:
    - Customer
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
  responses:
    '200':
      description: Customer account frozen successfully
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/CustomerManagerCustomer'
    '400':
      description: Bad request (customer not in active state)
```

**Step 2: Create id_recover.yaml**

In `bin-openapi-manager/openapi/paths/customers/id_recover.yaml`:

```yaml
post:
  summary: Cancel customer deletion (recover account)
  description: |
    Cancels a scheduled deletion and restores the account to active state.
    Only works during the 30-day grace period.
    Admin-only endpoint.
  tags:
    - Customer
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
  responses:
    '200':
      description: Customer account recovered successfully
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/CustomerManagerCustomer'
    '404':
      description: Customer not in frozen state
```

**Step 3: Update openapi.yaml**

Add path references and update CustomerManagerCustomer schema with new fields:

```yaml
# In paths section:
/v1.0/customers/{id}/deletion:
  $ref: './paths/customers/id_deletion.yaml'
/v1.0/customers/{id}/recover:
  $ref: './paths/customers/id_recover.yaml'

# In CustomerManagerCustomer schema, add:
status:
  type: string
  enum: [active, frozen, deleted]
  description: Account lifecycle status
tm_deletion_scheduled:
  type: string
  format: date-time
  description: When account deletion was requested (null if not scheduled)
```

**Step 3: Regenerate**

```bash
cd bin-openapi-manager && go generate ./...
cd bin-api-manager && go generate ./...
```

**Step 4: Run verification on both**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-openapi-manager: Add /v1/customers/{id}/deletion (POST freeze) and /v1/customers/{id}/recover (POST recover) endpoint schemas
- bin-openapi-manager: Add status and tm_deletion_scheduled fields to CustomerManagerCustomer schema
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

## Task 8: API Manager - Self-Service Auth Endpoints

**Files:**
- Create: `bin-api-manager/lib/service/unregister.go`
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (interface)
- Modify: `bin-api-manager/pkg/servicehandler/customer.go` (implementation)
- Modify: `bin-api-manager/cmd/api-manager/main.go` (route registration)

**Step 1: Add to ServiceHandler interface**

In `bin-api-manager/pkg/servicehandler/main.go`, add:

```go
CustomerFreeze(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
CustomerRecover(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
```

**Step 2: Implement service handler methods**

In `bin-api-manager/pkg/servicehandler/customer.go`, add:

```go
func (h *serviceHandler) CustomerFreeze(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error) {
    // Permission: customer owner (CustomerAdmin) OR project super admin
    // Call h.reqHandler.CustomerV1CustomerFreeze(ctx, customerID)
    // Return res.ConvertWebhookMessage()
}

func (h *serviceHandler) CustomerRecover(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error) {
    // Permission: customer owner (CustomerAdmin) OR project super admin
    // Call h.reqHandler.CustomerV1CustomerRecover(ctx, customerID)
    // Return res.ConvertWebhookMessage()
}
```

**Step 3: Create auth endpoint handlers**

In `bin-api-manager/lib/service/unregister.go`:

```go
type RequestBodyUnregisterPOST struct {
    Password           string `json:"password"`
    ConfirmationPhrase string `json:"confirmation_phrase"`
}

func PostAuthUnregister(c *gin.Context) {
    // 1. This endpoint REQUIRES authentication (unlike login/signup)
    //    Extract agent from middleware context
    // 2. Bind request body
    // 3. Validate: exactly one of password or confirmation_phrase provided
    // 4. If password: validate against stored hash via serviceHandler
    //    If confirmation_phrase: must equal "DELETE"
    // 5. Call serviceHandler.CustomerFreeze(ctx, agent, agent.CustomerID)
    // 6. Return 200 with customer
}

func DeleteAuthUnregister(c *gin.Context) {
    // 1. Extract agent from middleware context
    // 2. No request body needed
    // 3. Call serviceHandler.CustomerRecover(ctx, agent, agent.CustomerID)
    // 4. Return 200 with customer
}
```

**Step 4: Register routes**

In `bin-api-manager/cmd/api-manager/main.go`, the `/auth/unregister` endpoints require authentication unlike other `/auth` routes. Create a separate authenticated auth group:

```go
// Existing unauthenticated auth routes
auth := app.Group("/auth")
auth.POST("/login", service.PostLogin)
// ... existing routes

// New authenticated auth routes (require middleware)
authProtected := app.Group("/auth")
authProtected.Use(middleware.Authenticate())
authProtected.POST("/unregister", service.PostAuthUnregister)
authProtected.DELETE("/unregister", service.DeleteAuthUnregister)
```

**Step 5: Run verification and commit**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-api-manager: Add POST /auth/unregister (freeze) and DELETE /auth/unregister (recover) endpoints
- bin-api-manager: Add CustomerFreeze and CustomerRecover service handler methods
- bin-api-manager: Register authenticated /auth/unregister routes with middleware"
```

---

## Task 9: API Manager - Admin Endpoints

**Files:**
- Modify: `bin-api-manager/server/customers.go`
- Modify: `bin-api-manager/pkg/servicehandler/customer.go` (admin-specific permission)

**Step 1: Implement admin endpoint handlers**

In `bin-api-manager/server/customers.go`, add the generated handler stubs from OpenAPI:

```go
func (h *server) PostCustomersIdDeletion(c *gin.Context, id string) {
    // Extract agent, parse UUID
    // Permission: PermissionProjectSuperAdmin only
    // Call serviceHandler.CustomerFreeze(ctx, agent, customerID)
    // Return 200 with customer
}

func (h *server) PostCustomersIdRecover(c *gin.Context, id string) {
    // Extract agent, parse UUID
    // Permission: PermissionProjectSuperAdmin only
    // Call serviceHandler.CustomerRecover(ctx, agent, customerID)
    // Return 200 with customer
}
```

Follow the exact pattern of `DeleteCustomersId` in the same file.

**Step 2: Run verification and commit**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-api-manager: Add POST /v1/customers/{id}/deletion and POST /v1/customers/{id}/recover admin endpoint handlers
- bin-api-manager: Admin endpoints require PermissionProjectSuperAdmin"
```

---

## Task 10: API Manager - Frozen Account Check

**Files:**
- Modify: `bin-api-manager/lib/middleware/authenticate.go` (or create new middleware)

**Step 1: Add frozen status check to authentication middleware**

After successful authentication in `authenticate.go`, check if the customer's status is frozen. If so, and the request is NOT to an allowed endpoint, return 403.

```go
// After extracting agent, check customer frozen status
// Allowed endpoints when frozen:
//   - DELETE /auth/unregister (recovery)
//   - POST /v1.0/customers/{id}/recover (admin recovery)
//   - GET /v1.0/customers/{id} (view status)
//   - GET /auth/* and POST /auth/login (authentication)
//
// For all other endpoints, if customer status == "frozen":
//   return 403 with DELETION_SCHEDULED error body
```

The check should fetch customer status via `reqHandler.CustomerV1CustomerGet()` and cache the result for the duration of the request. Check if customer.Status == "frozen".

**Important:** Consider performance. This adds an RPC call per request. Options:
- Cache frozen customer IDs in Redis with short TTL (recommended)
- Check only on the customer model already available in the agent context
- Add the Status field to the agent/token claims

Check how the agent struct is populated — if `agent.CustomerID` is available, we can fetch the customer once per request.

**Step 2: Define error response**

```go
type DeletionScheduledError struct {
    Error               string `json:"error"`
    Message             string `json:"message"`
    DeletionScheduledAt string `json:"deletion_scheduled_at"`
    DeletionEffectiveAt string `json:"deletion_effective_at"`
    RecoveryEndpoint    string `json:"recovery_endpoint"`
}
```

**Step 3: Run verification and commit**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-api-manager: Add frozen account check in authentication middleware
- bin-api-manager: Return 403 DELETION_SCHEDULED for frozen accounts on non-allowed endpoints"
```

---

## Task 11: Call Manager - Frozen Event Handler

**Files:**
- Modify: `bin-call-manager/pkg/subscribehandler/main.go` (event routing)
- Modify: `bin-call-manager/pkg/subscribehandler/customer_manager.go` (event handler)
- Modify: `bin-call-manager/pkg/callhandler/main.go` (interface)
- Modify: `bin-call-manager/pkg/callhandler/event.go` (implementation)

**Step 1: Add event routing**

In `bin-call-manager/pkg/subscribehandler/main.go`, add case to `processEvent` switch:

```go
case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cucustomer.EventTypeCustomerFrozen:
    err = h.processEventCUCustomerFrozen(ctx, m)
```

**Step 2: Add event handler**

In `bin-call-manager/pkg/subscribehandler/customer_manager.go`, add:

```go
func (h *subscribeHandler) processEventCUCustomerFrozen(ctx context.Context, m *sock.Event) error {
    // Unmarshal customer from event data
    // Call h.callHandler.EventCUCustomerFrozen(ctx, cu)
    // Also call h.groupcallHandler and h.confbridgeHandler if they have the method
}
```

**Step 3: Add to CallHandler interface**

In `bin-call-manager/pkg/callhandler/main.go`, add:

```go
EventCUCustomerFrozen(ctx context.Context, cu *cucustomer.Customer) error
```

**Step 4: Implement hangup-all for frozen customer**

In `bin-call-manager/pkg/callhandler/event.go`, add:

```go
func (h *callHandler) EventCUCustomerFrozen(ctx context.Context, cu *cucustomer.Customer) error {
    // Same pattern as EventCUCustomerDeleted, but use HangingUp instead of Delete:
    // 1. List all active calls for customer_id
    // 2. For each call: h.HangingUp(ctx, call.ID, call.HangupReasonNormal)
    // 3. Log each hangup, continue on errors
}
```

**Step 5: Run verification and commit**

```bash
cd bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
git add bin-call-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-call-manager: Subscribe to customer_frozen event
- bin-call-manager: Hangup all active calls when customer is frozen"
```

---

## Task 12: Call Manager - Frozen Call Rejection

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/validate.go`
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go`
- Modify: `bin-call-manager/pkg/callhandler/start.go`

**Step 1: Add frozen validation function**

In `bin-call-manager/pkg/callhandler/validate.go`, add:

```go
func (h *callHandler) ValidateCustomerNotFrozen(ctx context.Context, customerID uuid.UUID) bool {
    log := logrus.WithFields(logrus.Fields{
        "func":        "ValidateCustomerNotFrozen",
        "customer_id": customerID,
    })

    cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
    if err != nil {
        log.Errorf("Could not get customer info. err: %v", err)
        return false
    }
    log.WithField("customer", cu).Debugf("Retrieved customer info. customer_id: %s", cu.ID)

    if cu.Status == cucustomer.StatusFrozen {
        log.Infof("Customer account is frozen. Rejecting call.")
        return false
    }

    return true
}
```

**Step 2: Add check to outgoing calls**

In `bin-call-manager/pkg/callhandler/outgoing_call.go`, add frozen check BEFORE the balance validation (around line 133):

```go
// validate customer is not frozen
if !h.ValidateCustomerNotFrozen(ctx, customerID) {
    log.Infof("Customer account is frozen. Rejecting outgoing call. customer_id: %s", customerID)
    return nil, fmt.Errorf("customer account is frozen")
}
```

**Step 3: Add check to incoming calls**

In `bin-call-manager/pkg/callhandler/start.go`, add frozen check BEFORE the balance validation in `startCallTypeFlow` (around line 549):

```go
// validate customer is not frozen
if !h.ValidateCustomerNotFrozen(ctx, customerID) {
    log.Errorf("Customer account is frozen. Rejecting incoming call. customer_id: %s", customerID)
    _, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
    return
}
```

**Step 4: Run verification and commit**

```bash
cd bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
git add bin-call-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-call-manager: Add ValidateCustomerNotFrozen check
- bin-call-manager: Reject outgoing calls for frozen customers at call setup
- bin-call-manager: Reject incoming calls for frozen customers at call setup"
```

---

## Task 13: Billing Manager - Frozen/Recovered Events

**Files:**
- Modify: `bin-billing-manager/pkg/subscribehandler/main.go` (event routing)
- Modify: `bin-billing-manager/pkg/subscribehandler/customer.go` (event handlers)
- Modify: `bin-billing-manager/pkg/accounthandler/main.go` (interface)
- Modify: `bin-billing-manager/pkg/accounthandler/event.go` (implementation)

**Step 1: Add event routing**

In `bin-billing-manager/pkg/subscribehandler/main.go`, add cases to `processEvent` switch:

```go
case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cucustomer.EventTypeCustomerFrozen:
    err = h.processEventCUCustomerFrozen(ctx, m)

case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cucustomer.EventTypeCustomerRecovered:
    err = h.processEventCUCustomerRecovered(ctx, m)
```

**Step 2: Add event handlers**

In `bin-billing-manager/pkg/subscribehandler/customer.go`, add:

```go
func (h *subscribeHandler) processEventCUCustomerFrozen(ctx context.Context, m *sock.Event) error {
    // Unmarshal customer
    // Call h.accountHandler.EventCUCustomerFrozen(ctx, cu)
}

func (h *subscribeHandler) processEventCUCustomerRecovered(ctx context.Context, m *sock.Event) error {
    // Unmarshal customer
    // Call h.accountHandler.EventCUCustomerRecovered(ctx, cu)
}
```

**Step 3: Add to AccountHandler interface**

In `bin-billing-manager/pkg/accounthandler/main.go`, add:

```go
EventCUCustomerFrozen(ctx context.Context, cu *cucustomer.Customer) error
EventCUCustomerRecovered(ctx context.Context, cu *cucustomer.Customer) error
```

**Step 4: Implement account freeze/unfreeze**

In `bin-billing-manager/pkg/accounthandler/event.go`, add:

```go
func (h *accountHandler) EventCUCustomerFrozen(ctx context.Context, cu *cucustomer.Customer) error {
    // 1. List all accounts for customer_id (same pattern as EventCUCustomerDeleted)
    // 2. For each account: soft-delete (sets tm_delete) to block new billing
    //    This reuses existing AccountDelete which sets tm_delete
    // 3. Log results
}

func (h *accountHandler) EventCUCustomerRecovered(ctx context.Context, cu *cucustomer.Customer) error {
    // 1. List all accounts for customer_id WITH deleted=true filter
    // 2. For each account: restore by clearing tm_delete
    //    Use AccountUpdate with FieldTMDelete set to nil/zero
    // 3. Log results
}
```

**Note:** For the freeze, reusing the existing soft-delete on billing accounts is pragmatic — the billing system already checks `tm_delete` before creating charges. On recovery, we reverse by clearing `tm_delete`. This requires checking that account restore logic works correctly with the existing `AccountUpdate` method.

**Step 5: Run verification and commit**

```bash
cd bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
git add bin-billing-manager/
git commit -m "NOJIRA-Customer-account-deletion-design

- bin-billing-manager: Subscribe to customer_frozen and customer_recovered events
- bin-billing-manager: Freeze billing accounts on customer_frozen (soft-delete to block new charges)
- bin-billing-manager: Restore billing accounts on customer_recovered (clear tm_delete)"
```

---

## Task 14: Final Integration Verification

**Step 1: Run verification on all modified services**

```bash
for svc in bin-customer-manager bin-api-manager bin-call-manager bin-billing-manager bin-openapi-manager bin-common-handler; do
    echo "=== Verifying $svc ==="
    cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Customer-account-deletion-design/$svc
    go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
done
```

**Step 2: Review all changes**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Customer-account-deletion-design
git log --oneline main..HEAD
git diff --stat main..HEAD
```

**Step 3: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-Customer-account-deletion-design
gh pr create --title "NOJIRA-Customer-account-deletion-design" --body "$(cat <<'EOF'
Add graceful customer account deletion (unregister) with 30-day recovery period
and three-layer enforcement to prevent billing leakage.

- bin-dbscheme-manager: Add status and tm_deletion_scheduled columns to customer_customers table
- bin-customer-manager: Add Status/TMDeletionScheduled fields, Freeze/Recover handlers, expiry cron
- bin-customer-manager: Add customer_frozen and customer_recovered event types
- bin-customer-manager: Add RPC routes for POST/DELETE /v1/customers/{id}/deletion
- bin-common-handler: Add CustomerV1CustomerFreeze and CustomerV1CustomerRecover RPC methods
- bin-openapi-manager: Add /v1/customers/{id}/deletion and /v1/customers/{id}/recover endpoint schemas
- bin-api-manager: Add POST/DELETE /auth/unregister self-service endpoints
- bin-api-manager: Add POST /v1/customers/{id}/deletion and POST /v1/customers/{id}/recover admin endpoints
- bin-api-manager: Add frozen account middleware (403 DELETION_SCHEDULED)
- bin-call-manager: Subscribe to customer_frozen, hangup active calls, reject new calls
- bin-billing-manager: Subscribe to customer_frozen/recovered, freeze/restore billing accounts
EOF
)"
```

---

## Task Dependency Graph

```
Task 1 (DB Migration) ─────────────────────────────────────┐
                                                            │
Task 2 (Customer Model) ───┬── Task 3 (Freeze/Recover) ──┬─┤
                           │                               │ │
                           ├── Task 4 (Expiry Cron) ───────┤ │
                           │                               │ │
                           ├── Task 5 (ListenHandler) ─────┤ │
                           │                               │ │
                           └── Task 6 (RequestHandler) ──┬─┤ │
                                                         │ │ │
Task 7 (OpenAPI) ───────────────────────────────────────┬┤ │ │
                                                        ││ │ │
Task 8 (Auth Endpoints) ←── Tasks 6, 7 ────────────────┘│ │ │
Task 9 (Admin Endpoints) ←── Tasks 6, 7 ────────────────┘│ │
Task 10 (Frozen Check) ←── Task 6 ───────────────────────┘│ │
                                                           │ │
Task 11 (Call Events) ←── Task 2 ─────────────────────────┘ │
Task 12 (Call Rejection) ←── Task 11 ───────────────────────┤
Task 13 (Billing Events) ←── Task 2 ────────────────────────┘
                                                             │
Task 14 (Integration) ←── All tasks ─────────────────────────┘
```

**Parallelizable groups:**
- After Task 2: Tasks 3+4+5, Task 6, Task 7, Task 11, Task 13 can run in parallel
- After Task 6+7: Tasks 8, 9, 10 can run in parallel
- Task 12 requires Task 11
- Task 14 requires all others
