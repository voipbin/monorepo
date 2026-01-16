package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostAgents(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqBody []byte

		responseAgent *amagent.WebhookMessage

		expectedUsername   string
		expectedPassword   string
		expectedName       string
		expectedDetail     string
		expectedRingMethod amagent.RingMethod
		expectedPermission amagent.Permission
		expectedTagIDs     []uuid.UUID
		expectedAddresses  []commonaddress.Address
		expectRes          string
	}{
		{
			name: "full data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7cb6256c-8df4-11ee-bc2b-476ff1dc3eb8"),
				},
			},

			reqBody: []byte(`{"username":"test1","password":"password1","name":"test1 name","detail":"test1 detail","ring_method":"ringall","permission":255,"tag_ids":["682374ac-d7c6-11ef-8f43-8f2c18384cd4","70100856-d7c6-11ef-9eeb-d389722f0caf"],"addresses":[{"type":"tel","target":"+123456789"}]}`),

			responseAgent: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
				},
			},

			expectedUsername:   "test1",
			expectedPassword:   "password1",
			expectedName:       "test1 name",
			expectedDetail:     "test1 detail",
			expectedRingMethod: amagent.RingMethodRingAll,
			expectedPermission: 255,
			expectedTagIDs: []uuid.UUID{
				uuid.FromStringOrNil("682374ac-d7c6-11ef-8f43-8f2c18384cd4"),
				uuid.FromStringOrNil("70100856-d7c6-11ef-9eeb-d389722f0caf"),
			},
			expectedAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+123456789",
				},
			},
			expectRes: `{"id":"bd8cee04-4f21-11ec-9955-db7041b6d997","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
		{
			name: "empty",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7cf444aa-8df4-11ee-abd9-b762d225dc87"),
				},
			},

			reqBody: []byte(`{}`),

			responseAgent: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3071bee2-79af-11ec-9f30-83b56e9d88b5"),
				},
			},

			expectedUsername:   "",
			expectedPassword:   "",
			expectedName:       "",
			expectedDetail:     "",
			expectedRingMethod: "",
			expectedPermission: 0,
			expectedTagIDs:     []uuid.UUID{},
			expectedAddresses:  []commonaddress.Address{},
			expectRes:          `{"id":"3071bee2-79af-11ec-9f30-83b56e9d88b5","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
		}}

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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", "/agents", bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AgentCreate(req.Context(), &tt.agent, tt.expectedUsername, tt.expectedPassword, tt.expectedName, tt.expectedDetail, tt.expectedRingMethod, tt.expectedPermission, tt.expectedTagIDs, tt.expectedAddresses).Return(tt.responseAgent, nil)

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

