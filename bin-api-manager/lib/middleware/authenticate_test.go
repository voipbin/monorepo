package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	modelscommon "monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
)

func Test_getTokenString(t *testing.T) {

	tests := []struct {
		name         string
		setupRequest func(c *gin.Context)
		expectRes    string
	}{
		{
			name: "Token from Cookie",
			setupRequest: func(c *gin.Context) {
				c.Request.AddCookie(&http.Cookie{Name: "token", Value: "cookieToken"})
			},
			expectRes: "cookieToken",
		},
		{
			name: "Token from Query Parameter",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "token=queryToken"
			},
			expectRes: "queryToken",
		},
		{
			name: "Token from Authorization Header",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer headerToken")
			},
			expectRes: "headerToken",
		},
		{
			name:         "No Token Provided",
			setupRequest: func(c *gin.Context) {},
			expectRes:    "",
		},
		{
			name: "Invalid Authorization Header",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "InvalidHeader headerToken")
			},
			expectRes: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Apply the test setup
			tt.setupRequest(c)

			res := getTokenString(c)
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getAccesskey(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func(c *gin.Context)
		expectRes    string
	}{
		{
			name: "Accesskey from Cookie",
			setupRequest: func(c *gin.Context) {
				c.Request.AddCookie(&http.Cookie{Name: "accesskey", Value: "cookieAccesskey"})
			},
			expectRes: "cookieAccesskey",
		},
		{
			name: "Accesskey from Query Parameter",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=queryAccesskey"
			},
			expectRes: "queryAccesskey",
		},
		{
			name:         "No Accesskey Provided",
			setupRequest: func(c *gin.Context) {},
			expectRes:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Apply the test setup
			tt.setupRequest(c)

			res := getAccesskey(c)
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getAuthString(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func(c *gin.Context)
		expectType   string
		expectString string
		expectErr    bool
	}{
		{
			name: "Token auth",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer testToken")
			},
			expectType:   authTypeToken,
			expectString: "testToken",
			expectErr:    false,
		},
		{
			name: "Accesskey auth",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=testAccesskey"
			},
			expectType:   authTypeAccesskey,
			expectString: "testAccesskey",
			expectErr:    false,
		},
		{
			name:         "No auth provided",
			setupRequest: func(c *gin.Context) {},
			expectType:   authTypeNone,
			expectString: "",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			tt.setupRequest(c)

			authType, authString, err := getAuthString(c)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if authType != tt.expectType {
				t.Errorf("Wrong auth type. expect: %v, got: %v", tt.expectType, authType)
			}
			if authString != tt.expectString {
				t.Errorf("Wrong auth string. expect: %v, got: %v", tt.expectString, authString)
			}
		})
	}
}

func Test_buildJWTIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		authData  map[string]interface{}
		expectErr bool
		expectTyp auth.Type
	}{
		{
			name: "Agent JWT (default type)",
			authData: map[string]interface{}{
				"agent": amagent.Agent{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					Permission: amagent.PermissionCustomerAdmin,
				},
			},
			expectErr: false,
			expectTyp: auth.TypeAgent,
		},
		{
			name: "Agent JWT (explicit type)",
			authData: map[string]interface{}{
				"type": "agent",
				"agent": amagent.Agent{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					Permission: amagent.PermissionCustomerAdmin,
				},
			},
			expectErr: false,
			expectTyp: auth.TypeAgent,
		},
		{
			name: "Direct JWT",
			authData: map[string]interface{}{
				"type": "direct",
				"direct": map[string]interface{}{
					"customer_id":            "5f621078-8e5f-11ee-97b2-cfe7337b701c",
					"resource_type":          "aicall",
					"resource_id":            "a1b2c3d4-0000-0000-0000-000000000000",
					"allowed_resource_types": []string{"aicall"},
				},
			},
			expectErr: false,
			expectTyp: auth.TypeDirect,
		},
		{
			name: "Invalid agent data",
			authData: map[string]interface{}{
				"agent": make(chan int),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.WithField("func", "test")
			identity, err := buildJWTIdentity(log, tt.authData)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}
			if identity.Type != tt.expectTyp {
				t.Errorf("Wrong type. expect: %v, got: %v", tt.expectTyp, identity.Type)
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testAgent := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	tests := []struct {
		name          string
		setupRequest  func(c *gin.Context)
		mockSetup     func(mockSH *servicehandler.MockServiceHandler)
		expectStatus  int
		expectAborted bool
	}{
		{
			name: "Valid authentication",
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer validToken")
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().AuthJWTParse(gomock.Any(), "validToken").Return(map[string]interface{}{
					"agent": testAgent,
				}, nil)
				mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), testAgent.CustomerID).Return(&cscustomer.Customer{
					Status: cscustomer.StatusActive,
				}, nil)
			},
			expectStatus:  200,
			expectAborted: false,
		},
		{
			name: "Invalid authentication - no token",
			setupRequest: func(c *gin.Context) {
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
			},
			expectStatus:  401,
			expectAborted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSH := servicehandler.NewMockServiceHandler(mc)
			tt.mockSetup(mockSH)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			c.Request = req

			router.Use(func(c *gin.Context) {
				c.Set(modelscommon.OBJServiceHandler, mockSH)
			})
			router.Use(Authenticate())
			router.GET("/", func(c *gin.Context) {
				c.Status(200)
			})

			tt.setupRequest(c)

			router.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, w.Code)
			}
		})
	}
}

