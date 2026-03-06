# Customer Metadata Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `Metadata` JSON field to the Customer model with a `rtp_debug` flag, updatable only by SuperAdmin via a dedicated endpoint.

**Architecture:** New `Metadata` struct stored as JSON column in `customer_customers`. Read automatically via existing SuperAdmin GET endpoints. Write via new `PUT /v1/customers/{id}/metadata` endpoint. Regular users never see it (excluded from WebhookMessage).

**Tech Stack:** Go, MySQL (JSON column), Alembic (migration), Squirrel (query builder), RabbitMQ RPC, OpenAPI + oapi-codegen

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-customer-metadata-support/`

**All file paths below are relative to the worktree root.**

---

### Task 1: Database Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_customer_customers_add_column_metadata.py`

**Step 1: Create the Alembic migration file**

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "customer_customers add column metadata"
```

If `alembic.ini` is not configured locally, manually create the migration file with a unique hash. The current head revision is `f2b3c4d5e6f7`.

**Step 2: Write the migration SQL**

Edit the generated file:

```python
def upgrade():
    op.execute("""ALTER TABLE customer_customers ADD COLUMN metadata JSON DEFAULT NULL;""")

def downgrade():
    op.execute("""ALTER TABLE customer_customers DROP COLUMN metadata;""")
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Add-customer-metadata-support

- bin-dbscheme-manager: Add metadata JSON column to customer_customers table"
```

---

### Task 2: Customer Model — Metadata Type, Field Constant, Struct Update

**Files:**
- Create: `bin-customer-manager/models/customer/metadata.go`
- Modify: `bin-customer-manager/models/customer/customer.go:21-49` (add Metadata field)
- Modify: `bin-customer-manager/models/customer/field.go:6-34` (add FieldMetadata)

**Step 1: Create `metadata.go`**

```go
package customer

// Metadata holds internal-use configuration flags for a customer.
// Managed exclusively by ProjectSuperAdmin. Not exposed in WebhookMessage.
type Metadata struct {
	RTPDebug bool `json:"rtp_debug"` // enable RTPEngine RTP capture (PCAP)
}
```

**Step 2: Add `Metadata` field to `Customer` struct**

In `customer.go`, add this field after `TermsAgreedIP`:

```go
	TermsAgreedIP      string `json:"terms_agreed_ip,omitempty" db:"terms_agreed_ip"`

	Metadata Metadata `json:"metadata" db:"metadata,json"` // internal options (admin-only)

	TMDeletionScheduled *time.Time `json:"tm_deletion_scheduled" db:"tm_deletion_scheduled"`
```

**Step 3: Add `FieldMetadata` constant**

In `field.go`, add after `FieldTermsAgreedIP`:

```go
	FieldTermsAgreedIP      Field = "terms_agreed_ip"

	FieldMetadata Field = "metadata"

	FieldTMCreate Field = "tm_create"
```

**Step 4: Run verification for bin-customer-manager**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass. Lint clean.

**Step 5: Commit**

```bash
git add bin-customer-manager/models/customer/
git commit -m "NOJIRA-Add-customer-metadata-support

- bin-customer-manager: Add Metadata struct with RTPDebug field
- bin-customer-manager: Add metadata JSON field to Customer model
- bin-customer-manager: Add FieldMetadata constant"
```

---

### Task 3: Internal RPC — Request Struct, Listen Handler, Business Logic

**Files:**
- Modify: `bin-customer-manager/pkg/listenhandler/models/request/customers.go` (add request struct)
- Modify: `bin-customer-manager/pkg/listenhandler/main.go` (add regex + route)
- Modify: `bin-customer-manager/pkg/listenhandler/v1_customers.go` (add handler function)
- Modify: `bin-customer-manager/pkg/customerhandler/main.go` (add interface method)
- Modify: `bin-customer-manager/pkg/customerhandler/db.go` (add UpdateMetadata)

**Step 1: Add request struct**

In `bin-customer-manager/pkg/listenhandler/models/request/customers.go`, add:

```go
// V1DataCustomersIDMetadataPut is
// v1 data type request struct for
// /v1/customers/<customer-id>/metadata PUT
type V1DataCustomersIDMetadataPut struct {
	Metadata customer.Metadata `json:"metadata"`
}
```

**Step 2: Add `UpdateMetadata` to CustomerHandler interface**

In `bin-customer-manager/pkg/customerhandler/main.go`, add to the `CustomerHandler` interface:

```go
	UpdateMetadata(ctx context.Context, id uuid.UUID, metadata customer.Metadata) (*customer.Customer, error)
