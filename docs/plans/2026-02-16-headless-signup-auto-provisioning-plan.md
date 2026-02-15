# Headless Signup & Auto-Provisioning Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable AI agents and CLI tools to sign up for VoIPbin and get an API key via a headless OTP flow, while also auto-provisioning keys on the existing web verification path.

**Architecture:** All new business logic lives in bin-customer-manager (owns Customer + AccessKey). bin-api-manager gets a new HTTP handler generated from OpenAPI. bin-agent-manager suppresses welcome email for headless signups. The event payload carries a `headless` flag via a new wrapper struct.

**Tech Stack:** Go, Redis, RabbitMQ RPC, oapi-codegen, gomock

---

### Task 1: Add signup session Redis methods to bin-customer-manager cachehandler

**Files:**
- Modify: `bin-customer-manager/pkg/cachehandler/main.go` (interface)
- Modify: `bin-customer-manager/pkg/cachehandler/handler.go` (implementation)

**Context:** The cachehandler currently has `EmailVerifyToken{Set,Get,Delete}`. We need three new methods for the signup session (JSON struct in Redis) and one for the rate-limit counter.

**Step 1: Write failing tests for the new cache methods**

Create: `bin-customer-manager/pkg/cachehandler/signup_session_test.go`

```go
package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
)

func Test_SignupSessionSet(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	h := &handler{
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}
	ctx := context.Background()

	session := &SignupSession{
		CustomerID:  uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
		OTPCode:     "123456",
		VerifyToken: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}

	err := h.SignupSessionSet(ctx, "temptoken123", session, time.Hour)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

func Test_SignupSessionGet(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	h := &handler{
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}
	ctx := context.Background()

	session := &SignupSession{
		CustomerID:  uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
		OTPCode:     "123456",
		VerifyToken: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}
	_ = h.SignupSessionSet(ctx, "temptoken123", session, time.Hour)

	res, err := h.SignupSessionGet(ctx, "temptoken123")
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if res.OTPCode != "123456" {
		t.Errorf("expected OTPCode 123456, got: %s", res.OTPCode)
	}
}

func Test_SignupSessionDelete(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	h := &handler{
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}
	ctx := context.Background()

	session := &SignupSession{
		CustomerID:  uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
		OTPCode:     "123456",
		VerifyToken: "token",
	}
	_ = h.SignupSessionSet(ctx, "temptoken123", session, time.Hour)
	err := h.SignupSessionDelete(ctx, "temptoken123")
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	_, err = h.SignupSessionGet(ctx, "temptoken123")
	if err == nil {
		t.Errorf("expected error after delete, got nil")
	}
}

func Test_SignupAttemptIncrement(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	h := &handler{
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}
	ctx := context.Background()

	count, err := h.SignupAttemptIncrement(ctx, "temptoken123", time.Hour)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got: %d", count)
	}

	count, err = h.SignupAttemptIncrement(ctx, "temptoken123", time.Hour)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got: %d", count)
	}
}
```

Note: Check if `miniredis` is already vendored. If not, you will need to write integration-style tests using the mock interface instead, following the existing test patterns in the codebase. The tests above illustrate intent — adapt to whatever testing pattern the project uses for Redis.

**Step 2: Run tests to verify they fail**

```bash
cd bin-customer-manager && go test ./pkg/cachehandler/... -v -run "Test_SignupSession|Test_SignupAttempt"
```

Expected: FAIL — methods not defined yet.

**Step 3: Implement the cache methods**

Add the `SignupSession` struct and methods to `handler.go`:

