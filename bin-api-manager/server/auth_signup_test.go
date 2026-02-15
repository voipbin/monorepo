package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func Test_PostAuthSignup(t *testing.T) {

	type test struct {
		name string

		reqQuery string
		reqBody  []byte

		responseCust *cscustomer.SignupResult

		expectRes string
	}

	tests := []test{
		{
			name: "normal",

			reqQuery: "/auth/signup",
			reqBody:  []byte(`{"email":"test@example.com"}`),

			responseCust: &cscustomer.SignupResult{},

			expectRes: `{}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerSignup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseCust, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body.String())
			}
		})
	}
}

func Test_GetAuthEmailVerify(t *testing.T) {

	type test struct {
		name string

		reqQuery string

		expectCode        int
		expectContentType string
	}

	tests := []test{
		{
			name: "normal",

			reqQuery: "/auth/email-verify?token=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",

			expectCode:        http.StatusOK,
			expectContentType: "text/html",
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCode, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.expectContentType) {
				t.Errorf("Wrong content type.\nexpect contains: %v\ngot: %v", tt.expectContentType, contentType)
			}
		})
	}
}

func Test_PostAuthEmailVerify(t *testing.T) {

	type test struct {
		name string

		reqQuery string
		reqBody  []byte

		responseCust *cscustomer.EmailVerifyResult

		expectRes string
	}

	tests := []test{
		{
			name: "normal",

			reqQuery: "/auth/email-verify",
			reqBody:  []byte(`{"token":"test-verification-token"}`),

			responseCust: &cscustomer.EmailVerifyResult{},

			expectRes: `{}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerEmailVerify(gomock.Any(), "test-verification-token").Return(tt.responseCust, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body.String())
			}
		})
	}
}