```

Add it after `UpdateBillingAccountID`.

**Step 3: Implement `UpdateMetadata` in `db.go`**

In `bin-customer-manager/pkg/customerhandler/db.go`, add after `UpdateBillingAccountID`:

```go
// UpdateMetadata updates the customer's metadata.
func (h *customerHandler) UpdateMetadata(ctx context.Context, id uuid.UUID, metadata customer.Metadata) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdateMetadata",
		"customer_id": id,
	})
	log.Debug("Updating the customer's metadata.")

	fields := map[customer.Field]any{
		customer.FieldMetadata: metadata,
	}

	if err := h.db.CustomerUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the metadata. err: %v", err)
		return nil, err
	}

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated customer. err: %v", err)
		return nil, fmt.Errorf("could not get updated customer")
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerUpdated, res)

	return res, nil
}
```

**Step 4: Add regex and route in listen handler**

In `bin-customer-manager/pkg/listenhandler/main.go`, add regex after `regV1CustomersIDIsBillingAccountID`:

```go
	regV1CustomersIDIsMetadata = regexp.MustCompile("/v1/customers/" + regUUID + "/metadata$")
```

In the `processRequest` switch in `main.go`, add a new case **before** the `PUT /customers/<customer-id>` case (before line ~207, the `regV1CustomersID` PUT case). It must be placed before the generic `regV1CustomersID` match since that regex would also match `/metadata` URLs:

```go
	// PUT /customers/<customer-id>/metadata
	case regV1CustomersIDIsMetadata.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1CustomersIDMetadataPut(ctx, m)
		requestType = "/v1/customers/<customer_id>/metadata"
```

**Step 5: Add handler function in `v1_customers.go`**

At the end of `bin-customer-manager/pkg/listenhandler/v1_customers.go`, add:

```go
// processV1CustomersIDMetadataPut handles Put /v1/customers/<customer-id>/metadata request
func (h *listenHandler) processV1CustomersIDMetadataPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":        "processV1CustomersIDMetadataPut",
		"customer_id": id,
	})
	log.Debug("Executing processV1CustomersIDMetadataPut.")

	var req request.V1DataCustomersIDMetadataPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.UpdateMetadata(ctx, id, req.Metadata)
	if err != nil {
		log.Errorf("Could not update the customer's metadata. err: %v", err)
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

**Step 6: Write test for `UpdateMetadata`**

In `bin-customer-manager/pkg/customerhandler/db_test.go`, add a test following the `Test_UpdateBillingAccountID` pattern:

```go
func Test_UpdateMetadata(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		metadata customer.Metadata
	}{
		{
			"normal",
			uuid.FromStringOrNil("f2eb3d1e-0f8f-11ee-b3bb-178ed8e3acb7"),
			customer.Metadata{
				RTPDebug: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerUpdate(gomock.Any(), tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerUpdated, gomock.Any()).Return()

			_, err := h.UpdateMetadata(ctx, tt.id, tt.metadata)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}
```

**Step 7: Run verification**

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass including the new `Test_UpdateMetadata`.

**Step 8: Commit**

```bash
git add bin-customer-manager/
git commit -m "NOJIRA-Add-customer-metadata-support

- bin-customer-manager: Add UpdateMetadata business logic method
- bin-customer-manager: Add PUT /v1/customers/{id}/metadata RPC route and handler
- bin-customer-manager: Add V1DataCustomersIDMetadataPut request struct
- bin-customer-manager: Add Test_UpdateMetadata unit test"
```

---

### Task 4: Shared RPC Client — bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/customer_customer.go` (add UpdateMetadata function)
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (add interface method)

**Step 1: Add interface method**

In `bin-common-handler/pkg/requesthandler/main.go`, add to the `RequestHandler` interface after `CustomerV1CustomerUpdateBillingAccountID`:

```go
	CustomerV1CustomerUpdateMetadata(ctx context.Context, customerID uuid.UUID, metadata cscustomer.Metadata) (*cscustomer.Customer, error)
```

**Step 2: Add implementation**

In `bin-common-handler/pkg/requesthandler/customer_customer.go`, add after `CustomerV1CustomerUpdateBillingAccountID`:

```go
// CustomerV1CustomerUpdateMetadata sends a request to customer-manager
// to update the customer's metadata.
func (r *requestHandler) CustomerV1CustomerUpdateMetadata(ctx context.Context, customerID uuid.UUID, metadata cscustomer.Metadata) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s/metadata", customerID)

	data := &csrequest.V1DataCustomersIDMetadataPut{
		Metadata: metadata,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPut, "customer/customers/<customer-id>/metadata", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

**Step 3: Run verification for bin-common-handler**

```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Update bin-customer-manager vendor** (it depends on bin-common-handler)

```bash
cd bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-common-handler/ bin-customer-manager/
git commit -m "NOJIRA-Add-customer-metadata-support

- bin-common-handler: Add CustomerV1CustomerUpdateMetadata RPC method
- bin-customer-manager: Update vendor for bin-common-handler changes"
```

---

### Task 5: OpenAPI Schema + API Manager Endpoint

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add metadata schema + endpoint)
- Modify: `bin-api-manager/server/customers.go` (add PutCustomersIdMetadata handler)
- Modify: `bin-api-manager/pkg/servicehandler/customer.go` (add CustomerUpdateMetadata method)

