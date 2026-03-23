# Split Billing Account API Permissions — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Split billing account APIs into singular `/billing_account` (customer admin) and plural `/billing_accounts` (project admin), add list endpoint, add admin OpenAPI schema.

**Architecture:** Follow existing `/customer` vs `/customers` pattern. Singular endpoints auto-resolve billing account from `agent.CustomerID` → customer's `BillingAccountID`. Add `BillingV1AccountGets` RPC for list. Plural endpoints return raw `Account` model via new `BillingManagerAccountAdmin` schema.

**Tech Stack:** Go, RabbitMQ RPC, OpenAPI 3.0, oapi-codegen

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions`

**Design doc:** `docs/plans/2026-03-23-split-billing-account-api-permissions-design.md`

---

### Task 1: Add account list handler to bin-billing-manager

The billing-manager already has `AccountList()` in dbhandler and `List()` in accounthandler. We need to expose it via RPC.

**Files:**
- Modify: `bin-billing-manager/pkg/listenhandler/main.go` (add regex + route)
- Modify: `bin-billing-manager/pkg/listenhandler/v1_accounts.go` (add handler)

**Step 1: Add regex pattern in `main.go`**

Add after line 57 (after `regV1AccountsIsValidResourceLimitByCustomerID`):

```go
regV1AccountsGet = regexp.MustCompile(`/v1/accounts\?`)
```

**Step 2: Add route in `processRequest()` switch**

Add BEFORE the `regV1AccountsID` case (before line 157) — order matters because `regV1AccountsID` would also match list URIs. Place the new route at the top of the accounts section:

```go
// GET /accounts?<filters>
case regV1AccountsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
	response, err = h.processV1AccountsGet(ctx, m)
	requestType = "/v1/accounts"
