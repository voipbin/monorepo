package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_agentsGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCalls []*amagent.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

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
			expectRes:       `{"result":[{"id":"2e0e4bc4-3fa1-11ef-956a-cfb5ea5ac8ef","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"2e6cb808-3fa1-11ef-a2c1-9b3188520125","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentAgentList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseCalls, nil)

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
		agent amagent.Agent

		reqQuery string

		responseAgent *amagent.WebhookMessage

		expectAgentID uuid.UUID
		expectRes     string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/service_agents/agents/5fc7c6d6-3fa1-11ef-8f91-2b9c5b095cab",

			responseAgent: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5fc7c6d6-3fa1-11ef-8f91-2b9c5b095cab"),
				},
				TMCreate: "2020-09-20T03:23:21.995000Z",
			},

			expectAgentID: uuid.FromStringOrNil("5fc7c6d6-3fa1-11ef-8f91-2b9c5b095cab"),
			expectRes:     `{"id":"5fc7c6d6-3fa1-11ef-8f91-2b9c5b095cab","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"2020-09-20T03:23:21.995000Z","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentAgentGet(req.Context(), &tt.agent, tt.expectAgentID).Return(tt.responseAgent, nil)
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