```go
const signupSessionKeyPrefix = "signup_session:"
const signupAttemptsKeyPrefix = "signup_attempts:"

// SignupSession stores the headless signup session data in Redis.
type SignupSession struct {
	CustomerID  uuid.UUID `json:"customer_id"`
	OTPCode     string    `json:"otp_code"`
	VerifyToken string    `json:"verify_token"`
}

// SignupSessionSet stores a signup session in Redis with TTL.
func (h *handler) SignupSessionSet(ctx context.Context, tempToken string, session *SignupSession, ttl time.Duration) error {
	key := signupSessionKeyPrefix + tempToken
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return h.Cache.Set(ctx, key, data, ttl).Err()
}

// SignupSessionGet retrieves a signup session from Redis.
func (h *handler) SignupSessionGet(ctx context.Context, tempToken string) (*SignupSession, error) {
	key := signupSessionKeyPrefix + tempToken
	val, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("signup session not found or expired")
		}
		return nil, err
	}

	var session SignupSession
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, fmt.Errorf("could not parse signup session: %v", err)
	}
	return &session, nil
}

// SignupSessionDelete removes a signup session from Redis.
func (h *handler) SignupSessionDelete(ctx context.Context, tempToken string) error {
	key := signupSessionKeyPrefix + tempToken
	return h.Cache.Del(ctx, key).Err()
}

// SignupAttemptIncrement increments the attempt counter for a temp_token. Returns the new count.
func (h *handler) SignupAttemptIncrement(ctx context.Context, tempToken string, ttl time.Duration) (int64, error) {
	key := signupAttemptsKeyPrefix + tempToken
	count, err := h.Cache.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set TTL on first increment
	if count == 1 {
		h.Cache.Expire(ctx, key, ttl)
	}
	return count, nil
}

// SignupAttemptDelete removes the attempt counter from Redis.
func (h *handler) SignupAttemptDelete(ctx context.Context, tempToken string) error {
	key := signupAttemptsKeyPrefix + tempToken
	return h.Cache.Del(ctx, key).Err()
}
```

Add the new methods to the `CacheHandler` interface in `main.go`:

```go
SignupSessionSet(ctx context.Context, tempToken string, session *SignupSession, ttl time.Duration) error
SignupSessionGet(ctx context.Context, tempToken string) (*SignupSession, error)
SignupSessionDelete(ctx context.Context, tempToken string) error
SignupAttemptIncrement(ctx context.Context, tempToken string, ttl time.Duration) (int64, error)
SignupAttemptDelete(ctx context.Context, tempToken string) error
```

**Step 4: Regenerate mocks and run tests**

```bash
cd bin-customer-manager && go generate ./pkg/cachehandler/... && go test ./pkg/cachehandler/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-customer-manager/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-customer-manager: Add SignupSession Redis cache methods for OTP-based headless signup
- bin-customer-manager: Add SignupAttemptIncrement for rate limiting per temp_token"
```

---

### Task 2: Add SignupResult model and modify Signup() to return temp_token

**Files:**
- Create: `bin-customer-manager/models/customer/signup.go`
- Modify: `bin-customer-manager/pkg/customerhandler/main.go` (interface signature)
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go` (implementation)
- Modify: `bin-customer-manager/pkg/customerhandler/signup_test.go`

**Context:** The `Signup()` method currently returns `(*customer.Customer, error)`. We need it to also return the `temp_token` so the caller can give it to the API response. We introduce a `SignupResult` struct that wraps the customer + temp_token.

**Step 1: Create the SignupResult model**

Create `bin-customer-manager/models/customer/signup.go`:

```go
package customer

// SignupResult contains the result of a signup operation.
// Includes the customer and the temp_token for headless verification.
type SignupResult struct {
	Customer  *Customer `json:"customer,omitempty"`
	TempToken string    `json:"temp_token,omitempty"`
}
```

**Step 2: Update the CustomerHandler interface**

In `bin-customer-manager/pkg/customerhandler/main.go`, change:

```go
// Old:
Signup(...) (*customer.Customer, error)
EmailVerify(ctx context.Context, token string) (*customer.Customer, error)

// New:
Signup(...) (*customer.SignupResult, error)
EmailVerify(ctx context.Context, token string) (*customer.EmailVerifyResult, error)
CompleteSignup(ctx context.Context, tempToken string, code string) (*customer.CompleteSignupResult, error)
```

Also create `bin-customer-manager/models/customer/signup.go` with all three result structs:

```go
package customer

import "monorepo/bin-customer-manager/models/accesskey"

// SignupResult contains the result of a signup operation.
type SignupResult struct {
	Customer  *Customer `json:"customer,omitempty"`
	TempToken string    `json:"temp_token,omitempty"`
}

// CompleteSignupResult contains the result of a headless signup completion.
type CompleteSignupResult struct {
	CustomerID string             `json:"customer_id"`
	Accesskey  *accesskey.Accesskey `json:"accesskey,omitempty"`
}

