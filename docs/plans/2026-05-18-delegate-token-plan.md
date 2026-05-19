# Delegate Token Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `POST /auth/delegate` so a platform superadmin can mint a short-lived JWT granting full customer-admin access to a specific target customer.

**Architecture:** New `TypeDelegate` in `models/auth/auth.go` + new case in `buildJWTIdentity` middleware + new `AuthDelegate` in servicehandler + new Gin handler registered under `/auth`. Reuses existing JWT signing infrastructure (`authJWTGenerateWithExpiration`) and customer RPC. No DB migration needed.

**Tech Stack:** Go, Gin, golang-jwt/jwt/v5, logrus, Prometheus (promauto), gofrs/uuid, go:generate mockgen

---

## Prerequisite: Work in the Worktree

All work happens in the feature worktree. Verify before every task:
```bash
pwd
# expected: /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-delegate-token
```

---

## Task 1: Add `TypeDelegate` and `DelegateScope` to `models/auth/auth.go`

**Files:**
- Modify: `bin-api-manager/models/auth/auth.go`
- Test: `bin-api-manager/models/auth/auth_test.go`

### Step 1: Write failing tests

Add to `bin-api-manager/models/auth/auth_test.go` (after the existing `Test_HasPermission` function):

```go
func Test_NewDelegateIdentity(t *testing.T) {
	type test struct {
		name             string
		scope            *DelegateScope
		expectType       Type
		expectCustomerID uuid.UUID
		expectJTI        string
	}

	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")

	tests := []test{
		{
			"normal",
			&DelegateScope{
				CustomerID: customerID,
				IssuedBy:   uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				JTI:        "jti-value-123",
			},
			TypeDelegate,
			customerID,
			"jti-value-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := NewDelegateIdentity(tt.scope)
			if res.Type != tt.expectType {
				t.Errorf("Wrong Type. expect: %v, got: %v", tt.expectType, res.Type)
			}
			if res.CustomerID != tt.expectCustomerID {
				t.Errorf("Wrong CustomerID. expect: %v, got: %v", tt.expectCustomerID, res.CustomerID)
			}
			if res.DelegateScope == nil {
				t.Fatal("DelegateScope is nil")
			}
			if res.DelegateScope.JTI != tt.expectJTI {
				t.Errorf("Wrong JTI. expect: %v, got: %v", tt.expectJTI, res.DelegateScope.JTI)
			}
		})
	}
}

func Test_HasPermission_Delegate(t *testing.T) {
	type test struct {
		name       string
		identity   AuthIdentity
		permission amagent.Permission
		expectRes  bool
	}

	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")

	tests := []test{
		{
			"delegate grants PermissionCustomerAdmin",
			AuthIdentity{
				Type:       TypeDelegate,
				CustomerID: customerID,
				DelegateScope: &DelegateScope{
					CustomerID: customerID,
				},
			},
			amagent.PermissionCustomerAdmin,
			true,
		},
		{
			"delegate denies PermissionProjectSuperAdmin",
			AuthIdentity{
				Type:       TypeDelegate,
				CustomerID: customerID,
				DelegateScope: &DelegateScope{
					CustomerID: customerID,
				},
			},
			amagent.PermissionProjectSuperAdmin,
			false,
		},
		{
			"delegate denies PermissionProjectAll",
			AuthIdentity{
				Type:       TypeDelegate,
				CustomerID: customerID,
				DelegateScope: &DelegateScope{
					CustomerID: customerID,
				},
			},
			amagent.PermissionProjectAll,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.HasPermission(tt.permission)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_IsDelegate(t *testing.T) {
	type test struct {
		name      string
		identity  AuthIdentity
		expectRes bool
	}

	tests := []test{
		{"delegate type returns true", AuthIdentity{Type: TypeDelegate}, true},
		{"agent type returns false", AuthIdentity{Type: TypeAgent}, false},
		{"accesskey type returns false", AuthIdentity{Type: TypeAccesskey}, false},
		{"direct type returns false", AuthIdentity{Type: TypeDirect}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.IsDelegate()
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
```

### Step 2: Run tests — expect FAIL

