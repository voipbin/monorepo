package emails

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

			reqQuery: "/v1.0/emails/sendgrid",
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

			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().Email(gomock.Any(), tt.reqQuery, body).Return(nil)

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

func Test_EmailsPOST_ReadBodyError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	setupServer(r)

	// Create request with error body reader
	req, _ := http.NewRequest("POST", "/v1.0/emails/sendgrid", &errorReader{})
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}

func Test_EmailsPOST_ServiceHandlerError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	tests := []struct {
		name string

		reqQuery string
		reqBody  []byte

		expectCode int
	}{
		{
			name: "service handler error",

			reqQuery: "/v1.0/emails/sendgrid",
			reqBody:  []byte(`{"key1":"val1"}`),

			expectCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
			})
			setupServer(r)

			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().Email(gomock.Any(), gomock.Any(), body).Return(fmt.Errorf("service handler error"))

			r.ServeHTTP(w, req)
			if w.Code != tt.expectCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCode, w.Code)
			}
		})
	}
}

// errorReader is a test helper that always returns an error on Read
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (e *errorReader) Close() error {
	return nil
}