**Step 1: Update OpenAPI spec**

Before modifying the spec, read `bin-openapi-manager/CLAUDE.md` for AI-Native rules.

Add a new schema `CustomerManagerMetadata` and a new path `/v1.0/customers/{id}/metadata` with a PUT operation.

Schema to add (in `components/schemas` section):

```yaml
    CustomerManagerMetadata:
      type: object
      properties:
        rtp_debug:
          type: boolean
          description: Enable RTPEngine RTP capture (PCAP) for debugging audio issues.
```

Path to add (in `paths` section, near other `/customers/{id}/*` paths):

```yaml
  /v1.0/customers/{id}/metadata:
    put:
      operationId: putCustomersIdMetadata
      summary: Update customer metadata (admin only)
      description: Updates internal metadata flags for a customer. Requires ProjectSuperAdmin permission.
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CustomerManagerMetadata'
      responses:
        '200':
          description: Updated customer
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CustomerManagerCustomer'
      tags:
        - customer
```

**Step 2: Regenerate OpenAPI types and API server**

```bash
cd bin-openapi-manager && go generate ./...
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./...
```

**Step 3: Add `CustomerUpdateMetadata` to api-manager servicehandler**

In `bin-api-manager/pkg/servicehandler/customer.go`, add:

```go
// CustomerUpdateMetadata updates the customer's internal metadata.
// Requires ProjectSuperAdmin permission.
func (h *serviceHandler) CustomerUpdateMetadata(
	ctx context.Context,
	a *amagent.Agent,
	customerID uuid.UUID,
	metadata cscustomer.Metadata,
) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerUpdateMetadata",
		"customer_id": customerID,
	})

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	_, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdateMetadata(ctx, customerID, metadata)
	if err != nil {
		log.Errorf("Could not update the customer's metadata. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

**Step 4: Add HTTP handler in `bin-api-manager/server/customers.go`**

Add the `PutCustomersIdMetadata` handler function. Follow the `PutCustomersIdBillingAccountId` pattern:

```go
func (h *server) PutCustomersIdMetadata(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCustomersIdMetadata",
		"request_address": c.ClientIP,
		"customer_id":     id,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCustomersIdMetadataJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	metadata := cucustomer.Metadata{
		RTPDebug: req.RtpDebug != nil && *req.RtpDebug,
	}

	res, err := h.serviceHandler.CustomerUpdateMetadata(c.Request.Context(), &a, target, metadata)
	if err != nil {
		log.Errorf("Could not update the customer metadata. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

Note: The exact field name on the generated `PutCustomersIdMetadataJSONBody` struct depends on what `oapi-codegen` generates. Check `bin-api-manager/gens/openapi_server/gen.go` after regeneration to see the exact field name (likely `RtpDebug *bool`). Adjust the handler code accordingly.

**Step 5: Run verification for all three services**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Add-customer-metadata-support

- bin-openapi-manager: Add CustomerManagerMetadata schema and PUT /customers/{id}/metadata endpoint
- bin-api-manager: Add PutCustomersIdMetadata HTTP handler
- bin-api-manager: Add CustomerUpdateMetadata servicehandler method with SuperAdmin permission check"
```

---

### Task 6: Final Verification + Push

**Step 1: Run full verification across all changed services**

```bash
cd bin-customer-manager && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-common-handler && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-openapi-manager && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-api-manager && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Add-customer-metadata-support
```

Create PR with title matching branch name: `NOJIRA-Add-customer-metadata-support`

---

### Out of Scope (Document for Future)

- **RST documentation**: Not needed since metadata is admin-only (not user-facing). No RST updates required.
- **Kamailio integration**: How calls read `rtp_debug` at call time and pass `record-call` flag to RTPEngine. Separate design needed.
- **customer-control CLI**: Optionally add `customer update-metadata` subcommand. Not critical for initial release.
