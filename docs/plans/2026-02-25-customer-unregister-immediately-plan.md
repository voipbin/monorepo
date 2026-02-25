# Customer Unregister Immediately — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `immediate` boolean to `POST /auth/unregister` so customers can skip the 30-day grace period and delete their account immediately.

**Architecture:** New `FreezeAndDelete` method in bin-customer-manager reuses existing `Freeze()` + anonymization logic from `cleanupFrozenExpired`. Exposed via new RPC endpoint, called from bin-api-manager when `immediate: true`.

**Tech Stack:** Go, RabbitMQ RPC, OpenAPI 3.0, Sphinx RST docs

---

### Task 1: Add FreezeAndDelete to CustomerHandler Interface (bin-customer-manager)

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/main.go:37` (interface)

**Step 1: Add FreezeAndDelete to CustomerHandler interface**

In `bin-customer-manager/pkg/customerhandler/main.go`, add after the `Freeze` method (line 37):

```go
FreezeAndDelete(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
```

The interface block should have these customer lifecycle methods together:

```go
Delete(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
Freeze(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
FreezeAndDelete(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
```

**Step 2: Verify it compiles (expect failure — method not implemented yet)**

Run: `cd bin-customer-manager && go build ./...`
Expected: FAIL with "customerHandler does not implement CustomerHandler (missing FreezeAndDelete method)"

---

### Task 2: Implement FreezeAndDelete Method (bin-customer-manager)

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/freeze.go` (add method after Recover)

**Step 1: Add FreezeAndDelete method**

Append to `bin-customer-manager/pkg/customerhandler/freeze.go` after the `Recover` method:

```go
// FreezeAndDelete freezes the customer account and immediately deletes it.
// This skips the 30-day grace period by combining freeze + PII anonymization + deletion event.
func (h *customerHandler) FreezeAndDelete(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FreezeAndDelete",
		"customer_id": id,
	})
	log.Debug("Freezing and deleting the customer account immediately.")

	// Step 1: Freeze (idempotent — handles already-frozen)
	c, err := h.Freeze(ctx, id)
	if err != nil {
		log.Errorf("Could not freeze the customer. err: %v", err)
		return nil, err
	}

	// Step 2: Guard — if already deleted, return early (idempotent)
	if c.Status == customer.StatusDeleted || c.TMDelete != nil {
		log.Infof("Customer already deleted. customer_id: %s", c.ID)
		return c, nil
	}

	// Step 3: Anonymize PII (same logic as cleanupFrozenExpired)
	shortID := c.ID.String()[:8]
	anonName := fmt.Sprintf("deleted_user_%s", shortID)
	anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)

	if err := h.db.CustomerAnonymizePII(ctx, c.ID, anonName, anonEmail); err != nil {
		log.Errorf("Could not anonymize customer PII. customer_id: %s, err: %v", c.ID, err)
		return nil, err
	}

	// Step 4: Fetch updated customer after anonymization
	res, err := h.db.CustomerGet(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not get anonymized customer. customer_id: %s, err: %v", c.ID, err)
		return nil, fmt.Errorf("could not get anonymized customer")
	}

	// Step 5: Publish customer_deleted event (cascades all resource cleanup)
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerDeleted, res)
	log.Infof("Customer frozen and deleted immediately. customer_id: %s", c.ID)

	return res, nil
}
```

**Step 2: Verify it compiles**

Run: `cd bin-customer-manager && go build ./...`
Expected: PASS

---

### Task 3: Add Listenhandler Route for FreezeAndDelete (bin-customer-manager)

**Files:**
- Modify: `bin-customer-manager/pkg/listenhandler/main.go:63-64` (add regex)
- Modify: `bin-customer-manager/pkg/listenhandler/main.go:186-193` (add route case)
- Modify: `bin-customer-manager/pkg/listenhandler/v1_customers_freeze.go` (add handler function)

**Step 1: Add regex in main.go**

In `bin-customer-manager/pkg/listenhandler/main.go`, add after `regV1CustomersIDRecover` (line 64):

```go
regV1CustomersIDFreezeAndDelete = regexp.MustCompile("/v1/customers/" + regUUID + "/freeze_and_delete$")
```

**Step 2: Add route case in main.go**

In the `switch` block, add after the `/recover` case (after line 193):

```go
	// POST /customers/<customer-id>/freeze_and_delete
	case regV1CustomersIDFreezeAndDelete.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CustomersIDFreezeAndDeletePost(ctx, m)
		requestType = "/v1/customers/freeze_and_delete"
```

**IMPORTANT:** This case must come BEFORE the `regV1CustomersID` catch-all cases (GET/PUT/DELETE on line 196+). Place it right after the `/recover` case.

**Step 3: Add handler function in v1_customers_freeze.go**

Append to `bin-customer-manager/pkg/listenhandler/v1_customers_freeze.go`:

```go
// processV1CustomersIDFreezeAndDeletePost handles POST /v1/customers/<customer-id>/freeze_and_delete
func (h *listenHandler) processV1CustomersIDFreezeAndDeletePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":        "processV1CustomersIDFreezeAndDeletePost",
		"customer_id": id,
	})
	log.Debug("Executing processV1CustomersIDFreezeAndDeletePost.")

	tmp, err := h.customerHandler.FreezeAndDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not freeze and delete the customer. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
```

**Step 4: Regenerate mocks and verify**

Run: `cd bin-customer-manager && go generate ./... && go build ./...`
Expected: PASS

---

### Task 4: Add RPC Method to RequestHandler (bin-common-handler)

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:672` (interface — add after CustomerV1CustomerFreeze)
- Modify: `bin-common-handler/pkg/requesthandler/customer_customer.go:304` (add method after CustomerV1CustomerFreeze)

**Step 1: Add to RequestHandler interface**

In `bin-common-handler/pkg/requesthandler/main.go`, add after `CustomerV1CustomerFreeze` (line 672):

```go
CustomerV1CustomerFreezeAndDelete(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error)
```

**Step 2: Add implementation**

In `bin-common-handler/pkg/requesthandler/customer_customer.go`, add after the `CustomerV1CustomerFreeze` method (after line 304):

```go
// CustomerV1CustomerFreezeAndDelete sends the request to freeze and immediately delete the customer
func (r *requestHandler) CustomerV1CustomerFreezeAndDelete(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s/freeze_and_delete", customerID)

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPost, "customer/customers/<customer-id>/freeze_and_delete", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

**Step 3: Regenerate mocks and verify**

Run: `cd bin-common-handler && go generate ./... && go build ./...`
Expected: PASS

---

### Task 5: Vendor Updated bin-common-handler in Dependent Services

**Files:**
- Modify: `bin-customer-manager/vendor/` (re-vendor)
- Modify: `bin-api-manager/vendor/` (re-vendor)

**Step 1: Re-vendor bin-customer-manager**

Run: `cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./...`
Expected: PASS (picks up new mock for RequestHandler)

**Step 2: Re-vendor bin-api-manager**

Run: `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./...`
Expected: PASS

---

### Task 6: Add CustomerSelfFreezeAndDelete to ServiceHandler (bin-api-manager)

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go:479` (interface — add after CustomerSelfFreeze)
- Modify: `bin-api-manager/pkg/servicehandler/customer.go:364` (add method after CustomerSelfFreeze)

**Step 1: Add to ServiceHandler interface**

In `bin-api-manager/pkg/servicehandler/main.go`, add after `CustomerSelfFreeze` (line 478):

```go
CustomerSelfFreezeAndDelete(ctx context.Context, a *amagent.Agent) (*cscustomer.WebhookMessage, error)
```

**Step 2: Add implementation**

In `bin-api-manager/pkg/servicehandler/customer.go`, add after `CustomerSelfFreeze` method (after line 364):

```go
// CustomerSelfFreezeAndDelete handles self-service immediate account deletion.
// Freezes and immediately deletes the account (skips 30-day grace period).
// Requires CustomerAdmin permission.
func (h *serviceHandler) CustomerSelfFreezeAndDelete(ctx context.Context, a *amagent.Agent) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfFreezeAndDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	_, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.CustomerV1CustomerFreezeAndDelete(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not freeze and delete the customer. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}
```

**Step 3: Regenerate mocks and verify**

Run: `cd bin-api-manager && go generate ./... && go build ./...`
Expected: PASS

---

### Task 7: Update PostAuthUnregister Handler (bin-api-manager)

**Files:**
- Modify: `bin-api-manager/lib/service/unregister.go:13-16` (struct), `bin-api-manager/lib/service/unregister.go:70-77` (branching)

**Step 1: Add Immediate field to request struct**

In `bin-api-manager/lib/service/unregister.go`, update the struct (line 13-16):

```go
// RequestBodyUnregisterPOST is request body for POST /auth/unregister
type RequestBodyUnregisterPOST struct {
	Password           string `json:"password"`
	ConfirmationPhrase string `json:"confirmation_phrase"`
	Immediate          bool   `json:"immediate"`
}
```

**Step 2: Add branching logic**

Replace lines 70-77 (the `CustomerSelfFreeze` call and response) with:

```go
	var res *cscustomer.WebhookMessage
	if req.Immediate {
		res, err = serviceHandler.CustomerSelfFreezeAndDelete(c.Request.Context(), &a)
		if err != nil {
			log.Errorf("Could not freeze and delete the customer. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	} else {
		res, err = serviceHandler.CustomerSelfFreeze(c.Request.Context(), &a)
		if err != nil {
			log.Errorf("Could not freeze the customer. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	}

	c.JSON(200, res)
```

Note: You need to add `var err error` before the credential validation block or restructure. Check the existing code — `err` is already used in the `if hasPassword` block's `:=` scope but not declared at the function level. Add `var err error` after the request body bind, and change the inner `:=` to `=` where needed. Actually, looking at the existing code, the `err` in the `serviceHandler.AuthLogin` call uses `:=` inside the `if` block, so it's scoped. The new code needs its own `err` — use `:=` in the `if/else` or declare `var err error` at function level.

Simplest approach — keep `err` scoped within the branching block as shown above (`:=` is not used, `err` is fresh in the `if req.Immediate` block via `=`). Declare `var err error` before the branching:

The full handler after credential validation becomes:

```go
	var (
		res *cscustomer.WebhookMessage
		err error
	)
	if req.Immediate {
		res, err = serviceHandler.CustomerSelfFreezeAndDelete(c.Request.Context(), &a)
		if err != nil {
			log.Errorf("Could not freeze and delete the customer. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	} else {
		res, err = serviceHandler.CustomerSelfFreeze(c.Request.Context(), &a)
		if err != nil {
			log.Errorf("Could not freeze the customer. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	}

	c.JSON(200, res)
```

You'll need to add the import for `cscustomer "monorepo/bin-customer-manager/models/customer"` at the top of the file.

**Step 3: Verify build**

Run: `cd bin-api-manager && go build ./...`
Expected: PASS

---

### Task 8: Write Tests for CustomerSelfFreezeAndDelete (bin-api-manager)

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/customer_test.go` (append test)

**Step 1: Add test**

Append to `bin-api-manager/pkg/servicehandler/customer_test.go`:

```go
func Test_CustomerSelfFreezeAndDelete(t *testing.T) {
	tests := []struct {
		name string

		agent *amagent.Agent

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
				Status: cscustomer.StatusDeleted,
			},
			expectRes: &cscustomer.WebhookMessage{
				ID:     uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
				Status: cscustomer.StatusDeleted,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().CustomerV1CustomerFreezeAndDelete(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)

			res, err := h.CustomerSelfFreezeAndDelete(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
```

**Step 2: Run tests**

Run: `cd bin-api-manager && go test ./pkg/servicehandler/ -run Test_CustomerSelfFreezeAndDelete -v`
Expected: PASS

---

### Task 9: Run Full Verification on bin-customer-manager

**Step 1: Run verification workflow**

Run:
```bash
cd bin-customer-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: ALL PASS

---

### Task 10: Run Full Verification on bin-api-manager

**Step 1: Run verification workflow**

Run:
```bash
cd bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: ALL PASS

---

### Task 11: Update OpenAPI Spec (bin-openapi-manager)

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml:6185-6198` (schema)
- Modify: `bin-openapi-manager/openapi/paths/auth/unregister.yaml:1-11` (description)

**Step 1: Add immediate field to schema**

In `bin-openapi-manager/openapi/openapi.yaml`, in the `RequestBodyAuthUnregisterPOST` schema (after `confirmation_phrase` property, around line 6198), add:

```yaml
        immediate:
          type: boolean
          description: "If true, skip the 30-day grace period and delete the account immediately. The account is frozen and then permanently deleted (PII anonymized, all resources cascade-deleted) in a single request. Default: false."
          example: false
```

**Step 2: Update POST /auth/unregister description**

In `bin-openapi-manager/openapi/paths/auth/unregister.yaml`, update the `description` field (lines 3-6) to:

```yaml
  description: |
    Marks the authenticated customer's account for deletion. The account enters 'frozen' state immediately.
    Active calls are terminated and new operations are blocked.
    The customer has 30 days to recover via `DELETE /auth/unregister` before permanent deletion.

    If `immediate` is set to `true`, the account is frozen and then permanently deleted in a single request,
    skipping the 30-day grace period. PII is anonymized and all resources (agents, numbers, flows, etc.)
    are cascade-deleted. This action cannot be undone.

    Exactly one of `password` or `confirmation_phrase` must be provided:
    - Password-based accounts: provide `password` for re-authentication.
    - SSO or API-key authenticated requests: provide `confirmation_phrase` set to `"DELETE"`.
```

**Step 3: Regenerate and verify**

Run:
```bash
cd bin-openapi-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: ALL PASS

**Step 4: Re-vendor bin-api-manager (picks up new generated types)**

Run:
```bash
cd bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: ALL PASS

---

### Task 12: Update RST Documentation — Customer Struct (bin-api-manager)

**Files:**
- Rewrite: `bin-api-manager/docsdev/source/customer_struct_customer.rst`

**Step 1: Rewrite to match WebhookMessage fields**

Replace the entire content of `bin-api-manager/docsdev/source/customer_struct_customer.rst` with content that matches the `WebhookMessage` struct fields exactly:

Fields to document (from `bin-customer-manager/models/customer/webhook.go`):
- `id` (UUID)
- `name` (String, Optional)
- `detail` (String, Optional)
- `email` (String, Optional)
- `phone_number` (String, Optional)
- `address` (String, Optional)
- `webhook_method` (enum string, Optional): POST, GET, PUT, DELETE
- `webhook_uri` (String, Optional)
- `billing_account_id` (UUID, Optional): Obtained from `GET /billing-accounts`
- `email_verified` (Boolean)
- `status` (enum string): `initial`, `active`, `frozen`, `deleted`, `expired`
- `tm_deletion_scheduled` (ISO 8601, nullable): Set when account is frozen
- `tm_create` (ISO 8601)
- `tm_update` (ISO 8601)
- `tm_delete` (ISO 8601, nullable)

Fields to REMOVE (not in WebhookMessage): `username`, `line_secret`, `line_token`, `permission_ids`

Follow the AI-Native RST Writing Guidelines from `bin-api-manager/CLAUDE.md`:
- Every field has explicit type
- Every ID field states its source endpoint
- Status enum lists all values with descriptions
- Include AI Implementation Hint

---

### Task 13: Update RST Documentation — Customer Overview (bin-api-manager)

**Files:**
- Modify: `bin-api-manager/docsdev/source/customer_overview.rst`

**Step 1: Add Account Deletion Lifecycle section**

Add a new section after "Managing Customers" that documents:

1. **Account Status Lifecycle** — diagram showing: `initial → active → frozen → deleted` and `initial → expired`
2. **Status enum** with all values: `initial` (pending email verification), `active` (normal operation), `frozen` (deletion scheduled, 30-day grace), `deleted` (permanently deleted, PII anonymized), `expired` (unverified signup expired)
3. **Self-Service Unregistration** — `POST /auth/unregister` freezes account, `DELETE /auth/unregister` recovers
4. **Immediate Deletion** — `POST /auth/unregister` with `immediate: true` skips grace period
5. **What happens on freeze** — active calls terminated
6. **What happens on delete** — all resources cascade-deleted (agents, numbers, flows, queues, etc.)

Update the "Customer Properties" table to match WebhookMessage fields (remove `username`, `line_secret`, `line_token`, `permission_ids`; add `email`, `status`, `tm_deletion_scheduled`, etc.)

---

### Task 14: Update RST Documentation — Customer Tutorial (bin-api-manager)

**Files:**
- Modify: `bin-api-manager/docsdev/source/customer_tutorial.rst`

**Step 1: Add unregistration tutorials**

Add sections with curl examples:

**Unregister account (schedule deletion):**
```
POST https://api.voipbin.net/auth/unregister
Body: {"password": "yourPassword"}
```

**Unregister account immediately:**
```
POST https://api.voipbin.net/auth/unregister
Body: {"confirmation_phrase": "DELETE", "immediate": true}
```

**Cancel unregistration:**
```
DELETE https://api.voipbin.net/auth/unregister
```

Include response examples showing the customer object with `status: "frozen"` and `status: "deleted"`.

---

### Task 15: Rebuild RST HTML and Verify

**Step 1: Clean rebuild**

Run:
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```
Expected: Build succeeds with no errors

**Step 2: Verify locally**

Open `bin-api-manager/docsdev/build/html/customer_overview.html` and check the new sections render correctly.

---

### Task 16: Update Root CLAUDE.md with WebhookMessage Rule

**Files:**
- Modify: `CLAUDE.md` (root, line 647-673)

**Step 1: Add WebhookMessage rule to "Feature Changes Require RST Documentation Updates" section**

After the existing "When updating RST docs:" list (line 665), add:

```markdown
**RST struct docs must match `WebhookMessage`, not internal model structs.**
The `WebhookMessage` struct (defined in `models/<entity>/webhook.go`) determines exactly which fields are exposed to external users via the API. RST struct documentation (`*_struct_*.rst`) must only include fields present in `WebhookMessage`. Do not document internal-only fields (e.g., `PodID`, `Username`, `PermissionIDs`) that are stripped by `ConvertWebhookMessage()`. When verifying RST accuracy, always compare against `WebhookMessage` fields, not the internal model struct.
```

**Step 2: Add RST check reminder to "Regular Code Changes Workflow" section**

After the verification workflow code block (around line 48-50), add:

```markdown
**After making user-facing changes**, also verify RST docs in `bin-api-manager/docsdev/source/` are in sync with the code. Compare struct docs against the relevant `WebhookMessage` fields (in `models/<entity>/webhook.go`), not the internal model struct. If RST updates are needed, rebuild HTML: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build` and force-add: `git add -f bin-api-manager/docsdev/build/`.
```

---

### Task 17: Commit and Push

**Step 1: Run final verification on all changed services**

Run these in sequence:
```bash
cd bin-common-handler && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: ALL PASS for all 4 services

**Step 2: Stage and commit**

```bash
git add -f bin-api-manager/docsdev/build/
git add bin-common-handler/pkg/requesthandler/customer_customer.go
git add bin-common-handler/pkg/requesthandler/main.go
git add bin-common-handler/pkg/requesthandler/mock_main.go
git add bin-customer-manager/pkg/customerhandler/freeze.go
git add bin-customer-manager/pkg/customerhandler/main.go
git add bin-customer-manager/pkg/customerhandler/mock_main.go
git add bin-customer-manager/pkg/listenhandler/main.go
git add bin-customer-manager/pkg/listenhandler/v1_customers_freeze.go
git add bin-api-manager/lib/service/unregister.go
git add bin-api-manager/pkg/servicehandler/customer.go
git add bin-api-manager/pkg/servicehandler/main.go
git add bin-api-manager/pkg/servicehandler/mock_main.go
git add bin-api-manager/pkg/servicehandler/customer_test.go
git add bin-api-manager/docsdev/source/customer_struct_customer.rst
git add bin-api-manager/docsdev/source/customer_overview.rst
git add bin-api-manager/docsdev/source/customer_tutorial.rst
git add bin-openapi-manager/openapi/openapi.yaml
git add bin-openapi-manager/openapi/paths/auth/unregister.yaml
git add CLAUDE.md
git add docs/plans/2026-02-25-customer-unregister-immediately-design.md
git add docs/plans/2026-02-25-customer-unregister-immediately-plan.md
```

Commit message:
```
NOJIRA-customer-unregister-immediately

Add immediate account deletion option to self-service unregister endpoint.

- bin-customer-manager: Add FreezeAndDelete method (freeze + anonymize PII + cascade delete)
- bin-customer-manager: Add /v1/customers/{id}/freeze_and_delete RPC route
- bin-common-handler: Add CustomerV1CustomerFreezeAndDelete RPC method
- bin-api-manager: Add immediate field to POST /auth/unregister request body
- bin-api-manager: Add CustomerSelfFreezeAndDelete servicehandler method
- bin-api-manager: Update customer RST docs to match WebhookMessage fields
- bin-api-manager: Add unregister tutorials to customer documentation
- bin-openapi-manager: Add immediate field to RequestBodyAuthUnregisterPOST schema
- docs: Add design and implementation plan
- CLAUDE.md: Add WebhookMessage rule for RST docs
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-customer-unregister-immediately
```

Then create PR with `gh pr create`.
