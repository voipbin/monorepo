# Restrict /customers API Permissions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restrict all `/customers` (plural) admin endpoints to ProjectSuperAdmin only, while keeping `/customer` (singular) self-service endpoints at their appropriate permission levels.

**Architecture:** Currently, the self-service `/customer` routes and admin `/customers/{id}` routes share the same servicehandler functions (`CustomerGet`, `CustomerUpdate`, `CustomerUpdateBillingAccountID`). Since these two route groups now need different permission levels, we split them: the existing functions become SuperAdmin-only for the admin routes, and new `CustomerSelf*` methods handle the self-service routes with their own permission checks.

**Tech Stack:** Go, gomock

---

## Target Access Matrix

| Endpoint | Agent | Manager | Admin | Super Admin |
|---|---|---|---|---|
| `GET /customer` (own) | - | yes | yes | yes |
| `PUT /customer` (own) | - | - | yes | yes |
| `PUT /customer/billing_account_id` (own) | - | - | yes | yes |
| `GET /service_agents/customer` (own) | yes | yes | yes | yes |
| `POST /customers` | - | - | - | yes |
| `GET /customers` | - | - | - | yes |
| `GET /customers/{id}` | - | - | - | yes |
| `PUT /customers/{id}` | - | - | - | yes |
| `DELETE /customers/{id}` | - | - | - | yes |
| `PUT /customers/{id}/billing_account_id` | - | - | - | yes |
| `POST /customers/{id}/freeze` | - | - | - | yes |
| `POST /customers/{id}/recover` | - | - | - | yes |

## Files to Change

- `bin-api-manager/pkg/servicehandler/main.go` — Add 3 new methods to ServiceHandler interface
- `bin-api-manager/pkg/servicehandler/customer.go` — Change 3 existing permission checks to SuperAdmin; add 3 new CustomerSelf* methods
- `bin-api-manager/pkg/servicehandler/customer_test.go` — Update 3 existing tests (SuperAdmin agents); add 3 new self-service tests
- `bin-api-manager/server/customer.go` — Update 3 route handlers to call CustomerSelf* methods
- `bin-api-manager/server/customer_test.go` — Update mock expectations to call CustomerSelf* methods

---

