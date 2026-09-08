package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func setupServer(app *gin.Engine) {
	auth := app.Group("/auth")
	auth.POST("/login", PostLogin)
	auth.POST("/password-forgot", PostPasswordForgot)
	auth.POST("/password-reset", PostPasswordReset)
	auth.GET("/password-reset", GetPasswordReset)
}

func Test_loginPOST(t *testing.T) {

	type test struct {
		name string

		reqBody RequestBodyLoginPOST

		responseToken string
		expectCookie  string
		expectRes     string
	}

	tests := []test{
		{
			name: "normal",

			reqBody: RequestBodyLoginPOST{
				Username: "test@test.com",
				Password: "testpassword",
			},

			responseToken: "test_token",
			expectCookie:  "token=test_token",
			expectRes:     `{"username":"test@test.com","token":"test_token"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AuthLogin(req.Context(), tt.reqBody.Username, tt.reqBody.Password).Return(tt.responseToken, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			cookie := w.Header().Get("Set-Cookie")
			if !strings.Contains(cookie, tt.expectCookie) {
				t.Errorf("Wrong match. expect contains: %s, got: %s", tt.expectCookie, cookie)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func TestPostLogin_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	setupServer(r)

	// Invalid JSON
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got: %d", w.Code)
	}
}

func TestPostLogin_AuthFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	setupServer(r)

	reqBody := RequestBodyLoginPOST{
		Username: "test@test.com",
		Password: "wrongpassword",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	mockSvc.EXPECT().AuthLogin(req.Context(), reqBody.Username, reqBody.Password).Return("", errors.New("auth failed"))

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for auth failure, got: %d", w.Code)
	}
	// Anti-enumeration regression guard (design 4-1-2): a credential failure
	// must stay body-less, so a caller cannot tell "wrong password" apart from
	// "no such user" -- or from an expired account, which DOES get a body.
	if w.Body.Len() != 0 {
		t.Errorf("Expected an empty body for a credential failure, got: %s", w.Body.String())
	}
}

// TestPostLogin_AccountStatusEnvelope covers design 4-1c. Without this mapping
// PostLogin would collapse AuthLogin's new status refusals into the historical
// body-less 400 and the affected users would be told nothing at all -- they
// cannot reach the v1 gate's ACCOUNT_EXPIRED guidance either, precisely
// because login is now shut.
func TestPostLogin_AccountStatusEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string

		loginErr error

		expectStatus  int
		expectReason  string
		expectDetails bool
	}{
		{
			name: "expired account gets a 403 envelope carrying the recovery endpoint",

			loginErr: fmt.Errorf("%w: customer_id: some-id", serviceerrors.ErrAccountExpired),

			expectStatus:  http.StatusForbidden,
			expectReason:  "ACCOUNT_EXPIRED",
			expectDetails: true,
		},
		{
			name: "deleted account gets a 403 envelope with no details",

			loginErr: fmt.Errorf("%w: customer_id: some-id", serviceerrors.ErrAccountDeleted),

			expectStatus: http.StatusForbidden,
			expectReason: "ACCOUNT_DELETED",
			// No recovery path exists for a deleted account, so there is
			// nothing honest to put in details.
			expectDetails: false,
		},
		{
			// The customer-lookup RPC inside AuthLogin fails closed, but that
			// is not an expiry determination and must not be dressed up as
			// one. It keeps the opaque 400.
			name: "customer lookup failure stays an opaque 400",

			loginErr: fmt.Errorf("could not get the customer info: %w", errors.New("rabbitmq down")),

			expectStatus: http.StatusBadRequest,
			expectReason: "",
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
			setupServer(r)

			reqBody := RequestBodyLoginPOST{
				Username: "test@test.com",
				Password: "testpassword",
			}
			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AuthLogin(req.Context(), reqBody.Username, reqBody.Password).Return("", tt.loginErr)

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Fatalf("Wrong status code. expect: %d, got: %d; body=%s", tt.expectStatus, w.Code, w.Body.String())
			}

			if tt.expectReason == "" {
				if w.Body.Len() != 0 {
					t.Errorf("Expected an empty body, got: %s", w.Body.String())
				}
				return
			}

			var full map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &full); err != nil {
				t.Fatalf("Could not unmarshal the body. err: %v; body=%s", err, w.Body.String())
			}
			errObj, ok := full["error"].(map[string]any)
			if !ok {
				t.Fatalf("body.error is not an object: %+v", full)
			}
			if got, _ := errObj["reason"].(string); got != tt.expectReason {
				t.Errorf("Wrong reason. expect: %s, got: %s; body=%s", tt.expectReason, got, w.Body.String())
			}
			if got, _ := errObj["status"].(string); got != "PERMISSION_DENIED" {
				t.Errorf("Wrong status. expect: PERMISSION_DENIED, got: %s", got)
			}
			// The internal Domain field must never cross the API boundary.
			if _, hasDomain := errObj["domain"]; hasDomain {
				t.Errorf("domain key MUST be absent from the external response; body=%s", w.Body.String())
			}

			details, hasDetails := errObj["details"].([]any)
			if !tt.expectDetails {
				if hasDetails {
					t.Errorf("Expected no details, got: %v", details)
				}
				return
			}
			if !hasDetails || len(details) != 1 {
				t.Fatalf("Expected exactly one details entry, got: %v; body=%s", errObj["details"], w.Body.String())
			}
			entry, ok := details[0].(map[string]any)
			if !ok {
				t.Fatalf("details[0] is not an object: %+v", details[0])
			}
			// The whole point of details is that a client does not have to
			// string-match the message to find the recovery path.
			if got, _ := entry["recovery_endpoint"].(string); got != middleware.RecoveryEndpointAccountExpired {
				t.Errorf("Wrong recovery_endpoint. expect: %s, got: %s", middleware.RecoveryEndpointAccountExpired, got)
			}
		})
	}
}

func TestPostPasswordForgot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		reqBody      RequestBodyPasswordForgotPOST
		expectStatus int
	}{
		{
			name: "valid request",
			reqBody: RequestBodyPasswordForgotPOST{
				Username: "test@test.com",
			},
			expectStatus: 200,
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
			setupServer(r)

			body, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/auth/password-forgot", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AuthPasswordForgot(req.Context(), tt.reqBody.Username).Return(nil)

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestPostPasswordForgot_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	setupServer(r)

	req, _ := http.NewRequest("POST", "/auth/password-forgot", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got: %d", w.Code)
	}
}

func TestPostPasswordReset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		reqBody      RequestBodyPasswordResetPOST
		mockSetup    func(*servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name: "valid reset",
			reqBody: RequestBodyPasswordResetPOST{
				Token:    "valid_reset_token_64_chars_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Password: "newpassword",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthPasswordReset(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectStatus: 200,
		},
		{
			name: "invalid reset - error from service",
			reqBody: RequestBodyPasswordResetPOST{
				Token:    "invalid_token",
				Password: "newpassword",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthPasswordReset(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("invalid token"))
			},
			expectStatus: 400,
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
			setupServer(r)

			tt.mockSetup(mockSvc)

			body, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/auth/password-reset", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestPostPasswordReset_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	setupServer(r)

	req, _ := http.NewRequest("POST", "/auth/password-reset", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got: %d", w.Code)
	}
}

func TestGetPasswordReset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		token        string
		expectStatus int
	}{
		{
			name:         "valid token",
			token:        "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6abcd",
			expectStatus: 200,
		},
		{
			name:         "invalid token - too short",
			token:        "short",
			expectStatus: 400,
		},
		{
			name:         "invalid token - invalid chars",
			token:        "invalid!!!token!!!aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expectStatus: 400,
		},
		{
			name:         "empty token",
			token:        "",
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			setupServer(r)

			req, _ := http.NewRequest("GET", "/auth/password-reset?token="+tt.token, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, w.Code)
			}

			// Verify HTML content is returned for valid tokens
			if tt.expectStatus == 200 {
				if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
					t.Error("Expected HTML response for valid token")
				}
				if !strings.Contains(w.Body.String(), "Reset Password") {
					t.Error("Expected 'Reset Password' in HTML response")
				}
			}
		})
	}
}

func TestValidResetTokenRegex(t *testing.T) {
	tests := []struct {
		name  string
		token string
		valid bool
	}{
		{
			name:  "valid 64 char hex",
			token: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6abcd",
			valid: true,
		},
		{
			name:  "valid 64 char all numbers",
			token: "1234567890123456789012345678901234567890123456789012345678901234",
			valid: true,
		},
		{
			name:  "invalid - too short",
			token: "abc123",
			valid: false,
		},
		{
			name:  "invalid - uppercase hex",
			token: "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6ABCD",
			valid: false,
		},
		{
			name:  "invalid - special chars",
			token: "a1b2c3d4!!!6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6abcd",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validResetToken.MatchString(tt.token)
			if result != tt.valid {
				t.Errorf("Expected %v for token %s, got %v", tt.valid, tt.token, result)
			}
		})
	}
}
