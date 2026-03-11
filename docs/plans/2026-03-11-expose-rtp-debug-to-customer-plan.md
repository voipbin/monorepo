# Expose RTPDebug to Customer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow CustomerAdmin to view and update the `rtp_debug` flag via the self-service `/customer` API.

**Architecture:** Add `Metadata` to `WebhookMessage` for read access, create `PUT /customer/metadata` endpoint following the existing `PUT /customer/billing_account_id` pattern. Reuses existing backend RPC call `CustomerV1CustomerUpdateMetadata`.

**Tech Stack:** Go, OpenAPI 3.0, oapi-codegen, gin, gomock

---

### Task 1: Add Metadata to WebhookMessage

**Files:**
- Modify: `bin-customer-manager/models/customer/webhook.go:11-63`
- Modify: `bin-customer-manager/models/customer/metadata.go:11-12`

**Step 1: Update metadata.go comment**

In `bin-customer-manager/models/customer/metadata.go`, update the comment on line 11-12:

```go
// Old:
// Metadata holds internal-use configuration flags for a customer.
// Managed exclusively by ProjectSuperAdmin. Not exposed in WebhookMessage.

// New:
// Metadata holds configuration flags for a customer.
// Can be updated by ProjectSuperAdmin via PUT /customers/{id}/metadata
// or by CustomerAdmin via PUT /customer/metadata.
```

**Step 2: Add Metadata field to WebhookMessage**

In `bin-customer-manager/models/customer/webhook.go`, add the `Metadata` field after `BillingAccountID` (after line 25):

```go
Metadata Metadata `json:"metadata"` // customer metadata
```

**Step 3: Update ConvertWebhookMessage to copy Metadata**

In `bin-customer-manager/models/customer/webhook.go`, add to `ConvertWebhookMessage()` return struct (after `BillingAccountID: h.BillingAccountID,`):

```go
Metadata: h.Metadata,
```

**Step 4: Run verification for bin-customer-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass. Note: test expected JSON responses in bin-api-manager will need updating later (they now include `"metadata":{"rtp_debug":false}`).

**Step 5: Commit**

```bash
git add bin-customer-manager/models/customer/webhook.go bin-customer-manager/models/customer/metadata.go
git commit -m "NOJIRA-Expose-rtp-debug-to-customer

- bin-customer-manager: Add Metadata field to WebhookMessage struct
- bin-customer-manager: Update ConvertWebhookMessage to copy Metadata
- bin-customer-manager: Update metadata comment to reflect new access"
```

---

### Task 2: Update OpenAPI Spec — Add metadata to CustomerManagerCustomer schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (line ~3333, after `billing_account_id` property in `CustomerManagerCustomer`)

**Step 1: Add metadata property to CustomerManagerCustomer schema**

In `bin-openapi-manager/openapi/openapi.yaml`, inside the `CustomerManagerCustomer` schema properties (after the `billing_account_id` block, before `email_verified`), add:

```yaml
        metadata:
          $ref: '#/components/schemas/CustomerManagerMetadata'
          description: "Customer configuration flags (e.g., RTP debug). Updatable by CustomerAdmin via `PUT /customer/metadata`."
```

**Step 2: Update CustomerManagerMetadata description**

Update the `CustomerManagerMetadata` description (line ~3270) to remove "managed exclusively by ProjectSuperAdmin" language:

```yaml
    CustomerManagerMetadata:
      type: object
      description: |
        Configuration flags for a customer account. Controls platform behavior
        such as RTP packet capture for debugging audio issues.
        Updatable by CustomerAdmin via `PUT /customer/metadata`
        or by ProjectSuperAdmin via `PUT /customers/{id}/metadata`.
      properties:
        rtp_debug:
          type: boolean
          description: |
            When set to `true`, RTPEngine captures RTP traffic as PCAP files for this customer's calls.
            Use this to debug audio quality issues (one-way audio, codec problems, jitter).
            Default is `false`. Enabling this increases storage usage — disable after debugging.
          example: true
```

**Step 3: Run verification for bin-openapi-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: Pass. `gens/models/gen.go` regenerated with metadata in `CustomerManagerCustomer`.

**Step 4: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml bin-openapi-manager/gens/models/gen.go
git commit -m "NOJIRA-Expose-rtp-debug-to-customer