```bash
cd bin-api-manager
go test ./models/auth/... -run "Test_NewDelegateIdentity|Test_HasPermission_Delegate|Test_IsDelegate" -v
```
Expected: FAIL — `TypeDelegate`, `DelegateScope`, `NewDelegateIdentity`, `IsDelegate` undefined.

### Step 3: Implement in `models/auth/auth.go`

**Add after the existing constants block:**
```go
	TypeDelegate  Type = "delegate"
```

**Add `DelegateScope` struct after `DirectScope`:**
```go
// DelegateScope represents a superadmin-issued delegate JWT claim set.
type DelegateScope struct {
	CustomerID uuid.UUID `json:"customer_id"`
	IssuedBy   uuid.UUID `json:"issued_by"`
	JTI        string    `json:"jti"`
}
```

**Add `DelegateScope *DelegateScope` field to `AuthIdentity`** (after `DirectScope`):
```go
	DelegateScope *DelegateScope  // non-nil for TypeDelegate
```

**Add to `HasPermission()` switch** (before the `default` case):
```go
	case TypeDelegate:
		// Delegate tokens have PermissionCustomerAdmin-equivalent access.
		// Explicitly excludes project-level permissions.
		return (amagent.PermissionCustomerAdmin & p) != 0 &&
			(p&(amagent.PermissionProjectSuperAdmin|amagent.PermissionProjectAll)) == 0
```

**Add `IsDelegate()` method** (after `IsDirect()`):
```go
// IsDelegate returns true if the identity was created from a delegate JWT.
func (a *AuthIdentity) IsDelegate() bool {
	return a.Type == TypeDelegate
}
```

**Add to `DisplayName()` switch** (before `default`):
```go
	case TypeDelegate:
		if a.DelegateScope != nil {
			return "delegate:" + a.DelegateScope.CustomerID.String()
		}
		return "delegate"
```

**Add constructor** (after `NewDirectIdentity`):
```go
// NewDelegateIdentity constructs an AuthIdentity from a delegate scope.
func NewDelegateIdentity(scope *DelegateScope) *AuthIdentity {
	return &AuthIdentity{
		Type:          TypeDelegate,
		CustomerID:    scope.CustomerID,
		DelegateScope: scope,
	}
}
```

### Step 4: Run tests — expect PASS

```bash
go test ./models/auth/... -v
```
Expected: all tests PASS.

### Step 5: Commit

```bash
cd bin-api-manager
git add models/auth/auth.go models/auth/auth_test.go
git commit -m "NOJIRA-Add-delegate-token

- bin-api-manager: Add TypeDelegate, DelegateScope, NewDelegateIdentity, IsDelegate"
```

---

## Task 2: Update `lib/middleware/authenticate.go` for `TypeDelegate`

**Files:**
- Modify: `bin-api-manager/lib/middleware/authenticate.go`
- Test: `bin-api-manager/lib/middleware/authenticate_test.go`

### Step 1: Write failing tests

Open `bin-api-manager/lib/middleware/authenticate_test.go` and read its existing structure to understand the test harness. Then add:

```go
func Test_buildJWTIdentity_Delegate(t *testing.T) {
	type test struct {
		name           string
		authData       map[string]interface{}
		expectErr      bool
		expectType     auth.Type
		expectCustomer string
		expectJTI      string
	}

	customerID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	issuedBy   := "11111111-1111-1111-1111-111111111111"
	jti        := "some-jti-value"

	tests := []test{
		{
			name: "valid delegate token",
			authData: map[string]interface{}{
				"type":        "delegate",
				"aud":         "voipbin-api",
				"customer_id": customerID,
				"sub":         issuedBy,
				"jti":         jti,
			},
			expectErr:      false,
			expectType:     auth.TypeDelegate,
			expectCustomer: customerID,
			expectJTI:      jti,
		},
		{
			name: "delegate token wrong aud rejected",
			authData: map[string]interface{}{
				"type":        "delegate",
				"aud":         "wrong-audience",
				"customer_id": customerID,
				"sub":         issuedBy,
				"jti":         jti,
			},
			expectErr: true,
		},
		{
			name: "delegate token missing customer_id rejected",
			authData: map[string]interface{}{
				"type": "delegate",
				"aud":  "voipbin-api",
				"sub":  issuedBy,
				"jti":  jti,
			},
			expectErr: true,
		},
		{
			name: "unknown token type returns error",
			authData: map[string]interface{}{
				"type": "unknown-type",
			},
			expectErr: true,
		},
	}

	log := logrus.NewEntry(logrus.New())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := buildJWTIdentity(log, tt.authData)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Type != tt.expectType {
				t.Errorf("Wrong Type. expect: %v, got: %v", tt.expectType, res.Type)
			}
			if res.CustomerID.String() != tt.expectCustomer {
				t.Errorf("Wrong CustomerID. expect: %v, got: %v", tt.expectCustomer, res.CustomerID)
			}
			if tt.expectJTI != "" && res.DelegateScope.JTI != tt.expectJTI {
				t.Errorf("Wrong JTI. expect: %v, got: %v", tt.expectJTI, res.DelegateScope.JTI)
			}
		})
	}
}
```

### Step 2: Run tests — expect FAIL

```bash
go test ./lib/middleware/... -run "Test_buildJWTIdentity_Delegate" -v
```
Expected: FAIL — delegate case not handled; unknown type returns nil instead of error.

### Step 3: Implement in `lib/middleware/authenticate.go`

In `buildJWTIdentity()`, **replace the current `switch` block**:

```go
func buildJWTIdentity(log *logrus.Entry, authData map[string]interface{}) (*auth.AuthIdentity, error) {
	tokenType, _ := authData["type"].(string)

	switch tokenType {
	case string(auth.TypeDirect):
		raw, ok := authData["direct"]
		if !ok {
			return nil, fmt.Errorf("direct token missing direct scope")
		}
		buf, err := json.Marshal(raw)
		if err != nil {
			log.Errorf("Could not marshal direct scope. err: %v", err)
			return nil, fmt.Errorf("invalid direct scope")
		}
		var scope auth.DirectScope
		if err := json.Unmarshal(buf, &scope); err != nil {
			log.Errorf("Could not unmarshal direct scope. err: %v", err)
			return nil, fmt.Errorf("invalid direct scope")
		}
		return auth.NewDirectIdentity(&scope), nil

	case string(auth.TypeDelegate):
		// Enforce audience — only for delegate tokens (TypeAgent/TypeDirect predate aud claim)
		if aud, _ := authData["aud"].(string); aud != delegateAudience {
			return nil, fmt.Errorf("delegate token: invalid audience %q", aud)
		}
		customerIDStr, ok := authData["customer_id"].(string)
		if !ok || customerIDStr == "" {
			return nil, fmt.Errorf("delegate token missing customer_id")
		}
		customerID, err := uuid.FromString(customerIDStr)
		if err != nil {
			return nil, fmt.Errorf("delegate token: invalid customer_id: %w", err)
		}
		issuedByStr, _ := authData["sub"].(string)
		issuedBy, _ := uuid.FromString(issuedByStr)
		jti, _ := authData["jti"].(string)

		scope := &auth.DelegateScope{
			CustomerID: customerID,
			IssuedBy:   issuedBy,
			JTI:        jti,
		}
		return auth.NewDelegateIdentity(scope), nil

	case string(auth.TypeAgent), "":
		// "agent" or missing type (backward compat) — treat as agent token
		raw, ok := authData["agent"]
		if !ok {
			return nil, fmt.Errorf("token missing agent data")
		}
		buf, err := json.Marshal(raw)
		if err != nil {
			log.Errorf("Could not marshal agent data. err: %v", err)
			return nil, fmt.Errorf("invalid agent data")
		}
		var a amagent.Agent
		if err := json.Unmarshal(buf, &a); err != nil {
			log.Errorf("Could not unmarshal agent data. err: %v", err)
			return nil, fmt.Errorf("invalid agent data")
		}
		return auth.NewAgentIdentity(&a), nil

	default:
		return nil, fmt.Errorf("unknown token type: %q", tokenType)
	}
}
```