// assertAuthErrorEnvelope decodes the response body and asserts the
// standard error envelope fields used by the Authenticate middleware.
// The external envelope intentionally does NOT include a "domain" field —
// see bin-api-manager/lib/apierror.
func assertAuthErrorEnvelope(t *testing.T, body []byte, wantStatus, wantReason string) {
	t.Helper()
	var decoded struct {
		Error struct {
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v; body: %s", err, string(body))
	}
	if decoded.Error.Status != wantStatus {
		t.Errorf("wrong status: got %q, want %q", decoded.Error.Status, wantStatus)
	}
	if decoded.Error.Reason != wantReason {
		t.Errorf("wrong reason: got %q, want %q", decoded.Error.Reason, wantReason)
	}
	if decoded.Error.Message == "" {
		t.Error("message missing")
	}
	if decoded.Error.RequestID == "" {
		t.Error("request_id missing")
	}
	// Structural check: parse the body and verify the "domain" key is
	// absent from the error object (not a substring scan, which would
	// false-positive on a Details payload containing a field named
	// "domain").
	var full map[string]any
	if err := json.Unmarshal(body, &full); err != nil {
		t.Fatalf("unmarshal full body for domain check: %v; body=%s", err, string(body))
	}
	errObj, ok := full["error"].(map[string]any)
	if !ok {
		t.Fatalf("body.error is not an object: %+v", full)
	}
	if _, hasDomain := errObj["domain"]; hasDomain {
		t.Errorf("domain key MUST be absent from external response; body=%s", string(body))
	}
}

func TestAuthenticate_MissingHeaderEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RequestID())
	r.Use(Authenticate())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want 401", w.Code)
	}
	assertAuthErrorEnvelope(t, w.Body.Bytes(), "UNAUTHENTICATED", "AUTHENTICATION_REQUIRED")
}

func TestAuthenticate_InvalidCredentialsEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), "badToken").Return(nil, fmt.Errorf("invalid signature"))

	r := gin.New()
	r.Use(RequestID())
	r.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
	})
	r.Use(Authenticate())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer badToken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want 401", w.Code)
	}
	assertAuthErrorEnvelope(t, w.Body.Bytes(), "UNAUTHENTICATED", "INVALID_CREDENTIALS")
}

func TestAuthenticate_FrozenAccountEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	deletionTime := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	testAgent := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: testCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), "validToken").Return(map[string]interface{}{
		"agent": testAgent,
	}, nil)
	mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), testCustomerID).Return(&cscustomer.Customer{
		Status:              cscustomer.StatusFrozen,
		TMDeletionScheduled: &deletionTime,
	}, nil)

	r := gin.New()
	r.Use(RequestID())
	r.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
	})
	r.Use(Authenticate())
	r.GET("/v1.0/agents", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/v1.0/agents", nil)
	req.Header.Set("Authorization", "Bearer validToken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d want 403", w.Code)
	}
	assertAuthErrorEnvelope(t, w.Body.Bytes(), "PERMISSION_DENIED", "ACCOUNT_FROZEN")

	// The frozen-account response must also carry the deletion schedule and
	// recovery endpoint in the envelope's details array — self-service
	// recovery clients depend on these exact JSON keys.
	var body struct {
		Error struct {
			Details []map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal details: %v; body=%s", err, w.Body.String())
	}
	if len(body.Error.Details) != 1 {
		t.Fatalf("expected 1 details entry, got %d; body=%s", len(body.Error.Details), w.Body.String())
	}
	entry := body.Error.Details[0]
	for _, key := range []string{"deletion_scheduled_at", "deletion_effective_at", "recovery_endpoint"} {
		if _, ok := entry[key]; !ok {
			gotKeys := make([]string, 0, len(entry))
			for k := range entry {
				gotKeys = append(gotKeys, k)
			}
			t.Errorf("details missing %q; got keys=%v", key, gotKeys)
		}
	}
	if got, want := entry["recovery_endpoint"], "DELETE /auth/unregister"; got != want {
		t.Errorf("recovery_endpoint = %v, want %q", got, want)
	}
}

