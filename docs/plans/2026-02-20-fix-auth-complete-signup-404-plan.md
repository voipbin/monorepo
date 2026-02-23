# Fix /auth/complete-signup 404 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the 404 on `POST /auth/complete-signup` by adding a public route handler, and remove the redundant `/auth/*` routes from the OpenAPI spec.

**Architecture:** Add a legacy-style handler in `lib/service/signup.go` and register it as a public route in `main.go`. Remove all `/auth/*` paths from the OpenAPI spec and delete the corresponding OpenAPI-generated handlers in `server/`. Regenerate code for both `bin-openapi-manager` and `bin-api-manager`.

**Tech Stack:** Go, Gin HTTP framework, oapi-codegen

**Design doc:** `docs/plans/2026-02-20-fix-auth-complete-signup-404-design.md`

---

### Task 1: Add PostCustomerCompleteSignup handler and register public route

**Files:**
- Modify: `bin-api-manager/lib/service/signup.go` (add handler + request struct at end of file)
- Modify: `bin-api-manager/cmd/api-manager/main.go:221` (add route registration)

**Step 1: Add the handler to signup.go**

Append the following after the `GetCustomerEmailVerify` function (before the `emailVerifyHTML` const), at the end of the handler functions section in `bin-api-manager/lib/service/signup.go`:

```go
// RequestBodyCompleteSignupPOST is request body for POST /auth/complete-signup
type RequestBodyCompleteSignupPOST struct {
	TempToken string `json:"temp_token" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