func Test_GetAgents(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAgents []*amagent.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedFilters   map[string]string
		expectRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d2835bc-8df4-11ee-bde2-377a8d7b62a2"),
				},
			},

			reqQuery: "/agents?page_size=11&page_token=2020-09-20T03:23:20.995000&tag_ids=b79599f2-4f2a-11ec-b49d-df70a67f68d3",

			responseAgents: []*amagent.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectedPageSize:  11,
			expectedPageToken: "2020-09-20T03:23:20.995000",
			expectedFilters: map[string]string{
				"customer_id": "00000000-0000-0000-0000-000000000000",
				"deleted":     "false",
				"tag_ids":     "b79599f2-4f2a-11ec-b49d-df70a67f68d3",
			},
			expectRes: `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "1 tag id and status",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d5c0be4-8df4-11ee-866d-e7d040a2316f"),
				},
			},

			reqQuery: "/agents?page_size=10&page_token=&tag_ids=b79599f2-4f2a-11ec-b49d-df70a67f68d3&status=available",

			responseAgents: []*amagent.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "",
			expectedFilters: map[string]string{
				"customer_id": "00000000-0000-0000-0000-000000000000",
				"deleted":     "false",
				"tag_ids":     "b79599f2-4f2a-11ec-b49d-df70a67f68d3",
				"status":      string(amagent.StatusAvailable),
			},
			expectRes: `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 tag ids",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d961122-8df4-11ee-8e1b-9bd95bec6c75"),
				},
			},

			reqQuery: "/agents?page_size=10&tag_ids=b79599f2-4f2a-11ec-b49d-df70a67f68d3,39fa07ce-4fb8-11ec-8e5b-db7c7886455c&status=available",

			responseAgents: []*amagent.WebhookMessage{
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

			expectedPageSize:  10,
			expectedPageToken: "",
			expectedFilters: map[string]string{
				"customer_id": "00000000-0000-0000-0000-000000000000",
				"deleted":     "false",
				"tag_ids":     "b79599f2-4f2a-11ec-b49d-df70a67f68d3,39fa07ce-4fb8-11ec-8e5b-db7c7886455c",
				"status":      string(amagent.StatusAvailable),
			},
			expectRes: `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"39fa07ce-4fb8-11ec-8e5b-db7c7886455c","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().AgentList(req.Context(), &tt.agent, tt.expectedPageSize, tt.expectedPageToken, tt.expectedFilters).Return(tt.responseAgents, nil)

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

func Test_PutAgentsIdStatus(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAgent *amagent.WebhookMessage

		expectedAgentID uuid.UUID
		expectedStatus  amagent.Status
		expectedRes     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d961122-8df4-11ee-8e1b-9bd95bec6c75"),
				},
			},

			reqQuery: "/agents/a8ba6662-540a-11ec-9a9f-b31de1a77615/status",
			reqBody:  []byte(`{"status":"available"}`),

			responseAgent: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a8ba6662-540a-11ec-9a9f-b31de1a77615"),
				},
			},

			expectedAgentID: uuid.FromStringOrNil("a8ba6662-540a-11ec-9a9f-b31de1a77615"),
			expectedStatus:  amagent.StatusAvailable,
			expectedRes:     `{"id":"a8ba6662-540a-11ec-9a9f-b31de1a77615","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().AgentUpdateStatus(req.Context(), &tt.agent, tt.expectedAgentID, tt.expectedStatus).Return(tt.responseAgent, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_PutAgentsIdPermission(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAgent *amagent.WebhookMessage

		expectedAgentID    uuid.UUID
		expectedPermission amagent.Permission
		expectedRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d961122-8df4-11ee-8e1b-9bd95bec6c75"),
				},
			},

			reqQuery: "/agents/a8ba6662-540a-11ec-9a9f-b31de1a77615/permission",
			reqBody:  []byte(`{"permission":32}`),

			responseAgent: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a8ba6662-540a-11ec-9a9f-b31de1a77615"),
				},
			},

			expectedAgentID:    uuid.FromStringOrNil("a8ba6662-540a-11ec-9a9f-b31de1a77615"),
			expectedPermission: amagent.PermissionCustomerAdmin,
			expectedRes:        `{"id":"a8ba6662-540a-11ec-9a9f-b31de1a77615","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().AgentUpdatePermission(req.Context(), &tt.agent, tt.expectedAgentID, tt.expectedPermission).Return(tt.responseAgent, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_PutAgentsIdPassword(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAgent *amagent.WebhookMessage

		expectedAgentID  uuid.UUID
		expectedPassword string
		expectedRes      string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d3481932-d3cf-11ee-ab64-5b6368efe4ce"),
				},
			},

			reqQuery: "/agents/d3481932-d3cf-11ee-ab64-5b6368efe4ce/password",
			reqBody:  []byte(`{"password":"updatepassword"}`),

			responseAgent: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d3481932-d3cf-11ee-ab64-5b6368efe4ce"),
				},
			},

			expectedAgentID:  uuid.FromStringOrNil("d3481932-d3cf-11ee-ab64-5b6368efe4ce"),
			expectedPassword: "updatepassword",
			expectedRes:      `{"id":"d3481932-d3cf-11ee-ab64-5b6368efe4ce","customer_id":"00000000-0000-0000-0000-000000000000","username":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().AgentUpdatePassword(req.Context(), &tt.agent, tt.expectedAgentID, tt.expectedPassword).Return(tt.responseAgent, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}