**Add constant** at the top of the file, near the other constants:
```go
const (
	// ...existing constants...
	delegateAudience = "voipbin-api"
)
```

**Add per-request delegate tracing** in `Authenticate()`, after `c.Set("auth_identity", identity)`:
```go
		// For delegate tokens, annotate the logger with delegate context for tracing
		if identity.IsDelegate() && identity.DelegateScope != nil {
			c.Set("delegate_jti", identity.DelegateScope.JTI)
		}
```

### Step 4: Run tests — expect PASS

```bash
go test ./lib/middleware/... -v
```
Expected: all tests PASS.

### Step 5: Commit

```bash
git add lib/middleware/authenticate.go lib/middleware/authenticate_test.go
git commit -m "NOJIRA-Add-delegate-token

- bin-api-manager: Handle TypeDelegate in JWT middleware with aud enforcement and request tracing"
```

---

## Task 3: Add `AuthDelegate` to servicehandler

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (add to interface + constant)
- Create: `bin-api-manager/pkg/servicehandler/auth_delegate.go`
- Create: `bin-api-manager/pkg/servicehandler/auth_delegate_test.go`
- Regenerate: `bin-api-manager/pkg/servicehandler/mock_main.go`

### Step 1: Add constant and interface method

In `bin-api-manager/pkg/servicehandler/main.go`:

**Add constant** (after `BootExpiration`):
```go
	DelegateExpiration = time.Hour * 8 // delegate token expiration. 8 hours
```

**Add to `ServiceHandler` interface** (in the `// auth handlers` section, after `AuthBoot`):
```go
	AuthDelegate(ctx context.Context, a *auth.AuthIdentity, targetCustomerID uuid.UUID, reason string) (*DelegateResponse, error)
```

### Step 2: Write failing test

Create `bin-api-manager/pkg/servicehandler/auth_delegate_test.go`:

```go
package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AuthDelegate(t *testing.T) {
	superAdminID  := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	targetCustomerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	superAdminIdentity := &auth.AuthIdentity{
		Type:       auth.TypeAgent,
		CustomerID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		Agent: &amagent.Agent{
			Identity:   commonidentity.Identity{ID: superAdminID},
			Permission: amagent.PermissionProjectSuperAdmin,
		},
	}

	delegateIdentity := &auth.AuthIdentity{
		Type:       auth.TypeDelegate,
		CustomerID: targetCustomerID,
		DelegateScope: &auth.DelegateScope{
			CustomerID: targetCustomerID,
		},
	}

	activeCustomer := &cscustomer.Customer{
		Identity: commonidentity.Identity{ID: targetCustomerID},
		Status:   cscustomer.StatusActive,
	}

	tests := []struct {
		name             string
		identity         *auth.AuthIdentity
		targetCustomerID uuid.UUID
		reason           string
		mockCustomer     *cscustomer.Customer
		mockCustomerErr  error
		expectErr        bool
	}{
		{
			name:             "superadmin gets delegate token",
			identity:         superAdminIdentity,
			targetCustomerID: targetCustomerID,
			reason:           "investigating dropped call for customer",
			mockCustomer:     activeCustomer,
			expectErr:        false,
		},
		{
			name:             "TypeDelegate caller rejected (recursive delegation)",
			identity:         delegateIdentity,
			targetCustomerID: targetCustomerID,
			reason:           "investigating dropped call for customer",
			expectErr:        true,
		},
		{
			name: "non-superadmin rejected",
			identity: &auth.AuthIdentity{
				Type: auth.TypeAgent,
				Agent: &amagent.Agent{
					Permission: amagent.PermissionCustomerAdmin,
				},
			},
			targetCustomerID: targetCustomerID,
			reason:           "investigating dropped call for customer",
			expectErr:        true,
		},
		{
			name:             "customer not found returns error",
			identity:         superAdminIdentity,
			targetCustomerID: targetCustomerID,
			reason:           "investigating dropped call for customer",
			mockCustomerErr:  fmt.Errorf("not found"),
			expectErr:        true,
		},
		{
			name:             "reason too short rejected",
			identity:         superAdminIdentity,
			targetCustomerID: targetCustomerID,
			reason:           "short",
			expectErr:        true,
		},
		{
			name:             "reason with control char rejected",
			identity:         superAdminIdentity,
			targetCustomerID: targetCustomerID,
			reason:           "valid reason\nwith newline",
			expectErr:        true,
		},
		{
			name:             "reason too long rejected",
			identity:         superAdminIdentity,
			targetCustomerID: targetCustomerID,
			reason:           string(make([]byte, 201)),
			expectErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}

			if !tt.expectErr || tt.mockCustomer != nil || tt.mockCustomerErr != nil {
				// Only expect RPC call if we pass the early guards
				if tt.identity.Type != auth.TypeDelegate &&
					tt.identity.HasPermission(amagent.PermissionProjectSuperAdmin) &&
					len(tt.reason) >= 10 && len(tt.reason) <= 200 {
					mockUtil.EXPECT().TimeGetCurTimeAdd(gomock.Any()).Return("2026-05-19T06:00:00Z").AnyTimes()
					if tt.mockCustomerErr != nil {
						mockReq.EXPECT().CustomerV1CustomerGet(gomock.Any(), tt.targetCustomerID).Return(nil, tt.mockCustomerErr)
					} else if tt.mockCustomer != nil {
						mockReq.EXPECT().CustomerV1CustomerGet(gomock.Any(), tt.targetCustomerID).Return(tt.mockCustomer, nil)
					}
				}
			}

			res, err := h.AuthDelegate(context.Background(), tt.identity, tt.targetCustomerID, tt.reason)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res == nil {
				t.Fatal("expected non-nil response")
			}
			if res.Token == "" {
				t.Error("expected non-empty token")
			}
			if res.CustomerID != tt.targetCustomerID {
				t.Errorf("Wrong CustomerID. expect: %v, got: %v", tt.targetCustomerID, res.CustomerID)
			}
		})
	}
}
```

