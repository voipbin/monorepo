package messages

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"

	"monorepo/bin-hook-manager/api/models/common"
	"monorepo/bin-hook-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.Group("/v1.0")
	ApplyRoutes(v1)
}

func Test_MessagesPOST(t *testing.T) {

	tests := []struct {
		name string

		reqQuery string
		reqBody  []byte

		expectRes string
	}{
		{
			name: "normal",

			reqQuery: "/v1.0/messages/telnyx",
			reqBody:  []byte(`{"key1":"val1"}`),

			expectRes: ``,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().Message(gomock.Any(), gomock.Any()).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_MessagesPOST_Error(t *testing.T) {

	tests := []struct {
		name string

		reqQuery  string
		reqBody   []byte
		returnErr error

		expectRes  string
		expectCode int
	}{
		{
			name: "service handler error",

			reqQuery: "/v1.0/messages/telnyx",
			reqBody:  []byte(`{"key1":"val1"}`),

			expectRes:  ``,
			expectCode: http.StatusInternalServerError,
		},
		{
			name: "a callback that is refused is a client error",

			reqQuery:  "/v1.0/messages/twilio",
			reqBody:   []byte(`MessageStatus=delivered`),
			returnErr: &servicehandler.ValidationError{Err: fmt.Errorf("refused")},

			expectRes:  ``,
			expectCode: http.StatusBadRequest,
		},
		{
			name: "a refusal that is wrapped is a client error",

			reqQuery:  "/v1.0/messages/twilio",
			reqBody:   []byte(`MessageStatus=delivered`),
			returnErr: fmt.Errorf("wrapped: %w", &servicehandler.ValidationError{Err: fmt.Errorf("refused")}),

			expectRes:  ``,
			expectCode: http.StatusBadRequest,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			returnErr := tt.returnErr
			if returnErr == nil {
				returnErr = fmt.Errorf("service handler error")
			}
			mockSvc.EXPECT().Message(gomock.Any(), gomock.Any()).Return(returnErr)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCode, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}