// PostCustomerCompleteSignup handles POST /auth/complete-signup request.
// It validates an OTP code and completes the headless signup flow.
func PostCustomerCompleteSignup(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCustomerCompleteSignup",
		"request_address": c.ClientIP,
	})
	log.Debug("Processing complete signup.")

	var req RequestBodyCompleteSignupPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	sh := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := sh.CustomerCompleteSignup(c.Request.Context(), req.TempToken, req.Code)
	if err != nil {
		log.Debugf("Complete signup failed. err: %v", err)
		if errors.Is(err, requesthandler.ErrTooManyRequests) {
			c.AbortWithStatus(429)
			return
		}
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

Also add the new imports to `signup.go`. The existing imports are:

```go
import (
	"fmt"
	"regexp"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)
```

Change to:

```go
import (
	"errors"
	"fmt"
	"regexp"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)
```

**Step 2: Register the route in main.go**

In `bin-api-manager/cmd/api-manager/main.go`, find line 221 (after `auth.POST("/email-verify", service.PostCustomerEmailVerify)`) and add the new route:

```go
	auth.POST("/complete-signup", service.PostCustomerCompleteSignup)
```

So lines 219-222 become:

```go
	auth.POST("/signup", service.PostCustomerSignup)
	auth.GET("/email-verify", service.GetCustomerEmailVerify)
	auth.POST("/email-verify", service.PostCustomerEmailVerify)
	auth.POST("/complete-signup", service.PostCustomerCompleteSignup)
```

**Step 3: Add unit tests**

Add tests to `bin-api-manager/lib/service/signup_test.go`. Append:

```go
func TestPostCustomerCompleteSignup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		reqBody      RequestBodyCompleteSignupPOST
		mockSetup    func(*servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name: "valid complete signup",
			reqBody: RequestBodyCompleteSignupPOST{
				TempToken: "tmp_abcdef123",
				Code:      "123456",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerCompleteSignup(
					gomock.Any(),
					"tmp_abcdef123",
					"123456",
				).Return(&cscustomer.CompleteSignupResult{
					CustomerID: "550e8400-e29b-41d4-a716-446655440000",
				}, nil)
			},
			expectStatus: 200,
		},
		{
			name: "invalid temp token",
			reqBody: RequestBodyCompleteSignupPOST{
				TempToken: "invalid_token",
				Code:      "123456",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerCompleteSignup(
					gomock.Any(),
					"invalid_token",
					"123456",
				).Return(nil, errors.New("invalid token"))
			},
			expectStatus: 400,
		},
		{
			name: "rate limited",
			reqBody: RequestBodyCompleteSignupPOST{
				TempToken: "tmp_ratelimit",
				Code:      "000000",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerCompleteSignup(
					gomock.Any(),
					"tmp_ratelimit",
					"000000",
				).Return(nil, requesthandler.ErrTooManyRequests)
			},
			expectStatus: 429,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
			})
			r.POST("/auth/complete-signup", PostCustomerCompleteSignup)

			tt.mockSetup(mockSvc)

			body, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/auth/complete-signup", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestPostCustomerCompleteSignup_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	r.POST("/auth/complete-signup", PostCustomerCompleteSignup)

	req, _ := http.NewRequest("POST", "/auth/complete-signup", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got: %d", w.Code)
	}
}
```

Also add `requesthandler` import to `signup_test.go`. The existing imports are:

```go
import (
	"bytes"
	"encoding/json"
	"errors"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)
```

Change to:

```go
import (
	"bytes"
	"encoding/json"
	"errors"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)
```

**Step 4: Run tests to verify**

Run: `cd bin-api-manager && go test -v ./lib/service/...`
Expected: All tests pass including the new `TestPostCustomerCompleteSignup*` tests

**Step 5: Commit**

```
git add bin-api-manager/lib/service/signup.go bin-api-manager/lib/service/signup_test.go bin-api-manager/cmd/api-manager/main.go
git commit -m "NOJIRA-fix-auth-complete-signup-404

- bin-api-manager: Add PostCustomerCompleteSignup handler in lib/service/signup.go
- bin-api-manager: Register POST /auth/complete-signup as public route in main.go
- bin-api-manager: Add unit tests for complete-signup handler (200, 400, 429)"
```

---

### Task 2: Remove /auth/* paths from OpenAPI spec and delete orphaned schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (remove path entries + AuthLoginResponse schema)
- Delete: `bin-openapi-manager/openapi/paths/auth/complete-signup.yaml`
- Delete: `bin-openapi-manager/openapi/paths/auth/email-verify.yaml`
- Delete: `bin-openapi-manager/openapi/paths/auth/login.yaml`
- Delete: `bin-openapi-manager/openapi/paths/auth/password-forgot.yaml`
- Delete: `bin-openapi-manager/openapi/paths/auth/password-reset.yaml`
- Delete: `bin-openapi-manager/openapi/paths/auth/signup.yaml`

**Step 1: Remove path entries from openapi.yaml**

In `bin-openapi-manager/openapi/openapi.yaml`, remove lines 5819-5830 (the 6 `/auth/*` path references):

```yaml
  /auth/login:
    $ref: './paths/auth/login.yaml'
  /auth/password-forgot:
    $ref: './paths/auth/password-forgot.yaml'
  /auth/password-reset:
    $ref: './paths/auth/password-reset.yaml'
  /auth/signup:
    $ref: './paths/auth/signup.yaml'
  /auth/email-verify:
    $ref: './paths/auth/email-verify.yaml'
  /auth/complete-signup:
    $ref: './paths/auth/complete-signup.yaml'
```

**Step 2: Remove the AuthLoginResponse schema**

In `bin-openapi-manager/openapi/openapi.yaml`, remove lines 224-240:

```yaml
#########################################
# Auth
#########################################
    AuthLoginResponse:
      type: object
      description: Authentication response containing JWT token
      required:
        - username
        - token
      properties:
        username:
          type: string
          description: The authenticated username
          example: "user@example.com"
        token:
          type: string
          description: JWT token for API access
          example: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**Step 3: Delete path files**

```bash
rm bin-openapi-manager/openapi/paths/auth/complete-signup.yaml
rm bin-openapi-manager/openapi/paths/auth/email-verify.yaml
rm bin-openapi-manager/openapi/paths/auth/login.yaml
rm bin-openapi-manager/openapi/paths/auth/password-forgot.yaml
rm bin-openapi-manager/openapi/paths/auth/password-reset.yaml
rm bin-openapi-manager/openapi/paths/auth/signup.yaml
rmdir bin-openapi-manager/openapi/paths/auth
```

**Step 4: Regenerate openapi-manager models**

Run: `cd bin-openapi-manager && go generate ./...`
Expected: Regeneration succeeds, `gens/models/gen.go` no longer contains `AuthLoginResponse`

**Step 5: Run verification for bin-openapi-manager**

Run: `cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 6: Commit**

```
git add -A bin-openapi-manager/
git commit -m "NOJIRA-fix-auth-complete-signup-404

- bin-openapi-manager: Remove /auth/* paths from OpenAPI spec (login, signup, complete-signup, email-verify, password-forgot, password-reset)
- bin-openapi-manager: Delete auth path YAML files
- bin-openapi-manager: Remove orphaned AuthLoginResponse schema"
```

---

### Task 3: Delete OpenAPI auth handlers from bin-api-manager server package

**Files:**
- Delete: `bin-api-manager/server/auth.go`
- Delete: `bin-api-manager/server/auth_test.go`
- Delete: `bin-api-manager/server/auth_password_test.go`
- Delete: `bin-api-manager/server/auth_signup.go`
- Delete: `bin-api-manager/server/auth_signup_test.go`
- Delete: `bin-api-manager/server/auth_complete_signup.go`
- Delete: `bin-api-manager/server/auth_complete_signup_test.go`

**Step 1: Delete the files**

```bash
rm bin-api-manager/server/auth.go
rm bin-api-manager/server/auth_test.go
rm bin-api-manager/server/auth_password_test.go
rm bin-api-manager/server/auth_signup.go
rm bin-api-manager/server/auth_signup_test.go
rm bin-api-manager/server/auth_complete_signup.go
rm bin-api-manager/server/auth_complete_signup_test.go
```

**Step 2: Regenerate api-manager code and vendor**

The OpenAPI spec no longer has auth paths, so the generated `gen.go` will no longer have `PostAuthLogin`, `PostAuthSignup`, etc. in the `ServerInterface`. After regeneration, the `server` struct no longer needs to implement those methods.

Run: `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./...`
Expected: Regeneration succeeds. The `ServerInterface` in `gens/openapi_server/gen.go` no longer requires auth handler methods.

**Step 3: Run full verification for bin-api-manager**

Run: `cd bin-api-manager && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All tests pass, no lint errors

**Step 4: Commit**

```
git add -A bin-api-manager/
git commit -m "NOJIRA-fix-auth-complete-signup-404

- bin-api-manager: Delete OpenAPI auth handlers (auth.go, auth_signup.go, auth_complete_signup.go and their tests)
- bin-api-manager: Regenerate OpenAPI server code without /auth/* routes"
```

---

### Task 4: Verify the fix end-to-end

**Step 1: Run the E2E test that was failing**

This test cannot pass against the live API until the fix is deployed, but verify it no longer fails with 404 by checking that the public route is correctly wired up.

Run the unit tests one more time to confirm everything is clean:

```bash
cd bin-api-manager && go test -v ./lib/service/... -run TestPostCustomerCompleteSignup
```

Expected: All 4 test cases pass (valid, invalid token, rate limited, invalid body)

**Step 2: Verify no regressions in either service**

```bash
cd bin-openapi-manager && go test ./...
cd bin-api-manager && go test ./...
```

Expected: All tests pass in both services

**Step 3: Final commit with all changes if needed**

If any fixups were needed, commit them. Otherwise this step is a no-op.