// EmailVerifyResult contains the result of an email verification.
type EmailVerifyResult struct {
	Customer  *Customer            `json:"customer,omitempty"`
	Accesskey *accesskey.Accesskey `json:"accesskey,omitempty"`
}
```

**Step 3: Modify Signup() implementation**

In `signup.go`, modify `Signup()` to:
1. Generate OTP code (6 digits): `otpCode := fmt.Sprintf("%06d", cryptoRandInt(100000, 999999))`
2. Generate temp_token (16 random bytes → 32 hex chars)
3. Store signup session in Redis via `h.cache.SignupSessionSet()`
4. Modify `sendVerificationEmail()` to accept and include the OTP code
5. Return `*customer.SignupResult{Customer: res, TempToken: tempToken}`

Add a helper to generate crypto-secure random int in range:

```go
func cryptoRandInt(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	return int(n.Int64()) + min
}
```

Note: Use `crypto/rand` and `math/big` for the random int. The `crypto/rand` package is already imported.

**Step 4: Modify sendVerificationEmail()**

Update signature to `sendVerificationEmail(ctx, email, token, otpCode string)` and update the email template:

```go
subject := fmt.Sprintf("VoIPBin - Verify Your Email (Code: %s)", otpCode)
content := fmt.Sprintf(
    "Welcome to VoIPBin!\n\n"+
        "Your verification code is: %s\n\n"+
        "API Users: POST this code with your temp_token to /v1/auth/complete-signup\n\n"+
        "Or click the link below to verify via browser (expires in 1 hour):\n\n"+
        "%s\n\n"+
        "If you did not create this account, you can safely ignore this email.",
    otpCode,
    verifyLink,
)
```

**Step 5: Update tests**

Update `Test_Signup` to expect `SignupSessionSet` call on the mock cache and return `*customer.SignupResult`.
Update `Test_sendVerificationEmail` to pass the new `otpCode` parameter.

**Step 6: Run verification and commit**

```bash
cd bin-customer-manager && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-customer-manager/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-customer-manager: Add SignupResult, CompleteSignupResult, EmailVerifyResult models
- bin-customer-manager: Modify Signup() to generate OTP code and temp_token
- bin-customer-manager: Update verification email to include OTP code"
```

---

### Task 3: Add CompleteSignup() to customerhandler and modify EmailVerify() for auto-provisioning

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/main.go` (add accesskeyHandler dependency)
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go` (add CompleteSignup, modify EmailVerify)
- Modify: `bin-customer-manager/pkg/customerhandler/signup_test.go`

**Context:** The `customerHandler` currently has no reference to `accesskeyHandler`. We need to inject it so that both `CompleteSignup()` and `EmailVerify()` can create an AccessKey.

**Step 1: Add accesskeyHandler dependency**

In `main.go`, add `accesskeyHandler` to the struct and constructor:

```go
type customerHandler struct {
    utilHandler      utilhandler.UtilHandler
    reqHandler       requesthandler.RequestHandler
    db               dbhandler.DBHandler
    cache            cachehandler.CacheHandler
    notifyHandler    notifyhandler.NotifyHandler
    accesskeyHandler accesskeyhandler.AccesskeyHandler  // NEW
}

func NewCustomerHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, cache cachehandler.CacheHandler, notifyHandler notifyhandler.NotifyHandler, accesskeyHandler accesskeyhandler.AccesskeyHandler) CustomerHandler {
```

**CRITICAL:** This changes the constructor signature. You must find ALL callers of `NewCustomerHandler` across the monorepo (likely in `bin-customer-manager/cmd/customer-manager/main.go` and `bin-customer-manager/cmd/customer-control/main.go`) and update them to pass the new parameter.

**Step 2: Implement CompleteSignup()**

Add to `signup.go`:

```go
const maxSignupAttempts = 5

// CompleteSignup validates an OTP code and completes the headless signup flow.
func (h *customerHandler) CompleteSignup(ctx context.Context, tempToken string, code string) (*customer.CompleteSignupResult, error) {
    log := logrus.WithFields(logrus.Fields{
        "func": "CompleteSignup",
    })
    log.Debug("Processing headless signup completion.")

    // Rate limit check
    count, err := h.cache.SignupAttemptIncrement(ctx, tempToken, emailVerifyTokenTTL)
    if err != nil {
        log.Errorf("Could not increment attempt counter. err: %v", err)
        return nil, fmt.Errorf("internal error")
    }
    if count > maxSignupAttempts {
        log.Infof("Too many attempts for temp_token.")
        metricshandler.CompleteSignupTotal.WithLabelValues("rate_limited").Inc()
        return nil, fmt.Errorf("too many attempts")
    }

    // Get signup session from Redis
    session, err := h.cache.SignupSessionGet(ctx, tempToken)
    if err != nil {
        log.Errorf("Could not get signup session. err: %v", err)
        metricshandler.CompleteSignupTotal.WithLabelValues("invalid_token").Inc()
        return nil, fmt.Errorf("invalid or expired temp_token")
    }
    log.Debugf("Found signup session. customer_id: %s", session.CustomerID)

    // Validate OTP
    if session.OTPCode != code {
        log.Infof("Invalid OTP code.")
        metricshandler.CompleteSignupTotal.WithLabelValues("invalid_code").Inc()
        return nil, fmt.Errorf("invalid verification code")
    }

    // Mark customer as verified
    fields := map[customer.Field]any{
        customer.FieldEmailVerified: true,
    }
    if err := h.db.CustomerUpdate(ctx, session.CustomerID, fields); err != nil {
        log.Errorf("Could not update customer. err: %v", err)
        metricshandler.CompleteSignupTotal.WithLabelValues("error").Inc()
        return nil, fmt.Errorf("could not verify customer")
    }

    // Delete all Redis keys (session + attempts + email verify token)
    _ = h.cache.SignupSessionDelete(ctx, tempToken)
    _ = h.cache.SignupAttemptDelete(ctx, tempToken)
    _ = h.cache.EmailVerifyTokenDelete(ctx, session.VerifyToken)

    // Create AccessKey
    ak, err := h.accesskeyHandler.Create(ctx, session.CustomerID, "default", "Auto-provisioned API key", 0)
    if err != nil {
        log.Errorf("Could not create access key. err: %v", err)
        metricshandler.CompleteSignupTotal.WithLabelValues("error").Inc()
        return nil, fmt.Errorf("could not create access key")
    }
    log.WithField("accesskey", ak).Debugf("Created access key. accesskey_id: %s", ak.ID)

    // Get verified customer for event publishing
    cu, err := h.db.CustomerGet(ctx, session.CustomerID)
    if err != nil {
        log.Errorf("Could not get verified customer. err: %v", err)
    }

    // Publish customer_created event with headless=true
    if cu != nil {
        h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerCreated, &customer.CustomerCreatedEvent{
            Customer: cu,
            Headless: true,
        })
    }

    metricshandler.CompleteSignupTotal.WithLabelValues("success").Inc()

    return &customer.CompleteSignupResult{
        CustomerID: session.CustomerID.String(),
        Accesskey:  ak,
    }, nil
}
```

**Step 3: Modify EmailVerify() for auto-provisioning**

After the existing `h.notifyHandler.PublishEvent(...)` line in `EmailVerify()`, add AccessKey creation:

```go
// Create AccessKey (auto-provisioning)
ak, err := h.accesskeyHandler.Create(ctx, customerID, "default", "Auto-provisioned API key", 0)
if err != nil {
    log.Errorf("Could not create access key during email verify. err: %v", err)
    // Non-fatal — customer is verified but key creation failed
}

// Change the event to include headless=false
h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerCreated, &customer.CustomerCreatedEvent{
    Customer: res,
    Headless: false,
})

return &customer.EmailVerifyResult{
    Customer:  res,
    Accesskey: ak,
}, nil
```

**Step 4: Create the CustomerCreatedEvent wrapper**

Add to `bin-customer-manager/models/customer/event.go`:

```go
// CustomerCreatedEvent wraps the Customer with headless flag for the customer_created event.
type CustomerCreatedEvent struct {
    *Customer
    Headless bool `json:"headless"`
}
```

**Step 5: Write tests for CompleteSignup()**

Add to `signup_test.go`:

```go
func Test_CompleteSignup(t *testing.T) {
    // Test normal flow: valid temp_token + valid OTP → customer verified + AccessKey created
    // Mock expectations:
    //   cache.SignupAttemptIncrement → (1, nil)
    //   cache.SignupSessionGet → session
    //   db.CustomerUpdate → nil
    //   cache.SignupSessionDelete → nil
    //   cache.SignupAttemptDelete → nil
    //   cache.EmailVerifyTokenDelete → nil
    //   accesskeyHandler.Create → accesskey
    //   db.CustomerGet → customer
    //   notifyHandler.PublishEvent → called with CustomerCreatedEvent{Headless: true}
}

func Test_CompleteSignup_invalidCode(t *testing.T) {
    // Test wrong OTP code returns error
}

func Test_CompleteSignup_rateLimited(t *testing.T) {
    // Test >5 attempts returns error
}