- bin-openapi-manager: Add metadata property to CustomerManagerCustomer schema
- bin-openapi-manager: Update CustomerManagerMetadata description for self-service access"
```

---

### Task 3: Create OpenAPI Path for PUT /customer/metadata

**Files:**
- Create: `bin-openapi-manager/openapi/paths/customer/metadata.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (paths section, ~line 6634)

**Step 1: Create metadata.yaml path file**

Create `bin-openapi-manager/openapi/paths/customer/metadata.yaml` following the `billing_account_id.yaml` pattern:

```yaml
put:
  summary: Update customer metadata
  description: Update the metadata configuration for the authenticated customer's account.
  tags:
    - Customer
  requestBody:
    description: Customer metadata update payload
    required: true
    content:
      application/json:
        schema:
          type: object
          properties:
            rtp_debug:
              type: boolean
              description: |
                When set to `true`, RTPEngine captures RTP traffic as PCAP files for this customer's calls.
                Default is `false`. Enabling this increases storage usage — disable after debugging.
              example: true
  responses:
    '200':
      description: The updated customer information.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/CustomerManagerCustomer'
```

**Step 2: Register the path in openapi.yaml**

In `bin-openapi-manager/openapi/openapi.yaml`, in the paths section (around line 6634), add the new path reference. Add it before the existing `/customer/billing_account_id` entry:

```yaml
  /customer/metadata:
    $ref: './paths/customer/metadata.yaml'
```

**Step 3: Run verification for bin-openapi-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: Pass. `gens/models/gen.go` regenerated.

**Step 4: Commit**

```bash
git add bin-openapi-manager/openapi/paths/customer/metadata.yaml bin-openapi-manager/openapi/openapi.yaml bin-openapi-manager/gens/models/gen.go
git commit -m "NOJIRA-Expose-rtp-debug-to-customer

- bin-openapi-manager: Add PUT /customer/metadata path definition
- bin-openapi-manager: Register new path in main OpenAPI spec"
```

---

### Task 4: Add CustomerSelfUpdateMetadata to ServiceHandler Interface and Implementation

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (~line 492, after `CustomerSelfUpdateBillingAccountID`)
- Modify: `bin-api-manager/pkg/servicehandler/customer.go` (after `CustomerSelfUpdateBillingAccountID` method, ~line 520)

**Step 1: Add method to ServiceHandler interface**

In `bin-api-manager/pkg/servicehandler/main.go`, add after the `CustomerSelfUpdateBillingAccountID` line (~line 492):

```go
	CustomerSelfUpdateMetadata(ctx context.Context, a *amagent.Agent, metadata cscustomer.Metadata) (*cscustomer.WebhookMessage, error)
```

**Step 2: Implement CustomerSelfUpdateMetadata**

In `bin-api-manager/pkg/servicehandler/customer.go`, add after `CustomerSelfUpdateBillingAccountID` method (after line 520):

```go
// CustomerSelfUpdateMetadata updates the authenticated agent's own customer's metadata.
// Requires CustomerAdmin permission.
func (h *serviceHandler) CustomerSelfUpdateMetadata(ctx context.Context, a *amagent.Agent, metadata cscustomer.Metadata) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfUpdateMetadata",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdateMetadata(ctx, a.CustomerID, metadata)
	if err != nil {
		log.Errorf("Could not update the customer's metadata. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}
```

**Step 3: Regenerate mocks and vendor, then run verification**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: Tests may fail because existing test expected JSON doesn't include metadata. Fix in Task 6.

**Step 4: Commit** (if tests pass; otherwise defer to after Task 6)

```bash
git add bin-api-manager/pkg/servicehandler/main.go bin-api-manager/pkg/servicehandler/customer.go bin-api-manager/pkg/servicehandler/mock_main.go
git commit -m "NOJIRA-Expose-rtp-debug-to-customer

- bin-api-manager: Add CustomerSelfUpdateMetadata to ServiceHandler interface
- bin-api-manager: Implement CustomerSelfUpdateMetadata with CustomerAdmin permission"
```

---

### Task 5: Add PutCustomerMetadata Server Handler

**Files:**
- Modify: `bin-api-manager/server/customer.go` (after `PutCustomerBillingAccountId`, ~line 107)

**Step 1: Add PutCustomerMetadata handler**

In `bin-api-manager/server/customer.go`, add after `PutCustomerBillingAccountId` (after line 107):

