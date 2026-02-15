# Fix Allowance Sentinel and Add Current Endpoint - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the billing_allowances sentinel/NULL inconsistency that prevents allowance creation, and add a GET endpoint for the current allowance cycle.

**Architecture:** New Alembic migration aligns billing_allowances with the NULL-based soft-delete convention. One code fix in dbhandler. New single-resource endpoint follows existing list-endpoint patterns across billing-manager, common-handler, openapi-manager, and api-manager.

**Tech Stack:** Go, Alembic (Python), MySQL, RabbitMQ RPC, OpenAPI 3.0

---

### Task 1: Create Alembic migration to fix billing_allowances tm_delete

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f7_billing_allowances_fix_tm_delete_sentinel.py`

**Step 1: Create migration file**

```python
"""billing_allowances_fix_tm_delete_sentinel

Revision ID: a1b2c3d4e5f7
Revises: fd3b4c5d6e7f
Create Date: 2026-02-15 18:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f7'
down_revision = 'fd3b4c5d6e7f'
branch_labels = None
depends_on = None

SENTINEL = '9999-01-01 00:00:00.000000'


def upgrade():
    # Make tm_delete nullable (currently NOT NULL with sentinel default)
    op.execute("ALTER TABLE `billing_allowances` MODIFY `tm_delete` DATETIME(6) NULL")

    # Convert existing sentinel values to NULL
    op.execute(f"""
        UPDATE `billing_allowances`
        SET `tm_delete` = NULL
        WHERE `tm_delete` = '{SENTINEL}'
    """)


def downgrade():
    # Restore sentinel values from NULL
    op.execute(f"""
        UPDATE `billing_allowances`
        SET `tm_delete` = '{SENTINEL}'
        WHERE `tm_delete` IS NULL
    """)

    # Restore NOT NULL constraint with sentinel default
    op.execute(
        f"ALTER TABLE `billing_allowances` MODIFY `tm_delete` DATETIME(6) NOT NULL DEFAULT '{SENTINEL}'"
    )
```

**Step 2: Verify the file is syntactically correct**

Run: `python3 -c "import ast; ast.parse(open('bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f7_billing_allowances_fix_tm_delete_sentinel.py').read()); print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f7_billing_allowances_fix_tm_delete_sentinel.py
git commit -m "NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint

- bin-dbscheme-manager: Fix billing_allowances tm_delete sentinel to NULL"
```

---

### Task 2: Fix AllowanceGetCurrentByAccountID sentinel query

**Files:**
- Modify: `bin-billing-manager/pkg/dbhandler/allowance.go` (line 102)

**Step 1: Fix the query**

In `AllowanceGetCurrentByAccountID`, change the sentinel comparison to NULL check:

```go
// Before (line 102):
Where(sq.Eq{"tm_delete": "9999-01-01 00:00:00.000000"}).

// After:
Where(sq.Eq{"tm_delete": nil}).
```

Also remove the now-obsolete comment at lines 90-92 about sentinel values:

```go
// Before (lines 90-92):
// NOTE: The billing_allowances table uses a sentinel value ('9999-01-01 00:00:00.000000')
// for tm_delete (NOT NULL with DEFAULT), unlike billing_billings which uses NULL.
// This difference comes from the table schema — query patterns must match each table's design.

// After: remove these 3 comment lines entirely
```

**Step 2: Run tests**

Run: `cd bin-billing-manager && go test ./pkg/dbhandler/...`
Expected: PASS

**Step 3: Run full verification**

Run: `cd bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 4: Commit**

```bash
git add bin-billing-manager/pkg/dbhandler/allowance.go
git commit -m "NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint

- bin-billing-manager: Fix AllowanceGetCurrentByAccountID to use NULL instead of sentinel"
```

---

### Task 3: Add current-allowance listenhandler endpoint in billing-manager

**Files:**
- Modify: `bin-billing-manager/pkg/listenhandler/main.go` (add route regex and case)
- Create: `bin-billing-manager/pkg/listenhandler/v1_allowance.go` (singular)

**Step 1: Add the route regex in main.go**

After the existing `regV1AccountsIDAllowances` line (line 62), add:

```go
regV1AccountsIDAllowance  = regexp.MustCompile(`/v1/accounts/` + regUUID + `/allowance$`)
```

**Step 2: Add the case in processRequest switch**

After the existing allowances case (line 211), add:

```go
// GET /accounts/<account-id>/allowance
case regV1AccountsIDAllowance.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
	response, err = h.processV1AccountsIDAllowanceGet(ctx, m)
	requestType = "/v1/accounts/<account-id>/allowance"
```

**Step 3: Create handler file v1_allowance.go**

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1AccountsIDAllowanceGet handles GET /v1/accounts/<account-id>/allowance request
func (h *listenHandler) processV1AccountsIDAllowanceGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDAllowanceGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	accountID := uuid.FromStringOrNil(uriItems[3])

	a, err := h.allowanceHandler.GetCurrentCycle(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get current allowance. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(a)
	if err != nil {
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
```

**Step 4: Run full verification**

Run: `cd bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 5: Commit**

```bash
git add bin-billing-manager/pkg/listenhandler/main.go bin-billing-manager/pkg/listenhandler/v1_allowance.go
git commit -m "NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint

- bin-billing-manager: Add GET /v1/accounts/{id}/allowance endpoint for current cycle"
```

---

### Task 4: Add RPC method in bin-common-handler requesthandler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (interface, ~line 397)
- Modify: `bin-common-handler/pkg/requesthandler/billing_accounts.go` (new method)

**Step 1: Add interface method**

After `BillingV1AccountAllowancesGet` (line 397), add:

```go
BillingV1AccountAllowanceGet(ctx context.Context, accountID uuid.UUID) (*bmallowance.Allowance, error)
```

**Step 2: Add implementation in billing_accounts.go**

After the existing `BillingV1AccountAllowancesGet` function (after line 251), add:

```go
// BillingV1AccountAllowanceGet returns the current allowance cycle for the given billing account.
func (r *requestHandler) BillingV1AccountAllowanceGet(ctx context.Context, accountID uuid.UUID) (*bmallowance.Allowance, error) {
	uri := fmt.Sprintf("/v1/accounts/%s/allowance", accountID)

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/accounts/<account-id>/allowance", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res bmallowance.Allowance
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 3: Run full verification**

Run: `cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 4: Commit**

```bash
git add bin-common-handler/pkg/requesthandler/main.go bin-common-handler/pkg/requesthandler/billing_accounts.go
git commit -m "NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint

- bin-common-handler: Add BillingV1AccountAllowanceGet RPC method"
```

---

### Task 5: Add OpenAPI spec for current allowance endpoint

**Files:**
- Create: `bin-openapi-manager/openapi/paths/billing_accounts/id_allowance.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add path reference)

**Step 1: Create path spec file**

```yaml
get:
  summary: Get current billing account allowance
  description: Returns the current active token allowance cycle for the given billing account. Returns 404 if no cycle exists.
  tags:
    - Billing
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
      description: The ID of the billing account.
  responses:
    200:
      description: Successfully retrieved current allowance cycle
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/BillingManagerAllowance'
    404:
      description: No active allowance cycle found
```

**Step 2: Add path reference in openapi.yaml**

After the existing `/billing_accounts/{id}/allowances:` line (~line 4122), add:

```yaml
  /billing_accounts/{id}/allowance:
    $ref: './paths/billing_accounts/id_allowance.yaml'
```

**Step 3: Regenerate OpenAPI types**

Run: `cd bin-openapi-manager && go generate ./...`
Expected: Types regenerated successfully

**Step 4: Run full verification for openapi-manager**

Run: `cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/paths/billing_accounts/id_allowance.yaml bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint

- bin-openapi-manager: Add OpenAPI spec for GET /billing_accounts/{id}/allowance"
```

---

### Task 6: Add api-manager handler for current allowance endpoint

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (interface, ~line 158)
- Modify: `bin-api-manager/pkg/servicehandler/billingaccount.go` (new method)
- Modify: `bin-api-manager/server/billing_accounts.go` (new HTTP handler)

**Step 1: Regenerate api-manager server code**

Run: `cd bin-api-manager && go generate ./...`

This generates the new `GetBillingAccountsIdAllowance` stub in `gens/openapi_server/gen.go`.

**Step 2: Add ServiceHandler interface method**

After `BillingAccountAllowancesGet` in `pkg/servicehandler/main.go` (line 158), add:

```go
BillingAccountAllowanceGet(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmallowance.WebhookMessage, error)
```

**Step 3: Add service handler implementation**

After `BillingAccountAllowancesGet` in `pkg/servicehandler/billingaccount.go` (after line 216), add:

```go
// BillingAccountAllowanceGet returns the current allowance cycle for the given billing account.
func (h *serviceHandler) BillingAccountAllowanceGet(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmallowance.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountAllowanceGet",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"billing_account_id": billingAccountID,
	})

	// get billing account to validate ownership
	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.BillingV1AccountAllowanceGet(ctx, billingAccountID)
	if err != nil {
		log.Errorf("Could not get current allowance. err: %v", err)
		return nil, errors.Wrap(err, "could not get current allowance")
	}
	log.WithField("allowance", tmp).Debugf("Retrieved current allowance. allowance_id: %s", tmp.ID)

	return tmp.ConvertWebhookMessage(), nil
}
```

**Step 4: Add HTTP handler in server/billing_accounts.go**

After `GetBillingAccountsIdAllowances` function (after line 66), add:

```go
func (h *server) GetBillingAccountsIdAllowance(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillingAccountsIdAllowance",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.BillingAccountAllowanceGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get current allowance. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

**Step 5: Vendor the updated common-handler**

Run: `cd bin-api-manager && go mod tidy && go mod vendor`

**Step 6: Regenerate mocks and run full verification**

Run: `cd bin-api-manager && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 7: Commit**

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint

- bin-api-manager: Add GET /billing_accounts/{id}/allowance handler"
```

---

### Task 7: Re-vendor common-handler in billing-manager and final verification

**Files:**
- Modify: `bin-billing-manager/vendor/` (re-vendor to pick up new requesthandler mock)

**Step 1: Re-vendor billing-manager**

The mock for requesthandler was regenerated in Task 4 via `go generate`. Billing-manager needs to pick up the updated mock.

Run: `cd bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 2: Commit if vendor changed**

```bash
git add bin-billing-manager/vendor/ bin-billing-manager/go.sum
git commit -m "NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint

- bin-billing-manager: Re-vendor to pick up updated common-handler"
```

---

### Task 8: Pull main, check conflicts, push and create PR

**Step 1: Fetch latest main and check for conflicts**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

If conflicts: rebase, resolve, re-run verification.
If clean: proceed.

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint
```

PR title: `NOJIRA-Fix-allowance-sentinel-and-add-current-endpoint`

PR body:
```
Fix billing_allowances sentinel/NULL inconsistency that prevented allowance cycle
creation for existing customers, and add endpoint to retrieve current allowance cycle.

- bin-dbscheme-manager: Fix billing_allowances tm_delete from NOT NULL sentinel to nullable NULL
- bin-billing-manager: Fix AllowanceGetCurrentByAccountID to use NULL instead of sentinel
- bin-billing-manager: Add GET /v1/accounts/{id}/allowance endpoint for current cycle
- bin-common-handler: Add BillingV1AccountAllowanceGet RPC method
- bin-openapi-manager: Add OpenAPI spec for GET /billing_accounts/{id}/allowance
- bin-api-manager: Add handler for current allowance endpoint
```
