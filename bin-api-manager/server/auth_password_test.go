package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func Test_PostAuthPasswordForgot(t *testing.T) {
	tests := []struct {
		name string

		reqBody openapi_server.PostAuthPasswordForgotJSONBody

		expectedCode int
	}{
		{
			name: "normal",
			reqBody: openapi_server.PostAuthPasswordForgotJSONBody{
				Username: "test@voipbin.net",
			},
			expectedCode: http.StatusOK,
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

			openapi_server.RegisterHandlers(r, h)

			body, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/auth/password-forgot", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AuthPasswordForgot(req.Context(), tt.reqBody.Username).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectedCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func Test_PostAuthPasswordForgot_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("POST", "/auth/password-forgot", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}

func Test_GetAuthPasswordReset(t *testing.T) {
	tests := []struct {
		name string

		token string

		expectedCode    int
		expectContains  string
	}{
		{
			name:  "valid token",
			token: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",

			expectedCode:   http.StatusOK,
			expectContains: "Reset Password",
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

			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", "/auth/password-reset?token="+tt.token, nil)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectedCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectedCode, w.Code)
			}

			if tt.expectContains != "" && !strings.Contains(w.Body.String(), tt.expectContains) {
				t.Errorf("Response body does not contain expected string. expect: %s", tt.expectContains)
			}
		})
	}
}

func Test_GetAuthPasswordReset_InvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "XSS attempt",
			token: `";alert(1);//`,
		},
		{
			name:  "too short",
			token: "abc123",
		},
		{
			name:  "uppercase hex",
			token: "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
		},
		{
			name:  "non-hex characters",
			token: "ghijklmnopqrstuv1234567890abcdef1234567890abcdef1234567890abcdef",
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

			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", "/auth/password-reset?token="+tt.token, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func Test_PostAuthPasswordReset(t *testing.T) {
	tests := []struct {
		name string

		reqBody openapi_server.PostAuthPasswordResetJSONBody

		responseErr error

		expectedCode int
	}{
		{
			name: "normal",
			reqBody: openapi_server.PostAuthPasswordResetJSONBody{
				Token:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				Password: "newpassword123",
			},
			responseErr:  nil,
			expectedCode: http.StatusOK,
		},
		{
			name: "reset failed",
			reqBody: openapi_server.PostAuthPasswordResetJSONBody{
				Token:    "invalidtoken",
				Password: "newpassword123",
			},
			responseErr:  errors.New("invalid or expired token"),
			expectedCode: http.StatusBadRequest,
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

			openapi_server.RegisterHandlers(r, h)

			body, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/auth/password-reset", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AuthPasswordReset(req.Context(), tt.reqBody.Token, tt.reqBody.Password).Return(tt.responseErr)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectedCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func Test_PostAuthPasswordReset_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("POST", "/auth/password-reset", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}
