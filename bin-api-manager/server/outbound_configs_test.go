package server

import (
	"bytes"
	"context"
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

func Test_outboundConfigsGET(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseConfigs []*cmoutboundconfig.WebhookMessage

		expectPageSize uint64
		expectRes      string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/outbound_configs",

			responseConfigs: []*cmoutboundconfig.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
					Name:       "test config",
					Detail:     "test detail",
					TMCreate:   &t1,
				},
			},

			expectPageSize: 100,
			expectRes:      `{"result":[{"id":"7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d","customer_id":"e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2","name":"test config","detail":"test detail","destination_whitelist":null,"codecs":"","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","tm_create":"2024-01-15T10:30:00Z","tm_update":null,"tm_delete":null}],"next_page_token":"2024-01-15T10:30:00.000000Z"}`,
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

			mockSvc.EXPECT().OutboundConfigList(
				req.Context(),
				tt.agent,
				tt.expectPageSize,
				gomock.Any(),
			).Return(tt.responseConfigs, nil)

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

func Test_outboundConfigsPOST(t *testing.T) {
	name := "test config"
	detail := "test detail"

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseConfig *cmoutboundconfig.WebhookMessage

		expectReq *cmoutboundconfig.UpdateRequest
		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/outbound_configs",
			reqBody:  []byte(`{"name":"test config","detail":"test detail"}`),

			responseConfig: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
				CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				Name:       "test config",
				Detail:     "test detail",
			},

			expectReq: &cmoutboundconfig.UpdateRequest{
				Name:   &name,
				Detail: &detail,
			},
			expectRes: `{"id":"7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d","customer_id":"e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2","name":"test config","detail":"test detail","destination_whitelist":null,"codecs":"","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().OutboundConfigCreate(
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

func Test_outboundConfigsIdGET(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseConfig *cmoutboundconfig.WebhookMessage

		expectID  uuid.UUID
		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/outbound_configs/7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",

			responseConfig: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
				CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				Name:       "test config",
				Detail:     "test detail",
			},

			expectID:  uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
			expectRes: `{"id":"7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d","customer_id":"e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2","name":"test config","detail":"test detail","destination_whitelist":null,"codecs":"","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().OutboundConfigGet(
				req.Context(),
				tt.agent,
				tt.expectID,
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

func Test_outboundConfigsIdPUT(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseConfig *cmoutboundconfig.WebhookMessage

		expectID  uuid.UUID
		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/outbound_configs/7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
			reqBody:  []byte(`{"name":"updated config","detail":"updated detail"}`),

			responseConfig: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
				CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				Name:       "updated config",
				Detail:     "updated detail",
			},

			expectID:  uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
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

			mockSvc.EXPECT().OutboundConfigUpdate(
				req.Context(),
				tt.agent,
				tt.expectID,
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

func Test_outboundConfigsIdDELETE(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseConfig *cmoutboundconfig.WebhookMessage

		expectID  uuid.UUID
		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/outbound_configs/7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",

			responseConfig: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
				CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				Name:       "test config",
				Detail:     "test detail",
			},

			expectID:  uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
			expectRes: `{"id":"7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d","customer_id":"e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2","name":"test config","detail":"test detail","destination_whitelist":null,"codecs":"","default_outgoing_source_number_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().OutboundConfigDelete(
				req.Context(),
				tt.agent,
				tt.expectID,
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

// Test_outboundConfigsIdPUT_DefaultOutgoingSourceNumberId verifies that the
// PUT /outbound_configs/{id} endpoint passes DefaultOutgoingSourceNumberId
// through to the internal UpdateRequest.
//
// Without convertOutboundConfigUpdateRequest mapping the field, the entire
// feature is dead from the public API even though every layer underneath
// supports it.
func Test_outboundConfigsIdPUT_DefaultOutgoingSourceNumberId(t *testing.T) {
	defaultID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseConfig *cmoutboundconfig.WebhookMessage

		expectID                uuid.UUID
		expectDefaultSourceUUID uuid.UUID
	}{
		{
			name: "non-zero default_outgoing_source_number_id is propagated",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					CustomerID: uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			reqQuery: "/outbound_configs/7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
			reqBody:  []byte(`{"default_outgoing_source_number_id":"11111111-2222-3333-4444-555555555555"}`),

			responseConfig: &cmoutboundconfig.WebhookMessage{
				ID:                            uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
				CustomerID:                    uuid.FromStringOrNil("e0ea7f86-6a34-11ec-b0d7-034e45d9dfc2"),
				DefaultOutgoingSourceNumberID: defaultID,
			},

			expectID:                uuid.FromStringOrNil("7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d"),
			expectDefaultSourceUUID: defaultID,
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

			// Capture the *UpdateRequest passed to OutboundConfigUpdate to assert
			// the DefaultOutgoingSourceNumberID is correctly mapped.
			mockSvc.EXPECT().OutboundConfigUpdate(
				req.Context(),
				tt.agent,
				tt.expectID,
				gomock.Any(),
			).DoAndReturn(func(_ context.Context, _ *auth.AuthIdentity, _ uuid.UUID, updateReq *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error) {
				if updateReq == nil {
					t.Fatalf("Expected non-nil UpdateRequest, got nil")
				}
				if updateReq.DefaultOutgoingSourceNumberID == nil {
					t.Fatalf("Expected DefaultOutgoingSourceNumberID to be non-nil, got nil")
				}
				if *updateReq.DefaultOutgoingSourceNumberID != tt.expectDefaultSourceUUID {
					t.Errorf("Wrong DefaultOutgoingSourceNumberID. expect: %s, got: %s", tt.expectDefaultSourceUUID, *updateReq.DefaultOutgoingSourceNumberID)
				}
				return tt.responseConfig, nil
			})

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

