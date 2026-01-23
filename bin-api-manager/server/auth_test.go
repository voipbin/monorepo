package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func Test_PostAuthLogin(t *testing.T) {
	tests := []struct {
		name string

		reqBody openapi_server.PostAuthLoginJSONBody

		responseToken string
		responseErr   error

		expectedCode int
		expectedRes  string
	}{
		{
			name: "normal",
			reqBody: openapi_server.PostAuthLoginJSONBody{
				Username: "testuser@example.com",
				Password: "password123",
			},
			responseToken: "test-jwt-token",
			responseErr:   nil,

			expectedCode: http.StatusOK,
			expectedRes:  `{"token":"test-jwt-token","username":"testuser@example.com"}`,
		},
		{
			name: "invalid credentials",
			reqBody: openapi_server.PostAuthLoginJSONBody{
				Username: "baduser@example.com",
				Password: "wrongpassword",
			},
			responseToken: "",
			responseErr:   errors.New("invalid credentials"),

			expectedCode: http.StatusBadRequest,
			expectedRes:  "",
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
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AuthLogin(req.Context(), tt.reqBody.Username, tt.reqBody.Password).Return(tt.responseToken, tt.responseErr)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectedCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectedCode, w.Code)
			}

			if tt.expectedCode == http.StatusOK && w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong response body.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body.String())
			}
		})
	}
}

func Test_PostAuthLogin_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}
