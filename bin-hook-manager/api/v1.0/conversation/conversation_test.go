package conversation

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

func Test_conversationPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	tests := []struct {
		name string

		target string
		req    []byte

		expectRes string
	}{
		{
			"normal",

			"/v1.0/conversation/customers/93a3d022-ea7a-11ec-a955-03ad3ccd0ea9/line",
			[]byte(`{"test string"}`),
			``,
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

			req, _ := http.NewRequest("POST", tt.target, bytes.NewBuffer(tt.req))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().Conversation(gomock.Any(), gomock.Any()).Return("", nil)

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

func Test_conversationPOST_ServiceHandlerError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	tests := []struct {
		name string

		target string
		req    []byte

		expectCode int
	}{
		{
			name: "service handler error",

			target: "/v1.0/conversation/customers/id/line",
			req:    []byte(`{"test":"data"}`),

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

			req, _ := http.NewRequest("POST", tt.target, bytes.NewBuffer(tt.req))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().Conversation(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("service handler error"))

			r.ServeHTTP(w, req)
			if w.Code != tt.expectCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCode, w.Code)
			}
		})
	}
}

func Test_conversationGET_Challenge(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	tests := []struct {
		name string

		target     string
		challenge  string
		expectCode int
		expectBody string
	}{
		{
			name: "GET with challenge",

			target:     "/v1.0/conversation/customers/id/whatsapp?hub.mode=subscribe&hub.challenge=abc123&hub.verify_token=token",
			challenge:  "abc123",
			expectCode: http.StatusOK,
			expectBody: "abc123",
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

			req, _ := http.NewRequest("GET", tt.target, nil)

			mockSvc.EXPECT().Conversation(gomock.Any(), gomock.Any()).Return(tt.challenge, nil)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCode, w.Code)
			}

			if w.Body.String() != tt.expectBody {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectBody, w.Body.String())
			}
		})
	}
}
