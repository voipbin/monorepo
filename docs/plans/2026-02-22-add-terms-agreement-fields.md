# Add Terms Agreement Fields Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `terms_agreed_version` and `terms_agreed_ip` fields to the customer model, populated during signup, with a required `accepted_tos` validation in the API layer.

**Architecture:** Two new string fields on the Customer struct flow from api-manager (where client IP is extracted) through RPC to customer-manager (where they're persisted). The `accepted_tos` boolean is validated at the API layer only — not stored. Fields are internal-only (no WebhookMessage/OpenAPI changes).

**Tech Stack:** Go, Alembic (Python) for DB migration, gomock for tests

---

### Task 1: Add Fields to Customer Model

**Files:**
- Modify: `bin-customer-manager/models/customer/customer.go:37-42`
- Modify: `bin-customer-manager/models/customer/field.go:22-27`

**Step 1: Add struct fields**

In `bin-customer-manager/models/customer/customer.go`, add two fields after the `Status`/`TMDeletionScheduled` block and before the timestamps:

```go
	Status              Status     `json:"status" db:"status"`
	TMDeletionScheduled *time.Time `json:"tm_deletion_scheduled" db:"tm_deletion_scheduled"`

	TermsAgreedVersion string `json:"terms_agreed_version,omitempty" db:"terms_agreed_version"`
	TermsAgreedIP      string `json:"terms_agreed_ip,omitempty" db:"terms_agreed_ip"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"` // Created timestamp.
```

**Step 2: Add field constants**

In `bin-customer-manager/models/customer/field.go`, add after `FieldTMDeletionScheduled`:

```go
	FieldTermsAgreedVersion Field = "terms_agreed_version"
	FieldTermsAgreedIP      Field = "terms_agreed_ip"
```

**Step 3: Verify build**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-customer-manager && go build ./...`
Expected: Compiles without errors.

**Step 4: Commit**

```bash
git add bin-customer-manager/models/customer/customer.go bin-customer-manager/models/customer/field.go
git commit -m "NOJIRA-add-terms-agreement-fields

- bin-customer-manager: Add TermsAgreedVersion and TermsAgreedIP fields to Customer model
- bin-customer-manager: Add corresponding field constants"
```

---

### Task 2: Add Database Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f6_customer_add_terms_agreed_columns.py`

**Step 1: Create migration file**

Create `bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f6_customer_add_terms_agreed_columns.py`:

```python
"""customer_add_terms_agreed_columns

Revision ID: a1b2c3d4e5f6
Revises: 8f42ac0555e7
Create Date: 2026-02-22 00:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f6'
down_revision = '8f42ac0555e7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE customer_customers
            ADD COLUMN terms_agreed_version VARCHAR(255) DEFAULT NULL AFTER status,
            ADD COLUMN terms_agreed_ip VARCHAR(255) DEFAULT NULL AFTER terms_agreed_version;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE customer_customers
            DROP COLUMN terms_agreed_ip,
            DROP COLUMN terms_agreed_version;
    """)
```

**Step 2: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f6_customer_add_terms_agreed_columns.py
git commit -m "NOJIRA-add-terms-agreement-fields

- bin-dbscheme-manager: Add Alembic migration for terms_agreed_version and terms_agreed_ip columns"
```

---

### Task 3: Update RPC Request Model and Customer-Manager Signup

This task updates the customer-manager side: the RPC request model, the listen handler, the customer handler interface, and the signup implementation.

**Files:**
- Modify: `bin-customer-manager/pkg/listenhandler/models/request/customers.go:44-55`
- Modify: `bin-customer-manager/pkg/customerhandler/main.go:54-63`
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go:47-87`
- Modify: `bin-customer-manager/pkg/listenhandler/v1_customers_signup.go:28-37`

**Step 1: Add `ClientIP` to request model**

In `bin-customer-manager/pkg/listenhandler/models/request/customers.go`, update `V1DataCustomersSignupPost`:

```go
// V1DataCustomersSignupPost is request struct for POST /v1/customers/signup
type V1DataCustomersSignupPost struct {
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Address     string `json:"address"`

	WebhookMethod customer.WebhookMethod `json:"webhook_method,omitempty"`
	WebhookURI    string                 `json:"webhook_uri,omitempty"`

	ClientIP string `json:"client_ip,omitempty"`
}
```

**Step 2: Update CustomerHandler interface**

In `bin-customer-manager/pkg/customerhandler/main.go`, add `clientIP string` parameter to `Signup`:

```go
	Signup(
		ctx context.Context,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod customer.WebhookMethod,
		webhookURI string,
		clientIP string,
	) (*customer.SignupResult, error)
```

**Step 3: Update Signup implementation**

In `bin-customer-manager/pkg/customerhandler/signup.go`:

1. Add `clientIP string` parameter to the `Signup` function signature (line 47-56):

```go
func (h *customerHandler) Signup(
	ctx context.Context,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod customer.WebhookMethod,
	webhookURI string,
	clientIP string,
) (*customer.SignupResult, error) {
```

2. Set the new fields on the customer struct (lines 73-87). Add `TermsAgreedVersion` and `TermsAgreedIP` to the struct literal:

```go
	u := &customer.Customer{
		ID: id,

		Name:   name,
		Detail: detail,

		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,

		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		EmailVerified: false,

		TermsAgreedVersion: time.Now().UTC().Format(time.RFC3339),
		TermsAgreedIP:      clientIP,
	}
```

**Step 4: Update listenhandler to pass clientIP**

In `bin-customer-manager/pkg/listenhandler/v1_customers_signup.go`, add `reqData.ClientIP` as the last argument to `h.customerHandler.Signup(...)`:

```go
	tmp, err := h.customerHandler.Signup(
		ctx,
		reqData.Name,
		reqData.Detail,
		reqData.Email,
		reqData.PhoneNumber,
		reqData.Address,
		reqData.WebhookMethod,
		reqData.WebhookURI,
		reqData.ClientIP,
	)
```

**Step 5: Regenerate mocks and verify build**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-customer-manager
go generate ./...
go build ./...
```
Expected: Compiles without errors. Mock files updated.

**Step 6: Commit**

```bash
git add bin-customer-manager/pkg/listenhandler/models/request/customers.go \
        bin-customer-manager/pkg/customerhandler/main.go \
        bin-customer-manager/pkg/customerhandler/signup.go \
        bin-customer-manager/pkg/listenhandler/v1_customers_signup.go \
        bin-customer-manager/pkg/customerhandler/mock_main.go
git commit -m "NOJIRA-add-terms-agreement-fields

- bin-customer-manager: Add clientIP parameter to Signup interface and implementation
- bin-customer-manager: Set TermsAgreedVersion and TermsAgreedIP during customer creation
- bin-customer-manager: Pass client_ip from RPC request to Signup handler"
```

---

### Task 4: Update RequestHandler (bin-common-handler)

This task updates the shared request handler to pass `clientIP` through the RPC call.

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:690-699`
- Modify: `bin-common-handler/pkg/requesthandler/customer_customer.go:167-205`

**Step 1: Update interface**

In `bin-common-handler/pkg/requesthandler/main.go`, add `clientIP string` to `CustomerV1CustomerSignup`:

```go
	CustomerV1CustomerSignup(
		ctx context.Context,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
		clientIP string,
	) (*cscustomer.SignupResult, error)
```

**Step 2: Update implementation**

In `bin-common-handler/pkg/requesthandler/customer_customer.go`, add `clientIP string` parameter and include it in the request data:

```go
func (r *requestHandler) CustomerV1CustomerSignup(
	ctx context.Context,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
	clientIP string,
) (*cscustomer.SignupResult, error) {
	uri := "/v1/customers/signup"

	reqData := csrequest.V1DataCustomersSignupPost{
		Name:          name,
		Detail:        detail,
		Email:         email,
		PhoneNumber:   phoneNumber,
		Address:       address,
		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,
		ClientIP:      clientIP,
	}
```

**Step 3: Regenerate mocks and verify build**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-common-handler
go generate ./...
go build ./...
```
Expected: Compiles without errors. Mock files updated.

**Step 4: Commit**

```bash
git add bin-common-handler/pkg/requesthandler/main.go \
        bin-common-handler/pkg/requesthandler/customer_customer.go \
        bin-common-handler/pkg/requesthandler/mock_main.go
git commit -m "NOJIRA-add-terms-agreement-fields

- bin-common-handler: Add clientIP parameter to CustomerV1CustomerSignup RPC call"
```

---

### Task 5: Update API-Manager (accepted_tos validation + clientIP pass-through)

This task updates the API gateway to validate `accepted_tos` and pass `clientIP` through.

**Files:**
- Modify: `bin-api-manager/lib/service/signup.go:21-70`
- Modify: `bin-api-manager/pkg/servicehandler/main.go:471-480`
- Modify: `bin-api-manager/pkg/servicehandler/customer.go:379-401`

**Step 1: Add `AcceptedTOS` to request body and validate**

In `bin-api-manager/lib/service/signup.go`, update `RequestBodySignupPOST`:

```go
// RequestBodySignupPOST is request body for POST /auth/signup
type RequestBodySignupPOST struct {
	Name          string `json:"name"`
	Detail        string `json:"detail"`
	Email         string `json:"email" binding:"required"`
	PhoneNumber   string `json:"phone_number"`
	Address       string `json:"address"`
	WebhookMethod string `json:"webhook_method"`
	WebhookURI    string `json:"webhook_uri"`
	AcceptedTOS   *bool  `json:"accepted_tos" binding:"required"`
}
```

Note: Use `*bool` so `binding:"required"` works correctly — a plain `bool` with `required` would reject `false` (zero value), but we want to reject *missing* fields. We separately check the value is `true`.

Update `PostCustomerSignup` to validate `accepted_tos == true` and pass `c.ClientIP()`:

```go
func PostCustomerSignup(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCustomerSignup",
		"request_address": c.ClientIP,
	})

	var req RequestBodySignupPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	if !*req.AcceptedTOS {
		log.Infof("Terms of service not accepted.")
		c.AbortWithStatus(400)
		return
	}

	log = log.WithFields(logrus.Fields{
		"email": req.Email,
	})
	log.Debugf("Processing customer signup. email: %s", req.Email)

	sh := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := sh.CustomerSignup(
		c.Request.Context(),
		req.Name,
		req.Detail,
		req.Email,
		req.PhoneNumber,
		req.Address,
		cscustomer.WebhookMethod(req.WebhookMethod),
		req.WebhookURI,
		c.ClientIP(),
	)
	if err != nil {
		log.Debugf("Customer signup failed. err: %v", err)
		// Return 200 with empty body to prevent email enumeration
		c.JSON(200, gin.H{})
		return
	}

	c.JSON(200, res)
}
```

**Step 2: Update ServiceHandler interface**

In `bin-api-manager/pkg/servicehandler/main.go`, add `clientIP string` to `CustomerSignup`:

```go
	CustomerSignup(
		ctx context.Context,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
		clientIP string,
	) (*cscustomer.SignupResult, error)