func Test_CompleteSignup_expiredToken(t *testing.T) {
    // Test expired/missing temp_token returns error
}
```

**Step 6: Run verification and commit**

```bash
cd bin-customer-manager && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-customer-manager/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-customer-manager: Add CompleteSignup() for headless OTP verification
- bin-customer-manager: Modify EmailVerify() to auto-provision AccessKey
- bin-customer-manager: Add CustomerCreatedEvent wrapper with headless flag
- bin-customer-manager: Inject accesskeyHandler into customerHandler"
```

---

### Task 4: Add RPC route for complete_signup in bin-customer-manager listenhandler

**Files:**
- Create: `bin-customer-manager/pkg/listenhandler/models/request/complete_signup.go` (or add to `customers.go`)
- Modify: `bin-customer-manager/pkg/listenhandler/main.go` (add route)
- Create: `bin-customer-manager/pkg/listenhandler/v1_customers_complete_signup.go`

**Context:** The listenhandler routes RPC requests by regex pattern matching on URI + method. We need a new route: `POST /v1/customers/complete_signup`.

**Step 1: Add request model**

Add to `bin-customer-manager/pkg/listenhandler/models/request/customers.go`:

```go
// V1DataCustomersCompleteSignupPost is request struct for POST /v1/customers/complete_signup
type V1DataCustomersCompleteSignupPost struct {
    TempToken string `json:"temp_token"`
    Code      string `json:"code"`
}
```

**Step 2: Add the listenhandler processor**

Create `bin-customer-manager/pkg/listenhandler/v1_customers_complete_signup.go`:

```go
package listenhandler

import (
    "context"
    "encoding/json"

    "monorepo/bin-common-handler/models/sock"
    "monorepo/bin-customer-manager/pkg/listenhandler/models/request"

    "github.com/sirupsen/logrus"
)

// processV1CustomersCompleteSignupPost handles POST /v1/customers/complete_signup
func (h *listenHandler) processV1CustomersCompleteSignupPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":    "processV1CustomersCompleteSignupPost",
        "request": m,
    })
    log.Debug("Executing processV1CustomersCompleteSignupPost.")

    var reqData request.V1DataCustomersCompleteSignupPost
    if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
        log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
        return simpleResponse(400), nil
    }

    tmp, err := h.customerHandler.CompleteSignup(ctx, reqData.TempToken, reqData.Code)
    if err != nil {
        log.Errorf("Could not complete signup. err: %v", err)
        return simpleResponse(400), nil
    }

    data, err := json.Marshal(tmp)
    if err != nil {
        log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
        return simpleResponse(500), nil
    }

    return &sock.Response{
        StatusCode: 200,
        DataType:   "application/json",
        Data:       data,
    }, nil
}
```

**Step 3: Add route to main.go**

In `main.go`, add the regex and route:

```go
// Add regex
regV1CustomersCompleteSignup = regexp.MustCompile("/v1/customers/complete_signup$")

// Add case in processRequest switch (BEFORE the generic regV1Customers cases)
case regV1CustomersCompleteSignup.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1CustomersCompleteSignupPost(ctx, m)
    requestType = "/v1/customers/complete_signup"
```

**IMPORTANT:** Place this case BEFORE the `regV1CustomersSignup` case in the switch to avoid regex conflicts. The order matters.

**Step 4: Update the listenhandler for the modified Signup return type**

The `processV1CustomersSignupPost` currently marshals a `*customer.Customer`. It now receives `*customer.SignupResult`. Update `v1_customers_signup.go`:

```go
// processV1CustomersSignupPost — change return type handling
tmp, err := h.customerHandler.Signup(...)
// tmp is now *customer.SignupResult, marshal it directly
data, err := json.Marshal(tmp)
```

Similarly update `processV1CustomersEmailVerifyPost` to handle `*customer.EmailVerifyResult`.

**Step 5: Run verification and commit**

```bash
cd bin-customer-manager && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-customer-manager/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-customer-manager: Add POST /v1/customers/complete_signup RPC route
- bin-customer-manager: Update signup and email_verify RPC handlers for new return types"
```

---

### Task 5: Add RPC method to bin-common-handler requesthandler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (interface)
- Modify: `bin-common-handler/pkg/requesthandler/customer_customer.go` (implementation)

**Context:** bin-api-manager calls bin-customer-manager via requesthandler RPC methods. We need a new method `CustomerV1CustomerCompleteSignup` and need to update the return types for `CustomerV1CustomerSignup` and `CustomerV1CustomerEmailVerify`.

**Step 1: Add the new RPC method**

In `customer_customer.go`, add:

```go
// CustomerV1CustomerCompleteSignup sends a complete-signup request to customer-manager.
func (r *requestHandler) CustomerV1CustomerCompleteSignup(ctx context.Context, tempToken string, code string) (*cscustomer.CompleteSignupResult, error) {
    uri := "/v1/customers/complete_signup"

    reqData := csrequest.V1DataCustomersCompleteSignupPost{
        TempToken: tempToken,
        Code:      code,
    }

    m, err := json.Marshal(reqData)
    if err != nil {
        return nil, err
    }

    tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPost, "customer/customers/complete_signup", requestTimeoutDefault, 0, ContentTypeJSON, m)
    if err != nil {
        return nil, err
    }

    var res cscustomer.CompleteSignupResult
    if errParse := parseResponse(tmp, &res); errParse != nil {
        return nil, errParse
    }

    return &res, nil
}
```

**Step 2: Update existing RPC methods' return types**

- `CustomerV1CustomerSignup` → change return type from `*cscustomer.Customer` to `*cscustomer.SignupResult`
- `CustomerV1CustomerEmailVerify` → change return type from `*cscustomer.Customer` to `*cscustomer.EmailVerifyResult`

**Step 3: Add to interface in main.go**

```go
CustomerV1CustomerCompleteSignup(ctx context.Context, tempToken string, code string) (*cscustomer.CompleteSignupResult, error)
```

And update the existing method signatures in the interface.

**Step 4: Regenerate mocks**

```bash
cd bin-common-handler && go generate ./... && go test ./...
```

**Step 5: Commit**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-common-handler/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-common-handler: Add CustomerV1CustomerCompleteSignup RPC method
- bin-common-handler: Update Signup and EmailVerify RPC return types for new result structs"
```

