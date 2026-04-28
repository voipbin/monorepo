package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_agentsGET(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseCalls []*amagent.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/agents?page_token=2020-09-20T03:23:20.995000Z&page_size=10",

			responseCalls: []*amagent.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e0e4bc4-3fa1-11ef-956a-cfb5ea5ac8ef"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e6cb808-3fa1-11ef-a2c1-9b3188520125"),
					},
				},
			},

			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"2e0e4bc4-3fa1-11ef-956a-cfb5ea5ac8ef","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"direct_hash":""},{"id":"2e6cb808-3fa1-11ef-a2c1-9b3188520125","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"direct_hash":""}],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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
			mockSvc.EXPECT().ServiceAgentAgentList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseCalls, nil)

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

func Test_agentsIDGET(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAgent *amagent.WebhookMessage

		expectAgentID uuid.UUID
		expectRes     string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/agents/5fc7c6d6-3fa1-11ef-8f91-2b9c5b095cab",

			responseAgent: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5fc7c6d6-3fa1-11ef-8f91-2b9c5b095cab"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectAgentID: uuid.FromStringOrNil("5fc7c6d6-3fa1-11ef-8f91-2b9c5b095cab"),
			expectRes:     `{"id":"5fc7c6d6-3fa1-11ef-8f91-2b9c5b095cab","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"direct_hash":"","tm_create":"2020-09-20T03:23:21.995Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			mockSvc.EXPECT().ServiceAgentAgentGet(req.Context(), tt.agent, tt.expectAgentID).Return(tt.responseAgent, nil)
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

// Test_serviceAgentsAgentsIDGet_InvalidID verifies
// GetServiceAgentsAgentsId rejects a malformed UUID in the path with
// INVALID_ARGUMENT / INVALID_ID before the servicehandler is consulted.
func Test_serviceAgentsAgentsIDGet_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodGet, "/service_agents/agents/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}