```

**Step 3: Update ServiceHandler implementation**

In `bin-api-manager/pkg/servicehandler/customer.go`, add `clientIP string` and pass it to the RPC call:

```go
func (h *serviceHandler) CustomerSignup(
	ctx context.Context,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
	clientIP string,
) (*cscustomer.SignupResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "CustomerSignup",
		"email": email,
	})
	log.Debug("Processing customer signup.")

	res, err := h.reqHandler.CustomerV1CustomerSignup(ctx, name, detail, email, phoneNumber, address, webhookMethod, webhookURI, clientIP)
	if err != nil {
		log.Errorf("Could not signup customer. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

**Step 4: Regenerate mocks and verify build**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-api-manager
go generate ./...
go build ./...
```
Expected: Compiles without errors. Mock files updated.

**Step 5: Commit**

```bash
git add bin-api-manager/lib/service/signup.go \
        bin-api-manager/pkg/servicehandler/main.go \
        bin-api-manager/pkg/servicehandler/customer.go \
        bin-api-manager/pkg/servicehandler/mock_main.go
git commit -m "NOJIRA-add-terms-agreement-fields

- bin-api-manager: Add accepted_tos validation to signup endpoint (reject if not true)
- bin-api-manager: Pass client IP through to customer-manager via RPC"
```

---

### Task 6: Update Tests

This task updates all tests that call the modified `Signup` and `CustomerSignup` functions.

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/signup_test.go`
- Modify: `bin-customer-manager/pkg/listenhandler/v1_customers_signup_test.go`
- Modify: `bin-api-manager/lib/service/signup_test.go`

**Step 1: Update customer-manager signup_test.go**

In `bin-customer-manager/pkg/customerhandler/signup_test.go`:

1. In `Test_Signup`, add `clientIP` field to test struct and the Signup call (line 104):
```go
res, err := h.Signup(ctx, tt.userName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI, "192.168.1.1")
```

2. Also verify the terms fields are set on the created customer. Update the `mockDB.EXPECT().CustomerCreate(ctx, gomock.Any())` to use a matcher that checks the fields:
```go
mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *customer.Customer) error {
    if c.TermsAgreedIP != "192.168.1.1" {
        t.Errorf("Expected TermsAgreedIP=192.168.1.1, got: %s", c.TermsAgreedIP)
    }
    if c.TermsAgreedVersion == "" {
        t.Errorf("Expected TermsAgreedVersion to be set, got empty")
    }
    return nil
})
```

3. Update all other test functions that call `Signup` (`Test_Signup_invalidEmail`, `Test_Signup_duplicateEmail`, `Test_Signup_customerCreateError`, `Test_Signup_emailVerifyTokenSetError`, `Test_Signup_signupSessionSetError`, `Test_Signup_emailSendFailureNonFatal`) to include the extra `clientIP` argument.

For test functions that use `gomock.Any()` matchers for Signup:
```go
mockCustomer.EXPECT().Signup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(...)
```
Note: This adds one more `gomock.Any()` for the `clientIP` parameter.

**Step 2: Update customer-manager listenhandler signup test**

In `bin-customer-manager/pkg/listenhandler/v1_customers_signup_test.go`:

1. In `Test_processV1CustomersSignupPost`, update the test request JSON data to include `client_ip`:
```go
Data: []byte(`{"name":"test signup","detail":"signup detail","email":"signup@voipbin.net","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com","client_ip":"10.0.0.1"}`),
```

2. Update the `mockCustomer.EXPECT().Signup(...)` call to include the `clientIP` argument:
```go
mockCustomer.EXPECT().Signup(
    gomock.Any(),
    "test signup",
    "signup detail",
    "signup@voipbin.net",
    "+821100000001",
    "somewhere",
    customer.WebhookMethod("POST"),
    "test.com",
    "10.0.0.1",
).Return(tt.responseSignupResult, nil)
```

3. In `Test_processV1CustomersSignupPost_signupError`, add one more `gomock.Any()`:
```go
mockCustomer.EXPECT().Signup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("signup failed"))
```

**Step 3: Update api-manager signup_test.go**

In `bin-api-manager/lib/service/signup_test.go`:

1. Update `RequestBodySignupPOST` in all test cases to include `AcceptedTOS`. Since it's now `*bool`, create a helper:
```go
func boolPtr(b bool) *bool { return &b }
```

2. Update the "valid signup" test case:
```go
reqBody: RequestBodySignupPOST{
    Name:          "Test Customer",
    Detail:        "Test details",
    Email:         "test@example.com",
    PhoneNumber:   "+1234567890",
    Address:       "123 Test St",
    WebhookMethod: "POST",
    WebhookURI:    "https://example.com/webhook",
    AcceptedTOS:   boolPtr(true),
},
```

3. Update mock expectations to include `clientIP` (9th arg — gomock.Any() for the IP):
```go
m.EXPECT().CustomerSignup(
    gomock.Any(),
    "Test Customer",
    "Test details",
    "test@example.com",
    "+1234567890",
    "123 Test St",
    cscustomer.WebhookMethod("POST"),
    "https://example.com/webhook",
    gomock.Any(), // clientIP
).Return(&cscustomer.SignupResult{}, nil)
```

4. Update the "signup failed" test case similarly with `AcceptedTOS: boolPtr(true)` and an extra `gomock.Any()`.

5. Add a new test case for `accepted_tos == false`:
```go
{
    name: "rejected - accepted_tos false",
    reqBody: RequestBodySignupPOST{
        Name:        "Test",
        Email:       "test@example.com",
        AcceptedTOS: boolPtr(false),
    },
    mockSetup:    func(m *servicehandler.MockServiceHandler) {},
    expectStatus: 400,
},
```

6. Add a test case for missing `accepted_tos`:
```go
// In TestPostCustomerSignup_MissingFields or a new test
{
    name:         "missing accepted_tos",
    body:         `{"email":"test@example.com"}`,
    expectStatus: 400,
},
```

**Step 4: Run all tests**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-customer-manager
go test ./...

cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-api-manager
go test ./...
```
Expected: All tests pass.

**Step 5: Commit**

```bash
git add bin-customer-manager/pkg/customerhandler/signup_test.go \
        bin-customer-manager/pkg/listenhandler/v1_customers_signup_test.go \
        bin-api-manager/lib/service/signup_test.go
git commit -m "NOJIRA-add-terms-agreement-fields

- bin-customer-manager: Update signup tests for clientIP parameter
- bin-customer-manager: Add test assertions for TermsAgreedVersion and TermsAgreedIP
- bin-api-manager: Update signup tests for accepted_tos validation and clientIP"
```

---

### Task 7: Full Verification and Final Commit

Run the full verification workflow for all modified services.

**Step 1: Verify bin-customer-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All steps pass.

**Step 2: Verify bin-common-handler**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All steps pass.

**Step 3: Verify bin-api-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All steps pass.

**Step 4: Push and create PR**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-terms-agreement-fields
git push -u origin NOJIRA-add-terms-agreement-fields
```

Then create a PR per project conventions.