---

### Task 6: Add Prometheus metrics for complete_signup

**Files:**
- Modify: `bin-customer-manager/pkg/metricshandler/metrics.go` (or equivalent)

**Context:** The existing code uses `metricshandler.SignupTotal` and `metricshandler.EmailVerificationTotal`. We need a new `CompleteSignupTotal` counter.

**Step 1: Add the metric**

Check `bin-customer-manager/pkg/metricshandler/` for the existing metric definitions. Add:

```go
CompleteSignupTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: namespace,
        Name:      "complete_signup_total",
        Help:      "Total number of headless complete-signup attempts",
    },
    []string{"status"},
)
```

Register it in `init()`.

**CRITICAL:** Check `bin-common-handler/pkg/requesthandler/main.go` `initPrometheus()` to ensure the metric name doesn't conflict. The namespace for customer-manager metrics should be unique.

**Step 2: Commit**

```bash
cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-customer-manager/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-customer-manager: Add Prometheus metrics for complete_signup endpoint"
```

---

### Task 7: Add OpenAPI spec for complete-signup endpoint

**Files:**
- Create: `bin-openapi-manager/openapi/paths/auth/complete-signup.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add path reference + response schemas)
- Modify: `bin-openapi-manager/openapi/paths/auth/signup.yaml` (update response schema)
- Modify: `bin-openapi-manager/openapi/paths/auth/email-verify.yaml` (update response schema)

**Step 1: Create complete-signup.yaml**

```yaml
post:
  summary: Complete headless signup verification
  description: |
    Validates a 6-digit OTP code and completes the signup process for headless/API clients.

    Returns the customer ID and an auto-provisioned API access key.

    This endpoint does not require authentication.

    Rate limited to 5 attempts per temp_token.
  tags:
    - Auth
  security: []
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          required:
            - temp_token
            - code
          properties:
            temp_token:
              type: string
              description: Temporary token returned from POST /auth/signup
              example: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
            code:
              type: string
              description: 6-digit verification code from email
              example: "123456"
  responses:
    '200':
      description: Signup completed successfully
      content:
        application/json:
          schema:
            type: object
            properties:
              customer_id:
                type: string
                format: uuid
              accesskey:
                $ref: '#/components/schemas/CustomerManagerAccesskey'
    '400':
      description: Invalid or expired temp_token, or wrong verification code
    '429':
      description: Too many attempts (max 5 per temp_token)
```

**Step 2: Add path reference in openapi.yaml**

After the `/auth/email-verify:` line (~line 4072), add:

```yaml
  /auth/complete-signup:
    $ref: './paths/auth/complete-signup.yaml'
```

**Step 3: Update signup.yaml response**

Change the 200 response schema from `CustomerManagerCustomer` to an inline schema that includes `temp_token`:

```yaml
  responses:
    '200':
      description: Request accepted (always returns 200 to prevent email enumeration)
      content:
        application/json:
          schema:
            type: object
            properties:
              customer:
                $ref: '#/components/schemas/CustomerManagerCustomer'
              temp_token:
                type: string
                description: Temporary token for headless verification via /auth/complete-signup