### Step 3: Run tests — expect FAIL

```bash
go test ./pkg/servicehandler/... -run "Test_AuthDelegate" -v
```
Expected: FAIL — `AuthDelegate` undefined, `DelegateResponse` undefined.

### Step 4: Implement `auth_delegate.go`

Create `bin-api-manager/pkg/servicehandler/auth_delegate.go`:

```go
package servicehandler

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	metricDelegateIssued = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_delegate_token_issued_total",
		Help: "Total number of delegate tokens issued.",
	}, []string{"issued_by_agent_id"})
)

// DelegateResponse is the typed response for POST /auth/delegate.
type DelegateResponse struct {
	Token      string    `json:"token"`
	CustomerID uuid.UUID `json:"customer_id"`
	Expire     string    `json:"expire"`
}

// AuthDelegate issues a short-lived JWT granting PermissionCustomerAdmin-equivalent
// access scoped to targetCustomerID. Only PermissionProjectSuperAdmin agents may call this.
func (h *serviceHandler) AuthDelegate(ctx context.Context, a *auth.AuthIdentity, targetCustomerID uuid.UUID, reason string) (*DelegateResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "AuthDelegate",
		"target_customer_id": targetCustomerID,
	})

	// Step 1: Block recursive delegation
	if a.IsDelegate() {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"denial_reason": "recursive_delegation",
		}).Warn("Delegate token request denied")
		return nil, fmt.Errorf("%w: recursive delegation not permitted", serviceerrors.ErrPermissionDenied)
	}

	// Step 2: Verify caller has PermissionProjectSuperAdmin
	if !a.HasPermission(amagent.PermissionProjectSuperAdmin) {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"sub":           a.AgentID(),
			"denial_reason": "not_superadmin",
		}).Warn("Delegate token request denied")
		return nil, fmt.Errorf("%w: PermissionProjectSuperAdmin required", serviceerrors.ErrPermissionDenied)
	}

	agentID := a.AgentID()

	// Step 3: Validate reason
	if err := validateDelegateReason(reason); err != nil {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"sub":           agentID,
			"denial_reason": "invalid_input",
		}).Warn("Delegate token request denied")
		return nil, fmt.Errorf("%w: %v", serviceerrors.ErrInvalidArgument, err)
	}

	// Step 4: Verify target customer exists and is not deleted
	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, targetCustomerID)
	if err != nil {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"sub":           agentID,
			"denial_reason": "customer_not_found",
		}).Warn("Delegate token request denied")
		return nil, fmt.Errorf("%w: target customer not found", serviceerrors.ErrNotFound)
	}
	if cu.Status == cscustomer.StatusDeleted {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"sub":           agentID,
			"denial_reason": "customer_not_found",
		}).Warn("Delegate token request denied — customer is deleted")
		return nil, fmt.Errorf("%w: target customer not found", serviceerrors.ErrNotFound)
	}

	// Step 5: Generate jti
	jti, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Could not generate jti. err: %v", err)
		return nil, fmt.Errorf("%w: jti generation failed", serviceerrors.ErrInternal)
	}

	// Step 6: Generate JWT
	data := map[string]interface{}{
		"type":        string(auth.TypeDelegate),
		"sub":         agentID.String(),
		"aud":         "voipbin-api",
		"jti":         jti.String(),
		"customer_id": targetCustomerID.String(),
	}
	token, expire, err := h.authJWTGenerateWithExpiration(data, DelegateExpiration)
	if err != nil {
		log.Errorf("Could not generate delegate JWT. err: %v", err)
		return nil, fmt.Errorf("%w: token generation failed", serviceerrors.ErrInternal)
	}

	// Step 7: Write audit log
	log.WithFields(logrus.Fields{
		"audit":               true,
		"event":              "delegate_token_issued",
		"jti":                jti.String(),
		"sub":                agentID,
		"target_customer_id": targetCustomerID,
		"reason":             reason,
		"expire":             expire,
	}).Info("Delegate token issued")

	// Step 8: Emit metric
	metricDelegateIssued.WithLabelValues(agentID.String()).Inc()

	return &DelegateResponse{
		Token:      token,
		CustomerID: targetCustomerID,
		Expire:     expire,
	}, nil
}

// validateDelegateReason enforces reason field constraints: 10–200 printable chars, no control chars.
func validateDelegateReason(reason string) error {
	if len(reason) < 10 {
		return fmt.Errorf("reason must be at least 10 characters")
	}
	if len(reason) > 200 {
		return fmt.Errorf("reason must be at most 200 characters")
	}
	for _, r := range reason {
		if unicode.IsControl(r) {
			return fmt.Errorf("reason must not contain control characters")
		}
	}
	// Allow printable ASCII + space; reject non-ASCII for simplicity
	for _, r := range reason {
		if r > unicode.MaxASCII || (!unicode.IsPrint(r) && r != ' ') {
			return fmt.Errorf("reason must contain only printable ASCII characters")
		}
	}
	return nil
}

// Note: validateDelegateReason intentionally uses len() on the string (byte count),
// not utf8.RuneCountInString(), because reason is restricted to ASCII — byte count == rune count.
var _ = strings.TrimSpace // imported for potential future use; remove if unused after lint
```

