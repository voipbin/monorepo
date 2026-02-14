package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
