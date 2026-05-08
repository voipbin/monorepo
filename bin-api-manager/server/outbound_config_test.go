package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_outboundConfigGET(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseConfig *cmoutboundconfig.WebhookMessage

		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/outbound_config",

			responseConfig: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
				CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				Name:       "test config",
				Detail:     "test detail",
				TMCreate:   &t1,
			},

			expectRes: `{"id":"7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d","customer_id":"e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2","name":"test config","detail":"test detail","destination_whitelist":null,"codecs":"","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","tm_create":"2024-01-15T10:30:00Z","tm_update":null,"tm_delete":null}`,
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

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().OutboundConfigSelfGet(
				req.Context(),
				tt.agent,
			).Return(tt.responseConfig, nil)

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

func Test_outboundConfigPUT(t *testing.T) {
	name := "updated config"
	detail := "updated detail"

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseConfig *cmoutboundconfig.WebhookMessage

		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/outbound_config",
			reqBody:  []byte(`{"name":"updated config","detail":"updated detail"}`),

			responseConfig: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
				CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				Name:       name,
				Detail:     detail,
			},

			expectRes: `{"id":"7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d","customer_id":"e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2","name":"updated config","detail":"updated detail","destination_whitelist":null,"codecs":"","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().OutboundConfigSelfUpdate(
				req.Context(),
				tt.agent,
				gomock.Any(),
			).Return(tt.responseConfig, nil)

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