```go
func (h *server) PutCustomerMetadata(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomerMetadata",
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

	var req openapi_server.PutCustomerMetadataJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	metadata := cmcustomer.Metadata{
		RTPDebug: req.RtpDebug != nil && *req.RtpDebug,
	}

	res, err := h.serviceHandler.CustomerSelfUpdateMetadata(c.Request.Context(), &a, metadata)
	if err != nil {
		log.Errorf("Could not update the customer metadata. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

Note: The `cmcustomer` import alias and `openapi_server` import already exist in `customer.go`.

**Step 2: Regenerate and run verification**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: May fail on existing tests due to metadata in JSON responses. Fix in Task 6.

---

### Task 6: Update Existing Tests and Add New Test

**Files:**
- Modify: `bin-api-manager/server/customer_test.go`

**Step 1: Update expected JSON responses in existing tests**

The `WebhookMessage` now includes `"metadata":{"rtp_debug":false}`. Update all `expectedRes` strings in the 3 existing tests.

In `Test_customerGET` (line 49), update `expectedRes`:
```go
expectedRes: `{"id":"e25f1af8-c44f-11ef-9d46-bfaf61e659c2","metadata":{"rtp_debug":false},"billing_account_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
```

In `Test_customerPut` (line 130), update `expectedRes`:
```go
expectedRes: `{"id":"4b7dcc68-c451-11ef-a289-33cbfe065115","metadata":{"rtp_debug":false},"billing_account_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
```

In `Test_customerBillingAccountIDPut` (line 201), update `expectedRes`:
```go
expectedRes: `{"id":"2422306e-c514-11ef-a89d-2f0585ee15f9","metadata":{"rtp_debug":false},"billing_account_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
```

**IMPORTANT:** The exact position of `"metadata"` in the JSON depends on the field order in `WebhookMessage`. Since we add `Metadata` after `BillingAccountID`, it may appear before or after it in the serialized output. Run the test first with a dummy value and check the actual JSON order to determine the correct expected string.

**Step 2: Add Test_customerMetadataPut test**

Add the new test following the `Test_customerBillingAccountIDPut` pattern:

```go
func Test_customerMetadataPut(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectedMetadata cscustomer.Metadata
		expectedRes      string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					CustomerID: uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f12345678901"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			reqQuery: "/customer/metadata",
			reqBody:  []byte(`{"rtp_debug":true}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f12345678901"),
				Metadata: cscustomer.Metadata{
					RTPDebug: true,
				},
			},

			expectedMetadata: cscustomer.Metadata{
				RTPDebug: true,
			},
			expectedRes: `{"id":"b2c3d4e5-f6a7-8901-bcde-f12345678901","metadata":{"rtp_debug":true},"billing_account_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest(http.MethodPut, tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerSelfUpdateMetadata(req.Context(), &tt.agent, tt.expectedMetadata).Return(tt.responseCustomer, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}
```

**Step 3: Run full verification for bin-api-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass.

**Step 4: Commit**

```bash
git add bin-api-manager/server/customer.go bin-api-manager/server/customer_test.go bin-api-manager/pkg/servicehandler/main.go bin-api-manager/pkg/servicehandler/customer.go bin-api-manager/pkg/servicehandler/mock_main.go bin-api-manager/gens/openapi_server/
git commit -m "NOJIRA-Expose-rtp-debug-to-customer

- bin-api-manager: Add PutCustomerMetadata server handler
- bin-api-manager: Add CustomerSelfUpdateMetadata service handler with CustomerAdmin permission
- bin-api-manager: Update existing customer tests for metadata in WebhookMessage
- bin-api-manager: Add Test_customerMetadataPut for new endpoint"
```

---

### Task 7: Final Verification and Squash Commits

**Step 1: Run verification for all affected services**

```bash
# bin-customer-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass for all 3 services.

**Step 2: Check for other tests that reference customer WebhookMessage JSON**

Search for other tests that might need updating due to the new metadata field in WebhookMessage:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer
grep -r "email_verified.*false.*status.*tm_deletion" --include="*_test.go" -l
```

If other test files match, update their expected JSON strings to include `"metadata":{"rtp_debug":false}`.

**Step 3: Push and create PR**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Expose-rtp-debug-to-customer
git push -u origin NOJIRA-Expose-rtp-debug-to-customer
```

Create PR with title: `NOJIRA-Expose-rtp-debug-to-customer`