```

**Step 3: Add `processV1AccountsGet` handler in `v1_accounts.go`**

Follow the exact pattern from `processV1BillingsGet` in `v1_billings.go`:

```go
// processV1AccountsGet handles GET /v1/accounts?<filters> request
func (h *listenHandler) processV1AccountsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[account.FieldStruct, account.Field](account.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	as, err := h.accountHandler.List(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get accounts info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(as)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
```

Add required imports to `v1_accounts.go`: `"net/url"`, `"strconv"`, `utilhandler "monorepo/bin-common-handler/pkg/utilhandler"`.

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-billing-manager/
git commit -m "NOJIRA-Split-billing-account-api-permissions

- bin-billing-manager: Add processV1AccountsGet list handler for accounts
- bin-billing-manager: Add regex route for GET /v1/accounts"
```

---

### Task 2: Add `BillingV1AccountGets` RPC method to bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/billing_accounts.go` (add method)
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (add to interface, after line 427)

**Step 1: Add `BillingV1AccountGets` to the `RequestHandler` interface in `main.go`**

Add after `BillingV1AccountUpdatePaymentInfo` (line 427):

```go
BillingV1AccountGets(ctx context.Context, pageToken string, pageSize uint64, filters map[bmaccount.Field]any) ([]bmaccount.Account, error)
```

**Step 2: Implement `BillingV1AccountGets` in `billing_accounts.go`**

Follow the `CustomerV1CustomerList` pattern:

```go
// BillingV1AccountGets returns a list of billing accounts.
func (r *requestHandler) BillingV1AccountGets(ctx context.Context, pageToken string, pageSize uint64, filters map[bmaccount.Field]any) ([]bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, fmt.Errorf("could not marshal filters. err: %w", err)
	}

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/accounts", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
```

Add `"net/url"` to imports if not already present.

**Step 3: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Note: `go generate ./...` will regenerate `mock_main.go` to include the new interface method.

**Step 4: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Split-billing-account-api-permissions

- bin-common-handler: Add BillingV1AccountGets RPC method for listing billing accounts
- bin-common-handler: Add interface definition to RequestHandler"
```

---

### Task 3: Add OpenAPI path definitions and admin schema

**Files:**
- Create: `bin-openapi-manager/openapi/paths/billing_account/main.yaml`
- Create: `bin-openapi-manager/openapi/paths/billing_account/payment_info.yaml`
- Create: `bin-openapi-manager/openapi/paths/billing_accounts/main.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add path refs + admin schema + status enum)
- Modify: `bin-openapi-manager/openapi/paths/billing_accounts/id.yaml` (update response schema to admin)
- Modify: `bin-openapi-manager/openapi/paths/billing_accounts/id_payment_info.yaml` (update response schema to admin)

**Step 1: Create `billing_account/main.yaml`** (singular — customer admin)

Follow `paths/customer/main.yaml` pattern. No path parameters — auto-resolved from auth.

```yaml
get:
  summary: Get billing account info
  description: Retrieve the billing account of the authenticated customer. The billing account is automatically resolved from the authenticated user's customer record.
  tags:
    - Billing
  responses:
    '200':
      description: The billing account information.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/BillingManagerAccount'

put:
  summary: Update billing account
  description: Update the billing account name and detail of the authenticated customer.
  tags:
    - Billing
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          properties:
            name:
              type: string
              description: The display name of the billing account.
              example: "Production Account"
            detail:
              type: string
              description: A human-readable note describing the purpose of this account.
              example: "Main billing account for production services"
  responses:
    '200':
      description: Successfully updated billing account.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/BillingManagerAccount'
```

**Step 2: Create `billing_account/payment_info.yaml`**

```yaml
put:
  summary: Update billing account payment info
  description: Update the payment type and method of the authenticated customer's billing account.
  tags:
    - Billing
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          properties:
            payment_type:
              description: The type of payment for the account.
              example: "prepaid"
              $ref: '#/components/schemas/BillingManagerAccountPaymentType'
            payment_method:
              description: The method of payment for the account.
              example: "credit card"
              $ref: '#/components/schemas/BillingManagerAccountPaymentMethod'
  responses:
    '200':
      description: Successfully updated billing account payment info.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/BillingManagerAccount'
```

**Step 3: Create `billing_accounts/main.yaml`** (plural — list for project admin)

Follow the standard list pattern with pagination. Uses `BillingManagerAccountAdmin` schema.

```yaml
get:
  summary: Get list of billing accounts
  description: Returns a list of all billing accounts. Requires project super admin permission.
  tags:
    - Billing
  parameters:
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
  responses:
    '200':
      description: Successfully retrieved billing accounts list.
      content:
        application/json:
          schema:
            type: object
            properties:
              next_page_token:
                type: string
                description: Token for the next page of results.
                example: "2026-01-15T09:30:00.000000Z"
              result:
                type: array
                items:
                  $ref: '#/components/schemas/BillingManagerAccountAdmin'
```

**Step 4: Update `billing_accounts/id.yaml` response schema**

Change both GET and PUT `200` response schemas from `BillingManagerAccount` to `BillingManagerAccountAdmin`:

```yaml
# In GET 200 response
schema:
  $ref: '#/components/schemas/BillingManagerAccountAdmin'

# In PUT 200 response
schema:
  $ref: '#/components/schemas/BillingManagerAccountAdmin'
```

**Step 5: Update `billing_accounts/id_payment_info.yaml` response schema**

Change PUT `200` response schema from `BillingManagerAccount` to `BillingManagerAccountAdmin`.

**Step 6: Add `BillingManagerAccountStatus` enum and `BillingManagerAccountAdmin` schema to `openapi.yaml`**

Add after the existing `BillingManagerAccount` schema (after line 481):

```yaml
    BillingManagerAccountStatus:
      type: string
      description: The status of the billing account.
      example: "active"
      enum:
        - active
        - frozen
        - deleted
      x-enum-varnames:
        - BillingManagerAccountStatusActive
        - BillingManagerAccountStatusFrozen
        - BillingManagerAccountStatusDeleted

    BillingManagerAccountAdmin:
      type: object
      description: Internal billing account representation for project admins. Includes all fields including status.
      properties:
        id:
          type: string
          format: uuid
          x-go-type: string
          description: The unique identifier of the account.
          example: "b8c9d0e1-f2a3-4567-8901-23456789abcd"
        customer_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the associated customer. Returned from the `GET /customers` response."
          example: "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"
        status:
          description: The status of the billing account.
          example: "active"
          $ref: '#/components/schemas/BillingManagerAccountStatus'
        name:
          type: string
          description: The display name of the billing account.
          example: "Production Account"
        detail:
          type: string
          description: A human-readable note describing the purpose of this account.
          example: "Main billing account for production services"
        plan_type:
          description: The plan tier of the billing account.
          example: "basic"
          $ref: '#/components/schemas/BillingManagerAccountPlanType'
        balance_credit:
          type: integer
          format: int64
          description: The credit balance of the account in micros (1 USD = 1,000,000).
          example: 1500000
        balance_token:
          type: integer
          format: int64
          description: The token balance of the account.
          example: 500
        payment_type:
          description: The type of payment associated with the account.
          example: "prepaid"
          $ref: '#/components/schemas/BillingManagerAccountPaymentType'
        payment_method:
          description: The method of payment used for the account.
          example: "credit card"
          $ref: '#/components/schemas/BillingManagerAccountPaymentMethod'
        paddle_subscription_id:
          type: string
          description: "The Paddle subscription identifier for this billing account."
          example: "sub_01h8bxq9f3e4t5a6g7h8j9k0"
        paddle_customer_id:
          type: string
          description: "The Paddle customer identifier for this billing account."
          example: "ctm_01h8bxq9f3e4t5a6g7h8j9k0"
        tm_last_topup:
          type: string
          format: date-time
          x-go-type: string
          description: The timestamp of the last token top-up.
          example: "2026-01-15T09:30:00.000000Z"
        tm_next_topup:
          type: string
          format: date-time
          x-go-type: string
          description: The timestamp of the next scheduled token top-up.
          example: "2026-01-15T09:30:00.000000Z"
        tm_create:
          type: string
          format: date-time
          x-go-type: string
          description: The timestamp when the account was created.
          example: "2026-01-15T09:30:00.000000Z"
        tm_update:
          type: string
          format: date-time
          x-go-type: string
          description: The timestamp when the account was last updated.
          example: "2026-01-15T09:30:00.000000Z"
        tm_delete:
          type: string
          format: date-time
          x-go-type: string
          description: The timestamp when the account was deleted, if applicable.
          example: "2026-01-15T09:30:00.000000Z"
```

**Step 7: Add path references to `openapi.yaml`**

In the `paths:` section (before line 6688, before existing `/billing_accounts/{id}`), add:

```yaml
  /billing_account:
    $ref: './paths/billing_account/main.yaml'
  /billing_account/payment_info:
    $ref: './paths/billing_account/payment_info.yaml'

  /billing_accounts:
    $ref: './paths/billing_accounts/main.yaml'
```

**Step 8: Regenerate and verify**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 9: Commit**

```bash
git add bin-openapi-manager/
git commit -m "NOJIRA-Split-billing-account-api-permissions

- bin-openapi-manager: Add singular /billing_account path definitions (GET, PUT, PUT payment_info)
- bin-openapi-manager: Add /billing_accounts list endpoint path definition
- bin-openapi-manager: Add BillingManagerAccountAdmin schema and BillingManagerAccountStatus enum
- bin-openapi-manager: Update plural /billing_accounts/{id} endpoints to use admin schema
- bin-openapi-manager: Add path references to openapi.yaml"
```

---

### Task 4: Add singular endpoint handlers, list handler, and tighten permissions in bin-api-manager

**Files:**
- Create: `bin-api-manager/server/billing_account.go` (singular handlers)
- Modify: `bin-api-manager/server/billing_accounts.go` (add list handler)
- Modify: `bin-api-manager/pkg/servicehandler/billingaccount.go` (add Self* methods, List, tighten permissions, change return types)
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (update interface)

**Step 1: Update the `ServiceHandler` interface in `main.go`**

Update existing methods and add new ones. At line 153-157, change:

```go
// Existing (change return types from *WebhookMessage to *Account)
BillingAccountGet(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmaccount.Account, error)
BillingAccountUpdateBasicInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, name string, detail string) (*bmaccount.Account, error)
BillingAccountUpdatePaymentInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.Account, error)

// New methods
BillingAccountSelfGet(ctx context.Context, a *amagent.Agent) (*bmaccount.WebhookMessage, error)
BillingAccountSelfUpdateBasicInfo(ctx context.Context, a *amagent.Agent, name string, detail string) (*bmaccount.WebhookMessage, error)
BillingAccountSelfUpdatePaymentInfo(ctx context.Context, a *amagent.Agent, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error)
BillingAccountList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*bmaccount.Account, error)
```

**Step 2: Add self-methods to `pkg/servicehandler/billingaccount.go`**

Follow the `CustomerSelfGet` / `CustomerSelfUpdate` pattern. Key: resolve billing account ID from agent's customer record.

```go
// BillingAccountSelfGet returns the authenticated agent's own billing account.
func (h *serviceHandler) BillingAccountSelfGet(ctx context.Context, a *amagent.Agent) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountSelfGet",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get customer to resolve billing account ID
	c, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	if c.BillingAccountID == uuid.Nil {
		log.Info("Customer has no billing account.")
		return nil, fmt.Errorf("no billing account")
	}

	ba, err := h.billingAccountGet(ctx, c.BillingAccountID)
	if err != nil {
		log.Errorf("Could not get the billing account info. err: %v", err)
		return nil, err
	}
	log.WithField("billing_account", ba).Debugf("Retrieved billing account info. billing_account_id: %s", ba.ID)

	return ba.ConvertWebhookMessage(), nil
}

// BillingAccountSelfUpdateBasicInfo updates the authenticated agent's own billing account's basic info.
func (h *serviceHandler) BillingAccountSelfUpdateBasicInfo(ctx context.Context, a *amagent.Agent, name string, detail string) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountSelfUpdateBasicInfo",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	c, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	if c.BillingAccountID == uuid.Nil {
		log.Info("Customer has no billing account.")
		return nil, fmt.Errorf("no billing account")
	}

	tmp, err := h.reqHandler.BillingV1AccountUpdateBasicInfo(ctx, c.BillingAccountID, name, detail)
	if err != nil {
		log.Infof("Could not update account info. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// BillingAccountSelfUpdatePaymentInfo updates the authenticated agent's own billing account's payment info.
func (h *serviceHandler) BillingAccountSelfUpdatePaymentInfo(ctx context.Context, a *amagent.Agent, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountSelfUpdatePaymentInfo",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	c, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	if c.BillingAccountID == uuid.Nil {
		log.Info("Customer has no billing account.")
		return nil, fmt.Errorf("no billing account")
	}

	tmp, err := h.reqHandler.BillingV1AccountUpdatePaymentInfo(ctx, c.BillingAccountID, paymentType, paymentMethod)
	if err != nil {
		log.Infof("Could not update account payment info. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// BillingAccountList returns a list of all billing accounts.
func (h *serviceHandler) BillingAccountList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "BillingAccountList",
		"agent":   a,
		"size":    size,
		"token":   token,
		"filters": filters,
	})
	log.Debug("Received request detail.")

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if size <= 0 {
		size = 10
	}
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertBillingAccountFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.BillingV1AccountGets(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get billing accounts info. err: %v", err)
		return nil, err
	}

	res := make([]*bmaccount.Account, len(tmps))
	for i := range tmps {
		res[i] = &tmps[i]
	}

	return res, nil
}

// convertBillingAccountFilters converts map[string]string to map[bmaccount.Field]any
func (h *serviceHandler) convertBillingAccountFilters(filters map[string]string) (map[bmaccount.Field]any, error) {
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, bmaccount.Account{})
	if err != nil {
		return nil, err
	}

	result := make(map[bmaccount.Field]any, len(typed))
	for k, v := range typed {
		result[bmaccount.Field(k)] = v
	}

	return result, nil
}
```

Imports to add if not present: `commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"`.

**Step 3: Tighten permissions on existing plural methods**

In `billingaccount.go`, update `BillingAccountGet`:
- Change return type from `*bmaccount.WebhookMessage` to `*bmaccount.Account`
- Change permission: `h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin)` → `h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin)`
- Remove `.ConvertWebhookMessage()` call — return `ba` directly

For `BillingAccountUpdateBasicInfo`: Same permission + return type change. Return `tmp` directly instead of `tmp.ConvertWebhookMessage()`.

For `BillingAccountUpdatePaymentInfo`: Same permission + return type change. Return `tmp` directly instead of `tmp.ConvertWebhookMessage()`.

The `BillingAccountAddBalanceForce` and `BillingAccountSubtractBalanceForce` methods also need return type changes from `*bmaccount.WebhookMessage` to `*bmaccount.Account` for consistency (they already check `PermissionProjectSuperAdmin`). Remove `.ConvertWebhookMessage()` from these too.

**Step 4: Create `server/billing_account.go`** (singular endpoint handlers)

Follow `server/customer.go` pattern:

```go
package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	bmaccount "monorepo/bin-billing-manager/models/account"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetBillingAccount(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillingAccount",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithField("agent", a)

	res, err := h.serviceHandler.BillingAccountSelfGet(c.Request.Context(), &a)
	if err != nil {
		log.Infof("Could not get the billing account info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutBillingAccount(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutBillingAccount",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithField("agent", a)

	var req openapi_server.PutBillingAccountJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}

	res, err := h.serviceHandler.BillingAccountSelfUpdateBasicInfo(c.Request.Context(), &a, name, detail)
	if err != nil {
		log.Errorf("Could not update. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutBillingAccountPaymentInfo(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutBillingAccountPaymentInfo",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithField("agent", a)

	var req openapi_server.PutBillingAccountPaymentInfoJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	paymentType := bmaccount.PaymentTypeNone
	if req.PaymentType != nil {
		paymentType = bmaccount.PaymentType(*req.PaymentType)
	}

	paymentMethod := bmaccount.PaymentMethodNone
	if req.PaymentMethod != nil {
		paymentMethod = bmaccount.PaymentMethod(*req.PaymentMethod)
	}

	res, err := h.serviceHandler.BillingAccountSelfUpdatePaymentInfo(c.Request.Context(), &a, paymentType, paymentMethod)
	if err != nil {
		log.Errorf("Could not update payment info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

Note: The generated type names (`PutBillingAccountJSONBody`, `PutBillingAccountPaymentInfoJSONBody`) depend on what oapi-codegen generates. After `go generate`, verify the type names match and adjust if needed.

**Step 5: Add `GetBillingAccounts` list handler to `server/billing_accounts.go`**

Follow `GetCustomers` pattern in `customers.go`:

```go
func (h *server) GetBillingAccounts(c *gin.Context, params openapi_server.GetBillingAccountsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillingAccounts",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	filters := map[string]string{
		"deleted": "false",
	}

	tmps, err := h.serviceHandler.BillingAccountList(c.Request.Context(), &a, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get billing accounts list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}
```

**Step 6: Regenerate and verify bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Note: `go generate` will regenerate `gens/openapi_server/gen.go` from the updated OpenAPI spec. The new generated types must match what the handlers use. If the generated type names differ, adjust the handler code to match.

**Step 7: Commit**

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Split-billing-account-api-permissions

- bin-api-manager: Add singular /billing_account endpoint handlers (GET, PUT, PUT payment_info)
- bin-api-manager: Add /billing_accounts list endpoint handler
- bin-api-manager: Add BillingAccountSelf* service handler methods for customer admin
- bin-api-manager: Add BillingAccountList service handler method for project admin
- bin-api-manager: Tighten existing /billing_accounts/{id} endpoints to project admin only
- bin-api-manager: Change plural endpoint return types from WebhookMessage to Account"
```

---

### Task 5: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/billing_account_overview.rst` (or create if missing)

**Step 1: Check existing billing account RST docs**

```bash
ls ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-api-manager/docsdev/source/billing_account*
```

**Step 2: Update the billing account overview RST**

Document the new singular `/billing_account` endpoints for customer admin users:
- `GET https://api.voipbin.net/v1.0/billing_account` — retrieve authenticated customer's billing account
- `PUT https://api.voipbin.net/v1.0/billing_account` — update name/detail
- `PUT https://api.voipbin.net/v1.0/billing_account/payment_info` — update payment type/method

Note that `/billing_accounts/{id}` (plural) is now admin-only and should NOT appear in user-facing docs. Use `/billing_account` (singular) in all customer-facing documentation.

Follow RST writing guidelines from `bin-api-manager/CLAUDE.md`:
- Use fully qualified URLs
- Include AI Context block
- Add AI Implementation Hint
- Document troubleshooting

**Step 3: Rebuild HTML**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-api-manager/docsdev
rm -rf build && python3 -m sphinx -M html source build
```

**Step 4: Stage and commit**

```bash
git add bin-api-manager/docsdev/source/billing_account*
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Split-billing-account-api-permissions

- bin-api-manager: Update RST docs for new /billing_account singular endpoints
- bin-api-manager: Rebuild HTML documentation"
```

---

### Task 6: Final verification across all changed services

Run the full verification workflow for each changed service.

**Step 1: Verify bin-billing-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify bin-common-handler**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Verify bin-openapi-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Check for impact on other services that import bin-common-handler**

The new `BillingV1AccountGets` method is an addition to the `RequestHandler` interface. Since `go generate` produces a mock, all services that use the mock will need regeneration. Run:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions
grep -rl "MockRequestHandler" --include="*.go" | grep -v vendor | grep -v bin-common-handler | head -20
```

For each service that uses `MockRequestHandler` in tests, run `go generate ./...` and `go test ./...` to ensure the mock is updated and tests pass. The interface change is additive (new method only), so existing tests should compile without code changes — only the mock files need regeneration.

---

### Task 7: Push and create PR

**Step 1: Fetch latest main and check for conflicts**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Split-billing-account-api-permissions
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 2: Push**

```bash
git push -u origin NOJIRA-Split-billing-account-api-permissions
```

**Step 3: Create PR**

Title: `NOJIRA-Split-billing-account-api-permissions`

Body:
```
Split billing account APIs into singular /billing_account for customer admins
and plural /billing_accounts for project super admins. Add list endpoint for
project admins. Plural endpoints now return raw Account model with status field
via new BillingManagerAccountAdmin schema.

- bin-billing-manager: Add processV1AccountsGet list handler for accounts
- bin-common-handler: Add BillingV1AccountGets RPC method for listing billing accounts
- bin-openapi-manager: Add singular /billing_account path definitions
- bin-openapi-manager: Add /billing_accounts list endpoint path definition
- bin-openapi-manager: Add BillingManagerAccountAdmin schema and BillingManagerAccountStatus enum
- bin-openapi-manager: Update plural endpoints to use admin schema
- bin-api-manager: Add singular /billing_account endpoint handlers (GET, PUT, PUT payment_info)
- bin-api-manager: Add /billing_accounts list endpoint handler
- bin-api-manager: Add BillingAccountSelf* service handler methods for customer admin
- bin-api-manager: Tighten existing /billing_accounts/{id} to project admin only
- bin-api-manager: Change plural endpoint return types from WebhookMessage to Account
- bin-api-manager: Update RST documentation for new singular endpoints
```
