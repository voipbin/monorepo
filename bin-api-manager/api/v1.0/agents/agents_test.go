package agents

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_AgentsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent
		req   request.BodyAgentsPOST

		username   string
		password   string
		agentName  string
		detail     string
		ringMethod amagent.RingMethod
		permission amagent.Permission
		tagIDs     []uuid.UUID
		addresses  []commonaddress.Address

		res *amagent.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7cb6256c-8df4-11ee-bc2b-476ff1dc3eb8"),
				},
			},
			request.BodyAgentsPOST{
				Username:   "test1",
				Password:   "password1",
				Name:       "test1 name",
				Detail:     "test1 detail",
				RingMethod: "ringall",
				Permission: 0,
				TagIDs:     []uuid.UUID{},
				Addresses:  []commonaddress.Address{},
			},

			"test1",
			"password1",
			"test1 name",
			"test1 detail",
			amagent.RingMethodRingAll,
			0,
			[]uuid.UUID{},
			[]commonaddress.Address{},

			&amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
				},
			},
		},
		{
			"have webhook",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7cf444aa-8df4-11ee-abd9-b762d225dc87"),
				},
			},
			request.BodyAgentsPOST{
				Username:   "test1",
				Password:   "password1",
				Name:       "test1 name",
				Detail:     "test1 detail",
				RingMethod: "ringall",
				Permission: 0,
				TagIDs:     []uuid.UUID{},
				Addresses:  []commonaddress.Address{},
			},

			"test1",
			"password1",
			"test1 name",
			"test1 detail",
			amagent.RingMethodRingAll,
			0,
			[]uuid.UUID{},
			[]commonaddress.Address{},

			&amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3071bee2-79af-11ec-9f30-83b56e9d88b5"),
				},
			},
		}}

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

			// create body
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/agents", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AgentCreate(req.Context(), &tt.agent, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tagIDs, tt.addresses).Return(tt.res, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_AgentsGET(t *testing.T) {

	tests := []struct {
		name     string
		agent    amagent.Agent
		reqQuery string

		pageSize  uint64
		pageToken string
		filters   map[string]string

		resAgents []*amagent.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d2835bc-8df4-11ee-bde2-377a8d7b62a2"),
				},
			},
			"/v1.0/agents?page_size=11&page_token=2020-09-20T03:23:20.995000&tag_ids=b79599f2-4f2a-11ec-b49d-df70a67f68d3",

			11,
			"2020-09-20T03:23:20.995000",
			map[string]string{
				"customer_id": "00000000-0000-0000-0000-000000000000",
				"deleted":     "false",
				"tag_ids":     "b79599f2-4f2a-11ec-b49d-df70a67f68d3",
			},

			[]*amagent.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
		},
		{
			"1 tag id and status",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d5c0be4-8df4-11ee-866d-e7d040a2316f"),
				},
			},
			"/v1.0/agents?page_size=10&page_token=&tag_ids=b79599f2-4f2a-11ec-b49d-df70a67f68d3&status=available",

			10,
			"",
			map[string]string{
				"customer_id": "00000000-0000-0000-0000-000000000000",
				"deleted":     "false",
				"tag_ids":     "b79599f2-4f2a-11ec-b49d-df70a67f68d3",
				"status":      string(amagent.StatusAvailable),
			},

			[]*amagent.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
		},
		{
			"more than 2 tag ids",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d961122-8df4-11ee-8e1b-9bd95bec6c75"),
				},
			},
			"/v1.0/agents?tag_ids=b79599f2-4f2a-11ec-b49d-df70a67f68d3,39fa07ce-4fb8-11ec-8e5b-db7c7886455c&status=available",

			10,
			"",
			map[string]string{
				"customer_id": "00000000-0000-0000-0000-000000000000",
				"deleted":     "false",
				"tag_ids":     "b79599f2-4f2a-11ec-b49d-df70a67f68d3,39fa07ce-4fb8-11ec-8e5b-db7c7886455c",
				"status":      string(amagent.StatusAvailable),
			},

			[]*amagent.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("39fa07ce-4fb8-11ec-8e5b-db7c7886455c"),
					},
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

			mockSvc.EXPECT().AgentGets(req.Context(), &tt.agent, tt.pageSize, tt.pageToken, tt.filters).Return(tt.resAgents, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_AgentsIDStatusPUT(t *testing.T) {

	tests := []struct {
		name string

		agent    amagent.Agent
		reqQuery string
		reqBody  []byte

		agentID uuid.UUID
		status  amagent.Status
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d961122-8df4-11ee-8e1b-9bd95bec6c75"),
				},
			},
			"/v1.0/agents/a8ba6662-540a-11ec-9a9f-b31de1a77615/status",
			[]byte(`{"status":"available"}`),

			uuid.FromStringOrNil("a8ba6662-540a-11ec-9a9f-b31de1a77615"),
			amagent.StatusAvailable,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().AgentUpdateStatus(req.Context(), &tt.agent, tt.agentID, tt.status).Return(&amagent.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_agentsIDPermissionPUT(t *testing.T) {

	tests := []struct {
		name string

		agent    amagent.Agent
		reqQuery string
		reqBody  []byte

		agentID    uuid.UUID
		permission amagent.Permission
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d961122-8df4-11ee-8e1b-9bd95bec6c75"),
				},
			},
			"/v1.0/agents/a8ba6662-540a-11ec-9a9f-b31de1a77615/permission",
			[]byte(`{"permission":32}`),

			uuid.FromStringOrNil("a8ba6662-540a-11ec-9a9f-b31de1a77615"),
			amagent.PermissionCustomerAdmin,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().AgentUpdatePermission(req.Context(), &tt.agent, tt.agentID, tt.permission).Return(&amagent.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_agentsIDPasswordPUT(t *testing.T) {

	tests := []struct {
		name string

		agent    amagent.Agent
		reqQuery string
		reqBody  []byte

		agentID  uuid.UUID
		password string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d3481932-d3cf-11ee-ab64-5b6368efe4ce"),
				},
			},
			"/v1.0/agents/d3481932-d3cf-11ee-ab64-5b6368efe4ce/password",
			[]byte(`{"password":"updatepassword"}`),

			uuid.FromStringOrNil("d3481932-d3cf-11ee-ab64-5b6368efe4ce"),
			"updatepassword",
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().AgentUpdatePassword(req.Context(), &tt.agent, tt.agentID, tt.password).Return(&amagent.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
