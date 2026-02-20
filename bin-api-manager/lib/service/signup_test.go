package service

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

func TestPostCustomerSignup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		reqBody      RequestBodySignupPOST
		mockSetup    func(*servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name: "valid signup",
			reqBody: RequestBodySignupPOST{
				Name:          "Test Customer",
				Detail:        "Test details",
				Email:         "test@example.com",
				PhoneNumber:   "+1234567890",
				Address:       "123 Test St",
				WebhookMethod: "POST",
				WebhookURI:    "https://example.com/webhook",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSignup(
					gomock.Any(),
					"Test Customer",
					"Test details",
					"test@example.com",
					"+1234567890",
					"123 Test St",
					cscustomer.WebhookMethod("POST"),
					"https://example.com/webhook",
				).Return(&cscustomer.SignupResult{}, nil)
			},
			expectStatus: 200,
		},
		{
			name: "signup failed - email already exists",
			reqBody: RequestBodySignupPOST{
				Name:          "Test Customer",
				Detail:        "Test details",
				Email:         "existing@example.com",
				PhoneNumber:   "+1234567890",
				Address:       "123 Test St",
				WebhookMethod: "POST",
				WebhookURI:    "https://example.com/webhook",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSignup(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, errors.New("email already exists"))
			},
			expectStatus: 200, // Returns 200 to prevent email enumeration
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
			r.POST("/auth/signup", PostCustomerSignup)

			tt.mockSetup(mockSvc)

			body, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/auth/signup", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestPostCustomerSignup_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	r.POST("/auth/signup", PostCustomerSignup)

	req, _ := http.NewRequest("POST", "/auth/signup", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got: %d", w.Code)
	}
}

func TestPostCustomerEmailVerify(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		reqBody      RequestBodyEmailVerifyPOST
		mockSetup    func(*servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name: "valid verification",
			reqBody: RequestBodyEmailVerifyPOST{
				Token: "valid_token_64_chars_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerEmailVerify(gomock.Any(), gomock.Any()).Return(&cscustomer.EmailVerifyResult{}, nil)
			},
			expectStatus: 200,
		},
		{
			name: "invalid verification",
			reqBody: RequestBodyEmailVerifyPOST{
				Token: "invalid_token",
			},
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerEmailVerify(gomock.Any(), gomock.Any()).Return(nil, errors.New("invalid token"))
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
			r.POST("/auth/email-verify", PostCustomerEmailVerify)

			tt.mockSetup(mockSvc)

			body, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/auth/email-verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestPostCustomerEmailVerify_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	r.POST("/auth/email-verify", PostCustomerEmailVerify)

	req, _ := http.NewRequest("POST", "/auth/email-verify", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got: %d", w.Code)
	}
}

func TestGetCustomerEmailVerify(t *testing.T) {
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

			r.GET("/auth/email-verify", GetCustomerEmailVerify)

			req, _ := http.NewRequest("GET", "/auth/email-verify?token="+tt.token, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, w.Code)
			}

			// Verify HTML content is returned for valid tokens
			if tt.expectStatus == 200 {
				if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
					t.Error("Expected HTML response for valid token")
				}
				if !strings.Contains(w.Body.String(), "Verify Your Email") {
					t.Error("Expected 'Verify Your Email' in HTML response")
				}
			}
		})
	}
}

func TestValidVerifyTokenRegex(t *testing.T) {
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
			result := validVerifyToken.MatchString(tt.token)
			if result != tt.valid {
				t.Errorf("Expected %v for token %s, got %v", tt.valid, tt.token, result)
			}
		})
	}
}

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

func TestPostCustomerCompleteSignup_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		body         string
		expectStatus int
	}{
		{
			name:         "missing temp_token",
			body:         `{"code": "123456"}`,
			expectStatus: 400,
		},
		{
			name:         "missing code",
			body:         `{"temp_token": "tmp_abc123"}`,
			expectStatus: 400,
		},
		{
			name:         "empty body",
			body:         `{}`,
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
			r.POST("/auth/complete-signup", PostCustomerCompleteSignup)

			req, _ := http.NewRequest("POST", "/auth/complete-signup", bytes.NewBuffer([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, w.Code)
			}
		})
	}
}
