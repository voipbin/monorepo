package agentresources

import (
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
	amresource "monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_AgentResourcesGET(t *testing.T) {

	tests := []struct {
		name     string
		agent    amagent.Agent
		reqQuery string

		pageSize  uint64
		pageToken string
		filters   map[string]string

		resAgents []*amresource.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("7d2835bc-8df4-11ee-bde2-377a8d7b62a2"),
			},
			"/v1.0/agent_resources?page_size=11&page_token=2020-09-20T03:23:20.995000&reference_type=call",

			11,
			"2020-09-20T03:23:20.995000",
			map[string]string{
				"customer_id":    "00000000-0000-0000-0000-000000000000",
				"owner_id":       "7d2835bc-8df4-11ee-bde2-377a8d7b62a2",
				"deleted":        "false",
				"reference_type": "call",
			},

			[]*amresource.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().AgentResourceGets(req.Context(), &tt.agent, tt.pageSize, tt.pageToken, tt.filters).Return(tt.resAgents, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_AgentResourcesIDGET(t *testing.T) {

	type test struct {
		name     string
		agent    amagent.Agent
		reqQuery string

		resourceID uuid.UUID

		responseAgentResource *amresource.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			"/v1.0/agent_resources/e3fc22e8-2670-11ef-b88b-b77ac608a816",

			uuid.FromStringOrNil("e3fc22e8-2670-11ef-b88b-b77ac608a816"),
			&amresource.WebhookMessage{
				ID:       uuid.FromStringOrNil("e3fc22e8-2670-11ef-b88b-b77ac608a816"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().AgentResourceGet(req.Context(), &tt.agent, tt.resourceID).Return(tt.responseAgentResource, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseAgentResource)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_AgentResourcesIDDELETE(t *testing.T) {

	type test struct {
		name     string
		agent    amagent.Agent
		reqQuery string

		resourceID uuid.UUID

		responseAgentResource *amresource.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			"/v1.0/agent_resources/4205dab4-2671-11ef-9289-bb4e2b7d726b",

			uuid.FromStringOrNil("4205dab4-2671-11ef-9289-bb4e2b7d726b"),
			&amresource.WebhookMessage{
				ID:       uuid.FromStringOrNil("4205dab4-2671-11ef-9289-bb4e2b7d726b"),
				TMCreate: "2020-09-20T03:23:21.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().AgentResourceDelete(req.Context(), &tt.agent, tt.resourceID).Return(tt.responseAgentResource, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseAgentResource)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}