**Note:** Remove the `strings` import if unused after implementing. Check with `go build`.

### Step 5: Regenerate mock

```bash
cd bin-api-manager
go generate ./pkg/servicehandler/...
```

### Step 6: Run tests — expect PASS

```bash
go test ./pkg/servicehandler/... -run "Test_AuthDelegate" -v
```
Expected: all tests PASS.

### Step 7: Run full servicehandler tests

```bash
go test ./pkg/servicehandler/... -v
```
Expected: all tests PASS.

### Step 8: Commit

```bash
git add pkg/servicehandler/main.go pkg/servicehandler/auth_delegate.go pkg/servicehandler/auth_delegate_test.go pkg/servicehandler/mock_main.go
git commit -m "NOJIRA-Add-delegate-token

- bin-api-manager: Implement AuthDelegate service handler with reason validation, audit logging, and Prometheus metric"
```

---

## Task 4: Add HTTP handler `PostDelegate`

**Files:**
- Create: `bin-api-manager/lib/service/auth_delegate.go`

### Step 1: Write handler

Create `bin-api-manager/lib/service/auth_delegate.go`:

```go
package service

import (
	"net/http"

	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// RequestBodyDelegatePOST is the request body for POST /auth/delegate.
type RequestBodyDelegatePOST struct {
	CustomerID string `json:"customer_id" binding:"required"`
	Reason     string `json:"reason"      binding:"required"`
}

// PostDelegate handles POST /auth/delegate.
// Issues a short-lived delegate JWT granting customer-admin access to a specific customer.
// Requires PermissionProjectSuperAdmin.
func PostDelegate(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostDelegate",
		"request_address": c.ClientIP(),
	})

	var req RequestBodyDelegatePOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind request body. err: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	targetCustomerID, err := uuid.FromString(req.CustomerID)
	if err != nil {
		log.Warnf("Invalid customer_id. err: %v", err)
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	identity := getAuthIdentity(c)
	if identity == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	sh := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := sh.AuthDelegate(c.Request.Context(), identity, targetCustomerID, req.Reason)
	if err != nil {
		log.Infof("AuthDelegate failed. err: %v", err)
		switch {
		case serviceerrors.IsPermissionDenied(err):
			c.AbortWithStatus(http.StatusForbidden)
		case serviceerrors.IsNotFound(err):
			c.AbortWithStatus(http.StatusNotFound)
		case serviceerrors.IsInvalidArgument(err):
			c.AbortWithStatus(http.StatusUnprocessableEntity)
		default:
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	log.Infof("Delegate token issued. customer_id: %s", targetCustomerID)
	c.JSON(http.StatusOK, res)
}
```

