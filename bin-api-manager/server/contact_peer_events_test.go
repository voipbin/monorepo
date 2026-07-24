package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetContactPeerEvents(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	filterContactID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		expectMockCalled  bool
		expectContactID   uuid.UUID
		expectPeerAddress *commonaddress.Address
		responseItems     []*tmpeerevent.PeerEvent
		responseToken     string
		expectStatus      int
	}{
		{
			name: "normal - filter by contact_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:          "/contact_peer_events?contact_id=11111111-0000-0000-0000-000000000001",
			expectMockCalled:  true,
			expectContactID:   filterContactID,
			expectPeerAddress: nil,
			responseItems:     []*tmpeerevent.PeerEvent{},
			responseToken:     "",
			expectStatus:      http.StatusOK,
		},
		{
			name: "normal - filter by peer_type+peer_target",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:          "/contact_peer_events?peer_type=tel&peer_target=%2B155****1111",
			expectMockCalled:  true,
			expectContactID:   uuid.Nil,
			expectPeerAddress: &commonaddress.Address{Type: commonaddress.Type("tel"), Target: "+155****1111"},
			responseItems:     []*tmpeerevent.PeerEvent{{EventType: "call_hangup"}},
			responseToken:     "",
			expectStatus:      http.StatusOK,
		},
		{
			name: "bad request - no filter",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:     "/contact_peer_events",
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - two filters provided simultaneously",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:     "/contact_peer_events?contact_id=11111111-0000-0000-0000-000000000001&peer_type=tel&peer_target=%2B155****1111",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/contact_peer_events?contact_id=11111111-0000-0000-0000-000000000001",
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

			if tt.expectMockCalled && tt.agent != nil {
				mockSvc.EXPECT().
					PeerEventList(req.Context(), tt.agent, tt.expectContactID, tt.expectPeerAddress, "", uint64(100)).
					Return(tt.responseItems, tt.responseToken, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_GetServiceAgentsContactPeerEvents(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		expectMockCalled  bool
		expectContactID   uuid.UUID
		expectPeerAddress *commonaddress.Address
		responseItems     []*tmpeerevent.PeerEvent
		responseToken     string
		expectStatus      int
	}{
		{
			name: "normal - filter by peer_type+peer_target",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:          "/service_agents/contact_peer_events?peer_type=tel&peer_target=%2B155****1111",
			expectMockCalled:  true,
			expectContactID:   uuid.Nil,
			expectPeerAddress: &commonaddress.Address{Type: commonaddress.Type("tel"), Target: "+155****1111"},
			responseItems:     []*tmpeerevent.PeerEvent{},
			responseToken:     "",
			expectStatus:      http.StatusOK,
		},
		{
			name: "bad request - no filter",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
			}),
			reqQuery:     "/service_agents/contact_peer_events",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/service_agents/contact_peer_events?peer_type=tel&peer_target=%2B155****1111",
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

			if tt.expectMockCalled && tt.agent != nil {
				mockSvc.EXPECT().
					ServiceAgentPeerEventList(req.Context(), tt.agent, tt.expectContactID, tt.expectPeerAddress, "", uint64(100)).
					Return(tt.responseItems, tt.responseToken, nil)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}
