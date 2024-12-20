package messages

import (
	"bytes"
	"encoding/json"
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
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}

func Test_MessagesPOST(t *testing.T) {

	tests := []struct {
		name string

		target string
		req    []byte

		expectRes string
	}{
		{
			name: "normal",

			target: "/v1.0/messages/telnyx",
			req:    []byte(`{"key1":"val1"}`),

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

			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().Message(gomock.Any(), tt.target, body).Return(nil)

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

		target string
		req    []byte

		expectRes  string
		expectCode int
	}{
		{
			name: "service handler error",

			target: "/v1.0/messages/telnyx",
			req:    []byte(`{"key1":"val1"}`),

			expectRes:  ``,
			expectCode: http.StatusInternalServerError,
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

			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().Message(gomock.Any(), tt.target, body).Return(fmt.Errorf("service handler error"))

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