func TestAuthenticateWithInvalidAgentData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)

	// Return invalid agent data that can't be marshaled
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), "invalidToken").Return(map[string]interface{}{
		"agent": make(chan int), // channels can't be marshaled
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalidToken")
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	c.Request = req

	router.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
	})
	router.Use(Authenticate())
	router.GET("/", func(c *gin.Context) {
		c.Status(200)
	})

	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401 for invalid agent data, got: %v", w.Code)
	}
}

func TestAuthenticateWithMalformedAgentJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)

	// Return data where agent field exists but is not valid Agent struct
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), "malformedToken").Return(map[string]interface{}{
		"agent": map[string]interface{}{
			"invalid_field": "this won't unmarshal to Agent",
		},
	}, nil)
	// Agent will have zero-value fields (including zero Permission), so frozen check runs
	mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), uuid.Nil).Return(&cscustomer.Customer{
		Status: cscustomer.StatusActive,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer malformedToken")
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	c.Request = req

	router.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
	})
	router.Use(Authenticate())
	router.GET("/", func(c *gin.Context) {
		c.Status(200)
	})

	router.ServeHTTP(w, req)

	// Should succeed but agent might be partially populated
	// The actual behavior depends on how json.Unmarshal handles extra fields
	// For Agent struct, it should handle missing fields gracefully
	if w.Code == 500 {
		t.Errorf("Unexpected 500 error")
	}
}

func Test_buildJWTIdentity_Delegate(t *testing.T) {
	customerID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	issuedBy   := "11111111-1111-1111-1111-111111111111"
	jti        := "some-jti-value"

	tests := []struct {
		name           string
		authData       map[string]interface{}
		expectErr      bool
		expectType     auth.Type
		expectCustomer string
		expectJTI      string
	}{
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
			name: "delegate token missing aud rejected",
			authData: map[string]interface{}{
				"type":        "delegate",
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
			name: "delegate token invalid customer_id rejected",
			authData: map[string]interface{}{
				"type":        "delegate",
				"aud":         "voipbin-api",
				"customer_id": "not-a-uuid",
				"sub":         issuedBy,
				"jti":         jti,
			},
			expectErr: true,
		},
		{
			name: "unknown token type returns error",
			authData: map[string]interface{}{
				"type": "unknown-type-xyz",
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
			if tt.expectJTI != "" {
				if res.DelegateScope == nil {
					t.Fatal("DelegateScope is nil")
				}
				if res.DelegateScope.JTI != tt.expectJTI {
					t.Errorf("Wrong JTI. expect: %v, got: %v", tt.expectJTI, res.DelegateScope.JTI)
				}
			}
		})
	}
}

func TestAuthConstants(t *testing.T) {
	if authTypeNone != "" {
		t.Errorf("authTypeNone should be empty string, got: %v", authTypeNone)
	}
	if authTypeToken != "token" {
		t.Errorf("authTypeToken should be 'token', got: %v", authTypeToken)
	}
	if authTypeAccesskey != "accesskey" {
		t.Errorf("authTypeAccesskey should be 'accesskey', got: %v", authTypeAccesskey)
	}
}

func TestAuthenticateAgentStoredInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	testAgent := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
		Username:   "testuser",
	}

	mockSH := servicehandler.NewMockServiceHandler(mc)
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), "validToken").Return(map[string]interface{}{
		"agent": testAgent,
	}, nil)
	mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), testAgent.CustomerID).Return(&cscustomer.Customer{
		Status: cscustomer.StatusActive,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer validToken")
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	c.Request = req

	var capturedIdentity *auth.AuthIdentity
	router.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
	})
	router.Use(Authenticate())
	router.GET("/", func(c *gin.Context) {
		// Capture the auth identity from context
		if tmp, exists := c.Get("auth_identity"); exists {
			capturedIdentity = tmp.(*auth.AuthIdentity)
		}
		c.Status(200)
	})

	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got: %v", w.Code)
	}

	// Verify agent was stored correctly in context
	if capturedIdentity == nil {
		t.Fatal("AuthIdentity should not be nil")
	}
	if capturedIdentity.Agent == nil {
		t.Errorf("Agent should not be nil in AuthIdentity")
	} else {
		if capturedIdentity.Agent.ID != testAgent.ID {
			t.Errorf("Agent ID mismatch. expect: %v, got: %v", testAgent.ID, capturedIdentity.Agent.ID)
		}
		if capturedIdentity.Agent.Username != testAgent.Username {
			t.Errorf("Agent username mismatch. expect: %v, got: %v", testAgent.Username, capturedIdentity.Agent.Username)
		}
	}
}