### Task 1: Change CustomerGet to SuperAdmin-only and create CustomerSelfGet

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/customer.go:82`
- Modify: `bin-api-manager/pkg/servicehandler/customer_test.go:118-179`

**Step 1: Update CustomerGet permission check**

In `customer.go`, change line 82 from:
```go
if !h.hasPermission(ctx, a, tmp.ID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
```
to:
```go
if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
```

**Step 2: Update TestCustomerGet to use SuperAdmin agent**

In `customer_test.go`, change the test agent from `PermissionCustomerAdmin` to `PermissionProjectSuperAdmin` and remove the CustomerID match (since SuperAdmin bypasses ownership check).

**Step 3: Create CustomerSelfGet method**

Add to `customer.go`:
```go
func (h *serviceHandler) CustomerSelfGet(ctx context.Context, a *amagent.Agent) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfGet",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}
```

**Step 4: Add CustomerSelfGet to ServiceHandler interface**

In `main.go`, add after `CustomerGet`:
```go
CustomerSelfGet(ctx context.Context, a *amagent.Agent) (*cscustomer.WebhookMessage, error)
```

**Step 5: Write test for CustomerSelfGet**

Add to `customer_test.go`:
```go
func Test_CustomerSelfGet(t *testing.T) {
	tests := []test with PermissionCustomerAdmin agent, calling CustomerSelfGet
```

**Step 6: Update server/customer.go GetCustomer route**

Change `server/customer.go` `GetCustomer` from:
```go
res, err := h.serviceHandler.CustomerGet(c.Request.Context(), &a, a.CustomerID)
```
to:
```go
res, err := h.serviceHandler.CustomerSelfGet(c.Request.Context(), &a)
```

**Step 7: Update server/customer_test.go**

Update `Test_customerGET` mock expectation from `CustomerGet` to `CustomerSelfGet`.

**Step 8: Run tests**

```bash
cd bin-api-manager && go generate ./... && go test ./pkg/servicehandler/... ./server/...
```

---

### Task 2: Change CustomerUpdate to SuperAdmin-only and create CustomerSelfUpdate

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/customer.go:169`
- Modify: `bin-api-manager/pkg/servicehandler/customer_test.go:257-333`

**Step 1: Update CustomerUpdate permission check**

In `customer.go`, change line 169 from:
```go
if !h.hasPermission(ctx, a, c.ID, amagent.PermissionCustomerAdmin) {
```
to:
```go
if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
```

**Step 2: Update Test_CustomerUpdate to use SuperAdmin agent**

Change the test agent from `PermissionCustomerAdmin` to `PermissionProjectSuperAdmin`.

**Step 3: Create CustomerSelfUpdate method**

Add to `customer.go`:
```go
func (h *serviceHandler) CustomerSelfUpdate(
	ctx context.Context,
	a *amagent.Agent,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfUpdate",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdate(ctx, a.CustomerID, name, detail, email, phoneNumber, address, webhookMethod, webhookURI)
	if err != nil {
		log.Errorf("Could not update the customer's basic info. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}
```

**Step 4: Add CustomerSelfUpdate to ServiceHandler interface**

**Step 5: Write test for CustomerSelfUpdate**

**Step 6: Update server/customer.go PutCustomer route to call CustomerSelfUpdate**

**Step 7: Update server/customer_test.go Test_customerPut**

**Step 8: Run tests**

---

### Task 3: Change CustomerUpdateBillingAccountID to SuperAdmin-only and create CustomerSelfUpdateBillingAccountID

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/customer.go:351,362`
- Modify: `bin-api-manager/pkg/servicehandler/customer_test.go:398-472`

**Step 1: Update CustomerUpdateBillingAccountID permission checks**

Change both `hasPermission` calls (lines 351, 362) from `PermissionCustomerAdmin` to `PermissionProjectSuperAdmin` with `uuid.Nil`.

**Step 2: Update Test_CustomerUpdateBillingAccountID to use SuperAdmin agent**

**Step 3: Create CustomerSelfUpdateBillingAccountID method**

```go
func (h *serviceHandler) CustomerSelfUpdateBillingAccountID(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "CustomerSelfUpdateBillingAccountID",
		"customer_id":        a.CustomerID,
		"billing_account_id": billingAccountID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Errorf("Could not validate the billing account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdateBillingAccountID(ctx, a.CustomerID, billingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer's billing account. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}
```

**Step 4: Add to interface, write test, update server route, update server test**

**Step 5: Run tests**

---

### Task 4: Regenerate mocks and run full verification

**Step 1: Regenerate mocks**

```bash
cd bin-api-manager && go generate ./...
```

**Step 2: Run full verification workflow**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Commit**

```bash
git add -A
git commit -m "NOJIRA-restrict-customers-api-permissions

Restrict all /customers (plural) admin endpoints to ProjectSuperAdmin only.
Create CustomerSelf* methods for /customer (singular) self-service routes.

- bin-api-manager: Change CustomerGet permission to ProjectSuperAdmin
- bin-api-manager: Change CustomerUpdate permission to ProjectSuperAdmin
- bin-api-manager: Change CustomerUpdateBillingAccountID permission to ProjectSuperAdmin
- bin-api-manager: Add CustomerSelfGet (Manager+Admin) for GET /customer
- bin-api-manager: Add CustomerSelfUpdate (Admin) for PUT /customer
- bin-api-manager: Add CustomerSelfUpdateBillingAccountID (Admin) for PUT /customer/billing_account_id
- bin-api-manager: Update server routes to use new self-service methods
- bin-api-manager: Update tests for new permission requirements
"
```