```

**Step 4: Update email-verify.yaml POST response**

Change the 200 response schema to include accesskey:

```yaml
  responses:
    '200':
      description: Email successfully verified
      content:
        application/json:
          schema:
            type: object
            properties:
              customer:
                $ref: '#/components/schemas/CustomerManagerCustomer'
              accesskey:
                $ref: '#/components/schemas/CustomerManagerAccesskey'
```

**Step 5: Regenerate and commit**

```bash
cd bin-openapi-manager && go generate ./... && go test ./...
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-openapi-manager: Add POST /auth/complete-signup endpoint spec
- bin-openapi-manager: Update signup response to include temp_token
- bin-openapi-manager: Update email-verify response to include accesskey
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 8: Add HTTP handler for complete-signup in bin-api-manager

**Files:**
- Create: `bin-api-manager/server/auth_complete_signup.go`
- Modify: `bin-api-manager/pkg/servicehandler/customer.go` (add service method)
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (add to interface)

**Context:** After running `go generate` with the updated OpenAPI spec (Task 7), the generated `ServerInterface` will include `PostAuthCompleteSignup(c *gin.Context)`. We need to implement it.

**Step 1: Add servicehandler method**

In `bin-api-manager/pkg/servicehandler/customer.go`, add:

```go
// CustomerCompleteSignup validates OTP and returns an auto-provisioned AccessKey.
// This is a public endpoint — no authentication required.
func (h *serviceHandler) CustomerCompleteSignup(ctx context.Context, tempToken string, code string) (*cscustomer.CompleteSignupResult, error) {
    log := logrus.WithFields(logrus.Fields{
        "func": "CustomerCompleteSignup",
    })
    log.Debug("Processing customer complete signup.")

    res, err := h.reqHandler.CustomerV1CustomerCompleteSignup(ctx, tempToken, code)
    if err != nil {
        log.Errorf("Could not complete signup. err: %v", err)
        return nil, err
    }

    return res, nil
}
```

Also update `CustomerSignup` return type from `*cscustomer.WebhookMessage` to `*cscustomer.SignupResult`.
And update `CustomerEmailVerify` return type from `*cscustomer.WebhookMessage` to `*cscustomer.EmailVerifyResult`.

Add to the `ServiceHandler` interface in `main.go`:

```go
CustomerCompleteSignup(ctx context.Context, tempToken string, code string) (*cscustomer.CompleteSignupResult, error)
```

**Step 2: Implement the HTTP handler**

Create `bin-api-manager/server/auth_complete_signup.go`:

```go
package server

import (
    "monorepo/bin-api-manager/gens/openapi_server"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

func (h *server) PostAuthCompleteSignup(c *gin.Context) {
    log := logrus.WithFields(logrus.Fields{
        "func":            "PostAuthCompleteSignup",
        "request_address": c.ClientIP,
    })
    log.Debug("Processing complete signup.")

    var req openapi_server.PostAuthCompleteSignupJSONBody
    if err := c.BindJSON(&req); err != nil {
        log.Warnf("Could not bind the request body. err: %v", err)
        c.AbortWithStatus(400)
        return
    }

    res, err := h.serviceHandler.CustomerCompleteSignup(c.Request.Context(), req.TempToken, req.Code)
    if err != nil {
        log.Debugf("Complete signup failed. err: %v", err)
        // Check if rate limited
        if err.Error() == "too many attempts" {
            c.AbortWithStatus(429)
            return
        }
        c.AbortWithStatus(400)
        return
    }

    c.JSON(200, res)
}
```

**Step 3: Update PostAuthSignup handler**

Modify `auth_signup.go` `PostAuthSignup()` to use the new `SignupResult`:
- On success, return `res` which now includes `temp_token`
- On failure, return `gin.H{"message": "Verification code sent to email."}` (no temp_token)

**Step 4: Update PostAuthEmailVerify handler**

Modify `auth_signup.go` `PostAuthEmailVerify()` to return the `EmailVerifyResult` (includes accesskey).

**Step 5: Update the email verify HTML page**

Modify `emailVerifyHTML` constant to display the AccessKey after successful verification. Update the success handler in the JavaScript `fetch().then()`:

```javascript
if (resp.ok) {
    resp.json().then(function(data) {
        var msg = 'Email verified successfully!';
        if (data.accesskey && data.accesskey.token) {
            msg += ' Your API Key: ' + data.accesskey.token + ' (save this — it will not be shown again)';
        }
        msg += ' Check your inbox for a welcome email with instructions to set your password.';
        msgEl.textContent = msg;
        msgEl.className = 'message success';
        btn.style.display = 'none';
    });
}
```

**Step 6: Run verification and commit**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-api-manager/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-api-manager: Add POST /auth/complete-signup HTTP handler
- bin-api-manager: Update PostAuthSignup to return temp_token
- bin-api-manager: Update PostAuthEmailVerify to return AccessKey
- bin-api-manager: Update email verify HTML to display AccessKey"
```

---

### Task 9: Suppress welcome email for headless signups in bin-agent-manager

**Files:**
- Modify: `bin-agent-manager/pkg/subscribehandler/customer_manager.go`
- Modify: `bin-agent-manager/pkg/agenthandler/event.go`

**Context:** The `customer_created` event now carries a `CustomerCreatedEvent` wrapper with a `headless` field. The agent-manager subscribe handler unmarshals this as a `Customer` struct — the extra `headless` field will be ignored by default. We need to parse the new field and pass it to `EventCustomerCreated`.

**Step 1: Update the event handler signature**

In `bin-agent-manager/pkg/agenthandler/event.go`, change:

```go
// Old:
func (h *agentHandler) EventCustomerCreated(ctx context.Context, cu *cmcustomer.Customer) error {

// New:
func (h *agentHandler) EventCustomerCreated(ctx context.Context, cu *cmcustomer.Customer, headless bool) error {
```

Add the conditional at the end:

```go
// Send welcome email with password reset link — only for non-headless signups
if !headless {
    if err := h.PasswordForgot(ctx, cu.Email, PasswordResetEmailTypeWelcome); err != nil {
        log.Errorf("Could not send welcome email. err: %v", err)
    }
}
```

**Step 2: Update the subscribe handler to parse headless flag**

In `subscribehandler/customer_manager.go`, change:

```go
func (h *subscribeHandler) processEventCMCustomerCreated(ctx context.Context, m *sock.Event) error {
    // Parse with headless field
    var event struct {
        cmcustomer.Customer
        Headless bool `json:"headless"`
    }
    if err := json.Unmarshal([]byte(m.Data), &event); err != nil {
        log.Errorf("Could not unmarshal the data. err: %v", err)
        return err
    }

    cu := &event.Customer
    if errEvent := h.agentHandler.EventCustomerCreated(ctx, cu, event.Headless); errEvent != nil {
        // ...
    }
    return nil
}
```

**Step 3: Update the AgentHandler interface**

In `bin-agent-manager/pkg/agenthandler/main.go`, update the interface:

```go
EventCustomerCreated(ctx context.Context, cu *cmcustomer.Customer, headless bool) error
```

**Step 4: Update tests**

All existing tests calling `EventCustomerCreated` need the new `headless` parameter added. Existing tests should pass `false` to maintain current behavior.

**Step 5: Run verification and commit**

```bash
cd bin-agent-manager && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-agent-manager/
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- bin-agent-manager: Skip welcome email for headless signups (headless=true)
- bin-agent-manager: Parse headless flag from customer_created event payload"
```

---

### Task 10: Cross-service integration verification

**Files:** All changed services

**Context:** Multiple services were modified. Run the full verification workflow on each changed service to ensure everything compiles and tests pass together.

**Step 1: Vendor and verify bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Vendor and verify bin-customer-manager**

```bash
cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Vendor and verify bin-agent-manager**

```bash
cd bin-agent-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Vendor and verify bin-openapi-manager**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./...
```

**Step 5: Vendor and verify bin-api-manager**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Check for other services that import bin-common-handler requesthandler**

The requesthandler interface changed (new method + modified return types). Any service that imports and mocks `requesthandler.RequestHandler` will need `go mod vendor && go generate ./...` to regenerate mocks. Run a search:

```bash
grep -rl "requesthandler.NewMockRequestHandler\|requesthandler.MockRequestHandler" --include="*.go" | grep -v vendor
```

For each service found, run:

```bash
cd bin-<service> && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit any cross-service fixups**

```bash
git add -A
git commit -m "NOJIRA-Headless-signup-auto-provisioning

- Cross-service verification: regenerate mocks and vendor for all affected services"
```

---

### Task 11: Final commit squash and PR

**Step 1: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

If conflicts exist, rebase and resolve before proceeding.

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-Headless-signup-auto-provisioning
```

Create PR following the project conventions. Do NOT include AI attribution.