func TestAuthenticateAccesskey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	testAccesskeyID := uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001")

	tests := []struct {
		name             string
		setupRequest     func(c *gin.Context)
		mockSetup        func(mockSH *servicehandler.MockServiceHandler)
		expectStatus     int
		expectIdentityFn func(t *testing.T, identity *auth.AuthIdentity)
	}{
		{
			name: "Valid accesskey",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=valid-token"
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().AccesskeyRawGetByToken(gomock.Any(), "valid-token").Return(&csaccesskey.Accesskey{
					ID:         testAccesskeyID,
					CustomerID: testCustomerID,
					Name:       "test-key",
				}, nil)
				mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), testCustomerID).Return(&cscustomer.Customer{
					Status: cscustomer.StatusActive,
				}, nil)
			},
			expectStatus: 200,
			expectIdentityFn: func(t *testing.T, identity *auth.AuthIdentity) {
				if identity == nil {
					t.Fatal("Expected non-nil identity")
				}
				if identity.Type != auth.TypeAccesskey {
					t.Errorf("Wrong type. expect: %v, got: %v", auth.TypeAccesskey, identity.Type)
				}
				if identity.CustomerID != testCustomerID {
					t.Errorf("Wrong customer_id. expect: %v, got: %v", testCustomerID, identity.CustomerID)
				}
			},
		},
		{
			name: "Expired accesskey",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=expired-token"
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				expireTime := time.Now().UTC().Add(-1 * time.Hour)
				mockSH.EXPECT().AccesskeyRawGetByToken(gomock.Any(), "expired-token").Return(&csaccesskey.Accesskey{
					ID:         testAccesskeyID,
					CustomerID: testCustomerID,
					TMExpire:   &expireTime,
				}, nil)
			},
			expectStatus: 401,
		},
		{
			name: "Deleted accesskey",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=deleted-token"
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				deleteTime := time.Now().UTC().Add(-1 * time.Hour)
				mockSH.EXPECT().AccesskeyRawGetByToken(gomock.Any(), "deleted-token").Return(&csaccesskey.Accesskey{
					ID:         testAccesskeyID,
					CustomerID: testCustomerID,
					TMDelete:   &deleteTime,
				}, nil)
			},
			expectStatus: 401,
		},
		{
			name: "Invalid accesskey token",
			setupRequest: func(c *gin.Context) {
				c.Request.URL.RawQuery = "accesskey=bad-token"
			},
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().AccesskeyRawGetByToken(gomock.Any(), "bad-token").Return(nil, fmt.Errorf("not found"))
			},
			expectStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSH := servicehandler.NewMockServiceHandler(mc)
			tt.mockSetup(mockSH)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			c.Request = req

			var capturedIdentity *auth.AuthIdentity
			router.Use(func(c *gin.Context) {
				c.Set(modelscommon.OBJServiceHandler, mockSH)
			})
			router.Use(Authenticate())
			router.GET("/", func(c *gin.Context) {
				if tmp, exists := c.Get("auth_identity"); exists {
					capturedIdentity = tmp.(*auth.AuthIdentity)
				}
				c.Status(200)
			})

			tt.setupRequest(c)
			router.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, w.Code)
			}
			if tt.expectIdentityFn != nil {
				tt.expectIdentityFn(t, capturedIdentity)
			}
		})
	}
}