**Check how `getAuthIdentity` is defined** — run:
```bash
grep -rn "getAuthIdentity\|auth_identity" bin-api-manager/lib/service/ bin-api-manager/server/ | head -20
```
If it's defined differently (e.g., `c.MustGet("auth_identity")`), adapt accordingly. If it doesn't exist, add a helper:
```go
func getAuthIdentity(c *gin.Context) *auth.AuthIdentity {
	v, exists := c.Get("auth_identity")
	if !exists {
		return nil
	}
	identity, ok := v.(*auth.AuthIdentity)
	if !ok {
		return nil
	}
	return identity
}
```

Also check how `serviceerrors.IsPermissionDenied` / `IsNotFound` / `IsInvalidArgument` are defined:
```bash
grep -rn "IsPermissionDenied\|IsNotFound\|IsInvalidArgument" bin-api-manager/pkg/serviceerrors/ | head -10
```
Adapt the error switch to match the actual error type helpers in this codebase.

### Step 2: Build — expect success

```bash
cd bin-api-manager
go build ./...
```

### Step 3: Commit

```bash
git add lib/service/auth_delegate.go
git commit -m "NOJIRA-Add-delegate-token

- bin-api-manager: Add PostDelegate HTTP handler for POST /auth/delegate"
```

---

## Task 5: Register the route

**Files:**
- Modify: `bin-api-manager/cmd/api-manager/main.go`

### Step 1: Register route

In `bin-api-manager/cmd/api-manager/main.go`, add to the authenticated auth group (after `authProtected.DELETE("/unregister", ...)`):

```go
	authProtected.POST("/delegate", service.PostDelegate)
```

**Important:** `POST /auth/delegate` requires authentication (the caller must be a superadmin), so it belongs in `authProtected`, NOT the unauthenticated `auth` group.

### Step 2: Build — expect success

```bash
cd bin-api-manager
go build ./...
```

### Step 3: Run all tests

```bash
go test ./... -v 2>&1 | tail -30
```
Expected: all tests PASS.

### Step 4: Commit

```bash
git add cmd/api-manager/main.go
git commit -m "NOJIRA-Add-delegate-token

- bin-api-manager: Register POST /auth/delegate route under authenticated auth group"
```

---

## Task 6: Full Verification Workflow

Run the mandatory verification workflow from the service directory:

```bash
cd bin-api-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Fix any issues before proceeding. Do NOT skip steps.

### Commit any tidy/generate changes

```bash
git add go.mod go.sum
git commit -m "NOJIRA-Add-delegate-token

