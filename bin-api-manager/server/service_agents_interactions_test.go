package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cminteraction "monorepo/bin-contact-manager/models/interaction"
	cmresolution "monorepo/bin-contact-manager/models/resolution"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsInteractions(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseItems []*cminteraction.Interaction
		responseToken string
		expectStatus  int
	}{
		{
			name: "normal - filter by contact_id, plain agent permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:      "/service_agents/interactions?contact_id=11111111-0000-0000-0000-000000000001",
			responseItems: []*cminteraction.Interaction{},
			responseToken: "",
			expectStatus:  http.StatusOK,
		},
		{
			name: "normal - admin permission also passes",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			reqQuery:      "/service_agents/interactions?contact_id=11111111-0000-0000-0000-000000000001",
			responseItems: []*cminteraction.Interaction{},
			responseToken: "",
			expectStatus:  http.StatusOK,
		},
		{
			name: "normal - no filter, unfiltered mode with default 30d since",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:      "/service_agents/interactions",
			responseItems: []*cminteraction.Interaction{},
			responseToken: "",
			expectStatus:  http.StatusOK,
		},
		{
			name: "normal - no filter, explicit since",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:      "/service_agents/interactions?since=7d",
			responseItems: []*cminteraction.Interaction{},
			responseToken: "",
			expectStatus:  http.StatusOK,
		},
		{
			name: "bad request - since invalid format",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:     "/service_agents/interactions?since=7days",
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - since exceeds 180d max",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:     "/service_agents/interactions?since=181d",
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - two filters provided simultaneously",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:     "/service_agents/interactions?contact_id=11111111-0000-0000-0000-000000000001&peer_type=tel&peer_target=%2B155****1111",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/service_agents/interactions?contact_id=11111111-0000-0000-0000-000000000001",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			if tt.responseItems != nil && tt.agent != nil {
				mockSvc.EXPECT().
					ServiceAgentInteractionList(req.Context(), tt.agent, uint64(100), "", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(tt.responseItems, tt.responseToken, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_GetServiceAgentsInteractionsUnresolved(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery      string
		responseItems []*cminteraction.Interaction
		responseToken string
		expectStatus  int
	}{
		{
			name: "normal - default since",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:      "/service_agents/interactions/unresolved",
			responseItems: []*cminteraction.Interaction{},
			responseToken: "",
			expectStatus:  http.StatusOK,
		},
		{
			name: "bad request - invalid since format",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:     "/service_agents/interactions/unresolved?since=30",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/service_agents/interactions/unresolved",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			if tt.responseItems != nil && tt.agent != nil {
				mockSvc.EXPECT().
					ServiceAgentInteractionListUnresolved(req.Context(), tt.agent, uint64(100), "", gomock.Any()).
					Return(tt.responseItems, tt.responseToken, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_GetServiceAgentsInteractionsId(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	interactionID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery            string
		responseInteraction *cminteraction.Interaction
		expectStatus        int
	}{
		{
			name: "normal - plain agent permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery: "/service_agents/interactions/11111111-0000-0000-0000-000000000001",
			responseInteraction: &cminteraction.Interaction{
				ID:         interactionID,
				CustomerID: customerID,
				Direction:  "incoming",
			},
			expectStatus: http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/service_agents/interactions/11111111-0000-0000-0000-000000000001",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			if tt.responseInteraction != nil && tt.agent != nil {
				mockSvc.EXPECT().
					ServiceAgentInteractionGet(req.Context(), tt.agent, interactionID).
					Return(tt.responseInteraction, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_PostServiceAgentsInteractionsIdResolutions(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	interactionID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")
	resolvedByID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery     string
		reqBody      map[string]string
		responseRes  *cmresolution.Resolution
		expectStatus int
	}{
		{
			name: "normal - plain agent permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery: "/service_agents/interactions/11111111-0000-0000-0000-000000000001/resolutions",
			reqBody: map[string]string{
				"contact_id":       contactID.String(),
				"resolution_type":  "positive",
				"resolved_by_type": "agent",
				"resolved_by_id":   resolvedByID.String(),
			},
			responseRes: &cmresolution.Resolution{
				ID:            uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004"),
				CustomerID:    customerID,
				InteractionID: &interactionID,
				ContactID:     contactID,
			},
			expectStatus: http.StatusCreated,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/service_agents/interactions/11111111-0000-0000-0000-000000000001/resolutions",
			reqBody:      map[string]string{},
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			bodyBytes, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			if tt.responseRes != nil && tt.agent != nil {
				mockSvc.EXPECT().
					ServiceAgentResolutionCreate(req.Context(), tt.agent, interactionID, contactID, "positive", "agent", resolvedByID).
					Return(tt.responseRes, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_DeleteServiceAgentsInteractionsIdResolutionsRid(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	interactionID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	resolutionID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery     string
		expectStatus int
	}{
		{
			name: "normal - plain agent permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			reqQuery:     "/service_agents/interactions/11111111-0000-0000-0000-000000000001/resolutions/44444444-0000-0000-0000-000000000004",
			expectStatus: http.StatusOK,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/service_agents/interactions/11111111-0000-0000-0000-000000000001/resolutions/44444444-0000-0000-0000-000000000004",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			if tt.agent != nil {
				mockSvc.EXPECT().
					ServiceAgentResolutionDelete(req.Context(), tt.agent, interactionID, resolutionID).
					Return(nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}
