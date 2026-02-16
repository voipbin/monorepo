package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostAuthCompleteSignup(t *testing.T) {

	type test struct {
		name string

		reqQuery string
		reqBody  []byte

		responseResult *cscustomer.CompleteSignupResult

		expectCode int
	}

	tests := []test{
		{
			name: "normal",

			reqQuery: "/auth/complete-signup",
			reqBody:  []byte(`{"temp_token":"tmp_abcdef123","code":"123456"}`),

			responseResult: &cscustomer.CompleteSignupResult{
				CustomerID: "d1d2d3d4-0000-0000-0000-000000000001",
				Accesskey: &csaccesskey.Accesskey{
					ID: uuid.FromStringOrNil("aaaa1111-bbbb-cccc-dddd-eeeeeeeeeeee"),
				},
			},

			expectCode: http.StatusOK,
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

			mockSvc.EXPECT().CustomerCompleteSignup(gomock.Any(), "tmp_abcdef123", "123456").Return(tt.responseResult, nil)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCode, w.Code)
			}
		})
	}
}

func Test_PostAuthCompleteSignup_badRequest(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("POST", "/auth/complete-signup", bytes.NewBuffer([]byte(`invalid json`)))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}

func Test_PostAuthCompleteSignup_rateLimited(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("POST", "/auth/complete-signup", bytes.NewBuffer([]byte(`{"temp_token":"tmp_abc","code":"123456"}`)))
	req.Header.Set("Content-Type", "application/json")

	mockSvc.EXPECT().CustomerCompleteSignup(gomock.Any(), "tmp_abc", "123456").Return(nil, requesthandler.ErrTooManyRequests)

	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusTooManyRequests, w.Code)
	}
}

func Test_PostAuthCompleteSignup_otherError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("POST", "/auth/complete-signup", bytes.NewBuffer([]byte(`{"temp_token":"tmp_abc","code":"999999"}`)))
	req.Header.Set("Content-Type", "application/json")

	mockSvc.EXPECT().CustomerCompleteSignup(gomock.Any(), "tmp_abc", "999999").Return(nil, fmt.Errorf("invalid verification code"))

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}