- bin-api-manager: go mod tidy after delegate token implementation"
```

---

## Task 7: RST Docs Update

Per `bin-api-manager/CLAUDE.md`, user-facing API changes require RST doc updates.

**Files:**
- Modify: `bin-api-manager/docsdev/source/auth_overview.rst` (or the appropriate auth RST file)

### Step 1: Find the right RST file

```bash
ls bin-api-manager/docsdev/source/ | grep -i auth
```

Open the relevant file and add a section for `POST /auth/delegate`.

### Step 2: Add RST content

Add under the existing auth endpoints section:

```rst
POST /auth/delegate
~~~~~~~~~~~~~~~~~~~

Issues a short-lived delegate token granting customer-admin access to a specific customer account.

**Requires:** ``PermissionProjectSuperAdmin``

**Request:**

.. code-block:: json

   {
     "customer_id": "<uuid>",
     "reason": "<10–200 printable ASCII chars>"
   }

**Response:**

.. code-block:: json

   {
     "token": "<signed JWT>",
     "customer_id": "<uuid>",
     "expire": "<RFC3339 UTC timestamp>"
   }

**Errors:**

- ``401`` — not authenticated
- ``403`` — caller lacks ``PermissionProjectSuperAdmin``, or caller is already a delegate identity
- ``404`` — target customer not found or deleted
- ``422`` — ``reason`` fails validation or ``customer_id`` is malformed

**Notes:**

- Token lifetime: 8 hours
- The issued token grants ``PermissionCustomerAdmin``-equivalent access scoped to the target customer
- All issuance events are logged with ``audit=true`` for security audit trails
```

### Step 3: Clean rebuild

```bash
cd bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

### Step 4: Force-add build output and commit

```bash
cd bin-api-manager
git add -f docsdev/build/
git add docsdev/source/
git commit -m "NOJIRA-Add-delegate-token

- bin-api-manager: Update RST docs to document POST /auth/delegate endpoint"
```

---

## Task 8: Create PR

### Step 1: Fetch latest main and check conflicts

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-delegate-token
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

If conflicts exist: rebase, resolve, re-run Task 6 verification, then create the PR.

### Step 2: Push

```bash
git push -u origin NOJIRA-Add-delegate-token
```

### Step 3: Create PR

```bash
gh pr create --title "NOJIRA-Add-delegate-token" --body "$(cat <<'EOF'
Add delegate token endpoint for platform superadmins to obtain short-lived customer-admin access.

- bin-api-manager: Add TypeDelegate, DelegateScope to models/auth/auth.go with HasPermission and IsDelegate
- bin-api-manager: Handle TypeDelegate in JWT middleware with aud enforcement and per-request JTI tracing
- bin-api-manager: Implement AuthDelegate service handler with reason validation, audit logging, Prometheus metric
- bin-api-manager: Add POST /auth/delegate HTTP handler and register under authenticated auth group
- bin-api-manager: Update RST docs to document the new endpoint
EOF
)"
```

---

## Quick Reference

| File | Change |
|------|--------|
| `bin-api-manager/models/auth/auth.go` | `TypeDelegate`, `DelegateScope`, `NewDelegateIdentity`, `IsDelegate`, `HasPermission` case, `DisplayName` case |
| `bin-api-manager/lib/middleware/authenticate.go` | `TypeDelegate` case in `buildJWTIdentity`, `delegateAudience` const, per-request JTI tracing |
| `bin-api-manager/pkg/servicehandler/main.go` | `DelegateExpiration` const, `AuthDelegate` interface method |
| `bin-api-manager/pkg/servicehandler/auth_delegate.go` | `AuthDelegate` implementation, `DelegateResponse`, `validateDelegateReason`, Prometheus metric |
| `bin-api-manager/pkg/servicehandler/mock_main.go` | Regenerated via `go generate` |
| `bin-api-manager/lib/service/auth_delegate.go` | `PostDelegate` Gin handler |
| `bin-api-manager/cmd/api-manager/main.go` | Route registration |
| `bin-api-manager/docsdev/source/` | RST docs for new endpoint |