func Test_isFrozenAccountBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	deletionTime := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		agent        *auth.AuthIdentity
		method       string
		path         string
		mockSetup    func(mockSH *servicehandler.MockServiceHandler)
		expectBlock  bool
		expectStatus int
	}{
		{
			name: "Active account - not blocked",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{CustomerID: testCustomerID},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			method: http.MethodGet,
			path:   "/v1.0/agents",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), testCustomerID).Return(&cscustomer.Customer{
					Status: cscustomer.StatusActive,
				}, nil)
			},
			expectBlock:  false,
			expectStatus: 200,
		},
		{
			name: "Frozen account - blocked",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{CustomerID: testCustomerID},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			method: http.MethodGet,
			path:   "/v1.0/agents",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), testCustomerID).Return(&cscustomer.Customer{
					Status:              cscustomer.StatusFrozen,
					TMDeletionScheduled: &deletionTime,
				}, nil)
			},
			expectBlock:  true,
			expectStatus: 403,
		},
		{
			name: "Frozen account - DELETE /auth/unregister allowed",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{CustomerID: testCustomerID},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			method:       http.MethodDelete,
			path:         "/auth/unregister",
			mockSetup:    func(mockSH *servicehandler.MockServiceHandler) {},
			expectBlock:  false,
			expectStatus: 200,
		},
		{
			name: "Frozen account - POST /auth/unregister allowed",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{CustomerID: testCustomerID},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			method:       http.MethodPost,
			path:         "/auth/unregister",
			mockSetup:    func(mockSH *servicehandler.MockServiceHandler) {},
			expectBlock:  false,
			expectStatus: 200,
		},
		{
			name: "Direct token - skip frozen check",
			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           testCustomerID,
				ResourceType:         "aicall",
				ResourceID:           uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000000"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			method:       http.MethodGet,
			path:         "/v1.0/aicalls",
			mockSetup:    func(mockSH *servicehandler.MockServiceHandler) {},
			expectBlock:  false,
			expectStatus: 200,
		},
		{
			name: "Project super admin - not blocked even if frozen",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{CustomerID: testCustomerID},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			method:       http.MethodGet,
			path:         "/v1.0/agents",
			mockSetup:    func(mockSH *servicehandler.MockServiceHandler) {},
			expectBlock:  false,
			expectStatus: 200,
		},
		{
			name: "CustomerGet error - fail open (not blocked)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{CustomerID: testCustomerID},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			method: http.MethodGet,
			path:   "/v1.0/agents",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), testCustomerID).Return(nil, fmt.Errorf("service unavailable"))
			},
			expectBlock:  false,
			expectStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSH := servicehandler.NewMockServiceHandler(mc)
			tt.mockSetup(mockSH)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set(modelscommon.OBJServiceHandler, mockSH)

			blocked := isFrozenAccountBlocked(c, tt.agent)
			if blocked != tt.expectBlock {
				t.Errorf("Wrong blocked result. expect: %v, got: %v", tt.expectBlock, blocked)
			}
			if blocked && w.Code != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, w.Code)
			}
			if blocked {
				// Structural check: verify the "domain" key is absent
				// from the error object — see assertAuthErrorEnvelope.
				var full map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &full); err != nil {
					t.Fatalf("unmarshal full body for domain check: %v; body=%s", err, w.Body.String())
				}
				errObj, ok := full["error"].(map[string]any)
				if !ok {
					t.Fatalf("body.error is not an object: %+v", full)
				}
				if _, hasDomain := errObj["domain"]; hasDomain {
					t.Errorf("domain key MUST be absent from external response; body=%s", w.Body.String())
				}
			}
		})
	}
}

func TestAuthenticateFrozenAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	deletionTime := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	testAgent := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: testCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	tests := []struct {
		name         string
		method       string
		path         string
		mockSetup    func(mockSH *servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name:   "Frozen account returns 403",
			method: http.MethodGet,
			path:   "/v1.0/agents",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().AuthJWTParse(gomock.Any(), "validToken").Return(map[string]interface{}{
					"agent": testAgent,
				}, nil)
				mockSH.EXPECT().CustomerGet(gomock.Any(), gomock.Any(), testCustomerID).Return(&cscustomer.Customer{
					Status:              cscustomer.StatusFrozen,
					TMDeletionScheduled: &deletionTime,
				}, nil)
			},
			expectStatus: 403,
		},
		{
			name:   "Frozen account allows DELETE /auth/unregister",
			method: http.MethodDelete,
			path:   "/auth/unregister",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().AuthJWTParse(gomock.Any(), "validToken").Return(map[string]interface{}{
					"agent": testAgent,
				}, nil)
			},
			expectStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSH := servicehandler.NewMockServiceHandler(mc)
			tt.mockSetup(mockSH)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer validToken")
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			c.Request = req

			router.Use(func(c *gin.Context) {
				c.Set(modelscommon.OBJServiceHandler, mockSH)
			})
			router.Use(Authenticate())

			// Register routes to match the test paths
			handler := func(c *gin.Context) {
				c.Status(200)
			}
			router.GET("/v1.0/agents", handler)
			router.DELETE("/auth/unregister", handler)

			router.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %v, got: %v", tt.expectStatus, w.Code)
			}
		})
	}
}
